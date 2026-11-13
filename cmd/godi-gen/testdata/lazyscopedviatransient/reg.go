package lazyscopedviatransient

import "github.com/gonewx/godi"

type IB interface{ B() }
type B struct{}

func (*B) B()  {}
func NewB() *B { return &B{} }

type IT interface{ Dep() IB }
type T struct{ b IB }

func (t *T) Dep() IB { return t.b }
func NewT(b IB) *T   { return &T{b: b} }

type IA interface{ T() IT }
type A struct{ t IT }

func (a *A) T() IT { return a.t }
func NewA(t IT) *A { return &A{t: t} }

func register() {
	var r godi.Registry
	godi.AddScoped[IA, A](&r)
	godi.AddTransient[IT, T](&r)
	godi.AddScoped[IB, B](&r)
}
