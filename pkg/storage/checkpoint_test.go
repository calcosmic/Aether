package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAutoCheckpoint_CreatesFile verifies that AutoCheckpoint creates a file
// in the checkpoints/ directory with timestamp-based naming.
func TestAutoCheckpoint_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	beforeState := []byte(`{"version":"1.0","state":"READY"}`)
	err = AutoCheckpoint(s, beforeState)
	if err != nil {
		t.Fatalf("AutoCheckpoint: %v", err)
	}

	checkpointsDir := filepath.Join(dir, "checkpoints")
	entries, err := os.ReadDir(checkpointsDir)
	if err != nil {
		t.Fatalf("read checkpoints dir: %v", err)
	}

	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "auto-") && strings.HasSuffix(e.Name(), ".json") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no auto-* checkpoint file found in checkpoints/")
	}
}

// TestAutoCheckpoint_ExactContent verifies that the checkpoint file contains
// the before-state bytes exactly.
func TestAutoCheckpoint_ExactContent(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	beforeState := []byte(`{"version":"1.0","current_phase":3,"events":["test"]}`)
	err = AutoCheckpoint(s, beforeState)
	if err != nil {
		t.Fatalf("AutoCheckpoint: %v", err)
	}

	checkpointsDir := filepath.Join(dir, "checkpoints")
	entries, err := os.ReadDir(checkpointsDir)
	if err != nil {
		t.Fatalf("read checkpoints dir: %v", err)
	}

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "auto-") {
			data, err := os.ReadFile(filepath.Join(checkpointsDir, e.Name()))
			if err != nil {
				t.Fatalf("read checkpoint file: %v", err)
			}
			trimmed := strings.TrimSpace(string(data))
			if trimmed != string(beforeState) {
				t.Errorf("checkpoint content mismatch:\ngot:  %s\nwant: %s", trimmed, string(beforeState))
			}
		}
	}
}

// TestAutoCheckpoint_PruneOldAutoCheckpoints verifies that AutoCheckpoint keeps
// only the last 10 auto-checkpoints and deletes older ones.
func TestAutoCheckpoint_PruneOldAutoCheckpoints(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Create 15 auto-checkpoints with unique timestamp names
	checkpointsDir := filepath.Join(dir, "checkpoints")
	os.MkdirAll(checkpointsDir, 0755)
	for i := 0; i < 15; i++ {
		ts := fmtCheckpointTimestamp(i)
		name := filepath.Join(checkpointsDir, "auto-"+ts+".json")
		data := []byte(`{"seq":` + itoa(i) + `}`)
		if err := os.WriteFile(name, data, 0644); err != nil {
			t.Fatalf("create checkpoint %d: %v", i, err)
		}
	}

	// Now create one more via AutoCheckpoint which should trigger pruning
	err = AutoCheckpoint(s, []byte(`{"final":"checkpoint"}`))
	if err != nil {
		t.Fatalf("AutoCheckpoint: %v", err)
	}

	entries, err := os.ReadDir(checkpointsDir)
	if err != nil {
		t.Fatalf("read checkpoints dir: %v", err)
	}

	var autoCount int
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "auto-") {
			autoCount++
		}
	}
	if autoCount > 10 {
		t.Errorf("expected at most 10 auto-checkpoints after pruning, got %d", autoCount)
	}
}

// TestAutoCheckpoint_PreservesManualCheckpoints verifies that AutoCheckpoint
// does NOT delete manual checkpoints (files without "auto-" prefix).
func TestAutoCheckpoint_PreservesManualCheckpoints(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Create 12 auto-checkpoints and 3 manual checkpoints
	checkpointsDir := filepath.Join(dir, "checkpoints")
	os.MkdirAll(checkpointsDir, 0755)
	for i := 0; i < 12; i++ {
		ts := fmtCheckpointTimestamp(i)
		name := filepath.Join(checkpointsDir, "auto-"+ts+".json")
		os.WriteFile(name, []byte(`{"auto":true}`), 0644)
	}
	for i := 0; i < 3; i++ {
		name := filepath.Join(checkpointsDir, "manual-checkpoint-"+itoa(i)+".json")
		os.WriteFile(name, []byte(`{"manual":true}`), 0644)
	}

	// Trigger pruning
	err = AutoCheckpoint(s, []byte(`{"trigger":"prune"}`))
	if err != nil {
		t.Fatalf("AutoCheckpoint: %v", err)
	}

	entries, err := os.ReadDir(checkpointsDir)
	if err != nil {
		t.Fatalf("read checkpoints dir: %v", err)
	}

	var manualCount int
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "auto-") {
			manualCount++
		}
	}
	if manualCount != 3 {
		t.Errorf("expected 3 manual checkpoints preserved, got %d", manualCount)
	}
}

// TestAutoCheckpoint_ValidJSON verifies that the checkpoint file is valid JSON.
func TestAutoCheckpoint_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	beforeState := []byte(`{"test":true}`)
	err = AutoCheckpoint(s, beforeState)
	if err != nil {
		t.Fatalf("AutoCheckpoint: %v", err)
	}

	checkpointsDir := filepath.Join(dir, "checkpoints")
	entries, err := os.ReadDir(checkpointsDir)
	if err != nil {
		t.Fatalf("read checkpoints dir: %v", err)
	}

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "auto-") {
			data, err := os.ReadFile(filepath.Join(checkpointsDir, e.Name()))
			if err != nil {
				t.Fatalf("read checkpoint file: %v", err)
			}
			if !json.Valid(data) {
				t.Errorf("checkpoint %s is not valid JSON: %s", e.Name(), string(data))
			}
		}
	}
}

// Helper functions for tests

func fmtCheckpointTimestamp(i int) string {
	hour := i / 60
	min := i % 60
	return "20240101-" + pad2(hour) + pad2(min)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return itoa(n)
}

func itoa(i int) string {
	s := ""
	if i == 0 {
		return "0"
	}
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}
