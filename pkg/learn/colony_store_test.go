package learn

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

func newTestColonyStore(t *testing.T) (*ColonyStore, *storage.Store, string) {
	t.Helper()
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data", "learn")
	store, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	cs := NewColonyStore(store)
	return cs, store, dataDir
}

func makeEntry(id, content string, confidence float64) Entry {
	return Entry{
		ID:             id,
		Content:        content,
		Classification: ClassRepoLocal,
		CreatedAt:      "2026-05-01T00:00:00Z",
		Phase:          1,
		Confidence:     confidence,
		Evidence: Evidence{
			RunID:   "run-1",
			Phase:   1,
			Workers: []WorkerEvidence{{Name: "Builder-1", Caste: "builder", Status: "done"}},
			GatesPassed: 3,
			GatesTotal:  3,
			Confidence:  confidence,
			Timestamp:   "2026-05-01T00:00:00Z",
			Scope:       "repo",
		},
	}
}

// Test 1: ColonyStore.Add writes entry to entries.json and assigns ID
func TestColonyStoreAdd(t *testing.T) {
	cs, store, _ := newTestColonyStore(t)

	entry := makeEntry("", "learned something", 0.8)
	if err := cs.Add(entry); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if entry.ID == "" {
		t.Error("Add should assign an ID")
	}

	// Verify persistence
	var loaded []Entry
	if err := store.LoadJSON("entries.json", &loaded); err != nil {
		t.Fatalf("load entries: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded))
	}
	if loaded[0].ID != entry.ID {
		t.Errorf("loaded ID = %q, want %q", loaded[0].ID, entry.ID)
	}
	if loaded[0].Content != "learned something" {
		t.Errorf("loaded Content = %q, want %q", loaded[0].Content, "learned something")
	}
}

// Test 2: ColonyStore.Get retrieves entry by ID
func TestColonyStoreGet(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	entry := makeEntry("", "retrievable", 0.9)
	if err := cs.Add(entry); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := cs.Get(entry.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil for existing entry")
	}
	if got.Content != "retrievable" {
		t.Errorf("Get Content = %q, want %q", got.Content, "retrievable")
	}
}

// Test 3: ColonyStore.Get returns nil for nonexistent ID
func TestColonyStoreGetNotFound(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	got, err := cs.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get nonexistent: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent ID, got %+v", got)
	}
}

// Test 4: ColonyStore.List returns all entries (empty slice when none)
func TestColonyStoreListAll(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	// Empty store should return empty slice, not nil
	entries, err := cs.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if entries == nil {
		t.Error("List should return empty slice, not nil")
	}
	if len(entries) != 0 {
		t.Errorf("List empty = %d entries, want 0", len(entries))
	}

	// Add entries and list
	for i := 0; i < 3; i++ {
		if err := cs.Add(makeEntry("", "entry", float64(i))); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	entries, err = cs.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("List = %d entries, want 3", len(entries))
	}
}

// Test 5: ColonyStore.List filters by phase
func TestColonyStoreListFilterPhase(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	e1 := makeEntry("", "phase 1", 0.8)
	e1.Phase = 1
	e2 := makeEntry("", "phase 2", 0.8)
	e2.Phase = 2
	cs.Add(e1)
	cs.Add(e2)

	entries, err := cs.List(EntryFilter{Phase: 2})
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

// Test 6: ColonyStore.List filters by classification
func TestColonyStoreListFilterClassification(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	e1 := makeEntry("", "local", 0.8)
	e1.Classification = ClassRepoLocal
	e2 := makeEntry("", "shareable", 0.9)
	e2.Classification = ClassHiveShareable
	cs.Add(e1)
	cs.Add(e2)

	entries, err := cs.List(EntryFilter{Classification: ClassHiveShareable})
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

// Test 7: ColonyStore.Replace updates existing entry content and classification
func TestColonyStoreReplace(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	entry := makeEntry("", "original", 0.5)
	cs.Add(entry)

	updated := entry
	updated.Content = "updated"
	updated.Classification = ClassHiveShareable
	if err := cs.Replace(entry.ID, updated); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	got, err := cs.Get(entry.ID)
	if err != nil {
		t.Fatalf("Get after replace: %v", err)
	}
	if got.Content != "updated" {
		t.Errorf("Content = %q, want %q", got.Content, "updated")
	}
	if got.Classification != ClassHiveShareable {
		t.Errorf("Classification = %q, want %q", got.Classification, ClassHiveShareable)
	}
}

// Test 8: ColonyStore.Remove deletes entry by ID
func TestColonyStoreRemove(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	entry := makeEntry("", "to remove", 0.7)
	cs.Add(entry)

	if err := cs.Remove(entry.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	got, err := cs.Get(entry.ID)
	if err != nil {
		t.Fatalf("Get after remove: %v", err)
	}
	if got != nil {
		t.Error("expected nil after Remove, got entry")
	}
}

// Test 9: ColonyStore.Compact removes lowest-confidence entries to fit budget
func TestColonyStoreCompact(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	// Add 3 entries with different confidences
	e1 := makeEntry("", "low confidence long content here", 0.3)
	e2 := makeEntry("", "medium", 0.6)
	e3 := makeEntry("", "high", 0.9)
	cs.Add(e1)
	cs.Add(e2)
	cs.Add(e3)

	// Budget of ~20 chars -- should keep high-confidence entries first
	if err := cs.Compact(20); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	entries, err := cs.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List after compact: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("Compact removed all entries")
	}

	// Verify lowest-confidence entry was removed
	for _, e := range entries {
		if e.Content == "low confidence long content here" {
			t.Error("low-confidence entry should have been removed by Compact")
		}
	}

	// High-confidence entry should survive
	foundHigh := false
	for _, e := range entries {
		if e.Content == "high" {
			foundHigh = true
		}
	}
	if !foundHigh {
		t.Error("high-confidence entry should survive Compact")
	}
}

// Test 10: Two ColonyStore instances with different dirs have isolated entries (LRN-05)
func TestColonyStoreRepoIsolation(t *testing.T) {
	t.Parallel()

	// Store A
	dirA := t.TempDir()
	dataDirA := filepath.Join(dirA, ".aether", "data", "learn")
	storeA, err := storage.NewStore(dataDirA)
	if err != nil {
		t.Fatalf("create store A: %v", err)
	}
	csA := NewColonyStore(storeA)

	// Store B
	dirB := t.TempDir()
	dataDirB := filepath.Join(dirB, ".aether", "data", "learn")
	storeB, err := storage.NewStore(dataDirB)
	if err != nil {
		t.Fatalf("create store B: %v", err)
	}
	csB := NewColonyStore(storeB)

	// Add to A only
	entryA := makeEntry("", "only in A", 0.8)
	if err := csA.Add(entryA); err != nil {
		t.Fatalf("Add to A: %v", err)
	}

	// Verify B doesn't see A's entry
	entriesB, err := csB.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List B: %v", err)
	}
	if len(entriesB) != 0 {
		t.Errorf("Store B should have 0 entries, got %d", len(entriesB))
	}

	// Verify A sees its own entry
	entriesA, err := csA.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List A: %v", err)
	}
	if len(entriesA) != 1 {
		t.Errorf("Store A should have 1 entry, got %d", len(entriesA))
	}

	// Verify physical isolation -- B's data dir should not contain entries.json
	if _, err := os.Stat(filepath.Join(dataDirB, "entries.json")); err == nil {
		t.Error("Store B should not have entries.json on disk")
	}
}

// Test 11: ColonyStore.Add assigns unique IDs
func TestColonyStoreAddUniqueIDs(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	e1 := makeEntry("", "first", 0.8)
	e2 := makeEntry("", "second", 0.8)
	cs.Add(e1)
	cs.Add(e2)

	if e1.ID == e2.ID {
		t.Errorf("IDs should be unique: both = %q", e1.ID)
	}
}

// Test 12: ColonyStore.List filters by MinConfidence
func TestColonyStoreListFilterMinConfidence(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	cs.Add(makeEntry("", "low", 0.3))
	cs.Add(makeEntry("", "high", 0.8))

	entries, err := cs.List(EntryFilter{MinConfidence: 0.5})
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

// Test 13: ColonyStore.Compact with budget larger than content does nothing
func TestColonyStoreCompactNoop(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	cs.Add(makeEntry("", "small", 0.5))
	cs.Add(makeEntry("", "entry", 0.7))

	if err := cs.Compact(10000); err != nil {
		t.Fatalf("Compact large budget: %v", err)
	}

	entries, err := cs.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries after large-budget compact, got %d", len(entries))
	}
}

// Test 14: ColonyStore.Replace returns error for nonexistent ID
func TestColonyStoreReplaceNotFound(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	err := cs.Replace("nonexistent", makeEntry("nonexistent", "x", 0.5))
	if err == nil {
		t.Error("Replace should return error for nonexistent ID")
	}
}

// Test 15: ColonyStore.Remove returns error for nonexistent ID
func TestColonyStoreRemoveNotFound(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	err := cs.Remove("nonexistent")
	if err == nil {
		t.Error("Remove should return error for nonexistent ID")
	}
}
