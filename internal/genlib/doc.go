// Package genlib emits generated container code from a validated plan. It is
// internal: only cmd/godi-gen consumes it (AD-2). The sole code-emission path is
// genlib.Generate(reg), which runs reg.Validate() before emitting.
//
// Story 1.1 placeholder: package declaration only. Emission lands in Story 1.4.
package genlib
