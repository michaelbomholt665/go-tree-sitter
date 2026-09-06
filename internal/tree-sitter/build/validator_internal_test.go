package build

import (
	"context"
	"debug/macho"
	"debug/pe"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

func TestValidateConstructorSymbolsRejectsMissingOrWrongConstructor(t *testing.T) {
	t.Parallel()
	for _, symbols := range [][]string{{"tree_sitter_javascript"}, {"ordinary_symbol"}, {"tree_sitter_python", "tree_sitter_javascript"}} {
		if err := validateConstructorSymbols("grammar.so", "tree_sitter_python", symbols); err == nil {
			t.Fatalf("expected constructors %v to fail", symbols)
		}
	}
}

func TestValidateRuntimeABIRejectsOutsideSelectedRuntimeRange(t *testing.T) {
	t.Parallel()
	if err := validateRuntimeABI(15, 13, 14); err == nil || !strings.Contains(err.Error(), "outside runtime range 13-14") {
		t.Fatalf("unexpected result: %v", err)
	}
}

func TestQueryFromDifferentGrammarRevisionIsRejectedWithLocation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	nodeTypes := filepath.Join(dir, "node-types.json")
	queryPath := filepath.Join(dir, "highlights.scm")
	writeTestFile(t, nodeTypes, "[]\n")
	writeTestFile(t, queryPath, "(node_from_another_revision) @capture\n")
	language := sitter.NewLanguage(python.Language())
	err := validateLanguageArtifacts(language, ValidationItem{
		Language: _jsii.Language{Name: "python"}, NodeTypesPath: nodeTypes,
		QueryPaths: []string{queryPath}, Sample: "x = 1\n",
	})
	if err == nil || !strings.Contains(err.Error(), queryPath) || !strings.Contains(err.Error(), "row 1 column") {
		t.Fatalf("expected located query error, got %v", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePEBinaryValid(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dllPath := filepath.Join(dir, "python-windows-amd64.dll")
	dllBytes := createMockPEDLL(t, pe.IMAGE_FILE_MACHINE_AMD64, "tree_sitter_python")
	if err := os.WriteFile(dllPath, dllBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := validateBinaryFormat(dllPath, "windows", "amd64", "tree_sitter_python"); err != nil {
		t.Fatalf("expected PE validation to pass, got: %v", err)
	}
}

func TestValidatePEBinaryWrongMachine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dllPath := filepath.Join(dir, "python-windows-amd64.dll")
	dllBytes := createMockPEDLL(t, 0x14c, "tree_sitter_python")
	if err := os.WriteFile(dllPath, dllBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateBinaryFormat(dllPath, "windows", "amd64", "tree_sitter_python")
	if err == nil || !strings.Contains(err.Error(), "expected amd64") {
		t.Fatalf("expected machine mismatch error, got: %v", err)
	}
}

func TestValidatePEBinaryMissingExports(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dllPath := filepath.Join(dir, "python-windows-amd64.dll")
	dllBytes := createMockPEDLL(t, pe.IMAGE_FILE_MACHINE_AMD64, "")
	if err := os.WriteFile(dllPath, dllBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateBinaryFormat(dllPath, "windows", "amd64", "tree_sitter_python")
	if err == nil || !strings.Contains(err.Error(), "PE export directory") {
		t.Fatalf("expected missing exports error, got: %v", err)
	}
}

func TestValidatePEBinaryWrongConstructor(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dllPath := filepath.Join(dir, "python-windows-amd64.dll")
	dllBytes := createMockPEDLL(t, pe.IMAGE_FILE_MACHINE_AMD64, "tree_sitter_javascript")
	if err := os.WriteFile(dllPath, dllBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateBinaryFormat(dllPath, "windows", "amd64", "tree_sitter_python")
	if err == nil || !strings.Contains(err.Error(), "expected only \"tree_sitter_python\"") {
		t.Fatalf("expected constructor mismatch error, got: %v", err)
	}
}

func TestValidateMachOBinaryValidAmd64(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dylibPath := filepath.Join(dir, "python-macos-amd64.dylib")
	dylibBytes := createMockMachODylib(t, macho.CpuAmd64, macho.TypeDylib, "_tree_sitter_python")
	if err := os.WriteFile(dylibPath, dylibBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := validateBinaryFormat(dylibPath, "macos", "amd64", "tree_sitter_python"); err != nil {
		t.Fatalf("expected Mach-O amd64 validation to pass, got: %v", err)
	}
}

func TestValidateMachOBinaryValidArm64(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dylibPath := filepath.Join(dir, "python-macos-arm64.dylib")
	dylibBytes := createMockMachODylib(t, macho.CpuArm64, macho.TypeDylib, "tree_sitter_python")
	if err := os.WriteFile(dylibPath, dylibBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := validateBinaryFormat(dylibPath, "macos", "arm64", "tree_sitter_python"); err != nil {
		t.Fatalf("expected Mach-O arm64 validation to pass, got: %v", err)
	}
}

func TestValidateMachOBinaryWrongCPU(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dylibPath := filepath.Join(dir, "python-macos-arm64.dylib")
	dylibBytes := createMockMachODylib(t, macho.CpuAmd64, macho.TypeDylib, "_tree_sitter_python")
	if err := os.WriteFile(dylibPath, dylibBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateBinaryFormat(dylibPath, "macos", "arm64", "tree_sitter_python")
	if err == nil || !strings.Contains(err.Error(), "expected CpuArm64") {
		t.Fatalf("expected CPU mismatch error, got: %v", err)
	}
}

func TestValidateMachOBinaryWrongType(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dylibPath := filepath.Join(dir, "python-macos-amd64.dylib")
	dylibBytes := createMockMachODylib(t, macho.CpuAmd64, macho.TypeObj, "_tree_sitter_python")
	if err := os.WriteFile(dylibPath, dylibBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateBinaryFormat(dylibPath, "macos", "amd64", "tree_sitter_python")
	if err == nil || !strings.Contains(err.Error(), "expected TypeDylib") {
		t.Fatalf("expected file type error, got: %v", err)
	}
}

func TestValidateMachOBinaryWrongConstructor(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dylibPath := filepath.Join(dir, "python-macos-amd64.dylib")
	dylibBytes := createMockMachODylib(t, macho.CpuAmd64, macho.TypeDylib, "_tree_sitter_ruby")
	if err := os.WriteFile(dylibPath, dylibBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateBinaryFormat(dylibPath, "macos", "amd64", "tree_sitter_python")
	if err == nil || !strings.Contains(err.Error(), "expected only \"tree_sitter_python\"") {
		t.Fatalf("expected constructor mismatch error, got: %v", err)
	}
}

func TestDualModeValidationCrossPlatform(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	winPath := filepath.Join(dir, "python-windows-amd64.dll")
	winBytes := createMockPEDLL(t, pe.IMAGE_FILE_MACHINE_AMD64, "tree_sitter_python")
	if err := os.WriteFile(winPath, winBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	macPath := filepath.Join(dir, "python-macos-arm64.dylib")
	macBytes := createMockMachODylib(t, macho.CpuArm64, macho.TypeDylib, "_tree_sitter_python")
	if err := os.WriteFile(macPath, macBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	nodeTypes := filepath.Join(dir, "node-types.json")
	if err := os.WriteFile(nodeTypes, []byte("[]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	validator := NewNativeValidator()
	items := []ValidationItem{
		{
			Language:      _jsii.Language{Name: "python", Grammar: "python", Constructor: "tree_sitter_python"},
			BinaryPath:    winPath,
			Platform:      "windows",
			Arch:          "amd64",
			NodeTypesPath: nodeTypes,
			GenerateABI:   15,
		},
		{
			Language:      _jsii.Language{Name: "python", Grammar: "python", Constructor: "tree_sitter_python"},
			BinaryPath:    macPath,
			Platform:      "macos",
			Arch:          "arm64",
			NodeTypesPath: nodeTypes,
			GenerateABI:   15,
		},
	}

	abis, err := validator.Validate(context.Background(), items)
	if err != nil {
		t.Fatalf("expected cross-platform validation to succeed, got: %v", err)
	}

	if abis[winPath] != 15 {
		t.Errorf("expected winPath ABI 15, got %d", abis[winPath])
	}
	if abis[macPath] != 15 {
		t.Errorf("expected macPath ABI 15, got %d", abis[macPath])
	}
}

func createMockPEDLL(t *testing.T, machine uint16, constructor string) []byte {
	t.Helper()
	dosHeader := make([]byte, 64)
	copy(dosHeader[0:2], "MZ")
	dosHeader[60] = 64

	peSig := []byte("PE\x00\x00")

	coffHeader := make([]byte, 20)
	coffHeader[0] = byte(machine)
	coffHeader[1] = byte(machine >> 8)
	coffHeader[2] = 1     // NumberOfSections
	coffHeader[16] = 240  // SizeOfOptionalHeader
	coffHeader[18] = 0x20 // Characteristics
	coffHeader[19] = 0x20

	optHeader := make([]byte, 240)
	optHeader[0] = 0x0b // PE32+ (0x020b)
	optHeader[1] = 0x02
	optHeader[33] = 0x10 // SectionAlignment (0x1000)
	optHeader[37] = 0x02 // FileAlignment (0x200)
	optHeader[57] = 0x20 // SizeOfImage (0x2000)
	optHeader[61] = 0x02 // SizeOfHeaders (0x200)
	optHeader[108] = 16  // NumberOfRvaAndSizes
	if constructor != "" {
		optHeader[113] = 0x10 // Export Table RVA (0x1000)
		optHeader[116] = 0x00
		optHeader[117] = 0x01 // Export Table Size (0x100)
	}

	secHeader := make([]byte, 40)
	copy(secHeader[0:8], ".edata\x00\x00")
	secHeader[9] = 0x01  // VirtualSize (0x100)
	secHeader[13] = 0x10 // VirtualAddress (0x1000)
	secHeader[17] = 0x02 // SizeOfRawData (0x200)
	secHeader[21] = 0x02 // PointerToRawData (0x200)
	secHeader[36] = 0x40 // Characteristics
	secHeader[39] = 0x40

	headers := append(dosHeader, peSig...)
	headers = append(headers, coffHeader...)
	headers = append(headers, optHeader...)
	headers = append(headers, secHeader...)
	if len(headers) < 512 {
		headers = append(headers, make([]byte, 512-len(headers))...)
	}

	secData := make([]byte, 512)
	if constructor != "" {
		// IMAGE_EXPORT_DIRECTORY (40 bytes)
		secData[12] = 0x50 // Name RVA (0x1050)
		secData[13] = 0x10
		secData[16] = 1    // Base = 1
		secData[20] = 1    // NumberOfFunctions = 1
		secData[24] = 1    // NumberOfNames = 1
		secData[28] = 0x28 // AddressOfFunctions (0x1028)
		secData[29] = 0x10
		secData[32] = 0x2c // AddressOfNames (0x102C)
		secData[33] = 0x10
		secData[36] = 0x30 // AddressOfNameOrdinals (0x1030)
		secData[37] = 0x10

		secData[40] = 0x00 // Function RVA (0x1000)
		secData[41] = 0x10
		secData[44] = 0x60 // Name RVA (0x1060)
		secData[45] = 0x10
		secData[48] = 0 // Ordinal 0

		copy(secData[80:], "grammar.dll\x00")
		copy(secData[96:], constructor+"\x00")
	}

	return append(headers, secData...)
}

func createMockMachODylib(t *testing.T, cpu macho.Cpu, fileType macho.Type, constructor string) []byte {
	t.Helper()
	header := make([]byte, 32)
	header[0] = 0xcf // MH_MAGIC_64 (0xfeedfacf)
	header[1] = 0xfa
	header[2] = 0xed
	header[3] = 0xfe
	header[4] = byte(cpu)
	header[5] = byte(cpu >> 8)
	header[6] = byte(cpu >> 16)
	header[7] = byte(cpu >> 24)
	header[8] = 3
	header[12] = byte(fileType)
	header[16] = 1  // ncmds
	header[20] = 24 // sizeofcmds

	symName := []byte("\x00" + constructor + "\x00")
	strSize := uint32(len(symName))
	symOff := uint32(32 + 24)
	strOff := symOff + 16

	lc := make([]byte, 24)
	lc[0] = 2 // LC_SYMTAB
	lc[4] = 24
	lc[8] = byte(symOff)
	lc[9] = byte(symOff >> 8)
	lc[12] = 1 // nsyms
	lc[16] = byte(strOff)
	lc[17] = byte(strOff >> 8)
	lc[20] = byte(strSize)
	lc[21] = byte(strSize >> 8)

	nlist := make([]byte, 16)
	nlist[0] = 1    // n_strx
	nlist[4] = 0x0f // N_EXT | N_SECT
	nlist[5] = 1

	data := append(header, lc...)
	data = append(data, nlist...)
	data = append(data, symName...)
	return data
}
