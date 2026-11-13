package lazyscopedorder

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/gonewx/godi"
)

// resolvePanic resolves T on p and returns the recovered panic value (nil when
// resolution succeeded).
func resolvePanic[T any](p godi.Provider) (v any) {
	defer func() { v = recover() }()
	_ = godi.GetRequired[T](p)
	return nil
}

// Scoped services are built on first resolution within their scope — neither
// NewRoot nor CreateScope builds any — and a scope disposes them in reverse
// resolution order. A failing, unrelated scoped constructor only fails its own
// resolution, and a retry re-runs the constructor instead of reporting a cycle.
func TestConstructsOnDemandAndDisposesInReverseResolutionOrder(t *testing.T) {
	Built, Disposed, FailC = nil, nil, false

	root, err := NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	if len(Built) != 0 {
		t.Fatalf("NewRoot built scoped services: %v", Built)
	}
	sc := root.CreateScope()
	if len(Built) != 0 {
		t.Fatalf("CreateScope built scoped services: %v", Built)
	}
	_ = godi.GetRequired[IB](sc)
	_ = godi.GetRequired[IA](sc)
	_ = godi.GetRequired[IA](sc)
	if want := []string{"B", "A"}; !reflect.DeepEqual(Built, want) {
		t.Fatalf("Built = %v, want %v", Built, want)
	}
	if err := sc.Close(); err != nil {
		t.Fatalf("scope Close: %v", err)
	}
	if want := []string{"A", "B"}; !reflect.DeepEqual(Disposed, want) {
		t.Fatalf("Disposed = %v, want %v", Disposed, want)
	}

	FailC = true
	var s2 godi.Scope
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatalf("CreateScope panicked although no service was resolved: %v", rec)
			}
		}()
		s2 = root.CreateScope()
	}()
	defer s2.Close()
	if v := resolvePanic[IA](s2); v != nil {
		t.Fatalf("resolving A panicked because of unrelated C: %v", v)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		v := resolvePanic[IC](s2)
		if v == nil {
			t.Fatalf("attempt %d: resolving C with FailC=true did not panic", attempt)
		}
		if got := fmt.Sprint(v); got != "C unavailable" {
			t.Fatalf("attempt %d: panic = %q, want %q", attempt, got, "C unavailable")
		}
		if e, ok := v.(error); ok && errors.Is(e, godi.ErrCircularDependency) {
			t.Fatalf("attempt %d: retry reported a cycle: %v", attempt, e)
		}
	}
}
