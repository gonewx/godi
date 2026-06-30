package plan

import (
	"fmt"
	"strings"
)

// reservedMethodNames are Provider interface methods (AD-16) emitted on every
// generated Root/scope struct. A service whose Name matches one of these would
// produce a typed accessor that redeclares the method with a different
// signature, breaking compilation.
var reservedMethodNames = map[string]struct{}{
	"GetService":       {},
	"IsService":        {},
	"GetKeyedService":  {},
	"GetKeyedServices": {},
	"CreateScope":      {},
	"Close":            {},
}

// Validate checks the registry for structural errors: duplicate/reserved names,
// missing dependencies (AD-3, Story 5-1), captive dependencies (AD-3,
// Story 5-2), and circular dependencies (AD-3, Story 5-3).
//
// Errors are aggregated into []error — all violations are reported in one
// pass, not fail-on-first (Consistency Conventions).
func (r *Registry) Validate() []error {
	var errs []error

	seen := make(map[string]struct{}, len(r.services)+len(r.keyedServices))
	allServices := append(r.services[:len(r.services):len(r.services)], r.keyedServices...)
	for _, s := range allServices {
		if _, dup := seen[s.Name]; dup {
			errs = append(errs, fmt.Errorf(
				"duplicate service name: %q registered more than once",
				s.Name,
			))
		}
		if _, reserved := reservedMethodNames[s.Name]; reserved {
			errs = append(errs, fmt.Errorf(
				"reserved service name: %q collides with Provider method; choose a different implementation name",
				s.Name,
			))
		}
		seen[s.Name] = struct{}{}
	}

	for _, s := range allServices {
		visited := map[string]struct{}{s.Name: {}}
		r.walkMissing(s.Name, s.Lifetime, s, []string{s.Name}, visited, &errs)
	}

	for _, s := range allServices {
		if s.Lifetime != Singleton {
			continue
		}
		visited := map[string]struct{}{s.Name: {}}
		r.walkCaptive(s.Name, s, []string{s.Name}, visited, &errs)
	}

	cycleState := make(map[string]int, len(allServices))
	for _, s := range allServices {
		if cycleState[s.Name] == 0 {
			r.walkCycle(s.Name, cycleState, nil, &errs)
		}
	}

	return errs
}

func (r *Registry) walkMissing(rootName string, rootLifetime Lifetime, current Service, chain []string, visited map[string]struct{}, errs *[]error) {
	for _, dep := range current.Deps {
		if strings.HasPrefix(dep, "[]") {
			goType := dep[2:]
			if len(r.byGoType[goType]) == 0 {
				if _, seen := visited[dep]; seen {
					continue
				}
				visited[dep] = struct{}{}
				fullChain := append(chain[:len(chain):len(chain)], dep)
				*errs = append(*errs, fmt.Errorf(
					"missing dependency: %s %q cannot be constructed: %s (%s is not registered)",
					rootLifetime, rootName, strings.Join(fullChain, " → "), dep,
				))
				continue
			}
			for _, implName := range r.byGoType[goType] {
				if _, seen := visited[implName]; seen {
					continue
				}
				visited[implName] = struct{}{}
				nextChain := append(chain[:len(chain):len(chain)], implName)
				if implSvc, ok := r.byName[implName]; ok {
					r.walkMissing(rootName, rootLifetime, implSvc, nextChain, visited, errs)
				}
			}
		} else {
			if _, seen := visited[dep]; seen {
				continue
			}
			visited[dep] = struct{}{}
			depSvc, ok := r.byName[dep]
			if !ok {
				fullChain := append(chain[:len(chain):len(chain)], dep)
				*errs = append(*errs, fmt.Errorf(
					"missing dependency: %s %q cannot be constructed: %s (%s is not registered)",
					rootLifetime, rootName, strings.Join(fullChain, " → "), dep,
				))
				continue
			}
			nextChain := append(chain[:len(chain):len(chain)], dep)
			r.walkMissing(rootName, rootLifetime, depSvc, nextChain, visited, errs)
		}
	}
}

func (r *Registry) walkCaptive(rootName string, current Service, chain []string, visited map[string]struct{}, errs *[]error) {
	for _, dep := range current.Deps {
		if strings.HasPrefix(dep, "[]") {
			goType := dep[2:]
			for _, implName := range r.byGoType[goType] {
				if _, seen := visited[implName]; seen {
					continue
				}
				visited[implName] = struct{}{}
				implSvc, ok := r.byName[implName]
				if !ok {
					continue
				}
				if implSvc.Lifetime == Scoped {
					fullChain := append(chain[:len(chain):len(chain)], implName)
					*errs = append(*errs, fmt.Errorf(
						"captive dependency: Singleton %q captures Scoped %q: %s (Scoped %q would be captured by Singleton %q)",
						rootName, implName, strings.Join(fullChain, " → "), implName, rootName,
					))
					continue
				}
				nextChain := append(chain[:len(chain):len(chain)], implName)
				r.walkCaptive(rootName, implSvc, nextChain, visited, errs)
			}
			continue
		}
		if _, seen := visited[dep]; seen {
			continue
		}
		visited[dep] = struct{}{}
		depSvc, ok := r.byName[dep]
		if !ok {
			continue
		}
		if depSvc.Lifetime == Scoped {
			fullChain := append(chain[:len(chain):len(chain)], dep)
			*errs = append(*errs, fmt.Errorf(
				"captive dependency: Singleton %q captures Scoped %q: %s (Scoped %q would be captured by Singleton %q)",
				rootName, dep, strings.Join(fullChain, " → "), dep, rootName,
			))
			continue
		}
		nextChain := append(chain[:len(chain):len(chain)], dep)
		r.walkCaptive(rootName, depSvc, nextChain, visited, errs)
	}
}

func (r *Registry) walkCycle(name string, state map[string]int, stack []string, errs *[]error) {
	switch state[name] {
	case 1:
		for i, n := range stack {
			if n == name {
				cyclePath := make([]string, len(stack)-i+1)
				copy(cyclePath, stack[i:])
				cyclePath[len(cyclePath)-1] = name
				*errs = append(*errs, fmt.Errorf(
					"circular dependency: %s", strings.Join(cyclePath, " → "),
				))
				return
			}
		}
		return
	case 2:
		return
	}

	state[name] = 1
	stack = append(stack, name)

	s, ok := r.byName[name]
	if !ok {
		state[name] = 2
		return
	}

	for _, dep := range s.Deps {
		if strings.HasPrefix(dep, "[]") {
			goType := dep[2:]
			for _, implName := range r.byGoType[goType] {
				r.walkCycle(implName, state, stack, errs)
			}
		} else {
			if _, exists := r.byName[dep]; !exists {
				continue
			}
			r.walkCycle(dep, state, stack, errs)
		}
	}

	state[name] = 2
}
