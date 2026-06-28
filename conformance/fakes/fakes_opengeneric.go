package fakes

// --- Minimal open-generic fakes (Story 2.5, Task 2) -------------------------
//
// IFakeOpenGenericService[T] and FakeOpenGenericService[T] are the Go port of
// dotnet's IFakeOpenGenericService<T> / FakeOpenGenericService<T>. Only the
// MINIMAL definition is ported here — no constraint classes, no open-generic
// registration mechanism (those belong to Epic 4).
//
// Story 2.5 uses these exclusively as CLOSED generic types
// (IFakeOpenGenericService[IFakeService]) to test ClosedGenericsWithIsService
// (AC7). In godi, a closed generic type is an ordinary registered type — Go's
// type system instantiates generics at compile time.

// IFakeOpenGenericService is a sealed generic interface mirroring dotnet's
// IFakeOpenGenericService<T>. The Value() accessor exposes the type parameter
// for assertion (prove the correct closed instance was resolved).
type IFakeOpenGenericService[T any] interface {
	isFakeOpenGenericService()
	Value() T
}

// FakeOpenGenericService is the canonical implementation of
// IFakeOpenGenericService[T]. The padding field ensures distinct heap
// allocations produce distinct pointer identities.
type FakeOpenGenericService[T any] struct {
	value T
	_     int
}

func (*FakeOpenGenericService[T]) isFakeOpenGenericService() {}

func (f *FakeOpenGenericService[T]) Value() T { return f.value }

// NewFakeOpenGenericService is the form-A constructor (AD-8 New{Impl}).
func NewFakeOpenGenericService[T any]() *FakeOpenGenericService[T] {
	return &FakeOpenGenericService[T]{}
}
