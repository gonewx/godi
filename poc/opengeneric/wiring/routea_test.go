package wiring

import (
	"testing"

	"godipoc/opengeneric/domain"
	"godipoc/opengeneric/godi"
)

// TestRouteA_Explicit demonstrates the explicit-closed-registration route:
// a human enumerates each closed generic by hand via AddRepository[X]. Verbose
// but fully type-safe, with no reflection involved in instantiation.
func TestRouteA_Explicit(t *testing.T) {
	c := godi.New()
	AddLogger(c, "[A] ")

	// --- manual enumeration: the Go-idiomatic open-generic equivalent ---
	AddRepository[domain.User](c)
	AddRepository[domain.Order](c)

	userRepo, err := godi.Get[domain.IRepository[domain.User]](c)
	if err != nil {
		t.Fatalf("resolve User repo: %v", err)
	}
	userRepo.Add(domain.User{ID: 1, Name: "Ada"})
	if got := len(userRepo.All()); got != 1 {
		t.Fatalf("want 1 user, got %d", got)
	}

	orderRepo, err := godi.Get[domain.IRepository[domain.Order]](c)
	if err != nil {
		t.Fatalf("resolve Order repo: %v", err)
	}
	orderRepo.Add(domain.Order{ID: 9, Total: 100})

	t.Logf("Route A OK | %s | %s", userRepo.Describe(), orderRepo.Describe())
}
