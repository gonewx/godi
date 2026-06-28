// Conformance suite for Story 3.4 (AC1-AC6) — keyed IsService query
// (IsKeyedService) across BOTH providers (runtime oracle + codegen B2) with
// cross-provider differential equivalence (AD-15/AD-16/AD-17).
//
// Ported from dotnet ServiceProviderIsKeyedServiceSpecificationTests (v10.0.9):
//
//	ExplicitServiceRegistrationWithIsKeyedService (:17)  → AC1 (3.4-SPEC-001)
//	NonKeyedServiceWithIsKeyedService            (:108) → AC2 (3.4-SPEC-002)
//	(AnyKey pure-query, no dotnet 1:1)                  → AC3 (3.4-SPEC-003)
//	ClosedGenericsWithIsKeyedService             (:62)  → AC4 (3.4-SPEC-004)
//	IEnumerableWithIsKeyedServiceAlwaysReturnsTrue(:84) → AC5 (3.4-SPEC-005, Go diff)
//	OpenGenericsWithIsKeyedService               (:39)  → Epic 4 (deferred, not here)
//
// Go adaptation (AC1): godi exposes IsKeyedService as a PROVIDER METHOD, not a
// resolvable IServiceProviderIsKeyedService service (the Provider does not
// self-register, Story 2-5 D-2), so the test calls p.IsKeyedService(...) directly
// rather than GetService<IServiceProviderIsKeyedService>().IsKeyedService(...).
//
// AD-16 method face revised 6 → 7 in this story: IsKeyedService(t, key) bool is
// the keyed counterpart of IsService and a PURE QUERY (never panics). The
// AnyKey contrast is the dangerous boundary (D-3): GetKeyedService(AnyKey) panics,
// but IsKeyedService(AnyKey) returns false and must NOT panic.
//
// The codegen provider is the full container_gen.go (registration.go has the
// "service1" keyed IKeyedService, the unkeyed IFakeService, and the "gen-key"
// closed-generic keyed registration this story added). The runtime factory
// mirrors the probed subset from the SAME logical source (AD-14): no AnyKey is
// registered (rejected by both providers, Story 3-3 AC4).
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

// keyedISType is the keyed service interface under test (concrete-key + nil routing).
func keyedISType() reflect.Type { return reflect.TypeFor[fakes.IKeyedService]() }

// closedGenericKeyedType is the closed-generic keyed service type (AC4).
func closedGenericKeyedType() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]()
}

// rtFactoryKeyedIsService builds the runtime oracle for keyed IsService. It
// mirrors the probed subset of conformance/codegen/registration.go (AD-14, one
// logical source): a concrete-keyed IKeyedService under "service1", an unkeyed
// IFakeService (the nil-routing target for AC2), and a closed-generic keyed
// service under "gen-key" (AC4). No AnyKey registration — AnyKey-as-key is
// rejected (Story 3-3 AC4) and IsKeyedService(AnyKey) is a pure-query false.
var rtFactoryKeyedIsService = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()

		// Concrete keyed IKeyedService — AC1 hit target.
		runtime.AddKeyedSingleton[fakes.IKeyedService](b, "service1", func(godi.Provider) fakes.IKeyedService {
			return fakes.NewKeyedServiceOne()
		})
		// Unkeyed IFakeService — AC2 nil-routing target (IsKeyedService(t,nil)==IsService(t)).
		runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
			return fakes.NewFakeService()
		})
		// Closed-generic keyed — AC4 (closed generic is an ordinary registered type).
		runtime.AddKeyedSingleton[fakes.IFakeOpenGenericService[fakes.IFakeService]](b, "gen-key", func(godi.Provider) fakes.IFakeOpenGenericService[fakes.IFakeService] {
			return fakes.NewFakeOpenGenericService[fakes.IFakeService]()
		})

		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryKeyedIsService builds the codegen provider from the pre-generated B2
// container. registration.go already carries the "service1" keyed IKeyedService,
// the unkeyed IFakeService, and the "gen-key" closed-generic keyed registration.
var cgFactoryKeyedIsService = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// keyedIsServiceSpecCases run against BOTH providers via RunSuite.
var keyedIsServiceSpecCases = []SpecCase{
	// --- 3.4-SPEC-001 (AC1, P0) — registered (type,key) true, others false -----
	//
	// ExplicitServiceRegistrationWithIsKeyedService: the registered (IKeyedService,
	// "service1") pair reports true; the implementation type (*KeyedServiceOne) under
	// the same key reports false (registration keys on the interface, not the impl);
	// a stranger key reports false.
	{
		Name: "keyed-isservice: registered (type,key) true; impl-type and stranger key false (ExplicitServiceRegistrationWithIsKeyedService, AC1)",
		Run: func(t *testing.T, p Provider) {
			testutil.True(t, p.IsKeyedService(keyedISType(), "service1"),
				"registered (IKeyedService, \"service1\") must report true")
			// Implementation type under the registered key is NOT registered (keys on interface).
			testutil.True(t, !p.IsKeyedService(reflect.TypeFor[*fakes.KeyedServiceOne](), "service1"),
				"implementation type *KeyedServiceOne under \"service1\" must report false")
			// Stranger key on the registered interface type.
			testutil.True(t, !p.IsKeyedService(keyedISType(), "nonexistent"),
				"unregistered key must report false")
		},
	},
	// --- 3.4-SPEC-002 (AC2, P0) — nil key routes to unkeyed IsService -----------
	//
	// NonKeyedServiceWithIsKeyedService: a nil key routes to the unkeyed namespace,
	// so IsKeyedService(t, nil) == IsService(t) (null == unkeyed, AD-17). Both the
	// registered (IFakeService → true) and unregistered (int → false) branches hold.
	{
		Name: "keyed-isservice: nil key routes to unkeyed IsService (NonKeyedServiceWithIsKeyedService, AC2)",
		Run: func(t *testing.T, p Provider) {
			ft := reflect.TypeFor[fakes.IFakeService]()
			testutil.Equal(t, p.IsKeyedService(ft, nil), p.IsService(ft))
			testutil.True(t, p.IsKeyedService(ft, nil),
				"unkeyed IFakeService via nil key must report true")
			// Unregistered type via nil key routes to IsService → false.
			ut := reflect.TypeFor[int]()
			testutil.Equal(t, p.IsKeyedService(ut, nil), p.IsService(ut))
			testutil.True(t, !p.IsKeyedService(ut, nil),
				"unregistered type via nil key must report false")
		},
	},
	// --- 3.4-SPEC-003 (AC3, P0) — IsKeyedService(AnyKey) false, NEVER panics ----
	//
	// The dangerous boundary (D-3): GetKeyedService(AnyKey) panics, but
	// IsKeyedService(AnyKey) is a pure query — it returns false and must NOT panic.
	// AnyKey is never a registration key (both providers reject it), so an AnyKey
	// query can never hit. Holds on root AND scope; an unregistered type is also
	// false-not-panic.
	{
		Name: "keyed-isservice: IsKeyedService(AnyKey) returns false and never panics, root+scope (AC3)",
		Run: func(t *testing.T, p Provider) {
			// root: false, no panic (the call returning normally proves no panic).
			testutil.True(t, !p.IsKeyedService(keyedISType(), godi.AnyKey),
				"IsKeyedService(registered type, AnyKey) must be false")
			testutil.True(t, !p.IsKeyedService(reflect.TypeFor[int](), godi.AnyKey),
				"IsKeyedService(unregistered type, AnyKey) must be false")
			// scope parity: same false-not-panic from a child scope.
			sc := p.CreateScope()
			t.Cleanup(func() { _ = sc.Close() })
			testutil.True(t, !sc.IsKeyedService(keyedISType(), godi.AnyKey),
				"scope IsKeyedService(registered type, AnyKey) must be false")
		},
	},
	// --- 3.4-SPEC-004 (AC4, P0) — closed-generic keyed IsKeyedService true ------
	//
	// ClosedGenericsWithIsKeyedService: a closed generic is an ordinary registered
	// type (Story 2-5 D-3), so IsKeyedService(IFakeOpenGenericService[IFakeService],
	// "gen-key") reports true; a stranger key on the same type reports false.
	{
		Name: "keyed-isservice: closed-generic keyed reports true (ClosedGenericsWithIsKeyedService, AC4)",
		Run: func(t *testing.T, p Provider) {
			testutil.True(t, p.IsKeyedService(closedGenericKeyedType(), "gen-key"),
				"closed-generic keyed (IFakeOpenGenericService[IFakeService], \"gen-key\") must report true")
			testutil.True(t, !p.IsKeyedService(closedGenericKeyedType(), "nonexistent"),
				"closed-generic under a stranger key must report false")
		},
	},
	// --- 3.4-SPEC-005 (AC5, P0) — element-type query true; no implicit IEnumerable
	//
	// Go diff annotation for IEnumerableWithIsKeyedServiceAlwaysReturnsTrue: godi has
	// NO implicit IEnumerable<T> → slice mapping (Story 2-5 D-1 / AC5). IsKeyedService
	// answers only "can GetKeyedService(t, key) hit", i.e. the ELEMENT type. The
	// collection query is GetKeyedServices(t, key), independent of IsKeyedService.
	// So IsKeyedService(element type, key) reports true for a registered keyed type;
	// there is no "slice type always true" behavior to assert.
	{
		Name: "keyed-isservice: element-type query true, no implicit IEnumerable mapping (IEnumerableWithIsKeyedServiceAlwaysReturnsTrue Go diff, AC5)",
		Run: func(t *testing.T, p Provider) {
			// Element type under its registered key resolves → IsKeyedService true.
			testutil.True(t, p.IsKeyedService(keyedISType(), "service1"),
				"element type IKeyedService under \"service1\" must report true")
			// No implicit slice mapping: []fakes.IKeyedService is not a registered
			// service type, so IsKeyedService reports false for the slice type. The
			// collection query lives in GetKeyedServices, not IsKeyedService (D-1).
			sliceType := reflect.TypeFor[[]fakes.IKeyedService]()
			testutil.True(t, !p.IsKeyedService(sliceType, "service1"),
				"slice type []IKeyedService must report false (no implicit IEnumerable mapping)")
		},
	},
}

// --- 3.4-SPEC-RUN (AC1-AC5, P0) ----------------------------------------------

func TestConformanceKeyedIsService_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryKeyedIsService, keyedIsServiceSpecCases)
}

func TestConformanceKeyedIsService_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryKeyedIsService, keyedIsServiceSpecCases)
}

// --- 3.4-DIFF (AC6, P0) — cross-provider differential equivalence ------------
//
// AD-15 differential equivalence for the keyed IsService axis. Probes use
// provider-stable observables (bool), never pointer identity across providers (R4).
func TestConformanceKeyedIsService_DifferentialEquivalence(t *testing.T) {
	kt := keyedISType()
	ft := reflect.TypeFor[fakes.IFakeService]()
	gt := closedGenericKeyedType()

	// AC1: registered (type, key) hit equivalence.
	AssertEquivalent(t, rtFactoryKeyedIsService, cgFactoryKeyedIsService, func(p Provider) bool {
		return p.IsKeyedService(kt, "service1")
	})
	// AC1: stranger key miss equivalence.
	AssertEquivalent(t, rtFactoryKeyedIsService, cgFactoryKeyedIsService, func(p Provider) bool {
		return p.IsKeyedService(kt, "nonexistent")
	})
	// AC2: nil key routes to unkeyed IsService equivalence.
	AssertEquivalent(t, rtFactoryKeyedIsService, cgFactoryKeyedIsService, func(p Provider) bool {
		return p.IsKeyedService(ft, nil)
	})
	// AC3: AnyKey query is false (not panic) in BOTH providers.
	AssertEquivalent(t, rtFactoryKeyedIsService, cgFactoryKeyedIsService, func(p Provider) bool {
		return p.IsKeyedService(kt, godi.AnyKey)
	})
	// AC3: AnyKey query never panics in BOTH providers.
	AssertEquivalent(t, rtFactoryKeyedIsService, cgFactoryKeyedIsService, func(p Provider) bool {
		return notPanicsKeyedIsServiceAnyKey(p, kt)
	})
	// AC4: closed-generic keyed hit equivalence.
	AssertEquivalent(t, rtFactoryKeyedIsService, cgFactoryKeyedIsService, func(p Provider) bool {
		return p.IsKeyedService(gt, "gen-key")
	})
}

// notPanicsKeyedIsServiceAnyKey reports whether IsKeyedService(t, AnyKey) returns
// WITHOUT panicking — the AC3 pure-query guarantee, probed as a comparable bool
// for differential equivalence (both providers must agree: true = did not panic).
func notPanicsKeyedIsServiceAnyKey(p Provider, t reflect.Type) (didNotPanic bool) {
	defer func() {
		if r := recover(); r != nil {
			didNotPanic = false
		}
	}()
	_ = p.IsKeyedService(t, godi.AnyKey)
	return true
}
