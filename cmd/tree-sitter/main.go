package main

import (
	"context"
	"fmt"
	"os"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	tsbuild "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

func main() {
	runner := _jsii.NewExecRunner(os.Stdout, os.Stderr)
	app := _jsii.NewApp(
		os.Stdout,
		os.Stderr,
		_jsii.OSPathLookup{},
		_jsii.LoadConfig,
		tsbuild.NewBuilder(runner, os.Stdout, os.Stderr),
		tsbuild.NewCompiler(runner, _jsii.OSPathLookup{}, os.Stdout, os.Stderr),
		tsbuild.NewMover(tsbuild.RealClock{}, tsbuild.NewCleaner(), os.Stdout, os.Stderr),
	)

	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
