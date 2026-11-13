// Package tcconcreteparam is a type-check gate fixture: NewBar injects the
// concrete *Foo, but the container stores the Foo service in a field typed as
// the IFoo interface, so the emitted NewBar(r.f_Foo) passes an IFoo where *Foo
// is required and does not compile. Test INPUT only (testdata/).
package tcconcreteparam

import "github.com/gonewx/godi"

type IFoo interface{ Foo() }
type Foo struct{}

func (*Foo) Foo()  {}
func NewFoo() *Foo { return &Foo{} }

type IBar interface{ Bar() }
type Bar struct{ f *Foo }

func (*Bar) Bar()        {}
func NewBar(f *Foo) *Bar { return &Bar{f: f} }

// init is an AD-7 tripwire: static analysis must never execute this fixture.
func init() {
	panic("AD-7 violation: tcconcreteparam registration fixture was executed during static analysis")
}

func register() {
	var r godi.Registry
	godi.AddSingleton[IFoo, Foo](&r)
	godi.AddSingleton[IBar, Bar](&r)
}
