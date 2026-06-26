// Package plan is the registration model + build-time validator + singleton
// topological order. A generator builds a Registry (from spec), runs Validate,
// and — only if valid — emits wiring.
package plan

import (
	"fmt"
	"strings"
)

type Lifetime int

const (
	Singleton Lifetime = iota
	Scoped
	Transient
)

func (l Lifetime) String() string {
	switch l {
	case Singleton:
		return "singleton"
	case Scoped:
		return "scoped"
	case Transient:
		return "transient"
	default:
		return "?"
	}
}

// Service is one lifetime-managed registration. GoType and Ctor are source-text
// fragments (e.g. "*domain.DbConnection", "domain.NewDbConnection") because the
// generator emits code from them.
type Service struct {
	Name       string
	GoType     string
	Ctor       string
	Lifetime   Lifetime
	Deps       []string // names of services passed to Ctor, in argument order
	Disposable bool     // true if GoType implements io.Closer and must be disposed
}

// EnumGroup is an IEnumerable[Iface] registration: several impls behind one iface.
type EnumGroup struct {
	Accessor  string   // generated method name, e.g. "Handlers"
	IfaceType string   // e.g. "domain.IHandler"
	Ctors     []string // construction expressions, in registration order
}

// KeyedGroup is a keyed-service registration.
type KeyedGroup struct {
	Accessor  string // generated method name, e.g. "Cache"
	IfaceType string // e.g. "domain.ICache"
	Items     []KeyedItem
}

type KeyedItem struct {
	Key  string
	Ctor string
}

// Registry is the ordered set of registrations.
type Registry struct {
	services []Service
	byName   map[string]Service
	enums    []EnumGroup
	keyed    []KeyedGroup
}

func NewRegistry() *Registry { return &Registry{byName: map[string]Service{}} }

func (r *Registry) Service(s Service) *Registry {
	r.services = append(r.services, s)
	r.byName[s.Name] = s
	return r
}

func (r *Registry) Enum(e EnumGroup) *Registry  { r.enums = append(r.enums, e); return r }
func (r *Registry) Keyed(k KeyedGroup) *Registry { r.keyed = append(r.keyed, k); return r }

func (r *Registry) Services() []Service          { return r.services }
func (r *Registry) Enums() []EnumGroup           { return r.enums }
func (r *Registry) KeyedGroups() []KeyedGroup    { return r.keyed }
func (r *Registry) Lookup(name string) (Service, bool) {
	s, ok := r.byName[name]
	return s, ok
}

// Validate reports every problem; empty slice means valid. This is the B2
// equivalent of dotnet's ValidateOnBuild + ValidateScopes, run at generation
// time so a bad graph never produces code.
func (r *Registry) Validate() []error {
	var errs []error

	for _, s := range r.services {
		for _, dep := range s.Deps {
			if _, ok := r.byName[dep]; !ok {
				errs = append(errs, fmt.Errorf(
					"missing dependency: %s %q depends on unregistered %q", s.Lifetime, s.Name, dep))
			}
		}
	}

	for _, s := range r.services {
		if s.Lifetime != Singleton {
			continue
		}
		if captive := r.scopedInClosure(s, map[string]bool{}); captive != "" {
			errs = append(errs, fmt.Errorf(
				"captive dependency: singleton %q captures scoped %q (singleton would outlive the scope)",
				s.Name, captive))
		}
	}

	for _, s := range r.services {
		if cycle := r.findCycle(s.Name, map[string]int{}, nil); cycle != nil {
			errs = append(errs, fmt.Errorf("dependency cycle: %s", strings.Join(cycle, " -> ")))
			break
		}
	}

	return errs
}

func (r *Registry) scopedInClosure(s Service, seen map[string]bool) string {
	for _, dep := range s.Deps {
		if seen[dep] {
			continue
		}
		seen[dep] = true
		ds, ok := r.byName[dep]
		if !ok {
			continue
		}
		if ds.Lifetime == Scoped {
			return ds.Name
		}
		if found := r.scopedInClosure(ds, seen); found != "" {
			return found
		}
	}
	return ""
}

func (r *Registry) findCycle(start string, state map[string]int, stack []string) []string {
	switch state[start] {
	case 1:
		for i, n := range stack {
			if n == start {
				return append(append([]string{}, stack[i:]...), start)
			}
		}
		return []string{start, start}
	case 2:
		return nil
	}
	state[start] = 1
	stack = append(stack, start)
	if s, ok := r.byName[start]; ok {
		for _, dep := range s.Deps {
			if _, exists := r.byName[dep]; !exists {
				continue
			}
			if c := r.findCycle(dep, state, stack); c != nil {
				return c
			}
		}
	}
	state[start] = 2
	return nil
}

// SingletonsTopo returns the singleton services in dependency order (each
// service after the singletons it depends on), so NewRoot can build fields by
// direct reference. Assumes Validate already passed (acyclic, no captive deps).
func (r *Registry) SingletonsTopo() []Service {
	isSingleton := map[string]bool{}
	for _, s := range r.services {
		if s.Lifetime == Singleton {
			isSingleton[s.Name] = true
		}
	}

	var out []Service
	visited := map[string]bool{}
	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		s := r.byName[name]
		for _, dep := range s.Deps {
			if isSingleton[dep] {
				visit(dep)
			}
		}
		out = append(out, s)
	}
	for _, s := range r.services {
		if s.Lifetime == Singleton {
			visit(s.Name)
		}
	}
	return out
}
