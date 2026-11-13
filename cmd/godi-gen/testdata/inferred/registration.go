// Package inferred is the inferred-type-argument marker fixture (Task 7). The
// form-B marker below omits its [ILog] type argument; Go infers it from the
// factory's result type (gopls' infertypeargs analyzer even suggests deleting
// the explicit argument). The generator must still read ILog from
// Info.Instances and register it instead of skipping it with a warning.
//
// It is test INPUT under testdata/, never part of the module build.
package inferred

import "github.com/gonewx/godi"

type IFoo interface{ Foo() }
type Foo struct{}

func (*Foo) Foo()  {}
func NewFoo() *Foo { return &Foo{} }

type ILog interface{ Log() }
type Log struct{}

func (*Log) Log() {}

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture the
// process would panic; go/packages only parses and type-checks it.
func init() {
	panic("AD-7 violation: inferred registration fixture was executed during static analysis")
}

func register() {
	var r godi.Registry
	godi.AddSingleton[IFoo, Foo](&r)
	// 类型实参由 Go 推断（gopls infertypeargs 会建议写成这样）
	godi.AddSingletonFactory(&r, func(p godi.Provider) ILog { return &Log{} })
}
