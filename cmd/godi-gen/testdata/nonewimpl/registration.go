// Package nonewimplfix is the AC2 error-gate fixture: a form-A registration of
// an implementation that has NO New{Impl} constructor. The Story 2.1 generator
// must reject it at generation time with a hint to use form B
// (AddSingletonFactory), emit ZERO bytes, and exit non-zero (NFR2 hard gate).
//
// It is test INPUT under testdata/, never part of the module build.
package nonewimplfix

import "github.com/gonewx/godi"

// IOrphan / Orphan deliberately has NO NewOrphan constructor — there is no
// form-A New{Impl} for the generator to discover, which is exactly the AC2
// condition.
type IOrphan interface{ orphan() }

type Orphan struct{}

func (*Orphan) orphan() {}

// register marks Orphan with form A even though it lacks a New{Impl} — the
// generator's missing-constructor gate must fire on this.
func register() {
	var r godi.Registry
	godi.AddSingleton[IOrphan, Orphan](&r)
}
