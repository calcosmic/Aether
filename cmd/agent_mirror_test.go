package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetiredPackagedAgentMirrorsAreAbsent(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, rel := range []string{
		".aether/agents-claude",
		".aether/agents-codex",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(rel))); err == nil {
			t.Fatalf("retired packaged agent mirror exists: %s", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", rel, err)
		}
	}
}

func TestCanonicalAgentSourcesRemainAligned(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	claudeDir := filepath.Join(repoRoot, ".claude", "agents", "ant")
	opencodeDir := filepath.Join(repoRoot, ".opencode", "agents")
	codexDir := filepath.Join(repoRoot, ".codex", "agents")

	claudeNames := agentBaseNames(t, claudeDir, ".md")
	opencodeNames := agentBaseNames(t, opencodeDir, ".md")
	codexNames := listShippedAetherCodexAgentBaseNames(t, codexDir)

	assertSameAgentBaseNames(t, "OpenCode", claudeNames, opencodeNames)
	assertSameAgentBaseNames(t, "Codex", claudeNames, codexNames)
}

func agentBaseNames(t *testing.T, dir, ext string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ext {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ext))
	}
	return names
}

func assertSameAgentBaseNames(t *testing.T, platform string, want, got []string) {
	t.Helper()
	wantSet := map[string]bool{}
	for _, name := range want {
		wantSet[name] = true
	}
	gotSet := map[string]bool{}
	for _, name := range got {
		gotSet[name] = true
	}
	for name := range wantSet {
		if !gotSet[name] {
			t.Fatalf("%s missing canonical agent %s", platform, name)
		}
	}
	for name := range gotSet {
		if !wantSet[name] {
			t.Fatalf("%s has extra agent %s", platform, name)
		}
	}
}
