package treesitter

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

// ─── Wizard result ────────────────────────────────────────────────────────────

// WizardResult holds every choice the user made in the interactive wizard.
// It maps directly onto the existing Request types so callers can translate
// it without any extra logic.
type WizardResult struct {
	// Step 1 – pipeline mode
	Pipeline PipelineMode

	// Step 2 – target platforms (one or more)
	Platforms []Platform

	// Step 3 – artifacts & options
	GenerateManifest bool
	CopySCM          bool
	Compact          bool
	CopySource       bool
	CopyJS           bool
	IncludeWasm      bool
	Force            bool
	Prune            bool

	// Step 4 – selected grammar names
	Grammars []string
}

// PipelineMode describes which build phases to execute.
type PipelineMode string

const (
	PipelineBuildOnly           PipelineMode = "build"
	PipelineBuildCompile        PipelineMode = "build+compile"
	PipelineBuildCompilePublish PipelineMode = "build+compile+publish"
	PipelineCompileOnly         PipelineMode = "compile"
	PipelinePublishOnly         PipelineMode = "publish"
)

// Platform is a target OS/arch pair (or wasm).
type Platform struct {
	OS   string // "linux" | "macos" | "windows" | "wasm"
	Arch string // "amd64" | "arm64" | "" (wasm)
}

func (p Platform) String() string {
	if p.OS == "wasm" {
		return "wasm"
	}
	return p.OS + "/" + p.Arch
}

// ─── Public entry point ───────────────────────────────────────────────────────

// RunWizard runs the full interactive wizard and returns the user's choices.
// It returns (nil, false, nil) if the user aborted at the confirm step.
func RunWizard(languages []Language) (*WizardResult, bool, error) {
	header()

	result := &WizardResult{}

	if err := askPipeline(result); err != nil {
		return nil, false, err
	}
	if err := askPlatforms(result); err != nil {
		return nil, false, err
	}
	if err := askArtifacts(result); err != nil {
		return nil, false, err
	}
	if err := askGrammars(result, languages); err != nil {
		return nil, false, err
	}

	ok, err := confirmSummary(result)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		color.Yellow("\nAborted.")
		return nil, false, nil
	}

	return result, true, nil
}

// ─── Step helpers ─────────────────────────────────────────────────────────────

func askPipeline(r *WizardResult) error {
	bold := color.New(color.Bold)
	bold.Println("\nStep 1 of 4 — Pipeline")

	choices := []string{
		"Build + Compile + Publish  (recommended – full end-to-end pipeline)",
		"Build + Compile            (generate sources and shared libraries only)",
		"Build only                 (clone repo + run tree-sitter generate)",
		"Compile only               (re-compile already-built sources)",
		"Publish (move) only        (validate and publish already-compiled artifacts)",
	}

	// Override Filter so that pressing Space (or any whitespace) does not
	// enter filter-mode and hide all options. Only non-blank input filters.
	noSpaceFilter := func(filter string, value string, index int) bool {
		trimmed := strings.TrimSpace(filter)
		if trimmed == "" {
			return true
		}
		return strings.Contains(strings.ToLower(value), strings.ToLower(trimmed))
	}

	var answer string
	if err := survey.AskOne(&survey.Select{
		Message: "What do you want to do?",
		Options: choices,
		Filter:  noSpaceFilter,
	}, &answer, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	switch {
	case strings.HasPrefix(answer, "Build + Compile + Publish"):
		r.Pipeline = PipelineBuildCompilePublish
	case strings.HasPrefix(answer, "Build + Compile"):
		r.Pipeline = PipelineBuildCompile
	case strings.HasPrefix(answer, "Build only"):
		r.Pipeline = PipelineBuildOnly
	case strings.HasPrefix(answer, "Compile only"):
		r.Pipeline = PipelineCompileOnly
	default:
		r.Pipeline = PipelinePublishOnly
	}
	return nil
}

func askPlatforms(r *WizardResult) error {
	// Platform selection is only meaningful when we will actually compile or publish.
	if r.Pipeline == PipelineBuildOnly {
		// No binaries involved – default to host.
		r.Platforms = []Platform{{OS: "linux", Arch: "amd64"}}
		return nil
	}

	bold := color.New(color.Bold)
	bold.Println("\nStep 2 of 4 — Target Platforms")

	choices := []string{
		"Linux amd64     Native .so shared library (current host)",
		"macOS arm64     Apple Silicon .dylib (cross-compile via Zig/osxcross)",
		"macOS amd64     Intel Mac .dylib (cross-compile via Zig/osxcross)",
		"Windows amd64   .dll shared library (cross-compile via MinGW or Zig)",
		"WebAssembly     Universal .wasm (requires emcc, docker, or podman)",
	}

	var answers []string
	if err := survey.AskOne(&survey.MultiSelect{
		Message: "Select one or more target platforms:",
		Options: choices,
		Default: []string{choices[0]}, // pre-tick Linux amd64
	}, &answers, survey.WithValidator(survey.MinItems(1))); err != nil {
		return err
	}

	for _, a := range answers {
		switch {
		case strings.HasPrefix(a, "Linux amd64"):
			r.Platforms = append(r.Platforms, Platform{OS: "linux", Arch: "amd64"})
		case strings.HasPrefix(a, "macOS arm64"):
			r.Platforms = append(r.Platforms, Platform{OS: "macos", Arch: "arm64"})
		case strings.HasPrefix(a, "macOS amd64"):
			r.Platforms = append(r.Platforms, Platform{OS: "macos", Arch: "amd64"})
		case strings.HasPrefix(a, "Windows amd64"):
			r.Platforms = append(r.Platforms, Platform{OS: "windows", Arch: "amd64"})
		case strings.HasPrefix(a, "WebAssembly"):
			r.Platforms = append(r.Platforms, Platform{OS: "wasm", Arch: ""})
		}
	}
	return nil
}

func askArtifacts(r *WizardResult) error {
	// Artifact selection only matters when we publish (move phase).
	if r.Pipeline == PipelineBuildOnly || r.Pipeline == PipelineCompileOnly {
		// Sensible defaults – nothing to publish yet.
		r.GenerateManifest = false
		r.CopySCM = false
		r.Force = false
		r.Prune = false
		return nil
	}

	bold := color.New(color.Bold)
	bold.Println("\nStep 3 of 4 — Artifacts & Options")

	wasmSelected := false
	for _, p := range r.Platforms {
		if p.OS == "wasm" {
			wasmSelected = true
			break
		}
	}

	artifactChoices := []string{
		"manifest.json              Downstream metadata: versions, checksums, ABI",
		".scm query files           Syntax highlight & query patterns for editors",
		"compact-node-types.yaml   AI-friendly AST schema, 75% smaller than node-types.json",
		"C/C++ parser sources       Raw parser.c / scanner.c under src/ for distribution",
		"grammar.js                 Authoritative grammar specification file",
	}
	if wasmSelected {
		artifactChoices = append(artifactChoices, "WebAssembly artifact        .wasm file alongside native binaries")
	}

	defaults := []string{
		artifactChoices[0], // manifest.json
		artifactChoices[1], // .scm query files
	}

	var artifactAnswers []string
	if err := survey.AskOne(&survey.MultiSelect{
		Message: "Select artifacts to publish:",
		Options: artifactChoices,
		Default: defaults,
	}, &artifactAnswers); err != nil {
		return err
	}

	for _, a := range artifactAnswers {
		switch {
		case strings.HasPrefix(a, "manifest.json"):
			r.GenerateManifest = true
		case strings.HasPrefix(a, ".scm"):
			r.CopySCM = true
		case strings.HasPrefix(a, "compact-node-types.yaml"):
			r.Compact = true
		case strings.HasPrefix(a, "C/C++ parser sources"):
			r.CopySource = true
		case strings.HasPrefix(a, "grammar.js"):
			r.CopyJS = true
		case strings.HasPrefix(a, "WebAssembly artifact"):
			r.IncludeWasm = true
		}
	}

	optionChoices := []string{
		"Force overwrite    Re-publish even if artifacts already exist (--force)",
		"Prune build cache  Delete .git / node_modules after build to save disk (--prune)",
	}

	var optionAnswers []string
	if err := survey.AskOne(&survey.MultiSelect{
		Message: "Additional options:",
		Options: optionChoices,
	}, &optionAnswers); err != nil {
		return err
	}

	for _, a := range optionAnswers {
		switch {
		case strings.HasPrefix(a, "Force overwrite"):
			r.Force = true
		case strings.HasPrefix(a, "Prune build cache"):
			r.Prune = true
		}
	}

	return nil
}

func askGrammars(r *WizardResult, languages []Language) error {
	bold := color.New(color.Bold)
	bold.Println("\nStep 4 of 4 — Grammar Selection")

	options := make([]string, 0, len(languages)+1)
	options = append(options, "[ ALL ]  Build every configured grammar")
	for _, lang := range languages {
		options = append(options, fmt.Sprintf("%-20s  %s", lang.Name, lang.Version))
	}

	var answers []string
	if err := survey.AskOne(&survey.MultiSelect{
		Message: "Select grammars to process (Space to toggle, Enter to confirm):",
		Options: options,
	}, &answers, survey.WithValidator(survey.MinItems(1))); err != nil {
		return err
	}

	// If "ALL" was selected – return empty slice (callers treat "" as "all").
	for _, a := range answers {
		if strings.HasPrefix(a, "[ ALL ]") {
			r.Grammars = nil
			return nil
		}
	}

	for _, a := range answers {
		// Name is the first word before the padding spaces.
		name := strings.Fields(a)[0]
		r.Grammars = append(r.Grammars, name)
	}
	return nil
}

// ─── Confirm / summary ────────────────────────────────────────────────────────

func confirmSummary(r *WizardResult) (bool, error) {
	dim := color.New(color.Faint)
	cyan := color.New(color.FgCyan)
	bold := color.New(color.Bold)

	fmt.Println()
	bold.Println("┌─ Ready to build ──────────────────────────────────────┐")

	printRow := func(label, value string) {
		cyan.Printf("│  %-12s", label)
		fmt.Printf("  %-40s│\n", value)
	}

	printRow("Pipeline", string(r.Pipeline))

	platformStrs := make([]string, 0, len(r.Platforms))
	for _, p := range r.Platforms {
		platformStrs = append(platformStrs, p.String())
	}
	printRow("Targets", strings.Join(platformStrs, ", "))

	artifacts := collectArtifactLabels(r)
	if len(artifacts) == 0 {
		printRow("Artifacts", dim.Sprint("(none)"))
	} else {
		printRow("Artifacts", strings.Join(artifacts, ", "))
	}

	var opts []string
	if r.Force {
		opts = append(opts, "--force")
	}
	if r.Prune {
		opts = append(opts, "--prune")
	}
	if len(opts) == 0 {
		printRow("Options", dim.Sprint("(none)"))
	} else {
		printRow("Options", strings.Join(opts, ", "))
	}

	grammarLabel := "all"
	if len(r.Grammars) > 0 {
		grammarLabel = fmt.Sprintf("%s  (%d)", strings.Join(r.Grammars, ", "), len(r.Grammars))
	}
	// Truncate if too long for the box.
	if len(grammarLabel) > 40 {
		grammarLabel = grammarLabel[:37] + "..."
	}
	printRow("Grammars", grammarLabel)

	bold.Println("└───────────────────────────────────────────────────────┘")
	fmt.Println()

	var proceed bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Proceed?",
		Default: true,
	}, &proceed); err != nil {
		return false, err
	}
	return proceed, nil
}

func collectArtifactLabels(r *WizardResult) []string {
	var labels []string
	if r.GenerateManifest {
		labels = append(labels, "manifest.json")
	}
	if r.CopySCM {
		labels = append(labels, ".scm")
	}
	if r.Compact {
		labels = append(labels, "compact-yaml")
	}
	if r.CopySource {
		labels = append(labels, "c-source")
	}
	if r.CopyJS {
		labels = append(labels, "grammar.js")
	}
	if r.IncludeWasm {
		labels = append(labels, "wasm")
	}
	return labels
}

// ─── Header ───────────────────────────────────────────────────────────────────

func header() {
	cyan := color.New(color.FgCyan, color.Bold)
	dim := color.New(color.Faint)

	fmt.Println()
	cyan.Println("  ts-build  —  interactive grammar builder")
	dim.Println("  Use arrow keys to navigate, Space to toggle, Enter to confirm.")
	fmt.Println()
}
