package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
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

func TestLegacyAnsweredDecisionTextIsScopedToActiveSessionAndGoal(t *testing.T) {
	const question = "Should exported CSV include archived rows?"
	initializedAt := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	currentGoal := "Export the active customer ledger"
	currentSession := "session-current-export"

	tests := []struct {
		name             string
		decisionType     string
		sessionID        string
		goalHash         string
		createdAt        time.Time
		wantStillPending bool
	}{
		{
			name:         "matching scope suppresses the answered question",
			decisionType: clarificationDecisionType,
			sessionID:    currentSession,
			goalHash:     pendingDecisionGoalHash(currentGoal),
			createdAt:    initializedAt.Add(time.Minute),
		},
		{
			name:             "foreign session cannot suppress current work",
			decisionType:     clarificationDecisionType,
			sessionID:        "session-foreign-export",
			goalHash:         pendingDecisionGoalHash(currentGoal),
			createdAt:        initializedAt.Add(time.Minute),
			wantStillPending: true,
		},
		{
			name:             "foreign goal cannot suppress current work",
			decisionType:     clarificationDecisionType,
			goalHash:         pendingDecisionGoalHash("Export every archived customer ledger"),
			createdAt:        initializedAt.Add(time.Minute),
			wantStillPending: true,
		},
		{
			name:             "stale unscoped row cannot suppress current work",
			decisionType:     clarificationDecisionType,
			createdAt:        initializedAt.Add(-time.Minute),
			wantStillPending: true,
		},
		{
			name:             "protected checkpoint text is not a legacy answer",
			decisionType:     autopilotCheckpointTypeVisual,
			sessionID:        currentSession,
			goalHash:         pendingDecisionGoalHash(currentGoal),
			createdAt:        initializedAt.Add(time.Minute),
			wantStillPending: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveGlobalsCmd(t)
			s, tmpDir := newTestStoreCmd(t)
			t.Cleanup(func() { os.RemoveAll(tmpDir) })
			store = s
			if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{
				Version:       "3.0",
				Goal:          &currentGoal,
				SessionID:     &currentSession,
				InitializedAt: &initializedAt,
			}); err != nil {
				t.Fatalf("seed current colony scope: %v", err)
			}
			seedHandoffOpenDecision(t, question)

			resolvedAt := tt.createdAt.Add(time.Minute).UTC().Format(time.RFC3339)
			if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
				ID:          "pd_legacy_answer",
				Type:        tt.decisionType,
				Description: formatClarificationDescription(question, nil),
				Source:      "worker-handoff",
				SessionID:   tt.sessionID,
				GoalHash:    tt.goalHash,
				Resolution:  "No — active rows only",
				Resolved:    true,
				CreatedAt:   tt.createdAt.UTC().Format(time.RFC3339),
				ResolvedAt:  resolvedAt,
			}}}); err != nil {
				t.Fatalf("seed resolved legacy answer: %v", err)
			}

			got := pendingHandoffDecisions(0)
			if tt.wantStillPending {
				if len(got) != 1 || got[0].Question != question {
					t.Fatalf("stale or protected answer hid current owner work: %+v", got)
				}
				return
			}
			if len(got) != 0 {
				t.Fatalf("current-scope answer should suppress its matching question: %+v", got)
			}
		})
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

func TestDecisionAnswerRejectsCheckpointAuthorizationFailures(t *testing.T) {
	tests := []struct {
		name              string
		phase             string
		capability        func(t *testing.T, refs []autopilotCheckpointReference) string
		includeCapability bool
	}{
		{name: "missing capability", phase: "8"},
		{name: "wrong capability", phase: "8", includeCapability: true, capability: func(t *testing.T, refs []autopilotCheckpointReference) string {
			return "not-the-displayed-capability"
		}},
		{name: "cross-row capability", phase: "8", includeCapability: true, capability: func(t *testing.T, refs []autopilotCheckpointReference) string {
			return checkpointCapabilityFromReference(t, refs[1])
		}},
		{name: "wrong phase", phase: "9", includeCapability: true, capability: func(t *testing.T, refs []autopilotCheckpointReference) string {
			return checkpointCapabilityFromReference(t, refs[0])
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveGlobalsCmd(t)
			resetRootCmd(t)
			forceJSONOutputModeForTest(t)
			s, tmpDir := newTestStoreCmd(t)
			t.Cleanup(func() { os.RemoveAll(tmpDir) })
			store = s
			phase := colony.Phase{ID: 8, Name: "Capability boundary", Status: colony.PhaseInProgress}
			checkpointTestState(t, phase, colony.StateBUILT)
			criteria := []codexCriterionVerification{
				{TaskID: "8.1", Criterion: "The primary flow feels right", State: criterionStateNeedsOwnerConfirmation},
				{TaskID: "8.2", Criterion: "The fallback flow feels right", State: criterionStateNeedsOwnerConfirmation},
			}
			refs, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria, checkpointTestGeneration(t, "handoff-authorization", "handoff-authorization-evidence"))
			if err != nil || len(refs) != 2 {
				t.Fatalf("materialize checkpoints: refs=%#v err=%v", refs, err)
			}
			before := pendingDecisionBytes(t)

			args := []string{
				"decision-answer",
				"--question", refs[0].Question,
				"--answer", "confirmed",
				"--phase", tt.phase,
			}
			if tt.includeCapability {
				args = append(args, "--checkpoint-capability", tt.capability(t, refs))
			}
			var outBuf, errBuf bytes.Buffer
			stdout = &outBuf
			stderr = &errBuf
			rootCmd.SetArgs(args)
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("decision-answer returned Cobra error: %v", err)
			}
			if errBuf.Len() == 0 {
				t.Fatalf("authorization failure was accepted: %s", outBuf.String())
			}
			if envelope := parseEnvelope(t, errBuf.String()); envelope["ok"] != false {
				t.Fatalf("authorization failure did not return an error envelope: %#v", envelope)
			}
			if after := pendingDecisionBytes(t); !bytes.Equal(before, after) {
				t.Fatalf("authorization failure mutated pending decisions:\nbefore=%s\nafter=%s", before, after)
			}
		})
	}
}

func TestRecordDecisionAnswerDoesNotResolveCheckpoint(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	phase := colony.Phase{ID: 10, Name: "Generic helper boundary", Status: colony.PhaseInProgress}
	checkpointTestState(t, phase, colony.StateBUILT)
	checkpoint, _, err := upsertAutopilotCheckpoint(PendingDecision{
		Type:        autopilotCheckpointTypeVisual,
		Description: formatClarificationDescription("Phase 10: inspect the owner-facing layout", nil),
		Source:      "generic-helper-test",
	}, phase.ID, "visual-boundary", checkpointTestGeneration(t, "handoff-generic", "handoff-generic-evidence"))
	if err != nil {
		t.Fatalf("seed checkpoint: %v", err)
	}
	question := checkpointDecisionQuestion(checkpoint)

	recorded, err := recordDecisionAnswer(question, "worker attempted bypass", phase.ID, "worker-handoff")
	if err != nil {
		t.Fatalf("record generic answer: %v", err)
	}
	if recorded.ID == checkpoint.ID {
		t.Fatalf("generic helper resolved protected checkpoint %s", checkpoint.ID)
	}
	decisions := loadCheckpointDecisions(t)
	if len(decisions) != 2 {
		t.Fatalf("generic helper should append an ordinary clarification without mutating the checkpoint: %#v", decisions)
	}
	if decisions[0].ID != checkpoint.ID || decisions[0].Resolved {
		t.Fatalf("generic helper changed protected checkpoint: %#v", decisions[0])
	}
	if decisions[1].Type != clarificationDecisionType || !decisions[1].Resolved {
		t.Fatalf("generic helper did not preserve ordinary clarification behavior: %#v", decisions[1])
	}
}
