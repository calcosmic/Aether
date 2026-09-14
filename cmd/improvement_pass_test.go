package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// seedZeroStateImprovementPassFixtures seeds an empty-but-valid consolidation
// fixture (matching TestRunPhaseEndConsolidationZeroState's own pattern) so
// runPhaseEndConsolidation's own learning pipeline runs cleanly and reports
// ZeroState, leaving these tests free to assert only on the improvement pass
// half of the summary.
func seedZeroStateImprovementPassFixtures(t *testing.T) {
	t.Helper()
	if err := store.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{}}); err != nil {
		t.Fatalf("seed empty instincts.json: %v", err)
	}
	if err := store.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed empty learning-observations.json: %v", err)
	}
}

// TestAutomaticImprovementPassTracerEndToEnd is the plan's Test 5: a colony
// with one declared, unexpired, beneficial candidate whose scope is exactly
// "project knowledge" runs runPhaseEndConsolidation once and ends with a
// canary run recorded in status running, its checkpoint saved, and an
// improvementPassSummary naming the candidate and the action taken.
func TestAutomaticImprovementPassTracerEndToEnd(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	seedZeroStateImprovementPassFixtures(t)

	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load the real committed fixture bank: %v", err)
	}
	var unguardedWords []string
	unguardedCount := 0
	for _, f := range bank.Fixtures {
		if f.Guard == nil {
			unguardedCount++
			unguardedWords = append(unguardedWords, shadowFixtureSubjectWords(f)...)
		}
	}
	if unguardedCount == 0 {
		t.Fatal("fixture-bank honesty check failed: found zero unguarded fixtures in the real bank -- cannot build a beneficial candidate")
	}

	expires := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	record, isNew, err := declareShadowCandidate(
		"candidate-tracer",
		string(canaryScopeProjectKnowledge),
		"addresses "+strings.Join(unguardedWords, " "),
		shadowGraderBenignHarms,
		expires,
		"revert the declared change",
	)
	if err != nil {
		t.Fatalf("declare candidate: %v", err)
	}
	if !isNew {
		t.Fatal("expected a new declaration")
	}

	summary := runPhaseEndConsolidation(1)
	if !summary.Ran {
		t.Fatalf("expected consolidation Ran == true, got %+v", summary)
	}

	pass := summary.ImprovementPass
	if !pass.Ran {
		t.Fatalf("expected the improvement pass Ran == true, got %+v", pass)
	}
	if pass.CandidatesConsidered != 1 {
		t.Fatalf("expected 1 candidate considered, got %d (failures: %v)", pass.CandidatesConsidered, pass.Failures)
	}
	if pass.Compared != 1 {
		t.Fatalf("expected 1 comparison run, got %d (failures: %v)", pass.Compared, pass.Failures)
	}
	if pass.Admitted != 1 {
		t.Fatalf("expected the candidate to be admitted, got Admitted=%d Refused=%d failures=%v", pass.Admitted, pass.Refused, pass.Failures)
	}
	if pass.CanariesStarted != 1 {
		t.Fatalf("expected 1 canary started, got %d", pass.CanariesStarted)
	}
	if len(pass.Events) != 1 {
		t.Fatalf("expected exactly 1 event, got %+v", pass.Events)
	}
	if pass.Events[0].Kind != improvementPassEventStarted {
		t.Fatalf("expected a 'started' event, got %+v", pass.Events[0])
	}
	if pass.Events[0].CandidateID != record.ID {
		t.Fatalf("event names candidate %q, want %q", pass.Events[0].CandidateID, record.ID)
	}

	run, found, err := loadCanaryRun(record.ID)
	if err != nil {
		t.Fatalf("load canary run: %v", err)
	}
	if !found {
		t.Fatal("expected a canary run to be recorded for the admitted candidate")
	}
	if run.Status != canaryRunStatusRunning {
		t.Fatalf("expected canary run status running, got %q", run.Status)
	}
	if run.CheckpointID == "" {
		t.Fatal("expected the canary run to carry a saved checkpoint id")
	}
}

// TestImprovementPassClosingLineIsPlainEnglish is the plan's Test 6: the
// tracer fixture's result["improvement_pass"] renders one plain-English
// closing-card line naming what happened, with no candidate identifier, no
// verdict token, and no raw key=value pair in the rendered text. Exercised
// against BOTH dual-typed shapes: the in-process struct and the
// snake_case-keyed map attachConsolidationSummary produces.
func TestImprovementPassClosingLineIsPlainEnglish(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	seedZeroStateImprovementPassFixtures(t)

	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load the real committed fixture bank: %v", err)
	}
	var unguardedWords []string
	for _, f := range bank.Fixtures {
		if f.Guard == nil {
			unguardedWords = append(unguardedWords, shadowFixtureSubjectWords(f)...)
		}
	}
	if len(unguardedWords) == 0 {
		t.Fatal("fixture-bank honesty check failed: found zero unguarded fixtures in the real bank")
	}

	expires := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	record, _, err := declareShadowCandidate(
		"candidate-closing-line",
		string(canaryScopeProjectKnowledge),
		"addresses "+strings.Join(unguardedWords, " "),
		shadowGraderBenignHarms,
		expires,
		"revert the declared change",
	)
	if err != nil {
		t.Fatalf("declare candidate: %v", err)
	}

	summary := runPhaseEndConsolidation(1)
	pass := summary.ImprovementPass
	if pass.Admitted != 1 {
		t.Fatalf("test setup: expected the candidate to be admitted, got %+v", pass)
	}

	forbidden := []string{
		record.ID,
		"beneficial", "not_beneficial", "overfit", "tied", "inconclusive",
		"project knowledge", "routing",
		"running", "completed", "rolled_back",
		"=",
	}

	assertPlainEnglish := func(t *testing.T, label, rendered string) {
		t.Helper()
		if strings.TrimSpace(rendered) == "" {
			t.Fatalf("%s: expected a non-empty closing-card line", label)
		}
		for _, f := range forbidden {
			if strings.Contains(rendered, f) {
				t.Fatalf("%s: rendered text contains forbidden token %q:\n%s", label, f, rendered)
			}
		}
	}

	t.Run("struct shape (in-process)", func(t *testing.T) {
		rendered := renderImprovementPassBeat(pass)
		assertPlainEnglish(t, "struct shape", rendered)
	})

	t.Run("map shape (attachConsolidationSummary / JSON round-trip)", func(t *testing.T) {
		result := map[string]interface{}{}
		attachConsolidationSummary(result, summary)
		rendered := renderImprovementPassBeat(result["improvement_pass"])
		assertPlainEnglish(t, "map shape", rendered)
	})
}

// TestCheckWithNoDeclaredCandidateCostsNothing is the plan's Test 7: a
// colony with no declared candidate performs no comparison, opens no
// canary, writes no canary or shadow file, and the closing card shows no
// improvement line at all.
func TestCheckWithNoDeclaredCandidateCostsNothing(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	seedZeroStateImprovementPassFixtures(t)

	summary := runPhaseEndConsolidation(1)
	if !summary.Ran {
		t.Fatalf("expected consolidation Ran == true, got %+v", summary)
	}
	pass := summary.ImprovementPass
	if !pass.Ran {
		t.Fatal("expected the improvement pass to report Ran == true even with nothing declared")
	}
	if pass.CandidatesConsidered != 0 {
		t.Fatalf("expected zero candidates considered, got %d", pass.CandidatesConsidered)
	}
	if len(pass.Events) != 0 {
		t.Fatalf("expected zero events, got %+v", pass.Events)
	}

	if _, err := os.Stat(filepath.Join(s.BasePath(), "canary", "runs.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no canary/runs.json to be created; stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.BasePath(), "shadow", "candidates.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no shadow/candidates.json to be created; stat error: %v", err)
	}

	result := map[string]interface{}{}
	attachConsolidationSummary(result, summary)
	if rendered := renderImprovementPassBeat(result["improvement_pass"]); rendered != "" {
		t.Fatalf("expected the closing card to show no improvement line at all, got %q", rendered)
	}
	if rendered := renderImprovementPassBeat(pass); rendered != "" {
		t.Fatalf("expected the closing card to show no improvement line at all (struct shape), got %q", rendered)
	}
}

// TestImprovementPassFailureNeverBlocksTheCheck is the plan's Test 8: with
// the candidate store made unreadable, runPhaseEndConsolidation still
// returns Ran: true and the pass records its own failure reason instead of
// propagating an error.
func TestImprovementPassFailureNeverBlocksTheCheck(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	seedZeroStateImprovementPassFixtures(t)

	shadowDir := filepath.Join(s.BasePath(), "shadow")
	if err := os.MkdirAll(shadowDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", shadowDir, err)
	}
	if err := os.WriteFile(filepath.Join(shadowDir, "candidates.json"), []byte("{ this is not valid json"), 0o644); err != nil {
		t.Fatalf("write corrupt candidate store: %v", err)
	}

	summary := runPhaseEndConsolidation(1)
	if !summary.Ran {
		t.Fatalf("expected consolidation Ran == true despite the unreadable candidate store, got %+v", summary)
	}
	pass := summary.ImprovementPass
	if !pass.Ran {
		t.Fatal("expected the improvement pass to report Ran == true despite an unreadable candidate store")
	}
	if len(pass.Failures) == 0 {
		t.Fatal("expected the pass to record its own failure reason for the unreadable candidate store")
	}
}
