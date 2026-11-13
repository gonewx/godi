package lazyerrtransient

import "github.com/gonewx/godi"

type IT interface{ T() }
type T struct{}

func (*T) T()           {}
func NewT() (*T, error) { return &T{}, nil }

type IA interface{ A() }
type A struct{}

func (*A) A()      {}
func NewA(t IT) *A { return &A{} }

type IB interface{ B() }
type B struct{}

func (*B) B()      {}
func NewB(t IT) *B { return &B{} }

func register() {
	var r godi.Registry
	godi.AddTransient[IT, T](&r)
	godi.AddSingleton[IA, A](&r)
	godi.AddSingleton[IB, B](&r)
}
