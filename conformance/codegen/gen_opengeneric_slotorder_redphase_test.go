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
// The "red" comes from:
//   - fakes.IGenSlotA/IGenSlotB + ClosedFakeOpenGenericService[T] +
//     InstanceFakeOpenGenericService[T] do not exist yet (Task 1) → compile failure.
//   - container_gen.go has no slot-order cases until registration.go gains the 4.3
//     block and the container is regenerated (Task 2) → CGPATH-001/002/003 resolve
//     nil. (CGPATH-004 pins idempotency regardless of registry content.)
//
// Activation: delete the //go:build line and the blank line below it to join the
// default build (after Task 1/2 land the fakes + registration + regen).
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
