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
go get -tool github.com/gonewx/godi/cmd/godi-gen
```

The second command records the generator as a `tool` in your `go.mod`.
`go tool godi-gen` then always runs the generator at the version of
`github.com/gonewx/godi` your module requires, so the generator and the library
cannot drift apart.

## Quick Start

### 1. Declare the dependency graph

This is a static marker — it never runs. Form A discovers `New{Impl}`
constructors by strict naming convention; when no suitable constructor exists,
a form B factory is the escape hatch:

```go
package di

import "github.com/gonewx/godi"

//go:generate go tool godi-gen -pkg di -out container_gen.go .

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
(guards against assembly drift). The second line of the file records the
generator version (`godi-gen -version` prints it), so upgrading the library
without regenerating shows up as drift too.

## Known Limitations

- **Concurrency.** After `NewRoot()` returns, resolving singletons from a `Root`, resolving transients that do not depend — directly or indirectly — on a scoped service, and calling `CreateScope` are safe from multiple goroutines. Scoped instances are not: use each `Scope` from one goroutine at a time, and do not let a `Root` that is shared across goroutines resolve anything scoped — a scoped service itself, a transient that depends on one, or `GetServices` over a type that has scoped registrations. Call `Close` only after resolution has finished.
- **Form B factories are validated at run time, not at `go generate`.** Missing-registration, captive-dependency, and cycle checks are static only for form A (`New{Impl}`) constructors. A singleton form B factory that resolves an unregistered service or closes a cycle makes `NewRoot()` return an error; a scoped form B factory with the same problem panics when that service is first resolved within its scope. A singleton factory that resolves a scoped service through a named function or a keyed registration is not diagnosed.
- **Construction errors surface at different times by lifetime.** When a singleton's constructor returns an error, `NewRoot()` returns it wrapped — check it with `errors.Is` / `errors.As`, not `==`. When a scoped or transient constructor returns an error, neither `NewRoot()` nor `CreateScope()` fails; the error is raised as a panic when that service is first resolved (unless a singleton depends on it, in which case `NewRoot()` returns it as above).
- **Form B factories are inlined as source text.** The factory expression is copied verbatim into the generated container, so it cannot capture local variables or parameters of the registration function. Such a capture is usually rejected at generation time, but if the package happens to have a package-level identifier of the same name, the factory silently binds to that identifier instead. Values known only at run time (configuration, an existing connection) can currently reach a factory only through package-level variables.
- **Form B products are owned by the container.** Whatever a form B factory returns is tracked for disposal when its dynamic type satisfies `io.Closer` or `godi.AsyncDisposable`. A factory that returns an instance the container already tracks — for example one it resolved from the provider to expose under a second interface — gets that instance closed more than once, so its `Close` must be idempotent. A factory that returns an instance held outside the container, such as a database connection in a package-level variable or `os.Stdout`, hands it over to the container as well: if its dynamic type has `Close() error` or `CloseAsync(ctx) error`, closing the container closes it (`root.Close()`, or `root.CloseAsync(ctx)` for a `CloseAsync`-only instance), even when the registered service type has no `Close`. To keep such an instance open, return a wrapper type that has no `Close` or `CloseAsync` method and forwards the service's methods to the instance.
- **Registration order is source position, not call order.** The generator orders all marker calls in the package by file name, then by offset within the file, regardless of call relationships or control flow: markers inside helper functions and inside `if` branches all take effect, and their source position decides "last registration wins" and collection order. To make a registration take precedence, place it later in source — further down the same file, or in a file whose name sorts later.
- **The generator and the library versions must match.** A regenerated container calls helper symbols in the library, so a generator newer than the `github.com/gonewx/godi` version in `go.mod` emits a container that fails the type-check, and generation is rejected. Running the generator as `go tool godi-gen` (see [Installation](#installation)) keeps the two at the same version; a generator installed separately with `go install` is not tied to `go.mod` and can drift. After upgrading the library, regenerate every container.

## Versioning

Releases are semver tags (`vMAJOR.MINOR.PATCH`). The library and the generator
are one module and share one version; every release is described in the
[Changelog](./CHANGELOG.md).

- **The API is not frozen before v1.0.0.** A minor release (`v0.2.0` → `v0.3.0`)
  may change the public API, the shape of generated containers, or run-time
  behavior. Each such change is marked **Breaking** in the changelog and comes
  with upgrade steps.
- **A patch release (`v0.2.0` → `v0.2.1`) contains fixes only.** It removes no
  exported API.
- **Regenerate after every upgrade.** Run `go generate ./...` whenever the
  `github.com/gonewx/godi` version in `go.mod` changes. A container is supported
  only with the library version that generated it.
- **The minimum Go version is the `go` directive in `go.mod`.** Raising it is a
  minor release.

## Documentation

Full documentation is bilingual (`docs/en` / `docs/zh`) — start at the index:

| Language | Index |
| --- | --- |
| English | [docs/en/index.md](./docs/en/index.md) |
| 中文 | [docs/zh/index.md](./docs/zh/index.md) |

| Topic | Content |
| --- | --- |
| [Testing Guide](./TESTING.md) | Test framework, commands, best practices |
| [Changelog](./CHANGELOG.md) | Changes per release, upgrade steps |
| [Divergence notes](./conformance/) | Behavioral differences from .NET DI (Activator / Async / CallSite / Extensions / Open Generic) |

## License

MIT, see [LICENSE](./LICENSE).

---

[中文版 →](./README.zh-CN.md)
