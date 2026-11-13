package lazykeyedscopederr

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gonewx/godi"
)

// Keyed scoped and keyed transient registrations whose constructors return an
// error compile, resolve on a scope and on Root, and turn a constructor error
// into a panic at the resolution site.
func TestKeyedScopedAndTransientConstructorsWithError(t *testing.T) {
	FailBar = false
	root, err := NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	sc := root.CreateScope()
	defer sc.Close()
	for name, p := range map[string]godi.Provider{"scope": sc, "root": root} {
		if godi.GetRequiredKeyed[IFoo](p, "scoped") == nil {
			t.Fatalf("%s: keyed scoped IFoo is nil", name)
		}
		if godi.GetRequiredKeyed[IBar](p, "transient") == nil {
			t.Fatalf("%s: keyed transient IBar is nil", name)
		}
		if got := len(p.GetKeyedServices(reflect.TypeFor[IBar](), godi.AnyKey)); got != 1 {
			t.Fatalf("%s: GetKeyedServices(IBar, AnyKey) has %d items, want 1", name, got)
		}
	}

	FailBar = true
	v := func() (v any) {
		defer func() { v = recover() }()
		_ = godi.GetRequiredKeyed[IBar](sc, "transient")
		return nil
	}()
	if got := fmt.Sprint(v); got != "bar unavailable" {
		t.Fatalf("panic = %v, want %q", v, "bar unavailable")
	}
}
