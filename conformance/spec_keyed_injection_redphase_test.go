//go:build atdd_redphase

// ATDD scaffold for Story 3.2 (AC1-AC5, AC7) — conformance suite:
// dual-provider keyed lifecycle-via-scope + keyed dependency injection
// SpecCases + cross-provider differential equivalence.
//
// This file extends the conformance suite with keyed DI injection spec cases
// ported from dotnet KeyedDependencyInjectionSpecificationTests (12 methods,
// Story 3-2 slice). Each case runs against BOTH providers (runtime oracle +
// codegen B2).
//
// New Fakes needed (Task 1): IKeyedServiceWithDep + KeyedServiceWithDep,
// IKeyedServiceWithKey + KeyedServiceWithKey, IKeyedServiceWithProvider +
// KeyedServiceWithProvider — all in fakes_keyed.go.
//
// RED PHASE — this scaffold is behind //go:build atdd_redphase because:
//   - New Fake types may not yet exist (compile error if Task 1 not done)
//   - Codegen container does not include injection registrations
//   - Runtime injection tests should pass once Task 1 Fakes are created
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-3-2-keyed-lifetime-injection.md.
package conformance

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// ---------------------------------------------------------------------------
// AC1 — Keyed Lifecycle via Scope (uses existing rtFactoryKeyed registrations)
// ---------------------------------------------------------------------------

// keyedLifecycleSpecCases tests keyed services resolved FROM a scope service
// provider, verifying lifecycle semantics match non-keyed behavior (AC1).
// These extend Story 3-1's SPEC-010 (basic scoped isolation) into a complete
// lifecycle matrix: singleton sharing, transient isolation, scoped isolation.
var keyedLifecycleSpecCases = []SpecCase{
	// --- 3.2-SPEC-001 (AC1, P0) — Singleton keyed via scope --------------------
	//
	// dotnet: ResolveKeyedServiceSingletonFactoryFromScopeServiceProvider
	// Singleton keyed service resolved from root and from scope must be the SAME
	// instance — singletons are shared across the entire provider hierarchy.
	{
		Name: "keyed: singleton via scope — root and scope resolve same instance (ResolveKeyedServiceSingletonFactoryFromScope)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedService]()

			rootResult := p.GetKeyedService(st, "service1")
			testutil.NotNil(t, rootResult)

			scope := p.CreateScope()
			t.Cleanup(func() { _ = scope.Close() })
			scopeResult := scope.GetKeyedService(st, "service1")
			testutil.NotNil(t, scopeResult)

			testutil.Equal(t, rootResult, scopeResult)
		},
	},
	// --- 3.2-SPEC-002 (AC1, P0) — Transient keyed via scope -------------------
	//
	// dotnet: ResolveKeyedServiceTransientFactoryFromScopeServiceProvider
	// Transient keyed service resolved twice from a scope must return DIFFERENT
	// instances — transients are never cached.
	{
		Name: "keyed: transient via scope — two scope resolves return different instances (ResolveKeyedServiceTransientFactoryFromScope)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedService]()

			scope := p.CreateScope()
			t.Cleanup(func() { _ = scope.Close() })
			r1 := scope.GetKeyedService(st, "transient")
			r2 := scope.GetKeyedService(st, "transient")

			testutil.NotNil(t, r1)
			testutil.NotNil(t, r2)
			testutil.True(t, r1 != r2,
				"transient keyed via scope must return different instances (AD-14)")
		},
	},
	// --- 3.2-SPEC-003 (AC1, P0) — Scoped keyed via scope ----------------------
	//
	// dotnet: ResolveKeyedServiceScopedFactoryFromScopeServiceProvider
	// Extends 3.1-SPEC-010: scoped keyed from scope is same within scope, different
	// across scopes. This test mirrors the dotnet "FromScopeServiceProvider" variant
	// which emphasizes scope-provider resolution context.
	{
		Name: "keyed: scoped via scope — same scope shared, cross scope isolated (ResolveKeyedServiceScopedFactoryFromScope)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedService]()

			scope1 := p.CreateScope()
			t.Cleanup(func() { _ = scope1.Close() })
			s1r1 := scope1.GetKeyedService(st, "scoped")
			s1r2 := scope1.GetKeyedService(st, "scoped")
			testutil.NotNil(t, s1r1)
			testutil.Equal(t, s1r1, s1r2)

			scope2 := p.CreateScope()
			t.Cleanup(func() { _ = scope2.Close() })
			s2r1 := scope2.GetKeyedService(st, "scoped")
			testutil.NotNil(t, s2r1)
			testutil.True(t, s1r1 != s2r1,
				"scoped keyed must be isolated across scopes")
		},
	},
}

func TestConformanceKeyedLifecycle_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryKeyed, keyedLifecycleSpecCases)
}

func TestConformanceKeyedLifecycle_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryKeyed, keyedLifecycleSpecCases)
}

// ---------------------------------------------------------------------------
// AC2-AC5 — Keyed Dependency Injection (needs new Fakes + new factory)
// ---------------------------------------------------------------------------

// rtFactoryKeyedInjection builds the runtime provider with keyed injection
// registrations for Story 3-2. It includes the "service1" base registration
// from Story 3-1 (needed as dependency target for AC2) plus the new injection
// Fakes. This factory is self-contained for the ATDD scaffold; during dev-story,
// these registrations merge into the existing rtFactoryKeyed.
var rtFactoryKeyedInjection = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()

		// Base: "service1" singleton (dependency target for AC2/AC5)
		runtime.AddKeyedSingleton[fakes.IKeyedService](b, "service1", func(p godi.Provider) fakes.IKeyedService {
			return fakes.NewKeyedServiceOne()
		})

		// AC2: keyed dependency injection — form-B closure calls GetKeyedService
		// (Go equivalent of [FromKeyedServices("service1")])
		runtime.AddKeyedSingleton[fakes.IKeyedServiceWithDep](b, "with-dep", func(p godi.Provider) fakes.IKeyedServiceWithDep {
			dep := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "service1")
			return fakes.NewKeyedServiceWithDep(dep.(fakes.IKeyedService))
		})

		// AC3: key value injection — closure captures key variable
		// (Go equivalent of [ServiceKey])
		key := "injected-key"
		runtime.AddKeyedSingleton[fakes.IKeyedServiceWithKey](b, key, func(p godi.Provider) fakes.IKeyedServiceWithKey {
			return fakes.NewKeyedServiceWithKey(key)
		})

		// AC4: missing keyed dependency — factory calls GetKeyedService for
		// an unregistered key, receives nil (D-4: Go doesn't auto-throw)
		runtime.AddKeyedSingleton[fakes.IKeyedServiceWithDep](b, "missing-dep", func(p godi.Provider) fakes.IKeyedServiceWithDep {
			dep := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "nonexistent")
			if dep == nil {
				return fakes.NewKeyedServiceWithDep(nil)
			}
			return fakes.NewKeyedServiceWithDep(dep.(fakes.IKeyedService))
		})

		// AC5: provider injection (singleton) — factory stores provider reference
		// (Go equivalent of IServiceProvider constructor injection)
		runtime.AddKeyedSingleton[fakes.IKeyedServiceWithProvider](b, "with-provider", func(p godi.Provider) fakes.IKeyedServiceWithProvider {
			return fakes.NewKeyedServiceWithProvider(p)
		})

		// AC5: provider injection (scoped) — for scope context verification
		runtime.AddKeyedScoped[fakes.IKeyedServiceWithProvider](b, "scoped-with-provider", func(p godi.Provider) fakes.IKeyedServiceWithProvider {
			return fakes.NewKeyedServiceWithProvider(p)
		})

		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// keyedInjectionSpecCases tests keyed dependency injection patterns (AC2-AC5).
// These are the Go adaptations of dotnet's [FromKeyedServices], [ServiceKey],
// and IServiceProvider constructor injection — all realized through form-B
// factory closures in godi.
var keyedInjectionSpecCases = []SpecCase{
	// --- 3.2-SPEC-004 (AC2, P0) — Keyed dependency injection ------------------
	//
	// dotnet: ResolveKeyedServiceSingletonInstanceWithKeyedParameter
	// Go difference (D-1): [FromKeyedServices("service1")] → form-B closure
	// calling p.GetKeyedService. The factory receives the dependency through
	// explicit provider resolution, not attribute-driven injection.
	{
		Name: "keyed: form-B closure injects keyed dependency via GetKeyedService (AC2, D-1, ResolveKeyedServiceWithKeyedParameter)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedServiceWithDep]()
			result := p.GetKeyedService(st, "with-dep")
			if result == nil {
				t.Fatal("expected non-nil IKeyedServiceWithDep for key \"with-dep\"")
			}

			svc := result.(fakes.IKeyedServiceWithDep)
			testutil.NotNil(t, svc.Dep())
			testutil.Equal(t, reflect.TypeOf(svc.Dep()).String(), "*fakes.KeyedServiceOne")
		},
	},
	// --- 3.2-SPEC-005 (AC3, P0) — Key value injection -------------------------
	//
	// dotnet: ResolveKeyedServiceSingletonInstanceWithKeyInjection
	// Go difference (D-2): [ServiceKey] → closure capture. The factory captures
	// the key variable from the enclosing scope and passes it to the constructor.
	{
		Name: "keyed: closure capture injects key value (AC3, D-2, ResolveKeyedServiceWithKeyInjection)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedServiceWithKey]()
			result := p.GetKeyedService(st, "injected-key")
			if result == nil {
				t.Fatal("expected non-nil IKeyedServiceWithKey for key \"injected-key\"")
			}

			svc := result.(fakes.IKeyedServiceWithKey)
			testutil.Equal(t, svc.Key(), any("injected-key"))
		},
	},
	// --- 3.2-SPEC-006 (AC4, P1) — Missing keyed dependency --------------------
	//
	// dotnet: ResolveKeyedServiceSingletonInstanceWithKeyedParameter_MissingRegistration_Throws
	// Go difference (D-4): Go does not auto-throw on missing keyed dep. The
	// factory's GetKeyedService returns nil for an unknown key; the factory
	// decides how to handle it (here: stores nil).
	{
		Name: "keyed: missing keyed dependency — factory receives nil (AC4, D-4, MissingRegistration)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedServiceWithDep]()
			result := p.GetKeyedService(st, "missing-dep")
			if result == nil {
				t.Fatal("expected non-nil IKeyedServiceWithDep for key \"missing-dep\"")
			}

			svc := result.(fakes.IKeyedServiceWithDep)
			testutil.Nil(t, svc.Dep())
		},
	},
	// --- 3.2-SPEC-007 (AC5, P0) — Provider injection + dynamic resolve --------
	//
	// dotnet: ResolveKeyedServiceSingletonInstanceFromInjectedServiceProvider +
	//         SimpleServiceKeyedResolution
	// Go difference (D-3): form-B closure parameter p IS the injected provider.
	// The factory stores it; later the test uses it to dynamically resolve a
	// keyed service.
	{
		Name: "keyed: injected provider resolves keyed service dynamically (AC5, D-3, FromInjectedServiceProvider)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IKeyedServiceWithProvider]()
			result := p.GetKeyedService(st, "with-provider")
			if result == nil {
				t.Fatal("expected non-nil IKeyedServiceWithProvider for key \"with-provider\"")
			}

			svc := result.(fakes.IKeyedServiceWithProvider)
			testutil.NotNil(t, svc.ServiceProvider())

			injectedProvider := svc.ServiceProvider().(Provider)
			resolved := injectedProvider.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "service1")
			testutil.NotNil(t, resolved)
			testutil.Equal(t, reflect.TypeOf(resolved).String(), "*fakes.KeyedServiceOne")
		},
	},
	// --- 3.2-SPEC-008 (AC5, P0) — Scope provider injection --------------------
	//
	// dotnet: ResolveKeyedServiceSingletonInstanceFromScopeServiceProvider_SingletonDep +
	//         SimpleServiceKeyedResolution
	// Verifies that when resolved from a scope, the injected provider is the
	// scope's provider, and singletons are still accessible (traversing to root).
	{
		Name: "keyed: scope provider injection — singleton accessible from scope (AC5, FromScopeServiceProvider_SingletonDep)",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()
			t.Cleanup(func() { _ = scope.Close() })

			st := reflect.TypeFor[fakes.IKeyedServiceWithProvider]()
			result := scope.GetKeyedService(st, "scoped-with-provider")
			if result == nil {
				t.Fatal("expected non-nil IKeyedServiceWithProvider for key \"scoped-with-provider\"")
			}

			svc := result.(fakes.IKeyedServiceWithProvider)
			testutil.NotNil(t, svc.ServiceProvider())

			injectedProvider := svc.ServiceProvider().(Provider)
			singleton := injectedProvider.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "service1")
			testutil.NotNil(t, singleton)
			testutil.Equal(t, reflect.TypeOf(singleton).String(), "*fakes.KeyedServiceOne")

			rootSingleton := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "service1")
			testutil.Equal(t, singleton, rootSingleton)
		},
	},
}

func TestConformanceKeyedInjection_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryKeyedInjection, keyedInjectionSpecCases)
}

func TestConformanceKeyedInjection_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryKeyed, keyedInjectionSpecCases)
}

// ---------------------------------------------------------------------------
// AC7 — Differential Equivalence (AD-15 cross-provider)
// ---------------------------------------------------------------------------

// TestConformanceKeyedLifecycle_DifferentialEquivalence: AD-15 cross-provider
// differential for the keyed lifecycle-via-scope axis. Uses rtFactoryKeyed and
// cgFactoryKeyed which share the same logical registration source.
func TestConformanceKeyedLifecycle_DifferentialEquivalence(t *testing.T) {
	// AC1: singleton keyed via scope — same instance conclusion
	AssertEquivalent(t, rtFactoryKeyed, cgFactoryKeyed, func(p Provider) bool {
		st := reflect.TypeFor[fakes.IKeyedService]()
		rootR := p.GetKeyedService(st, "service1")
		scope := p.CreateScope()
		defer func() { _ = scope.Close() }()
		scopeR := scope.GetKeyedService(st, "service1")
		return rootR == scopeR
	})

	// AC1: transient keyed via scope — different instances conclusion
	AssertEquivalent(t, rtFactoryKeyed, cgFactoryKeyed, func(p Provider) bool {
		st := reflect.TypeFor[fakes.IKeyedService]()
		scope := p.CreateScope()
		defer func() { _ = scope.Close() }()
		r1 := scope.GetKeyedService(st, "transient")
		r2 := scope.GetKeyedService(st, "transient")
		return r1 != r2
	})

	// AC1: scoped keyed via scope — same within scope conclusion
	AssertEquivalent(t, rtFactoryKeyed, cgFactoryKeyed, func(p Provider) bool {
		st := reflect.TypeFor[fakes.IKeyedService]()
		scope := p.CreateScope()
		defer func() { _ = scope.Close() }()
		r1 := scope.GetKeyedService(st, "scoped")
		r2 := scope.GetKeyedService(st, "scoped")
		return r1 == r2
	})
}

// TestConformanceKeyedInjection_DifferentialEquivalence: AD-15 cross-provider
// differential for the keyed injection axis. Uses rtFactoryKeyedInjection and
// cgFactoryKeyed. In the red phase, cgFactoryKeyed lacks injection registrations
// so these will show mismatches (expected).
func TestConformanceKeyedInjection_DifferentialEquivalence(t *testing.T) {
	// AC2: keyed dependency type name equivalence
	AssertEquivalent(t, rtFactoryKeyedInjection, cgFactoryKeyed, func(p Provider) string {
		result := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedServiceWithDep](), "with-dep")
		if result == nil {
			return "<nil>"
		}
		return dynTypeName(result)
	})

	// AC3: key value equivalence
	AssertEquivalent(t, rtFactoryKeyedInjection, cgFactoryKeyed, func(p Provider) bool {
		result := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedServiceWithKey](), "injected-key")
		if result == nil {
			return false
		}
		return result.(fakes.IKeyedServiceWithKey).Key() == any("injected-key")
	})

	// AC4: missing dependency nil conclusion
	AssertEquivalent(t, rtFactoryKeyedInjection, cgFactoryKeyed, func(p Provider) bool {
		result := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedServiceWithDep](), "missing-dep")
		if result == nil {
			return false
		}
		return result.(fakes.IKeyedServiceWithDep).Dep() == nil
	})
}
