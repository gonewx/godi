// ATDD tests for Story 5.2 — plan package: captive dependency detection
// (singleton captures scoped, direct and transitive).
package plan_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// --- 5.2-PLAN-T1 (AC1, P0) — direct captive: Singleton→Scoped ---------------

// TestValidate_CaptiveDep_SingletonDependsOnScoped: a Singleton directly
// depends on a Scoped service. Validate must reject with a captive error.
func TestValidate_CaptiveDep_SingletonDependsOnScoped(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "ScopedSvc",
		GoType:   "*svc.ScopedSvc",
		Ctor:     "svc.NewScopedSvc",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "SingletonSvc",
		GoType:   "*svc.SingletonSvc",
		Ctor:     "svc.NewSingletonSvc",
		Lifetime: plan.Singleton,
		Deps:     []string{"ScopedSvc"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 1,
		"direct captive dependency must produce at least one error")

	foundCaptive := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "captive dependency") &&
			strings.Contains(msg, "SingletonSvc") &&
			strings.Contains(msg, "ScopedSvc") {
			foundCaptive = true
			break
		}
	}
	testutil.True(t, foundCaptive,
		"error must contain 'captive dependency' + singleton name + scoped name")
}

// --- 5.2-PLAN-T2 (AC2, P0) — transitive captive through transient -----------

// TestValidate_CaptiveDep_ThroughTransient: Singleton→Transient→Scoped.
// Transient is transparent for captive detection — the singleton still captures
// the scoped service because the transient is instantiated in singleton context.
func TestValidate_CaptiveDep_ThroughTransient(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "ScopedSvc",
		GoType:   "*svc.ScopedSvc",
		Ctor:     "svc.NewScopedSvc",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "TransientSvc",
		GoType:   "*svc.TransientSvc",
		Ctor:     "svc.NewTransientSvc",
		Lifetime: plan.Transient,
		Deps:     []string{"ScopedSvc"},
	})
	reg.Service(plan.Service{
		Name:     "SingletonSvc",
		GoType:   "*svc.SingletonSvc",
		Ctor:     "svc.NewSingletonSvc",
		Lifetime: plan.Singleton,
		Deps:     []string{"TransientSvc"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 1,
		"captive through transient must be detected")

	foundCaptive := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "captive dependency") &&
			strings.Contains(msg, "SingletonSvc") &&
			strings.Contains(msg, "ScopedSvc") {
			foundCaptive = true
			break
		}
	}
	testutil.True(t, foundCaptive,
		"error must mention the singleton and the captured scoped service")
}

// --- 5.2-PLAN-T3 (AC2, P0) — transitive captive through singleton -----------

// TestValidate_CaptiveDep_ThroughSingleton: Singleton_A→Singleton_B→Scoped.
// Both singletons report captive errors: B directly captures Scoped, and A's
// transitive closure contains Scoped.
func TestValidate_CaptiveDep_ThroughSingleton(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "ScopedSvc",
		GoType:   "*svc.ScopedSvc",
		Ctor:     "svc.NewScopedSvc",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "Singleton_B",
		GoType:   "*svc.SingletonB",
		Ctor:     "svc.NewSingletonB",
		Lifetime: plan.Singleton,
		Deps:     []string{"ScopedSvc"},
	})
	reg.Service(plan.Service{
		Name:     "Singleton_A",
		GoType:   "*svc.SingletonA",
		Ctor:     "svc.NewSingletonA",
		Lifetime: plan.Singleton,
		Deps:     []string{"Singleton_B"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 2,
		"both singletons A and B must report captive errors")

	foundA := false
	foundB := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "captive dependency") && strings.Contains(msg, "ScopedSvc") {
			if strings.Contains(msg, "Singleton_A") {
				foundA = true
			}
			if strings.Contains(msg, "Singleton_B") {
				foundB = true
			}
		}
	}
	testutil.True(t, foundA,
		"Singleton_A must have captive error for ScopedSvc (transitive)")
	testutil.True(t, foundB,
		"Singleton_B must have captive error for ScopedSvc (direct)")
}

// --- 5.2-PLAN-T4 (AC2, P0) — deep transitive: Singleton→Transient→Singleton→Scoped

// TestValidate_CaptiveDep_DeepTransitive: Singleton_X→Transient→Singleton_Y→Scoped.
// Both singletons must report: Y directly captures, X captures transitively.
func TestValidate_CaptiveDep_DeepTransitive(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "ScopedSvc",
		GoType:   "*svc.ScopedSvc",
		Ctor:     "svc.NewScopedSvc",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "Singleton_Y",
		GoType:   "*svc.SingletonY",
		Ctor:     "svc.NewSingletonY",
		Lifetime: plan.Singleton,
		Deps:     []string{"ScopedSvc"},
	})
	reg.Service(plan.Service{
		Name:     "TransientSvc",
		GoType:   "*svc.TransientSvc",
		Ctor:     "svc.NewTransientSvc",
		Lifetime: plan.Transient,
		Deps:     []string{"Singleton_Y"},
	})
	reg.Service(plan.Service{
		Name:     "Singleton_X",
		GoType:   "*svc.SingletonX",
		Ctor:     "svc.NewSingletonX",
		Lifetime: plan.Singleton,
		Deps:     []string{"TransientSvc"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 2,
		"deep transitive captive must produce errors for both singletons")

	foundX := false
	foundY := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "captive dependency") && strings.Contains(msg, "ScopedSvc") {
			if strings.Contains(msg, "Singleton_X") {
				foundX = true
			}
			if strings.Contains(msg, "Singleton_Y") {
				foundY = true
			}
		}
	}
	testutil.True(t, foundX,
		"Singleton_X must report captive for ScopedSvc (deep transitive)")
	testutil.True(t, foundY,
		"Singleton_Y must report captive for ScopedSvc (direct)")
}

// --- 5.2-PLAN-T5 (AC3, P0) — legal: Singleton→Singleton→Singleton ----------

// TestValidate_NoCaptive_SingletonChain: a pure singleton chain with no scoped
// dependencies. Validate must pass with zero captive errors.
func TestValidate_NoCaptive_SingletonChain(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "Singleton_C",
		GoType:   "*svc.C",
		Ctor:     "svc.NewC",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "Singleton_B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"Singleton_C"},
	})
	reg.Service(plan.Service{
		Name:     "Singleton_A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"Singleton_B"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) == 0,
		"pure singleton chain must produce zero errors")
}

// --- 5.2-PLAN-T6 (AC3, P1) — legal: Scoped→Scoped --------------------------

// TestValidate_NoCaptive_ScopedDependsOnScoped: a scoped service depending on
// another scoped service is perfectly legal.
func TestValidate_NoCaptive_ScopedDependsOnScoped(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "Scoped_B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "Scoped_A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Scoped,
		Deps:     []string{"Scoped_B"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) == 0,
		"scoped depending on scoped is legal, must produce zero errors")
}

// --- 5.2-PLAN-T7 (AC3, P1) — legal: Transient→Scoped -----------------------

// TestValidate_NoCaptive_TransientDependsOnScoped: a transient depending on a
// scoped service is legal — the transient is instantiated in the scope's context
// and the scoped dependency lives as long as the scope.
func TestValidate_NoCaptive_TransientDependsOnScoped(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "ScopedSvc",
		GoType:   "*svc.ScopedSvc",
		Ctor:     "svc.NewScopedSvc",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "TransientSvc",
		GoType:   "*svc.TransientSvc",
		Ctor:     "svc.NewTransientSvc",
		Lifetime: plan.Transient,
		Deps:     []string{"ScopedSvc"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) == 0,
		"transient depending on scoped is legal, must produce zero errors")
}

// --- 5.2-PLAN-T8 (AC3, P1) — legal: Scoped→Singleton ------------------------

// TestValidate_NoCaptive_ScopedDependsOnSingleton: a scoped service depending
// on a singleton is legal.
func TestValidate_NoCaptive_ScopedDependsOnSingleton(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "SingletonSvc",
		GoType:   "*svc.SingletonSvc",
		Ctor:     "svc.NewSingletonSvc",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "ScopedSvc",
		GoType:   "*svc.ScopedSvc",
		Ctor:     "svc.NewScopedSvc",
		Lifetime: plan.Scoped,
		Deps:     []string{"SingletonSvc"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) == 0,
		"scoped depending on singleton is legal, must produce zero errors")
}

// --- 5.2-PLAN-T9 (AC1/AC2, P0) — multiple captives, aggregated --------------

// TestValidate_CaptiveDep_MultipleCaptives_Aggregated: two independent captive
// violations (Singleton_X→Scoped_A, Singleton_Y→Scoped_B). Both must be
// reported in the aggregated []error.
func TestValidate_CaptiveDep_MultipleCaptives_Aggregated(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "Scoped_A",
		GoType:   "*svc.ScopedA",
		Ctor:     "svc.NewScopedA",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "Scoped_B",
		GoType:   "*svc.ScopedB",
		Ctor:     "svc.NewScopedB",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "Singleton_X",
		GoType:   "*svc.SingletonX",
		Ctor:     "svc.NewSingletonX",
		Lifetime: plan.Singleton,
		Deps:     []string{"Scoped_A"},
	})
	reg.Service(plan.Service{
		Name:     "Singleton_Y",
		GoType:   "*svc.SingletonY",
		Ctor:     "svc.NewSingletonY",
		Lifetime: plan.Singleton,
		Deps:     []string{"Scoped_B"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 2,
		"two independent captive violations must produce at least two errors")

	foundX := false
	foundY := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "captive dependency") {
			if strings.Contains(msg, "Singleton_X") && strings.Contains(msg, "Scoped_A") {
				foundX = true
			}
			if strings.Contains(msg, "Singleton_Y") && strings.Contains(msg, "Scoped_B") {
				foundY = true
			}
		}
	}
	testutil.True(t, foundX,
		"Singleton_X captive of Scoped_A must be reported")
	testutil.True(t, foundY,
		"Singleton_Y captive of Scoped_B must be reported")
}

// --- 5.2-PLAN-T10 (AC3, P0) — mixed valid and captive -----------------------

// TestValidate_CaptiveDep_MixedValidAndCaptive: a legal singleton chain
// coexists with a captive violation. Only the captive chain must report errors.
func TestValidate_CaptiveDep_MixedValidAndCaptive(t *testing.T) {
	reg := plan.NewRegistry()
	// Legal chain: Singleton_X → Singleton_Y (no scoped)
	reg.Service(plan.Service{
		Name:     "Singleton_Y",
		GoType:   "*svc.Y",
		Ctor:     "svc.NewY",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "Singleton_X",
		GoType:   "*svc.X",
		Ctor:     "svc.NewX",
		Lifetime: plan.Singleton,
		Deps:     []string{"Singleton_Y"},
	})
	// Captive chain: Singleton_A → Scoped_B
	reg.Service(plan.Service{
		Name:     "Scoped_B",
		GoType:   "*svc.ScopedB",
		Ctor:     "svc.NewScopedB",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "Singleton_A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"Scoped_B"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 1,
		"captive violation must be reported even when legal chains coexist")

	// Only captive chain errors; legal chain must not produce errors
	for _, e := range errs {
		msg := e.Error()
		testutil.True(t, !strings.Contains(msg, "Singleton_X"),
			"legal service Singleton_X must not appear in errors")
		testutil.True(t, !strings.Contains(msg, "Singleton_Y"),
			"legal service Singleton_Y must not appear in errors")
	}

	foundCaptive := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "captive dependency") &&
			strings.Contains(msg, "Singleton_A") &&
			strings.Contains(msg, "Scoped_B") {
			foundCaptive = true
			break
		}
	}
	testutil.True(t, foundCaptive,
		"Singleton_A captive of Scoped_B must be reported")
}

// --- 5.2-PLAN-T11 (AC1, P1) — slice dep captive: Singleton→[]IFoo(Scoped) ---

// TestValidate_CaptiveDep_SliceDep: a Singleton depends on []IFoo where the
// IFoo implementation is Scoped. The captive must be detected through the slice
// dependency resolution.
func TestValidate_CaptiveDep_SliceDep(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Service(plan.Service{
		Name:     "FooImpl",
		GoType:   "svc.IFoo",
		Ctor:     "svc.NewFooImpl",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "SingletonSvc",
		GoType:   "*svc.SingletonSvc",
		Ctor:     "svc.NewSingletonSvc",
		Lifetime: plan.Singleton,
		Deps:     []string{"[]svc.IFoo"},
	})

	errs := reg.Validate()

	testutil.True(t, len(errs) >= 1,
		"captive through slice dep must be detected")

	foundCaptive := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "captive dependency") &&
			strings.Contains(msg, "SingletonSvc") {
			foundCaptive = true
			break
		}
	}
	testutil.True(t, foundCaptive,
		"SingletonSvc must report captive dependency through its slice dep")
}
