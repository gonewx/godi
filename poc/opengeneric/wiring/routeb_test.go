package wiring

import (
	"testing"

	"godipoc/opengeneric/domain"
	"godipoc/opengeneric/godi"
)

// TestRouteB1_Codegen_Container uses generated registrations against the reflect
// container. The element enumeration comes from wiring/manifest.txt instead of
// being hand-written — same runtime behaviour as Route A, less boilerplate.
func TestRouteB1_Codegen_Container(t *testing.T) {
	c := godi.New()
	AddLogger(c, "[B1] ")
	RegisterGeneratedRepositories(c) // generated from manifest.txt

	userRepo, err := godi.Get[domain.IRepository[domain.User]](c)
	if err != nil {
		t.Fatalf("resolve User repo: %v", err)
	}
	userRepo.Add(domain.User{ID: 2, Name: "Linus"})
	if got := len(userRepo.All()); got != 1 {
		t.Fatalf("want 1 user, got %d", got)
	}
	t.Logf("Route B1 (codegen + reflect container) OK | %s", userRepo.Describe())
}

// TestRouteB2_Codegen_Direct uses the generated reflection-free direct wiring
// (wire-style): no container, no map, no reflect — fully compile-time safe.
func TestRouteB2_Codegen_Direct(t *testing.T) {
	app := NewAppRepositories(domain.Logger{Prefix: "[B2] "})

	app.User.Add(domain.User{ID: 3, Name: "Rob"})
	app.Order.Add(domain.Order{ID: 7, Total: 42})
	if len(app.User.All()) != 1 || len(app.Order.All()) != 1 {
		t.Fatal("direct wiring failed")
	}
	t.Logf("Route B2 (codegen + direct wiring) OK | %s | %s", app.User.Describe(), app.Order.Describe())
}
