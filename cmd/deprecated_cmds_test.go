package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Deprecated command registration tests
// ---------------------------------------------------------------------------

func TestDeprecatedCommandsRegistered(t *testing.T) {
	commands := []string{
		"semantic-init",
		"semantic-index",
		"semantic-search",
		"semantic-rebuild",
		"semantic-status",
		"semantic-context",
		"survey-clear",
		"survey-verify-fresh",
	}
	for _, name := range commands {
		cmd, _, err := rootCmd.Find([]string{name})
		if err != nil {
			t.Errorf("command %q not registered: %v", name, err)
			continue
		}
		if !strings.HasPrefix(cmd.Use, name) {
			t.Errorf("found command Use = %q, want prefix %q", cmd.Use, name)
		}
	}
}

// ---------------------------------------------------------------------------
// semantic-init tests
// ---------------------------------------------------------------------------

func TestSemanticInitReturnsDeprecated(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"semantic-init"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("semantic-init returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("semantic-init produced invalid JSON: %v", err)
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("deprecated = %v, want true", result["deprecated"])
	}
	if result["command"] != "semantic-init" {
		t.Errorf("command = %v, want semantic-init", result["command"])
	}
}

// ---------------------------------------------------------------------------
// semantic-index tests
// ---------------------------------------------------------------------------

func TestSemanticIndexReturnsDeprecated(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"semantic-index", "some text", "source"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("semantic-index returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("semantic-index produced invalid JSON: %v", err)
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("deprecated = %v, want true", result["deprecated"])
	}
	if result["command"] != "semantic-index" {
		t.Errorf("command = %v, want semantic-index", result["command"])
	}
}

// ---------------------------------------------------------------------------
// semantic-search tests
// ---------------------------------------------------------------------------

func TestSemanticSearchReturnsDeprecated(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"semantic-search", "test query"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("semantic-search returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("semantic-search produced invalid JSON: %v", err)
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("deprecated = %v, want true", result["deprecated"])
	}
}

// ---------------------------------------------------------------------------
// semantic-rebuild tests
// ---------------------------------------------------------------------------

func TestSemanticRebuildReturnsDeprecated(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"semantic-rebuild"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("semantic-rebuild returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("semantic-rebuild produced invalid JSON: %v", err)
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("deprecated = %v, want true", result["deprecated"])
	}
}

// ---------------------------------------------------------------------------
// semantic-status tests
// ---------------------------------------------------------------------------

func TestSemanticStatusReturnsDeprecated(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"semantic-status"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("semantic-status returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("semantic-status produced invalid JSON: %v", err)
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("deprecated = %v, want true", result["deprecated"])
	}
}

// ---------------------------------------------------------------------------
// semantic-context tests
// ---------------------------------------------------------------------------

func TestSemanticContextReturnsDeprecated(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"semantic-context", "test query"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("semantic-context returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("semantic-context produced invalid JSON: %v", err)
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("deprecated = %v, want true", result["deprecated"])
	}
}

// ---------------------------------------------------------------------------
// survey-clear tests
// ---------------------------------------------------------------------------

func TestSurveyClearReturnsDeprecated(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"survey-clear"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("survey-clear returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("survey-clear produced invalid JSON: %v", err)
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("deprecated = %v, want true", result["deprecated"])
	}
	if result["command"] != "survey-clear" {
		t.Errorf("command = %v, want survey-clear", result["command"])
	}
}

func TestSurveyClearDryRunFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	// The --dry-run flag should be accepted without error (ignored since deprecated)
	rootCmd.SetArgs([]string{"survey-clear", "--dry-run"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("survey-clear --dry-run returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// survey-verify-fresh tests
// ---------------------------------------------------------------------------

func TestSurveyVerifyFreshReturnsDeprecated(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"survey-verify-fresh", "1234567890"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("survey-verify-fresh returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("survey-verify-fresh produced invalid JSON: %v", err)
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})
	if result["deprecated"] != true {
		t.Errorf("deprecated = %v, want true", result["deprecated"])
	}
	if result["command"] != "survey-verify-fresh" {
		t.Errorf("command = %v, want survey-verify-fresh", result["command"])
	}
}

func TestSurveyVerifyFreshForceFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	// The --force flag should be accepted without error (ignored since deprecated)
	rootCmd.SetArgs([]string{"survey-verify-fresh", "--force", "1234567890"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("survey-verify-fresh --force returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// Deprecated message content test
// ---------------------------------------------------------------------------

func TestDeprecatedMessageContainsFutureVersion(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"semantic-init"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("semantic-init returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	json.Unmarshal([]byte(output), &envelope)
	result := envelope["result"].(map[string]interface{})

	msg, ok := result["message"].(string)
	if !ok {
		t.Fatal("result.message is not a string")
	}
	if !strings.Contains(msg, "deprecated") && !strings.Contains(msg, "Deprecated") {
		t.Errorf("message should mention deprecation, got: %s", msg)
	}
	if !strings.Contains(msg, "future version") {
		t.Errorf("message should mention future version removal, got: %s", msg)
	}
}
