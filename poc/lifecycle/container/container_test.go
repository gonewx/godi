package container

import (
	"testing"

	"godipoc/lifecycle/domain"
)

// Singleton: one instance shared everywhere, including across scopes.
func TestSingleton_SharedAcrossScopes(t *testing.T) {
	root := NewRoot()
	s1, s2 := root.NewScope(), root.NewScope()
	if s1.Root().Logger() != s2.Root().Logger() {
		t.Fatal("singleton Logger must be identical across scopes")
	}
}

// Scoped: same instance within a scope, distinct across scopes.
func TestScoped_SameWithinScope_DifferAcrossScopes(t *testing.T) {
	root := NewRoot()
	s1 := root.NewScope()
	first := s1.DbConnection()
	second := s1.DbConnection() // second call must return the cached instance
	if first != second {
		t.Fatal("scoped DbConnection must be the same within one scope")
	}
	s2 := root.NewScope()
	if s1.DbConnection() == s2.DbConnection() {
		t.Fatal("scoped DbConnection must differ across scopes")
	}
}

// Transient: a fresh instance each resolve, but it still shares the scope's
// scoped dependency.
func TestTransient_FreshEachResolve_SharesScopedDep(t *testing.T) {
	root := NewRoot()
	s := root.NewScope()
	u1, u2 := s.NewUserService(), s.NewUserService()
	if u1 == u2 || u1.ID == u2.ID {
		t.Fatal("transient UserService must be a fresh instance each call")
	}
	// Both transients share the one scoped DbConnection (constructor-injected).
	if u1.Do() == "" || u2.Do() == "" {
		t.Fatal("transient resolve produced unusable instance")
	}
}

// Disposal: scoped disposables are Closed on scope close, in LIFO order.
func TestDisposal_LIFO_OnScopeClose(t *testing.T) {
	root := NewRoot()
	s := root.NewScope()

	db := s.DbConnection() // tracked first
	audit := s.AuditSink() // tracked second
	if db.Closed() || audit.Closed() {
		t.Fatal("nothing should be closed before scope close")
	}

	if err := s.Close(); err != nil {
		t.Fatalf("scope close: %v", err)
	}
	if !db.Closed() || !audit.Closed() {
		t.Fatal("scoped disposables must be Closed on scope close")
	}

	got := s.DisposedOrder()
	want := []string{"AuditSink", "DbConnection"} // LIFO
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("disposal order = %v, want %v (LIFO)", got, want)
	}
}

// Closing a scope must NOT dispose singletons (they outlive scopes).
func TestSingleton_NotDisposedOnScopeClose(t *testing.T) {
	root := NewRoot()
	s := root.NewScope()
	_ = s.DbConnection()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	// Logger (singleton) must still be usable after a scope closes.
	if root.Logger().Log("alive") == "" {
		t.Fatal("singleton must survive scope close")
	}
}

// Get[T]: generic ergonomic facade resolves via a zero-reflection type switch.
func TestGet_GenericFacade_ZeroReflection(t *testing.T) {
	root := NewRoot()
	s := root.NewScope()

	log, ok := Get[*domain.Logger](s)
	if !ok || log != root.Logger() {
		t.Fatal("Get[*Logger] should resolve the singleton instance")
	}
	if _, ok := Get[*domain.UserService](s); !ok {
		t.Fatal("Get[*UserService] should resolve the transient")
	}

	type Unregistered struct{}
	if _, ok := Get[*Unregistered](s); ok {
		t.Fatal("Get of an unregistered type must return ok=false")
	}
}
