# AGENTS.md — godi

Agent instructions for `github.com/gonewx/godi` (compile-time DI library for Go).

## Documentation conventions (mandatory)

The docs are **bilingual with English as the primary language**. Any change to
user-facing documentation MUST keep both languages in sync within the same
commit.

### File layout

| Path | Role |
| --- | --- |
| `README.md` | English primary (rendered by GitHub by default) |
| `README.zh-CN.md` | Chinese mirror of `README.md`, section structure 1:1 |
| `CHANGELOG.md` | English primary changelog; release notes are cut from it |
| `CHANGELOG.zh-CN.md` | Chinese mirror of `CHANGELOG.md`, same versions and entries |
| `docs/index.md` | Language router linking to both indexes |
| `docs/en/*.md` | English docs (`index`, `project-overview`, `architecture`, `source-tree-analysis`, `development-guide`) |
| `docs/zh/*.md` | Chinese mirrors of `docs/en/*.md`, section structure 1:1 |

### Mirror rules

- Every zh doc pairs with exactly one en doc at the mirrored path; headings
  must correspond 1:1 in order and level.
- Each file carries BOTH:
  - a top-of-file language switcher line: `English | [中文](./README.zh-CN.md)`
    (or the reverse in the zh file),
  - a bottom cross-link after `---`: `[中文版 →](./README.zh-CN.md)` /
    `[English version →](./README.md)` (adjust relative path per location).
- New root-level docs get one row in both `docs/{en,zh}/index.md`
  "Existing Documentation" tables.
- All relative link targets must exist; verify before committing.
- Every user-visible change adds an entry under `## [Unreleased]` in BOTH
  changelog files in the same commit; mark breaking changes `**Breaking:**` /
  `**破坏性变更：**`. Section headings (`### Changed` …) stay in English in
  both files.
- Code comments inside examples follow each file's language; identifiers,
  lifetimes, and DSL function names stay verbatim (`AddScopedFactory`,
  `singleton / scoped / transient`, 形态 A / 形态 B ↔ form A / form B).

## Commands and tooling (mandatory)

- Every build, test, CI and release command is a `Makefile` target; the
  toolchain and environment come from `mise.toml`. Do NOT add standalone
  scripts (`scripts/*.sh`, ad-hoc Python, inline shell in workflows): add or
  extend a Makefile target and have the workflow call `make <target>`.
- Tools a target needs are pinned in `mise.toml` `[tools]`; environment a
  target needs goes in `mise.toml` `[env]` (`GOWORK=off` lives there).
- Releases are cut only through `make release-tag` / `make release-push`; see
  "Releasing" in `docs/{en,zh}/development-guide.md`.

## Verified facts for doc examples (do not regress)

Checked against source on 2026-08-22 — re-verify if the generator changes:

1. Generated containers always emit `func NewRoot() (*Root, error)` — every
   usage example must handle the error
   (see `internal/genlib/generate.go`).
2. `//go:generate` directives have NO space after `//`; a space silently
   disables them under `go generate`.
3. Canonical form-A registration shape (matches `skill/SKILL.md`,
   `cmd/godi-gen/testdata/**`): local `var r godi.Registry`, pass `&r`, use
   the bare impl type name — the generator resolves constructors via
   `Lookup("New" + implName)` (`cmd/godi-gen/main.go`). Example bodies must be
   self-consistent: never resolve a service that is not registered.
4. Consumers run the generator through the `tool` directive: install with
   `go get -tool github.com/gonewx/godi/cmd/godi-gen`, and write the directive
   as `//go:generate go tool godi-gen -pkg <name> -out container_gen.go .`.
   Never document `go install …@latest`: it lets the generator drift from the
   library version in `go.mod` (checked 2026-11-13). This repository's own
   `conformance/codegen` directive uses `go run` on the in-module package and
   is not the consumer form.

Canonical Quick Start example lives in `README.md`; `README.zh-CN.md` and
`docs/{en,zh}/project-overview.md` must stay consistent with it.
