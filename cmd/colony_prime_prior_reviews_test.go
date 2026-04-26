package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// writeTestLedger writes a review ledger file for the given domain.
func writeTestLedger(t *testing.T, s *storage.Store, domain string, entries []colony.ReviewLedgerEntry) {
	t.Helper()
	lf := colony.ReviewLedgerFile{
		Entries: entries,
		Summary: colony.ComputeSummary(entries),
	}
	if err := s.SaveJSON(fmt.Sprintf("reviews/%s/ledger.json", domain), lf); err != nil {
		t.Fatal(err)
	}
}

// setupPriorReviewsTest creates a test store for prior-reviews tests.
func setupPriorReviewsTest(t *testing.T) *storage.Store {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	return s
}

func TestPriorReviews_OmittedWhenEmpty(t *testing.T) {
	s := setupPriorReviewsTest(t)

	section, count := buildPriorReviewsSection(s, false)
	if count != 0 {
		t.Errorf("count = %d, want 0 when no ledgers", count)
	}
	if section.name != "" {
		t.Errorf("section.name = %q, want empty when no ledgers", section.name)
	}
	if section.content != "" {
		t.Errorf("section.content should be empty, got: %s", section.content)
	}
}

func TestPriorReviews_BasicFormat(t *testing.T) {
	s := setupPriorReviewsTest(t)

	entries := []colony.ReviewLedgerEntry{
		{
			ID: "sec-2-001", Phase: 2, Agent: "gatekeeper", GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Status: "open", Severity: colony.ReviewSeverityHigh,
			File: "auth.go", Line: 42, Description: "exposed secret in handler",
		},
	}
	writeTestLedger(t, s, "security", entries)

	section, count := buildPriorReviewsSection(s, false)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if section.name != "prior_reviews" {
		t.Errorf("section.name = %q, want prior_reviews", section.name)
	}
	if section.priority != 8 {
		t.Errorf("section.priority = %d, want 8", section.priority)
	}
	// Should contain domain name, open count, severity, and file:line
	if !strings.Contains(section.content, "Security (1 open)") {
		t.Errorf("content should contain 'Security (1 open)', got: %s", section.content)
	}
	if !strings.Contains(section.content, "HIGH") {
		t.Errorf("content should contain 'HIGH', got: %s", section.content)
	}
	if !strings.Contains(section.content, "auth.go:42") {
		t.Errorf("content should contain 'auth.go:42', got: %s", section.content)
	}
}

func TestPriorReviews_MultipleDomains_SeverityOrder(t *testing.T) {
	s := setupPriorReviewsTest(t)

	// Quality has MEDIUM, security has HIGH -- security should appear first
	qualEntries := []colony.ReviewLedgerEntry{
		{
			ID: "qlt-2-001", Phase: 2, Agent: "auditor", GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Status: "open", Severity: colony.ReviewSeverityMedium,
			File: "handler.go", Line: 10, Description: "missing error check",
		},
	}
	writeTestLedger(t, s, "quality", qualEntries)

	secEntries := []colony.ReviewLedgerEntry{
		{
			ID: "sec-2-001", Phase: 2, Agent: "gatekeeper", GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Status: "open", Severity: colony.ReviewSeverityHigh,
			File: "auth.go", Line: 5, Description: "bcrypt weakness",
		},
	}
	writeTestLedger(t, s, "security", secEntries)

	section, _ := buildPriorReviewsSection(s, false)
	secIdx := strings.Index(section.content, "Security")
	qualIdx := strings.Index(section.content, "Quality")
	if secIdx >= qualIdx {
		t.Errorf("Security (HIGH) should appear before Quality (MEDIUM); secIdx=%d, qualIdx=%d", secIdx, qualIdx)
	}
}

func TestPriorReviews_TwoFindingsPerDomain(t *testing.T) {
	s := setupPriorReviewsTest(t)

	entries := []colony.ReviewLedgerEntry{
		{ID: "sec-2-001", Phase: 2, Agent: "gatekeeper", GeneratedAt: time.Now().UTC().Format(time.RFC3339), Status: "open", Severity: colony.ReviewSeverityHigh, File: "a.go", Line: 1, Description: "issue 1"},
		{ID: "sec-2-002", Phase: 2, Agent: "gatekeeper", GeneratedAt: time.Now().UTC().Format(time.RFC3339), Status: "open", Severity: colony.ReviewSeverityMedium, File: "b.go", Line: 2, Description: "issue 2"},
		{ID: "sec-2-003", Phase: 2, Agent: "gatekeeper", GeneratedAt: time.Now().UTC().Format(time.RFC3339), Status: "open", Severity: colony.ReviewSeverityLow, File: "c.go", Line: 3, Description: "issue 3"},
		{ID: "sec-2-004", Phase: 2, Agent: "gatekeeper", GeneratedAt: time.Now().UTC().Format(time.RFC3339), Status: "open", Severity: colony.ReviewSeverityInfo, File: "d.go", Line: 4, Description: "issue 4"},
		{ID: "sec-2-005", Phase: 2, Agent: "gatekeeper", GeneratedAt: time.Now().UTC().Format(time.RFC3339), Status: "open", Severity: colony.ReviewSeverityInfo, File: "e.go", Line: 5, Description: "issue 5"},
	}
	writeTestLedger(t, s, "security", entries)

	section, count := buildPriorReviewsSection(s, false)
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
	if !strings.Contains(section.content, "+3 more") {
		t.Errorf("content should contain '+3 more' for truncated findings, got: %s", section.content)
	}
}

func TestPriorReviews_ExcludesResolved(t *testing.T) {
	s := setupPriorReviewsTest(t)

	now := time.Now().UTC().Format(time.RFC3339)
	entries := []colony.ReviewLedgerEntry{
		{ID: "sec-2-001", Phase: 2, Agent: "gatekeeper", GeneratedAt: now, Status: "open", Severity: colony.ReviewSeverityHigh, File: "a.go", Line: 1, Description: "open issue"},
		{ID: "sec-2-002", Phase: 2, Agent: "gatekeeper", GeneratedAt: now, Status: "resolved", Severity: colony.ReviewSeverityMedium, File: "b.go", Line: 2, Description: "resolved issue"},
	}
	writeTestLedger(t, s, "security", entries)

	section, count := buildPriorReviewsSection(s, false)
	if count != 1 {
		t.Errorf("count = %d, want 1 (only open)", count)
	}
	if !strings.Contains(section.content, "Security (1 open)") {
		t.Errorf("content should show 1 open, got: %s", section.content)
	}
	if strings.Contains(section.content, "resolved issue") {
		t.Errorf("content should NOT contain resolved finding, got: %s", section.content)
	}
}

func TestPriorReviews_BudgetCap800(t *testing.T) {
	s := setupPriorReviewsTest(t)

	now := time.Now().UTC().Format(time.RFC3339)
	// Create 7 domains with 3 findings each to stress the 800-char budget
	for _, domain := range colony.DomainOrder {
		entries := []colony.ReviewLedgerEntry{
			{ID: fmt.Sprintf("%s-2-001", domain[:3]), Phase: 2, GeneratedAt: now, Status: "open", Severity: colony.ReviewSeverityHigh, File: fmt.Sprintf("%s_a.go", domain), Line: 1, Description: "first finding with a somewhat longer description to test budget"},
			{ID: fmt.Sprintf("%s-2-002", domain[:3]), Phase: 2, GeneratedAt: now, Status: "open", Severity: colony.ReviewSeverityMedium, File: fmt.Sprintf("%s_b.go", domain), Line: 2, Description: "second finding also with some text to eat into budget"},
			{ID: fmt.Sprintf("%s-2-003", domain[:3]), Phase: 2, GeneratedAt: now, Status: "open", Severity: colony.ReviewSeverityLow, File: fmt.Sprintf("%s_c.go", domain), Line: 3, Description: "third finding description text"},
		}
		writeTestLedger(t, s, domain, entries)
	}

	section, _ := buildPriorReviewsSection(s, false)
	if len(section.content) > 800 {
		t.Errorf("content length %d exceeds 800-char budget:\n%s", len(section.content), section.content)
	}
}

func TestPriorReviews_BudgetCap400Compact(t *testing.T) {
	s := setupPriorReviewsTest(t)

	now := time.Now().UTC().Format(time.RFC3339)
	for _, domain := range colony.DomainOrder {
		entries := []colony.ReviewLedgerEntry{
			{ID: fmt.Sprintf("%s-2-001", domain[:3]), Phase: 2, GeneratedAt: now, Status: "open", Severity: colony.ReviewSeverityHigh, File: fmt.Sprintf("%s.go", domain), Line: 1, Description: "finding description"},
			{ID: fmt.Sprintf("%s-2-002", domain[:3]), Phase: 2, GeneratedAt: now, Status: "open", Severity: colony.ReviewSeverityMedium, File: fmt.Sprintf("%s2.go", domain), Line: 2, Description: "another finding"},
		}
		writeTestLedger(t, s, domain, entries)
	}

	section, _ := buildPriorReviewsSection(s, true)
	if len(section.content) > 400 {
		t.Errorf("compact content length %d exceeds 400-char budget:\n%s", len(section.content), section.content)
	}
}

func TestPriorReviews_CacheHit(t *testing.T) {
	s := setupPriorReviewsTest(t)

	entries := []colony.ReviewLedgerEntry{
		{
			ID: "sec-2-001", Phase: 2, Agent: "gatekeeper", GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Status: "open", Severity: colony.ReviewSeverityHigh,
			File: "auth.go", Line: 42, Description: "exposed secret",
		},
	}
	writeTestLedger(t, s, "security", entries)

	// First call writes the cache
	section1, count1 := buildPriorReviewsSection(s, false)
	if count1 != 1 {
		t.Fatalf("first call count = %d, want 1", count1)
	}

	// Delete the ledger file so a fresh read would fail
	os.Remove(filepath.Join(s.BasePath(), "reviews", "security", "ledger.json"))

	// Second call should return cached text
	section2, count2 := buildPriorReviewsSection(s, false)
	if count2 != count1 {
		t.Errorf("cached count = %d, want %d", count2, count1)
	}
	if section2.content != section1.content {
		t.Errorf("cached content differs:\ngot:  %s\nwant: %s", section2.content, section1.content)
	}
}

func TestPriorReviews_CacheStale(t *testing.T) {
	s := setupPriorReviewsTest(t)

	now := time.Now().UTC().Format(time.RFC3339)
	entries := []colony.ReviewLedgerEntry{
		{
			ID: "sec-2-001", Phase: 2, Agent: "gatekeeper", GeneratedAt: now,
			Status: "open", Severity: colony.ReviewSeverityHigh,
			File: "auth.go", Line: 42, Description: "old finding",
		},
	}
	writeTestLedger(t, s, "security", entries)

	// First call writes cache
	buildPriorReviewsSection(s, false)

	// Ensure the ledger file gets a newer mtime than the cache
	time.Sleep(10 * time.Millisecond)
	newEntries := []colony.ReviewLedgerEntry{
		{
			ID: "sec-2-001", Phase: 2, Agent: "gatekeeper", GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Status: "open", Severity: colony.ReviewSeverityMedium,
			File: "new.go", Line: 10, Description: "new finding",
		},
	}
	writeTestLedger(t, s, "security", newEntries)

	// Second call should detect stale cache and rebuild
	section, count := buildPriorReviewsSection(s, false)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if !strings.Contains(section.content, "new finding") {
		t.Errorf("cache should have been rebuilt with new data, got: %s", section.content)
	}
	if strings.Contains(section.content, "old finding") {
		t.Errorf("cache should not contain old data, got: %s", section.content)
	}
}

func TestPriorReviews_Priority8(t *testing.T) {
	s := setupPriorReviewsTest(t)

	entries := []colony.ReviewLedgerEntry{
		{ID: "sec-2-001", Phase: 2, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Status: "open", Severity: colony.ReviewSeverityHigh, Description: "test"},
	}
	writeTestLedger(t, s, "security", entries)

	section, _ := buildPriorReviewsSection(s, false)
	if section.priority != 8 {
		t.Errorf("priority = %d, want 8", section.priority)
	}
}

func TestPriorReviews_Scores(t *testing.T) {
	s := setupPriorReviewsTest(t)

	entries := []colony.ReviewLedgerEntry{
		{ID: "sec-2-001", Phase: 2, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Status: "open", Severity: colony.ReviewSeverityHigh, Description: "test"},
	}
	writeTestLedger(t, s, "security", entries)

	section, _ := buildPriorReviewsSection(s, false)
	if section.confirmationScore != 1.0 {
		t.Errorf("confirmationScore = %f, want 1.0", section.confirmationScore)
	}
	if section.freshnessScore <= 0 {
		t.Errorf("freshnessScore = %f, want > 0", section.freshnessScore)
	}
}

func TestPriorReviews_TruncationToLowerSeverity(t *testing.T) {
	s := setupPriorReviewsTest(t)

	now := time.Now().UTC().Format(time.RFC3339)
	// Create many findings across domains to force truncation
	for _, domain := range colony.DomainOrder {
		entries := []colony.ReviewLedgerEntry{
			{ID: fmt.Sprintf("%s-2-001", domain[:3]), Phase: 2, GeneratedAt: now, Status: "open", Severity: colony.ReviewSeverityHigh, File: "a.go", Line: 1, Description: "long description that takes up space in the budget to force truncation of lower severity domains"},
			{ID: fmt.Sprintf("%s-2-002", domain[:3]), Phase: 2, GeneratedAt: now, Status: "open", Severity: colony.ReviewSeverityMedium, File: "b.go", Line: 2, Description: "another finding with some text"},
			{ID: fmt.Sprintf("%s-2-003", domain[:3]), Phase: 2, GeneratedAt: now, Status: "open", Severity: colony.ReviewSeverityLow, File: "c.go", Line: 3, Description: "yet another finding"},
		}
		writeTestLedger(t, s, domain, entries)
	}

	section, _ := buildPriorReviewsSection(s, false)
	// Should have at least one domain as full line and at least one truncated to counts-only
	if len(section.content) > 800 {
		t.Errorf("content exceeds 800 chars: %d", len(section.content))
	}
	// Verify the content has at least some domain entries
	if !strings.Contains(section.content, "Security") {
		t.Errorf("Security (HIGH severity) should always appear, got: %s", section.content)
	}
}

func TestPriorReviews_DescriptionTruncation(t *testing.T) {
	s := setupPriorReviewsTest(t)

	longDesc := strings.Repeat("a", 200)
	entries := []colony.ReviewLedgerEntry{
		{
			ID: "sec-2-001", Phase: 2, GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Status: "open", Severity: colony.ReviewSeverityHigh,
			Description: longDesc,
		},
	}
	writeTestLedger(t, s, "security", entries)

	section, _ := buildPriorReviewsSection(s, false)
	// The description should be truncated, not the full 200 chars
	for _, line := range strings.Split(section.content, "\n") {
		if strings.Contains(line, "aaaa") && len(line) > 120 {
			t.Errorf("description in line not truncated (len=%d): %s", len(line), line)
		}
	}
}

func TestPriorReviews_IntegratedInBuildOutput(t *testing.T) {
	s := setupPriorReviewsTest(t)

	// Create a minimal COLONY_STATE.json so buildColonyPrimeOutput works
	stateData := map[string]interface{}{
		"state":         "building",
		"current_phase": 2,
		"goal":          "test goal",
	}
	if err := s.SaveJSON("COLONY_STATE.json", stateData); err != nil {
		t.Fatal(err)
	}

	entries := []colony.ReviewLedgerEntry{
		{ID: "sec-2-001", Phase: 2, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Status: "open", Severity: colony.ReviewSeverityHigh, File: "auth.go", Line: 42, Description: "exposed secret"},
	}
	writeTestLedger(t, s, "security", entries)

	result := buildColonyPrimeOutput(false)
	if !strings.Contains(result.LogLine, "review(s)") {
		t.Errorf("LogLine should contain 'review(s)', got: %s", result.LogLine)
	}
	if result.ReviewCount < 1 {
		t.Errorf("ReviewCount = %d, want >= 1", result.ReviewCount)
	}
}
