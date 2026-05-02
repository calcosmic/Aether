package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHeartbeatFileMarshalUnmarshal(t *testing.T) {
	hf := HeartbeatFile{
		WorkerID:  "Hammer-23",
		Caste:     "builder",
		Timestamp: "2026-05-02T14:30:00Z",
		Phase:     1,
	}

	data, err := json.Marshal(hf)
	if err != nil {
		t.Fatalf("marshal heartbeat file: %v", err)
	}

	var parsed HeartbeatFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal heartbeat file: %v", err)
	}

	if parsed.WorkerID != hf.WorkerID {
		t.Errorf("WorkerID = %q, want %q", parsed.WorkerID, hf.WorkerID)
	}
	if parsed.Caste != hf.Caste {
		t.Errorf("Caste = %q, want %q", parsed.Caste, hf.Caste)
	}
	if parsed.Timestamp != hf.Timestamp {
		t.Errorf("Timestamp = %q, want %q", parsed.Timestamp, hf.Timestamp)
	}
	if parsed.Phase != hf.Phase {
		t.Errorf("Phase = %d, want %d", parsed.Phase, hf.Phase)
	}
}

func TestHeartbeatScanValidNotStale(t *testing.T) {
	saveGlobals(t)
	t.Setenv("AETHER_FORCE_VISUAL", "1")

	tmpDir := t.TempDir()
	hf := HeartbeatFile{
		WorkerID:  "Hammer-23",
		Caste:     "builder",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Phase:     1,
	}
	writeHeartbeatFile(t, tmpDir, "heartbeat-Hammer-23.json", hf)

	var buf strings.Builder
	stdout = &buf
	scanHeartbeatFiles(tmpDir)

	output := buf.String()
	if strings.Contains(output, "stale") {
		t.Errorf("expected no stale warning for fresh heartbeat, got: %s", output)
	}
}

func TestHeartbeatScanDetectsStale(t *testing.T) {
	saveGlobals(t)
	t.Setenv("AETHER_FORCE_VISUAL", "1")

	tmpDir := t.TempDir()
	// Create a heartbeat file with a timestamp 2 minutes old (past the 90s threshold)
	staleTime := time.Now().UTC().Add(-2 * time.Minute)
	hf := HeartbeatFile{
		WorkerID:  "Mason-67",
		Caste:     "builder",
		Timestamp: staleTime.Format(time.RFC3339),
		Phase:     2,
	}
	writeHeartbeatFile(t, tmpDir, "heartbeat-Mason-67.json", hf)

	var buf strings.Builder
	stdout = &buf
	scanHeartbeatFiles(tmpDir)

	output := buf.String()
	if !strings.Contains(output, "stale") {
		t.Errorf("expected stale warning for old heartbeat, got: %s", output)
	}
	if !strings.Contains(output, "Mason-67") {
		t.Errorf("expected worker ID in stale warning, got: %s", output)
	}
}

func TestHeartbeatScanSkipsNonHeartbeatFiles(t *testing.T) {
	saveGlobals(t)
	t.Setenv("AETHER_FORCE_VISUAL", "1")

	tmpDir := t.TempDir()
	// Create a non-heartbeat file
	if err := os.WriteFile(filepath.Join(tmpDir, "other-data.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("write non-heartbeat file: %v", err)
	}
	// Create a valid non-stale heartbeat to confirm scanning still works
	hf := HeartbeatFile{
		WorkerID:  "Hammer-23",
		Caste:     "builder",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Phase:     1,
	}
	writeHeartbeatFile(t, tmpDir, "heartbeat-Hammer-23.json", hf)

	var buf strings.Builder
	stdout = &buf
	scanHeartbeatFiles(tmpDir)

	output := buf.String()
	// Should not contain any stale warning (the valid file is fresh)
	if strings.Contains(output, "stale") {
		t.Errorf("expected no stale warning, got: %s", output)
	}
}

func TestHeartbeatScanSkipsMalformedJSON(t *testing.T) {
	saveGlobals(t)
	t.Setenv("AETHER_FORCE_VISUAL", "1")

	tmpDir := t.TempDir()
	// Create a malformed heartbeat file
	if err := os.WriteFile(filepath.Join(tmpDir, "heartbeat-BadWorker.json"), []byte("not valid json{{{"), 0644); err != nil {
		t.Fatalf("write malformed file: %v", err)
	}

	var buf strings.Builder
	stdout = &buf
	scanHeartbeatFiles(tmpDir)

	output := buf.String()
	// Should not crash or produce errors, just skip silently
	if strings.Contains(output, "error") {
		t.Errorf("expected silent skip of malformed JSON, got: %s", output)
	}
}

func TestHeartbeatCleanupByWorkerID(t *testing.T) {
	tmpDir := t.TempDir()

	hf := HeartbeatFile{
		WorkerID:  "Hammer-23",
		Caste:     "builder",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Phase:     1,
	}
	writeHeartbeatFile(t, tmpDir, "heartbeat-Hammer-23.json", hf)

	// Also create a different worker's heartbeat that should NOT be cleaned
	hf2 := HeartbeatFile{
		WorkerID:  "Mason-67",
		Caste:     "builder",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Phase:     1,
	}
	writeHeartbeatFile(t, tmpDir, "heartbeat-Mason-67.json", hf2)

	err := cleanupHeartbeatFiles(tmpDir, "Hammer-23")
	if err != nil {
		t.Fatalf("cleanup heartbeat files: %v", err)
	}

	// Hammer-23 should be gone
	if _, err := os.Stat(filepath.Join(tmpDir, "heartbeat-Hammer-23.json")); !os.IsNotExist(err) {
		t.Error("expected Hammer-23 heartbeat file to be removed")
	}
	// Mason-67 should still exist
	if _, err := os.Stat(filepath.Join(tmpDir, "heartbeat-Mason-67.json")); err != nil {
		t.Errorf("expected Mason-67 heartbeat file to still exist: %v", err)
	}
}

func TestHeartbeatCleanupNonexistentIsFine(t *testing.T) {
	tmpDir := t.TempDir()

	err := cleanupHeartbeatFiles(tmpDir, "Nonexistent-99")
	if err != nil {
		t.Errorf("cleanup nonexistent worker should not error, got: %v", err)
	}
}

func TestHeartbeatMonitorStopsOnCancel(t *testing.T) {
	tmpDir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)

	// Start the monitor - it should be running
	stop := StartHeartbeatMonitor(ctx, tmpDir)

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
		wg.Done()
	}()

	// Give it time to process cancellation
	time.Sleep(200 * time.Millisecond)

	// Call stop (the cancel func) - should not panic
	stop()

	wg.Wait()
}

func TestHeartbeatCleanupAllFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple heartbeat files
	for _, name := range []string{"Hammer-23", "Mason-67", "Drill-42"} {
		hf := HeartbeatFile{
			WorkerID:  name,
			Caste:     "builder",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Phase:     1,
		}
		writeHeartbeatFile(t, tmpDir, "heartbeat-"+name+".json", hf)
	}

	// Also create a non-heartbeat file that should NOT be removed
	if err := os.WriteFile(filepath.Join(tmpDir, "other-data.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("write non-heartbeat file: %v", err)
	}

	cleanupAllHeartbeatFiles(tmpDir)

	// All heartbeat files should be gone
	matches, _ := filepath.Glob(filepath.Join(tmpDir, "heartbeat-*.json"))
	if len(matches) > 0 {
		t.Errorf("expected all heartbeat files removed, found: %v", matches)
	}

	// Non-heartbeat file should remain
	if _, err := os.Stat(filepath.Join(tmpDir, "other-data.json")); err != nil {
		t.Errorf("expected non-heartbeat file to remain: %v", err)
	}
}

func writeHeartbeatFile(t *testing.T, dir, name string, hf HeartbeatFile) {
	t.Helper()
	data, err := json.MarshalIndent(hf, "", "  ")
	if err != nil {
		t.Fatalf("marshal heartbeat: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
		t.Fatalf("write heartbeat file: %v", err)
	}
}
