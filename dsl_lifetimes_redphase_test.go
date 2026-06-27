//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.2 (AC1) — DSL marker contract for
// scoped and transient lifetimes.
//
// Story 2.1 delivered AddSingleton[I,Impl] (form A) and AddSingletonFactory[I]
// (form B). This story adds four markers for the remaining two lifetimes:
//
//	godi.AddScoped[I, Impl](r *Registry) *Registry         // form A
//	godi.AddTransient[I, Impl](r *Registry) *Registry       // form A
//	godi.AddScopedFactory[I](r *Registry, factory) *Registry   // form B
//	godi.AddTransientFactory[I](r *Registry, factory) *Registry // form B
//
// All are PURE static markers (AD-7): they return r unchanged and have no
// runtime behavior. The generator reads them BY NAME and maps the lifetime
// (Scoped/Transient) from the function name (D-6).
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-2-scoped-transient-lifetimes.md.
package godi

import (
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// Local stand-ins so DSL markers are exercised without real service types.
type iScopedSeed interface{ scopedSeed() }
type scopedSeedImpl struct{}

func (scopedSeedImpl) scopedSeed() {}

type iTransientSeed interface{ transientSeed() }
type transientSeedImpl struct{}

func (transientSeedImpl) transientSeed() {}

// --- 2.2-DSL-001 (AC1, P0) ---------------------------------------------------

// TestDSL_AddScopedIsPureMarker: AddScoped[I, Impl] must be a PURE static
// marker (AD-7) — returns r unchanged, no runtime behavior.
func TestDSL_AddScopedIsPureMarker(t *testing.T) {
	var r Registry

	got := AddScoped[iScopedSeed, scopedSeedImpl](&r)

	testutil.NotNil(t, got)
	testutil.Equal(t, got, &r)
}

// --- 2.2-DSL-002 (AC1, P0) ---------------------------------------------------

// TestDSL_AddTransientIsPureMarker: AddTransient[I, Impl] must be a PURE
// static marker (AD-7).
func TestDSL_AddTransientIsPureMarker(t *testing.T) {
	var r Registry

	got := AddTransient[iTransientSeed, transientSeedImpl](&r)

	testutil.NotNil(t, got)
	testutil.Equal(t, got, &r)
}

// --- 2.2-DSL-003 (AC1, P0) ---------------------------------------------------

// TestDSL_AddScopedFactoryIsPureMarker: AddScopedFactory[I] must be a PURE
// static marker (AD-7) — form B escape hatch for scoped lifetime (D-6).
func TestDSL_AddScopedFactoryIsPureMarker(t *testing.T) {
	var r Registry

	factory := func(Provider) iScopedSeed { return scopedSeedImpl{} }
	got := AddScopedFactory[iScopedSeed](&r, factory)

	testutil.NotNil(t, got)
	testutil.Equal(t, got, &r)
}

// --- 2.2-DSL-004 (AC1, P0) ---------------------------------------------------

// TestDSL_AddTransientFactoryIsPureMarker: AddTransientFactory[I] must be a
// PURE static marker (AD-7) — form B escape hatch for transient lifetime (D-6).
func TestDSL_AddTransientFactoryIsPureMarker(t *testing.T) {
	var r Registry

	factory := func(Provider) iTransientSeed { return transientSeedImpl{} }
	got := AddTransientFactory[iTransientSeed](&r, factory)

	testutil.NotNil(t, got)
	testutil.Equal(t, got, &r)
}

// --- 2.2-DSL-005 (AC1, P1) ---------------------------------------------------

// TestDSL_SixMarkersCoexistAndChain: all six DSL markers (AddSingleton/
// AddScoped/AddTransient × form A/B) are DISTINCT names (D-6: Go forbids
// overloading) and can be chained against one registry.
func TestDSL_SixMarkersCoexistAndChain(t *testing.T) {
	var r Registry

	got := AddTransientFactory[iTransientSeed](
		AddScopedFactory[iScopedSeed](
			AddTransient[iTransientSeed, transientSeedImpl](
				AddScoped[iScopedSeed, scopedSeedImpl](
					AddSingletonFactory[iLoggerSeed](
						AddSingleton[iLoggerSeed, loggerSeed](&r),
						func(Provider) iLoggerSeed { return loggerSeed{} },
					),
				),
			),
			func(Provider) iScopedSeed { return scopedSeedImpl{} },
		),
		func(Provider) iTransientSeed { return transientSeedImpl{} },
	)

	testutil.Equal(t, got, &r)
}
