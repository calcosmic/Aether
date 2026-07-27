package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// D-04: `aether data-clean` prunes worker-debug artifacts.
//
// The dry-run case is the one that matters most. CLAUDE.md's Definition of Done
// names this corollary explicitly — "an inspection or --dry-run command must not
// mutate state" — because `consolidation-phase-end --dry-run` and
// `consolidation-seal --dry-run` wrote to instincts.json for months while their
// flag help read "Report without modifying".

func seedWorkerDebugDir(t *testing.T, dir string, fresh, stale int) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < fresh; i++ {
		p := filepath.Join(dir, fmt.Sprintf("fresh-%03d.json", i))
		if err := os.WriteFile(p, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		mt := time.Now().Add(-time.Duration(i) * time.Minute)
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < stale; i++ {
		p := filepath.Join(dir, fmt.Sprintf("stale-%03d.json", i))
		if err := os.WriteFile(p, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		old := time.Now().Add(-codex.WorkerDebugRetentionMaxAge - time.Hour)
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}
}

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			n++
		}
	}
	return n
}

func TestDataCleanWorkerDebugDryRunDoesNotMutate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "worker-debug")
	seedWorkerDebugDir(t, dir, 3, 5)
	before := countFiles(t, dir)

	total, prunable, removed, err := pruneWorkerDebugDirectory(dir, false)
	if err != nil {
		t.Fatalf("pruneWorkerDebugDirectory: %v", err)
	}

	if removed != 0 {
		t.Errorf("dry run reported %d removed, want 0", removed)
	}
	if after := countFiles(t, dir); after != before {
		t.Errorf("dry run deleted files: %d before, %d after — an inspection command must not mutate state (CLAUDE.md Definition of Done)", before, after)
	}
	if total != before {
		t.Errorf("total = %d, want %d", total, before)
	}
	if prunable != 5 {
		t.Errorf("prunable = %d, want 5 — dry run must still report what WOULD be removed, or it is not useful as an inspection", prunable)
	}
}

func TestDataCleanWorkerDebugConfirmPrunesExpired(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "worker-debug")
	seedWorkerDebugDir(t, dir, 3, 5)

	_, prunable, removed, err := pruneWorkerDebugDirectory(dir, true)
	if err != nil {
		t.Fatalf("pruneWorkerDebugDirectory: %v", err)
	}

	if removed != prunable {
		t.Errorf("removed = %d but prunable = %d — confirmed mode must delete exactly what the dry run advertised", removed, prunable)
	}
	if after := countFiles(t, dir); after != 3 {
		t.Errorf("%d files remain, want 3 (the fresh ones) — expired artifacts were not pruned", after)
	}
}

func TestDataCleanWorkerDebugEnforcesFileCap(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "worker-debug")
	over := codex.WorkerDebugRetentionMaxFiles + 12
	seedWorkerDebugDir(t, dir, over, 0)

	if _, _, _, err := pruneWorkerDebugDirectory(dir, true); err != nil {
		t.Fatalf("pruneWorkerDebugDirectory: %v", err)
	}

	if after := countFiles(t, dir); after > codex.WorkerDebugRetentionMaxFiles {
		t.Errorf("%d files remain, cap is %d — the file cap is not enforced by data-clean", after, codex.WorkerDebugRetentionMaxFiles)
	}
}

// A missing directory is the normal case on a fresh colony; it must not error.
func TestDataCleanWorkerDebugMissingDirectoryIsNotAnError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")

	total, prunable, removed, err := pruneWorkerDebugDirectory(dir, true)
	if err != nil {
		t.Errorf("missing worker-debug dir returned error %v — a fresh colony would fail data-clean", err)
	}
	if total != 0 || prunable != 0 || removed != 0 {
		t.Errorf("missing dir reported total=%d prunable=%d removed=%d, want all zero", total, prunable, removed)
	}
}
