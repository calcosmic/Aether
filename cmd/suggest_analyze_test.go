package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupSuggestAnalyzeTest creates a temp dir with .aether/data, sets up store
// and stdout capture. Returns the temp dir and bytes buffer.
func setupSuggestAnalyzeTest(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	store = s

	// Create a minimal colony state with a goal so loadActiveColonyState succeeds.
	goal := "test goal"
	cs := colony.ColonyState{
		Version: "1.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	data, _ := json.Marshal(cs)
	_ = store.AtomicWrite("COLONY_STATE.json", data)

	return tmpDir, &buf
}

// setupPheromones writes pheromones.json with the given signals to the store.
func setupPheromones(t *testing.T, signals []colony.PheromoneSignal) {
	t.Helper()
	pf := colony.PheromoneFile{Signals: signals}
	data, _ := json.Marshal(pf)
	_ = store.AtomicWrite("pheromones.json", data)
}

// execGitRevParse runs git rev-parse HEAD in the given directory.
func execGitRevParse(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// initGitRepo initializes a bare git repo in dir for testing.
func initTestGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
}

// ---------------------------------------------------------------------------
// Test 1: suggest-analyze returns ok:true with suggestions when patterns detected
// ---------------------------------------------------------------------------

func TestSuggestAnalyze_ReturnsSuggestions(t *testing.T) {
	tmpDir, buf := setupSuggestAnalyzeTest(t)
	defer os.RemoveAll(tmpDir)

	// Create a .env file to trigger the "secrets" pattern
	envPath := filepath.Join(tmpDir, ".env")
	_ = os.WriteFile(envPath, []byte("SECRET_KEY=abc123\n"), 0644)

	rootCmd.SetArgs([]string{"suggest-analyze", "--target", tmpDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	suggestions, ok := result["suggestions"].([]interface{})
	if !ok {
		t.Fatalf("expected suggestions array, got %T", result["suggestions"])
	}
	if len(suggestions) == 0 {
		t.Error("expected at least 1 suggestion from .env detection")
	}
}

// ---------------------------------------------------------------------------
// Test 2: suggest-analyze filters out suggestions matching active pheromones
// ---------------------------------------------------------------------------

func TestSuggestAnalyze_FiltersActivePheromones(t *testing.T) {
	tmpDir, buf := setupSuggestAnalyzeTest(t)
	defer os.RemoveAll(tmpDir)

	// Create a .env file
	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("KEY=val\n"), 0644)

	// Pre-create an active pheromone with the same content hash as what
	// generatePheromoneSuggestions would produce for the .env pattern.
	content := "never commit secrets or .env files to version control"
	hash := "sha256:" + sha256Sum(content)
	sig := colony.PheromoneSignal{
		ID:          "sig_test_1",
		Type:        "REDIRECT",
		Content:     json.RawMessage(`{"text":"never commit secrets or .env files to version control"}`),
		Active:      true,
		ContentHash: &hash,
	}
	setupPheromones(t, []colony.PheromoneSignal{sig})

	rootCmd.SetArgs([]string{"suggest-analyze", "--target", tmpDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	suggestions := result["suggestions"].([]interface{})

	// The .env suggestion should have been filtered out because an active
	// pheromone with the same type (REDIRECT) and content hash exists.
	for _, s := range suggestions {
		m := s.(map[string]interface{})
		if m["content"] == content {
			t.Errorf("suggestion matching active pheromone should have been filtered: %v", m["content"])
		}
	}
}

// ---------------------------------------------------------------------------
// Test 3: suggest-analyze shows suggestions matching inactive (expired) pheromones
// ---------------------------------------------------------------------------

func TestSuggestAnalyze_ShowsInactivePheromoneSuggestions(t *testing.T) {
	tmpDir, buf := setupSuggestAnalyzeTest(t)
	defer os.RemoveAll(tmpDir)

	// Create a .env file
	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("KEY=val\n"), 0644)

	// Pre-create an INACTIVE pheromone with the same content hash.
	content := "never commit secrets or .env files to version control"
	hash := "sha256:" + sha256Sum(content)
	sig := colony.PheromoneSignal{
		ID:          "sig_test_2",
		Type:        "REDIRECT",
		Content:     json.RawMessage(`{"text":"never commit secrets or .env files to version control"}`),
		Active:      false,
		ContentHash: &hash,
	}
	setupPheromones(t, []colony.PheromoneSignal{sig})

	rootCmd.SetArgs([]string{"suggest-analyze", "--target", tmpDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	suggestions := result["suggestions"].([]interface{})

	// The .env suggestion should NOT be filtered because the pheromone is inactive.
	found := false
	for _, s := range suggestions {
		m := s.(map[string]interface{})
		if m["content"] == content {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected suggestion matching inactive pheromone to be present")
	}
}

// ---------------------------------------------------------------------------
// Test 4: suggest-analyze persists pending suggestions to COLONY_STATE.json
// ---------------------------------------------------------------------------

func TestSuggestAnalyze_PersistsSuggestions(t *testing.T) {
	tmpDir, buf := setupSuggestAnalyzeTest(t)
	defer os.RemoveAll(tmpDir)

	// Create a .env file
	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("KEY=val\n"), 0644)

	rootCmd.SetArgs([]string{"suggest-analyze", "--target", tmpDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	newCount := result["new_count"].(float64)
	if newCount == 0 {
		t.Fatal("expected new_count > 0 after first analysis")
	}

	// Reload colony state and verify pending_suggestions was persisted.
	var reloaded colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &reloaded); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if reloaded.PendingSuggestions == nil || len(*reloaded.PendingSuggestions) == 0 {
		t.Error("expected pending_suggestions to be persisted in COLONY_STATE.json")
	}
}

// ---------------------------------------------------------------------------
// Test 5: suggest-analyze returns ok:true with empty suggestions on error
// ---------------------------------------------------------------------------

func TestSuggestAnalyze_NonBlockingOnError(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	// No store set -- triggers the nil guard path
	store = nil

	rootCmd.SetArgs([]string{"suggest-analyze", "--target", "."})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true even on error, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	suggestions := result["suggestions"].([]interface{})
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions on error, got %d", len(suggestions))
	}
}

// ---------------------------------------------------------------------------
// Test 6: suggest-analyze runs build-specific extra patterns
// ---------------------------------------------------------------------------

func TestSuggestAnalyze_BuildSpecificPatterns(t *testing.T) {
	tmpDir, buf := setupSuggestAnalyzeTest(t)
	defer os.RemoveAll(tmpDir)

	// Create files with high TODO/FIXME density
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)
	todoContent := strings.Repeat("// TODO: fix this\n// TODO: fix that\n// FIXME: broken\n", 4)
	_ = os.WriteFile(filepath.Join(srcDir, "main.go"), []byte(todoContent), 0644)

	rootCmd.SetArgs([]string{"suggest-analyze", "--target", tmpDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	suggestions := result["suggestions"].([]interface{})

	// At least one suggestion should mention TODO/FIXME density
	found := false
	for _, s := range suggestions {
		m := s.(map[string]interface{})
		content, _ := m["content"].(string)
		if strings.Contains(content, "TODO") || strings.Contains(content, "FIXME") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected build-specific pattern to detect TODO/FIXME density")
	}
}

// ---------------------------------------------------------------------------
// Test 7: suggest-analyze respects --dry-run flag
// ---------------------------------------------------------------------------

func TestSuggestAnalyze_DryRun(t *testing.T) {
	tmpDir, buf := setupSuggestAnalyzeTest(t)
	defer os.RemoveAll(tmpDir)

	// Create a .env file
	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("KEY=val\n"), 0644)

	rootCmd.SetArgs([]string{"suggest-analyze", "--target", tmpDir, "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	if result["dry_run"] != true {
		t.Errorf("expected dry_run=true, got %v", result["dry_run"])
	}

	// Verify suggestions were NOT persisted (dry-run skips persistence).
	var reloaded colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &reloaded); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if reloaded.PendingSuggestions != nil && len(*reloaded.PendingSuggestions) > 0 {
		t.Error("expected no pending_suggestions in dry-run mode")
	}
}

// ---------------------------------------------------------------------------
// Test 8: suggest-analyze skips analysis when changes below threshold
// ---------------------------------------------------------------------------

func TestSuggestAnalyze_SkipBelowThreshold(t *testing.T) {
	tmpDir, buf := setupSuggestAnalyzeTest(t)
	defer os.RemoveAll(tmpDir)

	// Initialize git repo in tmpDir with a commit so HEAD is valid.
	initTestGitRepo(t, tmpDir)
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial")

	headCommit := execGitRevParse(t, tmpDir)

	// Set LastAnalyzeCommit so the change detection path runs.
	goal := "test goal"
	cs := colony.ColonyState{
		Version:           "1.0",
		Goal:              &goal,
		State:             colony.StateREADY,
		LastAnalyzeCommit: &headCommit,
	}
	data, _ := json.Marshal(cs)
	_ = store.AtomicWrite("COLONY_STATE.json", data)

	// Create a .env file (but since HEAD == HEAD, diff will be empty = below threshold)
	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("KEY=val\n"), 0644)

	rootCmd.SetArgs([]string{"suggest-analyze", "--target", tmpDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	suggestions := result["suggestions"].([]interface{})

	// Should return existing pending suggestions (empty since we haven't persisted any),
	// NOT new ones from pattern detection.
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions when change count below threshold, got %d", len(suggestions))
	}
}

// ---------------------------------------------------------------------------
// Test 9: suggest-analyze re-runs analysis when changes exceed threshold
// ---------------------------------------------------------------------------

func TestSuggestAnalyze_ReRunAboveThreshold(t *testing.T) {
	tmpDir, buf := setupSuggestAnalyzeTest(t)
	defer os.RemoveAll(tmpDir)

	// Initialize git repo in tmpDir so we can create commits.
	initTestGitRepo(t, tmpDir)

	// Create initial commit
	_ = os.WriteFile(filepath.Join(tmpDir, "initial.go"), []byte("package main\n"), 0644)
	runGit(t, tmpDir, "add", "initial.go")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Get the initial commit hash as our "last analyze" point.
	oldCommit := execGitRevParse(t, tmpDir)

	// Set LastAnalyzeCommit to the old commit.
	goal := "test goal"
	cs := colony.ColonyState{
		Version:           "1.0",
		Goal:              &goal,
		State:             colony.StateREADY,
		LastAnalyzeCommit: &oldCommit,
	}
	data, _ := json.Marshal(cs)
	_ = store.AtomicWrite("COLONY_STATE.json", data)

	// Create 6 new files (above threshold of 5) and commit them.
	for i := 0; i < 6; i++ {
		name := filepath.Join(tmpDir, "changed.go")
		_ = os.WriteFile(name, []byte("package main\n"), 0644)
		runGit(t, tmpDir, "add", "changed.go")
	}
	runGit(t, tmpDir, "commit", "-m", "many changes")

	// Create a .env file for pattern detection.
	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("KEY=val\n"), 0644)

	rootCmd.SetArgs([]string{"suggest-analyze", "--target", tmpDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	suggestions := result["suggestions"].([]interface{})

	// Should have re-run analysis and found the .env pattern.
	if len(suggestions) == 0 {
		t.Error("expected suggestions when change count exceeds threshold")
	}
}
