// Conformance specification suite: dual-provider spec cases + cross-provider
// differential equivalence (AD-14 / AD-15). Both factories build from the same
// logical registration source; divergence in observable behavior is a regression
// in the architecture's equivalence guarantee.
//
// Story 1.5 wired the single-service slice; Story 2.1 grows it to a dependency
// graph — a FakeConsumer singleton depending on an IFakeService singleton — built
// by both providers from one registration (runtime form-B closures resolving the
// dependency through the provider; codegen the real //go:generate container). The
// runtime Build and codegen NewRoot now both return (provider, error) (D-1).
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
			runtime.AddSingleton[fakes.IFakeConsumer](b, func(p godi.Provider) fakes.IFakeConsumer {
				dep := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
				return fakes.NewFakeConsumer(dep)
			})
			runtime.AddTransient[fakes.IFakeOpenGenericService[fakes.IFakeService]](b, func(p godi.Provider) fakes.IFakeOpenGenericService[fakes.IFakeService] {
				// Story 4.1: the open-generic ctor now injects `value T`; resolve the
				// registered IFakeService singleton and pass it (mirrors codegen's
				// form-A dependency wiring — AD-14 same logical closed set).
				dep := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
				return fakes.NewFakeOpenGenericService[fakes.IFakeService](dep)
			})
			p, err := b.Build()
			testutil.NoError(t, err)
			return p
		},
	}
	cgFactory = ProviderFactory{
		Name: "codegen",
		Build: func(t *testing.T) Provider {
			t.Helper()
			root, err := codegen.NewRoot()
			testutil.NoError(t, err)
			return root
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
		// Story 2.1 dependency slice: the consumer resolves and its New{Impl}
		// constructor was wired with the injected IFakeService dependency.
		Name: "resolves IFakeConsumer with injected IFakeService dependency",
		Run: func(t *testing.T, p Provider) {
			got := p.GetService(reflect.TypeFor[fakes.IFakeConsumer]())
			testutil.NotNil(t, got)
			testutil.Equal(t, reflect.TypeOf(got).String(), "*fakes.FakeConsumer")
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

// --- 2.1-SPEC-001/002 (AC1/AC3, P0) ------------------------------------------

func TestConformance_RuntimeProvider(t *testing.T) { RunSuite(t, rtFactory, specCases) }

func TestConformance_CodegenProvider(t *testing.T) { RunSuite(t, cgFactory, specCases) }

// --- 2.1-SPEC-003 (AC3, P0) --------------------------------------------------

// TestConformance_DifferentialEquivalence: AD-15 cross-provider differential
// equivalence. Probes use provider-stable observables (IsService bool, dynamic
// type name) — never pointer identity (two providers build separate instances, R4).
func TestConformance_DifferentialEquivalence(t *testing.T) {
	// Single-service observables (Story 1.5).
	AssertEquivalent(t, rtFactory, cgFactory, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeService]())
	})
	AssertEquivalent(t, rtFactory, cgFactory, func(p Provider) string {
		return dynTypeName(p.GetService(reflect.TypeFor[fakes.IFakeService]()))
	})

	// Dependency-graph observables (Story 2.1): the injected consumer resolves
	// equivalently across both providers.
	AssertEquivalent(t, rtFactory, cgFactory, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeConsumer]())
	})
	AssertEquivalent(t, rtFactory, cgFactory, func(p Provider) string {
		return dynTypeName(p.GetService(reflect.TypeFor[fakes.IFakeConsumer]()))
	})

	AssertEquivalent(t, rtFactory, cgFactory, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[int]())
	})
}

// --- 2.1-SPEC-004 (AC3, P1) --------------------------------------------------

// TestConformance_SingletonSharedNotReconstructed: each service has exactly one
// constructor on the assembly path (AD-8), so the dependency injected into the
// consumer is the SAME singleton instance the provider resolves directly. Pointer
// identity is compared WITHIN a single provider (legal — R4 only forbids it
// ACROSS providers).
func TestConformance_SingletonSharedNotReconstructed(t *testing.T) {
	for _, f := range []ProviderFactory{rtFactory, cgFactory} {
		p := f.Build(t)
		t.Cleanup(func() { _ = p.Close() })

		var direct any = p.GetService(reflect.TypeFor[fakes.IFakeService]())
		var injected any = p.GetService(reflect.TypeFor[fakes.IFakeConsumer]()).(fakes.IFakeConsumer).Dependency()

		testutil.NotNil(t, direct)
		testutil.Equal(t, direct, injected) // same singleton instance: ctor ran once
	}
}

// dynTypeName is the dynamic type name of v, or "<nil>" — a provider-stable
// observable for differential equivalence (never pointer identity, R4).
func dynTypeName(v any) string {
	if v == nil {
		return "<nil>"
	}
	return reflect.TypeOf(v).String()
}
