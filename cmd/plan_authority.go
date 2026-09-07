package cmd

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planAuthorityClassification names the only two plan lineages that may
// execute. A current plan has exact owner acceptance; a legacy plan is usable
// only because migration explicitly classified its older, unbound authority.
type planAuthorityClassification string

const (
	planAuthorityCurrentAccepted planAuthorityClassification = "current_accepted"
	planAuthorityLegacyUnbound   planAuthorityClassification = "legacy_unbound"
)

// planAuthorityRefusalCode is stable machine-readable recovery data shared by
// build and run. Renderers may explain these values but must not infer them
// from error prose.
type planAuthorityRefusalCode string

const (
	planAuthorityRefusalStateUnavailable         planAuthorityRefusalCode = "plan_authority_state_unavailable"
	planAuthorityRefusalMissingPolicy            planAuthorityRefusalCode = "plan_authority_policy_missing"
	planAuthorityRefusalNoActivePlan             planAuthorityRefusalCode = "plan_authority_no_active_plan"
	planAuthorityRefusalLegacyInvalid            planAuthorityRefusalCode = "plan_authority_legacy_invalid"
	planAuthorityRefusalSpecificationNotApproved planAuthorityRefusalCode = "plan_authority_specification_not_approved"
	planAuthorityRefusalStaleSpecification       planAuthorityRefusalCode = "plan_authority_stale_specification"
	planAuthorityRefusalCandidateNotAccepted     planAuthorityRefusalCode = "plan_authority_candidate_not_accepted"
	planAuthorityRefusalCandidateInvalid         planAuthorityRefusalCode = "plan_authority_candidate_invalid"
	planAuthorityRefusalStaleBase                planAuthorityRefusalCode = "plan_authority_stale_base"
	planAuthorityRefusalBrokenTimeline           planAuthorityRefusalCode = "plan_authority_broken_timeline"
	planAuthorityRefusalAcceptanceInvalid        planAuthorityRefusalCode = "plan_authority_acceptance_invalid"
	planAuthorityRefusalAffectedScope            planAuthorityRefusalCode = "plan_authority_affected_scope"
)

// planAuthorityBinding is a compact ID plus immutable hash attribution. Hash
// means content hash for revisions/specs/candidates/receipts and timeline
// digest for timelines.
type planAuthorityBinding struct {
	ID   string `json:"id,omitempty"`
	Hash string `json:"hash,omitempty"`
}

// planAuthorityDecision is the complete, side-effect-free result consumed by
// both execution surfaces.
type planAuthorityDecision struct {
	Eligible            bool                        `json:"eligible"`
	Classification      planAuthorityClassification `json:"classification,omitempty"`
	ActiveRevision      planAuthorityBinding        `json:"active_revision,omitempty"`
	Specification       planAuthorityBinding        `json:"specification,omitempty"`
	Candidate           planAuthorityBinding        `json:"candidate,omitempty"`
	Timeline            planAuthorityBinding        `json:"timeline,omitempty"`
	Acceptance          planAuthorityBinding        `json:"acceptance,omitempty"`
	RefusalCode         planAuthorityRefusalCode    `json:"refusal_code,omitempty"`
	AffectedSemanticIDs []string                    `json:"affected_semantic_ids,omitempty"`
	RecoveryCommand     string                      `json:"recovery_command,omitempty"`
	Diagnostic          string                      `json:"diagnostic,omitempty"`
}

// planAuthorityVerifiedBindings carries artifact bytes already verified by
// repository loaders. Error fields preserve which verification boundary
// failed without making the pure policy perform I/O.
type planAuthorityVerifiedBindings struct {
	Candidate       *colony.PlanCandidate
	Acceptance      *colony.PlanAcceptanceReceipt
	Timeline        *colony.PlanningTimelineBinding
	Cards           []colony.PlanningIterationCard
	CandidateError  string
	TimelineError   string
	AcceptanceError string
}

// validateAcceptedPlanAuthority is the single pure execution-authority policy.
// It reads immutable lifecycle facts and verified artifacts, never state stores.
func validateAcceptedPlanAuthority(facts LifecycleFacts, bindings planAuthorityVerifiedBindings) planAuthorityDecision {
	decision := planAuthorityDecision{}
	if len(facts.Planning.Value.AffectedUnresolvedSemanticIDs) > 0 {
		decision.AffectedSemanticIDs = uniqueSortedStrings(facts.Planning.Value.AffectedUnresolvedSemanticIDs)
	}
	if facts.State.Source.Provenance != LifecycleFactConfirmed {
		return refusePlanAuthority(decision, planAuthorityRefusalStateUnavailable, "aether status", emptyFallback(strings.TrimSpace(facts.State.Source.Diagnostic), "authoritative colony state is unavailable"))
	}

	state := facts.State.Value
	plan := state.Plan
	if len(plan.Phases) == 0 {
		return refusePlanAuthority(decision, planAuthorityRefusalNoActivePlan, "aether plan", "no active plan phases are present")
	}

	switch plan.AcceptancePolicy {
	case colony.PlanAcceptanceLegacyUnbound:
		return validateLegacyPlanAuthority(state, decision)
	case colony.PlanAcceptanceExplicitOwner:
		return validateCurrentPlanAuthority(state, facts.Planning.Value, bindings, decision)
	case "":
		return refusePlanAuthority(decision, planAuthorityRefusalMissingPolicy, "aether plan", "the plan has no explicit acceptance or migration policy")
	default:
		return refusePlanAuthority(decision, planAuthorityRefusalMissingPolicy, "aether plan", fmt.Sprintf("unsupported plan acceptance policy %q", plan.AcceptancePolicy))
	}
}

func validateLegacyPlanAuthority(state colony.ColonyState, decision planAuthorityDecision) planAuthorityDecision {
	plan := state.Plan
	if state.Specification != nil || planHasCurrentAuthority(plan) {
		return refusePlanAuthority(decision, planAuthorityRefusalLegacyInvalid, "aether plan", "legacy_unbound authority contains current specification, candidate, or binding data")
	}
	if !orderedPlanAuthorityPhases(plan.Phases) || firstBuildablePhase(plan.Phases) == 0 {
		return refusePlanAuthority(decision, planAuthorityRefusalNoActivePlan, "aether plan", "legacy_unbound authority has no valid active phase")
	}
	hash, err := planDefinitionHash(plan.Phases)
	if err != nil {
		return refusePlanAuthority(decision, planAuthorityRefusalLegacyInvalid, "aether plan", fmt.Sprintf("hash legacy plan: %v", err))
	}
	decision.Eligible = true
	decision.Classification = planAuthorityLegacyUnbound
	decision.ActiveRevision = planAuthorityBinding{ID: activePlanRevisionID(plan), Hash: hash}
	return decision
}

func validateCurrentPlanAuthority(state colony.ColonyState, planning LifecyclePlanningFacts, bindings planAuthorityVerifiedBindings, decision planAuthorityDecision) planAuthorityDecision {
	if state.Specification == nil {
		return refusePlanAuthority(decision, planAuthorityRefusalSpecificationNotApproved, "aether spec", "current plan authority requires an approved specification")
	}
	currentSpec, ok := currentSpecificationRevision(*state.Specification)
	if !ok || currentSpec.Status != colony.SpecStatusApproved || currentSpec.Approval == nil {
		return refusePlanAuthority(decision, planAuthorityRefusalSpecificationNotApproved, "aether spec", "the current specification revision is not explicitly approved")
	}
	if err := currentSpec.Approval.Validate(); err != nil ||
		currentSpec.Approval.SpecificationID != state.Specification.ID ||
		currentSpec.Approval.RevisionID != currentSpec.ID ||
		currentSpec.Approval.RevisionContentHash != currentSpec.ContentHash {
		return refusePlanAuthority(decision, planAuthorityRefusalSpecificationNotApproved, "aether spec", "the current specification approval receipt does not bind the exact revision")
	}
	decision.Specification = planAuthorityBinding{ID: currentSpec.ID, Hash: currentSpec.ContentHash}

	active, ok := activePlanRevision(state.Plan)
	if !ok {
		return refusePlanAuthority(decision, planAuthorityRefusalNoActivePlan, "aether plan", "active_revision_id does not name a retained immutable revision")
	}
	decision.ActiveRevision = planAuthorityBinding{ID: active.ID, Hash: active.PlanHash}
	if len(decision.AffectedSemanticIDs) > 0 || planning.AcceptanceBindingStatus == LifecyclePlanBindingAffected {
		return refusePlanAuthority(decision, planAuthorityRefusalAffectedScope, "aether plan", "the accepted plan has specification-affected scope that is not reconciled")
	}
	if active.SpecificationRevisionID != currentSpec.ID || active.SpecificationRevisionHash != currentSpec.ContentHash {
		return refusePlanAuthority(decision, planAuthorityRefusalStaleSpecification, "aether plan", "the active revision does not bind the current approved specification")
	}

	retained, ok := planAuthorityCandidateByID(state.Plan.Candidates, active.CandidateID)
	if !ok || retained.Status != colony.PlanCandidateAccepted || retained.Acceptance == nil {
		return refusePlanAuthority(decision, planAuthorityRefusalCandidateNotAccepted, "aether plan --candidate", "the active revision has no explicitly accepted candidate")
	}
	decision.Candidate = planAuthorityBinding{ID: retained.ID, Hash: retained.ContentHash}
	if bindings.Candidate == nil {
		detail := emptyFallback(strings.TrimSpace(bindings.CandidateError), "the accepted candidate artifact was not verified")
		return refusePlanAuthority(decision, planAuthorityRefusalCandidateInvalid, "aether plan --candidate", detail)
	}
	candidate := *bindings.Candidate
	if candidate.Status != colony.PlanCandidateAccepted || candidate.Acceptance == nil {
		return refusePlanAuthority(decision, planAuthorityRefusalCandidateNotAccepted, "aether plan --candidate", "candidate status is not accepted")
	}
	if !reflect.DeepEqual(retained, candidate) {
		return refusePlanAuthority(decision, planAuthorityRefusalCandidateInvalid, "aether plan --candidate", "the verified candidate artifact diverges from retained state")
	}
	if err := validatePlanningRecordHashes(candidate); err != nil {
		return refusePlanAuthority(decision, planAuthorityRefusalCandidateInvalid, "aether plan --candidate", fmt.Sprintf("candidate binding: %v", err))
	}
	if err := candidate.Validate(); err != nil {
		return refusePlanAuthority(decision, planAuthorityRefusalCandidateInvalid, "aether plan --candidate", fmt.Sprintf("candidate: %v", err))
	}
	if candidate.SpecificationRevisionID != currentSpec.ID || candidate.SpecificationRevisionHash != currentSpec.ContentHash ||
		candidate.Proposal.ID != active.ID || candidate.ProposalHash != active.PlanHash ||
		active.CandidateContentHash != candidate.ContentHash {
		return refusePlanAuthority(decision, planAuthorityRefusalStaleSpecification, "aether plan", "candidate, active revision, and specification bindings are not exact")
	}
	if err := validateStandalonePlanRevision(active); err != nil || !reflect.DeepEqual(active.Phases, state.Plan.Phases) {
		return refusePlanAuthority(decision, planAuthorityRefusalCandidateInvalid, "aether plan", "the active plan does not match the accepted immutable proposal")
	}

	base, ok := planAuthorityRevisionByID(state.Plan.Revisions, candidate.BasePlanRevisionID)
	if !ok || base.PlanHash != candidate.BasePlanRevisionHash || active.ParentID != base.ID {
		return refusePlanAuthority(decision, planAuthorityRefusalStaleBase, "aether plan", "the accepted candidate does not bind the active revision's exact base")
	}

	decision.Timeline = planAuthorityBinding{ID: candidate.Timeline.ID, Hash: candidate.Timeline.TimelineDigest}
	if bindings.Timeline == nil {
		detail := emptyFallback(strings.TrimSpace(bindings.TimelineError), "the accepted candidate timeline was not verified")
		return refusePlanAuthority(decision, planAuthorityRefusalBrokenTimeline, "aether plan", detail)
	}
	if !reflect.DeepEqual(*bindings.Timeline, candidate.Timeline) ||
		candidate.Timeline.ID != active.PlanningTimelineID || candidate.Timeline.TimelineDigest != active.PlanningTimelineDigest {
		return refusePlanAuthority(decision, planAuthorityRefusalBrokenTimeline, "aether plan", "the verified timeline binding diverges from candidate or active revision")
	}
	if err := validatePlanningTimelineBinding(*bindings.Timeline, bindings.Cards); err != nil {
		return refusePlanAuthority(decision, planAuthorityRefusalBrokenTimeline, "aether plan", fmt.Sprintf("timeline: %v", err))
	}

	receipt := candidate.Acceptance
	decision.Acceptance = planAuthorityBinding{ID: receipt.ID, Hash: receipt.ContentHash}
	if bindings.Acceptance == nil {
		detail := emptyFallback(strings.TrimSpace(bindings.AcceptanceError), "the acceptance receipt artifact was not verified")
		return refusePlanAuthority(decision, planAuthorityRefusalAcceptanceInvalid, "aether plan --candidate", detail)
	}
	if !reflect.DeepEqual(*bindings.Acceptance, *receipt) || !reflect.DeepEqual(*retained.Acceptance, *receipt) {
		return refusePlanAuthority(decision, planAuthorityRefusalAcceptanceInvalid, "aether plan --candidate", "the acceptance receipt artifact diverges from candidate or retained state")
	}
	if err := receipt.Validate(); err != nil {
		return refusePlanAuthority(decision, planAuthorityRefusalAcceptanceInvalid, "aether plan --candidate", fmt.Sprintf("acceptance receipt: %v", err))
	}
	if err := validatePlanCandidateAcceptanceReceiptHash(*receipt); err != nil {
		return refusePlanAuthority(decision, planAuthorityRefusalAcceptanceInvalid, "aether plan --candidate", err.Error())
	}
	expectedTokenHash := strings.TrimPrefix(lifecycleDigest([]byte(planCandidateAcceptanceToken(candidate))), "sha256:")
	if receipt.CandidateID != candidate.ID || receipt.CandidateContentHash != candidate.ContentHash ||
		receipt.SpecificationRevisionID != currentSpec.ID || receipt.SpecificationRevisionHash != currentSpec.ContentHash ||
		receipt.BasePlanRevisionID != base.ID || receipt.BasePlanRevisionHash != base.PlanHash ||
		receipt.TimelineID != candidate.Timeline.ID || receipt.TimelineDigest != candidate.Timeline.TimelineDigest ||
		receipt.ProposalHash != active.PlanHash || receipt.AcceptanceTokenHash != expectedTokenHash ||
		receipt.ActivatedPlanRevisionID != active.ID || receipt.ActivatedPlanRevisionHash != active.PlanHash {
		return refusePlanAuthority(decision, planAuthorityRefusalAcceptanceInvalid, "aether plan --candidate", "the acceptance receipt does not bind every exact authority field")
	}
	if err := validateCurrentPlanningState(state); err != nil {
		return refusePlanAuthority(decision, planAuthorityRefusalCandidateInvalid, "aether plan", fmt.Sprintf("current plan authority: %v", err))
	}

	decision.Eligible = true
	decision.Classification = planAuthorityCurrentAccepted
	return decision
}

// loadPlanAuthorityVerifiedBindings performs the read-only artifact checks
// needed before the pure validator is called by an execution entry point.
func loadPlanAuthorityVerifiedBindings(root string, facts LifecycleFacts) planAuthorityVerifiedBindings {
	state := facts.State.Value
	active, ok := activePlanRevision(state.Plan)
	if !ok || strings.TrimSpace(active.CandidateID) == "" {
		return planAuthorityVerifiedBindings{}
	}
	retained, ok := planAuthorityCandidateByID(state.Plan.Candidates, active.CandidateID)
	if !ok {
		return planAuthorityVerifiedBindings{}
	}
	if retained.Status != colony.PlanCandidateAccepted || retained.Acceptance == nil {
		candidate := retained
		return planAuthorityVerifiedBindings{Candidate: &candidate}
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return planAuthorityVerifiedBindings{CandidateError: "repository root is unavailable for accepted candidate verification"}
	}

	artifact, err := loadPlanCandidateArtifact(root, active.CandidateID)
	if err != nil {
		return planAuthorityVerifiedBindings{CandidateError: err.Error()}
	}
	bindings := planAuthorityVerifiedBindings{Candidate: &artifact.Candidate}
	timeline, err := verifiedPlanCandidateTimeline(root, artifact.Candidate)
	if err != nil {
		bindings.TimelineError = err.Error()
		return bindings
	}
	bindings.Timeline = timeline.Binding
	bindings.Cards = append([]colony.PlanningIterationCard(nil), timeline.Cards...)

	repositoryRoot, err := canonicalPlanningTimelineRoot(root)
	if err != nil {
		bindings.AcceptanceError = err.Error()
		return bindings
	}
	content, exists, err := readOptionalPlanningStageFile(repositoryRoot, planningRouteAcceptanceRepositoryPath(artifact.Candidate.Timeline.RunID))
	if err != nil {
		bindings.AcceptanceError = err.Error()
		return bindings
	}
	if !exists {
		bindings.AcceptanceError = "accepted plan receipt artifact is missing"
		return bindings
	}
	var receipt colony.PlanAcceptanceReceipt
	if err := decodePlanningStageJSON(content, &receipt); err != nil {
		bindings.AcceptanceError = fmt.Sprintf("decode accepted plan receipt: %v", err)
		return bindings
	}
	bindings.Acceptance = &receipt
	return bindings
}

func refusePlanAuthority(decision planAuthorityDecision, code planAuthorityRefusalCode, recovery, diagnostic string) planAuthorityDecision {
	decision.Eligible = false
	decision.Classification = ""
	decision.RefusalCode = code
	decision.RecoveryCommand = recovery
	decision.Diagnostic = strings.TrimSpace(diagnostic)
	return decision
}

func planAuthorityCandidateByID(candidates []colony.PlanCandidate, id string) (colony.PlanCandidate, bool) {
	for i := range candidates {
		if candidates[i].ID == id {
			return candidates[i], true
		}
	}
	return colony.PlanCandidate{}, false
}

func planAuthorityRevisionByID(revisions []colony.PlanRevision, id string) (colony.PlanRevision, bool) {
	for i := range revisions {
		if revisions[i].ID == id {
			return revisions[i], true
		}
	}
	return colony.PlanRevision{}, false
}

func orderedPlanAuthorityPhases(phases []colony.Phase) bool {
	previous := 0
	for _, phase := range phases {
		if phase.ID <= previous {
			return false
		}
		previous = phase.ID
	}
	return len(phases) > 0
}
