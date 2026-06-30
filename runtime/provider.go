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

// keyedLifetime tags which lifetime map a keyed registration lives in, so an
// AnyKey-query (GetKeyedServices(AnyKey)) can resolve each entry against the
// SAME cache the concrete-key path uses (cache consistency, AC2/D-4).
type keyedLifetime int

const (
	keyedSingleton keyedLifetime = iota
	keyedScoped
	keyedTransient
)

// keyedReg records one concrete-keyed registration in registration order. Go maps
// are unordered, so the keyed factory maps alone cannot answer AnyKey-query order;
// keyedReg gives the explicit cross-key registration sequence AnyKey-query needs
// (D-4). slot is this registration's index inside its keyedEntry factory slice, so
// the resolved instance lines up with the right factory under duplicate keys.
type keyedReg struct {
	ke       keyedEntry
	lifetime keyedLifetime
	slot     int
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

	// keyedOrder is the cross-key registration sequence of all concrete-keyed
	// services (nil/AnyKey excluded at registration), used by AnyKey-query (D-4).
	keyedOrder []keyedReg

	materializing      map[reflect.Type]bool
	materializingKeyed map[keyedEntry]bool

	disp      *Disposer
	closeOnce sync.Once
	closeErr  error
}

var _ godi.Provider = (*root)(nil)

func (r *root) GetService(t reflect.Type) any {
	if factories, ok := r.singletonFactories[t]; ok {
		if _, cached := r.singletons[t]; !cached {
			if r.materializing[t] {
				panic(fmt.Sprintf("circular dependency detected for type %v", t))
			}
			r.materializing[t] = true
			defer delete(r.materializing, t)
		}
		instances := materializeInto(r.singletons, t, factories, r, r.disp)
		return instances[len(instances)-1]
	}
	if factories, ok := r.scopedFactories[t]; ok {
		if _, cached := r.rootScoped[t]; !cached {
			if r.materializing[t] {
				panic(fmt.Sprintf("circular dependency detected for type %v", t))
			}
			r.materializing[t] = true
			defer delete(r.materializing, t)
		}
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

// IsKeyedService reports whether (t, key) can be resolved (CAP-5/6, AD-16 face
// revised 6→7 in Story 3.4). It is a PURE QUERY — never panics — and routes keys
// like GetKeyedService EXCEPT for AnyKey: a nil key routes to the unkeyed
// IsService, AnyKey returns false (NOT a panic, the deliberate contrast with
// GetKeyedService(AnyKey)), and a concrete key hits when any of the three keyed
// factory maps holds (t, key). AnyKey is never a registration key (both providers
// reject it), so an AnyKey query can never hit — false (Story 3.4 D-3).
func (r *root) IsKeyedService(t reflect.Type, key any) bool {
	if key == nil {
		return r.IsService(t)
	}
	if key == godi.AnyKey {
		return false
	}
	ke := keyedEntry{t: t, key: key}
	if _, ok := r.keyedSingletonFactories[ke]; ok {
		return true
	}
	if _, ok := r.keyedScopedFactories[ke]; ok {
		return true
	}
	if _, ok := r.keyedTransientFactories[ke]; ok {
		return true
	}
	return false
}

func (r *root) GetKeyedService(t reflect.Type, key any) any {
	if key == nil {
		return r.GetService(t)
	}
	// AnyKey is a query-only sentinel: intercept BEFORE the keyed-map lookup so a
	// single-value resolve always panics, even if a concrete key exists (AD-17).
	if key == godi.AnyKey {
		panic(godi.AnyKeySingleResolveError(t))
	}
	ke := keyedEntry{t: t, key: key}
	if factories, ok := r.keyedSingletonFactories[ke]; ok {
		if _, cached := r.keyedSingletons[ke]; !cached {
			if r.materializingKeyed[ke] {
				panic(fmt.Sprintf("circular dependency detected for keyed type %v key %v", t, key))
			}
			r.materializingKeyed[ke] = true
			defer delete(r.materializingKeyed, ke)
		}
		instances := materializeKeyedInto(r.keyedSingletons, ke, factories, r, r.disp)
		return instances[len(instances)-1]
	}
	if factories, ok := r.keyedScopedFactories[ke]; ok {
		if _, cached := r.keyedRootScoped[ke]; !cached {
			if r.materializingKeyed[ke] {
				panic(fmt.Sprintf("circular dependency detected for keyed type %v key %v", t, key))
			}
			r.materializingKeyed[ke] = true
			defer delete(r.materializingKeyed, ke)
		}
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
	if key == godi.AnyKey {
		return r.getAnyKeyedServices(t)
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

// getAnyKeyedServices returns every concrete-keyed registration of type t in
// registration order — the GetKeyedServices(AnyKey) result (AC2). It walks
// keyedOrder (NOT the unordered factory maps, D-4), and resolves each entry
// against the SAME cache the concrete-key path uses so singletons/scoped stay
// identical across AnyKey-query and concrete-key resolution (cache consistency,
// R1). Unkeyed/null services never enter keyedOrder, so they are excluded for
// free. No match → nil.
func (r *root) getAnyKeyedServices(t reflect.Type) []any {
	var out []any
	for _, kr := range r.keyedOrder {
		if kr.ke.t != t {
			continue
		}
		switch kr.lifetime {
		case keyedSingleton:
			instances := materializeKeyedInto(r.keyedSingletons, kr.ke, r.keyedSingletonFactories[kr.ke], r, r.disp)
			out = append(out, instances[kr.slot])
		case keyedScoped:
			instances := materializeKeyedInto(r.keyedRootScoped, kr.ke, r.keyedScopedFactories[kr.ke], r, r.disp)
			out = append(out, instances[kr.slot])
		case keyedTransient:
			v := r.keyedTransientFactories[kr.ke][kr.slot](r)
			if c, ok := v.(io.Closer); ok {
				r.disp.Track(c)
			}
			out = append(out, v)
		}
	}
	return out
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

func (s *scope) IsKeyedService(t reflect.Type, key any) bool {
	return s.root.IsKeyedService(t, key)
}

func (s *scope) GetKeyedService(t reflect.Type, key any) any {
	if key == nil {
		return s.GetService(t)
	}
	// AnyKey single-value resolve panics from a scope too (root+scope parity, AD-17).
	if key == godi.AnyKey {
		panic(godi.AnyKeySingleResolveError(t))
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
	if key == godi.AnyKey {
		return s.getAnyKeyedServices(t)
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

// getAnyKeyedServices is the scope-level GetKeyedServices(AnyKey) (AC2). It walks
// root.keyedOrder for the cross-key registration order, but materializes scoped
// entries into THIS scope's own keyedScoped cache (singletons stay on root,
// transients build anew) — the scoped-from-root materialization pattern from
// Story 3-2 review P0 (R2). No match → nil.
func (s *scope) getAnyKeyedServices(t reflect.Type) []any {
	var out []any
	for _, kr := range s.root.keyedOrder {
		if kr.ke.t != t {
			continue
		}
		switch kr.lifetime {
		case keyedSingleton:
			instances := materializeKeyedInto(s.root.keyedSingletons, kr.ke, s.root.keyedSingletonFactories[kr.ke], s.root, s.root.disp)
			out = append(out, instances[kr.slot])
		case keyedScoped:
			instances := materializeKeyedInto(s.keyedScoped, kr.ke, s.root.keyedScopedFactories[kr.ke], s, s.disp)
			out = append(out, instances[kr.slot])
		case keyedTransient:
			v := s.root.keyedTransientFactories[kr.ke][kr.slot](s)
			if c, ok := v.(io.Closer); ok {
				s.disp.Track(c)
			}
			out = append(out, v)
		}
	}
	return out
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

	// keyedOrder records concrete-keyed registrations in call order (D-4).
	keyedOrder []keyedReg
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
	t := reflect.TypeFor[T]()
	wrapped := func(p godi.Provider) any { return factory(p) }
	// Check order: AnyKey (reject) → nil (route to unkeyed) → concrete key.
	if key == godi.AnyKey {
		panic(godi.AnyKeyRegistrationError(t))
	}
	if key == nil {
		// null key == unkeyed (AD-17): route to the unkeyed namespace, NOT the
		// keyed index, symmetric to nil resolution (AC1 registration side).
		b.singletonFactories[t] = append(b.singletonFactories[t], wrapped)
		return b
	}
	ke := keyedEntry{t: t, key: key}
	slot := len(b.keyedSingletonFactories[ke])
	b.keyedSingletonFactories[ke] = append(b.keyedSingletonFactories[ke], wrapped)
	b.keyedOrder = append(b.keyedOrder, keyedReg{ke: ke, lifetime: keyedSingleton, slot: slot})
	return b
}

func AddKeyedScoped[T any](b *Builder, key any, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	wrapped := func(p godi.Provider) any { return factory(p) }
	if key == godi.AnyKey {
		panic(godi.AnyKeyRegistrationError(t))
	}
	if key == nil {
		b.scopedFactories[t] = append(b.scopedFactories[t], wrapped)
		return b
	}
	ke := keyedEntry{t: t, key: key}
	slot := len(b.keyedScopedFactories[ke])
	b.keyedScopedFactories[ke] = append(b.keyedScopedFactories[ke], wrapped)
	b.keyedOrder = append(b.keyedOrder, keyedReg{ke: ke, lifetime: keyedScoped, slot: slot})
	return b
}

func AddKeyedTransient[T any](b *Builder, key any, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	wrapped := func(p godi.Provider) any { return factory(p) }
	if key == godi.AnyKey {
		panic(godi.AnyKeyRegistrationError(t))
	}
	if key == nil {
		b.transientFactories[t] = append(b.transientFactories[t], wrapped)
		return b
	}
	ke := keyedEntry{t: t, key: key}
	slot := len(b.keyedTransientFactories[ke])
	b.keyedTransientFactories[ke] = append(b.keyedTransientFactories[ke], wrapped)
	b.keyedOrder = append(b.keyedOrder, keyedReg{ke: ke, lifetime: keyedTransient, slot: slot})
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
	ko := make([]keyedReg, len(b.keyedOrder))
	copy(ko, b.keyedOrder)

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
		keyedOrder:              ko,
		materializing:           make(map[reflect.Type]bool),
		materializingKeyed:      make(map[keyedEntry]bool),
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
