// ATDD tests for Story 5.3 — plan package: circular dependency
// detection via three-color DFS (walkCycle).
//
// Story 5-1 delivered transitive missing dependency detection (walkMissing DFS).
// Story 5-2 delivered captive dependency detection (walkCaptive DFS).
// This story adds the third detection dimension: circular dependency detection
// via three-color marking DFS (white/gray/black). A back-edge (gray→gray)
// indicates a cycle.
//
// Red-phase focus: current Validate has no walkCycle — cycles in registered
// services pass validation silently. All assertions checking for
// "circular dependency" in error messages will fail.
//
// Activation: delete the //go:build line and the blank line after it once
// plan.Validate implements walkCycle (Story 5-3 Task 2).
//
// AC mapping:
//
//	AC1 — Validate detects cycles, reports path, rejects bad graph
//	AC3 — CircularDependencyTests (10 methods) coverage via plan-layer tests
package plan_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// --- 5.3-PLAN-T1 (AC1/AC3#1, P0) — self-dependency: A→A ---------------------

// TestValidate_Cycle_SelfDependency: Service A depends on itself. Validate must
// detect this degenerate cycle and report it.
// Maps to dotnet CircularDependencyTests: SelfCircularDependency (#1),
// SelfCircularDependencyGenericDirect (#3), SelfCircularDependencyWithInterface (#6).
func TestValidate_Cycle_SelfDependency(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"A"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 1,
		"self-dependency must produce at least one error")

	foundCycle := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "circular dependency") &&
			strings.Contains(msg, "A") {
			foundCycle = true
			break
		}
	}
	testutil.True(t, foundCycle,
		"error must contain 'circular dependency' and the self-referencing service name")
}

// --- 5.3-PLAN-T2 (AC1/AC3#7, P0) — direct cycle: A→B→A ---------------------

// TestValidate_Cycle_DirectAB: A depends on B, B depends on A. Classic direct
// two-node cycle.
// Maps to dotnet CircularDependencyTests: DirectCircularDependency (#7).
func TestValidate_Cycle_DirectAB(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B"},
	})
	reg.Service(plan.Service{
		Name:     "B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"A"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 1,
		"direct A↔B cycle must produce at least one error")

	foundCycle := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "circular dependency") &&
			strings.Contains(msg, "A") &&
			strings.Contains(msg, "B") {
			foundCycle = true
			break
		}
	}
	testutil.True(t, foundCycle,
		"error must contain 'circular dependency' with both A and B in the cycle path")
}

// --- 5.3-PLAN-T3 (AC1/AC3#8, P0) — indirect cycle: A→B→C→A -----------------

// TestValidate_Cycle_IndirectABC: three-node indirect cycle A→B→C→A.
// Maps to dotnet CircularDependencyTests: IndirectCircularDependency (#8).
func TestValidate_Cycle_IndirectABC(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B"},
	})
	reg.Service(plan.Service{
		Name:     "B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"C"},
	})
	reg.Service(plan.Service{
		Name:     "C",
		GoType:   "*svc.C",
		Ctor:     "svc.NewC",
		Lifetime: plan.Singleton,
		Deps:     []string{"A"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 1,
		"indirect A→B→C→A cycle must produce at least one error")

	foundCycle := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "circular dependency") &&
			strings.Contains(msg, "A") &&
			strings.Contains(msg, "B") &&
			strings.Contains(msg, "C") {
			foundCycle = true
			break
		}
	}
	testutil.True(t, foundCycle,
		"error must contain the full three-node cycle path")
}

// --- 5.3-PLAN-T4 (AC1/AC3#10, P0) — dependency on cyclic subtree ------------

// TestValidate_Cycle_DepOnCyclicSubtree: D depends on A, and A↔B form a cycle.
// D itself is not part of the cycle, but its dependency subtree contains one.
// The cycle A↔B must be detected.
// Maps to dotnet CircularDependencyTests: DependencyOnCircularDependency (#10).
func TestValidate_Cycle_DepOnCyclicSubtree(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "D",
		GoType:   "*svc.D",
		Ctor:     "svc.NewD",
		Lifetime: plan.Singleton,
		Deps:     []string{"A"},
	})
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B"},
	})
	reg.Service(plan.Service{
		Name:     "B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"A"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 1,
		"cycle in dependency subtree must be detected")

	foundCycle := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "circular dependency") &&
			strings.Contains(msg, "A") &&
			strings.Contains(msg, "B") {
			foundCycle = true
			break
		}
	}
	testutil.True(t, foundCycle,
		"A↔B cycle must be detected even when reached from non-cyclic root D")
}

// --- 5.3-PLAN-T5 (AC1, P1) — cycle via slice dep ----------------------------

// TestValidate_Cycle_ViaSliceDep: A depends on []IFoo, IFoo impl C depends on
// A. The cycle A→(via []IFoo)→C→A must be detected through slice dep expansion.
func TestValidate_Cycle_ViaSliceDep(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "C",
		GoType:   "svc.IFoo",
		Ctor:     "svc.NewC",
		Lifetime: plan.Singleton,
		Deps:     []string{"A"},
	})
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"[]svc.IFoo"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 1,
		"cycle through slice dep must be detected")

	foundCycle := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "circular dependency") {
			foundCycle = true
			break
		}
	}
	testutil.True(t, foundCycle,
		"error must indicate circular dependency through slice dep resolution")
}

// --- 5.3-PLAN-T6 (AC1/AC3#9, P0) — no cycle: diamond graph -----------------

// TestValidate_NoCycle_Diamond: A depends on B and C, B depends on C. Classic
// diamond — no cycle. Validate must pass with zero circular-dependency errors.
// Maps to dotnet CircularDependencyTests: NoCircularDependencySameTypeMultipleTimes (#9).
func TestValidate_NoCycle_Diamond(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "C",
		GoType:   "*svc.C",
		Ctor:     "svc.NewC",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"C"},
	})
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B", "C"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) == 0,
		"diamond graph (no cycle) must produce zero errors")
}

// --- 5.3-PLAN-T7 (AC1/AC3#5, P0) — no cycle: linear chain -------------------

// TestValidate_NoCycle_Linear: A→B→C→D, linear chain with no cycles.
// Maps to dotnet CircularDependencyTests: NoCircularDependencyGeneric (#5) —
// where a pre-built instance breaks what would otherwise be a cycle.
func TestValidate_NoCycle_Linear(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "D",
		GoType:   "*svc.D",
		Ctor:     "svc.NewD",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "C",
		GoType:   "*svc.C",
		Ctor:     "svc.NewC",
		Lifetime: plan.Singleton,
		Deps:     []string{"D"},
	})
	reg.Service(plan.Service{
		Name:     "B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"C"},
	})
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) == 0,
		"linear chain (no cycle) must produce zero errors")
}

// --- 5.3-PLAN-T8 (AC1, P0) — multiple independent cycles --------------------

// TestValidate_Cycle_MultipleCycles: two independent cycles (A↔B, C↔D). Both
// must be independently reported in the aggregated []error.
func TestValidate_Cycle_MultipleCycles(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B"},
	})
	reg.Service(plan.Service{
		Name:     "B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"A"},
	})
	reg.Service(plan.Service{
		Name:     "C",
		GoType:   "*svc.C",
		Ctor:     "svc.NewC",
		Lifetime: plan.Scoped,
		Deps:     []string{"D"},
	})
	reg.Service(plan.Service{
		Name:     "D",
		GoType:   "*svc.D",
		Ctor:     "svc.NewD",
		Lifetime: plan.Scoped,
		Deps:     []string{"C"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 2,
		"two independent cycles must produce at least two errors")

	foundAB := false
	foundCD := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "circular dependency") {
			if strings.Contains(msg, "A") && strings.Contains(msg, "B") {
				foundAB = true
			}
			if strings.Contains(msg, "C") && strings.Contains(msg, "D") {
				foundCD = true
			}
		}
	}
	testutil.True(t, foundAB,
		"A↔B cycle must be reported")
	testutil.True(t, foundCD,
		"C↔D cycle must be reported")
}

// --- 5.3-PLAN-T9 (AC1, P0) — mixed: cycle + missing + captive coexist -------

// TestValidate_Cycle_MixedWithMissingAndCaptive: three independent violations
// coexist — a cycle (E↔F), a missing dep (G→Missing_X), and a captive dep
// (Singleton_H→Scoped_I). All three error categories must be reported.
func TestValidate_Cycle_MixedWithMissingAndCaptive(t *testing.T) {
	reg := plan.NewRegistry()
	// Cycle: E↔F
	reg.Service(plan.Service{
		Name:     "E",
		GoType:   "*svc.E",
		Ctor:     "svc.NewE",
		Lifetime: plan.Singleton,
		Deps:     []string{"F"},
	})
	reg.Service(plan.Service{
		Name:     "F",
		GoType:   "*svc.F",
		Ctor:     "svc.NewF",
		Lifetime: plan.Singleton,
		Deps:     []string{"E"},
	})
	// Missing dep: G→Missing_X
	reg.Service(plan.Service{
		Name:     "G",
		GoType:   "*svc.G",
		Ctor:     "svc.NewG",
		Lifetime: plan.Transient,
		Deps:     []string{"Missing_X"},
	})
	// Captive dep: Singleton_H→Scoped_I
	reg.Service(plan.Service{
		Name:     "Scoped_I",
		GoType:   "*svc.I",
		Ctor:     "svc.NewI",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "Singleton_H",
		GoType:   "*svc.H",
		Ctor:     "svc.NewH",
		Lifetime: plan.Singleton,
		Deps:     []string{"Scoped_I"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 3,
		"cycle + missing + captive must produce at least three errors")

	foundCycle := false
	foundMissing := false
	foundCaptive := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "circular dependency") {
			foundCycle = true
		}
		if strings.Contains(msg, "missing dependency") && strings.Contains(msg, "Missing_X") {
			foundMissing = true
		}
		if strings.Contains(msg, "captive dependency") &&
			strings.Contains(msg, "Singleton_H") {
			foundCaptive = true
		}
	}
	testutil.True(t, foundCycle,
		"circular dependency error must be reported")
	testutil.True(t, foundMissing,
		"missing dependency error must be reported")
	testutil.True(t, foundCaptive,
		"captive dependency error must be reported")
}
