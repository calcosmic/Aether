package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// expectedContracts lists the 16 lifecycle commands that must have contract documents.
var expectedContracts = []string{
	"init", "discuss", "colonize", "plan", "build", "continue",
	"seal", "entomb", "publish", "update", "recover", "status",
	"resume", "watch", "patrol", "profile",
}

// requiredSections lists the 4 section headers every contract must contain.
var requiredSections = []string{
	"## Inputs",
	"## Outputs",
	"## State Mutations",
	"## Preconditions",
}

// TestLifecycleContracts verifies all 16 contract files exist in cmd/contracts/
// and that no extra .md files are present beyond the expected set.
func TestLifecycleContracts(t *testing.T) {
	contractsDir := filepath.Join("contracts")

	// Check each expected contract file exists and is readable
	missing := []string{}
	for _, name := range expectedContracts {
		path := filepath.Join(contractsDir, name+".md")
		if _, err := os.ReadFile(path); err != nil {
			missing = append(missing, name+".md")
		}
	}
	if len(missing) > 0 {
		t.Errorf("Missing contract files: %v", missing)
	}

	// Check for unexpected extra .md files
	entries, err := os.ReadDir(contractsDir)
	if err != nil {
		t.Fatalf("Failed to read contracts directory: %v", err)
	}

	expectedSet := make(map[string]bool, len(expectedContracts))
	for _, name := range expectedContracts {
		expectedSet[name+".md"] = true
	}

	extra := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if !expectedSet[entry.Name()] {
			extra = append(extra, entry.Name())
		}
	}
	if len(extra) > 0 {
		t.Errorf("Unexpected extra contract files: %v", extra)
	}

	// Verify exact count
	mdCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			mdCount++
		}
	}
	if mdCount != len(expectedContracts) {
		t.Errorf("Expected %d contract files, found %d", len(expectedContracts), mdCount)
	}
}

// TestContractStructure verifies each contract file contains all 4 required
// section headers, a title line, and a "Last verified:" date line.
func TestContractStructure(t *testing.T) {
	contractsDir := filepath.Join("contracts")

	for _, name := range expectedContracts {
		path := filepath.Join(contractsDir, name+".md")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Cannot read contract %s: %v", name, err)
			continue
		}

		text := string(content)

		// Check title line
		if !strings.Contains(text, "# "+name+" --") && !strings.Contains(text, "# "+name+" —") {
			t.Errorf("Contract %s missing title line with command name", name)
		}

		// Check Last verified date
		if !strings.Contains(text, "Last verified:") {
			t.Errorf("Contract %s missing 'Last verified:' date line", name)
		}

		// Check all 4 required sections
		for _, section := range requiredSections {
			if !strings.Contains(text, section) {
				t.Errorf("Contract %s missing required section: %s", name, section)
			}
		}
	}
}
