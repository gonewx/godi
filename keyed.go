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
