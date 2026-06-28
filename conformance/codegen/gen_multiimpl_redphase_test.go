// ATDD red-phase scaffold for Story 2.4 (AC5, Task 6) — codegen conformance:
// Path γ strict tests for IEnumerable multi-impl on the pre-generated B2
// container.
//
// Path γ (AD-15): IEnumerable registration order is listed as a Path γ 升格
// scenario — it must be run STRICTLY on the pre-generated B2 container, not
// just via differential equivalence. This verifies the codegen output directly.
//
// The "red" comes from:
//   - container_gen.go does NOT have multi-impl registrations yet
//   - GetKeyedServices in the generated container returns nil
//   - conformanceRegistry() does not include multi-impl services
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-4-ienumerable-resolution.md.
package codegen_test

import (
	"os"
	"reflect"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// conformanceRegistryMultiImpl mirrors the UPDATED registration.go that
// includes multi-impl services (Story 2.4 Task 6). Used by the
// regeneration-comparison test.
func conformanceRegistryMultiImpl() *plan.Registry {
	return conformanceRegistry()
}

// --- 2.4-CGPATH-001 (AC5, P0) — Path γ: collection resolution strict on B2 --

// TestCodegen_PathGamma_MultiImpl_CollectionResolution: on the pre-generated
// B2 container, GetKeyedServices(IFakeMultipleService, nil) must return both
// implementations in registration order. AD-15 Path γ: IEnumerable 注册序
// is 保真敏感能力.
func TestCodegen_PathGamma_MultiImpl_CollectionResolution(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 2)
	testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*fakes.FakeOneMultipleService")
	testutil.Equal(t, reflect.TypeOf(all[1]).String(), "*fakes.FakeTwoMultipleService")
}

// --- 2.4-CGPATH-002 (AC5, P0) — Path γ: GetService last-wins strict on B2 ---

// TestCodegen_PathGamma_MultiImpl_LastWins: on the B2 container, GetService
// for a multi-impl type must return the LAST registered implementation.
func TestCodegen_PathGamma_MultiImpl_LastWins(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	got := root.GetService(reflect.TypeFor[fakes.IFakeMultipleService]())
	testutil.NotNil(t, got)
	testutil.Equal(t, reflect.TypeOf(got).String(), "*fakes.FakeTwoMultipleService")
}

// --- 2.4-CGPATH-003 (AC5/AC6, P0) — Path γ: OuterService injection strict ---

// TestCodegen_PathGamma_MultiImpl_OuterServiceInjection: on the B2 container,
// resolving IFakeOuterService must return a service with correctly injected
// single (IFakeService) and multi ([]IFakeMultipleService) dependencies.
func TestCodegen_PathGamma_MultiImpl_OuterServiceInjection(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	got := root.GetService(reflect.TypeFor[fakes.IFakeOuterService]())
	testutil.NotNil(t, got)

	outer := got.(fakes.IFakeOuterService)

	// Single dependency
	testutil.NotNil(t, outer.SingleService())
	testutil.Equal(t, reflect.TypeOf(outer.SingleService()).String(), "*fakes.FakeService")

	// Multi dependency — registration order preserved
	multi := outer.MultipleServices()
	testutil.Equal(t, len(multi), 2)
	testutil.Equal(t, reflect.TypeOf(multi[0]).String(), "*fakes.FakeOneMultipleService")
	testutil.Equal(t, reflect.TypeOf(multi[1]).String(), "*fakes.FakeTwoMultipleService")
}

// --- 2.4-CGPATH-004 (AC4, P1) — Path γ: empty collection for unregistered ---

// TestCodegen_PathGamma_MultiImpl_EmptyCollection: on the B2 container,
// GetKeyedServices for an unregistered type returns nil.
func TestCodegen_PathGamma_MultiImpl_EmptyCollection(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(reflect.TypeFor[interface{ notRegistered() }](), nil)
	testutil.Nil(t, all)
}

// --- 2.4-CGPATH-005 (AC3, P1) — Path γ: single reg collection on B2 ---------

// TestCodegen_PathGamma_MultiImpl_SingleRegCollection: single-registration
// type also resolves as a collection with one element.
func TestCodegen_PathGamma_MultiImpl_SingleRegCollection(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(reflect.TypeFor[fakes.IFakeService](), nil)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 1)
	testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*fakes.FakeService")
}

// --- 2.4-CGPATH-006 (AC5, P0) — Regen byte-identical --------------------------

// TestCodegen_RegenComparison_WithMultiImpl: the committed container_gen.go
// must be byte-identical to a fresh genlib.Generate run with the multi-impl
// registry. This guards deterministic output (AD-14).
func TestCodegen_RegenComparison_WithMultiImpl(t *testing.T) {
	reg := conformanceRegistryMultiImpl()
	freshSrc, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)

	_, thisFile, _, _ := goruntime.Caller(0)
	containerPath := strings.Replace(thisFile, "gen_multiimpl_redphase_test.go", "container_gen.go", 1)
	committed, readErr := os.ReadFile(containerPath)
	testutil.NoError(t, readErr)

	testutil.Equal(t, string(freshSrc), string(committed))
}

// --- 2.4-CGPATH-007 (AC5, P1) — IsService consistency for multi-impl --------

// TestCodegen_PathGamma_MultiImpl_IsService: IsService must return true for
// all registered multi-impl types on the B2 container.
func TestCodegen_PathGamma_MultiImpl_IsService(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	testutil.True(t, root.IsService(reflect.TypeFor[fakes.IFakeMultipleService]()),
		"IFakeMultipleService must be registered")
	testutil.True(t, root.IsService(reflect.TypeFor[fakes.IFakeOuterService]()),
		"IFakeOuterService must be registered")
	testutil.True(t, !root.IsService(reflect.TypeFor[interface{ notRegistered() }]()),
		"unregistered type must report IsService=false")
}
