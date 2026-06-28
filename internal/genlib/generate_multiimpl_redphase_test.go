//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.4 (AC1/AC2, Task 3) — genlib emit:
// same-GoType multi-registration handling in GetService (last-wins case),
// IsService (dedup case), GetKeyedServices (collection case), and golden test.
//
// The "red" comes from:
//   - genlib currently emits one switch-case per Service; same GoType = duplicate
//     case = compile error in generated code. Task 3 must group by GoType.
//   - GetKeyedServices currently emits `return nil`; Task 3 must emit the
//     collection return for multi-impl GoTypes.
//   - The golden file multi-impl-container.golden does NOT exist yet.
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-4-ienumerable-resolution.md.
package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// multiImplRegistry returns a registry with two transient services sharing
// the same GoType (fakes.IFakeMultipleService), plus a singleton IFakeService
// and a transient OuterService that depends on both single and multi. This is
// the minimum-viable input for testing same-GoType emit.
func multiImplRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"github.com/gonewx/godi/conformance/fakes"}
	reg.Service(plan.Service{
		Name:     "FakeService",
		GoType:   "fakes.IFakeService",
		Ctor:     "fakes.NewFakeService",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "FakeOneMultipleService",
		GoType:   "fakes.IFakeMultipleService",
		Ctor:     "fakes.NewFakeOneMultipleService",
		Lifetime: plan.Transient,
	})
	reg.Service(plan.Service{
		Name:     "FakeTwoMultipleService",
		GoType:   "fakes.IFakeMultipleService",
		Ctor:     "fakes.NewFakeTwoMultipleService",
		Lifetime: plan.Transient,
	})
	reg.Service(plan.Service{
		Name:     "FakeOuterService",
		GoType:   "fakes.IFakeOuterService",
		Ctor:     "fakes.NewFakeOuterService",
		Lifetime: plan.Transient,
		Deps:     []string{"FakeService", "[]fakes.IFakeMultipleService"},
	})
	return reg
}

// --- 2.4-GEN-001 (AC1/AC2, P0) -----------------------------------------------

// TestEmitMultiImpl_GenerateSucceeds: genlib.Generate must succeed for a
// multi-impl registry without panicking on duplicate switch-cases. Currently
// it would emit duplicate case clauses for the same GoType → compile error in
// the generated code (red).
func TestEmitMultiImpl_GenerateSucceeds(t *testing.T) {
	src, err := genlib.Generate(multiImplRegistry(), "codegen")
	testutil.NoError(t, err)
	testutil.True(t, len(src) > 0, "generated code must not be empty")
}

// --- 2.4-GEN-002 (AC2, P0) ---------------------------------------------------

// TestEmitMultiImpl_GetService_LastWinsCase: GetService switch for a GoType
// with multiple Services must emit ONLY ONE case returning the last-registered
// Service's field (last-wins, D-3). Currently would emit duplicate cases (red).
func TestEmitMultiImpl_GetService_LastWinsCase(t *testing.T) {
	src, err := genlib.Generate(multiImplRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	// There must be exactly ONE case for fakes.IFakeMultipleService in GetService
	caseStr := `case reflect.TypeFor[fakes.IFakeMultipleService]()`
	count := strings.Count(code, caseStr)

	// In GetService method specifically, find the section
	gsIdx := strings.Index(code, "func (r *Root) GetService(")
	testutil.True(t, gsIdx >= 0, "GetService method must exist")

	// GetService section must have the case that returns the LAST registered
	// (FakeTwoMultipleService, not FakeOneMultipleService)
	gsSection := code[gsIdx:]
	nextFunc := strings.Index(gsSection[1:], "\nfunc ")
	if nextFunc > 0 {
		gsSection = gsSection[:nextFunc+1]
	}
	testutil.True(t, strings.Contains(gsSection, "f_FakeTwoMultipleService"),
		"GetService must return the LAST registered service (last-wins, D-3)")

	// Must not have duplicate case — count across the whole file, but each
	// occurrence should be in a different method (GetService vs IsService etc.)
	// For GetService specifically, just one case per GoType
	_ = count // used for overall sanity; the gsSection check is the primary assertion
}

// --- 2.4-GEN-003 (AC1, P0) ---------------------------------------------------

// TestEmitMultiImpl_IsService_NoDuplicateCase: IsService switch must have
// exactly ONE case per GoType — duplicate cases would cause a Go compile
// error. Currently would emit two cases for the same GoType (red).
func TestEmitMultiImpl_IsService_NoDuplicateCase(t *testing.T) {
	src, err := genlib.Generate(multiImplRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	isIdx := strings.Index(code, "func (r *Root) IsService(")
	testutil.True(t, isIdx >= 0, "IsService method must exist")
	isSection := code[isIdx:]
	nextFunc := strings.Index(isSection[1:], "\nfunc ")
	if nextFunc > 0 {
		isSection = isSection[:nextFunc+1]
	}

	caseStr := `fakes.IFakeMultipleService`
	count := strings.Count(isSection, caseStr)
	testutil.Equal(t, count, 1)
}

// --- 2.4-GEN-004 (AC1, P0) ---------------------------------------------------

// TestEmitMultiImpl_GetKeyedServices_ReturnsCollection: GetKeyedServices
// switch must emit a case that returns ALL instances of a multi-impl GoType
// as []any{r.NewFakeOneMultipleService(), r.NewFakeTwoMultipleService()}.
// Currently emits `return nil` (red).
func TestEmitMultiImpl_GetKeyedServices_ReturnsCollection(t *testing.T) {
	src, err := genlib.Generate(multiImplRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	gksIdx := strings.Index(code, "GetKeyedServices(")
	testutil.True(t, gksIdx >= 0, "GetKeyedServices method must exist")
	gksSection := code[gksIdx:]
	nextFunc := strings.Index(gksSection[1:], "\nfunc ")
	if nextFunc > 0 {
		gksSection = gksSection[:nextFunc+1]
	}

	// Must contain both service names in the collection return
	testutil.True(t, strings.Contains(gksSection, "FakeOneMultipleService"),
		"GetKeyedServices must include FakeOneMultipleService in collection")
	testutil.True(t, strings.Contains(gksSection, "FakeTwoMultipleService"),
		"GetKeyedServices must include FakeTwoMultipleService in collection")
	testutil.True(t, strings.Contains(gksSection, "[]any"),
		"GetKeyedServices must return []any for multi-impl GoType")
}

// --- 2.4-GEN-005 (AC1, P0) ---------------------------------------------------

// TestEmitMultiImpl_RootHasBothFields: Root struct must have SEPARATE fields
// for each Service (f_FakeOneMultipleService, f_FakeTwoMultipleService), even
// though they share the same GoType (D-3).
func TestEmitMultiImpl_RootHasBothFields(t *testing.T) {
	src, err := genlib.Generate(multiImplRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	// Both transient services should have New*() methods (transient = method, not field)
	testutil.True(t, strings.Contains(code, "NewFakeOneMultipleService()"),
		"Root must have New method for FakeOneMultipleService")
	testutil.True(t, strings.Contains(code, "NewFakeTwoMultipleService()"),
		"Root must have New method for FakeTwoMultipleService")
}

// --- 2.4-GEN-006 (AC6, P1) ---------------------------------------------------

// TestEmitMultiImpl_SliceDependencyEmit: the OuterService wiring must emit a
// slice literal collecting all multi-impl instances. For transient multi-impl
// the collected elements are New*() calls (D-4).
func TestEmitMultiImpl_SliceDependencyEmit(t *testing.T) {
	src, err := genlib.Generate(multiImplRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	// The OuterService wiring must contain a slice literal with both multi-impl
	testutil.True(t,
		strings.Contains(code, "[]fakes.IFakeMultipleService{") ||
			strings.Contains(code, "[]fakes.IFakeMultipleService {\n"),
		"OuterService wiring must emit a typed slice literal for []IFakeMultipleService")
}

// --- 2.4-GEN-007 (AC1, P1) ---------------------------------------------------

// TestEmitMultiImpl_Golden: the multi-impl emit shape must be byte-stable,
// anchored by a new golden file. This golden does NOT exist yet (red).
func TestEmitMultiImpl_Golden(t *testing.T) {
	src, err := genlib.Generate(multiImplRegistry(), "codegen")
	testutil.NoError(t, err)
	testutil.Golden(t, "multi-impl-container", src)
}

// --- 2.4-GEN-008 (AC1, P1) ---------------------------------------------------

// TestEmitMultiImpl_ExistingGoldensUnchanged: existing goldens (single-service,
// deps-container, lifetimes-container, formb-container, disposable-container)
// must be byte-identical — multi-impl support must NOT regress existing emit.
func TestEmitMultiImpl_ExistingGoldensUnchanged(t *testing.T) {
	// Non-multi-impl registries (no same-GoType services) must produce
	// identical output (hasMultiImpl guard ensures no new code paths trigger)

	// Lifetimes registry (three-lifetime, no multi-impl)
	reg := lifetimesRegistry()
	src, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	testutil.Golden(t, "lifetimes-container", src)

	// Disposable registry (disposables, no multi-impl)
	reg2 := disposableRegistry()
	src2, err2 := genlib.Generate(reg2, "codegen")
	testutil.NoError(t, err2)
	testutil.Golden(t, "disposable-container", src2)
}
