//go:build atdd_redphase

// ATDD scaffold for Story 1.5 — conformance suite: dual-provider single-service
// spec cases + cross-provider differential equivalence (AD-14 / AD-15).
//
// This file replaces the throwaway stub-based spec_sample_test.go once activated.
// Red cause: fakes.IFakeService / fakes.NewFakeService / codegen.NewRoot are
// undefined (packages are doc.go placeholders) → compilation failure.
//
// Activation (TDD red → green):
//  1. Implement Tasks 1–4 (fakes, emit evolution, codegen container, wiring).
//  2. Delete spec_sample_test.go (stub code) and contract_alias_test.go (stub proofs).
//  3. Delete the //go:build line above and the blank line after it.
//  4. Rename this file to spec_test.go.
//  5. GOWORK=off go test -race ./conformance/... → green.
//
// See _bmad-output/test-artifacts/atdd-checklist-1-5-conformance-dual-provider.md.
//
// API CONTRACT THIS SCAFFOLD ASSUMES:
//
//	runtime.NewBuilder() *Builder
//	runtime.AddSingleton[T](b, func(godi.Provider) T) *Builder
//	(*Builder).Build() godi.Provider
//	codegen.NewRoot() *Root   (implements godi.Provider)
//	fakes.IFakeService        (interface with seal method)
//	fakes.NewFakeService() *FakeService
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
