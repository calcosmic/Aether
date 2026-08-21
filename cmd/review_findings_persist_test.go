package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// WS8c regression locks (Pocket-Chopper field report): auditor and
// gatekeeper have no Bash tool by explicit design, yet their continue-review
// briefs instructed them to persist findings via `aether review-ledger-write`
// — an unsatisfiable task they answered by self-reporting blocked, which
// then blocked phase advancement. The runtime now persists the findings the
// workers RETURN, and no review brief may instruct a CLI command.

func TestContinuePersistsReviewFindingsInProcess(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	steps := []codexContinueWorkerFlowStep{
		{
			Stage: "review",
			Caste: "gatekeeper",
			Name:  "Sentinel-7",
			Findings: []codexReviewFinding{
				{
					// Domain deliberately empty: gatekeeper maps to exactly
					// one review domain, so the runtime fills it in.
					Severity:    "HIGH",
					File:        "cmd/api.go",
					Line:        42,
					Category:    "secrets",
					Description: "API key committed in source",
					Suggestion:  "move the key to an environment variable",
				},
			},
		},
		{
			Stage: "review",
			Caste: "auditor",
			Name:  "Ledger-3",
			Findings: []codexReviewFinding{
				{
					Domain:      "quality",
					Severity:    "MEDIUM",
					File:        "cmd/api.go",
					Line:        10,
					Category:    "completeness",
					Description: "task claims a CSV export but only JSON ships",
					Suggestion:  "implement the CSV branch or narrow the task claim",
				},
				{
					Domain:      "not-a-domain",
					Severity:    "LOW",
					Description: "finding with an invalid domain must be skipped with a note, not dropped silently",
				},
			},
		},
	}

	persisted, notes := persistReviewFindingsToLedgers(3, "Harden the API", steps)
	if persisted != 2 {
		t.Fatalf("persisted = %d, want 2 (notes: %v)", persisted, notes)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "no valid review domain") {
		t.Fatalf("invalid-domain finding did not produce a skip note: %v", notes)
	}

	var security colony.ReviewLedgerFile
	if err := store.LoadJSON("reviews/security/ledger.json", &security); err != nil {
		t.Fatalf("security ledger not written: %v", err)
	}
	if len(security.Entries) != 1 {
		t.Fatalf("security ledger entries = %d, want 1", len(security.Entries))
	}
	entry := security.Entries[0]
	if entry.Agent != "gatekeeper" || entry.AgentName != "Sentinel-7" || entry.Phase != 3 {
		t.Fatalf("security entry attribution wrong: %+v", entry)
	}
	if entry.Suggestion == "" {
		t.Fatalf("suggestion was lost on the in-process persist path: %+v", entry)
	}

	var quality colony.ReviewLedgerFile
	if err := store.LoadJSON("reviews/quality/ledger.json", &quality); err != nil {
		t.Fatalf("quality ledger not written: %v", err)
	}
	if len(quality.Entries) != 1 {
		t.Fatalf("quality ledger entries = %d, want 1", len(quality.Entries))
	}
}

// TestReviewSpecsDoNotInstructBashlessCastes sweeps EVERY continue-review
// task brief for CLI persistence instructions. The runtime persists findings
// itself; a review spec that tells any caste to run `aether
// review-ledger-write` recreates the unsatisfiable-brief deadlock for the
// Bash-less castes and is redundant for the rest.
func TestReviewSpecsDoNotInstructBashlessCastes(t *testing.T) {
	for _, spec := range codexContinueReviewSpecs {
		if strings.Contains(spec.Task, "review-ledger-write") {
			t.Fatalf("continue review spec for %s instructs review-ledger-write; the runtime persists findings in-process", spec.Caste)
		}
	}
	for _, caste := range []string{"gatekeeper", "auditor", "probe", "measurer", "chaos", "includer", "keeper", "sage", "medic", "fixer"} {
		spec, ok := continueReviewSpecForCaste(caste)
		if !ok {
			continue
		}
		if strings.Contains(spec.Task, "review-ledger-write") {
			t.Fatalf("continue review spec for %s instructs review-ledger-write; the runtime persists findings in-process", caste)
		}
	}

	// The rendered continue-review brief for the two Bash-less castes must
	// never instruct a CLI command either — that is where the field failure
	// actually originated.
	for _, caste := range []string{"gatekeeper", "auditor"} {
		spec, ok := continueReviewSpecForCaste(caste)
		if !ok {
			t.Fatalf("no continue review spec for %s", caste)
		}
		brief := renderCodexContinueReviewBrief(t.TempDir(), colony.Phase{ID: 1, Name: "phase"}, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, spec)
		if strings.Contains(brief, "review-ledger-write") {
			t.Fatalf("%s continue brief instructs a CLI command the caste cannot run:\n%s", caste, brief)
		}
		if !strings.Contains(brief, "findings") {
			t.Fatalf("%s continue brief no longer tells the worker how findings reach the ledger:\n%s", caste, brief)
		}
	}
}
