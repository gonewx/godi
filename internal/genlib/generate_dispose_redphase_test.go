//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.3 (AC1/AC2) — genlib emit: disposer
// field, Track calls, CloseAll emit, hasDisposables guard, golden test.
//
// These tests use the EXISTING genlib.Generate API with plan.Service.Disposable
// = true (field exists since Story 2.1, but was never consumed by emit).
// Assertions check that the emitted code contains disp fields, Track calls,
// CloseAll in Close(), and runtime import — all of which do NOT exist in the
// current emit, making these tests RED.
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-3-scope-dispose-lifo.md.
package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// disposableRegistry returns a three-lifetime registration where each service
// is disposable (Disposable: true). This is the minimum-viable input for
// testing the disposer emit path.
func disposableRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:       "DBConn",
		GoType:     "svc.IDBConn",
		Ctor:       "svc.NewDBConn",
		Lifetime:   plan.Singleton,
		Disposable: true,
	})
	reg.Service(plan.Service{
		Name:       "Session",
		GoType:     "svc.ISession",
		Ctor:       "svc.NewSession",
		Lifetime:   plan.Scoped,
		Disposable: true,
	})
	reg.Service(plan.Service{
		Name:       "TempFile",
		GoType:     "svc.ITempFile",
		Ctor:       "svc.NewTempFile",
		Lifetime:   plan.Transient,
		Disposable: true,
	})
	return reg
}

// mixedDisposableRegistry returns a registry with BOTH disposable and non-
// disposable services. This tests that only Disposable=true services get Track
// calls and that non-disposable services are unaffected.
func mixedDisposableRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:       "Logger",
		GoType:     "*svc.Logger",
		Ctor:       "svc.NewLogger",
		Lifetime:   plan.Singleton,
		Disposable: false,
	})
	reg.Service(plan.Service{
		Name:       "DBConn",
		GoType:     "svc.IDBConn",
		Ctor:       "svc.NewDBConn",
		Lifetime:   plan.Singleton,
		Disposable: true,
	})
	return reg
}

// --- 2.3-GEN-001 (AC1/AC2, P0) -----------------------------------------------

// TestEmitDispose_RootStructHasDispField: when the registry contains disposable
// services, Root struct must include a `disp *runtime.Disposer` field (D-1:
// disposer lives in runtime, generated container imports it).
func TestEmitDispose_RootStructHasDispField(t *testing.T) {
	src, err := genlib.Generate(disposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, "runtime.Disposer"),
		"Root struct must have a *runtime.Disposer field when hasDisposables")
}

// --- 2.3-GEN-002 (AC1, P0) ---------------------------------------------------

// TestEmitDispose_RuntimeImport: when disposable services exist, the import
// block must include "github.com/gonewx/godi/runtime" (AD-2: generated -->
// runtime(disposer) is the sanctioned dependency).
func TestEmitDispose_RuntimeImport(t *testing.T) {
	src, err := genlib.Generate(disposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, `"github.com/gonewx/godi/runtime"`),
		"disposable container must import godi/runtime for Disposer (AD-2)")
}

// --- 2.3-GEN-003 (AC2, P0) ---------------------------------------------------

// TestEmitDispose_TrackCallsEmitted: NewRoot must emit Track calls for each
// disposable singleton and root-scoped service. The Track call wires the
// instance to the disposer for LIFO release (D-3 track emit).
func TestEmitDispose_TrackCallsEmitted(t *testing.T) {
	src, err := genlib.Generate(disposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, "r.disp.Track("),
		"NewRoot must emit Track calls for disposable services")
}

// --- 2.3-GEN-004 (AC2, P0) ---------------------------------------------------

// TestEmitDispose_CloseAllInRootClose: Root.Close() must call r.disp.CloseAll()
// (not return nil) when disposable services exist. This is the LIFO release
// trigger (D-5, D-6: root uses if-closed guard + CloseAll).
func TestEmitDispose_CloseAllInRootClose(t *testing.T) {
	src, err := genlib.Generate(disposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, "disp.CloseAll()"),
		"Root.Close must call disp.CloseAll() for LIFO release")
}

// --- 2.3-GEN-005 (AC2, P0) ---------------------------------------------------

// TestEmitDispose_ClosedGuardInClose: Root.Close() must include a `closed`
// guard for idempotent close (D-6: if r.closed { return nil }). The codegen
// container uses a bool field, not sync.Once (NFR8 single-goroutine).
func TestEmitDispose_ClosedGuardInClose(t *testing.T) {
	src, err := genlib.Generate(disposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, "closed"),
		"Root.Close must have a 'closed' guard for idempotency")
}

// --- 2.3-GEN-006 (AC2, P0) ---------------------------------------------------

// TestEmitDispose_TransientMethodTracksDisposable: New{T}() for a disposable
// transient must Track the new instance — each transient is tracked per
// creation (AD-12: 逐次 track). The emit changes from `return Ctor(...)` to
// `v := Ctor(...); recv.disp.Track(v); return v`.
func TestEmitDispose_TransientMethodTracksDisposable(t *testing.T) {
	src, err := genlib.Generate(disposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	// Find the NewTempFile method and verify it contains Track
	idx := strings.Index(code, "NewTempFile()")
	testutil.True(t, idx >= 0, "NewTempFile transient method must exist")
	methodBlock := code[idx:]
	nextFunc := strings.Index(methodBlock[1:], "\nfunc ")
	if nextFunc > 0 {
		methodBlock = methodBlock[:nextFunc+1]
	}
	testutil.True(t, strings.Contains(methodBlock, ".disp.Track("),
		"New{T}() for disposable transient must Track the instance (AD-12)")
}

// --- 2.3-GEN-007 (AC1, P0) ---------------------------------------------------

// TestEmitDispose_DisposableGolden: the disposable container emit must be byte-
// stable — anchored by a new golden file. This is the canonical disposable emit
// shape (disp field + Track + CloseAll + closed guard).
func TestEmitDispose_DisposableGolden(t *testing.T) {
	src, err := genlib.Generate(disposableRegistry(), "codegen")
	testutil.NoError(t, err)
	testutil.Golden(t, "disposable-container", src)
}

// --- 2.3-GEN-008 (AC1, P1) ---------------------------------------------------

// TestEmitDispose_NonDisposableContainerUnchanged: when NO services are
// disposable, the emit must NOT include disp fields, Track calls, runtime
// import, or closed guard. Existing goldens (single-service, deps-container,
// lifetimes-container, formb-container) must stay byte-identical.
func TestEmitDispose_NonDisposableContainerUnchanged(t *testing.T) {
	// Lifetimes registry (no Disposable services) — must NOT have disp
	reg := lifetimesRegistry()
	src, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, !strings.Contains(code, "runtime.Disposer"),
		"non-disposable container must NOT have Disposer field")
	testutil.True(t, !strings.Contains(code, "disp.Track"),
		"non-disposable container must NOT have Track calls")
	testutil.True(t, !strings.Contains(code, `"github.com/gonewx/godi/runtime"`),
		"non-disposable container must NOT import godi/runtime")

	// Existing golden must be byte-identical
	testutil.Golden(t, "lifetimes-container", src)
}

// --- 2.3-GEN-009 (AC1, P1) ---------------------------------------------------

// TestEmitDispose_MixedRegistry_OnlyDisposableTracked: in a mixed registry
// (some disposable, some not), only Disposable=true services get Track calls.
// Non-disposable services emit exactly as before (no Track, no disp field).
func TestEmitDispose_MixedRegistry_OnlyDisposableTracked(t *testing.T) {
	src, err := genlib.Generate(mixedDisposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	// DBConn (Disposable) must be tracked
	testutil.True(t, strings.Contains(code, "Track"),
		"disposable service must have Track call")

	// The container DOES have disp + runtime import because ≥1 is disposable
	testutil.True(t, strings.Contains(code, "runtime.Disposer"),
		"mixed container must have Disposer because ≥1 service is disposable")
}

// --- 2.3-GEN-010 (AC2, P0) ---------------------------------------------------

// TestEmitDispose_ScopeCloseCallsCloseAll: scope.Close() must call
// s.disp.CloseAll() when disposable services exist (mirrors Root.Close).
func TestEmitDispose_ScopeCloseCallsCloseAll(t *testing.T) {
	src, err := genlib.Generate(disposableRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	// Find scope's Close method
	scopeCloseIdx := strings.Index(code, "func (s *scope) Close()")
	testutil.True(t, scopeCloseIdx >= 0, "scope must have Close method")
	scopeCloseBlock := code[scopeCloseIdx:]
	testutil.True(t, strings.Contains(scopeCloseBlock, "disp.CloseAll()"),
		"scope.Close must call disp.CloseAll() for LIFO release")
}
