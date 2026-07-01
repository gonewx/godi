// Story 6.4 (AC5) — threshold gate assertion:
// 55 dotnet extension/collection test methods × godi coverage mapping.
//
// Source files (5):
//   - ServiceProviderServiceExtensionsTest (SPSE, 17)
//   - ServiceCollectionServiceExtensionsTest (SCSE, 14)
//   - ServiceCollectionKeyedServiceExtensionsTest (SCKSE, 19)
//   - ServiceCollectionDescriptorExtensionsTests (SCDE, 4)
//   - ServiceCollectionTests (SCT, 1)
//
// Counting convention (aligned with threshold_116_test.go / threshold_spct_test.go):
//
//	covered  — directly mapped to a godi test
//	adapted  — rewritten for Go差異 but covers same semantic intent
//	na       — Go mechanism wall / deferred scope (with mandatory justification)
//	Formula: covered + adapted + na = 55 (no MISSING gap)
//
// Reuses: coverageStatus/covered/adapted/na (threshold_116_test.go:8-13),
// contains/searchString (threshold_spct_test.go:181-192). DO NOT redefine.
package conformance

import "testing"

type extMethodMapping struct {
	ID           string
	DotnetMethod string
	DotnetFile   string // SPSE | SCSE | SCKSE | SCDE | SCT
	DotnetType   string // Fact | Theory | Theory(N)
	GodiTest     string
	Status       coverageStatus
	Note         string
}

var dotnetExtMethodMap = []extMethodMapping{
	// ══ ServiceProviderServiceExtensionsTest (SPSE, 17 Fact) ═════════════════

	{ID: "SPSE-01", DotnetMethod: "GetService_Returns_CorrectService", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: GetService<T> returns correct service (dual provider)", Status: covered},
	{ID: "SPSE-02", DotnetMethod: "ISupportRequiredService_GetRequiredService_Returns_CorrectService", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "", Status: na, Note: "Go no ISupportRequiredService optional extension point interface (D-7)"},
	{ID: "SPSE-03", DotnetMethod: "GetRequiredService_Throws_WhenNoServiceRegistered", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: GetRequired[T] panics with type name when no service registered (dual provider)", Status: covered},
	{ID: "SPSE-04", DotnetMethod: "ISupportRequiredService_GetRequiredService_Throws_WhenNoServiceRegistered", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "", Status: na, Note: "Go no ISupportRequiredService; RankException transparent propagation adapted to panic/error value propagation (D-7)"},
	{ID: "SPSE-05", DotnetMethod: "NonGeneric_GetRequiredService_Throws_WhenNoServiceRegistered", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: GetRequired[T] panic (non-generic = same path in Go)", Status: adapted, Note: "Go has no non-generic overload; Provider.GetService returns nil, GetRequired[T] wraps it"},
	{ID: "SPSE-06", DotnetMethod: "ISupportRequiredService_NonGeneric_GetRequiredService_Throws_WhenNoServiceRegistered", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "", Status: na, Note: "Go no ISupportRequiredService (D-7)"},
	{ID: "SPSE-07", DotnetMethod: "GetServices_Returns_AllServices", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: GetServices[T] returns all registered services (dual provider)", Status: covered},
	{ID: "SPSE-08", DotnetMethod: "NonGeneric_GetServices_Returns_AllServices", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: Provider.GetKeyedServices(t,nil) = non-generic path (existing)", Status: adapted, Note: "non-generic = Provider.GetKeyedServices(t,nil), already tested in spec_multiimpl"},
	{ID: "SPSE-09", DotnetMethod: "GetServices_Returns_SingleService", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: GetServices[T] returns single-element slice for one registration", Status: covered},
	{ID: "SPSE-10", DotnetMethod: "NonGeneric_GetServices_Returns_SingleService", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: Provider.GetKeyedServices(t,nil) single service (existing path)", Status: adapted, Note: "non-generic path tested via existing Provider interface"},
	{ID: "SPSE-11", DotnetMethod: "GetServices_Returns_CorrectTypes", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: GetServices[T] returns correct dynamic types", Status: covered},
	{ID: "SPSE-12", DotnetMethod: "GetServices_Returns_EmptyArray_WhenNoServicesAvailable", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: GetServices[T] returns empty non-nil slice for unregistered type", Status: covered},
	{ID: "SPSE-13", DotnetMethod: "NonGeneric_GetServices_Returns_EmptyArray_WhenNoServicesAvailable", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: Provider.GetKeyedServices(t,nil) returns nil for unregistered", Status: adapted, Note: "Go nil slice vs dotnet empty array; Provider returns nil, GetServices[T] adapts to empty non-nil"},
	{ID: "SPSE-14", DotnetMethod: "GetServices_WithBuildServiceProvider_Returns_EmptyList_WhenNoServicesAvailable", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: GetServices[T] returns empty slice from fresh Build provider", Status: covered},
	{ID: "SPSE-15", DotnetMethod: "NonGeneric_GetServices_WithBuildServiceProvider_Returns_EmptyList_WhenNoServicesAvailable", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "spec_extensions: Provider.GetKeyedServices fresh build returns nil", Status: adapted, Note: "same non-generic adaptation as SPSE-13"},
	{ID: "SPSE-16", DotnetMethod: "CreateAsyncScope_Returns_AsyncServiceScope_Wrapping_ServiceScope", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "async_scope: CreateAsyncScope wraps scope (Story 6-3, async.go:75)", Status: covered, Note: "already implemented and tested in Story 6-3"},
	{ID: "SPSE-17", DotnetMethod: "CreateAsyncScope_Returns_AsyncServiceScope_Wrapping_ServiceScope_For_IServiceScopeFactory", DotnetFile: "SPSE", DotnetType: "Fact",
		GodiTest: "", Status: na, Note: "Go no IServiceScopeFactory injectable service; godi uses Provider.CreateScope() (deferred 2-5 D-2, D-7)"},

	// ══ ServiceCollectionServiceExtensionsTest (SCSE, 14: 5 Fact + 9 Theory) ═

	{ID: "SCSE-01", DotnetMethod: "AddWithTypeAddsServiceWithRightLifecyle", DotnetFile: "SCSE", DotnetType: "Theory(3)",
		GodiTest: "spec_lifetimes + spec_extensions: lifetime observable through resolve behavior", Status: adapted, Note: "descriptor Lifetime property → observable resolve (singleton=shared, scoped=per-scope, transient=each-new)"},
	{ID: "SCSE-02", DotnetMethod: "AddWithFactoryAddsServiceWithRightLifecyle", DotnetFile: "SCSE", DotnetType: "Theory(3)",
		GodiTest: "spec_lifetimes: factory registration lifetime verified through observable resolve", Status: adapted, Note: "form-B factory registration; descriptor property → observable resolve"},
	{ID: "SCSE-03", DotnetMethod: "AddSingleton_AddsWithSingletonLifecycle", DotnetFile: "SCSE", DotnetType: "Theory",
		GodiTest: "spec_extensions: instance registration resolves same pointer (singleton observable)", Status: adapted, Note: "descriptor.ImplementationType property → GetService dynamic type; instance singleton verified via identity"},
	{ID: "SCSE-04", DotnetMethod: "TryAddNoOpFailsIfExists", DotnetFile: "SCSE", DotnetType: "Theory(3)",
		GodiTest: "collection_ops: TryAdd no-op when service type already registered", Status: covered},
	{ID: "SCSE-05", DotnetMethod: "TryAdd_WithType_AddsService", DotnetFile: "SCSE", DotnetType: "Theory(3)",
		GodiTest: "collection_ops: TryAdd adds service when type not yet registered", Status: covered},
	{ID: "SCSE-06", DotnetMethod: "TryAdd_WithType_DoesNotAddDuplicate", DotnetFile: "SCSE", DotnetType: "Theory(3)",
		GodiTest: "collection_ops: TryAdd does not add duplicate (same ServiceType)", Status: covered},
	{ID: "SCSE-07", DotnetMethod: "TryAddIfMissingActuallyAdds", DotnetFile: "SCSE", DotnetType: "Fact",
		GodiTest: "collection_ops: TryAdd adds when missing, resolvable after Build", Status: covered},
	{ID: "SCSE-08", DotnetMethod: "TryAddEnumerable_AddsService", DotnetFile: "SCSE", DotnetType: "Theory(3)",
		GodiTest: "collection_ops: TryAddEnumerable adds service when ServiceType not registered", Status: adapted, Note: "Go form-B factory cannot statically distinguish concrete impl types without invoking the factory, so dedup degenerates from dotnet's (ServiceType,ImplType,Key) to (ServiceType,Key) — adds-when-missing is verified, but the ImplType dimension of dotnet's dedup key has no Go equivalent (EXTENSIONS_DIVERGENCE.md EXT-D5/EXT-D9)"},
	{ID: "SCSE-09", DotnetMethod: "TryAddEnumerable_DoesNotAddDuplicate", DotnetFile: "SCSE", DotnetType: "Theory(3)",
		GodiTest: "collection_ops: TryAddEnumerable does not re-add an identical registration", Status: adapted, Note: "only verified for a second call registering the SAME impl (degenerate ServiceType-only dedup); registering a genuinely DIFFERENT implementation is also (incorrectly, relative to dotnet) treated as a duplicate and dropped — this is the real behavioral cost of EXT-D9, not a semantic equivalence"},
	{ID: "SCSE-10", DotnetMethod: "TryAddEnumerable_ThrowsWhenAddingIndistinguishableImplementationType", DotnetFile: "SCSE", DotnetType: "Theory(3)",
		GodiTest: "collection_ops: TryAddEnumerable indistinguishable impl adapted", Status: adapted, Note: "Go form-B cannot see the concrete impl type without invoking the factory, so every registration is unconditionally treated as indistinguishable from same-ServiceType siblings — a stricter, always-on version of dotnet's conditional object-erasure check (EXTENSIONS_DIVERGENCE.md EXT-D9, corrects the D-7 rationale in the story text, which conflated the static service type T with the dynamic impl type)"},
	{ID: "SCSE-11", DotnetMethod: "AddSequence_AddsServicesToCollection", DotnetFile: "SCSE", DotnetType: "Fact",
		GodiTest: "collection_ops: multiple Add calls append in order (no batch AddSequence API)", Status: adapted, Note: "godi has no IEnumerable<ServiceDescriptor> batch Add; sequential calls achieve same result"},
	{ID: "SCSE-12", DotnetMethod: "Replace_AddsServiceIfServiceTypeIsNotRegistered", DotnetFile: "SCSE", DotnetType: "Fact",
		GodiTest: "collection_ops: Replace adds when no existing registration for ServiceType", Status: covered},
	{ID: "SCSE-13", DotnetMethod: "Replace_ReplacesFirstServiceWithMatchingServiceType", DotnetFile: "SCSE", DotnetType: "Fact",
		GodiTest: "collection_ops: Replace removes first match + appends new at end (order-sensitive)", Status: covered},
	{ID: "SCSE-14", DotnetMethod: "RemoveAll_RemovesAllServicesWithMatchingServiceType", DotnetFile: "SCSE", DotnetType: "Fact",
		GodiTest: "collection_ops: RemoveAll removes all non-keyed matching ServiceType", Status: covered},

	// ══ ServiceCollectionKeyedServiceExtensionsTest (SCKSE, 19: 8 Fact + 11 Theory) ═

	{ID: "SCKSE-01", DotnetMethod: "AddWithTypeAddsServiceWithRightLifecyle", DotnetFile: "SCKSE", DotnetType: "Theory(3)",
		GodiTest: "spec_keyed + spec_extensions: keyed lifetime observable through resolve", Status: adapted, Note: "keyed descriptor Lifetime property → observable keyed resolve behavior"},
	{ID: "SCKSE-02", DotnetMethod: "AddWithFactoryAddsServiceWithRightLifecyle", DotnetFile: "SCKSE", DotnetType: "Theory(3)",
		GodiTest: "spec_keyed: keyed factory registration lifetime observable", Status: adapted, Note: "keyed form-B factory; descriptor → observable resolve"},
	{ID: "SCKSE-03", DotnetMethod: "AddSingleton_AddsWithSingletonLifecycle", DotnetFile: "SCKSE", DotnetType: "Theory",
		GodiTest: "spec_extensions: keyed instance singleton verified via identity", Status: adapted, Note: "keyed descriptor property → keyed singleton resolve identity"},
	{ID: "SCKSE-04", DotnetMethod: "TryAddNoOpFailsIfExists", DotnetFile: "SCKSE", DotnetType: "Theory(3)",
		GodiTest: "collection_ops: keyed TryAdd no-op when (ServiceType,key) exists", Status: covered},
	{ID: "SCKSE-05", DotnetMethod: "TryAdd_WithType_AddsService", DotnetFile: "SCKSE", DotnetType: "Theory(3)",
		GodiTest: "collection_ops: keyed TryAdd adds when (ServiceType,key) missing", Status: covered},
	{ID: "SCKSE-06", DotnetMethod: "TryAdd_WithType_DoesNotAddDuplicate", DotnetFile: "SCKSE", DotnetType: "Theory(3)",
		GodiTest: "collection_ops: keyed TryAdd does not add duplicate (ServiceType,key)", Status: covered},
	{ID: "SCKSE-07", DotnetMethod: "TryAddIfMissingActuallyAdds", DotnetFile: "SCKSE", DotnetType: "Fact",
		GodiTest: "collection_ops: keyed TryAdd adds and resolves when missing", Status: covered},
	{ID: "SCKSE-08", DotnetMethod: "TryAddEnumerable_AddsService", DotnetFile: "SCKSE", DotnetType: "Theory(3)",
		GodiTest: "", Status: na, Note: "no TryAddKeyedEnumerable function exists in runtime/collection.go; Decker scope decision 2026-07-01 (code review of story-6.4): not implemented, honestly archived as N/A rather than claiming false coverage (EXTENSIONS_DIVERGENCE.md EXT-D9)"},
	{ID: "SCKSE-09", DotnetMethod: "TryAddEnumerable_DoesNotAddDuplicate", DotnetFile: "SCKSE", DotnetType: "Theory(3)",
		GodiTest: "", Status: na, Note: "no TryAddKeyedEnumerable function exists; same scope decision as SCKSE-08 (EXTENSIONS_DIVERGENCE.md EXT-D9)"},
	{ID: "SCKSE-10", DotnetMethod: "TryAddEnumerable_ThrowsWhenAddingIndistinguishableImplementationType", DotnetFile: "SCKSE", DotnetType: "Theory",
		GodiTest: "", Status: na, Note: "no TryAddKeyedEnumerable function exists; same scope decision as SCKSE-08 (EXTENSIONS_DIVERGENCE.md EXT-D9)"},
	{ID: "SCKSE-11", DotnetMethod: "AddSequence_AddsServicesToCollection", DotnetFile: "SCKSE", DotnetType: "Fact",
		GodiTest: "collection_ops: multiple keyed Add calls append in order", Status: adapted, Note: "no batch AddSequence API; sequential AddKeyed* calls"},
	{ID: "SCKSE-12", DotnetMethod: "Replace_AddsServiceIfServiceTypeIsNotRegistered", DotnetFile: "SCKSE", DotnetType: "Fact",
		GodiTest: "collection_ops: keyed Replace adds when (ServiceType,key) missing", Status: covered},
	{ID: "SCKSE-13", DotnetMethod: "Replace_ReplacesFirstServiceWithMatchingServiceType", DotnetFile: "SCKSE", DotnetType: "Fact",
		GodiTest: "collection_ops: keyed Replace removes first (ServiceType,key) match + appends", Status: covered},
	{ID: "SCKSE-14", DotnetMethod: "Replace_ReplacesFirstServiceWithMatchingServiceTypeWhenKeyIsInt", DotnetFile: "SCKSE", DotnetType: "Fact",
		GodiTest: "collection_ops: keyed Replace with int key (key=any comparable, AD-17)", Status: covered},
	{ID: "SCKSE-15", DotnetMethod: "RemoveAll_RemovesAllServicesWithMatchingServiceType", DotnetFile: "SCKSE", DotnetType: "Fact",
		GodiTest: "collection_ops: RemoveAllKeyed removes all (ServiceType,key) matches", Status: covered},
	{ID: "SCKSE-16", DotnetMethod: "RemoveAll_RemovesAllMatchingServicesWhenKeyIsEnum", DotnetFile: "SCKSE", DotnetType: "Fact",
		GodiTest: "collection_ops: TestRemoveAllKeyed_WithEnumKey — RemoveAllKeyed with a genuine named-constant Go enum key (collKeyKind), not an int literal standing in for one (code review of story-6.4, 2026-07-01)", Status: covered},
	{ID: "SCKSE-17", DotnetMethod: "RemoveAll_RemovesAllMatchingServicesWhenKeyIsInt", DotnetFile: "SCKSE", DotnetType: "Fact",
		GodiTest: "collection_ops: RemoveAllKeyed with int key", Status: covered},
	{ID: "SCKSE-18", DotnetMethod: "NullServiceKey_IsKeyedServiceFalse", DotnetFile: "SCKSE", DotnetType: "Theory",
		GodiTest: "spec_keyed_isservice: IsKeyedService(t,nil)==IsService(t) (provider.go:40-51, Story 3-4 AC2)", Status: covered, Note: "null key routes to unkeyed; already implemented and tested"},
	{ID: "SCKSE-19", DotnetMethod: "NotNullServiceKey_IsKeyedServiceTrue", DotnetFile: "SCKSE", DotnetType: "Theory",
		GodiTest: "spec_keyed_isservice: non-null key registered → IsKeyedService true (Story 3-4)", Status: covered, Note: "already tested in keyed IsService conformance"},

	// ══ ServiceCollectionDescriptorExtensionsTests (SCDE, 4: 3 Fact + 1 Theory) ═

	{ID: "SCDE-01", DotnetMethod: "Add_AddsDescriptorToServiceDescriptors", DotnetFile: "SCDE", DotnetType: "Fact",
		GodiTest: "collection_ops: runtime.Add* appends registration to builder", Status: adapted, Note: "godi has no standalone ServiceDescriptor object; runtime.Add* is the append mechanism"},
	{ID: "SCDE-02", DotnetMethod: "Add_AddsMultipleDescriptorToServiceDescriptors", DotnetFile: "SCDE", DotnetType: "Fact",
		GodiTest: "collection_ops: multiple runtime.Add* calls all take effect", Status: adapted, Note: "no descriptor list API; multiple Add calls verified via multi-impl resolve"},
	{ID: "SCDE-03", DotnetMethod: "ServiceDescriptors_AllowsRemovingPreviousRegisteredServices", DotnetFile: "SCDE", DotnetType: "Fact",
		GodiTest: "collection_ops: Remove removes registrations from builder", Status: adapted, Note: "dotnet Remove(descriptor) removes ONE registration by reference; godi has no standalone descriptor object to reference, so Remove[T] removes ALL unkeyed registrations of T (== RemoveAll[T]) — divergence only observable when multiple registrations of T coexist (EXTENSIONS_DIVERGENCE.md EXT-D10)"},
	{ID: "SCDE-04", DotnetMethod: "ServiceDescriptor_ToString", DotnetFile: "SCDE", DotnetType: "Theory",
		GodiTest: "", Status: na, Note: "godi has no formattable ServiceDescriptor object; no ToString equivalent (D-7)"},

	// ══ ServiceCollectionTests (SCT, 1 Fact) ═════════════════════════════════

	{ID: "SCT-01", DotnetMethod: "TestMakeReadOnly", DotnetFile: "SCT", DotnetType: "Fact",
		GodiTest: "collection_ops: MakeReadOnly freezes builder; mutations after freeze return error", Status: covered},
}

func TestExt_ThresholdGate_55Methods(t *testing.T) {
	const expectedTotal = 55

	if len(dotnetExtMethodMap) != expectedTotal {
		t.Fatalf("EXT mapping declares %d methods, want %d — update map or expectedTotal",
			len(dotnetExtMethodMap), expectedTotal)
	}

	var coveredCount, adaptedCount, naCount, missingCount int
	for _, m := range dotnetExtMethodMap {
		switch m.Status {
		case covered:
			coveredCount++
		case adapted:
			adaptedCount++
		case na:
			naCount++
		default:
			missingCount++
			t.Errorf("EXT MISSING: %s (%s.%s) — status=%q", m.ID, m.DotnetFile, m.DotnetMethod, m.Status)
		}
	}

	total := coveredCount + adaptedCount + naCount
	if total != expectedTotal {
		t.Errorf("EXT gap: covered(%d) + adapted(%d) + na(%d) = %d, want %d — %d MISSING",
			coveredCount, adaptedCount, naCount, total, expectedTotal, expectedTotal-total)
	}
	if missingCount > 0 {
		t.Fatalf("EXT has %d methods with invalid/missing status", missingCount)
	}

	t.Logf("EXT threshold: covered=%d adapted=%d na=%d total=%d (effective coverage = %d/%d)",
		coveredCount, adaptedCount, naCount, total,
		coveredCount+adaptedCount, expectedTotal-naCount)
}

func TestExt_NACategorySummary(t *testing.T) {
	categories := map[string]int{
		"ISupportRequiredService":    0,
		"IServiceScopeFactory":       0,
		"ServiceDescriptor.ToString": 0,
		"C# type-erasure object":     0,
	}

	for _, m := range dotnetExtMethodMap {
		if m.Status != na {
			continue
		}
		switch {
		case contains(m.Note, "ISupportRequiredService"):
			categories["ISupportRequiredService"]++
		case contains(m.Note, "IServiceScopeFactory"):
			categories["IServiceScopeFactory"]++
		case contains(m.Note, "ToString"):
			categories["ServiceDescriptor.ToString"]++
		case contains(m.Note, "type-erasure") || contains(m.Note, "object erasure"):
			categories["C# type-erasure object"]++
		}
	}

	for cat, count := range categories {
		t.Logf("N/A category: %s = %d", cat, count)
	}
}

func TestExt_FileGroupCounts(t *testing.T) {
	groups := map[string]struct{ covered, adapted, na int }{
		"SPSE":  {},
		"SCSE":  {},
		"SCKSE": {},
		"SCDE":  {},
		"SCT":   {},
	}

	for _, m := range dotnetExtMethodMap {
		g := groups[m.DotnetFile]
		switch m.Status {
		case covered:
			g.covered++
		case adapted:
			g.adapted++
		case na:
			g.na++
		}
		groups[m.DotnetFile] = g
	}

	for file, g := range groups {
		total := g.covered + g.adapted + g.na
		t.Logf("%-5s: covered=%d adapted=%d na=%d total=%d", file, g.covered, g.adapted, g.na, total)
	}
}
