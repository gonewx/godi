// ATDD red-phase scaffold for Story 1.1 (AC3) — spike conclusion archival.
// Activation (Story 1.1 Task 4): once cmd/godi-gen/doc.go records the AD-7
// conclusion, delete the `//go:build atdd_redphase` line to enable.
package main

import (
	"os"
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// TestSpikeConclusion_IsArchived covers AC3 / 1.1-DOC-001.
//
// The package must document that the AD-7 path has no feasibility blocker and
// archive the key API facts for Story 1.4 to reuse.
func TestSpikeConclusion_IsArchived(t *testing.T) {
	raw, err := os.ReadFile("doc.go")
	testutil.NoError(t, err)
	doc := string(raw)

	// Each marker must appear in doc.go: the AD-7 invariant the spike de-risks,
	// the two API facts Story 1.4 reuses (Info.Instances for reading generic
	// type arguments, types.Implements for Disposable detection), and the
	// handoff to Story 1.4's real generator.
	for _, want := range []string{"AD-7", "Info.Instances", "types.Implements", "Story 1.4"} {
		testutil.True(t, strings.Contains(doc, want),
			"cmd/godi-gen/doc.go must record the spike conclusion mentioning "+want)
	}
}
