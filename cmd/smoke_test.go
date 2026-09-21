package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/smoke"
	"github.com/calcosmic/Aether/pkg/storage"
)

// TestSmokeTestWritesPlatformHealth verifies that the smoke test produces
// a platform-health.json file that the dashboard consumer (computeWarnings)
// can read. This wires the producer and consumer together.
func TestSmokeTestWritesPlatformHealth(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	// Run smoke test checks and collect results
	commands := rootCmd.Commands()
	var failedCommands []string
	for _, cmd := range commands {
		var buf bytes.Buffer
		stdout = &buf
		stderr = &buf
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		rootCmd.SetArgs([]string{cmd.Name(), "--help"})
		err := rootCmd.Execute()
		if err != nil {
			failedCommands = append(failedCommands, cmd.Name())
		}
	}

	// Write platform-health.json via shared writer (producer side)
	if err := smoke.WritePlatformHealth(s, failedCommands, nil, nil); err != nil {
		t.Fatalf("failed to write platform-health.json: %v", err)
	}

	// Verify the file was written and can be read back
	var loaded map[string]interface{}
	if err := s.LoadJSON("platform-health.json", &loaded); err != nil {
		t.Fatalf("failed to read back platform-health.json: %v", err)
	}
	if _, ok := loaded["failed_commands"]; !ok {
		t.Error("platform-health.json missing 'failed_commands' key")
	}
	if _, ok := loaded["flag_mismatches"]; !ok {
		t.Error("platform-health.json missing 'flag_mismatches' key")
	}

	// Verify computeWarnings consumes it (consumer side)
	goal := "smoke health test"
	warnings := computeWarnings(colony.ColonyState{Goal: &goal}, s)
	if len(failedCommands) > 0 {
		found := false
		for _, w := range warnings {
			if bytes.Contains([]byte(w), []byte("command(s) failed smoke test")) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected platform health warning for %d failed commands, got: %v", len(failedCommands), warnings)
		}
	}
}

// TestSubcommandSmokeTest verifies that all registered subcommands
// respond cleanly to --help. This catches missing flags, panics,
// and broken registrations. Covers PLAT-05.
func TestSubcommandSmokeTest(t *testing.T) {
	commands := rootCmd.Commands()

	for _, cmd := range commands {
		cmd := cmd
		t.Run(cmd.Name(), func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)

			var buf bytes.Buffer
			stdout = &buf
			stderr = &buf
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)

			rootCmd.SetArgs([]string{cmd.Name(), "--help"})
			err := rootCmd.Execute()

			if err != nil {
				t.Errorf("subcommand %q --help returned error: %v", cmd.Name(), err)
			}
			if buf.Len() == 0 {
				t.Errorf("subcommand %q --help produced no output", cmd.Name())
			}
		})
	}
}

// TestNewSubcommandFlags validates that the subcommands registered in plan 71-02
// accept their flags without error. This ensures the new commands are wired
// correctly beyond just --help.
func TestNewSubcommandFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"suggest-approve dry-run", []string{"suggest-approve", "--dry-run"}},
		{"versions", []string{"versions"}},
		{"chamber-compare", []string{"chamber-compare", "--name", "test"}},
		{"flag-create alias", []string{"flag-create", "--title", "test", "--severity", "low"}},
		{"council parent", []string{"council", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)

			var buf bytes.Buffer
			stdout = &buf
			stderr = &buf

			tmpDir := t.TempDir()
			dataDir := tmpDir + "/.aether/data"
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				t.Fatal(err)
			}
			// The command re-opens its store from the folder it is pointed
			// at, not from the store global set below. Without this line the
			// flag-create case wrote a flag titled "test" into the REAL
			// checkout's data on every full-suite run (37 rows by 2026-09-21).
			t.Setenv("AETHER_ROOT", tmpDir)
			realFlags := realCheckoutFlagFileBytes(t)
			s, storeErr := storage.NewStore(dataDir)
			if storeErr != nil {
				t.Fatal(storeErr)
			}
			store = s

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			if err != nil {
				t.Errorf("command %v returned error: %v", tt.args, err)
			}
			if after := realCheckoutFlagFileBytes(t); !bytes.Equal(realFlags, after) {
				t.Fatalf("command %v changed the real checkout's flag file; the test is not isolated", tt.args)
			}
		})
	}
}

// realCheckoutFlagFileBytes reads the flag file of the checkout the test
// binary is running from (the package folder is cmd/, so one level up), or
// nil when there is none. Used to prove a test left real data untouched.
func realCheckoutFlagFileBytes(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", ".aether", "data", "pending-decisions.json"))
	if err != nil {
		return nil
	}
	return data
}
