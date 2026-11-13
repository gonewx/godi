// Story 4.1 (AC1/AC2/AC4) — generator unit coverage for the per-service
// open-generic helper (Mechanism A) against the testdata/opengeneric fixture.
//
// These tests isolate the generator from the conformance dual-provider suite:
// they prove the generator (1) RECOGNIZES the per-service helper AddBox[T] and
// reduces it to closed registrations, (2) SUBSTITUTES the type parameter so the
// closed constructor's injected `value T` becomes a concrete wired dependency,
// (3) SKIPS the inner template marker godi.AddTransient[IBox[T], Box[T]] (whose
// type args still mention T), and (4) DISAMBIGUATES the two closed instantiations
// into distinct field/method identifiers (AC4). The fixture is test INPUT under
// testdata/, never part of the module build.
package main

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// TestGenerator_PerServiceHelper_RecognizedAndReduced covers 4.1-GEN-001 (AC1).
//
// A successful generate is itself positive evidence the inner TEMPLATE marker was
// SKIPPED: were it processed, the open Box[T] constructor's `value T` parameter
// would read an unresolvable type parameter and Validate would reject the graph
// with a phantom missing-dependency error (AC1 skip rule).
func TestGenerator_PerServiceHelper_RecognizedAndReduced(t *testing.T) {
	src, err := generate("testdata/opengeneric", "opengenericfix", "")
	testutil.NoError(t, err)

	code := string(src)
	// Both closed instantiations are materialized as resolvable services (AC1).
	testutil.True(t, strings.Contains(code, "reflect.TypeFor[IBox[IDep]]()"),
		"helper AddBox[IDep] must materialize the closed IBox[IDep] service (Mechanism A)")
	testutil.True(t, strings.Contains(code, "reflect.TypeFor[IBox[IOther]]()"),
		"helper AddBox[IOther] must materialize the closed IBox[IOther] service (Mechanism A)")
}

// TestGenerator_PerServiceHelper_InjectsClosedDependency covers 4.1-GEN-002 (AC2).
//
// The closed constructor NewBox[T](value T) must be instantiated with the closed
// type args so the injected `value T` becomes a CONCRETE dependency wired from the
// registered service — NewBox[IDep](r.f_Dep), not NewBox[IDep]() with a phantom
// type-param parameter.
func TestGenerator_PerServiceHelper_InjectsClosedDependency(t *testing.T) {
	src, err := generate("testdata/opengeneric", "opengenericfix", "")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, strings.Contains(code, "NewBox[IDep](r.f_Dep)"),
		"closed Box[IDep] must inject the registered IDep dependency (AC2: ctor signature instantiated)")
	testutil.True(t, strings.Contains(code, "NewBox[IOther](r.f_Other)"),
		"closed Box[IOther] must inject the registered IOther dependency (AC2)")
}

// TestGenerator_PerServiceHelper_DisambiguatesInstantiations covers 4.1-GEN-003 (AC4).
//
// Two closed instantiations of the SAME open generic must get DISTINCT names
// (type args folded into the name): without disambiguation both would be "Box" →
// a plan.Validate duplicate and an f_Box field collision. Distinct method names
// prove the type args were folded (deferred-work.md:57).
func TestGenerator_PerServiceHelper_DisambiguatesInstantiations(t *testing.T) {
	src, err := generate("testdata/opengeneric", "opengenericfix", "")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, strings.Contains(code, "NewBox_opengenericfix_IDep()"),
		"closed Box[IDep] must get a type-arg-disambiguated method name (AC4)")
	testutil.True(t, strings.Contains(code, "NewBox_opengenericfix_IOther()"),
		"closed Box[IOther] must get a DISTINCT type-arg-disambiguated method name (AC4)")
}

// TestGenerator_PerServiceHelper_NoOpenTemplateLeak covers 4.1-GEN-004 (AC1).
//
// The generated container must NOT reference the open template Box[T] / IBox[T]
// (a bare type parameter) — that would mean the helper-body template marker
// leaked into the registry instead of being skipped.
func TestGenerator_PerServiceHelper_NoOpenTemplateLeak(t *testing.T) {
	src, err := generate("testdata/opengeneric", "opengenericfix", "")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, !strings.Contains(code, "IBox[T]") && !strings.Contains(code, "Box[T]"),
		"the open template Box[T]/IBox[T] must never appear in the generated container (AC1 skip rule)")
}
