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

// newTestStoreWithRoot creates a fresh store in a temp dir for testing.
// It sets both COLONY_DATA_DIR and AETHER_ROOT so all path resolution
// (including ResolveAetherRoot()) uses the temp directory.
func newTestStoreWithRoot(t *testing.T) (*storage.Store, string) {
	t.Helper()
	origColonyDataDir := os.Getenv("COLONY_DATA_DIR")
	origAetherRoot := os.Getenv("AETHER_ROOT")
	t.Cleanup(func() {
		os.Setenv("COLONY_DATA_DIR", origColonyDataDir)
		os.Setenv("AETHER_ROOT", origAetherRoot)
	})
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	os.Setenv("COLONY_DATA_DIR", dataDir)
	os.Setenv("AETHER_ROOT", tmpDir)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return s, tmpDir
}

// --- Chamber Tests ---

func TestChamberCreate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"chamber-create", "--name", "test-chamber", "--goal", "build stuff", "--milestone", "Brood Stable"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["created"] != true {
		t.Errorf("created = %v, want true", result["created"])
	}
	if result["name"] != "test-chamber" {
		t.Errorf("name = %v, want test-chamber", result["name"])
	}

	// Verify the directory and manifest.json exist
	chambersDir := filepath.Join(tmpDir, ".aether", "chambers", "test-chamber")
	info, err := os.Stat(chambersDir)
	if err != nil || !info.IsDir() {
		t.Errorf("chamber directory not created at %s", chambersDir)
	}
	manifestPath := filepath.Join(chambersDir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("manifest.json not created at %s", manifestPath)
	}
}

func TestChamberCreateWithPhases(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"chamber-create", "--name", "phase-chamber", "--phases-completed", "5", "--total-phases", "8"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["name"] != "phase-chamber" {
		t.Errorf("name = %v, want phase-chamber", result["name"])
	}

	// Verify manifest has phase info
	manifestPath := filepath.Join(tmpDir, ".aether", "chambers", "phase-chamber", "manifest.json")
	data, _ := os.ReadFile(manifestPath)
	var manifest map[string]interface{}
	json.Unmarshal(data, &manifest)
	if manifest["phases_completed"] != float64(5) {
		t.Errorf("phases_completed = %v, want 5", manifest["phases_completed"])
	}
	if manifest["total_phases"] != float64(8) {
		t.Errorf("total_phases = %v, want 8", manifest["total_phases"])
	}
}

func TestChamberCreateMissingName(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Reset the --name flag to ensure clean state from previous tests
	chamberCreateCmd.Flags().Set("name", "")

	rootCmd.SetArgs([]string{"chamber-create"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing --name, got: %v", env["ok"])
	}
}

func TestChamberCreateNilStore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Reset the --name flag from previous tests
	chamberCreateCmd.Flags().Set("name", "")

	// After PersistentPreRunE sets store, force nil to test the guard.
	// We can't easily prevent PersistentPreRunE from running, so instead
	// we test the nil-store path by calling the RunE function directly.
	store = nil

	rootCmd.SetArgs([]string{"chamber-create", "--name", "test"})

	// Call RunE directly to bypass PersistentPreRunE which would set store
	err := chamberCreateCmd.RunE(chamberCreateCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for nil store, got: %v", env["ok"])
	}
}

func TestChamberVerify(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a chamber first
	chambersDir := filepath.Join(tmpDir, ".aether", "chambers", "verify-test")
	os.MkdirAll(chambersDir, 0755)
	os.WriteFile(filepath.Join(chambersDir, "manifest.json"), []byte(`{"name":"verify-test","goal":"test"}`), 0644)

	rootCmd.SetArgs([]string{"chamber-verify", "--name", "verify-test"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["name"] != "verify-test" {
		t.Errorf("name = %v, want verify-test", result["name"])
	}
	if result["valid"] != true {
		t.Errorf("valid = %v, want true", result["valid"])
	}
}

func TestChamberVerifyMissing(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"chamber-verify", "--name", "nonexistent"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing chamber, got: %v", env["ok"])
	}
}

func TestChamberListEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"chamber-list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["total"] != float64(0) {
		t.Errorf("total = %v, want 0 for empty chambers", result["total"])
	}
	chambers := result["chambers"].([]interface{})
	if len(chambers) != 0 {
		t.Errorf("chambers = %v, want empty", chambers)
	}
}

func TestChamberListWithEntries(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create two chambers
	chambersRoot := filepath.Join(tmpDir, ".aether", "chambers")

	ch1 := filepath.Join(chambersRoot, "chamber-alpha")
	os.MkdirAll(ch1, 0755)
	os.WriteFile(filepath.Join(ch1, "manifest.json"), []byte(`{"name":"chamber-alpha","milestone":"Crowned Anthill"}`), 0644)

	ch2 := filepath.Join(chambersRoot, "chamber-beta")
	os.MkdirAll(ch2, 0755)
	os.WriteFile(filepath.Join(ch2, "manifest.json"), []byte(`{"name":"chamber-beta","milestone":"Brood Stable"}`), 0644)

	rootCmd.SetArgs([]string{"chamber-list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["total"] != float64(2) {
		t.Errorf("total = %v, want 2", result["total"])
	}
}

// --- Maintenance Tests ---

func TestDataCleanDryRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "test_abc", Active: true, Content: json.RawMessage(`{"text":"test signal"}`)},
			{ID: "demo_xyz", Active: true, Content: json.RawMessage(`{"text":"demo pattern"}`)},
			{ID: "sig_real", Active: true, Content: json.RawMessage(`{"text":"real signal"}`)},
		},
	}
	s.SaveJSON("pheromones.json", pf)

	rootCmd.SetArgs([]string{"data-clean"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["scanned"] != true {
		t.Errorf("scanned = %v, want true", result["scanned"])
	}
	if result["removed"] != float64(0) {
		t.Errorf("removed = %v, want 0 in dry-run mode", result["removed"])
	}
	if result["dry_run"] != true {
		t.Errorf("dry_run = %v, want true by default", result["dry_run"])
	}
}

func TestDataCleanConfirm(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "test_abc", Active: true, Content: json.RawMessage(`{"text":"test signal"}`)},
			{ID: "demo_xyz", Active: true, Content: json.RawMessage(`{"text":"demo pattern"}`)},
			{ID: "sig_real", Active: true, Content: json.RawMessage(`{"text":"real signal"}`)},
		},
	}
	s.SaveJSON("pheromones.json", pf)

	rootCmd.SetArgs([]string{"data-clean", "--confirm"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["removed"] != float64(2) {
		t.Errorf("removed = %v, want 2 (test_ and demo_ entries)", result["removed"])
	}
	if result["dry_run"] != false {
		t.Errorf("dry_run = %v, want false with --confirm", result["dry_run"])
	}

	// Verify the real signal is still there
	var updated colony.PheromoneFile
	s.LoadJSON("pheromones.json", &updated)
	if len(updated.Signals) != 1 {
		t.Errorf("expected 1 signal remaining, got %d", len(updated.Signals))
	}
}

func TestBackupPruneGlobal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create backup dir with 5 files
	backupDir := filepath.Join(s.BasePath(), "backups")
	os.MkdirAll(backupDir, 0755)
	for i := 0; i < 5; i++ {
		os.WriteFile(filepath.Join(backupDir, "backup-"+strings.Repeat("0", i)+".json"), []byte("{}"), 0644)
	}

	rootCmd.SetArgs([]string{"backup-prune-global", "--cap", "3"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["pruned"] != float64(2) {
		t.Errorf("pruned = %v, want 2", result["pruned"])
	}
	if result["kept"] != float64(3) {
		t.Errorf("kept = %v, want 3", result["kept"])
	}
}

func TestBackupPruneGlobalNoDir(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"backup-prune-global"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["dir_exists"] != false {
		t.Errorf("dir_exists = %v, want false when no backup dir", result["dir_exists"])
	}
	if result["pruned"] != float64(0) {
		t.Errorf("pruned = %v, want 0", result["pruned"])
	}
}

func TestTempClean(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create temp dir with an old file and a recent file
	tempDir := filepath.Join(tmpDir, ".aether", "temp")
	os.MkdirAll(tempDir, 0755)

	// Old file (8 days ago)
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	os.WriteFile(filepath.Join(tempDir, "old-file.txt"), []byte("old"), 0644)
	os.Chtimes(filepath.Join(tempDir, "old-file.txt"), oldTime, oldTime)

	// Recent file
	os.WriteFile(filepath.Join(tempDir, "recent-file.txt"), []byte("recent"), 0644)

	rootCmd.SetArgs([]string{"temp-clean"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["cleaned"] != float64(1) {
		t.Errorf("cleaned = %v, want 1 (only the old file)", result["cleaned"])
	}

	// Verify recent file still exists
	if _, err := os.Stat(filepath.Join(tempDir, "recent-file.txt")); err != nil {
		t.Error("recent file should still exist after temp-clean")
	}
}

func TestTempCleanNoDir(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"temp-clean"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["dir_exists"] != false {
		t.Errorf("dir_exists = %v, want false when no temp dir", result["dir_exists"])
	}
	if result["cleaned"] != float64(0) {
		t.Errorf("cleaned = %v, want 0", result["cleaned"])
	}
}
