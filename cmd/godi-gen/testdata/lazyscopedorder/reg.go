package lazyscopedorder

import (
	"errors"

	"github.com/gonewx/godi"
)

var Built []string
var Disposed []string
var FailC bool

type IA interface{ A() }
type A struct{}

func (*A) A()           {}
func (*A) Close() error { Disposed = append(Disposed, "A"); return nil }
func NewA() *A          { Built = append(Built, "A"); return &A{} }

type IB interface{ B() }
type B struct{}

func (*B) B()           {}
func (*B) Close() error { Disposed = append(Disposed, "B"); return nil }
func NewB() *B          { Built = append(Built, "B"); return &B{} }

type IC interface{ C() }
type C struct{}

func (*C) C() {}
func NewC() (*C, error) {
	Built = append(Built, "C")
	if FailC {
		return nil, errors.New("C unavailable")
	}
	return &C{}, nil
}

func register() {
	var r godi.Registry
	godi.AddScoped[IA, A](&r)
	godi.AddScoped[IB, B](&r)
	godi.AddScoped[IC, C](&r)
}
