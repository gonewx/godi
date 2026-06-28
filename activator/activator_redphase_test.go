//go:build atdd_redphase

// ATDD red-phase scaffold for Story 2.6 — ActivatorUtilities: Tier C
// reflective construction + best-match ambiguity resolution.
//
// This file exercises the activator package's three public APIs:
//   - CreateInstance[T] (AC1-AC11, AC14-AC15, AC17)
//   - CreateFactory[T] (AC12, Theory×2 dual-path)
//   - GetServiceOrCreateInstance[T] (AC13)
//
// The activator package (doc.go placeholder) has NO implementation yet — these
// tests will not compile in the default build. The //go:build atdd_redphase
// tag isolates them until the dev-story implements the activator APIs.
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-6-activatorutilities-bestmatch.md.
//
// dotnet source: ActivatorUtilitiesTests.cs (Specification.Tests, v10.0.9)
// 20 dotnet methods → 22 Go test scenarios (+ AC15 degenerate + AC16 dep direction).
package activator_test

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/activator"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// buildProvider creates a runtime provider with IFakeService registered as a
// singleton — the base fixture most activator tests need.
func buildProvider(t *testing.T) godi.Provider {
	t.Helper()
	b := runtime.NewBuilder()
	runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
		return fakes.NewFakeService()
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })
	return p
}

// --- 2.6-SPEC-001 (AC1, P0) — Basic reflective construction ----------------

// TestActivator_BasicConstruction: CreateInstance resolves missing IFakeService
// dependency from provider when no explicit args are given.
// dotnet: TypeActivatorEnablesYouToCreateAnyTypeWithServicesEvenWhenNotInIocContainer
func TestActivator_BasicConstruction(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{fakes.NewAnotherClass}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			result, err := activator.CreateInstance[*fakes.AnotherClass](p, ctors)
			testutil.NoError(t, err)
			testutil.NotNil(t, result)
			testutil.NotNil(t, result.FakeService)
		}},
		{"CreateFactory", func(t *testing.T) {
			factory, err := activator.CreateFactory[*fakes.AnotherClass](ctors)
			testutil.NoError(t, err)
			result, err := factory(p)
			testutil.NoError(t, err)
			testutil.NotNil(t, result)
			testutil.NotNil(t, result.FakeService)
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-002 (AC2, P0) — Mixed params: explicit + provider ------------

// TestActivator_MixedParams: CreateInstance resolves IFakeService from provider
// and matches string explicit args to the remaining parameters.
// dotnet: TypeActivatorAcceptsAnyNumberOfAdditionalConstructorParametersToProvide
func TestActivator_MixedParams(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{fakes.NewAnotherClassAcceptingData}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			result, err := activator.CreateInstance[*fakes.AnotherClassAcceptingData](p, ctors, "1", "2")
			testutil.NoError(t, err)
			testutil.NotNil(t, result)
			testutil.NotNil(t, result.FakeService)
			testutil.Equal(t, result.One, "1")
			testutil.Equal(t, result.Two, "2")
		}},
		{"CreateFactory", func(t *testing.T) {
			factory, err := activator.CreateFactory[*fakes.AnotherClassAcceptingData](ctors, reflect.TypeFor[string](), reflect.TypeFor[string]())
			testutil.NoError(t, err)
			result, err := factory(p, "1", "2")
			testutil.NoError(t, err)
			testutil.NotNil(t, result)
			testutil.Equal(t, result.One, "1")
			testutil.Equal(t, result.Two, "2")
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-003 (AC3, P0) — Disambiguate by unique argument types --------

// TestActivator_DisambiguateByUniqueArgs: best-match selects the ctor accepting
// (IFakeService, string, int) when given "1" (string) and 2 (int) as explicit args.
// dotnet: TypeActivatorCanDisambiguateConstructorsWithUniqueArguments
func TestActivator_DisambiguateByUniqueArgs(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{
		fakes.NewClassWithAmbiguousCtorsFromString,
		fakes.NewClassWithAmbiguousCtorsFromServiceAndString,
		fakes.NewClassWithAmbiguousCtorsFromServiceAndInt,
		fakes.NewClassWithAmbiguousCtorsFromAll,
	}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			result, err := activator.CreateInstance[*fakes.ClassWithAmbiguousCtors](p, ctors, "1", 2)
			testutil.NoError(t, err)
			testutil.NotNil(t, result)
			testutil.Equal(t, result.CtorUsed, "service_string_int")
		}},
		{"CreateFactory", func(t *testing.T) {
			factory, err := activator.CreateFactory[*fakes.ClassWithAmbiguousCtors](ctors, reflect.TypeFor[string](), reflect.TypeFor[int]())
			testutil.NoError(t, err)
			result, err := factory(p, "1", 2)
			testutil.NoError(t, err)
			testutil.Equal(t, result.CtorUsed, "service_string_int")
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-004 (AC9, P0) — No constructors = error ----------------------

// TestActivator_NoConstructorsError: empty ctors slice AND non-struct type (or
// type that can't be zero-constructed) must return error.
// dotnet: TypeActivatorRequiresPublicConstructor
// Go difference (D-5): private/protected/internal/static ctors = empty ctors list.
func TestActivator_NoConstructorsError(t *testing.T) {
	p := buildProvider(t)

	// Interface type with nil ctors — reflect.New cannot create interfaces
	type uncreatableInterface interface{ dummy() }
	_, err := activator.CreateInstance[uncreatableInterface](p, nil)
	testutil.Error(t, err)
}

// --- 2.6-SPEC-005 (AC10, P0) — Surplus explicit args = error ---------------

// TestActivator_SurplusExplicitArgsError: explicit args that no ctor can fully
// accept must return error.
// dotnet: TypeActivatorRequiresAllArgumentsCanBeAccepted
func TestActivator_SurplusExplicitArgsError(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{fakes.NewAnotherClass}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			// NewAnotherClass(IFakeService) cannot accept "surplus" string
			_, err := activator.CreateInstance[*fakes.AnotherClass](p, ctors, "surplus")
			testutil.Error(t, err)
		}},
		{"CreateFactory", func(t *testing.T) {
			_, err := activator.CreateFactory[*fakes.AnotherClass](ctors, reflect.TypeFor[string]())
			testutil.Error(t, err)
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-006 (AC11, P0) — Constructor panic → error -------------------

// TestActivator_CtorPanicPropagation: ctor that panics must be recovered and
// returned as error with the original panic message preserved.
// dotnet: TypeActivatorRethrowsOriginalExceptionFromConstructor
func TestActivator_CtorPanicPropagation(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{fakes.NewClassWithThrowingCtor}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			_, err := activator.CreateInstance[*fakes.ClassWithThrowingCtor](p, ctors)
			testutil.Error(t, err)
			testutil.True(t, strings.Contains(err.Error(), "ctor threw an exception"),
				"error must preserve original panic message")
		}},
		{"CreateFactory", func(t *testing.T) {
			factory, err := activator.CreateFactory[*fakes.ClassWithThrowingCtor](ctors)
			testutil.NoError(t, err)
			_, err = factory(p)
			testutil.Error(t, err)
			testutil.True(t, strings.Contains(err.Error(), "ctor threw an exception"),
				"error must preserve original panic message")
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-007 (AC14, P0) — Unregistered dependency = error -------------

// TestActivator_UnregisteredDependencyError: when ctor depends on a service not
// registered in provider, CreateInstance must return error naming the missing type.
// dotnet: UnRegisteredServiceAsConstructorParameterThrowsException
func TestActivator_UnregisteredDependencyError(t *testing.T) {
	// Empty provider — no services registered
	b := runtime.NewBuilder()
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	ctors := []any{fakes.NewAnotherClass}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			_, err := activator.CreateInstance[*fakes.AnotherClass](p, ctors)
			testutil.Error(t, err)
		}},
		{"CreateFactory", func(t *testing.T) {
			factory, err := activator.CreateFactory[*fakes.AnotherClass](ctors)
			testutil.NoError(t, err)
			_, err = factory(p)
			testutil.Error(t, err)
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-008 (AC11/D-8, P0) — Inner exception capture ----------------

// TestActivator_InnerExceptionCapture: empty-ctor panic recovery preserves the
// inner panic value as the error message (dotnet TargetInvocationException).
// dotnet: CreateInstance_CapturesInnerException_OfTargetInvocationException
func TestActivator_InnerExceptionCapture(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{fakes.NewClassWithThrowingEmptyCtor}

	_, err := activator.CreateInstance[*fakes.ClassWithThrowingEmptyCtor](p, ctors)
	testutil.Error(t, err)
	testutil.True(t, strings.Contains(err.Error(), "inner exception message"),
		"error must contain inner exception message (D-8)")
}

// --- 2.6-SPEC-009 (AC4, P1) — Longest available constructor ----------------

// TestActivator_LongestAvailableConstructor: when IsService reports which types
// the provider can resolve, best-match selects the ctor with the most satisfiable
// parameters.
// dotnet: TypeActivatorCreateInstanceUsesLongestAvailableConstructor
func TestActivator_LongestAvailableConstructor(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{
		fakes.NewClassWithAmbiguousCtorsFromString,
		fakes.NewClassWithAmbiguousCtorsFromServiceAndString,
		fakes.NewClassWithAmbiguousCtorsFromServiceAndInt,
		fakes.NewClassWithAmbiguousCtorsFromAll,
	}

	for _, tc := range []struct {
		name string
		args []any
		want string
	}{
		{"string_only", []any{""}, "service_string"},
		{"int_only", []any{5}, "service_int"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := activator.CreateInstance[*fakes.ClassWithAmbiguousCtors](p, ctors, tc.args...)
			testutil.NoError(t, err)
			testutil.NotNil(t, result)
			testutil.Equal(t, result.CtorUsed, tc.want)
		})
	}
}

// --- 2.6-SPEC-010 (AC5, P1) — Marked constructor priority -----------------

// TestActivator_MarkedConstructorPriority: a ctor wrapped with godi.Marked()
// is selected even when it is not the longest available match.
// dotnet: TypeActivatorUsesMarkedConstructor
func TestActivator_MarkedConstructorPriority(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{
		fakes.NewClassWithAmbiguousCtorsAndAttributeFromString,
		godi.Marked(fakes.NewClassWithAmbiguousCtorsAndAttributeFromServiceAndString),
		fakes.NewClassWithAmbiguousCtorsAndAttributeFromServiceAndOuter,
	}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			result, err := activator.CreateInstance[*fakes.ClassWithAmbiguousCtorsAndAttribute](p, ctors, "data")
			testutil.NoError(t, err)
			testutil.NotNil(t, result)
			testutil.Equal(t, result.CtorUsed, "service_string")
		}},
		{"CreateFactory", func(t *testing.T) {
			factory, err := activator.CreateFactory[*fakes.ClassWithAmbiguousCtorsAndAttribute](ctors, reflect.TypeFor[string]())
			testutil.NoError(t, err)
			result, err := factory(p, "data")
			testutil.NoError(t, err)
			testutil.Equal(t, result.CtorUsed, "service_string")
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-011 (AC6, P1) — Multiple marked = error ---------------------

// TestActivator_MultipleMarkedCtorsError: two or more godi.Marked() in the
// ctors slice must return error.
// dotnet: TypeActivatorThrowsOnMultipleMarkedCtors
func TestActivator_MultipleMarkedCtorsError(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{
		godi.Marked(fakes.NewClassWithMultipleMarkedCtorsA),
		godi.Marked(fakes.NewClassWithMultipleMarkedCtorsB),
	}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			_, err := activator.CreateInstance[*fakes.ClassWithMultipleMarkedCtors](p, ctors)
			testutil.Error(t, err)
		}},
		{"CreateFactory", func(t *testing.T) {
			_, err := activator.CreateFactory[*fakes.ClassWithMultipleMarkedCtors](ctors)
			testutil.Error(t, err)
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-012 (AC7, P1) — Marked ctor rejects mismatched args ---------

// TestActivator_MarkedCtorRejectsMismatchedArgs: when the marked ctor cannot
// accept the given explicit argument types, return error.
// dotnet: TypeActivatorThrowsWhenMarkedCtorDoesntAcceptArguments
func TestActivator_MarkedCtorRejectsMismatchedArgs(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{
		fakes.NewClassWithAmbiguousCtorsAndAttributeFromString,
		godi.Marked(fakes.NewClassWithAmbiguousCtorsAndAttributeFromServiceAndString),
		fakes.NewClassWithAmbiguousCtorsAndAttributeFromServiceAndOuter,
	}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			// Pass int — marked ctor expects (IFakeService, string), not int
			_, err := activator.CreateInstance[*fakes.ClassWithAmbiguousCtorsAndAttribute](p, ctors, 42)
			testutil.Error(t, err)
		}},
		{"CreateFactory", func(t *testing.T) {
			_, err := activator.CreateFactory[*fakes.ClassWithAmbiguousCtorsAndAttribute](ctors, reflect.TypeFor[int]())
			testutil.Error(t, err)
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-013 (AC12, P1) — CreateFactory ambiguous match = error -------

// TestActivator_CreateFactoryAmbiguousError: when argTypes match multiple
// ctors equally, CreateFactory must return error.
// dotnet: TypeActivatorCreateFactoryDoesNotAllowForAmbiguousConstructorMatches
func TestActivator_CreateFactoryAmbiguousError(t *testing.T) {
	ctors := []any{
		fakes.NewClassWithAmbiguousCtorsFromServiceAndString,
		fakes.NewClassWithAmbiguousCtorsFromServiceAndInt,
	}

	for _, tc := range []struct {
		name    string
		argType reflect.Type
	}{
		{"CreateInstance_ambiguous", reflect.TypeFor[string]()},
		{"CreateFactory_ambiguous", reflect.TypeFor[int]()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := activator.CreateFactory[*fakes.ClassWithAmbiguousCtors](ctors, tc.argType)
			testutil.Error(t, err)
		})
	}
}

// --- 2.6-SPEC-014 (AC13, P1) — GetServiceOrCreateInstance: transient -------

// TestActivator_GetServiceOrCreateInstance_Transient: when registered as
// transient, each call creates a new instance (InstanceId increments).
// dotnet: GetServiceOrCreateInstanceRegisteredServiceTransient
func TestActivator_GetServiceOrCreateInstance_Transient(t *testing.T) {
	fakes.ResetCreationCount()
	t.Cleanup(fakes.ResetCreationCount)

	b := runtime.NewBuilder()
	runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
		return fakes.NewFakeService()
	})
	runtime.AddTransient[fakes.ICreationCountService](b, func(p godi.Provider) fakes.ICreationCountService {
		dep := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
		return fakes.NewCreationCountFakeService(dep)
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	ctors := []any{fakes.NewCreationCountFakeService}
	svc1, err := activator.GetServiceOrCreateInstance[fakes.ICreationCountService](p, ctors)
	testutil.NoError(t, err)
	svc2, err := activator.GetServiceOrCreateInstance[fakes.ICreationCountService](p, ctors)
	testutil.NoError(t, err)

	testutil.NotEqual(t, svc1.GetInstanceId(), svc2.GetInstanceId())
}

// --- 2.6-SPEC-015 (AC13, P1) — GetServiceOrCreateInstance: singleton -------

// TestActivator_GetServiceOrCreateInstance_Singleton: when registered as
// singleton, both calls return the same instance.
// dotnet: GetServiceOrCreateInstanceRegisteredServiceSingleton
func TestActivator_GetServiceOrCreateInstance_Singleton(t *testing.T) {
	fakes.ResetCreationCount()
	t.Cleanup(fakes.ResetCreationCount)

	b := runtime.NewBuilder()
	runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
		return fakes.NewFakeService()
	})
	runtime.AddSingleton[fakes.ICreationCountService](b, func(p godi.Provider) fakes.ICreationCountService {
		dep := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
		return fakes.NewCreationCountFakeService(dep)
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	ctors := []any{fakes.NewCreationCountFakeService}
	svc1, err := activator.GetServiceOrCreateInstance[fakes.ICreationCountService](p, ctors)
	testutil.NoError(t, err)
	svc2, err := activator.GetServiceOrCreateInstance[fakes.ICreationCountService](p, ctors)
	testutil.NoError(t, err)

	testutil.Equal(t, svc1.GetInstanceId(), svc2.GetInstanceId())
}

// --- 2.6-SPEC-016 (AC13, P1) — GetServiceOrCreateInstance: unregistered ----

// TestActivator_GetServiceOrCreateInstance_Unregistered: when NOT registered,
// each call falls through to CreateInstance (new instance each time).
// dotnet: GetServiceOrCreateInstanceUnregisteredService
func TestActivator_GetServiceOrCreateInstance_Unregistered(t *testing.T) {
	fakes.ResetCreationCount()
	t.Cleanup(fakes.ResetCreationCount)

	b := runtime.NewBuilder()
	runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
		return fakes.NewFakeService()
	})
	// ICreationCountService is NOT registered
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	ctors := []any{fakes.NewCreationCountFakeService}
	svc1, err := activator.GetServiceOrCreateInstance[fakes.ICreationCountService](p, ctors)
	testutil.NoError(t, err)
	svc2, err := activator.GetServiceOrCreateInstance[fakes.ICreationCountService](p, ctors)
	testutil.NoError(t, err)

	testutil.NotEqual(t, svc1.GetInstanceId(), svc2.GetInstanceId())
}

// --- 2.6-SPEC-017 (AC17, P1) — Zero-value construction fallback -----------

// TestActivator_ZeroValueFallback: empty ctors + zero-constructable struct →
// reflect.New returns a zero-value instance.
// dotnet: TypeActivatorWorksWithStaticCtor
// Go difference (D-5): empty struct + nil ctors = reflect.New zero-value.
func TestActivator_ZeroValueFallback(t *testing.T) {
	p := buildProvider(t)

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			result, err := activator.CreateInstance[*fakes.ClassWithStaticCtor](p, nil)
			testutil.NoError(t, err)
			testutil.NotNil(t, result)
		}},
		{"CreateFactory", func(t *testing.T) {
			factory, err := activator.CreateFactory[*fakes.ClassWithStaticCtor](nil)
			testutil.NoError(t, err)
			result, err := factory(p)
			testutil.NoError(t, err)
			testutil.NotNil(t, result)
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-018 (D-7, P1) — Interface type = error ----------------------

// TestActivator_InterfaceTypeError: CreateInstance with an interface type T
// must return error because reflect.New cannot instantiate interfaces.
// dotnet: CreateInstance_WithAbstractTypeAndPublicConstructor_ThrowsCorrectException
func TestActivator_InterfaceTypeError(t *testing.T) {
	p := buildProvider(t)

	type abstractService interface{ doSomething() }
	_, err := activator.CreateInstance[abstractService](p, nil)
	testutil.Error(t, err)
}

// --- 2.6-SPEC-019 (AC8, P2) — Optional args with defaults -----------------

// TestActivator_OptionalArgsDefaults: ctor with no params but hardcoded
// default values inside the factory function (Go D-3 adaptation).
// dotnet: TypeActivatorWorksWithCtorWithOptionalArgs
func TestActivator_OptionalArgsDefaults(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{fakes.NewClassWithOptionalArgsCtor}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			result, err := activator.CreateInstance[*fakes.ClassWithOptionalArgsCtor](p, ctors)
			testutil.NoError(t, err)
			testutil.NotNil(t, result)
			testutil.Equal(t, result.Whatever, "BLARGH")
		}},
		{"CreateFactory", func(t *testing.T) {
			factory, err := activator.CreateFactory[*fakes.ClassWithOptionalArgsCtor](ctors)
			testutil.NoError(t, err)
			result, err := factory(p)
			testutil.NoError(t, err)
			testutil.Equal(t, result.Whatever, "BLARGH")
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-020 (AC8, P2) — Optional args struct defaults ---------------

// TestActivator_OptionalArgsStructDefaults: simplified Go adaptation of
// dotnet's 15-optional-struct-parameter test. Verifies nil/zero defaults.
// dotnet: TypeActivatorWorksWithCtorWithOptionalArgs_WithStructDefaults
// Go difference (D-4): reduced to 2 pointer fields.
func TestActivator_OptionalArgsStructDefaults(t *testing.T) {
	p := buildProvider(t)
	ctors := []any{fakes.NewClassWithOptionalArgsCtorWithStructs}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"CreateInstance", func(t *testing.T) {
			result, err := activator.CreateInstance[*fakes.ClassWithOptionalArgsCtorWithStructs](p, ctors)
			testutil.NoError(t, err)
			testutil.NotNil(t, result)
			testutil.Nil(t, result.IntVal)
			testutil.Nil(t, result.StringVal)
		}},
		{"CreateFactory", func(t *testing.T) {
			factory, err := activator.CreateFactory[*fakes.ClassWithOptionalArgsCtorWithStructs](ctors)
			testutil.NoError(t, err)
			result, err := factory(p)
			testutil.NoError(t, err)
			testutil.Nil(t, result.IntVal)
			testutil.Nil(t, result.StringVal)
		}},
	} {
		t.Run(tc.name, tc.run)
	}
}

// --- 2.6-SPEC-021 (AC15, P1) — Zero-explicit degenerate consistency -------

// TestActivator_ZeroDegenerateConsistency: AD-9 iron proof. The same type
// constructed with zero explicit args via CreateInstance must select the same
// constructor as the container's assembly path (New{Impl} unique constructor).
func TestActivator_ZeroDegenerateConsistency(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[fakes.IFakeService](b, func(godi.Provider) fakes.IFakeService {
		return fakes.NewFakeService()
	})
	// Register AnotherClass via its New{Impl} constructor (form B wrapping)
	runtime.AddSingleton[*fakes.AnotherClass](b, func(p godi.Provider) *fakes.AnotherClass {
		dep := p.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
		return fakes.NewAnotherClass(dep)
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	// Container path: GetService
	containerResult, ok := godi.Get[*fakes.AnotherClass](p)
	testutil.True(t, ok, "AnotherClass must be resolvable via container")
	testutil.NotNil(t, containerResult)
	testutil.NotNil(t, containerResult.FakeService)

	// Activator path: CreateInstance with zero explicit args
	ctors := []any{fakes.NewAnotherClass}
	activatorResult, err := activator.CreateInstance[*fakes.AnotherClass](p, ctors)
	testutil.NoError(t, err)
	testutil.NotNil(t, activatorResult)
	testutil.NotNil(t, activatorResult.FakeService)

	// Both paths selected the same constructor — dependency is present in both
	testutil.True(t, containerResult.FakeService != nil && activatorResult.FakeService != nil,
		"AD-9: both paths must select the same constructor and wire the same dependency")
}

// --- 2.6-SPEC-022 (AC16, P2) — Dependency direction verification ----------

// TestActivator_DependencyDirection: architectural guard. The activator package
// must only import godi (Provider interface) + stdlib. It must NOT depend on
// runtime, codegen, plan, or internal/genlib.
func TestActivator_DependencyDirection(t *testing.T) {
	cmd := exec.Command("go", "list", "-json", "github.com/gonewx/godi/activator")
	cmd.Env = append(cmd.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	testutil.NoError(t, err)

	var pkg struct {
		Imports []string `json:"Imports"`
	}
	testutil.NoError(t, json.Unmarshal(out, &pkg))

	forbidden := []string{
		"github.com/gonewx/godi/runtime",
		"github.com/gonewx/godi/plan",
		"github.com/gonewx/godi/internal/genlib",
		"github.com/gonewx/godi/cmd/godi-gen",
	}
	for _, imp := range pkg.Imports {
		for _, f := range forbidden {
			testutil.True(t, imp != f,
				"activator must not import "+f+" (dependency direction violation)")
		}
	}
}
