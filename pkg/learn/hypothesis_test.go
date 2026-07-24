package learn

import (
	"testing"
)

// Test 1: Add() defaults Status to hypothesis when not provided
func TestAddDefaultsStatusToHypothesis(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	entry := makeEntry("", "test content", 0.8)
	// Explicitly leave Status empty
	entry.Status = ""

	if err := cs.Add(entry); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entries, err := cs.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Status != StatusHypothesis {
		t.Errorf("Status = %q, want %q", entries[0].Status, StatusHypothesis)
	}
}

// Test 2: Add() preserves explicitly set Status
func TestAddPreservesExplicitStatus(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	entry := makeEntry("", "validated content", 0.9)
	entry.Status = StatusValidated

	if err := cs.Add(entry); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entries, err := cs.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if entries[0].Status != StatusValidated {
		t.Errorf("Status = %q, want %q", entries[0].Status, StatusValidated)
	}
}

// Test 3: List() filters by status correctly
func TestListFilterByStatus(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	// Add entries with different statuses
	e1 := makeEntry("", "hypothesis entry", 0.5)
	e1.Status = StatusHypothesis
	cs.Add(e1)

	e2 := makeEntry("", "validated entry", 0.8)
	e2.Status = StatusValidated
	cs.Add(e2)

	e3 := makeEntry("", "disproven entry", 0.3)
	e3.Status = StatusDisproven
	cs.Add(e3)

	// Filter for hypothesis
	hypothesisEntries, err := cs.List(EntryFilter{Status: StatusHypothesis})
	if err != nil {
		t.Fatalf("List hypothesis: %v", err)
	}
	if len(hypothesisEntries) != 1 {
		t.Fatalf("expected 1 hypothesis entry, got %d", len(hypothesisEntries))
	}
	if hypothesisEntries[0].Content != "hypothesis entry" {
		t.Errorf("content = %q, want %q", hypothesisEntries[0].Content, "hypothesis entry")
	}

	// Filter for validated
	validatedEntries, err := cs.List(EntryFilter{Status: StatusValidated})
	if err != nil {
		t.Fatalf("List validated: %v", err)
	}
	if len(validatedEntries) != 1 {
		t.Fatalf("expected 1 validated entry, got %d", len(validatedEntries))
	}

	// Filter for disproven
	disprovenEntries, err := cs.List(EntryFilter{Status: StatusDisproven})
	if err != nil {
		t.Fatalf("List disproven: %v", err)
	}
	if len(disprovenEntries) != 1 {
		t.Fatalf("expected 1 disproven entry, got %d", len(disprovenEntries))
	}
}

// Test 4: Replace() updates status (validate/disprove) and persists
func TestReplaceUpdatesStatus(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	entry := makeEntry("", "original", 0.5)
	entry.Status = StatusHypothesis
	cs.Add(entry)

	entries, _ := cs.List(EntryFilter{})
	originalID := entries[0].ID

	// Update to validated
	updated := entries[0]
	updated.Status = StatusValidated
	updated.ParentID = originalID // Link to original hypothesis
	if err := cs.Replace(originalID, updated); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	got, err := cs.Get(originalID)
	if err != nil {
		t.Fatalf("Get after replace: %v", err)
	}
	if got.Status != StatusValidated {
		t.Errorf("Status = %q, want %q", got.Status, StatusValidated)
	}
	if got.ParentID != originalID {
		t.Errorf("ParentID = %q, want %q", got.ParentID, originalID)
	}
}

// Test 5: ParentID field is persisted and retrievable
func TestParentIDPersistence(t *testing.T) {
	cs, _, _ := newTestColonyStore(t)

	// Create hypothesis
	hypothesis := makeEntry("", "hypothesis", 0.6)
	hypothesis.Status = StatusHypothesis
	cs.Add(hypothesis)

	entries, _ := cs.List(EntryFilter{})
	hypothesisID := entries[0].ID

	// Create validated entry linking to hypothesis
	validated := makeEntry("", "validated", 0.85)
	validated.Status = StatusValidated
	validated.ParentID = hypothesisID
	cs.Add(validated)

	allEntries, err := cs.List(EntryFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(allEntries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(allEntries))
	}

	// Find the validated one
	var found *Entry
	for _, e := range allEntries {
		if e.Status == StatusValidated {
			found = &e
			break
		}
	}
	if found == nil {
		t.Fatal("validated entry not found")
	}
	if found.ParentID != hypothesisID {
		t.Errorf("ParentID = %q, want %q", found.ParentID, hypothesisID)
	}
}

// Test 6: Backward compatibility - entries without Status field default to hypothesis on read
func TestBackwardCompatibilityNoStatus(t *testing.T) {
	cs, store, _ := newTestColonyStore(t)

	// Simulate old entry without Status by writing raw JSON
	oldJSON := `[
  {
    "id": "lrn_old_1",
    "content": "legacy entry",
    "classification": "repo-local",
    "created_at": "2026-01-01T00:00:00Z",
    "phase": 1,
    "confidence": 0.7,
    "evidence": {
      "run_id": "run-1",
      "phase": 1,
      "workers": [],
      "gates_passed": 1,
      "gates_total": 1,
      "confidence": 0.7,
      "timestamp": "2026-01-01T00:00:00Z",
      "scope": "repo"
    }
  }
]`
	if err := store.AtomicWrite("entries.json", []byte(oldJSON)); err != nil {
		t.Fatalf("write old JSON: %v", err)
	}

	got, err := cs.Get("lrn_old_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected to find legacy entry")
	}
	if got.Status != StatusHypothesis {
		t.Errorf("legacy entry Status = %q, want %q (default)", got.Status, StatusHypothesis)
	}
}

// Test 7: Status constants exist with correct values
func TestStatusConstants(t *testing.T) {
	if StatusHypothesis != "hypothesis" {
		t.Errorf("StatusHypothesis = %q, want %q", StatusHypothesis, "hypothesis")
	}
	if StatusValidated != "validated" {
		t.Errorf("StatusValidated = %q, want %q", StatusValidated, "validated")
	}
	if StatusDisproven != "disproven" {
		t.Errorf("StatusDisproven = %q, want %q", StatusDisproven, "disproven")
	}
}
