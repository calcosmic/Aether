package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

// --- Task 1: the wisdom review runs exactly once, before anything changes ---

// sealTestState builds the minimal completed-colony fixture every Task 1
// test seals against: one completed phase, nothing blocking.
func sealTestState(goal string) colony.ColonyState {
	return colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Complete work",
			Status: colony.PhaseCompleted,
		}}},
	}
}

// countConsolidationSealEvents reads the real event bus a completed seal
// wrote to and counts how many times runSealConsolidation's own
// "consolidation.seal" topic was published -- instrumentation fed by the
// real call path (pkg/events' persisted JSONL), never a mock of
// runSealWisdomReview or runSealConsolidation themselves.
func countConsolidationSealEvents(t *testing.T, s interface {
	ReadJSONL(string) ([]json.RawMessage, error)
}) int {
	t.Helper()
	lines, err := s.ReadJSONL("event-bus.jsonl")
	if err != nil {
		t.Fatalf("read event bus: %v", err)
	}
	count := 0
	for _, line := range lines {
		var evt events.Event
		if err := json.Unmarshal(line, &evt); err != nil {
			t.Fatalf("unmarshal persisted event: %v", err)
		}
		if evt.Topic == "consolidation.seal" {
			count++
		}
	}
	return count
}

// TestSealWisdomReviewRunsExactlyOncePerSeal pins D-05's "runs once": a
// complete seal must publish exactly one consolidation.seal event, proving
// runSealConsolidation (the review's core) ran a single time -- not once
// for a stand-alone review pass and again inside completeSealRuntime.
func TestSealWisdomReviewRunsExactlyOncePerSeal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	stdout = &bytes.Buffer{}

	if err := s.SaveJSON("COLONY_STATE.json", sealTestState("Wisdom review runs once")); err != nil {
		t.Fatalf("save state: %v", err)
	}

	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal returned error: %v", err)
	}

	if got := countConsolidationSealEvents(t, s); got != 1 {
		t.Fatalf("consolidation.seal published %d time(s), want exactly 1", got)
	}
}

// TestSealWisdomReviewPrecedesStateChange asserts the learned-lessons store
// (local QUEEN.md, written by the seal-side promotion loop inside
// runSealWisdomReview) is written before COLONY_STATE.json records the
// finished state -- by comparing the two files' observed write order in a
// seeded fixture store, per D-05.
func TestSealWisdomReviewPrecedesStateChange(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	store = s
	stdout = &bytes.Buffer{}

	// An instinct eligible for the SEAL-SIDE local promotion loop
	// (Confidence >= 0.8, non-empty Action) but NOT QueenEligible via
	// consolidation (application history below 3), so promoteInstinctLocal
	// itself is the writer that touches QUEEN.md for this fixture.
	instincts := colony.InstinctsFile{Version: "1.0", Instincts: []colony.InstinctEntry{{
		ID:         "precedes-state-change",
		Trigger:    "seal ordering fixture",
		Action:     "Keep the wisdom review ahead of the state mutation",
		Domain:     "testing",
		Confidence: 0.95,
	}}}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("seed instincts: %v", err)
	}
	if err := s.SaveJSON("COLONY_STATE.json", sealTestState("Wisdom review precedes state change")); err != nil {
		t.Fatalf("save state: %v", err)
	}

	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal returned error: %v", err)
	}

	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	queenStat, err := os.Stat(queenPath)
	if err != nil {
		t.Fatalf("expected local QUEEN.md to be written by the review: %v", err)
	}
	statePath := filepath.Join(tmpDir, ".aether", "data", "COLONY_STATE.json")
	stateStat, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("stat COLONY_STATE.json: %v", err)
	}

	if queenStat.ModTime().After(stateStat.ModTime()) {
		t.Fatalf("QUEEN.md (the review's own lessons store) was written AFTER COLONY_STATE.json recorded the finished state: queen=%s state=%s",
			queenStat.ModTime(), stateStat.ModTime())
	}

	var after colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &after); err != nil {
		t.Fatalf("load state after seal: %v", err)
	}
	if after.State != colony.StateCOMPLETED {
		t.Fatalf("state after seal = %s, want COMPLETED", after.State)
	}

	queenText, err := os.ReadFile(queenPath)
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	if !bytes.Contains(queenText, []byte("Keep the wisdom review ahead of the state mutation")) {
		t.Fatalf("QUEEN.md missing the promoted instinct's action text:\n%s", queenText)
	}
}

// hashTree returns a stable per-file content hash for every regular file
// under root, keyed by the path relative to root.
func hashTree(t *testing.T, root string) map[string]string {
	t.Helper()
	hashes := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// .aether/locks/ holds advisory lock files storage.Store creates
			// (and .cache_*.json read-through caches) as a side effect of
			// EVERY read, mutating or not -- infra bookkeeping, not colony
			// state or the learned-lessons store, so it is excluded from
			// this invariant.
			if info.Name() == "locks" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(info.Name(), ".cache_") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		hashes[rel] = fmt.Sprintf("%x", sha256.Sum256(data))
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return hashes
}

// TestSealPlanOnlyDoesNotMutate pins the CLAUDE.md dry-run corollary for the
// seal inspection path: --plan-only must never write to colony state or the
// learned-lessons store, byte-for-byte, across every file in the fixture
// store.
func TestSealPlanOnlyDoesNotMutate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	store = s
	stdout = &bytes.Buffer{}

	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         stringPtrForSealConfirmationTest("Seal plan-only must not mutate"),
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Complete work",
			Status: colony.PhaseCompleted,
			Mode:   colony.PhaseModeProduction,
		}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	before := hashTree(t, tmpDir)

	rootCmd.SetArgs([]string{"seal", "--plan-only"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal --plan-only returned error: %v", err)
	}

	after := hashTree(t, tmpDir)

	if len(before) != len(after) {
		beforeKeys := sortedKeys(before)
		afterKeys := sortedKeys(after)
		t.Fatalf("file set changed: before=%v after=%v", beforeKeys, afterKeys)
	}
	for path, wantHash := range before {
		gotHash, ok := after[path]
		if !ok {
			t.Fatalf("file disappeared during --plan-only: %s", path)
		}
		if gotHash != wantHash {
			t.Fatalf("file mutated by --plan-only: %s", path)
		}
	}
}

func stringPtrForSealConfirmationTest(s string) *string {
	return &s
}
