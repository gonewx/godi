// Runtime provider: several registrations of one service type with different
// lifetimes resolve in registration order, as in dotnet (eval fix A4). A
// single resolve returns the last registration whatever its lifetime; a
// collection resolve returns every registration in registration order, each
// element taking its instance by its own lifetime.
package runtime_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

type iMixed interface{ Tag() string }

// mixedSvc carries the tag of the registration that built it; the closed flag
// records disposal.
type mixedSvc struct {
	tag    string
	closed bool
}

func (m *mixedSvc) Tag() string  { return m.tag }
func (m *mixedSvc) Close() error { m.closed = true; return nil }

func tags(vs []any) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.(iMixed).Tag()
	}
	return out
}

func TestRuntime_MixedLifetime_KeyedFollowsSameRules(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddKeyedScoped[iMixed](b, "k", func(godi.Provider) iMixed { return &mixedSvc{tag: "scoped"} })
	runtime.AddKeyedTransient[iMixed](b, "k", func(godi.Provider) iMixed { return &mixedSvc{tag: "transient"} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })
	s := p.CreateScope()
	t.Cleanup(func() { _ = s.Close() })

	typ := reflect.TypeFor[iMixed]()
	for _, r := range []struct {
		name string
		p    godi.Provider
	}{{"root", p}, {"scope", s}} {
		t.Run(r.name, func(t *testing.T) {
			single := r.p.GetKeyedService(typ, "k")
			testutil.Equal(t, single.(iMixed).Tag(), "transient")
			testutil.True(t, single != r.p.GetKeyedService(typ, "k"), "the transient single resolve must be new each time")

			first := r.p.GetKeyedServices(typ, "k")
			second := r.p.GetKeyedServices(typ, "k")
			testutil.DeepEqual(t, tags(first), []string{"scoped", "transient"})
			if len(first) != 2 || len(second) != 2 {
				t.Fatalf("got %d and %d elements, want 2", len(first), len(second))
			}
			testutil.True(t, first[0] == second[0], "the scoped element must be the same instance within one scope")
			testutil.True(t, first[1] != second[1], "the transient element must be new on every resolve")

			// AnyKey enumerates the same registrations in the same order.
			testutil.DeepEqual(t, tags(r.p.GetKeyedServices(typ, godi.AnyKey)), []string{"scoped", "transient"})
		})
	}
	testutil.True(t, p.GetKeyedService(typ, "k") != nil && p.GetKeyedServices(typ, "k")[0] != s.GetKeyedServices(typ, "k")[0],
		"root and scope must hold their own scoped instance")
}

// TestRuntime_MixedLifetime_ScopeResolvesEachElementByItsLifetime: resolved
// from a scope, the singleton element is the root's instance, the scoped
// element is the scope's own (same twice in one scope, different across
// scopes), and the transient element is new each time. Singletons are tracked
// by the root's disposer and scoped/transient elements by the resolving scope.
func TestRuntime_MixedLifetime_ScopeResolvesEachElementByItsLifetime(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iMixed](b, func(godi.Provider) iMixed { return &mixedSvc{tag: "singleton"} })
	runtime.AddScoped[iMixed](b, func(godi.Provider) iMixed { return &mixedSvc{tag: "scoped"} })
	runtime.AddTransient[iMixed](b, func(godi.Provider) iMixed { return &mixedSvc{tag: "transient"} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	typ := reflect.TypeFor[iMixed]()
	fromRoot := p.GetKeyedServices(typ, nil)
	testutil.DeepEqual(t, tags(fromRoot), []string{"singleton", "scoped", "transient"})
	if len(fromRoot) != 3 {
		t.Fatalf("got %d elements, want 3", len(fromRoot))
	}

	s1 := p.CreateScope()
	s2 := p.CreateScope()
	a := s1.GetKeyedServices(typ, nil)
	again := s1.GetKeyedServices(typ, nil)
	other := s2.GetKeyedServices(typ, nil)
	testutil.DeepEqual(t, tags(a), []string{"singleton", "scoped", "transient"})

	testutil.True(t, a[0] == fromRoot[0] && other[0] == fromRoot[0], "the singleton element must be the root's instance")
	testutil.True(t, a[1] == again[1], "the scoped element must be the same instance within one scope")
	testutil.True(t, a[1] != other[1], "the scoped element must differ across scopes")
	testutil.True(t, a[1] != fromRoot[1], "a scope's scoped element must not be the root's")
	testutil.True(t, a[2] != again[2], "the transient element must be new on every resolve")

	single := s1.GetService(typ)
	testutil.Equal(t, single.(iMixed).Tag(), "transient")

	testutil.NoError(t, s1.Close())
	testutil.True(t, a[1].(*mixedSvc).closed, "closing the scope must release its scoped element")
	testutil.True(t, a[2].(*mixedSvc).closed && single.(*mixedSvc).closed, "closing the scope must release its transients")
	testutil.True(t, !a[0].(*mixedSvc).closed, "closing a scope must not release the singleton")
	testutil.True(t, !other[1].(*mixedSvc).closed, "closing one scope must not release another scope's instance")

	testutil.NoError(t, s2.Close())
	testutil.NoError(t, p.Close())
	testutil.True(t, fromRoot[0].(*mixedSvc).closed, "closing the root must release the singleton")
	testutil.True(t, fromRoot[1].(*mixedSvc).closed, "closing the root must release its root-level scoped element")
}

// TestRuntime_MixedLifetime_SingletonBuiltAtBuildWhenNotLast: Build eagerly
// constructs every singleton registration even when the type's last
// registration is not a singleton.
func TestRuntime_MixedLifetime_SingletonBuiltAtBuildWhenNotLast(t *testing.T) {
	calls := 0
	b := runtime.NewBuilder()
	runtime.AddSingleton[iMixed](b, func(godi.Provider) iMixed { calls++; return &mixedSvc{tag: "singleton"} })
	runtime.AddTransient[iMixed](b, func(godi.Provider) iMixed { return &mixedSvc{tag: "transient"} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })
	testutil.Equal(t, calls, 1)

	testutil.Equal(t, p.GetService(reflect.TypeFor[iMixed]()).(iMixed).Tag(), "transient")
	_ = p.GetKeyedServices(reflect.TypeFor[iMixed](), nil)
	testutil.Equal(t, calls, 1)
}

// TestRuntime_SingleLifetime_BehaviorUnchanged: several registrations of one
// type with one lifetime behave as before: the single resolve returns the last,
// the collection returns all in registration order, and the first resolution
// materializes the whole group so each factory runs exactly once.
func TestRuntime_SingleLifetime_BehaviorUnchanged(t *testing.T) {
	typ := reflect.TypeFor[iMixed]()

	t.Run("singletons", func(t *testing.T) {
		calls := map[string]int{}
		b := runtime.NewBuilder()
		for _, tag := range []string{"one", "two", "three"} {
			runtime.AddSingleton[iMixed](b, func(godi.Provider) iMixed { calls[tag]++; return &mixedSvc{tag: tag} })
		}
		p, err := b.Build()
		testutil.NoError(t, err)
		t.Cleanup(func() { _ = p.Close() })
		s := p.CreateScope()
		t.Cleanup(func() { _ = s.Close() })

		last := p.GetService(typ)
		testutil.Equal(t, last.(iMixed).Tag(), "three")
		all := p.GetKeyedServices(typ, nil)
		testutil.DeepEqual(t, tags(all), []string{"one", "two", "three"})
		testutil.True(t, all[2] == last, "the collection's last element must be the single-resolve instance")
		fromScope := s.GetKeyedServices(typ, nil)
		for i := range all {
			testutil.True(t, fromScope[i] == all[i], "a scope must see the root's singletons")
		}
		testutil.True(t, s.GetService(typ) == last, "a scope's single resolve must be the root's singleton")
		testutil.DeepEqual(t, calls, map[string]int{"one": 1, "two": 1, "three": 1})
	})

	t.Run("scoped", func(t *testing.T) {
		calls := map[string]int{}
		b := runtime.NewBuilder()
		for _, tag := range []string{"one", "two", "three"} {
			runtime.AddScoped[iMixed](b, func(godi.Provider) iMixed { calls[tag]++; return &mixedSvc{tag: tag} })
		}
		p, err := b.Build()
		testutil.NoError(t, err)
		t.Cleanup(func() { _ = p.Close() })
		s := p.CreateScope()
		t.Cleanup(func() { _ = s.Close() })

		last := s.GetService(typ)
		testutil.Equal(t, last.(iMixed).Tag(), "three")
		// The first resolution materialized the whole scoped group.
		testutil.DeepEqual(t, calls, map[string]int{"one": 1, "two": 1, "three": 1})
		all := s.GetKeyedServices(typ, nil)
		testutil.DeepEqual(t, tags(all), []string{"one", "two", "three"})
		testutil.True(t, all[2] == last, "the collection's last element must be the single-resolve instance")
		testutil.DeepEqual(t, calls, map[string]int{"one": 1, "two": 1, "three": 1})
	})

	t.Run("transients", func(t *testing.T) {
		calls := map[string]int{}
		b := runtime.NewBuilder()
		for _, tag := range []string{"one", "two", "three"} {
			runtime.AddTransient[iMixed](b, func(godi.Provider) iMixed { calls[tag]++; return &mixedSvc{tag: tag} })
		}
		p, err := b.Build()
		testutil.NoError(t, err)
		t.Cleanup(func() { _ = p.Close() })

		testutil.Equal(t, p.GetService(typ).(iMixed).Tag(), "three")
		testutil.DeepEqual(t, calls, map[string]int{"three": 1})
		testutil.DeepEqual(t, tags(p.GetKeyedServices(typ, nil)), []string{"one", "two", "three"})
		testutil.DeepEqual(t, calls, map[string]int{"one": 1, "two": 1, "three": 2})
	})
}
