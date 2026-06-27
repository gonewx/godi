//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.2 (AC1) — plan package: ScopedTopo()
// and Transients() lifetime-aware query methods.
//
// Story 2.1 delivered SingletonsTopo() for singleton dependency order. This
// story activates the Scoped and Transient Lifetime values (already defined
// but dormant) and adds two new query methods:
//
//	(*Registry).ScopedTopo() []Service   // scoped in dependency order
//	(*Registry).Transients() []Service   // transients in registration order
//
// ScopedTopo follows the same depth-first post-order as SingletonsTopo, but
// scoped services can depend on singletons (access r.f_Singleton) or other
// scoped (same scope, topo order). Transients have no dependency ordering need
// (each New*() is independent) but a stable enumeration order for emit (D-4).
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-2-scoped-transient-lifetimes.md.
package plan_test

import (
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// --- 2.2-PLAN-001 (AC1, P0) --------------------------------------------------

// TestScopedTopo_ReturnsScopedInDependencyOrder: ScopedTopo must return only
// scoped services, in dependency order — a scoped that depends on another
// scoped appears after it.
func TestScopedTopo_ReturnsScopedInDependencyOrder(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{Name: "SB", Lifetime: plan.Scoped, Deps: []string{"SA"}})
	reg.Service(plan.Service{Name: "SA", Lifetime: plan.Scoped})

	order := topoNames(reg.ScopedTopo())

	testutil.Equal(t, len(order), 2)
	testutil.True(t, indexOf(order, "SA") < indexOf(order, "SB"),
		"SA must be wired before SB (dependency order)")
}

// --- 2.2-PLAN-002 (AC1, P0) --------------------------------------------------

// TestTransients_ReturnsTransientsInRegistrationOrder: Transients must return
// only transient services, in registration order (stable emit order, D-4).
func TestTransients_ReturnsTransientsInRegistrationOrder(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{Name: "T1", Lifetime: plan.Transient})
	reg.Service(plan.Service{Name: "S1", Lifetime: plan.Singleton})
	reg.Service(plan.Service{Name: "T2", Lifetime: plan.Transient})

	ts := transientNames(reg.Transients())

	testutil.DeepEqual(t, ts, []string{"T1", "T2"})
}

// --- 2.2-PLAN-003 (AC1, P1) --------------------------------------------------

// TestScopedTopo_ScopedCanDependOnSingleton: a scoped service depending on a
// singleton is legal — ScopedTopo must include the scoped but NOT the
// singleton (the singleton is wired separately in NewRoot).
func TestScopedTopo_ScopedCanDependOnSingleton(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{Name: "Logger", Lifetime: plan.Singleton})
	reg.Service(plan.Service{Name: "Req", Lifetime: plan.Scoped, Deps: []string{"Logger"}})

	order := topoNames(reg.ScopedTopo())

	testutil.Equal(t, len(order), 1)
	testutil.Equal(t, order[0], "Req")
}

// --- 2.2-PLAN-004 (AC1, P0) --------------------------------------------------

// TestMixedLifetimes_SingletonsTopoUnchanged: adding scoped and transient
// services must NOT affect SingletonsTopo output — regression guard for the
// existing query used by emitNewRoot.
func TestMixedLifetimes_SingletonsTopoUnchanged(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{Name: "A", Lifetime: plan.Singleton})
	reg.Service(plan.Service{Name: "S1", Lifetime: plan.Scoped})
	reg.Service(plan.Service{Name: "B", Lifetime: plan.Singleton, Deps: []string{"A"}})
	reg.Service(plan.Service{Name: "T1", Lifetime: plan.Transient})

	order := topoNames(reg.SingletonsTopo())

	testutil.DeepEqual(t, order, []string{"A", "B"})
}

// --- 2.2-PLAN-005 (AC1, P1) --------------------------------------------------

// TestTransients_NoDepsPreservesRegistrationOrder: transients with no
// dependencies must preserve registration order exactly (byte-identical
// regeneration precondition, AD-14).
func TestTransients_NoDepsPreservesRegistrationOrder(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{Name: "C", Lifetime: plan.Transient})
	reg.Service(plan.Service{Name: "A", Lifetime: plan.Transient})
	reg.Service(plan.Service{Name: "B", Lifetime: plan.Transient})

	ts := transientNames(reg.Transients())

	testutil.DeepEqual(t, ts, []string{"C", "A", "B"})
}

// --- helpers ----------------------------------------------------------------

func transientNames(svcs []plan.Service) []string {
	out := make([]string, len(svcs))
	for i, s := range svcs {
		out[i] = s.Name
	}
	return out
}
