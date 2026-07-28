package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var generatedCommandHeaderPattern = regexp.MustCompile(`^<!-- Aether-managed: runtime spec at (\.aether/commands/[^ ]+\.yaml)\. Synced by aether update\. -->$`)

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

func TestCouncilYamlSourceUsesRealRuntimeSubcommands(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, ".aether", "commands", "council.yaml"))
	if err != nil {
		t.Fatalf("read council yaml: %v", err)
	}

	text := string(content)
	for _, want := range []string{
		"council-deliberate",
		"council-budget-check",
		"council-advocate",
		"council-challenger",
		"council-sage",
		"council-history",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("council yaml missing real runtime subcommand %q", want)
		}
	}
	if strings.Contains(text, "aether council $ARGUMENTS") {
		t.Fatal("council yaml still references nonexistent `aether council` command")
	}
}

func TestRepoRootForCommandSourceTestSkipsGeneratedPackageArtifacts(t *testing.T) {
	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "Aether")
	for _, rel := range []string{
		filepath.Join(".aether", "commands"),
		filepath.Join(".claude", "commands", "ant"),
		filepath.Join(".opencode", "commands", "ant"),
		filepath.Join(".codex", "agents"),
		"cmd",
	} {
		if err := os.MkdirAll(filepath.Join(repoRoot, rel), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
	}
	for _, rel := range []string{"AGENTS.md", "go.mod", filepath.Join("cmd", "AGENTS.md")} {
		if err := os.WriteFile(filepath.Join(repoRoot, rel), []byte("test\n"), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	got, err := repoRootForCommandSourceTestFrom(filepath.Join(repoRoot, "cmd"))
	if err != nil {
		t.Fatalf("repo root lookup failed: %v", err)
	}
	if got != repoRoot {
		t.Fatalf("repo root = %s, want %s", got, repoRoot)
	}
}

func repoRootForCommandSourceTest() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return repoRootForCommandSourceTestFrom(wd)
}

func repoRootForCommandSourceTestFrom(wd string) (string, error) {
	candidates := []string{wd, filepath.Dir(wd), filepath.Dir(filepath.Dir(wd))}
	for _, candidate := range candidates {
		if isCommandSourceRepoRoot(candidate) {
			return candidate, nil
		}
	}

	return "", os.ErrNotExist
}

func isCommandSourceRepoRoot(candidate string) bool {
	for _, rel := range []string{
		"AGENTS.md",
		"go.mod",
		filepath.Join(".aether", "commands"),
		filepath.Join(".claude", "commands", "ant"),
		filepath.Join(".opencode", "commands", "ant"),
		filepath.Join(".codex", "agents"),
	} {
		if _, err := os.Stat(filepath.Join(candidate, rel)); err != nil {
			return false
		}
	}
	return true
}
