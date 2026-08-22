// ATDD test for Story 1.1 (AC2) — the AD-7 integration spike, the core
// de-risking gate (AR11). Implemented as a self-verifying test per the story
// Dev Notes: it proves the go/packages + go/types path can read a generic DSL
// call's type arguments WITHOUT executing user registration code, then emits a
// compilable minimal container to a temp dir.
//
// Activated in Story 1.1 Task 3 (main.go stub + testdata fixture +
// golang.org/x/tools (then v0.47.0) in go.mod; the original //go:build
// atdd_redphase red-phase guard was removed and emitMinimalContainer
// implemented).
//
// Regression risk R1: this test emits ONLY to t.TempDir() and is NOT wired to
// `//go:generate`, so the CI Generate gate stays a clean no-op.
package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"golang.org/x/tools/go/packages"
)

const spikeFixtureDir = "testdata/spike"

// TestADSpike_ReadsTypeArgumentsStatically covers AC2 / 1.1-INT-001 + 1.1-INT-002.
//
// Load the fixture statically and read the two type arguments of the
// godi.AddSingleton[ILogger, Logger] generic call from go/types, in
// declaration order: ILogger (interface), Logger (implementation).
func TestADSpike_ReadsTypeArgumentsStatically(t *testing.T) {
	pkg := loadSpikeFixture(t)

	var found bool
	for ident, inst := range pkg.TypesInfo.Instances {
		if ident.Name != "AddSingleton" {
			continue
		}
		found = true
		ta := inst.TypeArgs
		testutil.Equal(t, ta.Len(), 2)
		testutil.Equal(t, typeName(ta.At(0)), "ILogger")
		testutil.Equal(t, typeName(ta.At(1)), "Logger")
	}
	testutil.True(t, found,
		"AddSingleton generic instantiation must appear in TypesInfo.Instances")
}

// TestADSpike_DoesNotExecuteUserCode covers AC2 / 1.1-INT-003 — the AD-7
// invariant.
//
// Static analysis must NOT execute user registration code. The fixture's
// init() panics if ever run; reaching the end of loadSpikeFixture without a
// panic is positive, executable evidence that packages.Load only parsed and
// type-checked the DSL marker — it never invoked it.
func TestADSpike_DoesNotExecuteUserCode(t *testing.T) {
	pkg := loadSpikeFixture(t) // would panic via the fixture tripwire if executed
	testutil.NotNil(t, pkg.TypesInfo)
}

// TestADSpike_EmitsCompilableContainer covers AC2 / 1.1-INT-004.
//
// From the resolved type arguments, emit a minimal container and prove it is
// COMPILABLE: gofmt-legal (format.Source) and type-checkable (re-parse +
// types.Config.Check). Never `go build` a loose file — it has no go.mod.
func TestADSpike_EmitsCompilableContainer(t *testing.T) {
	pkg := loadSpikeFixture(t)
	args := addSingletonTypeArgs(t, pkg)

	src, err := emitMinimalContainer(args)
	testutil.NoError(t, err)

	// gofmt-legal: format.Source errors on syntactically invalid Go.
	formatted, err := format.Source(src)
	testutil.NoError(t, err)

	// Type-checkable: re-parse and run the type checker over the emitted file.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "container_gen.go", formatted, parser.AllErrors)
	testutil.NoError(t, err)

	conf := types.Config{Importer: importer.Default()}
	_, err = conf.Check("godigen", fset, []*ast.File{file}, nil)
	testutil.NoError(t, err)

	// Emit to a temp dir only (R1: never commit container_gen.go).
	out := filepath.Join(t.TempDir(), "container_gen.go")
	testutil.NoError(t, os.WriteFile(out, formatted, 0o644))
}

// --- helpers ----------------------------------------------------------------

// loadSpikeFixture statically loads the testdata fixture with full type info.
func loadSpikeFixture(t *testing.T) *packages.Package {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports |
			packages.NeedDeps,
		Dir: spikeFixtureDir,
	}
	pkgs, err := packages.Load(cfg, ".")
	testutil.NoError(t, err)
	testutil.Equal(t, len(pkgs), 1)
	// Type checking MUST succeed: Instances is populated by the type checker;
	// any load/type error leaves it empty and would falsely fail the spike.
	if packages.PrintErrors(pkgs) > 0 {
		t.Fatalf("fixture failed to load/type-check; see errors above")
	}
	return pkgs[0]
}

// addSingletonTypeArgs returns the type arguments of the AddSingleton call.
func addSingletonTypeArgs(t *testing.T, pkg *packages.Package) []types.Type {
	t.Helper()
	for ident, inst := range pkg.TypesInfo.Instances {
		if ident.Name == "AddSingleton" {
			ta := inst.TypeArgs
			out := make([]types.Type, 0, ta.Len())
			for i := 0; i < ta.Len(); i++ {
				out = append(out, ta.At(i))
			}
			return out
		}
	}
	t.Fatalf("AddSingleton instantiation not found")
	return nil
}

// typeName is the declared name of a named type, else its string form.
func typeName(t types.Type) string {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	return t.String()
}

// emitMinimalContainer renders a throwaway, compilable container from the
// resolved type arguments. THROWAWAY: proving the chain end-to-end is the goal,
// not a production generator (that is Story 1.4).
//
// The emitted source is a self-contained `package godigen` with NO imports, so
// types.Config.Check passes with the default importer. It declares the recovered
// service and implementation type names locally and stores the singleton in a
// Root field — mirroring AD-1, where lifetime is encoded by storage location
// (a Root field = a singleton), not a runtime flag. That the recovered names
// reach valid, type-checkable Go is the end-to-end proof the spike exists for.
func emitMinimalContainer(args []types.Type) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("emitMinimalContainer: want 2 type args [I, Impl], got %d", len(args))
	}
	iface := typeName(args[0])
	impl := typeName(args[1])

	src := fmt.Sprintf(`// Code generated by the godi-gen AD-7 spike; DO NOT EDIT.
//
// Throwaway proof (Story 1.1) that the static-analysis chain reached emission:
// the type names below were recovered from godi.AddSingleton's type arguments
// via go/types Info.Instances, never by executing user code (AD-7).
package godigen

// %[1]s is the service interface recovered from the first type argument.
type %[1]s interface{}

// %[2]s is the implementation recovered from the second type argument.
type %[2]s struct{}

// Container is the minimal generated container. Storing the singleton in a Root
// field encodes its lifetime (AD-1); no runtime flag distinguishes lifetimes.
type Container struct {
	f_%[1]s %[1]s
}

// NewContainer wires the singleton: %[2]s satisfies %[1]s.
func NewContainer() *Container {
	return &Container{f_%[1]s: %[2]s{}}
}
`, iface, impl)

	return []byte(src), nil
}
