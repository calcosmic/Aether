package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// Stage 1 / Work Package 1 (owner rulings D6, D7): an honest "nothing needed
// changing" result is a first-class success with mandatory evidence, and a
// quota/rate-limit interruption is a recognized resumable terminal outcome.
// Spec §2.4: "The system must never force a fake edit simply to satisfy an
// accounting schema."

func TestNoChangeStatusVocabulary(t *testing.T) {
	for raw, want := range map[string]string{
		"no_change":           "completed_no_change",
		"no-change":           "completed_no_change",
		"nochange":            "completed_no_change",
		"unchanged":           "completed_no_change",
		"completed_no_change": "completed_no_change",
		"verified_existing":   "completed_no_change",
		"already_complete":    "completed_no_change",
		"already_correct":     "completed_no_change",
		"interrupted":         "interrupted",
		"suspended_quota":     "interrupted",
		"rate_limit":          "interrupted",
		"rate_limited":        "interrupted",
		"completed":           "completed",
		"code_written":        "completed",
	} {
		if got := normalizeExternalBuildStatus(raw); got != want {
			t.Errorf("normalizeExternalBuildStatus(%q) = %q, want %q", raw, got, want)
		}
	}

	for _, status := range []string{"completed_no_change", "interrupted"} {
		if !isTerminalExternalBuildStatus(status) {
			t.Errorf("%q must be a terminal status", status)
		}
	}
	if !isSuccessfulExternalBuildStatus("completed_no_change") {
		t.Error("completed_no_change must count as success")
	}
	if isSuccessfulExternalBuildStatus("interrupted") {
		t.Error("interrupted is terminal but must NOT count as success")
	}
}

func noChangeResultFixture() codexExternalBuildWorkerResult {
	return codexExternalBuildWorkerResult{
		Name:    "Mason-67",
		Caste:   "builder",
		Status:  "verified_existing",
		Summary: "The guide already satisfies every acceptance criterion; no edit needed.",
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "pass",
			CommandsRun:        []string{"go test ./cmd/ -run TestGuide -count=1 (ok)"},
		},
	}
}

func noChangeManifestFixture() codexBuildManifest {
	return codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}
}

// TestNoChangeResultWithEvidenceIsAccepted: audit §8.8 case "already-correct
// code returns verified_existing with reproducible evidence and no fake file
// change".
func TestNoChangeResultWithEvidenceIsAccepted(t *testing.T) {
	dispatches, violations, err := mergeExternalBuildResults(noChangeManifestFixture(), []codexExternalBuildWorkerResult{noChangeResultFixture()})
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("an evidenced no-change result must merge cleanly, got violations: %+v", violations)
	}
	if dispatches[0].Status != "completed_no_change" {
		t.Errorf("status = %q, want completed_no_change", dispatches[0].Status)
	}
	if dispatches[0].Disposition != "verified_existing" {
		t.Errorf("disposition = %q, want verified_existing (raw status carried it)", dispatches[0].Disposition)
	}
}

// TestNoChangeResultWithoutEvidenceIsRejected: no free pass — a no-change
// completion that cannot say what it verified is a named contract violation,
// not a success.
func TestNoChangeResultWithoutEvidenceIsRejected(t *testing.T) {
	result := noChangeResultFixture()
	result.Handoff.CommandsRun = nil

	_, violations, err := mergeExternalBuildResults(noChangeManifestFixture(), []codexExternalBuildWorkerResult{result})
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.Rule == violationRuleNoChangeEvidence {
			found = true
			if !strings.Contains(v.Message, "commands_run") {
				t.Errorf("violation message should name the missing evidence, got %q", v.Message)
			}
		}
	}
	if !found {
		t.Fatalf("expected a %s violation, got %+v", violationRuleNoChangeEvidence, violations)
	}
}

// TestInterruptedIsTerminalButNotSuccess: ruling D7 — a rate-limit stop is a
// resumable interruption, never a code failure and never a completion.
func TestInterruptedIsTerminalButNotSuccess(t *testing.T) {
	result := codexExternalBuildWorkerResult{
		Name:    "Mason-67",
		Caste:   "builder",
		Status:  "rate_limited",
		Summary: "Claude allowance exhausted mid-slice; durable checkpoint written.",
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "not_run",
			KnownFailures:      []string{"5h quota exhausted"},
		},
	}

	dispatches, violations, err := mergeExternalBuildResults(noChangeManifestFixture(), []codexExternalBuildWorkerResult{result})
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	for _, v := range violations {
		if v.Rule == violationRuleStatusTerminal {
			t.Fatalf("interrupted must be recognized as terminal, got %+v", v)
		}
	}
	if dispatches[0].Status != "interrupted" {
		t.Errorf("status = %q, want interrupted", dispatches[0].Status)
	}
}

// TestProvenanceAcceptsEvidencedNoChangeBuild: a build whose only honest
// outcome was "everything already correct" passes provenance WITH evidence
// and fails it without — the phantom-build guard keeps its teeth.
func TestProvenanceAcceptsEvidencedNoChangeBuild(t *testing.T) {
	if err := validateBuildProvenance([]codexExternalBuildWorkerResult{noChangeResultFixture()}); err != nil {
		t.Fatalf("evidenced no-change build should satisfy provenance, got %v", err)
	}

	bare := noChangeResultFixture()
	bare.Handoff.CommandsRun = nil
	if err := validateBuildProvenance([]codexExternalBuildWorkerResult{bare}); err == nil {
		t.Fatal("evidence-free no-change build must still fail provenance (phantom-build guard)")
	}
}

// TestContinueProvenanceAcceptsNoChangeDispatch: the continue-side check must
// not call an honest no-change dispatch a phantom build for having no file
// outputs — its evidence is the verification it ran, validated at finalize.
func TestContinueProvenanceAcceptsNoChangeDispatch(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Name: "Mason-67", Status: "completed_no_change"},
	}
	if err := traceContinueProvenance(dispatches); err != nil {
		t.Fatalf("no-change dispatch should satisfy continue provenance, got %v", err)
	}
}

// The worker-facing half (the response contract offering completed_no_change)
// is asserted in pkg/codex/handoff_guidance_test.go, where the renderer lives.
