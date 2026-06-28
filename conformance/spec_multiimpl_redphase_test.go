//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.4 (AC1-AC6, Task 6) — conformance suite:
// dual-provider multi-impl SpecCases + cross-provider differential equivalence
// for IEnumerable registration-order axis.
//
// This file extends the conformance suite with multi-implementation SpecCases
// that run against BOTH providers (runtime oracle + codegen B2). AD-15 mandates
// that IEnumerable registration order be covered in codegen with ≥1 scenario +
// cross-provider differential equivalence (覆盖义务).
//
// New fakes needed (Task 5): IFakeMultipleService, FakeOneMultipleService,
// FakeTwoMultipleService, IFakeOuterService, FakeOuterService — all in
// conformance/fakes/fakes_multiimpl.go. These already exist as stubs.
//
// The "red" comes from:
//   - runtime GetKeyedServices(t, nil) returns nil (no collection resolution)
//   - codegen container_gen.go has no multi-impl registrations
//   - rtFactoryMultiImpl registers two factories for the same type → current
//     map-overwrite loses the first registration
//
// Test standard R4: pointer identity within one provider only; across
// providers use provider-stable observables (type name sequences).
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-4-ienumerable-resolution.md.
package conformance

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// rtFactoryMultiImpl builds the runtime provider with multi-impl registrations.
// Two transient factories registered under IFakeMultipleService (currently the
// second overwrites the first — red for multi-registration). Plus IFakeService
// singleton and IFakeOuterService transient for AC6.
var rtFactoryMultiImpl = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		// Singleton IFakeService (for OuterService single dependency)
		runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
			return fakes.NewFakeService()
		})
		// Multi-impl: two transients under IFakeMultipleService (registration order)
		runtime.AddTransient[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
			return fakes.NewFakeOneMultipleService()
		})
		runtime.AddTransient[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
			return fakes.NewFakeTwoMultipleService()
		})
		// OuterService: depends on IFakeService + []IFakeMultipleService
		runtime.AddTransient[fakes.IFakeOuterService](b, func(p godi.Provider) fakes.IFakeOuterService {
			single := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
			allAny := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil)
			multi := make([]fakes.IFakeMultipleService, len(allAny))
			for i, v := range allAny {
				multi[i] = v.(fakes.IFakeMultipleService)
			}
			return fakes.NewFakeOuterService(single, multi)
		})
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryMultiImpl builds the codegen provider from the regenerated
// container_gen.go with multi-impl registrations. Will fail until container
// is regenerated with multi-impl support (red).
var cgFactoryMultiImpl = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// multiImplSpecCases covers AC1-AC6 for IEnumerable multi-implementation
// resolution. Each case runs against BOTH providers.
var multiImplSpecCases = []SpecCase{
	// --- 2.4-SPEC-001 (AC3, P1) — SingleServiceCanBeIEnumerableResolved -------
	{
		Name: "multi-impl: single registration can be collection-resolved (AC3, DISpec:199)",
		Run: func(t *testing.T, p Provider) {
			// IFakeService has a single registration — collection must return it
			all := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeService](), nil)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 1)
			testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*fakes.FakeService")
		},
	},
	// --- 2.4-SPEC-002 (AC1, P0) — MultipleServiceCanBeIEnumerableResolved -----
	{
		Name: "multi-impl: multiple registrations resolved as collection (AC1, DISpec:216)",
		Run: func(t *testing.T, p Provider) {
			all := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 2)
		},
	},
	// --- 2.4-SPEC-003 (AC1, P0) — RegistrationOrderIsPreserved ----------------
	{
		Name: "multi-impl: registration order preserved in collection (AC1, DISpec:234)",
		Run: func(t *testing.T, p Provider) {
			all := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil)
			testutil.Equal(t, len(all), 2)
			testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*fakes.FakeOneMultipleService")
			testutil.Equal(t, reflect.TypeOf(all[1]).String(), "*fakes.FakeTwoMultipleService")
		},
	},
	// --- 2.4-SPEC-004 (AC2, P0) — LastServiceReplacesPreviousServices ---------
	{
		Name: "multi-impl: GetService returns last-wins (AC2, DISpec:354)",
		Run: func(t *testing.T, p Provider) {
			got := p.GetService(reflect.TypeFor[fakes.IFakeMultipleService]())
			testutil.NotNil(t, got)
			testutil.Equal(t, reflect.TypeOf(got).String(), "*fakes.FakeTwoMultipleService")
		},
	},
	// --- 2.4-SPEC-005 (AC4, P1) — NonexistentServiceCanBeIEnumerableResolved --
	{
		Name: "multi-impl: unregistered type returns empty collection (AC4, DISpec:801)",
		Run: func(t *testing.T, p Provider) {
			// Use a type that is NOT registered
			all := p.GetKeyedServices(reflect.TypeFor[interface{ notRegistered() }](), nil)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 0)
		},
	},
	// --- 2.4-SPEC-006 (AC6, P1) — OuterServiceCanHaveOtherServicesInjected ----
	{
		Name: "multi-impl: OuterService injected with single + multi slice (AC6, DISpec:261)",
		Run: func(t *testing.T, p Provider) {
			got := p.GetService(reflect.TypeFor[fakes.IFakeOuterService]())
			testutil.NotNil(t, got)

			outer := got.(fakes.IFakeOuterService)

			// SingleService returns the injected IFakeService singleton
			testutil.NotNil(t, outer.SingleService())
			testutil.Equal(t, reflect.TypeOf(outer.SingleService()).String(), "*fakes.FakeService")

			// MultipleServices returns both in registration order
			multi := outer.MultipleServices()
			testutil.Equal(t, len(multi), 2)
			testutil.Equal(t, reflect.TypeOf(multi[0]).String(), "*fakes.FakeOneMultipleService")
			testutil.Equal(t, reflect.TypeOf(multi[1]).String(), "*fakes.FakeTwoMultipleService")
		},
	},
}

// --- 2.4-SPEC-RUN (AC1-AC6, P0) ---------------------------------------------

func TestConformanceMultiImpl_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryMultiImpl, multiImplSpecCases)
}

func TestConformanceMultiImpl_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryMultiImpl, multiImplSpecCases)
}

// --- 2.4-SPEC-007 (AC5, P0) — Differential Equivalence ----------------------

// TestConformanceMultiImpl_DifferentialEquivalence: AD-15 cross-provider
// differential equivalence for the IEnumerable registration-order axis. Uses
// type name sequences as provider-stable observables (R4).
func TestConformanceMultiImpl_DifferentialEquivalence(t *testing.T) {
	// IsService consistency for multi-impl types
	AssertEquivalent(t, rtFactoryMultiImpl, cgFactoryMultiImpl, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeMultipleService]())
	})
	AssertEquivalent(t, rtFactoryMultiImpl, cgFactoryMultiImpl, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeOuterService]())
	})

	// GetService last-wins dynamic type
	AssertEquivalent(t, rtFactoryMultiImpl, cgFactoryMultiImpl, func(p Provider) string {
		return dynTypeName(p.GetService(reflect.TypeFor[fakes.IFakeMultipleService]()))
	})

	// Collection size equivalence
	AssertEquivalent(t, rtFactoryMultiImpl, cgFactoryMultiImpl, func(p Provider) int {
		all := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil)
		if all == nil {
			return -1
		}
		return len(all)
	})

	// Single-registration collection size
	AssertEquivalent(t, rtFactoryMultiImpl, cgFactoryMultiImpl, func(p Provider) int {
		all := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeService](), nil)
		if all == nil {
			return -1
		}
		return len(all)
	})

	// Empty collection for unregistered
	AssertEquivalent(t, rtFactoryMultiImpl, cgFactoryMultiImpl, func(p Provider) int {
		all := p.GetKeyedServices(reflect.TypeFor[interface{ notRegistered() }](), nil)
		if all == nil {
			return -1
		}
		return len(all)
	})

	// OuterService injection: SingleService type name
	AssertEquivalent(t, rtFactoryMultiImpl, cgFactoryMultiImpl, func(p Provider) string {
		outer := p.GetService(reflect.TypeFor[fakes.IFakeOuterService]())
		if outer == nil {
			return "<nil>"
		}
		return dynTypeName(outer.(fakes.IFakeOuterService).SingleService())
	})

	// OuterService injection: MultipleServices count
	AssertEquivalent(t, rtFactoryMultiImpl, cgFactoryMultiImpl, func(p Provider) int {
		outer := p.GetService(reflect.TypeFor[fakes.IFakeOuterService]())
		if outer == nil {
			return -1
		}
		return len(outer.(fakes.IFakeOuterService).MultipleServices())
	})
}

// --- 2.4-SPEC-008 (AC1, P0) — Registration-order type-name sequence ----------

// TestConformanceMultiImpl_RegistrationOrderSequence: the type-name sequence
// from collection resolution must be identical across both providers. This is
// the strongest registration-order differential equivalence assertion (AD-15
// covers "IEnumerable 注册序" as a 覆盖义务 behavior axis).
func TestConformanceMultiImpl_RegistrationOrderSequence(t *testing.T) {
	var sequences [2]string
	for i, factory := range []ProviderFactory{rtFactoryMultiImpl, cgFactoryMultiImpl} {
		p := factory.Build(t)
		t.Cleanup(func() { _ = p.Close() })

		all := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil)
		var names []string
		for _, v := range all {
			names = append(names, reflect.TypeOf(v).String())
		}
		sequences[i] = join(names, ",")
	}
	testutil.Equal(t, sequences[0], sequences[1])
}

func join(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}
