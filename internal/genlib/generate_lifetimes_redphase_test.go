// ATDD red-phase scaffold for Story 2.2 (AC1/AC2) — genlib emit: three-layer
// structure (Root fields + Scope fields + New{T}() methods).
//
// Story 2.1's emit only handles singletons as Root fields. This story evolves
// the emit shape for the three lifetime tiers (AD-1):
//   - Singleton = Root field (unchanged, NewRoot eager)
//   - Scoped = Root field (root-level, D-2) + scope field (scope-level)
//   - Transient = New{T}() method on Root and scope (every call new, D-4)
//
// GetService/IsService dispatch must route by lifetime (D-5).
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-2-2-scoped-transient-lifetimes.md.
package genlib_test

import (
	"go/format"
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// lifetimesRegistry returns a three-lifetime graph: a singleton logger, a
// scoped request (depends on logger), and a transient widget (depends on
// request). This is the minimum-viable registration for three-layer emit.
func lifetimesRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:     "Logger",
		GoType:   "*svc.Logger",
		Ctor:     "svc.NewLogger",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "Request",
		GoType:   "svc.IRequest",
		Ctor:     "svc.NewRequest",
		Lifetime: plan.Scoped,
		Deps:     []string{"Logger"},
	})
	reg.Service(plan.Service{
		Name:     "Widget",
		GoType:   "svc.IWidget",
		Ctor:     "svc.NewWidget",
		Lifetime: plan.Transient,
		Deps:     []string{"Request"},
	})
	return reg
}

// --- 2.2-EMIT-001 (AC1, P0) --------------------------------------------------

// TestEmitLifetimes_RootStructHasScopedFields: the Root struct must contain
// BOTH singleton fields (f_Logger) and root-level scoped fields (f_Request)
// because root acts as the root scope (D-2).
func TestEmitLifetimes_RootStructHasScopedFields(t *testing.T) {
	src, err := genlib.Generate(lifetimesRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, "f_Logger"),
		"Root struct must contain singleton field f_Logger")
	testutil.True(t, strings.Contains(code, "f_Request"),
		"Root struct must contain root-level scoped field f_Request (D-2: root is root scope)")
}

// --- 2.2-EMIT-002 (AC1, P0) --------------------------------------------------

// TestEmitLifetimes_ScopeStructHasScopedFields: the scope struct must have its
// own scoped fields (f_Request) separate from Root — each scope holds its own
// scoped instances (D-2).
func TestEmitLifetimes_ScopeStructHasScopedFields(t *testing.T) {
	src, err := genlib.Generate(lifetimesRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, "type scope struct"),
		"scope struct must be emitted")
	// The scope struct must have its own scoped fields, not just a root pointer.
	scopeIdx := strings.Index(code, "type scope struct")
	testutil.True(t, scopeIdx >= 0, "scope struct must exist")
	scopeBlock := code[scopeIdx:]
	testutil.True(t, strings.Contains(scopeBlock, "root"),
		"scope must reference root (for singleton access)")
	testutil.True(t, strings.Contains(scopeBlock, "f_Request"),
		"scope must have its own scoped field f_Request")
}

// --- 2.2-EMIT-003 (AC1, P0) --------------------------------------------------

// TestEmitLifetimes_ScopeHasNewMethodForTransient: scope must emit a
// New{Type}() method for transient services — each call creates a new instance
// (D-4: transient = New*() method, AD-14: no caching).
func TestEmitLifetimes_ScopeHasNewMethodForTransient(t *testing.T) {
	src, err := genlib.Generate(lifetimesRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, "func (s *scope) NewWidget()"),
		"scope must emit NewWidget() method for transient (D-4)")
}

// --- 2.2-EMIT-004 (AC1, P0) --------------------------------------------------

// TestEmitLifetimes_RootHasNewMethodForTransient: Root must also emit a
// New{Type}() method — transients can be resolved from root (D-4).
func TestEmitLifetimes_RootHasNewMethodForTransient(t *testing.T) {
	src, err := genlib.Generate(lifetimesRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Contains(code, "func (r *Root) NewWidget()"),
		"Root must emit NewWidget() method for transient (D-4)")
}

// --- 2.2-EMIT-005 (AC1, P0) --------------------------------------------------

// TestEmitLifetimes_GetServiceDispatchesByLifetime: GetService on Root and
// scope must route by lifetime (D-5): singleton = root field, scoped = layer
// field, transient = New{T}().
func TestEmitLifetimes_GetServiceDispatchesByLifetime(t *testing.T) {
	src, err := genlib.Generate(lifetimesRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	// Root GetService must handle all three types
	testutil.True(t, strings.Contains(code, "func (r *Root) GetService"),
		"Root must have GetService")
	// Scope GetService must also dispatch (not just delegate everything to root)
	testutil.True(t, strings.Contains(code, "func (s *scope) GetService"),
		"scope must have its own GetService (not pure delegation)")
}

// --- 2.2-EMIT-006 (AC1/AC2, P0) ----------------------------------------------

// TestEmitLifetimes_MultiLifetimeGolden: the three-lifetime container output
// must be byte-stable across runs — anchored by a golden file. This is the
// canonical multi-lifetime emit shape (Root/scope fields + New{T}() + D-5
// dispatch).
func TestEmitLifetimes_MultiLifetimeGolden(t *testing.T) {
	src, err := genlib.Generate(lifetimesRegistry(), "codegen")
	testutil.NoError(t, err)
	testutil.Golden(t, "lifetimes-container", src)
}

// --- 2.2-EMIT-007 (AC1, P1) --------------------------------------------------

// TestEmitLifetimes_OutputIsFormatted: the three-lifetime emit must produce
// gofmt-legal code (format.Source succeeds) and keep the header (regression).
func TestEmitLifetimes_OutputIsFormatted(t *testing.T) {
	src, err := genlib.Generate(lifetimesRegistry(), "codegen")
	testutil.NoError(t, err)

	_, fmtErr := format.Source(src)
	testutil.NoError(t, fmtErr)

	first := strings.SplitN(string(src), "\n", 2)[0]
	testutil.True(t, strings.HasPrefix(first, "// Code generated by"),
		"first line must be the '// Code generated by ...; DO NOT EDIT.' header")
}

// --- 2.2-EMIT-008 (AC1, P1) --------------------------------------------------

// TestEmitLifetimes_ExistingGoldensUnchanged: the existing single-service and
// deps-container goldens must NOT change — regression guard that the new
// three-layer emit does not break singleton-only containers.
func TestEmitLifetimes_ExistingGoldensUnchanged(t *testing.T) {
	// Single-service golden (Story 1.4)
	reg1 := singleServiceRegistry()
	src1, err := genlib.Generate(reg1, "container")
	testutil.NoError(t, err)
	testutil.Golden(t, "single-service", src1)

	// Deps golden (Story 2.1)
	reg2 := depsRegistry()
	src2, err := genlib.Generate(reg2, "codegen")
	testutil.NoError(t, err)
	testutil.Golden(t, "deps-container", src2)
}

// --- 2.2-EMIT-009 (AC2, P0) --------------------------------------------------

// TestEmitLifetimes_TransientDepsAccessScopeFields: a transient's New{T}()
// on scope must resolve its scoped dependency through the scope's own accessor
// (s.get_Request(), which builds it on first use and caches it in s.f_Request),
// not from root — this is the codegen side of AC2 (transient depends on scoped,
// same scope reuses scoped).
func TestEmitLifetimes_TransientDepsAccessScopeFields(t *testing.T) {
	src, err := genlib.Generate(lifetimesRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	body := methodBody(t, code, "func (s *scope) NewWidget()")
	testutil.True(t, strings.Contains(body, "s.get_Request()"),
		"scope's NewWidget() must resolve its scoped dependency through the scope accessor:\n"+body)
}
