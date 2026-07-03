// godi-gen Task-0 extension (gois story 2-1) — generator unit coverage for
// multi-statement per-service helpers, form-B factory forwarding, factory-dep
// ordering, and import alias collision, against the testdata/optionshelper and
// testdata/aliascollide fixtures.
//
// The optionshelper fixture mirrors the dotnet Options "four-piece"
// AddOptions[T] shape: a CRTP-constrained helper whose body is four unkeyed
// marker statements (two form A + two form B), closed over two configuration
// types. The pre-extension generator recognized only single-statement form-A
// forwarders (deferred-work.md:129 ②) — every test here exercises what that
// deferred entry promised.
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

// TestGenerator_MultiStatementHelper_ExpandsAllMarkers proves a helper body of
// FOUR marker statements (ExprStmts + final return marker) reduces to four
// closed registrations per call — none silently dropped (NFR2).
func TestGenerator_MultiStatementHelper_ExpandsAllMarkers(t *testing.T) {
	src, err := generate("testdata/optionshelper", "codegen")
	testutil.NoError(t, err)

	code := string(src)
	for _, want := range []string{
		// form-A singleton pieces (Options manager + cache), both closures
		"f_UnnamedOptionsManager__optionsfix_MyCfg",
		"f_UnnamedOptionsManager__optionsfix_OtherCfg",
		"f_OptionsCache__optionsfix_MyCfg",
		"f_OptionsCache__optionsfix_OtherCfg",
		// form-B singleton piece (monitor), both closures
		"f_OptionsMonitor__optionsfix_MyCfg",
		"f_OptionsMonitor__optionsfix_OtherCfg",
		// form-B transient piece (factory) emits a New method, both closures
		"func (r *Root) NewOptionsFactory__optionsfix_MyCfg()",
		"func (r *Root) NewOptionsFactory__optionsfix_OtherCfg()",
	} {
		testutil.True(t, strings.Contains(code, want),
			"multi-statement helper must materialize every marker per closure; missing: "+want)
	}
}

// TestGenerator_HelperFormB_RewritesFactoryText proves the form-B factory
// expression is inlined with the helper type parameter CLOSED and — emitting
// into a different package — registration-package identifiers QUALIFIED.
func TestGenerator_HelperFormB_RewritesFactoryText(t *testing.T) {
	src, err := generate("testdata/optionshelper", "codegen")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, strings.Contains(code,
		"return optionsfix.NewOptionsMonitor[*optionsfix.MyCfg]("),
		"factory body's bare constructor must gain the registration-package qualifier and the closed type argument")
	testutil.True(t, strings.Contains(code,
		"godi.GetRequired[optionsfix.OptionsFactory[*optionsfix.MyCfg]](p)"),
		"godi.Get… type arguments inside the factory must close T and qualify the element type")
	testutil.True(t, strings.Contains(code,
		"godi.GetServices[optionsfix.ConfigureOptions[*optionsfix.MyCfg]](p)"),
		"a GetServices over a never-registered element type must still inline (empty at runtime)")
}

// TestGenerator_HelperFormB_SamePackageKeepsBareIdents proves the
// //go:generate-in-place layout (container emitted INTO the registration
// package, the gois di/ case): bare registration-package identifiers must stay
// bare — qualifying them would not compile.
func TestGenerator_HelperFormB_SamePackageKeepsBareIdents(t *testing.T) {
	src, err := generate("testdata/optionshelper", "optionsfix")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, strings.Contains(code,
		"return NewOptionsMonitor[*MyCfg]("),
		"same-package emit must keep the factory's bare constructor identifier bare")
}

// TestGenerator_FormBFactoryDeps_OrderSingletonAssembly proves the factory's
// godi.Get… calls feed SingletonsTopo: the cache is the LAST marker in the
// helper body, yet the monitor's factory GetRequired-s it — assembling the
// monitor first would hand its constructor a nil cache (AD-15). This is the
// ordering half of deferred-work.md "form-B codegen 完整支持依赖型/跨包工厂".
func TestGenerator_FormBFactoryDeps_OrderSingletonAssembly(t *testing.T) {
	src, err := generate("testdata/optionshelper", "codegen")
	testutil.NoError(t, err)

	code := string(src)
	for _, cfg := range []string{"MyCfg", "OtherCfg"} {
		cache := strings.Index(code, "r.f_OptionsCache__optionsfix_"+cfg+" = ")
		monitor := strings.Index(code, "r.f_OptionsMonitor__optionsfix_"+cfg+" = ")
		testutil.True(t, cache >= 0 && monitor >= 0,
			cfg+": both the cache and the monitor must be wired in NewRoot")
		testutil.True(t, cache < monitor,
			cfg+": the cache must be assembled BEFORE the monitor whose factory GetRequired-s it")
	}
}

// TestGenerator_HelperExpansions_StayIndependent proves the two closures do not
// cross-type (MyCfg's factory never mentions OtherCfg and vice versa) and no
// open template leaks ([T] never appears).
func TestGenerator_HelperExpansions_StayIndependent(t *testing.T) {
	src, err := generate("testdata/optionshelper", "codegen")
	testutil.NoError(t, err)

	code := string(src)
	// The container's own generic Get[T] accessor methods legitimately mention
	// [T]; the OPEN TEMPLATE would leak as a fixture type closed over T.
	for _, leak := range []string{
		"OptionsMonitor[T]", "OptionsFactory[T]", "ConfigureOptions[T]",
		"OptionsMonitorCache[T]", "Options[T]", "NewOptionsMonitor[T]",
	} {
		testutil.True(t, !strings.Contains(code, leak),
			"the open template "+leak+" must never appear in the generated container")
	}

	myStart := strings.Index(code, "r.f_OptionsMonitor__optionsfix_MyCfg = ")
	myEnd := strings.Index(code[myStart:], ")(r)")
	testutil.True(t, myStart >= 0 && myEnd >= 0, "MyCfg monitor factory inline must exist")
	myFactory := code[myStart : myStart+myEnd]
	testutil.True(t, !strings.Contains(myFactory, "OtherCfg"),
		"the MyCfg closure's factory must not mention OtherCfg (closures must not cross-type)")
}

// TestGenerator_RewrittenOutput_Compiles is the compile-level guard the
// string-match tests cannot give: the pre-extension form-B inline was known
// NON-compilable (unqualified identifiers, missing imports), and only a real
// type-check proves that gap is closed. Each fixture's cross-package container
// is written into a throwaway package directory INSIDE the module (import
// paths under testdata resolve only from within) and `go vet`-ed, which fully
// type-checks it. Subprocess-based like TestGenerator_NoNewImpl_ExitsNonZero.
func TestGenerator_RewrittenOutput_Compiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess (go vet) compile check in -short mode")
	}
	for _, fixture := range []string{"optionshelper", "aliascollide"} {
		src, err := generate("testdata/"+fixture, "compilecheck")
		testutil.NoError(t, err)

		dir := filepath.Join("testdata", fixture, "zz_compile_check")
		testutil.NoError(t, os.MkdirAll(dir, 0o755))
		t.Cleanup(func() { _ = os.RemoveAll(dir) })
		testutil.NoError(t, os.WriteFile(filepath.Join(dir, "container_gen.go"), src, 0o644))

		cmd := exec.Command("go", "vet", "./"+dir)
		cmd.Env = append(os.Environ(), "GOWORK=off")
		out, vetErr := cmd.CombinedOutput()
		testutil.True(t, vetErr == nil,
			fixture+": the rewritten container must type-check; go vet said:\n"+string(out))
	}
}

// TestGenerator_ImportAliasCollision proves two service packages sharing a
// declared name (the gois options / config/options reality) are imported under
// distinct identifiers, and a factory body's own qualifier is remapped to the
// emitted alias.
func TestGenerator_ImportAliasCollision(t *testing.T) {
	src, err := generate("testdata/aliascollide", "codegen")
	testutil.NoError(t, err)

	code := string(src)
	testutil.True(t, strings.Contains(code,
		`"github.com/gonewx/godi/cmd/godi-gen/testdata/aliascollide/suba"`),
		"the first same-named package keeps a plain import (binds its declared name)")
	testutil.True(t, strings.Contains(code,
		`opts2 "github.com/gonewx/godi/cmd/godi-gen/testdata/aliascollide/subb"`),
		"the second same-named package must be renamed in the import block")
	testutil.True(t, strings.Contains(code, "f_IThingB opts2.IThingB"),
		"types from the renamed package must be qualified by the assigned alias")
	testutil.True(t, strings.Contains(code, "return opts2.NewThingB()"),
		"the factory body's registration-file qualifier (bopts.) must be remapped to the emitted alias")
}
