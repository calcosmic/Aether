package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var generatedCommandHeaderPattern = regexp.MustCompile(`^<!-- Aether-managed: runtime spec at (\.aether/commands/[^ ]+\.yaml)\. Synced by aether update\. -->$`)

func TestCommandSourceHygiene(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	if issues := auditCommandSourceHygiene(repoRoot, t.TempDir()); len(issues) > 0 {
		t.Fatalf("current command sources are not hygienic:\n%s", formatCommandSourceHygieneIssues(issues))
	}

	t.Run("missing canonical source", func(t *testing.T) {
		fixture := newCommandSourceHygieneFixture(t)
		if err := os.Remove(filepath.Join(fixture, ".aether", "commands", "status.yaml")); err != nil {
			t.Fatalf("remove canonical source: %v", err)
		}
		assertCommandSourceHygieneFailure(t, fixture, "no matching YAML source")
	})

	t.Run("missing required peer", func(t *testing.T) {
		fixture := newCommandSourceHygieneFixture(t)
		if err := os.Remove(filepath.Join(fixture, ".opencode", "commands", "ant", "status.md")); err != nil {
			t.Fatalf("remove required OpenCode peer: %v", err)
		}
		assertCommandSourceHygieneFailure(t, fixture, "generated wrapper missing for YAML source")
	})

	t.Run("stale generated body", func(t *testing.T) {
		fixture := newCommandSourceHygieneFixture(t)
		path := filepath.Join(fixture, ".claude", "commands", "ant", "status.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read Claude wrapper: %v", err)
		}
		data = []byte(strings.Replace(string(data), "If docs and runtime disagree", "If documentation and runtime disagree", 1))
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("write stale Claude wrapper: %v", err)
		}
		assertCommandSourceHygieneFailure(t, fixture, "wrapper body drift")
	})

	t.Run("false managed source header", func(t *testing.T) {
		fixture := newCommandSourceHygieneFixture(t)
		path := filepath.Join(fixture, ".claude", "commands", "ant", "status.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read Claude wrapper: %v", err)
		}
		data = []byte(strings.Replace(string(data), ".aether/commands/status.yaml", ".aether/commands/not-status.yaml", 1))
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("write false managed header: %v", err)
		}
		assertCommandSourceHygieneFailure(t, fixture, "points at the wrong YAML source")
	})

	t.Run("orphan managed output", func(t *testing.T) {
		fixture := newCommandSourceHygieneFixture(t)
		orphan := sourceCheckWrapperFixture(
			".aether/commands/orphan.yaml",
			"ant-orphan",
			"Orphan managed command fixture",
			"AETHER_OUTPUT_MODE=visual aether orphan",
		)
		writeFile(t, fixture, filepath.Join(".claude", "commands", "ant", "orphan.md"), orphan)
		assertCommandSourceHygieneFailure(t, fixture, "no matching YAML source")
	})

	t.Run("unmanaged custom decoy is ignored and preserved", func(t *testing.T) {
		fixture := newCommandSourceHygieneFixture(t)
		decoyPath := filepath.Join(fixture, ".claude", "commands", "custom", "status.md")
		decoy := []byte("# Project-owned status command\n\nThis is not Aether-managed.\n")
		writeFile(t, fixture, filepath.Join(".claude", "commands", "custom", "status.md"), decoy)

		if issues := auditCommandSourceHygiene(fixture, t.TempDir()); len(issues) > 0 {
			t.Fatalf("unmanaged custom decoy was reported as generated drift:\n%s", formatCommandSourceHygieneIssues(issues))
		}
		got, err := os.ReadFile(decoyPath)
		if err != nil {
			t.Fatalf("unmanaged custom decoy was removed: %v", err)
		}
		if string(got) != string(decoy) {
			t.Fatalf("unmanaged custom decoy changed:\nwant: %q\ngot:  %q", decoy, got)
		}
	})
}

// auditCommandSourceHygiene composes the production source checker, the
// production platform-sync rules, and the existing cross-platform semantic
// normalization. It stays read-only with respect to root: regeneration is
// directed into scratchRoot, which every caller owns as a disposable fixture.
func auditCommandSourceHygiene(root, scratchRoot string) []sourceCheckIssue {
	_, issues := checkGeneratedCommandSurfaces(root)
	issues = append(issues, checkCommandSourceHygieneDryRunSync(root, scratchRoot)...)
	issues = append(issues, checkCommandSourceHygienePeerParity(root)...)
	sortSourceCheckIssues(issues)
	return issues
}

func checkCommandSourceHygieneDryRunSync(root, scratchRoot string) []sourceCheckIssue {
	var issues []sourceCheckIssue
	for index, pair := range commandSourceHygieneSyncPairs() {
		sourceDir := filepath.Join(root, filepath.FromSlash(pair.srcRel))
		destination := filepath.Join(scratchRoot, fmt.Sprintf("platform-%d", index))
		result := syncDir(sourceDir, destination, syncOptions{
			cleanup:              pair.cleanup,
			preserveLocalChanges: pair.preserveLocalChanges,
			validate:             pair.validate,
			include:              pair.include,
			mapRelPath:           pair.mapRelPath,
			cleanupInclude:       pair.cleanupInclude,
		})
		for _, syncError := range result.errors {
			issues = append(issues, sourceCheckIssue{
				Area:    "commands",
				Path:    pair.srcRel,
				Message: "dry-run platform regeneration failed: " + syncError,
			})
		}

		for _, rel := range sourceCheckFiles(root, pair.srcRel, func(rel string) bool {
			return !strings.Contains(filepath.ToSlash(rel), "/") && filepath.Ext(rel) == ".md"
		}) {
			sourcePath := filepath.Join(sourceDir, filepath.FromSlash(rel))
			sourceData, err := os.ReadFile(sourcePath)
			if err != nil {
				issues = append(issues, sourceCheckIssue{Area: "commands", Path: filepath.ToSlash(filepath.Join(pair.srcRel, rel)), Message: "managed source is unreadable", Actual: err.Error()})
				continue
			}
			if !isGeneratedAetherCommandWrapper(sourceData) {
				continue
			}
			destinationRel := rel
			if pair.mapRelPath != nil {
				destinationRel = pair.mapRelPath(rel)
			}
			generatedData, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(destinationRel)))
			if err != nil {
				issues = append(issues, sourceCheckIssue{
					Area:     "commands",
					Path:     filepath.ToSlash(filepath.Join(pair.srcRel, rel)),
					Message:  "dry-run platform regeneration omitted managed output",
					Expected: destinationRel,
					Actual:   "missing",
				})
				continue
			}
			if string(generatedData) != string(sourceData) {
				issues = append(issues, sourceCheckIssue{
					Area:     "commands",
					Path:     filepath.ToSlash(filepath.Join(pair.srcRel, rel)),
					Message:  "managed output differs from dry-run platform regeneration",
					Expected: "byte-identical production sync output",
					Actual:   "content drift",
				})
			}
		}
	}
	return issues
}

func commandSourceHygieneSyncPairs() []installSyncPair {
	seenSources := map[string]bool{}
	var pairs []installSyncPair
	for _, pair := range installSyncPairs() {
		if !strings.HasPrefix(pair.label, "Commands (") || seenSources[pair.srcRel] {
			continue
		}
		seenSources[pair.srcRel] = true
		pairs = append(pairs, pair)
	}
	slices.SortFunc(pairs, func(left, right installSyncPair) int {
		return strings.Compare(left.srcRel, right.srcRel)
	})
	return pairs
}

func checkCommandSourceHygienePeerParity(root string) []sourceCheckIssue {
	dirs := commandSourceHygieneManagedSourceDirs()
	snapshots := make(map[string]map[string]commandWrapperSnapshot)
	for _, dir := range dirs {
		for _, rel := range sourceCheckFiles(root, dir, func(rel string) bool {
			return !strings.Contains(filepath.ToSlash(rel), "/") && filepath.Ext(rel) == ".md"
		}) {
			path := filepath.Join(root, filepath.FromSlash(dir), filepath.FromSlash(rel))
			data, err := os.ReadFile(path)
			if err != nil || !isGeneratedAetherCommandWrapper(data) {
				continue
			}
			firstLine := strings.SplitN(string(data), "\n", 2)[0]
			matches := sourceCheckGeneratedHeader.FindStringSubmatch(firstLine)
			if matches == nil {
				continue
			}
			name := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
			if snapshots[name] == nil {
				snapshots[name] = make(map[string]commandWrapperSnapshot, len(dirs))
			}
			snapshots[name][dir] = commandWrapperSnapshot{
				source: matches[1],
				body:   normalizeCommandWrapper(string(data)),
			}
		}
	}

	var issues []sourceCheckIssue
	var names []string
	for name := range snapshots {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		family := snapshots[name]
		var baseline commandWrapperSnapshot
		baselineDir := ""
		for _, dir := range dirs {
			snapshot, ok := family[dir]
			if !ok {
				issues = append(issues, sourceCheckIssue{
					Area:     "commands",
					Path:     filepath.ToSlash(filepath.Join(dir, name+".md")),
					Message:  "managed command family is missing a required platform peer",
					Expected: "managed wrapper",
					Actual:   "missing",
				})
				continue
			}
			if baselineDir == "" {
				baseline = snapshot
				baselineDir = dir
				continue
			}
			if snapshot.source != baseline.source {
				issues = append(issues, sourceCheckIssue{
					Area:     "commands",
					Path:     filepath.ToSlash(filepath.Join(dir, name+".md")),
					Message:  "managed command family has source-header drift",
					Expected: baseline.source,
					Actual:   snapshot.source,
				})
			}
			if snapshot.body != baseline.body {
				issues = append(issues, sourceCheckIssue{
					Area:     "commands",
					Path:     filepath.ToSlash(filepath.Join(dir, name+".md")),
					Message:  "wrapper body drift between managed platform peers",
					Expected: "semantic parity with " + baselineDir,
					Actual:   "content drift",
				})
			}
		}
	}
	return issues
}

func commandSourceHygieneManagedSourceDirs() []string {
	var dirs []string
	seen := map[string]bool{}
	for _, pair := range commandSourceHygieneSyncPairs() {
		if seen[pair.srcRel] {
			continue
		}
		seen[pair.srcRel] = true
		dirs = append(dirs, pair.srcRel)
	}
	slices.Sort(dirs)
	return dirs
}

func newCommandSourceHygieneFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, filepath.Join(".aether", "commands", "status.yaml"), []byte(`name: ant-status
description: "Show colony status through the Aether runtime"
source_of_truth: "Use the Go `+"`"+`aether`+"`"+` CLI as the source of truth."
runtime:
  command: "AETHER_OUTPUT_MODE=visual aether status $ARGUMENTS"
`))
	wrapper := sourceCheckWrapperFixture(
		".aether/commands/status.yaml",
		"ant-status",
		"Show colony status through the Aether runtime",
		"AETHER_OUTPUT_MODE=visual aether status",
	)
	writeFile(t, root, filepath.Join(".claude", "commands", "ant", "status.md"), wrapper)
	writeFile(t, root, filepath.Join(".opencode", "commands", "ant", "status.md"), wrapper)
	return root
}

func assertCommandSourceHygieneFailure(t *testing.T, root, want string) {
	t.Helper()
	issues := auditCommandSourceHygiene(root, t.TempDir())
	for _, issue := range issues {
		if strings.Contains(issue.Message, want) {
			return
		}
	}
	t.Fatalf("command source hygiene issues do not contain %q:\n%s", want, formatCommandSourceHygieneIssues(issues))
}

func formatCommandSourceHygieneIssues(issues []sourceCheckIssue) string {
	if len(issues) == 0 {
		return "(none)"
	}
	lines := make([]string, 0, len(issues))
	for _, issue := range issues {
		lines = append(lines, fmt.Sprintf("%s: %s", issue.Path, issue.Message))
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n")
}

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
