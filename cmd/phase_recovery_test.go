package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestContinueMissingBuildPacketSuggestsForceRedispatch(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Recover missing build packet"
	startedAt := time.Now().UTC().Add(-20 * time.Minute)
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &startedAt,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Stuck active phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Work that never reported", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Blocked next phase",
					Status: colony.PhasePending,
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	if got := strings.TrimSpace(stderr.(*bytes.Buffer).String()); got != "" {
		t.Fatalf("expected no stderr for blocked recovery result, got: %s", got)
	}
	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked=true, got %v", result)
	}
	if missing, _ := result["missing_packet"].(bool); !missing {
		t.Fatalf("expected missing_packet=true, got %v", result)
	}
	if next := result["next"].(string); next != "aether build 1 --force" {
		t.Fatalf("next = %q, want force redispatch", next)
	}
	recovery := result["recovery"].(map[string]interface{})
	if got := recovery["redispatch_command"].(string); got != "aether build 1 --force" {
		t.Fatalf("redispatch_command = %q, want force build", got)
	}
	if got := recovery["skip_command"].(string); got != "aether skip-phase 1 --force" {
		t.Fatalf("skip_command = %q, want skip phase command", got)
	}
}

func TestBuildForceRedispatchesActiveExecutingPhase(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Force redispatch active phase"
	startedAt := time.Now().UTC().Add(-20 * time.Minute)
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &startedAt,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Redispatch me",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Finish after timeout", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1", "--synthetic", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if force, _ := result["force"].(bool); !force {
		t.Fatalf("expected force=true in result, got %v", result["force"])
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.CurrentPhase != 1 {
		t.Fatalf("current_phase = %d, want 1", state.CurrentPhase)
	}
	if state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase status = %s, want in_progress awaiting continue", state.Plan.Phases[0].Status)
	}
}

func TestSkipPhaseForceAdvancesFromExecuting(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Skip deadlocked phase"
	startedAt := time.Now().UTC().Add(-20 * time.Minute)
	taskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &startedAt,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Deadlocked phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Timed out work", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Continue after skip", Status: colony.TaskPending}},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"skip-phase", "1", "--force", "--reason", "worker timeouts"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skip-phase returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if skipped, _ := result["skipped"].(bool); !skipped {
		t.Fatalf("expected skipped=true, got %v", result)
	}
	if next := result["next"].(string); next != "aether build 2" {
		t.Fatalf("next = %q, want aether build 2", next)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", state.State)
	}
	if state.CurrentPhase != 2 {
		t.Fatalf("current_phase = %d, want 2", state.CurrentPhase)
	}
	if state.BuildStartedAt != nil {
		t.Fatalf("build_started_at should be cleared after skip")
	}
	if state.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Fatalf("phase 1 status = %s, want completed", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[1].Status != colony.PhaseReady {
		t.Fatalf("phase 2 status = %s, want ready", state.Plan.Phases[1].Status)
	}
	if !strings.Contains(strings.Join(state.Events, "\n"), "phase_skipped|skip-phase|Force skipped phase 1: worker timeouts") {
		t.Fatalf("expected phase_skipped audit event, got %v", state.Events)
	}
}

func TestSkipPhaseRequiresForce(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Require force for skip"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Ready phase", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Goal: "Do work", Status: colony.TaskPending}}}},
		},
	})

	rootCmd.SetArgs([]string{"skip-phase", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skip-phase returned error: %v", err)
	}
	if !strings.Contains(stderr.(*bytes.Buffer).String(), "skip-phase is destructive") {
		t.Fatalf("expected force error, got: %s", stderr.(*bytes.Buffer).String())
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.Plan.Phases[0].Status != colony.PhaseReady {
		t.Fatalf("phase status changed without force: %s", state.Plan.Phases[0].Status)
	}
}
