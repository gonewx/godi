//go:build atdd_redphase

// ATDD red-phase scaffold for Story 1.1 (AC1) — go.mod dependency facts.
// Activation (Story 1.1 Task 1): once `golang.org/x/tools v0.47.0` is added to
// go.mod, delete the `//go:build atdd_redphase` line to enable.
package godi_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// TestGoMod_DeclaresStaticAnalysisDependency covers AC1 / 1.1-UNIT-003.
//
// go.mod must pin the exact static-analysis dependency and keep the toolchain
// caveat: NFR6 nails golang.org/x/tools to v0.47.0, and the `go` directive
// stays at 1.26 because Go 1.27 is pre-GA (bumping breaks the build).
func TestGoMod_DeclaresStaticAnalysisDependency(t *testing.T) {
	// The godi package lives at the module root, so go.mod is in the test's
	// working directory.
	raw, err := os.ReadFile("go.mod")
	testutil.NoError(t, err)
	src := string(raw)

	testutil.True(t,
		strings.Contains(src, "golang.org/x/tools v0.47.0"),
		"go.mod must require golang.org/x/tools v0.47.0 (NFR6 pins this exact version)")

	testutil.True(t,
		strings.Contains(src, "go 1.26"),
		"go directive must stay at 1.26 (Go 1.27 pre-GA — bumping breaks the build)")
}
