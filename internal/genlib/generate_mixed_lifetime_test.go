package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// caseBranch returns the body of the `case <label>:` branch in body, up to the
// next case label or the end of the enclosing switch at the same indentation.
func caseBranch(t *testing.T, body, indent, label string) string {
	t.Helper()
	head := indent + "case " + label + ":\n"
	idx := strings.Index(body, head)
	if idx < 0 {
		t.Fatalf("missing branch %q in:\n%s", label, body)
	}
	branch := body[idx+len(head):]
	end := len(branch)
	for _, stop := range []string{"\n" + indent + "case ", "\n" + indent + "}"} {
		if i := strings.Index(branch, stop); i >= 0 && i < end {
			end = i
		}
	}
	return strings.TrimSpace(branch[:end])
}

func mixedLifetimeRegistry() *plan.Registry {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{Name: "S", GoType: "svc.IMixed", Ctor: "svc.NewS", Lifetime: plan.Singleton})
	reg.Service(plan.Service{Name: "T", GoType: "svc.IMixed", Ctor: "svc.NewT", Lifetime: plan.Transient})
	reg.Service(plan.Service{
		Name: `KC#"k"`, GoType: "svc.IKeyed", Ctor: "svc.NewKC", Lifetime: plan.Scoped, Key: `"k"`,
	})
	reg.Service(plan.Service{
		Name: `KT#"k"`, GoType: "svc.IKeyed", Ctor: "svc.NewKT", Lifetime: plan.Transient, Key: `"k"`,
	})
	reg.Service(plan.Service{
		Name: `KT2#"r"`, GoType: "svc.IKeyed", Ctor: "svc.NewKT2", Lifetime: plan.Transient, Key: `"r"`,
	})
	reg.Service(plan.Service{
		Name: `KC2#"r"`, GoType: "svc.IKeyed", Ctor: "svc.NewKC2", Lifetime: plan.Scoped, Key: `"r"`,
	})
	return reg
}

// TestGenerate_MixedLifetime_GetServiceReturnsLastRegistered: one GoType
// registered singleton "S" then transient "T". The single resolve returns the
// last registration (T), the collection resolve lists both in registration
// order, each by its own lifetime; on Root and on a scope, unkeyed and keyed.
func TestGenerate_MixedLifetime_GetServiceReturnsLastRegistered(t *testing.T) {
	src, err := genlib.Generate(mixedLifetimeRegistry(), "codegen")
	testutil.NoError(t, err)
	code := string(src)
	const mixed = "reflect.TypeFor[svc.IMixed]()"
	const keyed = "reflect.TypeFor[svc.IKeyed]()"

	rootGet := methodBody(t, code, "func (r *Root) GetService(")
	testutil.Equal(t, caseBranch(t, rootGet, "\t", mixed), "return r.NewT()")
	scopeGet := methodBody(t, code, "func (s *scope) GetService(")
	testutil.Equal(t, caseBranch(t, scopeGet, "\t", mixed), "return s.NewT()")

	rootAll := methodBody(t, code, "func (r *Root) GetKeyedServices(")
	testutil.Equal(t, caseBranch(t, rootAll, "\t\t", mixed), "return []any{r.get_S(), r.NewT()}")
	scopeAll := methodBody(t, code, "func (s *scope) GetKeyedServices(")
	testutil.Equal(t, caseBranch(t, scopeAll, "\t\t", mixed), "return []any{s.root.get_S(), s.NewT()}")

	// Keyed: (svc.IKeyed, "k") is scoped KC then transient KT; (svc.IKeyed,
	// "r") is transient KT2 then scoped KC2.
	keyBranch := func(body, key string) string {
		return caseBranch(t, caseBranch(t, body, "\t", keyed), "\t\t", key)
	}
	rootKeyed := methodBody(t, code, "func (r *Root) GetKeyedService(")
	testutil.Equal(t, keyBranch(rootKeyed, `"k"`), "return r.NewKT__k_()")
	testutil.Equal(t, keyBranch(rootKeyed, `"r"`), "return r.get_KC2__r_()")
	scopeKeyed := methodBody(t, code, "func (s *scope) GetKeyedService(")
	testutil.Equal(t, keyBranch(scopeKeyed, `"k"`), "return s.NewKT__k_()")
	testutil.Equal(t, keyBranch(scopeKeyed, `"r"`), "return s.get_KC2__r_()")
	concreteKeys := func(body string) string {
		body = body[strings.Index(body, "if key == godi.AnyKey"):]
		return body[strings.Index(body, "\n\tswitch t {"):]
	}
	rootKeyedAll := concreteKeys(rootAll)
	testutil.Equal(t, keyBranch(rootKeyedAll, `"k"`), "return []any{r.get_KC__k_(), r.NewKT__k_()}")
	testutil.Equal(t, keyBranch(rootKeyedAll, `"r"`), "return []any{r.NewKT2__r_(), r.get_KC2__r_()}")
	scopeKeyedAll := concreteKeys(scopeAll)
	testutil.Equal(t, keyBranch(scopeKeyedAll, `"k"`), "return []any{s.get_KC__k_(), s.NewKT__k_()}")
	testutil.Equal(t, keyBranch(scopeKeyedAll, `"r"`), "return []any{s.NewKT2__r_(), s.get_KC2__r_()}")
}

// TestGenerate_MixedLifetime_TransientWithErrorIsBoundInCollection: an
// error-returning transient interleaved with a singleton is bound before the
// collection literal, keeping registration order.
func TestGenerate_MixedLifetime_TransientWithErrorIsBoundInCollection(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{Name: "T", GoType: "svc.IMixed", Ctor: "svc.NewT", Lifetime: plan.Transient, CtorErr: true})
	reg.Service(plan.Service{Name: "S", GoType: "svc.IMixed", Ctor: "svc.NewS", Lifetime: plan.Singleton})
	src, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	code := string(src)

	rootGet := methodBody(t, code, "func (r *Root) GetService(")
	testutil.Equal(t, caseBranch(t, rootGet, "\t", "reflect.TypeFor[svc.IMixed]()"), "return r.get_S()")
	rootAll := methodBody(t, code, "func (r *Root) GetKeyedServices(")
	branch := caseBranch(t, rootAll, "\t", "reflect.TypeFor[svc.IMixed]()")
	testutil.True(t, strings.Contains(branch, "v0, err := r.NewT()"), "the error-returning transient must be bound first:\n"+branch)
	testutil.True(t, strings.HasSuffix(branch, "return []any{v0, r.get_S()}"), "the collection must keep registration order:\n"+branch)
}
