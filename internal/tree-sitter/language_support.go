package treesitter

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// LanguageSupportAssessment records a syntax-support claim for one immutable
// grammar revision, independently of the dependency currently selected.
type LanguageSupportAssessment struct {
	AssessedRevision string                    `yaml:"assessed_revision" json:"assessed_revision"`
	Claim            string                    `yaml:"claim,omitempty" json:"claim"`
	MinVersion       string                    `yaml:"min_version,omitempty" json:"min_version,omitempty"`
	MaxVersion       string                    `yaml:"max_version,omitempty" json:"max_version,omitempty"`
	Limitations      []string                  `yaml:"limitations,omitempty" json:"limitations,omitempty"`
	Evidence         []LanguageSupportEvidence `yaml:"evidence,omitempty" json:"evidence,omitempty"`
}

// LanguageSupportEvidence is an assessment record, not a request to fetch
// documentation or run tests during a build. Description states the scope.
type LanguageSupportEvidence struct {
	Kind           string   `yaml:"kind" json:"kind"`
	Source         string   `yaml:"source" json:"source"`
	Description    string   `yaml:"description" json:"description"`
	SHA256         string   `yaml:"sha256,omitempty" json:"sha256,omitempty"`
	Result         string   `yaml:"result,omitempty" json:"result,omitempty"`
	TestedVersions []string `yaml:"tested_versions,omitempty" json:"tested_versions,omitempty"`
}

// UnmarshalYAML rejects misspelled fields rather than silently losing exclusions.
func (a *LanguageSupportAssessment) UnmarshalYAML(node *yaml.Node) error {
	type plain LanguageSupportAssessment
	return decodeSupportFields(node, (*plain)(a), "assessed_revision", "claim", "min_version", "max_version", "limitations", "evidence")
}

func (e *LanguageSupportEvidence) UnmarshalYAML(node *yaml.Node) error {
	type plain LanguageSupportEvidence
	return decodeSupportFields(node, (*plain)(e), "kind", "source", "description", "sha256", "result", "tested_versions")
}

func decodeSupportFields(node *yaml.Node, target any, allowed ...string) error {
	var fields map[string]yaml.Node
	if err := node.Decode(&fields); err != nil {
		return err
	}
	for field := range fields {
		if !slices.Contains(allowed, field) {
			return fmt.Errorf("unknown language_support field %q", field)
		}
	}
	return node.Decode(target)
}

// Validate checks recorded evidence and pins structurally. It does not establish
// the truth of a claim, fetch its source, or rerun a recorded fixture suite.
func (a *LanguageSupportAssessment) Validate() error {
	if a == nil {
		return nil
	}
	if !isHexDigest(a.AssessedRevision, 40) {
		return fmt.Errorf("assessed_revision must be a full 40-character commit")
	}
	if a.Claim != "" && a.Claim != "unverified" && a.Claim != "verified" {
		return fmt.Errorf("claim must be unverified or verified")
	}
	if a.MinVersion != strings.TrimSpace(a.MinVersion) || a.MaxVersion != strings.TrimSpace(a.MaxVersion) {
		return fmt.Errorf("version bounds must not be blank or have surrounding whitespace")
	}
	for _, limitation := range a.Limitations {
		if strings.TrimSpace(limitation) == "" {
			return fmt.Errorf("limitations must not contain empty entries")
		}
	}
	verified := a.Claim == "verified"
	if verified && a.MinVersion == "" && a.MaxVersion == "" && len(a.Limitations) == 0 {
		return fmt.Errorf("verified assessment requires a version bound or explicit limitation")
	}
	if verified && len(a.Evidence) == 0 {
		return fmt.Errorf("verified assessment requires pinned documentation or successful version-specific syntax fixture evidence")
	}
	for i, evidence := range a.Evidence {
		if err := evidence.validate(a.AssessedRevision, verified); err != nil {
			return fmt.Errorf("evidence[%d]: %w", i, err)
		}
	}
	return nil
}

func (e LanguageSupportEvidence) validate(revision string, verified bool) error {
	if strings.TrimSpace(e.Source) == "" || strings.TrimSpace(e.Description) == "" {
		return fmt.Errorf("source and description (claim or tested scope) are required")
	}
	switch e.Kind {
	case "documentation":
		source, err := url.Parse(e.Source)
		if err != nil || source.Scheme != "https" || source.Host == "" || source.User != nil {
			return fmt.Errorf("documentation source must be an HTTPS URL without credentials")
		}
		if verified {
			pinned := false
			for _, part := range strings.Split(source.Path, "/") {
				if strings.EqualFold(part, revision) {
					pinned = true
				}
			}
			if !pinned {
				return fmt.Errorf("verified documentation source must pin assessed_revision in its URL path")
			}
		}
		if e.SHA256 != "" || e.Result != "" || len(e.TestedVersions) != 0 {
			return fmt.Errorf("sha256, result and tested_versions apply only to syntax_fixtures evidence")
		}
	case "syntax_fixtures":
		if e.Result != "passed" || !isHexDigest(e.SHA256, 64) || len(e.TestedVersions) == 0 {
			return fmt.Errorf("syntax_fixtures evidence requires result: passed, the report sha256 and tested_versions")
		}
		for _, version := range e.TestedVersions {
			if strings.TrimSpace(version) == "" {
				return fmt.Errorf("tested_versions must not contain empty entries")
			}
		}
	default:
		return fmt.Errorf("kind must be documentation or syntax_fixtures")
	}
	return nil
}

func isHexDigest(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
