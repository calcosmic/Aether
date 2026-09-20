package agent

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

func newRepositorySpawnTree200(t *testing.T) (*SpawnTree, *storage.Store, string) {
	t.Helper()

	repositoryRoot := t.TempDir()
	dataPath := filepath.Join(repositoryRoot, ".aether", "data")
	authority, err := storage.OpenRepositoryRoot(repositoryRoot, dataPath)
	if err != nil {
		t.Fatalf("OpenRepositoryRoot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := authority.Close(); err != nil {
			t.Errorf("close repository authority: %v", err)
		}
	})
	store, err := storage.NewRepositoryStore(authority)
	if err != nil {
		t.Fatalf("NewRepositoryStore() error = %v", err)
	}
	return NewSpawnTree(store, "spawn-tree.txt"), store, dataPath
}

func TestSpawnTreeRepositoryStoreFresh200(t *testing.T) {
	tree, store, _ := newRepositorySpawnTree200(t)

	entries, err := tree.Parse()
	if err != nil {
		t.Fatalf("Parse() on absent repository ledger error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Parse() on absent repository ledger returned %d entries, want 0", len(entries))
	}

	if err := tree.RecordSpawn("Queen", "builder", "Mason-45", "first repository spawn", 1); err != nil {
		t.Fatalf("RecordSpawn() first repository append error = %v", err)
	}
	replayed := NewSpawnTree(store, "spawn-tree.txt")
	entries, err = replayed.Parse()
	if err != nil {
		t.Fatalf("Parse() replay error = %v", err)
	}
	if len(entries) != 1 || entries[0].AgentName != "Mason-45" {
		t.Fatalf("Parse() replay = %#v, want the first repository spawn", entries)
	}
}

func TestSpawnTreeRepositoryStoreCorrupt200(t *testing.T) {
	t.Run("corrupt spawn ledger is fingerprinted and never overwritten", func(t *testing.T) {
		tree, store, dataPath := newRepositorySpawnTree200(t)
		corrupt := []byte("CORRUPT-SPAWN-LEDGER-200\n")
		if err := store.AtomicWrite("spawn-tree.txt", corrupt); err != nil {
			t.Fatalf("write corrupt spawn ledger: %v", err)
		}
		before := sha256.Sum256(corrupt)

		if _, err := tree.Parse(); !errors.Is(err, ErrSpawnTreeCorrupt) {
			t.Fatalf("Parse() error = %v, want ErrSpawnTreeCorrupt", err)
		}
		if err := tree.RecordSpawn("Queen", "builder", "Mason-45", "must not overwrite", 1); !errors.Is(err, ErrSpawnTreeCorrupt) {
			t.Fatalf("RecordSpawn() error = %v, want ErrSpawnTreeCorrupt", err)
		}
		afterBytes, err := os.ReadFile(filepath.Join(dataPath, "spawn-tree.txt"))
		if err != nil {
			t.Fatalf("read corrupt spawn ledger after refusal: %v", err)
		}
		if after := sha256.Sum256(afterBytes); after != before {
			t.Fatalf("corrupt spawn ledger fingerprint changed: before=%x after=%x", before, after)
		}
	})

	t.Run("linked spawn ledger remains linked and is never followed", func(t *testing.T) {
		tree, _, dataPath := newRepositorySpawnTree200(t)
		targetPath := filepath.Join(filepath.Dir(dataPath), "linked-target.txt")
		targetBytes := []byte("LINK-TARGET-CANARY-200\n")
		if err := os.WriteFile(targetPath, targetBytes, 0644); err != nil {
			t.Fatalf("write linked target: %v", err)
		}
		if err := os.Symlink(targetPath, filepath.Join(dataPath, "spawn-tree.txt")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		before := sha256.Sum256(targetBytes)

		if _, err := tree.Parse(); err == nil || errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("Parse() linked-ledger error = %v, want non-absence refusal", err)
		}
		if err := tree.RecordSpawn("Queen", "builder", "Mason-45", "must not follow", 1); err == nil || errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("RecordSpawn() linked-ledger error = %v, want non-absence refusal", err)
		}
		info, err := os.Lstat(filepath.Join(dataPath, "spawn-tree.txt"))
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("linked ledger was replaced: info=%v err=%v", info, err)
		}
		afterBytes, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("read linked target after refusal: %v", err)
		}
		if after := sha256.Sum256(afterBytes); after != before {
			t.Fatalf("linked target fingerprint changed: before=%x after=%x", before, after)
		}
	})

	t.Run("unreadable spawn ledger is not absence", func(t *testing.T) {
		tree, _, dataPath := newRepositorySpawnTree200(t)
		ledgerPath := filepath.Join(dataPath, "spawn-tree.txt")
		if err := os.WriteFile(ledgerPath, []byte("2026-09-09T00:00:00Z|Queen|builder|Mason-45|task|1|spawned\n"), 0600); err != nil {
			t.Fatalf("write unreadable spawn ledger: %v", err)
		}
		if err := os.Chmod(ledgerPath, 0000); err != nil {
			t.Fatalf("chmod unreadable spawn ledger: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(ledgerPath, 0600) })

		_, err := tree.Parse()
		if err == nil {
			t.Skip("permission denial could not be induced for this process")
		}
		if errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("unreadable spawn ledger misclassified as absence: %v", err)
		}
	})

	t.Run("corrupt run ledger is fingerprinted and never overwritten", func(t *testing.T) {
		tree, _, dataPath := newRepositorySpawnTree200(t)
		runPath := filepath.Join(dataPath, defaultSpawnRunFile)
		corrupt := []byte("CORRUPT-RUN-LEDGER-200\n")
		if err := os.WriteFile(runPath, corrupt, 0644); err != nil {
			t.Fatalf("write corrupt run ledger: %v", err)
		}
		before := sha256.Sum256(corrupt)

		if _, _, err := tree.CurrentRun(); err == nil || errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("CurrentRun() corrupt-ledger error = %v, want non-absence refusal", err)
		}
		if _, err := tree.BeginRun("build", time.Now()); err == nil || errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("BeginRun() corrupt-ledger error = %v, want non-absence refusal", err)
		}
		afterBytes, err := os.ReadFile(runPath)
		if err != nil {
			t.Fatalf("read corrupt run ledger after refusal: %v", err)
		}
		if after := sha256.Sum256(afterBytes); after != before {
			t.Fatalf("corrupt run ledger fingerprint changed: before=%x after=%x", before, after)
		}
	})
}

func TestSpawnTreeRepositoryRunLedgerFresh200(t *testing.T) {
	tree, store, _ := newRepositorySpawnTree200(t)
	if _, ok, err := tree.CurrentRun(); err != nil || ok {
		t.Fatalf("CurrentRun() on absent repository ledger = (_, %v, %v), want (_, false, nil)", ok, err)
	}

	startedAt := time.Date(2026, time.September, 9, 10, 0, 0, 0, time.UTC)
	run, err := tree.BeginRun("build", startedAt)
	if err != nil {
		t.Fatalf("BeginRun() first repository run error = %v", err)
	}
	replayed := NewSpawnTree(store, "spawn-tree.txt")
	got, ok, err := replayed.CurrentRun()
	if err != nil {
		t.Fatalf("CurrentRun() replay error = %v", err)
	}
	if !ok || got.ID != run.ID || got.Command != "build" || got.Status != spawnRunStatusActive {
		t.Fatalf("CurrentRun() replay = (%#v, %v), want active run %#v", got, ok, run)
	}
}

func TestSpawnTreeRecordSpawn(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")

	err = st.RecordSpawn("colony-prime", "builder", "worker-1", "build phase 1", 1)
	if err != nil {
		t.Fatalf("RecordSpawn() error: %v", err)
	}

	entries := st.entries
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].AgentName != "worker-1" {
		t.Errorf("AgentName = %q, want %q", entries[0].AgentName, "worker-1")
	}
	if entries[0].ParentName != "colony-prime" {
		t.Errorf("ParentName = %q, want %q", entries[0].ParentName, "colony-prime")
	}
	if entries[0].Caste != "builder" {
		t.Errorf("Caste = %q, want %q", entries[0].Caste, "builder")
	}
	if entries[0].Task != "build phase 1" {
		t.Errorf("Task = %q, want %q", entries[0].Task, "build phase 1")
	}
	if entries[0].Depth != 1 {
		t.Errorf("Depth = %d, want 1", entries[0].Depth)
	}
	if entries[0].Status != "spawned" {
		t.Errorf("Status = %q, want %q", entries[0].Status, "spawned")
	}
	if entries[0].Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

func TestSpawnTreeUpdateStatus(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	st.RecordSpawn("colony-prime", "builder", "worker-1", "build task", 1)

	err = st.UpdateStatus("worker-1", "completed", "")
	if err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	if st.entries[0].Status != "completed" {
		t.Errorf("Status = %q, want %q", st.entries[0].Status, "completed")
	}

	if len(st.completions) != 1 {
		t.Fatalf("expected 1 completion line, got %d", len(st.completions))
	}
	if st.completions[0].Name != "worker-1" {
		t.Errorf("completion name = %q, want %q", st.completions[0].Name, "worker-1")
	}
	if st.completions[0].Status != "completed" {
		t.Errorf("completion status = %q, want %q", st.completions[0].Status, "completed")
	}

	// Test updating non-existent agent
	err = st.UpdateStatus("nonexistent", "failed", "")
	if err == nil {
		t.Error("expected error for non-existent agent")
	}
}

func TestSpawnTreeUpdateStatusTargetsMostRecentDuplicateName(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	st.RecordSpawn("colony-prime", "builder", "worker-1", "first task", 1)
	st.RecordSpawn("colony-prime", "builder", "worker-1", "second task", 1)

	if err := st.UpdateStatus("worker-1", "completed", "latest run"); err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	if st.entries[0].Status != "spawned" {
		t.Fatalf("first duplicate status = %q, want spawned", st.entries[0].Status)
	}
	if st.entries[1].Status != "completed" {
		t.Fatalf("last duplicate status = %q, want completed", st.entries[1].Status)
	}
}

func TestSpawnTreeUpdateStatusPreservesEntriesFromStaleInstance(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	stale := NewSpawnTree(store, "spawn-tree.txt")
	if err := stale.RecordSpawn("colony-prime", "builder", "worker-1", "first task", 1); err != nil {
		t.Fatalf("RecordSpawn(worker-1): %v", err)
	}

	fresh := NewSpawnTree(store, "spawn-tree.txt")
	if err := fresh.RecordSpawn("colony-prime", "watcher", "worker-2", "second task", 1); err != nil {
		t.Fatalf("RecordSpawn(worker-2): %v", err)
	}

	if err := stale.UpdateStatus("worker-1", "completed", "done"); err != nil {
		t.Fatalf("UpdateStatus(worker-1): %v", err)
	}

	reloaded := NewSpawnTree(store, "spawn-tree.txt")
	entries, err := reloaded.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Parse() returned %d entries, want 2", len(entries))
	}

	byName := make(map[string]SpawnEntry, len(entries))
	for _, entry := range entries {
		byName[entry.AgentName] = entry
	}
	if byName["worker-1"].Status != "completed" {
		t.Fatalf("worker-1 status = %q, want completed", byName["worker-1"].Status)
	}
	if byName["worker-2"].Status != "spawned" {
		t.Fatalf("worker-2 status = %q, want spawned", byName["worker-2"].Status)
	}
}

func TestSpawnTreeParseActiveStatusRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("colony-prime", "builder", "worker-1", "build task", 1); err != nil {
		t.Fatalf("RecordSpawn() error: %v", err)
	}
	if err := st.UpdateStatus("worker-1", "active", "Running"); err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	reloaded := NewSpawnTree(store, "spawn-tree.txt")
	entries, err := reloaded.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Parse() returned %d entries, want 1", len(entries))
	}
	if entries[0].Status != "active" {
		t.Fatalf("entries[0].Status = %q, want active", entries[0].Status)
	}
	if entries[0].Summary != "Running" {
		t.Fatalf("entries[0].Summary = %q, want %q", entries[0].Summary, "Running")
	}
}

func TestSpawnTreeActiveFiltersToCurrentRun(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	now := time.Now().UTC()
	runOne, err := st.BeginRun("build", now.Add(-2*time.Minute))
	if err != nil {
		t.Fatalf("BeginRun() error: %v", err)
	}
	if err := st.RecordSpawn("Queen", "builder", "Ghost-41", "Old worker", 1); err != nil {
		t.Fatalf("RecordSpawn(old) error: %v", err)
	}
	st.entries[len(st.entries)-1].Timestamp = now.Add(-110 * time.Second).Format(time.RFC3339)
	if err := st.UpdateStatus("Ghost-41", "active", "Still looks live"); err != nil {
		t.Fatalf("UpdateStatus(old) error: %v", err)
	}
	st.completions[len(st.completions)-1].Timestamp = now.Add(-100 * time.Second).Format(time.RFC3339)
	if err := st.Persist(); err != nil {
		t.Fatalf("Persist(old) error: %v", err)
	}
	if err := st.EndRun(runOne.ID, "completed", now.Add(-90*time.Second)); err != nil {
		t.Fatalf("EndRun(old) error: %v", err)
	}

	runTwo, err := st.BeginRun("plan", now.Add(-30*time.Second))
	if err != nil {
		t.Fatalf("BeginRun(second) error: %v", err)
	}
	if err := st.RecordSpawn("Queen", "scout", "Scout-7", "Current worker", 1); err != nil {
		t.Fatalf("RecordSpawn(current) error: %v", err)
	}
	st.entries[len(st.entries)-1].Timestamp = now.Add(-10 * time.Second).Format(time.RFC3339)
	if err := st.UpdateStatus("Scout-7", "active", "Planning"); err != nil {
		t.Fatalf("UpdateStatus(current) error: %v", err)
	}
	st.completions[len(st.completions)-1].Timestamp = now.Add(-5 * time.Second).Format(time.RFC3339)
	if err := st.Persist(); err != nil {
		t.Fatalf("Persist(current) error: %v", err)
	}

	currentRun, ok, err := st.CurrentRun()
	if err != nil {
		t.Fatalf("CurrentRun() error: %v", err)
	}
	if !ok {
		t.Fatal("CurrentRun() returned ok=false, want true")
	}
	if currentRun.ID != runTwo.ID {
		t.Fatalf("current run = %q, want %q", currentRun.ID, runTwo.ID)
	}

	active := st.Active()
	if len(active) != 1 {
		t.Fatalf("Active() returned %d entries, want 1", len(active))
	}
	if active[0].AgentName != "Scout-7" {
		t.Fatalf("Active()[0].AgentName = %q, want %q", active[0].AgentName, "Scout-7")
	}

	entries, err := st.EntriesForRun(runOne.ID)
	if err != nil {
		t.Fatalf("EntriesForRun(old) error: %v", err)
	}
	if len(entries) != 1 || entries[0].AgentName != "Ghost-41" {
		t.Fatalf("EntriesForRun(old) = %#v, want Ghost-41 only", entries)
	}
}

func TestSpawnTreeFormat(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	st.RecordSpawn("colony-prime", "builder", "agent-name", "task", 1)

	// Read the raw file
	data, err := store.ReadFile("spawn-tree.txt")
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	line := strings.TrimSpace(string(data))

	// Must have exactly 6 pipe separators (7 fields)
	pipeCount := strings.Count(line, "|")
	if pipeCount != 6 {
		t.Errorf("line has %d pipe separators, want 6 (7 fields): %q", pipeCount, line)
	}

	// Verify field structure
	fields := strings.Split(line, "|")
	if len(fields) != 7 {
		t.Fatalf("line has %d fields, want 7: %q", len(fields), line)
	}
	if fields[1] != "colony-prime" {
		t.Errorf("field[1] (parent) = %q, want %q", fields[1], "colony-prime")
	}
	if fields[2] != "builder" {
		t.Errorf("field[2] (caste) = %q, want %q", fields[2], "builder")
	}
	if fields[3] != "agent-name" {
		t.Errorf("field[3] (name) = %q, want %q", fields[3], "agent-name")
	}
	if fields[4] != "task" {
		t.Errorf("field[4] (task) = %q, want %q", fields[4], "task")
	}
	if fields[5] != "1" {
		t.Errorf("field[5] (depth) = %q, want %q", fields[5], "1")
	}
	if fields[6] != "spawned" {
		t.Errorf("field[6] (status) = %q, want %q", fields[6], "spawned")
	}
}

func TestSpawnTreeParseRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")

	// Record multiple entries
	st.RecordSpawn("colony-prime", "builder", "worker-1", "build task", 1)
	st.RecordSpawn("colony-prime", "watcher", "worker-2", "watch task", 1)
	st.UpdateStatus("worker-1", "completed", "")
	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Parse() returned %d entries, want 2", len(entries))
	}

	// First entry should have status merged from completion line
	if entries[0].AgentName != "worker-1" {
		t.Errorf("entries[0].AgentName = %q, want %q", entries[0].AgentName, "worker-1")
	}
	if entries[0].Status != "completed" {
		t.Errorf("entries[0].Status = %q, want %q", entries[0].Status, "completed")
	}

	// Second entry should still be spawned
	if entries[1].AgentName != "worker-2" {
		t.Errorf("entries[1].AgentName = %q, want %q", entries[1].AgentName, "worker-2")
	}
	if entries[1].Status != "spawned" {
		t.Errorf("entries[1].Status = %q, want %q", entries[1].Status, "spawned")
	}
}

func TestSpawnTreeParseShellFormat(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	// Write shell-formatted content directly
	shellContent := `2026-04-01T12:00:00Z|colony-prime|builder|worker-1|build task|1|spawned
2026-04-01T12:00:01Z|colony-prime|watcher|worker-2|watch task|1|spawned
2026-04-01T12:05:00Z|worker-1|completed|
`
	store.AtomicWrite("spawn-tree.txt", []byte(shellContent))

	st := NewSpawnTree(store, "spawn-tree.txt")
	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Parse() returned %d entries, want 2", len(entries))
	}

	// First entry should have status merged from completion line
	if entries[0].AgentName != "worker-1" {
		t.Errorf("entries[0].AgentName = %q, want %q", entries[0].AgentName, "worker-1")
	}
	if entries[0].Status != "completed" {
		t.Errorf("entries[0].Status = %q, want completed (merged from completion line)", entries[0].Status)
	}
	if entries[0].Caste != "builder" {
		t.Errorf("entries[0].Caste = %q, want %q", entries[0].Caste, "builder")
	}
	if entries[0].Depth != 1 {
		t.Errorf("entries[0].Depth = %d, want 1", entries[0].Depth)
	}

	// Second entry should still be spawned
	if entries[1].AgentName != "worker-2" {
		t.Errorf("entries[1].AgentName = %q, want %q", entries[1].AgentName, "worker-2")
	}
	if entries[1].Status != "spawned" {
		t.Errorf("entries[1].Status = %q, want %q", entries[1].Status, "spawned")
	}
}

func TestSpawnTreeSanitizesReservedFieldCharacters(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("queen|prime", "builder", "worker\n1", "build | feature\nset", 1); err != nil {
		t.Fatalf("RecordSpawn() error: %v", err)
	}
	if err := st.UpdateStatus("worker\n1", "completed", "fixed | verified\ncleanly"); err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Parse() returned %d entries, want 1", len(entries))
	}

	entry := entries[0]
	if strings.Contains(entry.ParentName, "|") || strings.Contains(entry.AgentName, "\n") || strings.Contains(entry.Task, "|") || strings.Contains(entry.Summary, "\n") {
		t.Fatalf("reserved characters were not sanitized: %#v", entry)
	}
	if got, want := entry.ParentName, "queen¦prime"; got != want {
		t.Fatalf("ParentName = %q, want %q", got, want)
	}
	if got, want := entry.AgentName, "worker 1"; got != want {
		t.Fatalf("AgentName = %q, want %q", got, want)
	}
	if got, want := entry.Task, "build ¦ feature set"; got != want {
		t.Fatalf("Task = %q, want %q", got, want)
	}
	if got, want := entry.Summary, "fixed ¦ verified cleanly"; got != want {
		t.Fatalf("Summary = %q, want %q", got, want)
	}
}

func TestSpawnTreeActive(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	st.RecordSpawn("colony-prime", "builder", "worker-1", "build", 1)
	st.RecordSpawn("colony-prime", "builder", "worker-2", "build", 1)
	st.RecordSpawn("colony-prime", "watcher", "worker-3", "watch", 1)
	st.UpdateStatus("worker-1", "completed", "")

	active := st.Active()
	if len(active) != 2 {
		t.Fatalf("Active() returned %d entries, want 2", len(active))
	}

	names := make(map[string]bool)
	for _, e := range active {
		names[e.AgentName] = true
		if e.Status != "spawned" {
			t.Errorf("Active() entry %q has status %q, want %q", e.AgentName, e.Status, "spawned")
		}
	}
	if !names["worker-2"] || !names["worker-3"] {
		t.Errorf("Active() returned agents %v, want worker-2 and worker-3", names)
	}
}

func TestSpawnTreeToJSON(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	st.RecordSpawn("colony-prime", "builder", "worker-1", "build task", 1)
	st.RecordSpawn("colony-prime", "watcher", "worker-2", "watch task", 1)
	st.UpdateStatus("worker-1", "completed", "")

	data, err := st.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}

	// Verify top-level keys
	if _, ok := result["spawns"]; !ok {
		t.Error("JSON missing 'spawns' key")
	}
	if _, ok := result["metadata"]; !ok {
		t.Error("JSON missing 'metadata' key")
	}

	// Verify metadata
	var metadata struct {
		TotalCount     int `json:"total_count"`
		ActiveCount    int `json:"active_count"`
		CompletedCount int `json:"completed_count"`
	}
	if err := json.Unmarshal(result["metadata"], &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if metadata.TotalCount != 2 {
		t.Errorf("total_count = %d, want 2", metadata.TotalCount)
	}
	if metadata.ActiveCount != 1 {
		t.Errorf("active_count = %d, want 1", metadata.ActiveCount)
	}
	if metadata.CompletedCount != 1 {
		t.Errorf("completed_count = %d, want 1", metadata.CompletedCount)
	}

	// Verify spawns array
	var spawns []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(result["spawns"], &spawns); err != nil {
		t.Fatalf("unmarshal spawns: %v", err)
	}
	if len(spawns) != 2 {
		t.Fatalf("spawns has %d entries, want 2", len(spawns))
	}
}

func TestSpawnTreeUpdateStatusWithSummary(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	st.RecordSpawn("colony-prime", "builder", "worker-1", "build task", 1)

	err = st.UpdateStatus("worker-1", "completed", "Task completed successfully")
	if err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	if st.entries[0].Status != "completed" {
		t.Errorf("Status = %q, want %q", st.entries[0].Status, "completed")
	}

	if len(st.completions) != 1 {
		t.Fatalf("expected 1 completion line, got %d", len(st.completions))
	}
	if st.completions[0].Name != "worker-1" {
		t.Errorf("completion name = %q, want %q", st.completions[0].Name, "worker-1")
	}
	if st.completions[0].Status != "completed" {
		t.Errorf("completion status = %q, want %q", st.completions[0].Status, "completed")
	}
	if st.completions[0].Summary != "Task completed successfully" {
		t.Errorf("completion summary = %q, want %q", st.completions[0].Summary, "Task completed successfully")
	}

	// Verify the summary is persisted to the entry as well
	if st.entries[0].Summary != "Task completed successfully" {
		t.Errorf("entry summary = %q, want %q", st.entries[0].Summary, "Task completed successfully")
	}

	// Verify round-trip: re-parse the file and check summary
	st2 := NewSpawnTree(store, "spawn-tree.txt")
	entries, err := st2.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after re-parse, got %d", len(entries))
	}
	if entries[0].Summary != "Task completed successfully" {
		t.Errorf("re-parsed entry summary = %q, want %q", entries[0].Summary, "Task completed successfully")
	}
}

func TestSpawnTreeUpdateStatusEmptySummary(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	st := NewSpawnTree(store, "spawn-tree.txt")
	st.RecordSpawn("colony-prime", "builder", "worker-1", "build task", 1)

	// Call with empty summary (backward compatibility)
	err = st.UpdateStatus("worker-1", "completed", "")
	if err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	if st.completions[0].Summary != "" {
		t.Errorf("completion summary = %q, want empty string", st.completions[0].Summary)
	}

	// Verify the file can still be parsed by a new SpawnTree instance
	st2 := NewSpawnTree(store, "spawn-tree.txt")
	entries, err := st2.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Status != "completed" {
		t.Errorf("re-parsed status = %q, want completed", entries[0].Status)
	}
}

func TestSpawnTreeParseOldFormatCompletion(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	// Write old-format completion line (no summary field): timestamp|name|status|
	shellContent := `2026-04-01T12:00:00Z|colony-prime|builder|worker-1|build task|1|spawned
2026-04-01T12:05:00Z|worker-1|completed|
`
	store.AtomicWrite("spawn-tree.txt", []byte(shellContent))

	st := NewSpawnTree(store, "spawn-tree.txt")
	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Parse() returned %d entries, want 1", len(entries))
	}
	if entries[0].Status != "completed" {
		t.Errorf("Status = %q, want completed", entries[0].Status)
	}
	if entries[0].Summary != "" {
		t.Errorf("Summary = %q, want empty for old format", entries[0].Summary)
	}
}

func TestSpawnTreeEmpty(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	// No file exists -- should not error
	st := NewSpawnTree(store, "spawn-tree.txt")

	if len(st.entries) != 0 {
		t.Errorf("expected 0 entries for missing file, got %d", len(st.entries))
	}

	active := st.Active()
	if len(active) != 0 {
		t.Errorf("Active() returned %d entries, want 0", len(active))
	}

	// ToJSON on empty tree
	data, err := st.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	var result struct {
		Spawns   []interface{} `json:"spawns"`
		Metadata struct {
			TotalCount     int `json:"total_count"`
			ActiveCount    int `json:"active_count"`
			CompletedCount int `json:"completed_count"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal empty JSON: %v", err)
	}
	if result.Metadata.TotalCount != 0 {
		t.Errorf("empty total_count = %d, want 0", result.Metadata.TotalCount)
	}
}

// TestSpawnStatusAbandonedIsTerminalNotLive is plan 09's Task 1 proof
// (SPAWN-08/D-15..D-18): the reaper's "abandoned" status must stop counting
// as live work the moment it is applied, and must never be mistaken for
// in-flight work by any code that branches on IsLiveSpawnStatus.
func TestSpawnStatusAbandonedIsTerminalNotLive(t *testing.T) {
	if !IsTerminalSpawnStatus(SpawnStatusAbandoned) {
		t.Errorf("IsTerminalSpawnStatus(%q) = false, want true", SpawnStatusAbandoned)
	}
	if IsLiveSpawnStatus(SpawnStatusAbandoned) {
		t.Errorf("IsLiveSpawnStatus(%q) = true, want false", SpawnStatusAbandoned)
	}
	// Mixed-case / whitespace input must normalize the same way every other
	// status already does (normalizeSpawnStatus's generic lowercase+trim
	// path), not via a special-cased comparison that could drift.
	if !IsTerminalSpawnStatus("  Abandoned ") {
		t.Errorf("IsTerminalSpawnStatus(mixed-case/whitespace) = false, want true")
	}
}

// TestSpawnTreeParseTreatsAnAbsentLedgerAsEmptyButACorruptOneAsAnError is
// plan 11's Task 1 red-proof (T-173-50/T-173-52): a spawn ledger that has
// never been written, a zero-byte one, and a whitespace-only one must all
// parse as an empty ledger with a nil error -- the T-173-24 exception a
// fresh colony depends on to spawn at all. A ledger with content that is
// not valid spawn-tree pipe format must instead return a non-nil error
// wrapping ErrSpawnTreeCorrupt, naming the file and the offending line
// number, and MUST NOT contain the offending line's own text -- a
// spawn-tree line carries worker task text, and that must never leak
// through an error string.
func TestSpawnTreeParseTreatsAnAbsentLedgerAsEmptyButACorruptOneAsAnError(t *testing.T) {
	cases := []struct {
		name      string
		writeFile bool
		content   string
		wantErr   bool
	}{
		{name: "absent", writeFile: false},
		{name: "zero-byte", writeFile: true, content: ""},
		{name: "whitespace-only", writeFile: true, content: "   \n\n\t\n  "},
		{name: "corrupt", writeFile: true, content: "garbage not pipe format LEDGER-CONTENT-CANARY", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			store, err := storage.NewStore(dir)
			if err != nil {
				t.Fatalf("create store: %v", err)
			}
			if tc.writeFile {
				if err := store.AtomicWrite("spawn-tree.txt", []byte(tc.content)); err != nil {
					t.Fatalf("write fixture: %v", err)
				}
			}

			st := NewSpawnTree(store, "spawn-tree.txt")
			entries, err := st.Parse()

			if !tc.wantErr {
				if err != nil {
					t.Fatalf("Parse() error = %v, want nil", err)
				}
				if len(entries) != 0 {
					t.Fatalf("Parse() returned %d entries, want 0", len(entries))
				}
				return
			}

			if err == nil {
				t.Fatal("Parse() error = nil, want a corruption error")
			}
			if !errors.Is(err, ErrSpawnTreeCorrupt) {
				t.Fatalf("errors.Is(err, ErrSpawnTreeCorrupt) = false for err: %v", err)
			}
			if !strings.Contains(err.Error(), "spawn-tree.txt") {
				t.Errorf("error %q does not name the file", err.Error())
			}
			if !strings.Contains(err.Error(), "line 1") {
				t.Errorf("error %q does not name the line number", err.Error())
			}
			if strings.Contains(err.Error(), "LEDGER-CONTENT-CANARY") {
				t.Errorf("error %q leaks the offending line's own content", err.Error())
			}
		})
	}
}

// TestSpawnTreeParseAcceptsEveryShapeTheWriterProduces is plan 11's Task 1
// red-proof (T-173-66): the negative control that stops an always-error
// parser from passing, and the guard against the parser rejecting bytes its
// own writer produced.
func TestSpawnTreeParseAcceptsEveryShapeTheWriterProduces(t *testing.T) {
	t.Run("valid entries, completions, legacy empty-summary and blank lines all parse cleanly", func(t *testing.T) {
		dir := t.TempDir()
		store, err := storage.NewStore(dir)
		if err != nil {
			t.Fatalf("create store: %v", err)
		}

		content := strings.Join([]string{
			"2026-04-01T12:00:00Z|colony-prime|builder|worker-1|build task|1|spawned",
			"2026-04-01T12:00:01Z|colony-prime|watcher|worker-2|watch task|1|spawned",
			"",
			"2026-04-01T12:05:00Z|worker-1|completed|a summary",
			"2026-04-01T12:06:00Z|worker-2|completed|",
		}, "\n") + "\n"
		if err := store.AtomicWrite("spawn-tree.txt", []byte(content)); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		st := NewSpawnTree(store, "spawn-tree.txt")
		entries, err := st.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v, want nil", err)
		}
		if len(entries) != 2 {
			t.Fatalf("Parse() returned %d entries, want 2", len(entries))
		}
		if entries[0].Status != "completed" {
			t.Errorf("entries[0].Status = %q, want completed (merged from completion line)", entries[0].Status)
		}
		if entries[0].Summary != "a summary" {
			t.Errorf("entries[0].Summary = %q, want %q", entries[0].Summary, "a summary")
		}
		if entries[1].Status != "completed" {
			t.Errorf("entries[1].Status = %q, want completed (merged from legacy empty-summary completion line)", entries[1].Status)
		}
	})

	t.Run("writer round-trip never gets rejected by the guard it feeds", func(t *testing.T) {
		entries := []SpawnEntry{
			{
				Timestamp:  "2026-04-01T12:00:00Z",
				ParentName: "colony-prime",
				Caste:      "builder",
				AgentName:  "worker-1",
				Task:       "",
				Depth:      1,
				Status:     "",
			},
		}
		completions := []completionLine{
			{
				Timestamp: "2026-04-01T12:05:00Z",
				Name:      "worker-1",
				Status:    "",
				Summary:   "",
			},
		}

		data := formatSpawnTreeLines(entries, completions)
		parsedEntries, _, err := parseSpawnTreeBytes(data)
		if err != nil {
			t.Fatalf("parseSpawnTreeBytes(writer output) error = %v, want nil -- a corruption guard must never reject bytes its own writer emits", err)
		}
		if len(parsedEntries) != 1 {
			t.Fatalf("parseSpawnTreeBytes(writer output) returned %d entries, want 1", len(parsedEntries))
		}

		emptyData := formatSpawnTreeLines(nil, nil)
		emptyEntries, _, err := parseSpawnTreeBytes(emptyData)
		if err != nil {
			t.Fatalf("parseSpawnTreeBytes(empty tree) error = %v, want nil", err)
		}
		if len(emptyEntries) != 0 {
			t.Fatalf("parseSpawnTreeBytes(empty tree) returned %d entries, want 0", len(emptyEntries))
		}
	})

	t.Run("a non-numeric depth field is corrupt, never a bogus completion", func(t *testing.T) {
		content := "2026-04-01T12:00:00Z|colony-prime|builder|worker-1|build task|not-a-number|spawned\n"
		_, _, err := parseSpawnTreeBytes([]byte(content))
		if err == nil {
			t.Fatal("parseSpawnTreeBytes() error = nil, want a corruption error for a non-numeric depth field")
		}
		if !errors.Is(err, ErrSpawnTreeCorrupt) {
			t.Fatalf("errors.Is(err, ErrSpawnTreeCorrupt) = false for err: %v", err)
		}
		if !strings.Contains(err.Error(), "line 1") {
			t.Errorf("error %q does not name the line number", err.Error())
		}
	})
}

// TestSpawnTreeRefusesToRewriteACorruptLedger is plan 11's Task 1 red-proof
// (T-173-51): recording a spawn or a status update onto a corrupt ledger
// must fail instead of silently discarding the corrupt line and writing a
// clean file over it -- the second half of 173-VERIFICATION.md's
// reproduction, where the recorder replaced tampered bytes with a fresh
// file carrying a single new entry.
func TestSpawnTreeRefusesToRewriteACorruptLedger(t *testing.T) {
	corruptContent := []byte("garbage not pipe format\n")

	t.Run("RecordSpawn", func(t *testing.T) {
		dir := t.TempDir()
		store, err := storage.NewStore(dir)
		if err != nil {
			t.Fatalf("create store: %v", err)
		}
		if err := store.AtomicWrite("spawn-tree.txt", corruptContent); err != nil {
			t.Fatalf("write corrupt fixture: %v", err)
		}
		treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
		before, err := os.ReadFile(treePath)
		if err != nil {
			t.Fatalf("read fixture before RecordSpawn: %v", err)
		}

		st := NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn("colony-prime", "builder", "worker-1", "build task", 1); err == nil {
			t.Fatal("RecordSpawn() error = nil, want non-nil against a corrupt ledger")
		}

		after, err := os.ReadFile(treePath)
		if err != nil {
			t.Fatalf("read fixture after RecordSpawn: %v", err)
		}
		if string(before) != string(after) {
			t.Errorf("RecordSpawn() rewrote the corrupt ledger:\nbefore: %q\nafter:  %q", before, after)
		}
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		dir := t.TempDir()
		store, err := storage.NewStore(dir)
		if err != nil {
			t.Fatalf("create store: %v", err)
		}
		if err := store.AtomicWrite("spawn-tree.txt", corruptContent); err != nil {
			t.Fatalf("write corrupt fixture: %v", err)
		}
		treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
		before, err := os.ReadFile(treePath)
		if err != nil {
			t.Fatalf("read fixture before UpdateStatus: %v", err)
		}

		st := NewSpawnTree(store, "spawn-tree.txt")
		if err := st.UpdateStatus("worker-1", "completed", ""); err == nil {
			t.Fatal("UpdateStatus() error = nil, want non-nil against a corrupt ledger")
		}

		after, err := os.ReadFile(treePath)
		if err != nil {
			t.Fatalf("read fixture after UpdateStatus: %v", err)
		}
		if string(before) != string(after) {
			t.Errorf("UpdateStatus() rewrote the corrupt ledger:\nbefore: %q\nafter:  %q", before, after)
		}
	})
}

// TestSpawnRunStateTellsAnAbsentRunFileApartFromAnUnreadableOne is plan 11's
// Task 1 red-proof (T-173-69): spawn-runs.json draws the same three-way
// distinction as spawn-tree.txt. An absent file must still mean "no run has
// begun yet" with a nil error, because BeginRun must be able to create the
// very first one for a fresh colony. A directory obstructing the path, or
// content that fails to unmarshal as JSON, must both be errors instead of
// an empty run history.
func TestSpawnRunStateTellsAnAbsentRunFileApartFromAnUnreadableOne(t *testing.T) {
	t.Run("absent run-state file is empty with a nil error, and BeginRun can still create the first run", func(t *testing.T) {
		dir := t.TempDir()
		store, err := storage.NewStore(dir)
		if err != nil {
			t.Fatalf("create store: %v", err)
		}

		st := NewSpawnTree(store, "spawn-tree.txt")
		_, ok, err := st.CurrentRun()
		if err != nil {
			t.Fatalf("CurrentRun() error = %v, want nil for an absent run-state file", err)
		}
		if ok {
			t.Fatal("CurrentRun() ok = true, want false for an absent run-state file")
		}

		if _, err := st.BeginRun("test", time.Now()); err != nil {
			t.Fatalf("BeginRun() error = %v, want nil -- a fresh colony must be able to start its first run", err)
		}
	})

	t.Run("a directory obstructing spawn-runs.json's path is an error, not an empty history", func(t *testing.T) {
		dir := t.TempDir()
		store, err := storage.NewStore(dir)
		if err != nil {
			t.Fatalf("create store: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "spawn-runs.json"), 0755); err != nil {
			t.Fatalf("create obstructing directory: %v", err)
		}

		st := NewSpawnTree(store, "spawn-tree.txt")
		if _, _, err := st.CurrentRun(); err == nil {
			t.Fatal("CurrentRun() error = nil, want non-nil when spawn-runs.json's path is obstructed by a directory")
		}
	})

	t.Run("invalid JSON in spawn-runs.json is still an error, exactly as before this change", func(t *testing.T) {
		dir := t.TempDir()
		store, err := storage.NewStore(dir)
		if err != nil {
			t.Fatalf("create store: %v", err)
		}
		// store.AtomicWrite validates JSON for .json paths and would refuse
		// to write invalid content, so this fixture is written directly.
		runFilePath := filepath.Join(dir, "spawn-runs.json")
		if err := os.WriteFile(runFilePath, []byte("not valid json"), 0644); err != nil {
			t.Fatalf("write invalid JSON fixture: %v", err)
		}

		st := NewSpawnTree(store, "spawn-tree.txt")
		if _, _, err := st.CurrentRun(); err == nil {
			t.Fatal("CurrentRun() error = nil, want non-nil for invalid JSON in spawn-runs.json")
		}
	})
}
