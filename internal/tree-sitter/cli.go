package treesitter

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

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
}

type CompileRequest struct {
	Language              string
	OS                    string
	Arch                  string
	AllowCrossValidation  bool
	StaticCrossValidation bool
}

type MoveRequest struct {
	Language              string
	Mode                  MoveMode
	Clean                 bool
	Force                 bool
	AllowCrossValidation  bool
	StaticCrossValidation bool
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

type ConfigLoader func(string) (*Config, error)

type PathLookup interface {
	LookPath(string) (string, error)
}

type OSPathLookup struct{}

func (OSPathLookup) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

type App struct {
	stdout   io.Writer
	stderr   io.Writer
	lookup   PathLookup
	load     ConfigLoader
	builder  Builder
	compiler Compiler
	mover    Mover
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

func (a *App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		a.printRootUsage()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		a.printRootUsage()
		return nil
	case "build":
		return a.runBuild(ctx, args[1:])
	case "compile":
		return a.runCompile(ctx, args[1:])
	case "move":
		return a.runMove(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (a *App) runBuild(ctx context.Context, args []string) error {
	if a.builder == nil {
		return errors.New("build command is not configured")
	}

	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	configPath := fs.String("config", DefaultConfigPath, configFlagDescription)
	language := fs.String("language", "", "Only build the specified language.")
	force := fs.Bool("force", false, "Re-clone existing repositories before building.")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cfg, err := a.load(*configPath)
	if err != nil {
		return err
	}
	if err := RequireTools(a.lookup, "git", "tree-sitter"); err != nil {
		return err
	}

	return a.builder.Build(ctx, cfg, BuildRequest{
		Language: *language,
		Force:    *force,
	})
}

func (a *App) runCompile(ctx context.Context, args []string) error {
	if a.compiler == nil {
		return errors.New("compile command is not configured")
	}

	fs := flag.NewFlagSet("compile", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	configPath := fs.String("config", DefaultConfigPath, configFlagDescription)
	language := fs.String("language", "", "Only compile the specified language.")
	targetOS := fs.String("os", "", "Target operating system (linux, windows, macos).")
	targetArch := fs.String("arch", "", "Target architecture (amd64, arm64).")
	allowCrossValidation := fs.Bool("allow-cross-validation", false, "Allow cross-platform static validation for non-host binaries.")
	staticCrossValidation := fs.Bool("static-cross-validation", false, "Alias for --allow-cross-validation.")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cfg, err := a.load(*configPath)
	if err != nil {
		return err
	}
	if err := RequireTools(a.lookup, "tree-sitter"); err != nil {
		return err
	}

	return a.compiler.Compile(ctx, cfg, CompileRequest{
		Language:              *language,
		OS:                    *targetOS,
		Arch:                  *targetArch,
		AllowCrossValidation:  *allowCrossValidation || *staticCrossValidation,
		StaticCrossValidation: *staticCrossValidation,
	})
}

func (a *App) runMove(ctx context.Context, args []string) error {
	if a.mover == nil {
		return errors.New("move command is not configured")
	}

	fs := flag.NewFlagSet("move", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	configPath := fs.String("config", DefaultConfigPath, configFlagDescription)
	language := fs.String("language", "", "Only move the specified language.")
	jsonMode := fs.Bool("json", false, "Move binaries and node-types.json only.")
	scmMode := fs.Bool("scm", false, "Move binaries and queries/ only.")
	bothMode := fs.Bool("both", false, "Move binaries, node-types.json, and queries/.")
	clean := fs.Bool("clean", true, "Clean the build directory for each successfully moved language.")
	force := fs.Bool("force", false, "Overwrite existing output files.")
	allowCrossValidation := fs.Bool("allow-cross-validation", false, "Allow cross-platform static validation for non-host binaries.")
	staticCrossValidation := fs.Bool("static-cross-validation", false, "Alias for --allow-cross-validation.")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cfg, err := a.load(*configPath)
	if err != nil {
		return err
	}

	mode, err := ResolveMoveMode(*jsonMode, *scmMode, *bothMode, cfg.Output.DefaultMoveMode)
	if err != nil {
		return err
	}

	return a.mover.Move(ctx, cfg, MoveRequest{
		Language:              *language,
		Mode:                  mode,
		Clean:                 *clean,
		Force:                 *force,
		AllowCrossValidation:  *allowCrossValidation || *staticCrossValidation,
		StaticCrossValidation: *staticCrossValidation,
	})
}

func (a *App) printRootUsage() {
	fmt.Fprintln(a.stdout, "Tree-Sitter grammar builder")
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "Usage:")
	fmt.Fprintln(a.stdout, "  tree-sitter <command> [flags]")
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "Commands:")
	fmt.Fprintln(a.stdout, "  build    Clone repositories and generate grammar sources")
	fmt.Fprintln(a.stdout, "  compile  Build platform-specific shared libraries from grammar sources")
	fmt.Fprintln(a.stdout, "  move     Organize compiled artifacts (--json, --scm, --both), create manifests, and clean build output")
}

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

type ExecRunner struct {
	stdout io.Writer
	stderr io.Writer
}

func NewExecRunner(stdout, stderr io.Writer) *ExecRunner {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return &ExecRunner{stdout: stdout, stderr: stderr}
}

func (r *ExecRunner) Run(ctx context.Context, cmd Command) error {
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	execCmd.Dir = cmd.Dir
	execCmd.Env = mergeEnv(os.Environ(), cmd.Env)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	execCmd.Stdout = io.MultiWriter(r.stdout, &stdout)
	if cmd.SilenceStderr {
		execCmd.Stderr = &stderr
	} else {
		execCmd.Stderr = io.MultiWriter(r.stderr, &stderr)
	}

	if err := execCmd.Run(); err != nil {
		outStr := strings.TrimSpace(stdout.String())
		errStr := strings.TrimSpace(stderr.String())
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
