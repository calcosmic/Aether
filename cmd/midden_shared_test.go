package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestLoadMiddenFileMissingFileReturnsErrorAndZeroValue locks the contract
// every existing consumer already depends on: a missing midden file degrades
// to a zero-value colony.MiddenFile plus a non-nil error, never a panic.
func TestLoadMiddenFileMissingFileReturnsErrorAndZeroValue(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	mf, err := loadMiddenFile(s)
	if err == nil {
		t.Fatal("expected an error when the canonical midden file does not exist, got nil")
	}
	if len(mf.Entries) != 0 || mf.Version != "" {
		t.Errorf("expected a zero-value MiddenFile on a missing file, got %+v", mf)
	}
}

// TestLoadMiddenFilePopulatedReturnsEntries proves loadMiddenFile reads the
// canonical flat path, not the dead nested one.
func TestLoadMiddenFilePopulatedReturnsEntries(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	seed := colony.MiddenFile{
		Version: "1.0.0",
		Entries: []colony.MiddenEntry{
			{ID: "m1", Timestamp: "2026-08-19T00:00:00Z", Category: "build", Source: "builder", Message: "seeded failure"},
		},
	}
	if err := s.SaveJSON(middenCanonicalPath, seed); err != nil {
		t.Fatalf("seed canonical midden file: %v", err)
	}

	mf, err := loadMiddenFile(s)
	if err != nil {
		t.Fatalf("unexpected error loading a populated midden file: %v", err)
	}
	if len(mf.Entries) != 1 {
		t.Fatalf("Entries length = %d, want 1", len(mf.Entries))
	}
	if mf.Entries[0].Message != "seeded failure" {
		t.Errorf("Entries[0].Message = %q, want %q", mf.Entries[0].Message, "seeded failure")
	}
}

// TestAppendMiddenEntryFreshStoreCreatesFile proves appendMiddenEntry creates
// the file on a fresh store, populating every field midden-write itself sets.
func TestAppendMiddenEntryFreshStoreCreatesFile(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	if err := appendMiddenEntry(s, "chaos", "test", "adversarial probe found a real bug", []string{"critical"}); err != nil {
		t.Fatalf("appendMiddenEntry: %v", err)
	}

	mf, err := loadMiddenFile(s)
	if err != nil {
		t.Fatalf("loadMiddenFile after append: %v", err)
	}
	if mf.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", mf.Version)
	}
	if len(mf.Entries) != 1 {
		t.Fatalf("Entries length = %d, want 1", len(mf.Entries))
	}
	e := mf.Entries[0]
	if e.Category != "chaos" || e.Source != "test" || e.Message != "adversarial probe found a real bug" {
		t.Errorf("entry fields wrong: %+v", e)
	}
	if e.ID == "" {
		t.Error("entry ID should not be empty")
	}
	if e.Reviewed {
		t.Error("entry Reviewed should default to false")
	}
	if _, err := time.Parse(time.RFC3339, e.Timestamp); err != nil {
		t.Errorf("entry Timestamp %q is not RFC3339: %v", e.Timestamp, err)
	}
	if len(e.Tags) != 1 || e.Tags[0] != "critical" {
		t.Errorf("Tags = %v, want [critical]", e.Tags)
	}
}

// TestAppendMiddenEntryTwiceAppendsBothOldestFirst proves the second call
// appends rather than overwrites, the first entry survives byte-for-byte,
// and both existing production writers' shape (append-then-atomic-write) is
// honored rather than accidentally clobbered by a naive Load-then-Save pair.
func TestAppendMiddenEntryTwiceAppendsBothOldestFirst(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	if err := appendMiddenEntry(s, "build", "builder", "first failure", nil); err != nil {
		t.Fatalf("first append: %v", err)
	}
	if err := appendMiddenEntry(s, "test", "watcher", "second failure", nil); err != nil {
		t.Fatalf("second append: %v", err)
	}

	mf, err := loadMiddenFile(s)
	if err != nil {
		t.Fatalf("loadMiddenFile: %v", err)
	}
	if len(mf.Entries) != 2 {
		t.Fatalf("Entries length = %d, want 2", len(mf.Entries))
	}
	if mf.Entries[0].Message != "first failure" {
		t.Errorf("Entries[0].Message = %q, want %q (oldest first)", mf.Entries[0].Message, "first failure")
	}
	if mf.Entries[1].Message != "second failure" {
		t.Errorf("Entries[1].Message = %q, want %q", mf.Entries[1].Message, "second failure")
	}
	// The first entry must be untouched, byte-for-byte, by the second append.
	if mf.Entries[0].Category != "build" || mf.Entries[0].Source != "builder" {
		t.Errorf("Entries[0] mutated by second append: %+v", mf.Entries[0])
	}
}

// TestAppendMiddenEntryNilTagsBecomesEmptyNonNilSlice locks midden-write's own
// convention (Tags: []string{}), so a nil tags argument never round-trips as
// JSON null.
func TestAppendMiddenEntryNilTagsBecomesEmptyNonNilSlice(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	if err := appendMiddenEntry(s, "build", "builder", "no tags here", nil); err != nil {
		t.Fatalf("appendMiddenEntry: %v", err)
	}

	mf, err := loadMiddenFile(s)
	if err != nil {
		t.Fatalf("loadMiddenFile: %v", err)
	}
	if mf.Entries[0].Tags == nil {
		t.Error("Tags should default to a non-nil empty slice, got nil")
	}
	if len(mf.Entries[0].Tags) != 0 {
		t.Errorf("Tags = %v, want empty", mf.Entries[0].Tags)
	}
}
