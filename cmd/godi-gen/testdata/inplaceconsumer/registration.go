// Package inplaceconsumer is the in-place-layout fixture: the container is
// generated into this directory, and hand-written code in the same package
// refers to it (Root, NewRoot, typed accessors). inplace_consumer_test.go writes
// the container, the consumer and any extra registration files at run time (and
// removes them afterwards) — a committed consumer would make the very first
// generation fail, and a committed container would go stale whenever the
// emitter changes. It is test INPUT, not part of the module build: it lives
// under testdata/.
package inplaceconsumer

import "github.com/gonewx/godi"

// ILog / Log is the dependency leaf: NewLog takes no arguments.
type ILog interface{ log() }

type Log struct{}

func (*Log) log() {}

func NewLog() *Log { return &Log{} }

// IFoo / Foo depends on ILog.
type IFoo interface{ foo() }

type Foo struct{ l ILog }

func (*Foo) foo() {}

func NewFoo(l ILog) *Foo { return &Foo{l: l} }

// IBar / Bar is not registered here; tests register it from a file written at
// run time.
type IBar interface{ bar() }

type Bar struct{}

func (*Bar) bar() {}

func NewBar() *Bar { return &Bar{} }

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture, the
// process would panic. go/packages only parses and type-checks it.
func init() {
	panic("AD-7 violation: inplaceconsumer registration fixture was executed during static analysis")
}

// register holds the statically-analyzed DSL markers; it is never invoked.
func register() {
	var r godi.Registry
	godi.AddSingleton[ILog, Log](&r)
	godi.AddSingleton[IFoo, Foo](&r)
}
