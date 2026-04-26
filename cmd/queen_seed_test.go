package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestQueenSeedFromHiveFiltersDuplicates(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	s, tmpDir := newTestStore(t)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", "") })

	// Create hive wisdom with two entries
	wisdomDir := filepath.Join(hubDir, "hive")
	os.MkdirAll(wisdomDir, 0755)
	wisdomData := map[string]interface{}{
		"entries": []map[string]interface{}{
			{"text": "Always test your code", "confidence": 0.9},
			{"text": "Ship fast learn fast", "confidence": 0.85},
		},
	}
	wisdomJSON, _ := json.Marshal(wisdomData)
	os.WriteFile(filepath.Join(wisdomDir, "wisdom.json"), wisdomJSON, 0644)

	// Create hub QUEEN.md with one of the entries already present
	queenContent := "# QUEEN.md\n\n## Wisdom\n\n- Always test your code (promoted 2026-04-01)\n"
	os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(queenContent), 0644)

	// Run seed command
	rootCmd.SetArgs([]string{"queen-seed-from-hive"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}
	result := env["result"].(map[string]interface{})
	seeded := int(result["seeded"].(float64))
	skipped := int(result["skipped"].(float64))
	if seeded != 1 {
		t.Fatalf("expected 1 new entry seeded, got %d", seeded)
	}
	if skipped != 1 {
		t.Fatalf("expected 1 entry skipped (duplicate), got %d", skipped)
	}
}

func TestQueenSeedFromHiveSecondRunSeedsZero(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	s, tmpDir := newTestStore(t)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", "") })

	// Create hive wisdom
	wisdomDir := filepath.Join(hubDir, "hive")
	os.MkdirAll(wisdomDir, 0755)
	wisdomData := map[string]interface{}{
		"entries": []map[string]interface{}{
			{"text": "Unique wisdom here", "confidence": 0.9},
		},
	}
	wisdomJSON, _ := json.Marshal(wisdomData)
	os.WriteFile(filepath.Join(wisdomDir, "wisdom.json"), wisdomJSON, 0644)

	// Create empty QUEEN.md
	queenContent := "# QUEEN.md\n\n## Wisdom\n\n"
	os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(queenContent), 0644)

	// First run -- should seed 1
	rootCmd.SetArgs([]string{"queen-seed-from-hive"})
	rootCmd.Execute()

	// Second run -- should seed 0
	buf.Reset()
	rootCmd.SetArgs([]string{"queen-seed-from-hive"})
	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	seeded := int(result["seeded"].(float64))
	if seeded != 0 {
		t.Fatalf("expected 0 new entries on second run, got %d", seeded)
	}
}

func TestIsEntryInText(t *testing.T) {
	text := "# QUEEN.md\n\n## Wisdom\n\n- Always test your code (promoted 2026-04-01)\n"
	if !isEntryInText(text, "- Always test your code (hive wisdom)") {
		t.Fatal("expected duplicate to be detected")
	}
	if isEntryInText(text, "- Completely different wisdom (hive wisdom)") {
		t.Fatal("expected non-duplicate to pass")
	}
}
