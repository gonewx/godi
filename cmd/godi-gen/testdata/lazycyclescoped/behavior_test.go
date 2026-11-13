package lazycyclescoped

import (
	"errors"
	"testing"

	"github.com/gonewx/godi"
)

// Two scoped factories that resolve each other: NewRoot builds no scoped
// service, so it succeeds; resolving one of them on a scope panics with
// godi.ErrCircularDependency, and the scope stays usable for unrelated services.
func TestScopedCyclePanicsOnResolution(t *testing.T) {
	root, err := NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	sc := root.CreateScope()
	defer sc.Close()

	v := func() (v any) {
		defer func() { v = recover() }()
		_ = godi.GetRequired[IA](sc)
		return nil
	}()
	if v == nil {
		t.Fatal("resolving IA did not panic")
	}
	if e, ok := v.(error); !ok || !errors.Is(e, godi.ErrCircularDependency) {
		t.Fatalf("panic = %v, want an error wrapping godi.ErrCircularDependency", v)
	}
	if godi.GetRequired[IC](sc) == nil {
		t.Fatal("unrelated scoped IC is nil after the cycle")
	}
}
