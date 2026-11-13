// Package codegen hosts the codegen-provider side of the dual-provider
// conformance suite: scenarios run against the B2 generated container, asserted
// cross-provider-equivalent to the runtime oracle, whose registrations are
// kept equivalent by hand (AD-14 / AD-15).
//
// container_gen.go is the deterministic output of the real cmd/godi-gen
// static-analysis CLI, wired behind the //go:generate directive in
// registration.go and re-emitted by `go generate ./...`; it is never edited by
// hand. It covers every registration in registration.go: singletons and scoped
// services are built through lazy accessors, transients through New<Name>(),
// and singletons are all built by NewRoot() (*Root, error) (D-1). gen_test.go's
// regeneration-comparison guard pins it to a fresh genlib.Generate run,
// byte-for-byte.
package codegen
