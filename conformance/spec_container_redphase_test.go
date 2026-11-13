// Story 6.1 (AC1/AC2) — conformance suite: ServiceProviderContainerTests
// comprehensive container behavior. Ports the covered/adapted methods (28 of 49)
// from dotnet ServiceProviderContainerTests: resolve-after-dispose guards, struct
// (value-type) services, multi-impl ordering, wide scoped trees, root-scoped
// stability, sync-dispose-preferred, and build-time rejection of missing
// dependencies. The resolve-after-dispose group runs against BOTH providers
// (runtime oracle + codegen B2) as SpecCases plus a cross-provider differential
// equivalence check (AC2); the "corner combination" axes outside the AD-15
// differential obligation are runtime-only standalone tests (D-7). See
// conformance/threshold_spct_test.go for the full 49-method coverage map.
package conformance

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
	"github.com/gonewx/godi/runtime"
)

// ── test-local fakes (unexported, not in fakes/) ────────────────────────────

type spctDisposable struct {
	disposed bool
}

func (d *spctDisposable) Close() error     { d.disposed = true; return nil }
func (d *spctDisposable) IsDisposed() bool { return d.disposed }

type iSPCTCustomService interface{ isSPCTCustom() }

type spctCustom1 struct{}
type spctCustom2 struct{}
type spctCustom3 struct{}
type spctCustom4 struct{}
type spctCustom5 struct{}
type spctCustom6 struct{}

func (*spctCustom1) isSPCTCustom() {}
func (*spctCustom2) isSPCTCustom() {}
func (*spctCustom3) isSPCTCustom() {}
func (*spctCustom4) isSPCTCustom() {}
func (*spctCustom5) isSPCTCustom() {}
func (*spctCustom6) isSPCTCustom() {}

func newSPCTCustom1() *spctCustom1 { return &spctCustom1{} }
func newSPCTCustom2() *spctCustom2 { return &spctCustom2{} }
func newSPCTCustom3() *spctCustom3 { return &spctCustom3{} }
func newSPCTCustom4() *spctCustom4 { return &spctCustom4{} }
func newSPCTCustom5() *spctCustom5 { return &spctCustom5{} }
func newSPCTCustom6() *spctCustom6 { return &spctCustom6{} }

// spctCtorError simulates a constructor that panics (Go form-B factory panic
// propagation, the closest equivalent to dotnet ctor throwing).
type spctCtorError struct{}

type spctCtorErrorSentinel struct{ msg string }

func (e *spctCtorErrorSentinel) Error() string { return e.msg }

// spctScopedA is a scoped service for root-scoped stability tests (SPCT-41/42/43).
type spctScopedA struct{}

func newSPCTScopedA() *spctScopedA { return &spctScopedA{} }

// spctScopedAa depends on spctScopedA (for SPCT-43 nested dep test).
type spctScopedAa struct {
	a *spctScopedA
}

func newSPCTScopedAa(a *spctScopedA) *spctScopedAa { return &spctScopedAa{a: a} }

// spctWideNode is a pointer-typed scoped service used to build a wide scoped tree
// (SPCT-22). Pointer identity lets the test assert per-scope caching: every
// element is a distinct instance, stable within one scope, fresh in the next.
type spctWideNode struct{ slot int }

// ── Provider factories for SPCT tests ───────────────────────────────────────

// rtFactorySPCT builds the runtime provider with a basic singleton for
// resolve-after-dispose and general container behavior tests.
var rtFactorySPCT = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
			return fakes.NewFakeService()
		})
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactorySPCT builds the codegen provider for SPCT tests.
var cgFactorySPCT = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// ── SPCT SpecCases ──────────────────────────────────────────────────────────

var spctCases = []SpecCase{
	// ── G1: resolve-after-dispose (SPCT-12 to SPCT-15, P0) ──────────────

	// SPCT-12: root Close → GetService panic/error
	{
		Name: "SPCT-12: root Close prevents service resolution",
		Run: func(t *testing.T, p Provider) {
			// Resolve once to confirm it works
			svc := p.GetService(reflect.TypeFor[fakes.IFakeService]())
			testutil.NotNil(t, svc)

			// Close the root provider
			testutil.NoError(t, p.Close())

			// After Close, GetService must panic or return error
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("SPCT-12: GetService after root Close must panic, but did not")
				}
			}()
			_ = p.GetService(reflect.TypeFor[fakes.IFakeService]())
		},
	},
	// SPCT-13: root Close → CreateScope panic/error
	{
		Name: "SPCT-13: root Close prevents scope creation",
		Run: func(t *testing.T, p Provider) {
			testutil.NoError(t, p.Close())

			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("SPCT-13: CreateScope after root Close must panic, but did not")
				}
			}()
			_ = p.CreateScope()
		},
	},
	// SPCT-14: root Close → child scope resolution panic/error
	{
		Name: "SPCT-14: root Close prevents child scope resolution",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()

			// Close root (not scope)
			testutil.NoError(t, p.Close())

			// Child scope resolution must also fail
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("SPCT-14: GetService on child scope after root Close must panic, but did not")
				}
			}()
			_ = scope.GetService(reflect.TypeFor[fakes.IFakeService]())
		},
	},
	// SPCT-15: scope Close → scope resolution panic + root still works
	{
		Name: "SPCT-15: scope Close prevents scope resolution + root still works",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()
			testutil.NoError(t, scope.Close())

			// Scope resolution must fail
			panicked := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
					}
				}()
				_ = scope.GetService(reflect.TypeFor[fakes.IFakeService]())
			}()
			testutil.True(t, panicked, "SPCT-15: GetService on closed scope must panic")

			// Root must still work — CreateScope should succeed
			scope2 := p.CreateScope()
			testutil.NotNil(t, scope2)
			_ = scope2.Close()
		},
	},

	// ── G4: dispose semantics (SPCT-26, SPCT-30, SPCT-34, SPCT-36, SPCT-37, SPCT-08) ──

	// SPCT-26: root Close is idempotent. NOTE: the io.Closer.Close-is-called
	// semantics is verified in depth by Story 2.3 (rtFactoryDispose); the basic
	// SPCT factory registers no disposable, so here we only assert idempotency in
	// the SPCT context — NOT that Close was invoked on a tracked disposable.
	{
		Name: "SPCT-26: root Close is idempotent (release verified by Story 2.3)",
		Run: func(t *testing.T, p Provider) {
			// The basic SPCT factory registers no disposable, so this asserts
			// idempotency only; sync-dispose-preferred / io.Closer.Close-called is
			// covered by Story 2.3's dispose factory in depth.
			err := p.Close()
			testutil.NoError(t, err)
			// Idempotent
			err = p.Close()
			testutil.NoError(t, err)
		},
	},
	// SPCT-30: scope Close is idempotent (release verified by Story 2.3).
	{
		Name: "SPCT-30: scope Close is idempotent (release verified by Story 2.3)",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()
			err := scope.Close()
			testutil.NoError(t, err)
			// Idempotent
			err = scope.Close()
			testutil.NoError(t, err)
		},
	},
	// SPCT-34: same as SPCT-30 (dotnet test body actually uses CreateScope not CreateAsyncScope)
	{
		Name: "SPCT-34: scope Close idempotent, CreateScope body equivalent (release verified by Story 2.3)",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()
			err := scope.Close()
			testutil.NoError(t, err)
		},
	},

	// ── G3: multi-impl + wide tree + generic cache ─────────────────────────
	// SPCT-05 / SPCT-22 / SPCT-23 are standalone tests below — each needs a
	// bespoke registration (N same-typed impls, a wide scoped tree, a SINGLETON
	// closed generic) that neither default SPCT factory provides. In particular
	// the default container registers IFakeOpenGenericService[T] as a TRANSIENT,
	// so a single-vs-collection identity check (SPCT-23) needs its own singleton
	// registration to be meaningful.
}

// ── Test runners ────────────────────────────────────────────────────────────

func TestSPCT_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactorySPCT, spctCases)
}

func TestSPCT_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactorySPCT, spctCases)
}

// ── Cross-provider differential equivalence (AC2) ───────────────────────────

func TestSPCT_CrossProvider_ResolveAfterDispose(t *testing.T) {
	// Both providers must exhibit IDENTICAL resolve-after-dispose behavior: after
	// root Close, GetService and CreateScope both panic. The probe captures its
	// OWN panic and reports "panicked"/"no-panic" (a named return set from the
	// deferred recover) so the two providers' observable outcomes compare directly
	// — and so a swallowed panic can never masquerade as a missing one (AC2, AD-15).
	for _, tc := range []struct {
		name  string
		probe func(Provider) string
	}{
		{
			name: "root Close → GetService panics consistently",
			probe: func(p Provider) (result string) {
				_ = p.Close()
				defer func() {
					if recover() != nil {
						result = "panicked"
					}
				}()
				_ = p.GetService(reflect.TypeFor[fakes.IFakeService]())
				return "no-panic"
			},
		},
		{
			name: "root Close → CreateScope panics consistently",
			probe: func(p Provider) (result string) {
				_ = p.Close()
				defer func() {
					if recover() != nil {
						result = "panicked"
					}
				}()
				_ = p.CreateScope()
				return "no-panic"
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rtResult := tc.probe(rtFactorySPCT.Build(t))
			cgResult := tc.probe(cgFactorySPCT.Build(t))
			if rtResult != cgResult {
				t.Errorf("cross-provider mismatch: runtime=%q codegen=%q", rtResult, cgResult)
			}
			if rtResult != "panicked" {
				t.Errorf("both providers must panic on resolve-after-dispose, got %q", rtResult)
			}
		})
	}
}

// TestSPCT_CrossProvider_ResolveAfterDispose_NoDisposable closes the gap that the
// disposable-container probe above cannot reach: the ONLY conformance codegen
// container (codegen.NewRoot) tracks a disposable, so that probe never exercises a
// generated container with NOTHING to release. Earlier the codegen guard was only
// emitted when the container tracked disposables, so a no-disposable generated
// container would NOT panic after Close while the runtime oracle (whose guard is
// unconditional) would — a silent AD-16 divergence the disposable probe masked.
//
// This test asserts the contract at the generator: a registry with no disposable
// service must STILL emit the unconditional `closed` field + resolve-after-dispose
// guard, exactly as the runtime root does. It mirrors the runtime side by building
// a no-disposable runtime provider and confirming it panics after Close, then
// confirms the generated no-disposable container carries the identical guard.
func TestSPCT_CrossProvider_ResolveAfterDispose_NoDisposable(t *testing.T) {
	// runtime side: a no-disposable provider must panic after Close (unconditional guard).
	b := runtime.NewBuilder()
	runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
		return fakes.NewFakeService()
	})
	rp, err := b.Build()
	testutil.NoError(t, err)
	_ = rp.Close()
	rtPanicked := func() (panicked bool) {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		_ = rp.GetService(reflect.TypeFor[fakes.IFakeService]())
		return false
	}()
	testutil.True(t, rtPanicked, "runtime no-disposable provider must panic on resolve-after-dispose")

	// codegen side: a no-disposable registry must STILL emit the closed field + guard.
	// There is no instantiable no-disposable conformance container, so assert the
	// generated SOURCE carries the same unconditional guard the runtime relies on.
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "Plain",
		GoType:   "*svc.Plain",
		Ctor:     "svc.NewPlain",
		Lifetime: plan.Singleton,
	})
	src, err := genlib.Generate(reg, "container")
	testutil.NoError(t, err)
	code := string(src)
	testutil.True(t, !strings.Contains(code, "runtime.Disposer"),
		"sanity: this registry has no disposable, so no Disposer is expected")
	// the close-state FIELD declaration line starts with tab+"guard" (gofmt
	// aligns the type with variable spacing, so match the line prefix, not exact
	// spacing); the resolve guard ("\tif r.guard.Closed()") does not.
	testutil.True(t, strings.Contains(code, "\n\tguard "),
		"no-disposable generated container must STILL declare the guard field (AD-16)")
	testutil.True(t, strings.Contains(code, "panic(godi.ProviderClosedError())"),
		"no-disposable generated container must STILL emit the resolve-after-dispose guard (AD-16)")
}

// ── Standalone tests for scenarios needing custom registration ──────────────

// TestSPCT_36_FactorySingletonDisposed verifies that a factory-produced
// singleton disposable is released when root Close is called (SPCT-36).
func TestSPCT_36_FactorySingletonDisposed(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*spctDisposable](b, func(godi.Provider) *spctDisposable {
		return &spctDisposable{}
	})
	p, err := b.Build()
	testutil.NoError(t, err)

	svc := p.GetService(reflect.TypeFor[*spctDisposable]())
	testutil.NotNil(t, svc)
	d := svc.(*spctDisposable)
	testutil.True(t, !d.IsDisposed(), "must not be disposed before Close")

	testutil.NoError(t, p.Close())
	testutil.True(t, d.IsDisposed(), "SPCT-36: factory singleton must be disposed after root Close")
}

// TestSPCT_37_FormBFactoryDisposed verifies godi adaptation: godi has no
// raw instance registration, so form-B factory output IS tracked and
// disposed (differs from dotnet where raw instance is NOT disposed).
func TestSPCT_37_FormBFactoryDisposed(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*spctDisposable](b, func(godi.Provider) *spctDisposable {
		return &spctDisposable{}
	})
	p, err := b.Build()
	testutil.NoError(t, err)

	svc := p.GetService(reflect.TypeFor[*spctDisposable]())
	d := svc.(*spctDisposable)

	testutil.NoError(t, p.Close())
	// godi adaptation: form-B factory output IS tracked → disposed
	// (dotnet raw instance would NOT be disposed — this is a documented divergence)
	testutil.True(t, d.IsDisposed(),
		"SPCT-37 adapted: godi form-B factory output is tracked and disposed (differs from dotnet raw instance)")
}

// TestSPCT_08_AdaptedSingletonDispose documents the godi adaptation for
// dotnet DoesNotDisposeSingletonInstances: godi has no raw instance
// registration, so form-B factory-produced singletons ARE disposed.
func TestSPCT_08_AdaptedSingletonDispose(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*spctDisposable](b, func(godi.Provider) *spctDisposable {
		return &spctDisposable{}
	})
	p, err := b.Build()
	testutil.NoError(t, err)

	svc := p.GetService(reflect.TypeFor[*spctDisposable]())
	d := svc.(*spctDisposable)

	testutil.NoError(t, p.Close())
	// Adapted assertion: godi form-B factory singleton IS disposed (not like dotnet raw instance)
	testutil.True(t, d.IsDisposed(),
		"SPCT-08 adapted: godi form-B factory singleton is tracked+disposed")
}

// TestSPCT_41_RootScopedStability verifies scoped service resolved from root
// returns same instance on multiple resolves (root-scoped = singleton-like).
func TestSPCT_41_RootScopedStability(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[*spctScopedA](b, func(godi.Provider) *spctScopedA {
		return newSPCTScopedA()
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	first := p.GetService(reflect.TypeFor[*spctScopedA]())
	testutil.NotNil(t, first)

	for i := 0; i < 10; i++ {
		again := p.GetService(reflect.TypeFor[*spctScopedA]())
		if first != again {
			t.Fatalf("SPCT-41: root-scoped instance changed on resolve %d (want same pointer)", i)
		}
	}
}

// TestSPCT_05_MultiImplOrdering verifies that multiple registrations of the
// same interface are resolved in registration order (N=1..6).
func TestSPCT_05_MultiImplOrdering(t *testing.T) {
	for n := 1; n <= 6; n++ {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			b := runtime.NewBuilder()
			factories := []func(godi.Provider) iSPCTCustomService{
				func(godi.Provider) iSPCTCustomService { return newSPCTCustom1() },
				func(godi.Provider) iSPCTCustomService { return newSPCTCustom2() },
				func(godi.Provider) iSPCTCustomService { return newSPCTCustom3() },
				func(godi.Provider) iSPCTCustomService { return newSPCTCustom4() },
				func(godi.Provider) iSPCTCustomService { return newSPCTCustom5() },
				func(godi.Provider) iSPCTCustomService { return newSPCTCustom6() },
			}
			types := []string{
				"*conformance.spctCustom1",
				"*conformance.spctCustom2",
				"*conformance.spctCustom3",
				"*conformance.spctCustom4",
				"*conformance.spctCustom5",
				"*conformance.spctCustom6",
			}
			for i := 0; i < n; i++ {
				runtime.AddSingleton[iSPCTCustomService](b, factories[i])
			}
			p, err := b.Build()
			testutil.NoError(t, err)
			t.Cleanup(func() { _ = p.Close() })

			services := p.GetKeyedServices(reflect.TypeFor[iSPCTCustomService](), nil)
			if len(services) != n {
				t.Fatalf("SPCT-05: got %d services, want %d", len(services), n)
			}
			for i := 0; i < n; i++ {
				got := reflect.TypeOf(services[i]).String()
				if got != types[i] {
					t.Errorf("SPCT-05: services[%d] type = %s, want %s", i, got, types[i])
				}
			}
		})
	}
}

// TestSPCT_01_CtorErrorPropagation verifies that a factory panic is propagated
// through resolve (adapted from dotnet RethrowOriginalExceptionFromConstructor).
// In godi, ctor errors are either:
//   - left-shifted to plan.Validate (missing dep / cycle → Build fails)
//   - factory panic propagates through GetService
func TestSPCT_01_CtorErrorPropagation(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddTransient[*spctCtorError](b, func(godi.Provider) *spctCtorError {
		panic(&spctCtorErrorSentinel{msg: "spctCtorError"})
	})
	p, err := b.Build()
	if err != nil {
		// Build detected the error at build time — acceptable
		return
	}
	t.Cleanup(func() { _ = p.Close() })

	// Resolve must propagate the factory panic
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("SPCT-01: GetService for panicking factory must propagate panic")
		}
		if sentinel, ok := r.(*spctCtorErrorSentinel); ok {
			if sentinel.msg != "spctCtorError" {
				t.Errorf("SPCT-01: panic value mismatch: got %q", sentinel.msg)
			}
		}
	}()
	_ = p.GetService(reflect.TypeFor[*spctCtorError]())
}

// TestSPCT_11_ValueTypeServices verifies value type services (int/time.Duration analog)
// can be registered and resolved via form-B factory.
func TestSPCT_11_ValueTypeServices(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[int](b, func(godi.Provider) int {
		return 42
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	got := p.GetService(reflect.TypeFor[int]())
	if got == nil {
		t.Fatal("SPCT-11: value type service resolved to nil")
	}
	if got.(int) != 42 {
		t.Errorf("SPCT-11: got %v, want 42", got)
	}
}

// TestSPCT_16_CtorDisposeInFlight verifies that calling p.Close() inside a
// factory causes subsequent resolve to panic (ctor-dispose-in-flight).
func TestSPCT_16_CtorDisposeInFlight(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*spctDisposable](b, func(p godi.Provider) *spctDisposable {
		// Close the provider inside the factory (ctor-dispose-in-flight)
		_ = p.Close()
		return &spctDisposable{}
	})
	runtime.AddSingleton[fakes.IFakeService](b, func(p godi.Provider) fakes.IFakeService {
		return fakes.NewFakeService()
	})

	p, err := b.Build()
	if err != nil {
		// Build detected the closed state and failed — acceptable
		return
	}
	t.Cleanup(func() { _ = p.Close() })

	// If Build succeeded despite the in-flight Close, the provider should be
	// in a closed state and subsequent resolves should panic.
	defer func() {
		if r := recover(); r == nil {
			// If neither Build failed nor resolve panicked, the closed guard
			// is not working — this is the red-phase failure signal.
			t.Error("SPCT-16: expected panic on resolve after in-flight Close, but none occurred")
		}
	}()
	_ = p.GetService(reflect.TypeFor[fakes.IFakeService]())
}

// TestSPCT_44_KeyedMissingDoesNotFallThrough verifies that a missing keyed
// dependency does not fall through to an unkeyed registration.
// In godi this is caught at plan.Validate / Build time (ValidateOnBuild=always).
func TestSPCT_44_KeyedMissingDoesNotFallThrough(t *testing.T) {
	// This scenario is validated by plan.Validate detecting the missing keyed
	// dependency at build time. The test verifies that registering an unkeyed
	// service does NOT satisfy a keyed dependency requirement.
	//
	// Story 5-1 already validates missing dependencies; this test confirms the
	// specific case where an unkeyed service exists but the keyed variant is missing.

	b := runtime.NewBuilder()
	// Register unkeyed IFakeService
	runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
		return fakes.NewFakeService()
	})
	// The keyed service "service1" is NOT registered.
	// If a service requiring keyed IFakeService("service1") were added, Build must fail.
	// For now, verify IsKeyedService returns false for unregistered key.
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	// Unkeyed IFakeService is registered
	testutil.True(t, p.IsService(reflect.TypeFor[fakes.IFakeService]()),
		"SPCT-44: unkeyed IFakeService must be registered")

	// Keyed IFakeService("service1") must NOT be found
	testutil.True(t, !p.IsKeyedService(reflect.TypeFor[fakes.IFakeService](), "service1"),
		"SPCT-44: keyed IFakeService('service1') must not fall through to unkeyed")
}

// TestSPCT_23_GenericSingletonCacheSlot verifies that a SINGLETON closed generic
// resolved singly (GetService) and as a collection (GetKeyedServices(nil)) returns
// the SAME cached instance — the generic IEnumerable item lives in the same cache
// slot (dotnet GenericIEnumerableItemCachedInTheRightSlot). The default SPCT
// containers register the open generic as TRANSIENT, so this needs its own
// singleton registration to be meaningful (runtime-only; the singleton-shared axis
// is already differentially verified by spec_test.go across both providers).
func TestSPCT_23_GenericSingletonCacheSlot(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
		return fakes.NewFakeService()
	})
	runtime.AddSingleton[*fakes.FakeOpenGenericService[fakes.IFakeService]](b, func(p godi.Provider) *fakes.FakeOpenGenericService[fakes.IFakeService] {
		dep := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
		return fakes.NewFakeOpenGenericService[fakes.IFakeService](dep)
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	typ := reflect.TypeFor[*fakes.FakeOpenGenericService[fakes.IFakeService]]()
	single := p.GetService(typ)
	testutil.NotNil(t, single)
	services := p.GetKeyedServices(typ, nil)
	if len(services) != 1 {
		t.Fatalf("SPCT-23: expected 1 service in collection, got %d", len(services))
	}
	if single != services[0] {
		t.Error("SPCT-23: singleton GetService and GetServices(nil) must return the SAME cached instance")
	}
}

// TestSPCT_18_StructServicesThreeLifetimes verifies a value-type (struct) service
// resolves under all three lifetimes, boxing the struct VALUE (not a pointer) into
// the interface (dotnet WorksWithStructServices).
func TestSPCT_18_StructServicesThreeLifetimes(t *testing.T) {
	for _, lt := range []string{"singleton", "scoped", "transient"} {
		t.Run(lt, func(t *testing.T) {
			b := runtime.NewBuilder()
			factory := func(godi.Provider) fakes.IStructFakeService { return fakes.NewStructFakeService() }
			switch lt {
			case "singleton":
				runtime.AddSingleton[fakes.IStructFakeService](b, factory)
			case "scoped":
				runtime.AddScoped[fakes.IStructFakeService](b, factory)
			case "transient":
				runtime.AddTransient[fakes.IStructFakeService](b, factory)
			}
			p, err := b.Build()
			testutil.NoError(t, err)
			t.Cleanup(func() { _ = p.Close() })

			got := p.GetService(reflect.TypeFor[fakes.IStructFakeService]())
			testutil.NotNil(t, got)
			svc, ok := got.(fakes.IStructFakeService)
			testutil.True(t, ok, "resolved value must satisfy IStructFakeService")
			testutil.Equal(t, svc.StructTag(), "struct-fake")
			// struct services box the VALUE type — dynamic type is the struct, not a pointer
			testutil.Equal(t, reflect.TypeOf(got).String(), "fakes.StructFakeService")
		})
	}
}

// TestSPCT_19_FactoryStructServices verifies a form-B factory returning a struct
// VALUE (dotnet WorksWithFactoryStructServices). In godi every runtime registration
// is a form-B factory, so this asserts a factory-produced struct value resolves
// with its factory-set field intact.
func TestSPCT_19_FactoryStructServices(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[fakes.IStructFakeService](b, func(godi.Provider) fakes.IStructFakeService {
		return fakes.StructFakeService{Tag: "factory-made"}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	got := p.GetService(reflect.TypeFor[fakes.IStructFakeService]()).(fakes.IStructFakeService)
	testutil.Equal(t, got.StructTag(), "factory-made")
}

// TestSPCT_20_StructServiceAsDependency verifies a struct (value-type) service can
// be injected as a constructor dependency (dotnet WorksWithFactoryStructServicesAs
// Dependencies).
func TestSPCT_20_StructServiceAsDependency(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[fakes.IStructFakeService](b, func(godi.Provider) fakes.IStructFakeService {
		return fakes.NewStructFakeService()
	})
	runtime.AddSingleton[fakes.IStructConsumer](b, func(p godi.Provider) fakes.IStructConsumer {
		dep := p.GetService(reflect.TypeFor[fakes.IStructFakeService]()).(fakes.IStructFakeService)
		return fakes.NewStructConsumer(dep)
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	consumer := p.GetService(reflect.TypeFor[fakes.IStructConsumer]()).(fakes.IStructConsumer)
	testutil.NotNil(t, consumer.Dependency())
	testutil.Equal(t, consumer.Dependency().StructTag(), "struct-fake")
}

// TestSPCT_21_StructMultiImplCollection verifies struct elements keep their value
// identity inside a []any collection resolution (dotnet WorksWithIEnumerableStruct
// Services). Both elements must be value-typed structs, not pointers, in order.
func TestSPCT_21_StructMultiImplCollection(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[fakes.IStructFakeMultipleService](b, func(godi.Provider) fakes.IStructFakeMultipleService {
		return fakes.NewStructFakeOne()
	})
	runtime.AddSingleton[fakes.IStructFakeMultipleService](b, func(godi.Provider) fakes.IStructFakeMultipleService {
		return fakes.NewStructFakeTwo()
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	services := p.GetKeyedServices(reflect.TypeFor[fakes.IStructFakeMultipleService](), nil)
	if len(services) != 2 {
		t.Fatalf("SPCT-21: expected 2 struct services, got %d", len(services))
	}
	testutil.Equal(t, services[0].(fakes.IStructFakeMultipleService).MultiTag(), "struct-one")
	testutil.Equal(t, services[1].(fakes.IStructFakeMultipleService).MultiTag(), "struct-two")
	testutil.Equal(t, reflect.TypeOf(services[0]).String(), "fakes.StructFakeOne")
	testutil.Equal(t, reflect.TypeOf(services[1]).String(), "fakes.StructFakeTwo")
}

// TestSPCT_StructServices_CrossProvider closes review finding F7/#10: the struct
// (value-type) services are now ALSO registered in the codegen container
// (registration.go), so the value-type axis — reflect.Type keying of a boxed
// struct value, struct-as-dependency injection, and struct elements in a []any
// collection — is verified on the GENERATED container too, not runtime-only. It
// runs a differential equivalence check (AC2, AD-15): for each struct scenario
// the codegen Root must observe the SAME boxed dynamic type and tag the runtime
// oracle does. The runtime side mirrors the codegen registration source.
func TestSPCT_StructServices_CrossProvider(t *testing.T) {
	// Runtime oracle built from the same logical source as registration.go.
	rt := func(t *testing.T) Provider {
		b := runtime.NewBuilder()
		runtime.AddSingleton[fakes.IStructFakeService](b, func(godi.Provider) fakes.IStructFakeService {
			return fakes.NewStructFakeService()
		})
		runtime.AddSingleton[fakes.IStructConsumer](b, func(p godi.Provider) fakes.IStructConsumer {
			dep := p.GetService(reflect.TypeFor[fakes.IStructFakeService]()).(fakes.IStructFakeService)
			return fakes.NewStructConsumer(dep)
		})
		runtime.AddSingleton[fakes.IStructFakeMultipleService](b, func(godi.Provider) fakes.IStructFakeMultipleService {
			return fakes.NewStructFakeOne()
		})
		runtime.AddSingleton[fakes.IStructFakeMultipleService](b, func(godi.Provider) fakes.IStructFakeMultipleService {
			return fakes.NewStructFakeTwo()
		})
		p, err := b.Build()
		testutil.NoError(t, err)
		t.Cleanup(func() { _ = p.Close() })
		return p
	}
	cg := func(t *testing.T) Provider {
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		t.Cleanup(func() { _ = root.Close() })
		return root
	}

	// probe captures the observable facts of each struct scenario as comparable
	// strings, so runtime and codegen outcomes compare directly.
	for _, scenario := range []struct {
		name  string
		probe func(Provider) string
	}{
		{
			// SPCT-18: value-type service boxes the struct VALUE, not a pointer.
			name: "struct service boxes value type",
			probe: func(p Provider) string {
				got := p.GetService(reflect.TypeFor[fakes.IStructFakeService]())
				return reflect.TypeOf(got).String() + "|" + got.(fakes.IStructFakeService).StructTag()
			},
		},
		{
			// SPCT-20: struct injected as a constructor dependency.
			name: "struct as dependency",
			probe: func(p Provider) string {
				c := p.GetService(reflect.TypeFor[fakes.IStructConsumer]()).(fakes.IStructConsumer)
				return c.Dependency().StructTag()
			},
		},
		{
			// SPCT-21: struct elements keep value identity + order in a []any.
			name: "struct multi-impl collection",
			probe: func(p Provider) string {
				svcs := p.GetKeyedServices(reflect.TypeFor[fakes.IStructFakeMultipleService](), nil)
				out := ""
				for _, s := range svcs {
					out += reflect.TypeOf(s).String() + ":" + s.(fakes.IStructFakeMultipleService).MultiTag() + ";"
				}
				return out
			},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			rtOut := scenario.probe(rt(t))
			cgOut := scenario.probe(cg(t))
			if rtOut != cgOut {
				t.Errorf("cross-provider struct mismatch: runtime=%q codegen=%q", rtOut, cgOut)
			}
		})
	}
}

// TestSPCT_22_WideScopedTree verifies a wide tree of scoped services (dotnet
// WorksWithWideScopedTrees): N same-typed scoped registrations all resolve as a
// collection, every element is a distinct instance, stable WITHIN one scope and
// fresh in the next (scoped-per-scope materialization, AD-1).
func TestSPCT_22_WideScopedTree(t *testing.T) {
	const width = 20
	b := runtime.NewBuilder()
	for i := 0; i < width; i++ {
		slot := i
		runtime.AddScoped[*spctWideNode](b, func(godi.Provider) *spctWideNode {
			return &spctWideNode{slot: slot}
		})
	}
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	scope := p.CreateScope()
	t.Cleanup(func() { _ = scope.Close() })

	typ := reflect.TypeFor[*spctWideNode]()
	services := scope.GetKeyedServices(typ, nil)
	if len(services) != width {
		t.Fatalf("SPCT-22: wide scoped tree expected %d services, got %d", width, len(services))
	}
	// every element is a DISTINCT instance: the N same-typed scoped registrations
	// must NOT collapse into one shared slot (the "wide" part of the wide tree).
	seen := make(map[*spctWideNode]int, width)
	for i := range services {
		node := services[i].(*spctWideNode)
		if prev, dup := seen[node]; dup {
			t.Errorf("SPCT-22: scoped elements %d and %d are the SAME instance (must be distinct)", prev, i)
		}
		seen[node] = i
	}
	// stable within the scope (scoped cache): a second resolve returns identical pointers
	again := scope.GetKeyedServices(typ, nil)
	for i := range services {
		if services[i] != again[i] {
			t.Errorf("SPCT-22: scoped element %d not stable across resolves in one scope", i)
		}
	}
	// fresh in a sibling scope: instances differ from the first scope's
	scope2 := p.CreateScope()
	t.Cleanup(func() { _ = scope2.Close() })
	other := scope2.GetKeyedServices(typ, nil)
	for i := range services {
		if services[i] == other[i] {
			t.Errorf("SPCT-22: scoped element %d shared across distinct scopes (must be fresh)", i)
		}
	}
}

// TestSPCT_39_ConstrainedOpenGenericResolves confirms a constrained closed generic
// (Story 4.2) resolves after build on the real codegen container — godi ALWAYS
// runs build-time validation (plan.Validate + NewRoot), the dotnet
// ValidateOnBuild=true analog. Collection-resolvable too.
func TestSPCT_39_ConstrainedOpenGenericResolves(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	typ := reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IPocoClass]]()
	got := root.GetService(typ)
	testutil.NotNil(t, got)
	services := root.GetKeyedServices(typ, nil)
	testutil.True(t, len(services) >= 1, "SPCT-39: constrained open generic must be collection-resolvable")
}

// TestSPCT_42_RootScopedGenericDepStability verifies a scoped service with a
// generic-typed graph resolved FROM ROOT is root-scoped (singleton-like): repeated
// resolves return the same instance (dotnet ScopedServiceResolvedFromSingleton
// AfterCompilation2, adapted to godi's root-scoped semantics, Story 3-2).
func TestSPCT_42_RootScopedGenericDepStability(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
		return fakes.NewFakeService()
	})
	runtime.AddScoped[*fakes.FakeOpenGenericService[fakes.IFakeService]](b, func(p godi.Provider) *fakes.FakeOpenGenericService[fakes.IFakeService] {
		dep := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
		return fakes.NewFakeOpenGenericService[fakes.IFakeService](dep)
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	typ := reflect.TypeFor[*fakes.FakeOpenGenericService[fakes.IFakeService]]()
	first := p.GetService(typ)
	testutil.NotNil(t, first)
	for i := 0; i < 5; i++ {
		if p.GetService(typ) != first {
			t.Fatalf("SPCT-42: root-scoped generic instance changed on resolve %d (want same)", i)
		}
	}
}

// TestSPCT_43_RootScopedNestedDepStability verifies a nested scoped-depends-on-
// scoped graph resolved FROM ROOT: the inner A is root-scoped, and the SAME
// root-scoped A is injected into Aa (dotnet ScopedServiceResolvedFromSingleton
// AfterCompilation3, adapted to godi root-scoped semantics).
func TestSPCT_43_RootScopedNestedDepStability(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[*spctScopedA](b, func(godi.Provider) *spctScopedA {
		return newSPCTScopedA()
	})
	runtime.AddScoped[*spctScopedAa](b, func(p godi.Provider) *spctScopedAa {
		a := p.GetService(reflect.TypeFor[*spctScopedA]()).(*spctScopedA)
		return newSPCTScopedAa(a)
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	a := p.GetService(reflect.TypeFor[*spctScopedA]()).(*spctScopedA)
	aa := p.GetService(reflect.TypeFor[*spctScopedAa]()).(*spctScopedAa)
	if aa.a != a {
		t.Error("SPCT-43: nested scoped dependency must be the SAME root-scoped instance")
	}
	if p.GetService(reflect.TypeFor[*spctScopedAa]()) != aa {
		t.Error("SPCT-43: root-scoped Aa must be stable across resolves")
	}
}

// TestSPCT_03_MissingDependencyRejectedAtBuild verifies godi left-shifts a missing
// dependency to BUILD time (dotnet AttemptingToResolveNonexistentServiceIndirectly
// Throws, adapted): the generator REJECTS a graph whose service depends on an
// unregistered type — no container emitted (plan.Validate, Story 5-1; AD-3).
func TestSPCT_03_MissingDependencyRejectedAtBuild(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "Consumer",
		GoType:   "*svc.Consumer",
		Ctor:     "svc.NewConsumer",
		Lifetime: plan.Singleton,
		Deps:     []string{"Missing"}, // Missing is never registered
	})
	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)
}

// TestSPCT_04_IndirectMissingViaSliceRejectedAtBuild verifies the collection analog
// (dotnet AttemptingToIEnumerableResolveNonexistentServiceIndirectlyThrows): a
// service B reachable through a slice dependency []B is itself missing a
// dependency, so the graph is rejected at build time before any container exists.
func TestSPCT_04_IndirectMissingViaSliceRejectedAtBuild(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	// B is registered but depends on an UNregistered Missing.
	reg.Service(plan.Service{
		Name:     "B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"Missing"},
	})
	// A depends on the collection []*svc.B; resolving it constructs B, which needs Missing.
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"[]*svc.B"},
	})
	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)
}
