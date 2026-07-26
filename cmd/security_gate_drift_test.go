package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The Gatekeeper playbooks invoke `aether check-antipattern "<file>"` with the
// file as a positional argument. For the framework's entire life the command
// was cobra.NoArgs + --file only, so every playbook invocation errored, the
// error was piped to /dev/null, and the security gate never executed once.
// These tests bind the command's contract to the playbooks' exact invocation
// form so the two cannot silently drift apart again.

func TestCheckAntipatternAcceptsPlaybookInvocationForm(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// A file with a hardcoded credential — the universal critical pattern.
	target := filepath.Join(tmpDir, "config.go")
	if err := os.WriteFile(target, []byte("package config\n\nvar apiKey = \"sk-live-4f9a8b7c6d\"\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	// The playbook's exact form: positional argument, no --file flag.
	rootCmd.SetArgs([]string{"check-antipattern", target})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("playbook-form invocation failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("playbook-form invocation not ok: %s", buf.String())
	}
	result := env["result"].(map[string]interface{})
	criticals, _ := result["critical"].([]interface{})
	if len(criticals) == 0 {
		t.Fatalf("hardcoded credential not detected; the gate would pass a leaked secret: %s", buf.String())
	}
	if result["clean"] == true {
		t.Fatal("file with a hardcoded credential reported clean")
	}
}

func TestCheckAntipatternFlagFormStillWorks(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := filepath.Join(tmpDir, "clean.go")
	if err := os.WriteFile(target, []byte("package clean\n\nfunc Add(a, b int) int { return a + b }\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	rootCmd.SetArgs([]string{"check-antipattern", "--file", target})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("--file invocation failed: %v", err)
	}
	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["clean"] != true {
		t.Fatalf("clean file not reported clean: %s", buf.String())
	}
}

// TestGatekeeperPlaybooksUsePositionalForm pins the other side of the
// contract: if a playbook rewrites its invocation to some new shape, this
// fails and forces the command contract to be checked with it.
func TestGatekeeperPlaybooksUsePositionalForm(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	for _, playbook := range []string{
		filepath.Join(repoRoot, ".aether", "docs", "command-playbooks", "continue-gates.md"),
		filepath.Join(repoRoot, ".aether", "docs", "command-playbooks", "continue-full.md"),
	} {
		data, err := os.ReadFile(playbook)
		if err != nil {
			t.Fatalf("read %s: %v", playbook, err)
		}
		text := string(data)
		if !strings.Contains(text, `aether check-antipattern "{file_path}"`) {
			t.Errorf("%s no longer uses the positional check-antipattern form this contract was verified against; re-verify the command accepts the new form and update this test", playbook)
		}
	}
}
