//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.1 (AC1/AC3) — dual-provider dependency
// slice + cross-provider differential equivalence (Task 5). This is the
// end-to-end acceptance probe: the same logical registration (a Consumer
// singleton depending on an IFakeService singleton) is built by BOTH providers
// and must be observably equivalent (AD-14/AD-15).
//
// RED: it depends on (a) the new dependency fixture fakes.NewFakeConsumer,
// (b) the new Build() (godi.Provider, error) signature, and (c) the new
// codegen.NewRoot() (*Root, error) signature — none exist yet → compilation
// fails under `-tags atdd_redphase`. Activation: port the deps fixture, evolve
// both providers, regenerate the codegen container, then delete the //go:build
// line and the blank line after it (and fold these cases into spec_test.go).
//
// Distinct names (rtFactoryDeps / cgFactoryDeps / specCasesDeps /
// TestConformanceDeps_*) avoid clashing with spec_test.go under `-tags
// atdd_redphase`, where both files compile into package conformance together.
//
// API CONTRACT THIS SCAFFOLD ASSUMES:
//
//	fakes.IFakeConsumer  — sealed interface, impl *fakes.FakeConsumer
//	fakes.NewFakeConsumer(fakes.IFakeService) *fakes.FakeConsumer  // form-A New{Impl}
//	fakes.IFakeConsumer.Dependency() fakes.IFakeService            // exposes injected dep
//	runtime.(*Builder).Build() (godi.Provider, error)
//	codegen.NewRoot() (*Root, error)
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

// rtFactoryDeps builds the runtime oracle from form-B closures: IFakeService and
// an IFakeConsumer whose closure resolves its dependency through the provider.
var rtFactoryDeps = ProviderFactory{
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
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryDeps builds the codegen provider from the regenerated dependency
// container (real //go:generate output for the same registration source).
var cgFactoryDeps = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

var specCasesDeps = []SpecCase{
	{
		Name: "resolves IFakeConsumer with injected IFakeService dependency",
		Run: func(t *testing.T, p Provider) {
			got := p.GetService(reflect.TypeFor[fakes.IFakeConsumer]())
			testutil.NotNil(t, got)
			testutil.Equal(t, reflect.TypeOf(got).String(), "*fakes.FakeConsumer")
		},
	},
}

// --- 2.1-SPEC-001/002 (AC1/AC3, P0) ------------------------------------------

func TestConformanceDeps_RuntimeProvider(t *testing.T) { RunSuite(t, rtFactoryDeps, specCasesDeps) }

func TestConformanceDeps_CodegenProvider(t *testing.T) { RunSuite(t, cgFactoryDeps, specCasesDeps) }

// --- 2.1-SPEC-003 (AC3, P0) --------------------------------------------------

// TestConformanceDeps_DifferentialEquivalence: the two providers built from the
// same dependency registration must agree on provider-stable observables
// (IsService bool, dynamic type name) — never pointer identity (R4: separate
// instances per provider).
func TestConformanceDeps_DifferentialEquivalence(t *testing.T) {
	AssertEquivalent(t, rtFactoryDeps, cgFactoryDeps, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeConsumer]())
	})

	AssertEquivalent(t, rtFactoryDeps, cgFactoryDeps, func(p Provider) string {
		v := p.GetService(reflect.TypeFor[fakes.IFakeConsumer]())
		if v == nil {
			return "<nil>"
		}
		return reflect.TypeOf(v).String()
	})
}

// --- 2.1-SPEC-004 (AC3, P1) --------------------------------------------------

// TestConformanceDeps_SingletonSharedNotReconstructed: each service has exactly
// one constructor on the assembly path (AD-8), so the dependency injected into
// the consumer is the SAME singleton instance the provider resolves directly.
// Pointer identity is compared WITHIN a single provider (legal — R4 only forbids
// it ACROSS providers).
func TestConformanceDeps_SingletonSharedNotReconstructed(t *testing.T) {
	for _, f := range []ProviderFactory{rtFactoryDeps, cgFactoryDeps} {
		p := f.Build(t)
		t.Cleanup(func() { _ = p.Close() })

		var direct any = p.GetService(reflect.TypeFor[fakes.IFakeService]())
		var injected any = p.GetService(reflect.TypeFor[fakes.IFakeConsumer]()).(fakes.IFakeConsumer).Dependency()

		testutil.NotNil(t, direct)
		testutil.Equal(t, direct, injected) // same singleton instance: ctor ran once
	}
}
