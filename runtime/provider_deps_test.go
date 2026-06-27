// Story 2.1 (AC1) — runtime provider: dependency wiring + Build error signature
// + registration-order independence (Task 4).
//
// Epic 1's Build() eagerly materialized singletons by iterating the factory map
// in random order — when a dependent was built before its dependency, the
// factory's p.GetService(dep) returned nil (deferred 1.3, fixed for this story).
// D-3 fixes this with lazy-memoize: GetService builds on first access and caches,
// so a dependent's factory recursively materializes its dependency regardless of
// map order; Build then forces all of them (eager completion, NFR8). D-1 evolves
// Build() to (godi.Provider, error) to match the codegen NewRoot signature.
//
// API CONTRACT (D-1/D-3):
//
//	func (b *runtime.Builder) Build() (godi.Provider, error)
//	  - dependencies resolved via p.GetService inside factories, order-independent.
package runtime_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// Local two-type dependency graph: consumerRT depends on iDepRT. These are
// self-contained (no conformance/fakes dependency) so this slice is independent.
type iDepRT interface{ depRT() }

type depRT struct{}

func (depRT) depRT() {}

type iConsumerRT interface{ dependency() iDepRT }

type consumerRT struct{ dep iDepRT }

func (c consumerRT) dependency() iDepRT { return c.dep }

// TestRTDeps_BuildReturnsError_AndWiresDependency covers 2.1-RT-001 / 2.1-RT-002 (AC1).
//
// Build adopts the (godi.Provider, error) signature (D-1) and the consumer's
// factory resolves its dependency through the provider — the injected dependency
// must be non-nil.
func TestRTDeps_BuildReturnsError_AndWiresDependency(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[iDepRT](b, func(godi.Provider) iDepRT { return depRT{} })
	runtime.AddSingleton[iConsumerRT](b, func(p godi.Provider) iConsumerRT {
		dep := p.GetService(reflect.TypeFor[iDepRT]()).(iDepRT)
		return consumerRT{dep: dep}
	})

	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	got := p.GetService(reflect.TypeFor[iConsumerRT]())
	testutil.NotNil(t, got)
	testutil.NotNil(t, got.(iConsumerRT).dependency())
}

// TestRTDeps_RegistrationOrderIndependent covers 2.1-RT-003 (AC1).
//
// Registering the dependent (consumer) BEFORE its dependency (dep) must still
// wire correctly — the lazy-memoize materialization (D-3) makes the random map
// iteration order harmless. Before the fix, this is exactly the nil-dependency
// crash deferred from 1.3.
func TestRTDeps_RegistrationOrderIndependent(t *testing.T) {
	b := runtime.NewBuilder()
	// Dependent first, dependency second — the order that broke eager Build.
	runtime.AddSingleton[iConsumerRT](b, func(p godi.Provider) iConsumerRT {
		dep := p.GetService(reflect.TypeFor[iDepRT]()).(iDepRT)
		return consumerRT{dep: dep}
	})
	runtime.AddSingleton[iDepRT](b, func(godi.Provider) iDepRT { return depRT{} })

	p, err := b.Build()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	c := p.GetService(reflect.TypeFor[iConsumerRT]()).(iConsumerRT)
	testutil.True(t, c.dependency() != nil, "dependency must be materialized regardless of registration order (D-3)")
}
