package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	fullReleaseFlag := porterCheckCmd.Flags().Lookup("full-release")
	if fullReleaseFlag == nil {
		t.Fatal("porter check missing --full-release flag")
	}
}

func TestPorterCheckIncludesIntegrityChecks(t *testing.T) {
	// Running inside the Aether source repo, so source checks apply
	checks := buildPorterChecks("stable", true)
	expectedNames := map[string]bool{
		"Source version":         false,
		"Binary version":         false,
		"Hub version":            false,
		"Hub companion files":    false,
		"Downstream simulation":  false,
		"Git status":             false,
		"Git stashes":            false,
		"Git worktrees":          false,
		"Test status":            false,
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
	// Source repo context: 10 checks
	checks := buildPorterChecks("stable", true)
	if len(checks) != 10 {
		t.Errorf("expected 10 porter checks (source), got %d", len(checks))
	}
}

func TestPorterConsumerChecksHasCorrectCount(t *testing.T) {
	checks := buildPorterChecksForContext(porterContextConsumer, "stable", true)
	if len(checks) != 6 {
		t.Errorf("expected 6 consumer porter checks, got %d", len(checks))
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

func TestCheckGitStashesFunction(t *testing.T) {
	result := checkGitStashes()
	if result.Name != "Git stashes" {
		t.Errorf("expected name 'Git stashes', got %q", result.Name)
	}
	if result.Status != "pass" && result.Status != "fail" {
		t.Errorf("expected pass or fail, got %q", result.Status)
	}
	if result.Status == "fail" && result.RecoveryCommand == "" {
		t.Error("failed git stashes check should have recovery command")
	}
}

func TestCheckGitWorktreesFunction(t *testing.T) {
	result := checkGitWorktrees()
	if result.Name != "Git worktrees" {
		t.Errorf("expected name 'Git worktrees', got %q", result.Name)
	}
	if result.Status != "pass" && result.Status != "fail" && result.Status != "skip" {
		t.Errorf("expected pass/fail/skip, got %q", result.Status)
	}
	if result.Status == "fail" && result.RecoveryCommand == "" {
		t.Error("failed git worktrees check should have recovery command")
	}
}

func TestPorterTestEnvClearsOutputMode(t *testing.T) {
	env := porterTestEnv([]string{
		"KEEP_ME=1",
		"AETHER_OUTPUT_MODE=json",
		"AETHER_OUTPUT_MODE=visual",
	})

	if !containsString(env, "KEEP_ME=1") {
		t.Fatal("expected unrelated environment variable to be preserved")
	}
	count := 0
	for _, entry := range env {
		if strings.HasPrefix(entry, "AETHER_OUTPUT_MODE=") {
			count++
			if entry != "AETHER_OUTPUT_MODE=" {
				t.Fatalf("expected AETHER_OUTPUT_MODE to be cleared, got %q", entry)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected one cleared AETHER_OUTPUT_MODE entry, got %d in %v", count, env)
	}
}

func TestLastPorterOutputLinesRedactsTokenShapedOutput(t *testing.T) {
	rawToken := "sk-testtokenvalue1234567890"
	message := lastPorterOutputLines([]byte(strings.Join([]string{
		"line zero",
		"line one",
		"line two",
		"line three",
		"api_key=" + rawToken,
		"last line",
	}, "\n")), 5)

	if strings.Contains(message, rawToken) {
		t.Fatalf("lastPorterOutputLines leaked raw token in %q", message)
	}
	if !strings.Contains(message, "[redacted]") {
		t.Fatalf("lastPorterOutputLines = %q, want redacted marker", message)
	}
	if !strings.Contains(message, "last line") {
		t.Fatalf("lastPorterOutputLines = %q, want non-secret diagnostic context preserved", message)
	}
	if strings.Contains(message, "line zero") {
		t.Fatalf("lastPorterOutputLines = %q, want only the last diagnostic lines", message)
	}
}

func TestRunPorterCommandCheckRedactsTokenShapedOutput(t *testing.T) {
	rawToken := "github_pat_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	script := "printf 'build failed\\nTOKEN=" + rawToken + "\\n'; exit 1"

	check := runPorterCommandCheck(porterCommandCheck{
		Name:            "Token leak regression",
		Args:            []string{"sh", "-c", script},
		Timeout:         time.Second,
		RecoveryCommand: "Fix failing command",
	})

	if check.Status != "fail" {
		t.Fatalf("Status = %q, want fail", check.Status)
	}
	if strings.Contains(check.Message, rawToken) {
		t.Fatalf("runPorterCommandCheck leaked raw token in %q", check.Message)
	}
	if !strings.Contains(check.Message, "[redacted]") {
		t.Fatalf("Message = %q, want redacted marker", check.Message)
	}
}

func TestDetectPorterContext(t *testing.T) {
	ctx := detectPorterContext()
	if ctx != porterContextSource && ctx != porterContextConsumer {
		t.Errorf("expected source or consumer, got %q", ctx)
	}
	// Running inside the Aether source repo, should detect source
	if ctx != porterContextSource {
		t.Logf("Note: detected %q context (expected source if running in Aether repo)", ctx)
	}
}

func TestPorterSourceChecksIncludesStashAndWorktree(t *testing.T) {
	checks := buildPorterChecksForContext(porterContextSource, "stable", true)
	names := map[string]bool{}
	for _, c := range checks {
		names[c.Name] = true
	}
	if !names["Git stashes"] {
		t.Error("source checks missing 'Git stashes'")
	}
	if !names["Git worktrees"] {
		t.Error("source checks missing 'Git worktrees'")
	}
	if !names["Changelog completeness"] {
		t.Error("source checks missing 'Changelog completeness'")
	}
	if !names["Test status"] {
		t.Error("source checks missing 'Test status'")
	}
}

func TestPorterConsumerChecksExcludesSourceOnlyChecks(t *testing.T) {
	checks := buildPorterChecksForContext(porterContextConsumer, "stable", true)
	names := map[string]bool{}
	for _, c := range checks {
		names[c.Name] = true
	}
	if names["Source version"] {
		t.Error("consumer checks should not include 'Source version'")
	}
	if names["Downstream simulation"] {
		t.Error("consumer checks should not include 'Downstream simulation'")
	}
	if names["Test status"] {
		t.Error("consumer checks should not include 'Test status'")
	}
	if names["Changelog completeness"] {
		t.Error("consumer checks should not include 'Changelog completeness'")
	}
	if !names["Git stashes"] {
		t.Error("consumer checks missing 'Git stashes'")
	}
	if !names["Git worktrees"] {
		t.Error("consumer checks missing 'Git worktrees'")
	}
	if !names["Hub companion sync"] {
		t.Error("consumer checks missing 'Hub companion sync'")
	}
}

func TestPorterQuickAndFullReleaseScopesAreDistinct(t *testing.T) {
	quick := buildPorterChecksForContext(porterContextSource, "stable", true)
	full := buildPorterChecksForContextWithScope(porterContextSource, "stable", true, porterReadinessScopeFullRelease)

	quickNames := map[string]bool{}
	for _, c := range quick {
		quickNames[c.Name] = true
	}
	fullNames := map[string]bool{}
	for _, c := range full {
		fullNames[c.Name] = true
	}

	if fullNames["Release version agreement"] == false {
		t.Fatal("full release checks should include release version agreement")
	}
	if fullNames["Source surface alignment"] == false {
		t.Fatal("full release checks should include source surface alignment")
	}
	for _, want := range []string{
		"Go vet",
		"Go race tests",
		"Go binary build",
		"GoReleaser config",
		"GoReleaser snapshot",
		"TS host typecheck",
		"TS host tests",
		"TS host build",
		"npm package tests",
		"Aether binary smoke",
	} {
		if !fullNames[want] {
			t.Fatalf("full release checks should include %q", want)
		}
	}
	if quickNames["Release version agreement"] || quickNames["Source surface alignment"] {
		t.Fatal("quick porter checks should not include full-release-only checks")
	}
	if !fullNames["Git status"] {
		t.Fatal("full release checks must preserve clean-worktree blocking")
	}
}

func TestReleaseVersionAgreementDetectsNpmMismatch(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".aether"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "npm"), 0755); err != nil {
		t.Fatal(err)
	}
	hub := filepath.Join(root, "hub")
	if err := os.MkdirAll(hub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".aether", "version.json"), []byte(`{"version":"1.2.3"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "npm", "package.json"), []byte(`{"version":"1.2.4"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hub, "version.json"), []byte(`{"version":"1.2.3"}`), 0644); err != nil {
		t.Fatal(err)
	}

	check := checkReleaseVersionAgreementAt(root, hub, "1.2.3")

	if check.Status != "fail" {
		t.Fatalf("Status = %q, want fail", check.Status)
	}
	if !strings.Contains(check.Message, "npm package version 1.2.4") {
		t.Fatalf("Message = %q, want npm mismatch detail", check.Message)
	}
	if check.RecoveryCommand == "" {
		t.Fatal("mismatched release versions should include a recovery command")
	}
}

func TestRecordPorterReadinessEvidencePersistsScope(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	result := integrityResult{
		Context:     "source",
		Channel:     "stable",
		Scope:       string(porterReadinessScopeFullRelease),
		GeneratedAt: "2026-05-17T12:00:00Z",
		Checks: []integrityCheck{
			{Name: "Git status", Status: "pass", Message: "Working tree clean"},
			{Name: "Test status", Status: "skip", Message: "Skipped in test mode"},
		},
		Overall:       "ok",
		SkippedChecks: []string{"Test status"},
	}

	rel, err := recordPorterReadinessEvidence(result)
	if err != nil {
		t.Fatalf("recordPorterReadinessEvidence error: %v", err)
	}
	if rel != porterReadinessReportRel {
		t.Fatalf("rel = %q, want %q", rel, porterReadinessReportRel)
	}

	var saved integrityResult
	if err := s.LoadJSON(porterReadinessReportRel, &saved); err != nil {
		t.Fatalf("load porter readiness evidence: %v", err)
	}
	if saved.Scope != string(porterReadinessScopeFullRelease) {
		t.Fatalf("Scope = %q, want full release", saved.Scope)
	}
	if len(saved.SkippedChecks) != 1 || saved.SkippedChecks[0] != "Test status" {
		t.Fatalf("SkippedChecks = %v, want [Test status]", saved.SkippedChecks)
	}
}

func TestPorterVisualDisclosesScopeEvidenceAndSkippedChecks(t *testing.T) {
	visual := buildPorterVisual(integrityResult{
		Context:       "source",
		Channel:       "stable",
		Scope:         string(porterReadinessScopeQuick),
		GeneratedAt:   "2026-05-17T12:00:00Z",
		Evidence:      ".aether/data/porter/readiness.json",
		SkippedChecks: []string{"Go race tests", "GoReleaser snapshot"},
		Checks: []integrityCheck{
			{Name: "Git status", Status: "pass", Message: "Working tree clean"},
			{Name: "Go race tests", Status: "skip", Message: "Skipped in quick scope"},
		},
		Overall: "ok",
	})

	for _, want := range []string{
		"Scope: quick",
		"Quick scope omits slower release-tool checks",
		"Evidence: .aether/data/porter/readiness.json",
		"Skipped: Go race tests, GoReleaser snapshot",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("porter visual missing %q:\n%s", want, visual)
		}
	}
}

func TestPorterVisualDisclosesFullReleaseScope(t *testing.T) {
	visual := buildPorterVisual(integrityResult{
		Context:     "source",
		Channel:     "stable",
		Scope:       string(porterReadinessScopeFullRelease),
		GeneratedAt: "2026-05-17T12:00:00Z",
		Checks: []integrityCheck{
			{Name: "Go vet", Status: "pass", Message: "Command passed"},
		},
		Overall: "ok",
	})

	for _, want := range []string{
		"Scope: full-release",
		"Full-release scope runs slower release-tool checks",
	} {
		if !strings.Contains(visual, want) {
			t.Fatalf("porter visual missing %q:\n%s", want, visual)
		}
	}
}
