// ATDD red-phase scaffold for Story 6.3 (AC2/AC5/AC6) — runtime provider
// async Close: root/scope CloseAsync release disposables, singleton stays
// root-only under CloseAsync (AD-12 parity), sync Close errors on an
// async-only service (AC3 at the provider level), and dispose-once is a
// SHARED guard across Close/CloseAsync (D-6: first-caller-wins).
//
// root.CloseAsync / scope.CloseAsync do not exist yet (Task 3) — this file
// will NOT compile until they are added. CloseAsync is a STRUCTURAL opt-in
// method (NOT part of godi.Provider/Scope, D-2), so tests reach it through
// the local asyncCloser type assertion, exactly like a real caller would.
//
// This file reuses disposeTracker / closableSvc / iClosableSingleton /
// iClosableScoped from provider_dispose_redphase_test.go (same package,
// already compiled) for the "sync-only" leg of the D-4 matrix.
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-6-3-async-scope-dispose.md.
package runtime_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// asyncCloser is the structural opt-in surface CloseAsync lives behind
// (D-2) — root/scope implement it without it being part of godi.Provider.
type asyncCloser interface {
	CloseAsync(ctx context.Context) error
}

// --- async-only + both fixtures (async-specific; sync-only reuses provider_dispose_redphase_test.go) ---

// asyncClosableSvc implements ONLY godi.AsyncDisposable — the "async-only"
// row of the D-4 prefer-async matrix.
type asyncClosableSvc struct {
	label    string
	tracker  *disposeTracker
	disposed bool
}

func (a *asyncClosableSvc) CloseAsync(ctx context.Context) error {
	a.disposed = true
	if a.tracker != nil {
		a.tracker.labels = append(a.tracker.labels, a.label)
	}
	return nil
}

type iAsyncOnlySingleton interface{ asyncOnlySingleton() }
type iAsyncOnlyScoped interface{ asyncOnlyScoped() }

func (*asyncClosableSvc) asyncOnlySingleton() {}
func (*asyncClosableSvc) asyncOnlyScoped()    {}

// bothClosableSvc implements BOTH io.Closer and godi.AsyncDisposable.
// closedVia records which one actually fired — the prefer-async witness.
type bothClosableSvc struct {
	label     string
	tracker   *disposeTracker
	closedVia string
}

func (b *bothClosableSvc) Close() error {
	b.closedVia = "sync"
	if b.tracker != nil {
		b.tracker.labels = append(b.tracker.labels, b.label)
	}
	return nil
}

func (b *bothClosableSvc) CloseAsync(ctx context.Context) error {
	b.closedVia = "async"
	if b.tracker != nil {
		b.tracker.labels = append(b.tracker.labels, b.label)
	}
	return nil
}

type iBothSingleton interface{ bothSingleton() }

func (*bothClosableSvc) bothSingleton() {}

// --- 6.3-PROV-001 (AC2/AC5, P0) ---------------------------------------------

// TestRT_RootCloseAsync_ReleasesAsyncOnlySingleton: root.CloseAsync must
// release an async-only singleton via its CloseAsync method.
func TestRT_RootCloseAsync_ReleasesAsyncOnlySingleton(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddSingleton[iAsyncOnlySingleton](b, func(godi.Provider) iAsyncOnlySingleton {
		return &asyncClosableSvc{label: "singleton", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)

	svc := p.GetService(reflect.TypeFor[iAsyncOnlySingleton]()).(*asyncClosableSvc)
	testutil.True(t, !svc.disposed, "async-only singleton must not be disposed before CloseAsync")

	ac, ok := p.(asyncCloser)
	testutil.True(t, ok, "root must implement CloseAsync (structural opt-in, D-2)")
	testutil.NoError(t, ac.CloseAsync(context.Background()))
	testutil.True(t, svc.disposed, "async-only singleton must be disposed after root.CloseAsync")
}

// --- 6.3-PROV-002 (AC3, P0) --------------------------------------------------

// TestRT_RootClose_Sync_ErrorsOnAsyncOnlySingleton: root's SYNC Close must
// return an explicit error (not panic) when it holds an async-only service,
// naming CloseAsync as the remedy (AC3 at the provider level).
func TestRT_RootClose_Sync_ErrorsOnAsyncOnlySingleton(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iAsyncOnlySingleton](b, func(godi.Provider) iAsyncOnlySingleton {
		return &asyncClosableSvc{label: "singleton"}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	_ = p.GetService(reflect.TypeFor[iAsyncOnlySingleton]())

	closeErr := p.Close()
	testutil.Error(t, closeErr)
	testutil.True(t, strings.Contains(closeErr.Error(), "CloseAsync"),
		"AC3: root.Close on an async-only singleton must name CloseAsync in the error")
}

// --- 6.3-PROV-003 (AC2, P0) --------------------------------------------------

// TestRT_RootCloseAsync_PrefersAsyncOnBothImplementor: through the REAL
// provider, a both-implementor singleton must be released via CloseAsync
// when root.CloseAsync runs (D-4 prefer-async, end-to-end).
func TestRT_RootCloseAsync_PrefersAsyncOnBothImplementor(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddSingleton[iBothSingleton](b, func(godi.Provider) iBothSingleton {
		return &bothClosableSvc{label: "svc", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	svc := p.GetService(reflect.TypeFor[iBothSingleton]()).(*bothClosableSvc)

	ac := p.(asyncCloser)
	testutil.NoError(t, ac.CloseAsync(context.Background()))

	testutil.Equal(t, svc.closedVia, "async")
	testutil.Equal(t, len(tracker.labels), 1)
}

// --- 6.3-PROV-004 (AC5, P0) --------------------------------------------------

// TestRT_SingletonNotReleased_OnScopeCloseAsync: closing a scope's
// CloseAsync must NOT release singletons — the async analogue of AD-12's
// iron proof (only root.CloseAsync/Close releases singletons).
func TestRT_SingletonNotReleased_OnScopeCloseAsync(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddSingleton[iAsyncOnlySingleton](b, func(godi.Provider) iAsyncOnlySingleton {
		return &asyncClosableSvc{label: "singleton", tracker: tracker}
	})
	runtime.AddScoped[iAsyncOnlyScoped](b, func(godi.Provider) iAsyncOnlyScoped {
		return &asyncClosableSvc{label: "scoped", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)

	scope := p.CreateScope()
	singleton := scope.GetService(reflect.TypeFor[iAsyncOnlySingleton]()).(*asyncClosableSvc)
	scoped := scope.GetService(reflect.TypeFor[iAsyncOnlyScoped]()).(*asyncClosableSvc)

	sac, ok := scope.(asyncCloser)
	testutil.True(t, ok, "scope must implement CloseAsync (structural opt-in, D-2)")
	testutil.NoError(t, sac.CloseAsync(context.Background()))

	testutil.True(t, scoped.disposed, "scoped async-only service must be disposed after scope.CloseAsync")
	testutil.True(t, !singleton.disposed,
		"singleton must NOT be disposed on scope.CloseAsync — only root (AD-12 async parity, AC5)")

	rac := p.(asyncCloser)
	testutil.NoError(t, rac.CloseAsync(context.Background()))
	testutil.True(t, singleton.disposed, "singleton must be disposed after root.CloseAsync")
}

// --- 6.3-PROV-005 (AC2, P1) --------------------------------------------------

// TestRT_ScopeCloseAsync_LIFO_MixedSyncAsync: scope.CloseAsync must release
// mixed sync-only/async-only scoped services in LIFO (creation-reverse) order.
func TestRT_ScopeCloseAsync_LIFO_MixedSyncAsync(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddScoped[iClosableScoped](b, func(godi.Provider) iClosableScoped {
		return &closableSvc{label: "sync-scoped", tracker: tracker}
	})
	runtime.AddScoped[iAsyncOnlyScoped](b, func(godi.Provider) iAsyncOnlyScoped {
		return &asyncClosableSvc{label: "async-scoped", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)

	scope := p.CreateScope()
	// Creation order: sync-scoped first, then async-scoped.
	_ = scope.GetService(reflect.TypeFor[iClosableScoped]())
	_ = scope.GetService(reflect.TypeFor[iAsyncOnlyScoped]())

	ac := scope.(asyncCloser)
	testutil.NoError(t, ac.CloseAsync(context.Background()))

	// LIFO: async-scoped (created last) releases first.
	testutil.Equal(t, len(tracker.labels), 2)
	testutil.Equal(t, tracker.labels[0], "async-scoped")
	testutil.Equal(t, tracker.labels[1], "sync-scoped")
}

// --- 6.3-PROV-006 (AC2/AC6, P0) — dispose-once shared guard (D-6) -----------

// TestRT_DisposeOnce_CloseFirst_ThenCloseAsyncNoOp: once root.Close (sync)
// has run, a subsequent root.CloseAsync must be a no-op — the two methods
// share ONE closed guard, first-caller-wins (D-6, #38 DisposeAsyncOnce).
func TestRT_DisposeOnce_CloseFirst_ThenCloseAsyncNoOp(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddSingleton[iBothSingleton](b, func(godi.Provider) iBothSingleton {
		return &bothClosableSvc{label: "svc", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	_ = p.GetService(reflect.TypeFor[iBothSingleton]())

	testutil.NoError(t, p.Close()) // first caller: sync wins
	testutil.Equal(t, len(tracker.labels), 1)
	testutil.Equal(t, tracker.labels[0], "svc")

	ac := p.(asyncCloser)
	testutil.NoError(t, ac.CloseAsync(context.Background())) // must be a no-op
	testutil.True(t, len(tracker.labels) == 1,
		"CloseAsync after Close must be a no-op sharing the SAME closed guard (D-6)")
}

// --- 6.3-PROV-007 (AC2/AC6, P0) — dispose-once shared guard, reverse order --

// TestRT_DisposeOnce_CloseAsyncFirst_ThenCloseNoOp: the reverse ordering —
// CloseAsync first, then Close must be a no-op (D-6 first-caller-wins,
// symmetric to 6.3-PROV-006).
func TestRT_DisposeOnce_CloseAsyncFirst_ThenCloseNoOp(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddSingleton[iBothSingleton](b, func(godi.Provider) iBothSingleton {
		return &bothClosableSvc{label: "svc", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	_ = p.GetService(reflect.TypeFor[iBothSingleton]())

	ac := p.(asyncCloser)
	testutil.NoError(t, ac.CloseAsync(context.Background())) // first caller: async wins (prefer-async)
	testutil.Equal(t, len(tracker.labels), 1)
	testutil.Equal(t, tracker.labels[0], "svc")

	testutil.NoError(t, p.Close()) // must be a no-op
	testutil.True(t, len(tracker.labels) == 1,
		"Close after CloseAsync must be a no-op sharing the SAME closed guard (D-6)")
}
