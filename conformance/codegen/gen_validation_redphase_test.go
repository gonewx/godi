// ATDD red-phase scaffold for Story 5.4 (AC2) — conformance/codegen layer:
// Path γ strict validation tests. Each test verifies that genlib.Generate
// rejects a specific illegal graph category (missing/captive/cycle) by
// returning error + nil output (AD-3: bad graph → zero-byte code).
//
// Path γ = AD-15 保真敏感能力升格为严格跑。校验门禁的 codegen 路径不仅
// 做差分等价，还直接在预生成路径断言 Generate 的拒绝行为。
//
// Activation: delete the //go:build line and blank line to join the default
// build.
package codegen_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// --- 5.4-PG-01 (AC2, P0) — Path γ: missing dependency ----------------------

func TestCodegen_Validation_PathGamma_MissingDep(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "Foo", GoType: "*svc.Foo", Ctor: "svc.NewFoo", Lifetime: plan.Singleton,
		Deps: []string{"Bar_Missing"},
	})

	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)
	testutil.True(t, len(src) == 0,
		"missing dep graph must produce zero-byte code (AD-3)")
	testutil.True(t, strings.Contains(err.Error(), "validation failed"),
		"Generate error must indicate validation failure")
}

// --- 5.4-PG-02 (AC2, P0) — Path γ: captive dependency ----------------------

func TestCodegen_Validation_PathGamma_CaptiveDep(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "ScopedSvc", GoType: "*svc.ScopedSvc", Ctor: "svc.NewScopedSvc", Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name: "SingletonSvc", GoType: "*svc.SingletonSvc", Ctor: "svc.NewSingletonSvc", Lifetime: plan.Singleton,
		Deps: []string{"ScopedSvc"},
	})

	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)
	testutil.True(t, len(src) == 0,
		"captive dep graph must produce zero-byte code (AD-3)")
	testutil.True(t, strings.Contains(err.Error(), "captive dependency"),
		"Generate error must mention captive dependency")
}

// --- 5.4-PG-03 (AC2, P0) — Path γ: circular dependency ---------------------

func TestCodegen_Validation_PathGamma_CycleDep(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name: "A", GoType: "*svc.A", Ctor: "svc.NewA", Lifetime: plan.Singleton,
		Deps: []string{"B"},
	})
	reg.Service(plan.Service{
		Name: "B", GoType: "*svc.B", Ctor: "svc.NewB", Lifetime: plan.Singleton,
		Deps: []string{"A"},
	})

	src, err := genlib.Generate(reg, "container")
	testutil.Error(t, err)
	testutil.Nil(t, src)
	testutil.True(t, len(src) == 0,
		"cyclic graph must produce zero-byte code (AD-3)")
	testutil.True(t, strings.Contains(err.Error(), "circular dependency"),
		"Generate error must mention circular dependency")
}
