// Package domain holds the sample services for the open-generic DI PoC.
//
// IRepository[T] / Repository[T] are the open generic contract + implementation,
// the Go analogue of dotnet's typeof(IRepository<>) -> typeof(Repository<>).
package domain

import "fmt"

// Logger is a plain non-generic dependency injected into every repository
// (exercises ordinary closed-type resolution — Tier 1).
type Logger struct{ Prefix string }

func (l Logger) Log(msg string) string { return l.Prefix + msg }

// Entity constrains repository element types and lets a repository report its
// element name without reflection.
type Entity interface{ EntityName() string }

// User and Order are the element types registered (Route A) / generated (Route B).
type User struct {
	ID   int
	Name string
}

func (User) EntityName() string { return "User" }

type Order struct {
	ID    int
	Total int
}

func (Order) EntityName() string { return "Order" }

// Product is deliberately never registered nor generated; the limitation test
// uses it to show an unregistered closed generic cannot be resolved at runtime.
type Product struct {
	ID  int
	SKU string
}

func (Product) EntityName() string { return "Product" }

// IRepository is the open generic service contract.
type IRepository[T Entity] interface {
	Add(item T)
	All() []T
	Describe() string
}

// Repository is the open generic implementation.
type Repository[T Entity] struct {
	log   Logger
	items []T
}

// NewRepository is the generic constructor; Logger is injected (Tier 1).
func NewRepository[T Entity](log Logger) *Repository[T] {
	return &Repository[T]{log: log}
}

func (r *Repository[T]) Add(item T) { r.items = append(r.items, item) }

func (r *Repository[T]) All() []T { return r.items }

func (r *Repository[T]) Describe() string {
	var zero T
	return fmt.Sprintf("Repository[%s](%d items)%s", zero.EntityName(), len(r.items), r.log.Log(" ok"))
}
