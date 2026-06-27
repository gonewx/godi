// Package genlib emits generated container code from a validated plan. It is
// internal: only cmd/godi-gen consumes it (AD-2). The sole code-emission path is
// genlib.Generate(reg, pkgName), which runs reg.Validate() before emitting.
//
// Current state (Story 1.4): Generate emits a minimal self-contained container
// (Root struct, accessors, Close) for single-service singletons. Fields are
// typed as any; real typed fields land with go/types integration (Story 2.1).
// The full godi.Provider implementation is wired in Story 1.5 (conformance).
package genlib
