//go:build atdd_redphase

// Story 5.4 (AC3) — M2 second-tier threshold gate assertion:
// ServiceProviderValidationTests (26) + CircularDependencyTests (10) = 36
// dotnet methods × godi coverage mapping.
//
// Counting convention (aligned with threshold_116_test.go):
//   covered  — directly mapped to a godi test
//   adapted  — rewritten for Go差異 but covers same semantic intent
//   na       — Go mechanism wall (with mandatory justification)
//   Formula: covered + adapted + na = 36 (no MISSING gap)
//
// Activation: delete the //go:build line and blank line to join the default build.
package conformance

import "testing"

type m2MethodMapping struct {
	DotnetMethod string
	DotnetFile   string
	GodiTest     string
	Status       coverageStatus
	Note         string
}

// ── ServiceProviderValidationTests (26 methods) ──────────────────────────

var dotnetServiceProviderValidationMap = []m2MethodMapping{
	// Group A: captive detection (validateScopes → plan.Validate)
	{DotnetMethod: "GetService_Throws_WhenScopedIsInjectedIntoSingleton", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_CaptiveDirect_SingletonDependsOnScoped + plan/validate_captive_test.go", Status: covered},
	{DotnetMethod: "GetService_Throws_WhenScopedIsInjectedIntoSingletonThroughTransient", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_CaptiveTransitive_SingletonViaTransientToScoped + plan/validate_captive_test.go", Status: covered},
	{DotnetMethod: "GetService_Throws_WhenScopedIsInjectedIntoSingletonThroughSingleton", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_CaptiveTransitive_SingletonViaSingletonToScoped", Status: covered},
	{DotnetMethod: "GetService_Throws_WhenScopedIsInjectedIntoSingletonThroughSingletonAndScopedWhileInScope", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_CaptiveTransitive_ScopedViaSingletonToScoped", Status: covered},

	// Group B: root-resolve scoped (validateScopes → Go root-scoped semantics)
	{DotnetMethod: "GetService_Throws_WhenGetServiceForScopedServiceIsCalledOnRoot", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_RootResolveScoped_GoAdaptation", Status: adapted, Note: "godi root-scoped semantics: root acts as root scope, resolve succeeds (dotnet validateScopes throws). Captive left-shifted to plan.Validate (AD-3)"},
	{DotnetMethod: "GetService_Throws_WhenGetServiceForScopedServiceIsCalledOnRoot_IL_Replacement", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_RootResolveScoped_ILReplacement_GoAdaptation", Status: adapted, Note: "dotnet IL replacement is engine-internal optimization; godi root-scoped semantics apply"},
	{DotnetMethod: "GetService_Throws_WhenGetServiceForScopedServiceIsCalledOnRootViaTransient", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_RootResolveScopedViaTransient_GoAdaptation", Status: adapted, Note: "godi root-scoped: transient with scoped dep resolves from root"},

	// Group C: polymorphic / multi-impl scope mix
	{DotnetMethod: "GetService_DoesNotThrow_WhenGetServiceForPolymorphicServiceIsCalledOnRoot_AndTheLastOneIsNotScoped", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_PolymorphicLastNotScoped_BuildSucceeds", Status: covered},

	// Group N/A: ServiceKey attribute (Go has no attributes, AD-13)
	{DotnetMethod: "BuildServiceProvider_AnyKey_ServiceKeyWithStronglyTypedArgument", DotnetFile: "SPVTests", GodiTest: "", Status: na, Note: "Go has no [ServiceKey] attribute; requires runtime reflection to inject key into ctor param (AD-13 forbids reflect-based construction on main path)"},
	{DotnetMethod: "BuildServiceProvider_AnyKey_ServiceKeyWithObjectTypedArgument", DotnetFile: "SPVTests", GodiTest: "", Status: na, Note: "Go has no [ServiceKey] attribute (AD-13)"},
	{DotnetMethod: "BuildServiceProvider_ServiceKeyWithObjectTypedArgument", DotnetFile: "SPVTests", GodiTest: "", Status: na, Note: "Go has no [ServiceKey] attribute (AD-13)"},
	{DotnetMethod: "BuildServiceProvider_AnyKey_ServiceKeyWithObjectAndIntTypedArguments", DotnetFile: "SPVTests", GodiTest: "", Status: na, Note: "Go has no [ServiceKey] attribute (AD-13)"},

	// Group C continued: scope validation for generic collections
	{DotnetMethod: "ScopeValidation_ShouldBeAbleToDistingushGenericCollections_WhenGetServiceIsCalledOnRoot", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_ScopeValidation_MixedLifetimeCollections", Status: adapted, Note: "godi root-scoped: both scoped and transient resolve from root; no validateScopes throw"},

	// Group N/A: IServiceScopeFactory
	{DotnetMethod: "GetService_DoesNotThrow_WhenScopeFactoryIsInjectedIntoSingleton", DotnetFile: "SPVTests", GodiTest: "", Status: na, Note: "godi uses Provider.CreateScope() interface method, not injectable IServiceScopeFactory service"},

	// Group C continued: multi-impl scope
	{DotnetMethod: "GetService_DoesNotThrow_WhenGetServiceForServiceWithMultipleImplementationScopesWhereLastIsNotScoped", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_MultiImplScope_LastNotScoped_Resolves", Status: covered},
	{DotnetMethod: "GetService_Throws_WhenGetServiceForServiceWithMultipleImplementationScopesWhereLastIsScoped", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_MultiImplScope_LastIsScoped_GoAdaptation", Status: adapted, Note: "godi root-scoped: last-wins scoped resolves from root (dotnet validateScopes throws)"},
	{DotnetMethod: "GetService_DoesNotThrow_WhenGetServiceForNonScopedImplementationWithMultipleImplementationScopesWhereLastIsScoped", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_MultiImplScope_NonScopedImpl_Resolves", Status: covered},

	// Group D: ValidateOnBuild → plan.Validate + genlib.Generate
	{DotnetMethod: "BuildServiceProvider_ValidateOnBuild_Throws_WhenScopedIsInjectedIntoSingleton", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_ValidateOnBuild_CaptiveDirect", Status: covered},
	{DotnetMethod: "BuildServiceProvider_ValidateOnBuild_Throws_WhenScopedIsInjectedIntoSingleton_ReverseRegistrationOrder", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_ValidateOnBuild_CaptiveDirect_ReverseOrder", Status: covered},

	// Group N/A: IServiceScopeFactory ValidateOnBuild
	{DotnetMethod: "BuildServiceProvider_ValidateOnBuild_DoesNotThrow_WhenScopeFactoryIsInjectedIntoSingleton", DotnetFile: "SPVTests", GodiTest: "", Status: na, Note: "godi uses Provider.CreateScope(), not injectable IServiceScopeFactory"},

	// Group D continued: CachedCallSites
	{DotnetMethod: "BuildServiceProvider_ValidateOnBuild_Throws_WhenScopedIsInjectedIntoSingleton_CachedCallSites", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_ValidateOnBuild_CaptiveChain", Status: adapted, Note: "dotnet CallSite caching is internal; godi tests captive through dependency chain"},
	{DotnetMethod: "BuildServiceProvider_ValidateOnBuild_DoesNotThrow_CachedCallSites", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_ValidateOnBuild_AllScoped_NoCaptive", Status: adapted, Note: "all-scoped chain has no captive → Validate passes"},

	// Group E: mixed unresolvable + all-validated
	{DotnetMethod: "BuildServiceProvider_ValidateOnBuild_ThrowsForUnresolvableServices", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_ValidateOnBuild_MixedErrors_MissingAndCycle", Status: covered},
	{DotnetMethod: "BuildServiceProvider_ValidateOnBuild_SkipsOpenGenerics", DotnetFile: "SPVTests", GodiTest: "", Status: na, Note: "godi has no open generic registration (only closed instantiations, AD-4/AD-10); semantically auto-satisfied"},
	{DotnetMethod: "BuildServiceProvider_ValidateOnBuild_ValidatesAllDescriptors", DotnetFile: "SPVTests", GodiTest: "spec_validation: TestSPV_ValidateOnBuild_ValidatesAllDescriptors", Status: covered},
	{DotnetMethod: "BuildServiceProvider_ValidateOnBuild_ThrowsWhenImplementationIsNotAssignableToService", DotnetFile: "SPVTests", GodiTest: "", Status: na, Note: "Go generics enforce I/Impl type safety at compile time (AddSingleton[I,Impl] requires Impl implements I)"},
}

// ── CircularDependencyTests (10 methods) ─────────────────────────────────

var dotnetCircularDependencyMap = []m2MethodMapping{
	{DotnetMethod: "SelfCircularDependency", DotnetFile: "CDTests", GodiTest: "plan/validate_cycle_test.go: TestValidate_Cycle_SelfDependency + runtime/provider_cycle_test.go: TestBuild_Cycle_SelfDependency", Status: covered},
	{DotnetMethod: "SelfCircularDependencyGenericDirect", DotnetFile: "CDTests", GodiTest: "plan/validate_cycle_test.go: TestValidate_Cycle_SelfDep_Generic", Status: covered},
	{DotnetMethod: "SelfCircularDependencyGenericIndirect", DotnetFile: "CDTests", GodiTest: "plan/validate_cycle_test.go: TestValidate_Cycle_IndirectGenericCycle", Status: covered},
	{DotnetMethod: "DirectCircularDependency", DotnetFile: "CDTests", GodiTest: "plan/validate_cycle_test.go: TestValidate_Cycle_DirectAB + runtime/provider_cycle_test.go: TestBuild_Cycle_DirectAB", Status: covered},
	{DotnetMethod: "IndirectCircularDependency", DotnetFile: "CDTests", GodiTest: "plan/validate_cycle_test.go: TestValidate_Cycle_IndirectABC", Status: covered},
	{DotnetMethod: "NoCircularDependencyGeneric", DotnetFile: "CDTests", GodiTest: "plan/validate_cycle_test.go: TestValidate_Cycle_NoFalsePositive_Generic", Status: covered},
	{DotnetMethod: "SelfCircularDependencyWithInterface", DotnetFile: "CDTests", GodiTest: "plan/validate_cycle_test.go: TestValidate_Cycle_SelfDependency (interface-typed service)", Status: covered},
	{DotnetMethod: "DirectCircularDependencyWithInterface", DotnetFile: "CDTests", GodiTest: "plan/validate_cycle_test.go: TestValidate_Cycle_DirectAB", Status: covered},
	{DotnetMethod: "IndirectCircularDependencyWithInterface", DotnetFile: "CDTests", GodiTest: "plan/validate_cycle_test.go: TestValidate_Cycle_IndirectABC", Status: covered},
	{DotnetMethod: "NoCircularDependency", DotnetFile: "CDTests", GodiTest: "plan/validate_cycle_test.go: TestValidate_Cycle_NoCycle_ValidGraph + runtime/provider_cycle_test.go: TestBuild_Cycle_NoCycle_ValidGraph", Status: covered},
}

func TestM2Threshold_36_CoverageAssertion(t *testing.T) {
	const (
		spvExpected = 26
		cdExpected  = 10
		m2Total     = spvExpected + cdExpected
	)

	if len(dotnetServiceProviderValidationMap) != spvExpected {
		t.Fatalf("dotnetServiceProviderValidationMap has %d entries, want exactly %d",
			len(dotnetServiceProviderValidationMap), spvExpected)
	}
	if len(dotnetCircularDependencyMap) != cdExpected {
		t.Fatalf("dotnetCircularDependencyMap has %d entries, want exactly %d",
			len(dotnetCircularDependencyMap), cdExpected)
	}

	allMethods := make([]m2MethodMapping, 0, m2Total)
	allMethods = append(allMethods, dotnetServiceProviderValidationMap...)
	allMethods = append(allMethods, dotnetCircularDependencyMap...)

	var coveredCount, adaptedCount, naCount, unmappedCount int
	var unmapped []string

	fileGroupCounts := map[string]struct{ total, covered, adapted, na int }{}

	for _, m := range allMethods {
		fg := fileGroupCounts[m.DotnetFile]
		fg.total++

		switch m.Status {
		case covered:
			coveredCount++
			fg.covered++
		case adapted:
			adaptedCount++
			fg.adapted++
		case na:
			naCount++
			fg.na++
		default:
			unmappedCount++
			unmapped = append(unmapped, m.DotnetMethod)
		}

		fileGroupCounts[m.DotnetFile] = fg
	}

	if unmappedCount > 0 {
		t.Errorf("found %d unmapped methods (no status assigned): %v", unmappedCount, unmapped)
	}

	effectiveCoverage := coveredCount + adaptedCount
	effectiveTarget := m2Total - naCount

	if effectiveCoverage < effectiveTarget {
		t.Errorf("M2 coverage gap: covered(%d) + adapted(%d) = %d, but need %d (= %d - %d N/A)",
			coveredCount, adaptedCount, effectiveCoverage, effectiveTarget, m2Total, naCount)
	}

	t.Logf("M2 Second-Tier Threshold Coverage:")
	t.Logf("  Total methods:     %d", m2Total)
	t.Logf("  Covered:           %d", coveredCount)
	t.Logf("  Adapted:           %d", adaptedCount)
	t.Logf("  N/A:               %d", naCount)
	t.Logf("  Effective:         %d / %d", effectiveCoverage, effectiveTarget)

	expectedFileCounts := map[string]int{
		"SPVTests": spvExpected,
		"CDTests":  cdExpected,
	}

	for file, want := range expectedFileCounts {
		got := fileGroupCounts[file]
		if got.total != want {
			t.Errorf("file group %s: got %d methods, want %d", file, got.total, want)
		}
		t.Logf("  %-12s total=%-3d covered=%-3d adapted=%-3d na=%-3d",
			file, got.total, got.covered, got.adapted, got.na)
	}
}
