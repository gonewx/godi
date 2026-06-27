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
	form     string     // "A" (AddSingleton[I,Impl]) or "B" (AddSingletonFactory[I])
	pos      token.Pos  // marker position — sorted on for determinism (deferred 1.1)
	iType    types.Type // service type (first type argument)
	implType types.Type // form A only: implementation type (second type argument)
	factory  ast.Expr   // form B only: the factory argument expression
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
			"expected at least one godi.AddSingleton[I, Impl] or godi.AddSingletonFactory[I] "+
			"marker imported from %q with explicit type arguments", pkg.PkgPath, godiPkgPath)
	}

	// Pass 1: index every service's type(s) -> its service Name, so a
	// constructor parameter type can be resolved to the dependency it names. Both
	// the service interface (I) and the implementation (Impl, and *Impl) are
	// indexed so a constructor may inject either form.
	depIndex := make(map[string]string, len(regs)*2)
	for _, rg := range regs {
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

	for _, rg := range regs {
		s := plan.Service{
			Name:     serviceName(rg),
			GoType:   renderType(rg.iType, imports),
			Lifetime: plan.Singleton,
		}

		if rg.form == "B" {
			// Form B names the service after its interface's short name; a non-named
			// type argument (*Foo, []Foo) would yield an illegal field name like
			// f_*pkg.Foo and a non-compilable container. Require a valid identifier.
			if !token.IsIdentifier(s.Name) {
				genErrs = append(genErrs, fmt.Sprintf(
					"form-B service: type argument %s is not a named type — "+
						"AddSingletonFactory[I] requires I to be a named interface "+
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
		imports.addPackage(implPkg)
		s.Deps = constructorDeps(sig, depIndex)
		s.CtorErr = hasTrailingError(sig)
		s.Disposable = implementsCloser(rg.implType)

		reg.Service(s)
	}

	if len(genErrs) > 0 {
		return nil, fmt.Errorf("generation rejected (%d error(s)):\n%s",
			len(genErrs), strings.Join(genErrs, "\n"))
	}

	reg.Imports = imports.list()
	return reg, nil
}

// collectRegistrations walks the package AST for godi.AddSingleton /
// AddSingletonFactory marker calls, reads their type arguments from
// Info.Instances, and returns them sorted by source position. Sorting makes the
// registry order deterministic regardless of the Info.Instances map iteration
// order (deferred 1.1).
func collectRegistrations(pkg *packages.Package) []registration {
	var out []registration

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			// A call whose bare callee name is AddSingleton / AddSingletonFactory is
			// a SUSPECTED marker. It is accepted only when it is a generic
			// instantiation of a selector that resolves to the godi package. If it
			// looks like a marker but fails any check (type inference dropped [I], a
			// dot-import, or a vanity/forked import path), warn instead of silently
			// dropping it — a dropped registration yields an incomplete container.
			suspected := isDSLName(calleeName(call))

			funcIdent := dslCallee(call)
			if funcIdent == nil {
				warnUnrecognized(pkg, call, suspected,
					"not a recognized generic marker call (a type argument [I] may have "+
						"been inferred away, or godi is dot-imported)")
				return true
			}
			// Confirm the callee is the godi package's marker, not a same-named
			// function elsewhere.
			fn, ok := pkg.TypesInfo.Uses[funcIdent].(*types.Func)
			if !ok || fn.Pkg() == nil || fn.Pkg().Path() != godiPkgPath {
				warnUnrecognized(pkg, call, suspected, fmt.Sprintf(
					"callee does not resolve to the godi package %q (vanity import, fork, or replace?)",
					godiPkgPath))
				return true
			}
			inst, ok := pkg.TypesInfo.Instances[funcIdent]
			if !ok {
				warnUnrecognized(pkg, call, suspected,
					"no generic instantiation recorded by the type checker")
				return true
			}
			ta := inst.TypeArgs

			switch fn.Name() {
			case "AddSingleton":
				if ta.Len() != 2 {
					return true
				}
				out = append(out, registration{
					form: "A", pos: call.Pos(),
					iType: ta.At(0), implType: ta.At(1),
				})
			case "AddSingletonFactory":
				if ta.Len() != 1 {
					return true
				}
				var factory ast.Expr
				if len(call.Args) >= 2 {
					factory = call.Args[1]
				}
				out = append(out, registration{
					form: "B", pos: call.Pos(),
					iType: ta.At(0), factory: factory,
				})
			}
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

// dslCallee extracts the function-name identifier of a generic DSL call, or nil
// if the call is not a generic instantiation of a selector. AddSingleton[I,Impl]
// is an IndexListExpr (two type args); AddSingletonFactory[I] is an IndexExpr
// (one type arg). Both wrap a SelectorExpr (godi.AddSingleton…); its .Sel is the
// ident keyed in Info.Instances.
func dslCallee(call *ast.CallExpr) *ast.Ident {
	switch fun := call.Fun.(type) {
	case *ast.IndexExpr:
		if sel, ok := fun.X.(*ast.SelectorExpr); ok {
			return sel.Sel
		}
	case *ast.IndexListExpr:
		if sel, ok := fun.X.(*ast.SelectorExpr); ok {
			return sel.Sel
		}
	}
	return nil
}

// calleeName returns the bare function name a call invokes, unwrapping a generic
// instantiation (foo[T] / foo[T1, T2]) and a package selector (pkg.foo) or a bare
// identifier (a dot-imported foo). It is empty when the callee is not a plain
// function name. Used to flag a SUSPECTED DSL marker even when dslCallee declines
// it, so warnUnrecognized can report it instead of dropping it silently.
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
	return name == "AddSingleton" || name == "AddSingletonFactory"
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
func serviceName(rg registration) string {
	if rg.form == "A" && rg.implType != nil {
		if named, ok := rg.implType.(*types.Named); ok {
			return named.Obj().Name()
		}
	}
	return shortTypeName(rg.iType)
}

// constructorDeps reads a constructor's parameters left to right and maps each
// to the service Name it injects (AD-8). A parameter type not matching any
// registered service falls back to its FULLY-QUALIFIED type key (not its short
// name), so plan.Validate reports it as a missing dependency by an unambiguous
// name. Falling back to the short name risked a false match: an unregistered
// pkgB.Config would collide with a registered service named "Config"
// (pkgA.Config) in Validate's by-name lookup and silently inject the wrong
// instance instead of reporting the missing dependency.
func constructorDeps(sig *types.Signature, depIndex map[string]string) []string {
	params := sig.Params()
	if params.Len() == 0 {
		return nil
	}
	deps := make([]string, 0, params.Len())
	for i := 0; i < params.Len(); i++ {
		pt := params.At(i).Type()
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
