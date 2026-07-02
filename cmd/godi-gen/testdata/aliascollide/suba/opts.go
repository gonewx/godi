// Package opts (suba) deliberately shares its declared package NAME with
// testdata/aliascollide/subb: the generator must rename one of them in the
// emitted import block (options / options2-style) instead of emitting two
// same-named imports. Test INPUT only.
package opts

// IThingA / ThingA is a plain form-A service living in the FIRST of the two
// same-named packages.
type IThingA interface{ isThingA() }

type ThingA struct{}

func (*ThingA) isThingA() {}

func NewThingA() *ThingA { return &ThingA{} }
