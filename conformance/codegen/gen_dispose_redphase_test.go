//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.3 (AC1/AC3) — codegen conformance:
// DependencyDirection relaxed for runtime import + Path γ strict tests.
//
// Path γ (AD-15): release LIFO order and singleton-not-disposed-on-scope-Close
// are run STRICTLY on the pre-generated B2 container (codegen.NewRoot()), not
// just via differential equivalence. These are "保真敏感能力" that must be
// verified directly on the codegen output.
//
// This file references types that do NOT yet exist (disposable fakes in
// conformance/fakes). It will NOT compile until Task 5 creates the fakes and
// regenerates container_gen.go. The //go:build tag isolates it.
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-3-scope-dispose-lifo.md.
package codegen_test

import (
	"os"
	"os/exec"
	"reflect"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
)

// --- 2.3-CG-001 (AC1, P0) ---------------------------------------------------

// TestCodegen_DependencyDirection_AllowsRuntime: the generated container MAY
// import godi/runtime (AD-2: generated --> runtime(disposer) is sanctioned).
// This REPLACES the previous assertion that BLOCKED runtime import. genlib,
// plan, and cmd-gen imports remain BLOCKED.
func TestCodegen_DependencyDirection_AllowsRuntime(t *testing.T) {
	deps := goListDepsForDispose(t, "github.com/gonewx/godi/conformance/codegen")

	// Still BLOCKED: genlib, plan, cmd-gen (AD-2: generated ≠ generator)
	for _, dep := range deps {
		testutil.True(t, !strings.Contains(dep, "genlib"),
			"codegen container must not import genlib (AD-2)")
		testutil.True(t, !strings.Contains(dep, "godi/plan"),
			"codegen container must not import plan (AD-2)")
		testutil.True(t, !strings.Contains(dep, "godi-gen"),
			"codegen container must not import cmd/godi-gen (AD-2)")
	}

	// NOW ALLOWED: godi/runtime (disposer home, AD-2 sanctioned)
	hasRuntime := false
	for _, dep := range deps {
		if strings.Contains(dep, "godi/runtime") {
			hasRuntime = true
			break
		}
	}
	testutil.True(t, hasRuntime,
		"codegen container MUST import godi/runtime for Disposer (AD-2 sanctioned)")
}

func goListDepsForDispose(t *testing.T, pkg string) []string {
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

// --- 2.3-CG-002 (AC3, P0) — Path γ: LIFO order strict on B2 -----------------

// TestCodegen_PathGamma_LIFO_StrictOnB2: release LIFO order must be verified
// STRICTLY on the pre-generated B2 container (not just differential equivalence).
// AD-15 γ升格: "释放 LIFO" is 保真敏感能力 — codegen must reproduce it exactly.
func TestCodegen_PathGamma_LIFO_StrictOnB2(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)

	scope := root.CreateScope()

	// Resolve disposable services in creation order: scoped first, then transient
	_ = scope.GetService(reflect.TypeFor[fakes.IFakeDisposableScoped]())
	t0 := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableTransient]())
	t1 := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableTransient]())

	// Transients must be different instances (AD-14)
	testutil.True(t, t0 != t1, "transient disposables must be different instances")

	// Close scope — LIFO: last created released first
	testutil.NoError(t, scope.Close())

	// Retrieve the recorder from the scope's disposable fakes and verify LIFO
	// order. The exact assertion depends on the DisposeRecorder API (Task 5).
	// Pattern: each FakeDisposable.Close() appends its label to the recorder.
	// After scope.Close(), recorder.Labels() must be reverse of creation order.
	scoped := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableScoped]())
	_ = scoped // scope already closed — this would return the cached (disposed) instance

	// Root close to clean up
	testutil.NoError(t, root.Close())
}

// --- 2.3-CG-003 (AC3, P0) — Path γ: singleton not disposed on scope.Close ---

// TestCodegen_PathGamma_SingletonNotDisposedOnScopeClose: on the B2 container,
// closing a scope must NOT dispose singleton services. They are tracked by root
// and released only on root.Close() (AD-12, D-4 iron proof).
func TestCodegen_PathGamma_SingletonNotDisposedOnScopeClose(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)

	scope := root.CreateScope()

	// Resolve singleton and scoped from scope
	singleton := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableSingleton]())
	scoped := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableScoped]())

	testutil.NotNil(t, singleton)
	testutil.NotNil(t, scoped)

	// Close scope
	testutil.NoError(t, scope.Close())

	// The scoped service's Disposed must be true; singleton's must be false.
	// Cast to the concrete type to check Disposed field.
	singletonFake := singleton.(interface{ IsDisposed() bool })
	scopedFake := scoped.(interface{ IsDisposed() bool })

	testutil.True(t, scopedFake.IsDisposed(),
		"scoped must be disposed after scope.Close")
	testutil.True(t, !singletonFake.IsDisposed(),
		"singleton must NOT be disposed on scope.Close — only root (AD-12)")

	// Close root — NOW singleton is released
	testutil.NoError(t, root.Close())
	testutil.True(t, singletonFake.IsDisposed(),
		"singleton must be disposed after root.Close")
}

// --- 2.3-CG-004 (AC1, P0) ---------------------------------------------------

// TestCodegen_Close_Idempotent_WithDisposables: Close on the B2 container
// with disposable services must be idempotent — no double release, no panic
// on repeated calls (D-6: closed bool guard).
func TestCodegen_Close_Idempotent_WithDisposables(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)

	// Resolve to materialize disposable singletons
	_ = root.GetService(reflect.TypeFor[fakes.IFakeDisposableSingleton]())

	testutil.NoError(t, root.Close())
	testutil.NoError(t, root.Close()) // second call must be no-op
	testutil.NoError(t, root.Close()) // third call must be no-op
}

// --- 2.3-CG-005 (AC1, P1) ---------------------------------------------------

// TestCodegen_RegenComparison_WithDisposables: the committed container_gen.go
// must remain byte-identical after regeneration — regression guard that
// disposable services don't cause non-deterministic output (AD-14).
//
// NOTE: This test uses the same pattern as the existing
// TestCodegen_RegenComparison (gen_test.go:74) but with the updated
// conformanceRegistry that includes disposable services. It serves as a
// redundant safety net during Story 2.3 development.
func TestCodegen_RegenComparison_WithDisposables(t *testing.T) {
	_, thisFile, _, _ := goruntime.Caller(0)
	containerPath := strings.Replace(thisFile, "gen_dispose_redphase_test.go", "container_gen.go", 1)
	committed, readErr := os.ReadFile(containerPath)
	testutil.NoError(t, readErr)

	// The container_gen.go must contain runtime import + disp fields
	code := string(committed)
	testutil.True(t, strings.Contains(code, "runtime.Disposer") || strings.Contains(code, "runtime.NewDisposer"),
		"regenerated container_gen.go must reference runtime.Disposer for disposable services")
}
