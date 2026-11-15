# Third-Party Notices & Licenses

This notice records Go dependency scan results and the configured native grammar sources for release 1.10.0.

## Project information

- **Name:** go-tree-sitter (`ts-build`)
- **Version:** 1.10.0
- **License:** MIT; see [LICENSE](LICENSE).
- **Copyright:** (c) 2026 Michael Bomholt

## Scan scope and results

The following reports were generated with `go-licenses`:

```bash
go-licenses report ./cmd/ts-build ./cmd/sync-config
go-licenses check ./cmd/ts-build ./cmd/sync-config
GOFLAGS='-tags=grammars,grammar_imports' go-licenses report ./internal/grammars
GOFLAGS='-tags=grammars,grammar_imports' go-licenses check ./internal/grammars
```

The CLI check passed. The optional binding check failed because the classifier reports `Unknown` for `github.com/pupli/tree-sitter-cypher/bindings/go`.

`Unknown` means the tool did not recognize a license. It does not establish permission to redistribute that dependency. Resolve this classification before distributing the optional Cypher binding.

The report also warns that it cannot inspect non-Go dependencies. A binding module scan does not verify every native grammar license at its configured revision.

## Go dependencies

Versions below come from the selected Go module graph. License names and links come from the reports; they are classifications, not legal conclusions.

| Module | Version | Reported license | Applies to |
| --- | --- | --- | --- |
| `github.com/AlecAivazis/survey/v2` | `v2.3.7` | [MIT](https://github.com/AlecAivazis/survey/blob/v2.3.7/LICENSE) | CLI |
| `github.com/DerekStride/tree-sitter-sql` | `v0.3.13` | [MIT](https://github.com/DerekStride/tree-sitter-sql/blob/v0.3.13/LICENSE) | Optional bindings |
| `github.com/UserNobody14/tree-sitter-dart` | `v0.0.0-20261018035034-0038280b774b` | [MIT](https://github.com/UserNobody14/tree-sitter-dart/blob/0038280b774b/LICENSE) | Optional bindings |
| `github.com/airbus-cert/tree-sitter-powershell` | `v0.26.6` | [MIT](https://github.com/airbus-cert/tree-sitter-powershell/blob/v0.26.6/LICENSE) | Optional bindings |
| `github.com/coder3101/tree-sitter-proto` | `v0.0.0-20261007121631-d7d15427321d` | [MIT](https://github.com/coder3101/tree-sitter-proto/blob/d7d15427321d/LICENSE) | Optional bindings |
| `github.com/fatih/color` | `v1.19.0` | [MIT](https://github.com/fatih/color/blob/v1.19.0/LICENSE.md) | CLI |
| `github.com/fwcd/tree-sitter-kotlin` | `v0.0.0-20261031170920-d6af96d2e23e` | [MIT](https://github.com/fwcd/tree-sitter-kotlin/blob/d6af96d2e23e/LICENSE) | Optional bindings |
| `github.com/gleam-lang/tree-sitter-gleam` | `v1.1.1-0.20260918091344-fa6d0d94804f` | [Apache-2.0](https://github.com/gleam-lang/tree-sitter-gleam/blob/fa6d0d94804f/LICENSE) | Optional bindings |
| `github.com/ionide/tree-sitter-fsharp` | `v0.2.1-0.20260912141928-aefd0c8741bd` | [MIT](https://github.com/ionide/tree-sitter-fsharp/blob/aefd0c8741bd/LICENSE) | Optional bindings |
| `github.com/joakker/tree-sitter-json5.git` | `v0.1.1-0.20260505013339-248b85645670` | [MIT](https://github.com/Joakker/tree-sitter-json5/blob/248b85645670/LICENSE) | Optional bindings |
| `github.com/kballard/go-shellquote` | `v0.0.0-20180428030007-95032a82bc51` | [MIT](https://github.com/kballard/go-shellquote/blob/95032a82bc51/LICENSE) | CLI |
| `github.com/mattn/go-colorable` | `v0.1.14` | [MIT](https://github.com/mattn/go-colorable/blob/v0.1.14/LICENSE) | CLI |
| `github.com/mattn/go-isatty` | `v0.0.20` | [MIT](https://github.com/mattn/go-isatty/blob/v0.0.20/LICENSE) | CLI |
| `github.com/mattn/go-pointer` | `v0.0.1` | [MIT](https://github.com/mattn/go-pointer/blob/v0.0.1/LICENSE) | CLI, Optional bindings |
| `github.com/mgutz/ansi` | `v0.0.0-20170206155736-9520e82c474b` | [MIT](https://github.com/mgutz/ansi/blob/9520e82c474b/LICENSE) | CLI |
| `github.com/pupli/tree-sitter-cypher` | `v0.0.2-0.20241111152014-775717a2de6c` | Unknown | Optional bindings |
| `github.com/spf13/cobra` | `v1.10.2` | [Apache-2.0](https://github.com/spf13/cobra/blob/v1.10.2/LICENSE.txt) | CLI |
| `github.com/spf13/pflag` | `v1.0.9` | [BSD-3-Clause](https://github.com/spf13/pflag/blob/v1.0.9/LICENSE) | CLI |
| `github.com/tree-sitter-grammars/tree-sitter-hcl` | `v1.2.1-0.20260117144248-64ad62785d44` | [Apache-2.0](https://github.com/tree-sitter-grammars/tree-sitter-hcl/blob/64ad62785d44/LICENSE) | Optional bindings |
| `github.com/tree-sitter-grammars/tree-sitter-lua` | `v0.5.0` | [MIT](https://github.com/tree-sitter-grammars/tree-sitter-lua/blob/v0.5.0/LICENSE.md) | Optional bindings |
| `github.com/tree-sitter-grammars/tree-sitter-make` | `v1.1.1` | [MIT](https://github.com/tree-sitter-grammars/tree-sitter-make/blob/v1.1.1/LICENSE) | Optional bindings |
| `github.com/tree-sitter-grammars/tree-sitter-objc` | `v1.2.1-0.20250131075517-181a81b8f23a` | [MIT](https://github.com/tree-sitter-grammars/tree-sitter-objc/blob/181a81b8f23a/LICENSE) | Optional bindings |
| `github.com/tree-sitter-grammars/tree-sitter-svelte` | `v1.0.3-0.20241019183240-ae5199db4775` | [MIT](https://github.com/tree-sitter-grammars/tree-sitter-svelte/blob/ae5199db4775/LICENSE) | Optional bindings |
| `github.com/tree-sitter-grammars/tree-sitter-toml` | `v0.7.0` | [MIT](https://github.com/tree-sitter-grammars/tree-sitter-toml/blob/v0.7.0/LICENSE) | Optional bindings |
| `github.com/tree-sitter-grammars/tree-sitter-yaml` | `v0.7.3-0.20260522055615-a1c4812a73ec` | [MIT](https://github.com/tree-sitter-grammars/tree-sitter-yaml/blob/a1c4812a73ec/LICENSE) | Optional bindings |
| `github.com/tree-sitter-grammars/tree-sitter-zig` | `v1.1.2` | [MIT](https://github.com/tree-sitter-grammars/tree-sitter-zig/blob/v1.1.2/LICENSE) | Optional bindings |
| `github.com/tree-sitter/go-tree-sitter` | `v0.25.0` | [MIT](https://github.com/tree-sitter/go-tree-sitter/blob/v0.25.0/LICENSE) | CLI, Optional bindings |
| `github.com/tree-sitter/tree-sitter-bash` | `v0.25.1` | [MIT](https://github.com/tree-sitter/tree-sitter-bash/blob/v0.25.1/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-c` | `v0.24.2` | [MIT](https://github.com/tree-sitter/tree-sitter-c/blob/v0.24.2/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-c-sharp` | `v0.23.6-0.20260929081004-8c0abe0b84a3` | [MIT](https://github.com/tree-sitter/tree-sitter-c-sharp/blob/8c0abe0b84a3/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-cpp` | `v0.23.5-0.20260917200152-c00922280863` | [MIT](https://github.com/tree-sitter/tree-sitter-cpp/blob/c00922280863/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-css` | `v0.25.0` | [MIT](https://github.com/tree-sitter/tree-sitter-css/blob/v0.25.0/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-go` | `v0.25.0` | [MIT](https://github.com/tree-sitter/tree-sitter-go/blob/v0.25.0/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-haskell` | `v0.24.1` | [MIT](https://github.com/tree-sitter/tree-sitter-haskell/blob/v0.24.1/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-html` | `v0.23.2` | [MIT](https://github.com/tree-sitter/tree-sitter-html/blob/v0.23.2/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-java` | `v0.23.5` | [MIT](https://github.com/tree-sitter/tree-sitter-java/blob/v0.23.5/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-javascript` | `v0.25.0` | [MIT](https://github.com/tree-sitter/tree-sitter-javascript/blob/v0.25.0/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-json` | `v0.24.9-0.20260817103940-254c42a64764` | [MIT](https://github.com/tree-sitter/tree-sitter-json/blob/254c42a64764/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-julia` | `v0.25.0` | [MIT](https://github.com/tree-sitter/tree-sitter-julia/blob/v0.25.0/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-ocaml` | `v0.26.0` | [MIT](https://github.com/tree-sitter/tree-sitter-ocaml/blob/v0.26.0/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-php` | `v0.25.1` | [MIT](https://github.com/tree-sitter/tree-sitter-php/blob/v0.25.1/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-python` | `v0.25.1-0.20250915213335-26855eabccb1` | [MIT](https://github.com/tree-sitter/tree-sitter-python/blob/26855eabccb1/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-regex` | `v0.25.0` | [MIT](https://github.com/tree-sitter/tree-sitter-regex/blob/v0.25.0/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-ruby` | `v0.23.2-0.20260310062741-ad907a69da0c` | [MIT](https://github.com/tree-sitter/tree-sitter-ruby/blob/ad907a69da0c/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-rust` | `v0.24.2` | [MIT](https://github.com/tree-sitter/tree-sitter-rust/blob/v0.24.2/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-scala` | `v0.26.13` | [MIT](https://github.com/tree-sitter/tree-sitter-scala/blob/v0.26.13/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-swift` | `v0.0.0-20261004031734-82bb3a533e08` | [MIT](https://github.com/alex-pinkus/tree-sitter-swift/blob/82bb3a533e08/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-typescript` | `v0.23.3-0.20250130221139-75b3874edb2d` | [MIT](https://github.com/tree-sitter/tree-sitter-typescript/blob/75b3874edb2d/LICENSE) | Optional bindings |
| `github.com/tree-sitter/tree-sitter-zsh` | `v0.63.5` | [MIT](https://github.com/georgeharker/tree-sitter-zsh/blob/v0.63.5/LICENSE) | Optional bindings |
| `github.com/victorhqc/tree-sitter-prisma` | `v1.6.0` | [MIT](https://github.com/victorhqc/tree-sitter-prisma/blob/v1.6.0/LICENSE) | Optional bindings |
| `golang.org/x/sys` | `v0.42.0` | [BSD-3-Clause](https://cs.opensource.google/go/x/sys/+/v0.42.0:LICENSE) | CLI |
| `golang.org/x/term` | `v0.0.0-20210927222741-03fcf44c2211` | [BSD-3-Clause](https://cs.opensource.google/go/x/term/+/03fcf44c:LICENSE) | CLI |
| `golang.org/x/text` | `v0.4.0` | [BSD-3-Clause](https://cs.opensource.google/go/x/text/+/v0.4.0:LICENSE) | CLI |
| `gopkg.in/yaml.v3` | `v3.0.1` | [MIT](https://github.com/go-yaml/yaml/blob/v3.0.1/LICENSE) | CLI |
| `www.github.com/srazzak/tree-sitter-mdx` | `v0.0.0-20251226211558-3aa29e8de1bf` | [MIT](https://github.com/srazzak/tree-sitter-mdx/blob/3aa29e8de1bf/LICENSE) | Optional bindings |

## Native grammar source inventory

The selected repository configuration contains 65 grammar entries. Multiple entries can share one upstream repository.

This table records the current package labels and full source pins. The final column preserves labels from the previous NOTICE; those labels were not reverified at these pins.

`Not recorded` means the previous notice supplied no label. `Unverified` marks the unresolved Cypher source; its earlier MIT label is not retained as a verified finding.

Before distributing native sources or binaries, inspect the upstream license and attribution requirements at the exact configured revision. Do not treat this inventory as a complete license-text bundle.

| Grammar | Repository | Package label | Revision | Previously recorded license (not reverified) |
| --- | --- | --- | --- | --- |
| `bash` | https://github.com/tree-sitter/tree-sitter-bash.git | `v0.25.1` | `a06c2e4415e9bc0346c6b86d401879ffb44058f7` | MIT |
| `blade` | https://github.com/EmranMR/tree-sitter-blade.git | `v0.12.3` | `cc764dadcbbceb3f259396fef66f970c72e94f96` | MIT |
| `c` | https://github.com/tree-sitter/tree-sitter-c.git | `v0.24.2` | `b780e47fc780ddc8da13afa35a3f4ed5c157823d` | MIT |
| `clojure` | https://github.com/sogaiu/tree-sitter-clojure.git | `v0.0.13` | `e43eff80d17cf34852dcd92ca5e6986d23a7040f` | Not recorded |
| `cmake` | https://github.com/uyha/tree-sitter-cmake.git | `v0.7.6` | `a3ec2b2144daf7132e7ee824d8cab9610523d10f` | MIT |
| `cpp` | https://github.com/tree-sitter/tree-sitter-cpp.git | `v0.23.5-0.20260917200152-c00922280863` | `c009222808634c1014f82438d4883753516a2c24` | MIT |
| `csharp` | https://github.com/tree-sitter/tree-sitter-c-sharp.git | `v0.23.6-0.20260929081004-8c0abe0b84a3` | `8c0abe0b84a3681d3e6852ce2e2a1eecb4f731c5` | MIT |
| `css` | https://github.com/tree-sitter/tree-sitter-css.git | `v0.25.0` | `dda5cfc5722c429eaba1c910ca32c2c0c5bb1a3f` | MIT |
| `cue` | https://github.com/eonpatapon/tree-sitter-cue.git | `v0.2.0` | `eefee1c1ee76ccf0d7880814155b67c1738d5f89` | Not recorded |
| `cypher` | https://github.com/simplificare-org/tree-sitter-cypher.git | `v0.0.2-0.20241111152014-775717a2de6c` | `775717a2de6ca76a5a75e8076b411dbc1ae4c281` | Unverified |
| `dart` | https://github.com/UserNobody14/tree-sitter-dart.git | `v0.0.0-20261018035034-0038280b774b` | `0038280b774b65c1b89ba7fec50624d8baed9fe1` | Not recorded |
| `dockerfile` | https://github.com/camdencheek/tree-sitter-dockerfile.git | `v0.2.0` | `868e44ce378deb68aac902a9db68ff82d2299dd0` | MIT |
| `elixir` | https://github.com/elixir-lang/tree-sitter-elixir.git | `v0.3.5` | `e2d9e6e0e76b0c436fa48a0b8c32a031d0cbdf49` | Apache-2.0 |
| `fish` | https://github.com/ram02z/tree-sitter-fish.git | `v3.7.0` | `b7f1d682941e0c62dfcd6bf9ef481638351bab16` | Not recorded |
| `fsharp` | https://github.com/ionide/tree-sitter-fsharp.git | `v0.2.1-0.20260912141928-aefd0c8741bd` | `aefd0c8741bdf3aeb827a228aa4a996a6536697e` | Not recorded |
| `fsharp-signature` | https://github.com/ionide/tree-sitter-fsharp.git | `v0.2.1-0.20260912141928-aefd0c8741bd` | `aefd0c8741bdf3aeb827a228aa4a996a6536697e` | Not recorded |
| `gleam` | https://github.com/gleam-lang/tree-sitter-gleam.git | `v1.1.1-0.20260918091344-fa6d0d94804f` | `fa6d0d94804f1b342e175865a3dbeb316ee5115e` | Not recorded |
| `go` | https://github.com/tree-sitter/tree-sitter-go.git | `v0.25.0` | `1547678a9da59885853f5f5cc8a99cc203fa2e2c` | MIT |
| `go.mod` | https://github.com/camdencheek/tree-sitter-go-mod.git | `v1.1.1-0.20251023052317-2e886870578e` | `2e886870578eeba1927a2dc4bd2e2b3f598c5f9a` | MIT |
| `go.sum` | https://github.com/tree-sitter-grammars/tree-sitter-go-sum.git | `v1.0.0` | `afd6da80a9509c2edbc0f3ea540564d100babd67` | MIT |
| `groovy` | https://github.com/murtaza64/tree-sitter-groovy.git | `v0.0.0-20260927220505-2a6ddd558b6a` | `2a6ddd558b6aa39c5b77d8db9fe9baf817486b2c` | Not recorded |
| `haskell` | https://github.com/tree-sitter/tree-sitter-haskell.git | `v0.24.1` | `97288e585b0bd44199720280d067fa35bacd81ff` | MIT |
| `hcl` | https://github.com/tree-sitter-grammars/tree-sitter-hcl.git | `v1.2.1-0.20260117144248-64ad62785d44` | `64ad62785d442eb4d45df3a1764962dafd5bc98b` | Not recorded |
| `html` | https://github.com/tree-sitter/tree-sitter-html.git | `v0.23.2` | `5a5ca8551a179998360b4a4ca2c0f366a35acc03` | MIT |
| `java` | https://github.com/tree-sitter/tree-sitter-java.git | `v0.23.5` | `94703d5a6bed02b98e438d7cad1136c01a60ba2c` | MIT |
| `javascript` | https://github.com/tree-sitter/tree-sitter-javascript.git | `v0.25.0` | `44c892e0be055ac465d5eeddae6d3e194424e7de` | MIT |
| `json` | https://github.com/tree-sitter/tree-sitter-json.git | `v0.24.9-0.20260817103940-254c42a64764` | `254c42a6476413b776221e03982ac8ae159eeb72` | MIT |
| `json5` | https://github.com/Joakker/tree-sitter-json5.git | `v0.1.1-0.20260505013339-248b85645670` | `248b8564567087d7866be76569b182f6dd7e14e9` | Not recorded |
| `julia` | https://github.com/tree-sitter/tree-sitter-julia.git | `v0.25.0` | `e0f9dcd180fdcfcfa8d79a3531e11d99e79321d3` | MIT |
| `kotlin` | https://github.com/fwcd/tree-sitter-kotlin.git | `v0.0.0-20261031170920-d6af96d2e23e` | `d6af96d2e23ef0a8ff5ab512864378cf08fba311` | Not recorded |
| `lean` | https://github.com/Julian/tree-sitter-lean.git | `v0.2.0` | `0d28e5d6e8c4755f5df7ef9bde3b6ec6f59fae94` | Not recorded |
| `less` | https://github.com/mdovale/tree-sitter-less.git | `v0.0.0-20260415221443-02988c765d30` | `02988c765d30adb0476657b5d220e8dfde1c07d3` | Not recorded |
| `lua` | https://github.com/tree-sitter-grammars/tree-sitter-lua.git | `v0.5.0` | `10fe0054734eec83049514ea2e718b2a56acd0c9` | MIT |
| `make` | https://github.com/tree-sitter-grammars/tree-sitter-make.git | `v1.1.2-0.20260226090825-70613f3d812c` | `70613f3d812cbabbd7f38d104d60a409c4008b43` | MIT |
| `markdown` | https://github.com/tree-sitter-grammars/tree-sitter-markdown.git | `v0.5.4` | `41ff24a6c775492c543b09228aeb98d1cd2b3a0f` | MIT |
| `markdown-inline` | https://github.com/tree-sitter-grammars/tree-sitter-markdown.git | `v0.5.4` | `41ff24a6c775492c543b09228aeb98d1cd2b3a0f` | MIT |
| `mdx` | https://github.com/srazzak/tree-sitter-mdx.git | `v0.0.0-20251226211558-3aa29e8de1bf` | `3aa29e8de1bf0213948a04fe953039b6ab73777b` | Not recorded |
| `nix` | https://github.com/cstrahan/tree-sitter-nix.git | `v0.3.0` | `64d4f7827dc1a8c7c88108ffdb47b961a729df05` | Not recorded |
| `objc` | https://github.com/tree-sitter-grammars/tree-sitter-objc.git | `v1.2.1-0.20250131075517-181a81b8f23a` | `181a81b8f23a2d593e7ab4259981f50122909fda` | Not recorded |
| `ocaml` | https://github.com/tree-sitter/tree-sitter-ocaml.git | `v0.26.0` | `e3c9cf368f68bffd2f81188229aefa7b434eda65` | MIT |
| `ocaml-interface` | https://github.com/tree-sitter/tree-sitter-ocaml.git | `v0.26.0` | `e3c9cf368f68bffd2f81188229aefa7b434eda65` | MIT |
| `php` | https://github.com/tree-sitter/tree-sitter-php.git | `v0.25.1` | `58d5643086ae60e93a3746cca648363ee97c8761` | MIT |
| `php_only` | https://github.com/tree-sitter/tree-sitter-php.git | `v0.25.1` | `58d5643086ae60e93a3746cca648363ee97c8761` | MIT |
| `powershell` | https://github.com/airbus-cert/tree-sitter-powershell.git | `v0.26.6` | `65ea6919e816a7ac142acec7ee72757f89db8439` | Not recorded |
| `prisma` | https://github.com/victorhqc/tree-sitter-prisma.git | `v1.6.0` | `3556b2c1f20ec9ac91e92d32c43d9d2a0ca3cc49` | Not recorded |
| `proto` | https://github.com/coder3101/tree-sitter-proto.git | `v0.0.0-20261007121631-d7d15427321d` | `d7d15427321da271cb1c25c12735e180ef981473` | MIT |
| `python` | https://github.com/tree-sitter/tree-sitter-python.git | `v0.25.1-0.20250915213335-26855eabccb1` | `26855eabccb19c6abf499fbc5b8dc7cc9ab8bc64` | MIT |
| `regex` | https://github.com/tree-sitter/tree-sitter-regex.git | `v0.25.0` | `b2ac15e27fce703d2f37a79ccd94a5c0cbe9720b` | MIT |
| `ruby` | https://github.com/tree-sitter/tree-sitter-ruby.git | `v0.23.2-0.20260310062741-ad907a69da0c` | `ad907a69da0c8a4f7a943a7fe012712208da6dee` | MIT |
| `rust` | https://github.com/tree-sitter/tree-sitter-rust.git | `v0.24.2` | `77a3747266f4d621d0757825e6b11edcbf991ca5` | MIT |
| `sass` | https://github.com/bajrangCoder/tree-sitter-sass.git | `v0.1.0` | `fb280c41b070657e4ff4d4e5e6eea6cb19efd9b8` | MIT |
| `scala` | https://github.com/tree-sitter/tree-sitter-scala.git | `v0.26.13` | `e8728b1c62ea096c753a9a70325f77ebbf2192de` | MIT |
| `scss` | https://github.com/tree-sitter-grammars/tree-sitter-scss.git | `v1.0.0` | `eb69c8f55ecc17235f43358c65ec644578554f92` | MIT |
| `solidity` | https://github.com/JoranHonig/tree-sitter-solidity.git | `v1.2.13` | `4e938a46c7030dd001bc99e1ac0f0c750ac98254` | MIT |
| `sql` | https://github.com/DerekStride/tree-sitter-sql.git | `v0.3.13` | `4cd7224bfcf1fee23227c4d626760341c9b26772` | MIT |
| `svelte` | https://github.com/tree-sitter-grammars/tree-sitter-svelte.git | `v1.0.3-0.20241019183240-ae5199db4775` | `ae5199db47757f785e43a14b332118a5474de1a2` | Not recorded |
| `swift` | https://github.com/alex-pinkus/tree-sitter-swift.git | `v0.0.0-20261004031734-82bb3a533e08` | `82bb3a533e0801fd2bbaa11dc49676e10bf41948` | Not recorded |
| `toml` | https://github.com/tree-sitter-grammars/tree-sitter-toml.git | `v0.7.0` | `64b56832c2cffe41758f28e05c756a3a98d16f41` | MIT |
| `tsx` | https://github.com/tree-sitter/tree-sitter-typescript.git | `v0.23.3-0.20250130221139-75b3874edb2d` | `75b3874edb2dc714fb1fd77a32013d0f8699989f` | MIT |
| `typescript` | https://github.com/tree-sitter/tree-sitter-typescript.git | `v0.23.3-0.20250130221139-75b3874edb2d` | `75b3874edb2dc714fb1fd77a32013d0f8699989f` | MIT |
| `typst` | https://github.com/uben0/tree-sitter-typst.git | `v0.11.1-0.20250210143020-46cf4ded12ee` | `46cf4ded12ee974a70bf8457263b67ad7ee0379d` | Not recorded |
| `vue` | https://github.com/tree-sitter-grammars/tree-sitter-vue.git | `v0.1.0` | `ce8011a414fdf8091f4e4071752efc376f4afb08` | Not recorded |
| `yaml` | https://github.com/tree-sitter-grammars/tree-sitter-yaml.git | `v0.7.3-0.20260522055615-a1c4812a73ec` | `a1c4812a73ec5e089de8e441fdea3a921e8d5079` | MIT |
| `zig` | https://github.com/tree-sitter-grammars/tree-sitter-zig.git | `v1.1.3-0.20250910133711-6479aa13f32f` | `6479aa13f32f701c383083d8b28360ebd682fb7d` | MIT |
| `zsh` | https://github.com/georgeharker/tree-sitter-zsh.git | `v0.63.5` | `7a593401efb5418ffdedbe3c0e4c61c6d240166d` | Not recorded |

## Redistribution

Preserve required copyright notices and license texts from the dependencies and source revisions you distribute. Follow any additional attribution requirements in those licenses.

This file does not assert that every configured grammar has a verified permissive license. The optional Cypher classification and exact-revision native source review remain unresolved.
