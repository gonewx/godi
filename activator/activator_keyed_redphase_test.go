// ATDD scaffold for Story 3.2 (AC6) — ActivatorUtilities + keyed parameter.
//
// dotnet: CreateServiceWithKeyedParameter
// Go difference (D-6): godi's activator.CreateInstance matches explicit args by
// type. A keyed service is just an any value — the user resolves it from the
// provider and passes it as an explicit arg. No activator-level keyed awareness
// is needed.
//
// This test verifies that the existing CreateInstance mechanism correctly handles
// keyed service instances as explicit arguments — no new activator code is needed.
//
// This file began as an ATDD red-phase scaffold behind //go:build atdd_redphase;
// that constraint was removed once the keyed injection Fakes landed. It passes
// with the existing activator.CreateInstance — no activator code changes needed.
package activator_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/activator"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// --- 3.2-SPEC-009 (AC6, P1) — CreateInstance with keyed service arg ----------

// TestActivator_CreateInstanceWithKeyedParameter: a keyed service instance
// resolved from the provider is passed as an explicit argument to
// CreateInstance. The activator matches it by type (IKeyedService) to the
// constructor parameter. This exercises the dotnet CreateServiceWithKeyedParameter
// scenario without requiring any activator-level keyed awareness.
//
// Go difference (D-6): dotnet's test uses [FromKeyedServices] inside
// CreateInstance's ctor parameter resolution. In Go, the caller explicitly
// resolves the keyed service and passes the VALUE — activator is keyed-unaware.
func TestActivator_CreateInstanceWithKeyedParameter(t *testing.T) {
	// Build provider with a keyed IKeyedService
	b := runtime.NewBuilder()
	runtime.AddKeyedSingleton[fakes.IKeyedService](b, "service1", func(p godi.Provider) fakes.IKeyedService {
		return fakes.NewKeyedServiceOne()
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	// Resolve the keyed service (user-side resolution)
	keyedSvc := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "service1")
	testutil.NotNil(t, keyedSvc)

	// CreateInstance with the keyed service as explicit arg — activator matches
	// it by type (IKeyedService → KeyedServiceWithDep's constructor parameter)
	ctors := []any{fakes.NewKeyedServiceWithDep}
	result, err := activator.CreateInstance[*fakes.KeyedServiceWithDep](p, ctors, keyedSvc)
	testutil.NoError(t, err)
	testutil.NotNil(t, result)
	testutil.NotNil(t, result.Dep())
	testutil.Equal(t, reflect.TypeOf(result.Dep()).String(), "*fakes.KeyedServiceOne")

	// Verify it's the same instance we resolved (not a new one)
	testutil.Equal(t, result.Dep(), keyedSvc.(fakes.IKeyedService))
}
