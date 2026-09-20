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
	t.Run("visual mode overrides a buffered host writer", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		acceptedOutputModeBuildTestColony(t)
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		rootCmd.SetArgs([]string{"build", "1"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build returned error: %v", err)
		}

		assertVisualOutputBreaksJSON(t, stdout.(*bytes.Buffer).String(), "B U I L D")
	})

	t.Run("json mode is one clean envelope on a buffered host writer", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		acceptedOutputModeBuildTestColony(t)
		t.Setenv("AETHER_OUTPUT_MODE", "json")

		rootCmd.SetArgs([]string{"build", "1"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build returned error: %v", err)
		}

		output := bytes.TrimSpace(stdout.(*bytes.Buffer).Bytes())
		var envelope map[string]interface{}
		if err := json.Unmarshal(output, &envelope); err != nil {
			t.Fatalf("json mode did not produce one valid envelope: %v\n%s", err, output)
		}
		if envelope["ok"] != true {
			t.Fatalf("json mode envelope = %#v, want ok:true", envelope)
		}
		for _, visualNoise := range []string{"B U I L D", "──", "🐜", "[FAKE-NARRATOR]"} {
			if bytes.Contains(output, []byte(visualNoise)) {
				t.Fatalf("json mode leaked visual or narrator text %q:\n%s", visualNoise, output)
			}
		}
	})

	t.Run("explicit modes override terminal writer inference", func(t *testing.T) {
		terminal, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			t.Fatalf("open terminal-shaped writer: %v", err)
		}
		defer terminal.Close()
		if !isTerminalWriter(terminal) {
			t.Skipf("%s is not reported as a character device on this platform", os.DevNull)
		}

		t.Setenv("AETHER_OUTPUT_MODE", "visual")
		if !shouldRenderVisualOutput(terminal) {
			t.Fatal("explicit visual mode lost to terminal writer inference")
		}
		t.Setenv("AETHER_OUTPUT_MODE", "json")
		if shouldRenderVisualOutput(terminal) {
			t.Fatal("explicit json mode lost to terminal writer inference")
		}
	})
}

func acceptedOutputModeBuildTestColony(t *testing.T) {
	t.Helper()
	goal := "Capture buffered build output regression"
	taskID := "1.1"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Reproduce failure",
				Description: "Show that explicit output mode controls buffered build output",
				Status:      colony.PhaseReady,
				Tasks:       []colony.Task{{ID: &taskID, Goal: "Trigger build output", Status: colony.TaskPending}},
			}},
		},
	})
	withWorkingDir(t, accepted.Root)
}

func TestInstallBufferedOutputBreaksJSONUnderVisualEnv(t *testing.T) {
	// Manages its own hub via --home-dir; opt out of suite-wide hub isolation.
	t.Setenv("AETHER_HUB_DIR", "")
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
