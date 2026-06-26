package app

import (
	"testing"

	"godipoc/gen/domain"
)

// Singleton: one instance shared across scopes; built once (with its singleton
// dependency Config wired in topo order).
func TestGenerated_Singleton(t *testing.T) {
	root := NewRoot()
	s1, s2 := root.NewScope(), root.NewScope()
	if s1.Root().Logger() != s2.Root().Logger() {
		t.Fatal("singleton Logger must be identical across scopes")
	}
	if root.Logger().Env != "dev" {
		t.Fatalf("singleton dependency Config not wired: Env=%q", root.Logger().Env)
	}
}

// Scoped: same within a scope, distinct across scopes; disposed LIFO on close.
func TestGenerated_ScopedAndDisposalLIFO(t *testing.T) {
	root := NewRoot()
	s := root.NewScope()

	db := s.DbConnection()
	if db != s.DbConnection() {
		t.Fatal("scoped DbConnection must be cached within a scope")
	}
	audit := s.AuditSink() // tracked after db

	other := root.NewScope()
	if other.DbConnection() == db {
		t.Fatal("scoped DbConnection must differ across scopes")
	}

	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if !db.Closed() || !audit.Closed() {
		t.Fatal("scoped disposables must be Closed on scope close")
	}
	got := s.DisposedOrder()
	want := []string{"AuditSink", "DbConnection"} // LIFO
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("disposal order = %v, want %v", got, want)
	}
}

// Transient: a fresh instance each resolve, sharing the scope's scoped dep.
func TestGenerated_Transient(t *testing.T) {
	root := NewRoot()
	s := root.NewScope()
	u1, u2 := s.NewUserService(), s.NewUserService()
	if u1 == u2 || u1.ID == u2.ID {
		t.Fatal("transient UserService must be fresh each call")
	}
	if u1.Do() == "" {
		t.Fatal("transient resolve produced unusable instance")
	}
}

// Get[T]: reflection-free generic resolver over the generated type switch.
func TestGenerated_Get(t *testing.T) {
	root := NewRoot()
	s := root.NewScope()
	if log, ok := Get[*domain.Logger](s); !ok || log != root.Logger() {
		t.Fatal("Get[*Logger] should resolve the singleton")
	}
	if _, ok := Get[*domain.UserService](s); !ok {
		t.Fatal("Get[*UserService] should resolve the transient")
	}
	type Unregistered struct{}
	if _, ok := Get[*Unregistered](s); ok {
		t.Fatal("Get of an unregistered type must return ok=false")
	}
}

// IEnumerable[T]: all impls, registration order.
func TestGenerated_IEnumerable(t *testing.T) {
	s := NewRoot().NewScope()
	var got []string
	for _, h := range s.Handlers() {
		got = append(got, h.Handle())
	}
	want := []string{"email", "sms", "push"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("handlers = %v, want %v", got, want)
	}
}

// KeyedServices: resolve by key; unknown key -> ok=false.
func TestGenerated_Keyed(t *testing.T) {
	s := NewRoot().NewScope()
	if c, ok := s.Cache("redis"); !ok || c.Kind() != "redis" {
		t.Fatal("Cache(redis)")
	}
	if c, ok := s.Cache("memory"); !ok || c.Kind() != "memory" {
		t.Fatal("Cache(memory)")
	}
	if _, ok := s.Cache("nope"); ok {
		t.Fatal("unknown key must return ok=false")
	}
}
