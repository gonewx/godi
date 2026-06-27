// Compile-time contract proof migrated from the stub-based contract_alias_test.go
// (Story 1.2 R2) to the real codegen provider (Story 1.5 AC2). The var
// declaration below is the build-time guard: any drift in godi.Provider's
// method face that codegen.Root no longer satisfies fails the build here
// rather than at first resolve (AD-16).
package conformance

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
)

// Compile-time proof: codegen.Root satisfies godi.Provider (AD-16).
// Replaces the stub-based proof from Story 1.2 (R2).
var _ godi.Provider = (*codegen.Root)(nil)

// --- 1.5-CTR-001 (AC2/AC3, P0) -----------------------------------------------

// TestContract_CodegenFlowsThroughGodiProvider: the pre-generated codegen
// container resolves IFakeService through the godi.Provider interface, proving
// the conformance suite references the single source of truth (AD-14 / AD-16).
func TestContract_CodegenFlowsThroughGodiProvider(t *testing.T) {
	var p godi.Provider = codegen.NewRoot()
	t.Cleanup(func() { _ = p.Close() })

	got := p.GetService(reflect.TypeFor[fakes.IFakeService]())
	testutil.NotNil(t, got)
	testutil.Equal(t, reflect.TypeOf(got).String(), "*fakes.FakeService")
	testutil.True(t, !p.IsService(reflect.TypeFor[int]()), "unregistered type must not be a service")
}
