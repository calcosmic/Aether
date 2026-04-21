package cmd

import (
	"fmt"
	"path/filepath"
)

// Expected file counts across Aether surfaces. Update when commands/agents are added or removed.
const (
	expectedYAMLCommands     = 50 // 49 existing + 1 medic
	expectedClaudeCommands   = 50
	expectedOpenCodeCommands = 50
	expectedClaudeAgents     = 25 // 24 existing + 1 medic
	expectedOpenCodeAgents   = 25
	expectedCodexAgents      = 25
	expectedClaudeMirror     = 25
	expectedCodexMirror      = 25
	expectedColonySkills     = 11 // 10 existing + 1 medic
	expectedDomainSkills     = 18
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

	surfaces := []wrapperSurface{
		{"YAML commands", filepath.Join(fc.repoRoot, ".aether", "commands", "*.yaml"), expectedYAMLCommands},
		{"Claude commands", filepath.Join(fc.repoRoot, ".claude", "commands", "ant", "*.md"), expectedClaudeCommands},
		{"OpenCode commands", filepath.Join(fc.repoRoot, ".opencode", "commands", "ant", "*.md"), expectedOpenCodeCommands},
		{"Codex agents", filepath.Join(fc.repoRoot, ".codex", "agents", "*.toml"), expectedCodexAgents},
		{"Claude agents", filepath.Join(fc.repoRoot, ".claude", "agents", "ant", "*.md"), expectedClaudeAgents},
		{"OpenCode agents", filepath.Join(fc.repoRoot, ".opencode", "agents", "*.md"), expectedOpenCodeAgents},
		{"Claude mirror", filepath.Join(fc.repoRoot, ".aether", "agents-claude", "*.md"), expectedClaudeMirror},
		{"Codex mirror", filepath.Join(fc.repoRoot, ".aether", "agents-codex", "*.toml"), expectedCodexMirror},
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

	// Cross-surface consistency: agent counts must match
	codexAgentCount := counts["Codex agents"]
	claudeAgentCount := counts["Claude agents"]
	opencodeAgentCount := counts["OpenCode agents"]
	claudeMirrorCount := counts["Claude mirror"]
	codexMirrorCount := counts["Codex mirror"]
	if claudeAgentCount != codexAgentCount || claudeAgentCount != opencodeAgentCount ||
		claudeAgentCount != claudeMirrorCount || claudeAgentCount != codexMirrorCount {
		issues = append(issues, issueWarning("wrapper", "agents",
			fmt.Sprintf("Agent count mismatch: Claude=%d, OpenCode=%d, Codex=%d, ClaudeMirror=%d, CodexMirror=%d",
				claudeAgentCount, opencodeAgentCount, codexAgentCount, claudeMirrorCount, codexMirrorCount)))
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

// countFilesInDir returns the number of files matching the given glob pattern.
func countFilesInDir(pattern string) int {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0
	}
	return len(matches)
}
