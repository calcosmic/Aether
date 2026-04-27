package cmd

import (
	"encoding/json"
	"testing"
)

func TestPorterCheckCommandRegistered(t *testing.T) {
	cmd := rootCmd
	porterCmd, _, err := cmd.Find([]string{"porter", "check"})
	if err != nil {
		t.Fatalf("porter check command not found: %v", err)
	}
	if porterCmd == nil {
		t.Fatal("porter check command is nil")
	}
	if porterCmd.Use != "check" {
		t.Fatalf("expected Use 'check', got %q", porterCmd.Use)
	}
}

func TestPorterCheckJSONOutput(t *testing.T) {
	// Verify the --json flag exists
	porterCheckCmd, _, err := rootCmd.Find([]string{"porter", "check"})
	if err != nil {
		t.Fatalf("porter check command not found: %v", err)
	}
	jsonFlag := porterCheckCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("porter check missing --json flag")
	}
	channelFlag := porterCheckCmd.Flags().Lookup("channel")
	if channelFlag == nil {
		t.Fatal("porter check missing --channel flag")
	}
}

func TestPorterCheckIncludesIntegrityChecks(t *testing.T) {
	// Verify porter check composes integrity check functions
	// by checking that the result structure includes expected check names
	// when run in a test environment (may have skip/fail status)
	checks := buildPorterChecks("stable", true)
	expectedNames := map[string]bool{
		"Source version":        false,
		"Binary version":        false,
		"Hub version":           false,
		"Hub companion files":   false,
		"Downstream simulation": false,
		"Git status":            false,
		"Test status":           false,
		"Changelog completeness": false,
	}
	for _, c := range checks {
		if _, ok := expectedNames[c.Name]; ok {
			expectedNames[c.Name] = true
		}
	}
	for name, found := range expectedNames {
		if !found {
			t.Errorf("porter check missing expected check: %s", name)
		}
	}
}

func TestPorterCheckResultStructure(t *testing.T) {
	checks := buildPorterChecks("stable", true)
	data, err := json.Marshal(checks)
	if err != nil {
		t.Fatalf("failed to marshal checks to JSON: %v", err)
	}
	// Verify it's valid JSON
	var parsed []map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("checks JSON is invalid: %v", err)
	}
	if len(parsed) == 0 {
		t.Fatal("porter checks array is empty")
	}
	// Verify each check has required fields
	for i, c := range parsed {
		if _, ok := c["name"]; !ok {
			t.Errorf("check %d missing 'name' field", i)
		}
		if _, ok := c["status"]; !ok {
			t.Errorf("check %d missing 'status' field", i)
		}
		status, _ := c["status"].(string)
		if status != "pass" && status != "fail" && status != "skip" {
			t.Errorf("check %d has invalid status %q", i, status)
		}
	}
}

func TestPorterCheckHasCorrectCount(t *testing.T) {
	checks := buildPorterChecks("stable", true)
	if len(checks) != 8 {
		t.Errorf("expected 8 porter checks, got %d", len(checks))
	}
}

func TestCheckGitStatusFunction(t *testing.T) {
	result := checkGitStatus()
	if result.Name != "Git status" {
		t.Errorf("expected name 'Git status', got %q", result.Name)
	}
	// Status should be pass or fail (skip not expected)
	if result.Status != "pass" && result.Status != "fail" {
		t.Errorf("expected pass or fail, got %q", result.Status)
	}
	if result.Status == "fail" && result.RecoveryCommand == "" {
		t.Error("failed git status check should have recovery command")
	}
}

func TestCheckChangelogCompletenessFunction(t *testing.T) {
	result := checkChangelogCompleteness()
	if result.Name != "Changelog completeness" {
		t.Errorf("expected name 'Changelog completeness', got %q", result.Name)
	}
	if result.Status != "pass" && result.Status != "fail" && result.Status != "skip" {
		t.Errorf("expected pass/fail/skip, got %q", result.Status)
	}
}
