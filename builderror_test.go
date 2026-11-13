package godi_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/gonewx/godi"
)

// TestBuildError_WrapsErrorsAndValues pins the two helpers a generated NewRoot
// uses to turn a construction-time panic into its returned error: an error
// value stays reachable through errors.Is, a non-error value is formatted.
func TestBuildError_WrapsErrorsAndValues(t *testing.T) {
	cycle := godi.CircularDependencyError("X")
	if !errors.Is(cycle, godi.ErrCircularDependency) {
		t.Fatalf("CircularDependencyError must wrap ErrCircularDependency, got %v", cycle)
	}
	if !strings.Contains(cycle.Error(), "X is already being constructed") {
		t.Fatalf("CircularDependencyError must name the service, got %q", cycle.Error())
	}

	wrapped := godi.BuildError(cycle)
	if !errors.Is(wrapped, godi.ErrCircularDependency) {
		t.Fatalf("BuildError must keep an error value reachable via errors.Is, got %v", wrapped)
	}
	if !strings.HasPrefix(wrapped.Error(), "godi: container build failed: ") {
		t.Fatalf("unexpected BuildError text: %q", wrapped.Error())
	}

	if got := godi.BuildError("boom").Error(); got != "godi: container build failed: boom" {
		t.Fatalf("BuildError(non-error) = %q", got)
	}
}
