package codegen

import (
	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/fakes"
)

// container_gen.go is produced from the form-A registration below by the real
// godi-gen static-analysis CLI (Story 2.1). Running `go generate ./...` from the
// module root re-emits it; TestCodegen_RegenComparison guards that the committed
// file stays byte-identical to a fresh run (AD-14: deterministic output, no
// hand-fork). Run under GOWORK=off so the generator analyzes this package in
// isolation from the auth workspace's sibling modules.
//
//go:generate go run github.com/gonewx/godi/cmd/godi-gen -pkg codegen -out container_gen.go .

// register is the static registration source the generator analyzes (AD-7): the
// AddSingleton markers are read from go/types, never executed. It mirrors the
// runtime oracle's registration (spec_test.go's rtFactory) so both providers are
// built from one logical source (AD-14). Form A: FakeService has no dependency;
// FakeConsumer depends on IFakeService, discovered through NewFakeConsumer's
// signature.
func register() {
	var r godi.Registry
	godi.AddSingleton[fakes.IFakeService, fakes.FakeService](&r)
	godi.AddSingleton[fakes.IFakeConsumer, fakes.FakeConsumer](&r)
	godi.AddScoped[fakes.IFakeScopedService, fakes.FakeScopedService](&r)
	godi.AddTransient[fakes.IFakeTransientService, fakes.FakeTransientService](&r)
	godi.AddTransient[fakes.IFakeTransientWithScopedDep, fakes.FakeTransientWithScopedDep](&r)
	godi.AddSingleton[fakes.IFakeDisposableSingleton, fakes.FakeDisposableSingleton](&r)
	godi.AddScoped[fakes.IFakeDisposableScoped, fakes.FakeDisposableScoped](&r)
	godi.AddTransient[fakes.IFakeDisposableTransient, fakes.FakeDisposableTransient](&r)
	godi.AddTransient[fakes.IFakeMultipleService, fakes.FakeOneMultipleService](&r)
	godi.AddTransient[fakes.IFakeMultipleService, fakes.FakeTwoMultipleService](&r)
	godi.AddTransient[fakes.IFakeOuterService, fakes.FakeOuterService](&r)
	godi.AddTransient[fakes.IFakeOpenGenericService[fakes.IFakeService], fakes.FakeOpenGenericService[fakes.IFakeService]](&r)

	// Keyed registrations (Story 3-1, AD-17)
	godi.AddKeyedSingleton[fakes.IKeyedService, fakes.KeyedServiceOne](&r, "service1")
	godi.AddKeyedSingleton[fakes.IKeyedService, fakes.KeyedServiceTwo](&r, "service2")
	godi.AddKeyedSingleton[fakes.IKeyedService, fakes.KeyedServiceOne](&r, "both")
	godi.AddKeyedSingleton[fakes.IKeyedService, fakes.KeyedServiceTwo](&r, "both")
	godi.AddKeyedSingleton[fakes.IKeyedService, fakes.KeyedServiceOne](&r, "singleton-factory")
	godi.AddKeyedTransient[fakes.IKeyedService, fakes.KeyedServiceOne](&r, "transient")
}

// _ keeps register reachable: it exists solely as the generator's static-analysis
// input (read via go/types, never executed — AD-7), so nothing else calls it.
var _ = register
