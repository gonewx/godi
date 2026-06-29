// ATDD scaffold for Story 4.4 (AC1-AC4, Task 3) — codegen Path γ:
// open-generic IsService + IsKeyedService run STRICTLY on the pre-generated
// B2 container (codegen.NewRoot()), not only via cross-provider differential
// equivalence.
//
// Path γ (AD-15): open-generic closed materialization is an AD-15 升格 axis —
// the generated container's IsService/IsKeyedService behavior for materialized
// vs. unmaterialized closed generic types must be exercised directly here.
//
// Core insight: materialized closed types ARE ordinary registered types in the
// B2 container's IsService switch-case / IsKeyedService keyed-switch. No
// special open-generic logic exists — the test locks this architectural
// property against future regressions.
package codegen_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
)

// --- 4.4-CGPATH-001 (AC1, P0) — B2 IsService true for materialized closed ---
//
// On the B2 container, IsService returns true for the materialized closed
// generic type IFakeOpenGenericService[IFakeService] (registered as a closed
// generic transient in container_gen.go via Story 4.1 registration).
func TestCodegen_PathGamma_OpenGenericIsService_MaterializedTrue(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	testutil.True(t,
		root.IsService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]()),
		"B2 IsService(IFakeOpenGenericService[IFakeService]) must be true — materialized closed = ordinary registered (touchpoint 9)")
}

// --- 4.4-CGPATH-002 (AC2, P0) — B2 IsService false for unmaterialized -------
//
// On the B2 container, IsService returns false for an unmaterialized closed
// type (IFakeOpenGenericService[int] was never enumerated in registration).
// "Only query, never construct" — the B2 container has no entry for this type.
func TestCodegen_PathGamma_OpenGenericIsService_UnmaterializedFalse(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	testutil.True(t,
		!root.IsService(reflect.TypeFor[fakes.IFakeOpenGenericService[int]]()),
		"B2 IsService(IFakeOpenGenericService[int]) must be false — unmaterialized, only-query-never-construct (AD-4)")
}

// --- 4.4-CGPATH-003 (AC3, P0) — B2 IsKeyedService true for correct key ------
//
// On the B2 container, IsKeyedService returns true for the materialized closed
// generic type under its registered key "gen-key" (registered in Story 3-4 AC4).
func TestCodegen_PathGamma_OpenGenericIsKeyedService_CorrectKeyTrue(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	testutil.True(t,
		root.IsKeyedService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]](), "gen-key"),
		"B2 IsKeyedService(IFakeOpenGenericService[IFakeService], \"gen-key\") must be true — materialized closed keyed (keyed touchpoint 5)")
}

// --- 4.4-CGPATH-004 (AC3 negative, P0) — B2 IsKeyedService false for wrong --
//
// On the B2 container, IsKeyedService returns false for the materialized closed
// type under a wrong key (no registration exists under "wrong-key").
func TestCodegen_PathGamma_OpenGenericIsKeyedService_WrongKeyFalse(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	testutil.True(t,
		!root.IsKeyedService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]](), "wrong-key"),
		"B2 IsKeyedService(IFakeOpenGenericService[IFakeService], \"wrong-key\") must be false — wrong key")
}
