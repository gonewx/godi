// Package plan holds the registration model, Validate, and the topological
// ordering over the dependency graph. It is the single source of registration
// facts the generator emits from; per AD-2 the generate pipeline depends one way
// (cmd/godi-gen -> internal/genlib -> plan) and plan never imports its callers.
//
// Story 1.1 placeholder: package declaration only. The model and Validate land
// with the capability epics.
package plan
