package wiring

import (
	"testing"

	"godipoc/opengeneric/domain"
	"godipoc/opengeneric/godi"
)

// TestLimitation_NoRuntimeOpenGeneric is the Tier-3 wall.
//
// Product is neither hand-registered (Route A) nor listed in manifest.txt
// (Route B), so IRepository[Product] cannot be resolved. Unlike dotnet — which
// runs Type.MakeGenericType(typeof(Repository<>), typeof(Product)) at runtime —
// Go cannot instantiate Repository[Product] from an open generic. The closed
// type MUST appear in source, written by hand or by the generator.
//
// This is exactly why "register the open generic once, resolve any closed type"
// has no faithful runtime equivalent in Go, and why Go 1.27 does not change it
// (it adds no dynamic type creation / runtime generic instantiation).
func TestLimitation_NoRuntimeOpenGeneric(t *testing.T) {
	c := godi.New()
	AddLogger(c, "[lim] ")
	RegisterGeneratedRepositories(c) // User, Order only

	_, err := godi.Get[domain.IRepository[domain.Product]](c)
	if err == nil {
		t.Fatal("expected resolution to FAIL for unregistered closed generic IRepository[Product]")
	}
	t.Logf("Tier-3 wall confirmed (as expected): %v", err)
}
