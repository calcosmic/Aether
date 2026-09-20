package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestVerificationCommandsSkipTscWithoutTsconfig locks the 1.0.47 acceptance
// fix: a plain-JavaScript repo (package.json, no tsconfig) must not default
// the types check to `npx tsc --noEmit` — tsc without a tsconfig prints usage
// help and exits 1, failing verification in every JS-only project.
func TestVerificationCommandsSkipTscWithoutTsconfig(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"js-only"}`), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	commands := resolveCodexVerificationCommands(root)
	if commands.Type != "" {
		t.Fatalf("JS-only repo defaulted a type check: %q", commands.Type)
	}

	if err := os.WriteFile(filepath.Join(root, "tsconfig.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write tsconfig.json: %v", err)
	}
	commands = resolveCodexVerificationCommands(root)
	if commands.Type != "npx tsc --noEmit" {
		t.Fatalf("TypeScript repo missing type check default: %q", commands.Type)
	}
}

// TestResolveHostBoundaryWatcherPrefersBuildWatcher locks the 1.0.47
// acceptance fix: when continue would auto-skip its watcher for a
// wrapper-mediated build, a real passing watcher result recorded in the build
// packet is trusted instead — otherwise criteria with a `watcher` check can
// never pass on the primary wrapper path.
func TestResolveHostBoundaryWatcherPrefersBuildWatcher(t *testing.T) {
	buildWatcher := codexWatcherVerification{Present: true, Passed: true, Status: "completed", Worker: "Keen-6", Summary: "watcher verdict pass"}
	got := resolveHostBoundaryWatcher(buildWatcher, "skip summary")
	if got.Worker != "Keen-6" || got.Status != "completed" {
		t.Fatalf("build watcher not trusted: %+v", got)
	}

	// No usable build watcher: fall back to the skip placeholder.
	got = resolveHostBoundaryWatcher(codexWatcherVerification{}, "skip summary")
	if got.Status != "skipped" || got.Worker != "auto-skip" || !got.Passed {
		t.Fatalf("skip placeholder wrong: %+v", got)
	}

	// A skipped build watcher must not masquerade as an executed review.
	got = resolveHostBoundaryWatcher(codexWatcherVerification{Present: true, Passed: true, Status: "skipped", Worker: "auto-skip"}, "skip summary")
	if got.Status != "skipped" {
		t.Fatalf("skipped build watcher not preserved as skip: %+v", got)
	}
}

// TestTasklessPhaseAdvancesOnVerifiedClaims locks the 1.0.47 acceptance fix:
// a phase inserted via `aether phase-insert` has no task list, and its
// implementation evidence is the build's verified claims — requiring
// task-bound evidence made every inserted phase permanently unadvanceable.
func TestTasklessPhaseAdvancesOnVerifiedClaims(t *testing.T) {
	if continueTasksSupportAdvancement(nil, true) != true {
		t.Fatal("taskless phase with verified claims cannot advance")
	}
	if continueTasksSupportAdvancement(nil, false) != false {
		t.Fatal("taskless phase without verified claims must not advance")
	}
}

// TestReconciledTaskAdvancesWhenVerified locks the H-04 fix, extended by
// 193-04 (FLOOR-03, closes the 2026-08-01 folded todo): the runtime's own
// recovery hint is `--reconcile-task <id>`, so a reconciled task with a
// passing deterministic floor (task.Verified -- verification.ChecksPassed,
// which already folds criterion evidence in) must count as advancement
// evidence, even when the generic claimsSatisfied flag is false — an
// operator reconciling work done outside the pipeline structurally has no
// claims file to satisfy, which is precisely why the finalize lane
// deadlocked before this fix. "Reconcile is not a bypass" still holds
// through task.Verified itself: a reconciled task whose deterministic floor
// (and therefore verification.ChecksPassed) genuinely failed still blocks,
// and claimsSatisfied still gates every UNRECONCILED task via the
// artifactEvidenceTrusted path in classifyContinueTaskAssessment, untouched
// by this change.
func TestReconciledTaskAdvancesWhenVerified(t *testing.T) {
	reconciledVerified := []codexContinueTaskAssessment{{TaskID: "2.2", Outcome: "manually_reconciled", Verified: true}}
	if !continueTasksSupportAdvancement(reconciledVerified, true) {
		t.Fatal("reconciled+verified task blocked advancement")
	}
	reconciledUnverified := []codexContinueTaskAssessment{{TaskID: "2.2", Outcome: "manually_reconciled", Verified: false}}
	if continueTasksSupportAdvancement(reconciledUnverified, true) {
		t.Fatal("reconciled task without verification advanced")
	}
	// FLOOR-03: a reconciled+verified task now advances even when the
	// generic claimsSatisfied flag is false -- claims absence alone is no
	// longer sufficient to block a reconciled task (only a genuinely failed
	// deterministic floor, i.e. Verified:false, still blocks one).
	if !continueTasksSupportAdvancement(reconciledVerified, false) {
		t.Fatal("reconciled+verified task should advance regardless of claimsSatisfied (FLOOR-03)")
	}
	// "Not a bypass" still holds: reconciled but NOT verified (deterministic
	// floor failed) blocks regardless of claimsSatisfied.
	if continueTasksSupportAdvancement(reconciledUnverified, false) {
		t.Fatal("reconciled task with a failed deterministic floor must still block")
	}
}
