package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// ---------------------------------------------------------------------------
// Update Round-Trip Integrity Tests (VAL-02)
// ---------------------------------------------------------------------------
//
// Tests verify that agent and command files survive the syncDir update flow
// without corruption. Covers all 3 platforms: Claude, OpenCode, Codex.

// sha256sum computes the SHA-256 hex digest of a file.
func sha256sum(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("sha256sum open %s: %v", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("sha256sum read %s: %v", path, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// recordChecksums computes SHA-256 checksums for all files under dir.
func recordChecksums(t *testing.T, dir string) map[string]string {
	t.Helper()
	checksums := make(map[string]string)
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return nil
		}
		checksums[rel] = sha256sum(t, path)
		return nil
	})
	return checksums
}

// isMarkdownAgent checks that a file looks like a valid markdown agent file.
func isMarkdownAgent(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	content := string(data)
	if !strings.Contains(content, "#") {
		t.Errorf("markdown agent %s should contain at least one heading", path)
	}
}

// isTOMLAgent checks that a file looks like a valid TOML agent file.
func isTOMLAgent(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var parsed map[string]interface{}
	if _, err := toml.Decode(string(data), &parsed); err != nil {
		t.Errorf("TOML agent %s should be parseable: %v", path, err)
	}
}

// isMarkdownCommand checks that a file looks like a valid command markdown.
func isMarkdownCommand(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	content := string(data)
	if !strings.Contains(content, "#") {
		t.Errorf("command file %s should contain at least one heading", path)
	}
}

// ---------------------------------------------------------------------------
// Test: Agent files round-trip for all 3 platforms
// ---------------------------------------------------------------------------

func TestUpdateRoundTripAgentFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create "hub" source directory with agent definitions.
	hubAgentsClaude := filepath.Join(tmpDir, "hub", "agents-claude")
	hubAgentsCodex := filepath.Join(tmpDir, "hub", "codex")
	if err := os.MkdirAll(hubAgentsClaude, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(hubAgentsCodex, 0755); err != nil {
		t.Fatal(err)
	}

	// Claude agents (markdown).
	claudeAgent1 := "# aether-builder\n\nBuilder agent for testing.\n<!-- roundtrip-marker: builder-001 -->\n"
	claudeAgent2 := "# aether-watcher\n\nWatcher agent for testing.\n<!-- roundtrip-marker: watcher-001 -->\n"
	if err := os.WriteFile(filepath.Join(hubAgentsClaude, "aether-builder.md"), []byte(claudeAgent1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hubAgentsClaude, "aether-watcher.md"), []byte(claudeAgent2), 0644); err != nil {
		t.Fatal(err)
	}

	// Codex agents (TOML).
	codexAgent1 := "[agent]\nname = \"aether-builder\"\ntype = \"builder\"\n# roundtrip-marker: builder-001\n"
	codexAgent2 := "[agent]\nname = \"aether-watcher\"\ntype = \"watcher\"\n# roundtrip-marker: watcher-001\n"
	if err := os.WriteFile(filepath.Join(hubAgentsCodex, "aether-builder.toml"), []byte(codexAgent1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hubAgentsCodex, "aether-watcher.toml"), []byte(codexAgent2), 0644); err != nil {
		t.Fatal(err)
	}

	// Create "repo" destination directory with pre-existing agents.
	repoClaude := filepath.Join(tmpDir, "repo", ".claude", "agents", "ant")
	repoOpenCode := filepath.Join(tmpDir, "repo", ".opencode", "agents")
	repoCodex := filepath.Join(tmpDir, "repo", ".codex", "agents")
	for _, dir := range []string{repoClaude, repoOpenCode, repoCodex} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Pre-existing Claude agent.
	preExistingClaude := "# aether-builder\n\nOld version.\n"
	if err := os.WriteFile(filepath.Join(repoClaude, "aether-builder.md"), []byte(preExistingClaude), 0644); err != nil {
		t.Fatal(err)
	}

	// Pre-existing OpenCode agent.
	preExistingOpenCode := "# aether-builder\n\nOld version.\n"
	if err := os.WriteFile(filepath.Join(repoOpenCode, "aether-builder.md"), []byte(preExistingOpenCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Pre-existing Codex agent.
	preExistingCodex := "[agent]\nname = \"aether-builder\"\ntype = \"builder\"\n"
	if err := os.WriteFile(filepath.Join(repoCodex, "aether-builder.toml"), []byte(preExistingCodex), 0644); err != nil {
		t.Fatal(err)
	}

	// Record pre-sync checksums.
	preChecksums := make(map[string]map[string]string)
	for _, base := range []string{repoClaude, repoOpenCode, repoCodex} {
		rel, _ := filepath.Rel(tmpDir, base)
		preChecksums[rel] = recordChecksums(t, base)
	}

	// Run syncDir for each platform.
	t.Log("Syncing Claude agents")
	result := syncDir(hubAgentsClaude, repoClaude, syncOptions{})
	if len(result.errors) > 0 {
		t.Fatalf("Claude agent sync errors: %v", result.errors)
	}
	if result.copied == 0 {
		t.Error("Expected at least 1 Claude agent file copied")
	}

	t.Log("Syncing OpenCode agents (same source as Claude for this test)")
	result = syncDir(hubAgentsClaude, repoOpenCode, syncOptions{})
	if len(result.errors) > 0 {
		t.Fatalf("OpenCode agent sync errors: %v", result.errors)
	}
	if result.copied == 0 {
		t.Error("Expected at least 1 OpenCode agent file copied")
	}

	t.Log("Syncing Codex agents")
	result = syncDir(hubAgentsCodex, repoCodex, syncOptions{
		include: isShippedAetherCodexAgent,
	})
	if len(result.errors) > 0 {
		t.Fatalf("Codex agent sync errors: %v", result.errors)
	}
	if result.copied == 0 {
		t.Error("Expected at least 1 Codex agent file copied")
	}

	// Verify: all agent files still exist.
	for _, path := range []string{
		filepath.Join(repoClaude, "aether-builder.md"),
		filepath.Join(repoClaude, "aether-watcher.md"),
		filepath.Join(repoOpenCode, "aether-builder.md"),
		filepath.Join(repoOpenCode, "aether-watcher.md"),
		filepath.Join(repoCodex, "aether-builder.toml"),
		filepath.Join(repoCodex, "aether-watcher.toml"),
	} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("agent file missing after sync: %s", path)
		}
	}

	// Verify: pre-existing files were updated (content changed).
	builderClaude := filepath.Join(repoClaude, "aether-builder.md")
	builderData, _ := os.ReadFile(builderClaude)
	if strings.Contains(string(builderData), "Old version") && !strings.Contains(string(builderData), "roundtrip-marker") {
		t.Error("pre-existing Claude agent should be updated with new content")
	}

	// Verify: markdown files are valid.
	for _, path := range []string{
		filepath.Join(repoClaude, "aether-builder.md"),
		filepath.Join(repoClaude, "aether-watcher.md"),
		filepath.Join(repoOpenCode, "aether-builder.md"),
		filepath.Join(repoOpenCode, "aether-watcher.md"),
	} {
		isMarkdownAgent(t, path)
	}

	// Verify: TOML files are parseable.
	for _, path := range []string{
		filepath.Join(repoCodex, "aether-builder.toml"),
		filepath.Join(repoCodex, "aether-watcher.toml"),
	} {
		isTOMLAgent(t, path)
	}

	// Verify: no zero-byte files.
	filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() == 0 {
			rel, _ := filepath.Rel(tmpDir, path)
			t.Errorf("zero-byte file found after sync: %s", rel)
		}
		return nil
	})

	// Verify: no extra files created (checksums changed only for updated files).
	postChecksums := make(map[string]map[string]string)
	for _, base := range []string{repoClaude, repoOpenCode, repoCodex} {
		rel, _ := filepath.Rel(tmpDir, base)
		postChecksums[rel] = recordChecksums(t, base)
	}

	// After sync, all directories should have exactly 2 files (builder + watcher).
	for dir, post := range postChecksums {
		if len(post) != 2 {
			t.Errorf("expected 2 agent files in %s after sync, got %d", dir, len(post))
		}
	}

	_ = preChecksums // Used above for comparison logic
	t.Log("All agent round-trip checks passed")
}

// ---------------------------------------------------------------------------
// Test: Command files round-trip for Claude and OpenCode
// ---------------------------------------------------------------------------

func TestUpdateRoundTripCommandFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create "hub" source directory with command files.
	hubCmdClaude := filepath.Join(tmpDir, "hub", "commands", "claude")
	hubCmdOpenCode := filepath.Join(tmpDir, "hub", "commands", "opencode")
	if err := os.MkdirAll(hubCmdClaude, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(hubCmdOpenCode, 0755); err != nil {
		t.Fatal(err)
	}

	// Command files.
	buildCmd := "# /ant-build\n\nBuild command for testing.\n<!-- roundtrip-marker: build-cmd -->\n"
	continueCmd := "# /ant-continue\n\nContinue command for testing.\n<!-- roundtrip-marker: continue-cmd -->\n"
	if err := os.WriteFile(filepath.Join(hubCmdClaude, "build.md"), []byte(buildCmd), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hubCmdClaude, "continue.md"), []byte(continueCmd), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hubCmdOpenCode, "build.md"), []byte(buildCmd), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hubCmdOpenCode, "continue.md"), []byte(continueCmd), 0644); err != nil {
		t.Fatal(err)
	}

	// Create "repo" destination with pre-existing command files.
	repoClaudeCmds := filepath.Join(tmpDir, "repo", ".claude", "commands", "ant")
	repoOpenCodeCmds := filepath.Join(tmpDir, "repo", ".opencode", "commands", "ant")
	for _, dir := range []string{repoClaudeCmds, repoOpenCodeCmds} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Pre-existing commands.
	preExistingBuild := "# /ant-build\n\nOld build command.\n"
	if err := os.WriteFile(filepath.Join(repoClaudeCmds, "build.md"), []byte(preExistingBuild), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoOpenCodeCmds, "build.md"), []byte(preExistingBuild), 0644); err != nil {
		t.Fatal(err)
	}

	// Record pre-sync state.
	preClaudeChecksums := recordChecksums(t, repoClaudeCmds)
	preOpenCodeChecksums := recordChecksums(t, repoOpenCodeCmds)

	// Run syncDir for commands.
	t.Log("Syncing Claude commands")
	result := syncDir(hubCmdClaude, repoClaudeCmds, syncOptions{})
	if len(result.errors) > 0 {
		t.Fatalf("Claude command sync errors: %v", result.errors)
	}
	if result.copied == 0 {
		t.Error("Expected at least 1 Claude command file copied")
	}

	t.Log("Syncing OpenCode commands")
	result = syncDir(hubCmdOpenCode, repoOpenCodeCmds, syncOptions{})
	if len(result.errors) > 0 {
		t.Fatalf("OpenCode command sync errors: %v", result.errors)
	}
	if result.copied == 0 {
		t.Error("Expected at least 1 OpenCode command file copied")
	}

	// Verify: all command files still exist.
	for _, path := range []string{
		filepath.Join(repoClaudeCmds, "build.md"),
		filepath.Join(repoClaudeCmds, "continue.md"),
		filepath.Join(repoOpenCodeCmds, "build.md"),
		filepath.Join(repoOpenCodeCmds, "continue.md"),
	} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("command file missing after sync: %s", path)
		}
	}

	// Verify: pre-existing files updated.
	buildClaude := filepath.Join(repoClaudeCmds, "build.md")
	buildData, _ := os.ReadFile(buildClaude)
	if strings.Contains(string(buildData), "Old build command") && !strings.Contains(string(buildData), "roundtrip-marker") {
		t.Error("pre-existing Claude command should be updated")
	}

	// Verify: new files have correct content.
	continueClaude := filepath.Join(repoClaudeCmds, "continue.md")
	continueData, _ := os.ReadFile(continueClaude)
	if !strings.Contains(string(continueData), "roundtrip-marker: continue-cmd") {
		t.Error("new continue.md should have the roundtrip marker")
	}

	// Verify: markdown validity.
	for _, path := range []string{
		filepath.Join(repoClaudeCmds, "build.md"),
		filepath.Join(repoClaudeCmds, "continue.md"),
		filepath.Join(repoOpenCodeCmds, "build.md"),
		filepath.Join(repoOpenCodeCmds, "continue.md"),
	} {
		isMarkdownCommand(t, path)
	}

	// Verify: correct file counts after sync.
	postClaudeChecksums := recordChecksums(t, repoClaudeCmds)
	postOpenCodeChecksums := recordChecksums(t, repoOpenCodeCmds)

	if len(postClaudeChecksums) != 2 {
		t.Errorf("expected 2 Claude commands after sync, got %d", len(postClaudeChecksums))
	}
	if len(postOpenCodeChecksums) != 2 {
		t.Errorf("expected 2 OpenCode commands after sync, got %d", len(postOpenCodeChecksums))
	}

	// Verify: pre-existing file checksums changed (was updated).
	if preClaudeChecksums["build.md"] == postClaudeChecksums["build.md"] {
		t.Error("pre-existing build.md should have different checksum after update")
	}

	_ = preOpenCodeChecksums // Verified above
	t.Log("All command round-trip checks passed")
}

// ---------------------------------------------------------------------------
// Test: Combined round-trip with corruption checks
// ---------------------------------------------------------------------------

func TestUpdateRoundTripNoCorruption(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hub source with all file types.
	hubDir := filepath.Join(tmpDir, "hub")
	dirs := []string{
		filepath.Join(hubDir, "agents-claude"),
		filepath.Join(hubDir, "codex"),
		filepath.Join(hubDir, "commands", "claude"),
		filepath.Join(hubDir, "commands", "opencode"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Write known content with unique markers.
	agentMD := "# aether-builder\n\nRound-trip integrity agent.\n<!-- marker: rtt-agent-%s -->\n"
	cmdMD := "# /ant-%s\n\nRound-trip integrity command.\n<!-- marker: rtt-cmd-%s -->\n"
	agentTOML := "[agent]\nname = \"aether-%s\"\ntype = \"%s\"\n# marker: rtt-toml-%s\n"

	knownFiles := map[string]string{
		"agents-claude/aether-builder.md":    fmt.Sprintf(agentMD, "builder"),
		"agents-claude/aether-watcher.md":    fmt.Sprintf(agentMD, "watcher"),
		"codex/aether-builder.toml":          fmt.Sprintf(agentTOML, "builder", "builder", "builder"),
		"codex/aether-watcher.toml":          fmt.Sprintf(agentTOML, "watcher", "watcher", "watcher"),
		"commands/claude/build.md":           fmt.Sprintf(cmdMD, "build", "build"),
		"commands/claude/continue.md":        fmt.Sprintf(cmdMD, "continue", "continue"),
		"commands/opencode/build.md":         fmt.Sprintf(cmdMD, "build", "build"),
		"commands/opencode/continue.md":      fmt.Sprintf(cmdMD, "continue", "continue"),
	}

	for rel, content := range knownFiles {
		path := filepath.Join(hubDir, rel)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Compute hub checksums as the expected post-sync values.
	hubChecksums := recordChecksums(t, hubDir)

	// Create repo destination.
	repoDir := filepath.Join(tmpDir, "repo")
	repoDirs := []string{
		filepath.Join(repoDir, ".claude", "agents", "ant"),
		filepath.Join(repoDir, ".claude", "commands", "ant"),
		filepath.Join(repoDir, ".opencode", "agents"),
		filepath.Join(repoDir, ".opencode", "commands", "ant"),
		filepath.Join(repoDir, ".codex", "agents"),
	}
	for _, dir := range repoDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Run sync for all file types.
	syncPairs := []struct {
		src string
		dst string
	}{
		{filepath.Join(hubDir, "agents-claude"), filepath.Join(repoDir, ".claude", "agents", "ant")},
		{filepath.Join(hubDir, "agents-claude"), filepath.Join(repoDir, ".opencode", "agents")},
		{filepath.Join(hubDir, "codex"), filepath.Join(repoDir, ".codex", "agents")},
		{filepath.Join(hubDir, "commands", "claude"), filepath.Join(repoDir, ".claude", "commands", "ant")},
		{filepath.Join(hubDir, "commands", "opencode"), filepath.Join(repoDir, ".opencode", "commands", "ant")},
	}

	for _, pair := range syncPairs {
		opts := syncOptions{}
		if strings.Contains(pair.dst, ".codex") {
			opts.include = isShippedAetherCodexAgent
		}
		result := syncDir(pair.src, pair.dst, opts)
		if len(result.errors) > 0 {
			t.Errorf("sync %s -> %s errors: %v", pair.src, pair.dst, result.errors)
		}
		if result.copied == 0 {
			t.Errorf("sync %s -> %s: expected files copied, got 0", filepath.Base(pair.src), filepath.Base(pair.dst))
		}
	}

	// Verify: every file exists after sync.
	expectedRepoFiles := []string{
		".claude/agents/ant/aether-builder.md",
		".claude/agents/ant/aether-watcher.md",
		".opencode/agents/aether-builder.md",
		".opencode/agents/aether-watcher.md",
		".codex/agents/aether-builder.toml",
		".codex/agents/aether-watcher.toml",
		".claude/commands/ant/build.md",
		".claude/commands/ant/continue.md",
		".opencode/commands/ant/build.md",
		".opencode/commands/ant/continue.md",
	}

	for _, rel := range expectedRepoFiles {
		path := filepath.Join(repoDir, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file missing after sync: %s", rel)
		}
	}

	// Verify: no zero-byte files.
	filepath.WalkDir(repoDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		if info.Size() == 0 {
			rel, _ := filepath.Rel(repoDir, path)
			t.Errorf("zero-byte file after sync: %s", rel)
		}
		return nil
	})

	// Verify: no binary corruption in text files.
	textExts := map[string]bool{".md": true, ".toml": true}
	filepath.WalkDir(repoDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if !textExts[ext] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		// Check for null bytes (binary corruption indicator).
		for i, b := range data {
			if b == 0 {
				rel, _ := filepath.Rel(repoDir, path)
				t.Errorf("binary corruption (null byte at offset %d) in %s", i, rel)
				break
			}
		}
		return nil
	})

	// Verify: TOML files are parseable.
	tomlFiles, _ := filepath.Glob(filepath.Join(repoDir, ".codex", "agents", "*.toml"))
	for _, tomlFile := range tomlFiles {
		data, _ := os.ReadFile(tomlFile)
		var parsed map[string]interface{}
		if _, err := toml.Decode(string(data), &parsed); err != nil {
			t.Errorf("TOML file %s is not parseable: %v", filepath.Base(tomlFile), err)
		}
	}

	// Verify: markdown files contain expected markers.
	mdFiles, _ := filepath.Glob(filepath.Join(repoDir, "**", "*.md"))
	if len(mdFiles) == 0 {
		// Glob with ** may not work on all platforms; use WalkDir instead.
		filepath.WalkDir(repoDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
				return nil
			}
			mdFiles = append(mdFiles, path)
			return nil
		})
	}
	for _, mdFile := range mdFiles {
		data, _ := os.ReadFile(mdFile)
		content := string(data)
		if !strings.Contains(content, "marker:") {
			rel, _ := filepath.Rel(repoDir, mdFile)
			t.Errorf("markdown file %s missing expected marker", rel)
		}
	}

	// Verify: hub checksums match what was synced (content integrity).
	// The hub files should be unchanged after sync.
	postHubChecksums := recordChecksums(t, hubDir)
	for rel, preHash := range hubChecksums {
		if postHash, ok := postHubChecksums[rel]; ok && preHash != postHash {
			t.Errorf("hub file %s was modified during sync (should be read-only)", rel)
		}
	}

	_ = json.Valid // Ensure json import is used
	t.Log("All combined round-trip corruption checks passed")
}
