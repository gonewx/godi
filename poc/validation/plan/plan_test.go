package plan

import (
	"strings"
	"testing"
)

// A correct graph validates clean: scoped->singleton and transient->scoped are
// both legal.
func TestValidate_GoodGraph(t *testing.T) {
	r := NewRegistry().
		Add(Service{Name: "Logger", Lifetime: Singleton}).
		Add(Service{Name: "DbConnection", Lifetime: Scoped, DependsOn: []string{"Logger"}}).
		Add(Service{Name: "UserService", Lifetime: Transient, DependsOn: []string{"DbConnection", "Logger"}})

	if errs := r.Validate(); len(errs) != 0 {
		t.Fatalf("good graph should validate, got: %v", errs)
	}
}

// dotnet catches this at first resolve; B2 catches it at generation time.
func TestValidate_MissingDependency(t *testing.T) {
	r := NewRegistry().
		Add(Service{Name: "UserService", Lifetime: Transient, DependsOn: []string{"DbConnection"}})

	errs := r.Validate()
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "missing dependency") {
		t.Fatalf("expected a missing-dependency error, got: %v", errs)
	}
	t.Logf("caught at plan time: %v", errs[0])
}

// The classic captive-dependency bug: a singleton depending on a scoped service.
// dotnet surfaces it only with ValidateScopes=true; B2 surfaces it always, early.
func TestValidate_CaptiveDependency(t *testing.T) {
	r := NewRegistry().
		Add(Service{Name: "Logger", Lifetime: Singleton}).
		Add(Service{Name: "DbConnection", Lifetime: Scoped, DependsOn: []string{"Logger"}}).
		Add(Service{Name: "BadCache", Lifetime: Singleton, DependsOn: []string{"DbConnection"}}) // BUG

	errs := r.Validate()
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "captive dependency") {
		t.Fatalf("expected a captive-dependency error, got: %v", errs)
	}
	t.Logf("caught at plan time: %v", errs[0])
}

// Transitive captive dependency: singleton -> transient -> scoped is still captive.
func TestValidate_CaptiveDependency_Transitive(t *testing.T) {
	r := NewRegistry().
		Add(Service{Name: "Scope1", Lifetime: Scoped}).
		Add(Service{Name: "Mid", Lifetime: Transient, DependsOn: []string{"Scope1"}}).
		Add(Service{Name: "Root1", Lifetime: Singleton, DependsOn: []string{"Mid"}})

	errs := r.Validate()
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "captive dependency") {
		t.Fatalf("expected a transitive captive-dependency error, got: %v", errs)
	}
	t.Logf("caught at plan time: %v", errs[0])
}

func TestValidate_Cycle(t *testing.T) {
	r := NewRegistry().
		Add(Service{Name: "A", Lifetime: Singleton, DependsOn: []string{"B"}}).
		Add(Service{Name: "B", Lifetime: Singleton, DependsOn: []string{"A"}})

	errs := r.Validate()
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "dependency cycle") {
			found = true
			t.Logf("caught at plan time: %v", e)
		}
	}
	if !found {
		t.Fatalf("expected a cycle error, got: %v", errs)
	}
}
