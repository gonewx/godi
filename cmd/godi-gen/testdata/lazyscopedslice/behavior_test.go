package lazyscopedslice

import (
	"testing"

	"github.com/gonewx/godi"
)

// A scoped consumer of []IPlugin registered BEFORE the scoped plugins must see
// both built plugins, on a scope and on Root (the root scope), not nil elements.
func TestSliceDependencyRegisteredAfterConsumer(t *testing.T) {
	root, err := NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	sc := root.CreateScope()
	defer sc.Close()
	for name, p := range map[string]godi.Provider{"scope": sc, "root": root} {
		items := godi.GetRequired[IConsumer](p).Items()
		if len(items) != 2 {
			t.Fatalf("%s: len(Items()) = %d, want 2", name, len(items))
		}
		for i, it := range items {
			if it == nil {
				t.Fatalf("%s: Items()[%d] is nil", name, i)
			}
		}
	}
}
