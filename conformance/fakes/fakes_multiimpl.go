package fakes

// --- Multi-implementation fakes (Story 2.4, Task 5) --------------------------
//
// IEnumerable[T] multi-implementation resolution fixtures. These mirror dotnet's
// IFakeMultipleService / FakeOneMultipleService / FakeTwoMultipleService and
// IFakeOuterService / FakeOuterService, adapted to Go sealed interfaces.
//
// Go difference (D-5): IFakeMultipleService does NOT embed IFakeService. In
// dotnet, IFakeMultipleService : IFakeService is inheritance for polymorphism;
// godi's IEnumerable tests resolve at the IFakeMultipleService level only —
// no upcast needed. Each interface has an independent sealed method.

// IFakeMultipleService is the marker interface for multi-implementation
// resolution tests. Two implementations (FakeOneMultipleService,
// FakeTwoMultipleService) are registered under this single interface type to
// test collection resolution and registration-order preservation (CAP-4).
type IFakeMultipleService interface {
	isFakeMultipleService()
}

// FakeOneMultipleService is the first implementation of IFakeMultipleService.
// The padding field ensures distinct heap allocations produce distinct pointer
// identities (same lesson as FakeScopedService).
type FakeOneMultipleService struct{ _ int }

func (*FakeOneMultipleService) isFakeMultipleService() {}

// NewFakeOneMultipleService is the form-A constructor (AD-8 New{Impl}).
func NewFakeOneMultipleService() *FakeOneMultipleService {
	return &FakeOneMultipleService{}
}

// FakeTwoMultipleService is the second implementation of IFakeMultipleService.
type FakeTwoMultipleService struct{ _ int }

func (*FakeTwoMultipleService) isFakeMultipleService() {}

// NewFakeTwoMultipleService is the form-A constructor (AD-8 New{Impl}).
func NewFakeTwoMultipleService() *FakeTwoMultipleService {
	return &FakeTwoMultipleService{}
}

// IFakeOuterService is a service that depends on BOTH a single IFakeService AND
// a collection of IFakeMultipleService (slice dependency). This tests
// OuterServiceCanHaveOtherServicesInjected (DISpec line 261, AC6).
type IFakeOuterService interface {
	isFakeOuterService()
	SingleService() IFakeService
	MultipleServices() []IFakeMultipleService
}

// FakeOuterService is the canonical IFakeOuterService implementation. Its
// constructor takes IFakeService + []IFakeMultipleService — the slice parameter
// is the Go equivalent of dotnet's IEnumerable<IFakeMultipleService> constructor
// injection (D-4 slice dependency).
type FakeOuterService struct {
	single   IFakeService
	multiple []IFakeMultipleService
}

func (*FakeOuterService) isFakeOuterService() {}

func (f *FakeOuterService) SingleService() IFakeService {
	return f.single
}

func (f *FakeOuterService) MultipleServices() []IFakeMultipleService {
	return f.multiple
}

// NewFakeOuterService is the form-A constructor (AD-8 New{Impl}). The
// []IFakeMultipleService parameter is the slice dependency that the generator
// must recognize and emit as a collected slice literal (D-4).
func NewFakeOuterService(svc IFakeService, multi []IFakeMultipleService) *FakeOuterService {
	return &FakeOuterService{single: svc, multiple: multi}
}
