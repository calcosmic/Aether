package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"time"
)

// setupSealTestStore creates a fresh temp store with a minimal colony state
// where all phases are completed, ready for seal.
func setupSealTestStore(t *testing.T) (*storage.Store, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	goal := "Test colony goal"
	state := colony.ColonyState{
		Goal:         &goal,
		CurrentPhase: 1,
		State:        colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Discovery", Status: colony.PhaseCompleted},
			},
		},
		Memory: colony.Memory{
			PhaseLearnings: []colony.PhaseLearning{},
		},
		Events: []string{},
	}

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Create local QUEEN.md for promotion tests
	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	if err := os.WriteFile(queenPath, []byte(queenDefaultContent), 0644); err != nil {
		t.Fatal(err)
	}

	return s, tmpDir
}

// runSealCmd runs the seal command with the given args and returns stdout/stderr output.
func runSealCmd(t *testing.T, s *storage.Store, tmpDir string, args []string) (string, string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	t.Setenv("COLONY_DATA_DIR", dataDir)

	store = s
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	stdout = outBuf
	stderr = errBuf

	allArgs := append([]string{"seal"}, args...)
	rootCmd.SetArgs(allArgs)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.Execute()

	return outBuf.String(), errBuf.String()
}

// TestSealBlockerCheck verifies that seal blocks when blocker flags exist.
func TestSealBlockerCheck(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add a blocker flag
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{
				ID:          "blk-001",
				Type:        "blocker",
				Description: "Critical issue blocking seal",
				Resolved:    false,
				CreatedAt:   "2026-04-27",
				Source:      "test",
			},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatal(err)
	}

	_, errOut := runSealCmd(t, s, tmpDir, nil)

	// Should output error containing BLOCKED
	if !strings.Contains(errOut, "BLOCKED") {
		t.Errorf("expected error output to contain 'BLOCKED', got: %s", errOut)
	}
	if !strings.Contains(errOut, "blk-001") {
		t.Errorf("expected error output to contain blocker ID 'blk-001', got: %s", errOut)
	}

	// Verify colony state was NOT mutated (still READY, not COMPLETED)
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State == colony.StateCOMPLETED {
		t.Error("seal should not have mutated colony state when blockers exist")
	}
}

// TestSealForceBlockers verifies that seal --force proceeds despite blockers.
func TestSealForceBlockers(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add a blocker flag
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{
				ID:          "blk-002",
				Type:        "blocker",
				Description: "Critical issue",
				Resolved:    false,
				CreatedAt:   "2026-04-27",
				Source:      "test",
			},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatal(err)
	}

	out, _ := runSealCmd(t, s, tmpDir, []string{"--force"})

	// Should contain the warning about overriding
	if !strings.Contains(out, "WARNING: Overriding") {
		t.Errorf("expected stdout to contain override warning, got: %s", out)
	}

	// Verify colony state WAS mutated (COMPLETED)
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Errorf("expected state COMPLETED, got: %s", state.State)
	}
}

// TestSealIssueWarning verifies that seal proceeds with a warning when issues exist but no blockers.
func TestSealIssueWarning(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add an issue (not blocker) flag
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{
				ID:          "issue-001",
				Type:        "issue",
				Description: "Non-critical issue",
				Resolved:    false,
				CreatedAt:   "2026-04-27",
				Source:      "test",
			},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatal(err)
	}

	out, _ := runSealCmd(t, s, tmpDir, nil)

	// Should contain the NOTE about unresolved issues
	if !strings.Contains(out, "NOTE:") {
		t.Errorf("expected stdout to contain NOTE about issues, got: %s", out)
	}

	// Verify colony state WAS mutated (seal proceeded)
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Errorf("expected state COMPLETED, got: %s", state.State)
	}
}

// TestCheckSealBlockers unit tests the checkSealBlockers helper.
func TestCheckSealBlockers(t *testing.T) {
	s, _ := setupSealTestStore(t)

	// No flags file: should return empty
	blockers, issues := checkSealBlockers(s)
	if len(blockers) != 0 || len(issues) != 0 {
		t.Errorf("expected empty with no flags file, got %d blockers, %d issues", len(blockers), len(issues))
	}

	// Mixed flags
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{ID: "b1", Type: "blocker", Resolved: false},
			{ID: "b2", Type: "blocker", Resolved: true},
			{ID: "i1", Type: "issue", Resolved: false},
			{ID: "n1", Type: "note", Resolved: false},
		},
	}
	_ = s.SaveJSON("pending-decisions.json", flags)

	blockers, issues = checkSealBlockers(s)
	if len(blockers) != 1 || blockers[0].ID != "b1" {
		t.Errorf("expected 1 unresolved blocker 'b1', got %d: %v", len(blockers), blockers)
	}
	if len(issues) != 1 || issues[0].ID != "i1" {
		t.Errorf("expected 1 unresolved issue 'i1', got %d: %v", len(issues), issues)
	}
}

// TestRenderBlockerSummary verifies the blocker summary table output.
func TestRenderBlockerSummary(t *testing.T) {
	blockers := []colony.FlagEntry{
		{ID: "blk-001", Description: "Critical blocker", Type: "blocker", CreatedAt: "2026-04-27"},
	}
	issues := []colony.FlagEntry{
		{ID: "issue-001", Description: "Non-critical", Type: "issue", CreatedAt: "2026-04-27"},
	}

	out := renderBlockerSummary(blockers, issues)

	if !strings.Contains(out, "blk-001") {
		t.Error("summary should contain blocker ID")
	}
	if !strings.Contains(out, "BLOCKED") {
		t.Error("summary should contain BLOCKED message")
	}
	if !strings.Contains(out, "--resolve") {
		t.Error("summary should contain resolution hint")
	}
	if !strings.Contains(out, "issue-severity") {
		t.Error("summary should mention issue-severity flags")
	}
}

// TestCountResolvedFlags unit tests the countResolvedFlags helper.
func TestCountResolvedFlags(t *testing.T) {
	s, _ := setupSealTestStore(t)

	// No flags file
	count := countResolvedFlags(s)
	if count != 0 {
		t.Errorf("expected 0 with no flags file, got %d", count)
	}

	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{ID: "b1", Resolved: true},
			{ID: "b2", Resolved: true},
			{ID: "i1", Resolved: false},
		},
	}
	_ = s.SaveJSON("pending-decisions.json", flags)

	count = countResolvedFlags(s)
	if count != 2 {
		t.Errorf("expected 2 resolved, got %d", count)
	}
}

// TestSealBlockerSummaryJSON verifies the error output is valid JSON.
func TestSealBlockerSummaryJSON(t *testing.T) {
	s, _ := setupSealTestStore(t)

	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{ID: "blk-json", Type: "blocker", Description: "JSON test blocker", Resolved: false, CreatedAt: "2026-04-27", Source: "test"},
		},
	}
	_ = s.SaveJSON("pending-decisions.json", flags)

	// Use JSON output mode (no visual rendering)
	saveGlobals(t)
	resetRootCmd(t)
	store = s
	dataDir := s.BasePath()
	t.Setenv("COLONY_DATA_DIR", dataDir)
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	stdout = outBuf
	stderr = errBuf
	os.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Cleanup(func() { os.Unsetenv("AETHER_OUTPUT_MODE") })

	rootCmd.SetArgs([]string{"seal"})
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.Execute()

	// stderr should be valid JSON envelope
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(errBuf.String())), &envelope); err != nil {
		t.Fatalf("expected valid JSON error, got: %s", errBuf.String())
	}
	if ok, _ := envelope["ok"].(bool); ok {
		t.Error("expected ok:false in error envelope")
	}

	// State should not be mutated
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State == colony.StateCOMPLETED {
		t.Error("seal should not have completed when blockers exist")
	}
}

// TestSealPromoteInstincts verifies that seal promotes high-confidence instincts
// to local QUEEN.md only (not global).
func TestSealPromoteInstincts(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add instincts with confidence >= 0.8
	instincts := colony.InstinctsFile{
		Version: "1",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst-001",
				Trigger:    "test pattern",
				Action:     "Always write tests first",
				Domain:     "testing",
				Confidence: 0.9,
				Archived:   false,
			},
			{
				ID:         "inst-002",
				Trigger:    "low confidence pattern",
				Action:     "Maybe do something",
				Domain:     "general",
				Confidence: 0.5,
				Archived:   false,
			},
		},
	}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatal(err)
	}

	out, _ := runSealCmd(t, s, tmpDir, nil)

	// Should contain SUGGESTION for hive promotion (1 instinct >= 0.8)
	if !strings.Contains(out, "SUGGESTION:") {
		t.Errorf("expected SUGGESTION line, got: %s", out)
	}

	// Verify local QUEEN.md has the promoted instinct
	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	queenData, err := os.ReadFile(queenPath)
	if err != nil {
		t.Fatal(err)
	}
	queenText := string(queenData)
	if !strings.Contains(queenText, "inst-001") {
		t.Error("local QUEEN.md should contain promoted instinct inst-001")
	}
	if !strings.Contains(queenText, "Always write tests first") {
		t.Error("local QUEEN.md should contain the instinct action text")
	}
	// Low-confidence instinct should NOT be promoted
	if strings.Contains(queenText, "inst-002") {
		t.Error("local QUEEN.md should NOT contain low-confidence instinct inst-002")
	}
}

// TestSealHiveEligibleLog verifies that seal outputs a SUGGESTION line for hive-eligible instincts.
func TestSealHiveEligibleLog(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add instincts with confidence >= 0.8
	instincts := colony.InstinctsFile{
		Version: "1",
		Instincts: []colony.InstinctEntry{
			{ID: "hive-1", Trigger: "t1", Action: "a1", Domain: "d1", Confidence: 0.85, Archived: false},
			{ID: "hive-2", Trigger: "t2", Action: "a2", Domain: "d2", Confidence: 0.95, Archived: false},
		},
	}
	_ = s.SaveJSON("instincts.json", instincts)

	out, _ := runSealCmd(t, s, tmpDir, nil)

	if !strings.Contains(out, "SUGGESTION:") {
		t.Error("expected SUGGESTION line for hive-eligible instincts")
	}
	if !strings.Contains(out, "2 instinct(s) eligible") {
		t.Errorf("expected 2 hive-eligible instincts, got: %s", out)
	}
}

// TestSealExpireFocus verifies that seal expires all FOCUS pheromones
// while preserving REDIRECT pheromones.
func TestSealExpireFocus(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	now := time.Now().UTC().Format(time.RFC3339)
	pheromones := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "focus-1", Type: "FOCUS", Content: json.RawMessage(`"pay attention here"`), Active: true, CreatedAt: now},
			{ID: "focus-2", Type: "FOCUS", Content: json.RawMessage(`"another focus"`), Active: true, CreatedAt: now},
			{ID: "redirect-1", Type: "REDIRECT", Content: json.RawMessage(`"never do this"`), Active: true, CreatedAt: now},
			{ID: "feedback-1", Type: "FEEDBACK", Content: json.RawMessage(`"adjust this"`), Active: true, CreatedAt: now},
			{ID: "focus-3", Type: "FOCUS", Content: json.RawMessage(`"expired focus"`), Active: false, CreatedAt: now, ExpiresAt: &now},
		},
	}
	if err := s.SaveJSON("pheromones.json", pheromones); err != nil {
		t.Fatal(err)
	}

	runSealCmd(t, s, tmpDir, nil)

	// Verify FOCUS signals are expired
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatal(err)
	}
	for _, sig := range pf.Signals {
		switch sig.ID {
		case "focus-1", "focus-2":
			if sig.Active {
				t.Errorf("FOCUS signal %s should be expired after seal", sig.ID)
			}
		case "focus-3":
			if sig.Active {
				t.Error("already-expired FOCUS signal should remain expired")
			}
		case "redirect-1":
			if !sig.Active {
				t.Error("REDIRECT signal should be preserved after seal")
			}
		case "feedback-1":
			if !sig.Active {
				t.Error("FEEDBACK signal should be preserved after seal")
			}
		}
	}
}

// TestCrownedAnthillEnrichment verifies that CROWNED-ANTHILL.md contains
// the Colony Statistics table with all 5 metrics.
func TestCrownedAnthillEnrichment(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add some learnings
	state := colony.ColonyState{}
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	state.Memory.PhaseLearnings = []colony.PhaseLearning{
		{Phase: 1, Learnings: []colony.Learning{{Claim: "Learned something useful"}}},
		{Phase: 1, Learnings: []colony.Learning{{Claim: "Another learning"}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Add resolved flags
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{ID: "r1", Resolved: true},
			{ID: "r2", Resolved: true},
			{ID: "r3", Resolved: false},
		},
	}
	_ = s.SaveJSON("pending-decisions.json", flags)

	runSealCmd(t, s, tmpDir, nil)

	// Read CROWNED-ANTHILL.md
	anthillPath := filepath.Join(tmpDir, ".aether", "CROWNED-ANTHILL.md")
	data, err := os.ReadFile(anthillPath)
	if err != nil {
		t.Fatalf("CROWNED-ANTHILL.md not found: %v", err)
	}
	content := string(data)

	// Verify Colony Statistics table
	if !strings.Contains(content, "## Colony Statistics") {
		t.Error("CROWNED-ANTHILL.md should contain '## Colony Statistics' section")
	}
	if !strings.Contains(content, "| Learnings captured | 2 |") {
		t.Error("CROWNED-ANTHILL.md should show 2 learnings captured")
	}
	if !strings.Contains(content, "| Flags resolved | 2 |") {
		t.Error("CROWNED-ANTHILL.md should show 2 flags resolved")
	}
	if !strings.Contains(content, "| FOCUS signals expired | 0 |") {
		t.Error("CROWNED-ANTHILL.md should show FOCUS signals expired metric")
	}
	if !strings.Contains(content, "| Hive-eligible instincts | 0 |") {
		t.Error("CROWNED-ANTHILL.md should show hive-eligible instincts metric")
	}
	if !strings.Contains(content, "| Instincts promoted | 0 |") {
		t.Error("CROWNED-ANTHILL.md should show instincts promoted metric")
	}

	// Verify Signal Cleanup section
	if !strings.Contains(content, "### Signal Cleanup") {
		t.Error("CROWNED-ANTHILL.md should contain '### Signal Cleanup' section")
	}
	if !strings.Contains(content, "REDIRECT signals preserved") {
		t.Error("CROWNED-ANTHILL.md should mention REDIRECT signals preserved")
	}
}

// TestExpireSignalsByType unit tests the expireSignalsByType helper.
func TestExpireSignalsByType(t *testing.T) {
	s, _ := setupSealTestStore(t)

	now := time.Now().UTC().Format(time.RFC3339)
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "f1", Type: "FOCUS", Active: true, CreatedAt: now},
			{ID: "f2", Type: "FOCUS", Active: true, CreatedAt: now},
			{ID: "r1", Type: "REDIRECT", Active: true, CreatedAt: now},
			{ID: "f3", Type: "FOCUS", Active: false, CreatedAt: now, ExpiresAt: &now},
		},
	}
	_ = s.SaveJSON("pheromones.json", pf)

	// Expire FOCUS signals
	count := expireSignalsByType(s, "FOCUS")
	if count != 2 {
		t.Errorf("expected 2 FOCUS signals expired, got %d", count)
	}

	// Verify REDIRECT still active
	var loaded colony.PheromoneFile
	_ = s.LoadJSON("pheromones.json", &loaded)
	for _, sig := range loaded.Signals {
		if sig.ID == "r1" && !sig.Active {
			t.Error("REDIRECT signal should still be active")
		}
	}

	// Expiring again should return 0
	count2 := expireSignalsByType(s, "FOCUS")
	if count2 != 0 {
		t.Errorf("expected 0 on second expire, got %d", count2)
	}
}

// TestPromoteInstinctLocal unit tests the promoteInstinctLocal helper.
func TestPromoteInstinctLocal(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)
	saveGlobals(t)
	store = s
	t.Setenv("COLONY_DATA_DIR", s.BasePath())

	err := promoteInstinctLocal(s, "test-inst-1", "Write tests before code")
	if err != nil {
		t.Fatalf("promoteInstinctLocal failed: %v", err)
	}

	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	data, err := os.ReadFile(queenPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "test-inst-1") {
		t.Error("QUEEN.md should contain instinct ID")
	}
	if !strings.Contains(content, "Write tests before code") {
		t.Error("QUEEN.md should contain instinct action")
	}
	if !strings.Contains(content, "## Wisdom") {
		t.Error("Entry should be in Wisdom section")
	}
}

// TestBuildSealSummaryEnrichment unit tests the enriched buildSealSummary.
func TestBuildSealSummaryEnrichment(t *testing.T) {
	goal := "Test goal"
	state := colony.ColonyState{
		Goal:         &goal,
		CurrentPhase: 3,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "P1", Status: colony.PhaseCompleted},
				{ID: 2, Name: "P2", Status: colony.PhaseCompleted},
				{ID: 3, Name: "P3", Status: colony.PhaseCompleted},
			},
		},
	}

	enrichment := sealEnrichment{
		LearningsCount:    5,
		InstinctsPromoted: []string{"inst-1", "inst-2"},
		HiveEligible:      3,
		SignalsExpired:    4,
		FlagsResolved:     2,
	}

	summary := buildSealSummary(state, "2026-04-27T12:00:00Z", nil, enrichment)

	if !strings.Contains(summary, "## Colony Statistics") {
		t.Error("summary should contain Colony Statistics section")
	}
	if !strings.Contains(summary, "| Learnings captured | 5 |") {
		t.Error("summary should show 5 learnings")
	}
	if !strings.Contains(summary, "| Instincts promoted | 2 |") {
		t.Error("summary should show 2 promoted instincts")
	}
	if !strings.Contains(summary, "### Promoted Instincts") {
		t.Error("summary should contain Promoted Instincts section")
	}
	if !strings.Contains(summary, "- inst-1") {
		t.Error("summary should list inst-1 in promoted instincts")
	}
	if !strings.Contains(summary, "### Signal Cleanup") {
		t.Error("summary should contain Signal Cleanup section")
	}
	if !strings.Contains(summary, "FOCUS signals expired: 4") {
		t.Error("summary should show 4 expired FOCUS signals")
	}
}
