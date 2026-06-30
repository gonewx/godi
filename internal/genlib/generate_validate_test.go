// ATDD red-phase scaffold for Story 5.1 — genlib package: Generate gate
// verification for transitive missing dependencies.
//
// The Generate function (AD-2 unique code-emission path) calls plan.Validate
// first; a non-empty Validate result aborts emission and returns nil output +
// error (AD-3: bad graph → zero-byte code). These tests verify that transitive
// missing dependency errors flow through the Generate gate correctly.
//
// Activation (TDD red → green): delete the //go:build line above and the blank
// line after it once plan.Validate performs transitive DFS walking (Story 5.1
// Task 2). The file then joins the default build.
//
// Note: The Generate logic itself is NOT modified in Story 5.1 — only Validate
// is enhanced. These tests confirm the existing gate behavior holds for the
// enhanced error set.
package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// --- 5.1-GEN-T8 (AC1, P0) — Generate rejects transitive missing dep ----------

// TestGenerate_TransitiveMissingDep_ReturnsError: A→B→missing C. Generate must
// return error because Validate catches the transitive missing dependency.
func TestGenerate_TransitiveMissingDep_ReturnsError(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "B_Impl",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"Missing_C"},
	})
	reg.Service(plan.Service{
		Name:     "A_Impl",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B_Impl"},
	})

	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)

	// The error message must contain "validation failed" and mention the missing dep
	testutil.True(t, strings.Contains(err.Error(), "validation failed"),
		"Generate error must indicate validation failure")
	testutil.True(t, strings.Contains(err.Error(), "Missing_C"),
		"Generate error must mention the missing dependency name")

	// Transitive attribution: A_Impl must be mentioned in the aggregated error
	testutil.True(t, strings.Contains(err.Error(), "A_Impl"),
		"Generate error must attribute transitive failure to A_Impl")
}

// --- 5.1-GEN-T9 (AC1, P0) — Generate produces nil output (zero-byte code) ----

// TestGenerate_TransitiveMissingDep_ZeroByteOutput: same setup as T8. Generate
// must return nil output (zero-byte code) — AD-3 hard gate.
func TestGenerate_TransitiveMissingDep_ZeroByteOutput(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "B_Impl",
		GoType:   "*svc.B",
		Ctor:     "svc.NewB",
		Lifetime: plan.Singleton,
		Deps:     []string{"Missing_C"},
	})
	reg.Service(plan.Service{
		Name:     "A_Impl",
		GoType:   "*svc.A",
		Ctor:     "svc.NewA",
		Lifetime: plan.Singleton,
		Deps:     []string{"B_Impl"},
	})

	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)
	testutil.True(t, len(src) == 0,
		"bad graph must produce zero-byte code output (AD-3)")
}
