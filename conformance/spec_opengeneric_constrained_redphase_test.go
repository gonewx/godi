//go:build atdd_redphase

// ATDD scaffold for Story 4.2 (AC1-AC5, Task 3) — conformance suite:
// constrained open-generic CLOSED materialization across BOTH providers
// (runtime oracle + codegen B2) with multi-implementation collection resolution,
// Go compile-time constraint guarantee, and cross-provider differential
// equivalence (AD-14 / AD-15 / AD-4 / AD-10).
//
// Ported from dotnet DependencyInjectionSpecificationTests (v10.0.9):
//
//	ConstrainedOpenGenericServicesCanBeResolved (:665, touch-point 2/9)       → AC2 (4.2-SPEC-001)
//	(non-satisfying count=1, touch-point 2 reverse)                          → AC2 (4.2-SPEC-002)
//	ConstrainedOpenGenericServicesReturnsEmptyWithNoMatches (:688, t-p 3/9)   → AC3 (4.2-SPEC-003)
//	InterfaceConstrainedOpenGenericServicesCanBeResolved (:702, t-p 4/9)      → AC2 (4.2-SPEC-004)
//	AbstractClassConstrainedOpenGenericServicesCanBeResolved (:725, t-p 5/9)  → AC2 (4.2-SPEC-005)
//	(AD-15 cross-provider differential equivalence)                          → AC5 (4.2-SPEC-DIFF)
//
// Go difference (AD-4/AD-10): dotnet registers open generic with constraint
// and checks constraint satisfaction at RUNTIME (MakeGenericType + constraint
// reflection); godi's per-service helper carries the Go type constraint, so the
// Go compiler rejects non-satisfying types at COMPILE TIME — same observable
// result (non-satisfying types are unresolvable), different mechanism.
//
// RED PHASE: excluded from the default build by the atdd_redphase tag until
// Task 1 (fakes) and Task 2 (registration + regen) are complete. Delete the
// build tag and the blank line below it to activate.
//
// Test standard R4: pointer identity within ONE provider only; across providers
// use provider-stable observables (counts, dynamic type names).
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

// --- Runtime-side constrained generic closure helpers (AD-14 same closed set) --

// addConstrainedOpenGenericRT is the runtime counterpart of the codegen
// per-service helper AddConstrainedOpenGenericService[T PocoClassLike] — both
// providers enumerate the SAME logical closed set (AD-14, never hand-forked).
func addConstrainedOpenGenericRT[T fakes.PocoClassLike](b *runtime.Builder) {
	runtime.AddTransient[fakes.IFakeOpenGenericService[T]](b, func(p godi.Provider) fakes.IFakeOpenGenericService[T] {
		dep := p.GetService(reflect.TypeFor[T]()).(T)
		return fakes.NewConstrainedFakeOpenGenericService[T](dep)
	})
}

// addInterfaceConstrainedRT mirrors AddInterfaceConstrainedService[T IterableLike].
func addInterfaceConstrainedRT[T fakes.IterableLike](b *runtime.Builder) {
	runtime.AddTransient[fakes.IFakeOpenGenericService[T]](b, func(p godi.Provider) fakes.IFakeOpenGenericService[T] {
		dep := p.GetService(reflect.TypeFor[T]()).(T)
		return fakes.NewClassWithInterfaceConstraint[T](dep)
	})
}

// addAbstractConstrainedRT mirrors AddAbstractConstrainedService[T AbstractClassLike].
func addAbstractConstrainedRT[T fakes.AbstractClassLike](b *runtime.Builder) {
	runtime.AddTransient[fakes.IFakeOpenGenericService[T]](b, func(p godi.Provider) fakes.IFakeOpenGenericService[T] {
		dep := p.GetService(reflect.TypeFor[T]()).(T)
		return fakes.NewClassWithAbstractClassConstraint[T](dep)
	})
}

// rtFactoryConstrainedOpenGeneric builds the runtime oracle: 3 candidate type
// singletons + IFakeService singleton (non-satisfying baseline) + unconstrained
// open generic for all 4 types + constrained open generic for the 3 satisfying
// types only (IFakeService CANNOT be passed to constrained helpers — Go compiler
// rejects at compile time, AC1).
var rtFactoryConstrainedOpenGeneric = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()

		// Candidate type singletons (the injected T values for open generics).
		runtime.AddSingleton[fakes.IPocoClass](b, func(godi.Provider) fakes.IPocoClass {
			return fakes.NewPocoClass()
		})
		runtime.AddSingleton[fakes.IClassImplementingIterable](b, func(godi.Provider) fakes.IClassImplementingIterable {
			return fakes.NewClassImplementingIterable()
		})
		runtime.AddSingleton[fakes.IClassInheritingAbstract](b, func(godi.Provider) fakes.IClassInheritingAbstract {
			return fakes.NewClassInheritingAbstract()
		})
		// IFakeService singleton: non-satisfying baseline — does NOT satisfy
		// PocoClassLike / IterableLike / AbstractClassLike.
		runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
			return fakes.NewFakeService()
		})

		// Unconstrained open generic for ALL candidate types + IFakeService.
		addOpenGenericRT[fakes.IPocoClass](b)
		addOpenGenericRT[fakes.IClassImplementingIterable](b)
		addOpenGenericRT[fakes.IClassInheritingAbstract](b)
		addOpenGenericRT[fakes.IFakeService](b)

		// Constrained open generic for ONLY satisfying candidate types.
		// IFakeService CANNOT be passed — Go compiler rejects (AC1).
		addConstrainedOpenGenericRT[fakes.IPocoClass](b)
		addInterfaceConstrainedRT[fakes.IClassImplementingIterable](b)
		addAbstractConstrainedRT[fakes.IClassInheritingAbstract](b)

		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryConstrainedOpenGeneric builds the codegen provider from the
// regenerated container_gen.go (Story 4.2): the committed container includes
// the 3 candidate singletons + unconstrained + constrained registrations.
var cgFactoryConstrainedOpenGeneric = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// Closed generic type helpers for readability in assertions below.
func openGenericOfPocoClassType() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IPocoClass]]()
}
func openGenericOfIterableType() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IClassImplementingIterable]]()
}
func openGenericOfAbstractType() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IClassInheritingAbstract]]()
}

// constrainedOpenGenericSpecCases run against BOTH providers via RunSuite (AC5).
var constrainedOpenGenericSpecCases = []SpecCase{
	// --- 4.2-SPEC-001 (AC2, P0, touch-point 2) — ConstrainedOpenGenericServicesCanBeResolved
	{
		Name: "constrained-opengeneric: satisfying [IPocoClass] returns 2 impls (unconstrained+constrained), both .Value()==singleton (ConstrainedOpenGenericServicesCanBeResolved, DISpec:665)",
		Run: func(t *testing.T, p Provider) {
			all := p.GetKeyedServices(openGenericOfPocoClassType(), nil)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 2)
			if len(all) != 2 {
				return
			}

			// Both implementations' .Value() must be the SAME singleton the
			// provider resolves directly — dependency injection proof (AC2).
			// Pointer identity within one provider is legal (R4).
			singleton := p.GetService(reflect.TypeFor[fakes.IPocoClass]()).(fakes.IPocoClass)
			testutil.NotNil(t, singleton)
			for _, svc := range all {
				gen := svc.(fakes.IFakeOpenGenericService[fakes.IPocoClass])
				testutil.NotNil(t, gen.Value())
				testutil.Equal(t, gen.Value(), singleton)
			}
		},
	},
	// --- 4.2-SPEC-002 (AC2 reverse, P0, touch-point 2) — Non-satisfying type count=1
	{
		Name: "constrained-opengeneric: non-satisfying [IFakeService] returns 1 impl (unconstrained only) — Go compile-time constraint filtering (DISpec:665 reverse)",
		Run: func(t *testing.T, p Provider) {
			// IFakeService does NOT satisfy PocoClassLike. In godi, the
			// constrained helper CANNOT accept IFakeService (compile error,
			// AC1) — so only the unconstrained FakeOpenGenericService is
			// registered. Go difference (AD-4/AD-10): dotnet filters at
			// runtime; godi filters at compile time. Same result: count=1.
			all := p.GetKeyedServices(openGenericOfServiceType(), nil)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 1)
			if len(all) != 1 {
				return
			}

			// The single element is the unconstrained FakeOpenGenericService,
			// its .Value() is the IFakeService singleton.
			gen := all[0].(fakes.IFakeOpenGenericService[fakes.IFakeService])
			direct := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
			testutil.Equal(t, gen.Value(), direct)
		},
	},
	// --- 4.2-SPEC-003 (AC3, P0, touch-point 3) — ConstrainedOpenGenericServicesReturnsEmptyWithNoMatches
	{
		Name: "constrained-opengeneric: unregistered closed type returns nil — constraint blocks + no unconstrained (ConstrainedOpenGenericServicesReturnsEmptyWithNoMatches, DISpec:688)",
		Run: func(t *testing.T, p Provider) {
			// IFakeScopedService has NO open-generic registration (neither
			// unconstrained nor constrained) for IFakeOpenGenericService.
			// Go difference (AD-4/AD-10): dotnet registers the open generic
			// with a constraint, then at runtime the constraint check fails
			// and returns empty; godi's constraint prevents the registration
			// call from compiling — same result (count=0), different mechanism
			// (compile-time vs runtime). This is the core AD-4/AD-10 tradeoff
			// positive acceptance.
			all := p.GetKeyedServices(openGenericOfScopedType(), nil)
			testutil.Nil(t, all)
		},
	},
	// --- 4.2-SPEC-004 (AC2, P0, touch-point 4) — InterfaceConstrainedOpenGenericServicesCanBeResolved
	{
		Name: "constrained-opengeneric: interface constraint [IClassImplementingIterable] returns 2 impls (unconstrained+interface-constrained) (InterfaceConstrainedOpenGenericServicesCanBeResolved, DISpec:702)",
		Run: func(t *testing.T, p Provider) {
			all := p.GetKeyedServices(openGenericOfIterableType(), nil)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 2)
			if len(all) != 2 {
				return
			}

			singleton := p.GetService(reflect.TypeFor[fakes.IClassImplementingIterable]()).(fakes.IClassImplementingIterable)
			testutil.NotNil(t, singleton)
			for _, svc := range all {
				gen := svc.(fakes.IFakeOpenGenericService[fakes.IClassImplementingIterable])
				testutil.NotNil(t, gen.Value())
				testutil.Equal(t, gen.Value(), singleton)
			}
		},
	},
	// --- 4.2-SPEC-005 (AC2, P0, touch-point 5) — AbstractClassConstrainedOpenGenericServicesCanBeResolved
	{
		Name: "constrained-opengeneric: abstract-class constraint [IClassInheritingAbstract] returns 2 impls (unconstrained+abstract-constrained) (AbstractClassConstrainedOpenGenericServicesCanBeResolved, DISpec:725)",
		Run: func(t *testing.T, p Provider) {
			all := p.GetKeyedServices(openGenericOfAbstractType(), nil)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 2)
			if len(all) != 2 {
				return
			}

			singleton := p.GetService(reflect.TypeFor[fakes.IClassInheritingAbstract]()).(fakes.IClassInheritingAbstract)
			testutil.NotNil(t, singleton)
			for _, svc := range all {
				gen := svc.(fakes.IFakeOpenGenericService[fakes.IClassInheritingAbstract])
				testutil.NotNil(t, gen.Value())
				testutil.Equal(t, gen.Value(), singleton)
			}
		},
	},
}

// --- 4.2-SPEC-RUN (AC2-AC3, P0) ---------------------------------------------

func TestConformanceConstrainedOpenGeneric_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryConstrainedOpenGeneric, constrainedOpenGenericSpecCases)
}

func TestConformanceConstrainedOpenGeneric_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryConstrainedOpenGeneric, constrainedOpenGenericSpecCases)
}

// --- 4.2-SPEC-DIFF (AC5, P0) — Cross-provider differential equivalence ------

// TestConformanceConstrainedOpenGeneric_DifferentialEquivalence: AD-15
// cross-provider differential equivalence for the constrained open-generic
// materialization axis. Probes use provider-stable observables only — collection
// counts (int) and dynamic type names (string) — never pointer identity across
// providers (R4).
func TestConformanceConstrainedOpenGeneric_DifferentialEquivalence(t *testing.T) {
	rt := rtFactoryConstrainedOpenGeneric
	cg := cgFactoryConstrainedOpenGeneric

	// Collection count for satisfying type IPocoClass (2 both).
	AssertEquivalent(t, rt, cg, func(p Provider) int {
		all := p.GetKeyedServices(openGenericOfPocoClassType(), nil)
		if all == nil {
			return -1
		}
		return len(all)
	})
	// Collection count for non-satisfying type IFakeService (1 both).
	AssertEquivalent(t, rt, cg, func(p Provider) int {
		all := p.GetKeyedServices(openGenericOfServiceType(), nil)
		if all == nil {
			return -1
		}
		return len(all)
	})
	// Collection count for unregistered closed type (nil/-1 both — AC3).
	AssertEquivalent(t, rt, cg, func(p Provider) int {
		all := p.GetKeyedServices(openGenericOfScopedType(), nil)
		if all == nil {
			return -1
		}
		return len(all)
	})
	// Collection count for interface constraint (2 both).
	AssertEquivalent(t, rt, cg, func(p Provider) int {
		all := p.GetKeyedServices(openGenericOfIterableType(), nil)
		if all == nil {
			return -1
		}
		return len(all)
	})
	// Collection count for abstract constraint (2 both).
	AssertEquivalent(t, rt, cg, func(p Provider) int {
		all := p.GetKeyedServices(openGenericOfAbstractType(), nil)
		if all == nil {
			return -1
		}
		return len(all)
	})
	// Injected value dynamic type for satisfying type first element.
	AssertEquivalent(t, rt, cg, func(p Provider) string {
		all := p.GetKeyedServices(openGenericOfPocoClassType(), nil)
		if all == nil || len(all) == 0 {
			return "<nil>"
		}
		return dynTypeName(all[0].(fakes.IFakeOpenGenericService[fakes.IPocoClass]).Value())
	})
}
