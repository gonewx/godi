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
// Story 1.4 replaces the throwaway spike emitter with the real layered generator
// (go/packages load -> plan.Validate -> genlib emit). The spike artifact is
// disposable; only these API facts and the "no blocker" conclusion carry over.
//
// No //go:generate directive is wired here or anywhere else (regression risk
// R1): the CI Generate gate stays a clean no-op until Story 1.4.
package main
