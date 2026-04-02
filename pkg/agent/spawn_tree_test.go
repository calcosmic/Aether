package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aether-colony/aether/pkg/storage"
)

func TestSpawnTreeRecordSpawn(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")

	err = st.RecordSpawn("colony-prime", "builder", "worker-1", "build phase 1", 1)
	if err != nil {
		t.Fatalf("RecordSpawn() error: %v", err)
	}

	entries := st.entries
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].AgentName != "worker-1" {
		t.Errorf("AgentName = %q, want %q", entries[0].AgentName, "worker-1")
	}
	if entries[0].ParentName != "colony-prime" {
		t.Errorf("ParentName = %q, want %q", entries[0].ParentName, "colony-prime")
	}
	if entries[0].Caste != "builder" {
		t.Errorf("Caste = %q, want %q", entries[0].Caste, "builder")
	}
	if entries[0].Task != "build phase 1" {
		t.Errorf("Task = %q, want %q", entries[0].Task, "build phase 1")
	}
	if entries[0].Depth != 1 {
		t.Errorf("Depth = %d, want 1", entries[0].Depth)
	}
	if entries[0].Status != "spawned" {
		t.Errorf("Status = %q, want %q", entries[0].Status, "spawned")
	}
	if entries[0].Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

func TestSpawnTreeUpdateStatus(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	st.RecordSpawn("colony-prime", "builder", "worker-1", "build task", 1)

	err = st.UpdateStatus("worker-1", "completed")
	if err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	if st.entries[0].Status != "completed" {
		t.Errorf("Status = %q, want %q", st.entries[0].Status, "completed")
	}

	if len(st.completions) != 1 {
		t.Fatalf("expected 1 completion line, got %d", len(st.completions))
	}
	if st.completions[0].Name != "worker-1" {
		t.Errorf("completion name = %q, want %q", st.completions[0].Name, "worker-1")
	}
	if st.completions[0].Status != "completed" {
		t.Errorf("completion status = %q, want %q", st.completions[0].Status, "completed")
	}

	// Test updating non-existent agent
	err = st.UpdateStatus("nonexistent", "failed")
	if err == nil {
		t.Error("expected error for non-existent agent")
	}
}

func TestSpawnTreeFormat(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	st.RecordSpawn("colony-prime", "builder", "agent-name", "task", 1)

	// Read the raw file
	data, err := store.ReadFile("spawn-tree.txt")
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	line := strings.TrimSpace(string(data))

	// Must have exactly 6 pipe separators (7 fields)
	pipeCount := strings.Count(line, "|")
	if pipeCount != 6 {
		t.Errorf("line has %d pipe separators, want 6 (7 fields): %q", pipeCount, line)
	}

	// Verify field structure
	fields := strings.Split(line, "|")
	if len(fields) != 7 {
		t.Fatalf("line has %d fields, want 7: %q", len(fields), line)
	}
	if fields[1] != "colony-prime" {
		t.Errorf("field[1] (parent) = %q, want %q", fields[1], "colony-prime")
	}
	if fields[2] != "builder" {
		t.Errorf("field[2] (caste) = %q, want %q", fields[2], "builder")
	}
	if fields[3] != "agent-name" {
		t.Errorf("field[3] (name) = %q, want %q", fields[3], "agent-name")
	}
	if fields[4] != "task" {
		t.Errorf("field[4] (task) = %q, want %q", fields[4], "task")
	}
	if fields[5] != "1" {
		t.Errorf("field[5] (depth) = %q, want %q", fields[5], "1")
	}
	if fields[6] != "spawned" {
		t.Errorf("field[6] (status) = %q, want %q", fields[6], "spawned")
	}
}

func TestSpawnTreeParseRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")

	// Record multiple entries
	st.RecordSpawn("colony-prime", "builder", "worker-1", "build task", 1)
	st.RecordSpawn("colony-prime", "watcher", "worker-2", "watch task", 1)
	st.UpdateStatus("worker-1", "completed")

	// Parse the file
	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Parse() returned %d entries, want 2", len(entries))
	}

	// First entry should have status merged from completion line
	if entries[0].AgentName != "worker-1" {
		t.Errorf("entries[0].AgentName = %q, want %q", entries[0].AgentName, "worker-1")
	}
	if entries[0].Status != "completed" {
		t.Errorf("entries[0].Status = %q, want %q", entries[0].Status, "completed")
	}

	// Second entry should still be spawned
	if entries[1].AgentName != "worker-2" {
		t.Errorf("entries[1].AgentName = %q, want %q", entries[1].AgentName, "worker-2")
	}
	if entries[1].Status != "spawned" {
		t.Errorf("entries[1].Status = %q, want %q", entries[1].Status, "spawned")
	}
}

func TestSpawnTreeParseShellFormat(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	// Write shell-formatted content directly
	shellContent := `2026-04-01T12:00:00Z|colony-prime|builder|worker-1|build task|1|spawned
2026-04-01T12:00:01Z|colony-prime|watcher|worker-2|watch task|1|spawned
2026-04-01T12:05:00Z|worker-1|completed|
`
	store.AtomicWrite("spawn-tree.txt", []byte(shellContent))

	st := NewSpawnTree(store, "spawn-tree.txt")
	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Parse() returned %d entries, want 2", len(entries))
	}

	// First entry should have status merged from completion line
	if entries[0].AgentName != "worker-1" {
		t.Errorf("entries[0].AgentName = %q, want %q", entries[0].AgentName, "worker-1")
	}
	if entries[0].Status != "completed" {
		t.Errorf("entries[0].Status = %q, want completed (merged from completion line)", entries[0].Status)
	}
	if entries[0].Caste != "builder" {
		t.Errorf("entries[0].Caste = %q, want %q", entries[0].Caste, "builder")
	}
	if entries[0].Depth != 1 {
		t.Errorf("entries[0].Depth = %d, want 1", entries[0].Depth)
	}

	// Second entry should still be spawned
	if entries[1].AgentName != "worker-2" {
		t.Errorf("entries[1].AgentName = %q, want %q", entries[1].AgentName, "worker-2")
	}
	if entries[1].Status != "spawned" {
		t.Errorf("entries[1].Status = %q, want %q", entries[1].Status, "spawned")
	}
}

func TestSpawnTreeActive(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	st.RecordSpawn("colony-prime", "builder", "worker-1", "build", 1)
	st.RecordSpawn("colony-prime", "builder", "worker-2", "build", 1)
	st.RecordSpawn("colony-prime", "watcher", "worker-3", "watch", 1)
	st.UpdateStatus("worker-1", "completed")

	active := st.Active()
	if len(active) != 2 {
		t.Fatalf("Active() returned %d entries, want 2", len(active))
	}

	names := make(map[string]bool)
	for _, e := range active {
		names[e.AgentName] = true
		if e.Status != "spawned" {
			t.Errorf("Active() entry %q has status %q, want %q", e.AgentName, e.Status, "spawned")
		}
	}
	if !names["worker-2"] || !names["worker-3"] {
		t.Errorf("Active() returned agents %v, want worker-2 and worker-3", names)
	}
}

func TestSpawnTreeToJSON(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	st.RecordSpawn("colony-prime", "builder", "worker-1", "build task", 1)
	st.RecordSpawn("colony-prime", "watcher", "worker-2", "watch task", 1)
	st.UpdateStatus("worker-1", "completed")

	data, err := st.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}

	// Verify top-level keys
	if _, ok := result["spawns"]; !ok {
		t.Error("JSON missing 'spawns' key")
	}
	if _, ok := result["metadata"]; !ok {
		t.Error("JSON missing 'metadata' key")
	}

	// Verify metadata
	var metadata struct {
		TotalCount     int `json:"total_count"`
		ActiveCount    int `json:"active_count"`
		CompletedCount int `json:"completed_count"`
	}
	if err := json.Unmarshal(result["metadata"], &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if metadata.TotalCount != 2 {
		t.Errorf("total_count = %d, want 2", metadata.TotalCount)
	}
	if metadata.ActiveCount != 1 {
		t.Errorf("active_count = %d, want 1", metadata.ActiveCount)
	}
	if metadata.CompletedCount != 1 {
		t.Errorf("completed_count = %d, want 1", metadata.CompletedCount)
	}

	// Verify spawns array
	var spawns []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(result["spawns"], &spawns); err != nil {
		t.Fatalf("unmarshal spawns: %v", err)
	}
	if len(spawns) != 2 {
		t.Fatalf("spawns has %d entries, want 2", len(spawns))
	}
}

func TestSpawnTreeEmpty(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	// No file exists -- should not error
	st := NewSpawnTree(store, "spawn-tree.txt")

	if len(st.entries) != 0 {
		t.Errorf("expected 0 entries for missing file, got %d", len(st.entries))
	}

	active := st.Active()
	if len(active) != 0 {
		t.Errorf("Active() returned %d entries, want 0", len(active))
	}

	// ToJSON on empty tree
	data, err := st.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	var result struct {
		Spawns   []interface{} `json:"spawns"`
		Metadata struct {
			TotalCount     int `json:"total_count"`
			ActiveCount    int `json:"active_count"`
			CompletedCount int `json:"completed_count"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal empty JSON: %v", err)
	}
	if result.Metadata.TotalCount != 0 {
		t.Errorf("empty total_count = %d, want 0", result.Metadata.TotalCount)
	}
}
