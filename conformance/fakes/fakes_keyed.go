package fakes

// --- Keyed service fakes (Story 3-1, Task 7) --------------------------------
//
// IKeyedService is the marker interface for keyed DI resolution tests (Story 3-1),
// ported from dotnet's KeyedDependencyInjectionSpecificationTests test-local IService.
// Two implementations (KeyedServiceOne, KeyedServiceTwo) are registered under
// distinct keys to test keyed resolve, keyed collection, singleton/transient
// lifetime sharing, and cross-provider differential equivalence (AD-15/AD-17).
//
// These fakes are separate from IFakeService to keep keyed and unkeyed test
// namespaces independent — keyed tests must verify that GetService does NOT
// return keyed services (AC2, D-3 independent namespace).

// IKeyedService is the sealed marker interface for keyed resolution tests.
type IKeyedService interface {
	isKeyedService()
}

// KeyedServiceOne is the first keyed implementation. The padding field ensures
// distinct heap allocations produce distinct pointer identities (Go zero-sized
// structs may share one address).
type KeyedServiceOne struct{ _ int }

func (*KeyedServiceOne) isKeyedService() {}

func NewKeyedServiceOne() *KeyedServiceOne { return &KeyedServiceOne{} }

// KeyedServiceTwo is the second keyed implementation, used to verify distinct-key
// resolution (AC1) and keyed collection registration-order preservation (AC3).
type KeyedServiceTwo struct{ _ int }

func (*KeyedServiceTwo) isKeyedService() {}

func NewKeyedServiceTwo() *KeyedServiceTwo { return &KeyedServiceTwo{} }
