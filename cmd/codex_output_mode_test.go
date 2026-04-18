package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBuildBufferedOutputBreaksJSONUnderVisualEnv(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	goal := "Capture buffered build output regression"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Reproduce failure",
				Description: "Show that ambient visual mode contaminates buffered build output",
				Status:      colony.PhaseReady,
				Tasks:       []colony.Task{{ID: &taskID, Goal: "Trigger build output", Status: colony.TaskPending}},
			}},
		},
	})

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	assertVisualOutputBreaksJSON(t, stdout.(*bytes.Buffer).String(), "B U I L D")
}

func TestInstallBufferedOutputBreaksJSONUnderVisualEnv(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	homeDir := t.TempDir()
	workDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	rootCmd.SetArgs([]string{"install", "--home-dir", homeDir, "--skip-build-binary"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	assertVisualOutputBreaksJSON(t, buf.String(), "I N S T A L L")
}

func assertVisualOutputBreaksJSON(t *testing.T, output, marker string) {
	t.Helper()

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err == nil {
		t.Fatalf("expected visual output instead of JSON envelope, got %v", envelope)
	}
	if !strings.Contains(output, marker) {
		t.Fatalf("expected visual output marker %q, got:\n%s", marker, output)
	}
}
