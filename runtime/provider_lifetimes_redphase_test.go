// ATDD red-phase scaffold for Story 2.2 (AC1/AC2) — runtime provider: scoped
// and transient resolution + real CreateScope + nesting.
//
// Story 1.3/2.1 delivered singleton-only resolution. scope was a pure root
// delegate. This story upgrades runtime to support three lifetimes:
//   - Scoped: lazy-memoize per-provider layer (root = rootScoped, scope = scope.scoped)
//   - Transient: factory executed every time, never cached (AD-14)
//   - CreateScope returns a REAL scope with independent scoped cache
//
// New registration API:
//
//	runtime.AddScoped[T](b *Builder, factory func(godi.Provider) T) *Builder
//	runtime.AddTransient[T](b *Builder, factory func(godi.Provider) T) *Builder
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-2-scoped-transient-lifetimes.md.
package runtime_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// --- in-test fakes (local, no conformance/fakes dependency) ----------------

type iScopedSvc interface{ scopedSvc() }

type scopedSvc struct{ id int }

func (*scopedSvc) scopedSvc() {}

var scopedIDCounter int

func newScopedSvc() *scopedSvc {
	scopedIDCounter++
	return &scopedSvc{id: scopedIDCounter}
}

type iTransientSvc interface{ transientSvc() }

type transientSvc struct{ id int }

func (*transientSvc) transientSvc() {}

var transientIDCounter int

func newTransientSvc() *transientSvc {
	transientIDCounter++
	return &transientSvc{id: transientIDCounter}
}

// iTransientWithScopedDep is the AC2 fixture: a transient that holds a scoped
// dependency, exposing it for identity comparison.
type iTransientWithScopedDep interface {
	transientWithScopedDep()
	ScopedDep() iScopedSvc
}

type transientWithScopedDep struct {
	dep iScopedSvc
}

func (*transientWithScopedDep) transientWithScopedDep() {}
func (t *transientWithScopedDep) ScopedDep() iScopedSvc { return t.dep }

// --- 2.2-RT-001 (AC1, P0) ---------------------------------------------------

// TestRT_ScopedFromRoot_RootActsAsRootScope: resolving a scoped service from
// root returns a root-level scoped instance (D-2: root is root scope). Two
// resolves from root return the SAME instance.
func TestRT_ScopedFromRoot_RootActsAsRootScope(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[iScopedSvc](b, func(godi.Provider) iScopedSvc {
		return newScopedSvc()
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	st := reflect.TypeFor[iScopedSvc]()
	first := p.GetService(st)
	second := p.GetService(st)

	testutil.NotNil(t, first)
	testutil.Equal(t, first, second) // root-level scoped: same instance
}

// --- 2.2-RT-002 (AC1, P0) ---------------------------------------------------

// TestRT_ScopedFromScope_DiffersFromRoot: a scoped service resolved from a
// child scope must be a DIFFERENT instance than the root-level scoped
// (ScopedServiceCanBeResolved — D-2 iron proof).
func TestRT_ScopedFromScope_DiffersFromRoot(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[iScopedSvc](b, func(godi.Provider) iScopedSvc {
		return newScopedSvc()
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	st := reflect.TypeFor[iScopedSvc]()
	rootScoped := p.GetService(st)

	scope := p.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })

	scopeScoped := scope.GetService(st)

	testutil.NotNil(t, rootScoped)
	testutil.NotNil(t, scopeScoped)
	testutil.True(t, rootScoped != scopeScoped,
		"root-level scoped must differ from scope-level scoped (D-2)")
}

// --- 2.2-RT-003 (AC1, P0) ---------------------------------------------------

// TestRT_ScopedWithinScope_SameInstance: two resolves of a scoped service from
// the SAME scope return the SAME instance (memoized per scope).
func TestRT_ScopedWithinScope_SameInstance(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[iScopedSvc](b, func(godi.Provider) iScopedSvc {
		return newScopedSvc()
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	scope := p.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })

	st := reflect.TypeFor[iScopedSvc]()
	first := scope.GetService(st)
	second := scope.GetService(st)

	testutil.NotNil(t, first)
	testutil.Equal(t, first, second) // same scope → same scoped instance
}

// --- 2.2-RT-004 (AC1, P0) ---------------------------------------------------

// TestRT_NestedScope_ScopedDiffers: nested scopes each have independent scoped
// caches — outer and inner scoped instances differ (NestedScopedServiceCanBeResolved).
func TestRT_NestedScope_ScopedDiffers(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[iScopedSvc](b, func(godi.Provider) iScopedSvc {
		return newScopedSvc()
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	outer := p.CreateScope()
	t.Cleanup(func() { _ = outer.Close() })

	inner := outer.CreateScope()
	t.Cleanup(func() { _ = inner.Close() })

	st := reflect.TypeFor[iScopedSvc]()
	outerScoped := outer.GetService(st)
	innerScoped := inner.GetService(st)

	testutil.NotNil(t, outerScoped)
	testutil.NotNil(t, innerScoped)
	testutil.True(t, outerScoped != innerScoped,
		"nested scope scoped must differ from outer scope scoped")
}

// --- 2.2-RT-005 (AC1, P0) ---------------------------------------------------

// TestRT_TransientEveryResolveIsDifferent: two resolves of a transient service
// from the same provider return DIFFERENT instances — transients are never
// cached (AD-14).
func TestRT_TransientEveryResolveIsDifferent(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddTransient[iTransientSvc](b, func(godi.Provider) iTransientSvc {
		return newTransientSvc()
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	tt := reflect.TypeFor[iTransientSvc]()
	first := p.GetService(tt)
	second := p.GetService(tt)

	testutil.NotNil(t, first)
	testutil.NotNil(t, second)
	testutil.True(t, first != second,
		"transient services must return different instances per resolution (AD-14)")
}

// --- 2.2-RT-006 (AC1, P0) ---------------------------------------------------

// TestRT_TransientFromScope_AlsoDifferent: a transient resolved from a child
// scope returns a different instance from root AND from the same scope on
// repeat resolve (TransientServiceCanBeResolvedFromScope).
func TestRT_TransientFromScope_AlsoDifferent(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddTransient[iTransientSvc](b, func(godi.Provider) iTransientSvc {
		return newTransientSvc()
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	rootT := p.GetService(reflect.TypeFor[iTransientSvc]())

	scope := p.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })

	scopeT1 := scope.GetService(reflect.TypeFor[iTransientSvc]())
	scopeT2 := scope.GetService(reflect.TypeFor[iTransientSvc]())

	testutil.True(t, rootT != scopeT1, "root transient ≠ scope transient")
	testutil.True(t, rootT != scopeT2, "root transient ≠ scope transient")
	testutil.True(t, scopeT1 != scopeT2, "scope transient ≠ scope transient (AD-14)")
}

// --- 2.2-RT-007 (AC1, P1) ---------------------------------------------------

// TestRT_SingletonSharedAcrossScopes_Regression: singletons must still be
// shared across root and scope — a regression guard that the new scoped/
// transient support does not break AD-1.
func TestRT_SingletonSharedAcrossScopes_Regression(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*logger](b, func(godi.Provider) *logger { return &logger{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	lt := reflect.TypeFor[*logger]()
	rootInstance, _ := p.GetService(lt).(*logger)

	scope := p.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })

	inScope, _ := scope.GetService(lt).(*logger)
	testutil.Equal(t, inScope, rootInstance)
}

// --- 2.2-RT-008 (AC2, P0) ---------------------------------------------------

// TestRT_TransientDependsOnScoped_SameScopeReusesScoped: AC2 core — a
// transient depending on a scoped service. Within the same scope, two
// transient resolves yield DIFFERENT transient instances but the SAME scoped
// dependency (FactoryServicesAreCreatedAsPartOfCreatingObjectGraph).
func TestRT_TransientDependsOnScoped_SameScopeReusesScoped(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[iScopedSvc](b, func(godi.Provider) iScopedSvc {
		return newScopedSvc()
	})
	runtime.AddTransient[iTransientWithScopedDep](b, func(p godi.Provider) iTransientWithScopedDep {
		dep := p.GetService(reflect.TypeFor[iScopedSvc]()).(iScopedSvc)
		return &transientWithScopedDep{dep: dep}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	tt := reflect.TypeFor[iTransientWithScopedDep]()

	// Resolve from root (root acts as root scope)
	t1 := p.GetService(tt).(iTransientWithScopedDep)
	t2 := p.GetService(tt).(iTransientWithScopedDep)

	testutil.True(t, t1 != t2, "two transients must be different instances")
	testutil.True(t, t1.ScopedDep() == t2.ScopedDep(),
		"AC2: both transients must share the SAME scoped dependency within root scope")

	// Resolve from a child scope — different scoped dep than root
	scope := p.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })

	t3 := scope.GetService(tt).(iTransientWithScopedDep)
	testutil.True(t, t3.ScopedDep() != t1.ScopedDep(),
		"scope-level transient must use scope-level scoped dep, not root-level")
}

// --- 2.2-RT-009 (AC1, P1) ---------------------------------------------------

// TestRT_IsServiceForAllLifetimes: IsService must return true for all three
// lifetime types when registered.
func TestRT_IsServiceForAllLifetimes(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*logger](b, func(godi.Provider) *logger { return &logger{} })
	runtime.AddScoped[iScopedSvc](b, func(godi.Provider) iScopedSvc { return newScopedSvc() })
	runtime.AddTransient[iTransientSvc](b, func(godi.Provider) iTransientSvc { return newTransientSvc() })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	testutil.True(t, p.IsService(reflect.TypeFor[*logger]()), "singleton must be IsService=true")
	testutil.True(t, p.IsService(reflect.TypeFor[iScopedSvc]()), "scoped must be IsService=true")
	testutil.True(t, p.IsService(reflect.TypeFor[iTransientSvc]()), "transient must be IsService=true")
	testutil.True(t, !p.IsService(reflect.TypeFor[int]()), "unregistered must be IsService=false")
}

// --- 2.2-RT-010 (AC1, P1) ---------------------------------------------------

// TestRT_ScopeIsService_MirrorsRoot: scope.IsService must agree with
// root.IsService for all lifetime types.
func TestRT_ScopeIsService_MirrorsRoot(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*logger](b, func(godi.Provider) *logger { return &logger{} })
	runtime.AddScoped[iScopedSvc](b, func(godi.Provider) iScopedSvc { return newScopedSvc() })
	runtime.AddTransient[iTransientSvc](b, func(godi.Provider) iTransientSvc { return newTransientSvc() })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	scope := p.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })

	testutil.True(t, scope.IsService(reflect.TypeFor[*logger]()), "scope: singleton IsService=true")
	testutil.True(t, scope.IsService(reflect.TypeFor[iScopedSvc]()), "scope: scoped IsService=true")
	testutil.True(t, scope.IsService(reflect.TypeFor[iTransientSvc]()), "scope: transient IsService=true")
	testutil.True(t, !scope.IsService(reflect.TypeFor[int]()), "scope: unregistered IsService=false")
}
