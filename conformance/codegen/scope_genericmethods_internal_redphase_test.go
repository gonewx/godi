// ATDD test for Story 6.6 (AC3/D-3, Task 4) — codegen internal (white-box)
// test: scope.Get[T]() / GetRequired[T]() lifecycle correctness.
//
// Activated in Story 6.6 Task 4 (container_gen.go regenerated with scope
// Get[T]/GetRequired[T]; original //go:build atdd_redphase removed).
//
// This is `package codegen` (white-box) because scope is unexported — the
// only way to exercise scope.Get[T] directly is from within the package.
package codegen

import (
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
)

// --- 6.6-SCOPE-001 (AC3/D-3, P0) — scoped service: scope's own instance ----

// TestScope_Get_Scoped_ReturnsOwnInstance_NotRoot: scope.Get[T]() for a
// SCOPED service must return the SCOPE's own instance (s.f_FakeScopedService)
// — never root's (r.f_FakeScopedService). This is exactly the failure mode a
// short-circuited s.root.GetService(...) implementation would produce.
func TestScope_Get_Scoped_ReturnsOwnInstance_NotRoot(t *testing.T) {
	root, err := NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	sc, ok := root.CreateScope().(*scope)
	testutil.True(t, ok, "CreateScope() must return the concrete *scope type")
	t.Cleanup(func() { _ = sc.Close() })

	got, hit := sc.Get[fakes.IFakeScopedService]()
	testutil.True(t, hit, "scope.Get[IFakeScopedService]() must hit — FakeScopedService is registered")
	testutil.True(t, got == sc.f_FakeScopedService,
		"scope.Get[T] must return the SCOPE's own field")
	testutil.True(t, got != root.f_FakeScopedService,
		"scope.Get[T] for a scoped service must NOT return root's field (D-3 short-circuit guard)")
}

// --- 6.6-SCOPE-002 (AC3/D-3, P0) — scoped service equivalence with free fn -

// TestScope_Get_Scoped_MatchesFreeFunction: the method form and the free
// function godi.Get[T](scope) (which resolves through the Provider interface
// — proven correct since Story 2.2) must agree.
func TestScope_Get_Scoped_MatchesFreeFunction(t *testing.T) {
	root, err := NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	sc, ok := root.CreateScope().(*scope)
	testutil.True(t, ok, "CreateScope() must return the concrete *scope type")
	t.Cleanup(func() { _ = sc.Close() })

	methodVal, methodOk := sc.Get[fakes.IFakeScopedService]()
	freeVal, freeOk := godi.Get[fakes.IFakeScopedService](sc)

	testutil.True(t, methodOk && freeOk, "both forms must hit")
	testutil.True(t, methodVal == freeVal,
		"scope.Get[T]() and godi.Get[T](scope) must resolve to the identical scoped instance")
}

// --- 6.6-SCOPE-003 (AC3/D-3, P0) — transient: fresh instance each call -----

// TestScope_Get_Transient_FreshEachCall: scope.Get[T]() for a TRANSIENT
// service must construct a NEW instance every call (never cached) — the same
// contract as s.NewFakeTransientService(). A short-circuit to
// s.root.GetService would still construct fresh instances but would fail to
// track them in the SCOPE's disposer (a leak for disposable transients —
// D-3); that tracking regression is covered procedurally by the existing
// dispose conformance suite, so this test focuses on the observable
// freshness contract shared by both correct and leaking implementations.
func TestScope_Get_Transient_FreshEachCall(t *testing.T) {
	root, err := NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	sc, ok := root.CreateScope().(*scope)
	testutil.True(t, ok, "CreateScope() must return the concrete *scope type")
	t.Cleanup(func() { _ = sc.Close() })

	first, ok1 := sc.Get[fakes.IFakeTransientService]()
	second, ok2 := sc.Get[fakes.IFakeTransientService]()
	testutil.True(t, ok1 && ok2, "both resolutions must hit — FakeTransientService is registered")
	testutil.True(t, first != second,
		"scope.Get[T] for a transient service must construct a fresh instance every call")
}

// --- 6.6-SCOPE-004 (AC3, P0) — singleton: forwarded to root, shared --------

// TestScope_Get_Singleton_ForwardsToRoot: scope.Get[T]() for a SINGLETON must
// return root's shared instance (s.GetService already forwards singletons to
// root — this proves the method form doesn't disturb that dispatch).
func TestScope_Get_Singleton_ForwardsToRoot(t *testing.T) {
	root, err := NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	sc, ok := root.CreateScope().(*scope)
	testutil.True(t, ok, "CreateScope() must return the concrete *scope type")
	t.Cleanup(func() { _ = sc.Close() })

	scopeVal, scopeOk := sc.Get[fakes.IFakeService]()
	rootVal, rootOk := root.Get[fakes.IFakeService]()
	testutil.True(t, scopeOk && rootOk, "both resolutions must hit")
	testutil.True(t, scopeVal == rootVal,
		"scope.Get[T] for a singleton must return the SAME instance as root.Get[T]")
}

// --- 6.6-SCOPE-005 (AC3, P0) — scope.GetRequired[T] miss panics ------------

func TestScope_GetRequired_Miss_Panics(t *testing.T) {
	root, err := NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	sc, ok := root.CreateScope().(*scope)
	testutil.True(t, ok, "CreateScope() must return the concrete *scope type")
	t.Cleanup(func() { _ = sc.Close() })

	recovered := testutil.Panics(t, func() {
		sc.GetRequired[int]()
	})
	_, isErr := recovered.(error)
	testutil.True(t, isErr, "scope.GetRequired[int]() must panic with an error value")
}
