package runtime

import (
	"reflect"

	"github.com/gonewx/godi"
)

// MakeReadOnly freezes the builder so that any subsequent mutation panics.
// It is idempotent: calling it more than once is safe.
func MakeReadOnly(b *Builder) { b.readOnly = true }

// AddSingletonInstance registers a pre-constructed instance as a singleton.
// The provider returns the same instance on every resolve within the same
// provider (single-provider pointer identity is legal, R4 only prohibits
// cross-provider identity). The instance is tracked for disposal if it
// implements io.Closer or AsyncDisposable.
func AddSingletonInstance[T any](b *Builder, inst T) *Builder {
	b.addDescriptor(serviceDescriptor{
		serviceType: reflect.TypeFor[T](),
		lifetime:    Singleton,
		factory:     func(godi.Provider) any { return inst },
	})
	return b
}

// AddKeyedSingletonInstance registers a pre-constructed instance as a keyed
// singleton. A nil key routes to the unkeyed namespace (AD-17).
func AddKeyedSingletonInstance[T any](b *Builder, key any, inst T) *Builder {
	t := reflect.TypeFor[T]()
	if key == godi.AnyKey {
		panic(godi.AnyKeyRegistrationError(t))
	}
	b.addDescriptor(serviceDescriptor{
		serviceType: t,
		lifetime:    Singleton,
		key:         key,
		factory:     func(godi.Provider) any { return inst },
	})
	return b
}

// TryAddSingleton adds a singleton registration only if no registration for
// the same ServiceType (unkeyed) already exists. If one does, this is a no-op.
func TryAddSingleton[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	return tryAdd(b, reflect.TypeFor[T](), Singleton, nil, func(p godi.Provider) any { return factory(p) })
}

// TryAddScoped adds a scoped registration only if no registration for the same
// ServiceType (unkeyed) already exists.
func TryAddScoped[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	return tryAdd(b, reflect.TypeFor[T](), Scoped, nil, func(p godi.Provider) any { return factory(p) })
}

// TryAddTransient adds a transient registration only if no registration for
// the same ServiceType (unkeyed) already exists.
func TryAddTransient[T any](b *Builder, factory func(godi.Provider) T) *Builder {
	return tryAdd(b, reflect.TypeFor[T](), Transient, nil, func(p godi.Provider) any { return factory(p) })
}

// TryAddKeyedSingleton adds a keyed singleton only if no registration for
// (ServiceType, key) already exists.
func TryAddKeyedSingleton[T any](b *Builder, key any, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	if key == godi.AnyKey {
		panic(godi.AnyKeyRegistrationError(t))
	}
	return tryAdd(b, t, Singleton, key, func(p godi.Provider) any { return factory(p) })
}

// TryAddKeyedScoped adds a keyed scoped only if no registration for
// (ServiceType, key) already exists.
func TryAddKeyedScoped[T any](b *Builder, key any, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	if key == godi.AnyKey {
		panic(godi.AnyKeyRegistrationError(t))
	}
	return tryAdd(b, t, Scoped, key, func(p godi.Provider) any { return factory(p) })
}

// TryAddKeyedTransient adds a keyed transient only if no registration for
// (ServiceType, key) already exists.
func TryAddKeyedTransient[T any](b *Builder, key any, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	if key == godi.AnyKey {
		panic(godi.AnyKeyRegistrationError(t))
	}
	return tryAdd(b, t, Transient, key, func(p godi.Provider) any { return factory(p) })
}

func tryAdd(b *Builder, t reflect.Type, lt Lifetime, key any, factory func(godi.Provider) any) *Builder {
	b.readOnlyGuard()
	for _, d := range b.descriptors {
		if d.serviceType == t && d.key == key {
			return b
		}
	}
	b.descriptors = append(b.descriptors, serviceDescriptor{
		serviceType: t,
		lifetime:    lt,
		key:         key,
		factory:     factory,
	})
	return b
}

// TryAddEnumerable adds a registration only if no registration with the same
// (ServiceType, Key) already exists. In Go form-B the implementation type is
// the factory's return type T which equals the service type, so deduplication
// degenerates to (ServiceType, Key) — the same as TryAdd. This is the Go
// adaptation of dotnet's (ServiceType, ImplementationType, Key) dedup where
// ImplementationType is always known statically (D-7).
func TryAddEnumerable[T any](b *Builder, lt Lifetime, factory func(godi.Provider) T) *Builder {
	return tryAdd(b, reflect.TypeFor[T](), lt, nil, func(p godi.Provider) any { return factory(p) })
}

// Replace removes the first registration matching (ServiceType, Key=nil) and
// appends a new one at the end. If no match exists, the new registration is
// simply appended. Order is significant: {A, B}.Replace(C) → {B, C} (D-3).
func Replace[T any](b *Builder, lt Lifetime, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	b.readOnlyGuard()
	for i, d := range b.descriptors {
		if d.serviceType == t && d.key == nil {
			b.descriptors = append(b.descriptors[:i], b.descriptors[i+1:]...)
			break
		}
	}
	b.descriptors = append(b.descriptors, serviceDescriptor{
		serviceType: t,
		lifetime:    lt,
		factory:     func(p godi.Provider) any { return factory(p) },
	})
	return b
}

// ReplaceKeyed removes the first registration matching (ServiceType, key) and
// appends a new one at the end.
func ReplaceKeyed[T any](b *Builder, key any, lt Lifetime, factory func(godi.Provider) T) *Builder {
	t := reflect.TypeFor[T]()
	if key == godi.AnyKey {
		panic(godi.AnyKeyRegistrationError(t))
	}
	b.readOnlyGuard()
	for i, d := range b.descriptors {
		if d.serviceType == t && d.key == key {
			b.descriptors = append(b.descriptors[:i], b.descriptors[i+1:]...)
			break
		}
	}
	b.descriptors = append(b.descriptors, serviceDescriptor{
		serviceType: t,
		lifetime:    lt,
		key:         key,
		factory:     func(p godi.Provider) any { return factory(p) },
	})
	return b
}

// RemoveAll removes every unkeyed registration for ServiceType T (Key==nil).
// Keyed registrations are NOT affected — use RemoveAllKeyed for those (D-3
// asymmetry).
func RemoveAll[T any](b *Builder) *Builder {
	t := reflect.TypeFor[T]()
	b.readOnlyGuard()
	n := 0
	for _, d := range b.descriptors {
		if d.serviceType == t && d.key == nil {
			continue
		}
		b.descriptors[n] = d
		n++
	}
	b.descriptors = b.descriptors[:n]
	return b
}

// RemoveAllKeyed removes every registration matching (ServiceType, key).
func RemoveAllKeyed[T any](b *Builder, key any) *Builder {
	t := reflect.TypeFor[T]()
	b.readOnlyGuard()
	n := 0
	for _, d := range b.descriptors {
		if d.serviceType == t && d.key == key {
			continue
		}
		b.descriptors[n] = d
		n++
	}
	b.descriptors = b.descriptors[:n]
	return b
}

// Remove is the Go adaptation of dotnet's IServiceCollection.Remove(descriptor),
// which removes ONE registration identified by reference. godi exposes no
// standalone descriptor object to reference, so Remove operates by type
// instead: it removes ALL unkeyed registrations of T (== RemoveAll[T]). This
// diverges from dotnet when multiple registrations of T coexist — dotnet
// removes exactly one, godi removes every unkeyed one (EXTENSIONS_DIVERGENCE.md
// EXT-D10).
func Remove[T any](b *Builder) *Builder {
	return RemoveAll[T](b)
}
