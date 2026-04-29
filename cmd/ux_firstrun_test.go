package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestFirstRunShowsWelcomeWhenNoMarkerAndNoColony(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := tmpDir
	// Ensure .aether/data/ exists (the PersistentPreRunE creates it)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}

	// Force visual mode since bytes.Buffer is not a TTY
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	checkAndEmitFirstRun(dataDir)

	output := buf.String()
	if !strings.Contains(output, "W E L C O M E") && !strings.Contains(output, "Welcome") {
		t.Errorf("expected welcome banner to contain welcome text, got: %s", output)
	}
	if !strings.Contains(output, "aether lay-eggs") {
		t.Errorf("expected welcome banner to contain 'aether lay-eggs', got: %s", output)
	}
	if !strings.Contains(output, "aether init") {
		t.Errorf("expected welcome banner to contain 'aether init', got: %s", output)
	}

	// Marker file should have been created
	if _, err := os.Stat(dataDir + "/.welcomed"); err != nil {
		t.Errorf("expected .welcomed marker file to be created, got error: %v", err)
	}
}

func TestFirstRunSkipsWhenMarkerExists(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	// Create marker file before call
	if err := os.WriteFile(tmpDir+"/.welcomed", []byte(""), 0644); err != nil {
		t.Fatalf("failed to create marker file: %v", err)
	}

	var buf bytes.Buffer
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	checkAndEmitFirstRun(tmpDir)

	if buf.Len() != 0 {
		t.Errorf("expected no output when marker exists, got: %s", buf.String())
	}
}

func TestFirstRunSkipsWhenColonyExists(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	// Create COLONY_STATE.json with valid colony data
	colonyData := `{"goal":"test","state":"READY"}`
	if err := os.WriteFile(tmpDir+"/COLONY_STATE.json", []byte(colonyData), 0644); err != nil {
		t.Fatalf("failed to create colony state: %v", err)
	}

	var buf bytes.Buffer
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	checkAndEmitFirstRun(tmpDir)

	if buf.Len() != 0 {
		t.Errorf("expected no output when colony exists, got: %s", buf.String())
	}
}

func TestFirstRunSkipsInJSONMode(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}

	t.Setenv("AETHER_OUTPUT_MODE", "json")

	var buf bytes.Buffer
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	checkAndEmitFirstRun(tmpDir)

	if buf.Len() != 0 {
		t.Errorf("expected no output in JSON mode, got: %s", buf.String())
	}
}
