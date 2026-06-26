// Package spikefix is the minimal user-registration fixture the AD-7 spike
// loads via go/packages. It is test INPUT, not part of the module build: it
// lives under testdata/, so `go build ./...` and `go test ./...` never compile
// it. The spike reads the type arguments of the AddSingleton marker below
// statically (go/types) and never executes this code.
package spikefix

import "github.com/gonewx/godi"

// ILogger is the service interface; Logger is its implementation. The spike
// must recover both names from godi.AddSingleton's type arguments.
type ILogger interface {
	Log(msg string)
}

type Logger struct{}

func (Logger) Log(string) {}

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture, the
// process would panic. go/packages only parses and type-checks it, so this
// never runs — the spike's green result is positive evidence of non-execution.
func init() {
	panic("AD-7 violation: registration fixture was executed during static analysis")
}

// register holds the statically-analyzed DSL marker. The generic call
// AddSingleton[ILogger, Logger] is what the spike reads from
// types.Info.Instances; the function is never called at runtime.
func register() {
	var r godi.Registry
	godi.AddSingleton[ILogger, Logger](&r)
}
