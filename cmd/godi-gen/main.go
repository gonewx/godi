// Command godi-gen is godi's static-analysis generator CLI (AD-2 / AD-7). It
// loads a user registration package with go/packages, reads type facts with
// go/types — never executing user code — discovers each form-A service's
// New{Impl} constructor and its dependencies, then emits the B2 container via
// the single code-emission path genlib.Generate.
//
// Story 2.1 turns this from a Story 1.1 placeholder into the real layered
// pipeline: go/packages load -> go/types read -> plan.Registry -> plan.Validate
// (inside Generate) -> genlib emit. A form-A service with no New{Impl}, or any
// other bad-graph condition, is rejected at generation time: zero bytes of code
// and a non-zero exit (NFR2 hard gate, AD-3).
//
// Usage:
//
//	godi-gen -pkg <name> [-out <file>] <srcDir>
//
// <srcDir> is the directory of the registration package to analyze; -pkg is the
// package name of the emitted container; -out is the output file (stdout when
// omitted). The //go:generate consumer wires this as a build step + CI gate.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/plan"
	"golang.org/x/tools/go/packages"
)

// godiPkgPath is the import path of the host package whose AddSingleton /
// AddSingletonFactory markers the generator dispatches on. Matching by package
// path (not just the function name) avoids misreading a same-named function from
// another package as a DSL marker.
const godiPkgPath = "github.com/gonewx/godi"

func main() {
	pkgName := flag.String("pkg", "", "package name of the emitted container (required)")
	out := flag.String("out", "", "output file path (default: stdout)")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 || *pkgName == "" {
		fmt.Fprintln(os.Stderr, "usage: godi-gen -pkg <name> [-out <file>] <srcDir>")
		os.Exit(2)
	}

	src, err := generate(args[0], *pkgName, *out)
	if err != nil {
		// AD-3 / NFR2: a bad graph (missing New{Impl}, validation failure, load
		// error, emitted container that does not type-check) produces ZERO bytes
		// and a non-zero exit. The //go:generate gate
		// fails the build rather than emitting half a container.
		fmt.Fprintf(os.Stderr, "godi-gen: %v\n", err)
		os.Exit(1)
	}

	if *out == "" {
		_, _ = os.Stdout.Write(src)
		return
	}
	if writeErr := os.WriteFile(*out, src, 0o644); writeErr != nil {
		fmt.Fprintf(os.Stderr, "godi-gen: write %s: %v\n", *out, writeErr)
		os.Exit(1)
	}
}

// generate is the testable generator entry point. It loads the registration
// package at srcDir, builds the plan.Registry from go/types facts, and emits the
// container as package pkgName, then type-checks the emitted source where it
// will live (typeCheckOutput). outPath is the -out file; "" means stdout. On any
// error it returns nil output (NFR2: a rejected graph emits zero bytes).
func generate(srcDir, pkgName, outPath string) ([]byte, error) {
	pkg, err := loadPackage(srcDir)
	if err != nil {
		return nil, err
	}
	reg, err := buildRegistry(pkg, pkgName)
	if err != nil {
		return nil, err
	}
	src, err := genlib.Generate(reg, pkgName)
	if err != nil {
		return nil, err
	}
	if err := typeCheckOutput(srcDir, pkgName, outPath, pkgName == pkg.Name, src); err != nil {
		return nil, err
	}
	return src, nil
}

// loadPackage statically loads the single registration package at dir with full
// type information. Type checking MUST succeed: Info.Instances is populated by
// the type checker, so any load/type error would leave the markers unreadable.
//
// Files in dir that godi-gen itself generated earlier (first line equals
// generatedHeader) are stubbed to bare package clauses via generatedFileOverlay:
// in the in-place layout a stale container_gen.go would otherwise fail to
// type-check (or parse) after a constructor signature changes and block the
// very regeneration that fixes it.
func loadPackage(dir string) (*packages.Package, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("load %q: %w", dir, err)
	}
	overlay, err := generatedFileOverlay(absDir)
	if err != nil {
		return nil, fmt.Errorf("load %q: %w", dir, err)
	}
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports |
			packages.NeedDeps,
		Dir:     dir,
		Overlay: overlay,
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("load %q: %w", dir, err)
	}
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("load %q: expected exactly 1 package, got %d", dir, len(pkgs))
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		msgs := make([]string, len(pkg.Errors))
		for i, e := range pkg.Errors {
			msgs[i] = e.Error()
		}
		return nil, fmt.Errorf("package %q failed to load/type-check:\n%s",
			dir, strings.Join(msgs, "\n"))
	}
	return pkg, nil
}

// registration captures one statically-read DSL marker. Reading happens in two
// passes (index service types, then discover constructors), so the raw type
// facts are captured here first in deterministic source order.
type registration struct {
	form     string        // "A" (AddSingleton[I,Impl]) or "B" (AddSingletonFactory[I])
	lifetime plan.Lifetime // derived from marker name: AddScoped*→Scoped, etc.
	pos      token.Pos     // marker position — sorted on for determinism (deferred 1.1)
	seq      int           // secondary sort key: body order of a multi-marker helper's registrations (0 for direct markers)
	iType    types.Type    // service type (first type argument)
	implType types.Type    // form A only: implementation type (second type argument)
	factory  ast.Expr      // form B only: the factory argument expression
	keyed    bool          // true for AddKeyed* markers
	keyExpr  ast.Expr      // keyed only: the key argument expression
	keyText  string        // keyed only: rendered key expression source text
	// nameSuffix disambiguates duplicate raw service names while preserving
	// GoType grouping for GetServices. Zero means no suffix; positive values
	// render as "__N" where N is 2 for the second duplicate, 3 for the third, etc.
	nameSuffix int

	// factoryTP/factoryTA carry the per-service helper's type parameters and the
	// call's closed type arguments for a form-B marker forwarded from a helper
	// body (Mechanism A): the factory expression text still mentions the type
	// parameters (NewMonitor[T], GetServices[Configure[T]]), so buildRegistry
	// rewrites T → the closed type when inlining. nil for a direct marker.
	factoryTP *types.TypeParamList
	factoryTA *types.TypeList
}

// buildRegistry turns the package's DSL markers into a validated-input
// plan.Registry. It maps each form-A constructor's dependency parameters back to
// registered service names, derives CtorErr / Disposable from go/types, and
// inlines form-B factory expressions (D-2) — rewritten for the emitted package
// (helper type-parameter closure, package qualification, import aliases). A
// form-A service with no New{Impl} aggregates into a generation error (AC2).
//
// pkgName is the package the container is emitted into. When it equals the
// analyzed package's name the container joins the registration package (the
// //go:generate-in-place layout), so a factory body's bare identifiers stay
// bare; otherwise they gain the registration package's qualifier.
func buildRegistry(pkg *packages.Package, pkgName string) (*plan.Registry, error) {
	regs, collectErrs := collectRegistrations(pkg)

	// NFR2: a package with zero recognized DSL markers must be rejected — emitting
	// an empty container (exit 0) would silently hand the user a container missing
	// every service. A same-named call that does not resolve to godi is surfaced
	// separately as a collectRegistrations stderr warning. When collection itself
	// reported errors, those positioned reasons win (they join genErrs below).
	if len(regs) == 0 && len(collectErrs) == 0 {
		return nil, fmt.Errorf("no godi DSL registrations recognized in package %q: "+
			"expected at least one godi.Add{Singleton,Scoped,Transient}[I, Impl] or "+
			"godi.Add{Singleton,Scoped,Transient}Factory[I] marker imported from %q",
			pkg.PkgPath, godiPkgPath)
	}

	// Resolve the godi.AsyncDisposable interface ONCE (Story 6.3, D-7): reading
	// the REAL interface type from the loaded godi package is more robust than
	// hand-synthesizing a signature that references the named type
	// context.Context (unlike io.Closer, whose signature uses only the universe
	// `error`). nil if godi is not imported — any package carrying godi.Add
	// markers imports it, so this is only nil in degenerate inputs, in which case
	// every service simply records AsyncDisposable=false.
	asyncIface := asyncDisposableInterface(pkg)

	// AC1 registration side (AD-17): a nil key == unkeyed. Normalize nil-keyed
	// registrations to plain unkeyed ones BEFORE the dependency-index pass, so a
	// nil-key service joins the unkeyed namespace fully — resolvable via GetService
	// AND injectable as a dependency target — symmetric to the runtime routing a
	// nil key to b.singletonFactories. Detected by source text ("nil"); this runs
	// before the D-1 constant check so `nil` (which has no constant value) is never
	// mistaken for a non-constant key.
	for i := range regs {
		if regs[i].keyed && regs[i].keyText == "nil" {
			regs[i].keyed = false
			regs[i].keyExpr = nil
			regs[i].keyText = ""
		}
	}
	rawNameCounts := make(map[string]int, len(regs))
	for i := range regs {
		raw := rawServiceName(regs[i])
		rawNameCounts[raw]++
		if rawNameCounts[raw] > 1 {
			regs[i].nameSuffix = rawNameCounts[raw]
		}
	}

	// Pass 1: index every service's type(s) -> its service Name, so a
	// constructor parameter type can be resolved to the dependency it names. Both
	// the service interface (I) and the implementation (Impl, and *Impl) are
	// indexed so a constructor may inject either form. Keyed services are NOT
	// indexed here — keyed and unkeyed are independent namespaces (D-3).
	// elemIndex additionally maps a service type to ALL registrations carrying it
	// (multi-impl), for factoryDeps' GetServices ordering constraints.
	depIndex := make(map[string]string, len(regs)*2)
	elemIndex := make(map[string][]string, len(regs))
	for _, rg := range regs {
		if rg.keyed {
			continue
		}
		name := serviceName(rg)
		depIndex[typeKey(rg.iType)] = name
		elemIndex[typeKey(rg.iType)] = append(elemIndex[typeKey(rg.iType)], name)
		if rg.form == "A" && rg.implType != nil {
			depIndex[typeKey(rg.implType)] = name
			depIndex[typeKey(types.NewPointer(rg.implType))] = name
		}
	}

	reg := plan.NewRegistry()
	localImportPath := ""
	if pkgName == pkg.Name {
		localImportPath = pkg.PkgPath
	}
	imports := newImportSet(localImportPath)
	// A call that resolves to a godi marker (or a same-package per-service
	// helper) but cannot be read is a hard error: skipping it would emit a
	// container silently missing that service (NFR2).
	genErrs := collectErrs

	// keyedConstValues tracks (GoType, constant value string) to detect
	// value-equal but text-distinct keys that would produce duplicate switch
	// case arms in the generated container (review F1).
	type keyedSeen struct {
		goTypeKey string
		valStr    string
	}
	seenKeyed := make(map[keyedSeen]string) // value → first keyText

	for _, rg := range regs {
		s := plan.Service{
			Name:     serviceName(rg),
			GoType:   renderType(rg.iType, imports),
			Lifetime: rg.lifetime,
		}
		if rg.keyed && rg.keyText != "" {
			// AC4 (D-3): AnyKey is a query-only sentinel — reject it as a registration
			// key with a SPECIFIC message, BEFORE the generic non-constant check below
			// (which would otherwise fire, because AnyKey is a var, not a constant, and
			// bury the real reason). dotnet's catch-all registration — a service
			// resolved by ANY concrete key, with that key injected — has no Go
			// mechanism: form-B func(Provider) I / form-A New{Impl} cannot receive a
			// resolve-time key. Both providers reject AnyKey as a key.
			if isAnyKeyExpr(pkg, rg.keyExpr) {
				genErrs = append(genErrs, fmt.Sprintf(
					"keyed service %q: AnyKey is a query-only sentinel and cannot be used as a "+
						"registration key (catch-all registration has no Go mechanism, AD-17/D-3)",
					s.Name))
				continue
			}

			s.Key = rg.keyText

			// D-1: codegen emits key text as a case expression; non-constant
			// keys produce invalid or non-deterministic switch arms.
			tv, tvOK := pkg.TypesInfo.Types[rg.keyExpr]
			if !tvOK || tv.Value == nil {
				genErrs = append(genErrs, fmt.Sprintf(
					"keyed service %q: key expression %s is not a compile-time constant; "+
						"codegen requires constant keys for switch case arms (D-1)",
					s.Name, rg.keyText))
				continue
			}

			// Detect value-equal but text-distinct keys for the same type,
			// which would produce duplicate case arms → compile error.
			sk := keyedSeen{goTypeKey: typeKey(rg.iType), valStr: tv.Value.String()}
			if prev, dup := seenKeyed[sk]; dup && prev != rg.keyText {
				genErrs = append(genErrs, fmt.Sprintf(
					"keyed service %q: key %s has the same value as %s "+
						"(both evaluate to %s); duplicate keys for the same type "+
						"produce duplicate switch case arms",
					s.Name, rg.keyText, prev, tv.Value.String()))
				continue
			}
			seenKeyed[sk] = rg.keyText
		}

		if rg.form == "B" {
			// Form B names the service after its service type. Named types keep their
			// historical short name; concrete pointer/slice/etc. service types are
			// normalized by serviceBaseName into a valid identifier.
			// For keyed services, the Name contains "#key" suffix which is sanitized
			// later — check only the base part before the "#".
			baseName := s.Name
			if idx := strings.Index(baseName, "#"); idx >= 0 {
				baseName = baseName[:idx]
			}
			if !token.IsIdentifier(baseName) {
				genErrs = append(genErrs, fmt.Sprintf(
					"form-B service: type argument %s produced invalid service name %q",
					s.GoType, s.Name))
				continue
			}
			// Form B escape hatch (D-2): inline the factory expression, rewritten
			// for the emitted package — helper type parameters closed, package
			// qualifiers remapped to the emitted file's import aliases, and (when
			// emitting into a different package) bare registration-package
			// identifiers qualified. No New{Impl} discovery.
			factoryText, ferr := rewriteFactoryExpr(pkg, rg.factory, rg.factoryTP, rg.factoryTA, imports, pkgName == pkg.Name)
			if ferr != nil {
				genErrs = append(genErrs, fmt.Sprintf(
					"form-B service %q: %v", s.Name, ferr))
				continue
			}
			s.FactoryExpr = factoryText
			// The factory's godi.Get… calls are read as dependencies for
			// Validate's static checks (cycles, captive dependencies). Emitted
			// accessors build every service on first access, so they carry no
			// ordering role. Keyed registrations are skipped — keyed deps live in
			// an independent namespace (D-3).
			if !rg.keyed {
				s.Deps = factoryDeps(pkg, rg.factory, rg.factoryTP, rg.factoryTA, depIndex, elemIndex)
			}
			reg.Service(s)
			continue
		}

		// Form A: discover the unique New{Impl} constructor (AD-8).
		impl, ok := rg.implType.(*types.Named)
		if !ok {
			genErrs = append(genErrs, fmt.Sprintf(
				"form-A service %q: implementation type %s is not a named type",
				s.Name, shortTypeName(rg.implType)))
			continue
		}
		implName := impl.Obj().Name()
		implPkg := impl.Obj().Pkg()
		if implPkg == nil {
			genErrs = append(genErrs, fmt.Sprintf(
				"form-A service %q: implementation %s has no package", s.Name, implName))
			continue
		}

		ctorObj := implPkg.Scope().Lookup("New" + implName)
		ctorFn, _ := ctorObj.(*types.Func)
		if ctorFn == nil {
			// AC2 gen-time gate: no New{Impl} -> reject with a form-B hint.
			genErrs = append(genErrs, fmt.Sprintf(
				"form-A service %q (%s): no constructor New%s found in package %q. "+
					"Add a New%s constructor, or register it with form B "+
					"godi.AddSingletonFactory[%s](r, factory) (AD-8).",
				implName, shortTypeName(rg.iType), implName, implPkg.Path(),
				implName, shortTypeName(rg.iType)))
			continue
		}

		sig, ok := ctorFn.Type().(*types.Signature)
		if !ok {
			genErrs = append(genErrs, fmt.Sprintf(
				"form-A service %q: New%s is not a function", s.Name, implName))
			continue
		}

		// For a CLOSED GENERIC implementation the discovered New{Impl} is itself
		// generic (its parameters/results mention the type parameter T). Instantiate
		// its signature with the impl's type arguments so result validation and
		// dependency reading see the CONCRETE parameter types (value IFakeService),
		// not the un-instantiated type parameter (value T). Without this
		// constructorDeps reads an unresolvable *types.TypeParam and Validate reports
		// a phantom missing dependency (AC2). The New{Impl} convention keeps the
		// constructor's type parameters aligned with the impl's, so the impl's type
		// args instantiate the ctor signature directly (single type parameter in
		// Story 4.1; multi/permuted parameters are deferred).
		if typeArgs := impl.TypeArgs(); typeArgs != nil && typeArgs.Len() > 0 {
			if typeArgs.Len() > 1 {
				fmt.Fprintf(os.Stderr, "godi-gen: warning: form-A service %q has %d type arguments — "+
					"multi-type-parameter closed generics are deferred; skipping ctor "+
					"instantiation (only single type parameter is supported)\n", s.Name, typeArgs.Len())
			}
			targs := make([]types.Type, typeArgs.Len())
			for i := range targs {
				targs[i] = typeArgs.At(i)
			}
			instSig, instErr := types.Instantiate(nil, ctorFn.Type(), targs, false)
			if instErr != nil {
				genErrs = append(genErrs, fmt.Sprintf(
					"form-A service %q: cannot instantiate constructor New%s with the closed "+
						"type arguments: %v", s.Name, implName, instErr))
				continue
			}
			if isig, ok := instSig.(*types.Signature); ok {
				sig = isig
			}
		}

		// Validate the result shape before trusting CtorErr: New{Impl} must return
		// exactly (Impl) or (Impl, error). Any other shape — (T, *CustomError),
		// (T1, T2, error), zero results — would make the emitted single/comma-err
		// assignment mismatch the real arity and produce a non-compilable container,
		// so reject it at gen time with a clear message (AD-8) instead.
		if resErr := validateCtorResults(sig); resErr != nil {
			genErrs = append(genErrs, fmt.Sprintf(
				"form-A service %q: New%s %v", s.Name, implName, resErr))
			continue
		}

		// Qualify the ctor by the emitted file's identifier for its package —
		// the assigned alias when the package's name clashes with another import.
		// Same-package constructors stay bare because an emitted file cannot import
		// its own package.
		if qualifier := imports.aliasOf(implPkg); qualifier != "" {
			s.Ctor = qualifier + ".New" + implName
		} else {
			s.Ctor = "New" + implName
		}
		if typeArgs := impl.TypeArgs(); typeArgs != nil && typeArgs.Len() > 0 {
			args := make([]string, typeArgs.Len())
			for i := range args {
				args[i] = renderType(typeArgs.At(i), imports)
			}
			s.Ctor += "[" + strings.Join(args, ", ") + "]"
		}
		s.Deps = constructorDeps(sig, depIndex, imports)
		s.CtorErr = hasTrailingError(sig)
		s.Disposable = implementsCloser(rg.implType)
		s.AsyncDisposable = implementsAsyncDisposable(rg.implType, asyncIface)

		reg.Service(s)
	}

	if len(genErrs) > 0 {
		return nil, fmt.Errorf("generation rejected (%d error(s)):\n%s",
			len(genErrs), strings.Join(genErrs, "\n"))
	}

	reg.Imports = imports.list()
	return reg, nil
}

// collectRegistrations walks the package AST for godi.Add… marker calls AND
// per-service open-generic helpers (Mechanism A, AC1), reads their type arguments
// from Info.Instances, and returns them sorted by source position. Sorting makes
// the registry order deterministic regardless of map iteration order (deferred 1.1).
//
// The second result lists positioned error texts ("file:line:col: reason") for
// calls that resolve to a godi marker or to a same-package per-service helper
// but cannot be read; buildRegistry rejects generation with them. A same-named
// call that does not resolve to godi is only warned about (warnUnrecognized).
func collectRegistrations(pkg *packages.Package) ([]registration, []string) {
	var out []registration
	var errs []string

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			// A call whose bare callee name is a godi marker is a SUSPECTED marker.
			// If it does not resolve to the godi package (a same-named local
			// function, a vanity/forked import path), warn instead of silently
			// dropping it (NFR2 no silent drop).
			suspected := isDSLName(calleeName(call))

			// markerCallee handles a godi selector (godi.AddSingleton[…] or, with
			// the type arguments inferred, godi.AddSingletonFactory(…)), a
			// dot-imported bare marker, and a bare-ident per-service helper call
			// (AddRepository[User]). Recognition is by what the identifier
			// RESOLVES to, never by whether the call spells its type arguments.
			ident := markerCallee(call)
			if ident == nil {
				warnUnrecognized(pkg, call, suspected, "callee is not a plain function name")
				return true
			}
			fn, ok := pkg.TypesInfo.Uses[ident].(*types.Func)
			if !ok || fn.Pkg() == nil {
				warnUnrecognized(pkg, call, suspected,
					"callee does not resolve to a package-level function")
				return true
			}

			// Direct godi marker (the primary path). Other godi functions
			// (godi.Get, godi.Marked, …) are not markers and are ignored. A marker
			// whose type arguments still mention a type parameter — the TEMPLATE
			// inside a per-service helper body, godi.AddTransient[IFoo[T], Foo[T]]
			// — is skipped: only the helper instantiation that forwards to it is a
			// concrete registration (AC1).
			if fn.Pkg().Path() == godiPkgPath {
				if !isDSLName(fn.Name()) {
					return true
				}
				var err error
				if out, err = appendGodiMarker(out, pkg, ident, fn, call); err != nil {
					errs = append(errs, err.Error())
				}
				return true
			}

			// Per-service open-generic helper (Mechanism A, AC1): a generic function
			// DEFINED IN THE ANALYZED PACKAGE whose body forwards to godi markers.
			// readPerServiceHelper follows the forwarder and substitutes the
			// helper's type parameters with this call's type arguments. A helper
			// it cannot reduce is an error; an ordinary call (reflect.TypeFor, a
			// non-generic local function, …) returns handled=false and falls through.
			regs, handled, err := readPerServiceHelper(pkg, call, ident, fn)
			if err != nil {
				errs = append(errs, err.Error())
				return true
			}
			if handled {
				out = append(out, regs...)
				return true
			}
			warnUnrecognized(pkg, call, suspected, fmt.Sprintf(
				"callee does not resolve to the godi package %q (vanity import, fork, or replace?)",
				godiPkgPath))
			return true
		})
	}

	sort.Slice(out, func(i, j int) bool {
		pi := pkg.Fset.Position(out[i].pos)
		pj := pkg.Fset.Position(out[j].pos)
		if pi.Filename != pj.Filename {
			return pi.Filename < pj.Filename
		}
		if pi.Offset != pj.Offset {
			return pi.Offset < pj.Offset
		}
		// Same position = registrations reduced from ONE multi-marker helper
		// call; seq keeps them in helper-body order.
		return out[i].seq < out[j].seq
	})
	return out, errs
}

// markerError formats a collection error as "file:line:col: reason".
func markerError(pkg *packages.Package, call *ast.CallExpr, reason string) error {
	return fmt.Errorf("%s: %s", pkg.Fset.Position(call.Pos()), reason)
}

// appendGodiMarker reads a direct godi.Add… marker call and appends the
// registration it denotes. A marker whose type arguments still mention a type
// parameter (the TEMPLATE marker inside a per-service helper body) is SKIPPED —
// it is not a concrete registration; only the helper call that forwards to it is
// (Mechanism A skip rule, AC1). The unified rule: the generator only processes
// registrations whose type arguments are fully concrete.
//
// The type arguments come from Info.Instances whether the call spells them
// (godi.AddSingletonFactory[ILog](&r, f)) or Go inferred them
// (godi.AddSingletonFactory(&r, f)). A marker that still cannot be read is an
// error, never a skip (NFR2).
func appendGodiMarker(out []registration, pkg *packages.Package, ident *ast.Ident, fn *types.Func, call *ast.CallExpr) ([]registration, error) {
	inst, ok := pkg.TypesInfo.Instances[ident]
	if !ok || inst.TypeArgs == nil {
		return out, markerError(pkg, call, fmt.Sprintf(
			"godi.%s call has no type arguments recorded by the type checker; "+
				"write them explicitly, e.g. godi.%s[I, …](…)", fn.Name(), fn.Name()))
	}
	ta := inst.TypeArgs
	if typeArgsHaveParam(ta) {
		return out, nil
	}
	out, err := appendMarkerRegistration(out, pkg, fn.Name(), call.Pos(), ta, call.Args)
	if err != nil {
		return out, markerError(pkg, call, err.Error())
	}
	return out, nil
}

// appendMarkerRegistration builds the registration for a godi marker named
// markerName with type arguments ta and call arguments args. It is the shared
// reduction target for a direct concrete marker AND a per-service-helper-reduced
// closed marker (Mechanism A): both arrive here as a fully-concrete (markerName,
// ta) pair, so the two paths produce identical registrations (AD-14).
//
// A type-argument count or a missing factory argument that does not match the
// marker's shape is returned as an error (the caller adds the source position).
func appendMarkerRegistration(out []registration, pkg *packages.Package, markerName string, pos token.Pos, ta *types.TypeList, args []ast.Expr) ([]registration, error) {
	wantTA := func(n int) error {
		return fmt.Errorf("godi.%s needs %d type argument(s), the type checker recorded %d",
			markerName, n, ta.Len())
	}
	switch markerName {
	case "AddSingleton", "AddScoped", "AddTransient":
		if ta.Len() != 2 {
			return out, wantTA(2)
		}
		return append(out, registration{
			form: "A", lifetime: lifetimeFromName(markerName), pos: pos,
			iType: ta.At(0), implType: ta.At(1),
		}), nil
	case "AddSingletonFactory", "AddScopedFactory", "AddTransientFactory":
		if ta.Len() != 1 {
			return out, wantTA(1)
		}
		if len(args) < 2 {
			return out, fmt.Errorf("godi.%s has no factory argument; pass func(godi.Provider) I", markerName)
		}
		return append(out, registration{
			form: "B", lifetime: lifetimeFromName(markerName), pos: pos,
			iType: ta.At(0), factory: args[1],
		}), nil
	case "AddKeyedSingleton", "AddKeyedScoped", "AddKeyedTransient":
		if ta.Len() != 2 {
			return out, wantTA(2)
		}
		var keyArg ast.Expr
		var keyTxt string
		if len(args) >= 2 {
			keyArg = args[1]
			keyTxt = renderExpr(pkg.Fset, keyArg)
		}
		return append(out, registration{
			form: "A", lifetime: lifetimeFromName(markerName), pos: pos,
			iType: ta.At(0), implType: ta.At(1),
			keyed: true, keyExpr: keyArg, keyText: keyTxt,
		}), nil
	case "AddKeyedSingletonFactory", "AddKeyedScopedFactory", "AddKeyedTransientFactory":
		if ta.Len() != 1 {
			return out, wantTA(1)
		}
		if len(args) < 3 {
			return out, fmt.Errorf("godi.%s has no factory argument; pass func(godi.Provider) I", markerName)
		}
		keyArg := args[1]
		return append(out, registration{
			form: "B", lifetime: lifetimeFromName(markerName), pos: pos,
			iType: ta.At(0), factory: args[2],
			keyed: true, keyExpr: keyArg, keyText: renderExpr(pkg.Fset, keyArg),
		}), nil
	}
	return out, nil
}

// genericCallee extracts the function-name identifier of a generic instantiation
// call, handling BOTH a package selector (godi.AddSingleton[…]) and a bare
// identifier (a same-package per-service helper AddRepository[User]). It returns
// nil when the call is not a generic instantiation of a function. The bare-ident
// form is what distinguishes a local helper call from a godi marker selector.
// Marker recognition uses markerCallee instead (type arguments may be
// inferred); genericCallee remains for factoryDeps' godi.Get[…] ordering hints.
func genericCallee(call *ast.CallExpr) *ast.Ident {
	var x ast.Expr
	switch fun := call.Fun.(type) {
	case *ast.IndexExpr:
		x = fun.X
	case *ast.IndexListExpr:
		x = fun.X
	default:
		return nil
	}
	switch e := x.(type) {
	case *ast.SelectorExpr:
		return e.Sel
	case *ast.Ident:
		return e
	}
	return nil
}

// markerCallee extracts the function-name identifier a call invokes, with or
// without explicit type arguments: godi.AddSingleton[I, Impl](…),
// godi.AddSingletonFactory(…) (type argument inferred), a dot-imported
// AddSingleton[…](…), or a same-package helper AddRepository[User](…). Unlike
// genericCallee it does not require an instantiation in the source; the caller
// decides by what the identifier resolves to (Info.Uses / Info.Instances).
func markerCallee(call *ast.CallExpr) *ast.Ident {
	fun := call.Fun
	switch f := fun.(type) {
	case *ast.IndexExpr:
		fun = f.X
	case *ast.IndexListExpr:
		fun = f.X
	}
	switch e := fun.(type) {
	case *ast.SelectorExpr:
		return e.Sel
	case *ast.Ident:
		return e
	}
	return nil
}

// readPerServiceHelper recognizes a per-service open-generic helper call
// (Mechanism A, AC1) and reduces it to the closed registrations it denotes.
//
// A per-service helper is a generic function DEFINED IN THE ANALYZED PACKAGE
// whose body is a sequence of godi.Add… marker statements (each an expression
// statement or a `return <marker>`; a final `return r` of the registry
// parameter is also allowed). The generator reads the OUTER call's type
// arguments (AddRepository[User] → [User]) and substitutes the helper's type
// parameters in each inner marker's type arguments via types.Instantiate to
// materialize the closed registrations — form A flows through the SAME pipeline
// as a hand-written closed marker (AD-14); a form-B (factory) marker carries
// its factory expression plus the substitution context, and buildRegistry
// rewrites the factory text (T → closed type) when inlining.
//
// handled=true means the call is (or clearly intends to be) a helper, so the
// caller does not additionally warn; handled=false means it is an ordinary call
// (reflect.TypeFor, a non-generic local function, …) the caller treats by its
// suspected-name rule. A call that IS a same-package generic helper forwarding
// to godi markers but whose shape is unsupported returns a positioned err.
//
// Scope: helpers in the analyzed package, forwarding to UNKEYED markers (form A
// and form B), single type parameter. Mixed bodies, keyed forwarders, and
// multi/permuted type parameters reject generation — never silently dropped
// (NFR2). Cross-package helpers are not followed (handled=false). An
// unsupported marker rejects the WHOLE helper call: materializing a subset of
// the helper's registrations would hand the user a container silently missing
// services.
func readPerServiceHelper(pkg *packages.Package, call *ast.CallExpr, ident *ast.Ident, fn *types.Func) (regs []registration, handled bool, err error) {
	// Only a generic package-level function can be a per-service helper. A
	// non-generic local function (func register(r *godi.Registry) { … }) has
	// its concrete markers read directly by the AST walk.
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() != nil || sig.TypeParams().Len() == 0 {
		return nil, false, nil
	}
	decl := findFuncDecl(pkg, fn)
	if decl == nil || decl.Body == nil {
		return nil, false, nil // not defined in the analyzed package (e.g. reflect.TypeFor)
	}
	markers, pure := helperBodyMarkers(pkg, decl)
	if len(markers) == 0 {
		return nil, false, nil // body forwards to no godi marker — ordinary func
	}
	// From here the call is INTENDED as a per-service helper; any shortfall
	// rejects generation (NFR2).
	reject := func(reason string) ([]registration, bool, error) {
		return nil, true, markerError(pkg, call, fmt.Sprintf(
			"per-service helper %s(...) cannot be reduced to registrations: %s; "+
				"rewrite the helper into a supported shape or register its services "+
				"with explicit godi.Add… markers", fn.Name(), reason))
	}
	if !pure {
		return reject("its body mixes godi markers with other statements — every " +
			"statement must be a godi.Add… call (or a final `return r`)")
	}

	outerInst, ok := pkg.TypesInfo.Instances[ident]
	if !ok || outerInst.TypeArgs == nil || outerInst.TypeArgs.Len() == 0 {
		return reject("the call has no type arguments recorded by the type checker")
	}
	tparams := sig.TypeParams()
	outerArgs := outerInst.TypeArgs
	if tparams.Len() != outerArgs.Len() {
		return reject("type-parameter / type-argument count mismatch")
	}
	if tparams.Len() > 1 {
		return reject(fmt.Sprintf("it has %d type parameters — only a single "+
			"type parameter is supported", tparams.Len()))
	}

	for i, m := range markers {
		markerName := m.fn.Name()
		innerInst, instOK := pkg.TypesInfo.Instances[m.ident]

		switch markerName {
		case "AddSingleton", "AddScoped", "AddTransient":
			if !instOK || innerInst.TypeArgs == nil || innerInst.TypeArgs.Len() != 2 {
				return reject("its form-A marker must have [I, Impl] type arguments")
			}
			innerArgs := innerInst.TypeArgs
			iClosed, err1 := substituteTypeParams(innerArgs.At(0), tparams, outerArgs)
			implClosed, err2 := substituteTypeParams(innerArgs.At(1), tparams, outerArgs)
			if err1 != nil || err2 != nil {
				return reject(fmt.Sprintf("type substitution failed: %v / %v", err1, err2))
			}
			// pos is the OUTER helper call (distinct per closure); seq keeps the
			// helper-body marker order within one closure (collectRegistrations
			// sorts on both).
			regs = append(regs, registration{
				form: "A", lifetime: lifetimeFromName(markerName), pos: call.Pos(), seq: i,
				iType: iClosed, implType: implClosed,
			})

		case "AddSingletonFactory", "AddScopedFactory", "AddTransientFactory":
			if !instOK || innerInst.TypeArgs == nil || innerInst.TypeArgs.Len() != 1 {
				return reject("its form-B marker must have an [I] type argument")
			}
			if len(m.call.Args) < 2 {
				return reject("its form-B marker has no factory argument")
			}
			iClosed, err := substituteTypeParams(innerInst.TypeArgs.At(0), tparams, outerArgs)
			if err != nil {
				return reject(fmt.Sprintf("type substitution failed: %v", err))
			}
			regs = append(regs, registration{
				form: "B", lifetime: lifetimeFromName(markerName), pos: call.Pos(), seq: i,
				iType: iClosed, factory: m.call.Args[1],
				factoryTP: tparams, factoryTA: outerArgs,
			})

		default:
			return reject(fmt.Sprintf("it forwards to godi.%s — only unkeyed "+
				"Add{Singleton,Scoped,Transient}[I, Impl] and "+
				"Add{Singleton,Scoped,Transient}Factory[I] are supported "+
				"(keyed forwarders are not)", markerName))
		}
	}

	return regs, true, nil
}

// findFuncDecl returns the *ast.FuncDecl in the analyzed package that declares
// fn, or nil when fn is not defined there (an imported function such as
// reflect.TypeFor). It matches via TypesInfo.Defs — Go has no overloading, so the
// ident→object identity is exact.
func findFuncDecl(pkg *packages.Package, fn *types.Func) *ast.FuncDecl {
	for _, file := range pkg.Syntax {
		for _, d := range file.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Name == nil {
				continue
			}
			if pkg.TypesInfo.Defs[fd.Name] == fn {
				return fd
			}
		}
	}
	return nil
}

// helperMarker is one godi.Add… marker statement inside a per-service helper
// body: the marker call, its generic-callee ident, and the resolved godi
// *types.Func.
type helperMarker struct {
	call  *ast.CallExpr
	ident *ast.Ident
	fn    *types.Func
}

// helperBodyMarkers extracts the godi markers a helper body forwards to, in
// body order. A recognized body consists of statements that are each one of:
//
//   - an expression statement whose call resolves to a godi.Add… marker
//     (godi.AddSingleton[I[T], Impl[T]](r) — the multi-statement form), or
//   - a `return <marker call>` (the classic single-statement forwarder, or a
//     multi-statement body's final registration), or
//   - a final `return r` returning the registry parameter unchanged.
//
// pure=false reports a body that contains AT LEAST one marker statement but
// also statements outside this shape — the caller rejects generation instead of
// materializing a partial registration set (NFR2). A body with NO marker
// statements returns (nil, whatever): it is an ordinary generic function, not a
// helper.
func helperBodyMarkers(pkg *packages.Package, decl *ast.FuncDecl) (markers []helperMarker, pure bool) {
	pure = true
	if decl.Body == nil {
		return nil, true
	}
	for i, stmt := range decl.Body.List {
		var result ast.Expr
		switch st := stmt.(type) {
		case *ast.ExprStmt:
			result = st.X
		case *ast.ReturnStmt:
			if len(st.Results) != 1 {
				pure = false
				continue
			}
			result = st.Results[0]
			// A final bare `return r` (any lone identifier — the type checker
			// already constrains it to the declared result type) carries no
			// marker and is part of the recognized shape.
			if _, isIdent := result.(*ast.Ident); isIdent && i == len(decl.Body.List)-1 {
				continue
			}
		default:
			pure = false
			continue
		}
		if paren, ok := result.(*ast.ParenExpr); ok {
			result = paren.X
		}
		inner, ok := result.(*ast.CallExpr)
		if !ok {
			pure = false
			continue
		}
		// markerCallee: an inner marker may have its type arguments inferred
		// (godi.AddSingletonFactory(r, func(godi.Provider) IFoo[T] {…})).
		innerIdent := markerCallee(inner)
		if innerIdent == nil {
			pure = false
			continue
		}
		innerFn, ok := pkg.TypesInfo.Uses[innerIdent].(*types.Func)
		if !ok || innerFn.Pkg() == nil || innerFn.Pkg().Path() != godiPkgPath ||
			!isDSLName(innerFn.Name()) {
			pure = false
			continue
		}
		markers = append(markers, helperMarker{call: inner, ident: innerIdent, fn: innerFn})
	}
	return markers, pure
}

// substituteTypeParams returns t with every helper type parameter (tparams[i])
// replaced by the corresponding call type argument (targs[i]), re-instantiating
// any generic named type whose arguments changed. It handles the Story 4.1
// shapes: a bare type parameter (T), a generic named type whose arguments mention
// the parameters (IFoo[T]), and pointer/slice wrappers; nested generics recurse.
func substituteTypeParams(t types.Type, tparams *types.TypeParamList, targs *types.TypeList) (types.Type, error) {
	switch ty := t.(type) {
	case *types.TypeParam:
		for i := 0; i < tparams.Len(); i++ {
			if tparams.At(i).Obj() == ty.Obj() {
				return targs.At(i), nil
			}
		}
		return t, nil
	case *types.Named:
		ntargs := ty.TypeArgs()
		if ntargs == nil || ntargs.Len() == 0 {
			return t, nil
		}
		newArgs := make([]types.Type, ntargs.Len())
		changed := false
		for i := 0; i < ntargs.Len(); i++ {
			sub, err := substituteTypeParams(ntargs.At(i), tparams, targs)
			if err != nil {
				return nil, err
			}
			if sub != ntargs.At(i) {
				changed = true
			}
			newArgs[i] = sub
		}
		if !changed {
			return t, nil
		}
		return types.Instantiate(nil, ty.Origin(), newArgs, false)
	case *types.Pointer:
		elem, err := substituteTypeParams(ty.Elem(), tparams, targs)
		if err != nil {
			return nil, err
		}
		if elem == ty.Elem() {
			return t, nil
		}
		return types.NewPointer(elem), nil
	case *types.Slice:
		elem, err := substituteTypeParams(ty.Elem(), tparams, targs)
		if err != nil {
			return nil, err
		}
		if elem == ty.Elem() {
			return t, nil
		}
		return types.NewSlice(elem), nil
	}
	if hasTypeParam(t) {
		return nil, fmt.Errorf("unsupported type shape %T containing a type parameter — "+
			"only Named, Pointer, and Slice wrappers are handled (map/chan/func/array are deferred)", t)
	}
	return t, nil
}

// typeArgsHaveParam reports whether any type argument in ta still mentions a type
// parameter — the signature of a TEMPLATE marker inside a per-service helper body
// (godi.AddTransient[IFoo[T], Foo[T]]), which is NOT a concrete registration.
func typeArgsHaveParam(ta *types.TypeList) bool {
	if ta == nil {
		return false
	}
	for i := 0; i < ta.Len(); i++ {
		if hasTypeParam(ta.At(i)) {
			return true
		}
	}
	return false
}

// hasTypeParam reports whether t mentions a *types.TypeParam anywhere (the type
// itself, a generic type's arguments, or a pointer/slice element). It is the test
// behind "the generator only processes registrations whose type arguments are
// fully concrete" — skipping un-instantiated helper template markers (AC1).
func hasTypeParam(t types.Type) bool {
	switch ty := t.(type) {
	case *types.TypeParam:
		return true
	case *types.Named:
		if targs := ty.TypeArgs(); targs != nil {
			for i := 0; i < targs.Len(); i++ {
				if hasTypeParam(targs.At(i)) {
					return true
				}
			}
		}
	case *types.Pointer:
		return hasTypeParam(ty.Elem())
	case *types.Slice:
		return hasTypeParam(ty.Elem())
	}
	return false
}

// isAnyKeyExpr reports whether expr refers to godi.AnyKey — the query-only
// sentinel that cannot be a registration key (AC4, D-3). It resolves the
// referenced object (a selector godi.AnyKey, or a dot-imported AnyKey ident) via
// go/types and checks it is the AnyKey var declared in the godi package, so a
// same-named symbol from another package is not misdetected.
func isAnyKeyExpr(pkg *packages.Package, expr ast.Expr) bool {
	var id *ast.Ident
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		id = e.Sel
	case *ast.Ident:
		id = e
	default:
		return false
	}
	obj := pkg.TypesInfo.Uses[id]
	if obj == nil {
		return false
	}
	return obj.Name() == "AnyKey" && obj.Pkg() != nil && obj.Pkg().Path() == godiPkgPath
}

// calleeName returns the bare function name a call invokes, unwrapping a generic
// instantiation (foo[T] / foo[T1, T2]) and a package selector (pkg.foo) or a bare
// identifier (a dot-imported foo). It is empty when the callee is not a plain
// function name. Used to flag a SUSPECTED DSL marker that does not resolve to
// godi, so warnUnrecognized can report it instead of dropping it silently.
func calleeName(call *ast.CallExpr) string {
	if id := markerCallee(call); id != nil {
		return id.Name
	}
	return ""
}

// isDSLName reports whether name is one of godi's DSL marker function names.
func isDSLName(name string) bool {
	switch name {
	case "AddSingleton", "AddSingletonFactory",
		"AddScoped", "AddScopedFactory",
		"AddTransient", "AddTransientFactory",
		"AddKeyedSingleton", "AddKeyedSingletonFactory",
		"AddKeyedScoped", "AddKeyedScopedFactory",
		"AddKeyedTransient", "AddKeyedTransientFactory":
		return true
	}
	return false
}

// lifetimeFromName maps a DSL marker function name to its plan.Lifetime.
func lifetimeFromName(name string) plan.Lifetime {
	switch {
	case strings.Contains(name, "Transient"):
		return plan.Transient
	case strings.Contains(name, "Scoped"):
		return plan.Scoped
	default:
		return plan.Singleton
	}
}

// warnUnrecognized emits a build-time warning to stderr when a call that looks
// like a godi DSL marker does not resolve to the godi package (NFR2's "no
// silent drop"). It is a warning, not an error: a same-named function from
// another package is a legitimate non-match. A call that DOES resolve to a godi
// marker but cannot be read is an error instead (collectRegistrations). Static
// analysis only — no user code runs (AD-7).
func warnUnrecognized(pkg *packages.Package, call *ast.CallExpr, suspected bool, reason string) {
	if !suspected {
		return
	}
	fmt.Fprintf(os.Stderr,
		"godi-gen: warning: %s: suspected DSL call %s(...) not recognized as a godi marker (%s); registration skipped\n",
		pkg.Fset.Position(call.Pos()), calleeName(call), reason)
}

// serviceName is the f_<Name> field name and dependency key for a registration.
// Form A names the service after its implementation (FakeService, Consumer);
// form B has no implementation, so it names the service after its interface.
// Keyed services append #keyExpr for uniqueness (D-2).
func serviceName(rg registration) string {
	base := rawServiceName(rg)
	if rg.nameSuffix > 1 {
		base += fmt.Sprintf("__%d", rg.nameSuffix)
	}
	return base
}

func rawServiceName(rg registration) string {
	base := serviceBaseName(rg)
	if rg.keyed && rg.keyText != "" {
		return base + "#" + rg.keyText
	}
	return base
}

// serviceBaseName derives the key-free base name for a registration. Form A
// names the service after its implementation; form B (and a non-named impl)
// after its interface. For a CLOSED GENERIC instantiation the type arguments are
// folded into the name as a sanitized identifier segment (AC4) so two
// instantiations of the same generic — Foo[A] and Foo[B] — get DISTINCT names.
// Without this they would both be "Foo": plan.Validate would report a duplicate
// (validate.go) and the generated f_Foo field/method/accessor identifiers would
// collide (deferred-work.md:57). The type-arg segment is sanitized HERE rather
// than in genlib so the unkeyed emit path — which uses the raw Name verbatim as
// f_<Name> / New<Name>() / <Name>() — stays valid Go without sanitizing every
// genlib emit site (D-4).
func serviceBaseName(rg registration) string {
	var named *types.Named
	if rg.form == "A" && rg.implType != nil {
		named, _ = rg.implType.(*types.Named)
	}
	if named == nil {
		named, _ = rg.iType.(*types.Named)
	}
	if named == nil {
		return serviceTypeIdent(rg.iType)
	}
	base := named.Obj().Name()
	if targs := named.TypeArgs(); targs != nil && targs.Len() > 0 {
		base += "_" + sanitizeIdent(typeArgsString(targs))
	}
	return base
}

func serviceTypeIdent(t types.Type) string {
	switch tt := t.(type) {
	case *types.Pointer:
		return "Ptr_" + serviceTypeIdent(tt.Elem())
	case *types.Slice:
		return "Slice_" + serviceTypeIdent(tt.Elem())
	case *types.Array:
		return "Array_" + serviceTypeIdent(tt.Elem())
	case *types.Map:
		return "Map_" + serviceTypeIdent(tt.Key()) + "_" + serviceTypeIdent(tt.Elem())
	case *types.Named:
		base := tt.Obj().Name()
		if targs := tt.TypeArgs(); targs != nil && targs.Len() > 0 {
			base += "_" + sanitizeIdent(typeArgsString(targs))
		}
		return base
	default:
		return sanitizeIdent(types.TypeString(t, func(p *types.Package) string { return p.Name() }))
	}
}

// typeArgsString renders a closed generic's type arguments package-NAME qualified
// and comma-joined (e.g. "fakes.IFakeService"). The caller sanitizes it into an
// identifier segment for the service Name (AC4); the rendering itself mirrors the
// Ctor's type-arg rendering so name and ctor stay consistent.
func typeArgsString(targs *types.TypeList) string {
	parts := make([]string, targs.Len())
	for i := range parts {
		parts[i] = types.TypeString(targs.At(i), func(p *types.Package) string { return p.Name() })
	}
	return strings.Join(parts, ",")
}

// sanitizeIdent replaces every non-identifier character with '_', mirroring
// internal/genlib.sanitizeFieldName. It is kept local to cmd/godi-gen (the genlib
// helper is unexported) so the generator can pre-sanitize closed-generic names
// before they reach the emit layer (D-4): unkeyed emit uses the raw Name verbatim.
func sanitizeIdent(s string) string {
	var b strings.Builder
	for _, c := range s {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' {
			b.WriteRune(c)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// constructorDeps reads a constructor's parameters left to right and maps each
// to the service Name it injects (AD-8). A parameter type not matching any
// registered service falls back to its FULLY-QUALIFIED type key (not its short
// name), so plan.Validate reports it as a missing dependency by an unambiguous
// name. Falling back to the short name risked a false match: an unregistered
// pkgB.Config would collide with a registered service named "Config"
// (pkgA.Config) in Validate's by-name lookup and silently inject the wrong
// instance instead of reporting the missing dependency.
func constructorDeps(sig *types.Signature, depIndex map[string]string, imports *importSet) []string {
	params := sig.Params()
	if params.Len() == 0 {
		return nil
	}
	deps := make([]string, 0, params.Len())
	for i := 0; i < params.Len(); i++ {
		pt := params.At(i).Type()
		if sliceType, ok := pt.(*types.Slice); ok {
			elemType := sliceType.Elem()
			if _, ok := depIndex[typeKey(elemType)]; ok {
				goType := renderType(elemType, imports)
				deps = append(deps, "[]"+goType)
				continue
			}
		}
		if name, ok := depIndex[typeKey(pt)]; ok {
			deps = append(deps, name)
			continue
		}
		// Unresolved: use the fully-qualified key so Validate cannot confuse it
		// with a same-short-named registered service.
		deps = append(deps, typeKey(pt))
	}
	return deps
}

// validateCtorResults checks that a New{Impl} constructor returns exactly one
// value, or a value plus a trailing builtin error — the only two shapes the emit
// step can wire (AD-8). Any other arity/shape would emit a single/comma-err
// assignment that mismatches the real return count and fails to compile, so it
// is a gen-time error rather than a broken container.
func validateCtorResults(sig *types.Signature) error {
	res := sig.Results()
	switch res.Len() {
	case 1:
		return nil
	case 2:
		if !isErrorType(res.At(1).Type()) {
			return fmt.Errorf("returns (%s, %s); a two-result constructor's second "+
				"result must be error (allowed shapes: (T) or (T, error))",
				shortTypeName(res.At(0).Type()), shortTypeName(res.At(1).Type()))
		}
		return nil
	default:
		return fmt.Errorf("returns %d values; a constructor must return (T) or (T, error)",
			res.Len())
	}
}

// hasTrailingError reports whether the constructor's last result is the builtin
// error type — the AD-8 trailing-error convention that drives CtorErr.
func hasTrailingError(sig *types.Signature) bool {
	res := sig.Results()
	if res.Len() == 0 {
		return false
	}
	return isErrorType(res.At(res.Len() - 1).Type())
}

// isErrorType reports whether t is the universe `error` interface (a named type
// whose object lives in no package — the universe scope).
func isErrorType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	return named.Obj().Name() == "error" && named.Obj().Pkg() == nil
}

// implementsCloser reports whether the implementation (or a pointer to it)
// satisfies io.Closer, derived statically (AD-11). Story 2.1 only records this on
// plan.Service.Disposable; the LIFO disposer that consumes it is Story 2.3.
func implementsCloser(impl types.Type) bool {
	closer := closerInterface()
	if types.Implements(impl, closer) {
		return true
	}
	return types.Implements(types.NewPointer(impl), closer)
}

// closerInterface builds the io.Closer interface — interface{ Close() error } —
// without loading the io package, so the check works on any loaded program.
func closerInterface() *types.Interface {
	errorType := types.Universe.Lookup("error").Type()
	results := types.NewTuple(types.NewVar(token.NoPos, nil, "", errorType))
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), results, false)
	closeMethod := types.NewFunc(token.NoPos, nil, "Close", sig)
	return types.NewInterfaceType([]*types.Func{closeMethod}, nil).Complete()
}

// implementsAsyncDisposable reports whether the implementation (or a pointer to
// it) satisfies godi.AsyncDisposable — CloseAsync(context.Context) error —
// derived statically (Story 6.3, D-7). iface is the real interface type read
// from the godi package by asyncDisposableInterface; a nil iface (godi not among
// the loaded imports) yields false. A both-disposable satisfies BOTH this and
// implementsCloser — the caller records both flags and genlib's D-8 dispatch
// keeps such a service on the unchanged sync Track path.
func implementsAsyncDisposable(impl types.Type, iface *types.Interface) bool {
	if impl == nil || iface == nil {
		return false
	}
	if types.Implements(impl, iface) {
		return true
	}
	return types.Implements(types.NewPointer(impl), iface)
}

// asyncDisposableInterface returns the godi.AsyncDisposable interface type from
// the loaded godi package (D-7 — read the real type rather than synthesize a
// context.Context signature). The generator loads dependencies with
// NeedDeps|NeedTypes, so the godi package's types are available via pkg.Imports.
// Returns nil if godi is not among the direct imports.
func asyncDisposableInterface(pkg *packages.Package) *types.Interface {
	godiPkg := pkg.Imports[godiPkgPath]
	if godiPkg == nil || godiPkg.Types == nil {
		return nil
	}
	obj := godiPkg.Types.Scope().Lookup("AsyncDisposable")
	if obj == nil {
		return nil
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil
	}
	iface, _ := named.Underlying().(*types.Interface)
	return iface
}

// renderType renders t as emit-ready Go source, package-qualified by the
// identifier the emitted file imports each package under (normally its name;
// an assigned alias when two service packages share a name), and records each
// referenced package in imports as a side effect. types.TypeString handles
// pointers, qualified names, and closed generic instances correctly (deferred
// 1.1 robustness).
func renderType(t types.Type, imports *importSet) string {
	return types.TypeString(t, imports.aliasOf)
}

// typeKey is a fully-package-path-qualified rendering used only as a map key for
// dependency matching: identical types render to one string, so a constructor
// parameter type matches the service type that produced it.
func typeKey(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Path() })
}

// shortTypeName is the short declared name of a named type, else its string
// form. (spike_test.go has its own throwaway typeName; this production helper is
// named distinctly to avoid colliding with that Story 1.1 spike file.)
func shortTypeName(t types.Type) string {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	return t.String()
}

// renderExpr renders an AST expression back to source text — the form-B factory
// expression the generator inlines (D-2). format.Node reproduces the written
// expression from the AST without needing the original file bytes.
func renderExpr(fset *token.FileSet, expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	var b strings.Builder
	if err := format.Node(&b, fset, expr); err != nil {
		return ""
	}
	return b.String()
}

// textEdit is one span replacement inside a factory expression's source range.
type textEdit struct {
	pos, end token.Pos
	text     string
}

// rewriteFactoryExpr renders a form-B factory expression as it must read inside
// the emitted container package:
//
//  1. helper type parameters (tparams/targs non-nil — the marker was forwarded
//     from a per-service helper) are replaced with the closed type's rendering
//     (NewMonitor[T] → NewMonitor[*options.MyCfg]);
//  2. package-qualifier identifiers are remapped to the identifier the emitted
//     file imports that package under (a registration file's `optspkg.X` → the
//     assigned alias; an aliased godi import → godi), and the import recorded;
//  3. when the container is emitted into a DIFFERENT package than the
//     registration (samePkg=false), bare identifiers referring to the analyzed
//     package's own top-level declarations gain its qualifier — the gap that
//     made the verbatim inline non-compilable (deferred-work.md "form-B codegen
//     完整支持依赖型/跨包工厂" qualification half).
//
// With no applicable edits the format.Node rendering renderExpr produced before
// this extension is returned unchanged, keeping prior output byte-identical
// (AD-14 regen determinism). With edits, the ORIGINAL source span is spliced —
// genlib's final format.Source normalizes whitespace.
func rewriteFactoryExpr(pkg *packages.Package, expr ast.Expr, tparams *types.TypeParamList, targs *types.TypeList, imports *importSet, samePkg bool) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("factory argument expression is missing")
	}

	// Identifiers that must never be qualified or rewritten: a selector's Sel
	// (method/field access — x.Foo), and a composite literal's field keys
	// (Struct{Field: v}).
	skip := make(map[*ast.Ident]struct{})
	ast.Inspect(expr, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.SelectorExpr:
			skip[e.Sel] = struct{}{}
		case *ast.CompositeLit:
			for _, elt := range e.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					if key, ok := kv.Key.(*ast.Ident); ok {
						skip[key] = struct{}{}
					}
				}
			}
		}
		return true
	})

	var edits []textEdit
	var editErr error
	ast.Inspect(expr, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok || editErr != nil {
			return editErr == nil
		}
		if _, skipped := skip[id]; skipped {
			return true
		}
		obj := pkg.TypesInfo.Uses[id]
		if obj == nil {
			return true
		}

		// (1) helper type parameter → closed type text.
		if tn, isType := obj.(*types.TypeName); isType && tparams != nil {
			if tp, isParam := tn.Type().(*types.TypeParam); isParam {
				for i := 0; i < tparams.Len(); i++ {
					if tparams.At(i).Obj() != tp.Obj() {
						continue
					}
					closed := targs.At(i)
					edits = append(edits, textEdit{pos: id.Pos(), end: id.End(), text: renderType(closed, imports)})
					return true
				}
				editErr = fmt.Errorf("factory expression mentions type parameter %s not bound by the helper call", id.Name)
				return false
			}
		}

		// (2) package qualifier → the emitted file's identifier for that package.
		if pn, isPkg := obj.(*types.PkgName); isPkg {
			alias := imports.aliasOf(pn.Imported())
			if alias != id.Name {
				edits = append(edits, textEdit{pos: id.Pos(), end: id.End(), text: alias})
			}
			return true
		}

		// (3) cross-package emit: qualify the registration package's own
		// top-level identifiers.
		if !samePkg && obj.Pkg() == pkg.Types && obj.Parent() == pkg.Types.Scope() {
			edits = append(edits, textEdit{pos: id.Pos(), end: id.End(), text: imports.aliasOf(pkg.Types) + "." + id.Name})
		}
		return true
	})
	if editErr != nil {
		return "", editErr
	}
	if len(edits) == 0 {
		return renderExpr(pkg.Fset, expr), nil
	}

	src, err := exprSourceSpan(pkg, expr)
	if err != nil {
		return "", err
	}
	// Apply right-to-left so earlier offsets stay valid.
	sort.Slice(edits, func(i, j int) bool { return edits[i].pos > edits[j].pos })
	base := pkg.Fset.Position(expr.Pos()).Offset
	out := []byte(src)
	for _, e := range edits {
		lo := pkg.Fset.Position(e.pos).Offset - base
		hi := pkg.Fset.Position(e.end).Offset - base
		if lo < 0 || hi > len(out) || lo > hi {
			return "", fmt.Errorf("factory expression rewrite: edit span out of range")
		}
		out = append(out[:lo], append([]byte(e.text), out[hi:]...)...)
	}
	return string(out), nil
}

// exprSourceSpan returns the original source bytes of expr's span, read from
// the file that declares it. The generator holds ASTs, not file bytes, so the
// span is re-read here; go/packages already parsed the same bytes.
func exprSourceSpan(pkg *packages.Package, expr ast.Expr) (string, error) {
	pos := pkg.Fset.Position(expr.Pos())
	end := pkg.Fset.Position(expr.End())
	if pos.Filename == "" || pos.Filename != end.Filename {
		return "", fmt.Errorf("factory expression rewrite: cannot locate source file for %v", pos)
	}
	data, err := os.ReadFile(pos.Filename)
	if err != nil {
		return "", fmt.Errorf("factory expression rewrite: %w", err)
	}
	if end.Offset > len(data) || pos.Offset > end.Offset {
		return "", fmt.Errorf("factory expression rewrite: span exceeds %s", pos.Filename)
	}
	return string(data[pos.Offset:end.Offset]), nil
}

// factoryDeps reads a form-B factory expression for godi.Get / GetRequired /
// GetServices calls and maps each resolvable requested type (helper type
// parameters closed first) to the registered service Name(s) it denotes. The
// names feed plan.Service.Deps for Validate's static checks only; construction
// order needs no help, since every emitted accessor builds what the factory Gets
// on first access (AD-15; deferred-work.md "form-B codegen 完整支持依赖型/跨包工厂").
// The emit paths never render form-B Deps as arguments (the factory text
// carries its own resolution calls).
//
// A requested type that resolves to no registered service adds no constraint —
// GetServices over an unregistered element type legitimately yields an empty
// slice (the AddOptions pattern's empty configure chain). Keyed Get* variants
// are not scanned (keyed namespace, D-3).
func factoryDeps(pkg *packages.Package, expr ast.Expr, tparams *types.TypeParamList, targs *types.TypeList, depIndex map[string]string, elemIndex map[string][]string) []string {
	if expr == nil {
		return nil
	}
	var deps []string
	seen := make(map[string]struct{})
	add := func(name string) {
		if _, dup := seen[name]; dup {
			return
		}
		seen[name] = struct{}{}
		deps = append(deps, name)
	}

	ast.Inspect(expr, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident := genericCallee(call)
		if ident == nil {
			return true
		}
		fn, ok := pkg.TypesInfo.Uses[ident].(*types.Func)
		if !ok || fn.Pkg() == nil || fn.Pkg().Path() != godiPkgPath {
			return true
		}
		inst, ok := pkg.TypesInfo.Instances[ident]
		if !ok || inst.TypeArgs == nil || inst.TypeArgs.Len() == 0 {
			return true
		}
		requested, err := substituteTypeParams(inst.TypeArgs.At(0), tparams, targs)
		if err != nil {
			return true // unhandled shape: no ordering constraint, factory still inlines
		}
		switch fn.Name() {
		case "Get", "GetRequired":
			if name, ok := depIndex[typeKey(requested)]; ok {
				add(name)
			}
		case "GetServices":
			for _, name := range elemIndex[typeKey(requested)] {
				add(name)
			}
		}
		return true
	})
	return deps
}

// importSet collects service-package imports in first-seen order, deduped, and
// assigns each package the identifier the emitted file refers to it by. The
// identifier is normally the package's declared name; when two packages share a
// name (gois's options and config/options are BOTH package options) the later
// one gets a numbered alias (options2) — without this the emitted file would
// import two same-named packages and fail to compile. Names genlib's fixed
// imports already bind (reflect, godi, io, context, runtime) are reserved.
// genlib's emitImports sorts and re-dedupes (and always adds reflect + godi),
// so only the user service packages need to flow through here.
type importSet struct {
	localPath string
	aliases   map[string]string // package path → emitted identifier
	taken     map[string]string // emitted identifier → package path (or "" for reserved)
	order     []importEntry
}

type importEntry struct {
	path  string
	name  string // the package's declared name
	alias string // the identifier the emitted file binds — == name when no clash
}

func newImportSet(localPath string) *importSet {
	s := &importSet{
		localPath: localPath,
		aliases:   make(map[string]string),
		taken:     make(map[string]string),
	}
	// Seed each name genlib's own imports may bind with the path that
	// legitimately owns it: that exact path keeps its name (a factory body's
	// reflect.TypeFor stays reflect — emitImports dedups against the fixed
	// block), while ANOTHER package with the same declared name is renamed.
	for name, owner := range map[string]string{
		"reflect": "reflect",
		"godi":    godiPkgPath,
		"io":      "io",
		"context": "context",
		"runtime": godiPkgPath + "/runtime",
	} {
		s.taken[name] = owner
	}
	return s
}

// aliasOf returns the identifier the emitted file refers to p by, assigning and
// recording the import on first sight. The godi host package is always "godi"
// (genlib imports it unconditionally; a registration file's own alias for godi
// is remapped here).
func (s *importSet) aliasOf(p *types.Package) string {
	if p == nil {
		return ""
	}
	path := p.Path()
	if path == godiPkgPath {
		return "godi"
	}
	if path == s.localPath {
		return ""
	}
	if path == "" {
		return p.Name()
	}
	if a, ok := s.aliases[path]; ok {
		return a
	}
	alias := p.Name()
	for i := 2; ; i++ {
		owner, clash := s.taken[alias]
		if !clash || owner == path {
			break
		}
		alias = fmt.Sprintf("%s%d", p.Name(), i)
	}
	s.aliases[path] = alias
	s.taken[alias] = path
	s.order = append(s.order, importEntry{path: path, name: p.Name(), alias: alias})
	return alias
}

func (s *importSet) addPackage(p *types.Package) { _ = s.aliasOf(p) }

// list renders the entries for plan.Registry.Imports: a plain path when the
// package's declared name needs no rename, "alias path" (space-separated) when
// the emitted file must bind it under an alias — genlib's emitImports splits on
// the space and emits `alias "path"`.
func (s *importSet) list() []string {
	out := make([]string, 0, len(s.order))
	for _, e := range s.order {
		if e.alias == e.name {
			out = append(out, e.path)
			continue
		}
		out = append(out, e.alias+" "+e.path)
	}
	return out
}
