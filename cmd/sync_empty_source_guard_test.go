package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A sync whose source yields zero files must never mass-delete a populated
// destination. This is the 2026-09-12 incident: an update/lay-eggs pass read
// the hub mid-publish (or corrupted), saw an empty source, and deleted all 64
// user-level ant-*.md commands. Cleanup must refuse and say why.
func TestSyncCleanupRefusesEmptySourceAgainstPopulatedDest(t *testing.T) {
	src := t.TempDir()  // exists but empty — the mid-publish shape
	dest := t.TempDir() // the user's populated command dir

	for _, name := range []string{"ant-build.md", "ant-status.md", "ant-plan.md"} {
		if err := os.WriteFile(filepath.Join(dest, name), []byte("wrapper"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	res := syncDir(src, dest, syncOptions{cleanup: true})

	if len(res.removed) != 0 {
		t.Fatalf("cleanup removed %d files from a populated destination despite an empty source: %v", len(res.removed), res.removed)
	}
	survivors, _ := filepath.Glob(filepath.Join(dest, "ant-*.md"))
	if len(survivors) != 3 {
		t.Fatalf("expected all 3 destination files to survive, found %d", len(survivors))
	}
	if len(res.errors) == 0 || !strings.Contains(strings.Join(res.errors, " "), "refusing") {
		t.Fatalf("expected an explicit refusal message in errors, got: %v", res.errors)
	}
}

// The hub-facing sync has its own cleanup loop and needs the same refusal.
func TestSyncToHubRefusesEmptySourceAgainstPopulatedDest(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	for _, name := range []string{"build.yaml", "status.yaml"} {
		if err := os.WriteFile(filepath.Join(dest, name), []byte("spec"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	res := syncDirToHub(src, dest)

	if len(res.removed) != 0 {
		t.Fatalf("hub cleanup removed %d files despite an empty source: %v", len(res.removed), res.removed)
	}
	if len(res.errors) == 0 || !strings.Contains(strings.Join(res.errors, " "), "refusing") {
		t.Fatalf("expected an explicit refusal message in errors, got: %v", res.errors)
	}
}

// The transactional maintenance sync (the 199-18 path that update/lay-eggs
// actually run) is the code that deleted all 64 user-level commands on
// 2026-09-12 when it read the hub mid-publish. An empty source directory must
// refuse to prune a populated destination, loudly.
func TestMaintenanceSyncRefusesEmptySourceAgainstManagedDest(t *testing.T) {
	fixture := newMaintenanceMutation199Fixture(t)
	source := filepath.Join(fixture.hub, "system", "commands", "claude")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"ant-build.md", "ant-status.md"} {
		writeMaintenanceMutation199File(t, filepath.Join(fixture.claude, name),
			[]byte("<!-- Aether-managed: runtime spec at .aether/commands/x.yaml. Synced by aether update. -->\nwrapper\n"))
	}

	plan := fixture.plan("empty-source-refusal")
	err := appendMaintenanceSyncTargets(&plan, maintenanceSyncSpec{
		Root:            lifecycleTransactionRootClaudeHome,
		SourceDir:       source,
		DestinationBase: ".",
		Options: syncOptions{
			cleanup:        true,
			cleanupInclude: isManagedFlatClaudeCommandPath,
		},
	})
	if err == nil {
		t.Fatalf("expected refusal error for empty source against populated destination, got nil (targets: %#v)", plan.Targets)
	}
	if !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("refusal error should say why, got: %v", err)
	}
	for _, target := range plan.Targets {
		if target.Action == lifecycleTransactionRemove {
			t.Fatalf("a remove target was planned despite the refusal: %#v", target)
		}
	}
}

// The destination side of a cleanup pass walks the user's repo, where
// symlinks (node_modules/.bin, etc.) are normal and can never be owned
// managed files. They must be skipped, not abort the whole update
// (2026-09-12: CosmicDashboard update failed on .aether/ts-host/node_modules).
func TestMaintenanceCleanupSkipsDestinationSymlinks(t *testing.T) {
	fixture := newMaintenanceMutation199Fixture(t)
	source := filepath.Join(fixture.hub, "system", "commands", "claude")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMaintenanceMutation199File(t, filepath.Join(source, "ant-build.md"),
		[]byte("<!-- Aether-managed: runtime spec at .aether/commands/build.yaml. Synced by aether update. -->\nlive\n"))

	writeMaintenanceMutation199File(t, filepath.Join(fixture.claude, "ant-stale.md"),
		[]byte("<!-- Aether-managed: runtime spec at .aether/commands/stale.yaml. Synced by aether update. -->\nstale\n"))
	binDir := filepath.Join(fixture.claude, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/bin/true", filepath.Join(binDir, "esbuild")); err != nil {
		t.Fatal(err)
	}

	plan := fixture.plan("cleanup-skips-symlinks")
	if err := appendMaintenanceSyncTargets(&plan, maintenanceSyncSpec{
		Root:            lifecycleTransactionRootClaudeHome,
		SourceDir:       source,
		DestinationBase: ".",
		Options: syncOptions{
			cleanup:        true,
			cleanupInclude: isManagedFlatClaudeCommandPath,
		},
	}); err != nil {
		t.Fatalf("cleanup planning aborted on a destination symlink it does not own: %v", err)
	}

	prunedStale := false
	for _, target := range plan.Targets {
		if target.Action == lifecycleTransactionRemove {
			if target.RelativeTarget == "ant-stale.md" {
				prunedStale = true
			} else {
				t.Fatalf("cleanup planned removal beyond the stale wrapper: %#v", target)
			}
		}
	}
	if !prunedStale {
		t.Fatalf("stale managed wrapper was not pruned: %#v", plan.Targets)
	}
}
