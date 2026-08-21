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

// pauseRaceFixture seeds COLONY_STATE.json for the FIELD-04 pause-race tests
// below. It mirrors advancePhaseFixture (cmd/advance_phase_test.go) but
// additionally exposes Paused, which that fixture has no need for.
func pauseRaceFixture(t *testing.T, phases []colony.Phase, currentPhase int, buildStartedAt *time.Time, paused bool) string {
	t.Helper()
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	goal := "pause race fixture"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   currentPhase,
		InitializedAt:  &now,
		BuildStartedAt: buildStartedAt,
		Paused:         paused,
		Plan:           colony.Plan{Phases: phases},
	})
	return dataDir
}

// rewriteColonyState reads COLONY_STATE.json, applies mutate, and writes it
// back -- the test-side equivalent of "a stop hook (or an operator resuming
// it) changes colony state underneath an in-flight continue run."
func rewriteColonyState(t *testing.T, dataDir string, mutate func(*colony.ColonyState)) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read COLONY_STATE.json: %v", err)
	}
	var state colony.ColonyState
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal COLONY_STATE.json: %v", err)
	}
	mutate(&state)
	createTestColonyState(t, dataDir, state)
}

// pendingAdvanceFixturePayload builds a minimal, already-PASSING
// verification/assessment/gates/review bundle standing in for what a real
// ~10-minute continue verification would have produced.
func pendingAdvanceFixturePayload() (codexContinueVerificationReport, codexContinueAssessment, codexContinueGateReport, codexContinueReviewReport, colony.VerificationDepth) {
	verification := codexContinueVerificationReport{Phase: 1, ChecksPassed: true, Passed: true, CriteriaPassed: true}
	assessment := codexContinueAssessment{Phase: 1, Passed: true, VerificationPassed: true, PositiveEvidence: true}
	gates := codexContinueGateReport{Phase: 1, Passed: true}
	review := codexContinueReviewReport{Phase: 1, Passed: true}
	return verification, assessment, gates, review, colony.VerificationDepthLight
}

// TestPausedVerificationIsPreservedAndReplayedAfterResume reproduces the
// FIELD-04 field-reported gap
// (.planning/todos/pending/2026-08-21-continue-checker-captures-wrong-field.md,
// "Also fold in (same subsystem): the pause race"): a ~10-minute continue
// verification finishes just as a stop hook pauses the colony underneath it,
// and the completed, PASSING result used to be thrown away entirely. This
// drives the actual race through the real wrapper-external call site
// (advanceExternalContinue) and then through the real shared replay entry
// point (replayPendingContinueAdvance -- the same function called at the top
// of both runCodexContinue and runCodexContinueFinalize), not a generic
// "state gets saved somewhere" stand-in.
func TestPausedVerificationIsPreservedAndReplayedAfterResume(t *testing.T) {
	buildStartedAt := time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)
	phases := []colony.Phase{
		{ID: 1, Name: "Phase One", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: strPtr("1.1"), Goal: "phase one work", Status: colony.TaskPending}}},
	}
	// (1) colony state has the phase in progress, not paused.
	dataDir := pauseRaceFixture(t, phases, 1, &buildStartedAt, false)
	phase := phases[0]
	verification, assessment, gates, review, reviewDepth := pendingAdvanceFixturePayload()
	pendingPath := filepath.Join(dataDir, "build", "phase-1", "pending-advance.json")

	// Precondition sanity: nothing preserved yet.
	if _, statErr := os.Stat(pendingPath); !os.IsNotExist(statErr) {
		t.Fatalf("precondition failed: pending-advance.json already exists before the race: %v", statErr)
	}

	// (2) mark state.Paused = true on disk, simulating the stop hook racing
	// in while the caller's own in-memory snapshot (below) still reflects
	// the pre-pause world -- the exact shape of the real race: the caller
	// captured `state` before spending ~10 minutes on verification, and only
	// the DISK's current state has since flipped Paused.
	rewriteColonyState(t, dataDir, func(s *colony.ColonyState) { s.Paused = true })
	callerState := colony.ColonyState{CurrentPhase: 1, BuildStartedAt: &buildStartedAt, Plan: colony.Plan{Phases: phases}}

	// (3) call the real wrapper-external advance path with the fixture
	// payload.
	result, _, _, _, _, err := advanceExternalContinue(
		dataDir, callerState, phase, codexContinueManifest{},
		verification, assessment, gates, review,
		"build/phase-1/review.json", nil, nil,
		time.Now().UTC(), "build/phase-1/verification.json", "build/phase-1/gates.json",
		reviewDepth,
	)
	if err != nil {
		t.Fatalf("advanceExternalContinue returned error: %v", err)
	}
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Errorf("result[blocked] = %v, want true (same superseded shape as today -- no visible behavior change)", result["blocked"])
	}
	if superseded, _ := result["superseded"].(bool); !superseded {
		t.Errorf("result[superseded] = %v, want true", result["superseded"])
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Errorf("result[advanced] = %v, want false -- nothing should have been committed", result["advanced"])
	}

	// A durable pending-advance record must now exist, scoped to this phase
	// and this build's identity, carrying enough of the fixture payload to
	// re-drive advancement without recomputing it.
	if _, statErr := os.Stat(pendingPath); statErr != nil {
		t.Fatalf("pending-advance.json was not written: %v", statErr)
	}
	record, ok := loadPendingContinueAdvance(1, &buildStartedAt)
	if !ok {
		t.Fatalf("loadPendingContinueAdvance did not find the just-preserved record")
	}
	if !record.Payload.Assessment.Passed || !record.Payload.Gates.Passed || !record.Payload.Review.Passed {
		t.Errorf("preserved payload does not carry the passing fixture forward: %+v", record.Payload)
	}

	// (4) simulate resume: same phase, same BuildStartedAt identity,
	// Paused flips back to false.
	rewriteColonyState(t, dataDir, func(s *colony.ColonyState) { s.Paused = false })
	resumedState := colony.ColonyState{CurrentPhase: 1, BuildStartedAt: &buildStartedAt, Plan: colony.Plan{Phases: phases}}

	// (5) invoke the continue entry path again WITHOUT supplying fresh
	// verification/assessment inputs -- replayPendingContinueAdvance's own
	// signature does not even accept them, so the phase can only advance
	// here by pulling the PRESERVED payload back off disk, not by a fresh
	// computation.
	outcome := replayPendingContinueAdvance(resumedState, phase, "continue-finalize", time.Now().UTC())
	if !outcome.Handled {
		t.Fatalf("replayPendingContinueAdvance did not handle a matching pending record")
	}
	if outcome.Err != nil {
		t.Fatalf("replayPendingContinueAdvance returned error: %v", outcome.Err)
	}
	if outcome.Phase.Status != colony.PhaseCompleted {
		t.Errorf("outcome.Phase.Status = %q, want %q", outcome.Phase.Status, colony.PhaseCompleted)
	}
	if len(outcome.State.Plan.Phases) == 0 || outcome.State.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Errorf("COLONY_STATE phase 1 status = %+v, want %q", outcome.State.Plan.Phases, colony.PhaseCompleted)
	}
	if advanced, _ := outcome.Result["advanced"].(bool); !advanced {
		t.Errorf("outcome.Result[advanced] = %v, want true -- the phase actually advanced", outcome.Result["advanced"])
	}
	onDisk := readColonyStateFile(t, dataDir)
	var persisted colony.ColonyState
	if err := json.Unmarshal(onDisk, &persisted); err != nil {
		t.Fatalf("unmarshal persisted COLONY_STATE.json: %v", err)
	}
	if persisted.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Errorf("persisted phase 1 status = %q, want %q -- advancePhase must have actually committed", persisted.Plan.Phases[0].Status, colony.PhaseCompleted)
	}

	// The pending record must be cleared -- it cannot be replayed twice.
	if _, statErr := os.Stat(pendingPath); !os.IsNotExist(statErr) {
		t.Fatalf("pending-advance.json still exists after a successful replay (stat err: %v) -- it must not be replayable twice", statErr)
	}
	secondAttempt := replayPendingContinueAdvance(outcome.State, outcome.Phase, "continue-finalize", time.Now().UTC())
	if secondAttempt.Handled {
		t.Errorf("a second replay attempt was Handled=true; the cleared record must not be replayable twice")
	}
}

// TestStalePendingAdvanceDiscardedNotReplayed proves the other half of
// FIELD-04's contract: a preserved verification whose build identity has
// genuinely changed underneath it (a different build actually happened) is
// discarded, never blindly replayed against evidence that no longer
// describes the current build.
func TestStalePendingAdvanceDiscardedNotReplayed(t *testing.T) {
	originalBuildStartedAt := time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)
	phases := []colony.Phase{
		{ID: 1, Name: "Phase One", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: strPtr("1.1"), Goal: "phase one work", Status: colony.TaskPending}}},
	}
	dataDir := pauseRaceFixture(t, phases, 1, &originalBuildStartedAt, true)
	phase := phases[0]
	verification, assessment, gates, review, reviewDepth := pendingAdvanceFixturePayload()
	pendingPath := filepath.Join(dataDir, "build", "phase-1", "pending-advance.json")

	// Set up a pending record the same way a real paused advance would --
	// through the real preserve call, not a hand-crafted file.
	callerState := colony.ColonyState{CurrentPhase: 1, BuildStartedAt: &originalBuildStartedAt, Plan: colony.Plan{Phases: phases}}
	result, _, _, _, _, err := advanceExternalContinue(
		dataDir, callerState, phase, codexContinueManifest{},
		verification, assessment, gates, review,
		"build/phase-1/review.json", nil, nil,
		time.Now().UTC(), "build/phase-1/verification.json", "build/phase-1/gates.json",
		reviewDepth,
	)
	if err != nil {
		t.Fatalf("advanceExternalContinue returned error: %v", err)
	}
	if superseded, _ := result["superseded"].(bool); !superseded {
		t.Fatalf("setup failed: expected a superseded result while paused, got: %+v", result)
	}
	// Precondition: the pending record genuinely exists before we go on to
	// prove it gets discarded -- otherwise this test would vacuously pass.
	if _, statErr := os.Stat(pendingPath); statErr != nil {
		t.Fatalf("precondition failed: pending-advance.json was not created by setup: %v", statErr)
	}

	// A genuinely different build happened in between: BuildStartedAt moves
	// forward and the colony resumes (Paused = false) into that new build,
	// still on the same phase.
	newBuildStartedAt := originalBuildStartedAt.Add(2 * time.Hour)
	rewriteColonyState(t, dataDir, func(s *colony.ColonyState) {
		s.Paused = false
		s.BuildStartedAt = &newBuildStartedAt
	})
	staleResumedState := colony.ColonyState{CurrentPhase: 1, BuildStartedAt: &newBuildStartedAt, Plan: colony.Plan{Phases: phases}}

	outcome := replayPendingContinueAdvance(staleResumedState, phase, "continue-finalize", time.Now().UTC())
	if outcome.Handled {
		t.Errorf("replayPendingContinueAdvance Handled = true for a stale record, want false (caller must fall through to ordinary fresh verification)")
	}

	// The stale record must be gone -- discarded, not just skipped, so it
	// can never be replayed later either.
	if _, statErr := os.Stat(pendingPath); !os.IsNotExist(statErr) {
		t.Fatalf("stale pending-advance.json still exists (stat err: %v); it must be discarded, not merely ignored", statErr)
	}

	// The phase must NOT have silently advanced using the stale evidence.
	onDisk := readColonyStateFile(t, dataDir)
	var persisted colony.ColonyState
	if err := json.Unmarshal(onDisk, &persisted); err != nil {
		t.Fatalf("unmarshal persisted COLONY_STATE.json: %v", err)
	}
	if persisted.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Errorf("persisted phase 1 status = %q, want %q -- a stale record must never silently advance the phase", persisted.Plan.Phases[0].Status, colony.PhaseInProgress)
	}
}

// TestBothContinueCallSitesShareOnePreserveFunction is the "light parity
// check" the plan calls for: the SAME preserve function is called from both
// cmd/codex_continue.go's and cmd/codex_continue_finalize.go's
// errRuntimeStateSuperseded handling for the "colony is paused" reason,
// rather than either call site growing its own independent discard-on-pause
// logic. A direct unit call against preserveIfPausedSupersession proves the
// mechanism itself is correctly parameterized by Source for either caller;
// the source-text checks prove neither production call site has drifted away
// from actually calling it.
func TestBothContinueCallSitesShareOnePreserveFunction(t *testing.T) {
	for _, source := range []string{"continue", "continue-finalize"} {
		t.Run(source, func(t *testing.T) {
			buildStartedAt := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
			phases := []colony.Phase{
				{ID: 1, Name: "Phase One", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: strPtr("1.1"), Goal: "work", Status: colony.TaskPending}}},
			}
			pauseRaceFixture(t, phases, 1, &buildStartedAt, true)
			verification, assessment, gates, review, reviewDepth := pendingAdvanceFixturePayload()

			preserveIfPausedSupersession(1, &buildStartedAt, source, time.Now().UTC(), pendingContinueAdvancePayload{
				Verification: verification,
				Assessment:   assessment,
				Gates:        gates,
				Review:       review,
				ReviewDepth:  reviewDepth,
			})

			record, ok := loadPendingContinueAdvance(1, &buildStartedAt)
			if !ok {
				t.Fatalf("preserveIfPausedSupersession(source=%q) did not preserve a record while paused", source)
			}
			if record.Source != source {
				t.Errorf("record.Source = %q, want %q", record.Source, source)
			}
		})
	}

	for _, file := range []string{"codex_continue.go", "codex_continue_finalize.go"} {
		t.Run("source wiring/"+file, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read %s: %v", file, err)
			}
			if !strings.Contains(string(data), "preserveIfPausedSupersession(") {
				t.Errorf("%s does not call preserveIfPausedSupersession -- its errRuntimeStateSuperseded handling must use the shared preserve mechanism (cmd/advance_phase.go), not an independent copy", file)
			}
		})
	}
}

// TestNonPauseSupersessionDoesNotPreserve is the other bullet of the same
// parity requirement: every OTHER supersession reason (a genuinely
// different phase or build, not a pause) continues to discard exactly as
// before -- it must never create a pending-advance record.
func TestNonPauseSupersessionDoesNotPreserve(t *testing.T) {
	buildStartedAt := time.Date(2026, 8, 21, 11, 0, 0, 0, time.UTC)
	phases := []colony.Phase{
		{ID: 1, Name: "Phase One", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: strPtr("1.1"), Goal: "work", Status: colony.TaskPending}}},
	}
	// Paused = false: any supersession this test triggers must be for a
	// reason OTHER than pause.
	dataDir := pauseRaceFixture(t, phases, 1, &buildStartedAt, false)
	verification, assessment, gates, review, reviewDepth := pendingAdvanceFixturePayload()
	pendingPath := filepath.Join(dataDir, "build", "phase-1", "pending-advance.json")

	staleCapture := buildStartedAt.Add(1 * time.Hour)
	callerState := colony.ColonyState{CurrentPhase: 1, BuildStartedAt: &staleCapture, Plan: colony.Plan{Phases: phases}}
	phase := phases[0]

	result, _, _, _, _, err := advanceExternalContinue(
		dataDir, callerState, phase, codexContinueManifest{},
		verification, assessment, gates, review,
		"build/phase-1/review.json", nil, nil,
		time.Now().UTC(), "build/phase-1/verification.json", "build/phase-1/gates.json",
		reviewDepth,
	)
	if err != nil {
		t.Fatalf("advanceExternalContinue returned error: %v", err)
	}
	if superseded, _ := result["superseded"].(bool); !superseded {
		t.Fatalf("expected a superseded result from the BuildStartedAt mismatch, got: %+v", result)
	}

	if _, statErr := os.Stat(pendingPath); !os.IsNotExist(statErr) {
		t.Errorf("pending-advance.json was written for a non-pause supersession (stat err: %v); only the pause reason may preserve", statErr)
	}
}
