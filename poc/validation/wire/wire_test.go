package wire

import "testing"

// IEnumerable[T]: all implementations resolve together, in registration order.
func TestIEnumerable_AllImplsInRegistrationOrder(t *testing.T) {
	c := New()
	var got []string
	for _, h := range c.Handlers() {
		got = append(got, h.Handle())
	}
	want := []string{"email", "sms", "push"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order mismatch at %d: got %v, want %v", i, got, want)
		}
	}
}

// KeyedServices: resolve a specific implementation by key; unknown key -> ok=false.
func TestKeyed_ResolveByKey(t *testing.T) {
	c := New()

	redis, ok := c.Cache("redis")
	if !ok || redis.Kind() != "redis" {
		t.Fatalf("key %q -> %v, ok=%v", "redis", redis, ok)
	}
	mem, ok := c.Cache("memory")
	if !ok || mem.Kind() != "memory" {
		t.Fatalf("key %q -> %v, ok=%v", "memory", mem, ok)
	}
	if _, ok := c.Cache("nope"); ok {
		t.Fatal("unknown key must return ok=false")
	}
}
