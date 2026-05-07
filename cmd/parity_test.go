package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ParitySnapshot captures command names from all 5 surfaces for golden testing.
type ParitySnapshot struct {
	YAMLCatalog            []string          `json:"yaml_catalog"`
	ClaudeWrapperCatalog   []string          `json:"claude_wrapper_catalog"`
	OpenCodeWrapperCatalog []string          `json:"opencode_wrapper_catalog"`
	CommandGuideCatalog    []string          `json:"command_guide_catalog"`
	RuntimeCatalogNames    []string          `json:"runtime_catalog_names"`
	RuntimeResolvedNames   map[string]string `json:"runtime_resolved_names"`
}

// yamlToRuntimeName maps YAML slash-command names to their Cobra primary names.
// Only entries where the YAML name differs from the Cobra Use field are listed.
// Commands not listed here have a direct 1:1 name match with the runtime.
var yamlToRuntimeName = map[string]string{
	"assumptions":   "assumptions-analyze",
	"export-signals": "pheromone-export-xml",
	"import-signals": "pheromone-import-xml",
	"flag":           "flag-add",
	"flags":          "flag-list",
	"insert-phase":   "phase-insert",
	"memory-details": "memory-metrics",
	"patrol":         "patrol-check",
	"pheromones":     "pheromone-display",
	"profile":        "profile-read",
	"resume":         "resume-colony",
	"shelf":          "shelf-list",
}

// promptOnlyCommands have no runtime command -- they are pure LLM prompt wrappers.
var promptOnlyCommands = map[string]bool{
	"archaeology": true,
	"chaos":       true,
	"dream":       true,
	"interpret":   true,
	"organize":    true,
}

// cobraBuiltinCommands are excluded by IsAvailableCommand() but have YAML+wrappers.
var cobraBuiltinCommands = map[string]bool{
	"help": true,
}

// extractYAMLNames reads .aether/commands/ and returns sorted YAML command names.
func extractYAMLNames(t *testing.T) []string {
	t.Helper()
	return yamlCommandNamesForGuideTest(t)
}

// extractWrapperNames reads a wrapper directory and returns sorted command names.
func extractWrapperNames(t *testing.T, repoRoot, subdir string) []string {
	t.Helper()
	dir := filepath.Join(repoRoot, subdir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".md"))
	}
	sort.Strings(names)
	return names
}

// extractCommandGuideNames returns sorted keys from commandGuideCatalog().
func extractCommandGuideNames() []string {
	catalog := commandGuideCatalog()
	names := make([]string, 0, len(catalog))
	for name := range catalog {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// extractRuntimeNames returns sorted command names from the Cobra runtime.
func extractRuntimeNames() []string {
	catalog := buildAuditCatalog(rootCmd)
	names := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		names = append(names, entry.Name)
	}
	sort.Strings(names)
	return names
}

// resolveYAMLName maps a YAML name to its Cobra primary name.
// If no mapping exists, returns the YAML name unchanged.
func resolveYAMLName(yamlName string) string {
	if resolved, ok := yamlToRuntimeName[yamlName]; ok {
		return resolved
	}
	return yamlName
}

// isExcludedFromRuntime returns true if a YAML name is expected to be absent
// from the runtime catalog (prompt-only or Cobra builtin).
func isExcludedFromRuntime(name string) bool {
	return promptOnlyCommands[name] || cobraBuiltinCommands[name]
}

func TestPlatformParityGolden(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	yamlNames := extractYAMLNames(t)
	claudeNames := extractWrapperNames(t, repoRoot, filepath.Join(".claude", "commands", "ant"))
	opencodeNames := extractWrapperNames(t, repoRoot, filepath.Join(".opencode", "commands", "ant"))
	guideNames := extractCommandGuideNames()
	runtimeNames := extractRuntimeNames()

	// Build resolved names map -- only entries where they differ.
	resolved := make(map[string]string)
	for _, name := range yamlNames {
		if resolvedName, ok := yamlToRuntimeName[name]; ok {
			resolved[name] = resolvedName
		}
	}

	snapshot := ParitySnapshot{
		YAMLCatalog:            yamlNames,
		ClaudeWrapperCatalog:   claudeNames,
		OpenCodeWrapperCatalog: opencodeNames,
		CommandGuideCatalog:    guideNames,
		RuntimeCatalogNames:    runtimeNames,
		RuntimeResolvedNames:   resolved,
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("marshal parity snapshot: %v", err)
	}

	goldenPath := "testdata/parity_snapshot.json"

	if *updateGolden {
		if err := os.WriteFile(goldenPath, append(data, '\n'), 0644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		t.Logf("golden file updated: %s", goldenPath)
		return
	}

	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file: %v (run with -update-golden to create)", err)
	}

	got := string(data) + "\n"
	want := string(golden)
	if got != want {
		t.Errorf("parity snapshot golden mismatch; run with -update-golden to refresh")
		t.Logf("  got  %d bytes", len(got))
		t.Logf("  want %d bytes", len(want))
	}
}

func TestNoPhantomCommands(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	// Build runtime name set.
	runtimeCatalog := buildAuditCatalog(rootCmd)
	runtimeNames := make(map[string]bool, len(runtimeCatalog))
	for _, entry := range runtimeCatalog {
		runtimeNames[entry.Name] = true
	}

	var phantoms []string

	// Check YAML names.
	yamlNames := extractYAMLNames(t)
	for _, name := range yamlNames {
		if isExcludedFromRuntime(name) {
			continue
		}
		resolved := resolveYAMLName(name)
		if !runtimeNames[resolved] {
			phantoms = append(phantoms, fmt.Sprintf("YAML command %q resolves to %q but not in runtime", name, resolved))
		}
	}

	// Check Claude wrapper names.
	claudeNames := extractWrapperNames(t, repoRoot, filepath.Join(".claude", "commands", "ant"))
	for _, name := range claudeNames {
		if isExcludedFromRuntime(name) {
			continue
		}
		resolved := resolveYAMLName(name)
		if !runtimeNames[resolved] {
			phantoms = append(phantoms, fmt.Sprintf("Claude wrapper %q resolves to %q but not in runtime", name, resolved))
		}
	}

	// Check OpenCode wrapper names.
	opencodeNames := extractWrapperNames(t, repoRoot, filepath.Join(".opencode", "commands", "ant"))
	for _, name := range opencodeNames {
		if isExcludedFromRuntime(name) {
			continue
		}
		resolved := resolveYAMLName(name)
		if !runtimeNames[resolved] {
			phantoms = append(phantoms, fmt.Sprintf("OpenCode wrapper %q resolves to %q but not in runtime", name, resolved))
		}
	}

	// Check command-guide names.
	guideNames := extractCommandGuideNames()
	for _, name := range guideNames {
		if isExcludedFromRuntime(name) {
			continue
		}
		resolved := resolveYAMLName(name)
		if !runtimeNames[resolved] {
			phantoms = append(phantoms, fmt.Sprintf("command-guide %q resolves to %q but not in runtime", name, resolved))
		}
	}

	if len(phantoms) > 0 {
		t.Fatalf("phantom commands detected:\n%s", strings.Join(phantoms, "\n"))
	}
}

func TestAllYamlHaveWrappersAndGuide(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	yamlNames := extractYAMLNames(t)
	claudeNames := extractWrapperNames(t, repoRoot, filepath.Join(".claude", "commands", "ant"))
	opencodeNames := extractWrapperNames(t, repoRoot, filepath.Join(".opencode", "commands", "ant"))
	guideCatalog := commandGuideCatalog()

	yamlSet := make(map[string]bool, len(yamlNames))
	for _, name := range yamlNames {
		yamlSet[name] = true
	}
	claudeSet := make(map[string]bool, len(claudeNames))
	for _, name := range claudeNames {
		claudeSet[name] = true
	}
	opencodeSet := make(map[string]bool, len(opencodeNames))
	for _, name := range opencodeNames {
		opencodeSet[name] = true
	}

	var drift []string

	// Each YAML command must have both wrappers and a guide entry.
	for _, name := range yamlNames {
		if !claudeSet[name] {
			drift = append(drift, fmt.Sprintf("YAML command %q has no Claude wrapper", name))
		}
		if !opencodeSet[name] {
			drift = append(drift, fmt.Sprintf("YAML command %q has no OpenCode wrapper", name))
		}
		if _, ok := guideCatalog[name]; !ok {
			drift = append(drift, fmt.Sprintf("YAML command %q has no command-guide entry", name))
		}
	}

	// Each wrapper should have a matching YAML source (no orphan wrappers).
	for _, name := range claudeNames {
		if !yamlSet[name] {
			drift = append(drift, fmt.Sprintf("Claude wrapper %q has no YAML source", name))
		}
	}
	for _, name := range opencodeNames {
		if !yamlSet[name] {
			drift = append(drift, fmt.Sprintf("OpenCode wrapper %q has no YAML source", name))
		}
	}

	if len(drift) > 0 {
		t.Fatalf("YAML/wrapper/guide parity drift:\n%s", strings.Join(drift, "\n"))
	}
}

func TestAliasResolutionCompleteness(t *testing.T) {
	// Every entry in yamlToRuntimeName must resolve to an actual runtime command.
	runtimeCatalog := buildAuditCatalog(rootCmd)
	runtimeNames := make(map[string]bool, len(runtimeCatalog))
	for _, entry := range runtimeCatalog {
		runtimeNames[entry.Name] = true
	}

	var stale []string
	for yamlName, runtimeName := range yamlToRuntimeName {
		if !runtimeNames[runtimeName] {
			stale = append(stale, fmt.Sprintf("alias %q -> %q but %q not in runtime catalog", yamlName, runtimeName, runtimeName))
		}
	}

	if len(stale) > 0 {
		t.Fatalf("stale alias resolution entries:\n%s", strings.Join(stale, "\n"))
	}
}
