// Package tcnotimpl is a type-check gate fixture: Baz does not implement IFoo,
// so the emitted container assigns a *Baz to an IFoo field and does not
// compile. Test INPUT only (testdata/).
package tcnotimpl

import "github.com/gonewx/godi"

type IFoo interface{ Foo() }
type Baz struct{}

func NewBaz() *Baz { return &Baz{} }

// init is an AD-7 tripwire: static analysis must never execute this fixture.
func init() {
	panic("AD-7 violation: tcnotimpl registration fixture was executed during static analysis")
}

func register() {
	var r godi.Registry
	godi.AddSingleton[IFoo, Baz](&r)
}
