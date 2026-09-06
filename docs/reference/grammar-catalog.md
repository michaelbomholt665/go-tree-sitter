# Supported Grammar Catalog & Symbol Registry

This document lists all 38 officially configured grammars in [`tree-sitter-config.yaml`](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml) and their bindings in [`internal/grammars/registry.go`](file:///home/michael/projects/go/tree-sitter/internal/grammars/registry.go).

---

## Grammar Matrix

| Identifier (`name`) | Grammar (`grammar`) | Exported Constructor | Pinned Version | Pinned Revision SHA | Subdir |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `bash` | `bash` | `tree_sitter_bash` | `v0.25.1` | `a06c2e4415e9bc0346c6b86d401879ffb44058f7` | |
| `c` | `c` | `tree_sitter_c` | `v0.24.2` | `b780e47fc780ddc8da13afa35a3f4ed5c157823d` | |
| `c-sharp` | `c_sharp` | `tree_sitter_c_sharp` | `v0.23.5` | `cac6d5fb595f5811a076336682d5d595ac1c9e85` | |
| `cmake` | `cmake` | `tree_sitter_cmake` | `v0.7.4` | `ca627bb5828616b6246aafdc3c3222789e728e37` | |
| `cpp` | `cpp` | `tree_sitter_cpp` | `v0.23.4` | `f41e1a044c8a84ea9fa8577fdd2eab92ec96de02` | |
| `css` | `css` | `tree_sitter_css` | `v0.25.0` | `dda5cfc5722c429eaba1c910ca32c2c0c5bb1a3f` | |
| `cypher` | `cypher` | `tree_sitter_cypher` | `v0.0.2-0.20241111152014-775717a2de6c` | `775717a2de6ca76a5a75e8076b411dbc1ae4c281` | |
| `dockerfile` | `dockerfile` | `tree_sitter_dockerfile` | `v0.2.0` | `868e44ce378deb68aac902a9db68ff82d2299dd0` | |
| `elixir` | `elixir` | `tree_sitter_elixir` | `v0.3.5` | `e2d9e6e0e76b0c436fa48a0b8c32a031d0cbdf49` | |
| `go` | `go` | `tree_sitter_go` | `v0.25.0` | `1547678a9da59885853f5f5cc8a99cc203fa2e2c` | |
| `go.mod` | `gomod` | `tree_sitter_gomod` | `v1.1.0` | `3b01edce2b9ea6766ca19328d1850e456fde3103` | |
| `go.sum` | `gosum` | `tree_sitter_gosum` | `v1.0.0` | `afd6da80a9509c2edbc0f3ea540564d100babd67` | |
| `haskell` | `haskell` | `tree_sitter_haskell` | `v0.23.1` | `c30d812bc90827f1a54106a25bc9a6307f5cdcec` | |
| `html` | `html` | `tree_sitter_html` | `v0.23.2` | `5a5ca8551a179998360b4a4ca2c0f366a35acc03` | |
| `java` | `java` | `tree_sitter_java` | `v0.23.5` | `94703d5a6bed02b98e438d7cad1136c01a60ba2c` | |
| `javascript` | `javascript` | `tree_sitter_javascript` | `v0.25.0` | `44c892e0be055ac465d5eeddae6d3e194424e7de` | |
| `json` | `json` | `tree_sitter_json` | `v0.24.8` | `ee35a6ebefcef0c5c416c0d1ccec7370cfca5a24` | |
| `julia` | `julia` | `tree_sitter_julia` | `v0.25.0` | `e0f9dcd180fdcfcfa8d79a3531e11d99e79321d3` | |
| `lua` | `lua` | `tree_sitter_lua` | `v0.5.0` | `10fe0054734eec83049514ea2e718b2a56acd0c9` | |
| `make` | `make` | `tree_sitter_make` | `v1.1.1` | `5e9e8f8ff3387b0edcaa90f46ddf3629f4cfeb1d` | |
| `markdown` | `markdown` | `tree_sitter_markdown` | `v0.5.3` | `f969cd3ae3f9fbd4e43205431d0ae286014c05b5` | `tree-sitter-markdown` |
| `markdown-inline` | `markdown_inline` | `tree_sitter_markdown_inline` | `v0.5.3` | `f969cd3ae3f9fbd4e43205431d0ae286014c05b5` | `tree-sitter-markdown-inline` |
| `ocaml` | `ocaml` | `tree_sitter_ocaml` | `v0.25.0` | `6902a86ab5b3b80c622030210aae2d8cb95eb775` | `grammars/ocaml` |
| `ocaml-interface` | `ocaml_interface` | `tree_sitter_ocaml_interface` | `v0.25.0` | `6902a86ab5b3b80c622030210aae2d8cb95eb775` | `grammars/interface` |
| `php` | `php` | `tree_sitter_php` | `v0.24.2` | `5b5627faaa290d89eb3d01b9bf47c3bb9e797dea` | `php` |
| `php_only` | `php_only` | `tree_sitter_php_only` | `v0.24.2` | `5b5627faaa290d89eb3d01b9bf47c3bb9e797dea` | `php_only` |
| `proto` | `proto` | `tree_sitter_proto` | `v0.0.0-20260901194749-6c878d18628e` | `6c878d18628ebbff3474479d2fdd6d8ba1954c3e` | |
| `python` | `python` | `tree_sitter_python` | `v0.25.0` | `293fdc02038ee2bf0e2e206711b69c90ac0d413f` | |
| `regex` | `regex` | `tree_sitter_regex` | `v0.25.0` | `b2ac15e27fce703d2f37a79ccd94a5c0cbe9720b` | |
| `ruby` | `ruby` | `tree_sitter_ruby` | `v0.23.1` | `71bd32fb7607035768799732addba884a37a6210` | |
| `rust` | `rust` | `tree_sitter_rust` | `v0.24.2` | `77a3747266f4d621d0757825e6b11edcbf991ca5` | |
| `scala` | `scala` | `tree_sitter_scala` | `v0.26.2` | `b931fcc338390925eb893d70ad070033f5856ccf` | |
| `sql` | `sql` | `tree_sitter_sql` | `v0.3.11` | `1156b55f26a5949784827f21c6abc72c05e26600` | |
| `toml` | `toml` | `tree_sitter_toml` | `v0.7.0` | `64b56832c2cffe41758f28e05c756a3a98d16f41` | |
| `tsx` | `tsx` | `tree_sitter_tsx` | `v0.23.3-0.20250130221139-75b3874edb2d` | `75b3874edb2dc714fb1fd77a32013d0f8699989f` | `tsx` |
| `typescript` | `typescript` | `tree_sitter_typescript` | `v0.23.3-0.20250130221139-75b3874edb2d` | `75b3874edb2dc714fb1fd77a32013d0f8699989f` | `typescript` |
| `yaml` | `yaml` | `tree_sitter_yaml` | `v0.7.2` | `7708026449bed86239b1cd5bce6e3c34dbca6415` | |
| `zig` | `zig` | `tree_sitter_zig` | `v1.1.2` | `b670c8df85a1568f498aa5c8cae42f51a90473c0` | |

---

## Repositories, Licenses & Smoke Test Samples

| Identifier | Upstream Repository | License | Smoke Test Sample |
| :--- | :--- | :--- | :--- |
| `bash` | [tree-sitter/tree-sitter-bash](https://github.com/tree-sitter/tree-sitter-bash.git) | MIT | `echo "hello world"` |
| `c` | [tree-sitter/tree-sitter-c](https://github.com/tree-sitter/tree-sitter-c.git) | MIT | `int main(void) { return 0; }` |
| `c-sharp` | [tree-sitter/tree-sitter-c-sharp](https://github.com/tree-sitter/tree-sitter-c-sharp.git) | MIT | `class Program { static void Main() {} }` |
| `cmake` | [uyha/tree-sitter-cmake](https://github.com/uyha/tree-sitter-cmake.git) | MIT | `cmake_minimum_required(VERSION 3.10)` |
| `cpp` | [tree-sitter/tree-sitter-cpp](https://github.com/tree-sitter/tree-sitter-cpp.git) | MIT | `#include <vector>\nint main() { std::vector<int> values; }` |
| `css` | [tree-sitter/tree-sitter-css](https://github.com/tree-sitter/tree-sitter-css.git) | MIT | `a { color: red; }` |
| `cypher` | [pupli/tree-sitter-cypher](https://github.com/pupli/tree-sitter-cypher.git) | MIT | `MATCH (n) RETURN n;` |
| `dockerfile` | [camdencheek/tree-sitter-dockerfile](https://github.com/camdencheek/tree-sitter-dockerfile.git) | Apache-2.0 | `FROM alpine:latest` |
| `elixir` | [elixir-lang/tree-sitter-elixir](https://github.com/elixir-lang/tree-sitter-elixir.git) | Apache-2.0 | `IO.puts("hello")` |
| `go` | [tree-sitter/tree-sitter-go](https://github.com/tree-sitter/tree-sitter-go.git) | MIT | `package p` |
| `go.mod` | [camdencheek/tree-sitter-go-mod](https://github.com/camdencheek/tree-sitter-go-mod.git) | Apache-2.0 | `module example.com/x\n\ngo 1.23` |
| `go.sum` | [tree-sitter-grammars/tree-sitter-go-sum](https://github.com/tree-sitter-grammars/tree-sitter-go-sum.git) | Apache-2.0 | `example.com/x v1.0.0 h1:AAAA...` |
| `haskell` | [tree-sitter/tree-sitter-haskell](https://github.com/tree-sitter/tree-sitter-haskell.git) | MIT | `main = putStrLn "hello"` |
| `html` | [tree-sitter/tree-sitter-html](https://github.com/tree-sitter/tree-sitter-html.git) | MIT | `<div></div>` |
| `java` | [tree-sitter/tree-sitter-java](https://github.com/tree-sitter/tree-sitter-java.git) | MIT | `class A {}` |
| `javascript` | [tree-sitter/tree-sitter-javascript](https://github.com/tree-sitter/tree-sitter-javascript.git) | MIT | `const x = 1;` |
| `json` | [tree-sitter/tree-sitter-json](https://github.com/tree-sitter/tree-sitter-json.git) | MIT | `{}` |
| `julia` | [tree-sitter/tree-sitter-julia](https://github.com/tree-sitter/tree-sitter-julia.git) | MIT | `println("hello")` |
| `lua` | [tree-sitter-grammars/tree-sitter-lua](https://github.com/tree-sitter-grammars/tree-sitter-lua.git) | MIT | `print("hello")` |
| `make` | [tree-sitter-grammars/tree-sitter-make](https://github.com/tree-sitter-grammars/tree-sitter-make.git) | MIT | `all:\n\t@echo hello` |
| `markdown` | [tree-sitter-grammars/tree-sitter-markdown](https://github.com/tree-sitter-grammars/tree-sitter-markdown.git) | MIT | `# Hello` |
| `markdown-inline` | [tree-sitter-grammars/tree-sitter-markdown](https://github.com/tree-sitter-grammars/tree-sitter-markdown.git) | MIT | `hello *world*` |
| `ocaml` | [tree-sitter/tree-sitter-ocaml](https://github.com/tree-sitter/tree-sitter-ocaml.git) | MIT | `let x = 1` |
| `ocaml-interface` | [tree-sitter/tree-sitter-ocaml](https://github.com/tree-sitter/tree-sitter-ocaml.git) | MIT | `val x : int` |
| `php` | [tree-sitter/tree-sitter-php](https://github.com/tree-sitter/tree-sitter-php.git) | MIT | `<?php echo 'hello';` |
| `php_only` | [tree-sitter/tree-sitter-php](https://github.com/tree-sitter/tree-sitter-php.git) | MIT | `echo 'hello';` |
| `proto` | [coder3101/tree-sitter-proto](https://github.com/coder3101/tree-sitter-proto.git) | MIT | `syntax = "proto3"; message X {}` |
| `python` | [tree-sitter/tree-sitter-python](https://github.com/tree-sitter/tree-sitter-python.git) | MIT | `x = 1` |
| `regex` | [tree-sitter/tree-sitter-regex](https://github.com/tree-sitter/tree-sitter-regex.git) | MIT | `abc` |
| `ruby` | [tree-sitter/tree-sitter-ruby](https://github.com/tree-sitter/tree-sitter-ruby.git) | MIT | `puts 'hello'` |
| `rust` | [tree-sitter/tree-sitter-rust](https://github.com/tree-sitter/tree-sitter-rust.git) | MIT | `fn main() { let x: i32 = 1; }` |
| `scala` | [tree-sitter/tree-sitter-scala](https://github.com/tree-sitter/tree-sitter-scala.git) | MIT | `val x = 1` |
| `sql` | [DerekStride/tree-sitter-sql](https://github.com/DerekStride/tree-sitter-sql.git) | MIT | `SELECT 1;` |
| `toml` | [tree-sitter-grammars/tree-sitter-toml](https://github.com/tree-sitter-grammars/tree-sitter-toml.git) | MIT | `x = 1` |
| `tsx` | [tree-sitter/tree-sitter-typescript](https://github.com/tree-sitter/tree-sitter-typescript.git) | MIT | `const x = <div />;` |
| `typescript` | [tree-sitter/tree-sitter-typescript](https://github.com/tree-sitter/tree-sitter-typescript.git) | MIT | `const x: number = 1;` |
| `yaml` | [tree-sitter-grammars/tree-sitter-yaml](https://github.com/tree-sitter-grammars/tree-sitter-yaml.git) | MIT | `x: 1` |
| `zig` | [tree-sitter-grammars/tree-sitter-zig](https://github.com/tree-sitter-grammars/tree-sitter-zig.git) | MIT | `pub fn main() void {}` |

---

## Go Binding Aliases in `GetLanguage`

The [`GetLanguage`](file:///home/michael/projects/go/tree-sitter/internal/grammars/registry.go#L28) function in `internal/grammars/registry.go` recognizes convenient alias variations:

- `"bash"` ➔ `bashbinding.Language()`
- `"c"` ➔ `cbinding.Language()`
- `"c-sharp"`, `"c_sharp"`, `"csharp"` ➔ `csharpbinding.Language()`
- `"cpp"`, `"c++"` ➔ `cppbinding.Language()`
- `"css"` ➔ `cssbinding.Language()`
- `"cypher"` ➔ `cypherbinding.Language()`
- `"go"` ➔ `gobinding.Language()`
- `"haskell"` ➔ `haskellbinding.Language()`
- `"html"` ➔ `htmlbinding.Language()`
- `"java"` ➔ `javabinding.Language()`
- `"javascript"` ➔ `javascriptbinding.Language()`
- `"json"` ➔ `jsonbinding.Language()`
- `"julia"` ➔ `juliabinding.Language()`
- `"lua"` ➔ `luabinding.Language()`
- `"make"`, `"makefile"` ➔ `makebinding.Language()`
- `"ocaml"` ➔ `ocamlbinding.LanguageOCaml()`
- `"ocaml-interface"`, `"ocaml_interface"` ➔ `ocamlbinding.LanguageOCamlInterface()`
- `"php"` ➔ `phpbinding.LanguagePHP()`
- `"php_only"`, `"php-only"` ➔ `phpbinding.LanguagePHPOnly()`
- `"proto"` ➔ `protobinding.Language()`
- `"python"` ➔ `pythonbinding.Language()`
- `"regex"` ➔ `regexbinding.Language()`
- `"ruby"` ➔ `rubybinding.Language()`
- `"rust"` ➔ `rustbinding.Language()`
- `"scala"` ➔ `scalabinding.Language()`
- `"toml"` ➔ `tomlbinding.Language()`
- `"tsx"` ➔ `typescriptbinding.LanguageTSX()`
- `"typescript"` ➔ `typescriptbinding.LanguageTypescript()`
- `"yaml"` ➔ `yamlbinding.Language()`
- `"zig"` ➔ `zigbinding.Language()`
