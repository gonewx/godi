// Story 2.1 (AC1/AC2) — the real godi-gen CLI: static-analysis constructor
// discovery + the missing-New{Impl} gen-time error gate (Task 3). This is the
// first story where cmd/godi-gen does real work (its main() was a placeholder).
//
// The generator loads a registration package with go/packages, reads type facts
// with go/types (reusing the spike's Info.Instances path), discovers the unique
// New{Impl} constructor for each form-A service, reads its dependencies left to
// right from the signature, detects a trailing error, runs plan.Validate (inside
// genlib.Generate), and emits the container. A form-A service with no New{Impl}
// is a gen-time error that hints form B and emits ZERO bytes (NFR2, AD-8).
//
// generate(srcDir, pkgName) ([]byte, error) is the testable entry point:
//   - legal form-A graph  -> emitted container source, nil error
//   - form-A with no New{Impl} -> nil output + error hinting form B
//   - static analysis NEVER executes user code (AD-7)
//
// The testdata fixtures (deps / nonewimpl / formb) are test INPUT under
// testdata/, never part of the module build. AC2's non-zero exit is verified
// end-to-end with: GOWORK=off go run . -pkg codegen testdata/nonewimpl
package main

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// TestGenerator_FormA_DiscoversCtorAndWiresDeps covers 2.1-GEN-001 (AC1).
//
// For the form-A deps fixture the generator must discover NewConsumer, read its
// dependency parameter, and emit a container that passes that dependency as a
// constructor argument.
func TestGenerator_FormA_DiscoversCtorAndWiresDeps(t *testing.T) {
	src, err := generate("testdata/deps", "codegen")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, strings.Contains(code, "NewConsumer(r.f_"),
		"generator must discover NewConsumer and pass its discovered dependency as an argument")
}

// TestGenerator_FormA_DetectsTrailingErrorCtor covers 2.1-GEN-002 (AC1).
//
// When a discovered constructor has a trailing error return, the emitted NewRoot
// must adopt the (*Root, error) signature so the error can propagate (AD-8/D-1).
func TestGenerator_FormA_DetectsTrailingErrorCtor(t *testing.T) {
	src, err := generate("testdata/deps", "codegen")
	testutil.NoError(t, err)

	testutil.True(t, strings.Contains(string(src), "func NewRoot() (*Root, error)"),
		"generator must emit an error-propagating NewRoot when a discovered ctor returns error")
}

// TestGenerator_FormA_NoNewImpl_ErrorsWithFormBHint covers 2.1-GEN-003 (AC2).
//
// This is the AC2 gen-time error gate — a "codegen-only" obligation with no
// runtime counterpart (AD-15). A form-A service with no New{Impl} must fail
// generation with a hint to use form B AND emit zero bytes (NFR2).
func TestGenerator_FormA_NoNewImpl_ErrorsWithFormBHint(t *testing.T) {
	src, err := generate("testdata/nonewimpl", "codegen")

	testutil.Error(t, err)
	testutil.Nil(t, src) // zero-byte code on a rejected graph (NFR2 hard gate)

	msg := err.Error()
	hintsFormB := strings.Contains(msg, "AddSingletonFactory") ||
		strings.Contains(strings.ToLower(msg), "form b") ||
		strings.Contains(msg, "形态 B")
	testutil.True(t, hintsFormB,
		"missing New{Impl} must error with a hint to use form B AddSingletonFactory (AD-8)")
}

// TestGenerator_StaticAnalysis_DoesNotExecuteUserCode covers 2.1-GEN-004 (AC1/AD-7).
//
// The deps fixture's init() panics if executed. Reaching a clean (non-panicking)
// generate result is positive evidence go/packages only parsed and type-checked
// the registration — it never ran it (AD-7).
func TestGenerator_StaticAnalysis_DoesNotExecuteUserCode(t *testing.T) {
	_, err := generate("testdata/deps", "codegen")
	testutil.NoError(t, err) // a panic from the fixture tripwire would fail here
}

// TestGenerator_FormB_InlinesFactoryExpression covers 2.1-GEN-005 (AC1).
//
// Form B is the fallback: the generator statically reads the factory argument
// expression and inlines it into NewRoot as `r.f_I = (factoryExpr)(r)` (D-2),
// rather than discovering a New{Impl}.
func TestGenerator_FormB_InlinesFactoryExpression(t *testing.T) {
	src, err := generate("testdata/formb", "codegen")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, strings.Contains(code, "r.f_") && strings.Contains(code, "(r)"),
		"form-B factory expression must be inlined into NewRoot as (factoryExpr)(r) (D-2)")
}
