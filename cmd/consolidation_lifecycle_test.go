package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// captureStderrForConsolidationTest redirects os.Stderr for the duration of
// fn and returns everything written to it. Modelled on the identical local
// helper in cmd/hive_policy_test.go (TestHiveRuntimePolicyUnrecognizedWarns).
func captureStderrForConsolidationTest(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()

	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy stderr: %v", err)
	}
	return buf.String()
}

// TestRunPhaseEndConsolidationMutatesOnRealPath proves runPhaseEndConsolidation
// takes the real (mutating) path, exactly as consolidationPhaseEndCmd's own
// non-dry-run branch does -- mirroring TestConsolidationRealRunStillMutates'
// shape (cmd/consolidation_dryrun_test.go).
func TestRunPhaseEndConsolidationMutatesOnRealPath(t *testing.T) {
	saveGlobals(t)

	dataDir := seedConsolidationFixture(t)
	instincts := filepath.Join(dataDir, "instincts.json")
	before := hashFileForTest(t, instincts)

	summary := runPhaseEndConsolidation(1)

	if !summary.Ran {
		t.Fatalf("expected Ran == true on the real path, got summary: %+v", summary)
	}
	if after := hashFileForTest(t, instincts); after == before {
		t.Fatal("runPhaseEndConsolidation left instincts.json untouched; real path did not mutate")
	}
}

// TestRunPhaseEndConsolidationIsNonBlockingOnFailure asserts D-05: a
// consolidation failure never panics or exits, and is reported through the
// summary rather than propagated as an error the caller could act on.
func TestRunPhaseEndConsolidationIsNonBlockingOnFailure(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	store = s

	// instincts.json containing invalid JSON makes ConsolidationService.Run's
	// LoadJSON step fail; that failure lands in result.Errors rather than a
	// top-level error, which runPhaseEndConsolidation must still treat as a
	// non-blocking failure.
	instinctsPath := filepath.Join(s.BasePath(), "instincts.json")
	if err := os.WriteFile(instinctsPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("seed invalid instincts.json: %v", err)
	}

	var summary phaseEndConsolidationSummary
	stderr := captureStderrForConsolidationTest(t, func() {
		summary = runPhaseEndConsolidation(1)
	})

	if summary.Ran {
		t.Fatalf("expected Ran == false on failure, got summary: %+v", summary)
	}
	if summary.Reason == "" {
		t.Fatal("expected a non-empty Reason on failure")
	}
	if !strings.Contains(stderr, "phase advanced WITHOUT consolidation —") {
		t.Fatalf("expected unmissable D-05 stderr warning, got: %q", stderr)
	}
}

// TestRunPhaseEndConsolidationZeroState asserts D-06/D-07's zero-state
// contract: an empty but VALID store (files exist, parse cleanly, contain
// nothing) still runs cleanly and reports ZeroState() == true.
func TestRunPhaseEndConsolidationZeroState(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	store = s

	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{}}); err != nil {
		t.Fatalf("seed empty instincts.json: %v", err)
	}
	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed empty learning-observations.json: %v", err)
	}

	summary := runPhaseEndConsolidation(1)

	if !summary.Ran {
		t.Fatalf("expected Ran == true on an empty-but-valid store, got summary: %+v", summary)
	}
	if summary.PromotionCandidates != 0 {
		t.Errorf("expected PromotionCandidates == 0, got %d", summary.PromotionCandidates)
	}
	if summary.QueenEligible != 0 {
		t.Errorf("expected QueenEligible == 0, got %d", summary.QueenEligible)
	}
	if !summary.ZeroState() {
		t.Fatal("expected ZeroState() == true for an empty-but-valid store")
	}
}
