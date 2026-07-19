# Keel — Makefile
#
# Thin wrapper over the shared SliceSoft base, which the ss-workstation installer
# provisions to ~/.config/slicesoft/base.mk. The base provides the standard
# commands (setup, test, lint, clean, check-secrets); this file adds the
# keel-specific Go targets.
#
# No base? Run the ss-workstation setup, or use the raw `go` commands documented
# in CONTRIBUTING.md (go build -o keel . / go test ./...).

BINARY     := keel
PKG        := github.com/slice-soft/keel/cmd
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w \
	-X $(PKG).version=$(VERSION) \
	-X $(PKG).commit=$(COMMIT) \
	-X $(PKG).buildDate=$(BUILD_DATE)

# Extra paths for `make clean` (consumed by base.mk)
CLEAN_EXTRA := $(BINARY) coverage.out

# ─── Shared base (setup / test / lint / clean / check-secrets / help) ───────────
SS_BASE ?= $(HOME)/.config/slicesoft/base.mk
ifeq ($(wildcard $(SS_BASE)),)
    $(error SliceSoft base.mk not found at $(SS_BASE) — run the ss-workstation setup [github.com/slice-soft/ss-workstation], or use the raw go commands in CONTRIBUTING.md)
endif
include $(SS_BASE)

# ─── keel-specific targets ──────────────────────────────────────────────────────
.PHONY: build
build: ## Build the keel binary (with version ldflags)
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .
	@echo "Built ./$(BINARY) ($(VERSION))"

.PHONY: run
run: ## Run the CLI locally (e.g. make run ARGS="new my-app")
	go run . $(ARGS)

.PHONY: coverage
coverage: ## Run tests with a coverage summary
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

.PHONY: install
install: ## Build and install keel into GOBIN
	go install -ldflags "$(LDFLAGS)" .

.PHONY: snapshot
snapshot: ## Build a local multi-platform snapshot (goreleaser)
	@command -v goreleaser &>/dev/null && goreleaser release --snapshot --clean \
		|| echo "goreleaser not installed: go install github.com/goreleaser/goreleaser/v2@latest"

.PHONY: fmt
fmt: ## Format the codebase
	gofmt -l -w .

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	go mod tidy
