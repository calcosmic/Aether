package cmd

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanImpactChangedRequirementClosesOverDependenciesAndProofs(t *testing.T) {
	linkedID := "1.1"
	independentID := "1.2"
	dependentID := "2.1"
	plan := []colony.Phase{
		{
			ID: 1, SemanticID: "phase-account", Status: colony.PhaseCompleted,
			Tasks: []colony.Task{
				{
					ID: &linkedID, SemanticID: "task-account", Status: colony.TaskCompleted,
					RequirementProofLinks: []string{"requirement-account"},
					AcceptanceProofLinks:  []string{"acceptance-account"},
					NegativeProofLinks:    []string{"negative-account"},
					RecoveryProofLinks:    []string{"recovery-account"},
					PublicPathProofLinks:  []string{"path-account"},
				},
				{
					ID: &independentID, SemanticID: "task-independent", Status: colony.TaskCompleted,
					RequirementProofLinks: []string{"requirement-independent"},
					AcceptanceProofLinks:  []string{"acceptance-independent"},
					NegativeProofLinks:    []string{"negative-independent"},
					RecoveryProofLinks:    []string{"recovery-independent"},
					PublicPathProofLinks:  []string{"path-independent"},
				},
			},
		},
		{
			ID: 2, SemanticID: "phase-notification", Status: colony.PhaseReady,
			Tasks: []colony.Task{{
				ID: &dependentID, SemanticID: "task-notification", Status: colony.TaskPending,
				DependsOn:             []string{"task-account"},
				RequirementProofLinks: []string{"requirement-notification"},
				AcceptanceProofLinks:  []string{"acceptance-notification"},
				NegativeProofLinks:    []string{"negative-notification"},
				RecoveryProofLinks:    []string{"recovery-notification"},
				PublicPathProofLinks:  []string{"path-notification"},
			}},
		},
	}
	revision := colony.SpecRevision{
		ID: "spec-revision-successor", ContentHash: strings.Repeat("a", 64),
		Delta: colony.SpecRevisionDelta{
			PredecessorRevisionID: "spec-revision-base",
			Requirements:          colony.SpecItemDelta{ModifiedIDs: []string{"requirement-account"}},
		},
	}

	impact, err := computePlanImpactClosure(revision, plan)
	if err != nil {
		t.Fatal(err)
	}
	wantAffected := []string{
		"acceptance-account", "acceptance-notification", "negative-account", "negative-notification",
		"path-account", "path-notification", "phase-account", "phase-notification",
		"recovery-account", "recovery-notification", "requirement-account", "requirement-notification",
		"task-account", "task-notification",
	}
	if !reflect.DeepEqual(impact.AffectedSemanticIDs, wantAffected) {
		t.Fatalf("affected closure = %#v, want %#v", impact.AffectedSemanticIDs, wantAffected)
	}
	if containsPlanImpactID(impact.AffectedSemanticIDs, "task-independent") {
		t.Fatalf("independent completed task was invalidated: %#v", impact.AffectedSemanticIDs)
	}
	if !containsPlanImpactID(impact.PreservedSemanticIDs, "task-independent") {
		t.Fatalf("independent completed task was not preserved: %#v", impact.PreservedSemanticIDs)
	}
}

func TestPlanImpactAdditionAndRemovalAreDeterministic(t *testing.T) {
	taskID := "1.1"
	phases := []colony.Phase{{
		ID: 1, SemanticID: "phase-export",
		Tasks: []colony.Task{{
			ID: &taskID, SemanticID: "task-export",
			RequirementProofLinks: []string{"requirement-removed"},
			AcceptanceProofLinks:  []string{"acceptance-export"},
		}},
	}}
	revision := colony.SpecRevision{
		ID: "spec-revision-successor", ContentHash: strings.Repeat("b", 64),
		Delta: colony.SpecRevisionDelta{Requirements: colony.SpecItemDelta{
			AddedIDs: []string{"requirement-added"}, RemovedIDs: []string{"requirement-removed"},
		}},
	}

	first, err := computePlanImpactClosure(revision, phases)
	if err != nil {
		t.Fatal(err)
	}
	second, err := computePlanImpactClosure(revision, clonePhases(phases))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same semantic graph produced different impact\nfirst=%#v\nsecond=%#v", first, second)
	}
	for _, id := range []string{"requirement-added", "requirement-removed", "task-export", "phase-export", "acceptance-export"} {
		if !containsPlanImpactID(first.AffectedSemanticIDs, id) {
			t.Errorf("affected closure omits %q: %#v", id, first.AffectedSemanticIDs)
		}
	}
}

func TestPlanRevisionAffectedCandidateMustCoverExactClosure(t *testing.T) {
	taskID := "1.1"
	impact := planImpactClosure{AffectedSemanticIDs: []string{"requirement-account", "task-account", "task-dependent"}}
	candidate := colony.PlanCandidate{
		Proposal: colony.PlanRevision{
			Phases: []colony.Phase{{
				ID: 1, SemanticID: "phase-account", RequirementProofLinks: []string{"requirement-account"},
				Tasks: []colony.Task{{
					ID: &taskID, SemanticID: "task-account", RequirementProofLinks: []string{"requirement-account"},
				}},
			}},
		},
	}
	if err := validatePlanCandidateImpactCoverage(candidate, impact); err == nil || !strings.Contains(err.Error(), "task-dependent") {
		t.Fatalf("expected omitted affected ID refusal, got %v", err)
	}
	candidate.SemanticDelta.Tasks = []colony.PlanningSemanticChange{{SemanticID: "task-dependent", Kind: colony.PlanningSemanticChangeRemoved}}
	if err := validatePlanCandidateImpactCoverage(candidate, impact); err != nil {
		t.Fatalf("candidate with explicit affected removal rejected: %v", err)
	}
}

func TestPlanRevisionPreservesOnlyUnaffectedCompletedTasks(t *testing.T) {
	affectedID := "1.1"
	preservedID := "1.2"
	previous := []colony.Phase{{
		ID: 1, SemanticID: "phase-account", Status: colony.PhaseCompleted, WatcherFailureCount: 2,
		Tasks: []colony.Task{
			{ID: &affectedID, SemanticID: "task-account", Goal: "Old account contract", Status: colony.TaskCompleted},
			{ID: &preservedID, SemanticID: "task-independent", Goal: "Independent contract", Status: colony.TaskCompleted},
		},
	}}
	proposal := clonePhases(previous)
	proposal[0].Status = colony.PhaseReady
	proposal[0].WatcherFailureCount = 0
	proposal[0].Tasks[0].Goal = "Reconciled account contract"
	proposal[0].Tasks[0].Status = colony.TaskPending
	proposal[0].Tasks[1].Status = colony.TaskPending
	predecessor := colony.PlanRevision{ID: "plan-r1-predecessor", Phases: clonePhases(previous)}
	before, err := json.Marshal(predecessor)
	if err != nil {
		t.Fatal(err)
	}

	activated, preservedPhases, err := preserveCompletedCandidateWorkForImpact(previous, proposal, planImpactClosure{
		AffectedSemanticIDs:  []string{"phase-account", "task-account"},
		PreservedSemanticIDs: []string{"task-independent"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if activated[0].Status == colony.PhaseCompleted || activated[0].Tasks[0].Status == colony.TaskCompleted {
		t.Fatalf("affected completed work retained completion: %#v", activated[0])
	}
	if activated[0].Tasks[1].Status != colony.TaskCompleted {
		t.Fatalf("unaffected compatible task lost completion: %#v", activated[0].Tasks[1])
	}
	if len(preservedPhases) != 0 {
		t.Fatalf("partly affected phase reported as wholly preserved: %v", preservedPhases)
	}
	after, err := json.Marshal(predecessor)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("predecessor revision mutated\nbefore=%s\nafter=%s", before, after)
	}
}

func TestPlanImpactLifecycleClearsOnlyAfterExactSpecificationBinding(t *testing.T) {
	taskID := "1.1"
	currentSpec := colony.SpecRevision{
		ID: "spec-revision-current", ContentHash: strings.Repeat("c", 64), Status: colony.SpecStatusApproved,
		Delta: colony.SpecRevisionDelta{PredecessorRevisionID: "spec-revision-old", Requirements: colony.SpecItemDelta{ModifiedIDs: []string{"requirement-account"}}},
	}
	state := colony.ColonyState{
		Specification: &colony.Specification{CurrentRevisionID: currentSpec.ID, Revisions: []colony.SpecRevision{
			{ID: "spec-revision-old", ContentHash: strings.Repeat("d", 64)},
			currentSpec,
		}},
		Plan: colony.Plan{
			ActiveRevisionID: "plan-r1-old",
			Revisions: []colony.PlanRevision{{
				ID: "plan-r1-old", SpecificationRevisionID: "spec-revision-old", SpecificationRevisionHash: strings.Repeat("d", 64),
				Phases: []colony.Phase{{
					ID: 1, SemanticID: "phase-account",
					Tasks: []colony.Task{{ID: &taskID, SemanticID: "task-account", RequirementProofLinks: []string{"requirement-account"}}},
				}},
			}},
			Phases: []colony.Phase{{
				ID: 1, SemanticID: "phase-account",
				Tasks: []colony.Task{{ID: &taskID, SemanticID: "task-account", RequirementProofLinks: []string{"requirement-account"}}},
			}},
		},
	}

	got := lifecycleAffectedSemanticIDs(state, nil)
	for _, id := range []string{"requirement-account", "task-account", "phase-account"} {
		if !containsPlanImpactID(got, id) {
			t.Fatalf("unreconciled lifecycle scope omits %q: %v", id, got)
		}
	}
	state.Plan.Revisions[0].SpecificationRevisionID = currentSpec.ID
	state.Plan.Revisions[0].SpecificationRevisionHash = currentSpec.ContentHash
	state.Plan.Revisions[0].AffectedSemanticIDs = append([]string(nil), got...)
	if got := lifecycleAffectedSemanticIDs(state, nil); len(got) != 0 {
		t.Fatalf("historical impact stayed live after exact reconciliation: %v", got)
	}
}

func TestPlanImpactPlanningStateAdmitsApprovedSuccessorWithoutMutatingAcceptedPredecessor(t *testing.T) {
	state, _ := validCurrentPlanningState(t)
	activeIndex := len(state.Plan.Revisions) - 1
	before, err := json.Marshal(state.Plan.Revisions[activeIndex])
	if err != nil {
		t.Fatal(err)
	}
	predecessor, ok := currentSpecificationRevision(*state.Specification)
	if !ok {
		t.Fatal("missing current specification")
	}
	now := predecessor.CreatedAt.Add(10 * time.Minute)
	specification, successor, _, err := buildSpecificationSuccessor(*state.Specification, state.Plan, specificationRevisionRequest{
		PredecessorRevisionID: predecessor.ID, PredecessorContentHash: predecessor.ContentHash,
		Scope: predecessor.Scope, CreatedAt: now,
		Changes: []specificationRevisionChange{{
			Operation: specificationChangeModify, Section: specificationSectionRequirements,
			TargetID: predecessor.Requirements[0].ID,
			Item:     specificationItemInput{Description: "The plan is grounded in revised intent", EvidenceIDs: []string{"evidence-2"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	token := specificationApprovalToken(specification.ID, successor.ID, successor.ContentHash)
	approval, err := buildSpecificationApprovalReceipt(specification.ID, successor, specificationApprovalRequest{
		RevisionID: successor.ID, RevisionContentHash: successor.ContentHash, ApprovalToken: token,
		ApprovedBy: "owner", ApprovedAt: now.Add(time.Minute),
	}, specificationApprovalTokenHash(token))
	if err != nil {
		t.Fatal(err)
	}
	specification.Revisions[len(specification.Revisions)-1].Status = colony.SpecStatusApproved
	specification.Revisions[len(specification.Revisions)-1].Approval = &approval
	state.Specification = &specification

	if err := validatePlanningState(state); err != nil {
		t.Fatalf("approved successor should coexist with the immutable accepted predecessor while reconciliation is pending: %v", err)
	}
	after, err := json.Marshal(state.Plan.Revisions[activeIndex])
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("accepted predecessor mutated while admitting successor\nbefore=%s\nafter=%s", before, after)
	}
	if affected := lifecycleAffectedSemanticIDs(state, nil); !containsPlanImpactID(affected, predecessor.Requirements[0].ID) {
		t.Fatalf("approved successor did not become an affected lifecycle boundary: %v", affected)
	}
}

func containsPlanImpactID(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
