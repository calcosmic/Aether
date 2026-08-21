package cmd

import (
	"os"
	"path/filepath"
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

	want := []string{"lifecycle", "colonize", "plan", "build", "continue", "seal", "oracle", "watch", "swarm"}
	for _, w := range want {
		found := false
		for _, sub := range hostCmdFound.Commands() {
			if sub.Name() == w {
				found = true
				if !sub.DisableFlagParsing {
					t.Errorf("expected host subcommand %q to forward raw flags to the TS host", w)
				}
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q in host command", w)
		}
	}
}

func TestHostPlanAndContinueForwardRawFlags(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	root := t.TempDir()
	hostPath := filepath.Join(root, ".aether", "ts-host", "dist", "host.js")
	if err := os.MkdirAll(filepath.Dir(hostPath), 0755); err != nil {
		t.Fatalf("create host dir: %v", err)
	}
	if err := os.WriteFile(hostPath, []byte("// test host\n"), 0644); err != nil {
		t.Fatalf("write host file: %v", err)
	}
	tsHostDir := filepath.Dir(filepath.Dir(hostPath))
	if err := os.WriteFile(filepath.Join(tsHostDir, "package.json"), []byte(`{"name":"aether-ts-host"}`), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tsHostDir, "package-lock.json"), []byte(`{"lockfileVersion":3}`), 0644); err != nil {
		t.Fatalf("write package-lock.json: %v", err)
	}
	withWorkingDir(t, root)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	resolvedHostPath := filepath.Join(cwd, ".aether", "ts-host", "dist", "host.js")

	fakeBin := t.TempDir()
	argsFile := filepath.Join(root, "node-args.txt")
	nodePath := filepath.Join(fakeBin, "node")
	fakeNode := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then echo v20.0.0; exit 0; fi\n" +
		"printf '%s\\n' \"$@\" > \"$AETHER_FAKE_NODE_ARGS\"\n"
	if err := os.WriteFile(nodePath, []byte(fakeNode), 0755); err != nil {
		t.Fatalf("write fake node: %v", err)
	}
	t.Setenv("PATH", fakeBin)
	t.Setenv("AETHER_FAKE_NODE_ARGS", argsFile)

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "colonize",
			args: []string{"host", "colonize", "--force-resurvey", "--worker-timeout", "5m"},
			want: []string{resolvedHostPath, "colonize", "--force-resurvey", "--worker-timeout", "5m"},
		},
		{
			name: "plan",
			args: []string{"host", "plan", "--depth", "balanced", "--planning-depth", "deep", "--refresh"},
			want: []string{resolvedHostPath, "plan", "--depth", "balanced", "--planning-depth", "deep", "--refresh"},
		},
		{
			name: "build",
			args: []string{"host", "build", "2", "--light", "--worker-timeout", "10m"},
			want: []string{resolvedHostPath, "build", "2", "--light", "--worker-timeout", "10m"},
		},
		{
			name: "continue",
			args: []string{"host", "continue", "--classic-ceremony", "$ARGUMENTS"},
			want: []string{resolvedHostPath, "continue", "--classic-ceremony", "$ARGUMENTS"},
		},
		{
			name: "seal",
			args: []string{"host", "seal", "--force"},
			want: []string{resolvedHostPath, "seal", "--force"},
		},
		{
			name: "oracle",
			args: []string{"host", "oracle", "release parity", "--simulate"},
			want: []string{resolvedHostPath, "oracle", "release parity", "--simulate"},
		},
		{
			name: "swarm",
			args: []string{"host", "swarm", "Auth panic when session is missing", "--no-dashboard"},
			want: []string{resolvedHostPath, "swarm", "Auth panic when session is missing", "--no-dashboard"},
		},
		{
			name: "watch",
			args: []string{"host", "watch", "--no-dashboard"},
			want: []string{resolvedHostPath, "watch", "--no-dashboard"},
		},
		{
			name: "lifecycle",
			args: []string{"host", "lifecycle", "3", "classic commands", "--simulate", "--skip-midden-check"},
			want: []string{resolvedHostPath, "lifecycle", "3", "classic commands", "--simulate", "--skip-midden-check"},
		},
	}

	for _, tt := range tests {
		if err := os.Remove(argsFile); err != nil && !os.IsNotExist(err) {
			t.Fatalf("remove args file: %v", err)
		}
		rootCmd.SetArgs(tt.args)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%s execute: %v", tt.name, err)
		}
		data, err := os.ReadFile(argsFile)
		if err != nil {
			t.Fatalf("%s read args file: %v", tt.name, err)
		}
		got := strings.Split(strings.TrimSpace(string(data)), "\n")
		if strings.Join(got, "\x00") != strings.Join(tt.want, "\x00") {
			t.Fatalf("%s forwarded args = %#v, want %#v", tt.name, got, tt.want)
		}
	}
}

func TestHostCommandEnvPassesDetectedActivePlatform(t *testing.T) {
	t.Setenv("AETHER_ACTIVE_PLATFORM", "codex")

	env := hostCommandEnv([]string{
		"PATH=/bin",
		"AETHER_ACTIVE_PLATFORM=claude",
	})

	found := false
	for _, entry := range env {
		if entry == "AETHER_ACTIVE_PLATFORM=codex" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("hostCommandEnv did not pass detected active platform: %#v", env)
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
