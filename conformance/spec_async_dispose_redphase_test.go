// ATDD red-phase scaffold for Story 6.3 (AC2/AC3/AC5/AC6) — conformance
// suite: dual-provider async release SpecCases + cross-provider differential
// equivalence + AsyncScope integration through a REAL container. AD-15
// mandates that the "async release LIFO" axis be covered in codegen with
// ≥1 scenario + cross-provider differential equivalence, and Path γ (strict,
// not just differential) — see conformance/codegen/gen_async_dispose_redphase_test.go.
//
// New fakes needed (Task 6): fakes.FakeAsyncOnlyDisposable (CloseAsync only),
// fakes.FakeSyncAsyncDisposable (Close + CloseAsync, prefer-async witness),
// fakes.AsyncDisposeRecorder — all in conformance/fakes/. Reuses Story 2.3's
// fakes.FakeDisposableSingleton/Scoped (sync-only) for the "sync-only" leg
// of the D-4 matrix (D-notes: reuse, do not modify). This file will NOT
// compile until the new fakes exist.
//
// Test standard R4: pointer identity within one provider only; across
// providers use provider-stable observables (label sequences, Disposed
// bools).
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-6-3-async-scope-dispose.md.
//go:build atdd_redphase

package conformance

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// asyncCloser is the structural opt-in surface CloseAsync lives behind
// (D-2) — root/scope implement it without it being part of godi.Provider.
type asyncCloser interface {
	CloseAsync(ctx context.Context) error
}

// rtFactoryAsyncDispose builds the runtime provider with a singleton
// (sync-only, reused from Story 2.3), a scoped async-only, and a scoped
// both-disposable service — exercising all three D-4 matrix rows.
var rtFactoryAsyncDispose = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		runtime.AddSingleton[fakes.IFakeDisposableSingleton](b, func(godi.Provider) fakes.IFakeDisposableSingleton {
			return fakes.NewFakeDisposableSingleton()
		})
		runtime.AddScoped[*fakes.FakeAsyncOnlyDisposable](b, func(godi.Provider) *fakes.FakeAsyncOnlyDisposable {
			return fakes.NewFakeAsyncOnlyDisposable()
		})
		runtime.AddScoped[*fakes.FakeSyncAsyncDisposable](b, func(godi.Provider) *fakes.FakeSyncAsyncDisposable {
			return fakes.NewFakeSyncAsyncDisposable()
		})
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryAsyncDispose builds the codegen provider from the regenerated
// container_gen.go with the same async service registrations (Task 6 wires
// these into conformance/codegen/registration.go).
var cgFactoryAsyncDispose = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// asyncDisposeSpecCases are SpecCases specific to async release semantics.
// Each runs against BOTH providers (AD-15).
var asyncDisposeSpecCases = []SpecCase{
	// --- 6.3-SPEC-001 (AC2, P0) — async-only scoped released via CloseAsync ---
	{
		Name: "async dispose: scope.CloseAsync releases async-only scoped",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()
			svc := scope.GetService(reflect.TypeFor[*fakes.FakeAsyncOnlyDisposable]())
			testutil.NotNil(t, svc)

			disposable := svc.(interface{ IsDisposed() bool })
			testutil.True(t, !disposable.IsDisposed(), "must NOT be disposed before CloseAsync")

			ac, ok := scope.(asyncCloser)
			testutil.True(t, ok, "scope must implement CloseAsync (structural opt-in, D-2)")
			testutil.NoError(t, ac.CloseAsync(context.Background()))
			testutil.True(t, disposable.IsDisposed(), "must be disposed after scope.CloseAsync")
		},
	},
	// --- 6.3-SPEC-002 (AC2, P0) — prefer-async on both-implementor ------------
	{
		Name: "async dispose: scope.CloseAsync prefers CloseAsync on both-implementor",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()
			svc := scope.GetService(reflect.TypeFor[*fakes.FakeSyncAsyncDisposable]())
			testutil.NotNil(t, svc)

			ac := scope.(asyncCloser)
			testutil.NoError(t, ac.CloseAsync(context.Background()))

			preferred := svc.(interface{ ClosedVia() string })
			testutil.Equal(t, preferred.ClosedVia(), "async")
		},
	},
	// --- 6.3-SPEC-003 (AC3, P0) — sync Close errors on async-only ------------
	{
		Name: "async dispose: sync scope.Close errors on async-only service (AC3)",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()
			_ = scope.GetService(reflect.TypeFor[*fakes.FakeAsyncOnlyDisposable]())

			err := scope.Close()
			testutil.Error(t, err)
			testutil.True(t, strings.Contains(err.Error(), "CloseAsync"),
				"AC3: sync Close on an async-only service must name CloseAsync in the error")
		},
	},
	// --- 6.3-SPEC-004 (AC5, P0) — singleton NOT released by scope.CloseAsync --
	{
		Name: "async dispose: singleton NOT released on scope.CloseAsync (AD-12 async parity)",
		Run: func(t *testing.T, p Provider) {
			scope := p.CreateScope()
			singleton := scope.GetService(reflect.TypeFor[fakes.IFakeDisposableSingleton]())
			testutil.NotNil(t, singleton)

			ac := scope.(asyncCloser)
			testutil.NoError(t, ac.CloseAsync(context.Background()))

			sSingle := singleton.(interface{ IsDisposed() bool })
			testutil.True(t, !sSingle.IsDisposed(),
				"singleton must NOT be disposed on scope.CloseAsync — only root (AD-12)")

			rac := p.(asyncCloser)
			testutil.NoError(t, rac.CloseAsync(context.Background()))
			testutil.True(t, sSingle.IsDisposed(), "singleton must be disposed after root.CloseAsync")
		},
	},
}

func TestConformanceAsyncDispose_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryAsyncDispose, asyncDisposeSpecCases)
}

func TestConformanceAsyncDispose_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryAsyncDispose, asyncDisposeSpecCases)
}

// --- 6.3-SPEC-005 (AC6, P0) — Differential Equivalence ----------------------

// TestConformanceAsyncDispose_DifferentialEquivalence: AD-15 cross-provider
// differential equivalence for the async release axis. Uses provider-stable
// observables — never pointer identity across providers (R4).
func TestConformanceAsyncDispose_DifferentialEquivalence(t *testing.T) {
	AssertEquivalent(t, rtFactoryAsyncDispose, cgFactoryAsyncDispose, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[*fakes.FakeAsyncOnlyDisposable]())
	})
	AssertEquivalent(t, rtFactoryAsyncDispose, cgFactoryAsyncDispose, func(p Provider) bool {
		return p.IsService(reflect.TypeFor[*fakes.FakeSyncAsyncDisposable]())
	})

	// Async-only disposed-state equivalence after scope.CloseAsync.
	AssertEquivalent(t, rtFactoryAsyncDispose, cgFactoryAsyncDispose, func(p Provider) bool {
		scope := p.CreateScope()
		svc := scope.GetService(reflect.TypeFor[*fakes.FakeAsyncOnlyDisposable]())
		_ = scope.(asyncCloser).CloseAsync(context.Background())
		return svc.(interface{ IsDisposed() bool }).IsDisposed()
	})

	// Prefer-async equivalence: both-implementor releases via CloseAsync on
	// both providers.
	AssertEquivalent(t, rtFactoryAsyncDispose, cgFactoryAsyncDispose, func(p Provider) string {
		scope := p.CreateScope()
		svc := scope.GetService(reflect.TypeFor[*fakes.FakeSyncAsyncDisposable]())
		_ = scope.(asyncCloser).CloseAsync(context.Background())
		return svc.(interface{ ClosedVia() string }).ClosedVia()
	})
}

// --- 6.3-SPEC-006 (AC2, P0) — LIFO order differential equivalence -----------

// TestConformanceAsyncDispose_LIFO_DifferentialEquivalence: async LIFO
// release order must be identical across both providers. Uses an
// AsyncDisposeRecorder label sequence as the provider-stable observable
// (D-7 pattern, extended for async).
func TestConformanceAsyncDispose_LIFO_DifferentialEquivalence(t *testing.T) {
	var sequences [2][]string
	for i, factory := range []ProviderFactory{rtFactoryAsyncDispose, cgFactoryAsyncDispose} {
		p := factory.Build(t)

		scope := p.CreateScope()
		recorder := fakes.NewAsyncDisposeRecorder()

		// Creation order: async-only, then both.
		asyncOnly := scope.GetService(reflect.TypeFor[*fakes.FakeAsyncOnlyDisposable]())
		asyncOnly.(interface {
			SetRecorder(*fakes.AsyncDisposeRecorder, string)
		}).SetRecorder(recorder, "async-only")

		both := scope.GetService(reflect.TypeFor[*fakes.FakeSyncAsyncDisposable]())
		both.(interface {
			SetRecorder(*fakes.AsyncDisposeRecorder, string)
		}).SetRecorder(recorder, "both")

		testutil.NoError(t, scope.(asyncCloser).CloseAsync(context.Background()))

		// LIFO: last created released first.
		labels := recorder.Labels()
		testutil.Equal(t, len(labels), 2)
		testutil.Equal(t, labels[0], "both")
		testutil.Equal(t, labels[1], "async-only")

		sequences[i] = labels
		_ = p.Close()
	}
	testutil.DeepEqual(t, sequences[0], sequences[1])
}

// --- 6.3-SPEC-007 (AC4/AC6, P0) — AsyncScope wrapper over a REAL container --

// TestConformanceAsyncDispose_AsyncScopeWrapper_ReleasesRealScope: godi.
// CreateAsyncScope wrapping a real container scope must release its
// disposables through CloseAsync — ties Task 1's wrapper to a real provider.
func TestConformanceAsyncDispose_AsyncScopeWrapper_ReleasesRealScope(t *testing.T) {
	for _, factory := range []ProviderFactory{rtFactoryAsyncDispose, cgFactoryAsyncDispose} {
		p := factory.Build(t)

		asyncScope := godi.CreateAsyncScope(p)
		svc := asyncScope.Provider().GetService(reflect.TypeFor[*fakes.FakeAsyncOnlyDisposable]())
		testutil.NotNil(t, svc)

		testutil.NoError(t, asyncScope.CloseAsync(context.Background()))
		testutil.True(t, svc.(interface{ IsDisposed() bool }).IsDisposed(),
			"AsyncScope.CloseAsync must release the wrapped real scope's disposables")

		_ = p.Close()
	}
}
