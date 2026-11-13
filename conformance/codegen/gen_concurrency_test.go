package codegen_test

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
)

// TestCodegen_ConcurrentTransientResolutionTracksEveryInstance: 8 goroutines
// released at once each resolve the disposable transient 500 times from one
// generated Root, while another goroutine keeps resolving a singleton and
// opening and closing scopes. root.Close() must then release every one of the
// 4000 transients, each exactly once.
func TestCodegen_ConcurrentTransientResolutionTracksEveryInstance(t *testing.T) {
	const workers, perWorker = 8, 500
	root, err := codegen.NewRoot()
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	transientType := reflect.TypeFor[fakes.IFakeDisposableTransient]()
	singletonType := reflect.TypeFor[fakes.IFakeService]()

	// Each transient gets its own label on one shared recorder. Recording
	// happens inside root.Close on this goroutine, after every worker is done.
	recorder := fakes.NewDisposeRecorder()
	got := make([][]*fakes.FakeDisposableTransient, workers)
	start := make(chan struct{})
	stop := make(chan struct{})
	var workersWG, sideWG sync.WaitGroup
	for w := range workers {
		workersWG.Add(1)
		go func() {
			defer workersWG.Done()
			<-start
			out := make([]*fakes.FakeDisposableTransient, 0, perWorker)
			for i := range perWorker {
				v := root.GetService(transientType).(*fakes.FakeDisposableTransient)
				v.SetRecorder(recorder, fmt.Sprintf("%d/%d", w, i))
				out = append(out, v)
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
			if root.GetService(singletonType) == nil {
				t.Error("singleton resolved to nil")
				return
			}
			sc := root.CreateScope()
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

	if err := root.Close(); err != nil {
		t.Fatalf("root Close: %v", err)
	}
	for w := range got {
		for i, v := range got[w] {
			if !v.IsDisposed() {
				t.Fatalf("transient %d/%d was not released by root.Close", w, i)
			}
		}
	}
	labels := recorder.Labels()
	if len(labels) != workers*perWorker {
		t.Fatalf("root.Close released %d transients, want %d", len(labels), workers*perWorker)
	}
	seen := make(map[string]bool, len(labels))
	for _, l := range labels {
		if seen[l] {
			t.Fatalf("transient %s released more than once", l)
		}
		seen[l] = true
	}
}

// scopedAccessor is the generated typed accessor of the disposable scoped
// service; the generated scope type is unexported, so tests reach it through
// this interface.
type scopedAccessor interface {
	FakeDisposableScoped() fakes.IFakeDisposableScoped
}

// requireProviderClosedPanic runs fn and fails unless it panics with
// godi.ErrProviderClosed.
func requireProviderClosedPanic(t *testing.T, what string, fn func()) {
	t.Helper()
	var rec any
	func() {
		defer func() { rec = recover() }()
		fn()
	}()
	if rec == nil {
		t.Fatalf("%s: expected a panic with godi.ErrProviderClosed, got none", what)
	}
	err, ok := rec.(error)
	if !ok || !errors.Is(err, godi.ErrProviderClosed) {
		t.Fatalf("%s: expected godi.ErrProviderClosed, got %v", what, rec)
	}
}

// TestCodegen_ScopedTypedAccessorAfterClosePanics: calling a scoped service's
// typed accessor after Close must panic with godi.ErrProviderClosed, like
// GetService, instead of building a fresh instance into a disposer that has
// already been released. A scope checks its own state and its root's.
func TestCodegen_ScopedTypedAccessorAfterClosePanics(t *testing.T) {
	t.Run("closed root", func(t *testing.T) {
		root, err := codegen.NewRoot()
		if err != nil {
			t.Fatalf("NewRoot: %v", err)
		}
		if err := root.Close(); err != nil {
			t.Fatalf("root Close: %v", err)
		}
		requireProviderClosedPanic(t, "root.FakeDisposableScoped after root Close", func() {
			_ = root.FakeDisposableScoped()
		})
	})
	t.Run("closed scope", func(t *testing.T) {
		root, err := codegen.NewRoot()
		if err != nil {
			t.Fatalf("NewRoot: %v", err)
		}
		defer root.Close()
		sc := root.CreateScope()
		if err := sc.Close(); err != nil {
			t.Fatalf("scope Close: %v", err)
		}
		requireProviderClosedPanic(t, "scope.FakeDisposableScoped after scope Close", func() {
			_ = sc.(scopedAccessor).FakeDisposableScoped()
		})
	})
	t.Run("open scope of a closed root", func(t *testing.T) {
		root, err := codegen.NewRoot()
		if err != nil {
			t.Fatalf("NewRoot: %v", err)
		}
		sc := root.CreateScope()
		defer sc.Close()
		if err := root.Close(); err != nil {
			t.Fatalf("root Close: %v", err)
		}
		requireProviderClosedPanic(t, "scope.FakeDisposableScoped after root Close", func() {
			_ = sc.(scopedAccessor).FakeDisposableScoped()
		})
	})
	t.Run("open scope still resolves", func(t *testing.T) {
		root, err := codegen.NewRoot()
		if err != nil {
			t.Fatalf("NewRoot: %v", err)
		}
		defer root.Close()
		sc := root.CreateScope()
		v := sc.(scopedAccessor).FakeDisposableScoped()
		if v == nil {
			t.Fatal("open scope must resolve the scoped service")
		}
		if err := sc.Close(); err != nil {
			t.Fatalf("scope Close: %v", err)
		}
		if !v.IsDisposed() {
			t.Fatal("scope Close must release the scoped instance")
		}
	})
}
