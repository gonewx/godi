// Integration test for Story 1.2 (AC4) — conformance collapses onto the single
// source of truth for the Provider contract. harness.go's Provider / Scope are
// now type aliases for godi.Provider / godi.Scope, so stubProvider.CreateScope()
// returning the harness Scope (== godi.Scope) is the R2 compile-time check: it
// proves godi.Provider's method face is exactly the face AD-16 pins in the
// harness — a missing or drifted method would not compile here.
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
