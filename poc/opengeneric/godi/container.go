// Package godi is a minimal reflect-based DI container — just enough to compare
// two approaches to dotnet-style open-generic registration in Go.
package godi

import (
	"fmt"
	"reflect"
	"sync"
)

// Container resolves singleton services keyed by their runtime reflect.Type.
type Container struct {
	mu         sync.Mutex
	factories  map[reflect.Type]func(*Container) any
	singletons map[reflect.Type]any
}

func New() *Container {
	return &Container{
		factories:  make(map[reflect.Type]func(*Container) any),
		singletons: make(map[reflect.Type]any),
	}
}

// Register stores a singleton factory for service type T, keyed by
// reflect.TypeFor[T](). T is usually an interface, e.g. IRepository[User].
func Register[T any](c *Container, build func(*Container) T) {
	t := reflect.TypeFor[T]()
	c.mu.Lock()
	c.factories[t] = func(cc *Container) any { return build(cc) }
	c.mu.Unlock()
}

// Get resolves a previously registered service of type T. It returns an error
// when T was never registered: Go cannot instantiate an arbitrary closed
// generic at runtime, so only explicitly-registered closed types resolve.
func Get[T any](c *Container) (T, error) {
	t := reflect.TypeFor[T]()

	c.mu.Lock()
	if v, ok := c.singletons[t]; ok {
		c.mu.Unlock()
		return v.(T), nil
	}
	build, ok := c.factories[t]
	c.mu.Unlock()

	if !ok {
		var zero T
		return zero, fmt.Errorf("godi: no service registered for %v", t)
	}

	// Build with the lock released so factories may resolve their own
	// dependencies recursively without deadlocking on the non-reentrant mutex.
	v := build(c)

	c.mu.Lock()
	c.singletons[t] = v
	c.mu.Unlock()
	return v.(T), nil
}

// MustGet panics on resolution failure; convenient inside factories.
func MustGet[T any](c *Container) T {
	v, err := Get[T](c)
	if err != nil {
		panic(err)
	}
	return v
}
