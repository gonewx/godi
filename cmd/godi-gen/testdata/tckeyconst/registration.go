// Package tckeyconst is a type-check gate fixture: the registration key is a
// constant from another package (keys.Primary), whose qualifier the emitted
// container does not import, so it does not compile. Test INPUT only
// (testdata/).
package tckeyconst

import (
	"github.com/gonewx/godi/cmd/godi-gen/testdata/tckeyconst/keys"

	"github.com/gonewx/godi"
)

type IFoo interface{ Foo() }
type Foo struct{}

func (*Foo) Foo()  {}
func NewFoo() *Foo { return &Foo{} }

// init is an AD-7 tripwire: static analysis must never execute this fixture.
func init() {
	panic("AD-7 violation: tckeyconst registration fixture was executed during static analysis")
}

func register() {
	var r godi.Registry
	godi.AddKeyedSingleton[IFoo, Foo](&r, keys.Primary)
}
