package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestLifecycleTransactionCrossFilesystemRollback(t *testing.T) {
	fixture := newLifecycleTransactionFixture(t)
	repositoryTarget := filepath.Join(fixture.repositoryRoot, "repository.txt")
	hubTarget := filepath.Join(fixture.hubRoot, "hub.txt")
	mustWriteLifecycleFixtureFile(t, repositoryTarget, []byte("repository-before"))
	mustWriteLifecycleFixtureFile(t, hubTarget, []byte("hub-before"))

	config := fixture.config("cross-filesystem-rollback")
	config.Rename = func(oldPath, newPath string) error {
		if filepath.Dir(oldPath) != filepath.Dir(newPath) {
			t.Fatalf("rename crossed directories: %s -> %s", oldPath, newPath)
		}
		if newPath == hubTarget {
			return syscall.EXDEV
		}
		return os.Rename(oldPath, newPath)
	}
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootRepository, "repository.txt", []byte("repository-after")); err != nil {
		t.Fatal(err)
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootHubStable, "hub.txt", []byte("hub-after")); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(); !errors.Is(err, syscall.EXDEV) {
		t.Fatalf("commit error = %v, want EXDEV", err)
	}

	if got := string(mustReadLifecycleFixtureFile(t, repositoryTarget)); got != "repository-before" {
		t.Fatalf("repository target after rollback = %q", got)
	}
	if got := string(mustReadLifecycleFixtureFile(t, hubTarget)); got != "hub-before" {
		t.Fatalf("hub target after rollback = %q", got)
	}
	progressBytes := mustReadLifecycleFixtureFile(t, filepath.Join(lifecycleJournalPath(fixture, "cross-filesystem-rollback"), "progress.json"))
	var progress struct {
		Stage colony.LifecycleTransactionStage `json:"stage"`
	}
	if err := json.Unmarshal(progressBytes, &progress); err != nil {
		t.Fatalf("decode progress: %v", err)
	}
	if progress.Stage != colony.TransactionStageRolledBack {
		t.Fatalf("progress stage = %q, want rolled_back", progress.Stage)
	}
	if _, err := os.Stat(filepath.Join(lifecycleJournalPath(fixture, "cross-filesystem-rollback"), "receipt.json")); !os.IsNotExist(err) {
		t.Fatalf("rolled-back transaction emitted success receipt: %v", err)
	}
}

func TestLifecycleTransactionMultiRootFaultMatrix(t *testing.T) {
	fixture, config, tx, targets := prepareLifecycleFaultTransaction(t, "multi-root-exdev", "")
	hubTarget := targets[2].path
	renameCount := 0
	config.Rename = func(oldPath, newPath string) error {
		renameCount++
		if filepath.Dir(oldPath) != filepath.Dir(newPath) {
			t.Fatalf("rename crossed root/filesystem boundary: %s -> %s", oldPath, newPath)
		}
		if newPath == hubTarget {
			return syscall.EXDEV
		}
		return os.Rename(oldPath, newPath)
	}
	tx.config.Rename = config.Rename

	if _, err := tx.Commit(); !errors.Is(err, syscall.EXDEV) {
		t.Fatalf("commit error = %v, want fake EXDEV", err)
	}
	assertLifecycleTransactionSnapshot(t, targets, false)
	if renameCount < 3 {
		t.Fatalf("rename seam observed %d operations, want commit plus reverse-order rollback", renameCount)
	}
	progressBytes := mustReadLifecycleFixtureFile(t, filepath.Join(lifecycleJournalPath(fixture, "multi-root-exdev"), "progress.json"))
	var progress lifecycleTransactionProgress
	if err := decodeLifecycleJSON(progressBytes, &progress); err != nil {
		t.Fatalf("decode progress: %v", err)
	}
	if progress.Stage != colony.TransactionStageRolledBack || progress.StateEffect != colony.LifecycleStateEffectRolledBack {
		t.Fatalf("rollback progress = %#v", progress)
	}
}
