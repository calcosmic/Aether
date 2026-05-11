package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestOrchestratorBoundaryClarificationSourceFormatsWorkflowPhaseCategory(t *testing.T) {
	got := orchestratorBoundaryClarificationSource("build", 4, "scope", false)
	want := "orchestrator:build:phase:4:scope"
	if got != want {
		t.Fatalf("source = %q, want %q", got, want)
	}
}

func TestOrchestratorBoundaryClarificationSourceKeepsHardSuffixLast(t *testing.T) {
	got := orchestratorBoundaryClarificationSource("continue", 7, "advance", true)
	want := "orchestrator:continue:phase:7:advance:hard"
	if got != want {
		t.Fatalf("source = %q, want %q", got, want)
	}
	if !clarificationIsHardConstraint(PendingDecision{Source: got}) {
		t.Fatalf("source %q should be treated as a hard clarification", got)
	}
}

func TestOrchestratorBoundaryClarificationSourceNormalizesPhaseAndCategory(t *testing.T) {
	got := orchestratorBoundaryClarificationSource(" Continue Plan ", -2, " Build / Scope?! ", false)
	want := "orchestrator:continue-plan:phase:0:build-scope"
	if got != want {
		t.Fatalf("source = %q, want %q", got, want)
	}
}

func TestMaterializeOrchestratorBoundaryClarificationsCreatesPendingClarification(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	questions, createdCount, existingCount, err := materializeOrchestratorBoundaryClarifications("Build", 4, loadPendingDecisionFile(), []discussQuestion{{
		Category:       "Scope",
		Question:       "What should the next build optimize for?",
		Options:        []string{"smallest slice", "broad coverage", "pause"},
		Reasoning:      "The next build can trade speed against breadth.",
		HardConstraint: true,
	}}, 3, false)
	if err != nil {
		t.Fatalf("materialize boundary clarifications: %v", err)
	}
	if createdCount != 1 || existingCount != 0 || len(questions) != 1 {
		t.Fatalf("created=%d existing=%d questions=%d, want 1/0/1", createdCount, existingCount, len(questions))
	}
	if questions[0].Status != "new" {
		t.Fatalf("question status = %q, want new", questions[0].Status)
	}

	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		t.Fatalf("load pending decisions: %v", err)
	}
	if len(file.Decisions) != 1 {
		t.Fatalf("decisions = %d, want 1", len(file.Decisions))
	}
	decision := file.Decisions[0]
	if decision.Type != clarificationDecisionType {
		t.Fatalf("decision type = %q, want %q", decision.Type, clarificationDecisionType)
	}
	if decision.Source != "orchestrator:build:phase:4:scope:hard" {
		t.Fatalf("decision source = %q", decision.Source)
	}
	if decision.Phase == nil || *decision.Phase != 4 {
		t.Fatalf("decision phase = %v, want 4", decision.Phase)
	}
	if decision.Description != formatClarificationDescription("What should the next build optimize for?", []string{"smallest slice", "broad coverage", "pause"}) {
		t.Fatalf("decision description = %q", decision.Description)
	}
}

func TestMaterializeOrchestratorBoundaryClarificationsDedupesBySource(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	phase := 4
	source := orchestratorBoundaryClarificationSource("build", phase, "scope", false)
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{
		Decisions: []PendingDecision{{
			ID:          "pd_existing",
			Type:        clarificationDecisionType,
			Description: formatClarificationDescription("Existing boundary question?", []string{"existing", "other"}),
			Source:      source,
			Resolved:    false,
			CreatedAt:   "2026-04-19T10:00:00Z",
		}},
	}); err != nil {
		t.Fatalf("seed pending decisions: %v", err)
	}

	questions, createdCount, existingCount, err := materializeOrchestratorBoundaryClarifications(" Build ", phase, loadPendingDecisionFile(), []discussQuestion{{
		Category:  " Scope ",
		Question:  "New boundary question?",
		Options:   []string{"new", "alternate"},
		Reasoning: "This candidate should reuse the existing source.",
	}}, 3, false)
	if err != nil {
		t.Fatalf("materialize boundary clarifications: %v", err)
	}
	if createdCount != 0 || existingCount != 1 || len(questions) != 1 {
		t.Fatalf("created=%d existing=%d questions=%d, want 0/1/1", createdCount, existingCount, len(questions))
	}
	if questions[0].ID != "pd_existing" {
		t.Fatalf("question id = %q, want pd_existing", questions[0].ID)
	}
	if questions[0].Question != "Existing boundary question?" {
		t.Fatalf("question = %q, want persisted question text", questions[0].Question)
	}
	if questions[0].Status != "pending" {
		t.Fatalf("question status = %q, want pending", questions[0].Status)
	}

	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		t.Fatalf("load pending decisions: %v", err)
	}
	if len(file.Decisions) != 1 {
		t.Fatalf("decisions = %d, want dedupe to keep one", len(file.Decisions))
	}
}

func TestMaterializeOrchestratorBoundaryClarificationsStampsScopeOnNewDecisions(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Build a dashboard for internal operations"
	sessionID := "sess_scope_stamp_test"
	initializedAt := time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC)
	if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		SessionID:     &sessionID,
		InitializedAt: &initializedAt,
	}); err != nil {
		t.Fatalf("seed colony state: %v", err)
	}

	_, createdCount, _, err := materializeOrchestratorBoundaryClarifications("build", 4, loadPendingDecisionFile(), []discussQuestion{{
		Category:       "Scope",
		Question:       "What should the next build optimize for?",
		Options:        []string{"smallest slice", "broad coverage"},
		HardConstraint: true,
	}}, 3, false)
	if err != nil {
		t.Fatalf("materialize boundary clarifications: %v", err)
	}
	if createdCount != 1 {
		t.Fatalf("createdCount = %d, want 1", createdCount)
	}

	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		t.Fatalf("load pending decisions: %v", err)
	}
	decision := file.Decisions[0]
	if decision.GoalHash == "" {
		t.Fatal("boundary decision GoalHash should be stamped from colony state, got empty")
	}
	if decision.SessionID != sessionID {
		t.Fatalf("boundary decision SessionID = %q, want %q", decision.SessionID, sessionID)
	}
	expectedGoalHash := pendingDecisionGoalHash(goal)
	if decision.GoalHash != expectedGoalHash {
		t.Fatalf("boundary decision GoalHash = %q, want %q", decision.GoalHash, expectedGoalHash)
	}
}
