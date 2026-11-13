package lazynamedfactory

import (
	"testing"

	"github.com/gonewx/godi"
)

// The named factory newFoo is registered before *Cfg and resolves it through the
// Provider; the generator cannot see that dependency, so construction order must
// come from first access, not from a static order.
func TestNamedFactorySeesLaterRegisteredDependency(t *testing.T) {
	root, err := NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	if godi.GetRequired[IFoo](root).Cfg() == nil {
		t.Fatal("factory saw a nil *Cfg: it ran before the singleton it resolves was built")
	}
}
