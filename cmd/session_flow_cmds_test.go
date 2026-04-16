package cmd

import (
	"bytes"
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
	for _, want := range []string{"# Colony Handoff", "Pause this colony cleanly", "Implement pause-colony", "aether resume-colony"} {
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
	if err := os.WriteFile(handoffPath, []byte("# Colony Handoff\n\nNext: aether continue\n"), 0644); err != nil {
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
