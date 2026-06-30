// ATDD red-phase scaffold for Story 5.4 (AC1/AC3) — conformance suite:
// dual-provider validation consistency + ServiceProviderValidationTests
// (26 dotnet methods) coverage.
//
// AC1: same illegal graph → codegen (genlib.Generate) rejects AND runtime
// (Builder.Build) rejects, with consistent rejection semantics.
// AC3: ServiceProviderValidationTests 26 methods mapped to godi equivalents.
//
// Dual-provider validation tests do NOT use RunSuite/SpecCase because they
// test Build() failure (no Provider to pass). They construct both plan.Registry
// (codegen path) and runtime.Builder (runtime path) from the same logical
// illegal graph, then assert both paths reject.
//
// Captive dependency is a KNOWN DESIGN DIFFERENCE: codegen hard-rejects via
// plan.Validate; runtime Build() may succeed because root acts as root-scope
// (scoped-from-root semantics, Story 3-2). Tests explicitly annotate this.
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-5-4-validation-dual-provider-m2.md.
package conformance

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
	"github.com/gonewx/godi/runtime"
)

// ═══════════════════════════════════════════════════════════════════════════
// Local validation fakes (unexported, avoid polluting conformance/fakes/)
// ═══════════════════════════════════════════════════════════════════════════

type valIFoo interface{ isValFoo() }
type valFoo struct{ bar valIBar }

func (*valFoo) isValFoo() {}

type valFoo2 struct{ bar valIBar }

func (*valFoo2) isValFoo() {}

type valIBar interface{ isValBar() }
type valBar struct{}

func (*valBar) isValBar() {}

type valBar2 struct{ baz valIBaz }

func (*valBar2) isValBar() {}

type valBar3 struct{}

func (*valBar3) isValBar() {}

type valIBaz interface{ isValBaz() }
type valBaz struct{}

func (*valBaz) isValBaz() {}

type valBaz2 struct{}

func (*valBaz2) isValBaz() {}

type valBazRecursive struct{ baz valIBaz }

func (*valBazRecursive) isValBaz() {}

// ═══════════════════════════════════════════════════════════════════════════
// AC1 — Dual-Provider Validation Consistency (5.4-VAL-01/02/03)
// ═══════════════════════════════════════════════════════════════════════════

// --- 5.4-VAL-01 (AC1, P0) — Missing dependency: both providers reject ------

func TestValidation_DualProvider_MissingDep(t *testing.T) {
	// codegen path: plan.Registry with A(deps=[B]), B not registered
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "Foo",
		GoType:   "*svc.Foo",
		Ctor:     "svc.NewFoo",
		Lifetime: plan.Singleton,
		Deps:     []string{"Bar"},
	})

	src, genErr := genlib.Generate(reg, "container")
	testutil.Error(t, genErr)
	testutil.Nil(t, src)
	testutil.True(t, strings.Contains(genErr.Error(), "validation failed"),
		"codegen path must indicate validation failure")

	// runtime path: Foo factory resolves Bar (unregistered) → type assertion panic → Build error
	b := runtime.NewBuilder()
	runtime.AddSingleton[valIFoo](b, func(p godi.Provider) valIFoo {
		dep := p.GetService(reflect.TypeFor[valIBar]()).(valIBar)
		return &valFoo{bar: dep}
	})

	provider, buildErr := b.Build()
	testutil.Error(t, buildErr)
	testutil.Nil(t, provider)

	t.Logf("codegen error: %s", genErr)
	t.Logf("runtime error: %s", buildErr)
}

// --- 5.4-VAL-02 (AC1, P0) — Captive dependency: codegen rejects, runtime
// known design difference (root-scoped semantics) -------------------------

func TestValidation_DualProvider_CaptiveDep(t *testing.T) {
	// codegen path: Singleton depends on Scoped → plan.Validate captive error
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "ScopedBar",
		GoType:   "*svc.Bar",
		Ctor:     "svc.NewBar",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "SingletonFoo",
		GoType:   "*svc.Foo",
		Ctor:     "svc.NewFoo",
		Lifetime: plan.Singleton,
		Deps:     []string{"ScopedBar"},
	})

	src, genErr := genlib.Generate(reg, "container")
	testutil.Error(t, genErr)
	testutil.Nil(t, src)
	testutil.True(t, strings.Contains(genErr.Error(), "captive dependency"),
		"codegen path must detect captive dependency")

	// runtime path: KNOWN DESIGN DIFFERENCE — root acts as root-scope, so
	// Build() may succeed (scoped service materializes as root-scoped instance).
	// This is NOT a bug: captive detection relies on plan.Validate (AD-3
	// validation left-shift), not runtime Build(). Annotated per D-1.
	b := runtime.NewBuilder()
	runtime.AddScoped[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar{}
	})
	runtime.AddSingleton[valIFoo](b, func(p godi.Provider) valIFoo {
		dep := p.GetService(reflect.TypeFor[valIBar]()).(valIBar)
		return &valFoo{bar: dep}
	})

	provider, buildErr := b.Build()

	// runtime may or may not error — document actual behavior
	if buildErr != nil {
		t.Logf("runtime also rejected captive (bonus safety net): %s", buildErr)
	} else {
		t.Logf("runtime Build() succeeded (root-scoped semantics) — " +
			"captive detection relies on plan.Validate (AD-3), annotated as known design difference")
		testutil.NotNil(t, provider)
		_ = provider.Close()
	}

	t.Logf("codegen error: %s", genErr)
}

// --- 5.4-VAL-03 (AC1, P0) — Circular dependency: both providers reject -----

func TestValidation_DualProvider_CycleDep(t *testing.T) {
	// codegen path: A↔B cycle
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B"},
	})
	reg.Service(plan.Service{
		Name:     "B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"A"},
	})

	src, genErr := genlib.Generate(reg, "container")
	testutil.Error(t, genErr)
	testutil.Nil(t, src)
	testutil.True(t, strings.Contains(genErr.Error(), "circular dependency"),
		"codegen path must detect circular dependency")

	// runtime path: A↔B cycle → materializing guard panic → Build error
	b := runtime.NewBuilder()
	runtime.AddSingleton[valIFoo](b, func(p godi.Provider) valIFoo {
		dep := p.GetService(reflect.TypeFor[valIBar]()).(valIBar)
		return &valFoo{bar: dep}
	})
	runtime.AddSingleton[valIBar](b, func(p godi.Provider) valIBar {
		dep := p.GetService(reflect.TypeFor[valIFoo]()).(valIFoo)
		_ = dep
		return &valBar{}
	})

	provider, buildErr := b.Build()
	testutil.Error(t, buildErr)
	testutil.Nil(t, provider)
	testutil.True(t, strings.Contains(buildErr.Error(), "circular dependency"),
		"runtime Build error must mention circular dependency")

	t.Logf("codegen error: %s", genErr)
	t.Logf("runtime error: %s", buildErr)
}

// ═══════════════════════════════════════════════════════════════════════════
// AC3 — ServiceProviderValidationTests (26 methods) godi coverage
// ═══════════════════════════════════════════════════════════════════════════
//
// Methods mapped to godi equivalents. Tests grouped by dotnet category:
//   Group A: ValidateScopes captive (1-4) → plan.Validate captive
//   Group B: ValidateScopes root-resolve (5-7) → Go差异: root-scoped
//   Group C: Polymorphic/MultiImpl scope (8, 15-17) → plan.Validate / Build
//   Group D: ValidateOnBuild variants (18-22) → plan.Validate
//   Group E: Mixed unresolvable + all-validated (23, 25) → plan.Validate
//
// N/A methods (documented in threshold_m2_test.go):
//   9-12: ServiceKey attribute (Go no attributes, AD-13)
//   14, 20: IServiceScopeFactory (godi uses Provider.CreateScope)
//   24: SkipsOpenGenerics (godi no open generic registration)
//   26: ThrowsWhenImplementationIsNotAssignableToService (Go generics type-safe)

// --- Group A: captive detection via plan.Validate (dotnet methods 1-4) ------

// 5.4-SPV-01: GetService_Throws_WhenScopedIsInjectedIntoSingleton
func TestSPV_CaptiveDirect_SingletonDependsOnScoped(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "Bar", GoType: "*svc.Bar", Ctor: "svc.NewBar", Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name: "Foo", GoType: "*svc.Foo", Ctor: "svc.NewFoo", Lifetime: plan.Singleton,
		Deps: []string{"Bar"},
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) >= 1, "captive: Singleton→Scoped must error")
	foundCaptive := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "captive dependency") {
			foundCaptive = true
		}
	}
	testutil.True(t, foundCaptive, "must contain captive dependency error")
}

// 5.4-SPV-02: GetService_Throws_WhenScopedIsInjectedIntoSingletonThroughTransient
func TestSPV_CaptiveTransitive_SingletonViaTransientToScoped(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "Baz", GoType: "*svc.Baz", Ctor: "svc.NewBaz", Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name: "Bar2", GoType: "*svc.Bar2", Ctor: "svc.NewBar2", Lifetime: plan.Transient,
		Deps: []string{"Baz"},
	})
	reg.Service(plan.Service{
		Name: "Foo", GoType: "*svc.Foo", Ctor: "svc.NewFoo", Lifetime: plan.Singleton,
		Deps: []string{"Bar2"},
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) >= 1,
		"captive: Singleton→Transient→Scoped must error (transitive)")
	foundCaptive := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "captive dependency") {
			foundCaptive = true
		}
	}
	testutil.True(t, foundCaptive, "must contain captive dependency error")
}

// 5.4-SPV-03: GetService_Throws_WhenScopedIsInjectedIntoSingletonThroughSingleton
func TestSPV_CaptiveTransitive_SingletonViaSingletonToScoped(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "Baz", GoType: "*svc.Baz", Ctor: "svc.NewBaz", Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name: "Bar2", GoType: "*svc.Bar2", Ctor: "svc.NewBar2", Lifetime: plan.Singleton,
		Deps: []string{"Baz"},
	})
	reg.Service(plan.Service{
		Name: "Foo", GoType: "*svc.Foo", Ctor: "svc.NewFoo", Lifetime: plan.Singleton,
		Deps: []string{"Bar2"},
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) >= 1,
		"captive: Singleton→Singleton→Scoped must error (transitive)")
}

// 5.4-SPV-04: GetService_Throws_WhenScopedIsInjectedIntoSingletonThroughSingletonAndScopedWhileInScope
func TestSPV_CaptiveTransitive_ScopedViaSingletonToScoped(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "Baz", GoType: "*svc.Baz", Ctor: "svc.NewBaz", Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name: "Bar2", GoType: "*svc.Bar2", Ctor: "svc.NewBar2", Lifetime: plan.Singleton,
		Deps: []string{"Baz"},
	})
	reg.Service(plan.Service{
		Name: "Foo", GoType: "*svc.Foo", Ctor: "svc.NewFoo", Lifetime: plan.Scoped,
		Deps: []string{"Bar2"},
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) >= 1,
		"captive: Singleton→Scoped via Scoped consumer must error")
}

// --- Group B: root-resolve scoped — Go差异 (dotnet methods 5-7) ------------
// dotnet validateScopes blocks scoped resolve from root. godi uses root-scoped
// semantics (root acts as root scope, returns real instances). These tests
// verify godi's divergent but intentional behavior.

// 5.4-SPV-05: GetService_Throws_WhenGetServiceForScopedServiceIsCalledOnRoot
// Go差异: godi root-scoped semantics → resolves successfully (not throws)
func TestSPV_RootResolveScoped_GoAdaptation(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar{}
	})

	provider, err := b.Build()
	testutil.NoError(t, err)
	testutil.NotNil(t, provider)
	t.Cleanup(func() { _ = provider.Close() })

	// godi root-scoped: resolves successfully (dotnet would throw with validateScopes:true)
	got := provider.GetService(reflect.TypeFor[valIBar]())
	testutil.NotNil(t, got)
	t.Logf("Go差异: root-scoped resolve succeeded (dotnet validateScopes would throw)")
}

// 5.4-SPV-06: GetService_Throws_WhenGetServiceForScopedServiceIsCalledOnRoot_IL_Replacement
// Go差異: same as SPV-05. dotnet IL replacement is engine-internal; N/A to godi.
func TestSPV_RootResolveScoped_ILReplacement_GoAdaptation(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar{}
	})

	provider, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	// resolve from scope first (dotnet triggers IL replacement after 2 calls)
	scope := provider.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })

	scopeResult := scope.GetService(reflect.TypeFor[valIBar]())
	testutil.NotNil(t, scopeResult)

	// root resolve: godi succeeds (dotnet would throw)
	rootResult := provider.GetService(reflect.TypeFor[valIBar]())
	testutil.NotNil(t, rootResult)
}

// 5.4-SPV-07: GetService_Throws_WhenGetServiceForScopedServiceIsCalledOnRootViaTransient
// Go差異: godi transient factory resolving scoped from root → root-scoped materializes
func TestSPV_RootResolveScopedViaTransient_GoAdaptation(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar{}
	})
	runtime.AddTransient[valIFoo](b, func(p godi.Provider) valIFoo {
		dep := p.GetService(reflect.TypeFor[valIBar]()).(valIBar)
		return &valFoo{bar: dep}
	})

	provider, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	got := provider.GetService(reflect.TypeFor[valIFoo]())
	testutil.NotNil(t, got)
	t.Logf("Go差異: transient with scoped dep resolves from root (dotnet would throw)")
}

// --- Group C: polymorphic / multi-impl scope mix (dotnet methods 8, 15-17) --

// 5.4-SPV-08: GetService_DoesNotThrow_WhenGetServiceForPolymorphicServiceIsCalledOnRoot
// dotnet: validateScopes doesn't throw when last registration is not scoped.
// godi: root-scoped semantics → Build + resolve succeed regardless of lifetime mix.
// Runtime resolves by lifetime priority (singleton > scoped > transient), not
// dotnet's "last descriptor wins" — only Assert Build success + non-nil resolve.
func TestSPV_PolymorphicLastNotScoped_BuildSucceeds(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar{}
	})
	runtime.AddTransient[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar3{}
	})

	provider, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	got := provider.GetService(reflect.TypeFor[valIBar]())
	testutil.NotNil(t, got)
}

// 5.4-SPV-15: GetService_DoesNotThrow_WhenGetServiceForServiceWithMultipleImplementationScopesWhereLastIsNotScoped
func TestSPV_MultiImplScope_LastNotScoped_Resolves(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar{}
	})
	runtime.AddSingleton[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar2{baz: &valBaz{}}
	})

	provider, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	got := provider.GetService(reflect.TypeFor[valIBar]())
	testutil.NotNil(t, got)
}

// 5.4-SPV-16: GetService_Throws_WhenGetServiceForServiceWithMultipleImplementationScopesWhereLastIsScoped
// Go差異: godi root-scoped → resolves (dotnet with validateScopes would throw)
func TestSPV_MultiImplScope_LastIsScoped_GoAdaptation(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar{}
	})
	runtime.AddScoped[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar2{baz: &valBaz{}}
	})

	provider, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	// godi root-scoped: last-wins is scoped, root resolves it as root-scoped
	got := provider.GetService(reflect.TypeFor[valIBar]())
	testutil.NotNil(t, got)
	t.Logf("Go差異: last-wins scoped from root resolves (dotnet validateScopes throws)")
}

// 5.4-SPV-17: GetService_DoesNotThrow_WhenGetServiceForNonScopedImplementation
func TestSPV_MultiImplScope_NonScopedImpl_Resolves(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar{}
	})
	runtime.AddSingleton[*valBar](b, func(p godi.Provider) *valBar {
		return &valBar{}
	})
	runtime.AddScoped[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar2{baz: &valBaz{}}
	})

	provider, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	got := provider.GetService(reflect.TypeFor[*valBar]())
	testutil.NotNil(t, got)
}

// --- Group D: ValidateOnBuild variants via plan.Validate (dotnet methods 18-19, 21-22) --

// 5.4-SPV-18: BuildServiceProvider_ValidateOnBuild_Throws_WhenScopedIsInjectedIntoSingleton
// Equivalent to SPV-01 but via plan.Validate + genlib.Generate gate
func TestSPV_ValidateOnBuild_CaptiveDirect(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "Bar", GoType: "*svc.Bar", Ctor: "svc.NewBar", Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name: "Foo", GoType: "*svc.Foo", Ctor: "svc.NewFoo", Lifetime: plan.Singleton,
		Deps: []string{"Bar"},
	})

	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)
	testutil.True(t, strings.Contains(err.Error(), "captive dependency"),
		"ValidateOnBuild must detect captive")
}

// 5.4-SPV-19: same as 18 with reverse registration order
func TestSPV_ValidateOnBuild_CaptiveDirect_ReverseOrder(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "Foo", GoType: "*svc.Foo", Ctor: "svc.NewFoo", Lifetime: plan.Singleton,
		Deps: []string{"Bar"},
	})
	reg.Service(plan.Service{
		Name: "Bar", GoType: "*svc.Bar", Ctor: "svc.NewBar", Lifetime: plan.Scoped,
	})

	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)
}

// 5.4-SPV-21: ValidateOnBuild_Throws_CachedCallSites — captive through chain
func TestSPV_ValidateOnBuild_CaptiveChain(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "Baz", GoType: "*svc.Baz", Ctor: "svc.NewBaz", Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name: "Bar2", GoType: "*svc.Bar2", Ctor: "svc.NewBar2", Lifetime: plan.Scoped,
		Deps: []string{"Baz"},
	})
	reg.Service(plan.Service{
		Name: "Foo", GoType: "*svc.Foo", Ctor: "svc.NewFoo", Lifetime: plan.Scoped,
		Deps: []string{"Bar2"},
	})
	reg.Service(plan.Service{
		Name: "Foo2", GoType: "*svc.Foo2", Ctor: "svc.NewFoo2", Lifetime: plan.Singleton,
		Deps: []string{"Bar2"},
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) >= 1,
		"captive chain: Singleton Foo2→Scoped Bar2 must error")
}

// 5.4-SPV-22: ValidateOnBuild_DoesNotThrow_CachedCallSites — all scoped, no captive
func TestSPV_ValidateOnBuild_AllScoped_NoCaptive(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "Baz", GoType: "*svc.Baz", Ctor: "svc.NewBaz", Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name: "Bar2", GoType: "*svc.Bar2", Ctor: "svc.NewBar2", Lifetime: plan.Scoped,
		Deps: []string{"Baz"},
	})
	reg.Service(plan.Service{
		Name: "Foo", GoType: "*svc.Foo", Ctor: "svc.NewFoo", Lifetime: plan.Scoped,
		Deps: []string{"Bar2"},
	})
	reg.Service(plan.Service{
		Name: "Foo2", GoType: "*svc.Foo2", Ctor: "svc.NewFoo2", Lifetime: plan.Scoped,
		Deps: []string{"Bar2"},
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) == 0,
		"all scoped chain → no captive → Validate must pass")
}

// --- Group E: mixed unresolvable + all-validated (dotnet methods 23, 25) ----

// 5.4-SPV-23: ValidateOnBuild_ThrowsForUnresolvableServices — mixed errors
// Foo depends on unregistered Bar (missing dep) + BazRecursive self-cycle
func TestSPV_ValidateOnBuild_MixedErrors_MissingAndCycle(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "Foo", GoType: "*svc.Foo", Ctor: "svc.NewFoo", Lifetime: plan.Transient,
		Deps: []string{"Bar_Missing"},
	})
	reg.Service(plan.Service{
		Name: "BazRecursive", GoType: "*svc.BazRecursive", Ctor: "svc.NewBazRecursive", Lifetime: plan.Transient,
		Deps: []string{"BazRecursive"},
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) >= 2,
		"mixed errors: must report ≥2 errors (missing dep + cycle)")

	var foundMissing, foundCycle bool
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "missing") || strings.Contains(msg, "unresolved") {
			foundMissing = true
		}
		if strings.Contains(msg, "circular dependency") {
			foundCycle = true
		}
	}
	testutil.True(t, foundMissing, "must contain missing dependency error")
	testutil.True(t, foundCycle, "must contain circular dependency error")
}

// 5.4-SPV-25: ValidateOnBuild_ValidatesAllDescriptors
// BazRecursive (self-cycle) + Baz (valid). Validator must not skip the cyclic one.
func TestSPV_ValidateOnBuild_ValidatesAllDescriptors(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "BazRecursive", GoType: "*svc.BazRecursive", Ctor: "svc.NewBazRecursive", Lifetime: plan.Transient,
		Deps: []string{"BazRecursive"},
	})
	reg.Service(plan.Service{
		Name: "Baz", GoType: "*svc.Baz", Ctor: "svc.NewBaz", Lifetime: plan.Transient,
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) >= 1,
		"BazRecursive self-cycle must be detected even when valid Baz co-exists")
	foundCycle := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "circular dependency") {
			foundCycle = true
		}
	}
	testutil.True(t, foundCycle, "must detect BazRecursive self-cycle")
}

// 5.4-SPV-13: ScopeValidation_ShouldBeAbleToDistingushGenericCollections
// Adapted: multi-registration with mixed scoped/transient lifetimes
func TestSPV_ScopeValidation_MixedLifetimeCollections(t *testing.T) {
	b := runtime.NewBuilder()
	// IBar: transient then scoped
	runtime.AddTransient[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar{}
	})
	runtime.AddScoped[valIBar](b, func(p godi.Provider) valIBar {
		return &valBar3{}
	})
	// IBaz: both transient (no scope issue)
	runtime.AddTransient[valIBaz](b, func(p godi.Provider) valIBaz {
		return &valBaz{}
	})
	runtime.AddTransient[valIBaz](b, func(p godi.Provider) valIBaz {
		return &valBaz2{}
	})

	provider, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	// godi root-scoped: both IBar and IBaz resolve from root
	barResult := provider.GetService(reflect.TypeFor[valIBar]())
	testutil.NotNil(t, barResult)

	bazResult := provider.GetService(reflect.TypeFor[valIBaz]())
	testutil.NotNil(t, bazResult)
}
