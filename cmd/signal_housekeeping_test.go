package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func writeTestPheromones(t *testing.T, dataDir string, pf colony.PheromoneFile) {
	t.Helper()
	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal pheromones: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "pheromones.json"), data, 0644); err != nil {
		t.Fatalf("failed to write pheromones: %v", err)
	}
}

func TestSignalHousekeepingExpiresSignalsAndShrinksPrompt(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	now := time.Date(2026, 4, 15, 17, 34, 14, 0, time.UTC)

	goal := "guidance foundation"
	initializedAt := now
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 4,
		InitializedAt: &initializedAt,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "p1", Status: colony.PhaseCompleted},
				{ID: 2, Name: "p2", Status: colony.PhaseCompleted},
				{ID: 3, Name: "p3", Status: colony.PhaseCompleted},
				{ID: 4, Name: "p4", Status: colony.PhaseReady},
			},
		},
		Events: []string{
			"2026-04-12T09:00:00Z|phase_advanced|continue|Completed phase 1, ready for phase 2",
			"2026-04-13T09:00:00Z|phase_advanced|continue|Completed phase 2, ready for phase 3",
			"2026-04-14T09:00:00Z|phase_advanced|continue|Completed phase 3, ready for phase 4",
		},
	})

	s0_6 := 0.6
	s1_0 := 1.0
	writeTestPheromones(t, dataDir, colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:        "sig_expired",
				Type:      "FEEDBACK",
				Priority:  "low",
				Source:    "worker:continue",
				CreatedAt: "2026-04-10T09:00:00Z",
				ExpiresAt: ptrString("2026-04-11T09:00:00Z"),
				Active:    true,
				Strength:  &s0_6,
				Content:   json.RawMessage(`{"text":"` + strings.Repeat("expired signal ", 20) + `"}`),
			},
			{
				ID:        "sig_weak",
				Type:      "FEEDBACK",
				Priority:  "low",
				Source:    "auto:success",
				CreatedAt: "2025-01-01T09:00:00Z",
				Active:    true,
				Strength:  &s0_6,
				Content:   json.RawMessage(`{"text":"` + strings.Repeat("weak signal ", 20) + `"}`),
			},
			{
				ID:        "sig_continue",
				Type:      "FEEDBACK",
				Priority:  "low",
				Source:    "worker:continue",
				CreatedAt: "2026-04-11T09:00:00Z",
				Active:    true,
				Strength:  &s0_6,
				Content:   json.RawMessage(`{"text":"` + strings.Repeat("continue signal ", 20) + `"}`),
			},
			{
				ID:        "sig_keep",
				Type:      "REDIRECT",
				Priority:  "high",
				Source:    "user",
				CreatedAt: "2026-04-15T09:00:00Z",
				Active:    true,
				Strength:  &s1_0,
				Content:   json.RawMessage(`{"text":"avoid globals"}`),
			},
		},
	})

	var beforeBuf bytes.Buffer
	stdout = &beforeBuf
	rootCmd.SetArgs([]string{"pheromone-prime"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-prime before housekeeping failed: %v", err)
	}
	beforeEnv := parseLifecycleEnvelope(t, beforeBuf.String())
	before := beforeEnv["result"].(map[string]interface{})
	beforeSection := before["section"].(string)
	beforeCount := before["signal_count"].(float64)

	saveGlobals(t)
	resetRootCmd(t)
	stdout = &bytes.Buffer{}

	if store == nil {
		t.Fatal("expected store to be initialized")
	}
	result := applySignalHousekeeping(loadPheromones(), &colony.ColonyState{
		Events: []string{
			"2026-04-12T09:00:00Z|phase_advanced|continue|Completed phase 1, ready for phase 2",
			"2026-04-13T09:00:00Z|phase_advanced|continue|Completed phase 2, ready for phase 3",
			"2026-04-14T09:00:00Z|phase_advanced|continue|Completed phase 3, ready for phase 4",
		},
	}, now, false)
	if result.ExpiredByTime != 1 {
		t.Fatalf("expired_by_time = %d, want 1", result.ExpiredByTime)
	}
	if result.DeactivatedByStrength != 1 {
		t.Fatalf("deactivated_by_strength = %d, want 1", result.DeactivatedByStrength)
	}
	if result.ExpiredWorkerContinue != 1 {
		t.Fatalf("expired_worker_continue = %d, want 1", result.ExpiredWorkerContinue)
	}

	// Persist the mutation through the real command path.
	saveGlobals(t)
	resetRootCmd(t)
	var housekeepingBuf bytes.Buffer
	stdout = &housekeepingBuf
	rootCmd.SetArgs([]string{"signal-housekeeping"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("signal-housekeeping failed: %v", err)
	}
	housekeepingEnv := parseLifecycleEnvelope(t, housekeepingBuf.String())
	housekeeping := housekeepingEnv["result"].(map[string]interface{})
	if housekeeping["active_before"] != float64(4) {
		t.Fatalf("active_before = %v, want 4", housekeeping["active_before"])
	}
	if housekeeping["active_after"] != float64(1) {
		t.Fatalf("active_after = %v, want 1", housekeeping["active_after"])
	}

	saveGlobals(t)
	resetRootCmd(t)
	var afterBuf bytes.Buffer
	stdout = &afterBuf
	rootCmd.SetArgs([]string{"pheromone-prime"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-prime after housekeeping failed: %v", err)
	}
	afterEnv := parseLifecycleEnvelope(t, afterBuf.String())
	after := afterEnv["result"].(map[string]interface{})
	afterSection := after["section"].(string)
	afterCount := after["signal_count"].(float64)

	if beforeCount != 4 {
		t.Fatalf("before signal_count = %v, want 4", beforeCount)
	}
	if afterCount != 1 {
		t.Fatalf("after signal_count = %v, want 1", afterCount)
	}
	if len(afterSection) >= len(beforeSection) {
		t.Fatalf("expected pheromone prompt section to shrink after housekeeping, before=%d after=%d", len(beforeSection), len(afterSection))
	}
}

func TestContinueRunsSignalHousekeeping(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "guidance foundation"
	initializedAt := time.Date(2026, 4, 15, 17, 34, 14, 0, time.UTC)
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		InitializedAt: &initializedAt,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "phase 1", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: ptrString("1.1"), Goal: "do work", Status: colony.TaskPending}}},
				{ID: 2, Name: "phase 2", Status: colony.PhasePending},
			},
		},
	})

	s0_6 := 0.6
	writeTestPheromones(t, dataDir, colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:        "sig_expired",
				Type:      "FEEDBACK",
				Priority:  "low",
				Source:    "worker:continue",
				CreatedAt: "2026-04-10T09:00:00Z",
				ExpiresAt: ptrString("2026-04-11T09:00:00Z"),
				Active:    true,
				Strength:  &s0_6,
				Content:   json.RawMessage(`{"text":"phase-summary"}`),
			},
		},
	})

	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue failed: %v", err)
	}
	env := parseLifecycleEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["next_phase"] != float64(2) {
		t.Fatalf("next_phase = %v, want 2", result["next_phase"])
	}
	housekeeping := result["signal_housekeeping"].(map[string]interface{})
	if housekeeping["expired_by_time"] != float64(1) {
		t.Fatalf("expired_by_time = %v, want 1", housekeeping["expired_by_time"])
	}

	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("failed to reload pheromones: %v", err)
	}
	if pf.Signals[0].Active {
		t.Fatal("expected expired signal to be deactivated by continue housekeeping")
	}
}

func ptrString(s string) *string {
	return &s
}
