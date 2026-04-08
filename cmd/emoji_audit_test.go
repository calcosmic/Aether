package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestEmojiAudit(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))

	// Create test directories and files
	cmdDir := filepath.Join(projectRoot, ".claude", "commands", "ant")
	os.MkdirAll(cmdDir, 0755)
	os.WriteFile(filepath.Join(cmdDir, "test.md"), []byte("# Test \U0001F41C Command\nSome text"), 0644)

	yamlDir := filepath.Join(projectRoot, ".aether", "commands")
	os.MkdirAll(yamlDir, 0755)
	os.WriteFile(filepath.Join(yamlDir, "test.yaml"), []byte("name: test\ndesc: \u2705 OK"), 0644)

	// Change to project root so relative globs work
	origDir, _ := os.Getwd()
	os.Chdir(projectRoot)
	defer os.Chdir(origDir)

	rootCmd.SetArgs([]string{"emoji-audit"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["files_scanned"] != float64(2) {
		t.Errorf("files_scanned = %v, want 2", result["files_scanned"])
	}
	if result["total_emojis"] == nil || result["total_emojis"].(float64) < 2 {
		t.Errorf("total_emojis = %v, want >= 2", result["total_emojis"])
	}
}

func TestEmojiAuditEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Use a clean temp dir with no matching files
	emptyDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(emptyDir)
	defer os.Chdir(origDir)

	rootCmd.SetArgs([]string{"emoji-audit"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["files_scanned"] != float64(0) {
		t.Errorf("files_scanned = %v, want 0", result["files_scanned"])
	}
	if result["total_emojis"] != float64(0) {
		t.Errorf("total_emojis = %v, want 0", result["total_emojis"])
	}
	files := result["files"].([]interface{})
	if len(files) != 0 {
		t.Errorf("files = %v, want empty", files)
	}
}

func TestCountEmojisInString(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello", 0},
		{"\U0001F41C ant", 1},
		{"\U0001F41C\U0001F41C", 2},
		{"\u2705 check \u274C cross", 2},
		{"\U0001F528\U0001F3D7 build", 2},
	}

	for _, tc := range tests {
		got := countEmojisInString(tc.input)
		if got != tc.want {
			t.Errorf("countEmojisInString(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
