TREE_SITTER         ?= tree-sitter
TREE_SITTER_VERSION ?= 0.26.8
NPM                 ?= npm
CONFIG              ?= tree-sitter-config.yaml
LANG                ?=
LANGUAGE            ?=
LANG_ARG            ?= $(if $(LANG),-l $(LANG),$(if $(LANGUAGE),-l $(LANGUAGE),))

.PHONY: help build sync-config sync-config-apply \
        grammar-build build-all \
        compile compile-wasm compile-linux compile-windows compile-mac compile-all \
        move \
        release release-linux release-windows release-mac release-all \
        test fmt vet lint tool-help \
        check-tree-sitter install-tree-sitter-cli \
        licenses clean

# ──────────────────────────────────────────────────────────────────────────────

help:
	@echo "go-tree-sitter - Tree-Sitter grammar builder"
	@echo ""
	@echo "Available commands:"
	@echo "  make build                  Build the Go packages and CLI"
	@echo "  make build-all              Full pipeline for host: build, compile, compact, move & clean build cache"
	@echo "                              (Supports LANG=<name>, e.g. make build-all LANG=python)"
	@echo "  make grammar-build          Clone/generate grammar sources (run once before compiling)"
	@echo ""
	@echo "  make compile                Compile for the default target (OS_TARGET in config)"
	@echo "  make compile-wasm           Compile WebAssembly parser (.wasm)"
	@echo "  make compile-linux          Compile for Linux amd64"
	@echo "  make compile-windows        Compile for Windows amd64"
	@echo "  make compile-mac            Compile for macOS amd64 + arm64"
	@echo "  make compile-all            Compile for all platforms"
	@echo ""
	@echo "  make move                   Publish artifacts to data/tree-sitter/grammar/, generate compact node types"
	@echo ""
	@echo "  make release                Full pipeline for the default target (grammar-build + compile + move)"
	@echo "  make release-linux          Full pipeline for Linux amd64"
	@echo "  make release-windows        Full pipeline for Windows amd64"
	@echo "  make release-mac            Full pipeline for macOS amd64 + arm64"
	@echo "  make release-all            Full pipeline for all platforms"
	@echo ""
	@echo "  make sync-config            Preview go.mod → config version sync (dry-run)"
	@echo "  make sync-config-apply      Apply go.mod → config version sync"
	@echo "  make check-tree-sitter      Check for the external tree-sitter CLI"
	@echo "  make install-tree-sitter-cli Install tree-sitter CLI via npm"
	@echo "  make test                   Run tests"
	@echo "  make fmt                    Format code"
	@echo "  make vet                    Analyze code for bugs"
	@echo "  make lint                   Run golangci-lint"
	@echo "  make tool-help              Show the tree-sitter CLI help"
	@echo "  make licenses               Regenerate NOTICE.md"
	@echo "  make clean                  Clean build artifacts"

# ── Version sync ──────────────────────────────────────────────────────────────

sync-config:
	go run ./cmd/sync-config --config $(CONFIG) --dry-run

sync-config-apply:
	go run ./cmd/sync-config --config $(CONFIG)

# ── Tree-sitter CLI ───────────────────────────────────────────────────────────

check-tree-sitter:
	@command -v $(TREE_SITTER) >/dev/null 2>&1 || { \
		echo "Missing external tree-sitter CLI: $(TREE_SITTER)"; \
		echo ""; \
		echo "Install it with:"; \
		echo "  make install-tree-sitter-cli"; \
		echo ""; \
		echo "Or install it manually with:"; \
		echo "  npm install -g tree-sitter-cli"; \
		echo ""; \
		echo "Then make sure '$(TREE_SITTER)' is on PATH."; \
		exit 1; \
	}

install-tree-sitter-cli:
	$(NPM) install -g tree-sitter-cli@$(TREE_SITTER_VERSION)

# ── Build Go packages ─────────────────────────────────────────────────────────

build:
	go build ./...

# ── Grammar source generation (platform-independent) ─────────────────────────

grammar-build: check-tree-sitter
	go run ./cmd/ts-build build --config $(CONFIG) $(LANG_ARG)

# ── Compile ───────────────────────────────────────────────────────────────────

compile: check-tree-sitter
	go run ./cmd/ts-build compile --config $(CONFIG) $(LANG_ARG)

compile-wasm: check-tree-sitter
	go run ./cmd/ts-build compile --config $(CONFIG) --wasm $(LANG_ARG)

compile-linux: check-tree-sitter
	go run ./cmd/ts-build compile --config $(CONFIG) --os linux --arch amd64 $(LANG_ARG)

compile-windows: check-tree-sitter
	go run ./cmd/ts-build compile --config $(CONFIG) --os windows --arch amd64 $(LANG_ARG)

compile-mac: check-tree-sitter
	go run ./cmd/ts-build compile --config $(CONFIG) --os macos --arch amd64 $(LANG_ARG)
	go run ./cmd/ts-build compile --config $(CONFIG) --os macos --arch arm64 $(LANG_ARG)

compile-all: compile-linux compile-windows compile-mac

# ── Move / publish ────────────────────────────────────────────────────────────

move:
	go run ./cmd/ts-build move --config $(CONFIG) --compact --force $(LANG_ARG)

# ── Release pipelines ─────────────────────────────────────────────────────────

build-all: check-tree-sitter grammar-build compile move

release: check-tree-sitter grammar-build compile move

release-linux: check-tree-sitter grammar-build compile-linux move

release-windows: check-tree-sitter grammar-build compile-windows move

release-mac: check-tree-sitter grammar-build compile-mac move

release-all: check-tree-sitter grammar-build compile-all move

# ── Testing and code quality ──────────────────────────────────────────────────

test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

tool-help:
	go run ./cmd/ts-build --help

# ── Licenses ──────────────────────────────────────────────────────────────────

licenses:
	@echo "Checking third-party Go dependency licenses..."
	@go-licenses check ./cmd/ts-build 2>/dev/null && echo "✓ All dependencies have approved open-source licenses"
	@echo ""
	@echo "Dependency license report:"
	@go-licenses report ./cmd/ts-build 2>/dev/null

# ── Clean ─────────────────────────────────────────────────────────────────────

clean:
	rm -f coverage.out coverage.html NOTICE.md.tmp
	go clean

.DEFAULT_GOAL := help
