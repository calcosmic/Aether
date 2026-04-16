package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

type installSyncPair struct {
	srcRel               string
	destRel              string
	label                string
	cleanup              bool
	preserveLocalChanges bool
	validate             syncValidator
}

type repoSyncPair struct {
	hubRel               string
	destRel              string
	label                string
	cleanup              bool
	preserveLocalChanges bool
	validate             syncValidator
}

type syncValidator func(srcPath, relPath string, data []byte) error

type codexAgentDefinition struct {
	Name                  string   `toml:"name"`
	Description           string   `toml:"description"`
	NicknameCandidates    []string `toml:"nickname_candidates"`
	DeveloperInstructions string   `toml:"developer_instructions"`
}

func installSyncPairs() []installSyncPair {
	return []installSyncPair{
		{srcRel: ".claude/commands/ant", destRel: ".claude/commands/ant", label: "Commands (claude)", cleanup: true},
		{srcRel: ".claude/agents/ant", destRel: ".claude/agents/ant", label: "Agents (claude)", cleanup: true},
		{srcRel: ".opencode/commands/ant", destRel: ".opencode/command", label: "Commands (opencode)", cleanup: true},
		{srcRel: ".opencode/agents", destRel: ".opencode/agent", label: "Agents (opencode)", cleanup: false},
		{srcRel: ".codex/agents", destRel: ".codex/agents", label: "Agents (codex)", cleanup: false, validate: validateCodexAgentFile},
		{srcRel: ".aether/skills-codex", destRel: ".codex/skills/aether", label: "Skills (codex)", cleanup: false},
	}
}

func repoSyncPairs() []repoSyncPair {
	return []repoSyncPair{
		{hubRel: ".", destRel: ".", label: "System files"},
		{hubRel: "commands/claude", destRel: "../.claude/commands/ant", label: "Commands (claude)"},
		{hubRel: "commands/opencode", destRel: "../.opencode/commands/ant", label: "Commands (opencode)"},
		{hubRel: "agents", destRel: "../.opencode/agents", label: "Agents (opencode)"},
		{hubRel: "agents-claude", destRel: "../.claude/agents/ant", label: "Agents (claude)"},
		{hubRel: "codex", destRel: "../.codex/agents", label: "Agents (codex)", preserveLocalChanges: true, validate: validateCodexAgentFile},
		{hubRel: "skills-codex", destRel: "../.codex/skills/aether", label: "Skills (codex)", preserveLocalChanges: true},
		{hubRel: "rules", destRel: "../.claude/rules", label: "Rules (claude)"},
	}
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
