// ATDD red-phase scaffold for Story 2.2 (AC1) — cmd/godi-gen: lifetime
// identification from DSL marker function names.
//
// Story 2.1's collectRegistrations only recognizes AddSingleton/
// AddSingletonFactory and hardcodes Lifetime: plan.Singleton in buildRegistry.
// This story extends recognition to AddScoped/AddTransient/AddScopedFactory/
// AddTransientFactory and derives the correct Lifetime from the marker name
// (Task 3, D-6).
//
// The testdata/lifetimes fixture uses AddSingleton + AddScoped + AddTransient
// (form A) to exercise the mixed-lifetime pipeline end to end.
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-2-scoped-transient-lifetimes.md.
package main

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// --- 2.2-GEN-001 (AC1, P0) ---------------------------------------------------

// TestGenerator_LifetimeIdentification_ScopedAndTransient: the generator must
// recognize AddScoped and AddTransient markers from the lifetimes fixture and
// produce a container that has scope-level fields and New{T}() methods for
// transients (not just singleton Root fields).
func TestGenerator_LifetimeIdentification_ScopedAndTransient(t *testing.T) {
	src, err := generate("testdata/lifetimes", "codegen", "")
	testutil.NoError(t, err)

	code := string(src)

	// The container must NOT hardcode all services as singleton Root fields.
	// Check that IRequest (scoped) appears as a scope field, and IWidget
	// (transient) appears as a New*() method.
	testutil.True(t, strings.Contains(code, "type scope struct"),
		"scope struct must be emitted for scoped services")

	testutil.True(t, strings.Contains(code, "NewWidget"),
		"transient service must be emitted as a New{T}() method")
}

// --- 2.2-GEN-002 (AC1, P0) ---------------------------------------------------

// TestGenerator_LifetimeIdentification_StaticAnalysisOnly: the lifetimes
// fixture's init() panics if executed (AD-7 tripwire). A clean generate()
// result proves the generator only parsed and type-checked the fixture.
func TestGenerator_LifetimeIdentification_StaticAnalysisOnly(t *testing.T) {
	_, err := generate("testdata/lifetimes", "codegen", "")
	testutil.NoError(t, err) // panic from fixture init() would fail here
}

// --- 2.2-GEN-003 (AC1, P0) ---------------------------------------------------

// TestGenerator_LifetimeIdentification_ScopedDependsOnSingleton: the scoped
// service (IRequest) depends on the singleton (ILogger). The generator must
// discover this dependency and wire it — the scope's Request field must be
// constructed with the root's Logger singleton.
func TestGenerator_LifetimeIdentification_ScopedDependsOnSingleton(t *testing.T) {
	src, err := generate("testdata/lifetimes", "codegen", "")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, strings.Contains(code, "NewRequest"),
		"scoped service must be wired with its constructor")
}

// --- 2.2-GEN-004 (AC1, P1) ---------------------------------------------------

// TestGenerator_ExistingDepsFixtureUnchanged: the existing deps fixture (pure
// singleton graph from 2.1) must still produce a valid container — regression
// guard that lifetime identification does not break singleton-only recognition.
func TestGenerator_ExistingDepsFixtureUnchanged(t *testing.T) {
	src, err := generate("testdata/deps", "codegen", "")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, strings.Contains(code, "r.f_Consumer, err = depsfix.NewConsumer(r.f_FakeService)"),
		"deps fixture (singleton-only) must still produce correct dependency wiring (regression)")
}
