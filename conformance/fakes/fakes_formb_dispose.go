package fakes

// --- Form-B disposable fakes (eval fix A3) ----------------------------------
//
// Registered in conformance/codegen through form-B factories, whose product
// type the generator cannot inspect: the generated container must still
// release them, by asserting io.Closer at run time (Disposer.TrackAny).

// IFakeFormBDisposableSingleton is a form-B singleton whose product is an
// io.Closer.
type IFakeFormBDisposableSingleton interface {
	CloseCount() int
}

// IFakeFormBDisposableScoped is a form-B scoped service whose product is an
// io.Closer.
type IFakeFormBDisposableScoped interface {
	CloseCount() int
}

// FakeFormBCloser counts its Close calls.
type FakeFormBCloser struct {
	closes int
}

func (f *FakeFormBCloser) Close() error {
	f.closes++
	return nil
}

func (f *FakeFormBCloser) CloseCount() int { return f.closes }

// NewFakeFormBCloser is called from the form-B factories.
func NewFakeFormBCloser() *FakeFormBCloser { return &FakeFormBCloser{} }
