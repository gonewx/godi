package runtime_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonewx/godi/runtime"
)

// countingCloser counts its Close calls; the counter is atomic so a test can
// read it after concurrent tracking without itself racing.
type countingCloser struct{ closes atomic.Int32 }

func (c *countingCloser) Close() error { c.closes.Add(1); return nil }

// TestDisposer_ConcurrentTrackKeepsEveryItem: 8 goroutines released at once
// each TrackAny 500 closers into one Disposer. Every item must be kept
// (Len == 4000) and CloseAll must close each exactly once.
func TestDisposer_ConcurrentTrackKeepsEveryItem(t *testing.T) {
	const workers, perWorker = 8, 500
	d := runtime.NewDisposer()
	items := make([][]*countingCloser, workers)
	for w := range items {
		items[w] = make([]*countingCloser, perWorker)
		for i := range items[w] {
			items[w][i] = &countingCloser{}
		}
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for _, c := range items[w] {
				d.TrackAny(c)
			}
		}()
	}
	close(start)
	wg.Wait()

	if n := d.Len(); n != workers*perWorker {
		t.Fatalf("Len() = %d, want %d", n, workers*perWorker)
	}
	if err := d.CloseAll(); err != nil {
		t.Fatalf("CloseAll: %v", err)
	}
	for w := range items {
		for i, c := range items[w] {
			if n := c.closes.Load(); n != 1 {
				t.Fatalf("item %d/%d closed %d times, want 1", w, i, n)
			}
		}
	}
}

// trackOnClose tracks next into d from inside its own Close, the shape of a
// disposable whose release resolves (and so tracks) another disposable.
type trackOnClose struct {
	d      *runtime.Disposer
	next   *countingCloser
	closes atomic.Int32
}

func (c *trackOnClose) Close() error {
	c.closes.Add(1)
	c.d.TrackAny(c.next)
	return nil
}

// TestDisposer_TrackDuringCloseAllDoesNotDeadlock: an item that tracks another
// disposable from inside its Close must not deadlock CloseAll. Items are
// released outside the disposer's lock, so the newly tracked item lands in the
// (already emptied) list: this CloseAll does not release it, a later one does.
func TestDisposer_TrackDuringCloseAllDoesNotDeadlock(t *testing.T) {
	d := runtime.NewDisposer()
	late := &countingCloser{}
	first := &trackOnClose{d: d, next: late}
	d.TrackAny(first)

	done := make(chan error, 1)
	go func() { done <- d.CloseAll() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("CloseAll: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CloseAll deadlocked when an item tracked during release")
	}
	if n := first.closes.Load(); n != 1 {
		t.Fatalf("first item closed %d times, want 1", n)
	}
	if n := d.Len(); n != 1 {
		t.Fatalf("Len() after CloseAll = %d, want 1 (the item tracked during release)", n)
	}
	if n := late.closes.Load(); n != 0 {
		t.Fatalf("item tracked during release closed %d times by the same CloseAll, want 0", n)
	}
	if err := d.CloseAll(); err != nil {
		t.Fatalf("second CloseAll: %v", err)
	}
	if n := late.closes.Load(); n != 1 {
		t.Fatalf("item tracked during release closed %d times after second CloseAll, want 1", n)
	}
}
