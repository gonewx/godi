// ATDD test for Story 1.1 (AC1) — Structural Seed package existence.
// Activated in Story 1.1 Task 2 (the placeholder packages exist; the original
// //go:build atdd_redphase red-phase guard was removed). Now part of the
// default build.
package godi_test

import (
	"testing"

	// Structural Seed (ARCHITECTURE-SPINE): every library package must exist
	// and be importable. A missing package fails THIS file's compile,
	// pinpointing which placeholder is absent.
	_ "github.com/gonewx/godi/activator"
	_ "github.com/gonewx/godi/conformance/codegen"
	_ "github.com/gonewx/godi/conformance/fakes"
	_ "github.com/gonewx/godi/internal/genlib"
	_ "github.com/gonewx/godi/plan"
	_ "github.com/gonewx/godi/runtime"
)

// TestStructuralSeedPackagesExist covers AC1 / 1.1-UNIT-002.
//
// The Structural Seed library packages are present and importable. The blank
// imports above are the real assertion — they resolve at compile time.
// cmd/godi-gen (package main) is intentionally NOT blank-imported here: a
// package main cannot be imported; its existence is proven instead by the
// AD-7 spike test in cmd/godi-gen/spike_test.go compiling.
func TestStructuralSeedPackagesExist(t *testing.T) {
	// Reaching this point means every import above resolved at compile time.
	t.Log("Structural Seed library packages import cleanly")
}
