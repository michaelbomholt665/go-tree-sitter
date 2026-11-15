module github.com/michaelbomholt665/go-tree-sitter

// Upstream's .git module suffix is not resolvable through the Go proxy.
replace github.com/joakker/tree-sitter-json5.git => github.com/Joakker/tree-sitter-json5 v0.1.1-0.20260505013339-248b85645670

go 1.26.0

// Upstream declares a www.github.com module path, but GitHub hosts the source here.
replace www.github.com/srazzak/tree-sitter-mdx => github.com/srazzak/tree-sitter-mdx v0.0.0-20251226211558-3aa29e8de1bf

// Upstream Swift/Zsh binding tests import these historical module paths.
replace github.com/tree-sitter/tree-sitter-swift => github.com/alex-pinkus/tree-sitter-swift v0.0.0-20261004031734-82bb3a533e08

replace github.com/tree-sitter/tree-sitter-zsh => github.com/georgeharker/tree-sitter-zsh v0.63.5

require (
	github.com/DerekStride/tree-sitter-sql v0.3.13
	github.com/UserNobody14/tree-sitter-dart v0.0.0-20261018035034-0038280b774b
	github.com/airbus-cert/tree-sitter-powershell v0.26.6
	github.com/coder3101/tree-sitter-proto v0.0.0-20261007121631-d7d15427321d
	github.com/fwcd/tree-sitter-kotlin v0.0.0-20261031170920-d6af96d2e23e
	github.com/gleam-lang/tree-sitter-gleam v1.1.1-0.20260918091344-fa6d0d94804f
	github.com/ionide/tree-sitter-fsharp v0.2.1-0.20260912141928-aefd0c8741bd
	github.com/joakker/tree-sitter-json5.git v0.1.1-0.20260505013339-248b85645670
	github.com/pupli/tree-sitter-cypher v0.0.2-0.20241111152014-775717a2de6c
	github.com/tree-sitter-grammars/tree-sitter-hcl v1.2.1-0.20260117144248-64ad62785d44
	github.com/tree-sitter-grammars/tree-sitter-objc v1.2.1-0.20250131075517-181a81b8f23a
	github.com/tree-sitter-grammars/tree-sitter-svelte v1.0.3-0.20241019183240-ae5199db4775
	github.com/tree-sitter-grammars/tree-sitter-toml v0.7.0
	github.com/tree-sitter-grammars/tree-sitter-yaml v0.7.3-0.20260522055615-a1c4812a73ec
	github.com/tree-sitter/go-tree-sitter v0.25.0
	github.com/tree-sitter/tree-sitter-c v0.24.2
	github.com/tree-sitter/tree-sitter-c-sharp v0.23.6-0.20260929081004-8c0abe0b84a3
	github.com/tree-sitter/tree-sitter-cpp v0.23.5-0.20260917200152-c00922280863
	github.com/tree-sitter/tree-sitter-css v0.25.0
	github.com/tree-sitter/tree-sitter-go v0.25.0
	github.com/tree-sitter/tree-sitter-html v0.23.2
	github.com/tree-sitter/tree-sitter-java v0.23.5
	github.com/tree-sitter/tree-sitter-javascript v0.25.0
	github.com/tree-sitter/tree-sitter-json v0.24.9-0.20260817103940-254c42a64764
	github.com/tree-sitter/tree-sitter-python v0.25.1-0.20250915213335-26855eabccb1
	github.com/tree-sitter/tree-sitter-rust v0.24.2
	github.com/tree-sitter/tree-sitter-swift v0.0.0-20261004031734-82bb3a533e08
	github.com/tree-sitter/tree-sitter-typescript v0.23.3-0.20250130221139-75b3874edb2d
	github.com/tree-sitter/tree-sitter-zsh v0.63.5
	github.com/victorhqc/tree-sitter-prisma v1.6.0
	gopkg.in/yaml.v3 v3.0.1
	www.github.com/srazzak/tree-sitter-mdx v0.0.0-20251226211558-3aa29e8de1bf
)

require (
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/fatih/color v1.19.0
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/tree-sitter-grammars/tree-sitter-lua v0.5.0
	// Legacy Make/Zig Go bindings; native grammar revisions are newer and pinned in YAML.
	github.com/tree-sitter-grammars/tree-sitter-make v1.1.1
	github.com/tree-sitter-grammars/tree-sitter-zig v1.1.2
	github.com/tree-sitter/tree-sitter-bash v0.25.1
	github.com/tree-sitter/tree-sitter-haskell v0.24.1
	github.com/tree-sitter/tree-sitter-julia v0.25.0
	github.com/tree-sitter/tree-sitter-ocaml v0.26.0
	github.com/tree-sitter/tree-sitter-php v0.25.1
	github.com/tree-sitter/tree-sitter-regex v0.25.0
	github.com/tree-sitter/tree-sitter-ruby v0.23.2-0.20260310062741-ad907a69da0c
	github.com/tree-sitter/tree-sitter-scala v0.26.13
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.4.0 // indirect
)
