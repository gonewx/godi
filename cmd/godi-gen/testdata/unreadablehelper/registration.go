// Package unreadablehelper is the unsupported per-service helper fixture (Task
// 7). AddBox is a generic helper whose body mixes a godi marker with an
// ordinary statement, a shape the generator cannot reduce. Its call must
// REJECT generation with a source-positioned error rather than warn and emit a
// container silently missing the helper's service. The legal form-A marker
// proves the rejection is not the "no registrations" error.
//
// It is test INPUT under testdata/, never part of the module build.
package unreadablehelper

import "github.com/gonewx/godi"

type IFoo interface{ Foo() }
type Foo struct{}

func (*Foo) Foo()  {}
func NewFoo() *Foo { return &Foo{} }

type IBox[T any] interface{ Box() T }
type Box[T any] struct{ v T }

func (b *Box[T]) Box() T     { return b.v }
func NewBox[T any]() *Box[T] { return &Box[T]{} }

var calls int

// AddBox mixes a non-marker statement into the helper body — unsupported.
func AddBox[T any](r *godi.Registry) *godi.Registry {
	calls++
	return godi.AddTransient[IBox[T], Box[T]](r)
}

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture the
// process would panic; go/packages only parses and type-checks it.
func init() {
	panic("AD-7 violation: unreadablehelper registration fixture was executed during static analysis")
}

func register() {
	var r godi.Registry
	godi.AddSingleton[IFoo, Foo](&r)
	AddBox[int](&r)
}
