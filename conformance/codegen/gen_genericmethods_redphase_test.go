// ATDD test for Story 6.6 (AC3/AC4, Task 4) — codegen conformance:
// Root.Get[T]() / Root.GetRequired[T]() equivalence with the free functions
// godi.Get[T](p) / godi.GetRequired[T](p), on the B2 generated container.
//
// Activated in Story 6.6 Task 4 (container_gen.go regenerated with Get[T]/
// GetRequired[T]; original //go:build atdd_redphase removed).
package codegen_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
)

// --- 6.6-CG-001 (AC3, P0) — Root.Get[T] hit == free godi.Get[T] -------------

// TestRoot_Get_MatchesFreeFunction_Hit: for a registered singleton,
// root.Get[T]() and godi.Get[T](root) must resolve to the SAME instance —
// both ultimately dispatch through Root.GetService(reflect.TypeFor[T]()).
func TestRoot_Get_MatchesFreeFunction_Hit(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	methodVal, methodOk := root.Get[fakes.IFakeService]()
	freeVal, freeOk := godi.Get[fakes.IFakeService](root)

	testutil.True(t, methodOk, "root.Get[IFakeService]() must hit — FakeService is registered")
	testutil.True(t, freeOk, "godi.Get[IFakeService](root) must hit — FakeService is registered")
	testutil.True(t, methodVal == freeVal,
		"root.Get[T]() and godi.Get[T](root) must resolve to the identical instance")
}

// --- 6.6-CG-002 (AC3, P0) — Root.GetRequired[T] hit == free GetRequired[T] -

func TestRoot_GetRequired_MatchesFreeFunction_Hit(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	methodVal := root.GetRequired[fakes.IFakeService]()
	freeVal := godi.GetRequired[fakes.IFakeService](root)

	testutil.True(t, methodVal == freeVal,
		"root.GetRequired[T]() and godi.GetRequired[T](root) must resolve to the identical instance")
}

// --- 6.6-CG-003 (AC3, P0) — Root.Get[T] miss == (zero, false) --------------

// TestRoot_Get_Miss_ReturnsZeroFalse: an unregistered type must report a
// clean miss — never a panic (AD-16 unmet-lookup semantics carried through
// the method form).
func TestRoot_Get_Miss_ReturnsZeroFalse(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	v, ok := root.Get[int]()
	testutil.True(t, !ok, "root.Get[int]() must miss — int is not registered")
	testutil.Equal(t, v, 0)
}

// --- 6.6-CG-004 (AC3, P0) — Root.GetRequired[T] miss panics ----------------

// TestRoot_GetRequired_Miss_Panics: mirrors the free function's panic-on-miss
// contract (get.go:43-48).
func TestRoot_GetRequired_Miss_Panics(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	recovered := testutil.Panics(t, func() {
		root.GetRequired[int]()
	})
	_, ok := recovered.(error)
	testutil.True(t, ok, "root.GetRequired[int]() must panic with an error value")
}

// --- 6.6-CG-005 (AC3/AD-16, P0) — resolve-after-dispose panics -------------

// TestRoot_Get_AfterClose_Panics: after Close, root.Get[T]() must panic with
// the shared ErrProviderClosed sentinel — same as GetService (AD-16
// resolve-after-dispose consistency, carried through the method form).
func TestRoot_Get_AfterClose_Panics(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	testutil.NoError(t, root.Close())

	recovered := testutil.Panics(t, func() {
		root.Get[fakes.IFakeService]()
	})
	err2, ok := recovered.(error)
	testutil.True(t, ok, "root.Get[T]() after Close must panic with an error value")
	if ok {
		testutil.True(t, errors.Is(err2, godi.ErrProviderClosed),
			"post-Close panic must be (or wrap) godi.ErrProviderClosed")
	}
}

// --- 6.6-CG-006 (AC2, P1) — Get[T] is not reachable through the Provider
// interface (AD-5 stays intact) -----------------------------------------

// TestRoot_ProviderInterface_HasNoGetMethod: godi.Provider must remain
// non-generic (AD-5) — a caller holding only a godi.Provider value (not the
// concrete *codegen.Root) has no Get[T]/GetRequired[T] available and must
// fall back to the free functions. This is a structural/documentation check:
// it verifies the concrete Root value assigned to a Provider variable is
// still usable through the unchanged 7-method face (AD-16), i.e. the new
// methods are additive sugar, not a Provider face change.
func TestRoot_ProviderInterface_HasNoGetMethod(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	var p godi.Provider = root
	testutil.True(t, p.GetService(reflect.TypeFor[fakes.IFakeService]()) != nil,
		"Provider face (GetService) must still resolve FakeService — Get[T] is additive, not a replacement")
}
