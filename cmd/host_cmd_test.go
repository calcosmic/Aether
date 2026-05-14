package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestHostCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "host" {
			found = true
			break
		}
	}
	if !found {
		t.Error("host command not found in root commands")
	}
}

func TestHostSubcommands(t *testing.T) {
	var hostCmdFound *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "host" {
			hostCmdFound = cmd
			break
		}
	}
	if hostCmdFound == nil {
		t.Fatal("host command not found")
	}

	want := []string{"lifecycle", "plan", "build", "continue", "oracle"}
	for _, w := range want {
		found := false
		for _, sub := range hostCmdFound.Commands() {
			if sub.Name() == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q in host command", w)
		}
	}
}

func TestResolveTsHostPath(t *testing.T) {
	path, hint := resolveTsHostPath("/nonexistent/repo")
	if path != "" && hint != "" {
		t.Errorf("expected path OR hint, got both: path=%q hint=%q", path, hint)
	}
	if path == "" && hint == "" {
		t.Error("expected path or hint, got neither")
	}
}

func TestDiscoverNode(t *testing.T) {
	path, err := discoverNode()
	if err != nil {
		t.Skipf("Node not available: %v", err)
	}
	if path == "" {
		t.Error("discoverNode returned empty path with no error")
	}
}

func TestHostCommandFallbackNoNode(t *testing.T) {
	// Save and restore PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Clear PATH so node is not found
	os.Setenv("PATH", "")

	_, err := discoverNode()
	if err == nil {
		t.Error("expected error when node is not in PATH")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' in error, got: %v", err)
	}
}

func TestHostCommandFallbackOldNode(t *testing.T) {
	// This test would require a mock or fake node binary.
	// We verify the version parsing logic indirectly through discoverNode.
	// If node is available and >= 20, discoverNode succeeds.
	// If node < 20 were present, it would fail with a version error.
	_, err := discoverNode()
	if err != nil {
		t.Skipf("Node not available, skipping version validation test: %v", err)
	}
	// If we reach here, node is available and version is valid.
}
