package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateRoundTripLeavesGlobalAssetsOutOfRepo(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()
	hubSystem := filepath.Join(hubDir, "system")
	for _, dir := range []string{
		filepath.Join(hubSystem, "commands", "claude"),
		filepath.Join(hubSystem, "commands", "opencode"),
		filepath.Join(hubSystem, "agents"),
		filepath.Join(hubSystem, "codex"),
		filepath.Join(hubSystem, "skills", "colony", "build-discipline"),
		filepath.Join(hubSystem, "docs"),
		filepath.Join(hubSystem, "templates"),
		filepath.Join(hubSystem, "utils"),
		filepath.Join(hubSystem, "exchange"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}
	}
	files := map[string]string{
		filepath.Join("commands", "claude", "build.md"):                   "# Build\n",
		filepath.Join("commands", "opencode", "build.md"):                 "# Build\n",
		filepath.Join("agents", "aether-builder.md"):                      "# Builder\n",
		filepath.Join("codex", "aether-builder.toml"):                     string(validCodexAgentTOML("aether-builder", "builder")),
		filepath.Join("skills", "colony", "build-discipline", "SKILL.md"): "# Skill\n",
		filepath.Join("docs", "guide.md"):                                 "# Guide\n",
		filepath.Join("templates", "colony-state.template.json"):          "{}\n",
		filepath.Join("utils", "helper.md"):                               "# Helper\n",
		filepath.Join("exchange", "pheromones.xsd"):                       "<schema />\n",
		"workers.md": "# Workers\n",
	}
	for rel, content := range files {
		path := filepath.Join(hubSystem, rel)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	result := runUpdateSync(hubDir, repoDir, true)
	if len(result.errors) > 0 {
		t.Fatalf("runUpdateSync errors: %v", result.errors)
	}

	for _, rel := range []string{
		filepath.Join(".claude", "commands", "ant-build.md"),
		filepath.Join(".opencode", "commands", "ant", "build.md"),
		filepath.Join(".opencode", "agents", "aether-builder.md"),
		filepath.Join(".claude", "agents", "ant", "aether-builder.md"),
		filepath.Join(".codex", "agents", "aether-builder.toml"),
		filepath.Join(".codex", "skills", "aether"),
		filepath.Join(".aether", "docs", "guide.md"),
		filepath.Join(".aether", "templates", "colony-state.template.json"),
		filepath.Join(".aether", "utils", "helper.md"),
		filepath.Join(".aether", "exchange", "pheromones.xsd"),
		filepath.Join(".aether", "workers.md"),
	} {
		if _, err := os.Stat(filepath.Join(repoDir, rel)); err == nil {
			t.Fatalf("global asset %s should not be copied into repo", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", rel, err)
		}
	}
}
