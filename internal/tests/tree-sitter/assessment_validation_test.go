package treesitter_test

import (
	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	"strings"
	"testing"
)

func supportAssessment() *ts.LanguageSupportAssessment {
	pin := strings.Repeat("a", 40)
	return &ts.LanguageSupportAssessment{
		AssessedRevision: pin,
		Claim:            "verified",
		MinVersion:       "1.23",
		MaxVersion:       "1.25",
		Evidence: []ts.LanguageSupportEvidence{{
			Kind:        "documentation",
			Source:      "https://github.com/example/grammar/blob/" + pin + "/README.md",
			Description: "Explicit upstream syntax compatibility claim (synthetic test data).",
		}},
	}
}

func TestLanguageSupportAssessmentValidation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		edit      func(*ts.LanguageSupportAssessment)
		wantError bool
	}{
		{"pinned documentation", func(*ts.LanguageSupportAssessment) {}, false},
		{"unverified range without evidence", func(a *ts.LanguageSupportAssessment) { a.Claim = "unverified"; a.Evidence = nil }, false},
		{"omitted claim defaults unverified", func(a *ts.LanguageSupportAssessment) { a.Claim = ""; a.Evidence = nil }, false},
		{"unverified unpinned documentation", func(a *ts.LanguageSupportAssessment) {
			a.Claim = "unverified"
			a.Evidence[0].Source = "https://example.com/docs"
		}, false},
		{"missing assessment pin", func(a *ts.LanguageSupportAssessment) { a.AssessedRevision = "" }, true},
		{"short assessment pin", func(a *ts.LanguageSupportAssessment) { a.AssessedRevision = "abcdef" }, true},
		{"unknown claim", func(a *ts.LanguageSupportAssessment) { a.Claim = "probably" }, true},
		{"verified without evidence", func(a *ts.LanguageSupportAssessment) { a.Evidence = nil }, true},
		{"verified without range or limitations", func(a *ts.LanguageSupportAssessment) { a.MinVersion = ""; a.MaxVersion = "" }, true},
		{"verified limitation only", func(a *ts.LanguageSupportAssessment) {
			a.MinVersion = ""
			a.MaxVersion = ""
			a.Limitations = []string{"Feature X is unsupported."}
		}, false},
		{"unpinned documentation", func(a *ts.LanguageSupportAssessment) { a.Evidence[0].Source = "https://example.com/docs" }, true},
		{"wrong documentation pin", func(a *ts.LanguageSupportAssessment) {
			a.Evidence[0].Source = "https://example.com/" + strings.Repeat("b", 40) + "/README.md"
		}, true},
		{"pin only in query", func(a *ts.LanguageSupportAssessment) {
			a.Evidence[0].Source = "https://example.com/docs?revision=" + a.AssessedRevision
		}, true},
		{"unknown evidence kind", func(a *ts.LanguageSupportAssessment) { a.Evidence[0].Kind = "guess" }, true},
		{"empty evidence description", func(a *ts.LanguageSupportAssessment) { a.Evidence[0].Description = "" }, true},
		{"fixture evidence", func(a *ts.LanguageSupportAssessment) {
			a.Evidence = []ts.LanguageSupportEvidence{{Kind: "syntax_fixtures", Source: "reports/go-syntax.json", Description: "Selected syntax fixtures, not full conformance.", SHA256: strings.Repeat("b", 64), Result: "passed", TestedVersions: []string{"1.23", "1.25"}}}
		}, false},
		{"fixture without successful result", func(a *ts.LanguageSupportAssessment) {
			a.Evidence = []ts.LanguageSupportEvidence{{Kind: "syntax_fixtures", Source: "report.json", Description: "Syntax tests", SHA256: strings.Repeat("b", 64), TestedVersions: []string{"1.25"}}}
		}, true},
		{"fixture without report digest", func(a *ts.LanguageSupportAssessment) {
			a.Evidence = []ts.LanguageSupportEvidence{{Kind: "syntax_fixtures", Source: "report.json", Description: "Syntax tests", Result: "passed", TestedVersions: []string{"1.25"}}}
		}, true},
		{"fixture without version coverage", func(a *ts.LanguageSupportAssessment) {
			a.Evidence = []ts.LanguageSupportEvidence{{Kind: "syntax_fixtures", Source: "report.json", Description: "Syntax tests", Result: "passed", SHA256: strings.Repeat("b", 64)}}
		}, true},
		{"non-semver language standard", func(a *ts.LanguageSupportAssessment) { a.MinVersion = "C++17"; a.MaxVersion = "C++23" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := supportAssessment()
			tc.edit(a)
			if err := a.Validate(); (err != nil) != tc.wantError {
				t.Fatalf("Validate() error = %v, wantError %v", err, tc.wantError)
			}
		})
	}
}
