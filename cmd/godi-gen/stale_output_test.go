// Stale-output tolerance (evaluation report §7.3, finding C1). In the in-place
// layout the registration package also holds godi-gen's previous product,
// container_gen.go. Once a constructor's signature changes that file no longer
// type-checks (or, after an interrupted write, no longer parses); these tests
// pin that the generator ignores it while loading instead of refusing to run.
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestGenerate_IgnoresStaleGeneratedOutput: the stale container calls NewFoo()
// with no arguments, but NewFoo now takes an ILog. Regeneration must succeed
// and wire the dependency into the new call.
func TestGenerate_IgnoresStaleGeneratedOutput(t *testing.T) {
	out, err := generate("testdata/staleoutput", "staleoutput", "")
	if err != nil {
		t.Fatalf("stale output must not block regeneration: %v", err)
	}
	if !bytes.Contains(out, []byte("r.f_Foo = NewFoo(r.f_Log)")) {
		t.Fatalf("regenerated container must call NewFoo with its ILog dependency, got:\n%s", out)
	}
	if bytes.Contains(out, []byte("NewFoo()")) {
		t.Fatalf("regenerated container must not keep the stale zero-argument NewFoo() call, got:\n%s", out)
	}
}

// TestGenerate_IgnoresSyntacticallyBrokenGeneratedOutput: a header-marked file
// whose body does not parse must not block regeneration either, including when
// the package clause itself is broken. The broken file is written at run time
// (and removed afterwards) instead of being committed: an unparseable .go file
// under testdata/ would make the CI `gofmt -l` step fail.
func TestGenerate_IgnoresSyntacticallyBrokenGeneratedOutput(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"broken body", "package garbageoutput\n\nfunc NewRoot( {{{ this is not Go <<<\n"},
		{"broken package clause", "pack age garbageoutput {{{ this is not Go <<<\n"},
	}
	dir := filepath.Join("testdata", "garbageoutput")
	stale := filepath.Join(dir, "container_gen.go")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				if err := os.Remove(stale); err != nil && !os.IsNotExist(err) {
					t.Errorf("remove %s: %v", stale, err)
				}
			})
			if err := os.WriteFile(stale, []byte(generatedHeader+"\n"+tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := generate(dir, "garbageoutput", ""); err != nil {
				t.Fatal(err)
			}
		})
	}
}
