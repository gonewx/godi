//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.1 (AC3) — codegen dependency container:
// real //go:generate wiring + regeneration determinism + new NewRoot signature
// (Task 5). Story 1.5 committed the single-service container by hand; this story
// wires the real cmd/godi-gen CLI behind //go:generate and regenerates a
// dependency container, guarded byte-identical (AD-14: no hand-fork).
//
// RED: it depends on the regenerated dependency container (FakeConsumer field +
// dependency wiring), the new codegen.NewRoot() (*Root, error) signature, and
// the new fakes.IFakeConsumer fixture — none exist yet → compilation fails under
// `-tags atdd_redphase`. Activation: add conformance/codegen/registration.go
// (form-A DSL for FakeService + FakeConsumer), wire `//go:generate` to
// cmd/godi-gen, regenerate container_gen.go, then delete the //go:build line and
// the blank line after it (folding these into gen_test.go).
//
// goListDeps is reused from gen_test.go (same package codegen_test) — not
// redefined here.
//
// API CONTRACT THIS SCAFFOLD ASSUMES:
//
//	codegen.NewRoot() (*Root, error)
//	genlib.Generate(reg, "codegen") byte-identical to committed container_gen.go
//	fakes.IFakeConsumer / *fakes.FakeConsumer, NewFakeConsumer(IFakeService)
package codegen_test

import (
	"os"
	"path/filepath"
	"reflect"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// conformanceDepsRegistry mirrors the registration that conformance/codegen's
// //go:generate produces: a FakeService singleton and a FakeConsumer singleton
// that depends on it.
func conformanceDepsRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"github.com/gonewx/godi/conformance/fakes"}
	reg.Service(plan.Service{
		Name:     "FakeService",
		GoType:   "fakes.IFakeService",
		Ctor:     "fakes.NewFakeService",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "FakeConsumer",
		GoType:   "fakes.IFakeConsumer",
		Ctor:     "fakes.NewFakeConsumer",
		Lifetime: plan.Singleton,
		Deps:     []string{"FakeService"},
	})
	return reg
}

// --- 2.1-CG-001 (AC3, P0) ----------------------------------------------------

// TestCodegenDeps_RegenComparison: the committed container_gen.go must be
// byte-identical to a fresh genlib.Generate run for the dependency registration
// — drift means a hand-fork (AD-14). After green this guards the real
// //go:generate output stays deterministic.
func TestCodegenDeps_RegenComparison(t *testing.T) {
	freshSrc, err := genlib.Generate(conformanceDepsRegistry(), "codegen")
	testutil.NoError(t, err)

	_, thisFile, _, _ := goruntime.Caller(0)
	containerPath := filepath.Join(filepath.Dir(thisFile), "container_gen.go")
	committed, readErr := os.ReadFile(containerPath)
	testutil.NoError(t, readErr)

	testutil.Equal(t, string(freshSrc), string(committed))
}

// --- 2.1-CG-002 (AC3, P0) ----------------------------------------------------

// TestCodegenDeps_NewRootResolvesInjectedConsumer: the regenerated container
// adopts NewRoot() (*Root, error) (D-1) and resolves the dependency-injected
// FakeConsumer to a non-nil *fakes.FakeConsumer.
func TestCodegenDeps_NewRootResolvesInjectedConsumer(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	got := root.GetService(reflect.TypeFor[fakes.IFakeConsumer]())
	testutil.NotNil(t, got)
	testutil.Equal(t, reflect.TypeOf(got).String(), "*fakes.FakeConsumer")
}

// --- 2.1-CG-003 (AC3, P1) ----------------------------------------------------

// TestCodegenDeps_DependencyDirection: the generated dependency container must
// still NOT import plan, genlib, runtime, or cmd (AD-2) — a regression guard
// that the richer wiring did not pull in the generator's packages.
func TestCodegenDeps_DependencyDirection(t *testing.T) {
	deps := goListDeps(t, "github.com/gonewx/godi/conformance/codegen")
	for _, dep := range deps {
		testutil.True(t, !strings.Contains(dep, "genlib"),
			"codegen container must not import genlib (AD-2)")
		testutil.True(t, !strings.Contains(dep, "godi/plan"),
			"codegen container must not import plan (AD-2)")
		testutil.True(t, !strings.Contains(dep, "godi/runtime"),
			"codegen container must not import runtime (AD-2)")
		testutil.True(t, !strings.Contains(dep, "godi-gen"),
			"codegen container must not import cmd/godi-gen (AD-2)")
	}
}
