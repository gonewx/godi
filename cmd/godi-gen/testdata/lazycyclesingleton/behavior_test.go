package lazycyclesingleton

import (
	"errors"
	"testing"

	"github.com/gonewx/godi"
)

// Two singleton factories that resolve each other form a cycle the generator
// cannot see; NewRoot must return it as an error wrapping
// godi.ErrCircularDependency, together with a nil root.
func TestSingletonCycleBecomesNewRootError(t *testing.T) {
	var (
		root *Root
		err  error
	)
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatalf("NewRoot panicked: %v", rec)
			}
		}()
		root, err = NewRoot()
	}()
	if !errors.Is(err, godi.ErrCircularDependency) {
		t.Fatalf("NewRoot error = %v, want one wrapping godi.ErrCircularDependency", err)
	}
	if root != nil {
		t.Fatal("NewRoot must return a nil *Root together with the error")
	}
}
