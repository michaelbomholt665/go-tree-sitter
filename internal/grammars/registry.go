//go:build grammars

package grammars

import (
	"unsafe"

	sitter "github.com/tree-sitter/go-tree-sitter"

	protobinding "github.com/coder3101/tree-sitter-proto/bindings/go"
	cypherbinding "github.com/pupli/tree-sitter-cypher/bindings/go"
	tomlbinding "github.com/tree-sitter-grammars/tree-sitter-toml/bindings/go"
	yamlbinding "github.com/tree-sitter-grammars/tree-sitter-yaml/bindings/go"
	csharpbinding "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	cbinding "github.com/tree-sitter/tree-sitter-c/bindings/go"
	cppbinding "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	cssbinding "github.com/tree-sitter/tree-sitter-css/bindings/go"
	gobinding "github.com/tree-sitter/tree-sitter-go/bindings/go"
	htmlbinding "github.com/tree-sitter/tree-sitter-html/bindings/go"
	javabinding "github.com/tree-sitter/tree-sitter-java/bindings/go"
	javascriptbinding "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	jsonbinding "github.com/tree-sitter/tree-sitter-json/bindings/go"
	pythonbinding "github.com/tree-sitter/tree-sitter-python/bindings/go"
	rustbinding "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	typescriptbinding "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	luabinding "github.com/tree-sitter-grammars/tree-sitter-lua/bindings/go"
	makebinding "github.com/tree-sitter-grammars/tree-sitter-make/bindings/go"
	zigbinding "github.com/tree-sitter-grammars/tree-sitter-zig/bindings/go"
	bashbinding "github.com/tree-sitter/tree-sitter-bash/bindings/go"
	haskellbinding "github.com/tree-sitter/tree-sitter-haskell/bindings/go"
	juliabinding "github.com/tree-sitter/tree-sitter-julia/bindings/go"
	ocamlbinding "github.com/tree-sitter/tree-sitter-ocaml/bindings/go"
	phpbinding "github.com/tree-sitter/tree-sitter-php/bindings/go"
	regexbinding "github.com/tree-sitter/tree-sitter-regex/bindings/go"
	rubybinding "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	scalabinding "github.com/tree-sitter/tree-sitter-scala/bindings/go"
)

func GetLanguage(lang string) *sitter.Language {
	var ptr unsafe.Pointer

	switch lang {
	case "bash":
		ptr = unsafe.Pointer(bashbinding.Language())
	case "c":
		ptr = unsafe.Pointer(cbinding.Language())
	case "c-sharp", "c_sharp", "csharp":
		ptr = unsafe.Pointer(csharpbinding.Language())
	case "cpp", "c++":
		ptr = unsafe.Pointer(cppbinding.Language())
	case "css":
		ptr = unsafe.Pointer(cssbinding.Language())
	case "cypher":
		ptr = unsafe.Pointer(cypherbinding.Language())
	case "go":
		ptr = unsafe.Pointer(gobinding.Language())
	case "haskell":
		ptr = unsafe.Pointer(haskellbinding.Language())
	case "html":
		ptr = unsafe.Pointer(htmlbinding.Language())
	case "java":
		ptr = unsafe.Pointer(javabinding.Language())
	case "javascript":
		ptr = unsafe.Pointer(javascriptbinding.Language())
	case "json":
		ptr = unsafe.Pointer(jsonbinding.Language())
	case "julia":
		ptr = unsafe.Pointer(juliabinding.Language())
	case "lua":
		ptr = unsafe.Pointer(luabinding.Language())
	case "make", "makefile":
		ptr = unsafe.Pointer(makebinding.Language())
	case "ocaml":
		ptr = unsafe.Pointer(ocamlbinding.LanguageOCaml())
	case "ocaml-interface", "ocaml_interface":
		ptr = unsafe.Pointer(ocamlbinding.LanguageOCamlInterface())
	case "php":
		ptr = unsafe.Pointer(phpbinding.LanguagePHP())
	case "php_only", "php-only":
		ptr = unsafe.Pointer(phpbinding.LanguagePHPOnly())
	case "proto":
		ptr = unsafe.Pointer(protobinding.Language())
	case "python":
		ptr = unsafe.Pointer(pythonbinding.Language())
	case "regex":
		ptr = unsafe.Pointer(regexbinding.Language())
	case "ruby":
		ptr = unsafe.Pointer(rubybinding.Language())
	case "rust":
		ptr = unsafe.Pointer(rustbinding.Language())
	case "scala":
		ptr = unsafe.Pointer(scalabinding.Language())
	case "toml":
		ptr = unsafe.Pointer(tomlbinding.Language())
	case "tsx":
		ptr = unsafe.Pointer(typescriptbinding.LanguageTSX())
	case "typescript":
		ptr = unsafe.Pointer(typescriptbinding.LanguageTypescript())
	case "yaml":
		ptr = unsafe.Pointer(yamlbinding.Language())
	case "zig":
		ptr = unsafe.Pointer(zigbinding.Language())
	default:
		return nil
	}

	return sitter.NewLanguage(ptr)
}
