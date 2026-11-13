package lazykeyeddep

import (
	"testing"

	"github.com/gonewx/godi"
)

// An unkeyed form-B singleton resolves a keyed singleton; the keyed one must be
// built on demand even though keyed singletons are triggered after unkeyed ones.
func TestUnkeyedFactoryResolvesKeyedSingleton(t *testing.T) {
	root, err := NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	if godi.GetRequired[IFoo](root).Dep() == nil {
		t.Fatal("Foo.Dep() is nil")
	}
}
