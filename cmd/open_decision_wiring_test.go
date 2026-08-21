package cmd

import (
	"os"
	"strings"
	"testing"
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
