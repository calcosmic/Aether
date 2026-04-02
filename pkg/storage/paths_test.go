package storage

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAetherRoot_EnvVar(t *testing.T) {
	custom := "/tmp/custom-aether"
	t.Setenv("AETHER_ROOT", custom)

	root := ResolveAetherRoot()
	if root != custom {
		t.Errorf("ResolveAetherRoot with AETHER_ROOT set: got %q, want %q", root, custom)
	}
}

func TestResolveAetherRoot_GitFallback(t *testing.T) {
	t.Setenv("AETHER_ROOT", "")

	root := ResolveAetherRoot()

	// Should return the git root since we're in a git repo
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err == nil {
		expected := strings.TrimSpace(string(out))
		// The root should contain .aether (this repo)
		if !strings.Contains(root, "Aether") {
			t.Errorf("ResolveAetherRoot git fallback: got %q, expected a path containing 'Aether'", root)
		}
		if root != expected {
			t.Errorf("ResolveAetherRoot git fallback: got %q, want %q", root, expected)
		}
	}
}

func TestResolveDataDir_ColonyDataDir(t *testing.T) {
	custom := "/tmp/my-colony-data"
	t.Setenv("COLONY_DATA_DIR", custom)

	dir := ResolveDataDir()
	if dir != custom {
		t.Errorf("ResolveDataDir with COLONY_DATA_DIR: got %q, want %q", dir, custom)
	}
}

func TestResolveDataDir_Default(t *testing.T) {
	t.Setenv("COLONY_DATA_DIR", "")
	t.Setenv("AETHER_ROOT", "/tmp/testroot")

	dir := ResolveDataDir()
	expected := filepath.Join("/tmp/testroot", ".aether", "data")
	if dir != expected {
		t.Errorf("ResolveDataDir default: got %q, want %q", dir, expected)
	}
}
