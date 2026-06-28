//go:build atdd_redphase

// ATDD scaffold for Story 3.3 (AC1-AC6) — conformance suite: AnyKey sentinel +
// null-key boundary semantics across BOTH providers (runtime oracle + codegen B2)
// with cross-provider differential equivalence (AD-15/AD-16/AD-17).
//
// Ported from dotnet KeyedDependencyInjectionSpecificationTests (v10.0.9) AnyKey/
// null slice: CombinationalRegistration(:18, adapted), ResolveNullKeyedService(:116),
// ResolveNonKeyedService(:134), ResolveKeyedServicesAnyKey(:189),
// ResolveKeyedServicesAnyKeyOrdering(:565), ResolveKeyedServicesAnyKeyConsistency(:254).
//
// The catch-all REGISTRATION family (:638-874) is N/A — a Go mechanism wall (D-3):
// form-B `func(Provider) I` / form-A `New{Impl}` cannot receive a resolve-time key,
// so a service registered under AnyKey to be resolved by ANY concrete key (with the
// key injected) has no Go equivalent. Both providers REJECT AnyKey as a registration
// key instead (AC4). See the story Dev Notes D-3 and deferred-work.md.
//
// A DEDICATED fake IAnyKeyService (fakes_keyed.go) keeps the AnyKey-query set small
// and order-exact across providers; the 3-1/3-2 IKeyedService keys are untouched.
//
// AD-17 truth table pinned here (Go-adapted, the AnyKey-reg row removed, D-3):
//
//	GetService(t) / GetKeyedService(t, nil)  → unkeyed service (null == unkeyed)
//	GetKeyedServices(t, AnyKey)              → all CONCRETE-keyed, registration order
//	GetKeyedService(t, AnyKey)               → ALWAYS panic (even with no registration)
//
// RED PHASE — this scaffold fails before implementation because:
//   - GetKeyedServices(t, godi.AnyKey) is not implemented → returns nil instead of
//     the concrete-keyed set in registration order (runtime + codegen).
//   - GetKeyedService(t, godi.AnyKey) does not panic → returns nil (runtime + codegen).
//   - runtime.AddKeyed*(b, godi.AnyKey, …) does not panic (AnyKey rejection, AC4).
//   - AddKeyed*(b, nil, …) lands in the keyed map, not the unkeyed namespace, so the
//     null-key registration is unreachable via GetService (AC1 registration side).
//   - the codegen container has no IAnyKeyService registrations — registration.go +
//     regen are green-phase work (Task 6) — so its AnyKey assertions fail too.
//
// Activation: delete the //go:build line and the blank line below it to join the
// default build, once Tasks 1-6 are implemented. See
// _bmad-output/test-artifacts/atdd-checklist-3-3-anykey-null-semantics.md.
package conformance

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// anyKeyType is the dedicated AnyKey/null boundary service type under test.
func anyKeyType() reflect.Type { return reflect.TypeFor[fakes.IAnyKeyService]() }

// rtFactoryKeyedAnyKey builds the runtime oracle for the AnyKey/null boundary. The
// registration MIRRORS the codegen source that green-phase Task 6 adds to
// conformance/codegen/registration.go (AD-14, one logical source):
//
//	unkeyed IAnyKeyService → A   // exclusion baseline + null-resolution target (AC1/AC2)
//	keyed "alpha"          → A   // concrete-keyed reg #1 (AC2)
//	keyed "beta"           → B   // concrete-keyed reg #2 — ordering middle (AC2)
//	keyed "alpha"          → A   // concrete-keyed reg #3 — interleaved (AC2 ordering)
//
// No AnyKey registration here: AnyKey-as-key is rejected (AC4) and would panic the
// whole Build in green; that rejection is asserted by a standalone test below.
var rtFactoryKeyedAnyKey = ProviderFactory{
	Name: "runtime",
	Build: func(t *testing.T) Provider {
		t.Helper()
		b := runtime.NewBuilder()

		// Unkeyed IAnyKeyService — AnyKey-query must EXCLUDE this; nil key routes here.
		runtime.AddSingleton[fakes.IAnyKeyService](b, func(godi.Provider) fakes.IAnyKeyService {
			return fakes.NewAnyKeyServiceA()
		})

		// Concrete keyed, interleaved keys for registration-order assertion (AC2).
		runtime.AddKeyedSingleton[fakes.IAnyKeyService](b, "alpha", func(godi.Provider) fakes.IAnyKeyService {
			return fakes.NewAnyKeyServiceA()
		})
		runtime.AddKeyedSingleton[fakes.IAnyKeyService](b, "beta", func(godi.Provider) fakes.IAnyKeyService {
			return fakes.NewAnyKeyServiceB()
		})
		runtime.AddKeyedSingleton[fakes.IAnyKeyService](b, "alpha", func(godi.Provider) fakes.IAnyKeyService {
			return fakes.NewAnyKeyServiceA()
		})

		p, err := b.Build()
		testutil.NoError(t, err)
		return p
	},
}

// cgFactoryKeyedAnyKey builds the codegen provider. In the red phase the
// regenerated container does NOT yet include IAnyKeyService (Task 6), nor does
// genlib emit the AnyKey-query / AnyKey-panic branches (Task 4), so every AnyKey
// assertion fails. It mirrors rtFactoryKeyedAnyKey's registration only after green.
var cgFactoryKeyedAnyKey = ProviderFactory{
	Name: "codegen",
	Build: func(t *testing.T) Provider {
		t.Helper()
		root, err := codegen.NewRoot()
		testutil.NoError(t, err)
		return root
	},
}

// anyKeySpecCases run against BOTH providers via RunSuite. The null-REGISTRATION
// (SPEC-002) and AnyKey-REGISTRATION-rejection (AC4) cases use their own builders
// and live as standalone tests below — they reshape the registration in ways
// incompatible with the shared factory.
var anyKeySpecCases = []SpecCase{
	// --- 3.3-SPEC-001 (AC1, P0) — nil key resolves as unkeyed -------------------
	//
	// ResolveNonKeyedService: a nil key routes to the unkeyed service, so
	// GetKeyedService(t, nil) == GetService(t) (AD-17). Resolution-side nil routing
	// is already implemented (story D-1), so this is GREEN on runtime in the red
	// phase and RED on codegen (no unkeyed IAnyKeyService registered yet).
	{
		Name: "anykey: nil key routes to unkeyed service (ResolveNonKeyedService, AC1)",
		Run: func(t *testing.T, p Provider) {
			st := anyKeyType()
			viaNil := p.GetKeyedService(st, nil)
			viaUnkeyed := p.GetService(st)
			testutil.NotNil(t, viaNil)
			testutil.NotNil(t, viaUnkeyed)
			testutil.Equal(t, dynTypeName(viaNil), "*fakes.AnyKeyServiceA")
			testutil.Equal(t, viaNil, viaUnkeyed) // same singleton: nil == unkeyed (within one provider, R4)
		},
	},
	// --- 3.3-SPEC-003 (AC2, P0) — GetKeyedServices(AnyKey) = all concrete-keyed -
	//
	// ResolveKeyedServicesAnyKey: returns every service under a non-nil CONCRETE key,
	// EXCLUDING unkeyed/null. With the registration above that is the three keyed
	// entries [A(alpha), B(beta), A(alpha)]; the unkeyed A is excluded.
	{
		Name: "anykey: GetKeyedServices(AnyKey) returns all concrete-keyed, excludes unkeyed (ResolveKeyedServicesAnyKey, AC2)",
		Run: func(t *testing.T, p Provider) {
			all := p.GetKeyedServices(anyKeyType(), godi.AnyKey)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 3) // 3 concrete-keyed; unkeyed excluded
		},
	},
	// --- 3.3-SPEC-004 (AC2, P0) — AnyKey order = registration order, not grouped -
	//
	// ResolveKeyedServicesAnyKeyOrdering: interleaved keys alpha,beta,alpha must come
	// back in REGISTRATION order [A, B, A], NOT grouped-by-key [A, A, B]. The type
	// sequence distinguishes the two: beta→B sits BETWEEN the two alpha→A registrations.
	{
		Name: "anykey: GetKeyedServices(AnyKey) preserves registration order, not key grouping (ResolveKeyedServicesAnyKeyOrdering, AC2)",
		Run: func(t *testing.T, p Provider) {
			all := p.GetKeyedServices(anyKeyType(), godi.AnyKey)
			testutil.NotNil(t, all)
			testutil.Equal(t, len(all), 3)
			if len(all) != 3 {
				return // len assertion above already failed (red phase); avoid an index panic
			}
			testutil.Equal(t, dynTypeName(all[0]), "*fakes.AnyKeyServiceA")
			testutil.Equal(t, dynTypeName(all[1]), "*fakes.AnyKeyServiceB") // grouped-by-key would put B last
			testutil.Equal(t, dynTypeName(all[2]), "*fakes.AnyKeyServiceA")
		},
	},
	// --- 3.3-SPEC-005 (AC3, P0) — GetKeyedService(AnyKey) ALWAYS panics, root ---
	//
	// ResolveKeyedServicesAnyKeyConsistency's Assert.Throws<InvalidOperationException>:
	// AnyKey is query-only, never a single-value key. The interception happens after
	// the nil check and BEFORE the keyed-map lookup, so it panics even though concrete
	// keys exist. Mirrors the GetRequired panic style (Story 1.2).
	{
		Name: "anykey: GetKeyedService(AnyKey) always panics, root (ResolveKeyedServicesAnyKeyConsistency, AC3)",
		Run: func(t *testing.T, p Provider) {
			testutil.Panics(t, func() {
				_ = p.GetKeyedService(anyKeyType(), godi.AnyKey)
			})
		},
	},
	// --- 3.3-SPEC-005b (AC3, P0) — AnyKey single-resolve panics from a scope too -
	{
		Name: "anykey: GetKeyedService(AnyKey) always panics, scope (AC3, root+scope parity)",
		Run: func(t *testing.T, p Provider) {
			sc := p.CreateScope()
			t.Cleanup(func() { _ = sc.Close() })
			testutil.Panics(t, func() {
				_ = sc.GetKeyedService(anyKeyType(), godi.AnyKey)
			})
		},
	},
	// --- 3.3-SPEC-006 (AC5, P0) — CombinationalRegistration adapted truth table -
	//
	// One case walks the AD-17 truth table (Go-adapted, AnyKey-reg row removed, D-3):
	//   GetService / GetKeyedService(nil) → unkeyed A (non-nil)
	//   GetKeyedService("alpha")          → A (concrete key still resolves)
	//   GetKeyedServices(AnyKey)          → 3 concrete-keyed
	//   GetKeyedService(AnyKey)           → panic
	{
		Name: "anykey: combinational truth table holds (CombinationalRegistration adapted, AC5)",
		Run: func(t *testing.T, p Provider) {
			st := anyKeyType()
			// unkeyed / nil-key → unkeyed instance
			testutil.NotNil(t, p.GetService(st))
			testutil.Equal(t, dynTypeName(p.GetKeyedService(st, nil)), "*fakes.AnyKeyServiceA")
			// concrete key still resolves
			testutil.Equal(t, dynTypeName(p.GetKeyedService(st, "alpha")), "*fakes.AnyKeyServiceA")
			// AnyKey-query → all concrete-keyed
			testutil.Equal(t, len(p.GetKeyedServices(st, godi.AnyKey)), 3)
			// AnyKey single-resolve → panic
			testutil.Panics(t, func() { _ = p.GetKeyedService(st, godi.AnyKey) })
		},
	},
}

// --- 3.3-SPEC-RUN (AC1-AC5, P0) ----------------------------------------------

func TestConformanceAnyKey_RuntimeProvider(t *testing.T) {
	RunSuite(t, rtFactoryKeyedAnyKey, anyKeySpecCases)
}

func TestConformanceAnyKey_CodegenProvider(t *testing.T) {
	RunSuite(t, cgFactoryKeyedAnyKey, anyKeySpecCases)
}

// --- 3.3-SPEC-002 (AC1, P0) — nil-key REGISTRATION routes to unkeyed ----------
//
// ResolveNullKeyedService: registering with a nil key is symmetric to resolving
// with one — it lands in the UNKEYED namespace, not the keyed index. Standalone
// (own builder) because routing a registration into the unkeyed namespace would
// change the shared factory's other assertions.
//
// RED: runtime.AddKeyedSingleton(b, nil, …) currently appends to the keyed map
// under a nil key, so the service is unreachable via GetService → GetService
// returns nil. GREEN (Task 1): nil key routes to b.singletonFactories[t] (unkeyed).
func TestConformanceAnyKey_NilKeyRegistrationRoutesToUnkeyed_Runtime(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddKeyedSingleton[fakes.IAnyKeyService](b, nil, func(godi.Provider) fakes.IAnyKeyService {
		return fakes.NewAnyKeyServiceB()
	})
	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	st := anyKeyType()
	// the nil-key registration is reachable as an UNKEYED service.
	testutil.NotNil(t, p.GetService(st))
	testutil.Equal(t, dynTypeName(p.GetService(st)), "*fakes.AnyKeyServiceB")
	// and via the nil key (which itself routes to unkeyed).
	testutil.Equal(t, dynTypeName(p.GetKeyedService(st, nil)), "*fakes.AnyKeyServiceB")
	// it must NOT appear under a concrete key nor in the AnyKey-query set.
	testutil.Nil(t, p.GetKeyedService(st, "anything"))
	testutil.Equal(t, len(p.GetKeyedServices(st, godi.AnyKey)), 0)
}

// --- 3.3-SPEC-AC4-RT (AC4, P0) — AnyKey as a registration KEY is rejected -----
//
// catch-all registration is a Go mechanism wall (D-3): form-B `func(Provider) I`
// cannot receive a resolve-time key. Both providers REJECT AnyKey as a key — the
// runtime rejects at the AddKeyed* call, symmetric to codegen's generate-time
// rejection (asserted in the codegen Path γ scaffold).
//
// RED: AddKeyedSingleton(b, godi.AnyKey, …) silently stores an unreachable keyed
// entry (no panic). GREEN (Task 1): it panics with a message naming AnyKey as a
// query-only sentinel unsupported as a registration key.
func TestConformanceAnyKey_AnyKeyRegistrationRejected_Runtime(t *testing.T) {
	b := runtime.NewBuilder()
	testutil.Panics(t, func() {
		runtime.AddKeyedSingleton[fakes.IAnyKeyService](b, godi.AnyKey, func(godi.Provider) fakes.IAnyKeyService {
			return fakes.NewAnyKeyServiceA()
		})
	})
}

// --- 3.3-DIFF (AC6, P0) — cross-provider differential equivalence -------------
//
// AD-15 differential equivalence for the AnyKey/null axis. Probes use
// provider-stable observables (length, dynamic-type sequence, nil/non-nil,
// panic-or-not) — never pointer identity across providers (R4).
func TestConformanceAnyKey_DifferentialEquivalence(t *testing.T) {
	st := anyKeyType()

	// AC2: AnyKey-query length equivalence.
	AssertEquivalent(t, rtFactoryKeyedAnyKey, cgFactoryKeyedAnyKey, func(p Provider) int {
		return len(p.GetKeyedServices(st, godi.AnyKey))
	})

	// AC2: AnyKey-query registration-order type sequence equivalence.
	AssertEquivalent(t, rtFactoryKeyedAnyKey, cgFactoryKeyedAnyKey, func(p Provider) string {
		return typeSeq(p.GetKeyedServices(st, godi.AnyKey))
	})

	// AC1: nil key routes to unkeyed — dynamic type equivalence.
	AssertEquivalent(t, rtFactoryKeyedAnyKey, cgFactoryKeyedAnyKey, func(p Provider) string {
		return dynTypeName(p.GetKeyedService(st, nil))
	})

	// AC3: GetKeyedService(AnyKey) panics in BOTH providers (unified semantics).
	AssertEquivalent(t, rtFactoryKeyedAnyKey, cgFactoryKeyedAnyKey, func(p Provider) bool {
		return panicsOnAnyKeySingle(p, st)
	})
}

// typeSeq joins the dynamic type names of vs with "," — a provider-stable,
// order-sensitive observable for AnyKey-query registration order (R4: never
// pointer identity across providers).
func typeSeq(vs []any) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = dynTypeName(v)
	}
	return strings.Join(parts, ",")
}

// panicsOnAnyKeySingle reports whether GetKeyedService(t, AnyKey) panics — the
// AD-17 single-resolve guard, probed as a comparable bool for differential
// equivalence (both providers must agree: true).
func panicsOnAnyKeySingle(p Provider, st reflect.Type) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	_ = p.GetKeyedService(st, godi.AnyKey)
	return false
}
