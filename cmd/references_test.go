package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReferenceCommandsRegistered(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	for _, name := range []string{"reference-index", "reference-list", "reference-match"} {
		cmd, _, err := rootCmd.Find([]string{name})
		if err != nil {
			t.Fatalf("%s command not found: %v", name, err)
		}
		if cmd == nil {
			t.Fatalf("%s command is nil", name)
		}
	}
}

func TestReferenceMatchUsesHubReferences(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	home := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", home)
	writeReferenceFixture(t, filepath.Join(home, "system", "references"))

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	rootCmd.SetArgs([]string{"reference-match", "--role", "builder", "--workflow", "build", "--task", "implement update safety for hub references"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("reference-match failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}
	result := env["result"].(map[string]interface{})
	if int(result["total"].(float64)) == 0 {
		t.Fatalf("expected at least one reference match, got %v", result)
	}
}

func TestResolveReferenceSectionRendersTopMatches(t *testing.T) {
	saveGlobals(t)

	home := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", home)
	writeReferenceFixture(t, filepath.Join(home, "system", "references"))

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	section := resolveReferenceSection("builder", "implement update safety for hub references", "build")
	if !strings.Contains(section, "## Reference Library") {
		t.Fatalf("missing reference library heading:\n%s", section)
	}
	if !strings.Contains(section, "Update Safety Contract") {
		t.Fatalf("missing matched reference title:\n%s", section)
	}
	if !strings.Contains(section, "Safety body") {
		t.Fatalf("missing matched reference body:\n%s", section)
	}
}

func writeReferenceFixture(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir references: %v", err)
	}
	content := `---
schema_version: "1.0"
id: update-safety-contract
kind: contract
category: contracts
title: Update Safety Contract
description: "Safety rules for update and hub reference work."
output_types: [distribution-review]
agent_roles: [builder, watcher]
task_types: [update, reference]
task_keywords: [update, hub, references, safety]
workflow_triggers: [build]
priority: critical
version: "1.0"
render:
  mode: full
  max_chars: 1000
---
# Update Safety Contract

Safety body.
`
	if err := os.WriteFile(filepath.Join(dir, "update-safety-contract.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write reference fixture: %v", err)
	}
}
