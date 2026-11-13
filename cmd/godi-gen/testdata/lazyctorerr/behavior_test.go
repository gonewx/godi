package lazyctorerr

import (
	"errors"
	"testing"

	"github.com/gonewx/godi"
)

func TestBuildsThroughErrorReturningConstructors(t *testing.T) {
	FailRepo, ConnClosed = false, false
	root, err := NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	if got := len(godi.GetRequired[IRepo](root).Parts()); got != 2 {
		t.Fatalf("Repo got %d parts, want 2", got)
	}
	if godi.GetRequired[IReq](root).Repo() != godi.GetRequired[IRepo](root) {
		t.Fatal("root-level scoped Req must receive the singleton Repo")
	}
}

// A singleton constructor error makes NewRoot return it (reachable through
// errors.Is) and releases the disposable singleton built before it.
func TestConstructorErrorIsReturnedAndReleasesTracked(t *testing.T) {
	FailRepo, ConnClosed = true, false
	t.Cleanup(func() { FailRepo = false })
	root, err := NewRoot()
	if root != nil || err == nil {
		t.Fatalf("NewRoot = (%v, %v), want (nil, error)", root, err)
	}
	if !errors.Is(err, ErrRepo) {
		t.Fatalf("NewRoot error %v must wrap ErrRepo", err)
	}
	if !ConnClosed {
		t.Fatal("the Conn built before the failure must be released")
	}
}
