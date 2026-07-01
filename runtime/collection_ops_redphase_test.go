// ATDD red-phase scaffold for Story 6.4 (AC2/AC3/AC4) — runtime package:
// collection operations on Builder (TryAdd, TryAddEnumerable, Replace,
// RemoveAll, instance registration, MakeReadOnly).
//
// These APIs DO NOT EXIST YET on runtime.Builder — this file will not compile
// until the dev-story implements them. The //go:build atdd_redphase tag isolates
// it from the default build.
//
// Ported from dotnet:
//   - ServiceCollectionServiceExtensionsTest (SCSE, 14 methods → AC2)
//   - ServiceCollectionKeyedServiceExtensionsTest (SCKSE, 19 methods → AC3)
//   - ServiceCollectionDescriptorExtensionsTests (SCDE, 4 methods → AC2)
//   - ServiceCollectionTests (SCT, 1 method → AC2 MakeReadOnly)
//
// Core decisions (D-1 方案γ, D-3 精确规则, D-4 instance, D-9 MakeReadOnly):
//   - Collection operations are RUNTIME-ONLY (codegen hard wall, AC4/AD-7)
//   - Descriptor semantics verified through OBSERVABLE RESOLVE (not property reflection)
//   - instance registration = AddSingletonInstance[T](b, inst) internally wraps form-B
//   - key = any (comparable), null key routes unkeyed (AD-17)
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-6-4-extensions-collection-ops.md.
package runtime_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// ── test-local fakes (unexported, minimal) ─────────────────────────────────

type iCollSvc interface{ isCollSvc() }
type collSvcA struct{ _ int }
type collSvcB struct{ _ int }
type collSvcC struct{ _ int }

func (*collSvcA) isCollSvc() {}
func (*collSvcB) isCollSvc() {}
func (*collSvcC) isCollSvc() {}

type iCollOther interface{ isCollOther() }
type collOtherImpl struct{ _ int }

func (*collOtherImpl) isCollOther() {}

// ══ AC2: Instance Registration (SCSE #3 adapted, D-4) ══════════════════════

func TestAddSingletonInstance_ResolvesIdenticalInstance(t *testing.T) {
	inst := &collSvcA{}
	b := runtime.NewBuilder()
	runtime.AddSingletonInstance[iCollSvc](b, inst)
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	v1 := p.GetService(reflect.TypeFor[iCollSvc]())
	v2 := p.GetService(reflect.TypeFor[iCollSvc]())
	testutil.True(t, v1 == v2, "instance registration must return same pointer within one provider")
	testutil.True(t, v1 == inst, "instance registration must return the original instance")
}

// ══ AC2: TryAdd (SCSE #4/#5/#6/#7, D-3 判重键=(ServiceType,ServiceKey)) ═══

func TestTryAddSingleton_NoOp_WhenExists(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.TryAddSingleton[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcB{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	v := p.GetService(reflect.TypeFor[iCollSvc]())
	testutil.Equal(t, reflect.TypeOf(v).String(), "*runtime_test.collSvcA")
}

func TestTryAddSingleton_Adds_WhenMissing(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.TryAddSingleton[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcB{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	v := p.GetService(reflect.TypeFor[iCollSvc]())
	testutil.NotNil(t, v)
	testutil.Equal(t, reflect.TypeOf(v).String(), "*runtime_test.collSvcB")
}

func TestTryAddScoped_NoOp_WhenExists(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddScoped[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.TryAddScoped[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcB{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	v := p.GetService(reflect.TypeFor[iCollSvc]())
	testutil.Equal(t, reflect.TypeOf(v).String(), "*runtime_test.collSvcA")
}

func TestTryAddTransient_NoOp_WhenExists(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddTransient[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.TryAddTransient[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcB{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	v := p.GetService(reflect.TypeFor[iCollSvc]())
	testutil.Equal(t, reflect.TypeOf(v).String(), "*runtime_test.collSvcA")
}

func TestTryAdd_DoesNotAddDuplicate_SameServiceType(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.TryAddSingleton[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcB{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	all := p.GetKeyedServices(reflect.TypeFor[iCollSvc](), nil)
	testutil.Equal(t, len(all), 1)
}

func TestTryAdd_AddsAndResolvable(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.TryAddSingleton[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.TryAddSingleton[iCollOther](b, func(godi.Provider) iCollOther { return &collOtherImpl{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	testutil.NotNil(t, p.GetService(reflect.TypeFor[iCollSvc]()))
	testutil.NotNil(t, p.GetService(reflect.TypeFor[iCollOther]()))
}

// ══ AC2: TryAddEnumerable (SCSE #8/#9/#10) ═════════════════════════════════

func TestTryAddEnumerable_AddsService(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.TryAddEnumerable[iCollSvc](b, runtime.Singleton, func(godi.Provider) iCollSvc { return &collSvcA{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	testutil.NotNil(t, p.GetService(reflect.TypeFor[iCollSvc]()))
}

func TestTryAddEnumerable_DoesNotAddDuplicate(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.TryAddEnumerable[iCollSvc](b, runtime.Singleton, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.TryAddEnumerable[iCollSvc](b, runtime.Singleton, func(godi.Provider) iCollSvc { return &collSvcA{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	all := p.GetKeyedServices(reflect.TypeFor[iCollSvc](), nil)
	testutil.Equal(t, len(all), 1)
}

// ══ AC2: Replace (SCSE #12/#13, D-3) ══════════════════════════════════════

func TestReplace_Adds_WhenNotRegistered(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.Replace[iCollSvc](b, runtime.Singleton, func(godi.Provider) iCollSvc { return &collSvcA{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	testutil.NotNil(t, p.GetService(reflect.TypeFor[iCollSvc]()))
}

func TestReplace_RemovesFirst_AppendsEnd(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.AddSingleton[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcB{} })
	runtime.Replace[iCollSvc](b, runtime.Singleton, func(godi.Provider) iCollSvc { return &collSvcC{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	all := p.GetKeyedServices(reflect.TypeFor[iCollSvc](), nil)
	testutil.Equal(t, len(all), 2)
	testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*runtime_test.collSvcB")
	testutil.Equal(t, reflect.TypeOf(all[1]).String(), "*runtime_test.collSvcC")
}

// ══ AC2: RemoveAll (SCSE #14, D-3 不对称性) ═══════════════════════════════

func TestRemoveAll_RemovesOnlyUnkeyed(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.AddKeyedSingleton[iCollSvc](b, "k1", func(godi.Provider) iCollSvc { return &collSvcB{} })
	runtime.RemoveAll[iCollSvc](b)
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	testutil.Nil(t, p.GetService(reflect.TypeFor[iCollSvc]()))
	testutil.NotNil(t, p.GetKeyedService(reflect.TypeFor[iCollSvc](), "k1"))
}

// ══ AC2: MakeReadOnly / IsReadOnly (SCT #1, D-9) ══════════════════════════

func TestMakeReadOnly_FreezesPreventsModification(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.MakeReadOnly(b)

	recovered := testutil.Panics(t, func() {
		runtime.AddSingleton[iCollOther](b, func(godi.Provider) iCollOther { return &collOtherImpl{} })
	})
	if e, ok := recovered.(error); ok {
		testutil.True(t, e != nil, "MakeReadOnly must prevent mutations with error/panic")
	}
}

func TestMakeReadOnly_Idempotent(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.MakeReadOnly(b)
	runtime.MakeReadOnly(b)
	testutil.True(t, b.IsReadOnly(), "second MakeReadOnly must be a no-op")
}

func TestBuild_ImplicitlyFreezesBuilder(t *testing.T) {
	b := runtime.NewBuilder()
	_, err := b.Build()
	testutil.NoError(t, err)
	testutil.True(t, b.IsReadOnly(), "Build must implicitly freeze builder")
}

// ══ AC3: Keyed TryAdd (SCKSE #4/#5/#6/#7) ═════════════════════════════════

func TestTryAddKeyedSingleton_NoOp_WhenKeyExists(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddKeyedSingleton[iCollSvc](b, "k1", func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.TryAddKeyedSingleton[iCollSvc](b, "k1", func(godi.Provider) iCollSvc { return &collSvcB{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	v := p.GetKeyedService(reflect.TypeFor[iCollSvc](), "k1")
	testutil.Equal(t, reflect.TypeOf(v).String(), "*runtime_test.collSvcA")
}

func TestTryAddKeyedSingleton_Adds_WhenKeyMissing(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.TryAddKeyedSingleton[iCollSvc](b, "k1", func(godi.Provider) iCollSvc { return &collSvcA{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	testutil.NotNil(t, p.GetKeyedService(reflect.TypeFor[iCollSvc](), "k1"))
}

// ══ AC3: Keyed Replace (SCKSE #12/#13/#14) ════════════════════════════════

func TestReplaceKeyed_RemovesFirst_AppendsEnd(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddKeyedSingleton[iCollSvc](b, "k", func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.AddKeyedSingleton[iCollSvc](b, "k", func(godi.Provider) iCollSvc { return &collSvcB{} })
	runtime.ReplaceKeyed[iCollSvc](b, "k", runtime.Singleton, func(godi.Provider) iCollSvc { return &collSvcC{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	all := p.GetKeyedServices(reflect.TypeFor[iCollSvc](), "k")
	testutil.Equal(t, len(all), 2)
	testutil.Equal(t, reflect.TypeOf(all[0]).String(), "*runtime_test.collSvcB")
	testutil.Equal(t, reflect.TypeOf(all[1]).String(), "*runtime_test.collSvcC")
}

func TestReplaceKeyed_WithIntKey(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddKeyedSingleton[iCollSvc](b, 42, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.ReplaceKeyed[iCollSvc](b, 42, runtime.Singleton, func(godi.Provider) iCollSvc { return &collSvcB{} })
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	v := p.GetKeyedService(reflect.TypeFor[iCollSvc](), 42)
	testutil.Equal(t, reflect.TypeOf(v).String(), "*runtime_test.collSvcB")
}

// ══ AC3: RemoveAllKeyed (SCKSE #15/#16/#17) ═══════════════════════════════

func TestRemoveAllKeyed_RemovesAllMatchingKey(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddKeyedSingleton[iCollSvc](b, "k1", func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.AddKeyedSingleton[iCollSvc](b, "k1", func(godi.Provider) iCollSvc { return &collSvcB{} })
	runtime.AddKeyedSingleton[iCollSvc](b, "k2", func(godi.Provider) iCollSvc { return &collSvcC{} })
	runtime.RemoveAllKeyed[iCollSvc](b, "k1")
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	testutil.True(t, p.GetKeyedService(reflect.TypeFor[iCollSvc](), "k1") == nil, "k1 should be removed")
	testutil.NotNil(t, p.GetKeyedService(reflect.TypeFor[iCollSvc](), "k2"))
}

func TestRemoveAllKeyed_WithIntKey(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddKeyedSingleton[iCollSvc](b, 9, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.AddKeyedSingleton[iCollSvc](b, 7, func(godi.Provider) iCollSvc { return &collSvcB{} })
	runtime.RemoveAllKeyed[iCollSvc](b, 9)
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	testutil.True(t, p.GetKeyedService(reflect.TypeFor[iCollSvc](), 9) == nil, "int key 9 should be removed")
	testutil.NotNil(t, p.GetKeyedService(reflect.TypeFor[iCollSvc](), 7))
}

// ══ AC3: Instance keyed (AddKeyedSingletonInstance) ════════════════════════

func TestAddKeyedSingletonInstance_ResolvesIdentical(t *testing.T) {
	inst := &collSvcA{}
	b := runtime.NewBuilder()
	runtime.AddKeyedSingletonInstance[iCollSvc](b, "inst-key", inst)
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	v := p.GetKeyedService(reflect.TypeFor[iCollSvc](), "inst-key")
	testutil.True(t, v == inst, "keyed instance registration must return same pointer")
}

// ══ AC3: null key routes unkeyed (SCKSE #18, already fixed in provider.go) ═

func TestAddKeyedSingletonInstance_NilKey_RoutesUnkeyed(t *testing.T) {
	inst := &collSvcA{}
	b := runtime.NewBuilder()
	runtime.AddKeyedSingletonInstance[iCollSvc](b, nil, inst)
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	v := p.GetService(reflect.TypeFor[iCollSvc]())
	testutil.True(t, v == inst, "nil key instance must route to unkeyed namespace")
}

// ══ AC3: AnyKey as registration key → panic (AD-17, keyed.go:33) ══════════

func TestAddKeyedSingleton_AnyKey_Panics(t *testing.T) {
	b := runtime.NewBuilder()
	testutil.Panics(t, func() {
		runtime.AddKeyedSingleton[iCollSvc](b, godi.AnyKey, func(godi.Provider) iCollSvc { return &collSvcA{} })
	})
}

// ══ SCDE: Add/Remove (SCDE #1/#2/#3) ══════════════════════════════════════

func TestRemove_RemovesRegistration(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iCollSvc](b, func(godi.Provider) iCollSvc { return &collSvcA{} })
	runtime.AddSingleton[iCollOther](b, func(godi.Provider) iCollOther { return &collOtherImpl{} })
	runtime.Remove[iCollSvc](b)
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	testutil.Nil(t, p.GetService(reflect.TypeFor[iCollSvc]()))
	testutil.NotNil(t, p.GetService(reflect.TypeFor[iCollOther]()))
}
