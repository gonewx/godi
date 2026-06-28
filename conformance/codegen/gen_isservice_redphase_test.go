// ATDD scaffold for Story 2.5 (AC8, Task 5) — codegen conformance:
// Path γ strict tests for IsService 覆盖義務 + typed accessor verification
// on the pre-generated B2 container.
//
// Path γ (AD-15): IsService is listed as a 覆盖義務 behavior axis — it must
// be verified on the codegen path with ≥1 scenario + cross-provider
// differential equivalence. Typed accessor verification runs STRICTLY on the
// B2 container (codegen.NewRoot() direct calls).
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
)

// --- 2.5-CG-001 (AC8, P0) — IsService true for all registered types ---------

// TestCodegen_PathGamma_IsService_RegisteredTypes: on the B2 container,
// IsService must return true for ALL registered types including the closed
// generic type (AC7). This is the Path γ strict verification (AD-15).
func TestCodegen_PathGamma_IsService_RegisteredTypes(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	// Existing registered types
	testutil.True(t, root.IsService(reflect.TypeFor[fakes.IFakeService]()),
		"IFakeService must be registered")
	testutil.True(t, root.IsService(reflect.TypeFor[fakes.IFakeConsumer]()),
		"IFakeConsumer must be registered")
	testutil.True(t, root.IsService(reflect.TypeFor[fakes.IFakeScopedService]()),
		"IFakeScopedService must be registered")
	testutil.True(t, root.IsService(reflect.TypeFor[fakes.IFakeTransientService]()),
		"IFakeTransientService must be registered")

	// Closed generic type (AC7, D-3: closed generic = ordinary registered type)
	testutil.True(t,
		root.IsService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]()),
		"IFakeOpenGenericService[IFakeService] must be registered (ClosedGenericsWithIsService)")
}

// --- 2.5-CG-002 (AC8, P0) — IsService false for unregistered ----------------

// TestCodegen_PathGamma_IsService_Unregistered: on the B2 container,
// IsService must return false for unregistered types.
func TestCodegen_PathGamma_IsService_Unregistered(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	testutil.True(t, !root.IsService(reflect.TypeFor[int]()),
		"unregistered type must report IsService=false")
}

// --- 2.5-CG-003 (AC4/AC8, P0) — IsService false for impl type ---------------

// TestCodegen_PathGamma_IsService_ImplTypeFalse: on the B2 container,
// IsService for implementation (concrete) types must return false — only
// service types (registered interfaces) are recognized (D-5).
func TestCodegen_PathGamma_IsService_ImplTypeFalse(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	testutil.True(t, !root.IsService(reflect.TypeFor[*fakes.FakeService]()),
		"IsService(*FakeService) must be false — impl type, not service type (D-5)")
}

// --- 2.5-CG-004 (AC1, P0) — Typed accessor on Root --------------------------

// TestCodegen_PathGamma_TypedAccessor_Root: on the B2 container, Root must
// have typed accessor methods for non-transient services (AC1). Uses interface
// type assertion to detect the accessor method at runtime.
//
// Typed accessors are the AD-5 answer to interface-type resolution: instead
// of any(zero).(type) dispatch (which mis-matches nil interfaces), the
// generated container provides type-safe methods that return the concrete
// interface type.
func TestCodegen_PathGamma_TypedAccessor_Root(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	// Singleton typed accessor
	type singletonAccessor interface {
		FakeService() fakes.IFakeService
	}
	sa, ok := any(root).(singletonAccessor)
	testutil.True(t, ok,
		"codegen.Root must have typed accessor FakeService() fakes.IFakeService")
	if ok {
		svc := sa.FakeService()
		testutil.NotNil(t, svc)
		testutil.Equal(t, reflect.TypeOf(svc).String(), "*fakes.FakeService")
	}

	// Scoped typed accessor (root acts as root scope)
	type scopedAccessor interface {
		FakeScopedService() fakes.IFakeScopedService
	}
	sca, ok := any(root).(scopedAccessor)
	testutil.True(t, ok,
		"codegen.Root must have typed accessor FakeScopedService() fakes.IFakeScopedService")
	if ok {
		svc := sca.FakeScopedService()
		testutil.NotNil(t, svc)
	}
}

// --- 2.5-CG-005 (AC7/AC8, P0) — Closed generic resolution strict on B2 -----

// TestCodegen_PathGamma_ClosedGeneric_Resolution: on the B2 container,
// resolving the closed generic type must return the correct instance.
func TestCodegen_PathGamma_ClosedGeneric_Resolution(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	got := root.GetService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]())
	testutil.NotNil(t, got)
	if got == nil {
		return
	}
	testutil.Equal(t, reflect.TypeOf(got).String(),
		"*fakes.FakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IFakeService]")
}

// --- 2.5-CG-006 (AC8, P0) — Regen byte-identical ----------------------------

// TestCodegen_RegenComparison_WithIsService: the committed container_gen.go
// must be byte-identical to a fresh genlib.Generate run with the updated
// registry (including closed generic + typed accessors). Guards deterministic
// output (AD-14).
func TestCodegen_RegenComparison_WithIsService(t *testing.T) {
	reg := conformanceRegistry()
	freshSrc, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)

	_, thisFile, _, _ := goruntime.Caller(0)
	containerPath := strings.Replace(thisFile, "gen_isservice_redphase_test.go", "container_gen.go", 1)
	committed, readErr := os.ReadFile(containerPath)
	testutil.NoError(t, readErr)

	testutil.Equal(t, string(freshSrc), string(committed))
}
