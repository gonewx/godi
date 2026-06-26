package container

import "io"

type disposable struct {
	label string
	c     io.Closer
}

// disposer tracks io.Closer instances and disposes them LIFO, mirroring
// dotnet's scope disposal order. No reflection.
type disposer struct {
	items []disposable
	done  []string
}

func newDisposer() *disposer { return &disposer{} }

func (d *disposer) track(label string, c io.Closer) {
	d.items = append(d.items, disposable{label, c})
}

// closeAll closes tracked disposables in last-in-first-out order.
func (d *disposer) closeAll() error {
	var first error
	for i := len(d.items) - 1; i >= 0; i-- {
		if err := d.items[i].c.Close(); err != nil && first == nil {
			first = err
		}
		d.done = append(d.done, d.items[i].label)
	}
	d.items = nil
	return first
}

func (d *disposer) order() []string { return append([]string(nil), d.done...) }
