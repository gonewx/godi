// ATDD red-phase scaffold for Story 5.1 — plan package: transitive missing
// dependency detection with chain reporting and aggregated error output.
//
// Story 1.4 delivered flat (direct-only) missing dependency detection. This
// story extends Validate to walk the full transitive dependency closure via DFS,
// attributing errors to each affected ROOT service with the complete chain path.
//
// Red-phase focus: the current Validate detects "B depends on missing C" but
// does NOT generate an error for A when A→B→C (because A's direct dep B is
// registered). These tests assert the transitive attribution that is missing.
//
// Activation (TDD red → green): delete the //go:build line above and the blank
// line after it once plan.Validate performs transitive DFS walking (Story 5.1
// Task 2). The file then joins the default build and drives the slice to green.
//
// COMPATIBILITY: Story 1.4 tests (registry_test.go) use weak assertions
// (strings.Contains + len(errs) >=). The enhanced error format MUST still
// contain dependency name keywords to keep those tests green.
package plan_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// --- 5.1-PLAN-T2 (AC2, P0) — two-level transitive missing dep ----------------

// TestValidate_TransitiveMissingDep_ReportsChain: A→B→missing C, B is registered.
// Current flat Validate catches "B depends on missing C" but does NOT report an
// error attributed to A. This test asserts that A gets a transitive attribution
// error containing the full chain.
func TestValidate_TransitiveMissingDep_ReportsChain(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "C_Impl",
		GoType:   "*svc.C",
		Ctor:     "svc.NewC",
		Lifetime: plan.Singleton,
		Deps:     []string{"Missing_Dep"},
	})
	reg.Service(plan.Service{
		Name:     "B_Impl",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"C_Impl"},
	})
	reg.Service(plan.Service{
		Name:     "A_Impl",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B_Impl"},
	})

	errs := reg.Validate()

	// At minimum: C_Impl direct missing + B_Impl transitive + A_Impl transitive
	testutil.True(t, len(errs) >= 2,
		"transitive missing dep must produce errors for root services beyond the direct offender")

	// A_Impl must have a transitive attribution error containing the chain
	foundAChain := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "A_Impl") && strings.Contains(msg, "Missing_Dep") {
			foundAChain = true
			break
		}
	}
	testutil.True(t, foundAChain,
		"A_Impl must have an error attributing the transitive missing dep (chain: A_Impl → B_Impl → C_Impl → Missing_Dep)")

	// B_Impl must also have a transitive or direct attribution error
	foundBChain := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "B_Impl") && strings.Contains(msg, "Missing_Dep") {
			foundBChain = true
			break
		}
	}
	testutil.True(t, foundBChain,
		"B_Impl must have an error for its transitive dep chain to Missing_Dep")
}

// --- 5.1-PLAN-T3 (AC2, P0) — three-level transitive missing dep ---------------

// TestValidate_DeepTransitiveMissingDep: A→B→C→missing D (three-level transitive).
// All three registered services (A, B, C) must have attribution errors and A's
// error must contain the full deep chain.
func TestValidate_DeepTransitiveMissingDep(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "D_Impl",
		GoType:   "*svc.D",
		Ctor:     "svc.NewD",
		Lifetime: plan.Singleton,
		Deps:     []string{"Unregistered_Svc"},
	})
	reg.Service(plan.Service{
		Name:     "C_Impl",
		GoType:   "*svc.C",
		Ctor:     "svc.NewC",
		Lifetime: plan.Singleton,
		Deps:     []string{"D_Impl"},
	})
	reg.Service(plan.Service{
		Name:     "B_Impl",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"C_Impl"},
	})
	reg.Service(plan.Service{
		Name:     "A_Impl",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B_Impl"},
	})

	errs := reg.Validate()

	// A, B, C all affected (D_Impl is the direct offender, also reported)
	testutil.True(t, len(errs) >= 3,
		"deep transitive missing dep must produce errors for A, B, and C (plus D direct)")

	// A_Impl must mention both A_Impl and Unregistered_Svc in a single error
	foundADeep := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "A_Impl") && strings.Contains(msg, "Unregistered_Svc") {
			foundADeep = true
			break
		}
	}
	testutil.True(t, foundADeep,
		"A_Impl error must contain the deep transitive chain to Unregistered_Svc")
}

// --- 5.1-PLAN-T6 (AC3, P0) — multiple missing deps across independent services -

// TestValidate_MultipleMissingDeps_AcrossServices: A→missing X, B→missing Y,
// two independent roots with different missing deps. Must report both.
func TestValidate_MultipleMissingDeps_AcrossServices(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "Svc_A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"Missing_X"},
	})
	reg.Service(plan.Service{
		Name:     "Svc_B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Scoped,
		Deps:     []string{"Missing_Y"},
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) >= 2,
		"two independent missing deps must produce at least two errors")

	foundX := false
	foundY := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "Missing_X") {
			foundX = true
		}
		if strings.Contains(msg, "Missing_Y") {
			foundY = true
		}
	}
	testutil.True(t, foundX, "Missing_X must be reported")
	testutil.True(t, foundY, "Missing_Y must be reported")
}

// --- 5.1-PLAN-T4 (AC2/AC3, P0) — shared missing dep affects multiple roots ----

// TestValidate_TransitiveMissingDep_SharedDep: A→B→missing C, D→B→missing C.
// Same missing dep reachable from two independent roots. Both A and D must get
// independent attribution errors.
func TestValidate_TransitiveMissingDep_SharedDep(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "B_Impl",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"Missing_C"},
	})
	reg.Service(plan.Service{
		Name:     "A_Impl",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B_Impl"},
	})
	reg.Service(plan.Service{
		Name:     "D_Impl",
		GoType:   "*svc.D",
		Ctor:     "svc.NewD",
		Lifetime: plan.Scoped,
		Deps:     []string{"B_Impl"},
	})

	errs := reg.Validate()

	// B_Impl direct + A_Impl transitive + D_Impl transitive = at least 3
	testutil.True(t, len(errs) >= 3,
		"shared missing dep reachable from two roots must produce errors for both roots + direct")

	foundA := false
	foundD := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "A_Impl") && strings.Contains(msg, "Missing_C") {
			foundA = true
		}
		if strings.Contains(msg, "D_Impl") && strings.Contains(msg, "Missing_C") {
			foundD = true
		}
	}
	testutil.True(t, foundA, "A_Impl must have independent attribution for Missing_C")
	testutil.True(t, foundD, "D_Impl must have independent attribution for Missing_C")
}

// --- 5.1-PLAN-T7 (AC3, P1) — mixed valid and invalid chains -------------------

// TestValidate_MixedValidAndInvalid: legal chain (X→Y, both registered) +
// illegal chain (A→missing Z). Only the illegal chain reports errors; the legal
// chain must not generate false positives.
func TestValidate_MixedValidAndInvalid(t *testing.T) {
	reg := plan.NewRegistry()
	// Legal chain
	reg.Service(plan.Service{
		Name:     "Y_Impl",
		GoType:   "*svc.Y",
		Ctor:     "svc.NewY",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "X_Impl",
		GoType:   "*svc.X",
		Ctor:     "svc.NewX",
		Lifetime: plan.Singleton,
		Deps:     []string{"Y_Impl"},
	})
	// Illegal chain
	reg.Service(plan.Service{
		Name:     "A_Impl",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Transient,
		Deps:     []string{"Missing_Z"},
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) >= 1,
		"at least one error for the illegal chain")

	// No errors should mention X_Impl or Y_Impl
	for _, e := range errs {
		msg := e.Error()
		testutil.True(t, !strings.Contains(msg, "X_Impl"),
			"legal service X_Impl must not appear in errors")
		testutil.True(t, !strings.Contains(msg, "Y_Impl"),
			"legal service Y_Impl must not appear in errors")
	}

	// Missing_Z must be reported
	foundZ := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "Missing_Z") {
			foundZ = true
			break
		}
	}
	testutil.True(t, foundZ, "Missing_Z must be reported")
}

// --- 5.1-PLAN-T5 (AC2, P1) — slice dep transitive missing ---------------------

// TestValidate_SliceDep_TransitiveMissing: a service depends on []IFoo (slice
// dep); no service with GoType IFoo is registered. The slice dep itself is
// detected as missing by current Validate (flat check). This test verifies
// that if a service has a slice dep whose elements are registered but THOSE
// elements have missing transitive deps, the root is attributed.
//
// Scenario: A depends on []IFoo. IFoo_Impl is registered with GoType "svc.IFoo"
// but IFoo_Impl→missing B. A must be attributed the transitive failure.
func TestValidate_SliceDep_TransitiveMissing(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "IFoo_Impl",
		GoType:   "svc.IFoo",
		Ctor:     "svc.NewFooImpl",
		Lifetime: plan.Singleton,
		Deps:     []string{"Missing_B"},
	})
	reg.Service(plan.Service{
		Name:     "A_Impl",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"[]svc.IFoo"},
	})

	errs := reg.Validate()

	// IFoo_Impl has direct missing dep Missing_B.
	// A_Impl has slice dep []svc.IFoo → resolved to IFoo_Impl → Missing_B transitive.
	testutil.True(t, len(errs) >= 2,
		"slice dep transitive missing must produce errors for both the element and the consumer")

	// A_Impl must be attributed (transitive through slice dep resolution)
	foundA := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "A_Impl") && strings.Contains(msg, "Missing_B") {
			foundA = true
			break
		}
	}
	testutil.True(t, foundA,
		"A_Impl must be attributed the transitive missing dep through its slice dep []svc.IFoo → IFoo_Impl → Missing_B")
}
