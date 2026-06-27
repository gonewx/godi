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

// IFakeConsumer is a service that DEPENDS on IFakeService — the Go port of the
// dotnet OuterService single-dependency slice (OuterServiceCanHaveOtherServices
// Injected). The full FakeOuterService (which also depends on IEnumerable) is
// Story 2.4; this slice carries just the one-singleton-dependency case Story 2.1
// needs. The sealed isFakeConsumer() ties implementations to this package.
//
// Dependency() exposes the injected IFakeService so a test can assert the
// dependency wired into the consumer is the SAME singleton the provider resolves
// directly (AD-8: exactly one constructor on the assembly path).
type IFakeConsumer interface {
	isFakeConsumer()
	Dependency() IFakeService
}

// FakeConsumer is the canonical IFakeConsumer implementation. It holds the
// IFakeService dependency injected through its New{Impl} constructor.
type FakeConsumer struct {
	dep IFakeService
}

func (*FakeConsumer) isFakeConsumer() {}

// Dependency returns the injected IFakeService — the seam tests use to prove the
// dependency is the shared singleton (its constructor ran once, AD-8).
func (c *FakeConsumer) Dependency() IFakeService { return c.dep }

// NewFakeConsumer is the form-A New{Impl} constructor for FakeConsumer. Its
// IFakeService parameter is the dependency the generator discovers and wires as
// r.f_FakeConsumer = fakes.NewFakeConsumer(r.f_FakeService); the runtime factory
// wraps it in a form-B closure that resolves the dependency through the provider.
func NewFakeConsumer(dep IFakeService) *FakeConsumer { return &FakeConsumer{dep: dep} }
