// ATDD test for Story 6.6 (AC1, Task 1) — toolchain version facts: go.mod,
// the parent workspace's go.work, and mise.toml must all declare/pin Go 1.27
// (unblocking method-level generics, AD-5's remaining open item).
//
// Activated in Story 6.6 Task 1 (go.mod bumped to 1.27, go.work synced,
// mise.toml already at 1.27rc1; original //go:build atdd_redphase removed).
package godi_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// --- 6.6-MOD-001 (AC1, P0) — go.mod declares go 1.27 ------------------------

func TestGoMod_DeclaresGo127(t *testing.T) {
	raw, err := os.ReadFile("go.mod")
	testutil.NoError(t, err)
	src := string(raw)

	testutil.True(t, strings.Contains(src, "go 1.27"),
		"go.mod's `go` directive must be 1.27 once Go 1.27 is GA (AC1)")
	testutil.True(t, !strings.Contains(src, "go 1.26"),
		"go.mod must no longer declare `go 1.26` — the pre-GA placeholder is retired (AC1)")
}

// --- 6.6-MOD-002 (AC1, P0) — workspace go.work stays in lockstep -----------

// TestGoWork_DeclaresGo127: the parent auth workspace's go.work (godi is a
// workspace member, not a standalone module) must declare `go 1.27` too —
// AC1's explicit "workspace consistency" requirement.
func TestGoWork_DeclaresGo127(t *testing.T) {
	raw, err := os.ReadFile("../go.work")
	testutil.NoError(t, err)
	src := string(raw)

	testutil.True(t, strings.Contains(src, "go 1.27"),
		"../go.work must declare `go 1.27` in lockstep with godi/go.mod (AC1)")
}

// --- 6.6-MOD-003 (AC1, P0) — mise.toml pins a 1.27.x toolchain -------------

func TestMiseToml_PinsGo127(t *testing.T) {
	raw, err := os.ReadFile("mise.toml")
	testutil.NoError(t, err)
	src := string(raw)

	testutil.True(t, strings.Contains(src, `go = "1.27`),
		`mise.toml must pin the toolchain to a 1.27.x release, e.g. go = "1.27.x" (AC1)`)
}
