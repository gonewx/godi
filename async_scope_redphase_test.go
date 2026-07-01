// ATDD red-phase scaffold for Story 6.3 (AC1/AC4) — godi package: the
// AsyncDisposable structural contract, the AsyncScope wrapper, and
// CreateAsyncScope. Ports AsyncServiceScopeTests (6 Fact, dotnet
// .../DI.Tests/AsyncServiceScopeTests.cs) using two hand-rolled fake scopes
// (fakeSyncScope / fakeAsyncScope) — this test never touches a real
// container, mirroring dotnet's FakeSyncServiceScope / FakeAsyncServiceScope
// (D-5).
//
// None of godi.AsyncDisposable / godi.AsyncScope / godi.NewAsyncScope /
// godi.CreateAsyncScope exist yet (Task 1). This file will NOT compile until
// they are added to the godi package — the //go:build tag isolates it from
// the default build.
//
// API CONTRACT (Story 6.3 D-1/D-2/D-5):
//
//	type AsyncDisposable interface { CloseAsync(ctx context.Context) error }
//	type AsyncScope struct{ ... }               // value-semantics wrapper
//	func NewAsyncScope(s Scope) AsyncScope       // s == nil -> panic
//	func (a AsyncScope) Provider() Provider      // exposes the wrapped scope
//	func (a AsyncScope) Close() error            // ALWAYS forwards to scope.Close (sync)
//	func (a AsyncScope) CloseAsync(ctx context.Context) error // prefers scope.(AsyncDisposable).CloseAsync, else scope.Close
//	func CreateAsyncScope(p Provider) AsyncScope { return NewAsyncScope(p.CreateScope()) }
//
// AsyncDisposable is a STRUCTURAL opt-in interface — it must NOT be added to
// the Provider/Scope interface (AD-16 seven-method face stays exactly as-is,
// D-2). fakeSyncScope below implements godi.Scope but deliberately does NOT
// implement AsyncDisposable, so it drives the CloseAsync-falls-back-to-Close
// branch (test 4); fakeAsyncScope adds CloseAsync on top, driving the
// smart-delegation branch (test 6).
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-6-3-async-scope-dispose.md.
package godi_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
)

// --- fake scopes (mirror dotnet FakeSyncServiceScope / FakeAsyncServiceScope) ---

// fakeSyncScope implements godi.Scope (the full seven-method AD-16 face) as a
// no-op, plus records whether Close was called. It does NOT implement
// AsyncDisposable — this is the "wrapped scope only supports sync dispose"
// half of the AsyncScope delegation matrix.
type fakeSyncScope struct {
	closeCalled bool
}

var _ godi.Scope = (*fakeSyncScope)(nil)

func (f *fakeSyncScope) GetService(t reflect.Type) any                  { return nil }
func (f *fakeSyncScope) IsService(t reflect.Type) bool                  { return false }
func (f *fakeSyncScope) IsKeyedService(t reflect.Type, key any) bool    { return false }
func (f *fakeSyncScope) GetKeyedService(t reflect.Type, key any) any    { return nil }
func (f *fakeSyncScope) GetKeyedServices(t reflect.Type, key any) []any { return nil }
func (f *fakeSyncScope) CreateScope() godi.Scope                        { return f }
func (f *fakeSyncScope) Close() error {
	f.closeCalled = true
	return nil
}

// fakeAsyncScope embeds fakeSyncScope and additionally implements
// godi.AsyncDisposable — the "wrapped scope supports async dispose" half.
type fakeAsyncScope struct {
	fakeSyncScope
	closeAsyncCalled bool
}

var _ godi.AsyncDisposable = (*fakeAsyncScope)(nil)

func (f *fakeAsyncScope) CloseAsync(ctx context.Context) error {
	f.closeAsyncCalled = true
	return nil
}

// fakeScopeFactory is a minimal godi.Provider whose CreateScope returns a
// preset scope — just enough surface for CreateAsyncScope (AC4) to exercise.
type fakeScopeFactory struct{ scope godi.Scope }

var _ godi.Provider = (*fakeScopeFactory)(nil)

func (f *fakeScopeFactory) GetService(t reflect.Type) any                  { return nil }
func (f *fakeScopeFactory) IsService(t reflect.Type) bool                  { return false }
func (f *fakeScopeFactory) IsKeyedService(t reflect.Type, key any) bool    { return false }
func (f *fakeScopeFactory) GetKeyedService(t reflect.Type, key any) any    { return nil }
func (f *fakeScopeFactory) GetKeyedServices(t reflect.Type, key any) []any { return nil }
func (f *fakeScopeFactory) CreateScope() godi.Scope                        { return f.scope }
func (f *fakeScopeFactory) Close() error                                   { return nil }

// --- 6.3-SCOPE-001 (AC4, P0) — ThrowsIfServiceScopeIsNull -------------------

func TestAsyncScope_NilScope_Panics(t *testing.T) {
	testutil.Panics(t, func() {
		godi.NewAsyncScope(nil)
	})
}

// --- 6.3-SCOPE-002 (AC4, P0) — ReturnsServiceProviderFromWrappedScope -------

func TestAsyncScope_Provider_ReturnsWrappedScope(t *testing.T) {
	inner := &fakeSyncScope{}
	a := godi.NewAsyncScope(inner)

	testutil.True(t, a.Provider() == godi.Provider(inner),
		"AsyncScope.Provider() must return the exact wrapped scope")
}

// --- 6.3-SCOPE-003 (AC4, P0) — CallsDisposeOnWrappedSyncScopeOnDispose ------

func TestAsyncScope_Close_ForwardsToWrappedSyncScope(t *testing.T) {
	inner := &fakeSyncScope{}
	a := godi.NewAsyncScope(inner)

	testutil.NoError(t, a.Close())
	testutil.True(t, inner.closeCalled, "Close() must forward to the wrapped scope's Close")
}

// --- 6.3-SCOPE-004 (AC4, P0) — CallsDisposeOnWrappedSyncScopeOnDisposeAsync -

func TestAsyncScope_CloseAsync_FallsBackToSyncOnSyncOnlyScope(t *testing.T) {
	inner := &fakeSyncScope{}
	a := godi.NewAsyncScope(inner)

	testutil.NoError(t, a.CloseAsync(context.Background()))
	testutil.True(t, inner.closeCalled,
		"CloseAsync() on a sync-only wrapped scope must fall back to Close()")
}

// --- 6.3-SCOPE-005 (AC4, P0) — CallsDisposeOnWrappedAsyncScopeOnDispose -----

func TestAsyncScope_Close_OnAsyncScope_StaysSynchronous(t *testing.T) {
	inner := &fakeAsyncScope{}
	a := godi.NewAsyncScope(inner)

	testutil.NoError(t, a.Close())
	testutil.True(t, inner.closeCalled,
		"Close() must ALWAYS forward to the wrapped scope's sync Close — even when the scope also supports async")
	testutil.True(t, !inner.closeAsyncCalled,
		"Close() must NOT call CloseAsync, even on a scope that supports it")
}

// --- 6.3-SCOPE-006 (AC4, P0) — CallsDisposeAsyncOnWrappedSyncScopeOnDisposeAsync

func TestAsyncScope_CloseAsync_PrefersAsyncOnAsyncScope(t *testing.T) {
	inner := &fakeAsyncScope{}
	a := godi.NewAsyncScope(inner)

	testutil.NoError(t, a.CloseAsync(context.Background()))
	testutil.True(t, inner.closeAsyncCalled,
		"CloseAsync() must call the wrapped scope's CloseAsync when it implements AsyncDisposable")
	testutil.True(t, !inner.closeCalled,
		"CloseAsync() must NOT fall back to sync Close when async is available (prefer-async, D-4)")
}

// --- 6.3-SCOPE-007 (AC4, P0) — CreateAsyncScope wraps p.CreateScope() -------

func TestCreateAsyncScope_WrapsProviderCreateScope(t *testing.T) {
	inner := &fakeSyncScope{}
	p := &fakeScopeFactory{scope: inner}

	a := godi.CreateAsyncScope(p)

	testutil.True(t, a.Provider() == godi.Provider(inner),
		"CreateAsyncScope must wrap p.CreateScope()'s result")
}
