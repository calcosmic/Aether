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
	"github.com/calcosmic/Aether/pkg/storage"
)

func resumeTestPhasesThrough(current int) []colony.Phase {
	phases := make([]colony.Phase, 0, current)
	for i := 1; i <= current; i++ {
		status := colony.PhaseCompleted
		name := "Earlier"
		if i == current {
			status = colony.PhaseInProgress
			name = "Current"
		}
		phases = append(phases, colony.Phase{ID: i, Name: name, Status: status})
	}
	return phases
}

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

func TestResumeColonyRestoresInvalidStateFromHandoffSnapshot(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Recover runtime state from handoff"
	taskID := "1.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Recovered execution",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Make resume repair runtime state", Status: colony.TaskPending}},
				},
				{ID: 2, Name: "Verification", Status: colony.PhaseReady},
			},
		},
	}
	createTestColonyState(t, dataDir, state)
	session := colony.SessionFile{
		SessionID:      "resume-snapshot",
		StartedAt:      "2026-04-15T10:00:00Z",
		ColonyGoal:     goal,
		CurrentPhase:   1,
		SuggestedNext:  "aether build 1",
		ContextCleared: true,
		Summary:        "Snapshot should repair corrupted state",
	}
	if err := store.SaveJSON("session.json", session); err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}
	if err := writeHandoffDocument(renderHandoffSnapshot(state, session, "Paused Colony", "aether build 1", session.Summary)); err != nil {
		t.Fatalf("failed to seed handoff: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "COLONY_STATE.json"), []byte(`{"test":true}`), 0644); err != nil {
		t.Fatalf("failed to corrupt state: %v", err)
	}

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	if !strings.Contains(buf.String(), `"state_recovered_from_handoff":true`) {
		t.Fatalf("expected handoff recovery marker, got: %s", buf.String())
	}

	var recovered colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &recovered); err != nil {
		t.Fatalf("failed to reload recovered state: %v", err)
	}
	if recovered.Goal == nil || *recovered.Goal != goal {
		t.Fatalf("goal = %v, want %q", recovered.Goal, goal)
	}
	if recovered.State != colony.StateREADY || recovered.CurrentPhase != 1 {
		t.Fatalf("state/current_phase = %s/%d, want READY/1", recovered.State, recovered.CurrentPhase)
	}
	if len(recovered.Plan.Phases) != 2 || recovered.Plan.Phases[0].Name != "Recovered execution" {
		t.Fatalf("plan phases not restored from snapshot: %+v", recovered.Plan.Phases)
	}
	if len(recovered.Plan.Phases[0].Tasks) != 1 || recovered.Plan.Phases[0].Tasks[0].Goal != "Make resume repair runtime state" {
		t.Fatalf("phase task not restored from snapshot: %+v", recovered.Plan.Phases[0].Tasks)
	}
}

func TestResumeColonyRestoresInvalidStateFromLegacyHandoff(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	handoff := `# Colony Session — Paused Colony

## Goal

- Legacy recovered colony

## Phase

- Current: 1/6 — First phase
- State: READY

## Tasks

- Rebuild runtime state

## Next Step

- Run ` + "`aether build 1`" + `
`
	if err := writeHandoffDocument(handoff); err != nil {
		t.Fatalf("failed to seed handoff: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "COLONY_STATE.json"), []byte(`{"test":true}`), 0644); err != nil {
		t.Fatalf("failed to corrupt state: %v", err)
	}

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	if !strings.Contains(buf.String(), `"state_recovered_from_handoff":true`) {
		t.Fatalf("expected legacy handoff recovery marker, got: %s", buf.String())
	}

	var recovered colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &recovered); err != nil {
		t.Fatalf("failed to reload recovered state: %v", err)
	}
	if recovered.Goal == nil || *recovered.Goal != "Legacy recovered colony" {
		t.Fatalf("goal = %v, want legacy recovered colony", recovered.Goal)
	}
	if recovered.State != colony.StateREADY || recovered.CurrentPhase != 1 {
		t.Fatalf("state/current_phase = %s/%d, want READY/1", recovered.State, recovered.CurrentPhase)
	}
	if len(recovered.Plan.Phases) != 6 {
		t.Fatalf("phase count = %d, want 6", len(recovered.Plan.Phases))
	}
	if recovered.Plan.Phases[0].Name != "First phase" {
		t.Fatalf("phase name = %q, want First phase", recovered.Plan.Phases[0].Name)
	}
	if len(recovered.Plan.Phases[0].Tasks) != 1 || recovered.Plan.Phases[0].Tasks[0].Goal != "Rebuild runtime state" {
		t.Fatalf("legacy task not restored: %+v", recovered.Plan.Phases[0].Tasks)
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
			"goal":         "Test goal",
			"state":        "ready",
			"phase":        1,
			"total_phases": 3,
		},
		"blockers": []string{},
		"signals":  map[string]interface{}{"items": []string{}, "count": 0},
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

// --- Stale FOCUS Pherrmone Detection Tests (Phase 63, Plan 03) ---

func TestResumeDetectsStaleFocusSignals(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Resume with stale pheromones"
	sourcePhase := 2
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 5,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{
			Phases: resumeTestPhasesThrough(5),
		},
	})

	// Create pheromones with a stale FOCUS signal from phase 2
	contentJSON, _ := json.Marshal(map[string]string{"text": "pay attention to auth"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:          "sig-stale-focus",
				Type:        "FOCUS",
				Priority:    "normal",
				Active:      true,
				Content:     contentJSON,
				SourcePhase: &sourcePhase,
			},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("seed pheromones: %v", err)
	}

	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID: "stale-test",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"stale_signals"`) {
		t.Fatalf("expected stale_signals in resume output\n%s", output)
	}
	if !strings.Contains(output, `"source_phase":2`) {
		t.Fatalf("expected source_phase:2 in stale signal data\n%s", output)
	}
	if !strings.Contains(output, `"current_phase":5`) {
		t.Fatalf("expected current_phase:5 in stale signal data\n%s", output)
	}
}

func TestResumeNoStaleWhenSourcePhaseMatchesCurrent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Resume with current pheromones"
	sourcePhase := 3
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 3,
		Plan: colony.Plan{
			Phases: resumeTestPhasesThrough(3),
		},
	})

	contentJSON, _ := json.Marshal(map[string]string{"text": "current focus"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:          "sig-current-focus",
				Type:        "FOCUS",
				Priority:    "normal",
				Active:      true,
				Content:     contentJSON,
				SourcePhase: &sourcePhase,
			},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("seed pheromones: %v", err)
	}

	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID: "no-stale-test",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `"stale_signals"`) {
		t.Fatalf("expected no stale_signals when source_phase matches current_phase\n%s", output)
	}
}

func TestResumeNilSourcePhaseNotFlagged(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Resume with old pheromone without source_phase"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 5,
		Plan: colony.Plan{
			Phases: resumeTestPhasesThrough(5),
		},
	})

	// FOCUS signal with nil SourcePhase (backward compat)
	contentJSON, _ := json.Marshal(map[string]string{"text": "old focus without phase"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:          "sig-no-phase",
				Type:        "FOCUS",
				Priority:    "normal",
				Active:      true,
				Content:     contentJSON,
				SourcePhase: nil, // nil = backward compat, should NOT be flagged
			},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("seed pheromones: %v", err)
	}

	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID: "nil-phase-test",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `"stale_signals"`) {
		t.Fatalf("expected no stale_signals for nil SourcePhase (backward compat)\n%s", output)
	}
}

func TestResumeOnlyFocusFlaggedNotRedirect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Resume with stale redirect"
	stalePhase := 1
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 5,
		Plan: colony.Plan{
			Phases: resumeTestPhasesThrough(5),
		},
	})

	contentJSON, _ := json.Marshal(map[string]string{"text": "avoid this"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:          "sig-stale-redirect",
				Type:        "REDIRECT",
				Priority:    "high",
				Active:      true,
				Content:     contentJSON,
				SourcePhase: &stalePhase,
			},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("seed pheromones: %v", err)
	}

	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID: "redirect-test",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `"stale_signals"`) {
		t.Fatalf("expected no stale_signals for REDIRECT signals (only FOCUS checked)\n%s", output)
	}
}

func TestResumeInactiveFocusNotFlagged(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Resume with inactive stale focus"
	stalePhase := 1
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 5,
		Plan: colony.Plan{
			Phases: resumeTestPhasesThrough(5),
		},
	})

	contentJSON, _ := json.Marshal(map[string]string{"text": "inactive focus"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:          "sig-inactive-focus",
				Type:        "FOCUS",
				Priority:    "normal",
				Active:      false, // inactive -- should NOT be flagged
				Content:     contentJSON,
				SourcePhase: &stalePhase,
			},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("seed pheromones: %v", err)
	}

	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID: "inactive-test",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `"stale_signals"`) {
		t.Fatalf("expected no stale_signals for inactive FOCUS signals\n%s", output)
	}
}

func TestResumeVisualStaleWarning(t *testing.T) {
	result := map[string]interface{}{
		"freshness": map[string]interface{}{
			"fresh":     true,
			"age_hours": "1.0",
		},
		"current": map[string]interface{}{
			"goal":         "Test",
			"state":        "ready",
			"phase":        5,
			"total_phases": 10,
		},
		"stale_signals": []map[string]interface{}{
			{"id": "sig-1", "type": "FOCUS", "content": "pay attention to auth", "source_phase": 2, "current_phase": 5},
		},
		"blockers": []string{},
		"signals":  map[string]interface{}{"items": []string{}, "count": 0},
	}

	output := renderResumeVisual(result, "", true)
	if !strings.Contains(output, "stale FOCUS") {
		t.Errorf("expected stale FOCUS warning in visual output\n%s", output)
	}
	if !strings.Contains(output, "Phase 2") {
		t.Errorf("expected source phase in visual output\n%s", output)
	}
	if !strings.Contains(output, "pay attention to auth") {
		t.Errorf("expected signal content in visual output\n%s", output)
	}
}

func TestResumeVisualNoWarningWhenNoStale(t *testing.T) {
	result := map[string]interface{}{
		"freshness": map[string]interface{}{
			"fresh":     true,
			"age_hours": "1.0",
		},
		"current": map[string]interface{}{
			"goal":         "Test",
			"state":        "ready",
			"phase":        3,
			"total_phases": 5,
		},
		"blockers": []string{},
		"signals":  map[string]interface{}{"items": []string{}, "count": 0},
	}

	output := renderResumeVisual(result, "", true)
	if strings.Contains(output, "stale FOCUS") {
		t.Errorf("expected no stale FOCUS warning when no stale signals\n%s", output)
	}
}

// --- Direct unit tests for detectStaleFocusSignals (Phase 66, Plan 02) ---

func TestDetectStaleFocusSignals_EqualPhaseNotFlagged(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	origStore := store
	store = s
	defer func() { store = origStore }()

	sourcePhase := 3
	contentJSON, _ := json.Marshal(map[string]string{"text": "current phase focus"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:          "sig-equal-phase",
				Type:        "FOCUS",
				Priority:    "normal",
				Active:      true,
				Content:     contentJSON,
				SourcePhase: &sourcePhase,
			},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("save pheromones: %v", err)
	}

	stale := detectStaleFocusSignals(store, 3)
	if len(stale) != 0 {
		t.Errorf("expected 0 stale signals when SourcePhase == currentPhase, got %d: %+v", len(stale), stale)
	}
}

func TestDetectStaleFocusSignals_FuturePhaseNotFlagged(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	origStore := store
	store = s
	defer func() { store = origStore }()

	sourcePhase := 5
	contentJSON, _ := json.Marshal(map[string]string{"text": "future phase focus"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:          "sig-future-phase",
				Type:        "FOCUS",
				Priority:    "normal",
				Active:      true,
				Content:     contentJSON,
				SourcePhase: &sourcePhase,
			},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("save pheromones: %v", err)
	}

	stale := detectStaleFocusSignals(store, 3)
	if len(stale) != 0 {
		t.Errorf("expected 0 stale signals when SourcePhase > currentPhase, got %d: %+v", len(stale), stale)
	}
}

func TestDetectStaleFocusSignals_PastPhaseFlagged(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	origStore := store
	store = s
	defer func() { store = origStore }()

	sourcePhase := 1
	contentJSON, _ := json.Marshal(map[string]string{"text": "old focus"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:          "sig-past-phase",
				Type:        "FOCUS",
				Priority:    "normal",
				Active:      true,
				Content:     contentJSON,
				SourcePhase: &sourcePhase,
			},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("save pheromones: %v", err)
	}

	stale := detectStaleFocusSignals(store, 3)
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale signal when SourcePhase < currentPhase, got %d", len(stale))
	}
	if stale[0].ID != "sig-past-phase" {
		t.Errorf("expected stale signal ID 'sig-past-phase', got '%s'", stale[0].ID)
	}
	if stale[0].SourcePhase != 1 {
		t.Errorf("expected SourcePhase 1, got %d", stale[0].SourcePhase)
	}
	if stale[0].CurrentPhase != 3 {
		t.Errorf("expected CurrentPhase 3, got %d", stale[0].CurrentPhase)
	}
}
