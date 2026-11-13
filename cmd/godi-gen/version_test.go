package main

import (
	"bytes"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"testing"
)

// TestVersionFromBuildInfo: the generator reports the module version Go
// stamped into the binary, and "(devel)" whenever there is none.
func TestVersionFromBuildInfo(t *testing.T) {
	cases := []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{"no build info", nil, false, "(devel)"},
		{"build info unavailable", &debug.BuildInfo{Main: debug.Module{Version: "v0.2.0"}}, false, "(devel)"},
		{"go test binary has no main version", &debug.BuildInfo{}, true, "(devel)"},
		{"go run in the repository", &debug.BuildInfo{Main: debug.Module{Path: godiPkgPath, Version: "(devel)"}}, true, "(devel)"},
		{"released dependency", &debug.BuildInfo{Main: debug.Module{Path: godiPkgPath, Version: "v0.2.0"}}, true, "v0.2.0"},
		{"pseudo-version dependency", &debug.BuildInfo{Main: debug.Module{Path: godiPkgPath, Version: "v0.2.1-0.20261113153831-38b28ff737e8"}}, true, "v0.2.1-0.20261113153831-38b28ff737e8"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := versionFromBuildInfo(tc.info, tc.ok); got != tc.want {
				t.Fatalf("versionFromBuildInfo = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestGenerate_StampsGeneratorVersionOnSecondLine: the version generate is
// given lands on the second line, and the first line stays the exact marker
// that identifies a previous godi-gen product.
func TestGenerate_StampsGeneratorVersionOnSecondLine(t *testing.T) {
	out, err := generate("testdata/deps", "container", "", "v1.2.3")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	lines := strings.SplitN(string(out), "\n", 4)
	if len(lines) < 4 {
		t.Fatalf("output too short:\n%s", out)
	}
	if lines[0] != generatedHeader {
		t.Fatalf("line 1 = %q, want %q", lines[0], generatedHeader)
	}
	if want := "// godi-gen version: v1.2.3"; lines[1] != want {
		t.Fatalf("line 2 = %q, want %q", lines[1], want)
	}
	if !hasGeneratedHeader(out) {
		t.Fatal("a versioned container must still be recognized as a godi-gen product")
	}
}

// TestCLI_VersionFlag runs the real CLI: -version prints the generator version
// and exits 0 without needing -pkg or a source directory. Inside this
// repository `go run` stamps no module version, so the answer is "(devel)" —
// the same value the committed container_gen.go carries, which keeps
// `go generate` drift-free.
func TestCLI_VersionFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess (go run) test in -short mode")
	}
	cmd := exec.Command("go", "run", ".", "-version")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("godi-gen -version: %v\nstderr: %s", err, stderr.String())
	}
	if got, want := stdout.String(), "godi-gen (devel)\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
