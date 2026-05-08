package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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
	seen := make(map[string]bool, len(catalog))
	names := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		if !seen[entry.Name] {
			seen[entry.Name] = true
			names = append(names, entry.Name)
		}
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
		gotLines := strings.Split(got, "\n")
		wantLines := strings.Split(want, "\n")
		for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
			if gotLines[i] != wantLines[i] {
				t.Logf("first diff at line %d:\n  got:  %s\n  want: %s", i+1, gotLines[i], wantLines[i])
				break
			}
		}
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

// --- Flag Parity (Task 1) ---

// yamlCommandFlags maps YAML command name -> sorted flag names.
func extractFlagsFromYAML(t *testing.T, repoRoot string) map[string][]string {
	t.Helper()
	dir := filepath.Join(repoRoot, ".aether", "commands")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read YAML commands dir: %v", err)
	}
	result := make(map[string][]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var doc struct {
			Flags []struct {
				Name string `yaml:"name"`
			} `yaml:"flags"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		var flags []string
		for _, f := range doc.Flags {
			if f.Name != "" {
				flags = append(flags, f.Name)
			}
		}
		sort.Strings(flags)
		result[name] = flags
	}
	return result
}

// runtimeCommandFlags maps runtime command name -> sorted flag names.
func extractRuntimeFlags() map[string][]string {
	catalog := buildAuditCatalog(rootCmd)
	result := make(map[string][]string, len(catalog))
	for _, entry := range catalog {
		result[entry.Name] = entry.Flags
	}
	return result
}

func TestFlagParityAcrossSurfaces(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	yamlFlags := extractFlagsFromYAML(t, repoRoot)
	runtimeFlags := extractRuntimeFlags()

	// Build resolved runtime flag map (alias-aware).
	resolvedRuntimeFlags := make(map[string][]string)
	for cmdName, flags := range runtimeFlags {
		resolvedRuntimeFlags[cmdName] = flags
	}

	var mismatches []string
	for yamlName, yFlags := range yamlFlags {
		// Skip prompt-only commands (no runtime equivalent).
		if isExcludedFromRuntime(yamlName) {
			continue
		}
		// Skip commands where YAML does not define flags -- parity cannot be verified
		// when one surface has no data. Most YAML files omit flags because they are
		// documented in the runtime source of truth instead.
		if len(yFlags) == 0 {
			continue
		}
		runtimeName := resolveYAMLName(yamlName)
		rFlags, ok := resolvedRuntimeFlags[runtimeName]
		if !ok {
			mismatches = append(mismatches, fmt.Sprintf("%q: runtime command %q not found", yamlName, runtimeName))
			continue
		}

		// Compare flag sets using map-based comparison.
		ySet := make(map[string]bool, len(yFlags))
		for _, f := range yFlags {
			ySet[f] = true
		}
		rSet := make(map[string]bool, len(rFlags))
		for _, f := range rFlags {
			rSet[f] = true
		}

		// Check YAML flags present in runtime.
		for f := range ySet {
			if !rSet[f] {
				mismatches = append(mismatches, fmt.Sprintf("%q: flag %q in YAML but not in runtime", yamlName, f))
			}
		}

		// Check runtime flags present in YAML.
		for f := range rSet {
			if !ySet[f] {
				mismatches = append(mismatches, fmt.Sprintf("%q: flag %q in runtime but not in YAML", yamlName, f))
			}
		}
	}

	if len(mismatches) > 0 {
		t.Fatalf("flag parity mismatches:\n%s", strings.Join(mismatches, "\n"))
	}
}

// --- YAML/Wrapper Contract Verification (Task 2) ---

// yamlContractInfo captures frontmatter fields we verify for each YAML command.
type yamlContractInfo struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Runtime     struct {
		Command string `yaml:"command"`
	} `yaml:"runtime"`
	Guardrails []string `yaml:"guardrails"`
}

// wrapperContractInfo captures frontmatter fields we verify for each wrapper.
type wrapperContractInfo struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func parseYAMLContracts(t *testing.T, repoRoot string) map[string]yamlContractInfo {
	t.Helper()
	dir := filepath.Join(repoRoot, ".aether", "commands")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read YAML commands dir: %v", err)
	}
	result := make(map[string]yamlContractInfo)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var info yamlContractInfo
		if err := yaml.Unmarshal(data, &info); err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		result[name] = info
	}
	return result
}

func parseWrapperContracts(t *testing.T, repoRoot, subdir string) map[string]wrapperContractInfo {
	t.Helper()
	dir := filepath.Join(repoRoot, subdir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	result := make(map[string]wrapperContractInfo)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		content := string(data)
		// Extract frontmatter between --- delimiters.
		if !strings.HasPrefix(content, "---") {
			continue
		}
		end := strings.Index(content[3:], "---")
		if end == -1 {
			continue
		}
		frontmatter := content[3 : 3+end]
		var info wrapperContractInfo
		if err := yaml.Unmarshal([]byte(frontmatter), &info); err != nil {
			t.Fatalf("parse frontmatter %s: %v", entry.Name(), err)
		}
		result[name] = info
	}
	return result
}

func TestYAMLWrapperContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	yamlContracts := parseYAMLContracts(t, repoRoot)
	claudeContracts := parseWrapperContracts(t, repoRoot, filepath.Join(".claude", "commands", "ant"))
	opencodeContracts := parseWrapperContracts(t, repoRoot, filepath.Join(".opencode", "commands", "ant"))

	var violations []string

	// Verify YAML contracts.
	for name, info := range yamlContracts {
		// Prompt-only commands have no runtime command by design.
		if !isExcludedFromRuntime(name) && info.Runtime.Command == "" {
			violations = append(violations, fmt.Sprintf("YAML %q: missing runtime.command", name))
		}
		if len(info.Guardrails) == 0 {
			violations = append(violations, fmt.Sprintf("YAML %q: missing guardrails", name))
		}
	}

	// Verify wrapper contracts and cross-match with YAML.
	for name, info := range claudeContracts {
		if info.Name == "" {
			violations = append(violations, fmt.Sprintf("Claude wrapper %q: missing name", name))
		}
		if info.Description == "" {
			violations = append(violations, fmt.Sprintf("Claude wrapper %q: missing description", name))
		}
		yamlInfo, ok := yamlContracts[name]
		if !ok {
			violations = append(violations, fmt.Sprintf("Claude wrapper %q: no matching YAML source", name))
			continue
		}
		if info.Name != yamlInfo.Name {
			violations = append(violations, fmt.Sprintf("Claude wrapper %q: name %q != YAML name %q", name, info.Name, yamlInfo.Name))
		}
		if info.Description != yamlInfo.Description {
			violations = append(violations, fmt.Sprintf("Claude wrapper %q: description mismatch", name))
		}
	}

	for name, info := range opencodeContracts {
		if info.Name == "" {
			violations = append(violations, fmt.Sprintf("OpenCode wrapper %q: missing name", name))
		}
		if info.Description == "" {
			violations = append(violations, fmt.Sprintf("OpenCode wrapper %q: missing description", name))
		}
		yamlInfo, ok := yamlContracts[name]
		if !ok {
			violations = append(violations, fmt.Sprintf("OpenCode wrapper %q: no matching YAML source", name))
			continue
		}
		if info.Name != yamlInfo.Name {
			violations = append(violations, fmt.Sprintf("OpenCode wrapper %q: name %q != YAML name %q", name, info.Name, yamlInfo.Name))
		}
		if info.Description != yamlInfo.Description {
			violations = append(violations, fmt.Sprintf("OpenCode wrapper %q: description mismatch", name))
		}
	}

	if len(violations) > 0 {
		t.Fatalf("YAML/wrapper contract violations:\n%s", strings.Join(violations, "\n"))
	}
}

// --- Codex TOML Coverage Documentation ---

// TestCodexCoverageByDesign documents that Codex TOML agents represent worker
// castes, not command wrappers. All 60 commands are covered by
// commandGuideCatalog(). This is accepted tech debt -- see KNOWN-GAPS.md I-01.
func TestCodexCoverageByDesign(t *testing.T) {
	guide := commandGuideCatalog()
	if len(guide) == 0 {
		t.Fatal("commandGuideCatalog() returned empty catalog")
	}
	// The guide should cover at least the 60 YAML commands.
	// We verify it's non-empty rather than asserting exactly 60, because the
	// guide may contain intelligent entries beyond literal YAML mappings.
	t.Logf("commandGuideCatalog() covers %d entries", len(guide))
}

// --- Phase 105-03 Review Findings ---

// TestNoOrphanRuntimeCommands checks that every runtime command which is the
// target of an alias in yamlToRuntimeName has a corresponding YAML source.
// This verifies the reverse direction of the alias map without requiring every
// runtime command to have a YAML wrapper (the runtime has 300+ commands, most
// are internal subcommands intentionally not surfaced).
func TestNoOrphanRuntimeCommands(t *testing.T) {
	yamlNames := extractYAMLNames(t)
	yamlSet := make(map[string]bool, len(yamlNames))
	for _, name := range yamlNames {
		yamlSet[name] = true
	}

	runtimeNames := extractRuntimeNames()
	runtimeSet := make(map[string]bool, len(runtimeNames))
	for _, name := range runtimeNames {
		runtimeSet[name] = true
	}

	// Build reverse alias map and verify each aliased runtime command exists
	// and maps back to a valid YAML source.
	var orphans []string
	for yamlName, runtimeName := range yamlToRuntimeName {
		if !runtimeSet[runtimeName] {
			orphans = append(orphans, fmt.Sprintf("alias target %q (from YAML %q) not in runtime", runtimeName, yamlName))
			continue
		}
		if !yamlSet[yamlName] {
			orphans = append(orphans, fmt.Sprintf("YAML source %q for alias target %q not found", yamlName, runtimeName))
		}
	}

	if len(orphans) > 0 {
		t.Errorf("orphan runtime commands detected:\n%s", strings.Join(orphans, "\n"))
	}
}

// TestPromptOnlyCommandsAreCurrent validates that promptOnlyCommands and
// cobraBuiltinCommands exclusion lists contain only actual YAML commands.
func TestPromptOnlyCommandsAreCurrent(t *testing.T) {
	yamlNames := extractYAMLNames(t)
	yamlSet := make(map[string]bool, len(yamlNames))
	for _, name := range yamlNames {
		yamlSet[name] = true
	}

	for name := range promptOnlyCommands {
		if !yamlSet[name] {
			t.Errorf("promptOnlyCommands entry %q not found in YAML catalog (stale entry?)", name)
		}
	}

	for name := range cobraBuiltinCommands {
		if !yamlSet[name] {
			t.Errorf("cobraBuiltinCommands entry %q not found in YAML catalog (stale entry?)", name)
		}
	}
}

// TestAliasMapCompleteness checks that every YAML command (except prompt-only
// and builtins) either exists directly in the runtime or has an alias entry.
func TestAliasMapCompleteness(t *testing.T) {
	yamlNames := extractYAMLNames(t)
	runtimeNames := extractRuntimeNames()
	runtimeSet := make(map[string]bool, len(runtimeNames))
	for _, name := range runtimeNames {
		runtimeSet[name] = true
	}

	var missing []string
	for _, yamlName := range yamlNames {
		// Skip prompt-only and builtin commands.
		if promptOnlyCommands[yamlName] || cobraBuiltinCommands[yamlName] {
			continue
		}

		// Check if YAML name exists in runtime directly.
		if runtimeSet[yamlName] {
			continue
		}

		// Check if YAML name has an alias.
		if _, hasAlias := yamlToRuntimeName[yamlName]; hasAlias {
			continue
		}

		missing = append(missing, yamlName)
	}

	if len(missing) > 0 {
		t.Errorf("%d YAML commands have no runtime mapping (missing alias?): %v", len(missing), missing)
	}
}
