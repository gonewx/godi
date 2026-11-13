// Type-check gate (evaluation report §7.2, finding D). format.Source only
// proves the emitted container parses; these tests pin that godi-gen also
// type-checks it and, when it does not type-check, writes zero bytes (NFR2)
// instead of exiting 0 with an uncompilable file.
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerate_RejectsUncompilableOutput feeds four registration graphs that
// slip past every earlier gate yet emit a container that does not compile:
// a concrete-typed constructor parameter, an impl that does not satisfy its
// interface, a cross-package key constant, and impl names that collide with
// Root's own methods.
func TestGenerate_RejectsUncompilableOutput(t *testing.T) {
	for _, dir := range []string{"tcconcreteparam", "tcnotimpl", "tckeyconst", "tcreserved"} {
		t.Run(dir, func(t *testing.T) {
			out, err := generate(filepath.Join("testdata", dir), dir, "")
			if err == nil {
				t.Fatalf("want type-check error, got nil")
			}
			if out != nil {
				t.Fatalf("want zero bytes on rejection (NFR2), got %d", len(out))
			}
			if !strings.Contains(err.Error(), "does not type-check") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestGenerate_StdoutModeIgnoresExistingGeneratedFile (Review Focus 1): in
// stdout mode the checked file is a synthetic sibling of the registration
// sources, so a stale container_gen.go already sitting in the directory must be
// stubbed out — otherwise its Root would collide with the fresh Root and the
// gate would reject a legal graph. pkgName is the fixture's real package name
// (depsfix) so the stale file joins the registration package — the
// same-package layout where the Root collision can actually occur.
func TestGenerate_StdoutModeIgnoresExistingGeneratedFile(t *testing.T) {
	src, err := generate("testdata/deps", "depsfix", "")
	if err != nil {
		t.Fatalf("first generate: %v", err)
	}

	stale := filepath.Join("testdata", "deps", "container_gen.go")
	t.Cleanup(func() { _ = os.Remove(stale) })
	if err := os.WriteFile(stale, src, 0o644); err != nil {
		t.Fatalf("write stale generated file: %v", err)
	}

	again, err := generate("testdata/deps", "depsfix", "")
	if err != nil {
		t.Fatalf("second generate with an existing generated file: %v", err)
	}
	if !bytes.Equal(again, src) {
		t.Fatalf("output changed with an existing generated file present")
	}
}

// TestGeneratedFileOverlay_StubsOnlyGeneratedFiles pins the overlay contract
// Task 3 reuses: only files whose first line is generatedHeader are stubbed,
// each to a bare package clause; a generated file whose own package clause is
// unparsable borrows the package name of a hand-written sibling.
func TestGeneratedFileOverlay_StubsOnlyGeneratedFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"a_gen.go": generatedHeader + "\n\npackage widgets\n\ntype Root struct{}\n",
		"b.go":     "package widgets\n\ntype Thing struct{}\n",
		"c_gen.go": generatedHeader + "\n\npackage\n\nthis is not go\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	overlay, err := generatedFileOverlay(dir)
	if err != nil {
		t.Fatalf("generatedFileOverlay: %v", err)
	}
	if len(overlay) != 2 {
		t.Fatalf("want 2 overlay entries (a_gen.go, c_gen.go), got %d: %v", len(overlay), overlay)
	}
	for _, name := range []string{"a_gen.go", "c_gen.go"} {
		got, ok := overlay[filepath.Join(dir, name)]
		if !ok {
			t.Fatalf("overlay missing %s; keys: %v", name, overlay)
		}
		if string(got) != "package widgets\n" {
			t.Fatalf("%s stub = %q, want %q", name, got, "package widgets\n")
		}
	}
}
