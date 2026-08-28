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
// carrying the pause/pause-colony wrapper pair on both platforms -- the
// real shape `aether install` reads, not a hand-built stand-in.
func buildAliasReconcilePackageDir(t *testing.T) string {
	t.Helper()
	packageDir := t.TempDir()

	mustMkdirAllForAliasFixture(t, filepath.Join(packageDir, ".aether"))
	mustWriteFileForAliasFixture(t, filepath.Join(packageDir, ".aether", "workers.md"), "# Workers\n")

	claudeCmds := filepath.Join(packageDir, ".claude", "commands", "ant")
	mustMkdirAllForAliasFixture(t, claudeCmds)
	mustWriteFileForAliasFixture(t, filepath.Join(claudeCmds, "pause.md"), aliasReconcileWrapperBody("pause", "pause"))
	mustWriteFileForAliasFixture(t, filepath.Join(claudeCmds, "pause-colony.md"), aliasReconcileWrapperBody("pause", "pause-colony"))

	opencodeCmds := filepath.Join(packageDir, ".opencode", "commands", "ant")
	mustMkdirAllForAliasFixture(t, opencodeCmds)
	mustWriteFileForAliasFixture(t, filepath.Join(opencodeCmds, "pause.md"), aliasReconcileWrapperBody("pause", "pause"))
	mustWriteFileForAliasFixture(t, filepath.Join(opencodeCmds, "pause-colony.md"), aliasReconcileWrapperBody("pause", "pause-colony"))

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

	resetRootCmd(t)
	buf.Reset()
	stdout = &buf
	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Sanity: both names landed where a real project reads them from.
	for _, name := range []string{"pause", "pause-colony"} {
		p := filepath.Join(homeDir, ".claude", "commands", "ant-"+name+".md")
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("install did not create %s: %v", p, err)
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

// TestUpdateRestoresAMissingAliasWrapper proves the missing wrapper for a
// declared alias comes back after `aether update --force`, over a project
// built by the real install path and then damaged.
func TestUpdateRestoresAMissingAliasWrapper(t *testing.T) {
	homeDir, repoDir := setUpAliasReconcileProject(t)

	missing := filepath.Join(homeDir, ".claude", "commands", "ant-pause-colony.md")
	if err := os.Remove(missing); err != nil {
		t.Fatalf("remove %s: %v", missing, err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be gone before update, stat err = %v", missing, err)
	}

	runAliasReconcileUpdate(t, homeDir, repoDir)

	if _, err := os.Stat(missing); err != nil {
		t.Fatalf("expected update --force to restore %s, stat err = %v", missing, err)
	}
}

// TestUpdateReportsTheAliasRepairByName asserts the command name appears in
// the reported output -- not only a larger copied-file count -- and that
// the wording is distinguishable from the unrelated stale-publish signal.
func TestUpdateReportsTheAliasRepairByName(t *testing.T) {
	homeDir, repoDir := setUpAliasReconcileProject(t)

	missing := filepath.Join(homeDir, ".claude", "commands", "ant-pause-colony.md")
	if err := os.Remove(missing); err != nil {
		t.Fatalf("remove %s: %v", missing, err)
	}

	result := runAliasReconcileUpdate(t, homeDir, repoDir)

	message, _ := result["message"].(string)
	if !strings.Contains(message, "pause-colony") {
		t.Fatalf("expected update message to name pause-colony, got: %q", message)
	}
	if strings.Contains(message, "republish") || strings.Contains(message, "stale") {
		t.Fatalf("alias repair message should read distinctly from the stale-publish signal, got: %q", message)
	}

	repairs, ok := result["alias_wrapper_repairs"].([]interface{})
	if !ok || len(repairs) == 0 {
		t.Fatalf("expected a non-empty alias_wrapper_repairs list, got: %v", result["alias_wrapper_repairs"])
	}
	entry, ok := repairs[0].(map[string]interface{})
	if !ok || entry["alias"] != "pause-colony" {
		t.Fatalf("expected alias_wrapper_repairs[0].alias = pause-colony, got: %v", repairs[0])
	}

	stalePublish, _ := result["stale_publish"].(map[string]interface{})
	staleMessage, _ := stalePublish["message"].(string)
	if strings.Contains(staleMessage, "pause-colony") {
		t.Fatalf("stale_publish message should not carry the alias repair wording, got: %q", staleMessage)
	}
}

// TestUpdateReportsNoRepairWhenNothingIsMissing asserts an update that finds
// nothing missing reports no repair and changes nothing -- an update that
// reports a repair every time it runs is noise, and noise is how a real
// repair gets ignored.
func TestUpdateReportsNoRepairWhenNothingIsMissing(t *testing.T) {
	homeDir, repoDir := setUpAliasReconcileProject(t)

	result := runAliasReconcileUpdate(t, homeDir, repoDir)

	message, _ := result["message"].(string)
	if strings.Contains(message, "Restored missing command") {
		t.Fatalf("expected no repair wording when nothing was missing, got: %q", message)
	}
	if repairs, ok := result["alias_wrapper_repairs"].([]interface{}); ok && len(repairs) != 0 {
		t.Fatalf("expected an empty alias_wrapper_repairs list, got: %v", repairs)
	}

	for _, name := range []string{"pause", "pause-colony"} {
		p := filepath.Join(homeDir, ".claude", "commands", "ant-"+name+".md")
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s to remain present: %v", p, err)
		}
	}
}
