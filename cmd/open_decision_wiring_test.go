package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestResolvedOpenDecisionReachesNextWorkerPrompt locks the full relay: a
// worker leaves an open decision -> the owner answers it -> the answer is
// injected into subsequent worker context -> the question stops being
// listed. If any link breaks, workers go back to inheriting guesses instead
// of the owner's ruling.
func TestResolvedOpenDecisionReachesNextWorkerPrompt(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	const question = "Should exported CSV include archived rows?"
	const answer = "No — active rows only"

	seedHandoffOpenDecision(t, question)

	pending := pendingHandoffDecisions(0)
	if len(pending) != 1 || pending[0].Question != question {
		t.Fatalf("worker's open decision not listed for the owner: %+v", pending)
	}

	if _, err := recordDecisionAnswer(question, answer, 1, "worker-handoff"); err != nil {
		t.Fatalf("record answer: %v", err)
	}

	capsule := resolveCodexWorkerContext()
	if !strings.Contains(capsule, "CLARIFIED INTENT") {
		t.Fatalf("worker context capsule missing the CLARIFIED INTENT section after an answer was recorded.\ncapsule:\n%s", capsule)
	}
	if !strings.Contains(capsule, answer) {
		t.Fatalf("worker context capsule missing the owner's answer.\ncapsule:\n%s", capsule)
	}

	if still := pendingHandoffDecisions(0); len(still) != 0 {
		t.Fatalf("answered decision still listed as pending: %+v", still)
	}
}

// TestBuildBriefTellsWorkersToRouteJudgementCalls: the wrapper-delivered
// build brief must carry the same open-decisions routing rule as the native
// response contract — a worker that guesses silently defeats the relay.
func TestBuildBriefTellsWorkersToRouteJudgementCalls(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	phase := probeGatingPhase("Add user authentication", "Implement login endpoints and session handling", colony.PhaseModeProduction)
	dispatch := codexBuildDispatch{Caste: "builder", Name: "Mason-1", Task: "Implement the endpoint", Stage: "build", Wave: 1}
	brief := composeBuildManifestBrief(tmpDir, phase, dispatch, time.Now(), false)
	if !strings.Contains(brief, codex.HandoffOpenDecisionsGuidance) {
		t.Fatalf("build brief missing the open-decisions routing guidance.\nbrief tail:\n%s", brief[maxInt(0, len(brief)-600):])
	}
}

// TestNewestOwnerAnswerReachesTheWorkerPrompt (Phase 210 blocker 17): an
// owner's answer recorded with `aether decision-answer` must reach the very
// next worker, even when the project already holds more earlier answers than
// the capped CLARIFIED INTENT section can carry. The cap used to keep the
// OLDEST answers, so a just-given ruling was silently dropped and the chat had
// to relay it by hand. Every answer here is written by the runtime's own
// writer, so the ordering is the one a real project produces.
func TestNewestOwnerAnswerReachesTheWorkerPrompt(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	for i := 1; i <= clarifiedIntentMaxEntries; i++ {
		if _, err := recordDecisionAnswer(fmt.Sprintf("Earlier question %d?", i), fmt.Sprintf("Earlier answer %d", i), 1, "worker-handoff"); err != nil {
			t.Fatalf("record earlier answer %d: %v", i, err)
		}
	}
	const answer = "Leave the templates alone; only the data file changes"
	newest, err := recordDecisionAnswer("Which file should the fix touch?", answer, 2, "worker-handoff")
	if err != nil {
		t.Fatalf("record newest answer: %v", err)
	}

	// The prompt_section `aether decision-answer` hands back to the wrapper.
	if section := renderClarifiedIntentSection(); !strings.Contains(section, answer) {
		t.Fatalf("decision-answer prompt_section dropped the answer just given.\nsection:\n%s", section)
	}
	// The capsule every later worker brief starts from.
	if capsule := resolveCodexWorkerContext(); !strings.Contains(capsule, answer) {
		t.Fatalf("worker context capsule dropped the answer just given.\ncapsule:\n%s", capsule)
	}
	rendered := clarifiedIntentPromptRenderResult()
	found := false
	for _, id := range rendered.DecisionIDs {
		if id == newest.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("newest decision %s missing from the delivered decision IDs %v", newest.ID, rendered.DecisionIDs)
	}
	if len(rendered.Lines) > clarifiedIntentMaxEntries {
		t.Fatalf("section grew past its cap: %d lines", len(rendered.Lines))
	}
}
