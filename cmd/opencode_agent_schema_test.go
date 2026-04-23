package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// findOpenCodeRepoRoot walks up from the current working directory looking for
// the .opencode/agents directory as a marker for the repo root.
func findOpenCodeRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".opencode", "agents")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root (no .opencode/agents/ found)")
		}
		dir = parent
	}
}

// TestOpenCodeAgentSchema validates that all 25 OpenCode agent files
// in .opencode/agents/ have valid YAML frontmatter per the OpenCode spec.
func TestOpenCodeAgentSchema(t *testing.T) {
	repoRoot, err := findOpenCodeRepoRoot()
	if err != nil {
		t.Skipf("repo root not found: %v", err)
	}

	agentsDir := filepath.Join(repoRoot, ".opencode", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip(".opencode/agents/ directory not found — skipping")
		}
		t.Fatalf("failed to read .opencode/agents/: %v", err)
	}

	var agentFiles []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "aether-") || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		agentFiles = append(agentFiles, filepath.Join(agentsDir, e.Name()))
	}

	if len(agentFiles) == 0 {
		t.Fatal("no aether-*.md files found in .opencode/agents/")
	}
	if len(agentFiles) != 25 {
		t.Errorf("expected 25 agent files, found %d", len(agentFiles))
	}

	hexColorRe := regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	themeColors := map[string]bool{
		"primary": true, "secondary": true, "accent": true,
		"success": true, "warning": true, "error": true, "info": true,
	}

	for _, path := range agentFiles {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			fm, err := extractYAMLFrontmatter(data)
			if err != nil {
				t.Fatalf("failed to parse frontmatter: %v", err)
			}

			// Rule 1: description must exist and be at least 20 characters
			desc, ok := fm["description"].(string)
			if !ok || strings.TrimSpace(desc) == "" {
				t.Error("missing or empty description field")
			} else if len(desc) < 20 {
				t.Errorf("description too short (%d chars, need 20+): %q", len(desc), desc)
			}

			// Rule 2: mode must be one of the valid values
			mode, ok := fm["mode"].(string)
			if !ok {
				t.Error("missing mode field")
			} else if mode != "primary" && mode != "subagent" && mode != "all" {
				t.Errorf("invalid mode %q (must be primary, subagent, or all)", mode)
			}

			// Rule 3: tools must be a map/object, not a string
			tools := fm["tools"]
			if tools == nil {
				t.Error("missing tools field")
			} else if _, ok := tools.(map[string]interface{}); !ok {
				if _, isStr := tools.(string); isStr {
					t.Error("tools is a string (should be an object with true/false values)")
				} else {
					t.Errorf("tools has unexpected type %T", tools)
				}
			}

			// Rule 4: color must be hex (#rrggbb) or a theme color name
			color, ok := fm["color"].(string)
			if !ok || strings.TrimSpace(color) == "" {
				t.Error("missing or empty color field")
			} else if !hexColorRe.MatchString(color) && !themeColors[color] {
				t.Errorf("invalid color %q (must be hex #rrggbb or theme color)", color)
			}

			// Rule 5: no name field (filename is the name)
			if _, hasName := fm["name"]; hasName {
				t.Error("name field must not exist (filename IS the name)")
			}

			// Rule 6: model must use provider/model-id format (contains /)
			model, ok := fm["model"].(string)
			if !ok || strings.TrimSpace(model) == "" {
				t.Error("missing or empty model field")
			} else if !strings.Contains(model, "/") {
				t.Errorf("model %q must use provider/model-id format (e.g. anthropic/claude-sonnet-4-20250514)", model)
			}
		})
	}
}

// extractYAMLFrontmatter parses the content between --- delimiters and
// unmarshals it as YAML into a generic map.
func extractYAMLFrontmatter(data []byte) (map[string]interface{}, error) {
	content := string(data)

	// Find opening ---
	start := strings.Index(content, "---")
	if start == -1 {
		return nil, fmt.Errorf("no frontmatter delimiter found")
	}

	// Find closing --- after the opening
	end := strings.Index(content[start+3:], "---")
	if end == -1 {
		return nil, fmt.Errorf("no closing frontmatter delimiter found")
	}

	yamlContent := content[start+3 : start+3+end]

	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &result); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return result, nil
}
