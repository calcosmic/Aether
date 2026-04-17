package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPauseColonyWritesHandoffAndSession(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Pause this colony cleanly"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Execution",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Implement pause-colony", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"pause-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pause-colony returned error: %v", err)
	}

	if !strings.Contains(buf.String(), `"paused":true`) {
		t.Fatalf("expected paused:true JSON, got: %s", buf.String())
	}

	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		t.Fatalf("expected session.json to be written: %v", err)
	}
	if session.LastCommand != "pause-colony" {
		t.Fatalf("session.LastCommand = %q, want pause-colony", session.LastCommand)
	}
	if !session.ContextCleared {
		t.Fatal("expected ContextCleared to be true after pause")
	}

	handoffPath := filepath.Join(os.Getenv("AETHER_ROOT"), ".aether", "HANDOFF.md")
	data, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("expected handoff file: %v", err)
	}
	handoff := string(data)
	for _, want := range []string{"# Colony Session — Paused Colony", "Pause this colony cleanly", "Implement pause-colony", "aether resume"} {
		if !strings.Contains(handoff, want) {
			t.Errorf("handoff missing %q\n%s", want, handoff)
		}
	}
}

func TestResumeColonyRestoresSessionAndClearsHandoff(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Resume this colony cleanly"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Execution", Status: colony.PhaseInProgress},
				{ID: 2, Name: "Verification", Status: colony.PhaseReady},
			},
		},
	})

	session := colony.SessionFile{
		SessionID:      "resume-test",
		StartedAt:      "2026-04-15T10:00:00Z",
		ColonyGoal:     goal,
		CurrentPhase:   1,
		SuggestedNext:  "aether status",
		ContextCleared: true,
		Summary:        "Paused mid-execution",
	}
	if err := store.SaveJSON("session.json", session); err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}

	handoffPath := filepath.Join(os.Getenv("AETHER_ROOT"), ".aether", "HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte("# Colony Session — Paused Colony\n\n- Run `aether continue`\n"), 0644); err != nil {
		t.Fatalf("failed to seed handoff: %v", err)
	}

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	if !strings.Contains(buf.String(), `"resumed":true`) {
		t.Fatalf("expected resumed:true JSON, got: %s", buf.String())
	}

	if _, err := os.Stat(handoffPath); !os.IsNotExist(err) {
		t.Fatalf("expected handoff to be removed, stat err=%v", err)
	}

	var updated colony.SessionFile
	if err := store.LoadJSON("session.json", &updated); err != nil {
		t.Fatalf("failed to reload session: %v", err)
	}
	if updated.LastCommand != "resume-colony" {
		t.Fatalf("session.LastCommand = %q, want resume-colony", updated.LastCommand)
	}
	if updated.ContextCleared {
		t.Fatal("expected ContextCleared to be false after resume")
	}
	if updated.ResumedAt == nil || *updated.ResumedAt == "" {
		t.Fatal("expected ResumedAt to be populated")
	}
}

func TestResumeDashboardShowsNextPlannedPhaseAndSessionTodos(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Keep the plan after context clears"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Planned phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Implement the durable resume path", Status: colony.TaskPending}},
				},
			},
		},
	})

	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID:      "resume-dashboard",
		StartedAt:      "2026-04-15T10:00:00Z",
		ColonyGoal:     goal,
		SuggestedNext:  "aether build 1",
		ContextCleared: true,
		Summary:        "Plan persisted for later recovery",
	}); err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}

	result := buildResumeDashboardResult()
	nextPhase, ok := result["next_phase"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected next_phase in resume dashboard, got %v", result)
	}
	if got := intValue(nextPhase["id"]); got != 1 {
		t.Fatalf("next phase id = %d, want 1", got)
	}

	session, ok := result["session"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected session block in resume dashboard, got %v", result)
	}
	if summary := stringValue(session["summary"]); summary != "Plan persisted for later recovery" {
		t.Fatalf("summary = %q, want seeded summary", summary)
	}
	todos := stringSliceValue(session["active_todos"])
	if len(todos) != 1 || todos[0] != "Implement the durable resume path" {
		t.Fatalf("active_todos = %v, want seeded phase task", todos)
	}
}

func TestResumeDashboardRestoresLegacySessionMirror(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Resume dashboard recovery"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Recovered phase", Status: colony.PhaseReady},
			},
		},
	})

	aetherRoot := os.Getenv("AETHER_ROOT")
	legacyRoot := filepath.Join(aetherRoot, ".aether", "data", "colonies")
	if err := os.MkdirAll(filepath.Join(legacyRoot, "older-colony"), 0755); err != nil {
		t.Fatalf("create older colony dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(legacyRoot, "resume-dashboard-recovery"), 0755); err != nil {
		t.Fatalf("create matching colony dir: %v", err)
	}

	writeLegacy := func(path string, session colony.SessionFile) {
		t.Helper()
		data, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			t.Fatalf("marshal session: %v", err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
			t.Fatalf("write legacy session: %v", err)
		}
	}
	writeLegacy(filepath.Join(legacyRoot, "older-colony", "session.json"), colony.SessionFile{
		SessionID:     "older",
		ColonyGoal:    "Older colony",
		Summary:       "stale",
		SuggestedNext: "aether init \"other\"",
	})
	writeLegacy(filepath.Join(legacyRoot, "resume-dashboard-recovery", "session.json"), colony.SessionFile{
		SessionID:      "matching",
		ColonyGoal:     goal,
		Summary:        "Recovered from colony-scoped session",
		SuggestedNext:  "aether entomb",
		ContextCleared: true,
	})

	result := buildResumeDashboardResult()
	sessionBlock, ok := result["session"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected session block, got %v", result)
	}
	if summary := stringValue(sessionBlock["summary"]); summary != "Recovered from colony-scoped session" {
		t.Fatalf("summary = %q, want restored legacy session summary", summary)
	}

	var mirrored colony.SessionFile
	if err := store.LoadJSON("session.json", &mirrored); err != nil {
		t.Fatalf("expected top-level session mirror to be restored: %v", err)
	}
	if mirrored.SessionID != "matching" {
		t.Fatalf("restored session id = %q, want matching", mirrored.SessionID)
	}
}
