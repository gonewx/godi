package lazyscopedviatransient

import (
	"testing"

	"github.com/gonewx/godi"
)

// Scoped A depends on transient T, which depends on scoped B registered after
// A. Resolving A on a scope must hand T a built B.
func TestDependencyThroughTransient(t *testing.T) {
	root, err := NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	sc := root.CreateScope()
	defer sc.Close()
	if godi.GetRequired[IA](sc).T().Dep() == nil {
		t.Fatal("A.T().Dep() is nil")
	}
}
