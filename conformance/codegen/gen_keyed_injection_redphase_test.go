//go:build atdd_redphase

// ATDD scaffold for Story 3.2 (AC1-AC2, AC7, Task 4) — codegen conformance:
// Path γ strict tests for keyed lifecycle-via-scope + keyed dependency injection
// on the pre-generated B2 container + golden regeneration comparison.
//
// Path γ (AD-15): keyed lifecycle and injection are listed as Path γ scenarios —
// they must run STRICTLY on the pre-generated B2 container, not just via
// differential equivalence. This verifies the codegen output directly.
//
// RED PHASE — this scaffold fails because:
//   - container_gen.go does NOT have injection registrations
//   - registration.go does not include injection DSL calls
//   - GetKeyedService for IKeyedServiceWithDep etc returns nil
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-3-2-keyed-lifetime-injection.md.
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

// --- 3.2-CGPATH-001 (AC1, P0) — Path γ: singleton keyed via scope on B2 -----

// TestCodegen_PathGamma_Keyed_SingletonViaScope: on the pre-generated B2
// container, a singleton keyed service resolved from root and from scope must
// be the SAME instance (pointer identity within one provider).
func TestCodegen_PathGamma_Keyed_SingletonViaScope(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	st := reflect.TypeFor[fakes.IKeyedService]()

	rootResult := root.GetKeyedService(st, "service1")
	testutil.NotNil(t, rootResult)

	scope := root.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })
	scopeResult := scope.GetKeyedService(st, "service1")
	testutil.NotNil(t, scopeResult)

	testutil.Equal(t, rootResult, scopeResult)
}

// --- 3.2-CGPATH-002 (AC2, P0) — Path γ: keyed dependency injection on B2 ----

// TestCodegen_PathGamma_Keyed_InjectionViaFormB: on the pre-generated B2
// container, a keyed service registered with form-B closure that calls
// GetKeyedService must correctly inject the keyed dependency.
//
// RED PHASE: this test fails because container_gen.go does not include
// IKeyedServiceWithDep registration. Once registration.go is updated and
// container regenerated, this should pass.
func TestCodegen_PathGamma_Keyed_InjectionViaFormB(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	st := reflect.TypeFor[fakes.IKeyedServiceWithDep]()
	result := root.GetKeyedService(st, "with-dep")
	if result == nil {
		t.Fatal("expected non-nil IKeyedServiceWithDep for key \"with-dep\"")
	}

	svc := result.(fakes.IKeyedServiceWithDep)
	testutil.NotNil(t, svc.Dep())
	testutil.Equal(t, reflect.TypeOf(svc.Dep()).String(), "*fakes.KeyedServiceOne")
}

// --- 3.2-CGPATH-003 (AC7, P0) — Regen byte-identical with injection registry -

// TestCodegen_RegenComparison_WithInjection: the committed container_gen.go
// must be byte-identical to a fresh genlib.Generate run with the registry that
// includes injection registrations. This guards deterministic output (AD-14).
//
// RED PHASE: this test uses conformanceRegistry() which includes Story 3-2
// registrations once registration.go is updated. Until then, the registry
// matches the current container_gen.go (both lack injection entries), so this
// test may PASS even in red phase — it guards against drift, not completeness.
func TestCodegen_RegenComparison_WithInjection(t *testing.T) {
	reg := conformanceRegistry()
	freshSrc, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)

	_, thisFile, _, _ := goruntime.Caller(0)
	containerPath := strings.Replace(thisFile, "gen_keyed_injection_redphase_test.go", "container_gen.go", 1)
	committed, readErr := os.ReadFile(containerPath)
	testutil.NoError(t, readErr)

	testutil.Equal(t, string(freshSrc), string(committed))
}
