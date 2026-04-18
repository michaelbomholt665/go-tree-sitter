# Tree-Sitter Grammar Builder - Design Document

**Version:** 1.0  
**Last Updated:** 2026-04-18  
**Status:** Design

---

## 1. Overview

The Tree-Sitter Grammar Builder is a Go-based CLI tool that automates the building, compilation, and deployment of tree-sitter language grammars. It acts as a macro wrapper around `tree-sitter-cli` to streamline the process of managing grammar artifacts across multiple languages, versions, and target platforms.

### Goals
- Simplify grammar building and compilation workflows
- Automate cross-platform binary compilation (Windows, macOS, Linux)
- Provide flexible artifact management with JSON and SCM modes
- Enable clean, repeatable builds with automatic cleanup
- Maintain organized grammar storage in `data/tree-sitter/grammar/`
- Generate manifest metadata for runtime ABI verification and tooling integration
- Support rich graph analysis across code, configuration, and data formats

---

## 2. Requirements

### Technology Stack
- **Go:** 1.26 or later
- **tree-sitter-cli:** Latest version (required dependency)
- **Configuration Format:** YAML (root-level config file)
- **Build Output:** Cross-platform binaries + language-specific artifacts

### Target Platforms
- Windows (x86_64)
- macOS (x86_64, arm64)
- Linux (x86_64)

---

## 3. Architecture

### High-Level Components

```
cmd/tree-sitter/
├── main.go                          # Entry point, CLI dispatcher

internal/tree-sitter/
├── cli.go                           # tree-sitter-cli wrapper & command execution
├── config.go                        # YAML configuration parsing
├── build/
│   ├── builder.go                   # Grammar build orchestration
│   ├── compiler.go                  # Binary compilation logic
│   ├── mover.go                     # Artifact movement & organization
│   └── cleaner.go                   # Build directory cleanup
```

### Data Flow

```
[YAML Config] 
      ↓
[CLI Command] → [CLI Parser] → [Config Loader] → [Executor]
                                                      ↓
                                    ┌─────────────────┼─────────────────┐
                                    ↓                 ↓                 ↓
                              [Builder]          [Compiler]        [Mover]
                                ↓                   ↓                 ↓
                          [tree-sitter       [Cross-platform    [Organize
                           clone/build]       compilation]       Artifacts]
                                                                     ↓
                                                              [Cleaner]
                                                           [Reset build/]
```

---

## 4. Configuration

### YAML Configuration File (`tree-sitter-config.yaml`)

Located in the repository root, this file defines build targets and output directories.

```yaml
version: "1.0"

build_dir: "build"  # Directory for intermediate build artifacts

# ABI Version Mapping (maintained manually or auto-detected)
abi_versions:
  "0.19.0": { min: 13, max: 14 }
  "0.23.2": { min: 13, max: 14 }
  "0.24.0": { min: 13, max: 14 }
  "0.24.8": { min: 13, max: 14 }
  "0.25.0": { min: 13, max: 14 }

languages:
  # Core Languages (code)
  - name: "python"
    version: "v0.25.0"
    tree_sitter_version: "0.25.0"
    repository: "https://github.com/tree-sitter/tree-sitter-python.git"
    build_mode: "both"
    
  - name: "go"
    version: "v0.25.0"
    tree_sitter_version: "0.25.0"
    repository: "https://github.com/tree-sitter/tree-sitter-go.git"
    build_mode: "both"
    
  - name: "typescript"
    version: "v0.23.2"
    tree_sitter_version: "0.23.2"
    repository: "https://github.com/tree-sitter/tree-sitter-typescript.git"
    build_mode: "both"

  # Web Stack
  - name: "javascript"
    version: "v0.25.0"
    tree_sitter_version: "0.25.0"
    repository: "https://github.com/tree-sitter/tree-sitter-javascript.git"
    build_mode: "both"
    
  - name: "html"
    version: "v0.24.0"
    tree_sitter_version: "0.24.0"
    repository: "https://github.com/tree-sitter/tree-sitter-html.git"
    build_mode: "both"
    
  - name: "css"
    version: "v0.25.0"
    tree_sitter_version: "0.25.0"
    repository: "https://github.com/tree-sitter/tree-sitter-css.git"
    build_mode: "both"

  # React/Vue Components (from TypeScript repo)
  - name: "tsx"
    version: "v0.23.2"
    tree_sitter_version: "0.23.2"
    repository: "https://github.com/tree-sitter/tree-sitter-typescript.git"
    build_mode: "both"
    note: "Built from tree-sitter-typescript, TSX variant"

  # Configuration & Data Formats
  - name: "json"
    version: "v0.24.8"
    tree_sitter_version: "0.24.8"
    repository: "https://github.com/tree-sitter/tree-sitter-json.git"
    build_mode: "json"
    
  - name: "yaml"
    version: "v0.5.0"
    tree_sitter_version: "0.20.8"
    repository: "https://github.com/ikatyang/tree-sitter-yaml.git"
    build_mode: "json"
    note: "Community maintained, different versioning scheme"
    
  - name: "toml"
    version: "v0.5.1"
    tree_sitter_version: "0.20.8"
    repository: "https://github.com/tree-sitter/tree-sitter-toml.git"
    build_mode: "json"

  # Dependency Management
  - name: "go.mod"
    version: "v1.1.0"
    tree_sitter_version: "0.20.8"
    repository: "https://github.com/camdencheek/tree-sitter-go-mod.git"
    build_mode: "json"

  - name: "go.sum"
    version: "latest"
    tree_sitter_version: "0.20.8"
    repository: "https://github.com/tree-sitter-grammars/tree-sitter-go-sum.git"
    build_mode: "json"
    note: "Lock file for resolved dependencies"

  # gRPC & Distributed Systems
  - name: "proto"
    version: "latest"
    tree_sitter_version: "0.20.8"
    repository: "https://github.com/coder3101/tree-sitter-proto.git"
    build_mode: "both"

  # Database Languages
  - name: "sql"
    version: "latest"
    tree_sitter_version: "0.20.8"
    repository: "https://github.com/DerekStride/tree-sitter-sql.git"
    build_mode: "both"
    note: "Generic SQL; covers DuckDB and PostgreSQL dialects"
    
  - name: "cypher"
    version: "latest"
    tree_sitter_version: "0.20.8"
    repository: "https://github.com/taekwombo/tree-sitter-cypher.git"
    build_mode: "both"
    note: "OpenCypher for graph database queries (Kuzu/Ladybug)"

output:
  grammar_base: "data/tree-sitter/grammar"  # Base directory for organized grammars
  generate_manifest: true                    # Auto-generate manifest.json per grammar
```

### Configuration Structure
- **version:** Config format version for future compatibility
- **build_dir:** Intermediate directory for cloned and built grammars
- **abi_versions:** Mapping of tree-sitter versions to min/max ABI versions
- **languages:** Array of grammar definitions
  - **name:** Language identifier (used in output paths)
  - **version:** Git tag/branch to check out
  - **tree_sitter_version:** tree-sitter version for ABI mapping (optional, auto-detected if omitted)
  - **repository:** GitHub repository URL (supports community-maintained repos)
  - **build_mode:** Artifact mode ("json", "scm", or "both")
  - **note:** Optional descriptive note for special cases (e.g., TSX variant)
- **output.grammar_base:** Root directory for final artifact organization
- **output.generate_manifest:** Whether to create manifest.json files (default: true)

---

## 5. Command Specification

### Command 1: `build`

**Purpose:** Clone and build grammar source code from configured repositories.

**Usage:**
```bash
tree-sitter build [flags]
```

**Flags:**
- `--config` (optional): Path to config file (default: `tree-sitter-config.yaml`)
- `--language` (optional): Build only a specific language (default: all in config)

**Behavior:**
1. Load configuration from YAML file
2. For each language (or specified language):
   - Clone repository to `{build_dir}/{language}/` (if not already present)
   - Check out specified version/tag
   - Run `tree-sitter build-wasm` or equivalent to prepare source

**Error Handling:**
- Fail if config file not found
- Skip if source already present (allow `--force` flag to re-clone)
- Report errors for invalid repository URLs

**Output:**
- Generated grammar artifacts in `build/{language}/`
- Compiled binary and source metadata ready for next step

---

### Command 2: `compile`

**Purpose:** Compile built grammars into platform-specific binaries.

**Usage:**
```bash
tree-sitter compile [flags]
```

**Flags:**
- `--config` (optional): Path to config file (default: `tree-sitter-config.yaml`)
- `--language` (optional): Compile only a specific language
- `--os` (optional): Target OS (windows, macos, linux) - default: current OS
- `--arch` (optional): Target architecture (amd64, arm64) - default: current arch

**Behavior:**
1. Load configuration
2. For each language in `build/` directory:
   - For each target OS/arch:
     - Set Go environment variables: `GOOS`, `GOARCH`, `CGO_ENABLED`
     - Run compilation commands via tree-sitter-cli
     - Generate binaries with platform-specific naming
3. Store compiled binaries in structured locations for next step

**Binary Naming Convention:**
```
{language}-{version}-{OS}-{arch}(.exe on Windows)
```

**Output:**
- Cross-platform binaries in `build/{language}/bin/`
- Metadata files (node-types.json, queries/) alongside binaries

**Error Handling:**
- Fail if source not found in build directory (suggest running `build` first)
- Report compilation errors with clear messaging
- Skip unavailable platforms gracefully

---

### Command 3: `move`

**Purpose:** Organize compiled artifacts into final locations, generate manifest metadata, and clean build directory.

**Usage:**
```bash
tree-sitter move [flags]
```

**Flags:**
- `--config` (optional): Path to config file (default: `tree-sitter-config.yaml`)
- `--json`: Move binary + node-types.json only
- `--scm`: Move binary + queries/ directory only
- `--both`: Move binary + node-types.json AND queries/ directory (default)
- `--language` (optional): Move only a specific language
- `--clean` (optional): Clean build directory after move (default: true)
- `--no-manifest` (optional): Skip manifest.json generation (default: false)

**Behavior:**
1. Load configuration
2. Validate move mode (--json, --scm, or --both)
3. For each language:
   - Create output directory: `data/tree-sitter/grammar/{language}/`
   - **JSON mode:**
     - Copy binary file to output dir
     - Copy `node-types.json` to output dir
   - **SCM mode:**
     - Copy binary file to output dir
     - Copy `queries/` folder recursively to output dir
   - **Both mode:**
     - Copy binary file to output dir
     - Copy `node-types.json` to output dir
     - Copy `queries/` folder recursively to output dir
   - **Generate manifest.json** (unless --no-manifest):
     - Record grammar name and version
     - List all binaries with platform/arch and checksums
     - Include ABI version info (min/max from config)
     - Record compile timestamp (ISO 8601)
     - List available artifacts (node-types, queries, etc.)
4. Clean build directory:
   - Remove `build/{language}/` completely
   - Ensure `build/` remains empty for next cycle

**Output Structure:**
```
data/tree-sitter/grammar/
├── python/
│   ├── manifest.json                         # Metadata file
│   ├── python-v0.20.8-linux-amd64
│   ├── python-v0.20.8-windows-amd64.exe
│   ├── python-v0.20.8-macos-arm64
│   ├── node-types.json                       [JSON mode]
│   └── queries/                              [SCM mode]
│       ├── highlights.scm
│       ├── indents.scm
│       └── ...
├── typescript/
│   ├── manifest.json
│   ├── typescript-v0.20.5-{os}-{arch}(.exe)
│   ├── node-types.json
│   └── queries/
├── go/
│   ├── manifest.json
│   ├── go-v0.20.0-{os}-{arch}(.exe)
│   ├── node-types.json
│   └── queries/
├── json/
│   ├── manifest.json
│   ├── json-v0.19.0-{os}-{arch}(.exe)
│   └── node-types.json
├── toml/
│   ├── manifest.json
│   ├── toml-v0.5.2-{os}-{arch}(.exe)
│   └── node-types.json
└── yaml/
    ├── manifest.json
    ├── yaml-v0.5.0-{os}-{arch}(.exe)
    └── node-types.json
```

**Manifest File Format** (`manifest.json`):
```json
{
  "grammar": "python",
  "version": "v0.20.8",
  "compiled_at": "2026-04-18T17:28:51Z",
  "tree_sitter_version": "0.20.8",
  "binaries": [
    {
      "platform": "linux",
      "arch": "amd64",
      "filename": "python-v0.20.8-linux-amd64",
      "checksum_sha256": "abc123def456..."
    },
    {
      "platform": "windows",
      "arch": "amd64",
      "filename": "python-v0.20.8-windows-amd64.exe",
      "checksum_sha256": "xyz789..."
    },
    {
      "platform": "macos",
      "arch": "arm64",
      "filename": "python-v0.20.8-macos-arm64",
      "checksum_sha256": "lmn456..."
    }
  ],
  "abi": {
    "min_version": 13,
    "max_version": 14,
    "parser_version": "0.20.8"
  },
  "artifacts": {
    "has_node_types": true,
    "has_queries": true,
    "has_wasm": false
  }
}
```

**Error Handling:**
- Fail if compiled binaries not found
- Warn if expected artifact files (node-types.json, queries/) are missing but continue
- Create output directories if they don't exist
- Preserve existing files; optionally allow `--force` to overwrite
- Report checksum calculation failures

**Cleanup:**
- After successful move, remove the entire `build/{language}/` directory
- Leave `build/` directory empty but present for next build cycle
- Skip cleanup if move had errors (requires `--clean` or manual intervention)

**Usage Examples:**

```bash
# Move all artifacts with manifest generation
tree-sitter move --both

# Move only JSON mode without manifest
tree-sitter move --json --no-manifest

# Move specific language with SCM mode
tree-sitter move --scm --language python

# Force overwrite existing artifacts
tree-sitter move --both --force
```

---

## 6. Project Structure

```
tree-sitter/
├── cmd/
│   └── tree-sitter/
│       └── main.go                  # CLI entry point, command dispatch
│
├── internal/
│   ├── tree-sitter/
│   │   ├── cli.go                   # tree-sitter-cli wrapper
│   │   ├── config.go                # YAML config parsing & validation
│   │   └── build/
│   │       ├── builder.go           # Build orchestration
│   │       ├── compiler.go          # Compilation logic
│   │       ├── mover.go             # Artifact organization
│   │       ├── cleaner.go           # Build directory cleanup
│   │       └── manifest.go          # Manifest generation
│   │
│   └── tests/
│       ├── tree-sitter/
│       │   ├── cli_test.go
│       │   └── config_test.go
│       └── build/
│           ├── builder_test.go
│           ├── compiler_test.go
│           ├── mover_test.go
│           ├── cleaner_test.go
│           └── manifest_test.go
│
├── data/
│   └── tree-sitter/
│       └── grammar/                 # Final organized grammars (gitignored)
│       └── .gitkeep
│
├── build/                           # Intermediate artifacts (gitignored)
│   └── .gitkeep
│
├── docs/
│   └── planning/
│       └── design/
│           └── 001-tree-sitter-design.md  # This document
│
├── tree-sitter-config.yaml          # Configuration file
├── go.mod
├── go.sum
├── Makefile
├── LICENSE
├── NOTICE.md
├── README.md
└── .gitignore
```

---

## 7. Implementation Details

### 7.1 Config Package (`internal/tree-sitter/config.go`)

**Responsibilities:**
- Parse YAML configuration file
- Validate language definitions
- Provide structured access to configuration

**Key Types:**
```go
type Config struct {
    Version    string
    BuildDir   string
    Languages  []Language
    Output     Output
}

type Language struct {
    Name       string
    Version    string
    Repository string
    BuildMode  string  // "json", "scm", "both"
}

type Output struct {
    GrammarBase string
}
```

**Functions:**
- `LoadConfig(filePath string) (*Config, error)`
- `(c *Config) Validate() error`

---

### 7.2 CLI Package (`internal/tree-sitter/cli.go`)

**Responsibilities:**
- Wrap tree-sitter-cli commands
- Execute shell commands for building and compilation
- Provide clean Go interface to CLI operations

**Key Functions:**
- `BuildGrammar(ctx context.Context, lang Language, buildDir string) error`
- `CompileGrammar(ctx context.Context, lang Language, buildDir string, goos, goarch string) error`
- `ExecuteCommand(ctx context.Context, cmd string, args []string) error`

---

### 7.3 Build Package (`internal/tree-sitter/build/`)

#### Builder (`internal/tree-sitter/build/builder.go`)
- Clone grammar repositories
- Initialize grammar source in build directory
- Handle version/tag checkout

**Tests:** `internal/tests/build/builder_test.go`

#### Compiler (`internal/tree-sitter/build/compiler.go`)
- Orchestrate cross-platform compilation
- Manage GOOS/GOARCH environment variables
- Handle binary generation and naming

**Tests:** `internal/tests/build/compiler_test.go`

#### Mover (`internal/tree-sitter/build/mover.go`)
- Copy binaries to output directories
- Handle JSON and SCM mode logic
- Validate artifact presence before moving
- Create output directory structure
- Delegate manifest generation to manifest package

**Tests:** `internal/tests/build/mover_test.go`

#### Cleaner (`internal/tree-sitter/build/cleaner.go`)
- Remove build/{language}/ directories
- Ensure clean state for next build
- Provide recovery options if needed

**Tests:** `internal/tests/build/cleaner_test.go`

#### Manifest Generator (`internal/tree-sitter/build/manifest.go`)
- Generate manifest.json for each compiled grammar
- Calculate SHA256 checksums for binaries
- Record compile timestamp (ISO 8601)
- Include ABI version information from config
- List available artifacts (node-types.json, queries/, etc.)
- Support for future artifact types (wasm, etc.)

**Key Types:**
```go
type ManifestData struct {
    Grammar        string            `json:"grammar"`
    Version        string            `json:"version"`
    CompiledAt     string            `json:"compiled_at"`      // ISO 8601
    TreeSitterVer  string            `json:"tree_sitter_version"`
    Binaries       []BinaryInfo      `json:"binaries"`
    ABI            ABIInfo           `json:"abi"`
    Artifacts      ArtifactInfo      `json:"artifacts"`
}

type BinaryInfo struct {
    Platform      string `json:"platform"`  // "linux", "windows", "macos"
    Arch          string `json:"arch"`      // "amd64", "arm64"
    Filename      string `json:"filename"`
    ChecksumSHA256 string `json:"checksum_sha256"`
}

type ABIInfo struct {
    MinVersion    int    `json:"min_version"`
    MaxVersion    int    `json:"max_version"`
    ParserVersion string `json:"parser_version"`
}

type ArtifactInfo struct {
    HasNodeTypes bool `json:"has_node_types"`
    HasQueries   bool `json:"has_queries"`
    HasWasm      bool `json:"has_wasm"`
}
```

**Key Functions:**
- `GenerateManifest(grammarName, version, treeSitterVer string, binaries []string, config *Config) (*ManifestData, error)`
- `(m *ManifestData) WriteToFile(outputPath string) error`
- `CalculateChecksums(filePaths []string) (map[string]string, error)`

**Tests:** `internal/tests/build/manifest_test.go`

---

## 8. Workflow Example

**Scenario:** Build, compile, and deploy comprehensive grammar set covering code, config, and data with multi-ecosystem support.

## 8.1 NOTICE

build_mode: "both" should NOT be per language, but a "universal" setting with 3 modes:
- --json
- --scm
- --both

## TREE SITTER VERSIONS:

# Core Tree-Sitter Library
go get github.com/tree-sitter/go-tree-sitter@v0.25.0

# Core & Web Languages
go get github.com/tree-sitter/tree-sitter-python@v0.25.0
go get github.com/tree-sitter/tree-sitter-go@v0.25.0
go get github.com/tree-sitter/tree-sitter-javascript@v0.25.0

go get github.com/tree-sitter/tree-sitter-typescript@v0.23.2  <--- for typescript version 5.4.+
go get github.com/tree-sitter/tree-sitter-typescript/bindings/go@master <-- possible support for newer versions (typescript 5.9+) -- PREFERRED OVER REGULAR TYPESCRIPT
go get github.com/tree-sitter/tree-sitter-typescript/tsx/bindings/go@master <-- possible support for newer versions (typescript 5.9+) -- PREFERRED

go get github.com/tree-sitter/tree-sitter-html@v0.25.0
go get github.com/tree-sitter/tree-sitter-css@v0.25.0

# Data, Config, & Go Dependency Files
go get github.com/tree-sitter/tree-sitter-json@v0.24.8
go get github.com/tree-sitter-grammars/tree-sitter-yaml@latest
go get github.com/tree-sitter/tree-sitter-toml@v0.5.1
go get github.com/camdencheek/tree-sitter-go-mod@v1.1.0
go get github.com/tree-sitter-grammars/tree-sitter-go-sum@latest

# Query & RPC Languages
go get github.com/DerekStride/tree-sitter-sql@latest
go get github.com/coder3101/tree-sitter-proto@latest
go get github.com/taekwombo/tree-sitter-cypher@latest

the below config example SHOWS THE WRONG VERSIONS

## SCM - saved for later use
https://github.com/repowise-dev/repowise/blob/main/packages/core/src/repowise/core/ingestion/queries/python.scm
https://github.com/repowise-dev/repowise/blob/main/packages/core/src/repowise/core/ingestion/queries/typescript.scm
https://github.com/repowise-dev/repowise/blob/main/packages/core/src/repowise/core/ingestion/queries/go.scm
https://github.com/repowise-dev/repowise/blob/main/packages/core/src/repowise/core/ingestion/queries/java.scm
https://github.com/repowise-dev/repowise/blob/main/packages/core/src/repowise/core/ingestion/queries/javascript.scm

**Step 1: Configure (14 grammars - Project Focused)**
```yaml
languages:
  # Core Languages (3)
  - name: "python"
    version: "v0.20.8"
    tree_sitter_version: "0.20.8"
    build_mode: "both"
  - name: "go"
    version: "v0.20.0"
    tree_sitter_version: "0.20.0"
    build_mode: "both"
  - name: "typescript"
    version: "v0.20.5"
    tree_sitter_version: "0.20.5"
    build_mode: "both"

  # Web Stack (3)
  - name: "javascript"
    version: "v0.20.1"
    tree_sitter_version: "0.20.8"
    build_mode: "both"
  - name: "html"
    version: "v0.20.0"
    tree_sitter_version: "0.20.8"
    build_mode: "both"
  - name: "css"
    version: "v0.20.0"
    tree_sitter_version: "0.20.8"
    build_mode: "both"

  # Components (1)
  - name: "tsx"
    version: "v0.20.5"
    tree_sitter_version: "0.20.5"
    build_mode: "both"

  # Configuration (3)
  - name: "json"
    version: "v0.19.0"
    tree_sitter_version: "0.19.0"
    build_mode: "json"
  - name: "yaml"
    version: "v0.5.0"
    tree_sitter_version: "0.20.8"
    build_mode: "json"
  - name: "toml"
    version: "v0.5.2"
    tree_sitter_version: "0.20.8"
    build_mode: "json"

  # gRPC & Protocol Buffers (1) [CRITICAL FOR ARROW FLIGHT]
  - name: "proto"
    version: "v0.1.0"
    tree_sitter_version: "0.20.8"
    build_mode: "both"

  # Go Dependency Management (2)
  - name: "go.mod"
    version: "v0.1.1"
    tree_sitter_version: "0.20.8"
    build_mode: "json"
  - name: "go.sum"
    version: "v0.1.1"
    tree_sitter_version: "0.20.8"
    build_mode: "json"

  # Database Languages (2) [YOUR PRIMARY DATA LAYER]
  - name: "sql"
    version: "v1.17.0"
    tree_sitter_version: "0.20.8"
    build_mode: "both"
    note: "DuckDB + PostgreSQL dialects"
  - name: "cypher"
    version: "v1.4.0"
    tree_sitter_version: "0.20.8"
    build_mode: "both"
    note: "OpenCypher for Kuzu/Ladybug MCP graph DB"
```

**Step 2-3: Build & Compile**
```bash
tree-sitter build
tree-sitter compile --os linux --arch amd64
tree-sitter compile --os windows --arch amd64
tree-sitter compile --os macos --arch arm64
```

**Step 4: Move with Manifest Generation**
```bash
tree-sitter move --both
```

**Final State (14 Grammars):**

- The files in data/tree-sitter/grammar/{language}/ SHOULD be built by the compiler/move workflow, I should not have to create them manually.

```
data/tree-sitter/grammar/
├── python/                          [Core Languages]
├── go/
├── typescript/
├── javascript/                      [Web Stack]
├── html/
├── css/
├── tsx/                             [Components]
├── json/                            [Configuration]
├── yaml/
├── toml/
├── proto/                           [gRPC + Arrow Flight]
├── go.mod/                          [Go Dependencies]
├── go.sum/
├── sql/                             [Data Layer]
├── cypher/                          [Graph DB]
```

**Architecture Coverage:**
```
Your MCP Server Stack
├── Frontend: JS/TS/TSX/HTML/CSS
├── gRPC Layer: Proto (gRPC services + Arrow Flight)
├── Go Backend: Go language + go.mod/go.sum dependencies
├── Data Layer: SQL (DuckDB/PostgreSQL)
└── Graph DB: Cypher (Kuzu/Ladybug with OpenCypher)

Configuration: JSON, YAML, TOML
```

**Benefits for YOUR Project:**
- ✅ **gRPC Ready**: Proto grammar for service definitions and Arrow Flight schemas
- ✅ **Arrow Flight Support**: Via Proto grammar + JSON metadata
- ✅ **Complete Go Stack**: Language + dependency resolution (go.mod + go.sum)
- ✅ **DuckDB + PostgreSQL**: Single SQL grammar covers both dialects
- ✅ **Graph DB Ready**: Cypher for Kuzu/Ladybug OpenCypher queries
- ✅ **Universal Parser**: Once you have 14 grammars, cross-language analysis is trivial
- ✅ **MCP Integration**: All parsers feed directly into your graph DB

---

## 9. Manifest.json Design Details

### Purpose & Use Cases

**Runtime ABI Verification:**
```go
// Load manifest and check ABI compatibility
var manifest ManifestData
json.Unmarshal(manifestBytes, &manifest)
if myABIVersion >= manifest.ABI.MinVersion && myABIVersion <= manifest.ABI.MaxVersion {
    // Load this grammar's binary
}
```

**Tooling & Graph DB Integration:**
```sql
-- Query all grammars compiled after a certain date
SELECT grammar, version FROM manifest_index 
WHERE compiled_at > '2026-04-01'

-- Find binaries for specific platform
SELECT filename FROM manifests 
WHERE grammar='python' AND binaries.platform='linux' AND binaries.arch='amd64'

-- ABI compatibility analysis
SELECT grammar, abi.min_version, abi.max_version 
FROM manifests ORDER BY abi.max_version DESC
```

### Structure Rationale

**Why include all platforms in one manifest:**
- Single source of truth for a grammar version
- Easier discovery of available binaries
- Supports multi-platform deployments
- Common pattern in package ecosystems (npm, pip, cargo)

**Why include checksums:**
- Verification before loading binaries
- Integrity checking in CI/CD pipelines
- Tamper detection
- Standard practice for security

**Why explicit ABI version mapping:**
- tree-sitter parser ABI can change between versions
- Critical for runtime binary selection
- Prevents loading incompatible binaries
- Explicit mapping avoids runtime parsing errors

---

## 10. Error Handling Strategy

### Build Phase
- Missing config file → Fatal error with helpful path message
- Invalid repository URL → Skip language, log warning
- Clone failure → Retry once, then fail with error details
- Missing tree-sitter-cli → Fatal error with installation instructions

### Compile Phase
- Missing source in build/ → Error suggesting `build` command
- Compilation failure → Report compiler error output
- Unsupported platform combination → Skip gracefully
- GOOS/GOARCH validation → Validate before execution

### Move Phase
- Missing binaries → Error suggesting `compile` command
- Missing artifact files (node-types.json, queries/) → Warning, continue
- Manifest generation failure → Error (critical metadata)
- Checksum calculation failure → Error (integrity concern)
- Output directory creation failure → Error with permissions hint
- Preserve existing files; optionally allow `--force` to overwrite

### Manifest Generation
- Missing tree_sitter_version in config → Warn, attempt auto-detect from package.json
- Invalid ABI version mapping → Error with ABI version table reference
- Checksum read error → Fail, skip moving corrupted binary
- Encoding error → Fail with clear JSON error message

### Recovery
- `--force` flag on build to re-clone repositories
- `--force` flag on move to overwrite existing artifacts
- `--no-manifest` flag to skip manifest generation if issues arise
- Manual cleanup of `build/` if needed

---

## 10. Future Considerations

### Phase 2 Features
- **Watch mode:** Auto-rebuild on grammar source changes
- **Parallel compilation:** Build multiple languages simultaneously
- **Incremental builds:** Skip unchanged languages
- **Version management:** Track installed grammar versions
- **Health check:** Verify binaries work correctly before moving
- **Manifest indexing:** Generate index.json aggregating all grammar manifests

### Phase 3 Features
- **Remote storage:** Upload artifacts to cloud storage
- **Package generation:** Create distribution packages (tar.gz, zip)
- **Documentation generation:** Auto-generate grammar documentation
- **Testing integration:** Run grammar tests before deployment
- **Graph DB integration:** Auto-ingest manifests into graph database
- **Binary compatibility checker:** CLI tool to validate binary compatibility

---

## 11. Gitignore Entries

```gitignore
# Build artifacts
build/
*.o
*.a
*.so
*.exe
*.dll

# Generated grammar data (large binaries)
data/tree-sitter/grammar/
!data/tree-sitter/grammar/.gitkeep
```

---

## 12. Dependencies

### Go Modules
- `gopkg.in/yaml.v3` - YAML parsing
- Standard library: `os`, `exec`, `path/filepath`, `io/fs`, `context`

### External Tools
- `tree-sitter-cli` (must be installed and in $PATH)
- Go 1.26+

---

## 13. Testing Strategy

### Unit Tests
- Config parsing and validation
- Path construction and validation
- Mover logic (dry-run tests)

### Integration Tests
- End-to-end workflow with mock tree-sitter-cli
- Error handling for missing dependencies
- Artifact organization verification

### Manual Testing
- Real grammar build (Python, small grammar)
- Cross-platform compilation verification
- Artifact move and cleanup

---

## 14. Success Criteria

✓ Single YAML config file defines all 14 build targets with tree-sitter version mapping  
✓ Three CLI commands fully implement build workflow  
✓ Cross-platform binaries generated for Windows, macOS, Linux  
✓ JSON and SCM modes produce correct artifact organization  
✓ **manifest.json auto-generated for each grammar with ABI version info**  
✓ Build directory cleaned automatically after move  
✓ Clear error messages guide users on failures  
✓ No manual file organization required  
✓ Checksum validation prevents corrupted artifacts  
✓ **14 grammars enable comprehensive full-stack + gRPC analysis**  
✓ Support for community-maintained grammars (go.mod, go.sum, sql, cypher, proto)  
✓ **Protobuf grammar supports gRPC service definitions**  
✓ **Complete Go dependency graph via go.mod + go.sum**

---

## 15. Grammar Coverage Rationale

### Comprehensive 14-Grammar Set (Project-Focused)

| Grammar | Category | Tier | Use Case | Build Mode | Context |
|---------|----------|------|----------|------------|---------|
| **Python** | Language | Core | Server-side, scripts, ML | both | Primary backend |
| **Go** | Language | Core | Systems, cloud, APIs | both | Primary backend |
| **TypeScript** | Language | Core | Type-safe async code | both | Backend services |
| **JavaScript** | Language | Core | Client-side, Node.js | both | Frontend runtime |
| **HTML** | Web | Stack | DOM structure, templates | both | Frontend markup |
| **CSS** | Web | Stack | Styles, design systems | both | Frontend styling |
| **TSX** | Components | Stack | React/Vue components | both | Frontend components |
| **JSON** | Data | Config | APIs, metadata, Arrow schemas | json | Universal data exchange |
| **YAML** | Config | Data | K8s, CI/CD, IaC | json | Infrastructure configs |
| **TOML** | Config | Data | Rust/Python configs | json | Application configs |
| **Proto** | gRPC | **Core** | Service definitions, Arrow Flight | both | **gRPC server + Arrow Flight** |
| **go.mod** | Dependency | Mgmt | Declared dependencies | json | Go build graph |
| **go.sum** | Dependency | Lock | Resolved dependencies | json | Go version pinning |
| **SQL** | Database | Query | DuckDB + PostgreSQL dialects | both | **Primary DB parsers** |
| **Cypher** | Graph | Query | OpenCypher for Kuzu/Ladybug | both | **MCP graph database** |

### Project Context

**Your MCP Server Architecture:**
```
Frontend (JS/TS/TSX/HTML/CSS)
    ↓
Go Backend Services + gRPC (Proto + Arrow Flight)
    ↓
Data Layer: DuckDB + PostgreSQL (SQL)
    ↓
Graph DB: Kuzu/Ladybug with OpenCypher (Cypher)
    ↓
Configuration: YAML/TOML/JSON
```

**Grammar Coverage by Layer:**

**1. gRPC & Distributed Systems:**
   - **Proto** (essential): Define gRPC services and Arrow Flight schemas
   - Arrow Flight uses Arrow serialization (binary, covered by JSON for schema metadata)
   - Complete protocol definition coverage

**2. Go Ecosystem:**
   - **go.mod** + **go.sum**: Complete dependency resolution graph
   - Can trace all imports from entry point to transitive dependencies
   - Build graph analysis becomes possible once you have both

**3. Data Layer:**
   - **SQL**: DuckDB and PostgreSQL dialects (both based on PostgreSQL syntax)
   - **Cypher**: Query your graph DB (Ladybug/Kuzu with OpenCypher)
   - Two-database support for your current architecture

**4. Configuration & Metadata:**
   - JSON, YAML, TOML for all config files
   - Arrow schemas stored as JSON metadata (if needed)
   - Complete infrastructure-as-code coverage

**5. Frontend & Services:**
   - Full modern web stack (TS/TSX/JS/HTML/CSS)
   - Can parse any web-based component of your system

### Ecosystem Completeness

**Why 14 is optimal for YOUR project:**

1. **Covers all actual use cases:**
   - ✅ gRPC service definitions (Proto)
   - ✅ Arrow Flight protocol (via Proto + JSON)
   - ✅ Go builds and dependencies (go.mod + go.sum)
   - ✅ DuckDB/PostgreSQL queries (SQL)
   - ✅ Graph database queries (Cypher + Ladybug)
   - ✅ Full frontend stack (TS/JS/TSX/HTML/CSS)
   - ✅ Configuration management (YAML/TOML/JSON)

2. **Universal parser goal:**
   - Once you have the complete set, building multi-grammar analysis tools is trivial
   - Any future need for cross-language analysis uses existing grammars
   - No need to add new grammars for common patterns

3. **Maintenance burden is low:**
   - All from tree-sitter org or official community repos
   - Regular updates come from upstream
   - ~7 minute compile time for full set

### Why NOT included

**Arrow file format:**
- Arrow is a binary columnar storage format (not text-parseable)
- Arrow Flight is a gRPC protocol (covered by Proto grammar)
- Arrow schemas are defined in code or stored as JSON (covered by JSON grammar)
- **Recommendation**: If you need schema analysis, use JSON grammar for metadata files

**go.sum alone (initially):**
- Including both go.mod AND go.sum enables complete dependency analysis
- go.mod = declared dependencies (what you want)
- go.sum = resolved versions (what you actually use)
- Together they form a complete dependency graph

**Additional SQL dialects:**
- Starting with generic SQL covers both DuckDB and PostgreSQL
- Can add PostgreSQL-specific grammar later if needed
- DuckDB is heavily based on PostgreSQL syntax anyway

### Future Expansion Path

**Phase 2 (Project Evolution):**
- `package.json` - npm/node dependency tracking (if JS becomes more prominent)
- `Dockerfile` - Container definitions (if moving to containerized deployments)
- `Markdown` - Documentation parsing (if building doc automation)

**Phase 3 (Advanced Analysis):**
- `PostgreSQL` dialect grammar (if PostgreSQL becomes primary DB)
- `GraphQL` schema definitions (if adding GraphQL layer)
- Parquet format support (if using columnar storage alongside Arrow)

---

**Document End**
