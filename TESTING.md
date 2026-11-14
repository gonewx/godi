# Testing godi

How the godi test framework is structured and how to run it. Go keeps tests
beside the code they exercise (no dedicated `tests/` tree), so this document
lives at the module root rather than in a test directory.

## Setup

The Go toolchain is pinned project-wide by [mise](https://mise.jdx.dev) via
`mise.toml` (`go = "1.27.0"`). With mise activated in your shell, `go` resolves
to the pinned version automatically:

```bash
mise install          # install the pinned Go toolchain
go version            # → go1.27.0
```

No third-party dependencies are required: the suite uses the standard library
only (no testify), keeping the library's dependency surface minimal.

> The architecture targets **Go 1.27** (`ARCHITECTURE-SPINE` Stack table). Go 1.27
> GA shipped on 2026-08-19; both `mise.toml` **and** the `go` directive in
> `go.mod` are at 1.27, which unblocked the `provider.Get[T]()` generic-method
> API that landed in Story 6.6.

## Running tests

```bash
make test          # all tests, race detector + coverage profile
make test-short    # fast pass, no race detector
make conformance   # dual-provider conformance suite (verbose)
make golden        # regenerate testdata/*.golden fixtures
make cover         # HTML coverage report (coverage.html)
make ci            # everything CI runs, in CI order (make check is an alias)
make help          # list all targets
```

Equivalent raw commands:

```bash
go test ./...                # plain run
go test -race ./...          # with the race detector
go test -cover ./...         # with coverage summary
go test ./... -update-golden # rewrite golden fixtures (review the diff!)
```

## Architecture overview

```text
internal/testutil/      stdlib-only test helpers (importable across the module)
  assert.go             ergonomic assertions over a TB seam: Equal, Nil, Panics, …
  factory.go            functional-options data factory: Build / BuildFrom
  golden.go             golden-file comparator with the -update-golden flag
  testdata/             committed golden fixtures
conformance/            ported .NET DI specification suite (the test framework's core)
  harness.go            Provider contract, ProviderFactory seam, RunSuite, AssertEquivalent
  spec_test.go          the 116 ported specification cases
  fakes/                the 52 ported fakes
  codegen/              pre-generated container + its tests (Path γ sensitive scenarios)
```

- **Assertions** are tiny stdlib wrappers, not testify. They take a `TB`
  interface (the subset of `testing.TB` used), so failure behavior is itself unit
  tested via a fake `TB`.
- **Data factories** use the functional-options pattern (`Build(opts...)`) — the
  idiomatic Go alternative to faker-style fixture builders with overrides.
- **Golden files** back the generator's code-emission tests: `Golden(t, name,
  got)` compares emitted bytes against `testdata/<name>.golden`; `-update-golden`
  rewrites them.
- **Conformance harness** is the heart of the framework. A single suite, written
  once against the `Provider` contract, is parameterized over a `ProviderFactory`
  (the Go equivalent of dotnet's `abstract CreateServiceProvider`) and run against
  every provider. `AssertEquivalent` is the cross-provider differential-equivalence
  check: both providers, built from equivalent registrations, must yield identical
  results.

## Best practices

- **Table-driven subtests.** Express specification cases as `[]SpecCase` and let
  `RunSuite` drive them as nested `t.Run` subtests (provider/case).
- **Clean up with `t.Cleanup`.** `RunSuite` closes each built provider via
  `t.Cleanup`; follow the same pattern for any resource a case opens.
- **Equivalent registrations, two providers.** The runtime factory registers
  hand-written `runtime.Add*` closures; the codegen factory uses the container
  generated from the `godi.Add*` markers in `conformance/codegen`. Keep the two
  registration sets equivalent by hand (AD-14).
- **Resolve by `reflect.Type`.** Use `reflect.TypeFor[T]()` for type keys — it
  matches the Provider's Tier B `reflect.Type` dispatch (AD-13).
- **Mind the nil-interface footgun.** A nil interface value must not read as a
  present service (AD-5); `testutil.Nil` treats typed-nil as nil for exactly this.
- **Assert behavior, not implementation.** Spec cases touch only the `Provider`
  contract so both providers are held to identical, observable behavior.

## CI integration

Every command CI runs is a Makefile target, and `mise.toml` supplies the
toolchain and the environment (`GOWORK=off`), so `make ci` locally runs exactly
what `.github/workflows/test.yml` runs on push/PR:

1. `jdx/mise-action` provisions the toolchain and environment from `mise.toml`.
2. `make fmt-check`, `make vet`, `make build`
3. `make generate-check` — committed generated code must match what the
   generator emits.
4. `make changelog-check` — both changelog files list the same versions.
5. `make test` — `go test -race -covermode=atomic -coverprofile=coverage.out ./...`,
   covering the conformance dual-provider suite and golden tests, with the race
   detector as the quality gate.
6. Coverage is uploaded as a build artifact.

godi is a library: there is no deploy step. A release is a semver tag plus a
GitHub Release, cut with `make release VERSION=vX.Y.Z`; see
"Releasing" in the [development guide](./docs/en/development-guide.md#releasing).

## Troubleshooting

| Symptom | Cause / fix |
| --- | --- |
| `go: command not found` or wrong Go version | mise isn't active in this shell. Run `mise install` then `eval "$(mise activate bash)"` (or use `mise exec -- go ...`). |
| `go.work lists go 1.23.1, but ... requires go >= 1.27` | A parent `go.work` took over module resolution. `mise.toml` sets `GOWORK=off`, so this means mise is not active in this shell: run the target as `mise exec -- make <target>`. |
| `cannot find toolchain ... go1.27` / download fails | mise couldn't fetch the pinned toolchain for your platform. Confirm mise can pull that patch (`mise ls-remote go \| grep 1.27`), then re-run `mise install`; if a different patch was intended, adjust the pin in `mise.toml`. |
| `golden: read testdata/...: no such file` | Fixture missing — create it with `go test ./... -update-golden`, then review the diff before committing. |
| golden mismatch | The emitted bytes changed. If intended, run `make golden` and commit the new fixture; otherwise it's a real regression. |

## Architecture alignment

The harness encodes the spine's testing invariants — see
`_bmad-output/planning-artifacts/architecture/architecture-godi-2026-06-26/ARCHITECTURE-SPINE.md`:

| Invariant | Where it shows up |
| --- | --- |
| **AD-14** dual provider, equivalent registrations | `ProviderFactory` × 2, `RunSuite` |
| **AD-15** cross-provider differential equivalence | `AssertEquivalent` |
| **AD-16** pinned Provider method set + unmet-lookup semantics | `Provider` interface in `harness.go` |
| **AD-17** nil key routes to unkeyed; AnyKey semantics | `GetKeyedService` contract + sample case |

**Acceptance target** (`_bmad-output/specs/spec-godi/`): 116 specification cases +
52 fakes. `Provider` / `Scope` in `conformance/harness.go` are aliases of
`godi.Provider` / `godi.Scope`. `conformance/spec_test.go` defines the two real
factories — `rtFactory` (runtime oracle) and `cgFactory` (generated container) —
and runs the ported cases against both through `RunSuite` / `AssertEquivalent`;
further case groups live in the other `conformance/spec_*_test.go` files.
