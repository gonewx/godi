// ATDD red-phase scaffold for Story 2.3 (AC2) — runtime Disposer unit tests:
// LIFO release order, first-error-continue, idempotent CloseAll.
//
// The Disposer type does not exist yet (runtime/disposer.go, Task 1). This
// file will NOT compile until the type is created — the //go:build tag isolates
// it from the default build. Once Disposer exists, delete the build tag + blank
// line to activate.
//
// API CONTRACT (Story 2.3 D-1 / D-5):
//
//	runtime.NewDisposer() *Disposer
//	(*Disposer).Track(c io.Closer)
//	(*Disposer).CloseAll() error     // LIFO, first-error-continue, clears items
//	(*Disposer).Len() int            // assertion helper
//
// See _bmad-output/test-artifacts/atdd-checklist-2-3-scope-dispose-lifo.md.
package runtime_test

import (
	"errors"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/runtime"
)

// --- disposer test helpers ---------------------------------------------------

// orderRecorder collects Close() call indices for LIFO verification.
type orderRecorder struct {
	order []int
}

// dispItem records its index in the recorder when closed.
type dispItem struct {
	idx      int
	recorder *orderRecorder
	err      error // optional error to return on Close
}

func (d *dispItem) Close() error {
	d.recorder.order = append(d.recorder.order, d.idx)
	return d.err
}

// --- 2.3-DISP-001 (AC2, P0) -------------------------------------------------

// TestDisposer_LIFO_Order: three tracked closers must be released in reverse
// creation order (LIFO) — the core of DisposesInReverseOrderOfCreation (AD-12).
func TestDisposer_LIFO_Order(t *testing.T) {
	rec := &orderRecorder{}
	d := runtime.NewDisposer()

	d.Track(&dispItem{idx: 0, recorder: rec})
	d.Track(&dispItem{idx: 1, recorder: rec})
	d.Track(&dispItem{idx: 2, recorder: rec})

	testutil.Equal(t, d.Len(), 3)

	err := d.CloseAll()
	testutil.NoError(t, err)

	// LIFO: close order must be 2, 1, 0
	testutil.Equal(t, len(rec.order), 3)
	testutil.Equal(t, rec.order[0], 2)
	testutil.Equal(t, rec.order[1], 1)
	testutil.Equal(t, rec.order[2], 0)
}

// --- 2.3-DISP-002 (AC2, P0) -------------------------------------------------

// TestDisposer_FirstErrorContinue: if a middle closer returns an error,
// CloseAll must still close the remaining items and return the first error.
func TestDisposer_FirstErrorContinue(t *testing.T) {
	rec := &orderRecorder{}
	boom := errors.New("boom")
	d := runtime.NewDisposer()

	d.Track(&dispItem{idx: 0, recorder: rec})
	d.Track(&dispItem{idx: 1, recorder: rec, err: boom}) // middle fails
	d.Track(&dispItem{idx: 2, recorder: rec})

	err := d.CloseAll()

	// First error returned
	testutil.Error(t, err)
	testutil.Equal(t, err.Error(), "boom")

	// All three still released (LIFO: 2, 1, 0)
	testutil.Equal(t, len(rec.order), 3)
	testutil.Equal(t, rec.order[0], 2)
	testutil.Equal(t, rec.order[1], 1)
	testutil.Equal(t, rec.order[2], 0)
}

// --- 2.3-DISP-003 (AC2, P1) -------------------------------------------------

// TestDisposer_CloseAll_Idempotent: calling CloseAll a second time must be a
// no-op (items cleared after first call). This is the idempotent foundation
// that root/scope Close builds on (D-5: clear items prevents double release).
func TestDisposer_CloseAll_Idempotent(t *testing.T) {
	rec := &orderRecorder{}
	d := runtime.NewDisposer()

	d.Track(&dispItem{idx: 0, recorder: rec})
	d.Track(&dispItem{idx: 1, recorder: rec})

	err1 := d.CloseAll()
	testutil.NoError(t, err1)
	testutil.Equal(t, len(rec.order), 2)

	// Second CloseAll: no additional releases
	err2 := d.CloseAll()
	testutil.NoError(t, err2)
	testutil.Equal(t, len(rec.order), 2) // still 2, not 4

	testutil.Equal(t, d.Len(), 0) // cleared
}

// --- 2.3-DISP-004 (AC2, P2) -------------------------------------------------

// TestDisposer_EmptyCloseAll_NoError: CloseAll on a fresh disposer (nothing
// tracked) returns nil — edge case.
func TestDisposer_EmptyCloseAll_NoError(t *testing.T) {
	d := runtime.NewDisposer()
	testutil.Equal(t, d.Len(), 0)
	testutil.NoError(t, d.CloseAll())
}
