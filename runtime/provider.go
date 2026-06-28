package runtime

import (
	"fmt"
	"io"
	"maps"
	"reflect"
	"sync"

	"github.com/gonewx/godi"
)

// root is the reflect-free runtime provider — the compliance oracle (AD-14).
// It supports three lifetimes (AD-1):
//   - Singleton: cached once in root, shared across all scopes
//   - Scoped: cached per scope layer (root = rootScoped, child = scope.scoped)
//   - Transient: factory executed every time, never cached (AD-14)
//
// reflect.Type is used ONLY as a dispatch key (Tier B, AD-13); construction
// is performed by the closure, never by reflect.New / reflect.Value.Call.
//
// Materialization is lazy-memoize (D-3): GetService builds on first access
// and caches (singletons in root, scoped in the resolving layer). Transients
// never cache.
type root struct {
	singletonFactories map[reflect.Type]func(godi.Provider) any
	scopedFactories    map[reflect.Type]func(godi.Provider) any
	transientFactories map[reflect.Type]func(godi.Provider) any

	singletons map[reflect.Type]any
	rootScoped map[reflect.Type]any

	disp      *Disposer
	closeOnce sync.Once
	closeErr  error
}

var _ godi.Provider = (*root)(nil)

// GetService returns the service registered for t, or nil on a miss (AD-16).
// Dispatch by lifetime (D-5):
//   - Singleton: lazy-memoize in root (shared across scopes)
//   - Scoped: lazy-memoize in rootScoped (root acts as root scope, D-2)
//   - Transient: factory every time, never cached (AD-14)
func (r *root) GetService(t reflect.Type) any {
	if v, ok := r.singletons[t]; ok {
		return v
	}
	if factory, ok := r.singletonFactories[t]; ok {
		v := factory(r)
		r.singletons[t] = v
		if c, ok := v.(io.Closer); ok {
			r.disp.Track(c)
		}
		return v
	}
	if v, ok := r.rootScoped[t]; ok {
		return v
	}
	if factory, ok := r.scopedFactories[t]; ok {
		v := factory(r)
		r.rootScoped[t] = v
		if c, ok := v.(io.Closer); ok {
			r.disp.Track(c)
		}
		return v
	}
	if factory, ok := r.transientFactories[t]; ok {
		v := factory(r)
		if c, ok := v.(io.Closer); ok {
			r.disp.Track(c)
		}
		return v
	}
	return nil
}

func (r *root) IsService(t reflect.Type) bool {
	if _, ok := r.singletonFactories[t]; ok {
		return true
	}
	if _, ok := r.scopedFactories[t]; ok {
		return true
	}
	if _, ok := r.transientFactories[t]; ok {
		return true
	}
	return false
}

func (r *root) GetKeyedService(t reflect.Type, key any) any {
	if key == nil {
		return r.GetService(t)
	}
	return nil
}

func (r *root) GetKeyedServices(t reflect.Type, key any) []any {
	return nil
}

func (r *root) CreateScope() godi.Scope {
	return &scope{
		root:   r,
		scoped: make(map[reflect.Type]any),
		disp:   NewDisposer(),
	}
}

func (r *root) Close() error {
	r.closeOnce.Do(func() {
		r.closeErr = r.disp.CloseAll()
	})
	return r.closeErr
}

// scope is a child scope with its own scoped cache. Singletons are shared via
// root; scoped services are independently lazy-memoized per scope (D-2);
// transients are created fresh every time (AD-14).
type scope struct {
	root      *root
	scoped    map[reflect.Type]any
	disp      *Disposer
	closeOnce sync.Once
	closeErr  error
}

var _ godi.Scope = (*scope)(nil)

// GetService dispatches by lifetime from the scope's perspective (D-5):
//   - Singleton: delegate to root (shared)
//   - Scoped: lazy-memoize in THIS scope's cache (independent from root/siblings)
//   - Transient: factory every time, passing THIS scope as provider (AC2)
func (s *scope) GetService(t reflect.Type) any {
	if v, ok := s.root.singletons[t]; ok {
		return v
	}
	if factory, ok := s.root.singletonFactories[t]; ok {
		v := factory(s.root)
		s.root.singletons[t] = v
		if c, ok := v.(io.Closer); ok {
			s.root.disp.Track(c)
		}
		return v
	}
	if v, ok := s.scoped[t]; ok {
		return v
	}
	if factory, ok := s.root.scopedFactories[t]; ok {
		v := factory(s)
		s.scoped[t] = v
		if c, ok := v.(io.Closer); ok {
			s.disp.Track(c)
		}
		return v
	}
	if factory, ok := s.root.transientFactories[t]; ok {
		v := factory(s)
		if c, ok := v.(io.Closer); ok {
			s.disp.Track(c)
		}
		return v
	}
	return nil
}

func (s *scope) IsService(t reflect.Type) bool { return s.root.IsService(t) }

func (s *scope) GetKeyedService(t reflect.Type, key any) any {
	if key == nil {
		return s.GetService(t)
	}
	return nil
}

func (s *scope) GetKeyedServices(t reflect.Type, key any) []any {
	return nil
}

func (s *scope) CreateScope() godi.Scope {
	return &scope{
		root:   s.root,
		scoped: make(map[reflect.Type]any),
		disp:   NewDisposer(),
	}
}

func (s *scope) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = s.disp.CloseAll()
	})
	return s.closeErr
}

// Builder accumulates form-B closure registrations for all three lifetimes
// and materializes them into a runtime provider.
type Builder struct {
	singletonFactories map[reflect.Type]func(godi.Provider) any
	scopedFactories    map[reflect.Type]func(godi.Provider) any
	transientFactories map[reflect.Type]func(godi.Provider) any
}

func NewBuilder() *Builder {
	return &Builder{
		singletonFactories: make(map[reflect.Type]func(godi.Provider) any),
		scopedFactories:    make(map[reflect.Type]func(godi.Provider) any),
		transientFactories: make(map[reflect.Type]func(godi.Provider) any),
	}
}

// AddSingleton registers a singleton of type T built by factory (form B).
func AddSingleton[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	b.singletonFactories[reflect.TypeFor[T]()] = func(p godi.Provider) any {
		return factory(p)
	}
	return b
}

// AddScoped registers a scoped service of type T built by factory (form B).
// Scoped instances are cached per scope — same within a scope, distinct across
// scopes (AD-1, D-2).
func AddScoped[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	b.scopedFactories[reflect.TypeFor[T]()] = func(p godi.Provider) any {
		return factory(p)
	}
	return b
}

// AddTransient registers a transient service of type T built by factory (form B).
// Transient instances are never cached — each resolution creates a new instance
// (AD-14).
func AddTransient[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	b.transientFactories[reflect.TypeFor[T]()] = func(p godi.Provider) any {
		return factory(p)
	}
	return b
}

// Build materializes the registered services and returns the runtime provider.
// Singletons are eagerly materialized (D-3 lazy-memoize forced by Build);
// scoped and transient factories are stored for later lazy use.
func (b *Builder) Build() (provider godi.Provider, err error) {
	sf := make(map[reflect.Type]func(godi.Provider) any, len(b.singletonFactories))
	maps.Copy(sf, b.singletonFactories)
	scf := make(map[reflect.Type]func(godi.Provider) any, len(b.scopedFactories))
	maps.Copy(scf, b.scopedFactories)
	tf := make(map[reflect.Type]func(godi.Provider) any, len(b.transientFactories))
	maps.Copy(tf, b.transientFactories)

	r := &root{
		singletonFactories: sf,
		scopedFactories:    scf,
		transientFactories: tf,
		singletons:         make(map[reflect.Type]any, len(sf)),
		rootScoped:         make(map[reflect.Type]any, len(scf)),
		disp:               NewDisposer(),
	}
	defer func() {
		if rec := recover(); rec != nil {
			_ = r.disp.CloseAll()
			provider, err = nil, fmt.Errorf("runtime build failed materializing singletons: %v", rec)
		}
	}()
	for t := range sf {
		_ = r.GetService(t)
	}
	return r, nil
}
