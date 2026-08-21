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
	merge                syncMerger
}

type syncValidator func(srcPath, relPath string, data []byte) error
type syncFilter func(relPath string) bool
type syncRelPathMapper func(relPath string) string

// syncMerger combines shipped template bytes with existing destination bytes.
// Returning the destination bytes unchanged marks the file as up to date.
type syncMerger func(templateData, existingData []byte) ([]byte, error)

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
		{srcRel: ".opencode/commands/ant", destRel: ".opencode/command", label: "Commands (opencode home)", cleanup: false},
		{srcRel: ".opencode/agents", destRel: ".opencode/agent", label: "Agents (opencode home)", cleanup: false, validate: validateOpenCodeAgentFile},
		{srcRel: ".opencode/commands/ant", destRel: ".config/opencode/commands/ant", label: "Commands (opencode)", cleanup: true},
		{srcRel: ".opencode/agents", destRel: ".config/opencode/agents", label: "Agents (opencode)", cleanup: false, validate: validateOpenCodeAgentFile},
		{srcRel: ".codex/agents", destRel: ".codex/agents", label: "Agents (codex)", cleanup: false, preserveLocalChanges: true, validate: validateCodexAgentFile, include: isShippedAetherCodexAgent},
	}
}

func platformHomeHubSyncPairs() []installSyncPair {
	return []installSyncPair{
		{srcRel: "commands/claude", destRel: ".claude/commands", label: "Commands (claude)", cleanup: true, mapRelPath: claudeCommandDestRelPath, cleanupInclude: isManagedFlatClaudeCommandPath, cleanupLegacyClaude: true},
		{srcRel: "agents-claude", destRel: ".claude/agents/ant", label: "Agents (claude)", cleanup: true},
		{srcRel: "commands/opencode", destRel: ".opencode/command", label: "Commands (opencode home)", cleanup: false},
		{srcRel: "agents", destRel: ".opencode/agent", label: "Agents (opencode home)", cleanup: false, validate: validateOpenCodeAgentFile},
		{srcRel: "commands/opencode", destRel: ".config/opencode/commands/ant", label: "Commands (opencode)", cleanup: true},
		{srcRel: "agents", destRel: ".config/opencode/agents", label: "Agents (opencode)", cleanup: false, validate: validateOpenCodeAgentFile},
		{srcRel: "codex", destRel: ".codex/agents", label: "Agents (codex)", cleanup: false, preserveLocalChanges: true, validate: validateCodexAgentFile, include: isShippedAetherCodexAgent},
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
		{hubRel: "settings/claude", destRel: "../.claude", label: "Settings (claude)", preserveLocalChanges: true, include: isClaudeSettingsFile, merge: mergeClaudeSettings},
		{hubRel: "rules", destRel: "../.claude/rules", label: "Rules (claude)"},
	}
}

type codexSkillShim struct {
	Dir              string
	Name             string
	Description      string
	Body             string
	WorkflowTriggers []string
	TaskKeywords     []string
}

func codexSkillShims() []codexSkillShim {
	shims := []codexSkillShim{
		{
			Dir:         "aether-command-guide",
			Name:        "aether-command-guide",
			Description: "Use for Aether lifecycle commands; ask the runtime for current orchestration guidance before acting.",
			Body:        "Run `aether command-guide <command> --platform codex` before intelligent Aether flows. Follow the guide over stale local notes. For raw user commands, run the literal command.",
		},
		{
			Dir:         "aether-skill-loader",
			Name:        "aether-skill-loader",
			Description: "Explains where Aether worker skill content comes from -- no on-demand loader command exists.",
			Body:        "Skill content is already included automatically in the worker brief text returned by `aether build`, `aether colonize`, `aether plan`, and `aether continue` -- it is assembled in-process from the matched shipped and custom Aether skills. There is no separate command to fetch it on demand (skill-inject, the CLI command this shim used to call, was deleted in Phase 191 as dead CLI surface -- its underlying matching logic is what dispatches use automatically). Do not preload full skill mirrors.",
		},
		{
			Dir:         "aether-colony-creation",
			Name:        "aether-colony-creation",
			Description: "Use when initializing an Aether colony in Codex; refine intent before calling the runtime.",
			Body:        "For `aether init` or setup requests, use `aether command-guide init --platform codex`, ask compact clarifying questions when needed, ask the user to choose Colony Mode or Orchestrator Mode, synthesize a precise charter, then run the runtime with `--colony-mode <selected>` so it creates state.",
		},
		{
			Dir:         "aether-colony-research",
			Name:        "aether-colony-research",
			Description: "Use when running Oracle or discuss flows in Codex; scope research before persistence begins.",
			Body:        "For `aether oracle` or `aether discuss`, use `aether command-guide <oracle|discuss> --platform codex`, clarify output shape, scope, depth, and confidence, then run the runtime flow.",
		},
		{
			Dir:              "aether-colony-build-cycle",
			Name:             "aether-colony-build-cycle",
			Description:      "Use when Codex is asked to colonize, plan, build, continue, swarm, or seal an Aether colony and must mirror wrapper orchestration safely.",
			Body:             "For `aether colonize`, `aether plan`, `aether build`, `aether continue`, `aether swarm`, or `aether seal`, run `aether command-guide <command> --platform codex`, use runtime JSON manifests and finalizers, pass worker briefs verbatim, honor loop guards, and never hand-edit `.aether/data`.",
			WorkflowTriggers: []string{"colonize", "plan", "build", "continue", "swarm", "seal"},
			TaskKeywords:     []string{"aether colonize", "aether plan", "aether build", "aether continue", "aether swarm", "aether seal", "dispatch manifest", "plan-only", "finalize"},
		},
	}
	return append(shims, codexCommandSkillShims()...)
}

func codexCommandSkillShims() []codexSkillShim {
	commands := []string{"init", "discuss", "oracle", "colonize", "plan", "build", "continue", "swarm", "seal"}
	catalog := commandGuideCatalog()
	shims := make([]codexSkillShim, 0, len(commands))
	for _, command := range commands {
		def, ok := catalog[command]
		if !ok || def.Literal {
			continue
		}
		keywords := []string{
			"aether " + command,
			"/ant-" + command,
			"ant-" + command,
			"command-guide " + command,
			"aether command-guide " + command,
		}
		if def.SkillReference != "" {
			keywords = append(keywords, def.SkillReference)
		}
		shims = append(shims, codexSkillShim{
			Dir:              "aether-" + command,
			Name:             "aether-" + command,
			Description:      fmt.Sprintf("Use when Codex is asked to run `aether %s` or the equivalent Aether lifecycle action.", command),
			Body:             renderCodexCommandSkillShimBody(command, def),
			WorkflowTriggers: []string{command},
			TaskKeywords:     keywords,
		})
	}
	return shims
}

func renderCodexCommandSkillShimBody(command string, def commandGuideDefinition) string {
	var b strings.Builder
	fmt.Fprintf(&b, "This is the Codex command-shaped skill for `aether %s`. Use it instead of relying on free-form natural language for this lifecycle action.\n\n", command)
	fmt.Fprintf(&b, "1. Run `aether command-guide %s --platform codex` first and treat that runtime guide as authoritative.\n", command)
	if def.SkillReference != "" {
		fmt.Fprintf(&b, "2. Load or follow `%s`; this command-specific skill is the entrypoint, not a replacement for the shared lifecycle skill.\n", def.SkillReference)
	} else {
		b.WriteString("2. Follow the runtime guide directly.\n")
	}
	b.WriteString("3. Preserve runtime ownership of state: wrappers and skills may interview, synthesize, spawn workers, and summarize, but must not hand-edit `.aether/data`.\n")
	b.WriteString("4. Honor raw/exact/no-orchestration requests by using the raw bypass below.\n\n")

	if def.Intent != "" {
		fmt.Fprintf(&b, "## Intent\n%s\n\n", def.Intent)
	}
	writeCodexCommandSkillList(&b, "Pre-steps", def.PreSteps)
	if def.RunCommand != "" {
		fmt.Fprintf(&b, "## Runtime Command\n`%s`\n\n", def.RunCommand)
	}
	writeCodexCommandSkillList(&b, "Post-steps", def.PostSteps)
	writeCodexCommandSkillList(&b, "Drift Guards", def.DriftGuards)
	if def.RawBypass != "" {
		fmt.Fprintf(&b, "## Raw Bypass\n%s\n", def.RawBypass)
	}
	return strings.TrimSpace(b.String())
}

func writeCodexCommandSkillList(b *strings.Builder, heading string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(b, "## %s\n", heading)
	for _, value := range values {
		fmt.Fprintf(b, "- %s\n", value)
	}
	b.WriteString("\n")
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
	extraFrontmatter := ""
	if len(shim.WorkflowTriggers) > 0 {
		extraFrontmatter += fmt.Sprintf("workflow_triggers: [%s]\n", strings.Join(shim.WorkflowTriggers, ", "))
	}
	if len(shim.TaskKeywords) > 0 {
		extraFrontmatter += fmt.Sprintf("task_keywords: [%s]\n", strings.Join(shim.TaskKeywords, ", "))
	}
	return fmt.Sprintf(`---
name: %s
description: %s
source: shipped
type: codex-shim
domains: [aether, codex, orchestration]
%spriority: high
version: "1.0"
---

# %s

%s
`, shim.Name, shim.Description, extraFrontmatter, shim.Name, shim.Body)
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

func isOraclePhaseDirectivesFile(relPath string) bool {
	return filepath.Base(relPath) == "oracle-phase-directives.yaml"
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

// isGeneratedAetherCommandWrapper marks a file as Aether-managed for
// update/prune. It accepts both the current header and the legacy
// "Generated from" form so downstream repos installed before the header
// reform still get their stale wrappers pruned.
func isGeneratedAetherCommandWrapper(data []byte) bool {
	firstLine := strings.SplitN(string(data), "\n", 2)[0]
	if strings.HasPrefix(firstLine, "<!-- Aether-managed: runtime spec at .aether/commands/") &&
		strings.HasSuffix(firstLine, ". Synced by aether update. -->") {
		return true
	}
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

	// A durable-state marker, because .aether/ is indistinguishable from a
	// build cache to an outside observer: repo root, untracked, dominated by
	// ts-host/node_modules. A routine disk cleanup deleted one on 2026-08-16
	// for exactly that reason (.planning/field-reports/
	// 2026-08-16-init-obsidian-vault.md §4) — no colony existed there, but a
	// running colony would have been destroyed.
	markerPath := filepath.Join(localAether, "WHAT-IS-THIS.md")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		marker := `# What is this directory?

This is Aether's colony state for this repository — durable working memory,
not a build cache. Deleting it destroys any colony running here: its goal,
phase plan, learned lessons, and steering signals.

Safe to delete: ts-host/node_modules/ only (npm packages, ~60 MB — Aether
reinstalls them on demand).

Everything else here should be treated like your project's own files.
Managed by the aether CLI (https://github.com/calcosmic/Aether).
`
		if writeErr := os.WriteFile(markerPath, []byte(marker), 0644); writeErr != nil {
			result.errors = append(result.errors, fmt.Sprintf("write %s: %v", markerPath, writeErr))
		} else {
			result.copied++
		}
	} else if err == nil {
		result.skipped++
	}

	gitignorePath := filepath.Join(localAether, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		// ts-host/node_modules is ~60 MB of npm packages installed by
		// `aether update`; without this line a `git add .aether` (which the
		// versioned QUEEN.md invites) commits all of it.
		content := "# Aether local state - not versioned\ndata/\ncheckpoints/\nlocks/\ndreams/\noracle/\nts-host/node_modules/\n"
		if writeErr := os.WriteFile(gitignorePath, []byte(content), 0644); writeErr != nil {
			result.errors = append(result.errors, fmt.Sprintf("write %s: %v", gitignorePath, writeErr))
		} else {
			result.copied++
		}
	} else if err == nil {
		// Idempotent upgrade for repos scaffolded before the ts-host ignore
		// line existed: append it once, never rewrite user content.
		if data, readErr := os.ReadFile(gitignorePath); readErr == nil && !strings.Contains(string(data), "ts-host/node_modules") {
			appended := strings.TrimRight(string(data), "\n") + "\nts-host/node_modules/\n"
			if writeErr := os.WriteFile(gitignorePath, []byte(appended), 0644); writeErr != nil {
				result.errors = append(result.errors, fmt.Sprintf("append %s: %v", gitignorePath, writeErr))
			} else {
				result.copied++
			}
		} else {
			result.skipped++
		}
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
