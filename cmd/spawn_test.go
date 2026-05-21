package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/agent"
)

func TestSpawnTreeLoadReturnsJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed a spawn entry directly via SpawnTree
	st := agent.NewSpawnTree(s, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Mason-1", "Build feature X", 1); err != nil {
		t.Fatalf("record spawn: %v", err)
	}

	rootCmd.SetArgs([]string{"spawn-tree-load"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-tree-load returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	// JSON output should contain spawns key
	if _, ok := result["spawns"]; !ok {
		t.Fatalf("expected spawns in output, got keys: %v", result)
	}
}

func TestSpawnTreeActiveListsActive(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	st := agent.NewSpawnTree(s, "spawn-tree.txt")
	_ = st.RecordSpawn("Queen", "builder", "Mason-1", "Task A", 1)
	_ = st.RecordSpawn("Queen", "watcher", "Watcher-1", "Task B", 1)
	// Complete one
	_ = st.UpdateStatus("Mason-1", "completed", "")

	rootCmd.SetArgs([]string{"spawn-tree-active"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-tree-active returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	count, ok := result["count"].(float64)
	if !ok {
		t.Fatalf("expected count as number, got: %T", result["count"])
	}
	if count != 1 {
		t.Fatalf("expected 1 active spawn, got %v", count)
	}
}

func TestSpawnEfficiencyCalculatesPercent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	st := agent.NewSpawnTree(s, "spawn-tree.txt")
	_ = st.RecordSpawn("Queen", "builder", "Mason-1", "Task A", 1)
	_ = st.RecordSpawn("Queen", "builder", "Mason-2", "Task B", 1)
	_ = st.UpdateStatus("Mason-1", "completed", "")

	rootCmd.SetArgs([]string{"spawn-efficiency"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-efficiency returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	total := result["total"].(float64)
	completed := result["completed"].(float64)
	efficiency := result["efficiency"].(string)
	if total != 2 {
		t.Fatalf("expected total=2, got %v", total)
	}
	if completed != 1 {
		t.Fatalf("expected completed=1, got %v", completed)
	}
	if efficiency != "50.0%" {
		t.Fatalf("expected efficiency=50.0%%, got %q", efficiency)
	}
}

func TestValidateWorkerResponseChecksLength(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"validate-worker-response", "--response", "OK"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("validate-worker-response returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Fatalf("expected valid=false for short response, got: %v", result["valid"])
	}
	reason := result["reason"].(string)
	if reason != "response too short" {
		t.Fatalf("expected reason 'response too short', got: %q", reason)
	}
}

func TestValidateWorkerResponseValidJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"validate-worker-response", "--response", `{"status":"ok"}`, "--expect-json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("validate-worker-response returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	if result["valid"] != true {
		t.Fatalf("expected valid=true for valid JSON, got: %v", result["valid"])
	}
}

func TestValidateWorkerResponseInvalidJSON_Spawn(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"validate-worker-response", "--response", "not json at all here", "--expect-json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("validate-worker-response returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Fatalf("expected valid=false for invalid JSON, got: %v", result["valid"])
	}
}
