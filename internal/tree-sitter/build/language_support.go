package build

import (
	"fmt"
	"slices"
	"strings"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

// ManifestLanguageSupport separates the current binary's claim from a possibly
// historical assessment. Bounds in a stale assessment do not describe this binary.
type ManifestLanguageSupport struct {
	Claim      string                        `json:"claim"`
	Reason     string                        `json:"reason,omitempty"`
	Assessment *ts.LanguageSupportAssessment `json:"assessment,omitempty"`
}

func manifestLanguageSupport(assessment *ts.LanguageSupportAssessment, provenance []BinaryProvenance) (*ManifestLanguageSupport, error) {
	info := &ManifestLanguageSupport{Claim: "unverified", Reason: "no_assessment"}
	if assessment == nil {
		return info, nil
	}
	if err := assessment.Validate(); err != nil {
		return nil, fmt.Errorf("language_support: %w", err)
	}
	// Snapshot without normalizing or moving the assessment's pin in configuration.
	copy := *assessment
	copy.Limitations = slices.Clone(assessment.Limitations)
	copy.Evidence = slices.Clone(assessment.Evidence)
	for i := range copy.Evidence {
		copy.Evidence[i].TestedVersions = slices.Clone(copy.Evidence[i].TestedVersions)
	}
	if copy.Claim == "" {
		copy.Claim = "unverified"
	}
	info.Assessment = &copy
	info.Reason = "revision_mismatch"
	if len(provenance) == 0 {
		return info, nil
	}
	for _, binary := range provenance {
		if !strings.EqualFold(binary.SourceRevision, assessment.AssessedRevision) {
			return info, nil
		}
	}
	info.Claim = copy.Claim
	info.Reason = "assessment_unverified"
	if info.Claim == "verified" {
		info.Reason = ""
	}
	return info, nil
}
