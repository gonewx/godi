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

// IFakeScopedService is the scoped-lifetime fixture used by Story 2.2 lifetime
// conformance tests. Each scope creates its own instance; root acts as root
// scope (D-2).
type IFakeScopedService interface {
	isFakeScopedService()
}

// FakeScopedService has a padding field to ensure distinct heap allocations
// produce distinct pointer identities (Go zero-sized structs may share one
// address).
type FakeScopedService struct{ _ int }

func (*FakeScopedService) isFakeScopedService() {}

func NewFakeScopedService() *FakeScopedService { return &FakeScopedService{} }

// IFakeTransientService is the transient-lifetime fixture used by Story 2.2.
// Every resolution creates a new instance, never cached (AD-14).
type IFakeTransientService interface {
	isFakeTransientService()
}

type FakeTransientService struct{ _ int }

func (*FakeTransientService) isFakeTransientService() {}

func NewFakeTransientService() *FakeTransientService { return &FakeTransientService{} }

// IFakeTransientWithScopedDep is the AC2 fixture: a transient service that
// depends on a scoped service. Tests verify that two transient resolutions
// within the same scope yield different transient instances but share the
// same scoped dependency.
type IFakeTransientWithScopedDep interface {
	isFakeTransientWithScopedDep()
	ScopedDep() IFakeScopedService
}

type FakeTransientWithScopedDep struct {
	dep IFakeScopedService
	_   int
}

func (*FakeTransientWithScopedDep) isFakeTransientWithScopedDep()   {}
func (f *FakeTransientWithScopedDep) ScopedDep() IFakeScopedService { return f.dep }

func NewFakeTransientWithScopedDep(dep IFakeScopedService) *FakeTransientWithScopedDep {
	return &FakeTransientWithScopedDep{dep: dep}
}

// --- Disposable fakes (Story 2.3) -------------------------------------------

// IFakeDisposableSingleton is a disposable singleton — implements io.Closer.
type IFakeDisposableSingleton interface {
	isFakeDisposableSingleton()
	IsDisposed() bool
}

// IFakeDisposableScoped is a disposable scoped service.
type IFakeDisposableScoped interface {
	isFakeDisposableScoped()
	IsDisposed() bool
}

// IFakeDisposableTransient is a disposable transient service.
type IFakeDisposableTransient interface {
	isFakeDisposableTransient()
	IsDisposed() bool
}

// DisposeRecorder collects labels in Close() call order (D-7).
// Tests use it to verify LIFO release order as a provider-stable observable.
type DisposeRecorder struct {
	labels []string
}

func NewDisposeRecorder() *DisposeRecorder {
	return &DisposeRecorder{}
}

func (r *DisposeRecorder) Record(label string) {
	r.labels = append(r.labels, label)
}

func (r *DisposeRecorder) Labels() []string {
	return r.labels
}

// FakeDisposableSingleton is the singleton disposable implementation.
type FakeDisposableSingleton struct {
	disposed bool
	recorder *DisposeRecorder
	label    string
	_        int
}

func (f *FakeDisposableSingleton) Close() error {
	f.disposed = true
	if f.recorder != nil {
		f.recorder.Record(f.label)
	}
	return nil
}
func (f *FakeDisposableSingleton) IsDisposed() bool         { return f.disposed }
func (*FakeDisposableSingleton) isFakeDisposableSingleton() {}
func (f *FakeDisposableSingleton) SetRecorder(r *DisposeRecorder, label string) {
	f.recorder = r
	f.label = label
}

func NewFakeDisposableSingleton() *FakeDisposableSingleton { return &FakeDisposableSingleton{} }

// FakeDisposableScoped is the scoped disposable implementation.
type FakeDisposableScoped struct {
	disposed bool
	recorder *DisposeRecorder
	label    string
	_        int
}

func (f *FakeDisposableScoped) Close() error {
	f.disposed = true
	if f.recorder != nil {
		f.recorder.Record(f.label)
	}
	return nil
}
func (f *FakeDisposableScoped) IsDisposed() bool      { return f.disposed }
func (*FakeDisposableScoped) isFakeDisposableScoped() {}
func (f *FakeDisposableScoped) SetRecorder(r *DisposeRecorder, label string) {
	f.recorder = r
	f.label = label
}

func NewFakeDisposableScoped() *FakeDisposableScoped { return &FakeDisposableScoped{} }

// FakeDisposableTransient is the transient disposable implementation.
type FakeDisposableTransient struct {
	disposed bool
	recorder *DisposeRecorder
	label    string
	_        int
}

func (f *FakeDisposableTransient) Close() error {
	f.disposed = true
	if f.recorder != nil {
		f.recorder.Record(f.label)
	}
	return nil
}
func (f *FakeDisposableTransient) IsDisposed() bool         { return f.disposed }
func (*FakeDisposableTransient) isFakeDisposableTransient() {}
func (f *FakeDisposableTransient) SetRecorder(r *DisposeRecorder, label string) {
	f.recorder = r
	f.label = label
}

func NewFakeDisposableTransient() *FakeDisposableTransient { return &FakeDisposableTransient{} }
