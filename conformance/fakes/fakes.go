package fakes

// IFakeService is the Go equivalent of dotnet's empty marker interface
// `IFakeService { }`. Because Go's empty interface (`any`) is satisfied by
// every type, an empty interface here would be meaningless: every resolve
// would type-assert true. The unexported sealed method isFakeService() ties
// implementations to this package — only types declared here can satisfy
// IFakeService, mirroring the dotnet marker's intent (D1).
type IFakeService interface {
	isFakeService()
}

// FakeService is the canonical implementation of IFakeService used by the
// Walking Skeleton's single-service slice. The full Value/Disposed surface
// from the dotnet fixture migrates incrementally as Epic 2 ports
// construction-injection and dispose cases.
type FakeService struct{}

func (*FakeService) isFakeService() {}

// NewFakeService is the form-A constructor target (AD-8 `New{Impl}` naming).
// The codegen emit calls it through Ctor="fakes.NewFakeService" to wire the
// singleton, and the runtime factory wraps it in a form-B closure.
func NewFakeService() *FakeService { return &FakeService{} }
