// ATDD scaffold for Story 4.1 (AC2/AC4/AC6, Task 6) — codegen
// conformance: Path γ STRICT tests for open-generic closed materialization on
// the pre-generated B2 container (codegen.NewRoot() direct calls, not just
// differential equivalence).
//
// Path γ (AD-15): open-generic materialization is explicitly named a fidelity-
// sensitive capability that escalates to Path γ — it must be verified STRICTLY
// on the generated B2 container, not only via cross-provider differential
// equivalence. This file pins:
//
//	4.1-CGPATH-001 (AC2) — B2 resolves [IFakeService] with injected singleton (.Value()==singleton)
//	4.1-CGPATH-002 (AC4) — B2 resolves the SECOND instantiation [IFakeConsumer] (disambiguation, no field collision)
//	4.1-CGPATH-003 (AC3) — B2 reports unmaterialized closed type nil + IsService=false (no slot)
//	4.1-CGPATH-004 (AC6) — regen byte-identical (consecutive generate idempotent, AD-14)
//
// ACTIVATED (Story 4.1): the build tag was removed once Task 4 landed the codegen
// per-service helper registration + the second instantiation + regen, so the B2
// container resolves both closed instantiations with injected dependencies — this
// Path γ suite is now GREEN.
package codegen_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
)

const (
	dynCGFakeOpenGenericOfService  = "*fakes.FakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IFakeService]"
	dynCGFakeOpenGenericOfConsumer = "*fakes.FakeOpenGenericService[github.com/gonewx/godi/conformance/fakes.IFakeConsumer]"
)

// --- 4.1-CGPATH-001 (AC2, P0) — B2 closed resolution + dependency injection --

// TestCodegen_PathGamma_OpenGeneric_Injection: on the B2 container, the closed
// generic [IFakeService] resolves and its injected Value() is the SAME singleton
// the container resolves directly (dependency injection materialized correctly —
// dotnet OpenGenericServicesCanBeResolved). Pointer identity within one provider
// is legal (R4).
func TestCodegen_PathGamma_OpenGeneric_Injection(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	svc := root.GetService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]())
	testutil.NotNil(t, svc)
	if svc == nil {
		return
	}
	testutil.Equal(t, reflect.TypeOf(svc).String(), dynCGFakeOpenGenericOfService)

	injected := svc.(fakes.IFakeOpenGenericService[fakes.IFakeService]).Value()
	direct := root.GetService(reflect.TypeFor[fakes.IFakeService]()).(fakes.IFakeService)
	testutil.NotNil(t, injected)
	testutil.Equal(t, injected, direct) // same singleton: ctor wired the injected dep (AD-8)
}

// --- 4.1-CGPATH-002 (AC4, P0) — B2 second instantiation (disambiguation) -----

// TestCodegen_PathGamma_OpenGeneric_SecondInstantiation: on the B2 container,
// the SECOND closed instantiation [IFakeConsumer] of the same open generic
// resolves independently of [IFakeService]. Proves serviceName disambiguation
// (type args folded into the name + sanitizeFieldName) produced distinct,
// non-colliding fields/methods and the container compiles (deferred-work.md:57).
func TestCodegen_PathGamma_OpenGeneric_SecondInstantiation(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	a := root.GetService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeService]]())
	b := root.GetService(reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeConsumer]]())
	testutil.NotNil(t, a)
	testutil.NotNil(t, b)
	if a == nil || b == nil {
		return
	}
	testutil.Equal(t, reflect.TypeOf(a).String(), dynCGFakeOpenGenericOfService)
	testutil.Equal(t, reflect.TypeOf(b).String(), dynCGFakeOpenGenericOfConsumer)

	// Each instantiation injects its own dependency type.
	injB := b.(fakes.IFakeOpenGenericService[fakes.IFakeConsumer]).Value()
	testutil.NotNil(t, injB)
	testutil.Equal(t, reflect.TypeOf(injB).String(), "*fakes.FakeConsumer")
}

// --- 4.1-CGPATH-003 (AC3, P0) — B2 unmaterialized closed type has no slot -----

// TestCodegen_PathGamma_OpenGeneric_Unmaterialized: on the B2 container, a
// structurally-valid-but-unenumerated closed type has NO generated slot —
// GetService returns nil and IsService returns false. The codegen-side meaning
// of "build-time unresolvable" (AC3): the container literally has no case for it
// (Go has no runtime MakeGenericType, AD-4).
func TestCodegen_PathGamma_OpenGeneric_Unmaterialized(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	unmaterialized := reflect.TypeFor[fakes.IFakeOpenGenericService[fakes.IFakeScopedService]]()
	testutil.Nil(t, root.GetService(unmaterialized))
	testutil.True(t, !root.IsService(unmaterialized),
		"unenumerated closed generic must have no slot in the B2 container (CAP-7, AD-4)")
}

// --- 4.1-CGPATH-004 (AC6, P0) — regen idempotent (deterministic output) -------

// TestCodegen_PathGamma_OpenGeneric_RegenIdempotent: generating from the same
// registry twice must produce byte-identical output (AD-14 deterministic emit).
// open-generic materialization (closed type names with sanitized type-arg
// segments) must not introduce nondeterminism (e.g. map iteration order) into
// the generated container. conformanceRegistry() lives in gen_test.go (same
// package) and is updated by Task 4 to include the injected dep + second
// instantiation; this guard pins idempotency regardless of that content.
func TestCodegen_PathGamma_OpenGeneric_RegenIdempotent(t *testing.T) {
	reg := conformanceRegistry()
	first, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	second, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	testutil.Equal(t, string(first), string(second))
}
