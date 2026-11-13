package fakes

import "context"

// --- Async disposable fakes (Story 6.3) -------------------------------------
//
// These fakes exist ONLY here (new types), never modifying the Story 2.3
// sync-only fakes above — mutating those to implement CloseAsync would turn them
// async-disposable and drift the existing goldens/containers (D-notes: reuse, do
// not modify). They map dotnet's AsyncDisposable / SyncAsyncDisposable test
// fixtures.

// AsyncDisposeRecorder collects labels in Close()/CloseAsync() call order — the
// async analogue of DisposeRecorder (D-7), used as the provider-stable
// observable for LIFO release order across BOTH providers (R4: never pointer
// identity across providers).
type AsyncDisposeRecorder struct {
	labels []string
}

func NewAsyncDisposeRecorder() *AsyncDisposeRecorder {
	return &AsyncDisposeRecorder{}
}

func (r *AsyncDisposeRecorder) Record(label string) {
	r.labels = append(r.labels, label)
}

func (r *AsyncDisposeRecorder) Labels() []string {
	return r.labels
}

// FakeAsyncOnlyDisposable implements ONLY godi.AsyncDisposable (CloseAsync) and
// deliberately does NOT implement io.Closer — the "async-only" row of the D-4
// matrix, mapping dotnet's AsyncDisposable fake. Sync release must error on it
// (AC3); CloseAsync releases it. It is registered by its POINTER type
// (*FakeAsyncOnlyDisposable) so the codegen field is concrete; the emitted
// container tracks it through Disposer.TrackAny, which dispatches on the dynamic
// type.
type FakeAsyncOnlyDisposable struct {
	disposed bool
	recorder *AsyncDisposeRecorder
	label    string
	_        int // distinct pointer identity (2.2 zero-size lesson)
}

// NewFakeAsyncOnlyDisposable is the form-A New{Impl} constructor.
func NewFakeAsyncOnlyDisposable() *FakeAsyncOnlyDisposable { return &FakeAsyncOnlyDisposable{} }

func (f *FakeAsyncOnlyDisposable) CloseAsync(ctx context.Context) error {
	f.disposed = true
	if f.recorder != nil {
		f.recorder.Record(f.label)
	}
	return nil
}

func (f *FakeAsyncOnlyDisposable) IsDisposed() bool { return f.disposed }

func (f *FakeAsyncOnlyDisposable) SetRecorder(r *AsyncDisposeRecorder, label string) {
	f.recorder = r
	f.label = label
}

// FakeSyncAsyncDisposable implements BOTH io.Closer and godi.AsyncDisposable —
// dotnet's SyncAsyncDisposable, the prefer-async witness (D-4). closedVia
// records which method actually fired: under CloseAsync it must be "async"
// (prefer-async), under sync Close it is "sync".
type FakeSyncAsyncDisposable struct {
	disposed  bool
	closedVia string
	recorder  *AsyncDisposeRecorder
	label     string
	_         int
}

// NewFakeSyncAsyncDisposable is the form-A New{Impl} constructor.
func NewFakeSyncAsyncDisposable() *FakeSyncAsyncDisposable { return &FakeSyncAsyncDisposable{} }

func (f *FakeSyncAsyncDisposable) Close() error {
	f.disposed = true
	f.closedVia = "sync"
	if f.recorder != nil {
		f.recorder.Record(f.label)
	}
	return nil
}

func (f *FakeSyncAsyncDisposable) CloseAsync(ctx context.Context) error {
	f.disposed = true
	f.closedVia = "async"
	if f.recorder != nil {
		f.recorder.Record(f.label)
	}
	return nil
}

func (f *FakeSyncAsyncDisposable) IsDisposed() bool { return f.disposed }

func (f *FakeSyncAsyncDisposable) ClosedVia() string { return f.closedVia }

func (f *FakeSyncAsyncDisposable) SetRecorder(r *AsyncDisposeRecorder, label string) {
	f.recorder = r
	f.label = label
}
