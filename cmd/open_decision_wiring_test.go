package cmd

import (
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
