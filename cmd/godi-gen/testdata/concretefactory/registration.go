// Package concretefactory is a fixture for form-B factories whose service type
// is a concrete pointer rather than a named interface.
package concretefactory

import "github.com/gonewx/godi"

type Config struct {
	Name string
}

type Child struct {
	Config *Config
}

func register() {
	var r godi.Registry
	godi.AddSingletonFactory[*Config](&r, func(godi.Provider) *Config {
		return &Config{Name: "default"}
	})
	godi.AddTransientFactory[*Child](&r, func(p godi.Provider) *Child {
		return &Child{Config: godi.GetRequired[*Config](p)}
	})
}
