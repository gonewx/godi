// ATDD tests for Story 5.3 — runtime package: circular dependency detection
// via materializing guard in Build(). Cycles during eager singleton
// materialization are converted from fatal goroutine stack overflow into
// recoverable panics caught by Build()'s defer-recover, producing a build error.
package runtime_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

type cycleA struct{}
type cycleB struct{}

// TestBuild_Cycle_SelfDependency: A's factory resolves A → circular. Build must
// return an error instead of fatal stack overflow.
func TestBuild_Cycle_SelfDependency(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton(b, func(p godi.Provider) *cycleA {
		p.GetService(reflect.TypeFor[*cycleA]())
		return &cycleA{}
	})

	provider, err := b.Build()
	testutil.Error(t, err)
	testutil.Nil(t, provider)
	testutil.True(t, strings.Contains(err.Error(), "circular dependency"),
		"Build error must mention circular dependency")
}

// TestBuild_Cycle_DirectAB: A's factory resolves B, B's factory resolves A →
// circular. Build must return an error.
func TestBuild_Cycle_DirectAB(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton(b, func(p godi.Provider) *cycleA {
		p.GetService(reflect.TypeFor[*cycleB]())
		return &cycleA{}
	})
	runtime.AddSingleton(b, func(p godi.Provider) *cycleB {
		p.GetService(reflect.TypeFor[*cycleA]())
		return &cycleB{}
	})

	provider, err := b.Build()
	testutil.Error(t, err)
	testutil.Nil(t, provider)
	testutil.True(t, strings.Contains(err.Error(), "circular dependency"),
		"Build error must mention circular dependency")
}

// TestBuild_Cycle_NoCycle_ValidGraph: A depends on B (no cycle). Build must
// succeed.
func TestBuild_Cycle_NoCycle_ValidGraph(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton(b, func(p godi.Provider) *cycleB {
		return &cycleB{}
	})
	runtime.AddSingleton(b, func(p godi.Provider) *cycleA {
		p.GetService(reflect.TypeFor[*cycleB]())
		return &cycleA{}
	})

	provider, err := b.Build()
	testutil.NoError(t, err)
	testutil.NotNil(t, provider)
}
