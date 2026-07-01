package godi

import (
	"fmt"
	"reflect"
)

type keyedAnyKey struct{}

// AnyKey is the sentinel value used to query all keyed registrations for a given
// service type (AD-17). Story 3-3 implements the full AnyKey semantics
// (GetKeyedService(AnyKey) panics, GetKeyedServices(AnyKey) returns all keyed);
// this story only defines the sentinel.
var AnyKey = keyedAnyKey{}

// AnyKeySingleResolveError builds the error used when GetKeyedService is called
// with the AnyKey sentinel. AnyKey is a query-only sentinel for GetKeyedServices
// and is never valid for single-value resolution (AD-17). Both providers panic
// with this error so runtime and generated code share ONE message source —
// generated code calls godi.AnyKeySingleResolveError instead of building its own
// fmt.Errorf, avoiding a new import in the generated file.
func AnyKeySingleResolveError(t reflect.Type) error {
	return fmt.Errorf("godi: AnyKey is a query-only sentinel for GetKeyedServices and cannot be used with GetKeyedService for type %s (AD-17)", t)
}

// AnyKeyRegistrationError builds the error used when AnyKey is used as a
// registration key. dotnet's AnyKey-catch-all registration (a service resolved by
// ANY concrete key, with that key injected into the constructor) has no Go
// mechanism — a form-B `func(Provider) I` / form-A `New{Impl}` cannot receive a
// resolve-time key (D-3). Both providers REJECT AnyKey as a registration key: the
// runtime panics with this at AddKeyed*, symmetric to the generator's
// generate-time rejection (AC4).
func AnyKeyRegistrationError(t reflect.Type) error {
	return fmt.Errorf("godi: AnyKey is a query-only sentinel and cannot be used as a registration key for type %s (catch-all registration has no Go mechanism, AD-17)", t)
}

// GetRequiredKeyed resolves the keyed service of type T from p, returning the
// instance on a hit and PANICKING on a miss. The panic value is an error naming
// the missing type and key (AC9).
func GetRequiredKeyed[T any](p Provider, key any) T {
	v := p.GetKeyedService(reflect.TypeFor[T](), key)
	if v == nil {
		panic(fmt.Errorf("godi: no keyed service registered for type %s with key %v (%T)", reflect.TypeFor[T](), key, key))
	}
	t, ok := v.(T)
	if !ok {
		panic(fmt.Errorf("godi: keyed service for type %s with key %v is %T, not %s", reflect.TypeFor[T](), key, v, reflect.TypeFor[T]()))
	}
	return t
}

// GetKeyed resolves the keyed service of type T from p, returning (value, true)
// on a hit and (zero, false) on a miss — the non-required keyed counterpart of
// GetRequiredKeyed, symmetric to Get[T] (D-2 naming convention).
func GetKeyed[T any](p Provider, key any) (T, bool) {
	v := p.GetKeyedService(reflect.TypeFor[T](), key)
	if v == nil {
		var zero T
		return zero, false
	}
	t, ok := v.(T)
	return t, ok
}

// GetKeyedServices resolves every keyed service registered for (T, key) and
// returns them as a typed slice in registration order. An unregistered
// (type, key) pair returns an empty non-nil slice (never nil), mirroring dotnet
// GetKeyedServices<T>(key). Underneath it calls p.GetKeyedServices(t, key) and
// asserts each element to T.
func GetKeyedServices[T any](p Provider, key any) []T {
	all := p.GetKeyedServices(reflect.TypeFor[T](), key)
	if all == nil {
		return []T{}
	}
	out := make([]T, len(all))
	for i, v := range all {
		out[i] = v.(T)
	}
	return out
}
