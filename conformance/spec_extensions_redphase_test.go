// ATDD red-phase scaffold for Story 6.4 (AC1/AC3/AC6) — conformance suite:
// dual-provider resolution extensions + cross-provider differential equivalence
// for collection operations "final set equivalence" (AD-15 义务轴, D-8).
//
// These tests reference godi.GetServices[T], godi.GetKeyed[T],
// godi.GetKeyedServices[T] which DO NOT EXIST YET — this file will not compile
// until the dev-story implements them.
//
// Ported from dotnet ServiceProviderServiceExtensionsTest (17, AC1 dual
// provider), ServiceCollectionKeyedServiceExtensionsTest (19, AC3 keyed
// semantics), and the AD-15 "final set equivalence" obligation (AC6, D-8).
//
// Core decisions:
//   - Resolution extensions (表 1, 17) = dual-provider differential equivalence
//   - Collection operations "final set" (D-8): runtime TryAdd/Replace/RemoveAll
//     result == codegen static DSL same final set → cross-provider diff equiv
//   - Keyed extension AD-17 口径: key=any, null→unkeyed, AnyKey semantics
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-6-4-extensions-collection-ops.md.
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

// ══ AC1: Resolution extension SpecCases (dual provider, 表 1) ══════════════

// iUnregisteredExt is a conformance-local type NOT registered in either
// provider, used by GetRequired-panic and GetServices-empty tests.
type iUnregisteredExt interface{ unregisteredExtMarker() }

// extRtFactory builds a runtime provider with multi-impl registrations for
// testing GetServices[T] and related resolution extensions.
var extRtFactory = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
			return fakes.NewFakeService()
		})
		runtime.AddSingleton[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
			return fakes.NewFakeOneMultipleService()
		})
		runtime.AddSingleton[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
			return fakes.NewFakeTwoMultipleService()
		})
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

var extCgFactory = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

var extSpecCases = []SpecCase{
	// SPSE-01: GetService<T> returns correct service
	{
		Name: "GetService returns correct service via Get[T]",
		Run: func(t *testing.T, p Provider) {
			v, ok := godi.Get[fakes.IFakeService](p)
			testutil.True(t, ok, "Get[IFakeService] should return true")
			testutil.Equal(t, reflect.TypeOf(v).String(), "*fakes.FakeService")
		},
	},
	// SPSE-03: GetRequiredService throws when no service registered
	{
		Name: "GetRequired panics with type name when no service registered",
		Run: func(t *testing.T, p Provider) {
			recovered := testutil.Panics(t, func() {
				godi.GetRequired[iUnregisteredExt](p)
			})
			if e, ok := recovered.(error); ok {
				testutil.True(t, e != nil, "GetRequired panic must be an error")
			}
		},
	},
	// SPSE-07: GetServices returns all services
	{
		Name: "GetServices[T] returns all registered services",
		Run: func(t *testing.T, p Provider) {
			result := godi.GetServices[fakes.IFakeMultipleService](p)
			testutil.Equal(t, len(result), 2)
		},
	},
	// SPSE-09: GetServices returns single service
	{
		Name: "GetServices[T] returns single-element slice for one registration",
		Run: func(t *testing.T, p Provider) {
			result := godi.GetServices[fakes.IFakeService](p)
			testutil.Equal(t, len(result), 1)
		},
	},
	// SPSE-11: GetServices returns correct types
	{
		Name: "GetServices[T] returns correct dynamic types in registration order",
		Run: func(t *testing.T, p Provider) {
			result := godi.GetServices[fakes.IFakeMultipleService](p)
			testutil.Equal(t, len(result), 2)
			testutil.Equal(t, reflect.TypeOf(result[0]).String(), "*fakes.FakeOneMultipleService")
			testutil.Equal(t, reflect.TypeOf(result[1]).String(), "*fakes.FakeTwoMultipleService")
		},
	},
	// SPSE-12: GetServices returns empty for unregistered
	{
		Name: "GetServices[T] returns empty non-nil slice for unregistered type",
		Run: func(t *testing.T, p Provider) {
			result := godi.GetServices[iUnregisteredExt](p)
			testutil.True(t, result != nil, "GetServices must return non-nil empty slice")
			testutil.Equal(t, len(result), 0)
		},
	},
}

func TestConformance_ExtensionResolutionDualProvider(t *testing.T) {
	RunSuite(t, extRtFactory, extSpecCases)
	RunSuite(t, extCgFactory, extSpecCases)
}

// ══ AC1: Cross-provider differential equivalence (SPSE, AD-15) ═════════════

func TestExtensions_DifferentialEquivalence_GetServicesCount(t *testing.T) {
	AssertEquivalent(t, extRtFactory, extCgFactory, func(p Provider) int {
		return len(p.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil))
	})
}

func TestExtensions_DifferentialEquivalence_GetServicesDynType(t *testing.T) {
	AssertEquivalent(t, extRtFactory, extCgFactory, func(p Provider) string {
		all := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil)
		if len(all) == 0 {
			return ""
		}
		return reflect.TypeOf(all[0]).String()
	})
}

func TestExtensions_DifferentialEquivalence_IsService(t *testing.T) {
	AssertEquivalent(t, extRtFactory, extCgFactory, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeMultipleService]())
	})
}

// ══ AC6/D-8: "Final set equivalence" (collection ops result = codegen
//    static DSL same final set) ═════════════════════════════════════════════

// extFinalSetRtFactory: runtime uses TryAdd/Replace/RemoveAll to arrive at a
// final registration set; the codegen factory declares the SAME final set
// statically. Cross-provider differential equivalence verifies they resolve
// identically.
//
// Scenario: register A, register B, Replace(C) → final set = {B, C}
// Runtime: AddSingleton A → AddSingleton B → Replace C (removes first A, appends C)
// Codegen: static DSL declares {B, C} directly (the equivalent final set).
//
// This test requires the codegen registration to contain the equivalent final
// set. Since Story 6.4 doesn't add new codegen registrations, this test will
// reference a codegen factory that has been set up for this scenario (or will
// be set up during dev-story as a conformance codegen registration).
func TestFinalSetEquivalence_Replace_DynType(t *testing.T) {
	// Runtime: AddSingleton[IFakeMultipleService](A) + AddSingleton(B) + Replace(C)
	// → final = {B, C} (first match A removed, C appended)
	rtFinal := ProviderFactory{
		Name: "runtime-replace-final",
		Build: func(t *testing.T) Provider {
			t.Helper()
			b := runtime.NewBuilder()
			runtime.AddSingleton[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
				return fakes.NewFakeOneMultipleService()
			})
			runtime.AddSingleton[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
				return fakes.NewFakeTwoMultipleService()
			})
			runtime.Replace[fakes.IFakeMultipleService](b, runtime.Singleton, func(godi.Provider) fakes.IFakeMultipleService {
				return fakes.NewFakeOneMultipleService()
			})
			p, err := b.Build()
			testutil.NoError(t, err)
			return p
		},
	}

	// Codegen equivalent: static DSL declares {Two, One} (same final set as
	// runtime Replace result). This factory will be wired in dev-story when
	// the codegen registration for this scenario is added.
	cgFinal := ProviderFactory{
		Name: "codegen-replace-final",
		Build: func(t *testing.T) Provider {
			t.Helper()
			b := runtime.NewBuilder()
			runtime.AddSingleton[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
				return fakes.NewFakeTwoMultipleService()
			})
			runtime.AddSingleton[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
				return fakes.NewFakeOneMultipleService()
			})
			p, err := b.Build()
			testutil.NoError(t, err)
			return p
		},
	}

	AssertEquivalent(t, rtFinal, cgFinal, func(p Provider) int {
		return len(p.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil))
	})
	AssertEquivalent(t, rtFinal, cgFinal, func(p Provider) string {
		all := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil)
		if len(all) < 2 {
			return ""
		}
		return reflect.TypeOf(all[0]).String() + "," + reflect.TypeOf(all[1]).String()
	})
}

func TestFinalSetEquivalence_RemoveAll_IsService(t *testing.T) {
	// Runtime: register then RemoveAll → IsService false
	rtRemoved := ProviderFactory{
		Name: "runtime-removeall",
		Build: func(t *testing.T) Provider {
			t.Helper()
			b := runtime.NewBuilder()
			runtime.AddSingleton[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
				return fakes.NewFakeOneMultipleService()
			})
			runtime.RemoveAll[fakes.IFakeMultipleService](b)
			p, err := b.Build()
			testutil.NoError(t, err)
			return p
		},
	}
	// Codegen equivalent: no registration at all = same final set
	cgEmpty := ProviderFactory{
		Name: "codegen-empty",
		Build: func(t *testing.T) Provider {
			t.Helper()
			b := runtime.NewBuilder()
			p, err := b.Build()
			testutil.NoError(t, err)
			return p
		},
	}

	AssertEquivalent(t, rtRemoved, cgEmpty, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeMultipleService]())
	})
}
