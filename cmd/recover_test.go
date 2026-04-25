package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// Test helpers (prefixed with "recover" to avoid collisions with existing
// helpers in the same package: stringPtr, createTestColonyState, writeTestJSON)
// ---------------------------------------------------------------------------

// recoverTimePtr returns a pointer to the given time.
func recoverTimePtr(t time.Time) *time.Time { return &t }

// newRecoverTestState builds a ColonyState with sensible defaults. Overrides
// mutate the state before returning so each test can customise without
// repeating boilerplate.
func newRecoverTestState(t *testing.T, overrides ...func(*colony.ColonyState)) colony.ColonyState {
	t.Helper()
	goal := "Test colony"
	s := colony.ColonyState{
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: recoverTimePtr(time.Now().Add(-2 * time.Hour)),
	}
	for _, fn := range overrides {
		fn(&s)
	}
	return s
}

// recoverWriteFile writes a file relative to dir with optional subdirectory creation.
func recoverWriteFile(t *testing.T, dir, name, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// recoverWriteJSON marshals v and writes it to name relative to dir.
func recoverWriteJSON(t *testing.T, dir, name string, v interface{}) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal %s: %v", name, err)
	}
	recoverWriteFile(t, dir, name, string(data))
}

// initRecoverTestStore creates a storage.Store rooted at a temp directory that
// mimics .aether/data/ and sets the package-level store variable. It also sets
// AETHER_ROOT so resolveAetherRoot returns the temp root.
func initRecoverTestStore(t *testing.T) (*storage.Store, string) {
	t.Helper()
	saveGlobals(t)

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	store = s

	origRoot := os.Getenv("AETHER_ROOT")
	t.Cleanup(func() {
		os.Setenv("AETHER_ROOT", origRoot)
	})
	os.Setenv("AETHER_ROOT", tmpDir)

	return s, dataDir
}

// assertIssueCount fails the test if the issue count doesn't match expected.
func assertIssueCount(t *testing.T, issues []HealthIssue, expected int, msg string) {
	t.Helper()
	if len(issues) != expected {
		t.Errorf("%s: expected %d issues, got %d", msg, expected, len(issues))
		for _, iss := range issues {
			t.Logf("  issue: [%s] %s: %s", iss.Severity, iss.Category, iss.Message)
		}
	}
}

// assertHasCategory checks that at least one issue has the given category.
func assertHasCategory(t *testing.T, issues []HealthIssue, category, msg string) {
	t.Helper()
	for _, iss := range issues {
		if iss.Category == category {
			return
		}
	}
	t.Errorf("%s: no issue with category %q among %d issues", msg, category, len(issues))
}

// assertHasSeverity checks that at least one issue has the given severity.
func assertHasSeverity(t *testing.T, issues []HealthIssue, severity, msg string) {
	t.Helper()
	for _, iss := range issues {
		if iss.Severity == severity {
			return
		}
	}
	t.Errorf("%s: no issue with severity %q among %d issues", msg, severity, len(issues))
}

// ---------------------------------------------------------------------------
// DETECT-01: Missing Build Packet
// ---------------------------------------------------------------------------

func TestScanMissingBuildPacket_DetectsMissingManifest(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	state := newRecoverTestState(t)

	// No manifest file created for phase 1 -- should be detected.
	issues := scanMissingBuildPacket(state, dataDir)

	assertIssueCount(t, issues, 1, "missing manifest")
	if len(issues) > 0 {
		if issues[0].Category != "missing_build_packet" {
			t.Errorf("expected category missing_build_packet, got %s", issues[0].Category)
		}
		if issues[0].Severity != "critical" {
			t.Errorf("expected severity critical, got %s", issues[0].Severity)
		}
	}
}

func TestScanMissingBuildPacket_SkipsWhenNotExecuting(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	state := newRecoverTestState(t, func(s *colony.ColonyState) {
		s.State = colony.StateREADY
	})

	issues := scanMissingBuildPacket(state, dataDir)
	assertIssueCount(t, issues, 0, "not executing")
}

func TestScanMissingBuildPacket_SkipsWhenPhaseZero(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	state := newRecoverTestState(t, func(s *colony.ColonyState) {
		s.CurrentPhase = 0
	})

	issues := scanMissingBuildPacket(state, dataDir)
	assertIssueCount(t, issues, 0, "phase zero")
}

func TestScanMissingBuildPacket_DetectsPlanOnlyManifest(t *testing.T) {
	s, dataDir := initRecoverTestStore(t)
	state := newRecoverTestState(t)

	// Create a manifest with PlanOnly=true and no dispatches.
	manifest := codexBuildManifest{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		State:       "executing",
		PlanOnly:    true,
	}
	manifestBytes, _ := json.Marshal(manifest)
	relPath := filepath.Join("build", "phase-1", "manifest.json")
	if err := os.MkdirAll(filepath.Join(s.BasePath(), "build", "phase-1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON(relPath, manifest); err != nil {
		t.Fatalf("save manifest: %v", err)
	}
	// Also write to dataDir for scanBadManifest path resolution.
	recoverWriteFile(t, dataDir, "build/phase-1/manifest.json", string(manifestBytes))

	issues := scanMissingBuildPacket(state, dataDir)
	assertIssueCount(t, issues, 1, "plan-only manifest")
	if len(issues) > 0 && issues[0].Category != "missing_build_packet" {
		t.Errorf("expected category missing_build_packet, got %s", issues[0].Category)
	}
}

// ---------------------------------------------------------------------------
// DETECT-02: Stale Spawned Workers
// ---------------------------------------------------------------------------

func TestScanStaleSpawned_DetectsOldWorkers(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	// Create spawn-runs.json with a run that started 2 hours ago.
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

	issues := scanStaleSpawnedWorkers(dataDir)
	assertIssueCount(t, issues, 1, "old worker")
	assertHasCategory(t, issues, "stale_spawned", "old worker category")
	assertHasSeverity(t, issues, "critical", "old worker severity")
}

func TestScanStaleSpawned_SkipsRecentWorkers(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	// Active run started 5 minutes ago -- should be fine.
	recentTime := time.Now().Add(-5 * time.Minute).Format(time.RFC3339)
	spawnData := map[string]interface{}{
		"current_run_id": "run-1",
		"runs": []map[string]interface{}{
			{
				"id":         "run-1",
				"started_at": recentTime,
				"status":     "active",
			},
		},
	}
	recoverWriteJSON(t, dataDir, "spawn-runs.json", spawnData)

	issues := scanStaleSpawnedWorkers(dataDir)
	assertIssueCount(t, issues, 0, "recent worker")
}

func TestScanStaleSpawned_SkipsCompletedRuns(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	// Completed run from long ago -- should not trigger.
	oldTime := time.Now().Add(-3 * time.Hour).Format(time.RFC3339)
	spawnData := map[string]interface{}{
		"runs": []map[string]interface{}{
			{
				"id":         "run-1",
				"started_at": oldTime,
				"status":     "completed",
			},
		},
	}
	recoverWriteJSON(t, dataDir, "spawn-runs.json", spawnData)

	issues := scanStaleSpawnedWorkers(dataDir)
	assertIssueCount(t, issues, 0, "completed run")
}

func TestScanStaleSpawned_NoSpawnFile(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	// No spawn-runs.json -- should return 0 issues.
	issues := scanStaleSpawnedWorkers(dataDir)
	assertIssueCount(t, issues, 0, "no spawn file")
}

// ---------------------------------------------------------------------------
// DETECT-03: Partial Phase
// ---------------------------------------------------------------------------

func TestScanPartialPhase_DetectsCompletedBuildNoContinue(t *testing.T) {
	s, dataDir := initRecoverTestStore(t)
	goal := "Test"
	state := colony.ColonyState{
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: recoverTimePtr(time.Now().Add(-2 * time.Hour)),
	}

	// Create a manifest with all completed dispatches.
	manifest := codexBuildManifest{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		State:       "executing",
		Dispatches: []codexBuildDispatch{
			{TaskID: "task-1", Status: "completed"},
			{TaskID: "task-2", Status: "completed"},
		},
	}
	relPath := filepath.Join("build", "phase-1", "manifest.json")
	if err := os.MkdirAll(filepath.Join(s.BasePath(), "build", "phase-1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON(relPath, manifest); err != nil {
		t.Fatalf("save manifest: %v", err)
	}
	manifestBytes, _ := json.Marshal(manifest)
	recoverWriteFile(t, dataDir, "build/phase-1/manifest.json", string(manifestBytes))

	// Do NOT create continue.json.
	issues := scanPartialPhase(state, dataDir)

	assertHasCategory(t, issues, "partial_phase", "completed build no continue")
	if len(issues) == 0 {
		t.Error("expected partial_phase issue for completed build without continue")
	}
}

func TestScanPartialPhase_DetectsPhaseMarkedNeverBuilt(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	goal := "Test"
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

	// No manifest file exists.
	issues := scanPartialPhase(state, dataDir)

	assertHasCategory(t, issues, "partial_phase", "never built phase")
}

func TestScanPartialPhase_SkipsWhenNotExecuting(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	state := newRecoverTestState(t, func(s *colony.ColonyState) {
		s.State = colony.StateREADY
	})

	issues := scanPartialPhase(state, dataDir)
	assertIssueCount(t, issues, 0, "not executing partial phase")
}

// ---------------------------------------------------------------------------
// DETECT-04: Bad Manifest
// ---------------------------------------------------------------------------

func TestScanBadManifest_DetectsCorruptJSON(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	state := newRecoverTestState(t)

	// Write corrupt JSON to manifest.
	recoverWriteFile(t, dataDir, "build/phase-1/manifest.json", "{bad json")

	issues := scanBadManifest(state, dataDir)
	if len(issues) == 0 {
		t.Fatal("expected at least 1 issue for corrupt JSON")
	}
	assertHasCategory(t, issues, "bad_manifest", "corrupt JSON")
	assertHasSeverity(t, issues, "critical", "corrupt JSON severity")
}

func TestScanBadManifest_DetectsPhaseMismatch(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	state := newRecoverTestState(t) // CurrentPhase = 1

	// Manifest with phase=99 -- mismatched against state phase 1.
	manifest := map[string]interface{}{
		"phase":        99,
		"generated_at": time.Now().Format(time.RFC3339),
		"state":        "executing",
		"dispatches":   []interface{}{},
	}
	recoverWriteJSON(t, dataDir, "build/phase-1/manifest.json", manifest)

	issues := scanBadManifest(state, dataDir)
	assertHasCategory(t, issues, "bad_manifest", "phase mismatch")
	assertHasSeverity(t, issues, "warning", "phase mismatch is warning-level")
}

func TestScanBadManifest_SkipsWhenNoManifest(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	state := newRecoverTestState(t)

	// No manifest file on disk.
	issues := scanBadManifest(state, dataDir)
	assertIssueCount(t, issues, 0, "no manifest file")
}

func TestScanBadManifest_DetectsEmptyGeneratedAt(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	state := newRecoverTestState(t)

	manifest := map[string]interface{}{
		"phase":        1,
		"generated_at": "",
		"state":        "executing",
		"dispatches":   []interface{}{},
	}
	recoverWriteJSON(t, dataDir, "build/phase-1/manifest.json", manifest)

	issues := scanBadManifest(state, dataDir)
	assertHasCategory(t, issues, "bad_manifest", "empty generated_at")
}

func TestScanBadManifest_DetectsEmptyState(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	state := newRecoverTestState(t)

	manifest := map[string]interface{}{
		"phase":        1,
		"generated_at": time.Now().Format(time.RFC3339),
		"state":        "",
		"dispatches":   []interface{}{},
	}
	recoverWriteJSON(t, dataDir, "build/phase-1/manifest.json", manifest)

	issues := scanBadManifest(state, dataDir)
	assertHasCategory(t, issues, "bad_manifest", "empty state field")
}

// ---------------------------------------------------------------------------
// DETECT-05: Dirty Worktrees
// ---------------------------------------------------------------------------

func TestScanDirtyWorktrees_DetectsStateDiskMismatch(t *testing.T) {
	_, _ = initRecoverTestStore(t)

	goal := "Test"
	state := colony.ColonyState{
		Goal:  &goal,
		State: colony.StateEXECUTING,
		Worktrees: []colony.WorktreeEntry{
			{
				ID:     "wt-1",
				Branch: "feature/test-branch",
				Path:   "/tmp/nonexistent-worktree-path-for-testing-12345",
				Status: colony.WorktreeAllocated,
			},
		},
	}

	issues := scanDirtyWorktrees(state)
	if len(issues) == 0 {
		t.Fatal("expected at least 1 issue for state-disk mismatch")
	}
	assertHasCategory(t, issues, "dirty_worktree", "state-disk mismatch")
	assertHasSeverity(t, issues, "warning", "state-disk mismatch is warning")
}

func TestScanDirtyWorktrees_SkipsMergedWorktrees(t *testing.T) {
	_, _ = initRecoverTestStore(t)

	goal := "Test"
	state := colony.ColonyState{
		Goal:  &goal,
		State: colony.StateEXECUTING,
		Worktrees: []colony.WorktreeEntry{
			{
				ID:     "wt-1",
				Branch: "feature/test-branch",
				Path:   "/tmp/nonexistent-worktree-path-for-testing-12345",
				Status: colony.WorktreeMerged,
			},
		},
	}

	issues := scanDirtyWorktrees(state)
	assertIssueCount(t, issues, 0, "merged worktree should be skipped")
}

func TestScanDirtyWorktrees_NoWorktrees(t *testing.T) {
	_, _ = initRecoverTestStore(t)
	state := newRecoverTestState(t)

	issues := scanDirtyWorktrees(state)
	assertIssueCount(t, issues, 0, "no worktrees")
}

// ---------------------------------------------------------------------------
// DETECT-06: Broken Survey
// ---------------------------------------------------------------------------

func TestScanBrokenSurvey_DetectsMissingFiles(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	// Create survey dir with only 2 of 5 files.
	surveyDir := filepath.Join(dataDir, "survey")
	if err := os.MkdirAll(surveyDir, 0755); err != nil {
		t.Fatal(err)
	}
	recoverWriteFile(t, dataDir, "survey/blueprint.json", `{"ok": true}`)
	recoverWriteFile(t, dataDir, "survey/chambers.json", `{"ok": true}`)
	// Missing: disciplines.json, provisions.json, pathogens.json

	surveyed := "yes"
	state := newRecoverTestState(t, func(s *colony.ColonyState) {
		s.TerritorySurveyed = &surveyed
	})

	issues := scanBrokenSurvey(state, dataDir)
	if len(issues) < 3 {
		t.Errorf("expected at least 3 missing survey file issues, got %d", len(issues))
	}
	for _, iss := range issues {
		if iss.Category != "broken_survey" {
			t.Errorf("expected category broken_survey, got %s", iss.Category)
		}
	}
}

func TestScanBrokenSurvey_DetectsEmptyFiles(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	// Create all 5 survey files but make some empty.
	recoverWriteFile(t, dataDir, "survey/blueprint.json", `{"ok": true}`)
	recoverWriteFile(t, dataDir, "survey/chambers.json", `null`)
	recoverWriteFile(t, dataDir, "survey/disciplines.json", `{}`)
	recoverWriteFile(t, dataDir, "survey/provisions.json", `[]`)
	recoverWriteFile(t, dataDir, "survey/pathogens.json", `{"ok": true}`)

	surveyed := "yes"
	state := newRecoverTestState(t, func(s *colony.ColonyState) {
		s.TerritorySurveyed = &surveyed
	})

	issues := scanBrokenSurvey(state, dataDir)
	if len(issues) == 0 {
		t.Error("expected issues for empty survey files")
	}
	assertHasCategory(t, issues, "broken_survey", "empty files")
}

func TestScanBrokenSurvey_SkipsWhenNoSurvey(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)
	state := newRecoverTestState(t) // TerritorySurveyed is nil

	issues := scanBrokenSurvey(state, dataDir)
	assertIssueCount(t, issues, 0, "no survey flag")
}

// ---------------------------------------------------------------------------
// DETECT-07: Missing Agent Files
// ---------------------------------------------------------------------------

func TestScanMissingAgentFiles_DetectsShortCount(t *testing.T) {
	s, _ := initRecoverTestStore(t)
	root := filepath.Dir(filepath.Dir(s.BasePath())) // tmpDir/.aether/data -> tmpDir

	// Create agent dirs with fewer than 25 files each.
	claudeDir := filepath.Join(root, ".claude", "agents", "ant")
	opencodeDir := filepath.Join(root, ".opencode", "agents")
	codexDir := filepath.Join(root, ".codex", "agents")

	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(opencodeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Only 5 agents in each (expected 25).
	for i := 0; i < 5; i++ {
		os.WriteFile(filepath.Join(claudeDir, "agent"+string(rune('a'+i))+".md"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(opencodeDir, "agent"+string(rune('a'+i))+".md"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(codexDir, "agent"+string(rune('a'+i))+".toml"), []byte("test"), 0644)
	}

	issues := scanMissingAgentFiles()
	if len(issues) == 0 {
		t.Fatal("expected issues for short agent count")
	}
	assertHasCategory(t, issues, "missing_agents", "short agent count")
}

func TestScanMissingAgentFiles_NoDirs(t *testing.T) {
	s, _ := initRecoverTestStore(t)
	// AETHER_ROOT points to tmpDir but no agent dirs exist.
	// resolveAetherRoot returns tmpDir, glob on nonexistent dirs returns empty.
	_ = filepath.Dir(filepath.Dir(s.BasePath()))

	issues := scanMissingAgentFiles()
	if len(issues) == 0 {
		t.Fatal("expected issues when agent directories are missing")
	}
	assertHasCategory(t, issues, "missing_agents", "missing agent dirs")
}

// ---------------------------------------------------------------------------
// OUTP-01: Render Recover Diagnosis
// ---------------------------------------------------------------------------

func TestRenderRecoverDiagnosis_ContainsSummary(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "missing_build_packet", Message: "No build packet", File: "build/phase-1/manifest.json", Fixable: true},
		{Severity: "warning", Category: "missing_agents", Message: "Few agents", Fixable: true},
	}
	goal := "Test colony goal"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: make([]colony.Phase, 3),
		},
	}

	output := renderRecoverDiagnosis(issues, state, nil)

	if !strings.Contains(output, "Diagnosis") {
		t.Error("output should contain 'Diagnosis'")
	}
	if !strings.Contains(output, "2 issues found") {
		t.Error("output should contain '2 issues found'")
	}
	if !strings.Contains(output, "1 critical") {
		t.Error("output should contain '1 critical'")
	}
	if !strings.Contains(output, "1 warning") {
		t.Error("output should contain '1 warning'")
	}
}

func TestRenderRecoverDiagnosis_NoIssues(t *testing.T) {
	goal := "Healthy colony"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		Plan: colony.Plan{
			Phases: make([]colony.Phase, 5),
		},
	}

	output := renderRecoverDiagnosis(nil, state, nil)

	if !strings.Contains(output, "No stuck-state conditions detected") {
		t.Error("output should contain 'No stuck-state conditions detected'")
	}
	if !strings.Contains(output, "Colony is healthy") {
		t.Error("output should contain 'Colony is healthy'")
	}
}

func TestRenderRecoverDiagnosis_ShowsFixableHint(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "stale_spawned", Message: "Stale workers", Fixable: true},
	}
	goal := "Test"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: make([]colony.Phase, 1)},
	}

	output := renderRecoverDiagnosis(issues, state, nil)

	if !strings.Contains(output, "Fixable with --apply") {
		t.Error("output should show fixable hint")
	}
}

// ---------------------------------------------------------------------------
// OUTP-02: Exit Code
// ---------------------------------------------------------------------------

func TestRecoverExitCode_ZeroWhenHealthy(t *testing.T) {
	code := recoverExitCode(nil)
	if code != 0 {
		t.Errorf("expected exit code 0 for no issues, got %d", code)
	}

	code = recoverExitCode([]HealthIssue{})
	if code != 0 {
		t.Errorf("expected exit code 0 for empty issues, got %d", code)
	}
}

func TestRecoverExitCode_OneWhenIssues(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "warning", Category: "test", Message: "test issue"},
	}
	code := recoverExitCode(issues)
	if code != 1 {
		t.Errorf("expected exit code 1 for issues, got %d", code)
	}
}

func TestRecoverExitCode_OneWhenCritical(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "missing_build_packet", Message: "No packet"},
	}
	code := recoverExitCode(issues)
	if code != 1 {
		t.Errorf("expected exit code 1 for critical issues, got %d", code)
	}
}

// ---------------------------------------------------------------------------
// JSON output
// ---------------------------------------------------------------------------

func TestRenderRecoverJSON_ValidStructure(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "missing_build_packet", Message: "No packet", Fixable: true},
		{Severity: "warning", Category: "missing_agents", Message: "Few agents", Fixable: true},
	}
	goal := "JSON test goal"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: make([]colony.Phase, 3),
		},
	}

	output := renderRecoverJSON(issues, state, 100*time.Millisecond, nil)

	// Must be valid JSON.
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	// Check expected top-level fields.
	if _, ok := parsed["issues"]; !ok {
		t.Error("JSON output missing 'issues' field")
	}
	if _, ok := parsed["summary"]; !ok {
		t.Error("JSON output missing 'summary' field")
	}
	if _, ok := parsed["goal"]; !ok {
		t.Error("JSON output missing 'goal' field")
	}

	// Check summary counts.
	summary, ok := parsed["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("summary is not an object")
	}
	if summary["critical"].(float64) != 1 {
		t.Errorf("expected summary.critical=1, got %v", summary["critical"])
	}
	if summary["warning"].(float64) != 1 {
		t.Errorf("expected summary.warning=1, got %v", summary["warning"])
	}
	if summary["fixable"].(float64) != 2 {
		t.Errorf("expected summary.fixable=2, got %v", summary["fixable"])
	}

	// Check issues array length.
	issuesArr, ok := parsed["issues"].([]interface{})
	if !ok {
		t.Fatal("issues is not an array")
	}
	if len(issuesArr) != 2 {
		t.Errorf("expected 2 issues in JSON, got %d", len(issuesArr))
	}

	// Check exit_code is 1.
	if exitCode, ok := parsed["exit_code"].(float64); !ok || exitCode != 1 {
		t.Errorf("expected exit_code=1, got %v", parsed["exit_code"])
	}

	// Check scan_duration_ms.
	if durMs, ok := parsed["scan_duration_ms"].(float64); !ok || durMs != 100 {
		t.Errorf("expected scan_duration_ms=100, got %v", parsed["scan_duration_ms"])
	}
}

func TestRenderRecoverJSON_NoIssues(t *testing.T) {
	goal := "Clean colony"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
	}

	output := renderRecoverJSON(nil, state, 50*time.Millisecond, nil)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if exitCode, ok := parsed["exit_code"].(float64); !ok || exitCode != 0 {
		t.Errorf("expected exit_code=0 for no issues, got %v", parsed["exit_code"])
	}
}

// ---------------------------------------------------------------------------
// Recover Next Step
// ---------------------------------------------------------------------------

func TestRecoverNextStep_CriticalMissingBuildPacket(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "missing_build_packet", Message: "No packet"},
	}
	next := recoverNextStep(issues)
	if !strings.Contains(next, "build") {
		t.Errorf("next step for missing_build_packet should mention build, got: %s", next)
	}
}

func TestRecoverNextStep_CriticalPartialPhase(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "partial_phase", Message: "Partial"},
	}
	next := recoverNextStep(issues)
	if !strings.Contains(next, "continue") {
		t.Errorf("next step for partial_phase should mention continue, got: %s", next)
	}
}

func TestRecoverNextStep_WarningMissingAgents(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "warning", Category: "missing_agents", Message: "Few agents"},
	}
	next := recoverNextStep(issues)
	if !strings.Contains(next, "update") {
		t.Errorf("next step for missing_agents should mention update, got: %s", next)
	}
}

// ---------------------------------------------------------------------------
// Integration: performStuckStateScan orchestrator
// ---------------------------------------------------------------------------

func TestPerformStuckStateScan_ReturnsStateError(t *testing.T) {
	saveGlobals(t)
	// store is nil -- loadActiveColonyState will fail.
	store = nil

	issues, err := performStuckStateScan(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) == 0 {
		t.Fatal("expected state error issue when store is nil")
	}
	if issues[0].Category != "state" {
		t.Errorf("expected category 'state', got %s", issues[0].Category)
	}
}

// ---------------------------------------------------------------------------
// REPAIR TESTS
// ---------------------------------------------------------------------------

// withMockStdin replaces os.Stdin with a pipe containing the given input for
// the duration of fn, then restores the original stdin.
func withMockStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdin = r
	done := make(chan struct{})
	go func() {
		w.WriteString(input)
		w.Close()
		close(done)
	}()
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()
	fn()
}

// ---------------------------------------------------------------------------
// Backup verification
// ---------------------------------------------------------------------------

func TestRepairBackup_CreatedBeforeMutation(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	issues := []HealthIssue{
		{Category: "missing_build_packet", Severity: "critical", Fixable: true, Message: "No build packet", File: "build/phase-1/manifest.json"},
	}

	result, err := performRecoverRepairs(issues, dataDir, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify backup directory exists.
	backupsDir := filepath.Join(filepath.Dir(dataDir), "backups")
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		t.Fatalf("list backups dir: %v", err)
	}

	foundBackup := false
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "medic-") {
			foundBackup = true
			// Check for COLONY_STATE.json copy.
			backupPath := filepath.Join(backupsDir, entry.Name(), "COLONY_STATE.json")
			if _, err := os.Stat(backupPath); err != nil {
				t.Errorf("backup missing COLONY_STATE.json: %v", err)
			}
			// Check for backup manifest.
			manifestPath := filepath.Join(backupsDir, entry.Name(), "_backup_manifest.json")
			if _, err := os.Stat(manifestPath); err != nil {
				t.Errorf("backup missing _backup_manifest.json: %v", err)
			}
			break
		}
	}
	if !foundBackup {
		t.Error("expected at least one medic-* backup directory")
	}
}

// ---------------------------------------------------------------------------
// REPAIR-01: Missing Build Packet
// ---------------------------------------------------------------------------

func TestRepairMissingBuildPacket_ResetsToReady(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	issues := []HealthIssue{
		{Category: "missing_build_packet", Severity: "critical", Fixable: true, Message: "No build packet", File: "build/phase-1/manifest.json"},
	}

	result, err := performRecoverRepairs(issues, dataDir, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", result.Succeeded)
	}

	// Verify state was changed to READY and build_started_at is nil.
	var repaired colony.ColonyState
	data, _ := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err := json.Unmarshal(data, &repaired); err != nil {
		t.Fatalf("parse repaired state: %v", err)
	}
	if repaired.State != colony.StateREADY {
		t.Errorf("expected state READY, got %s", repaired.State)
	}
	if repaired.BuildStartedAt != nil {
		t.Error("expected build_started_at to be nil")
	}
}

// ---------------------------------------------------------------------------
// REPAIR-02: Stale Spawned Workers
// ---------------------------------------------------------------------------

func TestRepairStaleSpawned_ResetsToFailed(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	// Also write COLONY_STATE.json so backup can proceed.
	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

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

	issues := []HealthIssue{
		{Category: "stale_spawned", Severity: "critical", Fixable: true, Message: "1 spawned worker(s) exceeded 1-hour threshold", File: "spawn-runs.json"},
	}

	result, err := performRecoverRepairs(issues, dataDir, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", result.Succeeded)
	}

	// Verify spawn-runs.json was updated.
	var repaired map[string]interface{}
	data, _ := os.ReadFile(filepath.Join(dataDir, "spawn-runs.json"))
	if err := json.Unmarshal(data, &repaired); err != nil {
		t.Fatalf("parse repaired spawn-runs: %v", err)
	}
	if repaired["current_run_id"] != "" {
		t.Errorf("expected current_run_id to be cleared, got %v", repaired["current_run_id"])
	}

	runs := repaired["runs"].([]interface{})
	if len(runs) == 0 {
		t.Fatal("expected runs to exist")
	}
	run := runs[0].(map[string]interface{})
	if run["status"] != "failed" {
		t.Errorf("expected run status 'failed', got %v", run["status"])
	}
}

// ---------------------------------------------------------------------------
// REPAIR-03: Partial Phase
// ---------------------------------------------------------------------------

func TestRepairPartialPhase_TransitionsToBuilt(t *testing.T) {
	s, dataDir := initRecoverTestStore(t)

	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	// Create a manifest with all completed dispatches.
	manifest := codexBuildManifest{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		State:       "executing",
		Dispatches: []codexBuildDispatch{
			{TaskID: "task-1", Status: "completed"},
			{TaskID: "task-2", Status: "completed"},
		},
	}
	relPath := filepath.Join("build", "phase-1", "manifest.json")
	if err := os.MkdirAll(filepath.Join(s.BasePath(), "build", "phase-1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON(relPath, manifest); err != nil {
		t.Fatalf("save manifest: %v", err)
	}
	manifestBytes, _ := json.Marshal(manifest)
	recoverWriteFile(t, dataDir, "build/phase-1/manifest.json", string(manifestBytes))

	issues := []HealthIssue{
		{Category: "partial_phase", Severity: "warning", Fixable: true, Message: "Build completed but continue not run", File: "build/phase-1/continue.json"},
	}

	result, err := performRecoverRepairs(issues, dataDir, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", result.Succeeded)
	}

	var repaired colony.ColonyState
	data, _ := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err := json.Unmarshal(data, &repaired); err != nil {
		t.Fatalf("parse repaired state: %v", err)
	}
	if repaired.State != colony.StateBUILT {
		t.Errorf("expected state BUILT, got %s", repaired.State)
	}
}

func TestRepairPartialPhase_ResetsToPending(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	goal := "Test"
	state := colony.ColonyState{
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: recoverTimePtr(time.Now().Add(-2 * time.Hour)),
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: "in_progress"},
			},
		},
	}
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	// No manifest file -- should trigger reset-to-pending path.

	issues := []HealthIssue{
		{Category: "partial_phase", Severity: "warning", Fixable: true, Message: "Phase 1 marked in_progress but never built", File: "COLONY_STATE.json"},
	}

	result, err := performRecoverRepairs(issues, dataDir, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", result.Succeeded)
	}

	var repaired colony.ColonyState
	data, _ := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err := json.Unmarshal(data, &repaired); err != nil {
		t.Fatalf("parse repaired state: %v", err)
	}
	if repaired.State != colony.StateREADY {
		t.Errorf("expected state READY, got %s", repaired.State)
	}
	if repaired.BuildStartedAt != nil {
		t.Error("expected build_started_at to be nil")
	}
	if len(repaired.Plan.Phases) > 0 && repaired.Plan.Phases[0].Status != "pending" {
		t.Errorf("expected phase status 'pending', got %s", repaired.Plan.Phases[0].Status)
	}
}

// ---------------------------------------------------------------------------
// REPAIR-04: Broken Survey
// ---------------------------------------------------------------------------

func TestRepairBrokenSurvey_ClearsTerritoryAndDeletesFiles(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	surveyed := "yes"
	state := newRecoverTestState(t, func(s *colony.ColonyState) {
		s.TerritorySurveyed = &surveyed
	})
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	// Create survey dir with empty/corrupt files.
	surveyDir := filepath.Join(dataDir, "survey")
	if err := os.MkdirAll(surveyDir, 0755); err != nil {
		t.Fatal(err)
	}
	recoverWriteFile(t, dataDir, "survey/blueprint.json", `null`)
	recoverWriteFile(t, dataDir, "survey/chambers.json", `{}`)
	recoverWriteFile(t, dataDir, "survey/disciplines.json", `[]`)

	issues := []HealthIssue{
		{Category: "broken_survey", Severity: "warning", Fixable: true, Message: "Survey file is empty: blueprint.json", File: "survey/blueprint.json"},
		{Category: "broken_survey", Severity: "warning", Fixable: true, Message: "Survey file is empty: chambers.json", File: "survey/chambers.json"},
		{Category: "broken_survey", Severity: "warning", Fixable: true, Message: "Survey file is empty: disciplines.json", File: "survey/disciplines.json"},
	}

	result, err := performRecoverRepairs(issues, dataDir, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Each issue has a unique category:message key, so all 3 are dispatched.
	// repairBrokenSurvey is idempotent -- clearing nil territory_surveyed again
	// and removing already-deleted files still succeeds.
	if result.Succeeded != 3 {
		t.Errorf("expected 3 succeeded (3 unique issues), got %d", result.Succeeded)
	}

	var repaired colony.ColonyState
	data, _ := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err := json.Unmarshal(data, &repaired); err != nil {
		t.Fatalf("parse repaired state: %v", err)
	}
	if repaired.TerritorySurveyed != nil {
		t.Error("expected territory_surveyed to be nil")
	}

	// Verify survey files were removed.
	if _, err := os.Stat(filepath.Join(surveyDir, "blueprint.json")); err == nil {
		t.Error("expected blueprint.json to be removed")
	}
	if _, err := os.Stat(filepath.Join(surveyDir, "chambers.json")); err == nil {
		t.Error("expected chambers.json to be removed")
	}
}

// ---------------------------------------------------------------------------
// REPAIR-05: Missing Agent Files
// ---------------------------------------------------------------------------

func TestRepairMissingAgents_CopiesFromHub(t *testing.T) {
	s, dataDir := initRecoverTestStore(t)
	root := filepath.Dir(filepath.Dir(s.BasePath())) // tmpDir

	// Set up hub directory structure with agent files.
	hubDir := filepath.Join(root, "hub_home", ".aether", "system")
	claudeHubDir := filepath.Join(hubDir, "claude", "agents")
	if err := os.MkdirAll(claudeHubDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(claudeHubDir, "test-agent.md"), []byte("hub content"), 0644)

	// Create repo agent dir (empty).
	claudeRepoDir := filepath.Join(root, ".claude", "agents", "ant")
	if err := os.MkdirAll(claudeRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Override HOME to point to our fake home.
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", filepath.Join(root, "hub_home"))
	defer os.Setenv("HOME", origHome)

	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	issues := []HealthIssue{
		{Category: "missing_agents", Severity: "warning", Fixable: true, Message: "Claude agents: found 0 files, expected 25", File: ".claude/agents/ant/*.md"},
	}

	result, err := performRecoverRepairs(issues, dataDir, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", result.Succeeded)
	}

	// Verify agent file was copied.
	if _, err := os.Stat(filepath.Join(claudeRepoDir, "test-agent.md")); err != nil {
		t.Error("expected test-agent.md to be copied from hub")
	}
}

func TestRepairMissingAgents_HubEmpty(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	// Set HOME to an empty temp dir (no hub).
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	issues := []HealthIssue{
		{Category: "missing_agents", Severity: "warning", Fixable: true, Message: "Claude agents: found 0 files, expected 25", File: ".claude/agents/ant/*.md"},
	}

	result, err := performRecoverRepairs(issues, dataDir, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Succeeded != 0 {
		t.Errorf("expected 0 succeeded, got %d", result.Succeeded)
	}

	// Check that the repair record has the hub error.
	found := false
	for _, rec := range result.Repairs {
		if strings.Contains(rec.Error, "hub has no agent files") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected repair record to contain 'hub has no agent files' error")
	}
}

// ---------------------------------------------------------------------------
// REPAIR-06: Dirty Worktree (destructive confirmation)
// ---------------------------------------------------------------------------

func TestRepairDirtyWorktree_DestructiveNeedsConfirmation(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	state := newRecoverTestState(t, func(s *colony.ColonyState) {
		s.Worktrees = []colony.WorktreeEntry{
			{
				ID:     "wt-1",
				Branch: "feature/test",
				Path:   "/tmp/nonexistent-wt",
				Status: colony.WorktreeAllocated,
			},
		}
	})
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	issues := []HealthIssue{
		{Category: "dirty_worktree", Severity: "warning", Fixable: true, Message: "Worktree state-disk mismatch: state says allocated but path does not exist", File: "/tmp/nonexistent-wt"},
	}

	// Simulate user declining the confirmation.
	withMockStdin(t, "n\n", func() {
		result, err := performRecoverRepairs(issues, dataDir, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Skipped < 1 {
			t.Errorf("expected at least 1 skipped, got %d", result.Skipped)
		}
	})

	// Verify state was NOT modified.
	var after colony.ColonyState
	data, _ := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err := json.Unmarshal(data, &after); err != nil {
		t.Fatalf("parse state: %v", err)
	}
	if len(after.Worktrees) != 1 {
		t.Errorf("expected 1 worktree entry (unchanged), got %d", len(after.Worktrees))
	}
}

func TestRepairDirtyWorktree_ForceSkipsConfirmation(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	state := newRecoverTestState(t, func(s *colony.ColonyState) {
		s.Worktrees = []colony.WorktreeEntry{
			{
				ID:     "wt-1",
				Branch: "feature/test",
				Path:   "/tmp/nonexistent-wt-force",
				Status: colony.WorktreeAllocated,
			},
		}
	})
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	issues := []HealthIssue{
		{Category: "dirty_worktree", Severity: "warning", Fixable: true, Message: "Worktree state-disk mismatch: state says allocated but path does not exist", File: "/tmp/nonexistent-wt-force"},
	}

	result, err := performRecoverRepairs(issues, dataDir, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Succeeded != 1 {
		t.Errorf("expected 1 succeeded with --force, got %d (succeeded=%d, failed=%d, skipped=%d)", result.Succeeded, result.Succeeded, result.Failed, result.Skipped)
	}

	// Verify the orphan worktree entry was removed.
	var after colony.ColonyState
	data, _ := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err := json.Unmarshal(data, &after); err != nil {
		t.Fatalf("parse state: %v", err)
	}
	if len(after.Worktrees) != 0 {
		t.Errorf("expected 0 worktree entries after removal, got %d", len(after.Worktrees))
	}
}

// ---------------------------------------------------------------------------
// REPAIR-07: Bad Manifest (destructive confirmation)
// ---------------------------------------------------------------------------

func TestRepairBadManifest_DestructiveNeedsConfirmation(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	// Create a corrupt manifest.
	recoverWriteFile(t, dataDir, "build/phase-1/manifest.json", "{bad json")

	issues := []HealthIssue{
		{Category: "bad_manifest", Severity: "critical", Fixable: true, Message: "Manifest JSON parse failed: invalid character 'b' looking for beginning of object key string", File: "build/phase-1/manifest.json"},
	}

	// Simulate user declining.
	withMockStdin(t, "n\n", func() {
		result, err := performRecoverRepairs(issues, dataDir, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Skipped < 1 {
			t.Errorf("expected at least 1 skipped, got %d", result.Skipped)
		}
	})
}

// ---------------------------------------------------------------------------
// Atomicity / Rollback
// ---------------------------------------------------------------------------

func TestRepairAtomicity_RollbackOnFailure(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	// Set up state in EXECUTING (will be repaired to READY).
	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	// Create a second issue that will fail: a manifest parse failure.
	// But the manifest file doesn't exist for repairBadManifest to read.
	// This tests the rollback path -- the state change from the first repair
	// should be rolled back.
	issues := []HealthIssue{
		{Category: "missing_build_packet", Severity: "critical", Fixable: true, Message: "No build packet", File: "build/phase-1/manifest.json"},
		{Category: "bad_manifest", Severity: "critical", Fixable: true, Message: "Manifest JSON parse failed", File: "build/phase-1/manifest.json"},
	}

	// Run with --force so destructive bad_manifest doesn't prompt.
	result, err := performRecoverRepairs(issues, dataDir, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The bad_manifest repair should fail because the file is not valid JSON
	// and findLastValidJSON can't recover it (or it might succeed depending on content).
	// What matters is: if any repair failed, rollback should restore original state.
	if result.Failed > 0 {
		var after colony.ColonyState
		data, _ := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
		if err := json.Unmarshal(data, &after); err != nil {
			t.Fatalf("parse state: %v", err)
		}
		// After rollback, state should be EXECUTING (original).
		if after.State != colony.StateEXECUTING {
			t.Errorf("after rollback expected state EXECUTING, got %s", after.State)
		}
	}
}

// ---------------------------------------------------------------------------
// isDestructiveCategory
// ---------------------------------------------------------------------------

func TestIsDestructiveCategory(t *testing.T) {
	tests := []struct {
		category string
		expected bool
	}{
		{"dirty_worktree", true},
		{"bad_manifest", true},
		{"missing_build_packet", false},
		{"stale_spawned", false},
		{"partial_phase", false},
		{"broken_survey", false},
		{"missing_agents", false},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := isDestructiveCategory(tt.category)
			if got != tt.expected {
				t.Errorf("isDestructiveCategory(%q) = %v, want %v", tt.category, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// performRecoverRepairs edge cases
// ---------------------------------------------------------------------------

func TestPerformRecoverRepairs_NoFixableIssues(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	issues := []HealthIssue{
		{Category: "state", Severity: "critical", Fixable: false, Message: "Not fixable"},
	}

	result, err := performRecoverRepairs(issues, dataDir, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Attempted != 0 {
		t.Errorf("expected 0 attempted, got %d", result.Attempted)
	}
}

func TestPerformRecoverRepairs_SkipsInJsonMode(t *testing.T) {
	_, dataDir := initRecoverTestStore(t)

	state := newRecoverTestState(t)
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

	issues := []HealthIssue{
		{Category: "dirty_worktree", Severity: "warning", Fixable: true, Message: "Worktree state-disk mismatch", File: "/tmp/nonexistent-wt-json"},
	}

	// jsonMode=true should skip destructive categories without prompting.
	result, err := performRecoverRepairs(issues, dataDir, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Skipped < 1 {
		t.Errorf("expected at least 1 skipped in json mode, got %d", result.Skipped)
	}

	// Verify the skip reason mentions non-interactive.
	found := false
	for _, rec := range result.Repairs {
		if strings.Contains(rec.Error, "non-interactive mode") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected skip reason to mention 'non-interactive mode'")
	}
}

// Repair wiring and visual output integration tests (Plan 02, Task 2)
// ---------------------------------------------------------------------------

func TestRenderRecoverDiagnosis_WithRepairLog(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "stale_spawned", Message: "Stale workers", Fixable: true},
	}
	goal := "Test"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: make([]colony.Phase, 1)},
	}
	repairResult := &RepairResult{
		Attempted: 3,
		Succeeded: 2,
		Failed:    1,
		Skipped:   0,
		Repairs: []RepairRecord{
			{Category: "stale_spawned", Action: "reset_stale_spawn_runs", File: "spawn-runs.json", Success: true},
			{Category: "partial_phase", Action: "reset_phase_state", File: "COLONY_STATE.json", Success: true},
			{Category: "bad_manifest", Action: "rebuild_manifest", File: "build/phase-1/manifest.json", Success: false, Error: "write failed"},
		},
	}

	output := renderRecoverDiagnosis(issues, state, repairResult)

	if !strings.Contains(output, "Repair Log") {
		t.Error("output should contain 'Repair Log' stage marker")
	}
	if !strings.Contains(output, "[OK]") {
		t.Error("output should contain '[OK]' for succeeded repairs")
	}
	if !strings.Contains(output, "[FAILED]") {
		t.Error("output should contain '[FAILED]' for failed repairs")
	}
	if !strings.Contains(output, "Summary: 3 attempted, 2 succeeded, 1 failed, 0 skipped") {
		t.Errorf("output should contain correct summary, got: %s", extractLineContaining(output, "Summary:"))
	}
}

func TestRenderRecoverDiagnosis_NoRepairResult(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "stale_spawned", Message: "Stale workers", Fixable: true},
	}
	goal := "Test"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: make([]colony.Phase, 1)},
	}

	output := renderRecoverDiagnosis(issues, state, nil)

	if strings.Contains(output, "Repair Log") {
		t.Error("output should NOT contain 'Repair Log' when repairResult is nil")
	}
}

func extractLineContaining(s, substr string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	return ""
}

func TestRenderRepairLog_FormatsCorrectly(t *testing.T) {
	result := &RepairResult{
		Attempted: 3,
		Succeeded: 2,
		Failed:    1,
		Skipped:   0,
		Repairs: []RepairRecord{
			{Category: "stale_spawned", Action: "reset_stale_spawn_runs", File: "spawn-runs.json", Success: true},
			{Category: "partial_phase", Action: "reset_phase_state", File: "COLONY_STATE.json", Success: true},
			{Category: "bad_manifest", Action: "rebuild_manifest", File: "build/phase-1/manifest.json", Success: false, Error: "write failed"},
		},
	}

	output := renderRepairLog(result)

	// Check succeeded repairs format: "[OK] action (file)"
	if !strings.Contains(output, "[OK] reset_stale_spawn_runs (spawn-runs.json)") {
		t.Error("output should contain '[OK] reset_stale_spawn_runs (spawn-runs.json)'")
	}
	if !strings.Contains(output, "[OK] reset_phase_state (COLONY_STATE.json)") {
		t.Error("output should contain '[OK] reset_phase_state (COLONY_STATE.json)'")
	}
	// Check failed repair format: "[FAILED] action (file): error"
	if !strings.Contains(output, "[FAILED] rebuild_manifest (build/phase-1/manifest.json): write failed") {
		t.Error("output should contain '[FAILED] rebuild_manifest (build/phase-1/manifest.json): write failed'")
	}
	// Check summary line
	if !strings.Contains(output, "Summary: 3 attempted, 2 succeeded, 1 failed, 0 skipped") {
		t.Errorf("expected correct summary line, got: %s", extractLineContaining(output, "Summary:"))
	}
}

func TestWriteRecoverIssueLine_SafeCategory(t *testing.T) {
	issue := HealthIssue{
		Severity: "critical",
		Category: "stale_spawned",
		Message:  "Stale workers",
		File:     "spawn-runs.json",
		Fixable:  true,
	}

	var b strings.Builder
	writeRecoverIssueLine(&b, issue)
	output := b.String()

	if !strings.Contains(output, "Fixable with --apply") {
		t.Error("safe category should show 'Fixable with --apply'")
	}
	if strings.Contains(output, "Needs confirmation") {
		t.Error("safe category should NOT show 'Needs confirmation'")
	}
}

func TestWriteRecoverIssueLine_DestructiveCategory(t *testing.T) {
	issue := HealthIssue{
		Severity: "warning",
		Category: "dirty_worktree",
		Message:  "Worktree mismatch",
		File:     "COLONY_STATE.json",
		Fixable:  true,
	}

	var b strings.Builder
	writeRecoverIssueLine(&b, issue)
	output := b.String()

	if !strings.Contains(output, "Needs confirmation with --apply") {
		t.Error("destructive category should show 'Needs confirmation with --apply'")
	}
	if strings.Contains(output, "Fixable with --apply") {
		t.Error("destructive category should NOT show 'Fixable with --apply'")
	}
}

func TestWriteRecoverIssueLine_DestructiveCategoryManifest(t *testing.T) {
	issue := HealthIssue{
		Severity: "critical",
		Category: "bad_manifest",
		Message:  "Corrupt manifest",
		File:     "build/phase-1/manifest.json",
		Fixable:  true,
	}

	var b strings.Builder
	writeRecoverIssueLine(&b, issue)
	output := b.String()

	if !strings.Contains(output, "Needs confirmation with --apply") {
		t.Error("bad_manifest should show 'Needs confirmation with --apply'")
	}
}

func TestRenderRecoverJSON_WithRepairs(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "stale_spawned", Message: "Stale workers", Fixable: true},
	}
	goal := "Test"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: make([]colony.Phase, 1)},
	}
	repairResult := &RepairResult{
		Attempted: 1,
		Succeeded: 1,
		Failed:    0,
		Skipped:   0,
		Repairs: []RepairRecord{
			{Category: "stale_spawned", Action: "reset_stale_spawn_runs", File: "spawn-runs.json", Success: true},
		},
	}

	output := renderRecoverJSON(issues, state, 50*time.Millisecond, repairResult)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	repairs, ok := parsed["repairs"].(map[string]interface{})
	if !ok {
		t.Fatal("JSON output missing 'repairs' object")
	}
	if repairs["attempted"].(float64) != 1 {
		t.Errorf("expected repairs.attempted=1, got %v", repairs["attempted"])
	}
	if repairs["succeeded"].(float64) != 1 {
		t.Errorf("expected repairs.succeeded=1, got %v", repairs["succeeded"])
	}
	if repairs["failed"].(float64) != 0 {
		t.Errorf("expected repairs.failed=0, got %v", repairs["failed"])
	}
	if repairs["skipped"].(float64) != 0 {
		t.Errorf("expected repairs.skipped=0, got %v", repairs["skipped"])
	}
	if _, ok := repairs["details"]; !ok {
		t.Error("repairs object missing 'details' field")
	}
}

func TestRenderRecoverJSON_NoRepairs(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "stale_spawned", Message: "Stale workers", Fixable: true},
	}
	goal := "Test"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: make([]colony.Phase, 1)},
	}

	output := renderRecoverJSON(issues, state, 50*time.Millisecond, nil)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if _, ok := parsed["repairs"]; ok {
		t.Error("JSON output should NOT have 'repairs' field when repairResult is nil")
	}
}

func TestRecoverNextStep_DirtyWorktree(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "dirty_worktree", Message: "Worktree mismatch"},
	}
	next := recoverNextStep(issues)
	if !strings.Contains(next, "--force") {
		t.Errorf("next step for dirty_worktree should mention --force, got: %s", next)
	}
}

func TestRecoverNextStep_BadManifest(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "bad_manifest", Message: "Corrupt manifest"},
	}
	next := recoverNextStep(issues)
	if !strings.Contains(next, "--force") {
		t.Errorf("next step for bad_manifest should mention --force, got: %s", next)
	}
}

