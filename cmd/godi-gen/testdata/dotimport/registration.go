// Package dotimport is the dot-imported godi fixture (Task 7). Markers are
// called unqualified, and the form-B marker's [ILog] type argument is
// inferred. The callee identifier still resolves to the godi package through
// Info.Uses, so the generator must register both services.
//
// The form-B factory is a named function: a factory literal would spell the
// dot-imported godi.Provider as a bare `Provider`, which the emitted container
// (importing godi by name) cannot resolve — an emission limit, not a
// recognition one.
//
// It is test INPUT under testdata/, never part of the module build.
package dotimport

import . "github.com/gonewx/godi"

type IFoo interface{ Foo() }
type Foo struct{}

func (*Foo) Foo()  {}
func NewFoo() *Foo { return &Foo{} }

type ILog interface{ Log() }
type Log struct{}

func (*Log) Log() {}

func newLog(p Provider) ILog { return &Log{} }

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture the
// process would panic; go/packages only parses and type-checks it.
func init() {
	panic("AD-7 violation: dotimport registration fixture was executed during static analysis")
}

func register() {
	var r Registry
	AddSingleton[IFoo, Foo](&r)
	AddSingletonFactory(&r, newLog)
}
