package cmd

import (
	"strings"
	"testing"
)

// Phase 198 plan 06 (SHOW-02/D-11) -- the closing summary currently says
// "Verification: 3 passed, 1 skipped" and "Gates: 5/6 passed" and stops. This
// file proves the named checks, named gates, their fix hints, and the
// evidence behind every claimed requirement all reach the screen -- on the
// typed in-process shape AND the JSON-round-tripped completion-file shape,
// following the dual-type rendering precedent renderContinueWorkerFlowValue
// already established (198-PATTERNS.md).

// ---- Task 1: named checks and named gates ----

// TestVerificationDetailNamesEveryCheck proves four checks in produces four
// named lines out, with the skipped and failed rows asserted separately, on
// both the typed and JSON-round-tripped shapes.
func TestVerificationDetailNamesEveryCheck(t *testing.T) {
	report := codexContinueVerificationReport{
		Steps: []codexVerificationStep{
			{Name: "build", Passed: true, Duration: 1.2},
			{Name: "types", Passed: true, Duration: 0.8},
			{Name: "lint", Skipped: true, Summary: "no lint command configured for this project"},
			{Name: "tests", Passed: false, Summary: "2 of 12 tests failed", Command: "npm test", Duration: 4.3},
		},
	}

	var typedBuilder strings.Builder
	renderContinueVerificationDetail(&typedBuilder, report)
	typedOutput := typedBuilder.String()

	for _, want := range []string{
		"Build ✓ (1.2s)",
		"Types ✓ (0.8s)",
		"Lint — skipped: no lint command configured for this project",
		"Tests ✗ (4.3s) — 2 of 12 tests failed",
		"└── npm test",
	} {
		if !strings.Contains(typedOutput, want) {
			t.Errorf("typed render missing %q, got:\n%s", want, typedOutput)
		}
	}
	// A passed check never carries the failed check's nested command line.
	if strings.Contains(typedOutput, "Build ✓ (1.2s)\n      └──") {
		t.Errorf("a passed check must not render a nested command line, got:\n%s", typedOutput)
	}

	asMap := roundTripToMap(t, map[string]interface{}{"verification": report})
	var mapBuilder strings.Builder
	renderContinueVerificationDetail(&mapBuilder, asMap["verification"])
	mapOutput := mapBuilder.String()

	if typedOutput != mapOutput {
		t.Fatalf("round-tripped render diverged from typed render:\n%s", firstDiffLine(typedOutput, mapOutput))
	}
}

// TestGateDetailNamesEveryGateAndItsFixHint proves each gate renders its
// plain-English name and outcome, and a failing gate's fix hint appears on
// the nested detail line, on both shapes.
func TestGateDetailNamesEveryGateAndItsFixHint(t *testing.T) {
	report := codexContinueGateReport{
		Checks: []gateCheck{
			{Name: "manifest_present", Passed: true},
			{Name: "verification_steps_passed", Passed: false, FixHint: "fix the failing build/test check and run aether continue"},
		},
	}

	var typedBuilder strings.Builder
	renderContinueGateDetail(&typedBuilder, report)
	typedOutput := typedBuilder.String()

	for _, want := range []string{
		"✓ the build's own plan file is on disk",
		"✗ the build/test checks passed",
		"└── fix the failing build/test check and run aether continue",
	} {
		if !strings.Contains(typedOutput, want) {
			t.Errorf("typed render missing %q, got:\n%s", want, typedOutput)
		}
	}
	if strings.Contains(typedOutput, "manifest_present") || strings.Contains(typedOutput, "verification_steps_passed") {
		t.Errorf("internal gate keys leaked into the rendered output:\n%s", typedOutput)
	}

	asMap := roundTripToMap(t, map[string]interface{}{"gates": report})
	var mapBuilder strings.Builder
	renderContinueGateDetail(&mapBuilder, asMap["gates"])
	mapOutput := mapBuilder.String()

	if typedOutput != mapOutput {
		t.Fatalf("round-tripped render diverged from typed render:\n%s", firstDiffLine(typedOutput, mapOutput))
	}
}

// TestNestedDetailCapReportsWhatItOmitted proves an over-cap input reports
// an omitted count equal to the arithmetic remainder computed independently
// here, for both verification checks and gates.
func TestNestedDetailCapReportsWhatItOmitted(t *testing.T) {
	t.Run("verification checks", func(t *testing.T) {
		total := continueDetailCap + 4
		steps := make([]codexVerificationStep, 0, total)
		for i := 0; i < total; i++ {
			steps = append(steps, codexVerificationStep{Name: "tests", Passed: true})
		}
		report := codexContinueVerificationReport{Steps: steps}

		var b strings.Builder
		renderContinueVerificationDetail(&b, report)
		output := b.String()

		wantOmitted := total - continueDetailCap
		want := "(+4 more checks)"
		if wantOmitted != 4 {
			t.Fatalf("test setup error: expected omitted count 4, computed %d", wantOmitted)
		}
		if !strings.Contains(output, want) {
			t.Errorf("expected honest omitted count %q, got:\n%s", want, output)
		}
	})

	t.Run("gates", func(t *testing.T) {
		total := continueDetailCap + 3
		checks := make([]gateCheck, 0, total)
		for i := 0; i < total; i++ {
			checks = append(checks, gateCheck{Name: "manifest_present", Passed: true})
		}
		report := codexContinueGateReport{Checks: checks}

		var b strings.Builder
		renderContinueGateDetail(&b, report)
		output := b.String()

		wantOmitted := total - continueDetailCap
		want := "(+3 more gates)"
		if wantOmitted != 3 {
			t.Fatalf("test setup error: expected omitted count 3, computed %d", wantOmitted)
		}
		if !strings.Contains(output, want) {
			t.Errorf("expected honest omitted count %q, got:\n%s", want, output)
		}
	})
}

// ---- Task 2: evidence lines and specialist findings ----

// TestEvidenceLineNeverAppearsWithoutItsProof proves no satisfied mark (✓)
// appears on any row lacking evidence, and every satisfied row's line
// contains its own evidence text -- across satisfied, unproven, blocked, and
// awaiting-owner rows.
func TestEvidenceLineNeverAppearsWithoutItsProof(t *testing.T) {
	tests := []struct {
		name      string
		criterion codexCriterionVerification
		wantTick  bool
		wantText  []string
	}{
		{
			name: "satisfied",
			criterion: codexCriterionVerification{
				Criterion: "Login works",
				Evidence:  []string{"3 tests passed", "auth.go present"},
				Passed:    true,
			},
			wantTick: true,
			wantText: []string{"✓ Login works — proved by: 3 tests passed, auth.go present"},
		},
		{
			name: "unproven",
			criterion: codexCriterionVerification{
				Criterion: "Export works",
				Passed:    true,
			},
			wantTick: false,
			wantText: []string{"Export works", "unproven"},
		},
		{
			name: "blocked",
			criterion: codexCriterionVerification{
				Criterion:      "Payment succeeds",
				Passed:         false,
				BlockingIssues: []string{"payment API returned 500"},
			},
			wantTick: false,
			wantText: []string{"Payment succeeds", "payment API returned 500"},
		},
		{
			name: "awaiting owner",
			criterion: codexCriterionVerification{
				Criterion: "Accessibility reviewed",
				Passed:    true,
				State:     criterionStateNeedsOwnerConfirmation,
			},
			wantTick: false,
			wantText: []string{"Accessibility reviewed", "awaiting your confirmation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := codexContinueVerificationReport{Criteria: []codexCriterionVerification{tt.criterion}}

			var b strings.Builder
			renderCriterionEvidenceLines(&b, report)
			output := b.String()

			tickLine := "✓ " + tt.criterion.Criterion
			if strings.Contains(output, tickLine) != tt.wantTick {
				t.Errorf("satisfied-mark presence mismatch: wantTick=%v, output:\n%s", tt.wantTick, output)
			}
			for _, want := range tt.wantText {
				if !strings.Contains(output, want) {
					t.Errorf("missing %q in output:\n%s", want, output)
				}
			}

			// Dual-type: the JSON-round-tripped shape renders identically.
			asMap := roundTripToMap(t, map[string]interface{}{"verification": report})
			var mapBuilder strings.Builder
			renderCriterionEvidenceLines(&mapBuilder, asMap["verification"])
			if mapBuilder.String() != output {
				t.Fatalf("round-tripped render diverged from typed render:\n%s", firstDiffLine(output, mapBuilder.String()))
			}
		})
	}
}

// TestSpecialistFindingsGetTheirOwnBlock proves reviewer findings,
// recommendations, weak spots, edge cases, and blockers render as their own
// headed block, visible without reading each worker's nested detail line --
// and a worker with nothing to report is silently excluded.
func TestSpecialistFindingsGetTheirOwnBlock(t *testing.T) {
	flow := []codexContinueWorkerFlowStep{
		{
			Stage: "review", Caste: "watcher", Name: "Keen-12", Status: "completed",
			Findings:        []codexReviewFinding{{Severity: "high", Title: "race condition in dispatch loop"}},
			Recommendations: []string{"add a regression test for the race"},
			WeakSpots:       []string{"no timeout on the retry loop"},
			EdgeCases:       []string{"concurrent dispatch from two waves"},
			Blockers:        []string{"circuit breaker not reset between attempts"},
		},
		{
			Stage: "review", Caste: "auditor", Name: "Ledger-3", Status: "completed",
			// No findings at all -- must not appear in the block.
		},
	}

	var typedBuilder strings.Builder
	renderSpecialistFindingBlocks(&typedBuilder, flow)
	typedOutput := typedBuilder.String()

	for _, want := range []string{
		"🔍 Specialist Findings",
		"Keen-12",
		"found: high: race condition in dispatch loop",
		"recommends: add a regression test for the race",
		"weak spot: no timeout on the retry loop",
		"edge case: concurrent dispatch from two waves",
		"blocker: circuit breaker not reset between attempts",
	} {
		if !strings.Contains(typedOutput, want) {
			t.Errorf("missing %q in output:\n%s", want, typedOutput)
		}
	}
	if strings.Contains(typedOutput, "Ledger-3") {
		t.Errorf("a worker with nothing to report must not appear in the specialist findings block, got:\n%s", typedOutput)
	}

	asMap := roundTripToMap(t, map[string]interface{}{"worker_flow": flow})
	var mapBuilder strings.Builder
	renderSpecialistFindingBlocks(&mapBuilder, asMap["worker_flow"])
	mapOutput := mapBuilder.String()

	if typedOutput != mapOutput {
		t.Fatalf("round-tripped render diverged from typed render:\n%s", firstDiffLine(typedOutput, mapOutput))
	}
}
