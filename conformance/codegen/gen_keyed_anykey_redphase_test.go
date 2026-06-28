//go:build atdd_redphase

// ATDD scaffold for Story 3.3 (AC2/AC3/AC6, Task 6) — codegen conformance: Path γ
// STRICT tests for AnyKey-query + AnyKey single-resolve panic on the pre-generated
// B2 container, plus the deterministic-regen guard.
//
// Path γ (AD-15): AnyKey-query registration order and the AnyKey single-resolve
// panic are 保真敏感 behaviors that must run STRICTLY on the generated container,
// not only via cross-provider differential equivalence.
//
// RED PHASE — this scaffold fails before implementation because:
//   - registration.go has no IAnyKeyService registrations and container_gen.go was
//     not regenerated (green-phase Task 6), so GetKeyedServices(IAnyKeyService,
//     AnyKey) returns nil.
//   - genlib does not emit an `if key == godi.AnyKey` branch in the generated
//     GetKeyedService/GetKeyedServices, so AnyKey neither panics nor collects (Task 4).
//
// Activation: delete the //go:build line and the blank line below it to join the
// default build, once Tasks 4-6 are implemented. See
// _bmad-output/test-artifacts/atdd-checklist-3-3-anykey-null-semantics.md.
package codegen_test

import (
	"os"
	"reflect"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
)

// --- 3.3-CGPATH-001 (AC2, P0) — Path γ: AnyKey-query order on B2 -------------

// TestCodegen_PathGamma_AnyKey_QueryRegistrationOrder: on the regenerated B2
// container, GetKeyedServices(IAnyKeyService, AnyKey) returns every concrete-keyed
// registration in REGISTRATION order [A, B, A] (interleaved keys alpha,beta,alpha),
// excluding the unkeyed registration. Green-phase Task 6 adds these registrations to
// conformance/codegen/registration.go and regenerates container_gen.go.
func TestCodegen_PathGamma_AnyKey_QueryRegistrationOrder(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	all := root.GetKeyedServices(reflect.TypeFor[fakes.IAnyKeyService](), godi.AnyKey)
	testutil.NotNil(t, all)
	testutil.Equal(t, len(all), 3)
	if len(all) != 3 {
		return // len assertion above already failed (red phase); avoid an index panic
	}
	testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*fakes.AnyKeyServiceA")
	testutil.Equal(t, reflect.TypeOf(all[1]).String(), "*fakes.AnyKeyServiceB") // grouped-by-key would put B last
	testutil.Equal(t, reflect.TypeOf(all[2]).String(), "*fakes.AnyKeyServiceA")
}

// --- 3.3-CGPATH-002 (AC3, P0) — Path γ: AnyKey single-resolve panics on B2 ---

// TestCodegen_PathGamma_AnyKey_SingleResolvePanics: the generated container's
// GetKeyedService must panic for godi.AnyKey (AD-17), even though concrete keys
// exist. The emitted guard intercepts AnyKey after the nil check, before the
// type switch.
func TestCodegen_PathGamma_AnyKey_SingleResolvePanics(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	testutil.Panics(t, func() {
		_ = root.GetKeyedService(reflect.TypeFor[fakes.IAnyKeyService](), godi.AnyKey)
	})
}

// --- 3.3-CGPATH-002b (AC3, P0) — Path γ: AnyKey panic from a scope too -------

func TestCodegen_PathGamma_AnyKey_SingleResolvePanicsScope(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	sc := root.CreateScope()
	t.Cleanup(func() { _ = sc.Close() })

	testutil.Panics(t, func() {
		_ = sc.GetKeyedService(reflect.TypeFor[fakes.IAnyKeyService](), godi.AnyKey)
	})
}

// --- 3.3-CGPATH-003 (AC6, P1) — regen byte-identical guard -------------------
//
// Determinism guard (AD-14): committed container_gen.go must equal a fresh
// genlib.Generate of conformanceRegistry() (defined in gen_test.go). It PASSES both
// before AND after the AnyKey work, as long as registration.go (the generator's
// input) and gen_test.go's conformanceRegistry() (genlib's input) stay in sync —
// green-phase Task 6 adds the IAnyKeyService entries to BOTH and this guards that
// they regenerate identically (no hand-fork, no drift).
func TestCodegen_RegenComparison_WithAnyKey(t *testing.T) {
	reg := conformanceRegistry()
	freshSrc, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)

	_, thisFile, _, _ := goruntime.Caller(0)
	containerPath := strings.Replace(thisFile, "gen_keyed_anykey_redphase_test.go", "container_gen.go", 1)
	committed, readErr := os.ReadFile(containerPath)
	testutil.NoError(t, readErr)

	testutil.Equal(t, string(freshSrc), string(committed))
}
