// ATDD scaffold for Story 6.3 (AC4) — AsyncServiceScopeTests threshold gate
// assertion: 6 dotnet Fact methods (.../DI.Tests/AsyncServiceScopeTests.cs)
// × godi coverage mapping.
//
// Unlike the other red-phase files in this story, this gate is a pure data
// table (like threshold_spct_test.go / threshold_m2_test.go) — it does not
// reference not-yet-existing symbols, so it compiles and passes immediately.
// It exists to make the 6-method AsyncServiceScopeTests coverage obligation
// (AC4) explicit and auditable, mirroring the existing threshold gates.
//
// NOTE: the SEPARATE 12-item reclassification of threshold_spct_test.go's
// na → covered/adapted entries (SPCT-17,24,25,27,28,29,31,32,33,35,38 + C1,
// AC5) is intentionally NOT done here — it is a dev-story task performed
// AFTER the corresponding godi tests are actually green (NFR7 honesty
// threshold: coverage must not be declared ahead of green tests).
//
// Counting convention (aligned with threshold_spct_test.go / threshold_m2_test.go):
//
//	covered  — directly mapped to a godi test
//	adapted  — rewritten for Go差異 but covers same semantic intent
//	na       — Go mechanism wall / deferred scope (with mandatory justification)
//	Formula: covered + adapted + na = 6 (no MISSING gap)
package conformance

import "testing"

type asyncScopeMethodMapping struct {
	ID           string
	DotnetMethod string
	GodiTest     string
	Status       coverageStatus
	Note         string
}

var dotnetAsyncScopeMethodMap = []asyncScopeMethodMapping{
	{ID: "ASYNCSCOPE-01", DotnetMethod: "ThrowsIfServiceScopeIsNull", GodiTest: "godi: TestAsyncScope_NilScope_Panics", Status: covered},
	{ID: "ASYNCSCOPE-02", DotnetMethod: "ReturnsServiceProviderFromWrappedScope", GodiTest: "godi: TestAsyncScope_Provider_ReturnsWrappedScope", Status: covered},
	{ID: "ASYNCSCOPE-03", DotnetMethod: "CallsDisposeOnWrappedSyncScopeOnDispose", GodiTest: "godi: TestAsyncScope_Close_ForwardsToWrappedSyncScope", Status: covered},
	{ID: "ASYNCSCOPE-04", DotnetMethod: "CallsDisposeOnWrappedSyncScopeOnDisposeAsync", GodiTest: "godi: TestAsyncScope_CloseAsync_FallsBackToSyncOnSyncOnlyScope", Status: covered},
	{ID: "ASYNCSCOPE-05", DotnetMethod: "CallsDisposeOnWrappedAsyncScopeOnDispose", GodiTest: "godi: TestAsyncScope_Close_OnAsyncScope_StaysSynchronous", Status: covered},
	{ID: "ASYNCSCOPE-06", DotnetMethod: "CallsDisposeAsyncOnWrappedSyncScopeOnDisposeAsync", GodiTest: "godi: TestAsyncScope_CloseAsync_PrefersAsyncOnAsyncScope", Status: covered},
}

func TestAsyncScope_ThresholdGate_6Methods(t *testing.T) {
	const expectedTotal = 6

	if len(dotnetAsyncScopeMethodMap) != expectedTotal {
		t.Fatalf("AsyncServiceScopeTests mapping declares %d methods, want %d — update map or expectedTotal",
			len(dotnetAsyncScopeMethodMap), expectedTotal)
	}

	var coveredCount, adaptedCount, naCount, missingCount int
	for _, m := range dotnetAsyncScopeMethodMap {
		switch m.Status {
		case covered:
			coveredCount++
		case adapted:
			adaptedCount++
		case na:
			naCount++
		default:
			missingCount++
			t.Errorf("AsyncServiceScopeTests MISSING: %s (%s) — status=%q", m.ID, m.DotnetMethod, m.Status)
		}
	}

	total := coveredCount + adaptedCount + naCount
	if total != expectedTotal {
		t.Errorf("AsyncServiceScopeTests gap: covered(%d) + adapted(%d) + na(%d) = %d, want %d — %d MISSING",
			coveredCount, adaptedCount, naCount, total, expectedTotal, expectedTotal-total)
	}
	if missingCount > 0 {
		t.Fatalf("AsyncServiceScopeTests has %d methods with invalid/missing status", missingCount)
	}

	t.Logf("AsyncServiceScopeTests threshold: covered=%d adapted=%d na=%d total=%d",
		coveredCount, adaptedCount, naCount, total)
}
