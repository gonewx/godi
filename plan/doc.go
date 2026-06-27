// Package plan holds the registration model, Validate, and (future) topological
// ordering over the dependency graph. It is the single source of registration
// facts the generator emits from; per AD-2 the generate pipeline depends one way
// (cmd/godi-gen -> internal/genlib -> plan) and plan never imports its callers.
//
// Current state (Story 1.4): Registry + Service model with Lifetime enum,
// chainable registration, lookup, and Validate (missing-dependency detection).
// Captured-dependency and circular-dependency checks land in Epic 5.
package plan
