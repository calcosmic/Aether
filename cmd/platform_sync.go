package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

type installSyncPair struct {
	srcRel               string
	destRel              string
	label                string
	cleanup              bool
	preserveLocalChanges bool
	validate             syncValidator
	include              syncFilter
	mapRelPath           syncRelPathMapper
	cleanupInclude       syncFilter
	cleanupLegacyClaude  bool
}

type repoSyncPair struct {
	hubRel               string
	destRel              string
	label                string
	cleanup              bool
	preserveLocalChanges bool
	validate             syncValidator
	include              syncFilter
	mapRelPath           syncRelPathMapper
	cleanupInclude       syncFilter
	cleanupLegacyClaude  bool
	consumerOnly         bool
}

type syncValidator func(srcPath, relPath string, data []byte) error
type syncFilter func(relPath string) bool
type syncRelPathMapper func(relPath string) string

type codexAgentDefinition struct {
	Name                  string   `toml:"name"`
	Description           string   `toml:"description"`
	NicknameCandidates    []string `toml:"nickname_candidates"`
	DeveloperInstructions string   `toml:"developer_instructions"`
}

func installSyncPairs() []installSyncPair {
	return []installSyncPair{
		{srcRel: ".claude/commands/ant", destRel: ".claude/commands", label: "Commands (claude)", cleanup: true, mapRelPath: claudeCommandDestRelPath, cleanupInclude: isManagedFlatClaudeCommandPath, cleanupLegacyClaude: true},
		{srcRel: ".claude/agents/ant", destRel: ".claude/agents/ant", label: "Agents (claude)", cleanup: true},
		{srcRel: ".opencode/commands/ant", destRel: ".config/opencode/commands/ant", label: "Commands (opencode)", cleanup: true},
		{srcRel: ".opencode/agents", destRel: ".config/opencode/agents", label: "Agents (opencode)", cleanup: false, validate: validateOpenCodeAgentFile},
		{srcRel: ".codex/agents", destRel: ".codex/agents", label: "Agents (codex)", cleanup: false, preserveLocalChanges: true, validate: validateCodexAgentFile, include: isShippedAetherCodexAgent},
	}
}

func repoSyncPairs() []repoSyncPair {
	return []repoSyncPair{
		{
			hubRel:         ".",
			destRel:        ".",
			label:          "Repo .aether cleanup",
			cleanup:        true,
			include:        neverSyncPath,
			cleanupInclude: isManagedAetherSystemPath,
			consumerOnly:   true,
		},
		{hubRel: "settings/claude", destRel: "../.claude", label: "Settings (claude)", preserveLocalChanges: true, include: isClaudeSettingsFile},
		{hubRel: "rules", destRel: "../.claude/rules", label: "Rules (claude)"},
	}
}

type codexSkillShim struct {
	Dir         string
	Name        string
	Description string
	Body        string
}

func codexSkillShims() []codexSkillShim {
	return []codexSkillShim{
		{
			Dir:         "aether-command-guide",
			Name:        "aether-command-guide",
			Description: "Use for Aether lifecycle commands; ask the runtime for current orchestration guidance before acting.",
			Body:        "Run `aether command-guide <command> --platform codex` before intelligent Aether flows. Follow the guide over stale local notes. For raw user commands, run the literal command.",
		},
		{
			Dir:         "aether-skill-loader",
			Name:        "aether-skill-loader",
			Description: "Use when Aether worker context needs skills; load matched skill content from the runtime on demand.",
			Body:        "Run `aether skill-inject --workflow <workflow> --role <role> --task \"<task>\"` to fetch the relevant shipped and custom Aether skills. Do not preload full skill mirrors.",
		},
		{
			Dir:         "aether-colony-creation",
			Name:        "aether-colony-creation",
			Description: "Use when initializing an Aether colony in Codex; refine intent before calling the runtime.",
			Body:        "For `aether init` or setup requests, use `aether command-guide init --platform codex`, ask compact clarifying questions when needed, synthesize a precise charter, then let the runtime create state.",
		},
		{
			Dir:         "aether-colony-research",
			Name:        "aether-colony-research",
			Description: "Use when running Oracle or discuss flows in Codex; scope research before persistence begins.",
			Body:        "For `aether oracle` or `aether discuss`, use `aether command-guide <oracle|discuss> --platform codex`, clarify output shape, scope, depth, and confidence, then run the runtime flow.",
		},
	}
}

func syncCodexSkillShims(destDir string) syncResult {
	result := syncResult{}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", destDir, err))
		return result
	}

	allowed := map[string]bool{}
	for _, shim := range codexSkillShims() {
		allowed[filepath.ToSlash(shim.Dir)] = true
	}

	for _, dir := range findSkillDirs(destDir) {
		rel, err := filepath.Rel(destDir, dir)
		if err != nil {
			result.errors = append(result.errors, fmt.Sprintf("rel %s: %v", dir, err))
			continue
		}
		rel = filepath.ToSlash(rel)
		if allowed[rel] {
			continue
		}
		if skillDirDeclaresSource(dir, "custom") {
			result.skipped++
			continue
		}
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", dir, err))
			continue
		}
		result.removed = append(result.removed, rel)
	}

	for _, shim := range codexSkillShims() {
		skillPath := filepath.Join(destDir, filepath.FromSlash(shim.Dir), "SKILL.md")
		content := renderCodexSkillShim(shim)
		if current, err := os.ReadFile(skillPath); err == nil && string(current) == content {
			result.skipped++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(skillPath), 0755); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", filepath.Dir(skillPath), err))
			continue
		}
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("write %s: %v", skillPath, err))
			continue
		}
		result.copied++
	}

	cleanEmptyDirs(destDir)
	return result
}

func renderCodexSkillShim(shim codexSkillShim) string {
	return fmt.Sprintf(`---
name: %s
description: %s
source: shipped
type: codex-shim
domains: [aether, codex, orchestration]
priority: high
version: "1.0"
---

# %s

%s
`, shim.Name, shim.Description, shim.Name, shim.Body)
}

func skillDirDeclaresSource(dir, expected string) bool {
	raw, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return false
	}
	fm := parseSkillFrontmatter(string(raw))
	return fm != nil && strings.TrimSpace(fm.Source) == expected
}

func neverSyncPath(string) bool {
	return false
}

var managedAetherSystemDirs = map[string]bool{
	"agents":        true,
	"agents-claude": true,
	"agents-codex":  true,
	"codex":         true,
	"commands":      true,
	"docs":          true,
	"exchange":      true,
	"references":    true,
	"rules":         true,
	"schemas":       true,
	"settings":      true,
	"skills-codex":  true,
	"templates":     true,
	"ts":            true,
	"utils":         true,
}

var managedAetherSystemFiles = map[string]bool{
	".npmignore":          true,
	"aether-utils.sh":     true,
	"ledger.jsonl":        true,
	"manifest.json":       true,
	"model-profiles.yaml": true,
	"registry.json":       true,
	"version.json":        true,
	"workers.md":          true,
}

func isManagedAetherSystemPath(relPath string) bool {
	clean := filepath.ToSlash(filepath.Clean(relPath))
	if clean == "." || clean == "" {
		return false
	}
	first := clean
	if idx := strings.Index(clean, "/"); idx >= 0 {
		first = clean[:idx]
	}
	if managedAetherSystemDirs[first] {
		return true
	}
	if strings.Contains(clean, "/") {
		return false
	}
	return managedAetherSystemFiles[clean]
}

func isShippedAetherCodexAgent(relPath string) bool {
	base := filepath.Base(relPath)
	return filepath.Ext(base) == ".toml" && strings.HasPrefix(base, "aether-")
}

func isClaudeSettingsFile(relPath string) bool {
	return filepath.Base(relPath) == "settings.json"
}

func claudeCommandDestRelPath(relPath string) string {
	base := filepath.Base(filepath.Clean(relPath))
	if filepath.Ext(base) != ".md" {
		return relPath
	}
	if strings.HasPrefix(base, "ant-") {
		return base
	}
	return "ant-" + base
}

func isManagedFlatClaudeCommandPath(relPath string) bool {
	clean := filepath.ToSlash(filepath.Clean(relPath))
	if strings.Contains(clean, "/") {
		return false
	}
	base := filepath.Base(clean)
	return strings.HasPrefix(base, "ant-") && filepath.Ext(base) == ".md"
}

func isGeneratedAetherCommandWrapper(data []byte) bool {
	firstLine := strings.SplitN(string(data), "\n", 2)[0]
	return strings.HasPrefix(firstLine, "<!-- Generated from .aether/commands/") &&
		strings.HasSuffix(firstLine, ".yaml - DO NOT EDIT DIRECTLY -->")
}

func removeLegacyClaudeCommandNamespace(commandsDir string) ([]string, []string) {
	legacyDir := filepath.Join(commandsDir, "ant")
	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{fmt.Sprintf("read legacy Claude commands %s: %v", legacyDir, err)}
	}

	var removed []string
	var errs []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		path := filepath.Join(legacyDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("read legacy Claude command %s: %v", path, err))
			continue
		}
		if !isGeneratedAetherCommandWrapper(data) {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("remove legacy Claude command %s: %v", path, err))
			continue
		}
		removed = append(removed, filepath.Join("ant", entry.Name()))
	}

	if len(removed) > 0 {
		if err := os.Remove(legacyDir); err != nil && !os.IsNotExist(err) {
			if entries, readErr := os.ReadDir(legacyDir); readErr == nil && len(entries) > 0 {
				return removed, errs
			}
			errs = append(errs, fmt.Sprintf("remove legacy Claude command namespace %s: %v", legacyDir, err))
		}
	}

	return removed, errs
}

func appendSyncResult(details *[]map[string]interface{}, totals *updateSyncResult, label string, result syncResult) {
	entry := map[string]interface{}{
		"label":   label,
		"copied":  result.copied,
		"skipped": result.skipped,
		"removed": len(result.removed),
	}
	if len(result.errors) > 0 {
		entry["errors"] = result.errors
		totals.errors = append(totals.errors, result.errors...)
	}
	*details = append(*details, entry)
	totals.copied += result.copied
	totals.skipped += result.skipped
}

func pruneLegacyRepoPlatformAssets(repoDir string) syncResult {
	result := syncResult{}
	if isAetherSourceCheckout(repoDir) {
		return result
	}

	pruners := []struct {
		label string
		fn    func() syncResult
	}{
		{
			label: "claude commands",
			fn: func() syncResult {
				return pruneGeneratedCommandFiles(filepath.Join(repoDir, ".claude", "commands"))
			},
		},
		{
			label: "opencode commands",
			fn: func() syncResult {
				return pruneGeneratedCommandFiles(filepath.Join(repoDir, ".opencode", "commands", "ant"))
			},
		},
		{
			label: "claude agents",
			fn: func() syncResult {
				return pruneAetherNamedFiles(filepath.Join(repoDir, ".claude", "agents", "ant"), ".md")
			},
		},
		{
			label: "opencode agents",
			fn: func() syncResult {
				return pruneAetherNamedFiles(filepath.Join(repoDir, ".opencode", "agents"), ".md")
			},
		},
		{
			label: "codex agents",
			fn: func() syncResult {
				return pruneAetherNamedFiles(filepath.Join(repoDir, ".codex", "agents"), ".toml")
			},
		},
		{
			label: "codex skills",
			fn: func() syncResult {
				return pruneDirectoryTree(filepath.Join(repoDir, ".codex", "skills", "aether"))
			},
		},
	}

	for _, pruner := range pruners {
		pruned := pruner.fn()
		for _, removed := range pruned.removed {
			result.removed = append(result.removed, filepath.ToSlash(filepath.Join(pruner.label, removed)))
		}
		result.errors = append(result.errors, pruned.errors...)
	}
	return result
}

func pruneGeneratedCommandFiles(dir string) syncResult {
	result := syncResult{}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", dir, err))
		return result
	}
	if !info.IsDir() {
		return result
	}

	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			result.errors = append(result.errors, fmt.Sprintf("read %s: %v", path, readErr))
			return nil
		}
		if !isGeneratedAetherCommandWrapper(data) {
			return nil
		}
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", path, removeErr))
			return nil
		}
		if rel, relErr := filepath.Rel(dir, path); relErr == nil {
			result.removed = append(result.removed, filepath.ToSlash(rel))
		}
		return nil
	})
	if len(result.removed) > 0 {
		cleanEmptyDirs(dir)
	}
	return result
}

func pruneAetherNamedFiles(dir string, extensions ...string) syncResult {
	result := syncResult{}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", dir, err))
		return result
	}
	if !info.IsDir() {
		return result
	}

	allowed := map[string]bool{}
	for _, ext := range extensions {
		allowed[ext] = true
	}

	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if !strings.HasPrefix(base, "aether-") {
			return nil
		}
		if len(allowed) > 0 && !allowed[filepath.Ext(base)] {
			return nil
		}
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", path, removeErr))
			return nil
		}
		if rel, relErr := filepath.Rel(dir, path); relErr == nil {
			result.removed = append(result.removed, filepath.ToSlash(rel))
		}
		return nil
	})
	if len(result.removed) > 0 {
		cleanEmptyDirs(dir)
	}
	return result
}

func pruneDirectoryTree(dir string) syncResult {
	result := syncResult{}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", dir, err))
		return result
	}
	if !info.IsDir() {
		return result
	}
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", dir, err))
		return result
	}
	result.removed = append(result.removed, filepath.Base(dir))
	return result
}

func pruneShippedRepoSkills(hubSystem, localAether string, force bool) syncResult {
	result := syncResult{}
	if isAetherSourceCheckout(filepath.Dir(localAether)) {
		return result
	}

	hubSkills := filepath.Join(hubSystem, "skills")
	localSkills := filepath.Join(localAether, "skills")
	info, err := os.Stat(localSkills)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", localSkills, err))
		return result
	}
	if !info.IsDir() {
		return result
	}

	for _, rel := range listFilesRecursive(localSkills) {
		if syncPathIgnored(rel) {
			continue
		}
		localPath := filepath.Join(localSkills, rel)
		hubPath := filepath.Join(hubSkills, rel)
		if _, err := os.Stat(hubPath); err != nil {
			if !os.IsNotExist(err) {
				result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", hubPath, err))
			}
			continue
		}
		remove := force
		if !remove {
			localHash, localErr := fileSHA256(localPath)
			hubHash, hubErr := fileSHA256(hubPath)
			remove = localErr == nil && hubErr == nil && localHash == hubHash
		}
		if !remove {
			result.skipped++
			continue
		}
		if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", localPath, err))
			continue
		}
		result.removed = append(result.removed, filepath.ToSlash(rel))
	}
	if len(result.removed) > 0 {
		cleanEmptyDirs(localSkills)
	}
	return result
}

func pruneShippedFromUserSkillsDir(hubSystem, hubDir string) syncResult {
	result := syncResult{}
	shippedSkills := filepath.Join(hubSystem, "skills")
	userSkills := filepath.Join(hubDir, "skills")
	info, err := os.Stat(userSkills)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", userSkills, err))
		return result
	}
	if !info.IsDir() {
		return result
	}

	for _, rel := range listFilesRecursive(userSkills) {
		if syncPathIgnored(rel) {
			continue
		}
		userPath := filepath.Join(userSkills, rel)
		shippedPath := filepath.Join(shippedSkills, rel)
		if _, err := os.Stat(shippedPath); err != nil {
			if !os.IsNotExist(err) {
				result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", shippedPath, err))
			}
			continue
		}
		userHash, userErr := fileSHA256(userPath)
		shippedHash, shippedErr := fileSHA256(shippedPath)
		if userErr != nil || shippedErr != nil || userHash != shippedHash {
			result.skipped++
			continue
		}
		if err := os.Remove(userPath); err != nil && !os.IsNotExist(err) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", userPath, err))
			continue
		}
		result.removed = append(result.removed, filepath.ToSlash(rel))
	}
	if len(result.removed) > 0 {
		cleanEmptyDirs(userSkills)
	}
	return result
}

func pruneRepoCodexSkillMirror(repoDir string, force bool) syncResult {
	result := syncResult{}
	if !force || isAetherSourceCheckout(repoDir) {
		return result
	}
	root := filepath.Join(repoDir, ".codex", "skills", "aether")
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", root, err))
		return result
	}
	if !info.IsDir() {
		return result
	}

	for _, dir := range findSkillDirs(root) {
		if skillDirDeclaresSource(dir, "custom") {
			result.skipped++
			continue
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			result.errors = append(result.errors, fmt.Sprintf("rel %s: %v", dir, err))
			continue
		}
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", dir, err))
			continue
		}
		result.removed = append(result.removed, filepath.ToSlash(rel))
	}
	if len(result.removed) > 0 {
		cleanEmptyDirs(root)
	}
	return result
}

func ensureRepoLocalScaffold(localAether string) syncResult {
	result := syncResult{}
	for _, dir := range []string{"data", "dreams", "oracle", "checkpoints", "locks"} {
		path := filepath.Join(localAether, dir)
		if _, err := os.Stat(path); err == nil {
			result.skipped++
			continue
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", path, err))
			continue
		}
		result.copied++
	}

	gitignorePath := filepath.Join(localAether, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		content := "# Aether local state - not versioned\ndata/\ncheckpoints/\nlocks/\ndreams/\noracle/\n"
		if writeErr := os.WriteFile(gitignorePath, []byte(content), 0644); writeErr != nil {
			result.errors = append(result.errors, fmt.Sprintf("write %s: %v", gitignorePath, writeErr))
		} else {
			result.copied++
		}
	} else if err == nil {
		result.skipped++
	} else {
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", gitignorePath, err))
	}

	queenPath := filepath.Join(localAether, "QUEEN.md")
	if _, err := os.Stat(queenPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(queenPath), 0755); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", filepath.Dir(queenPath), err))
		} else if err := os.WriteFile(queenPath, []byte(queenDefaultContent), 0644); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("write %s: %v", queenPath, err))
		} else {
			result.copied++
		}
	} else if err == nil {
		result.skipped++
	} else {
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", queenPath, err))
	}

	return result
}

func validateCodexAgentFile(srcPath, relPath string, data []byte) error {
	if filepath.Ext(relPath) != ".toml" {
		return fmt.Errorf("%s must use the .toml extension", relPath)
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("%s is not valid UTF-8 text", relPath)
	}

	var agent codexAgentDefinition
	if err := toml.Unmarshal(data, &agent); err != nil {
		return fmt.Errorf("%s is not valid TOML: %w", relPath, err)
	}

	baseName := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	switch {
	case strings.TrimSpace(agent.Name) == "":
		return fmt.Errorf("%s is missing name", relPath)
	case agent.Name != baseName:
		return fmt.Errorf("%s name %q does not match filename %q", relPath, agent.Name, baseName)
	case strings.TrimSpace(agent.Description) == "":
		return fmt.Errorf("%s is missing description", relPath)
	case len(agent.NicknameCandidates) < 2:
		return fmt.Errorf("%s must define at least 2 nickname_candidates", relPath)
	case strings.TrimSpace(agent.DeveloperInstructions) == "":
		return fmt.Errorf("%s is missing developer_instructions", relPath)
	}

	// Reject binary-like content masquerading as text by ensuring the source can
	// be read back as a regular file. This keeps the validator conservative while
	// still allowing normal multiline TOML strings.
	if info, err := os.Stat(srcPath); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", relPath)
	}

	return nil
}

// openCodeAgentFrontmatter defines the expected YAML fields for an OpenCode
// agent file. The `name` field is required — it identifies the agent to the
// OpenCode runtime.
type openCodeAgentFrontmatter struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Mode        string                 `yaml:"mode"`
	Tools       map[string]interface{} `yaml:"tools"`
	Color       string                 `yaml:"color"`
	Model       string                 `yaml:"model"`
}

var openCodeThemeColors = map[string]bool{
	"primary": true, "secondary": true, "accent": true,
	"success": true, "warning": true, "error": true, "info": true,
}

var openCodeHexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// validateOpenCodeAgentFile validates an OpenCode agent markdown file.
// It checks that the YAML frontmatter conforms to the OpenCode agent schema:
// name (required), description (20+ chars), tools (object/map), color (hex or theme),
// and model (provider/model-id format).
func validateOpenCodeAgentFile(srcPath, relPath string, data []byte) error {
	// Rule 1: must have .md extension
	if filepath.Ext(relPath) != ".md" {
		return fmt.Errorf("%s must use the .md extension", relPath)
	}

	// Rule 2: must be valid UTF-8
	if !utf8.Valid(data) {
		return fmt.Errorf("%s is not valid UTF-8 text", relPath)
	}

	// Rule 3: must have YAML frontmatter between --- delimiters
	content := string(data)
	start := strings.Index(content, "---")
	if start == -1 {
		return fmt.Errorf("%s is missing YAML frontmatter (no opening ---)", relPath)
	}
	end := strings.Index(content[start+3:], "---")
	if end == -1 {
		return fmt.Errorf("%s is missing YAML frontmatter (no closing ---)", relPath)
	}
	yamlContent := content[start+3 : start+3+end]

	var fm openCodeAgentFrontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return fmt.Errorf("%s has invalid YAML frontmatter: %w", relPath, err)
	}

	// Rule 4: description must be present and at least 20 characters
	desc := strings.TrimSpace(fm.Description)
	if desc == "" {
		return fmt.Errorf("%s is missing description in frontmatter", relPath)
	}
	if len(desc) < 20 {
		return fmt.Errorf("%s description too short (%d chars, need at least 20): %q", relPath, len(desc), desc)
	}

	// Rule 5: mode must be a valid value
	mode := strings.TrimSpace(fm.Mode)
	if mode == "" {
		return fmt.Errorf("%s is missing mode in frontmatter", relPath)
	}
	if mode != "primary" && mode != "subagent" && mode != "all" {
		return fmt.Errorf("%s mode %q must be primary, subagent, or all", relPath, mode)
	}

	// Rule 6: tools must be a map/object (not a string, not nil)
	if fm.Tools == nil {
		return fmt.Errorf("%s is missing tools field in frontmatter", relPath)
	}
	// Also check the raw YAML to detect tools as a string (yaml.Unmarshal
	// would not error on that but would produce nil map). Re-parse the raw
	// frontmatter to check the actual type of tools.
	var rawFM map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &rawFM); err != nil {
		return fmt.Errorf("%s has invalid YAML: %w", relPath, err)
	}
	rawTools := rawFM["tools"]
	if rawTools == nil {
		return fmt.Errorf("%s is missing tools field in frontmatter", relPath)
	}
	if _, ok := rawTools.(map[string]interface{}); !ok {
		if _, isStr := rawTools.(string); isStr {
			return fmt.Errorf("%s tools must be a map/object with true/false values, not a string", relPath)
		}
		return fmt.Errorf("%s tools has unexpected type %T (must be a map/object)", relPath, rawTools)
	}

	// Rule 7: color must be a hex color or a theme color name
	color := strings.TrimSpace(fm.Color)
	if color == "" {
		return fmt.Errorf("%s is missing color in frontmatter", relPath)
	}
	if !openCodeHexColorRe.MatchString(color) && !openCodeThemeColors[color] {
		return fmt.Errorf("%s color %q must be a hex color (#rrggbb) or a theme color (primary, secondary, accent, success, warning, error, info)", relPath, color)
	}

	// Rule 8: name field is required
	if strings.TrimSpace(fm.Name) == "" {
		return fmt.Errorf("%s is missing name in frontmatter", relPath)
	}

	// Rule 9: model is optional — when absent, OpenCode uses its global default

	// Reject binary-like content masquerading as text
	if info, err := os.Stat(srcPath); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", relPath)
	}

	return nil
}
