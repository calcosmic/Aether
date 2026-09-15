package cmd

import (
	"encoding/json"
	"os"
	"testing"
)

// loadClassicCoveragePhase204 mirrors loadClassicCoveragePhase203
// (cmd/classic_coverage_ratchet_test.go) for phase "204".
func loadClassicCoveragePhase204(t *testing.T, root string) classicCoverageDocument {
	t.Helper()
	path, err := classicCoverageJSONPathInRoot(root, "204")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read classic coverage: %v", err)
	}
	var document classicCoverageDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode classic coverage: %v", err)
	}
	return document
}

// TestClassicCoveragePhase204Signed loads 204-CLASSIC-COVERAGE.json and runs
// the shared validator for phase "204", expecting no error. Phase 204's
// study (204-CLASSIC-SYNTHESIS.md ruling (c)) flagged CAP-001 and CAP-057 as
// frozen against a state the code has since left; both were independently
// confirmed correct in direction, so both rows carry the ledger's own
// disposition value with the ruling recorded as refined evidence -- neither
// needed the re-adjudication marker this file's other tests exercise.
func TestClassicCoveragePhase204Signed(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	document := loadClassicCoveragePhase204(t, root)
	if err := validateClassicCoverage(document, root, "204"); err != nil {
		t.Fatal(err)
	}
}
