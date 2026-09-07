package cmd

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningDecisionClassifyMaterialAndSuppressesNonOwnerChoices(t *testing.T) {
	t.Parallel()

	material := planningDecisionCandidate{
		StableID:            "decision-auth-mode",
		Domain:              planningDecisionDomainAuthority,
		Decision:            "May the generated plan change the authentication boundary?",
		WhyNow:              "The evidence supports two implementations with different authority.",
		Evidence:            []colony.PlanningEvidenceRef{planningDecisionEvidenceFixture("evidence-auth", false)},
		QueenRecommendation: "Keep the existing authentication boundary.",
		Impact: planningDecisionContractImpact{
			Authority: "changes who may approve authentication behavior",
		},
		AffectedSemanticIDs: []string{"requirement:auth-01"},
		ResumeInstruction:   "Answer once to resume planning from this batch.",
	}

	classification, err := classifyPlanningDecision(material)
	if err != nil {
		t.Fatalf("classify material decision: %v", err)
	}
	if !classification.RequiresOwner || classification.Reason != planningDecisionReasonMaterial {
		t.Fatalf("material classification = %+v, want owner material", classification)
	}

	evidenceAnswerable := material
	evidenceAnswerable.StableID = "decision-auth-evidence"
	evidenceAnswerable.Evidence = []colony.PlanningEvidenceRef{planningDecisionEvidenceFixture("evidence-auth-answer", true)}
	evidenceAnswerable.AnswerableEvidenceIDs = []string{"evidence-auth-answer"}
	classification, err = classifyPlanningDecision(evidenceAnswerable)
	if err != nil {
		t.Fatalf("classify evidence-answerable decision: %v", err)
	}
	if classification.RequiresOwner || classification.Reason != planningDecisionReasonEvidenceAnswerable {
		t.Fatalf("evidence-answerable classification = %+v, want autonomous suppression", classification)
	}

	preference := material
	preference.StableID = "decision-colour"
	preference.Decision = "Which colour should the planning heading use?"
	preference.Impact = planningDecisionContractImpact{}
	preference.AffectedSemanticIDs = nil
	classification, err = classifyPlanningDecision(preference)
	if err != nil {
		t.Fatalf("classify generic preference: %v", err)
	}
	if classification.RequiresOwner || classification.Reason != planningDecisionReasonGenericPreference {
		t.Fatalf("generic preference classification = %+v, want autonomous suppression", classification)
	}

	routine := material
	routine.StableID = "decision-research-depth"
	routine.Domain = planningDecisionDomainRoutineResearch
	routine.Decision = "How many Scout sources should be collected?"
	classification, err = classifyPlanningDecision(routine)
	if err != nil {
		t.Fatalf("classify routine research choice: %v", err)
	}
	if classification.RequiresOwner || classification.Reason != planningDecisionReasonRoutineAutonomous {
		t.Fatalf("routine classification = %+v, want autonomous handling", classification)
	}
}

func TestPlanningDecisionBatchStableFirstPass(t *testing.T) {
	t.Parallel()

	boundary := planningDecisionBoundary{
		RunID: "planning-run-1",
		Pass:  1,
		ScoutReceipt: planningDecisionStageReceipt{
			ID: "scout-receipt-1", ContentHash: strings.Repeat("a", 64), RunID: "planning-run-1", Pass: 1,
		},
		RecoveryCommand: "aether plan --resume planning-run-1",
	}
	candidates := []planningDecisionCandidate{
		planningDecisionMaterialFixture("scope-z", planningDecisionDomainScope, "task:scope-z"),
		planningDecisionMaterialFixture("behavior-a", planningDecisionDomainBehavior, "requirement:behavior-a"),
		planningDecisionMaterialFixture("risk-c", planningDecisionDomainRiskTolerance, "acceptance:risk-c"),
	}

	batch, err := buildPlanningDecisionBatch(boundary, candidates)
	if err != nil {
		t.Fatalf("build first-pass batch: %v", err)
	}
	if batch == nil || len(batch.Decisions) != 3 {
		t.Fatalf("batch = %+v, want one batch with all three decisions", batch)
	}
	wantOrder := []string{"behavior-a", "risk-c", "scope-z"}
	gotOrder := make([]string, 0, len(batch.Decisions))
	for _, decision := range batch.Decisions {
		gotOrder = append(gotOrder, decision.StableID)
		if len(decision.Evidence) == 0 || len(decision.AffectedSemanticIDs) == 0 {
			t.Fatalf("decision lost evidence or affected IDs: %+v", decision)
		}
	}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("decision order = %v, want %v", gotOrder, wantOrder)
	}

	reversed := []planningDecisionCandidate{candidates[2], candidates[1], candidates[0]}
	again, err := buildPlanningDecisionBatch(boundary, reversed)
	if err != nil {
		t.Fatalf("build reversed batch: %v", err)
	}
	if !reflect.DeepEqual(again, batch) {
		t.Fatalf("input order changed batch:\n got: %+v\nwant: %+v", again, batch)
	}
}

func TestPlanningDecisionBoundaryRequiresCompletedLaterPass(t *testing.T) {
	t.Parallel()

	candidate := planningDecisionMaterialFixture("acceptance-late", planningDecisionDomainAcceptanceMeaning, "acceptance:late")
	boundary := planningDecisionBoundary{
		RunID: "planning-run-2",
		Pass:  2,
		ScoutReceipt: planningDecisionStageReceipt{
			ID: "scout-receipt-2", ContentHash: strings.Repeat("b", 64), RunID: "planning-run-2", Pass: 2,
		},
		RecoveryCommand: "aether plan --resume planning-run-2",
	}

	if _, err := buildPlanningDecisionBatch(boundary, []planningDecisionCandidate{candidate}); err == nil || !strings.Contains(err.Error(), "route-setter") {
		t.Fatalf("Scout-only later boundary error = %v, want route-setter requirement", err)
	}

	boundary.RouteSetterReceipt = &planningDecisionStageReceipt{
		ID: "route-receipt-2", ContentHash: strings.Repeat("c", 64), RunID: "planning-run-2", Pass: 2,
	}
	if _, err := buildPlanningDecisionBatch(boundary, []planningDecisionCandidate{candidate}); err == nil || !strings.Contains(err.Error(), "iteration card") {
		t.Fatalf("receipt-only later boundary error = %v, want iteration card requirement", err)
	}

	boundary.IterationCard = &planningDecisionIterationCardReceipt{
		ID:                     "iteration-card-2",
		ContentHash:            strings.Repeat("d", 64),
		RunID:                  "planning-run-2",
		Pass:                   2,
		ScoutReceiptID:         boundary.ScoutReceipt.ID,
		ScoutReceiptHash:       boundary.ScoutReceipt.ContentHash,
		RouteSetterReceiptID:   boundary.RouteSetterReceipt.ID,
		RouteSetterReceiptHash: boundary.RouteSetterReceipt.ContentHash,
	}
	batch, err := buildPlanningDecisionBatch(boundary, []planningDecisionCandidate{candidate})
	if err != nil {
		t.Fatalf("build completed later-pass batch: %v", err)
	}
	if batch == nil || batch.BoundaryCardHash != boundary.IterationCard.ContentHash {
		t.Fatalf("later batch = %+v, want persisted card binding", batch)
	}
}

func planningDecisionMaterialFixture(id string, domain planningDecisionDomain, semanticID string) planningDecisionCandidate {
	return planningDecisionCandidate{
		StableID:            id,
		Domain:              domain,
		Decision:            "Choose the material outcome for " + id,
		WhyNow:              "Current evidence leaves materially different outcomes.",
		Evidence:            []colony.PlanningEvidenceRef{planningDecisionEvidenceFixture("evidence-"+id, false)},
		QueenRecommendation: "Preserve the approved contract.",
		Impact:              planningDecisionContractImpact{Behavior: "changes " + id},
		AffectedSemanticIDs: []string{semanticID},
		ResumeInstruction:   "Answer this batch to resume planning.",
	}
}

func planningDecisionEvidenceFixture(id string, answerable bool) colony.PlanningEvidenceRef {
	return colony.PlanningEvidenceRef{
		SchemaVersion:           colony.PlanningEvidenceSchemaVersion,
		ID:                      id,
		ContentHash:             strings.Repeat("e", 64),
		Kind:                    colony.PlanningEvidenceResearch,
		Origin:                  "fixture:" + id,
		GoalID:                  "goal-1",
		SessionID:               "session-1",
		SpecificationRevisionID: "spec-1",
		PlanRevisionID:          "plan-1",
		SourceRevision:          "source-1",
		ObservedAt:              time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
		ExcerptDigest:           strings.Repeat("f", 64),
		ApplicableDimensions:    []colony.PlanningDimension{colony.PlanningDimensionRequirements},
		Fresh:                   true,
		Admissible:              true,
		AdmissibilityReason:     map[bool]string{true: "directly answers the candidate", false: "supports why owner authority is required"}[answerable],
	}
}
