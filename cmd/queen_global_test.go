package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQueenPromoteInstinctWritesGlobal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	s, tmpDir := newTestStore(t)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", "") })

	// Create hub QUEEN.md
	os.MkdirAll(hubDir, 0755)
	queenContent := "# QUEEN.md\n\n## Wisdom\n\n"
	os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(queenContent), 0644)

	// Create instincts.json with a test instinct
	instinctData := map[string]interface{}{
		"instincts": []map[string]interface{}{
			{
				"id":        "inst_test_001",
				"action":    "Always write tests first",
				"confidence": 0.95,
			},
		},
	}
	instinctJSON, _ := json.Marshal(instinctData)
	os.WriteFile(filepath.Join(s.BasePath(), "instincts.json"), instinctJSON, 0644)

	// Run promote-instinct
	rootCmd.SetArgs([]string{"queen-promote-instinct", "--id", "inst_test_001"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}
	result := env["result"].(map[string]interface{})
	if result["promoted"] != true {
		t.Fatal("expected promoted:true")
	}

	// Verify hub QUEEN.md was written
	hubQueen, err := os.ReadFile(filepath.Join(hubDir, "QUEEN.md"))
	if err != nil {
		t.Fatalf("read hub QUEEN.md: %v", err)
	}
	if !strings.Contains(string(hubQueen), "Always write tests first") {
		t.Fatalf("hub QUEEN.md missing promoted instinct:\n%s", string(hubQueen))
	}
}

func TestQueenPromoteInstinctSucceedsWithoutHub(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	s, tmpDir := newTestStore(t)
	store = s

	// No hub directory -- hubStore() will return nil
	os.Setenv("AETHER_HUB_DIR", filepath.Join(tmpDir, "nonexistent_hub"))
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", "") })

	// Create instincts.json
	instinctData := map[string]interface{}{
		"instincts": []map[string]interface{}{
			{
				"id":        "inst_test_002",
				"action":    "Test without hub",
				"confidence": 0.85,
			},
		},
	}
	instinctJSON, _ := json.Marshal(instinctData)
	os.WriteFile(filepath.Join(s.BasePath(), "instincts.json"), instinctJSON, 0644)

	// Run promote-instinct -- should succeed (local write)
	rootCmd.SetArgs([]string{"queen-promote-instinct", "--id", "inst_test_002"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true even without hub, got %v", env)
	}
}
