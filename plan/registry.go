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
	Name       string
	GoType     string
	Ctor       string
	Lifetime   Lifetime
	Deps       []string
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
