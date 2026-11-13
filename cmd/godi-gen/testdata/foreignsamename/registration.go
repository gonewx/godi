// Package foreignsamename is the same-named foreign function fixture (Task 7).
// AddSingleton below is a local, non-generic function unrelated to godi that
// merely shares a marker's name. Calling it is a legitimate non-match: the
// generator warns but must NOT reject generation.
//
// It also guards two ordinary shapes that must stay non-helpers: a
// NON-generic local function whose body holds concrete markers
// (registerMore), and a generic local wrapper around a non-marker godi call
// (resolve, used inside a form-B factory). Neither may be mistaken for an
// unsupported per-service helper.
//
// It is test INPUT under testdata/, never part of the module build.
package foreignsamename

import "github.com/gonewx/godi"

type IFoo interface{ Foo() }
type Foo struct{}

func (*Foo) Foo()  {}
func NewFoo() *Foo { return &Foo{} }

type IBar interface{ Bar() }
type Bar struct{}

func (*Bar) Bar()  {}
func NewBar() *Bar { return &Bar{} }

type IBaz interface{ Baz() }
type Baz struct{ foo IFoo }

func (*Baz) Baz() {}

var names []string

// registerMore is an ordinary non-generic registration function.
func registerMore(r *godi.Registry) {
	godi.AddSingleton[IBar, Bar](r)
}

// resolve is a generic wrapper around a godi call that is not a marker.
func resolve[T any](p godi.Provider) T {
	return godi.GetRequired[T](p)
}

// AddSingleton is NOT a godi marker — same name, unrelated function.
func AddSingleton(name string) { names = append(names, name) }

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture the
// process would panic; go/packages only parses and type-checks it.
func init() {
	panic("AD-7 violation: foreignsamename registration fixture was executed during static analysis")
}

func register() {
	var r godi.Registry
	godi.AddSingleton[IFoo, Foo](&r)
	AddSingleton("not-a-marker")
	registerMore(&r)
	godi.AddSingletonFactory[IBaz](&r, func(p godi.Provider) IBaz { return &Baz{foo: resolve[IFoo](p)} })
}
