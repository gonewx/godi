// ATDD red-phase scaffold for Story 2.2 (AC1) — plan package: Transients()
// lifetime-aware query method.
//
//	(*Registry).Transients() []Service   // transients in registration order
//
// Transients have no dependency ordering need (each New*() is independent) but
// a stable enumeration order for emit (D-4). The scoped dependency-order query
// that used to live here (ScopedTopo) was removed when scoped services became
// lazily built accessors: construction order now follows first resolution.
package plan_test

import (
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

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
