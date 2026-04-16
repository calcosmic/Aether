package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

func TestHistoryJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"history", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("history --json returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	events, ok := result["events"].([]interface{})
	if !ok {
		t.Fatalf("result.events is not an array, got: %T", result["events"])
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

func TestHistoryJSONEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"history", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("history --json with empty events returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
	result := envelope["result"].(map[string]interface{})
	events := result["events"].([]interface{})
	if len(events) != 0 {
		t.Errorf("expected 0 events for empty case, got %d", len(events))
	}
}

func TestHistoryDefault(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"history"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("history returned error: %v", err)
	}

	output := buf.String()

	// Testdata has 3 events
	if !strings.Contains(output, "📜") {
		t.Errorf("expected visual banner, got: %s", output)
	}
	if !strings.Contains(output, "init") {
		t.Errorf("expected 'init' event type, got: %s", output)
	}
	if !strings.Contains(output, "build") {
		t.Errorf("expected 'build' event type, got: %s", output)
	}
	if !strings.Contains(output, "complete") {
		t.Errorf("expected 'complete' event type, got: %s", output)
	}
	// Events should show newest first
	completeIdx := strings.Index(output, "complete")
	buildIdx := strings.Index(output, "build")
	initIdx := strings.Index(output, "init")
	if completeIdx >= buildIdx {
		t.Errorf("expected newest event first (complete before build)")
	}
	if buildIdx >= initIdx {
		t.Errorf("expected newest event first (build before init)")
	}
}

func TestHistoryWithLimit(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"history", "--limit", "2"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("history --limit 2 returned error: %v", err)
	}

	output := buf.String()
	// With limit 2, should only show the 2 newest events
	if !strings.Contains(output, "complete") {
		t.Errorf("expected newest event 'complete', got: %s", output)
	}
	if !strings.Contains(output, "build") {
		t.Errorf("expected second newest 'build', got: %s", output)
	}
	// init should NOT appear (it's the oldest)
	initCount := strings.Count(output, "init")
	if initCount > 1 {
		// "init" appears in column header "Timestamp" -> count appearances carefully
		// Actually check for the event message "Colony initialized"
		if strings.Contains(output, "Colony initialized") {
			t.Errorf("did not expect oldest event with --limit 2, got: %s", output)
		}
	}
}

func TestHistoryWithFilter(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"history", "--filter", "build"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("history --filter build returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "build") {
		t.Errorf("expected 'build' event, got: %s", output)
	}
}

func TestHistoryEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	// Create store with colony state but no events
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	// Write colony state with empty events
	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"history"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("history returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No events") {
		t.Errorf("expected 'No events' for empty history, got: %s", output)
	}
}
