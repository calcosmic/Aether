package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var generatedCommandHeaderPattern = regexp.MustCompile(`^<!-- Generated from (\.aether/commands/[^ ]+\.yaml) - DO NOT EDIT DIRECTLY -->$`)

func TestCommandWrappersReferenceRealYamlSources(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperDirs := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant"),
	}

	var missingHeaders []string
	var missingSources []string

	for _, dir := range wrapperDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}

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
				relativePath, err := filepath.Rel(repoRoot, wrapperPath)
				if err != nil {
					t.Fatalf("relative path for %s: %v", wrapperPath, err)
				}
				missingHeaders = append(missingHeaders, relativePath)
				continue
			}

			sourcePath := filepath.Join(repoRoot, matches[1])
			if _, err := os.Stat(sourcePath); err != nil {
				missingSources = append(missingSources, matches[1]+" <- "+filepath.Base(wrapperPath))
			}
		}
	}

	if len(missingHeaders) > 0 {
		slices.Sort(missingHeaders)
		t.Fatalf("command wrappers missing generated-from headers:\n%s", strings.Join(missingHeaders, "\n"))
	}

	if len(missingSources) > 0 {
		slices.Sort(missingSources)
		t.Fatalf("generated-from headers reference missing YAML sources:\n%s", strings.Join(missingSources, "\n"))
	}
}

func TestSourceOfTruthMapDocumentsWrapperYamlOwnership(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, ".aether", "docs", "source-of-truth-map.md"))
	if err != nil {
		t.Fatalf("read source-of-truth map: %v", err)
	}

	text := string(content)
	required := []string{
		".aether/commands/*.yaml",
		"Slash-command wrapper specs",
		".claude/commands/ant/*.md",
		".opencode/commands/ant/*.md",
	}
	for _, want := range required {
		if !strings.Contains(text, want) {
			t.Fatalf("source-of-truth map missing %q", want)
		}
	}
}

func repoRootForCommandSourceTest() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	candidates := []string{wd, filepath.Dir(wd)}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "AGENTS.md")); err == nil {
			return candidate, nil
		}
	}

	return "", os.ErrNotExist
}
