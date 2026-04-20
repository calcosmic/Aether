package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type commandWrapperSnapshot struct {
	source string
	body   string
}

func TestClaudeOpenCodeCommandParity(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	claudeWrappers := loadYAMLBackedCommandWrappers(t, repoRoot, filepath.Join(repoRoot, ".claude", "commands", "ant"))
	opencodeWrappers := loadYAMLBackedCommandWrappers(t, repoRoot, filepath.Join(repoRoot, ".opencode", "commands", "ant"))

	seen := map[string]bool{}
	var basenames []string
	for name := range claudeWrappers {
		if !seen[name] {
			basenames = append(basenames, name)
			seen[name] = true
		}
	}
	for name := range opencodeWrappers {
		if !seen[name] {
			basenames = append(basenames, name)
			seen[name] = true
		}
	}
	slices.Sort(basenames)

	var drift []string
	for _, basename := range basenames {
		claudeSnapshot, claudeOK := claudeWrappers[basename]
		opencodeSnapshot, opencodeOK := opencodeWrappers[basename]

		switch {
		case !claudeOK:
			drift = append(drift, fmt.Sprintf("%s missing from .claude/commands/ant", basename))
		case !opencodeOK:
			drift = append(drift, fmt.Sprintf("%s missing from .opencode/commands/ant", basename))
		case claudeSnapshot.source != opencodeSnapshot.source:
			drift = append(drift, fmt.Sprintf("%s generated-from header drift: Claude=%s OpenCode=%s", basename, claudeSnapshot.source, opencodeSnapshot.source))
		case claudeSnapshot.body != opencodeSnapshot.body:
			drift = append(drift, fmt.Sprintf("%s wrapper body drift between Claude and OpenCode", basename))
		}
	}

	if len(drift) > 0 {
		t.Fatalf("Claude/OpenCode command parity drift:\n%s", strings.Join(drift, "\n"))
	}
}

func loadYAMLBackedCommandWrappers(t *testing.T, repoRoot, dir string) map[string]commandWrapperSnapshot {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	wrappers := make(map[string]commandWrapperSnapshot, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		wrapperPath := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPath, err)
		}

		firstLine := strings.SplitN(string(content), "\n", 2)[0]
		matches := generatedCommandHeaderPattern.FindStringSubmatch(firstLine)
		if matches == nil {
			relativePath, relErr := filepath.Rel(repoRoot, wrapperPath)
			if relErr != nil {
				t.Fatalf("relative path for %s: %v", wrapperPath, relErr)
			}
			t.Fatalf("%s is missing a generated-from header", relativePath)
		}

		basename := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		wrappers[basename] = commandWrapperSnapshot{
			source: matches[1],
			body:   normalizeCommandWrapper(string(content)),
		}
	}

	return wrappers
}

func normalizeCommandWrapper(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "Use the runtime CLI and current slash-command surface as the source of truth.", "Use the runtime CLI as the source of truth.")
	content = strings.ReplaceAll(content, "- Use AskUserQuestion with 3 options: proceed, revise goal, cancel.", "- Ask with 3 options: proceed, revise goal, cancel.")
	return strings.TrimSpace(content)
}
