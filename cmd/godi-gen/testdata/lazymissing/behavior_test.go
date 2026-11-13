package lazymissing

import (
	"strings"
	"testing"
)

// A form-B singleton factory that resolves an unregistered service must make
// NewRoot return an error instead of panicking out of it.
func TestMissingDependencyBecomesNewRootError(t *testing.T) {
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
	if err == nil {
		t.Fatal("NewRoot returned nil error for a factory resolving an unregistered service")
	}
	if root != nil {
		t.Fatal("NewRoot must return a nil *Root together with the error")
	}
	if !strings.Contains(err.Error(), "no service registered") {
		t.Fatalf("unexpected error: %v", err)
	}
}
