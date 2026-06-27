//go:build atdd_redphase

// ATDD red-phase scaffold for Story 1.2 (AC2 / AC3) — the Get[T] / GetRequired[T]
// free-generic convenience layer and its dotnet-aligned unmet-lookup semantics.
//
// Activation (TDD red -> green): delete the //go:build line above and the blank
// line after it once godi.Get / godi.GetRequired exist (Story 1.2 Task 2).
package godi_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
)

// --- minimal local stub Provider --------------------------------------------
//
// fakeProvider is a throwaway, map-backed godi.Provider used only to drive the
// convenience layer. It constructs nothing and manages no lifetimes — it returns
// a registered value or nil, which is all Get / GetRequired need to exercise the
// hit / miss / panic states. The compile-time assertion pins that it satisfies
// the AD-16 face (and so doubles as a guard that the face stays implementable).
type fakeProvider struct{ services map[reflect.Type]any }

var _ godi.Provider = (*fakeProvider)(nil)

func newFake(services map[reflect.Type]any) *fakeProvider {
	return &fakeProvider{services: services}
}

func (f *fakeProvider) GetService(t reflect.Type) any { return f.services[t] }
func (f *fakeProvider) IsService(t reflect.Type) bool { _, ok := f.services[t]; return ok }

func (f *fakeProvider) GetKeyedService(t reflect.Type, key any) any {
	if key == nil {
		return f.GetService(t) // AD-17: nil key routes to the unkeyed service
	}
	return nil
}

func (f *fakeProvider) GetKeyedServices(t reflect.Type, key any) []any { return nil }
func (f *fakeProvider) CreateScope() godi.Scope                        { return f }
func (f *fakeProvider) Close() error                                   { return nil }

// greeter exercises the interface-type resolution path (design decision D2).
type greeter interface{ Greet() string }

type englishGreeter struct{}

func (englishGreeter) Greet() string { return "hello" }

// widget is a distinctive named type so the GetRequired panic-message assertion
// is unambiguous.
type widget struct{ id int }

// TestGet_HitAndMiss (1.2-UNIT-003, AC2/AC3, P0): Get[T] returns (value, true)
// for a registered type and (zero, false) — never a panic — for an unregistered
// one (dotnet TryGet parity).
func TestGet_HitAndMiss(t *testing.T) {
	p := newFake(map[reflect.Type]any{
		reflect.TypeFor[string](): "hello",
		reflect.TypeFor[int]():    42,
	})

	t.Run("registered string -> (v, true)", func(t *testing.T) {
		got, ok := godi.Get[string](p)
		testutil.True(t, ok, "string must resolve")
		testutil.Equal(t, got, "hello")
	})

	t.Run("registered int -> (v, true)", func(t *testing.T) {
		got, ok := godi.Get[int](p)
		testutil.True(t, ok, "int must resolve")
		testutil.Equal(t, got, 42)
	})

	t.Run("unregistered -> (zero, false), no panic", func(t *testing.T) {
		got, ok := godi.Get[int64](p)
		testutil.True(t, !ok, "int64 must not resolve")
		testutil.Equal(t, got, int64(0)) // zero value returned, no panic
	})
}

// TestGet_InterfaceTypeIsSafe (1.2-UNIT-004, AC2/AC3, P0): Get[T] is safe for
// interface T (design decision D2). The stub stores a concrete englishGreeter
// under the greeter key; Get[greeter] resolves it by asserting the STORED value
// (v.(T)), not by dispatching on a nil-interface zero value — the any(zero).
// (type) footgun AD-5 forbids. An empty provider yields (nil, false), not a
// false positive.
func TestGet_InterfaceTypeIsSafe(t *testing.T) {
	t.Run("registered interface type -> (impl, true)", func(t *testing.T) {
		p := newFake(map[reflect.Type]any{
			reflect.TypeFor[greeter](): englishGreeter{},
		})
		got, ok := godi.Get[greeter](p)
		testutil.True(t, ok, "greeter must resolve")
		testutil.NotNil(t, got)
		testutil.Equal(t, got.Greet(), "hello")
	})

	t.Run("unregistered interface type -> (nil, false)", func(t *testing.T) {
		got, ok := godi.Get[greeter](newFake(nil))
		testutil.True(t, !ok, "greeter must not resolve from an empty provider")
		testutil.Nil(t, got)
	})
}

// TestGetRequired_HitReturnsAndMissPanics (1.2-UNIT-005, AC2/AC3, P0):
// GetRequired[T] returns the instance when registered, and panics with a value
// naming the missing type when not (design decision D1 — signature returns only
// T, mirroring dotnet GetRequiredService).
func TestGetRequired_HitReturnsAndMissPanics(t *testing.T) {
	p := newFake(map[reflect.Type]any{
		reflect.TypeFor[string](): "hello",
	})

	t.Run("registered -> instance", func(t *testing.T) {
		testutil.Equal(t, godi.GetRequired[string](p), "hello")
	})

	t.Run("unregistered -> panic naming the type", func(t *testing.T) {
		recovered := testutil.Panics(t, func() { _ = godi.GetRequired[widget](p) })
		msg := panicMessage(recovered)
		want := reflect.TypeFor[widget]().String()
		testutil.True(t, strings.Contains(msg, want),
			"panic must name the missing type "+want+", got: "+msg)
	})
}

// panicMessage renders a recovered panic value to a string whether it was raised
// as an error, a string, or anything else.
func panicMessage(v any) string {
	switch x := v.(type) {
	case error:
		return x.Error()
	case string:
		return x
	default:
		return reflect.TypeOf(v).String() + ": (non-string panic value)"
	}
}
