//go:build atdd_redphase

// ATDD scaffold for Story 1.5 — compile-time contract proof migrated from
// stub-based contract_alias_test.go to the real codegen provider (AC2).
//
// Red cause: codegen.Root / codegen.NewRoot undefined → compilation failure.
//
// Activation (TDD red → green):
//  1. Implement Tasks 1–3 and Task 5 (fakes, codegen container, contract migration).
//  2. Delete contract_alias_test.go (stub-based proofs).
//  3. Delete the //go:build line above and the blank line after it.
//  4. Rename this file to contract_alias_test.go.
//  5. GOWORK=off go test -race ./conformance/... → green.
//
// See _bmad-output/test-artifacts/atdd-checklist-1-5-conformance-dual-provider.md.
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
