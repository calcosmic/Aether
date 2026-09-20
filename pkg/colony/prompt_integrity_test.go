package colony

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptIntegrityClassifyPromptSource(t *testing.T) {
	cases := []struct {
		source string
		want   PromptTrustClass
	}{
		{source: "/tmp/repo/.aether/data/COLONY_STATE.json", want: PromptTrustAuthorized},
		{source: "/tmp/home/.aether/QUEEN.md", want: PromptTrustTrusted},
		{source: "/tmp/home/.aether/hive/wisdom.json", want: PromptTrustAuthorized},
		{source: "/tmp/repo/README.md", want: PromptTrustUnknown},
		{source: "inline:task-brief", want: PromptTrustUnknown},
	}

	for _, tc := range cases {
		if got := ClassifyPromptSource(tc.source); got != tc.want {
			t.Errorf("ClassifyPromptSource(%q) = %q, want %q", tc.source, got, tc.want)
		}
	}
}

func TestPromptIntegrityDetectsRepoFixtureInstruction(t *testing.T) {
	fixture := filepath.Join("..", "..", "cmd", "testdata", "prompt-integrity-fixtures", "repo-instruction", "README.md")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	assessment := AssessPromptSource(filepath.ToSlash(filepath.Clean(fixture)), string(data))
	if assessment.BaseTrustClass != PromptTrustUnknown {
		t.Fatalf("base trust = %q, want %q", assessment.BaseTrustClass, PromptTrustUnknown)
	}
	if assessment.TrustClass != PromptTrustSuspicious {
		t.Fatalf("trust class = %q, want %q", assessment.TrustClass, PromptTrustSuspicious)
	}
	if assessment.Action != PromptIntegrityActionBlock {
		t.Fatalf("action = %q, want %q", assessment.Action, PromptIntegrityActionBlock)
	}
	if len(assessment.Findings) == 0 {
		t.Fatal("expected prompt integrity findings for suspicious fixture")
	}
	if !strings.Contains(strings.ToLower(assessment.Findings[0].Message), "prompt injection") {
		t.Fatalf("unexpected finding message: %q", assessment.Findings[0].Message)
	}
}

// TestPromptIntegritySecretsPathRuleIsPathShaped locks the Phase 205 wave-1
// post-merge regression: secretsPathRuleSpecs must reject a secrets FILE PATH
// (the 2026-09-14 field report's `cp ../dashboard/.env.local .` lesson, and a
// bare `cp .env.local .`) while leaving ordinary guidance that merely mentions
// .env files untouched. The first version of the rule matched any `.env`
// token, which refused suggest-analyze's own built-in REDIRECT, "never commit
// secrets or .env files to version control", and broke
// TestSuggestAnalyze_ShowsInactivePheromoneSuggestions.
func TestPromptIntegritySecretsPathRuleIsPathShaped(t *testing.T) {
	rejected := []string{
		"cd /private/tmp/claude/cosmic-verify-appsurf/dashboard && cp ../dashboard/.env.local . 2>/dev/null; npm install",
		"cp .env.local .",
		"cat ~/.ssh/id_rsa",
		"source ./config/secrets.json",
	}
	for _, content := range rejected {
		if !hasPromptIntegrityFindingKind(DetectPromptIntegrityFindings(content), "secrets_path") {
			t.Errorf("expected a secrets_path finding for %q", content)
		}
	}

	allowed := []string{
		"never commit secrets or .env files to version control",
		"keep credentials out of the repo; use environment variables instead",
		"head over to the .env section of the README before deploying",
	}
	for _, content := range allowed {
		if hasPromptIntegrityFindingKind(DetectPromptIntegrityFindings(content), "secrets_path") {
			t.Errorf("did not expect a secrets_path finding for %q", content)
		}
	}

	// The built-in suggestion must survive the shared sanitizer end to end,
	// because that is the exact call suggest-analyze makes.
	if _, err := SanitizeSignalContent("never commit secrets or .env files to version control"); err != nil {
		t.Fatalf("built-in suggestion must survive SanitizeSignalContent: %v", err)
	}
}

func hasPromptIntegrityFindingKind(findings []PromptIntegrityFinding, kind string) bool {
	for _, f := range findings {
		if f.Kind == kind {
			return true
		}
	}
	return false
}
