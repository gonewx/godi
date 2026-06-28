//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.4 (AC1-AC4/AC6, Task 2) — runtime
// provider: multi-registration append semantics, GetService last-wins,
// GetKeyedServices(t, nil) collection resolution, empty collection for
// unregistered types, OuterService slice dependency injection.
//
// These tests use EXISTING runtime.NewBuilder / AddSingleton / AddTransient
// API with LOCAL helper types. The "red" comes from:
//   - Multiple Add{Lifetime}[T] calls currently OVERWRITE (map semantics);
//     Story 2.4 changes them to APPEND (slice semantics)
//   - GetKeyedServices(t, nil) currently returns nil; Story 2.4 changes it
//     to return all unkeyed registrations for type t
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-4-ienumerable-resolution.md.
package runtime_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// --- local helper types for multi-impl tests (package-local, not fakes) ------

type iMulti interface{ multiSeal() }

type multiOne struct{ _ int }

func (*multiOne) multiSeal() {}

type multiTwo struct{ _ int }

func (*multiTwo) multiSeal() {}

type iSingleSvc interface{ singleSeal() }

type singleSvc struct{ _ int }

func (*singleSvc) singleSeal() {}

type iOuterSvc interface {
	outerSeal()
	Single() iSingleSvc
	Multi() []iMulti
}

type outerSvc struct {
	single iSingleSvc
	multi  []iMulti
}

func (*outerSvc) outerSeal()          {}
func (o *outerSvc) Single() iSingleSvc { return o.single }
func (o *outerSvc) Multi() []iMulti    { return o.multi }

// --- 2.4-PROV-001 (AC1, P0) — MultipleServiceCanBeIEnumerableResolved -------

// TestRT_MultiImpl_CollectionResolution: two transients registered under the
// same type → GetKeyedServices(t, nil) returns both instances in a []any.
// Currently returns nil (red).
func TestRT_MultiImpl_CollectionResolution(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiOne{}
	})
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiTwo{}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	all := p.GetKeyedServices(reflect.TypeFor[iMulti](), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 2)
}

// --- 2.4-PROV-002 (AC1, P0) — RegistrationOrderIsPreserved -----------------

// TestRT_MultiImpl_RegistrationOrder: collection resolution must return
// instances in registration order (first registered = first element). Uses
// dynamic type names as provider-stable observables (R4).
func TestRT_MultiImpl_RegistrationOrder(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiOne{}
	})
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiTwo{}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	all := p.GetKeyedServices(reflect.TypeFor[iMulti](), nil)
	testutil.Equal(t, len(all), 2)
	testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*runtime_test.multiOne")
	testutil.Equal(t, reflect.TypeOf(all[1]).String(), "*runtime_test.multiTwo")
}

// --- 2.4-PROV-003 (AC2, P0) — LastServiceReplacesPreviousServices -----------

// TestRT_MultiImpl_GetService_LastWins: GetService returns the LAST registered
// implementation when multiple implementations share one type. Currently the
// map-overwrite happens to do this, but after switching to slice-append the
// last-wins semantics must be explicitly preserved (instances[len-1]).
func TestRT_MultiImpl_GetService_LastWins(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiOne{}
	})
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiTwo{}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	got := p.GetService(reflect.TypeFor[iMulti]())
	testutil.NotNil(t, got)
	testutil.Equal(t, reflect.TypeOf(got).String(), "*runtime_test.multiTwo")
}

// --- 2.4-PROV-004 (AC3, P1) — SingleServiceCanBeIEnumerableResolved ---------

// TestRT_MultiImpl_SingleRegistration_Collection: a type with only ONE
// registration → GetKeyedServices(t, nil) returns a single-element []any.
// Currently returns nil (red).
func TestRT_MultiImpl_SingleRegistration_Collection(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iSingleSvc](b, func(godi.Provider) iSingleSvc {
		return &singleSvc{}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	all := p.GetKeyedServices(reflect.TypeFor[iSingleSvc](), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 1)
	testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*runtime_test.singleSvc")
}

// --- 2.4-PROV-005 (AC4, P1) — NonexistentServiceCanBeIEnumerableResolved ----

// TestRT_MultiImpl_Unregistered_EmptyCollection: GetKeyedServices for an
// unregistered type must return an EMPTY slice (not nil). dotnet returns
// Enumerable.Empty<T>() (DISpec line 801). Currently returns nil (red).
func TestRT_MultiImpl_Unregistered_EmptyCollection(t *testing.T) {
	b := runtime.NewBuilder()
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	all := p.GetKeyedServices(reflect.TypeFor[iMulti](), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 0)
}

// --- 2.4-PROV-006 (AC6, P1) — OuterServiceCanHaveOtherServicesInjected ------

// TestRT_MultiImpl_SliceDependencyInjection: an outer service's factory
// receives both a single service and a collected []iMulti slice. This tests
// the runtime's ability to provide slice dependencies — the factory manually
// collects via GetKeyedServices (the Go equivalent of dotnet's IEnumerable<T>
// constructor injection).
func TestRT_MultiImpl_SliceDependencyInjection(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iSingleSvc](b, func(godi.Provider) iSingleSvc {
		return &singleSvc{}
	})
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiOne{}
	})
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiTwo{}
	})
	runtime.AddTransient[iOuterSvc](b, func(p godi.Provider) iOuterSvc {
		single := p.GetService(reflect.TypeFor[iSingleSvc]()).(iSingleSvc)
		allAny := p.GetKeyedServices(reflect.TypeFor[iMulti](), nil)
		multi := make([]iMulti, len(allAny))
		for i, v := range allAny {
			multi[i] = v.(iMulti)
		}
		return &outerSvc{single: single, multi: multi}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	got := p.GetService(reflect.TypeFor[iOuterSvc]())
	testutil.NotNil(t, got)

	outer := got.(iOuterSvc)
	testutil.NotNil(t, outer.Single())
	testutil.Equal(t, len(outer.Multi()), 2)
	testutil.Equal(t, reflect.TypeOf(outer.Multi()[0]).String(), "*runtime_test.multiOne")
	testutil.Equal(t, reflect.TypeOf(outer.Multi()[1]).String(), "*runtime_test.multiTwo")
}

// --- 2.4-PROV-007 (AC1, P1) — Transient multi-impl each-resolve-new --------

// TestRT_MultiImpl_Transient_EachResolveCreatesNew: each GetKeyedServices call
// for transient multi-registrations creates NEW instances — transients are
// never cached (AD-14, D-6).
func TestRT_MultiImpl_Transient_EachResolveCreatesNew(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiOne{}
	})
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiTwo{}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	first := p.GetKeyedServices(reflect.TypeFor[iMulti](), nil)
	second := p.GetKeyedServices(reflect.TypeFor[iMulti](), nil)
	testutil.Equal(t, len(first), 2)
	testutil.Equal(t, len(second), 2)

	// Each collection has fresh instances (transient = never cached)
	testutil.True(t, first[0] != second[0],
		"transient multi-impl first element must be fresh on each resolve")
	testutil.True(t, first[1] != second[1],
		"transient multi-impl second element must be fresh on each resolve")
}

// --- 2.4-PROV-008 (AC1, P1) — Scope collection resolution ------------------

// TestRT_MultiImpl_Scope_CollectionResolution: collection resolution from a
// child scope must also work — scope.GetKeyedServices(t, nil) returns all
// instances, respecting lifetime semantics.
func TestRT_MultiImpl_Scope_CollectionResolution(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiOne{}
	})
	runtime.AddTransient[iMulti](b, func(godi.Provider) iMulti {
		return &multiTwo{}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	scope := p.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })

	all := scope.GetKeyedServices(reflect.TypeFor[iMulti](), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 2)
	testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*runtime_test.multiOne")
	testutil.Equal(t, reflect.TypeOf(all[1]).String(), "*runtime_test.multiTwo")
}
