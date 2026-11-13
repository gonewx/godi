package runtime

import (
	"context"
	"fmt"
	"io"

	"github.com/gonewx/godi"
)

// Disposer tracks disposable instances and releases them in LIFO order. It is
// exported because generated containers (in a separate package) consume it
// (AD-2: generated --> runtime(disposer)).
//
// items holds each tracked instance as `any` (Story 6.3 — was []io.Closer):
// synchronous disposables (io.Closer) and both-disposables enter via Track,
// async-only disposables (godi.AsyncDisposable) via TrackAsync. Storing the
// dynamic value lets CloseAllAsync re-assert godi.AsyncDisposable on a
// both-disposable that was tracked through the sync Track path (prefer-async,
// D-3/D-4). The field-type change is internal to runtime — generated code only
// calls TrackAny/CloseAll/CloseAllAsync and never touches items.
type Disposer struct {
	items []any
}

func NewDisposer() *Disposer {
	return &Disposer{}
}

// Track records a synchronous (io.Closer) disposable. Both-disposables
// (io.Closer + godi.AsyncDisposable) also enter here because TrackAny asserts
// io.Closer first (D-3).
func (d *Disposer) Track(c io.Closer) {
	d.items = append(d.items, c)
}

// TrackAsync records an async-only disposable (implements godi.AsyncDisposable
// but NOT io.Closer). Sync CloseAll cannot release it and reports an explicit
// error (AC3); CloseAllAsync releases it via CloseAsync.
func (d *Disposer) TrackAsync(ad godi.AsyncDisposable) {
	d.items = append(d.items, ad)
}

// TrackAny records v by its dynamic type: an io.Closer (including a
// both-disposable) via Track, otherwise a godi.AsyncDisposable via TrackAsync.
// A nil v, or one that is neither, is not tracked. It is the single tracking
// entry point for both the runtime provider and generated containers — the
// latter cannot know a form-B factory's product type at generation time.
func (d *Disposer) TrackAny(v any) {
	if v == nil {
		return
	}
	if c, ok := v.(io.Closer); ok {
		d.Track(c)
		return
	}
	if ad, ok := v.(godi.AsyncDisposable); ok {
		d.TrackAsync(ad)
	}
}

// CloseAll releases every tracked item synchronously in LIFO (reverse-creation)
// order. An item implementing io.Closer is released via Close. An async-only
// item (godi.AsyncDisposable without io.Closer) CANNOT be released
// synchronously — it contributes an explicit error naming CloseAsync as the
// remedy (AC3, mapping dotnet's InvalidOperationException "'{0}' type only
// implements IAsyncDisposable. Use DisposeAsync to dispose the container."),
// while every releasable item on either side of it still releases. The first
// error is retained (first-error-continue, unchanged from Story 2.3). Items are
// cleared so a second call is a no-op.
func (d *Disposer) CloseAll() error {
	var first error
	for i := len(d.items) - 1; i >= 0; i-- {
		v := d.items[i]
		if c, ok := v.(io.Closer); ok {
			if err := c.Close(); err != nil && first == nil {
				first = err
			}
			continue
		}
		if first == nil {
			first = fmt.Errorf("godi: %T only implements AsyncDisposable; use CloseAsync to dispose it (AD-16/AR12)", v)
		}
	}
	d.items = nil
	return first
}

// CloseAllAsync releases every tracked item in LIFO order, preferring async: an
// item implementing godi.AsyncDisposable is released via CloseAsync(ctx) — even
// a both-disposable tracked through the sync Track path, because the assertion
// order (async first) decides, not how the item was tracked (D-4). Otherwise an
// io.Closer is released via its sync Close (D-4 row 1 fallback). ctx is passed
// through to each CloseAsync unmodified — the Go "context-aware graceful
// release" reading of async (D-1). The first error is retained while the rest
// still release (first-error-continue). Items are cleared so a second call is a
// no-op.
func (d *Disposer) CloseAllAsync(ctx context.Context) error {
	var first error
	for i := len(d.items) - 1; i >= 0; i-- {
		v := d.items[i]
		if ad, ok := v.(godi.AsyncDisposable); ok {
			if err := ad.CloseAsync(ctx); err != nil && first == nil {
				first = err
			}
			continue
		}
		if c, ok := v.(io.Closer); ok {
			if err := c.Close(); err != nil && first == nil {
				first = err
			}
		}
	}
	d.items = nil
	return first
}

func (d *Disposer) Len() int {
	return len(d.items)
}
