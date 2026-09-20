package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningRouteStageValidateExactProposalContract(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)

	validated, err := validatePlanningRouteStageResult(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if len(validated.ProposalHash) != 64 || validated.ProposalHash != validated.ProposalSnapshot.ContentHash {
		t.Fatalf("validated proposal hash = %q snapshot=%q, want one canonical SHA-256 address", validated.ProposalHash, validated.ProposalSnapshot.ContentHash)
	}
	if len(validated.Confidence.Assessments) != len(colony.PlanningDimensions()) {
		t.Fatalf("validated assessments = %d, want exactly five", len(validated.Confidence.Assessments))
	}
	if validated.ScoutReceipt.ID != manifest.ScoutReceipt.ID || validated.ScoutReceipt.ContentHash != manifest.ScoutReceipt.ContentHash {
		t.Fatalf("validated Scout receipt = %+v, want exact manifest receipt %+v", validated.ScoutReceipt, manifest.ScoutReceipt)
	}
	for _, section := range planningRouteStageTestDeltaSections(validated.SemanticDelta) {
		for _, change := range section {
			if change.Kind != colony.PlanningSemanticChangePreserved && len(change.EvidenceIDs) == 0 {
				t.Fatalf("semantic change %q has no current-frontier evidence: %+v", change.SemanticID, change)
			}
		}
	}
}

func TestPlanningRouteStageRejectsStaleBindingsAndForbiddenAuthority(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*planningRouteStageResult)
		wantErr string
	}{
		{
			name: "wrong Scout receipt",
			mutate: func(result *planningRouteStageResult) {
				result.ScoutReceipt.ContentHash = planningStageTestHash("9")
			},
			wantErr: "Scout receipt",
		},
		{
			name: "stale base revision",
			mutate: func(result *planningRouteStageResult) {
				result.BasePlanRevisionHash = planningStageTestHash("8")
			},
			wantErr: "base plan revision",
		},
		{
			name: "stale specification",
			mutate: func(result *planningRouteStageResult) {
				result.Specification.ContentHash = planningStageTestHash("7")
			},
			wantErr: "specification",
		},
		{
			name: "wrong caste",
			mutate: func(result *planningRouteStageResult) {
				result.Caste = planningStageCasteScout
			},
			wantErr: "caste",
		},
		{
			name: "missing recovery proof",
			mutate: func(result *planningRouteStageResult) {
				result.Proposal.Phases[0].Tasks[0].RecoveryProofLinks = nil
			},
			wantErr: "recovery_proof_links",
		},
		{
			name: "missing PLAN-05 public path proof",
			mutate: func(result *planningRouteStageResult) {
				result.Proposal.Phases[0].Tasks[0].PublicPathProofLinks = nil
			},
			wantErr: "public_path_proof_links",
		},
		{
			name: "incomplete automated checks",
			mutate: func(result *planningRouteStageResult) {
				result.Proposal.Phases[0].Tasks[0].EvidenceRequirements[0].Checks = nil
			},
			wantErr: "checks",
		},
		{
			name: "worker supplied phase authority",
			mutate: func(result *planningRouteStageResult) {
				result.Proposal.Phases[0].Status = colony.PhaseCompleted
			},
			wantErr: "status",
		},
		{
			name: "worker supplied overall",
			mutate: func(result *planningRouteStageResult) {
				overall := 99
				result.SuppliedOverall = &overall
			},
			wantErr: "supplied overall",
		},
		{
			name: "missing causal gap field",
			mutate: func(result *planningRouteStageResult) {
				result.DimensionAssessments[0].RemainingGap.EvidenceThatWouldChange = ""
			},
			wantErr: "evidence_that_would_change",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, manifest, result := planningRouteStageTestFixture(t)
			test.mutate(&result)
			if _, err := validatePlanningRouteStageResult(root, manifest, planningRouteStageTestBytes(t, result)); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("validation error = %v, want %q", err, test.wantErr)
			}
		})
	}

	t.Run("unknown activation field", func(t *testing.T) {
		root, manifest, result := planningRouteStageTestFixture(t)
		var payload map[string]interface{}
		if err := json.Unmarshal(planningRouteStageTestBytes(t, result), &payload); err != nil {
			t.Fatal(err)
		}
		payload["activation"] = map[string]interface{}{"status": "accepted"}
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := validatePlanningRouteStageResult(root, manifest, raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("activation validation error = %v, want strict unknown-field refusal", err)
		}
	})
}

func TestPlanningRouteStageNormalizeFormattingKeepsProposalHash(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)
	first, err := validatePlanningRouteStageResult(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}

	reformatted := planningRouteStageCloneResult(t, result)
	reformatted.Proposal.Phases[0].Name = "  Route   finalization  "
	reformatted.Proposal.Phases[0].Description = "Finalize\n the   exact Route proposal"
	reformatted.Proposal.Phases[0].Tasks[0].Goal = " Validate   the Route\n proposal "
	reformatted.Proposal.TaskDeclarations[0].Files = []string{"cmd/planning_route_stage_200_test.go", "cmd/codex_plan_finalize.go"}
	reformatted.ProposalEvidenceIDs = []string{result.ProposalEvidenceIDs[0], result.ProposalEvidenceIDs[0]}
	second, err := validatePlanningRouteStageResult(root, manifest, planningRouteStageTestBytes(t, reformatted))
	if err != nil {
		t.Fatal(err)
	}
	if first.ProposalHash != second.ProposalHash {
		t.Fatalf("formatting-only proposal changed canonical hash: %s != %s", first.ProposalHash, second.ProposalHash)
	}
}

func TestPlanningRouteStageCardDerivesConfidenceAndSemanticDelta(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)
	completed, err := finalizePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if completed.Receipt.ResultingState != planningStageContinueReady || completed.Card.Decision.Reason != colony.PlanningStopContinue {
		t.Fatalf("Route completion = receipt:%+v decision:%+v, want a Go-derived continue boundary", completed.Receipt, completed.Card.Decision)
	}
	if completed.Card.RouteSetterReceiptID != completed.Receipt.ID || completed.Card.RouteSetterReceiptHash != completed.Receipt.ContentHash {
		t.Fatalf("card does not bind exact Route receipt: card=%+v receipt=%+v", completed.Card, completed.Receipt)
	}
	if completed.Card.ScoutReceiptID != manifest.ScoutReceipt.ID || completed.Card.ScoutReceiptHash != manifest.ScoutReceipt.ContentHash {
		t.Fatalf("card does not bind exact Scout receipt: %+v", completed.Card)
	}
	if completed.StopPolicy.Decision.ContentHash != completed.Card.Decision.ContentHash || completed.Card.SemanticDelta.ContentHash != completed.Validation.SemanticDelta.ContentHash {
		t.Fatalf("card did not preserve Go-derived stop/delta truth: %+v", completed.Card)
	}
	scores := planningConfidenceScores{}
	for _, assessment := range completed.Card.DimensionAssessments {
		scores.Set(assessment.Dimension, assessment.After)
	}
	if scores.Overall != completed.StopPolicy.Diagnostics[0].Overall {
		t.Fatalf("derived overall = %d diagnostic=%d", scores.Overall, completed.StopPolicy.Diagnostics[0].Overall)
	}
	timeline, err := loadPlanningTimeline(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline.Cards) != 1 || timeline.Cards[0].ContentHash != completed.Card.ContentHash {
		t.Fatalf("timeline cards = %+v, want the exact completed card", timeline.Cards)
	}
}

func TestPlanningRouteStageAtomicCardReceiptAndTimeline(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)
	crash := errors.New("simulated Route card commit interruption")
	_, err := finalizePlanningRouteStageWithOptions(root, manifest, planningRouteStageTestBytes(t, result), planningRouteStageFinalizeOptions{
		Fault: func(point string) error {
			if point == "after_intent" {
				return crash
			}
			return nil
		},
	})
	if !errors.Is(err, crash) {
		t.Fatalf("interrupted Route finalizer error = %v, want injected crash", err)
	}
	if _, statErr := os.Lstat(filepath.Join(root, filepath.FromSlash(planningTimelineIndexRepositoryPath(manifest.RunID)))); !os.IsNotExist(statErr) {
		t.Fatalf("timeline index exists before atomic Route commit: %v", statErr)
	}
	chain, err := readPlanningStageReceiptChain(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.Receipts) != 1 || chain.Receipts[0].Caste != planningStageCasteScout {
		t.Fatalf("partial Route receipt escaped atomic boundary: %+v", chain.Receipts)
	}
	state, err := loadPlanningStageState(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Stage != planningStageRouteRunning || state.ActiveManifestID != manifest.ID {
		t.Fatalf("interrupted Route finalizer advanced state: %+v", state)
	}

	completed, err := finalizePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if completed.Card.ID == "" || completed.Receipt.ID == "" {
		t.Fatalf("recovered Route finalization is incomplete: %+v", completed)
	}
}

func TestPlanningRouteStageReplayReturnsExactCardAndReceipt(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)
	raw := planningRouteStageTestBytes(t, result)
	first, err := finalizePlanningRouteStage(root, manifest, raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := finalizePlanningRouteStage(root, manifest, raw)
	if err != nil {
		t.Fatal(err)
	}
	if first.Receipt.ID != second.Receipt.ID || first.Receipt.ContentHash != second.Receipt.ContentHash || first.Card.ID != second.Card.ID || first.Card.ContentHash != second.Card.ContentHash {
		t.Fatalf("exact replay diverged: first=%+v second=%+v", first, second)
	}
	chain, err := readPlanningStageReceiptChain(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.Receipts) != 2 || len(chain.Cards) != 1 {
		t.Fatalf("exact replay duplicated Route history: receipts=%d cards=%d", len(chain.Receipts), len(chain.Cards))
	}
}

func TestPlanningRouteStageContinueDispatchesScoutAtWeakestGap(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)
	coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if coordinated.ScoutDispatch == nil || coordinated.Candidate != nil || coordinated.DecisionCheckpoint != nil {
		t.Fatalf("continue coordination = %+v, want exactly one next Scout dispatch", coordinated)
	}
	dispatch := coordinated.ScoutDispatch
	if dispatch.ProposalHash != coordinated.Route.Validation.ProposalHash || dispatch.PriorCardHash != coordinated.Route.Card.ContentHash {
		t.Fatalf("next Scout bindings = %+v, want proposal %s and card %s", dispatch, coordinated.Route.Validation.ProposalHash, coordinated.Route.Card.ContentHash)
	}
	if dispatch.Manifest.ExpectedCaste != planningStageCasteScout || dispatch.Manifest.Pass != manifest.Pass+1 || dispatch.Manifest.WeakestGap == nil ||
		dispatch.Manifest.WeakestGap.ContentHash != coordinated.Route.Card.WeakestGap.ContentHash ||
		dispatch.Manifest.WeakestGap.EvidenceThatWouldChange != coordinated.Route.Card.EvidenceThatWouldChange {
		t.Fatalf("next Scout manifest does not target the exact weakest causal gap: %+v", dispatch.Manifest)
	}
	state, err := loadPlanningStageState(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Stage != planningStageScoutRunning || state.ActiveManifestID != dispatch.Manifest.ID {
		t.Fatalf("continued state = %+v, want one active Scout manifest", state)
	}
}

func TestPlanningRouteStageStopPersistsNonActiveCandidate(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)
	planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
	before := mustReadSpecificationTestState(t, root).Plan

	coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if coordinated.Candidate == nil || coordinated.ScoutDispatch != nil || coordinated.DecisionCheckpoint != nil {
		t.Fatalf("stop coordination = %+v, want one non-active candidate", coordinated)
	}
	candidate := coordinated.Candidate
	if candidate.Status != colony.PlanCandidatePendingReview || candidate.StopDecision.Reason != colony.PlanningStopTargetMet {
		t.Fatalf("candidate status/stop = %s/%s, want pending_review/target_met", candidate.Status, candidate.StopDecision.Reason)
	}
	if candidate.Recommendation.Disposition != colony.PlanRecommendationAccept || candidate.Recommendation.Producer != colony.PlanRecommendationProducerQueen ||
		candidate.Recommendation.ProducerID == "" || candidate.Recommendation.Rationale == "" || len(candidate.Recommendation.EvidenceIDs) == 0 {
		t.Fatalf("candidate recommendation is not authorized, typed, and evidence-grounded: %+v", candidate.Recommendation)
	}
	if candidate.EvidenceThatWouldChange == "" || len(candidate.ResidualGaps) != len(colony.PlanningDimensions()) {
		t.Fatalf("candidate omitted residual causal evidence: %+v", candidate)
	}
	if err := candidate.Validate(); err != nil {
		t.Fatalf("persisted candidate is invalid: %v", err)
	}
	after := mustReadSpecificationTestState(t, root).Plan
	beforeBytes, _ := json.Marshal(before)
	afterBytes, _ := json.Marshal(after)
	if !bytes.Equal(beforeBytes, afterBytes) {
		t.Fatalf("candidate stop changed active plan bytes:\nbefore=%s\nafter=%s", beforeBytes, afterBytes)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(planningRouteCandidateRepositoryPath(manifest.RunID))))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte(candidate.ID)) {
		t.Fatalf("candidate artifact does not contain %q: %s", candidate.ID, content)
	}
}

func TestPlanningRouteStageStopBelowTargetRecommendsRevise(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)
	planningRouteStageSetPolicy(t, root, manifest.RunID, 90, 1)
	coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if coordinated.Candidate == nil || coordinated.Candidate.StopDecision.Reason != colony.PlanningStopPassCap || coordinated.Candidate.Recommendation.Disposition != colony.PlanRecommendationRevise {
		t.Fatalf("below-target cap stop = %+v, want a revise-only candidate", coordinated.Candidate)
	}
}

func TestPlanningRouteStageMaterialDecisionOccursAfterCard(t *testing.T) {
	root, manifest, result := planningRouteStageMaterialFixture(t)
	coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if coordinated.DecisionCheckpoint == nil || coordinated.Candidate != nil || coordinated.ScoutDispatch != nil {
		t.Fatalf("material coordination = %+v, want one owner checkpoint and no progression", coordinated)
	}
	checkpoint := coordinated.DecisionCheckpoint
	if checkpoint.CompletedCardHash != coordinated.Route.Card.ContentHash || checkpoint.Batch.BoundaryCardHash != coordinated.Route.Card.ContentHash {
		t.Fatalf("material checkpoint is not bound after the complete card: checkpoint=%+v card=%+v", checkpoint, coordinated.Route.Card)
	}
	timeline, err := loadPlanningTimeline(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline.Cards) != 1 || timeline.Cards[0].ContentHash != coordinated.Route.Card.ContentHash {
		t.Fatalf("material boundary appeared before the card: %+v", timeline.Cards)
	}
}

func TestPlanningRouteStageMaterialDirectAnswerResumesScout(t *testing.T) {
	root, manifest, result := planningRouteStageMaterialFixture(t)
	coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := coordinated.DecisionCheckpoint
	card := checkpoint.Cards[0]
	resume, err := buildPlanningScoutDecisionResumeToken(*checkpoint, []planningScoutDecisionAnswer{{
		DecisionID: card.DecisionID, ChoiceID: "continue-research", Answer: "Continue research before accepting this risk.",
	}})
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := resumePlanningRouteDecision(root, manifest.RunID, resume, time.Date(2026, time.September, 7, 20, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if resumed.ScoutDispatch == nil || resumed.SuccessorSpecification != nil || resumed.ResumeToken == nil {
		t.Fatalf("equivalent material answer = %+v, want direct next-Scout resume", resumed)
	}
}

func TestPlanningRouteStageMaterialContractAnswerCreatesSuccessorDraft(t *testing.T) {
	root, manifest, result := planningRouteStageMaterialFixture(t)
	coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := coordinated.DecisionCheckpoint
	card := checkpoint.Cards[0]
	resume, err := buildPlanningScoutDecisionResumeToken(*checkpoint, []planningScoutDecisionAnswer{{
		DecisionID: card.DecisionID, ChoiceID: "proceed-with-risk", Answer: "Proceed despite the documented residual risk.",
	}})
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := resumePlanningRouteDecision(root, manifest.RunID, resume, time.Date(2026, time.September, 7, 20, 5, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if resumed.SuccessorSpecification == nil || resumed.SuccessorSpecification.Revision.Status != colony.SpecStatusDraft || resumed.ScoutDispatch != nil || resumed.Candidate != nil {
		t.Fatalf("contract-changing material answer = %+v, want successor DRAFT and no progression", resumed)
	}
	state, err := loadPlanningStageState(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Stage != planningStageSpecApprovalRequired || state.PendingSpecification == nil || state.PendingSpecification.RevisionID != resumed.SuccessorSpecification.Revision.ID {
		t.Fatalf("successor boundary = %+v, want exact spec approval requirement", state)
	}
}

func TestPlanningRouteStageMaterialLaterPassPausesAfterCompleteCard(t *testing.T) {
	root, firstManifest, firstResult := planningRouteStageTestFixture(t)
	first, err := coordinatePlanningRouteStage(root, firstManifest, planningRouteStageTestBytes(t, firstResult))
	if err != nil {
		t.Fatal(err)
	}
	if first.ScoutDispatch == nil {
		t.Fatal("first Route pass did not authorize the next Scout")
	}
	scoutManifest := first.ScoutDispatch.Manifest
	fresh := planningRouteStageEvidence(t, scoutManifest.Specification, scoutManifest.BasePlanRevisionID, "route-late-material", "A later Scout pass found a material behavior decision.", time.Date(2026, time.September, 7, 20, 10, 0, 0, time.UTC))
	scoutGap := planningRouteStageGap("late-scout-gap", colony.PlanningDimensionRisks, fresh.Reference.ID, colony.PlanningGapNonMaterial, 20)
	material := planningScoutStageMaterialCandidate(fresh.Reference, "late-route-material-decision")
	scoutResult := planningScoutStageResult{
		ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
		RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
		Specification: scoutManifest.Specification, BasePlanRevisionID: scoutManifest.BasePlanRevisionID, BasePlanRevisionHash: scoutManifest.BasePlanRevisionHash,
		InputFrontierHash: scoutManifest.InputFrontierHash,
		Findings:          []planningScoutStageFinding{{StableID: "late-material-finding", Summary: "The full pass must complete before owner review.", EvidenceIDs: []string{fresh.Reference.ID}}},
		NewEvidence:       []planningEvidenceRecord{fresh}, UnresolvedGaps: []colony.PlanningGap{scoutGap}, DecisionCandidates: []planningDecisionCandidate{material},
	}
	scoutCompleted, err := coordinatePlanningScoutStage(root, scoutManifest, planningScoutStageTestBytes(t, scoutResult))
	if err != nil {
		t.Fatal(err)
	}
	if scoutCompleted.RouteDispatch == nil || len(scoutCompleted.RouteDispatch.MaterialDecisionCandidates) != 1 {
		t.Fatalf("late Scout did not carry its decision through Route authorization: %+v", scoutCompleted)
	}
	routeManifest := scoutCompleted.RouteDispatch.Manifest
	secondResult := planningRouteStageResult{
		ResultType: planningStageResultRouteSetter, ManifestID: routeManifest.ID, ManifestHash: routeManifest.ContentHash,
		RunID: routeManifest.RunID, Pass: routeManifest.Pass, Caste: planningStageCasteRouteSetter,
		Specification: routeManifest.Specification, BasePlanRevisionID: routeManifest.BasePlanRevisionID, BasePlanRevisionHash: routeManifest.BasePlanRevisionHash,
		PriorCardHash: routeManifest.PriorCardHash, InputFrontierHash: routeManifest.InputFrontierHash,
		ScoutReceipt: *routeManifest.ScoutReceipt, CandidateSnapshotHash: routeManifest.CandidateSnapshotHash,
		Proposal: firstResult.Proposal, ProposalEvidenceIDs: []string{fresh.Reference.ID}, MaterialDecisionCandidates: []planningDecisionCandidate{material},
	}
	for index, dimension := range colony.PlanningDimensions() {
		before := first.Route.Card.DimensionAssessments[index].After
		secondResult.DimensionAssessments = append(secondResult.DimensionAssessments, colony.PlanningDimensionAssessment{
			SchemaVersion: colony.PlanningSchemaVersion, ID: "late-route-assessment-" + string(dimension), ContentHash: planningStageTestHash(string(rune('a' + index))),
			Dimension: dimension, Before: before, After: before, FreshEvidenceIDs: []string{fresh.Reference.ID},
			RemainingGap: planningRouteStageGap("late-route-gap-"+string(dimension), dimension, fresh.Reference.ID, colony.PlanningGapNonMaterial, 15+index),
			Rationale:    "The later Scout evidence leaves this dimension unchanged.", ProducerReceiptID: routeManifest.ID,
		})
	}
	second, err := coordinatePlanningRouteStage(root, routeManifest, planningRouteStageTestBytes(t, secondResult))
	if err != nil {
		t.Fatal(err)
	}
	if second.Route.Card.Iteration != 2 || second.Route.Card.Decision.Reason != colony.PlanningStopOwnerDecision || second.DecisionCheckpoint == nil || second.ScoutDispatch != nil || second.Candidate != nil {
		t.Fatalf("late material Route completion = %+v, want pass-two card then owner decision", second)
	}
	if second.DecisionCheckpoint.CompletedCardHash != second.Route.Card.ContentHash || len(second.DecisionCheckpoint.Batch.Decisions) != 1 || second.DecisionCheckpoint.Batch.Decisions[0].StableID != material.StableID {
		t.Fatalf("late material checkpoint lost its exact card or Scout candidate: %+v", second.DecisionCheckpoint)
	}
}

func TestPlanningRouteStageCandidateReplayReturnsExactArtifact(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)
	planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
	raw := planningRouteStageTestBytes(t, result)
	first, err := coordinatePlanningRouteStage(root, manifest, raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := coordinatePlanningRouteStage(root, manifest, raw)
	if err != nil {
		t.Fatal(err)
	}
	if first.Candidate == nil || second.Candidate == nil || first.Candidate.ID != second.Candidate.ID || first.Candidate.ContentHash != second.Candidate.ContentHash {
		t.Fatalf("candidate replay diverged: first=%+v second=%+v", first.Candidate, second.Candidate)
	}
}

func planningRouteStageMaterialFixture(t *testing.T) (string, planningStageManifest, planningRouteStageResult) {
	t.Helper()
	root, manifest, result := planningRouteStageTestFixture(t)
	planningRouteStageSetPolicy(t, root, manifest.RunID, 90, 1)
	result.DimensionAssessments[0].RemainingGap.Materiality = colony.PlanningGapMaterial
	result.DimensionAssessments[0].RemainingGap.Severity = 100
	return root, manifest, result
}

func planningRouteStageSetPolicy(t *testing.T, root, runID string, target, passCap int) {
	t.Helper()
	headerPath := filepath.Join(root, ".aether", "data", "planning", runID, "run-header.json")
	content, err := os.ReadFile(headerPath)
	if err != nil {
		t.Fatal(err)
	}
	var header planningRunHeader
	if err := json.Unmarshal(content, &header); err != nil {
		t.Fatal(err)
	}
	header.TargetConfidence = target
	header.PassCap = passCap
	header.ID = ""
	header.ContentHash = ""
	hash, err := jsonSHA256(header)
	if err != nil {
		t.Fatal(err)
	}
	header.ContentHash = hash
	header.ID = "planning-run-header-" + hash[:16]
	planningStageReceiptTestWriteJSON(t, headerPath, header)
}

// planningRouteStageFixtureNow is the instant every candidate built on this
// fixture observes as "now": after the last September-8 evidence literal any
// dependent fixture adds (classicPhase200TwoPassCandidate's second pass at
// 01:30, planning_real_repo_200's at 02:20), before the September-9 literals
// tests pass explicitly, and inside the seven-day acceptance window production
// stamps from the seed evidence below. The
// fixture's dates are fixed literals, so its clock must be fixed too -- read
// against the wall clock, every candidate it produces expired at
// 2026-09-14T18:05:00Z and eleven tests started failing by calendar.
var planningRouteStageFixtureNow = time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)

func planningRouteStageTestFixture(t *testing.T) (string, planningStageManifest, planningRouteStageResult) {
	t.Helper()
	previousClock := planCandidateNow
	planCandidateNow = func() time.Time { return planningRouteStageFixtureNow }
	t.Cleanup(func() { planCandidateNow = previousClock })
	root := newSpecificationTestRepository(t, colony.ColonyState{})
	draftRequest := specificationTestDraftRequest(t, colony.SpecScopeWholeGoal)
	draftRequest.Scope.GoalID = "goal-200"
	draftRequest.Scope.SessionID = "session-200"
	draft, err := createSpecificationDraft(root, draftRequest, specificationMutationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := approveSpecification(root, specificationApprovalRequest{
		RevisionID:          draft.Revision.ID,
		RevisionContentHash: draft.Revision.ContentHash,
		ApprovalToken:       specificationApprovalToken(draft.Specification.ID, draft.Revision.ID, draft.Revision.ContentHash),
		ApprovedBy:          "owner",
		ApprovedAt:          draftRequest.CreatedAt.Add(time.Minute),
	}, specificationMutationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	approvalHash, err := jsonSHA256(*approved.Revision.Approval)
	if err != nil {
		t.Fatal(err)
	}
	baseState := mustReadSpecificationTestState(t, root)
	baseHash, err := planStateHash(baseState.Plan)
	if err != nil {
		t.Fatal(err)
	}
	baseID, baseHash := planningBaseRevisionIdentity(baseState.Plan, baseHash)
	binding := planningStageSpecificationBinding{
		RevisionID:          approved.Revision.ID,
		ContentHash:         approved.Revision.ContentHash,
		Status:              colony.SpecStatusApproved,
		ApprovalReceiptID:   approved.Revision.Approval.ID,
		ApprovalReceiptHash: approvalHash,
	}

	manifest, result := planningRouteStageTestRun(t, root, approved.Revision, binding, baseID, baseHash, "planning-route-stage-run")
	return root, manifest, result
}

// planningRouteStageTestRun drives one planning run in an existing repository
// from a fresh Scout to a Route-Setter manifest and a complete Route result.
// Runs other than the default use their own evidence origins, so two runs in
// one repository -- the state a planning restart leaves -- never share evidence.
func planningRouteStageTestRun(t *testing.T, root string, spec colony.SpecRevision, binding planningStageSpecificationBinding, baseID, baseHash, runID string) (planningStageManifest, planningRouteStageResult) {
	t.Helper()
	origin := func(name string) string {
		if runID == "planning-route-stage-run" {
			return name
		}
		return name + "-" + runID
	}
	seedRecord := planningRouteStageEvidence(t, binding, baseID, origin("route-seed"), "Seed evidence authorizes the Scout frontier.", time.Date(2026, time.September, 7, 18, 0, 0, 0, time.UTC))
	state := planningStageState{
		Stage: planningStageScoutReady, RunID: runID, Pass: 1,
		Preset: planningStagePresetBalanced, Specification: binding,
		BasePlanRevisionID: baseID, BasePlanRevisionHash: baseHash,
		PriorCardHash: planningStageTestHash("d"), InputFrontierHash: planningStageTestHash("e"),
	}
	authorization := planningStageAuthorization{
		ID: "authorization-route-stage-scout", ExpectedCaste: planningStageCasteScout,
		InputFrontierHash: state.InputFrontierHash,
		EvidenceFrontier:  []planningStageEvidenceBinding{{ID: seedRecord.Reference.ID, ContentHash: seedRecord.Reference.ContentHash}},
		WeakestGap:        planningStageTestGap("route-stage-initial-gap"),
	}
	running, scoutManifest, err := reducePlanningStage(state, planningStageTransition{To: planningStageScoutRunning, Authorization: &authorization})
	if err != nil {
		t.Fatal(err)
	}
	if err := recordPlanningStageDispatch(root, running, *scoutManifest, planningStageWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	header := planningRouteStageRunHeader(t, *scoutManifest, seedRecord)
	planningStageReceiptTestWriteJSON(t, filepath.Join(root, ".aether", "data", "planning", scoutManifest.RunID, "run-header.json"), header)

	freshRecord := planningRouteStageEvidence(t, binding, baseID, origin("route-fresh"), "Fresh evidence supports the complete Route proposal.", time.Date(2026, time.September, 7, 18, 5, 0, 0, time.UTC))
	scoutGap := planningRouteStageGap("scout-route-gap", colony.PlanningDimensionKnowledge, freshRecord.Reference.ID, colony.PlanningGapNonMaterial, 25)
	scoutResult := planningScoutStageResult{
		ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
		RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
		Specification: binding, BasePlanRevisionID: baseID, BasePlanRevisionHash: baseHash,
		InputFrontierHash: scoutManifest.InputFrontierHash,
		Findings: []planningScoutStageFinding{{
			StableID: "route-stage-finding", Summary: "The exact Route proposal can now be validated.", EvidenceIDs: []string{freshRecord.Reference.ID},
		}},
		NewEvidence: []planningEvidenceRecord{freshRecord}, UnresolvedGaps: []colony.PlanningGap{scoutGap},
	}
	coordinated, err := coordinatePlanningScoutStage(root, *scoutManifest, planningScoutStageTestBytes(t, scoutResult))
	if err != nil {
		t.Fatal(err)
	}
	if coordinated.RouteDispatch == nil {
		t.Fatal("Scout did not authorize Route-Setter")
	}
	routeManifest := coordinated.RouteDispatch.Manifest

	proposal := planningRouteStageProposal(spec)
	assessments := make([]colony.PlanningDimensionAssessment, 0, len(colony.PlanningDimensions()))
	for index, dimension := range colony.PlanningDimensions() {
		gap := planningRouteStageGap("route-gap-"+string(dimension), dimension, freshRecord.Reference.ID, colony.PlanningGapNonMaterial, 10+index)
		assessments = append(assessments, colony.PlanningDimensionAssessment{
			SchemaVersion: colony.PlanningSchemaVersion,
			ID:            "route-assessment-" + string(dimension), ContentHash: planningStageTestHash(string(rune('1' + index))),
			Dimension: dimension, Before: 0, After: 70 + index,
			FreshEvidenceIDs: []string{freshRecord.Reference.ID}, RemainingGap: gap,
			Rationale:         "Fresh Scout evidence supports the proposed " + string(dimension) + " score.",
			ProducerReceiptID: routeManifest.ID,
		})
	}
	return routeManifest, planningRouteStageResult{
		ResultType: planningStageResultRouteSetter, ManifestID: routeManifest.ID, ManifestHash: routeManifest.ContentHash,
		RunID: routeManifest.RunID, Pass: routeManifest.Pass, Caste: planningStageCasteRouteSetter,
		Specification: binding, BasePlanRevisionID: baseID, BasePlanRevisionHash: baseHash,
		PriorCardHash: routeManifest.PriorCardHash, InputFrontierHash: routeManifest.InputFrontierHash,
		ScoutReceipt: *routeManifest.ScoutReceipt, CandidateSnapshotHash: routeManifest.CandidateSnapshotHash,
		Proposal: proposal, ProposalEvidenceIDs: []string{freshRecord.Reference.ID}, DimensionAssessments: assessments,
	}
}

func planningRouteStageRunHeader(t *testing.T, manifest planningStageManifest, seed planningEvidenceRecord) planningRunHeader {
	t.Helper()
	header := planningRunHeader{
		SchemaVersion: planningRunHeaderSchemaVersion, RunID: manifest.RunID,
		Goal: "Restore visible staged planning", GoalID: "goal-200", SessionID: "session-200",
		Specification: manifest.Specification, BasePlanRevisionID: manifest.BasePlanRevisionID, BasePlanRevisionHash: manifest.BasePlanRevisionHash,
		Preset: manifest.Preset, TargetConfidence: 90, PassCap: 6,
		EvidenceCatalogue: []planningEvidenceRecord{seed}, EvidenceFrontier: append([]planningStageEvidenceBinding(nil), manifest.EvidenceFrontier...),
		InputFrontierHash: manifest.InputFrontierHash,
		ResearchPolicy:    phaseResearchAutomaticPolicy{SchemaVersion: phaseResearchAutomaticPolicySchemaVersion, Preset: manifest.Preset, OwnerDecisionBoundary: "after_scout", EvidenceContract: automaticPhaseResearchEvidenceContract()},
		WeakestGap:        *manifest.WeakestGap, StageManifestID: manifest.ID, StageManifestHash: manifest.ContentHash,
		CreatedAt: time.Date(2026, time.September, 7, 17, 0, 0, 0, time.UTC),
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

func planningRouteStageEvidence(t *testing.T, binding planningStageSpecificationBinding, baseID, origin, content string, observedAt time.Time) planningEvidenceRecord {
	t.Helper()
	record, err := normalizePlanningEvidence(planningEvidenceSource{
		Kind: colony.PlanningEvidenceResearch, Origin: origin, Content: []byte(content),
		Scope:          planningEvidenceScope{GoalID: "goal-200", SessionID: "session-200", SpecificationRevisionID: binding.RevisionID, PlanRevisionID: baseID},
		SourceRevision: origin + "-revision", ObservedAt: observedAt,
		ApplicableDimensions: colony.PlanningDimensions(), State: planningEvidenceSourceCurrent,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func planningRouteStageProposal(spec colony.SpecRevision) planningRoutePlanProposal {
	taskID := "1.1"
	requirement := spec.Requirements[0].ID
	acceptance := spec.AcceptanceChecks[0].ID
	negative := spec.NegativeExpectations[0].ID
	recovery := spec.RecoveryExpectations[0].ID
	publicPath := spec.AffectedPublicPaths[0].ID
	criterion := "The exact Route proposal is validated"
	phase := colony.Phase{
		ID: 1, SemanticID: "phase-route-finalization", Name: "Route finalization", Description: "Finalize the exact Route proposal", Mode: colony.PhaseModeProduction,
		RequirementProofLinks: []string{requirement}, AcceptanceProofLinks: []string{acceptance}, NegativeProofLinks: []string{negative}, RecoveryProofLinks: []string{recovery}, PublicPathProofLinks: []string{publicPath},
		SuccessCriteria: []string{criterion}, EvidenceRequirements: []colony.CriterionEvidenceRequirement{{Criterion: criterion, Checks: []string{"tests"}}},
		Tasks: []colony.Task{{
			ID: &taskID, SemanticID: "task-route-validate", Goal: "Validate the Route proposal",
			RequirementProofLinks: []string{requirement}, AcceptanceProofLinks: []string{acceptance}, NegativeProofLinks: []string{negative}, RecoveryProofLinks: []string{recovery}, PublicPathProofLinks: []string{publicPath},
			SuccessCriteria: []string{criterion}, EvidenceRequirements: []colony.CriterionEvidenceRequirement{{Criterion: criterion, Checks: []string{"tests"}}},
		}},
	}
	return planningRoutePlanProposal{
		SemanticID: "plan-route-finalization", Phases: []colony.Phase{phase},
		TaskDeclarations:      []planningRouteTaskDeclaration{{TaskSemanticID: "task-route-validate", Files: []string{"cmd/codex_plan_finalize.go", "cmd/planning_route_stage_200_test.go"}, UserFacing: true}},
		UserFacingSemanticIDs: []string{"phase-route-finalization", "task-route-validate"},
	}
}

func planningRouteStageGap(id string, dimension colony.PlanningDimension, evidenceID string, materiality colony.PlanningGapMateriality, severity int) colony.PlanningGap {
	return colony.PlanningGap{
		SchemaVersion: colony.PlanningSchemaVersion, ID: id, ContentHash: planningStageTestHash("6"),
		Dimension: dimension, Materiality: materiality, Severity: severity,
		Description: "Remaining " + string(dimension) + " gap", EvidenceIDs: []string{evidenceID},
		EvidenceThatWouldChange: "Fresh evidence resolving the remaining " + string(dimension) + " gap.",
	}
}

func planningRouteStageTestBytes(t *testing.T, result planningRouteStageResult) []byte {
	t.Helper()
	content, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func planningRouteStageCloneResult(t *testing.T, result planningRouteStageResult) planningRouteStageResult {
	t.Helper()
	content := planningRouteStageTestBytes(t, result)
	var cloned planningRouteStageResult
	if err := json.Unmarshal(content, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}

func planningRouteStageTestDeltaSections(delta colony.PlanningSemanticDelta) [][]colony.PlanningSemanticChange {
	return [][]colony.PlanningSemanticChange{
		delta.Phases, delta.Tasks, delta.Dependencies, delta.RequirementLinks,
		delta.AcceptanceChecks, delta.NegativeExpectations, delta.RecoveryExpectations, delta.PublicPaths,
	}
}
