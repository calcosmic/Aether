package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQueenInitCreatesBothFiles(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	os.MkdirAll(hubDir, 0755) // hub dir must exist for hubStore()
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	rootCmd.SetArgs([]string{"queen-init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}

	// Check global was created
	if _, err := os.Stat(filepath.Join(hubDir, "QUEEN.md")); err != nil {
		t.Fatalf("global QUEEN.md not created: %v", err)
	}

	// Check local was created
	localPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("local QUEEN.md not created at %s: %v", localPath, err)
	}

	// Verify local has standard sections
	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("read local: %v", err)
	}
	text := string(data)
	for _, section := range []string{"## Wisdom", "## Patterns", "## Colony Charter"} {
		if !strings.Contains(text, section) {
			t.Fatalf("local QUEEN.md missing %q", section)
		}
	}
}

func TestCharterWriteTargetsLocal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	rootCmd.SetArgs([]string{"charter-write", "--name", "TestColony", "--goal", "Fix queen"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}

	// Verify written to local, not global
	localPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("read local QUEEN.md: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "TestColony") {
		t.Fatalf("local QUEEN.md missing colony name:\n%s", text)
	}
	if !strings.Contains(text, "Fix queen") {
		t.Fatalf("local QUEEN.md missing colony goal:\n%s", text)
	}

	// Verify NOT in global
	globalPath := filepath.Join(hubDir, "QUEEN.md")
	if gData, err := os.ReadFile(globalPath); err == nil {
		gText := string(gData)
		if strings.Contains(gText, "TestColony") {
			t.Fatal("global QUEEN.md should NOT contain local colony charter")
		}
	}
}

func TestQueenWriteLearningsTargetsLocal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	learnings := `[{"claim":"Always use tabs for Go indentation"}]`
	rootCmd.SetArgs([]string{"queen-write-learnings", learnings})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}
	result, _ := env["result"].(map[string]interface{})
	if result["target"] != "local" {
		t.Fatalf("expected target:local, got %v", result["target"])
	}

	// Verify in local
	localPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("read local QUEEN.md: %v", err)
	}
	if !strings.Contains(string(data), "Always use tabs for Go indentation") {
		t.Fatalf("local QUEEN.md missing learning:\n%s", string(data))
	}
}

func TestQueenPromoteInstinctTargetsLocal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	// Create instincts.json with a test instinct
	instincts := map[string]interface{}{
		"instincts": []map[string]interface{}{
			{"id": "inst-test-1", "action": "Prefer table-driven tests", "confidence": 0.9},
		},
	}
	instData, _ := json.Marshal(instincts)
	if err := os.WriteFile(filepath.Join(s.BasePath(), "instincts.json"), instData, 0644); err != nil {
		t.Fatalf("write instincts: %v", err)
	}

	rootCmd.SetArgs([]string{"queen-promote-instinct", "--id", "inst-test-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}

	// Verify in local
	localPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("read local QUEEN.md: %v", err)
	}
	if !strings.Contains(string(data), "Prefer table-driven tests") {
		t.Fatalf("local QUEEN.md missing instinct:\n%s", string(data))
	}
}

func TestReadQUEENMdBulletFormat(t *testing.T) {
	content := `# QUEEN.md

## Wisdom

- Always use tabs for Go indentation (phase learning, 2026-04-25)
- Prefer table-driven tests (instinct inst-test-1, 2026-04-25)

## Patterns

- Factory pattern for agent creation (promoted 2026-04-24)
`

	tmpFile := filepath.Join(t.TempDir(), "QUEEN.md")
	os.WriteFile(tmpFile, []byte(content), 0644)

	result := readQUEENMd(tmpFile)
	if len(result) == 0 {
		t.Fatal("readQUEENMd returned empty map for bullet-format content")
	}

	// Should find at least the bullet entries
	found1 := false
	found2 := false
	for k := range result {
		if strings.Contains(k, "Always use tabs") {
			found1 = true
		}
		if strings.Contains(k, "Factory pattern") {
			found2 = true
		}
	}
	if !found1 {
		t.Fatalf("missing wisdom bullet entry, got: %v", result)
	}
	if !found2 {
		t.Fatalf("missing patterns bullet entry, got: %v", result)
	}
}

func TestReadQUEENMdMixedFormat(t *testing.T) {
	content := `# QUEEN.md

## Wisdom

key_style: Some key-value entry
- Bullet style entry (promoted 2026-04-25)
`

	tmpFile := filepath.Join(t.TempDir(), "QUEEN.md")
	os.WriteFile(tmpFile, []byte(content), 0644)

	result := readQUEENMd(tmpFile)
	if len(result) < 2 {
		t.Fatalf("expected at least 2 entries, got %d: %v", len(result), result)
	}

	hasKey := false
	hasBullet := false
	for k := range result {
		if k == "key_style" {
			hasKey = true
		}
		if strings.Contains(k, "Bullet style entry") {
			hasBullet = true
		}
	}
	if !hasKey {
		t.Fatal("missing key:value entry")
	}
	if !hasBullet {
		t.Fatal("missing bullet entry")
	}
}
