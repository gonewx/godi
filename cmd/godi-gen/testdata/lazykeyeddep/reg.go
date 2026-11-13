package lazykeyeddep

import "github.com/gonewx/godi"

type IDep interface{ D() }
type Dep struct{}

func (*Dep) D()    {}
func NewDep() *Dep { return &Dep{} }

type IFoo interface{ Dep() IDep }
type Foo struct{ d IDep }

func (f *Foo) Dep() IDep { return f.d }

func register() {
	var r godi.Registry
	godi.AddKeyedSingleton[IDep, Dep](&r, "k")
	godi.AddSingletonFactory[IFoo](&r, func(p godi.Provider) IFoo {
		return &Foo{d: godi.GetRequiredKeyed[IDep](p, "k")}
	})
}
