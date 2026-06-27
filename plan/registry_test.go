// ATDD scaffold for Story 1.4 — plan package: Registry model +
// Validate minimum slice (missing dependency detection).
//
// Activation (TDD red -> green): delete the //go:build line above and the blank
// line after it once Registry, Service, Lifetime, and Validate exist in the plan
// package (Story 1.4 Tasks 1-2). The file then joins the default build and these
// assertions drive the slice to green.
// See _bmad-output/test-artifacts/atdd-checklist-1-4-codegen-validate-gate.md.
//
// API CONTRACT THIS SCAFFOLD ASSUMES (Story 1.4 Tasks 1-2):
//
//	plan.NewRegistry() *Registry
//	(*Registry).Service(s Service) *Registry   // chainable
//	(*Registry).Services() []Service
//	(*Registry).Lookup(name string) (Service, bool)
//	(*Registry).Validate() []error
//
//	type Service struct { Name, GoType, Ctor string; Lifetime Lifetime; Deps []string; Disposable bool }
//	type Lifetime int   const (Singleton Lifetime = iota; Scoped; Transient)
//
// If the dev pins a different API shape, update the calls below — the BEHAVIORS
// asserted (registration, lookup, validate errors, aggregation) are the contract
// and must not change.
package plan_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// --- 1.4-PLAN-001 (AC2/AC4, P0) -----------------------------------------------

// TestRegistry_ServiceRegistrationAndLookup: basic Registry operations — register
// a Service via the chainable .Service() method, look it up by name, verify the
// miss path, and list all registered services.
func TestRegistry_ServiceRegistrationAndLookup(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "Logger",
		GoType:   "*logger.Logger",
		Ctor:     "logger.NewLogger",
		Lifetime: plan.Singleton,
	})

	svc, ok := reg.Lookup("Logger")
	testutil.True(t, ok, "registered service must be found")
	testutil.Equal(t, svc.Name, "Logger")
	testutil.Equal(t, svc.GoType, "*logger.Logger")
	testutil.Equal(t, svc.Ctor, "logger.NewLogger")
	testutil.Equal(t, svc.Lifetime, plan.Singleton)

	_, ok = reg.Lookup("NotRegistered")
	testutil.True(t, !ok, "unregistered service must miss")

	all := reg.Services()
	testutil.Equal(t, len(all), 1)
	testutil.Equal(t, all[0].Name, "Logger")
}

// --- 1.4-PLAN-002 (AC2, P0) ---------------------------------------------------

// TestValidate_LegalGraph_NoDeps_ReturnsNil: a graph with a single no-dep
// singleton is legal; Validate returns nil (empty []error).
func TestValidate_LegalGraph_NoDeps_ReturnsNil(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "Logger",
		GoType:   "*logger.Logger",
		Ctor:     "logger.NewLogger",
		Lifetime: plan.Singleton,
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) == 0, "legal graph must produce zero validation errors")
}

// --- 1.4-PLAN-003 (AC2, P0) ---------------------------------------------------

// TestValidate_MissingDependency_ReturnsError: a service depending on an
// unregistered type triggers a validation error. The error message must describe
// the missing dependency (AD-3: bad graph rejected; NFR2: hard gate).
func TestValidate_MissingDependency_ReturnsError(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "Handler",
		GoType:   "*http.Handler",
		Ctor:     "NewHandler",
		Lifetime: plan.Singleton,
		Deps:     []string{"Logger"},
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) > 0, "missing dependency must produce validation errors")

	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "Logger") {
			found = true
			break
		}
	}
	testutil.True(t, found, "error must name the missing dependency")
}

// --- 1.4-PLAN-004 (AC2, P0) ---------------------------------------------------

// TestValidate_MultipleMissingDeps_AggregatesErrors: multiple missing
// dependencies produce MULTIPLE errors in one pass — Consistency Conventions
// require aggregated []error, not fail-on-first.
func TestValidate_MultipleMissingDeps_AggregatesErrors(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "Handler",
		GoType:   "*http.Handler",
		Ctor:     "NewHandler",
		Lifetime: plan.Singleton,
		Deps:     []string{"Logger", "Config"},
	})

	errs := reg.Validate()
	testutil.True(t, len(errs) >= 2,
		"two missing deps must produce at least two errors (aggregate, not fail-on-first)")
}

// --- 1.4-PLAN-005 (AC2, P1) ---------------------------------------------------

// TestValidate_EmptyGraph_ReturnsNil: an empty Registry (zero services) is legal;
// Validate returns nil.
func TestValidate_EmptyGraph_ReturnsNil(t *testing.T) {
	reg := plan.NewRegistry()
	errs := reg.Validate()
	testutil.True(t, len(errs) == 0, "empty graph must be legal")
}

// --- 1.4-PLAN-006 (AC2/AC4, P1) -----------------------------------------------

// TestRegistry_ChainableRegistration: the Service() method returns *Registry for
// fluent chaining of multiple registrations. A graph with satisfied dependencies
// (all deps registered) validates clean.
func TestRegistry_ChainableRegistration(t *testing.T) {
	reg := plan.NewRegistry()
	ret := reg.
		Service(plan.Service{
			Name:     "Logger",
			GoType:   "*logger.Logger",
			Ctor:     "logger.NewLogger",
			Lifetime: plan.Singleton,
		}).
		Service(plan.Service{
			Name:     "Handler",
			GoType:   "*http.Handler",
			Ctor:     "NewHandler",
			Lifetime: plan.Singleton,
			Deps:     []string{"Logger"},
		})

	testutil.Equal(t, ret, reg) // chainable returns same registry
	testutil.Equal(t, len(reg.Services()), 2)

	errs := reg.Validate()
	testutil.True(t, len(errs) == 0, "graph with satisfied deps must validate clean")
}
