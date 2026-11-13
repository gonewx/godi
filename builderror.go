package godi

import (
	"errors"
	"fmt"
)

// ErrCircularDependency is the sentinel a generated container panics with
// (wrapped by CircularDependencyError) when a service is requested while it is
// still being constructed. Form-B factories are opaque to the generator, so a
// cycle that passes through them can only be detected at construction time.
var ErrCircularDependency = errors.New("godi: circular dependency")

// CircularDependencyError builds the panic value a generated lazy accessor uses
// when it is re-entered for service before that service finished constructing.
// Generated code calls this helper instead of building the error itself, so the
// emitted file needs no fmt/errors import (mirrors ProviderClosedError).
func CircularDependencyError(service string) error {
	return fmt.Errorf("%w: %s is already being constructed", ErrCircularDependency, service)
}

// BuildError converts a value recovered from a panic during a generated
// NewRoot into the error NewRoot returns. An error value stays reachable through
// errors.Is / errors.As; any other value is formatted with %v.
func BuildError(recovered any) error {
	if e, ok := recovered.(error); ok {
		return fmt.Errorf("godi: container build failed: %w", e)
	}
	return fmt.Errorf("godi: container build failed: %v", recovered)
}
