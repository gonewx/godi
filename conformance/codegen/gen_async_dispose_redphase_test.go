// ATDD red-phase scaffold for Story 6.3 (AC6) — codegen conformance: Path γ
// strict async release tests run directly on the PRE-GENERATED B2 container
// (codegen.NewRoot()), not just via differential equivalence (AD-15 γ升格:
// "异步释放 LIFO" is a 保真敏感能力 that must be verified exactly on codegen
// output, mirroring Story 2.3's sync Path γ tests in gen_dispose_redphase_test.go).
//
// This file references fakes that do NOT exist yet (conformance/fakes.
// FakeAsyncOnlyDisposable / FakeSyncAsyncDisposable / AsyncDisposeRecorder,
// Task 6) and assumes conformance/codegen/registration.go registers them
// (regenerating container_gen.go) — neither exists yet. This file will NOT
// compile until Task 6 completes.
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-6-3-async-scope-dispose.md.
//go:build atdd_redphase

package codegen_test

import (
	"context"
	"os"
	"reflect"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
)

// asyncCloser is the structural opt-in surface CloseAsync lives behind
// (D-2) — root/scope implement it without it being part of godi.Provider.
type asyncCloser interface {
	CloseAsync(ctx context.Context) error
}

// --- 6.3-CG-001 (AC6, P0) — Path γ: async LIFO order strict on B2 -----------

// TestCodegen_PathGamma_AsyncLIFO_StrictOnB2: async release LIFO order must
// be verified STRICTLY on the pre-generated B2 container, not just
// differential equivalence.
func TestCodegen_PathGamma_AsyncLIFO_StrictOnB2(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)

	scope := root.CreateScope()
	recorder := fakes.NewAsyncDisposeRecorder()

	asyncOnly := scope.GetService(reflect.TypeFor[*fakes.FakeAsyncOnlyDisposable]())
	asyncOnly.(interface {
		SetRecorder(*fakes.AsyncDisposeRecorder, string)
	}).SetRecorder(recorder, "async-only")

	both := scope.GetService(reflect.TypeFor[*fakes.FakeSyncAsyncDisposable]())
	both.(interface {
		SetRecorder(*fakes.AsyncDisposeRecorder, string)
	}).SetRecorder(recorder, "both")

	ac, ok := scope.(asyncCloser)
	testutil.True(t, ok, "generated scope must implement CloseAsync when async-disposable services exist")
	testutil.NoError(t, ac.CloseAsync(context.Background()))

	labels := recorder.Labels()
	testutil.Equal(t, len(labels), 2)
	testutil.Equal(t, labels[0], "both")
	testutil.Equal(t, labels[1], "async-only")

	testutil.NoError(t, root.Close())
}

// --- 6.3-CG-002 (AC6, P0) — Path γ: singleton not disposed on scope.CloseAsync

// TestCodegen_PathGamma_AsyncSingletonNotDisposedOnScopeCloseAsync: on the B2
// container, scope.CloseAsync must NOT dispose singleton services (AD-12
// async parity) — only root.CloseAsync/Close releases them.
func TestCodegen_PathGamma_AsyncSingletonNotDisposedOnScopeCloseAsync(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)

	scope := root.CreateScope()
	singleton := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableSingleton]())
	testutil.NotNil(t, singleton)

	ac := scope.(asyncCloser)
	testutil.NoError(t, ac.CloseAsync(context.Background()))

	singletonFake := singleton.(interface{ IsDisposed() bool })
	testutil.True(t, !singletonFake.IsDisposed(),
		"singleton must NOT be disposed on scope.CloseAsync — only root (AD-12)")

	rootAc := root.(asyncCloser)
	testutil.NoError(t, rootAc.CloseAsync(context.Background()))
	testutil.True(t, singletonFake.IsDisposed(), "singleton must be disposed after root.CloseAsync")
}

// --- 6.3-CG-003 (AC3, P0) — Path γ: sync Close errors on async-only on B2 ---

// TestCodegen_PathGamma_SyncClose_ErrorsOnAsyncOnly: on the B2 container, the
// SYNC Close must error (not panic) when it holds an async-only service
// (AC3 parity between codegen and runtime).
func TestCodegen_PathGamma_SyncClose_ErrorsOnAsyncOnly(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)

	scope := root.CreateScope()
	_ = scope.GetService(reflect.TypeFor[*fakes.FakeAsyncOnlyDisposable]())

	closeErr := scope.Close()
	testutil.Error(t, closeErr)
	testutil.True(t, strings.Contains(closeErr.Error(), "CloseAsync"),
		"AC3: generated scope.Close on an async-only service must name CloseAsync in the error")

	testutil.NoError(t, root.Close())
}

// --- 6.3-CG-004 (AC6, P0) — Close/CloseAsync dispose-once on B2 -------------

// TestCodegen_CloseAsync_Idempotent_WithAsyncDisposables: CloseAsync on the
// B2 container must be idempotent, and share the SAME closed guard as Close
// (D-6) — repeated calls across either method are no-ops.
func TestCodegen_CloseAsync_Idempotent_WithAsyncDisposables(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)

	_ = root.GetService(reflect.TypeFor[*fakes.FakeSyncAsyncDisposable]())

	ac := root.(asyncCloser)
	testutil.NoError(t, ac.CloseAsync(context.Background()))
	testutil.NoError(t, ac.CloseAsync(context.Background())) // no-op
	testutil.NoError(t, root.Close())                        // no-op, shares the guard (D-6)
}

// --- 6.3-CG-005 (AC6, P1) — regen comparison guard --------------------------

// TestCodegen_RegenComparison_WithAsyncDisposables: the committed
// container_gen.go must reference the async disposer machinery — regression
// guard that async-disposable services are actually wired through codegen.
func TestCodegen_RegenComparison_WithAsyncDisposables(t *testing.T) {
	_, thisFile, _, _ := goruntime.Caller(0)
	containerPath := strings.Replace(thisFile, "gen_async_dispose_redphase_test.go", "container_gen.go", 1)
	committed, readErr := os.ReadFile(containerPath)
	testutil.NoError(t, readErr)

	code := string(committed)
	testutil.True(t, strings.Contains(code, "CloseAllAsync") || strings.Contains(code, "TrackAsync"),
		"regenerated container_gen.go must reference the async disposer machinery (CloseAllAsync/TrackAsync)")
}
