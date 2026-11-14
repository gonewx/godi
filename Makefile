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

# Flags for `git tag` in release; set GIT_TAG_FLAGS=-s to sign the tag.
GIT_TAG_FLAGS ?= -a

# awk program printing the body of the changelog section whose heading starts
# with the value of h. It compares strings (index), so the dots in a version
# are not treated as regular-expression wildcards.
CHANGELOG_SECTION = index($$0, h) == 1 { p = 1; next } \
	p && (/^\#\# \[/ || /^\[[^]]*\]: / || /^---$$/) { exit } \
	p && !s && /^[[:space:]]*$$/ { next } \
	p { s = 1; print }

# awk program cutting a release in a changelog file: it turns the Unreleased
# section into "## [num] - date" under a fresh, empty Unreleased heading, and
# rewrites the "[Unreleased]: .../compare/<previous>...HEAD" link reference
# into one for Unreleased and one for the new version. It takes num, tag and
# date, and exits non-zero when the heading or the link reference is missing.
CHANGELOG_CUT = $$0 == "\#\# [Unreleased]" && !cut { cut = 1; print; print ""; print "\#\# [" num "] - " date; next } \
	index($$0, "[Unreleased]: ") == 1 && !linked { \
		i = index($$0, "/compare/"); j = index($$0, "...HEAD"); \
		if (i && j > i) { \
			base = substr($$0, 15, i - 15); prev = substr($$0, i + 9, j - i - 9); \
			print "[Unreleased]: " base "/compare/" tag "...HEAD"; \
			print "[" num "]: " base "/compare/" prev "..." tag; \
			linked = 1; next \
		} \
	} \
	{ print } \
	END { exit !(cut && linked) }

# Targets with prerequisites (ci, release-check, release) rely on their order.
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
#   make release VERSION=vX.Y.Z
#
# is the whole procedure: cut the changelogs and commit, run the full gate
# locally, push $(MAIN_BRANCH), wait for the test workflow to pass on that
# commit, and only then create and push the tag. The release workflow runs
# release-check and release-publish on the tag; the command waits for it too.
#
# Day to day, entries only ever go under "## [Unreleased]"; nobody edits
# version headings or pushes tags by hand.

.PHONY: release-version
release-version: # Validate the format of VERSION.
	@if ! printf '%s\n' "$$VERSION" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$$'; then \
		echo "error: VERSION must be a semver tag such as v0.2.0 (got '$$VERSION'), for example 'make release VERSION=v0.2.0'" >&2; \
		exit 1; \
	fi

.PHONY: release-prepare
release-prepare: release-version ## Move the Unreleased entries of both changelogs under VERSION=vX.Y.Z with today's date, and commit.
	@num="$${VERSION#v}"; \
	dated() { awk -v h="## [$$num] - " 'index($$0, h) == 1 { found = 1 } END { exit !found }' "$$1"; }; \
	if dated $(CHANGELOG) && dated $(CHANGELOG_ZH); then \
		echo "changelogs already have a [$$num] section; nothing to cut"; \
		exit 0; \
	fi; \
	if dated $(CHANGELOG) || dated $(CHANGELOG_ZH); then \
		echo "error: only one of $(CHANGELOG) and $(CHANGELOG_ZH) has a [$$num] section; make them match by hand" >&2; \
		exit 1; \
	fi; \
	if ! git diff --quiet HEAD --; then \
		echo "error: tracked files have uncommitted changes; commit them before releasing" >&2; \
		exit 1; \
	fi; \
	for f in $(CHANGELOG) $(CHANGELOG_ZH); do \
		if [ -z "$$(awk -v h="## [Unreleased]" '$(CHANGELOG_SECTION)' "$$f")" ]; then \
			echo "error: the Unreleased section of $$f is empty; there is nothing to release" >&2; \
			exit 1; \
		fi; \
	done; \
	date="$$(date +%Y-%m-%d)"; \
	for f in $(CHANGELOG) $(CHANGELOG_ZH); do \
		if ! awk -v num="$$num" -v tag="$$VERSION" -v date="$$date" '$(CHANGELOG_CUT)' "$$f" > "$$f.tmp"; then \
			rm -f "$$f.tmp"; \
			git checkout -q -- $(CHANGELOG) $(CHANGELOG_ZH); \
			echo "error: $$f needs a '## [Unreleased]' heading and an '[Unreleased]: .../compare/<previous tag>...HEAD' link reference" >&2; \
			exit 1; \
		fi; \
		mv "$$f.tmp" "$$f"; \
	done; \
	git add $(CHANGELOG) $(CHANGELOG_ZH) && git commit -q -m "docs: release $$VERSION" || exit 1; \
	echo "cut $$VERSION ($$date) in $(CHANGELOG) and $(CHANGELOG_ZH); committed as $$(git rev-parse --short HEAD)"

.PHONY: release-preflight
release-preflight: release-version # Validate both changelog sections and the working tree.
	@num="$${VERSION#v}"; \
	for f in $(CHANGELOG) $(CHANGELOG_ZH); do \
		if ! awk -v h="## [$$num] - " 'index($$0, h) == 1 && substr($$0, length(h) + 1) ~ /^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]$$/ { found = 1 } END { exit !found }' "$$f"; then \
			echo "error: $$f has no '## [$$num] - YYYY-MM-DD' section; 'make release VERSION=$$VERSION' cuts it from Unreleased" >&2; \
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

.PHONY: release-start
release-start: release-version # Cheap checks that run before anything changes: right branch, version not taken.
	@branch="$$(git rev-parse --abbrev-ref HEAD)"; \
	if [ "$$branch" != "$(MAIN_BRANCH)" ]; then \
		echo "error: releases are cut on $(MAIN_BRANCH), not on $$branch" >&2; \
		exit 1; \
	fi; \
	if [ -n "$$(git ls-remote --tags $(REMOTE) "refs/tags/$$VERSION")" ]; then \
		echo "error: $$VERSION is already released (the tag exists on $(REMOTE))" >&2; \
		exit 1; \
	fi; \
	if git rev-parse -q --verify "refs/tags/$$VERSION" >/dev/null && \
		[ "$$(git rev-parse "refs/tags/$$VERSION^{commit}")" != "$$(git rev-parse HEAD)" ]; then \
		echo "error: a local tag $$VERSION points at another commit; remove it with 'git tag -d $$VERSION'" >&2; \
		exit 1; \
	fi

.PHONY: release-push-branch
release-push-branch: # Push the release commit on the main branch, without any tag.
	git push $(REMOTE) $(MAIN_BRANCH)

# Wait for a workflow run on the HEAD commit and fail unless it succeeds.
release-wait-ci: WORKFLOW := test.yml
release-wait-publish: WORKFLOW := release.yml
.PHONY: release-wait-ci release-wait-publish
release-wait-ci release-wait-publish: # Wait for $(WORKFLOW) on the pushed HEAD commit; fail unless it passes.
	@sha="$$(git rev-parse HEAD)"; \
	echo "waiting for $(WORKFLOW) on $$(git rev-parse --short HEAD) ..."; \
	id=""; tries=0; \
	while [ -z "$$id" ] && [ "$$tries" -lt 36 ]; do \
		id="$$(gh run list --workflow $(WORKFLOW) --commit "$$sha" --limit 1 --json databaseId --jq '.[0].databaseId // empty')" || exit 1; \
		if [ -z "$$id" ]; then tries=$$((tries + 1)); sleep 5; fi; \
	done; \
	if [ -z "$$id" ]; then \
		echo "error: no $(WORKFLOW) run appeared for $$sha within 3 minutes" >&2; \
		exit 1; \
	fi; \
	url="$$(gh run view "$$id" --json url --jq .url)"; \
	if ! gh run watch "$$id" --exit-status --interval 15 >/dev/null 2>&1; then \
		echo "error: $(WORKFLOW) did not pass on $$sha: $$url" >&2; \
		exit 1; \
	fi; \
	echo "$(WORKFLOW) passed: $$url"

.PHONY: release-create-tag
release-create-tag: # Create the annotated tag at HEAD with the release notes as its message.
	@if git rev-parse -q --verify "refs/tags/$$VERSION" >/dev/null; then \
		echo "tag $$VERSION already exists at HEAD"; \
		exit 0; \
	fi; \
	notes="$$($(MAKE) -s --no-print-directory release-notes)" || exit 1; \
	printf '%s\n\n%s\n' "$$VERSION" "$$notes" | git tag $(GIT_TAG_FLAGS) "$$VERSION" --cleanup=whitespace -F - || exit 1; \
	echo "tagged $$VERSION at $$(git rev-parse --short HEAD)"

.PHONY: release-push-tag
release-push-tag: # Push the tag; this is the step that makes the release public.
	git push $(REMOTE) "refs/tags/$$VERSION"

.PHONY: release-report
release-report: # Print where the published release lives.
	@echo "released $$VERSION: $$(gh release view "$$VERSION" --json url --jq .url)"

# The order is the release procedure. Nothing is tagged until the test
# workflow has passed on the pushed release commit, and the command can be
# re-run after any failure: steps that are already done are skipped.
.PHONY: release
release: release-start release-prepare release-check release-push-branch release-wait-ci release-create-tag release-push-tag release-wait-publish release-report ## Release VERSION=vX.Y.Z end to end: cut the changelogs, verify, push, wait for CI, tag, publish.

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
