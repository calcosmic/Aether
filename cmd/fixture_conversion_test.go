package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// --- Task 1: the versioned fixture bank and its schema ---

// newFixtureBankTestRoot creates an isolated temp root carrying a copy of
// the real, committed fixture bank schema (so schema-validation tests
// exercise the actual authored schema, not a test-only stand-in), redirects
// fixtureBankRepoRootOverride to it for the duration of the test, and
// restores the previous override on cleanup. Writes/loads under this root
// never touch the real committed cmd/testdata/fixture-bank/v1/bank.json.
func newFixtureBankTestRoot(t *testing.T) string {
	t.Helper()
	realRoot := findTestModuleRoot(t)
	realSchema, err := os.ReadFile(filepath.Join(realRoot, fixtureBankSchemaPath))
	if err != nil {
		t.Fatalf("read real fixture bank schema: %v", err)
	}

	tmpRoot := t.TempDir()
	schemaDir := filepath.Join(tmpRoot, filepath.Dir(fixtureBankSchemaPath))
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("mkdir schema dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpRoot, fixtureBankSchemaPath), realSchema, 0o644); err != nil {
		t.Fatalf("write test schema copy: %v", err)
	}

	prev := fixtureBankRepoRootOverride
	fixtureBankRepoRootOverride = tmpRoot
	t.Cleanup(func() { fixtureBankRepoRootOverride = prev })
	return tmpRoot
}

// validFixtureDocumentForTest returns a fixture-shaped map[string]any
// carrying every field the schema requires, so individual tests can delete
// exactly the field under test rather than retyping the whole shape.
func validFixtureDocumentForTest() map[string]any {
	return map[string]any{
		"id":    "fixture-abc123def456",
		"title": "example fixture",
		"failing_baseline": map[string]any{
			"description": "the state in which this fixture's check fails",
			"reference":   "midden:midden_1700000000_123",
		},
		"invariant":         "the thing that must stay true",
		"allowed_mutations": []any{"wording"},
		"provenance": []any{
			map[string]any{"kind": "failure_log", "identifier": "midden_1700000000_123"},
		},
		"privacy":        "repo_local",
		"severity":       "medium",
		"content_digest": strings.Repeat("a", 64),
	}
}

func fixtureBankDocumentForTest(fixtures ...map[string]any) map[string]any {
	items := make([]any, 0, len(fixtures))
	for _, f := range fixtures {
		items = append(items, f)
	}
	return map[string]any{
		"schema_version": "fixture-bank/v1",
		"fixtures":       items,
	}
}

func TestFixtureBankSchemaRejectsIncompleteFixtures(t *testing.T) {
	newFixtureBankTestRoot(t)

	requiredFields := []string{
		"id", "title", "failing_baseline", "invariant",
		"allowed_mutations", "provenance", "privacy", "severity", "content_digest",
	}

	for _, field := range requiredFields {
		t.Run("missing_"+field, func(t *testing.T) {
			fixture := validFixtureDocumentForTest()
			delete(fixture, field)
			doc := fixtureBankDocumentForTest(fixture)
			messages, err := validateFixtureBankDocument(doc)
			if err != nil {
				t.Fatalf("validate: %v", err)
			}
			if len(messages) == 0 {
				t.Fatalf("expected a violation for missing %q, got none", field)
			}
			joined := strings.Join(messages, " | ")
			if !strings.Contains(joined, field) {
				t.Fatalf("violation messages %v do not name the missing field %q", messages, field)
			}
		})
	}

	t.Run("id_pattern_violation_names_the_identifier", func(t *testing.T) {
		fixture := validFixtureDocumentForTest()
		fixture["id"] = "BAD ID!!"
		doc := fixtureBankDocumentForTest(fixture)
		messages, err := validateFixtureBankDocument(doc)
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		if len(messages) == 0 {
			t.Fatal("expected a violation for a malformed id, got none")
		}
		joined := strings.Join(messages, " | ")
		if !strings.Contains(joined, "BAD ID!!") {
			t.Fatalf("violation messages %v do not name the offending identifier %q", messages, "BAD ID!!")
		}
	})

	t.Run("allowed_mutations_undeclared_value_lists_declared_values", func(t *testing.T) {
		fixture := validFixtureDocumentForTest()
		fixture["allowed_mutations"] = []any{"rewrite-everything"}
		doc := fixtureBankDocumentForTest(fixture)
		messages, err := validateFixtureBankDocument(doc)
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		if len(messages) == 0 {
			t.Fatal("expected a violation for an undeclared allowed_mutations value, got none")
		}
		joined := strings.Join(messages, " | ")
		for _, want := range fixtureAllowedMutationNames() {
			if !strings.Contains(joined, want) {
				t.Fatalf("violation messages %v do not list declared value %q", messages, want)
			}
		}
	})
}

func TestFixtureRetirementRequiresASuccessor(t *testing.T) {
	newFixtureBankTestRoot(t)

	fixture := validFixtureDocumentForTest()
	fixture["retired_by"] = "superseded by fixture-successor-000000000000"
	// Deliberately no successor_id.
	doc := fixtureBankDocumentForTest(fixture)

	messages, err := validateFixtureBankDocument(doc)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("expected retired_by without successor_id to be refused, got no violation")
	}

	t.Run("both_present_is_valid", func(t *testing.T) {
		fixture := validFixtureDocumentForTest()
		fixture["retired_by"] = "superseded by fixture-successor-000000000000"
		fixture["successor_id"] = "fixture-successor-000000000000"
		doc := fixtureBankDocumentForTest(fixture)
		messages, err := validateFixtureBankDocument(doc)
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		if len(messages) != 0 {
			t.Fatalf("expected a fixture with both retired_by and successor_id to validate, got: %v", messages)
		}
	})
}

func TestFixtureVocabulariesAreClosed(t *testing.T) {
	newFixtureBankTestRoot(t)
	root, err := fixtureBankRepoRoot()
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, fixtureBankSchemaPath))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	defs, ok := schema["$defs"].(map[string]any)
	if !ok {
		t.Fatal("schema has no $defs")
	}
	fixtureDef, ok := defs["fixture"].(map[string]any)
	if !ok {
		t.Fatal("schema has no $defs.fixture")
	}
	fixtureProps, ok := fixtureDef["properties"].(map[string]any)
	if !ok {
		t.Fatal("$defs.fixture has no properties")
	}
	provenanceDef, ok := defs["provenance_ref"].(map[string]any)
	if !ok {
		t.Fatal("schema has no $defs.provenance_ref")
	}
	provenanceProps, ok := provenanceDef["properties"].(map[string]any)
	if !ok {
		t.Fatal("$defs.provenance_ref has no properties")
	}

	cases := []struct {
		name       string
		schemaEnum []any
		goVocab    []string
	}{
		{
			name:       "allowed_mutations",
			schemaEnum: schemaEnumAt(t, fixtureProps, "allowed_mutations", "items", "enum"),
			goVocab:    fixtureAllowedMutationNames(),
		},
		{
			name:       "provenance kind",
			schemaEnum: schemaEnumAt(t, provenanceProps, "kind", "enum"),
			goVocab:    fixtureProvenanceKindNames(),
		},
		{
			name:       "privacy",
			schemaEnum: schemaEnumAt(t, fixtureProps, "privacy", "enum"),
			goVocab:    fixturePrivacyNames(),
		},
		{
			name:       "severity",
			schemaEnum: schemaEnumAt(t, fixtureProps, "severity", "enum"),
			goVocab:    fixtureSeverityNames(),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if len(c.schemaEnum) != len(c.goVocab) {
				t.Fatalf("%s: schema declares %d values %v, Go vocabulary declares %d values %v -- these must agree",
					c.name, len(c.schemaEnum), c.schemaEnum, len(c.goVocab), c.goVocab)
			}
			want := map[string]bool{}
			for _, v := range c.goVocab {
				want[v] = true
			}
			for _, se := range c.schemaEnum {
				s, ok := se.(string)
				if !ok {
					t.Fatalf("%s: schema enum member %v is not a string", c.name, se)
				}
				if !want[s] {
					t.Fatalf("%s: schema declares %q, which the Go vocabulary %v does not", c.name, s, c.goVocab)
				}
			}
		})
	}

	// A value outside the declared vocabulary is refused by the
	// declared-membership predicate, by name.
	if fixtureAllowedMutationDeclared(fixtureAllowedMutation("rewrite-everything")) {
		t.Fatal("fixtureAllowedMutationDeclared accepted an undeclared value")
	}
	if fixtureProvenanceKindDeclared(fixtureProvenanceKind("guesswork")) {
		t.Fatal("fixtureProvenanceKindDeclared accepted an undeclared value")
	}
	if fixturePrivacyDeclared(fixturePrivacy("classified")) {
		t.Fatal("fixturePrivacyDeclared accepted an undeclared value")
	}
	if fixtureSeverityDeclared(fixtureSeverity("catastrophic")) {
		t.Fatal("fixtureSeverityDeclared accepted an undeclared value")
	}
}

// schemaEnumAt walks a chain of map keys under root and returns the final
// "enum" array as []any, failing the test if any hop is missing or the
// wrong shape.
func schemaEnumAt(t *testing.T, root map[string]any, path ...string) []any {
	t.Helper()
	var cur any = root
	for _, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("schema path %v: expected object at %q, got %T", path, key, cur)
		}
		next, ok := m[key]
		if !ok {
			t.Fatalf("schema path %v: missing key %q", path, key)
		}
		cur = next
	}
	arr, ok := cur.([]any)
	if !ok {
		t.Fatalf("schema path %v: expected an array, got %T", path, cur)
	}
	return arr
}

func TestEmptyFixtureBankIsValid(t *testing.T) {
	newFixtureBankTestRoot(t)
	doc := map[string]any{
		"schema_version": "fixture-bank/v1",
		"fixtures":       []any{},
	}
	messages, err := validateFixtureBankDocument(doc)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected an empty fixture bank to validate, got: %v", messages)
	}
}

func TestFixtureBankOrderingIsTotalAndStable(t *testing.T) {
	bank := regressionFixtureBank{
		SchemaVersion: fixtureBankSchemaVersion,
		Fixtures: []regressionFixture{
			{ID: "fixture-c00000000000"},
			{ID: "fixture-a00000000000"},
			{ID: "fixture-b00000000000"},
		},
	}

	first := fixtureBankSortedByID(bank)
	second := fixtureBankSortedByID(bank)

	wantOrder := []string{"fixture-a00000000000", "fixture-b00000000000", "fixture-c00000000000"}
	if !sort.SliceIsSorted(first, func(i, j int) bool { return first[i].ID < first[j].ID }) {
		t.Fatalf("first sort is not ascending: %v", idsOf(first))
	}
	if got := idsOf(first); !equalStrings(got, wantOrder) {
		t.Fatalf("first sort = %v, want %v", got, wantOrder)
	}
	if got := idsOf(second); !equalStrings(got, wantOrder) {
		t.Fatalf("second sort = %v, want %v (identical to the first)", got, wantOrder)
	}

	root := newFixtureBankTestRoot(t)
	bankPath := filepath.Join(root, fixtureBankPath)
	if err := os.MkdirAll(filepath.Dir(bankPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	encoded, err := json.MarshalIndent(regressionFixtureBank{SchemaVersion: fixtureBankSchemaVersion, Fixtures: []regressionFixture{
		{ID: "fixture-c00000000000"},
		{ID: "fixture-a00000000000"},
	}}, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(bankPath, encoded, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	loadedOnce, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load once: %v", err)
	}
	loadedTwice, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load twice: %v", err)
	}
	if got := idsOf(loadedOnce.Fixtures); !equalStrings(got, []string{"fixture-a00000000000", "fixture-c00000000000"}) {
		t.Fatalf("first load = %v, want ascending order", got)
	}
	if got1, got2 := idsOf(loadedOnce.Fixtures), idsOf(loadedTwice.Fixtures); !equalStrings(got1, got2) {
		t.Fatalf("loading the bank twice produced different orders: %v vs %v", got1, got2)
	}
}

func idsOf(fixtures []regressionFixture) []string {
	ids := make([]string, len(fixtures))
	for i, f := range fixtures {
		ids[i] = f.ID
	}
	return ids
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- Task 2: convert confirmed incidents, refusing everything unconfirmed ---

// seedMiddenEntryForTest appends one real failure-log entry, in the
// failure log's own real shape, directly through the store -- so
// confirmation-gate tests drive the real reader against a real seeded
// entry, never a hand-built confirmedIncident.
func seedMiddenEntryForTest(t *testing.T, s *storage.Store, entry colony.MiddenEntry) {
	t.Helper()
	var mf colony.MiddenFile
	err := s.UpdateJSONAtomically(middenCanonicalPath, &mf, func() error {
		if mf.Entries == nil {
			mf.Entries = []colony.MiddenEntry{}
		}
		if mf.Version == "" {
			mf.Version = "1.0.0"
		}
		mf.Entries = append(mf.Entries, entry)
		return nil
	})
	if err != nil {
		t.Fatalf("seed midden entry: %v", err)
	}
}

func fixtureBoolPtr(b bool) *bool { return &b }

func writeAuditFindingsFixture(t *testing.T, path string, confirmed []map[string]any) {
	t.Helper()
	doc := map[string]any{"confirmed": confirmed}
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal audit findings fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatalf("write audit findings fixture: %v", err)
	}
}

func writeDefectRegisterFixture(t *testing.T, path string, entries []map[string]any) {
	t.Helper()
	encoded, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatalf("marshal defect register fixture: %v", err)
	}
	var b strings.Builder
	b.WriteString("# Broken Windows Ledger\n\n")
	b.WriteString("| id | phase | kind | file | description | status |\n")
	b.WriteString("|----|-------|------|------|-------------|--------|\n")
	b.WriteString("\n```json\n")
	b.Write(encoded)
	b.WriteString("\n```\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write defect register fixture: %v", err)
	}
}

func writePhaseReportFixture(t *testing.T, phasesDir, phaseName, table string) {
	t.Helper()
	dir := filepath.Join(phasesDir, phaseName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "# Phase Verification\n\n## Test evidence\n\n" + table + "\n"
	path := filepath.Join(dir, phaseName+"-VERIFICATION.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write phase report fixture: %v", err)
	}
}

func TestUnconfirmedIncidentProducesNoFixture(t *testing.T) {
	t.Run("failure_log", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		seedMiddenEntryForTest(t, s, colony.MiddenEntry{
			ID: "midden_not_reviewed", Category: "deviation", Source: "test",
			Message: "not reviewed yet", Reviewed: false,
		})
		seedMiddenEntryForTest(t, s, colony.MiddenEntry{
			ID: "midden_not_acknowledged", Category: "deviation", Source: "test",
			Message: "reviewed but not acknowledged", Reviewed: true, Acknowledged: boolPtr(false),
		})

		result, err := confirmedIncidentsFromFailureLog()
		if err != nil {
			t.Fatalf("confirmedIncidentsFromFailureLog: %v", err)
		}
		if len(result.Incidents) != 0 {
			t.Fatalf("expected zero confirmed incidents, got %d: %+v", len(result.Incidents), result.Incidents)
		}
		if len(result.Refused) != 2 {
			t.Fatalf("expected 2 refusals, got %d: %+v", len(result.Refused), result.Refused)
		}
		foundReviewed, foundAcknowledged := false, false
		for _, r := range result.Refused {
			if r.Identifier == "midden_not_reviewed" && strings.Contains(r.Reason, "reviewed") {
				foundReviewed = true
			}
			if r.Identifier == "midden_not_acknowledged" && strings.Contains(r.Reason, "acknowledged") {
				foundAcknowledged = true
			}
		}
		if !foundReviewed {
			t.Errorf("no refusal named midden_not_reviewed's missing confirmation fact (reviewed); got %+v", result.Refused)
		}
		if !foundAcknowledged {
			t.Errorf("no refusal named midden_not_acknowledged's missing confirmation fact (acknowledged); got %+v", result.Refused)
		}
	})

	t.Run("audit_finding", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "audit.json")
		writeAuditFindingsFixture(t, path, []map[string]any{
			{"subject": "unverified claim", "status": "dropped", "claim": "not checked", "severity": "low", "verified": false},
		})
		result, err := confirmedIncidentsFromAuditFindings(path)
		if err != nil {
			t.Fatalf("confirmedIncidentsFromAuditFindings: %v", err)
		}
		if len(result.Incidents) != 0 {
			t.Fatalf("expected zero confirmed incidents, got %+v", result.Incidents)
		}
		if len(result.Refused) != 1 || !strings.Contains(result.Refused[0].Reason, "verified") {
			t.Fatalf("expected one refusal naming 'verified', got %+v", result.Refused)
		}
	})

	t.Run("defect_register", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "WINDOWS.md")
		writeDefectRegisterFixture(t, path, []map[string]any{
			{"id": 1, "kind": "deviation", "phase": "204", "file": "cmd/x.go", "description": "still open", "status": "open", "recorded_at": "2026-09-14T00:00:00Z"},
		})
		result, err := confirmedIncidentsFromDefectRegister(path)
		if err != nil {
			t.Fatalf("confirmedIncidentsFromDefectRegister: %v", err)
		}
		if len(result.Incidents) != 0 {
			t.Fatalf("expected zero confirmed incidents, got %+v", result.Incidents)
		}
		if len(result.Refused) != 1 || !strings.Contains(result.Refused[0].Reason, "fixed") {
			t.Fatalf("expected one refusal naming 'fixed', got %+v", result.Refused)
		}
	})

	t.Run("phase_report_no_commit", func(t *testing.T) {
		root := t.TempDir()
		phasesDir := filepath.Join(root, "phases")
		table := "| Fixed | Cause | Commit |\n|---|---|---|\n| TestSomething | a cause | no commit here |\n"
		writePhaseReportFixture(t, phasesDir, "204-test-phase", table)
		result, err := confirmedIncidentsFromPhaseReports(phasesDir)
		if err != nil {
			t.Fatalf("confirmedIncidentsFromPhaseReports: %v", err)
		}
		if len(result.Incidents) != 0 {
			t.Fatalf("expected zero confirmed incidents, got %+v", result.Incidents)
		}
		if len(result.Refused) != 1 || !strings.Contains(result.Refused[0].Reason, "commit") {
			t.Fatalf("expected one refusal naming 'commit', got %+v", result.Refused)
		}
	})
}

func TestSameIncidentTwiceProducesOneFixture(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	newFixtureBankTestRoot(t)

	seedMiddenEntryForTest(t, s, colony.MiddenEntry{
		ID: "midden_confirmed_once", Category: "deviation", Source: "test",
		Message: "a real confirmed failure", Reviewed: true, Acknowledged: boolPtr(true),
	})

	runOnce := func() []byte {
		result, err := confirmedIncidentsFromFailureLog()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		bank, _, err := convertConfirmedIncidentsToFixtures(result.Incidents)
		if err != nil {
			t.Fatalf("convert: %v", err)
		}
		if err := writeFixtureBank(bank); err != nil {
			t.Fatalf("write: %v", err)
		}
		root, _ := fixtureBankRepoRoot()
		data, err := os.ReadFile(filepath.Join(root, fixtureBankPath))
		if err != nil {
			t.Fatalf("read bank file: %v", err)
		}
		return data
	}

	first := runOnce()
	second := runOnce()

	if string(first) != string(second) {
		t.Fatalf("re-running conversion against the same confirmed incident changed the bank file:\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(bank.Fixtures) != 1 {
		t.Fatalf("expected exactly one fixture after feeding the same incident twice, got %d", len(bank.Fixtures))
	}
}

func TestTwoIncidentsWithOneDigestShareAFixture(t *testing.T) {
	incidents := []confirmedIncident{
		{SourceKind: fixtureProvenanceFailureLog, SourceIdentifier: "midden_a", Title: "same content", Detail: "identical detail text"},
		{SourceKind: fixtureProvenanceAuditFinding, SourceIdentifier: "confirmed[7]", Title: "same content", Detail: "identical detail text"},
	}
	bank, summary, err := convertConfirmedIncidentsToFixtures(incidents)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(bank.Fixtures) != 1 {
		t.Fatalf("expected exactly one fixture for two identically-content incidents, got %d", len(bank.Fixtures))
	}
	if summary.Deduplicated != 1 {
		t.Fatalf("expected summary.Deduplicated = 1, got %d", summary.Deduplicated)
	}
	if len(bank.Fixtures[0].Provenance) != 2 {
		t.Fatalf("expected the shared fixture to carry 2 provenance references, got %d: %+v", len(bank.Fixtures[0].Provenance), bank.Fixtures[0].Provenance)
	}
}

func TestUnsanitisableIncidentIsDroppedNotStored(t *testing.T) {
	incidents := []confirmedIncident{
		{SourceKind: fixtureProvenanceFailureLog, SourceIdentifier: "midden_bad", Title: "bad incident", Detail: "contains <task> a structural tag"},
	}
	bank, summary, err := convertConfirmedIncidentsToFixtures(incidents)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(bank.Fixtures) != 0 {
		t.Fatalf("expected the unsanitisable incident to produce no fixture, got %d", len(bank.Fixtures))
	}
	if len(summary.Dropped) != 1 || summary.Dropped[0].Identifier != "midden_bad" {
		t.Fatalf("expected exactly one dropped entry naming midden_bad, got %+v", summary.Dropped)
	}
}

func TestRetirementNamesASuccessorAndKeepsTheFixture(t *testing.T) {
	newFixtureBankTestRoot(t)

	seed := regressionFixtureBank{
		SchemaVersion: fixtureBankSchemaVersion,
		Fixtures: []regressionFixture{
			{
				ID: "fixture-000000000001", Title: "old", Invariant: "must hold",
				FailingBaseline:  regressionFixtureFailingBaseline{Description: "d", Reference: "r"},
				AllowedMutations: []fixtureAllowedMutation{fixtureMutationWording},
				Provenance:       []regressionFixtureProvenance{{Kind: fixtureProvenanceFailureLog, Identifier: "midden_1"}},
				Privacy:          fixturePrivacyRepoLocal, Severity: fixtureSeverityMedium,
				ContentDigest: strings.Repeat("a", 64),
			},
			{
				ID: "fixture-000000000002", Title: "new", Invariant: "must also hold",
				FailingBaseline:  regressionFixtureFailingBaseline{Description: "d2", Reference: "r2"},
				AllowedMutations: []fixtureAllowedMutation{fixtureMutationWording},
				Provenance:       []regressionFixtureProvenance{{Kind: fixtureProvenanceFailureLog, Identifier: "midden_2"}},
				Privacy:          fixturePrivacyRepoLocal, Severity: fixtureSeverityMedium,
				ContentDigest: strings.Repeat("b", 64),
			},
		},
	}
	if err := writeFixtureBank(seed); err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Run("unknown_successor_is_refused", func(t *testing.T) {
		if err := retireFixture("fixture-000000000001", "fixture-does-not-exist", "superseded"); err == nil {
			t.Fatal("expected retiring against an unknown successor to be refused")
		}
	})

	if err := retireFixture("fixture-000000000001", "fixture-000000000002", "superseded by the newer fixture"); err != nil {
		t.Fatalf("retire: %v", err)
	}

	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(bank.Fixtures) != 2 {
		t.Fatalf("expected retirement to keep the fixture count at 2, got %d", len(bank.Fixtures))
	}
	var retired *regressionFixture
	for i := range bank.Fixtures {
		if bank.Fixtures[i].ID == "fixture-000000000001" {
			retired = &bank.Fixtures[i]
		}
	}
	if retired == nil {
		t.Fatal("retired fixture is missing from the bank -- retirement must never delete")
	}
	if retired.SuccessorID != "fixture-000000000002" {
		t.Fatalf("retired.SuccessorID = %q, want fixture-000000000002", retired.SuccessorID)
	}
	if retired.RetiredBy == "" {
		t.Fatal("retired.RetiredBy is empty")
	}
}

func TestEmptyIncidentSetWritesAnEmptyBank(t *testing.T) {
	newFixtureBankTestRoot(t)
	bank, summary, err := convertConfirmedIncidentsToFixtures(nil)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(bank.Fixtures) != 0 {
		t.Fatalf("expected an empty bank, got %d fixtures", len(bank.Fixtures))
	}
	if summary.FinalFixtureCount != 0 {
		t.Fatalf("summary.FinalFixtureCount = %d, want 0", summary.FinalFixtureCount)
	}
	if err := writeFixtureBank(bank); err != nil {
		t.Fatalf("write: %v", err)
	}
	loaded, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Fixtures) != 0 {
		t.Fatalf("expected an empty bank on disk, got %d fixtures", len(loaded.Fixtures))
	}
}

// TestFixtureBankHasOneWriter proves, by AST scan, that no function other
// than writeFixtureBank writes cmd/testdata/fixture-bank/v1/bank.json.
func TestFixtureBankHasOneWriter(t *testing.T) {
	repoRoot := findTestModuleRoot(t)
	violations := scanForFixtureBankWritersOutsideWriteFixtureBank(t, filepath.Join(repoRoot, "cmd"))
	if len(violations) != 0 {
		t.Fatalf("found a non-test Go function writing the fixture bank path outside writeFixtureBank:\n%s", strings.Join(violations, "\n"))
	}

	t.Run("a synthetic sneaky writer is caught by name", func(t *testing.T) {
		fixtureSrc := `package cmd

import "os"

func sneakyFixtureBankWriter(payload []byte) error {
	return os.WriteFile("cmd/testdata/fixture-bank/v1/bank.json", payload, 0644)
}
`
		violations := fixtureBankWriterViolationsInSource(t, "fixture_sneaky_writer.go", fixtureSrc)
		if len(violations) == 0 {
			t.Fatal("scanner failed to detect a synthetic write into the fixture bank path")
		}
		if !strings.Contains(violations[0], "sneakyFixtureBankWriter") {
			t.Fatalf("violation %q does not name the offending function sneakyFixtureBankWriter", violations[0])
		}
	})
}

func scanForFixtureBankWritersOutsideWriteFixtureBank(t *testing.T, dir string) []string {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	fset := token.NewFileSet()
	var violations []string
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		violations = append(violations, fixtureBankWriterViolations(fset, file)...)
	}
	return violations
}

func fixtureBankWriterViolationsInSource(t *testing.T, filename, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Fatalf("parse fixture source: %v", err)
	}
	return fixtureBankWriterViolations(fset, file)
}

func fixtureBankWriterViolations(fset *token.FileSet, file *ast.File) []string {
	const bankBasename = "fixture-bank/v1/bank.json"
	var violations []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if fn.Name.Name == "writeFixtureBank" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, arg := range call.Args {
				lit, ok := arg.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					continue
				}
				if strings.Contains(value, bankBasename) {
					pos := fset.Position(call.Pos())
					violations = append(violations, fmt.Sprintf("%s:%d in func %s references %q", pos.Filename, pos.Line, fn.Name.Name, bankBasename))
				}
			}
			return true
		})
	}
	return violations
}

// --- Task 3: seed the bank from this repository's own confirmed failures ---

// realConfirmedIncidents reads this repository's own real confirmed
// incidents through the four production readers, exactly as the committed
// seed was generated -- used by both TestSeededBankIsReproducible and
// TestSeededBankFixturesAllCiteRealProvenance so neither test can drift
// from how the bank was actually built.
func realConfirmedIncidents(t *testing.T, repoRoot string) (failureLog, audit, defects, phaseReports fixtureIncidentReadResult) {
	t.Helper()
	saveGlobals(t)
	s, err := storage.NewStore(filepath.Join(repoRoot, ".aether", "data"))
	if err != nil {
		t.Fatalf("open real store: %v", err)
	}
	store = s

	failureLog, err = confirmedIncidentsFromFailureLog()
	if err != nil {
		t.Fatalf("failure log: %v", err)
	}
	audit, err = confirmedIncidentsFromAuditFindings(filepath.Join(repoRoot, ".planning", "audits", "2026-08-30-whole-system-audit.json"))
	if err != nil {
		t.Fatalf("audit findings: %v", err)
	}
	defects, err = confirmedIncidentsFromDefectRegister(filepath.Join(repoRoot, ".planning", "WINDOWS.md"))
	if err != nil {
		t.Fatalf("defect register: %v", err)
	}
	phaseReports, err = confirmedIncidentsFromPhaseReports(filepath.Join(repoRoot, ".planning", "phases"))
	if err != nil {
		t.Fatalf("phase reports: %v", err)
	}
	return
}

func TestSeededBankValidatesAgainstItsSchema(t *testing.T) {
	repoRoot := findTestModuleRoot(t)
	raw, err := os.ReadFile(filepath.Join(repoRoot, fixtureBankPath))
	if err != nil {
		t.Fatalf("read committed bank: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal committed bank: %v", err)
	}

	prev := fixtureBankRepoRootOverride
	fixtureBankRepoRootOverride = repoRoot
	t.Cleanup(func() { fixtureBankRepoRootOverride = prev })

	messages, err := validateFixtureBankDocument(doc)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("committed fixture bank fails its own schema: %v", messages)
	}

	var typed regressionFixtureBank
	if err := json.Unmarshal(raw, &typed); err != nil {
		t.Fatalf("unmarshal into regressionFixtureBank: %v", err)
	}
	if len(typed.Fixtures) == 0 {
		t.Fatal("committed fixture bank has zero fixtures -- expected a real seeded bank")
	}
}

func TestSeededBankFixturesAllCiteRealProvenance(t *testing.T) {
	repoRoot := findTestModuleRoot(t)
	raw, err := os.ReadFile(filepath.Join(repoRoot, fixtureBankPath))
	if err != nil {
		t.Fatalf("read committed bank: %v", err)
	}
	var bank regressionFixtureBank
	if err := json.Unmarshal(raw, &bank); err != nil {
		t.Fatalf("unmarshal committed bank: %v", err)
	}

	failureLog, _, defects, phaseReports := realConfirmedIncidents(t, repoRoot)

	failureLogIDs := map[string]bool{}
	for _, inc := range failureLog.Incidents {
		failureLogIDs[inc.SourceIdentifier] = true
	}
	var auditFile fixtureAuditFindingsFile
	auditRaw, err := os.ReadFile(filepath.Join(repoRoot, ".planning", "audits", "2026-08-30-whole-system-audit.json"))
	if err != nil {
		t.Fatalf("read audit findings: %v", err)
	}
	if err := json.Unmarshal(auditRaw, &auditFile); err != nil {
		t.Fatalf("unmarshal audit findings: %v", err)
	}
	defectIDs := map[string]bool{}
	for _, inc := range defects.Incidents {
		defectIDs[inc.SourceIdentifier] = true
	}
	phaseReportIDs := map[string]bool{}
	for _, inc := range phaseReports.Incidents {
		phaseReportIDs[inc.SourceIdentifier] = true
	}

	kindCounts := map[fixtureProvenanceKind]int{}
	emptyCount := 0
	for _, fixture := range bank.Fixtures {
		if len(fixture.Provenance) == 0 {
			emptyCount++
			continue
		}
		for _, ref := range fixture.Provenance {
			kindCounts[ref.Kind]++
			switch ref.Kind {
			case fixtureProvenanceFailureLog:
				if !failureLogIDs[ref.Identifier] {
					t.Errorf("fixture %s cites failure_log identifier %q, which is not a currently confirmed failure-log entry", fixture.ID, ref.Identifier)
				}
			case fixtureProvenanceAuditFinding:
				var idx int
				if _, err := fmt.Sscanf(ref.Identifier, "confirmed[%d]", &idx); err != nil {
					t.Errorf("fixture %s cites malformed audit_finding identifier %q", fixture.ID, ref.Identifier)
					continue
				}
				if idx < 0 || idx >= len(auditFile.Confirmed) {
					t.Errorf("fixture %s cites audit_finding identifier %q, out of range for a %d-entry confirmed array", fixture.ID, ref.Identifier, len(auditFile.Confirmed))
					continue
				}
				if !auditFile.Confirmed[idx].Verified {
					t.Errorf("fixture %s cites audit_finding identifier %q, which is not marked verified in the source", fixture.ID, ref.Identifier)
				}
			case fixtureProvenanceDefectRegister:
				if !defectIDs[ref.Identifier] {
					t.Errorf("fixture %s cites defect_register identifier %q, which is not a currently fixed defect-register entry", fixture.ID, ref.Identifier)
				}
			case fixtureProvenancePhaseVerification, fixtureProvenancePhaseReview:
				if !phaseReportIDs[ref.Identifier] {
					t.Errorf("fixture %s cites phase report identifier %q, which is not a currently confirmed phase-report row", fixture.ID, ref.Identifier)
				}
			default:
				t.Errorf("fixture %s cites undeclared provenance kind %q", fixture.ID, ref.Kind)
			}
		}
	}
	if emptyCount != 0 {
		t.Fatalf("%d fixtures in the committed bank have an empty provenance array", emptyCount)
	}
	if len(kindCounts) < 3 {
		t.Fatalf("expected fixtures from at least 3 distinct provenance kinds, got %v", kindCounts)
	}
	for _, want := range []fixtureProvenanceKind{fixtureProvenanceAuditFinding, fixtureProvenanceDefectRegister, fixtureProvenancePhaseVerification} {
		if kindCounts[want] == 0 {
			t.Errorf("expected at least one fixture citing provenance kind %q, got none", want)
		}
	}
}

func TestSeededBankIsReproducible(t *testing.T) {
	repoRoot := findTestModuleRoot(t)
	failureLog, audit, defects, phaseReports := realConfirmedIncidents(t, repoRoot)

	var all []confirmedIncident
	all = append(all, failureLog.Incidents...)
	all = append(all, audit.Incidents...)
	all = append(all, defects.Incidents...)
	all = append(all, phaseReports.Incidents...)

	bank, _, err := convertConfirmedIncidentsToFixtures(all)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	regenerated, err := json.MarshalIndent(bank, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	regenerated = append(regenerated, '\n')

	committedRaw, err := os.ReadFile(filepath.Join(repoRoot, fixtureBankPath))
	if err != nil {
		t.Fatalf("read committed bank: %v", err)
	}

	// 204-07 (LEARN-05, Task 3) layers a manually-curated `guard` field onto
	// the committed bank -- the test that would fail if a fixture's own
	// invariant broke. convertConfirmedIncidentsToFixtures has no way to
	// derive that curation mechanically from confirmed-incident sources
	// (nothing in a failure log, an audit finding, a defect-register entry,
	// or a phase report names the Go test that guards it), so this
	// reproducibility check compares everything the mechanical conversion
	// DOES own -- every field except guard -- and never re-derives guard
	// itself. TestEveryFixtureNamesItsGuardOrIsCountedUnguarded
	// (cmd/seed_bank_test.go) is the check that holds the guard layer to
	// account.
	var committedBank regressionFixtureBank
	if err := json.Unmarshal(committedRaw, &committedBank); err != nil {
		t.Fatalf("unmarshal committed bank: %v", err)
	}
	for i := range committedBank.Fixtures {
		committedBank.Fixtures[i].Guard = nil
	}
	committedWithoutGuards, err := json.MarshalIndent(committedBank, "", "  ")
	if err != nil {
		t.Fatalf("marshal committed bank without guards: %v", err)
	}
	committedWithoutGuards = append(committedWithoutGuards, '\n')

	if !bytes.Equal(regenerated, committedWithoutGuards) {
		t.Fatalf("re-running the readers and the conversion against the same sources does not reproduce the committed bank byte for byte (guard fields excluded from this comparison, see comment above)")
	}
}
