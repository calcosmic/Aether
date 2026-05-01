package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestUnblock_NoGateResults(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Test colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 5,
	})

	rootCmd.SetArgs([]string{"unblock", "--phase", "5"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unblock returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, "No gate results found for phase 5") {
		t.Fatalf("expected 'No gate results found for phase 5' in output, got: %s", output)
	}
}

func TestUnblock_ShowFailedGates(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Test colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 3,
	})

	// Write gate results with 2 failed and 1 passed gate
	results := []GateCheckResult{
		{
			Name:      "tests_pass",
			Status:    "failed",
			Detail:    "3 tests failed in pkg/core",
			FixHint:   "Run go test ./pkg/core to see failures",
			Timestamp: "2026-05-01T10:00:00Z",
			RetryCount: 0,
		},
		{
			Name:      "flags",
			Status:    "failed",
			Detail:    "2 unresolved blocker flags",
			FixHint:   "Resolve blockers with /ant-flags --resolve",
			RecoveryOptions: []string{"Fix flag 1", "Fix flag 2"},
			Timestamp: "2026-05-01T10:00:01Z",
			RetryCount: 1,
		},
		{
			Name:      "spawn_gate",
			Status:    "passed",
			Detail:    "3 workers spawned",
			Timestamp: "2026-05-01T10:00:02Z",
			RetryCount: 0,
		},
	}
	data, _ := json.MarshalIndent(results, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-3.json"), data, 0644); err != nil {
		t.Fatalf("failed to write gate results: %v", err)
	}

	rootCmd.SetArgs([]string{"unblock", "--phase", "3"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unblock returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	summary, ok := result["summary"].(string)
	if !ok {
		t.Fatalf("expected 'summary' field in output, got: %s", output)
	}

	// Must show the 2 failed gates
	if !strings.Contains(summary, "tests_pass") {
		t.Error("summary should contain failed gate 'tests_pass'")
	}
	if !strings.Contains(summary, "flags") {
		t.Error("summary should contain failed gate 'flags'")
	}

	// Must show fix hints
	if !strings.Contains(summary, "Run go test ./pkg/core to see failures") {
		t.Error("summary should contain fix_hint for tests_pass gate")
	}
	if !strings.Contains(summary, "Resolve blockers with /ant-flags --resolve") {
		t.Error("summary should contain fix_hint for flags gate")
	}

	// Must show recovery options
	if !strings.Contains(summary, "Fix flag 1") {
		t.Error("summary should contain recovery option 'Fix flag 1'")
	}
	if !strings.Contains(summary, "Fix flag 2") {
		t.Error("summary should contain recovery option 'Fix flag 2'")
	}

	// Must NOT list the passed gate as failed
	if strings.Contains(summary, "Failed Gates:\n\n  Gate: spawn_gate") {
		t.Error("summary should NOT list passed gate 'spawn_gate' as failed")
	}
}

func TestUnblock_RecoveryOptions(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Test colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
	})

	// Write gate results with 1 failed gate
	results := []GateCheckResult{
		{
			Name:      "anti_pattern",
			Status:    "failed",
			Detail:    "Critical anti-pattern detected",
			FixHint:   "Fix the anti-pattern",
			Timestamp: "2026-05-01T10:00:00Z",
			RetryCount: 0,
		},
	}
	data, _ := json.MarshalIndent(results, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-2.json"), data, 0644); err != nil {
		t.Fatalf("failed to write gate results: %v", err)
	}

	rootCmd.SetArgs([]string{"unblock", "--phase", "2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unblock returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	summary, ok := result["summary"].(string)
	if !ok {
		t.Fatalf("expected 'summary' field in output, got: %s", output)
	}

	// Must include recovery option 1
	if !strings.Contains(summary, "Fix the issues above manually, then run /ant-continue") {
		t.Error("summary should contain recovery option 1: fix manually")
	}

	// Must include recovery option 2
	if !strings.Contains(summary, "View detailed fix hints for each gate above") {
		t.Error("summary should contain recovery option 2: view fix hints")
	}
}

func TestUnblock_NoForbiddenStrings(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Test colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 4,
	})

	// Write gate results with 1 failed gate
	results := []GateCheckResult{
		{
			Name:      "tests_pass",
			Status:    "failed",
			Detail:    "Tests failed",
			FixHint:   "Fix the tests",
			Timestamp: "2026-05-01T10:00:00Z",
			RetryCount: 0,
		},
	}
	data, _ := json.MarshalIndent(results, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-4.json"), data, 0644); err != nil {
		t.Fatalf("failed to write gate results: %v", err)
	}

	rootCmd.SetArgs([]string{"unblock", "--phase", "4"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unblock returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()

	// Parse JSON to get the summary string
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	summary, ok := result["summary"].(string)
	if !ok {
		t.Fatalf("expected 'summary' field in output, got: %s", output)
	}

	// Output must NOT contain forbidden strings
	if strings.Contains(summary, "CRITICAL: Do NOT proceed") {
		t.Error("summary must NOT contain forbidden string 'CRITICAL: Do NOT proceed'")
	}
	if strings.Contains(summary, "The phase will NOT advance") {
		t.Error("summary must NOT contain forbidden string 'The phase will NOT advance'")
	}
}
