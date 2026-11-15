package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

func rewriteQueries(directory string, language ts.Language, policy ts.QueryPolicy) error {
	for _, rule := range policy.Rewrites {
		if rule.Language != language.Name {
			continue
		}
		if filepath.Base(rule.File) != rule.File || !strings.HasSuffix(rule.File, ".scm") || rule.Find == "" {
			return fmt.Errorf("invalid query rewrite for %s", language.Name)
		}
		path := filepath.Join(directory, rule.File)
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(strings.ReplaceAll(string(raw), rule.Find, rule.Replace)), 0o644); err != nil {
			return err
		}
	}
	return nil
}
