package transientcollectiondeps

import "github.com/gonewx/godi"

type ISetup interface {
	Apply()
}

type IFactory interface {
	Create()
}

type IMonitor interface {
	Current()
}

type setup struct{}

func (setup) Apply() {}

type factory struct {
	setups []ISetup
}

func (factory) Create() {}

type monitor struct {
	factory IFactory
}

func (monitor) Current() {}

func register() {
	var r godi.Registry
	godi.AddSingletonFactory[IMonitor](&r, func(p godi.Provider) IMonitor {
		return monitor{factory: godi.GetRequired[IFactory](p)}
	})
	godi.AddTransientFactory[IFactory](&r, func(p godi.Provider) IFactory {
		return factory{setups: godi.GetServices[ISetup](p)}
	})
	godi.AddSingletonFactory[ISetup](&r, func(godi.Provider) ISetup {
		return setup{}
	})
}
