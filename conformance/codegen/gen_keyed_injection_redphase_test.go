// ATDD scaffold for Story 3.2 (AC1-AC2, AC7, Task 4) — codegen conformance:
// Path γ strict tests for keyed lifecycle-via-scope + keyed dependency injection
// on the pre-generated B2 container + golden regeneration comparison.
//
// Path γ (AD-15): keyed lifecycle and injection are listed as Path γ scenarios —
// they must run STRICTLY on the pre-generated B2 container, not just via
// differential equivalence. This verifies the codegen output directly.
//
// This file began as an ATDD red-phase scaffold behind //go:build atdd_redphase;
// that constraint was removed once registration.go gained the injection DSL calls
// and container_gen.go was regenerated with the injection registrations.
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

// --- 3.2-CGPATH-004 (AC1, review P0) — Path γ: scoped keyed from root on B2 --

// TestCodegen_PathGamma_Keyed_ScopedFromRoot: on the pre-generated B2 container,
// a keyed scoped service resolved directly from root must return a real, non-nil
// instance (root acts as the root scope), distinct from a child scope's instance.
// Regression guard for review P0: NewRoot must materialize keyed scoped fields,
// mirroring non-keyed scoped and the runtime oracle.
func TestCodegen_PathGamma_Keyed_ScopedFromRoot(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	st := reflect.TypeFor[fakes.IKeyedService]()

	rootScoped := root.GetKeyedService(st, "scoped")
	testutil.NotNil(t, rootScoped)

	scope := root.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })
	scopeScoped := scope.GetKeyedService(st, "scoped")
	testutil.NotNil(t, scopeScoped)

	testutil.True(t, rootScoped != scopeScoped,
		"child scope keyed scoped must be isolated from the root-scope instance")
}

// --- 3.2-CGPATH-002 (AC2, P0) — Path γ: keyed dependency injection on B2 ----

// TestCodegen_PathGamma_Keyed_InjectionViaFormB: on the pre-generated B2
// container, a keyed service registered with form-B closure that calls
// GetKeyedService must correctly inject the keyed dependency.
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
// Note: this guards against drift (committed output ≠ fresh generation), not
// completeness — that the injection registrations behave correctly is covered by
// the Path γ and SpecCase tests above.
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
