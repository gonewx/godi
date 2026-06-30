//go:build atdd_redphase

// ATDD red-phase scaffold for Story 5.3 — genlib package: Generate gate
// verification for circular dependency detection. The Generate function (AD-2)
// calls plan.Validate first; a non-empty result aborts emission and returns nil
// output + error (AD-3).
//
// Red-phase focus: once plan.Validate detects cycles, Generate must reject the
// graph. Until walkCycle is implemented, these tests fail because Validate
// returns no errors for cyclic graphs.
//
// Activation: delete the //go:build line and the blank line after it once
// plan.Validate implements walkCycle (Story 5-3 Task 2).
package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// --- 5.3-GEN-T1 (AC1, P0) — Generate rejects circular dependency -----------

// TestGenerate_Cycle_ReturnsError: A↔B cycle. Generate must return error
// because Validate catches the circular dependency.
func TestGenerate_Cycle_ReturnsError(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B"},
	})
	reg.Service(plan.Service{
		Name:     "B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"A"},
	})

	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)

	if err == nil {
		return
	}
	testutil.True(t, strings.Contains(err.Error(), "validation failed"),
		"Generate error must indicate validation failure")
	testutil.True(t, strings.Contains(err.Error(), "circular dependency"),
		"Generate error must mention circular dependency")
}

// --- 5.3-GEN-T2 (AC1, P0) — Generate produces nil output (zero-byte code) ---

// TestGenerate_Cycle_ZeroByteOutput: same A↔B cycle. Generate must return nil
// output (zero-byte code) — AD-3 hard gate.
func TestGenerate_Cycle_ZeroByteOutput(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "A",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B"},
	})
	reg.Service(plan.Service{
		Name:     "B",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"A"},
	})

	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)
	testutil.True(t, len(src) == 0,
		"cyclic graph must produce zero-byte code output (AD-3)")
}
