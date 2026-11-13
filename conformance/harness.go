// Package conformance hosts the ported .NET DependencyInjection specification
// suite (116 Spec cases + 52 Fakes) run against every godi Provider
// implementation, plus the cross-provider differential-equivalence harness.
//
// Design (AD-14 / AD-15 / AD-16): the suite is written once against the Provider
// contract and executed against BOTH providers, built from equivalent
// registrations — the reflect-free runtime provider (the semantic oracle) and
// the codegen provider — asserting identical observable results.
//
// spec_test.go defines the two ProviderFactory values: the runtime factory
// registers hand-written runtime.Add* closures, and the codegen factory builds
// the container generated from the godi.Add* markers in conformance/codegen.
// The two registration sets are kept equivalent by hand (AD-14). The ported
// spec cases run against both through RunSuite / AssertEquivalent.
package conformance

import (
	"testing"

	"github.com/gonewx/godi"
)

// Provider is the harness alias for the single shared runtime contract pinned by
// AD-16. It was a local interface during the scaffold phase; Story 1.2 collapsed
// it onto godi.Provider so the conformance suite references one source of truth
// rather than a parallel copy (AD-14 / AD-16).
type Provider = godi.Provider

// Scope is the harness alias for godi.Scope, a resolution scope exposing the same
// resolution surface as its parent Provider (AD-16).
type Scope = godi.Scope

// ProviderFactory builds a fresh, fully-configured Provider for one spec case.
// It is the Go equivalent of dotnet's `abstract CreateServiceProvider`: each
// implementation (runtime oracle, codegen) supplies one factory and the same
// suite runs against all of them. Name labels the provider in subtest output.
type ProviderFactory struct {
	Name  string
	Build func(t *testing.T) Provider
}

// SpecCase is one specification case. The same case is replayed against every
// ProviderFactory; it must touch only the Provider contract so all providers
// are held to identical behavior.
type SpecCase struct {
	Name string
	Run  func(t *testing.T, p Provider)
}

// RunSuite runs every case against the given factory as nested subtests
// (factory.Name/case.Name), closing each built provider via t.Cleanup. M1 calls
// this once per provider with the full 116-case slice.
func RunSuite(t *testing.T, factory ProviderFactory, cases []SpecCase) {
	t.Helper()
	t.Run(factory.Name, func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.Name, func(t *testing.T) {
				p := factory.Build(t)
				t.Cleanup(func() { _ = p.Close() })
				c.Run(t, p)
			})
		}
	})
}

// AssertEquivalent is the cross-provider differential-equivalence assertion at
// the heart of AD-15: probe runs against providers built by two factories from
// equivalent registrations, and the two results must be equal — the
// codegen provider must match the runtime oracle.
func AssertEquivalent[T comparable](t *testing.T, a, b ProviderFactory, probe func(Provider) T) {
	t.Helper()
	pa := a.Build(t)
	t.Cleanup(func() { _ = pa.Close() })
	pb := b.Build(t)
	t.Cleanup(func() { _ = pb.Close() })

	got, want := probe(pa), probe(pb)
	if got != want {
		t.Errorf("provider differential mismatch on identical registration:\n  %s: %v\n  %s: %v",
			a.Name, got, b.Name, want)
	}
}
