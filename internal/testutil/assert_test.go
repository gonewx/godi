package testutil

import (
	"errors"
	"fmt"
	"testing"
)

// fakeTB records failures so a test can assert that an assertion helper reports
// (or does not report) a failure — without aborting the real test the way a
// genuine *testing.T.Fatalf would.
type fakeTB struct {
	errs   []string
	fatals []string
}

func (f *fakeTB) Helper()                        {}
func (f *fakeTB) Errorf(format string, a ...any) { f.errs = append(f.errs, fmt.Sprintf(format, a...)) }
func (f *fakeTB) Fatalf(format string, a ...any) {
	f.fatals = append(f.fatals, fmt.Sprintf(format, a...))
}
func (f *fakeTB) failed() bool { return len(f.errs)+len(f.fatals) > 0 }

func TestEqual(t *testing.T) {
	pass := &fakeTB{}
	Equal(pass, 42, 42)
	Equal(pass, "a", "a")
	if pass.failed() {
		t.Errorf("Equal should pass on equal values, got %v", pass.errs)
	}

	fail := &fakeTB{}
	Equal(fail, 1, 2)
	if !fail.failed() {
		t.Error("Equal should fail on unequal values")
	}
}

func TestNoErrorAndError(t *testing.T) {
	pass := &fakeTB{}
	NoError(pass, nil)
	Error(pass, errors.New("boom"))
	if pass.failed() {
		t.Errorf("NoError(nil)/Error(err) should pass; errs=%v fatals=%v", pass.errs, pass.fatals)
	}

	fail := &fakeTB{}
	NoError(fail, errors.New("boom")) // must Fatalf
	Error(fail, nil)                  // must Errorf
	if len(fail.fatals) == 0 {
		t.Error("NoError(err) should Fatalf")
	}
	if len(fail.errs) == 0 {
		t.Error("Error(nil) should Errorf")
	}
}

func TestNilTreatsTypedNilAsNil(t *testing.T) {
	var p *int
	pass := &fakeTB{}
	Nil(pass, nil)
	Nil(pass, p) // a typed-nil pointer must count as nil (AD-5 footgun)
	if pass.failed() {
		t.Errorf("Nil should treat typed-nil as nil, got %v", pass.errs)
	}

	fail := &fakeTB{}
	Nil(fail, 5)
	if !fail.failed() {
		t.Error("Nil(5) should fail")
	}
}

func TestNotEqualTrueNotNil(t *testing.T) {
	x := 7
	pass := &fakeTB{}
	NotEqual(pass, 1, 2)
	True(pass, true, "should hold")
	NotNil(pass, 5)
	NotNil(pass, &x)
	if pass.failed() {
		t.Errorf("expected all to pass, got %v", pass.errs)
	}

	var p *int
	fail := &fakeTB{}
	NotEqual(fail, 3, 3)      // equal → fail
	True(fail, false, "nope") // false → fail
	NotNil(fail, nil)         // nil → fail
	NotNil(fail, p)           // typed-nil → fail
	if len(fail.errs) != 4 {
		t.Errorf("expected 4 failures, got %d: %v", len(fail.errs), fail.errs)
	}
}

func TestDeepEqual(t *testing.T) {
	pass := &fakeTB{}
	DeepEqual(pass, []int{1, 2}, []int{1, 2})
	if pass.failed() {
		t.Errorf("DeepEqual on equal slices should pass, got %v", pass.errs)
	}

	fail := &fakeTB{}
	DeepEqual(fail, []int{1}, []int{2})
	if !fail.failed() {
		t.Error("DeepEqual on mismatched slices should fail")
	}
}

func TestPanics(t *testing.T) {
	pass := &fakeTB{}
	got := Panics(pass, func() { panic("kaboom") })
	if pass.failed() {
		t.Errorf("Panics should pass when fn panics, got %v", pass.errs)
	}
	if got != "kaboom" {
		t.Errorf("Panics should return the recovered value, got %v", got)
	}

	fail := &fakeTB{}
	Panics(fail, func() {}) // does not panic
	if !fail.failed() {
		t.Error("Panics should fail when fn does not panic")
	}
}
