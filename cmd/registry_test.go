package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- registry-add tests ---

func setupRegistryHubDir(t *testing.T) string {
	t.Helper()
	hubDir := t.TempDir()
	os.MkdirAll(filepath.Join(hubDir, "registry"), 0755)
	os.MkdirAll(filepath.Join(hubDir, "hive"), 0755)
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() {
		os.Setenv("AETHER_HUB_DIR", origHub)
	})
	return hubDir
}

func parseRegistry(t *testing.T, hubDir string) registryData {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(hubDir, "registry", "registry.json"))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	var rd registryData
	if err := json.Unmarshal(raw, &rd); err != nil {
		t.Fatalf("parse registry: %v", err)
	}
	return rd
}

// Test 1: --path flag works as alias for --repo
func TestRegistryAdd_PathFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := setupRegistryHubDir(t)

	rootCmd.SetArgs([]string{"registry-add", "--path", "/tmp/myrepo"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	rd := parseRegistry(t, hubDir)
	if len(rd.Colonies) != 1 {
		t.Fatalf("expected 1 colony, got %d", len(rd.Colonies))
	}
	if rd.Colonies[0].RepoPath != "/tmp/myrepo" {
		t.Errorf("repo_path = %q, want /tmp/myrepo", rd.Colonies[0].RepoPath)
	}
}

// Test 2: --tags flag works as alias for --domain
func TestRegistryAdd_TagsFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := setupRegistryHubDir(t)

	rootCmd.SetArgs([]string{"registry-add", "--path", "/tmp/myrepo", "--tags", "web,api"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	rd := parseRegistry(t, hubDir)
	if len(rd.Colonies) != 1 {
		t.Fatalf("expected 1 colony, got %d", len(rd.Colonies))
	}
	if len(rd.Colonies[0].Domains) != 2 || rd.Colonies[0].Domains[0] != "web" || rd.Colonies[0].Domains[1] != "api" {
		t.Errorf("domains = %v, want [web api]", rd.Colonies[0].Domains)
	}
}

// Test 3: --goal flag stores the colony goal
func TestRegistryAdd_GoalFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := setupRegistryHubDir(t)

	rootCmd.SetArgs([]string{"registry-add", "--path", "/tmp/myrepo", "--goal", "Build feature X"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	rd := parseRegistry(t, hubDir)
	if len(rd.Colonies) != 1 {
		t.Fatalf("expected 1 colony, got %d", len(rd.Colonies))
	}
	if rd.Colonies[0].LastGoal != "Build feature X" {
		t.Errorf("last_goal = %q, want 'Build feature X'", rd.Colonies[0].LastGoal)
	}
}

// Test 4: Positional version arg is accepted (but ignored/unused for now)
func TestRegistryAdd_PositionalVersion(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := setupRegistryHubDir(t)

	rootCmd.SetArgs([]string{"registry-add", "--path", "/tmp/myrepo", "v1.0.0"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	rd := parseRegistry(t, hubDir)
	if len(rd.Colonies) != 1 {
		t.Fatalf("expected 1 colony, got %d", len(rd.Colonies))
	}
}

// Test 5: Full init-style call matches what markdown uses
func TestRegistryAdd_FullInitCall(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := setupRegistryHubDir(t)

	rootCmd.SetArgs([]string{
		"registry-add",
		"--path", "/tmp/myrepo",
		"v1.0.0",
		"--goal", "Build feature X",
		"--active=true",
		"--tags", "web,api",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	rd := parseRegistry(t, hubDir)
	if len(rd.Colonies) != 1 {
		t.Fatalf("expected 1 colony, got %d", len(rd.Colonies))
	}
	if rd.Colonies[0].RepoPath != "/tmp/myrepo" {
		t.Errorf("repo_path = %q, want /tmp/myrepo", rd.Colonies[0].RepoPath)
	}
	if rd.Colonies[0].LastGoal != "Build feature X" {
		t.Errorf("last_goal = %q, want 'Build feature X'", rd.Colonies[0].LastGoal)
	}
	if !rd.Colonies[0].Active {
		t.Errorf("active = false, want true")
	}
	if len(rd.Colonies[0].Domains) != 2 || rd.Colonies[0].Domains[0] != "web" || rd.Colonies[0].Domains[1] != "api" {
		t.Errorf("domains = %v, want [web api]", rd.Colonies[0].Domains)
	}
}

// Test 6: --path takes priority over --repo when both provided
func TestRegistryAdd_PathTakesPriorityOverRepo(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := setupRegistryHubDir(t)

	rootCmd.SetArgs([]string{"registry-add", "--path", "/tmp/path", "--repo", "/tmp/repo"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rd := parseRegistry(t, hubDir)
	if len(rd.Colonies) != 1 {
		t.Fatalf("expected 1 colony, got %d", len(rd.Colonies))
	}
	if rd.Colonies[0].RepoPath != "/tmp/path" {
		t.Errorf("repo_path = %q, want /tmp/path (--path should take priority)", rd.Colonies[0].RepoPath)
	}
}

// Test 7: --tags takes priority over --domain when both provided
func TestRegistryAdd_TagsTakesPriorityOverDomain(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := setupRegistryHubDir(t)

	rootCmd.SetArgs([]string{"registry-add", "--path", "/tmp/myrepo", "--tags", "web", "--domain", "api"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rd := parseRegistry(t, hubDir)
	if len(rd.Colonies) != 1 {
		t.Fatalf("expected 1 colony, got %d", len(rd.Colonies))
	}
	if len(rd.Colonies[0].Domains) != 1 || rd.Colonies[0].Domains[0] != "web" {
		t.Errorf("domains = %v, want [web] (--tags should take priority)", rd.Colonies[0].Domains)
	}
}

// Test 8: --active=false works for seal-style call
func TestRegistryAdd_ActiveFalse(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := setupRegistryHubDir(t)

	rootCmd.SetArgs([]string{"registry-add", "--repo", "/tmp/myrepo", "--active=false"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rd := parseRegistry(t, hubDir)
	if len(rd.Colonies) != 1 {
		t.Fatalf("expected 1 colony, got %d", len(rd.Colonies))
	}
	if rd.Colonies[0].Active {
		t.Errorf("active = true, want false")
	}
}

// Test 9: Existing --repo and --domain flags still work (backward compat)
func TestRegistryAdd_BackwardCompatRepoAndDomain(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := setupRegistryHubDir(t)

	rootCmd.SetArgs([]string{"registry-add", "--repo", "/tmp/myrepo", "--domain", "web,api", "--active=true"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	rd := parseRegistry(t, hubDir)
	if len(rd.Colonies) != 1 {
		t.Fatalf("expected 1 colony, got %d", len(rd.Colonies))
	}
	if rd.Colonies[0].RepoPath != "/tmp/myrepo" {
		t.Errorf("repo_path = %q, want /tmp/myrepo", rd.Colonies[0].RepoPath)
	}
	if len(rd.Colonies[0].Domains) != 2 || rd.Colonies[0].Domains[0] != "web" || rd.Colonies[0].Domains[1] != "api" {
		t.Errorf("domains = %v, want [web api]", rd.Colonies[0].Domains)
	}
}

// Test 10: Goal is updated on re-registration
func TestRegistryAdd_UpdateGoalOnReregister(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := setupRegistryHubDir(t)

	// First registration
	rootCmd.SetArgs([]string{"registry-add", "--path", "/tmp/myrepo", "--goal", "Goal A"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first reg error: %v", err)
	}

	buf.Reset()

	// Re-registration with new goal (same repo path triggers update)
	rootCmd.SetArgs([]string{"registry-add", "--path", "/tmp/myrepo", "--goal", "Goal B"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("re-reg error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result, ok := env["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result object, full output: %s", buf.String())
	}
	if result["updated"] != true {
		t.Fatalf("expected updated:true, got: %v, full output: %s", result["updated"], buf.String())
	}

	rd := parseRegistry(t, hubDir)
	if len(rd.Colonies) != 1 {
		t.Fatalf("expected 1 colony, got %d", len(rd.Colonies))
	}
	if rd.Colonies[0].LastGoal != "Goal B" {
		t.Errorf("last_goal = %q, want 'Goal B' (should update on re-reg)", rd.Colonies[0].LastGoal)
	}
}
