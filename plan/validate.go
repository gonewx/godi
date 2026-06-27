package plan

import "fmt"

// Validate checks the registry for structural errors. This story implements
// the minimum slice: missing-dependency detection (AD-3). Captured-dependency
// (Epic 5.2) and circular-dependency (Epic 5.3) checks land later.
//
// Errors are aggregated into []error — all violations are reported in one
// pass, not fail-on-first (Consistency Conventions).
func (r *Registry) Validate() []error {
	var errs []error

	seen := make(map[string]struct{}, len(r.services))
	for _, s := range r.services {
		if _, dup := seen[s.Name]; dup {
			errs = append(errs, fmt.Errorf(
				"duplicate service name: %q registered more than once",
				s.Name,
			))
		}
		seen[s.Name] = struct{}{}
	}

	for _, s := range r.services {
		for _, dep := range s.Deps {
			if _, ok := r.byName[dep]; !ok {
				errs = append(errs, fmt.Errorf(
					"missing dependency: %s %q depends on unregistered %q",
					s.Lifetime, s.Name, dep,
				))
			}
		}
	}
	return errs
}
