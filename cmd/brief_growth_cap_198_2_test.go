package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 198.2 plan 07 (WIRE-07, D-09..D-11): the previous-phase carry-forward
// section, its wiring into the build brief, and the phase-wide brief-growth
// cap (folded worker-turnaround todo).

// seedCarryForwardState198_2 creates an isolated store and a two-phase
// colony state (prevID before currentID, in that plan order) so
// resolvePreviousPhaseCarryForward can resolve "immediately preceding"
// correctly. The caller sets store = s itself is done here; callers only
// need to seed verification.json/review.json/outcome.md afterwards.
func seedCarryForwardState198_2(t *testing.T, prevID, currentID int, prevName string) *colony.ColonyState {
	t.Helper()
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	goal := "carry-forward fixture"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: currentID,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: prevID, Name: prevName, Status: colony.PhaseCompleted},
			{ID: currentID, Name: "Current phase", Status: colony.PhaseInProgress},
		}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed colony state: %v", err)
	}
	return &state
}

// ---------------------------------------------------------------------------
// Task 1: resolvePreviousPhaseCarryForward
// ---------------------------------------------------------------------------

// TestCarryForwardNamesOnlyWhatFailedOrWasFlagged proves the section names
// exactly the failed/flagged items -- two failed checks, one unmet
// criterion, one reviewer blocking issue -- one sentence each, and proves
// "only what failed" by exclusion: a passing check's own text must never
// appear.
func TestCarryForwardNamesOnlyWhatFailedOrWasFlagged(t *testing.T) {
	saveGlobalsCmd(t)
	seedCarryForwardState198_2(t, 1, 2, "Ship the thing")

	verification := codexContinueVerificationReport{
		Phase: 1,
		Steps: []codexVerificationStep{
			{Name: "go build", Passed: false, Summary: "compile error in cmd/foo.go"},
			{Name: "go test", Passed: false, Summary: "TestFoo failed"},
			{Name: "go vet", Passed: true, Summary: "vet-clean-sentinel-should-not-appear"},
		},
		Criteria: []codexCriterionVerification{
			{Criterion: "Dashboard renders exporter output", Passed: false, Summary: "no evidence found"},
		},
	}
	if err := store.SaveJSON(continuePlanArtifactsPath(1, "verification.json"), verification); err != nil {
		t.Fatalf("seed verification.json: %v", err)
	}

	review := codexContinueReviewReport{
		Phase:          1,
		Passed:         false,
		BlockingIssues: []string{"security reviewer found a hardcoded secret"},
	}
	if err := store.SaveJSON(continuePlanArtifactsPath(1, "review.json"), review); err != nil {
		t.Fatalf("seed review.json: %v", err)
	}

	section := resolvePreviousPhaseCarryForward(2)
	if section == "" {
		t.Fatal("expected a non-empty carry-forward section")
	}

	for _, want := range []string{
		"compile error in cmd/foo.go",
		"TestFoo failed",
		"Dashboard renders exporter output",
		"hardcoded secret",
	} {
		if !strings.Contains(section, want) {
			t.Errorf("carry-forward section missing %q, got:\n%s", want, section)
		}
	}
	if strings.Contains(section, "vet-clean-sentinel-should-not-appear") {
		t.Errorf("carry-forward section leaked a passing check's own text, got:\n%s", section)
	}
}

// TestAllPassedPhaseCollapsesToOneLine proves that when nothing failed and
// nothing was flagged, the section collapses to the single named-count line
// (D-09's own example shape: "Phase 3: all 12 checks passed, review clean.").
func TestAllPassedPhaseCollapsesToOneLine(t *testing.T) {
	saveGlobalsCmd(t)
	seedCarryForwardState198_2(t, 3, 4, "All green")

	verification := codexContinueVerificationReport{
		Phase: 3,
		Steps: []codexVerificationStep{
			{Name: "go build", Passed: true}, {Name: "go vet", Passed: true},
			{Name: "go test", Passed: true}, {Name: "gofmt", Passed: true},
			{Name: "goreleaser check", Passed: true}, {Name: "step6", Passed: true},
			{Name: "step7", Passed: true}, {Name: "step8", Passed: true},
			{Name: "step9", Passed: true}, {Name: "step10", Passed: true},
			{Name: "step11", Passed: true}, {Name: "step12", Passed: true},
		},
	}
	if err := store.SaveJSON(continuePlanArtifactsPath(3, "verification.json"), verification); err != nil {
		t.Fatalf("seed verification.json: %v", err)
	}
	review := codexContinueReviewReport{Phase: 3, Passed: true}
	if err := store.SaveJSON(continuePlanArtifactsPath(3, "review.json"), review); err != nil {
		t.Fatalf("seed review.json: %v", err)
	}

	section := resolvePreviousPhaseCarryForward(4)
	want := "Phase 3: all 12 checks passed, review clean."
	if !strings.Contains(section, want) {
		t.Errorf("carry-forward section missing collapsed line %q, got:\n%s", want, section)
	}
	if strings.Contains(section, "go build") {
		t.Errorf("collapsed section should not name individual passing checks, got:\n%s", section)
	}
}

// TestCarryForwardCarriesTheClosingSummaryWordForWord proves the section
// always carries the persisted outcome.md text byte-for-byte -- read from
// the file, never re-derived from the reports.
func TestCarryForwardCarriesTheClosingSummaryWordForWord(t *testing.T) {
	saveGlobalsCmd(t)
	seedCarryForwardState198_2(t, 5, 6, "Closing summary phase")

	outcomeText := "Phase 5: Closing summary phase\n\nPhase 5 verified and completed. Next: aether build 6\n"
	if err := store.AtomicWrite(continuePlanArtifactsPath(5, "outcome.md"), []byte(outcomeText)); err != nil {
		t.Fatalf("seed outcome.md: %v", err)
	}

	// Read the persisted bytes back rather than reusing the local variable --
	// the comparison must be against what is actually on disk, not against a
	// re-derived sentence (the plan's own acceptance criterion).
	persisted, err := store.ReadFile(continuePlanArtifactsPath(5, "outcome.md"))
	if err != nil {
		t.Fatalf("read back seeded outcome.md: %v", err)
	}

	section := resolvePreviousPhaseCarryForward(6)
	if !strings.Contains(section, strings.TrimSpace(string(persisted))) {
		t.Errorf("carry-forward section does not carry outcome.md word-for-word, got:\n%s\nwant substring:\n%s", section, string(persisted))
	}
}

// TestNoPrecedingPhaseMeansNoCarryForward proves phase 1 (no preceding
// phase) and a preceding phase with no persisted records both produce the
// empty string -- not an empty heading (WIRE-07 unclassified truth).
func TestNoPrecedingPhaseMeansNoCarryForward(t *testing.T) {
	saveGlobalsCmd(t)

	t.Run("first phase in the plan", func(t *testing.T) {
		s, tmpDir := newTestStoreCmd(t)
		defer os.RemoveAll(tmpDir)
		store = s
		goal := "first phase fixture"
		state := colony.ColonyState{
			Version: "3.0", Goal: &goal, CurrentPhase: 1,
			Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "First phase"}}},
		}
		if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatalf("seed colony state: %v", err)
		}
		if got := resolvePreviousPhaseCarryForward(1); got != "" {
			t.Errorf("expected empty string for the first phase, got:\n%s", got)
		}
	})

	t.Run("preceding phase has no persisted records", func(t *testing.T) {
		seedCarryForwardState198_2(t, 10, 11, "Never checked")
		if got := resolvePreviousPhaseCarryForward(11); got != "" {
			t.Errorf("expected empty string when the preceding phase has no records, got:\n%s", got)
		}
	})
}

// TestCarryForwardStaysInsideItsBudgetAndNamesOmissions proves the
// failed/flagged item list never exceeds phaseCarryForwardBudgetChars, and
// that an item which does not fit is named rather than silently dropped.
func TestCarryForwardStaysInsideItsBudgetAndNamesOmissions(t *testing.T) {
	saveGlobalsCmd(t)
	seedCarryForwardState198_2(t, 20, 21, "Overflowing phase")

	longSummary := strings.Repeat("x", 500)
	var steps []codexVerificationStep
	for i := 0; i < 10; i++ {
		steps = append(steps, codexVerificationStep{
			Name:    fmt.Sprintf("check-%02d", i),
			Passed:  false,
			Summary: longSummary,
		})
	}
	verification := codexContinueVerificationReport{Phase: 20, Steps: steps}
	if err := store.SaveJSON(continuePlanArtifactsPath(20, "verification.json"), verification); err != nil {
		t.Fatalf("seed verification.json: %v", err)
	}

	section := resolvePreviousPhaseCarryForward(21)
	itemsBody := section
	if idx := strings.Index(section, "_Not included here"); idx >= 0 {
		itemsBody = section[:idx]
	}
	if len(itemsBody) > phaseCarryForwardBudgetChars+200 {
		t.Errorf("carry-forward item list (%d chars) grew well past its %d-char budget", len(itemsBody), phaseCarryForwardBudgetChars)
	}
	if !strings.Contains(section, "check-09") && !strings.Contains(section, "_Not included here") {
		t.Errorf("expected either check-09 to be named as an omission or included, got:\n%s", section)
	}
	if !strings.Contains(section, "check-00") {
		t.Errorf("expected the earliest item to fit inside the budget, got:\n%s", section)
	}
}

// TestOnlyTheImmediatelyPrecedingPhaseIsCarried seeds records for two
// earlier phases and fails if the older one's text appears (D-10).
func TestOnlyTheImmediatelyPrecedingPhaseIsCarried(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s
	goal := "multi-phase fixture"
	state := colony.ColonyState{
		Version: "3.0", Goal: &goal, CurrentPhase: 30,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 28, Name: "Two phases back"},
			{ID: 29, Name: "Immediately preceding"},
			{ID: 30, Name: "Current"},
		}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed colony state: %v", err)
	}

	oldVerification := codexContinueVerificationReport{
		Phase: 28,
		Steps: []codexVerificationStep{{Name: "old-check", Passed: false, Summary: "OLDPHASETEXT-should-not-appear"}},
	}
	if err := store.SaveJSON(continuePlanArtifactsPath(28, "verification.json"), oldVerification); err != nil {
		t.Fatalf("seed old verification.json: %v", err)
	}
	newVerification := codexContinueVerificationReport{
		Phase: 29,
		Steps: []codexVerificationStep{{Name: "new-check", Passed: false, Summary: "NEWPHASETEXT-should-appear"}},
	}
	if err := store.SaveJSON(continuePlanArtifactsPath(29, "verification.json"), newVerification); err != nil {
		t.Fatalf("seed new verification.json: %v", err)
	}

	section := resolvePreviousPhaseCarryForward(30)
	if !strings.Contains(section, "NEWPHASETEXT-should-appear") {
		t.Errorf("expected the immediately preceding phase's text, got:\n%s", section)
	}
	if strings.Contains(section, "OLDPHASETEXT-should-not-appear") {
		t.Errorf("carry-forward section leaked a phase older than the immediately preceding one, got:\n%s", section)
	}
}

// ---------------------------------------------------------------------------
// Task 2: wiring into the build brief, and proof the previous phase's
// records survive the current phase's build-attempt cleanup.
// ---------------------------------------------------------------------------

// setupCarryForwardFailingBuildFixture198_2 seeds a two-phase colony whose
// phase 1 has a genuinely failing "tests" check (AGENTS.md's verification
// command for tests is `false`), drives phase 1's check through the real
// continue pipeline so verification.json/gates.json are written the way the
// runtime actually writes them, and leaves phase 2 as the next phase whose
// build brief plan 07's carry-forward section must reach.
func setupCarryForwardFailingBuildFixture198_2(t *testing.T, name string) string {
	t.Helper()
	s, root := newTestStore(t)
	store = s
	writeAgentsVerificationCommands(t, root,
		"- build: true", "- types: true", "- lint: true", "- tests: false")

	goal := name
	now := time.Now().UTC()
	taskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, s.BasePath(), colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: name, Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: &taskID, Goal: "Ship it", Status: colony.TaskInProgress}}},
				{ID: 2, Name: "Next phase", Status: colony.PhasePending, Tasks: []colony.Task{{ID: &nextTaskID, Goal: "Continue forward", Status: colony.TaskPending}}},
			},
		},
	})

	if err := store.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed observations fixture: %v", err)
	}

	seedContinueBuildPacket(t, s.BasePath(), 1, name, goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-cf-1", Task: "Ship it", Status: "completed", TaskID: taskID},
	})

	return root
}

// TestPreviousPhaseFailureReachesTheNextBuildersBrief seeds a real failing
// check on the preceding phase, assembles the next phase's build brief on
// both the direct lane (renderCodexBuildWorkerBrief) and the delegate lane
// (composeBuildManifestBrief, the wrapper-facing composer every manifest
// brief is built from), and asserts the failure's own wording arrives in
// each -- both lanes, per the standing two-lane rule.
func TestPreviousPhaseFailureReachesTheNextBuildersBrief(t *testing.T) {
	saveGlobals(t)
	root := setupCarryForwardFailingBuildFixture198_2(t, "Previous phase failure reaches the brief")

	result, _, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{})
	if err != nil {
		t.Fatalf("runCodexContinue: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("fixture must produce a blocked (failing) check, got advanced:true, result=%+v", result)
	}

	phase2 := colony.Phase{ID: 2, Name: "Next phase"}
	dispatch := codexBuildDispatch{Name: "Hammer-cf-2", Caste: "builder", Task: "Continue forward"}
	startedAt := time.Now()

	const wantFailureWording = `Check "tests" failed`

	t.Run("direct lane", func(t *testing.T) {
		brief := renderCodexBuildWorkerBrief(root, phase2, dispatch, startedAt)
		if !strings.Contains(brief, wantFailureWording) {
			t.Errorf("direct-lane build brief for phase 2 is missing the previous phase's failure (%q), got:\n%s", wantFailureWording, brief)
		}
	})

	t.Run("delegate lane", func(t *testing.T) {
		brief := composeBuildManifestBrief(root, phase2, dispatch, startedAt, false)
		if !strings.Contains(brief, wantFailureWording) {
			t.Errorf("delegate-lane build brief for phase 2 is missing the previous phase's failure (%q), got:\n%s", wantFailureWording, brief)
		}
	})
}

// TestBuildCleanupLeavesThePreviousPhasesRecords seeds all four artifacts
// cleanupStaleBuildAttemptArtifacts removes (verification.json, gates.json,
// continue.json, review.json) for the PRECEDING phase, runs the cleanup path
// for the CURRENT phase, and asserts all four of the preceding phase's files
// still exist and the carry-forward section still renders -- proof by test,
// not by inspection, that the current phase's own build-attempt cleanup
// never reaches the previous phase's directory.
func TestBuildCleanupLeavesThePreviousPhasesRecords(t *testing.T) {
	saveGlobalsCmd(t)
	seedCarryForwardState198_2(t, 40, 41, "Cleanup must not reach me")

	verification := codexContinueVerificationReport{
		Phase: 40,
		Steps: []codexVerificationStep{{Name: "tests", Passed: false, Summary: "cleanup-survives-sentinel"}},
	}
	gates := codexContinueGateReport{Checks: []gateCheck{{Name: "verification_steps_passed", Passed: false}}}
	review := codexContinueReviewReport{Phase: 40, Passed: false, BlockingIssues: []string{"reviewer blocking issue"}}
	continueReport := codexContinueReport{Phase: 40, Summary: "blocked"}

	for _, seed := range []struct {
		name string
		data interface{}
	}{
		{"verification.json", verification},
		{"gates.json", gates},
		{"review.json", review},
		{"continue.json", continueReport},
	} {
		if err := store.SaveJSON(continuePlanArtifactsPath(40, seed.name), seed.data); err != nil {
			t.Fatalf("seed %s: %v", seed.name, err)
		}
	}

	// Run the CURRENT phase's own build-attempt cleanup -- the code path
	// under test.
	cleanupStaleBuildAttemptArtifacts(41)

	for _, name := range []string{"verification.json", "gates.json", "continue.json", "review.json"} {
		path := filepath.Join(store.BasePath(), filepath.FromSlash(continuePlanArtifactsPath(40, name)))
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("preceding phase's %s was removed by the current phase's cleanup: %v", name, statErr)
		}
	}

	section := resolvePreviousPhaseCarryForward(41)
	if !strings.Contains(section, "cleanup-survives-sentinel") {
		t.Errorf("carry-forward section no longer renders after cleanup, got:\n%s", section)
	}
}
