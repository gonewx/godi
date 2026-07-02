# godi — Development Guide

## Prerequisites

The Go toolchain is pinned project-wide by [mise](https://mise.jdx.dev) via
`mise.toml` (`go = "1.27rc1"`). With mise activated in your shell, `go` resolves
to the pinned version automatically:

```bash
mise install          # install the pinned Go toolchain (go 1.27rc1)
go version            # → go1.27rc1
```

No third-party dependencies are required to build or test: the suite uses the
standard library only (no testify), keeping the library's dependency surface
minimal. The only external module dependency is `golang.org/x/tools`, used by the
generator for static analysis.

> Go 1.27 unlocks method-level type parameters (AD-5). Until GA, the pin stays at
> `1.27rc1`; bump `mise.toml` **and** the `go` directive in `go.mod` together when
> the GA patch ships.

## Common Commands

| Command | Purpose |
| --- | --- |
| `make test` | All tests with the race detector + coverage profile |
| `make test-short` | Fast pass, no race detector, slow cases skipped |
| `make conformance` | Dual-provider conformance suite (verbose) |
| `make golden` | Regenerate generator emission goldens (`-update-golden`) |
| `make cover` | HTML coverage report (`coverage.html`) |
| `make vet` | `go vet ./...` |
| `make fmt` | Format all Go sources |
| `make tidy` | Tidy the module graph |
| `make check` | CI parity: `vet` + full race/coverage test run |
| `make help` | List all targets |

Equivalent raw commands:

```bash
go test ./...                # plain run
go test -race ./...          # with the race detector
go test -cover ./...         # with a coverage summary
go test ./... -update-golden # rewrite golden fixtures (review the diff!)
```

## Testing Conventions

- **Standard-library assertions only.** Assertions in `internal/testutil` are thin
  wrappers over a `testing.TB` seam (`Equal`, `Nil`, `Panics`, ...) — no testify.
  Their failure behavior is itself unit-tested via a fake `TB`.
- **Red-phase build tags.** ATDD scaffolds carry `//go:build atdd_redphase` and are
  inert until the tag is removed to activate them (`*_redphase_test.go`).
- **Golden files.** Generator emission tests compare emitted bytes against
  `internal/genlib/testdata/*.golden`; `-update-golden` rewrites them — always
  review the diff before committing.
- **One registration source, two providers.** Never hand-fork the runtime and
  codegen registrations (AD-14). Both factories build from the same source; the
  codegen output is the generator's deterministic transform of it.
- **Resolve by `reflect.Type`.** Use `reflect.TypeFor[T]()` for type keys — it
  matches the provider's Tier B `reflect.Type` dispatch (AD-13).
- **Assert behavior, not implementation.** Spec cases touch only the `Provider`
  contract, so both providers are held to identical, observable behavior via
  `RunSuite` + `AssertEquivalent`.

## Continuous Integration

`.github/workflows/test.yml` runs on push to `main` and on every pull request:

1. `jdx/mise-action` provisions the toolchain from `mise.toml` (CI uses the same
   pinned Go as local).
2. **Format check** — `gofmt -l` over the main module's package dirs; unclean
   sources fail the build.
3. `go vet ./...`
4. `go build ./...`
5. **Generate gate** — once `cmd/godi-gen` is present, `go generate ./...` runs and
   the build fails on drift (committed generated code out of date) or on a
   bad-graph registration (which must produce zero code, AD-3).
6. **Test** — `go test -race -covermode=atomic -coverprofile=coverage.out ./...`,
   covering the dual-provider conformance suite and golden tests, with the race
   detector as the quality gate.
7. Coverage is uploaded as a build artifact.

godi is a library: there is no deploy step. Release = a semver tag; consumers run
`go generate` as a build step once `cmd/godi-gen` lands.

## Contribution Guidelines

- **Format before committing.** Run `make fmt` (or `gofmt -w`) — CI enforces a
  gofmt-clean tree.
- **Keep the dependency surface minimal.** Do not add third-party test
  dependencies; extend `internal/testutil` instead.
- **Respect the architecture invariants.** Emit code only through `genlib.Generate`
  (AD-2); keep reflective construction confined to the activator (AD-13); derive
  both providers from one registration source (AD-14).
- **Regenerate goldens deliberately.** When emission changes intentionally, run
  `make golden` and commit the reviewed diff; an unexplained golden mismatch is a
  real regression.
- **Clean up with `t.Cleanup`.** `RunSuite` closes each built provider via
  `t.Cleanup`; follow the same pattern for any resource a case opens.

## Troubleshooting

| Symptom | Cause / fix |
| --- | --- |
| `go: command not found` or wrong Go version | mise isn't active in this shell. Run `mise install`, then `eval "$(mise activate bash)"` (or use `mise exec -- go ...`). |
| `go.work lists go 1.23.1, but ... requires go >= 1.26` | The parent `go.work` predates godi's `go` directive. Raise the `go` line in the parent `go.work` (siblings stay valid). |
| `cannot find toolchain ... go1.27` / download fails | Go 1.27 GA may not be published for your platform yet. Keep the `1.27rc1` pin until GA ships. |
| `golden: read testdata/...: no such file` | Fixture missing — create it with `go test ./... -update-golden`, then review the diff before committing. |
| golden mismatch | The emitted bytes changed. If intended, run `make golden` and commit the new fixture; otherwise it's a real regression. |
