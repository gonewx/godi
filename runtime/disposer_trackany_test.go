package runtime_test

import (
	"context"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// trackAnyCloser implements only io.Closer.
type trackAnyCloser struct{ closes int }

func (c *trackAnyCloser) Close() error { c.closes++; return nil }

// trackAnyAsyncOnly implements only godi.AsyncDisposable.
type trackAnyAsyncOnly struct{ closes int }

func (c *trackAnyAsyncOnly) CloseAsync(context.Context) error { c.closes++; return nil }

// trackAnyPlain implements neither disposer interface.
type trackAnyPlain struct{ _ int }

// TestDisposer_TrackAny: TrackAny dispatches on the dynamic type — an io.Closer
// is released by CloseAll, an async-only value by CloseAllAsync, and a value
// that is neither (or nil) is not tracked at all. Generated containers call it
// for form-B products whose disposability is unknown at generation time.
func TestDisposer_TrackAny(t *testing.T) {
	t.Run("io.Closer is released by CloseAll", func(t *testing.T) {
		d := runtime.NewDisposer()
		c := &trackAnyCloser{}
		d.TrackAny(c)
		testutil.Equal(t, d.Len(), 1)
		testutil.NoError(t, d.CloseAll())
		testutil.Equal(t, c.closes, 1)
	})
	t.Run("async-only is released by CloseAllAsync", func(t *testing.T) {
		d := runtime.NewDisposer()
		a := &trackAnyAsyncOnly{}
		d.TrackAny(a)
		testutil.Equal(t, d.Len(), 1)
		testutil.NoError(t, d.CloseAllAsync(context.Background()))
		testutil.Equal(t, a.closes, 1)
	})
	t.Run("neither is not tracked", func(t *testing.T) {
		d := runtime.NewDisposer()
		d.TrackAny(&trackAnyPlain{})
		testutil.Equal(t, d.Len(), 0)
	})
	t.Run("nil is not tracked", func(t *testing.T) {
		d := runtime.NewDisposer()
		d.TrackAny(nil)
		testutil.Equal(t, d.Len(), 0)
	})
}
