package runtime

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gonewx/godi"
)

type keyedEntry struct {
	t   reflect.Type
	key any
}

// regRef locates one registration: its lifetime, and its index (slot) inside
// that lifetime's factory slice for its type or (type, key). Each type and each
// (type, key) keeps its registrations as a regRef list in registration order,
// so a single resolve takes the LAST registration whatever its lifetime and a
// collection resolve walks them in order, each element by its own lifetime —
// dotnet's semantics for mixed-lifetime registrations (eval fix A4). The
// per-lifetime factory slices stay grouped, so a singleton or scoped group is
// still materialized as a whole on first resolution.
type regRef struct {
	lifetime Lifetime
	slot     int
}

// keyedReg records one concrete-keyed registration in registration order. Go maps
// are unordered, so the keyed factory maps alone cannot answer AnyKey-query order;
// keyedReg gives the explicit cross-key registration sequence AnyKey-query needs
// (D-4).
type keyedReg struct {
	ke keyedEntry
	regRef
}

// lifetimeGroup names the singleton or scoped factory group of a type, or of a
// (type, key) when keyed is set, for the materializing (cycle) guard. The
// lifetime is part of the key: with mixed lifetimes, building one group may
// legitimately resolve the type's other group.
type lifetimeGroup struct {
	ke       keyedEntry
	keyed    bool
	lifetime Lifetime
}

func materializeKeyedInto(cache map[keyedEntry][]any, ke keyedEntry, factories []func(godi.Provider) any, provider godi.Provider, disp *Disposer) []any {
	if instances, ok := cache[ke]; ok {
		return instances
	}
	instances := make([]any, len(factories))
	for i, f := range factories {
		v := f(provider)
		disp.TrackAny(v)
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
		disp.TrackAny(v)
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

	// unkeyedRegs / keyedRegs list each type's (and each (type, key)'s)
	// registrations in registration order (see regRef).
	unkeyedRegs map[reflect.Type][]regRef
	keyedRegs   map[keyedEntry][]regRef

	singletons      map[reflect.Type][]any
	rootScoped      map[reflect.Type][]any
	keyedSingletons map[keyedEntry][]any
	keyedRootScoped map[keyedEntry][]any

	// keyedOrder is the cross-key registration sequence of all concrete-keyed
	// services (nil/AnyKey excluded at registration), used by AnyKey-query (D-4).
	keyedOrder []keyedReg

	materializing map[lifetimeGroup]bool

	disp  *Disposer
	guard godi.CloseGuard
}

var _ godi.Provider = (*root)(nil)

// root and scope implement CloseAsync structurally (godi.AsyncDisposable), NOT
// as part of the pinned Provider/Scope face (AD-16 unchanged, D-2). These
// compile-time checks keep that opt-in surface honest.
var (
	_ godi.AsyncDisposable = (*root)(nil)
	_ godi.AsyncDisposable = (*scope)(nil)
)

// resolver resolves single registrations against one resolving provider: the
// root itself, or a scope. Singletons always come from the root's cache and
// disposer; scoped instances from the resolver's own cache and disposer;
// transients are built anew with the resolving provider and tracked by its
// disposer. guard enables the root's materializing (cycle) guard; it is on for
// the root's single resolves and Build's eager pass, the paths that had it.
type resolver struct {
	root        *root
	self        godi.Provider
	scoped      map[reflect.Type][]any
	keyedScoped map[keyedEntry][]any
	disp        *Disposer
	guard       bool
}

func (r *root) resolver(guard bool) resolver {
	return resolver{root: r, self: r, scoped: r.rootScoped, keyedScoped: r.keyedRootScoped, disp: r.disp, guard: guard}
}

func (s *scope) resolver() resolver {
	return resolver{root: s.root, self: s, scoped: s.scoped, keyedScoped: s.keyedScoped, disp: s.disp}
}

// enter marks group g as being materialized, panicking if it already is (a
// cycle). The returned func clears the mark; it is a no-op without the guard
// or when the group is already cached.
func (v resolver) enter(g lifetimeGroup, cached bool) func() {
	if !v.guard || cached {
		return func() {}
	}
	if v.root.materializing[g] {
		if g.keyed {
			panic(fmt.Sprintf("circular dependency detected for keyed type %v key %v", g.ke.t, g.ke.key))
		}
		panic(fmt.Sprintf("circular dependency detected for type %v", g.ke.t))
	}
	v.root.materializing[g] = true
	return func() { delete(v.root.materializing, g) }
}

// group materializes (once) and returns the whole singleton or scoped group of
// type t.
func (v resolver) group(lt Lifetime, t reflect.Type) []any {
	r := v.root
	cache, factories, provider, disp := r.singletons, r.singletonFactories[t], godi.Provider(r), r.disp
	if lt == Scoped {
		cache, factories, provider, disp = v.scoped, r.scopedFactories[t], v.self, v.disp
	}
	_, cached := cache[t]
	defer v.enter(lifetimeGroup{ke: keyedEntry{t: t}, lifetime: lt}, cached)()
	return materializeInto(cache, t, factories, provider, disp)
}

// keyedGroup is group for the (type, key) registrations of ke.
func (v resolver) keyedGroup(lt Lifetime, ke keyedEntry) []any {
	r := v.root
	cache, factories, provider, disp := r.keyedSingletons, r.keyedSingletonFactories[ke], godi.Provider(r), r.disp
	if lt == Scoped {
		cache, factories, provider, disp = v.keyedScoped, r.keyedScopedFactories[ke], v.self, v.disp
	}
	_, cached := cache[ke]
	defer v.enter(lifetimeGroup{ke: ke, keyed: true, lifetime: lt}, cached)()
	return materializeKeyedInto(cache, ke, factories, provider, disp)
}

// unkeyed returns the instance of the unkeyed registration ref of type t.
func (v resolver) unkeyed(t reflect.Type, ref regRef) any {
	if ref.lifetime == Transient {
		x := v.root.transientFactories[t][ref.slot](v.self)
		v.disp.TrackAny(x)
		return x
	}
	return v.group(ref.lifetime, t)[ref.slot]
}

// keyed returns the instance of the keyed registration ref of ke.
func (v resolver) keyed(ke keyedEntry, ref regRef) any {
	if ref.lifetime == Transient {
		x := v.root.keyedTransientFactories[ke][ref.slot](v.self)
		v.disp.TrackAny(x)
		return x
	}
	return v.keyedGroup(ref.lifetime, ke)[ref.slot]
}

// last resolves the last registration of t (nil when t is unregistered).
func (v resolver) last(t reflect.Type) any {
	refs := v.root.unkeyedRegs[t]
	if len(refs) == 0 {
		return nil
	}
	return v.unkeyed(t, refs[len(refs)-1])
}

// all resolves every registration of t in registration order (nil when t is
// unregistered).
func (v resolver) all(t reflect.Type) []any {
	refs := v.root.unkeyedRegs[t]
	if len(refs) == 0 {
		return nil
	}
	out := make([]any, len(refs))
	for i, ref := range refs {
		out[i] = v.unkeyed(t, ref)
	}
	return out
}

// lastKeyed resolves the last registration of ke (nil when ke is unregistered).
func (v resolver) lastKeyed(ke keyedEntry) any {
	refs := v.root.keyedRegs[ke]
	if len(refs) == 0 {
		return nil
	}
	return v.keyed(ke, refs[len(refs)-1])
}

// allKeyed resolves every registration of ke in registration order (nil when ke
// is unregistered).
func (v resolver) allKeyed(ke keyedEntry) []any {
	refs := v.root.keyedRegs[ke]
	if len(refs) == 0 {
		return nil
	}
	out := make([]any, len(refs))
	for i, ref := range refs {
		out[i] = v.keyed(ke, ref)
	}
	return out
}

// anyKeyed returns every concrete-keyed registration of type t in registration
// order — the GetKeyedServices(AnyKey) result (AC2). It walks keyedOrder (NOT the
// unordered factory maps, D-4), and resolves each entry against the SAME cache
// the concrete-key path uses so singletons/scoped stay identical across
// AnyKey-query and concrete-key resolution (cache consistency, R1). From a scope,
// scoped entries materialize into that scope's own cache (R2). Unkeyed/null
// services never enter keyedOrder, so they are excluded for free. No match → nil.
func (v resolver) anyKeyed(t reflect.Type) []any {
	var out []any
	for _, kr := range v.root.keyedOrder {
		if kr.ke.t != t {
			continue
		}
		out = append(out, v.keyed(kr.ke, kr.regRef))
	}
	return out
}

func (r *root) GetService(t reflect.Type) any {
	if r.guard.Closed() {
		panic(godi.ProviderClosedError())
	}
	return r.resolver(true).last(t)
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
	if r.guard.Closed() {
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
	return r.resolver(true).lastKeyed(keyedEntry{t: t, key: key})
}

func (r *root) GetKeyedServices(t reflect.Type, key any) []any {
	if r.guard.Closed() {
		panic(godi.ProviderClosedError())
	}
	if key == nil {
		return r.resolver(false).all(t)
	}
	if key == godi.AnyKey {
		return r.resolver(false).anyKeyed(t)
	}
	return r.resolver(false).allKeyed(keyedEntry{t: t, key: key})
}

func (r *root) CreateScope() godi.Scope {
	if r.guard.Closed() {
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
// GetService, and rejects post-Close resolution (AD-16). The guard is a
// godi.CloseGuard shared with CloseAsync: concurrent Close calls release once,
// and a disposable whose Close calls back into this Close gets nil at once
// instead of deadlocking (D-6 count-exactly-once).
func (r *root) Close() error {
	return r.guard.Close(func() error { return r.disp.CloseAll() })
}

// CloseAsync releases this provider and its disposables in LIFO order,
// preferring each item's CloseAsync over Close (D-4). It is a STRUCTURAL opt-in
// method (godi.AsyncDisposable), NOT part of the pinned Provider face (AD-16
// unchanged, D-2) — a caller reaches it through a type assertion. It shares the
// SAME closed guard as Close (first-caller-wins, D-6 / #38 DisposeAsyncOnce): if
// Close already ran, this returns the cached error without re-releasing.
func (r *root) CloseAsync(ctx context.Context) error {
	return r.guard.Close(func() error { return r.disp.CloseAllAsync(ctx) })
}

type scope struct {
	root        *root
	scoped      map[reflect.Type][]any
	keyedScoped map[keyedEntry][]any
	disp        *Disposer
	guard       godi.CloseGuard
}

var _ godi.Scope = (*scope)(nil)

// closedGuard reports whether this scope can no longer resolve: either the scope
// itself was closed, or its root was closed (a closed root invalidates every live
// child scope — SPCT-14). A closed scope leaves the root usable (SPCT-15).
func (s *scope) closedGuard() bool { return s.guard.Closed() || s.root.guard.Closed() }

func (s *scope) GetService(t reflect.Type) any {
	if s.closedGuard() {
		panic(godi.ProviderClosedError())
	}
	return s.resolver().last(t)
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
	return s.resolver().lastKeyed(keyedEntry{t: t, key: key})
}

func (s *scope) GetKeyedServices(t reflect.Type, key any) []any {
	if s.closedGuard() {
		panic(godi.ProviderClosedError())
	}
	if key == nil {
		return s.resolver().all(t)
	}
	if key == godi.AnyKey {
		return s.resolver().anyKeyed(t)
	}
	return s.resolver().allKeyed(keyedEntry{t: t, key: key})
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
	return s.guard.Close(func() error { return s.disp.CloseAll() })
}

// CloseAsync releases this scope's disposables in LIFO order, preferring async
// per item (D-4). Structural opt-in (godi.AsyncDisposable), not part of the
// Scope face (D-2); shares the SAME closed guard as Close (D-6). Singletons are
// tracked by root.disp, so scope.CloseAsync never releases them (AD-12 async
// parity, AC5).
func (s *scope) CloseAsync(ctx context.Context) error {
	return s.guard.Close(func() error { return s.disp.CloseAllAsync(ctx) })
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
	ur := make(map[reflect.Type][]regRef)
	kr := make(map[keyedEntry][]regRef)
	var ko []keyedReg

	for _, d := range b.descriptors {
		if d.key == nil {
			var tables map[reflect.Type][]func(godi.Provider) any
			switch d.lifetime {
			case Singleton:
				tables = sf
			case Scoped:
				tables = scf
			case Transient:
				tables = tf
			default:
				return nil, fmt.Errorf("godi: invalid lifetime %d for type %s", d.lifetime, d.serviceType)
			}
			ref := regRef{lifetime: d.lifetime, slot: len(tables[d.serviceType])}
			tables[d.serviceType] = append(tables[d.serviceType], d.factory)
			ur[d.serviceType] = append(ur[d.serviceType], ref)
		} else {
			ke := keyedEntry{t: d.serviceType, key: d.key}
			var tables map[keyedEntry][]func(godi.Provider) any
			switch d.lifetime {
			case Singleton:
				tables = ksf
			case Scoped:
				tables = kscf
			case Transient:
				tables = ktf
			default:
				return nil, fmt.Errorf("godi: invalid lifetime %d for type %s key %v", d.lifetime, d.serviceType, d.key)
			}
			ref := regRef{lifetime: d.lifetime, slot: len(tables[ke])}
			tables[ke] = append(tables[ke], d.factory)
			kr[ke] = append(kr[ke], ref)
			ko = append(ko, keyedReg{ke: ke, regRef: ref})
		}
	}

	r := &root{
		singletonFactories:      sf,
		scopedFactories:         scf,
		transientFactories:      tf,
		keyedSingletonFactories: ksf,
		keyedScopedFactories:    kscf,
		keyedTransientFactories: ktf,
		unkeyedRegs:             ur,
		keyedRegs:               kr,
		singletons:              make(map[reflect.Type][]any, len(sf)),
		rootScoped:              make(map[reflect.Type][]any, len(scf)),
		keyedSingletons:         make(map[keyedEntry][]any, len(ksf)),
		keyedRootScoped:         make(map[keyedEntry][]any, len(kscf)),
		keyedOrder:              ko,
		materializing:           make(map[lifetimeGroup]bool, len(sf)+len(scf)+len(ksf)+len(kscf)),
		disp:                    NewDisposer(),
	}
	defer func() {
		if rec := recover(); rec != nil {
			_ = r.disp.CloseAll()
			provider, err = nil, fmt.Errorf("runtime build failed materializing singletons: %v", rec)
		}
	}()
	// Eagerly build every singleton group. This materializes the groups
	// directly rather than through GetService, whose result is a type's LAST
	// registration and may not be a singleton. A factory that closes the
	// provider mid-Build still trips the closed guard before the next group.
	for t := range sf {
		if r.guard.Closed() {
			panic(godi.ProviderClosedError())
		}
		_ = r.resolver(true).group(Singleton, t)
	}
	for ke := range ksf {
		if r.guard.Closed() {
			panic(godi.ProviderClosedError())
		}
		_ = r.resolver(true).keyedGroup(Singleton, ke)
	}
	return r, nil
}
