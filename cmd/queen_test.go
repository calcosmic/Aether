package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

func TestQueenInitCreatesQUEENmd(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	os.MkdirAll(hubDir, 0755)
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

	// Verify global QUEEN.md was created
	globalPath := filepath.Join(hubDir, "QUEEN.md")
	if _, err := os.Stat(globalPath); err != nil {
		t.Fatalf("global QUEEN.md not created: %v", err)
	}
	data, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("read global QUEEN.md: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "## Wisdom") {
		t.Error("global QUEEN.md missing Wisdom section")
	}
	if !strings.Contains(text, "## Patterns") {
		t.Error("global QUEEN.md missing Patterns section")
	}

	// Verify local QUEEN.md was created
	localPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("local QUEEN.md not created: %v", err)
	}
}

func TestQueenPromoteAddsToSection(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	os.MkdirAll(hubDir, 0755)
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	// Initialize QUEEN.md first
	rootCmd.SetArgs([]string{"queen-init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("queen-init error: %v", err)
	}
	buf.Reset()
	stdout = &buf

	rootCmd.SetArgs([]string{
		"queen-promote",
		"--section", "Wisdom",
		"--content", "Always test before shipping",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["promoted"] != true {
		t.Errorf("promoted = %v, want true", result["promoted"])
	}
	if result["section"] != "Wisdom" {
		t.Errorf("section = %v, want Wisdom", result["section"])
	}

	// Verify filesystem state
	globalPath := filepath.Join(hubDir, "QUEEN.md")
	data, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("read global QUEEN.md: %v", err)
	}
	if !strings.Contains(string(data), "Always test before shipping") {
		t.Errorf("QUEEN.md missing promoted content")
	}
}

func TestQueenReadReturnsContent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	os.MkdirAll(hubDir, 0755)
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	// Initialize and promote
	rootCmd.SetArgs([]string{"queen-init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("queen-init error: %v", err)
	}

	rootCmd.SetArgs([]string{
		"queen-promote",
		"--section", "Patterns",
		"--content", "Use early returns",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("queen-promote error: %v", err)
	}

	buf.Reset()
	stdout = &buf

	rootCmd.SetArgs([]string{"queen-read"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}

	result := env["result"].(map[string]interface{})
	content, ok := result["content"].(string)
	if !ok || content == "" {
		t.Fatal("expected non-empty content in result")
	}
	if !strings.Contains(content, "Use early returns") {
		t.Error("queen-read content missing promoted entry")
	}
	if result["size"] == float64(0) {
		t.Error("expected size > 0")
	}
}

func TestQueenThresholdsReturnsDefaults(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"queen-thresholds"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["trust_promote_threshold"] != float64(0.75) {
		t.Errorf("trust_promote_threshold = %v, want 0.75", result["trust_promote_threshold"])
	}
	if result["trust_hive_threshold"] != float64(0.80) {
		t.Errorf("trust_hive_threshold = %v, want 0.80", result["trust_hive_threshold"])
	}
	if result["trust_floor"] != float64(0.2) {
		t.Errorf("trust_floor = %v, want 0.2", result["trust_floor"])
	}
	if result["max_instincts"] != float64(50) {
		t.Errorf("max_instincts = %v, want 50", result["max_instincts"])
	}
	if result["max_wisdom_entries"] != float64(200) {
		t.Errorf("max_wisdom_entries = %v, want 200", result["max_wisdom_entries"])
	}
}

func TestQueenMigrateUpgradesFormat(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	os.MkdirAll(hubDir, 0755)
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	// Write a v1 QUEEN.md in the hub (no Colony Charter section)
	v1Content := "# QUEEN.md\n\n## Wisdom\n\n## Patterns\n"
	hubStore, err := storage.NewStore(hubDir)
	if err != nil {
		t.Fatalf("create hub store: %v", err)
	}
	if err := hubStore.AtomicWrite("QUEEN.md", []byte(v1Content)); err != nil {
		t.Fatalf("write v1 QUEEN.md: %v", err)
	}

	rootCmd.SetArgs([]string{"queen-migrate"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["migrated"] != true {
		t.Errorf("migrated = %v, want true", result["migrated"])
	}

	// Verify filesystem state in hub
	data, err := hubStore.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	if !strings.Contains(string(data), "## Colony Charter") {
		t.Error("migrated QUEEN.md missing Colony Charter section")
	}
}
