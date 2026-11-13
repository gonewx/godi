package codegen

import (
	"reflect"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/fakes"
)

// container_gen.go is produced from the registrations below by the real
// godi-gen static-analysis CLI (Story 2.1). Running `go generate ./...` from the
// module root re-emits it; TestCodegen_RegenComparison guards that the committed
// file stays byte-identical to a fresh run (AD-14: deterministic output, never
// edited by hand). Run under GOWORK=off so the generator analyzes this package in
// isolation from the auth workspace's sibling modules.
//
//go:generate go run github.com/gonewx/godi/cmd/godi-gen -pkg codegen -out container_gen.go .

// register is the static registration source the generator analyzes (AD-7): the
// godi.Add… markers are read from go/types, never executed. It mirrors the
// runtime oracle's registration (spec_test.go's rtFactory); the two are kept
// equivalent by hand (AD-14 / AD-15). Form A: FakeService has no dependency;
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

	// Story 4.3 (AC1/AC2, 接触点 6/7): closed-preferred-over-open via registration
	// ORDER. godi has no open/closed runtime distinction (AD-4/AD-10); "closed
	// preferred" is reproduced by registering the closed-specific impl LAST so
	// GetService last-wins returns it. Enumerable [open, closed] is order-
	// insensitive in dotnet (.Any), so the reversal is observable-equivalent.
	godi.AddSingleton[fakes.IGenSlotA, fakes.GenSlotA](&r)                                                                     // element (injected into open-produced)
	AddFakeOpenGenericService[fakes.IGenSlotA](&r)                                                                             // open-produced — FIRST
	godi.AddTransient[fakes.IFakeOpenGenericService[fakes.IGenSlotA], fakes.ClosedFakeOpenGenericService[fakes.IGenSlotA]](&r) // closed — LAST → wins single resolve

	// Story 4.3 (AC3, 接触点 8): mixed open/closed enumerable in dotnet slot order
	// [closed#1, open-produced, instance]. ALL transient: lifetime doesn't affect
	// slot order. Mixing lifetimes under one service type is supported too — both
	// providers resolve such registrations in registration order (eval fix A4,
	// covered by IMixedA / IMixedB below) — but this scenario keeps one lifetime.
	godi.AddSingleton[fakes.IGenSlotB, fakes.GenSlotB](&r)                                                                       // element
	godi.AddTransient[fakes.IFakeOpenGenericService[fakes.IGenSlotB], fakes.ClosedFakeOpenGenericService[fakes.IGenSlotB]](&r)   // closed#1 → enumerable[0]
	AddFakeOpenGenericService[fakes.IGenSlotB](&r)                                                                               // open-produced → enumerable[1]
	godi.AddTransient[fakes.IFakeOpenGenericService[fakes.IGenSlotB], fakes.InstanceFakeOpenGenericService[fakes.IGenSlotB]](&r) // instance → enumerable[2]

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

	// Story 6-1 (SPCT-18/20/21, AD-15 codegen coverage): struct (value-type)
	// services on the codegen path. dotnet boxes struct services into `object`;
	// the godi analog is a struct with VALUE-receiver interface methods, so the
	// struct value (not *struct) satisfies the interface and is boxed into `any`
	// when resolved. These verify the GENERATED container keys value types by
	// reflect.Type and stores them in []any collections exactly as for pointers
	// (D-3 risk) — a coverage gap the runtime-only standalone tests cannot close.
	// IStructFakeService as a singleton (SPCT-18 value-type resolution), injected
	// into StructConsumer (SPCT-20 struct-as-dependency), plus two struct impls of
	// IStructFakeMultipleService for collection ordering (SPCT-21).
	godi.AddSingleton[fakes.IStructFakeService, fakes.StructFakeService](&r)
	godi.AddSingleton[fakes.IStructConsumer, fakes.StructConsumer](&r)
	godi.AddSingleton[fakes.IStructFakeMultipleService, fakes.StructFakeOne](&r)
	godi.AddSingleton[fakes.IStructFakeMultipleService, fakes.StructFakeTwo](&r)

	// Story 6-2 (CSF-19, AD-15 codegen coverage): the self-referencing structural
	// constraint fake ClassWithSelfReferencingConstraint[T] materialized on the
	// codegen path. It is the ONLY one of the 5 structural constraint fakes with a
	// form-A New{Impl} ctor (the other four — class/struct/new/no-constraint — have
	// no ctor and stay runtime-only, Dev Notes D-1). ISelfComparable is the injected
	// candidate (value T dependency); the per-service helper mirrors 4.1/4.2's
	// Mechanism A. A FRESH closed type IFakeOpenGenericService[ISelfComparable] — no
	// collision with existing closed instantiations. This gives the "open-generic
	// closed materialization" obligation axis (AD-15) a structural-constraint
	// scenario in codegen + cross-provider differential equivalence, closing the
	// 6.1 runtime-only gap.
	godi.AddSingleton[fakes.ISelfComparable, fakes.SelfComparable](&r)
	AddSelfReferencingConstrainedService[fakes.ISelfComparable](&r)

	// Story 6-3 (AC5/AC6, AD-15 codegen coverage): async-disposable services on
	// the codegen path. Registered by POINTER type so the service key matches the
	// spec test's reflect.TypeFor[*fakes.FakeAsyncOnlyDisposable]() and the emitted
	// field is concrete (the emitted TrackAny call receives it directly).
	//
	// TRANSIENT — not scoped. The choice dates from when codegen eagerly
	// materialized every scoped service in CreateScope, where a scoped async-only
	// service in this SHARED container would have made every sync scope.Close
	// error (AC3: "only implements AsyncDisposable"). Scoped services are now
	// built on first resolution, so that constraint is gone; the registration is
	// kept as is. Transient is tracked only when the async tests explicitly
	// resolve it, preserving Story 2.3 zero-regression (AC6) while still
	// exercising scope-level async release: the async tests resolve these FROM
	// a scope, so that scope's disposer tracks them and scope.CloseAsync releases
	// them (AC2). The runtime spec oracle
	// registers them scoped; the axis under test (async release / prefer-async /
	// LIFO / sync-errors) is observably identical across both (AD-15).
	//
	// async-only is registered BEFORE both so the resolve/track order makes the
	// LIFO release sequence [both, async-only]. FakeAsyncOnlyDisposable is
	// async-only (CloseAsync, no io.Closer → sync Close errors, AC3);
	// FakeSyncAsyncDisposable is both (prefer-async under CloseAsync, AC2).
	godi.AddTransient[*fakes.FakeAsyncOnlyDisposable, fakes.FakeAsyncOnlyDisposable](&r)
	godi.AddTransient[*fakes.FakeSyncAsyncDisposable, fakes.FakeSyncAsyncDisposable](&r)

	// Story 6.4 code review (2026-07-01, AD-15/D-8 "final set equivalence"): the
	// codegen leg for TestFinalSetEquivalence_Replace_DynType declares DIRECTLY
	// the same final set the runtime leg reaches via mutation (Add(One) +
	// Add(Two) + Replace(One) removes the first ServiceType match and appends
	// the new one -> final = [Two, One-new]). Registered in that exact order so
	// GetKeyedServices(t, nil) returns the same [Two, One] dynamic-type sequence
	// on both providers. Dedicated fakes (fakes_extensions_finalset.go) so this
	// cannot perturb the existing IFakeMultipleService count/order assertions.
	// IFakeRemovedService (the RemoveAll_IsService scenario) is deliberately
	// NOT registered here — codegen's generated IsService for it is naturally
	// false, matching the runtime leg's post-RemoveAll state without needing an
	// explicit "absence" declaration.
	godi.AddSingleton[fakes.IFakeReplaceFinalService, fakes.FakeReplaceFinalTwo](&r)
	godi.AddSingleton[fakes.IFakeReplaceFinalService, fakes.FakeReplaceFinalOne](&r)

	// Story 6.4 code review (2026-07-01, D-8 instance-registration result
	// obligation): codegen has no raw-instance DSL entry (D-1/D-4 scope γ), so
	// the equivalent final state is declared via a form-B factory producing a
	// value of the same concrete type AddSingletonInstance would hold on the
	// runtime leg. Equivalence is asserted on observable resolve results
	// (dynamic type name), never on cross-provider pointer identity (R4).
	godi.AddSingletonFactory[fakes.IFakeInstanceService](&r, func(p godi.Provider) fakes.IFakeInstanceService {
		return fakes.NewFakeInstanceService()
	})

	// Eval fix A3: form-B products are released too. The generator cannot see
	// the factory's product type, so the container hands every form-B product to
	// Disposer.TrackAny, which releases these io.Closer fakes on root/scope Close.
	godi.AddSingletonFactory[fakes.IFakeFormBDisposableSingleton](&r, func(p godi.Provider) fakes.IFakeFormBDisposableSingleton {
		return fakes.NewFakeFormBCloser()
	})
	godi.AddScopedFactory[fakes.IFakeFormBDisposableScoped](&r, func(p godi.Provider) fakes.IFakeFormBDisposableScoped {
		return fakes.NewFakeFormBCloser()
	})

	// Eval fix A4: one service type registered with mixed lifetimes resolves in
	// registration order, as in dotnet — GetService returns the LAST registration
	// whatever its lifetime, GetKeyedServices(t, nil) returns all of them in
	// registration order, each element keeping its own lifetime. IMixedA is
	// singleton then transient, IMixedB transient then singleton; the runtime
	// leg (spec_mixed_lifetime_test.go) registers the same sequence.
	godi.AddSingleton[fakes.IMixedA, fakes.MixedSingletonFirst](&r)
	godi.AddTransient[fakes.IMixedA, fakes.MixedTransientLast](&r)
	godi.AddTransient[fakes.IMixedB, fakes.MixedTransientFirst](&r)
	godi.AddSingleton[fakes.IMixedB, fakes.MixedSingletonLast](&r)
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

// AddSelfReferencingConstrainedService is the Story 6.2 structural per-service
// helper (Mechanism A): mirrors dotnet ClassWithSelfReferencingConstraint<T>
// where T : IComparable<T>. The Go self-referencing constraint
// interface{ CompareTo(T) int } is F-bounded; the generator is transparent to it
// (AD-10) and reduces the call to the closed form-A registration exactly as for
// the 4.2 constrained helpers — reading NewClassWithSelfReferencingConstraint's
// `value T` parameter and wiring the registered T dependency (AC2).
func AddSelfReferencingConstrainedService[T interface{ CompareTo(T) int }](r *godi.Registry) *godi.Registry {
	return godi.AddTransient[fakes.IFakeOpenGenericService[T], fakes.ClassWithSelfReferencingConstraint[T]](r)
}

// _ keeps register reachable: it exists solely as the generator's static-analysis
// input (read via go/types, never executed — AD-7), so nothing else calls it.
var _ = register
