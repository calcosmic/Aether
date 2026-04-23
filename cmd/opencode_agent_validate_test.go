package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: write a temp .md file with the given YAML frontmatter, return path and content bytes.
func writeTempAgentFile(t *testing.T, dir, frontmatter string) (string, []byte) {
	t.Helper()
	content := "---\n" + frontmatter + "\n---\n\n# Agent body\n"
	path := filepath.Join(dir, "aether-test-agent.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path, []byte(content)
}

// helper: write a temp file with completely raw content (no frontmatter wrapping).
func writeTempRawFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestValidateOpenCodeAgent(t *testing.T) {
	tmpDir := t.TempDir()

	validFrontmatter := `description: "This is a valid agent description for OpenCode"
mode: subagent
model: anthropic/claude-sonnet-4-20250514
color: "#ff0000"
tools:
  write: true
  edit: true
  bash: true
  grep: true
  glob: true
  task: true`

	t.Run("valid agent with hex color", func(t *testing.T) {
		path, data := writeTempAgentFile(t, tmpDir, validFrontmatter)
		if err := validateOpenCodeAgentFile(path, "aether-test-agent.md", data); err != nil {
			t.Fatalf("expected valid, got error: %v", err)
		}
	})

	t.Run("valid agent with theme color", func(t *testing.T) {
		fm := strings.Replace(validFrontmatter, `color: "#ff0000"`, "color: primary", 1)
		path, data := writeTempAgentFile(t, tmpDir, fm)
		if err := validateOpenCodeAgentFile(path, "aether-test-agent.md", data); err != nil {
			t.Fatalf("expected valid with theme color, got error: %v", err)
		}
	})

	t.Run("missing description", func(t *testing.T) {
		fm := `mode: subagent
model: anthropic/claude-sonnet-4-20250514
color: "#ff0000"
tools:
  write: true`
		path, data := writeTempAgentFile(t, tmpDir, fm)
		err := validateOpenCodeAgentFile(path, "aether-test-agent.md", data)
		if err == nil || !strings.Contains(err.Error(), "description") {
			t.Fatalf("expected description error, got: %v", err)
		}
	})

	t.Run("short description under 20 chars", func(t *testing.T) {
		fm := `description: "Too short"
mode: subagent
model: anthropic/claude-sonnet-4-20250514
color: "#ff0000"
tools:
  write: true`
		path, data := writeTempAgentFile(t, tmpDir, fm)
		err := validateOpenCodeAgentFile(path, "aether-test-agent.md", data)
		if err == nil || !strings.Contains(err.Error(), "20") {
			t.Fatalf("expected 20-char error, got: %v", err)
		}
	})

	t.Run("tools as string instead of map", func(t *testing.T) {
		fm := `description: "This is a valid agent description for OpenCode"
mode: subagent
model: anthropic/claude-sonnet-4-20250514
color: "#ff0000"
tools: "Read, Write, Edit, Bash"`
		path, data := writeTempAgentFile(t, tmpDir, fm)
		err := validateOpenCodeAgentFile(path, "aether-test-agent.md", data)
		if err == nil {
			t.Fatalf("expected error for string tools, got nil")
		}
		// Either the struct unmarshal rejects it as invalid YAML, or the raw
		// re-parse detects the wrong type — both are acceptable error paths.
	})

	t.Run("missing tools field entirely", func(t *testing.T) {
		fm := `description: "This is a valid agent description for OpenCode"
mode: subagent
model: anthropic/claude-sonnet-4-20250514
color: "#ff0000"`
		path, data := writeTempAgentFile(t, tmpDir, fm)
		err := validateOpenCodeAgentFile(path, "aether-test-agent.md", data)
		if err == nil || !strings.Contains(err.Error(), "tools") {
			t.Fatalf("expected missing tools error, got: %v", err)
		}
	})

	t.Run("named color not allowed", func(t *testing.T) {
		fm := strings.Replace(validFrontmatter, `color: "#ff0000"`, "color: red", 1)
		path, data := writeTempAgentFile(t, tmpDir, fm)
		err := validateOpenCodeAgentFile(path, "aether-test-agent.md", data)
		if err == nil || (!strings.Contains(err.Error(), "hex") && !strings.Contains(err.Error(), "theme")) {
			t.Fatalf("expected hex/theme color error, got: %v", err)
		}
	})

	t.Run("valid theme colors all pass", func(t *testing.T) {
		for _, theme := range []string{"primary", "secondary", "accent", "success", "warning", "error", "info"} {
			fm := strings.Replace(validFrontmatter, `color: "#ff0000"`, "color: "+theme, 1)
			path, data := writeTempAgentFile(t, tmpDir, fm)
			if err := validateOpenCodeAgentFile(path, "aether-test-agent.md", data); err != nil {
				t.Errorf("theme color %q should be valid, got: %v", theme, err)
			}
		}
	})

	t.Run("name field present must fail", func(t *testing.T) {
		fm := `name: aether-test-agent
description: "This is a valid agent description for OpenCode"
mode: subagent
model: anthropic/claude-sonnet-4-20250514
color: "#ff0000"
tools:
  write: true`
		path, data := writeTempAgentFile(t, tmpDir, fm)
		err := validateOpenCodeAgentFile(path, "aether-test-agent.md", data)
		if err == nil || (!strings.Contains(err.Error(), "name") && !strings.Contains(err.Error(), "filename")) {
			t.Fatalf("expected name-field error, got: %v", err)
		}
	})

	t.Run("model without slash must fail", func(t *testing.T) {
		fm := strings.Replace(validFrontmatter, "anthropic/claude-sonnet-4-20250514", "opus", 1)
		path, data := writeTempAgentFile(t, tmpDir, fm)
		err := validateOpenCodeAgentFile(path, "aether-test-agent.md", data)
		if err == nil || !strings.Contains(err.Error(), "provider/model") {
			t.Fatalf("expected provider/model error, got: %v", err)
		}
	})

	t.Run("missing model must fail", func(t *testing.T) {
		fm := `description: "This is a valid agent description for OpenCode"
mode: subagent
color: "#ff0000"
tools:
  write: true`
		path, data := writeTempAgentFile(t, tmpDir, fm)
		err := validateOpenCodeAgentFile(path, "aether-test-agent.md", data)
		if err == nil || !strings.Contains(err.Error(), "model") {
			t.Fatalf("expected model error, got: %v", err)
		}
	})

	t.Run("missing frontmatter delimiters", func(t *testing.T) {
		path := writeTempRawFile(t, tmpDir, "no-frontmatter.md", "# Just a markdown file\n")
		err := validateOpenCodeAgentFile(path, "no-frontmatter.md", []byte("# Just a markdown file\n"))
		if err == nil || !strings.Contains(err.Error(), "frontmatter") {
			t.Fatalf("expected frontmatter error, got: %v", err)
		}
	})

	t.Run("non-md extension rejected", func(t *testing.T) {
		path := writeTempRawFile(t, tmpDir, "agent.txt", "---\ndescription: test\n---\n")
		err := validateOpenCodeAgentFile(path, "agent.txt", []byte("---\ndescription: test\n---\n"))
		if err == nil || !strings.Contains(err.Error(), ".md") {
			t.Fatalf("expected .md extension error, got: %v", err)
		}
	})

	t.Run("invalid YAML rejected", func(t *testing.T) {
		content := "---\n\t: invalid yaml [[[\n---\n"
		path := writeTempRawFile(t, tmpDir, "bad-yaml.md", content)
		err := validateOpenCodeAgentFile(path, "bad-yaml.md", []byte(content))
		if err == nil {
			t.Fatalf("expected YAML parse error, got nil")
		}
	})

	t.Run("all 25 real agent files pass validation", func(t *testing.T) {
		repoRoot, err := findOpenCodeRepoRoot()
		if err != nil {
			t.Skip("repo root not found, skipping real file validation")
		}
		agentsDir := filepath.Join(repoRoot, ".opencode", "agents")
		entries, err := os.ReadDir(agentsDir)
		if err != nil {
			t.Fatalf("read agents dir: %v", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
				continue
			}
			path := filepath.Join(agentsDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", entry.Name(), err)
			}
			if err := validateOpenCodeAgentFile(path, entry.Name(), data); err != nil {
				t.Errorf("real agent file %s failed validation: %v", entry.Name(), err)
			}
		}
	})
}
