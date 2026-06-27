package runtime

import (
	"reflect"
	"sync"

	"github.com/gonewx/godi"
)

// root is the reflect-free runtime provider — the compliance oracle (AD-14).
// This story delivers its minimal slice: a single SINGLETON resolved by a
// form-B closure, an honest miss for unregistered types, and an idempotent
// Close. Scoped/transient lifetimes, IEnumerable, keyed services and the LIFO
// disposer are out of scope (Epic 2/3, Story 2.3).
//
// Lifetime is encoded by WHERE the instance lives (AD-1): a singleton is a Root
// field, so it is cached once in the singletons map and every resolve returns
// that same cached value. reflect.Type is used ONLY as a dispatch key (Tier B,
// AD-13); construction is performed by the closure (`&Logger{}`), never by
// reflect.New / reflect.Value.Call.
type root struct {
	// singletons holds every singleton instance, eagerly materialized at Build()
	// time (D2). reflect.Type is the dispatch key (Tier B, AD-13); the stored
	// value is whatever the closure constructed.
	singletons map[reflect.Type]any
	// closeOnce gates Close so the release body runs at most once (D4). This
	// story builds no LIFO disposer (Story 2.3) — the gate is the seam for it.
	closeOnce sync.Once
}

// Compile-time proof that root honors the full six-method AD-16 contract (R2 /
// AC4). This naming the unexported concrete type is the most direct integration
// check that the runtime provider satisfies godi.Provider.
var _ godi.Provider = (*root)(nil)

// GetService returns the singleton registered for t, or nil on a miss (AD-16).
// reflect.Type is only the map dispatch key; no reflective construction happens
// here (AD-13).
func (r *root) GetService(t reflect.Type) any {
	return r.singletons[t]
}

// IsService reports whether t resolves to a registered singleton (CAP-6). It is
// self-consistent with GetService and Get[T]'s miss semantics (AD-16).
func (r *root) IsService(t reflect.Type) bool {
	_, ok := r.singletons[t]
	return ok
}

// GetKeyedService routes a nil key to the unkeyed service (AD-17: nil key =
// unkeyed). A non-nil key always misses this epoch — keyed registration is
// Epic 3, so there is no keyed service to return and we honestly report nil
// rather than pretending support.
func (r *root) GetKeyedService(t reflect.Type, key any) any {
	if key == nil {
		return r.GetService(t)
	}
	return nil
}

// GetKeyedServices returns nil: keyed collections are Epic 3 (CAP-4 / CAP-5).
func (r *root) GetKeyedServices(t reflect.Type, key any) []any {
	return nil
}

// CreateScope opens a minimal child scope (D3). Singletons are shared across
// scopes (AD-1), so the scope simply delegates resolution to root; its Close is
// an idempotent no-op (a singleton is released only with root, never when a
// child scope closes — AD-12). The real scoped lifetime (scope fields, AddScoped)
// is Epic 2.2; returning a delegating scope rather than root itself keeps
// scope.Close from being mistaken for root.Close.
func (r *root) CreateScope() godi.Scope {
	return &scope{root: r}
}

// Close releases this provider. It is idempotent via a one-shot gate (D4): the
// body runs at most once and every call returns nil. This story tracks no
// disposables (Story 2.3 adds the LIFO disposer), so the body is currently
// empty — the gate is what AC3 pins.
func (r *root) Close() error {
	r.closeOnce.Do(func() {
		// LIFO disposer / track / closeAll lands in Story 2.3 (AD-11 / AD-12).
		// The one-shot gate is in place so that release runs at most once.
	})
	return nil
}

// scope is the minimal child scope (D3). It owns no services of its own this
// epoch: it delegates every resolution to root so singletons are shared (AD-1),
// and its Close is an idempotent no-op that must NOT release root's singletons
// (AD-12). The real scoped lifetime is Epic 2.2.
type scope struct {
	root *root
}

var _ godi.Scope = (*scope)(nil)

func (s *scope) GetService(t reflect.Type) any { return s.root.GetService(t) }

func (s *scope) IsService(t reflect.Type) bool { return s.root.IsService(t) }

func (s *scope) GetKeyedService(t reflect.Type, key any) any {
	return s.root.GetKeyedService(t, key)
}

func (s *scope) GetKeyedServices(t reflect.Type, key any) []any {
	return s.root.GetKeyedServices(t, key)
}

func (s *scope) CreateScope() godi.Scope { return &scope{root: s.root} }

// Close is an idempotent no-op: closing a child scope must not release root's
// singletons (AD-12). Scoped disposables arrive with the real scope in Epic 2.
func (s *scope) Close() error { return nil }

// Builder accumulates form-B closure registrations and materializes them into a
// runtime provider (D1). It is the runtime's OWN registration API — distinct
// from the root godi package's static AddSingleton marker (AD-7), which is
// codegen-only and never executed at runtime. The runtime provider eats form-B
// closures (`func(godi.Provider) T`), never the form-A static marker (AD-14).
type Builder struct {
	// factories holds each singleton's closure keyed by its reflect.Type. The
	// typed factory is wrapped into func(godi.Provider) any so the heterogeneous
	// registrations share one map (Tier B: reflect is only the key, AD-13).
	factories map[reflect.Type]func(godi.Provider) any
}

// NewBuilder returns an empty Builder ready for AddSingleton registrations.
func NewBuilder() *Builder {
	return &Builder{factories: map[reflect.Type]func(godi.Provider) any{}}
}

// AddSingleton registers a singleton of type T built by factory (form B). It is
// a free generic FUNCTION, not a method (AD-5): T's reflect.Type is the dispatch
// key (reflect.TypeFor[T], Tier B) and the typed factory is wrapped into
// func(godi.Provider) any. The factory carries a godi.Provider parameter as the
// seam for Epic 2.1 dependency injection; this story's singletons have no
// dependencies, so factories may ignore it. Returns b so registrations chain.
func AddSingleton[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	b.factories[reflect.TypeFor[T]()] = func(p godi.Provider) any {
		return factory(p)
	}
	return b
}

// Build materializes the registered singletons and returns the runtime provider
// as a godi.Provider. Materialization is EAGER (D2): every singleton factory is
// executed exactly once here and the instance is cached, aligning with AD-1
// (singleton = Root field) and NFR8 (the root singleton graph is constructed
// once in NewRoot, with no per-resolve reconstruction and no concurrent
// construction). After Build, GetService is a pure map lookup.
func (b *Builder) Build() godi.Provider {
	r := &root{singletons: make(map[reflect.Type]any, len(b.factories))}
	for t, factory := range b.factories {
		r.singletons[t] = factory(r)
	}
	return r
}
