package treesitter

import "context"

// WithToolPaths resolves symbolic build-tool names without shell interpolation.
func WithToolPaths(runner CommandRunner, tools ToolPaths) CommandRunner {
	return toolRunner{runner: runner, tools: tools}
}

type toolRunner struct {
	runner CommandRunner
	tools  ToolPaths
}

func (r toolRunner) command(cmd Command) Command {
	switch cmd.Name {
	case "tree-sitter":
		cmd.Name = r.tools.TreeSitter
	case "git":
		cmd.Name = r.tools.Git
	case "npm":
		cmd.Name = r.tools.NPM
	}
	return cmd
}
func (r toolRunner) Run(ctx context.Context, cmd Command) error {
	return r.runner.Run(ctx, r.command(cmd))
}
func (r toolRunner) Output(ctx context.Context, cmd Command) (string, error) {
	return r.runner.Output(ctx, r.command(cmd))
}
