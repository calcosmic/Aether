package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningStageTransitionTableRejectsEverySkippedStage(t *testing.T) {
	want := map[planningStage][]planningStage{
		planningStagePresetRequired:         {planningStageScoutReady, planningStageFailed},
		planningStageScoutReady:             {planningStageScoutRunning, planningStageFailed},
		planningStageScoutRunning:           {planningStageOwnerDecision, planningStageRouteReady, planningStageFailed},
		planningStageOwnerDecision:          {planningStageScoutReady, planningStageSpecApprovalRequired, planningStageRouteReady, planningStageFailed},
		planningStageSpecApprovalRequired:   {planningStageReconciliationRequired, planningStageFailed},
		planningStageReconciliationRequired: {planningStageScoutReady, planningStageFailed},
		planningStageRouteReady:             {planningStageRouteRunning, planningStageFailed},
		planningStageRouteRunning:           {planningStageOwnerDecision, planningStageContinueReady, planningStageCandidateReady, planningStageFailed},
		planningStageContinueReady:          {planningStageScoutReady, planningStageFailed},
		planningStageCandidateReady:         {planningStageAccepted, planningStageFailed},
		planningStageAccepted:               {},
		planningStageFailed:                 {},
	}

	states := planningStageValues()
	if len(states) != len(want) {
		t.Fatalf("stage vocabulary has %d values, want %d", len(states), len(want))
	}
	for _, from := range states {
		allowed := map[planningStage]bool{}
		for _, to := range want[from] {
			allowed[to] = true
		}
		for _, to := range states {
			got := planningStageTransitionAllowed(from, to)
			if got != allowed[to] {
				t.Errorf("transition %s -> %s allowed=%v, want %v", from, to, got, allowed[to])
			}
		}
	}

	state := planningStageTestState(planningStageScoutReady)
	before := state
	if _, _, err := reducePlanningStage(state, planningStageTransition{To: planningStageRouteRunning}); err == nil || !strings.Contains(err.Error(), "illegal planning stage transition") {
		t.Fatalf("skipped-stage transition error = %v", err)
	}
	if state.Stage != before.Stage || state.Pass != before.Pass {
		t.Fatalf("failed pure transition mutated its input: before=%+v after=%+v", before, state)
	}
}

func TestPlanningStageManifestAuthorizesExactlyOneScoutAction(t *testing.T) {
	state := planningStageTestState(planningStageScoutReady)
	authorization := planningStageAuthorization{
		ID:                "authorization-scout-1",
		ExpectedCaste:     planningStageCasteScout,
		InputFrontierHash: planningStageTestHash("1"),
		EvidenceFrontier: []planningStageEvidenceBinding{
			{ID: "evidence-survey", ContentHash: planningStageTestHash("2")},
			{ID: "evidence-spec", ContentHash: planningStageTestHash("3")},
		},
		WeakestGap: planningStageTestGap("knowledge-gap"),
	}

	next, manifest, err := reducePlanningStage(state, planningStageTransition{
		To:            planningStageScoutRunning,
		Authorization: &authorization,
	})
	if err != nil {
		t.Fatal(err)
	}
	if next.Stage != planningStageScoutRunning {
		t.Fatalf("next stage = %q, want %q", next.Stage, planningStageScoutRunning)
	}
	if manifest == nil {
		t.Fatal("Scout authorization did not emit a manifest")
	}
	if manifest.ExpectedCaste != planningStageCasteScout || manifest.ExpectedResultType != planningStageResultScout {
		t.Fatalf("worker contract = %q/%q, want Scout/%q", manifest.ExpectedCaste, manifest.ExpectedResultType, planningStageResultScout)
	}
	if manifest.AuthorizationID != authorization.ID || manifest.RunID != state.RunID || manifest.Pass != state.Pass {
		t.Fatalf("manifest lost exact authorization frontier: %+v", manifest)
	}
	if manifest.Specification.RevisionID != state.Specification.RevisionID || manifest.Specification.ContentHash != state.Specification.ContentHash || manifest.Specification.Status != colony.SpecStatusApproved {
		t.Fatalf("manifest specification binding = %+v, want %+v", manifest.Specification, state.Specification)
	}
	if manifest.Preset != state.Preset || manifest.BasePlanRevisionID != state.BasePlanRevisionID || manifest.BasePlanRevisionHash != state.BasePlanRevisionHash || manifest.PriorCardHash != state.PriorCardHash {
		t.Fatalf("manifest omitted required planning authority: %+v", manifest)
	}
	if len(manifest.EvidenceFrontier) != 2 || manifest.WeakestGap == nil || manifest.InputFrontierHash != authorization.InputFrontierHash {
		t.Fatalf("Scout inputs are incomplete: %+v", manifest)
	}
	if manifest.ScoutReceipt != nil || manifest.CandidateSnapshotHash != "" {
		t.Fatalf("Scout manifest leaked future Route-Setter inputs: %+v", manifest)
	}
	if err := validatePlanningStageManifest(*manifest); err != nil {
		t.Fatalf("emitted Scout manifest is invalid: %v", err)
	}
	if !planningSHA256Pattern.MatchString(manifest.ContentHash) || !strings.HasSuffix(manifest.ID, manifest.ContentHash[:16]) {
		t.Fatalf("manifest is not content addressed: %+v", manifest)
	}
	if len(next.UsedAuthorizationIDs) != 1 || next.UsedAuthorizationIDs[0] != authorization.ID {
		t.Fatalf("authorization use not recorded: %v", next.UsedAuthorizationIDs)
	}
}

func TestPlanningStageManifestAuthorizesExactlyOneRouteSetterAction(t *testing.T) {
	state := planningStageTestState(planningStageRouteReady)
	state.ScoutReceipt = &planningStageReceiptRef{
		ID:           "stage-receipt-scout",
		ContentHash:  planningStageTestHash("4"),
		RunID:        state.RunID,
		Pass:         state.Pass,
		Caste:        planningStageCasteScout,
		ManifestHash: planningStageTestHash("5"),
	}
	authorization := planningStageAuthorization{
		ID:                    "authorization-route-1",
		ExpectedCaste:         planningStageCasteRouteSetter,
		InputFrontierHash:     planningStageTestHash("6"),
		ScoutReceipt:          state.ScoutReceipt,
		CandidateSnapshotHash: planningStageTestHash("7"),
	}

	next, manifest, err := reducePlanningStage(state, planningStageTransition{
		To:            planningStageRouteRunning,
		Authorization: &authorization,
	})
	if err != nil {
		t.Fatal(err)
	}
	if next.Stage != planningStageRouteRunning || manifest == nil {
		t.Fatalf("Route-Setter authorization result = %+v, manifest=%+v", next, manifest)
	}
	if manifest.ExpectedCaste != planningStageCasteRouteSetter || manifest.ExpectedResultType != planningStageResultRouteSetter {
		t.Fatalf("worker contract = %q/%q", manifest.ExpectedCaste, manifest.ExpectedResultType)
	}
	if manifest.ScoutReceipt == nil || manifest.ScoutReceipt.ContentHash != state.ScoutReceipt.ContentHash || manifest.ScoutReceipt.ID != state.ScoutReceipt.ID {
		t.Fatalf("Route manifest does not bind exact Scout receipt: %+v", manifest.ScoutReceipt)
	}
	if manifest.CandidateSnapshotHash != authorization.CandidateSnapshotHash {
		t.Fatalf("candidate snapshot = %q, want %q", manifest.CandidateSnapshotHash, authorization.CandidateSnapshotHash)
	}
	if len(manifest.EvidenceFrontier) != 0 || manifest.WeakestGap != nil {
		t.Fatalf("Route manifest leaked Scout-only fields: %+v", manifest)
	}
	if err := validatePlanningStageManifest(*manifest); err != nil {
		t.Fatalf("emitted Route manifest is invalid: %v", err)
	}
}

func TestPlanningStageManifestRejectsWrongCasteReuseAndFutureAuthority(t *testing.T) {
	t.Run("wrong caste", func(t *testing.T) {
		state := planningStageTestState(planningStageScoutReady)
		_, _, err := reducePlanningStage(state, planningStageTransition{
			To: planningStageScoutRunning,
			Authorization: &planningStageAuthorization{
				ID:                "authorization-wrong-caste",
				ExpectedCaste:     planningStageCasteRouteSetter,
				InputFrontierHash: planningStageTestHash("8"),
				EvidenceFrontier:  []planningStageEvidenceBinding{{ID: "evidence", ContentHash: planningStageTestHash("9")}},
				WeakestGap:        planningStageTestGap("risk-gap"),
			},
		})
		if err == nil || !strings.Contains(err.Error(), "expected caste") {
			t.Fatalf("wrong-caste error = %v", err)
		}
	})

	t.Run("reused authorization", func(t *testing.T) {
		state := planningStageTestState(planningStageScoutReady)
		state.UsedAuthorizationIDs = []string{"authorization-reused"}
		_, _, err := reducePlanningStage(state, planningStageTransition{
			To: planningStageScoutRunning,
			Authorization: &planningStageAuthorization{
				ID:                "authorization-reused",
				ExpectedCaste:     planningStageCasteScout,
				InputFrontierHash: planningStageTestHash("a"),
				EvidenceFrontier:  []planningStageEvidenceBinding{{ID: "evidence", ContentHash: planningStageTestHash("b")}},
				WeakestGap:        planningStageTestGap("effort-gap"),
			},
		})
		if err == nil || !strings.Contains(err.Error(), "already used") {
			t.Fatalf("reuse error = %v", err)
		}
	})

	t.Run("future route fields on Scout", func(t *testing.T) {
		manifest := planningStageTestManifest(planningStageCasteScout)
		manifest.ScoutReceipt = &planningStageReceiptRef{
			ID: "future-receipt", ContentHash: planningStageTestHash("c"), RunID: manifest.RunID,
			Pass: manifest.Pass, Caste: planningStageCasteScout, ManifestHash: planningStageTestHash("d"),
		}
		manifest.ContentHash = ""
		manifest.ID = ""
		if err := addressPlanningStageManifest(&manifest); err == nil || !strings.Contains(err.Error(), "future-stage") {
			t.Fatalf("future-stage error = %v", err)
		}
	})

	t.Run("missing Scout receipt on Route", func(t *testing.T) {
		manifest := planningStageTestManifest(planningStageCasteRouteSetter)
		manifest.ScoutReceipt = nil
		manifest.ContentHash = ""
		manifest.ID = ""
		if err := addressPlanningStageManifest(&manifest); err == nil || !strings.Contains(err.Error(), "Scout receipt") {
			t.Fatalf("missing predecessor error = %v", err)
		}
	})

	t.Run("activation field is not in the schema", func(t *testing.T) {
		manifest := planningStageTestManifest(planningStageCasteScout)
		raw, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		raw = []byte(strings.TrimSuffix(string(raw), "}") + `,"activate_plan":true}`)
		if _, err := decodePlanningStageManifest(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("activation field decode error = %v", err)
		}
	})
}

func TestPlanningStageHumanAndCandidateStatesNeverDispatch(t *testing.T) {
	for _, stage := range []planningStage{
		planningStagePresetRequired,
		planningStageOwnerDecision,
		planningStageSpecApprovalRequired,
		planningStageReconciliationRequired,
		planningStageContinueReady,
		planningStageCandidateReady,
		planningStageAccepted,
		planningStageFailed,
	} {
		t.Run(string(stage), func(t *testing.T) {
			if _, err := authorizePlanningStage(planningStageTestState(stage), planningStageAuthorization{
				ID:                "authorization-should-not-exist",
				ExpectedCaste:     planningStageCasteScout,
				InputFrontierHash: planningStageTestHash("e"),
				EvidenceFrontier:  []planningStageEvidenceBinding{{ID: "evidence", ContentHash: planningStageTestHash("f")}},
				WeakestGap:        planningStageTestGap("gap"),
			}); err == nil || !strings.Contains(err.Error(), "does not dispatch") {
				t.Fatalf("stage %s authorization error = %v", stage, err)
			}
		})
	}
}

func TestPlanningStageContractDecisionRequiresExactSuccessorAndReconciliation(t *testing.T) {
	state := planningStageTestState(planningStageOwnerDecision)
	state.DecisionResumeStage = planningStageRouteReady
	draft := planningStageSpecificationBinding{
		RevisionID:            "spec-revision-successor",
		ContentHash:           planningStageTestHash("1"),
		PredecessorRevisionID: state.Specification.RevisionID,
		Status:                colony.SpecStatusDraft,
	}
	resolution := planningDecisionResolution{
		Disposition:         planningDecisionDispositionSuccessorSpecRequired,
		AffectedSemanticIDs: []string{"acceptance-check-1", "requirement-1"},
		RevisionEvidence: []planningDecisionRevisionEvidence{{
			Dimension: "behavior", ApprovedValue: "old", SelectedValue: "new",
		}},
	}

	waitingApproval, manifest, err := reducePlanningStage(state, planningStageTransition{
		To:                     planningStageSpecApprovalRequired,
		DecisionResolution:     &resolution,
		SuccessorSpecification: &draft,
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest != nil || waitingApproval.Stage != planningStageSpecApprovalRequired {
		t.Fatalf("contract decision should wait without dispatch: state=%+v manifest=%+v", waitingApproval, manifest)
	}
	if waitingApproval.PendingSpecification == nil || waitingApproval.PendingSpecification.RevisionID != draft.RevisionID {
		t.Fatalf("pending successor not bound exactly: %+v", waitingApproval.PendingSpecification)
	}

	if _, _, err := reducePlanningStage(waitingApproval, planningStageTransition{To: planningStageRouteReady}); err == nil {
		t.Fatal("contract-changing decision skipped approval and reconciliation")
	}

	wrongApproval := draft
	wrongApproval.ContentHash = planningStageTestHash("2")
	wrongApproval.Status = colony.SpecStatusApproved
	wrongApproval.ApprovalReceiptID = "approval-wrong"
	wrongApproval.ApprovalReceiptHash = planningStageTestHash("3")
	if _, _, err := reducePlanningStage(waitingApproval, planningStageTransition{
		To:                    planningStageReconciliationRequired,
		ApprovedSpecification: &wrongApproval,
	}); err == nil || !strings.Contains(err.Error(), "exact pending successor") {
		t.Fatalf("wrong successor approval error = %v", err)
	}

	approved := draft
	approved.Status = colony.SpecStatusApproved
	approved.ApprovalReceiptID = "approval-successor"
	approved.ApprovalReceiptHash = planningStageTestHash("4")
	reconciling, manifest, err := reducePlanningStage(waitingApproval, planningStageTransition{
		To:                    planningStageReconciliationRequired,
		ApprovedSpecification: &approved,
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest != nil || reconciling.Stage != planningStageReconciliationRequired || reconciling.Specification.RevisionID != approved.RevisionID {
		t.Fatalf("approval transition = %+v manifest=%+v", reconciling, manifest)
	}
	if _, _, err := reducePlanningStage(reconciling, planningStageTransition{To: planningStageCandidateReady}); err == nil {
		t.Fatal("approved successor skipped affected-scope reconciliation")
	}

	reboundFrontier := planningStageTestHash("5")
	restarted, manifest, err := reducePlanningStage(reconciling, planningStageTransition{
		To: planningStageScoutReady,
		Reconciliation: &planningStageReconciliation{
			SpecificationRevisionID:   approved.RevisionID,
			SpecificationRevisionHash: approved.ContentHash,
			AffectedSemanticIDs:       []string{"requirement-1", "acceptance-check-1"},
			InputFrontierHash:         reboundFrontier,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest != nil || restarted.Stage != planningStageScoutReady || restarted.InputFrontierHash != reboundFrontier {
		t.Fatalf("reconciliation did not restart Scout frontier: state=%+v manifest=%+v", restarted, manifest)
	}
	if restarted.PendingSpecification != nil || len(restarted.PendingAffectedSemanticIDs) != 0 {
		t.Fatalf("settled reconciliation left pending authority: %+v", restarted)
	}
}

func planningStageTestState(stage planningStage) planningStageState {
	return planningStageState{
		Stage:  stage,
		RunID:  "planning-run-1",
		Pass:   1,
		Preset: planningStagePresetBalanced,
		Specification: planningStageSpecificationBinding{
			RevisionID:            "spec-revision-1",
			ContentHash:           planningStageTestHash("a"),
			PredecessorRevisionID: "spec-revision-0",
			Status:                colony.SpecStatusApproved,
			ApprovalReceiptID:     "approval-1",
			ApprovalReceiptHash:   planningStageTestHash("b"),
		},
		BasePlanRevisionID:   "plan-revision-base",
		BasePlanRevisionHash: planningStageTestHash("c"),
		PriorCardHash:        planningStageTestHash("d"),
		InputFrontierHash:    planningStageTestHash("e"),
	}
}

func planningStageTestManifest(caste planningStageWorkerCaste) planningStageManifest {
	state := planningStageTestState(planningStageScoutReady)
	authorization := planningStageAuthorization{
		ID:                "authorization-manifest",
		ExpectedCaste:     caste,
		InputFrontierHash: planningStageTestHash("f"),
	}
	if caste == planningStageCasteScout {
		authorization.EvidenceFrontier = []planningStageEvidenceBinding{{ID: "evidence", ContentHash: planningStageTestHash("1")}}
		authorization.WeakestGap = planningStageTestGap("gap")
	} else {
		state.Stage = planningStageRouteReady
		state.ScoutReceipt = &planningStageReceiptRef{
			ID: "scout-receipt", ContentHash: planningStageTestHash("2"), RunID: state.RunID,
			Pass: state.Pass, Caste: planningStageCasteScout, ManifestHash: planningStageTestHash("3"),
		}
		authorization.ScoutReceipt = state.ScoutReceipt
		authorization.CandidateSnapshotHash = planningStageTestHash("4")
	}
	manifest, err := buildPlanningStageManifest(state, authorization)
	if err != nil {
		panic(err)
	}
	return manifest
}

func planningStageTestGap(label string) *colony.PlanningGap {
	return &colony.PlanningGap{
		SchemaVersion:           colony.PlanningSchemaVersion,
		ID:                      label,
		ContentHash:             planningStageTestHash("9"),
		Dimension:               colony.PlanningDimensionKnowledge,
		Materiality:             colony.PlanningGapNonMaterial,
		Severity:                10,
		Description:             label,
		EvidenceIDs:             []string{"evidence"},
		EvidenceThatWouldChange: "fresh evidence for " + label,
	}
}

func planningStageTestHash(character string) string {
	return strings.Repeat(character, 64)
}
