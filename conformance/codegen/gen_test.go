// Conformance tests for the codegen container: regeneration-comparison guard
// (AD-14 deterministic output), behavior smoke tests, and dependency-direction
// self-check (AD-2).
//
// Story 2.1 grows the registration to a dependency graph — FakeService plus a
// FakeConsumer that depends on it — produced by the real cmd/godi-gen CLI behind
// //go:generate (registration.go). NewRoot now returns (*Root, error) (D-1) and
// the regeneration guard pins the real generate output byte-for-byte.
package codegen_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// conformanceRegistry mirrors conformance/codegen/registration.go, the source
// the //go:generate CLI analyzes: singletons FakeService/FakeConsumer, scoped
// FakeScopedService, and transients FakeTransientService/FakeTransientWithScopedDep.
func conformanceRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"github.com/gonewx/godi/conformance/fakes"}
	reg.Service(plan.Service{
		Name:     "FakeService",
		GoType:   "fakes.IFakeService",
		Ctor:     "fakes.NewFakeService",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "FakeConsumer",
		GoType:   "fakes.IFakeConsumer",
		Ctor:     "fakes.NewFakeConsumer",
		Lifetime: plan.Singleton,
		Deps:     []string{"FakeService"},
	})
	reg.Service(plan.Service{
		Name:     "FakeScopedService",
		GoType:   "fakes.IFakeScopedService",
		Ctor:     "fakes.NewFakeScopedService",
		Lifetime: plan.Scoped,
	})
	reg.Service(plan.Service{
		Name:     "FakeTransientService",
		GoType:   "fakes.IFakeTransientService",
		Ctor:     "fakes.NewFakeTransientService",
		Lifetime: plan.Transient,
	})
	reg.Service(plan.Service{
		Name:     "FakeTransientWithScopedDep",
		GoType:   "fakes.IFakeTransientWithScopedDep",
		Ctor:     "fakes.NewFakeTransientWithScopedDep",
		Lifetime: plan.Transient,
		Deps:     []string{"FakeScopedService"},
	})
	reg.Service(plan.Service{
		Name:       "FakeDisposableSingleton",
		GoType:     "fakes.IFakeDisposableSingleton",
		Ctor:       "fakes.NewFakeDisposableSingleton",
		Lifetime:   plan.Singleton,
		Disposable: true,
	})
	reg.Service(plan.Service{
		Name:       "FakeDisposableScoped",
		GoType:     "fakes.IFakeDisposableScoped",
		Ctor:       "fakes.NewFakeDisposableScoped",
		Lifetime:   plan.Scoped,
		Disposable: true,
	})
	reg.Service(plan.Service{
		Name:       "FakeDisposableTransient",
		GoType:     "fakes.IFakeDisposableTransient",
		Ctor:       "fakes.NewFakeDisposableTransient",
		Lifetime:   plan.Transient,
		Disposable: true,
	})
	reg.Service(plan.Service{
		Name:     "FakeOneMultipleService",
		GoType:   "fakes.IFakeMultipleService",
		Ctor:     "fakes.NewFakeOneMultipleService",
		Lifetime: plan.Transient,
	})
	reg.Service(plan.Service{
		Name:     "FakeTwoMultipleService",
		GoType:   "fakes.IFakeMultipleService",
		Ctor:     "fakes.NewFakeTwoMultipleService",
		Lifetime: plan.Transient,
	})
	reg.Service(plan.Service{
		Name:     "FakeOuterService",
		GoType:   "fakes.IFakeOuterService",
		Ctor:     "fakes.NewFakeOuterService",
		Lifetime: plan.Transient,
		Deps:     []string{"FakeService", "[]fakes.IFakeMultipleService"},
	})
	reg.Service(plan.Service{
		Name:     "FakeOpenGenericService",
		GoType:   "fakes.IFakeOpenGenericService[fakes.IFakeService]",
		Ctor:     "fakes.NewFakeOpenGenericService[fakes.IFakeService]",
		Lifetime: plan.Transient,
	})
	// Keyed registrations (Story 3-1)
	reg.Service(plan.Service{
		Name:     "KeyedServiceOne#\"service1\"",
		GoType:   "fakes.IKeyedService",
		Ctor:     "fakes.NewKeyedServiceOne",
		Lifetime: plan.Singleton,
		Key:      "\"service1\"",
	})
	reg.Service(plan.Service{
		Name:     "KeyedServiceTwo#\"service2\"",
		GoType:   "fakes.IKeyedService",
		Ctor:     "fakes.NewKeyedServiceTwo",
		Lifetime: plan.Singleton,
		Key:      "\"service2\"",
	})
	reg.Service(plan.Service{
		Name:     "KeyedServiceOne#\"both\"",
		GoType:   "fakes.IKeyedService",
		Ctor:     "fakes.NewKeyedServiceOne",
		Lifetime: plan.Singleton,
		Key:      "\"both\"",
	})
	reg.Service(plan.Service{
		Name:     "KeyedServiceTwo#\"both\"",
		GoType:   "fakes.IKeyedService",
		Ctor:     "fakes.NewKeyedServiceTwo",
		Lifetime: plan.Singleton,
		Key:      "\"both\"",
	})
	reg.Service(plan.Service{
		Name:     "KeyedServiceOne#\"singleton-factory\"",
		GoType:   "fakes.IKeyedService",
		Ctor:     "fakes.NewKeyedServiceOne",
		Lifetime: plan.Singleton,
		Key:      "\"singleton-factory\"",
	})
	reg.Service(plan.Service{
		Name:     "KeyedServiceOne#\"transient\"",
		GoType:   "fakes.IKeyedService",
		Ctor:     "fakes.NewKeyedServiceOne",
		Lifetime: plan.Transient,
		Key:      "\"transient\"",
	})
	reg.Service(plan.Service{
		Name:        "IKeyedService#\"transient-factory\"",
		GoType:      "fakes.IKeyedService",
		FactoryExpr: "func(p godi.Provider) fakes.IKeyedService {\n\treturn fakes.NewKeyedServiceOne()\n}",
		Lifetime:    plan.Transient,
		Key:         "\"transient-factory\"",
	})
	reg.Service(plan.Service{
		Name:     "KeyedServiceOne#\"scoped\"",
		GoType:   "fakes.IKeyedService",
		Ctor:     "fakes.NewKeyedServiceOne",
		Lifetime: plan.Scoped,
		Key:      "\"scoped\"",
	})
	// Story 3-2: Keyed injection registrations (AC2-AC5)
	reg.Service(plan.Service{
		Name:        "IKeyedServiceWithDep#\"with-dep\"",
		GoType:      "fakes.IKeyedServiceWithDep",
		FactoryExpr: "func(p godi.Provider) fakes.IKeyedServiceWithDep {\n\tdep := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), \"service1\")\n\tif dep == nil {\n\t\treturn fakes.NewKeyedServiceWithDep(nil)\n\t}\n\treturn fakes.NewKeyedServiceWithDep(dep.(fakes.IKeyedService))\n}",
		Lifetime:    plan.Singleton,
		Key:         "\"with-dep\"",
	})
	reg.Service(plan.Service{
		Name:        "IKeyedServiceWithKey#\"injected-key\"",
		GoType:      "fakes.IKeyedServiceWithKey",
		FactoryExpr: "func(p godi.Provider) fakes.IKeyedServiceWithKey {\n\treturn fakes.NewKeyedServiceWithKey(\"injected-key\")\n}",
		Lifetime:    plan.Singleton,
		Key:         "\"injected-key\"",
	})
	reg.Service(plan.Service{
		Name:        "IKeyedServiceWithDep#\"missing-dep\"",
		GoType:      "fakes.IKeyedServiceWithDep",
		FactoryExpr: "func(p godi.Provider) fakes.IKeyedServiceWithDep {\n\tdep := p.GetKeyedService(reflect.TypeFor[fakes.IKeyedService](), \"nonexistent\")\n\tif dep == nil {\n\t\treturn fakes.NewKeyedServiceWithDep(nil)\n\t}\n\treturn fakes.NewKeyedServiceWithDep(dep.(fakes.IKeyedService))\n}",
		Lifetime:    plan.Singleton,
		Key:         "\"missing-dep\"",
	})
	reg.Service(plan.Service{
		Name:        "IKeyedServiceWithProvider#\"with-provider\"",
		GoType:      "fakes.IKeyedServiceWithProvider",
		FactoryExpr: "func(p godi.Provider) fakes.IKeyedServiceWithProvider {\n\treturn fakes.NewKeyedServiceWithProvider(p)\n}",
		Lifetime:    plan.Singleton,
		Key:         "\"with-provider\"",
	})
	reg.Service(plan.Service{
		Name:        "IKeyedServiceWithProvider#\"scoped-with-provider\"",
		GoType:      "fakes.IKeyedServiceWithProvider",
		FactoryExpr: "func(p godi.Provider) fakes.IKeyedServiceWithProvider {\n\treturn fakes.NewKeyedServiceWithProvider(p)\n}",
		Lifetime:    plan.Scoped,
		Key:         "\"scoped-with-provider\"",
	})
	// Story 3-3: AnyKey / null-key boundary registrations (AD-17). The unkeyed
	// AnyKeyServiceA is the AnyKey-query exclusion baseline + nil-resolution target;
	// the interleaved keys alpha→A, beta→B, alpha→A (the second alpha is form-B so
	// its name differs from the form-A one) prove AnyKey-query registration order.
	reg.Service(plan.Service{
		Name:     "AnyKeyServiceA",
		GoType:   "fakes.IAnyKeyService",
		Ctor:     "fakes.NewAnyKeyServiceA",
		Lifetime: plan.Singleton,
	})
	reg.Service(plan.Service{
		Name:     "AnyKeyServiceA#\"alpha\"",
		GoType:   "fakes.IAnyKeyService",
		Ctor:     "fakes.NewAnyKeyServiceA",
		Lifetime: plan.Singleton,
		Key:      "\"alpha\"",
	})
	reg.Service(plan.Service{
		Name:     "AnyKeyServiceB#\"beta\"",
		GoType:   "fakes.IAnyKeyService",
		Ctor:     "fakes.NewAnyKeyServiceB",
		Lifetime: plan.Singleton,
		Key:      "\"beta\"",
	})
	reg.Service(plan.Service{
		Name:        "IAnyKeyService#\"alpha\"",
		GoType:      "fakes.IAnyKeyService",
		FactoryExpr: "func(p godi.Provider) fakes.IAnyKeyService {\n\treturn fakes.NewAnyKeyServiceA()\n}",
		Lifetime:    plan.Singleton,
		Key:         "\"alpha\"",
	})
	return reg
}

// --- 2.1-CG-001 (AC3, P0) ----------------------------------------------------

// TestCodegen_RegenComparison: the committed container_gen.go must be
// byte-identical to a fresh genlib.Generate run for the dependency registration —
// drift means a hand-fork (AD-14). This guards that the real //go:generate output
// stays deterministic.
func TestCodegen_RegenComparison(t *testing.T) {
	reg := conformanceRegistry()
	freshSrc, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)

	_, thisFile, _, _ := goruntime.Caller(0)
	containerPath := filepath.Join(filepath.Dir(thisFile), "container_gen.go")
	committed, readErr := os.ReadFile(containerPath)
	testutil.NoError(t, readErr)

	testutil.Equal(t, string(freshSrc), string(committed))
}

// --- 2.1-CG-002 (AC3, P0) ----------------------------------------------------

// TestCodegen_NewRootResolvesServices: the regenerated container adopts
// NewRoot() (*Root, error) (D-1) and resolves both the leaf FakeService and the
// dependency-injected FakeConsumer to their non-nil concrete types (AD-1/AD-16).
func TestCodegen_NewRootResolvesServices(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	svc := root.GetService(reflect.TypeFor[fakes.IFakeService]())
	testutil.NotNil(t, svc)
	testutil.Equal(t, reflect.TypeOf(svc).String(), "*fakes.FakeService")

	cons := root.GetService(reflect.TypeFor[fakes.IFakeConsumer]())
	testutil.NotNil(t, cons)
	testutil.Equal(t, reflect.TypeOf(cons).String(), "*fakes.FakeConsumer")
}

// TestCodegen_IsService_Consistency: IsService must agree with GetService
// (AD-16 self-consistency).
func TestCodegen_IsService_Consistency(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	testutil.True(t, root.IsService(reflect.TypeFor[fakes.IFakeService]()),
		"registered type must report IsService=true")
	testutil.True(t, root.IsService(reflect.TypeFor[fakes.IFakeConsumer]()),
		"registered dependency consumer must report IsService=true")
	testutil.True(t, !root.IsService(reflect.TypeFor[int]()),
		"unregistered type must report IsService=false")
}

// TestCodegen_Close_Idempotent: Close is idempotent (AD-16).
func TestCodegen_Close_Idempotent(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	testutil.NoError(t, root.Close())
	testutil.NoError(t, root.Close())
}

// TestCodegen_SingletonSharedNotReconstructed: the dependency injected into
// FakeConsumer is the SAME singleton the container resolves directly — exactly
// one constructor on the assembly path (AD-8). Pointer identity within one
// provider is legal (R4 forbids it only across providers).
func TestCodegen_SingletonSharedNotReconstructed(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	var direct any = root.GetService(reflect.TypeFor[fakes.IFakeService]())
	var injected any = root.GetService(reflect.TypeFor[fakes.IFakeConsumer]()).(fakes.IFakeConsumer).Dependency()
	testutil.NotNil(t, direct)
	testutil.Equal(t, direct, injected)
}

// --- 2.1-CG-003 (AC3, P1) ----------------------------------------------------

// TestCodegen_DependencyDirection: the generated container must NOT import plan,
// genlib, or cmd (AD-2 dependency direction). Runtime IS allowed — generated
// containers depend on runtime for the Disposer (AD-2: generated --> runtime).
func TestCodegen_DependencyDirection(t *testing.T) {
	deps := goListDeps(t, "github.com/gonewx/godi/conformance/codegen")
	for _, dep := range deps {
		testutil.True(t, !strings.Contains(dep, "genlib"),
			"codegen container must not import genlib (AD-2)")
		testutil.True(t, !strings.Contains(dep, "godi/plan"),
			"codegen container must not import plan (AD-2)")
		testutil.True(t, !strings.Contains(dep, "godi-gen"),
			"codegen container must not import cmd/godi-gen (AD-2)")
	}
}

func goListDeps(t *testing.T, pkg string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-deps", pkg)
	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GOWORK=") {
			env = append(env, e)
		}
	}
	env = append(env, "GOWORK=off")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps %s: %v", pkg, err)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}
