package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCanonicalAlias keeps two contracts separate: real public aliases remain
// Cobra metadata and participate in wrapper reconciliation, while the bounded
// lifecycle migration tokens remain pre-Cobra parser plumbing only.
func TestCanonicalAlias(t *testing.T) {
	t.Run("metadata", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		canonical, _, err := rootCmd.Find([]string{"flag-list"})
		if err != nil {
			t.Fatalf("resolve flag-list: %v", err)
		}
		alias, _, err := rootCmd.Find([]string{"flags"})
		if err != nil {
			t.Fatalf("resolve public alias flags: %v", err)
		}
		if canonical != alias {
			t.Fatalf("flag-list and flags resolve to different commands (%p vs %p)", canonical, alias)
		}

		for _, retired := range []string{"pause-colony", "resume-colony"} {
			if wrapperCommandNames[retired] {
				t.Errorf("parser-only token %q remains in the public wrapper map", retired)
			}
			for _, command := range rootCmd.Commands() {
				if command.Name() == retired {
					t.Errorf("parser-only token %q is a Cobra command", retired)
				}
				for _, publicAlias := range command.Aliases {
					if publicAlias == retired {
						t.Errorf("parser-only token %q is a Cobra alias of %q", retired, command.Name())
					}
				}
			}
		}

		var sawFlags bool
		for _, surface := range declaredAliasSurfaces(t.TempDir()) {
			if surface.Alias == "flags" {
				sawFlags = true
			}
			if surface.Alias == "pause-colony" || surface.Alias == "resume-colony" {
				t.Errorf("parser-only token leaked into alias reconciliation: %+v", surface)
			}
		}
		if !sawFlags {
			t.Fatal("public alias flags is missing from alias reconciliation")
		}
	})

	t.Run("update_reconciliation", assertCanonicalAliasUpdateReconciliation)
}

// aliasCheckFixtureDir builds a minimal, realistic .aether/commands +
// .claude/commands/ant + .claude/commands (flat) + .opencode/commands/ant
// tree: one canonical command declaring one alias, with wrapper files
// present at all three locations for both names. Individual tests mutate
// this fixture to prove the checker fails when it should.
func aliasCheckFixtureDir(t *testing.T, canonicalName, aliasName string) string {
	t.Helper()
	root := t.TempDir()

	mustMkdirAllForAliasFixture(t, filepath.Join(root, ".aether", "commands"))
	mustMkdirAllForAliasFixture(t, filepath.Join(root, ".claude", "commands", "ant"))
	mustMkdirAllForAliasFixture(t, filepath.Join(root, ".opencode", "commands", "ant"))

	yamlContent := fmt.Sprintf(
		"name: ant-%s\ndescription: \"Test command\"\nsource_of_truth: \"Use the Go `aether` CLI as the source of truth.\"\nruntime:\n  command: \"AETHER_OUTPUT_MODE=visual aether %s $ARGUMENTS\"\naliases:\n  - %s\nguardrails:\n  - \"Test guardrail.\"\n",
		canonicalName, canonicalName, aliasName,
	)
	mustWriteFileForAliasFixture(t, filepath.Join(root, ".aether", "commands", canonicalName+".yaml"), yamlContent)

	for _, name := range []string{canonicalName, aliasName} {
		body := fmt.Sprintf(
			"<!-- Aether-managed: runtime spec at .aether/commands/%s.yaml. Synced by aether update. -->\n---\nname: ant-%s\ndescription: \"Test command\"\n---\n\nUse the Go `aether` CLI as the source of truth.\n\n- Execute `AETHER_OUTPUT_MODE=visual aether %s` directly.\n",
			canonicalName, name, canonicalName,
		)
		mustWriteFileForAliasFixture(t, filepath.Join(root, ".claude", "commands", "ant", name+".md"), body)
		mustWriteFileForAliasFixture(t, filepath.Join(root, ".claude", "commands", "ant-"+name+".md"), body)
		mustWriteFileForAliasFixture(t, filepath.Join(root, ".opencode", "commands", "ant", name+".md"), body)
	}

	return root
}

func mustMkdirAllForAliasFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func mustWriteFileForAliasFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// aliasWrapperFixtureLocations returns the three hand-maintained wrapper
// paths for a given command name, relative to a fixture root -- mirroring
// aliasCheckFixtureDir's layout.
func aliasWrapperFixtureLocations(root, name string) []string {
	return []string{
		filepath.Join(root, ".claude", "commands", "ant", name+".md"),
		filepath.Join(root, ".claude", "commands", "ant-"+name+".md"),
		filepath.Join(root, ".opencode", "commands", "ant", name+".md"),
	}
}

// TestDeclaredAliasesHaveWrappersOnEveryPlatform proves the checker requires
// all three copies of a declared alias's wrapper, and fails by naming the
// missing one when any single copy is removed.
func TestDeclaredAliasesHaveWrappersOnEveryPlatform(t *testing.T) {
	t.Run("passes_when_all_three_copies_present", func(t *testing.T) {
		root := aliasCheckFixtureDir(t, "foo", "foo-bar")
		_, issues := checkGeneratedCommandSurfaces(root)
		if len(issues) != 0 {
			t.Fatalf("expected no issues with all three alias wrapper copies present, got: %+v", issues)
		}
	})

	for i, label := range []string{"nested_claude", "flat_claude", "opencode"} {
		i, label := i, label
		t.Run("fails_when_"+label+"_copy_removed", func(t *testing.T) {
			root := aliasCheckFixtureDir(t, "foo", "foo-bar")
			paths := aliasWrapperFixtureLocations(root, "foo-bar")
			removed := paths[i]
			if err := os.Remove(removed); err != nil {
				t.Fatalf("remove %s: %v", removed, err)
			}

			_, issues := checkGeneratedCommandSurfaces(root)
			if len(issues) == 0 {
				t.Fatalf("expected an issue after removing %s, got none", removed)
			}
			var found bool
			for _, issue := range issues {
				if strings.Contains(issue.Path, "foo-bar") && issue.Message == "declared alias has no wrapper at this location" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected an issue naming the missing foo-bar wrapper, got: %+v", issues)
			}
		})
	}
}

// TestAliasWrapperNeedsADeclaration covers both directions of the
// enforcement: a declared alias with a missing wrapper is reported, and a
// wrapper that looks like an alias but was never declared anywhere is still
// reported too.
func TestAliasWrapperNeedsADeclaration(t *testing.T) {
	t.Run("declared_alias_missing_wrapper_is_reported", func(t *testing.T) {
		root := aliasCheckFixtureDir(t, "foo", "foo-bar")
		missing := filepath.Join(root, ".claude", "commands", "ant", "foo-bar.md")
		if err := os.Remove(missing); err != nil {
			t.Fatalf("remove %s: %v", missing, err)
		}

		_, issues := checkGeneratedCommandSurfaces(root)
		if len(issues) == 0 {
			t.Fatal("expected an issue for a declared alias missing its wrapper")
		}
	})

	t.Run("undeclared_alias_wrapper_is_reported", func(t *testing.T) {
		root := aliasCheckFixtureDir(t, "foo", "foo-bar")

		// A wrapper that exists in both scanned directories but is declared
		// by nothing -- no YAML file named "orphan-alias" and no canonical
		// command's `aliases:` field names it.
		body := "<!-- Aether-managed: runtime spec at .aether/commands/orphan-alias.yaml. Synced by aether update. -->\n---\nname: ant-orphan-alias\ndescription: \"Test\"\n---\n\nUse the Go `aether` CLI as the source of truth.\n"
		mustWriteFileForAliasFixture(t, filepath.Join(root, ".claude", "commands", "ant", "orphan-alias.md"), body)
		mustWriteFileForAliasFixture(t, filepath.Join(root, ".opencode", "commands", "ant", "orphan-alias.md"), body)

		_, issues := checkGeneratedCommandSurfaces(root)
		var found bool
		for _, issue := range issues {
			if strings.Contains(issue.Path, "orphan-alias") && issue.Message == "generated wrapper has no matching YAML source" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected an issue for the undeclared orphan-alias wrapper, got: %+v", issues)
		}
	})

	t.Run("declared_alias_with_all_wrappers_present_has_no_issue", func(t *testing.T) {
		root := aliasCheckFixtureDir(t, "foo", "foo-bar")
		_, issues := checkGeneratedCommandSurfaces(root)
		if len(issues) != 0 {
			t.Fatalf("declared alias with its wrapper present should have no issue, got: %+v", issues)
		}
	})
}
