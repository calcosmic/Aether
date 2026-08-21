package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Deliberately-RED phases (Phase.ExpectFailingTests) — the Pocket-Chopper
// field failure's second regression lock. Aether's own route-setter plans
// TDD red-first phases whose deliverable IS a failing test run, but the
// tests check treated every non-zero exit as a blocker, so such a phase
// could never advance. These tests drive the real verification snapshot
// (real shell exec of the resolved test command) so they fail if the
// inversion is ever unwired from the caller, not merely deleted from the
// helper.

// redPhaseTestRoot builds a temp repo whose ONLY resolved verification
// command is the tests command given — "false" for a failing suite, "true"
// for a green one (both pass looksLikeVerificationCommand).
func redPhaseTestRoot(t *testing.T, testCommand string) string {
	t.Helper()
	root := t.TempDir()
	content := "## Verification Commands\n\n```bash\n# Run Go tests\n" + testCommand + "\n```\n"
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	return root
}

func TestRedPhaseAdvancesOnFailingTests(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	root := redPhaseTestRoot(t, "false")
	phase := colony.Phase{ID: 1, Name: "Reproduce and lock the defects (RED)", ExpectFailingTests: true}

	report := runCodexContinueVerificationSnapshot(root, phase, codexContinueManifest{}, time.Now().UTC(), time.Minute, true)

	if !report.ChecksPassed {
		t.Fatalf("a RED phase with a genuinely failing test run must pass verification; blockers: %v", report.BlockingIssues)
	}
	for _, b := range report.BlockingIssues {
		if strings.Contains(b, "tests") {
			t.Fatalf("failing tests were still reported as a blocker on a RED phase: %v", report.BlockingIssues)
		}
	}
	// Control: the SAME failing run without the marker must still block —
	// the inversion is opt-in per phase, never ambient.
	control := runCodexContinueVerificationSnapshot(root, colony.Phase{ID: 1, Name: "ordinary phase"}, codexContinueManifest{}, time.Now().UTC(), time.Minute, true)
	if control.ChecksPassed {
		t.Fatalf("a failing test run passed verification on an ordinary phase — the inversion leaked past ExpectFailingTests")
	}
}

func TestRedPhaseBlocksOnGreenTests(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	root := redPhaseTestRoot(t, "true")
	phase := colony.Phase{ID: 1, Name: "Reproduce and lock the defects (RED)", ExpectFailingTests: true}

	report := runCodexContinueVerificationSnapshot(root, phase, codexContinueManifest{}, time.Now().UTC(), time.Minute, true)

	if report.ChecksPassed {
		t.Fatalf("a RED phase with a GREEN test run must block — the failing tests that prove the defect were never written")
	}
	found := false
	for _, b := range report.BlockingIssues {
		if strings.Contains(b, "expected failing tests") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the green-suite blocker does not explain the RED expectation: %v", report.BlockingIssues)
	}
}

// TestRedPhaseTimeoutIsNotAnExpectedFailure pins the guard: a test command
// that times out (or is environment-blocked) is not a failing test suite,
// and must NOT satisfy a RED phase's expectation.
func TestRedPhaseTimeoutIsNotAnExpectedFailure(t *testing.T) {
	steps := []codexVerificationStep{
		{Name: "tests", Passed: false, TimedOut: true, Summary: "tests timed out"},
	}
	out := applyExpectedTestFailure(steps, colony.Phase{ExpectFailingTests: true})
	if out[0].Passed {
		t.Fatalf("a timed-out test run was credited as the expected RED failure")
	}
	envStep := []codexVerificationStep{
		{Name: "tests", Passed: false, ErrorClass: ErrorClassEnvironment, Summary: "missing toolchain"},
	}
	out = applyExpectedTestFailure(envStep, colony.Phase{ExpectFailingTests: true})
	if out[0].Passed {
		t.Fatalf("an environment fault was credited as the expected RED failure")
	}
}

// TestRedPhasePlannerRoundTrip proves the authored field survives from the
// route-setter's plan artifact into colony state and back out of JSON — the
// planner can express the phase shape its own gate now executes.
func TestRedPhasePlannerRoundTrip(t *testing.T) {
	artifact := codexWorkerPlanArtifact{
		Phases: []codexWorkerPlanPhase{
			{
				Name:               "Reproduce and lock the defects (RED)",
				Description:        "Write failing tests that prove each reported bug exists.",
				ExpectFailingTests: true,
				Tasks:              []codexWorkerPlanTask{{Goal: "write the failing repro tests"}},
			},
			{
				Name:        "Fix the defects (GREEN)",
				Description: "Make the repro tests pass.",
				Tasks:       []codexWorkerPlanTask{{Goal: "fix until green"}},
			},
		},
	}
	phases := buildWorkerPlanPhases(artifact)
	if len(phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(phases))
	}
	if !phases[0].ExpectFailingTests {
		t.Fatalf("expect_failing_tests did not survive from the plan artifact to colony state")
	}
	if phases[1].ExpectFailingTests {
		t.Fatalf("expect_failing_tests leaked onto a phase that did not author it")
	}

	raw, err := json.Marshal(phases[0])
	if err != nil {
		t.Fatalf("marshal phase: %v", err)
	}
	if !strings.Contains(string(raw), `"expect_failing_tests":true`) {
		t.Fatalf("phase JSON does not carry expect_failing_tests: %s", raw)
	}
	var restored colony.Phase
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatalf("unmarshal phase: %v", err)
	}
	if !restored.ExpectFailingTests {
		t.Fatalf("expect_failing_tests lost on JSON round-trip")
	}

	// The planner prompt must teach the field, or route-setters keep
	// emitting RED phases without it.
	if !strings.Contains(renderPhasePlanSchemaGuidance(), "expect_failing_tests") {
		t.Fatalf("phase-plan schema guidance does not mention expect_failing_tests")
	}
}
