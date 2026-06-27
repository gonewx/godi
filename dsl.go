package godi

// Registry is the handle a registration file writes against. Story 1.1 keeps it
// an empty struct: the DSL is a static marker (AD-7), so the registry carries no
// runtime state — the generator reads the registration calls' type arguments
// from go/types, never this value. Story 2.x expands its shape.
type Registry struct{}

// AddSingleton records a singleton mapping from interface I to implementation
// Impl. It is a PURE static marker (AD-7): the generator reads the [I, Impl]
// type arguments via go/types Info.Instances at generation time and never
// executes this function. It returns r so registrations can chain.
//
// Per AD-5 the resolution generics stay free functions / concrete-container
// methods; this marker is a free generic function and never an interface method.
func AddSingleton[I any, Impl any](r *Registry) *Registry { return r }
