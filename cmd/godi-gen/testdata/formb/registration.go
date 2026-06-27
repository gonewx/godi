// Package formbfix is the form-B escape-hatch fixture (D-2). The implementation
// is registered with AddSingletonFactory and a factory expression; the Story 2.1
// generator must statically read that factory ARGUMENT EXPRESSION (go/ast source
// text) and inline it into NewRoot as r.f_I = (factoryExpr)(r), rather than
// discovering a New{Impl} constructor.
//
// It is test INPUT under testdata/, never part of the module build.
package formbfix

import "github.com/gonewx/godi"

// IWidget / Widget is registered through form B. Whether or not it has a
// New{Impl}, form B inlines the factory expression verbatim (the generator does
// not look for a constructor on the form-B path).
type IWidget interface{ widget() }

type Widget struct{}

func (*Widget) widget() {}

// NewWidget exists so the factory body is type-correct; the generator never
// discovers it — it inlines the whole factory expression instead (D-2).
func NewWidget() *Widget { return &Widget{} }

// register holds the form-B marker. The generator reads the second argument
// (the factory func literal) as source text and inlines it; the call is never
// executed at runtime (AD-7).
func register() {
	var r godi.Registry
	godi.AddSingletonFactory[IWidget](&r, func(p godi.Provider) IWidget { return NewWidget() })
}
