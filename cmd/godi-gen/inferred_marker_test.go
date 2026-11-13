// Task 7 — marker recognition no longer depends on the call spelling its type
// arguments, and a call that resolves to godi (or to a same-package
// per-service helper) but cannot be read REJECTS generation instead of being
// skipped with a warning. Only a same-named function that does not resolve to
// godi stays a warning.
package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStderr runs fn with os.Stderr redirected and returns what it wrote,
// so warning behavior can be asserted and does not leak into test output.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	defer func() { os.Stderr = orig }()
	fn()
	_ = w.Close()
	os.Stderr = orig
	return <-done
}

func TestGenerate_InferredTypeArgumentMarkerIsRegistered(t *testing.T) {
	var out []byte
	var err error
	stderr := captureStderr(t, func() { out, err = generate("testdata/inferred", "inferred", "", develVersion) })
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("reflect.TypeFor[ILog]()")) {
		t.Fatal("ILog registered through an inferred marker must be in the container")
	}
	if stderr != "" {
		t.Fatalf("an inferred marker must be read silently, got stderr:\n%s", stderr)
	}
}

func TestGenerate_UnsupportedHelperShapeIsAnError(t *testing.T) {
	var out []byte
	var err error
	stderr := captureStderr(t, func() { out, err = generate("testdata/unreadablehelper", "unreadablehelper", "", develVersion) })
	if err == nil || out != nil {
		t.Fatalf("want rejection with zero bytes, got err=%v len=%d", err, len(out))
	}
	if !strings.Contains(err.Error(), "registration.go:") {
		t.Fatalf("error must carry a source position: %v", err)
	}
	if !strings.Contains(err.Error(), "generation rejected") || !strings.Contains(err.Error(), "AddBox") {
		t.Fatalf("error must reject generation and name the helper call: %v", err)
	}
	if stderr != "" {
		t.Fatalf("the unsupported helper must be an error, not a warning; got stderr:\n%s", stderr)
	}
}

func TestGenerate_ForeignSameNamedFunctionIsNotAnError(t *testing.T) {
	var out []byte
	var err error
	stderr := captureStderr(t, func() { out, err = generate("testdata/foreignsamename", "foreignsamename", "", develVersion) })
	if err != nil {
		t.Fatalf("a same-named non-godi function must not reject generation: %v", err)
	}
	for _, want := range []string{"reflect.TypeFor[IFoo]()", "reflect.TypeFor[IBar]()", "reflect.TypeFor[IBaz]()"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Fatalf("the legal godi registrations must still be in the container: missing %s", want)
		}
	}
	if strings.Count(stderr, "godi-gen: warning") != 1 {
		t.Fatalf("only the same-named AddSingleton call may be warned about, got stderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, "warning") || !strings.Contains(stderr, "AddSingleton") {
		t.Fatalf("a same-named non-godi function must still be warned about, got stderr:\n%s", stderr)
	}
}

func TestGenerate_InferredMarkerInsideHelperBodyIsRegistered(t *testing.T) {
	var out []byte
	var err error
	stderr := captureStderr(t, func() { out, err = generate("testdata/inferredhelper", "inferredhelper", "", develVersion) })
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("reflect.TypeFor[IFoo[int]]()")) {
		t.Fatalf("helper AddFoo[int] forwarding to an inferred marker must materialize IFoo[int]:\n%s", out)
	}
	if stderr != "" {
		t.Fatalf("an inferred helper-body marker must be read silently, got stderr:\n%s", stderr)
	}
}

func TestGenerate_DotImportedInferredMarkerIsRegistered(t *testing.T) {
	var out []byte
	var err error
	stderr := captureStderr(t, func() { out, err = generate("testdata/dotimport", "dotimport", "", develVersion) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"reflect.TypeFor[IFoo]()", "reflect.TypeFor[ILog]()"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Fatalf("dot-imported marker must be registered: missing %s", want)
		}
	}
	if stderr != "" {
		t.Fatalf("dot-imported markers must be read silently, got stderr:\n%s", stderr)
	}
}
