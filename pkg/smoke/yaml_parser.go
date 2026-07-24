package smoke

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLCommand represents a command definition extracted from a YAML source file.
type YAMLCommand struct {
	Name       string   `json:"name"`
	Flags      []string `json:"flags"`
	IsHost     bool     `json:"is_host"`
	SourceFile string   `json:"source_file"`
}

// Host command names that use DisableFlagParsing and forward to the TS host.
var hostCommandNames = map[string]bool{
	"ant-plan":      true,
	"ant-build":     true,
	"ant-continue":  true,
	"ant-oracle":    true,
	"ant-watch":     true,
	"ant-swarm":     true,
	"ant-lifecycle": true,
}

// flagPattern matches --flag-name style flags.
var flagPattern = regexp.MustCompile(`--[a-zA-Z0-9-]+`)

// codeBlockPattern matches markdown code blocks.
var codeBlockPattern = regexp.MustCompile("(?s)`{1,3}[^`]*`{1,3}")

// ParseYAMLCommands reads all YAML files matching the glob, extracts command
// names and flag mentions, and returns a slice of YAMLCommand.
func ParseYAMLCommands(glob string) ([]YAMLCommand, error) {
	matches, err := filepath.Glob(glob)
	if err != nil {
		return nil, fmt.Errorf("glob failed: %w", err)
	}

	var commands []YAMLCommand
	for _, path := range matches {
		cmd, err := parseSingleYAML(path)
		if err != nil {
			// Parse errors are warnings, not failures
			continue
		}
		commands = append(commands, cmd)
	}

	return commands, nil
}

func parseSingleYAML(path string) (YAMLCommand, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return YAMLCommand{}, err
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return YAMLCommand{}, err
	}

	name, _ := doc["name"].(string)
	if name == "" {
		return YAMLCommand{}, fmt.Errorf("missing name in %s", path)
	}

	isHost := hostCommandNames[name]

	// Extract all text fields that might contain flags
	var textFields []string

	// runtime.command
	if runtime, ok := doc["runtime"].(map[string]interface{}); ok {
		if cmd, ok := runtime["command"].(string); ok {
			textFields = append(textFields, cmd)
		}
	}

	// wrapper_additions values
	if additions, ok := doc["wrapper_additions"].(map[string]interface{}); ok {
		for _, v := range additions {
			if s, ok := v.(string); ok {
				textFields = append(textFields, s)
			}
		}
	}

	// guardrails values
	if guardrails, ok := doc["guardrails"].([]interface{}); ok {
		for _, v := range guardrails {
			if s, ok := v.(string); ok {
				textFields = append(textFields, s)
			}
		}
	}

	// intent_refinement values
	if intent, ok := doc["intent_refinement"].([]interface{}); ok {
		for _, v := range intent {
			if s, ok := v.(string); ok {
				textFields = append(textFields, s)
			}
		}
	}

	// follow_up values
	if followUp, ok := doc["follow_up"].(map[string]interface{}); ok {
		for _, v := range followUp {
			if s, ok := v.(string); ok {
				textFields = append(textFields, s)
			}
		}
	}

	// codex_orchestration values
	if codex, ok := doc["codex_orchestration"].(map[string]interface{}); ok {
		for _, v := range codex {
			if s, ok := v.(string); ok {
				textFields = append(textFields, s)
			}
		}
	}

	flags := extractFlags(textFields)

	return YAMLCommand{
		Name:       name,
		Flags:      flags,
		IsHost:     isHost,
		SourceFile: filepath.Base(path),
	}, nil
}

// extractFlags finds flag mentions in text fields using a heuristic:
// a flag counts if it appears at least once as a standalone token
// (whitespace-delimited) or inside a markdown code block.
func extractFlags(texts []string) []string {
	seen := make(map[string]bool)

	for _, text := range texts {
		// Find all code blocks first and mark flags inside them
		codeBlocks := codeBlockPattern.FindAllString(text, -1)
		for _, block := range codeBlocks {
			matches := flagPattern.FindAllString(block, -1)
			for _, m := range matches {
				seen[m] = true
			}
		}

		// Also accept flags that appear as standalone tokens
		matches := flagPattern.FindAllString(text, -1)
		for _, m := range matches {
			if isStandaloneToken(text, m) {
				seen[m] = true
			}
		}
	}

	// Exclude --help from individual command flags since it's universal
	delete(seen, "--help")

	var result []string
	for f := range seen {
		result = append(result, f)
	}
	sort.Strings(result)
	return result
}

// isStandaloneToken checks if the flag appears as a standalone token
// (surrounded by whitespace or punctuation, not embedded in a word).
func isStandaloneToken(text, flag string) bool {
	idx := strings.Index(text, flag)
	if idx == -1 {
		return false
	}

	// Check prefix
	if idx > 0 {
		before := text[idx-1]
		if before != ' ' && before != '\t' && before != '\n' && before != '`' &&
			before != '(' && before != '[' && before != '{' && before != '"' &&
			before != '\'' && before != '<' && before != '|' && before != '-' {
			return false
		}
	}

	// Check suffix
	afterIdx := idx + len(flag)
	if afterIdx < len(text) {
		after := text[afterIdx]
		if after != ' ' && after != '\t' && after != '\n' && after != '`' &&
			after != ')' && after != ']' && after != '}' && after != '"' &&
			after != '\'' && after != '>' && after != '|' && after != ',' &&
			after != '.' && after != ':' && after != ';' && after != '!' &&
			after != '?' {
			return false
		}
	}

	return true
}
