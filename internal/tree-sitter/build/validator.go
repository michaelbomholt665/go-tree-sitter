package build

import (
	"context"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type ValidationItem struct {
	Language      _jsii.Language
	BinaryPath    string
	Platform      string
	Arch          string
	NodeTypesPath string
	QueryPaths    []string
	Sample        string
	GenerateABI   uint32
	Provenance    *BinaryProvenance
}

type ArtifactValidator interface {
	Validate(context.Context, []ValidationItem) (map[string]uint32, error)
}

type NativeValidator struct{}

func NewNativeValidator() *NativeValidator { return &NativeValidator{} }

func (v *NativeValidator) Validate(ctx context.Context, items []ValidationItem) (map[string]uint32, error) {
	ordered := append([]ValidationItem(nil), items...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].BinaryPath < ordered[j].BinaryPath })
	versions := make(map[string]uint32, len(ordered))
	constructorsByTarget := make(map[string]string, len(ordered))
	constructorToLang := make(map[string]string, len(ordered))
	var libraries []*nativeLibrary
	var validationErrors []error
	defer func() {
		for i := len(libraries) - 1; i >= 0; i-- {
			_ = libraries[i].Close()
		}
	}()

	for _, item := range ordered {
		if err := ctx.Err(); err != nil {
			validationErrors = append(validationErrors, err)
			break
		}

		targetKey := item.Platform + "/" + item.Arch + ":" + item.Language.Constructor
		if previous, exists := constructorsByTarget[targetKey]; exists {
			validationErrors = append(validationErrors, fmt.Errorf("duplicate constructor %q in %q and %q", item.Language.Constructor, previous, item.BinaryPath))
			continue
		}
		constructorsByTarget[targetKey] = item.BinaryPath

		if prevLang, exists := constructorToLang[item.Language.Constructor]; exists && prevLang != item.Language.Name {
			validationErrors = append(validationErrors, fmt.Errorf("constructor %q used by both %q and %q", item.Language.Constructor, prevLang, item.Language.Name))
			continue
		}
		constructorToLang[item.Language.Constructor] = item.Language.Name

		if isHostTarget(item.Platform, item.Arch) {
			// Host validation path (dynamic)
			if err := validateBinaryFormat(item.BinaryPath, item.Platform, item.Arch, item.Language.Constructor); err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("%s: %w", item.Language.Name, err))
				continue
			}
			library, pointer, err := openNativeLanguage(item.BinaryPath, item.Language.Constructor)
			if err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("%s: %w", item.Language.Name, err))
				continue
			}
			libraries = append(libraries, library)
			language := sitter.NewLanguage(pointer)
			abi := language.AbiVersion()
			versions[item.BinaryPath] = abi
			if err := validateRuntimeABI(abi, sitter.MIN_COMPATIBLE_LANGUAGE_VERSION, sitter.LANGUAGE_VERSION); err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("%s: %w", item.Language.Name, err))
				continue
			}
			if err := validateLanguageArtifacts(language, item); err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("%s: %w", item.Language.Name, err))
			}
		} else {
			// Cross-platform validation path (static)
			if err := validateBinaryFormat(item.BinaryPath, item.Platform, item.Arch, item.Language.Constructor); err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("%s: %w", item.Language.Name, err))
				continue
			}
			abi, err := resolveCrossPlatformABI(item)
			if err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("%s: %w", item.Language.Name, err))
				continue
			}
			versions[item.BinaryPath] = abi
			if err := validateRuntimeABI(abi, sitter.MIN_COMPATIBLE_LANGUAGE_VERSION, sitter.LANGUAGE_VERSION); err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("%s: %w", item.Language.Name, err))
				continue
			}
			if err := validateStaticLanguageArtifacts(item); err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("%s: %w", item.Language.Name, err))
			}
		}
	}
	return versions, errors.Join(validationErrors...)
}

func isHostTarget(platform, arch string) bool {
	expectedPlatform := platformName(runtime.GOOS)
	return platform == expectedPlatform && arch == runtime.GOARCH
}

func resolveCrossPlatformABI(item ValidationItem) (uint32, error) {
	if item.GenerateABI > 0 {
		return item.GenerateABI, nil
	}
	if item.Provenance != nil && item.Provenance.GenerateABI != nil && *item.Provenance.GenerateABI > 0 {
		return uint32(*item.Provenance.GenerateABI), nil
	}

	// Try reading binary provenance file alongside the binary
	var bp BinaryProvenance
	if err := readJSON(binaryProvenancePath(item.BinaryPath), &bp); err == nil {
		if bp.GenerateABI != nil && *bp.GenerateABI > 0 {
			return uint32(*bp.GenerateABI), nil
		}
	}

	// Try reading source provenance file from binary dir or ancestor dirs
	dir := filepath.Dir(item.BinaryPath)
	for i := 0; i < 4; i++ {
		sourceProv := filepath.Join(dir, sourceProvenanceFilename)
		var sp SourceProvenance
		if err := readJSON(sourceProv, &sp); err == nil {
			if sp.GenerateABI != nil && *sp.GenerateABI > 0 {
				return uint32(*sp.GenerateABI), nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return 0, fmt.Errorf("cannot determine parser ABI for cross-platform binary %q", item.BinaryPath)
}

func validateRuntimeABI(abi, minimum, maximum uint32) error {
	if abi < minimum || abi > maximum {
		return fmt.Errorf("parser ABI %d is outside runtime range %d-%d", abi, minimum, maximum)
	}
	return nil
}

func validateLanguageArtifacts(language *sitter.Language, item ValidationItem) error {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(language); err != nil {
		return fmt.Errorf("assign final language to runtime parser: %w", err)
	}
	tree := parser.Parse([]byte(item.Sample), nil)
	if tree == nil {
		return errors.New("parse minimal sample: runtime returned a null tree")
	}
	defer tree.Close()
	if tree.RootNode().HasError() {
		return errors.New("parse minimal sample: syntax tree contains errors")
	}

	if item.NodeTypesPath != "" {
		payload, err := os.ReadFile(item.NodeTypesPath)
		if err != nil {
			return fmt.Errorf("read node-types.json: %w", err)
		}
		var nodeTypes []json.RawMessage
		if err := json.Unmarshal(payload, &nodeTypes); err != nil {
			return fmt.Errorf("parse node-types.json: %w", err)
		}
	}
	for _, queryPath := range item.QueryPaths {
		querySource, err := os.ReadFile(queryPath)
		if err != nil {
			return fmt.Errorf("read query %q: %w", queryPath, err)
		}
		query, queryErr := sitter.NewQuery(language, string(querySource))
		if queryErr != nil {
			return fmt.Errorf("compile query %q at row %d column %d: %w", queryPath, queryErr.Row+1, queryErr.Column+1, queryErr)
		}
		query.Close()
	}
	return nil
}

func validateStaticLanguageArtifacts(item ValidationItem) error {
	if item.NodeTypesPath != "" {
		payload, err := os.ReadFile(item.NodeTypesPath)
		if err != nil {
			return fmt.Errorf("read node-types.json: %w", err)
		}
		var nodeTypes []json.RawMessage
		if err := json.Unmarshal(payload, &nodeTypes); err != nil {
			return fmt.Errorf("parse node-types.json: %w", err)
		}
	}
	for _, queryPath := range item.QueryPaths {
		if _, err := os.ReadFile(queryPath); err != nil {
			return fmt.Errorf("read query %q: %w", queryPath, err)
		}
	}
	return nil
}

func validateBinaryFormat(path, platform, arch, constructor string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect binary %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("binary %q is not a regular file", path)
	}
	switch platform {
	case "linux":
		return validateELFBinary(path, arch, constructor)
	case "macos":
		return validateMachOBinary(path, arch, constructor)
	case "windows":
		return validatePEBinary(path, arch, constructor)
	default:
		return fmt.Errorf("unsupported binary platform %q", platform)
	}
}

func validateELFBinary(path, arch, constructor string) error {
	file, err := elf.Open(path)
	if err != nil {
		return fmt.Errorf("open ELF binary %q: %w", path, err)
	}
	defer file.Close()

	switch arch {
	case "amd64":
		if file.Machine != elf.EM_X86_64 {
			return fmt.Errorf("binary %q has ELF machine %s, expected x86-64", path, file.Machine)
		}
	case "arm64":
		if file.Machine != elf.EM_AARCH64 {
			return fmt.Errorf("binary %q has ELF machine %s, expected AArch64", path, file.Machine)
		}
	default:
		return fmt.Errorf("unsupported ELF architecture %q", arch)
	}

	symbols, err := file.DynamicSymbols()
	if err != nil {
		return fmt.Errorf("read dynamic symbols from %q: %w", path, err)
	}
	return validateConstructorSymbols(path, constructor, elfSymbolNames(symbols))
}

func validateMachOBinary(path, arch, constructor string) error {
	file, err := macho.Open(path)
	if err != nil {
		return fmt.Errorf("open Mach-O binary %q: %w", path, err)
	}
	defer file.Close()

	switch arch {
	case "amd64":
		if file.Cpu != macho.CpuAmd64 {
			return fmt.Errorf("binary %q has Mach-O cpu %s, expected CpuAmd64", path, file.Cpu)
		}
	case "arm64":
		if file.Cpu != macho.CpuArm64 {
			return fmt.Errorf("binary %q has Mach-O cpu %s, expected CpuArm64", path, file.Cpu)
		}
	default:
		return fmt.Errorf("unsupported Mach-O architecture %q", arch)
	}

	if file.Type != macho.TypeDylib {
		return fmt.Errorf("binary %q has Mach-O type %s, expected TypeDylib", path, file.Type)
	}

	if file.Symtab == nil || len(file.Symtab.Syms) == 0 {
		return fmt.Errorf("binary %q has no Mach-O symbol table", path)
	}

	var names []string
	for _, symbol := range file.Symtab.Syms {
		names = append(names, symbol.Name)
	}
	return validateConstructorSymbols(path, constructor, names)
}

func validatePEBinary(path, arch, constructor string) error {
	file, err := pe.Open(path)
	if err != nil {
		return fmt.Errorf("open PE binary %q: %w", path, err)
	}
	defer file.Close()

	switch arch {
	case "amd64":
		if file.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
			return fmt.Errorf("binary %q has PE machine %#x, expected amd64 (%#x)", path, file.Machine, pe.IMAGE_FILE_MACHINE_AMD64)
		}
	case "arm64":
		if file.Machine != pe.IMAGE_FILE_MACHINE_ARM64 {
			return fmt.Errorf("binary %q has PE machine %#x, expected arm64 (%#x)", path, file.Machine, pe.IMAGE_FILE_MACHINE_ARM64)
		}
	default:
		return fmt.Errorf("unsupported PE architecture %q", arch)
	}

	if len(file.Sections) == 0 {
		return fmt.Errorf("binary %q has no PE sections", path)
	}

	exportNames, err := readPEExportNames(file)
	if err != nil {
		return fmt.Errorf("binary %q: %w", path, err)
	}

	return validateConstructorSymbols(path, constructor, exportNames)
}

func readPEExportNames(file *pe.File) ([]string, error) {
	var exportDir pe.DataDirectory
	switch opt := file.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		if len(opt.DataDirectory) <= pe.IMAGE_DIRECTORY_ENTRY_EXPORT {
			return nil, errors.New("missing PE export directory")
		}
		exportDir = opt.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT]
	case *pe.OptionalHeader32:
		if len(opt.DataDirectory) <= pe.IMAGE_DIRECTORY_ENTRY_EXPORT {
			return nil, errors.New("missing PE export directory")
		}
		exportDir = opt.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT]
	default:
		return nil, fmt.Errorf("unsupported PE optional header %T", file.OptionalHeader)
	}

	if exportDir.VirtualAddress == 0 || exportDir.Size == 0 {
		return nil, errors.New("PE export directory table is empty or missing")
	}

	headerData, err := readPERVA(file, exportDir.VirtualAddress, 40)
	if err != nil {
		return nil, fmt.Errorf("read PE export directory: %w", err)
	}

	numberOfNames := binary.LittleEndian.Uint32(headerData[24:28])
	addressOfNames := binary.LittleEndian.Uint32(headerData[32:36])
	if numberOfNames == 0 || addressOfNames == 0 {
		return nil, errors.New("PE export directory has no exported names")
	}

	namesData, err := readPERVA(file, addressOfNames, numberOfNames*4)
	if err != nil {
		return nil, fmt.Errorf("read PE export names table: %w", err)
	}

	names := make([]string, 0, numberOfNames)
	for i := uint32(0); i < numberOfNames; i++ {
		nameRVA := binary.LittleEndian.Uint32(namesData[i*4 : (i+1)*4])
		name, err := readPEStringAtRVA(file, nameRVA)
		if err != nil {
			return nil, fmt.Errorf("read PE export name at index %d: %w", i, err)
		}
		names = append(names, name)
	}
	return names, nil
}

func readPERVA(file *pe.File, rva, size uint32) ([]byte, error) {
	for _, sec := range file.Sections {
		secSize := sec.VirtualSize
		if secSize == 0 {
			secSize = sec.Size
		}
		if rva >= sec.VirtualAddress && rva < sec.VirtualAddress+secSize {
			offset := rva - sec.VirtualAddress
			data, err := sec.Data()
			if err != nil {
				return nil, err
			}
			if uint64(offset)+uint64(size) > uint64(len(data)) {
				return nil, fmt.Errorf("RVA %#x length %d exceeds section raw data size %d", rva, size, len(data))
			}
			return data[offset : offset+size], nil
		}
	}
	return nil, fmt.Errorf("RVA %#x not found in any PE section", rva)
}

func readPEStringAtRVA(file *pe.File, rva uint32) (string, error) {
	for _, sec := range file.Sections {
		secSize := sec.VirtualSize
		if secSize == 0 {
			secSize = sec.Size
		}
		if rva >= sec.VirtualAddress && rva < sec.VirtualAddress+secSize {
			offset := rva - sec.VirtualAddress
			data, err := sec.Data()
			if err != nil {
				return "", err
			}
			if offset >= uint32(len(data)) {
				return "", fmt.Errorf("RVA %#x offset exceeds section raw data size %d", rva, len(data))
			}
			end := offset
			for end < uint32(len(data)) && data[end] != 0 {
				end++
			}
			return string(data[offset:end]), nil
		}
	}
	return "", fmt.Errorf("RVA %#x not found in any PE section", rva)
}

func elfSymbolNames(symbols []elf.Symbol) []string {
	names := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		names = append(names, symbol.Name)
	}
	return names
}

func validateConstructorSymbols(path, expected string, symbols []string) error {
	var constructors []string
	for _, symbol := range symbols {
		trimmed := strings.TrimPrefix(symbol, "_")
		if strings.HasPrefix(trimmed, "tree_sitter_") && isCIdentifier(trimmed) && !strings.Contains(trimmed, "_external_scanner_") {
			constructors = append(constructors, trimmed)
		}
	}
	sort.Strings(constructors)
	constructors = slices.Compact(constructors)
	if len(constructors) != 1 || constructors[0] != expected {
		return fmt.Errorf("binary %q exports grammar constructors %v, expected only %q", path, constructors, expected)
	}
	return nil
}

func isCIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func collectQueryPaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".scm" {
			paths = append(paths, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	sort.Strings(paths)
	return paths, err
}
