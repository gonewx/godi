package fakes

// --- Extensions "final set equivalence" fakes (code review of story-6.4, 2026-07-01) --
//
// Dedicated fixtures for the AD-15/D-8 "final set equivalence" obligation axis:
// runtime reaches a final registration set via collection mutation (Replace /
// RemoveAll); codegen reaches the SAME logical final set by declaring it
// directly through the static DSL. These fakes are deliberately fresh (not
// reused from fakes_multiimpl.go) so wiring them into the codegen registration
// source cannot perturb the registration count/order already asserted by the
// existing resolution-extension tests over IFakeMultipleService.

// IFakeReplaceFinalService is the marker interface for the Replace "final set"
// scenario: runtime registers One then Two then Replace(One) (removes the
// first ServiceType match, appends the new one) -> final = [Two, One-new].
// codegen declares that same final order directly.
type IFakeReplaceFinalService interface {
	isFakeReplaceFinalService()
}

// FakeReplaceFinalOne is the first implementation.
type FakeReplaceFinalOne struct{ _ int }

func (*FakeReplaceFinalOne) isFakeReplaceFinalService() {}

// NewFakeReplaceFinalOne is the form-A constructor (AD-8 New{Impl}).
func NewFakeReplaceFinalOne() *FakeReplaceFinalOne { return &FakeReplaceFinalOne{} }

// FakeReplaceFinalTwo is the second implementation.
type FakeReplaceFinalTwo struct{ _ int }

func (*FakeReplaceFinalTwo) isFakeReplaceFinalService() {}

// NewFakeReplaceFinalTwo is the form-A constructor (AD-8 New{Impl}).
func NewFakeReplaceFinalTwo() *FakeReplaceFinalTwo { return &FakeReplaceFinalTwo{} }

// IFakeRemovedService is the marker interface for the RemoveAll "final set"
// scenario: runtime registers it then calls RemoveAll -> final = empty ->
// IsService false. This type is deliberately registered NOWHERE in
// conformance/codegen/registration.go, so the generated container's
// IsService for it is naturally false — the SAME final state runtime reaches
// via mutation, without needing codegen to declare an "absence".
type IFakeRemovedService interface {
	isFakeRemovedService()
}

// FakeRemovedService is the sole implementation, used only on the runtime leg
// (registered, then removed via RemoveAll before Build).
type FakeRemovedService struct{ _ int }

func (*FakeRemovedService) isFakeRemovedService() {}

// NewFakeRemovedService is the form-A constructor (AD-8 New{Impl}).
func NewFakeRemovedService() *FakeRemovedService { return &FakeRemovedService{} }

// IFakeInstanceService is the marker interface for the instance-registration
// "result obligation" axis (D-8): runtime uses AddSingletonInstance with a
// pre-constructed value; codegen has no raw-instance DSL entry (D-4) so it
// declares the equivalent via a form-B factory. Cross-provider equivalence is
// asserted on OBSERVABLE resolve results (dynamic type / IsService), never on
// pointer identity (R4).
type IFakeInstanceService interface {
	isFakeInstanceService()
}

// FakeInstanceService is the sole implementation.
type FakeInstanceService struct{ _ int }

func (*FakeInstanceService) isFakeInstanceService() {}

// NewFakeInstanceService is the form-A constructor (AD-8 New{Impl}).
func NewFakeInstanceService() *FakeInstanceService { return &FakeInstanceService{} }
