package storage

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestResolveAetherRoot_EnvVar(t *testing.T) {
	custom := "/tmp/custom-aether"
	t.Setenv("AETHER_ROOT", custom)

	root := ResolveAetherRoot(context.Background())
	if root != custom {
		t.Errorf("ResolveAetherRoot with AETHER_ROOT set: got %q, want %q", root, custom)
	}
}

func TestResolveAetherRoot_GitFallback(t *testing.T) {
	t.Setenv("AETHER_ROOT", "")

	// t.TempDir embeds this test's name, which would mask an "Aether" name dependency.
	repoDir, err := os.MkdirTemp("", "root-resolver-")
	if err != nil {
		t.Fatalf("create git fixture: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(repoDir); err != nil {
			t.Errorf("remove git fixture: %v", err)
		}
	})
	cmd := exec.Command("git", "init", "--quiet", repoDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("initialize git fixture: %v\n%s", err, out)
	}

	// Git reports the physical path, including macOS's /private temporary roots.
	expected, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatalf("resolve git fixture path: %v", err)
	}
	nested := filepath.Join(expected, "nested", "child")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}
	t.Chdir(nested)

	root := ResolveAetherRoot(context.Background())
	if root != expected {
		t.Errorf("ResolveAetherRoot git fallback: got %q, want %q", root, expected)
	}
}

func TestResolveDataDir_ColonyDataDir(t *testing.T) {
	custom := "/tmp/my-colony-data"
	t.Setenv("COLONY_DATA_DIR", custom)

	dir := ResolveDataDir(context.Background())
	if dir != custom {
		t.Errorf("ResolveDataDir with COLONY_DATA_DIR: got %q, want %q", dir, custom)
	}
}

func TestResolveDataDir_Default(t *testing.T) {
	t.Setenv("COLONY_DATA_DIR", "")
	t.Setenv("AETHER_ROOT", "/tmp/testroot")

	dir := ResolveDataDir(context.Background())
	expected := filepath.Join("/tmp/testroot", ".aether", "data")
	if dir != expected {
		t.Errorf("ResolveDataDir default: got %q, want %q", dir, expected)
	}
}
