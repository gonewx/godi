// Package inferredhelper is the per-service helper fixture whose BODY uses an
// inferred-type-argument marker (Task 7): godi.AddSingletonFactory(r, f) with
// [IFoo[T]] inferred from the factory's result type. The generator must still
// recognize AddFoo as a helper and materialize the closed IFoo[int] service.
//
// It is test INPUT under testdata/, never part of the module build.
package inferredhelper

import "github.com/gonewx/godi"

type IDep interface{ Dep() }
type Dep struct{}

func (*Dep) Dep()  {}
func NewDep() *Dep { return &Dep{} }

type IFoo[T any] interface{ Foo() T }
type Foo[T any] struct{ v T }

func (f *Foo[T]) Foo() T { return f.v }

// AddFoo is a per-service helper forwarding to an inferred form-B marker.
func AddFoo[T any](r *godi.Registry) *godi.Registry {
	return godi.AddSingletonFactory(r, func(p godi.Provider) IFoo[T] { return &Foo[T]{} })
}

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture the
// process would panic; go/packages only parses and type-checks it.
func init() {
	panic("AD-7 violation: inferredhelper registration fixture was executed during static analysis")
}

func register() {
	var r godi.Registry
	godi.AddSingleton[IDep, Dep](&r)
	AddFoo[int](&r)
}
