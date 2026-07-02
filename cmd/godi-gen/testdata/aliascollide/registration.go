// Package aliasfix registers services from TWO packages that share the
// declared name "opts" (suba + subb) — the gois reality where
// gois/options and gois/config/options are both `package options`. The
// generator must bind one of them under a numbered alias in the emitted
// import block and qualify every reference (GoType, ctor, rewritten factory
// text) consistently. Test INPUT under testdata/, never part of the module
// build.
package aliasfix

import (
	"github.com/gonewx/godi"
	aopts "github.com/gonewx/godi/cmd/godi-gen/testdata/aliascollide/suba"
	bopts "github.com/gonewx/godi/cmd/godi-gen/testdata/aliascollide/subb"
)

// init is an AD-7 tripwire (see the other fixtures).
func init() {
	panic("AD-7 violation: aliascollide registration fixture was executed during static analysis")
}

// register holds the statically-analyzed markers: form A from the first
// same-named package, form B from the second — the factory body's `bopts.`
// qualifier must be remapped to whatever identifier the emitted file imports
// subb under.
func register() {
	var r godi.Registry
	godi.AddSingleton[aopts.IThingA, aopts.ThingA](&r)
	godi.AddSingletonFactory[bopts.IThingB](&r, func(p godi.Provider) bopts.IThingB {
		return bopts.NewThingB()
	})
}
