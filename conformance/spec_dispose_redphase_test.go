// ATDD red-phase scaffold for Story 2.3 (AC1/AC2/AC3) — conformance suite:
// dual-provider dispose spec cases + cross-provider LIFO differential
// equivalence + singleton dispose state.
//
// This file extends the conformance suite with dispose-specific SpecCases that
// run against BOTH providers (runtime oracle + codegen B2). AD-15 mandates
// that the release LIFO axis be covered in codegen with ≥1 scenario + cross-
// provider differential equivalence.
//
// New fakes needed (Task 5): IFakeDisposableSingleton, IFakeDisposableScoped,
// IFakeDisposableTransient, FakeDisposable, DisposeRecorder — all in
// conformance/fakes/. This file will NOT compile until they exist.
//
// Test standard R4: pointer identity within one provider only; across
// providers use provider-stable observables (label sequences, Disposed bools).
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-3-scope-dispose-lifo.md.
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

// rtFactoryDispose builds the runtime provider with disposable services (all
// three lifetimes). Each disposable fake appends to a shared DisposeRecorder
// on Close — the provider-stable observable for LIFO verification (D-7).
var rtFactoryDispose = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		// Disposable singleton
		runtime.AddSingleton[fakes.IFakeDisposableSingleton](b, func(godi.Provider) fakes.IFakeDisposableSingleton {
			return fakes.NewFakeDisposableSingleton()
		})
		// Disposable scoped
		runtime.AddScoped[fakes.IFakeDisposableScoped](b, func(godi.Provider) fakes.IFakeDisposableScoped {
			return fakes.NewFakeDisposableScoped()
		})
		// Disposable transient
		runtime.AddTransient[fakes.IFakeDisposableTransient](b, func(godi.Provider) fakes.IFakeDisposableTransient {
			return fakes.NewFakeDisposableTransient()
		})
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryDispose builds the codegen provider from the regenerated
// container_gen.go with disposable service registrations.
var cgFactoryDispose = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// disposeSpecCases are SpecCases specific to dispose/release semantics. Each
// case runs against BOTH providers (Task 5). Provider-stable observables only
// — never pointer identity across providers (R4).
var disposeSpecCases = []SpecCase{
	// --- 2.3-SPEC-001 (AC2, P0) — DisposingScopeDisposesService (core) ---------
	{
		Name: "dispose: scope.Close releases scoped (DisposingScopeDisposesService)",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()

			svc := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableScoped]())
			testutil.NotNil(t, svc)

			disposable := svc.(interface{ IsDisposed() bool })
			testutil.True(t, !disposable.IsDisposed(),
				"scoped must NOT be disposed before scope.Close")

			testutil.NoError(t, scope.Close())
			testutil.True(t, disposable.IsDisposed(),
				"scoped must be disposed after scope.Close")
		},
	},
	// --- 2.3-SPEC-002 (AC2, P0) — DisposingScopeDisposesService (transient) ----
	{
		Name: "dispose: scope.Close releases transient disposables",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()

			t0 := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableTransient]())
			t1 := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableTransient]())

			testutil.True(t, t0 != t1, "transients must be different instances (AD-14)")

			d0 := t0.(interface{ IsDisposed() bool })
			d1 := t1.(interface{ IsDisposed() bool })
			testutil.True(t, !d0.IsDisposed(), "transient0 not disposed before scope.Close")
			testutil.True(t, !d1.IsDisposed(), "transient1 not disposed before scope.Close")

			testutil.NoError(t, scope.Close())
			testutil.True(t, d0.IsDisposed(), "transient0 must be disposed after scope.Close")
			testutil.True(t, d1.IsDisposed(), "transient1 must be disposed after scope.Close")
		},
	},
	// --- 2.3-SPEC-003 (AC3, P0) — singleton NOT released on scope.Close --------
	{
		Name: "dispose: singleton NOT released on scope.Close (AD-12 iron proof)",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()

			singleton := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableSingleton]())
			scoped := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableScoped]())

			testutil.NotNil(t, singleton)
			testutil.NotNil(t, scoped)

			testutil.NoError(t, scope.Close())

			sSingle := singleton.(interface{ IsDisposed() bool })
			sScoped := scoped.(interface{ IsDisposed() bool })

			testutil.True(t, sScoped.IsDisposed(),
				"scoped must be disposed after scope.Close")
			testutil.True(t, !sSingle.IsDisposed(),
				"singleton must NOT be disposed on scope.Close — only root (AD-12)")

			// Root close releases singleton
			testutil.NoError(t, p.Close())
			testutil.True(t, sSingle.IsDisposed(),
				"singleton must be disposed after root.Close")
		},
	},
	// --- 2.3-SPEC-004 (AC1, P1) — ServiceProviderIsDisposable -----------------
	{
		Name: "dispose: provider satisfies Close contract (ServiceProviderIsDisposable)",
		Run: func(t *testing.T, p Provider) {
			// Close is part of the Provider interface (AD-16 seven-method face).
			// Calling Close must not panic and must return nil or an error.
			err := p.Close()
			testutil.NoError(t, err)
			// Idempotent
			err = p.Close()
			testutil.NoError(t, err)
		},
	},
}

// --- 2.3-SPEC-RUN (AC1/AC2/AC3, P0) -----------------------------------------

func TestConformanceDispose_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryDispose, disposeSpecCases)
}

func TestConformanceDispose_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryDispose, disposeSpecCases)
}

// --- 2.3-SPEC-005 (AC2/AC3, P0) — Differential Equivalence ------------------

// TestConformanceDispose_DifferentialEquivalence: AD-15 cross-provider
// differential equivalence for the release axis. Uses provider-stable
// observables — never pointer identity across providers (R4).
//
// Observables:
//   - Disposed state (bool) after scope.Close for each lifetime
//   - IsService consistency for disposable types
func TestConformanceDispose_DifferentialEquivalence(t *testing.T) {
	// IsService must agree for all disposable types
	AssertEquivalent(t, rtFactoryDispose, cgFactoryDispose, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeDisposableSingleton]())
	})
	AssertEquivalent(t, rtFactoryDispose, cgFactoryDispose, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeDisposableScoped]())
	})
	AssertEquivalent(t, rtFactoryDispose, cgFactoryDispose, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[fakes.IFakeDisposableTransient]())
	})

	// Dynamic type name equivalence
	AssertEquivalent(t, rtFactoryDispose, cgFactoryDispose, func(p Provider) string {
		return dynTypeName(p.GetService(reflect.TypeFor[fakes.IFakeDisposableSingleton]()))
	})
	AssertEquivalent(t, rtFactoryDispose, cgFactoryDispose, func(p Provider) string {
		return dynTypeName(p.GetService(reflect.TypeFor[fakes.IFakeDisposableScoped]()))
	})

	// Dispose semantics equivalence: after scope.Close, scoped=disposed,
	// singleton=NOT disposed (provider-stable bool observable)
	AssertEquivalent(t, rtFactoryDispose, cgFactoryDispose, func(p Provider) bool {
		scope := p.CreateScope()
		scoped := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableScoped]())
		_ = scope.Close()
		return scoped.(interface{ IsDisposed() bool }).IsDisposed()
	})
	AssertEquivalent(t, rtFactoryDispose, cgFactoryDispose, func(p Provider) bool {
		scope := p.CreateScope()
		singleton := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableSingleton]())
		_ = scope.Close()
		return singleton.(interface{ IsDisposed() bool }).IsDisposed()
	})
}

// --- 2.3-SPEC-006 (AC2, P0) — LIFO order differential equivalence -----------

// TestConformanceDispose_LIFO_DifferentialEquivalence: LIFO release order must
// be identical across both providers. Uses label sequence as provider-stable
// observable (D-7: DisposeRecorder collects labels, not pointers).
//
// NOTE: This test requires DisposeRecorder to be injectable or observable via
// the disposable fakes. The exact mechanism depends on Task 5's fake design.
// The test skeleton below captures the assertion contract — the dev fills in
// the recorder wiring when fakes are created.
func TestConformanceDispose_LIFO_DifferentialEquivalence(t *testing.T) {
	var sequences [2][]string
	for i, factory := range []ProviderFactory{rtFactoryDispose, cgFactoryDispose} {
		p := factory.Build(t)

		scope := p.CreateScope()

		recorder := fakes.NewDisposeRecorder()

		// Creation order: scoped, transient-0, transient-1
		scoped := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableScoped]())
		scoped.(interface {
			SetRecorder(*fakes.DisposeRecorder, string)
		}).SetRecorder(recorder, "scoped")

		t0 := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableTransient]())
		t0.(interface {
			SetRecorder(*fakes.DisposeRecorder, string)
		}).SetRecorder(recorder, "transient-0")

		t1 := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableTransient]())
		t1.(interface {
			SetRecorder(*fakes.DisposeRecorder, string)
		}).SetRecorder(recorder, "transient-1")

		testutil.NoError(t, scope.Close())

		// LIFO: last created released first (D-7, AD-12)
		labels := recorder.Labels()
		testutil.Equal(t, len(labels), 3)
		testutil.Equal(t, labels[0], "transient-1")
		testutil.Equal(t, labels[1], "transient-0")
		testutil.Equal(t, labels[2], "scoped")

		sequences[i] = labels
		_ = p.Close()
	}
	// Cross-provider differential equivalence: both providers' label
	// sequences must match (AD-15, D-7)
	testutil.DeepEqual(t, sequences[0], sequences[1])
}
