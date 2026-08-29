package cmd

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// setVisualOutputMode is a small test helper that sets AETHER_OUTPUT_MODE for
// the duration of the test and restores whatever value was there before.
func setVisualOutputMode(t *testing.T, mode string) {
	t.Helper()
	orig := os.Getenv("AETHER_OUTPUT_MODE")
	os.Setenv("AETHER_OUTPUT_MODE", mode)
	t.Cleanup(func() {
		os.Setenv("AETHER_OUTPUT_MODE", orig)
	})
}

// assertLiveCheckLinesForAllChecks fails the test unless the captured output
// contains both a start line ("Running {check}…") and a finish line
// ({Check} ✓/✗ ...) for every check name given (SHOW-03 D-01: two lines per
// check, live).
func assertLiveCheckLinesForAllChecks(t *testing.T, output string, checks []string) {
	t.Helper()
	lower := strings.ToLower(output)
	for _, name := range checks {
		label := verificationStepDisplayName(name)
		startNeedle := "running " + strings.ToLower(label)
		if !strings.Contains(lower, startNeedle) {
			t.Errorf("missing start line for %q (wanted %q) in output:\n%s", name, startNeedle, output)
		}
		if !strings.Contains(output, label+" ✓") && !strings.Contains(output, label+" ✗") {
			t.Errorf("missing finish line for %q in output:\n%s", name, output)
		}
	}
}

// TestBothContinueLanesEmitLiveCheckLines proves SHOW-03 D-01 holds on both
// continue lanes: the in-process lane (runCodexContinueVerification) and the
// wrapper/external lane (runCodexContinueVerificationSnapshot). Both reach
// runVerificationStep only through runDeterministicFloor, so a start line and
// a finish line for all four checks must appear on both lanes' captured
// output — a guarantee that holds only on one lane is worth nothing
// (CLAUDE.md).
func TestBothContinueLanesEmitLiveCheckLines(t *testing.T) {
	checks := []string{"build", "types", "lint", "tests"}

	t.Run("in-process lane", func(t *testing.T) {
		saveGlobals(t)
		setVisualOutputMode(t, "visual")

		var buf bytes.Buffer
		stdout = &buf

		fixture := deterministicFloorFixtures()[0] // "all checks green"
		root, phase, manifest := fixture.build(t)

		runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)

		assertLiveCheckLinesForAllChecks(t, buf.String(), checks)
	})

	t.Run("snapshot lane", func(t *testing.T) {
		saveGlobals(t)
		setVisualOutputMode(t, "visual")

		var buf bytes.Buffer
		stdout = &buf

		fixture := deterministicFloorFixtures()[0] // "all checks green"
		root, phase, manifest := fixture.build(t)

		runCodexContinueVerificationSnapshot(root, phase, manifest, time.Now().UTC(), 5*time.Second, true)

		assertLiveCheckLinesForAllChecks(t, buf.String(), checks)
	})
}

// TestFailedCheckLineCarriesItsReasonInline proves D-02: a failing check's
// finish line carries the step's own one-line Summary inline, on the same
// line as the fail mark — never a second, invented summary format.
func TestFailedCheckLineCarriesItsReasonInline(t *testing.T) {
	saveGlobals(t)
	setVisualOutputMode(t, "visual")

	var buf bytes.Buffer
	stdout = &buf

	s, root := newTestStore(t)
	store = s
	writeAgentsVerificationCommands(t, root,
		"- build: true", "- types: true", "- lint: true", "- tests: false")
	phase := colony.Phase{ID: 1, Name: "One check failing"}
	manifest := codexContinueManifest{}

	floor := runDeterministicFloor(context.Background(), root, phase, manifest, codexWatcherVerification{}, 5*time.Second)

	var failedStep codexVerificationStep
	found := false
	for _, step := range floor.Steps {
		if step.Name == "tests" {
			failedStep = step
			found = true
		}
	}
	if !found {
		t.Fatalf("no tests step found in floor: %+v", floor.Steps)
	}
	if failedStep.Passed {
		t.Fatalf("expected tests step to fail, got Passed=true: %+v", failedStep)
	}
	reason := strings.TrimSpace(failedStep.Summary)
	if reason == "" {
		t.Fatalf("expected the failing step to carry a non-empty Summary")
	}

	output := buf.String()
	var failLine string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "✗") && strings.Contains(strings.ToLower(line), "tests") {
			failLine = line
		}
	}
	if failLine == "" {
		t.Fatalf("no failing finish line found for tests in output:\n%s", output)
	}
	if !strings.Contains(failLine, reason) {
		t.Fatalf("failing line %q does not carry the step's own reason %q inline", failLine, reason)
	}
}

// TestVerificationStepDurationIsMeasured proves Phase 196 D-01 for the new
// field: the duration on a step that really ran a command is measured (never
// estimated), landing in a sane band around a deliberately slow fixture
// command, while a step that never ran a command (Skipped) reports zero
// duration and its finish line shows no elapsed time at all.
func TestVerificationStepDurationIsMeasured(t *testing.T) {
	saveGlobals(t)
	setVisualOutputMode(t, "visual")

	var buf bytes.Buffer
	stdout = &buf

	s, root := newTestStore(t)
	store = s
	// tests: deliberately slow so its measured duration is unambiguous.
	// lint: deliberately left unresolved (no command anywhere) so it stays
	// Skipped -- no go.mod/package.json/etc in this temp root to trigger a
	// language-fallback default.
	writeAgentsVerificationCommands(t, root,
		"- build: true", "- types: true", "- tests: sh -c \"sleep 0.3 && true\"")
	phase := colony.Phase{ID: 1, Name: "Duration fixture"}
	manifest := codexContinueManifest{}

	floor := runDeterministicFloor(context.Background(), root, phase, manifest, codexWatcherVerification{}, 5*time.Second)

	var testsStep, lintStep codexVerificationStep
	for _, step := range floor.Steps {
		switch step.Name {
		case "tests":
			testsStep = step
		case "lint":
			lintStep = step
		}
	}

	if testsStep.Duration <= 0.2 || testsStep.Duration > 5.0 {
		t.Fatalf("expected tests duration in a sane band around 0.3s, got %v", testsStep.Duration)
	}
	if !lintStep.Skipped {
		t.Fatalf("expected lint to be skipped (no command resolved), got %+v", lintStep)
	}
	if lintStep.Duration != 0 {
		t.Fatalf("expected a skipped step to report zero duration, got %v", lintStep.Duration)
	}

	output := buf.String()
	var testsFinishLine, lintFinishLine string
	for _, line := range strings.Split(output, "\n") {
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "tests") && (strings.Contains(line, "✓") || strings.Contains(line, "✗")) {
			testsFinishLine = line
		}
		if strings.Contains(lowerLine, "lint") && strings.Contains(lowerLine, "skip") {
			lintFinishLine = line
		}
	}
	if testsFinishLine == "" {
		t.Fatalf("no finish line found for tests in output:\n%s", output)
	}
	if !strings.Contains(testsFinishLine, "s)") {
		t.Fatalf("expected the tests finish line to show a measured elapsed time, got %q", testsFinishLine)
	}
	if lintFinishLine == "" {
		t.Fatalf("no skip line found for lint in output:\n%s", output)
	}
	if strings.ContainsAny(lintFinishLine, "0123456789") {
		t.Fatalf("expected a skipped check's line to claim no elapsed time, got %q", lintFinishLine)
	}
}
