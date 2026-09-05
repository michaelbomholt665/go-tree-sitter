TREE_SITTER ?= tree-sitter
TREE_SITTER_VERSION ?= 0.26.8
NPM ?= npm

.PHONY: help build grammar-build compile move release test clean licenses fmt lint vet tool-help check-tree-sitter install-tree-sitter-cli

help:
	@echo "go-tree-sitter - Tree-Sitter grammar builder"
	@echo ""
	@echo "Available commands:"
	@echo "  make build          - Build the Go packages and CLI"
	@echo "  make grammar-build  - Clone/generate configured grammar sources"
	@echo "  make compile        - Compile generated grammars for the configured target"
	@echo "  make move           - Move compiled grammar artifacts into the output directory, overwriting existing files"
	@echo "  make release        - Run the pinned generate, compile, validate, and atomic publish pipeline"
	@echo "  make check-tree-sitter - Check for the external tree-sitter CLI"
	@echo "  make install-tree-sitter-cli - Install the external tree-sitter CLI with npm"
	@echo "  make test           - Run tests"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Analyze code for bugs"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make tool-help      - Show the tree-sitter CLI help"
	@echo "  make licenses       - Generate NOTICE.md with license information"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make help           - Show this help message"

build:
	go build ./...

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

grammar-build: check-tree-sitter
	go run ./cmd/tree-sitter build

compile: check-tree-sitter
	go run ./cmd/tree-sitter compile

move:
	go run ./cmd/tree-sitter move --force

release: check-tree-sitter
	go run ./cmd/tree-sitter build --force
	go run ./cmd/tree-sitter compile
	go run ./cmd/tree-sitter move --both --force

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
	go run ./cmd/tree-sitter --help

licenses:
	@echo "Generating license notices..."
	go-licenses report github.com/michaelbomholt665/go-tree-sitter > NOTICE.md.tmp || true
	@if [ -f NOTICE.md.tmp ]; then \
		echo "# Third-Party Licenses\n" > NOTICE.md; \
		echo "This project includes code from the following projects:\n" >> NOTICE.md; \
		cat NOTICE.md.tmp >> NOTICE.md; \
		rm NOTICE.md.tmp; \
		echo "✓ NOTICE.md generated"; \
	else \
		echo "⚠ go-licenses failed; skipping NOTICE.md generation"; \
	fi

clean:
	rm -f coverage.out coverage.html NOTICE.md.tmp
	go clean

.DEFAULT_GOAL := help
