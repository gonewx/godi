package plan

import "fmt"

// Lifetime distinguishes the three DI lifetimes (AD-1). This story only
// exercises Singleton, but Scoped and Transient are defined now so that
// Validate's captured-dependency check (Epic 5) can reference them without
// a model change.
type Lifetime int

const (
	Singleton Lifetime = iota
	Scoped
	Transient
)

// String returns a human-readable label used in validation error messages.
func (l Lifetime) String() string {
	switch l {
	case Singleton:
		return "Singleton"
	case Scoped:
		return "Scoped"
	case Transient:
		return "Transient"
	default:
		return fmt.Sprintf("Lifetime(%d)", int(l))
	}
}

// Service describes a single registration in the plan. The string fields
// (GoType, Ctor) are the genlib emit inputs — NOT the canonical type-fact
// source. AD-7: canonical facts come from go/types; these strings are
// deterministically produced by the static analyzer (Story 2.1).
type Service struct {
	Name     string
	GoType   string
	Ctor     string
	Lifetime Lifetime
	// Deps lists the service Names this service's constructor depends on, in
	// left-to-right signature order. genlib emits each as the access expression
	// r.f_<Dep> when wiring the constructor call (Story 2.1).
	Deps []string
	// CtorErr records whether the constructor has a trailing error return. It is
	// an EMIT INPUT (AD-7): the static analyzer (cmd/godi-gen, Task 3) derives it
	// from the constructor's *types.Signature, and emitNewRoot uses it to switch
	// between `r.f_X = Ctor(...)` and the comma-err `r.f_X, err = Ctor(...)` form.
	CtorErr bool
	// FactoryExpr holds the form-B factory argument expression (go/ast source
	// text) when this service was registered via AddSingletonFactory. When
	// non-empty, genlib inlines it as `r.f_X = (FactoryExpr)(r)` (D-2) instead of
	// discovering a New{Impl} constructor; Ctor/Deps are then unused for wiring.
	FactoryExpr string
	// Disposable records whether the implementation satisfies io.Closer, derived
	// statically via types.Implements (AD-11). Story 2.1 only fills this field;
	// the LIFO disposer that consumes it is Story 2.3.
	Disposable bool
}

// Registry accumulates Service registrations and exposes them for
// validation and code generation.
type Registry struct {
	// Imports lists the third-party / project-local import paths the emitted
	// container needs (service packages whose types appear in field decls,
	// constructor calls, or accessor signatures). The codegen pipeline always
	// emits "reflect" and "github.com/gonewx/godi" on its own; Imports carries
	// only the user-supplied service packages. Order is preserved and
	// duplicates deduped at emit time.
	//
	// D3 caveat (AD-7): stringly-typed import paths are an emit input, not the
	// canonical type-fact source. Story 2.1's go/types integration derives
	// them from Info.Instances and writes here.
	Imports []string

	services []Service
	byName   map[string]Service
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]Service)}
}

// Service registers s and returns the Registry for chaining.
func (r *Registry) Service(s Service) *Registry {
	r.services = append(r.services, s)
	r.byName[s.Name] = s
	return r
}

// Services returns all registered services in registration order.
func (r *Registry) Services() []Service {
	out := make([]Service, len(r.services))
	copy(out, r.services)
	return out
}

// Lookup returns the service registered under name, or false if none.
func (r *Registry) Lookup(name string) (Service, bool) {
	s, ok := r.byName[name]
	return s, ok
}

// SingletonsTopo returns the registered singletons in dependency order: every
// service appears AFTER each singleton it depends on, so emitNewRoot can wire
// them top-to-bottom and a dependency's field (r.f_<Dep>) is always already
// constructed when used as a constructor argument (Story 2.1, AD-8).
//
// The order is a deterministic depth-first post-order over registration order:
// services are visited in the order they were registered; for each, its deps are
// recursively emitted first, then the service itself. A visited set dedups
// shared dependencies (the diamond case) so each singleton appears exactly once.
// With no dependencies the result equals registration order — a precondition for
// byte-identical regeneration (AD-14).
//
// Precondition: Validate has passed (no missing deps, no cycles). Cycle
// detection is Epic 5.3; this method assumes an acyclic graph. A cyclic graph
// would not infinitely loop here (the visited set bounds it) but would emit a
// dependency after its dependent — that bad graph is Validate's job to reject.
func (r *Registry) SingletonsTopo() []Service {
	visited := make(map[string]struct{}, len(r.services))
	out := make([]Service, 0, len(r.services))

	var visit func(s Service)
	visit = func(s Service) {
		if _, done := visited[s.Name]; done {
			return
		}
		visited[s.Name] = struct{}{}
		for _, dep := range s.Deps {
			d, ok := r.byName[dep]
			if !ok || d.Lifetime != Singleton {
				continue
			}
			visit(d)
		}
		out = append(out, s)
	}

	for _, s := range r.services {
		if s.Lifetime != Singleton {
			continue
		}
		visit(s)
	}
	return out
}

// ScopedTopo returns the registered scoped services in dependency order: every
// scoped service appears AFTER each scoped service it depends on, so CreateScope
// can wire them top-to-bottom. Singleton dependencies are NOT included (they are
// wired separately in NewRoot and accessed via s.root.f_X).
//
// The algorithm mirrors SingletonsTopo: deterministic depth-first post-order
// over registration order, restricted to Scoped lifetime services. A visited
// set dedups shared dependencies.
//
// Precondition: Validate has passed (no missing deps, no cycles).
func (r *Registry) ScopedTopo() []Service {
	visited := make(map[string]struct{}, len(r.services))
	out := make([]Service, 0, len(r.services))

	var visit func(s Service)
	visit = func(s Service) {
		if _, done := visited[s.Name]; done {
			return
		}
		visited[s.Name] = struct{}{}
		for _, dep := range s.Deps {
			d, ok := r.byName[dep]
			if !ok || d.Lifetime != Scoped {
				continue
			}
			visit(d)
		}
		out = append(out, s)
	}

	for _, s := range r.services {
		if s.Lifetime != Scoped {
			continue
		}
		visit(s)
	}
	return out
}

// Transients returns all registered transient services in registration order.
// Transients have no dependency-ordering need (each New*() is independent), but
// a stable enumeration order is required for deterministic code generation (AD-14).
func (r *Registry) Transients() []Service {
	out := make([]Service, 0)
	for _, s := range r.services {
		if s.Lifetime == Transient {
			out = append(out, s)
		}
	}
	return out
}
