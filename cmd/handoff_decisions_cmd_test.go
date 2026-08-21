package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

func seedHandoffOpenDecision(t *testing.T, question string) {
	t.Helper()
	dispatch := codex.WorkerDispatch{
		WorkerName: "Mason-67",
		Caste:      "builder",
		TaskID:     "1.1",
		Workflow:   "build",
		Phase:      1,
		Wave:       1,
	}
	result := codex.DispatchResult{
		WorkerName: "Mason-67",
		Status:     "completed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Mason-67",
			Caste:      "builder",
			TaskID:     "1.1",
			Status:     "completed",
			Summary:    "seeded handoff with an open decision",
			Handoff: codex.WorkerHandoff{
				ChangedFiles:  []string{"export.go"},
				OpenDecisions: []string{question},
				Freshness:     time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	if err := persistDispatchWorkerHandoff(dispatch, result); err != nil {
		t.Fatalf("seed handoff: %v", err)
	}
}

func TestHandoffDecisionsListsUnansweredOpenDecisions(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	const question = "Should exported CSV include archived rows?"
	seedHandoffOpenDecision(t, question)

	got := pendingHandoffDecisions(0)
	if len(got) != 1 {
		t.Fatalf("expected exactly one pending decision, got %d: %+v", len(got), got)
	}
	if got[0].Question != question {
		t.Fatalf("question = %q, want %q", got[0].Question, question)
	}
	if got[0].Caste != "builder" || got[0].Worker != "Mason-67" {
		t.Fatalf("provenance missing: %+v", got[0])
	}
	if got[0].ID == "" {
		t.Fatalf("expected a stable content id, got empty")
	}

	if phaseScoped := pendingHandoffDecisions(1); len(phaseScoped) != 1 {
		t.Fatalf("phase filter should keep the phase-1 decision, got %+v", phaseScoped)
	}
	if otherPhase := pendingHandoffDecisions(2); len(otherPhase) != 0 {
		t.Fatalf("phase filter should drop other phases, got %+v", otherPhase)
	}
}

func TestHandoffDecisionsExcludesAnsweredQuestions(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	const question = "Should exported CSV include archived rows?"
	seedHandoffOpenDecision(t, question)
	if _, err := recordDecisionAnswer(question, "No — active rows only", 1, "worker-handoff"); err != nil {
		t.Fatalf("record answer: %v", err)
	}

	if got := pendingHandoffDecisions(0); len(got) != 0 {
		t.Fatalf("answered question must stop being listed, got %+v", got)
	}
}

func TestHandoffDecisionsDoesNotMutate(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	var buf bytes.Buffer
	stdout = &buf

	seedHandoffOpenDecision(t, "Should exported CSV include archived rows?")

	handoffPath := filepath.Join(store.BasePath(), workerHandoffsPath)
	before, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("read seeded handoffs: %v", err)
	}
	decisionsBefore, _ := os.ReadFile(filepath.Join(store.BasePath(), pendingDecisionsFile))

	if err := handoffDecisionsCmd.RunE(handoffDecisionsCmd, nil); err != nil {
		t.Fatalf("handoff-decisions: %v", err)
	}

	after, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("read handoffs after listing: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("handoff-decisions mutated the handoff store — an inspection command must not write")
	}
	decisionsAfter, _ := os.ReadFile(filepath.Join(store.BasePath(), pendingDecisionsFile))
	if !bytes.Equal(decisionsBefore, decisionsAfter) {
		t.Fatalf("handoff-decisions mutated pending-decisions.json — an inspection command must not write")
	}
}

func TestDecisionAnswerRendersIntoClarifiedIntent(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	const question = "Should exported CSV include archived rows?"
	const answer = "No — active rows only"
	if _, err := recordDecisionAnswer(question, answer, 1, "worker-handoff"); err != nil {
		t.Fatalf("record answer: %v", err)
	}

	lines := clarifiedIntentPromptRenderResult().Lines
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, question) || !strings.Contains(joined, answer) {
		t.Fatalf("CLARIFIED INTENT missing the recorded answer.\nlines: %q", joined)
	}

	section := renderClarifiedIntentSection()
	if !strings.HasPrefix(section, "## CLARIFIED INTENT") {
		t.Fatalf("prompt section must render under the CLARIFIED INTENT heading, got %q", section)
	}
	if !strings.Contains(section, answer) {
		t.Fatalf("prompt section missing the answer, got %q", section)
	}
}
