package formbmulti

import "github.com/gonewx/godi"

type IWidget interface {
	Name() string
}

type Widget string

func (w Widget) Name() string { return string(w) }

func register() {
	var r godi.Registry
	godi.AddSingletonFactory[IWidget](&r, func(godi.Provider) IWidget { return Widget("first") })
	godi.AddSingletonFactory[IWidget](&r, func(godi.Provider) IWidget { return Widget("second") })
}
