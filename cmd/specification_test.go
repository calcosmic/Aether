package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestSpecificationDraftCreatesCompleteCanonicalLineagesAndReplaysExactly(t *testing.T) {
	for _, scopeKind := range []colony.SpecScopeKind{colony.SpecScopeWholeGoal, colony.SpecScopeFeature} {
		t.Run(string(scopeKind), func(t *testing.T) {
			root := newSpecificationTestRepository(t, colony.ColonyState{})
			request := specificationTestDraftRequest(t, scopeKind)

			first, err := createSpecificationDraft(root, request, specificationMutationOptions{})
			if err != nil {
				t.Fatalf("create specification draft: %v", err)
			}
			if first.Replayed {
				t.Fatal("first draft creation reported replay")
			}
			assertSpecificationTestBodyComplete(t, first.Revision)
			if first.Revision.Status != colony.SpecStatusDraft || first.Revision.Approval != nil {
				t.Fatalf("new revision authority = %q/%#v, want draft with no approval", first.Revision.Status, first.Revision.Approval)
			}
			if first.Revision.Scope.Kind != scopeKind {
				t.Fatalf("scope = %q, want %q", first.Revision.Scope.Kind, scopeKind)
			}
			if err := validateSpecificationState(first.Specification); err != nil {
				t.Fatalf("draft does not satisfy current specification validation: %v", err)
			}

			stateAfterFirst := mustReadSpecificationTestState(t, root)
			if stateAfterFirst.Specification == nil || !reflect.DeepEqual(*stateAfterFirst.Specification, first.Specification) {
				t.Fatalf("canonical state does not contain committed draft\nstate:  %#v\nresult: %#v", stateAfterFirst.Specification, first.Specification)
			}

			second, err := createSpecificationDraft(root, request, specificationMutationOptions{})
			if err != nil {
				t.Fatalf("replay specification draft: %v", err)
			}
			if !second.Replayed {
				t.Fatal("exact draft retry did not report replay")
			}
			if !reflect.DeepEqual(first.Revision, second.Revision) || !reflect.DeepEqual(first.Receipt, second.Receipt) {
				t.Fatalf("exact retry changed revision or receipt\nfirst:  %#v\nsecond: %#v", first, second)
			}
			if got := len(mustReadSpecificationTestState(t, root).Specification.Revisions); got != 1 {
				t.Fatalf("exact retry appended %d revisions, want 1", got)
			}
		})
	}
}

func TestSpecificationReviseClassifiesChangesAndPreservesUnaffectedIdentity(t *testing.T) {
	root := newSpecificationTestRepository(t, specificationTestPlanState())
	draftRequest := specificationTestDraftRequest(t, colony.SpecScopeWholeGoal)
	draftRequest.Exclusions = append(draftRequest.Exclusions, specificationItemInput{
		Lineage:     "future-marketplace",
		Description: "Do not add a marketplace in this goal.",
		EvidenceIDs: []string{"charter:scope"},
	})
	draft, err := createSpecificationDraft(root, draftRequest, specificationMutationOptions{})
	if err != nil {
		t.Fatalf("create predecessor: %v", err)
	}

	requirementID := draft.Revision.Requirements[0].ID
	removedID := draft.Revision.Exclusions[0].ID
	unaffectedBefore := draft.Revision.RecoveryExpectations[0]
	decision := specificationTestSuccessorDecision(requirementID)
	revisionRequest := specificationRevisionRequest{
		PredecessorRevisionID:  draft.Revision.ID,
		PredecessorContentHash: draft.Revision.ContentHash,
		Scope: colony.SpecScope{
			Kind:               colony.SpecScopeFeature,
			GoalID:             draftRequest.Scope.GoalID,
			SessionID:          "session-spec-revision",
			FeatureID:          "feature-visible-planning",
			RequirementIDs:     []string{requirementID},
			AcceptanceCheckIDs: []string{draft.Revision.AcceptanceChecks[0].ID},
		},
		Changes: []specificationRevisionChange{
			{
				Operation: specificationChangeModify,
				Section:   specificationSectionRequirements,
				TargetID:  requirementID,
				Item: specificationItemInput{
					Description: "The owner sees a causal Scout to Route-Setter planning loop.",
					EvidenceIDs: []string{"owner-answer:visible-loop", "research:classic-loop"},
				},
			},
			{
				Operation: specificationChangeAdd,
				Section:   specificationSectionIncludedBehaviors,
				Item: specificationItemInput{
					Lineage:     "show-weakest-gap",
					Description: "Show the weakest remaining planning gap after each pass.",
					EvidenceIDs: []string{"research:classic-loop"},
				},
			},
			{
				Operation: specificationChangeRemove,
				Section:   specificationSectionExclusions,
				TargetID:  removedID,
			},
		},
		DecisionResolution: &decision,
		CreatedAt:          time.Date(2026, time.September, 7, 16, 0, 0, 0, time.UTC),
	}
	planBefore := mustReadSpecificationTestState(t, root).Plan

	first, err := reviseSpecification(root, revisionRequest, specificationMutationOptions{})
	if err != nil {
		t.Fatalf("revise specification: %v", err)
	}
	if first.Revision.PredecessorID != draft.Revision.ID || first.Revision.Delta.PredecessorRevisionID != draft.Revision.ID {
		t.Fatalf("successor predecessor binding = %#v", first.Revision)
	}
	if !containsLifecycleString(first.Revision.Delta.Requirements.ModifiedIDs, requirementID) {
		t.Fatalf("requirement delta omitted modification: %#v", first.Revision.Delta.Requirements)
	}
	if len(first.Revision.Delta.IncludedBehaviors.AddedIDs) != 1 {
		t.Fatalf("included-behavior delta = %#v, want one add", first.Revision.Delta.IncludedBehaviors)
	}
	if !containsLifecycleString(first.Revision.Delta.Exclusions.RemovedIDs, removedID) {
		t.Fatalf("exclusion delta omitted removal: %#v", first.Revision.Delta.Exclusions)
	}
	if !reflect.DeepEqual(first.Revision.RecoveryExpectations[0], unaffectedBefore) {
		t.Fatalf("unaffected item changed\nbefore: %#v\nafter:  %#v", unaffectedBefore, first.Revision.RecoveryExpectations[0])
	}
	if len(first.Specification.Revisions) != 2 || first.Specification.Revisions[0].Status != colony.SpecStatusSuperseded || first.Revision.Status != colony.SpecStatusDraft {
		t.Fatalf("lineage statuses = %#v", first.Specification.Revisions)
	}
	if !reflect.DeepEqual(mustReadSpecificationTestState(t, root).Plan, planBefore) {
		t.Fatal("specification revision mutated the active plan")
	}
	if !reflect.DeepEqual(first.AffectedScope.RequirementIDs, []string{requirementID}) ||
		!reflect.DeepEqual(first.AffectedScope.TaskIDs, []string{"task-visible-loop"}) ||
		!reflect.DeepEqual(first.AffectedScope.ProofLinkIDs, []string{requirementID}) {
		t.Fatalf("affected closure = %#v, want only linked requirement/task/proof", first.AffectedScope)
	}

	replayed, err := reviseSpecification(root, revisionRequest, specificationMutationOptions{})
	if err != nil {
		t.Fatalf("replay revision: %v", err)
	}
	if !replayed.Replayed || !reflect.DeepEqual(replayed.Revision, first.Revision) || !reflect.DeepEqual(replayed.Receipt, first.Receipt) {
		t.Fatalf("exact revision retry was not idempotent\nfirst:  %#v\nreplay: %#v", first, replayed)
	}
}

func TestSpecificationReplayRejectsStaleOrDivergentRequestsWithoutMutation(t *testing.T) {
	root := newSpecificationTestRepository(t, colony.ColonyState{})
	request := specificationTestDraftRequest(t, colony.SpecScopeWholeGoal)
	draft, err := createSpecificationDraft(root, request, specificationMutationOptions{})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	t.Run("divergent draft", func(t *testing.T) {
		before := mustReadSpecificationTestStateBytes(t, root)
		divergent := request
		divergent.Outcomes = append([]specificationItemInput(nil), request.Outcomes...)
		divergent.Outcomes[0].Description = "A different goal outcome."
		if _, err := createSpecificationDraft(root, divergent, specificationMutationOptions{}); err == nil || !strings.Contains(err.Error(), "divergent") {
			t.Fatalf("divergent replay error = %v", err)
		}
		if after := mustReadSpecificationTestStateBytes(t, root); !bytes.Equal(before, after) {
			t.Fatal("divergent draft retry changed canonical state")
		}
	})

	t.Run("stale predecessor", func(t *testing.T) {
		before := mustReadSpecificationTestStateBytes(t, root)
		change := specificationRevisionRequest{
			PredecessorRevisionID:  draft.Revision.ID,
			PredecessorContentHash: strings.Repeat("0", 64),
			Scope:                  draft.Revision.Scope,
			Changes: []specificationRevisionChange{{
				Operation: specificationChangeModify,
				Section:   specificationSectionOutcomes,
				TargetID:  draft.Revision.Outcomes[0].ID,
				Item: specificationItemInput{
					Description: "Changed outcome.",
					EvidenceIDs: []string{"owner-answer:changed"},
				},
			}},
			CreatedAt: time.Date(2026, time.September, 7, 16, 10, 0, 0, time.UTC),
		}
		if _, err := reviseSpecification(root, change, specificationMutationOptions{}); err == nil || !strings.Contains(err.Error(), "stale predecessor") {
			t.Fatalf("stale predecessor error = %v", err)
		}
		if after := mustReadSpecificationTestStateBytes(t, root); !bytes.Equal(before, after) {
			t.Fatal("stale predecessor changed canonical state")
		}
	})

	t.Run("ambiguous removal", func(t *testing.T) {
		before := mustReadSpecificationTestStateBytes(t, root)
		change := specificationRevisionRequest{
			PredecessorRevisionID:  draft.Revision.ID,
			PredecessorContentHash: draft.Revision.ContentHash,
			Scope:                  draft.Revision.Scope,
			Changes: []specificationRevisionChange{
				{Operation: specificationChangeRemove, Section: specificationSectionOutcomes, TargetID: draft.Revision.Outcomes[0].ID},
				{Operation: specificationChangeRemove, Section: specificationSectionOutcomes, TargetID: draft.Revision.Outcomes[0].ID},
			},
			CreatedAt: time.Date(2026, time.September, 7, 16, 20, 0, 0, time.UTC),
		}
		if _, err := reviseSpecification(root, change, specificationMutationOptions{}); err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("ambiguous removal error = %v", err)
		}
		if after := mustReadSpecificationTestStateBytes(t, root); !bytes.Equal(before, after) {
			t.Fatal("ambiguous removal changed canonical state")
		}
	})

	t.Run("missing category", func(t *testing.T) {
		emptyRoot := newSpecificationTestRepository(t, colony.ColonyState{})
		invalid := request
		invalid.RecoveryExpectations = nil
		before := mustReadSpecificationTestStateBytes(t, emptyRoot)
		if _, err := createSpecificationDraft(emptyRoot, invalid, specificationMutationOptions{}); err == nil || !strings.Contains(err.Error(), "recovery_expectations") {
			t.Fatalf("missing category error = %v", err)
		}
		if after := mustReadSpecificationTestStateBytes(t, emptyRoot); !bytes.Equal(before, after) {
			t.Fatal("missing-category draft changed canonical state")
		}
	})
}

func specificationTestDraftRequest(t *testing.T, kind colony.SpecScopeKind) specificationDraftRequest {
	t.Helper()
	scope := colony.SpecScope{Kind: kind, GoalID: "goal-iterative-planning", SessionID: "session-spec-draft"}
	if kind == colony.SpecScopeFeature {
		requirementID, err := specificationStableID(specificationSectionRequirements, "causal-planning-loop")
		if err != nil {
			t.Fatal(err)
		}
		acceptanceID, err := specificationStableID(specificationSectionAcceptanceChecks, "owner-sees-evidence")
		if err != nil {
			t.Fatal(err)
		}
		scope.FeatureID = "feature-visible-planning"
		scope.RequirementIDs = []string{requirementID}
		scope.AcceptanceCheckIDs = []string{acceptanceID}
	}
	return specificationDraftRequest{
		Scope: scope,
		Outcomes: []specificationItemInput{{
			Lineage: "understand-improved-plan", Description: "The owner understands how evidence improved the plan.", EvidenceIDs: []string{"charter:goal"},
		}},
		IncludedBehaviors: []specificationItemInput{{
			Lineage: "scout-route-loop", Description: "Scout investigates and Route-Setter improves the route.", EvidenceIDs: []string{"decision:D-01"},
		}},
		Exclusions: []specificationItemInput{{
			Lineage: "no-build-cycle-change", Description: "Do not change the build worker cycle.", EvidenceIDs: []string{"charter:scope"},
		}},
		BindingDecisions: []specificationItemInput{{
			Lineage: "explicit-authority", Description: "Only the owner approves the specification.", EvidenceIDs: []string{"decision:D-10"},
		}},
		Requirements: []specificationItemInput{{
			Lineage: "causal-planning-loop", Description: "Every planning pass explains evidence and change.", EvidenceIDs: []string{"requirement:PLAN-05"},
		}},
		AcceptanceChecks: []specificationItemInput{{
			Lineage: "owner-sees-evidence", Description: "The owner can inspect the evidence behind a pass.", Verification: "Run the focused planning journey fixture.", EvidenceIDs: []string{"acceptance:V-200-E2E-01"},
		}},
		NegativeExpectations: []specificationItemInput{{
			Lineage: "no-inferred-approval", Description: "File presence never grants approval.", EvidenceIDs: []string{"decision:D-10"},
		}},
		RecoveryExpectations: []specificationItemInput{{
			Lineage: "exact-replay", Description: "An exact retry returns the original receipt.", EvidenceIDs: []string{"decision:D-12"},
		}},
		AffectedPublicPaths: []specificationItemInput{{
			Lineage: "ant-spec", Path: "/ant-spec", Description: "Review and approve the owner-readable contract.", EvidenceIDs: []string{"decision:D-09"},
		}},
		CreatedAt: time.Date(2026, time.September, 7, 15, 30, 0, 0, time.UTC),
	}
}

func specificationTestPlanState() colony.ColonyState {
	linkedID, _ := specificationStableID(specificationSectionRequirements, "causal-planning-loop")
	otherID, _ := specificationStableID(specificationSectionRequirements, "unrelated-requirement")
	linkedTaskID := "1.1"
	otherTaskID := "1.2"
	return colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{{
		ID: 1,
		Tasks: []colony.Task{
			{ID: &linkedTaskID, SemanticID: "task-visible-loop", RequirementProofLinks: []string{linkedID}},
			{ID: &otherTaskID, SemanticID: "task-unrelated", RequirementProofLinks: []string{otherID}},
		},
	}}}}
}

func specificationTestSuccessorDecision(affectedID string) planningDecisionResolution {
	return planningDecisionResolution{
		Disposition:         planningDecisionDispositionSuccessorSpecRequired,
		AffectedSemanticIDs: []string{affectedID},
		RevisionEvidence: []planningDecisionRevisionEvidence{{
			Dimension: "behavior", ApprovedValue: "opaque plan", SelectedValue: "causal visible loop", AffectedSemanticIDs: []string{affectedID},
		}},
	}
}

func newSpecificationTestRepository(t *testing.T, state colony.ColonyState) string {
	t.Helper()
	root := t.TempDir()
	dataRoot := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		t.Fatalf("create data root: %v", err)
	}
	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(filepath.Join(dataRoot, "COLONY_STATE.json"), content, 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}
	return root
}

func mustReadSpecificationTestState(t *testing.T, root string) colony.ColonyState {
	t.Helper()
	content := mustReadSpecificationTestStateBytes(t, root)
	var state colony.ColonyState
	if err := json.Unmarshal(content, &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	return state
}

func mustReadSpecificationTestStateBytes(t *testing.T, root string) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, ".aether", "data", "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	return content
}

func assertSpecificationTestBodyComplete(t *testing.T, revision colony.SpecRevision) {
	t.Helper()
	lengths := map[string]int{
		"outcomes":              len(revision.Outcomes),
		"included_behaviors":    len(revision.IncludedBehaviors),
		"exclusions":            len(revision.Exclusions),
		"binding_decisions":     len(revision.BindingDecisions),
		"requirements":          len(revision.Requirements),
		"acceptance_checks":     len(revision.AcceptanceChecks),
		"negative_expectations": len(revision.NegativeExpectations),
		"recovery_expectations": len(revision.RecoveryExpectations),
		"affected_public_paths": len(revision.AffectedPublicPaths),
	}
	for section, length := range lengths {
		if length == 0 {
			t.Fatalf("%s is empty", section)
		}
	}
}
