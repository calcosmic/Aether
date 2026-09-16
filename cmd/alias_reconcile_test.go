package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aliasReconcileWrapperBody is a minimal but realistic Aether-managed
// wrapper: it carries the generated header the platform sync and pruning
// logic actually key off, so the fixture behaves like the real package
// tree rather than a hand-built shape the runtime could never produce.
func aliasReconcileWrapperBody(yamlName, wrapperName string) string {
	return "<!-- Aether-managed: runtime spec at .aether/commands/" + yamlName + ".yaml. Synced by aether update. -->\n" +
		"---\nname: ant-" + wrapperName + "\ndescription: \"Test command\"\n---\n\n" +
		"Use the Go `aether` CLI as the source of truth.\n\n" +
		"- Execute `AETHER_OUTPUT_MODE=visual aether " + wrapperName + "` directly.\n"
}

// buildAliasReconcilePackageDir builds a minimal Aether package directory
// carrying canonical lifecycle wrappers plus one unrelated public alias. It
// deliberately omits pause-colony/resume-colony: those tokens are accepted
// only by the pre-Cobra migration parser and must never become package files.
func buildAliasReconcilePackageDir(t *testing.T) string {
	t.Helper()
	packageDir := t.TempDir()
	seedCodexSkillSupportFixture(t, packageDir)

	mustMkdirAllForAliasFixture(t, filepath.Join(packageDir, ".aether"))
	mustWriteFileForAliasFixture(t, filepath.Join(packageDir, ".aether", "workers.md"), "# Workers\n")

	claudeCmds := filepath.Join(packageDir, ".claude", "commands", "ant")
	mustMkdirAllForAliasFixture(t, claudeCmds)
	for _, name := range []string{"pause", "resume", "flags"} {
		mustWriteFileForAliasFixture(t, filepath.Join(claudeCmds, name+".md"), aliasReconcileWrapperBody(name, name))
	}

	opencodeCmds := filepath.Join(packageDir, ".opencode", "commands", "ant")
	mustMkdirAllForAliasFixture(t, opencodeCmds)
	for _, name := range []string{"pause", "resume", "flags"} {
		mustWriteFileForAliasFixture(t, filepath.Join(opencodeCmds, name+".md"), aliasReconcileWrapperBody(name, name))
	}

	return packageDir
}

// setUpAliasReconcileProject runs the real install and setup paths (not a
// hand-built directory tree) so damaging a file afterward reproduces the
// exact shape a real project ends up in. Returns homeDir and repoDir.
func setUpAliasReconcileProject(t *testing.T) (homeDir, repoDir string) {
	t.Helper()
	t.Setenv("AETHER_HUB_DIR", "")
	saveGlobals(t)
	resetRootCmd(t)

	packageDir := buildAliasReconcilePackageDir(t)
	homeDir = t.TempDir()
	repoDir = t.TempDir()

	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"install", "--package-dir", packageDir, "--home-dir", homeDir, "--skip-build-binary"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	assertCodexSkillFixtureInstalled(t, packageDir, homeDir)
	resetRootCmd(t)
	buf.Reset()
	stdout = &buf
	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Sanity: the canonical names and unrelated public alias landed where a
	// real project reads them from; parser-only names did not.
	for _, name := range []string{"pause", "resume", "flags"} {
		p := filepath.Join(homeDir, ".claude", "commands", "ant-"+name+".md")
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("install did not create %s: %v", p, err)
		}
	}
	for _, name := range []string{"pause-colony", "resume-colony"} {
		p := filepath.Join(homeDir, ".claude", "commands", "ant-"+name+".md")
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("install unexpectedly created parser-only wrapper %s: %v", p, err)
		}
	}

	return homeDir, repoDir
}

func runAliasReconcileUpdate(t *testing.T, homeDir, repoDir string) map[string]interface{} {
	t.Helper()
	t.Setenv("AETHER_HUB_DIR", "")
	t.Setenv("HOME", homeDir)
	saveGlobals(t)
	resetRootCmd(t)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("chdir to repo: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"update", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("update output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	inner, _ := result["result"].(map[string]interface{})
	if inner == nil {
		t.Fatalf("update output has no result envelope: %s", buf.String())
	}
	return inner
}

// assertCanonicalAliasUpdateReconciliation proves an unrelated, genuinely
// public Cobra alias still participates in update reconciliation after
// lifecycle parser redirects are removed from public metadata.
func assertCanonicalAliasUpdateReconciliation(t *testing.T) {
	homeDir, repoDir := setUpAliasReconcileProject(t)

	missing := filepath.Join(homeDir, ".claude", "commands", "ant-flags.md")
	if err := os.Remove(missing); err != nil {
		t.Fatalf("remove %s: %v", missing, err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be gone before update, stat err = %v", missing, err)
	}

	result := runAliasReconcileUpdate(t, homeDir, repoDir)

	if _, err := os.Stat(missing); err != nil {
		t.Fatalf("expected update --force to restore %s, stat err = %v", missing, err)
	}
	message, _ := result["message"].(string)
	if !strings.Contains(message, "flags") {
		t.Fatalf("expected update message to name public alias flags, got: %q", message)
	}
	if strings.Contains(message, "republish") || strings.Contains(message, "stale") {
		t.Fatalf("alias repair message should read distinctly from the stale-publish signal, got: %q", message)
	}

	repairs, ok := result["alias_wrapper_repairs"].([]interface{})
	if !ok || len(repairs) == 0 {
		t.Fatalf("expected a non-empty alias_wrapper_repairs list, got: %v", result["alias_wrapper_repairs"])
	}
	entry, ok := repairs[0].(map[string]interface{})
	if !ok || entry["alias"] != "flags" {
		t.Fatalf("expected alias_wrapper_repairs[0].alias = flags, got: %v", repairs[0])
	}

	stalePublish, _ := result["stale_publish"].(map[string]interface{})
	staleMessage, _ := stalePublish["message"].(string)
	if strings.Contains(staleMessage, "flags") {
		t.Fatalf("stale_publish message should not carry the alias repair wording, got: %q", staleMessage)
	}
}

// TestUpdateDoesNotRestoreParserOnlyAlias proves the hidden argv normalizer
// is not consulted as public alias metadata and therefore cannot manufacture
// a wrapper or repair report.
func TestUpdateDoesNotRestoreParserOnlyAlias(t *testing.T) {
	homeDir, repoDir := setUpAliasReconcileProject(t)

	result := runAliasReconcileUpdate(t, homeDir, repoDir)

	message, _ := result["message"].(string)
	if strings.Contains(message, "Restored missing command") {
		t.Fatalf("expected no repair wording when nothing was missing, got: %q", message)
	}
	if repairs, ok := result["alias_wrapper_repairs"].([]interface{}); ok && len(repairs) != 0 {
		t.Fatalf("expected an empty alias_wrapper_repairs list, got: %v", repairs)
	}

	for _, name := range []string{"pause-colony", "resume-colony"} {
		for _, p := range retiredAliasPlatformPaths(homeDir, name) {
			if _, err := os.Stat(p); !os.IsNotExist(err) {
				t.Fatalf("update unexpectedly created parser-only wrapper %s: %v", p, err)
			}
		}
	}
}

func retiredAliasPlatformPaths(homeDir, name string) []string {
	return []string{
		filepath.Join(homeDir, ".claude", "commands", "ant-"+name+".md"),
		filepath.Join(homeDir, ".claude", "commands", "ant", name+".md"),
		filepath.Join(homeDir, ".opencode", "command", name+".md"),
		filepath.Join(homeDir, ".config", "opencode", "commands", "ant", name+".md"),
	}
}

func seedRetiredAliasPlatformPaths(t *testing.T, homeDir, body string) map[string][]byte {
	t.Helper()
	written := make(map[string][]byte)
	for _, name := range []string{"pause-colony", "resume-colony"} {
		for _, path := range retiredAliasPlatformPaths(homeDir, name) {
			mustMkdirAllForAliasFixture(t, filepath.Dir(path))
			content := []byte(body)
			if body == "managed" {
				canonical := strings.TrimSuffix(name, "-colony")
				content = []byte(aliasReconcileWrapperBody(canonical, name))
			}
			mustWriteFileForAliasFixture(t, path, string(content))
			written[path] = content
		}
	}
	return written
}

// TestRetiredLifecycleAliasPruning199 exercises the real install -> hub ->
// update pipeline. Stale wrappers are removed only when their generated header
// proves Aether ownership; byte-identical custom commands at the same paths are
// left alone.
func TestRetiredLifecycleAliasPruning199(t *testing.T) {
	t.Run("managed_wrappers_are_removed_from_every_platform_path", func(t *testing.T) {
		homeDir, repoDir := setUpAliasReconcileProject(t)
		written := seedRetiredAliasPlatformPaths(t, homeDir, "managed")

		runAliasReconcileUpdate(t, homeDir, repoDir)

		for path := range written {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("managed retired wrapper survived at %s: %v", path, err)
			}
		}
	})

	t.Run("unmanaged_same_name_commands_survive_byte_for_byte", func(t *testing.T) {
		homeDir, repoDir := setUpAliasReconcileProject(t)
		written := seedRetiredAliasPlatformPaths(t, homeDir, "# My custom command\n")

		runAliasReconcileUpdate(t, homeDir, repoDir)

		for path, want := range written {
			got, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("custom retired-name command was removed at %s: %v", path, err)
				continue
			}
			if !bytes.Equal(got, want) {
				t.Errorf("custom retired-name command changed at %s: got %q want %q", path, got, want)
			}
		}
	})
}
