package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// scopedRegistry has one singleton, two scoped services (one with an
// error-returning constructor that depends on an error-returning transient), a
// keyed scoped and a keyed transient both with error-returning constructors.
func scopedRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{Name: "Log", GoType: "*svc.Log", Ctor: "svc.NewLog", Lifetime: plan.Singleton})
	reg.Service(plan.Service{
		Name: "Req", GoType: "svc.IReq", Ctor: "svc.NewReq", Lifetime: plan.Scoped,
		Deps: []string{"Log", "Tx"}, CtorErr: true, Disposable: true,
	})
	reg.Service(plan.Service{Name: "Tx", GoType: "svc.ITx", Ctor: "svc.NewTx", Lifetime: plan.Scoped})
	reg.Service(plan.Service{Name: "Tmp", GoType: "svc.ITmp", Ctor: "svc.NewTmp", Lifetime: plan.Transient, CtorErr: true})
	reg.Service(plan.Service{
		Name: "Uow", GoType: "svc.IUow", Ctor: "svc.NewUow", Lifetime: plan.Scoped,
		Deps: []string{"Tmp"},
	})
	reg.Service(plan.Service{
		Name: `KFoo#"k"`, GoType: "svc.IFoo", Ctor: "svc.NewFoo", Lifetime: plan.Scoped,
		Key: `"k"`, CtorErr: true,
	})
	reg.Service(plan.Service{
		Name: `KBar#"t"`, GoType: "svc.IBar", Ctor: "svc.NewBar", Lifetime: plan.Transient,
		Key: `"t"`, CtorErr: true,
	})
	return reg
}

// TestGenerate_ScopedAccessorResetsStateAfterPanic: a scope's scoped accessor
// resets its state when construction panics, so a retry re-runs the
// constructor instead of reporting a cycle.
func TestGenerate_ScopedAccessorResetsStateAfterPanic(t *testing.T) {
	src, err := genlib.Generate(scopedRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	for _, name := range []string{"Req", "Tx", "Uow", `KFoo__k_`} {
		body := methodBody(t, code, "func (s *scope) get_"+name+"() ")
		testutil.True(t, strings.Contains(body, "defer func() {"),
			"scope get_"+name+" must defer the state reset:\n"+body)
		testutil.True(t, strings.Contains(body, "s.st_"+name+" = 0"),
			"scope get_"+name+" must reset its state after a failed build:\n"+body)
	}
}

// TestGenerate_ScopedServicesUseLazyAccessors: every scoped service — keyed or
// not — has a state field and an accessor on both Root and scope, reads of a
// scoped service go through the accessor, and neither NewRoot nor CreateScope
// builds a scoped service.
func TestGenerate_ScopedServicesUseLazyAccessors(t *testing.T) {
	src, err := genlib.Generate(scopedRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)

	for _, n := range []string{"Req", "Tx", "Uow", "KFoo__k_"} {
		for _, recv := range []string{"r *Root", "s *scope"} {
			methodBody(t, code, "func ("+recv+") get_"+n+"() ")
		}
		testutil.True(t, strings.Count(code, "\tst_"+n+" ") == 2,
			"Root and scope must each declare st_"+n)
	}

	newRoot := methodBody(t, code, "func NewRoot(")
	testutil.True(t, !strings.Contains(newRoot, "return nil, err"),
		"NewRoot must fail only through its recover branch:\n"+newRoot)
	for _, n := range []string{"Req", "Tx", "Uow", "KFoo__k_"} {
		testutil.True(t, !(strings.Contains(newRoot, "get_"+n+"()") || strings.Contains(newRoot, "f_"+n)),
			"NewRoot must not build scoped "+n+":\n"+newRoot)
	}
	for _, h := range []string{"func (r *Root) CreateScope() ", "func (s *scope) CreateScope() "} {
		body := methodBody(t, code, h)
		testutil.True(t, !(strings.Contains(body, "get_") || strings.Contains(body, ".f_") || strings.Contains(body, "svc.New")),
			h+" must only allocate the scope:\n"+body)
	}

	// The scope accessor reads its singleton through Root and its scoped
	// dependency through the scope's own accessor; the constructor error panics.
	req := methodBody(t, code, "func (s *scope) get_Req() ")
	testutil.True(t, strings.Contains(req, "svc.NewReq(s.root.get_Log(), s.get_Tx())"),
		"scope get_Req must read deps through accessors:\n"+req)
	testutil.True(t, strings.Contains(req, "panic(err)"), "constructor error must panic:\n"+req)
	testutil.True(t, strings.Contains(req, "s.disp.TrackAny(v)"), "scope get_Req must track into the scope disposer:\n"+req)
	rootReq := methodBody(t, code, "func (r *Root) get_Req() ")
	testutil.True(t, strings.Contains(rootReq, "r.disp.TrackAny(v)"), "Root get_Req must track into Root's disposer:\n"+rootReq)
	uow := methodBody(t, code, "func (s *scope) get_Uow() ")
	testutil.True(t, strings.Contains(uow, "d0, err := s.NewTmp()"),
		"an error-returning transient dependency is bound to d0:\n"+uow)

	// No read of a scoped field outside its accessor.
	for _, n := range []string{"Req", "Tx", "Uow", "KFoo__k_"} {
		rest := code
		for _, recv := range []string{"r *Root", "s *scope"} {
			body := methodBody(t, code, "func ("+recv+") get_"+n+"() ")
			rest = strings.Replace(rest, body, "", 1)
		}
		for _, field := range []string{"r.f_" + n, "s.f_" + n} {
			testutil.True(t, !strings.Contains(rest, field),
				field+" is read outside its accessor")
		}
	}

	// A keyed transient with an error-returning constructor returns the error
	// from New<N>, and keyed resolution panics with it.
	testutil.True(t, strings.Contains(code, "func (s *scope) NewKBar__t_() (svc.IBar, error) {"),
		"keyed transient New method must return the constructor error")
	gks := methodBody(t, code, "func (s *scope) GetKeyedService(")
	testutil.True(t, strings.Contains(gks, "v, err := s.NewKBar__t_()"),
		"keyed GetKeyedService must bind the transient error:\n"+gks)
}
