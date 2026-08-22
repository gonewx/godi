// ATDD test for Story 1.1 (AC1) — go.mod dependency facts.
// Activated in Story 1.1 Task 1 (initially golang.org/x/tools v0.47.0, bumped
// to v0.49.0 by the Go 1.27 GA closeout; the original //go:build
// atdd_redphase red-phase guard was removed).
// Updated in Story 6.6 Task 1: go directive bumped from 1.26 to 1.27.
package godi_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// TestGoMod_DeclaresStaticAnalysisDependency covers AC1 / 1.1-UNIT-003.
//
// go.mod must pin the exact static-analysis dependency: NFR6 nails
// golang.org/x/tools (originally v0.47.0, now v0.49.0), and the `go` directive
// is 1.27 (unlocking method-level type parameters per AD-5).
func TestGoMod_DeclaresStaticAnalysisDependency(t *testing.T) {
	// The godi package lives at the module root, so go.mod is in the test's
	// working directory.
	raw, err := os.ReadFile("go.mod")
	testutil.NoError(t, err)
	src := string(raw)

	testutil.True(t,
		strings.Contains(src, "golang.org/x/tools v0.49.0"),
		"go.mod must require golang.org/x/tools v0.49.0 (NFR6 pins this exact version)")

	testutil.True(t,
		strings.Contains(src, "go 1.27"),
		"go directive must be 1.27 (unlocking method-level type parameters, AD-5)")
}
