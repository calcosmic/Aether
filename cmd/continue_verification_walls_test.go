package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestUnresolvableCommandSkipped locks in the rule that a verification command
// which cannot run in this repository is a Skipped check, not a Failed one.
// Before this rule, the language-fallback resolver fabricated commands like
// `npm run lint` for projects with no lint script, the command exited non-zero,
// and phase advancement hard-blocked with no remedy the user could see.
func TestUnresolvableCommandSkipped(t *testing.T) {
	cases := []struct {
		name     string
		output   string
		exitCode int
		want     bool
	}{
		{"shell command not found", "zsh: command not found: pyright", 127, true},
		{"exit 127 alone", "", 127, true},
		{"npm missing script", "npm error Missing script: \"lint\"", 1, true},
		{"npx no executable", "npm error could not determine executable to run", 1, true},
		{"make no target", "make: *** No rule to make target `lint'.  Stop.", 2, true},
		{"cargo no such command", "error: no such command: `clippy`", 101, true},
		{"real test failure is NOT skipped", "--- FAIL: TestThing (0.00s)", 1, false},
		{"real lint failure is NOT skipped", "src/index.ts:4:1 error no-unused-vars", 1, false},
		{"real build failure is NOT skipped", "cmd/foo.go:10:2: undefined: bar", 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isCommandUnresolvable(tc.output, tc.exitCode); got != tc.want {
				t.Errorf("isCommandUnresolvable(%q, %d) = %v, want %v", tc.output, tc.exitCode, tc.want, got)
			}
		})
	}
}

// TestCriterionCheckReportsOutcomeNotRawCommandText locks the FIELD-03 fix
// (.planning/todos/pending/2026-08-21-continue-checker-captures-wrong-field.md;
// 191.1-CONTEXT.md D-05): a required check that PASSES must report the
// verified outcome (step.Summary), never the raw configured shell command it
// ran (step.Command). A downstream colony's embedded continue watcher failed
// 3 of 5 runs, and on two of the "successful" runs the reported reason was
// literally the check's own command/description text. The strings below are
// the exact, verbatim strings quoted in that field report, used deliberately
// as the reproduction rather than a stand-in.
func TestCriterionCheckReportsOutcomeNotRawCommandText(t *testing.T) {
	cases := []struct {
		name        string
		check       string
		steps       []codexVerificationStep
		wantContain string
		wantMissing string
	}{
		{
			name:  "tests check reports the verified outcome, not the field-reported raw command",
			check: "tests",
			steps: []codexVerificationStep{
				{Name: "tests", Command: "/ant-continue 2>&1 | tail -100", Passed: true, Skipped: false, Summary: "tests passed"},
			},
			wantContain: "tests passed",
			wantMissing: "/ant-continue 2>&1 | tail -100",
		},
		{
			// A different check name, with its own plausible-but-different
			// Command/Summary pair, so the fix is proven general rather than
			// special-cased to the word "tests".
			name:  "a different check name is not special-cased -- the fix is general",
			check: "build",
			steps: []codexVerificationStep{
				{Name: "build", Command: "Preview what /ant-continue would do without mutating state", Passed: true, Skipped: false, Summary: "build succeeded"},
			},
			wantContain: "build succeeded",
			wantMissing: "Preview what /ant-continue would do without mutating state",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			passed, evidence, issue := evaluateCriterionCheck(tc.check, tc.steps, codexClaimVerification{}, codexWatcherVerification{})
			if !passed {
				t.Fatalf("evaluateCriterionCheck(%q) passed = false, want true (issue: %q)", tc.check, issue)
			}
			if strings.Contains(evidence, tc.wantMissing) {
				t.Errorf("evaluateCriterionCheck(%q) evidence = %q, must NOT contain the raw command/description text %q", tc.check, evidence, tc.wantMissing)
			}
			if !strings.Contains(evidence, tc.wantContain) {
				t.Errorf("evaluateCriterionCheck(%q) evidence = %q, want it to contain the real verified outcome %q", tc.check, evidence, tc.wantContain)
			}
		})
	}
}

// TestCriterionCheckFailureBranchStillUsesSummary guards the default-case
// FAILURE branch, three lines below the PASS branch fixed above, which
// already correctly used step.Summary before this fix and must continue to
// -- this catches a future edit that accidentally touches the wrong branch
// while "fixing" the PASS one.
func TestCriterionCheckFailureBranchStillUsesSummary(t *testing.T) {
	steps := []codexVerificationStep{
		{Name: "tests", Command: "go test ./...", Passed: false, Skipped: false, Summary: "tests failed: 2 failures"},
	}
	passed, evidence, issue := evaluateCriterionCheck("tests", steps, codexClaimVerification{}, codexWatcherVerification{})
	if passed {
		t.Fatalf("evaluateCriterionCheck passed = true, want false")
	}
	if evidence != "" {
		t.Errorf("evaluateCriterionCheck evidence = %q, want empty on failure", evidence)
	}
	if !strings.Contains(issue, "tests failed: 2 failures") {
		t.Errorf("evaluateCriterionCheck issue = %q, want it to contain the failure summary %q", issue, "tests failed: 2 failures")
	}
	if strings.Contains(issue, "go test ./...") {
		t.Errorf("evaluateCriterionCheck issue = %q, must not contain the raw command", issue)
	}
}

// TestCosmeticOperationalEvidenceGateIsGone asserts the always-pass
// operational_evidence gate never returns. A gate that hardcodes Passed=true
// reads as assurance while asserting nothing — operational issues now surface
// as report warnings instead.
func TestCosmeticOperationalEvidenceGateIsGone(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	report := runCodexContinueGates(
		colony.Phase{ID: 1, Name: "Test phase", Mode: colony.PhaseModeProduction},
		codexContinueManifest{},
		codexContinueVerificationReport{ChecksPassed: true},
		codexContinueAssessment{OperationalIssues: []string{"worker stalled twice"}},
		time.Now().UTC(),
		nil,
	)
	for _, check := range report.Checks {
		if check.Name == "operational_evidence" {
			t.Fatalf("operational_evidence gate has returned: %+v", check)
		}
	}
	found := false
	for _, w := range report.Warnings {
		if strings.Contains(w, "operational worker issues") {
			found = true
		}
	}
	if !found {
		t.Fatalf("operational issues did not surface as a warning: %+v", report.Warnings)
	}
}
