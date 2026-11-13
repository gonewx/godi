// Package lazytransientdupdep exercises a transient whose constructor takes
// the same error-returning transient twice: each argument must be bound to its
// own temporary in the emitted New<N>() body, both for a transient whose own
// constructor returns no error (Foo) and for one that does (ErrFoo). Test
// INPUT only (testdata/); it is generated, compiled and run by
// lazy_singleton_test.go.
package lazytransientdupdep

import "github.com/gonewx/godi"

// built numbers each X so two instances are distinguishable by value as well
// as by identity.
var built int

// IX / X is the error-returning transient dependency. X has a field: pointers
// to distinct zero-size values may compare equal.
type IX interface{ ID() int }

type X struct{ id int }

func (x *X) ID() int { return x.id }

func NewX() (*X, error) {
	built++
	return &X{id: built}, nil
}

// IFoo / Foo takes IX twice; NewFoo returns no error.
type IFoo interface{ Deps() (IX, IX) }

type Foo struct{ a, b IX }

func (f *Foo) Deps() (IX, IX) { return f.a, f.b }

func NewFoo(a IX, b IX) *Foo { return &Foo{a: a, b: b} }

// IErrFoo / ErrFoo takes IX twice; NewErrFoo itself returns an error.
type IErrFoo interface{ Deps() (IX, IX) }

type ErrFoo struct{ a, b IX }

func (f *ErrFoo) Deps() (IX, IX) { return f.a, f.b }

func NewErrFoo(a IX, b IX) (*ErrFoo, error) { return &ErrFoo{a: a, b: b}, nil }

func register() {
	var r godi.Registry
	godi.AddTransient[IX, X](&r)
	godi.AddTransient[IFoo, Foo](&r)
	godi.AddTransient[IErrFoo, ErrFoo](&r)
}
