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

// --- Story 3-2: Keyed injection fakes ----------------------------------------
//
// These fakes exercise keyed dependency injection patterns (AC2-AC5). Each
// corresponds to a dotnet KeyedDependencyInjectionSpecificationTests inner class:
//   - IKeyedServiceWithDep → [FromKeyedServices] via form-B closure (AC2)
//   - IKeyedServiceWithKey → [ServiceKey] via closure capture (AC3)
//   - IKeyedServiceWithProvider → IServiceProvider injection via closure param (AC5)

// IKeyedServiceWithDep is a keyed service that depends on another keyed service.
// The Go equivalent of dotnet's [FromKeyedServices("key")] attribute is a form-B
// factory closure calling p.GetKeyedService (D-1 from Story 3-2).
type IKeyedServiceWithDep interface {
	Dep() IKeyedService
}

// KeyedServiceWithDep implements IKeyedServiceWithDep. The dep field stores the
// keyed dependency injected via the form-B closure.
type KeyedServiceWithDep struct {
	dep IKeyedService
	_   int
}

func (s *KeyedServiceWithDep) Dep() IKeyedService { return s.dep }

func NewKeyedServiceWithDep(dep IKeyedService) *KeyedServiceWithDep {
	return &KeyedServiceWithDep{dep: dep}
}

// IKeyedServiceWithKey is a keyed service that receives its own registration key.
// The Go equivalent of dotnet's [ServiceKey] attribute is closure capture of the
// key variable (D-2 from Story 3-2).
type IKeyedServiceWithKey interface {
	Key() any
}

// KeyedServiceWithKey implements IKeyedServiceWithKey. The key field stores the
// registration key injected via closure capture.
type KeyedServiceWithKey struct {
	key any
	_   int
}

func (s *KeyedServiceWithKey) Key() any { return s.key }

func NewKeyedServiceWithKey(key any) *KeyedServiceWithKey {
	return &KeyedServiceWithKey{key: key}
}

// IKeyedServiceWithProvider is a keyed service that receives the provider itself.
// The Go equivalent of dotnet's IServiceProvider constructor injection is the
// form-B closure parameter p (D-3 from Story 3-2). The provider is stored as any
// to avoid importing godi in the fakes package.
type IKeyedServiceWithProvider interface {
	ServiceProvider() any
}

// KeyedServiceWithProvider implements IKeyedServiceWithProvider. The provider
// field stores the injected provider reference as any.
type KeyedServiceWithProvider struct {
	provider any
	_        int
}

func (s *KeyedServiceWithProvider) ServiceProvider() any { return s.provider }

func NewKeyedServiceWithProvider(p any) *KeyedServiceWithProvider {
	return &KeyedServiceWithProvider{provider: p}
}

// --- Story 3-3: AnyKey / null-key boundary fakes -----------------------------
//
// IAnyKeyService is a DEDICATED marker interface for the AnyKey-query
// (GetKeyedServices(AnyKey)) and null-key routing tests (AD-17, Story 3-3). It is
// intentionally separate from IKeyedService so the AnyKey test registration set
// stays small and decoupled from the 3-1/3-2 keys: GetKeyedServices(
// reflect.TypeFor[IAnyKeyService](), AnyKey) returns EXACTLY the AnyKey test set
// in registration order in BOTH providers, keeping the cross-provider
// differential-equivalence assertion (AD-15) clean and order-exact.
//
// Two type-distinguishable impls (A, B) let the interleaved-key ordering test
// (alpha→A, beta→B, alpha→A) assert registration order [A, B, A] versus the wrong
// grouped-by-key order [A, A, B] using dynamic type names alone — no per-instance
// identity tag is needed (ResolveKeyedServicesAnyKeyOrdering, AC2).
type IAnyKeyService interface{ isAnyKeyService() }

// AnyKeyServiceA is the first AnyKey impl. The padding field forces distinct heap
// allocations so pointer identities differ (Go zero-sized structs may share one
// address).
type AnyKeyServiceA struct{ _ int }

func (*AnyKeyServiceA) isAnyKeyService() {}

func NewAnyKeyServiceA() *AnyKeyServiceA { return &AnyKeyServiceA{} }

// AnyKeyServiceB is the second AnyKey impl, the type-distinguishable middle
// element in the interleaved-key registration-order test (AC2).
type AnyKeyServiceB struct{ _ int }

func (*AnyKeyServiceB) isAnyKeyService() {}

func NewAnyKeyServiceB() *AnyKeyServiceB { return &AnyKeyServiceB{} }
