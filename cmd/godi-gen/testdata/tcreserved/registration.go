// Package tcreserved is a type-check gate fixture: implementation names
// (IsKeyedService, Get) collide with methods the emitted Root already declares,
// so the generated accessors redeclare them and the container does not compile.
// Test INPUT only (testdata/).
package tcreserved

import "github.com/gonewx/godi"

type IFoo interface{ Foo() }
type IsKeyedService struct{}

func (*IsKeyedService) Foo()             {}
func NewIsKeyedService() *IsKeyedService { return &IsKeyedService{} }

type IBar interface{ Bar() }
type Get struct{}

func (*Get) Bar()  {}
func NewGet() *Get { return &Get{} }

// init is an AD-7 tripwire: static analysis must never execute this fixture.
func init() {
	panic("AD-7 violation: tcreserved registration fixture was executed during static analysis")
}

func register() {
	var r godi.Registry
	godi.AddSingleton[IFoo, IsKeyedService](&r)
	godi.AddSingleton[IBar, Get](&r)
}
