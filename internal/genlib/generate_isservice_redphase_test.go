//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.5 (AC1, Task 1) — genlib emit:
// typed accessor methods for non-transient services on Root and scope.
//
// The "red" comes from:
//   - genlib currently does NOT emit typed accessor methods for non-transient
//     services (only New{Name}() for transients exists)
//   - No golden file reflects typed accessor shape
//   - Existing goldens will diverge once typed accessors are added
//
// Tests verify the emitted code CONTAINS typed accessor methods. Until
// emitTypedAccessors is implemented in genlib, these tests fail at runtime
// (string assertions on generated code miss the accessor signatures).
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-5-typed-resolve-isservice.md.
package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// typedAccessorRegistry returns a registry with singleton, scoped, and
// transient services — the minimum-viable input for testing typed accessor
// emit. Typed accessors are emitted for singletons and scoped services
// only; transients already have New{Name}() (D-6).
func typedAccessorRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"github.com/gonewx/godi/conformance/fakes"}
	reg.Service(plan.Service{
		Name:     "FakeService",
		GoType:   "fakes.IFakeService",
		Ctor:     "fakes.NewFakeService",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "FakeScopedService",
		GoType:   "fakes.IFakeScopedService",
		Ctor:     "fakes.NewFakeScopedService",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "FakeTransientService",
		GoType:   "fakes.IFakeTransientService",
		Ctor:     "fakes.NewFakeTransientService",
		Lifetime: plan.Transient,
	})
	return reg
}

// --- 2.5-GEN-001 (AC1, P0) ---------------------------------------------------

// TestEmitTypedAccessors_RootHasSingletonAccessor: Root struct must have a
// typed accessor method for singleton services:
//
//	func (r *Root) FakeService() fakes.IFakeService { return r.f_FakeService }
//
// Currently genlib does NOT emit this → red.
func TestEmitTypedAccessors_RootHasSingletonAccessor(t *testing.T) {
	src, err := genlib.Generate(typedAccessorRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	// Find Root's typed accessor for singleton
	testutil.True(t,
		strings.Contains(code, "func (r *Root) FakeService() fakes.IFakeService"),
		"Root must have typed accessor for singleton: FakeService() fakes.IFakeService")
}

// --- 2.5-GEN-002 (AC1, P0) ---------------------------------------------------

// TestEmitTypedAccessors_RootHasScopedAccessor: Root struct must have a typed
// accessor method for scoped services (root acts as root scope, D-2):
//
//	func (r *Root) FakeScopedService() fakes.IFakeScopedService
func TestEmitTypedAccessors_RootHasScopedAccessor(t *testing.T) {
	src, err := genlib.Generate(typedAccessorRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t,
		strings.Contains(code, "func (r *Root) FakeScopedService() fakes.IFakeScopedService"),
		"Root must have typed accessor for scoped: FakeScopedService() fakes.IFakeScopedService")
}

// --- 2.5-GEN-003 (AC1, P0) ---------------------------------------------------

// TestEmitTypedAccessors_ScopeHasSingletonAccessor: scope struct must have a
// typed accessor for singletons that delegates to root:
//
//	func (s *scope) FakeService() fakes.IFakeService { return s.root.f_FakeService }
func TestEmitTypedAccessors_ScopeHasSingletonAccessor(t *testing.T) {
	src, err := genlib.Generate(typedAccessorRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t,
		strings.Contains(code, "func (s *scope) FakeService() fakes.IFakeService"),
		"scope must have typed accessor for singleton: FakeService() fakes.IFakeService")
}

// --- 2.5-GEN-004 (AC1, P0) ---------------------------------------------------

// TestEmitTypedAccessors_ScopeHasScopedAccessor: scope struct must have a
// typed accessor for scoped services returning its own field:
//
//	func (s *scope) FakeScopedService() fakes.IFakeScopedService { return s.f_FakeScopedService }
func TestEmitTypedAccessors_ScopeHasScopedAccessor(t *testing.T) {
	src, err := genlib.Generate(typedAccessorRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t,
		strings.Contains(code, "func (s *scope) FakeScopedService() fakes.IFakeScopedService"),
		"scope must have typed accessor for scoped: FakeScopedService() fakes.IFakeScopedService")
}

// --- 2.5-GEN-005 (AC1, P1) ---------------------------------------------------

// TestEmitTypedAccessors_TransientNoAccessor: transient services must NOT get
// an additional typed accessor — they already have New{Name}() (D-6). Adding
// a {Name}() accessor would mislead callers into thinking the instance is
// cached (accessor naming convention = cached, New*() = fresh).
func TestEmitTypedAccessors_TransientNoAccessor(t *testing.T) {
	src, err := genlib.Generate(typedAccessorRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	// Transient should have New*() but NOT a bare accessor
	testutil.True(t, strings.Contains(code, "NewFakeTransientService()"),
		"transient must keep its New{Name}() method")
	testutil.True(t,
		!strings.Contains(code, "func (r *Root) FakeTransientService() fakes.IFakeTransientService"),
		"transient must NOT have a bare typed accessor (only New*())")
}

// --- 2.5-GEN-006 (AC1, P1) ---------------------------------------------------

// TestEmitTypedAccessors_Golden: the typed-accessor emit shape must be byte-
// stable. Existing goldens must be UPDATED to include typed accessor methods.
// Since we're using the lifetimes-container golden (which has all three
// lifetimes), this test verifies the golden updates correctly.
func TestEmitTypedAccessors_Golden(t *testing.T) {
	src, err := genlib.Generate(typedAccessorRegistry(), "codegen")
	testutil.NoError(t, err)
	testutil.Golden(t, "lifetimes-container", src)
}

// --- 2.5-GEN-007 (AC1, P1) ---------------------------------------------------

// TestEmitTypedAccessors_SingleServiceGoldenUpdated: the single-service golden
// (singleton Logger) must also be updated with a typed accessor.
func TestEmitTypedAccessors_SingleServiceGoldenUpdated(t *testing.T) {
	reg := singleServiceRegistry()
	src, err := genlib.Generate(reg, "container")
	testutil.NoError(t, err)
	testutil.Golden(t, "single-service", src)
}
