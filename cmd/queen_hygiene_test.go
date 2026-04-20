package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestQueenWisdomHygiene(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	queenPath := filepath.Join(repoRoot, ".aether", "QUEEN.md")
	content, err := os.ReadFile(queenPath)
	if err != nil {
		t.Fatalf("read %s: %v", queenPath, err)
	}

	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	for _, marker := range []string{"test-colony", "test content", "lint:sync", "npm run lint"} {
		if strings.Contains(text, marker) {
			t.Fatalf("QUEEN.md contains blocked junk marker %q", marker)
		}
	}

	allowedDuplicates := map[string]bool{
		"---":                                true,
		"|------|--------|------|---------|": true,
	}

	counts := map[string]int{}
	for _, rawLine := range strings.Split(text, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || allowedDuplicates[line] {
			continue
		}
		counts[line]++
	}

	var duplicates []string
	for line, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, line)
		}
	}
	slices.Sort(duplicates)

	if len(duplicates) > 0 {
		t.Fatalf("QUEEN.md contains non-structural duplicate wisdom lines:\n%s", strings.Join(duplicates, "\n"))
	}
}
