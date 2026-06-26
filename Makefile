# godi test & quality targets.
#
# The Go toolchain is pinned project-wide by mise (mise.toml). With mise active
# in your shell (or via `mise exec -- make ...`), `go` resolves to the pinned
# version automatically.

GO   ?= go
PKGS ?= ./...

.DEFAULT_GOAL := help

.PHONY: help
help: ## List available targets.
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "} {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

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

.PHONY: tidy
tidy: ## Tidy the module graph.
	$(GO) mod tidy

.PHONY: check
check: vet test ## CI parity: vet + full race/coverage test run.
