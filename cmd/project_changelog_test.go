package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The owner's ruling, 2026-09-21: marking a project finished writes an entry
// to the project's changelog, creating the file when it is missing, carrying
// the date, the goal, and one line per phase.

// TestFinishingAProjectWritesItsChangelogEntry runs the REAL init -> plan ->
// build -> check -> mark-finished flow and then looks at the project folder,
// so it fails if the writer exists but nothing on the real path calls it --
// the exact way the older changelog commands sat unused.
func TestFinishingAProjectWritesItsChangelogEntry(t *testing.T) {
	root, sealed := runRealLifecycleToSealForTest(t)
	data, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("marking the project finished wrote no changelog: %v", err)
	}
	got := string(data)
	goal := strings.TrimSpace(derefGoal(sealed.Goal))
	if goal == "" {
		t.Fatal("fixture has no goal; the test would prove nothing")
	}
	if !strings.Contains(got, goal) {
		t.Errorf("changelog does not name the project's goal %q:\n%s", goal, got)
	}
	if len(sealed.Plan.Phases) == 0 {
		t.Fatal("fixture has no phases; the test would prove nothing")
	}
	for _, phase := range sealed.Plan.Phases {
		if !strings.Contains(got, strings.TrimSpace(phase.Name)) {
			t.Errorf("changelog does not list phase %q:\n%s", phase.Name, got)
		}
	}
	if !strings.Contains(got, time.Now().UTC().Format("2006-01-02")) {
		t.Errorf("changelog entry carries no date:\n%s", got)
	}
}

func changelogTestState(goal string, phases ...colony.Phase) colony.ColonyState {
	return colony.ColonyState{Version: "3.0", Goal: &goal, Plan: colony.Plan{Phases: phases}}
}

func TestProjectChangelogEntryShapeAndPlacement(t *testing.T) {
	root := t.TempDir()
	existing := "# Changelog\n\nIntro text.\n\n## [Unreleased]\n\n- pending thing\n\n## [1.2.0] - 2026-01-01\n\n- older release\n"
	path := filepath.Join(root, "CHANGELOG.md")
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	state := changelogTestState("Tidy the export packs\nsecond line must not break the heading",
		colony.Phase{ID: 1, Name: "Rename preview pages", Status: colony.PhaseCompleted},
		colony.Phase{ID: 2, Name: "Rebuild the pack index", Status: colony.PhaseCompleted},
	)
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	res := writeProjectChangelogEntry(root, state, colony.SealOutcome{Disposition: colony.SealDispositionVerified}, "seal-tx-1", now)
	if !res.Written || res.Created || res.Problem != "" {
		t.Fatalf("unexpected result: %+v", res)
	}
	got, _ := os.ReadFile(path)
	text := string(got)
	heading := "## 2026-09-21 — Tidy the export packs second line must not break the heading"
	for _, want := range []string{heading, "- Phase 1: Rename preview pages", "- Phase 2: Rebuild the pack index", "Intro text.", "- pending thing", "- older release"} {
		if !strings.Contains(text, want) {
			t.Errorf("changelog missing %q:\n%s", want, text)
		}
	}
	// Newest first, but never above the hand-kept Unreleased section.
	if !(strings.Index(text, "## [Unreleased]") < strings.Index(text, heading) && strings.Index(text, heading) < strings.Index(text, "## [1.2.0]")) {
		t.Errorf("entry is not placed after Unreleased and before older entries:\n%s", text)
	}

	// Replaying the same finish must not write a second entry.
	again := writeProjectChangelogEntry(root, state, colony.SealOutcome{Disposition: colony.SealDispositionVerified}, "seal-tx-1", now)
	after, _ := os.ReadFile(path)
	if again.Written || !again.AlreadyPresent || string(after) != text {
		t.Errorf("a replayed finish changed the changelog: %+v", again)
	}
}

func TestProjectChangelogCreatesTheFileWhenMissing(t *testing.T) {
	root := t.TempDir()
	state := changelogTestState("Ship the importer", colony.Phase{ID: 1, Name: "Parse the feed", Status: colony.PhaseCompleted})
	res := writeProjectChangelogEntry(root, state, colony.SealOutcome{Disposition: colony.SealDispositionVerified}, "seal-tx-2", time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC))
	if !res.Written || !res.Created {
		t.Fatalf("file was not created: %+v", res)
	}
	got, _ := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if !strings.HasPrefix(string(got), "# Changelog") || !strings.Contains(string(got), "## 2026-09-21 — Ship the importer") {
		t.Errorf("created changelog is malformed:\n%s", got)
	}
}

// A project closed before everything was verified must never be written up as
// if it were complete.
func TestProjectChangelogIsHonestAboutAForcedFinish(t *testing.T) {
	root := t.TempDir()
	state := changelogTestState("Migrate the notes",
		colony.Phase{ID: 1, Name: "Inventory", Status: colony.PhaseCompleted},
		colony.Phase{ID: 2, Name: "Copy approved rows", Status: colony.PhaseReady},
	)
	outcome := colony.SealOutcome{Disposition: colony.SealDispositionForcedIncomplete, IncompletePhases: []int{2}}
	res := writeProjectChangelogEntry(root, state, outcome, "seal-tx-3", time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC))
	if !res.Written {
		t.Fatalf("no entry: %+v", res)
	}
	got, _ := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	text := string(got)
	if !strings.Contains(text, "- Phase 2: Copy approved rows (not finished)") || !strings.Contains(text, "closed before every phase was finished") {
		t.Errorf("a forced finish reads as complete:\n%s", text)
	}
	if strings.Contains(text, "- Phase 1: Inventory (not finished)") {
		t.Errorf("a finished phase is marked unfinished:\n%s", text)
	}
}

// Writing the changelog is a courtesy. It must never follow a link out of the
// project, never overwrite something that is not an ordinary file, and never
// report an error a caller could turn into a failed finish.
func TestProjectChangelogNeverWritesThroughALinkAndNeverFails(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "elsewhere.md")
	if err := os.WriteFile(outside, []byte("private\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "CHANGELOG.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	state := changelogTestState("Anything", colony.Phase{ID: 1, Name: "One", Status: colony.PhaseCompleted})
	res := writeProjectChangelogEntry(root, state, colony.SealOutcome{Disposition: colony.SealDispositionVerified}, "seal-tx-4", time.Now().UTC())
	if res.Written || res.Problem == "" {
		t.Fatalf("wrote through a link: %+v", res)
	}
	if got, _ := os.ReadFile(outside); string(got) != "private\n" {
		t.Fatalf("the file outside the project was changed: %q", got)
	}

	noGoal := writeProjectChangelogEntry(t.TempDir(), colony.ColonyState{}, colony.SealOutcome{}, "seal-tx-5", time.Now().UTC())
	if noGoal.Written {
		t.Fatalf("wrote an entry for a project with no goal: %+v", noGoal)
	}
}
