package godi_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonewx/godi"
)

// TestCloseGuard_RunsReleaseOnceAndCachesError: the first Close runs release
// and caches its error; a second Close returns that same error without running
// release again.
func TestCloseGuard_RunsReleaseOnceAndCachesError(t *testing.T) {
	var g godi.CloseGuard
	want := errors.New("x")
	runs := 0
	release := func() error { runs++; return want }

	if g.Closed() {
		t.Fatal("a fresh guard must not report Closed")
	}
	if err := g.Close(release); err != want {
		t.Fatalf("first Close = %v, want %v", err, want)
	}
	if !g.Closed() {
		t.Fatal("guard must report Closed after Close")
	}
	if err := g.Close(release); err != want {
		t.Fatalf("second Close = %v, want the cached %v", err, want)
	}
	if runs != 1 {
		t.Fatalf("release ran %d times, want 1", runs)
	}
}

// TestCloseGuard_ReentrantCloseDoesNotDeadlock: a Close made from inside
// release returns nil at once (release has not finished, so nothing is cached
// yet) and does not run its own release; the outer Close then completes.
func TestCloseGuard_ReentrantCloseDoesNotDeadlock(t *testing.T) {
	var g godi.CloseGuard
	innerRuns := 0
	var innerErr error
	done := make(chan error, 1)
	go func() {
		done <- g.Close(func() error {
			innerErr = g.Close(func() error { innerRuns++; return errors.New("inner") })
			return nil
		})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("outer Close = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("re-entrant Close deadlocked")
	}
	if innerErr != nil {
		t.Fatalf("inner Close = %v, want nil", innerErr)
	}
	if innerRuns != 0 {
		t.Fatalf("inner release ran %d times, want 0", innerRuns)
	}
}

// TestCloseGuard_ConcurrentCloseRunsReleaseOnce: 32 goroutines released at
// once all call Close; release runs exactly once.
func TestCloseGuard_ConcurrentCloseRunsReleaseOnce(t *testing.T) {
	var g godi.CloseGuard
	var runs atomic.Int32
	release := func() error { runs.Add(1); return nil }

	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_ = g.Close(release)
		}()
	}
	close(start)
	wg.Wait()
	if n := runs.Load(); n != 1 {
		t.Fatalf("release ran %d times, want 1", n)
	}
	if !g.Closed() {
		t.Fatal("guard must report Closed")
	}
}
