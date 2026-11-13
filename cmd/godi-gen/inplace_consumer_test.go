// In-place layout with same-package consumers. The registration package also
// holds godi-gen's previous product, and hand-written code in that package
// refers to it (Root, NewRoot, typed accessors). Loading must keep the previous
// container's declarations visible so those references resolve, ignore only the
// errors located in generated files, and still report real errors in
// hand-written files. Rows refer to the acceptance matrix of the final fix wave
// (F-1); rows 2 and 4 are covered by stale_output_test.go.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"golang.org/x/tools/go/packages"
)

const inplaceDir = "testdata/inplaceconsumer"

// runtimeFileSeq makes runtimeFileName unique within one test process; the pid
// makes it unique across processes, so concurrent runs never remove each
// other's files.
var runtimeFileSeq atomic.Int64

// runtimeFileName returns a per-run unique .go file name under dir. Generated
// files are recognized by their first line, never by their name.
func runtimeFileName(dir, prefix string) string {
	return filepath.Join(dir, fmt.Sprintf("%s_%d_%d.go", prefix, os.Getpid(), runtimeFileSeq.Add(1)))
}

// writeRuntimeFile registers path's removal, then writes body to it.
func writeRuntimeFile(t *testing.T, path, body string) {
	t.Helper()
	t.Cleanup(func() {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			t.Errorf("remove %s: %v", path, err)
		}
	})
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// consumerSrc is hand-written same-package code that uses the container.
const consumerSrc = `package inplaceconsumer

func MustRoot() *Root {
	r, err := NewRoot()
	if err != nil {
		panic(err)
	}
	return r
}

func UseFoo() IFoo { return MustRoot().Foo() }
`

// generateInPlace runs the generator in the in-place layout (-out inside the
// registration directory) and, on success, writes the container to out.
func generateInPlace(t *testing.T, out string) []byte {
	t.Helper()
	src, err := generate(inplaceDir, "inplaceconsumer", out)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	writeRuntimeFile(t, out, string(src))
	return src
}

// Row 1: a valid previous container plus a same-package consumer regenerates
// to byte-identical output.
func TestInPlace_ValidContainerWithConsumer(t *testing.T) {
	out := runtimeFileName(inplaceDir, "container_gen")
	first := generateInPlace(t, out)
	writeRuntimeFile(t, runtimeFileName(inplaceDir, "app"), consumerSrc)

	again, err := generate(inplaceDir, "inplaceconsumer", out)
	if err != nil {
		t.Fatalf("regenerate with a same-package consumer: %v", err)
	}
	if !bytes.Equal(again, first) {
		t.Fatalf("regenerated container differs from the previous one:\n%s", again)
	}
}

// Row 3: a stale previous container (it calls NewFoo() although NewFoo now
// takes an ILog) plus a same-package consumer regenerates with the new call.
func TestInPlace_StaleContainerWithConsumer(t *testing.T) {
	staleSrc, err := os.ReadFile(filepath.Join("testdata", "staleoutput", "container_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	stale := strings.Replace(string(staleSrc), "package staleoutput", "package inplaceconsumer", 1)
	if !strings.Contains(stale, "NewFoo()") {
		t.Fatal("precondition: the stale container must call NewFoo()")
	}
	absDir, err := filepath.Abs(inplaceDir)
	if err != nil {
		t.Fatal(err)
	}
	// Error positions are classified by absolute path whether srcDir is given
	// relative or absolute.
	for _, dir := range []string{inplaceDir, absDir} {
		t.Run(dir, func(t *testing.T) {
			out := runtimeFileName(inplaceDir, "container_gen")
			writeRuntimeFile(t, out, stale)
			writeRuntimeFile(t, runtimeFileName(inplaceDir, "app"), consumerSrc)

			got, err := generate(dir, "inplaceconsumer", out)
			if err != nil {
				t.Fatalf("stale container with a consumer must not block regeneration: %v", err)
			}
			if !bytes.Contains(got, []byte("v := NewFoo(r.get_Log())")) {
				t.Fatalf("regenerated container must call NewFoo with its ILog dependency, got:\n%s", got)
			}
		})
	}
}

// brokenRegSrc is a registration file with a real type error on line 8.
const brokenRegSrc = `package inplaceconsumer

import "github.com/gonewx/godi"

func registerBroken() {
	var r godi.Registry
	godi.AddSingleton[IBar, Bar](&r)
	var n int = "not an int"
	_ = n
}
`

// Row 5: a real type error in a hand-written registration file is reported
// with its file:line, whatever the directory holds — and the report is the
// first load's, not the stubbed retry's (whose undefined: Root errors in the
// consumer are an artifact of the stub).
func TestInPlace_RegistrationTypeErrorIsReported(t *testing.T) {
	cases := []struct {
		name      string
		container bool
		consumer  bool
	}{
		{"valid container and consumer", true, true},
		{"no container, no consumer", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.container {
				generateInPlace(t, runtimeFileName(inplaceDir, "container_gen"))
			}
			if tc.consumer {
				writeRuntimeFile(t, runtimeFileName(inplaceDir, "app"), consumerSrc)
			}
			bad := runtimeFileName(inplaceDir, "broken")
			writeRuntimeFile(t, bad, brokenRegSrc)

			out, err := generate(inplaceDir, "inplaceconsumer", "")
			if err == nil {
				t.Fatal("want a load/type-check error, got nil")
			}
			if out != nil {
				t.Fatalf("want zero bytes on rejection, got %d", len(out))
			}
			msg := err.Error()
			prefix := fmt.Sprintf("package %q failed to load/type-check:", inplaceDir)
			if !strings.HasPrefix(msg, prefix) {
				t.Fatalf("error must start with %q, got: %v", prefix, msg)
			}
			if want := filepath.Base(bad) + ":8:"; !strings.Contains(msg, want) {
				t.Fatalf("error must locate the type error at %s, got: %v", want, msg)
			}
			if strings.Contains(msg, "undefined: Root") || strings.Contains(msg, "undefined: NewRoot") {
				t.Fatalf("error must not report the stub's undefined Root/NewRoot, got: %v", msg)
			}
		})
	}
}

// Row 6: a consumer that calls a typed accessor the regenerated container no
// longer has is rejected by the output type-check gate, not by the load.
func TestInPlace_ConsumerOfDroppedAccessorFailsTypeCheck(t *testing.T) {
	extra := runtimeFileName(inplaceDir, "extra")
	writeRuntimeFile(t, extra, `package inplaceconsumer

import "github.com/gonewx/godi"

func registerBar() {
	var r godi.Registry
	godi.AddSingleton[IBar, Bar](&r)
}
`)
	out := runtimeFileName(inplaceDir, "container_gen")
	first := generateInPlace(t, out)
	if !bytes.Contains(first, []byte("func (r *Root) Bar() IBar")) {
		t.Fatalf("precondition: the previous container must have a Bar accessor, got:\n%s", first)
	}
	if err := os.Remove(extra); err != nil {
		t.Fatal(err)
	}
	writeRuntimeFile(t, runtimeFileName(inplaceDir, "app"), consumerSrc+`
func UseBar() IBar { return MustRoot().Bar() }
`)

	got, err := generate(inplaceDir, "inplaceconsumer", out)
	if err == nil {
		t.Fatal("want a type-check error for the dropped accessor, got nil")
	}
	if got != nil {
		t.Fatalf("want zero bytes on rejection, got %d", len(got))
	}
	if !strings.HasPrefix(err.Error(), "generated container does not type-check") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Row 7: a form-B factory that mentions *Root resolves against the previous
// container while loading and regenerates successfully.
func TestInPlace_FactoryMentioningRoot(t *testing.T) {
	out := runtimeFileName(inplaceDir, "container_gen")
	generateInPlace(t, out)
	writeRuntimeFile(t, runtimeFileName(inplaceDir, "factory"), `package inplaceconsumer

import "github.com/gonewx/godi"

func registerBarFactory() {
	var r godi.Registry
	godi.AddSingletonFactory[IBar](&r, func(p godi.Provider) IBar {
		_ = p.(*Root)
		return NewBar()
	})
}
`)

	got, err := generate(inplaceDir, "inplaceconsumer", out)
	if err != nil {
		t.Fatalf("factory mentioning *Root must regenerate: %v", err)
	}
	if !bytes.Contains(got, []byte("p.(*Root)")) {
		t.Fatalf("regenerated container must inline the factory, got:\n%s", got)
	}
}

// Row 8: a consumer without any container cannot load (as on the branch
// base); the error must point at the consumer file.
func TestInPlace_ConsumerWithoutContainerReportsConsumer(t *testing.T) {
	app := runtimeFileName(inplaceDir, "app")
	writeRuntimeFile(t, app, consumerSrc)

	out, err := generate(inplaceDir, "inplaceconsumer", "")
	if err == nil {
		t.Fatal("want a load error, got nil")
	}
	if out != nil {
		t.Fatalf("want zero bytes on rejection, got %d", len(out))
	}
	if want := filepath.Base(app) + ":4:"; !strings.Contains(err.Error(), want) {
		t.Fatalf("error must locate the consumer at %s, got: %v", want, err)
	}
}

// A previous container is visible to the load now, but it is the generator's
// own product, not registration input: markers inside a generated file are
// never collected.
func TestInPlace_MarkersInGeneratedFilesAreIgnored(t *testing.T) {
	writeRuntimeFile(t, runtimeFileName(inplaceDir, "zz_gen"), generatedHeader+`
package inplaceconsumer

import "github.com/gonewx/godi"

func registerGeneratedBar() {
	var r godi.Registry
	godi.AddSingleton[IBar, Bar](&r)
}
`)
	got, err := generate(inplaceDir, "inplaceconsumer", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if bytes.Contains(got, []byte("IBar")) {
		t.Fatalf("a marker inside a generated file must not be collected, got:\n%s", got)
	}
}

// -out into another existing directory (cross-package emission) type-checks
// the container as a file of that directory's package.
func TestGenerate_OutIntoAnotherPackage(t *testing.T) {
	out := filepath.Join("testdata", "crosspkgout", "container_gen.go")
	src, err := generate(filepath.Join("testdata", "deps"), "crosspkgout", out)
	if err != nil {
		t.Fatalf("cross-package -out: %v", err)
	}
	if !bytes.HasPrefix(src, []byte(generatedHeader+"\n")) {
		t.Fatalf("unexpected output:\n%s", src)
	}
}

// A cross-package -out container that does not compile is rejected with zero
// bytes, as in the other layouts.
func TestGenerate_OutIntoAnotherPackageRejectsUncompilable(t *testing.T) {
	out := filepath.Join("testdata", "crosspkgout", "container_gen.go")
	src, err := generate(filepath.Join("testdata", "tcnotimpl"), "crosspkgout", out)
	if err == nil {
		t.Fatal("want type-check error, got nil")
	}
	if src != nil {
		t.Fatalf("want zero bytes on rejection, got %d", len(src))
	}
	if !strings.HasPrefix(err.Error(), "generated container does not type-check") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// F-3: an -out whose directory does not exist is reported as such before any
// type-checking, not as a type error.
func TestGenerate_OutDirectoryMissing(t *testing.T) {
	missing := filepath.Join("testdata", "no-such-output-dir")
	src, err := generate(filepath.Join("testdata", "deps"), "depsfix", filepath.Join(missing, "container_gen.go"))
	if err == nil {
		t.Fatal("want an error for the missing output directory, got nil")
	}
	if src != nil {
		t.Fatalf("want zero bytes on rejection, got %d", len(src))
	}
	msg := err.Error()
	if strings.Contains(msg, "does not type-check") {
		t.Fatalf("a missing output directory is not a type error: %v", msg)
	}
	if !strings.Contains(msg, "does not exist") || !strings.Contains(msg, missing) {
		t.Fatalf("error must say the output directory %s does not exist, got: %v", missing, msg)
	}
}

// loadErrorFile resolves an error position to an absolute file path: go/types
// and parse errors carry absolute paths, go list errors may carry paths
// relative to the package directory, and some errors carry no position.
func TestLoadErrorFile(t *testing.T) {
	abs := filepath.Join(string(filepath.Separator)+"src", "pkg")
	file := filepath.Join(abs, "container_gen.go")
	for _, tc := range []struct {
		pos, want string
	}{
		{file + ":23:19", file},
		{file + ":23", file},
		{"container_gen.go:2:1", file},
		{"./container_gen.go:2:1", file},
		{"-", ""},
		{"", ""},
		{file, ""},
	} {
		if got := loadErrorFile(packages.Error{Pos: tc.pos}, abs); got != tc.want {
			t.Errorf("loadErrorFile(%q) = %q, want %q", tc.pos, got, tc.want)
		}
	}
}
