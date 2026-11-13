package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gonewx/godi/internal/testutil"
)

func TestGenerator_FormBConcretePointerServiceTypes(t *testing.T) {
	src, err := generate("testdata/concretefactory", "codegen", "")
	testutil.NoError(t, err)

	code := string(src)
	for _, want := range []string{
		"f_Ptr_Config *concretefactory.Config",
		"func (r *Root) NewPtr_Child() *concretefactory.Child",
		"case reflect.TypeFor[*concretefactory.Config]():",
		"case reflect.TypeFor[*concretefactory.Child]():",
		"godi.GetRequired[*concretefactory.Config](p)",
	} {
		testutil.True(t, strings.Contains(code, want), "missing generated concrete-pointer support: "+want)
	}
}

func TestGenerator_FormBConcretePointerOutputCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess (go vet) compile check in -short mode")
	}

	src, err := generate("testdata/concretefactory", "compilecheck", "")
	testutil.NoError(t, err)

	dir := filepath.Join("testdata", "concretefactory", "zz_compile_check")
	testutil.NoError(t, os.MkdirAll(dir, 0o755))
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	testutil.NoError(t, os.WriteFile(filepath.Join(dir, "container_gen.go"), src, 0o644))

	cmd := exec.Command("go", "vet", "./"+dir)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, vetErr := cmd.CombinedOutput()
	testutil.True(t, vetErr == nil, "concrete-pointer container must type-check; go vet said:\n"+string(out))
}
