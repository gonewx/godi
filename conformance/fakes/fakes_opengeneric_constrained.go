package fakes

// --- Constrained open-generic fakes (Story 4.2) ------------------------------
//
// Go type constraints on per-service helpers provide compile-time guarantee of
// constraint satisfaction — dotnet's runtime constraint checks are replaced by
// the Go type system (AD-10). The generator is fully transparent to constraints.
//
// Constraint mapping (D-1):
//   dotnet "where T : PocoClass"      → PocoClassLike     (sealed, unexported marker)
//   dotnet "where T : IEnumerable"    → IterableLike      (exported Enumerate method, mirrors IEnumerable's PUBLIC contract — NOT sealed, unlike the other two)
//   dotnet "where T : AbstractClass"  → AbstractClassLike (sealed, unexported marker)

// ---------------------------------------------------------------------------
// Constraint interfaces (Go type constraints). PocoClassLike and AbstractClassLike
// are sealed via unexported markers; IterableLike is NOT sealed — it mirrors
// dotnet IEnumerable's public contract through an exported Enumerate method, so
// any type with Enumerate() satisfies it (acceptable: only this package's
// candidate types use it as a per-service helper constraint).
// ---------------------------------------------------------------------------

// PocoClassLike mirrors dotnet "where T : PocoClass".
type PocoClassLike interface {
	isPocoClass()
}

// IterableLike mirrors dotnet "where T : IEnumerable" through an exported
// Enumerate method (IEnumerable is a public contract). Unlike PocoClassLike /
// AbstractClassLike it is NOT sealed; this is intentional, not an oversight.
type IterableLike interface {
	Enumerate()
}

// AbstractClassLike mirrors dotnet "where T : AbstractClass".
type AbstractClassLike interface {
	isAbstractClass()
}

// ---------------------------------------------------------------------------
// Candidate service interfaces + concrete implementations
// ---------------------------------------------------------------------------

// IPocoClass is the candidate service type satisfying PocoClassLike.
type IPocoClass interface {
	PocoClassLike
}

// PocoClass is the canonical IPocoClass implementation.
type PocoClass struct{ _ int }

func (*PocoClass) isPocoClass() {}

// NewPocoClass is the form-A constructor (AD-8).
func NewPocoClass() *PocoClass { return &PocoClass{} }

// IClassImplementingIterable is the candidate service type satisfying IterableLike.
type IClassImplementingIterable interface {
	IterableLike
}

// ClassImplementingIterable is the canonical IClassImplementingIterable implementation.
type ClassImplementingIterable struct{ _ int }

func (*ClassImplementingIterable) Enumerate() {}

// NewClassImplementingIterable is the form-A constructor (AD-8).
func NewClassImplementingIterable() *ClassImplementingIterable {
	return &ClassImplementingIterable{}
}

// IClassInheritingAbstract is the candidate service type satisfying AbstractClassLike.
type IClassInheritingAbstract interface {
	AbstractClassLike
}

// ClassInheritingAbstract is the canonical IClassInheritingAbstract implementation.
type ClassInheritingAbstract struct{ _ int }

func (*ClassInheritingAbstract) isAbstractClass() {}

// NewClassInheritingAbstract is the form-A constructor (AD-8).
func NewClassInheritingAbstract() *ClassInheritingAbstract {
	return &ClassInheritingAbstract{}
}

// ---------------------------------------------------------------------------
// DISpec constrained implementations (3 — with constructor injection)
// ---------------------------------------------------------------------------

// ConstrainedFakeOpenGenericService mirrors dotnet's
// ConstrainedFakeOpenGenericService<T> where T : PocoClass.
type ConstrainedFakeOpenGenericService[T PocoClassLike] struct {
	value T
	_     int
}

func (*ConstrainedFakeOpenGenericService[T]) isFakeOpenGenericService() {}

func (f *ConstrainedFakeOpenGenericService[T]) Value() T { return f.value }

// NewConstrainedFakeOpenGenericService is the form-A constructor (AD-8).
func NewConstrainedFakeOpenGenericService[T PocoClassLike](value T) *ConstrainedFakeOpenGenericService[T] {
	return &ConstrainedFakeOpenGenericService[T]{value: value}
}

// ClassWithInterfaceConstraint mirrors dotnet's
// ClassWithInterfaceConstraint<T> where T : IEnumerable.
type ClassWithInterfaceConstraint[T IterableLike] struct {
	value T
	_     int
}

func (*ClassWithInterfaceConstraint[T]) isFakeOpenGenericService() {}

func (f *ClassWithInterfaceConstraint[T]) Value() T { return f.value }

// NewClassWithInterfaceConstraint is the form-A constructor (AD-8).
func NewClassWithInterfaceConstraint[T IterableLike](value T) *ClassWithInterfaceConstraint[T] {
	return &ClassWithInterfaceConstraint[T]{value: value}
}

// ClassWithAbstractClassConstraint mirrors dotnet's
// ClassWithAbstractClassConstraint<T> where T : AbstractClass.
type ClassWithAbstractClassConstraint[T AbstractClassLike] struct {
	value T
	_     int
}

func (*ClassWithAbstractClassConstraint[T]) isFakeOpenGenericService() {}

func (f *ClassWithAbstractClassConstraint[T]) Value() T { return f.value }

// NewClassWithAbstractClassConstraint is the form-A constructor (AD-8).
func NewClassWithAbstractClassConstraint[T AbstractClassLike](value T) *ClassWithAbstractClassConstraint[T] {
	return &ClassWithAbstractClassConstraint[T]{value: value}
}

// ---------------------------------------------------------------------------
// Structural constraint implementations (5 — compile verification, no SpecCase)
// Consumed by Epic 6 CallSiteFactoryTest.
// ---------------------------------------------------------------------------

// ClassWithClassConstraint mirrors dotnet "where T : class".
// Go has no class constraint; uses any. Difference noted.
type ClassWithClassConstraint[T any] struct{ _ int }

func (*ClassWithClassConstraint[T]) isFakeOpenGenericService() {}

func (f *ClassWithClassConstraint[T]) Value() T {
	var zero T
	return zero
}

// ClassWithStructConstraint mirrors dotnet "where T : struct".
// Go uses comparable as approximation. Difference noted.
type ClassWithStructConstraint[T comparable] struct{ _ int }

func (*ClassWithStructConstraint[T]) isFakeOpenGenericService() {}

func (f *ClassWithStructConstraint[T]) Value() T {
	var zero T
	return zero
}

// ClassWithNewConstraint mirrors dotnet "where T : new()".
// Go has no new() constraint; uses any. Difference noted.
type ClassWithNewConstraint[T any] struct{ _ int }

func (*ClassWithNewConstraint[T]) isFakeOpenGenericService() {}

func (f *ClassWithNewConstraint[T]) Value() T {
	var zero T
	return zero
}

// ClassWithNoConstraints is the unconstrained control group.
type ClassWithNoConstraints[T any] struct{ _ int }

func (*ClassWithNoConstraints[T]) isFakeOpenGenericService() {}

func (f *ClassWithNoConstraints[T]) Value() T {
	var zero T
	return zero
}

// ClassWithSelfReferencingConstraint mirrors dotnet "where T : IComparable<T>".
type ClassWithSelfReferencingConstraint[T interface{ CompareTo(T) int }] struct {
	value T
	_     int
}

func (*ClassWithSelfReferencingConstraint[T]) isFakeOpenGenericService() {}

func (f *ClassWithSelfReferencingConstraint[T]) Value() T { return f.value }

// NewClassWithSelfReferencingConstraint is the form-A constructor (AD-8).
func NewClassWithSelfReferencingConstraint[T interface{ CompareTo(T) int }](value T) *ClassWithSelfReferencingConstraint[T] {
	return &ClassWithSelfReferencingConstraint[T]{value: value}
}
