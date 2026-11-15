package treesitter

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/michaelbomholt665/go-tree-sitter/internal/fileupdate"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func (a *App) supportCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{Use: "support", Short: "Inspect or update language-version assessments"}
	show := a.listCmd(configPath)
	show.Use, show.Short = "show", "Show a grammar's recorded language-version assessment"
	_ = show.MarkFlagRequired("language")
	cmd.AddCommand(show, a.supportUpdateCmd(configPath))
	return cmd
}

type supportUpdateResult struct {
	Applied        bool           `json:"applied"`
	Changed        bool           `json:"changed"`
	ReportsChecked int            `json:"reports_checked"`
	Before         GrammarSupport `json:"before"`
	After          GrammarSupport `json:"after"`
}

func (a *App) supportUpdateCmd(configPath *string) *cobra.Command {
	var language, assessmentPath, expectedRevision string
	var apply, jsonOutput, checkReport bool
	cmd := &cobra.Command{
		Use: "update", Short: "Replace one assessment; preview unless --apply is supplied",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			defer func() {
				if err != nil && jsonOutput {
					_ = json.NewEncoder(a.stdout).Encode(map[string]string{"error": err.Error()})
				}
			}()
			if language == "" || assessmentPath == "" {
				return errors.New("--language and --assessment are required")
			}
			if apply && isDryRun(cmd) {
				return errors.New("--apply and --dry-run are mutually exclusive")
			}
			var input io.Reader = cmd.InOrStdin()
			base, err := os.Getwd()
			if err != nil {
				return err
			}
			if assessmentPath != "-" {
				file, openErr := os.Open(assessmentPath)
				if openErr != nil {
					return openErr
				}
				defer file.Close()
				input = file
				absolute, pathErr := filepath.Abs(assessmentPath)
				if pathErr != nil {
					return pathErr
				}
				base = filepath.Dir(absolute)
			}
			var assessment LanguageSupportAssessment
			decoder := json.NewDecoder(input)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&assessment); err != nil {
				return fmt.Errorf("decode assessment: %w", err)
			}
			var extra any
			if err := decoder.Decode(&extra); err != io.EOF {
				return errors.New("assessment must contain exactly one JSON object")
			}
			if assessment.Claim == "" {
				return errors.New("assessment claim must be explicit: unverified or verified")
			}
			if err := assessment.Validate(); err != nil {
				return err
			}
			checked, err := prepareSupportReports(&assessment, base, checkReport)
			if err != nil {
				return err
			}

			// Cooperative writers serialize updates. The content check also detects external edits.
			if apply {
				release, lockErr := lockConfigUpdate(*configPath)
				if lockErr != nil {
					return lockErr
				}
				defer release()
			}
			original, err := readCommandConfig(cmd, *configPath)
			if err != nil {
				return err
			}
			var cfg Config
			if err := yaml.Unmarshal(original, &cfg); err != nil {
				return err
			}
			applyDefaults(&cfg)
			if err := cfg.Validate(); err != nil {
				return err
			}
			var selected *Language
			for i := range cfg.Languages {
				if cfg.Languages[i].Name == language {
					selected = &cfg.Languages[i]
					break
				}
			}
			if selected == nil {
				return fmt.Errorf("unknown grammar %q", language)
			}
			if expectedRevision != "" && (!isHexDigest(expectedRevision, 40) || !strings.EqualFold(expectedRevision, selected.Revision)) {
				return errors.New("active grammar revision does not match --expect-revision")
			}
			if !strings.EqualFold(assessment.AssessedRevision, selected.Revision) {
				return errors.New("assessed_revision must match the active grammar revision")
			}
			result := supportUpdateResult{Before: grammarSupport(*selected), ReportsChecked: checked}
			selected.LanguageSupport = &assessment
			result.After = grammarSupport(*selected)
			before, _ := json.Marshal(result.Before.Assessment)
			after, _ := json.Marshal(assessment)
			result.Changed = !bytes.Equal(before, after)

			if result.Changed {
				updated, err := replaceSupportAssessment(original, language, &assessment)
				if err != nil {
					return err
				}
				var roundTrip Config
				if err := yaml.Unmarshal(updated, &roundTrip); err != nil {
					return fmt.Errorf("validate updated YAML: %w", err)
				}
				applyDefaults(&roundTrip)
				if err := roundTrip.Validate(); err != nil {
					return err
				}
				expected, _ := json.Marshal(cfg)
				actual, _ := json.Marshal(roundTrip)
				if !bytes.Equal(expected, actual) {
					return errors.New("update would change settings outside the selected assessment")
				}
				if apply {
					if err := cmd.Context().Err(); err != nil {
						return err
					}
					if err := writeConfigReplacement(*configPath, original, updated); err != nil {
						return err
					}
				}
			}
			result.Applied = apply
			if jsonOutput {
				return json.NewEncoder(a.stdout).Encode(result)
			}
			status := "Preview (not saved)"
			if apply {
				status = "Applied"
			}
			_, err = fmt.Fprintf(a.stdout, "%s: %s, claim %s -> %s, Rule %d; %d reports checked\n", status, language, result.Before.Claim, result.After.Claim, result.After.Rule, checked)
			if !jsonOutput {
				encoder := json.NewEncoder(a.stdout)
				encoder.SetIndent("", "  ")
				if err == nil {
					err = encoder.Encode(result)
				}
			}
			return err
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&language, "language", "l", "", "Exact configured grammar identifier")
	flags.StringVar(&assessmentPath, "assessment", "", "Complete assessment JSON file, or - for stdin")
	flags.StringVar(&expectedRevision, "expect-revision", "", "Reject if the active grammar pin differs from this full SHA")
	flags.BoolVar(&apply, "apply", false, "Save the validated assessment")
	flags.BoolVar(&checkReport, "check-report", false, "Verify local syntax-fixture report SHA-256 checksums; never execute tests")
	flags.BoolVar(&jsonOutput, "json", false, "Emit structured results and assessment validation errors")
	return cmd
}

func prepareSupportReports(assessment *LanguageSupportAssessment, base string, check bool) (int, error) {
	checked := 0
	for i := range assessment.Evidence {
		e := &assessment.Evidence[i]
		if e.Kind != "syntax_fixtures" {
			continue
		}
		path := e.Source
		parsed, err := url.Parse(path)
		if !filepath.IsAbs(path) && err == nil && parsed.Scheme != "" {
			if parsed.Scheme != "file" || (parsed.Host != "" && parsed.Host != "localhost") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
				if check {
					return checked, fmt.Errorf("evidence[%d]: --check-report requires a local report path or file URL", i)
				}
				continue
			}
			path = filepath.FromSlash(parsed.Path)
			if filepath.VolumeName(base) != "" && strings.HasPrefix(path, string(os.PathSeparator)) {
				path = strings.TrimPrefix(path, string(os.PathSeparator))
			}
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, path)
		}
		e.Source = filepath.Clean(path)
		if !check {
			continue
		}
		file, err := os.Open(e.Source)
		if err != nil {
			return checked, fmt.Errorf("evidence[%d]: %w", i, err)
		}
		info, statErr := file.Stat()
		if statErr != nil || !info.Mode().IsRegular() {
			file.Close()
			return checked, fmt.Errorf("evidence[%d]: report must be a regular file", i)
		}
		hash := sha256.New()
		_, readErr := io.Copy(hash, file)
		closeErr := file.Close()
		if readErr != nil {
			return checked, readErr
		}
		if closeErr != nil {
			return checked, closeErr
		}
		if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), e.SHA256) {
			return checked, fmt.Errorf("evidence[%d]: report SHA-256 mismatch", i)
		}
		checked++
	}
	if check && checked == 0 {
		return 0, errors.New("--check-report requires syntax_fixtures evidence")
	}
	return checked, nil
}

func replaceSupportAssessment(original []byte, language string, assessment *LanguageSupportAssessment) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(original, &doc); err != nil {
		return nil, err
	}
	root := doc.Content[0]
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value != "languages" {
			continue
		}
		for _, entry := range root.Content[i+1].Content {
			matched := false
			for j := 0; j < len(entry.Content); j += 2 {
				if entry.Content[j].Value == "name" && entry.Content[j+1].Value == language {
					matched = true
				}
			}
			if !matched {
				continue
			}
			var value yaml.Node
			if err := value.Encode(assessment); err != nil {
				return nil, err
			}
			replaced := false
			for j := 0; j < len(entry.Content); j += 2 {
				if entry.Content[j].Value != "language_support" {
					continue
				}
				old := entry.Content[j+1]
				value.HeadComment, value.LineComment, value.FootComment = old.HeadComment, old.LineComment, old.FootComment
				entry.Content[j+1], replaced = &value, true
				break
			}
			if !replaced {
				entry.Content = append(entry.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "language_support"}, &value)
			}
			var output bytes.Buffer
			encoder := yaml.NewEncoder(&output)
			encoder.SetIndent(2)
			if err := encoder.Encode(&doc); err != nil {
				return nil, err
			}
			if err := encoder.Close(); err != nil {
				return nil, err
			}
			return output.Bytes(), nil
		}
	}
	return nil, errors.New("grammar YAML entry not found")
}

func writeConfigReplacement(path string, original, updated []byte) error {
	return fileupdate.Replace(path, original, updated)
}
