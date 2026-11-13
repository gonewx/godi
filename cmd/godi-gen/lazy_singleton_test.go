// Lazy singleton construction (evaluation report §7.5, findings A2/B1/B2).
// Form-B factories are opaque to the generator, so no static order can be
// trusted for singletons; each fixture below is generated, compiled, and run
// for real through its committed behavior_test.go.
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runLazyFixture generates the container for testdata/<dir> next to its
// registration source (removed on cleanup) and runs the fixture's own
// behavior_test.go against it in a GOWORK=off subprocess.
func runLazyFixture(t *testing.T, dir string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping subprocess (go test) behavior check in -short mode")
	}
	fixture := filepath.Join("testdata", dir)
	out := filepath.Join(fixture, "container_gen.go")
	src, err := generate(fixture, dir, out, develVersion)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(out) })
	if err := os.WriteFile(out, src, 0o644); err != nil {
		t.Fatalf("write container: %v", err)
	}

	cmd := exec.Command("go", "test", "-count=1", "./"+fixture)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if outp, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s behavior test failed: %v\n%s", dir, err, outp)
	}
}

// TestLazySingleton_NamedFactorySeesLaterRegisteredDependency (A2): a named
// function factory registered before the singleton it resolves must observe the
// built instance, not a nil field.
func TestLazySingleton_NamedFactorySeesLaterRegisteredDependency(t *testing.T) {
	runLazyFixture(t, "lazynamedfactory")
}

// TestLazySingleton_UnkeyedFactoryResolvesKeyedSingleton (B1): an unkeyed
// factory resolving a keyed singleton must not depend on keyed singletons being
// wired first.
func TestLazySingleton_UnkeyedFactoryResolvesKeyedSingleton(t *testing.T) {
	runLazyFixture(t, "lazykeyeddep")
}

// TestLazySingleton_MissingDependencyBecomesNewRootError (B2): a factory that
// resolves an unregistered service makes NewRoot return an error, not panic.
func TestLazySingleton_MissingDependencyBecomesNewRootError(t *testing.T) {
	runLazyFixture(t, "lazymissing")
}

// TestLazySingleton_ConstructorErrorsBecomeNewRootError: error-returning
// transient dependencies (scalar and collection) of a singleton accessor and a
// root-level scoped constructor must compile and build; a singleton constructor
// error is returned from NewRoot wrapped (errors.Is) after the tracked
// disposables built so far are released.
func TestLazySingleton_ConstructorErrorsBecomeNewRootError(t *testing.T) {
	runLazyFixture(t, "lazyctorerr")
}

// TestTransient_DuplicateErrorReturningDependency (F-2): a transient taking the
// same error-returning transient twice must generate (one temporary per
// argument position, as in the lazy accessors) and receive two distinct
// instances.
func TestTransient_DuplicateErrorReturningDependency(t *testing.T) {
	runLazyFixture(t, "lazytransientdupdep")
}

// emittedFunc returns the emitted source of the function whose header starts
// with header, up to (not including) the next top-level func.
func emittedFunc(t *testing.T, code, header string) string {
	t.Helper()
	idx := strings.Index(code, header)
	if idx < 0 {
		t.Fatalf("missing emitted function: %s", header)
	}
	body := code[idx:]
	if next := strings.Index(body[1:], "\nfunc "); next > 0 {
		body = body[:next+1]
	}
	return body
}
