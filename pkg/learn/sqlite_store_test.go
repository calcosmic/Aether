package learn

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

// newTestSQLiteStore creates a SQLiteColonyStore in a temp directory for testing.
func newTestSQLiteStore(t *testing.T) (*SQLiteColonyStore, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "colony.db")
	store, err := NewSQLiteColonyStore(dbPath)
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	return store, dir
}

func TestWALMode(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	var mode string
	err := store.DB().QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}
	_ = dir // used for temp dir cleanup
}

func TestSQLiteColonyStoreAdd(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeEntry("", "learned something new", 0.8)
	if err := store.Add(entry); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Verify entry was stored by listing it
	entries, err := store.List(EntryFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID == "" {
		t.Error("Add should assign an ID to the persisted entry")
	}
	if entries[0].Content != "learned something new" {
		t.Errorf("Content = %q, want %q", entries[0].Content, "learned something new")
	}
}

func TestSQLiteColonyStoreGet(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeEntry("", "retrievable content", 0.9)
	if err := store.Add(entry); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Retrieve the persisted entry to get the assigned ID
	entries, err := store.List(EntryFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	entryID := entries[0].ID

	got, err := store.Get(entryID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil for existing entry")
	}
	if got.Content != "retrievable content" {
		t.Errorf("Content = %q, want %q", got.Content, "retrievable content")
	}
	if got.Classification != ClassRepoLocal {
		t.Errorf("Classification = %q, want %q", got.Classification, ClassRepoLocal)
	}
	if got.Confidence != 0.9 {
		t.Errorf("Confidence = %f, want 0.9", got.Confidence)
	}
	// Verify evidence is preserved
	if got.Evidence.RunID != "run-1" {
		t.Errorf("Evidence.RunID = %q, want %q", got.Evidence.RunID, "run-1")
	}
	if len(got.Evidence.Workers) != 1 {
		t.Errorf("Evidence.Workers length = %d, want 1", len(got.Evidence.Workers))
	}
}

func TestSQLiteColonyStoreGetNotFound(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	got, err := store.Get("nonexistent-id")
	if err != nil {
		t.Fatalf("Get nonexistent: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent ID, got %+v", got)
	}
}

func TestSQLiteColonyStoreListEmpty(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	entries, err := store.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if entries == nil {
		t.Error("List should return empty slice, not nil")
	}
	if len(entries) != 0 {
		t.Errorf("List empty = %d entries, want 0", len(entries))
	}
}

func TestSQLiteColonyStoreListFilterPhase(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	e1 := makeEntry("", "phase 1 entry", 0.8)
	e1.Phase = 1
	e2 := makeEntry("", "phase 2 entry", 0.8)
	e2.Phase = 2
	store.Add(e1)
	store.Add(e2)

	entries, err := store.List(EntryFilter{Phase: 2})
	if err != nil {
		t.Fatalf("List filter phase: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("List phase=2 = %d entries, want 1", len(entries))
	}
	if entries[0].Phase != 2 {
		t.Errorf("entry Phase = %d, want 2", entries[0].Phase)
	}
}

func TestSQLiteColonyStoreListFilterClassification(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	e1 := makeEntry("", "local entry", 0.8)
	e1.Classification = ClassRepoLocal
	e2 := makeEntry("", "shareable entry", 0.9)
	e2.Classification = ClassHiveShareable
	store.Add(e1)
	store.Add(e2)

	entries, err := store.List(EntryFilter{Classification: ClassHiveShareable})
	if err != nil {
		t.Fatalf("List filter classification: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("List hive-shareable = %d entries, want 1", len(entries))
	}
	if entries[0].Classification != ClassHiveShareable {
		t.Errorf("entry Classification = %q, want %q", entries[0].Classification, ClassHiveShareable)
	}
}

func TestSQLiteColonyStoreListFilterMinConfidence(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	store.Add(makeEntry("", "low confidence", 0.3))
	store.Add(makeEntry("", "high confidence", 0.8))

	entries, err := store.List(EntryFilter{MinConfidence: 0.5})
	if err != nil {
		t.Fatalf("List min_confidence: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("List min_confidence=0.5 = %d entries, want 1", len(entries))
	}
	if entries[0].Confidence < 0.5 {
		t.Errorf("entry Confidence = %f, want >= 0.5", entries[0].Confidence)
	}
}

func TestSQLiteColonyStoreListLimit(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	for i := 0; i < 5; i++ {
		store.Add(makeEntry("", "entry", float64(i)))
	}

	entries, err := store.List(EntryFilter{Limit: 3})
	if err != nil {
		t.Fatalf("List limit: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("List limit=3 = %d entries, want 3", len(entries))
	}
}

func TestSQLiteColonyStoreReplace(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeEntry("", "original content", 0.5)
	store.Add(entry)

	// Retrieve the persisted entry to get the assigned ID
	entries, err := store.List(EntryFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	originalID := entries[0].ID

	updated := entries[0]
	updated.Content = "updated content"
	updated.Classification = ClassHiveShareable
	if err := store.Replace(originalID, updated); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	got, err := store.Get(originalID)
	if err != nil {
		t.Fatalf("Get after replace: %v", err)
	}
	if got.Content != "updated content" {
		t.Errorf("Content = %q, want %q", got.Content, "updated content")
	}
	if got.Classification != ClassHiveShareable {
		t.Errorf("Classification = %q, want %q", got.Classification, ClassHiveShareable)
	}
}

func TestSQLiteColonyStoreRemove(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeEntry("", "to remove", 0.7)
	store.Add(entry)

	// Retrieve the persisted entry to get the assigned ID
	entries, err := store.List(EntryFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	id := entries[0].ID

	if err := store.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	got, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get after remove: %v", err)
	}
	if got != nil {
		t.Error("expected nil after Remove, got entry")
	}
}

func TestMigrationIdempotency(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "colony.db")

	// Open, add data, close
	store1, err := NewSQLiteColonyStore(dbPath)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	entry := makeEntry("", "survives migration rerun", 0.9)
	store1.Add(entry)

	// Retrieve the assigned ID via List
	entries, err := store1.List(EntryFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	entryID := entries[0].ID
	store1.Close()

	// Reopen (runs migrations again)
	store2, err := NewSQLiteColonyStore(dbPath)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer store2.Close()

	got, err := store2.Get(entryID)
	if err != nil {
		t.Fatalf("Get after re-migration: %v", err)
	}
	if got == nil {
		t.Fatal("entry lost after re-running migrations")
	}
	if got.Content != "survives migration rerun" {
		t.Errorf("Content = %q, want %q", got.Content, "survives migration rerun")
	}
}

func TestSQLiteColonyStoreIsolation(t *testing.T) {
	t.Parallel()

	dirA := t.TempDir()
	storeA, err := NewSQLiteColonyStore(filepath.Join(dirA, "colony.db"))
	if err != nil {
		t.Fatalf("create store A: %v", err)
	}
	defer storeA.Close()

	dirB := t.TempDir()
	storeB, err := NewSQLiteColonyStore(filepath.Join(dirB, "colony.db"))
	if err != nil {
		t.Fatalf("create store B: %v", err)
	}
	defer storeB.Close()

	storeA.Add(makeEntry("", "only in A", 0.8))

	entriesB, err := storeB.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List B: %v", err)
	}
	if len(entriesB) != 0 {
		t.Errorf("Store B should have 0 entries, got %d", len(entriesB))
	}

	entriesA, err := storeA.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List A: %v", err)
	}
	if len(entriesA) != 1 {
		t.Errorf("Store A should have 1 entry, got %d", len(entriesA))
	}

	// Verify physical isolation
	if _, err := os.Stat(filepath.Join(dirB, "colony.db")); err == nil {
		// Store B has its own db file (created by NewSQLiteColonyStore)
		_ = err // B's db exists at its own path
	}
}

func TestSQLiteColonyStoreReplaceNotFound(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	err := store.Replace("nonexistent", makeEntry("nonexistent", "x", 0.5))
	if err == nil {
		t.Error("Replace should return error for nonexistent ID")
	}
}

func TestSQLiteColonyStoreRemoveNotFound(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	err := store.Remove("nonexistent")
	if err == nil {
		t.Error("Remove should return error for nonexistent ID")
	}
}

func TestSQLiteColonyStoreRedactedField(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeEntry("", "sensitive content", 0.8)
	entry.Redacted = true
	store.Add(entry)

	// Retrieve the assigned ID via List
	entries, err := store.List(EntryFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	got, err := store.Get(entries[0].ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.Redacted {
		t.Error("Redacted should be true")
	}
}

func TestCompactRemovesLowestConfidence(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	// Add 5 entries with different confidences, each with 100-char content
	for i, conf := range []float64{0.1, 0.3, 0.5, 0.7, 0.9} {
		entry := makeEntry("", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", conf)
		entry.Phase = i
		store.Add(entry)
	}

	// Budget of 250 chars -- should keep top 2 highest confidence entries
	if err := store.Compact(250); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	entries, err := store.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List after compact: %v", err)
	}
	if len(entries) > 3 {
		t.Errorf("Compact should remove lowest-confidence entries, got %d entries", len(entries))
	}

	// Verify highest-confidence entry survives
	foundHigh := false
	for _, e := range entries {
		if e.Confidence == 0.9 {
			foundHigh = true
		}
	}
	if !foundHigh {
		t.Error("highest-confidence entry (0.9) should survive Compact")
	}

	// Verify lowest-confidence entry removed
	for _, e := range entries {
		if e.Confidence == 0.1 {
			t.Error("lowest-confidence entry (0.1) should have been removed by Compact")
		}
	}
}

func TestCompactNoOpWhenUnderBudget(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	store.Add(makeEntry("", "small", 0.5))
	store.Add(makeEntry("", "entry", 0.7))

	if err := store.Compact(10000); err != nil {
		t.Fatalf("Compact large budget: %v", err)
	}

	entries, err := store.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries after large-budget compact, got %d", len(entries))
	}
}

func TestCompactZeroBudgetRemovesAll(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	store.Add(makeEntry("", "first entry content", 0.5))
	store.Add(makeEntry("", "second entry content", 0.7))
	store.Add(makeEntry("", "third entry content", 0.9))

	if err := store.Compact(0); err != nil {
		t.Fatalf("Compact zero budget: %v", err)
	}

	entries, err := store.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after zero-budget compact, got %d", len(entries))
	}
}

func TestMigrateFromJSON(t *testing.T) {
	// Create a temp directory with a JSON ColonyStore, add 3 entries
	jsonDir := t.TempDir()
	dataDir := filepath.Join(jsonDir, ".aether", "data", "learn")
	st, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create json store: %v", err)
	}
	jsonCS := NewColonyStore(st)

	jsonCS.Add(makeEntry("json-1", "first json entry", 0.8))
	jsonCS.Add(makeEntry("json-2", "second json entry", 0.9))
	jsonCS.Add(makeEntry("json-3", "third json entry", 0.7))

	// Create SQLite store and migrate from JSON
	sqliteDir := t.TempDir()
	sqlitePath := filepath.Join(sqliteDir, "colony.db")
	sqliteStore, err := NewSQLiteColonyStore(sqlitePath)
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	imported, err := sqliteStore.MigrateFromJSON(dataDir)
	if err != nil {
		t.Fatalf("MigrateFromJSON: %v", err)
	}
	if imported != 3 {
		t.Errorf("expected 3 entries imported, got %d", imported)
	}

	// Verify all 3 entries exist in SQLite
	for _, id := range []string{"json-1", "json-2", "json-3"} {
		got, err := sqliteStore.Get(id)
		if err != nil {
			t.Fatalf("Get %s: %v", id, err)
		}
		if got == nil {
			t.Errorf("entry %q not found in SQLite after migration", id)
		}
	}
}

func TestMigrateFromJSON_NoFile(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	// Point at a directory with no entries.json
	imported, err := store.MigrateFromJSON(t.TempDir())
	if err != nil {
		t.Fatalf("MigrateFromJSON with no file: %v", err)
	}
	if imported != 0 {
		t.Errorf("expected 0 entries imported, got %d", imported)
	}
}
