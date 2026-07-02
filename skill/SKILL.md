---
name: use-godi
description: Guide for using the godi dependency-injection library and godi-gen code generator in Go projects. Use this skill whenever the user works with godi, dependency injection in Go, service registration, DI containers, scoped/transient/singleton services, keyed services, or needs to set up godi-gen code generation. Also trigger when the user mentions godi.Provider, godi.Registry, godi.AddSingleton, godi.Get, or any godi DSL function — even if they just say "wire up dependencies" or "add a service" in a Go project that imports github.com/gonewx/godi.
---

# godi — Go Dependency Injection

godi is a compile-time dependency-injection library for Go. Services are registered via a pure-static DSL (marker functions that are never executed at runtime), and a code generator (`godi-gen`) reads those registrations with `go/types` static analysis to emit a type-safe, reflect-free container.

Module path: `github.com/gonewx/godi`
Requires: Go 1.27+

## Core Concepts

### Static markers (AD-7)

Every `godi.Add*` function is a **pure static marker** — it returns `*Registry` for chaining but carries no runtime state. The generator reads the type arguments from `go/types` and **never executes** user registration code. This is the foundational invariant: if you see an `init()` panic in a registration file, that's an AD-7 tripwire proving non-execution.

### Two registration forms

**Form A** — constructor discovery: provide `[Interface, Implementation]` type arguments. The generator finds `New{Impl}()` automatically.

**Form B** — factory escape hatch: provide `[Interface]` plus a factory function. Use when the implementation has no `New{Impl}` constructor, or when you need custom wiring logic.

### Three lifetimes

| Lifetime | Storage | Behavior |
|---|---|---|
| Singleton | Root field | Created once, shared everywhere |
| Scoped | Scope field | Created once per scope, shared within that scope |
| Transient | Method (no field) | New instance on every resolution |

### Provider contract (AD-16)

`godi.Provider` is the single shared runtime interface — both the runtime oracle and the generated container implement it. Its method set is fixed:

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

`Scope` embeds `Provider` — a scope IS a provider.

## Registration DSL

### Form A — constructor-based

```go
import "github.com/gonewx/godi"

func register() {
    var r godi.Registry
    godi.AddSingleton[ILogger, Logger](&r)
    godi.AddScoped[IRequest, Request](&r)
    godi.AddTransient[IWidget, Widget](&r)
}
```

Requirements for form A:
- `Impl` must be a named type
- A constructor `New{Impl}()` must exist in the same package as `Impl`
- Constructor may return `(*Impl)` or `(*Impl, error)`
- Constructor parameters are resolved as dependencies automatically

### Form B — factory-based

```go
func register() {
    var r godi.Registry
    godi.AddSingletonFactory[IWidget](&r, func(p godi.Provider) IWidget {
        return NewWidget()
    })
    godi.AddScopedFactory[ICache](&r, func(p godi.Provider) ICache {
        logger, _ := godi.Get[ILogger](p)
        return NewCache(logger)
    })
    godi.AddTransientFactory[IHandler](&r, func(p godi.Provider) IHandler {
        return NewHandler()
    })
}
```

Use form B when:
- The implementation has no `New{Impl}` constructor
- You need custom wiring logic beyond simple constructor injection
- The factory argument expression is inlined verbatim into generated code

### Keyed services (AD-17)

Register multiple implementations of the same interface, distinguished by a compile-time constant key:

```go
func register() {
    var r godi.Registry
    godi.AddKeyedSingleton[ICache, RedisCache](&r, "redis")
    godi.AddKeyedSingleton[ICache, MemCache](&r, "memory")
    godi.AddKeyedScopedFactory[ICache](&r, "custom", func(p godi.Provider) ICache {
        return NewCustomCache()
    })
}
```

Key rules:
- Keys must be **compile-time constants** (string, int, etc.) — the generator emits them as `switch/case` arms
- A `nil` key routes to the unkeyed namespace (equivalent to the non-keyed `Add*` variant)
- `godi.AnyKey` is a **query-only sentinel** — it cannot be used as a registration key

### Open generic helpers (Mechanism A)

For generic service types, define a per-service helper that forwards to godi markers:

```go
func AddRepository[T any](r *godi.Registry) *godi.Registry {
    return godi.AddScoped[IRepository[T], Repository[T]](r)
}

func register() {
    var r godi.Registry
    AddRepository[User](&r)
    AddRepository[Order](&r)
}
```

The generator follows the helper, substitutes type arguments, and materializes the closed registrations.

A helper body may contain **multiple marker statements**, mixing Form A and Form B — the way to register a service bundle (e.g. the dotnet Options "four-piece" pattern) in one call:

```go
func AddOptions[T Entity[T]](r *godi.Registry) *godi.Registry {
    godi.AddSingleton[Options[T], UnnamedOptionsManager[T]](r)
    godi.AddSingletonFactory[OptionsMonitor[T]](r, func(p godi.Provider) OptionsMonitor[T] {
        return NewOptionsMonitor[T](
            godi.GetRequired[OptionsFactory[T]](p),
            godi.GetServices[ConfigureOptions[T]](p), // never-registered → empty slice, fine
            godi.GetRequired[OptionsMonitorCache[T]](p),
        )
    })
    godi.AddTransientFactory[OptionsFactory[T]](r, func(p godi.Provider) OptionsFactory[T] {
        return NewOptionsFactory[T](godi.GetServices[ConfigureOptions[T]](p))
    })
    return godi.AddSingleton[OptionsMonitorCache[T], OptionsCache[T]](r)
}
```

Guarantees:
- Form-B factory text is rewritten on closure: `T` → the concrete type, package qualifiers remapped to the generated file's imports (two service packages sharing a name get numbered aliases automatically).
- `godi.Get*` calls inside a factory body are read as **assembly-order dependencies** — a singleton factory may `GetRequired` a service registered later in the body (like the cache above); the generator assembles it first.

Limits (generation-time warning, never a silent drop): helper must live in the analyzed package, single type parameter, unkeyed markers only, and every body statement must be a marker call (or a final `return r`).

## Resolution API

### Free generic functions

```go
// Try-resolve (returns zero, false on miss)
svc, ok := godi.Get[ILogger](provider)

// Required resolve (panics on miss)
svc := godi.GetRequired[ILogger](provider)

// All registrations for a type (registration order)
all := godi.GetServices[ILogger](provider)

// Keyed resolution
svc, ok := godi.GetKeyed[ICache](provider, "redis")
svc := godi.GetRequiredKeyed[ICache](provider, "redis")
allKeyed := godi.GetKeyedServices[ICache](provider, godi.AnyKey)
```

### Generated container methods

The generated container also has typed accessor methods (zero-overhead, no reflection):

```go
root, err := container.NewRoot()
logger := root.Logger()              // typed accessor, no reflection
svc, ok := root.Get[ILogger]()       // generic method on the container
svc := root.GetRequired[ILogger]()   // panics on miss
```

## Code Generation Workflow

### 1. Create a registration file

In your service package, create a file with the registration markers:

```go
// myapp/registration.go
package myapp

import "github.com/gonewx/godi"

type ILogger interface{ Log(msg string) }
type Logger struct{}
func (*Logger) Log(msg string) {}
func NewLogger() *Logger { return &Logger{} }

type IService interface{ Do() }
type Service struct{ logger ILogger }
func (*Service) Do() {}
func NewService(logger ILogger) (*Service, error) { return &Service{logger: logger}, nil }

func register() {
    var r godi.Registry
    godi.AddSingleton[ILogger, Logger](&r)
    godi.AddSingleton[IService, Service](&r)
}
```

### 2. Add a go:generate directive

```go
// myapp/gen.go
package myapp

//go:generate godi-gen -pkg myapp -out container_gen.go .
```

### 3. Run the generator

```bash
# Install the generator
go install github.com/gonewx/godi/cmd/godi-gen@latest

# Generate
go generate ./...
```

### 4. Use the generated container

```go
root, err := myapp.NewRoot()
if err != nil {
    log.Fatal(err)
}
defer root.Close()

logger := root.Logger()
svc := root.Service()

// Or via the generic API:
logger2, ok := root.Get[ILogger]()

// Create a scope for request-scoped services:
scope := root.CreateScope()
defer scope.Close()
```

## Generated Code Structure

The generator emits:

- **`Root` struct** — holds singleton and scoped fields, a `Disposer` for lifecycle management
- **`NewRoot() (*Root, error)`** — constructs and wires all singletons eagerly in dependency order
- **Typed accessor methods** — `root.Logger()`, `root.Service()` — zero-overhead field returns
- **`Get[T]() (T, bool)` / `GetRequired[T]() T`** — generic methods on the container
- **`GetService(reflect.Type) any`** — `reflect.TypeFor[T]()` switch dispatch (Tier B)
- **`IsService` / `IsKeyedService`** — type-switch membership queries
- **`GetKeyedService` / `GetKeyedServices`** — nested type+key switch dispatch
- **`CreateScope() Scope`** — scope struct with scoped field re-instantiation
- **`Close() error`** — idempotent LIFO disposal via `runtime.Disposer`
- **`scope` struct** — holds scope-local fields, delegates singletons to `root`

## Disposable Services

Services that implement `io.Closer` are automatically tracked and disposed in LIFO order when the container/scope is closed.

Services that implement `godi.AsyncDisposable` (with `CloseAsync(ctx context.Context) error`) get async-aware disposal. If a service implements both, sync `Close` is preferred (the async path is available via `CloseAsync` on the root/scope).

## Common Patterns

### Constructor with dependencies

```go
type Service struct {
    logger ILogger
    cache  ICache
}

// Parameters are auto-resolved by the generator
func NewService(logger ILogger, cache ICache) *Service {
    return &Service{logger: logger, cache: cache}
}
```

### Constructor with error

```go
func NewService(dep IDep) (*Service, error) {
    if dep == nil {
        return nil, fmt.Errorf("dep required")
    }
    return &Service{dep: dep}, nil
}
```

The generated `NewRoot()` propagates the error.

### Slice dependencies (multi-impl injection)

When multiple implementations are registered for the same interface, a constructor can receive them as a slice:

```go
func NewAggregator(handlers []IHandler) *Aggregator {
    return &Aggregator{handlers: handlers}
}
```

### Provider injection

A factory can receive the `Provider` to do runtime resolution:

```go
godi.AddSingletonFactory[IService](&r, func(p godi.Provider) IService {
    dep := godi.GetRequired[IDep](p)
    return NewService(dep)
})
```

## Error Handling

- **`godi.ErrProviderClosed`** — panicked when resolving after `Close()` is called
- **`godi.AnyKeySingleResolveError(t)`** — panicked when `GetKeyedService` is called with `AnyKey`
- **`godi.AnyKeyRegistrationError(t)`** — panicked when `AnyKey` is used as a registration key
- Generator exits non-zero with zero bytes on any validation failure (NFR2)

## Gotchas

- Registration functions are **never executed** — they exist only for the generator's static analysis
- Form A requires a `New{Impl}` constructor in the **same package** as `Impl`
- Form B's factory expression is inlined **verbatim** as source text into the generated container
- Keyed service keys must be **compile-time constants**
- `godi.AnyKey` is for **querying** (GetKeyedServices) only — never for registration or single-value resolution
- The generated `container_gen.go` should be committed to version control — `go generate ./...` is a CI drift gate
- Always use `GOWORK=off` when building/testing godi itself (not required for consumer projects)

## Quick Reference

Read `references/api.md` for the complete list of all public types and functions with their signatures.
