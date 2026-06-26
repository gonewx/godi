package wiring

import (
	"testing"

	"godipoc/opengeneric/domain"
	"godipoc/opengeneric/godi"
)

// BenchmarkResolve_ReflectContainer measures resolution through the reflect
// container (Routes A and B1 share this path): a reflect.Type-keyed map lookup.
func BenchmarkResolve_ReflectContainer(b *testing.B) {
	c := godi.New()
	AddLogger(c, "")
	RegisterGeneratedRepositories(c)
	_, _ = godi.Get[domain.IRepository[domain.User]](c) // warm singleton cache

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, _ := godi.Get[domain.IRepository[domain.User]](c)
		_ = r
	}
}

// BenchmarkResolve_DirectWiring measures the generated reflection-free direct
// wiring (Route B2): a plain struct-field load, the codegen performance payoff.
func BenchmarkResolve_DirectWiring(b *testing.B) {
	app := NewAppRepositories(domain.Logger{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := app.User
		_ = r
	}
}
