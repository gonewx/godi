package testutil

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden, set via `go test -update-golden`, makes Golden rewrite golden
// files instead of comparing against them. Golden files back the generator's
// code-emission tests (AD-2/AD-15: golden 产码测试 — generated container code
// is asserted byte-for-byte against a committed fixture).
var updateGolden = flag.Bool("update-golden", false, "rewrite testdata/*.golden files instead of comparing against them")

// Golden compares got against testdata/<name>.golden (resolved relative to the
// calling test's package, per Go convention). With -update-golden it (re)writes
// the file and returns without comparing.
func Golden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("golden: mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("golden: write %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden: read %s: %v (run `go test -update-golden` to create it)", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch for %q:\n--- got ----\n%s\n--- want ---\n%s", name, got, want)
	}
}
