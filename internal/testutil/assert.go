// Package testutil provides standard-library-only test helpers shared across
// the godi module: ergonomic assertions, a data-factory builder, and a golden
// file comparator.
//
// No third-party dependency (no testify): assertions are thin wrappers over the
// stdlib testing package, keeping the library's dependency surface minimal per
// the ARCHITECTURE-SPINE Stack table and the PoC's stdlib-only convention.
package testutil

import "reflect"

// TB is the subset of testing.TB the assertion helpers rely on. Both *testing.T
// and *testing.B satisfy it; tests can pass a fake implementation to verify that
// a helper reports (or does not report) a failure on given input.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// Equal fails (non-fatally) when got != want, comparing with == over comparable
// types.
func Equal[T comparable](t TB, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("not equal:\n got: %v\nwant: %v", got, want)
	}
}

// NotEqual fails when got == dontWant.
func NotEqual[T comparable](t TB, got, dontWant T) {
	t.Helper()
	if got == dontWant {
		t.Errorf("expected value to differ from %v, but it did not", dontWant)
	}
}

// DeepEqual fails when got and want differ under reflect.DeepEqual. Use for
// slices, maps, and structs not comparable with ==.
func DeepEqual(t TB, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("not deep-equal:\n got: %#v\nwant: %#v", got, want)
	}
}

// NoError fails fatally when err is non-nil.
func NoError(t TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Error fails when err is nil.
func Error(t TB, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("expected an error, got nil")
	}
}

// True fails when cond is false; msg describes the expectation.
func True(t TB, cond bool, msg string) {
	t.Helper()
	if !cond {
		t.Errorf("expected condition to be true: %s", msg)
	}
}

// Nil fails when v is not nil. It treats a typed-nil interface as nil — the
// classic DI footgun the architecture calls out (AD-5: a nil interface value
// must not be mistaken for a present service).
func Nil(t TB, v any) {
	t.Helper()
	if !isNil(v) {
		t.Errorf("expected nil, got %#v", v)
	}
}

// NotNil fails when v is nil (including typed-nil).
func NotNil(t TB, v any) {
	t.Helper()
	if isNil(v) {
		t.Errorf("expected non-nil value, got nil")
	}
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}

// Panics fails when fn does not panic. It returns the recovered value so callers
// can assert on it — e.g. AD-17 requires GetKeyedService(AnyKey) to panic.
func Panics(t TB, fn func()) (recovered any) {
	t.Helper()
	defer func() {
		recovered = recover()
		if recovered == nil {
			t.Errorf("expected fn to panic, but it returned normally")
		}
	}()
	fn()
	return // reached only when fn did not panic; deferred sets recovered=nil + fails
}
