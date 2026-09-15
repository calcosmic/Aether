package cmd

import (
	"encoding/json"
	"os"
	"testing"
)

// loadClassicCoveragePhase loads and decodes a phase's
// <phase>-CLASSIC-COVERAGE.json against the given root, failing the test on
// any read or decode error. Mirrors loadClassicCoveragePhase203 in
// cmd/classic_coverage_ratchet_test.go, generalized to any phase.
func loadClassicCoveragePhase(t *testing.T, root, phase string) classicCoverageDocument {
	t.Helper()
	path, err := classicCoverageJSONPathInRoot(root, phase)
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

// TestClassicCoveragePhase200Signed loads 200-CLASSIC-COVERAGE.json and runs
// validateClassicCoverage for phase "200", expecting no error: all seven
// capability rows the frozen ledger routes to Phase 200 are signed with a
// proof individually executed and observed passing in this session.
func TestClassicCoveragePhase200Signed(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	document := loadClassicCoveragePhase(t, root, "200")
	if err := validateClassicCoverage(document, root, "200"); err != nil {
		t.Fatal(err)
	}
}

// TestClassicCoveragePhase201Signed loads 201-CLASSIC-COVERAGE.json and runs
// validateClassicCoverage for phase "201", expecting no error: all eight
// capability rows the frozen ledger routes to Phase 201 are signed with a
// proof individually executed and observed passing in this session.
func TestClassicCoveragePhase201Signed(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	document := loadClassicCoveragePhase(t, root, "201")
	if err := validateClassicCoverage(document, root, "201"); err != nil {
		t.Fatal(err)
	}
}

// TestClassicCoveragePhase202Signed loads 202-CLASSIC-COVERAGE.json and runs
// validateClassicCoverage for phase "202", expecting no error: all eight
// capability rows the frozen ledger routes to Phase 202 are signed with a
// proof individually executed and observed passing in this session.
func TestClassicCoveragePhase202Signed(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	document := loadClassicCoveragePhase(t, root, "202")
	if err := validateClassicCoverage(document, root, "202"); err != nil {
		t.Fatal(err)
	}
}
