package llm

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// TriggerConfig defines an event trigger parsed from agent YAML frontmatter.
type TriggerConfig struct {
	Topic  string         `yaml:"topic"`
	Filter map[string]any `yaml:"filter"`
}

// AgentConfig holds the parsed agent specification from YAML frontmatter.
type AgentConfig struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Tools       []string       `yaml:"tools"`
	Color       string         `yaml:"color"`
	Model       string         `yaml:"model"`
	Triggers    []TriggerConfig `yaml:"triggers"`
}

// ParseAgentSpec parses YAML frontmatter from raw agent definition content.
// The content must start with "---" and contain a second "---" delimiter.
// Returns an error for missing delimiters, unclosed frontmatter, invalid YAML,
// or an empty Name field.
func ParseAgentSpec(content []byte) (*AgentConfig, error) {
	content = bytes.TrimLeft(content, " \t\r\n")

	if !bytes.HasPrefix(content, []byte("---")) {
		return nil, errors.New("missing opening --- delimiter")
	}

	// Find the closing --- after the opening one (skip at least 3 bytes past opening)
	rest := content[3:]
	closeIdx := bytes.Index(rest, []byte("\n---"))
	if closeIdx == -1 {
		// Try without leading newline (in case --- is right at start of next line)
		closeIdx = bytes.Index(rest, []byte("---"))
		if closeIdx == 0 {
			// Empty frontmatter
			return nil, errors.New("empty frontmatter")
		}
		if closeIdx == -1 {
			return nil, errors.New("missing closing --- delimiter")
		}
	}

	yamlContent := rest[:closeIdx]
	// Strip leading newline from yamlContent if present
	yamlContent = bytes.TrimPrefix(yamlContent, []byte("\n"))

	var cfg AgentConfig
	if err := yaml.Unmarshal(yamlContent, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	if cfg.Name == "" {
		return nil, errors.New("agent name is required")
	}

	return &cfg, nil
}

// ParseAgentSpecFile reads a file and parses its YAML frontmatter into an AgentConfig.
func ParseAgentSpecFile(path string) (*AgentConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading agent spec file: %w", err)
	}
	return ParseAgentSpec(content)
}
