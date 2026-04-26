package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

// setupReviewLedgerTest creates a fresh test environment for review ledger tests.
func setupReviewLedgerTest(t *testing.T) (*bytes.Buffer, *bytes.Buffer, *storage.Store) {
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

// --- review-ledger-write tests ---

func TestReviewLedgerWrite_Basic(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	findings := `[{"severity":"HIGH","description":"exposed secret","file":"auth.go","line":42}]`
	rootCmd.SetArgs([]string{
		"review-ledger-write",
		"--domain", "security",
		"--phase", "2",
		"--agent", "gatekeeper",
		"--findings", findings,
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", buf.String())
	}

	result := env["result"].(map[string]interface{})
	if result["written"] != true {
		t.Errorf("written = %v, want true", result["written"])
	}
	if result["domain"] != "security" {
		t.Errorf("domain = %v, want security", result["domain"])
	}

	summary := result["summary"].(map[string]interface{})
	if summary["total"] != float64(1) {
		t.Errorf("summary.total = %v, want 1", summary["total"])
	}
	if summary["open"] != float64(1) {
		t.Errorf("summary.open = %v, want 1", summary["open"])
	}

	// Verify the entry was written to disk
	var lf interface{}
	if err := s.LoadJSON("reviews/security/ledger.json", &lf); err != nil {
		t.Fatalf("ledger file not found: %v", err)
	}

	ledger := lf.(map[string]interface{})
	entries := ledger["entries"].([]interface{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0].(map[string]interface{})
	if entry["id"] != "sec-2-001" {
		t.Errorf("entry.id = %v, want sec-2-001", entry["id"])
	}
	if entry["status"] != "open" {
		t.Errorf("entry.status = %v, want open", entry["status"])
	}
}

func TestReviewLedgerWrite_MultipleFindings(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	findings := `[
		{"severity":"HIGH","description":"exposed secret","file":"auth.go","line":42},
		{"severity":"MEDIUM","description":"weak hash","file":"crypto.go","line":15},
		{"severity":"LOW","description":"verbose logging","file":"log.go","line":8}
	]`
	rootCmd.SetArgs([]string{
		"review-ledger-write",
		"--domain", "security",
		"--phase", "2",
		"--agent", "gatekeeper",
		"--findings", findings,
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", buf.String())
	}

	result := env["result"].(map[string]interface{})
	summary := result["summary"].(map[string]interface{})
	if summary["total"] != float64(3) {
		t.Errorf("summary.total = %v, want 3", summary["total"])
	}

	// Verify IDs are sequential
	var lf map[string]interface{}
	s.LoadJSON("reviews/security/ledger.json", &lf)
	entries := lf["entries"].([]interface{})
	expectedIDs := []string{"sec-2-001", "sec-2-002", "sec-2-003"}
	for i, e := range entries {
		entry := e.(map[string]interface{})
		if entry["id"] != expectedIDs[i] {
			t.Errorf("entry[%d].id = %v, want %s", i, entry["id"], expectedIDs[i])
		}
	}
}

func TestReviewLedgerWrite_DeterministicIDsAcrossWrites(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	// First write: 2 findings
	findings1 := `[{"severity":"HIGH","description":"issue1"},{"severity":"MEDIUM","description":"issue2"}]`
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "2",
		"--agent", "gatekeeper", "--findings", findings1,
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first write error: %v", err)
	}

	// Reset buffer for second write
	buf.Reset()

	// Second write: 1 finding
	findings2 := `[{"severity":"LOW","description":"issue3"}]`
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "2",
		"--agent", "gatekeeper", "--findings", findings2,
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second write error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", buf.String())
	}

	// Verify third entry got sec-2-003
	var lf map[string]interface{}
	s.LoadJSON("reviews/security/ledger.json", &lf)
	entries := lf["entries"].([]interface{})
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	lastEntry := entries[2].(map[string]interface{})
	if lastEntry["id"] != "sec-2-003" {
		t.Errorf("third entry.id = %v, want sec-2-003", lastEntry["id"])
	}
}

func TestReviewLedgerWrite_InvalidDomain(t *testing.T) {
	_, errBuf, s := setupReviewLedgerTest(t)
	store = s

	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "invalid-domain", "--phase", "2",
		"--findings", `[{"severity":"HIGH","description":"test"}]`,
	})
	rootCmd.Execute()

	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "invalid domain") {
		t.Errorf("expected 'invalid domain' error, got: %s", errOutput)
	}
}

func TestReviewLedgerWrite_AgentDomainValidation(t *testing.T) {
	buf, errBuf, s := setupReviewLedgerTest(t)
	store = s

	// gatekeeper -> security: should succeed
	findings := `[{"severity":"HIGH","description":"exposed secret"}]`
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "2",
		"--agent", "gatekeeper", "--findings", findings,
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("gatekeeper->security should succeed: %v", err)
	}
	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("gatekeeper->security should be ok:true, got: %s", buf.String())
	}

	// gatekeeper -> quality: should fail
	buf.Reset()
	errBuf.Reset()
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "quality", "--phase", "2",
		"--agent", "gatekeeper", "--findings", findings,
	})
	rootCmd.Execute()

	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %s", errOutput)
	}
}

func TestReviewLedgerWrite_NoAgentSkipsValidation(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	// No --agent flag: should succeed to any valid domain
	findings := `[{"severity":"HIGH","description":"test"}]`
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "quality", "--phase", "2",
		"--findings", findings,
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("write without agent should succeed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", buf.String())
	}
}

func TestReviewLedgerWrite_InvalidJSON(t *testing.T) {
	_, errBuf, s := setupReviewLedgerTest(t)
	store = s

	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "2",
		"--findings", `{not valid json}`,
	})
	rootCmd.Execute()

	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "invalid --findings JSON") {
		t.Errorf("expected 'invalid --findings JSON' error, got: %s", errOutput)
	}
}

func TestReviewLedgerWrite_PhaseRequired(t *testing.T) {
	_, errBuf, s := setupReviewLedgerTest(t)
	store = s

	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "0",
		"--findings", `[{"severity":"HIGH","description":"test"}]`,
	})
	rootCmd.Execute()

	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "--phase is required") {
		t.Errorf("expected '--phase is required' error, got: %s", errOutput)
	}
}

// --- review-ledger-read tests ---

func TestReviewLedgerRead_Basic(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	// Write 3 entries first
	findings := `[
		{"severity":"HIGH","description":"issue1"},
		{"severity":"MEDIUM","description":"issue2"},
		{"severity":"LOW","description":"issue3"}
	]`
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "2",
		"--agent", "gatekeeper", "--findings", findings,
	})
	rootCmd.Execute()

	// Read all entries
	buf.Reset()
	rootCmd.SetArgs([]string{"review-ledger-read", "--domain", "security"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("read error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", buf.String())
	}

	result := env["result"].(map[string]interface{})
	entries := result["entries"].([]interface{})
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestReviewLedgerRead_FilterByStatus(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	// Write 3 entries
	findings := `[
		{"severity":"HIGH","description":"issue1"},
		{"severity":"MEDIUM","description":"issue2"},
		{"severity":"LOW","description":"issue3"}
	]`
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "2",
		"--agent", "gatekeeper", "--findings", findings,
	})
	rootCmd.Execute()

	// Resolve one entry
	rootCmd.SetArgs([]string{
		"review-ledger-resolve", "--domain", "security", "--id", "sec-2-001",
	})
	rootCmd.Execute()

	// Read open entries
	buf.Reset()
	rootCmd.SetArgs([]string{
		"review-ledger-read", "--domain", "security", "--status", "open",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("read error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	entries := result["entries"].([]interface{})
	if len(entries) != 2 {
		t.Errorf("expected 2 open entries, got %d", len(entries))
	}
}

func TestReviewLedgerRead_FilterByPhase(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	// Write to phase 2
	findings2 := `[{"severity":"HIGH","description":"issue1"}]`
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "2",
		"--agent", "gatekeeper", "--findings", findings2,
	})
	rootCmd.Execute()

	// Write to phase 3
	findings3 := `[{"severity":"LOW","description":"issue2"}]`
	buf.Reset()
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "3",
		"--agent", "gatekeeper", "--findings", findings3,
	})
	rootCmd.Execute()

	// Read phase 2 only
	buf.Reset()
	rootCmd.SetArgs([]string{
		"review-ledger-read", "--domain", "security", "--phase", "2",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("read error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	entries := result["entries"].([]interface{})
	if len(entries) != 1 {
		t.Errorf("expected 1 phase-2 entry, got %d", len(entries))
	}
	entry := entries[0].(map[string]interface{})
	if entry["phase"] != float64(2) {
		t.Errorf("entry.phase = %v, want 2", entry["phase"])
	}
}

func TestReviewLedgerRead_EmptyDomain(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	rootCmd.SetArgs([]string{"review-ledger-read", "--domain", "quality"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("read error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", buf.String())
	}

	result := env["result"].(map[string]interface{})
	entries := result["entries"].([]interface{})
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

// --- review-ledger-summary tests ---

func TestReviewLedgerSummary_MultipleDomains(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	// Write to security
	secFindings := `[{"severity":"HIGH","description":"sec issue"}]`
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "2",
		"--agent", "gatekeeper", "--findings", secFindings,
	})
	rootCmd.Execute()

	// Write to quality
	qualFindings := `[{"severity":"MEDIUM","description":"qual issue"}]`
	buf.Reset()
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "quality", "--phase", "2",
		"--agent", "auditor", "--findings", qualFindings,
	})
	rootCmd.Execute()

	// Get summary
	buf.Reset()
	rootCmd.SetArgs([]string{"review-ledger-summary"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("summary error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", buf.String())
	}

	result := env["result"].(map[string]interface{})
	domains := result["domains"].([]interface{})
	if len(domains) != 2 {
		t.Errorf("expected 2 domain summaries, got %d", len(domains))
	}
}

func TestReviewLedgerSummary_NoLedgers(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	rootCmd.SetArgs([]string{"review-ledger-summary"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("summary error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", buf.String())
	}

	result := env["result"].(map[string]interface{})
	domains := result["domains"].([]interface{})
	if len(domains) != 0 {
		t.Errorf("expected 0 domains, got %d", len(domains))
	}
}

// --- review-ledger-resolve tests ---

func TestReviewLedgerResolve_Basic(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	// Write an entry
	findings := `[{"severity":"HIGH","description":"exposed secret"}]`
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "2",
		"--agent", "gatekeeper", "--findings", findings,
	})
	rootCmd.Execute()

	// Resolve it
	buf.Reset()
	rootCmd.SetArgs([]string{
		"review-ledger-resolve", "--domain", "security", "--id", "sec-2-001",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", buf.String())
	}

	result := env["result"].(map[string]interface{})
	if result["resolved"] != true {
		t.Errorf("resolved = %v, want true", result["resolved"])
	}
	if result["id"] != "sec-2-001" {
		t.Errorf("id = %v, want sec-2-001", result["id"])
	}

	// Verify on disk
	var lf map[string]interface{}
	s.LoadJSON("reviews/security/ledger.json", &lf)
	entries := lf["entries"].([]interface{})
	entry := entries[0].(map[string]interface{})
	if entry["status"] != "resolved" {
		t.Errorf("entry.status = %v, want resolved", entry["status"])
	}
	if entry["resolved_at"] == nil || entry["resolved_at"] == "" {
		t.Errorf("entry.resolved_at should be set, got: %v", entry["resolved_at"])
	}
}

func TestReviewLedgerResolve_NotFound(t *testing.T) {
	_, errBuf, s := setupReviewLedgerTest(t)
	store = s

	rootCmd.SetArgs([]string{
		"review-ledger-resolve", "--domain", "security", "--id", "sec-2-999",
	})
	rootCmd.Execute()

	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "not found") {
		t.Errorf("expected 'not found' error, got: %s", errOutput)
	}
}

func TestReviewLedgerResolve_UpdatesSummary(t *testing.T) {
	buf, _, s := setupReviewLedgerTest(t)
	store = s

	// Write 3 entries
	findings := `[
		{"severity":"HIGH","description":"issue1"},
		{"severity":"MEDIUM","description":"issue2"},
		{"severity":"LOW","description":"issue3"}
	]`
	rootCmd.SetArgs([]string{
		"review-ledger-write", "--domain", "security", "--phase", "2",
		"--agent", "gatekeeper", "--findings", findings,
	})
	rootCmd.Execute()

	// Resolve one
	buf.Reset()
	rootCmd.SetArgs([]string{
		"review-ledger-resolve", "--domain", "security", "--id", "sec-2-002",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	// Read to verify summary
	buf.Reset()
	rootCmd.SetArgs([]string{"review-ledger-read", "--domain", "security"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("read error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	summary := result["summary"].(map[string]interface{})
	if summary["total"] != float64(3) {
		t.Errorf("summary.total = %v, want 3", summary["total"])
	}
	if summary["open"] != float64(2) {
		t.Errorf("summary.open = %v, want 2", summary["open"])
	}
	if summary["resolved"] != float64(1) {
		t.Errorf("summary.resolved = %v, want 1", summary["resolved"])
	}
}
