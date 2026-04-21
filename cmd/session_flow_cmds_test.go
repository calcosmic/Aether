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

func TestPauseColonyWritesHandoffAndSession(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Pause this colony cleanly"
	taskID := "task-1"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Milestone:      "Open Chambers",
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

	var paused colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &paused); err != nil {
		t.Fatalf("expected paused state to be written: %v", err)
	}
	if paused.State != colony.StateEXECUTING {
		t.Fatalf("paused lifecycle state = %q, want EXECUTING preserved", paused.State)
	}
	if !paused.Paused {
		t.Fatal("expected paused flag to be set")
	}
	if paused.PausedAt == nil || *paused.PausedAt == "" {
		t.Fatal("expected paused_at to be set")
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

	var resumed colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &resumed); err != nil {
		t.Fatalf("failed to reload resumed state: %v", err)
	}
	if resumed.Paused {
		t.Fatal("expected paused flag to be cleared on resume")
	}
	if resumed.PausedAt != nil {
		t.Fatalf("expected paused_at to be cleared on resume, got %v", *resumed.PausedAt)
	}
}

func TestResumeColonyRotatesStaleSpawnTreeForPausedColony(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Resume should clear ghost workers"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.State("PAUSED"),
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Execution", Status: colony.PhaseReady},
			},
		},
	})

	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID:      "resume-rotate-test",
		StartedAt:      "2026-04-15T10:00:00Z",
		ColonyGoal:     goal,
		CurrentPhase:   1,
		SuggestedNext:  "aether status",
		ContextCleared: true,
		Summary:        "Paused with stale worker history",
	}); err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}

	spawnTreePath := filepath.Join(dataDir, "spawn-tree.txt")
	if err := os.WriteFile(spawnTreePath, []byte("2026-04-18T10:00:00Z|Queen|builder|Ghost-41|old worker|1|spawned\n"), 0644); err != nil {
		t.Fatalf("failed to seed spawn tree: %v", err)
	}

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	data, err := os.ReadFile(spawnTreePath)
	if err != nil {
		t.Fatalf("failed to read rotated spawn tree: %v", err)
	}
	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("expected spawn-tree.txt to be rotated on resume, got:\n%s", string(data))
	}

	archiveDir := filepath.Join(dataDir, "spawn-tree-archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("expected spawn-tree archive dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected rotated spawn-tree archive to be created")
	}
}

func TestResumeColonyNormalizesLegacyPausedStateToReady(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Resume broken paused colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.State("PAUSED"),
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Recoverable phase", Status: colony.PhaseInProgress},
			},
		},
	})

	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID:      "legacy-paused",
		StartedAt:      "2026-04-15T10:00:00Z",
		ColonyGoal:     goal,
		CurrentPhase:   1,
		SuggestedNext:  "aether build 1",
		ContextCleared: true,
		Summary:        "Legacy paused colony",
	}); err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	var resumed colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &resumed); err != nil {
		t.Fatalf("failed to reload resumed state: %v", err)
	}
	if resumed.State != colony.StateREADY {
		t.Fatalf("state = %q, want READY", resumed.State)
	}
	if resumed.Paused {
		t.Fatal("expected paused flag to be cleared on resume")
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

// --- Session Freshness Tests (Phase 22) ---

func TestSessionFreshnessVerification(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	// Test 1: Fresh session
	now := time.Now().UTC().Format(time.RFC3339)
	goal := "Fresh test"
	session := colony.SessionFile{
		SessionID:      "test-fresh",
		StartedAt:      now,
		ColonyGoal:     goal,
		BaselineCommit: getGitHEAD(),
	}
	if err := store.SaveJSON("session.json", session); err != nil {
		t.Fatalf("save session: %v", err)
	}

	result := sessionVerifyFresh(store)
	if !result.Fresh {
		t.Errorf("session should be fresh: age=%v, gitMatch=%v", result.Age, result.GitMatch)
	}
	if result.SessionID != "test-fresh" {
		t.Errorf("session ID: got %q, want test-fresh", result.SessionID)
	}

	// Test 2: Stale session (48 hours old)
	staleTime := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
	session.StartedAt = staleTime
	if err := store.SaveJSON("session.json", session); err != nil {
		t.Fatalf("save stale session: %v", err)
	}

	result = sessionVerifyFresh(store)
	if result.Fresh {
		t.Error("48-hour session should be stale")
	}
	if result.Age < 47*time.Hour {
		t.Errorf("stale age too low: %v", result.Age)
	}

	// Test 3: Missing session
	if err := os.Remove(filepath.Join(dataDir, "session.json")); err != nil {
		t.Fatalf("remove session: %v", err)
	}
	result = sessionVerifyFresh(store)
	if result.Fresh {
		t.Error("missing session should not be fresh")
	}
}

func TestResumeVisualFreshnessWarning(t *testing.T) {
	// Test stale session shows warning
	result := map[string]interface{}{
		"freshness": map[string]interface{}{
			"fresh":      false,
			"age_hours":  "48.0",
			"git_match":  false,
			"git_check":  true,
			"session_id": "test",
		},
		"current": map[string]interface{}{
			"goal":    "Test goal",
			"state":   "ready",
			"phase":   1,
			"total_phases": 3,
		},
		"blockers": []string{},
		"signals": map[string]interface{}{"items": []string{}, "count": 0},
	}

	output := renderResumeVisual(result, "", true)
	if !strings.Contains(output, "⚠️ Session is 48.0 hours old") {
		t.Errorf("stale resume visual missing freshness warning\n%s", output)
	}

	// Test fresh session does NOT show warning
	result["freshness"] = map[string]interface{}{
		"fresh":      true,
		"age_hours":  "1.0",
		"git_match":  true,
		"git_check":  true,
		"session_id": "test",
	}
	output = renderResumeVisual(result, "", true)
	if strings.Contains(output, "⚠️") {
		t.Errorf("fresh resume visual should not show warning\n%s", output)
	}
}

func TestResumeColonyGCOphanedWorktrees(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Resume with worktree cleanup"
	now := time.Now().UTC().Format(time.RFC3339)
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Worktrees: []colony.WorktreeEntry{
			{ID: "wt-1", Branch: "test-branch", Path: ".aether/worktrees/test", Status: colony.WorktreeAllocated, Phase: 1, CreatedAt: now, UpdatedAt: now},
		},
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Execution", Status: colony.PhaseInProgress},
			},
		},
	})

	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID:  "resume-wt-test",
		StartedAt:  "2026-04-15T10:00:00Z",
		ColonyGoal: goal,
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	// Verify worktree entry was removed from state
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if len(state.Worktrees) > 0 {
		t.Errorf("expected worktree entry removed on resume, got %d", len(state.Worktrees))
	}

	// Verify resume output mentions worktree cleanup
	output := buf.String()
	if !strings.Contains(output, "worktree_gc") {
		t.Errorf("expected worktree_gc in resume output, got: %s", output)
	}
}

func TestResumeVisualWorktreeCleanup(t *testing.T) {
	result := map[string]interface{}{
		"freshness": map[string]interface{}{
			"fresh":     true,
			"age_hours": "1.0",
		},
		"current": map[string]interface{}{
			"goal":         "Test",
			"state":        "ready",
			"phase":        1,
			"total_phases": 2,
		},
		"worktree_gc": map[string]interface{}{
			"cleaned":  2,
			"orphaned": 1,
		},
		"blockers": []string{},
		"signals":  map[string]interface{}{"items": []string{}, "count": 0},
	}

	output := renderResumeVisual(result, "", true)
	if !strings.Contains(output, "2 stale worktree(s) cleaned up") {
		t.Errorf("missing cleaned worktree message\n%s", output)
	}
	if !strings.Contains(output, "1 worktree(s) could not be cleaned") {
		t.Errorf("missing orphaned worktree message\n%s", output)
	}
}
