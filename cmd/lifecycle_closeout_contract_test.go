package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Task 1 (205-03): a finish-and-archive run must complete even when the
// hand-off note is absent, and a resume must leave a usable note behind so
// the sequence never sees an empty required slot in the first place.
// See .planning/field-reports/2026-09-14-cosmic-seal-entomb-lifecycle.md,
// finding 2.
// ---------------------------------------------------------------------------

// TestResumeThenSealThenEntombCompletes reproduces the exact field-report
// sequence: pause, resume ("return to work"), seal, then the archive
// preflight -- with no other command in between. seedVerifiedEntombLifecycleAt
// only writes .aether/HANDOFF.md when it is absent, so the note resume writes
// below survives the simulated seal step untouched, proving seal never
// rewrites or removes it.
func TestResumeThenSealThenEntombCompletes(t *testing.T) {
	fixture := newPauseResume199Fixture(t)
	if _, err := pauseColonyAt(fixture.now); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if _, err := resumeColonyAt(fixture.now.Add(time.Minute)); err != nil {
		t.Fatalf("resume: %v", err)
	}

	handoffPath := filepath.Join(fixture.root, ".aether", "HANDOFF.md")
	if _, err := os.Stat(handoffPath); err != nil {
		t.Fatalf("return-to-work did not leave a hand-off note behind: %v", err)
	}

	seedVerifiedEntombLifecycleAt(t, fixture.root, fixture.dataDir)
	state := readEntombState199(t, fixture.dataDir)

	preflight, err := prepareEntombPreflight(entombTransactionInput{
		Root: fixture.root, DataRoot: fixture.dataDir, Now: fixture.now.Add(2 * time.Hour),
	}, state)
	if err != nil {
		t.Fatalf("archive preflight failed after resume -> seal: %v", err)
	}
	if len(preflight.Sources) == 0 {
		t.Fatal("expected a non-empty archive source list")
	}

	tombstone := entombPreparedSourceByKind(preflight, "tombstone_input")
	if tombstone == nil {
		t.Fatal("expected a tombstone_input source in the preflight")
	}
	if tombstone.Manifest.Synthesized {
		t.Fatal("the resume-written hand-off note should have been read from disk, not synthesized")
	}
}

// TestEntombStillFailsOnAGenuinelyMissingSource proves the synthesised
// fallback is scoped to exactly the tombstone_input (hand-off note) path: a
// genuinely missing COLONY_STATE.json -- a required source with no
// fallback -- still fails the archive by name.
func TestEntombStillFailsOnAGenuinelyMissingSource(t *testing.T) {
	saveGlobals(t)
	binding := bindCommandTestRepository(t)
	goal := "Exercise required-source failure"
	createTestColonyState(t, binding.DataDir, colony.ColonyState{
		Goal: &goal, CurrentPhase: 1, State: colony.StateREADY,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Discovery", Status: colony.PhaseCompleted}}},
	})
	seedVerifiedEntombLifecycleAt(t, binding.Root, binding.DataDir)
	state := readEntombState199(t, binding.DataDir)

	statePath := filepath.Join(binding.DataDir, "COLONY_STATE.json")
	if err := os.Remove(statePath); err != nil {
		t.Fatalf("remove colony state fixture: %v", err)
	}

	_, err := prepareEntombPreflight(entombTransactionInput{
		Root: binding.Root, DataRoot: binding.DataDir, Now: time.Now().UTC(),
	}, state)
	if err == nil {
		t.Fatal("expected the archive preflight to fail when COLONY_STATE.json is genuinely missing")
	}
	if !strings.Contains(err.Error(), "COLONY_STATE.json") {
		t.Fatalf("expected the error to name the missing COLONY_STATE.json source, got: %v", err)
	}
}

// TestEntombSynthesizedInputIsMarkedAsSynthesized proves the archive never
// presents invented content as retrieved content: a missing hand-off note
// produces a stand-in carrying the colony's own goal, phase count/completion
// state and the seal outcome's verdict, and that stand-in is recorded with a
// field distinguishing it from a source actually read from disk.
func TestEntombSynthesizedInputIsMarkedAsSynthesized(t *testing.T) {
	saveGlobals(t)
	binding := bindCommandTestRepository(t)
	goal := "Exercise synthesized tombstone input"
	createTestColonyState(t, binding.DataDir, colony.ColonyState{
		Goal: &goal, CurrentPhase: 1, State: colony.StateREADY,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Discovery", Status: colony.PhaseCompleted}}},
	})
	seedVerifiedEntombLifecycleAt(t, binding.Root, binding.DataDir)

	handoffPath := filepath.Join(binding.Root, ".aether", "HANDOFF.md")
	if err := os.Remove(handoffPath); err != nil {
		t.Fatalf("remove handoff fixture: %v", err)
	}
	state := readEntombState199(t, binding.DataDir)

	preflight, err := prepareEntombPreflight(entombTransactionInput{
		Root: binding.Root, DataRoot: binding.DataDir, Now: time.Now().UTC(),
	}, state)
	if err != nil {
		t.Fatalf("archive preflight with a missing hand-off note: %v", err)
	}

	tombstone := entombPreparedSourceByKind(preflight, "tombstone_input")
	if tombstone == nil {
		t.Fatal("expected a tombstone_input source in the preflight")
	}
	if !tombstone.Manifest.Synthesized {
		t.Fatal("expected the missing hand-off note's source to be marked synthesized")
	}
	if tombstone.Actual != "" {
		t.Fatalf("expected a synthesized source to carry no live Actual path, got %q", tombstone.Actual)
	}
	if !strings.Contains(string(tombstone.Content), goal) {
		t.Fatalf("synthesized stand-in does not carry the colony's own goal: %s", tombstone.Content)
	}
	if !strings.Contains(string(tombstone.Content), string(state.State)) {
		t.Fatalf("synthesized stand-in does not carry the colony's completion state: %s", tombstone.Content)
	}
	if !strings.Contains(string(tombstone.Content), string(state.SealOutcome.Disposition)) {
		t.Fatalf("synthesized stand-in does not carry the seal outcome's verdict: %s", tombstone.Content)
	}
}

func entombPreparedSourceByKind(preflight entombPreflight, kind string) *entombPreparedSource {
	for i := range preflight.Sources {
		if preflight.Sources[i].Manifest.Kind == kind {
			return &preflight.Sources[i]
		}
	}
	return nil
}
