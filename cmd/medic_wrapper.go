package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Expected file counts across Aether agent and skill surfaces. Update when
// agents or skills are added or removed.
//
// Menu commands have no typed count here on purpose. A typed "60" went stale
// as commands were added and made `medic --deep` call a correct install broken
// ("67 files, expected 60", Phase 210 blocker 17). The command set is the one
// compiled into this program, wrapperCommandNames, which
// TestWrapperCommandNamesMatchCanonicalCorpus keeps in lockstep with the
// shipped .claude/commands/ant/*.md files.
const (
	expectedClaudeAgents   = 27
	expectedOpenCodeAgents = 28 // 27 castes plus the restricted primary router
	expectedCodexAgents    = 27
	expectedColonySkills   = 55
	expectedDomainSkills   = 31
	expectedCodexSkills    = expectedColonySkills + expectedDomainSkills
)

// wrapperSurface represents a surface to check for file count parity.
type wrapperSurface struct {
	name     string
	pattern  string
	expected int
}

// scanWrapperParity checks that command and agent file counts match across all surfaces.
// Uses the repo root since wrapper files live at repo root, not inside .aether/data/.
func scanWrapperParity(fc *fileChecker) []HealthIssue {
	var issues []HealthIssue
	if !isAetherSourceCheckout(fc.repoRoot) {
		return issues
	}

	shippedCommands := shippedWrapperCommandCount()
	surfaces := []wrapperSurface{
		{"YAML commands", filepath.Join(fc.repoRoot, ".aether", "commands", "*.yaml"), shippedCommands},
		{"Claude commands", filepath.Join(fc.repoRoot, ".claude", "commands", "ant", "*.md"), shippedCommands},
		{"OpenCode commands", filepath.Join(fc.repoRoot, ".opencode", "commands", "ant", "*.md"), shippedCommands},
		{"Codex agents", filepath.Join(fc.repoRoot, ".codex", "agents", "*.toml"), expectedCodexAgents},
		{"Claude agents", filepath.Join(fc.repoRoot, ".claude", "agents", "ant", "*.md"), expectedClaudeAgents},
		{"OpenCode agents", filepath.Join(fc.repoRoot, ".opencode", "agents", "*.md"), expectedOpenCodeAgents},
	}

	// Count each surface and check against expected
	counts := make(map[string]int)
	for _, s := range surfaces {
		actual := countFilesInDir(s.pattern)
		counts[s.name] = actual
		if actual != s.expected {
			issues = append(issues, issueWarning("wrapper", s.name,
				fmt.Sprintf("%s has %d files, expected %d", s.name, actual, s.expected)))
		}
	}

	// Cross-surface consistency: command counts must match
	yamlCount := counts["YAML commands"]
	claudeCmdCount := counts["Claude commands"]
	opencodeCmdCount := counts["OpenCode commands"]
	if yamlCount != claudeCmdCount || yamlCount != opencodeCmdCount {
		issues = append(issues, issueWarning("wrapper", "commands",
			fmt.Sprintf("Command count mismatch: YAML=%d, Claude=%d, OpenCode=%d",
				yamlCount, claudeCmdCount, opencodeCmdCount)))
	}

	// Cross-surface consistency: caste counts must match. OpenCode also ships one
	// infrastructure-only primary router so provider-native task dispatch can be
	// locked down without pretending that router is a colony caste.
	codexAgentCount := counts["Codex agents"]
	claudeAgentCount := counts["Claude agents"]
	opencodeAgentCount := counts["OpenCode agents"]
	opencodeCasteCount := opencodeAgentCount
	routerPath := filepath.Join(fc.repoRoot, ".opencode", "agents", "aether-worker-router.md")
	if _, err := os.Stat(routerPath); err == nil {
		opencodeCasteCount--
	} else {
		issues = append(issues, issueWarning("wrapper", "OpenCode worker router",
			"required infrastructure agent aether-worker-router.md is missing"))
	}
	if claudeAgentCount != codexAgentCount || claudeAgentCount != opencodeCasteCount {
		issues = append(issues, issueWarning("wrapper", "agents",
			fmt.Sprintf("Caste count mismatch: Claude=%d, OpenCode=%d, Codex=%d",
				claudeAgentCount, opencodeCasteCount, codexAgentCount)))
	}

	// Colony skills count
	colonySkillsPattern := filepath.Join(fc.repoRoot, ".aether", "skills", "colony", "*", "SKILL.md")
	colonySkillCount := countFilesInDir(colonySkillsPattern)
	if colonySkillCount != expectedColonySkills {
		issues = append(issues, issueWarning("wrapper", "colony-skills",
			fmt.Sprintf("Colony skills has %d files, expected %d", colonySkillCount, expectedColonySkills)))
	}

	// Domain skills count
	domainSkillsPattern := filepath.Join(fc.repoRoot, ".aether", "skills", "domain", "*", "SKILL.md")
	domainSkillCount := countFilesInDir(domainSkillsPattern)
	if domainSkillCount != expectedDomainSkills {
		issues = append(issues, issueWarning("wrapper", "domain-skills",
			fmt.Sprintf("Domain skills has %d files, expected %d", domainSkillCount, expectedDomainSkills)))
	}

	return issues
}

// scanHubPublishIntegrity checks that the shared hub contains the published
// platform surfaces that downstream `aether update` depends on.
func scanHubPublishIntegrity() []HealthIssue {
	var issues []HealthIssue

	hubDir := resolveHubPath()
	if hubDir == "" {
		return issues
	}

	hubSystem := filepath.Join(hubDir, "system")
	info, err := os.Stat(hubSystem)
	if err != nil || !info.IsDir() {
		issues = append(issues, issueCritical("publish", hubSystem,
			fmt.Sprintf("Hub system directory missing at %s; run `aether publish --package-dir <Aether checkout>` from the Aether repo", hubSystem)))
		return issues
	}

	const republish = "Republish from the Aether repo with `aether publish --package-dir <Aether checkout>`, then rerun `aether update --force` in target repos."

	// Menu commands are checked by name against the set this program ships,
	// so a hub is only reported when a command it should carry is missing.
	commandSurfaces := []wrapperSurface{
		{"Hub Claude commands", filepath.Join(hubSystem, "commands", "claude", "*.md"), shippedWrapperCommandCount()},
		{"Hub OpenCode commands", filepath.Join(hubSystem, "commands", "opencode", "*.md"), shippedWrapperCommandCount()},
	}
	counts := make(map[string]int, len(commandSurfaces))
	for _, s := range commandSurfaces {
		counts[s.name] = countFilesInDir(s.pattern)
		dir := filepath.Dir(s.pattern)
		if missing := missingShippedWrappers(dir); len(missing) > 0 {
			issues = append(issues, issueCritical("publish", dir,
				fmt.Sprintf("%s is missing %d of the %d menu commands this version of Aether ships (%s), so the shared copy of Aether on this machine is incomplete. %s",
					s.name, len(missing), s.expected, summarizeNames(missing, 5), republish)))
		}
	}

	surfaces := []wrapperSurface{
		{"Hub Claude agents", filepath.Join(hubSystem, "agents-claude", "*.md"), expectedClaudeAgents},
		{"Hub OpenCode agents", filepath.Join(hubSystem, "agents", "*.md"), expectedOpenCodeAgents},
		{"Hub Codex agents", filepath.Join(hubSystem, "codex", "*.toml"), expectedCodexAgents},
		{"Hub shipped skills", filepath.Join(hubSystem, "skills", "*", "*", "SKILL.md"), expectedCodexSkills},
	}
	for _, s := range surfaces {
		actual := countFilesInDir(s.pattern)
		if actual != s.expected {
			issues = append(issues, issueCritical("publish", filepath.Dir(s.pattern),
				fmt.Sprintf("%s has %d files, expected %d. %s", s.name, actual, s.expected, republish)))
		}
	}

	if counts["Hub Claude commands"] != counts["Hub OpenCode commands"] {
		issues = append(issues, issueCritical("publish", filepath.Join(hubSystem, "commands"),
			fmt.Sprintf("Hub wrapper command mismatch: Claude=%d, OpenCode=%d. Republish the hub before trusting downstream `aether update` results.",
				counts["Hub Claude commands"], counts["Hub OpenCode commands"])))
	}

	return issues
}

// shippedWrapperCommandCount is how many `/ant-…` menu commands this program
// ships, read from the compiled-in wrapperCommandNames set rather than typed.
func shippedWrapperCommandCount() int {
	return len(wrapperCommandNames)
}

// missingShippedWrappers lists, sorted, the shipped menu commands that have no
// <name>.md wrapper in dir.
func missingShippedWrappers(dir string) []string {
	var missing []string
	for name := range wrapperCommandNames {
		if _, err := os.Stat(filepath.Join(dir, name+".md")); err != nil {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

// summarizeNames joins up to limit names and says how many more there are.
func summarizeNames(names []string, limit int) string {
	if len(names) <= limit {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(names[:limit], ", "), len(names)-limit)
}

// countFilesInDir returns the number of files matching the given glob pattern.
func countFilesInDir(pattern string) int {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0
	}
	return len(matches)
}
