// Package plan models a DI registration graph and validates it at BUILD TIME —
// the B2 (codegen) equivalent of dotnet's ValidateOnBuild + ValidateScopes,
// shifted left to run inside the generator BEFORE any wiring is emitted.
//
// In dotnet these checks run at BuildServiceProvider / first-resolve (runtime).
// In the B2 model the generator runs Validate() first; if it returns errors,
// nothing is generated and `go generate` fails — the bug never reaches a binary.
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

// Service is one registration: a name, a lifetime, and the names of the
// services its constructor depends on.
type Service struct {
	Name      string
	Lifetime  Lifetime
	DependsOn []string
}

// Registry is the ordered set of registrations (what a generator builds from a
// manifest/DSL before emitting wiring).
type Registry struct {
	byName map[string]Service
	order  []string
}

func NewRegistry() *Registry { return &Registry{byName: map[string]Service{}} }

func (r *Registry) Add(s Service) *Registry {
	if _, dup := r.byName[s.Name]; !dup {
		r.order = append(r.order, s.Name)
	}
	r.byName[s.Name] = s
	return r
}

// Validate reports every problem in the graph; an empty slice means valid.
func (r *Registry) Validate() []error {
	var errs []error

	// 1. Missing dependency: a DependsOn name that was never registered.
	//    (In real B2 codegen this would also fail to compile; the validator
	//    turns a cryptic build error into a precise, early message.)
	for _, name := range r.order {
		s := r.byName[name]
		for _, dep := range s.DependsOn {
			if _, ok := r.byName[dep]; !ok {
				errs = append(errs, fmt.Errorf(
					"missing dependency: %s %q depends on unregistered %q",
					s.Lifetime, s.Name, dep))
			}
		}
	}

	// 2. Captive dependency: a singleton whose dependency closure reaches a
	//    scoped service (it would outlive its scope). This is dotnet's
	//    ValidateScopes catch. Scoped->singleton and transient->anything are fine.
	for _, name := range r.order {
		s := r.byName[name]
		if s.Lifetime != Singleton {
			continue
		}
		if captive := r.scopedInClosure(s, map[string]bool{}); captive != "" {
			errs = append(errs, fmt.Errorf(
				"captive dependency: singleton %q captures scoped %q (singleton would outlive the scope)",
				s.Name, captive))
		}
	}

	// 3. Dependency cycle.
	for _, name := range r.order {
		if cycle := r.findCycle(name, map[string]int{}, nil); cycle != nil {
			errs = append(errs, fmt.Errorf("dependency cycle: %s", strings.Join(cycle, " -> ")))
			break // one report is enough
		}
	}

	return errs
}

// scopedInClosure returns the name of the first scoped service reachable through
// s's transitive dependencies, or "" if none.
func (r *Registry) scopedInClosure(s Service, seen map[string]bool) string {
	for _, dep := range s.DependsOn {
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

// findCycle does a DFS from start; state: 0 white, 1 gray (on stack), 2 black.
// It returns the cycle path on a back-edge, else nil.
func (r *Registry) findCycle(start string, state map[string]int, stack []string) []string {
	switch state[start] {
	case 1: // back-edge into the current stack: build the cycle path
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
		for _, dep := range s.DependsOn {
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
