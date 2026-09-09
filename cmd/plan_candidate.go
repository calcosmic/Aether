package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planCandidateNow is the one clock seam for a candidate command. Command
// handlers sample it once and pass that value through every authority check.
var planCandidateNow = func() time.Time { return time.Now().UTC() }

type planCandidateOperation string

const (
	planCandidateOperationNone   planCandidateOperation = ""
	planCandidateOperationReview planCandidateOperation = "candidate_review"
	planCandidateOperationDetail planCandidateOperation = "candidate_iteration_detail"
	planCandidateOperationAccept planCandidateOperation = "candidate_accept"
)

// planCandidateAcceptanceRequest names every owner-controlled input required
// to distinguish one candidate frontier from every stale or divergent one.
type planCandidateAcceptanceRequest struct {
	CandidateID               string `json:"candidate_id"`
	SpecificationRevisionID   string `json:"specification_revision_id"`
	SpecificationRevisionHash string `json:"specification_revision_hash"`
	BasePlanRevisionID        string `json:"base_plan_revision_id"`
	TimelineDigest            string `json:"timeline_digest"`
	ProposalHash              string `json:"proposal_hash"`
	AcceptanceToken           string `json:"acceptance_token"`
}

type planCandidateConfidenceScore struct {
	Dimension colony.PlanningDimension `json:"dimension"`
	Score     int                      `json:"score"`
}

type planCandidateStanding string

const (
	planCandidateStandingCurrent  planCandidateStanding = "current"
	planCandidateStandingExpired  planCandidateStanding = "expired"
	planCandidateStandingStale    planCandidateStanding = "stale"
	planCandidateStandingAccepted planCandidateStanding = "accepted"
)

type planCandidateStateEffect string

const (
	planCandidateStateEffectUnchanged     planCandidateStateEffect = "unchanged"
	planCandidateStateEffectMarkedExpired planCandidateStateEffect = "candidate_marked_expired"
)

type planCandidateActivePlanEffect string

const planCandidateActivePlanEffectUnchanged planCandidateActivePlanEffect = "unchanged"

const planCandidateRefreshCommand = "aether plan --refresh"

// planCandidateCurrentAuthority is the current repository frontier against
// which an immutable candidate is assessed. It contains facts, not policy;
// assessPlanCandidateStanding remains pure and performs no I/O.
type planCandidateCurrentAuthority struct {
	SpecificationRevisionID   string
	SpecificationRevisionHash string
	BasePlanRevisionID        string
	BasePlanRevisionHash      string
	ProposalHash              string
	Timeline                  colony.PlanningTimelineBinding
	Stage                     planningStageState
}

type planCandidateStandingAssessment struct {
	Standing            planCandidateStanding         `json:"standing"`
	WhyUnavailable      string                        `json:"why_unavailable,omitempty"`
	Evidence            []string                      `json:"evidence,omitempty"`
	AcceptanceAvailable bool                          `json:"acceptance_available"`
	StateEffect         planCandidateStateEffect      `json:"state_effect"`
	ActivePlanEffect    planCandidateActivePlanEffect `json:"active_plan_effect"`
	RecoveryCommand     string                        `json:"recovery_command,omitempty"`
}

// planCandidateRefusalDetails is the stable machine-readable refusal payload
// returned together with an error. Renderers never need to parse error prose.
type planCandidateRefusalDetails struct {
	CandidateID      string                        `json:"candidate_id"`
	CandidateStatus  colony.PlanCandidateStatus    `json:"candidate_status"`
	Standing         planCandidateStanding         `json:"standing"`
	ExpiresAt        time.Time                     `json:"expires_at"`
	WhyUnavailable   string                        `json:"why_unavailable"`
	Evidence         []string                      `json:"evidence"`
	StateEffect      planCandidateStateEffect      `json:"state_effect"`
	ActivePlanEffect planCandidateActivePlanEffect `json:"active_plan_effect"`
	RecoveryCommand  string                        `json:"recovery_command"`
}

type planCandidateRefusalError struct {
	Details planCandidateRefusalDetails
	Cause   error
}

func (e *planCandidateRefusalError) Error() string {
	if e == nil {
		return "plan candidate is unavailable"
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return fmt.Sprintf("plan candidate %s is %s: %s", e.Details.CandidateID, e.Details.Standing, e.Details.WhyUnavailable)
}

func (e *planCandidateRefusalError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func assessPlanCandidateStanding(candidate colony.PlanCandidate, current planCandidateCurrentAuthority, now time.Time) planCandidateStandingAssessment {
	result := planCandidateStandingAssessment{
		StateEffect:      planCandidateStateEffectUnchanged,
		ActivePlanEffect: planCandidateActivePlanEffectUnchanged,
	}
	refuse := func(standing planCandidateStanding, reason string, evidence ...string) planCandidateStandingAssessment {
		result.Standing = standing
		result.WhyUnavailable = reason
		result.Evidence = append([]string(nil), evidence...)
		result.AcceptanceAvailable = false
		result.RecoveryCommand = planCandidateRefreshCommand
		return result
	}

	createdAt := candidate.CreatedAt.UTC()
	expiresAt := candidate.ExpiresAt.UTC()
	now = now.UTC()
	if createdAt.IsZero() || expiresAt.IsZero() || !expiresAt.After(createdAt) {
		return refuse(planCandidateStandingStale, "candidate_lifetime_invalid",
			"candidate.expires_at must be strictly after candidate.created_at")
	}
	if now.IsZero() || now.Before(createdAt) {
		return refuse(planCandidateStandingStale, "clock_before_candidate_creation",
			fmt.Sprintf("candidate.created_at=%s", createdAt.Format(time.RFC3339Nano)),
			fmt.Sprintf("observed_at=%s", now.Format(time.RFC3339Nano)))
	}

	proposalHash, err := canonicalPlanCandidateProposalHash(candidate.Proposal)
	if err != nil || proposalHash != candidate.ProposalHash || candidate.Proposal.PlanHash != candidate.ProposalHash {
		return refuse(planCandidateStandingStale, "proposal_changed", candidateStandingEvidence("proposal", candidate.ProposalHash, proposalHash, err))
	}
	if err := validatePlanningRecordHashes(candidate); err != nil {
		return refuse(planCandidateStandingStale, "candidate_body_changed", "candidate canonical body: "+err.Error())
	}
	if err := candidate.Validate(); err != nil {
		return refuse(planCandidateStandingStale, "candidate_body_changed", "candidate validation: "+err.Error())
	}

	if candidate.Status == colony.PlanCandidateExpired {
		return refuse(planCandidateStandingExpired, "candidate_expired",
			fmt.Sprintf("candidate.expires_at=%s", expiresAt.Format(time.RFC3339Nano)))
	}
	if candidate.Status == colony.PlanCandidatePendingReview && !now.Before(expiresAt) {
		return refuse(planCandidateStandingExpired, "candidate_expired",
			fmt.Sprintf("candidate.expires_at=%s", expiresAt.Format(time.RFC3339Nano)),
			fmt.Sprintf("observed_at=%s", now.Format(time.RFC3339Nano)))
	}
	if candidate.Status != colony.PlanCandidatePendingReview && candidate.Status != colony.PlanCandidateAccepted {
		return refuse(planCandidateStandingStale, "candidate_status_changed", "candidate.status="+string(candidate.Status))
	}

	if current.SpecificationRevisionID != candidate.SpecificationRevisionID || current.SpecificationRevisionHash != candidate.SpecificationRevisionHash {
		return refuse(planCandidateStandingStale, "specification_changed",
			"candidate.specification="+candidate.SpecificationRevisionID+"@"+candidate.SpecificationRevisionHash,
			"current.specification="+current.SpecificationRevisionID+"@"+current.SpecificationRevisionHash)
	}
	if current.BasePlanRevisionID != candidate.BasePlanRevisionID || current.BasePlanRevisionHash != candidate.BasePlanRevisionHash {
		return refuse(planCandidateStandingStale, "base_plan_changed",
			"candidate.base_plan="+candidate.BasePlanRevisionID+"@"+candidate.BasePlanRevisionHash,
			"current.base_plan="+current.BasePlanRevisionID+"@"+current.BasePlanRevisionHash)
	}
	if current.ProposalHash != candidate.ProposalHash {
		return refuse(planCandidateStandingStale, "proposal_changed",
			"candidate.proposal_hash="+candidate.ProposalHash, "current.proposal_hash="+current.ProposalHash)
	}
	if !reflect.DeepEqual(current.Timeline, candidate.Timeline) {
		return refuse(planCandidateStandingStale, "timeline_changed",
			"candidate.timeline="+candidate.Timeline.ID+"@"+candidate.Timeline.TimelineDigest,
			"current.timeline="+current.Timeline.ID+"@"+current.Timeline.TimelineDigest)
	}

	wantStage := planningStageCandidateReady
	if candidate.Status == colony.PlanCandidateAccepted {
		wantStage = planningStageAccepted
	}
	if current.Stage.Stage != wantStage || current.Stage.RunID != candidate.Timeline.RunID ||
		current.Stage.Specification.RevisionID != candidate.SpecificationRevisionID ||
		current.Stage.Specification.ContentHash != candidate.SpecificationRevisionHash ||
		current.Stage.BasePlanRevisionID != candidate.BasePlanRevisionID ||
		current.Stage.BasePlanRevisionHash != candidate.BasePlanRevisionHash {
		return refuse(planCandidateStandingStale, "planning_stage_changed",
			fmt.Sprintf("candidate.expected_stage=%s", wantStage), fmt.Sprintf("current.stage=%s", current.Stage.Stage))
	}

	if candidate.Status == colony.PlanCandidateAccepted {
		if candidate.Acceptance == nil || candidate.Acceptance.AcceptedAt.Before(createdAt) || !candidate.Acceptance.AcceptedAt.Before(expiresAt) {
			return refuse(planCandidateStandingStale, "acceptance_time_invalid",
				fmt.Sprintf("candidate.created_at=%s", createdAt.Format(time.RFC3339Nano)),
				fmt.Sprintf("candidate.expires_at=%s", expiresAt.Format(time.RFC3339Nano)))
		}
		if err := validatePlanCandidateAcceptanceReceiptHash(*candidate.Acceptance); err != nil {
			return refuse(planCandidateStandingStale, "acceptance_receipt_invalid", "acceptance receipt: "+err.Error())
		}
		result.Standing = planCandidateStandingAccepted
		return result
	}

	result.Standing = planCandidateStandingCurrent
	result.AcceptanceAvailable = true
	return result
}

func candidateStandingEvidence(label, want, got string, err error) string {
	if err != nil {
		return label + ": " + err.Error()
	}
	return fmt.Sprintf("%s expected=%s actual=%s", label, want, got)
}

func planCandidateRefusal(candidate colony.PlanCandidate, assessment planCandidateStandingAssessment) planCandidateRefusalDetails {
	return planCandidateRefusalDetails{
		CandidateID: candidate.ID, CandidateStatus: candidate.Status, Standing: assessment.Standing,
		ExpiresAt: candidate.ExpiresAt.UTC(), WhyUnavailable: assessment.WhyUnavailable,
		Evidence: append([]string(nil), assessment.Evidence...), StateEffect: assessment.StateEffect,
		ActivePlanEffect: assessment.ActivePlanEffect, RecoveryCommand: assessment.RecoveryCommand,
	}
}

type planCandidateReview struct {
	Operation               planCandidateOperation         `json:"operation"`
	Candidate               colony.PlanCandidate           `json:"candidate"`
	TargetConfidence        int                            `json:"target_confidence"`
	ActualConfidence        int                            `json:"actual_confidence"`
	Scores                  []planCandidateConfidenceScore `json:"scores"`
	StopDecision            colony.PlanningStopDecision    `json:"stop_decision"`
	ResidualGaps            []colony.PlanningGap           `json:"residual_gaps"`
	EvidenceThatWouldChange string                         `json:"evidence_that_would_change"`
	SemanticDelta           colony.PlanningSemanticDelta   `json:"semantic_delta"`
	Recommendation          colony.QueenPlanRecommendation `json:"recommendation"`
	Timeline                colony.PlanningTimelineBinding `json:"timeline"`
	Iterations              []colony.PlanningIterationCard `json:"iterations"`
	Acceptance              planCandidateAcceptanceRequest `json:"acceptance"`
	AcceptanceCommand       string                         `json:"acceptance_command"`
	Standing                planCandidateStanding          `json:"standing"`
	Refusal                 *planCandidateRefusalDetails   `json:"refusal,omitempty"`
}

type planCandidateIterationDetail struct {
	Operation      planCandidateOperation       `json:"operation"`
	CandidateID    string                       `json:"candidate_id"`
	TimelineID     string                       `json:"timeline_id"`
	TimelineDigest string                       `json:"timeline_digest"`
	Card           colony.PlanningIterationCard `json:"card"`
}

type planCandidateArtifact struct {
	Candidate colony.PlanCandidate
	Stage     planningStageState
	Header    planningRunHeader
}

type planCandidateCommandInputs struct {
	DeprecatedAccept          bool
	Candidate                 bool
	ShowIteration             int
	ShowIterationSet          bool
	Details                   bool
	AcceptCandidate           string
	SpecificationRevisionID   string
	SpecificationRevisionHash string
	BasePlanRevisionID        string
	TimelineDigest            string
	ProposalHash              string
	AcceptanceToken           string
	ConflictingPlanFlags      []string
}

func resolvePlanCandidateOperation(inputs planCandidateCommandInputs) (planCandidateOperation, error) {
	if inputs.DeprecatedAccept {
		return planCandidateOperationNone, fmt.Errorf("--accept no longer accepts or activates a plan; inspect `aether plan --candidate`, then run the exact `aether plan --accept-candidate <candidate-id> ...` command shown there")
	}

	detailRequested := inputs.ShowIterationSet || inputs.ShowIteration != 0 || inputs.Details
	acceptanceFields := []struct {
		flag  string
		value string
	}{
		{flag: "--accept-candidate", value: inputs.AcceptCandidate},
		{flag: "--spec-revision", value: inputs.SpecificationRevisionID},
		{flag: "--spec-hash", value: inputs.SpecificationRevisionHash},
		{flag: "--base-plan-revision", value: inputs.BasePlanRevisionID},
		{flag: "--timeline-digest", value: inputs.TimelineDigest},
		{flag: "--proposal-hash", value: inputs.ProposalHash},
		{flag: "--acceptance-token", value: inputs.AcceptanceToken},
	}
	acceptanceRequested := false
	for _, field := range acceptanceFields {
		acceptanceRequested = acceptanceRequested || strings.TrimSpace(field.value) != ""
	}

	selected := 0
	if inputs.Candidate {
		selected++
	}
	if detailRequested {
		selected++
	}
	if acceptanceRequested {
		selected++
	}
	if selected > 1 {
		return planCandidateOperationNone, fmt.Errorf("candidate review, iteration detail, and acceptance cannot combine; run exactly one operation")
	}
	if selected > 0 && len(inputs.ConflictingPlanFlags) > 0 {
		return planCandidateOperationNone, fmt.Errorf("candidate operations cannot combine with planning flags %s", strings.Join(inputs.ConflictingPlanFlags, ", "))
	}

	if detailRequested {
		if !inputs.Details {
			return planCandidateOperationNone, fmt.Errorf("--show-iteration requires --details so the immutable card contract is explicit")
		}
		if !inputs.ShowIterationSet && inputs.ShowIteration == 0 {
			return planCandidateOperationNone, fmt.Errorf("--details requires --show-iteration N")
		}
		if inputs.ShowIteration <= 0 {
			return planCandidateOperationNone, fmt.Errorf("--show-iteration must be a positive pass ordinal")
		}
		return planCandidateOperationDetail, nil
	}
	if inputs.Candidate {
		return planCandidateOperationReview, nil
	}
	if acceptanceRequested {
		if strings.TrimSpace(inputs.AcceptCandidate) == "" {
			provided := "exact candidate bindings"
			for _, field := range acceptanceFields[1:] {
				if strings.TrimSpace(field.value) != "" {
					provided = field.flag
					break
				}
			}
			return planCandidateOperationNone, fmt.Errorf("%s requires --accept-candidate <candidate-id>; first run `aether plan --candidate`", provided)
		}
		missing := make([]string, 0)
		for _, field := range acceptanceFields[1:] {
			if strings.TrimSpace(field.value) == "" {
				missing = append(missing, field.flag)
			}
		}
		if len(missing) > 0 {
			return planCandidateOperationNone, fmt.Errorf("--accept-candidate requires %s from the exact command shown by `aether plan --candidate`", strings.Join(missing, ", "))
		}
		return planCandidateOperationAccept, nil
	}
	return planCandidateOperationNone, nil
}

func planCandidateRequestFromInputs(inputs planCandidateCommandInputs) planCandidateAcceptanceRequest {
	return planCandidateAcceptanceRequest{
		CandidateID: strings.TrimSpace(inputs.AcceptCandidate), SpecificationRevisionID: strings.TrimSpace(inputs.SpecificationRevisionID),
		SpecificationRevisionHash: strings.TrimSpace(inputs.SpecificationRevisionHash), BasePlanRevisionID: strings.TrimSpace(inputs.BasePlanRevisionID),
		TimelineDigest: strings.TrimSpace(inputs.TimelineDigest), ProposalHash: strings.TrimSpace(inputs.ProposalHash),
		AcceptanceToken: strings.TrimSpace(inputs.AcceptanceToken),
	}
}

// canonicalPlanCandidateProposalHash is the only proposal preimage used by
// construction, review, acceptance, standalone validation, and build/run
// authority. Candidate back-references are derived after addressing and are
// therefore cleared before hashing to avoid a fixed point.
func canonicalPlanCandidateProposalHash(revision colony.PlanRevision) (string, error) {
	stripped := clonePlanCandidateProposal(revision)
	stripped.CandidateID = ""
	stripped.CandidateContentHash = ""
	for phaseIndex := range stripped.Phases {
		stripped.Phases[phaseIndex].CandidateID = ""
		stripped.Phases[phaseIndex].CandidateContentHash = ""
		for taskIndex := range stripped.Phases[phaseIndex].Tasks {
			stripped.Phases[phaseIndex].Tasks[taskIndex].CandidateID = ""
			stripped.Phases[phaseIndex].Tasks[taskIndex].CandidateContentHash = ""
		}
	}
	return planDefinitionHash(stripped.Phases)
}

// canonicalPlanCandidateContentHash binds the complete immutable review and
// authority payload. Mutable status and acceptance are deliberately omitted;
// their separate receipt transition must not rewrite what the owner reviewed.
func canonicalPlanCandidateContentHash(candidate colony.PlanCandidate) (string, error) {
	proposal := clonePlanCandidateProposal(candidate.Proposal)
	proposal.CandidateID = ""
	proposal.CandidateContentHash = ""
	for phaseIndex := range proposal.Phases {
		proposal.Phases[phaseIndex].CandidateID = ""
		proposal.Phases[phaseIndex].CandidateContentHash = ""
		for taskIndex := range proposal.Phases[phaseIndex].Tasks {
			proposal.Phases[phaseIndex].Tasks[taskIndex].CandidateID = ""
			proposal.Phases[phaseIndex].Tasks[taskIndex].CandidateContentHash = ""
		}
	}
	recommendation := candidate.Recommendation
	recommendation.CandidateID = ""
	normalizeGap := func(gap colony.PlanningGap) colony.PlanningGap {
		gap.EvidenceIDs = uniqueSortedStrings(gap.EvidenceIDs)
		return gap
	}
	normalizeAssessment := func(assessment colony.PlanningDimensionAssessment) colony.PlanningDimensionAssessment {
		assessment.FreshEvidenceIDs = uniqueSortedStrings(assessment.FreshEvidenceIDs)
		assessment.ResolvedGapIDs = uniqueSortedStrings(assessment.ResolvedGapIDs)
		assessment.RemainingGap = normalizeGap(assessment.RemainingGap)
		return assessment
	}
	normalizeChanges := func(values []colony.PlanningSemanticChange) []colony.PlanningSemanticChange {
		result := append([]colony.PlanningSemanticChange(nil), values...)
		for index := range result {
			result[index].EvidenceIDs = uniqueSortedStrings(result[index].EvidenceIDs)
		}
		return result
	}
	delta := candidate.SemanticDelta
	delta.Phases = normalizeChanges(delta.Phases)
	delta.Tasks = normalizeChanges(delta.Tasks)
	delta.Dependencies = normalizeChanges(delta.Dependencies)
	delta.RequirementLinks = normalizeChanges(delta.RequirementLinks)
	delta.AcceptanceChecks = normalizeChanges(delta.AcceptanceChecks)
	delta.NegativeExpectations = normalizeChanges(delta.NegativeExpectations)
	delta.RecoveryExpectations = normalizeChanges(delta.RecoveryExpectations)
	delta.PublicPaths = normalizeChanges(delta.PublicPaths)
	delta.AuthorityImpacts = append([]colony.PlanningAuthorityImpact(nil), delta.AuthorityImpacts...)
	for index := range delta.AuthorityImpacts {
		delta.AuthorityImpacts[index].AffectedSemanticIDs = uniqueSortedStrings(delta.AuthorityImpacts[index].AffectedSemanticIDs)
	}
	assessments := append([]colony.PlanningDimensionAssessment(nil), candidate.DimensionAssessments...)
	for index := range assessments {
		assessments[index] = normalizeAssessment(assessments[index])
	}
	residual := append([]colony.PlanningGap(nil), candidate.ResidualGaps...)
	for index := range residual {
		residual[index] = normalizeGap(residual[index])
	}
	stop := candidate.StopDecision
	stop.EvidenceIDs = uniqueSortedStrings(stop.EvidenceIDs)
	recommendation.EvidenceIDs = uniqueSortedStrings(recommendation.EvidenceIDs)
	payload := struct {
		SchemaVersion             string                               `json:"schema_version"`
		CreatedAt                 time.Time                            `json:"created_at"`
		ExpiresAt                 time.Time                            `json:"expires_at"`
		Proposal                  colony.PlanRevision                  `json:"proposal"`
		ProposalHash              string                               `json:"proposal_hash"`
		BasePlanRevisionID        string                               `json:"base_plan_revision_id"`
		BasePlanRevisionHash      string                               `json:"base_plan_revision_hash"`
		SpecificationRevisionID   string                               `json:"specification_revision_id"`
		SpecificationRevisionHash string                               `json:"specification_revision_hash"`
		Timeline                  colony.PlanningTimelineBinding       `json:"timeline"`
		StopDecision              colony.PlanningStopDecision          `json:"stop_decision"`
		DimensionAssessments      []colony.PlanningDimensionAssessment `json:"dimension_assessments"`
		SemanticDelta             colony.PlanningSemanticDelta         `json:"semantic_delta"`
		ResidualGaps              []colony.PlanningGap                 `json:"residual_gaps"`
		EvidenceThatWouldChange   string                               `json:"evidence_that_would_change"`
		Recommendation            colony.QueenPlanRecommendation       `json:"recommendation"`
	}{
		SchemaVersion: candidate.SchemaVersion, CreatedAt: candidate.CreatedAt, ExpiresAt: candidate.ExpiresAt,
		Proposal: proposal, ProposalHash: candidate.ProposalHash,
		BasePlanRevisionID: candidate.BasePlanRevisionID, BasePlanRevisionHash: candidate.BasePlanRevisionHash,
		SpecificationRevisionID: candidate.SpecificationRevisionID, SpecificationRevisionHash: candidate.SpecificationRevisionHash,
		Timeline: candidate.Timeline, StopDecision: stop, DimensionAssessments: assessments,
		SemanticDelta: delta, ResidualGaps: residual, EvidenceThatWouldChange: candidate.EvidenceThatWouldChange,
		Recommendation: recommendation,
	}
	return jsonSHA256(payload)
}

// addressPlanCandidateReviewPayload derives proposal and recommendation
// identities, then candidate identity, and only then populates all backrefs.
func addressPlanCandidateReviewPayload(candidate *colony.PlanCandidate) error {
	if candidate == nil {
		return fmt.Errorf("plan candidate is required")
	}
	proposalHash, err := canonicalPlanCandidateProposalHash(candidate.Proposal)
	if err != nil {
		return fmt.Errorf("proposal_hash: %w", err)
	}
	candidate.ProposalHash = proposalHash
	candidate.Proposal.PlanHash = proposalHash
	candidate.Proposal.ID = fmt.Sprintf("plan-r%d-%s", candidate.Proposal.Number, proposalHash[:12])
	if err := colony.AddressQueenPlanRecommendation(&candidate.Recommendation); err != nil {
		return fmt.Errorf("recommendation: %w", err)
	}
	hash, err := canonicalPlanCandidateContentHash(*candidate)
	if err != nil {
		return fmt.Errorf("candidate content hash: %w", err)
	}
	candidate.ContentHash = hash
	candidate.ID = "plan-candidate-" + hash[:12]
	bindPlanCandidateBackReferences(candidate)
	return nil
}

func bindPlanCandidateBackReferences(candidate *colony.PlanCandidate) {
	candidate.Proposal.CandidateID = candidate.ID
	candidate.Proposal.CandidateContentHash = candidate.ContentHash
	for phaseIndex := range candidate.Proposal.Phases {
		candidate.Proposal.Phases[phaseIndex].CandidateID = candidate.ID
		candidate.Proposal.Phases[phaseIndex].CandidateContentHash = candidate.ContentHash
		for taskIndex := range candidate.Proposal.Phases[phaseIndex].Tasks {
			candidate.Proposal.Phases[phaseIndex].Tasks[taskIndex].CandidateID = candidate.ID
			candidate.Proposal.Phases[phaseIndex].Tasks[taskIndex].CandidateContentHash = candidate.ContentHash
		}
	}
	candidate.Recommendation.CandidateID = candidate.ID
}

func clonePlanCandidateProposal(revision colony.PlanRevision) colony.PlanRevision {
	clone := revision
	clone.Evidence = append([]string(nil), revision.Evidence...)
	clone.PreservedPhaseIDs = append([]int(nil), revision.PreservedPhaseIDs...)
	clone.SupersededPhaseIDs = append([]int(nil), revision.SupersededPhaseIDs...)
	clone.ReplacementPhaseIDs = append([]int(nil), revision.ReplacementPhaseIDs...)
	clone.RequirementProofLinks = append([]string(nil), revision.RequirementProofLinks...)
	clone.AcceptanceProofLinks = append([]string(nil), revision.AcceptanceProofLinks...)
	clone.NegativeProofLinks = append([]string(nil), revision.NegativeProofLinks...)
	clone.RecoveryProofLinks = append([]string(nil), revision.RecoveryProofLinks...)
	clone.PublicPathProofLinks = append([]string(nil), revision.PublicPathProofLinks...)
	clone.AffectedSemanticIDs = append([]string(nil), revision.AffectedSemanticIDs...)
	clone.PreservedSemanticIDs = append([]string(nil), revision.PreservedSemanticIDs...)
	clone.Phases = clonePhases(revision.Phases)
	return clone
}

func runPlanCandidateCommand(root string, inputs planCandidateCommandInputs) (map[string]interface{}, bool, error) {
	operation, err := resolvePlanCandidateOperation(inputs)
	if err != nil {
		return nil, true, err
	}
	switch operation {
	case planCandidateOperationReview:
		now := planCandidateNow().UTC()
		review, reviewErr := reviewPlanCandidateAt(root, now)
		if reviewErr != nil {
			return nil, true, reviewErr
		}
		return planCandidateReviewResult(review), true, nil
	case planCandidateOperationDetail:
		detail, detailErr := reviewPlanCandidateIteration(root, inputs.ShowIteration)
		if detailErr != nil {
			return nil, true, detailErr
		}
		return planCandidateDetailResult(detail), true, nil
	case planCandidateOperationAccept:
		now := planCandidateNow().UTC()
		accepted, acceptErr := acceptPlanCandidate(root, planCandidateRequestFromInputs(inputs), planCandidateAcceptanceOptions{AcceptedBy: "owner", AcceptedAt: now})
		if acceptErr != nil {
			if accepted.Refusal == nil {
				return nil, true, acceptErr
			}
			return map[string]interface{}{
				"operation": planCandidateOperationAccept, "candidate": accepted.Candidate,
				"refusal": *accepted.Refusal,
			}, true, acceptErr
		}
		return map[string]interface{}{
			"operation": planCandidateOperationAccept, "candidate": accepted.Candidate,
			"revision": accepted.Revision, "acceptance_receipt": accepted.Receipt, "replayed": accepted.Replayed,
		}, true, nil
	default:
		return nil, false, nil
	}
}

func planCandidateAcceptanceToken(candidate colony.PlanCandidate) string {
	material := strings.Join([]string{
		"plan-candidate-acceptance/v1", candidate.ID, candidate.ContentHash,
		candidate.SpecificationRevisionID, candidate.SpecificationRevisionHash,
		candidate.BasePlanRevisionID, candidate.BasePlanRevisionHash,
		candidate.Timeline.ID, candidate.Timeline.TimelineDigest, candidate.ProposalHash,
	}, "\n")
	digest := strings.TrimPrefix(lifecycleDigest([]byte(material)), "sha256:")
	return "accept-plan-" + digest[:24]
}

func reviewPlanCandidate(root string) (planCandidateReview, error) {
	// Preserve the strict internal loader contract used by integrity callers.
	// The public command uses reviewPlanCandidateAt directly so it can render a
	// safe, token-free stale/expired card instead of turning corruption into
	// actionable authority.
	artifact, err := loadPlanCandidateArtifact(root, "")
	if err != nil {
		return planCandidateReview{}, err
	}
	if err := validatePlanningRecordHashes(artifact.Candidate); err != nil {
		return planCandidateReview{}, fmt.Errorf("candidate %s: %w", artifact.Candidate.ID, err)
	}
	if err := artifact.Candidate.Validate(); err != nil {
		return planCandidateReview{}, fmt.Errorf("candidate %s: %w", artifact.Candidate.ID, err)
	}
	return reviewPlanCandidateAt(root, planCandidateNow().UTC())
}

func reviewPlanCandidateAt(root string, now time.Time) (planCandidateReview, error) {
	artifact, err := loadPlanCandidateArtifact(root, "")
	if err != nil {
		return planCandidateReview{}, err
	}
	timeline, err := loadPlanningTimeline(root, artifact.Candidate.Timeline.RunID)
	if err != nil {
		return planCandidateReview{}, err
	}
	if timeline.Binding == nil || timeline.Index == nil {
		return planCandidateReview{}, fmt.Errorf("candidate %s requires a complete indexed timeline", artifact.Candidate.ID)
	}
	authority, err := planCandidateAuthorityFromRepository(root, artifact, *timeline.Binding)
	if err != nil {
		return planCandidateReview{}, err
	}
	standing := assessPlanCandidateStanding(artifact.Candidate, authority, now)

	scores := make([]planCandidateConfidenceScore, 0, len(colony.PlanningDimensions()))
	weighted := planningConfidenceScores{}
	byDimension := make(map[colony.PlanningDimension]int, len(artifact.Candidate.DimensionAssessments))
	for _, assessment := range artifact.Candidate.DimensionAssessments {
		byDimension[assessment.Dimension] = assessment.After
		weighted.Set(assessment.Dimension, assessment.After)
	}
	for _, dimension := range colony.PlanningDimensions() {
		scores = append(scores, planCandidateConfidenceScore{Dimension: dimension, Score: byDimension[dimension]})
	}
	review := planCandidateReview{
		Operation: planCandidateOperationReview, Candidate: artifact.Candidate,
		TargetConfidence: artifact.Header.TargetConfidence, ActualConfidence: weighted.Overall, Scores: scores,
		StopDecision: artifact.Candidate.StopDecision, ResidualGaps: append([]colony.PlanningGap(nil), artifact.Candidate.ResidualGaps...),
		EvidenceThatWouldChange: artifact.Candidate.EvidenceThatWouldChange, SemanticDelta: artifact.Candidate.SemanticDelta,
		Recommendation: artifact.Candidate.Recommendation, Timeline: artifact.Candidate.Timeline,
		Iterations: append([]colony.PlanningIterationCard(nil), timeline.Cards...), Standing: standing.Standing,
	}
	if standing.AcceptanceAvailable {
		request := planCandidateAcceptanceRequest{
			CandidateID: artifact.Candidate.ID, SpecificationRevisionID: artifact.Candidate.SpecificationRevisionID,
			SpecificationRevisionHash: artifact.Candidate.SpecificationRevisionHash, BasePlanRevisionID: artifact.Candidate.BasePlanRevisionID,
			TimelineDigest: artifact.Candidate.Timeline.TimelineDigest, ProposalHash: artifact.Candidate.ProposalHash,
			AcceptanceToken: planCandidateAcceptanceToken(artifact.Candidate),
		}
		review.Acceptance = request
		review.AcceptanceCommand = planCandidateAcceptanceCommand(request)
	} else if standing.Standing != planCandidateStandingAccepted {
		refusal := planCandidateRefusal(artifact.Candidate, standing)
		review.Refusal = &refusal
	}
	return review, nil
}

func planCandidateAuthorityFromRepository(root string, artifact planCandidateArtifact, timeline colony.PlanningTimelineBinding) (planCandidateCurrentAuthority, error) {
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		return planCandidateCurrentAuthority{}, err
	}
	return planCandidateAuthorityFromState(state, artifact, timeline), nil
}

func planCandidateAuthorityFromState(state colony.ColonyState, artifact planCandidateArtifact, timeline colony.PlanningTimelineBinding) planCandidateCurrentAuthority {
	candidate := artifact.Candidate
	authority := planCandidateCurrentAuthority{Timeline: timeline, Stage: artifact.Stage}
	if state.Specification != nil {
		if specification, ok := currentSpecificationRevision(*state.Specification); ok {
			authority.SpecificationRevisionID = specification.ID
			authority.SpecificationRevisionHash = specification.ContentHash
		}
	}
	if candidate.Status == colony.PlanCandidateAccepted {
		authority.BasePlanRevisionID = candidate.BasePlanRevisionID
		authority.BasePlanRevisionHash = candidate.BasePlanRevisionHash
		if active, ok := activePlanRevision(state.Plan); ok {
			authority.ProposalHash = active.PlanHash
		}
	} else {
		if base, _, err := candidateAcceptanceBase(state.Plan, candidate.CreatedAt); err == nil {
			authority.BasePlanRevisionID = base.ID
			authority.BasePlanRevisionHash = base.Hash
		}
		if proposalHash, err := canonicalPlanCandidateProposalHash(candidate.Proposal); err == nil {
			authority.ProposalHash = proposalHash
		}
	}
	return authority
}

func reviewPlanCandidateIteration(root string, iteration int) (planCandidateIterationDetail, error) {
	if iteration <= 0 {
		return planCandidateIterationDetail{}, fmt.Errorf("planning iteration must be positive")
	}
	artifact, err := loadPlanCandidateArtifact(root, "")
	if err != nil {
		return planCandidateIterationDetail{}, err
	}
	timeline, err := verifiedPlanCandidateTimeline(root, artifact.Candidate)
	if err != nil {
		return planCandidateIterationDetail{}, err
	}
	if iteration > len(timeline.Cards) || timeline.Cards[iteration-1].Iteration != iteration {
		return planCandidateIterationDetail{}, fmt.Errorf("candidate %s has no immutable iteration %d (timeline contains passes 1-%d)", artifact.Candidate.ID, iteration, len(timeline.Cards))
	}
	card := timeline.Cards[iteration-1]
	return planCandidateIterationDetail{
		Operation: planCandidateOperationDetail, CandidateID: artifact.Candidate.ID,
		TimelineID: artifact.Candidate.Timeline.ID, TimelineDigest: artifact.Candidate.Timeline.TimelineDigest, Card: card,
	}, nil
}

func loadPlanCandidateArtifact(root, requestedID string) (planCandidateArtifact, error) {
	repositoryRoot, err := canonicalPlanningTimelineRoot(root)
	if err != nil {
		return planCandidateArtifact{}, err
	}
	planningRoot := filepath.Join(repositoryRoot, ".aether", "data", "planning")
	entries, err := os.ReadDir(planningRoot)
	if err != nil {
		return planCandidateArtifact{}, fmt.Errorf("read planning candidates: %w", err)
	}
	wanted := strings.TrimSpace(requestedID)
	matches := make([]planCandidateArtifact, 0, 1)
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return planCandidateArtifact{}, fmt.Errorf("planning run %q must not be a symlink", entry.Name())
		}
		if !entry.IsDir() {
			continue
		}
		runID := entry.Name()
		if err := validatePlanningTimelineSegment("run_id", runID); err != nil {
			return planCandidateArtifact{}, err
		}
		content, exists, readErr := readOptionalPlanningStageFile(repositoryRoot, planningRouteCandidateRepositoryPath(runID))
		if readErr != nil {
			return planCandidateArtifact{}, readErr
		}
		if !exists {
			continue
		}
		var candidate colony.PlanCandidate
		if err := decodePlanningStageJSON(content, &candidate); err != nil {
			return planCandidateArtifact{}, fmt.Errorf("decode candidate for run %q: %w", runID, err)
		}
		if wanted != "" && candidate.ID != wanted {
			continue
		}
		if wanted == "" && candidate.Status != colony.PlanCandidatePendingReview && candidate.Status != colony.PlanCandidateExpired {
			continue
		}
		if candidate.Timeline.RunID != runID {
			return planCandidateArtifact{}, fmt.Errorf("candidate %s path does not match timeline run", candidate.ID)
		}
		stage, err := loadPlanningStageState(repositoryRoot, runID)
		if err != nil {
			return planCandidateArtifact{}, err
		}
		header, err := loadPlanCandidateRunHeader(repositoryRoot, candidate)
		if err != nil {
			return planCandidateArtifact{}, err
		}
		matches = append(matches, planCandidateArtifact{Candidate: candidate, Stage: stage, Header: header})
	}
	if len(matches) == 0 {
		if wanted == "" {
			return planCandidateArtifact{}, fmt.Errorf("no reviewable plan candidate found; finish iterative planning first")
		}
		return planCandidateArtifact{}, fmt.Errorf("plan candidate %q was not found at a reviewable boundary", wanted)
	}
	if len(matches) > 1 {
		sort.Slice(matches, func(i, j int) bool { return matches[i].Candidate.CreatedAt.Before(matches[j].Candidate.CreatedAt) })
		ids := make([]string, len(matches))
		for i := range matches {
			ids[i] = matches[i].Candidate.ID
		}
		return planCandidateArtifact{}, fmt.Errorf("multiple reviewable plan candidates are present (%s); refuse ambiguous review until obsolete runs are resolved", strings.Join(ids, ", "))
	}
	return matches[0], nil
}

func loadPlanCandidateRunHeader(root string, candidate colony.PlanCandidate) (planningRunHeader, error) {
	headerPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", candidate.Timeline.RunID, "run-header.json"))
	content, exists, err := readOptionalPlanningStageFile(root, headerPath)
	if err != nil {
		return planningRunHeader{}, err
	}
	if !exists {
		return planningRunHeader{}, fmt.Errorf("candidate %s planning run header is missing", candidate.ID)
	}
	var header planningRunHeader
	if err := decodePlanningStageJSON(content, &header); err != nil {
		return planningRunHeader{}, fmt.Errorf("decode candidate planning run header: %w", err)
	}
	manifest, err := loadPlanningStageManifest(root, header.RunID, header.StageManifestID)
	if err != nil {
		return planningRunHeader{}, err
	}
	header, err = loadPlanningScoutRunHeader(root, manifest)
	if err != nil {
		return planningRunHeader{}, err
	}
	if header.Specification.RevisionID != candidate.SpecificationRevisionID || header.Specification.ContentHash != candidate.SpecificationRevisionHash {
		return planningRunHeader{}, fmt.Errorf("candidate %s does not match its immutable planning run header", candidate.ID)
	}
	return header, nil
}

func verifiedPlanCandidateTimeline(root string, candidate colony.PlanCandidate) (planningTimeline, error) {
	timeline, err := loadPlanningTimeline(root, candidate.Timeline.RunID)
	if err != nil {
		return planningTimeline{}, err
	}
	if timeline.Binding == nil || timeline.Index == nil {
		return planningTimeline{}, fmt.Errorf("candidate %s requires a complete indexed timeline", candidate.ID)
	}
	wantHash, err := jsonSHA256(candidate.Timeline)
	if err != nil {
		return planningTimeline{}, err
	}
	gotHash, err := jsonSHA256(*timeline.Binding)
	if err != nil {
		return planningTimeline{}, err
	}
	if wantHash != gotHash {
		return planningTimeline{}, fmt.Errorf("candidate %s timeline binding is stale or divergent", candidate.ID)
	}
	if err := validatePlanningTimelineBindingContent(candidate.Timeline, timeline.Cards); err != nil {
		return planningTimeline{}, fmt.Errorf("candidate %s timeline: %w", candidate.ID, err)
	}
	return timeline, nil
}

func planCandidateAcceptanceCommand(request planCandidateAcceptanceRequest) string {
	return strings.Join([]string{
		"aether plan", "--accept-candidate", shellQuotePlanArg(request.CandidateID),
		"--spec-revision", shellQuotePlanArg(request.SpecificationRevisionID),
		"--spec-hash", shellQuotePlanArg(request.SpecificationRevisionHash),
		"--base-plan-revision", shellQuotePlanArg(request.BasePlanRevisionID),
		"--timeline-digest", shellQuotePlanArg(request.TimelineDigest),
		"--proposal-hash", shellQuotePlanArg(request.ProposalHash),
		"--acceptance-token", shellQuotePlanArg(request.AcceptanceToken),
	}, " ")
}

func planCandidateReviewResult(review planCandidateReview) map[string]interface{} {
	confidence := codexPlanConfidence{}
	for _, score := range review.Scores {
		switch score.Dimension {
		case colony.PlanningDimensionKnowledge:
			confidence.Knowledge = planScore(score.Score)
		case colony.PlanningDimensionRequirements:
			confidence.Requirements = planScore(score.Score)
		case colony.PlanningDimensionRisks:
			confidence.Risks = planScore(score.Score)
		case colony.PlanningDimensionDependencies:
			confidence.Dependencies = planScore(score.Score)
		case colony.PlanningDimensionEffort:
			confidence.Effort = planScore(score.Score)
		}
	}
	confidence.Overall = planScore(review.ActualConfidence)
	result := map[string]interface{}{
		"operation": review.Operation, "candidate": review.Candidate, "phases": review.Candidate.Proposal.Phases,
		"target_confidence": review.TargetConfidence, "actual_confidence": review.ActualConfidence, "scores": review.Scores,
		"confidence": confidence, "stop_decision": review.StopDecision, "residual_gaps": review.ResidualGaps,
		"evidence_that_would_change": review.EvidenceThatWouldChange, "semantic_delta": review.SemanticDelta,
		"recommendation": review.Recommendation, "timeline": review.Timeline, "iterations": review.Iterations,
		"standing": review.Standing,
	}
	if review.AcceptanceCommand != "" {
		result["acceptance"] = review.Acceptance
		result["acceptance_command"] = review.AcceptanceCommand
	}
	if review.Refusal != nil {
		result["refusal"] = *review.Refusal
	}
	return result
}

func planCandidateDetailResult(detail planCandidateIterationDetail) map[string]interface{} {
	return map[string]interface{}{
		"operation": detail.Operation, "candidate_id": detail.CandidateID,
		"timeline_id": detail.TimelineID, "timeline_digest": detail.TimelineDigest, "iteration": detail.Card,
	}
}
