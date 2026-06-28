// ATDD red-phase scaffold for Story 2.4 (AC1, Task 1) — plan registry:
// ServicesByGoType index for same-GoType multi-registration.
//
// This file tests the NEW ServicesByGoType method on plan.Registry, which does
// NOT exist yet. It will NOT compile until Task 1 adds the method — the compile
// error IS the red-light anchor.
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-4-ienumerable-resolution.md.
package plan_test

import (
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// --- 2.4-PLAN-001 (AC1, P0) -------------------------------------------------

// TestRegistry_ServicesByGoType_MultiRegistration: when multiple services share
// the same GoType (different Names), ServicesByGoType returns them all in
// registration order. This is the plan-layer underpinning of IEnumerable
// multi-implementation resolution (CAP-4).
func TestRegistry_ServicesByGoType_MultiRegistration(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "FakeOneMultipleService",
		GoType:   "fakes.IFakeMultipleService",
		Ctor:     "fakes.NewFakeOneMultipleService",
		Lifetime: plan.Transient,
	})
	reg.Service(plan.Service{
		Name:     "FakeTwoMultipleService",
		GoType:   "fakes.IFakeMultipleService",
		Ctor:     "fakes.NewFakeTwoMultipleService",
		Lifetime: plan.Transient,
	})

	got := reg.ServicesByGoType("fakes.IFakeMultipleService")
	testutil.Equal(t, len(got), 2)
	testutil.Equal(t, got[0].Name, "FakeOneMultipleService")
	testutil.Equal(t, got[1].Name, "FakeTwoMultipleService")
}

// --- 2.4-PLAN-002 (AC1, P0) -------------------------------------------------

// TestRegistry_ServicesByGoType_SingleRegistration: a GoType with only one
// registration returns a single-element slice — no special-casing needed for
// the "single service can be IEnumerable resolved" case (AC3).
func TestRegistry_ServicesByGoType_SingleRegistration(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "FakeService",
		GoType:   "fakes.IFakeService",
		Ctor:     "fakes.NewFakeService",
		Lifetime: plan.Singleton,
	})

	got := reg.ServicesByGoType("fakes.IFakeService")
	testutil.Equal(t, len(got), 1)
	testutil.Equal(t, got[0].Name, "FakeService")
}

// --- 2.4-PLAN-003 (AC4, P1) -------------------------------------------------

// TestRegistry_ServicesByGoType_Unregistered: a GoType with no registrations
// returns an empty (not nil) slice — consistent with the runtime behavior of
// returning an empty collection for unregistered types (AC4).
func TestRegistry_ServicesByGoType_Unregistered(t *testing.T) {
	reg := plan.NewRegistry()
	got := reg.ServicesByGoType("fakes.INotRegistered")
	testutil.NotNil(t, got)
	testutil.Equal(t, len(got), 0)
}

// --- 2.4-PLAN-004 (AC1, P1) -------------------------------------------------

// TestRegistry_ServicesByGoType_DoesNotMixGoTypes: services with DIFFERENT
// GoTypes must NOT appear in each other's ServicesByGoType results — the index
// is keyed by GoType string.
func TestRegistry_ServicesByGoType_DoesNotMixGoTypes(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "FakeService",
		GoType:   "fakes.IFakeService",
		Ctor:     "fakes.NewFakeService",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "FakeOneMultipleService",
		GoType:   "fakes.IFakeMultipleService",
		Ctor:     "fakes.NewFakeOneMultipleService",
		Lifetime: plan.Transient,
	})

	got := reg.ServicesByGoType("fakes.IFakeMultipleService")
	testutil.Equal(t, len(got), 1)
	testutil.Equal(t, got[0].Name, "FakeOneMultipleService")
}

// --- 2.4-PLAN-005 (AC1, P1) -------------------------------------------------

// TestRegistry_Validate_AllowsSameGoTypeMultiRegistration: Validate must NOT
// report an error when the same GoType appears in multiple services with
// different Names. byName already keys by Name (not GoType), so no duplicate
// collision — but we assert this explicitly to guard against future regressions.
func TestRegistry_Validate_AllowsSameGoTypeMultiRegistration(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "FakeOneMultipleService",
		GoType:   "fakes.IFakeMultipleService",
		Ctor:     "fakes.NewFakeOneMultipleService",
		Lifetime: plan.Transient,
	})
	reg.Service(plan.Service{
		Name:     "FakeTwoMultipleService",
		GoType:   "fakes.IFakeMultipleService",
		Ctor:     "fakes.NewFakeTwoMultipleService",
		Lifetime: plan.Transient,
	})

	errs := reg.Validate()
	testutil.Equal(t, len(errs), 0)
}
