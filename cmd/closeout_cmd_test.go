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

func TestCloseoutRendersCompletionWorkerResults(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	store = s
	goal := "restore visual wrappers"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Visual closeout",
			Status: colony.PhaseInProgress,
		}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	completion := map[string]interface{}{
		"dispatch_manifest": map[string]interface{}{
			"phase":      1,
			"phase_name": "Visual closeout",
			"dispatches": []map[string]interface{}{
				{"name": "Forge-1", "caste": "builder"},
			},
		},
		"dispatches": []map[string]interface{}{
			{
				"name":           "Forge-1",
				"caste":          "builder",
				"status":         "completed",
				"summary":        "Implemented the wrapper restoration",
				"files_modified": []string{"cmd/closeout_cmd.go"},
			},
		},
	}
	data, err := json.Marshal(completion)
	if err != nil {
		t.Fatalf("marshal completion: %v", err)
	}
	completionPath := filepath.Join(tmpDir, "completion.json")
	if err := os.WriteFile(completionPath, data, 0644); err != nil {
		t.Fatalf("write completion: %v", err)
	}

	rootCmd.SetArgs([]string{"closeout", "build", "--completion-file", completionPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("closeout returned error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"Worker Results", "Forge-1", "Implemented the wrapper restoration", "cmd/closeout_cmd.go"} {
		if !strings.Contains(output, want) {
			t.Fatalf("closeout output missing %q:\n%s", want, output)
		}
	}
}

func TestSealCloseoutRendersPorterReadinessSection(t *testing.T) {
	output := renderCloseoutVisual(map[string]interface{}{
		"workflow":          "seal",
		"state_available":   true,
		"state":             string(colony.StateCOMPLETED),
		"current_phase":     1,
		"total_phases":      1,
		"completed_phases":  1,
		"porter_readiness":  "Porter readiness details",
		"completion_loaded": false,
		"next":              "Run `aether porter check`.",
	})
	for _, want := range []string{"Post-Seal: Delivery Readiness", "Porter readiness details"} {
		if !strings.Contains(output, want) {
			t.Fatalf("seal closeout missing %q:\n%s", want, output)
		}
	}
}
