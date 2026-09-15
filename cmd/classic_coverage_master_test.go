package cmd

import "testing"

func TestClassicCoveragePhase205Signed(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	document := loadClassicCoveragePhase(t, root, "205")
	if document.Version != "205/classic-coverage/v1" {
		t.Fatalf("phase 205 coverage version = %q, want 205/classic-coverage/v1", document.Version)
	}
	for _, row := range document.Rows {
		if row.Type != "GOAL" {
			t.Fatalf("phase 205 coverage contains %s/%s, want only GOAL rows", row.Type, row.ID)
		}
	}
	if err := validateClassicCoverage(document, root, "205"); err != nil {
		t.Fatal(err)
	}
}
