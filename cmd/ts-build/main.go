package main

import (
	"context"
	"fmt"
	"os"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	tsbuild "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

func main() {
	// The verbose flag is read by the App before constructing the runner; we
	// inspect args here so that main.go stays simple and zero-allocation.
	verbose := hasFlag(os.Args[1:], "--verbose", "-v")
	quiet := hasFlag(os.Args[1:], "--quiet", "-q")

	var reporter _jsii.Reporter
	if quiet {
		reporter = _jsii.SilentReporter{}
	} else {
		reporter = _jsii.NewTextReporter(os.Stdout, verbose)
	}

	runner := _jsii.NewVerboseExecRunner(os.Stdout, os.Stderr, verbose)
	cleaner := tsbuild.NewCleaner()
	compactor := tsbuild.NewCompactorWithReporter(reporter)
	app := _jsii.NewApp(
		os.Stdout,
		os.Stderr,
		_jsii.OSPathLookup{},
		_jsii.LoadConfig,
		tsbuild.NewBuilderWithReporter(runner, reporter),
		tsbuild.NewCompilerWithReporter(runner, _jsii.OSPathLookup{}, reporter),
		tsbuild.NewMoverWithReporter(cleaner, reporter),
	).WithCleaner(cleaner).WithCompactor(compactor)

	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// hasFlag reports whether any of the provided flag names appear in args.
func hasFlag(args []string, names ...string) bool {
	for _, arg := range args {
		for _, name := range names {
			if arg == name {
				return true
			}
		}
	}
	return false
}
