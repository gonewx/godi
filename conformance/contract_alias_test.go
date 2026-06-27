//go:build atdd_redphase

// ATDD red-phase scaffold for Story 1.2 (AC4) — conformance collapses onto the
// single source of truth for the Provider contract.
//
// Activation (TDD red -> green): delete the //go:build line above AND complete
// Story 1.2 Task 3 — replace harness.go's local Provider/Scope interface
// definitions with `type Provider = godi.Provider` and `type Scope = godi.Scope`
// and import the godi package. Until both are done this file fails to compile,
// because stubProvider.CreateScope() returns the harness Scope, which must BE
// godi.Scope for *stubProvider to satisfy godi.Provider. That compile-time link
// is the R2 integration check: it proves godi.Provider's method face is exactly
// the face AD-16 pins in the harness — a missing or drifted method here would
// not compile.
package conformance

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
)

// Compile-time proof the existing harness stub satisfies the shared contract
// with zero changes to its method set (driving the Task 3 alias).
var (
	_ godi.Provider = (*stubProvider)(nil)
	_ godi.Scope    = (Scope)(nil) // the harness Scope must alias godi.Scope
)

// TestContract_StubFlowsThroughGodiProvider (1.2-INT-001, AC4, P0): exercise the
// harness stub THROUGH the godi.Provider interface type, proving the conformance
// suite now resolves against the single source of truth rather than a local copy
// of the contract (AD-14 / AD-16). The existing TestSpec_* cases keep this honest
// as a regression guard once the alias lands.
func TestContract_StubFlowsThroughGodiProvider(t *testing.T) {
	var p godi.Provider = newStub(map[reflect.Type]any{
		reflect.TypeFor[string](): "hello",
	})
	t.Cleanup(func() { _ = p.Close() })

	testutil.Equal(t, p.GetService(reflect.TypeFor[string]()).(string), "hello")
	testutil.True(t, !p.IsService(reflect.TypeFor[int]()), "int must not be a service")
}
