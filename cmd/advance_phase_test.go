package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// advancePhaseFixture seeds a fresh COLONY_STATE.json with the given phases
// and returns the data directory it lives in, so a test can read the raw
// file back to prove (or disprove) a write occurred. saveGlobals runs first
// so the package-level store global (which setupBuildFlowTest points at this
// test's own temp dir) is restored once this test ends, rather than left
// dangling at a directory t.TempDir() has already removed -- setupBuildFlowTest
// itself does not restore store on cleanup, only stdout/stderr.
func advancePhaseFixture(t *testing.T, phases []colony.Phase, currentPhase int, state colony.State, buildStartedAt *time.Time) string {
	t.Helper()
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	goal := "advancePhase fixture"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          state,
		CurrentPhase:   currentPhase,
		InitializedAt:  &now,
		BuildStartedAt: buildStartedAt,
		Plan:           colony.Plan{Phases: phases},
	})
	return dataDir
}

func readColonyStateFile(t *testing.T, dataDir string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read COLONY_STATE.json: %v", err)
	}
	return data
}

// TestAdvancePhaseHappyPath covers the non-final (advance to next phase) and
// final (complete the colony) success cases as two cases of one table test,
// per the plan's grouping guidance.
func TestAdvancePhaseHappyPath(t *testing.T) {
	cases := []struct {
		name          string
		targetPhase   int
		wantFinal     bool
		wantNextCmd   string
		wantNextPhase bool
	}{
		{name: "non-final advances to next phase", targetPhase: 1, wantFinal: false, wantNextCmd: "aether build 2", wantNextPhase: true},
		{name: "final phase completes the colony", targetPhase: 2, wantFinal: true, wantNextCmd: "aether seal", wantNextPhase: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buildStartedAt := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
			phase1Status := colony.PhaseInProgress
			phase2Status := colony.PhasePending
			if tc.targetPhase == 2 {
				phase1Status = colony.PhaseCompleted
				phase2Status = colony.PhaseInProgress
			}
			phases := []colony.Phase{
				{ID: 1, Name: "Phase One", Status: phase1Status, Tasks: []colony.Task{{ID: strPtr("1.1"), Goal: "phase one work", Status: colony.TaskPending}}},
				{ID: 2, Name: "Phase Two", Status: phase2Status, Tasks: []colony.Task{{ID: strPtr("2.1"), Goal: "phase two work", Status: colony.TaskPending}}},
			}
			advancePhaseFixture(t, phases, tc.targetPhase, colony.StateBUILT, &buildStartedAt)

			result, err := advancePhase(advancePhaseParams{
				PhaseID:                tc.targetPhase,
				ExpectedBuildStartedAt: &buildStartedAt,
				AllowedStates:          []colony.State{colony.StateEXECUTING, colony.StateBUILT},
				Source:                 "continue",
				Now:                    time.Now().UTC(),
			})
			if err != nil {
				t.Fatalf("advancePhase returned error: %v", err)
			}

			idx := tc.targetPhase - 1
			if result.Updated.Plan.Phases[idx].Status != colony.PhaseCompleted {
				t.Errorf("phase %d status = %q, want %q", tc.targetPhase, result.Updated.Plan.Phases[idx].Status, colony.PhaseCompleted)
			}
			for _, task := range result.Updated.Plan.Phases[idx].Tasks {
				if task.Status != colony.TaskCompleted {
					t.Errorf("task %v status = %q, want %q", task.ID, task.Status, colony.TaskCompleted)
				}
			}
			if result.Updated.BuildStartedAt != nil {
				t.Errorf("BuildStartedAt = %v, want nil", result.Updated.BuildStartedAt)
			}
			if result.Updated.GateResults != nil {
				t.Errorf("GateResults = %v, want nil", result.Updated.GateResults)
			}
			if result.Final != tc.wantFinal {
				t.Errorf("Final = %v, want %v", result.Final, tc.wantFinal)
			}
			if result.NextCommand != tc.wantNextCmd {
				t.Errorf("NextCommand = %q, want %q", result.NextCommand, tc.wantNextCmd)
			}
			if tc.wantNextPhase && result.NextPhase == nil {
				t.Errorf("NextPhase = nil, want non-nil")
			}
			if !tc.wantNextPhase && result.NextPhase != nil {
				t.Errorf("NextPhase = %+v, want nil", result.NextPhase)
			}
			if tc.wantFinal {
				if result.Updated.State != colony.StateCOMPLETED {
					t.Errorf("State = %q, want %q", result.Updated.State, colony.StateCOMPLETED)
				}
				if result.Updated.CurrentPhase != tc.targetPhase {
					t.Errorf("CurrentPhase = %d, want %d (unchanged, not incremented past the plan)", result.Updated.CurrentPhase, tc.targetPhase)
				}
			} else {
				if result.Updated.State != colony.StateREADY {
					t.Errorf("State = %q, want %q", result.Updated.State, colony.StateREADY)
				}
				if result.Updated.CurrentPhase != tc.targetPhase+1 {
					t.Errorf("CurrentPhase = %d, want %d", result.Updated.CurrentPhase, tc.targetPhase+1)
				}
			}
		})
	}
}

// TestAdvancePhaseRefusesOnBuildStartedAtMismatch reproduces a concurrent
// process having started a new build between when the caller captured
// ExpectedBuildStartedAt and when advancePhase runs. This is the exact class
// of race advanceExternalContinue's clobber bug used to silently discard.
func TestAdvancePhaseRefusesOnBuildStartedAtMismatch(t *testing.T) {
	onDisk := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	staleCapture := onDisk.Add(1 * time.Hour)
	phases := []colony.Phase{
		{ID: 1, Name: "Phase One", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: strPtr("1.1"), Goal: "work", Status: colony.TaskPending}}},
	}
	dataDir := advancePhaseFixture(t, phases, 1, colony.StateBUILT, &onDisk)
	before := readColonyStateFile(t, dataDir)

	_, err := advancePhase(advancePhaseParams{
		PhaseID:                1,
		ExpectedBuildStartedAt: &staleCapture,
		AllowedStates:          []colony.State{colony.StateEXECUTING, colony.StateBUILT},
		Source:                 "continue-finalize",
		Now:                    time.Now().UTC(),
	})
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !errors.Is(err, errRuntimeStateSuperseded) {
		t.Fatalf("expected errRuntimeStateSuperseded, got: %v", err)
	}

	after := readColonyStateFile(t, dataDir)
	if !bytes.Equal(before, after) {
		t.Fatalf("COLONY_STATE.json changed on refusal;\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestAdvancePhaseRefusesWhenPhaseNotInProgress simulates a second advance
// attempt on a phase that a concurrent process already completed.
func TestAdvancePhaseRefusesWhenPhaseNotInProgress(t *testing.T) {
	buildStartedAt := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	phases := []colony.Phase{
		{ID: 1, Name: "Phase One", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: strPtr("1.1"), Goal: "work", Status: colony.TaskCompleted}}},
	}
	dataDir := advancePhaseFixture(t, phases, 1, colony.StateBUILT, &buildStartedAt)
	before := readColonyStateFile(t, dataDir)

	_, err := advancePhase(advancePhaseParams{
		PhaseID:                1,
		ExpectedBuildStartedAt: &buildStartedAt,
		AllowedStates:          []colony.State{colony.StateEXECUTING, colony.StateBUILT},
		Source:                 "continue",
		Now:                    time.Now().UTC(),
	})
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !errors.Is(err, errRuntimeStateSuperseded) {
		t.Fatalf("expected errRuntimeStateSuperseded, got: %v", err)
	}

	after := readColonyStateFile(t, dataDir)
	if !bytes.Equal(before, after) {
		t.Fatalf("COLONY_STATE.json changed on refusal;\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestAdvancePhaseEventsUseSourceLiteral proves every event string
// advancePhase appends carries the caller's own Source value, not a
// hardcoded "continue" literal left over from the code this was extracted
// from -- the finalize path's events must read "|continue-finalize|".
func TestAdvancePhaseEventsUseSourceLiteral(t *testing.T) {
	buildStartedAt := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	phases := []colony.Phase{
		{ID: 1, Name: "Phase One", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: strPtr("1.1"), Goal: "work", Status: colony.TaskPending}}},
	}
	advancePhaseFixture(t, phases, 1, colony.StateBUILT, &buildStartedAt)

	result, err := advancePhase(advancePhaseParams{
		PhaseID:                1,
		ExpectedBuildStartedAt: &buildStartedAt,
		AllowedStates:          []colony.State{colony.StateEXECUTING, colony.StateBUILT},
		Source:                 "continue-finalize",
		Now:                    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("advancePhase returned error: %v", err)
	}

	if len(result.Updated.Events) == 0 {
		t.Fatalf("expected events to be appended, got none")
	}
	foundSource := false
	for _, event := range result.Updated.Events {
		if strings.Contains(event, "|continue|") {
			t.Errorf("event %q used the bare continue literal, want continue-finalize", event)
		}
		if strings.Contains(event, "|continue-finalize|") {
			foundSource = true
		}
	}
	if !foundSource {
		t.Errorf("expected at least one event containing |continue-finalize|, got: %v", result.Updated.Events)
	}
}
