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
)

func GetLanguage(lang string) *sitter.Language {
	var ptr unsafe.Pointer

	switch lang {
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
	case "html":
		ptr = unsafe.Pointer(htmlbinding.Language())
	case "java":
		ptr = unsafe.Pointer(javabinding.Language())
	case "javascript":
		ptr = unsafe.Pointer(javascriptbinding.Language())
	case "json":
		ptr = unsafe.Pointer(jsonbinding.Language())
	case "proto":
		ptr = unsafe.Pointer(protobinding.Language())
	case "python":
		ptr = unsafe.Pointer(pythonbinding.Language())
	case "rust":
		ptr = unsafe.Pointer(rustbinding.Language())
	case "toml":
		ptr = unsafe.Pointer(tomlbinding.Language())
	case "tsx":
		ptr = unsafe.Pointer(typescriptbinding.LanguageTSX())
	case "typescript":
		ptr = unsafe.Pointer(typescriptbinding.LanguageTypescript())
	case "yaml":
		ptr = unsafe.Pointer(yamlbinding.Language())
	default:
		return nil
	}

	return sitter.NewLanguage(ptr)
}
