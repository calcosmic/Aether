package cmd

import (
	"encoding/json"
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

func planningRouteStageTestFixture(t *testing.T) (string, planningStageManifest, planningRouteStageResult) {
	t.Helper()
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

	seedRecord := planningRouteStageEvidence(t, binding, baseID, "route-seed", "Seed evidence authorizes the Scout frontier.", time.Date(2026, time.September, 7, 18, 0, 0, 0, time.UTC))
	state := planningStageState{
		Stage: planningStageScoutReady, RunID: "planning-route-stage-run", Pass: 1,
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

	freshRecord := planningRouteStageEvidence(t, binding, baseID, "route-fresh", "Fresh evidence supports the complete Route proposal.", time.Date(2026, time.September, 7, 18, 5, 0, 0, time.UTC))
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

	proposal := planningRouteStageProposal(approved.Revision)
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
	return root, routeManifest, planningRouteStageResult{
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
