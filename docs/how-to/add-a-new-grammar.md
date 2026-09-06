# How to Add a New Grammar Language

This guide explains how to add a new language grammar to the build pipeline and register its bindings.

---

## Overview of Steps

1. Add the grammar definition to [tree-sitter-config.yaml](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml).
2. (Optional) Add upstream Go bindings to [go.mod](file:///home/michael/projects/go/tree-sitter/go.mod).
3. Register the binding in [`internal/grammars/registry.go`](file:///home/michael/projects/go/tree-sitter/internal/grammars/registry.go).
4. Map the Go module path in [`cmd/sync-config/main.go`](file:///home/michael/projects/go/tree-sitter/cmd/sync-config/main.go).
5. Build, compile, and validate the new grammar.

---

## Step 1: Add the Grammar to `tree-sitter-config.yaml`

Open [tree-sitter-config.yaml](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml) and add a new entry under `languages:`:

```yaml
  - name: "ruby"
    grammar: "ruby"
    constructor: "tree_sitter_ruby"
    version: "v0.23.1"
    repository: "https://github.com/tree-sitter/tree-sitter-ruby.git"
    revision: "b213c490a6fa552d8e40f5ddc3cb7a8b417ff808"
    sample: "def hello; puts 'world'; end\n"
```

### Configuration Fields Checklist
- **`name`**: File path identifier (must match `^[A-Za-z0-9._-]+$`).
- **`grammar`**: Symbol suffix for the C function.
- **`constructor`**: Must strictly equal `tree_sitter_<grammar>`.
- **`version`**: Tag or Go module version.
- **`repository`**: Absolute HTTPS git repository URL.
- **`revision`**: Full 40-character hexadecimal commit SHA.
- **`source_subdir`**: Subdirectory if the repository contains multiple grammars (e.g. `typescript` / `tsx`).
- **`sample`**: Minimal valid code snippet used by `ts-build` to smoke-test parser output during validation.

---

## Step 2: Add Upstream Go Bindings (Optional)

If the grammar provides CGo bindings for direct Go import, add it to [go.mod](file:///home/michael/projects/go/tree-sitter/go.mod):

```bash
go get github.com/tree-sitter/tree-sitter-ruby@v0.23.1
```

---

## Step 3: Register in `internal/grammars/registry.go`

Open [`internal/grammars/registry.go`](file:///home/michael/projects/go/tree-sitter/internal/grammars/registry.go):

1. Import the Go binding package:
   ```go
   rubybinding "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
   ```
2. Add a case statement in [`GetLanguage`](file:///home/michael/projects/go/tree-sitter/internal/grammars/registry.go#L28):
   ```go
   case "ruby":
       ptr = unsafe.Pointer(rubybinding.Language())
   ```

---

## Step 4: Map in `cmd/sync-config/main.go`

To ensure automatic version synchronization works for the new grammar, add its Go module path to [`moduleToLanguage`](file:///home/michael/projects/go/tree-sitter/cmd/sync-config/main.go#L27):

```go
var moduleToLanguage = map[string][]string{
    // ... existing mappings ...
    "github.com/tree-sitter/tree-sitter-ruby": {"ruby"},
}
```

---

## Step 5: Verify the New Grammar

Execute the 3-phase pipeline for your new grammar:

```bash
# 1. Clone repository and generate source files
go run ./cmd/ts-build build --language ruby

# 2. Compile native shared library
go run ./cmd/ts-build compile --language ruby

# 3. Validate binary, test sample AST, and publish catalog
go run ./cmd/ts-build move --language ruby --force
```

Run test suite to verify tests pass:
```bash
make test
```
