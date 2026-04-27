package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscussAnalyzeBasicScan(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a project root with go.mod and a subdirectory
	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	if err := os.MkdirAll(filepath.Join(projectRoot, "cmd"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module example\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss-analyze", "--target", projectRoot})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})

	// Must contain both "scan" and "questions" keys
	if _, ok := result["scan"]; !ok {
		t.Fatal("output missing 'scan' key")
	}
	if _, ok := result["questions"]; !ok {
		t.Fatal("output missing 'questions' key")
	}

	// Scan should detect go project
	scan := result["scan"].(map[string]interface{})
	if scan["detected_type"] != "go" {
		t.Errorf("detected_type = %v, want go", scan["detected_type"])
	}

	// Questions should have at least one entry
	questions := result["questions"].([]interface{})
	if len(questions) == 0 {
		t.Fatal("expected at least one suggested question")
	}
}

func TestDiscussAnalyzeDistinctCategories(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	if err := os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module example\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss-analyze", "--target", projectRoot})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	questions := result["questions"].([]interface{})

	// Collect all categories from the output
	existingCategories := map[string]bool{
		"surface":      true,
		"integration":  true,
		"scope":        true,
		"verification": true,
	}

	for _, q := range questions {
		question := q.(map[string]interface{})
		category := question["category"].(string)
		if existingCategories[category] {
			t.Errorf("question category %q overlaps with existing discuss categories", category)
		}
	}

	// Verify expected analyze categories are present
	expectedCategories := map[string]bool{
		"architecture":           false,
		"dependencies":           false,
		"testing_infrastructure": false,
		"deployment":             false,
		"performance":            false,
	}
	for _, q := range questions {
		question := q.(map[string]interface{})
		category := question["category"].(string)
		if _, ok := expectedCategories[category]; ok {
			expectedCategories[category] = true
		}
	}
	for cat, found := range expectedCategories {
		if !found {
			t.Errorf("expected category %q not found in questions", cat)
		}
	}
}

func TestDiscussAnalyzeEmptyDir(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Use a clean empty temp dir as target
	emptyDir := t.TempDir()

	rootCmd.SetArgs([]string{"discuss-analyze", "--target", emptyDir})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	questions := result["questions"].([]interface{})

	// Should still have questions with generic (non-empty) options
	if len(questions) == 0 {
		t.Fatal("expected questions even for empty directory")
	}

	for _, q := range questions {
		question := q.(map[string]interface{})
		options := question["options"].([]interface{})
		if len(options) == 0 {
			t.Errorf("question %q has empty options, expected generic fallback", question["question"])
		}
	}
}

func TestDiscussAnalyzeGoalFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	if err := os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module example\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	rootCmd.SetArgs([]string{"discuss-analyze", "--target", projectRoot, "--goal", "Build a REST API"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	scan := result["scan"].(map[string]interface{})

	if scan["goal"] != "Build a REST API" {
		t.Errorf("goal = %v, want 'Build a REST API'", scan["goal"])
	}
}
