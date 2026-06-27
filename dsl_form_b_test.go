// Story 2.1 (AC1) — form-B DSL marker contract (AddSingletonFactory).
//
// Form B is the escape hatch (AD-8): when an implementation has no `New{Impl}`
// constructor for form-A discovery, the developer registers it with a factory.
// Go has NO generic-function overloading (D-4), so the form-B marker CANNOT
// share the name `AddSingleton` with form A — it is `AddSingletonFactory`, a
// distinct static marker the generator recognizes by name and reads the factory
// argument expression from (AD-7, never executed):
//
//	godi.AddSingletonFactory[I any](r *godi.Registry, factory func(godi.Provider) I) *godi.Registry
package godi_test

import (
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
)

// iLoggerFB / loggerFB are local stand-ins so the form-B marker is exercised
// without depending on any real service type. loggerFB satisfies iLoggerFB so
// the factory's `func(godi.Provider) iLoggerFB` return is type-correct.
type iLoggerFB interface{ logFB() }

type loggerFB struct{}

func (loggerFB) logFB() {}

// TestDSLFormB_AddSingletonFactoryIsPureMarker covers 2.1-DSL-001 (AC1).
//
// AddSingletonFactory must be a PURE static marker (AD-7): it returns its
// receiver unchanged and has no runtime behavior. The generator reads the
// factory argument expression statically; the function is never executed.
func TestDSLFormB_AddSingletonFactoryIsPureMarker(t *testing.T) {
	var r godi.Registry

	factory := func(godi.Provider) iLoggerFB { return loggerFB{} }
	got := godi.AddSingletonFactory[iLoggerFB](&r, factory)

	testutil.NotNil(t, got)
	testutil.Equal(t, got, &r) // returns the same receiver, unchanged
}

// TestDSLFormB_CoexistsWithFormAChaining covers 2.1-DSL-002 (AC1).
//
// Form A (AddSingleton[I,Impl]) and form B (AddSingletonFactory[I]) are
// DISTINCT names (D-4: Go forbids overloading), so both can be chained against
// one registry. This pins the two-name design — a single shared `AddSingleton`
// for both forms would be a duplicate-declaration compile error.
func TestDSLFormB_CoexistsWithFormAChaining(t *testing.T) {
	var r godi.Registry

	// Chain: form A returns *Registry, which is the receiver form B accepts.
	got := godi.AddSingletonFactory[iLoggerFB](
		godi.AddSingleton[iLoggerFB, loggerFB](&r),
		func(godi.Provider) iLoggerFB { return loggerFB{} },
	)

	testutil.Equal(t, got, &r)
}
