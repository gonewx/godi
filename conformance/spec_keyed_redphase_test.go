// ATDD scaffold for Story 3.1 (AC1-AC10) — conformance suite:
// dual-provider keyed SpecCases + cross-provider differential equivalence for
// the keyed resolution axis (AD-15/AD-17).
//
// This file extends the conformance suite with keyed DI spec cases ported from
// dotnet KeyedDependencyInjectionSpecificationTests (8 basic methods). Each case
// runs against BOTH providers (runtime oracle + codegen B2). AD-17 mandates
// key type = any (not string); AD-15 mandates keyed coverage in codegen with
// ≥1 scenario + cross-provider differential equivalence.
//
// Fakes needed (Task 7): IKeyedService + KeyedServiceOne/Two in fakes_keyed.go.
// New runtime registration: runtime.AddKeyedSingleton / AddKeyedTransient.
// New codegen registration: godi.AddKeyedSingleton / AddKeyedTransient in
// conformance/codegen/registration.go + regenerated container_gen.go.
// New convenience: godi.GetRequiredKeyed[T] in keyed.go.
//
// Test standard R4: pointer identity compared WITHIN a single provider only;
// ACROSS providers use provider-stable observables for differential equivalence.
//
// RED PHASE — this scaffold fails because:
//   - runtime.AddKeyedSingleton / AddKeyedTransient do not exist (compile error)
//   - godi.GetRequiredKeyed[T] does not exist (compile error)
//   - Even after API creation: GetKeyedService(t, non-nil key) stub returns nil
//   - codegen container has no keyed registrations
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-3-1-keyed-registration-resolve.md.
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

// rtFactoryKeyed builds the runtime provider with keyed registrations. This is
// the runtime oracle for keyed resolution — form-B closures keyed by any-typed
// keys (AD-17). The registration mirrors the codegen registration source
// (AD-14: dual provider derives from one logical source).
//
// Registration layout:
//   - key="service1" → KeyedServiceOne (singleton)      — AC1/AC4/AC5
//   - key="service2" → KeyedServiceTwo (singleton)      — AC1
//   - key="both"     → KeyedServiceOne + Two (singleton) — AC3 (collection)
//   - key="singleton-factory" → KeyedServiceOne (singleton) — AC6 (shared)
//   - key="transient" → KeyedServiceOne (transient)     — AC7/AC8
//   - NO unkeyed IKeyedService registration              — AC2 (namespace isolation)
var rtFactoryKeyed = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()

		// AC1/AC4/AC5: singleton keyed — distinct keys
		runtime.AddKeyedSingleton[fakes.IKeyedService](b, "service1", func(p godi.Provider) fakes.IKeyedService {
			return fakes.NewKeyedServiceOne()
		})
		runtime.AddKeyedSingleton[fakes.IKeyedService](b, "service2", func(p godi.Provider) fakes.IKeyedService {
			return fakes.NewKeyedServiceTwo()
		})

		// AC3: keyed collection — same key "both", two impls, registration order
		runtime.AddKeyedSingleton[fakes.IKeyedService](b, "both", func(p godi.Provider) fakes.IKeyedService {
			return fakes.NewKeyedServiceOne()
		})
		runtime.AddKeyedSingleton[fakes.IKeyedService](b, "both", func(p godi.Provider) fakes.IKeyedService {
			return fakes.NewKeyedServiceTwo()
		})

		// AC6: singleton factory — resolve twice must return same instance
		runtime.AddKeyedSingleton[fakes.IKeyedService](b, "singleton-factory", func(p godi.Provider) fakes.IKeyedService {
			return fakes.NewKeyedServiceOne()
		})

		// AC7/AC8: transient keyed — every resolve returns a new instance
		runtime.AddKeyedTransient[fakes.IKeyedService](b, "transient", func(p godi.Provider) fakes.IKeyedService {
			return fakes.NewKeyedServiceOne()
		})

		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryKeyed builds the codegen provider from the regenerated
// container_gen.go with keyed registrations. In the red phase, the codegen
// container does NOT include keyed services yet, so all keyed assertions fail.
var cgFactoryKeyed = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// keyedSpecCases are SpecCases for keyed DI resolution. Each case runs against
// BOTH providers (runtime oracle + codegen). The 8 dotnet methods are mapped
// to AC1-AC9; AC10 (differential equivalence) is tested separately below.
var keyedSpecCases = []SpecCase{
	// --- 3.1-SPEC-001 (AC1, P0) — ResolveKeyedService ★ core --------------------
	{
		Name: "keyed: resolve distinct keys returns distinct impls (ResolveKeyedService, KeyedDISpec)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedService]()

			result1 := p.GetKeyedService(st, "service1")
			testutil.NotNil(t, result1)
			testutil.Equal(t, reflect.TypeOf(result1).String(), "*fakes.KeyedServiceOne")

			result2 := p.GetKeyedService(st, "service2")
			testutil.NotNil(t, result2)
			testutil.Equal(t, reflect.TypeOf(result2).String(), "*fakes.KeyedServiceTwo")
		},
	},
	// --- 3.1-SPEC-002 (AC1, P1) — key type = any (not string) -------------------
	{
		Name: "keyed: key type is any, not string — non-string key resolves (AD-17)",
		Run: func(t *testing.T, p Provider) {
			// This test verifies that non-string keys (int) work as map keys.
			// In the shared factory, only string keys are registered, so an int
			// key must return nil (unknown key). The assertion is that the provider
			// does not panic on non-string keys (AD-17: key type = any).
			st := reflect.TypeFor[fakes.IKeyedService]()
			got := p.GetKeyedService(st, 42)
			testutil.Nil(t, got)
		},
	},
	// --- 3.1-SPEC-003 (AC2, P0) — unknown key returns nil -----------------------
	{
		Name: "keyed: unknown key returns nil, GetService returns nil (namespace isolation, D-3)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedService]()

			// Unknown key → nil (not panic)
			got := p.GetKeyedService(st, "unknown-key")
			testutil.Nil(t, got)

			// Unkeyed GetService → nil (keyed and unkeyed are independent namespaces)
			unkeyed := p.GetService(st)
			testutil.Nil(t, unkeyed)
		},
	},
	// --- 3.1-SPEC-004 (AC3, P0) — ResolveKeyedServices (collection, registration order)
	{
		Name: "keyed: collection resolve returns all impls in registration order (ResolveKeyedServices, KeyedDISpec)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedService]()

			all := p.GetKeyedServices(st, "both")
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 2)
			testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*fakes.KeyedServiceOne")
			testutil.Equal(t, reflect.TypeOf(all[1]).String(), "*fakes.KeyedServiceTwo")
		},
	},
	// --- 3.1-SPEC-005 (AC4/AC5, P0) — ResolveKeyedServiceSingletonInstance/Type -
	{
		Name: "keyed: singleton keyed resolves correct type, GetService nil (ResolveKeyedServiceSingletonInstance/Type)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedService]()

			result := p.GetKeyedService(st, "service1")
			testutil.NotNil(t, result)
			testutil.Equal(t, reflect.TypeOf(result).String(), "*fakes.KeyedServiceOne")

			// GetService must not return keyed services (namespace isolation)
			testutil.Nil(t, p.GetService(st))
		},
	},
	// --- 3.1-SPEC-006 (AC6, P0) — ResolveKeyedServiceSingletonFactory (shared) --
	{
		Name: "keyed: singleton factory — two resolves return same instance (ResolveKeyedServiceSingletonFactory)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedService]()

			result1 := p.GetKeyedService(st, "singleton-factory")
			result2 := p.GetKeyedService(st, "singleton-factory")

			testutil.NotNil(t, result1)
			testutil.NotNil(t, result2)
			testutil.Equal(t, result1, result2) // same singleton instance (pointer identity within one provider, R4)
		},
	},
	// --- 3.1-SPEC-007 (AC7, P0) — ResolveKeyedServiceTransientType ---------------
	{
		Name: "keyed: transient keyed — two resolves return different instances (ResolveKeyedServiceTransientType)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedService]()

			result1 := p.GetKeyedService(st, "transient")
			result2 := p.GetKeyedService(st, "transient")

			testutil.NotNil(t, result1)
			testutil.NotNil(t, result2)
			testutil.True(t, result1 != result2,
				"transient keyed services must be different instances (AD-14)")
		},
	},
	// --- 3.1-SPEC-008 (AC8, P0) — ResolveKeyedServiceTransientFactory ------------
	//
	// Go difference (D-1 from Story 3-1): factory signature is func(Provider) T —
	// key is NOT passed to the factory (unlike dotnet's (sp, key) => new Service()).
	// The key is available via closure capture. This test verifies transient factory
	// behavior: each resolve creates a new instance.
	{
		Name: "keyed: transient factory — different instances, key via closure (ResolveKeyedServiceTransientFactory, D-1)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedService]()

			result1 := p.GetKeyedService(st, "transient")
			result2 := p.GetKeyedService(st, "transient")

			testutil.NotNil(t, result1)
			testutil.NotNil(t, result2)
			testutil.True(t, result1 != result2,
				"transient factory keyed services must be different instances")
		},
	},
	// --- 3.1-SPEC-009 (AC9, P0) — ResolveRequiredKeyedServiceThrowsIfNotFound ----
	//
	// Go difference: godi panics (not exception) — mirrors GetRequired[T] behavior
	// pinned in Story 1.2. Panic message must contain type and key information.
	{
		Name: "keyed: GetRequiredKeyed panics for unregistered key (ResolveRequiredKeyedServiceThrowsIfNotFound)",
		Run: func(t *testing.T, p Provider) {
			testutil.Panics(t, func() {
				godi.GetRequiredKeyed[fakes.IKeyedService](p, "not-registered")
			})
		},
	},
}

// --- 3.1-SPEC-RUN (AC1-AC9, P0) -----------------------------------------------

func TestConformanceKeyed_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryKeyed, keyedSpecCases)
}

func TestConformanceKeyed_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryKeyed, keyedSpecCases)
}

// --- 3.1-SPEC-010 (AC10, P0) — Differential Equivalence -----------------------

// TestConformanceKeyed_DifferentialEquivalence: AD-15 cross-provider differential
// equivalence for the keyed resolution axis. Uses provider-stable observables
// (dynamic type name, same/not-same conclusions, nil/non-nil) — never pointer
// identity across providers (R4).
func TestConformanceKeyed_DifferentialEquivalence(t *testing.T) {
	// AC1: keyed resolve dynamic type equivalence
	AssertEquivalent(t, rtFactoryKeyed, cgFactoryKeyed, func(p Provider) string {
		return dynTypeName(p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "service1"))
	})
	AssertEquivalent(t, rtFactoryKeyed, cgFactoryKeyed, func(p Provider) string {
		return dynTypeName(p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "service2"))
	})

	// AC2: unknown key → nil
	AssertEquivalent(t, rtFactoryKeyed, cgFactoryKeyed, func(p Provider) bool {
		return p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "unknown") == nil
	})

	// AC2: namespace isolation — unkeyed GetService for keyed-only type → nil
	AssertEquivalent(t, rtFactoryKeyed, cgFactoryKeyed, func(p Provider) bool {
		return p.GetService(reflect.TypeFor[fakes.IKeyedService]()) == nil
	})

	// AC3: keyed collection count equivalence
	AssertEquivalent(t, rtFactoryKeyed, cgFactoryKeyed, func(p Provider) int {
		all := p.GetKeyedServices(reflect.TypeFor[fakes.IKeyedService](), "both")
		if all == nil {
			return -1
		}
		return len(all)
	})

	// AC6: singleton keyed same/not-same conclusion
	AssertEquivalent(t, rtFactoryKeyed, cgFactoryKeyed, func(p Provider) bool {
		st := reflect.TypeFor[fakes.IKeyedService]()
		r1 := p.GetKeyedService(st, "singleton-factory")
		r2 := p.GetKeyedService(st, "singleton-factory")
		return r1 == r2 // singleton: same instance
	})

	// AC7: transient keyed same/not-same conclusion
	AssertEquivalent(t, rtFactoryKeyed, cgFactoryKeyed, func(p Provider) bool {
		st := reflect.TypeFor[fakes.IKeyedService]()
		r1 := p.GetKeyedService(st, "transient")
		r2 := p.GetKeyedService(st, "transient")
		return r1 != r2 // transient: different instances
	})
}
