package genlib_test

import (
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/genlib"
	"github.com/gonewx/godi/internal/testutil"
	"github.com/gonewx/godi/plan"
)

// methodBody returns the emitted source of the function whose header starts
// with header, up to (not including) the next top-level func.
func methodBody(t *testing.T, code, header string) string {
	t.Helper()
	idx := strings.Index(code, header)
	testutil.True(t, idx >= 0, "missing emitted function: "+header)
	body := code[idx:]
	if next := strings.Index(body[1:], "\nfunc "); next > 0 {
		body = body[:next+1]
	}
	return body
}

// TestGenerate_FormBServicesAreTrackedDynamically: a form-B factory's product
// type is unknown at generation time (Disposable/AsyncDisposable stay false), so
// every form-B product — singleton, scoped (Root-as-scope, Root.CreateScope,
// scope.CreateScope) and transient (Root and scope New{T}) — is handed to
// disp.TrackAny, which asserts io.Closer / godi.AsyncDisposable at run time.
// CloseAsync is always emitted for such a container, and "io" is never imported.
func TestGenerate_FormBServicesAreTrackedDynamically(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:        "Single",
		GoType:      "svc.ISingle",
		Lifetime:    plan.Singleton,
		FactoryExpr: "func(p godi.Provider) svc.ISingle { return svc.NewSingle() }",
	})
	reg.Service(plan.Service{
		Name:        "Scoped",
		GoType:      "svc.IScoped",
		Lifetime:    plan.Scoped,
		FactoryExpr: "func(p godi.Provider) svc.IScoped { return svc.NewScoped() }",
	})
	reg.Service(plan.Service{
		Name:        "Trans",
		GoType:      "svc.ITrans",
		Lifetime:    plan.Transient,
		FactoryExpr: "func(p godi.Provider) svc.ITrans { return svc.NewTrans() }",
	})

	src, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, strings.Count(code, ".disp.TrackAny(") >= 3,
		"every form-B product must be tracked via disp.TrackAny")
	testutil.True(t, strings.Contains(methodBody(t, code, "func (r *Root) get_Single("), "r.disp.TrackAny(v)"),
		"the singleton accessor must track the form-B singleton")
	testutil.True(t, strings.Contains(methodBody(t, code, "func NewRoot("), "r.disp.TrackAny(r.f_Scoped)"),
		"NewRoot must track the Root-as-scope form-B scoped instance")
	testutil.True(t, strings.Contains(methodBody(t, code, "func (r *Root) CreateScope("), "s.disp.TrackAny(s.f_Scoped)"),
		"Root.CreateScope must track the form-B scoped instance")
	testutil.True(t, strings.Contains(methodBody(t, code, "func (s *scope) CreateScope("), "ns.disp.TrackAny(ns.f_Scoped)"),
		"scope.CreateScope must track the form-B scoped instance")
	testutil.True(t, strings.Contains(methodBody(t, code, "func (r *Root) NewTrans("), "r.disp.TrackAny(v)"),
		"Root.NewTrans must track the form-B transient")
	testutil.True(t, strings.Contains(methodBody(t, code, "func (s *scope) NewTrans("), "s.disp.TrackAny(v)"),
		"scope.NewTrans must track the form-B transient")

	testutil.True(t, strings.Contains(code, "func (r *Root) CloseAsync("),
		"a container with form-B services must emit Root.CloseAsync")
	testutil.True(t, !strings.Contains(code, `"io"`),
		"generated container must not import io")
}

// TestGenerate_DisposableTransientOnly_NoIoImport: when the only disposable is a
// form-A transient, the old emit imported "io" but used it nowhere (the ctor
// path passes v directly), so the container did not compile.
func TestGenerate_DisposableTransientOnly_NoIoImport(t *testing.T) {
	reg := plan.NewRegistry()
	reg.Imports = []string{"example.com/svc"}
	reg.Service(plan.Service{
		Name:       "TempFile",
		GoType:     "svc.ITempFile",
		Ctor:       "svc.NewTempFile",
		Lifetime:   plan.Transient,
		Disposable: true,
	})

	src, err := genlib.Generate(reg, "codegen")
	testutil.NoError(t, err)
	code := string(src)

	testutil.True(t, !strings.Contains(code, `"io"`),
		"a container whose only disposable is a transient must not import io")
	testutil.True(t, strings.Contains(code, "r.disp.TrackAny(v)"),
		"the disposable transient must be tracked via disp.TrackAny")
}
