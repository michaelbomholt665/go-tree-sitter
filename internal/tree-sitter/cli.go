package treesitter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

const AppVersion = "1.0.0"
const DefaultConfigPath = "tree-sitter-config.yaml"
const configFlagDescription = "Path to the YAML configuration file."

type MoveMode string

const (
	MoveModeJSON MoveMode = "json"
	MoveModeSCM  MoveMode = "scm"
	MoveModeBoth MoveMode = "both"
)

type BuildRequest struct {
	Language string
	Force    bool
	Prune    bool
}

type CompileRequest struct {
	Language              string
	OS                    string
	Arch                  string
	AllowCrossValidation  bool
	StaticCrossValidation bool
	BuildWasm             bool
}

type MoveRequest struct {
	Language              string
	Mode                  MoveMode
	Clean                 bool
	Force                 bool
	AllowCrossValidation  bool
	StaticCrossValidation bool
	GenerateManifest      bool
	CopySCM               bool
	CopySource            bool
	CopyJS                bool
	IncludeWasm           bool
	Compact               bool
	CheckCompact          bool
}

type CleanRequest struct {
	Language string
	Prune    bool
}

type CompactRequest struct {
	Language string
	Check    bool
}

type Builder interface {
	Build(context.Context, *Config, BuildRequest) error
}

type Compiler interface {
	Compile(context.Context, *Config, CompileRequest) error
}

type Mover interface {
	Move(context.Context, *Config, MoveRequest) error
}

type Cleaner interface {
	Clean(context.Context, *Config, CleanRequest) error
}

type Compactor interface {
	Compact(context.Context, *Config, CompactRequest) error
}

type ConfigLoader func(string) (*Config, error)

type PathLookup interface {
	LookPath(string) (string, error)
}

type OSPathLookup struct{}

func (OSPathLookup) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// ─── App ─────────────────────────────────────────────────────────────────────

type App struct {
	stdout    io.Writer
	stderr    io.Writer
	lookup    PathLookup
	load      ConfigLoader
	builder   Builder
	compiler  Compiler
	mover     Mover
	cleaner   Cleaner
	compactor Compactor
}

func NewApp(stdout, stderr io.Writer, lookup PathLookup, load ConfigLoader, builder Builder, compiler Compiler, mover Mover) *App {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if lookup == nil {
		lookup = OSPathLookup{}
	}
	if load == nil {
		load = LoadConfig
	}

	return &App{
		stdout:   stdout,
		stderr:   stderr,
		lookup:   lookup,
		load:     load,
		builder:  builder,
		compiler: compiler,
		mover:    mover,
	}
}

func (a *App) WithCleaner(cleaner Cleaner) *App {
	a.cleaner = cleaner
	return a
}

func (a *App) WithCompactor(compactor Compactor) *App {
	a.compactor = compactor
	return a
}

// Run executes the Cobra command tree with the supplied argument slice.
func (a *App) Run(ctx context.Context, args []string) error {
	root := a.NewRootCmd()
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}

// ─── Root command ─────────────────────────────────────────────────────────────

// NewRootCmd builds and returns the Cobra root command for use in tests and
// in main.go.
func (a *App) NewRootCmd() *cobra.Command {
	var configPath string
	var verbose bool
	var quiet bool

	root := &cobra.Command{
		Use:     "ts-build",
		Version: AppVersion,
		Short:   "Tree-Sitter grammar builder",
		Long: `ts-build — clone, build, cross-compile, and publish tree-sitter grammars.

Use --verbose to stream raw child-process output.
Use --quiet to silence all non-error output.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetOut(a.stdout)
	root.SetErr(a.stderr)

	// Persistent flags available to every subcommand.
	root.PersistentFlags().StringVarP(&configPath, "config", "c", DefaultConfigPath, configFlagDescription)
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Stream raw child-process stdout/stderr.")
	root.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Silence all non-error output.")

	// Helper to build a reporter respecting the global flags.
	makeReporter := func() Reporter {
		if quiet {
			return SilentReporter{}
		}
		return NewTextReporter(a.stdout, verbose)
	}

	root.AddCommand(
		a.buildCmd(&configPath, &verbose, makeReporter),
		a.compileCmd(&configPath, &verbose, makeReporter),
		a.moveCmd(&configPath, &verbose, makeReporter),
		a.cleanCmd(&configPath, &verbose, makeReporter),
		a.compactCmd(&configPath, &verbose, makeReporter),
		&cobra.Command{
			Use:   "version",
			Short: "Print the version of ts-build",
			Run: func(_ *cobra.Command, _ []string) {
				fmt.Fprintf(a.stdout, "ts-build version %s\n", AppVersion)
			},
		},
	)
	root.AddCommand(a.completionCmd(root))

	return root
}

// ─── build subcommand ─────────────────────────────────────────────────────────

func (a *App) buildCmd(configPath *string, verbose *bool, makeReporter func() Reporter) *cobra.Command {
	var language string
	var force bool
	var prune bool

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Clone repositories and generate grammar sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if a.builder == nil {
				return errors.New("build command is not configured")
			}
			cfg, err := a.load(*configPath)
			if err != nil {
				return err
			}
			if err := RequireTools(a.lookup, "git", "tree-sitter"); err != nil {
				return err
			}
			return a.builder.Build(cmd.Context(), cfg, BuildRequest{
				Language: language,
				Force:    force,
				Prune:    prune,
			})
		},
	}

	cmd.Flags().StringVarP(&language, "language", "l", "", "Only build the specified language.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Re-clone existing repositories before building.")
	cmd.Flags().BoolVar(&prune, "prune", false, "Prune build cache (.git, test corpus, etc.) after building.")

	// Suppress the unused parameter warning – makeReporter / verbose are used
	// by builder/compiler/mover once they consume the Reporter interface.
	_ = verbose
	_ = makeReporter

	return cmd
}

// ─── compile subcommand ───────────────────────────────────────────────────────

func (a *App) compileCmd(configPath *string, verbose *bool, makeReporter func() Reporter) *cobra.Command {
	var language string
	var targetOS string
	var targetArch string
	var allowCrossValidation bool
	var staticCrossValidation bool
	var wasmFlag bool

	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Build platform-specific shared libraries from grammar sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if a.compiler == nil {
				return errors.New("compile command is not configured")
			}
			cfg, err := a.load(*configPath)
			if err != nil {
				return err
			}
			if err := RequireTools(a.lookup, "tree-sitter"); err != nil {
				return err
			}
			return a.compiler.Compile(cmd.Context(), cfg, CompileRequest{
				Language:              language,
				OS:                    targetOS,
				Arch:                  targetArch,
				AllowCrossValidation:  allowCrossValidation || staticCrossValidation,
				StaticCrossValidation: staticCrossValidation,
				BuildWasm:             wasmFlag,
			})
		},
	}

	cmd.Flags().StringVarP(&language, "language", "l", "", "Only compile the specified language.")
	cmd.Flags().StringVarP(&targetOS, "os", "o", "", "Target operating system (linux, windows, macos).")
	cmd.Flags().StringVarP(&targetArch, "arch", "a", "", "Target architecture (amd64, arm64).")
	cmd.Flags().BoolVar(&allowCrossValidation, "allow-cross-validation", false, "Allow cross-platform static validation for non-host binaries.")
	cmd.Flags().BoolVar(&staticCrossValidation, "static-cross-validation", false, "Alias for --allow-cross-validation.")
	cmd.Flags().BoolVar(&wasmFlag, "wasm", false, "Build WebAssembly (WASM) parser artifact.")

	_ = verbose
	_ = makeReporter

	return cmd
}

// ─── move subcommand ──────────────────────────────────────────────────────────

func (a *App) moveCmd(configPath *string, verbose *bool, makeReporter func() Reporter) *cobra.Command {
	var language string
	var jsonMode bool
	var scmFlag bool
	var noSCM bool
	var bothMode bool
	var manifestFlag bool
	var noManifest bool
	var clean bool
	var force bool
	var allowCrossValidation bool
	var staticCrossValidation bool
	var sourceFlag bool
	var cSourceFlag bool
	var jsFlag bool
	var wasmFlag bool
	var compactFlag bool
	var checkCompactFlag bool

	cmd := &cobra.Command{
		Use:   "move",
		Short: "Organize compiled artifacts, create manifests, and clean build output",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if a.mover == nil {
				return errors.New("move command is not configured")
			}
			cfg, err := a.load(*configPath)
			if err != nil {
				return err
			}

			if noManifest {
				manifestFlag = false
			}
			if noSCM {
				scmFlag = false
			}

			// In legacy usage, --scm was a mutually exclusive mode flag alongside --json and --both.
			// When --scm is explicitly passed as true (or bare --scm), it acts as legacy scmMode
			// if other mode flags are checked.
			scmModeForLegacy := cmd.Flags().Changed("scm") && scmFlag
			var mode MoveMode
			if jsonMode || bothMode || scmModeForLegacy {
				mode, err = ParseMoveMode(jsonMode, scmModeForLegacy, bothMode)
				if err != nil {
					return err
				}
			} else {
				mode, err = ParseConfiguredMoveMode(cfg.Output.DefaultMoveMode)
				if err != nil {
					return err
				}
			}

			// Resolve CopySCM: explicit --scm / --no-scm flag takes precedence over move mode.
			copySCM := mode.IncludesQueries()
			if cmd.Flags().Changed("scm") || noSCM {
				copySCM = scmFlag
			}

			copySource := sourceFlag || cSourceFlag

			return a.mover.Move(cmd.Context(), cfg, MoveRequest{
				Language:              language,
				Mode:                  mode,
				Clean:                 clean,
				Force:                 force,
				AllowCrossValidation:  allowCrossValidation || staticCrossValidation,
				StaticCrossValidation: staticCrossValidation,
				GenerateManifest:      manifestFlag,
				CopySCM:               copySCM,
				CopySource:            copySource,
				CopyJS:                jsFlag,
				IncludeWasm:           wasmFlag,
				Compact:               compactFlag,
				CheckCompact:          checkCompactFlag,
			})
		},
	}

	cmd.Flags().StringVarP(&language, "language", "l", "", "Only move the specified language.")
	cmd.Flags().BoolVar(&jsonMode, "json", false, "Move binaries and node-types.json only.")
	cmd.Flags().BoolVar(&scmFlag, "scm", true, "Publish .scm query files.")
	cmd.Flags().BoolVar(&noSCM, "no-scm", false, "Skip publishing .scm query files.")
	_ = cmd.Flags().MarkHidden("no-scm")
	cmd.Flags().BoolVar(&bothMode, "both", false, "Move binaries, node-types.json, and queries/.")
	cmd.Flags().BoolVar(&manifestFlag, "manifest", true, "Generate and validate manifest.json.")
	cmd.Flags().BoolVar(&noManifest, "no-manifest", false, "Skip generating and validating manifest.json.")
	_ = cmd.Flags().MarkHidden("no-manifest")
	cmd.Flags().BoolVar(&clean, "clean", true, "Clean the build directory for each successfully moved language.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing output files.")
	cmd.Flags().BoolVar(&allowCrossValidation, "allow-cross-validation", false, "Allow cross-platform static validation for non-host binaries.")
	cmd.Flags().BoolVar(&staticCrossValidation, "static-cross-validation", false, "Alias for --allow-cross-validation.")
	cmd.Flags().BoolVar(&sourceFlag, "source", false, "Preserve C/C++ parser sources (src/).")
	cmd.Flags().BoolVar(&cSourceFlag, "c-source", false, "Alias for --source.")
	cmd.Flags().BoolVar(&jsFlag, "js", false, "Preserve grammar.js.")
	cmd.Flags().BoolVar(&wasmFlag, "wasm", false, "Stage WebAssembly (WASM) parser artifact.")
	cmd.Flags().BoolVar(&compactFlag, "compact", false, "Generate and validate compact-node-types.yaml alongside node-types.json.")
	cmd.Flags().BoolVar(&checkCompactFlag, "check-compact", false, "Validate existing compact-node-types.yaml against node-types.json.")

	_ = verbose
	_ = makeReporter

	return cmd
}

// ─── clean subcommand ─────────────────────────────────────────────────────────

func (a *App) cleanCmd(configPath *string, verbose *bool, makeReporter func() Reporter) *cobra.Command {
	var language string
	var prune bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean or prune build directories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if a.cleaner == nil {
				return errors.New("clean command is not configured")
			}
			cfg, err := a.load(*configPath)
			if err != nil {
				return err
			}
			return a.cleaner.Clean(cmd.Context(), cfg, CleanRequest{
				Language: language,
				Prune:    prune,
			})
		},
	}

	cmd.Flags().StringVarP(&language, "language", "l", "", "Only clean the specified language.")
	cmd.Flags().BoolVar(&prune, "prune", false, "Prune extraneous cache files while preserving grammar sources and manifests.")

	_ = verbose
	_ = makeReporter

	return cmd
}

// ─── compact subcommand ───────────────────────────────────────────────────────

func (a *App) compactCmd(configPath *string, verbose *bool, makeReporter func() Reporter) *cobra.Command {
	var language string
	var check bool

	cmd := &cobra.Command{
		Use:   "compact",
		Short: "Generate or validate token-efficient compact node types",
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.compactor == nil {
				return errors.New("compact command is not configured")
			}
			cfg, err := a.load(*configPath)
			if err != nil {
				return err
			}
			lang := language
			if lang == "" && len(args) > 0 {
				lang = args[0]
			}
			return a.compactor.Compact(cmd.Context(), cfg, CompactRequest{
				Language: lang,
				Check:    check,
			})
		},
	}

	cmd.Flags().StringVarP(&language, "language", "l", "", "Only compact the specified language.")
	cmd.Flags().BoolVar(&check, "check", false, "Validate existing compact YAML without regenerating.")

	_ = verbose
	_ = makeReporter

	return cmd
}

// ─── completion subcommand ────────────────────────────────────────────────────

func (a *App) completionCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `To enable shell completion, source the output of this command.

Bash:
  source <(ts-build completion bash)

Zsh:
  source <(ts-build completion zsh)

Fish:
  ts-build completion fish | source

PowerShell:
  ts-build completion powershell | Out-String | Invoke-Expression`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(_ *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(a.stdout)
			case "zsh":
				return root.GenZshCompletion(a.stdout)
			case "fish":
				return root.GenFishCompletion(a.stdout, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(a.stdout)
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
}

// ─── Move mode helpers ────────────────────────────────────────────────────────

func ParseMoveMode(jsonMode, scmMode, bothMode bool) (MoveMode, error) {
	selected := 0
	if jsonMode {
		selected++
	}
	if scmMode {
		selected++
	}
	if bothMode {
		selected++
	}

	if selected == 0 {
		return "", errors.New("no move mode selected")
	}
	if selected > 1 {
		return "", errors.New("move mode flags are mutually exclusive")
	}
	if jsonMode {
		return MoveModeJSON, nil
	}
	if scmMode {
		return MoveModeSCM, nil
	}
	return MoveModeBoth, nil
}

func ResolveMoveMode(jsonMode, scmMode, bothMode bool, configuredDefault string) (MoveMode, error) {
	if jsonMode || scmMode || bothMode {
		return ParseMoveMode(jsonMode, scmMode, bothMode)
	}
	return ParseConfiguredMoveMode(configuredDefault)
}

func ParseConfiguredMoveMode(value string) (MoveMode, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = strings.TrimPrefix(normalized, "--")

	switch MoveMode(normalized) {
	case "", MoveModeBoth:
		return MoveModeBoth, nil
	case MoveModeJSON:
		return MoveModeJSON, nil
	case MoveModeSCM:
		return MoveModeSCM, nil
	default:
		return "", fmt.Errorf("unsupported move mode %q", value)
	}
}

func (m MoveMode) IncludesNodeTypes() bool {
	return m == MoveModeJSON || m == MoveModeBoth
}

func (m MoveMode) IncludesQueries() bool {
	return m == MoveModeSCM || m == MoveModeBoth
}

// ─── Tool prerequisite check ──────────────────────────────────────────────────

func RequireTools(lookup PathLookup, names ...string) error {
	if lookup == nil {
		lookup = OSPathLookup{}
	}

	var missing []string
	for _, name := range names {
		if _, err := lookup.LookPath(name); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required tool(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

// ─── Command / runner types ───────────────────────────────────────────────────

type Command struct {
	Name          string
	Args          []string
	Dir           string
	Env           map[string]string
	SilenceStderr bool
}

func (c Command) String() string {
	if len(c.Args) == 0 {
		return c.Name
	}
	return c.Name + " " + strings.Join(c.Args, " ")
}

type CommandRunner interface {
	Run(context.Context, Command) error
	Output(context.Context, Command) (string, error)
}

// ─── ExecRunner ───────────────────────────────────────────────────────────────

// ExecRunner executes commands against the real OS.  When verbose=true it
// streams stdout/stderr in real time in addition to buffering them.  When
// verbose=false (clean mode) it buffers silently, surfacing output only on
// failure via CommandError.
type ExecRunner struct {
	stdout  io.Writer
	stderr  io.Writer
	verbose bool
}

func NewExecRunner(stdout, stderr io.Writer) *ExecRunner {
	return NewVerboseExecRunner(stdout, stderr, true)
}

// NewVerboseExecRunner constructs an ExecRunner that streams output when
// verbose=true and buffers silently when verbose=false.
func NewVerboseExecRunner(stdout, stderr io.Writer, verbose bool) *ExecRunner {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return &ExecRunner{stdout: stdout, stderr: stderr, verbose: verbose}
}

func (r *ExecRunner) Run(ctx context.Context, cmd Command) error {
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	execCmd.Dir = cmd.Dir
	execCmd.Env = mergeEnv(os.Environ(), cmd.Env)

	var stdoutBuf, stderrBuf bytes.Buffer

	if r.verbose {
		execCmd.Stdout = io.MultiWriter(r.stdout, &stdoutBuf)
		if cmd.SilenceStderr {
			execCmd.Stderr = &stderrBuf
		} else {
			execCmd.Stderr = io.MultiWriter(r.stderr, &stderrBuf)
		}
	} else {
		execCmd.Stdout = &stdoutBuf
		execCmd.Stderr = &stderrBuf
	}

	if err := execCmd.Run(); err != nil {
		outStr := strings.TrimSpace(stdoutBuf.String())
		errStr := strings.TrimSpace(stderrBuf.String())
		var combined string
		switch {
		case outStr != "" && errStr != "":
			combined = outStr + "\n" + errStr
		case outStr != "":
			combined = outStr
		default:
			combined = errStr
		}
		return &CommandError{
			Command: cmd,
			Output:  combined,
			Err:     err,
		}
	}
	return nil
}

func (r *ExecRunner) Output(ctx context.Context, cmd Command) (string, error) {
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	execCmd.Dir = cmd.Dir
	execCmd.Env = mergeEnv(os.Environ(), cmd.Env)

	output, err := execCmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		return "", &CommandError{Command: cmd, Output: trimmed, Err: err}
	}
	return trimmed, nil
}

// ─── CommandError ─────────────────────────────────────────────────────────────

type CommandError struct {
	Command Command
	Output  string
	Err     error
}

func (e *CommandError) Error() string {
	if e.Output == "" {
		return fmt.Sprintf("%s: %v", e.Command.String(), e.Err)
	}
	return fmt.Sprintf("%s: %v\n%s", e.Command.String(), e.Err, e.Output)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

// ─── env helpers ─────────────────────────────────────────────────────────────

func mergeEnv(base []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return base
	}

	merged := make(map[string]string, len(base)+len(overrides))
	for _, entry := range base {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		merged[key] = value
	}
	for key, value := range overrides {
		merged[key] = value
	}

	result := make([]string, 0, len(merged))
	for key, value := range merged {
		result = append(result, key+"="+value)
	}
	return result
}
