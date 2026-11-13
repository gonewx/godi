// Conformance suite for mixed-lifetime registrations under one service type
// (eval fix A4). dotnet resolves them by registration order: a single resolve
// returns the LAST registration whatever its lifetime, and a collection resolve
// returns every registration in registration order, each element keeping its
// own lifetime. Both providers are checked from the root and from a scope.
package conformance

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// rtFactoryMixedLifetime registers the same sequence as the codegen leg's
// registration.go: IMixedA singleton then transient, IMixedB transient then
// singleton.
var rtFactoryMixedLifetime = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()
		runtime.AddSingleton[fakes.IMixedA](b, func(godi.Provider) fakes.IMixedA { return fakes.NewMixedSingletonFirst() })
		runtime.AddTransient[fakes.IMixedA](b, func(godi.Provider) fakes.IMixedA { return fakes.NewMixedTransientLast() })
		runtime.AddTransient[fakes.IMixedB](b, func(godi.Provider) fakes.IMixedB { return fakes.NewMixedTransientFirst() })
		runtime.AddSingleton[fakes.IMixedB](b, func(godi.Provider) fakes.IMixedB { return fakes.NewMixedSingletonLast() })
		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// forEachMixedResolver runs check against the root and a scope of a fresh
// provider from each factory.
func forEachMixedResolver(t *testing.T, check func(t *testing.T, p Provider)) {
	t.Helper()
	for _, f := range []ProviderFactory{rtFactoryMixedLifetime, cgFactory} {
		t.Run(f.Name, func(t *testing.T) {
			p := f.Build(t)
			t.Cleanup(func() { _ = p.Close() })
			t.Run("root", func(t *testing.T) { check(t, p) })
			t.Run("scope", func(t *testing.T) {
				s := p.CreateScope()
				t.Cleanup(func() { _ = s.Close() })
				check(t, s)
			})
		})
	}
}

func mixedName(v any) string {
	if v == nil {
		return "<nil>"
	}
	return v.(fakes.IMixedLifetime).Name()
}

func mixedNames(vs []any) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = mixedName(v)
	}
	return out
}

func TestSpec_MixedLifetime_LastRegistrationWins(t *testing.T) {
	forEachMixedResolver(t, func(t *testing.T, p Provider) {
		testutil.Equal(t, godi.GetRequired[fakes.IMixedA](p).Name(), "MixedTransientLast")
		testutil.Equal(t, godi.GetRequired[fakes.IMixedB](p).Name(), "MixedSingletonLast")
	})
}

func TestSpec_MixedLifetime_GetServicesInRegistrationOrder(t *testing.T) {
	forEachMixedResolver(t, func(t *testing.T, p Provider) {
		testutil.DeepEqual(t, mixedNames(p.GetKeyedServices(reflect.TypeFor[fakes.IMixedA](), nil)),
			[]string{"MixedSingletonFirst", "MixedTransientLast"})
		testutil.DeepEqual(t, mixedNames(p.GetKeyedServices(reflect.TypeFor[fakes.IMixedB](), nil)),
			[]string{"MixedTransientFirst", "MixedSingletonLast"})
	})
}

func TestSpec_MixedLifetime_ElementsKeepTheirOwnLifetime(t *testing.T) {
	forEachMixedResolver(t, func(t *testing.T, p Provider) {
		ta := reflect.TypeFor[fakes.IMixedA]()
		first := p.GetKeyedServices(ta, nil)
		second := p.GetKeyedServices(ta, nil)
		if len(first) != 2 || len(second) != 2 {
			t.Fatalf("IMixedA collection: got %d and %d elements, want 2", len(first), len(second))
		}
		testutil.True(t, first[0] == second[0], "singleton element must be the same instance on every resolve")
		testutil.True(t, first[1] != second[1], "transient element must be a new instance on every resolve")

		// The singleton element is shared with the single resolve of IMixedB's
		// last (singleton) registration, and IMixedB's transient element is new.
		tb := reflect.TypeFor[fakes.IMixedB]()
		all := p.GetKeyedServices(tb, nil)
		if len(all) != 2 {
			t.Fatalf("IMixedB collection: got %d elements, want 2", len(all))
		}
		testutil.True(t, all[1] == p.GetService(tb), "IMixedB singleton element must be the instance GetService returns")
		testutil.True(t, all[0] != p.GetKeyedServices(tb, nil)[0], "IMixedB transient element must be new on every resolve")
	})
}
