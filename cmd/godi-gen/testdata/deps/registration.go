// Package depsfix is the form-A WITH-DEPENDENCY fixture the Story 2.1 generator
// loads via go/packages. It is test INPUT, not part of the module build: it
// lives under testdata/, so `go build ./...` / `go test ./...` never compile it.
//
// The generator reads the two AddSingleton[I, Impl] markers statically
// (go/types), discovers each implementation's New{Impl} constructor, reads
// NewConsumer's dependency parameter (IFakeService) and maps it back to the
// FakeService registration, detects NewConsumer's trailing error, and emits a
// dependency-wired container — all without executing this code (AD-7).
package depsfix

import "github.com/gonewx/godi"

// IFakeService / FakeService is the dependency leaf: a no-dependency singleton
// whose New{Impl} constructor takes no arguments and returns no error.
type IFakeService interface{ isFakeService() }

type FakeService struct{}

func (*FakeService) isFakeService() {}

// NewFakeService is the form-A New{Impl} constructor for FakeService — no
// dependencies, no trailing error.
func NewFakeService() *FakeService { return &FakeService{} }

// IConsumer / Consumer is the dependent: a singleton whose New{Impl} constructor
// takes the IFakeService dependency and returns a trailing error, exercising
// both dependency wiring (AC1) and error propagation (AD-8 / D-1).
type IConsumer interface{ consume() }

type Consumer struct{ dep IFakeService }

func (*Consumer) consume() {}

// NewConsumer is the form-A New{Impl} constructor for Consumer. Its IFakeService
// parameter is the dependency the generator must discover and wire as
// r.f_FakeService; the trailing error drives the NewRoot() (*Root, error)
// signature and comma-err propagation.
func NewConsumer(dep IFakeService) (*Consumer, error) { return &Consumer{dep: dep}, nil }

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture, the
// process would panic. go/packages only parses and type-checks it, so a clean
// generate result is positive evidence of non-execution.
func init() {
	panic("AD-7 violation: deps registration fixture was executed during static analysis")
}

// register holds the statically-analyzed DSL markers. The generic calls are read
// from types.Info.Instances; this function is never invoked at runtime.
func register() {
	var r godi.Registry
	godi.AddSingleton[IFakeService, FakeService](&r)
	godi.AddSingleton[IConsumer, Consumer](&r)
}
