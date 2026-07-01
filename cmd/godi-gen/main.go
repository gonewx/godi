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

	src, err := generate(args[0], *pkgName)
	if err != nil {
		// AD-3 / NFR2: a bad graph (missing New{Impl}, validation failure, load
		// error) produces ZERO bytes and a non-zero exit. The //go:generate gate
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
// container as package pkgName. On any error it returns nil output (NFR2: a
// rejected graph emits zero bytes).
func generate(srcDir, pkgName string) ([]byte, error) {
	pkg, err := loadPackage(srcDir)
	if err != nil {
		return nil, err
	}
	reg, err := buildRegistry(pkg)
	if err != nil {
		return nil, err
	}
	return genlib.Generate(reg, pkgName)
}

// loadPackage statically loads the single registration package at dir with full
// type information. Type checking MUST succeed: Info.Instances is populated by
// the type checker, so any load/type error would leave the markers unreadable.
func loadPackage(dir string) (*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports |
			packages.NeedDeps,
		Dir: dir,
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
	iType    types.Type    // service type (first type argument)
	implType types.Type    // form A only: implementation type (second type argument)
	factory  ast.Expr      // form B only: the factory argument expression
	keyed    bool          // true for AddKeyed* markers
	keyExpr  ast.Expr      // keyed only: the key argument expression
	keyText  string        // keyed only: rendered key expression source text
}

// buildRegistry turns the package's DSL markers into a validated-input
// plan.Registry. It maps each form-A constructor's dependency parameters back to
// registered service names, derives CtorErr / Disposable from go/types, and
// inlines form-B factory expressions (D-2). A form-A service with no New{Impl}
// aggregates into a generation error (AC2).
func buildRegistry(pkg *packages.Package) (*plan.Registry, error) {
	regs := collectRegistrations(pkg)

	// NFR2: a package with zero recognized DSL markers must be rejected — emitting
	// an empty container (exit 0) would silently hand the user a container missing
	// every service. The common cause (markers written in an unrecognized form) is
	// surfaced separately as collectRegistrations stderr warnings.
	if len(regs) == 0 {
		return nil, fmt.Errorf("no godi DSL registrations recognized in package %q: "+
			"expected at least one godi.Add{Singleton,Scoped,Transient}[I, Impl] or "+
			"godi.Add{Singleton,Scoped,Transient}Factory[I] marker imported from %q "+
			"with explicit type arguments", pkg.PkgPath, godiPkgPath)
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

	// Pass 1: index every service's type(s) -> its service Name, so a
	// constructor parameter type can be resolved to the dependency it names. Both
	// the service interface (I) and the implementation (Impl, and *Impl) are
	// indexed so a constructor may inject either form. Keyed services are NOT
	// indexed here — keyed and unkeyed are independent namespaces (D-3).
	depIndex := make(map[string]string, len(regs)*2)
	for _, rg := range regs {
		if rg.keyed {
			continue
		}
		name := serviceName(rg)
		depIndex[typeKey(rg.iType)] = name
		if rg.form == "A" && rg.implType != nil {
			depIndex[typeKey(rg.implType)] = name
			depIndex[typeKey(types.NewPointer(rg.implType))] = name
		}
	}

	reg := plan.NewRegistry()
	imports := newImportSet()
	var genErrs []string

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
			// Form B names the service after its interface's short name; a non-named
			// type argument (*Foo, []Foo) would yield an illegal field name like
			// f_*pkg.Foo and a non-compilable container. Require a valid identifier.
			// For keyed services, the Name contains "#key" suffix which is sanitized
			// later — check only the base part before the "#".
			baseName := s.Name
			if idx := strings.Index(baseName, "#"); idx >= 0 {
				baseName = baseName[:idx]
			}
			if !token.IsIdentifier(baseName) {
				genErrs = append(genErrs, fmt.Sprintf(
					"form-B service: type argument %s is not a named type — "+
						"Add{Lifetime}Factory[I] requires I to be a named interface "+
						"(would emit illegal field name f_%s)", s.GoType, s.Name))
				continue
			}
			// Form B escape hatch (D-2): inline the factory expression verbatim;
			// no New{Impl} discovery, no dependency reading.
			s.FactoryExpr = renderExpr(pkg.Fset, rg.factory)
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

		s.Ctor = implPkg.Name() + ".New" + implName
		if typeArgs := impl.TypeArgs(); typeArgs != nil && typeArgs.Len() > 0 {
			args := make([]string, typeArgs.Len())
			for i := range args {
				args[i] = renderType(typeArgs.At(i), imports)
			}
			s.Ctor += "[" + strings.Join(args, ", ") + "]"
		}
		imports.addPackage(implPkg)
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
func collectRegistrations(pkg *packages.Package) []registration {
	var out []registration

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			// A call whose bare callee name is a godi marker is a SUSPECTED marker.
			// It is accepted only when it is a generic instantiation that resolves to
			// the godi package. If it looks like a marker but fails any check (type
			// inference dropped [I], a dot-import, or a vanity/forked import path),
			// warn instead of silently dropping it (NFR2 no silent drop).
			suspected := isDSLName(calleeName(call))

			// genericCallee handles BOTH a godi selector (godi.AddSingleton[…]) and a
			// bare-ident per-service helper call (AddRepository[User]) — the latter is
			// what dslCallee (selector-only) used to miss.
			ident := genericCallee(call)
			if ident == nil {
				warnUnrecognized(pkg, call, suspected,
					"not a recognized generic marker call (a type argument [I] may have "+
						"been inferred away, or godi is dot-imported)")
				return true
			}
			fn, ok := pkg.TypesInfo.Uses[ident].(*types.Func)
			if !ok || fn.Pkg() == nil {
				warnUnrecognized(pkg, call, suspected,
					"callee does not resolve to a package-level function")
				return true
			}

			// Direct godi marker (the primary path). A marker whose type arguments
			// still mention a type parameter — the TEMPLATE inside a per-service
			// helper body, godi.AddTransient[IFoo[T], Foo[T]] — is skipped: only the
			// helper instantiation that forwards to it is a concrete registration (AC1).
			if fn.Pkg().Path() == godiPkgPath {
				out = appendGodiMarker(out, pkg, ident, fn, call)
				return true
			}

			// Per-service open-generic helper (Mechanism A, AC1): a generic function
			// DEFINED IN THE ANALYZED PACKAGE whose body forwards to a single godi
			// marker. readPerServiceHelper follows the forwarder and substitutes the
			// helper's type parameters with this call's type arguments. handled=true
			// suppresses the suspected-name warn; an ordinary non-godi generic call
			// (reflect.TypeFor, …) returns handled=false and falls through.
			if regs, handled := readPerServiceHelper(pkg, call, ident, fn); handled {
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
		return pi.Offset < pj.Offset
	})
	return out
}

// appendGodiMarker reads a direct godi.Add… marker call and appends the
// registration it denotes. A marker whose type arguments still mention a type
// parameter (the TEMPLATE marker inside a per-service helper body) is SKIPPED —
// it is not a concrete registration; only the helper call that forwards to it is
// (Mechanism A skip rule, AC1). The unified rule: the generator only processes
// registrations whose type arguments are fully concrete.
func appendGodiMarker(out []registration, pkg *packages.Package, ident *ast.Ident, fn *types.Func, call *ast.CallExpr) []registration {
	inst, ok := pkg.TypesInfo.Instances[ident]
	if !ok {
		warnUnrecognized(pkg, call, isDSLName(fn.Name()),
			"no generic instantiation recorded by the type checker")
		return out
	}
	ta := inst.TypeArgs
	if typeArgsHaveParam(ta) {
		return out
	}
	return appendMarkerRegistration(out, pkg, fn.Name(), call.Pos(), ta, call.Args)
}

// appendMarkerRegistration builds the registration for a godi marker named
// markerName with type arguments ta and call arguments args. It is the shared
// reduction target for a direct concrete marker AND a per-service-helper-reduced
// closed marker (Mechanism A): both arrive here as a fully-concrete (markerName,
// ta) pair, so the two paths produce identical registrations (AD-14).
func appendMarkerRegistration(out []registration, pkg *packages.Package, markerName string, pos token.Pos, ta *types.TypeList, args []ast.Expr) []registration {
	switch markerName {
	case "AddSingleton", "AddScoped", "AddTransient":
		if ta.Len() != 2 {
			return out
		}
		return append(out, registration{
			form: "A", lifetime: lifetimeFromName(markerName), pos: pos,
			iType: ta.At(0), implType: ta.At(1),
		})
	case "AddSingletonFactory", "AddScopedFactory", "AddTransientFactory":
		if ta.Len() != 1 {
			return out
		}
		var factory ast.Expr
		if len(args) >= 2 {
			factory = args[1]
		}
		return append(out, registration{
			form: "B", lifetime: lifetimeFromName(markerName), pos: pos,
			iType: ta.At(0), factory: factory,
		})
	case "AddKeyedSingleton", "AddKeyedScoped", "AddKeyedTransient":
		if ta.Len() != 2 {
			return out
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
		})
	case "AddKeyedSingletonFactory", "AddKeyedScopedFactory", "AddKeyedTransientFactory":
		if ta.Len() != 1 {
			return out
		}
		var keyArg ast.Expr
		var keyTxt string
		if len(args) >= 2 {
			keyArg = args[1]
			keyTxt = renderExpr(pkg.Fset, keyArg)
		}
		var factory ast.Expr
		if len(args) >= 3 {
			factory = args[2]
		}
		return append(out, registration{
			form: "B", lifetime: lifetimeFromName(markerName), pos: pos,
			iType: ta.At(0), factory: factory,
			keyed: true, keyExpr: keyArg, keyText: keyTxt,
		})
	}
	return out
}

// genericCallee extracts the function-name identifier of a generic instantiation
// call, handling BOTH a package selector (godi.AddSingleton[…]) and a bare
// identifier (a same-package per-service helper AddRepository[User]). It returns
// nil when the call is not a generic instantiation of a function. The bare-ident
// form is what distinguishes a local helper call from a godi marker selector.
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

// readPerServiceHelper recognizes a per-service open-generic helper call
// (Mechanism A, AC1) and reduces it to the closed registration it denotes.
//
// A per-service helper is a generic function DEFINED IN THE ANALYZED PACKAGE whose
// body is a single `return godi.Add{Lifetime}[I[T…], Impl[T…]](r)` forwarder. The
// generator follows the forwarder, reads the OUTER call's type arguments
// (AddRepository[User] → [User]), and substitutes the helper's type parameters in
// the inner marker's type arguments via types.Instantiate to materialize the
// closed (IRepository[User], Repository[User]) registration — which then flows
// through the SAME form-A pipeline as a hand-written closed marker (AD-14).
//
// handled=true means the call is (or clearly intends to be) a helper, so the
// caller does not additionally warn; handled=false means it is an ordinary
// non-godi generic call (reflect.TypeFor, …) the caller treats by its
// suspected-name rule.
//
// Story 4.1 scope: helpers in the analyzed package, forwarding to a single UNKEYED
// form-A marker, single type parameter. Cross-package helpers, keyed/factory
// forwarders, and multi/permuted type parameters are deferred — warned, never
// silently dropped (NFR2) — and recorded in deferred-work.md.
func readPerServiceHelper(pkg *packages.Package, call *ast.CallExpr, ident *ast.Ident, fn *types.Func) (regs []registration, handled bool) {
	decl := findFuncDecl(pkg, fn)
	if decl == nil || decl.Body == nil {
		return nil, false // not defined in the analyzed package (e.g. reflect.TypeFor)
	}
	innerIdent, innerFn := helperForwardMarker(pkg, decl)
	if innerFn == nil {
		return nil, false // body is not a single-statement godi forwarder — ordinary func
	}

	// From here the call is INTENDED as a per-service helper; any shortfall is a
	// warned defer (handled=true), never a silent drop (NFR2).
	outerInst, ok := pkg.TypesInfo.Instances[ident]
	if !ok || outerInst.TypeArgs == nil || outerInst.TypeArgs.Len() == 0 {
		warnUnrecognized(pkg, call, true, "per-service helper call has no recorded type arguments")
		return nil, true
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.TypeParams() == nil || sig.TypeParams().Len() == 0 {
		warnUnrecognized(pkg, call, true, "per-service helper is not generic")
		return nil, true
	}
	tparams := sig.TypeParams()
	outerArgs := outerInst.TypeArgs
	if tparams.Len() != outerArgs.Len() {
		warnUnrecognized(pkg, call, true,
			"per-service helper type-parameter / type-argument count mismatch")
		return nil, true
	}
	if tparams.Len() > 1 {
		warnUnrecognized(pkg, call, true, fmt.Sprintf(
			"per-service helper has %d type parameters — multi-type-parameter "+
				"helpers are deferred (only single type parameter is supported)",
			tparams.Len()))
		return nil, true
	}

	markerName := innerFn.Name()
	switch markerName {
	case "AddSingleton", "AddScoped", "AddTransient":
		// supported (Story 4.1): unkeyed form-A
	default:
		warnUnrecognized(pkg, call, true, fmt.Sprintf(
			"per-service helper forwards to %s — Story 4.1 supports only unkeyed form-A "+
				"Add{Singleton,Scoped,Transient}[I, Impl] helpers (keyed / factory / "+
				"constrained forwarders are deferred)", markerName))
		return nil, true
	}

	innerInst, ok := pkg.TypesInfo.Instances[innerIdent]
	if !ok || innerInst.TypeArgs == nil || innerInst.TypeArgs.Len() != 2 {
		warnUnrecognized(pkg, call, true,
			"per-service helper's form-A marker must have [I, Impl] type arguments")
		return nil, true
	}
	innerArgs := innerInst.TypeArgs

	iClosed, err1 := substituteTypeParams(innerArgs.At(0), tparams, outerArgs)
	implClosed, err2 := substituteTypeParams(innerArgs.At(1), tparams, outerArgs)
	if err1 != nil || err2 != nil {
		warnUnrecognized(pkg, call, true, fmt.Sprintf(
			"per-service helper type substitution failed: %v / %v", err1, err2))
		return nil, true
	}

	return []registration{{
		form: "A", lifetime: lifetimeFromName(markerName), pos: call.Pos(),
		iType: iClosed, implType: implClosed,
	}}, true
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

// helperForwardMarker extracts the single godi marker a helper body forwards to:
// the body must be exactly one `return <call>` statement whose call resolves to a
// godi.Add… marker. It returns the inner callee ident and resolved marker
// *types.Func, or (nil, nil) when the body is not a single-statement godi
// forwarder (the signal that the function is an ordinary generic function, not a
// per-service helper).
func helperForwardMarker(pkg *packages.Package, decl *ast.FuncDecl) (*ast.Ident, *types.Func) {
	if decl.Body == nil || len(decl.Body.List) != 1 {
		return nil, nil
	}
	ret, ok := decl.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return nil, nil
	}
	result := ret.Results[0]
	if paren, ok := result.(*ast.ParenExpr); ok {
		result = paren.X
	}
	inner, ok := result.(*ast.CallExpr)
	if !ok {
		return nil, nil
	}
	innerIdent := genericCallee(inner)
	if innerIdent == nil {
		return nil, nil
	}
	innerFn, ok := pkg.TypesInfo.Uses[innerIdent].(*types.Func)
	if !ok || innerFn.Pkg() == nil || innerFn.Pkg().Path() != godiPkgPath {
		return nil, nil
	}
	return innerIdent, innerFn
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
// function name. Used to flag a SUSPECTED DSL marker even when genericCallee
// declines it, so warnUnrecognized can report it instead of dropping it silently.
func calleeName(call *ast.CallExpr) string {
	fun := call.Fun
	switch f := fun.(type) {
	case *ast.IndexExpr:
		fun = f.X
	case *ast.IndexListExpr:
		fun = f.X
	}
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		return f.Sel.Name
	case *ast.Ident:
		return f.Name
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
// like a godi DSL marker was not recognized as one (NFR2's "no silent drop").
// It is a warning, not an error: a same-named call from another package is a
// legitimate non-match. Static analysis only — no user code runs (AD-7).
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
		return shortTypeName(rg.iType)
	}
	base := named.Obj().Name()
	if targs := named.TypeArgs(); targs != nil && targs.Len() > 0 {
		base += "_" + sanitizeIdent(typeArgsString(targs))
	}
	return base
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

// renderType renders t as emit-ready Go source, package-qualified by package
// name (fakes.IFakeService, *svc.Dep), and records each referenced package in
// imports as a side effect. types.TypeString handles pointers, qualified names,
// and closed generic instances correctly (deferred 1.1 robustness).
func renderType(t types.Type, imports *importSet) string {
	return types.TypeString(t, func(p *types.Package) string {
		imports.addPackage(p)
		return p.Name()
	})
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

// importSet collects service-package import paths in first-seen order, deduped.
// genlib's emitImports sorts and re-dedupes (and always adds reflect + godi), so
// only the user service packages need to flow through here.
type importSet struct {
	seen  map[string]struct{}
	order []string
}

func newImportSet() *importSet {
	return &importSet{seen: make(map[string]struct{})}
}

func (s *importSet) addPackage(p *types.Package) {
	if p == nil {
		return
	}
	path := p.Path()
	if path == "" || path == godiPkgPath {
		return
	}
	if _, ok := s.seen[path]; ok {
		return
	}
	s.seen[path] = struct{}{}
	s.order = append(s.order, path)
}

func (s *importSet) list() []string { return s.order }
