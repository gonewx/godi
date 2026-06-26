package genlib_test

import (
	"strings"
	"testing"

	"godipoc/gen/genlib"
	"godipoc/gen/plan"
	"godipoc/gen/spec"
)

// The real spec generates successfully and emits the expected entry points.
func TestGenerate_GoodGraph(t *testing.T) {
	src, err := genlib.Generate(spec.Registry())
	if err != nil {
		t.Fatalf("good graph should generate: %v", err)
	}
	for _, want := range []string{
		"func NewRoot() *Root",
		"func (s *Scope) DbConnection() *domain.DbConnection",
		"func (s *Scope) NewUserService() *domain.UserService",
		"func Get[T any](s *Scope)",
		"func (s *Scope) Handlers() []domain.IHandler",
		"func (s *Scope) Cache(key string) (domain.ICache, bool)",
	} {
		if !strings.Contains(string(src), want) {
			t.Errorf("generated source missing: %s", want)
		}
	}
}

// A captive dependency (singleton -> scoped) is REFUSED: no code is produced.
func TestGenerate_RejectsCaptiveDependency(t *testing.T) {
	bad := plan.NewRegistry().
		Service(plan.Service{Name: "Db", GoType: "*x.Db", Ctor: "x.NewDb", Lifetime: plan.Scoped}).
		Service(plan.Service{Name: "Cache", GoType: "*x.Cache", Ctor: "x.NewCache", Lifetime: plan.Singleton, Deps: []string{"Db"}})

	src, err := genlib.Generate(bad)
	if err == nil {
		t.Fatal("expected generation to be refused for a captive dependency")
	}
	if src != nil {
		t.Fatal("no code must be emitted for an invalid graph")
	}
	if !strings.Contains(err.Error(), "captive dependency") {
		t.Fatalf("error should explain the captive dependency, got: %v", err)
	}
	t.Logf("generation refused (as expected): %v", err)
}

// A missing dependency is likewise refused at generation time.
func TestGenerate_RejectsMissingDependency(t *testing.T) {
	bad := plan.NewRegistry().
		Service(plan.Service{Name: "Svc", GoType: "*x.Svc", Ctor: "x.NewSvc", Lifetime: plan.Transient, Deps: []string{"Nope"}})

	if _, err := genlib.Generate(bad); err == nil || !strings.Contains(err.Error(), "missing dependency") {
		t.Fatalf("expected missing-dependency refusal, got: %v", err)
	}
}
