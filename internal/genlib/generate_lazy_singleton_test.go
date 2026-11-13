package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// recoverBranch returns the part of NewRoot's body from its recover() call to
// the end of the deferred func, i.e. the code that runs when construction panics.
func recoverBranch(t *testing.T, code string) string {
	t.Helper()
	body := methodBody(t, code, "func NewRoot(")
	idx := strings.Index(body, "recover()")
	if idx < 0 {
		t.Fatalf("NewRoot must recover construction panics:\n%s", body)
	}
	branch := body[idx:]
	if end := strings.Index(branch, "\n\t}()"); end >= 0 {
		branch = branch[:end]
	}
	return branch
}

// TestGenerate_SingletonCycleThroughFactoriesIsBuildError: two form-B singletons
// whose factories resolve each other, with Deps left empty because the scanner
// cannot see inside the factory bodies. Static validation passes; the accessor's
// in-construction state must turn the cycle into a CircularDependencyError panic,
// which NewRoot's recover turns into its returned error.
func TestGenerate_SingletonCycleThroughFactoriesIsBuildError(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:        "A",
		GoType:      "svc.IA",
		Lifetime:    plan.Singleton,
		FactoryExpr: "func(p godi.Provider) svc.IA { return svc.NewA(godi.GetRequired[svc.IB](p)) }",
	})
	reg.Service(plan.Service{
		Name:        "B",
		GoType:      "svc.IB",
		Lifetime:    plan.Singleton,
		FactoryExpr: "func(p godi.Provider) svc.IB { return svc.NewB(godi.GetRequired[svc.IA](p)) }",
	})

	src, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	code := string(src)

	getA := methodBody(t, code, "func (r *Root) get_A() svc.IA {")
	testutil.True(t, strings.Contains(getA, `panic(godi.CircularDependencyError("A"))`),
		"get_A must panic with CircularDependencyError while A is being constructed:\n"+getA)
	testutil.True(t, strings.Contains(recoverBranch(t, code), "godi.BuildError(rec)"),
		"NewRoot must return godi.BuildError(rec) for a construction panic")
}

// TestGenerate_NewRootReleasesTrackedDisposablesOnFailure: when construction
// fails part-way, the disposables tracked so far must be released before NewRoot
// returns the error — nobody else holds a reference to close them.
func TestGenerate_NewRootReleasesTrackedDisposablesOnFailure(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:       "Conn",
		GoType:     "*svc.Conn",
		Ctor:       "svc.NewConn",
		Lifetime:   plan.Singleton,
		Disposable: true,
	})
	reg.Service(plan.Service{
		Name:     "Repo",
		GoType:   "*svc.Repo",
		Ctor:     "svc.NewRepo",
		Lifetime: plan.Singleton,
		Deps:     []string{"Conn"},
		CtorErr:  true,
	})

	src, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	branch := recoverBranch(t, string(src))
	testutil.True(t, strings.Contains(branch, "_ = r.disp.CloseAll()"),
		"NewRoot's recover branch must release tracked disposables:\n"+branch)
	testutil.True(t, strings.Contains(branch, "root, err = nil, godi.BuildError(rec)"),
		"NewRoot's recover branch must return nil root and the build error:\n"+branch)
}
