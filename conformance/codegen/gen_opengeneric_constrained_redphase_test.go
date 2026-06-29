// ATDD scaffold for Story 4.2 (AC2/AC5, Task 4) — codegen conformance:
// Path γ STRICT tests for constrained open-generic closed materialization on
// the pre-generated B2 container (codegen.NewRoot() direct calls, not just
// differential equivalence).
//
// Path γ (AD-15): open-generic materialization (including constrained variants)
// is explicitly named a fidelity-sensitive capability that escalates to Path γ
// — it must be verified STRICTLY on the generated B2 container, not only via
// cross-provider differential equivalence. This file pins:
//
//	4.2-CGPATH-001 (AC2) — B2 satisfying [IPocoClass] returns 2 impls with injected singleton
//	4.2-CGPATH-002 (AC2 reverse) — B2 non-satisfying [IFakeService] returns 1 impl (unconstrained only)
//	4.2-CGPATH-003 (AC2) — B2 interface constraint [IClassImplementingIterable] returns 2 impls
//	4.2-CGPATH-004 (AC2) — B2 abstract constraint [IClassInheritingAbstract] returns 2 impls
//	4.2-CGPATH-005 (AC5) — regen byte-identical (consecutive generate idempotent, AD-14)
//
// ACTIVATED (Story 4.2): the build tag was removed once Task 2 regenerated the
// B2 container with the constrained instantiations — these Path γ tests now run
// STRICTLY on the generated codegen.NewRoot() and are GREEN.
package codegen_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
)

// --- 4.2-CGPATH-001 (AC2, P0, touch-point 2) --------------------------------

// TestCodegen_PathGamma_ConstrainedOpenGeneric_SatisfyingCount2: on the B2
// container, the satisfying closed type IFakeOpenGenericService[IPocoClass]
// resolves as a collection of 2 elements — unconstrained + constrained — and
// both elements' .Value() is the SAME IPocoClass singleton (dependency
// injection materialized correctly — dotnet ConstrainedOpenGenericServicesCanBe
// Resolved). Pointer identity within one provider is legal (R4).
func TestCodegen_PathGamma_ConstrainedOpenGeneric_SatisfyingCount2(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(
		reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IPocoClass]](), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 2)
	if len(all) != 2 {
		return
	}

	singleton := root.GetService(reflect.TypeFor[fakes.IPocoClass]()).(fakes.IPocoClass)
	testutil.NotNil(t, singleton)
	for _, svc := range all {
		gen := svc.(fakes.IFakeOpenGenericService[fakes.IPocoClass])
		testutil.Equal(t, gen.Value(), singleton)
	}
	// P1 fix: the two impls must be distinct dynamic types (unconstrained +
	// ConstrainedFakeOpenGenericService) — count==2 alone would pass a generator
	// that duplicated one impl, missing Story 4-2's core invariant.
	testutil.NotEqual(t, reflect.TypeOf(all[0]).String(), reflect.TypeOf(all[1]).String())
	testutil.True(t,
		strings.Contains(reflect.TypeOf(all[0]).String(), "ConstrainedFakeOpenGenericService") ||
			strings.Contains(reflect.TypeOf(all[1]).String(), "ConstrainedFakeOpenGenericService"),
		"one impl must be ConstrainedFakeOpenGenericService")
}

// --- 4.2-CGPATH-002 (AC2 reverse, P0) ---------------------------------------

// TestCodegen_PathGamma_ConstrainedOpenGeneric_NonSatisfyingCount1: on the B2
// container, the non-satisfying closed type IFakeOpenGenericService[IFakeService]
// returns exactly 1 element (unconstrained only). The constrained helper
// AddConstrainedOpenGenericService[IFakeService] does not compile (AC1: Go
// compiler rejects IFakeService for PocoClassLike), so no constrained impl is
// registered. This is the Go compile-time constraint filtering positive acceptance.
func TestCodegen_PathGamma_ConstrainedOpenGeneric_NonSatisfyingCount1(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(
		reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]](), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 1)
}

// --- 4.2-CGPATH-003 (AC2, P0, touch-point 4) --------------------------------

// TestCodegen_PathGamma_ConstrainedOpenGeneric_InterfaceConstraintCount2: on the
// B2 container, interface constraint IClassImplementingIterable returns 2 impls
// (unconstrained + ClassWithInterfaceConstraint). Mirrors dotnet
// InterfaceConstrainedOpenGenericServicesCanBeResolved (DISpec:702).
func TestCodegen_PathGamma_ConstrainedOpenGeneric_InterfaceConstraintCount2(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(
		reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IClassImplementingIterable]](), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 2)
	if len(all) != 2 {
		return
	}

	singleton := root.GetService(reflect.TypeFor[fakes.IClassImplementingIterable]()).(fakes.IClassImplementingIterable)
	testutil.NotNil(t, singleton)
	for _, svc := range all {
		gen := svc.(fakes.IFakeOpenGenericService[fakes.IClassImplementingIterable])
		testutil.Equal(t, gen.Value(), singleton)
	}
	// P1 fix: the two impls must be distinct dynamic types (unconstrained +
	// ClassWithInterfaceConstraint).
	testutil.NotEqual(t, reflect.TypeOf(all[0]).String(), reflect.TypeOf(all[1]).String())
	testutil.True(t,
		strings.Contains(reflect.TypeOf(all[0]).String(), "ClassWithInterfaceConstraint") ||
			strings.Contains(reflect.TypeOf(all[1]).String(), "ClassWithInterfaceConstraint"),
		"one impl must be ClassWithInterfaceConstraint")
}

// --- 4.2-CGPATH-004 (AC2, P0, touch-point 5) --------------------------------

// TestCodegen_PathGamma_ConstrainedOpenGeneric_AbstractConstraintCount2: on the
// B2 container, abstract-class constraint IClassInheritingAbstract returns 2
// impls (unconstrained + ClassWithAbstractClassConstraint). Mirrors dotnet
// AbstractClassConstrainedOpenGenericServicesCanBeResolved (DISpec:725).
func TestCodegen_PathGamma_ConstrainedOpenGeneric_AbstractConstraintCount2(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(
		reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IClassInheritingAbstract]](), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 2)
	if len(all) != 2 {
		return
	}

	singleton := root.GetService(reflect.TypeFor[fakes.IClassInheritingAbstract]()).(fakes.IClassInheritingAbstract)
	testutil.NotNil(t, singleton)
	for _, svc := range all {
		gen := svc.(fakes.IFakeOpenGenericService[fakes.IClassInheritingAbstract])
		testutil.Equal(t, gen.Value(), singleton)
	}
	// P1 fix: the two impls must be distinct dynamic types (unconstrained +
	// ClassWithAbstractClassConstraint).
	testutil.NotEqual(t, reflect.TypeOf(all[0]).String(), reflect.TypeOf(all[1]).String())
	testutil.True(t,
		strings.Contains(reflect.TypeOf(all[0]).String(), "ClassWithAbstractClassConstraint") ||
			strings.Contains(reflect.TypeOf(all[1]).String(), "ClassWithAbstractClassConstraint"),
		"one impl must be ClassWithAbstractClassConstraint")
}

// --- 4.2-CGPATH-005 (AC5, P0) — regen idempotent ----------------------------

// TestCodegen_PathGamma_ConstrainedOpenGeneric_RegenIdempotent: generating from
// the same registry twice must produce byte-identical output (AD-14 deterministic
// emit). Constrained open-generic materialization (type args with sanitized
// segments from constraint impls) must not introduce nondeterminism into the
// generated container.
func TestCodegen_PathGamma_ConstrainedOpenGeneric_RegenIdempotent(t *testing.T) {
	reg := conformanceRegistry()
	first, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	second, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	testutil.Equal(t, string(first), string(second))
}
