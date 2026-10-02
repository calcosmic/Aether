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

// Phase 210 blocker 14 (2026-10-02): in a project that is not a git
// repository, a helper that stepped into a subfolder made every hook and
// command treat that subfolder as a brand-new project, leaving stray .aether
// folders (lock files, a welcome marker) inside the owner's deck templates.
// The project already set up further up the tree is the one that is meant.
func TestResolveAetherRootFindsTheProjectAboveASubfolder(t *testing.T) {
	t.Setenv("AETHER_ROOT", "")
	t.Setenv("AETHER_HUB_DIR", "")
	base, err := os.MkdirTemp("", "root-walk-")
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	t.Setenv("HOME", filepath.Join(base, "home"))

	project := filepath.Join(base, "projects", "deck")
	if err := os.MkdirAll(filepath.Join(project, ".aether", "data"), 0755); err != nil {
		t.Fatal(err)
	}
	templates := filepath.Join(project, "finish-the-track", "templates")
	if err := os.MkdirAll(templates, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(templates)
	if exec.Command("git", "rev-parse", "--show-toplevel").Run() == nil {
		t.Skip("fixture unexpectedly sits inside a git repository")
	}

	root := ResolveAetherRoot(context.Background())
	if !samePhysicalPath(t, root, project) {
		t.Fatalf("ResolveAetherRoot from a subfolder = %q, want the project above it %q", root, project)
	}
	if _, err := os.Stat(filepath.Join(templates, ".aether")); !os.IsNotExist(err) {
		t.Fatalf("resolving the root must not create anything in the subfolder (stat err = %v)", err)
	}
}

// The shared install every project reads from lives at ~/.aether, so the home
// folder looks like a project to a naive upward search. A folder with no
// project of its own must keep resolving to itself, never to the home folder.
func TestResolveAetherRootNeverMistakesTheHomeHubForAProject(t *testing.T) {
	t.Setenv("AETHER_ROOT", "")
	t.Setenv("AETHER_HUB_DIR", "")
	base, err := os.MkdirTemp("", "root-walk-hub-")
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	home := filepath.Join(base, "home")
	if err := os.MkdirAll(filepath.Join(home, ".aether", "data"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	fresh := filepath.Join(home, "repos", "fresh-project")
	if err := os.MkdirAll(fresh, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(fresh)
	if exec.Command("git", "rev-parse", "--show-toplevel").Run() == nil {
		t.Skip("fixture unexpectedly sits inside a git repository")
	}

	root := ResolveAetherRoot(context.Background())
	if !samePhysicalPath(t, root, fresh) {
		t.Fatalf("ResolveAetherRoot in a folder with no project = %q, want the folder itself %q (never the home hub)", root, fresh)
	}
}

func samePhysicalPath(t *testing.T, a, b string) bool {
	t.Helper()
	ai, errA := os.Stat(a)
	bi, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return false
	}
	return os.SameFile(ai, bi)
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
