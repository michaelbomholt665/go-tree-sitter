# Supported Grammar Catalog & Symbol Registry

This document lists the 65 configured grammar entries in `tree-sitter-config.yaml`. The tagged, repository-internal Go registry covers only a subset; it is not an external consumer API.

Make and Zig no longer provide Go bindings in their current upstream revisions. Their published native parsers use the revisions below.
The optional static Go registry retains Make `v1.1.1` and Zig `v1.1.2` bindings for compatibility.
`sync-config` excludes these two legacy modules so it cannot replace the newer native grammar versions.


See provider grammar coverage for all provider mappings, aliases, extensions, fallback providers, and documented limitations.
---

## Grammar Matrix

| Identifier (`name`) | Grammar (`grammar`) | Exported Constructor | Pinned Version | Pinned Revision SHA | Subdir |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `bash` | `bash` | `tree_sitter_bash` | `v0.25.1` | `a06c2e4415e9bc0346c6b86d401879ffb44058f7` | |
| `blade` | `blade` | `tree_sitter_blade` | `v0.12.3` | `cc764dadcbbceb3f259396fef66f970c72e94f96` | |
| `c` | `c` | `tree_sitter_c` | `v0.24.2` | `b780e47fc780ddc8da13afa35a3f4ed5c157823d` | |
| `csharp` | `c_sharp` | `tree_sitter_c_sharp` | `v0.23.6-0.20260929081004-8c0abe0b84a3` | `8c0abe0b84a3681d3e6852ce2e2a1eecb4f731c5` | |
| `cmake` | `cmake` | `tree_sitter_cmake` | `v0.7.6` | `a3ec2b2144daf7132e7ee824d8cab9610523d10f` | |
| `cpp` | `cpp` | `tree_sitter_cpp` | `v0.23.5-0.20260917200152-c00922280863` | `c009222808634c1014f82438d4883753516a2c24` | |
| `css` | `css` | `tree_sitter_css` | `v0.25.0` | `dda5cfc5722c429eaba1c910ca32c2c0c5bb1a3f` | |
| `cypher` | `cypher` | `tree_sitter_cypher` | `v0.0.2-0.20241111152014-775717a2de6c` | `775717a2de6ca76a5a75e8076b411dbc1ae4c281` | |
| `dockerfile` | `dockerfile` | `tree_sitter_dockerfile` | `v0.2.0` | `868e44ce378deb68aac902a9db68ff82d2299dd0` | |
| `elixir` | `elixir` | `tree_sitter_elixir` | `v0.3.5` | `e2d9e6e0e76b0c436fa48a0b8c32a031d0cbdf49` | |
| `go` | `go` | `tree_sitter_go` | `v0.25.0` | `1547678a9da59885853f5f5cc8a99cc203fa2e2c` | |
| `go.mod` | `gomod` | `tree_sitter_gomod` | `v1.1.1-0.20251023052317-2e886870578e` | `2e886870578eeba1927a2dc4bd2e2b3f598c5f9a` | |
| `go.sum` | `gosum` | `tree_sitter_gosum` | `v1.0.0` | `afd6da80a9509c2edbc0f3ea540564d100babd67` | |
| `haskell` | `haskell` | `tree_sitter_haskell` | `v0.24.1` | `97288e585b0bd44199720280d067fa35bacd81ff` | |
| `html` | `html` | `tree_sitter_html` | `v0.23.2` | `5a5ca8551a179998360b4a4ca2c0f366a35acc03` | |
| `java` | `java` | `tree_sitter_java` | `v0.23.5` | `94703d5a6bed02b98e438d7cad1136c01a60ba2c` | |
| `javascript` | `javascript` | `tree_sitter_javascript` | `v0.25.0` | `44c892e0be055ac465d5eeddae6d3e194424e7de` | |
| `json` | `json` | `tree_sitter_json` | `v0.24.9-0.20260817103940-254c42a64764` | `254c42a6476413b776221e03982ac8ae159eeb72` | |
| `julia` | `julia` | `tree_sitter_julia` | `v0.25.0` | `e0f9dcd180fdcfcfa8d79a3531e11d99e79321d3` | |
| `lua` | `lua` | `tree_sitter_lua` | `v0.5.0` | `10fe0054734eec83049514ea2e718b2a56acd0c9` | |
| `make` | `make` | `tree_sitter_make` | `v1.1.2-0.20260226090825-70613f3d812c` | `70613f3d812cbabbd7f38d104d60a409c4008b43` | |
| `markdown` | `markdown` | `tree_sitter_markdown` | `v0.5.4` | `41ff24a6c775492c543b09228aeb98d1cd2b3a0f` | `tree-sitter-markdown` |
| `markdown-inline` | `markdown_inline` | `tree_sitter_markdown_inline` | `v0.5.4` | `41ff24a6c775492c543b09228aeb98d1cd2b3a0f` | `tree-sitter-markdown-inline` |
| `ocaml` | `ocaml` | `tree_sitter_ocaml` | `v0.26.0` | `e3c9cf368f68bffd2f81188229aefa7b434eda65` | `grammars/ocaml` |
| `ocaml-interface` | `ocaml_interface` | `tree_sitter_ocaml_interface` | `v0.26.0` | `e3c9cf368f68bffd2f81188229aefa7b434eda65` | `grammars/interface` |
| `php` | `php` | `tree_sitter_php` | `v0.25.1` | `58d5643086ae60e93a3746cca648363ee97c8761` | `php` |
| `php_only` | `php_only` | `tree_sitter_php_only` | `v0.25.1` | `58d5643086ae60e93a3746cca648363ee97c8761` | `php_only` |
| `proto` | `proto` | `tree_sitter_proto` | `v0.0.0-20261007121631-d7d15427321d` | `d7d15427321da271cb1c25c12735e180ef981473` | |
| `python` | `python` | `tree_sitter_python` | `v0.25.1-0.20250915213335-26855eabccb1` | `26855eabccb19c6abf499fbc5b8dc7cc9ab8bc64` | |
| `regex` | `regex` | `tree_sitter_regex` | `v0.25.0` | `b2ac15e27fce703d2f37a79ccd94a5c0cbe9720b` | |
| `ruby` | `ruby` | `tree_sitter_ruby` | `v0.23.2-0.20260310062741-ad907a69da0c` | `ad907a69da0c8a4f7a943a7fe012712208da6dee` | |
| `rust` | `rust` | `tree_sitter_rust` | `v0.24.2` | `77a3747266f4d621d0757825e6b11edcbf991ca5` | |
| `sass` | `sass` | `tree_sitter_sass` | `v0.1.0` | `fb280c41b070657e4ff4d4e5e6eea6cb19efd9b8` | |
| `scala` | `scala` | `tree_sitter_scala` | `v0.26.13` | `e8728b1c62ea096c753a9a70325f77ebbf2192de` | |
| `scss` | `scss` | `tree_sitter_scss` | `v1.0.0` | `eb69c8f55ecc17235f43358c65ec644578554f92` | |
| `solidity` | `solidity` | `tree_sitter_solidity` | `v1.2.13` | `4e938a46c7030dd001bc99e1ac0f0c750ac98254` | |
| `sql` | `sql` | `tree_sitter_sql` | `v0.3.13` | `4cd7224bfcf1fee23227c4d626760341c9b26772` | |
| `toml` | `toml` | `tree_sitter_toml` | `v0.7.0` | `64b56832c2cffe41758f28e05c756a3a98d16f41` | |
| `tsx` | `tsx` | `tree_sitter_tsx` | `v0.23.3-0.20250130221139-75b3874edb2d` | `75b3874edb2dc714fb1fd77a32013d0f8699989f` | `tsx` |
| `typescript` | `typescript` | `tree_sitter_typescript` | `v0.23.3-0.20250130221139-75b3874edb2d` | `75b3874edb2dc714fb1fd77a32013d0f8699989f` | `typescript` |
| `yaml` | `yaml` | `tree_sitter_yaml` | `v0.7.3-0.20260522055615-a1c4812a73ec` | `a1c4812a73ec5e089de8e441fdea3a921e8d5079` | |
| `zig` | `zig` | `tree_sitter_zig` | `v1.1.3-0.20250910133711-6479aa13f32f` | `6479aa13f32f701c383083d8b28360ebd682fb7d` | |
| `powershell` | `powershell` | `tree_sitter_powershell` | `v0.26.6` | `65ea6919e816a7ac142acec7ee72757f89db8439` |  |
| `fsharp` | `fsharp` | `tree_sitter_fsharp` | `v0.2.1-0.20260912141928-aefd0c8741bd` | `aefd0c8741bdf3aeb827a228aa4a996a6536697e` | fsharp |
| `fsharp-signature` | `fsharp_signature` | `tree_sitter_fsharp_signature` | `v0.2.1-0.20260912141928-aefd0c8741bd` | `aefd0c8741bdf3aeb827a228aa4a996a6536697e` | fsharp_signature |
| `kotlin` | `kotlin` | `tree_sitter_kotlin` | `v0.0.0-20261031170920-d6af96d2e23e` | `d6af96d2e23ef0a8ff5ab512864378cf08fba311` |  |
| `groovy` | `groovy` | `tree_sitter_groovy` | `v0.0.0-20260927220505-2a6ddd558b6a` | `2a6ddd558b6aa39c5b77d8db9fe9baf817486b2c` |  |
| `lean` | `lean` | `tree_sitter_lean` | `v0.2.0` | `0d28e5d6e8c4755f5df7ef9bde3b6ec6f59fae94` |  |
| `swift` | `swift` | `tree_sitter_swift` | `v0.0.0-20261004031734-82bb3a533e08` | `82bb3a533e0801fd2bbaa11dc49676e10bf41948` |  |
| `dart` | `dart` | `tree_sitter_dart` | `v0.0.0-20261018035034-0038280b774b` | `0038280b774b65c1b89ba7fec50624d8baed9fe1` |  |
| `gleam` | `gleam` | `tree_sitter_gleam` | `v1.1.1-0.20260918091344-fa6d0d94804f` | `fa6d0d94804f1b342e175865a3dbeb316ee5115e` |  |
| `typst` | `typst` | `tree_sitter_typst` | `v0.11.1-0.20250210143020-46cf4ded12ee` | `46cf4ded12ee974a70bf8457263b67ad7ee0379d` |  |
| `clojure` | `clojure` | `tree_sitter_clojure` | `v0.0.13` | `e43eff80d17cf34852dcd92ca5e6986d23a7040f` |  |
| `cue` | `cue` | `tree_sitter_cue` | `v0.2.0` | `eefee1c1ee76ccf0d7880814155b67c1738d5f89` |  |
| `hcl` | `hcl` | `tree_sitter_hcl` | `v1.2.1-0.20260117144248-64ad62785d44` | `64ad62785d442eb4d45df3a1764962dafd5bc98b` |  |
| `nix` | `nix` | `tree_sitter_nix` | `v0.3.0` | `64d4f7827dc1a8c7c88108ffdb47b961a729df05` |  |
| `zsh` | `zsh` | `tree_sitter_zsh` | `v0.63.5` | `7a593401efb5418ffdedbe3c0e4c61c6d240166d` |  |
| `fish` | `fish` | `tree_sitter_fish` | `v3.7.0` | `b7f1d682941e0c62dfcd6bf9ef481638351bab16` |  |
| `prisma` | `prisma` | `tree_sitter_prisma` | `v1.6.0` | `3556b2c1f20ec9ac91e92d32c43d9d2a0ca3cc49` |  |
| `vue` | `vue` | `tree_sitter_vue` | `v0.1.0` | `ce8011a414fdf8091f4e4071752efc376f4afb08` |  |
| `svelte` | `svelte` | `tree_sitter_svelte` | `v1.0.3-0.20241019183240-ae5199db4775` | `ae5199db47757f785e43a14b332118a5474de1a2` |  |
| `objc` | `objc` | `tree_sitter_objc` | `v1.2.1-0.20250131075517-181a81b8f23a` | `181a81b8f23a2d593e7ab4259981f50122909fda` |  |
| `json5` | `json5` | `tree_sitter_json5` | `v0.1.1-0.20260505013339-248b85645670` | `248b8564567087d7866be76569b182f6dd7e14e9` |  |
| `less` | `less` | `tree_sitter_less` | `v0.0.0-20260415221443-02988c765d30` | `02988c765d30adb0476657b5d220e8dfde1c07d3` |  |
| `mdx` | `mdx` | `tree_sitter_mdx` | `v0.0.0-20251226211558-3aa29e8de1bf` | `3aa29e8de1bf0213948a04fe953039b6ab73777b` |  |

---


Use `ts-build list` or `ts-build list --language <name> --json` to inspect the active configuration; this matrix is a repository snapshot.
