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

// Validate checks the registry for structural errors. This story implements
// the minimum slice: missing-dependency detection (AD-3). Captured-dependency
// (Epic 5.2) and circular-dependency (Epic 5.3) checks land later.
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
		for _, dep := range s.Deps {
			if strings.HasPrefix(dep, "[]") {
				goType := dep[2:]
				if len(r.byGoType[goType]) == 0 {
					errs = append(errs, fmt.Errorf(
						"missing dependency: %s %q depends on unregistered slice %q",
						s.Lifetime, s.Name, dep,
					))
				}
				continue
			}
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
