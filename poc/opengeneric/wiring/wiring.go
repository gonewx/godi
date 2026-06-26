// Package wiring holds the application-specific registration helpers plus the
// two route demonstrations (explicit vs codegen) for open-generic DI.
package wiring

import (
	"godipoc/opengeneric/domain"
	"godipoc/opengeneric/godi"
)

// AddLogger registers the shared non-generic dependency (Tier 1).
func AddLogger(c *godi.Container, prefix string) {
	godi.Register[domain.Logger](c, func(*godi.Container) domain.Logger {
		return domain.Logger{Prefix: prefix}
	})
}

// AddRepository is the Go-idiomatic equivalent of dotnet's open-generic
// registration, materialised for ONE closed element type T. Both routes call
// this; they differ only in WHO enumerates the element types:
//
//   - Route A: a human writes AddRepository[X] per type, by hand.
//   - Route B: a generator emits AddRepository[X] from wiring/manifest.txt.
//
// The dependency (Logger) is resolved through the container, so this composes
// with ordinary constructor injection.
func AddRepository[T domain.Entity](c *godi.Container) {
	godi.Register[domain.IRepository[T]](c, func(cc *godi.Container) domain.IRepository[T] {
		return domain.NewRepository[T](godi.MustGet[domain.Logger](cc))
	})
}
