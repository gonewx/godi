package testutil

import "testing"

type svc struct {
	Name     string
	Lifetime string
	Deps     []string
}

func TestBuildAppliesOverridesInOrder(t *testing.T) {
	got := Build(
		func(s *svc) { s.Name = "auth" },
		func(s *svc) { s.Lifetime = "singleton" },
		func(s *svc) { s.Deps = append(s.Deps, "logger") },
	)
	want := svc{Name: "auth", Lifetime: "singleton", Deps: []string{"logger"}}
	DeepEqual(t, got, want)
}

func TestBuildZeroValueWhenNoOptions(t *testing.T) {
	got := Build[svc]()
	DeepEqual(t, got, svc{})
}

func TestBuildFromKeepsBaseThenOverrides(t *testing.T) {
	base := svc{Name: "base", Lifetime: "transient"}
	got := BuildFrom(base, func(s *svc) { s.Lifetime = "scoped" })
	Equal(t, got.Name, "base")
	Equal(t, got.Lifetime, "scoped")
}
