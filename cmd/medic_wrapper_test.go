package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// TestCountFilesInDir
// ---------------------------------------------------------------------------

func TestCountFilesInDir(t *testing.T) {
	dir := t.TempDir()

	// Create some files
	for i := 0; i < 5; i++ {
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("file%d.yaml", i)))
		if err != nil {
			t.Fatalf("create file: %v", err)
		}
		f.Close()
	}

	pattern := filepath.Join(dir, "*.yaml")
	count := countFilesInDir(pattern)
	if count != 5 {
		t.Errorf("expected 5 files, got %d", count)
	}

	// Non-matching pattern
	zeroCount := countFilesInDir(filepath.Join(dir, "*.md"))
	if zeroCount != 0 {
		t.Errorf("expected 0 for non-matching pattern, got %d", zeroCount)
	}
}

// ---------------------------------------------------------------------------
// TestScanWrapperParityHealthy
// ---------------------------------------------------------------------------

func TestScanWrapperParityHealthy(t *testing.T) {
	dir := t.TempDir()
	aetherDir := filepath.Join(dir, ".aether")
	claudeDir := filepath.Join(dir, ".claude")
	opencodeDir := filepath.Join(dir, ".opencode")
	codexDir := filepath.Join(dir, ".codex")

	// Create directories
	dirs := []string{
		filepath.Join(aetherDir, "commands"),
		filepath.Join(claudeDir, "commands", "ant"),
		filepath.Join(opencodeDir, "commands", "ant"),
		filepath.Join(codexDir, "agents"),
		filepath.Join(claudeDir, "agents", "ant"),
		filepath.Join(opencodeDir, "agents"),
		filepath.Join(aetherDir, "agents-claude"),
		filepath.Join(aetherDir, "agents-codex"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Create expected number of YAML commands (50)
	for i := 0; i < expectedYAMLCommands; i++ {
		writeFile(t, aetherDir, fmt.Sprintf("commands/cmd%d.yaml", i), []byte("test"))
	}
	// Create expected number of Claude commands (50)
	for i := 0; i < expectedClaudeCommands; i++ {
		writeFile(t, claudeDir, fmt.Sprintf("commands/ant/cmd%d.md", i), []byte("test"))
	}
	// Create expected number of OpenCode commands (50)
	for i := 0; i < expectedOpenCodeCommands; i++ {
		writeFile(t, opencodeDir, fmt.Sprintf("commands/ant/cmd%d.md", i), []byte("test"))
	}
	// Create expected number of Codex agents (25)
	for i := 0; i < expectedCodexAgents; i++ {
		writeFile(t, codexDir, fmt.Sprintf("agents/agent%d.toml", i), []byte("test"))
	}
	// Create expected number of Claude agents (25)
	for i := 0; i < expectedClaudeAgents; i++ {
		writeFile(t, claudeDir, fmt.Sprintf("agents/ant/agent%d.md", i), []byte("test"))
	}
	// Create expected number of OpenCode agents (25)
	for i := 0; i < expectedOpenCodeAgents; i++ {
		writeFile(t, opencodeDir, fmt.Sprintf("agents/agent%d.md", i), []byte("test"))
	}
	// Create expected number of Claude mirrors (25)
	for i := 0; i < expectedClaudeMirror; i++ {
		writeFile(t, aetherDir, fmt.Sprintf("agents-claude/agent%d.md", i), []byte("test"))
	}
	// Create expected number of Codex mirrors (25)
	for i := 0; i < expectedCodexMirror; i++ {
		writeFile(t, aetherDir, fmt.Sprintf("agents-codex/agent%d.toml", i), []byte("test"))
	}

	// Create colony skills
	for _, name := range []string{"build-discipline", "colony-interaction", "colony-lifecycle", "colony-visuals", "context-management", "error-presentation", "pheromone-protocol", "pheromone-visibility", "state-safety", "worker-priming", "medic"} {
		skillDir := filepath.Join(aetherDir, "skills", "colony", name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		writeFile(t, aetherDir, fmt.Sprintf("skills/colony/%s/SKILL.md", name), []byte("test"))
	}

	// Create domain skills (18 directories, skip README)
	domainSkills := []string{"django", "docker", "golang", "graphql", "html-css", "nextjs", "nodejs", "postgresql", "prisma", "python", "rails", "react", "rest-api", "svelte", "tailwind", "testing", "typescript", "vue"}
	for _, name := range domainSkills {
		skillDir := filepath.Join(aetherDir, "skills", "domain", name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		writeFile(t, aetherDir, fmt.Sprintf("skills/domain/%s/SKILL.md", name), []byte("test"))
	}

	fc := newFileChecker(filepath.Join(dir, ".aether", "data"))
	issues := scanWrapperParity(fc)

	// Healthy setup should produce no warnings or criticals
	for _, issue := range issues {
		if issue.Severity == "warning" || issue.Severity == "critical" {
			t.Errorf("healthy setup produced issue: [%s] %s", issue.Severity, issue.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// TestScanWrapperParityMismatch
// ---------------------------------------------------------------------------

func TestScanWrapperParityMismatch(t *testing.T) {
	dir := t.TempDir()
	aetherDir := filepath.Join(dir, ".aether")

	// Create only 3 YAML commands instead of expected 50
	commandsDir := filepath.Join(aetherDir, "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for i := 0; i < 3; i++ {
		writeFile(t, aetherDir, fmt.Sprintf("commands/cmd%d.yaml", i), []byte("test"))
	}

	fc := newFileChecker(filepath.Join(dir, ".aether", "data"))
	issues := scanWrapperParity(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "YAML commands") && contains(issue.Message, "3") && contains(issue.Message, fmt.Sprintf("%d", expectedYAMLCommands)) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected mismatch warning for YAML commands; got issues: %+v", issues)
	}
}

// ---------------------------------------------------------------------------
// TestScanWrapperParityCrossSurfaceMismatch
// ---------------------------------------------------------------------------

func TestScanWrapperParityCrossSurfaceMismatch(t *testing.T) {
	dir := t.TempDir()
	aetherDir := filepath.Join(dir, ".aether")
	claudeDir := filepath.Join(dir, ".claude")
	opencodeDir := filepath.Join(dir, ".opencode")

	// Create directories
	for _, d := range []string{
		filepath.Join(aetherDir, "commands"),
		filepath.Join(claudeDir, "commands", "ant"),
		filepath.Join(opencodeDir, "commands", "ant"),
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Create 50 YAML commands (correct count)
	for i := 0; i < expectedYAMLCommands; i++ {
		writeFile(t, aetherDir, fmt.Sprintf("commands/cmd%d.yaml", i), []byte("test"))
	}
	// Create only 48 Claude commands (mismatch)
	for i := 0; i < 48; i++ {
		writeFile(t, claudeDir, fmt.Sprintf("commands/ant/cmd%d.md", i), []byte("test"))
	}
	// Create only 47 OpenCode commands (mismatch)
	for i := 0; i < 47; i++ {
		writeFile(t, opencodeDir, fmt.Sprintf("commands/ant/cmd%d.md", i), []byte("test"))
	}

	fc := newFileChecker(filepath.Join(dir, ".aether", "data"))
	issues := scanWrapperParity(fc)

	// Should have cross-surface command count mismatch warning
	foundCrossSurface := false
	for _, issue := range issues {
		if issue.Severity == "warning" && issue.Category == "wrapper" && contains(issue.Message, "Command count mismatch") {
			foundCrossSurface = true
		}
	}
	if !foundCrossSurface {
		t.Errorf("expected cross-surface command mismatch warning; got: %+v", issues)
	}
}

// ---------------------------------------------------------------------------
// TestDeepScanIncludesWrapperParity
// ---------------------------------------------------------------------------

func TestDeepScanIncludesWrapperParity(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("AETHER_ROOT", dir)

	// Minimal healthy colony data
	goal := "Deep scan test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", map[string]string{
		"version": "3.0",
		"goal":    goal,
		"state":   "READY",
	})

	// Create wrapper directories with mismatched counts to trigger wrapper parity issues
	aetherDir := filepath.Join(dir, ".aether")
	commandsDir := filepath.Join(aetherDir, "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Only 1 YAML command instead of expected 50
	writeFile(t, aetherDir, "commands/just-one.yaml", []byte("test"))

	// Run deep scan
	opts := MedicOptions{Deep: true}
	result, err := performHealthScan(opts)
	if err != nil {
		t.Fatalf("performHealthScan failed: %v", err)
	}

	// Should have wrapper parity warning about YAML commands
	foundWrapper := false
	for _, issue := range result.Issues {
		if issue.Category == "wrapper" {
			foundWrapper = true
		}
	}
	if !foundWrapper {
		t.Error("deep scan should include wrapper parity issues")
	}

	// Run without deep -- should NOT have wrapper parity issues
	optsNoDeep := MedicOptions{Deep: false}
	resultNoDeep, err := performHealthScan(optsNoDeep)
	if err != nil {
		t.Fatalf("performHealthScan without deep failed: %v", err)
	}
	for _, issue := range resultNoDeep.Issues {
		if issue.Category == "wrapper" {
			t.Error("non-deep scan should NOT include wrapper parity issues")
		}
	}
}
