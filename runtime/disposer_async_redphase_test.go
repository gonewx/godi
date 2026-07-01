// ATDD red-phase scaffold for Story 6.3 (AC2/AC3) — runtime Disposer async
// evolution: TrackAsync + CloseAllAsync (LIFO, prefer-async, first-error-
// continue), and CloseAll (sync) returning an explicit error when it hits an
// async-only item instead of panicking/silently skipping/leaking (AC3).
//
// (*Disposer).TrackAsync and (*Disposer).CloseAllAsync do not exist yet
// (runtime/disposer.go, Task 2) — this file will NOT compile until they are
// added. The //go:build tag isolates it from the default build.
//
// API CONTRACT (Story 6.3 D-3/D-4):
//
//	(*Disposer).TrackAsync(ad godi.AsyncDisposable)     // new: async-only items
//	(*Disposer).CloseAllAsync(ctx context.Context) error // new: LIFO, prefer-async
//	(*Disposer).CloseAll() error                         // existing signature UNCHANGED;
//	                                                      // now errors (not panics) on an
//	                                                      // async-only tracked item (AC3)
//
// Activation: delete the //go:build line and blank line to join the default
// build. See _bmad-output/test-artifacts/atdd-checklist-6-3-async-scope-dispose.md.
package runtime_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// --- async disposer test helpers --------------------------------------------

// asyncOrderRecorder collects Close()/CloseAsync() call labels in call order —
// the async analogue of disposer_redphase_test.go's orderRecorder (D-7).
type asyncOrderRecorder struct {
	order []string
}

// syncOnlyItem implements ONLY io.Closer (tracked via the existing Track).
type syncOnlyItem struct {
	label    string
	recorder *asyncOrderRecorder
	err      error
}

func (s *syncOnlyItem) Close() error {
	s.recorder.order = append(s.recorder.order, s.label)
	return s.err
}

// asyncOnlyItem implements ONLY godi.AsyncDisposable (tracked via TrackAsync).
// CloseAll (sync) must error on it (AC3); CloseAllAsync must call CloseAsync.
type asyncOnlyItem struct {
	label    string
	recorder *asyncOrderRecorder
	err      error
	gotCtx   context.Context
}

func (a *asyncOnlyItem) CloseAsync(ctx context.Context) error {
	a.gotCtx = ctx
	a.recorder.order = append(a.recorder.order, a.label)
	return a.err
}

// bothItem implements BOTH io.Closer and godi.AsyncDisposable. closedVia
// records which method actually fired — the prefer-async witness (D-4).
type bothItem struct {
	label     string
	recorder  *asyncOrderRecorder
	closedVia string
}

func (b *bothItem) Close() error {
	b.closedVia = "sync"
	b.recorder.order = append(b.recorder.order, b.label)
	return nil
}

func (b *bothItem) CloseAsync(ctx context.Context) error {
	b.closedVia = "async"
	b.recorder.order = append(b.recorder.order, b.label)
	return nil
}

// --- 6.3-DISP-001 (AC2, P0) --------------------------------------------------

// TestDisposerAsync_CloseAllAsync_LIFO_MixedSyncAsync: CloseAllAsync must
// release BOTH sync-only (fallback) and async-only items in LIFO order —
// tracking order is what determines release order, not item kind.
func TestDisposerAsync_CloseAllAsync_LIFO_MixedSyncAsync(t *testing.T) {
	rec := &asyncOrderRecorder{}
	d := runtime.NewDisposer()

	d.Track(&syncOnlyItem{label: "0", recorder: rec})
	d.TrackAsync(&asyncOnlyItem{label: "1", recorder: rec})
	d.Track(&syncOnlyItem{label: "2", recorder: rec})

	err := d.CloseAllAsync(context.Background())
	testutil.NoError(t, err)

	testutil.Equal(t, len(rec.order), 3)
	testutil.Equal(t, rec.order[0], "2")
	testutil.Equal(t, rec.order[1], "1")
	testutil.Equal(t, rec.order[2], "0")
}

// --- 6.3-DISP-002 (AC2, P0) --------------------------------------------------

// TestDisposerAsync_CloseAllAsync_PrefersAsyncOnBothImplementor: an item
// implementing BOTH io.Closer and AsyncDisposable, tracked via the existing
// Track(io.Closer) path, must still be released through CloseAsync when
// CloseAllAsync runs (D-4 prefer-async — the interface assertion order
// matters, not how the item was tracked).
func TestDisposerAsync_CloseAllAsync_PrefersAsyncOnBothImplementor(t *testing.T) {
	rec := &asyncOrderRecorder{}
	both := &bothItem{label: "both", recorder: rec}
	d := runtime.NewDisposer()
	d.Track(both) // both satisfies io.Closer; stored via the existing sync Track (D-3)

	testutil.NoError(t, d.CloseAllAsync(context.Background()))

	testutil.Equal(t, len(rec.order), 1)
	testutil.Equal(t, both.closedVia, "async")
}

// --- 6.3-DISP-003 (AC2, P0) --------------------------------------------------

// TestDisposerAsync_CloseAll_PrefersSyncOnBothImplementor: the SYNC CloseAll
// must still call Close on a both-implementor (it only knows io.Closer) —
// sanity check that sync behavior is unaffected by the async addition.
func TestDisposerAsync_CloseAll_PrefersSyncOnBothImplementor(t *testing.T) {
	rec := &asyncOrderRecorder{}
	both := &bothItem{label: "both", recorder: rec}
	d := runtime.NewDisposer()
	d.Track(both)

	testutil.NoError(t, d.CloseAll())

	testutil.Equal(t, len(rec.order), 1)
	testutil.Equal(t, both.closedVia, "sync")
}

// --- 6.3-DISP-004 (AC2, P0) --------------------------------------------------

// TestDisposerAsync_CloseAllAsync_FallsBackToSyncOnly: an item that ONLY
// implements io.Closer (tracked via Track) must still be released by
// CloseAllAsync — falling back to its sync Close (D-4 row 1).
func TestDisposerAsync_CloseAllAsync_FallsBackToSyncOnly(t *testing.T) {
	rec := &asyncOrderRecorder{}
	d := runtime.NewDisposer()
	d.Track(&syncOnlyItem{label: "s", recorder: rec})

	testutil.NoError(t, d.CloseAllAsync(context.Background()))

	testutil.Equal(t, len(rec.order), 1)
	testutil.Equal(t, rec.order[0], "s")
}

// --- 6.3-DISP-005 (AC3, P0) --------------------------------------------------

// TestDisposerAsync_CloseAll_ReturnsErrorForAsyncOnlyItem_FirstErrorContinue:
// the SYNC CloseAll, hitting an async-only tracked item, must return an
// explicit error (not panic, not silently skip, not leak) that names
// CloseAsync as the remedy (AC3) — and must still release the sync-only
// items on either side of it (first-error-continue, unchanged from Story 2.3).
func TestDisposerAsync_CloseAll_ReturnsErrorForAsyncOnlyItem_FirstErrorContinue(t *testing.T) {
	rec := &asyncOrderRecorder{}
	d := runtime.NewDisposer()
	d.Track(&syncOnlyItem{label: "before", recorder: rec})
	d.TrackAsync(&asyncOnlyItem{label: "only", recorder: rec})
	d.Track(&syncOnlyItem{label: "after", recorder: rec})

	err := d.CloseAll()

	testutil.Error(t, err)
	testutil.True(t, strings.Contains(err.Error(), "CloseAsync"),
		"AC3: sync CloseAll on an async-only item must name CloseAsync in the error text")

	// LIFO + first-error-continue: "after" and "before" still release; "only"
	// cannot record via Close (it has none) so it contributes only the error.
	testutil.Equal(t, len(rec.order), 2)
	testutil.Equal(t, rec.order[0], "after")
	testutil.Equal(t, rec.order[1], "before")
}

// --- 6.3-DISP-006 (AC2, P1) --------------------------------------------------

// TestDisposerAsync_CloseAllAsync_FirstErrorContinue: CloseAllAsync must keep
// releasing remaining items after one returns an error, and surface the
// FIRST error (mirrors the sync CloseAll contract from Story 2.3).
func TestDisposerAsync_CloseAllAsync_FirstErrorContinue(t *testing.T) {
	rec := &asyncOrderRecorder{}
	boom := errors.New("boom")
	d := runtime.NewDisposer()

	d.TrackAsync(&asyncOnlyItem{label: "0", recorder: rec})
	d.TrackAsync(&asyncOnlyItem{label: "1", recorder: rec, err: boom})
	d.TrackAsync(&asyncOnlyItem{label: "2", recorder: rec})

	err := d.CloseAllAsync(context.Background())

	testutil.Error(t, err)
	testutil.Equal(t, err.Error(), "boom")

	testutil.Equal(t, len(rec.order), 3)
	testutil.Equal(t, rec.order[0], "2")
	testutil.Equal(t, rec.order[1], "1")
	testutil.Equal(t, rec.order[2], "0")
}

// --- 6.3-DISP-007 (AC2, P1) --------------------------------------------------

// TestDisposerAsync_CloseAllAsync_Idempotent: a second CloseAllAsync call
// must be a no-op (items cleared after the first call — mirrors the sync
// CloseAll idempotent contract).
func TestDisposerAsync_CloseAllAsync_Idempotent(t *testing.T) {
	rec := &asyncOrderRecorder{}
	d := runtime.NewDisposer()
	d.TrackAsync(&asyncOnlyItem{label: "a", recorder: rec})

	testutil.NoError(t, d.CloseAllAsync(context.Background()))
	testutil.Equal(t, len(rec.order), 1)

	testutil.NoError(t, d.CloseAllAsync(context.Background()))
	testutil.Equal(t, len(rec.order), 1) // still 1, not 2
}

// --- 6.3-DISP-008 (AC2, P1) --------------------------------------------------

// TestDisposerAsync_CloseAllAsync_PassesContextThrough: the ctx argument
// passed to CloseAllAsync must reach each tracked item's CloseAsync call
// unmodified (Go's "context-aware graceful release" reading of async, D-1).
func TestDisposerAsync_CloseAllAsync_PassesContextThrough(t *testing.T) {
	rec := &asyncOrderRecorder{}
	item := &asyncOnlyItem{label: "ctx", recorder: rec}
	d := runtime.NewDisposer()
	d.TrackAsync(item)

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "marker")

	testutil.NoError(t, d.CloseAllAsync(ctx))
	testutil.NotNil(t, item.gotCtx)
	testutil.Equal(t, item.gotCtx.Value(ctxKey{}), any("marker"))
}
