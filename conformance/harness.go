// Package conformance hosts the ported .NET DependencyInjection specification
// suite (116 Spec cases + 52 Fakes) run against every godi Provider
// implementation, plus the cross-provider differential-equivalence harness.
//
// Design (AD-14 / AD-15 / AD-16): the suite is written once against the Provider
// contract and executed against BOTH providers from the same registration
// source — the reflect-free runtime provider (the semantic oracle) and the
// codegen provider — asserting identical observable results.
//
// SCAFFOLD STATUS — M1 fills this in. The real godi.Provider, the runtime
// provider, and the codegen provider do not exist yet. This file defines the
// harness *contract* and *runner*; spec_sample_test.go proves the pattern runs
// green against an in-test stub. In M1:
//   - replace the local Provider/Scope interfaces with type aliases to
//     godi.Provider / godi.Scope once that package lands;
//   - register the two real ProviderFactory values (runtime, codegen) built from
//     a single shared registration source;
//   - port the 116 cases into spec_test.go and feed them to RunSuite.
package conformance

import (
	"reflect"
	"testing"
)

// Provider is the harness view of the shared runtime contract pinned by AD-16.
// Its method set and unmet-lookup semantics mirror dotnet's IServiceProvider
// family; M1 turns this into a type alias for godi.Provider.
type Provider interface {
	// GetService returns the service registered for t, or nil if none
	// (AD-16: unregistered → nil, never panic).
	GetService(t reflect.Type) any
	// IsService reports whether t can be resolved (CAP-6).
	IsService(t reflect.Type) bool
	// GetKeyedService returns the service registered for (t, key); a nil key
	// routes to the unkeyed service (AD-17).
	GetKeyedService(t reflect.Type, key any) any
	// GetKeyedServices returns all services registered for (t, key) in
	// registration order (CAP-4 / CAP-5).
	GetKeyedServices(t reflect.Type, key any) []any
	// CreateScope opens a child resolution scope (AD-1 / AD-3).
	CreateScope() Scope
	// Close releases this provider/scope; it is idempotent (AD-16).
	Close() error
}

// Scope is a resolution scope. It exposes the same resolution surface as its
// parent Provider; M1 aligns it with godi.Scope.
type Scope interface {
	Provider
}

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
// the heart of AD-15: probe runs against providers built by two factories for
// the same logical registration, and the two results must be equal — the
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
