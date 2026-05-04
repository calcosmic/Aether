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

// ---------------------------------------------------------------------------
// TestCreateBackup
// ---------------------------------------------------------------------------

func TestCreateBackup(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create some data files
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    stringPtr("test goal"),
		State:   colony.StateREADY,
	})
	writeFile(t, dataDir, "trace.jsonl", []byte(`{"id":"trc-1"}`+"\n"))
	writeJSONFile(t, dataDir, "midden/midden.json", colony.MiddenFile{Version: "1.0"})

	backupPath, err := createBackup(dataDir)
	if err != nil {
		t.Fatalf("createBackup failed: %v", err)
	}

	// Verify backup directory exists
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup dir not found: %v", err)
	}

	// Verify files were copied
	for _, f := range []string{"COLONY_STATE.json", "trace.jsonl", "midden/midden.json"} {
		if _, err := os.Stat(filepath.Join(backupPath, f)); err != nil {
			t.Errorf("backup missing file: %s", f)
		}
	}

	// Verify backup manifest exists
	manifestPath := filepath.Join(backupPath, "_backup_manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("backup manifest not found: %v", err)
	}
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("manifest not valid JSON: %v", err)
	}
	if manifest["source_path"] != dataDir {
		t.Errorf("manifest source_path = %v, want %v", manifest["source_path"], dataDir)
	}
}

// ---------------------------------------------------------------------------
// TestCleanupOldBackups
// ---------------------------------------------------------------------------

func TestCleanupOldBackups(t *testing.T) {
	dir := t.TempDir()
	backupsDir := filepath.Join(dir, "backups")

	// Create 5 backup directories
	for i := 0; i < 5; i++ {
		timestamp := time.Now().AddDate(0, 0, -(4 - i)).UTC().Format("20060102-150405")
		path := filepath.Join(backupsDir, "medic-"+timestamp)
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("create backup %d: %v", i, err)
		}
	}

	// Keep only 2 most recent
	if err := cleanupOldBackups(backupsDir, 2); err != nil {
		t.Fatalf("cleanupOldBackups failed: %v", err)
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		t.Fatalf("read backups dir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 backups after cleanup, got %d", len(entries))
	}
}

// ---------------------------------------------------------------------------
// TestRepairOrphanedWorktrees
// ---------------------------------------------------------------------------

func TestRepairOrphanedWorktrees(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	goal := "test goal"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Worktrees: []colony.WorktreeEntry{
			{ID: "wt-1", Status: colony.WorktreeOrphaned, Path: "/nonexistent/path/1"},
			{ID: "wt-2", Status: colony.WorktreeMerged},
			{ID: "wt-3", Status: colony.WorktreeOrphaned, Path: "/nonexistent/path/3"},
		},
	})

	issue := HealthIssue{
		Severity: "warning",
		Category: "state",
		File:     "COLONY_STATE.json",
		Message:  "2 orphaned worktree entries",
		Fixable:  true,
	}

	record := repairStateIssues(issue, MedicOptions{Fix: true}, dataDir)
	if !record.Success {
		t.Fatalf("repair failed: %s", record.Error)
	}
	if record.Action != "remove_orphaned_worktrees" {
		t.Errorf("action = %q, want %q", record.Action, "remove_orphaned_worktrees")
	}

	// Verify orphaned entries were removed
	data, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse state: %v", err)
	}
	if len(state.Worktrees) != 1 {
		t.Errorf("expected 1 worktree remaining, got %d", len(state.Worktrees))
	}
	if len(state.Worktrees) > 0 && state.Worktrees[0].ID != "wt-2" {
		t.Errorf("expected wt-2 to remain, got %s", state.Worktrees[0].ID)
	}
}

// ---------------------------------------------------------------------------
// TestRepairExpiredPheromones
// ---------------------------------------------------------------------------

func TestRepairExpiredPheromones(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	expired := time.Now().AddDate(0, 0, -1).UTC().Format(time.RFC3339)
	future := time.Now().AddDate(0, 0, 30).UTC().Format(time.RFC3339)
	expiredStr := expired
	futureStr := future

	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig-expired-1", Type: "FOCUS", Active: true, ExpiresAt: &expiredStr, Content: json.RawMessage(`{"text":"a"}`)},
			{ID: "sig-expired-2", Type: "REDIRECT", Active: true, ExpiresAt: &expiredStr, Content: json.RawMessage(`{"text":"b"}`)},
			{ID: "sig-active", Type: "FOCUS", Active: true, ExpiresAt: &futureStr, Content: json.RawMessage(`{"text":"c"}`)},
		},
	})

	issue := HealthIssue{
		Severity: "warning",
		Category: "pheromone",
		File:     "pheromones.json",
		Message:  "Signal 'sig-expired-1' has expired but is still active",
		Fixable:  true,
	}

	record := repairPheromoneIssues(issue, MedicOptions{Fix: true}, dataDir)
	if !record.Success {
		t.Fatalf("repair failed: %s", record.Error)
	}
	if record.Action != "deactivate_expired_signals" {
		t.Errorf("action = %q, want %q", record.Action, "deactivate_expired_signals")
	}

	// Verify expired signals deactivated
	data, err := os.ReadFile(filepath.Join(dataDir, "pheromones.json"))
	if err != nil {
		t.Fatalf("read pheromones: %v", err)
	}
	var pheromones colony.PheromoneFile
	if err := json.Unmarshal(data, &pheromones); err != nil {
		t.Fatalf("parse pheromones: %v", err)
	}

	for _, sig := range pheromones.Signals {
		if strings.HasPrefix(sig.ID, "sig-expired") && sig.Active {
			t.Errorf("expired signal %s should be inactive", sig.ID)
		}
		if sig.ID == "sig-active" && !sig.Active {
			t.Error("active signal should still be active")
		}
	}
}

// ---------------------------------------------------------------------------
// TestRepairLegacyState
// ---------------------------------------------------------------------------

func TestRepairLegacyState(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	goal := "test goal"
	// Write state with a deprecated state value (PAUSED)
	writeJSONFile(t, dataDir, "COLONY_STATE.json", map[string]interface{}{
		"version": "3.0",
		"goal":    goal,
		"state":   "PAUSED",
		"plan": map[string]interface{}{
			"phases": []interface{}{},
		},
	})

	issue := HealthIssue{
		Severity: "warning",
		Category: "state",
		File:     "COLONY_STATE.json",
		Message:  "State has deprecated value",
		Fixable:  true,
	}

	record := repairStateIssues(issue, MedicOptions{Fix: true}, dataDir)
	if !record.Success {
		t.Fatalf("repair failed: %s", record.Error)
	}
	if record.Action != "normalize_legacy_state" {
		t.Errorf("action = %q, want %q", record.Action, "normalize_legacy_state")
	}

	// Verify state was normalized
	data, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse state: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Errorf("expected READY, got %s", state.State)
	}
	if !state.Paused {
		t.Error("expected paused flag to be set")
	}
}

func TestRepairInvalidStateNormalizesRuntimeDrift(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	goal := "Repair invalid production state"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", map[string]interface{}{
		"version":       "3.0",
		"goal":          goal,
		"state":         "done",
		"current_phase": 1,
		"plan": map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{"id": 1, "name": "Complete", "status": "completed"},
			},
		},
	})

	issue := HealthIssue{
		Severity: "critical",
		Category: "state",
		File:     "COLONY_STATE.json",
		Message:  "Invalid state 'done'",
		Fixable:  true,
	}

	record := repairStateIssues(issue, MedicOptions{Fix: true}, dataDir)
	if !record.Success {
		t.Fatalf("repair failed: %s", record.Error)
	}
	if record.Action != "normalize_invalid_state" {
		t.Errorf("action = %q, want %q", record.Action, "normalize_invalid_state")
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse state: %v", err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Fatalf("state = %q, want COMPLETED", state.State)
	}
}

// ---------------------------------------------------------------------------
// TestRepairDeprecatedSignals
// ---------------------------------------------------------------------------

func TestRepairDeprecatedSignals(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	goal := "test goal"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Signals: []colony.Signal{
			{ID: "s1", Type: "FOCUS", Content: "focus here", Active: true},
			{ID: "s2", Type: "REDIRECT", Content: "avoid that", Active: false},
		},
	})

	issue := HealthIssue{
		Severity: "warning",
		Category: "state",
		File:     "COLONY_STATE.json",
		Message:  "deprecated 'signals' field has 2 entries -- should be migrated to pheromones.json",
		Fixable:  true,
	}

	record := repairStateIssues(issue, MedicOptions{Fix: true}, dataDir)
	if !record.Success {
		t.Fatalf("repair failed: %s", record.Error)
	}
	if record.Action != "migrate_deprecated_signals" {
		t.Errorf("action = %q, want %q", record.Action, "migrate_deprecated_signals")
	}

	// Verify signals were cleared
	data, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse state: %v", err)
	}
	if len(state.Signals) != 0 {
		t.Errorf("expected 0 signals, got %d", len(state.Signals))
	}
}

// ---------------------------------------------------------------------------
// TestRepairSessionMismatch
// ---------------------------------------------------------------------------

func TestRepairSessionMismatch(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	goal := "shared goal"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 3,
	})
	writeJSONFile(t, dataDir, "session.json", colony.SessionFile{
		SessionID:     "sess-mismatch",
		ColonyGoal:    goal,
		CurrentPhase:  1,
		LastCommandAt: time.Now().UTC().Format(time.RFC3339),
	})

	issue := HealthIssue{
		Severity: "warning",
		Category: "session",
		File:     "session.json",
		Message:  "session.json current_phase (1) doesn't match COLONY_STATE (3)",
		Fixable:  true,
	}

	record := repairSessionIssues(issue, MedicOptions{Fix: true}, dataDir)
	if !record.Success {
		t.Fatalf("repair failed: %s", record.Error)
	}
	if record.Action != "fix_phase_mismatch" {
		t.Errorf("action = %q, want %q", record.Action, "fix_phase_mismatch")
	}

	// Verify session phase was updated
	data, err := os.ReadFile(filepath.Join(dataDir, "session.json"))
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	var session colony.SessionFile
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatalf("parse session: %v", err)
	}
	if session.CurrentPhase != 3 {
		t.Errorf("expected phase 3, got %d", session.CurrentPhase)
	}
}

// ---------------------------------------------------------------------------
// TestRepairCorruptedJSON
// ---------------------------------------------------------------------------

func TestRepairCorruptedJSON(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write a corrupted JSON file with valid content followed by garbage
	corrupted := `{"version": "3.0", "goal": "test"}GARBAGE_DATA_HERE`
	writeFile(t, dataDir, "instincts.json", []byte(corrupted))

	issue := HealthIssue{
		Severity: "critical",
		Category: "file",
		File:     "instincts.json",
		Message:  "instincts.json is corrupted: invalid JSON",
		Fixable:  true,
	}

	record := repairDataIssues(issue, MedicOptions{Fix: true, Force: true}, dataDir)
	if !record.Success {
		t.Fatalf("repair failed: %s", record.Error)
	}
	if record.Action != "recover_corrupted_json" {
		t.Errorf("action = %q, want %q", record.Action, "recover_corrupted_json")
	}

	// Verify file is now valid JSON
	data, err := os.ReadFile(filepath.Join(dataDir, "instincts.json"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !json.Valid(data) {
		t.Errorf("recovered file is not valid JSON: %s", string(data))
	}
}

// ---------------------------------------------------------------------------
// TestRepairCorruptedJSONRequiresForce
// ---------------------------------------------------------------------------

func TestRepairCorruptedJSONRequiresForce(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	corrupted := `{"version": "3.0", "goal": "test"}GARBAGE`
	writeFile(t, dataDir, "instincts.json", []byte(corrupted))

	issue := HealthIssue{
		Severity: "critical",
		Category: "file",
		File:     "instincts.json",
		Message:  "instincts.json is corrupted: invalid JSON",
		Fixable:  true,
	}

	// Without --force, should fail with error
	record := repairDataIssues(issue, MedicOptions{Fix: true, Force: false}, dataDir)
	if record.Success {
		t.Error("expected repair to fail without --force")
	}
	if !strings.Contains(record.Error, "requires --force") {
		t.Errorf("expected force-required error, got: %s", record.Error)
	}

	// Verify file was NOT modified
	data, _ := os.ReadFile(filepath.Join(dataDir, "instincts.json"))
	if json.Valid(data) {
		t.Error("file should not have been modified without --force")
	}
}

// ---------------------------------------------------------------------------
// TestRepairStaleSpawnState
// ---------------------------------------------------------------------------

func TestRepairStaleSpawnState(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create spawn-runs with a stale running entry
	staleTime := time.Now().AddDate(0, 0, -2).UTC().Format(time.RFC3339)
	spawnData := map[string]interface{}{
		"current_run_id": "run-stale",
		"runs": []interface{}{
			map[string]interface{}{
				"id":         "run-stale",
				"started_at": staleTime,
				"status":     "running",
			},
			map[string]interface{}{
				"id":         "run-done",
				"started_at": staleTime,
				"status":     "completed",
			},
		},
	}
	writeJSONFile(t, dataDir, "spawn-runs.json", spawnData)

	issue := HealthIssue{
		Severity: "warning",
		Category: "data",
		File:     "spawn-runs.json",
		Message:  "spawn state stale",
		Fixable:  true,
	}

	record := repairDataIssues(issue, MedicOptions{Fix: true}, dataDir)
	if !record.Success {
		t.Fatalf("repair failed: %s", record.Error)
	}
	if record.Action != "reset_stale_spawn_state" {
		t.Errorf("action = %q, want %q", record.Action, "reset_stale_spawn_state")
	}

	// Verify stale run was reset
	data, err := os.ReadFile(filepath.Join(dataDir, "spawn-runs.json"))
	if err != nil {
		t.Fatalf("read spawn-runs: %v", err)
	}
	var spawnResult struct {
		CurrentRunID string `json:"current_run_id"`
		Runs         []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(data, &spawnResult); err != nil {
		t.Fatalf("parse spawn-runs: %v", err)
	}
	if spawnResult.CurrentRunID != "" {
		t.Errorf("expected empty current_run_id, got %q", spawnResult.CurrentRunID)
	}
	for _, run := range spawnResult.Runs {
		if run.ID == "run-stale" && run.Status != "failed" {
			t.Errorf("expected stale run to be failed, got %q", run.Status)
		}
	}
}

// ---------------------------------------------------------------------------
// TestLogRepairToTrace
// ---------------------------------------------------------------------------

func TestLogRepairToTrace(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	record := RepairRecord{
		Category: "state",
		File:     "COLONY_STATE.json",
		Action:   "remove_orphaned_worktrees",
		Before:   "2 worktrees",
		After:    "1 worktree",
		Success:  true,
	}

	logRepairToTrace(record, dataDir)

	// Verify trace file was created
	tracePath := filepath.Join(dataDir, "trace.jsonl")
	data, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `"level":"intervention"`) {
		t.Error("trace entry missing intervention level")
	}
	if !strings.Contains(content, `"topic":"medic.repair"`) {
		t.Error("trace entry missing medic.repair topic")
	}
	if !strings.Contains(content, `"action":"remove_orphaned_worktrees"`) {
		t.Error("trace entry missing action")
	}
}

// ---------------------------------------------------------------------------
// TestPerformRepairsIntegration
// ---------------------------------------------------------------------------

func TestPerformRepairsIntegration(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("AETHER_ROOT", dir)

	goal := "integration test"
	expired := time.Now().AddDate(0, 0, -1).UTC().Format(time.RFC3339)
	expiredStr := expired

	// Create state with orphaned worktrees
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Worktrees: []colony.WorktreeEntry{
			{ID: "wt-1", Status: colony.WorktreeOrphaned, Path: "/nonexistent"},
		},
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Phase 1", Status: colony.PhasePending},
		}},
		Events: []string{"2026-04-21T10:00:00Z|info|test|hello"},
	})

	// Create pheromones with expired signals
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig-exp", Type: "FOCUS", Active: true, ExpiresAt: &expiredStr, Content: json.RawMessage(`{"text":"a"}`)},
		},
	})

	// Create session matching state
	writeJSONFile(t, dataDir, "session.json", colony.SessionFile{
		SessionID:     "int-sess",
		ColonyGoal:    goal,
		CurrentPhase:  0,
		LastCommandAt: time.Now().UTC().Format(time.RFC3339),
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
	})

	// Other required files
	writeJSONFile(t, dataDir, "instincts.json", map[string]string{"version": "1.0"})
	writeJSONFile(t, dataDir, "midden/midden.json", map[string]string{"version": "1.0"})
	writeJSONFile(t, dataDir, "learning-observations.json", map[string]interface{}{})
	writeJSONFile(t, dataDir, "assumptions.json", map[string]string{"version": "1.0"})
	writeJSONFile(t, dataDir, "pending-decisions.json", map[string]string{"version": "1.0"})
	writeFile(t, dataDir, "constraints.json", []byte("{}\n"))

	// Run scan
	opts := MedicOptions{Fix: false, Deep: false}
	scanResult, err := performHealthScan(opts)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Count fixable issues
	fixableCount := 0
	for _, issue := range scanResult.Issues {
		if issue.Fixable {
			fixableCount++
		}
	}
	if fixableCount == 0 {
		t.Fatal("expected at least one fixable issue from scan")
	}

	// Run repairs
	opts = MedicOptions{Fix: true, Deep: false}
	repairResult, err := performRepairs(scanResult, opts, dataDir)
	if err != nil {
		t.Fatalf("performRepairs failed: %v", err)
	}

	if repairResult.Attempted == 0 {
		t.Error("expected at least one repair attempt")
	}

	t.Logf("Repairs: attempted=%d, succeeded=%d, failed=%d, skipped=%d",
		repairResult.Attempted, repairResult.Succeeded,
		repairResult.Failed, repairResult.Skipped)

	// Verify backup was created
	backupsDir := filepath.Join(dir, ".aether", "backups")
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		t.Fatalf("backups dir not found: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected backup to be created")
	}

	// Re-scan to verify fixes
	postResult, err := performHealthScan(MedicOptions{Fix: false, Deep: false})
	if err != nil {
		t.Fatalf("post-repair scan failed: %v", err)
	}

	// Count remaining fixable issues
	remainingFixable := 0
	for _, issue := range postResult.Issues {
		if issue.Fixable {
			remainingFixable++
		}
	}

	t.Logf("Before: %d fixable issues, After: %d fixable issues", fixableCount, remainingFixable)
	if remainingFixable >= fixableCount {
		t.Errorf("expected fewer fixable issues after repair")
	}
}

// ---------------------------------------------------------------------------
// TestRepairGhostConstraints
// ---------------------------------------------------------------------------

func TestRepairGhostConstraints(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, dataDir, "constraints.json", []byte(`{"focus_areas":["security"]}`))

	issue := HealthIssue{
		Severity: "warning",
		Category: "data",
		File:     "constraints.json",
		Message:  "constraints.json has content but Go code ignores it (ghost file)",
		Fixable:  true,
	}

	record := repairDataIssues(issue, MedicOptions{Fix: true}, dataDir)
	if !record.Success {
		t.Fatalf("repair failed: %s", record.Error)
	}
	if record.Action != "reset_ghost_constraints" {
		t.Errorf("action = %q, want %q", record.Action, "reset_ghost_constraints")
	}

	data, _ := os.ReadFile(filepath.Join(dataDir, "constraints.json"))
	trimmed := strings.TrimSpace(string(data))
	if trimmed != "{}" {
		t.Errorf("expected constraints to be reset to {}, got: %s", trimmed)
	}
}

// ---------------------------------------------------------------------------
// TestRepairMissingPheromoneID
// ---------------------------------------------------------------------------

func TestRepairMissingPheromoneID(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "", Type: "FOCUS", Active: true, Content: json.RawMessage(`{"text":"test"}`)},
			{ID: "sig-existing", Type: "FOCUS", Active: true, Content: json.RawMessage(`{"text":"existing"}`)},
		},
	})

	issue := HealthIssue{
		Severity: "warning",
		Category: "pheromone",
		File:     "pheromones.json",
		Message:  "Signal missing ID at index 0",
		Fixable:  true,
	}

	record := repairPheromoneIssues(issue, MedicOptions{Fix: true}, dataDir)
	if !record.Success {
		t.Fatalf("repair failed: %s", record.Error)
	}
	if record.Action != "assign_missing_ids" {
		t.Errorf("action = %q, want %q", record.Action, "assign_missing_ids")
	}

	// Verify ID was assigned
	data, err := os.ReadFile(filepath.Join(dataDir, "pheromones.json"))
	if err != nil {
		t.Fatalf("read pheromones: %v", err)
	}
	var pheromones colony.PheromoneFile
	if err := json.Unmarshal(data, &pheromones); err != nil {
		t.Fatalf("parse pheromones: %v", err)
	}
	if pheromones.Signals[0].ID == "" {
		t.Error("expected ID to be assigned")
	}
	if !strings.HasPrefix(pheromones.Signals[0].ID, "sig_") {
		t.Errorf("expected sig_ prefix, got: %s", pheromones.Signals[0].ID)
	}
}

// ---------------------------------------------------------------------------
// TestPerformRepairsNoFix
// ---------------------------------------------------------------------------

func TestPerformRepairsNoFix(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	scanResult := &ScannerResult{
		Issues: []HealthIssue{
			{Severity: "warning", Category: "state", Message: "test", Fixable: true},
		},
	}

	// Without --fix, should return nil
	result, err := performRepairs(scanResult, MedicOptions{Fix: false}, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when Fix=false")
	}
}

// ---------------------------------------------------------------------------
// TestRepairInvalidPheromoneType
// ---------------------------------------------------------------------------

func TestRepairInvalidPheromoneType(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig-bad", Type: "INVALID", Active: true, Content: json.RawMessage(`{"text":"test"}`)},
			{ID: "sig-ok", Type: "FOCUS", Active: true, Content: json.RawMessage(`{"text":"ok"}`)},
		},
	})

	issue := HealthIssue{
		Severity: "warning",
		Category: "pheromone",
		File:     "pheromones.json",
		Message:  "Invalid signal type 'INVALID' at index 0",
		Fixable:  true,
	}

	record := repairPheromoneIssues(issue, MedicOptions{Fix: true}, dataDir)
	if !record.Success {
		t.Fatalf("repair failed: %s", record.Error)
	}
	if record.Action != "fix_invalid_signal_types" {
		t.Errorf("action = %q, want %q", record.Action, "fix_invalid_signal_types")
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "pheromones.json"))
	if err != nil {
		t.Fatalf("read pheromones: %v", err)
	}
	var pheromones colony.PheromoneFile
	if err := json.Unmarshal(data, &pheromones); err != nil {
		t.Fatalf("parse pheromones: %v", err)
	}
	if pheromones.Signals[0].Type != "FOCUS" {
		t.Errorf("expected type FOCUS, got %s", pheromones.Signals[0].Type)
	}
}

// ---------------------------------------------------------------------------
// TestFindLastValidJSON
// ---------------------------------------------------------------------------

func TestFindLastValidJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		want    string
	}{
		{"valid object", `{"key":"value"}`, false, `{"key":"value"}`},
		{"object with garbage", `{"key":"value"}garbage`, false, `{"key":"value"}`},
		{"valid array", `[1,2,3]`, false, `[1,2,3]`},
		{"completely invalid", `not json at all`, true, ""},
		{"truncated object no close", `{"key":"value"`, true, ""},
		{"truncated array no close", `[1,2,3`, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findLastValidJSON([]byte(tt.input))
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %s", string(result))
				}
				return
			}
			if result == nil {
				t.Fatalf("expected result, got nil")
			}
			trimmed := strings.TrimSpace(string(result))
			if trimmed != tt.want {
				t.Errorf("got %q, want %q", trimmed, tt.want)
			}
		})
	}
}

// stringPtr returns a pointer to a string value.
func stringPtr(s string) *string {
	return &s
}
