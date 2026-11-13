// Package staleoutput is the stale-output fixture (evaluation report §7.3,
// finding C1). It sits beside container_gen.go, a previous godi-gen product
// emitted when NewFoo still took no arguments. NewFoo has since gained an ILog
// parameter, so the stale container no longer type-checks; the generator must
// ignore it when loading this package and regenerate from the registrations.
// It is test INPUT, not part of the module build: it lives under testdata/.
package staleoutput

import "github.com/gonewx/godi"

// ILog / Log is the dependency leaf: NewLog takes no arguments.
type ILog interface{ log() }

type Log struct{}

func (*Log) log() {}

func NewLog() *Log { return &Log{} }

// IFoo / Foo depends on ILog. NewFoo gained the l parameter after
// container_gen.go was generated.
type IFoo interface{ foo() }

type Foo struct{ l ILog }

func (*Foo) foo() {}

func NewFoo(l ILog) *Foo { return &Foo{l: l} }

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture, the
// process would panic. go/packages only parses and type-checks it.
func init() {
	panic("AD-7 violation: staleoutput registration fixture was executed during static analysis")
}

// register holds the statically-analyzed DSL markers; it is never invoked.
func register() {
	var r godi.Registry
	godi.AddSingleton[ILog, Log](&r)
	godi.AddSingleton[IFoo, Foo](&r)
}
