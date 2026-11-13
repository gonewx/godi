package runtime_test

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/runtime"
)

// concSingleton is a plain singleton the side goroutine resolves while the
// transient workers run.
type concSingleton struct{ _ int }

// TestRuntime_ConcurrentTransientResolutionTracksEveryInstance: 8 goroutines
// released at once each resolve a disposable transient 500 times from one
// root, while another goroutine keeps resolving a singleton and opening and
// closing scopes. root.Close() must then close every one of the 4000
// transients exactly once.
func TestRuntime_ConcurrentTransientResolutionTracksEveryInstance(t *testing.T) {
	const workers, perWorker = 8, 500
	b := runtime.NewBuilder()
	runtime.AddSingleton[*concSingleton](b, func(godi.Provider) *concSingleton { return &concSingleton{} })
	runtime.AddTransient[*countingCloser](b, func(godi.Provider) *countingCloser { return &countingCloser{} })
	p, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	transientType := reflect.TypeFor[*countingCloser]()
	singletonType := reflect.TypeFor[*concSingleton]()

	got := make([][]*countingCloser, workers)
	start := make(chan struct{})
	stop := make(chan struct{})
	var workersWG, sideWG sync.WaitGroup
	for w := range workers {
		workersWG.Add(1)
		go func() {
			defer workersWG.Done()
			<-start
			out := make([]*countingCloser, 0, perWorker)
			for range perWorker {
				out = append(out, p.GetService(transientType).(*countingCloser))
			}
			got[w] = out
		}()
	}
	sideWG.Add(1)
	go func() {
		defer sideWG.Done()
		<-start
		for {
			select {
			case <-stop:
				return
			default:
			}
			if p.GetService(singletonType) == nil {
				t.Error("singleton resolved to nil")
				return
			}
			sc := p.CreateScope()
			if err := sc.Close(); err != nil {
				t.Errorf("scope Close: %v", err)
				return
			}
		}
	}()
	close(start)
	workersWG.Wait()
	close(stop)
	sideWG.Wait()

	if err := p.Close(); err != nil {
		t.Fatalf("root Close: %v", err)
	}
	seen := make(map[*countingCloser]bool, workers*perWorker)
	for w := range got {
		for i, c := range got[w] {
			if seen[c] {
				t.Fatalf("transient %d/%d resolved twice", w, i)
			}
			seen[c] = true
			if n := c.closes.Load(); n != 1 {
				t.Fatalf("transient %d/%d closed %d times, want 1", w, i, n)
			}
		}
	}
	if len(seen) != workers*perWorker {
		t.Fatalf("resolved %d distinct transients, want %d", len(seen), workers*perWorker)
	}
}

// reentrantSingleton closes its own provider from inside Close.
type reentrantSingleton struct {
	p      godi.Provider
	closes atomic.Int32
}

func (s *reentrantSingleton) Close() error {
	s.closes.Add(1)
	return s.p.Close()
}

// TestRuntime_ReentrantCloseFromDisposable: a singleton whose Close calls the
// provider's Close must not deadlock root.Close, and must be closed exactly
// once.
func TestRuntime_ReentrantCloseFromDisposable(t *testing.T) {
	b := runtime.NewBuilder()
	runtime.AddSingleton[*reentrantSingleton](b, func(godi.Provider) *reentrantSingleton { return &reentrantSingleton{} })
	p, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	s := p.GetService(reflect.TypeFor[*reentrantSingleton]()).(*reentrantSingleton)
	s.p = p

	done := make(chan error, 1)
	go func() { done <- p.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("root Close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("root Close deadlocked on a re-entrant Close from a disposable")
	}
	if n := s.closes.Load(); n != 1 {
		t.Fatalf("singleton closed %d times, want 1", n)
	}
}
