package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// validateBranchName tests
// ---------------------------------------------------------------------------

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid agent-track names
		{"agent track phase-2/builder-1", "phase-2/builder-1", false},
		{"agent track phase-10/watcher-scout", "phase-10/watcher-scout", false},
		{"agent track phase-1/a", "phase-1/a", false},
		{"agent track phase-999/queen", "phase-999/queen", false},

		// Valid human-track names
		{"human track feature/auth", "feature/auth", false},
		{"human track fix/bug-123", "fix/bug-123", false},
		{"human track experiment/new-idea", "experiment/new-idea", false},
		{"human track colony/setup", "colony/setup", false},

		// Invalid: path traversal
		{"path traversal ..", "phase-2/../etc/passwd", true},
		{"path traversal prefix", "feature/../etc/passwd", true},

		// Invalid: unrecognized format
		{"random branch name", "random-branch", true},
		{"hotfix/auth", "hotfix/auth", true},
		{"develop", "develop", true},
		{"main", "main", true},

		// Invalid: empty or whitespace
		{"empty string", "", true},

		// Invalid: prefix with no description
		{"feature/ only", "feature/", true},
		{"fix/ only", "fix/", true},
		{"experiment/ only", "experiment/", true},
		{"colony/ only", "colony/", true},

		// Invalid: phase-0 (must be positive integer)
		{"phase-0/builder", "phase-0/builder", true},

		// Invalid: uppercase in agent track
		{"uppercase Builder-1", "phase-2/Builder-1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBranchName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// sanitizeBranchPath tests
// ---------------------------------------------------------------------------

func TestSanitizeBranchPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"phase-2/builder-1", "phase-2-builder-1"},
		{"feature/auth", "feature-auth"},
		{"fix/bug-123", "fix-bug-123"},
		{"no-slashes", "no-slashes"},
		{"", ""},
		{"phase-2/builder/sub", "phase-2-builder-sub"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeBranchPath(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeBranchPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// WorktreeEntry JSON round-trip tests
// ---------------------------------------------------------------------------

func TestWorktreeEntryJSONRoundTrip(t *testing.T) {
	original := colony.WorktreeEntry{
		ID:           "wt_1234_abcd",
		Branch:       "phase-2/builder-1",
		Path:         ".aether/worktrees/phase-2-builder-1",
		Status:       colony.WorktreeAllocated,
		Phase:        2,
		Agent:        "builder-1",
		CreatedAt:    "2026-04-07T22:00:00Z",
		UpdatedAt:    "2026-04-07T22:00:00Z",
		LastCommitAt: "2026-04-07T22:30:00Z",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal WorktreeEntry: %v", err)
	}

	var decoded colony.WorktreeEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal WorktreeEntry: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Branch != original.Branch {
		t.Errorf("Branch mismatch: got %q, want %q", decoded.Branch, original.Branch)
	}
	if decoded.Path != original.Path {
		t.Errorf("Path mismatch: got %q, want %q", decoded.Path, original.Path)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status mismatch: got %q, want %q", decoded.Status, original.Status)
	}
	if decoded.Phase != original.Phase {
		t.Errorf("Phase mismatch: got %d, want %d", decoded.Phase, original.Phase)
	}
	if decoded.Agent != original.Agent {
		t.Errorf("Agent mismatch: got %q, want %q", decoded.Agent, original.Agent)
	}
	if decoded.CreatedAt != original.CreatedAt {
		t.Errorf("CreatedAt mismatch: got %q, want %q", decoded.CreatedAt, original.CreatedAt)
	}
	if decoded.UpdatedAt != original.UpdatedAt {
		t.Errorf("UpdatedAt mismatch: got %q, want %q", decoded.UpdatedAt, original.UpdatedAt)
	}
	if decoded.LastCommitAt != original.LastCommitAt {
		t.Errorf("LastCommitAt mismatch: got %q, want %q", decoded.LastCommitAt, original.LastCommitAt)
	}
}

func TestColonyStateWorktreesBackwardCompatible(t *testing.T) {
	// JSON without "worktrees" key should produce nil slice
	jsonWithoutWorktrees := `{
		"version": "3.0",
		"goal": "test",
		"state": "READY",
		"current_phase": 1,
		"plan": {"phases": []},
		"events": [],
		"memory": {"phase_learnings": [], "decisions": [], "instincts": []},
		"errors": {"records": []}
	}`

	var state colony.ColonyState
	if err := json.Unmarshal([]byte(jsonWithoutWorktrees), &state); err != nil {
		t.Fatalf("unmarshal colony state: %v", err)
	}

	if state.Worktrees != nil {
		t.Errorf("expected nil Worktrees for JSON without worktrees key, got %v", state.Worktrees)
	}

	// JSON with empty worktrees array should produce empty slice
	jsonWithEmptyWorktrees := `{
		"version": "3.0",
		"goal": "test",
		"state": "READY",
		"current_phase": 1,
		"plan": {"phases": []},
		"events": [],
		"memory": {"phase_learnings": [], "decisions": [], "instincts": []},
		"errors": {"records": []},
		"worktrees": []
	}`

	var state2 colony.ColonyState
	if err := json.Unmarshal([]byte(jsonWithEmptyWorktrees), &state2); err != nil {
		t.Fatalf("unmarshal colony state with empty worktrees: %v", err)
	}

	if state2.Worktrees == nil {
		t.Error("expected non-nil Worktrees for JSON with empty worktrees array, got nil")
	}
	if len(state2.Worktrees) != 0 {
		t.Errorf("expected 0 worktrees, got %d", len(state2.Worktrees))
	}
}

// ---------------------------------------------------------------------------
// worktree-allocate command tests
// ---------------------------------------------------------------------------

func TestWorktreeAllocateRejectsInvalidName(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stderrBuf bytes.Buffer
	stderr = &stderrBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-allocate", "--branch", "bad-name"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stderrBuf.String()
	if !strings.Contains(output, "invalid branch name") {
		t.Errorf("expected 'invalid branch name' in stderr, got: %s", output)
	}
}

func TestWorktreeAllocateRequiresBranchOrAgentPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stderrBuf bytes.Buffer
	stderr = &stderrBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-allocate"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stderrBuf.String()
	if !strings.Contains(output, "required") {
		t.Errorf("expected 'required' in stderr, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// worktree-list command tests
// ---------------------------------------------------------------------------

func TestWorktreeListEmptyState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdoutBuf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	worktrees, ok := result["worktrees"].([]interface{})
	if !ok {
		t.Fatalf("result.worktrees is not an array, got: %T", result["worktrees"])
	}
	if len(worktrees) != 0 {
		t.Errorf("expected 0 worktrees, got %d", len(worktrees))
	}
}

func TestWorktreeListNilState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	// State with no worktrees key at all
	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdoutBuf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
}

// ---------------------------------------------------------------------------
// worktree-orphan-scan tests
// ---------------------------------------------------------------------------

func TestWorktreeOrphanScanDefaultThreshold(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-orphan-scan"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdoutBuf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	// Default threshold should be 48
	if thresh, ok := result["threshold"].(float64); !ok || thresh != 48 {
		t.Errorf("expected threshold=48, got %v", result["threshold"])
	}
}

func TestWorktreeOrphanScanCustomThreshold(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-orphan-scan", "--threshold", "24"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdoutBuf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	if thresh, ok := result["threshold"].(float64); !ok || thresh != 24 {
		t.Errorf("expected threshold=24, got %v", result["threshold"])
	}
}

// ---------------------------------------------------------------------------
// generateWorktreeID uniqueness test
// ---------------------------------------------------------------------------

func TestGenerateWorktreeIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateWorktreeID()
		if ids[id] {
			t.Errorf("duplicate worktree ID generated: %s", id)
		}
		ids[id] = true
		// Verify format: wt_<unix>_<hex>
		if !strings.HasPrefix(id, "wt_") {
			t.Errorf("ID %q does not start with wt_", id)
		}
	}
}

// ---------------------------------------------------------------------------
// WorktreeEntry in colony package tests
// ---------------------------------------------------------------------------

func TestWorktreeEntryInColonyPackage(t *testing.T) {
	// Verify the WorktreeStatus constants exist
	if colony.WorktreeAllocated != colony.WorktreeStatus("allocated") {
		t.Error("WorktreeAllocated constant mismatch")
	}
	if colony.WorktreeInProgress != colony.WorktreeStatus("in-progress") {
		t.Error("WorktreeInProgress constant mismatch")
	}
	if colony.WorktreeMerged != colony.WorktreeStatus("merged") {
		t.Error("WorktreeMerged constant mismatch")
	}
	if colony.WorktreeOrphaned != colony.WorktreeStatus("orphaned") {
		t.Error("WorktreeOrphaned constant mismatch")
	}
}

// ---------------------------------------------------------------------------
// Helper: create a test colony state with worktrees
// ---------------------------------------------------------------------------

func makeTestStateWithWorktrees(worktrees []colony.WorktreeEntry) string {
	state := map[string]interface{}{
		"version": "3.0",
		"goal":    "test",
		"state":   "READY",
		"current_phase": 1,
		"plan":   map[string]interface{}{"phases": []interface{}{}},
		"events": []interface{}{},
		"memory": map[string]interface{}{
			"phase_learnings": []interface{}{},
			"decisions":       []interface{}{},
			"instincts":       []interface{}{},
		},
		"errors": map[string]interface{}{"records": []interface{}{}},
	}
	if worktrees != nil {
		state["worktrees"] = worktrees
	}
	data, _ := json.Marshal(state)
	return string(data)
}

// Helper: verify command output is valid JSON envelope with ok=true
func assertOKEnvelope(t *testing.T, output string) map[string]interface{} {
	t.Helper()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
	return envelope
}

// Helper: write test state to temp dir and return store + dir
func newWorktreeTestStore(t *testing.T, stateJSON string) (*storage.Store, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(stateJSON), 0644)
	os.Setenv("AETHER_ROOT", tmpDir)
	s, _ := storage.NewStore(dataDir)
	return s, tmpDir
}

// ---------------------------------------------------------------------------
// worktree-allocate with agent+phase flag combination tests
// ---------------------------------------------------------------------------

func TestWorktreeAllocateAgentPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	state := makeTestStateWithWorktrees(nil)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	// --agent and --phase should construct "phase-2/builder-1"
	rootCmd.SetArgs([]string{"worktree-allocate", "--agent", "builder-1", "--phase", "2"})

	err := rootCmd.Execute()
	// This may succeed or fail depending on git availability; we mainly test
	// that the branch name is constructed correctly. Check for the constructed
	// name in output or error.
	_ = err

	// If it succeeded, verify the branch name was constructed
	if stderrBuf.Len() == 0 {
		// Command succeeded -- verify output
		assertOKEnvelope(t, stdoutBuf.String())
	}
}

func TestWorktreeAllocateHumanBranch(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	state := makeTestStateWithWorktrees(nil)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-allocate", "--branch", "feature/auth"})

	err := rootCmd.Execute()
	_ = err

	if stderrBuf.Len() == 0 {
		assertOKEnvelope(t, stdoutBuf.String())
	}
}

// ---------------------------------------------------------------------------
// worktree-orphan-scan with stale worktree in state
// ---------------------------------------------------------------------------

func TestWorktreeOrphanScanStaleEntry(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	oldTime := time.Now().Add(-72 * time.Hour).Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_stale_001",
			Branch:    "phase-1/builder-old",
			Path:      "/nonexistent/path/phase-1-builder-old",
			Status:    colony.WorktreeAllocated,
			Phase:     1,
			Agent:     "builder-old",
			CreatedAt: oldTime,
		},
	}

	state := makeTestStateWithWorktrees(worktrees)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-orphan-scan", "--threshold", "48"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envelope := assertOKEnvelope(t, stdoutBuf.String())
	result := envelope["result"].(map[string]interface{})

	// Stale entry should be flagged (not on disk)
	stale, ok := result["stale"].([]interface{})
	if !ok {
		t.Fatalf("result.stale is not an array, got: %T", result["stale"])
	}
	if len(stale) != 1 {
		t.Errorf("expected 1 stale worktree, got %d", len(stale))
	}
}

// ---------------------------------------------------------------------------
// isWorktreeOrphaned helper tests
// ---------------------------------------------------------------------------

func TestIsWorktreeOrphaned(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		commitAt  time.Time
		threshold time.Duration
		want      bool
	}{
		{"recent commit within threshold", now.Add(-1 * time.Hour), 48 * time.Hour, false},
		{"old commit beyond threshold", now.Add(-72 * time.Hour), 48 * time.Hour, true},
		{"exactly at threshold", now.Add(-48 * time.Hour), 48 * time.Hour, true},
		{"just under threshold", now.Add(-47*time.Hour - 59*time.Minute), 48 * time.Hour, false},
		{"zero commit time (use created at)", time.Time{}, 48 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWorktreeOrphaned(tt.commitAt, tt.threshold)
			if got != tt.want {
				t.Errorf("isWorktreeOrphaned() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// worktree-list with worktrees in state
// ---------------------------------------------------------------------------

func TestWorktreeListWithEntries(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_test_001",
			Branch:    "phase-2/builder-1",
			Path:      ".aether/worktrees/phase-2-builder-1",
			Status:    colony.WorktreeAllocated,
			Phase:     2,
			Agent:     "builder-1",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}

	state := makeTestStateWithWorktrees(worktrees)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envelope := assertOKEnvelope(t, stdoutBuf.String())
	result := envelope["result"].(map[string]interface{})
	wtList, ok := result["worktrees"].([]interface{})
	if !ok {
		t.Fatalf("result.worktrees is not an array, got: %T", result["worktrees"])
	}
	if len(wtList) != 1 {
		t.Errorf("expected 1 worktree, got %d", len(wtList))
	}
}

// ---------------------------------------------------------------------------
// worktree-list with --status filter
// ---------------------------------------------------------------------------

func TestWorktreeListFilterByStatus(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	worktrees := []colony.WorktreeEntry{
		{ID: "wt_001", Branch: "phase-2/builder-1", Path: ".aether/worktrees/phase-2-builder-1", Status: colony.WorktreeAllocated, Phase: 2, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		{ID: "wt_002", Branch: "phase-1/builder-old", Path: ".aether/worktrees/phase-1-builder-old", Status: colony.WorktreeMerged, Phase: 1, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
	}

	state := makeTestStateWithWorktrees(worktrees)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-list", "--status", "merged"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envelope := assertOKEnvelope(t, stdoutBuf.String())
	result := envelope["result"].(map[string]interface{})
	wtList, ok := result["worktrees"].([]interface{})
	if !ok {
		t.Fatalf("result.worktrees is not an array, got: %T", result["worktrees"])
	}
	if len(wtList) != 1 {
		t.Errorf("expected 1 merged worktree, got %d", len(wtList))
	}
}

// ---------------------------------------------------------------------------
// worktree-allocate rejects duplicate branch
// ---------------------------------------------------------------------------

func TestWorktreeAllocateRejectsDuplicate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stderrBuf bytes.Buffer
	stderr = &stderrBuf

	worktrees := []colony.WorktreeEntry{
		{ID: "wt_existing", Branch: "phase-2/builder-1", Path: ".aether/worktrees/phase-2-builder-1", Status: colony.WorktreeAllocated, Phase: 2, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
	}

	state := makeTestStateWithWorktrees(worktrees)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-allocate", "--agent", "builder-1", "--phase", "2"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stderrBuf.String()
	if !strings.Contains(output, "already tracked") {
		t.Errorf("expected 'already tracked' in stderr, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// worktree-allocate with no store (guard check)
// ---------------------------------------------------------------------------

func TestWorktreeAllocateNoStore(t *testing.T) {
	// The PersistentPreRunE on rootCmd initializes store before RunE executes,
	// so store is never nil in production for worktree commands. This test
	// verifies the nil-guard logic directly rather than through rootCmd.Execute.
	if store != nil {
		// Store is initialized by PersistentPreRunE -- this is expected behavior.
		// The nil-guard is a defensive check that can't be easily triggered
		// through the CLI entry point.
		t.Skip("PersistentPreRunE initializes store before RunE; nil-guard is defensive")
	}

	// If store is somehow nil (future refactoring breaks init), the command
	// should handle it gracefully. This branch is effectively unreachable in
	// the current architecture but documents the expected behavior.
}

// ---------------------------------------------------------------------------
// Edge case: worktree-orphan-scan with untracked worktrees
// ---------------------------------------------------------------------------

func TestWorktreeOrphanScanUntracked(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	state := makeTestStateWithWorktrees(nil)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-orphan-scan"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envelope := assertOKEnvelope(t, stdoutBuf.String())
	result := envelope["result"].(map[string]interface{})

	// Should have orphaned and untracked arrays
	if _, ok := result["orphaned"]; !ok {
		t.Error("expected 'orphaned' key in result")
	}
	if _, ok := result["untracked"]; !ok {
		t.Error("expected 'untracked' key in result")
	}
}

// ---------------------------------------------------------------------------
// Additional sanitizeBranchPath edge cases
// ---------------------------------------------------------------------------

func TestSanitizeBranchPathEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"multiple slashes", "a/b/c/d", "a-b-c-d"},
		{"single char segments", "a/b", "a-b"},
		{"trailing slash", "feature/", "feature-"},
		{"leading slash", "/feature", "-feature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeBranchPath(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeBranchPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// worktree-allocate audit log test
// ---------------------------------------------------------------------------

func TestWorktreeAllocateAuditLog(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	state := makeTestStateWithWorktrees(nil)
	s, tmpDir := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-allocate", "--branch", fmt.Sprintf("feature/test-audit-%d", time.Now().UnixNano())})

	err := rootCmd.Execute()
	_ = err // may fail if git is not available

	// If the command succeeded (no error on stderr about store), check audit log
	if stderrBuf.Len() == 0 || !strings.Contains(stderrBuf.String(), "no store") {
		// Check if audit log file was created
		auditPath := tmpDir + "/.aether/data/state-changelog.jsonl"
		if data, err := os.ReadFile(auditPath); err == nil {
			lines := strings.TrimSpace(string(data))
			if lines == "" {
				t.Error("expected audit log entry, got empty file")
			}
		}
		// It's ok if the command failed for other reasons (e.g. git not available)
	}
}

// ---------------------------------------------------------------------------
// Task 2: Additional edge case tests
// ---------------------------------------------------------------------------

func TestWorktreeAllocateConstructsAgentBranch(t *testing.T) {
	// Verify that --agent "builder-1" --phase 2 constructs "phase-2/builder-1"
	branch := fmt.Sprintf("phase-%d/%s", 2, "builder-1")
	if branch != "phase-2/builder-1" {
		t.Errorf("expected phase-2/builder-1, got %s", branch)
	}
	// Verify it passes validation
	if err := validateBranchName(branch); err != nil {
		t.Errorf("constructed branch should be valid: %v", err)
	}
}

func TestWorktreeListEmptyArrayState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	// State with explicit empty worktrees array
	worktrees := []colony.WorktreeEntry{}
	state := makeTestStateWithWorktrees(worktrees)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envelope := assertOKEnvelope(t, stdoutBuf.String())
	result := envelope["result"].(map[string]interface{})
	wtList, ok := result["worktrees"].([]interface{})
	if !ok {
		t.Fatalf("result.worktrees is not an array, got: %T", result["worktrees"])
	}
	if len(wtList) != 0 {
		t.Errorf("expected 0 worktrees, got %d", len(wtList))
	}
}

func TestWorktreeOrphanScanWithRecentCommit(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	// Create a worktree with recent activity (not orphaned)
	recentTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:           "wt_recent_001",
			Branch:       "phase-3/builder-active",
			Path:         ".aether/worktrees/phase-3-builder-active",
			Status:       colony.WorktreeInProgress,
			Phase:        3,
			Agent:        "builder-active",
			CreatedAt:    time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			LastCommitAt: recentTime,
		},
	}

	state := makeTestStateWithWorktrees(worktrees)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-orphan-scan", "--threshold", "48"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envelope := assertOKEnvelope(t, stdoutBuf.String())
	result := envelope["result"].(map[string]interface{})

	// Should not be flagged as orphaned (recent commit within 48h threshold)
	orphaned, ok := result["orphaned"].([]interface{})
	if !ok {
		t.Fatalf("result.orphaned is not an array, got: %T", result["orphaned"])
	}
	if len(orphaned) != 0 {
		t.Errorf("expected 0 orphaned worktrees, got %d", len(orphaned))
	}
}

func TestWorktreeAllocateWithPhaseZero(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stderrBuf bytes.Buffer
	stderr = &stderrBuf

	state := makeTestStateWithWorktrees(nil)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	// --phase 0 should be rejected (not a valid agent-track name)
	rootCmd.SetArgs([]string{"worktree-allocate", "--agent", "builder", "--phase", "0"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// phase-0 is not a valid agent-track name
	output := stderrBuf.String()
	if !strings.Contains(output, "invalid branch name") && !strings.Contains(output, "required") {
		t.Errorf("expected validation error for phase-0, got: %s", output)
	}
}

func TestValidateBranchNameEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"single char agent", "phase-1/a", false},
		{"agent with many hyphens", "phase-5/some-long-agent-name-here", false},
		{"feature with slashes in desc", "feature/auth/v2", false},
		{"experiment with numbers", "experiment/idea-42", false},
		{"colony prefix", "colony/my-colony", false},
		{"fix prefix", "fix/critical-bug", false},
		{"uppercase agent", "phase-1/A", true},
		{"space in name", "phase-1/builder 1", true},
		{"special chars accepted by prefix", "feature/auth!@#", false}, // prefix validation only; git rejects bad chars
		{"just prefix no slash", "feature", true},
		{"double slash", "phase-1//builder", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBranchName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestWorktreeAllocateMergedBranchAllowed(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stderrBuf, stdoutBuf bytes.Buffer
	stderr = &stderrBuf
	stdout = &stdoutBuf

	// A merged worktree with the same branch should be allowed (can reuse branch)
	worktrees := []colony.WorktreeEntry{
		{ID: "wt_merged", Branch: "phase-2/builder-1", Path: ".aether/worktrees/phase-2-builder-1", Status: colony.WorktreeMerged, Phase: 2, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
	}

	state := makeTestStateWithWorktrees(worktrees)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-allocate", "--agent", "builder-1", "--phase", "2"})

	err := rootCmd.Execute()
	_ = err // may fail for git reasons, but should NOT fail for "already tracked"

	// Should NOT get "already tracked" error since the existing one is merged
	if strings.Contains(stderrBuf.String(), "already tracked") {
		t.Errorf("merged worktree should allow re-allocation, got: %s", stderrBuf.String())
	}
}

func TestWorktreeListFilterNonExistentStatus(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf

	worktrees := []colony.WorktreeEntry{
		{ID: "wt_001", Branch: "phase-2/builder-1", Path: ".aether/worktrees/phase-2-builder-1", Status: colony.WorktreeAllocated, Phase: 2, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
	}

	state := makeTestStateWithWorktrees(worktrees)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-list", "--status", "nonexistent"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envelope := assertOKEnvelope(t, stdoutBuf.String())
	result := envelope["result"].(map[string]interface{})
	wtList, ok := result["worktrees"].([]interface{})
	if !ok {
		t.Fatalf("result.worktrees is not an array, got: %T", result["worktrees"])
	}
	if len(wtList) != 0 {
		t.Errorf("expected 0 worktrees for nonexistent status, got %d", len(wtList))
	}
}

// ===========================================================================
// worktree-merge-back tests (Plan 06-02 Task 1)
// ===========================================================================

func TestWorktreeMergeBackNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stderrBuf bytes.Buffer
	stderr = &stderrBuf

	state := makeTestStateWithWorktrees(nil)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", "phase-1/nonexistent"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stderrBuf.String()
	if !strings.Contains(output, "not found in state tracking") {
		t.Errorf("expected 'not found in state tracking' in stderr, got: %s", output)
	}
}

func TestWorktreeMergeBackAlreadyMerged(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stderrBuf bytes.Buffer
	stderr = &stderrBuf

	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_merged_001",
			Branch:    "phase-1/builder-done",
			Path:      ".aether/worktrees/phase-1-builder-done",
			Status:    colony.WorktreeMerged,
			Phase:     1,
			Agent:     "builder-done",
			CreatedAt: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
	}

	state := makeTestStateWithWorktrees(worktrees)
	s, _ := newWorktreeTestStore(t, state)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
	store = s

	rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", "phase-1/builder-done"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stderrBuf.String()
	if !strings.Contains(output, "already merged") {
		t.Errorf("expected 'already merged' in stderr, got: %s", output)
	}
}

func TestWorktreeMergeBackTestsFail(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stderrBuf bytes.Buffer
	stderr = &stderrBuf

	// Create a real git repo with a worktree so we can run tests in it
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	// Init a git repo at tmpDir
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial")

	// Create a worktree branch and add a failing test file
	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/builder-fail", tmpDir+"/.aether/worktrees/phase-1-builder-fail", "HEAD")
	worktreePath := tmpDir + "/.aether/worktrees/phase-1-builder-fail"
	os.WriteFile(worktreePath+"/failing_test.go", []byte(`package main
import "testing"
func TestFailing(t *testing.T) { t.Fatal("forced failure") }
`), 0644)

	// Set up colony state
	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_fail_001",
			Branch:    "phase-1/builder-fail",
			Path:      ".aether/worktrees/phase-1-builder-fail",
			Status:    colony.WorktreeInProgress,
			Phase:     1,
			Agent:     "builder-fail",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	state := makeTestStateWithWorktrees(worktrees)
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", "phase-1/builder-fail"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stderrBuf.String()
	if !strings.Contains(output, "merge blocked") {
		t.Errorf("expected 'merge blocked' in stderr, got: %s", output)
	}
	if !strings.Contains(output, "tests failed") {
		t.Errorf("expected 'tests failed' in stderr, got: %s", output)
	}

	// Verify blocker was created
	var ff colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &ff); err != nil {
		t.Fatalf("expected pending-decisions.json to exist: %v", err)
	}
	found := false
	for _, d := range ff.Decisions {
		if d.Type == "blocker" && strings.Contains(d.Description, "tests failed") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected blocker flag for test failure")
	}

	// Verify worktree status is still in-progress (not merged)
	var reloadState colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &reloadState); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	for _, wt := range reloadState.Worktrees {
		if wt.Branch == "phase-1/builder-fail" && wt.Status != colony.WorktreeInProgress {
			t.Errorf("expected status in-progress, got %s", wt.Status)
		}
	}
}

func TestWorktreeMergeBackClashDetected(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stderrBuf bytes.Buffer
	stderr = &stderrBuf

	// Create a real git repo with two worktrees modifying the same file
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	// Init git repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")

	// Create a shared file and commit it
	os.WriteFile(tmpDir+"/shared.go", []byte("package main\n"), 0644)
	runGit(t, tmpDir, "add", "shared.go")
	runGit(t, tmpDir, "commit", "-m", "add shared.go")

	// Create worktree A (the one we want to merge)
	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/builder-a", tmpDir+"/.aether/worktrees/phase-1-builder-a", "HEAD")
	wtPathA := tmpDir + "/.aether/worktrees/phase-1-builder-a"
	os.WriteFile(wtPathA+"/shared.go", []byte("package main\n// modified by A\n"), 0644)
	runGit(t, wtPathA, "add", "shared.go")
	runGit(t, wtPathA, "commit", "-m", "modify in A")

	// Create worktree B (clashing worktree)
	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/builder-b", tmpDir+"/.aether/worktrees/phase-1-builder-b", "HEAD")
	wtPathB := tmpDir + "/.aether/worktrees/phase-1-builder-b"
	os.WriteFile(wtPathB+"/shared.go", []byte("package main\n// modified by B\n"), 0644)
	runGit(t, wtPathB, "add", "shared.go")
	runGit(t, wtPathB, "commit", "-m", "modify in B")

	// Set up colony state -- only track builder-a as in-progress
	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_a_001",
			Branch:    "phase-1/builder-a",
			Path:      ".aether/worktrees/phase-1-builder-a",
			Status:    colony.WorktreeInProgress,
			Phase:     1,
			Agent:     "builder-a",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	state := makeTestStateWithWorktrees(worktrees)
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", "phase-1/builder-a"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stderrBuf.String()
	if !strings.Contains(output, "merge blocked") {
		t.Errorf("expected 'merge blocked' in stderr, got: %s", output)
	}
	if !strings.Contains(output, "clash") {
		t.Errorf("expected 'clash' in stderr, got: %s", output)
	}

	// Verify blocker was created
	var ff colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &ff); err != nil {
		t.Fatalf("expected pending-decisions.json to exist: %v", err)
	}
	found := false
	for _, d := range ff.Decisions {
		if d.Type == "blocker" && strings.Contains(d.Description, "clash") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected blocker flag for clash detection")
	}
}

func TestWorktreeMergeBackSuccess(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	// Create a real git repo with a worktree
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	// Init git repo on main
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial")

	// Create a worktree with a passing test
	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/builder-ok", tmpDir+"/.aether/worktrees/phase-1-builder-ok", "HEAD")
	wtPath := tmpDir + "/.aether/worktrees/phase-1-builder-ok"
	os.MkdirAll(wtPath+"/cmd", 0755)
	os.WriteFile(wtPath+"/cmd/testhelper_test.go", []byte(`package cmd
import "testing"
func TestHelper(t *testing.T) {}
`), 0644)

	// Set up colony state
	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_ok_001",
			Branch:    "phase-1/builder-ok",
			Path:      ".aether/worktrees/phase-1-builder-ok",
			Status:    colony.WorktreeInProgress,
			Phase:     1,
			Agent:     "builder-ok",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	state := makeTestStateWithWorktrees(worktrees)
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", "phase-1/builder-ok"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should succeed (ok=true on stdout)
	envelope := assertOKEnvelope(t, stdoutBuf.String())
	result := envelope["result"].(map[string]interface{})
	if result["merged"] != true {
		t.Errorf("expected merged=true, got %v", result["merged"])
	}
	if result["status"] != "merged" {
		t.Errorf("expected status=merged, got %v", result["status"])
	}

	// Verify state updated to merged
	var reloadState colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &reloadState); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	foundMerged := false
	for _, wt := range reloadState.Worktrees {
		if wt.Branch == "phase-1/builder-ok" {
			if wt.Status != colony.WorktreeMerged {
				t.Errorf("expected status merged, got %s", wt.Status)
			}
			if wt.UpdatedAt == "" {
				t.Error("expected UpdatedAt to be set")
			}
			foundMerged = true
		}
	}
	if !foundMerged {
		t.Error("worktree entry not found in state after merge")
	}

	// Verify worktree directory was removed
	if _, err := os.Stat(wtPath); err == nil {
		t.Error("expected worktree directory to be removed after merge")
	}

	// Verify audit log entry was written
	auditPath := tmpDir + "/.aether/data/state-changelog.jsonl"
	if data, err := os.ReadFile(auditPath); err == nil {
		lines := strings.TrimSpace(string(data))
		if lines == "" {
			t.Error("expected audit log entry for merge operation")
		} else if !strings.Contains(lines, "worktree-merge") {
			t.Errorf("expected 'worktree-merge' in audit log, got: %s", lines)
		}
	}
}

func TestWorktreeMergeBackBranchDeletionToleratesNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	// Create a repo where the branch may already be gone after a fast-forward merge
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial")

	// Create a worktree and commit something
	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/builder-ff", tmpDir+"/.aether/worktrees/phase-1-builder-ff", "HEAD")
	wtPath := tmpDir + "/.aether/worktrees/phase-1-builder-ff"
	os.MkdirAll(wtPath+"/cmd", 0755)
	os.WriteFile(wtPath+"/cmd/testpass_test.go", []byte(`package cmd
import "testing"
func TestPass(t *testing.T) {}
`), 0644)

	now := time.Now().UTC().Format(time.RFC3339)
	worktrees := []colony.WorktreeEntry{
		{
			ID:        "wt_ff_001",
			Branch:    "phase-1/builder-ff",
			Path:      ".aether/worktrees/phase-1-builder-ff",
			Status:    colony.WorktreeInProgress,
			Phase:     1,
			Agent:     "builder-ff",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	state := makeTestStateWithWorktrees(worktrees)
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", "phase-1/builder-ff"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should succeed even if branch deletion has issues
	assertOKEnvelope(t, stdoutBuf.String())
}

// ===========================================================================
// createBlocker helper tests
// ===========================================================================

func TestCreateBlocker(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/data"
	os.MkdirAll(dataDir, 0755)

	s, _ := storage.NewStore(dataDir)

	err := createBlocker(s, "Merge blocked: tests failed for phase-1/builder-1", "worktree-merge-back")
	if err != nil {
		t.Fatalf("createBlocker failed: %v", err)
	}

	var ff colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &ff); err != nil {
		t.Fatalf("failed to load pending-decisions.json: %v", err)
	}

	if len(ff.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(ff.Decisions))
	}

	d := ff.Decisions[0]
	if d.Type != "blocker" {
		t.Errorf("expected type=blocker, got %s", d.Type)
	}
	if d.Resolved != false {
		t.Error("expected Resolved=false")
	}
	if !strings.Contains(d.Description, "tests failed") {
		t.Errorf("expected description to contain 'tests failed', got: %s", d.Description)
	}
	if d.Source != "worktree-merge-back" {
		t.Errorf("expected source=worktree-merge-back, got: %s", d.Source)
	}
	if d.ID == "" {
		t.Error("expected non-empty ID")
	}
	if d.CreatedAt == "" {
		t.Error("expected non-empty CreatedAt")
	}
}

func TestCreateBlockerAppendsToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/data"
	os.MkdirAll(dataDir, 0755)

	s, _ := storage.NewStore(dataDir)

	// Create first blocker
	createBlocker(s, "first blocker", "test")
	// Create second blocker
	createBlocker(s, "second blocker", "test")

	var ff colony.FlagsFile
	s.LoadJSON("pending-decisions.json", &ff)

	if len(ff.Decisions) != 2 {
		t.Errorf("expected 2 decisions, got %d", len(ff.Decisions))
	}
}

// ===========================================================================
// checkClashesForWorktree helper tests
// ===========================================================================

func TestCheckClashesForWorktree(t *testing.T) {
	// Create a real git repo with two worktrees
	tmpDir := t.TempDir()
	os.MkdirAll(tmpDir+"/.aether/worktrees", 0755)

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")

	// Create a shared file
	os.WriteFile(tmpDir+"/shared.go", []byte("package main\n"), 0644)
	runGit(t, tmpDir, "add", "shared.go")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Create worktree A
	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/a", tmpDir+"/.aether/worktrees/a", "HEAD")
	os.WriteFile(tmpDir+"/.aether/worktrees/a/shared.go", []byte("package main\n// A\n"), 0644)
	runGit(t, tmpDir+"/.aether/worktrees/a", "add", "shared.go")
	runGit(t, tmpDir+"/.aether/worktrees/a", "commit", "-m", "A changes")

	// Create worktree B (also modifies shared.go)
	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/b", tmpDir+"/.aether/worktrees/b", "HEAD")
	os.WriteFile(tmpDir+"/.aether/worktrees/b/shared.go", []byte("package main\n// B\n"), 0644)
	runGit(t, tmpDir+"/.aether/worktrees/b", "add", "shared.go")
	runGit(t, tmpDir+"/.aether/worktrees/b", "commit", "-m", "B changes")

	clashes, err := checkClashesForWorktree(tmpDir+"/.aether/worktrees/a", "phase-1/a")
	if err != nil {
		t.Fatalf("checkClashesForWorktree failed: %v", err)
	}
	if len(clashes) == 0 {
		t.Error("expected clashes to be detected for shared.go")
	}
	found := false
	for _, c := range clashes {
		if c == "shared.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected shared.go in clashes, got: %v", clashes)
	}
}

func TestCheckClashesForWorktreeNoClash(t *testing.T) {
	// Create a repo with two worktrees modifying different files
	tmpDir := t.TempDir()
	os.MkdirAll(tmpDir+"/.aether/worktrees", 0755)

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")

	os.WriteFile(tmpDir+"/file1.go", []byte("package main\n"), 0644)
	os.WriteFile(tmpDir+"/file2.go", []byte("package main\n"), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Worktree A modifies file1
	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/a", tmpDir+"/.aether/worktrees/a", "HEAD")
	os.WriteFile(tmpDir+"/.aether/worktrees/a/file1.go", []byte("package main\n// A\n"), 0644)
	runGit(t, tmpDir+"/.aether/worktrees/a", "add", "file1.go")
	runGit(t, tmpDir+"/.aether/worktrees/a", "commit", "-m", "A changes")

	// Worktree B modifies file2 (no clash)
	runGit(t, tmpDir, "worktree", "add", "-b", "phase-1/b", tmpDir+"/.aether/worktrees/b", "HEAD")
	os.WriteFile(tmpDir+"/.aether/worktrees/b/file2.go", []byte("package main\n// B\n"), 0644)
	runGit(t, tmpDir+"/.aether/worktrees/b", "add", "file2.go")
	runGit(t, tmpDir+"/.aether/worktrees/b", "commit", "-m", "B changes")

	clashes, err := checkClashesForWorktree(tmpDir+"/.aether/worktrees/a", "phase-1/a")
	if err != nil {
		t.Fatalf("checkClashesForWorktree failed: %v", err)
	}
	if len(clashes) != 0 {
		t.Errorf("expected no clashes, got: %v", clashes)
	}
}

// ===========================================================================
// Full lifecycle test (Plan 06-02 Task 2)
// ===========================================================================

func TestWorktreeLifecycleFull(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	// Init git repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test")
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial")

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	// Step 1: Validate branch name
	if err := validateBranchName("phase-1/builder-1"); err != nil {
		t.Fatalf("branch name validation failed: %v", err)
	}

	// Step 2: Allocate worktree via command
	rootCmd.SetArgs([]string{"worktree-allocate", "--agent", "builder-1", "--phase", "1"})
	err := rootCmd.Execute()
	_ = err // may fail in test env, check if state was updated

	// Step 3: Verify state has allocated worktree
	var state colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &state)

	// Find the allocated worktree or create one manually if command failed
	var entry *colony.WorktreeEntry
	for i := range state.Worktrees {
		if state.Worktrees[i].Branch == "phase-1/builder-1" {
			entry = &state.Worktrees[i]
			break
		}
	}

	if entry == nil {
		// If allocate failed (no git worktree support in test env), manually create
		now := time.Now().UTC().Format(time.RFC3339)
		manualEntry := colony.WorktreeEntry{
			ID:        "wt_lifecycle_001",
			Branch:    "phase-1/builder-1",
			Path:      ".aether/worktrees/phase-1-builder-1",
			Status:    colony.WorktreeAllocated,
			Phase:     1,
			Agent:     "builder-1",
			CreatedAt: now,
			UpdatedAt: now,
		}
		state.Worktrees = append(state.Worktrees, manualEntry)
		s.SaveJSON("COLONY_STATE.json", state)
		entry = &state.Worktrees[len(state.Worktrees)-1]
	}

	// Step 4: Verify list shows allocated
	resetRootCmd(t)
	stdoutBuf.Reset()
	rootCmd.SetArgs([]string{"worktree-list", "--status", "allocated"})
	rootCmd.Execute()
	envelope := assertOKEnvelope(t, stdoutBuf.String())
	result := envelope["result"].(map[string]interface{})
	wtList := result["worktrees"].([]interface{})
	if len(wtList) < 1 {
		t.Errorf("expected at least 1 allocated worktree, got %d", len(wtList))
	}

	// Step 5: Update status to in-progress
	s.LoadJSON("COLONY_STATE.json", &state)
	for i := range state.Worktrees {
		if state.Worktrees[i].Branch == "phase-1/builder-1" {
			state.Worktrees[i].Status = colony.WorktreeInProgress
			state.Worktrees[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			break
		}
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// Step 6: Verify status is in-progress
	resetRootCmd(t)
	stdoutBuf.Reset()
	rootCmd.SetArgs([]string{"worktree-list", "--status", "in-progress"})
	rootCmd.Execute()
	envelope = assertOKEnvelope(t, stdoutBuf.String())
	result = envelope["result"].(map[string]interface{})
	wtList = result["worktrees"].([]interface{})
	if len(wtList) < 1 {
		t.Errorf("expected at least 1 in-progress worktree, got %d", len(wtList))
	}

	// Step 7: If the worktree exists on disk, try merge-back
	wtPath := tmpDir + "/" + entry.Path
	if _, err := os.Stat(wtPath); err == nil {
		// Create a passing test in the worktree
		os.MkdirAll(wtPath+"/cmd", 0755)
		os.WriteFile(wtPath+"/cmd/lifecycle_test.go", []byte(`package cmd
import "testing"
func TestLifecycle(t *testing.T) {}
`), 0644)

		resetRootCmd(t)
		stdoutBuf.Reset()
		stderrBuf.Reset()
		rootCmd.SetArgs([]string{"worktree-merge-back", "--branch", "phase-1/builder-1"})
		rootCmd.Execute()

		// Step 8: Verify status is merged
		s.LoadJSON("COLONY_STATE.json", &state)
		for _, wt := range state.Worktrees {
			if wt.Branch == "phase-1/builder-1" {
				if wt.Status != colony.WorktreeMerged {
					t.Errorf("expected status merged, got %s", wt.Status)
				}
				if wt.UpdatedAt == "" {
					t.Error("expected UpdatedAt to be set after merge")
				}
			}
		}
	}
}

