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
// generate(srcDir, pkgName, outPath) ([]byte, error) is the testable entry point:
//   - legal form-A graph  -> emitted container source, nil error
//   - form-A with no New{Impl} -> nil output + error hinting form B
//   - static analysis NEVER executes user code (AD-7)
//
// The testdata fixtures (deps / nonewimpl / formb) are test INPUT under
// testdata/, never part of the module build. AC2's non-zero exit is verified
// end-to-end with: GOWORK=off go run . -pkg codegen testdata/nonewimpl
package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
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
	src, err := generate("testdata/deps", "codegen", "")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, strings.Contains(code, "NewConsumer(r.get_"),
		"generator must discover NewConsumer and pass its discovered dependency as an argument")
}

// TestGenerator_FormA_DetectsTrailingErrorCtor covers 2.1-GEN-002 (AC1).
//
// When a discovered constructor has a trailing error return, the singleton's
// accessor must use comma-err form and panic with the error, which NewRoot
// recovers and returns (AD-8/D-1).
func TestGenerator_FormA_DetectsTrailingErrorCtor(t *testing.T) {
	src, err := generate("testdata/deps", "codegen", "")
	testutil.NoError(t, err)
	code := string(src)

	// The (*Root, error) signature alone is a tautology under D-1 (NewRoot is
	// unconditionally (*Root, error)); assert the actual comma-err WIRING that
	// proves hasTrailingError read CtorErr from NewConsumer's (*Consumer, error).
	consumer := emittedFunc(t, code, "func (r *Root) get_Consumer(")
	testutil.True(t, strings.Contains(consumer,
		"v, err := depsfix.NewConsumer(r.get_FakeService())\n\tif err != nil {\n\t\tpanic(err)\n\t}"),
		"a discovered trailing-error ctor must emit comma-err form that panics with the error")
	testutil.True(t, strings.Contains(emittedFunc(t, code, "func NewRoot("), "root, err = nil, godi.BuildError(rec)"),
		"a trailing-error ctor must propagate the error up from NewRoot")
	// Negative guard: the no-error sibling must use single-assignment form, proving
	// CtorErr is detected per-service, not emitted unconditionally.
	fake := emittedFunc(t, code, "func (r *Root) get_FakeService(")
	testutil.True(t, strings.Contains(fake, "v := depsfix.NewFakeService()") && !strings.Contains(fake, "err"),
		"a no-error ctor must emit single-assignment form, not comma-err")
}

// TestGenerator_FormA_NoNewImpl_ErrorsWithFormBHint covers 2.1-GEN-003 (AC2).
//
// This is the AC2 gen-time error gate — a "codegen-only" obligation with no
// runtime counterpart (AD-15). A form-A service with no New{Impl} must fail
// generation with a hint to use form B AND emit zero bytes (NFR2).
func TestGenerator_FormA_NoNewImpl_ErrorsWithFormBHint(t *testing.T) {
	src, err := generate("testdata/nonewimpl", "codegen", "")

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
	_, err := generate("testdata/deps", "codegen", "")
	testutil.NoError(t, err) // a panic from the fixture tripwire would fail here
}

// TestGenerator_FormB_InlinesFactoryExpression covers 2.1-GEN-005 (AC1).
//
// Form B is the fallback: the generator statically reads the factory argument
// expression and inlines it into the singleton's accessor as
// `v := (factoryExpr)(r)` (D-2).
//
// Since the Task-0 godi-gen extension (gois story 2-1), inlining into a
// DIFFERENT package than the registration package-qualifies the factory body's
// registration-package identifiers (IWidget → formbfix.IWidget) and imports
// that package — closing the deferred-work.md "form-B codegen 完整支持依赖型/跨包
// 工厂" qualification gap that previously made this fixture's inline
// non-compilable. This test pins the exact qualified inline plus the emitted
// import.
func TestGenerator_FormB_InlinesFactoryExpression(t *testing.T) {
	src, err := generate("testdata/formb", "codegen", "")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, strings.Contains(code,
		"v := (func(p godi.Provider) formbfix.IWidget { return formbfix.NewWidget() })(r)"),
		"a form-B factory inlined into a different package must package-qualify "+
			"registration-package identifiers (D-2 + Task-0 qualification)")
	testutil.True(t, strings.Contains(code,
		`"github.com/gonewx/godi/cmd/godi-gen/testdata/formb"`),
		"the emitted container must import the registration package the qualified "+
			"factory body references")
}

// TestGenerator_NoNewImpl_ExitsNonZero covers AC2/NFR2 at the PROCESS boundary.
//
// generate() returns (nil, err) on a missing-New{Impl} graph, but the NFR2 hard
// gate also requires the BINARY to exit non-zero with zero bytes on stdout —
// behavior owned by main()'s os.Exit(1), which no in-process test asserts. Run
// the real CLI as a subprocess to guard the exit code and empty stdout (the
// story previously verified this only by a manual `echo $?`).
func TestGenerator_NoNewImpl_ExitsNonZero(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess (go run) test in -short mode")
	}
	cmd := exec.Command("go", "run", ".", "-pkg", "codegen", "testdata/nonewimpl")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	var exitErr *exec.ExitError
	testutil.True(t, errors.As(runErr, &exitErr) && exitErr.ExitCode() != 0,
		"godi-gen must exit non-zero on a missing-New{Impl} graph (AC2/NFR2)")
	testutil.True(t, stdout.Len() == 0,
		"a rejected graph must emit zero bytes on stdout (NFR2)")
	testutil.True(t, strings.Contains(stderr.String(), "AddSingletonFactory") ||
		strings.Contains(stderr.String(), "form B") ||
		strings.Contains(stderr.String(), "形态 B"),
		"the rejection must hint form B on stderr")
}
