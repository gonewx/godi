// Package codegen hosts the codegen-provider side of the dual-provider
// conformance suite: scenarios run against the B2 generated container, asserted
// cross-provider-equivalent to the runtime oracle on the same registration
// source (AD-15).
//
// Current state (Story 2.1): container_gen.go is the deterministic output of the
// real cmd/godi-gen static-analysis CLI, wired behind the //go:generate directive
// in registration.go and re-emitted by `go generate ./...` (AD-14: no hand-fork).
// It now carries a dependency graph — FakeService plus a FakeConsumer that
// depends on it — wired through NewFakeConsumer with NewRoot() (*Root, error)
// (D-1). gen_test.go's regeneration-comparison guard pins it to a fresh
// genlib.Generate run, byte-for-byte.
package codegen
