package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

// setupContextUpdateTest creates a temp dir and configures COLONY_DATA_DIR
// so PersistentPreRunE initializes store to the temp dir. It also redirects
// stdout/stderr for output capture.
func setupContextUpdateTest(t *testing.T) (string, string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}

	os.Setenv("COLONY_DATA_DIR", dataDir)
	t.Cleanup(func() { os.Unsetenv("COLONY_DATA_DIR") })

	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	return tmpDir, dataDir
}

// getStore returns the package-level store (set by PersistentPreRunE during Execute).
func getStore(t *testing.T) *storage.Store {
	t.Helper()
	if store == nil {
		t.Fatal("store is nil after Execute -- PersistentPreRunE did not initialize it")
	}
	return store
}

// parseResult parses the JSON envelope from stdout and returns the result map.
func parseResult(t *testing.T, out string) map[string]interface{} {
	t.Helper()
	var envelope struct {
		OK     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envelope); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %q", err, out)
	}
	if !envelope.OK {
		t.Fatalf("expected ok=true, got envelope: %s", out)
	}
	return envelope.Result
}

// writeContextFile writes a CONTEXT.md to the store's base path.
func writeContextFile(t *testing.T, s *storage.Store, content string) {
	t.Helper()
	if err := s.AtomicWrite("CONTEXT.md", []byte(content)); err != nil {
		t.Fatalf("failed to write CONTEXT.md: %v", err)
	}
}

// readContextFile reads CONTEXT.md from the store.
func readContextFile(t *testing.T, s *storage.Store) string {
	t.Helper()
	data, err := s.ReadFile("CONTEXT.md")
	if err != nil {
		t.Fatalf("failed to read CONTEXT.md: %v", err)
	}
	return string(data)
}

// executeContextCmd runs rootCmd.Execute and returns stdout string.
func executeContextCmd(t *testing.T) string {
	t.Helper()
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	return stdout.(*bytes.Buffer).String()
}

// --- init sub-action ---

func TestContextUpdateInit(t *testing.T) {
	_, _ = setupContextUpdateTest(t)

	rootCmd.SetArgs([]string{"context-update", "init", "Build feature X"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["action"] != "init" {
		t.Errorf("action = %v, want \"init\"", result["action"])
	}
	if result["updated"] != true {
		t.Errorf("updated = %v, want true", result["updated"])
	}

	content := readContextFile(t, s)
	if !strings.Contains(content, "Build feature X") {
		t.Error("CONTEXT.md should contain the goal text")
	}
	if !strings.Contains(content, "What's In Progress") {
		t.Error("CONTEXT.md should contain What's In Progress section")
	}
	if !strings.Contains(content, "Safe to Clear?") {
		t.Error("CONTEXT.md should contain Safe to Clear? row")
	}
}

// --- build-start sub-action ---

func TestContextUpdateBuildStart(t *testing.T) {
	_, dataDir := setupContextUpdateTest(t)

	// Pre-write CONTEXT.md via direct file I/O (before Execute initializes store)
	initContent := `# Aether Colony — Current Context

---

## System Status

| Field | Value |
|-------|-------|
| **Last Updated** | old-timestamp |
| **Safe to Clear?** | YES |

---

## What's In Progress

Nothing yet.

---

## Next Steps
`
	os.WriteFile(filepath.Join(dataDir, "CONTEXT.md"), []byte(initContent), 0644)

	rootCmd.SetArgs([]string{"context-update", "build-start", "3", "4", "10"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["action"] != "build-start" {
		t.Errorf("action = %v, want \"build-start\"", result["action"])
	}
	if result["workers"] != "4" {
		t.Errorf("workers = %v, want 4", result["workers"])
	}

	content := readContextFile(t, s)
	if !strings.Contains(content, "Phase 3 Build IN PROGRESS") {
		t.Error("CONTEXT.md should contain 'Phase 3 Build IN PROGRESS'")
	}
	if !strings.Contains(content, "Workers: 4") {
		t.Error("CONTEXT.md should contain 'Workers: 4'")
	}
	if !strings.Contains(content, "NO — Build in progress") {
		t.Error("CONTEXT.md should have Safe to Clear = NO — Build in progress")
	}
}

// --- build-progress sub-action ---

func TestContextUpdateBuildProgress(t *testing.T) {
	_, dataDir := setupContextUpdateTest(t)

	content := `## What's In Progress

**Phase 2 Build IN PROGRESS**
- Workers: 3 | Tasks: 5
`
	os.WriteFile(filepath.Join(dataDir, "CONTEXT.md"), []byte(content), 0644)

	rootCmd.SetArgs([]string{"context-update", "build-progress", "3", "5"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["action"] != "build-progress" {
		t.Errorf("action = %v, want \"build-progress\"", result["action"])
	}
	if result["percent"] != float64(60) {
		t.Errorf("percent = %v, want 60", result["percent"])
	}

	updated := readContextFile(t, s)
	if !strings.Contains(updated, "60% complete") {
		t.Error("CONTEXT.md should contain '60% complete'")
	}
}

// --- build-complete sub-action ---

func TestContextUpdateBuildComplete(t *testing.T) {
	_, dataDir := setupContextUpdateTest(t)

	content := `# Aether Colony

| **Last Updated** | old-ts |
| **Safe to Clear?** | NO — Build in progress |

---

## What's In Progress

**Phase 1 Build IN PROGRESS**
- Workers: 3 | Tasks: 5
- Started: 2026-04-01T00:00:00Z

---

## Next Steps
`
	os.WriteFile(filepath.Join(dataDir, "CONTEXT.md"), []byte(content), 0644)

	rootCmd.SetArgs([]string{"context-update", "build-complete", "completed", "success"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["action"] != "build-complete" {
		t.Errorf("action = %v, want \"build-complete\"", result["action"])
	}
	if result["status"] != "completed" {
		t.Errorf("status = %v, want \"completed\"", result["status"])
	}

	updated := readContextFile(t, s)
	if !strings.Contains(updated, "Build completed") {
		t.Error("CONTEXT.md should contain 'Build completed'")
	}
	if !strings.Contains(updated, "YES — Build completed") {
		t.Error("CONTEXT.md should have Safe to Clear = YES — Build completed")
	}
	if strings.Contains(updated, "IN PROGRESS") {
		t.Error("CONTEXT.md should NOT contain 'IN PROGRESS' after build-complete")
	}
}

// --- worker-spawn sub-action ---

func TestContextUpdateWorkerSpawn(t *testing.T) {
	_, dataDir := setupContextUpdateTest(t)

	content := `## What's In Progress

**Phase 2 Build IN PROGRESS**
- Workers: 3 | Tasks: 5
- Started: 2026-04-01T00:00:00Z
`
	os.WriteFile(filepath.Join(dataDir, "CONTEXT.md"), []byte(content), 0644)

	rootCmd.SetArgs([]string{"context-update", "worker-spawn", "Branthos", "builder", "implement feature X"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["action"] != "worker-spawn" {
		t.Errorf("action = %v, want \"worker-spawn\"", result["action"])
	}
	if result["ant"] != "Branthos" {
		t.Errorf("ant = %v, want \"Branthos\"", result["ant"])
	}

	updated := readContextFile(t, s)
	if !strings.Contains(updated, "Spawned Branthos") {
		t.Error("CONTEXT.md should contain 'Spawned Branthos'")
	}
	if !strings.Contains(updated, "builder") {
		t.Error("CONTEXT.md should contain 'builder'")
	}
	if !strings.Contains(updated, "implement feature X") {
		t.Error("CONTEXT.md should contain the task description")
	}
}

// --- worker-complete sub-action ---

func TestContextUpdateWorkerComplete(t *testing.T) {
	_, dataDir := setupContextUpdateTest(t)

	content := `## What's In Progress

**Phase 2 Build IN PROGRESS**
- Workers: 3 | Tasks: 5
- Started: 2026-04-01T00:00:00Z
  - 2026-04-01T00:01:00Z: Spawned Branthos (builder) for: implement feature X
  - 2026-04-01T00:01:01Z: Spawned Watcher (watcher) for: verify tests
`
	os.WriteFile(filepath.Join(dataDir, "CONTEXT.md"), []byte(content), 0644)

	rootCmd.SetArgs([]string{"context-update", "worker-complete", "Branthos", "completed"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["action"] != "worker-complete" {
		t.Errorf("action = %v, want \"worker-complete\"", result["action"])
	}
	if result["ant"] != "Branthos" {
		t.Errorf("ant = %v, want \"Branthos\"", result["ant"])
	}

	updated := readContextFile(t, s)
	if !strings.Contains(updated, "Branthos: completed") {
		t.Error("CONTEXT.md should contain 'Branthos: completed'")
	}
	if !strings.Contains(updated, "Spawned Watcher") {
		t.Error("Watcher spawn line should remain unchanged")
	}
}

// --- unknown sub-action ---

func TestContextUpdateUnknownAction(t *testing.T) {
	_, _ = setupContextUpdateTest(t)

	rootCmd.SetArgs([]string{"context-update", "unknown-action"})

	// Should NOT return a cobra error -- we handle it ourselves
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no cobra error: %v", err)
	}

	errOutput := stderr.(*bytes.Buffer).String()
	if !strings.Contains(errOutput, `"ok":false`) {
		t.Errorf("expected error JSON on stderr, got: %q", errOutput)
	}
}

// --- missing CONTEXT.md for non-init actions ---

func TestContextUpdateBuildStartNoContextFile(t *testing.T) {
	_, _ = setupContextUpdateTest(t)

	rootCmd.SetArgs([]string{"context-update", "build-start", "1", "2", "3"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no cobra error: %v", err)
	}

	errOutput := stderr.(*bytes.Buffer).String()
	if !strings.Contains(errOutput, `"ok":false`) {
		t.Errorf("expected error when CONTEXT.md missing, got: %q", errOutput)
	}
}

// --- backward compatibility: --summary flag still works ---

func TestContextUpdateSummaryFlag(t *testing.T) {
	_, _ = setupContextUpdateTest(t)

	rootCmd.SetArgs([]string{"context-update", "--summary", "test summary"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["updated"] != true {
		t.Errorf("updated = %v, want true", result["updated"])
	}

	data, err := s.ReadFile("rolling-summary.log")
	if err != nil {
		t.Fatalf("failed to read rolling-summary.log: %v", err)
	}
	if !strings.Contains(string(data), "test summary") {
		t.Error("rolling-summary.log should contain 'test summary'")
	}
}

// --- positional arg takes priority over --summary ---

func TestContextUpdatePositionalOverridesSummary(t *testing.T) {
	_, dataDir := setupContextUpdateTest(t)

	initContent := `# Aether Colony

| **Last Updated** | old-ts |
| **Safe to Clear?** | YES |

---

## What's In Progress

Nothing yet.
`
	os.WriteFile(filepath.Join(dataDir, "CONTEXT.md"), []byte(initContent), 0644)

	// Both positional arg and --summary provided; positional should win
	rootCmd.SetArgs([]string{"context-update", "init", "goal text", "--summary", "ignored"})
	out := executeContextCmd(t)

	result := parseResult(t, out)
	if result["action"] != "init" {
		t.Errorf("action = %v, want \"init\" (positional should take priority)", result["action"])
	}
}

// --- activity sub-action ---

func TestContextUpdateActivity(t *testing.T) {
	_, dataDir := setupContextUpdateTest(t)

	content := `# Aether Colony

| **Last Updated** | old-ts |
| **Safe to Clear?** | YES |

---

## Recent Activity (Last 10 Actions)

| Timestamp | Command | Result | Files Changed |
|-----------|---------|--------|---------------|

---

## Next Steps
`
	os.WriteFile(filepath.Join(dataDir, "CONTEXT.md"), []byte(content), 0644)

	rootCmd.SetArgs([]string{"context-update", "activity", "build 3", "completed", "5"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["action"] != "activity" {
		t.Errorf("action = %v, want \"activity\"", result["action"])
	}

	updated := readContextFile(t, s)
	if !strings.Contains(updated, "| build 3 |") {
		t.Error("CONTEXT.md should contain 'build 3' in activity table")
	}
	if !strings.Contains(updated, "| completed |") {
		t.Error("CONTEXT.md should contain 'completed' in activity table")
	}
	if !strings.Contains(updated, "| 5 |") {
		t.Error("CONTEXT.md should contain '5' in Files Changed column")
	}
}

// --- update-phase sub-action ---

func TestContextUpdatePhase(t *testing.T) {
	_, dataDir := setupContextUpdateTest(t)

	content := `# Aether Colony

| Field | Value |
|-------|-------|
| **Last Updated** | old-ts |
| **Current Phase** | 1 |
| **Phase Name** | initialization |
| **Safe to Clear?** | NO — Build in progress |

---

## Next Steps
`
	os.WriteFile(filepath.Join(dataDir, "CONTEXT.md"), []byte(content), 0644)

	rootCmd.SetArgs([]string{"context-update", "update-phase", "3", "feature work", "YES", "Phase advanced"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["action"] != "update-phase" {
		t.Errorf("action = %v, want \"update-phase\"", result["action"])
	}

	updated := readContextFile(t, s)
	if !strings.Contains(updated, "| **Current Phase** | 3 |") {
		t.Error("CONTEXT.md should have Current Phase = 3")
	}
	if !strings.Contains(updated, "| **Phase Name** | feature work |") {
		t.Error("CONTEXT.md should have Phase Name = feature work")
	}
	if !strings.Contains(updated, "YES") {
		t.Error("CONTEXT.md should have Safe to Clear = YES")
	}
}

// --- decision sub-action ---

func TestContextUpdateDecision(t *testing.T) {
	_, dataDir := setupContextUpdateTest(t)

	content := `# Aether Colony

---

## Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|

---

## Next Steps
`
	os.WriteFile(filepath.Join(dataDir, "CONTEXT.md"), []byte(content), 0644)

	rootCmd.SetArgs([]string{"context-update", "decision", "Use React for frontend", "Better component model", "Queen"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["action"] != "decision" {
		t.Errorf("action = %v, want \"decision\"", result["action"])
	}

	updated := readContextFile(t, s)
	if !strings.Contains(updated, "| Use React for frontend |") {
		t.Error("CONTEXT.md should contain decision text")
	}
	if !strings.Contains(updated, "| Better component model |") {
		t.Error("CONTEXT.md should contain rationale")
	}
	if !strings.Contains(updated, "| Queen |") {
		t.Error("CONTEXT.md should contain 'Queen' as Made By")
	}
}

// --- safe-to-clear sub-action ---

func TestContextUpdateSafeToClear(t *testing.T) {
	_, dataDir := setupContextUpdateTest(t)

	content := `# Aether Colony

| Field | Value |
|-------|-------|
| **Last Updated** | old-ts |
| **Safe to Clear?** | NO — Build in progress |

---

## Next Steps
`
	os.WriteFile(filepath.Join(dataDir, "CONTEXT.md"), []byte(content), 0644)

	rootCmd.SetArgs([]string{"context-update", "safe-to-clear", "YES", "Build complete, ready to continue"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["action"] != "safe-to-clear" {
		t.Errorf("action = %v, want \"safe-to-clear\"", result["action"])
	}

	updated := readContextFile(t, s)
	if !strings.Contains(updated, "| **Safe to Clear?** | YES — Build complete, ready to continue |") {
		t.Errorf("Safe to Clear? row not updated correctly, got: %s", updated)
	}
}

// --- --section/--key/--content flags ---

func TestContextUpdateSectionKeyContent(t *testing.T) {
	_, dataDir := setupContextUpdateTest(t)

	content := `# Aether Colony

---

## Active Constraints (REDIRECT Signals)

| Constraint | Source | Date Set |
|------------|--------|----------|
| Existing constraint | CLAUDE.md | Permanent |

---

## Active Pheromones (FOCUS Signals)

*None active*

---

## Next Steps
`
	os.WriteFile(filepath.Join(dataDir, "CONTEXT.md"), []byte(content), 0644)

	rootCmd.SetArgs([]string{"context-update", "--section", "constraint", "--key", "redirect", "--content", "avoid pattern X", "user"})
	out := executeContextCmd(t)
	s := getStore(t)

	result := parseResult(t, out)
	if result["updated"] != true {
		t.Errorf("updated = %v, want true", result["updated"])
	}

	updated := readContextFile(t, s)
	if !strings.Contains(updated, "avoid pattern X") {
		t.Error("CONTEXT.md should contain the constraint content")
	}
}
