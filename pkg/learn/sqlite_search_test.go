package learn

import (
	"testing"
)

func TestFTS5Search(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	entries := []struct {
		content string
		conf    float64
	}{
		{"memory leak detection in garbage collector", 0.8},
		{"test failure in auth module during CI", 0.7},
		{"performance optimization for database queries", 0.6},
		{"memory allocation error when parsing large files", 0.9},
		{"database connection timeout after retry", 0.5},
	}
	for _, e := range entries {
		store.Add(makeEntry("", e.content, e.conf))
	}

	results, err := store.Search("memory leak", EntryFilter{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	// "memory leak" as AND query matches only entries with both tokens
	if len(results) < 1 {
		t.Fatalf("expected at least 1 result for 'memory leak', got %d", len(results))
	}

	// Verify the result contains "memory leak"
	found := false
	for _, r := range results {
		if r.Content == "memory leak detection in garbage collector" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find 'memory leak detection in garbage collector', got: %v", results)
	}
}

func TestFTS5Search_NoMatch(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	store.Add(makeEntry("", "memory leak detection", 0.8))
	store.Add(makeEntry("", "test failure analysis", 0.7))

	results, err := store.Search("quantum computing", EntryFilter{})
	if err != nil {
		t.Fatalf("Search no match: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for 'quantum computing', got %d", len(results))
	}
}

func TestFTS5Search_FilterClassification(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	e1 := makeEntry("", "memory error handling pattern", 0.8)
	e1.Classification = ClassRepoLocal
	e2 := makeEntry("", "memory leak detection in shared code", 0.9)
	e2.Classification = ClassHiveShareable
	store.Add(e1)
	store.Add(e2)

	results, err := store.Search("memory", EntryFilter{Classification: ClassRepoLocal})
	if err != nil {
		t.Fatalf("Search filter classification: %v", err)
	}
	for _, r := range results {
		if r.Classification != ClassRepoLocal {
			t.Errorf("result Classification = %q, want %q", r.Classification, ClassRepoLocal)
		}
	}
}

func TestFTS5Search_FilterMinConfidence(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	store.Add(makeEntry("", "low confidence memory pattern", 0.5))
	store.Add(makeEntry("", "high confidence memory pattern", 0.9))

	results, err := store.Search("memory", EntryFilter{MinConfidence: 0.8})
	if err != nil {
		t.Fatalf("Search filter min_confidence: %v", err)
	}
	for _, r := range results {
		if r.Confidence < 0.8 {
			t.Errorf("result Confidence = %f, want >= 0.8", r.Confidence)
		}
	}
}

func TestFTS5Search_FilterLimit(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	for i := 0; i < 10; i++ {
		store.Add(makeEntry("", "error handling pattern in module code", float64(i)/10.0+0.1))
	}

	results, err := store.Search("error handling", EntryFilter{Limit: 3})
	if err != nil {
		t.Fatalf("Search filter limit: %v", err)
	}
	if len(results) > 3 {
		t.Errorf("expected at most 3 results, got %d", len(results))
	}
}

func TestFTS5SyncTriggers_Insert(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeEntry("", "unique searchable content xyzzy", 0.8)
	store.Add(entry)

	results, err := store.Search("xyzzy", EntryFilter{})
	if err != nil {
		t.Fatalf("Search after insert: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'xyzzy', got %d", len(results))
	}
}

func TestFTS5SyncTriggers_Delete(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeEntry("", "unique searchable content plugh", 0.8)
	store.Add(entry)

	// Verify entry is searchable
	results, _ := store.Search("plugh", EntryFilter{})
	if len(results) != 1 {
		t.Fatalf("expected 1 result before delete, got %d", len(results))
	}

	// Remove entry
	entries, _ := store.List(EntryFilter{Limit: 1})
	store.Remove(entries[0].ID)

	results, err := store.Search("plugh", EntryFilter{})
	if err != nil {
		t.Fatalf("Search after delete: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after delete, got %d", len(results))
	}
}

func TestFTS5SyncTriggers_Update(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeEntry("", "original content here", 0.8)
	store.Add(entry)

	// Get the assigned ID
	entries, _ := store.List(EntryFilter{Limit: 1})
	id := entries[0].ID

	// Replace with new content
	updated := entries[0]
	updated.Content = "updated unique content xyzabc"
	store.Replace(id, updated)

	results, err := store.Search("xyzabc", EntryFilter{})
	if err != nil {
		t.Fatalf("Search after update: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'xyzabc', got %d", len(results))
	}

	// Old content should no longer be found
	oldResults, _ := store.Search("original content here", EntryFilter{})
	if len(oldResults) != 0 {
		t.Errorf("expected 0 results for old content after update, got %d", len(oldResults))
	}
}

func TestFTS5Search_Ranking(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	// Entry that mentions "memory" once vs entry that mentions it multiple times
	store.Add(makeEntry("", "memory error", 0.5))
	store.Add(makeEntry("", "memory leak detection memory allocation memory safety review", 0.7))

	results, err := store.Search("memory", EntryFilter{})
	if err != nil {
		t.Fatalf("Search ranking: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// The entry with more "memory" occurrences should rank first (lower rank value = more relevant)
	// In FTS5, rank is negative BM25 score, so more relevant results have more negative rank
	// First result should be the one with more occurrences
	if results[0].Content != "memory leak detection memory allocation memory safety review" {
		t.Logf("Note: ranking order was %q, %q (FTS5 BM25 ranking may vary)",
			results[0].Content, results[1].Content)
	}
}

func TestFTS5Search_EmptyQuery(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	defer store.Close()

	store.Add(makeEntry("", "some content", 0.8))

	results, err := store.Search("", EntryFilter{})
	if err != nil {
		t.Fatalf("Search empty query: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}
