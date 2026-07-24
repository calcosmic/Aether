package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

func TestMigrateStateRollbackRestoresExactBackupAndKeepsSafetyCopy(t *testing.T) {
	originalStore := store
	defer func() { store = originalStore }()
	dataDir := t.TempDir()
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create migration store: %v", err)
	}
	store = s
	legacy := []byte(`{
  "version": "2.0",
  "goal": "restore this exact state",
  "state": "READY",
  "plan": {"phases": []}
}`)
	if err := store.AtomicWrite("COLONY_STATE.json", legacy); err != nil {
		t.Fatalf("write legacy state: %v", err)
	}

	migration, err := runMigrateState(false)
	if err != nil {
		t.Fatalf("migrate state: %v", err)
	}
	backupPath := stringValue(migration["backup_path"])
	if backupPath == "" {
		t.Fatal("migration did not produce a backup")
	}
	migrated, err := store.ReadFile("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read migrated state: %v", err)
	}
	if bytes.Equal(migrated, legacy) {
		t.Fatal("migration did not change the state")
	}

	rollback, err := runMigrateStateRollback(backupPath, false)
	if err != nil {
		t.Fatalf("rollback migration: %v", err)
	}
	if !boolValue(rollback["rolled_back"]) {
		t.Fatalf("rollback did not report completion: %+v", rollback)
	}
	restored, err := store.ReadFile("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read restored state: %v", err)
	}
	if !bytes.Equal(restored, legacy) {
		t.Fatalf("rollback did not restore the byte-exact legacy state:\n%s", restored)
	}
	safetyPath := filepath.FromSlash(stringValue(rollback["safety_backup"]))
	safety, err := store.ReadFile(safetyPath)
	if err != nil {
		t.Fatalf("read pre-rollback safety backup: %v", err)
	}
	if !bytes.Equal(safety, migrated) {
		t.Fatal("pre-rollback safety backup does not contain the migrated state")
	}
	wantCommand := "aether migrate-state --rollback " + filepath.ToSlash(backupPath)
	if got := stringValue(migration["rollback_command"]); got != wantCommand {
		t.Fatalf("rollback command = %q, want %q", got, wantCommand)
	}
}

func TestMigrateStateRollbackRejectsUntrustedPaths(t *testing.T) {
	originalStore := store
	defer func() { store = originalStore }()
	dataDir := t.TempDir()
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create migration store: %v", err)
	}
	store = s
	if err := store.AtomicWrite("COLONY_STATE.json", []byte(`{"version":"3.0","state":"READY"}`)); err != nil {
		t.Fatalf("write current state: %v", err)
	}

	for _, path := range []string{
		"../COLONY_STATE.json",
		"COLONY_STATE.json",
		"backups/arbitrary.json",
		filepath.Join(dataDir, "backups", "COLONY_STATE.pre-migrate.escape.json"),
	} {
		if _, err := runMigrateStateRollback(path, false); err == nil {
			t.Fatalf("untrusted rollback path %q was accepted", path)
		}
	}

	backupDir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("create backup dir: %v", err)
	}
	target := filepath.Join(dataDir, "outside.json")
	if err := os.WriteFile(target, []byte(`{"version":"2.0","state":"READY"}`), 0644); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	symlink := filepath.Join(backupDir, "COLONY_STATE.pre-migrate.symlink.json")
	if err := os.Symlink(target, symlink); err == nil {
		_, rollbackErr := runMigrateStateRollback(filepath.Join("backups", filepath.Base(symlink)), false)
		if rollbackErr == nil || !strings.Contains(rollbackErr.Error(), "symbolic link") {
			t.Fatalf("symlink rollback error = %v, want symbolic-link rejection", rollbackErr)
		}
	}
}
