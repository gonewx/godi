package lazynamedfactory

import "github.com/gonewx/godi"

type Cfg struct{ DSN string }

func NewCfg() *Cfg { return &Cfg{DSN: "real"} }

type IFoo interface{ Cfg() *Cfg }
type Foo struct{ cfg *Cfg }

func (f *Foo) Cfg() *Cfg { return f.cfg }

func newFoo(p godi.Provider) IFoo { return &Foo{cfg: godi.GetRequired[*Cfg](p)} }

func register() {
	var r godi.Registry
	godi.AddSingletonFactory[IFoo](&r, newFoo)
	godi.AddSingleton[*Cfg, Cfg](&r)
}
