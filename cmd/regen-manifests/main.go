// regen-manifests is retained as a compatibility command, but deliberately
// refuses to rewrite release metadata without rebuilding and validating the
// native artifacts. Manifest ABI data must come from the final shared library.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "manifest-only regeneration is unsafe and no longer supported; run tree-sitter build, compile, and move so the final native binaries are validated")
	os.Exit(2)
}
