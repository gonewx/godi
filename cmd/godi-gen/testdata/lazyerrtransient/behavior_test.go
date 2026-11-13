package lazyerrtransient

import "testing"

// Two singletons sharing an error-returning transient dependency compile and
// build.
func TestSharedErrorTransientDependency(t *testing.T) {
	if _, err := NewRoot(); err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
}
