package colony

import (
	"encoding/json"
	"strings"
	"testing"
)

// --- RED phase: Tests for review ledger types ---
// These tests are written BEFORE the implementation exists.

func TestReviewLedgerEntryJSONRoundTrip(t *testing.T) {
	resolvedAt := "2026-04-26T10:00:00Z"
	entry := ReviewLedgerEntry{
		ID:          "sec-2-001",
		Phase:       2,
		PhaseName:   "security-hardening",
		Agent:       "gatekeeper",
		AgentName:   "Gatekeeper",
		GeneratedAt: "2026-04-26T09:00:00Z",
		Status:      "open",
		Severity:    ReviewSeverityHigh,
		File:        "cmd/auth.go",
		Line:        42,
		Category:    "exposed-secret",
		Description: "Hardcoded API key found in source code",
		Suggestion:  "Move to environment variable",
		ResolvedAt:  &resolvedAt,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ReviewLedgerEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != entry.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, entry.ID)
	}
	if decoded.Phase != entry.Phase {
		t.Errorf("Phase mismatch: got %d, want %d", decoded.Phase, entry.Phase)
	}
	if decoded.PhaseName != entry.PhaseName {
		t.Errorf("PhaseName mismatch: got %q, want %q", decoded.PhaseName, entry.PhaseName)
	}
	if decoded.Agent != entry.Agent {
		t.Errorf("Agent mismatch: got %q, want %q", decoded.Agent, entry.Agent)
	}
	if decoded.AgentName != entry.AgentName {
		t.Errorf("AgentName mismatch: got %q, want %q", decoded.AgentName, entry.AgentName)
	}
	if decoded.GeneratedAt != entry.GeneratedAt {
		t.Errorf("GeneratedAt mismatch: got %q, want %q", decoded.GeneratedAt, entry.GeneratedAt)
	}
	if decoded.Status != entry.Status {
		t.Errorf("Status mismatch: got %q, want %q", decoded.Status, entry.Status)
	}
	if decoded.Severity != entry.Severity {
		t.Errorf("Severity mismatch: got %q, want %q", decoded.Severity, entry.Severity)
	}
	if decoded.File != entry.File {
		t.Errorf("File mismatch: got %q, want %q", decoded.File, entry.File)
	}
	if decoded.Line != entry.Line {
		t.Errorf("Line mismatch: got %d, want %d", decoded.Line, entry.Line)
	}
	if decoded.Category != entry.Category {
		t.Errorf("Category mismatch: got %q, want %q", decoded.Category, entry.Category)
	}
	if decoded.Description != entry.Description {
		t.Errorf("Description mismatch: got %q, want %q", decoded.Description, entry.Description)
	}
	if decoded.Suggestion != entry.Suggestion {
		t.Errorf("Suggestion mismatch: got %q, want %q", decoded.Suggestion, entry.Suggestion)
	}
	if decoded.ResolvedAt == nil {
		t.Fatal("ResolvedAt is nil after round-trip")
	}
	if *decoded.ResolvedAt != resolvedAt {
		t.Errorf("ResolvedAt mismatch: got %q, want %q", *decoded.ResolvedAt, resolvedAt)
	}
}

func TestReviewLedgerEntryOmitEmpty(t *testing.T) {
	entry := ReviewLedgerEntry{
		ID:          "sec-2-001",
		Phase:       2,
		Agent:       "gatekeeper",
		GeneratedAt: "2026-04-26T09:00:00Z",
		Status:      "open",
		Severity:    ReviewSeverityHigh,
		Description: "Hardcoded API key found",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	// These optional fields must be absent when zero-value
	omittedFields := []string{"resolved_at", "phase_name", "agent_name", "file", "line", "category", "suggestion"}
	for _, field := range omittedFields {
		if _, ok := raw[field]; ok {
			t.Errorf("field %q should be omitted but is present: %v", field, raw[field])
		}
	}

	// These required fields must be present
	requiredFields := []string{"id", "phase", "agent", "generated_at", "status", "severity", "description"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("required field %q is missing", field)
		}
	}
}

func TestComputeSummary(t *testing.T) {
	entries := []ReviewLedgerEntry{
		{ID: "sec-2-001", Phase: 2, Agent: "gatekeeper", GeneratedAt: "2026-04-26T09:00:00Z", Status: "open", Severity: ReviewSeverityHigh, Description: "d1"},
		{ID: "sec-2-002", Phase: 2, Agent: "gatekeeper", GeneratedAt: "2026-04-26T09:00:00Z", Status: "open", Severity: ReviewSeverityHigh, Description: "d2"},
		{ID: "qlt-2-003", Phase: 2, Agent: "auditor", GeneratedAt: "2026-04-26T09:00:00Z", Status: "resolved", Severity: ReviewSeverityMedium, Description: "d3"},
		{ID: "tst-2-004", Phase: 2, Agent: "watcher", GeneratedAt: "2026-04-26T09:00:00Z", Status: "open", Severity: ReviewSeverityLow, Description: "d4"},
		{ID: "bug-2-005", Phase: 2, Agent: "tracker", GeneratedAt: "2026-04-26T09:00:00Z", Status: "resolved", Severity: ReviewSeverityInfo, Description: "d5"},
	}

	summary := ComputeSummary(entries)

	if summary.Total != 5 {
		t.Errorf("Total: got %d, want 5", summary.Total)
	}
	if summary.Open != 3 {
		t.Errorf("Open: got %d, want 3", summary.Open)
	}
	if summary.Resolved != 2 {
		t.Errorf("Resolved: got %d, want 2", summary.Resolved)
	}
	if summary.BySeverity.High != 2 {
		t.Errorf("BySeverity.High: got %d, want 2", summary.BySeverity.High)
	}
	if summary.BySeverity.Medium != 1 {
		t.Errorf("BySeverity.Medium: got %d, want 1", summary.BySeverity.Medium)
	}
	if summary.BySeverity.Low != 1 {
		t.Errorf("BySeverity.Low: got %d, want 1", summary.BySeverity.Low)
	}
	if summary.BySeverity.Info != 1 {
		t.Errorf("BySeverity.Info: got %d, want 1", summary.BySeverity.Info)
	}
}

func TestComputeSummaryEmpty(t *testing.T) {
	summary := ComputeSummary(nil)

	if summary.Total != 0 {
		t.Errorf("Total: got %d, want 0", summary.Total)
	}
	if summary.Open != 0 {
		t.Errorf("Open: got %d, want 0", summary.Open)
	}
	if summary.Resolved != 0 {
		t.Errorf("Resolved: got %d, want 0", summary.Resolved)
	}
	if summary.BySeverity.High != 0 || summary.BySeverity.Medium != 0 || summary.BySeverity.Low != 0 || summary.BySeverity.Info != 0 {
		t.Errorf("BySeverity should all be 0: got %+v", summary.BySeverity)
	}
}

func TestFormatEntryID(t *testing.T) {
	tests := []struct {
		prefix string
		phase  int
		index  int
		want   string
	}{
		{"sec", 2, 1, "sec-2-001"},
		{"qlt", 10, 23, "qlt-10-023"},
		{"bug", 1, 999, "bug-1-999"},
		{"res", 100, 1, "res-100-001"},
	}
	for _, tt := range tests {
		got := FormatEntryID(tt.prefix, tt.phase, tt.index)
		if got != tt.want {
			t.Errorf("FormatEntryID(%q, %d, %d) = %q, want %q", tt.prefix, tt.phase, tt.index, got, tt.want)
		}
	}
}

func TestNextEntryIndex(t *testing.T) {
	t.Run("empty entries returns 1", func(t *testing.T) {
		got := NextEntryIndex(nil, "sec", 2)
		if got != 1 {
			t.Errorf("expected 1, got %d", got)
		}
	})

	t.Run("matching entries returns max+1", func(t *testing.T) {
		entries := []ReviewLedgerEntry{
			{ID: "sec-2-001", Phase: 2, Agent: "g", GeneratedAt: "t", Status: "open", Severity: ReviewSeverityHigh, Description: "d"},
			{ID: "sec-2-002", Phase: 2, Agent: "g", GeneratedAt: "t", Status: "open", Severity: ReviewSeverityHigh, Description: "d"},
		}
		got := NextEntryIndex(entries, "sec", 2)
		if got != 3 {
			t.Errorf("expected 3, got %d", got)
		}
	})

	t.Run("non-matching prefix returns 1", func(t *testing.T) {
		entries := []ReviewLedgerEntry{
			{ID: "sec-2-001", Phase: 2, Agent: "g", GeneratedAt: "t", Status: "open", Severity: ReviewSeverityHigh, Description: "d"},
		}
		got := NextEntryIndex(entries, "qlt", 2)
		if got != 1 {
			t.Errorf("expected 1, got %d", got)
		}
	})

	t.Run("non-matching phase returns 1", func(t *testing.T) {
		entries := []ReviewLedgerEntry{
			{ID: "sec-2-001", Phase: 2, Agent: "g", GeneratedAt: "t", Status: "open", Severity: ReviewSeverityHigh, Description: "d"},
		}
		got := NextEntryIndex(entries, "sec", 3)
		if got != 1 {
			t.Errorf("expected 1, got %d", got)
		}
	})
}

// --- Task 2 tests ---

func TestReviewLedgerFileJSONRoundTrip(t *testing.T) {
	entries := []ReviewLedgerEntry{
		{ID: "sec-2-001", Phase: 2, Agent: "gatekeeper", GeneratedAt: "2026-04-26T09:00:00Z", Status: "open", Severity: ReviewSeverityHigh, Description: "d1"},
		{ID: "qlt-2-001", Phase: 2, Agent: "auditor", GeneratedAt: "2026-04-26T09:00:00Z", Status: "resolved", Severity: ReviewSeverityMedium, Description: "d2"},
	}
	summary := ComputeSummary(entries)

	file := ReviewLedgerFile{
		Entries: entries,
		Summary: summary,
	}

	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ReviewLedgerFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(decoded.Entries))
	}
	if decoded.Summary.Total != 2 {
		t.Errorf("Summary.Total: got %d, want 2", decoded.Summary.Total)
	}
	if decoded.Summary.Open != 1 {
		t.Errorf("Summary.Open: got %d, want 1", decoded.Summary.Open)
	}
	if decoded.Summary.Resolved != 1 {
		t.Errorf("Summary.Resolved: got %d, want 1", decoded.Summary.Resolved)
	}
	if decoded.Entries[0].ID != "sec-2-001" {
		t.Errorf("first entry ID mismatch: got %q", decoded.Entries[0].ID)
	}
	if decoded.Entries[1].Severity != ReviewSeverityMedium {
		t.Errorf("second entry Severity mismatch: got %q", decoded.Entries[1].Severity)
	}
}

func TestReviewLedgerFileEmptyEntries(t *testing.T) {
	file := ReviewLedgerFile{
		Entries: []ReviewLedgerEntry{},
		Summary: ReviewLedgerSummary{},
	}

	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify JSON contains "entries":[] not "entries":null
	if !strings.Contains(string(data), `"entries":[]`) {
		t.Errorf("empty entries should serialize as [], got: %s", string(data))
	}

	var decoded ReviewLedgerFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Entries == nil {
		t.Error("Entries should be non-nil empty slice after round-trip, got nil")
	}
	if len(decoded.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(decoded.Entries))
	}
}
