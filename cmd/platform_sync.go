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
		{srcRel: ".opencode/commands/ant", destRel: ".opencode/command", label: "Commands (opencode)", cleanup: true},
		{srcRel: ".opencode/agents", destRel: ".opencode/agent", label: "Agents (opencode)", cleanup: false, validate: validateOpenCodeAgentFile},
		{srcRel: ".codex/agents", destRel: ".codex/agents", label: "Agents (codex)", cleanup: false, preserveLocalChanges: true, validate: validateCodexAgentFile, include: isShippedAetherCodexAgent},
		{srcRel: ".aether/skills-codex", destRel: ".codex/skills/aether", label: "Skills (codex)", cleanup: false, preserveLocalChanges: true},
	}
}

func repoSyncPairs() []repoSyncPair {
	return []repoSyncPair{
		{hubRel: ".", destRel: ".", label: "System files"},
		{hubRel: "commands/claude", destRel: "../.claude/commands", label: "Commands (claude)", mapRelPath: claudeCommandDestRelPath, cleanupInclude: isManagedFlatClaudeCommandPath, cleanupLegacyClaude: true},
		{hubRel: "settings/claude", destRel: "../.claude", label: "Settings (claude)", preserveLocalChanges: true, include: isClaudeSettingsFile},
		{hubRel: "commands/opencode", destRel: "../.opencode/commands/ant", label: "Commands (opencode)"},
		{hubRel: "agents", destRel: "../.opencode/agents", label: "Agents (opencode)", validate: validateOpenCodeAgentFile},
		{hubRel: "agents-claude", destRel: "../.claude/agents/ant", label: "Agents (claude)"},
		{hubRel: "codex", destRel: "../.codex/agents", label: "Agents (codex)", preserveLocalChanges: true, validate: validateCodexAgentFile, include: isShippedAetherCodexAgent},
		{hubRel: "skills-codex", destRel: "../.codex/skills/aether", label: "Skills (codex)", preserveLocalChanges: true},
		{hubRel: "rules", destRel: "../.claude/rules", label: "Rules (claude)"},
	}
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
// agent file. The `name` field must NOT be present — filename is the agent name.
type openCodeAgentFrontmatter struct {
	Description string                 `yaml:"description"`
	Mode        string                 `yaml:"mode"`
	Tools       map[string]interface{} `yaml:"tools"`
	Color       string                 `yaml:"color"`
	Model       string                 `yaml:"model"`
	Name        string                 `yaml:"name"` // Must NOT be present
}

var openCodeThemeColors = map[string]bool{
	"primary": true, "secondary": true, "accent": true,
	"success": true, "warning": true, "error": true, "info": true,
}

var openCodeHexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// validateOpenCodeAgentFile validates an OpenCode agent markdown file.
// It checks that the YAML frontmatter conforms to the OpenCode agent schema:
// description (20+ chars), tools (object/map), color (hex or theme), no name field,
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

	// Rule 8: name field must NOT be present
	if _, hasName := rawFM["name"]; hasName {
		return fmt.Errorf("%s must not have a name field in frontmatter — filename is the agent name", relPath)
	}

	// Rule 9: model must contain a / (provider/model-id format)
	model := strings.TrimSpace(fm.Model)
	if model == "" {
		return fmt.Errorf("%s is missing model in frontmatter", relPath)
	}
	if !strings.Contains(model, "/") {
		return fmt.Errorf("%s model %q must use provider/model-id format (e.g. anthropic/claude-sonnet-4-20250514)", relPath, model)
	}

	// Reject binary-like content masquerading as text
	if info, err := os.Stat(srcPath); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", relPath)
	}

	return nil
}
