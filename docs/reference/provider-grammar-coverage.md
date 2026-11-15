# Provider-to-grammar coverage

This inventory maps `/mnt/storage/projects/csharp/lsp/PROVIDER_CATALOG.md` to build configuration, not published artifacts.
The inspected catalog identifies upstream baseline `6255a12431e9d46e7da3b3c210847b9d89debc11`.
It contains 45 in-scope primary providers and five auxiliary providers; ordinal 2 (Deno) is excluded.
The upstream baseline has 46 primary providers when Deno is included.
Four fallback descriptors duplicate language targets: python-jedi, omnisharp, expert, and docker-official.
Internal executable alternatives do not introduce languages.
The configuration preserves the original 42 identifiers and adds 23, for 65 total.

## Primary providers and extensions

An identifier in this table means a configured grammar, not full language conformance.
Extensions are routing decisions for downstream consumers; this repository does not implement provider dispatch.

| Provider IDs | Extensions or filenames | Grammar identifiers |
| --- | --- | --- |
| typescript | `.cjs .js .jsx .mjs .cts .mts .ts .tsx` | javascript; typescript; tsx |
| python; python-jedi (fallback) | `.py .pyi` | python |
| go | `.go` | go |
| rust | `.rs` | rust |
| ruby | `.gemspec .rake .rb .ru` | ruby |
| php | `.php` | php |
| powershell | `.ps1 .psm1 .psd1` | powershell |
| csharp; omnisharp (fallback) | `.cs` | csharp |
| fsharp | `.fs .fsx`; `.fsi` | fsharp; fsharp-signature |
| java | `.java` | java |
| kotlin | `.kt .kts` | kotlin |
| groovy | `.groovy .gvy .gy .gsh` | groovy |
| lean | `.lean` | lean |
| swift | `.swift` | swift |
| dart | `.dart` | dart |
| lua | `.lua` | lua |
| cpp | `.c .h`; `.cc .cpp .cxx .hh .hpp .hxx .ino`; `.m .mm` | c; cpp; objc (partial for `.mm`) |
| zig | `.zig .zon` | zig |
| haskell | `.hs .lhs` | haskell (literate preprocessing is separate) |
| elixir; expert (fallback) | `.ex .exs` | elixir |
| gleam | `.gleam` | gleam |
| tinymist | `.typ .typc` | typst |
| marksman | `.md .markdown` | markdown; markdown-inline |
| ocaml | `.ml`; `.mli` | ocaml; ocaml-interface |
| clojure | `.clj .cljc .cljs .edn` | clojure |
| cue | `.cue` | cue |
| terraform | `.tf .tfvars` | hcl |
| nix | `.nix` | nix |
| bash | `.bash .sh` without a Zsh dialect header | bash |
| shuck | `.zsh`; Zsh-dialect `.sh` or `.bash` | zsh |
| fish | `.fish` | fish |
| cmake | `.cmake`, `CMakeLists.txt` | cmake |
| docker; docker-official (fallback) | `.dockerfile`, `Dockerfile` | dockerfile |
| yaml | `.yaml .yml` | yaml |
| json | `.json`; `.json5`; `.jsonc` | json; json5; json or json5 (see below) |
| html | `.htm .html` | html |
| toml | `.toml` | toml |
| prisma | `.prisma` | prisma |
| vue | `.vue` | vue |
| svelte | `.svelte` | svelte |
| css | `.css .less .sass .scss` | css; less; sass; scss |

Ruby's ruby-lsp/solargraph/rubocop chain shares ruby.
Python's pyright/basedpyright/ty alternatives share python.
Kotlin's kotlin-lsp/kotlin-language-server alternatives share kotlin.
Groovy's binary/JAR alternatives share groovy.
TypeScript's Effect/native/classic alternatives share javascript, typescript, and tsx.
Svelte's svelteserver/svelte-language-server alternatives share svelte.

## Auxiliary providers and aliases

| Provider | Catalog language kinds or paths | Normalization |
| --- | --- | --- |
| opengrep | csharp, css, cxx, dart, docker, go, html, java, json, jsts, kotlin, lua, php, python, ruby, rust, shell, swift, terraform, yaml | Direct identifiers except the aliases below |
| ast-grep | csharp, cxx, css, elixir, go, haskell, html, java, json, jsts, kotlin, lua, nix, php, python, ruby, rust, scala, shell, swift, yaml | Direct identifiers except the aliases below; scala already exists |
| zizmor | `.github/workflows/*.yaml`/`*.yml`, `action.yaml`/`action.yml`, `.github/dependabot.yaml`/`*.yml` | yaml; only catalog-approved paths |
| typos | All opengrep-kind extensions plus `.md .markdown .mdx` | Same extension mapping, plus markdown/markdown-inline and mdx |
| fallow | `.cjs .cts .js .jsx .mjs .mts .ts .tsx` | javascript, typescript, tsx; excludes vue and svelte |

Normalize aliases before counting coverage:

- `JS`, `jsx`, and JavaScript use javascript; `TS` uses typescript; `.tsx` uses tsx.
- `jsts` expands to javascript, typescript, and tsx.
- `C#`, `c-sharp`, and `c_sharp` use csharp.
- `C++` and `cxx` select c or cpp by file type; Objective-C extensions select objc.
- `F#` uses fsharp; `.fsi` uses fsharp-signature.
- `docker` uses dockerfile; `tinymist` uses typst; `shuck` uses zsh.
- `terraform`, `terraform-vars`, and HCL use hcl.
- `shell` requires dialect selection: bash for POSIX sh/Bash, zsh for Zsh, fish for Fish.
- `ocaml_interface` uses ocaml-interface.
- Deno is a runtime, not another grammar. Its JS/TS files use javascript, typescript, and tsx.

## Syntax decisions and limits

- **Objective-C++ remains partial coverage.** The selected objc grammar extends C, not C++.
  Use objc for `.m`; `.mm` needs combined Objective-C/C++ parsing that neither objc nor cpp provides alone.
  Do not report complete `.mm` coverage from the clangd provider's broad extension list.
- `.ino` uses cpp. Arduino preprocessing and generated declarations are outside grammar parsing.
  `.h` needs project context to distinguish C from C++.
- JSON5 needs json5 for unquoted keys, single quotes, and trailing commas.
  The existing pinned json grammar accepts line/block comments, so it handles comments-only JSONC.
  Route trailing-comma JSONC through json5, a permissive superset; this is not strict JSONC validation.
  A separate jsonc grammar is unnecessary. Its original upstream README declares it obsolete.
- MDX needs mdx, not plain markdown: JSX and JavaScript expressions change the syntax.
  Less needs less, not plain css. Indented Sass and SCSS retain their existing separate identifiers.
- POSIX sh uses bash as a syntax superset; no separate sh grammar is required.
  This does not certify POSIX conformance. Header-based Zsh detection precedes extension selection.
- `.tfvars` contains HCL assignments. The LSP language ID `terraform-vars` does not require a second grammar.
  Provider extension coverage does not include `.tfvars.json`; JSON-form Terraform uses json if a consumer adds that route.
- Vue and Svelte grammars describe component structure. Embedded JS/TS/CSS parsing requires consumer-side query injections.
- Clojure, ClojureScript, reader conditionals, and EDN share clojure; this is not an independent conformance claim for each dialect.
- `.lhs` requires consumer-side literate extraction. No new Haskell grammar or preprocessor is included.
- No original grammar is removed because the catalog omits it, including scala and downstream data/query grammars.

## Revision evidence and integration

Each added entry records `language_support.assessed_revision` equal to its full immutable source pin.
Evidence URLs identify upstream documentation at that same revision.
See [the language-version research report](provider-language-version-research.md) for the 23 assessed pins, findings, search limits, and Kotlin fixture-report discrepancy.
Package versions identify dependencies, not supported source-language versions.
Spec links and examples do not establish version ranges.
Lean's README explicitly describes limited static Lean 4 syntax and exclusions; its assessment records that scope without a version range.
Other additions retain unverified version support unless an explicit upstream claim establishes a narrower scope.
No version-specific fixtures were authorized or used as version evidence.

Upstream Go bindings enter `internal/grammars/registry.go` and `cmd/sync-config/main.go` when usable.
Grammars without upstream Go bindings remain native build targets, following the existing native-only pattern.
Groovy, Typst, and Less contain nested Go modules that cannot be imported from the grammar repository root at these pins.
Their module paths also differ from the selected repositories; these three grammars use native builds instead.
The other native-only additions are Lean, Clojure, CUE, Nix, Fish, and Vue.
Swift uses an immutable upstream generated-files revision because ordinary source revisions omit `src/parser.c` required by Go bindings.
Fourteen additions have static Go registry entries; nine use native builds only.
JSON5 and MDX declare non-resolvable module paths. Pinned `go.mod` replacements map those declarations to their actual GitHub repositories.
Swift and Zsh use their historical module import paths because upstream binding tests still import those paths.
Pinned replacements resolve both paths to the selected fork revisions, so `go mod tidy` can resolve dependency tests.
The configuration contains the exact repositories, versions, constructors, subdirectories, samples, and assessment evidence.

Run the ordinary unit suite and the optional registry suite:

```bash
go test ./...
go test -tags grammars ./internal/tests/grammars
git diff --check
```

These checks do not publish grammars.
Native build/compile checks use temporary build paths, never `move` or production manifests.
The opt-in native smoke test expects `tree-sitter-<name>/<source_subdir>/src/node-types.json` below a temporary root.
It loads each library from `compiled/<name>.so` and checks the runtime ABI and configured sample.
Set `TS_PROVIDER_SMOKE_ROOT` only after generating and compiling all 23 additions at their configured pins:

```bash
TS_PROVIDER_SMOKE_ROOT=/tmp/provider-grammars go test ./internal/tests/build -run '^TestProviderNativeSmoke$' -count=1
```

These smoke samples check integration, not source-language version conformance.
All test files live below `internal/tests/`; production packages do not contain `*_test.go` files.
Publish only through the normal binary-validated build/compile/move workflow after a separate publication request.
