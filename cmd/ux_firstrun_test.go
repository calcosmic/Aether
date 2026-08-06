package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestFirstRunShowsWelcomeWhenNoMarkerAndNoColony covers both platforms
// deliberately. This banner is the first thing a new user ever sees, and it
// listed three raw CLI commands that a Claude Code user cannot type. It also
// pads its command column, so the two platforms exercise different widths —
// hand-counted padding aligned on exactly one of them.
func TestFirstRunShowsWelcomeWhenNoMarkerAndNoColony(t *testing.T) {
	for _, tc := range []struct {
		platform string
		wantCmds []string
	}{
		{"claude", []string{"/ant-lay-eggs", "/ant-init", "/ant-status"}},
		{"opencode", []string{"/ant-lay-eggs", "/ant-init", "/ant-status"}},
		{"codex", []string{"aether lay-eggs", "aether init", "aether status"}},
	} {
		t.Run(tc.platform, func(t *testing.T) {
			tmpDir := t.TempDir()
			dataDir := tmpDir
			// Ensure .aether/data/ exists (the PersistentPreRunE creates it)
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				t.Fatalf("failed to create data dir: %v", err)
			}

			// Force visual mode since bytes.Buffer is not a TTY
			t.Setenv("AETHER_OUTPUT_MODE", "visual")
			t.Setenv("AETHER_PLATFORM", tc.platform)

			var buf bytes.Buffer
			oldStdout := stdout
			stdout = &buf
			defer func() { stdout = oldStdout }()

			checkAndEmitFirstRun(dataDir)

			output := buf.String()
			if !strings.Contains(output, "W E L C O M E") && !strings.Contains(output, "Welcome") {
				t.Errorf("expected welcome banner to contain welcome text, got: %s", output)
			}
			for _, want := range tc.wantCmds {
				if !strings.Contains(output, want) {
					t.Errorf("expected welcome banner to contain %q, got: %s", want, output)
				}
			}

			// The command column must line up: every description starts at the
			// same offset regardless of how wide the platform's names are.
			// Anchored on the descriptions rather than the commands: one command
			// row carries a quoted argument, which a command-anchored
			// measurement would mistake for the start of the description.
			descriptions := []string{
				"Set up Aether in this repo",
				"Start a colony with a goal",
				"Check on your colony",
			}
			offsets := map[string]int{}
			for _, line := range strings.Split(output, "\n") {
				for _, desc := range descriptions {
					if idx := strings.Index(line, desc); idx >= 0 {
						offsets[desc] = idx
					}
				}
			}
			if len(offsets) != len(descriptions) {
				t.Fatalf("expected %d description rows on %s, found %d:\n%s",
					len(descriptions), tc.platform, len(offsets), output)
			}
			wantCol := offsets[descriptions[0]]
			for _, desc := range descriptions {
				if offsets[desc] != wantCol {
					t.Errorf("welcome banner columns misaligned on %s (%q at %d, expected %d):\n%s",
						tc.platform, desc, offsets[desc], wantCol, output)
				}
			}

			// Marker file should have been created
			if _, err := os.Stat(dataDir + "/.welcomed"); err != nil {
				t.Errorf("expected .welcomed marker file to be created, got error: %v", err)
			}
		})
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
