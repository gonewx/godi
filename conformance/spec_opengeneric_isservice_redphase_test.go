// ATDD scaffold for Story 4.4 (AC1-AC4) — conformance suite:
// dual-provider open-generic IsService + IsKeyedService SpecCases +
// cross-provider differential equivalence for the 9th open-generic
// touchpoint (non-keyed) and keyed variant touchpoint 5.
//
// Core insight (D-2): godi's IsService does NOT distinguish "ordinary
// registered" from "open-generic materialized" — both are identical.
// Materialized closed types are ordinary registered types; unmaterialized
// closed types are simply not registered → false. These tests LOCK this
// behavior against future regressions.
//
// Go difference (D-1): dotnet's typeof(IFakeOpenGenericService<>) produces
// an open generic Type object. Go's reflect.TypeFor requires a concrete type
// argument — there is NO reflect.Type for an open generic definition. The
// dotnet assertion `IsService(typeof(T<>)) == true` has no Go equivalent;
// we substitute with an unmaterialized concrete closed type (int) to cover
// the same "not registered → false" semantic intent.
//
// Ported from dotnet:
//
//	ServiceProviderIsServiceSpecificationTests.OpenGenericsWithIsService (:39)
//	ServiceProviderIsKeyedServiceSpecificationTests.OpenGenericsWithIsKeyedService (:39)
package conformance

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
)

// ---------------------------------------------------------------------------
// Task 1: non-keyed open-generic IsService SpecCases (AC1/AC2)
// ---------------------------------------------------------------------------

var openGenericIsServiceSpecCases = []SpecCase{
	// --- 4.4-SPEC-001 (AC1, P0, touchpoint 9) ---
	// Materialized closed type is an ordinary registered type → IsService true.
	// Aligns with dotnet: Assert.True(IsService(typeof(IFakeOpenGenericService<IFakeService>)))
	{
		Name: "open-generic-isservice: materialized closed type IsService=true (OpenGenericsWithIsService, touchpoint 9)",
		Run: func(t *testing.T, p Provider) {
			// Go difference annotation: dotnet also asserts IsService(typeof(T<>))==true
			// because open-generic registration + runtime MakeGenericType can materialize
			// any closed type on demand. In godi, open generic types have no reflect.Type
			// representation (reflect.TypeFor requires concrete type args) — this branch
			// is syntactically impossible in Go. Materialized closed == ordinary registered.
			testutil.True(t,
				p.IsService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]()),
				"IsService(IFakeOpenGenericService[IFakeService]) must be true — materialized closed = ordinary registered type (AD-4/AD-10)")
		},
	},
	// --- 4.4-SPEC-002 (AC2, P0, touchpoint 9 negative) ---
	// Unmaterialized closed type (int never enumerated in registration) → false.
	// "Only query, never construct" (只查不造) — AD-4: no runtime materialization.
	// Aligns with dotnet semantic: typeof(IFakeOpenGenericService<>) returns false for
	// the open definition when no runtime MakeGenericType is available.
	{
		Name: "open-generic-isservice: unmaterialized closed type IsService=false, only-query-never-construct (OpenGenericsWithIsService negative, touchpoint 9)",
		Run: func(t *testing.T, p Provider) {
			// Go difference annotation: dotnet tests typeof(IFakeOpenGenericService<>)
			// (open generic definition) returning false when no open registration exists.
			// Go has no open generic reflect.Type; we use IFakeOpenGenericService[int]
			// where int satisfies the `any` constraint but was never enumerated/registered.
			// Same semantic: "not registered/not materialized → false".
			testutil.True(t,
				!p.IsService(reflect.TypeFor[fakes.IFakeOpenGenericService[int]]()),
				"IsService(IFakeOpenGenericService[int]) must be false — unmaterialized closed type, only-query-never-construct (AD-4)")
		},
	},
}

// --- 4.4-SPEC-RUN (AC1/AC2, P0) — dual-provider suite -----------------------

func TestConformanceOpenGenericIsService_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryIsService, openGenericIsServiceSpecCases)
}

func TestConformanceOpenGenericIsService_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryIsService, openGenericIsServiceSpecCases)
}

// ---------------------------------------------------------------------------
// Task 2: keyed open-generic IsKeyedService SpecCases (AC3)
// ---------------------------------------------------------------------------

var openGenericIsKeyedServiceSpecCases = []SpecCase{
	// --- 4.4-SPEC-003 (AC3, P0, keyed touchpoint 5) ---
	// Materialized closed type under correct key → IsKeyedService true.
	// Aligns with dotnet: Assert.True(IsKeyedService(typeof(IFakeOpenGenericService<IFakeService>), key))
	{
		Name: "open-generic-iskeyed: materialized closed type + correct key IsKeyedService=true (OpenGenericsWithIsKeyedService, keyed touchpoint 5)",
		Run: func(t *testing.T, p Provider) {
			// Go difference annotation: dotnet also passes typeof(T<>) (open definition)
			// with a key, returning true when an open-generic keyed registration exists.
			// godi has no open generic reflect.Type — only concrete closed types are
			// queryable. Materialized closed + correct key == ordinary keyed registered.
			testutil.True(t,
				p.IsKeyedService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]](), "gen-key"),
				"IsKeyedService(IFakeOpenGenericService[IFakeService], \"gen-key\") must be true — materialized closed keyed = ordinary keyed registration")
		},
	},
	// --- 4.4-SPEC-004 (AC3 negative, P0) ---
	// Materialized closed type under WRONG key → false.
	// Aligns with dotnet: Assert.False(IsKeyedService(typeof(T<>), new object()))
	// adapted: concrete closed type + wrong key covers same "not registered under
	// this key → false" semantic.
	{
		Name: "open-generic-iskeyed: materialized closed type + wrong key IsKeyedService=false (AC3 negative)",
		Run: func(t *testing.T, p Provider) {
			// Go difference annotation: dotnet uses typeof(IFakeOpenGenericService<>)
			// with `new object()` (a key that was never registered). We use the
			// materialized closed type with "wrong-key" — same semantic: no registration
			// exists under this (type, key) pair → false.
			testutil.True(t,
				!p.IsKeyedService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]](), "wrong-key"),
				"IsKeyedService(IFakeOpenGenericService[IFakeService], \"wrong-key\") must be false — wrong key")
		},
	},
	// --- 4.4-SPEC-005 (AC3 negative, P0) ---
	// Unmaterialized closed type under any key → false.
	// Neither the type nor the key match any registration.
	{
		Name: "open-generic-iskeyed: unmaterialized closed type + any key IsKeyedService=false (AC3 negative)",
		Run: func(t *testing.T, p Provider) {
			// IFakeOpenGenericService[int] was never registered under any key.
			testutil.True(t,
				!p.IsKeyedService(reflect.TypeFor[fakes.IFakeOpenGenericService[int]](), "gen-key"),
				"IsKeyedService(IFakeOpenGenericService[int], \"gen-key\") must be false — unmaterialized type, no registration exists")
		},
	},
}

// --- 4.4-SPEC-RUN (AC3, P0) — dual-provider suite ----------------------------

func TestConformanceOpenGenericIsKeyedService_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryKeyedIsService, openGenericIsKeyedServiceSpecCases)
}

func TestConformanceOpenGenericIsKeyedService_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryKeyedIsService, openGenericIsKeyedServiceSpecCases)
}

// ---------------------------------------------------------------------------
// AC4: cross-provider differential equivalence (AD-15 coverage obligation:
// IsService + open-generic closed materialization intersection)
// ---------------------------------------------------------------------------

func TestConformanceOpenGenericIsService_DifferentialEquivalence(t *testing.T) {
	// AC1: materialized closed type IsService == true in both providers.
	AssertEquivalent(t, rtFactoryIsService, cgFactoryIsService, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]())
	})
	// AC2: unmaterialized closed type IsService == false in both providers.
	AssertEquivalent(t, rtFactoryIsService, cgFactoryIsService, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeOpenGenericService[int]]())
	})
}

func TestConformanceOpenGenericIsKeyedService_DifferentialEquivalence(t *testing.T) {
	// AC3 positive: materialized closed + correct key == true in both.
	AssertEquivalent(t, rtFactoryKeyedIsService, cgFactoryKeyedIsService, func(p Provider) bool {
		return p.IsKeyedService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]](), "gen-key")
	})
	// AC3 negative: materialized closed + wrong key == false in both.
	AssertEquivalent(t, rtFactoryKeyedIsService, cgFactoryKeyedIsService, func(p Provider) bool {
		return p.IsKeyedService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]](), "wrong-key")
	})
	// AC3 negative: unmaterialized closed + any key == false in both.
	AssertEquivalent(t, rtFactoryKeyedIsService, cgFactoryKeyedIsService, func(p Provider) bool {
		return p.IsKeyedService(reflect.TypeFor[fakes.IFakeOpenGenericService[int]](), "gen-key")
	})
}
