package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningVisualsSpecificationSemanticOrderStableIDs(t *testing.T) {
	result := planningVisualSpecificationFixture()
	output := renderPlanningSpecificationVisual(result, planningVisualOptions{Width: 96})

	wantOrder := []string{
		"📜 Specification", "What this goal delivers", "OUT-01", "Included", "BEH-01",
		"Explicitly excluded", "EXC-01", "Binding decisions", "DEC-01", "Requirements", "REQ-01",
		"Owner-checkable acceptance", "ACC-01", "Negative expectations", "NEG-01",
		"Recovery expectations", "REC-01", "Affected public paths", "PATH-01",
	}
	assertPlanningVisualOrder(t, output, wantOrder)
	if strings.Contains(output, "\nSpec\n") || strings.Contains(output, "│ Spec ") {
		t.Fatalf("full specification label was abbreviated:\n%s", output)
	}
}

func TestPlanningVisualsIterationSemanticContract(t *testing.T) {
	card := planningVisualIterationFixture(colony.PlanningStopContinue)
	output := renderPlanningIterationVisual(card, planningVisualOptions{Width: 96})

	assertPlanningVisualOrder(t, output, []string{
		"Scout", "Route-Setter", "Fresh evidence", "EVIDENCE-01", "Planning readiness",
		"Knowledge", "Requirements", "Risks", "Dependencies", "Effort", "Overall",
		"Weakest gap", "Evidence that would change it", "Plan delta", "Added", "Changed",
		"Removed", "Acceptance", "Negative / recovery / public paths", "Authority impact",
		"Decision", "CONTINUE", "Next research", "Details",
	})
	if strings.Contains(output, "no evidence") {
		t.Fatalf("fresh evidence was hidden:\n%s", output)
	}
}

func TestPlanningVisualsDecisionSemanticAuthority(t *testing.T) {
	card := planningDecisionCard{
		ID: "decision-card-01", DecisionID: "DECISION-01", Decision: "Choose the storage boundary",
		WhyNow: "The choice changes recovery behavior", QueenRecommendation: "Choose local durable state",
		Evidence:            []colony.PlanningEvidenceRef{{ID: "EVIDENCE-02"}},
		Choices:             []planningDecisionChoice{{ID: "CHOICE-01", Label: "Local durable state", Consequence: "No network dependency"}},
		AffectedSemanticIDs: []string{"REQ-01", "REC-01"}, PriorAnswer: "Remote state",
		Revalidation: "The recovery requirement changed", PlanningResumes: "Scout pass 2",
	}
	output := renderPlanningDecisionVisual(card, planningVisualOptions{Width: 96})

	assertPlanningVisualOrder(t, output, []string{
		"Decision", "Why now", "Evidence", "Queen recommends", "Choices", "Consequence",
		"Affected scope", "Prior answer", "Revalidation", "Planning resumes",
	})
}

func TestPlanningVisualsCandidateAuthority(t *testing.T) {
	review := planningVisualCandidateFixture()
	output := renderPlanningCandidateVisual(review, planningVisualOptions{Width: 96})

	assertPlanningVisualOrder(t, output, []string{
		"Plan Candidate", "CANDIDATE — NOT ACTIVE", "Approved specification", "Base plan",
		"Preset", "Stopped because", "Planning readiness", "Remaining gaps", "Evidence that would change it",
		"Planning timeline", "Proposed plan", "Queen recommendation", "Producer", "Candidate remains inactive",
		"Accept this candidate?", review.AcceptanceCommand,
	})
	if strings.Contains(output, "aether build") || strings.Contains(output, "aether run") {
		t.Fatalf("pending candidate exposed execution before owner acceptance:\n%s", output)
	}
}

func TestPlanningVisualsJSONStopLabelsAndEvidence(t *testing.T) {
	for machine, public := range map[colony.PlanningStopReason]string{
		colony.PlanningStopTargetMet:          "target sufficiency",
		colony.PlanningStopDiminishingReturns: "diminishing returns",
		colony.PlanningStopStalledGap:         "stall detected",
		colony.PlanningStopPassCap:            "iteration cap",
	} {
		review := planningVisualCandidateFixture()
		review.StopDecision.Reason = machine
		review.Candidate.StopDecision.Reason = machine
		projection := projectPlanningCandidate(review)
		encoded, err := json.Marshal(projection)
		if err != nil {
			t.Fatalf("marshal %s projection: %v", machine, err)
		}
		body := string(encoded)
		for _, want := range []string{
			`"stop_reason":"` + string(machine) + `"`,
			`"stop_reason_public_label":"` + public + `"`,
			`"evidence_that_would_change":"A verified dependency contract"`,
			`"recommendation_disposition":"accept"`,
			`"recommendation_producer":"queen"`,
			`"recommendation_evidence_ids":["EVIDENCE-01"]`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%s JSON missing %s: %s", machine, want, body)
			}
		}
	}
}

func planningVisualSpecificationFixture() specCommandResult {
	return specCommandResult{
		Command: "spec", Operation: specCommandOperationInspect, SpecificationID: "SPEC-01", RevisionNumber: 1,
		BeforeRevisionID: "SPEC-REV-00", AfterRevisionID: "SPEC-REV-01", Status: colony.SpecStatusApproved,
		Scope:                colony.SpecScope{Kind: colony.SpecScopeWholeGoal},
		Outcomes:             []colony.SpecOutcome{{ID: "OUT-01", Description: "A trusted plan"}},
		IncludedBehaviors:    []colony.SpecIncludedBehavior{{ID: "BEH-01", Description: "Iterative research"}},
		Exclusions:           []colony.SpecExclusion{{ID: "EXC-01", Description: "Silent activation"}},
		BindingDecisions:     []colony.SpecBindingDecision{{ID: "DEC-01", Description: "Owner accepts plans"}},
		Requirements:         []colony.SpecRequirement{{ID: "REQ-01", Description: "Keep stable identity"}},
		AcceptanceChecks:     []colony.SpecAcceptanceCheck{{ID: "ACC-01", Description: "Shows evidence", Verification: "Inspect output"}},
		NegativeExpectations: []colony.SpecNegativeExpectation{{ID: "NEG-01", Description: "Never imply authority"}},
		RecoveryExpectations: []colony.SpecRecoveryExpectation{{ID: "REC-01", Description: "Show one recovery action"}},
		AffectedPublicPaths:  []colony.SpecPublicPath{{ID: "PATH-01", Path: "aether plan", Description: "Planning entrypoint"}},
		NextAction:           "aether plan",
	}
}

func planningVisualIterationFixture(reason colony.PlanningStopReason) colony.PlanningIterationCard {
	dimensions := colony.PlanningDimensions()
	assessments := make([]colony.PlanningDimensionAssessment, 0, len(dimensions))
	for index, dimension := range dimensions {
		assessments = append(assessments, colony.PlanningDimensionAssessment{
			ID: "ASSESS-0" + string(rune('1'+index)), Dimension: dimension, Before: 50 + index,
			After: 70 + index, FreshEvidenceIDs: []string{"EVIDENCE-01"},
		})
	}
	return colony.PlanningIterationCard{
		ID: "ITERATION-01", Iteration: 1, ScoutReceiptID: "SCOUT-RECEIPT-01",
		RouteSetterReceiptID: "ROUTE-RECEIPT-01", EvidenceIDs: []string{"EVIDENCE-01"},
		DimensionAssessments: assessments,
		WeakestGap: colony.PlanningGap{ID: "GAP-01", Dimension: colony.PlanningDimensionRisks,
			Materiality: colony.PlanningGapNonMaterial, Description: "Dependency ownership is unverified",
			EvidenceThatWouldChange: "A verified dependency contract"},
		SemanticDelta: colony.PlanningSemanticDelta{
			Phases:               []colony.PlanningSemanticChange{{SemanticID: "PHASE-02", Kind: colony.PlanningSemanticChangeAdded}},
			Tasks:                []colony.PlanningSemanticChange{{SemanticID: "TASK-03", Kind: colony.PlanningSemanticChangeModified}},
			Dependencies:         []colony.PlanningSemanticChange{{SemanticID: "DEP-01", Kind: colony.PlanningSemanticChangeRemoved}},
			AcceptanceChecks:     []colony.PlanningSemanticChange{{SemanticID: "ACC-01", Kind: colony.PlanningSemanticChangePreserved}},
			NegativeExpectations: []colony.PlanningSemanticChange{{SemanticID: "NEG-01", Kind: colony.PlanningSemanticChangePreserved}},
			RecoveryExpectations: []colony.PlanningSemanticChange{{SemanticID: "REC-01", Kind: colony.PlanningSemanticChangePreserved}},
			PublicPaths:          []colony.PlanningSemanticChange{{SemanticID: "PATH-01", Kind: colony.PlanningSemanticChangePreserved}},
			AuthorityImpacts:     []colony.PlanningAuthorityImpact{{ID: "AUTH-01", Kind: colony.PlanningAuthorityImpactKind("preserved"), Rationale: "Owner authority is unchanged"}},
		},
		Decision:                colony.PlanningStopDecision{Reason: reason, Rationale: "One more pass can resolve the gap", EvidenceThatWouldChange: "A verified dependency contract"},
		EvidenceThatWouldChange: "A verified dependency contract",
	}
}

func planningVisualCandidateFixture() planCandidateReview {
	card := planningVisualIterationFixture(colony.PlanningStopTargetMet)
	stop := card.Decision
	recommendation := colony.QueenPlanRecommendation{
		Disposition: colony.PlanRecommendationAccept, EvidenceIDs: []string{"EVIDENCE-01"},
		Rationale: "The proposal meets the approved specification", Producer: colony.PlanRecommendationProducerQueen,
		ProducerID: "queen-runtime-01",
	}
	candidate := colony.PlanCandidate{
		ID: "CANDIDATE-01", Status: colony.PlanCandidatePendingReview,
		Proposal:           colony.PlanRevision{ID: "PLAN-REV-02", Phases: []colony.Phase{{ID: 2, Name: "Render planning truth"}}},
		BasePlanRevisionID: "PLAN-REV-01", SpecificationRevisionID: "SPEC-REV-01",
		StopDecision: stop, DimensionAssessments: card.DimensionAssessments, SemanticDelta: card.SemanticDelta,
		ResidualGaps: []colony.PlanningGap{card.WeakestGap}, EvidenceThatWouldChange: "A verified dependency contract",
		Recommendation: recommendation,
	}
	return planCandidateReview{
		Operation: planCandidateOperationReview, Candidate: candidate, TargetConfidence: 80, ActualConfidence: 82,
		StopDecision: stop, ResidualGaps: candidate.ResidualGaps, EvidenceThatWouldChange: candidate.EvidenceThatWouldChange,
		SemanticDelta: candidate.SemanticDelta, Recommendation: recommendation, Iterations: []colony.PlanningIterationCard{card},
		AcceptanceCommand: "aether plan --accept-candidate CANDIDATE-01 --acceptance-token TOKEN",
	}
}

func assertPlanningVisualOrder(t *testing.T, output string, values []string) {
	t.Helper()
	position := -1
	for _, value := range values {
		next := strings.Index(output[position+1:], value)
		if next < 0 {
			t.Fatalf("output missing %q after byte %d:\n%s", value, position, output)
		}
		position += next + 1
	}
}
