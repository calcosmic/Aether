package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// helper to create a JSON file in the data directory
func writeJSONFile(t *testing.T, dir, filename string, data interface{}) {
	t.Helper()
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", filename, err)
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, filename)), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filename, err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), b, 0644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}

// helper to write raw bytes to a file
func writeFile(t *testing.T, dir, filename string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, filename)), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filename, err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), content, 0644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}

// ---------------------------------------------------------------------------
// TestScanColonyStateHealthy
// ---------------------------------------------------------------------------

func TestScanColonyStateHealthy(t *testing.T) {
	dir := t.TempDir()
	goal := "Build something great"
	writeJSONFile(t, dir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Phase 1",
					Status: colony.PhasePending,
					Tasks: []colony.Task{
						{Goal: "Task 1", Status: colony.TaskPending},
					},
				},
			},
		},
		Events: []string{"2026-04-21T10:00:00Z|info|test|hello|world"},
	})

	fc := newFileChecker(dir)
	issues := scanColonyState(fc)

	critical := 0
	for _, issue := range issues {
		if issue.Severity == "critical" {
			critical++
			t.Errorf("unexpected critical issue: %s", issue.Message)
		}
	}
	if critical > 0 {
		t.Fatalf("healthy state produced %d critical issues", critical)
	}
}

// ---------------------------------------------------------------------------
// TestScanColonyStateCorrupted
// ---------------------------------------------------------------------------

func TestScanColonyStateCorrupted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "COLONY_STATE.json", []byte(`{not valid json`))

	fc := newFileChecker(dir)
	issues := scanColonyState(fc)
	// The corrupted JSON is caught by checkJSONFile (in fileChecker issues)
	// not by scanColonyState itself, since checkJSONFile returns ok=false
	allIssues := append(issues, fc.allIssues()...)

	found := false
	for _, issue := range allIssues {
		if issue.Severity == "critical" {
			found = true
		}
	}
	if !found {
		t.Error("corrupted JSON should produce critical issue")
	}
}

// ---------------------------------------------------------------------------
// TestScanColonyStateMissingGoal
// ---------------------------------------------------------------------------

func TestScanColonyStateMissingGoal(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, dir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		State:   colony.StateREADY,
	})

	fc := newFileChecker(dir)
	issues := scanColonyState(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "critical" && issue.Message == "Colony goal is missing" {
			found = true
		}
	}
	if !found {
		t.Error("missing goal should produce critical issue")
	}
}

// ---------------------------------------------------------------------------
// TestScanColonyStateInvalidState
// ---------------------------------------------------------------------------

func TestScanColonyStateInvalidState(t *testing.T) {
	dir := t.TempDir()
	goal := "test"
	writeJSONFile(t, dir, "COLONY_STATE.json", map[string]interface{}{
		"version": "3.0",
		"goal":    goal,
		"state":   "INVALID_STATE",
	})

	fc := newFileChecker(dir)
	issues := scanColonyState(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "critical" && issue.Category == "state" {
			found = true
			if !issue.Fixable {
				t.Error("invalid state issue should be fixable")
			}
		}
	}
	if !found {
		t.Error("invalid state should produce critical issue")
	}
}

// ---------------------------------------------------------------------------
// TestScanColonyStateDeprecatedSignals
// ---------------------------------------------------------------------------

func TestScanColonyStateDeprecatedSignals(t *testing.T) {
	dir := t.TempDir()
	goal := "test"
	writeJSONFile(t, dir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Signals: []colony.Signal{
			{ID: "s1", Type: "FOCUS", Content: "focus here", Active: true},
			{ID: "s2", Type: "REDIRECT", Content: "avoid that", Active: false},
		},
	})

	fc := newFileChecker(dir)
	issues := scanColonyState(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && issue.Category == "state" &&
			contains(issue.Message, "Deprecated") && contains(issue.Message, "signals") {
			found = true
		}
	}
	if !found {
		t.Error("deprecated signals should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanColonyStateOrphanedWorktrees
// ---------------------------------------------------------------------------

func TestScanColonyStateOrphanedWorktrees(t *testing.T) {
	dir := t.TempDir()
	goal := "test"
	writeJSONFile(t, dir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Worktrees: []colony.WorktreeEntry{
			{ID: "wt-1", Status: colony.WorktreeOrphaned},
			{ID: "wt-2", Status: colony.WorktreeMerged},
		},
	})

	fc := newFileChecker(dir)
	issues := scanColonyState(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "orphaned") {
			found = true
		}
	}
	if !found {
		t.Error("orphaned worktrees should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanSessionStale
// ---------------------------------------------------------------------------

func TestScanSessionStale(t *testing.T) {
	dir := t.TempDir()
	staleTime := time.Now().AddDate(0, 0, -10).UTC().Format(time.RFC3339)
	writeJSONFile(t, dir, "session.json", colony.SessionFile{
		SessionID:     "sess-1",
		StartedAt:     time.Now().AddDate(0, 0, -15).UTC().Format(time.RFC3339),
		LastCommandAt: staleTime,
		ColonyGoal:    "test goal",
	})

	fc := newFileChecker(dir)
	issues := scanSession(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "stale") {
			found = true
		}
	}
	if !found {
		t.Error("stale session should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanSessionCriticalStale
// ---------------------------------------------------------------------------

func TestScanSessionCriticalStale(t *testing.T) {
	dir := t.TempDir()
	staleTime := time.Now().AddDate(0, 0, -35).UTC().Format(time.RFC3339)
	writeJSONFile(t, dir, "session.json", colony.SessionFile{
		SessionID:     "sess-2",
		LastCommandAt: staleTime,
		ColonyGoal:    "test goal",
	})

	fc := newFileChecker(dir)
	issues := scanSession(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "critical" && contains(issue.Message, "critically stale") {
			found = true
		}
	}
	if !found {
		t.Error("session stale >30 days should produce critical issue")
	}
}

// ---------------------------------------------------------------------------
// TestScanSessionMismatch
// ---------------------------------------------------------------------------

func TestScanSessionMismatch(t *testing.T) {
	dir := t.TempDir()
	goal := "shared goal"
	writeJSONFile(t, dir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 3,
	})
	writeJSONFile(t, dir, "session.json", colony.SessionFile{
		SessionID:     "sess-3",
		ColonyGoal:    goal,
		CurrentPhase:  1,
		LastCommandAt: time.Now().UTC().Format(time.RFC3339),
	})

	fc := newFileChecker(dir)
	issues := scanSession(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "current_phase") {
			found = true
		}
	}
	if !found {
		t.Error("phase mismatch should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanPheromonesInvalidType
// ---------------------------------------------------------------------------

func TestScanPheromonesInvalidType(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, dir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig-1", Type: "INVALID", Active: true, Content: json.RawMessage(`{"text":"hello"}`)},
		},
	})

	fc := newFileChecker(dir)
	issues := scanPheromones(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "Invalid signal type") {
			found = true
		}
	}
	if !found {
		t.Error("invalid signal type should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanPheromonesExpiredActive
// ---------------------------------------------------------------------------

func TestScanPheromonesExpiredActive(t *testing.T) {
	dir := t.TempDir()
	expired := time.Now().AddDate(0, 0, -1).UTC().Format(time.RFC3339)
	expiredStr := expired
	writeJSONFile(t, dir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:        "sig-expired",
				Type:      "FOCUS",
				Active:    true,
				ExpiresAt: &expiredStr,
				Content:   json.RawMessage(`{"text":"focus"}`),
			},
		},
	})

	fc := newFileChecker(dir)
	issues := scanPheromones(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "expired") && contains(issue.Message, "active") {
			found = true
		}
	}
	if !found {
		t.Error("expired-but-active signal should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanPheromonesMissingID
// ---------------------------------------------------------------------------

func TestScanPheromonesMissingID(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, dir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "", Type: "FOCUS", Active: true, Content: json.RawMessage(`{"text":"test"}`)},
		},
	})

	fc := newFileChecker(dir)
	issues := scanPheromones(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "missing ID") {
			found = true
		}
	}
	if !found {
		t.Error("signal with empty ID should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanPheromonesDuplicateContentHash
// ---------------------------------------------------------------------------

func TestScanPheromonesDuplicateContentHash(t *testing.T) {
	dir := t.TempDir()
	hash1 := "abc123"
	hash2 := "abc123"
	writeJSONFile(t, dir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig-1", Type: "FOCUS", Active: true, ContentHash: &hash1, Content: json.RawMessage(`{"text":"a"}`)},
			{ID: "sig-2", Type: "FOCUS", Active: true, ContentHash: &hash2, Content: json.RawMessage(`{"text":"a"}`)},
		},
	})

	fc := newFileChecker(dir)
	issues := scanPheromones(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "Duplicate signal content") {
			found = true
		}
	}
	if !found {
		t.Error("duplicate content hash should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanDataFilesCorrupted
// ---------------------------------------------------------------------------

func TestScanDataFilesCorrupted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "midden.json", []byte(`{corrupted`))

	fc := newFileChecker(dir)
	issues := scanDataFiles(fc)

	// The corrupted file should produce a critical issue from checkJSONFile
	allIssues := append(issues, fc.allIssues()...)
	found := false
	for _, issue := range allIssues {
		if issue.Severity == "critical" && contains(issue.Message, "corrupted") {
			found = true
		}
	}
	if !found {
		t.Error("corrupted data file should produce critical issue")
	}
}

// ---------------------------------------------------------------------------
// TestScanDataFilesGhostConstraints
// ---------------------------------------------------------------------------

func TestScanDataFilesGhostConstraints(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "constraints.json", []byte(`{"focus_areas":["security"]}`))

	fc := newFileChecker(dir)
	issues := scanDataFiles(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "ghost file") {
			found = true
		}
	}
	if !found {
		t.Error("constraints.json with content should produce ghost file warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanJSONLMalformed
// ---------------------------------------------------------------------------

func TestScanJSONLMalformed(t *testing.T) {
	dir := t.TempDir()
	content := `{"id":"1","valid":true}
not a json line
{"id":"2","valid":true}
`
	writeFile(t, dir, "trace.jsonl", []byte(content))

	fc := newFileChecker(dir)
	issues := scanJSONL(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "malformed") {
			found = true
		}
	}
	if !found {
		t.Error("malformed JSONL lines should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanJSONLExpiredEvents
// ---------------------------------------------------------------------------

func TestScanJSONLExpiredEvents(t *testing.T) {
	dir := t.TempDir()
	expired := time.Now().AddDate(0, 0, -5).UTC().Format(time.RFC3339)
	content := fmt.Sprintf(`{"id":"evt-1","topic":"test","expires_at":"%s"}
{"id":"evt-2","topic":"test","expires_at":"2099-01-01T00:00:00Z"}
`, expired)
	writeFile(t, dir, "event-bus.jsonl", []byte(content))

	fc := newFileChecker(dir)
	issues := scanJSONL(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "expired events") {
			found = true
		}
	}
	if !found {
		t.Error("expired events should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanJSONLHealthy
// ---------------------------------------------------------------------------

func TestScanJSONLHealthy(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "trace.jsonl", []byte(`{"id":"trc-1","level":"state","topic":"test"}`+"\n"))

	fc := newFileChecker(dir)
	issues := scanJSONL(fc)

	for _, issue := range issues {
		if issue.Severity == "warning" || issue.Severity == "critical" {
			t.Errorf("healthy JSONL produced issue: %s", issue.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// TestPerformHealthScanIntegration
// ---------------------------------------------------------------------------

func TestPerformHealthScanIntegration(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("AETHER_ROOT", dir)

	goal := "Integration test colony"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhasePending},
			},
		},
	})
	writeJSONFile(t, dataDir, "session.json", colony.SessionFile{
		SessionID:     "int-sess",
		ColonyGoal:    goal,
		CurrentPhase:  0,
		LastCommandAt: time.Now().UTC().Format(time.RFC3339),
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
	})
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig-1", Type: "FOCUS", Active: true, Content: json.RawMessage(`{"text":"focus"}`)},
		},
	})
	writeJSONFile(t, dataDir, "instincts.json", colony.InstinctsFile{
		Version: "1.0",
	})
	writeJSONFile(t, dataDir, "midden.json", colony.MiddenFile{Version: "1.0"})
	writeJSONFile(t, dataDir, "learning-observations.json", colony.LearningFile{})
	writeJSONFile(t, dataDir, "assumptions.json", colony.AssumptionsFile{Version: "1.0"})
	writeJSONFile(t, dataDir, "pending-decisions.json", colony.FlagsFile{Version: "1.0"})
	writeFile(t, dataDir, "constraints.json", []byte(`{}`))
	writeFile(t, dataDir, "trace.jsonl", []byte(`{"id":"trc-1","level":"state","topic":"test"}`+"\n"))
	writeFile(t, dataDir, "event-bus.jsonl", []byte(`{"id":"evt-1","topic":"test"}`+"\n"))

	opts := MedicOptions{}
	result, err := performHealthScan(opts)
	if err != nil {
		t.Fatalf("performHealthScan failed: %v", err)
	}

	if !result.Healthy {
		t.Errorf("expected healthy scan, got issues:")
		for _, issue := range result.Issues {
			if issue.Severity == "critical" {
				t.Errorf("  critical: %s (%s)", issue.Message, issue.File)
			}
		}
	}

	if result.FilesChecked == 0 {
		t.Error("expected files to be checked")
	}
	if result.Duration == 0 {
		t.Error("expected non-zero duration")
	}
}

// ---------------------------------------------------------------------------
// TestPerformHealthScanWithCriticalIssues
// ---------------------------------------------------------------------------

func TestPerformHealthScanWithCriticalIssues(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("AETHER_ROOT", dir)

	// Write COLONY_STATE.json missing goal (critical issue)
	writeJSONFile(t, dataDir, "COLONY_STATE.json", map[string]string{
		"version": "3.0",
		"state":   "READY",
	})

	opts := MedicOptions{}
	result, err := performHealthScan(opts)
	if err != nil {
		t.Fatalf("performHealthScan failed: %v", err)
	}

	if result.Healthy {
		t.Error("expected unhealthy result when critical issues exist")
	}

	foundCritical := false
	for _, issue := range result.Issues {
		if issue.Severity == "critical" {
			foundCritical = true
		}
	}
	if !foundCritical {
		t.Error("expected at least one critical issue")
	}
}

// ---------------------------------------------------------------------------
// TestIssueHelpers
// ---------------------------------------------------------------------------

func TestIssueHelpers(t *testing.T) {
	c := issueCritical("cat", "file.json", "critical msg")
	if c.Severity != "critical" || c.Category != "cat" || c.Message != "critical msg" {
		t.Errorf("issueCritical: %+v", c)
	}
	if c.Fixable {
		t.Error("issueCritical should not be fixable by default")
	}

	w := issueWarning("cat", "file.json", "warning msg")
	if w.Severity != "warning" {
		t.Errorf("issueWarning: %+v", w)
	}

	i := issueInfo("cat", "file.json", "info msg")
	if i.Severity != "info" {
		t.Errorf("issueInfo: %+v", i)
	}

	f := fixableIssue(c)
	if !f.Fixable {
		t.Error("fixableIssue should set Fixable=true")
	}
}

// ---------------------------------------------------------------------------
// TestScanColonyStateExecutingNoPhase
// ---------------------------------------------------------------------------

func TestScanColonyStateExecutingNoPhase(t *testing.T) {
	dir := t.TempDir()
	goal := "test"
	writeJSONFile(t, dir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 0,
	})

	fc := newFileChecker(dir)
	issues := scanColonyState(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "EXECUTING") && contains(issue.Message, "no current phase") {
			found = true
		}
	}
	if !found {
		t.Error("EXECUTING with no phase should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanColonyStatePausedNotReady
// ---------------------------------------------------------------------------

func TestScanColonyStatePausedNotReady(t *testing.T) {
	dir := t.TempDir()
	goal := "test"
	writeJSONFile(t, dir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateEXECUTING,
		Paused:  true,
	})

	fc := newFileChecker(dir)
	issues := scanColonyState(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "Paused") && contains(issue.Message, "READY") {
			found = true
		}
	}
	if !found {
		t.Error("paused with non-READY state should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanColonyStateMalformedEvents
// ---------------------------------------------------------------------------

func TestScanColonyStateMalformedEvents(t *testing.T) {
	dir := t.TempDir()
	goal := "test"
	writeJSONFile(t, dir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Events:  []string{"malformed_no_pipe"},
	})

	fc := newFileChecker(dir)
	issues := scanColonyState(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "Event entry malformed") {
			found = true
		}
	}
	if !found {
		t.Error("malformed event entry should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanColonyStateInvalidParallelMode
// ---------------------------------------------------------------------------

func TestScanColonyStateInvalidParallelMode(t *testing.T) {
	dir := t.TempDir()
	goal := "test"
	writeJSONFile(t, dir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ParallelMode: "invalid-mode",
	})

	fc := newFileChecker(dir)
	issues := scanColonyState(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "parallel_mode") {
			found = true
		}
	}
	if !found {
		t.Error("invalid parallel_mode should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestScanSessionMissingFile
// ---------------------------------------------------------------------------

func TestScanSessionMissingFile(t *testing.T) {
	dir := t.TempDir()
	fc := newFileChecker(dir)
	issues := scanSession(fc)
	// Missing session should not produce critical issues
	for _, issue := range issues {
		if issue.Severity == "critical" {
			t.Errorf("missing session should not produce critical: %s", issue.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// TestScanPheromonesMissingFile
// ---------------------------------------------------------------------------

func TestScanPheromonesMissingFile(t *testing.T) {
	dir := t.TempDir()
	fc := newFileChecker(dir)
	issues := scanPheromones(fc)
	// Missing pheromones should not produce critical issues
	for _, issue := range issues {
		if issue.Severity == "critical" {
			t.Errorf("missing pheromones should not produce critical: %s", issue.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// TestScanSessionMissingSessionID
// ---------------------------------------------------------------------------

func TestScanSessionMissingSessionID(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, dir, "session.json", colony.SessionFile{
		SessionID:  "",
		ColonyGoal: "test",
	})

	fc := newFileChecker(dir)
	issues := scanSession(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "Session ID missing") {
			found = true
		}
	}
	if !found {
		t.Error("empty session ID should produce warning")
	}
}

// ---------------------------------------------------------------------------
// TestMedicReportsABuildPlanThatDisagreesWithTheRecord
// ---------------------------------------------------------------------------

// buildPlanDisagreementFindings returns the medic findings about the current
// phase's build plan and saved record listing different tasks.
func buildPlanDisagreementFindings(t *testing.T, deep bool) []HealthIssue {
	t.Helper()
	result, err := performHealthScan(MedicOptions{Deep: deep})
	if err != nil {
		t.Fatalf("performHealthScan: %v", err)
	}
	var found []HealthIssue
	for _, issue := range result.Issues {
		if issue.Category == "state" && contains(issue.File, filepath.Join("build", "phase-2", "manifest.json")) {
			found = append(found, issue)
		}
	}
	return found
}

// Phase 210 blocker 17: phase 2's build plan listed 2.1, 2.2, 2.3 and task-4
// while the saved project record listed 2.1, 2.2, 2.3, so every save and
// check refused, yet `aether medic --deep` never mentioned it. The build plan
// here is made by the runtime's own planner (codexBuildTaskPlans) from a phase
// that still had an unnumbered fourth task, which is how "task-4" arises; the
// saved record no longer has it.
func TestMedicReportsABuildPlanThatDisagreesWithTheRecord(t *testing.T) {
	saveGlobals(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	project := t.TempDir()
	dataDir := filepath.Join(project, ".aether", "data")
	t.Setenv("AETHER_ROOT", project)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	store = s

	taskID := func(id string) *string { return &id }
	planned := colony.Phase{
		ID:     2,
		Name:   "Finish the deck",
		Status: colony.PhaseInProgress,
		Tasks: []colony.Task{
			{ID: taskID("2.1"), Goal: "Lay out the cards", Status: colony.TaskCompleted},
			{ID: taskID("2.2"), Goal: "Write the prompts", Status: colony.TaskCompleted},
			{ID: taskID("2.3"), Goal: "Add the shuffle", Status: colony.TaskCompleted},
			{Goal: "Finish the shuffle", Status: colony.TaskPending},
		},
	}
	recorded := planned
	recorded.Tasks = append([]colony.Task{}, planned.Tasks[:3]...)

	goal := "Eno-style creative flashcards"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 2,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Set up", Status: colony.PhaseCompleted},
			recorded,
		}},
	})
	manifestRel := filepath.Join("build", "phase-2", "manifest.json")
	saveManifestFrom := func(phase colony.Phase) {
		t.Helper()
		if err := store.SaveJSON(manifestRel, codexBuildManifest{
			Phase:        phase.ID,
			PhaseName:    phase.Name,
			Goal:         goal,
			Root:         project,
			DispatchMode: "real",
			State:        "built",
			Tasks:        codexBuildTaskPlans(phase),
		}); err != nil {
			t.Fatalf("save manifest: %v", err)
		}
	}

	saveManifestFrom(planned)
	for _, deep := range []bool{true, false} {
		findings := buildPlanDisagreementFindings(t, deep)
		if len(findings) != 1 {
			t.Fatalf("deep=%v: want exactly one finding about phase 2's build plan and record disagreeing, got %+v", deep, findings)
		}
		got := findings[0]
		if got.Severity != "warning" {
			t.Errorf("deep=%v: severity = %q, want warning (a report, not a new stop)", deep, got.Severity)
		}
		if got.Fixable {
			t.Errorf("deep=%v: medic must not offer to repair this itself: %+v", deep, got)
		}
		for _, want := range []string{"phase 2", "task-4", "2.1, 2.2, 2.3"} {
			if !contains(got.Message, want) {
				t.Errorf("deep=%v: message %q does not mention %q", deep, got.Message, want)
			}
		}
	}

	// Control: a build plan made from the saved record itself agrees, and
	// medic must then say nothing about it.
	saveManifestFrom(recorded)
	if findings := buildPlanDisagreementFindings(t, true); len(findings) != 0 {
		t.Errorf("an agreeing build plan was reported: %+v", findings)
	}
}

// contains checks if s contains substr (case-sensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
