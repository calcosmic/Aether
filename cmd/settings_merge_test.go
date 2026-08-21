package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testAetherSettingsTemplate = `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {"type": "command", "command": "aether hook-pre-tool-use", "timeout": 10}
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {"type": "command", "command": "aether hook-stop", "timeout": 10}
        ]
      }
    ],
    "PreCompact": [
      {
        "matcher": "manual",
        "hooks": [
          {"type": "command", "command": "aether hook-pre-compact", "timeout": 10}
        ]
      }
    ]
  }
}
`

const testForeignSettings = `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {"type": "command", "command": "gsd-sdk hook pre-bash", "timeout": 30}
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Write",
        "hooks": [
          {"type": "command", "command": "gsd-sdk hook post-write"}
        ]
      }
    ]
  },
  "permissions": {
    "allow": ["Bash(npm test:*)", "Read(~/.zshrc)"]
  },
  "gsdMarker": "do-not-touch"
}
`

func settingsHookCommands(t *testing.T, data []byte, event string) []string {
	t.Helper()
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("settings.json is not valid JSON after merge: %v\n%s", err, data)
	}
	hooks, _ := parsed["hooks"].(map[string]interface{})
	entries, _ := hooks[event].([]interface{})
	var commands []string
	for _, entry := range entries {
		m, _ := entry.(map[string]interface{})
		inner, _ := m["hooks"].([]interface{})
		for _, h := range inner {
			hm, _ := h.(map[string]interface{})
			if cmd, ok := hm["command"].(string); ok {
				commands = append(commands, cmd)
			}
		}
	}
	return commands
}

// TestUpdateForcePreservesForeignSettings is the WP1 gate: a forced update
// against a repo whose .claude/settings.json carries GSD hooks, permissions,
// and custom keys must keep every foreign entry while installing Aether's
// hooks. This test fails on the pre-merge behavior (wholesale file replace).
func TestUpdateForcePreservesForeignSettings(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()
	settingsHub := filepath.Join(hubDir, "system", "settings", "claude")
	if err := os.MkdirAll(settingsHub, 0755); err != nil {
		t.Fatalf("create hub settings dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(settingsHub, "settings.json"), []byte(testAetherSettingsTemplate), 0644); err != nil {
		t.Fatalf("write hub settings template: %v", err)
	}
	repoSettingsPath := filepath.Join(repoDir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(repoSettingsPath), 0755); err != nil {
		t.Fatalf("create repo .claude: %v", err)
	}
	if err := os.WriteFile(repoSettingsPath, []byte(testForeignSettings), 0644); err != nil {
		t.Fatalf("write repo settings: %v", err)
	}

	result := runUpdateSync(hubDir, repoDir, true)
	if len(result.errors) > 0 {
		t.Fatalf("runUpdateSync errors: %v", result.errors)
	}

	merged, err := os.ReadFile(repoSettingsPath)
	if err != nil {
		t.Fatalf("read merged settings: %v", err)
	}

	pre := settingsHookCommands(t, merged, "PreToolUse")
	if !containsString(pre, "gsd-sdk hook pre-bash") {
		t.Fatalf("foreign GSD PreToolUse hook lost after force update:\n%s", merged)
	}
	if !containsString(pre, "aether hook-pre-tool-use") {
		t.Fatalf("aether PreToolUse hook missing after force update:\n%s", merged)
	}
	post := settingsHookCommands(t, merged, "PostToolUse")
	if !containsString(post, "gsd-sdk hook post-write") {
		t.Fatalf("foreign GSD PostToolUse hook lost after force update:\n%s", merged)
	}
	for _, event := range []string{"Stop", "PreCompact"} {
		cmds := settingsHookCommands(t, merged, event)
		if len(cmds) == 0 {
			t.Fatalf("aether %s hook missing after force update:\n%s", event, merged)
		}
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(merged, &parsed); err != nil {
		t.Fatalf("merged settings invalid JSON: %v", err)
	}
	if parsed["gsdMarker"] != "do-not-touch" {
		t.Fatalf("custom top-level key lost after force update:\n%s", merged)
	}
	permissions, _ := parsed["permissions"].(map[string]interface{})
	allow, _ := permissions["allow"].([]interface{})
	if len(allow) != 2 || allow[0] != "Bash(npm test:*)" || allow[1] != "Read(~/.zshrc)" {
		t.Fatalf("foreign permissions altered after force update:\n%s", merged)
	}
}

// A second forced update must leave the merged file byte-identical.
func TestUpdateSettingsMergeIsIdempotent(t *testing.T) {
	saveGlobals(t)

	hubDir := t.TempDir()
	repoDir := t.TempDir()
	settingsHub := filepath.Join(hubDir, "system", "settings", "claude")
	if err := os.MkdirAll(settingsHub, 0755); err != nil {
		t.Fatalf("create hub settings dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(settingsHub, "settings.json"), []byte(testAetherSettingsTemplate), 0644); err != nil {
		t.Fatalf("write hub settings template: %v", err)
	}
	repoSettingsPath := filepath.Join(repoDir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(repoSettingsPath), 0755); err != nil {
		t.Fatalf("create repo .claude: %v", err)
	}
	if err := os.WriteFile(repoSettingsPath, []byte(testForeignSettings), 0644); err != nil {
		t.Fatalf("write repo settings: %v", err)
	}

	if result := runUpdateSync(hubDir, repoDir, true); len(result.errors) > 0 {
		t.Fatalf("first update errors: %v", result.errors)
	}
	afterFirst, err := os.ReadFile(repoSettingsPath)
	if err != nil {
		t.Fatalf("read settings after first update: %v", err)
	}
	if result := runUpdateSync(hubDir, repoDir, true); len(result.errors) > 0 {
		t.Fatalf("second update errors: %v", result.errors)
	}
	afterSecond, err := os.ReadFile(repoSettingsPath)
	if err != nil {
		t.Fatalf("read settings after second update: %v", err)
	}
	if !bytes.Equal(afterFirst, afterSecond) {
		t.Fatalf("repeated update rewrote settings.json:\nfirst:\n%s\nsecond:\n%s", afterFirst, afterSecond)
	}
}

func TestMergeClaudeSettingsRemovesStaleAetherHooks(t *testing.T) {
	existing := []byte(`{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "aether hook-retired"}]}
    ],
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "gsd-sdk hook pre-bash"}]}
    ]
  }
}`)
	merged, err := mergeClaudeSettings([]byte(testAetherSettingsTemplate), existing)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if strings.Contains(string(merged), "aether hook-retired") {
		t.Fatalf("retired aether hook survived merge:\n%s", merged)
	}
	if !strings.Contains(string(merged), "gsd-sdk hook pre-bash") {
		t.Fatalf("foreign hook lost during stale-hook cleanup:\n%s", merged)
	}
	if !strings.Contains(string(merged), "aether hook-pre-tool-use") {
		t.Fatalf("template hook not installed:\n%s", merged)
	}
}

func TestMergeClaudeSettingsRefusesInvalidExistingJSON(t *testing.T) {
	if _, err := mergeClaudeSettings([]byte(testAetherSettingsTemplate), []byte("{not json")); err == nil {
		t.Fatal("expected error for invalid existing settings.json, got nil")
	}
}

func TestMergeClaudeSettingsPreservesMixedOwnershipEntries(t *testing.T) {
	existing := []byte(`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write",
        "hooks": [
          {"type": "command", "command": "aether hook-pre-tool-use"},
          {"type": "command", "command": "gsd-sdk hook custom-chain"}
        ]
      }
    ]
  }
}`)
	merged, err := mergeClaudeSettings([]byte(testAetherSettingsTemplate), existing)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if !strings.Contains(string(merged), "gsd-sdk hook custom-chain") {
		t.Fatalf("mixed-ownership entry deleted (user customization lost):\n%s", merged)
	}
}

func TestMergeClaudeSettingsNoOpIsByteIdentical(t *testing.T) {
	existing := []byte(testForeignSettings)
	merged, err := mergeClaudeSettings([]byte(testAetherSettingsTemplate), existing)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	again, err := mergeClaudeSettings([]byte(testAetherSettingsTemplate), merged)
	if err != nil {
		t.Fatalf("second merge failed: %v", err)
	}
	if !bytes.Equal(merged, again) {
		t.Fatalf("no-op merge rewrote bytes:\nfirst:\n%s\nsecond:\n%s", merged, again)
	}
}
