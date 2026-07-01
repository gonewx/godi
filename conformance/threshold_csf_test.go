// ATDD scaffold for Story 6.2 (AC1) — CSF threshold gate assertion:
// CallSiteFactoryTest (30 [Fact]/[Theory] + 4 [ConditionalFact] = 34) +
// CallSiteTests (13 [Fact]/[Theory]) = 47 dotnet methods × godi coverage mapping.
//
// Core positioning (D-1): CallSiteFactoryTest / CallSiteTests are WHITE-BOX tests
// of dotnet's internal ServiceCallSite IR (ConstructorCallSite / CacheLocation /
// GetSlot / ExpressionResolverBuilder). godi has NO CallSite IR by construction
// (AD-1: B2 codegen direct assembly + reflect-free runtime interpreter; lifecycle
// encoded by field position, not a cache flag). So this story is "semantic
// re-expression + strict archiving", NOT a new IR implementation. The honesty
// gate (NFR7) red line: NEVER fabricate a runtime CallSite/slot/cache
// representation to make a white-box test "pass".
//
// Counting convention (aligned with threshold_116_test.go / threshold_m2_test.go /
// threshold_spct_test.go):
//
//	covered  — directly mapped to a godi test
//	adapted  — rewritten for Go差異 but covers same semantic intent
//	na       — Go mechanism wall / architectural boundary (with mandatory justification)
//	Formula: covered + adapted + na = 47 (no MISSING gap)
//
// Estimate (Task 1): covered 0, adapted 26, na 21 = 47; effective coverage =
// (covered+adapted)/(47-na) = 26/26 = 100%.
//
// ACTIVATED (Story 6.2): the build tag was removed once Task 1 finalized the
// 47-method mapping and Tasks 2-5 landed the observable tests in
// spec_callsite_redphase_test.go — this gate is now GREEN on the default build.
// See _bmad-output/test-artifacts/atdd-checklist-6-2-callsite-construction.md.
//
// Reuses coverageStatus / covered / adapted / na (threshold_116_test.go) and
// contains (threshold_spct_test.go) — same conformance package.
package conformance

import "testing"

type csfMethodMapping struct {
	ID           string
	DotnetMethod string
	DotnetType   string // Fact | Theory(N) | ConditionalFact
	GodiTest     string
	Status       coverageStatus
	Note         string
}

// dotnetCSFMethodMap enumerates all 47 methods (CSF-01..30 + CSF-C1..C4 +
// CST-01..13). Every na entry carries an architectural anchor (AD-1/AD-4/AD-9/
// AD-10/AD-13/NFR8) so TestCSF_NACategorySummary can classify it.
var dotnetCSFMethodMap = []csfMethodMapping{
	// ══ CallSiteFactoryTest (CSF-01..30 + CSF-C1..C4 = 34) ══════════════════

	// ── Group A: factory / constructor call site shape (CSF-01..05) ──────────
	{ID: "CSF-01", DotnetMethod: "GetService_FactoryCallSite_Transient_DoesNotFail", DotnetType: "Fact", GodiTest: "spec_callsite: CST-01 analog / form-B transient factory resolves stable concrete type", Status: adapted, Note: "Go has no background JIT warm-up loop; form-B transient factory resolves a stable concrete type"},
	{ID: "CSF-02", DotnetMethod: "CreateCallSite_Throws_IfTypeHasNoPublicConstructors", DotnetType: "Fact", GodiTest: "cmd/godi-gen: TestGenerator_FormA_NoNewImpl (Story 2.1 gen-time gate)", Status: adapted, Note: "Go has no private ctor; form-A missing New{Impl} → gen-time error + form-B hint (AD-8)"},
	{ID: "CSF-03", DotnetMethod: "CreateCallSite_CreatesInstanceCallSite_IfTypeHasDefaultOrPublicParameterlessConstructor", DotnetType: "Theory(3)", GodiTest: "spec_test/spec_lifetimes: parameterless form-A service resolves", Status: adapted, Note: "parameterless form-A resolvable; ConstructorCallSite/Cache.Location IR assertion has no godi equivalent (AD-1)"},
	{ID: "CSF-04", DotnetMethod: "CreateCallSite_CreatesConstructorCallSite_IfTypeHasConstructorWithInjectableParameters", DotnetType: "Theory(4)", GodiTest: "spec_test: resolves IFakeConsumer with injected IFakeService dependency (Story 2.1)", Status: adapted, Note: "form-A reads deps left-to-right from New{Impl} signature (AD-8)"},
	{ID: "CSF-05", DotnetMethod: "CreateCallSite_CreatesConstructorWithEnumerableParameters", DotnetType: "Fact", GodiTest: "spec_multiimpl: form-A ctor with []IFakeService slice dependency (Story 2.4)", Status: adapted, Note: "form-A ctor with slice dependency = IEnumerable resolution (Story 2.4)"},

	// ── Group B: multi-ctor best-match = activator domain (CSF-06) ───────────
	{ID: "CSF-06", DotnetMethod: "CreateCallSite_UsesNullaryConstructorIfServicesCannotBeInjectedIntoOtherConstructors", DotnetType: "Fact", GodiTest: "", Status: na, Note: "multi-ctor nullary fallback = activator domain (AD-9); container path has a single New{Impl}"},

	// ── Group C: generic constraint satisfaction (CSF-07..19) ────────────────
	// Unsatisfied cases (07/09/11/13/15/17/18): cannot be written in Go — the
	// per-service helper only compiles for satisfying T (AD-10). Satisfied cases
	// (08/10/12/14/16/19): closed materialization resolution (adapted).
	{ID: "CSF-07", DotnetMethod: "CreateCallSite_Throws_IfClosedTypeDoesNotSatisfyStructGenericConstraint", DotnetType: "Fact", GodiTest: "", Status: na, Note: "Go comparable (struct analog) constraint blocks non-satisfying instantiation at compile-time (AD-10); 'unsatisfied→throw' cannot be written in Go"},
	{ID: "CSF-08", DotnetMethod: "CreateCallSite_ReturnsService_IfClosedTypeSatisfiesStructGenericConstraint", DotnetType: "Fact", GodiTest: "spec_callsite: CSF-08 ClassWithStructConstraint[int] closed materialization (this story)", Status: adapted, Note: "structural constraint fake consumed (CAP-7 二级靶; deferred-work.md line 77)"},
	{ID: "CSF-09", DotnetMethod: "CreateCallSite_Throws_IfClosedTypeDoesNotSatisfyClassGenericConstraint", DotnetType: "Fact", GodiTest: "", Status: na, Note: "Go class→any constraint; 'unsatisfied→throw' cannot be written in Go (compile-time, AD-10) — kind mismatch archived"},
	{ID: "CSF-10", DotnetMethod: "CreateCallSite_ReturnsService_IfClosedTypeSatisfiesClassGenericConstraint", DotnetType: "Fact", GodiTest: "spec_callsite: CSF-10 ClassWithClassConstraint[T] closed materialization (this story)", Status: adapted, Note: "structural constraint fake consumed (class→any)"},
	{ID: "CSF-11", DotnetMethod: "CreateCallSite_Throws_IfClosedTypeDoesNotSatisfyNewGenericConstraint", DotnetType: "Fact", GodiTest: "", Status: na, Note: "Go has no new() constraint (→any); 'unsatisfied→throw' cannot be written in Go (compile-time, AD-10)"},
	{ID: "CSF-12", DotnetMethod: "CreateCallSite_ReturnsService_IfClosedTypeSatisfiesNewGenericConstraint", DotnetType: "Fact", GodiTest: "spec_callsite: CSF-12 ClassWithNewConstraint[T] closed materialization (this story)", Status: adapted, Note: "structural constraint fake consumed (new→any)"},
	{ID: "CSF-13", DotnetMethod: "CreateCallSite_Throws_IfClosedTypeDoesNotSatisfyInterfaceGenericConstraint", DotnetType: "Fact", GodiTest: "", Status: na, Note: "Go interface constraint (IterableLike) blocks non-satisfying instantiation at compile-time (AD-10)"},
	{ID: "CSF-14", DotnetMethod: "CreateCallSite_ReturnsService_IfClosedTypeSatisfiesInterfaceGenericConstraint", DotnetType: "Fact", GodiTest: "spec_opengeneric_constrained: interface constraint [IClassImplementingIterable] (Story 4.2, confirmed)", Status: adapted, Note: "already covered by Story 4.2 InterfaceConstrainedOpenGenericServicesCanBeResolved — mapping reference only, no re-registration"},
	{ID: "CSF-15", DotnetMethod: "CreateCallSite_Throws_IfClosedTypeDoesNotSatisfyAbstractClassGenericConstraint", DotnetType: "Fact", GodiTest: "", Status: na, Note: "Go sealed-interface (abstract analog) constraint blocks at compile-time (AD-10)"},
	{ID: "CSF-16", DotnetMethod: "CreateCallSite_ReturnsService_IfClosedTypeSatisfiesAbstractClassGenericConstraint", DotnetType: "Fact", GodiTest: "spec_opengeneric_constrained: abstract-class constraint [IClassInheritingAbstract] (Story 4.2, confirmed)", Status: adapted, Note: "already covered by Story 4.2 AbstractClassConstrainedOpenGenericServicesCanBeResolved — mapping reference only"},
	{ID: "CSF-17", DotnetMethod: "CreateCallSite_Throws_IfClosedTypeDoesNotSatisfySelfReferencingConstraint", DotnetType: "Fact", GodiTest: "", Status: na, Note: "Go self-referencing constraint interface{CompareTo(T) int} blocks at compile-time (AD-10)"},
	{ID: "CSF-18", DotnetMethod: "CreateCallSite_Throws_IfComplexClosedTypeDoesNotSatisfySelfReferencingConstraint", DotnetType: "Fact", GodiTest: "", Status: na, Note: "complex-type self-referencing constraint blocks at compile-time (AD-10)"},
	{ID: "CSF-19", DotnetMethod: "CreateCallSite_ReturnsService_IfClosedTypeSatisfiesSelfReferencing", DotnetType: "Fact", GodiTest: "spec_callsite: CSF-19 ClassWithSelfReferencingConstraint[selfComparableKey] closed materialization (this story)", Status: adapted, Note: "structural constraint fake consumed (self-referencing; needs a T with CompareTo(T) int)"},

	// ── Group D: base-class constraint collection filtering (CSF-20..22) ─────
	{ID: "CSF-20", DotnetMethod: "CreateCallSite_ReturnsEmpty_IfClosedTypeSatisfiesBaseClassConstraintButRegisteredTypeNotExactMatch", DotnetType: "Fact", GodiTest: "spec_callsite: CSF-20 unmaterialized closed type → empty collection (this story)", Status: adapted, Note: "godi only materializes exact closed types; unmaterialized → empty (AD-4)"},
	{ID: "CSF-21", DotnetMethod: "CreateCallSite_ReturnsMatchingTypes_IfClosedTypeSatisfiesBaseClassConstraintAndRegisteredType", DotnetType: "Fact", GodiTest: "spec_callsite: CSF-21 materialized closed-type collection in registration order (this story)", Status: adapted, Note: "materialized closed-type collection resolves in registration order (Story 4.3 slot-order)"},
	{ID: "CSF-22", DotnetMethod: "CreateCallSite_ReturnsMatchingTypesThatMatchCorrectConstraints", DotnetType: "Theory(3)", GodiTest: "spec_callsite: CSF-22 observable collection subset (this story) + CALLSITE_DIVERGENCE.md", Status: adapted, Note: "observable collection (only materialized closed types) adapted; exact dotnet constraint matrix (struct/class/new/interface/abstract ≠ Go comparable/any/interface) is a documented divergence, NOT reproduced (AD-10)"},

	// ── Group E: multi-ctor selection = activator domain (CSF-23) ────────────
	{ID: "CSF-23", DotnetMethod: "CreateCallSite_PicksConstructorWithTheMostNumberOfResolvedParameters", DotnetType: "Theory(8)", GodiTest: "", Status: na, Note: "best-match most-resolvable-params = activator domain (AD-9); container path single New{Impl}"},

	// ── Group F: default ctor param values (CSF-24) ──────────────────────────
	{ID: "CSF-24", DotnetMethod: "CreateCallSite_ConsidersConstructorsWithDefaultValues", DotnetType: "Theory(3)", GodiTest: "", Status: na, Note: "Go has no optional/default ctor parameter values"},

	// ── Group G: single unresolvable ctor = plan.Validate left-shift (CSF-25) ─
	{ID: "CSF-25", DotnetMethod: "CreateCallSite_ThrowsIfTypeHasSingleConstructorWithUnresolvableParameters", DotnetType: "Fact", GodiTest: "spec_validation: single-ctor missing dependency detected by plan.Validate (Story 5.1)", Status: adapted, Note: "single ctor with missing dep → plan.Validate left-shift (Story 5.1)"},

	// ── Group H: internal slot / cache IR (CSF-26, CSF-30) ───────────────────
	{ID: "CSF-26", DotnetMethod: "GetSlotTests", DotnetType: "Theory(5)", GodiTest: "", Status: na, Note: "dotnet GetSlot() reverse index = internal CallSite IR; godi has no slot representation (AD-1); observable registration order covered by Story 2.4/4.3"},
	{ID: "CSF-30", DotnetMethod: "CreateCallSite_EnumberableCachedAtLowestLevel", DotnetType: "Theory(3)", GodiTest: "", Status: na, Note: "dotnet Cache.Location (None/Scope/Root) = internal CallSite IR; godi lifecycle encoded by field position, no CallSite cache (AD-1)"},

	// ── Group I: multi-ctor unresolvable / ambiguity = activator (CSF-27..29) ─
	{ID: "CSF-27", DotnetMethod: "CreateCallSite_ThrowsIfTypeHasNoConstructurWithResolvableParameters", DotnetType: "Theory(2)", GodiTest: "", Status: na, Note: "multi-ctor unresolvable (Superset/MultipleParameterized) = activator domain (AD-9)"},
	{ID: "CSF-28", DotnetMethod: "CreateCallSite_ThrowsIfMultipleNonOverlappingConstructorsCanBeResolved", DotnetType: "Theory(5)", GodiTest: "", Status: na, Note: "multi-ctor ambiguity detection = activator domain (AD-9)"},
	{ID: "CSF-29", DotnetMethod: "CreateCallSite_ThrowsIfMultipleNonOverlappingConstructorsForGenericTypesCanBeResolved", DotnetType: "Fact", GodiTest: "", Status: na, Note: "multi-ctor ambiguity (generic) = activator domain (AD-9)"},

	// ── Group J: ConditionalFact concurrency / trimming / NativeAOT (C1..C4) ──
	{ID: "CSF-C1", DotnetMethod: "CallSitesAreUniquePerServiceTypeAndSlot", DotnetType: "ConditionalFact", GodiTest: "", Status: na, Note: "NFR8 concurrency + CallSite IR uniqueness (no godi equivalent)"},
	{ID: "CSF-C2", DotnetMethod: "CallSitesAreUniquePerServiceTypeAndSlotWithOpenGenericInGraph", DotnetType: "ConditionalFact", GodiTest: "", Status: na, Note: "NFR8 concurrency + CallSite IR uniqueness with open generic in graph"},
	{ID: "CSF-C3", DotnetMethod: "VerifyOpenGenericTrimmabilityChecks", DotnetType: "ConditionalFact", GodiTest: "", Status: na, Note: "dotnet trimming annotations (DynamicallyAccessedMembers); godi codegen is reflection-free/trim-safe by construction (AD-13)"},
	{ID: "CSF-C4", DotnetMethod: "VerifyDynamicCodeNotSupportedChecks", DotnetType: "ConditionalFact", GodiTest: "", Status: na, Note: "dotnet NativeAOT no-dynamic-code fallback; godi has no dynamic code — this fallback IS godi's default (AD-1/AD-13)"},

	// ══ CallSiteTests (CST-01..13 = 13) ═════════════════════════════════════
	{ID: "CST-01", DotnetMethod: "BuiltExpressionWillReturnResolvedServiceWhenAppropriate", DotnetType: "Theory", GodiTest: "spec_lifetimes: three-lifetime identity + spec_callsite: CST-01 single-element IEnumerable (this story)", Status: adapted, Note: "observable resolution + lifetime identity; expression-compilation engine itself N/A (AD-6, Runtime engine only)"},
	{ID: "CST-02", DotnetMethod: "BuiltExpressionCanResolveNestedScopedService", DotnetType: "Fact", GodiTest: "spec_callsite: CST-02 nested scoped chain A←B←C (this story)", Status: adapted, Note: "nested scoped chain resolution (Story 2.2)"},
	{ID: "CST-03", DotnetMethod: "BuildExpressionAddsDisposableCaptureForDisposableServices", DotnetType: "Theory(3)", GodiTest: "spec_callsite: CST-03 disposable capture count = 3 (this story)", Status: adapted, Note: "disposer tracks io.Closer services across three lifetimes (Story 2.3; AD-11 io.Closer gate)"},
	{ID: "CST-04", DotnetMethod: "BuildExpressionAddsDisposableCaptureForDisposableFactoryServices", DotnetType: "Theory(3)", GodiTest: "spec_callsite: CST-04 form-B factory disposable capture (this story)", Status: adapted, Note: "form-B factory output implementing io.Closer is tracked (Story 2.3/6.1)"},
	{ID: "CST-05", DotnetMethod: "BuildExpressionElidesDisposableCaptureForNonDisposableServices", DotnetType: "Theory(2)", GodiTest: "spec_callsite: CST-05 non-disposable NOT tracked (this story)", Status: adapted, Note: "godi naturally elides non-io.Closer services — types.Implements(io.Closer) gate (AD-11), a type gate not an optimization"},
	{ID: "CST-06", DotnetMethod: "BuildExpressionElidesDisposableCaptureForEnumerableServices", DotnetType: "Theory(2)", GodiTest: "spec_callsite: CST-06 non-disposable collection elements NOT tracked (this story)", Status: adapted, Note: "io.Closer gate applies to collection elements too (AD-11)"},
	{ID: "CST-07", DotnetMethod: "BuiltExpressionRethrowsOriginalExceptionFromConstructor", DotnetType: "Fact", GodiTest: "spec_callsite: CST-07 ctor error propagation (this story) = SPCT-01", Status: adapted, Note: "Go ctor returns error, propagated on resolve (ClassWithThrowingCtor / ClassWithThrowingEmptyCtor exist in fakes_activator.go)"},
	{ID: "CST-08", DotnetMethod: "DoesNotThrowWhenServiceIsUsedAsEnumerableAndNotInOneCallSite", DotnetType: "Fact", GodiTest: "spec_callsite: CST-08 service as slice element AND direct dependency (this story)", Status: adapted, Note: "a service used both as []T element and direct dependency resolves without conflict (Story 2.4)"},
	{ID: "CST-09", DotnetMethod: "NoServiceCallsite_DefaultValueNull_DoesNotThrow", DotnetType: "Theory(5 engines)", GodiTest: "", Status: na, Note: "Go has no optional ctor parameter default values + ActivatorUtilities.CreateInstance path (activator domain)"},
	{ID: "CST-10", DotnetMethod: "CallSiteFactoryResolvesIEnumerableOfOpenGenericServiceAfterResolvingClosedImplementation", DotnetType: "Fact", GodiTest: "", Status: na, Note: "dotnet known bug disabled ([ActiveIssue #57333]) — not a passing test; godi slot-order determinism (Story 4.3) has no such bug"},
	{ID: "CST-11", DotnetMethod: "ServiceCallSite_ShouldHaveKey_WhenResolvingKeyedService", DotnetType: "Fact", GodiTest: "spec_keyed: keyed service resolves to correct impl (Story 3.1)", Status: adapted, Note: "callSite.Key IR assertion N/A; observable 'keyed service resolves + correct impl' covered by Story 3.1"},
	{ID: "CST-12", DotnetMethod: "ServiceCallSite_ShouldHaveKey_WhenResolvingKeyedClosedImplementationOfOpenGenericService", DotnetType: "Fact", GodiTest: "spec_keyed + spec_opengeneric: keyed closed-generic impl resolves (Story 3.x + 4.x)", Status: adapted, Note: "keyed + closed generic impl resolution (reuses existing keyed/open-generic fakes)"},
	{ID: "CST-13", DotnetMethod: "ServiceCallSite_ShouldHaveKey_WhenResolvingKeyedIEnumerableOfClosedImplementationOfOpenGenericService", DotnetType: "Fact", GodiTest: "spec_keyed + spec_opengeneric: keyed IEnumerable of closed-generic impl (Story 3.x + 4.x)", Status: adapted, Note: "keyed IEnumerable + closed generic impl resolution (reuses existing fakes)"},
}

// TestCSF_ThresholdGate_47Methods asserts the map declares exactly 47 methods and
// that covered + adapted + na = 47 with zero MISSING (AC1).
func TestCSF_ThresholdGate_47Methods(t *testing.T) {
	const expectedTotal = 47

	if len(dotnetCSFMethodMap) != expectedTotal {
		t.Fatalf("CSF mapping declares %d methods, want %d — update map or expectedTotal",
			len(dotnetCSFMethodMap), expectedTotal)
	}

	var coveredCount, adaptedCount, naCount, missingCount int
	for _, m := range dotnetCSFMethodMap {
		switch m.Status {
		case covered:
			coveredCount++
		case adapted:
			adaptedCount++
		case na:
			naCount++
		default:
			missingCount++
			t.Errorf("CSF MISSING: %s (%s) — status=%q", m.ID, m.DotnetMethod, m.Status)
		}
	}

	total := coveredCount + adaptedCount + naCount
	if total != expectedTotal {
		t.Errorf("CSF gap: covered(%d) + adapted(%d) + na(%d) = %d, want %d — %d MISSING",
			coveredCount, adaptedCount, naCount, total, expectedTotal, expectedTotal-total)
	}
	if missingCount > 0 {
		t.Fatalf("CSF has %d methods with invalid/missing status", missingCount)
	}

	// Task 1 estimate: covered 0, adapted 26, na 21.
	if coveredCount != 0 || adaptedCount != 26 || naCount != 21 {
		t.Errorf("CSF distribution drift: covered=%d(want 0) adapted=%d(want 26) na=%d(want 21)",
			coveredCount, adaptedCount, naCount)
	}

	t.Logf("CSF threshold: covered=%d adapted=%d na=%d total=%d (effective coverage = %d/%d)",
		coveredCount, adaptedCount, naCount, total,
		coveredCount+adaptedCount, expectedTotal-naCount)
}

// TestCSF_NACategorySummary classifies the 21 na methods into the 7 architectural
// categories (Task 1) and asserts each count. First-match order matters: NFR8 and
// "optional" are checked before "activator"/"CallSite" to avoid substring overlap.
func TestCSF_NACategorySummary(t *testing.T) {
	const (
		catCompileTime = "Go compile-time constraint (AD-10)"
		catActivator   = "best-match activator domain (AD-9)"
		catCallSiteIR  = "CallSite/slot/cache IR (AD-1)"
		catNFR8        = "NFR8 concurrency + CallSite IR"
		catTrim        = "trimming/NativeAOT normalized (AD-13)"
		catOptional    = "no optional/default ctor params"
		catKnownBug    = "dotnet known-bug disabled (#57333)"
	)
	want := map[string]int{
		catCompileTime: 7,
		catActivator:   5,
		catCallSiteIR:  2,
		catNFR8:        2,
		catTrim:        2,
		catOptional:    2,
		catKnownBug:    1,
	}
	got := map[string]int{}

	classify := func(note string) string {
		switch {
		case contains(note, "NFR8"):
			return catNFR8
		case contains(note, "compile-time"):
			return catCompileTime
		case contains(note, "trimming") || contains(note, "NativeAOT"):
			return catTrim
		case contains(note, "known bug"):
			return catKnownBug
		case contains(note, "optional"):
			return catOptional
		case contains(note, "activator"):
			return catActivator
		case contains(note, "CallSite") || contains(note, "slot") || contains(note, "cache"):
			return catCallSiteIR
		default:
			return ""
		}
	}

	var unclassified int
	for _, m := range dotnetCSFMethodMap {
		if m.Status != na {
			continue
		}
		cat := classify(m.Note)
		if cat == "" {
			unclassified++
			t.Errorf("CSF na %s has no classifiable Note anchor: %q", m.ID, m.Note)
			continue
		}
		got[cat]++
	}

	for cat, w := range want {
		if got[cat] != w {
			t.Errorf("CSF na category %q = %d, want %d", cat, got[cat], w)
		}
		t.Logf("N/A category: %s = %d", cat, got[cat])
	}
	if unclassified > 0 {
		t.Fatalf("CSF has %d unclassified na methods", unclassified)
	}
}
