package treesitter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

type dryRunKey struct{}
type dryRunState struct {
	out        io.Writer
	configPath string
	config     []byte
}

func isDryRun(cmd *cobra.Command) bool {
	value, _ := cmd.Flags().GetBool("dry-run")
	return value
}

func dryRunFrom(ctx context.Context) *dryRunState {
	state, _ := ctx.Value(dryRunKey{}).(*dryRunState)
	return state
}

// Preview adapters replace runtime operations, not flag parsing or request construction.
func (a *App) prepareDryRun(cmd *cobra.Command, path *string) error {
	state := &dryRunState{out: a.stdout}
	cmd.SetContext(context.WithValue(cmd.Context(), dryRunKey{}, state))
	switch cmd.Name() {
	case "help", "version", "completion", "bash", "zsh", "fish", "powershell", "regen-manifests", "__complete", "__completeNoDesc":
		return nil
	}
	if a.discoverConfig && !cmd.Flags().Changed("config") {
		selected, err := resolveConfigPath(false)
		if err != nil {
			return err
		}
		*path = selected
		if _, err := os.Stat(selected); errors.Is(err, os.ErrNotExist) {
			state.config, err = defaultUserConfig()
			if err != nil {
				return err
			}
			state.configPath = selected
			fmt.Fprintln(a.stderr, "Dry run: would initialize configuration at", selected, "(using defaults in memory; no file created)")
		} else if err != nil {
			return err
		}
	}
	load := a.load
	a.load = func(path string) (*Config, error) {
		if state.config != nil && path == state.configPath {
			cfg, err := decodeEditedConfig(path, state.config)
			if err != nil {
				return nil, err
			}
			return anchorConfigPaths(cfg, path)
		}
		return load(path)
	}
	preview := dryRunOperations{out: a.stdout}
	a.builder, a.compiler, a.mover = preview, preview, preview
	a.cleaner, a.compactor = preview, preview
	// Tool availability and versions are runtime checks, not simulated successes.
	a.lookup = dryRunLookup{}
	return nil
}

type dryRunLookup struct{}

func (dryRunLookup) LookPath(name string) (string, error) { return name, nil }

func readCommandConfig(cmd *cobra.Command, path string) ([]byte, error) {
	if state := dryRunFrom(cmd.Context()); state != nil && state.config != nil && state.configPath == path {
		return state.config, nil
	}
	return os.ReadFile(path)
}

func readCommandConfigDocument(cmd *cobra.Command, path string) (*configDocument, error) {
	content, err := readCommandConfig(cmd, path)
	if err != nil {
		return nil, err
	}
	return parseConfigDocument(path, content)
}

type dryRunOperations struct{ out io.Writer }

func (p dryRunOperations) plan(ctx context.Context, cfg *Config, name, language string, request any, actions ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := cfg.ValidateBuildPaths(); err != nil {
		return err
	}
	languages, err := cfg.ResolveLanguages(language)
	if err != nil && name == "compact" {
		if _, statErr := os.Stat(language); statErr == nil {
			languages, err = nil, nil // Direct file/directory operand, not the entire catalog.
		}
	}
	if err != nil {
		return err
	}
	type grammar struct {
		Name, Repository, Revision string
		GenerateABI                int
	}
	grammars := make([]grammar, 0, len(languages))
	for _, lang := range languages {
		abi := cfg.GenerateABI
		if lang.GenerateABI != nil {
			abi = *lang.GenerateABI
		}
		grammars = append(grammars, grammar{lang.Name, lang.Repository, lang.Revision, abi})
	}
	policy := cfg.EffectiveRuntime()
	var options any
	switch name {
	case "build":
		options = map[string]any{"node": policy.Node, "generation": policy.Generation}
	case "compile":
		options = map[string]any{"targets": cfg.Targets, "compiler": policy.Compiler, "wasm": policy.WASM, "build_flags": cfg.BuildFlags}
	case "move":
		options = map[string]any{"cleanup": policy.Cleanup, "queries": policy.Queries}
	case "clean":
		options = policy.Cleanup
	}
	encoder := json.NewEncoder(p.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]any{
		"dry_run": true, "command": name, "actions": actions, "request": request,
		"policy":   options,
		"grammars": grammars, "build_dir": cfg.BuildDir, "output": cfg.Output,
		"tools":       cfg.EffectiveRuntime().Tools,
		"not_checked": []string{"tool availability and versions", "remote repositories", "generated sources and binary validity", "compact schema equivalence", "runtime filesystem permissions"},
	})
}

func (p dryRunOperations) Build(ctx context.Context, cfg *Config, req BuildRequest) error {
	actions := []string{"Check out pinned grammar sources in the build workspace", "Install Node dependencies according to runtime.node", "Generate parser sources using the configured ABI and fallback policy"}
	if req.Force {
		actions = append([]string{"Remove selected grammar build roots before checkout"}, actions...)
	}
	if req.Prune || cfg.Prune {
		actions = append(actions, "Prune workspace using configured preservation and pruning rules")
	}
	return p.plan(ctx, cfg, "build", req.Language, req, actions...)
}

func (p dryRunOperations) Compile(ctx context.Context, cfg *Config, req CompileRequest) error {
	if !req.BuildWasm {
		osName, arch := req.OS, req.Arch
		if osName == "" && arch == "" && cfg.OSTarget != "" {
			target := cfg.Targets[cfg.OSTarget]
			osName, arch = target.OS, target.Arch
		}
		if osName == "" {
			osName = runtime.GOOS
		}
		if arch == "" {
			arch = runtime.GOARCH
		}
		switch strings.ToLower(strings.TrimSpace(osName)) {
		case "darwin", "macos", "mac":
			osName = "macos"
		case "windows", "win":
			osName = "windows"
		default:
			osName = strings.ToLower(strings.TrimSpace(osName))
		}
		arch = strings.ToLower(strings.TrimSpace(arch))
		if !slices.Contains(cfg.EffectiveRuntime().SupportedTargets[osName], arch) {
			return fmt.Errorf("unsupported target %s/%s", osName, arch)
		}
		req.OS, req.Arch = osName, arch
	}
	return p.plan(ctx, cfg, "compile", req.Language, req, "Select configured native compiler or WASM backend and permitted fallbacks", "Compile workspace sources and record binary provenance")
}

func (p dryRunOperations) Move(ctx context.Context, cfg *Config, req MoveRequest) error {
	if req.GenerateManifest && !cfg.Output.GenerateManifest {
		return errors.New("publication requires a validated manifest; generate_manifest=false is no longer supported")
	}
	if req.CleanupAction != "" && req.CleanupAction != "keep" && req.CleanupAction != "prune" && req.CleanupAction != "remove" {
		return fmt.Errorf("invalid cleanup action %q", req.CleanupAction)
	}
	actions := []string{"Stage selected compiled artifacts, node types, and optional source/query files", "Validate binaries and samples before publication", "Generate requested manifest and compact schema", "Atomically publish validated artifacts to output.grammar_base"}
	if req.Clean {
		actions = append(actions, "After successful publication, apply cleanup policy: "+req.CleanupAction)
	}
	return p.plan(ctx, cfg, "move", req.Language, req, actions...)
}

func (p dryRunOperations) Clean(ctx context.Context, cfg *Config, req CleanRequest) error {
	action := "Remove selected grammar workspace roots"
	if req.Prune {
		action = "Prune selected grammar workspace roots using configured rules"
	}
	return p.plan(ctx, cfg, "clean", req.Language, req, action)
}

func (p dryRunOperations) Compact(ctx context.Context, cfg *Config, req CompactRequest) error {
	action := "Generate compact node types and validate lossless equivalence"
	if req.Check {
		action = "Check existing compact node types for lossless equivalence"
	}
	return p.plan(ctx, cfg, "compact", req.Language, req, action)
}

func previewExport(ctx context.Context, source, destination string, names []string, force bool) error {
	for _, name := range names {
		if err := filepath.WalkDir(filepath.Join(source, name), func(_ string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 || (!entry.IsDir() && !entry.Type().IsRegular()) {
				return fmt.Errorf("export rejects symlinks and nonregular files: %s", entry.Name())
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return json.NewEncoder(dryRunFrom(ctx).out).Encode(map[string]any{"dry_run": true, "command": "export", "source": source, "destination": destination, "grammars": names, "replace_selected": force, "actions": []string{"Stage copies of selected grammar directories", "Replace selected destination directories with rollback on failure"}})
}
