// ATDD scaffold for Story 6.5 — ActivatorUtilities IMPL-LAYER
// construction-selection detail (dotnet DI.Tests/ActivatorUtilitiesTests.cs).
//
// This file is DISTINCT from activator_redphase_test.go (Story 2.6, which ported
// the Specification.Tests/ActivatorUtilitiesTests.cs — 20 spec methods, already
// green). See Story 6.5 D-1: two same-named dotnet files, different test cases.
// This file ports the impl-layer file's 17 Fact + 3 Theory = 20 non-conditional
// methods that exercise best-match SELECTION DETAIL (longest-available ctor,
// default-ctor degeneracy, marked-ctor priority over longest, same-length
// ambiguity, factory slots, error paths).
//
// Most scenarios verify the CORRECTNESS of the already-implemented activator
// best-match kernel (activator.go / factory.go from Story 2.6) — expected mostly
// GREEN once the build tag is removed. A minority are genuine RED→GREEN:
//   - #6/#7: explicit nil-provider check + message containing "provider" (AC4)
//   - #19:   ambiguity error message must name the type + ctor length (AC2)
//
// Adapted scenarios (#3, #5, #20, #21) carry documented Go divergence — see
// conformance/ACTIVATOR_DIVERGENCE.md.
//
// Activation: delete the //go:build line and the following blank line.
// The tests run in the activator package against a runtime provider — the
// activator only depends on the godi.Provider interface (Story 2.6, D-2).
//
// dotnet source: DI.Tests/ActivatorUtilitiesTests.cs (v10.0.9).
// Threshold accounting: conformance/threshold_actutil_test.go (20 methods).
package activator_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/activator"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// buildImplProvider builds a runtime provider, applying register to add the
// marker singletons a given test needs. The activator consumes it purely through
// the godi.Provider interface (D-2 dependency direction).
func buildImplProvider(t *testing.T, register func(b *runtime.Builder)) godi.Provider {
	t.Helper()
	b := runtime.NewBuilder()
	if register != nil {
		register(b)
	}
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })
	return p
}

// --- 6.5-IMPL-001 (#1, AC1, P0) — longest available ctor -------------------

// TestActivatorImpl_LongestAvailableConstructor: B and S resolvable from
// provider, A and C passed explicitly → the 4-param ctor (A,B,C,S) is the
// longest satisfiable and is selected; every field is populated.
// dotnet: CreateInstance_ClassWithABCS_UsesTheLongestAvailableConstructor
func TestActivatorImpl_LongestAvailableConstructor(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.B](b, &fakes.B{})
		runtime.AddSingletonInstance[*fakes.S](b, &fakes.S{})
	})
	a := &fakes.A{}
	c := &fakes.C{}
	ctors := []any{fakes.NewClassWithABCS4, fakes.NewClassWithABCS3}

	instance, err := activator.CreateInstance[*fakes.ClassWithABCS](p, ctors, a, c)
	testutil.NoError(t, err)
	testutil.NotNil(t, instance.B)
	testutil.NotNil(t, instance.S)
	testutil.True(t, instance.A == a, "A must be the explicitly-passed instance")
	testutil.True(t, instance.C == c, "C must be the explicitly-passed instance")
}

// --- 6.5-IMPL-002 (#2, AC1, P0) — single ctor is registered ----------------

// TestActivatorImpl_OneCtorIsRegistered: a single-ctor type constructs directly
// with its dependency resolved from the provider.
// dotnet: CreateInstance_OneCtor_IsRegistered_CreatesInstanceSuccessfully
func TestActivatorImpl_OneCtorIsRegistered(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.A](b, &fakes.A{})
	})
	ctors := []any{fakes.NewClassWithA}

	instance, err := activator.CreateInstance[*fakes.ClassWithA](p, ctors)
	testutil.NoError(t, err)
	testutil.NotNil(t, instance.A)
}

// --- 6.5-IMPL-003 (#3, AC4, P1, ADAPTED) — unambiguous construct succeeds ---

// TestActivatorImpl_UnambiguousConstructionSucceeds: dotnet verifies that even a
// badly-configured IServiceProviderIsService still allows an unambiguous
// construction and that IsService was invoked. In godi, IsService is an
// intrinsic Provider method (AD-16), NOT an injectable/replaceable service, so
// the "badly configured IsService" premise does not exist. ADAPTED: we retain
// the core assertion (unambiguous single-ctor construction succeeds); the
// IsServiceGotCalled probe is N/A. See ACTIVATOR_DIVERGENCE.md.
// dotnet: CreateInstance_BadlyConfiguredIServiceProviderIsService_ProperlyCreatesUnambiguousInstance
func TestActivatorImpl_UnambiguousConstructionSucceeds(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.A](b, &fakes.A{})
	})
	ctors := []any{fakes.NewClassWithA}

	instance, err := activator.CreateInstance[*fakes.ClassWithA](p, ctors)
	testutil.NoError(t, err)
	testutil.NotNil(t, instance)
	testutil.NotNil(t, instance.A)
	// N/A (AD-16): no injectable IServiceProviderIsService to probe for a
	// "IsServiceGotCalled" flag — IsService is a Provider-intrinsic method.
}

// --- 6.5-IMPL-004 (#4, AC1, P0, Theory×3) — different definition orders -----

// TestActivatorImpl_DifferentOrders: S resolvable from provider, B and C passed
// explicitly. Across three derived types with different ctor parameter orders,
// the (B,C,S) ctor is the longest satisfiable → A stays nil, B/C are the passed
// instances, S is resolved.
// dotnet: CreateInstance_DifferentOrders_CreatesInstanceSuccessfully (ABCS1/2/3)
func TestActivatorImpl_DifferentOrders(t *testing.T) {
	b := &fakes.B{}
	c := &fakes.C{}

	for _, tc := range []struct {
		name  string
		ctors []any
		// run constructs the derived type and returns its observable A/B/C/S.
		run func(t *testing.T, p godi.Provider) (aNil bool, gotB *fakes.B, gotC *fakes.C, sNotNil bool)
	}{
		{"ABCS1", []any{fakes.NewABCS1FromABC, fakes.NewABCS1FromBCS},
			func(t *testing.T, p godi.Provider) (bool, *fakes.B, *fakes.C, bool) {
				inst, err := activator.CreateInstance[*fakes.ABCS1](p, []any{fakes.NewABCS1FromABC, fakes.NewABCS1FromBCS}, b, c)
				testutil.NoError(t, err)
				return inst.A == nil, inst.B, inst.C, inst.S != nil
			}},
		{"ABCS2", []any{fakes.NewABCS2FromBAC, fakes.NewABCS2FromBSC},
			func(t *testing.T, p godi.Provider) (bool, *fakes.B, *fakes.C, bool) {
				inst, err := activator.CreateInstance[*fakes.ABCS2](p, []any{fakes.NewABCS2FromBAC, fakes.NewABCS2FromBSC}, b, c)
				testutil.NoError(t, err)
				return inst.A == nil, inst.B, inst.C, inst.S != nil
			}},
		{"ABCS3", []any{fakes.NewABCS3FromBSC, fakes.NewABCS3FromABC},
			func(t *testing.T, p godi.Provider) (bool, *fakes.B, *fakes.C, bool) {
				inst, err := activator.CreateInstance[*fakes.ABCS3](p, []any{fakes.NewABCS3FromBSC, fakes.NewABCS3FromABC}, b, c)
				testutil.NoError(t, err)
				return inst.A == nil, inst.B, inst.C, inst.S != nil
			}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := buildImplProvider(t, func(bld *runtime.Builder) {
				runtime.AddSingletonInstance[*fakes.S](bld, &fakes.S{})
			})
			aNil, gotB, gotC, sNotNil := tc.run(t, p)
			testutil.True(t, aNil, "A must be nil — (A,B,C) ctor not satisfiable, A unregistered/unpassed")
			testutil.True(t, gotB == b, "B must be the passed instance")
			testutil.True(t, gotC == c, "C must be the passed instance")
			testutil.True(t, sNotNil, "S must be resolved from the provider")
		})
	}
}

// --- 6.5-IMPL-005 (#5, AC4, P1, ADAPTED) — bad input surfaces as error ------

// TestActivatorImpl_BadInputSurfacesAsError: dotnet passes a typed null B as an
// explicit instance and expects InvalidOperationException. Go has no typed-null
// interface value that participates in argument matching the same way — an
// untyped nil arg loses its type and matches no parameter, so no ctor is
// satisfiable and construction fails cleanly. ADAPTED: the core intent ("bad
// input → clean error, not a deep NPE") is preserved; the precise
// null-typed-as-B mechanism is N/A. See ACTIVATOR_DIVERGENCE.md.
// dotnet: CreateInstance_NullInstance_HandlesBadInputWithInvalidOperationException
func TestActivatorImpl_BadInputSurfacesAsError(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.S](b, &fakes.S{})
	})
	// Only S is registered; pass an untyped-nil (unusable) arg plus C. Neither
	// (A,B,C) nor (B,C,S) ctor can be fully satisfied → error.
	ctors := []any{fakes.NewABCS1FromABC, fakes.NewABCS1FromBCS}

	_, err := activator.CreateInstance[*fakes.ABCS1](p, ctors, nil, &fakes.C{})
	testutil.Error(t, err)
}

// --- 6.5-IMPL-006 (#6, AC4, P0, RED→GREEN) — nil provider (CreateInstance) --

// TestActivatorImpl_NilProvider_CreateInstance: CreateInstance with a nil
// provider must fail with an explicit, provider-named error (mapping dotnet's
// ArgumentNullException). Drives adding an explicit nil-provider guard rather
// than relying on a recovered nil-deref panic. RED until AC4 guard is added.
// dotnet: TypeActivatorThrowsOnNullProvider
func TestActivatorImpl_NilProvider_CreateInstance(t *testing.T) {
	ctors := []any{fakes.NewClassWithABCS4, fakes.NewClassWithABCS3}

	_, err := activator.CreateInstance[*fakes.ClassWithABCS](nil, ctors, "hello")
	testutil.Error(t, err)
	testutil.True(t, strings.Contains(err.Error(), "provider"),
		"nil-provider error must explicitly name the provider (AC4)")
}

// --- 6.5-IMPL-007 (#7, AC4, P0, RED→GREEN) — nil provider (factory) ---------

// TestActivatorImpl_NilProvider_Factory: CreateFactory succeeds (compilation is
// provider-independent), but invoking the returned factory with a nil provider
// must fail with a provider-named error. RED until AC4 guard is added inside the
// factory closure.
// dotnet: FactoryActivatorThrowsOnNullProvider
func TestActivatorImpl_NilProvider_Factory(t *testing.T) {
	ctors := []any{fakes.NewClassWithA}

	factory, err := activator.CreateFactory[*fakes.ClassWithA](ctors)
	testutil.NoError(t, err)

	_, err = factory(nil)
	testutil.Error(t, err)
	testutil.True(t, strings.Contains(err.Error(), "provider"),
		"nil-provider error from factory must explicitly name the provider (AC4)")
}

// --- 6.5-IMPL-008 (#8, AC1, P0) — parameter order doesn't matter -----------

// TestActivatorImpl_ParameterOrderDoesntMatter: identical to #1 but the explicit
// args are passed in reversed order (c, a). Selection is by parameter TYPE, so
// the longest ctor is still chosen and every field populated.
// dotnet: CreateInstance_ClassWithABCS_UsesTheLongestAvailableConstructor_ParameterOrderDoesntMatter
func TestActivatorImpl_ParameterOrderDoesntMatter(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.B](b, &fakes.B{})
		runtime.AddSingletonInstance[*fakes.S](b, &fakes.S{})
	})
	a := &fakes.A{}
	c := &fakes.C{}
	ctors := []any{fakes.NewClassWithABCS4, fakes.NewClassWithABCS3}

	instance, err := activator.CreateInstance[*fakes.ClassWithABCS](p, ctors, c, a)
	testutil.NoError(t, err)
	testutil.NotNil(t, instance.B)
	testutil.NotNil(t, instance.S)
	testutil.True(t, instance.A == a, "A must be the explicitly-passed instance regardless of arg order")
	testutil.True(t, instance.C == c, "C must be the explicitly-passed instance regardless of arg order")
}

// --- 6.5-IMPL-009 (#9, AC1, P0, Theory×2) — default ctor degeneracy --------

// TestActivatorImpl_ChoosesDefaultConstructor: empty provider, no explicit args.
// No parameterized ctor is satisfiable, so best-match degenerates to the
// parameterless ctor regardless of its declaration position.
// dotnet: CreateInstance_ClassWithABC_ChoosesDefaultConstructorNoMatterCtorOrder
func TestActivatorImpl_ChoosesDefaultConstructor(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(t *testing.T, p godi.Provider) (aNil, bNil bool)
	}{
		{"DefaultConstructorFirst", func(t *testing.T, p godi.Provider) (bool, bool) {
			ctors := []any{
				fakes.NewClassWithABC_DefaultFirst0,
				fakes.NewClassWithABC_DefaultFirst1,
				fakes.NewClassWithABC_DefaultFirst2,
				fakes.NewClassWithABC_DefaultFirst3,
			}
			inst, err := activator.CreateInstance[*fakes.ClassWithABC_DefaultConstructorFirst](p, ctors)
			testutil.NoError(t, err)
			testutil.NotNil(t, inst)
			return inst.A == nil, inst.B == nil
		}},
		{"DefaultConstructorLast", func(t *testing.T, p godi.Provider) (bool, bool) {
			ctors := []any{
				fakes.NewClassWithABC_DefaultLast3,
				fakes.NewClassWithABC_DefaultLast2,
				fakes.NewClassWithABC_DefaultLast1,
				fakes.NewClassWithABC_DefaultLast0,
			}
			inst, err := activator.CreateInstance[*fakes.ClassWithABC_DefaultConstructorLast](p, ctors)
			testutil.NoError(t, err)
			testutil.NotNil(t, inst)
			return inst.A == nil, inst.B == nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := buildImplProvider(t, nil) // empty provider
			aNil, bNil := tc.run(t, p)
			testutil.True(t, aNil, "A must be nil — default (parameterless) ctor selected")
			testutil.True(t, bNil, "B must be nil — default (parameterless) ctor selected")
		})
	}
}

// --- 6.5-IMPL-010 (#10, AC1, P0) — longest ctor over registered+passed -----

// TestActivatorImpl_BNotRegistered_UsesLongestPossible: S registered, A and C
// passed explicitly, B NOT registered. The 4-param (A,B,C,S) ctor is
// unsatisfiable (B missing), so the 3-param (A,C,S) ctor — the longest that
// covers all registered + passed params — is selected → B stays nil.
// dotnet: CreateInstance_ClassWithABCS_BNotRegistered_UsesLongestPossibleCtorTakingAllRegisteredAndPassedParameters
func TestActivatorImpl_BNotRegistered_UsesLongestPossible(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.S](b, &fakes.S{})
	})
	a := &fakes.A{}
	c := &fakes.C{}
	ctors := []any{fakes.NewClassWithABCS4, fakes.NewClassWithABCS3}

	instance, err := activator.CreateInstance[*fakes.ClassWithABCS](p, ctors, c, a)
	testutil.NoError(t, err)
	testutil.True(t, instance.A == a, "A must be the passed instance")
	testutil.True(t, instance.C == c, "C must be the passed instance")
	testutil.NotNil(t, instance.S)
	testutil.Nil(t, instance.B)
}

// --- 6.5-IMPL-011 (#11, AC2, P0, Theory×2) — marked ctor picked, order-free -

// TestActivatorImpl_ConstructorWithAttribute_PicksMarked: A registered, B and C
// passed explicitly. The (A,B,C) ctor is marked (godi.Marked) — the marked
// candidate set is selected regardless of declaration order → A is the
// registered instance.
// dotnet: CreateInstance_ClassWithABC_ConstructorWithAttribute_PicksCtorWithAttr_NoMatterDefinitionOrder
func TestActivatorImpl_ConstructorWithAttribute_PicksMarked(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(t *testing.T, p godi.Provider, a *fakes.A) *fakes.A // returns selected instance's A
	}{
		{"FirstConstructorWithAttribute", func(t *testing.T, p godi.Provider, a *fakes.A) *fakes.A {
			ctors := []any{
				godi.Marked(fakes.NewClassWithABC_FirstAttr3),
				fakes.NewClassWithABC_FirstAttr2,
			}
			inst, err := activator.CreateInstance[*fakes.ClassWithABC_FirstConstructorWithAttribute](p, ctors, &fakes.B{}, &fakes.C{})
			testutil.NoError(t, err)
			return inst.A
		}},
		{"LastConstructorWithAttribute", func(t *testing.T, p godi.Provider, a *fakes.A) *fakes.A {
			ctors := []any{
				fakes.NewClassWithABC_LastAttr2,
				godi.Marked(fakes.NewClassWithABC_LastAttr3),
			}
			inst, err := activator.CreateInstance[*fakes.ClassWithABC_LastConstructorWithAttribute](p, ctors, &fakes.B{}, &fakes.C{})
			testutil.NoError(t, err)
			return inst.A
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &fakes.A{}
			p := buildImplProvider(t, func(b *runtime.Builder) {
				runtime.AddSingletonInstance[*fakes.A](b, a)
			})
			gotA := tc.run(t, p, a)
			testutil.True(t, gotA == a, "marked (A,B,C) ctor must be selected → A is the registered instance")
		})
	}
}

// --- 6.5-IMPL-013 (#13, AC2, P0) — ambiguous ctor, no marker → error -------

// TestActivatorImpl_FailsWithAmbiguousConstructor: A and B registered, ctors (A)
// and (B) both length-1 and satisfiable, neither marked → ambiguous → error.
// dotnet: CreateInstanceFailsWithAmbiguousConstructor
func TestActivatorImpl_FailsWithAmbiguousConstructor(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.A](b, &fakes.A{})
		runtime.AddSingletonInstance[*fakes.B](b, &fakes.B{})
	})
	ctors := []any{fakes.NewClassWithA_And_B_FromA, fakes.NewClassWithA_And_B_FromB}

	_, err := activator.CreateInstance[*fakes.ClassWithA_And_B](p, ctors)
	testutil.Error(t, err)
}

// --- 6.5-IMPL-014 (#14, AC2, P0) — ambiguous ctor, reversed order → error ---

// TestActivatorImpl_FailsWithAmbiguousConstructor_ReversedOrder: same as #13 but
// the ctors are declared in reversed order — still ambiguous → error.
// dotnet: CreateInstanceFailsWithAmbiguousConstructor_ReversedOrder
func TestActivatorImpl_FailsWithAmbiguousConstructor_ReversedOrder(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.A](b, &fakes.A{})
		runtime.AddSingletonInstance[*fakes.B](b, &fakes.B{})
	})
	ctors := []any{fakes.NewClassWithB_And_A_FromB, fakes.NewClassWithB_And_A_FromA}

	_, err := activator.CreateInstance[*fakes.ClassWithB_And_A](p, ctors)
	testutil.Error(t, err)
}

// --- 6.5-IMPL-015 (#15, AC2, P0) — marker breaks same-length ambiguity ------

// TestActivatorImpl_PassesWithAmbiguousConstructor: A and B registered, ctors
// (A)[marked] and (B). Marking (A) forms the sole candidate set → (A) selected,
// A non-nil.
// dotnet: CreateInstancePassesWithAmbiguousConstructor
func TestActivatorImpl_PassesWithAmbiguousConstructor(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.A](b, &fakes.A{})
		runtime.AddSingletonInstance[*fakes.B](b, &fakes.B{})
	})
	ctors := []any{
		godi.Marked(fakes.NewClassWithA_And_B_Attr_FromA),
		fakes.NewClassWithA_And_B_Attr_FromB,
	}

	instance, err := activator.CreateInstance[*fakes.ClassWithA_And_B_Attr](p, ctors)
	testutil.NoError(t, err)
	testutil.NotNil(t, instance.A)
}

// --- 6.5-IMPL-016 (#16, AC2, P0) — marker breaks ambiguity, reversed --------

// TestActivatorImpl_PassesWithAmbiguousConstructor_ReversedOrder: same as #15
// with (B) declared first, (A) marked and declared last → (A) still selected.
// dotnet: CreateInstancePassesWithAmbiguousConstructor_ReversedOrder
func TestActivatorImpl_PassesWithAmbiguousConstructor_ReversedOrder(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.A](b, &fakes.A{})
		runtime.AddSingletonInstance[*fakes.B](b, &fakes.B{})
	})
	ctors := []any{
		fakes.NewClassWithB_And_A_Attr_FromB,
		godi.Marked(fakes.NewClassWithB_And_A_Attr_FromA),
	}

	instance, err := activator.CreateInstance[*fakes.ClassWithB_And_A_Attr](p, ctors)
	testutil.NoError(t, err)
	testutil.NotNil(t, instance.A)
}

// --- 6.5-IMPL-017 (#17, AC2, P0) — marker priority overrides longest --------

// TestActivatorImpl_IgnoresAttributeForLongest: A and B registered, ctors
// (A)[marked] and (A,B). The marked candidate set is {(A)}, so best-match
// selects (A) even though (A,B) is longer → A non-nil, B STAYS NIL. This is the
// core "marked priority beats longest" invariant (D-3).
// dotnet: CreateInstanceIgnoresActivatorUtilitiesConstructorAttribute
func TestActivatorImpl_IgnoresAttributeForLongest(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.A](b, &fakes.A{})
		runtime.AddSingletonInstance[*fakes.B](b, &fakes.B{})
	})
	ctors := []any{
		godi.Marked(fakes.NewClassWithA_And_AB_Attr_FromA),
		fakes.NewClassWithA_And_AB_Attr_FromAB,
	}

	instance, err := activator.CreateInstance[*fakes.ClassWithA_And_AB_Attr](p, ctors)
	testutil.NoError(t, err)
	testutil.NotNil(t, instance.A)
	testutil.Nil(t, instance.B) // marked short ctor wins over longer unmarked ctor
}

// --- 6.5-IMPL-018 (#18, AC2, P0) — marker priority, reversed order ----------

// TestActivatorImpl_IgnoresAttributeForLongest_ReversedOrder: same as #17 with
// (A,B) declared first, (A) marked and declared last → (A) still selected, B nil.
// dotnet: CreateInstanceIgnoresActivatorUtilitiesConstructorAttribute_ReversedOrder
func TestActivatorImpl_IgnoresAttributeForLongest_ReversedOrder(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.A](b, &fakes.A{})
		runtime.AddSingletonInstance[*fakes.B](b, &fakes.B{})
	})
	ctors := []any{
		fakes.NewClassWithAB_And_A_Attr_FromAB,
		godi.Marked(fakes.NewClassWithAB_And_A_Attr_FromA),
	}

	instance, err := activator.CreateInstance[*fakes.ClassWithAB_And_A_Attr](p, ctors)
	testutil.NoError(t, err)
	testutil.NotNil(t, instance.A)
	testutil.Nil(t, instance.B)
}

// --- 6.5-IMPL-019 (#19, AC2, P0, RED→GREEN) — same-length ambiguity message -

// TestActivatorImpl_MultipleCtorsWithSameLength_ThrowsAmbiguous: A and B
// registered, ctors (), (A), (B). Two length-1 ctors are equally satisfiable →
// ambiguous. dotnet requires the message to name the type and the ctor length
// ("...were found with length 1."). RED until the activator's ambiguity error is
// enriched with the target type name + length (AC2, continuing the 2.6 AC14
// error-message patch).
// dotnet: CreateInstance_ClassWithABC_MultipleCtorsWithSameLength_ThrowsAmbiguous
func TestActivatorImpl_MultipleCtorsWithSameLength_ThrowsAmbiguous(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.A](b, &fakes.A{})
		runtime.AddSingletonInstance[*fakes.B](b, &fakes.B{})
	})
	ctors := []any{
		fakes.NewClassWithABC_SameLen0,
		fakes.NewClassWithABC_SameLenA,
		fakes.NewClassWithABC_SameLenB,
	}

	_, err := activator.CreateInstance[*fakes.ClassWithABC_MultipleCtorsWithSameLength](p, ctors)
	testutil.Error(t, err)
	testutil.True(t, strings.Contains(err.Error(), "ClassWithABC_MultipleCtorsWithSameLength"),
		"ambiguity error must name the target type (AC2)")
	testutil.True(t, strings.Contains(err.Error(), "length") || strings.Contains(err.Error(), "1"),
		"ambiguity error must convey the ctor length (AC2)")
}

// --- 6.5-IMPL-020 (#20, AC3, P1, ADAPTED) — CreateFactory 4 types, 3 injected

// TestActivatorImpl_CreateFactory_4Types_3Injected: factory declares an explicit
// [*B] slot; A, C, S resolve from the provider, B is passed at call time. All
// four fields are populated. ADAPTED: dotnet has both CreateFactory(Type,...) →
// ObjectFactory and CreateFactory<T> → ObjectFactory<T>; godi has only the
// generic CreateFactory[T] returning a func value — the Assert.IsType<ObjectFactory>
// type-shape assertions are N/A. See ACTIVATOR_DIVERGENCE.md.
// dotnet: CreateFactory_CreatesFactoryMethod_4Types_3Injected
func TestActivatorImpl_CreateFactory_4Types_3Injected(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.A](b, &fakes.A{})
		runtime.AddSingletonInstance[*fakes.C](b, &fakes.C{})
		runtime.AddSingletonInstance[*fakes.S](b, &fakes.S{})
	})
	ctors := []any{fakes.NewClassWithABCS4, fakes.NewClassWithABCS3}

	factory, err := activator.CreateFactory[*fakes.ClassWithABCS](ctors, reflect.TypeFor[*fakes.B]())
	testutil.NoError(t, err)

	instance, err := factory(p, &fakes.B{})
	testutil.NoError(t, err)
	testutil.NotNil(t, instance.A)
	testutil.NotNil(t, instance.B)
	testutil.NotNil(t, instance.C)
	testutil.NotNil(t, instance.S)
}

// --- 6.5-IMPL-021 (#21, AC3, P1, ADAPTED) — CreateFactory 5 types, 5 injected

// TestActivatorImpl_CreateFactory_5Types_5Injected: factory with no explicit
// slots; all five deps (A,B,C,S,Z) resolve from the provider. ADAPTED: dotnet's
// "5 is a threshold for fixed vs Span<> invoker args" is a reflection impl detail
// with no Go equivalent — semantics are equivalent, the threshold assertion is
// N/A. See ACTIVATOR_DIVERGENCE.md.
// dotnet: CreateFactory_CreatesFactoryMethod_5Types_5Injected
func TestActivatorImpl_CreateFactory_5Types_5Injected(t *testing.T) {
	p := buildImplProvider(t, func(b *runtime.Builder) {
		runtime.AddSingletonInstance[*fakes.A](b, &fakes.A{})
		runtime.AddSingletonInstance[*fakes.B](b, &fakes.B{})
		runtime.AddSingletonInstance[*fakes.C](b, &fakes.C{})
		runtime.AddSingletonInstance[*fakes.S](b, &fakes.S{})
		runtime.AddSingletonInstance[*fakes.Z](b, &fakes.Z{})
	})
	ctors := []any{fakes.NewClassWithABCSZ5}

	factory, err := activator.CreateFactory[*fakes.ClassWithABCSZ](ctors)
	testutil.NoError(t, err)

	instance, err := factory(p)
	testutil.NoError(t, err)
	testutil.NotNil(t, instance.A)
	testutil.NotNil(t, instance.B)
	testutil.NotNil(t, instance.C)
	testutil.NotNil(t, instance.S)
	testutil.NotNil(t, instance.Z)
}
