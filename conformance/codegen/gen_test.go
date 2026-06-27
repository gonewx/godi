// Conformance tests for Story 1.5 — codegen container: regeneration-comparison
// guard (AD-14 deterministic output), behavior smoke tests, and dependency
// direction self-check (AD-2).
package codegen_test

import (
	"os"
	"os/exec"
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

func conformanceRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"github.com/gonewx/godi/conformance/fakes"}
	reg.Service(plan.Service{
		Name:     "FakeService",
		GoType:   "fakes.IFakeService",
		Ctor:     "fakes.NewFakeService",
		Lifetime: plan.Singleton,
	})
	return reg
}

// --- 1.5-GEN-001 (AC3, P0) ---------------------------------------------------

// TestCodegen_RegenComparison: the committed container_gen.go must be
// byte-identical to a fresh genlib.Generate run. Drift means someone
// hand-forked the codegen container — violating AD-14.
func TestCodegen_RegenComparison(t *testing.T) {
	reg := conformanceRegistry()
	freshSrc, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)

	_, thisFile, _, _ := goruntime.Caller(0)
	containerPath := filepath.Join(filepath.Dir(thisFile), "container_gen.go")
	committed, readErr := os.ReadFile(containerPath)
	testutil.NoError(t, readErr)

	testutil.Equal(t, string(freshSrc), string(committed))
}

// --- 1.5-GEN-002 (AC3, P0) ---------------------------------------------------

// TestCodegen_GetService_IFakeService: the pre-generated codegen container
// resolves IFakeService to a non-nil *fakes.FakeService (AD-1/AD-16).
func TestCodegen_GetService_IFakeService(t *testing.T) {
	root := codegen.NewRoot()
	t.Cleanup(func() { _ = root.Close() })

	got := root.GetService(reflect.TypeFor[fakes.IFakeService]())
	testutil.NotNil(t, got)
	testutil.Equal(t, reflect.TypeOf(got).String(), "*fakes.FakeService")
}

// --- 1.5-GEN-003 (AC3, P0) ---------------------------------------------------

// TestCodegen_IsService_Consistency: IsService must agree with GetService
// (AD-16 self-consistency).
func TestCodegen_IsService_Consistency(t *testing.T) {
	root := codegen.NewRoot()
	t.Cleanup(func() { _ = root.Close() })

	testutil.True(t, root.IsService(reflect.TypeFor[fakes.IFakeService]()),
		"registered type must report IsService=true")
	testutil.True(t, !root.IsService(reflect.TypeFor[int]()),
		"unregistered type must report IsService=false")
}

// --- 1.5-GEN-004 (AC3, P1) ---------------------------------------------------

// TestCodegen_Close_Idempotent: Close is idempotent (AD-16).
func TestCodegen_Close_Idempotent(t *testing.T) {
	root := codegen.NewRoot()
	testutil.NoError(t, root.Close())
	testutil.NoError(t, root.Close())
}

// --- 1.5-GEN-005 (AC3/AC4, P0) -----------------------------------------------

// TestCodegen_DependencyDirection: the generated container must NOT import
// plan, genlib, runtime, or cmd (AD-2 dependency direction).
func TestCodegen_DependencyDirection(t *testing.T) {
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

func goListDeps(t *testing.T, pkg string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-deps", pkg)
	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GOWORK=") {
			env = append(env, e)
		}
	}
	env = append(env, "GOWORK=off")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps %s: %v", pkg, err)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}
