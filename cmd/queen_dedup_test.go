package cmd

import (
	"strings"
	"testing"
)

func TestAppendEntriesDedupSkipsDuplicate(t *testing.T) {
	base := "# QUEEN.md\n\n## Wisdom\n\n- Always use tabs (promoted 2026-04-01)\n"
	// Same content, different date suffix -- should be filtered
	entries := []string{"- Always use tabs (promoted 2026-04-05)"}
	result := appendEntriesToQueenSection(base, "Wisdom", entries)
	count := strings.Count(result, "Always use tabs")
	if count != 1 {
		t.Fatalf("expected 1 occurrence of wisdom line, got %d\n%s", count, result)
	}
}

func TestAppendEntriesDedupAllowsNewEntry(t *testing.T) {
	base := "# QUEEN.md\n\n## Wisdom\n\n- Always use tabs (promoted 2026-04-01)\n"
	entries := []string{"- Never commit to main (promoted 2026-04-05)"}
	result := appendEntriesToQueenSection(base, "Wisdom", entries)
	if !strings.Contains(result, "Always use tabs") {
		t.Fatal("existing entry removed")
	}
	if !strings.Contains(result, "Never commit to main") {
		t.Fatal("new entry not added")
	}
}

func TestAppendEntriesDedupStripsMultipleFormats(t *testing.T) {
	base := "# QUEEN.md\n\n## Wisdom\n\n- Test early test often (phase learning, 2026-04-01)\n"
	// Same content with different parenthetical format
	entries := []string{
		"- Test early test often (promoted 2026-04-05)",
		"- Test early test often (instinct inst_abc, 2026-04-06)",
		"- Test early test often (hive wisdom)",
	}
	result := appendEntriesToQueenSection(base, "Wisdom", entries)
	count := strings.Count(result, "Test early test often")
	if count != 1 {
		t.Fatalf("expected 1 occurrence (all are duplicates), got %d\n%s", count, result)
	}
}

func TestAppendEntriesDedupCreatesNewSection(t *testing.T) {
	base := "# QUEEN.md\n\n## Wisdom\n\n- Existing entry\n"
	entries := []string{"- New philosophical insight (promoted 2026-04-05)"}
	result := appendEntriesToQueenSection(base, "Philosophies", entries)
	if !strings.Contains(result, "## Philosophies") {
		t.Fatal("Philosophies section not created")
	}
	if !strings.Contains(result, "New philosophical insight") {
		t.Fatal("new entry not added to new section")
	}
}

func TestNormalizeQueenEntry(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"- Test early (promoted 2026-04-05)", "- Test early"},
		{"- Test early (phase learning, 2026-04-01)", "- Test early"},
		{"- Test early (instinct inst_abc, 2026-04-06)", "- Test early"},
		{"- Test early (hive wisdom)", "- Test early"},
		{"- No suffix here", "- No suffix here"},
		{"  -   Extra   Spaces   (promoted 2026-04-05)  ", "- Extra Spaces"},
	}
	for _, tc := range cases {
		got := normalizeQueenEntry(tc.input)
		if got != tc.expected {
			t.Errorf("normalizeQueenEntry(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
