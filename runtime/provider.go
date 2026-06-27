package runtime

import (
	"fmt"
	"maps"
	"reflect"
	"sync"

	"github.com/gonewx/godi"
)

// root is the reflect-free runtime provider — the compliance oracle (AD-14).
// Story 2.1 grows it to singletons WITH dependencies: a singleton's factory may
// resolve other singletons through the provider, so wiring must be independent
// of registration / map-iteration order. Scoped/transient lifetimes,
// IEnumerable, keyed services and the LIFO disposer remain out of scope (Epic
// 2.2/2.4, Story 2.3).
//
// Lifetime is encoded by WHERE the instance lives (AD-1): a singleton is a Root
// field, cached once and returned identically on every resolve. reflect.Type is
// used ONLY as a dispatch key (Tier B, AD-13); construction is performed by the
// closure, never by reflect.New / reflect.Value.Call.
//
// Materialization is lazy-memoize (D-3): GetService builds a singleton on first
// access and caches it, so a dependent's factory recursively materializes its
// dependency regardless of map order — fixing the deferred-1.3 bug where eager
// map-order iteration could hand a factory a not-yet-built (nil) dependency.
// Build then forces ALL singletons, so the graph is fully constructed when Build
// returns (eager completion, NFR8). After Build, GetService is a pure cache hit.
type root struct {
	// factories is the lazy source: each singleton's closure keyed by its
	// reflect.Type. GetService consults it on a cache miss.
	factories map[reflect.Type]func(godi.Provider) any
	// singletons is the materialization cache. Build fills it completely; a
	// later GetService is a pure lookup. reflect.Type is the dispatch key (Tier
	// B, AD-13); the stored value is whatever the closure constructed.
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
// On a cache hit it returns the stored instance; on a miss it materializes the
// singleton via its factory and caches it (lazy-memoize, D-3). The factory may
// call GetService for its own dependencies, which recursively materializes them
// — so a dependent built before its dependency still wires correctly, regardless
// of map-iteration order. reflect.Type is only the dispatch key; construction is
// the closure's, never reflective (AD-13).
func (r *root) GetService(t reflect.Type) any {
	if v, ok := r.singletons[t]; ok {
		return v
	}
	factory, ok := r.factories[t]
	if !ok {
		return nil // honest miss: unregistered type (AD-16)
	}
	v := factory(r)
	r.singletons[t] = v
	return v
}

// IsService reports whether t resolves to a registered singleton (CAP-6). It is
// keyed on registration (the factory set), so it is true for a registered type
// whether or not it has been materialized yet, and self-consistent with
// GetService / Get[T]'s miss semantics (AD-16).
func (r *root) IsService(t reflect.Type) bool {
	_, ok := r.factories[t]
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
// as a godi.Provider plus an error (D-1: the signature mirrors the codegen
// NewRoot() (*Root, error) so the conformance suite asserts one shape for both
// providers). Materialization is eager-via-lazy-memoize (D-3): Build forces
// every singleton through root.GetService, which builds each on first access and
// recursively materializes its dependencies — so map-iteration order is harmless
// (a dependent reached first pulls its dependency up). When Build returns with no
// error the whole singleton graph is constructed exactly once (AD-1, NFR8) and
// every subsequent GetService is a pure cache hit.
//
// A factory that panics while constructing (e.g. a missing dependency surfacing
// as a failed type assertion) is recovered into a build error — the seam for the
// runtime side of "bad graph is rejected at Build" (AD-3). Full graph validation
// is Epic 5; this story only needs the error channel to exist and propagate.
func (b *Builder) Build() (provider godi.Provider, err error) {
	// Copy the factory map so the built root is decoupled from the Builder: a
	// post-Build AddSingleton (or building twice) must not retroactively change a
	// root's registered set. Aliasing b.factories would let a later registration
	// make IsService report true for a never-materialized type and have GetService
	// lazily build it — breaking the "after Build, GetService is a pure cache hit"
	// invariant this method documents.
	factories := make(map[reflect.Type]func(godi.Provider) any, len(b.factories))
	maps.Copy(factories, b.factories)
	r := &root{
		factories:  factories,
		singletons: make(map[reflect.Type]any, len(b.factories)),
	}
	defer func() {
		if rec := recover(); rec != nil {
			provider, err = nil, fmt.Errorf("runtime build failed materializing singletons: %v", rec)
		}
	}()
	for t := range b.factories {
		_ = r.GetService(t) // force eager materialization; recursion wires deps
	}
	return r, nil
}
