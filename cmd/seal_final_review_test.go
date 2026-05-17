package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestSealReviewFindingBlockingIssuesBlocksHighReleaseRisks(t *testing.T) {
	findings := []sealFinalReviewFinding{
		{
			Domain:      "security",
			Severity:    "HIGH",
			Agent:       "gatekeeper",
			AgentName:   "Gate-1",
			Description: "release token can leak",
		},
		{
			Domain:      "performance",
			Severity:    "HIGH",
			Agent:       "measurer",
			AgentName:   "Measure-1",
			Description: "slow report generation",
		},
		{
			Domain:      "quality",
			Severity:    "HIGH",
			Agent:       "auditor",
			AgentName:   "Audit-1",
			Description: "release evidence overclaims readiness",
		},
	}

	blockers := sealReviewFindingBlockingIssues(findings)

	if len(blockers) != 2 {
		t.Fatalf("expected two high release-risk blockers, got %d: %v", len(blockers), blockers)
	}
	if !strings.Contains(blockers[0], "Gate-1 reported HIGH security finding") {
		t.Fatalf("blocker = %q, want high security finding detail", blockers[0])
	}
	if !strings.Contains(strings.Join(blockers, "\n"), "Audit-1 reported HIGH quality finding") {
		t.Fatalf("blockers = %q, want high quality finding detail", blockers)
	}
}

func TestSealReviewFindingBlockingIssuesStillHonorsExplicitBlocking(t *testing.T) {
	findings := []sealFinalReviewFinding{
		{
			Domain:      "testing",
			Severity:    "MEDIUM",
			Agent:       "probe",
			AgentName:   "Probe-1",
			Description: "smoke test skipped",
			Blocking:    true,
		},
	}

	blockers := sealReviewFindingBlockingIssues(findings)

	if len(blockers) != 1 {
		t.Fatalf("expected explicit blocking finding, got %d: %v", len(blockers), blockers)
	}
	if !strings.Contains(blockers[0], "Probe-1 reported MEDIUM testing finding") {
		t.Fatalf("blocker = %q, want explicit blocking detail", blockers[0])
	}
}

func TestSealFinalReviewBriefDocumentsHighSecurityQualityBlockers(t *testing.T) {
	goal := "release hardening"
	brief := renderSealFinalReviewBrief("/tmp/repo", colony.ColonyState{
		Goal: &goal,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Final", Status: colony.PhaseCompleted},
		}},
	}, colony.Phase{ID: 1, Name: "Final"}, codexContinueReviewSpec{
		Caste: "auditor",
		Task:  "Review release quality.",
	})

	for _, want := range []string{"HIGH security", "HIGH quality", "CRITICAL findings always stop sealing"} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief missing %q:\n%s", want, brief)
		}
	}
	if strings.Contains(brief, "CRITICAL severity only") {
		t.Fatalf("brief still contains old CRITICAL-only blocker contract:\n%s", brief)
	}
}
