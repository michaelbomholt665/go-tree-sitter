// sync-config is the compatibility entry point for ts-build sync-config.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/michaelbomholt665/go-tree-sitter/internal/configsync"
)

func main() {
	config := flag.String("config", "tree-sitter-config.yaml", "Path to YAML configuration")
	gomod := flag.String("gomod", "go.mod", "Path to go.mod")
	dryRun := flag.Bool("dry-run", false, "Print changes without writing")
	flag.Parse()
	if err := configsync.Run(*config, *gomod, *dryRun, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
