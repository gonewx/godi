package godi

import (
	"fmt"
	"reflect"
)

// Get resolves the service of type T from p, returning (value, true) on a hit
// and (zero, false) on a miss. A miss never panics — this mirrors dotnet's
// "try" resolution and the AD-16 unmet-lookup contract (an unregistered type
// resolves to nil, not an error).
//
// Get is a free generic FUNCTION, not a Provider method: an interface method may
// not declare type parameters (AD-5), so the typed ergonomics live here over the
// non-generic Provider face.
//
// Design decision D2 — interface-type safety: Get asserts the value actually
// stored by the provider (v.(T)), NOT a zero value. This is why it is correct
// for interface T as well as concrete T. It does NOT use the any(zero).(type)
// zero-value dispatch that AD-5 forbids: for an interface T that zero value is a
// nil interface, which a type switch would mis-match against case nil. That
// footgun belongs to a generated container's type-switch sugar (Story 2.5 / 6.6
// must avoid it); this free function asserting the real stored value is immune
// to it.
func Get[T any](p Provider) (T, bool) {
	v := p.GetService(reflect.TypeFor[T]())
	if v == nil {
		var zero T
		return zero, false
	}
	t, ok := v.(T)
	return t, ok
}

// GetRequired resolves the service of type T from p, returning the instance on a
// hit and PANICKING on a miss. The panic value is an error naming the missing
// type, so a recover site can type-assert and inspect it.
//
// Design decision D1 — panic, not (T, error): the convenience-layer signature is
// pinned to return only T (epics.md Story 1.2), mirroring dotnet's
// GetRequiredService, which raises rather than returning a sentinel. Do not
// widen the signature to (T, error).
func GetRequired[T any](p Provider) T {
	if v, ok := Get[T](p); ok {
		return v
	}
	panic(fmt.Errorf("godi: no service registered for type %s", reflect.TypeFor[T]()))
}

// GetServices resolves every service registered for type T and returns them as a
// typed slice in registration order. An unregistered type returns an empty
// non-nil slice (never nil), mirroring dotnet GetServices<T>(). Underneath it
// calls p.GetKeyedServices(t, nil) — nil key routes to the unkeyed namespace
// (AD-17) — and asserts each element to T.
func GetServices[T any](p Provider) []T {
	all := p.GetKeyedServices(reflect.TypeFor[T](), nil)
	if all == nil {
		return []T{}
	}
	out := make([]T, len(all))
	for i, v := range all {
		out[i] = v.(T)
	}
	return out
}
