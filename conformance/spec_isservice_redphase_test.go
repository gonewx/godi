//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.5 (AC1-AC8, Task 4) — conformance suite:
// dual-provider IsService SpecCases + Get[T]/GetRequired[T] verification +
// closed-generic IsService + Go difference annotations + cross-provider
// differential equivalence for the IsService axis.
//
// This file extends the conformance suite with IsService-specific SpecCases
// that run against BOTH providers (runtime oracle + codegen B2). AD-15 mandates
// that the IsService axis be covered in codegen with ≥1 scenario + cross-
// provider differential equivalence (覆盖义务).
//
// New fakes needed (Task 2): IFakeOpenGenericService[T] / FakeOpenGenericService[T]
// in conformance/fakes/fakes_opengeneric.go.
//
// The "red" comes from:
//   - container_gen.go does NOT have typed accessor methods
//   - container_gen.go does NOT have closed generic type registration
//   - rtFactoryIsService registers closed generic → works IF runtime supports it
//   - Typed accessor test only runs for codegen provider (runtime has no accessors)
//
// Test standard R4: pointer identity within one provider only; across
// providers use provider-stable observables (IsService booleans, type names).
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-5-typed-resolve-isservice.md.
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

// rtFactoryIsService builds the runtime provider with all services needed for
// IsService + Get[T] + closed generic tests. Includes the existing core
// services PLUS a closed generic transient (AC7).
var rtFactoryIsService = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		// Core singleton (for IsService true/false, Get[T])
		runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
			return fakes.NewFakeService()
		})
		// Closed generic transient (AC7: ClosedGenericsWithIsService)
		runtime.AddTransient[fakes.IFakeOpenGenericService[fakes.IFakeService]](b, func(godi.Provider) fakes.IFakeOpenGenericService[fakes.IFakeService] {
			return fakes.NewFakeOpenGenericService[fakes.IFakeService]()
		})
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryIsService builds the codegen provider from the regenerated
// container_gen.go with closed generic + typed accessor registrations.
var cgFactoryIsService = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// isServiceSpecCases are SpecCases for IsService + Get[T] verification.
// Each case runs against BOTH providers unless noted otherwise.
var isServiceSpecCases = []SpecCase{
	// --- 2.5-SPEC-001 (AC4, P0) — ExplicitServiceRegisterationWithIsService ----
	{
		Name: "isservice: registered interface=true, impl type=false (ExplicitServiceRegisterationWithIsService, IsServiceSpec:17)",
		Run: func(t *testing.T, p Provider) {
			// Registered interface type → true
			testutil.True(t, p.IsService(reflect.TypeFor[fakes.IFakeService]()),
				"IsService(IFakeService) must be true for registered interface")
			// Implementation (concrete) type → false
			testutil.True(t, !p.IsService(reflect.TypeFor[*fakes.FakeService]()),
				"IsService(*FakeService) must be false — only service type, not impl (D-5)")
		},
	},
	// --- 2.5-SPEC-002 (AC1, P0) — GetTReturnsValueForRegisteredService ---------
	{
		Name: "isservice: Get[T] resolves registered service (AC1 supplement)",
		Run: func(t *testing.T, p Provider) {
			svc, ok := godi.Get[fakes.IFakeService](p)
			testutil.True(t, ok, "Get[IFakeService] must return ok=true for registered service")
			testutil.NotNil(t, svc)
			testutil.Equal(t, reflect.TypeOf(svc).String(), "*fakes.FakeService")
		},
	},
	// --- 2.5-SPEC-003 (AC2, P0) — AttemptingToResolveNonexistentServiceReturnsNil
	{
		Name: "isservice: Get[T] unregistered returns (zero, false) (AttemptingToResolveNonexistentServiceReturnsNull, DISpec:787)",
		Run: func(t *testing.T, p Provider) {
			type unregistered interface{ notRegistered() }
			val, ok := godi.Get[unregistered](p)
			testutil.True(t, !ok, "Get[T] for unregistered must return ok=false")
			testutil.Nil(t, val)
		},
	},
	// --- 2.5-SPEC-004 (AC3, P0) — GetRequiredPanicsForUnregisteredType ---------
	{
		Name: "isservice: GetRequired[T] panics for unregistered type (AC3)",
		Run: func(t *testing.T, p Provider) {
			type unregistered interface{ notRegistered() }
			testutil.Panics(t, func() {
				godi.GetRequired[unregistered](p)
			})
		},
	},
	// --- 2.5-SPEC-005 (AC7, P0) — IsServiceForClosedGenericType ----------------
	{
		Name: "isservice: closed generic type IsService=true (ClosedGenericsWithIsService, IsServiceSpec:60)",
		Run: func(t *testing.T, p Provider) {
			// Closed generic type is an ordinary registered type in godi (D-3)
			testutil.True(t,
				p.IsService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]()),
				"IsService(IFakeOpenGenericService[IFakeService]) must be true — closed generic = ordinary registered type")
		},
	},
	// --- 2.5-SPEC-006 (AC5, P1) — IEnumerableWithIsServiceGoAdaptation ---------
	{
		Name: "isservice: Go adaptation — IsService covers GetService face only, not collection (IEnumerableWithIsServiceAlwaysReturnsTrue, IsServiceSpec:82)",
		Run: func(t *testing.T, p Provider) {
			// Go difference (D-1): godi's IsService answers "can GetService(t) return
			// non-nil?" — it does NOT cover collection resolution (GetKeyedServices).
			// dotnet's IEnumerable<T> is special (always true); godi has no such
			// implicit mapping.
			//
			// Adapted assertion: IsService(element type) is true for registered types
			// (verifying IsService itself works correctly for GetService-resolvable types).
			testutil.True(t, p.IsService(reflect.TypeFor[fakes.IFakeService]()),
				"Go adaptation: IsService(registered element type) must be true")
			// Unregistered type — IsService false (no implicit IEnumerable mapping)
			type unregistered interface{ notRegistered() }
			testutil.True(t, !p.IsService(reflect.TypeFor[unregistered]()),
				"Go adaptation: IsService(unregistered type) must be false — no IEnumerable<T> implicit mapping")
		},
	},
	// --- 2.5-SPEC-007 (AC6, P1) — BuiltInServicesWithIsServiceGoAdaptation -----
	{
		Name: "isservice: Go adaptation — Provider not auto-registered as service (BuiltInServicesWithIsServiceReturnsTrue, IsServiceSpec:104)",
		Run: func(t *testing.T, p Provider) {
			// Go difference (D-2): godi does NOT auto-register Provider/Scope as
			// resolvable services (unlike dotnet's IServiceProvider / IServiceScopeFactory
			// / IServiceProviderIsService). Provider is the resolver, not a resolved
			// service. Provider self-resolve deferred to Epic 6.
			//
			// Adapted assertion: verify Provider interface method face completeness
			// (the contract IS the interface, not self-registration).
			testutil.True(t, p.IsService(reflect.TypeFor[fakes.IFakeService]()),
				"Go adaptation: Provider resolves registered services correctly")
			testutil.True(t, !p.IsService(reflect.TypeFor[godi.Provider]()),
				"Go adaptation: Provider interface itself is NOT a registered service (deferred to Epic 6)")
		},
	},
}

// --- 2.5-SPEC-RUN (AC1-AC8, P0) ---------------------------------------------

func TestConformanceIsService_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryIsService, isServiceSpecCases)
}

func TestConformanceIsService_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryIsService, isServiceSpecCases)
}

// --- 2.5-SPEC-008 (AC1, P0) — Typed Accessor (codegen-only) ------------------

// TestConformanceIsService_TypedAccessor_CodegenOnly: typed accessors only exist
// on the codegen container (runtime provider is interface-based, no typed
// accessors). This test directly verifies that the generated container's typed
// accessor method returns the correct instance (AC1).
//
// Uses interface type assertion to avoid compile-level red: the assertion
// fails at runtime if the typed accessor method doesn't exist yet.
func TestConformanceIsService_TypedAccessor_CodegenOnly(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	// Type assertion: codegen.Root should have FakeService() fakes.IFakeService
	type fakeServiceAccessor interface {
		FakeService() fakes.IFakeService
	}
	accessor, ok := any(root).(fakeServiceAccessor)
	testutil.True(t, ok,
		"codegen.Root must implement typed accessor FakeService() — AC1: interface type resolved via generated accessor, not any(zero).(type)")
	if ok {
		svc := accessor.FakeService()
		testutil.NotNil(t, svc)
		testutil.Equal(t, reflect.TypeOf(svc).String(), "*fakes.FakeService")
	}
}

// --- 2.5-SPEC-009 (AC8, P0) — Differential Equivalence ----------------------

// TestConformanceIsService_DifferentialEquivalence: AD-15 cross-provider
// differential equivalence for the IsService axis. Uses provider-stable
// observables (IsService booleans, Get[T] ok booleans) — never pointer
// identity across providers (R4).
func TestConformanceIsService_DifferentialEquivalence(t *testing.T) {
	// IsService for registered interface
	AssertEquivalent(t, rtFactoryIsService, cgFactoryIsService, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeService]())
	})
	// IsService for implementation type (must be false for both)
	AssertEquivalent(t, rtFactoryIsService, cgFactoryIsService, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[*fakes.FakeService]())
	})
	// IsService for unregistered type
	AssertEquivalent(t, rtFactoryIsService, cgFactoryIsService, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[int]())
	})
	// IsService for closed generic type (AC7)
	AssertEquivalent(t, rtFactoryIsService, cgFactoryIsService, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]())
	})
	// Get[T] ok consistency
	AssertEquivalent(t, rtFactoryIsService, cgFactoryIsService, func(p Provider) bool {
		_, ok := godi.Get[fakes.IFakeService](p)
		return ok
	})
	// Get[T] miss consistency
	AssertEquivalent(t, rtFactoryIsService, cgFactoryIsService, func(p Provider) bool {
		type unregistered interface{ notRegistered() }
		_, ok := godi.Get[unregistered](p)
		return ok
	})
	// Dynamic type name for resolved service
	AssertEquivalent(t, rtFactoryIsService, cgFactoryIsService, func(p Provider) string {
		return dynTypeName(p.GetService(reflect.TypeFor[fakes.IFakeService]()))
	})
	// Dynamic type name for closed generic
	AssertEquivalent(t, rtFactoryIsService, cgFactoryIsService, func(p Provider) string {
		return dynTypeName(p.GetService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]()))
	})
}
