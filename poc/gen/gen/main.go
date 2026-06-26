// Command gen validates spec.Registry() and emits the wiring into the file
// given as the first argument (default "container_gen.go", relative to cwd).
//
//	go run ./gen app/container_gen.go      # from module root
//	go generate ./app                      # via //go:generate in app/
//
// If the registration graph is invalid, nothing is written and the command
// exits non-zero — the bug never reaches a binary.
package main

import (
	"fmt"
	"os"

	"godipoc/gen/genlib"
	"godipoc/gen/spec"
)

func main() {
	out := "container_gen.go"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}

	src, err := genlib.Generate(spec.Registry())
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, src, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	fmt.Printf("gen: wrote %s\n", out)
}
