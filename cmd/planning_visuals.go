package cmd

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planningVisualOptions keeps terminal presentation choices outside canonical
// planning state. Renderers may fold or expand the same facts, but never infer
// authority from presentation state.
type planningVisualOptions struct {
	Width  int
	Detail bool
}

type planningVisualWidthBand string

const (
	planningVisualBandStacked planningVisualWidthBand = "stacked"
	planningVisualBandCompact planningVisualWidthBand = "compact"
	planningVisualBandTable   planningVisualWidthBand = "table"
	planningVisualBandWide    planningVisualWidthBand = "wide"
)

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
	CandidateContentHash      string                           `json:"candidate_content_hash"`
	SpecificationRevisionID   string                           `json:"specification_revision_id"`
	SpecificationRevisionHash string                           `json:"specification_revision_hash"`
	BasePlanRevisionID        string                           `json:"base_plan_revision_id"`
	BasePlanRevisionHash      string                           `json:"base_plan_revision_hash"`
	ProposalRevisionID        string                           `json:"proposal_revision_id"`
	ProposalHash              string                           `json:"proposal_hash"`
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
	Standing                  planCandidateStanding            `json:"standing"`
	ExpiresAt                 time.Time                        `json:"expires_at"`
	WhyUnavailable            string                           `json:"why_unavailable,omitempty"`
	Evidence                  []string                         `json:"evidence,omitempty"`
	AcceptanceAvailable       bool                             `json:"acceptance_available"`
	StateEffect               planCandidateStateEffect         `json:"state_effect"`
	ActivePlanEffect          planCandidateActivePlanEffect    `json:"active_plan_effect"`
	AcceptanceCommand         string                           `json:"acceptance_command,omitempty"`
	ExecutionActions          []string                         `json:"execution_actions,omitempty"`
	Next                      string                           `json:"next"`
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
	standing := review.Standing
	if standing == "" {
		if candidate.Status == colony.PlanCandidateAccepted && candidate.Acceptance != nil {
			standing = planCandidateStandingAccepted
		} else {
			standing = planCandidateStandingCurrent
		}
	}
	active := standing == planCandidateStandingAccepted && candidate.Status == colony.PlanCandidateAccepted && candidate.Acceptance != nil
	projection := planningCandidateProjection{
		Screen: "plan_candidate", Identity: planningVisualIdentityProjection{Caste: "queen", Label: "Queen"},
		CandidateID: candidate.ID, CandidateStatus: string(candidate.Status), CandidateActive: active,
		CandidateContentHash:    candidate.ContentHash,
		SpecificationRevisionID: candidate.SpecificationRevisionID, SpecificationRevisionHash: candidate.SpecificationRevisionHash,
		BasePlanRevisionID: candidate.BasePlanRevisionID, BasePlanRevisionHash: candidate.BasePlanRevisionHash,
		ProposalRevisionID: candidate.Proposal.ID, ProposalHash: candidate.ProposalHash, TargetConfidence: review.TargetConfidence,
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
		Standing: standing, ExpiresAt: candidate.ExpiresAt.UTC(),
		StateEffect: planCandidateStateEffectUnchanged, ActivePlanEffect: planCandidateActivePlanEffectUnchanged,
	}
	if review.Refusal != nil {
		projection.Standing = review.Refusal.Standing
		projection.ExpiresAt = review.Refusal.ExpiresAt.UTC()
		projection.WhyUnavailable = review.Refusal.WhyUnavailable
		projection.Evidence = append([]string(nil), review.Refusal.Evidence...)
		projection.StateEffect = review.Refusal.StateEffect
		projection.ActivePlanEffect = review.Refusal.ActivePlanEffect
		projection.Next = review.Refusal.RecoveryCommand
	} else if active {
		projection.ExecutionActions = []string{"aether build", "aether run"}
		projection.Next = strings.Join(projection.ExecutionActions, " | ")
	} else if standing == planCandidateStandingCurrent && strings.TrimSpace(review.AcceptanceCommand) != "" {
		projection.AcceptanceAvailable = true
		projection.AcceptanceCommand = review.AcceptanceCommand
		projection.Next = review.AcceptanceCommand
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
	renderPlanningSpecSection(&builder, "goal", "What this goal delivers", specCommandOutcomeVisualItems(projection.Outcomes))
	renderPlanningSpecSection(&builder, "done", "Included", specCommandIncludedVisualItems(projection.IncludedBehaviors))
	renderPlanningSpecSection(&builder, "avoid", "Explicitly excluded", specCommandExclusionVisualItems(projection.Exclusions))
	renderPlanningSpecSection(&builder, "decision", "Binding decisions", specCommandDecisionVisualItems(projection.BindingDecisions))
	renderPlanningSpecSection(&builder, "requirement", "Requirements", specCommandRequirementVisualItems(projection.Requirements))
	renderPlanningSpecSection(&builder, "evidence", "Owner-checkable acceptance", specCommandAcceptanceVisualItems(projection.AcceptanceChecks))
	renderPlanningSpecSection(&builder, "avoid", "Negative expectations", specCommandNegativeVisualItems(projection.NegativeExpectations))
	renderPlanningSpecSection(&builder, "checkpoint", "Recovery expectations", specCommandRecoveryVisualItems(projection.RecoveryExpectations))
	renderPlanningSpecSection(&builder, "files", "Affected public paths", specCommandPublicPathVisualItems(projection.AffectedPublicPaths))
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
	return finalizePlanningVisual(builder.String(), options)
}

func renderPlanningIterationVisual(card colony.PlanningIterationCard, options planningVisualOptions) string {
	projection := projectPlanningIteration(card)
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Planning Iteration"))
	builder.WriteString("Card: Planning Iteration\n")
	builder.WriteString(casteIdentity("scout"))
	builder.WriteString(" → ")
	builder.WriteString(casteIdentity("route_setter"))
	builder.WriteString("\n")
	builder.WriteString(voiceLine("phase", fmt.Sprintf("Pass: %d", projection.Iteration)) + "\n")
	renderPlanningValueList(&builder, "evidence", "Fresh evidence", projection.EvidenceIDs)
	builder.WriteString(renderStageMarker("Planning readiness"))
	for _, score := range projection.Scores {
		delta := score.After - score.Before
		switch planningVisualBandForWidth(options.Width) {
		case planningVisualBandStacked:
			builder.WriteString(voiceLine("evidence", score.Dimension) + "\n")
			builder.WriteString(voiceLine("evidence", fmt.Sprintf("  Before: %d%%", score.Before)) + "\n")
			builder.WriteString(voiceLine("evidence", fmt.Sprintf("  After: %d%% (%+d)", score.After, delta)) + "\n")
		case planningVisualBandCompact:
			builder.WriteString(voiceLine("evidence", fmt.Sprintf("%s  %d%% → %d%%", score.Dimension, score.Before, score.After)) + "\n")
		case planningVisualBandTable:
			builder.WriteString(voiceLine("evidence", fmt.Sprintf("%s | Before %d%% | After %d%% | %+d", score.Dimension, score.Before, score.After, delta)) + "\n")
		default:
			builder.WriteString(voiceLine("evidence", fmt.Sprintf("%-12s %d%% → %d%% (%+d)", score.Dimension, score.Before, score.After, delta)) + "\n")
		}
	}
	builder.WriteString(voiceLine("evidence", fmt.Sprintf("Overall: %d%% → %d%%", projection.OverallBefore, projection.OverallAfter)) + "\n")
	builder.WriteString(renderStageMarker("Weakest gap"))
	builder.WriteString(voiceLine("blocked", fmt.Sprintf("%s (%s): %s", projection.WeakestGap.ID, planningEnumLabel(projection.WeakestGap.Materiality), projection.WeakestGap.Description)) + "\n")
	builder.WriteString(voiceLine("question", fmt.Sprintf("Evidence that would change it: %s", projection.WeakestGap.EvidenceThatWouldChange)) + "\n")
	builder.WriteString(renderStageMarker("Plan delta"))
	renderPlanningValueList(&builder, "history", "Added", projection.SemanticDelta.Added)
	renderPlanningValueList(&builder, "history", "Changed", projection.SemanticDelta.Changed)
	renderPlanningValueList(&builder, "history", "Removed", projection.SemanticDelta.Removed)
	renderPlanningValueList(&builder, "history", "Dependencies", projection.SemanticDelta.Dependencies)
	renderPlanningValueList(&builder, "history", "Acceptance", projection.SemanticDelta.Acceptance)
	combinedPaths := append(append(append([]string(nil), projection.SemanticDelta.Negative...), projection.SemanticDelta.Recovery...), projection.SemanticDelta.PublicPaths...)
	renderPlanningValueList(&builder, "history", "Negative / recovery / public paths", combinedPaths)
	renderPlanningValueList(&builder, "history", "Authority impact", projection.SemanticDelta.AuthorityImpact)
	builder.WriteString(renderStageMarker("Decision"))
	builder.WriteString(voiceLine("blocked", fmt.Sprintf("%s — %s", projection.Decision, projection.StopReasonPublicLabel)) + "\n")
	builder.WriteString(voiceLine("question", fmt.Sprintf("Evidence that would change: %s", projection.EvidenceThatWouldChange)) + "\n")
	if projection.Decision == "CONTINUE" {
		builder.WriteString(voiceLine("evidence", fmt.Sprintf("Next research: %s", projection.EvidenceThatWouldChange)) + "\n")
	}
	builder.WriteString(voiceLine("next", fmt.Sprintf("Details: %s", projection.DetailCommand)) + "\n")
	return finalizePlanningVisual(builder.String(), options)
}

func renderPlanningDecisionVisual(card planningDecisionCard, options planningVisualOptions) string {
	projection := projectPlanningDecision(card)
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Planning Decision"))
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString(" (Queen is this project's coordinator)\n")
	builder.WriteString(voiceLine("decision", fmt.Sprintf("Decision: %s — %s", projection.DecisionID, projection.Decision)) + "\n")
	builder.WriteString(voiceLine("decision", fmt.Sprintf("Why now: %s", projection.WhyNow)) + "\n")
	renderPlanningValueList(&builder, "evidence", "Evidence", projection.EvidenceIDs)
	builder.WriteString(voiceLine("decision", fmt.Sprintf("Queen recommends (this project's coordinator): %s", projection.QueenRecommendation)) + "\n")
	builder.WriteString(renderStageMarker("Choices"))
	for _, choice := range projection.Choices {
		builder.WriteString(voiceLine("alternative", fmt.Sprintf("%s: %s", choice.ID, choice.Label)) + "\n")
		builder.WriteString(voiceLine("alternative", fmt.Sprintf("Consequence: %s", choice.Consequence)) + "\n")
		if choice.Impact.material() {
			// Sentence form, not a bookkeeping key=value list (RESEARCH.md
			// criterion 4) -- an owner-facing line never carries that shape.
			builder.WriteString(voiceLine("alternative", fmt.Sprintf("Contract impact — behavior: %s; authority: %s; scope: %s; risk: %s; acceptance: %s", choice.Impact.Behavior, choice.Impact.Authority, choice.Impact.Scope, choice.Impact.Risk, choice.Impact.Acceptance)) + "\n")
		}
	}
	renderPlanningValueList(&builder, "history", "Affected scope", projection.AffectedSemanticIDs)
	builder.WriteString(voiceLine("history", fmt.Sprintf("Prior answer: %s", emptyFallback(projection.PriorAnswer, "none"))) + "\n")
	builder.WriteString(voiceLine("question", fmt.Sprintf("Revalidation: %s", projection.Revalidation)) + "\n")
	builder.WriteString(voiceLine("next", fmt.Sprintf("Planning resumes: %s", projection.PlanningResumes)) + "\n")
	return finalizePlanningVisual(builder.String(), options)
}

func renderPlanningCandidateVisual(review planCandidateReview, options planningVisualOptions) string {
	projection := projectPlanningCandidate(review)
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Plan Candidate"))
	builder.WriteString("Review: Plan Candidate\n")
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString(" (Queen is this project's coordinator)\n")
	if projection.CandidateActive {
		builder.WriteString(voiceLine("done", "ACCEPTED PLAN — ACTIVE") + "\n")
	} else {
		builder.WriteString(voiceLine("status", "CANDIDATE — NOT ACTIVE") + "\n")
	}
	builder.WriteString(voiceLine("history", fmt.Sprintf("Candidate: %s (%s)", projection.CandidateID, planningEnumLabel(projection.CandidateStatus))) + "\n")
	builder.WriteString(voiceLine("history", fmt.Sprintf("Candidate hash: %s", projection.CandidateContentHash)) + "\n")
	builder.WriteString(voiceLine("history", fmt.Sprintf("Approved specification: %s", projection.SpecificationRevisionID)) + "\n")
	builder.WriteString(voiceLine("history", fmt.Sprintf("SPEC hash: %s", projection.SpecificationRevisionHash)) + "\n")
	builder.WriteString(voiceLine("history", fmt.Sprintf("Base plan: %s", projection.BasePlanRevisionID)) + "\n")
	builder.WriteString(voiceLine("history", fmt.Sprintf("Base plan hash: %s", projection.BasePlanRevisionHash)) + "\n")
	builder.WriteString(voiceLine("history", fmt.Sprintf("Proposal hash: %s", projection.ProposalHash)) + "\n")
	builder.WriteString(voiceLine("status", fmt.Sprintf("Standing: %s", projection.Standing)) + "\n")
	builder.WriteString(voiceLine("elapsed", fmt.Sprintf("Expires: %s", projection.ExpiresAt.Format(time.RFC3339Nano))) + "\n")
	builder.WriteString(voiceLine("status", fmt.Sprintf("Active plan: %s", projection.ActivePlanEffect)) + "\n")
	builder.WriteString(voiceLine("decision", fmt.Sprintf("Preset: target %d%%", projection.TargetConfidence)) + "\n")
	builder.WriteString(voiceLine("blocked", fmt.Sprintf("Stopped because: %s", projection.StopReasonPublicLabel)) + "\n")
	if projection.StopRationale != "" {
		builder.WriteString(voiceLine("blocked", fmt.Sprintf("Reason: %s", projection.StopRationale)) + "\n")
	}
	builder.WriteString(renderStageMarker("Planning readiness"))
	for _, score := range projection.Scores {
		builder.WriteString(voiceLine("evidence", fmt.Sprintf("%s: %d%%", score.Dimension, score.After)) + "\n")
	}
	builder.WriteString(voiceLine("evidence", fmt.Sprintf("Overall: %d%% (target %d%%)", projection.ActualConfidence, projection.TargetConfidence)) + "\n")
	builder.WriteString(renderStageMarker("Remaining gaps"))
	if len(projection.ResidualGaps) == 0 {
		builder.WriteString(voiceLine("blocked", "none") + "\n")
	}
	for _, gap := range projection.ResidualGaps {
		builder.WriteString(voiceLine("blocked", fmt.Sprintf("%s (%s): %s", gap.ID, planningEnumLabel(gap.Materiality), gap.Description)) + "\n")
		builder.WriteString(voiceLine("question", fmt.Sprintf("Evidence that would change it: %s", gap.EvidenceThatWouldChange)) + "\n")
	}
	builder.WriteString(voiceLine("question", fmt.Sprintf("Evidence that would change it: %s", projection.EvidenceThatWouldChange)) + "\n")
	builder.WriteString(renderStageMarker("Planning timeline"))
	builder.WriteString(voiceLine("history", fmt.Sprintf("Timeline: %s", emptyFallback(projection.TimelineID, "not recorded"))) + "\n")
	builder.WriteString(voiceLine("history", fmt.Sprintf("Digest: %s", emptyFallback(projection.TimelineDigest, "not recorded"))) + "\n")
	builder.WriteString(renderStageMarker("Proposed plan"))
	builder.WriteString(voiceLine("history", fmt.Sprintf("Revision: %s", projection.ProposalRevisionID)) + "\n")
	for _, phase := range review.Candidate.Proposal.Phases {
		builder.WriteString(voiceLine("phase", fmt.Sprintf("Phase %d: %s", phase.ID, phase.Name)) + "\n")
	}
	builder.WriteString(renderStageMarker("Queen recommendation (this project's coordinator)"))
	builder.WriteString(voiceLine("decision", fmt.Sprintf("%s — %s", strings.ToUpper(projection.RecommendationDisposition), projection.RecommendationRationale)) + "\n")
	renderPlanningValueList(&builder, "evidence", "Evidence", projection.RecommendationEvidenceIDs)
	builder.WriteString(voiceLine("history", fmt.Sprintf("Producer: %s (%s) — this project's coordinator decides the recommendation", projection.RecommendationProducer, projection.RecommendationProducerID)) + "\n")
	if projection.CandidateActive {
		builder.WriteString(voiceLine("done", "Owner acceptance is recorded; build and run are equal execution choices.") + "\n")
		builder.WriteString(voiceLine("next", fmt.Sprintf("Next: %s", projection.Next)) + "\n")
	} else if projection.AcceptanceAvailable {
		builder.WriteString(voiceLine("blocked", "Candidate remains inactive until explicit owner acceptance.") + "\n")
		builder.WriteString(voiceLine("decision", "Accept this candidate?") + "\n")
		builder.WriteString(voiceLine("decision", fmt.Sprintf("Acceptance command: %s", projection.AcceptanceCommand)) + "\n")
		builder.WriteString(voiceLine("next", fmt.Sprintf("Next: %s", projection.Next)) + "\n")
	} else {
		builder.WriteString(voiceLine("avoid", fmt.Sprintf("Why unavailable: %s", projection.WhyUnavailable)) + "\n")
		renderPlanningValueList(&builder, "evidence", "Evidence", projection.Evidence)
		builder.WriteString(voiceLine("status", fmt.Sprintf("State: %s", projection.StateEffect)) + "\n")
		builder.WriteString(voiceLine("next", fmt.Sprintf("Next: %s", projection.Next)) + "\n")
	}
	return finalizePlanningVisual(builder.String(), options)
}

func renderPlanningPresetVisual(goal, specificationRevisionID string, selection planningPresetSelection, options planningVisualOptions) string {
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Plan"))
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString(" (Queen is this project's coordinator)\n")
	builder.WriteString(voiceLine("goal", fmt.Sprintf("Goal: %s", emptyFallback(goal, "Unreported"))) + "\n")
	builder.WriteString(voiceLine("requirement", fmt.Sprintf("Planning contract: SPEC %s [APPROVED]", emptyFallback(specificationRevisionID, "Unreported"))) + "\n")
	if selection.PresetRequired {
		builder.WriteString(renderStageMarker("Choose Planning Preset"))
		for _, option := range selection.Options {
			builder.WriteString(voiceLine("alternative", fmt.Sprintf("%-10s Target %-3d Up to %d passes", option.Label, option.TargetConfidence, option.PassCap)) + "\n")
		}
		builder.WriteString(voiceLine("decision", "Choose the planning preset: Fast, Balanced, Deep, or Exhaustive.") + "\n")
		builder.WriteString(voiceLine("status", "Planning did not start. State: unchanged.") + "\n")
		return finalizePlanningVisual(builder.String(), options)
	}
	presetLine := fmt.Sprintf("Preset: %s — target %d, up to %d passes", selection.Policy.Label, selection.Policy.TargetConfidence, selection.Policy.PassCap)
	if selection.SelectionSource == planningPresetSourceNamed || selection.SelectionSource == planningPresetSourceExplicitPair {
		presetLine += " — supplied by owner flag"
	}
	builder.WriteString(voiceLine("decision", presetLine+".") + "\n")
	return finalizePlanningVisual(builder.String(), options)
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
	builder.WriteString(casteIdentity(caste))
	if strings.TrimSpace(workerName) != "" {
		builder.WriteString(" ")
		builder.WriteString(workerName)
	}
	builder.WriteString("\n")
	if completed {
		builder.WriteString(voiceLine("done", fmt.Sprintf("%s %s [completed]", label, emptyFallback(workerName, "worker"))) + "\n")
		builder.WriteString(voiceLine("evidence", fmt.Sprintf("Fresh evidence: %d", freshEvidence)) + "\n")
		builder.WriteString(voiceLine("question", fmt.Sprintf("Remaining material questions: %d", remainingQuestions)) + "\n")
	} else {
		builder.WriteString(voiceLine("status", fmt.Sprintf("%s — %s [%s]", label, verb, emptyFallback(status, "started"))) + "\n")
		bindings := make([]string, 0, len(manifest.EvidenceFrontier))
		for _, evidence := range manifest.EvidenceFrontier {
			bindings = append(bindings, evidence.ID)
		}
		renderPlanningValueList(&builder, "evidence", "Evidence sources", bindings)
	}
	return finalizePlanningVisual(builder.String(), options)
}

func renderPlanningStopVisual(decision colony.PlanningStopDecision, residualGaps []colony.PlanningGap, score, target int, preset string, options planningVisualOptions) string {
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Planning Stop"))
	builder.WriteString(casteIdentity("route_setter"))
	builder.WriteString("\n")
	builder.WriteString(voiceLine("blocked", fmt.Sprintf("%s: %s — %s", planningDecisionMode(decision.Reason), planningStopReasonPublicLabel(decision.Reason), decision.Rationale)) + "\n")
	if score < target && decision.Reason != colony.PlanningStopContinue && decision.Reason != colony.PlanningStopOwnerDecision {
		builder.WriteString(voiceLine("blocked", fmt.Sprintf("Below target: %d/%d. Remaining gaps are non-material because %s.", score, target, decision.Rationale)) + "\n")
	}
	builder.WriteString(renderStageMarker("Residual gaps"))
	if len(residualGaps) == 0 {
		builder.WriteString(voiceLine("blocked", "none") + "\n")
	}
	for _, gap := range residualGaps {
		builder.WriteString(voiceLine("blocked", fmt.Sprintf("%s (%s): %s", gap.ID, planningEnumLabel(string(gap.Materiality)), gap.Description)) + "\n")
		builder.WriteString(voiceLine("question", fmt.Sprintf("Evidence that would change it: %s", gap.EvidenceThatWouldChange)) + "\n")
	}
	builder.WriteString(voiceLine("question", fmt.Sprintf("Evidence that would change this decision: %s", decision.EvidenceThatWouldChange)) + "\n")
	if preset != "" {
		builder.WriteString(voiceLine("decision", fmt.Sprintf("Preset: %s", preset)) + "\n")
	}
	builder.WriteString(voiceLine("status", "Stopping creates a candidate only; the active plan is unchanged.") + "\n")
	return finalizePlanningVisual(builder.String(), options)
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
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString(" (Queen is this project's coordinator)\n")
	if projection.Replayed {
		builder.WriteString(voiceLine("done", "Already accepted; the existing plan revision and receipt were retained.") + "\n")
	} else {
		builder.WriteString(voiceLine("done", "Plan accepted") + "\n")
	}
	builder.WriteString(voiceLine("history", fmt.Sprintf("Plan revision: %s", projection.PlanRevisionID)) + "\n")
	builder.WriteString(voiceLine("history", fmt.Sprintf("Candidate: %s (%s)", projection.CandidateID, planningVisualShortIdentity(projection.CandidateContentHash))) + "\n")
	builder.WriteString(voiceLine("requirement", fmt.Sprintf("SPEC: %s", projection.SpecificationRevisionID)) + "\n")
	builder.WriteString(voiceLine("history", fmt.Sprintf("Timeline: %d pass(es), digest %s", projection.TimelinePasses, planningVisualShortIdentity(projection.TimelineDigest))) + "\n")
	builder.WriteString(voiceLine("status", "State: accepted plan is READY") + "\n")
	builder.WriteString(voiceLine("next", "Next Up: choose an operating mode") + "\n")
	builder.WriteString(voiceLine("alternative", "aether build") + "\n")
	builder.WriteString(voiceLine("alternative", "aether run") + "\n")
	return finalizePlanningVisual(builder.String(), options)
}

func renderPlanningRevisionImpactVisual(reason string, evidence, requirements, tasks, proofs, completed, unaffected []string, historicalRevision, authority string, options planningVisualOptions) string {
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Living Plan Impact"))
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString(" (Queen is this project's coordinator)\n")
	builder.WriteString(voiceLine("decision", fmt.Sprintf("Why reality changed the route: %s", reason)) + "\n")
	renderPlanningValueList(&builder, "evidence", "Evidence", evidence)
	renderPlanningValueList(&builder, "requirement", "Affected requirements", requirements)
	renderPlanningValueList(&builder, "task", "Affected unfinished tasks", tasks)
	renderPlanningValueList(&builder, "checkpoint", "Affected proof links", proofs)
	renderPlanningValueList(&builder, "done", "Preserved completed work", completed)
	renderPlanningValueList(&builder, "done", "Preserved unaffected work", unaffected)
	builder.WriteString(voiceLine("history", fmt.Sprintf("Historical revision: retained as %s", historicalRevision)) + "\n")
	builder.WriteString(voiceLine("decision", fmt.Sprintf("Authority required: %s", emptyFallback(authority, "none"))) + "\n")
	return finalizePlanningVisual(builder.String(), options)
}

func projectPlanningRefusal(action, because, state, next string) planningRefusalProjection {
	return planningRefusalProjection{
		Screen: "planning_refusal", Identity: planningVisualIdentityProjection{Caste: "queen", Label: "Queen"},
		Action: action, Because: because, State: state, Next: next,
	}
}

// renderCanonicalPlanningResult selects the narrow typed card represented by
// a plan command result. Returning false preserves the legacy renderer for
// pre-Phase-200 whole-plan summaries and repair operations.
func renderCanonicalPlanningResult(result map[string]interface{}, options planningVisualOptions) (string, bool) {
	switch planCandidateOperation(stringValue(result["operation"])) {
	case planCandidateOperationReview:
		if review, ok := planningCandidateReviewFromResult(result); ok {
			return renderPlanningCandidateVisual(review, options), true
		}
	case planCandidateOperationDetail:
		if card, ok := planningIterationCardValue(result["iteration"]); ok {
			return renderPlanningIterationVisual(card, planningVisualOptions{Width: options.Width, Detail: true}), true
		}
	case planCandidateOperationAccept:
		candidate, candidateOK := planningCandidateValue(result["candidate"])
		revision, revisionOK := planningPlanRevisionValue(result["revision"])
		receipt, receiptOK := planningAcceptanceReceiptValue(result["acceptance_receipt"])
		if candidateOK && revisionOK && receiptOK {
			replayed, _ := result["replayed"].(bool)
			return renderPlanningAcceptanceVisual(candidate, revision, receipt, replayed, options), true
		}
	}

	if card, ok := planningIterationCardValue(result["iteration_card"]); ok {
		return renderPlanningIterationVisual(card, options), true
	}
	if cards, ok := result["decision_cards"].([]planningDecisionCard); ok && len(cards) > 0 {
		return renderPlanningDecisionVisual(cards[0], options), true
	}
	if card, ok := result["decision_card"].(planningDecisionCard); ok {
		return renderPlanningDecisionVisual(card, options), true
	}
	if required, _ := result["preset_required"].(bool); required {
		selection := planningPresetSelection{
			PresetRequired: true, SelectionSource: stringValue(result["selection_source"]),
		}
		switch values := result["preset_options"].(type) {
		case []planningPresetPolicy:
			selection.Options = append([]planningPresetPolicy(nil), values...)
		case []interface{}:
			for _, value := range values {
				if option, ok := value.(planningPresetPolicy); ok {
					selection.Options = append(selection.Options, option)
				}
			}
		}
		visual := renderPlanningPresetVisual(stringValue(result["goal"]), planningSpecificationRevisionFromResult(result), selection, options)
		visual += renderLifecycleClosing(result, "plan")
		return finalizePlanningVisual(visual, options), true
	}
	if manifest, ok := planningStageManifestValue(result["stage_manifest"]); ok {
		return renderPlanningStageVisual(manifest, stringValue(result["status"]), planningWorkerNameFromResult(result, manifest.ExpectedCaste), false, 0, 0, options), true
	}
	if manifest, ok := planningStageManifestValue(result["route_stage_manifest"]); ok {
		return renderPlanningStageVisual(manifest, stringValue(result["status"]), planningWorkerNameFromResult(result, manifest.ExpectedCaste), false, 0, 0, options), true
	}
	if manifest, ok := planningStageManifestValue(result["scout_stage_manifest"]); ok {
		return renderPlanningStageVisual(manifest, stringValue(result["status"]), planningWorkerNameFromResult(result, manifest.ExpectedCaste), false, 0, 0, options), true
	}
	return "", false
}

// projectPlanningWorkflowResult adds presentation siblings to planning JSON
// without deleting or renaming any canonical field consumed by wrappers.
func projectPlanningWorkflowResult(result interface{}) interface{} {
	raw, ok := result.(map[string]interface{})
	if !ok {
		return result
	}
	projected := make(map[string]interface{}, len(raw)+3)
	for key, value := range raw {
		projected[key] = value
	}
	recognized := false
	switch planCandidateOperation(stringValue(raw["operation"])) {
	case planCandidateOperationReview:
		if review, ok := planningCandidateReviewFromResult(raw); ok {
			projection := projectPlanningCandidate(review)
			projected["planning_projection"] = projection
			projected["stop_reason_public_label"] = projection.StopReasonPublicLabel
			projected["evidence_that_would_change"] = projection.EvidenceThatWouldChange
			recognized = true
		}
	case planCandidateOperationDetail:
		if card, ok := planningIterationCardValue(raw["iteration"]); ok {
			projection := projectPlanningIteration(card)
			projected["planning_projection"] = projection
			projected["stop_reason_public_label"] = projection.StopReasonPublicLabel
			recognized = true
		}
	case planCandidateOperationAccept:
		candidate, candidateOK := planningCandidateValue(raw["candidate"])
		revision, revisionOK := planningPlanRevisionValue(raw["revision"])
		receipt, receiptOK := planningAcceptanceReceiptValue(raw["acceptance_receipt"])
		if candidateOK && revisionOK && receiptOK {
			replayed, _ := raw["replayed"].(bool)
			projected["planning_projection"] = projectPlanningAcceptance(candidate, revision, receipt, replayed)
			projected["execution_actions"] = []string{"aether build", "aether run"}
			recognized = true
		}
	}
	if card, ok := planningIterationCardValue(raw["iteration_card"]); ok {
		projection := projectPlanningIteration(card)
		projected["planning_projection"] = projection
		projected["stop_reason_public_label"] = projection.StopReasonPublicLabel
		recognized = true
	}
	if !recognized {
		return result
	}
	return projected
}

func planningCandidateReviewFromResult(result map[string]interface{}) (planCandidateReview, bool) {
	candidate, ok := planningCandidateValue(result["candidate"])
	if !ok {
		return planCandidateReview{}, false
	}
	review := planCandidateReview{
		Operation: planCandidateOperationReview, Candidate: candidate,
		TargetConfidence: intValue(result["target_confidence"]), ActualConfidence: intValue(result["actual_confidence"]),
		EvidenceThatWouldChange: stringValue(result["evidence_that_would_change"]),
		AcceptanceCommand:       stringValue(result["acceptance_command"]),
	}
	if values, ok := result["scores"].([]planCandidateConfidenceScore); ok {
		review.Scores = append([]planCandidateConfidenceScore(nil), values...)
	}
	if value, ok := result["stop_decision"].(colony.PlanningStopDecision); ok {
		review.StopDecision = value
	}
	if values, ok := result["residual_gaps"].([]colony.PlanningGap); ok {
		review.ResidualGaps = append([]colony.PlanningGap(nil), values...)
	}
	if value, ok := result["semantic_delta"].(colony.PlanningSemanticDelta); ok {
		review.SemanticDelta = value
	}
	if value, ok := result["recommendation"].(colony.QueenPlanRecommendation); ok {
		review.Recommendation = value
	}
	if value, ok := result["timeline"].(colony.PlanningTimelineBinding); ok {
		review.Timeline = value
	}
	if values, ok := result["iterations"].([]colony.PlanningIterationCard); ok {
		review.Iterations = append([]colony.PlanningIterationCard(nil), values...)
	}
	if value, ok := result["acceptance"].(planCandidateAcceptanceRequest); ok {
		review.Acceptance = value
	}
	if value, ok := result["standing"].(planCandidateStanding); ok {
		review.Standing = value
	} else if value := strings.TrimSpace(stringValue(result["standing"])); value != "" {
		review.Standing = planCandidateStanding(value)
	}
	if value, ok := planningCandidateRefusalValue(result["refusal"]); ok {
		review.Refusal = &value
		if review.Standing == "" {
			review.Standing = value.Standing
		}
	}
	return review, true
}

func planningCandidateRefusalValue(value interface{}) (planCandidateRefusalDetails, bool) {
	switch refusal := value.(type) {
	case planCandidateRefusalDetails:
		return refusal, true
	case *planCandidateRefusalDetails:
		if refusal != nil {
			return *refusal, true
		}
	}
	return planCandidateRefusalDetails{}, false
}

func planningCandidateValue(value interface{}) (colony.PlanCandidate, bool) {
	switch candidate := value.(type) {
	case colony.PlanCandidate:
		return candidate, true
	case *colony.PlanCandidate:
		if candidate != nil {
			return *candidate, true
		}
	}
	return colony.PlanCandidate{}, false
}

func planningIterationCardValue(value interface{}) (colony.PlanningIterationCard, bool) {
	switch card := value.(type) {
	case colony.PlanningIterationCard:
		return card, true
	case *colony.PlanningIterationCard:
		if card != nil {
			return *card, true
		}
	}
	return colony.PlanningIterationCard{}, false
}

func planningPlanRevisionValue(value interface{}) (colony.PlanRevision, bool) {
	switch revision := value.(type) {
	case colony.PlanRevision:
		return revision, true
	case *colony.PlanRevision:
		if revision != nil {
			return *revision, true
		}
	}
	return colony.PlanRevision{}, false
}

func planningAcceptanceReceiptValue(value interface{}) (colony.PlanAcceptanceReceipt, bool) {
	switch receipt := value.(type) {
	case colony.PlanAcceptanceReceipt:
		return receipt, true
	case *colony.PlanAcceptanceReceipt:
		if receipt != nil {
			return *receipt, true
		}
	}
	return colony.PlanAcceptanceReceipt{}, false
}

func planningStageManifestValue(value interface{}) (planningStageManifest, bool) {
	switch manifest := value.(type) {
	case planningStageManifest:
		return manifest, true
	case *planningStageManifest:
		if manifest != nil {
			return *manifest, true
		}
	}
	return planningStageManifest{}, false
}

func planningSpecificationRevisionFromResult(result map[string]interface{}) string {
	if value := strings.TrimSpace(stringValue(result["specification_revision_id"])); value != "" {
		return value
	}
	if manifest, ok := planningStageManifestValue(result["stage_manifest"]); ok {
		return manifest.Specification.RevisionID
	}
	return ""
}

func planningWorkerNameFromResult(result map[string]interface{}, caste planningStageWorkerCaste) string {
	if value := strings.TrimSpace(stringValue(result["worker_name"])); value != "" {
		return value
	}
	if dispatches, ok := result["dispatches"].([]codexPlanningDispatch); ok {
		for _, dispatch := range dispatches {
			if strings.EqualFold(dispatch.Caste, string(caste)) {
				return dispatch.Name
			}
		}
	}
	return ""
}

func renderPlanningRefusalVisual(action, because, state, next string, options planningVisualOptions) string {
	projection := projectPlanningRefusal(action, because, state, next)
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Planning Refusal"))
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString(" (Queen is this project's coordinator)\n")
	builder.WriteString(voiceLine("blocked", projection.Action) + "\n")
	builder.WriteString(voiceLine("avoid", fmt.Sprintf("Because: %s", projection.Because)) + "\n")
	builder.WriteString(voiceLine("status", fmt.Sprintf("State: %s", projection.State)) + "\n")
	builder.WriteString(voiceLine("next", fmt.Sprintf("Next: %s", projection.Next)) + "\n")
	return finalizePlanningVisual(builder.String(), options)
}

func renderPlanningCandidateRefusalVisual(details planCandidateRefusalDetails, options planningVisualOptions) string {
	state := planningEnumLabel(string(details.StateEffect))
	var builder strings.Builder
	builder.WriteString(renderBanner(commandEmoji("plan"), "Plan Candidate Unavailable"))
	builder.WriteString(casteIdentity("queen"))
	builder.WriteString(" (Queen is this project's coordinator)\n")
	builder.WriteString(voiceLine("history", fmt.Sprintf("Candidate: %s", details.CandidateID)) + "\n")
	builder.WriteString(voiceLine("status", fmt.Sprintf("Candidate status: %s", planningEnumLabel(string(details.CandidateStatus)))) + "\n")
	builder.WriteString(voiceLine("status", fmt.Sprintf("Standing: %s", details.Standing)) + "\n")
	builder.WriteString(voiceLine("elapsed", fmt.Sprintf("Expires: %s", details.ExpiresAt.UTC().Format(time.RFC3339Nano))) + "\n")
	builder.WriteString(voiceLine("avoid", fmt.Sprintf("Why unavailable: %s", details.WhyUnavailable)) + "\n")
	renderPlanningValueList(&builder, "evidence", "Evidence", details.Evidence)
	builder.WriteString(voiceLine("status", fmt.Sprintf("State: %s", state)) + "\n")
	builder.WriteString(voiceLine("status", fmt.Sprintf("Active plan: %s", details.ActivePlanEffect)) + "\n")
	builder.WriteString(voiceLine("next", fmt.Sprintf("Next: %s", details.RecoveryCommand)) + "\n")
	return finalizePlanningVisual(builder.String(), options)
}

func planningVisualBandForWidth(width int) planningVisualWidthBand {
	width = planningVisualResolvedWidth(width)
	switch {
	case width < 48:
		return planningVisualBandStacked
	case width < 64:
		return planningVisualBandCompact
	case width < 96:
		return planningVisualBandTable
	default:
		return planningVisualBandWide
	}
}

func planningVisualResolvedWidth(width int) int {
	if width <= 0 {
		width = lifecycleStatusOutputWidth()
	}
	if width < 24 {
		return 24
	}
	return width
}

// finalizePlanningVisual applies the terminal-only guarantees after semantic
// rendering: append-only output, no cursor movement, and bounded lines. SGR
// colour escapes are retained on short lines but never count as columns.
func finalizePlanningVisual(output string, options planningVisualOptions) string {
	width := planningVisualResolvedWidth(options.Width)
	output = planningSanitizeTerminalControls(output)
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	rendered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if planningVisibleWidth(line) <= width || planningVisualLineMayOverflow(line) {
			rendered = append(rendered, line)
			continue
		}
		// A wrapped colour span is ambiguous across terminal implementations.
		// Keep colour on the identity-sized lines and wrap long prose as plain
		// text so layout remains deterministic everywhere.
		line = planningStripANSI(line)
		rendered = append(rendered, planningWrapPlainLine(line, width)...)
	}
	return strings.Join(rendered, "\n") + "\n"
}

func planningSanitizeTerminalControls(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\b", "")
	var builder strings.Builder
	for index := 0; index < len(value); {
		if value[index] != 0x1b {
			builder.WriteByte(value[index])
			index++
			continue
		}
		if index+1 >= len(value) {
			break
		}
		switch value[index+1] {
		case '[':
			end := index + 2
			for end < len(value) && (value[end] < '@' || value[end] > '~') {
				end++
			}
			if end >= len(value) {
				index = len(value)
				continue
			}
			// Only Select Graphic Rendition is presentation; every cursor,
			// erase, or position sequence is deliberately discarded.
			if value[end] == 'm' && shouldUseANSIColors() {
				builder.WriteString(value[index : end+1])
			}
			index = end + 1
		case ']':
			// OSC controls are not needed for planning output. Consume through
			// BEL or ST so terminal title/link controls cannot leak through.
			index += 2
			for index < len(value) {
				if value[index] == '\a' {
					index++
					break
				}
				if value[index] == 0x1b && index+1 < len(value) && value[index+1] == '\\' {
					index += 2
					break
				}
				index++
			}
		default:
			// Drop other two-byte terminal controls.
			index += 2
		}
	}
	return builder.String()
}

func planningVisibleWidth(value string) int {
	return spendDisplayWidth(planningStripANSI(value))
}

func planningStripANSI(value string) string {
	var builder strings.Builder
	for index := 0; index < len(value); {
		if value[index] != 0x1b || index+1 >= len(value) {
			builder.WriteByte(value[index])
			index++
			continue
		}
		if value[index+1] != '[' {
			index += 2
			continue
		}
		index += 2
		for index < len(value) {
			final := value[index] >= '@' && value[index] <= '~'
			index++
			if final {
				break
			}
		}
	}
	return builder.String()
}

func planningWrapPlainLine(line string, width int) []string {
	if line == "" || planningVisibleWidth(line) <= width {
		return []string{line}
	}
	leading := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	continuation := leading + "  "
	if planningVisibleWidth(continuation) >= width {
		continuation = "  "
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}
	result := make([]string, 0, 2)
	current := leading
	for _, word := range words {
		prefix := current
		if strings.TrimSpace(prefix) != "" {
			prefix += " "
		}
		if planningVisibleWidth(prefix+word) <= width {
			current = prefix + word
			continue
		}
		if strings.TrimSpace(current) != "" {
			result = append(result, strings.TrimRight(current, " "))
			current = continuation
		}
		available := width - planningVisibleWidth(current)
		chunks := planningSplitVisibleWord(word, available)
		for chunkIndex, chunk := range chunks {
			if chunkIndex > 0 {
				result = append(result, strings.TrimRight(current, " "))
				current = continuation
			}
			current += chunk
		}
	}
	if strings.TrimSpace(current) != "" {
		result = append(result, strings.TrimRight(current, " "))
	}
	return result
}

func planningSplitVisibleWord(word string, firstWidth int) []string {
	if firstWidth < 1 {
		firstWidth = 1
	}
	chunks := make([]string, 0, 2)
	var current strings.Builder
	currentWidth := 0
	limit := firstWidth
	for _, character := range word {
		characterWidth := planningRuneWidth(character)
		// A zero-width rune (variation selector, ZWJ, combining mark) never
		// starts its own chunk and is never split away from the rune before
		// it -- doing so would cut a single glyph (e.g. "➡️" = U+27A1 +
		// U+FE0F) across two wrapped lines.
		if characterWidth == 0 && current.Len() > 0 {
			current.WriteRune(character)
			continue
		}
		if currentWidth > 0 && currentWidth+characterWidth > limit {
			chunks = append(chunks, current.String())
			current.Reset()
			currentWidth = 0
			limit = firstWidth
		}
		current.WriteRune(character)
		currentWidth += characterWidth
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

func planningRuneWidth(character rune) int {
	if character == 0xFE0F || character == 0xFE0E || character == 0x200D || unicode.Is(unicode.Mn, character) {
		return 0
	}
	if character >= 0x1F000 || (character >= 0x2600 && character <= 0x27BF) {
		return 2
	}
	return 1
}

func planningVisualSafeIdentifier(label, identifier string) string {
	return label + ": " + identifier
}

func planningVisualLineMayOverflow(line string) bool {
	plain := strings.TrimSpace(planningStripANSI(line))
	if !utf8.ValidString(plain) {
		return false
	}
	// A voiceLine-composed content line carries a leading glyph the
	// prefixes below never accounted for ("🔎 Candidate hash: ..." rather
	// than "Candidate hash: ..."). Strip it before matching so an
	// indivisible identifier line is still recognized after this plan
	// routes every planning card renderer through voiceLine -- otherwise
	// every one of these lines would silently lose its overflow exemption
	// and start wrapping mid-hash, a correctness regression this change
	// would otherwise introduce.
	plain = planningStripLeadingVoiceGlyph(plain)
	for _, prefix := range []string{
		"Identifier: sha256:", "Candidate: ", "Candidate hash: ", "SPEC hash: ",
		"Base plan hash: ", "Proposal hash: ", "Expires: ",
		"Why unavailable: ",
		"Acceptance command: aether ", "Next: aether ",
	} {
		if strings.HasPrefix(plain, prefix) {
			return true
		}
	}
	return false
}

// planningStripLeadingVoiceGlyph removes a single leading voice glyph (and
// the space voiceLine always places after it) from an already-trimmed line,
// so a prefix check written against the pre-glyph text still recognizes the
// line's semantic content. A line with no leading glyph passes through
// unchanged.
func planningStripLeadingVoiceGlyph(plain string) string {
	// Resolve each kind through voiceGlyph, not the raw map, so an operator
	// override is stripped too -- reading the map directly made this the one
	// consumer that ignored the override every other call site honours.
	for kind := range voiceGlyphMap {
		if candidate := strings.TrimPrefix(plain, voiceGlyph(kind)+" "); candidate != plain {
			return candidate
		}
	}
	return plain
}

// renderPlanningSpecSection takes lineType as an explicit argument rather
// than deriving it from title -- deriving from the title string would make
// the glyph break the moment a section is renamed, mirroring
// renderSpecCommandVisualSection's identical fix in spec_cmd.go (Phase
// 202.1 plan 03) for the same renamer-fragility failure mode.
func renderPlanningSpecSection(builder *strings.Builder, lineType, title string, items []specCommandVisualItem) {
	builder.WriteString(renderStageMarker(title))
	if len(items) == 0 {
		builder.WriteString(voiceLine(lineType, "none") + "\n")
		return
	}
	for _, item := range items {
		builder.WriteString(voiceLine(lineType, fmt.Sprintf("%s  %s", item.ID, item.Description)) + "\n")
		if item.Detail != "" {
			builder.WriteString(voiceLine(lineType, "  "+item.Detail) + "\n")
		}
	}
}

// renderPlanningValueList takes lineType as an explicit argument for the same
// renamer-fragility reason as renderPlanningSpecSection above.
func renderPlanningValueList(builder *strings.Builder, lineType, label string, values []string) {
	builder.WriteString(voiceLine(lineType, fmt.Sprintf("%s: %s", label, planningValueSummary(values))) + "\n")
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

// planningEnumLabel turns an internal snake_case enum value into the plain
// English words CEC-09 criterion 4 requires at the point a value becomes
// text -- never by filtering the reader. Applied only inside a render
// function, over a value already destined for the rendered string; the
// underlying projection struct fields (e.g. planningGapProjection.Materiality,
// planningCandidateProjection.CandidateStatus) that also serialize into the
// machine-readable planning_projection JSON envelope are left untouched, so
// this never changes what a screen claims to a machine consumer -- only how
// a human reads the same value on the card.
func planningEnumLabel(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "_", " ")
}

func planningDimensionLabel(dimension colony.PlanningDimension) string {
	value := planningEnumLabel(string(dimension))
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
		projection.AuthorityImpact = append(projection.AuthorityImpact, fmt.Sprintf("%s: %s", planningEnumLabel(string(impact.Kind)), impact.Rationale))
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
