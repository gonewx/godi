package runtime

import (
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/gonewx/godi"
)

type keyedEntry struct {
	t   reflect.Type
	key any
}

func materializeKeyedInto(cache map[keyedEntry][]any, ke keyedEntry, factories []func(godi.Provider) any, provider godi.Provider, disp *Disposer) []any {
	if instances, ok := cache[ke]; ok {
		return instances
	}
	instances := make([]any, len(factories))
	for i, f := range factories {
		v := f(provider)
		if c, ok := v.(io.Closer); ok {
			disp.Track(c)
		}
		instances[i] = v
	}
	cache[ke] = instances
	return instances
}

func materializeInto(cache map[reflect.Type][]any, t reflect.Type, factories []func(godi.Provider) any, provider godi.Provider, disp *Disposer) []any {
	if instances, ok := cache[t]; ok {
		return instances
	}
	instances := make([]any, len(factories))
	for i, f := range factories {
		v := f(provider)
		if c, ok := v.(io.Closer); ok {
			disp.Track(c)
		}
		instances[i] = v
	}
	cache[t] = instances
	return instances
}

type root struct {
	singletonFactories map[reflect.Type][]func(godi.Provider) any
	scopedFactories    map[reflect.Type][]func(godi.Provider) any
	transientFactories map[reflect.Type][]func(godi.Provider) any

	keyedSingletonFactories map[keyedEntry][]func(godi.Provider) any
	keyedScopedFactories    map[keyedEntry][]func(godi.Provider) any
	keyedTransientFactories map[keyedEntry][]func(godi.Provider) any

	singletons      map[reflect.Type][]any
	rootScoped      map[reflect.Type][]any
	keyedSingletons map[keyedEntry][]any
	keyedRootScoped map[keyedEntry][]any

	disp      *Disposer
	closeOnce sync.Once
	closeErr  error
}

var _ godi.Provider = (*root)(nil)

func (r *root) GetService(t reflect.Type) any {
	if factories, ok := r.singletonFactories[t]; ok {
		instances := materializeInto(r.singletons, t, factories, r, r.disp)
		return instances[len(instances)-1]
	}
	if factories, ok := r.scopedFactories[t]; ok {
		instances := materializeInto(r.rootScoped, t, factories, r, r.disp)
		return instances[len(instances)-1]
	}
	if factories, ok := r.transientFactories[t]; ok {
		v := factories[len(factories)-1](r)
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
	ke := keyedEntry{t: t, key: key}
	if factories, ok := r.keyedSingletonFactories[ke]; ok {
		instances := materializeKeyedInto(r.keyedSingletons, ke, factories, r, r.disp)
		return instances[len(instances)-1]
	}
	if factories, ok := r.keyedScopedFactories[ke]; ok {
		instances := materializeKeyedInto(r.keyedRootScoped, ke, factories, r, r.disp)
		return instances[len(instances)-1]
	}
	if factories, ok := r.keyedTransientFactories[ke]; ok {
		v := factories[len(factories)-1](r)
		if c, ok := v.(io.Closer); ok {
			r.disp.Track(c)
		}
		return v
	}
	return nil
}

func (r *root) GetKeyedServices(t reflect.Type, key any) []any {
	if key == nil {
		return r.getUnkeyedServices(t)
	}
	ke := keyedEntry{t: t, key: key}
	if factories, ok := r.keyedSingletonFactories[ke]; ok {
		instances := materializeKeyedInto(r.keyedSingletons, ke, factories, r, r.disp)
		out := make([]any, len(instances))
		copy(out, instances)
		return out
	}
	if factories, ok := r.keyedScopedFactories[ke]; ok {
		instances := materializeKeyedInto(r.keyedRootScoped, ke, factories, r, r.disp)
		out := make([]any, len(instances))
		copy(out, instances)
		return out
	}
	if factories, ok := r.keyedTransientFactories[ke]; ok {
		instances := make([]any, len(factories))
		for i, f := range factories {
			v := f(r)
			if c, ok := v.(io.Closer); ok {
				r.disp.Track(c)
			}
			instances[i] = v
		}
		return instances
	}
	return nil
}

func (r *root) getUnkeyedServices(t reflect.Type) []any {
	if factories, ok := r.singletonFactories[t]; ok {
		instances := materializeInto(r.singletons, t, factories, r, r.disp)
		out := make([]any, len(instances))
		copy(out, instances)
		return out
	}
	if factories, ok := r.scopedFactories[t]; ok {
		instances := materializeInto(r.rootScoped, t, factories, r, r.disp)
		out := make([]any, len(instances))
		copy(out, instances)
		return out
	}
	if factories, ok := r.transientFactories[t]; ok {
		instances := make([]any, len(factories))
		for i, f := range factories {
			v := f(r)
			if c, ok := v.(io.Closer); ok {
				r.disp.Track(c)
			}
			instances[i] = v
		}
		return instances
	}
	return nil
}

func (r *root) CreateScope() godi.Scope {
	return &scope{
		root:        r,
		scoped:      make(map[reflect.Type][]any),
		keyedScoped: make(map[keyedEntry][]any),
		disp:        NewDisposer(),
	}
}

func (r *root) Close() error {
	r.closeOnce.Do(func() {
		r.closeErr = r.disp.CloseAll()
	})
	return r.closeErr
}

type scope struct {
	root        *root
	scoped      map[reflect.Type][]any
	keyedScoped map[keyedEntry][]any
	disp        *Disposer
	closeOnce   sync.Once
	closeErr    error
}

var _ godi.Scope = (*scope)(nil)

func (s *scope) GetService(t reflect.Type) any {
	if factories, ok := s.root.singletonFactories[t]; ok {
		instances := materializeInto(s.root.singletons, t, factories, s.root, s.root.disp)
		return instances[len(instances)-1]
	}
	if factories, ok := s.root.scopedFactories[t]; ok {
		instances := materializeInto(s.scoped, t, factories, s, s.disp)
		return instances[len(instances)-1]
	}
	if factories, ok := s.root.transientFactories[t]; ok {
		v := factories[len(factories)-1](s)
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
	ke := keyedEntry{t: t, key: key}
	if factories, ok := s.root.keyedSingletonFactories[ke]; ok {
		instances := materializeKeyedInto(s.root.keyedSingletons, ke, factories, s.root, s.root.disp)
		return instances[len(instances)-1]
	}
	if factories, ok := s.root.keyedScopedFactories[ke]; ok {
		instances := materializeKeyedInto(s.keyedScoped, ke, factories, s, s.disp)
		return instances[len(instances)-1]
	}
	if factories, ok := s.root.keyedTransientFactories[ke]; ok {
		v := factories[len(factories)-1](s)
		if c, ok := v.(io.Closer); ok {
			s.disp.Track(c)
		}
		return v
	}
	return nil
}

func (s *scope) GetKeyedServices(t reflect.Type, key any) []any {
	if key == nil {
		return s.getUnkeyedServices(t)
	}
	ke := keyedEntry{t: t, key: key}
	if factories, ok := s.root.keyedSingletonFactories[ke]; ok {
		instances := materializeKeyedInto(s.root.keyedSingletons, ke, factories, s.root, s.root.disp)
		out := make([]any, len(instances))
		copy(out, instances)
		return out
	}
	if factories, ok := s.root.keyedScopedFactories[ke]; ok {
		instances := materializeKeyedInto(s.keyedScoped, ke, factories, s, s.disp)
		out := make([]any, len(instances))
		copy(out, instances)
		return out
	}
	if factories, ok := s.root.keyedTransientFactories[ke]; ok {
		instances := make([]any, len(factories))
		for i, f := range factories {
			v := f(s)
			if c, ok := v.(io.Closer); ok {
				s.disp.Track(c)
			}
			instances[i] = v
		}
		return instances
	}
	return nil
}

func (s *scope) getUnkeyedServices(t reflect.Type) []any {
	if factories, ok := s.root.singletonFactories[t]; ok {
		instances := materializeInto(s.root.singletons, t, factories, s.root, s.root.disp)
		out := make([]any, len(instances))
		copy(out, instances)
		return out
	}
	if factories, ok := s.root.scopedFactories[t]; ok {
		instances := materializeInto(s.scoped, t, factories, s, s.disp)
		out := make([]any, len(instances))
		copy(out, instances)
		return out
	}
	if factories, ok := s.root.transientFactories[t]; ok {
		instances := make([]any, len(factories))
		for i, f := range factories {
			v := f(s)
			if c, ok := v.(io.Closer); ok {
				s.disp.Track(c)
			}
			instances[i] = v
		}
		return instances
	}
	return nil
}

func (s *scope) CreateScope() godi.Scope {
	return &scope{
		root:        s.root,
		scoped:      make(map[reflect.Type][]any),
		keyedScoped: make(map[keyedEntry][]any),
		disp:        NewDisposer(),
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
	singletonFactories map[reflect.Type][]func(godi.Provider) any
	scopedFactories    map[reflect.Type][]func(godi.Provider) any
	transientFactories map[reflect.Type][]func(godi.Provider) any

	keyedSingletonFactories map[keyedEntry][]func(godi.Provider) any
	keyedScopedFactories    map[keyedEntry][]func(godi.Provider) any
	keyedTransientFactories map[keyedEntry][]func(godi.Provider) any
}

func NewBuilder() *Builder {
	return &Builder{
		singletonFactories:      make(map[reflect.Type][]func(godi.Provider) any),
		scopedFactories:         make(map[reflect.Type][]func(godi.Provider) any),
		transientFactories:      make(map[reflect.Type][]func(godi.Provider) any),
		keyedSingletonFactories: make(map[keyedEntry][]func(godi.Provider) any),
		keyedScopedFactories:    make(map[keyedEntry][]func(godi.Provider) any),
		keyedTransientFactories: make(map[keyedEntry][]func(godi.Provider) any),
	}
}

func AddSingleton[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	b.singletonFactories[t] = append(b.singletonFactories[t], func(p godi.Provider) any {
		return factory(p)
	})
	return b
}

func AddScoped[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	b.scopedFactories[t] = append(b.scopedFactories[t], func(p godi.Provider) any {
		return factory(p)
	})
	return b
}

func AddTransient[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	b.transientFactories[t] = append(b.transientFactories[t], func(p godi.Provider) any {
		return factory(p)
	})
	return b
}

func AddKeyedSingleton[T any](b *Builder, key any, factory func(godi.Provider) T) *Builder {
	ke := keyedEntry{t: reflect.TypeFor[T](), key: key}
	b.keyedSingletonFactories[ke] = append(b.keyedSingletonFactories[ke], func(p godi.Provider) any {
		return factory(p)
	})
	return b
}

func AddKeyedScoped[T any](b *Builder, key any, factory func(godi.Provider) T) *Builder {
	ke := keyedEntry{t: reflect.TypeFor[T](), key: key}
	b.keyedScopedFactories[ke] = append(b.keyedScopedFactories[ke], func(p godi.Provider) any {
		return factory(p)
	})
	return b
}

func AddKeyedTransient[T any](b *Builder, key any, factory func(godi.Provider) T) *Builder {
	ke := keyedEntry{t: reflect.TypeFor[T](), key: key}
	b.keyedTransientFactories[ke] = append(b.keyedTransientFactories[ke], func(p godi.Provider) any {
		return factory(p)
	})
	return b
}

func copyFactoryMap(m map[reflect.Type][]func(godi.Provider) any) map[reflect.Type][]func(godi.Provider) any {
	out := make(map[reflect.Type][]func(godi.Provider) any, len(m))
	for t, fns := range m {
		cp := make([]func(godi.Provider) any, len(fns))
		copy(cp, fns)
		out[t] = cp
	}
	return out
}

func copyKeyedFactoryMap(m map[keyedEntry][]func(godi.Provider) any) map[keyedEntry][]func(godi.Provider) any {
	out := make(map[keyedEntry][]func(godi.Provider) any, len(m))
	for ke, fns := range m {
		cp := make([]func(godi.Provider) any, len(fns))
		copy(cp, fns)
		out[ke] = cp
	}
	return out
}

func (b *Builder) Build() (provider godi.Provider, err error) {
	sf := copyFactoryMap(b.singletonFactories)
	scf := copyFactoryMap(b.scopedFactories)
	tf := copyFactoryMap(b.transientFactories)
	ksf := copyKeyedFactoryMap(b.keyedSingletonFactories)
	kscf := copyKeyedFactoryMap(b.keyedScopedFactories)
	ktf := copyKeyedFactoryMap(b.keyedTransientFactories)

	r := &root{
		singletonFactories:      sf,
		scopedFactories:         scf,
		transientFactories:      tf,
		keyedSingletonFactories: ksf,
		keyedScopedFactories:    kscf,
		keyedTransientFactories: ktf,
		singletons:              make(map[reflect.Type][]any, len(sf)),
		rootScoped:              make(map[reflect.Type][]any, len(scf)),
		keyedSingletons:         make(map[keyedEntry][]any, len(ksf)),
		keyedRootScoped:         make(map[keyedEntry][]any, len(kscf)),
		disp:                    NewDisposer(),
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
	for ke := range ksf {
		_ = r.GetKeyedService(ke.t, ke.key)
	}
	return r, nil
}
