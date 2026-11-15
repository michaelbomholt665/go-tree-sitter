package build_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	tsbuild "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

func documentedSupport(pin string) *ts.LanguageSupportAssessment {
	return &ts.LanguageSupportAssessment{
		AssessedRevision: pin, Claim: "verified", MinVersion: "3.10", MaxVersion: "3.13",
		Limitations: []string{"Synthetic test claim, not a real Python support assessment."},
		Evidence: []ts.LanguageSupportEvidence{{
			Kind: "documentation", Source: "https://example.invalid/python/blob/" + pin + "/README.md",
			Description: "Explicit syntax support claim in pinned upstream documentation.",
		}},
	}
}

func TestManifestLanguageSupport(t *testing.T) {
	pin := "0123456789012345678901234567890123456789"
	for _, tc := range []struct {
		name       string
		assessment *ts.LanguageSupportAssessment
		wantClaim  string
		wantReason string
	}{
		{"unknown", nil, "unverified", "no_assessment"},
		{"matching documented assessment", documentedSupport(pin), "verified", ""},
		{"matching case-insensitive commit", documentedSupport(strings.Repeat("A", 40)), "verified", ""},
		{"stale documented assessment", documentedSupport(strings.Repeat("b", 40)), "unverified", "revision_mismatch"},
		{"unverified range", &ts.LanguageSupportAssessment{AssessedRevision: pin, Claim: "unverified", MinVersion: "3.10"}, "unverified", "assessment_unverified"},
		{"omitted claim", &ts.LanguageSupportAssessment{AssessedRevision: pin, MinVersion: "3.10"}, "unverified", "assessment_unverified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base, dir, measured := validTestManifest(t)
			path := filepath.Join(dir, base.Binaries[0].Filename)
			if tc.name == "matching case-insensitive commit" {
				base.BuildProvenance[0].SourceRevision = strings.Repeat("a", 40)
			}
			lang := ts.Language{Name: "python", Grammar: "python", Version: "v1.0.0", Revision: base.BuildProvenance[0].SourceRevision, LanguageSupport: tc.assessment}
			manifest, err := tsbuild.GenerateManifest(lang, []string{path}, measured, map[string]tsbuild.BinaryProvenance{path: base.BuildProvenance[0]}, base.Artifacts)
			if err != nil {
				t.Fatal(err)
			}
			info := manifest.LanguageSupport
			if info == nil || info.Claim != tc.wantClaim || info.Reason != tc.wantReason {
				t.Fatalf("language support = %+v, want %s/%s", info, tc.wantClaim, tc.wantReason)
			}
			output := filepath.Join(dir, "manifest.json")
			if err := manifest.WriteToFile(output); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			var decoded tsbuild.ManifestData
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.LanguageSupport.Claim != tc.wantClaim || decoded.LanguageSupport.Reason != tc.wantReason {
				t.Fatalf("language support did not survive manifest round-trip: %+v", decoded.LanguageSupport)
			}
			if tc.assessment != nil {
				if decoded.LanguageSupport.Assessment.AssessedRevision != tc.assessment.AssessedRevision || decoded.LanguageSupport.Assessment.MaxVersion != tc.assessment.MaxVersion {
					t.Fatal("historical assessment was lost or repinned")
				}
				if tc.name == "omitted claim" && tc.assessment.Claim != "" {
					t.Fatal("manifest generation mutated configuration")
				}
				info.Assessment.MinVersion = "changed"
				if tc.assessment.MinVersion == "changed" {
					t.Fatal("manifest assessment aliases configuration")
				}
			}
		})
	}
}

func TestManifestLanguageSupportRejectsForgedVerifiedClaim(t *testing.T) {
	for _, stale := range []bool{false, true} {
		manifest, dir, measured := validTestManifest(t)
		manifest.LanguageSupport.Claim = "verified"
		manifest.LanguageSupport.Reason = ""
		if stale {
			manifest.LanguageSupport.Assessment = documentedSupport(strings.Repeat("b", 40))
		}
		if err := tsbuild.ValidateManifest(manifest, dir, measured); err == nil || !strings.Contains(err.Error(), "language_support") {
			t.Fatalf("expected rejection of forged claim (stale=%v), got %v", stale, err)
		}
	}
}

func TestManifestLanguageSupportUsesEveryBinaryRevision(t *testing.T) {
	manifest, dir, measured := validTestManifest(t)
	first := manifest.BuildProvenance[0]
	path := filepath.Join(dir, manifest.Binaries[0].Filename)
	otherPath := filepath.Join(dir, "python-v1.0.0-windows-amd64.dll")
	if err := os.WriteFile(otherPath, []byte("other binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	other := first
	other.Filename = filepath.Base(otherPath)
	other.SourceRevision = strings.Repeat("b", 40)
	measured[otherPath] = 15
	lang := ts.Language{Name: "python", Grammar: "python", Version: "v1.0.0", Revision: first.SourceRevision, LanguageSupport: documentedSupport(first.SourceRevision)}
	generated, err := tsbuild.GenerateManifest(lang, []string{path, otherPath}, measured, map[string]tsbuild.BinaryProvenance{path: first, otherPath: other}, manifest.Artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if generated.LanguageSupport.Claim != "unverified" || generated.LanguageSupport.Reason != "revision_mismatch" {
		t.Fatalf("a foreign binary's stale assessment was accepted: %+v", generated.LanguageSupport)
	}
	if err := tsbuild.ValidateManifest(generated, dir, measured); err != nil {
		t.Fatal(err)
	}
}

func TestManifestLanguageSupportLegacyV2(t *testing.T) {
	manifest, dir, measured := validTestManifest(t)
	manifest.LanguageSupport = nil
	if err := tsbuild.ValidateManifest(manifest, dir, measured); err != nil {
		t.Fatalf("older schema-v2 manifests must remain readable as unverified: %v", err)
	}
}

func TestMoverPublishesHistoricalLanguageSupport(t *testing.T) {
	dir := t.TempDir()
	buildDir, _ := setupLanguageBuildDir(t, dir, "python", "v0.25.0")
	assessment := documentedSupport(strings.Repeat("b", 40))
	cfg := &ts.Config{
		BuildDir: buildDir,
		Languages: []ts.Language{{
			Name: "python", Grammar: "python", Constructor: "tree_sitter_python", Version: "v0.25.0",
			Repository: "https://example.invalid/grammar", Revision: "0123456789012345678901234567890123456789",
			LanguageSupport: assessment,
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}
	mover := tsbuild.NewMoverWithValidator(tsbuild.NewCleaner(), acceptingValidator{}, ioDiscard{}, ioDiscard{})
	prepareArtifactConfig(cfg)
	if err := mover.Move(context.Background(), cfg, ts.MoveRequest{Language: "python", Mode: ts.MoveModeBoth, GenerateManifest: true}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "out", "python", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var published tsbuild.ManifestData
	if err := json.Unmarshal(raw, &published); err != nil {
		t.Fatal(err)
	}
	info := published.LanguageSupport
	if info == nil || info.Claim != "unverified" || info.Reason != "revision_mismatch" || info.Assessment.Claim != "verified" || info.Assessment.AssessedRevision != assessment.AssessedRevision {
		t.Fatalf("publication lost the historical assessment or verified stale bounds: %+v", info)
	}
}
