// ATDD scaffold for Story 4.3 (AC3/AC5, Task 4) — codegen conformance:
// Path γ STRICT tests for open/closed priority and enumerable slot ORDER on the
// pre-generated B2 container (codegen.NewRoot() direct calls, not just cross-
// provider differential equivalence).
//
// Path γ (AD-15): open-generic materialization (including slot order / priority)
// is explicitly named a fidelity-sensitive capability that escalates to Path γ —
// it must be verified STRICTLY on the generated B2 container. This file pins:
//
//	4.3-CGPATH-001 (AC1, tp 6) — B2 single-resolve [IGenSlotA] == closed-specific impl (closed preferred via last-wins)
//	4.3-CGPATH-002 (AC2, tp 7) — B2 enumerable [IGenSlotA] count=2 containing open+closed (order-insensitive)
//	4.3-CGPATH-003 (AC3, tp 8) — B2 enumerable [IGenSlotB] count=3 ORDERED [closed#1, open, instance]
//	4.3-CGPATH-004 (AC5)       — regen byte-identical (consecutive generate idempotent, AD-14)
//
// godi has NO open/closed distinction at the generator level (AD-4/AD-10: no
// MakeGenericType); the B2 container materializes only CLOSED registrations, and
// slot order / priority are whatever the registration source encodes (single =
// last-wins, enumerable = registration order). The generator is UNCHANGED for
// 4.3 — the slot order is reproduced purely by the 4.3 registration block (Task 2)
// reusing the 4.2 multi-impl × closed-generic aggregation pipeline.
//
// This file began as an ATDD red-phase scaffold behind //go:build atdd_redphase;
// that constraint was removed (and the suite joined the default build) once the
// Task 1 Fakes and the Task 2 registration + regen landed.
package codegen_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
)

// Provider-stable dynamic type names for the B2 container (R4 observables).
const (
	dynCGSlotOpenA     = "*fakes.FakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IGenSlotA]"
	dynCGSlotClosedA   = "*fakes.ClosedFakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IGenSlotA]"
	dynCGSlotClosedB   = "*fakes.ClosedFakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IGenSlotB]"
	dynCGSlotOpenB     = "*fakes.FakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IGenSlotB]"
	dynCGSlotInstanceB = "*fakes.InstanceFakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IGenSlotB]"
)

func cgSlotServiceA() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IGenSlotA]]()
}
func cgSlotServiceB() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IGenSlotB]]()
}

// cgContainsType reports order-insensitive set membership by dynamic type name.
func cgContainsType(all []any, name string) bool {
	for _, v := range all {
		if reflect.TypeOf(v).String() == name {
			return true
		}
	}
	return false
}

// --- 4.3-CGPATH-001 (AC1, P0) — B2 closed preferred (single, last-wins) -------

// TestCodegen_PathGamma_OpenGenericSlot_ClosedPreferred: on the B2 container, the
// single resolve of the closed service type [IGenSlotA] returns the closed-
// specific impl (registered LAST → last-wins), NOT the open-produced impl. This
// is godi's reproduction of dotnet ClosedServicesPreferredOverOpenGenericServices
// via registration order (D-5).
func TestCodegen_PathGamma_OpenGenericSlot_ClosedPreferred(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	svc := root.GetService(cgSlotServiceA())
	testutil.NotNil(t, svc)
	if svc == nil {
		return
	}
	testutil.Equal(t, reflect.TypeOf(svc).String(), dynCGSlotClosedA)
}

// --- 4.3-CGPATH-002 (AC2, P0) — B2 enumerable contains both, count=2 ----------

// TestCodegen_PathGamma_OpenGenericSlot_EnumerableTwoSlots: on the B2 container,
// the enumerable of [IGenSlotA] has exactly 2 slots containing BOTH the open-
// produced and the closed-specific impl (order-insensitive, dotnet .Any —
// ResolvingEnumerableContainingOpenGenericServiceUsesCorrectSlot, GH#79938).
func TestCodegen_PathGamma_OpenGenericSlot_EnumerableTwoSlots(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(cgSlotServiceA(), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 2)
	testutil.True(t, cgContainsType(all, dynCGSlotOpenA),
		"B2 enumerable must contain the open-produced impl (order-insensitive)")
	testutil.True(t, cgContainsType(all, dynCGSlotClosedA),
		"B2 enumerable must contain the closed-specific impl (order-insensitive)")
}

// --- 4.3-CGPATH-003 (AC3, P0) — B2 mixed enumerable, count=3, ORDERED ----------

// TestCodegen_PathGamma_OpenGenericSlot_MixedEnumerableOrdered: on the B2
// container, the mixed enumerable of [IGenSlotB] has exactly 3 slots in
// registration order [closed#1, open, instance] (ResolvesMixedOpenClosedGenerics
// AsEnumerable, DISpec:908). Order-sensitive — the slot positions are exact.
func TestCodegen_PathGamma_OpenGenericSlot_MixedEnumerableOrdered(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(cgSlotServiceB(), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 3)
	if len(all) != 3 {
		return
	}
	testutil.Equal(t, reflect.TypeOf(all[0]).String(), dynCGSlotClosedB)   // slot 0: closed#1
	testutil.Equal(t, reflect.TypeOf(all[1]).String(), dynCGSlotOpenB)     // slot 1: open-produced
	testutil.Equal(t, reflect.TypeOf(all[2]).String(), dynCGSlotInstanceB) // slot 2: instance
}

// --- 4.3-CGPATH-004 (AC5, P1) — regen idempotent (deterministic output) -------

// TestCodegen_PathGamma_OpenGenericSlot_RegenIdempotent: generating from the same
// registry twice must produce byte-identical output (AD-14 deterministic emit).
// The slot-order block (multiple closed registrations under one generic service
// type, with sanitized type-arg segments) must not introduce nondeterminism
// (e.g. map iteration order) into the generated container. conformanceRegistry()
// lives in gen_test.go (same package) and is updated by Task 2 to include the 4.3
// entries; this guard pins idempotency regardless of that content.
func TestCodegen_PathGamma_OpenGenericSlot_RegenIdempotent(t *testing.T) {
	reg := conformanceRegistry()
	first, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	second, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	testutil.Equal(t, string(first), string(second))
}

// --- 4.3-CGPATH-005 (AC5, review patch) — B2 scope/root slot-order symmetry ----

// TestCodegen_PathGamma_OpenGenericSlot_ScopeSymmetry: the generator emits
// dedicated scope-side slot-order branches (scope GetService / scope
// GetKeyedServices nil / scope IsService). Resolving the slot-order types through
// a child scope on the B2 container must match the root observables: single
// [IGenSlotA] == closed, enumerable [IGenSlotA] count=2 (both types), single
// [IGenSlotB] == instance (last-wins), enumerable [IGenSlotB] count=3 ORDERED
// [closed#1, open, instance]. The root-only CGPATH-001..003 cannot catch a
// root/scope asymmetry in the generated container.
func TestCodegen_PathGamma_OpenGenericSlot_ScopeSymmetry(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	sc := root.CreateScope()
	t.Cleanup(func() { _ = sc.Close() })

	// touch-point 6/7 (IGenSlotA): single == closed, enumerable count=2 (both).
	testutil.Equal(t, reflect.TypeOf(sc.GetService(cgSlotServiceA())).String(), dynCGSlotClosedA)
	allA := sc.GetKeyedServices(cgSlotServiceA(), nil)
	testutil.Equal(t, len(allA), 2)
	testutil.True(t, cgContainsType(allA, dynCGSlotOpenA),
		"scope enumerable must contain the open-produced impl")
	testutil.True(t, cgContainsType(allA, dynCGSlotClosedA),
		"scope enumerable must contain the closed-specific impl")

	// touch-point 8 (IGenSlotB): single == instance (last-wins), enumerable
	// count=3 ORDERED [closed#1, open, instance] — identical to root.
	testutil.Equal(t, reflect.TypeOf(sc.GetService(cgSlotServiceB())).String(), dynCGSlotInstanceB)
	allB := sc.GetKeyedServices(cgSlotServiceB(), nil)
	testutil.Equal(t, len(allB), 3)
	if len(allB) != 3 {
		return
	}
	testutil.Equal(t, reflect.TypeOf(allB[0]).String(), dynCGSlotClosedB)
	testutil.Equal(t, reflect.TypeOf(allB[1]).String(), dynCGSlotOpenB)
	testutil.Equal(t, reflect.TypeOf(allB[2]).String(), dynCGSlotInstanceB)
}
