package fakes

// --- Open-generic fakes (Story 4.1, Task 1) ---------------------------------
//
// IFakeOpenGenericService[T] and FakeOpenGenericService[T] are the Go port of
// dotnet's IFakeOpenGenericService<T> / FakeOpenGenericService<T>. Story 2.5
// ported the MINIMAL no-arg shape; Story 4.1 upgrades the constructor to inject
// `value T` — aligning with dotnet's FakeOpenGenericService<TVal>(TVal value) so
// the closed instance's Value() is the injected dependency
// (OpenGenericServicesCanBeResolved, AC2). Constraint classes
// (ConstrainedFakeOpenGenericService / ClassWith*Constraint) remain UNPORTED —
// they belong to Story 4.2.
//
// In godi a closed generic type (IFakeOpenGenericService[IFakeService]) is an
// ordinary registered type — Go instantiates generics at compile time, so the
// closed constructor's `value T` parameter becomes a concrete dependency the
// form-A pipeline discovers and wires (AD-4/AD-10: no runtime MakeGenericType).

// IFakeOpenGenericService is a sealed generic interface mirroring dotnet's
// IFakeOpenGenericService<T>. The Value() accessor exposes the injected type
// parameter for assertion (prove the correct closed instance — and its injected
// dependency — was resolved).
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

// NewFakeOpenGenericService is the form-A constructor (AD-8 New{Impl}). It
// injects `value T` — aligning with dotnet's FakeOpenGenericService<TVal>(TVal
// value): when the generic is closed over a registered service type, the
// generator's form-A discovery instantiates this signature with the closed type
// args, reads the concrete `value <T>` parameter, and wires the registered
// dependency (Story 4.1 AC2).
func NewFakeOpenGenericService[T any](value T) *FakeOpenGenericService[T] {
	return &FakeOpenGenericService[T]{value: value}
}
