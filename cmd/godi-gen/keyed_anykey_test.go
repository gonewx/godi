// Story 3.3 (AC4 / AC1, codegen side) — the generator's AnyKey/null registration
// gates. These are "codegen-only" obligations with a symmetric runtime
// counterpart (AD-15): the runtime rejects AnyKey-as-key / routes nil-as-key at
// AddKeyed* (asserted in conformance), while the generator does the same at
// generate time, asserted here against testdata fixtures.
//
//   - AnyKey as a registration KEY -> generate error + zero bytes (AC4, D-3):
//     catch-all registration has no Go mechanism, so both providers reject it.
//   - nil registration KEY -> a plain UNKEYED service (AC1): symmetric to nil
//     resolution routing to GetService.
package main

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// TestGenerator_AnyKeyAsRegistrationKey_Rejected covers 3.3-GEN-001 (AC4 / D-3).
//
// A form-A keyed marker registered under godi.AnyKey must fail generation with a
// SPECIFIC message naming AnyKey as a query-only sentinel — replacing the generic
// "not a compile-time constant" D-1 error — and emit zero bytes (NFR2 hard gate).
func TestGenerator_AnyKeyAsRegistrationKey_Rejected(t *testing.T) {
	src, err := generate("testdata/keyedanykey", "codegen", "", develVersion)

	testutil.Error(t, err)
	testutil.Nil(t, src) // zero-byte code on a rejected graph (NFR2)

	msg := err.Error()
	testutil.True(t, strings.Contains(msg, "AnyKey"),
		"the rejection must name AnyKey explicitly (AC4)")
	testutil.True(t, strings.Contains(msg, "registration key"),
		"the rejection must explain AnyKey cannot be a registration key (AC4)")
	// The specific message must REPLACE the generic D-1 non-constant error, not sit
	// alongside it — otherwise the real reason (catch-all has no Go mechanism) is
	// buried under a misleading "not a compile-time constant".
	testutil.True(t, !strings.Contains(msg, "not a compile-time constant"),
		"the AnyKey-specific message must replace the generic non-constant error (D-3)")
}

// TestGenerator_NilKey_RoutesToUnkeyed covers 3.3-GEN-002 (AC1 registration side).
//
// A nil registration key normalizes to an unkeyed service: the generator emits an
// unkeyed accessor and wiring for it, and NO keyed switch arm. A nil key has no
// constant value, so without the pre-normalization it would be wrongly rejected
// by the D-1 constant check — this proves the normalization runs first (R3).
func TestGenerator_NilKey_RoutesToUnkeyed(t *testing.T) {
	src, err := generate("testdata/keyednil", "codegen", "", develVersion)
	testutil.NoError(t, err) // nil key must NOT trip the D-1 non-constant gate
	testutil.NotNil(t, src)

	code := string(src)
	// Wired as an ordinary unkeyed singleton (unkeyed field + accessor).
	testutil.True(t, strings.Contains(emittedFunc(t, code, "func (r *Root) get_Service("), "v := keyednilfix.NewService()"),
		"a nil-keyed service must be wired as an unkeyed singleton (AC1)")
	testutil.True(t, strings.Contains(code, "func (r *Root) Service() keyednilfix.IService"),
		"a nil-keyed service must get the unkeyed typed accessor (AC1)")
	// Reachable via the unkeyed GetService switch, not a keyed switch arm.
	testutil.True(t, strings.Contains(code, "case reflect.TypeFor[keyednilfix.IService]():"),
		"a nil-keyed service must be resolvable via GetService (unkeyed namespace)")
	// No keyed switch arm keyed by nil/<nil> — it never entered the keyed registry.
	testutil.True(t, !strings.Contains(code, "case nil:") && !strings.Contains(code, "case <nil>:"),
		"a nil-keyed service must NOT emit a keyed switch arm")
}
