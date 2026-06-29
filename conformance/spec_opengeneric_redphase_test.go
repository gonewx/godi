// ATDD scaffold for Story 4.1 (AC1-AC6, Task 5) — conformance suite:
// open-generic CLOSED materialization across BOTH providers (runtime oracle +
// codegen B2) with dependency injection, multi-instantiation disambiguation,
// the unmaterialized-is-unresolvable tradeoff, and cross-provider differential
// equivalence (AD-14 / AD-15 / AD-4 / AD-10).
//
// Ported from dotnet DependencyInjectionSpecificationTests (v10.0.9):
//
//	OpenGenericServicesCanBeResolved (:648, touch-point 1/9) → AC2 (4.1-SPEC-001)
//	(CAP-7 reverse: unmaterialized closed type)              → AC3 (4.1-SPEC-002)
//	(serviceName disambiguation, deferred-work.md:57)        → AC4 (4.1-SPEC-003)
//	(Go difference annotation: no MakeGenericType)           → AC5 (4.1-SPEC-004)
//	(AD-15 cross-provider differential equivalence)          → AC6 (4.1-SPEC-DIFF)
//
// Go difference (AD-4, the only hard wall of Epic 4): Go has no runtime generic
// instantiation (no MakeGenericType equivalent). dotnet registers the OPEN
// generic typeof(IFoo<>) and materializes ANY requested closed type at runtime;
// godi requires closed types to be ENUMERATED in source (per-service helper) and
// the generator materializes them statically — unenumerated closed types are
// unresolvable at build time (AC3). This is a deliberate tradeoff (CAP-7), not a
// defect.
//
// ACTIVATED (Story 4.1): the build tag was removed once Task 1 upgraded
// FakeOpenGenericService[T] to inject `value T` and Task 3/4 taught the generator
// the per-service helper + serviceName disambiguation and regenerated the codegen
// container with the second instantiation — this suite is now GREEN.
//
// Test standard R4: pointer identity within ONE provider only; across providers
// use provider-stable observables (IsService booleans, dynamic type names).
// dynTypeName lives in spec_test.go (same package).
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

// addOpenGenericRT is the runtime-provider counterpart of the codegen per-service
// helper (AD-14: both providers enumerate the SAME logical closed set, never
// hand-forked). Go instantiates the closed generic at COMPILE time, so the
// runtime provider needs no new mechanism — the closed type is just another
// reflect.Type key and the dependency is resolved through the provider and
// injected into the New{Impl} constructor (mirroring dotnet's
// FakeOpenGenericService<TVal>(TVal value)).
//
// Story 4.1: NewFakeOpenGenericService[T any](value T) injects the resolved
// dependency, mirroring the codegen form-A wiring on the same logical closed set.
func addOpenGenericRT[T any](b *runtime.Builder) {
	runtime.AddTransient[fakes.IFakeOpenGenericService[T]](b, func(p godi.Provider) fakes.IFakeOpenGenericService[T] {
		dep := p.GetService(reflect.TypeFor[T]()).(T)
		return fakes.NewFakeOpenGenericService[T](dep)
	})
}

// rtFactoryOpenGeneric builds the runtime oracle: a singleton IFakeService, a
// singleton IFakeConsumer (the T of the SECOND closed instantiation, proving
// disambiguation), and TWO closed instantiations of the open generic enumerated
// through the runtime helper — the same logical closed set the codegen provider
// materializes {IFakeService, IFakeConsumer} (AD-14).
var rtFactoryOpenGeneric = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		// Singletons that serve as the injected T values for the closed generics.
		runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
			return fakes.NewFakeService()
		})
		runtime.AddSingleton[fakes.IFakeConsumer](b, func(p godi.Provider) fakes.IFakeConsumer {
			dep := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
			return fakes.NewFakeConsumer(dep)
		})
		// Two closed instantiations of the SAME open generic — enumerated via the
		// runtime helper (Go compile-time instantiation). The injected value is
		// the singleton resolved for T (AC2), and the two instantiations coexist
		// without collision (AC4).
		addOpenGenericRT[fakes.IFakeService](b)
		addOpenGenericRT[fakes.IFakeConsumer](b)
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryOpenGeneric builds the codegen provider from the regenerated
// container_gen.go (Story 4.1): the committed container resolves BOTH closed
// instantiations [IFakeService] / [IFakeConsumer] with the injected dependency
// (AC2) and disambiguated fields/methods (AC4).
var cgFactoryOpenGeneric = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// Closed generic type aliases for readability in assertions below.
//   - openGenericOfService:  materialized (enumerated) — resolvable (AC2)
//   - openGenericOfConsumer: materialized (enumerated, second) — resolvable (AC4)
//   - openGenericOfScoped:   structurally valid but NOT enumerated — unresolvable (AC3)
func openGenericOfServiceType() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]()
}
func openGenericOfConsumerType() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeConsumer]]()
}
func openGenericOfScopedType() reflect.Type {
	// IFakeScopedService is a registered service, but the closed generic
	// IFakeOpenGenericService[IFakeScopedService] is NOT enumerated anywhere.
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeScopedService]]()
}

const (
	dynFakeOpenGenericOfService  = "*fakes.FakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IFakeService]"
	dynFakeOpenGenericOfConsumer = "*fakes.FakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IFakeConsumer]"
)

// openGenericSpecCases run against BOTH providers via RunSuite (AC6).
var openGenericSpecCases = []SpecCase{
	// --- 4.1-SPEC-001 (AC2, P0) — OpenGenericServicesCanBeResolved (1/9) -------
	{
		Name: "opengeneric: closed [IFakeService] resolves with injected singleton (.Value()==singleton) (OpenGenericServicesCanBeResolved, DISpec:648)",
		Run: func(t *testing.T, p Provider) {
			svc := p.GetService(openGenericOfServiceType())
			testutil.NotNil(t, svc)
			if svc == nil {
				return
			}
			testutil.Equal(t, reflect.TypeOf(svc).String(), dynFakeOpenGenericOfService)

			// Dependency injection (dotnet Assert.Same(singletonService, generic.Value)):
			// the closed instance's injected Value() is the SAME singleton the
			// provider resolves directly. Pointer identity WITHIN one provider is
			// legal (R4 forbids it only ACROSS providers).
			gen := svc.(fakes.IFakeOpenGenericService[fakes.IFakeService])
			injected := gen.Value()
			direct := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
			testutil.NotNil(t, injected)
			testutil.Equal(t, injected, direct) // same singleton: dependency injected (AD-8 one ctor)
		},
	},
	// --- 4.1-SPEC-002 (AC3, P0) — Unmaterialized closed type unresolvable ------
	{
		Name: "opengeneric: unmaterialized closed [IFakeScopedService] returns nil + IsService=false (CAP-7, no MakeGenericType)",
		Run: func(t *testing.T, p Provider) {
			// Go difference (AD-4): the closed type IFakeOpenGenericService[IFakeScopedService]
			// is a valid Go type but was NEVER enumerated, so there is no slot/factory
			// for it — unresolvable at build time. dotnet would materialize it from the
			// open registration via MakeGenericType; godi has no such runtime mechanism.
			testutil.Nil(t, p.GetService(openGenericOfScopedType()))
			testutil.True(t, !p.IsService(openGenericOfScopedType()),
				"unmaterialized closed generic must report IsService=false (CAP-7: no runtime MakeGenericType)")
		},
	},
	// --- 4.1-SPEC-003 (AC4, P0) — Multi-instantiation disambiguation -----------
	{
		Name: "opengeneric: two closed instantiations [IFakeService] + [IFakeConsumer] coexist, both resolve, no collision (serviceName disambiguation, deferred-work.md:57)",
		Run: func(t *testing.T, p Provider) {
			// Both closed instantiations of the SAME open generic must coexist —
			// disambiguated serviceName (type args folded into the name) prevents
			// the plan.Validate duplicate-name error and the generated f_<Name>
			// field collision.
			svcA := p.GetService(openGenericOfServiceType())
			svcB := p.GetService(openGenericOfConsumerType())
			testutil.NotNil(t, svcA)
			testutil.NotNil(t, svcB)
			if svcA == nil || svcB == nil {
				return
			}
			testutil.Equal(t, reflect.TypeOf(svcA).String(), dynFakeOpenGenericOfService)
			testutil.Equal(t, reflect.TypeOf(svcB).String(), dynFakeOpenGenericOfConsumer)

			// Independent: each carries its own injected dependency type.
			injB := svcB.(fakes.IFakeOpenGenericService[fakes.IFakeConsumer]).Value()
			testutil.NotNil(t, injB)
			testutil.Equal(t, reflect.TypeOf(injB).String(), "*fakes.FakeConsumer")
		},
	},
	// --- 4.1-SPEC-004 (AC5, P1) — Go difference: explicit enumeration vs MakeGenericType
	{
		Name: "opengeneric: Go adaptation — explicit closed enumeration resolves; unenumerated does not (no MakeGenericType, AD-4)",
		Run: func(t *testing.T, p Provider) {
			// godi's open-generic semantics are explicit-closed-enumeration, NOT
			// dotnet's open-registration + runtime MakeGenericType. This case pins
			// the difference as a single observable invariant: a type is resolvable
			// IFF it was enumerated in source.
			//
			// Enumerated → resolvable.
			testutil.True(t, p.IsService(openGenericOfServiceType()),
				"explicitly enumerated closed generic is a service")
			// Unenumerated (but structurally valid Go type) → NOT resolvable.
			testutil.True(t, !p.IsService(openGenericOfScopedType()),
				"Go has no MakeGenericType: unenumerated closed generic is NOT a service (AD-4 tradeoff, CAP-7)")
		},
	},
}

// --- 4.1-SPEC-RUN (AC2-AC5, P0) ---------------------------------------------

func TestConformanceOpenGeneric_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryOpenGeneric, openGenericSpecCases)
}

func TestConformanceOpenGeneric_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryOpenGeneric, openGenericSpecCases)
}

// --- 4.1-SPEC-DIFF (AC6, P0) — Cross-provider differential equivalence -------

// TestConformanceOpenGeneric_DifferentialEquivalence: AD-15 cross-provider
// differential equivalence for the open-generic-materialization axis (a behavior
// axis AD-15 explicitly names as a coverage obligation). Probes use
// provider-stable observables only — IsService booleans and dynamic type names —
// never pointer identity across providers (R4).
func TestConformanceOpenGeneric_DifferentialEquivalence(t *testing.T) {
	// IsService for the first materialized closed type (true both).
	AssertEquivalent(t, rtFactoryOpenGeneric, cgFactoryOpenGeneric, func(p Provider) bool {
		return p.IsService(openGenericOfServiceType())
	})
	// Dynamic type for the first materialized closed type.
	AssertEquivalent(t, rtFactoryOpenGeneric, cgFactoryOpenGeneric, func(p Provider) string {
		return dynTypeName(p.GetService(openGenericOfServiceType()))
	})
	// IsService for the second materialized closed type (disambiguation: true both).
	AssertEquivalent(t, rtFactoryOpenGeneric, cgFactoryOpenGeneric, func(p Provider) bool {
		return p.IsService(openGenericOfConsumerType())
	})
	// Dynamic type for the second materialized closed type.
	AssertEquivalent(t, rtFactoryOpenGeneric, cgFactoryOpenGeneric, func(p Provider) string {
		return dynTypeName(p.GetService(openGenericOfConsumerType()))
	})
	// IsService for the UNMATERIALIZED closed type (false both — AC3).
	AssertEquivalent(t, rtFactoryOpenGeneric, cgFactoryOpenGeneric, func(p Provider) bool {
		return p.IsService(openGenericOfScopedType())
	})
	// GetService for the UNMATERIALIZED closed type ("<nil>" both — AC3).
	AssertEquivalent(t, rtFactoryOpenGeneric, cgFactoryOpenGeneric, func(p Provider) string {
		return dynTypeName(p.GetService(openGenericOfScopedType()))
	})
	// Injected dependency's dynamic type for the first closed type — the
	// provider-stable witness that BOTH providers inject the SAME dependency type
	// (the value identity itself is per-provider, so we observe the type name, R4).
	AssertEquivalent(t, rtFactoryOpenGeneric, cgFactoryOpenGeneric, func(p Provider) string {
		svc := p.GetService(openGenericOfServiceType())
		if svc == nil {
			return "<nil>"
		}
		return dynTypeName(svc.(fakes.IFakeOpenGenericService[fakes.IFakeService]).Value())
	})
}
