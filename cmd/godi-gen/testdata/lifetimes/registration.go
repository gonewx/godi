// Package lifetimesfix is the three-lifetime fixture the Story 2.2 generator
// test loads via go/packages. It is test INPUT, not part of the module build:
// it lives under testdata/, so `go build ./...` / `go test ./...` never
// compile it.
//
// The generator must recognize AddScoped[I,Impl], AddTransient[I,Impl],
// AddScopedFactory[I], and AddTransientFactory[I] markers and map them to the
// correct plan.Lifetime (Task 3, D-6). The fixture also includes existing
// AddSingleton markers to test mixed-lifetime registration.
//
// AD-7 tripwire: init() panics if this fixture is ever executed.
package lifetimesfix

import "github.com/gonewx/godi"

type ILogger interface{ log() }
type Logger struct{}

func (*Logger) log() {}

func NewLogger() *Logger { return &Logger{} }

type IRequest interface{ request() }
type Request struct{ logger ILogger }

func (*Request) request() {}

func NewRequest(logger ILogger) *Request { return &Request{logger: logger} }

type IWidget interface{ widget() }
type Widget struct{ req IRequest }

func (*Widget) widget() {}

func NewWidget(req IRequest) *Widget { return &Widget{req: req} }

func init() {
	panic("AD-7 violation: lifetimes registration fixture was executed during static analysis")
}

func register() {
	var r godi.Registry
	godi.AddSingleton[ILogger, Logger](&r)
	godi.AddScoped[IRequest, Request](&r)
	godi.AddTransient[IWidget, Widget](&r)
}
