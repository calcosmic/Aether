package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestHistoryJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)
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

	binding := bindCommandTestRepository(t)

	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	if err := os.WriteFile(binding.DataDir+"/COLONY_STATE.json", []byte(state), 0644); err != nil {
		t.Fatalf("write empty colony state: %v", err)
	}

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

	bindSeededCommandTestRepository(t)
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

	bindSeededCommandTestRepository(t)
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

	bindSeededCommandTestRepository(t)
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
	binding := bindCommandTestRepository(t)

	// Write colony state with empty events
	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	if err := os.WriteFile(binding.DataDir+"/COLONY_STATE.json", []byte(state), 0644); err != nil {
		t.Fatalf("write empty colony state: %v", err)
	}

	rootCmd.SetArgs([]string{"history"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("history returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No events") {
		t.Errorf("expected 'No events' for empty history, got: %s", output)
	}
	for _, forbidden := range []string{
		"Worker Results",
		"S P A W N   P L A N",
		"Host platform should dispatch",
		"finalizer",
		"completed a real worker pass",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("history dashboard implied worker theatre via %q\n%s", forbidden, output)
		}
	}
}

func TestHistoryCurationFlagsRestoreRepositoryAuthority200(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{name: "curation", run: TestCurationSentinelDryRun},
		{name: "flags", run: TestFlagsListJSON},
		{name: "history", run: TestHistoryJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalRoot := t.TempDir()
			originalDataDir := originalRoot + "/.aether/data"
			if err := os.MkdirAll(originalDataDir, 0755); err != nil {
				t.Fatalf("create original data directory: %v", err)
			}
			t.Setenv("AETHER_ROOT", originalRoot)
			t.Setenv("COLONY_DATA_DIR", originalDataDir)
			store = nil
			tracer = nil

			t.Run("command fixture", tt.run)

			if got := os.Getenv("AETHER_ROOT"); got != originalRoot {
				t.Errorf("AETHER_ROOT = %q after cleanup, want %q", got, originalRoot)
			}
			if got := os.Getenv("COLONY_DATA_DIR"); got != originalDataDir {
				t.Errorf("COLONY_DATA_DIR = %q after cleanup, want %q", got, originalDataDir)
			}
			if store != nil || tracer != nil {
				t.Errorf("repository authority leaked after cleanup: store=%p tracer=%p", store, tracer)
			}
		})
	}
}
