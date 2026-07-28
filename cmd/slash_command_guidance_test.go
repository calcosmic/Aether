package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// LOUD-08. `cmd/unblock_cmd.go` told users to "Run /ant-unblock --dispatch" —
// a slash command that existed on no platform. Decision D-02 chose to BUILD the
// wrapper rather than reword the guidance, because the Go side (`aether unblock`
// with --phase/--dispatch/--fixer-mode) was already complete and working.
//
// This test generalises past that one string: any /ant-* slash command that Go
// code tells a user to run must actually exist as a wrapper on both maintained
// platforms. Fixing only the known-bad string would leave the next one free to
// appear the same way.

var slashCommandRe = regexp.MustCompile(`/ant-([a-z][a-z0-9-]*)`)

// Slash commands that are deliberately not backed by a wrapper file.
var slashCommandExemptions = map[string]string{}

func TestSlashCommandGuidancePointsAtRealCommands(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	cmdDir := filepath.Join(root, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		t.Fatalf("read cmd dir: %v", err)
	}

	type reference struct{ file, command string }
	var refs []reference

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(cmdDir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, m := range slashCommandRe.FindAllStringSubmatch(string(raw), -1) {
			refs = append(refs, reference{file: e.Name(), command: m[1]})
		}
	}

	if len(refs) == 0 {
		t.Fatal("found no /ant-* references in cmd/*.go — this test would pass vacuously; the extractor is broken")
	}

	seen := map[string]bool{}
	for _, r := range refs {
		if seen[r.command] || slashCommandExemptions[r.command] != "" {
			continue
		}
		seen[r.command] = true

		for _, platform := range []string{
			filepath.Join(".claude", "commands", "ant"),
			filepath.Join(".opencode", "commands", "ant"),
		} {
			wrapper := filepath.Join(root, platform, r.command+".md")
			if _, err := os.Stat(wrapper); err != nil {
				t.Errorf("%s tells the user to run /ant-%s, but %s does not exist — guidance that names a command available on no platform is a dead end for the user (LOUD-08)",
					r.file, r.command, filepath.Join(platform, r.command+".md"))
			}
		}
	}
}

// The wrapper this phase added must stay wired to the CLI that already works,
// and must not drift from its OpenCode mirror.
func TestUnblockWrapperIsWiredAndAtParity(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	yamlPath := filepath.Join(root, ".aether", "commands", "unblock.yaml")
	yaml, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read unblock.yaml: %v — the YAML source is the head of the generation chain (LOUD-08)", err)
	}
	if !strings.Contains(string(yaml), "aether unblock") {
		t.Error("unblock.yaml does not route to `aether unblock`; the wrapper must delegate to the existing CLI, not reimplement gate recovery")
	}

	claude, err := os.ReadFile(filepath.Join(root, ".claude", "commands", "ant", "unblock.md"))
	if err != nil {
		t.Fatalf("read Claude unblock wrapper: %v", err)
	}
	opencode, err := os.ReadFile(filepath.Join(root, ".opencode", "commands", "ant", "unblock.md"))
	if err != nil {
		t.Fatalf("read OpenCode unblock wrapper: %v", err)
	}
	if string(claude) != string(opencode) {
		t.Error("the Claude and OpenCode unblock wrappers differ; platform wrappers are maintained at parity")
	}

	// The Go command is the thing the wrapper promises. If a flag it documents
	// disappears, the wrapper starts lying.
	target, _, err := rootCmd.Find([]string{"unblock"})
	if err != nil || target == nil || target == rootCmd {
		t.Fatal("`aether unblock` is not a registered command — the wrapper would point at nothing")
	}
	for _, flag := range []string{"dispatch", "fixer-mode", "phase"} {
		if target.Flags().Lookup(flag) == nil {
			t.Errorf("`aether unblock` no longer has --%s, but the wrapper documents it", flag)
		}
	}
}
