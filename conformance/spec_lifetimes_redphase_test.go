// ATDD red-phase scaffold for Story 2.2 (AC1/AC2/AC3) — conformance suite:
// dual-provider lifetime spec cases + cross-provider differential equivalence.
//
// The conformance suite runs the SAME behavioral assertions against both
// providers (runtime oracle + codegen B2). AD-15 mandates that the lifetime
// sharing axis (scoped/transient resolution semantics) be covered in codegen
// with ≥1 scenario + cross-provider differential equivalence.
//
// New fakes needed (Task 5): IFakeScopedService, IFakeTransientService,
// IFakeTransientWithScopedDep + implementations in conformance/fakes/.
// New runtime registration: runtime.AddScoped / runtime.AddTransient.
// New codegen registration: godi.AddScoped / godi.AddTransient in
// conformance/codegen/registration.go + regenerated container_gen.go.
//
// Test standard R4: pointer identity compared WITHIN a single provider only;
// ACROSS providers use provider-stable observables for differential equivalence.
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-2-scoped-transient-lifetimes.md.
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

// rtFactoryLifetimes builds the runtime provider with all three lifetimes.
// This extends the existing rtFactory (singleton-only, spec_test.go) with
// scoped and transient registrations from the SAME logical source as codegen
// (AD-14: dual provider derives from one DSL source).
var rtFactoryLifetimes = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		// Singletons (retain existing registrations)
		runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
			return fakes.NewFakeService()
		})
		// Scoped
		runtime.AddScoped[fakes.IFakeScopedService](b, func(godi.Provider) fakes.IFakeScopedService {
			return fakes.NewFakeScopedService()
		})
		// Transient
		runtime.AddTransient[fakes.IFakeTransientService](b, func(godi.Provider) fakes.IFakeTransientService {
			return fakes.NewFakeTransientService()
		})
		// AC2: transient with scoped dependency
		runtime.AddTransient[fakes.IFakeTransientWithScopedDep](b, func(p godi.Provider) fakes.IFakeTransientWithScopedDep {
			dep := p.GetService(reflect.TypeFor[fakes.IFakeScopedService]()).(fakes.IFakeScopedService)
			return fakes.NewFakeTransientWithScopedDep(dep)
		})
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryLifetimes builds the codegen provider from the regenerated
// container_gen.go with scoped/transient registrations.
var cgFactoryLifetimes = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// lifetimeSpecCases extends the spec suite with scoped/transient resolution
// assertions. Each case is run against BOTH providers (Task 5). Pointer
// identity is compared within a single provider only (R4).
var lifetimeSpecCases = []SpecCase{
	// --- 2.2-SPEC-001 (AC1, P0) — ScopedServiceCanBeResolved ★ core ---------
	{
		Name: "scoped: root ≠ scope, scope internal same (ScopedServiceCanBeResolved)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IFakeScopedService]()

			rootScoped := p.GetService(st)

			scope := p.CreateScope()
			t.Cleanup(func() { _ = scope.Close() })

			scopeScoped1 := scope.GetService(st)
			scopeScoped2 := scope.GetService(st)

			testutil.NotNil(t, rootScoped)
			testutil.NotNil(t, scopeScoped1)
			testutil.True(t, rootScoped != scopeScoped1,
				"root-level scoped ≠ scope-level scoped (D-2)")
			testutil.Equal(t, scopeScoped1, scopeScoped2)
		},
	},
	// --- 2.2-SPEC-002 (AC1, P0) — NestedScopedServiceCanBeResolved ----------
	{
		Name: "scoped: nested scope outer ≠ inner (NestedScopedServiceCanBeResolved)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IFakeScopedService]()

			outer := p.CreateScope()
			t.Cleanup(func() { _ = outer.Close() })

			inner := outer.CreateScope()
			t.Cleanup(func() { _ = inner.Close() })

			outerScoped := outer.GetService(st)
			innerScoped := inner.GetService(st)

			testutil.NotNil(t, outerScoped)
			testutil.NotNil(t, innerScoped)
			testutil.True(t, outerScoped != innerScoped,
				"nested scope scoped must differ from outer scope scoped")
		},
	},
	// --- 2.2-SPEC-003 (AC1, P0) — ReturnDifferentInstancesPerResolution_ForTransientServices
	{
		Name: "transient: every resolve returns different instance",
		Run: func(t *testing.T, p Provider) {
			tt := reflect.TypeFor[fakes.IFakeTransientService]()

			svc1 := p.GetService(tt)
			svc2 := p.GetService(tt)

			testutil.NotNil(t, svc1)
			testutil.NotNil(t, svc2)
			testutil.True(t, svc1 != svc2,
				"transient services must be different instances (AD-14)")
		},
	},
	// --- 2.2-SPEC-004 (AC1, P0) — TransientServiceCanBeResolvedFromProvider
	{
		Name: "transient: resolvable from root provider",
		Run: func(t *testing.T, p Provider) {
			got := p.GetService(reflect.TypeFor[fakes.IFakeTransientService]())
			testutil.NotNil(t, got)
		},
	},
	// --- 2.2-SPEC-005 (AC1, P0) — TransientServiceCanBeResolvedFromScope
	{
		Name: "transient: resolvable from scope, differs from root and each other",
		Run: func(t *testing.T, p Provider) {
			tt := reflect.TypeFor[fakes.IFakeTransientService]()

			rootT := p.GetService(tt)

			scope := p.CreateScope()
			t.Cleanup(func() { _ = scope.Close() })

			scopeT1 := scope.GetService(tt)
			scopeT2 := scope.GetService(tt)

			testutil.True(t, rootT != scopeT1, "root transient ≠ scope transient")
			testutil.True(t, rootT != scopeT2, "root transient ≠ scope transient")
			testutil.True(t, scopeT1 != scopeT2, "scope transient ≠ scope transient (AD-14)")
		},
	},
	// --- 2.2-SPEC-006 (AC1, P1) — SingletonServiceCanBeResolvedFromScope (regression)
	{
		Name: "singleton: shared across scopes (regression, SingletonServiceCanBeResolvedFromScope)",
		Run: func(t *testing.T, p Provider) {
			st := reflect.TypeFor[fakes.IFakeService]()
			rootSingleton := p.GetService(st)

			scope := p.CreateScope()
			t.Cleanup(func() { _ = scope.Close() })

			scopeSingleton := scope.GetService(st)
			testutil.Equal(t, rootSingleton, scopeSingleton)
		},
	},
	// --- 2.2-SPEC-007 (AC2, P0) — FactoryServicesAreCreatedAsPartOfCreatingObjectGraph
	{
		Name: "AC2: transient depends on scoped — same scope reuses scoped",
		Run: func(t *testing.T, p Provider) {
			tt := reflect.TypeFor[fakes.IFakeTransientWithScopedDep]()

			t1 := p.GetService(tt).(fakes.IFakeTransientWithScopedDep)
			t2 := p.GetService(tt).(fakes.IFakeTransientWithScopedDep)

			testutil.True(t, t1 != t2,
				"two transients must be different instances")
			testutil.True(t, t1.ScopedDep() == t2.ScopedDep(),
				"AC2: both transients must share the SAME scoped dependency (root scope)")
		},
	},
}

// --- 2.2-SPEC-RUN (AC1/AC2/AC3, P0) -----------------------------------------

// TestConformanceLifetimes_RuntimeProvider: run the lifetime spec cases against
// the runtime provider (compliance oracle).
func TestConformanceLifetimes_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryLifetimes, lifetimeSpecCases)
}

// TestConformanceLifetimes_CodegenProvider: run the lifetime spec cases against
// the codegen provider.
func TestConformanceLifetimes_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryLifetimes, lifetimeSpecCases)
}

// --- 2.2-SPEC-008 (AC3, P0) — Differential Equivalence ---------------------

// TestConformanceLifetimes_DifferentialEquivalence: AD-15 cross-provider
// differential equivalence for the lifetime sharing axis. Uses provider-stable
// observables (IsService bool, dynamic type name, same/not-same conclusions)
// — never pointer identity across providers (R4).
func TestConformanceLifetimes_DifferentialEquivalence(t *testing.T) {
	// IsService must agree for all three lifetimes across both providers.
	AssertEquivalent(t, rtFactoryLifetimes, cgFactoryLifetimes, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeScopedService]())
	})
	AssertEquivalent(t, rtFactoryLifetimes, cgFactoryLifetimes, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeTransientService]())
	})

	// Dynamic type name equivalence for resolved services.
	AssertEquivalent(t, rtFactoryLifetimes, cgFactoryLifetimes, func(p Provider) string {
		return dynTypeName(p.GetService(reflect.TypeFor[fakes.IFakeScopedService]()))
	})
	AssertEquivalent(t, rtFactoryLifetimes, cgFactoryLifetimes, func(p Provider) string {
		return dynTypeName(p.GetService(reflect.TypeFor[fakes.IFakeTransientService]()))
	})

	// Scoped same/not-same conclusion as provider-stable observable.
	AssertEquivalent(t, rtFactoryLifetimes, cgFactoryLifetimes, func(p Provider) bool {
		st := reflect.TypeFor[fakes.IFakeScopedService]()
		rootScoped := p.GetService(st)
		scope := p.CreateScope()
		defer scope.Close()
		scopeScoped := scope.GetService(st)
		return rootScoped != scopeScoped // must be true (different instances)
	})

	// Transient same/not-same conclusion.
	AssertEquivalent(t, rtFactoryLifetimes, cgFactoryLifetimes, func(p Provider) bool {
		tt := reflect.TypeFor[fakes.IFakeTransientService]()
		t1 := p.GetService(tt)
		t2 := p.GetService(tt)
		return t1 != t2 // must be true (different instances)
	})
}
