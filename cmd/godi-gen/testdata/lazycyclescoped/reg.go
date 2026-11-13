package lazycyclescoped

import "github.com/gonewx/godi"

type IA interface{ A() }
type A struct{ b IB }

func (*A) A() {}

type IB interface{ B() }
type B struct{ a IA }

func (*B) B() {}

type IC interface{ C() }
type C struct{}

func (*C) C()  {}
func NewC() *C { return &C{} }

// newA and newB are named functions: the generator does not look inside them,
// so the cycle between them is only detectable when the container runs.
func newA(p godi.Provider) IA { return &A{b: godi.GetRequired[IB](p)} }
func newB(p godi.Provider) IB { return &B{a: godi.GetRequired[IA](p)} }

func register() {
	var r godi.Registry
	godi.AddScopedFactory[IA](&r, newA)
	godi.AddScopedFactory[IB](&r, newB)
	godi.AddScoped[IC, C](&r)
}
