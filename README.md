# godi — Compile-time Dependency Injection

> Go dependency-injection library · `github.com/gonewx/godi`

English | [中文](./README.zh-CN.md)

---

Declare the dependency graph with static markers; a code generator emits plain Go wiring code — **zero reflection on the main path**.

## Features

| Capability | Description |
| --- | --- |
| **Compile-time assembly** | `godi-gen` statically reads DSL calls via `go/packages` + `go/types`, never executing them; the emitted container is ordinary Go code |
| **Illegal graphs never build** | for form A constructors, cycles, captive dependencies, and missing registrations fail at `go generate` — see [Known Limitations](#known-limitations) for form B |
| **Three lifetimes** | singleton / scoped / transient, scopes dispose LIFO |
| **Keyed services & open generics** | `AddKeyed*` family; open generics via explicit closed registrations |
| **One external dependency** | `golang.org/x/tools`, and only the generator uses it |

## Requirements

Go 1.27+ (relies on method-level type parameters on concrete types).

## Installation

```bash
go get github.com/gonewx/godi
```

## Quick Start

### 1. Declare the dependency graph

This is a static marker — it never runs. Form A discovers `New{Impl}`
constructors by strict naming convention; when no suitable constructor exists,
a form B factory is the escape hatch:

```go
package di

import "github.com/gonewx/godi"

//go:generate godi-gen -pkg di -out container_gen.go .

func register() {
    var r godi.Registry

    godi.AddSingleton[Config, AppConfig](&r)   // form A: finds NewAppConfig
    godi.AddScoped[Repo, PgRepo](&r)           // scoped lifetime
    godi.AddTransientFactory[Logger](&r, func(p godi.Provider) Logger {
        return NewLogger(godi.GetRequired[Config](p)) // form B: factory escape hatch
    })
}
```

### 2. Generate the container

```bash
go install github.com/gonewx/godi/cmd/godi-gen@latest
go generate ./...
```

### 3. Resolve services

```go
root, err := di.NewRoot()
if err != nil {
    return err
}
defer root.Close()

cfg := godi.GetRequired[Config](root)   // panics on miss
log, ok := godi.Get[Logger](root)       // returns false on miss
all := godi.GetServices[Repo](root)     // every implementation
```

### Scopes

```go
scope := root.CreateScope()
defer scope.Close()   // disposes Disposables in this scope, LIFO

repo := godi.GetRequired[Repo](scope)
```

## Generated Output

`container_gen.go` is a **build input** — commit it, so consumers can build
without installing the generator. After changing a DSL declaration, re-run
`go generate ./...` and make sure `git status --porcelain` stays empty
(guards against assembly drift).

## Known Limitations

- **Concurrency.** After `NewRoot()` returns, resolving singletons from a `Root`, resolving transients that do not depend — directly or indirectly — on a scoped service, and calling `CreateScope` are safe from multiple goroutines. Scoped instances are not: use each `Scope` from one goroutine at a time, and do not let a `Root` that is shared across goroutines resolve anything scoped — a scoped service itself, a transient that depends on one, or `GetServices` over a type that has scoped registrations. Call `Close` only after resolution has finished.
- **Form B factories are validated at run time, not at `go generate`.** Missing-registration, captive-dependency, and cycle checks are static only for form A (`New{Impl}`) constructors. A singleton form B factory that resolves an unregistered service or closes a cycle makes `NewRoot()` return an error; a scoped form B factory with the same problem panics when that service is first resolved within its scope. A singleton factory that resolves a scoped service through a named function or a keyed registration is not diagnosed.
- **Form B products are owned by the container.** Whatever a form B factory returns is tracked for disposal when it satisfies `io.Closer` or `godi.AsyncDisposable`. A factory that returns an instance the container already tracks — for example one it resolved from the provider to expose under a second interface — gets that instance closed more than once, so its `Close` must be idempotent.

## Documentation

Full documentation is bilingual (`docs/en` / `docs/zh`) — start at the index:

| Language | Index |
| --- | --- |
| English | [docs/en/index.md](./docs/en/index.md) |
| 中文 | [docs/zh/index.md](./docs/zh/index.md) |

| Topic | Content |
| --- | --- |
| [Testing Guide](./TESTING.md) | Test framework, commands, best practices |
| [Divergence notes](./conformance/) | Behavioral differences from .NET DI (Activator / Async / CallSite / Extensions / Open Generic) |

## License

MIT, see [LICENSE](./LICENSE).

---

[中文版 →](./README.zh-CN.md)
