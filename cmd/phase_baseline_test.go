package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func gitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git unavailable (%v): %s", err, out)
		}
	}
}

func gitCommitAll(t *testing.T, dir, message string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-qm", message}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

// TestPhaseBaselineNamesPriorPhaseWorkAsPreExisting is the WP-7 gate, built
// from the real v1.0.47 failure: a phase-4 watcher ran `git status`, saw
// phase 1-3's uncommitted work, and returned FAIL for scope creep. The
// baseline must name that work as pre-existing so the verdict is judged
// against the phase's own changes.
func TestPhaseBaselineNamesPriorPhaseWorkAsPreExisting(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	if err := os.WriteFile(filepath.Join(root, "seed.txt"), []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	gitCommitAll(t, root, "seed")

	// Prior phases' verified work, uncommitted — exactly the shape that
	// triggered the false scope-creep verdicts.
	if err := os.WriteFile(filepath.Join(root, "strings.js"), []byte("export function slugify() {}\n"), 0644); err != nil {
		t.Fatalf("write prior work: %v", err)
	}
	// Aether's own state must never be reported as product scope.
	if err := os.MkdirAll(filepath.Join(root, ".aether", "data"), 0755); err != nil {
		t.Fatalf("mkdir .aether/data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".aether", "data", "COLONY_STATE.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("write colony state: %v", err)
	}

	baseline := capturePhaseBaseline(root)
	if baseline.HeadSHA == "" {
		t.Fatal("baseline captured no HEAD sha")
	}
	if !containsPath(baseline.DirtyPaths, "strings.js") {
		t.Fatalf("baseline missed pre-existing product change: %v", baseline.DirtyPaths)
	}
	for _, p := range baseline.DirtyPaths {
		if strings.HasPrefix(p, ".aether/") {
			t.Errorf("baseline reported Aether's own state as product scope: %s", p)
		}
	}

	section := renderPhaseBaselineSection(baseline)
	for _, want := range []string{
		"ALREADY modified before this phase began",
		"strings.js",
		"do not commit between phases",
	} {
		if !strings.Contains(section, want) {
			t.Fatalf("baseline section missing %q:\n%s", want, section)
		}
	}
}

// A clean tree must say so affirmatively: with no pre-existing changes, every
// uncommitted change really does belong to this phase.
func TestPhaseBaselineOnCleanTreeClaimsEverythingIsThisPhase(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	if err := os.WriteFile(filepath.Join(root, "seed.txt"), []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	gitCommitAll(t, root, "seed")

	section := renderPhaseBaselineSection(capturePhaseBaseline(root))
	if !strings.Contains(section, "No uncommitted product changes existed") {
		t.Fatalf("clean-tree baseline should say so:\n%s", section)
	}
}

// No git, no crash: the baseline degrades to an explanatory note.
func TestPhaseBaselineWithoutGitDoesNotFail(t *testing.T) {
	baseline := capturePhaseBaseline(t.TempDir())
	if baseline.Unavailable == "" {
		t.Fatalf("expected an explanatory note for a non-git workspace, got %+v", baseline)
	}
	if section := renderPhaseBaselineSection(baseline); section != "" {
		t.Fatalf("a workspace with no baseline should render nothing, got:\n%s", section)
	}
}

// The section must reach verifying castes and stay out of builder briefs.
func TestVerifierBriefCarriesPhaseBaseline(t *testing.T) {
	saveGlobals(t)
	root := t.TempDir()
	gitInit(t, root)
	if err := os.WriteFile(filepath.Join(root, "seed.txt"), []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	gitCommitAll(t, root, "seed")
	if err := os.WriteFile(filepath.Join(root, "prior.js"), []byte("// prior phase work\n"), 0644); err != nil {
		t.Fatalf("write prior work: %v", err)
	}

	phase := colony.Phase{ID: 4, Name: "Document usage", Description: "Write the README"}
	watcher := codexBuildDispatch{Caste: "watcher", Name: "Keen-6", Task: "Independent verification before advancement"}
	builder := codexBuildDispatch{Caste: "builder", Name: "Mason-1", Task: "Write README.md"}

	// includeSteeringSections=true: unrelated to what this test checks (the
	// phase-baseline section, which always renders regardless of the flag --
	// see composeBuildManifestBrief's doc comment); true matches the
	// self-contained/default composition shape.
	watcherBrief := composeBuildManifestBrief(root, phase, watcher, time.Now().UTC(), true)
	if !strings.Contains(watcherBrief, "## Phase Baseline") || !strings.Contains(watcherBrief, "prior.js") {
		t.Fatalf("watcher brief lacks the phase baseline:\n%s", watcherBrief)
	}

	builderBrief := composeBuildManifestBrief(root, phase, builder, time.Now().UTC(), true)
	if strings.Contains(builderBrief, "## Phase Baseline") {
		t.Fatalf("builder brief should not carry the baseline; it only dilutes the task:\n%s", builderBrief)
	}
}

func containsPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}
