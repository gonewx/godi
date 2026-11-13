// Package genlib emits generated container code from a validated plan. It is
// internal: only cmd/godi-gen consumes it (AD-2). The sole code-emission path
// is genlib.Generate(reg, pkgName), which runs reg.Validate() before emitting.
//
// Generate emits a full godi.Provider implementation — typed singleton fields on
// Root (AD-1) filled by lazy accessors that NewRoot triggers for every
// singleton, the seven-method AD-16 contract, and a delegating scope whose
// scoped services are built on first resolution (AD-12). reflect.Type is the dispatch
// key only (Tier B, AD-13); construction is the direct Ctor call (no reflect.New).
//
// Inputs (GoType / Ctor / Imports) are stringly-typed this epoch (D3 caveat);
// Story 2.1 replaces them with go/types facts (AD-7), but the emit shape stays
// the same — the conformance suite's regeneration-comparison guard pins it.
package genlib
