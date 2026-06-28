package runtime

import "io"

// Disposer tracks io.Closer instances and releases them in LIFO order.
// It is exported because generated containers (in a separate package)
// consume it (AD-2: generated --> runtime(disposer)).
type Disposer struct {
	items []io.Closer
}

func NewDisposer() *Disposer {
	return &Disposer{}
}

func (d *Disposer) Track(c io.Closer) {
	d.items = append(d.items, c)
}

func (d *Disposer) CloseAll() error {
	var first error
	for i := len(d.items) - 1; i >= 0; i-- {
		if err := d.items[i].Close(); err != nil && first == nil {
			first = err
		}
	}
	d.items = nil
	return first
}

func (d *Disposer) Len() int {
	return len(d.items)
}
