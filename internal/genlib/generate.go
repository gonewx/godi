package genlib

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"

	"github.com/gonewx/godi/plan"
)

// Generate is the unique code-emission path (AD-2). It validates the registry
// first; a non-empty Validate result aborts emission and returns nil output +
// error (AD-3: bad graph → zero-byte code, NFR2 hard gate).
//
// The emitted container is a full godi.Provider implementation (AD-16 six-method
// face) with singletons stored as typed Root fields and constructed eagerly in
// NewRoot (AD-1: lifetime = where the instance lives). reflect.Type is used as
// the dispatch key only (Tier B, AD-13); construction is the direct Ctor call,
// never reflect.New / reflect.Value.Call.
//
// Story 2.2 evolves the emit for three lifetimes (AD-1):
//   - Singleton = Root field (NewRoot eager, unchanged from 2.1)
//   - Scoped = Root field (root-level) + scope field (scope-level, D-2)
//   - Transient = New{T}() method on Root and scope (each call new, D-4)
func Generate(reg *plan.Registry, pkgName string) ([]byte, error) {
	if errs := reg.Validate(); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return nil, fmt.Errorf("validation failed:\n%s", strings.Join(msgs, "\n"))
	}

	// Pre-emit check: detect sanitized field name collisions across all
	// services. Two services whose Names sanitize to the same Go identifier
	// would produce duplicate struct fields → compile error (review F1).
	{
		type fieldOrigin struct {
			name      string
			sanitized string
		}
		fieldSeen := make(map[string]fieldOrigin)
		allSvcs := append(reg.Services(), reg.KeyedServices()...)
		for _, s := range allSvcs {
			sf := sanitizeFieldName(s.Name)
			if prev, dup := fieldSeen[sf]; dup && prev.name != s.Name {
				return nil, fmt.Errorf("field name collision: services %q and %q both "+
					"sanitize to field f_%s; use distinct key expressions that produce "+
					"different identifier suffixes", prev.name, s.Name, sf)
			}
			fieldSeen[sf] = fieldOrigin{name: s.Name, sanitized: sf}
		}
	}

	var buf bytes.Buffer
	services := reg.Services()
	singletons := filterByLifetime(services, plan.Singleton)
	scoped := filterByLifetime(services, plan.Scoped)
	transients := reg.Transients()
	singletonWiring := reg.SingletonsTopo()
	scopedWiring := reg.ScopedTopo()
	hasLifetimes := len(scoped) > 0 || len(transients) > 0
	disposables := hasDisposables(services)

	keyedServices := reg.KeyedServices()
	keyedSingletons := filterByLifetime(keyedServices, plan.Singleton)
	keyedScoped := filterByLifetime(keyedServices, plan.Scoped)
	keyedTransients := reg.KeyedTransients()
	keyedSingletonWiring := reg.KeyedSingletonsTopo()
	hasKeyed := len(keyedServices) > 0
	if hasKeyed {
		hasLifetimes = true
	}
	if hasDisposables(keyedServices) {
		disposables = true
	}
	// hasAsync gates the emitted `context` import and the CloseAsync methods
	// (D-8) — independent of `disposables` so a purely-sync disposable container
	// emits neither, keeping Story 2.3 goldens byte-identical.
	hasAsync := hasAsyncDisposables(services) || hasAsyncDisposables(keyedServices)

	emitHeader(&buf, pkgName)
	emitImports(&buf, reg.Imports, disposables, hasAsync)
	emitRootStruct(&buf, singletons, scoped, disposables, keyedSingletons, keyedScoped)
	emitProviderAssertion(&buf)
	emitNewRoot(&buf, singletonWiring, scopedWiring, reg, disposables, keyedSingletonWiring, reg.KeyedScopedTopo())

	if hasLifetimes {
		emitTransientMethods(&buf, transients, "r", "*Root", reg, disposables)
		emitKeyedTransientMethods(&buf, keyedTransients, "r", "*Root", reg, disposables)
		emitTypedAccessors(&buf, singletons, scoped, "r", "*Root")
		emitGenericGetMethods(&buf, "r", "*Root")
		emitGetServiceLifetimes(&buf, singletons, scoped, transients, disposables)
		emitIsServiceLifetimes(&buf, services)
	} else {
		emitTypedAccessors(&buf, singletons, nil, "r", "*Root")
		emitGenericGetMethods(&buf, "r", "*Root")
		emitGetServiceSingletonsOnly(&buf, singletons, disposables)
		emitIsServiceSingletonsOnly(&buf, singletons)
	}

	// IsKeyedService is an interface method (AD-16 face revised 6→7, Story 3.4), so
	// EVERY container must implement it — including unkeyed ones, where the switch
	// body is empty and only the nil/AnyKey/false lines remain. Emit unconditionally.
	emitIsKeyedService(&buf, keyedSingletons, keyedScoped, keyedTransients)
	emitGetKeyedServiceFull(&buf, reg, keyedSingletons, keyedScoped, keyedTransients, disposables)
	emitGetKeyedServicesRootFull(&buf, reg, singletons, scoped, transients, keyedSingletons, keyedScoped, keyedTransients, disposables)
	emitCreateScope(&buf, scopedWiring, reg, hasLifetimes, disposables, reg.KeyedScopedTopo())
	emitClose(&buf, disposables, hasAsync)
	emitScope(&buf, singletons, scoped, transients, reg, hasLifetimes, disposables, hasAsync, keyedSingletons, keyedScoped, keyedTransients)

	src, fmtErr := format.Source(buf.Bytes())
	if fmtErr != nil {
		return nil, fmt.Errorf("format generated code: %w", fmtErr)
	}
	return src, nil
}

func filterByLifetime(services []plan.Service, lt plan.Lifetime) []plan.Service {
	out := make([]plan.Service, 0, len(services))
	for _, s := range services {
		if s.Lifetime == lt {
			out = append(out, s)
		}
	}
	return out
}

func lastByGoType(services []plan.Service, goType string) plan.Service {
	var last plan.Service
	for _, s := range services {
		if s.GoType == goType {
			last = s
		}
	}
	return last
}

func servicesByGoType(services []plan.Service, goType string) []plan.Service {
	var out []plan.Service
	for _, s := range services {
		if s.GoType == goType {
			out = append(out, s)
		}
	}
	return out
}

func emitHeader(buf *bytes.Buffer, pkgName string) {
	fmt.Fprintf(buf, "// Code generated by godi-gen; DO NOT EDIT.\n")
	fmt.Fprintf(buf, "package %s\n\n", pkgName)
}

// emitImports writes the import block. "reflect" and the host godi package are
// always required (six-method contract uses reflect.Type and godi.Provider /
// godi.Scope); userImports carries service-package paths — a plain path, or
// "alias path" (space-separated) when the generator renamed a package whose
// declared name clashes with another import (two service packages both named
// options). Paths are deduped and service packages sorted for stable output.
func emitImports(buf *bytes.Buffer, userImports []string, disposables, hasAsync bool) {
	uniq := make(map[string]struct{}, len(userImports)+3)
	uniq["reflect"] = struct{}{}
	uniq["github.com/gonewx/godi"] = struct{}{}
	if disposables {
		uniq["github.com/gonewx/godi/runtime"] = struct{}{}
		uniq["io"] = struct{}{}
	}
	type userImport struct {
		alias string // "" for a plain import
		path  string
	}
	sorted := make([]userImport, 0, len(userImports))
	for _, p := range userImports {
		if p == "" {
			continue
		}
		imp := userImport{path: p}
		if i := strings.IndexByte(p, ' '); i >= 0 {
			imp.alias, imp.path = p[:i], p[i+1:]
		}
		if _, seen := uniq[imp.path]; seen {
			continue
		}
		uniq[imp.path] = struct{}{}
		sorted = append(sorted, imp)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].path < sorted[j].path })

	fmt.Fprintf(buf, "import (\n")
	if hasAsync {
		fmt.Fprintf(buf, "\t\"context\"\n")
	}
	if disposables {
		fmt.Fprintf(buf, "\t\"io\"\n")
	}
	fmt.Fprintf(buf, "\t\"reflect\"\n\n")
	fmt.Fprintf(buf, "\t\"github.com/gonewx/godi\"\n")
	if disposables {
		fmt.Fprintf(buf, "\t\"github.com/gonewx/godi/runtime\"\n")
	}
	for _, imp := range sorted {
		if imp.alias != "" {
			fmt.Fprintf(buf, "\t%s %q\n", imp.alias, imp.path)
			continue
		}
		fmt.Fprintf(buf, "\t%q\n", imp.path)
	}
	fmt.Fprintf(buf, ")\n\n")
}

func emitRootStruct(buf *bytes.Buffer, singletons, scoped []plan.Service, disposables bool, keyedSingletons, keyedScoped []plan.Service) {
	fmt.Fprintf(buf, "// Root is the generated singleton container (AD-1: singleton = Root field).\n")
	fmt.Fprintf(buf, "type Root struct {\n")
	for _, s := range singletons {
		fmt.Fprintf(buf, "\tf_%s %s\n", s.Name, s.GoType)
	}
	for _, s := range scoped {
		fmt.Fprintf(buf, "\tf_%s %s\n", s.Name, s.GoType)
	}
	for _, s := range keyedSingletons {
		fmt.Fprintf(buf, "\tf_%s %s\n", sanitizeFieldName(s.Name), s.GoType)
	}
	for _, s := range keyedScoped {
		fmt.Fprintf(buf, "\tf_%s %s\n", sanitizeFieldName(s.Name), s.GoType)
	}
	// `closed` is declared UNCONDITIONALLY so resolve-after-dispose is identical
	// across both providers (AD-16): even a container with nothing to release must
	// reject post-Close resolution, matching the runtime oracle. `disp`/`closeErr`
	// exist only when there is something to release.
	if disposables {
		fmt.Fprintf(buf, "\tdisp     *runtime.Disposer\n")
		fmt.Fprintf(buf, "\tclosed   bool\n")
		fmt.Fprintf(buf, "\tcloseErr error\n")
	} else {
		fmt.Fprintf(buf, "\tclosed bool\n")
	}
	fmt.Fprintf(buf, "}\n\n")
}

// emitProviderAssertion writes the compile-time proof that *Root satisfies
// godi.Provider. Drift in the contract (a renamed or missing method) fails
// here at build time rather than at first resolve (AD-16).
func emitProviderAssertion(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "var _ godi.Provider = (*Root)(nil)\n\n")
}

// emitNewRoot writes the eager singleton + root-level scoped wiring, plus keyed
// singleton and root-level keyed scoped wiring (AD-1, D-2). Root acts as the root
// scope, so keyed scoped services are materialized on Root just like unkeyed scoped
// — keeping keyed/unkeyed and codegen/runtime consistent (AC1, AD-15).
func emitNewRoot(buf *bytes.Buffer, singletonWiring, scopedWiring []plan.Service, reg *plan.Registry, disposables bool, keyedSingletonWiring, keyedScopedWiring []plan.Service) {
	needsErr := false
	for _, s := range singletonWiring {
		if s.CtorErr && s.FactoryExpr == "" {
			needsErr = true
			break
		}
	}
	if !needsErr {
		for _, s := range scopedWiring {
			if s.CtorErr && s.FactoryExpr == "" {
				needsErr = true
				break
			}
		}
	}

	fmt.Fprintf(buf, "// NewRoot constructs and wires all singletons eagerly in dependency order (AD-1).\n")
	fmt.Fprintf(buf, "func NewRoot() (*Root, error) {\n")
	fmt.Fprintf(buf, "\tr := &Root{}\n")
	if disposables {
		fmt.Fprintf(buf, "\tr.disp = runtime.NewDisposer()\n")
	}
	if needsErr {
		fmt.Fprintf(buf, "\tvar err error\n")
	}

	emitWiringBlock(buf, singletonWiring, "r", reg)
	emitWiringBlock(buf, scopedWiring, "r", reg)
	emitKeyedWiringBlock(buf, keyedSingletonWiring, "r", reg)
	emitKeyedWiringBlock(buf, keyedScopedWiring, "r", reg)

	fmt.Fprintf(buf, "\treturn r, nil\n")
	fmt.Fprintf(buf, "}\n\n")
}

// trackArg renders the argument to disp.Track / disp.TrackAsync for a value of
// declared type goType. An interface-typed service (e.g.
// fakes.IFakeDisposableSingleton) is type-asserted to the disposer interface —
// the UNCHANGED Story 2.3 form that keeps sync goldens byte-identical. A concrete
// pointer-typed service (e.g. *fakes.FakeAsyncOnlyDisposable) already implements
// the disposer interface and is passed directly, because a type assertion on a
// non-interface operand does not compile.
func trackArg(expr, goType, iface string) string {
	if strings.HasPrefix(goType, "*") {
		return expr
	}
	return expr + ".(" + iface + ")"
}

// emitFieldTrack writes the disp.Track / disp.TrackAsync line for a disposable
// field expression fieldExpr (declared type goType), tracked into recv.disp
// (D-8 dispatch): a Disposable service — including a both-disposable — keeps the
// UNCHANGED sync Track path; an async-only service uses TrackAsync. Nothing is
// written for a non-disposable service.
func emitFieldTrack(buf *bytes.Buffer, recv, fieldExpr, goType string, s plan.Service) {
	switch {
	case s.Disposable:
		fmt.Fprintf(buf, "\t%s.disp.Track(%s)\n", recv, trackArg(fieldExpr, goType, "io.Closer"))
	case s.AsyncDisposable:
		fmt.Fprintf(buf, "\t%s.disp.TrackAsync(%s)\n", recv, trackArg(fieldExpr, goType, "godi.AsyncDisposable"))
	}
}

// emitValueTrack writes the disp.Track / disp.TrackAsync line for a disposable
// transient's local value v. concrete=true (ctor path — v is the constructor's
// concrete return type) passes v directly; concrete=false (factory path — v has
// the declared goType, possibly an interface) asserts per trackArg. Same D-8
// dispatch as emitFieldTrack.
func emitValueTrack(buf *bytes.Buffer, recv, goType string, concrete bool, s plan.Service) {
	var iface, method string
	switch {
	case s.Disposable:
		iface, method = "io.Closer", "Track"
	case s.AsyncDisposable:
		iface, method = "godi.AsyncDisposable", "TrackAsync"
	default:
		return
	}
	arg := "v"
	if !concrete {
		arg = trackArg("v", goType, iface)
	}
	fmt.Fprintf(buf, "\t%s.disp.%s(%s)\n", recv, method, arg)
}

func emitWiringBlock(buf *bytes.Buffer, services []plan.Service, recv string, reg *plan.Registry) {
	for _, s := range services {
		if s.FactoryExpr != "" {
			fmt.Fprintf(buf, "\t%s.f_%s = (%s)(%s)\n", recv, s.Name, s.FactoryExpr, recv)
			emitFieldTrack(buf, recv, fmt.Sprintf("%s.f_%s", recv, s.Name), s.GoType, s)
			continue
		}
		emitErrTransientDeps(buf, s.Deps, recv, reg, "return nil, err")
		args := depArgsForCtx(s.Deps, recv, reg)
		if s.CtorErr {
			fmt.Fprintf(buf, "\t%s.f_%s, err = %s(%s)\n", recv, s.Name, s.Ctor, args)
			fmt.Fprintf(buf, "\tif err != nil {\n")
			fmt.Fprintf(buf, "\t\treturn nil, err\n")
			fmt.Fprintf(buf, "\t}\n")
			emitFieldTrack(buf, recv, fmt.Sprintf("%s.f_%s", recv, s.Name), s.GoType, s)
			continue
		}
		fmt.Fprintf(buf, "\t%s.f_%s = %s(%s)\n", recv, s.Name, s.Ctor, args)
		emitFieldTrack(buf, recv, fmt.Sprintf("%s.f_%s", recv, s.Name), s.GoType, s)
	}
}

func emitKeyedWiringBlock(buf *bytes.Buffer, services []plan.Service, recv string, reg *plan.Registry) {
	for _, s := range services {
		fname := keyedFieldName(s)
		if s.FactoryExpr != "" {
			fmt.Fprintf(buf, "\t%s.f_%s = (%s)(%s)\n", recv, fname, s.FactoryExpr, recv)
			emitFieldTrack(buf, recv, fmt.Sprintf("%s.f_%s", recv, fname), s.GoType, s)
			continue
		}
		args := depArgsForCtx(s.Deps, recv, reg)
		fmt.Fprintf(buf, "\t%s.f_%s = %s(%s)\n", recv, fname, s.Ctor, args)
		emitFieldTrack(buf, recv, fmt.Sprintf("%s.f_%s", recv, fname), s.GoType, s)
	}
}

// depArgsForCtx renders a service's dependencies as constructor arguments, with
// the access expression determined by the receiver context and each dependency's
// lifetime.
func depArgsForCtx(deps []string, recv string, reg *plan.Registry) string {
	if len(deps) == 0 {
		return ""
	}
	parts := make([]string, len(deps))
	for i, d := range deps {
		parts[i] = depExpr(d, recv, reg)
	}
	return strings.Join(parts, ", ")
}

func depExpr(dep, recv string, reg *plan.Registry) string {
	if strings.HasPrefix(dep, "[]") {
		return sliceDepExpr(dep, recv, reg)
	}
	return singleDepExpr(dep, recv, reg)
}

func sliceDepExpr(dep, recv string, reg *plan.Registry) string {
	goType := dep[2:]
	services := reg.ServicesByGoType(goType)
	if len(services) == 0 {
		return dep + "{}"
	}
	parts := make([]string, len(services))
	for i, s := range services {
		parts[i] = singleDepExpr(s.Name, recv, reg)
	}
	return dep + "{" + strings.Join(parts, ", ") + "}"
}

func singleDepExpr(dep, recv string, reg *plan.Registry) string {
	s, ok := reg.Lookup(dep)
	if !ok {
		return recv + ".f_" + dep
	}
	if s.Lifetime == plan.Transient && s.CtorErr {
		return "dep_" + dep
	}
	switch recv {
	case "r":
		if s.Lifetime == plan.Transient {
			return recv + ".New" + dep + "()"
		}
		return "r.f_" + dep
	case "s":
		switch s.Lifetime {
		case plan.Singleton:
			return "s.root.f_" + dep
		case plan.Scoped:
			return "s.f_" + dep
		case plan.Transient:
			return "s.New" + dep + "()"
		}
	}
	return recv + ".f_" + dep
}

// emitErrTransientDeps pre-emits temp variables for transient dependencies whose
// constructors return error. The temp var is named dep_{Name} and error is handled
// via errAction (e.g. "return nil, err" or "panic(err)").
func emitErrTransientDeps(buf *bytes.Buffer, deps []string, recv string, reg *plan.Registry, errAction string) {
	for _, d := range deps {
		if strings.HasPrefix(d, "[]") {
			goType := d[2:]
			for _, s := range reg.ServicesByGoType(goType) {
				if s.Lifetime != plan.Transient || !s.CtorErr {
					continue
				}
				fmt.Fprintf(buf, "\tdep_%s, err := %s.New%s()\n", s.Name, recv, s.Name)
				fmt.Fprintf(buf, "\tif err != nil {\n")
				fmt.Fprintf(buf, "\t\t%s\n", errAction)
				fmt.Fprintf(buf, "\t}\n")
			}
			continue
		}
		s, ok := reg.Lookup(d)
		if !ok || s.Lifetime != plan.Transient || !s.CtorErr {
			continue
		}
		fmt.Fprintf(buf, "\tdep_%s, err := %s.New%s()\n", d, recv, d)
		fmt.Fprintf(buf, "\tif err != nil {\n")
		fmt.Fprintf(buf, "\t\t%s\n", errAction)
		fmt.Fprintf(buf, "\t}\n")
	}
}

func emitTransientMethods(buf *bytes.Buffer, transients []plan.Service, recv, typeName string, reg *plan.Registry, disposables bool) {
	for _, s := range transients {
		disp := s.Disposable || s.AsyncDisposable
		if s.FactoryExpr != "" {
			fmt.Fprintf(buf, "func (%s %s) New%s() %s {\n", recv, typeName, s.Name, s.GoType)
			if disp {
				fmt.Fprintf(buf, "\tv := (%s)(%s)\n", s.FactoryExpr, recv)
				emitValueTrack(buf, recv, s.GoType, false, s)
				fmt.Fprintf(buf, "\treturn v\n")
			} else {
				fmt.Fprintf(buf, "\treturn (%s)(%s)\n", s.FactoryExpr, recv)
			}
			fmt.Fprintf(buf, "}\n\n")
		} else if s.CtorErr {
			fmt.Fprintf(buf, "func (%s %s) New%s() (%s, error) {\n", recv, typeName, s.Name, s.GoType)
			emitErrTransientDeps(buf, s.Deps, recv, reg, "return nil, err")
			args := depArgsForCtx(s.Deps, recv, reg)
			if disp {
				fmt.Fprintf(buf, "\tv, err := %s(%s)\n", s.Ctor, args)
				fmt.Fprintf(buf, "\tif err != nil {\n")
				fmt.Fprintf(buf, "\t\treturn nil, err\n")
				fmt.Fprintf(buf, "\t}\n")
				emitValueTrack(buf, recv, s.GoType, true, s)
				fmt.Fprintf(buf, "\treturn v, nil\n")
			} else {
				fmt.Fprintf(buf, "\treturn %s(%s)\n", s.Ctor, args)
			}
			fmt.Fprintf(buf, "}\n\n")
		} else {
			fmt.Fprintf(buf, "func (%s %s) New%s() %s {\n", recv, typeName, s.Name, s.GoType)
			emitErrTransientDeps(buf, s.Deps, recv, reg, "panic(err)")
			args := depArgsForCtx(s.Deps, recv, reg)
			if disp {
				fmt.Fprintf(buf, "\tv := %s(%s)\n", s.Ctor, args)
				emitValueTrack(buf, recv, s.GoType, true, s)
				fmt.Fprintf(buf, "\treturn v\n")
			} else {
				fmt.Fprintf(buf, "\treturn %s(%s)\n", s.Ctor, args)
			}
			fmt.Fprintf(buf, "}\n\n")
		}
	}
}

// emitGetServiceLifetimes writes GetService that dispatches by lifetime (D-5).
// Same-GoType multi-registrations are deduped to one case returning the
// last-registered service (last-wins, D-3).
func emitGetServiceLifetimes(buf *bytes.Buffer, singletons, scoped, transients []plan.Service, disposables bool) {
	fmt.Fprintf(buf, "// GetService dispatches by reflect.Type (Tier B, AD-13) — no reflective construction.\n")
	fmt.Fprintf(buf, "func (r *Root) GetService(t reflect.Type) any {\n")
	emitClosedGuard(buf, "r")
	all := concatServices(singletons, scoped, transients)
	if len(all) > 0 {
		seen := make(map[string]struct{})
		fmt.Fprintf(buf, "\tswitch t {\n")
		for _, s := range singletons {
			if _, dup := seen[s.GoType]; dup {
				continue
			}
			seen[s.GoType] = struct{}{}
			last := lastByGoType(singletons, s.GoType)
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", s.GoType)
			fmt.Fprintf(buf, "\t\treturn r.f_%s\n", last.Name)
		}
		for _, s := range scoped {
			if _, dup := seen[s.GoType]; dup {
				continue
			}
			seen[s.GoType] = struct{}{}
			last := lastByGoType(scoped, s.GoType)
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", s.GoType)
			fmt.Fprintf(buf, "\t\treturn r.f_%s\n", last.Name)
		}
		for _, s := range transients {
			if _, dup := seen[s.GoType]; dup {
				continue
			}
			seen[s.GoType] = struct{}{}
			last := lastByGoType(transients, s.GoType)
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", s.GoType)
			if last.CtorErr {
				fmt.Fprintf(buf, "\t\tv, err := r.New%s()\n", last.Name)
				fmt.Fprintf(buf, "\t\tif err != nil {\n")
				fmt.Fprintf(buf, "\t\t\tpanic(err)\n")
				fmt.Fprintf(buf, "\t\t}\n")
				fmt.Fprintf(buf, "\t\treturn v\n")
			} else {
				fmt.Fprintf(buf, "\t\treturn r.New%s()\n", last.Name)
			}
		}
		fmt.Fprintf(buf, "\t}\n")
	}
	fmt.Fprintf(buf, "\treturn nil\n")
	fmt.Fprintf(buf, "}\n\n")
}

// emitGetServiceSingletonsOnly preserves the legacy singleton-only emit shape.
func emitGetServiceSingletonsOnly(buf *bytes.Buffer, singletons []plan.Service, disposables bool) {
	fmt.Fprintf(buf, "// GetService dispatches by reflect.Type (Tier B, AD-13) — no reflective construction.\n")
	fmt.Fprintf(buf, "func (r *Root) GetService(t reflect.Type) any {\n")
	emitClosedGuard(buf, "r")
	if len(singletons) > 0 {
		seen := make(map[string]struct{})
		fmt.Fprintf(buf, "\tswitch t {\n")
		for _, s := range singletons {
			if _, dup := seen[s.GoType]; dup {
				continue
			}
			seen[s.GoType] = struct{}{}
			last := lastByGoType(singletons, s.GoType)
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", s.GoType)
			fmt.Fprintf(buf, "\t\treturn r.f_%s\n", last.Name)
		}
		fmt.Fprintf(buf, "\t}\n")
	}
	fmt.Fprintf(buf, "\treturn nil\n")
	fmt.Fprintf(buf, "}\n\n")
}

func emitIsServiceLifetimes(buf *bytes.Buffer, services []plan.Service) {
	fmt.Fprintf(buf, "func (r *Root) IsService(t reflect.Type) bool {\n")
	if len(services) > 0 {
		seen := make(map[string]struct{})
		fmt.Fprintf(buf, "\tswitch t {\n")
		for _, s := range services {
			if _, dup := seen[s.GoType]; dup {
				continue
			}
			seen[s.GoType] = struct{}{}
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", s.GoType)
			fmt.Fprintf(buf, "\t\treturn true\n")
		}
		fmt.Fprintf(buf, "\t}\n")
	}
	fmt.Fprintf(buf, "\treturn false\n")
	fmt.Fprintf(buf, "}\n\n")
}

func emitIsServiceSingletonsOnly(buf *bytes.Buffer, singletons []plan.Service) {
	fmt.Fprintf(buf, "func (r *Root) IsService(t reflect.Type) bool {\n")
	if len(singletons) > 0 {
		seen := make(map[string]struct{})
		fmt.Fprintf(buf, "\tswitch t {\n")
		for _, s := range singletons {
			if _, dup := seen[s.GoType]; dup {
				continue
			}
			seen[s.GoType] = struct{}{}
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", s.GoType)
			fmt.Fprintf(buf, "\t\treturn true\n")
		}
		fmt.Fprintf(buf, "\t}\n")
	}
	fmt.Fprintf(buf, "\treturn false\n")
	fmt.Fprintf(buf, "}\n\n")
}

// emitIsKeyedService writes the Root IsKeyedService (AD-16 face revised 6→7 in
// Story 3.4). It is a PURE QUERY counterpart of IsService: nil key routes to the
// unkeyed IsService, AnyKey returns false (NOT a panic — the deliberate contrast
// with GetKeyedService(AnyKey), Story 3.4 D-3), and a concrete (type, key) hit
// returns true. It is emitted UNCONDITIONALLY (D-6): IsKeyedService is part of the
// Provider contract, so EVERY generated container — including unkeyed ones — must
// implement it; with no keyed registration the type switch body is empty and only
// the nil/AnyKey/false routing remains. The type+key switch reuses the same
// keyedServicesByGoType / distinctKeys grouping as emitGetKeyedServiceFull so the
// two methods agree on which (type, key) pairs exist.
func emitIsKeyedService(buf *bytes.Buffer, keyedSingletons, keyedScoped, keyedTransients []plan.Service) {
	fmt.Fprintf(buf, "func (r *Root) IsKeyedService(t reflect.Type, key any) bool {\n")
	fmt.Fprintf(buf, "\tif key == nil {\n")
	fmt.Fprintf(buf, "\t\treturn r.IsService(t)\n")
	fmt.Fprintf(buf, "\t}\n")
	fmt.Fprintf(buf, "\tif key == godi.AnyKey {\n")
	fmt.Fprintf(buf, "\t\treturn false\n")
	fmt.Fprintf(buf, "\t}\n")

	allKeyed := concatServices(keyedSingletons, keyedScoped, keyedTransients)
	if len(allKeyed) > 0 {
		goTypes := distinctGoTypes(allKeyed)
		fmt.Fprintf(buf, "\tswitch t {\n")
		for _, goType := range goTypes {
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", goType)
			fmt.Fprintf(buf, "\t\tswitch key {\n")
			group := keyedServicesByGoType(allKeyed, goType)
			keys := distinctKeys(group)
			for _, key := range keys {
				fmt.Fprintf(buf, "\t\tcase %s:\n", key)
				fmt.Fprintf(buf, "\t\t\treturn true\n")
			}
			fmt.Fprintf(buf, "\t\t}\n")
		}
		fmt.Fprintf(buf, "\t}\n")
	}

	fmt.Fprintf(buf, "\treturn false\n")
	fmt.Fprintf(buf, "}\n\n")
}

// emitScopeIsKeyedService writes the scope IsKeyedService delegate (root holds the
// keyed registry; the scope only adds scoped-instance storage, so the query
// delegates, mirroring emitScopeIsService).
func emitScopeIsKeyedService(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "func (s *scope) IsKeyedService(t reflect.Type, key any) bool {\n")
	fmt.Fprintf(buf, "\treturn s.root.IsKeyedService(t, key)\n")
	fmt.Fprintf(buf, "}\n\n")
}

// emitGetKeyedService is no longer used — replaced by emitGetKeyedServiceFull.
// Kept as a compile-time assertion that the old signature was emitGetKeyedService(*bytes.Buffer).

// emitGetKeyedServicesRoot is no longer used — replaced by emitGetKeyedServicesRootFull.

func emitCollectionReturn(buf *bytes.Buffer, group []plan.Service, recv string) {
	hasErrTransient := false
	for _, s := range group {
		if s.Lifetime == plan.Transient && s.CtorErr {
			hasErrTransient = true
			break
		}
	}
	if hasErrTransient {
		for i, s := range group {
			if s.Lifetime == plan.Transient && s.CtorErr {
				fmt.Fprintf(buf, "\t\tv%d, err := %s.New%s()\n", i, recv, s.Name)
				fmt.Fprintf(buf, "\t\tif err != nil {\n")
				fmt.Fprintf(buf, "\t\t\tpanic(err)\n")
				fmt.Fprintf(buf, "\t\t}\n")
			}
		}
		fmt.Fprintf(buf, "\t\treturn []any{")
		for i, s := range group {
			if i > 0 {
				fmt.Fprintf(buf, ", ")
			}
			if s.Lifetime == plan.Transient && s.CtorErr {
				fmt.Fprintf(buf, "v%d", i)
			} else {
				fmt.Fprintf(buf, "%s", collectionItemExpr(s, recv))
			}
		}
		fmt.Fprintf(buf, "}\n")
	} else {
		fmt.Fprintf(buf, "\t\treturn []any{")
		for i, s := range group {
			if i > 0 {
				fmt.Fprintf(buf, ", ")
			}
			fmt.Fprintf(buf, "%s", collectionItemExpr(s, recv))
		}
		fmt.Fprintf(buf, "}\n")
	}
}

func collectionItemExpr(s plan.Service, recv string) string {
	switch recv {
	case "r":
		if s.Lifetime == plan.Transient {
			return recv + ".New" + s.Name + "()"
		}
		return recv + ".f_" + s.Name
	case "s", "ns":
		switch s.Lifetime {
		case plan.Singleton:
			return recv + ".root.f_" + s.Name
		case plan.Scoped:
			return recv + ".f_" + s.Name
		case plan.Transient:
			return recv + ".New" + s.Name + "()"
		}
	}
	return recv + ".f_" + s.Name
}

func emitCreateScope(buf *bytes.Buffer, scopedWiring []plan.Service, reg *plan.Registry, hasLifetimes bool, disposables bool, keyedScopedWiring []plan.Service) {
	fmt.Fprintf(buf, "func (r *Root) CreateScope() godi.Scope {\n")
	emitClosedGuard(buf, "r")
	if !hasLifetimes {
		fmt.Fprintf(buf, "\treturn &scope{root: r}\n")
	} else {
		fmt.Fprintf(buf, "\ts := &scope{root: r}\n")
		if disposables {
			fmt.Fprintf(buf, "\ts.disp = runtime.NewDisposer()\n")
		}
		needsErr := false
		for _, svc := range scopedWiring {
			if svc.CtorErr && svc.FactoryExpr == "" {
				needsErr = true
				break
			}
		}
		if needsErr {
			fmt.Fprintf(buf, "\tvar err error\n")
		}
		for _, svc := range scopedWiring {
			if svc.FactoryExpr != "" {
				fmt.Fprintf(buf, "\ts.f_%s = (%s)(s)\n", svc.Name, svc.FactoryExpr)
				emitFieldTrack(buf, "s", fmt.Sprintf("s.f_%s", svc.Name), svc.GoType, svc)
			} else {
				emitErrTransientDeps(buf, svc.Deps, "s", reg, "panic(err)")
				args := depArgsForCreateScope(svc.Deps, reg)
				if svc.CtorErr {
					fmt.Fprintf(buf, "\ts.f_%s, err = %s(%s)\n", svc.Name, svc.Ctor, args)
					fmt.Fprintf(buf, "\tif err != nil {\n")
					fmt.Fprintf(buf, "\t\tpanic(err)\n")
					fmt.Fprintf(buf, "\t}\n")
				} else {
					fmt.Fprintf(buf, "\ts.f_%s = %s(%s)\n", svc.Name, svc.Ctor, args)
				}
				emitFieldTrack(buf, "s", fmt.Sprintf("s.f_%s", svc.Name), svc.GoType, svc)
			}
		}
		emitKeyedWiringBlock(buf, keyedScopedWiring, "s", reg)
		fmt.Fprintf(buf, "\treturn s\n")
	}
	fmt.Fprintf(buf, "}\n\n")
}

func depArgsForCreateScope(deps []string, reg *plan.Registry) string {
	if len(deps) == 0 {
		return ""
	}
	parts := make([]string, len(deps))
	for i, d := range deps {
		if strings.HasPrefix(d, "[]") {
			parts[i] = sliceDepExprScope(d, "s", reg)
			continue
		}
		s, ok := reg.Lookup(d)
		if !ok {
			parts[i] = "s.root.f_" + d
			continue
		}
		switch s.Lifetime {
		case plan.Singleton:
			parts[i] = "s.root.f_" + d
		case plan.Scoped:
			parts[i] = "s.f_" + d
		case plan.Transient:
			if s.CtorErr {
				parts[i] = "dep_" + d
			} else {
				parts[i] = "s.New" + d + "()"
			}
		default:
			parts[i] = "s.root.f_" + d
		}
	}
	return strings.Join(parts, ", ")
}

func sliceDepExprScope(dep, recv string, reg *plan.Registry) string {
	goType := dep[2:]
	services := reg.ServicesByGoType(goType)
	if len(services) == 0 {
		return dep + "{}"
	}
	parts := make([]string, len(services))
	for i, s := range services {
		parts[i] = collectionItemExpr(s, recv)
	}
	return dep + "{" + strings.Join(parts, ", ") + "}"
}

// emitClosedGuard writes the resolve-after-dispose guard at the top of a
// resolution or scope-creation method. It is emitted UNCONDITIONALLY: every
// generated container carries a `closed` flag and Close marks it, so a closed
// container rejects resolution EXACTLY like the runtime oracle does — whose guard
// is also unconditional (AD-16: resolve-after-dispose identical across providers,
// regardless of whether the container tracks any disposable). A container with
// nothing to release still has a logical closed state to guard. For root recv "r"
// the guard is `if r.closed`; for a scope a closed ROOT also invalidates the live
// scope, so the guard is `if s.closed || s.root.closed` (SPCT-12..15). Both
// providers panic with godi.ProviderClosedError().
func emitClosedGuard(buf *bytes.Buffer, recv string) {
	if recv == "s" {
		fmt.Fprintf(buf, "\tif s.closed || s.root.closed {\n")
	} else {
		fmt.Fprintf(buf, "\tif %s.closed {\n", recv)
	}
	fmt.Fprintf(buf, "\t\tpanic(godi.ProviderClosedError())\n")
	fmt.Fprintf(buf, "\t}\n")
}

// emitClose writes Root.Close. It is idempotent (AD-16): the second call returns
// the SAME result as the first, matching the runtime oracle's sync.Once+closeErr
// caching. When the container tracks disposables, the first call marks closed and
// caches the CloseAll error in closeErr; subsequent calls return that cached err
// (NOT nil — a swallowed error would diverge from runtime, F4). When there is
// nothing to release, Close still marks closed so resolve-after-dispose trips the
// guard, and returns nil (no disposer to fail).
func emitClose(buf *bytes.Buffer, disposables, hasAsync bool) {
	fmt.Fprintf(buf, "func (r *Root) Close() error {\n")
	if disposables {
		fmt.Fprintf(buf, "\tif r.closed {\n")
		fmt.Fprintf(buf, "\t\treturn r.closeErr\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\tr.closed = true\n")
		fmt.Fprintf(buf, "\tr.closeErr = r.disp.CloseAll()\n")
		fmt.Fprintf(buf, "\treturn r.closeErr\n")
	} else {
		fmt.Fprintf(buf, "\tr.closed = true\n")
		fmt.Fprintf(buf, "\treturn nil\n")
	}
	fmt.Fprintf(buf, "}\n\n")

	// CloseAsync (structural opt-in godi.AsyncDisposable, D-2) is emitted only
	// when async-disposable services exist (D-8). It SHARES Root's `closed` guard
	// with Close (first-caller-wins, D-6): whichever runs first releases and
	// caches its error; the other returns the cached err.
	if hasAsync {
		fmt.Fprintf(buf, "func (r *Root) CloseAsync(ctx context.Context) error {\n")
		fmt.Fprintf(buf, "\tif r.closed {\n")
		fmt.Fprintf(buf, "\t\treturn r.closeErr\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\tr.closed = true\n")
		fmt.Fprintf(buf, "\tr.closeErr = r.disp.CloseAllAsync(ctx)\n")
		fmt.Fprintf(buf, "\treturn r.closeErr\n")
		fmt.Fprintf(buf, "}\n\n")
	}
}

func emitScope(buf *bytes.Buffer, singletons, scoped, transients []plan.Service, reg *plan.Registry, hasLifetimes bool, disposables, hasAsync bool, keyedSingletons, keyedScoped, keyedTransients []plan.Service) {
	if !hasLifetimes {
		emitScopeLegacy(buf)
		emitTypedAccessors(buf, singletons, nil, "s", "*scope")
		emitGenericGetMethods(buf, "s", "*scope")
		return
	}
	emitScopeStruct(buf, scoped, disposables, keyedScoped)
	emitScopeAssertion(buf)
	emitScopeGetService(buf, singletons, scoped, transients, disposables)
	emitScopeIsService(buf, singletons, scoped, transients)
	emitScopeIsKeyedService(buf)
	emitScopeGetKeyedServiceFull(buf, reg, keyedSingletons, keyedScoped, keyedTransients, disposables)
	emitScopeGetKeyedServicesFull(buf, reg, singletons, scoped, transients, keyedSingletons, keyedScoped, keyedTransients, disposables)
	emitScopeCreateScope(buf, reg.ScopedTopo(), reg, disposables, reg.KeyedScopedTopo())
	emitScopeClose(buf, disposables, hasAsync)
	emitTransientMethods(buf, transients, "s", "*scope", reg, disposables)
	emitKeyedTransientMethods(buf, keyedTransients, "s", "*scope", reg, disposables)
	emitTypedAccessors(buf, singletons, scoped, "s", "*scope")
	emitGenericGetMethods(buf, "s", "*scope")
}

func emitScopeLegacy(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "type scope struct{ root *Root }\n\n")
	fmt.Fprintf(buf, "var _ godi.Scope = (*scope)(nil)\n\n")
	fmt.Fprintf(buf, "func (s *scope) GetService(t reflect.Type) any { return s.root.GetService(t) }\n\n")
	fmt.Fprintf(buf, "func (s *scope) IsService(t reflect.Type) bool { return s.root.IsService(t) }\n\n")
	fmt.Fprintf(buf, "func (s *scope) IsKeyedService(t reflect.Type, key any) bool {\n")
	fmt.Fprintf(buf, "\treturn s.root.IsKeyedService(t, key)\n")
	fmt.Fprintf(buf, "}\n\n")
	fmt.Fprintf(buf, "func (s *scope) GetKeyedService(t reflect.Type, key any) any {\n")
	fmt.Fprintf(buf, "\treturn s.root.GetKeyedService(t, key)\n")
	fmt.Fprintf(buf, "}\n\n")
	fmt.Fprintf(buf, "func (s *scope) GetKeyedServices(t reflect.Type, key any) []any {\n")
	fmt.Fprintf(buf, "\treturn s.root.GetKeyedServices(t, key)\n")
	fmt.Fprintf(buf, "}\n\n")
	fmt.Fprintf(buf, "func (s *scope) CreateScope() godi.Scope { return &scope{root: s.root} }\n\n")
	fmt.Fprintf(buf, "func (s *scope) Close() error { return nil }\n")
}

func emitScopeStruct(buf *bytes.Buffer, scoped []plan.Service, disposables bool, keyedScoped []plan.Service) {
	fmt.Fprintf(buf, "type scope struct {\n")
	fmt.Fprintf(buf, "\troot *Root\n")
	for _, s := range scoped {
		fmt.Fprintf(buf, "\tf_%s %s\n", s.Name, s.GoType)
	}
	for _, s := range keyedScoped {
		fmt.Fprintf(buf, "\tf_%s %s\n", keyedFieldName(s), s.GoType)
	}
	// `closed` is declared UNCONDITIONALLY (see emitRootStruct): a closed scope
	// must reject resolution even with nothing to release, so resolve-after-dispose
	// matches the runtime oracle (AD-16). `disp`/`closeErr` exist only when there is
	// something to release.
	if disposables {
		fmt.Fprintf(buf, "\tdisp     *runtime.Disposer\n")
		fmt.Fprintf(buf, "\tclosed   bool\n")
		fmt.Fprintf(buf, "\tcloseErr error\n")
	} else {
		fmt.Fprintf(buf, "\tclosed bool\n")
	}
	fmt.Fprintf(buf, "}\n\n")
}

func emitScopeAssertion(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "var _ godi.Scope = (*scope)(nil)\n\n")
}

func emitScopeGetService(buf *bytes.Buffer, singletons, scoped, transients []plan.Service, disposables bool) {
	fmt.Fprintf(buf, "func (s *scope) GetService(t reflect.Type) any {\n")
	emitClosedGuard(buf, "s")
	all := concatServices(singletons, scoped, transients)
	if len(all) > 0 {
		seen := make(map[string]struct{})
		fmt.Fprintf(buf, "\tswitch t {\n")
		for _, svc := range singletons {
			if _, dup := seen[svc.GoType]; dup {
				continue
			}
			seen[svc.GoType] = struct{}{}
			last := lastByGoType(singletons, svc.GoType)
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", svc.GoType)
			fmt.Fprintf(buf, "\t\treturn s.root.f_%s\n", last.Name)
		}
		for _, svc := range scoped {
			if _, dup := seen[svc.GoType]; dup {
				continue
			}
			seen[svc.GoType] = struct{}{}
			last := lastByGoType(scoped, svc.GoType)
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", svc.GoType)
			fmt.Fprintf(buf, "\t\treturn s.f_%s\n", last.Name)
		}
		for _, svc := range transients {
			if _, dup := seen[svc.GoType]; dup {
				continue
			}
			seen[svc.GoType] = struct{}{}
			last := lastByGoType(transients, svc.GoType)
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", svc.GoType)
			if last.CtorErr {
				fmt.Fprintf(buf, "\t\tv, err := s.New%s()\n", last.Name)
				fmt.Fprintf(buf, "\t\tif err != nil {\n")
				fmt.Fprintf(buf, "\t\t\tpanic(err)\n")
				fmt.Fprintf(buf, "\t\t}\n")
				fmt.Fprintf(buf, "\t\treturn v\n")
			} else {
				fmt.Fprintf(buf, "\t\treturn s.New%s()\n", last.Name)
			}
		}
		fmt.Fprintf(buf, "\t}\n")
	}
	fmt.Fprintf(buf, "\treturn nil\n")
	fmt.Fprintf(buf, "}\n\n")
}

func emitScopeIsService(buf *bytes.Buffer, singletons, scoped, transients []plan.Service) {
	fmt.Fprintf(buf, "func (s *scope) IsService(t reflect.Type) bool {\n")
	all := concatServices(singletons, scoped, transients)
	if len(all) > 0 {
		seen := make(map[string]struct{})
		fmt.Fprintf(buf, "\tswitch t {\n")
		for _, svc := range all {
			if _, dup := seen[svc.GoType]; dup {
				continue
			}
			seen[svc.GoType] = struct{}{}
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", svc.GoType)
			fmt.Fprintf(buf, "\t\treturn true\n")
		}
		fmt.Fprintf(buf, "\t}\n")
	}
	fmt.Fprintf(buf, "\treturn false\n")
	fmt.Fprintf(buf, "}\n\n")
}

func emitScopeGetKeyedServiceFull(buf *bytes.Buffer, reg *plan.Registry, keyedSingletons, keyedScoped, keyedTransients []plan.Service, disposables bool) {
	fmt.Fprintf(buf, "func (s *scope) GetKeyedService(t reflect.Type, key any) any {\n")
	emitClosedGuard(buf, "s")
	fmt.Fprintf(buf, "\tif key == nil {\n")
	fmt.Fprintf(buf, "\t\treturn s.GetService(t)\n")
	fmt.Fprintf(buf, "\t}\n")
	emitAnyKeySingleGuard(buf)

	allKeyed := concatServices(keyedSingletons, keyedScoped, keyedTransients)
	if len(allKeyed) > 0 {
		goTypes := distinctGoTypes(allKeyed)
		fmt.Fprintf(buf, "\tswitch t {\n")
		for _, goType := range goTypes {
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", goType)
			fmt.Fprintf(buf, "\t\tswitch key {\n")
			group := keyedServicesByGoType(allKeyed, goType)
			keys := distinctKeys(group)
			for _, key := range keys {
				matches := keyedServicesByKey(group, key)
				last := matches[len(matches)-1]
				fmt.Fprintf(buf, "\t\tcase %s:\n", key)
				emitKeyedServiceReturn(buf, last, "s")
			}
			fmt.Fprintf(buf, "\t\t}\n")
		}
		fmt.Fprintf(buf, "\t}\n")
	}

	fmt.Fprintf(buf, "\treturn nil\n")
	fmt.Fprintf(buf, "}\n\n")
}

func emitScopeGetKeyedServicesFull(buf *bytes.Buffer, reg *plan.Registry, singletons, scoped, transients, keyedSingletons, keyedScoped, keyedTransients []plan.Service, disposables bool) {
	allKeyed := concatServices(keyedSingletons, keyedScoped, keyedTransients)
	unkeyedAll := concatServices(singletons, scoped, transients)

	fmt.Fprintf(buf, "func (s *scope) GetKeyedServices(t reflect.Type, key any) []any {\n")
	emitClosedGuard(buf, "s")

	if len(allKeyed) == 0 {
		fmt.Fprintf(buf, "\tif key != nil {\n")
		fmt.Fprintf(buf, "\t\treturn nil\n")
		fmt.Fprintf(buf, "\t}\n")
		if len(unkeyedAll) > 0 {
			seen := make(map[string]struct{})
			fmt.Fprintf(buf, "\tswitch t {\n")
			for _, sv := range unkeyedAll {
				if _, dup := seen[sv.GoType]; dup {
					continue
				}
				seen[sv.GoType] = struct{}{}
				group := servicesByGoType(unkeyedAll, sv.GoType)
				fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", sv.GoType)
				emitCollectionReturn(buf, group, "s")
			}
			fmt.Fprintf(buf, "\t}\n")
		}
		fmt.Fprintf(buf, "\treturn nil\n")
		fmt.Fprintf(buf, "}\n\n")
		return
	}

	fmt.Fprintf(buf, "\tif key == nil {\n")
	if len(unkeyedAll) > 0 {
		seen := make(map[string]struct{})
		fmt.Fprintf(buf, "\t\tswitch t {\n")
		for _, sv := range unkeyedAll {
			if _, dup := seen[sv.GoType]; dup {
				continue
			}
			seen[sv.GoType] = struct{}{}
			group := servicesByGoType(unkeyedAll, sv.GoType)
			fmt.Fprintf(buf, "\t\tcase reflect.TypeFor[%s]():\n", sv.GoType)
			emitCollectionReturnIndented(buf, group, "s", "\t\t")
		}
		fmt.Fprintf(buf, "\t\t}\n")
	}
	fmt.Fprintf(buf, "\t\treturn nil\n")
	fmt.Fprintf(buf, "\t}\n")

	emitAnyKeyQueryBranch(buf, reg, "s")

	goTypes := distinctGoTypes(allKeyed)
	fmt.Fprintf(buf, "\tswitch t {\n")
	for _, goType := range goTypes {
		fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", goType)
		fmt.Fprintf(buf, "\t\tswitch key {\n")
		group := keyedServicesByGoType(allKeyed, goType)
		keys := distinctKeys(group)
		for _, key := range keys {
			matches := keyedServicesByKey(group, key)
			fmt.Fprintf(buf, "\t\tcase %s:\n", key)
			emitKeyedCollectionReturn(buf, matches, "s")
		}
		fmt.Fprintf(buf, "\t\t}\n")
	}
	fmt.Fprintf(buf, "\t}\n")

	fmt.Fprintf(buf, "\treturn nil\n")
	fmt.Fprintf(buf, "}\n\n")
}

func emitScopeCreateScope(buf *bytes.Buffer, scopedWiring []plan.Service, reg *plan.Registry, disposables bool, keyedScopedWiring []plan.Service) {
	fmt.Fprintf(buf, "func (s *scope) CreateScope() godi.Scope {\n")
	emitClosedGuard(buf, "s")
	fmt.Fprintf(buf, "\tns := &scope{root: s.root}\n")
	if disposables {
		fmt.Fprintf(buf, "\tns.disp = runtime.NewDisposer()\n")
	}
	needsErr := false
	for _, svc := range scopedWiring {
		if svc.CtorErr && svc.FactoryExpr == "" {
			needsErr = true
			break
		}
	}
	if needsErr {
		fmt.Fprintf(buf, "\tvar err error\n")
	}
	for _, svc := range scopedWiring {
		if svc.FactoryExpr != "" {
			fmt.Fprintf(buf, "\tns.f_%s = (%s)(ns)\n", svc.Name, svc.FactoryExpr)
			emitFieldTrack(buf, "ns", fmt.Sprintf("ns.f_%s", svc.Name), svc.GoType, svc)
		} else {
			emitErrTransientDeps(buf, svc.Deps, "ns", reg, "panic(err)")
			args := depArgsForNestedScope(svc.Deps, reg)
			if svc.CtorErr {
				fmt.Fprintf(buf, "\tns.f_%s, err = %s(%s)\n", svc.Name, svc.Ctor, args)
				fmt.Fprintf(buf, "\tif err != nil {\n")
				fmt.Fprintf(buf, "\t\tpanic(err)\n")
				fmt.Fprintf(buf, "\t}\n")
			} else {
				fmt.Fprintf(buf, "\tns.f_%s = %s(%s)\n", svc.Name, svc.Ctor, args)
			}
			emitFieldTrack(buf, "ns", fmt.Sprintf("ns.f_%s", svc.Name), svc.GoType, svc)
		}
	}
	for _, svc := range keyedScopedWiring {
		fname := keyedFieldName(svc)
		if svc.FactoryExpr != "" {
			fmt.Fprintf(buf, "\tns.f_%s = (%s)(ns)\n", fname, svc.FactoryExpr)
			emitFieldTrack(buf, "ns", fmt.Sprintf("ns.f_%s", fname), svc.GoType, svc)
		} else {
			args := depArgsForNestedScope(svc.Deps, reg)
			fmt.Fprintf(buf, "\tns.f_%s = %s(%s)\n", fname, svc.Ctor, args)
			emitFieldTrack(buf, "ns", fmt.Sprintf("ns.f_%s", fname), svc.GoType, svc)
		}
	}
	fmt.Fprintf(buf, "\treturn ns\n")
	fmt.Fprintf(buf, "}\n\n")
}

func depArgsForNestedScope(deps []string, reg *plan.Registry) string {
	if len(deps) == 0 {
		return ""
	}
	parts := make([]string, len(deps))
	for i, d := range deps {
		if strings.HasPrefix(d, "[]") {
			parts[i] = sliceDepExprScope(d, "ns", reg)
			continue
		}
		s, ok := reg.Lookup(d)
		if !ok {
			parts[i] = "ns.root.f_" + d
			continue
		}
		switch s.Lifetime {
		case plan.Singleton:
			parts[i] = "ns.root.f_" + d
		case plan.Scoped:
			parts[i] = "ns.f_" + d
		case plan.Transient:
			if s.CtorErr {
				parts[i] = "dep_" + d
			} else {
				parts[i] = "ns.New" + d + "()"
			}
		default:
			parts[i] = "ns.root.f_" + d
		}
	}
	return strings.Join(parts, ", ")
}

// emitScopeClose writes scope.Close, idempotent and matching the runtime oracle
// (AD-16). Like Root.Close, the second call returns the cached closeErr (not nil)
// when there are disposables; when there is nothing to release Close still marks
// the scope closed so resolve-after-dispose trips the guard.
func emitScopeClose(buf *bytes.Buffer, disposables, hasAsync bool) {
	if disposables {
		fmt.Fprintf(buf, "func (s *scope) Close() error {\n")
		fmt.Fprintf(buf, "\tif s.closed {\n")
		fmt.Fprintf(buf, "\t\treturn s.closeErr\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\ts.closed = true\n")
		fmt.Fprintf(buf, "\ts.closeErr = s.disp.CloseAll()\n")
		fmt.Fprintf(buf, "\treturn s.closeErr\n")
		fmt.Fprintf(buf, "}\n\n")
	} else {
		fmt.Fprintf(buf, "func (s *scope) Close() error {\n")
		fmt.Fprintf(buf, "\ts.closed = true\n")
		fmt.Fprintf(buf, "\treturn nil\n")
		fmt.Fprintf(buf, "}\n\n")
	}
	// scope.CloseAsync mirrors Root.CloseAsync (D-6 shared guard, D-2 structural
	// opt-in), emitted only when async-disposable services exist (D-8).
	if hasAsync {
		fmt.Fprintf(buf, "func (s *scope) CloseAsync(ctx context.Context) error {\n")
		fmt.Fprintf(buf, "\tif s.closed {\n")
		fmt.Fprintf(buf, "\t\treturn s.closeErr\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\ts.closed = true\n")
		fmt.Fprintf(buf, "\ts.closeErr = s.disp.CloseAllAsync(ctx)\n")
		fmt.Fprintf(buf, "\treturn s.closeErr\n")
		fmt.Fprintf(buf, "}\n\n")
	}
}

func emitTypedAccessors(buf *bytes.Buffer, singletons, scoped []plan.Service, recv, typeName string) {
	for _, s := range singletons {
		expr := recv + ".f_" + s.Name
		if recv == "s" {
			expr = "s.root.f_" + s.Name
		}
		fmt.Fprintf(buf, "func (%s %s) %s() %s { return %s }\n\n", recv, typeName, s.Name, s.GoType, expr)
	}
	for _, s := range scoped {
		expr := recv + ".f_" + s.Name
		fmt.Fprintf(buf, "func (%s %s) %s() %s { return %s }\n\n", recv, typeName, s.Name, s.GoType, expr)
	}
}

func emitGenericGetMethods(buf *bytes.Buffer, recv, typeName string) {
	fmt.Fprintf(buf, "func (%s %s) Get[T any]() (T, bool) {\n", recv, typeName)
	fmt.Fprintf(buf, "\tv := %s.GetService(reflect.TypeFor[T]())\n", recv)
	fmt.Fprintf(buf, "\tif v == nil {\n\t\tvar zero T\n\t\treturn zero, false\n\t}\n")
	fmt.Fprintf(buf, "\tt, ok := v.(T)\n")
	fmt.Fprintf(buf, "\treturn t, ok\n}\n\n")

	fmt.Fprintf(buf, "func (%s %s) GetRequired[T any]() T {\n", recv, typeName)
	fmt.Fprintf(buf, "\tif v, ok := %s.Get[T](); ok {\n\t\treturn v\n\t}\n", recv)
	fmt.Fprintf(buf, "\tgodi.GetRequiredPanic[T]()\n")
	fmt.Fprintf(buf, "\tpanic(\"unreachable\")\n}\n\n")
}

func concatServices(slices ...[]plan.Service) []plan.Service {
	var total int
	for _, s := range slices {
		total += len(s)
	}
	out := make([]plan.Service, 0, total)
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

// hasDisposables reports whether any service opts into release (sync io.Closer
// OR async godi.AsyncDisposable, Story 6.3). It gates the disp field, the
// runtime import, and NewDisposer — so a purely-async-disposable container still
// gets a disposer. For a purely-sync container this equals Story 2.3's behavior
// (byte-identical goldens).
func hasDisposables(services []plan.Service) bool {
	for _, s := range services {
		if s.Disposable || s.AsyncDisposable {
			return true
		}
	}
	return false
}

// hasAsyncDisposables reports whether any service opts into async release
// (godi.AsyncDisposable). It gates the emitted `context` import and the
// CloseAsync methods on Root/scope — independent of hasDisposables so a
// purely-sync disposable container emits neither (D-8).
func hasAsyncDisposables(services []plan.Service) bool {
	for _, s := range services {
		if s.AsyncDisposable {
			return true
		}
	}
	return false
}

// sanitizeFieldName converts a keyed service Name (e.g. "KeyedServiceOne#\"service1\"")
// into a valid Go identifier for use as a struct field suffix. It replaces
// '#', '"', and other non-ident characters with '_'.
func sanitizeFieldName(name string) string {
	var b strings.Builder
	for _, c := range name {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' {
			b.WriteRune(c)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// keyedFieldName returns the sanitized field expression for a keyed service.
func keyedFieldName(s plan.Service) string {
	return sanitizeFieldName(s.Name)
}

func emitKeyedTransientMethods(buf *bytes.Buffer, transients []plan.Service, recv, typeName string, reg *plan.Registry, disposables bool) {
	for _, s := range transients {
		fname := keyedFieldName(s)
		disp := s.Disposable || s.AsyncDisposable
		if s.FactoryExpr != "" {
			fmt.Fprintf(buf, "func (%s %s) New%s() %s {\n", recv, typeName, fname, s.GoType)
			if disp {
				fmt.Fprintf(buf, "\tv := (%s)(%s)\n", s.FactoryExpr, recv)
				emitValueTrack(buf, recv, s.GoType, false, s)
				fmt.Fprintf(buf, "\treturn v\n")
			} else {
				fmt.Fprintf(buf, "\treturn (%s)(%s)\n", s.FactoryExpr, recv)
			}
			fmt.Fprintf(buf, "}\n\n")
		} else {
			fmt.Fprintf(buf, "func (%s %s) New%s() %s {\n", recv, typeName, fname, s.GoType)
			args := depArgsForCtx(s.Deps, recv, reg)
			if disp {
				fmt.Fprintf(buf, "\tv := %s(%s)\n", s.Ctor, args)
				emitValueTrack(buf, recv, s.GoType, true, s)
				fmt.Fprintf(buf, "\treturn v\n")
			} else {
				fmt.Fprintf(buf, "\treturn %s(%s)\n", s.Ctor, args)
			}
			fmt.Fprintf(buf, "}\n\n")
		}
	}
}

func emitGetKeyedServiceFull(buf *bytes.Buffer, reg *plan.Registry, keyedSingletons, keyedScoped, keyedTransients []plan.Service, disposables bool) {
	fmt.Fprintf(buf, "func (r *Root) GetKeyedService(t reflect.Type, key any) any {\n")
	emitClosedGuard(buf, "r")
	fmt.Fprintf(buf, "\tif key == nil {\n")
	fmt.Fprintf(buf, "\t\treturn r.GetService(t)\n")
	fmt.Fprintf(buf, "\t}\n")
	emitAnyKeySingleGuard(buf)

	allKeyed := concatServices(keyedSingletons, keyedScoped, keyedTransients)
	if len(allKeyed) > 0 {
		goTypes := distinctGoTypes(allKeyed)
		fmt.Fprintf(buf, "\tswitch t {\n")
		for _, goType := range goTypes {
			fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", goType)
			fmt.Fprintf(buf, "\t\tswitch key {\n")
			group := keyedServicesByGoType(allKeyed, goType)
			keys := distinctKeys(group)
			for _, key := range keys {
				matches := keyedServicesByKey(group, key)
				last := matches[len(matches)-1]
				fmt.Fprintf(buf, "\t\tcase %s:\n", key)
				emitKeyedServiceReturn(buf, last, "r")
			}
			fmt.Fprintf(buf, "\t\t}\n")
		}
		fmt.Fprintf(buf, "\t}\n")
	}

	fmt.Fprintf(buf, "\treturn nil\n")
	fmt.Fprintf(buf, "}\n\n")
}

func emitKeyedServiceReturn(buf *bytes.Buffer, s plan.Service, recv string) {
	fname := keyedFieldName(s)
	switch s.Lifetime {
	case plan.Transient:
		fmt.Fprintf(buf, "\t\t\treturn %s.New%s()\n", recv, fname)
	default:
		if recv == "s" {
			if s.Lifetime == plan.Singleton {
				fmt.Fprintf(buf, "\t\t\treturn %s.root.f_%s\n", recv, fname)
			} else {
				fmt.Fprintf(buf, "\t\t\treturn %s.f_%s\n", recv, fname)
			}
		} else {
			fmt.Fprintf(buf, "\t\t\treturn %s.f_%s\n", recv, fname)
		}
	}
}

func emitGetKeyedServicesRootFull(buf *bytes.Buffer, reg *plan.Registry, singletons, scoped, transients, keyedSingletons, keyedScoped, keyedTransients []plan.Service, disposables bool) {
	allKeyed := concatServices(keyedSingletons, keyedScoped, keyedTransients)
	unkeyedAll := concatServices(singletons, scoped, transients)

	fmt.Fprintf(buf, "func (r *Root) GetKeyedServices(t reflect.Type, key any) []any {\n")
	emitClosedGuard(buf, "r")

	if len(allKeyed) == 0 {
		fmt.Fprintf(buf, "\tif key != nil {\n")
		fmt.Fprintf(buf, "\t\treturn nil\n")
		fmt.Fprintf(buf, "\t}\n")
		if len(unkeyedAll) > 0 {
			seen := make(map[string]struct{})
			fmt.Fprintf(buf, "\tswitch t {\n")
			for _, s := range unkeyedAll {
				if _, dup := seen[s.GoType]; dup {
					continue
				}
				seen[s.GoType] = struct{}{}
				group := servicesByGoType(unkeyedAll, s.GoType)
				fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", s.GoType)
				emitCollectionReturn(buf, group, "r")
			}
			fmt.Fprintf(buf, "\t}\n")
		}
		fmt.Fprintf(buf, "\treturn nil\n")
		fmt.Fprintf(buf, "}\n\n")
		return
	}

	fmt.Fprintf(buf, "\tif key == nil {\n")
	if len(unkeyedAll) > 0 {
		seen := make(map[string]struct{})
		fmt.Fprintf(buf, "\t\tswitch t {\n")
		for _, s := range unkeyedAll {
			if _, dup := seen[s.GoType]; dup {
				continue
			}
			seen[s.GoType] = struct{}{}
			group := servicesByGoType(unkeyedAll, s.GoType)
			fmt.Fprintf(buf, "\t\tcase reflect.TypeFor[%s]():\n", s.GoType)
			emitCollectionReturnIndented(buf, group, "r", "\t\t")
		}
		fmt.Fprintf(buf, "\t\t}\n")
	}
	fmt.Fprintf(buf, "\t\treturn nil\n")
	fmt.Fprintf(buf, "\t}\n")

	emitAnyKeyQueryBranch(buf, reg, "r")

	goTypes := distinctGoTypes(allKeyed)
	fmt.Fprintf(buf, "\tswitch t {\n")
	for _, goType := range goTypes {
		fmt.Fprintf(buf, "\tcase reflect.TypeFor[%s]():\n", goType)
		fmt.Fprintf(buf, "\t\tswitch key {\n")
		group := keyedServicesByGoType(allKeyed, goType)
		keys := distinctKeys(group)
		for _, key := range keys {
			matches := keyedServicesByKey(group, key)
			fmt.Fprintf(buf, "\t\tcase %s:\n", key)
			emitKeyedCollectionReturn(buf, matches, "r")
		}
		fmt.Fprintf(buf, "\t\t}\n")
	}
	fmt.Fprintf(buf, "\t}\n")

	fmt.Fprintf(buf, "\treturn nil\n")
	fmt.Fprintf(buf, "}\n\n")
}

func emitCollectionReturnIndented(buf *bytes.Buffer, group []plan.Service, recv, indent string) {
	hasErrTransient := false
	for _, s := range group {
		if s.Lifetime == plan.Transient && s.CtorErr {
			hasErrTransient = true
			break
		}
	}
	if hasErrTransient {
		for i, s := range group {
			if s.Lifetime == plan.Transient && s.CtorErr {
				fmt.Fprintf(buf, "%s\tv%d, err := %s.New%s()\n", indent, i, recv, s.Name)
				fmt.Fprintf(buf, "%s\tif err != nil {\n", indent)
				fmt.Fprintf(buf, "%s\t\tpanic(err)\n", indent)
				fmt.Fprintf(buf, "%s\t}\n", indent)
			}
		}
		fmt.Fprintf(buf, "%s\treturn []any{", indent)
		for i, s := range group {
			if i > 0 {
				fmt.Fprintf(buf, ", ")
			}
			if s.Lifetime == plan.Transient && s.CtorErr {
				fmt.Fprintf(buf, "v%d", i)
			} else {
				fmt.Fprintf(buf, "%s", collectionItemExpr(s, recv))
			}
		}
		fmt.Fprintf(buf, "}\n")
	} else {
		fmt.Fprintf(buf, "%s\treturn []any{", indent)
		for i, s := range group {
			if i > 0 {
				fmt.Fprintf(buf, ", ")
			}
			fmt.Fprintf(buf, "%s", collectionItemExpr(s, recv))
		}
		fmt.Fprintf(buf, "}\n")
	}
}

func emitKeyedCollectionReturn(buf *bytes.Buffer, services []plan.Service, recv string) {
	fmt.Fprintf(buf, "\t\t\treturn []any{")
	for i, s := range services {
		if i > 0 {
			fmt.Fprintf(buf, ", ")
		}
		fname := keyedFieldName(s)
		switch s.Lifetime {
		case plan.Transient:
			fmt.Fprintf(buf, "%s.New%s()", recv, fname)
		default:
			if recv == "s" && s.Lifetime == plan.Singleton {
				fmt.Fprintf(buf, "%s.root.f_%s", recv, fname)
			} else {
				fmt.Fprintf(buf, "%s.f_%s", recv, fname)
			}
		}
	}
	fmt.Fprintf(buf, "}\n")
}

// emitAnyKeySingleGuard emits the AnyKey single-resolve guard shared by Root and
// scope GetKeyedService. AnyKey is a query-only sentinel: a single-value resolve
// ALWAYS panics, intercepted after the nil check and BEFORE the type switch, so it
// panics even with no keyed registration or a matching concrete key (AD-17). The
// panic value comes from godi.AnyKeySingleResolveError — the SAME error source the
// runtime uses — so generated code needs no fmt import (D-2).
func emitAnyKeySingleGuard(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "\tif key == godi.AnyKey {\n")
	fmt.Fprintf(buf, "\t\tpanic(godi.AnyKeySingleResolveError(t))\n")
	fmt.Fprintf(buf, "\t}\n")
}

// emitAnyKeyQueryBranch emits the GetKeyedServices(AnyKey) branch shared by Root
// and scope. It returns every concrete-keyed service of type t in REGISTRATION
// order — reg.KeyedServices() is registration-ordered, NOT lifetime/topo-ordered,
// so interleaved keys come back un-grouped (ResolveKeyedServicesAnyKeyOrdering,
// AC2). Unkeyed and nil-key services never enter the keyed registry, so they are
// excluded for free. recv is "r" (Root) or "s" (scope); the per-service field
// references reuse emitKeyedCollectionReturn, identical to the concrete-key path,
// so AnyKey-query and concrete-key resolution share the same instances.
func emitAnyKeyQueryBranch(buf *bytes.Buffer, reg *plan.Registry, recv string) {
	keyed := reg.KeyedServices()
	fmt.Fprintf(buf, "\tif key == godi.AnyKey {\n")
	if len(keyed) > 0 {
		fmt.Fprintf(buf, "\t\tswitch t {\n")
		for _, goType := range distinctGoTypes(keyed) {
			group := keyedServicesByGoType(keyed, goType)
			fmt.Fprintf(buf, "\t\tcase reflect.TypeFor[%s]():\n", goType)
			emitKeyedCollectionReturn(buf, group, recv)
		}
		fmt.Fprintf(buf, "\t\t}\n")
	}
	fmt.Fprintf(buf, "\t\treturn nil\n")
	fmt.Fprintf(buf, "\t}\n")
}

func distinctGoTypes(services []plan.Service) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range services {
		if _, ok := seen[s.GoType]; ok {
			continue
		}
		seen[s.GoType] = struct{}{}
		out = append(out, s.GoType)
	}
	return out
}

func distinctKeys(services []plan.Service) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range services {
		if _, ok := seen[s.Key]; ok {
			continue
		}
		seen[s.Key] = struct{}{}
		out = append(out, s.Key)
	}
	return out
}

func keyedServicesByGoType(services []plan.Service, goType string) []plan.Service {
	var out []plan.Service
	for _, s := range services {
		if s.GoType == goType {
			out = append(out, s)
		}
	}
	return out
}

func keyedServicesByKey(services []plan.Service, key string) []plan.Service {
	var out []plan.Service
	for _, s := range services {
		if s.Key == key {
			out = append(out, s)
		}
	}
	return out
}
