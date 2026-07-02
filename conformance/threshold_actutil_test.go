//go:build !atdd_redphase

// Story 6.5 (AC5/AC6) — ActivatorUtilities (impl-layer) threshold gate.
//
// dotnet source: .../DI.Tests/ActivatorUtilitiesTests.cs (v10.0.9)
//   - 17 [Fact] + 3 [Theory] = 20 non-conditional methods  → the 220 ledger count
//   - 12 [ConditionalTheory(typeof(RemoteExecutor))]        → Go-N/A, NOT counted
//
// D-1 (highest risk): this is the impl-layer file, DISTINCT from the Spec-layer
// Specification.Tests/ActivatorUtilitiesTests.cs (20 methods) that Story 2.6
// already ported to activator/activator_redphase_test.go. Do NOT conflate them.
//
// Mirrors the threshold_m2/csf/spct/async/ext accounting pattern. NOT tagged
// atdd_redphase — compiles in the default build as a live accounting ledger.
// The 12 ConditionalTheory methods are honestly archived here as N/A with
// architectural anchors (NFR7: N/A must carry justification, not be silently
// dropped). See conformance/ACTIVATOR_DIVERGENCE.md for the narrative.
//
// Counting convention (aligned with the sibling gates):
//
//	covered  — directly mapped to a green godi test
//	adapted  — rewritten for a Go difference but covers the same semantic intent
//	na       — Go mechanism wall / deferred capability (mandatory Note)
//	Formula (20 ledger methods): covered + adapted + na == 20 (no MISSING gap)
package conformance

import "testing"

// actUtilMethodMapping records the disposition of one impl-layer dotnet method.
type actUtilMethodMapping struct {
	ID           string
	DotnetMethod string
	DotnetType   string // "Fact" | "Theory(N)"
	GodiTest     string
	Status       coverageStatus
	Note         string
}

// dotnetActUtilMethodMap — the 20 non-conditional impl-layer methods
// (17 Fact + 3 Theory). The dotnet method numbering (#1..#21, skipping #12)
// follows the Story 6.5 AC text; it is preserved in the ID field for audit.
var dotnetActUtilMethodMap = []actUtilMethodMapping{
	// ── best-match: longest available constructor (AC1) ──────────────────────
	{ID: "ACTUTIL-01", DotnetMethod: "CreateInstance_ClassWithABCS_UsesTheLongestAvailableConstructor", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_LongestAvailableConstructor", Status: covered},
	{ID: "ACTUTIL-02", DotnetMethod: "CreateInstance_OneCtor_IsRegistered_CreatesInstanceSuccessfully", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_OneCtorIsRegistered", Status: covered},
	{ID: "ACTUTIL-04", DotnetMethod: "CreateInstance_DifferentOrders_CreatesInstanceSuccessfully", DotnetType: "Theory(3)",
		GodiTest: "activator_impl: TestActivatorImpl_DifferentOrders (ABCS1/2/3)", Status: covered},
	{ID: "ACTUTIL-08", DotnetMethod: "CreateInstance_ClassWithABCS_UsesTheLongestAvailableConstructor_ParameterOrderDoesntMatter", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_ParameterOrderDoesntMatter", Status: covered},
	{ID: "ACTUTIL-09", DotnetMethod: "CreateInstance_ClassWithABC_ChoosesDefaultConstructorNoMatterCtorOrder", DotnetType: "Theory(2)",
		GodiTest: "activator_impl: TestActivatorImpl_ChoosesDefaultConstructor (First/Last)", Status: covered},
	{ID: "ACTUTIL-10", DotnetMethod: "CreateInstance_ClassWithABCS_BNotRegistered_UsesLongestPossibleCtorTakingAllRegisteredAndPassedParameters", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_BNotRegistered_UsesLongestPossible", Status: covered},

	// ── ActivatorUtilitiesConstructorAttribute / marked ctor (AC2) ───────────
	{ID: "ACTUTIL-11", DotnetMethod: "CreateInstance_ClassWithABC_ConstructorWithAttribute_PicksCtorWithAttr_NoMatterDefinitionOrder", DotnetType: "Theory(2)",
		GodiTest: "activator_impl: TestActivatorImpl_ConstructorWithAttribute_PicksMarked (First/Last)", Status: covered},
	{ID: "ACTUTIL-13", DotnetMethod: "CreateInstanceFailsWithAmbiguousConstructor", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_FailsWithAmbiguousConstructor", Status: covered},
	{ID: "ACTUTIL-14", DotnetMethod: "CreateInstanceFailsWithAmbiguousConstructor_ReversedOrder", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_FailsWithAmbiguousConstructor_ReversedOrder", Status: covered},
	{ID: "ACTUTIL-15", DotnetMethod: "CreateInstancePassesWithAmbiguousConstructor", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_PassesWithAmbiguousConstructor", Status: covered},
	{ID: "ACTUTIL-16", DotnetMethod: "CreateInstancePassesWithAmbiguousConstructor_ReversedOrder", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_PassesWithAmbiguousConstructor_ReversedOrder", Status: covered},
	{ID: "ACTUTIL-17", DotnetMethod: "CreateInstanceIgnoresActivatorUtilitiesConstructorAttribute", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_IgnoresAttributeForLongest (B stays nil)", Status: covered,
		Note: "Ignores group: marked short ctor beats longer unmarked ctor; B == nil (D-3)"},
	{ID: "ACTUTIL-18", DotnetMethod: "CreateInstanceIgnoresActivatorUtilitiesConstructorAttribute_ReversedOrder", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_IgnoresAttributeForLongest_ReversedOrder (B stays nil)", Status: covered,
		Note: "Ignores group reversed; B == nil (D-3)"},
	{ID: "ACTUTIL-19", DotnetMethod: "CreateInstance_ClassWithABC_MultipleCtorsWithSameLength_ThrowsAmbiguous", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_MultipleCtorsWithSameLength_ThrowsAmbiguous", Status: covered,
		Note: "error message must carry the target type name + length (2.6 AC14 patch continuation)"},

	// ── CreateFactory pre-compiled factory (AC3) ─────────────────────────────
	{ID: "ACTUTIL-20", DotnetMethod: "CreateFactory_CreatesFactoryMethod_4Types_3Injected", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_CreateFactory_4Types_3Injected", Status: adapted,
		Note: "Go has one generic CreateFactory[T] returning a func value; no non-generic ObjectFactory / ObjectFactory<T> named-delegate distinction — Assert.IsType<ObjectFactory> is N/A (ACTIVATOR_DIVERGENCE.md)"},
	{ID: "ACTUTIL-21", DotnetMethod: "CreateFactory_CreatesFactoryMethod_5Types_5Injected", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_CreateFactory_5Types_5Injected", Status: adapted,
		Note: "dotnet's '5 = fixed vs Span<> invoker threshold' is a reflection impl detail; Go has no such split — semantic intent (all 5 provider-resolved) preserved, threshold assertion N/A"},

	// ── error paths & null handling (AC4) ────────────────────────────────────
	{ID: "ACTUTIL-03", DotnetMethod: "CreateInstance_BadlyConfiguredIServiceProviderIsService_ProperlyCreatesUnambiguousInstance", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_UnambiguousConstructionSucceeds", Status: adapted,
		Note: "godi IsService is an intrinsic Provider method, not an injectable/replaceable IServiceProviderIsService service (AD-16); the 'badly configured IsService' premise cannot hold — core 'unambiguous construction succeeds' assertion kept, IsServiceGotCalled probe is N/A (D-4)"},
	{ID: "ACTUTIL-05", DotnetMethod: "CreateInstance_NullInstance_HandlesBadInputWithInvalidOperationException", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_BadInputSurfacesAsError", Status: adapted,
		Note: "dotnet passes a null typed as B; Go has no compile-time type recovery for a nil interface arg — core 'bad input → error' kept, the null-typed-as-B mechanism is N/A"},
	{ID: "ACTUTIL-06", DotnetMethod: "TypeActivatorThrowsOnNullProvider", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_NilProvider_CreateInstance", Status: covered,
		Note: "red→green: requires an explicit nil-provider check in CreateInstance (maps ArgumentNullException)"},
	{ID: "ACTUTIL-07", DotnetMethod: "FactoryActivatorThrowsOnNullProvider", DotnetType: "Fact",
		GodiTest: "activator_impl: TestActivatorImpl_NilProvider_Factory", Status: covered,
		Note: "red→green: explicit nil-provider check at factory call; message contains \"provider\""},
}

// TestActUtil_ThresholdGate_20Methods asserts the impl-layer coverage ledger
// totals exactly 20 dotnet methods (17 Fact + 3 Theory) with no MISSING gap.
func TestActUtil_ThresholdGate_20Methods(t *testing.T) {
	const expectedTotal = 20

	if len(dotnetActUtilMethodMap) != expectedTotal {
		t.Fatalf("ActivatorUtilities(impl) mapping declares %d methods, want %d — update map or expectedTotal",
			len(dotnetActUtilMethodMap), expectedTotal)
	}

	var coveredCount, adaptedCount, naCount, missingCount int
	for _, m := range dotnetActUtilMethodMap {
		switch m.Status {
		case covered:
			coveredCount++
		case adapted:
			adaptedCount++
		case na:
			naCount++
		default:
			missingCount++
			t.Errorf("method %s (%s) has no valid coverage status", m.ID, m.DotnetMethod)
		}
		// adapted / na entries must carry a justification Note (NFR7).
		if (m.Status == adapted || m.Status == na) && m.Note == "" {
			t.Errorf("method %s (%s) is %q but carries no justification Note (NFR7)",
				m.ID, m.DotnetMethod, m.Status)
		}
	}

	total := coveredCount + adaptedCount + naCount
	if total != expectedTotal {
		t.Fatalf("coverage sum covered=%d adapted=%d na=%d total=%d, want %d (gap=%d — MISSING methods)",
			coveredCount, adaptedCount, naCount, total, expectedTotal, expectedTotal-total)
	}
	if missingCount != 0 {
		t.Fatalf("%d methods have MISSING status — red-phase not fully resolved", missingCount)
	}

	t.Logf("ActivatorUtilities(impl) threshold gate: %d methods (covered=%d, adapted=%d, na=%d)",
		expectedTotal, coveredCount, adaptedCount, naCount)
}

// conditionalTheoryNA records one of the 12 [ConditionalTheory(RemoteExecutor)]
// methods that Go cannot express. These are NOT part of the 220 ledger
// (acceptance-contract counts the impl file as 17+3); they are archived here
// with architectural anchors so the honesty threshold (NFR7) is met.
type conditionalTheoryNA struct {
	DotnetMethod string
	Category     string // RemoteExecutor | KeyedParams | CollectibleAssembly
	Anchor       string
}

var dotnetActUtilConditionalTheories = []conditionalTheoryNA{
	// KeyedParams ×4 — CreateFactory injecting [FromKeyedServices] parameters.
	{"CreateFactory_CreatesFactoryMethod_KeyedParams", "KeyedParams",
		"godi CreateFactory has no keyed-parameter injection slot; keyed factory params = future capability (deferred), AD-17 keyed semantics"},
	{"CreateFactory_CreatesFactoryMethod_KeyedParams_5Types", "KeyedParams",
		"same as KeyedParams: no keyed factory-param slot (deferred)"},
	{"CreateFactory_CreatesFactoryMethod_KeyedParams_ValueTypes", "KeyedParams",
		"same as KeyedParams: value-typed keys unsupported for factory params (deferred)"},
	{"CreateFactory_CreatesFactoryMethod_KeyedParams_1Injected", "KeyedParams",
		"same as KeyedParams: mixed explicit + keyed provider params (deferred)"},

	// RemoteExecutor ×6 — cross-process + DisableDynamicCode verification.
	{"CreateFactory_RemoteExecutor_CreatesFactoryMethod", "RemoteExecutor",
		"Go has no xunit RemoteExecutor (cross-process test framework) and no dynamic-code switch; single reflect-free codegen + Tier C reflection backend (AD-6 Runtime-engine only)"},
	{"CreateFactory_RemoteExecutor_NullArguments_Throws", "RemoteExecutor",
		"cross-process + DisableDynamicCode; no Go equivalent (AD-6)"},
	{"CreateFactory_RemoteExecutor_NoArguments_UseNullDefaultValue", "RemoteExecutor",
		"cross-process + DisableDynamicCode; C# optional default-value params have no Go ctor equivalent (AD-6)"},
	{"CreateFactory_RemoteExecutor_NoArguments_ThrowRequiredValue", "RemoteExecutor",
		"cross-process + DisableDynamicCode; required-value error path (AD-6)"},
	{"CreateFactory_RemoteExecutor_NullArgument_UseDefaultValue", "RemoteExecutor",
		"cross-process + DisableDynamicCode; C# default-value param (AD-6)"},
	{"CreateFactory_RemoteExecutor_NoParameters_Success", "RemoteExecutor",
		"cross-process + DisableDynamicCode; no-param factory (AD-6)"},

	// CollectibleAssembly ×2 — .NET AssemblyLoadContext (collectible assemblies).
	{"CreateInstance_CollectibleAssembly", "CollectibleAssembly",
		"Go has no collectible/unloadable assembly concept (AssemblyLoadContext); also runs under RemoteExecutor (AD-6)"},
	{"CreateInstance_NormalAssembly", "CollectibleAssembly",
		"control group for CollectibleAssembly; same RemoteExecutor + no-unloadable-assembly wall (AD-6)"},
}

// TestActUtil_ConditionalTheoryNASummary archives the 12 ConditionalTheory
// methods by category. They are counted here (for audit) but NOT added to the
// 20-method ledger or the 220 grand total.
func TestActUtil_ConditionalTheoryNASummary(t *testing.T) {
	const expectedConditional = 12

	if len(dotnetActUtilConditionalTheories) != expectedConditional {
		t.Fatalf("expected %d ConditionalTheory methods, got %d",
			expectedConditional, len(dotnetActUtilConditionalTheories))
	}

	categories := map[string]int{}
	for _, c := range dotnetActUtilConditionalTheories {
		if c.Anchor == "" {
			t.Errorf("ConditionalTheory %s has no architectural anchor (NFR7: N/A must be justified, not silently dropped)",
				c.DotnetMethod)
		}
		categories[c.Category]++
	}

	// Expected split: KeyedParams ×4 + RemoteExecutor ×6 + CollectibleAssembly ×2.
	want := map[string]int{"KeyedParams": 4, "RemoteExecutor": 6, "CollectibleAssembly": 2}
	for cat, n := range want {
		if categories[cat] != n {
			t.Errorf("ConditionalTheory category %q = %d, want %d", cat, categories[cat], n)
		}
	}

	for cat, n := range categories {
		t.Logf("ConditionalTheory N/A category: %s = %d (NOT counted in 220)", cat, n)
	}
	t.Logf("ActivatorUtilities(impl) ConditionalTheory total: %d (Go-N/A, excluded from ledger)", expectedConditional)
}
