package lazymissing

import "github.com/gonewx/godi"

type IDep interface{ D() }
type IFoo interface{ F() }
type Foo struct{ d IDep }

func (*Foo) F() {}

func register() {
	var r godi.Registry
	godi.AddSingletonFactory[IFoo](&r, func(p godi.Provider) IFoo { return &Foo{d: godi.GetRequired[IDep](p)} })
}
