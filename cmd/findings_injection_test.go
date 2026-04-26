package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// --- findingsInjectionForCaste tests (RED phase) ---

func TestFindingsInjectionForCaste_ReviewCastes(t *testing.T) {
	tests := []struct {
		caste         string
		wantDomains   []string // substrings that must appear
		wantCLI       string   // must appear
	}{
		{
			caste:       "watcher",
			wantDomains: []string{"testing", "quality"},
			wantCLI:     "review-ledger-write",
		},
		{
			caste:       "chaos",
			wantDomains: []string{"resilience"},
			wantCLI:     "review-ledger-write",
		},
		{
			caste:       "measurer",
			wantDomains: []string{"performance"},
			wantCLI:     "review-ledger-write",
		},
		{
			caste:       "archaeologist",
			wantDomains: []string{"history"},
			wantCLI:     "review-ledger-write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.caste, func(t *testing.T) {
			got := findingsInjectionForCaste(tt.caste)
			if got == "" {
				t.Fatalf("expected non-empty injection for %s, got empty string", tt.caste)
			}
			for _, domain := range tt.wantDomains {
				if !strings.Contains(got, domain) {
					t.Errorf("expected injection for %s to contain %q, got: %s", tt.caste, domain, got)
				}
			}
			if !strings.Contains(got, tt.wantCLI) {
				t.Errorf("expected injection for %s to contain %q, got: %s", tt.caste, tt.wantCLI, got)
			}
			// Verify the agent name is in the injection
			if !strings.Contains(got, "--agent "+tt.caste) {
				t.Errorf("expected injection for %s to contain --agent %s, got: %s", tt.caste, tt.caste, got)
			}
		})
	}
}

func TestFindingsInjectionForCaste_NonReviewCastes(t *testing.T) {
	nonReviewCastes := []string{
		"builder", "gatekeeper", "probe", "tracker", "scout", "oracle", "architect",
		"keeper", "weaver", "sage", "ambassador", "includer", "medic", "chronicler",
	}
	for _, caste := range nonReviewCastes {
		t.Run(caste, func(t *testing.T) {
			got := findingsInjectionForCaste(caste)
			if got != "" {
				t.Errorf("expected empty injection for non-review caste %s, got: %s", caste, got)
			}
		})
	}
}

// --- renderCodexContinueReviewBrief tests (RED phase) ---

func TestContinueReviewBrief_GatekeeperHasFindingsInjection(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Security hardening"}
	spec := codexContinueReviewSpec{
		Caste: "gatekeeper",
		Task:  "Review the phase for security, release, and integrity blockers before advancement.",
	}
	brief := renderCodexContinueReviewBrief("/tmp/test", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, spec)

	if !strings.Contains(brief, "review-ledger-write") {
		t.Error("gatekeeper brief should contain review-ledger-write")
	}
	if !strings.Contains(brief, "security") {
		t.Error("gatekeeper brief should contain security")
	}
	if !strings.Contains(brief, "persist") {
		t.Error("gatekeeper brief should contain persist findings language")
	}
}

func TestContinueReviewBrief_AuditorHasFindingsInjection(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Quality audit"}
	// Use the production spec which includes domain names in the Task text
	var spec codexContinueReviewSpec
	for _, s := range codexContinueReviewSpecs {
		if s.Caste == "auditor" {
			spec = s
			break
		}
	}
	brief := renderCodexContinueReviewBrief("/tmp/test", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, spec)

	if !strings.Contains(brief, "review-ledger-write") {
		t.Error("auditor brief should contain review-ledger-write")
	}

	// Must contain at least two of quality, security, performance
	domainCount := 0
	for _, d := range []string{"quality", "security", "performance"} {
		if strings.Contains(brief, d) {
			domainCount++
		}
	}
	if domainCount < 2 {
		t.Errorf("auditor brief should contain at least 2 domains (quality, security, performance), found %d", domainCount)
	}
}

func TestContinueReviewBrief_ProbeNoFindingsInjection(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Coverage check"}
	spec := codexContinueReviewSpec{
		Caste: "probe",
		Task:  "Probe the verification evidence for missing edge cases.",
	}
	brief := renderCodexContinueReviewBrief("/tmp/test", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, spec)

	if strings.Contains(brief, "review-ledger-write") {
		t.Error("probe brief should NOT contain review-ledger-write")
	}
}

func TestContinueReviewBrief_GatekeeperNotReadonly(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Security check"}
	spec := codexContinueReviewSpec{
		Caste: "gatekeeper",
		Task:  "Review the phase for security, release, and integrity blockers before advancement.",
	}
	brief := renderCodexContinueReviewBrief("/tmp/test", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, spec)

	if strings.Contains(brief, "read-only review") {
		t.Error("gatekeeper brief should NOT say 'read-only review'")
	}
	if !strings.Contains(brief, "persist findings") {
		t.Error("gatekeeper brief should say 'persist findings'")
	}
}

func TestContinueReviewBrief_ProbeReadonly(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Coverage check"}
	spec := codexContinueReviewSpec{
		Caste: "probe",
		Task:  "Probe the verification evidence for missing edge cases.",
	}
	brief := renderCodexContinueReviewBrief("/tmp/test", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, spec)

	if !strings.Contains(brief, "read-only review") {
		t.Error("probe brief should say 'read-only review'")
	}
}
