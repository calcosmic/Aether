package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestActivateGeneratedPlanPreservesCompletedPrefixAndOffsetsFutureDependencies(t *testing.T) {
	completedTaskID := "1.1"
	oldFutureTaskID := "2.1"
	previous := colony.Plan{
		EvidencePolicy: colony.PlanEvidenceBoundV1,
		Phases: []colony.Phase{
			{
				ID:              1,
				Name:            "Proven foundation",
				Description:     "Already accepted",
				Status:          colony.PhaseCompleted,
				SuccessCriteria: []string{"foundation works"},
				Tasks: []colony.Task{{
					ID:              &completedTaskID,
					Goal:            "Build foundation",
					Status:          colony.TaskCompleted,
					SuccessCriteria: []string{"test passes"},
				}},
			},
			{
				ID:     2,
				Name:   "Obsolete future",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &oldFutureTaskID, Goal: "Old approach", Status: colony.TaskPending}},
			},
		},
	}
	beforeCompleted, err := json.Marshal(previous.Phases[0])
	if err != nil {
		t.Fatal(err)
	}

	localTaskOne := "1.1"
	localTaskTwo := "2.1"
	candidate := []colony.Phase{
		{ID: 1, Name: "Corrected implementation", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &localTaskOne, Goal: "Use research-backed approach", Status: colony.TaskPending}}},
		{ID: 2, Name: "Verification", Status: colony.PhasePending, Tasks: []colony.Task{{ID: &localTaskTwo, Goal: "Prove corrected behavior", Status: colony.TaskPending, DependsOn: []string{"1.1"}}}},
	}
	confidence := 0.92
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	manifest := codexPlanManifest{
		Refresh:       true,
		PlanningRunID: "plan-research-1",
		Revision: &codexPlanRevisionContext{
			ReasonType: colony.PlanRevisionResearch,
			Reason:     "Oracle disproved the original integration assumption",
			Evidence:   []string{".aether/oracle/synthesis.md"},
		},
	}

	plan, revision, err := activateGeneratedPlan(previous, candidate, now, &confidence, colony.PlanEvidenceBoundV1, manifest, "evidence-hash")
	if err != nil {
		t.Fatalf("activateGeneratedPlan: %v", err)
	}
	afterCompleted, err := json.Marshal(plan.Phases[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(beforeCompleted) != string(afterCompleted) {
		t.Fatalf("completed phase changed during revision\nbefore=%s\nafter=%s", beforeCompleted, afterCompleted)
	}
	if len(plan.Phases) != 3 || plan.Phases[1].ID != 2 || plan.Phases[2].ID != 3 {
		t.Fatalf("revised phase ids = %+v, want immutable phase 1 plus replacements 2-3", phaseIDs(plan.Phases))
	}
	if got := ptrStr(plan.Phases[1].Tasks[0].ID); got != "2.1" {
		t.Fatalf("first replacement task id = %q, want 2.1", got)
	}
	if got := plan.Phases[2].Tasks[0].DependsOn; len(got) != 1 || got[0] != "2.1" {
		t.Fatalf("offset dependency = %v, want [2.1]", got)
	}
	if plan.ActiveRevisionID != revision.ID || revision.Number != 2 || revision.ParentID == "" {
		t.Fatalf("revision chain is incomplete: plan=%+v revision=%+v", plan, revision)
	}
	if len(plan.Revisions) != 2 || plan.Revisions[0].ReasonType != colony.PlanRevisionLegacyImport {
		t.Fatalf("legacy baseline was not preserved: %+v", plan.Revisions)
	}
	if len(revision.PreservedPhaseIDs) != 1 || revision.PreservedPhaseIDs[0] != 1 || len(revision.SupersededPhaseIDs) != 1 || revision.SupersededPhaseIDs[0] != 2 {
		t.Fatalf("revision mapping is incomplete: %+v", revision)
	}
}

func TestPlanDefinitionIdentitySurvivesLifecycleStatusChanges(t *testing.T) {
	taskID := "1.1"
	ready := colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Build", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Goal: "Implement", Status: colony.TaskPending}}}}}
	built := ready
	built.Phases = clonePhases(ready.Phases)
	built.Phases[0].Status = colony.PhaseInProgress
	built.Phases[0].Tasks[0].Status = colony.TaskCompleted
	if activePlanRevisionID(ready) != activePlanRevisionID(built) {
		t.Fatalf("legacy revision identity changed with lifecycle status: %s != %s", activePlanRevisionID(ready), activePlanRevisionID(built))
	}
	readyStateHash, _ := planStateHash(ready)
	builtStateHash, _ := planStateHash(built)
	if readyStateHash == builtStateHash {
		t.Fatal("full plan state hash must change when lifecycle status changes")
	}
}

func TestValidatePlanManifestBaseRejectsStalePlanningPacket(t *testing.T) {
	taskID := "1.1"
	state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Future", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Goal: "Original", Status: colony.TaskPending}}}}}}
	hash, err := planStateHash(state.Plan)
	if err != nil {
		t.Fatal(err)
	}
	manifest := codexPlanManifest{BaseRevisionID: activePlanRevisionID(state.Plan), BasePlanStateHash: hash}
	state.Plan.Phases[0].Tasks[0].Goal = "Changed after dispatch"
	if err := validatePlanManifestBase(t.TempDir(), manifest, state); err == nil || !strings.Contains(err.Error(), "stale packet") {
		t.Fatalf("expected stale plan packet rejection, got %v", err)
	}
}

func TestBuildPlanRevisionContextRequiresTraceableResearchEvidence(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	goal := "Revise future work"
	taskOne := "1.1"
	taskTwo := "2.1"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Done", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskOne, Goal: "Done", Status: colony.TaskCompleted}}},
			{ID: 2, Name: "Future", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskTwo, Goal: "Future", Status: colony.TaskPending}}},
		}},
	}
	if _, err := buildPlanRevisionContext(root, state, codexPlanOptions{Refresh: true, RevisionType: "research", RevisionReason: "Evidence changed"}); err == nil || !strings.Contains(err.Error(), "requires at least one") {
		t.Fatalf("expected missing research evidence rejection, got %v", err)
	}
	evidence := filepath.Join(root, ".aether", "oracle", "synthesis.md")
	if err := os.MkdirAll(filepath.Dir(evidence), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evidence, []byte("# Finding\nThe assumption is false.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	context, err := buildPlanRevisionContext(root, state, codexPlanOptions{
		Refresh:          true,
		RevisionType:     "research",
		RevisionReason:   "Evidence changed",
		RevisionEvidence: []string{".aether/oracle/synthesis.md"},
	})
	if err != nil {
		t.Fatalf("build revision context: %v", err)
	}
	if len(context.CompletedPhases) != 1 || len(context.SupersededPhases) != 1 || context.ReasonType != colony.PlanRevisionResearch {
		t.Fatalf("revision context = %+v", context)
	}
	if context.EvidenceHash == "" {
		t.Fatal("revision context did not bind the Oracle evidence contents")
	}
	manifest := codexPlanManifest{
		Refresh:           true,
		BaseRevisionID:    context.BaseRevisionID,
		BasePlanStateHash: context.BasePlanStateHash,
		Revision:          context,
	}
	if err := validatePlanManifestBase(root, manifest, state); err != nil {
		t.Fatalf("fresh revision evidence rejected: %v", err)
	}
	if err := os.WriteFile(evidence, []byte("# Finding\nThe evidence changed after dispatch.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validatePlanManifestBase(root, manifest, state); err == nil || !strings.Contains(err.Error(), "revision evidence changed") {
		t.Fatalf("expected changed revision evidence rejection, got %v", err)
	}
}

func TestValidateBuildManifestRejectsSupersededRevision(t *testing.T) {
	taskID := "1.1"
	state := colony.ColonyState{Plan: colony.Plan{ActiveRevisionID: "plan-r2-current", Phases: []colony.Phase{{ID: 1, Name: "Current", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Goal: "Current", Status: colony.TaskPending}}}}}}
	err := validateBuildManifestPlanRevision(codexBuildManifest{Phase: 1, PlanRevisionID: "plan-r1-old"}, state, false)
	if err == nil || !strings.Contains(err.Error(), "superseded plan revision") {
		t.Fatalf("expected superseded build packet rejection, got %v", err)
	}
}

func TestPlanRevisionCandidatePreservesCompletedCompatibleWork(t *testing.T) {
	completedTaskID := "1.1"
	newTaskID := "2.1"
	base := []colony.Phase{{
		ID: 1, SemanticID: "phase-stable", Name: "Stable work", Status: colony.PhaseCompleted, WatcherFailureCount: 2,
		CandidateID: "old-candidate", CandidateContentHash: strings.Repeat("a", 64), PlanningTimelineID: "old-timeline", PlanningTimelineDigest: strings.Repeat("b", 64),
		Tasks: []colony.Task{{ID: &completedTaskID, SemanticID: "task-stable", Goal: "Keep this work", Status: colony.TaskCompleted, CandidateID: "old-candidate", CandidateContentHash: strings.Repeat("a", 64)}},
	}}
	proposal := append(clonePhases(base), colony.Phase{
		ID: 2, SemanticID: "phase-new", Name: "New work", Status: colony.PhasePending,
		Tasks: []colony.Task{{ID: &newTaskID, SemanticID: "task-new", Goal: "Do the next thing", Status: colony.TaskPending}},
	})
	proposal[0].Status = colony.PhasePending
	proposal[0].WatcherFailureCount = 0
	proposal[0].CandidateID = "new-candidate"
	proposal[0].CandidateContentHash = strings.Repeat("c", 64)
	proposal[0].PlanningTimelineID = "new-timeline"
	proposal[0].PlanningTimelineDigest = strings.Repeat("d", 64)
	proposal[0].Tasks[0].Status = colony.TaskPending
	proposal[0].Tasks[0].CandidateID = "new-candidate"
	proposal[0].Tasks[0].CandidateContentHash = strings.Repeat("c", 64)
	wantCompleted := clonePhases(proposal[:1])[0]
	wantCompleted.Status = colony.PhaseCompleted
	wantCompleted.WatcherFailureCount = 2
	wantCompleted.Tasks[0].Status = colony.TaskCompleted
	wantCompletedBytes, err := json.Marshal(wantCompleted)
	if err != nil {
		t.Fatal(err)
	}
	wantHash, err := planDefinitionHash(proposal)
	if err != nil {
		t.Fatal(err)
	}

	activated, preserved, err := preserveCompletedCandidateWork(base, proposal)
	if err != nil {
		t.Fatal(err)
	}
	gotCompleted, err := json.Marshal(activated[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(gotCompleted) != string(wantCompletedBytes) {
		t.Fatalf("completed lifecycle status or new authority binding changed\nwant=%s\ngot=%s", wantCompletedBytes, gotCompleted)
	}
	if len(preserved) != 1 || preserved[0] != 1 {
		t.Fatalf("preserved phase ids = %v, want [1]", preserved)
	}
	gotHash, err := planDefinitionHash(activated)
	if err != nil {
		t.Fatal(err)
	}
	if gotHash != wantHash {
		t.Fatalf("lifecycle restoration changed immutable proposal identity: got %s want %s", gotHash, wantHash)
	}
}

func TestPlanRevisionReplayCandidateAcceptanceReturnsOriginalReceipt(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	request := planCandidateTestAcceptanceRequest(candidate)
	first, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{
		AcceptedBy: "owner", AcceptedAt: time.Date(2026, time.September, 7, 20, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	afterFirst := planCandidateTestSnapshot(t, root)
	second, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{
		AcceptedBy: "different-retry-actor", AcceptedAt: time.Date(2026, time.September, 8, 20, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed || !reflect.DeepEqual(second.Receipt, first.Receipt) || !reflect.DeepEqual(second.Revision, first.Revision) {
		t.Fatalf("exact replay = %+v, want original %+v", second, first)
	}
	planCandidateTestAssertSnapshot(t, root, afterFirst)
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Plan.Revisions) != 1 || len(state.Plan.Candidates) != 1 {
		t.Fatalf("replay duplicated immutable lineage: revisions=%d candidates=%d", len(state.Plan.Revisions), len(state.Plan.Candidates))
	}
}
