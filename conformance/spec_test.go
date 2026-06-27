// Conformance specification suite for Story 1.5: dual-provider single-service
// spec cases + cross-provider differential equivalence (AD-14 / AD-15). Both
// factories build from the same logical registration (singleton IFakeService
// constructed by NewFakeService); divergence in observable behavior is a
// regression in the architecture's equivalence guarantee.
package conformance

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

var (
	rtFactory = ProviderFactory{
		Name: "runtime",
		Build: func(t *testing.T) Provider {
			t.Helper()
			b := runtime.NewBuilder()
			runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
				return fakes.NewFakeService()
			})
			return b.Build()
		},
	}
	cgFactory = ProviderFactory{
		Name: "codegen",
		Build: func(t *testing.T) Provider {
			t.Helper()
			return codegen.NewRoot()
		},
	}
)

var specCases = []SpecCase{
	{
		Name: "resolves IFakeService to non-nil with correct dynamic type",
		Run: func(t *testing.T, p Provider) {
			got := p.GetService(reflect.TypeFor[fakes.IFakeService]())
			testutil.NotNil(t, got)
			testutil.Equal(t, reflect.TypeOf(got).String(), "*fakes.FakeService")
		},
	},
	{
		Name: "unregistered type returns nil and IsService false (AD-16)",
		Run: func(t *testing.T, p Provider) {
			testutil.Nil(t, p.GetService(reflect.TypeFor[int]()))
			testutil.True(t, !p.IsService(reflect.TypeFor[int]()), "unregistered type must not be a service")
		},
	},
	{
		Name: "nil key routes to unkeyed service (AD-17)",
		Run: func(t *testing.T, p Provider) {
			got := p.GetKeyedService(reflect.TypeFor[fakes.IFakeService](), nil)
			testutil.NotNil(t, got)
			testutil.Equal(t, reflect.TypeOf(got).String(), "*fakes.FakeService")
		},
	},
}

// --- 1.5-SPEC-001/002 (AC1/AC2/AC4, P0) --------------------------------------

func TestConformance_RuntimeProvider(t *testing.T) { RunSuite(t, rtFactory, specCases) }

func TestConformance_CodegenProvider(t *testing.T) { RunSuite(t, cgFactory, specCases) }

// --- 1.5-SPEC-003 (AC1, P0) --------------------------------------------------

// TestConformance_DifferentialEquivalence: AD-15 cross-provider differential
// equivalence. Probes use provider-stable observables (IsService bool, dynamic
// type name) — never pointer identity (two providers build separate instances, R4).
func TestConformance_DifferentialEquivalence(t *testing.T) {
	AssertEquivalent(t, rtFactory, cgFactory, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeService]())
	})

	AssertEquivalent(t, rtFactory, cgFactory, func(p Provider) string {
		v := p.GetService(reflect.TypeFor[fakes.IFakeService]())
		if v == nil {
			return "<nil>"
		}
		return reflect.TypeOf(v).String()
	})

	AssertEquivalent(t, rtFactory, cgFactory, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[int]())
	})
}
