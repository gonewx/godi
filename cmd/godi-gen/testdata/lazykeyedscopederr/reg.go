package lazykeyedscopederr

import (
	"errors"

	"github.com/gonewx/godi"
)

var FailBar bool

type IFoo interface{ Foo() }
type Foo struct{}

func (*Foo) Foo()           {}
func NewFoo() (*Foo, error) { return &Foo{}, nil }

type IBar interface{ Bar() }
type Bar struct{}

func (*Bar) Bar() {}
func NewBar() (*Bar, error) {
	if FailBar {
		return nil, errors.New("bar unavailable")
	}
	return &Bar{}, nil
}

func register() {
	var r godi.Registry
	godi.AddKeyedScoped[IFoo, Foo](&r, "scoped")
	godi.AddKeyedTransient[IBar, Bar](&r, "transient")
}
