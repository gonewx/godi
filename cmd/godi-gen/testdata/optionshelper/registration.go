// Package optionsfix is the multi-statement / form-B-forwarding per-service
// helper fixture (godi-gen Task-0 extension for gois story 2-1). It is test
// INPUT under testdata/, never part of the module build.
//
// It mirrors the dotnet Options "four-piece" AddOptions[T] shape the extension
// exists for:
//
//   - a CRTP-constrained helper (T Entity[T]) whose body is FOUR marker
//     statements (three expression statements + a final return marker);
//   - two form-A registrations (Options → UnnamedOptionsManager,
//     OptionsMonitorCache → OptionsCache);
//   - two form-B registrations whose implementations are UNEXPORTED and whose
//     constructors return the interface (OptionsMonitor, OptionsFactory) — the
//     shape form A cannot carry;
//   - factory bodies resolving through godi.GetRequired / godi.GetServices,
//     including a GetServices over a NEVER-REGISTERED element type
//     (ConfigureOptions — the legitimate empty configure chain);
//   - the cache registered LAST in body order while the monitor's factory
//     GetRequired-s it — the ordering case factoryDeps must feed
//     SingletonsTopo for.
package optionsfix

import "github.com/gonewx/godi"

// Entity is the CRTP constraint (the Go rendering of dotnet's `new()`): a
// zero value of T can mint a fresh T.
type Entity[T any] interface {
	New() T
}

// ConfigureOptions is the configure-chain element type. It is intentionally
// NEVER registered: the factory's GetServices must reduce to an empty slice
// and add no ordering constraint.
type ConfigureOptions[T Entity[T]] interface {
	Configure(T)
}

// Options / UnnamedOptionsManager is form-A piece #1: an exported impl with a
// New{Impl} constructor whose parameter injects the TRANSIENT OptionsFactory.
type Options[T Entity[T]] interface {
	Value() T
}

type UnnamedOptionsManager[T Entity[T]] struct{ factory OptionsFactory[T] }

func NewUnnamedOptionsManager[T Entity[T]](factory OptionsFactory[T]) *UnnamedOptionsManager[T] {
	return &UnnamedOptionsManager[T]{factory: factory}
}

func (m *UnnamedOptionsManager[T]) Value() T { return m.factory.Create() }

// OptionsMonitor / optionsMonitor is form-B piece #2: the impl is unexported
// and the constructor returns the INTERFACE, so form A cannot register it.
type OptionsMonitor[T Entity[T]] interface {
	CurrentValue() T
}

type optionsMonitor[T Entity[T]] struct {
	factory OptionsFactory[T]
	setups  []ConfigureOptions[T]
	cache   OptionsMonitorCache[T]
}

func NewOptionsMonitor[T Entity[T]](factory OptionsFactory[T], setups []ConfigureOptions[T], cache OptionsMonitorCache[T]) OptionsMonitor[T] {
	return &optionsMonitor[T]{factory: factory, setups: setups, cache: cache}
}

func (m *optionsMonitor[T]) CurrentValue() T { return m.cache.GetOrAdd(m.factory.Create) }

// OptionsFactory / optionsFactory is form-B piece #3 (transient): unexported
// impl, constructor returns the interface.
type OptionsFactory[T Entity[T]] interface {
	Create() T
}

type optionsFactory[T Entity[T]] struct{ setups []ConfigureOptions[T] }

func NewOptionsFactory[T Entity[T]](setups []ConfigureOptions[T]) OptionsFactory[T] {
	return &optionsFactory[T]{setups: setups}
}

func (f *optionsFactory[T]) Create() T {
	var zero T
	v := zero.New()
	for _, s := range f.setups {
		s.Configure(v)
	}
	return v
}

// OptionsMonitorCache / OptionsCache is form-A piece #4 — registered LAST in
// the helper body while the monitor's factory GetRequired-s it.
type OptionsMonitorCache[T Entity[T]] interface {
	GetOrAdd(create func() T) T
}

type OptionsCache[T Entity[T]] struct {
	value T
	set   bool
}

func NewOptionsCache[T Entity[T]]() *OptionsCache[T] { return &OptionsCache[T]{} }

func (c *OptionsCache[T]) GetOrAdd(create func() T) T {
	if !c.set {
		c.value, c.set = create(), true
	}
	return c.value
}

// AddOptions is the multi-statement per-service helper (Mechanism A, extended):
// four unkeyed markers — form A + form B mixed — closed per call type argument.
func AddOptions[T Entity[T]](r *godi.Registry) *godi.Registry {
	godi.AddSingleton[Options[T], UnnamedOptionsManager[T]](r)
	godi.AddSingletonFactory[OptionsMonitor[T]](r, func(p godi.Provider) OptionsMonitor[T] {
		return NewOptionsMonitor[T](
			godi.GetRequired[OptionsFactory[T]](p),
			godi.GetServices[ConfigureOptions[T]](p),
			godi.GetRequired[OptionsMonitorCache[T]](p),
		)
	})
	godi.AddTransientFactory[OptionsFactory[T]](r, func(p godi.Provider) OptionsFactory[T] {
		return NewOptionsFactory[T](godi.GetServices[ConfigureOptions[T]](p))
	})
	return godi.AddSingleton[OptionsMonitorCache[T], OptionsCache[T]](r)
}

// MyCfg / OtherCfg are the two closed configuration types: closing AddOptions
// over BOTH proves the two expansions stay independent (no cross-typing).
type MyCfg struct{ Name string }

func NewMyCfg() *MyCfg { return &MyCfg{Name: "my-default"} }

func (c *MyCfg) New() *MyCfg { return NewMyCfg() }

type OtherCfg struct{ Level int }

func NewOtherCfg() *OtherCfg { return &OtherCfg{Level: 3} }

func (c *OtherCfg) New() *OtherCfg { return NewOtherCfg() }

var (
	_ Entity[*MyCfg]    = (*MyCfg)(nil)
	_ Entity[*OtherCfg] = (*OtherCfg)(nil)
)

// init is an AD-7 tripwire. If static analysis ever EXECUTED this fixture the
// process would panic; go/packages only parses and type-checks it.
func init() {
	panic("AD-7 violation: optionshelper registration fixture was executed during static analysis")
}

// register holds the statically-analyzed markers: the four-piece helper closed
// over two distinct configuration types.
func register() {
	var r godi.Registry
	AddOptions[*MyCfg](&r)
	AddOptions[*OtherCfg](&r)
}
