// ATDD red-phase scaffold for Story 1.3 — the reflect-free runtime provider
// minimal slice: single SINGLETON resolution (shared instance), Get[T] miss,
// and idempotent Close (AD-1 / AD-13 / AD-14 / AD-16 / AD-17).
//
// Activation (TDD red -> green): delete the //go:build line above and the blank
// line after it once the runtime closure-registration API and the root provider
// exist (Story 1.3 Tasks 1-2). The file then joins the default build and these
// assertions drive the slice to green.
// See _bmad-output/test-artifacts/atdd-checklist-1-3-runtime-provider-singleton.md.
//
// API CONTRACT THIS SCAFFOLD ASSUMES (Story 1.3 Design Decision D1, form B):
//
//	runtime.NewBuilder() *Builder
//	runtime.AddSingleton[T any](b *Builder, factory func(godi.Provider) T) *Builder
//	(*Builder).Build() godi.Provider
//
// The factory carries a godi.Provider parameter as the seam for Epic 2.1
// dependency injection; this story's singletons have no dependencies, so the
// factories below ignore it. If the dev pins a different (but still form-B)
// registration shape, update the calls below to match — the BEHAVIORS asserted
// (shared singleton, miss semantics, idempotent Close, honest keyed/scope
// minimal behavior) are the contract and must not change.
//
// COMPILE-TIME PROVIDER CONTRACT (R2 / AC4): the internal assertion
// `var _ godi.Provider = (*root)(nil)` lives in the implementation file
// (runtime/provider.go, Task 2) because it names the unexported concrete type.
// At this external-package boundary the same guarantee is enforced structurally:
// (*Builder).Build() is declared to RETURN godi.Provider, so the concrete root
// type cannot compile unless it implements all six AD-16 methods.
package runtime_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// --- in-test fakes ----------------------------------------------------------

// logger is a trivial concrete singleton. Pointer identity (*logger is
// comparable) is the lever for the "same instance" assertions.
type logger struct{ id string }

// closableLogger implements io.Closer so AC3's "no double release" can be probed
// WITHOUT forcing release: Story 1.3's minimal Close may legitimately leave it
// unreleased (close count 0); Story 2.3's LIFO disposer will release it exactly
// once (close count 1). Both satisfy "<= 1".
type closableLogger struct{ onClose func() }

func (c *closableLogger) Close() error {
	if c.onClose != nil {
		c.onClose()
	}
	return nil
}

// --- 1.3-UNIT-001 (AC1, P0) -------------------------------------------------

// TestSingleton_SharedAcrossResolves_FactoryRunsOnce: two GetService calls for a
// closure-registered singleton return the SAME instance, and the factory runs at
// most once (singleton cached — AD-1: singleton = Root field). Works for both
// eager (D2 recommended) and lazy materialization: either way the count is 1
// after two resolves.
func TestSingleton_SharedAcrossResolves_FactoryRunsOnce(t *testing.T) {
	var calls int
	b := runtime.NewBuilder()
	runtime.AddSingleton[*logger](b, func(godi.Provider) *logger {
		calls++
		return &logger{}
	})
	p, _ := b.Build()
	t.Cleanup(func() { _ = p.Close() })

	lt := reflect.TypeFor[*logger]()
	first, _ := p.GetService(lt).(*logger)
	second, _ := p.GetService(lt).(*logger)

	testutil.NotNil(t, first)
	testutil.Equal(t, first, second) // identical pointer => shared singleton
	testutil.Equal(t, calls, 1)      // factory executed at most once
}

// --- 1.3-UNIT-002 (AC1, P1) -------------------------------------------------

// TestSingleton_InstanceComesFromClosure_NoReflectiveConstruction: the resolved
// instance is EXACTLY the one the closure built (AD-13: construction is the
// closure's `&logger{}`, reflect.Type is only a dispatch key). A reflective
// `reflect.New` path would hand back a different instance, so pointer identity
// against a sentinel is the observable proxy for "zero reflective construction".
// (The static guarantee that reflect.New/Value.Call never appear on the resolve
// path is enforced by AD-13 + code review; this asserts its consequence.)
func TestSingleton_InstanceComesFromClosure_NoReflectiveConstruction(t *testing.T) {
	sentinel := &logger{id: "from-closure"}
	b := runtime.NewBuilder()
	runtime.AddSingleton[*logger](b, func(godi.Provider) *logger { return sentinel })
	p, _ := b.Build()
	t.Cleanup(func() { _ = p.Close() })

	got, _ := p.GetService(reflect.TypeFor[*logger]()).(*logger)
	testutil.Equal(t, got, sentinel)
}

// --- 1.3-UNIT-003 (AC2, P0) -------------------------------------------------

// TestGet_UnregisteredType_ReturnsZeroFalse: an unregistered type resolves to
// (zero, false) via the reused godi.Get[T] (Story 1.2), never panicking, and the
// underlying GetService/IsService agree on the miss (AD-16).
func TestGet_UnregisteredType_ReturnsZeroFalse(t *testing.T) {
	p, _ := runtime.NewBuilder().Build()
	t.Cleanup(func() { _ = p.Close() })

	v, ok := godi.Get[*logger](p)
	testutil.True(t, !ok, "unregistered type must miss")
	testutil.Nil(t, v) // zero value of *logger is nil

	lt := reflect.TypeFor[*logger]()
	testutil.True(t, !p.IsService(lt), "IsService must be false on a miss")
	testutil.Nil(t, p.GetService(lt))
}

// --- 1.3-UNIT-004 (AC2, P0) -------------------------------------------------

// TestGet_RegisteredSingleton_ReturnsValueTrue_SameInstance: a registered
// singleton resolves to (value, true) via godi.Get[T], IsService is true, and
// the instance is the SAME one GetService returns (self-consistent with AC1).
func TestGet_RegisteredSingleton_ReturnsValueTrue_SameInstance(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*logger](b, func(godi.Provider) *logger { return &logger{} })
	p, _ := b.Build()
	t.Cleanup(func() { _ = p.Close() })

	v, ok := godi.Get[*logger](p)
	testutil.True(t, ok, "registered singleton must hit")
	testutil.NotNil(t, v)

	lt := reflect.TypeFor[*logger]()
	testutil.True(t, p.IsService(lt), "IsService must be true for a registered type")

	viaGetService, _ := p.GetService(lt).(*logger)
	testutil.Equal(t, v, viaGetService) // Get[T] and GetService yield one instance
}

// --- 1.3-UNIT-005 (AC3, P0) -------------------------------------------------

// TestClose_Idempotent: calling Close repeatedly is safe — no panic, and each
// call returns nil (AD-16: Close is idempotent via a one-shot gate).
func TestClose_Idempotent(t *testing.T) {
	p, _ := runtime.NewBuilder().Build()

	testutil.NoError(t, p.Close())
	testutil.NoError(t, p.Close())
	testutil.NoError(t, p.Close())
}

// TestClose_DoesNotDoubleRelease: Close must not release the same disposable
// twice. Story 1.3's minimal Close may not release at all (count 0); Story 2.3's
// LIFO disposer will release exactly once (count 1). The one-shot gate guarantees
// "<= 1" either way. This pins the gate without forcing the disposer (R5).
func TestClose_DoesNotDoubleRelease(t *testing.T) {
	var closeCount int
	b := runtime.NewBuilder()
	runtime.AddSingleton[*closableLogger](b, func(godi.Provider) *closableLogger {
		return &closableLogger{onClose: func() { closeCount++ }}
	})
	p, _ := b.Build()

	_ = p.Close()
	_ = p.Close()

	testutil.Equal(t, closeCount, 1)
}

// --- 1.3-UNIT-006 (AC4, P0) -------------------------------------------------

// TestProvider_SatisfiesGodiProviderContract: the value Build() returns is a
// godi.Provider exposing all six AD-16 methods, and CreateScope() yields a
// godi.Scope (itself a Provider). The `var _ godi.Provider` assignments are the
// compile-time gate; reaching them means the concrete root type implements the
// pinned method face (R2). The internal `var _ godi.Provider = (*root)(nil)`
// assertion is the dev's to add in runtime/provider.go (Task 2).
func TestProvider_SatisfiesGodiProviderContract(t *testing.T) {
	p, _ := runtime.NewBuilder().Build()
	t.Cleanup(func() { _ = p.Close() })

	lt := reflect.TypeFor[*logger]()
	// Exercise every method of the six-method face so the full contract is touched.
	_ = p.GetService(lt)
	_ = p.IsService(lt)
	_ = p.GetKeyedService(lt, nil)
	_ = p.GetKeyedServices(lt, nil)

	var s godi.Scope = p.CreateScope()
	t.Cleanup(func() { _ = s.Close() })
	testutil.NotNil(t, s)
}

// --- 1.3-UNIT-007 (AC4, P1) -------------------------------------------------

// TestCreateScope_SharesRootSingleton: a child scope resolves the SAME singleton
// instance as root (AD-1: singletons are shared across scopes; D3: the minimal
// scope delegates to root). Closing the scope is an idempotent no-op that must
// NOT release the root singleton (AD-12: a singleton is released only with root)
// — root still resolves the same instance afterward.
func TestCreateScope_SharesRootSingleton(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*logger](b, func(godi.Provider) *logger { return &logger{} })
	p, _ := b.Build()
	t.Cleanup(func() { _ = p.Close() })

	lt := reflect.TypeFor[*logger]()
	rootInstance, _ := p.GetService(lt).(*logger)

	scope := p.CreateScope()
	inScope, _ := scope.GetService(lt).(*logger)
	testutil.Equal(t, inScope, rootInstance) // singleton shared into the scope

	testutil.NoError(t, scope.Close()) // scope.Close is an idempotent no-op
	afterScopeClose, _ := p.GetService(lt).(*logger)
	testutil.Equal(t, afterScopeClose, rootInstance) // root singleton untouched
}

// --- 1.3-UNIT-008 (AC4, P2) -------------------------------------------------

// TestKeyed_HonestMinimalBehavior: the keyed methods give honest minimal
// behavior this epoch (no keyed registration until Epic 3). A nil key routes to
// the unkeyed service (AD-17); a non-nil key resolves to nil; GetKeyedServices
// returns nil. No pretending to support keyed lookup.
func TestKeyed_HonestMinimalBehavior(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*logger](b, func(godi.Provider) *logger { return &logger{} })
	p, _ := b.Build()
	t.Cleanup(func() { _ = p.Close() })

	lt := reflect.TypeFor[*logger]()

	viaNilKey, _ := p.GetKeyedService(lt, nil).(*logger)
	unkeyed, _ := p.GetService(lt).(*logger)
	testutil.Equal(t, viaNilKey, unkeyed) // nil key == unkeyed (AD-17)

	testutil.Nil(t, p.GetKeyedService(lt, "some-key")) // no keyed services (Epic 3)
	testutil.Nil(t, p.GetKeyedServices(lt, "some-key"))
}

// --- 1.3-UNIT-009 (AC4, P2) -------------------------------------------------

// TestScope_DelegatesFullResolutionSurface: the minimal child scope (D3) exposes
// the SAME resolution surface as root by delegating every method to it —
// IsService, GetKeyedService (nil-key routing, AD-17), GetKeyedServices, and
// nested CreateScope all behave identically through a scope as through root. A
// grandchild scope still shares the root singleton (AD-1). This pins the scope's
// honest minimal behavior across its whole face, not just GetService/Close.
func TestScope_DelegatesFullResolutionSurface(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*logger](b, func(godi.Provider) *logger { return &logger{} })
	p, _ := b.Build()
	t.Cleanup(func() { _ = p.Close() })

	lt := reflect.TypeFor[*logger]()
	rootInstance, _ := p.GetService(lt).(*logger)

	scope := p.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })

	// IsService delegates to root.
	testutil.True(t, scope.IsService(lt), "scope.IsService must mirror root")
	testutil.True(t, !scope.IsService(reflect.TypeFor[*closableLogger]()), "scope reports miss like root")

	// Keyed methods delegate: nil key routes to the unkeyed singleton (AD-17),
	// a non-nil key misses, and keyed collections are nil (Epic 3).
	viaNilKey, _ := scope.GetKeyedService(lt, nil).(*logger)
	testutil.Equal(t, viaNilKey, rootInstance)
	testutil.Nil(t, scope.GetKeyedService(lt, "some-key"))
	testutil.Nil(t, scope.GetKeyedServices(lt, "some-key"))

	// Nested CreateScope still shares the root singleton (AD-1).
	grandchild := scope.CreateScope()
	t.Cleanup(func() { _ = grandchild.Close() })
	inGrandchild, _ := grandchild.GetService(lt).(*logger)
	testutil.Equal(t, inGrandchild, rootInstance)
}
