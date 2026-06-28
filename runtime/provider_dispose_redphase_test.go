//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.3 (AC2/AC3) — runtime provider dispose:
// root/scope track, Close releases disposables, singleton not released on
// scope.Close, LIFO order through provider, Close idempotent with disposables.
//
// These tests use the EXISTING runtime.NewBuilder / AddSingleton / AddScoped /
// AddTransient API — the assertions test Close/dispose behavior that currently
// is a no-op (Close returns nil without releasing). The closableLogger fixture
// from provider_test.go is reused (same package: runtime_test).
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-3-scope-dispose-lifo.md.
package runtime_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// --- disposable test helpers (local, reuse closableLogger from provider_test.go) ---

// disposeTracker is a provider-local DisposeRecorder that collects Close()
// call labels in order — the provider-stable observable for LIFO verification
// and cross-provider differential equivalence (D-7).
type disposeTracker struct {
	labels []string
}

// closableSvc is a disposable service fixture. On Close it appends its label to
// the tracker and marks itself disposed. Unlike closableLogger (already in
// provider_test.go), this has a label for LIFO order verification.
type closableSvc struct {
	label    string
	tracker  *disposeTracker
	disposed bool
	_        int // padding for distinct pointer identity (2.2 zero-size lesson)
}

func (c *closableSvc) Close() error {
	c.disposed = true
	if c.tracker != nil {
		c.tracker.labels = append(c.tracker.labels, c.label)
	}
	return nil
}

// closable interfaces — sealed to this package, one per lifetime. Mirrors the
// three sealed interfaces Task 5 creates in conformance/fakes.
type iClosableSingleton interface{ closableSingleton() }
type iClosableScoped interface{ closableScoped() }
type iClosableTransient interface{ closableTransient() }

func (*closableSvc) closableSingleton() {}
func (*closableSvc) closableScoped()    {}
func (*closableSvc) closableTransient() {}

// --- 2.3-PROV-001 (AC2, P0) -------------------------------------------------

// TestRT_ScopeClose_ReleasesScoped: closing a scope must release scoped
// disposable services resolved within it (DisposingScopeDisposesService core).
func TestRT_ScopeClose_ReleasesScoped(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddScoped[iClosableScoped](b, func(godi.Provider) iClosableScoped {
		return &closableSvc{label: "scoped", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	scope := p.CreateScope()

	svc := scope.GetService(reflect.TypeFor[iClosableScoped]()).(*closableSvc)
	testutil.True(t, !svc.disposed, "scoped must NOT be disposed before scope.Close")

	testutil.NoError(t, scope.Close())
	testutil.True(t, svc.disposed, "scoped must be disposed after scope.Close")
}

// --- 2.3-PROV-002 (AC2, P0) -------------------------------------------------

// TestRT_ScopeClose_ReleasesTransient: closing a scope must release transient
// disposable services that were resolved within that scope. Each transient is
// tracked (AD-12:逐次 track) even though it's never cached (AD-14).
func TestRT_ScopeClose_ReleasesTransient(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddTransient[iClosableTransient](b, func(godi.Provider) iClosableTransient {
		return &closableSvc{label: "transient", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	scope := p.CreateScope()

	t1 := scope.GetService(reflect.TypeFor[iClosableTransient]()).(*closableSvc)
	t2 := scope.GetService(reflect.TypeFor[iClosableTransient]()).(*closableSvc)
	testutil.True(t, t1 != t2, "transients must be different instances (AD-14)")
	testutil.True(t, !t1.disposed, "transient1 not disposed before scope.Close")
	testutil.True(t, !t2.disposed, "transient2 not disposed before scope.Close")

	testutil.NoError(t, scope.Close())
	testutil.True(t, t1.disposed, "transient1 must be disposed after scope.Close")
	testutil.True(t, t2.disposed, "transient2 must be disposed after scope.Close")
}

// --- 2.3-PROV-003 (AC2, P0) -------------------------------------------------

// TestRT_RootClose_ReleasesSingletonAndRootTransient: closing root must
// release singleton disposable services and transients resolved from root.
func TestRT_RootClose_ReleasesSingletonAndRootTransient(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddSingleton[iClosableSingleton](b, func(godi.Provider) iClosableSingleton {
		return &closableSvc{label: "singleton", tracker: tracker}
	})
	runtime.AddTransient[iClosableTransient](b, func(godi.Provider) iClosableTransient {
		return &closableSvc{label: "transient", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)

	singleton := p.GetService(reflect.TypeFor[iClosableSingleton]()).(*closableSvc)
	rootTransient := p.GetService(reflect.TypeFor[iClosableTransient]()).(*closableSvc)

	testutil.True(t, !singleton.disposed, "singleton not disposed before root.Close")
	testutil.True(t, !rootTransient.disposed, "root transient not disposed before root.Close")

	testutil.NoError(t, p.Close())
	testutil.True(t, singleton.disposed, "singleton must be disposed after root.Close")
	testutil.True(t, rootTransient.disposed, "root transient must be disposed after root.Close")
}

// --- 2.3-PROV-004 (AC3, P0) -------------------------------------------------

// TestRT_SingletonNotDisposedOnScopeClose: closing a child scope must NOT
// release singletons — they are tracked by root.disp, not scope.disp (AD-12,
// D-4 iron proof: DisposingScopeDisposesService + SingletonServicesComeFromRootProvider).
func TestRT_SingletonNotDisposedOnScopeClose(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddSingleton[iClosableSingleton](b, func(godi.Provider) iClosableSingleton {
		return &closableSvc{label: "singleton", tracker: tracker}
	})
	runtime.AddScoped[iClosableScoped](b, func(godi.Provider) iClosableScoped {
		return &closableSvc{label: "scoped", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)

	scope := p.CreateScope()

	// Resolve both from scope
	singleton := scope.GetService(reflect.TypeFor[iClosableSingleton]()).(*closableSvc)
	scoped := scope.GetService(reflect.TypeFor[iClosableScoped]()).(*closableSvc)

	// Close scope
	testutil.NoError(t, scope.Close())

	// Scoped released; singleton NOT released
	testutil.True(t, scoped.disposed, "scoped must be disposed after scope.Close")
	testutil.True(t, !singleton.disposed,
		"singleton must NOT be disposed on scope.Close — only with root (AD-12)")

	// Close root: NOW singleton is released
	testutil.NoError(t, p.Close())
	testutil.True(t, singleton.disposed, "singleton must be disposed after root.Close")
}

// --- 2.3-PROV-005 (AC2, P0) -------------------------------------------------

// TestRT_ScopeClose_LIFO_Order: disposable services created within a scope must
// be released in reverse creation order (LIFO). This tests the LIFO contract
// through the runtime provider (DisposesInReverseOrderOfCreation simplified).
func TestRT_ScopeClose_LIFO_Order(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddScoped[iClosableScoped](b, func(godi.Provider) iClosableScoped {
		return &closableSvc{label: "scoped", tracker: tracker}
	})
	runtime.AddTransient[iClosableTransient](b, func(godi.Provider) iClosableTransient {
		return &closableSvc{label: "transient", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	scope := p.CreateScope()

	// Creation order: scoped (first resolve), transient-0, transient-1
	_ = scope.GetService(reflect.TypeFor[iClosableScoped]())
	_ = scope.GetService(reflect.TypeFor[iClosableTransient]())
	_ = scope.GetService(reflect.TypeFor[iClosableTransient]())

	testutil.NoError(t, scope.Close())

	// LIFO: transient-1, transient-0, scoped (reverse of creation)
	testutil.Equal(t, len(tracker.labels), 3)
	testutil.Equal(t, tracker.labels[0], "transient")
	testutil.Equal(t, tracker.labels[1], "transient")
	testutil.Equal(t, tracker.labels[2], "scoped")
}

// --- 2.3-PROV-006 (AC3, P1) -------------------------------------------------

// TestRT_CloseIdempotent_WithDisposables: Close must be idempotent even with
// disposable services — the LIFO disposer runs exactly once. This tightens the
// existing TestClose_DoesNotDoubleRelease (provider_test.go:162) from <= 1 to == 1.
func TestRT_CloseIdempotent_WithDisposables(t *testing.T) {
	var closeCount int
	b := runtime.NewBuilder()
	runtime.AddSingleton[*closableLogger](b, func(godi.Provider) *closableLogger {
		return &closableLogger{onClose: func() { closeCount++ }}
	})
	p, err := b.Build()
	testutil.NoError(t, err)

	// Force resolve to materialize
	_ = p.GetService(reflect.TypeFor[*closableLogger]())

	testutil.NoError(t, p.Close())
	testutil.NoError(t, p.Close())

	testutil.Equal(t, closeCount, 1) // exactly once, not 0 (no-op) or 2 (double)
}

// --- 2.3-PROV-007 (AC3, P1) -------------------------------------------------

// TestRT_ScopeClose_SyncOnceGuard: scope.Close must be guarded by sync.Once
// (consuming deferred from 1.3 review). Multiple Close calls must not panic
// or double-release.
func TestRT_ScopeClose_SyncOnceGuard(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddScoped[iClosableScoped](b, func(godi.Provider) iClosableScoped {
		return &closableSvc{label: "scoped", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	scope := p.CreateScope()
	_ = scope.GetService(reflect.TypeFor[iClosableScoped]())

	testutil.NoError(t, scope.Close())
	testutil.NoError(t, scope.Close()) // second call must be no-op
	testutil.NoError(t, scope.Close()) // third call must be no-op

	testutil.Equal(t, len(tracker.labels), 1) // released exactly once
}

// --- 2.3-PROV-008 (AC2, P1) -------------------------------------------------

// TestRT_RootClose_ReleasesRootScoped: root-level scoped services (resolved
// from root, which acts as root scope D-2) must be released on root.Close.
func TestRT_RootClose_ReleasesRootScoped(t *testing.T) {
	tracker := &disposeTracker{}
	b := runtime.NewBuilder()
	runtime.AddScoped[iClosableScoped](b, func(godi.Provider) iClosableScoped {
		return &closableSvc{label: "rootScoped", tracker: tracker}
	})
	p, err := b.Build()
	testutil.NoError(t, err)

	svc := p.GetService(reflect.TypeFor[iClosableScoped]()).(*closableSvc)
	testutil.True(t, !svc.disposed, "root-scoped not disposed before root.Close")

	testutil.NoError(t, p.Close())
	testutil.True(t, svc.disposed, "root-scoped must be disposed after root.Close")
}
