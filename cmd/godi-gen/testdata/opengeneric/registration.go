// Package opengenericfix is the per-service open-generic HELPER fixture the
// Story 4.1 generator loads via go/packages. It is test INPUT, not part of the
// module build: it lives under testdata/, so `go build ./...` / `go test ./...`
// never compile it.
//
// It exercises Mechanism A (AC1): the generator follows a per-service helper
// AddBox[T] (a single-statement forwarder to godi.AddTransient[IBox[T], Box[T]]),
// substitutes T with each call's type argument, and materializes the closed
// (IBox[IDep], Box[IDep]) / (IBox[IOther], Box[IOther]) registrations — wiring the
// injected `value T` dependency from the closed constructor signature (AC2). The
// inner TEMPLATE marker godi.AddTransient[IBox[T], Box[T]] (type args still
// mention T) must be SKIPPED, not processed as an open registration (AC1).
package opengenericfix

import "github.com/gonewx/godi"

// IDep / Dep is a leaf dependency: the T the first Box is closed over, injected
// into the closed Box[IDep] constructor.
type IDep interface{ isDep() }

type Dep struct{}

func (*Dep) isDep() {}

// NewDep is the form-A New{Impl} constructor for Dep — no dependencies.
func NewDep() *Dep { return &Dep{} }

// IOther / Other is a SECOND leaf dependency, proving the two closed
// instantiations (Box[IDep] / Box[IOther]) coexist with disambiguated names (AC4).
type IOther interface{ isOther() }

type Other struct{}

func (*Other) isOther() {}

// NewOther is the form-A New{Impl} constructor for Other — no dependencies.
func NewOther() *Other { return &Other{} }

// IBox[T] / Box[T] mirror the open-generic shape: the constructor INJECTS value T
// (like fakes.FakeOpenGenericService). When closed over a registered service the
// generator instantiates this signature and wires the injected dependency (AC2).
type IBox[T any] interface {
	isBox()
	Value() T
}

type Box[T any] struct{ value T }

func (*Box[T]) isBox() {}

func (b *Box[T]) Value() T { return b.value }

// NewBox is the form-A New{Impl} constructor injecting value T.
func NewBox[T any](value T) *Box[T] { return &Box[T]{value: value} }

// AddBox is the per-service open-generic helper (Mechanism A): a single-statement
// forwarder to a godi marker. The generator recognizes it, reads the call's type
// argument, and reduces the inner template marker to a closed form-A registration.
func AddBox[T any](r *godi.Registry) *godi.Registry {
	return godi.AddTransient[IBox[T], Box[T]](r)
}

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture the
// process would panic; go/packages only parses and type-checks it.
func init() {
	panic("AD-7 violation: opengeneric registration fixture was executed during static analysis")
}

// register holds the statically-analyzed markers. AddBox[IDep] / AddBox[IOther]
// enumerate two closed instantiations of the same open generic; the generator
// materializes each and injects its registered dependency.
func register() {
	var r godi.Registry
	godi.AddSingleton[IDep, Dep](&r)
	godi.AddSingleton[IOther, Other](&r)
	AddBox[IDep](&r)
	AddBox[IOther](&r)
}
