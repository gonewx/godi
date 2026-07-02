// ATDD test for Story 6.6 (AC2, Task 2) — genlib emit: container struct
// generic methods Get[T]/GetRequired[T] on Root and scope.
//
// Activated in Story 6.6 Task 2 (emitGenericGetMethods added to generate.go;
// original //go:build atdd_redphase removed).
package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
)

// --- 6.6-GEN-001 (AC2, P0) — Root.Get[T] signature emitted -------------------

// TestEmitGenericGetMethods_RootHasGet: Root must expose a method-level generic
// Get[T any]() (T, bool), delegating to r.GetService(reflect.TypeFor[T]()).
func TestEmitGenericGetMethods_RootHasGet(t *testing.T) {
	src, err := genlib.Generate(singleServiceRegistry(), "container")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t,
		strings.Contains(code, "func (r *Root) Get[T any]() (T, bool) {"),
		"Root must have method-level generic Get[T any]() (T, bool)")
	testutil.True(t,
		strings.Contains(code, "r.GetService(reflect.TypeFor[T]())"),
		"Root.Get[T] must delegate through r.GetService(reflect.TypeFor[T]()), same as free godi.Get[T]")
}

// --- 6.6-GEN-002 (AC2, P0) — Root.GetRequired[T] signature emitted ----------

// TestEmitGenericGetMethods_RootHasGetRequired: Root must expose
// GetRequired[T any]() T, panicking through godi.GetRequiredPanic[T]() rather
// than building its own fmt.Errorf (D-2: avoids a new "fmt" import in
// generated code).
func TestEmitGenericGetMethods_RootHasGetRequired(t *testing.T) {
	src, err := genlib.Generate(singleServiceRegistry(), "container")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t,
		strings.Contains(code, "func (r *Root) GetRequired[T any]() T {"),
		"Root must have method-level generic GetRequired[T any]() T")
	testutil.True(t,
		strings.Contains(code, "godi.GetRequiredPanic[T]()"),
		"Root.GetRequired[T] must panic via godi.GetRequiredPanic[T] (D-2), not a fresh fmt.Errorf")
}

// --- 6.6-GEN-003 (AC2, P0) — scope.Get[T] / GetRequired[T] emitted ----------

// TestEmitGenericGetMethods_ScopeHasGetAndGetRequired: scope must expose the
// same two methods.
func TestEmitGenericGetMethods_ScopeHasGetAndGetRequired(t *testing.T) {
	src, err := genlib.Generate(singleServiceRegistry(), "container")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t,
		strings.Contains(code, "func (s *scope) Get[T any]() (T, bool) {"),
		"scope must have method-level generic Get[T any]() (T, bool)")
	testutil.True(t,
		strings.Contains(code, "func (s *scope) GetRequired[T any]() T {"),
		"scope must have method-level generic GetRequired[T any]() T")
}

// --- 6.6-GEN-004 (AC2/D-3, P0) — scope.Get[T] delegates to s.GetService ----

// TestEmitGenericGetMethods_ScopeDelegatesToOwnGetService: this is the D-3
// regression guard. scope.Get[T] MUST call s.GetService(reflect.TypeFor[T]())
// — never s.root.GetService(reflect.TypeFor[T]()). Short-circuiting to root
// would silently break scoped/transient lifetime semantics on the method
// form while the free function godi.Get[T](scope) (which goes through the
// Provider interface's GetService) stays correct — the exact divergence
// AC3 exists to prevent.
func TestEmitGenericGetMethods_ScopeDelegatesToOwnGetService(t *testing.T) {
	src, err := genlib.Generate(singleServiceRegistry(), "container")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t,
		strings.Contains(code, "s.GetService(reflect.TypeFor[T]())"),
		"scope.Get[T] must delegate through s.GetService(reflect.TypeFor[T]())")
	testutil.True(t,
		!strings.Contains(code, "s.root.GetService(reflect.TypeFor[T]())"),
		"scope.Get[T] must NOT short-circuit to s.root.GetService — breaks scoped/transient lifetime (D-3)")
}

// --- 6.6-GEN-005 (AC2/D-2, P1) — no new "fmt" import ------------------------

// TestEmitGenericGetMethods_NoFmtImport: emitting GetRequired[T] must not pull
// in a new "fmt" import — single-service.golden currently imports only
// "reflect" and "github.com/gonewx/godi" (D-2).
func TestEmitGenericGetMethods_NoFmtImport(t *testing.T) {
	src, err := genlib.Generate(singleServiceRegistry(), "container")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, !strings.Contains(code, `"fmt"`),
		"generated container must not import \"fmt\" — GetRequired[T] panics via godi.GetRequiredPanic[T] (D-2)")
}

// --- 6.6-GEN-006 (AC2/AC4, P1) — single-service golden updated -------------

// TestEmitGenericGetMethods_SingleServiceGoldenUpdated: the single-service
// golden (already updated once for Story 2.5's typed accessor) must be
// updated again to include Get[T]/GetRequired[T] on Root. Byte-stability
// guard (AD-14).
func TestEmitGenericGetMethods_SingleServiceGoldenUpdated(t *testing.T) {
	src, err := genlib.Generate(singleServiceRegistry(), "container")
	testutil.NoError(t, err)
	testutil.Golden(t, "single-service", src)
}
