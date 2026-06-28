// Conformance tests for Story 3.4 (AC4/AC6, Task 6) — codegen Path γ: keyed
// IsService run STRICTLY on the pre-generated B2 container (codegen.NewRoot()),
// not only via cross-provider differential equivalence.
//
// Path γ (AD-15): keyed IsService is an AD-named coverage-obligation axis, so the
// generated container's IsKeyedService is exercised directly here — registered
// (type, key) hits (incl. the closed-generic keyed registration), unregistered
// misses, nil→unkeyed routing, and the AnyKey-returns-false-without-panic guard
// (the deliberate contrast with GetKeyedService(AnyKey), which panics).
//
// This file was authored GREEN (Tasks 1-4 already emit IsKeyedService on the B2
// container). The original ATDD red-phase scaffold would have failed because the
// generated container had no IsKeyedService method at all.
package codegen_test

import (
	"os"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"reflect"

	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
)

// --- 3.4-CGPATH-001 (AC1/AC4, P0) — B2 IsKeyedService true for registered ----
//
// On the regenerated B2 container, IsKeyedService returns true for every
// registered (type, key) pair, INCLUDING the closed-generic keyed registration
// IFakeOpenGenericService[IFakeService] under "gen-key" (a closed generic is an
// ordinary registered type in godi — ClosedGenericsWithIsKeyedService, AC4).
func TestCodegen_PathGamma_KeyedIsService_RegisteredTrue(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	keyedT := reflect.TypeFor[fakes.IKeyedService]()
	testutil.True(t, root.IsKeyedService(keyedT, "service1"),
		"registered concrete-keyed (IKeyedService, service1) must be IsKeyedService=true")
	testutil.True(t, root.IsKeyedService(keyedT, "scoped"),
		"registered keyed scoped (IKeyedService, scoped) must be IsKeyedService=true")
	testutil.True(t, root.IsKeyedService(keyedT, "transient"),
		"registered keyed transient (IKeyedService, transient) must be IsKeyedService=true")

	// AC4 — closed-generic keyed registration reports true.
	genT := reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]()
	testutil.True(t, root.IsKeyedService(genT, "gen-key"),
		"closed-generic keyed (IFakeOpenGenericService[IFakeService], gen-key) must be IsKeyedService=true")
}

// --- 3.4-CGPATH-002 (AC1, P0) — B2 IsKeyedService false for unregistered -----
//
// Unregistered (type, key) pairs report false: a stranger key on a registered
// type, and an unregistered type entirely.
func TestCodegen_PathGamma_KeyedIsService_UnregisteredFalse(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	keyedT := reflect.TypeFor[fakes.IKeyedService]()
	testutil.True(t, !root.IsKeyedService(keyedT, "no-such-key"),
		"stranger key on a registered type must be IsKeyedService=false")
	// An unregistered type entirely.
	testutil.True(t, !root.IsKeyedService(reflect.TypeFor[int](), "service1"),
		"unregistered type must be IsKeyedService=false")
	// The closed-generic type under a wrong key.
	genT := reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]()
	testutil.True(t, !root.IsKeyedService(genT, "wrong-key"),
		"closed-generic under a wrong key must be IsKeyedService=false")
}

// --- 3.4-CGPATH-003 (AC2, P0) — B2 IsKeyedService(nil) routes to unkeyed ------
//
// A nil key routes to the unkeyed IsService: IsKeyedService(t, nil) == IsService(t)
// for both a registered unkeyed type and an unregistered one (null == unkeyed,
// AD-17). The B2 container registers an unkeyed IAnyKeyService (Story 3-3).
func TestCodegen_PathGamma_KeyedIsService_NilRoutesToUnkeyed(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	// Registered unkeyed type → nil-key query equals IsService.
	unkeyedT := reflect.TypeFor[fakes.IAnyKeyService]()
	testutil.Equal(t, root.IsKeyedService(unkeyedT, nil), root.IsService(unkeyedT))
	testutil.True(t, root.IsKeyedService(unkeyedT, nil),
		"nil key on a registered unkeyed type must route to IsService=true")

	// Unregistered type → nil-key query equals IsService == false.
	missingT := reflect.TypeFor[int]()
	testutil.Equal(t, root.IsKeyedService(missingT, nil), root.IsService(missingT))
	testutil.True(t, !root.IsKeyedService(missingT, nil),
		"nil key on an unregistered type must route to IsService=false")
}

// --- 3.4-CGPATH-004 (AC3, P0) — B2 IsKeyedService(AnyKey) false, never panics -
//
// On the generated container IsKeyedService(t, AnyKey) returns false WITHOUT
// panicking, from root AND scope — the deliberate contrast with the generated
// GetKeyedService(AnyKey), which panics (AD-17). A pure query must never throw.
func TestCodegen_PathGamma_KeyedIsService_AnyKeyFalseNoPanic(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	keyedT := reflect.TypeFor[fakes.IKeyedService]()

	// root: false, no panic.
	testutil.True(t, !cgKeyedIsServiceAnyKeyPanics(root, keyedT),
		"root IsKeyedService(AnyKey) must not panic")
	testutil.True(t, !root.IsKeyedService(keyedT, godi.AnyKey),
		"root IsKeyedService(AnyKey) must be false")

	// scope: false, no panic (root+scope parity).
	sc := root.CreateScope()
	t.Cleanup(func() { _ = sc.Close() })
	testutil.True(t, !cgKeyedIsServiceAnyKeyPanics(sc, keyedT),
		"scope IsKeyedService(AnyKey) must not panic")
	testutil.True(t, !sc.IsKeyedService(keyedT, godi.AnyKey),
		"scope IsKeyedService(AnyKey) must be false")
}

// cgKeyedIsServiceAnyKeyPanics reports whether IsKeyedService(t, AnyKey) panics —
// it must NOT (a pure query never throws, the contrast with GetKeyedService).
func cgKeyedIsServiceAnyKeyPanics(p godi.Provider, t reflect.Type) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	_ = p.IsKeyedService(t, godi.AnyKey)
	return false
}

// --- 3.4-CGPATH-005 (AC6, P1) — regen byte-identical guard -------------------
//
// Determinism guard (AD-14): committed container_gen.go must equal a fresh
// genlib.Generate of conformanceRegistry() (defined in gen_test.go), now including
// the closed-generic keyed registration + the emitted IsKeyedService methods.
func TestCodegen_RegenComparison_WithKeyedIsService(t *testing.T) {
	reg := conformanceRegistry()
	freshSrc, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)

	_, thisFile, _, _ := goruntime.Caller(0)
	containerPath := strings.Replace(thisFile, "gen_keyed_isservice_redphase_test.go", "container_gen.go", 1)
	committed, readErr := os.ReadFile(containerPath)
	testutil.NoError(t, readErr)

	testutil.Equal(t, string(freshSrc), string(committed))
}
