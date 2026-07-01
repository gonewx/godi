// ATDD scaffold for Story 6.2 (AC1/AC2) — conformance suite: CallSite construction
// details re-expressed as OBSERVABLE godi behavior (semantic re-expression, NOT a
// new IR).
//
// Core positioning (Dev Notes D-1): dotnet's CallSiteFactoryTest / CallSiteTests
// are WHITE-BOX tests of the internal ServiceCallSite tree (ConstructorCallSite /
// CacheLocation / GetSlot / ExpressionResolverBuilder). godi has NO CallSite IR by
// construction (AD-1: B2 codegen direct assembly + reflect-free runtime
// interpreter; lifecycle encoded by field position, not a cache flag). This file
// re-expresses only the OBSERVABLE semantics those white-box tests exercise:
//
//	Task 2/3 — the structural constraint fakes' CLOSED materialization
//	           (CSF-08/10/12/19) + constraint-matrix collection order (CSF-20/21/22).
//	Task 4   — disposable capture/elide (CST-03..06, the io.Closer type gate AD-11),
//	           lifetime identity + nested scoped + ctor-error + collection
//	           coexistence (CST-01/02/07/08).
//	Task 5   — keyed resolution CallSite semantics (CST-11/12/13).
//
// Honesty gate (NFR7) red line: NO runtime CallSite/slot/cache representation is
// introduced to make a white-box test "pass". White-box IR assertions are archived
// N/A in threshold_csf_test.go + CALLSITE_DIVERGENCE.md, not faked here.
//
// AD-15 codegen coverage decision (Task 3, Dev Notes D-1/D-3): the "open-generic
// closed materialization" obligation axis already has codegen scenarios from 4.1
// (AddFakeOpenGenericService) and 4.2 (AddConstrainedOpenGenericService) with cross-
// provider differential equivalence. Story 6.2 ADDS a STRUCTURAL-constraint scenario
// to the codegen path — ClassWithSelfReferencingConstraint[ISelfComparable], the
// only structural fake with a form-A ctor (registration.go) — plus
// TestCallSite_SelfRef_DifferentialEquivalence, closing the 6.1 runtime-only gap.
// The other four structural fakes (class/struct/new/no-constraint) have no form-A
// ctor and stay runtime-only (documented decision, not a coverage gap).
//
// Test standard R4: pointer identity WITHIN one provider only; ACROSS providers use
// provider-stable observables (dynamic type names, collection counts, disposed
// bools). dynTypeName lives in spec_test.go (same package).
package conformance

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// ═══ Section 1: structural constraint runtime helpers (Task 2) ═══════════════
//
// Each helper is the runtime counterpart of a per-service open-generic forwarder
// for one structural constraint fake. Four of them (class/struct/new/no-constraint)
// have NO form-A ctor and take NO dependency — the factory constructs the impl
// directly (Value() returns zero T). The self-referencing fake injects value T
// through its form-A ctor (mirrors 4.1/4.2 addOpenGenericRT). CRUCIALLY, each helper
// COMPILES ONLY for T satisfying the Go constraint — the compile-time proof of
// AD-10: dotnet's "closed type does not satisfy constraint → runtime throw"
// (CSF-07/09/11/13/15/17/18) CANNOT be written in Go (it is a compile error), which
// is exactly why those methods are archived N/A, not MISSING.

// addStructConstraintRT mirrors ClassWithStructConstraint[T comparable]
// (dotnet where T : struct). Go comparable is the struct-constraint analog.
func addStructConstraintRT[T comparable](b *runtime.Builder) {
	runtime.AddTransient[fakes.IFakeOpenGenericService[T]](b, func(godi.Provider) fakes.IFakeOpenGenericService[T] {
		return &fakes.ClassWithStructConstraint[T]{}
	})
}

// addClassConstraintRT mirrors ClassWithClassConstraint[T any] (dotnet where T : class).
func addClassConstraintRT[T any](b *runtime.Builder) {
	runtime.AddTransient[fakes.IFakeOpenGenericService[T]](b, func(godi.Provider) fakes.IFakeOpenGenericService[T] {
		return &fakes.ClassWithClassConstraint[T]{}
	})
}

// addNewConstraintRT mirrors ClassWithNewConstraint[T any] (dotnet where T : new()).
func addNewConstraintRT[T any](b *runtime.Builder) {
	runtime.AddTransient[fakes.IFakeOpenGenericService[T]](b, func(godi.Provider) fakes.IFakeOpenGenericService[T] {
		return &fakes.ClassWithNewConstraint[T]{}
	})
}

// addNoConstraintsRT mirrors ClassWithNoConstraints[T any] (unconstrained control).
func addNoConstraintsRT[T any](b *runtime.Builder) {
	runtime.AddTransient[fakes.IFakeOpenGenericService[T]](b, func(godi.Provider) fakes.IFakeOpenGenericService[T] {
		return &fakes.ClassWithNoConstraints[T]{}
	})
}

// addSelfRefConstraintRT mirrors ClassWithSelfReferencingConstraint[T] (dotnet
// where T : IComparable<T>). It injects value T through the form-A ctor — the
// runtime counterpart of the codegen AddSelfReferencingConstrainedService[T].
func addSelfRefConstraintRT[T interface{ CompareTo(T) int }](b *runtime.Builder) {
	runtime.AddTransient[fakes.IFakeOpenGenericService[T]](b, func(p godi.Provider) fakes.IFakeOpenGenericService[T] {
		dep := p.GetService(reflect.TypeFor[T]()).(T)
		return fakes.NewClassWithSelfReferencingConstraint[T](dep)
	})
}

// ═══ Section 2: constraint materialization factory + cases (Task 2/3) ════════

// Closed-type reflect.Type helpers for the structural constraint materializations.
// Distinct closed types keep single-resolution (one impl each) from colliding with
// the collection (four impls over int); int16 is deliberately never enumerated.
func csfStructClosedType() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[int8]]()
}
func csfClassClosedType() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[string]]()
}
func csfNewClosedType() reflect.Type { return reflect.TypeFor[fakes.IFakeOpenGenericService[bool]]() }
func csfSelfRefClosedType() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.ISelfComparable]]()
}
func csfCollectionClosedType() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[int]]()
}
func csfUnmaterializedClosedType() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[int16]]()
}

// rtFactoryCallSiteConstraint materializes the structural constraint fakes over
// distinct closed types (Task 2/3). Single-resolution closed types carry one impl
// each (CSF-08/10/12/19); the collection over int carries FOUR impls in registration
// order (CSF-21/22); int16 is deliberately never enumerated (CSF-20). This is a
// RUNTIME-ONLY factory — the four no-ctor fakes cannot go form-A codegen; the
// self-ref fake's codegen coverage lives in registration.go (see the differential
// test below).
var rtFactoryCallSiteConstraint = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()

		// Candidate singleton injected into the self-referencing closed impl.
		runtime.AddSingleton[fakes.ISelfComparable](b, func(godi.Provider) fakes.ISelfComparable {
			return fakes.NewSelfComparable()
		})

		// Single-resolution: one structural impl per distinct closed type.
		addStructConstraintRT[int8](b)                   // CSF-08 (int8 satisfies comparable)
		addClassConstraintRT[string](b)                  // CSF-10 (string satisfies any)
		addNewConstraintRT[bool](b)                      // CSF-12 (bool satisfies any)
		addSelfRefConstraintRT[fakes.ISelfComparable](b) // CSF-19 (ISelfComparable satisfies self-ref)

		// Collection over int (CSF-21/22): four impls, registration order
		// struct→class→new→no-constraint. int satisfies comparable AND any, so all
		// four helpers COMPILE for int (the Go analog of dotnet's "int matches
		// struct/class/new" constraint-matrix row — see CALLSITE_DIVERGENCE.md).
		addStructConstraintRT[int](b)
		addClassConstraintRT[int](b)
		addNewConstraintRT[int](b)
		addNoConstraintsRT[int](b)

		// int16 is NEVER enumerated → CSF-20 unmaterialized-is-empty.

		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryCallSite is the codegen provider — it materializes the self-referencing
// closed type (registration.go, Story 6.2) among all other registered services.
var cgFactoryCallSite = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// callSiteConstraintRuntimeCases run against the RUNTIME provider only — the four
// no-ctor structural fakes materialize closed types (int8/string/bool/int) that the
// codegen provider does not register (they cannot go form-A). The self-ref closed
// type's codegen equivalence is covered by TestCallSite_SelfRef_DifferentialEquivalence.
var callSiteConstraintRuntimeCases = []SpecCase{
	// --- CSF-08 (adapted) — struct constraint satisfied → resolves --------------
	{
		Name: "callsite: CSF-08 ClassWithStructConstraint[int8] closed materialization resolves (T comparable satisfied)",
		Run: func(t *testing.T, p Provider) {
			svc := p.GetService(csfStructClosedType())
			testutil.NotNil(t, svc)
			testutil.True(t, strings.Contains(dynTypeName(svc), "ClassWithStructConstraint"),
				"struct-constrained closed type must resolve to ClassWithStructConstraint")
		},
	},
	// --- CSF-10 (adapted) — class constraint satisfied --------------------------
	{
		Name: "callsite: CSF-10 ClassWithClassConstraint[string] closed materialization resolves (T any / class analog)",
		Run: func(t *testing.T, p Provider) {
			svc := p.GetService(csfClassClosedType())
			testutil.NotNil(t, svc)
			testutil.True(t, strings.Contains(dynTypeName(svc), "ClassWithClassConstraint"),
				"class-constrained closed type must resolve to ClassWithClassConstraint")
		},
	},
	// --- CSF-12 (adapted) — new constraint satisfied ----------------------------
	{
		Name: "callsite: CSF-12 ClassWithNewConstraint[bool] closed materialization resolves (T any / new analog)",
		Run: func(t *testing.T, p Provider) {
			svc := p.GetService(csfNewClosedType())
			testutil.NotNil(t, svc)
			testutil.True(t, strings.Contains(dynTypeName(svc), "ClassWithNewConstraint"),
				"new-constrained closed type must resolve to ClassWithNewConstraint")
		},
	},
	// --- CSF-19 (adapted) — self-referencing constraint satisfied + injection ---
	{
		Name: "callsite: CSF-19 ClassWithSelfReferencingConstraint[ISelfComparable] closed materialization resolves + injects candidate (T IComparable<T> analog)",
		Run: func(t *testing.T, p Provider) {
			svc := p.GetService(csfSelfRefClosedType())
			testutil.NotNil(t, svc)
			testutil.True(t, strings.Contains(dynTypeName(svc), "ClassWithSelfReferencingConstraint"),
				"self-ref closed type must resolve to ClassWithSelfReferencingConstraint")
			// The injected value is the registered ISelfComparable singleton (AC2 DI).
			// Pointer identity WITHIN one provider is legal (R4).
			gen := svc.(fakes.IFakeOpenGenericService[fakes.ISelfComparable])
			testutil.NotNil(t, gen.Value())
			direct := p.GetService(reflect.TypeFor[fakes.ISelfComparable]()).(fakes.ISelfComparable)
			testutil.Equal(t, gen.Value(), direct)
		},
	},
	// --- CSF-21 (adapted) — materialized collection in registration order -------
	{
		Name: "callsite: CSF-21 materialized closed-type collection [int] resolves 4 impls in registration order (struct→class→new→no-constraint, Story 4.3 slot-order)",
		Run: func(t *testing.T, p Provider) {
			all := p.GetKeyedServices(csfCollectionClosedType(), nil)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 4)
			if len(all) != 4 {
				return
			}
			testutil.True(t, strings.Contains(dynTypeName(all[0]), "ClassWithStructConstraint"), "slot0=struct")
			testutil.True(t, strings.Contains(dynTypeName(all[1]), "ClassWithClassConstraint"), "slot1=class")
			testutil.True(t, strings.Contains(dynTypeName(all[2]), "ClassWithNewConstraint"), "slot2=new")
			testutil.True(t, strings.Contains(dynTypeName(all[3]), "ClassWithNoConstraints"), "slot3=no-constraint")
		},
	},
	// --- CSF-20 (adapted) — unmaterialized closed type → empty ------------------
	{
		Name: "callsite: CSF-20 unmaterialized closed type [int16] → empty collection + IsService false (godi materializes only enumerated closed types, AD-4/CAP-7)",
		Run: func(t *testing.T, p Provider) {
			all := p.GetKeyedServices(csfUnmaterializedClosedType(), nil)
			testutil.Nil(t, all)
			testutil.True(t, !p.IsService(csfUnmaterializedClosedType()),
				"unmaterialized closed generic must report IsService=false (no runtime MakeGenericType)")
		},
	},
	// --- CSF-22 (adapted + divergence) — constraint-matrix observable subset ----
	{
		Name: "callsite: CSF-22 constraint-matrix observable — collection[int] contains only impls whose Go constraint int satisfies; self-ref excluded (exact dotnet matrix N/A, CALLSITE_DIVERGENCE.md)",
		Run: func(t *testing.T, p Provider) {
			// Observable (adapted): int satisfies comparable(struct) + any(class/new/
			// no-constraint) → the four helpers compile for int; int does NOT satisfy
			// the self-referencing constraint (int has no CompareTo(int) int) → the
			// self-ref impl is NOT in the int collection. This is the Go-constraint
			// analog of dotnet's runtime constraint-matrix filter; the EXACT dotnet
			// matrix (struct/class/new/interface/abstract) is N/A — Go constraint kinds
			// do not map 1:1 (documented divergence, AD-10).
			all := p.GetKeyedServices(csfCollectionClosedType(), nil)
			testutil.Equal(t, len(all), 4)
			for _, svc := range all {
				testutil.True(t, !strings.Contains(dynTypeName(svc), "SelfReferencing"),
					"self-ref impl must NOT appear in the int collection (int lacks CompareTo(int) int)")
			}
		},
	},
}

// TestCallSite_Constraint_RuntimeProvider runs the structural constraint cases
// against the runtime oracle (CSF-08/10/12/19/20/21/22).
func TestCallSite_Constraint_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryCallSiteConstraint, callSiteConstraintRuntimeCases)
}

// TestCallSite_SelfRef_DifferentialEquivalence — AC2 / AD-15: the self-referencing
// structural constraint fake's CLOSED materialization is differentially equivalent
// across BOTH providers (runtime oracle + codegen B2). This is Story 6.2's codegen
// coverage of the open-generic-closed-materialization obligation axis WITH a
// structural constraint fake (closing the 6.1 runtime-only gap). Probes use
// provider-stable observables only (R4).
func TestCallSite_SelfRef_DifferentialEquivalence(t *testing.T) {
	rt := rtFactoryCallSiteConstraint
	cg := cgFactoryCallSite

	// IsService for the self-ref closed type (true both).
	AssertEquivalent(t, rt, cg, func(p Provider) bool {
		return p.IsService(csfSelfRefClosedType())
	})
	// Dynamic impl type for the self-ref closed type (same both).
	AssertEquivalent(t, rt, cg, func(p Provider) string {
		return dynTypeName(p.GetService(csfSelfRefClosedType()))
	})
	// Injected candidate's dynamic type — both providers inject the SAME
	// ISelfComparable impl type (value identity is per-provider, so observe the type).
	AssertEquivalent(t, rt, cg, func(p Provider) string {
		svc := p.GetService(csfSelfRefClosedType())
		if svc == nil {
			return "<nil>"
		}
		return dynTypeName(svc.(fakes.IFakeOpenGenericService[fakes.ISelfComparable]).Value())
	})
	// Collection count for the self-ref closed type (1 both).
	AssertEquivalent(t, rt, cg, func(p Provider) int {
		return len(p.GetKeyedServices(csfSelfRefClosedType(), nil))
	})
}

// ═══ Section 3: CallSiteTests observable behavior (Task 4) ═══════════════════

// rtFactoryCallSiteObservable registers the lifetime/nested/collection/ctor-error
// services for the CallSiteTests observable cases (CST-01/02/07/08). The ctor-error
// type uses a form-B factory that panics — resolved lazily, so it only fires in the
// CST-07 case.
var rtFactoryCallSiteObservable = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		// Three lifetimes (CST-01 identity).
		runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
			return fakes.NewFakeService()
		})
		runtime.AddScoped[fakes.IFakeScopedService](b, func(godi.Provider) fakes.IFakeScopedService {
			return fakes.NewFakeScopedService()
		})
		runtime.AddTransient[fakes.IFakeTransientService](b, func(godi.Provider) fakes.IFakeTransientService {
			return fakes.NewFakeTransientService()
		})
		// Nested scoped chain (CST-02): transient depends on scoped.
		runtime.AddTransient[fakes.IFakeTransientWithScopedDep](b, func(p godi.Provider) fakes.IFakeTransientWithScopedDep {
			dep := p.GetService(reflect.TypeFor[fakes.IFakeScopedService]()).(fakes.IFakeScopedService)
			return fakes.NewFakeTransientWithScopedDep(dep)
		})
		// Collection (CST-08): two impls of one service type.
		runtime.AddTransient[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
			return fakes.NewFakeOneMultipleService()
		})
		runtime.AddTransient[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
			return fakes.NewFakeTwoMultipleService()
		})
		// Direct dependency + collection coexistence (CST-08): FakeOuterService
		// depends on BOTH a single IFakeService AND a []IFakeMultipleService.
		runtime.AddTransient[fakes.IFakeOuterService](b, func(p godi.Provider) fakes.IFakeOuterService {
			single := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
			raw := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil)
			multi := make([]fakes.IFakeMultipleService, len(raw))
			for i, m := range raw {
				multi[i] = m.(fakes.IFakeMultipleService)
			}
			return fakes.NewFakeOuterService(single, multi)
		})
		// Ctor-error (CST-07): a form-B factory that panics on construction — the Go
		// analog of a throwing ctor (dotnet BuiltExpressionRethrowsOriginalException).
		runtime.AddTransient[*fakes.ClassWithThrowingCtor](b, func(p godi.Provider) *fakes.ClassWithThrowingCtor {
			dep := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
			return fakes.NewClassWithThrowingCtor(dep) // panics
		})
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

var callSiteObservableRuntimeCases = []SpecCase{
	// --- CST-01 (adapted) — three-lifetime identity + single-element IEnumerable -
	{
		Name: "callsite: CST-01 three-lifetime identity (singleton==, scoped== in-scope, transient!=) + single-element IEnumerable (expression engine N/A, AD-6 Runtime only)",
		Run: func(t *testing.T, p Provider) {
			// Singleton: same instance across resolves (R4 within one provider).
			s0 := p.GetService(reflect.TypeFor[fakes.IFakeService]())
			s1 := p.GetService(reflect.TypeFor[fakes.IFakeService]())
			testutil.Equal(t, s0, s1)

			// Transient: distinct instances each resolve (AD-14 not cached).
			tr0 := p.GetService(reflect.TypeFor[fakes.IFakeTransientService]())
			tr1 := p.GetService(reflect.TypeFor[fakes.IFakeTransientService]())
			testutil.True(t, tr0 != tr1, "transient must be a fresh instance each resolve")

			// Scoped: same within one scope.
			scope := p.CreateScope()
			sc0 := scope.GetService(reflect.TypeFor[fakes.IFakeScopedService]())
			sc1 := scope.GetService(reflect.TypeFor[fakes.IFakeScopedService]())
			testutil.Equal(t, sc0, sc1)
			testutil.NoError(t, scope.Close())

			// Single-element IEnumerable: IFakeService has exactly one registration.
			coll := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeService](), nil)
			testutil.Equal(t, len(coll), 1)
		},
	},
	// --- CST-02 (adapted) — nested scoped resolution ----------------------------
	{
		Name: "callsite: CST-02 nested scoped resolution — transient-with-scoped-dep resolves its scoped dependency (non-nil), shared within one scope (BuiltExpressionCanResolveNestedScopedService)",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()
			defer func() { _ = scope.Close() }()

			outer := scope.GetService(reflect.TypeFor[fakes.IFakeTransientWithScopedDep]())
			testutil.NotNil(t, outer)
			dep := outer.(fakes.IFakeTransientWithScopedDep).ScopedDep()
			testutil.NotNil(t, dep)

			// The nested scoped dependency is the SAME instance the scope resolves
			// directly — nested resolution shares the scoped instance within one scope.
			direct := scope.GetService(reflect.TypeFor[fakes.IFakeScopedService]()).(fakes.IFakeScopedService)
			testutil.Equal(t, dep, direct)
		},
	},
	// --- CST-07 (adapted) — ctor error propagation (= SPCT-01) ------------------
	{
		Name: "callsite: CST-07 ctor error propagation — resolving a panicking form-B factory propagates the panic (BuiltExpressionRethrowsOriginalExceptionFromConstructor = SPCT-01)",
		Run: func(t *testing.T, p Provider) {
			testutil.Panics(t, func() {
				_ = p.GetService(reflect.TypeFor[*fakes.ClassWithThrowingCtor]())
			})
		},
	},
	// --- CST-08 (adapted) — service as both []T element and direct dependency ---
	{
		Name: "callsite: CST-08 service used as both []T element and a direct dependency coexist without conflict (DoesNotThrowWhenServiceIsUsedAsEnumerableAndNotInOneCallSite)",
		Run: func(t *testing.T, p Provider) {
			// FakeOuterService depends on a single IFakeService AND a
			// []IFakeMultipleService — two distinct call-site shapes in one graph,
			// resolved without conflict.
			outer := p.GetService(reflect.TypeFor[fakes.IFakeOuterService]())
			testutil.NotNil(t, outer)
			o := outer.(fakes.IFakeOuterService)
			testutil.NotNil(t, o.SingleService())
			testutil.Equal(t, len(o.MultipleServices()), 2)

			// The collection also resolves directly as []IFakeMultipleService.
			coll := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil)
			testutil.Equal(t, len(coll), 2)
		},
	},
}

// TestCallSite_Observable_RuntimeProvider runs the CallSiteTests observable cases
// against the runtime oracle (CST-01/02/07/08). The expression-compilation engine
// (dotnet CompileCallSite) itself is N/A — godi has only the Runtime engine (AD-6).
func TestCallSite_Observable_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryCallSiteObservable, callSiteObservableRuntimeCases)
}

// rtFactoryCallSiteDisposable registers 3 disposable services (io.Closer, all
// lifetimes) + non-disposable services (IFakeService + a []IFakeMultipleService
// collection). The disposable capture/elide observable is the disposer track count
// (CST-03..06): only io.Closer services are tracked (AD-11 type gate), a fact godi
// gets for free — dotnet's "elide" is an expression-compiler optimization, godi's is
// a type wall (Dev Notes D-5).
var rtFactoryCallSiteDisposable = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		// Disposable (io.Closer) services — CAPTURED (CST-03/04).
		runtime.AddSingleton[fakes.IFakeDisposableSingleton](b, func(godi.Provider) fakes.IFakeDisposableSingleton {
			return fakes.NewFakeDisposableSingleton()
		})
		runtime.AddScoped[fakes.IFakeDisposableScoped](b, func(godi.Provider) fakes.IFakeDisposableScoped {
			return fakes.NewFakeDisposableScoped()
		})
		runtime.AddTransient[fakes.IFakeDisposableTransient](b, func(godi.Provider) fakes.IFakeDisposableTransient {
			return fakes.NewFakeDisposableTransient()
		})
		// Non-disposable services — ELIDED (CST-05 single, CST-06 collection).
		runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
			return fakes.NewFakeService()
		})
		runtime.AddTransient[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
			return fakes.NewFakeOneMultipleService()
		})
		runtime.AddTransient[fakes.IFakeMultipleService](b, func(godi.Provider) fakes.IFakeMultipleService {
			return fakes.NewFakeTwoMultipleService()
		})
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// TestCallSite_DisposableCaptureAndElide — CST-03/04/05/06, the highest-value new
// observable of this story (Dev Notes D-5). It witnesses AD-11's io.Closer type gate
// as a single observable: the disposer's release count equals EXACTLY the number of
// io.Closer services (3), NOT 3 + the resolved non-disposables (1 single + 2
// collection). dotnet ELIDES disposable capture for non-disposable services as an
// expression-compiler optimization; godi cannot even track them (Disposer.Track only
// accepts io.Closer) — a type wall, not an optimization. runtime-only observable
// (codegen's LIFO release order is covered by Story 2.3).
func TestCallSite_DisposableCaptureAndElide(t *testing.T) {
	p := rtFactoryCallSiteDisposable.Build(t)
	scope := p.CreateScope()
	recorder := fakes.NewDisposeRecorder()

	type recordable interface {
		SetRecorder(*fakes.DisposeRecorder, string)
	}
	type disposable interface{ IsDisposed() bool }

	// Capture: resolve the 3 disposables (all lifetimes), attach the recorder.
	single := p.GetService(reflect.TypeFor[fakes.IFakeDisposableSingleton]())
	single.(recordable).SetRecorder(recorder, "singleton")
	scoped := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableScoped]())
	scoped.(recordable).SetRecorder(recorder, "scoped")
	transient := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableTransient]())
	transient.(recordable).SetRecorder(recorder, "transient")

	// Elide: resolve non-disposable services — CST-05 (single) + CST-06 (collection).
	// These are NOT io.Closer, so they can never be tracked (AD-11 type gate).
	nonDisp := scope.GetService(reflect.TypeFor[fakes.IFakeService]())
	testutil.NotNil(t, nonDisp)
	nonDispColl := scope.GetKeyedServices(reflect.TypeFor[fakes.IFakeMultipleService](), nil)
	testutil.Equal(t, len(nonDispColl), 2)

	// Before close: nothing released yet.
	testutil.Equal(t, len(recorder.Labels()), 0)

	// Close scope → scoped + transient released (LIFO); singleton is root-owned.
	testutil.NoError(t, scope.Close())
	// Close root → singleton released.
	testutil.NoError(t, p.Close())

	// CST-03/04 capture: all 3 disposables tracked + released.
	// CST-05/06 elide: the non-disposable single + 2-element collection contributed
	// ZERO — the release count is EXACTLY 3, not 3+1+2.
	testutil.Equal(t, len(recorder.Labels()), 3)

	// Per-fake capture proof (the io.Closer gate let exactly these through).
	testutil.True(t, single.(disposable).IsDisposed(), "disposable singleton must be captured+released")
	testutil.True(t, scoped.(disposable).IsDisposed(), "disposable scoped must be captured+released")
	testutil.True(t, transient.(disposable).IsDisposed(), "disposable transient must be captured+released")
}

// ═══ Section 4: CallSiteTests keyed resolution (Task 5) ══════════════════════

// rtFactoryCallSiteKeyed registers keyed services (Story 3.1 fakes) + a keyed
// closed-generic impl for CST-11/12/13. The dotnet callSite.Key IR assertion is N/A
// (godi has no CallSite); the OBSERVABLE "keyed service resolves to the correct
// impl" is what these cases witness.
var rtFactoryCallSiteKeyed = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		// CST-11: two keyed impls under distinct keys (Story 3.1).
		runtime.AddKeyedSingleton[fakes.IKeyedService](b, "service1", func(godi.Provider) fakes.IKeyedService {
			return fakes.NewKeyedServiceOne()
		})
		runtime.AddKeyedSingleton[fakes.IKeyedService](b, "service2", func(godi.Provider) fakes.IKeyedService {
			return fakes.NewKeyedServiceTwo()
		})
		// CST-12/13: keyed CLOSED-generic impl — the injected value is IFakeService.
		runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
			return fakes.NewFakeService()
		})
		runtime.AddKeyedSingleton[fakes.IFakeOpenGenericService[fakes.IFakeService]](b, "gen-key", func(p godi.Provider) fakes.IFakeOpenGenericService[fakes.IFakeService] {
			dep := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
			return fakes.NewFakeOpenGenericService[fakes.IFakeService](dep)
		})
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

var callSiteKeyedRuntimeCases = []SpecCase{
	// --- CST-11 (adapted) — keyed service resolves to correct impl --------------
	{
		Name: "callsite: CST-11 keyed service resolves to correct impl per key (callSite.Key IR N/A; observable keyed resolution, Story 3.1)",
		Run: func(t *testing.T, p Provider) {
			one := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "service1")
			two := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "service2")
			testutil.NotNil(t, one)
			testutil.NotNil(t, two)
			testutil.True(t, strings.Contains(dynTypeName(one), "KeyedServiceOne"), "service1 → KeyedServiceOne")
			testutil.True(t, strings.Contains(dynTypeName(two), "KeyedServiceTwo"), "service2 → KeyedServiceTwo")
			// A missing key is a pure miss (nil), never a panic (AD-16/AD-17).
			testutil.Nil(t, p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), "nonexistent"))
		},
	},
	// --- CST-12 (adapted) — keyed closed-generic impl resolves ------------------
	{
		Name: "callsite: CST-12 keyed closed-generic impl resolves + injects value (ServiceCallSite_ShouldHaveKey_WhenResolvingKeyedClosedImplementationOfOpenGenericService)",
		Run: func(t *testing.T, p Provider) {
			svc := p.GetKeyedService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]](), "gen-key")
			testutil.NotNil(t, svc)
			gen := svc.(fakes.IFakeOpenGenericService[fakes.IFakeService])
			testutil.NotNil(t, gen.Value())
			direct := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
			testutil.Equal(t, gen.Value(), direct) // injected value = the singleton (R4)
		},
	},
	// --- CST-13 (adapted) — keyed IEnumerable of closed-generic impl ------------
	{
		Name: "callsite: CST-13 keyed IEnumerable of closed-generic impl resolves (ServiceCallSite_ShouldHaveKey_WhenResolvingKeyedIEnumerableOfClosedImplementationOfOpenGenericService)",
		Run: func(t *testing.T, p Provider) {
			all := p.GetKeyedServices(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]](), "gen-key")
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 1)
			if len(all) != 1 {
				return
			}
			testutil.True(t, strings.Contains(dynTypeName(all[0]), "FakeOpenGenericService"),
				"keyed closed-generic collection element must be FakeOpenGenericService")
		},
	},
}

// TestCallSite_Keyed_RuntimeProvider runs the keyed CallSite observable cases
// against the runtime oracle (CST-11/12/13). Keyed dual-provider equivalence is
// already covered by Story 3.x; these confirm the CallSite-context observables.
func TestCallSite_Keyed_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryCallSiteKeyed, callSiteKeyedRuntimeCases)
}
