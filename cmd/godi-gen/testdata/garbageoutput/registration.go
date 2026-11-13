// Package garbageoutput is the broken-output fixture (evaluation report §7.3,
// finding C1). TestGenerate_IgnoresSyntacticallyBrokenGeneratedOutput writes a
// container_gen.go beside it at run time (and removes it afterwards) that
// carries the godi-gen header but does not parse, as after an interrupted
// write; it is not committed because CI's gofmt step would choke on it. The
// generator must ignore that file when loading this package and regenerate
// from the registrations. It is test INPUT, not part of the module build: it
// lives under testdata/.
package garbageoutput

import "github.com/gonewx/godi"

// IFoo / Foo is a single no-dependency singleton.
type IFoo interface{ foo() }

type Foo struct{}

func (*Foo) foo() {}

func NewFoo() *Foo { return &Foo{} }

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture, the
// process would panic. go/packages only parses and type-checks it.
func init() {
	panic("AD-7 violation: garbageoutput registration fixture was executed during static analysis")
}

// register holds the statically-analyzed DSL markers; it is never invoked.
func register() {
	var r godi.Registry
	godi.AddSingleton[IFoo, Foo](&r)
}
