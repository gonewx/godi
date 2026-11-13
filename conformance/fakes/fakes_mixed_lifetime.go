package fakes

// --- Mixed-lifetime fakes (eval fix A4) ---------------------------------------
//
// One service type registered several times with DIFFERENT lifetimes. dotnet
// resolves such a type by registration order: a single resolve returns the
// last registration whatever its lifetime, and a collection resolve returns
// every registration in registration order, each element keeping its own
// lifetime. IMixedA is registered singleton then transient; IMixedB transient
// then singleton. The two interfaces share a method set but are declared
// separately so each is its own service type.

// IMixedLifetime is the common method set of IMixedA and IMixedB, used by the
// tests to read Name() from either.
type IMixedLifetime interface {
	Name() string
}

// IMixedA is registered singleton (MixedSingletonFirst) then transient
// (MixedTransientLast).
type IMixedA interface {
	Name() string
}

// IMixedB is registered transient (MixedTransientFirst) then singleton
// (MixedSingletonLast).
type IMixedB interface {
	Name() string
}

// MixedSingletonFirst is IMixedA's first, singleton registration. The field
// keeps the struct non-zero-sized so distinct instances have distinct addresses.
type MixedSingletonFirst struct{ _ int }

// Name returns the type name.
func (*MixedSingletonFirst) Name() string { return "MixedSingletonFirst" }

// NewMixedSingletonFirst is the form-A constructor (AD-8 New{Impl}).
func NewMixedSingletonFirst() *MixedSingletonFirst { return &MixedSingletonFirst{} }

// MixedTransientLast is IMixedA's last, transient registration.
type MixedTransientLast struct{ _ int }

// Name returns the type name.
func (*MixedTransientLast) Name() string { return "MixedTransientLast" }

// NewMixedTransientLast is the form-A constructor (AD-8 New{Impl}).
func NewMixedTransientLast() *MixedTransientLast { return &MixedTransientLast{} }

// MixedTransientFirst is IMixedB's first, transient registration.
type MixedTransientFirst struct{ _ int }

// Name returns the type name.
func (*MixedTransientFirst) Name() string { return "MixedTransientFirst" }

// NewMixedTransientFirst is the form-A constructor (AD-8 New{Impl}).
func NewMixedTransientFirst() *MixedTransientFirst { return &MixedTransientFirst{} }

// MixedSingletonLast is IMixedB's last, singleton registration.
type MixedSingletonLast struct{ _ int }

// Name returns the type name.
func (*MixedSingletonLast) Name() string { return "MixedSingletonLast" }

// NewMixedSingletonLast is the form-A constructor (AD-8 New{Impl}).
func NewMixedSingletonLast() *MixedSingletonLast { return &MixedSingletonLast{} }
