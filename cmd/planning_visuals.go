package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planningVisualOptions keeps terminal presentation choices outside canonical
// planning state. Renderers may fold or expand the same facts, but never infer
// authority from presentation state.
type planningVisualOptions struct {
	Width  int
	Detail bool
}

type planningVisualIdentityProjection struct {
	Caste string `json:"caste"`
	Label string `json:"label"`
}

type planningSpecificationProjection struct {
	Screen               string                           `json:"screen"`
	CommandLabel         string                           `json:"command_label"`
	Identity             planningVisualIdentityProjection `json:"identity"`
	Operation            specCommandOperation             `json:"operation"`
	SpecificationID      string                           `json:"specification_id"`
	BeforeRevisionID     string                           `json:"before_revision_id"`
	AfterRevisionID      string                           `json:"after_revision_id"`
	RevisionNumber       int                              `json:"revision_number"`
	Status               colony.SpecRevisionStatus        `json:"status"`
	Scope                colony.SpecScope                 `json:"scope"`
	Outcomes             []colony.SpecOutcome             `json:"outcomes"`
	IncludedBehaviors    []colony.SpecIncludedBehavior    `json:"included_behaviors"`
	Exclusions           []colony.SpecExclusion           `json:"exclusions"`
	BindingDecisions     []colony.SpecBindingDecision     `json:"binding_decisions"`
	Requirements         []colony.SpecRequirement         `json:"requirements"`
	AcceptanceChecks     []colony.SpecAcceptanceCheck     `json:"acceptance_checks"`
	NegativeExpectations []colony.SpecNegativeExpectation `json:"negative_expectations"`
	RecoveryExpectations []colony.SpecRecoveryExpectation `json:"recovery_expectations"`
	AffectedPublicPaths  []colony.SpecPublicPath          `json:"affected_public_paths"`
	NextAction           string                           `json:"next_action"`
}

type planningScoreProjection struct {
	Dimension string `json:"dimension"`
	Before    int    `json:"before"`
	After     int    `json:"after"`
}

type planningGapProjection struct {
	ID                      string `json:"id"`
	Dimension               string `json:"dimension"`
	Materiality             string `json:"materiality"`
	Description             string `json:"description"`
	EvidenceThatWouldChange string `json:"evidence_that_would_change"`
}

type planningSemanticDeltaProjection struct {
	Added           []string `json:"added"`
	Changed         []string `json:"changed"`
	Removed         []string `json:"removed"`
	Preserved       []string `json:"preserved"`
	Dependencies    []string `json:"dependencies"`
	Acceptance      []string `json:"acceptance"`
	Negative        []string `json:"negative_expectations"`
	Recovery        []string `json:"recovery_expectations"`
	PublicPaths     []string `json:"public_paths"`
	AuthorityImpact []string `json:"authority_impact"`
}

type planningIterationProjection struct {
	Screen                  string                             `json:"screen"`
	Identities              []planningVisualIdentityProjection `json:"identities"`
	Iteration               int                                `json:"iteration"`
	EvidenceIDs             []string                           `json:"evidence_ids"`
	Scores                  []planningScoreProjection          `json:"scores"`
	OverallBefore           int                                `json:"overall_before"`
	OverallAfter            int                                `json:"overall_after"`
	WeakestGap              planningGapProjection              `json:"weakest_gap"`
	SemanticDelta           planningSemanticDeltaProjection    `json:"semantic_delta"`
	Decision                string                             `json:"decision"`
	StopReason              string                             `json:"stop_reason"`
	StopReasonPublicLabel   string                             `json:"stop_reason_public_label"`
	EvidenceThatWouldChange string                             `json:"evidence_that_would_change"`
	DetailCommand           string                             `json:"detail_command"`
}

type planningDecisionProjection struct {
	Screen              string                           `json:"screen"`
	Identity            planningVisualIdentityProjection `json:"identity"`
	DecisionID          string                           `json:"decision_id"`
	Decision            string                           `json:"decision"`
	WhyNow              string                           `json:"why_now"`
	EvidenceIDs         []string                         `json:"evidence_ids"`
	QueenRecommendation string                           `json:"queen_recommendation"`
	Choices             []planningDecisionChoice         `json:"choices"`
	AffectedSemanticIDs []string                         `json:"affected_semantic_ids"`
	PriorAnswer         string                           `json:"prior_answer"`
	Revalidation        string                           `json:"revalidation"`
	PlanningResumes     string                           `json:"planning_resumes"`
}

type planningCandidateProjection struct {
	Screen                    string                           `json:"screen"`
	Identity                  planningVisualIdentityProjection `json:"identity"`
	CandidateID               string                           `json:"candidate_id"`
	CandidateStatus           string                           `json:"candidate_status"`
	CandidateActive           bool                             `json:"candidate_active"`
	SpecificationRevisionID   string                           `json:"specification_revision_id"`
	BasePlanRevisionID        string                           `json:"base_plan_revision_id"`
	ProposalRevisionID        string                           `json:"proposal_revision_id"`
	TargetConfidence          int                              `json:"target_confidence"`
	ActualConfidence          int                              `json:"actual_confidence"`
	Scores                    []planningScoreProjection        `json:"scores"`
	StopReason                string                           `json:"stop_reason"`
	StopReasonPublicLabel     string                           `json:"stop_reason_public_label"`
	StopRationale             string                           `json:"stop_rationale"`
	ResidualGaps              []planningGapProjection          `json:"residual_gaps"`
	EvidenceThatWouldChange   string                           `json:"evidence_that_would_change"`
	SemanticDelta             planningSemanticDeltaProjection  `json:"semantic_delta"`
	TimelineID                string                           `json:"timeline_id"`
	TimelineDigest            string                           `json:"timeline_digest"`
	RecommendationDisposition string                           `json:"recommendation_disposition"`
	RecommendationRationale   string                           `json:"recommendation_rationale"`
	RecommendationEvidenceIDs []string                         `json:"recommendation_evidence_ids"`
	RecommendationProducer    string                           `json:"recommendation_producer"`
	RecommendationProducerID  string                           `json:"recommendation_producer_id"`
	AcceptanceCommand         string                           `json:"acceptance_command,omitempty"`
	ExecutionActions          []string                         `json:"execution_actions,omitempty"`
}

type planningAcceptanceProjection struct {
	Screen                  string                           `json:"screen"`
	Identity                planningVisualIdentityProjection `json:"identity"`
	CandidateID             string                           `json:"candidate_id"`
	CandidateContentHash    string                           `json:"candidate_content_hash"`
	SpecificationRevisionID string                           `json:"specification_revision_id"`
	PlanRevisionID          string                           `json:"plan_revision_id"`
	TimelineID              string                           `json:"timeline_id"`
	TimelineDigest          string                           `json:"timeline_digest"`
	TimelinePasses          int                              `json:"timeline_passes"`
	State                   string                           `json:"state"`
	Replayed                bool                             `json:"replayed"`
	ExecutionActions        []string                         `json:"execution_actions"`
}

type planningRefusalProjection struct {
	Screen   string                           `json:"screen"`
	Identity planningVisualIdentityProjection `json:"identity"`
	Action   string                           `json:"action"`
	Because  string                           `json:"because"`
	State    string                           `json:"state"`
	Next     string                           `json:"next"`
}

func projectPlanningSpecification(result specCommandResult) planningSpecificationProjection {
	return planningSpecificationProjection{
		Screen: "specification", CommandLabel: "Specification",
		Identity:  planningVisualIdentityProjection{Caste: "queen", Label: "Queen"},
		Operation: result.Operation, SpecificationID: result.SpecificationID,
		BeforeRevisionID: result.BeforeRevisionID, AfterRevisionID: result.AfterRevisionID,
		RevisionNumber: result.RevisionNumber, Status: result.Status, Scope: result.Scope,
		Outcomes: result.Outcomes, IncludedBehaviors: result.IncludedBehaviors,
		Exclusions: result.Exclusions, BindingDecisions: result.BindingDecisions,
		Requirements: result.Requirements, AcceptanceChecks: result.AcceptanceChecks,
		NegativeExpectations: result.NegativeExpectations, RecoveryExpectations: result.RecoveryExpectations,
		AffectedPublicPaths: result.AffectedPublicPaths, NextAction: result.NextAction,
	}
}

func projectPlanningIteration(card colony.PlanningIterationCard) planningIterationProjection {
	scores, before, after := planningAssessmentProjection(card.DimensionAssessments)
	decision := planningDecisionMode(card.Decision.Reason)
	return planningIterationProjection{
		Screen:     "planning_iteration",
		Identities: []planningVisualIdentityProjection{{Caste: "scout", Label: "Scout"}, {Caste: "route_setter", Label: "Route-Setter"}},
		Iteration:  card.Iteration, EvidenceIDs: append([]string(nil), card.EvidenceIDs...), Scores: scores,
		OverallBefore: before, OverallAfter: after, WeakestGap: projectPlanningGap(card.WeakestGap),
		SemanticDelta: projectPlanningSemanticDelta(card.SemanticDelta), Decision: decision,
		StopReason: string(card.Decision.Reason), StopReasonPublicLabel: planningStopReasonPublicLabel(card.Decision.Reason),
		EvidenceThatWouldChange: firstNonEmptyPlanningText(card.EvidenceThatWouldChange, card.Decision.EvidenceThatWouldChange, card.WeakestGap.EvidenceThatWouldChange),
		DetailCommand:           fmt.Sprintf("aether plan --candidate --details --show-iteration %d", card.Iteration),
	}
}

func projectPlanningDecision(card planningDecisionCard) planningDecisionProjection {
	evidenceIDs := make([]string, 0, len(card.Evidence))
	for _, evidence := range card.Evidence {
		evidenceIDs = append(evidenceIDs, evidence.ID)
	}
	return planningDecisionProjection{
		Screen: "planning_decision", Identity: planningVisualIdentityProjection{Caste: "queen", Label: "Queen"},
		DecisionID: card.DecisionID, Decision: card.Decision, WhyNow: card.WhyNow,
		EvidenceIDs: evidenceIDs, QueenRecommendation: card.QueenRecommendation,
		Choices:             append([]planningDecisionChoice(nil), card.Choices...),
		AffectedSemanticIDs: append([]string(nil), card.AffectedSemanticIDs...), PriorAnswer: card.PriorAnswer,
		Revalidation: card.Revalidation, PlanningResumes: card.PlanningResumes,
	}
}

func projectPlanningCandidate(review planCandidateReview) planningCandidateProjection {
	candidate := review.Candidate
	decision := review.StopDecision
	if decision.Reason == "" {
		decision = candidate.StopDecision
	}
	recommendation := review.Recommendation
	if recommendation.Disposition == "" {
		recommendation = candidate.Recommendation
	}
	gaps := review.ResidualGaps
	if len(gaps) == 0 {
		gaps = candidate.ResidualGaps
	}
	projectedGaps := make([]planningGapProjection, 0, len(gaps))
	for _, gap := range gaps {
		projectedGaps = append(projectedGaps, projectPlanningGap(gap))
	}
	scores := planningCandidateScores(review)
	active := candidate.Status == colony.PlanCandidateAccepted && candidate.Acceptance != nil
	projection := planningCandidateProjection{
		Screen: "plan_candidate", Identity: planningVisualIdentityProjection{Caste: "queen", Label: "Queen"},
		CandidateID: candidate.ID, CandidateStatus: string(candidate.Status), CandidateActive: active,
		SpecificationRevisionID: candidate.SpecificationRevisionID, BasePlanRevisionID: candidate.BasePlanRevisionID,
		ProposalRevisionID: candidate.Proposal.ID, TargetConfidence: review.TargetConfidence,
		ActualConfidence: review.ActualConfidence, Scores: scores,
		StopReason: string(decision.Reason), StopReasonPublicLabel: planningStopReasonPublicLabel(decision.Reason),
		StopRationale: decision.Rationale, ResidualGaps: projectedGaps,
		EvidenceThatWouldChange:   firstNonEmptyPlanningText(review.EvidenceThatWouldChange, candidate.EvidenceThatWouldChange, decision.EvidenceThatWouldChange),
		SemanticDelta:             projectPlanningSemanticDelta(firstPlanningSemanticDelta(review.SemanticDelta, candidate.SemanticDelta)),
		TimelineID:                firstNonEmptyPlanningText(review.Timeline.ID, candidate.Timeline.ID),
		TimelineDigest:            firstNonEmptyPlanningText(review.Timeline.TimelineDigest, candidate.Timeline.TimelineDigest),
		RecommendationDisposition: string(recommendation.Disposition), RecommendationRationale: recommendation.Rationale,
		RecommendationEvidenceIDs: append([]string(nil), recommendation.EvidenceIDs...),
		RecommendationProducer:    string(recommendation.Producer), RecommendationProducerID: recommendation.ProducerID,
	}
	if active {
		projection.ExecutionActions = []string{"aether build", "aether run"}
	} else {
		projection.AcceptanceCommand = review.AcceptanceCommand
	}
	return projection
}

func renderPlanningSpecificationVisual(result specCommandResult, options planningVisualOptions) string {
	projection := projectPlanningSpecification(result)
	var builder strings.Builder
	builder.WriteString(renderBanner("📜", projection.CommandLabel))
	builder.WriteString("Command: 📜 Specification\n")
	builder.WriteString("Identity: ")
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Operation: %s\n", projection.Operation)
	fmt.Fprintf(&builder, "SPEC: %s revision %d\n", projection.SpecificationID, projection.RevisionNumber)
	fmt.Fprintf(&builder, "Before: %s\nAfter: %s\n", projection.BeforeRevisionID, projection.AfterRevisionID)
	fmt.Fprintf(&builder, "Status: %s\nScope: %s\n", strings.ToUpper(string(projection.Status)), projection.Scope.Kind)
	renderPlanningSpecSection(&builder, "What this goal delivers", specCommandOutcomeVisualItems(projection.Outcomes))
	renderPlanningSpecSection(&builder, "Included", specCommandIncludedVisualItems(projection.IncludedBehaviors))
	renderPlanningSpecSection(&builder, "Explicitly excluded", specCommandExclusionVisualItems(projection.Exclusions))
	renderPlanningSpecSection(&builder, "Binding decisions", specCommandDecisionVisualItems(projection.BindingDecisions))
	renderPlanningSpecSection(&builder, "Requirements", specCommandRequirementVisualItems(projection.Requirements))
	renderPlanningSpecSection(&builder, "Owner-checkable acceptance", specCommandAcceptanceVisualItems(projection.AcceptanceChecks))
	renderPlanningSpecSection(&builder, "Negative expectations", specCommandNegativeVisualItems(projection.NegativeExpectations))
	renderPlanningSpecSection(&builder, "Recovery expectations", specCommandRecoveryVisualItems(projection.RecoveryExpectations))
	renderPlanningSpecSection(&builder, "Affected public paths", specCommandPublicPathVisualItems(projection.AffectedPublicPaths))
	builder.WriteString(renderStageMarker("Revision Impact"))
	renderSpecCommandVisualDelta(&builder, "Outcome", result.ClassifiedDelta.Outcomes)
	renderSpecCommandVisualDelta(&builder, "Included", result.ClassifiedDelta.IncludedBehaviors)
	renderSpecCommandVisualDelta(&builder, "Exclusions", result.ClassifiedDelta.Exclusions)
	renderSpecCommandVisualDelta(&builder, "Binding decisions", result.ClassifiedDelta.BindingDecisions)
	renderSpecCommandVisualDelta(&builder, "Requirements", result.ClassifiedDelta.Requirements)
	renderSpecCommandVisualDelta(&builder, "Acceptance", result.ClassifiedDelta.AcceptanceChecks)
	renderSpecCommandVisualDelta(&builder, "Negative", result.ClassifiedDelta.NegativeExpectations)
	renderSpecCommandVisualDelta(&builder, "Recovery", result.ClassifiedDelta.RecoveryExpectations)
	renderSpecCommandVisualDelta(&builder, "Public paths", result.ClassifiedDelta.AffectedPublicPaths)
	fmt.Fprintf(&builder, "Plan impact: tasks=%s proofs=%s\n", specCommandIDSummary(result.AffectedScope.TaskIDs), specCommandIDSummary(result.AffectedScope.ProofLinkIDs))
	builder.WriteString("Unaffected work: retained and still valid\n")
	if result.Replayed {
		builder.WriteString("Already recorded; the exact revision and receipt were retained.\n")
	}
	if result.Approval != nil {
		fmt.Fprintf(&builder, "Approval receipt: %s\n", result.Approval.ID)
		builder.WriteString("Specification approval does not accept or activate a plan.\n")
	}
	fmt.Fprintf(&builder, "State effect: %s\n", result.StateEffect)
	if projection.NextAction != "" {
		builder.WriteString(renderNextUp(projection.NextAction))
	}
	return builder.String()
}

func renderPlanningIterationVisual(card colony.PlanningIterationCard, options planningVisualOptions) string {
	projection := projectPlanningIteration(card)
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Planning Iteration"))
	builder.WriteString("Identity: ")
	builder.WriteString(casteIdentity("scout"))
	builder.WriteString(" → ")
	builder.WriteString(casteIdentity("route_setter"))
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Pass: %d\n", projection.Iteration)
	renderPlanningValueList(&builder, "Fresh evidence", projection.EvidenceIDs)
	builder.WriteString(renderStageMarker("Planning readiness"))
	for _, score := range projection.Scores {
		fmt.Fprintf(&builder, "%s: %d%% → %d%%\n", score.Dimension, score.Before, score.After)
	}
	fmt.Fprintf(&builder, "Overall: %d%% → %d%%\n", projection.OverallBefore, projection.OverallAfter)
	builder.WriteString(renderStageMarker("Weakest gap"))
	fmt.Fprintf(&builder, "%s (%s): %s\n", projection.WeakestGap.ID, projection.WeakestGap.Materiality, projection.WeakestGap.Description)
	fmt.Fprintf(&builder, "Evidence that would change it: %s\n", projection.WeakestGap.EvidenceThatWouldChange)
	builder.WriteString(renderStageMarker("Plan delta"))
	renderPlanningValueList(&builder, "Added", projection.SemanticDelta.Added)
	renderPlanningValueList(&builder, "Changed", projection.SemanticDelta.Changed)
	renderPlanningValueList(&builder, "Removed", projection.SemanticDelta.Removed)
	renderPlanningValueList(&builder, "Dependencies", projection.SemanticDelta.Dependencies)
	renderPlanningValueList(&builder, "Acceptance", projection.SemanticDelta.Acceptance)
	combinedPaths := append(append(append([]string(nil), projection.SemanticDelta.Negative...), projection.SemanticDelta.Recovery...), projection.SemanticDelta.PublicPaths...)
	renderPlanningValueList(&builder, "Negative / recovery / public paths", combinedPaths)
	renderPlanningValueList(&builder, "Authority impact", projection.SemanticDelta.AuthorityImpact)
	builder.WriteString(renderStageMarker("Decision"))
	fmt.Fprintf(&builder, "%s — %s\n", projection.Decision, projection.StopReasonPublicLabel)
	fmt.Fprintf(&builder, "Evidence that would change: %s\n", projection.EvidenceThatWouldChange)
	if projection.Decision == "CONTINUE" {
		fmt.Fprintf(&builder, "Next research: %s\n", projection.EvidenceThatWouldChange)
	}
	fmt.Fprintf(&builder, "Details: %s\n", projection.DetailCommand)
	return builder.String()
}

func renderPlanningDecisionVisual(card planningDecisionCard, options planningVisualOptions) string {
	projection := projectPlanningDecision(card)
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Planning Decision"))
	builder.WriteString("Identity: ")
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Decision: %s — %s\n", projection.DecisionID, projection.Decision)
	fmt.Fprintf(&builder, "Why now: %s\n", projection.WhyNow)
	renderPlanningValueList(&builder, "Evidence", projection.EvidenceIDs)
	fmt.Fprintf(&builder, "Queen recommends: %s\n", projection.QueenRecommendation)
	builder.WriteString(renderStageMarker("Choices"))
	for _, choice := range projection.Choices {
		fmt.Fprintf(&builder, "%s: %s\n", choice.ID, choice.Label)
		fmt.Fprintf(&builder, "Consequence: %s\n", choice.Consequence)
		if choice.Impact.material() {
			fmt.Fprintf(&builder, "Contract impact: behavior=%s authority=%s scope=%s risk=%s acceptance=%s\n", choice.Impact.Behavior, choice.Impact.Authority, choice.Impact.Scope, choice.Impact.Risk, choice.Impact.Acceptance)
		}
	}
	renderPlanningValueList(&builder, "Affected scope", projection.AffectedSemanticIDs)
	fmt.Fprintf(&builder, "Prior answer: %s\n", emptyFallback(projection.PriorAnswer, "none"))
	fmt.Fprintf(&builder, "Revalidation: %s\n", projection.Revalidation)
	fmt.Fprintf(&builder, "Planning resumes: %s\n", projection.PlanningResumes)
	return builder.String()
}

func renderPlanningCandidateVisual(review planCandidateReview, options planningVisualOptions) string {
	projection := projectPlanningCandidate(review)
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Plan Candidate"))
	builder.WriteString("Review: Plan Candidate\n")
	builder.WriteString("Identity: ")
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString("\n")
	if projection.CandidateActive {
		builder.WriteString("ACCEPTED PLAN — ACTIVE\n")
	} else {
		builder.WriteString("CANDIDATE — NOT ACTIVE\n")
	}
	fmt.Fprintf(&builder, "Candidate: %s (%s)\n", projection.CandidateID, projection.CandidateStatus)
	fmt.Fprintf(&builder, "Approved specification: %s\n", projection.SpecificationRevisionID)
	fmt.Fprintf(&builder, "Base plan: %s\n", projection.BasePlanRevisionID)
	fmt.Fprintf(&builder, "Preset: target %d%%\n", projection.TargetConfidence)
	fmt.Fprintf(&builder, "Stopped because: %s\n", projection.StopReasonPublicLabel)
	if projection.StopRationale != "" {
		fmt.Fprintf(&builder, "Reason: %s\n", projection.StopRationale)
	}
	builder.WriteString(renderStageMarker("Planning readiness"))
	for _, score := range projection.Scores {
		fmt.Fprintf(&builder, "%s: %d%%\n", score.Dimension, score.After)
	}
	fmt.Fprintf(&builder, "Overall: %d%% (target %d%%)\n", projection.ActualConfidence, projection.TargetConfidence)
	builder.WriteString(renderStageMarker("Remaining gaps"))
	if len(projection.ResidualGaps) == 0 {
		builder.WriteString("none\n")
	}
	for _, gap := range projection.ResidualGaps {
		fmt.Fprintf(&builder, "%s (%s): %s\n", gap.ID, gap.Materiality, gap.Description)
		fmt.Fprintf(&builder, "Evidence that would change it: %s\n", gap.EvidenceThatWouldChange)
	}
	fmt.Fprintf(&builder, "Evidence that would change it: %s\n", projection.EvidenceThatWouldChange)
	builder.WriteString(renderStageMarker("Planning timeline"))
	fmt.Fprintf(&builder, "Timeline: %s\nDigest: %s\n", emptyFallback(projection.TimelineID, "not recorded"), emptyFallback(projection.TimelineDigest, "not recorded"))
	builder.WriteString(renderStageMarker("Proposed plan"))
	fmt.Fprintf(&builder, "Revision: %s\n", projection.ProposalRevisionID)
	for _, phase := range review.Candidate.Proposal.Phases {
		fmt.Fprintf(&builder, "Phase %d: %s\n", phase.ID, phase.Name)
	}
	builder.WriteString(renderStageMarker("Queen recommendation"))
	fmt.Fprintf(&builder, "%s — %s\n", strings.ToUpper(projection.RecommendationDisposition), projection.RecommendationRationale)
	renderPlanningValueList(&builder, "Evidence", projection.RecommendationEvidenceIDs)
	fmt.Fprintf(&builder, "Producer: %s (%s)\n", projection.RecommendationProducer, projection.RecommendationProducerID)
	if projection.CandidateActive {
		builder.WriteString("Owner acceptance is recorded; build and run are equal execution choices.\n")
		builder.WriteString(renderNextUp("aether build  |  aether run"))
	} else {
		builder.WriteString("Candidate remains inactive until explicit owner acceptance.\n")
		builder.WriteString("Accept this candidate?\n")
		fmt.Fprintf(&builder, "  %s\n", projection.AcceptanceCommand)
	}
	return builder.String()
}

func renderPlanningPresetVisual(goal, specificationRevisionID string, selection planningPresetSelection, options planningVisualOptions) string {
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Plan"))
	builder.WriteString("Identity: ")
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Goal: %s\n", emptyFallback(goal, "Unreported"))
	fmt.Fprintf(&builder, "Planning contract: SPEC %s [APPROVED]\n", emptyFallback(specificationRevisionID, "Unreported"))
	if selection.PresetRequired {
		builder.WriteString(renderStageMarker("Choose Planning Preset"))
		for _, option := range selection.Options {
			fmt.Fprintf(&builder, "%-10s Target %-3d Up to %d passes\n", option.Label, option.TargetConfidence, option.PassCap)
		}
		builder.WriteString("Choose the planning preset: Fast, Balanced, Deep, or Exhaustive.\n")
		builder.WriteString("Planning did not start. State: unchanged.\n")
		return builder.String()
	}
	fmt.Fprintf(&builder, "Preset: %s — target %d, up to %d passes", selection.Policy.Label, selection.Policy.TargetConfidence, selection.Policy.PassCap)
	if selection.SelectionSource == planningPresetSourceNamed || selection.SelectionSource == planningPresetSourceExplicitPair {
		builder.WriteString(" — supplied by owner flag")
	}
	builder.WriteString(".\n")
	return builder.String()
}

func renderPlanningStageVisual(manifest planningStageManifest, status string, workerName string, completed bool, freshEvidence, remainingQuestions int, options planningVisualOptions) string {
	caste := string(manifest.ExpectedCaste)
	label := "Planning Stage"
	verb := "Work from the approved planning boundary"
	if manifest.ExpectedCaste == planningStageCasteScout {
		label = "Scout"
		verb = "Investigate " + planningManifestGapDescription(manifest.WeakestGap, "the first-pass evidence scope")
	} else if manifest.ExpectedCaste == planningStageCasteRouteSetter {
		label = "Route-Setter"
		scoutReceiptID := "Unreported"
		if manifest.ScoutReceipt != nil {
			scoutReceiptID = planningVisualShortIdentity(manifest.ScoutReceipt.ID)
		}
		verb = "Improve the route from Scout receipt " + scoutReceiptID
	}
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), fmt.Sprintf("Pass %d · %s", manifest.Pass, label)))
	builder.WriteString("Identity: ")
	builder.WriteString(casteIdentity(caste))
	if strings.TrimSpace(workerName) != "" {
		builder.WriteString(" ")
		builder.WriteString(workerName)
	}
	builder.WriteString("\n")
	if completed {
		fmt.Fprintf(&builder, "✓ %s %s [completed]\n", label, emptyFallback(workerName, "worker"))
		fmt.Fprintf(&builder, "Fresh evidence: %d\n", freshEvidence)
		fmt.Fprintf(&builder, "Remaining material questions: %d\n", remainingQuestions)
	} else {
		fmt.Fprintf(&builder, "%s — %s [%s]\n", label, verb, emptyFallback(status, "started"))
		bindings := make([]string, 0, len(manifest.EvidenceFrontier))
		for _, evidence := range manifest.EvidenceFrontier {
			bindings = append(bindings, evidence.ID)
		}
		renderPlanningValueList(&builder, "Evidence sources", bindings)
	}
	return builder.String()
}

func renderPlanningStopVisual(decision colony.PlanningStopDecision, residualGaps []colony.PlanningGap, score, target int, preset string, options planningVisualOptions) string {
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Planning Stop"))
	builder.WriteString("Identity: ")
	builder.WriteString(casteIdentity("route_setter"))
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "%s: %s — %s\n", planningDecisionMode(decision.Reason), planningStopReasonPublicLabel(decision.Reason), decision.Rationale)
	if score < target && decision.Reason != colony.PlanningStopContinue && decision.Reason != colony.PlanningStopOwnerDecision {
		fmt.Fprintf(&builder, "Below target: %d/%d. Remaining gaps are non-material because %s.\n", score, target, decision.Rationale)
	}
	builder.WriteString(renderStageMarker("Residual gaps"))
	if len(residualGaps) == 0 {
		builder.WriteString("none\n")
	}
	for _, gap := range residualGaps {
		fmt.Fprintf(&builder, "%s (%s): %s\n", gap.ID, gap.Materiality, gap.Description)
		fmt.Fprintf(&builder, "Evidence that would change it: %s\n", gap.EvidenceThatWouldChange)
	}
	fmt.Fprintf(&builder, "Evidence that would change this decision: %s\n", decision.EvidenceThatWouldChange)
	if preset != "" {
		fmt.Fprintf(&builder, "Preset: %s\n", preset)
	}
	builder.WriteString("Stopping creates a candidate only; the active plan is unchanged.\n")
	return builder.String()
}

func projectPlanningAcceptance(candidate colony.PlanCandidate, revision colony.PlanRevision, receipt colony.PlanAcceptanceReceipt, replayed bool) planningAcceptanceProjection {
	return planningAcceptanceProjection{
		Screen: "plan_acceptance", Identity: planningVisualIdentityProjection{Caste: "queen", Label: "Queen"},
		CandidateID: candidate.ID, CandidateContentHash: candidate.ContentHash,
		SpecificationRevisionID: receipt.SpecificationRevisionID, PlanRevisionID: revision.ID,
		TimelineID: receipt.TimelineID, TimelineDigest: receipt.TimelineDigest,
		TimelinePasses: len(candidate.Timeline.CardIDs), State: "ready", Replayed: replayed,
		ExecutionActions: []string{"aether build", "aether run"},
	}
}

func renderPlanningAcceptanceVisual(candidate colony.PlanCandidate, revision colony.PlanRevision, receipt colony.PlanAcceptanceReceipt, replayed bool, options planningVisualOptions) string {
	projection := projectPlanningAcceptance(candidate, revision, receipt, replayed)
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Plan Accepted"))
	builder.WriteString("Identity: ")
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString("\n")
	if projection.Replayed {
		builder.WriteString("Already accepted; the existing plan revision and receipt were retained.\n")
	} else {
		builder.WriteString("✓ Plan accepted\n")
	}
	fmt.Fprintf(&builder, "Plan revision: %s\n", projection.PlanRevisionID)
	fmt.Fprintf(&builder, "Candidate: %s (%s)\n", projection.CandidateID, planningVisualShortIdentity(projection.CandidateContentHash))
	fmt.Fprintf(&builder, "SPEC: %s\n", projection.SpecificationRevisionID)
	fmt.Fprintf(&builder, "Timeline: %d pass(es), digest %s\n", projection.TimelinePasses, planningVisualShortIdentity(projection.TimelineDigest))
	builder.WriteString("State: accepted plan is READY\n")
	builder.WriteString("Next Up: choose an operating mode\n")
	builder.WriteString("  aether build\n")
	builder.WriteString("  aether run\n")
	return builder.String()
}

func renderPlanningRevisionImpactVisual(reason string, evidence, requirements, tasks, proofs, completed, unaffected []string, historicalRevision, authority string, options planningVisualOptions) string {
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Living Plan Impact"))
	builder.WriteString("Identity: ")
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Why reality changed the route: %s\n", reason)
	renderPlanningValueList(&builder, "Evidence", evidence)
	renderPlanningValueList(&builder, "Affected requirements", requirements)
	renderPlanningValueList(&builder, "Affected unfinished tasks", tasks)
	renderPlanningValueList(&builder, "Affected proof links", proofs)
	renderPlanningValueList(&builder, "Preserved completed work", completed)
	renderPlanningValueList(&builder, "Preserved unaffected work", unaffected)
	fmt.Fprintf(&builder, "Historical revision: retained as %s\n", historicalRevision)
	fmt.Fprintf(&builder, "Authority required: %s\n", emptyFallback(authority, "none"))
	return builder.String()
}

func projectPlanningRefusal(action, because, state, next string) planningRefusalProjection {
	return planningRefusalProjection{
		Screen: "planning_refusal", Identity: planningVisualIdentityProjection{Caste: "queen", Label: "Queen"},
		Action: action, Because: because, State: state, Next: next,
	}
}

func renderPlanningRefusalVisual(action, because, state, next string, options planningVisualOptions) string {
	projection := projectPlanningRefusal(action, because, state, next)
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Planning Refusal"))
	builder.WriteString("Identity: ")
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "⛔ %s\n", projection.Action)
	fmt.Fprintf(&builder, "Because: %s\n", projection.Because)
	fmt.Fprintf(&builder, "State: %s\n", projection.State)
	fmt.Fprintf(&builder, "Next: %s\n", projection.Next)
	return builder.String()
}

func renderPlanningSpecSection(builder *strings.Builder, title string, items []specCommandVisualItem) {
	builder.WriteString(renderStageMarker(title))
	if len(items) == 0 {
		builder.WriteString("none\n")
		return
	}
	for _, item := range items {
		fmt.Fprintf(builder, "%s  %s\n", item.ID, item.Description)
		if item.Detail != "" {
			fmt.Fprintf(builder, "  %s\n", item.Detail)
		}
	}
}

func renderPlanningValueList(builder *strings.Builder, label string, values []string) {
	fmt.Fprintf(builder, "%s: %s\n", label, planningValueSummary(values))
}

func planningValueSummary(values []string) string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			clean = append(clean, value)
		}
	}
	if len(clean) == 0 {
		return "none"
	}
	return strings.Join(clean, ", ")
}

func planningAssessmentProjection(assessments []colony.PlanningDimensionAssessment) ([]planningScoreProjection, int, int) {
	byDimension := make(map[colony.PlanningDimension]colony.PlanningDimensionAssessment, len(assessments))
	for _, assessment := range assessments {
		byDimension[assessment.Dimension] = assessment
	}
	scores := make([]planningScoreProjection, 0, len(colony.PlanningDimensions()))
	beforeScores := planningConfidenceScores{}
	afterScores := planningConfidenceScores{}
	for _, dimension := range colony.PlanningDimensions() {
		assessment := byDimension[dimension]
		scores = append(scores, planningScoreProjection{Dimension: planningDimensionLabel(dimension), Before: assessment.Before, After: assessment.After})
		setPlanningConfidenceDimension(&beforeScores, dimension, assessment.Before)
		setPlanningConfidenceDimension(&afterScores, dimension, assessment.After)
	}
	if len(scores) == 0 {
		return scores, 0, 0
	}
	return scores, planningConfidenceWeightedOverall(beforeScores), planningConfidenceWeightedOverall(afterScores)
}

func setPlanningConfidenceDimension(scores *planningConfidenceScores, dimension colony.PlanningDimension, value int) {
	switch dimension {
	case colony.PlanningDimensionKnowledge:
		scores.Knowledge = value
	case colony.PlanningDimensionRequirements:
		scores.Requirements = value
	case colony.PlanningDimensionRisks:
		scores.Risks = value
	case colony.PlanningDimensionDependencies:
		scores.Dependencies = value
	case colony.PlanningDimensionEffort:
		scores.Effort = value
	}
}

func planningCandidateScores(review planCandidateReview) []planningScoreProjection {
	if len(review.Scores) > 0 {
		byDimension := make(map[colony.PlanningDimension]int, len(review.Scores))
		for _, score := range review.Scores {
			byDimension[score.Dimension] = score.Score
		}
		result := make([]planningScoreProjection, 0, len(colony.PlanningDimensions()))
		for _, dimension := range colony.PlanningDimensions() {
			result = append(result, planningScoreProjection{Dimension: planningDimensionLabel(dimension), After: byDimension[dimension]})
		}
		return result
	}
	projected, _, _ := planningAssessmentProjection(review.Candidate.DimensionAssessments)
	return projected
}

func planningDimensionLabel(dimension colony.PlanningDimension) string {
	value := strings.ReplaceAll(string(dimension), "_", " ")
	if value == "" {
		return "Unknown"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func projectPlanningGap(gap colony.PlanningGap) planningGapProjection {
	return planningGapProjection{ID: gap.ID, Dimension: string(gap.Dimension), Materiality: string(gap.Materiality), Description: gap.Description, EvidenceThatWouldChange: gap.EvidenceThatWouldChange}
}

func projectPlanningSemanticDelta(delta colony.PlanningSemanticDelta) planningSemanticDeltaProjection {
	projection := planningSemanticDeltaProjection{}
	all := [][]colony.PlanningSemanticChange{delta.Phases, delta.Tasks, delta.RequirementLinks}
	for _, section := range all {
		for _, change := range section {
			appendPlanningSemanticChange(&projection, change)
		}
	}
	projection.Dependencies = planningSemanticIDs(delta.Dependencies)
	projection.Acceptance = planningSemanticIDs(delta.AcceptanceChecks)
	projection.Negative = planningSemanticIDs(delta.NegativeExpectations)
	projection.Recovery = planningSemanticIDs(delta.RecoveryExpectations)
	projection.PublicPaths = planningSemanticIDs(delta.PublicPaths)
	for _, impact := range delta.AuthorityImpacts {
		projection.AuthorityImpact = append(projection.AuthorityImpact, fmt.Sprintf("%s: %s", impact.Kind, impact.Rationale))
	}
	return projection
}

func appendPlanningSemanticChange(projection *planningSemanticDeltaProjection, change colony.PlanningSemanticChange) {
	switch string(change.Kind) {
	case "added":
		projection.Added = append(projection.Added, change.SemanticID)
	case "modified":
		projection.Changed = append(projection.Changed, change.SemanticID)
	case "removed":
		projection.Removed = append(projection.Removed, change.SemanticID)
	case "preserved":
		projection.Preserved = append(projection.Preserved, change.SemanticID)
	}
}

func planningSemanticIDs(changes []colony.PlanningSemanticChange) []string {
	values := make([]string, 0, len(changes))
	for _, change := range changes {
		values = append(values, fmt.Sprintf("%s (%s)", change.SemanticID, change.Kind))
	}
	return values
}

func planningStopReasonPublicLabel(reason colony.PlanningStopReason) string {
	switch reason {
	case colony.PlanningStopTargetMet:
		return "target sufficiency"
	case colony.PlanningStopDiminishingReturns:
		return "diminishing returns"
	case colony.PlanningStopStalledGap:
		return "stall detected"
	case colony.PlanningStopPassCap:
		return "iteration cap"
	case colony.PlanningStopOwnerDecision:
		return "owner decision required"
	case colony.PlanningStopContinue:
		return "more evidence can improve the plan"
	default:
		return "not determined"
	}
}

func planningDecisionMode(reason colony.PlanningStopReason) string {
	switch reason {
	case colony.PlanningStopContinue:
		return "CONTINUE"
	case colony.PlanningStopOwnerDecision:
		return "PAUSE"
	default:
		return "STOP"
	}
}

func firstNonEmptyPlanningText(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func firstPlanningSemanticDelta(values ...colony.PlanningSemanticDelta) colony.PlanningSemanticDelta {
	for _, value := range values {
		if value.ID != "" || len(value.Phases)+len(value.Tasks)+len(value.Dependencies)+len(value.RequirementLinks)+len(value.AcceptanceChecks)+len(value.NegativeExpectations)+len(value.RecoveryExpectations)+len(value.PublicPaths)+len(value.AuthorityImpacts) > 0 {
			return value
		}
	}
	return colony.PlanningSemanticDelta{}
}

func planningManifestGapDescription(gap *colony.PlanningGap, fallback string) string {
	if gap == nil || strings.TrimSpace(gap.Description) == "" {
		return fallback
	}
	return gap.Description
}

func planningVisualShortIdentity(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 12 {
		return emptyFallback(value, "Unreported")
	}
	return value[:12]
}
