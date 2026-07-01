//go:build atdd_redphase

// ATDD red-phase scaffold for Story 6.4 (AC1) — godi package: typed resolution
// extension functions GetServices[T], GetKeyedServices[T], GetKeyed[T].
//
// These functions DO NOT EXIST YET — this file will not compile until the
// dev-story implements them. The //go:build atdd_redphase tag isolates it from
// the default build.
//
// Ported from dotnet ServiceProviderServiceExtensionsTest (17 Fact, v10.0.9):
//   - GetServices[T] = typed []T wrapper over Provider.GetKeyedServices(t, nil)
//   - GetKeyedServices[T] = typed []T wrapper over Provider.GetKeyedServices(t, key)
//   - GetKeyed[T] = typed (T, bool) wrapper for non-required keyed resolve
//   - GetRequired[T] panic semantics = InvalidOperationException with type name
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-6-4-extensions-collection-ops.md.
package godi

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// ── test-local fakes (unexported, minimal) ─────────────────────────────────

type iExtSvc interface{ isExtSvc() }
type extSvcA struct{ _ int }
type extSvcB struct{ _ int }

func (*extSvcA) isExtSvc() {}
func (*extSvcB) isExtSvc() {}

// ── 6.4-SPEC: GetServices[T] (SPSE #7/#9/#11/#12/#14) ─────────────────────

func TestGetServices_ReturnsAllRegistered(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iExtSvc](b, func(Provider) iExtSvc { return &extSvcA{} })
	runtime.AddSingleton[iExtSvc](b, func(Provider) iExtSvc { return &extSvcB{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	result := GetServices[iExtSvc](p)
	testutil.Equal(t, len(result), 2)
	testutil.Equal(t, reflect.TypeOf(result[0]).String(), "*godi.extSvcA")
	testutil.Equal(t, reflect.TypeOf(result[1]).String(), "*godi.extSvcB")
}

func TestGetServices_ReturnsSingleService(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iExtSvc](b, func(Provider) iExtSvc { return &extSvcA{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	result := GetServices[iExtSvc](p)
	testutil.Equal(t, len(result), 1)
}

func TestGetServices_ReturnsCorrectDynamicTypes(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iExtSvc](b, func(Provider) iExtSvc { return &extSvcA{} })
	runtime.AddSingleton[iExtSvc](b, func(Provider) iExtSvc { return &extSvcB{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	result := GetServices[iExtSvc](p)
	types := make([]string, len(result))
	for i, s := range result {
		types[i] = reflect.TypeOf(s).String()
	}
	testutil.True(t, types[0] == "*godi.extSvcA", "first element dynamic type")
	testutil.True(t, types[1] == "*godi.extSvcB", "second element dynamic type")
}

func TestGetServices_ReturnsEmptyNonNilSlice_WhenUnregistered(t *testing.T) {
	b := runtime.NewBuilder()
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	result := GetServices[iExtSvc](p)
	testutil.True(t, result != nil, "GetServices must return non-nil slice for unregistered type")
	testutil.Equal(t, len(result), 0)
}

func TestGetServices_FreshBuild_ReturnsEmptySlice(t *testing.T) {
	b := runtime.NewBuilder()
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	result := GetServices[iExtSvc](p)
	testutil.True(t, result != nil, "GetServices from fresh Build must return non-nil empty slice")
	testutil.Equal(t, len(result), 0)
}

// ── 6.4-SPEC: GetKeyed[T] (SPSE maps to non-required keyed) ───────────────

func TestGetKeyed_ReturnsService_WhenRegistered(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddKeyedSingleton[iExtSvc](b, "alpha", func(Provider) iExtSvc { return &extSvcA{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	v, ok := GetKeyed[iExtSvc](p, "alpha")
	testutil.True(t, ok, "GetKeyed should return true for registered keyed service")
	testutil.Equal(t, reflect.TypeOf(v).String(), "*godi.extSvcA")
}

func TestGetKeyed_ReturnsFalse_WhenMissing(t *testing.T) {
	b := runtime.NewBuilder()
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	_, ok := GetKeyed[iExtSvc](p, "nonexistent")
	testutil.True(t, !ok, "GetKeyed should return false for unregistered keyed service")
}

// ── 6.4-SPEC: GetKeyedServices[T] ──────────────────────────────────────────

func TestGetKeyedServices_ReturnsAllKeyed(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddKeyedSingleton[iExtSvc](b, "k", func(Provider) iExtSvc { return &extSvcA{} })
	runtime.AddKeyedSingleton[iExtSvc](b, "k", func(Provider) iExtSvc { return &extSvcB{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	result := GetKeyedServices[iExtSvc](p, "k")
	testutil.Equal(t, len(result), 2)
}

// ── 6.4-SPEC: GetRequired[T] panic semantics (SPSE #3/#5) ─────────────────

func TestGetRequired_PanicsWithTypeName_WhenMissing(t *testing.T) {
	b := runtime.NewBuilder()
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	recovered := testutil.Panics(t, func() {
		GetRequired[iExtSvc](p)
	})
	if e, ok := recovered.(error); ok {
		testutil.True(t, strings.Contains(e.Error(), "iExtSvc"),
			"GetRequired panic message must include type name")
	} else {
		t.Errorf("GetRequired panic value should be an error, got %T", recovered)
	}
}
