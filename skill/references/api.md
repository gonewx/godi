# godi Public API Reference

Module: `github.com/gonewx/godi`

## Types

### Registry

```go
type Registry struct{}
```

The handle a registration file writes against. Pure static marker — carries no runtime state.

### Provider

```go
type Provider interface {
    GetService(t reflect.Type) any
    IsService(t reflect.Type) bool
    IsKeyedService(t reflect.Type, key any) bool
    GetKeyedService(t reflect.Type, key any) any
    GetKeyedServices(t reflect.Type, key any) []any
    CreateScope() Scope
    Close() error
}
```

The single shared runtime contract. Both the runtime oracle and the generated container implement it.

- `GetService` — returns nil on miss (never panics)
- `IsService` — pure query, never panics
- `IsKeyedService` — nil key → IsService; AnyKey → false (never panics)
- `GetKeyedService` — nil key → GetService; AnyKey → panics
- `GetKeyedServices` — nil key → unkeyed namespace; AnyKey → all keyed registrations
- `CreateScope` — panics with `ErrProviderClosed` if closed
- `Close` — idempotent

### Scope

```go
type Scope interface {
    Provider
}
```

A child resolution scope. A scope IS a Provider.

### AsyncDisposable

```go
type AsyncDisposable interface {
    CloseAsync(ctx context.Context) error
}
```

Opt-in async-release contract. Discovered by type assertion, not part of the Provider interface.

### AsyncScope

```go
type AsyncScope struct{ /* unexported */ }

func NewAsyncScope(s Scope) AsyncScope
func (a AsyncScope) Provider() Provider
func (a AsyncScope) Close() error
func (a AsyncScope) CloseAsync(ctx context.Context) error
```

Value-semantics wrapper for async scope disposal.

### MarkedConstructor

```go
type MarkedConstructor struct{ Fn any }

func Marked(fn any) MarkedConstructor
```

Marks a constructor for activator best-match selection.

### CloseGuard

```go
type CloseGuard struct{ /* unexported */ }

func (g *CloseGuard) Closed() bool
func (g *CloseGuard) Close(release func() error) error
```

Shared close-once state of a provider or scope, used by the runtime provider and by every generated `Root` and scope (as a `guard godi.CloseGuard` field). The zero value is an open guard; it must not be copied after first use.

- `Closed` — reports whether `Close` has started
- `Close` — the first caller marks the guard closed, runs `release` and caches its error; any other caller (including a re-entrant one from inside `release`) returns the cached error, or nil if `release` has not finished yet. Never blocks.

## Registration Functions (Static Markers)

All return `*Registry` for chaining. None execute at runtime.

### Unkeyed — Form A (constructor discovery)

```go
func AddSingleton[I any, Impl any](r *Registry) *Registry
func AddScoped[I any, Impl any](r *Registry) *Registry
func AddTransient[I any, Impl any](r *Registry) *Registry
```

### Unkeyed — Form B (factory escape hatch)

```go
func AddSingletonFactory[I any](r *Registry, factory func(Provider) I) *Registry
func AddScopedFactory[I any](r *Registry, factory func(Provider) I) *Registry
func AddTransientFactory[I any](r *Registry, factory func(Provider) I) *Registry
```

### Keyed — Form A

```go
func AddKeyedSingleton[I any, Impl any](r *Registry, key any) *Registry
func AddKeyedScoped[I any, Impl any](r *Registry, key any) *Registry
func AddKeyedTransient[I any, Impl any](r *Registry, key any) *Registry
```

### Keyed — Form B

```go
func AddKeyedSingletonFactory[I any](r *Registry, key any, factory func(Provider) I) *Registry
func AddKeyedScopedFactory[I any](r *Registry, key any, factory func(Provider) I) *Registry
func AddKeyedTransientFactory[I any](r *Registry, key any, factory func(Provider) I) *Registry
```

## Resolution Functions

### Unkeyed

```go
func Get[T any](p Provider) (T, bool)
func GetRequired[T any](p Provider) T           // panics on miss
func GetServices[T any](p Provider) []T          // empty slice on miss (never nil)
```

### Keyed

```go
func GetKeyed[T any](p Provider, key any) (T, bool)
func GetRequiredKeyed[T any](p Provider, key any) T   // panics on miss
func GetKeyedServices[T any](p Provider, key any) []T  // empty slice on miss
```

## Sentinels & Errors

```go
var AnyKey                                       // query-only sentinel for GetKeyedServices
var ErrProviderClosed = errors.New("...")         // panicked on resolve-after-Close

func AnyKeySingleResolveError(t reflect.Type) error  // GetKeyedService + AnyKey
func AnyKeyRegistrationError(t reflect.Type) error    // AnyKey used as registration key
func ProviderClosedError() error                      // returns ErrProviderClosed
func GetRequiredPanic[T any]()                        // used by generated code

var ErrCircularDependency = errors.New("godi: circular dependency")
func CircularDependencyError(service string) error    // wraps ErrCircularDependency; used by generated code
func BuildError(recovered any) error                  // NewRoot's error for a value recovered during construction; used by generated code
```

- `ErrCircularDependency` — a cycle through form-B factories is detected at construction time; `NewRoot()`'s error or the resolution panic value matches it with `errors.Is`
- `BuildError` — an error value stays reachable through `errors.Is` / `errors.As`; any other value is formatted with `%v`

## Async Helpers

```go
func CreateAsyncScope(p Provider) AsyncScope     // NewAsyncScope(p.CreateScope())
```

## Runtime Helpers

Package `github.com/gonewx/godi/runtime`, consumed by generated containers.

```go
func (d *Disposer) TrackAny(v any)   // tracks v by its dynamic type: io.Closer → Track, else godi.AsyncDisposable → TrackAsync; nil or neither → not tracked
```

## Generator CLI

```
godi-gen -pkg <name> [-out <file>] <srcDir>
```

- `<srcDir>` — directory of the registration package
- `-pkg` — package name for the emitted container (required)
- `-out` — output file path (stdout if omitted)
- Exit 0 + output on success; exit 1 + zero bytes on validation failure (NFR2)
