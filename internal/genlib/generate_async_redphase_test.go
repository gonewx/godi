// ATDD red-phase scaffold for Story 6.3 (AC2/AC6) — genlib emit: TrackAsync
// calls, CloseAsync emit on Root + scope, plan.Service.AsyncDisposable field,
// "context" import gating, and a new async-disposable golden — while the
// existing purely-sync disposable emit (Story 2.3) stays byte-identical
// (D-8: hasAnyAsync guards CloseAsync/context, independent of hasDisposables).
//
// plan.Service has no AsyncDisposable field yet (Task 4) — the struct
// literals below reference an unknown field, so this file will NOT compile
// until plan.Service gains it AND genlib.Generate consumes it (Task 4/5).
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-6-3-async-scope-dispose.md.
package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// asyncDisposableRegistry returns a three-lifetime registration where each
// service is ONLY async-disposable (AsyncDisposable: true, Disposable left
// false) — the minimum-viable input for the TrackAsync/CloseAsync emit path.
func asyncDisposableRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:            "DBConn",
		GoType:          "svc.IDBConn",
		Ctor:            "svc.NewDBConn",
		Lifetime:        plan.Singleton,
		AsyncDisposable: true,
	})
	reg.Service(plan.Service{
		Name:            "Session",
		GoType:          "svc.ISession",
		Ctor:            "svc.NewSession",
		Lifetime:        plan.Scoped,
		AsyncDisposable: true,
	})
	reg.Service(plan.Service{
		Name:            "TempFile",
		GoType:          "svc.ITempFile",
		Ctor:            "svc.NewTempFile",
		Lifetime:        plan.Transient,
		AsyncDisposable: true,
	})
	return reg
}

// mixedAsyncRegistry mixes a sync-only, an async-only, and a both-disposable
// singleton — verifies Track (sync/both, D-3 UNCHANGED path) vs TrackAsync
// (async-only) dispatch in the SAME container.
func mixedAsyncRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:       "SyncOnly",
		GoType:     "svc.ISyncOnly",
		Ctor:       "svc.NewSyncOnly",
		Lifetime:   plan.Singleton,
		Disposable: true,
	})
	reg.Service(plan.Service{
		Name:            "AsyncOnly",
		GoType:          "svc.IAsyncOnly",
		Ctor:            "svc.NewAsyncOnly",
		Lifetime:        plan.Singleton,
		AsyncDisposable: true,
	})
	reg.Service(plan.Service{
		Name:            "Both",
		GoType:          "svc.IBoth",
		Ctor:            "svc.NewBoth",
		Lifetime:        plan.Singleton,
		Disposable:      true,
		AsyncDisposable: true,
	})
	return reg
}

// --- 6.3-GEN-001 (AC2, P0) ---------------------------------------------------

// TestEmitAsyncDispose_TrackAsyncCallsEmitted: NewRoot must emit TrackAsync
// calls for async-only services (D-8 Track dispatch).
func TestEmitAsyncDispose_TrackAsyncCallsEmitted(t *testing.T) {
	src, err := genlib.Generate(asyncDisposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, ".disp.TrackAsync("),
		"NewRoot must emit TrackAsync calls for async-only disposable services")
}

// --- 6.3-GEN-002 (AC6, P0) ---------------------------------------------------

// TestEmitAsyncDispose_SyncTrackUnchangedForBoth: a both-disposable service
// (Disposable AND AsyncDisposable) must STILL go through the existing
// `disp.Track(x.(io.Closer))` path — D-3 says the sync emit path is
// UNCHANGED so Story 2.3's goldens stay byte-identical.
func TestEmitAsyncDispose_SyncTrackUnchangedForBoth(t *testing.T) {
	src, err := genlib.Generate(mixedAsyncRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, ".disp.Track(") && strings.Contains(code, ".(io.Closer))"),
		"a both-disposable service must still emit disp.Track(x.(io.Closer)) — the UNCHANGED sync path (D-3)")
	testutil.True(t, strings.Contains(code, ".disp.TrackAsync("),
		"the async-only sibling service in the same registry must emit TrackAsync")
}

// --- 6.3-GEN-003 (AC2, P0) ---------------------------------------------------

// TestEmitAsyncDispose_CloseAsyncMethodEmittedOnRoot: Root must get a
// CloseAsync(ctx context.Context) error method calling disp.CloseAllAsync(ctx)
// when async-disposable services exist.
func TestEmitAsyncDispose_CloseAsyncMethodEmittedOnRoot(t *testing.T) {
	src, err := genlib.Generate(asyncDisposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	idx := strings.Index(code, "func (r *Root) CloseAsync(ctx context.Context) error")
	testutil.True(t, idx >= 0, "Root must have a CloseAsync(ctx context.Context) error method")

	rootCloseAsyncBlock := code[idx:]
	nextFunc := strings.Index(rootCloseAsyncBlock[1:], "\nfunc ")
	if nextFunc > 0 {
		rootCloseAsyncBlock = rootCloseAsyncBlock[:nextFunc+1]
	}
	testutil.True(t, strings.Contains(rootCloseAsyncBlock, "disp.CloseAllAsync(ctx)"),
		"Root.CloseAsync must call disp.CloseAllAsync(ctx)")
}

// --- 6.3-GEN-004 (AC2, P0) ---------------------------------------------------

// TestEmitAsyncDispose_CloseAsyncMethodEmittedOnScope: scope must get the
// symmetric CloseAsync method calling s.disp.CloseAllAsync(ctx).
func TestEmitAsyncDispose_CloseAsyncMethodEmittedOnScope(t *testing.T) {
	src, err := genlib.Generate(asyncDisposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	idx := strings.Index(code, "func (s *scope) CloseAsync(ctx context.Context) error")
	testutil.True(t, idx >= 0, "scope must have a CloseAsync(ctx context.Context) error method")

	scopeCloseAsyncBlock := code[idx:]
	nextFunc := strings.Index(scopeCloseAsyncBlock[1:], "\nfunc ")
	if nextFunc > 0 {
		scopeCloseAsyncBlock = scopeCloseAsyncBlock[:nextFunc+1]
	}
	testutil.True(t, strings.Contains(scopeCloseAsyncBlock, "disp.CloseAllAsync(ctx)"),
		"scope.CloseAsync must call disp.CloseAllAsync(ctx)")
}

// --- 6.3-GEN-005 (AC6, P0) ---------------------------------------------------

// TestEmitAsyncDispose_ContextImportEmitted: a container with async-
// disposable services must import "context" (needed by the CloseAsync
// method signatures).
func TestEmitAsyncDispose_ContextImportEmitted(t *testing.T) {
	src, err := genlib.Generate(asyncDisposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, `"context"`),
		"an async-disposable container must import \"context\"")
}

// --- 6.3-GEN-006 (AC6, P1) ---------------------------------------------------

// TestEmitAsyncDispose_NoContextImportWhenNoAsyncDisposables: a purely-sync
// disposable container (Story 2.3's disposableRegistry, no AsyncDisposable
// anywhere) must NOT import "context" or emit CloseAsync — hasAnyAsync must
// be independent of hasDisposables (D-8), keeping the existing golden intact.
func TestEmitAsyncDispose_NoContextImportWhenNoAsyncDisposables(t *testing.T) {
	src, err := genlib.Generate(disposableRegistry(), "codegen") // from generate_dispose_redphase_test.go (Story 2.3)
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, !strings.Contains(code, `"context"`),
		"a purely-sync disposable container must NOT import \"context\" (D-8 hasAnyAsync guard)")
	testutil.True(t, !strings.Contains(code, "CloseAsync"),
		"a purely-sync disposable container must NOT emit CloseAsync (D-8 hasAnyAsync guard)")
	testutil.True(t, !strings.Contains(code, "TrackAsync"),
		"a purely-sync disposable container must NOT emit TrackAsync (D-8 hasAnyAsync guard)")

	// Story 2.3's golden must stay byte-identical — adding the AsyncDisposable
	// field/feature must NOT perturb pure-sync emit.
	testutil.Golden(t, "disposable-container", src)
}

// --- 6.3-GEN-007 (AC2, P0) ---------------------------------------------------

// TestEmitAsyncDispose_TransientMethodTracksAsyncDisposable: New{T}() for an
// async-only transient must call TrackAsync on the new instance.
func TestEmitAsyncDispose_TransientMethodTracksAsyncDisposable(t *testing.T) {
	src, err := genlib.Generate(asyncDisposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	idx := strings.Index(code, "NewTempFile()")
	testutil.True(t, idx >= 0, "NewTempFile transient method must exist")
	methodBlock := code[idx:]
	nextFunc := strings.Index(methodBlock[1:], "\nfunc ")
	if nextFunc > 0 {
		methodBlock = methodBlock[:nextFunc+1]
	}
	testutil.True(t, strings.Contains(methodBlock, ".disp.TrackAsync("),
		"New{T}() for an async-only transient must call disp.TrackAsync")
}

// --- 6.3-GEN-008 (AC6, P0) ---------------------------------------------------

// TestEmitAsyncDispose_RuntimeImportStillEmitted: an async-only container
// still needs godi/runtime for the Disposer, exactly like a sync-only one.
func TestEmitAsyncDispose_RuntimeImportStillEmitted(t *testing.T) {
	src, err := genlib.Generate(asyncDisposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, `"github.com/gonewx/godi/runtime"`),
		"an async-disposable container must still import godi/runtime for Disposer (AD-2)")
}

// --- 6.3-GEN-009 (AC2, P1) ---------------------------------------------------

// TestEmitAsyncDispose_AsyncDisposableGolden: the async-disposable container
// emit must be byte-stable — anchored by a new golden file (disp field +
// Track/TrackAsync + CloseAll/CloseAllAsync + context import + closed guard).
func TestEmitAsyncDispose_AsyncDisposableGolden(t *testing.T) {
	src, err := genlib.Generate(asyncDisposableRegistry(), "codegen")
	testutil.NoError(t, err)
	testutil.Golden(t, "async-disposable-container", src)
}
