package treesitter

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/michaelbomholt665/go-tree-sitter/internal/configsync"
	"github.com/spf13/cobra"
)

// GrammarSupport describes the current pin separately from its historical assessment.
type GrammarSupport struct {
	Name       string                     `json:"name"`
	Revision   string                     `json:"revision"`
	MinVersion string                     `json:"min_version,omitempty"`
	MaxVersion string                     `json:"max_version,omitempty"`
	Claim      string                     `json:"claim"`
	Rule       int                        `json:"rule"`
	Stale      bool                       `json:"stale"`
	Assessment *LanguageSupportAssessment `json:"assessment,omitempty"`
}

func grammarSupport(lang Language) GrammarSupport {
	r := GrammarSupport{Name: lang.Name, Revision: lang.Revision, Claim: "unverified", Rule: 1, Assessment: lang.LanguageSupport}
	a := lang.LanguageSupport
	if a == nil {
		return r
	}
	r.Rule = 2 // Includes recorded but inconclusive documentation assessments.
	for _, e := range a.Evidence {
		if e.Kind == "syntax_fixtures" && e.Result == "passed" && len(e.TestedVersions) > 0 && len(e.SHA256) == 64 {
			r.Rule = 3
		}
	}
	r.Stale = !strings.EqualFold(a.AssessedRevision, lang.Revision)
	if !r.Stale {
		r.MinVersion, r.MaxVersion = a.MinVersion, a.MaxVersion
		if a.Claim != "" {
			r.Claim = a.Claim
		}
	}
	return r
}

func (a *App) configCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use: "config", Short: "Inspect, validate, or edit non-language settings",
	}
	cmd.AddCommand(&cobra.Command{
		Use: "path", Short: "Print the selected configuration file's absolute path",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := filepath.Abs(*configPath)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(a.stdout, path)
			return err
		},
	})
	a.addSettingsCommands(cmd, configPath)
	return cmd
}

func (a *App) listCmd(configPath *string) *cobra.Command {
	var language string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use: "list", Short: "List configured grammars and recorded language-version support",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := a.load(*configPath)
			if err != nil {
				return err
			}
			langs, err := cfg.ResolveLanguages(language)
			if err != nil {
				return err
			}
			rows := make([]GrammarSupport, 0, len(langs))
			for _, lang := range langs {
				rows = append(rows, grammarSupport(lang))
			}
			if jsonOutput {
				encoder := json.NewEncoder(a.stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(rows)
			}
			w := tabwriter.NewWriter(a.stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "GRAMMAR\tMIN VERSION\tMAX VERSION\tCLAIM\tRULE\tASSESSMENT")
			for _, row := range rows {
				min, max, status := row.MinVersion, row.MaxVersion, "current"
				if min == "" {
					min = "unknown"
				}
				if max == "" {
					max = "unknown"
				}
				if row.Assessment == nil {
					status = "none"
				} else if row.Stale {
					status = "stale"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n", row.Name, min, max, row.Claim, row.Rule, status)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVarP(&language, "language", "l", "", "List one grammar by name or alias.")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Include full historical assessments and evidence as JSON.")
	return cmd
}

func (a *App) exportCmd(configPath *string) *cobra.Command {
	var destination, language string
	var force bool
	cmd := &cobra.Command{
		Use: "export --to PATH", Short: "Copy published grammar directories to an external destination",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := a.load(*configPath)
			if err != nil {
				return err
			}
			config, err := filepath.Abs(*configPath)
			if err != nil {
				return err
			}
			source := cfg.Output.GrammarBase
			if !filepath.IsAbs(source) {
				source = filepath.Join(filepath.Dir(config), source)
			}
			return exportCatalog(cmd.Context(), cfg, source, destination, language, force)
		},
	}
	cmd.Flags().StringVar(&destination, "to", "", "Destination root; grammar directories are copied beneath it.")
	cmd.Flags().StringVarP(&language, "language", "l", "", "Export one published grammar (default: all published configured grammars).")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Replace selected destination grammar directories; other directories remain untouched.")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func (a *App) syncConfigCmd(configPath *string) *cobra.Command {
	var gomod string
	cmd := &cobra.Command{
		Use: "sync-config", Short: "Synchronize grammar package versions from go.mod without changing pins",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if gomod == "" {
				gomod = filepath.Join(filepath.Dir(*configPath), "go.mod")
			}
			if isDryRun(cmd) {
				raw, err := readCommandConfig(cmd, *configPath)
				if err != nil {
					return err
				}
				return configsync.Preview(raw, gomod, a.stdout)
			}
			return configsync.Run(*configPath, gomod, isDryRun(cmd), a.stdout)
		},
	}
	cmd.Flags().StringVar(&gomod, "gomod", "", "Path to go.mod (default: beside the configuration).")
	return cmd
}

func (a *App) regenManifestsCmd() *cobra.Command {
	return &cobra.Command{
		Use: "regen-manifests", Short: "Explain why manifest-only regeneration is disabled",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("manifest-only regeneration is unsafe and disabled; use ts-build build, compile, and move --manifest=true to validate real binaries before publishing")
		},
	}
}
