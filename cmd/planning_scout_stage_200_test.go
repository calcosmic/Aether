package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningScoutStageFinalizeCommitsReceiptWithoutCard(t *testing.T) {
	root, manifest, result := planningScoutStageTestFixture(t)

	completed, err := finalizePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if completed.Receipt.Caste != planningStageCasteScout || completed.Receipt.ResultingState != planningStageRouteReady {
		t.Fatalf("Scout receipt = %+v, want Scout -> route_ready", completed.Receipt)
	}
	if completed.Artifact.Path != completed.Receipt.OutputPath || completed.Artifact.ContentHash != completed.Receipt.OutputHash {
		t.Fatalf("Scout artifact is not bound by receipt: artifact=%+v receipt=%+v", completed.Artifact, completed.Receipt)
	}
	if len(completed.Result.Findings) != 1 || len(completed.Result.NewEvidence) != 1 || len(completed.Result.UnresolvedGaps) != 1 {
		t.Fatalf("normalized Scout result lost content: %+v", completed.Result)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(completed.Artifact.Path))); err != nil {
		t.Fatalf("Scout artifact was not persisted: %v", err)
	}
	chain, err := readPlanningStageReceiptChain(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.Receipts) != 1 || chain.Receipts[0].ID != completed.Receipt.ID || len(chain.Cards) != 0 {
		t.Fatalf("Scout boundary = %+v, want one receipt and no iteration card", chain)
	}
	state := planningStageReceiptTestReadState(t, root, manifest.RunID)
	if state.ScoutReceipt == nil || state.ScoutReceipt.ID != completed.Receipt.ID || state.ActiveManifestID != "" {
		t.Fatalf("Scout completion did not durably advance the stage: %+v", state)
	}
}

func TestPlanningScoutStageRejectsMismatchedAuthorityWithoutMutation(t *testing.T) {
	tests := []struct {
		name   string
		change func(*planningScoutStageResult)
		want   string
	}{
		{name: "manifest hash", change: func(result *planningScoutStageResult) { result.ManifestHash = planningStageTestHash("0") }, want: "manifest"},
		{name: "run", change: func(result *planningScoutStageResult) { result.RunID = "another-run" }, want: "run"},
		{name: "pass", change: func(result *planningScoutStageResult) { result.Pass++ }, want: "pass"},
		{name: "caste", change: func(result *planningScoutStageResult) { result.Caste = planningStageCasteRouteSetter }, want: "caste"},
		{name: "specification", change: func(result *planningScoutStageResult) { result.Specification.ContentHash = planningStageTestHash("0") }, want: "specification"},
		{name: "base revision", change: func(result *planningScoutStageResult) { result.BasePlanRevisionHash = planningStageTestHash("0") }, want: "base plan"},
		{name: "frontier", change: func(result *planningScoutStageResult) { result.InputFrontierHash = planningStageTestHash("0") }, want: "frontier"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, manifest, result := planningScoutStageTestFixture(t)
			test.change(&result)
			statePath := filepath.Join(root, filepath.FromSlash(planningStageStateRepositoryPath(manifest.RunID)))
			before := planningStageReceiptTestReadBytes(t, statePath)

			_, err := finalizePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), test.want) {
				t.Fatalf("error = %v, want authority mismatch containing %q", err, test.want)
			}
			if after := planningStageReceiptTestReadBytes(t, statePath); !bytes.Equal(after, before) {
				t.Fatal("rejected Scout result changed stage state")
			}
			planningScoutStageAssertNoResultWrites(t, root, manifest)
		})
	}
}

func TestPlanningScoutStageRejectsForbiddenAndUnknownFields(t *testing.T) {
	for _, field := range []string{
		"confidence", "dimension_scores", "semantic_delta", "stop_reason", "candidate",
		"accepted", "activation", "state_patch", "unexpected_worker_claim",
	} {
		t.Run(field, func(t *testing.T) {
			root, manifest, result := planningScoutStageTestFixture(t)
			var payload map[string]any
			if err := json.Unmarshal(planningScoutStageTestBytes(t, result), &payload); err != nil {
				t.Fatal(err)
			}
			payload[field] = true
			raw, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := finalizePlanningScoutStage(root, manifest, raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("Scout field %q was not rejected strictly: %v", field, err)
			}
			planningScoutStageAssertNoResultWrites(t, root, manifest)
		})
	}
}

func TestPlanningScoutStageRejectsUncitedFindingAndAcceptsExplicitUnknown(t *testing.T) {
	root, manifest, result := planningScoutStageTestFixture(t)
	result.Findings[0].EvidenceIDs = nil
	if _, err := finalizePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result)); err == nil || !strings.Contains(err.Error(), "evidence or an explicit unknown") {
		t.Fatalf("uncited finding error = %v", err)
	}
	planningScoutStageAssertNoResultWrites(t, root, manifest)

	result.Findings[0].Unknown = true
	result.Findings[0].UnknownReason = "The repository contains no authoritative retention period."
	completed, err := finalizePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	if err != nil {
		t.Fatalf("explicit unknown was rejected: %v", err)
	}
	if !completed.Result.Findings[0].Unknown || completed.Result.Findings[0].UnknownReason == "" {
		t.Fatalf("explicit unknown was not preserved: %+v", completed.Result.Findings[0])
	}
}

func TestPlanningScoutStageReplayReturnsOriginalReceiptAndRejectsDivergence(t *testing.T) {
	root, manifest, result := planningScoutStageTestFixture(t)
	raw := planningScoutStageTestBytes(t, result)
	first, err := finalizePlanningScoutStage(root, manifest, raw)
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(root, filepath.FromSlash(planningStageStateRepositoryPath(manifest.RunID)))
	beforeState := planningStageReceiptTestReadBytes(t, statePath)
	beforeArtifact := planningStageReceiptTestReadBytes(t, filepath.Join(root, filepath.FromSlash(first.Artifact.Path)))

	replayed, err := finalizePlanningScoutStage(root, manifest, raw)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Receipt.ID != first.Receipt.ID || replayed.Receipt.ContentHash != first.Receipt.ContentHash {
		t.Fatalf("exact replay changed receipt: first=%+v replay=%+v", first.Receipt, replayed.Receipt)
	}
	if after := planningStageReceiptTestReadBytes(t, statePath); !bytes.Equal(after, beforeState) {
		t.Fatal("exact replay changed planning frontier")
	}

	result.Findings[0].Summary = "Divergent finding for the same manifest"
	_, err = finalizePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	var conflict *planningStageReceiptConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("divergent replay error = %T %v, want planningStageReceiptConflictError", err, err)
	}
	if after := planningStageReceiptTestReadBytes(t, statePath); !bytes.Equal(after, beforeState) {
		t.Fatal("divergent replay changed planning frontier")
	}
	if after := planningStageReceiptTestReadBytes(t, filepath.Join(root, filepath.FromSlash(first.Artifact.Path))); !bytes.Equal(after, beforeArtifact) {
		t.Fatal("divergent replay changed original Scout artifact")
	}
}

func TestPlanningScoutStageFirstPassMaterialQuestionsPersistOneDecisionBatchBeforeRoute(t *testing.T) {
	root, manifest, result := planningScoutStageTestFixture(t)
	result.DecisionCandidates = []planningDecisionCandidate{
		planningScoutStageMaterialCandidate(result.NewEvidence[0].Reference, "decision-owner-authority"),
		planningScoutStageMaterialCandidate(result.NewEvidence[0].Reference, "decision-visible-behavior"),
	}

	coordinated, err := coordinatePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if coordinated.DecisionCheckpoint == nil || coordinated.RouteDispatch != nil {
		t.Fatalf("first-pass material boundary = %+v, want one decision checkpoint and no Route-Setter dispatch", coordinated)
	}
	checkpoint := coordinated.DecisionCheckpoint
	if len(checkpoint.Batch.Decisions) != 2 || len(checkpoint.Cards) != 2 {
		t.Fatalf("decision checkpoint = %+v, want one two-question batch with two cards", checkpoint)
	}
	if checkpoint.Batch.ScoutReceiptHash != coordinated.Scout.Receipt.ContentHash || checkpoint.FrontierReceiptHash != coordinated.Scout.Receipt.ContentHash {
		t.Fatalf("decision checkpoint does not bind the completed Scout receipt: %+v", checkpoint)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(planningScoutDecisionCheckpointRepositoryPath(manifest.RunID)))); err != nil {
		t.Fatalf("decision checkpoint was not persisted: %v", err)
	}
	state := planningStageReceiptTestReadState(t, root, manifest.RunID)
	if state.Stage != planningStageOwnerDecision || state.ActiveManifestID != "" || len(state.UsedAuthorizationIDs) != 1 {
		t.Fatalf("first-pass decision state = %+v, want owner_decision with no Route-Setter authorization", state)
	}
}

func TestPlanningScoutStageNoMaterialQuestionAuthorizesExactlyOneRouteSetter(t *testing.T) {
	root, manifest, result := planningScoutStageTestFixture(t)

	coordinated, err := coordinatePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if coordinated.DecisionCheckpoint != nil || coordinated.RouteDispatch == nil {
		t.Fatalf("no-material boundary = %+v, want one Route-Setter dispatch and no decision checkpoint", coordinated)
	}
	route := coordinated.RouteDispatch
	if route.ScoutReceipt.ID != coordinated.Scout.Receipt.ID || route.ScoutReceipt.ContentHash != coordinated.Scout.Receipt.ContentHash {
		t.Fatalf("Route-Setter authorization does not bind exact Scout receipt: %+v", route)
	}
	if route.Manifest.ExpectedCaste != planningStageCasteRouteSetter || route.Manifest.AuthorizationID != route.Authorization.ID {
		t.Fatalf("Route-Setter dispatch is not the exact authorized manifest: %+v", route)
	}
	state := planningStageReceiptTestReadState(t, root, manifest.RunID)
	if state.Stage != planningStageRouteRunning || state.ActiveManifestID != route.Manifest.ID || len(state.UsedAuthorizationIDs) != 2 {
		t.Fatalf("Route-Setter state = %+v, want one new authorization after Scout", state)
	}
}

func TestPlanningScoutStageCompletedDirectAnswersResumeExactRouteSetter(t *testing.T) {
	root, manifest, result := planningScoutStageTestFixture(t)
	result.DecisionCandidates = []planningDecisionCandidate{
		planningScoutStageMaterialCandidate(result.NewEvidence[0].Reference, "decision-owner-authority"),
		planningScoutStageMaterialCandidate(result.NewEvidence[0].Reference, "decision-visible-behavior"),
	}
	coordinated, err := coordinatePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := coordinated.DecisionCheckpoint
	answers := make([]planningScoutDecisionAnswer, 0, len(checkpoint.Cards))
	for _, card := range checkpoint.Cards {
		answers = append(answers, planningScoutDecisionAnswer{
			DecisionID: card.DecisionID,
			ChoiceID:   card.Choices[0].ID,
			Answer:     card.Choices[0].Label,
		})
	}
	token, err := buildPlanningScoutDecisionResumeToken(*checkpoint, answers)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := resumePlanningScoutDecision(root, manifest.RunID, token, time.Date(2026, time.September, 7, 19, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if resumed.RouteDispatch == nil || resumed.SuccessorSpecification != nil {
		t.Fatalf("direct answer resume = %+v, want Route-Setter and no successor specification", resumed)
	}
	if resumed.ResumeToken == nil || resumed.ResumeToken.ContentHash != token.ContentHash {
		t.Fatalf("resume did not preserve exact answer token: %+v", resumed.ResumeToken)
	}
	state := planningStageReceiptTestReadState(t, root, manifest.RunID)
	if state.Stage != planningStageRouteRunning || state.ActiveManifestID != resumed.RouteDispatch.Manifest.ID {
		t.Fatalf("direct answer state = %+v, want route_running", state)
	}
}

func TestPlanningScoutStageDecisionRejectsPartialAndStaleAnswersWithoutRoute(t *testing.T) {
	root, manifest, result := planningScoutStageTestFixture(t)
	result.DecisionCandidates = []planningDecisionCandidate{
		planningScoutStageMaterialCandidate(result.NewEvidence[0].Reference, "decision-owner-authority"),
		planningScoutStageMaterialCandidate(result.NewEvidence[0].Reference, "decision-visible-behavior"),
	}
	coordinated, err := coordinatePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := coordinated.DecisionCheckpoint
	partial := []planningScoutDecisionAnswer{{
		DecisionID: checkpoint.Cards[0].DecisionID,
		ChoiceID:   checkpoint.Cards[0].Choices[0].ID,
		Answer:     checkpoint.Cards[0].Choices[0].Label,
	}}
	if _, err := buildPlanningScoutDecisionResumeToken(*checkpoint, partial); err == nil || !strings.Contains(err.Error(), "every card") {
		t.Fatalf("partial decision error = %v", err)
	}

	answers := make([]planningScoutDecisionAnswer, 0, len(checkpoint.Cards))
	for _, card := range checkpoint.Cards {
		answers = append(answers, planningScoutDecisionAnswer{DecisionID: card.DecisionID, ChoiceID: card.Choices[0].ID, Answer: card.Choices[0].Label})
	}
	token, err := buildPlanningScoutDecisionResumeToken(*checkpoint, answers)
	if err != nil {
		t.Fatal(err)
	}
	staleBinding := planningDecisionResumeBindingFromToken(token)
	staleBinding.FrontierReceiptHash = planningStageTestHash("e")
	stale, err := issuePlanningDecisionResumeToken(staleBinding)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resumePlanningScoutDecision(root, manifest.RunID, stale, time.Date(2026, time.September, 7, 19, 5, 0, 0, time.UTC)); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale resume error = %v", err)
	}
	state := planningStageReceiptTestReadState(t, root, manifest.RunID)
	if state.Stage != planningStageOwnerDecision || state.ActiveManifestID != "" || len(state.UsedAuthorizationIDs) != 1 {
		t.Fatalf("rejected decision answer changed authority: %+v", state)
	}
}

func TestPlanningScoutStageLateMaterialRoutesBeforeOwnerPause(t *testing.T) {
	root, manifest, result := planningScoutStageTestFixtureAtPass(t, 2, planningStageTestState(planningStageScoutReady).Specification)
	result.DecisionCandidates = []planningDecisionCandidate{
		planningScoutStageMaterialCandidate(result.NewEvidence[0].Reference, "decision-late-risk"),
	}

	coordinated, err := coordinatePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if coordinated.DecisionCheckpoint != nil || coordinated.RouteDispatch == nil {
		t.Fatalf("late material boundary = %+v, want Route-Setter before any owner pause", coordinated)
	}
	carried := coordinated.RouteDispatch.MaterialDecisionCandidates
	if len(carried) != 1 || carried[0].StableID != "decision-late-risk" || len(carried[0].Evidence) != 1 || carried[0].Evidence[0].ID != result.NewEvidence[0].Reference.ID {
		t.Fatalf("late material evidence was not carried into exact Route authorization: %+v", carried)
	}
	state := planningStageReceiptTestReadState(t, root, manifest.RunID)
	if state.Stage != planningStageRouteRunning {
		t.Fatalf("late material Scout paused at %q, want route_running", state.Stage)
	}
}

func TestPlanningScoutStageContractAnswerCreatesSuccessorDraftAndBlocksRoute(t *testing.T) {
	root, manifest, result, targetID := planningScoutStageContractFixture(t)
	candidate := planningScoutStageMaterialCandidate(result.NewEvidence[0].Reference, "decision-change-requirement")
	candidate.Impact = planningDecisionContractImpact{Behavior: "Keep the approved requirement wording."}
	candidate.Choices = []planningDecisionChoice{{
		ID:                  "change-contract",
		Label:               "Change the approved requirement",
		Consequence:         "The owner-visible contract gains the selected behavior before planning continues.",
		Impact:              planningDecisionContractImpact{Behavior: "Require the selected owner-visible behavior."},
		AffectedSemanticIDs: []string{targetID},
	}}
	result.DecisionCandidates = []planningDecisionCandidate{candidate}

	coordinated, err := coordinatePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	card := coordinated.DecisionCheckpoint.Cards[0]
	token, err := buildPlanningScoutDecisionResumeToken(*coordinated.DecisionCheckpoint, []planningScoutDecisionAnswer{{
		DecisionID: card.DecisionID,
		ChoiceID:   card.Choices[0].ID,
		Answer:     card.Choices[0].Label,
	}})
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := resumePlanningScoutDecision(root, manifest.RunID, token, time.Date(2026, time.September, 7, 19, 10, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if resumed.SuccessorSpecification == nil || resumed.RouteDispatch != nil {
		t.Fatalf("contract answer = %+v, want successor draft and no Route-Setter", resumed)
	}
	successor := resumed.SuccessorSpecification.Revision
	if successor.Status != colony.SpecStatusDraft || successor.PredecessorID != manifest.Specification.RevisionID || successor.ID == manifest.Specification.RevisionID {
		t.Fatalf("successor specification = %+v, want distinct draft over approved predecessor", successor)
	}
	state := planningStageReceiptTestReadState(t, root, manifest.RunID)
	if state.Stage != planningStageSpecApprovalRequired || state.PendingSpecification == nil || state.PendingSpecification.RevisionID != successor.ID {
		t.Fatalf("contract answer state = %+v, want exact successor approval boundary", state)
	}
	if _, err := authorizePlanningScoutRoute(root, state, coordinated.DecisionCheckpoint.Batch.Decisions, &token); err == nil || !strings.Contains(err.Error(), "route_ready") {
		t.Fatalf("Route-Setter dispatch before exact approval and reconciliation error = %v", err)
	}
}

func planningScoutStageMaterialCandidate(evidence colony.PlanningEvidenceRef, stableID string) planningDecisionCandidate {
	impact := planningDecisionContractImpact{Behavior: "Preserve the approved behavior contract."}
	return planningDecisionCandidate{
		StableID:            stableID,
		Domain:              planningDecisionDomainBehavior,
		Decision:            "Should the approved behavior remain unchanged?",
		WhyNow:              "Scout evidence exposes a choice that changes owner-visible behavior.",
		Evidence:            []colony.PlanningEvidenceRef{evidence},
		QueenRecommendation: "Keep the approved behavior unchanged.",
		Impact:              impact,
		Choices: []planningDecisionChoice{{
			ID:          "keep-approved",
			Label:       "Keep approved behavior",
			Consequence: "Planning resumes without changing the approved contract.",
			Impact:      impact,
		}},
		ResumeInstruction: "Planning resumes at Route-Setter after every answer is bound.",
	}
}

func planningScoutStageTestFixtureAtPass(t *testing.T, pass int, binding planningStageSpecificationBinding) (string, planningStageManifest, planningScoutStageResult) {
	t.Helper()
	root := t.TempDir()
	return planningScoutStageTestFixtureInRoot(t, root, pass, binding)
}

func planningScoutStageTestFixtureInRoot(t *testing.T, root string, pass int, binding planningStageSpecificationBinding) (string, planningStageManifest, planningScoutStageResult) {
	t.Helper()
	state := planningStageTestState(planningStageScoutReady)
	state.Pass = pass
	state.Specification = binding
	authorization := planningStageAuthorization{
		ID:                fmt.Sprintf("authorization-scout-stage-%d", pass),
		ExpectedCaste:     planningStageCasteScout,
		InputFrontierHash: state.InputFrontierHash,
		EvidenceFrontier:  []planningStageEvidenceBinding{{ID: "evidence", ContentHash: planningStageTestHash("1")}},
		WeakestGap:        planningStageTestGap("receipt-gap"),
	}
	running, stageManifest, err := reducePlanningStage(state, planningStageTransition{To: planningStageScoutRunning, Authorization: &authorization})
	if err != nil {
		t.Fatal(err)
	}
	if stageManifest == nil {
		t.Fatal("Scout dispatch did not emit a manifest")
	}
	if err := recordPlanningStageDispatch(root, running, *stageManifest, planningStageWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	manifest := *stageManifest
	header := planningScoutStageTestHeader(t, manifest)
	planningStageReceiptTestWriteJSON(t, filepath.Join(root, ".aether", "data", "planning", manifest.RunID, "run-header.json"), header)

	record, err := normalizePlanningEvidence(planningEvidenceSource{
		Kind:    colony.PlanningEvidenceResearch,
		Origin:  fmt.Sprintf("scout:pass-%d:repository-observation", pass),
		Content: []byte("The lifecycle transaction already provides a durable Scout receipt boundary."),
		Scope: planningEvidenceScope{
			GoalID:                  header.GoalID,
			SessionID:               header.SessionID,
			SpecificationRevisionID: manifest.Specification.RevisionID,
			PlanRevisionID:          manifest.BasePlanRevisionID,
		},
		SourceRevision:       fmt.Sprintf("scout-result-revision-%d", pass),
		ObservedAt:           time.Date(2026, time.September, 7, 18, pass, 0, 0, time.UTC),
		ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
	})
	if err != nil {
		t.Fatal(err)
	}
	gap := *planningStageTestGap(fmt.Sprintf("scout-unresolved-gap-%d", pass))
	gap.EvidenceIDs = []string{record.Reference.ID}
	result := planningScoutStageResult{
		ResultType:           planningStageResultScout,
		ManifestID:           manifest.ID,
		ManifestHash:         manifest.ContentHash,
		RunID:                manifest.RunID,
		Pass:                 manifest.Pass,
		Caste:                planningStageCasteScout,
		Specification:        manifest.Specification,
		BasePlanRevisionID:   manifest.BasePlanRevisionID,
		BasePlanRevisionHash: manifest.BasePlanRevisionHash,
		InputFrontierHash:    manifest.InputFrontierHash,
		Findings: []planningScoutStageFinding{{
			StableID:    fmt.Sprintf("scout-finding-stage-boundary-%d", pass),
			Summary:     "The existing transaction can commit a Scout receipt before Route-Setter starts.",
			EvidenceIDs: []string{record.Reference.ID},
		}},
		NewEvidence:    []planningEvidenceRecord{record},
		UnresolvedGaps: []colony.PlanningGap{gap},
	}
	return root, manifest, result
}

func planningScoutStageContractFixture(t *testing.T) (string, planningStageManifest, planningScoutStageResult, string) {
	t.Helper()
	root := newSpecificationTestRepository(t, colony.ColonyState{})
	request := specificationTestDraftRequest(t, colony.SpecScopeWholeGoal)
	request.Scope.GoalID = "goal-200"
	request.Scope.SessionID = "session-200"
	draft, err := createSpecificationDraft(root, request, specificationMutationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := approveSpecification(root, specificationApprovalRequest{
		RevisionID:          draft.Revision.ID,
		RevisionContentHash: draft.Revision.ContentHash,
		ApprovalToken:       specificationApprovalToken(draft.Specification.ID, draft.Revision.ID, draft.Revision.ContentHash),
		ApprovedBy:          "owner",
		ApprovedAt:          request.CreatedAt.Add(time.Minute),
	}, specificationMutationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	approvalHash, err := jsonSHA256(*approved.Revision.Approval)
	if err != nil {
		t.Fatal(err)
	}
	binding := planningStageSpecificationBinding{
		RevisionID:          approved.Revision.ID,
		ContentHash:         approved.Revision.ContentHash,
		Status:              colony.SpecStatusApproved,
		ApprovalReceiptID:   approved.Revision.Approval.ID,
		ApprovalReceiptHash: approvalHash,
	}
	_, manifest, result := planningScoutStageTestFixtureInRoot(t, root, 1, binding)
	return root, manifest, result, approved.Revision.Requirements[0].ID
}

func planningScoutStageTestFixture(t *testing.T) (string, planningStageManifest, planningScoutStageResult) {
	t.Helper()
	root := t.TempDir()
	_, manifest := planningStageReceiptTestScoutDispatch(t, root)
	header := planningScoutStageTestHeader(t, manifest)
	planningStageReceiptTestWriteJSON(t, filepath.Join(root, ".aether", "data", "planning", manifest.RunID, "run-header.json"), header)

	record, err := normalizePlanningEvidence(planningEvidenceSource{
		Kind:    colony.PlanningEvidenceResearch,
		Origin:  "scout:pass-1:repository-observation",
		Content: []byte("The lifecycle transaction already provides a durable Scout receipt boundary."),
		Scope: planningEvidenceScope{
			GoalID:                  header.GoalID,
			SessionID:               header.SessionID,
			SpecificationRevisionID: manifest.Specification.RevisionID,
			PlanRevisionID:          manifest.BasePlanRevisionID,
		},
		SourceRevision:       "scout-result-revision-1",
		ObservedAt:           time.Date(2026, time.September, 7, 18, 0, 0, 0, time.UTC),
		ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
	})
	if err != nil {
		t.Fatal(err)
	}
	gap := *planningStageTestGap("scout-unresolved-gap")
	gap.EvidenceIDs = []string{record.Reference.ID}
	return root, manifest, planningScoutStageResult{
		ResultType:           planningStageResultScout,
		ManifestID:           manifest.ID,
		ManifestHash:         manifest.ContentHash,
		RunID:                manifest.RunID,
		Pass:                 manifest.Pass,
		Caste:                planningStageCasteScout,
		Specification:        manifest.Specification,
		BasePlanRevisionID:   manifest.BasePlanRevisionID,
		BasePlanRevisionHash: manifest.BasePlanRevisionHash,
		InputFrontierHash:    manifest.InputFrontierHash,
		Findings: []planningScoutStageFinding{{
			StableID:    "scout-finding-stage-boundary",
			Summary:     "The existing transaction can commit a Scout receipt before Route-Setter starts.",
			EvidenceIDs: []string{record.Reference.ID},
		}},
		NewEvidence:    []planningEvidenceRecord{record},
		UnresolvedGaps: []colony.PlanningGap{gap},
	}
}

func planningScoutStageTestHeader(t *testing.T, manifest planningStageManifest) planningRunHeader {
	t.Helper()
	header := planningRunHeader{
		SchemaVersion:        planningRunHeaderSchemaVersion,
		RunID:                manifest.RunID,
		Goal:                 "Restore visible staged planning",
		GoalID:               "goal-200",
		SessionID:            "session-200",
		Specification:        manifest.Specification,
		BasePlanRevisionID:   manifest.BasePlanRevisionID,
		BasePlanRevisionHash: manifest.BasePlanRevisionHash,
		Preset:               manifest.Preset,
		TargetConfidence:     90,
		PassCap:              6,
		EvidenceFrontier:     append([]planningStageEvidenceBinding(nil), manifest.EvidenceFrontier...),
		InputFrontierHash:    manifest.InputFrontierHash,
		ResearchPolicy: phaseResearchAutomaticPolicy{
			SchemaVersion:         phaseResearchAutomaticPolicySchemaVersion,
			Preset:                manifest.Preset,
			OwnerDecisionBoundary: "after_scout",
			EvidenceContract:      automaticPhaseResearchEvidenceContract(),
		},
		WeakestGap:        *manifest.WeakestGap,
		StageManifestID:   manifest.ID,
		StageManifestHash: manifest.ContentHash,
		CreatedAt:         time.Date(2026, time.September, 7, 17, 0, 0, 0, time.UTC),
	}
	payload := header
	payload.ID = ""
	payload.ContentHash = ""
	hash, err := jsonSHA256(payload)
	if err != nil {
		t.Fatal(err)
	}
	header.ContentHash = hash
	header.ID = "planning-run-header-" + hash[:16]
	return header
}

func planningScoutStageTestBytes(t *testing.T, result planningScoutStageResult) []byte {
	t.Helper()
	content, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func planningScoutStageAssertNoResultWrites(t *testing.T, root string, manifest planningStageManifest) {
	t.Helper()
	for _, path := range []string{
		planningStageOutputRepositoryPath(manifest),
		planningStageReceiptIndexRepositoryPath(manifest.RunID),
	} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); !os.IsNotExist(err) {
			t.Fatalf("rejected Scout result wrote %s: %v", path, err)
		}
	}
}
