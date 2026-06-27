// ATDD test for Story 1.1 (AC1) — DSL marker seed contract.
//
// Activated in Story 1.1 Task 2 (godi.Registry + godi.AddSingleton exist; the
// original //go:build atdd_redphase red-phase guard was removed and the test
// driven red -> green). Now part of the default build.
package godi

import (
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// TestDSLSeed_AddSingletonIsPureMarker covers AC1 / 1.1-UNIT-001.
//
// The godi package seed must expose a usable Registry zero value and an
// AddSingleton marker that is a PURE static marker: it returns its receiver
// unchanged and has no runtime behavior. AD-7 — the DSL is a statically
// analyzed marker the generator reads via go/types, never executes. AD-5 — the
// marker stays a free generic function, never an interface method.
func TestDSLSeed_AddSingletonIsPureMarker(t *testing.T) {
	var r Registry // the zero value must be usable as a registration handle

	// AddSingleton[I, Impl] returns r to support chained registration and must
	// NOT panic or mutate global state — it is a marker, not behavior.
	got := AddSingleton[iLoggerSeed, loggerSeed](&r)

	testutil.NotNil(t, got)
	testutil.Equal(t, got, &r) // returns the same receiver, unchanged
}

// iLoggerSeed and loggerSeed are local stand-ins so this unit test exercises
// the generic marker without depending on any real service type. The real
// fixture the AD-7 spike consumes lives in cmd/godi-gen/testdata/spike.
type iLoggerSeed interface{ Log(msg string) }

type loggerSeed struct{}

func (loggerSeed) Log(string) {}
