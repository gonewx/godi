package godi

// Registry is the handle a registration file writes against. Story 1.1 keeps it
// an empty struct: the DSL is a static marker (AD-7), so the registry carries no
// runtime state — the generator reads the registration calls' type arguments
// from go/types, never this value. Story 2.x expands its shape.
type Registry struct{}

// AddSingleton records a singleton mapping from interface I to implementation
// Impl (form A). It is a PURE static marker (AD-7): the generator reads the
// [I, Impl] type arguments via go/types Info.Instances at generation time and
// never executes this function. It returns r so registrations can chain.
//
// Form A is the primary path (AD-8): the generator discovers the unique
// New{Impl} constructor, reads its dependencies left-to-right and wires them
// directly (Tier A, zero reflective construction). When Impl has no New{Impl}
// constructor, form B (AddSingletonFactory) is the escape hatch.
//
// Per AD-5 the resolution generics stay free functions / concrete-container
// methods; this marker is a free generic function and never an interface method.
func AddSingleton[I any, Impl any](r *Registry) *Registry { return r }

// AddSingletonFactory records a singleton of interface I built by factory
// (form B — the AD-8 escape hatch for implementations that have no New{Impl}
// constructor). Like AddSingleton it is a PURE static marker (AD-7): the
// generator recognizes this call BY NAME and statically reads the factory
// ARGUMENT EXPRESSION (go/ast source text) to inline into the generated
// container as `r.f_I = (factoryExpr)(r)` (D-2); it never executes the factory.
// It returns r so registrations can chain.
//
// Naming (D-4): Go forbids generic-function overloading, so form B CANNOT share
// the name AddSingleton — declaring both AddSingleton[I,Impl] and
// AddSingleton[I](factory) is a duplicate-declaration compile error. The
// distinct name AddSingletonFactory is what the generator dispatches on to tell
// the two forms apart. The factory's godi.Provider parameter is the seam the
// inlined expression resolves dependencies through at runtime.
func AddSingletonFactory[I any](r *Registry, factory func(Provider) I) *Registry {
	return r
}
