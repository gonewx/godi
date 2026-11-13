package lazykeyederr

import "github.com/gonewx/godi"

type IFoo interface{ Foo() }
type Foo struct{}

func (*Foo) Foo()           {}
func NewFoo() (*Foo, error) { return &Foo{}, nil }

func register() {
	var r godi.Registry
	godi.AddKeyedSingleton[IFoo, Foo](&r, "k")
}
