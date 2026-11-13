package lazytransientdupdep

import (
	"testing"

	"github.com/gonewx/godi"
)

func TestDuplicateErrorReturningTransientArguments(t *testing.T) {
	root, err := NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	a, b := godi.GetRequired[IFoo](root).Deps()
	if a == b || a.ID() == b.ID() {
		t.Fatalf("Foo's two IX arguments must be distinct instances, got %d and %d", a.ID(), b.ID())
	}
	ea, eb := godi.GetRequired[IErrFoo](root).Deps()
	if ea == eb || ea.ID() == eb.ID() {
		t.Fatalf("ErrFoo's two IX arguments must be distinct instances, got %d and %d", ea.ID(), eb.ID())
	}
}
