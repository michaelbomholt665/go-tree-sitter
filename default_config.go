// Package bundledconfig contains the configuration shipped with ts-build.
package bundledconfig

import _ "embed"

// YAML is the finalized repository catalog embedded at build time.
//
//go:embed tree-sitter-config.yaml
var YAML string
