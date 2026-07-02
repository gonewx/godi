// Package opts (subb) deliberately shares its declared package NAME with
// testdata/aliascollide/suba — see suba/opts.go. Test INPUT only.
package opts

// IThingB / ThingB is a service living in the SECOND same-named package; its
// constructor returns the interface, so it can only be registered form B —
// exercising the factory-rewrite path under an ALIASED package qualifier.
type IThingB interface{ isThingB() }

type thingB struct{}

func (*thingB) isThingB() {}

func NewThingB() IThingB { return &thingB{} }
