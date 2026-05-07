package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// setupPersistenceTest creates a fresh test environment for persistence tests.
func setupPersistenceTest(t *testing.T) (*bytes.Buffer, *bytes.Buffer, *storage.Store) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	s, _ := newTestStore(t)
	store = s
	return &buf, &errBuf, s
}

// writeFindings writes findings to a domain ledger via the CLI command.
func writeFindings(t *testing.T, domain string, phase int, findings []map[string]interface{}) {
	t.Helper()
	findingsJSON, err := json.Marshal(findings)
	if err != nil {
		t.Fatalf("marshal findings: %v", err)
	}
	rootCmd.SetArgs([]string{
		"review-ledger-write",
		"--domain", domain,
		"--phase", fmt.Sprintf("%d", phase),
		"--findings", string(findingsJSON),
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("review-ledger-write error: %v", err)
	}
}

// readLedger reads ledger entries for a domain via direct store access.
func readLedger(t *testing.T, s *storage.Store, domain string) []colony.ReviewLedgerEntry {
	t.Helper()
	var lf colony.ReviewLedgerFile
	ledgerPath := fmt.Sprintf("reviews/%s/ledger.json", domain)
	if err := s.LoadJSON(ledgerPath, &lf); err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file or directory") {
			return []colony.ReviewLedgerEntry{}
		}
		t.Fatalf("load ledger for %s: %v", domain, err)
	}
	return lf.Entries
}

// countLedgerEntries returns the number of entries in a domain ledger.
func countLedgerEntries(t *testing.T, s *storage.Store, domain string) int {
	t.Helper()
	entries := readLedger(t, s, domain)
	return len(entries)
}

// loadGoldenSnapshot loads the golden snapshot file.
func loadGoldenSnapshot(t *testing.T) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile("testdata/review_ledger_persistence_snapshot.json")
	if err != nil {
		t.Fatalf("read golden snapshot: %v", err)
	}
	var snapshot map[string]interface{}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("unmarshal golden snapshot: %v", err)
	}
	return snapshot
}

// --- Test 1: Basic Persistence ---

func TestReviewLedgerPersistence(t *testing.T) {
	buf, _, s := setupPersistenceTest(t)
	store = s

	// Write findings to 3 different domains
	writeFindings(t, "security", 100, []map[string]interface{}{
		{"severity": "HIGH", "description": "exposed secret", "file": "auth.go", "line": 42},
	})
	writeFindings(t, "quality", 100, []map[string]interface{}{
		{"severity": "MEDIUM", "description": "missing error check", "file": "handler.go", "line": 10},
	})
	writeFindings(t, "testing", 100, []map[string]interface{}{
		{"severity": "LOW", "description": "incomplete test coverage", "file": "api_test.go", "line": 5},
	})

	// Verify all written findings are present
	secEntries := readLedger(t, s, "security")
	if len(secEntries) != 1 {
		t.Errorf("security entries = %d, want 1", len(secEntries))
	}
	qualEntries := readLedger(t, s, "quality")
	if len(qualEntries) != 1 {
		t.Errorf("quality entries = %d, want 1", len(qualEntries))
	}
	tstEntries := readLedger(t, s, "testing")
	if len(tstEntries) != 1 {
		t.Errorf("testing entries = %d, want 1", len(tstEntries))
	}

	// Verify expected fields are present
	for _, entry := range secEntries {
		if entry.Phase != 100 {
			t.Errorf("entry.phase = %d, want 100", entry.Phase)
		}
		if entry.ID == "" {
			t.Errorf("entry.id is empty")
		}
		if entry.Severity == "" {
			t.Errorf("entry.severity is empty")
		}
		if entry.Description == "" {
			t.Errorf("entry.description is empty")
		}
		if entry.GeneratedAt == "" {
			t.Errorf("entry.generated_at is empty")
		}
	}

	// Verify golden snapshot values
	snapshot := loadGoldenSnapshot(t)
	if int(snapshot["domain_count"].(float64)) != 7 {
		t.Errorf("domain_count = %v, want 7", snapshot["domain_count"])
	}
	if int(snapshot["max_findings_per_write"].(float64)) != 50 {
		t.Errorf("max_findings_per_write = %v, want 50", snapshot["max_findings_per_write"])
	}

	// Verify all 7 domains are supported by writing to each
	allDomains := []string{"security", "quality", "performance", "resilience", "testing", "history", "bugs"}
	for _, domain := range allDomains {
		buf.Reset()
		writeFindings(t, domain, 101, []map[string]interface{}{
			{"severity": "INFO", "description": fmt.Sprintf("test finding for %s", domain)},
		})
		entries := readLedger(t, s, domain)
		found := false
		for _, e := range entries {
			if e.Description == fmt.Sprintf("test finding for %s", domain) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("domain %s: expected write/read to succeed", domain)
		}
	}
}

// --- Test 2: Cross-Phase Accumulation ---

func TestReviewLedgerCrossPhaseAccumulation(t *testing.T) {
	_, _, s := setupPersistenceTest(t)
	store = s

	// Phase 100 writes 2 findings to security
	writeFindings(t, "security", 100, []map[string]interface{}{
		{"severity": "HIGH", "description": "phase 100 issue 1"},
		{"severity": "MEDIUM", "description": "phase 100 issue 2"},
	})

	// Verify security ledger has 2 entries
	if count := countLedgerEntries(t, s, "security"); count != 2 {
		t.Errorf("after phase 100: security entries = %d, want 2", count)
	}

	// Phase 101 writes 2 more findings to security
	writeFindings(t, "security", 101, []map[string]interface{}{
		{"severity": "LOW", "description": "phase 101 issue 1"},
		{"severity": "INFO", "description": "phase 101 issue 2"},
	})

	// Verify security ledger now has 4 entries (accumulated, not overwritten)
	if count := countLedgerEntries(t, s, "security"); count != 4 {
		t.Errorf("after phase 101: security entries = %d, want 4", count)
	}

	// Phase 102 writes findings to a different domain (quality)
	writeFindings(t, "quality", 102, []map[string]interface{}{
		{"severity": "HIGH", "description": "quality issue 1"},
		{"severity": "MEDIUM", "description": "quality issue 2"},
	})

	// Verify security still has 4 entries
	if count := countLedgerEntries(t, s, "security"); count != 4 {
		t.Errorf("after phase 102: security entries = %d, want 4", count)
	}

	// Verify quality has its entries
	if count := countLedgerEntries(t, s, "quality"); count != 2 {
		t.Errorf("after phase 102: quality entries = %d, want 2", count)
	}

	// Verify total unique domains with entries >= 2
	domainsWithEntries := 0
	for _, domain := range colony.DomainOrder {
		if countLedgerEntries(t, s, domain) > 0 {
			domainsWithEntries++
		}
	}
	if domainsWithEntries < 2 {
		t.Errorf("domains with entries = %d, want >= 2", domainsWithEntries)
	}

	// Verify IDs are sequential across phases
	entries := readLedger(t, s, "security")
	expectedIDs := []string{"sec-100-001", "sec-100-002", "sec-101-001", "sec-101-002"}
	for i, expected := range expectedIDs {
		if i >= len(entries) {
			t.Fatalf("expected %d entries, got %d", len(expectedIDs), len(entries))
		}
		if entries[i].ID != expected {
			t.Errorf("entry[%d].id = %q, want %q", i, entries[i].ID, expected)
		}
	}
}

// --- Test 3: Session Reset Survival ---

func TestReviewLedgerSessionResetSurvival(t *testing.T) {
	_, _, s := setupPersistenceTest(t)
	store = s

	// Get the store's base path for later reuse
	storeBasePath := s.BasePath()

	// Write findings to 2 domains using store instance A
	writeFindings(t, "security", 100, []map[string]interface{}{
		{"severity": "HIGH", "description": "session A security issue"},
	})
	writeFindings(t, "quality", 100, []map[string]interface{}{
		{"severity": "MEDIUM", "description": "session A quality issue"},
	})

	// Verify instance A can read them back
	if count := countLedgerEntries(t, s, "security"); count != 1 {
		t.Fatalf("instance A: security entries = %d, want 1", count)
	}
	if count := countLedgerEntries(t, s, "quality"); count != 1 {
		t.Fatalf("instance A: quality entries = %d, want 1", count)
	}

	// Simulate session end: discard store instance A
	store = nil
	s = nil

	// Create fresh store instance B pointing to the SAME directory
	s2, err := storage.NewStore(storeBasePath)
	if err != nil {
		t.Fatalf("create store instance B: %v", err)
	}
	store = s2

	// Read ledgers via store instance B
	secEntries := readLedger(t, s2, "security")
	qualEntries := readLedger(t, s2, "quality")

	// Verify all findings written by instance A are readable by instance B
	if len(secEntries) != 1 {
		t.Errorf("instance B: security entries = %d, want 1", len(secEntries))
	}
	if len(qualEntries) != 1 {
		t.Errorf("instance B: quality entries = %d, want 1", len(qualEntries))
	}

	// Verify no data loss occurred
	if len(secEntries) > 0 && secEntries[0].Description != "session A security issue" {
		t.Errorf("instance B: security description = %q, want %q", secEntries[0].Description, "session A security issue")
	}
	if len(qualEntries) > 0 && qualEntries[0].Description != "session A quality issue" {
		t.Errorf("instance B: quality description = %q, want %q", qualEntries[0].Description, "session A quality issue")
	}

	// Verify we can also write new entries with instance B (proving full read-write)
	writeFindings(t, "security", 101, []map[string]interface{}{
		{"severity": "LOW", "description": "session B security issue"},
	})

	// Verify accumulation still works after session reset
	secEntries = readLedger(t, s2, "security")
	if len(secEntries) != 2 {
		t.Errorf("after instance B write: security entries = %d, want 2", len(secEntries))
	}

	// Verify the ledger files actually exist on disk
	for _, domain := range []string{"security", "quality"} {
		ledgerPath := filepath.Join(storeBasePath, "reviews", domain, "ledger.json")
		if _, err := os.Stat(ledgerPath); os.IsNotExist(err) {
			t.Errorf("ledger file missing: %s", ledgerPath)
		}
	}
}
