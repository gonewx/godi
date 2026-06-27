// Command godi-gen is godi's static-analysis generator CLI. It loads user
// registration files with go/packages and reads type facts with go/types, then
// (Story 1.4) emits the B2 container. Story 1.1 ships only a stub main plus the
// AD-7 integration spike in spike_test.go.
//
// # AD-7 spike conclusion (Story 1.1, the AR11 de-risking gate)
//
// The AD-7 static-analysis path has NO feasibility blocker. The spike proved,
// end to end on a real DSL marker, that the generator can read type facts purely
// statically — without ever executing user registration code (the fixture's
// init() tripwire never fires, which is positive evidence of non-execution).
//
// Key go/types API facts archived here for Story 1.4 to reuse:
//
//   - Info.Instances: the type checker records every generic call's
//     instantiation. Reading inst.TypeArgs recovers the [I, Impl] type arguments
//     of a marker such as godi.AddSingleton[ILogger, Logger], in declaration
//     order. This is how the generator learns the registered service and
//     implementation types without executing the DSL.
//   - types.Implements: testing a concrete implementation type against the
//     io.Closer interface decides Disposable-ness statically at generation time
//     (AD-11) — no hand-filled Disposable flag and no runtime reflection. NOTE:
//     this Story 1.1 spike exercises only Info.Instances; types.Implements
//     feasibility is carried from the architecture's stdlib precedent (go/types
//     TestImplements), not proven by a test here. Story 1.4 adds the first real
//     Disposable-detection assertion on the go/types path.
//
// Story 1.4 archived the real layered generator design (go/packages load ->
// plan.Validate -> genlib emit). The spike artifact is disposable; only these
// API facts and the "no blocker" conclusion carry over.
//
// # Story 2.1: the real CLI lands
//
// main.go is no longer a placeholder. The generator loads a registration package
// (go/packages), reads type facts (go/types: Info.Instances for the
// AddSingleton[I,Impl] / AddSingletonFactory[I] markers; constructor-signature
// reading for New{Impl} dependency discovery and trailing-error detection;
// types.Implements for Disposable), builds a plan.Registry, and emits through the
// single code-emission path genlib.Generate. A form-A service with no New{Impl}
// is rejected at generation time with zero output and a non-zero exit (NFR2,
// AD-3). The conformance/codegen package wires this behind //go:generate.
//
// # Form-B codegen: known limitation (deferred)
//
// Form A (AddSingleton[I, Impl]) is the primary path and emits a fully
// compilable, dependency-wired container. Form B (AddSingletonFactory[I]) is the
// escape hatch: the generator inlines the factory ARGUMENT EXPRESSION verbatim
// (D-2). It does NOT yet package-qualify identifiers the factory body references
// from the registration package, nor extract the factory's own dependencies for
// topo ordering. So a form-B factory that (a) names registration-package-local
// types/constructors or (b) resolves another singleton through the provider emits
// a container that does not compile or wires a nil dependency. Standard
// fixtures/conformance use form A and do not hit this; full form-B codegen is a
// deferred follow-up (see _bmad-output/implementation-artifacts/deferred-work.md).
// The runtime provider has no such limitation — it executes the factory closure
// directly.
package main
