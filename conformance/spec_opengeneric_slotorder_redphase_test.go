// ATDD scaffold for Story 4.3 (AC1-AC5, Task 3) — conformance suite:
// open/closed PRIORITY and ENUMERABLE slot ORDER for generic services across
// BOTH providers (runtime oracle + codegen B2), covering dotnet touch-points
// 6/7/8 of the 9 open-generic contact points.
//
// Ported from dotnet DependencyInjectionSpecificationTests (v10.0.9):
//
//	ClosedServicesPreferredOverOpenGenericServices            (:748, touch-point 6) → AC1 (4.3-SPEC-001)
//	ResolvingEnumerableContainingOpenGenericServiceUsesCorrectSlot (:766, tp 7, GH#79938) → AC2 (4.3-SPEC-002)
//	ResolvesMixedOpenClosedGenericsAsEnumerable               (:908, touch-point 8) → AC3 (4.3-SPEC-003 / -004)
//	(AD-15 cross-provider differential equivalence)           → AC5 (4.3-SPEC-DIFF / -SEQ)
//
// CORE ARCHITECTURE POINT (read 4-3 Dev Notes 本质 + D-1..D-5): godi has NO
// open/closed distinction at the generator level (AD-4/AD-10: no MakeGenericType —
// the generator sees only CLOSED registrations). dotnet's "exact closed preferred
// over open generic" is an order-INSENSITIVE RUNTIME rule baked into
// CallSiteFactory; godi has NO such rule. Slot order / priority are expressed by
// REGISTRATION ORDER instead:
//   - single resolve = LAST-WINS (the closed-specific impl registered LAST wins)
//   - enumerable      = REGISTRATION-ORDER slots
//
// This reproduces "closed preferred" (touch-point 6/7, D-5: register closed AFTER
// open) and "mixed enumerable" (touch-point 8, D-3: all transient to avoid the
// 2.4 mixed-lifetime unhandled axis) entirely from the registration source — a
// deliberate AD-4/AD-10 tradeoff, NOT a defect, explicitly annotated below.
//
// This file began as an ATDD red-phase scaffold behind //go:build atdd_redphase;
// that constraint was removed (and the suite joined the default build) once the
// Task 1 Fakes and the Task 2 registration + regen landed.
//
// Reused from the same package (do NOT redefine): addOpenGenericRT[T]
// (spec_opengeneric_redphase_test.go, the 4.1 injected open helper), dynTypeName
// (spec_test.go), join (spec_multiimpl_redphase_test.go).
//
// Test standard R4: pointer identity WITHIN one provider only; ACROSS providers
// use provider-stable observables (int counts, ordered dynamic-type-name strings).
package conformance

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// addClosedFakeOpenGenericRT is the runtime-provider counterpart of the codegen
// direct-closed form-A registration of the closed-SPECIFIC impl (AD-14: both
// providers enumerate the SAME logical closed set, same order, never hand-forked).
// ClosedFakeOpenGenericService[T] takes NO injected value (mirrors dotnet's
// non-generic FakeService whose Value is default null) — so the factory just
// constructs it. Transient (D-3: same service type stays single-lifetime to avoid
// the 2.4 mixed-lifetime unhandled axis; lifetime does not affect slot order).
func addClosedFakeOpenGenericRT[T any](b *runtime.Builder) {
	runtime.AddTransient[fakes.IFakeOpenGenericService[T]](b, func(godi.Provider) fakes.IFakeOpenGenericService[T] {
		return fakes.NewClosedFakeOpenGenericService[T]()
	})
}

// addInstanceFakeOpenGenericRT registers the INSTANCE-slot impl for touch-point 8.
// godi cannot reuse the open-produced FakeOpenGenericService[T] for the instance
// slot (D-4: two form-A registrations of the SAME impl type collide on
// serviceName — a Go mechanism wall), so the instance slot uses a DISTINCT impl
// type. Observation is by dynamic type + slot position, replacing dotnet's
// instance-identity assertion (Assert.Equal(instance, enumerable[2])) which R4
// forbids across providers. Transient (D-3).
func addInstanceFakeOpenGenericRT[T any](b *runtime.Builder) {
	runtime.AddTransient[fakes.IFakeOpenGenericService[T]](b, func(godi.Provider) fakes.IFakeOpenGenericService[T] {
		return fakes.NewInstanceFakeOpenGenericService[T]()
	})
}

// rtFactoryOpenGenericSlot builds the runtime oracle. Registration ORDER IS the
// slot order — it MUST match cgFactoryOpenGenericSlot / registration.go item-for-
// item (AD-14 same logical closed set, same order).
var rtFactoryOpenGenericSlot = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()

		// --- touch-point 6/7 (IGenSlotA): closed preferred via registration ORDER.
		// element T injected into the open-produced impl:
		runtime.AddSingleton[fakes.IGenSlotA](b, func(godi.Provider) fakes.IGenSlotA {
			return fakes.NewGenSlotA()
		})
		addOpenGenericRT[fakes.IGenSlotA](b)           // open-produced — FIRST
		addClosedFakeOpenGenericRT[fakes.IGenSlotA](b) // closed — LAST → single-resolve last-wins = closed (D-5)

		// --- touch-point 8 (IGenSlotB): mixed enumerable in dotnet slot order
		// [closed#1, open-produced, instance]. ALL transient (D-3).
		runtime.AddSingleton[fakes.IGenSlotB](b, func(godi.Provider) fakes.IGenSlotB {
			return fakes.NewGenSlotB()
		})
		addClosedFakeOpenGenericRT[fakes.IGenSlotB](b)   // closed#1 → enumerable[0]
		addOpenGenericRT[fakes.IGenSlotB](b)             // open-produced → enumerable[1]
		addInstanceFakeOpenGenericRT[fakes.IGenSlotB](b) // instance → enumerable[2]

		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryOpenGenericSlot builds the codegen provider from the regenerated
// container_gen.go (Task 2). Red until the 4.3 registration block is appended and
// the container is regenerated.
var cgFactoryOpenGenericSlot = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// Provider-stable dynamic type names (R4 observables). Generic type args render as
// the fully-qualified element interface (mirrors 4.1's
// "*fakes.FakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IFakeService]").
const (
	dynSlotOpenA     = "*fakes.FakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IGenSlotA]"
	dynSlotClosedA   = "*fakes.ClosedFakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IGenSlotA]"
	dynSlotClosedB   = "*fakes.ClosedFakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IGenSlotB]"
	dynSlotOpenB     = "*fakes.FakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IGenSlotB]"
	dynSlotInstanceB = "*fakes.InstanceFakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IGenSlotB]"
)

func slotServiceA() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IGenSlotA]]()
}
func slotServiceB() reflect.Type {
	return reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IGenSlotB]]()
}

// containsDynType reports whether the collection contains an element whose
// dynamic type name equals name (order-insensitive set membership, the Go
// equivalent of dotnet's services.Any(s => s.GetType()==...)).
func containsDynType(all []any, name string) bool {
	for _, v := range all {
		if dynTypeName(v) == name {
			return true
		}
	}
	return false
}

// openGenericSlotSpecCases run against BOTH providers via RunSuite (AC5).
var openGenericSlotSpecCases = []SpecCase{
	// --- 4.3-SPEC-001 (AC1, P0) — touch-point 6: closed preferred (single, last-wins)
	{
		Name: "slot-order: single resolve of [IGenSlotA] returns closed-specific impl (closed preferred via last-wins; ClosedServicesPreferredOverOpenGenericServices, DISpec:748)",
		Run: func(t *testing.T, p Provider) {
			// Go difference (D-5): dotnet's "closed preferred" is order-insensitive;
			// godi reproduces it by registering the closed-specific impl LAST so
			// single-resolve last-wins returns it (NOT the open-produced impl).
			svc := p.GetService(slotServiceA())
			testutil.NotNil(t, svc)
			if svc == nil {
				return
			}
			testutil.Equal(t, dynTypeName(svc), dynSlotClosedA)
			// Scope parity: child scope must resolve identically.
			sc := p.CreateScope()
			t.Cleanup(func() { _ = sc.Close() })
			scSvc := sc.GetService(slotServiceA())
			testutil.NotNil(t, scSvc)
			if scSvc != nil {
				testutil.Equal(t, dynTypeName(scSvc), dynSlotClosedA)
			}
		},
	},
	// --- 4.3-SPEC-002 (AC2, P0) — touch-point 7: enumerable contains both, count=2
	{
		Name: "slot-order: enumerable of [IGenSlotA] has count=2 containing open+closed (order-insensitive .Any) and single-resolve stays closed (ResolvingEnumerableContainingOpenGenericServiceUsesCorrectSlot, DISpec:766, GH#79938)",
		Run: func(t *testing.T, p Provider) {
			all := p.GetKeyedServices(slotServiceA(), nil)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 2)
			// Order-INSENSITIVE membership (dotnet .Any): godi's [open, closed] order
			// is the reverse of dotnet's source order but still contains both types,
			// so it satisfies the dotnet assertion (D-5).
			testutil.True(t, containsDynType(all, dynSlotOpenA),
				"enumerable must contain the open-produced impl (order-insensitive)")
			testutil.True(t, containsDynType(all, dynSlotClosedA),
				"enumerable must contain the closed-specific impl (order-insensitive)")
			// Re-verify single-resolve is the closed impl (last-wins, consistent with SPEC-001).
			single := p.GetService(slotServiceA())
			testutil.NotNil(t, single)
			testutil.Equal(t, dynTypeName(single), dynSlotClosedA)
			// Scope parity: enumerable count + membership.
			sc := p.CreateScope()
			t.Cleanup(func() { _ = sc.Close() })
			scAll := sc.GetKeyedServices(slotServiceA(), nil)
			testutil.NotNil(t, scAll)
			testutil.Equal(t, len(scAll), 2)
			testutil.True(t, containsDynType(scAll, dynSlotOpenA),
				"scope enumerable must contain the open-produced impl")
			testutil.True(t, containsDynType(scAll, dynSlotClosedA),
				"scope enumerable must contain the closed-specific impl")
		},
	},
	// --- 4.3-SPEC-003 (AC3, P0) — touch-point 8: mixed enumerable, count=3, ORDERED
	{
		Name: "slot-order: mixed enumerable of [IGenSlotB] has count=3 in registration-order slots [closed#1, open, instance] (ResolvesMixedOpenClosedGenericsAsEnumerable, DISpec:908)",
		Run: func(t *testing.T, p Provider) {
			// Go difference (D-4): dotnet's [1] and [2] are the SAME type
			// FakeOpenGenericService<PocoClass> distinguished by instance identity;
			// godi uses a DISTINCT impl type for the instance slot (serviceName
			// collision wall) and observes by dynamic type + slot position.
			all := p.GetKeyedServices(slotServiceB(), nil)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 3)
			if len(all) != 3 {
				return
			}
			var names []string
			for _, v := range all {
				names = append(names, dynTypeName(v))
			}
			// Order-SENSITIVE: the registration-order slot sequence must be exact.
			testutil.Equal(t, join(names, ","),
				join([]string{dynSlotClosedB, dynSlotOpenB, dynSlotInstanceB}, ","))
			// Scope parity: count=3 ordered.
			sc := p.CreateScope()
			t.Cleanup(func() { _ = sc.Close() })
			scAll := sc.GetKeyedServices(slotServiceB(), nil)
			testutil.NotNil(t, scAll)
			testutil.Equal(t, len(scAll), 3)
			if len(scAll) == 3 {
				var scNames []string
				for _, v := range scAll {
					scNames = append(scNames, dynTypeName(v))
				}
				testutil.Equal(t, join(scNames, ","),
					join([]string{dynSlotClosedB, dynSlotOpenB, dynSlotInstanceB}, ","))
			}
		},
	},
	// --- 4.3-SPEC-004 (AC3 suppl., P1) — godi-internal invariant: single resolve = last
	{
		Name: "slot-order: single resolve of [IGenSlotB] returns the LAST-registered (instance) impl (godi last-wins invariant; dotnet does not test the single-resolve here — annotated)",
		Run: func(t *testing.T, p Provider) {
			// dotnet's touch-point 8 only asserts the enumerable; the single-resolve
			// last-wins outcome is a godi-internal robustness invariant (the instance
			// slot is registered LAST → wins). Explicitly annotated as godi-internal.
			single := p.GetService(slotServiceB())
			testutil.NotNil(t, single)
			if single == nil {
				return
			}
			testutil.Equal(t, dynTypeName(single), dynSlotInstanceB)
			// Scope parity: single resolve = last.
			sc := p.CreateScope()
			t.Cleanup(func() { _ = sc.Close() })
			scSingle := sc.GetService(slotServiceB())
			testutil.NotNil(t, scSingle)
			if scSingle != nil {
				testutil.Equal(t, dynTypeName(scSingle), dynSlotInstanceB)
			}
		},
	},
}

// --- 4.3-SPEC-RUN (AC1-AC3, P0) ---------------------------------------------

func TestConformanceOpenGenericSlot_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryOpenGenericSlot, openGenericSlotSpecCases)
}

func TestConformanceOpenGenericSlot_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryOpenGenericSlot, openGenericSlotSpecCases)
}

// --- 4.3-SPEC-DIFF (AC5, P0) — cross-provider differential equivalence -------

// TestConformanceOpenGenericSlot_DifferentialEquivalence: AD-15 cross-provider
// differential equivalence for the open-generic slot-order / priority axis (a
// behavior axis AD-15 explicitly names as a coverage obligation). Probes use
// provider-stable observables only — int counts, dynamic type-name strings, and
// order-insensitive bool membership — never pointer/instance identity across
// providers (R4).
func TestConformanceOpenGenericSlot_DifferentialEquivalence(t *testing.T) {
	rt, cg := rtFactoryOpenGenericSlot, cgFactoryOpenGenericSlot

	// touch-point 6/7: single resolve [IGenSlotA] == closed (last-wins).
	AssertEquivalent(t, rt, cg, func(p Provider) string {
		return dynTypeName(p.GetService(slotServiceA()))
	})
	// touch-point 7: enumerable [IGenSlotA] count == 2.
	AssertEquivalent(t, rt, cg, func(p Provider) int {
		all := p.GetKeyedServices(slotServiceA(), nil)
		if all == nil {
			return -1
		}
		return len(all)
	})
	// touch-point 7: enumerable contains open (order-insensitive bool).
	AssertEquivalent(t, rt, cg, func(p Provider) bool {
		return containsDynType(p.GetKeyedServices(slotServiceA(), nil), dynSlotOpenA)
	})
	// touch-point 7: enumerable contains closed (order-insensitive bool).
	AssertEquivalent(t, rt, cg, func(p Provider) bool {
		return containsDynType(p.GetKeyedServices(slotServiceA(), nil), dynSlotClosedA)
	})
	// touch-point 8: single resolve [IGenSlotB] == instance (last-wins).
	AssertEquivalent(t, rt, cg, func(p Provider) string {
		return dynTypeName(p.GetService(slotServiceB()))
	})
	// touch-point 8: enumerable [IGenSlotB] count == 3.
	AssertEquivalent(t, rt, cg, func(p Provider) int {
		all := p.GetKeyedServices(slotServiceB(), nil)
		if all == nil {
			return -1
		}
		return len(all)
	})
	// IsService consistency for the closed service types.
	AssertEquivalent(t, rt, cg, func(p Provider) bool {
		return p.IsService(slotServiceA())
	})
	AssertEquivalent(t, rt, cg, func(p Provider) bool {
		return p.IsService(slotServiceB())
	})
}

// --- 4.3-SPEC-SEQ (AC3/AC5, P0) — registration-order sequence equivalence -----

// TestConformanceOpenGenericSlot_RegistrationOrderSequence: the type-name
// sequence from the mixed [IGenSlotB] enumerable must be identical across BOTH
// providers — the strongest order-sensitive differential-equivalence assertion
// for touch-point 8 (AD-15 covers slot order as a coverage obligation). Mirrors
// spec_multiimpl_redphase_test.go's RegistrationOrderSequence.
func TestConformanceOpenGenericSlot_RegistrationOrderSequence(t *testing.T) {
	var sequences [2]string
	for i, factory := range []ProviderFactory{rtFactoryOpenGenericSlot, cgFactoryOpenGenericSlot} {
		p := factory.Build(t)
		t.Cleanup(func() { _ = p.Close() })

		all := p.GetKeyedServices(slotServiceB(), nil)
		var names []string
		for _, v := range all {
			names = append(names, reflect.TypeOf(v).String())
		}
		sequences[i] = join(names, ",")
	}
	testutil.Equal(t, sequences[0], sequences[1])
}
