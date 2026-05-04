package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validAgentFrontmatter is a complete, valid OpenCode agent frontmatter for testing.
const validAgentFrontmatter = `name: aether-roundtrip-test
description: "OpenCode roundtrip test agent for verifying name field preservation"
mode: subagent
color: "#3b82f6"
tools:
  write: true
  edit: true
  bash: true
  grep: true
  glob: true
  task: true`

// TestOpenCodeAgentNamePreservedInSync verifies that an OpenCode agent file
// with a valid name field retains its name field after passing through the
// install sync pipeline (syncDir with validateOpenCodeAgentFile).
func TestOpenCodeAgentNamePreservedInSync(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source agent file
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	content := "---\n" + validAgentFrontmatter + "\n---\n\n# Roundtrip test agent\n"
	srcFile := filepath.Join(srcDir, "aether-roundtrip-test.md")
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("write src file: %v", err)
	}

	// Create dest dir
	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	// Run sync with validation (same as installSyncPairs for OpenCode agents)
	result := syncDir(srcDir, destDir, syncOptions{
		validate: validateOpenCodeAgentFile,
	})
	if len(result.errors) > 0 {
		t.Fatalf("sync failed: %v", result.errors)
	}
	if result.copied != 1 {
		t.Fatalf("expected 1 file copied, got %d", result.copied)
	}

	// Read the synced file and verify name field is intact
	destFile := filepath.Join(destDir, "aether-roundtrip-test.md")
	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("read dest file: %v", err)
	}

	// Verify it still passes validation
	if err := validateOpenCodeAgentFile(destFile, "aether-roundtrip-test.md", data); err != nil {
		t.Fatalf("synced file failed validation: %v", err)
	}

	// Verify name field is present in the synced content
	destContent := string(data)
	if !strings.Contains(destContent, "name: aether-roundtrip-test") {
		t.Error("name field missing from synced file")
	}

	// Verify byte-for-byte preservation (name must survive copy)
	if string(data) != content {
		t.Error("file content was modified during sync (name field may have been lost)")
	}
}

func TestInstallSyncPairsDoesNotMirrorFullSkillsToCodex(t *testing.T) {
	for _, pair := range installSyncPairs() {
		if pair.srcRel == ".aether/skills" && pair.destRel == ".codex/skills/aether" {
			t.Fatalf("install sync must not mirror full shipped skills into Codex native discovery: %+v", pair)
		}
	}
}

func TestSyncCodexSkillShimsPrunesFullMirrorAndPreservesCustom(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, ".codex", "skills", "aether")

	oldMirror := filepath.Join(destDir, "domain", "typescript")
	if err := os.MkdirAll(oldMirror, 0755); err != nil {
		t.Fatalf("mkdir old mirror: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oldMirror, "SKILL.md"), []byte("---\nname: typescript\nsource: shipped\ntype: domain\n---\nold full skill\n"), 0644); err != nil {
		t.Fatalf("write old mirror: %v", err)
	}

	customSkill := filepath.Join(destDir, "custom-routing")
	if err := os.MkdirAll(customSkill, 0755); err != nil {
		t.Fatalf("mkdir custom skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(customSkill, "SKILL.md"), []byte("---\nname: custom-routing\nsource: custom\ntype: codex-shim\n---\ncustom\n"), 0644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	result := syncCodexSkillShims(destDir)
	if len(result.errors) > 0 {
		t.Fatalf("syncCodexSkillShims errors: %v", result.errors)
	}
	if _, err := os.Stat(filepath.Join(oldMirror, "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("expected old full mirror to be pruned, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(customSkill, "SKILL.md")); err != nil {
		t.Fatalf("expected custom skill to be preserved: %v", err)
	}
	for _, shim := range codexSkillShims() {
		if _, err := os.Stat(filepath.Join(destDir, shim.Dir, "SKILL.md")); err != nil {
			t.Fatalf("expected shim %s to exist: %v", shim.Dir, err)
		}
	}

	dirs := findSkillDirs(destDir)
	if len(dirs) != len(codexSkillShims())+1 {
		t.Fatalf("expected %d shim/custom skills, got %d: %v", len(codexSkillShims())+1, len(dirs), dirs)
	}
}

func TestPruneShippedFromUserSkillsDirRemovesOnlyExactHubMatches(t *testing.T) {
	tmpDir := t.TempDir()
	hubSystem := filepath.Join(tmpDir, "system")
	hubDir := tmpDir

	shippedPath := filepath.Join(hubSystem, "skills", "domain", "typescript", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(shippedPath), 0755); err != nil {
		t.Fatalf("mkdir shipped: %v", err)
	}
	shippedContent := []byte("---\nname: typescript\nsource: shipped\ntype: domain\n---\ncurrent\n")
	if err := os.WriteFile(shippedPath, shippedContent, 0644); err != nil {
		t.Fatalf("write shipped: %v", err)
	}

	leakedPath := filepath.Join(hubDir, "skills", "domain", "typescript", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(leakedPath), 0755); err != nil {
		t.Fatalf("mkdir leaked: %v", err)
	}
	if err := os.WriteFile(leakedPath, shippedContent, 0644); err != nil {
		t.Fatalf("write leaked: %v", err)
	}

	customPath := filepath.Join(hubDir, "skills", "domain", "custom", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(customPath), 0755); err != nil {
		t.Fatalf("mkdir custom: %v", err)
	}
	if err := os.WriteFile(customPath, []byte("---\nname: custom\nsource: custom\ntype: domain\n---\ncustom\n"), 0644); err != nil {
		t.Fatalf("write custom: %v", err)
	}

	result := pruneShippedFromUserSkillsDir(hubSystem, hubDir)
	if len(result.errors) > 0 {
		t.Fatalf("prune errors: %v", result.errors)
	}
	if _, err := os.Stat(leakedPath); !os.IsNotExist(err) {
		t.Fatalf("expected exact leaked shipped skill to be removed, stat err: %v", err)
	}
	if _, err := os.Stat(customPath); err != nil {
		t.Fatalf("expected custom skill to be preserved: %v", err)
	}
}

func TestPruneRepoCodexSkillMirrorForcePreservesCustom(t *testing.T) {
	tmpDir := t.TempDir()
	mirror := filepath.Join(tmpDir, ".codex", "skills", "aether", "domain", "typescript")
	if err := os.MkdirAll(mirror, 0755); err != nil {
		t.Fatalf("mkdir mirror: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mirror, "SKILL.md"), []byte("---\nname: typescript\ntype: domain\n---\nold mirror\n"), 0644); err != nil {
		t.Fatalf("write mirror: %v", err)
	}
	custom := filepath.Join(tmpDir, ".codex", "skills", "aether", "custom")
	if err := os.MkdirAll(custom, 0755); err != nil {
		t.Fatalf("mkdir custom: %v", err)
	}
	if err := os.WriteFile(filepath.Join(custom, "SKILL.md"), []byte("---\nname: custom\nsource: custom\ntype: codex-shim\n---\ncustom\n"), 0644); err != nil {
		t.Fatalf("write custom: %v", err)
	}

	result := pruneRepoCodexSkillMirror(tmpDir, true)
	if len(result.errors) > 0 {
		t.Fatalf("prune errors: %v", result.errors)
	}
	if _, err := os.Stat(filepath.Join(mirror, "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("expected stale repo Codex mirror to be pruned, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(custom, "SKILL.md")); err != nil {
		t.Fatalf("expected custom Codex skill to be preserved: %v", err)
	}
}

// TestOpenCodeAgentNamePreservedInHubSync verifies that an OpenCode agent file
// retains its name field through the hub sync pipeline (syncDirToHubWithExclusion).
func TestOpenCodeAgentNamePreservedInHubSync(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source agent file (simulating .opencode/agents/)
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	content := "---\n" + validAgentFrontmatter + "\n---\n\n# Hub roundtrip test agent\n"
	srcFile := filepath.Join(srcDir, "aether-roundtrip-test.md")
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("write src file: %v", err)
	}

	// Create hub dest (simulating ~/.aether/system/agents/)
	destDir := filepath.Join(tmpDir, "hub")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("mkdir hub dest: %v", err)
	}

	// Run hub sync with validation (same as setupInstallHub for OpenCode agents)
	result := syncDirToHubWithExclusion(srcDir, destDir, nil, validateOpenCodeAgentFile, nil)
	if len(result.errors) > 0 {
		t.Fatalf("hub sync failed: %v", result.errors)
	}
	if result.copied != 1 {
		t.Fatalf("expected 1 file copied, got %d", result.copied)
	}

	// Read the hub file and verify name field is intact
	destFile := filepath.Join(destDir, "aether-roundtrip-test.md")
	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("read hub file: %v", err)
	}

	if err := validateOpenCodeAgentFile(destFile, "aether-roundtrip-test.md", data); err != nil {
		t.Fatalf("hub synced file failed validation: %v", err)
	}

	destContent := string(data)
	if !strings.Contains(destContent, "name: aether-roundtrip-test") {
		t.Error("name field missing from hub synced file")
	}
}

// TestOpenCodeAgentNamePreservedFullRoundtrip verifies the complete pipeline:
// source -> hub -> target (simulating aether publish then aether update).
func TestOpenCodeAgentNamePreservedFullRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()

	// Phase 1: Source (simulating .opencode/agents/ in repo)
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	content := "---\n" + validAgentFrontmatter + "\n---\n\n# Full roundtrip agent\n"
	srcFile := filepath.Join(srcDir, "aether-roundtrip-test.md")
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("write src file: %v", err)
	}

	// Phase 2: Publish to hub (source -> hub/system/agents/)
	hubDir := filepath.Join(tmpDir, "hub", "system", "agents")
	if err := os.MkdirAll(hubDir, 0755); err != nil {
		t.Fatalf("mkdir hub: %v", err)
	}
	hubResult := syncDirToHubWithExclusion(srcDir, hubDir, nil, validateOpenCodeAgentFile, nil)
	if len(hubResult.errors) > 0 {
		t.Fatalf("publish to hub failed: %v", hubResult.errors)
	}

	// Phase 3: Install from hub (hub -> target .opencode/agents/)
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	installResult := syncDir(hubDir, targetDir, syncOptions{
		validate: validateOpenCodeAgentFile,
	})
	if len(installResult.errors) > 0 {
		t.Fatalf("install from hub failed: %v", installResult.errors)
	}

	// Verify the final file has the name field
	finalFile := filepath.Join(targetDir, "aether-roundtrip-test.md")
	data, err := os.ReadFile(finalFile)
	if err != nil {
		t.Fatalf("read final file: %v", err)
	}

	if err := validateOpenCodeAgentFile(finalFile, "aether-roundtrip-test.md", data); err != nil {
		t.Fatalf("final file failed validation: %v", err)
	}

	finalContent := string(data)
	if !strings.Contains(finalContent, "name: aether-roundtrip-test") {
		t.Error("name field missing after full roundtrip (publish -> hub -> install)")
	}

	// Verify byte-for-byte preservation through the full roundtrip
	if finalContent != content {
		t.Errorf("file content changed during roundtrip:\n  expected: %q\n  got:      %q", content, finalContent)
	}
}

// TestOpenCodeAgentRejectsMissingNameInSync verifies that an agent file
// without a name field is rejected during the sync pipeline.
func TestOpenCodeAgentRejectsMissingNameInSync(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	// Agent without name field
	noNameContent := "---\ndescription: \"This agent has no name field\"\nmode: subagent\ncolor: \"#ff0000\"\ntools:\n  write: true\n---\n\n# No name agent\n"
	srcFile := filepath.Join(srcDir, "aether-noname.md")
	if err := os.WriteFile(srcFile, []byte(noNameContent), 0644); err != nil {
		t.Fatalf("write src file: %v", err)
	}

	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	result := syncDir(srcDir, destDir, syncOptions{
		validate: validateOpenCodeAgentFile,
	})

	// Should have errors (validation rejection)
	if len(result.errors) == 0 {
		t.Error("expected sync to reject agent file with missing name field")
	}

	// Should NOT have copied the file
	if result.copied != 0 {
		t.Errorf("expected 0 files copied, got %d", result.copied)
	}

	// Error should mention "name"
	foundNameError := false
	for _, e := range result.errors {
		if strings.Contains(e, "name") {
			foundNameError = true
			break
		}
	}
	if !foundNameError {
		t.Errorf("expected error to mention 'name', got: %v", result.errors)
	}
}

// TestAllOpenCodeAgentsHaveNameField verifies that every real OpenCode agent
// file in .opencode/agents/ has a non-empty name field in its frontmatter.
func TestAllOpenCodeAgentsHaveNameField(t *testing.T) {
	repoRoot, err := findOpenCodeRepoRoot()
	if err != nil {
		t.Skip("repo root not found, skipping real file validation")
	}
	agentsDir := filepath.Join(repoRoot, ".opencode", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		t.Fatalf("read agents dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		path := filepath.Join(agentsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		// Extract frontmatter and check for name field
		fm, err := extractYAMLFrontmatter(data)
		if err != nil {
			t.Errorf("%s: failed to parse frontmatter: %v", entry.Name(), err)
			continue
		}
		name, ok := fm["name"].(string)
		if !ok || strings.TrimSpace(name) == "" {
			t.Errorf("%s: missing or empty name field", entry.Name())
		}
		// Also verify the name matches the filename (without extension)
		expectedName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if name != expectedName {
			t.Errorf("%s: name field is %q, expected %q (should match filename)", entry.Name(), name, expectedName)
		}
	}
}
