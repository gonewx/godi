// ATDD tests for Story 5.3 — runtime package: circular dependency detection
// via materializing guard in Build(). Cycles during eager singleton
// materialization are converted from fatal goroutine stack overflow into
// recoverable panics caught by Build()'s defer-recover, producing a build error.
package runtime_test

import (
	"fmt"
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
	msg := err.Error()
	testutil.True(t, strings.Contains(msg, "circular dependency"),
		"Build error must mention circular dependency")
	testutil.True(t, strings.Contains(msg, "cycleA"),
		fmt.Sprintf("Build error must mention the offending type cycleA (got: %s)", msg))
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
	msg := err.Error()
	testutil.True(t, strings.Contains(msg, "circular dependency"),
		"Build error must mention circular dependency")
	testutil.True(t, strings.Contains(msg, "cycleA") || strings.Contains(msg, "cycleB"),
		fmt.Sprintf("Build error must mention an offending type cycleA or cycleB (got: %s)", msg))
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

// TestBuild_Cycle_KeyedSingleton: keyed singleton A(key) self-resolves →
// circular. Build must return an error mentioning keyed type + key, exercising
// the materializing guard on the keyed singleton group path
// (resolver.keyedGroup in runtime/provider.go).
func TestBuild_Cycle_KeyedSingleton(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddKeyedSingleton(b, "k1", func(p godi.Provider) *cycleA {
		p.GetKeyedService(reflect.TypeFor[*cycleA](), "k1")
		return &cycleA{}
	})

	provider, err := b.Build()
	testutil.Error(t, err)
	testutil.Nil(t, provider)
	msg := err.Error()
	testutil.True(t, strings.Contains(msg, "circular dependency"),
		"Build error must mention circular dependency")
	testutil.True(t, strings.Contains(msg, "keyed"),
		fmt.Sprintf("Build error must mention keyed (got: %s)", msg))
	testutil.True(t, strings.Contains(msg, "cycleA"),
		fmt.Sprintf("Build error must mention offending type cycleA (got: %s)", msg))
	testutil.True(t, strings.Contains(msg, "k1"),
		fmt.Sprintf("Build error must mention the offending key k1 (got: %s)", msg))
}

// TestBuild_Cycle_KeyedSingleton_DirectAB: keyed A(k1) resolves keyed B(k2),
// keyed B(k2) resolves keyed A(k1) → circular. Exercises the cross-key materializing
// guard.
func TestBuild_Cycle_KeyedSingleton_DirectAB(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddKeyedSingleton(b, "k1", func(p godi.Provider) *cycleA {
		p.GetKeyedService(reflect.TypeFor[*cycleB](), "k2")
		return &cycleA{}
	})
	runtime.AddKeyedSingleton(b, "k2", func(p godi.Provider) *cycleB {
		p.GetKeyedService(reflect.TypeFor[*cycleA](), "k1")
		return &cycleB{}
	})

	provider, err := b.Build()
	testutil.Error(t, err)
	testutil.Nil(t, provider)
	msg := err.Error()
	testutil.True(t, strings.Contains(msg, "circular dependency"),
		"Build error must mention circular dependency")
	testutil.True(t, strings.Contains(msg, "keyed"),
		fmt.Sprintf("Build error must mention keyed (got: %s)", msg))
}
