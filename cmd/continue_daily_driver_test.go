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
