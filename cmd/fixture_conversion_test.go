package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
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
