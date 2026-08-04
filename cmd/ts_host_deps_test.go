package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTsHostFixture(t *testing.T, tsHostDir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(tsHostDir, "dist"), 0755); err != nil {
		t.Fatalf("create ts-host dist: %v", err)
	}
	pkg := `{"name":"aether-ts-host","dependencies":{"js-yaml":"4.1.1"}}`
	if err := os.WriteFile(filepath.Join(tsHostDir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tsHostDir, "package-lock.json"), []byte(`{"lockfileVersion":3}`), 0644); err != nil {
		t.Fatalf("write package-lock.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tsHostDir, "dist", "host.js"), []byte("// host"), 0644); err != nil {
		t.Fatalf("write dist/host.js: %v", err)
	}
}

// stubTsHostNpm replaces the npm runner with a recorder that simulates a
// successful `npm ci` by creating node_modules. Restored on test cleanup.
func stubTsHostNpm(t *testing.T) *[][]string {
	t.Helper()
	var calls [][]string
	original := tsHostNpmCommand
	tsHostNpmCommand = func(dir string, args ...string) error {
		calls = append(calls, append([]string{dir}, args...))
		if len(args) > 0 && args[0] == "ci" {
			if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0755); err != nil {
				return err
			}
		}
		return nil
	}
	t.Cleanup(func() { tsHostNpmCommand = original })
	return &calls
}

func npmCiCalls(calls [][]string) int {
	n := 0
	for _, call := range calls {
		if len(call) > 1 && call[1] == "ci" {
			n++
		}
	}
	return n
}

// TestHostRunProvisionsDependencies is the WP2 gate: invoking the host with a
// dist/ build but no node_modules (the state every fresh install lands in)
// must provision dependencies before exec instead of dying inside node.
func TestHostRunProvisionsDependencies(t *testing.T) {
	saveGlobals(t)
	repoDir := t.TempDir()
	tsHostDir := filepath.Join(repoDir, ".aether", "ts-host")
	writeTsHostFixture(t, tsHostDir)
	calls := stubTsHostNpm(t)

	hostPath, hint := prepareTsHostForExec(repoDir)
	if hostPath == "" {
		t.Fatalf("prepareTsHostForExec failed: %s", hint)
	}
	if want := filepath.Join(tsHostDir, "dist", "host.js"); hostPath != want {
		t.Fatalf("resolved host path %q, want %q", hostPath, want)
	}
	if npmCiCalls(*calls) != 1 {
		t.Fatalf("expected exactly one npm ci, got calls: %v", *calls)
	}
	stamp, err := os.ReadFile(filepath.Join(tsHostDir, filepath.FromSlash(tsHostLockStampRel)))
	if err != nil {
		t.Fatalf("dependency stamp not written: %v", err)
	}
	wantHash, err := fileSHA256(filepath.Join(tsHostDir, "package-lock.json"))
	if err != nil {
		t.Fatalf("hash lock: %v", err)
	}
	if strings.TrimSpace(string(stamp)) != wantHash {
		t.Fatalf("stamp %q does not match lock hash %q", stamp, wantHash)
	}

	// Second invocation with a matching stamp must not reinstall.
	if hostPath, hint = prepareTsHostForExec(repoDir); hostPath == "" {
		t.Fatalf("second prepareTsHostForExec failed: %s", hint)
	}
	if npmCiCalls(*calls) != 1 {
		t.Fatalf("npm ci re-ran despite fresh stamp, calls: %v", *calls)
	}
}

// TestEnsureTsHostBuiltReinstallsOnLockChange locks the stale-deps fix: a
// node_modules installed against an older package-lock.json must trigger a
// reinstall instead of being silently accepted.
func TestEnsureTsHostBuiltReinstallsOnLockChange(t *testing.T) {
	saveGlobals(t)
	repoDir := t.TempDir()
	tsHostDir := filepath.Join(repoDir, ".aether", "ts-host")
	writeTsHostFixture(t, tsHostDir)
	calls := stubTsHostNpm(t)

	if err := ensureTsHostBuilt(repoDir); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	if npmCiCalls(*calls) != 1 {
		t.Fatalf("expected one npm ci on first ensure, calls: %v", *calls)
	}

	if err := os.WriteFile(filepath.Join(tsHostDir, "package-lock.json"), []byte(`{"lockfileVersion":3,"changed":true}`), 0644); err != nil {
		t.Fatalf("mutate lock: %v", err)
	}
	if err := ensureTsHostBuilt(repoDir); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	if npmCiCalls(*calls) != 2 {
		t.Fatalf("expected reinstall after lock change, calls: %v", *calls)
	}

	// Legacy installs (node_modules present, no stamp) must also reinstall once.
	if err := os.Remove(filepath.Join(tsHostDir, filepath.FromSlash(tsHostLockStampRel))); err != nil {
		t.Fatalf("remove stamp: %v", err)
	}
	if err := ensureTsHostBuilt(repoDir); err != nil {
		t.Fatalf("third ensure: %v", err)
	}
	if npmCiCalls(*calls) != 3 {
		t.Fatalf("expected reinstall when stamp missing, calls: %v", *calls)
	}
}
