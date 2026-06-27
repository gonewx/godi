// Story 2.1 (AC1/AC3) — plan.SingletonsTopo dependency-ordered toposort +
// Service.CtorErr emit-input field.
//
// emitNewRoot must wire singletons in dependency order (a service after every
// singleton it depends on), so plan exposes SingletonsTopo() for genlib to
// consume directly (Task 1). Service.CtorErr records whether the constructor
// has a trailing error return — an emit input derived from go/types by the
// static analyzer (AD-7), NOT a hand-set canonical fact:
//
//	func (r *plan.Registry) SingletonsTopo() []plan.Service  // dependency order
//	plan.Service.CtorErr bool                                // trailing-error flag
package plan_test

import (
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// TestTopo_LinearChain_DependencyBeforeDependent covers 2.1-TOPO-001 (AC1).
//
// For C→B→A (C depends on B, B depends on A), SingletonsTopo must place each
// dependency before its dependent: A before B before C. Registration order is
// the reverse (C, B, A) to prove the sort is by dependency, not insertion.
func TestTopo_LinearChain_DependencyBeforeDependent(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{Name: "C", Lifetime: plan.Singleton, Deps: []string{"B"}})
	reg.Service(plan.Service{Name: "B", Lifetime: plan.Singleton, Deps: []string{"A"}})
	reg.Service(plan.Service{Name: "A", Lifetime: plan.Singleton})

	order := topoNames(reg.SingletonsTopo())

	testutil.Equal(t, len(order), 3)
	testutil.True(t, indexOf(order, "A") < indexOf(order, "B"), "A must be wired before B")
	testutil.True(t, indexOf(order, "B") < indexOf(order, "C"), "B must be wired before C")
}

// TestTopo_Diamond_DedupesSharedDependency covers 2.1-TOPO-002 (AC1).
//
// Diamond: D→{B,C}, B→A, C→A. The shared dependency A must appear exactly once
// (visited dedup) and before everything; D last. This is the case that breaks a
// naive recursive emit without a visited set.
func TestTopo_Diamond_DedupesSharedDependency(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{Name: "D", Lifetime: plan.Singleton, Deps: []string{"B", "C"}})
	reg.Service(plan.Service{Name: "B", Lifetime: plan.Singleton, Deps: []string{"A"}})
	reg.Service(plan.Service{Name: "C", Lifetime: plan.Singleton, Deps: []string{"A"}})
	reg.Service(plan.Service{Name: "A", Lifetime: plan.Singleton})

	order := topoNames(reg.SingletonsTopo())

	testutil.Equal(t, len(order), 4) // A appears once, not twice (dedup)
	testutil.True(t, indexOf(order, "A") < indexOf(order, "B"), "A before B")
	testutil.True(t, indexOf(order, "A") < indexOf(order, "C"), "A before C")
	testutil.True(t, indexOf(order, "B") < indexOf(order, "D"), "B before D")
	testutil.True(t, indexOf(order, "C") < indexOf(order, "D"), "C before D")
}

// TestTopo_NoDeps_PreservesRegistrationOrder covers 2.1-TOPO-003 (AC3).
//
// With no dependencies, the topo order must equal registration order — stable,
// deterministic output (a precondition for byte-identical regeneration, AD-14).
func TestTopo_NoDeps_PreservesRegistrationOrder(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{Name: "First", Lifetime: plan.Singleton})
	reg.Service(plan.Service{Name: "Second", Lifetime: plan.Singleton})
	reg.Service(plan.Service{Name: "Third", Lifetime: plan.Singleton})

	order := topoNames(reg.SingletonsTopo())

	testutil.DeepEqual(t, order, []string{"First", "Second", "Third"})
}

// TestTopo_ServiceCtorErrField_IsReadWritable covers 2.1-TOPO-004 (AC1).
//
// Service.CtorErr is the emit input that drives error propagation in NewRoot
// (Task 2). This pins the field's existence; the static analyzer (Task 3)
// derives its value from the constructor signature via go/types (AD-7).
func TestTopo_ServiceCtorErrField_IsReadWritable(t *testing.T) {
	s := plan.Service{Name: "X", Lifetime: plan.Singleton, CtorErr: true}
	testutil.True(t, s.CtorErr, "Service.CtorErr emit-input field must exist and be settable")
}

// --- helpers ----------------------------------------------------------------

func topoNames(svcs []plan.Service) []string {
	out := make([]string, len(svcs))
	for i, s := range svcs {
		out[i] = s.Name
	}
	return out
}

func indexOf(xs []string, x string) int {
	for i, v := range xs {
		if v == x {
			return i
		}
	}
	return -1
}
