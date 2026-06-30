//go:build atdd_redphase

// ATDD red-phase scaffold for Story 6.1 (AC1) — SPCT threshold gate assertion:
// ServiceProviderContainerTests 44 [Fact]/[Theory] + 5 [ConditionalFact] = 49
// dotnet methods × godi coverage mapping.
//
// Counting convention (aligned with threshold_116_test.go / threshold_m2_test.go):
//
//	covered  — directly mapped to a godi test
//	adapted  — rewritten for Go差異 but covers same semantic intent
//	na       — Go mechanism wall / deferred scope (with mandatory justification)
//	Formula: covered + adapted + na = 49 (no MISSING gap)
//
// Activation: delete the //go:build line and blank line to join the default build.
package conformance

import "testing"

type spctMethodMapping struct {
	ID           string
	DotnetMethod string
	DotnetType   string // Fact | Theory(N) | ConditionalFact
	GodiTest     string
	Status       coverageStatus
	Note         string
}

var dotnetSPCTMethodMap = []spctMethodMapping{
	// ── Group 1: ctor error propagation (SPCT-01 to SPCT-04) ──────────────
	{ID: "SPCT-01", DotnetMethod: "RethrowOriginalExceptionFromConstructor", DotnetType: "Fact", GodiTest: "spec_container: SPCT-01 ctor error propagation", Status: adapted, Note: "Go ctor returns error (not throw); Build/resolve propagates"},
	{ID: "SPCT-02", DotnetMethod: "DependencyWithPrivateConstructorIsIdentifiedAsPartOfException", DotnetType: "Fact", GodiTest: "spec_container: SPCT-02 missing New{Impl} detected at gen", Status: adapted, Note: "Go has no private ctor; godi form-A New{Impl} not found → gen error"},
	{ID: "SPCT-03", DotnetMethod: "AttemptingToResolveNonexistentServiceIndirectlyThrows", DotnetType: "Fact", GodiTest: "spec_container: SPCT-03 missing dependency detected by plan.Validate", Status: adapted, Note: "godi plan.Validate left-shifts missing dep detection (Story 5-1)"},
	{ID: "SPCT-04", DotnetMethod: "AttemptingToIEnumerableResolveNonexistentServiceIndirectlyThrows", DotnetType: "Fact", GodiTest: "spec_container: SPCT-04 slice dep missing detected by plan.Validate", Status: adapted, Note: "same as SPCT-03 for collection dependency"},

	// ── Group 2: multi-impl ordering (SPCT-05) ────────────────────────────
	{ID: "SPCT-05", DotnetMethod: "MultipleServicesAreOrdered", DotnetType: "Theory(6)", GodiTest: "spec_container: SPCT-05 multi-impl registration order preserved", Status: covered},

	// ── Group 3: unresolvable types / open generics (SPCT-06, SPCT-07) ────
	{ID: "SPCT-06", DotnetMethod: "CreatingServiceProviderWithUnresolvableTypesThrows", DotnetType: "Theory(5)", GodiTest: "", Status: na, Note: "Go generics enforce I/Impl assignability at compile time (AddSingleton[I,Impl])"},
	{ID: "SPCT-07", DotnetMethod: "CreatingServiceProviderWithUnresolvableOpenGenericTypesThrows", DotnetType: "Theory(2)", GodiTest: "", Status: na, Note: "godi has no open generic registration (AD-4); only closed instantiations"},

	// ── Group 4: singleton instance dispose (SPCT-08) ─────────────────────
	{ID: "SPCT-08", DotnetMethod: "DoesNotDisposeSingletonInstances", DotnetType: "Fact", GodiTest: "spec_container: SPCT-08 godi form-B factory singleton IS tracked+disposed", Status: adapted, Note: "godi has no raw instance registration; form-B factory output is tracked (differs from dotnet raw instance)"},

	// ── Group 5: optional ctor with struct args (SPCT-09, SPCT-10) ────────
	{ID: "SPCT-09", DotnetMethod: "ResolvesServiceMixedServiceAndOptionalStructConstructorArguments", DotnetType: "Fact", GodiTest: "", Status: na, Note: "Go has no optional constructor parameters or default values"},
	{ID: "SPCT-10", DotnetMethod: "ResolvesServiceMixedServiceAndOptionalStructConstructorArgumentsReliably", DotnetType: "Fact", GodiTest: "", Status: na, Note: "Go has no optional constructor parameters or default values"},

	// ── Group 6: value type services (SPCT-11) ────────────────────────────
	{ID: "SPCT-11", DotnetMethod: "ResolvesConstantValueTypeServicesCorrectly", DotnetType: "Theory(3)", GodiTest: "spec_container: SPCT-11 value type services (int/time/enum analog)", Status: adapted, Note: "Go value types boxed to any; form-B factory returns value types"},

	// ── Group 7: resolve-after-dispose (SPCT-12 to SPCT-15) ──────────────
	{ID: "SPCT-12", DotnetMethod: "RootProviderDispose_PreventsServiceResolution", DotnetType: "Fact", GodiTest: "spec_container: SPCT-12 root Close prevents resolve", Status: covered},
	{ID: "SPCT-13", DotnetMethod: "RootProviderDispose_PreventsScopeCreation", DotnetType: "Fact", GodiTest: "spec_container: SPCT-13 root Close prevents CreateScope", Status: covered},
	{ID: "SPCT-14", DotnetMethod: "RootProviderDispose_PreventsServiceResolution_InChildScope", DotnetType: "Fact", GodiTest: "spec_container: SPCT-14 root Close prevents child scope resolve", Status: covered},
	{ID: "SPCT-15", DotnetMethod: "ScopeDispose_PreventsServiceResolution", DotnetType: "Fact", GodiTest: "spec_container: SPCT-15 scope Close prevents scope resolve + root still works", Status: covered},

	// ── Group 8: ctor-dispose-in-flight (SPCT-16) ─────────────────────────
	{ID: "SPCT-16", DotnetMethod: "GetService_DisposeOnSameThread_Throws", DotnetType: "Fact", GodiTest: "spec_container: SPCT-16 factory calls Close → subsequent resolve panics", Status: adapted, Note: "godi form-B factory calls p.Close() → closed guard triggers"},

	// ── Group 9: async disposal (N/A → Story 6-3) ─────────────────────────
	{ID: "SPCT-17", DotnetMethod: "AddDisposablesAndAsyncDisposables_DisposeAsync_AllDisposed", DotnetType: "Theory(2)", GodiTest: "", Status: na, Note: "IAsyncDisposable → Story 6-3 (AR12)"},

	// ── Group 10: struct services (SPCT-18 to SPCT-21) ────────────────────
	{ID: "SPCT-18", DotnetMethod: "WorksWithStructServices", DotnetType: "Theory(3)", GodiTest: "spec_container: SPCT-18 struct services three lifetimes", Status: covered},
	{ID: "SPCT-19", DotnetMethod: "WorksWithFactoryStructServices", DotnetType: "Theory(3)", GodiTest: "spec_container: SPCT-19 factory struct services", Status: covered},
	{ID: "SPCT-20", DotnetMethod: "WorksWithFactoryStructServicesAsDependencies", DotnetType: "Theory(3)", GodiTest: "spec_container: SPCT-20 factory struct as dependency", Status: covered},
	{ID: "SPCT-21", DotnetMethod: "WorksWithIEnumerableStructServices", DotnetType: "Theory(3)", GodiTest: "spec_container: SPCT-21 struct multi-impl collection", Status: covered},

	// ── Group 11: wide scoped tree (SPCT-22) ──────────────────────────────
	{ID: "SPCT-22", DotnetMethod: "WorksWithWideScopedTrees", DotnetType: "Fact", GodiTest: "spec_container: SPCT-22 wide scoped tree (20 services)", Status: covered},

	// ── Group 12: generic IEnumerable cache slot (SPCT-23) ────────────────
	{ID: "SPCT-23", DotnetMethod: "GenericIEnumerableItemCachedInTheRightSlot", DotnetType: "Fact", GodiTest: "spec_container: SPCT-23 singleton GetService == GetServices single (cache slot)", Status: covered},

	// ── Group 13: async disposal on provider/scope (SPCT-24 to SPCT-35, N/A → Story 6-3) ──
	{ID: "SPCT-24", DotnetMethod: "ProviderDisposeAsyncCallsDisposeAsyncOnServices", DotnetType: "Fact", GodiTest: "", Status: na, Note: "IAsyncDisposable → Story 6-3"},
	{ID: "SPCT-25", DotnetMethod: "ProviderDisposeAsyncPrefersDisposeAsyncOnServices", DotnetType: "Fact", GodiTest: "", Status: na, Note: "IAsyncDisposable → Story 6-3"},
	{ID: "SPCT-26", DotnetMethod: "ProviderDisposePrefersServiceDispose", DotnetType: "Fact", GodiTest: "spec_container: SPCT-26 sync Close calls io.Closer.Close on root", Status: covered},
	{ID: "SPCT-27", DotnetMethod: "ProviderDisposeThrowsWhenOnlyDisposeAsyncImplemented", DotnetType: "Fact", GodiTest: "", Status: na, Note: "IAsyncDisposable → Story 6-3"},
	{ID: "SPCT-28", DotnetMethod: "ProviderScopeDisposeAsyncCallsDisposeAsyncOnServices", DotnetType: "Fact", GodiTest: "", Status: na, Note: "IAsyncDisposable → Story 6-3"},
	{ID: "SPCT-29", DotnetMethod: "ProviderScopeDisposeAsyncPrefersDisposeAsyncOnServices", DotnetType: "Fact", GodiTest: "", Status: na, Note: "IAsyncDisposable → Story 6-3"},
	{ID: "SPCT-30", DotnetMethod: "ProviderScopeDisposePrefersServiceDispose", DotnetType: "Fact", GodiTest: "spec_container: SPCT-30 sync Close calls io.Closer.Close on scope", Status: covered},
	{ID: "SPCT-31", DotnetMethod: "ProviderScopeDisposeThrowsWhenOnlyDisposeAsyncImplemented", DotnetType: "Fact", GodiTest: "", Status: na, Note: "IAsyncDisposable → Story 6-3"},
	{ID: "SPCT-32", DotnetMethod: "ProviderAsyncScopeDisposeAsyncCallsDisposeAsyncOnServices", DotnetType: "Fact", GodiTest: "", Status: na, Note: "IAsyncDisposable → Story 6-3"},
	{ID: "SPCT-33", DotnetMethod: "ProviderAsyncScopeDisposeAsyncPrefersDisposeAsyncOnServices", DotnetType: "Fact", GodiTest: "", Status: na, Note: "IAsyncDisposable → Story 6-3"},
	{ID: "SPCT-34", DotnetMethod: "ProviderAsyncScopeDisposePrefersServiceDispose", DotnetType: "Fact", GodiTest: "spec_container: SPCT-34 sync Close on scope (dotnet CreateScope body equivalent)", Status: covered, Note: "dotnet test body actually uses CreateScope not CreateAsyncScope; equivalent to SPCT-30"},
	{ID: "SPCT-35", DotnetMethod: "ProviderAsyncScopeDisposeThrowsWhenOnlyDisposeAsyncImplemented", DotnetType: "Fact", GodiTest: "", Status: na, Note: "IAsyncDisposable → Story 6-3"},

	// ── Group 14: factory singleton dispose + instance dispose (SPCT-36, SPCT-37) ──
	{ID: "SPCT-36", DotnetMethod: "SingletonServiceCreatedFromFactoryIsDisposedWhenContainerIsDisposed", DotnetType: "Fact", GodiTest: "spec_container: SPCT-36 factory singleton disposed on root Close", Status: covered},
	{ID: "SPCT-37", DotnetMethod: "SingletonServiceCreatedFromInstanceIsNotDisposedWhenContainerIsDisposed", DotnetType: "Fact", GodiTest: "spec_container: SPCT-37 godi form-B factory = tracked+disposed (adapted)", Status: adapted, Note: "godi has no raw instance registration; form-B factory output IS tracked"},

	// ── Group 15: async dispose once (SPCT-38, N/A → Story 6-3) ───────────
	{ID: "SPCT-38", DotnetMethod: "ProviderDisposeAsyncCallsDisposeAsyncOnceOnServices", DotnetType: "Fact", GodiTest: "", Status: na, Note: "IAsyncDisposable → Story 6-3"},

	// ── Group 16: constrained open generic (SPCT-39) ──────────────────────
	{ID: "SPCT-39", DotnetMethod: "ValidateOnBuild_True_ResolvesConstrainedOpenGeneric", DotnetType: "Theory(2)", GodiTest: "spec_container: SPCT-39 constrained open generic collection resolve (Story 4.2 coverage)", Status: covered, Note: "Story 4.2 already verified; this confirms coverage in SPCT context"},

	// ── Group 17: concurrent stress (SPCT-40, N/A NFR8) ──────────────────
	{ID: "SPCT-40", DotnetMethod: "GetRequiredService_ResolveUniqueServicesConcurrently_StressTestSuccessful", DotnetType: "Fact", GodiTest: "", Status: na, Note: "NFR8: root singleton concurrent resolution safety deferred to C-2"},

	// ── Group 18: scoped-from-root after compilation (SPCT-41 to SPCT-43) ─
	{ID: "SPCT-41", DotnetMethod: "ScopedServiceResolvedFromSingletonAfterCompilation", DotnetType: "Fact", GodiTest: "spec_container: SPCT-41 root-scoped stability (multiple resolves same instance)", Status: adapted, Note: "godi has no JIT; adapted to root-scoped semantics (Story 3-2 decision)"},
	{ID: "SPCT-42", DotnetMethod: "ScopedServiceResolvedFromSingletonAfterCompilation2", DotnetType: "Fact", GodiTest: "spec_container: SPCT-42 root-scoped + generic dep stability", Status: adapted, Note: "godi root-scoped: scoped from root = root-scoped singleton-like"},
	{ID: "SPCT-43", DotnetMethod: "ScopedServiceResolvedFromSingletonAfterCompilation3", DotnetType: "Fact", GodiTest: "spec_container: SPCT-43 root-scoped nested dep stability", Status: adapted, Note: "singleton IFakeX<Aa> → scoped A → scoped Aa; all root-scoped when resolved from root"},

	// ── Group 19: keyed missing registration (SPCT-44) ────────────────────
	{ID: "SPCT-44", DotnetMethod: "ResolveKeyedServiceWithKeyedParameter_MissingRegistrationButWithUnkeyedService", DotnetType: "Fact", GodiTest: "spec_container: SPCT-44 keyed dep missing does not fall through to unkeyed", Status: adapted, Note: "godi plan.Validate detects missing keyed dep (ValidateOnBuild=always)"},

	// ── Group 20: ConditionalFact threading/concurrency (C1-C5, N/A NFR8) ─
	{ID: "SPCT-C1", DotnetMethod: "GetAsyncService_DisposeAsyncOnSameThread_ThrowsAndDoesNotHangAndDisposeAsyncGetsCalled", DotnetType: "ConditionalFact", GodiTest: "", Status: na, Note: "NFR8 concurrency + IAsyncDisposable → Story 6-3"},
	{ID: "SPCT-C2", DotnetMethod: "GetService_DisposeOnSameThread_ThrowsAndDoesNotHangAndDisposeGetsCalled", DotnetType: "ConditionalFact", GodiTest: "", Status: na, Note: "NFR8 concurrency: threading-aware dispose test"},
	{ID: "SPCT-C3", DotnetMethod: "GetRequiredService_ResolvingSameSingletonInTwoThreads_SameServiceReturned", DotnetType: "ConditionalFact", GodiTest: "", Status: na, Note: "NFR8: concurrent singleton resolution deferred to C-2"},
	{ID: "SPCT-C4", DotnetMethod: "GetRequiredService_UsesSingletonAndLazyLocks_NoDeadlock", DotnetType: "ConditionalFact", GodiTest: "", Status: na, Note: "NFR8: deadlock detection deferred to C-2"},
	{ID: "SPCT-C5", DotnetMethod: "GetRequiredService_BiggerObjectGraphWithOpenGenerics_NoDeadlock", DotnetType: "ConditionalFact", GodiTest: "", Status: na, Note: "NFR8: deadlock + open generics deferred to C-2"},
}

func TestSPCT_ThresholdGate_49Methods(t *testing.T) {
	const expectedTotal = 49

	if len(dotnetSPCTMethodMap) != expectedTotal {
		t.Fatalf("SPCT mapping declares %d methods, want %d — update map or expectedTotal",
			len(dotnetSPCTMethodMap), expectedTotal)
	}

	var coveredCount, adaptedCount, naCount, missingCount int
	for _, m := range dotnetSPCTMethodMap {
		switch m.Status {
		case covered:
			coveredCount++
		case adapted:
			adaptedCount++
		case na:
			naCount++
		default:
			missingCount++
			t.Errorf("SPCT MISSING: %s (%s) — status=%q", m.ID, m.DotnetMethod, m.Status)
		}
	}

	total := coveredCount + adaptedCount + naCount
	if total != expectedTotal {
		t.Errorf("SPCT gap: covered(%d) + adapted(%d) + na(%d) = %d, want %d — %d MISSING",
			coveredCount, adaptedCount, naCount, total, expectedTotal, expectedTotal-total)
	}
	if missingCount > 0 {
		t.Fatalf("SPCT has %d methods with invalid/missing status", missingCount)
	}

	t.Logf("SPCT threshold: covered=%d adapted=%d na=%d total=%d (effective coverage = %d/%d)",
		coveredCount, adaptedCount, naCount, total,
		coveredCount+adaptedCount, expectedTotal-naCount)
}

func TestSPCT_NACategorySummary(t *testing.T) {
	categories := map[string]int{
		"IAsyncDisposable → Story 6-3":        0,
		"NFR8 concurrency":                    0,
		"Go compile-time type safety":         0,
		"Go no optional ctor params":          0,
	}

	for _, m := range dotnetSPCTMethodMap {
		if m.Status != na {
			continue
		}
		switch {
		case contains(m.Note, "Story 6-3") || contains(m.Note, "IAsyncDisposable"):
			categories["IAsyncDisposable → Story 6-3"]++
		case contains(m.Note, "NFR8"):
			categories["NFR8 concurrency"]++
		case contains(m.Note, "compile time") || contains(m.Note, "open generic registration"):
			categories["Go compile-time type safety"]++
		case contains(m.Note, "optional"):
			categories["Go no optional ctor params"]++
		}
	}

	for cat, count := range categories {
		t.Logf("N/A category: %s = %d", cat, count)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
