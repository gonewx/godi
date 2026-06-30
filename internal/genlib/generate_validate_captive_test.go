// ATDD tests for Story 5.2 — genlib package: Generate gate verification for
// captive dependency detection. The Generate function (AD-2) calls
// plan.Validate first; a non-empty result aborts emission and returns nil
// output + error (AD-3).
package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// --- 5.2-GEN-T1 (AC1, P0) — Generate rejects captive dependency -------------

// TestGenerate_CaptiveDep_ReturnsError: Singleton→Scoped. Generate must return
// error because Validate catches the captive dependency.
func TestGenerate_CaptiveDep_ReturnsError(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "ScopedSvc",
		GoType:   "*svc.ScopedSvc",
		Ctor:     "svc.NewScopedSvc",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "SingletonSvc",
		GoType:   "*svc.SingletonSvc",
		Ctor:     "svc.NewSingletonSvc",
		Lifetime: plan.Singleton,
		Deps:     []string{"ScopedSvc"},
	})

	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)

	if err == nil {
		return
	}
	testutil.True(t, strings.Contains(err.Error(), "validation failed"),
		"Generate error must indicate validation failure")
	testutil.True(t, strings.Contains(err.Error(), "captive dependency"),
		"Generate error must mention captive dependency")
}

// --- 5.2-GEN-T2 (AC1, P0) — Generate produces nil output (zero-byte code) ---

// TestGenerate_CaptiveDep_ZeroByteOutput: same setup as T1. Generate must
// return nil output (zero-byte code) — AD-3 hard gate.
func TestGenerate_CaptiveDep_ZeroByteOutput(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "ScopedSvc",
		GoType:   "*svc.ScopedSvc",
		Ctor:     "svc.NewScopedSvc",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "SingletonSvc",
		GoType:   "*svc.SingletonSvc",
		Ctor:     "svc.NewSingletonSvc",
		Lifetime: plan.Singleton,
		Deps:     []string{"ScopedSvc"},
	})

	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)
	testutil.True(t, len(src) == 0,
		"captive dep graph must produce zero-byte code output (AD-3)")
}
