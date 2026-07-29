package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// setupSurveyStalenessTest creates a temp colony root with an initialized
// store and points the store global at it. Returns the data dir and the repo
// root (two levels above the data dir, matching the store.BasePath()
// convention used throughout cmd/).
func setupSurveyStalenessTest(t *testing.T) (dataDir string, root string) {
	t.Helper()
	root = t.TempDir()
	dataDir = filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	store = s
	return dataDir, root
}

// gitInitSurveyTestRepo initializes a git repo at root with a test identity
// configured, skipping the test if git itself is unavailable.
func gitInitSurveyTestRepo(t *testing.T, root string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@aether.invalid")
	run("config", "user.name", "Aether Test")
}

// gitCommitsSurveyTestRepo creates n empty commits in root, each dated "now".
func gitCommitsSurveyTestRepo(t *testing.T, root string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		cmd := exec.Command("git", "commit", "--allow-empty", "-m", "commit "+strconv.Itoa(i))
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git commit failed: %v\n%s", err, out)
		}
	}
}

// TestSurveyStaleness pins the five behaviors surveyStalenessNotice must
// have: never-surveyed naming the fix, a commit count once surveyed, loud
// wording at/above the threshold and quiet wording below it, a malformed
// timestamp never reaching git, and a git failure degrading to quiet wording
// rather than an error or a panic.
func TestSurveyStaleness(t *testing.T) {
	t.Run("never surveyed names the fixing command", func(t *testing.T) {
		saveGlobals(t)
		setupSurveyStalenessTest(t)

		if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{}); err != nil {
			t.Fatalf("save state: %v", err)
		}

		notice := surveyStalenessNotice()
		if notice == "" {
			t.Fatal("expected non-empty notice for a never-surveyed state")
		}
		if !strings.Contains(notice, "never") {
			t.Errorf("notice does not say the territory has never been surveyed:\n%s", notice)
		}
		if !strings.Contains(notice, "/ant-colonize") {
			t.Errorf("notice does not name the command that fixes it:\n%s", notice)
		}
	})

	t.Run("surveyed produces a notice containing the commit count", func(t *testing.T) {
		saveGlobals(t)
		_, root := setupSurveyStalenessTest(t)
		gitInitSurveyTestRepo(t, root)

		surveyedAt := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
		if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{TerritorySurveyed: &surveyedAt}); err != nil {
			t.Fatalf("save state: %v", err)
		}
		gitCommitsSurveyTestRepo(t, root, 3)

		notice := surveyStalenessNotice()
		if notice == "" {
			t.Fatal("expected non-empty notice when surveyed")
		}
		if !strings.Contains(notice, "3") {
			t.Errorf("notice does not contain the commit count:\n%s", notice)
		}
	})

	t.Run("threshold boundary switches quiet to loud wording", func(t *testing.T) {
		saveGlobals(t)
		_, root := setupSurveyStalenessTest(t)
		gitInitSurveyTestRepo(t, root)

		surveyedAt := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
		if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{TerritorySurveyed: &surveyedAt}); err != nil {
			t.Fatalf("save state: %v", err)
		}

		gitCommitsSurveyTestRepo(t, root, surveyStaleCommitThreshold-1)
		quietNotice := surveyStalenessNotice()
		if quietNotice == "" {
			t.Fatal("expected non-empty quiet notice below the threshold")
		}
		if strings.Contains(strings.ToUpper(quietNotice), "STALE") {
			t.Errorf("notice below threshold should be quiet age wording, got loud wording:\n%s", quietNotice)
		}

		gitCommitsSurveyTestRepo(t, root, 1) // now at surveyStaleCommitThreshold total
		loudNotice := surveyStalenessNotice()
		if !strings.Contains(strings.ToUpper(loudNotice), "STALE") {
			t.Errorf("notice at threshold should be loud stale wording, got:\n%s", loudNotice)
		}
	})

	t.Run("malformed timestamp produces empty notice and no git invocation", func(t *testing.T) {
		saveGlobals(t)
		setupSurveyStalenessTest(t)
		// Deliberately no git repo here. Every branch past the RFC3339 parse
		// step always returns non-empty text (either the counted line or the
		// "Commit count since survey unavailable" fallback) — so an empty
		// result here is only possible if the function returned before ever
		// reaching the git-calling branch, which is the proof this behavior
		// requires.
		bad := "not-a-valid-timestamp"
		if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{TerritorySurveyed: &bad}); err != nil {
			t.Fatalf("save state: %v", err)
		}

		notice := surveyStalenessNotice()
		if notice != "" {
			t.Errorf("expected empty notice for a malformed timestamp (proves no git invocation), got:\n%s", notice)
		}
	})

	t.Run("outside a git repo degrades to quiet wording without a count", func(t *testing.T) {
		saveGlobals(t)
		setupSurveyStalenessTest(t)
		// No git init here — root is not a git repository, so the
		// rev-list call inside surveyStalenessNotice must fail gracefully.

		surveyedAt := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
		if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{TerritorySurveyed: &surveyedAt}); err != nil {
			t.Fatalf("save state: %v", err)
		}

		notice := surveyStalenessNotice()
		if notice == "" {
			t.Fatal("expected non-empty quiet notice outside a git repo, not a panic or an error")
		}
		if strings.Contains(strings.ToUpper(notice), "STALE") {
			t.Errorf("notice outside a git repo should never claim staleness, got:\n%s", notice)
		}
	})
}
