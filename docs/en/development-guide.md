# godi — Development Guide

## Prerequisites

The Go toolchain is pinned project-wide by [mise](https://mise.jdx.dev) via
`mise.toml` (`go = "1.27.0"`). With mise activated in your shell, `go` resolves
to the pinned version automatically:

```bash
mise install          # install the pinned Go toolchain (go 1.27.0)
go version            # → go1.27.0
```

`mise.toml` also sets `GOWORK=off` (godi builds as a standalone module, so a
`go.work` in a parent directory must not take over) and pins `gh`, which only
`make release-publish` uses. In a shell without mise activated, run targets as
`mise exec -- make <target>`.

No third-party dependencies are required to build or test: the suite uses the
standard library only (no testify), keeping the library's dependency surface
minimal. The only external module dependency is `golang.org/x/tools`, used by the
generator for static analysis.

> Go 1.27 unlocks method-level type parameters (AD-5). Go 1.27 GA shipped on
> 2026-08-19; the pin is the stable `1.27.0` release. When moving to a newer
> **minor**, raise `mise.toml` **and** the `go` directive in `go.mod` together;
> 1.27.x patch bumps usually touch `mise.toml` alone.

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
| `make build` | `go build ./...` |
| `make fmt-check` | Fail when any Go source is not gofmt-clean |
| `make generate` | Re-run the generator (`go generate ./...`) |
| `make generate-check` | Fail when committed generated code is out of date |
| `make changelog-check` | Fail when the two changelog files list different versions |
| `make ci` | Everything CI runs, in CI order (`make check` is an alias) |
| `make version` | Print the generator version of this checkout |
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
- **Red-phase file names.** `*_redphase_test.go` files were written as ATDD
  red-phase scaffolds; their build tags have since been removed, so they now
  compile and run as part of the default test suite.
- **Golden files.** Generator emission tests compare emitted bytes against
  `internal/genlib/testdata/*.golden`; `-update-golden` rewrites them — always
  review the diff before committing.
- **Equivalent registrations, two providers.** The runtime factory registers
  hand-written `runtime.Add*` closures; the codegen factory uses the container
  generated from the `godi.Add*` markers in `conformance/codegen`. Keep the two
  registration sets equivalent by hand (AD-14).
- **Resolve by `reflect.Type`.** Use `reflect.TypeFor[T]()` for type keys — it
  matches the provider's Tier B `reflect.Type` dispatch (AD-13).
- **Assert behavior, not implementation.** Spec cases touch only the `Provider`
  contract, so both providers are held to identical, observable behavior via
  `RunSuite` + `AssertEquivalent`.

## Continuous Integration

Every command CI runs is a Makefile target; `mise.toml` supplies the toolchain
and the environment. There are no standalone scripts, so `make ci` locally runs
exactly what CI runs.

`.github/workflows/test.yml` runs on push to `main` and on every pull request:

1. `jdx/mise-action` provisions the toolchain and environment from `mise.toml`
   (Go, `gh`, `GOWORK=off`).
2. `make fmt-check` — `gofmt -l` over the main module's package dirs; unclean
   sources fail the build.
3. `make vet`
4. `make build`
5. `make generate-check` — runs `go generate ./...` and fails on drift
   (committed generated code out of date) or on a bad-graph registration (which
   must produce zero code, AD-3).
6. `make changelog-check` — `CHANGELOG.md` and `CHANGELOG.zh-CN.md` must list
   the same versions.
7. `make test` — `go test -race -covermode=atomic -coverprofile=coverage.out ./...`,
   covering the dual-provider conformance suite and golden tests, with the race
   detector as the quality gate.
8. Coverage is uploaded as a build artifact.

`.github/workflows/release.yml` runs when a `v*` tag is pushed; see
[Releasing](#releasing).

## Releasing

godi is a library: a release is a semver tag plus a GitHub Release. The
versioning policy is in the README; the steps below are all Makefile targets.

Day to day, changelog entries only ever go under `## [Unreleased]` in both
files. Nobody edits version headings by hand: a release is two commands.

1. `make release-tag VERSION=vX.Y.Z`, on `main`. Everything it does is local
   and nothing is pushed:
   - `release-prepare` moves the Unreleased entries of both changelog files
     under `## [X.Y.Z] - <today>`, leaves a fresh empty `## [Unreleased]`,
     rewrites the link references at the bottom, and commits the two files as
     `docs: release vX.Y.Z`. It refuses to run when Unreleased is empty or
     tracked files have uncommitted changes, and does nothing when the section
     already exists, so the command can be re-run after fixing a failure.
   - `release-check` verifies the version format, the dated non-empty section
     in both changelogs and a clean tree, then runs the whole `ci` target.
   - It then creates an annotated tag whose message is that changelog section.
     Set `GIT_TAG_FLAGS=-s` to sign the tag.
2. `make release-push VERSION=vX.Y.Z` — pushes `main` and the tag in one atomic
   push.

The release workflow then re-runs `make release-check` on the tagged commit and
runs `make release-publish`, which creates the GitHub Release from the same
changelog section. A version with a pre-release suffix (`v0.3.0-rc.1`) is
published as a pre-release.

`make release-notes VERSION=vX.Y.Z` prints the notes for any released version.

## Contribution Guidelines

- **Format before committing.** Run `make fmt` (or `gofmt -w`) — CI enforces a
  gofmt-clean tree.
- **Keep the dependency surface minimal.** Do not add third-party test
  dependencies; extend `internal/testutil` instead.
- **Respect the architecture invariants.** Emit code only through `genlib.Generate`
  (AD-2); keep reflective construction confined to the activator (AD-13); keep
  the two providers' conformance registrations equivalent (AD-14).
- **Regenerate goldens deliberately.** When emission changes intentionally, run
  `make golden` and commit the reviewed diff; an unexplained golden mismatch is a
  real regression.
- **Clean up with `t.Cleanup`.** `RunSuite` closes each built provider via
  `t.Cleanup`; follow the same pattern for any resource a case opens.

## Troubleshooting

| Symptom | Cause / fix |
| --- | --- |
| `go: command not found` or wrong Go version | mise isn't active in this shell. Run `mise install`, then `eval "$(mise activate bash)"` (or use `mise exec -- go ...`). |
| `go.work lists go 1.23.1, but ... requires go >= 1.27` | A parent `go.work` took over module resolution. `mise.toml` sets `GOWORK=off`, so this means mise is not active in this shell: run the target as `mise exec -- make <target>`. |
| `cannot find toolchain ... go1.27` / download fails | mise couldn't fetch the pinned toolchain for your platform. Confirm mise can pull that patch (`mise ls-remote go \| grep 1.27`), then re-run `mise install`; if a different patch was intended, adjust the pin in `mise.toml`. |
| `golden: read testdata/...: no such file` | Fixture missing — create it with `go test ./... -update-golden`, then review the diff before committing. |
| golden mismatch | The emitted bytes changed. If intended, run `make golden` and commit the new fixture; otherwise it's a real regression. |
