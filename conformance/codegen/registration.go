package codegen

import (
	"reflect"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/fakes"
)

// container_gen.go is produced from the form-A registration below by the real
// godi-gen static-analysis CLI (Story 2.1). Running `go generate ./...` from the
// module root re-emits it; TestCodegen_RegenComparison guards that the committed
// file stays byte-identical to a fresh run (AD-14: deterministic output, no
// hand-fork). Run under GOWORK=off so the generator analyzes this package in
// isolation from the auth workspace's sibling modules.
//
//go:generate go run github.com/gonewx/godi/cmd/godi-gen -pkg codegen -out container_gen.go .

// register is the static registration source the generator analyzes (AD-7): the
// AddSingleton markers are read from go/types, never executed. It mirrors the
// runtime oracle's registration (spec_test.go's rtFactory) so both providers are
// built from one logical source (AD-14). Form A: FakeService has no dependency;
// FakeConsumer depends on IFakeService, discovered through NewFakeConsumer's
// signature.
func register() {
	var r godi.Registry
	godi.AddSingleton[fakes.IFakeService, fakes.FakeService](&r)
	godi.AddSingleton[fakes.IFakeConsumer, fakes.FakeConsumer](&r)
	godi.AddScoped[fakes.IFakeScopedService, fakes.FakeScopedService](&r)
	godi.AddTransient[fakes.IFakeTransientService, fakes.FakeTransientService](&r)
	godi.AddTransient[fakes.IFakeTransientWithScopedDep, fakes.FakeTransientWithScopedDep](&r)
	godi.AddSingleton[fakes.IFakeDisposableSingleton, fakes.FakeDisposableSingleton](&r)
	godi.AddScoped[fakes.IFakeDisposableScoped, fakes.FakeDisposableScoped](&r)
	godi.AddTransient[fakes.IFakeDisposableTransient, fakes.FakeDisposableTransient](&r)
	godi.AddTransient[fakes.IFakeMultipleService, fakes.FakeOneMultipleService](&r)
	godi.AddTransient[fakes.IFakeMultipleService, fakes.FakeTwoMultipleService](&r)
	godi.AddTransient[fakes.IFakeOuterService, fakes.FakeOuterService](&r)
	// Story 4.1 (AC1, Mechanism A): the open generic is enumerated through the
	// per-service helper AddFakeOpenGenericService[T] (defined below) — the
	// generator follows the forwarder, substitutes T, and materializes the closed
	// implementation, injecting the closed ctor's `value T` dependency (AC2). Two
	// instantiations [IFakeService] / [IFakeConsumer] prove serviceName
	// disambiguation lets the two coexist without collision (AC4).
	AddFakeOpenGenericService[fakes.IFakeService](&r)
	AddFakeOpenGenericService[fakes.IFakeConsumer](&r)

	// Story 4.2 (AC1/AC2): candidate service types as singletons — injected into
	// constrained open-generic implementations via value T dependency.
	godi.AddSingleton[fakes.IPocoClass, fakes.PocoClass](&r)
	godi.AddSingleton[fakes.IClassImplementingIterable, fakes.ClassImplementingIterable](&r)
	godi.AddSingleton[fakes.IClassInheritingAbstract, fakes.ClassInheritingAbstract](&r)

	// Story 4.2: unconstrained open-generic instantiations for each candidate type
	AddFakeOpenGenericService[fakes.IPocoClass](&r)
	AddFakeOpenGenericService[fakes.IClassImplementingIterable](&r)
	AddFakeOpenGenericService[fakes.IClassInheritingAbstract](&r)

	// Story 4.2 (AC2): constrained open-generic instantiations — only types
	// satisfying the Go type constraint can be passed. The generator is transparent
	// to constraints (AD-10); Mechanism A pipeline handles these identically.
	AddConstrainedOpenGenericService[fakes.IPocoClass](&r)
	AddInterfaceConstrainedService[fakes.IClassImplementingIterable](&r)
	AddAbstractConstrainedService[fakes.IClassInheritingAbstract](&r)

	// Keyed registrations (Story 3-1, AD-17)
	godi.AddKeyedSingleton[fakes.IKeyedService, fakes.KeyedServiceOne](&r, "service1")
	godi.AddKeyedSingleton[fakes.IKeyedService, fakes.KeyedServiceTwo](&r, "service2")
	godi.AddKeyedSingleton[fakes.IKeyedService, fakes.KeyedServiceOne](&r, "both")
	godi.AddKeyedSingleton[fakes.IKeyedService, fakes.KeyedServiceTwo](&r, "both")
	godi.AddKeyedSingleton[fakes.IKeyedService, fakes.KeyedServiceOne](&r, "singleton-factory")
	godi.AddKeyedTransient[fakes.IKeyedService, fakes.KeyedServiceOne](&r, "transient")
	godi.AddKeyedTransientFactory[fakes.IKeyedService](&r, "transient-factory", func(p godi.Provider) fakes.IKeyedService {
		return fakes.NewKeyedServiceOne()
	})
	godi.AddKeyedScoped[fakes.IKeyedService, fakes.KeyedServiceOne](&r, "scoped")

	// Story 3-2: Keyed injection registrations (AC2-AC5)
	godi.AddKeyedSingletonFactory[fakes.IKeyedServiceWithDep](&r, "with-dep", func(p godi.Provider) fakes.IKeyedServiceWithDep {
		dep := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "service1")
		if dep == nil {
			return fakes.NewKeyedServiceWithDep(nil)
		}
		return fakes.NewKeyedServiceWithDep(dep.(fakes.IKeyedService))
	})
	godi.AddKeyedSingletonFactory[fakes.IKeyedServiceWithKey](&r, "injected-key", func(p godi.Provider) fakes.IKeyedServiceWithKey {
		return fakes.NewKeyedServiceWithKey("injected-key")
	})
	godi.AddKeyedSingletonFactory[fakes.IKeyedServiceWithDep](&r, "missing-dep", func(p godi.Provider) fakes.IKeyedServiceWithDep {
		dep := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "nonexistent")
		if dep == nil {
			return fakes.NewKeyedServiceWithDep(nil)
		}
		return fakes.NewKeyedServiceWithDep(dep.(fakes.IKeyedService))
	})
	godi.AddKeyedSingletonFactory[fakes.IKeyedServiceWithProvider](&r, "with-provider", func(p godi.Provider) fakes.IKeyedServiceWithProvider {
		return fakes.NewKeyedServiceWithProvider(p)
	})
	godi.AddKeyedScopedFactory[fakes.IKeyedServiceWithProvider](&r, "scoped-with-provider", func(p godi.Provider) fakes.IKeyedServiceWithProvider {
		return fakes.NewKeyedServiceWithProvider(p)
	})

	// Story 3-3: AnyKey / null-key boundary registrations (AD-17). The unkeyed A is
	// the AnyKey-query EXCLUSION baseline and the nil-resolution target (AC1/AC2);
	// the interleaved keys alpha→A, beta→B, alpha→A prove GetKeyedServices(AnyKey)
	// returns registration order [A, B, A], NOT key-grouped order [A, A, B] (AC2,
	// ResolveKeyedServicesAnyKeyOrdering). The second alpha→A is a form-B factory so
	// it gets a name (IAnyKeyService#"alpha") distinct from the form-A AnyKeyServiceA#
	// "alpha", avoiding a generated field-name collision while keeping the same
	// observable type. No AnyKey is registered as a key — that is rejected at
	// generate time (AC4); the rejection is asserted by a generator unit test.
	godi.AddSingleton[fakes.IAnyKeyService, fakes.AnyKeyServiceA](&r)
	godi.AddKeyedSingleton[fakes.IAnyKeyService, fakes.AnyKeyServiceA](&r, "alpha")
	godi.AddKeyedSingleton[fakes.IAnyKeyService, fakes.AnyKeyServiceB](&r, "beta")
	godi.AddKeyedSingletonFactory[fakes.IAnyKeyService](&r, "alpha", func(p godi.Provider) fakes.IAnyKeyService {
		return fakes.NewAnyKeyServiceA()
	})

	// Story 3-4: closed-generic keyed registration for IsKeyedService (AC4). A
	// closed generic is an ordinary registered type in godi (Story 2-5 D-3); keying
	// it is no different from keying any concrete type, so IsKeyedService(closed, key)
	// must report true. Reuses the Story 2-5 open-generic fakes; NO AnyKey key.
	godi.AddKeyedSingleton[fakes.IFakeOpenGenericService[fakes.IFakeService], fakes.FakeOpenGenericService[fakes.IFakeService]](&r, "gen-key")
}

// AddFakeOpenGenericService is the Story 4.1 per-service open-generic helper
// (Mechanism A, AC1): a single-statement forwarder to a godi marker. The
// generator recognizes it, reads each call's type argument (e.g. IFakeService),
// substitutes T, and reduces it to the closed form-A registration
// (IFakeOpenGenericService[T], FakeOpenGenericService[T]) — which flows through
// the same pipeline as a hand-written closed marker (AD-14). The inner template
// marker's type args still mention T, so it is skipped (only the closed
// instantiations above register). Defined IN this analyzed package (Story 4.1
// scope; cross-package helpers are deferred).
func AddFakeOpenGenericService[T any](r *godi.Registry) *godi.Registry {
	return godi.AddTransient[fakes.IFakeOpenGenericService[T], fakes.FakeOpenGenericService[T]](r)
}

// AddConstrainedOpenGenericService is the Story 4.2 constrained per-service
// helper (Mechanism A): mirrors dotnet ConstrainedFakeOpenGenericService<T>
// where T : PocoClass. The Go type constraint [T PocoClassLike] rejects
// non-conforming types at compile time (AD-10).
func AddConstrainedOpenGenericService[T fakes.PocoClassLike](r *godi.Registry) *godi.Registry {
	return godi.AddTransient[fakes.IFakeOpenGenericService[T], fakes.ConstrainedFakeOpenGenericService[T]](r)
}

// AddInterfaceConstrainedService mirrors dotnet ClassWithInterfaceConstraint<T>
// where T : IEnumerable.
func AddInterfaceConstrainedService[T fakes.IterableLike](r *godi.Registry) *godi.Registry {
	return godi.AddTransient[fakes.IFakeOpenGenericService[T], fakes.ClassWithInterfaceConstraint[T]](r)
}

// AddAbstractConstrainedService mirrors dotnet ClassWithAbstractClassConstraint<T>
// where T : AbstractClass.
func AddAbstractConstrainedService[T fakes.AbstractClassLike](r *godi.Registry) *godi.Registry {
	return godi.AddTransient[fakes.IFakeOpenGenericService[T], fakes.ClassWithAbstractClassConstraint[T]](r)
}

// _ keeps register reachable: it exists solely as the generator's static-analysis
// input (read via go/types, never executed — AD-7), so nothing else calls it.
var _ = register
