package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

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
			os.MkdirAll(dataDir, 0755)
			s, _ := storage.NewStore(dataDir)
			store = s

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			if err != nil {
				t.Errorf("command %v returned error: %v", tt.args, err)
			}
		})
	}
}
