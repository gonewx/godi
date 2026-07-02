// ATDD test for Story 6.6 (AC2, Task 2, D-2) — godi package: the
// GetRequiredPanic[T] helper the generated container's GetRequired[T]() method
// calls to raise the miss panic without pulling a new "fmt" import into
// generated code (single-service.golden currently imports only "reflect" and
// "github.com/gonewx/godi" — D-2).
//
// Activated in Story 6.6 Task 2 (GetRequiredPanic[T] added to get.go;
// original //go:build atdd_redphase removed).
package godi_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gonewx/godi"
	"github.com/gonewx/godi/internal/testutil"
)

// --- 6.6-UNIT-001 (AC2/D-2, P0) — GetRequiredPanic panics -------------------

// TestGetRequiredPanic_Panics: calling GetRequiredPanic[T] always panics (it
// is unconditional — the generated GetRequired[T] method only calls it on a
// confirmed miss, after Get[T] already reported !ok).
func TestGetRequiredPanic_Panics(t *testing.T) {
	testutil.Panics(t, func() {
		godi.GetRequiredPanic[int]()
	})
}

// --- 6.6-UNIT-002 (AC2/D-2, P0) — panic value names the missing type -------

// TestGetRequiredPanic_MessageNamesType: the panic value must be an error
// whose message names the missing type, matching GetRequired[T]'s panic
// shape (get.go:47): fmt.Errorf("godi: no service registered for type %s",
// reflect.TypeFor[T]()).
func TestGetRequiredPanic_MessageNamesType(t *testing.T) {
	recovered := testutil.Panics(t, func() {
		godi.GetRequiredPanic[int]()
	})

	err, ok := recovered.(error)
	testutil.True(t, ok, "GetRequiredPanic must panic with an error value, not a bare string")
	if !ok {
		return
	}
	wantType := reflect.TypeFor[int]().String()
	testutil.True(t, strings.Contains(err.Error(), wantType),
		"panic message must name the missing type ("+wantType+"): got "+err.Error())
	testutil.True(t, strings.Contains(err.Error(), "godi: no service registered for type"),
		"panic message must match the GetRequired[T] miss message shape (get.go:47): got "+err.Error())
}

// --- 6.6-UNIT-003 (AC3, P1) — message parity with free GetRequired[T] ------

// TestGetRequiredPanic_ParityWithFreeGetRequired: GetRequiredPanic[T]'s panic
// message must be byte-identical to what the free function GetRequired[T]
// panics with for the same T, on a Provider with nothing registered — one
// message source for both call paths (AC3).
func TestGetRequiredPanic_ParityWithFreeGetRequired(t *testing.T) {
	var freeMsg string
	func() {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					freeMsg = err.Error()
				}
			}
		}()
		godi.GetRequired[int](emptyProvider{})
	}()
	testutil.True(t, freeMsg != "", "godi.GetRequired[int] must have panicked with an error")

	recovered := testutil.Panics(t, func() {
		godi.GetRequiredPanic[int]()
	})
	err, ok := recovered.(error)
	testutil.True(t, ok, "GetRequiredPanic must panic with an error value")
	if !ok {
		return
	}
	testutil.Equal(t, err.Error(), freeMsg)
}

// emptyProvider is a minimal godi.Provider with nothing registered — enough
// to drive GetRequired[T]'s miss path without depending on the runtime or
// codegen provider packages (keeps this scaffold single-package, mirroring
// get_test.go's existing fakes).
type emptyProvider struct{}

func (emptyProvider) GetService(reflect.Type) any              { return nil }
func (emptyProvider) IsService(reflect.Type) bool              { return false }
func (emptyProvider) IsKeyedService(reflect.Type, any) bool    { return false }
func (emptyProvider) GetKeyedService(reflect.Type, any) any    { return nil }
func (emptyProvider) GetKeyedServices(reflect.Type, any) []any { return nil }
func (emptyProvider) CreateScope() godi.Scope                  { return nil }
func (emptyProvider) Close() error                             { return nil }
