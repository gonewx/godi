package codegen_test

import (
	"reflect"
	"testing"

	"github.com/gonewx/godi/conformance/codegen"
	"github.com/gonewx/godi/conformance/fakes"
	"github.com/gonewx/godi/internal/testutil"
)

// TestCodegen_FormBDisposablesAreReleased: form-B factory products that
// implement io.Closer are released by the generated container exactly like the
// runtime provider releases them — the scoped one on scope.Close, the
// singleton on root.Close (and not before).
func TestCodegen_FormBDisposablesAreReleased(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)

	scope := root.CreateScope()
	scoped, ok := scope.GetService(reflect.TypeFor[fakes.IFakeFormBDisposableScoped]()).(fakes.IFakeFormBDisposableScoped)
	testutil.True(t, ok, "form-B scoped service must resolve from the scope")
	singleton, ok := scope.GetService(reflect.TypeFor[fakes.IFakeFormBDisposableSingleton]()).(fakes.IFakeFormBDisposableSingleton)
	testutil.True(t, ok, "form-B singleton must resolve from the scope")

	testutil.NoError(t, scope.Close())
	testutil.Equal(t, scoped.CloseCount(), 1)
	testutil.Equal(t, singleton.CloseCount(), 0)

	testutil.NoError(t, root.Close())
	testutil.Equal(t, singleton.CloseCount(), 1)
	testutil.Equal(t, scoped.CloseCount(), 1)
}

// TestCodegen_FormBNonDisposableProductIsHarmless: a form-B product that is not
// disposable (IFakeInstanceService) is handed to TrackAny too; it must simply be
// skipped, so root.Close succeeds.
func TestCodegen_FormBNonDisposableProductIsHarmless(t *testing.T) {
	root, err := codegen.NewRoot()
	testutil.NoError(t, err)

	v := root.GetService(reflect.TypeFor[fakes.IFakeInstanceService]())
	testutil.NotNil(t, v)

	testutil.NoError(t, root.Close())
}
