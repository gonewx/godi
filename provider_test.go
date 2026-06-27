//go:build atdd_redphase

// ATDD red-phase scaffold for Story 1.2 (AC1) — the AD-16 Provider / Scope
// interface contract.
//
// Activation (TDD red -> green): delete the //go:build line above and the blank
// line after it once godi.Provider and godi.Scope exist (Story 1.2 Task 1). The
// file then joins the default build and these assertions drive the contract to
// green. See _bmad-output/test-artifacts/atdd-checklist-1-2-provider-interface.md.
package godi_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
)

// wantProviderMethods is the canonical AD-16 method face: EXACTLY these six
// methods, each with the pinned signature. A method renamed, dropped, added, or
// drifting in signature breaks 1.2-UNIT-001. Interface methods report their type
// without a receiver, so each value here is the bare func signature.
func wantProviderMethods() map[string]reflect.Type {
	return map[string]reflect.Type{
		"GetService":       reflect.TypeOf(func(reflect.Type) any { return nil }),
		"IsService":        reflect.TypeOf(func(reflect.Type) bool { return false }),
		"GetKeyedService":  reflect.TypeOf(func(reflect.Type, any) any { return nil }),
		"GetKeyedServices": reflect.TypeOf(func(reflect.Type, any) []any { return nil }),
		"CreateScope":      reflect.TypeOf(func() godi.Scope { return nil }),
		"Close":            reflect.TypeOf(func() error { return nil }),
	}
}

// TestProvider_MethodFaceIsExactlyAD16 (1.2-UNIT-001, AC1, P0): godi.Provider is
// a NON-generic interface (AD-5: an interface method may not declare type
// parameters, so Get[T] is deliberately absent) whose method set is exactly the
// six AD-16 methods with the pinned signatures.
func TestProvider_MethodFaceIsExactlyAD16(t *testing.T) {
	pt := reflect.TypeOf((*godi.Provider)(nil)).Elem()
	if pt.Kind() != reflect.Interface {
		t.Fatalf("godi.Provider must be an interface, got %s", pt.Kind())
	}

	want := wantProviderMethods()
	testutil.Equal(t, pt.NumMethod(), len(want))

	for i := 0; i < pt.NumMethod(); i++ {
		m := pt.Method(i)
		wantSig, ok := want[m.Name]
		testutil.True(t, ok, "unexpected Provider method: "+m.Name)
		if !ok {
			continue
		}
		if m.Type != wantSig {
			t.Errorf("Provider.%s signature mismatch:\n got: %s\nwant: %s", m.Name, m.Type, wantSig)
		}
		delete(want, m.Name)
	}
	for name := range want {
		t.Errorf("Provider is missing AD-16 method: %s", name)
	}
}

// _ pins, at compile time, that a Scope is assignable to a Provider — Scope
// exposes the same resolution face as its parent (AD-16).
var _ godi.Provider = (godi.Scope)(nil)

// TestScope_IsProviderWithNoExtraMethods (1.2-UNIT-002, AC1, P1): godi.Scope is
// exactly interface{ Provider } — it satisfies Provider and adds no method of
// its own (D3: Dispose is covered by Close, ServiceProvider by "a scope IS a
// Provider").
func TestScope_IsProviderWithNoExtraMethods(t *testing.T) {
	pt := reflect.TypeOf((*godi.Provider)(nil)).Elem()
	st := reflect.TypeOf((*godi.Scope)(nil)).Elem()
	if st.Kind() != reflect.Interface {
		t.Fatalf("godi.Scope must be an interface, got %s", st.Kind())
	}
	testutil.True(t, st.Implements(pt), "Scope must satisfy Provider (AD-16)")
	testutil.Equal(t, st.NumMethod(), pt.NumMethod()) // D3: no method beyond Provider's
}
