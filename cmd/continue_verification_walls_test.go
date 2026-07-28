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
