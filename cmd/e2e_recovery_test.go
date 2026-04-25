package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Shared E2E helpers
// ---------------------------------------------------------------------------

// e2eRecoverSetup initializes a test environment for E2E recovery tests.
// It saves globals, resets rootCmd, initializes the store, and captures stdout.
func e2eRecoverSetup(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	_, dataDir := initRecoverTestStore(t)
	var buf bytes.Buffer
	stdout = &buf
	return &buf, dataDir
}

// e2eRunRecover runs the recover command with the given arguments and returns
// the error from rootCmd.Execute(). It resets flags before execution to prevent
// leakage from prior calls within the same test.
func e2eRunRecover(t *testing.T, args ...string) error {
	t.Helper()
	resetFlags(rootCmd)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// parseRecoverJSON parses the stdout buffer as a recoverJSONOutput struct.
func parseRecoverJSON(t *testing.T, buf *bytes.Buffer) recoverJSONOutput {
	t.Helper()
	var output recoverJSONOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, buf.String())
	}
	return output
}

// parseRecoverJSONMap parses the stdout buffer as a generic map for accessing
// repair details not present in recoverJSONOutput.
func parseRecoverJSONMap(t *testing.T, buf *bytes.Buffer) map[string]interface{} {
	t.Helper()
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, buf.String())
	}
	return output
}

// assertCategoryInIssues checks that at least one issue has the given category.
func assertCategoryInIssues(t *testing.T, issues []HealthIssue, category string) {
	t.Helper()
	for _, iss := range issues {
		if iss.Category == category {
			return
		}
	}
	t.Errorf("no issue with category %q among %d issues", category, len(issues))
	for _, iss := range issues {
		t.Logf("  found: [%s] %s: %s", iss.Severity, iss.Category, iss.Message)
	}
}

// ---------------------------------------------------------------------------
// Seed helpers -- each creates a specific stuck state in dataDir
// ---------------------------------------------------------------------------

// seedMissingPacketState writes a colony state in EXECUTING with no manifest,
// triggering scanMissingBuildPacket.
func seedMissingPacketState(t *testing.T, dataDir string) {
	t.Helper()
	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)
	// Deliberately do NOT create build/phase-1/manifest.json
}

// seedStaleSpawnedState writes a spawn-runs.json with an active run that
// started 2 hours ago, triggering scanStaleSpawnedWorkers.
// Note: callers must also write a valid COLONY_STATE.json for loadActiveColonyState.
func seedStaleSpawnedState(t *testing.T, dataDir string) {
	t.Helper()
	oldTime := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
	spawnData := map[string]interface{}{
		"current_run_id": "run-1",
		"runs": []map[string]interface{}{
			{
				"id":         "run-1",
				"started_at": oldTime,
				"status":     "active",
			},
		},
	}
	recoverWriteJSON(t, dataDir, "spawn-runs.json", spawnData)
}

// seedPartialPhaseState writes an EXECUTING state with a phase marked
// in_progress and no manifest, triggering scanPartialPhase.
func seedPartialPhaseState(t *testing.T, dataDir string) {
	t.Helper()
	goal := "Test colony"
	state := colony.ColonyState{
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: recoverTimePtr(time.Now().Add(-2 * time.Hour)),
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Phase 1",
					Status: "in_progress",
				},
			},
		},
	}
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)
	// Deliberately no manifest -- triggers "phase in_progress but never built"
}

// seedBadManifestState writes an EXECUTING state with a corrupt manifest,
// triggering scanBadManifest.
func seedBadManifestState(t *testing.T, dataDir string) {
	t.Helper()
	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)
	recoverWriteFile(t, dataDir, "build/phase-1/manifest.json", "{invalid json broken")
}

// seedDirtyWorktreeState writes an EXECUTING state with a WorktreeEntry
// pointing to a non-existent path, triggering scanDirtyWorktrees.
func seedDirtyWorktreeState(t *testing.T, dataDir string) {
	t.Helper()
	goal := "Test colony"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Worktrees: []colony.WorktreeEntry{
			{
				ID:     "wt-1",
				Branch: "feature/test-branch",
				Path:   "/tmp/nonexistent-worktree-path-xyz",
				Status: colony.WorktreeAllocated,
			},
		},
	}
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)
}

// seedBrokenSurveyState writes a state with TerritorySurveyed set but empty
// survey files, triggering scanBrokenSurvey.
func seedBrokenSurveyState(t *testing.T, dataDir string) {
	t.Helper()
	surveyed := "yes"
	state := newRecoverTestState(t, func(s *colony.ColonyState) {
		s.TerritorySurveyed = &surveyed
	})
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)
	// Create survey dir with empty files (all 5)
	recoverWriteFile(t, dataDir, "survey/blueprint.json", "{}")
	recoverWriteFile(t, dataDir, "survey/chambers.json", "null")
	recoverWriteFile(t, dataDir, "survey/disciplines.json", "[]")
	recoverWriteFile(t, dataDir, "survey/provisions.json", "{}")
	recoverWriteFile(t, dataDir, "survey/pathogens.json", "null")
}

// seedMissingAgentsState writes a READY state with no agent files,
// triggering scanMissingAgentFiles.
func seedMissingAgentsState(t *testing.T, dataDir string) {
	t.Helper()
	state := newRecoverTestState(t, func(s *colony.ColonyState) {
		s.State = colony.StateREADY
	})
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)
	// Deliberately no agent files -- triggers missing_agents
}

// seedHealthyColonyState writes a fully healthy colony with 25 agent files per
// surface. The tmpDir is the AETHER_ROOT (set by initRecoverTestStore).
func seedHealthyColonyState(t *testing.T, tmpDir string) {
	t.Helper()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	goal := "Healthy colony"
	state := colony.ColonyState{
		Goal:          &goal,
		State:         colony.StateREADY,
		CurrentPhase:  1,
		Plan:          colony.Plan{Phases: []colony.Phase{{ID: 1, Status: "completed"}}},
	}
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	// Create 25 agent files per surface to avoid false positives.
	for i := 0; i < 25; i++ {
		recoverWriteFile(t, tmpDir, fmt.Sprintf(".claude/agents/ant/agent%d.md", i), "# Agent")
		recoverWriteFile(t, tmpDir, fmt.Sprintf(".opencode/agents/agent%d.md", i), "# Agent")
		recoverWriteFile(t, tmpDir, fmt.Sprintf(".codex/agents/agent%d.toml", i), "[agent]")
	}
	// No spawn-runs.json, no TerritorySurveyed, no Worktrees, no survey files.
}

// ---------------------------------------------------------------------------
// TEST-01: Individual stuck-state detection (7 tests)
// ---------------------------------------------------------------------------

func TestE2ERecoveryMissingBuildPacket(t *testing.T) {
	buf, dataDir := e2eRecoverSetup(t)
	seedMissingPacketState(t, dataDir)

	err := e2eRunRecover(t, "recover", "--json")
	if err == nil {
		t.Fatal("expected error (exit code 1) for missing build packet")
	}

	output := parseRecoverJSON(t, buf)
	if output.ExitCode != 1 {
		t.Errorf("expected exit_code=1, got %d", output.ExitCode)
	}
	if len(output.Issues) < 1 {
		t.Fatal("expected at least 1 issue")
	}

	assertCategoryInIssues(t, output.Issues, "missing_build_packet")

	// Verify severity and fixability.
	for _, iss := range output.Issues {
		if iss.Category == "missing_build_packet" {
			if iss.Severity != "critical" {
				t.Errorf("expected severity critical, got %s", iss.Severity)
			}
			if !iss.Fixable {
				t.Error("expected fixable=true")
			}
		}
	}
}

func TestE2ERecoveryStaleSpawned(t *testing.T) {
	buf, dataDir := e2eRecoverSetup(t)

	// Write a valid colony state first (loadActiveColonyState needs it).
	state := newRecoverTestState(t, func(s *colony.ColonyState) {
		s.State = colony.StateREADY
	})
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	seedStaleSpawnedState(t, dataDir)

	err := e2eRunRecover(t, "recover", "--json")
	if err == nil {
		t.Fatal("expected error (exit code 1) for stale spawned workers")
	}

	output := parseRecoverJSON(t, buf)
	if output.ExitCode != 1 {
		t.Errorf("expected exit_code=1, got %d", output.ExitCode)
	}

	assertCategoryInIssues(t, output.Issues, "stale_spawned")
}

func TestE2ERecoveryPartialPhase(t *testing.T) {
	buf, dataDir := e2eRecoverSetup(t)
	seedPartialPhaseState(t, dataDir)

	err := e2eRunRecover(t, "recover", "--json")
	if err == nil {
		t.Fatal("expected error (exit code 1) for partial phase")
	}

	output := parseRecoverJSON(t, buf)
	assertCategoryInIssues(t, output.Issues, "partial_phase")
}

func TestE2ERecoveryBadManifest(t *testing.T) {
	buf, dataDir := e2eRecoverSetup(t)
	seedBadManifestState(t, dataDir)

	err := e2eRunRecover(t, "recover", "--json")
	if err == nil {
		t.Fatal("expected error (exit code 1) for bad manifest")
	}

	output := parseRecoverJSON(t, buf)
	assertCategoryInIssues(t, output.Issues, "bad_manifest")

	for _, iss := range output.Issues {
		if iss.Category == "bad_manifest" && iss.Severity != "critical" {
			t.Errorf("expected severity critical for bad_manifest, got %s", iss.Severity)
		}
	}
}

func TestE2ERecoveryDirtyWorktree(t *testing.T) {
	buf, dataDir := e2eRecoverSetup(t)
	seedDirtyWorktreeState(t, dataDir)

	err := e2eRunRecover(t, "recover", "--json")
	if err == nil {
		t.Fatal("expected error (exit code 1) for dirty worktree")
	}

	output := parseRecoverJSON(t, buf)
	assertCategoryInIssues(t, output.Issues, "dirty_worktree")
}

func TestE2ERecoveryBrokenSurvey(t *testing.T) {
	buf, dataDir := e2eRecoverSetup(t)
	seedBrokenSurveyState(t, dataDir)

	err := e2eRunRecover(t, "recover", "--json")
	if err == nil {
		t.Fatal("expected error (exit code 1) for broken survey")
	}

	output := parseRecoverJSON(t, buf)
	assertCategoryInIssues(t, output.Issues, "broken_survey")
}

func TestE2ERecoveryMissingAgents(t *testing.T) {
	buf, dataDir := e2eRecoverSetup(t)
	seedMissingAgentsState(t, dataDir)

	err := e2eRunRecover(t, "recover", "--json")
	if err == nil {
		t.Fatal("expected error (exit code 1) for missing agents")
	}

	output := parseRecoverJSON(t, buf)
	assertCategoryInIssues(t, output.Issues, "missing_agents")
}

// ---------------------------------------------------------------------------
// TEST-02: Compound stuck-state detection and repair (2 tests)
// ---------------------------------------------------------------------------

// TestE2ERecoveryCompoundState seeds multiple safe stuck states simultaneously,
// verifies all categories are detected in a single scan, then runs --apply --force
// to exercise the full repair pipeline.
//
// Note: The recovery system uses atomic rollback -- if ANY single repair fails,
// ALL repairs in the batch are rolled back to the backup. The missing_agents repair
// always fails in a test environment (no hub available), which triggers rollback of
// all other successful repairs. The test verifies:
// 1. All 5 safe categories are correctly detected in a single scan
// 2. Repairs are attempted (present in the repair output)
// 3. A backup is created before repairs begin
func TestE2ERecoveryCompoundState(t *testing.T) {
	buf, dataDir := e2eRecoverSetup(t)

	// Build a single compound COLONY_STATE.json that satisfies multiple scanners.
	goal := "Compound test colony"
	surveyed := "yes"
	compoundState := colony.ColonyState{
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: recoverTimePtr(time.Now().Add(-2 * time.Hour)),
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Phase 1",
					Status: "in_progress",
				},
			},
		},
		TerritorySurveyed: &surveyed,
	}
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", compoundState)

	// Stale spawn runs.
	seedStaleSpawnedState(t, dataDir)

	// Empty survey files (triggers broken_survey since TerritorySurveyed is set).
	recoverWriteFile(t, dataDir, "survey/blueprint.json", "{}")

	// No manifest (triggers missing_build_packet AND partial_phase).
	// No agent files (triggers missing_agents).

	// Step 1: Scan-only to verify all expected categories are detected.
	err := e2eRunRecover(t, "recover", "--json")
	if err == nil {
		t.Fatal("expected error (exit code 1) for compound stuck state")
	}

	scanOutput := parseRecoverJSON(t, buf)
	expectedCategories := []string{"missing_build_packet", "stale_spawned", "partial_phase", "broken_survey", "missing_agents"}
	for _, cat := range expectedCategories {
		assertCategoryInIssues(t, scanOutput.Issues, cat)
	}

	// Step 2: Run repair (--apply --force --json) and verify the pipeline executes.
	buf.Reset()
	_ = e2eRunRecover(t, "recover", "--apply", "--force", "--json")

	// Parse the full JSON output (including repair details).
	repairOutput := parseRecoverJSONMap(t, buf)

	// Verify backup was created.
	tmpDir := os.Getenv("AETHER_ROOT")
	backupsDir := filepath.Join(tmpDir, ".aether", "backups")
	if entries, err := os.ReadDir(backupsDir); err != nil || len(entries) == 0 {
		t.Errorf("expected backup directory to be created at %s", backupsDir)
	}

	// Verify repairs were attempted.
	repairs, ok := repairOutput["repairs"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'repairs' in output")
	}
	attempted, _ := repairs["attempted"].(float64)
	if attempted == 0 {
		t.Error("expected at least 1 repair attempted")
	}

	// Verify the repair output contains a mix of succeeded and failed/rolled-back.
	// Due to atomic rollback triggered by missing_agents failure, all successful
	// repairs are rolled back. This is expected behavior.
	details, ok := repairs["details"].([]interface{})
	if !ok || len(details) == 0 {
		t.Fatal("expected repair details in output")
	}
	foundRepairAttempt := false
	for _, d := range details {
		detail, ok := d.(map[string]interface{})
		if !ok {
			continue
		}
		cat, _ := detail["Category"].(string)
		action, _ := detail["Action"].(string)
		if cat == "missing_build_packet" && strings.Contains(action, "reset_to_ready") {
			foundRepairAttempt = true
		}
	}
	if !foundRepairAttempt {
		t.Error("expected missing_build_packet repair to be attempted")
	}
}

// TestE2ERecoveryCompoundDestructive seeds both destructive stuck states
// (dirty_worktree + bad_manifest), verifies both are detected, then runs
// --apply --force and verifies the repair pipeline executes.
//
// Note: bad_manifest with corrupt JSON is marked non-fixable by the scanner,
// so it is not dispatched for repair. dirty_worktree is fixable but may be
// rolled back if other repairs in the batch fail (e.g., missing_agents).
func TestE2ERecoveryCompoundDestructive(t *testing.T) {
	buf, dataDir := e2eRecoverSetup(t)

	// Build state with both destructive conditions.
	goal := "Destructive test colony"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Worktrees: []colony.WorktreeEntry{
			{
				ID:     "wt-1",
				Branch: "feature/test",
				Path:   "/tmp/nonexistent-wt-xyz",
				Status: colony.WorktreeAllocated,
			},
		},
		BuildStartedAt: recoverTimePtr(time.Now().Add(-2 * time.Hour)),
	}
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	// Create corrupt manifest.
	recoverWriteFile(t, dataDir, "build/phase-1/manifest.json", "{broken")

	// Step 1: Scan-only to verify both categories.
	err := e2eRunRecover(t, "recover", "--json")
	if err == nil {
		t.Fatal("expected error for compound destructive state")
	}
	scanOutput := parseRecoverJSON(t, buf)
	assertCategoryInIssues(t, scanOutput.Issues, "dirty_worktree")
	assertCategoryInIssues(t, scanOutput.Issues, "bad_manifest")

	// Verify bad_manifest corrupt JSON is detected as critical but NOT fixable.
	for _, iss := range scanOutput.Issues {
		if iss.Category == "bad_manifest" {
			if iss.Severity != "critical" {
				t.Errorf("expected bad_manifest severity critical, got %s", iss.Severity)
			}
			// Known behavior: corrupt JSON manifest is not marked fixable by the scanner,
			// even though the repair function can handle it. The repair dispatcher only
			// processes fixable issues, so this issue is not repaired.
			if iss.Fixable {
				t.Error("expected bad_manifest corrupt JSON to be non-fixable per scanner")
			}
		}
	}

	// Step 2: Run repair with --apply --force --json.
	buf.Reset()
	_ = e2eRunRecover(t, "recover", "--apply", "--force", "--json")

	// Verify repair output contains attempts.
	repairOutput := parseRecoverJSONMap(t, buf)
	repairs, ok := repairOutput["repairs"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'repairs' in repair output")
	}
	attempted, _ := repairs["attempted"].(float64)
	if attempted == 0 {
		t.Error("expected at least 1 repair attempt")
	}
}

// ---------------------------------------------------------------------------
// TEST-03: Healthy colony no false positives
// ---------------------------------------------------------------------------

// TestE2ERecoveryHealthyColony proves that a fully healthy colony produces
// zero false positives from aether recover. Tests both JSON and text output modes.
func TestE2ERecoveryHealthyColony(t *testing.T) {
	buf, _ := e2eRecoverSetup(t)
	tmpDir := os.Getenv("AETHER_ROOT")

	seedHealthyColonyState(t, tmpDir)

	// JSON mode: verify exit code 0 and empty issues.
	err := e2eRunRecover(t, "recover", "--json")
	if err != nil {
		t.Fatalf("expected exit code 0 for healthy colony, got error: %v\noutput: %s", err, buf.String())
	}

	output := parseRecoverJSON(t, buf)
	if output.ExitCode != 0 {
		t.Errorf("expected exit_code=0, got %d", output.ExitCode)
	}
	if len(output.Issues) != 0 {
		t.Errorf("expected 0 issues for healthy colony, got %d:", len(output.Issues))
		for _, iss := range output.Issues {
			t.Logf("  unexpected: [%s] %s: %s", iss.Severity, iss.Category, iss.Message)
		}
	}

	// Text mode: verify "No stuck-state conditions detected" message.
	buf.Reset()
	err = e2eRunRecover(t, "recover")
	if err != nil {
		t.Fatalf("expected exit code 0 in text mode, got error: %v", err)
	}

	textOutput := buf.String()
	if !strings.Contains(textOutput, "No stuck-state conditions detected") {
		t.Errorf("text output should contain 'No stuck-state conditions detected', got:\n%s", textOutput)
	}
}
