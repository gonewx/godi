# godi build, test and release targets.
#
# The toolchain (Go, gh) and the environment (GOWORK=off) are pinned
# project-wide by mise (mise.toml). Run targets as `mise exec -- make <target>`,
# or as plain `make <target>` in a shell with mise activated. CI runs exactly
# these targets; there are no other scripts.

GO          ?= go
PKGS        ?= ./...
REMOTE      ?= origin
MAIN_BRANCH ?= main

# Release targets take VERSION=vX.Y.Z. It is exported and read by recipes as a
# shell variable, never expanded by make into the recipe text.
VERSION ?=
export VERSION

CHANGELOG    := CHANGELOG.md
CHANGELOG_ZH := CHANGELOG.zh-CN.md

# Flags for `git tag` in release-tag; set GIT_TAG_FLAGS=-s to sign the tag.
GIT_TAG_FLAGS ?= -a

# awk program printing the body of the changelog section whose heading starts
# with the value of h. It compares strings (index), so the dots in a version
# are not treated as regular-expression wildcards.
CHANGELOG_SECTION = index($$0, h) == 1 { p = 1; next } \
	p && (/^\#\# \[/ || /^\[[^]]*\]: / || /^---$$/) { exit } \
	p && !s && /^[[:space:]]*$$/ { next } \
	p { s = 1; print }

# Targets with prerequisites (ci, release-check, release-tag) rely on their
# order.
.NOTPARALLEL:

.DEFAULT_GOAL := help

.PHONY: help
help: ## List available targets.
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "} {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# --- Build and test -----------------------------------------------------------

.PHONY: build
build: ## Build all packages.
	$(GO) build $(PKGS)

.PHONY: test
test: ## Run all tests with the race detector and coverage.
	$(GO) test -race -covermode=atomic -coverprofile=coverage.out $(PKGS)

.PHONY: test-short
test-short: ## Run tests without the race detector, skipping slow cases.
	$(GO) test -short $(PKGS)

.PHONY: conformance
conformance: ## Run the dual-provider conformance suite (verbose).
	$(GO) test -race -v -run 'Spec|Conformance' ./conformance/...

.PHONY: golden
golden: ## Regenerate golden files (generator emission fixtures).
	$(GO) test $(PKGS) -run Golden -update-golden

.PHONY: cover
cover: test ## Produce an HTML coverage report (coverage.html).
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "coverage report written to coverage.html"

.PHONY: vet
vet: ## Run go vet.
	$(GO) vet $(PKGS)

.PHONY: fmt
fmt: ## Format all Go sources.
	$(GO) fmt $(PKGS)

.PHONY: fmt-check
fmt-check: ## Fail when any Go source in the module is not gofmt-clean.
	@unformatted="$$(gofmt -l $$($(GO) list -f '{{.Dir}}' $(PKGS)))" || exit 1; \
	if [ -n "$$unformatted" ]; then \
		echo "error: these files are not gofmt-clean; run 'make fmt':" >&2; \
		echo "$$unformatted" >&2; \
		exit 1; \
	fi; \
	echo "all Go sources are gofmt-clean"

.PHONY: tidy
tidy: ## Tidy the module graph.
	$(GO) mod tidy

.PHONY: generate
generate: ## Re-run the generator over the module (go generate).
	$(GO) generate $(PKGS)

.PHONY: generate-check
generate-check: ## Fail when committed generated code differs from what the generator emits.
	@snapshot() { { git status --porcelain; git diff; } | git hash-object --stdin; }; \
	before="$$(snapshot)" || exit 1; \
	$(GO) generate $(PKGS) || exit 1; \
	after="$$(snapshot)" || exit 1; \
	if [ "$$before" != "$$after" ]; then \
		echo "error: generated code is out of date; run 'make generate' and commit the result:" >&2; \
		git status --porcelain >&2; \
		git --no-pager diff --stat >&2; \
		exit 1; \
	fi; \
	echo "generated code is up to date"

.PHONY: changelog-check
changelog-check: ## Fail when the two changelog files do not list the same versions.
	@en="$$(grep '^## \[' $(CHANGELOG))"; zh="$$(grep '^## \[' $(CHANGELOG_ZH))"; \
	if [ -z "$$en" ] || [ "$$en" != "$$zh" ]; then \
		echo "error: $(CHANGELOG) and $(CHANGELOG_ZH) must list the same version headings" >&2; \
		exit 1; \
	fi; \
	echo "changelogs list the same versions"

.PHONY: ci
ci: fmt-check vet build generate-check changelog-check test ## Everything CI runs, in CI order.

.PHONY: check
check: ci ## Alias of ci.

.PHONY: version
version: ## Print the generator version of this checkout.
	@$(GO) run ./cmd/godi-gen -version

# --- Release ------------------------------------------------------------------
#
#   1. Rename "## [Unreleased]" to "## [X.Y.Z] - YYYY-MM-DD" in both changelog
#      files, add a fresh empty Unreleased section, commit on $(MAIN_BRANCH).
#   2. make release-tag VERSION=vX.Y.Z    verify everything, create the tag locally
#   3. make release-push VERSION=vX.Y.Z   push $(MAIN_BRANCH) and the tag together
#   4. The release workflow runs release-check and release-publish on the tag.

.PHONY: release-preflight
release-preflight: # Validate VERSION, both changelog sections and the working tree.
	@if ! printf '%s\n' "$$VERSION" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$$'; then \
		echo "error: VERSION must be a semver tag such as v0.2.0 (got '$$VERSION'), for example 'make release-check VERSION=v0.2.0'" >&2; \
		exit 1; \
	fi; \
	num="$${VERSION#v}"; \
	for f in $(CHANGELOG) $(CHANGELOG_ZH); do \
		if ! awk -v h="## [$$num] - " 'index($$0, h) == 1 && substr($$0, length(h) + 1) ~ /^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]$$/ { found = 1 } END { exit !found }' "$$f"; then \
			echo "error: $$f has no '## [$$num] - YYYY-MM-DD' section; move the Unreleased entries under that heading first" >&2; \
			exit 1; \
		fi; \
		if [ -z "$$(awk -v h="## [$$num]" '$(CHANGELOG_SECTION)' "$$f")" ]; then \
			echo "error: the [$$num] section of $$f is empty" >&2; \
			exit 1; \
		fi; \
	done; \
	if ! git diff --quiet HEAD --; then \
		echo "error: tracked files have uncommitted changes; commit them before releasing" >&2; \
		exit 1; \
	fi; \
	echo "release preflight passed for $$VERSION"

.PHONY: release-check
release-check: release-preflight ci ## Verify VERSION=vX.Y.Z is ready to release (changelog, clean tree, full CI gate).

.PHONY: release-notes
release-notes: ## Print the release notes for VERSION=vX.Y.Z (its section of CHANGELOG.md).
	@num="$${VERSION#v}"; \
	if [ -z "$$num" ]; then echo "error: VERSION is required, for example 'make release-notes VERSION=v0.2.0'" >&2; exit 1; fi; \
	notes="$$(awk -v h="## [$$num]" '$(CHANGELOG_SECTION)' $(CHANGELOG))"; \
	if [ -z "$$notes" ]; then \
		echo "error: $(CHANGELOG) has no non-empty section for [$$num]" >&2; \
		exit 1; \
	fi; \
	printf '%s\n' "$$notes"

.PHONY: release-tag
release-tag: release-check ## Create the annotated tag VERSION=vX.Y.Z locally, with the release notes as its message.
	@branch="$$(git rev-parse --abbrev-ref HEAD)"; \
	if [ "$$branch" != "$(MAIN_BRANCH)" ]; then \
		echo "error: releases are tagged on $(MAIN_BRANCH), not on $$branch" >&2; \
		exit 1; \
	fi; \
	if git rev-parse -q --verify "refs/tags/$$VERSION" >/dev/null; then \
		echo "error: tag $$VERSION already exists" >&2; \
		exit 1; \
	fi; \
	notes="$$($(MAKE) -s --no-print-directory release-notes)" || exit 1; \
	printf '%s\n\n%s\n' "$$VERSION" "$$notes" | git tag $(GIT_TAG_FLAGS) "$$VERSION" --cleanup=whitespace -F - || exit 1; \
	echo "tagged $$VERSION at $$(git rev-parse --short HEAD); publish it with 'make release-push VERSION=$$VERSION'"

.PHONY: release-push
release-push: ## Push the main branch and the tag VERSION=vX.Y.Z to the remote in one atomic push.
	@if ! git rev-parse -q --verify "refs/tags/$$VERSION" >/dev/null; then \
		echo "error: tag '$$VERSION' does not exist; run 'make release-tag VERSION=vX.Y.Z' first" >&2; \
		exit 1; \
	fi; \
	git push --atomic $(REMOTE) $(MAIN_BRANCH) "refs/tags/$$VERSION"

.PHONY: release-publish
release-publish: ## Create the GitHub Release for the pushed tag VERSION=vX.Y.Z (run by the release workflow).
	@if ! git merge-base --is-ancestor "refs/tags/$$VERSION^{commit}" "$(REMOTE)/$(MAIN_BRANCH)"; then \
		echo "error: tag '$$VERSION' is missing or is not on $(REMOTE)/$(MAIN_BRANCH)" >&2; \
		exit 1; \
	fi; \
	notes="$$($(MAKE) -s --no-print-directory release-notes)" || exit 1; \
	prerelease=""; \
	case "$$VERSION" in *-*) prerelease="--prerelease" ;; esac; \
	printf '%s\n' "$$notes" | gh release create "$$VERSION" --verify-tag --title "$$VERSION" $$prerelease --notes-file -
