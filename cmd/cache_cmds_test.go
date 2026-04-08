package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// cache-clean tests
// ---------------------------------------------------------------------------

func TestCacheCleanRemovesAllCacheFiles(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	// Create some .cache_* files
	os.WriteFile(dataDir+"/.cache_COLONY_STATE.json", []byte("{}"), 0644)
	os.WriteFile(dataDir+"/.cache_pheromones.json", []byte("{}"), 0644)
	// Also a non-cache file that should be preserved
	os.WriteFile(dataDir+"/pheromones.json", []byte("{}"), 0644)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"cache-clean"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("cache-clean returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	// Parse the JSON to check files_removed
	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			FilesRemoved int `json:"files_removed"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	if envelope.Result.FilesRemoved != 2 {
		t.Errorf("files_removed = %d, want 2", envelope.Result.FilesRemoved)
	}

	// Verify .cache_* files are gone
	if _, err := os.Stat(dataDir + "/.cache_COLONY_STATE.json"); err == nil {
		t.Error(".cache_COLONY_STATE.json should be removed")
	}
	if _, err := os.Stat(dataDir + "/.cache_pheromones.json"); err == nil {
		t.Error(".cache_pheromones.json should be removed")
	}
	// Non-cache file should still exist
	if _, err := os.Stat(dataDir + "/pheromones.json"); err != nil {
		t.Error("pheromones.json should be preserved")
	}
}

func TestCacheCleanEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"cache-clean"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("cache-clean returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			FilesRemoved int `json:"files_removed"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	if envelope.Result.FilesRemoved != 0 {
		t.Errorf("files_removed = %d, want 0", envelope.Result.FilesRemoved)
	}
}

// ---------------------------------------------------------------------------
// cache-clean-stale tests
// ---------------------------------------------------------------------------

func TestCacheCleanStaleRemovesOldFiles(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	// Create a stale file (48 hours old) and a fresh file
	staleFile := dataDir + "/.cache_stale.json"
	freshFile := dataDir + "/.cache_fresh.json"
	os.WriteFile(staleFile, []byte("{}"), 0644)
	os.WriteFile(freshFile, []byte("{}"), 0644)

	// Backdate the stale file
	oldTime := time.Now().Add(-48 * time.Hour)
	os.Chtimes(staleFile, oldTime, oldTime)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"cache-clean-stale"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("cache-clean-stale returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			FilesRemoved int `json:"files_removed"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	if envelope.Result.FilesRemoved != 1 {
		t.Errorf("files_removed = %d, want 1", envelope.Result.FilesRemoved)
	}

	// Stale file should be gone
	if _, err := os.Stat(staleFile); err == nil {
		t.Error("stale .cache_stale.json should be removed")
	}
	// Fresh file should remain
	if _, err := os.Stat(freshFile); err != nil {
		t.Error("fresh .cache_fresh.json should be preserved")
	}
}

func TestCacheCleanStaleCustomMaxAge(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	// Create a file that is 30 minutes old
	recentFile := dataDir + "/.cache_recent.json"
	os.WriteFile(recentFile, []byte("{}"), 0644)
	recentTime := time.Now().Add(-30 * time.Minute)
	os.Chtimes(recentFile, recentTime, recentTime)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	// Use a 1-hour max-age -- the 30-minute-old file should survive
	rootCmd.SetArgs([]string{"cache-clean-stale", "--max-age", "1h"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("cache-clean-stale returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			FilesRemoved int `json:"files_removed"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	if envelope.Result.FilesRemoved != 0 {
		t.Errorf("files_removed = %d, want 0 (file is newer than 1h)", envelope.Result.FilesRemoved)
	}

	// File should still exist (younger than max-age)
	if _, err := os.Stat(recentFile); err != nil {
		t.Error(".cache_recent.json should be preserved (only 30 min old, max-age 1h)")
	}
}

func TestCacheCleanStaleEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"cache-clean-stale"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("cache-clean-stale returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			FilesRemoved int `json:"files_removed"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	if envelope.Result.FilesRemoved != 0 {
		t.Errorf("files_removed = %d, want 0", envelope.Result.FilesRemoved)
	}
}

// ---------------------------------------------------------------------------
// command registration tests
// ---------------------------------------------------------------------------

func TestCacheCommandsRegistered(t *testing.T) {
	commands := []string{"cache-clean", "cache-clean-stale"}
	for _, name := range commands {
		cmd, _, err := rootCmd.Find([]string{name})
		if err != nil {
			t.Errorf("command %q not registered: %v", name, err)
			continue
		}
		if !strings.HasPrefix(cmd.Use, name) {
			t.Errorf("found command Use = %q, want prefix %q", cmd.Use, name)
		}
	}
}
