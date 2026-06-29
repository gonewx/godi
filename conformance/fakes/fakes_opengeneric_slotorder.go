package fakes

// --- Slot-order open/closed fakes (Story 4.3) --------------------------------
//
// These fixtures support touch-points 6/7/8 of the 9 open-generic contact points:
// open/closed PRIORITY (single resolve) and ENUMERABLE slot ORDER for generic
// services. The CORE ARCHITECTURE POINT (4-3 Dev Notes 本质 + D-1..D-5): godi has
// NO open/closed distinction at the generator level (AD-4/AD-10: no
// MakeGenericType — the generator sees only CLOSED registrations). dotnet's
// "exact closed preferred over open generic" is an order-INSENSITIVE runtime rule;
// godi has no such rule, so priority/slot order are expressed by REGISTRATION
// ORDER (single resolve = last-wins, enumerable = registration-order slots).
//
// This file ports two DEDICATED element types (IGenSlotA/IGenSlotB) and two
// closed-SPECIFIC generic impls (Closed/Instance, NO injection). The element
// types are dedicated to 4.3 to isolate the 4.1/4.2 slots — reusing IFakeService /
// IPocoClass etc. would pollute their counts and break 4.1's negative
// unmaterialized case (D-1, 关键禁忌).
//
// dotnet mapping (D-1):
//
//	PocoClass (closed type arg T)                       → IGenSlotA / IGenSlotB
//	FakeService : IFakeOpenGenericService<PocoClass>    → ClosedFakeOpenGenericService[T]
//	  (closed-specific, Value default null, NO injection)  (closed preferred + tp8 closed#1)
//	new FakeOpenGenericService<PocoClass>(null)         → InstanceFakeOpenGenericService[T]
//	  (pre-built instance slot)                            (tp8 instance slot; DISTINCT impl type per D-4)
//
// Why generic (not non-generic) closed-specific impls? dotnet's FakeService is a
// NON-generic class implementing the closed IFakeOpenGenericService<PocoClass>.
// godi could write a non-generic impl, but "non-generic impl of a closed generic
// interface" is an UNVERIFIED generator shape. Generic impls registered via direct
// closed form-A (AddTransient[IFakeOpenGenericService[IGenSlotA],
// ClosedFakeOpenGenericService[IGenSlotA]]) reuse the PROVEN path (3-4 gen-key +
// 4.2 constrained impls): observation-equivalent (distinct dynamic type), lowest
// risk (D-1/D-2).

// ---------------------------------------------------------------------------
// Element types (sealed markers) — the closed type arg T for the slot-order
// generics. They serve as the injected T value for the open-produced impl
// (FakeOpenGenericService[T] from Story 4.1). Single-registration, registered as
// singletons (a DISTINCT service type, no slot-order concern of their own).
// ---------------------------------------------------------------------------

// IGenSlotA is a sealed marker element type, dedicated to Story 4.3 (touch-point
// 6/7). It mirrors dotnet's PocoClass as the closed type argument.
type IGenSlotA interface {
	isGenSlotA()
}

// GenSlotA is the canonical IGenSlotA implementation. The padding field keeps
// distinct allocations at distinct pointer identities.
type GenSlotA struct{ _ int }

func (*GenSlotA) isGenSlotA() {}

// NewGenSlotA is the form-A constructor (AD-8 New{Impl}).
func NewGenSlotA() *GenSlotA { return &GenSlotA{} }

// IGenSlotB is a sealed marker element type, dedicated to Story 4.3 (touch-point
// 8, the mixed open/closed enumerable). Distinct from IGenSlotA so the two
// touch-point groups never share a slot.
type IGenSlotB interface {
	isGenSlotB()
}

// GenSlotB is the canonical IGenSlotB implementation.
type GenSlotB struct{ _ int }

func (*GenSlotB) isGenSlotB() {}

// NewGenSlotB is the form-A constructor (AD-8 New{Impl}).
func NewGenSlotB() *GenSlotB { return &GenSlotB{} }

// ---------------------------------------------------------------------------
// Closed-specific generic impls (NO injection). Each implements the sealed
// IFakeOpenGenericService[T] (Story 4.1) via isFakeOpenGenericService() + Value().
// Value() returns the zero value of T (mirrors dotnet FakeService.Value, default
// null) — they take NO constructor dependency, distinguishing them from the
// injected FakeOpenGenericService[T] (4.1) so "closed preferred" and the mixed
// enumerable slots are observable by DYNAMIC TYPE. The _ int padding guarantees
// distinct pointer identity.
// ---------------------------------------------------------------------------

// ClosedFakeOpenGenericService is the closed-SPECIFIC impl that "wins" single
// resolution when registered LAST (touch-point 6/7, D-5) and occupies enumerable
// slot[0] (closed#1) for touch-point 8. It is the Go equivalent of dotnet's
// non-generic FakeService — observation-equivalent via its distinct dynamic type,
// implemented as a generic to reuse the proven direct-closed form-A path (D-1).
type ClosedFakeOpenGenericService[T any] struct{ _ int }

func (*ClosedFakeOpenGenericService[T]) isFakeOpenGenericService() {}

// Value returns the zero value of T (no injection — mirrors dotnet FakeService's
// default-null Value).
func (f *ClosedFakeOpenGenericService[T]) Value() T {
	var zero T
	return zero
}

// NewClosedFakeOpenGenericService is the form-A constructor (AD-8). It takes NO
// dependency, so the generator reads an empty parameter list → Deps=[] (D-2).
func NewClosedFakeOpenGenericService[T any]() *ClosedFakeOpenGenericService[T] {
	return &ClosedFakeOpenGenericService[T]{}
}

// InstanceFakeOpenGenericService occupies the INSTANCE slot (enumerable slot[2])
// for touch-point 8. dotnet uses a pre-built instance of the SAME
// FakeOpenGenericService<PocoClass> type, distinguished by instance identity;
// godi cannot reuse the open-produced impl type (two form-A registrations of the
// SAME impl type collide on serviceName — a Go mechanism wall, D-4) and R4 forbids
// cross-provider instance identity, so a DISTINCT impl type is used and observed
// by dynamic type + slot position.
type InstanceFakeOpenGenericService[T any] struct{ _ int }

func (*InstanceFakeOpenGenericService[T]) isFakeOpenGenericService() {}

// Value returns the zero value of T (no injection).
func (f *InstanceFakeOpenGenericService[T]) Value() T {
	var zero T
	return zero
}

// NewInstanceFakeOpenGenericService is the form-A constructor (AD-8), no
// dependency → Deps=[] (D-2).
func NewInstanceFakeOpenGenericService[T any]() *InstanceFakeOpenGenericService[T] {
	return &InstanceFakeOpenGenericService[T]{}
}
