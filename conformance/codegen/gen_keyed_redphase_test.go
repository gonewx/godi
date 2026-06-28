// ATDD scaffold for Story 3.1 (AC5/AC10, Task 8) — codegen
// conformance: Path γ strict tests for keyed DI resolution on the pre-generated
// B2 container + golden regeneration comparison with keyed registry.
//
// Path γ (AD-15): keyed resolution is listed as a Path γ 升格 scenario — it
// must run STRICTLY on the pre-generated B2 container, not just via differential
// equivalence. This verifies the codegen output directly.
//
// RED PHASE — this scaffold fails because:
//   - plan.Service has no Key field (compile error)
//   - container_gen.go does NOT have keyed registrations
//   - GetKeyedService in the generated container returns nil for non-nil keys
//   - registration.go does not include godi.AddKeyedSingleton calls
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-3-1-keyed-registration-resolve.md.
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

// conformanceRegistryKeyed returns the same registry as conformanceRegistry()
// which now includes keyed services (Story 3-1).
func conformanceRegistryKeyed() *plan.Registry {
	return conformanceRegistry()
}

// --- 3.1-CGPATH-001 (AC1, P0) — Path γ: keyed resolve strict on B2 ----------

// TestCodegen_PathGamma_Keyed_ResolveService: on the pre-generated B2 container,
// GetKeyedService must return the correct implementation for each registered key.
func TestCodegen_PathGamma_Keyed_ResolveService(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	st := reflect.TypeFor[fakes.IKeyedService]()

	result1 := root.GetKeyedService(st, "service1")
	testutil.NotNil(t, result1)
	testutil.Equal(t, reflect.TypeOf(result1).String(), "*fakes.KeyedServiceOne")

	result2 := root.GetKeyedService(st, "service2")
	testutil.NotNil(t, result2)
	testutil.Equal(t, reflect.TypeOf(result2).String(), "*fakes.KeyedServiceTwo")
}

// --- 3.1-CGPATH-002 (AC2, P0) — Path γ: unknown key returns nil -------------

func TestCodegen_PathGamma_Keyed_UnknownKeyNil(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	st := reflect.TypeFor[fakes.IKeyedService]()
	testutil.Nil(t, root.GetKeyedService(st, "unknown-key"))
	testutil.Nil(t, root.GetService(st))
}

// --- 3.1-CGPATH-003 (AC3, P0) — Path γ: keyed collection strict on B2 -------

// TestCodegen_PathGamma_Keyed_CollectionResolution: on the B2 container,
// GetKeyedServices(T, "both") must return both implementations in registration
// order. AD-15 Path γ: keyed collection registration order is 保真敏感能力.
func TestCodegen_PathGamma_Keyed_CollectionResolution(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(reflect.TypeFor[fakes.IKeyedService](), "both")
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 2)
	testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*fakes.KeyedServiceOne")
	testutil.Equal(t, reflect.TypeOf(all[1]).String(), "*fakes.KeyedServiceTwo")
}

// --- 3.1-CGPATH-004 (AC6, P0) — Path γ: singleton shared on B2 --------------

func TestCodegen_PathGamma_Keyed_SingletonShared(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	st := reflect.TypeFor[fakes.IKeyedService]()
	r1 := root.GetKeyedService(st, "singleton-factory")
	r2 := root.GetKeyedService(st, "singleton-factory")
	testutil.NotNil(t, r1)
	testutil.Equal(t, r1, r2)
}

// --- 3.1-CGPATH-005 (AC7, P0) — Path γ: transient distinct on B2 ------------

func TestCodegen_PathGamma_Keyed_TransientDistinct(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	st := reflect.TypeFor[fakes.IKeyedService]()
	r1 := root.GetKeyedService(st, "transient")
	r2 := root.GetKeyedService(st, "transient")
	testutil.NotNil(t, r1)
	testutil.NotNil(t, r2)
	testutil.True(t, r1 != r2, "transient keyed must return different instances")
}

// --- 3.1-CGPATH-006 (AC10, P0) — Regen byte-identical with keyed registry ----

// TestCodegen_RegenComparison_WithKeyed: the committed container_gen.go must be
// byte-identical to a fresh genlib.Generate run with the keyed registry. This
// guards deterministic output (AD-14) after keyed emit is implemented.
func TestCodegen_RegenComparison_WithKeyed(t *testing.T) {
	reg := conformanceRegistryKeyed()
	freshSrc, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)

	_, thisFile, _, _ := goruntime.Caller(0)
	containerPath := strings.Replace(thisFile, "gen_keyed_redphase_test.go", "container_gen.go", 1)
	committed, readErr := os.ReadFile(containerPath)
	testutil.NoError(t, readErr)

	testutil.Equal(t, string(freshSrc), string(committed))
}

// --- 3.1-CGPATH-007 (AC3, P1) — Path γ: single-key collection on B2 ---------

func TestCodegen_PathGamma_Keyed_SingleKeyCollection(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	// "service1" has only one registration → collection of one
	all := root.GetKeyedServices(reflect.TypeFor[fakes.IKeyedService](), "service1")
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 1)
	testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*fakes.KeyedServiceOne")
}

// --- 3.1-CGPATH-008 (AC2, P1) — Path γ: unknown key collection nil/empty ----

func TestCodegen_PathGamma_Keyed_UnknownKeyCollection(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(reflect.TypeFor[fakes.IKeyedService](), "not-registered")
	// Unknown non-nil key → nil (not empty slice, matching GetKeyedServices contract)
	testutil.Nil(t, all)
}
