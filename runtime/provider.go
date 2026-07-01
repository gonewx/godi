package runtime

import (
	"context"
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

// trackDisposable records v with disp using the runtime analogue of the codegen
// emit dispatch (D-3/D-4): a v satisfying io.Closer (including a both-disposable)
// is tracked via Track — under CloseAllAsync it is re-asserted to
// AsyncDisposable, so prefer-async still holds; an async-only v (no io.Closer)
// is tracked via TrackAsync. A v that is neither is not tracked.
func trackDisposable(disp *Disposer, v any) {
	if c, ok := v.(io.Closer); ok {
		disp.Track(c)
		return
	}
	if ad, ok := v.(godi.AsyncDisposable); ok {
		disp.TrackAsync(ad)
	}
}

func materializeKeyedInto(cache map[keyedEntry][]any, ke keyedEntry, factories []func(godi.Provider) any, provider godi.Provider, disp *Disposer) []any {
	if instances, ok := cache[ke]; ok {
		return instances
	}
	instances := make([]any, len(factories))
	for i, f := range factories {
		v := f(provider)
		trackDisposable(disp, v)
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
		trackDisposable(disp, v)
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

	disp     *Disposer
	closeMu  sync.Mutex
	closed   bool
	closeErr error
}

var _ godi.Provider = (*root)(nil)

// root and scope implement CloseAsync structurally (godi.AsyncDisposable), NOT
// as part of the pinned Provider/Scope face (AD-16 unchanged, D-2). These
// compile-time checks keep that opt-in surface honest.
var (
	_ godi.AsyncDisposable = (*root)(nil)
	_ godi.AsyncDisposable = (*scope)(nil)
)

func (r *root) GetService(t reflect.Type) any {
	if r.closed {
		panic(godi.ProviderClosedError())
	}
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
		trackDisposable(r.disp, v)
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
	if r.closed {
		panic(godi.ProviderClosedError())
	}
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
		trackDisposable(r.disp, v)
		return v
	}
	return nil
}

func (r *root) GetKeyedServices(t reflect.Type, key any) []any {
	if r.closed {
		panic(godi.ProviderClosedError())
	}
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
			trackDisposable(r.disp, v)
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
			trackDisposable(r.disp, v)
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
			trackDisposable(r.disp, v)
			out = append(out, v)
		}
	}
	return out
}

func (r *root) CreateScope() godi.Scope {
	if r.closed {
		panic(godi.ProviderClosedError())
	}
	return &scope{
		root:        r,
		scoped:      make(map[reflect.Type][]any),
		keyedScoped: make(map[keyedEntry][]any),
		disp:        NewDisposer(),
	}
}

// Close releases this provider and its disposables synchronously in LIFO order.
// It is idempotent and shares ONE closed guard with CloseAsync
// (first-caller-wins, D-6): whichever runs first releases and caches its error;
// a later Close or CloseAsync is a no-op returning the cached error. Marking
// closed BEFORE releasing makes a factory that calls Close mid-Build
// (ctor-dispose-in-flight, SPCT-16) trip the resolve guard on the next
// GetService, and rejects post-Close resolution (AD-16). closeMu replaces the
// prior sync.Once so Close and CloseAsync can share the guard while keeping the
// count-exactly-once semantics (D-6).
func (r *root) Close() error {
	r.closeMu.Lock()
	defer r.closeMu.Unlock()
	if r.closed {
		return r.closeErr
	}
	r.closed = true
	r.closeErr = r.disp.CloseAll()
	return r.closeErr
}

// CloseAsync releases this provider and its disposables in LIFO order,
// preferring each item's CloseAsync over Close (D-4). It is a STRUCTURAL opt-in
// method (godi.AsyncDisposable), NOT part of the pinned Provider face (AD-16
// unchanged, D-2) — a caller reaches it through a type assertion. It shares the
// SAME closed guard as Close (first-caller-wins, D-6 / #38 DisposeAsyncOnce): if
// Close already ran, this returns the cached error without re-releasing.
func (r *root) CloseAsync(ctx context.Context) error {
	r.closeMu.Lock()
	defer r.closeMu.Unlock()
	if r.closed {
		return r.closeErr
	}
	r.closed = true
	r.closeErr = r.disp.CloseAllAsync(ctx)
	return r.closeErr
}

type scope struct {
	root        *root
	scoped      map[reflect.Type][]any
	keyedScoped map[keyedEntry][]any
	disp        *Disposer
	closeMu     sync.Mutex
	closed      bool
	closeErr    error
}

var _ godi.Scope = (*scope)(nil)

// closedGuard reports whether this scope can no longer resolve: either the scope
// itself was closed, or its root was closed (a closed root invalidates every live
// child scope — SPCT-14). A closed scope leaves the root usable (SPCT-15).
func (s *scope) closedGuard() bool { return s.closed || s.root.closed }

func (s *scope) GetService(t reflect.Type) any {
	if s.closedGuard() {
		panic(godi.ProviderClosedError())
	}
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
		trackDisposable(s.disp, v)
		return v
	}
	return nil
}

func (s *scope) IsService(t reflect.Type) bool { return s.root.IsService(t) }

func (s *scope) IsKeyedService(t reflect.Type, key any) bool {
	return s.root.IsKeyedService(t, key)
}

func (s *scope) GetKeyedService(t reflect.Type, key any) any {
	if s.closedGuard() {
		panic(godi.ProviderClosedError())
	}
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
		trackDisposable(s.disp, v)
		return v
	}
	return nil
}

func (s *scope) GetKeyedServices(t reflect.Type, key any) []any {
	if s.closedGuard() {
		panic(godi.ProviderClosedError())
	}
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
			trackDisposable(s.disp, v)
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
			trackDisposable(s.disp, v)
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
			trackDisposable(s.disp, v)
			out = append(out, v)
		}
	}
	return out
}

func (s *scope) CreateScope() godi.Scope {
	if s.closedGuard() {
		panic(godi.ProviderClosedError())
	}
	return &scope{
		root:        s.root,
		scoped:      make(map[reflect.Type][]any),
		keyedScoped: make(map[keyedEntry][]any),
		disp:        NewDisposer(),
	}
}

// Close releases this scope's disposables synchronously in LIFO order. Like
// root.Close it is idempotent and shares ONE closed guard with CloseAsync
// (first-caller-wins, D-6).
func (s *scope) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	s.closeErr = s.disp.CloseAll()
	return s.closeErr
}

// CloseAsync releases this scope's disposables in LIFO order, preferring async
// per item (D-4). Structural opt-in (godi.AsyncDisposable), not part of the
// Scope face (D-2); shares the SAME closed guard as Close (D-6). Singletons are
// tracked by root.disp, so scope.CloseAsync never releases them (AD-12 async
// parity, AC5).
func (s *scope) CloseAsync(ctx context.Context) error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	s.closeErr = s.disp.CloseAllAsync(ctx)
	return s.closeErr
}

// Lifetime identifies the caching strategy for a service registration.
type Lifetime int

const (
	Singleton Lifetime = iota
	Scoped
	Transient
)

// serviceDescriptor records one registration in the Builder's ordered list.
type serviceDescriptor struct {
	serviceType reflect.Type
	lifetime    Lifetime
	key         any // nil for unkeyed
	factory     func(godi.Provider) any
}

// Builder accumulates form-B closure registrations for all three lifetimes
// and materializes them into a runtime provider.
type Builder struct {
	descriptors []serviceDescriptor
	readOnly    bool
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) readOnlyGuard() {
	if b.readOnly {
		panic(fmt.Errorf("godi: service collection is read-only"))
	}
}

// IsReadOnly reports whether the builder has been frozen.
func (b *Builder) IsReadOnly() bool { return b.readOnly }

func (b *Builder) addDescriptor(d serviceDescriptor) {
	b.readOnlyGuard()
	b.descriptors = append(b.descriptors, d)
}

func AddSingleton[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	b.addDescriptor(serviceDescriptor{
		serviceType: reflect.TypeFor[T](),
		lifetime:    Singleton,
		factory:     func(p godi.Provider) any { return factory(p) },
	})
	return b
}

func AddScoped[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	b.addDescriptor(serviceDescriptor{
		serviceType: reflect.TypeFor[T](),
		lifetime:    Scoped,
		factory:     func(p godi.Provider) any { return factory(p) },
	})
	return b
}

func AddTransient[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	b.addDescriptor(serviceDescriptor{
		serviceType: reflect.TypeFor[T](),
		lifetime:    Transient,
		factory:     func(p godi.Provider) any { return factory(p) },
	})
	return b
}

func AddKeyedSingleton[T any](b *Builder, key any, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	if key == godi.AnyKey {
		panic(godi.AnyKeyRegistrationError(t))
	}
	b.addDescriptor(serviceDescriptor{
		serviceType: t,
		lifetime:    Singleton,
		key:         key,
		factory:     func(p godi.Provider) any { return factory(p) },
	})
	return b
}

func AddKeyedScoped[T any](b *Builder, key any, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	if key == godi.AnyKey {
		panic(godi.AnyKeyRegistrationError(t))
	}
	b.addDescriptor(serviceDescriptor{
		serviceType: t,
		lifetime:    Scoped,
		key:         key,
		factory:     func(p godi.Provider) any { return factory(p) },
	})
	return b
}

func AddKeyedTransient[T any](b *Builder, key any, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	if key == godi.AnyKey {
		panic(godi.AnyKeyRegistrationError(t))
	}
	b.addDescriptor(serviceDescriptor{
		serviceType: t,
		lifetime:    Transient,
		key:         key,
		factory:     func(p godi.Provider) any { return factory(p) },
	})
	return b
}

func (b *Builder) Build() (provider godi.Provider, err error) {
	b.readOnly = true

	sf := make(map[reflect.Type][]func(godi.Provider) any)
	scf := make(map[reflect.Type][]func(godi.Provider) any)
	tf := make(map[reflect.Type][]func(godi.Provider) any)
	ksf := make(map[keyedEntry][]func(godi.Provider) any)
	kscf := make(map[keyedEntry][]func(godi.Provider) any)
	ktf := make(map[keyedEntry][]func(godi.Provider) any)
	var ko []keyedReg

	for _, d := range b.descriptors {
		if d.key == nil {
			switch d.lifetime {
			case Singleton:
				sf[d.serviceType] = append(sf[d.serviceType], d.factory)
			case Scoped:
				scf[d.serviceType] = append(scf[d.serviceType], d.factory)
			case Transient:
				tf[d.serviceType] = append(tf[d.serviceType], d.factory)
			default:
				return nil, fmt.Errorf("godi: invalid lifetime %d for type %s", d.lifetime, d.serviceType)
			}
		} else {
			ke := keyedEntry{t: d.serviceType, key: d.key}
			switch d.lifetime {
			case Singleton:
				slot := len(ksf[ke])
				ksf[ke] = append(ksf[ke], d.factory)
				ko = append(ko, keyedReg{ke: ke, lifetime: keyedSingleton, slot: slot})
			case Scoped:
				slot := len(kscf[ke])
				kscf[ke] = append(kscf[ke], d.factory)
				ko = append(ko, keyedReg{ke: ke, lifetime: keyedScoped, slot: slot})
			case Transient:
				slot := len(ktf[ke])
				ktf[ke] = append(ktf[ke], d.factory)
				ko = append(ko, keyedReg{ke: ke, lifetime: keyedTransient, slot: slot})
			default:
				return nil, fmt.Errorf("godi: invalid lifetime %d for type %s key %v", d.lifetime, d.serviceType, d.key)
			}
		}
	}

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
		materializing:           make(map[reflect.Type]bool, len(sf)+len(scf)),
		materializingKeyed:      make(map[keyedEntry]bool, len(ksf)+len(kscf)),
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
