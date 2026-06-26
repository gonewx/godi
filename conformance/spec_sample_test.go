package conformance

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// --- in-test stub Provider --------------------------------------------------
//
// stubProvider is a throwaway, map-backed Provider used ONLY to prove the
// harness compiles and runs green before M1 delivers the real runtime/codegen
// providers. It is deliberately trivial: it does not construct dependencies,
// manage lifetimes, or track disposables. M1 deletes this and wires the two
// real ProviderFactory values from a single shared registration source (AD-14).

type keyedKey struct {
	t   reflect.Type
	key any
}

type stubProvider struct {
	services map[reflect.Type]any
	keyed    map[keyedKey][]any
}

func newStub(services map[reflect.Type]any) *stubProvider {
	return &stubProvider{services: services, keyed: map[keyedKey][]any{}}
}

func (p *stubProvider) GetService(t reflect.Type) any { return p.services[t] }

func (p *stubProvider) IsService(t reflect.Type) bool { _, ok := p.services[t]; return ok }

func (p *stubProvider) GetKeyedService(t reflect.Type, key any) any {
	if key == nil {
		return p.GetService(t) // AD-17: nil key routes to the unkeyed service
	}
	if vs := p.keyed[keyedKey{t, key}]; len(vs) > 0 {
		return vs[0]
	}
	return nil
}

func (p *stubProvider) GetKeyedServices(t reflect.Type, key any) []any {
	return p.keyed[keyedKey{t, key}]
}

func (p *stubProvider) CreateScope() Scope { return p } // stub: no real scoping yet

func (p *stubProvider) Close() error { return nil }

// sampleRegistration is the single shared registration source both factories
// build from. In M1 the two factories diverge into the runtime provider and the
// codegen provider — but still from one registration, never hand-forked.
func sampleRegistration(t *testing.T) Provider {
	t.Helper()
	return newStub(map[reflect.Type]any{
		reflect.TypeFor[string](): "hello",
		reflect.TypeFor[int]():    42,
	})
}

var (
	runtimeFactory = ProviderFactory{Name: "runtime", Build: sampleRegistration}
	codegenFactory = ProviderFactory{Name: "codegen", Build: sampleRegistration}
)

// sampleCases stands in for the 116 ported spec cases. Each touches only the
// Provider contract so both providers are held to identical behavior.
var sampleCases = []SpecCase{
	{
		Name: "resolves a registered service",
		Run: func(t *testing.T, p Provider) {
			got, ok := p.GetService(reflect.TypeFor[string]()).(string)
			testutil.True(t, ok, "string service should be present")
			testutil.Equal(t, got, "hello")
		},
	},
	{
		Name: "unregistered service resolves to nil (AD-16)",
		Run: func(t *testing.T, p Provider) {
			testutil.Nil(t, p.GetService(reflect.TypeFor[int64]()))
			testutil.True(t, !p.IsService(reflect.TypeFor[int64]()), "int64 must not be a service")
		},
	},
	{
		Name: "nil keyed key routes to the unkeyed service (AD-17)",
		Run: func(t *testing.T, p Provider) {
			got, _ := p.GetKeyedService(reflect.TypeFor[int](), nil).(int)
			testutil.Equal(t, got, 42)
		},
	},
}

func TestSpec_RuntimeProvider(t *testing.T) { RunSuite(t, runtimeFactory, sampleCases) }

func TestSpec_CodegenProvider(t *testing.T) { RunSuite(t, codegenFactory, sampleCases) }

// TestSpec_DifferentialEquivalence demonstrates AD-15: both providers, built
// from the same registration, must yield identical observable results.
func TestSpec_DifferentialEquivalence(t *testing.T) {
	AssertEquivalent(t, runtimeFactory, codegenFactory, func(p Provider) string {
		s, _ := p.GetService(reflect.TypeFor[string]()).(string)
		return s
	})
	AssertEquivalent(t, runtimeFactory, codegenFactory, func(p Provider) int {
		n, _ := p.GetService(reflect.TypeFor[int]()).(int)
		return n
	})
}
