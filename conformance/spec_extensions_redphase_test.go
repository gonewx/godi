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

// extFinalSetRtFactory: runtime uses Replace/RemoveAll to arrive at a final
// registration set; the codegen leg is the REAL generated container
// (codegen.NewRoot()), which declares the same logical final set directly
// through the static DSL (conformance/codegen/registration.go). Cross-provider
// differential equivalence verifies they resolve identically — this is the
// literal AD-15/D-8 obligation, not a runtime-vs-runtime stand-in (code review
// of story-6.4, 2026-07-01: the prior version compared two runtime.NewBuilder
// legs and never touched codegen at all).
//
// Scenario: register One, register Two, Replace(One) → final set = [Two, One]
// (first ServiceType match — One — removed, new One appended at the end).
// Dedicated fakes (fakes.IFakeReplaceFinalService) keep this isolated from the
// existing IFakeMultipleService count/order assertions elsewhere in this file.
func TestFinalSetEquivalence_Replace_DynType(t *testing.T) {
	rtFinal := ProviderFactory{
		Name: "runtime-replace-final",
		Build: func(t *testing.T) Provider {
			t.Helper()
			b := runtime.NewBuilder()
			runtime.AddSingleton[fakes.IFakeReplaceFinalService](b, func(godi.Provider) fakes.IFakeReplaceFinalService {
				return fakes.NewFakeReplaceFinalOne()
			})
			runtime.AddSingleton[fakes.IFakeReplaceFinalService](b, func(godi.Provider) fakes.IFakeReplaceFinalService {
				return fakes.NewFakeReplaceFinalTwo()
			})
			runtime.Replace[fakes.IFakeReplaceFinalService](b, runtime.Singleton, func(godi.Provider) fakes.IFakeReplaceFinalService {
				return fakes.NewFakeReplaceFinalOne()
			})
			p, err := b.Build()
			testutil.NoError(t, err)
			return p
		},
	}

	// Codegen equivalent: the REAL generated container. registration.go
	// declares fakes.IFakeReplaceFinalService{Two, One} directly, in that
	// order — the same final set the runtime leg reaches via mutation.
	cgFinal := ProviderFactory{
		Name: "codegen-replace-final",
		Build: func(t *testing.T) Provider {
			t.Helper()
			root, err := codegen.NewRoot()
			testutil.NoError(t, err)
			return root
		},
	}

	AssertEquivalent(t, rtFinal, cgFinal, func(p Provider) int {
		return len(p.GetKeyedServices(reflect.TypeFor[fakes.IFakeReplaceFinalService](), nil))
	})
	AssertEquivalent(t, rtFinal, cgFinal, func(p Provider) string {
		all := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeReplaceFinalService](), nil)
		if len(all) < 2 {
			return ""
		}
		return reflect.TypeOf(all[0]).String() + "," + reflect.TypeOf(all[1]).String()
	})
}

// TestFinalSetEquivalence_RemoveAll_IsService: runtime registers
// fakes.IFakeRemovedService then RemoveAll's it -> final set = empty ->
// IsService false. The codegen leg is the REAL generated container; this type
// is deliberately registered NOWHERE in registration.go, so its generated
// IsService is naturally false — the same final state, without codegen having
// to declare an "absence" (code review of story-6.4, 2026-07-01).
func TestFinalSetEquivalence_RemoveAll_IsService(t *testing.T) {
	rtRemoved := ProviderFactory{
		Name: "runtime-removeall",
		Build: func(t *testing.T) Provider {
			t.Helper()
			b := runtime.NewBuilder()
			runtime.AddSingleton[fakes.IFakeRemovedService](b, func(godi.Provider) fakes.IFakeRemovedService {
				return fakes.NewFakeRemovedService()
			})
			runtime.RemoveAll[fakes.IFakeRemovedService](b)
			p, err := b.Build()
			testutil.NoError(t, err)
			return p
		},
	}
	cgAbsent := ProviderFactory{
		Name: "codegen-never-registered",
		Build: func(t *testing.T) Provider {
			t.Helper()
			root, err := codegen.NewRoot()
			testutil.NoError(t, err)
			return root
		},
	}

	AssertEquivalent(t, rtRemoved, cgAbsent, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeRemovedService]())
	})
}

// TestFinalSetEquivalence_InstanceRegistration_DynType is the D-8 "instance
// registration result obligation" axis: runtime uses AddSingletonInstance with
// a pre-constructed value; codegen has no raw-instance DSL entry (D-4), so it
// declares the equivalent via a form-B factory (registration.go). Equivalence
// is asserted on the observable dynamic type, never on cross-provider pointer
// identity (R4) — previously this obligation axis had no differential
// equivalence test at all (code review of story-6.4, 2026-07-01).
func TestFinalSetEquivalence_InstanceRegistration_DynType(t *testing.T) {
	rtInstance := ProviderFactory{
		Name: "runtime-instance",
		Build: func(t *testing.T) Provider {
			t.Helper()
			b := runtime.NewBuilder()
			runtime.AddSingletonInstance[fakes.IFakeInstanceService](b, fakes.NewFakeInstanceService())
			p, err := b.Build()
			testutil.NoError(t, err)
			return p
		},
	}
	cgInstance := ProviderFactory{
		Name: "codegen-instance-equivalent",
		Build: func(t *testing.T) Provider {
			t.Helper()
			root, err := codegen.NewRoot()
			testutil.NoError(t, err)
			return root
		},
	}

	AssertEquivalent(t, rtInstance, cgInstance, func(p Provider) string {
		v, ok := godi.Get[fakes.IFakeInstanceService](p)
		if !ok {
			return ""
		}
		return reflect.TypeOf(v).String()
	})
	AssertEquivalent(t, rtInstance, cgInstance, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeInstanceService]())
	})
}
