package lazykeyederr

import (
	"testing"

	"github.com/gonewx/godi"
)

// A keyed registration whose constructor returns an error compiles and resolves.
func TestKeyedConstructorWithError(t *testing.T) {
	root, err := NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	if godi.GetRequiredKeyed[IFoo](root, "k") == nil {
		t.Fatal(`GetRequiredKeyed[IFoo](root, "k") is nil`)
	}
}
