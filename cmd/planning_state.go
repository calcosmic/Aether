package cmd

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

var planningSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// normalizePlanningState is the read-only structural normalization boundary for
// planning authority. It deliberately does not classify legacy state; migration
// owns that one additive mutation so validation can never manufacture consent.
func normalizePlanningState(state colony.ColonyState) (colony.ColonyState, error) {
	if err := validatePlanningState(state); err != nil {
		return colony.ColonyState{}, err
	}
	return state, nil
}

// validatePlanningState validates facts already present in ColonyState. Absent
// Phase-200 fields are a supported legacy shape; once current authority fields
// appear, the complete specification/candidate/acceptance chain is required.
func validatePlanningState(state colony.ColonyState) error {
	if state.Specification != nil {
		if err := validateSpecificationState(*state.Specification); err != nil {
			return fmt.Errorf("specification: %w", err)
		}
	}

	switch state.Plan.AcceptancePolicy {
	case "":
		if planHasCurrentAuthority(state.Plan) {
			return fmt.Errorf("plan acceptance_policy is required when current planning authority is populated")
		}
		return nil
	case colony.PlanAcceptanceLegacyUnbound:
		if state.Plan.PendingCandidateID != "" || len(state.Plan.Candidates) != 0 || planHasCurrentBindings(state.Plan) {
			return fmt.Errorf("legacy_unbound plan cannot contain current candidate or acceptance bindings")
		}
		return nil
	case colony.PlanAcceptanceExplicitOwner:
		return validateCurrentPlanningState(state)
	default:
		return fmt.Errorf("plan acceptance_policy %q is unsupported", state.Plan.AcceptancePolicy)
	}
}

func validateSpecificationState(specification colony.Specification) error {
	if err := validatePlanningSchemaVersion("schema_version", specification.SchemaVersion, colony.SpecificationSchemaVersion); err != nil {
		return err
	}
	for i := range specification.Revisions {
		revision := specification.Revisions[i]
		if err := validatePlanningSchemaVersion(fmt.Sprintf("revisions[%d].schema_version", i), revision.SchemaVersion, colony.SpecificationSchemaVersion); err != nil {
			return err
		}
		if err := validateSHA256(fmt.Sprintf("revisions[%d].content_hash", i), revision.ContentHash); err != nil {
			return err
		}
		if err := validateContentAddressedID(fmt.Sprintf("revisions[%d].id", i), revision.ID, revision.ContentHash); err != nil {
			return err
		}
		if i == 0 && revision.PredecessorID != "" {
			return fmt.Errorf("revisions[0].predecessor_id must be empty")
		}
		if i > 0 && revision.PredecessorID != specification.Revisions[i-1].ID {
			return fmt.Errorf("revisions[%d].predecessor_id must name the immediately preceding immutable revision", i)
		}
		if err := validateSpecificationRevisionHashes(i, revision); err != nil {
			return err
		}
	}
	if err := specification.Validate(); err != nil {
		return err
	}
	// Goal-addressed lineages are the current production format. Recompute
	// those values here while retaining structural compatibility for old
	// in-memory fixtures; persisted state is always checked strictly at load.
	canonicalID, err := colony.CanonicalSpecificationID(specification.GoalID)
	if err == nil && specification.ID == canonicalID {
		if err := specification.ValidateCanonical(); err != nil {
			return err
		}
	}
	return nil
}

// validateCanonicalSpecificationState treats every persisted identity as an
// untrusted claim and recomputes it from the typed owner contract. Structural
// validation remains separate for explicitly legacy in-memory fixtures.
func validateCanonicalSpecificationState(specification colony.Specification) error {
	if err := validateSpecificationState(specification); err != nil {
		return err
	}
	if err := specification.ValidateCanonical(); err != nil {
		return err
	}
	return nil
}

func validateSpecificationRevisionHashes(index int, revision colony.SpecRevision) error {
	sections := []struct {
		name   string
		hashes []string
	}{
		{name: "outcomes", hashes: specOutcomeHashes(revision.Outcomes)},
		{name: "included_behaviors", hashes: specIncludedBehaviorHashes(revision.IncludedBehaviors)},
		{name: "exclusions", hashes: specExclusionHashes(revision.Exclusions)},
		{name: "binding_decisions", hashes: specBindingDecisionHashes(revision.BindingDecisions)},
		{name: "requirements", hashes: specRequirementHashes(revision.Requirements)},
		{name: "acceptance_checks", hashes: specAcceptanceCheckHashes(revision.AcceptanceChecks)},
		{name: "negative_expectations", hashes: specNegativeExpectationHashes(revision.NegativeExpectations)},
		{name: "recovery_expectations", hashes: specRecoveryExpectationHashes(revision.RecoveryExpectations)},
		{name: "affected_public_paths", hashes: specPublicPathHashes(revision.AffectedPublicPaths)},
	}
	for _, section := range sections {
		for itemIndex, hash := range section.hashes {
			if err := validateSHA256(fmt.Sprintf("revisions[%d].%s[%d].content_hash", index, section.name, itemIndex), hash); err != nil {
				return err
			}
		}
	}
	if revision.Approval != nil {
		if err := validatePlanningSchemaVersion(fmt.Sprintf("revisions[%d].approval.schema_version", index), revision.Approval.SchemaVersion, colony.SpecificationSchemaVersion); err != nil {
			return err
		}
		if err := validateSHA256(fmt.Sprintf("revisions[%d].approval.revision_content_hash", index), revision.Approval.RevisionContentHash); err != nil {
			return err
		}
		if err := validateSHA256(fmt.Sprintf("revisions[%d].approval.approval_token_hash", index), revision.Approval.ApprovalTokenHash); err != nil {
			return err
		}
	}
	return nil
}

func validateCurrentPlanningState(state colony.ColonyState) error {
	if state.Specification == nil {
		return fmt.Errorf("explicit_owner plan requires a specification")
	}
	currentSpec, ok := currentSpecificationRevision(*state.Specification)
	if !ok {
		return fmt.Errorf("explicit_owner plan requires a current specification revision")
	}
	if len(state.Plan.Revisions) == 0 {
		return fmt.Errorf("explicit_owner plan requires immutable plan revisions")
	}
	if len(state.Plan.Candidates) == 0 {
		return fmt.Errorf("explicit_owner plan requires an accepted candidate")
	}

	revisions, err := validatePlanRevisionChain(state.Plan.Revisions, true)
	if err != nil {
		return err
	}
	active, ok := revisions[state.Plan.ActiveRevisionID]
	if !ok {
		return fmt.Errorf("active_revision_id %q does not name a plan revision", state.Plan.ActiveRevisionID)
	}
	if !reflect.DeepEqual(state.Plan.Phases, active.Phases) {
		return fmt.Errorf("active plan phases do not match active revision %q", active.ID)
	}
	boundSpec, ok := specificationRevisionByID(*state.Specification, active.SpecificationRevisionID)
	if !ok || boundSpec.ContentHash != active.SpecificationRevisionHash || boundSpec.Approval == nil ||
		(boundSpec.Status != colony.SpecStatusApproved && boundSpec.Status != colony.SpecStatusSuperseded) {
		return fmt.Errorf("active revision specification binding is not an exact historically approved revision")
	}
	if active.SpecificationRevisionID == currentSpec.ID && (currentSpec.Status != colony.SpecStatusApproved || currentSpec.Approval == nil) {
		return fmt.Errorf("explicit_owner plan requires the current approved specification revision")
	}
	if err := validateCurrentPlanRevisionBindings(active, boundSpec); err != nil {
		return fmt.Errorf("active revision: %w", err)
	}
	if err := validateCurrentPlanNodes(state.Plan.Phases, active, boundSpec); err != nil {
		return err
	}
	if active.SpecificationRevisionID != currentSpec.ID || active.SpecificationRevisionHash != currentSpec.ContentHash {
		if _, unresolved, err := unresolvedPlanImpact(state); err != nil {
			return fmt.Errorf("active revision specification successor: %w", err)
		} else if !unresolved {
			return fmt.Errorf("active revision specification successor has no affected scope to reconcile")
		}
	}

	candidates := make(map[string]colony.PlanCandidate, len(state.Plan.Candidates))
	for i := range state.Plan.Candidates {
		candidate := state.Plan.Candidates[i]
		if _, duplicate := candidates[candidate.ID]; duplicate {
			return fmt.Errorf("candidates[%d].id %q is duplicated", i, candidate.ID)
		}
		if err := validatePlanCandidateState(candidate, revisions, *state.Specification); err != nil {
			return fmt.Errorf("candidates[%d]: %w", i, err)
		}
		candidates[candidate.ID] = candidate
	}

	if pendingID := strings.TrimSpace(state.Plan.PendingCandidateID); pendingID != "" {
		pending, ok := candidates[pendingID]
		if !ok {
			return fmt.Errorf("pending_candidate_id %q does not name a candidate", pendingID)
		}
		if pending.Status != colony.PlanCandidatePendingReview {
			return fmt.Errorf("pending_candidate_id %q names candidate with status %q", pendingID, pending.Status)
		}
	}

	activeCandidate, ok := candidates[active.CandidateID]
	if !ok {
		return fmt.Errorf("active revision candidate_id %q does not name a candidate", active.CandidateID)
	}
	if activeCandidate.Status != colony.PlanCandidateAccepted || activeCandidate.Acceptance == nil {
		return fmt.Errorf("active revision candidate_id %q is not explicitly accepted", active.CandidateID)
	}
	if activeCandidate.ContentHash != active.CandidateContentHash {
		return fmt.Errorf("active revision candidate_content_hash does not match candidate")
	}
	if activeCandidate.SpecificationRevisionID != active.SpecificationRevisionID || activeCandidate.SpecificationRevisionHash != active.SpecificationRevisionHash {
		return fmt.Errorf("active candidate specification binding does not match active revision")
	}
	if activeCandidate.Timeline.ID != active.PlanningTimelineID || activeCandidate.Timeline.TimelineDigest != active.PlanningTimelineDigest {
		return fmt.Errorf("active candidate timeline binding does not match active revision")
	}
	if activeCandidate.Proposal.ID != active.ID || activeCandidate.ProposalHash != active.PlanHash {
		return fmt.Errorf("active candidate proposal does not match active revision")
	}
	if activeCandidate.Acceptance.ActivatedPlanRevisionID != active.ID || activeCandidate.Acceptance.ActivatedPlanRevisionHash != active.PlanHash {
		return fmt.Errorf("accepted candidate does not activate the active revision")
	}
	return nil
}

func validatePlanRevisionChain(values []colony.PlanRevision, current bool) (map[string]colony.PlanRevision, error) {
	byID := make(map[string]colony.PlanRevision, len(values))
	for i := range values {
		revision := values[i]
		if revision.SchemaVersion > planRevisionSchemaVersion {
			return nil, fmt.Errorf("revisions[%d].schema_version uses unsupported future plan revision major %d", i, revision.SchemaVersion)
		}
		if current && revision.SchemaVersion != planRevisionSchemaVersion {
			return nil, fmt.Errorf("revisions[%d].schema_version must be %d", i, planRevisionSchemaVersion)
		}
		if strings.TrimSpace(revision.ID) == "" {
			return nil, fmt.Errorf("revisions[%d].id is required", i)
		}
		if _, duplicate := byID[revision.ID]; duplicate {
			return nil, fmt.Errorf("revisions[%d].id %q is duplicated", i, revision.ID)
		}
		if revision.Number != i+1 {
			return nil, fmt.Errorf("revisions[%d].number must be strictly increasing from one", i)
		}
		if i == 0 && revision.ParentID != "" {
			return nil, fmt.Errorf("revisions[0].parent_id must be empty")
		}
		if i > 0 && revision.ParentID != values[i-1].ID {
			return nil, fmt.Errorf("revisions[%d].parent_id must name the immediately preceding immutable revision", i)
		}
		if !revision.ReasonType.Valid() {
			return nil, fmt.Errorf("revisions[%d].reason_type %q is invalid", i, revision.ReasonType)
		}
		if strings.TrimSpace(revision.Reason) == "" || strings.TrimSpace(revision.CreatedAt) == "" {
			return nil, fmt.Errorf("revisions[%d] requires created_at and reason", i)
		}
		if err := validateSHA256(fmt.Sprintf("revisions[%d].plan_hash", i), revision.PlanHash); err != nil {
			return nil, err
		}
		if current {
			wantID := fmt.Sprintf("plan-r%d-%s", revision.Number, revision.PlanHash[:12])
			if revision.ID != wantID {
				return nil, fmt.Errorf("revisions[%d].id %q is not content-addressed by plan_hash", i, revision.ID)
			}
			computed, err := canonicalPlanCandidateProposalHash(revision)
			if err != nil {
				return nil, fmt.Errorf("hash revisions[%d] canonical proposal: %w", i, err)
			}
			if computed != revision.PlanHash {
				legacyComputed, legacyErr := planDefinitionHash(revision.Phases)
				if legacyErr != nil {
					return nil, fmt.Errorf("hash revisions[%d] legacy proposal: %w", i, legacyErr)
				}
				if legacyComputed != revision.PlanHash {
					return nil, fmt.Errorf("revisions[%d].plan_hash does not match its phase snapshot", i)
				}
			}
		}
		byID[revision.ID] = revision
	}
	return byID, nil
}

func validateCurrentPlanRevisionBindings(revision colony.PlanRevision, specification colony.SpecRevision) error {
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "semantic_id", value: revision.SemanticID},
		{name: "specification_revision_id", value: revision.SpecificationRevisionID},
		{name: "specification_revision_hash", value: revision.SpecificationRevisionHash},
		{name: "candidate_id", value: revision.CandidateID},
		{name: "candidate_content_hash", value: revision.CandidateContentHash},
		{name: "planning_timeline_id", value: revision.PlanningTimelineID},
		{name: "planning_timeline_digest", value: revision.PlanningTimelineDigest},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("%s is required for current accepted revision", required.name)
		}
	}
	if revision.SpecificationRevisionID != specification.ID || revision.SpecificationRevisionHash != specification.ContentHash {
		return fmt.Errorf("specification revision binding does not match current approved revision")
	}
	for _, hash := range []struct {
		name  string
		value string
	}{
		{name: "specification_revision_hash", value: revision.SpecificationRevisionHash},
		{name: "candidate_content_hash", value: revision.CandidateContentHash},
		{name: "planning_timeline_digest", value: revision.PlanningTimelineDigest},
	} {
		if err := validateSHA256(hash.name, hash.value); err != nil {
			return err
		}
	}
	return validateProofLinks("active revision", revision.RequirementProofLinks, revision.AcceptanceProofLinks, revision.NegativeProofLinks, revision.RecoveryProofLinks, revision.PublicPathProofLinks, specification)
}

func validateCurrentPlanNodes(phases []colony.Phase, revision colony.PlanRevision, specification colony.SpecRevision) error {
	seen := map[string]string{revision.SemanticID: "active revision"}
	for phaseIndex := range phases {
		phase := phases[phaseIndex]
		label := fmt.Sprintf("phases[%d]", phaseIndex)
		if err := validateBoundPlanNode(label, phase.SemanticID, phase.SpecificationRevisionID, phase.SpecificationRevisionHash, phase.CandidateID, phase.CandidateContentHash, phase.PlanningTimelineID, phase.PlanningTimelineDigest, revision); err != nil {
			return err
		}
		if prior, duplicate := seen[phase.SemanticID]; duplicate {
			return fmt.Errorf("%s.semantic_id %q is a duplicate stable ID already used by %s", label, phase.SemanticID, prior)
		}
		seen[phase.SemanticID] = label
		if err := validateProofLinks(label, phase.RequirementProofLinks, phase.AcceptanceProofLinks, phase.NegativeProofLinks, phase.RecoveryProofLinks, phase.PublicPathProofLinks, specification); err != nil {
			return err
		}
		for taskIndex := range phase.Tasks {
			task := phase.Tasks[taskIndex]
			taskLabel := fmt.Sprintf("%s.tasks[%d]", label, taskIndex)
			if err := validateBoundPlanNode(taskLabel, task.SemanticID, task.SpecificationRevisionID, task.SpecificationRevisionHash, task.CandidateID, task.CandidateContentHash, task.PlanningTimelineID, task.PlanningTimelineDigest, revision); err != nil {
				return err
			}
			if prior, duplicate := seen[task.SemanticID]; duplicate {
				return fmt.Errorf("%s.semantic_id %q is a duplicate stable ID already used by %s", taskLabel, task.SemanticID, prior)
			}
			seen[task.SemanticID] = taskLabel
			if err := validateProofLinks(taskLabel, task.RequirementProofLinks, task.AcceptanceProofLinks, task.NegativeProofLinks, task.RecoveryProofLinks, task.PublicPathProofLinks, specification); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBoundPlanNode(label, semanticID, specID, specHash, candidateID, candidateHash, timelineID, timelineDigest string, revision colony.PlanRevision) error {
	values := []struct {
		name, got, want string
	}{
		{name: "semantic_id", got: semanticID, want: semanticID},
		{name: "specification_revision_id", got: specID, want: revision.SpecificationRevisionID},
		{name: "specification_revision_hash", got: specHash, want: revision.SpecificationRevisionHash},
		{name: "candidate_id", got: candidateID, want: revision.CandidateID},
		{name: "candidate_content_hash", got: candidateHash, want: revision.CandidateContentHash},
		{name: "planning_timeline_id", got: timelineID, want: revision.PlanningTimelineID},
		{name: "planning_timeline_digest", got: timelineDigest, want: revision.PlanningTimelineDigest},
	}
	for _, value := range values {
		if strings.TrimSpace(value.got) == "" {
			return fmt.Errorf("%s.%s is required for current accepted plan", label, value.name)
		}
		if value.name != "semantic_id" && value.got != value.want {
			return fmt.Errorf("%s.%s does not match active revision", label, value.name)
		}
	}
	return nil
}

func validateProofLinks(label string, requirementIDs, acceptanceIDs, negativeIDs, recoveryIDs, publicPathIDs []string, specification colony.SpecRevision) error {
	sets := []struct {
		name   string
		values []string
		known  map[string]struct{}
	}{
		{name: "requirement_proof_links", values: requirementIDs, known: specRequirementIDs(specification.Requirements)},
		{name: "acceptance_proof_links", values: acceptanceIDs, known: specAcceptanceCheckIDs(specification.AcceptanceChecks)},
		{name: "negative_proof_links", values: negativeIDs, known: specNegativeExpectationIDs(specification.NegativeExpectations)},
		{name: "recovery_proof_links", values: recoveryIDs, known: specRecoveryExpectationIDs(specification.RecoveryExpectations)},
		{name: "public_path_proof_links", values: publicPathIDs, known: specPublicPathIDs(specification.AffectedPublicPaths)},
	}
	for _, set := range sets {
		if len(set.values) == 0 {
			return fmt.Errorf("%s.%s is required for current accepted plan", label, set.name)
		}
		seen := make(map[string]struct{}, len(set.values))
		for _, id := range set.values {
			if _, duplicate := seen[id]; duplicate {
				return fmt.Errorf("%s.%s contains duplicate stable ID %q", label, set.name, id)
			}
			seen[id] = struct{}{}
			if _, ok := set.known[id]; !ok {
				return fmt.Errorf("%s.%s references absent specification ID %q", label, set.name, id)
			}
		}
	}
	return nil
}

func validatePlanCandidateState(candidate colony.PlanCandidate, revisions map[string]colony.PlanRevision, specification colony.Specification) error {
	if err := validatePlanningSchemaVersion("schema_version", candidate.SchemaVersion, colony.PlanCandidateSchemaVersion); err != nil {
		return err
	}
	if err := validateSHA256("content_hash", candidate.ContentHash); err != nil {
		return err
	}
	if err := validateContentAddressedID("id", candidate.ID, candidate.ContentHash); err != nil {
		return err
	}
	canonical := validatePlanningRecordHashes(candidate) == nil
	if canonical {
		if err := candidate.Validate(); err != nil {
			return err
		}
	} else if err := validateRetainedPlanCandidateShape(candidate); err != nil {
		return err
	}

	if candidate.BasePlanRevisionID == "plan-unbound" {
		if candidate.Proposal.ParentID != "" || candidate.Proposal.Number != 1 {
			return fmt.Errorf("initial unbound proposal must be revision one without a parent")
		}
		if retained, ok := revisions[candidate.Proposal.ID]; !ok || retained.PlanHash != candidate.ProposalHash {
			return fmt.Errorf("initial unbound proposal does not name its retained first revision")
		}
	} else {
		base, ok := revisions[candidate.BasePlanRevisionID]
		if !ok {
			return fmt.Errorf("base_plan_revision_id %q does not name a retained revision", candidate.BasePlanRevisionID)
		}
		if base.PlanHash != candidate.BasePlanRevisionHash {
			return fmt.Errorf("base_plan_revision_hash does not match retained revision")
		}
		if candidate.Proposal.ParentID != base.ID || candidate.Proposal.Number != base.Number+1 {
			return fmt.Errorf("proposal does not extend the exact base plan revision")
		}
	}
	specRevision, ok := specificationRevisionByID(specification, candidate.SpecificationRevisionID)
	if !ok {
		return fmt.Errorf("specification_revision_id %q does not name a specification revision", candidate.SpecificationRevisionID)
	}
	approvedBinding := specRevision.ContentHash == candidate.SpecificationRevisionHash && specRevision.Status == colony.SpecStatusApproved && specRevision.Approval != nil
	historicalAcceptedBinding := specRevision.ContentHash == candidate.SpecificationRevisionHash && candidate.Status == colony.PlanCandidateAccepted &&
		candidate.Acceptance != nil && specRevision.Status == colony.SpecStatusSuperseded && specRevision.Approval != nil
	if !approvedBinding && !historicalAcceptedBinding {
		return fmt.Errorf("specification revision binding is not an exact approved revision")
	}
	if candidate.Proposal.ID == "" || candidate.ProposalHash != candidate.Proposal.PlanHash {
		return fmt.Errorf("proposal binding is incomplete")
	}
	if canonical {
		if err := validateStandalonePlanRevision(candidate.Proposal); err != nil {
			return fmt.Errorf("proposal: %w", err)
		}
	} else if err := validateRetainedPlanRevisionShape(candidate.Proposal); err != nil {
		return fmt.Errorf("proposal: %w", err)
	}
	if err := validateCandidateGapReachability(candidate); err != nil {
		return err
	}
	if candidate.Acceptance == nil {
		return nil
	}
	receipt := candidate.Acceptance
	if receipt.SpecificationRevisionID != candidate.SpecificationRevisionID || receipt.SpecificationRevisionHash != candidate.SpecificationRevisionHash ||
		receipt.BasePlanRevisionID != candidate.BasePlanRevisionID || receipt.BasePlanRevisionHash != candidate.BasePlanRevisionHash ||
		receipt.TimelineID != candidate.Timeline.ID || receipt.TimelineDigest != candidate.Timeline.TimelineDigest || receipt.ProposalHash != candidate.ProposalHash {
		return fmt.Errorf("acceptance receipt does not exactly bind candidate authorities")
	}
	return nil
}

// validateRetainedPlanCandidateShape preserves read compatibility for state
// snapshots written before complete review-payload addressing. State alone is
// never execution authority: persisted candidate and timeline artifacts are
// independently canonicalized by their loaders before review or build/run.
func validateRetainedPlanCandidateShape(candidate colony.PlanCandidate) error {
	if !candidate.Status.Valid() || candidate.CreatedAt.IsZero() || candidate.ExpiresAt.IsZero() {
		return fmt.Errorf("candidate status and timestamps are required")
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "proposal.id", value: candidate.Proposal.ID},
		{name: "proposal_hash", value: candidate.ProposalHash},
		{name: "base_plan_revision_id", value: candidate.BasePlanRevisionID},
		{name: "base_plan_revision_hash", value: candidate.BasePlanRevisionHash},
		{name: "specification_revision_id", value: candidate.SpecificationRevisionID},
		{name: "specification_revision_hash", value: candidate.SpecificationRevisionHash},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("%s is required", required.name)
		}
	}
	if candidate.Proposal.PlanHash != candidate.ProposalHash {
		return fmt.Errorf("proposal_hash must match proposal.plan_hash")
	}
	if err := validateTimelineBindingStructure(candidate.Timeline); err != nil {
		return fmt.Errorf("timeline: %w", err)
	}
	if err := candidate.StopDecision.Validate(); err != nil {
		return fmt.Errorf("stop_decision: %w", err)
	}
	if candidate.StopDecision.Reason == colony.PlanningStopContinue || candidate.StopDecision.Reason == colony.PlanningStopOwnerDecision {
		return fmt.Errorf("stop_decision.reason %q is not candidate eligible", candidate.StopDecision.Reason)
	}
	if err := validateRetainedPlanningAssessments(candidate.DimensionAssessments); err != nil {
		return err
	}
	if err := validateRetainedPlanningDelta(candidate.SemanticDelta); err != nil {
		return fmt.Errorf("semantic_delta: %w", err)
	}
	for i := range candidate.ResidualGaps {
		if err := candidate.ResidualGaps[i].Validate(); err != nil {
			return fmt.Errorf("residual_gaps[%d]: %w", i, err)
		}
	}
	if strings.TrimSpace(candidate.EvidenceThatWouldChange) == "" {
		return fmt.Errorf("evidence_that_would_change is required")
	}
	if err := validateRetainedRecommendation(candidate.Recommendation, candidate.ID); err != nil {
		return fmt.Errorf("recommendation: %w", err)
	}
	if candidate.Status == colony.PlanCandidateAccepted {
		if candidate.Acceptance == nil {
			return fmt.Errorf("acceptance is required for accepted candidate")
		}
		if err := candidate.Acceptance.Validate(); err != nil {
			return fmt.Errorf("acceptance: %w", err)
		}
		if candidate.Acceptance.CandidateID != candidate.ID || candidate.Acceptance.CandidateContentHash != candidate.ContentHash {
			return fmt.Errorf("acceptance candidate binding must match candidate")
		}
	} else if candidate.Acceptance != nil {
		return fmt.Errorf("acceptance is permitted only for accepted candidate")
	}
	return nil
}

func validateRetainedPlanRevisionShape(revision colony.PlanRevision) error {
	if revision.SchemaVersion != planRevisionSchemaVersion || revision.Number <= 0 || strings.TrimSpace(revision.ID) == "" || strings.TrimSpace(revision.CreatedAt) == "" || !revision.ReasonType.Valid() || strings.TrimSpace(revision.Reason) == "" || len(revision.Phases) == 0 {
		return fmt.Errorf("current proposal revision is partially populated")
	}
	if err := validateSHA256("plan_hash", revision.PlanHash); err != nil {
		return err
	}
	if revision.ID != fmt.Sprintf("plan-r%d-%s", revision.Number, revision.PlanHash[:12]) {
		return fmt.Errorf("id %q is not content-addressed by plan_hash", revision.ID)
	}
	computed, err := planDefinitionHash(revision.Phases)
	if err != nil {
		return err
	}
	if computed != revision.PlanHash {
		return fmt.Errorf("plan_hash does not match proposal phases")
	}
	return nil
}

func validateTimelineBindingStructure(binding colony.PlanningTimelineBinding) error {
	if err := validatePlanningSchemaVersion("schema_version", binding.SchemaVersion, colony.PlanningTimelineSchemaVersion); err != nil {
		return err
	}
	if err := validateAddressedHash("planning timeline binding", binding.ID, binding.ContentHash); err != nil {
		return err
	}
	return binding.Validate()
}

func validateRetainedPlanningAssessments(assessments []colony.PlanningDimensionAssessment) error {
	dimensions := colony.PlanningDimensions()
	if len(assessments) != len(dimensions) {
		return fmt.Errorf("dimension_assessments must contain exactly five dimensions")
	}
	seen := make(map[colony.PlanningDimension]struct{}, len(assessments))
	for i := range assessments {
		assessment := assessments[i]
		if err := validatePlanningSchemaVersion(fmt.Sprintf("dimension_assessments[%d].schema_version", i), assessment.SchemaVersion, colony.PlanningSchemaVersion); err != nil {
			return err
		}
		if err := validateAddressedHash(fmt.Sprintf("dimension_assessments[%d]", i), assessment.ID, assessment.ContentHash); err != nil {
			return err
		}
		if !assessment.Dimension.Valid() || assessment.Before < 0 || assessment.Before > 100 || assessment.After < 0 || assessment.After > 100 {
			return fmt.Errorf("dimension_assessments[%d] has an invalid dimension or whole-number score", i)
		}
		if _, duplicate := seen[assessment.Dimension]; duplicate {
			return fmt.Errorf("dimension_assessments contains duplicate %q", assessment.Dimension)
		}
		seen[assessment.Dimension] = struct{}{}
		if err := validateRetainedIDs(fmt.Sprintf("dimension_assessments[%d].fresh_evidence_ids", i), assessment.FreshEvidenceIDs, false); err != nil {
			return err
		}
		if err := validateRetainedIDs(fmt.Sprintf("dimension_assessments[%d].resolved_gap_ids", i), assessment.ResolvedGapIDs, false); err != nil {
			return err
		}
		if err := assessment.RemainingGap.Validate(); err != nil {
			return fmt.Errorf("dimension_assessments[%d].remaining_gap: %w", i, err)
		}
		if assessment.RemainingGap.Dimension != assessment.Dimension || strings.TrimSpace(assessment.Rationale) == "" || strings.TrimSpace(assessment.ProducerReceiptID) == "" {
			return fmt.Errorf("dimension_assessments[%d] has incomplete gap or producer attribution", i)
		}
	}
	return nil
}

func validateRetainedPlanningDelta(delta colony.PlanningSemanticDelta) error {
	if err := validatePlanningSchemaVersion("schema_version", delta.SchemaVersion, colony.PlanningSchemaVersion); err != nil {
		return err
	}
	if err := validateAddressedHash("planning delta", delta.ID, delta.ContentHash); err != nil {
		return err
	}
	sections := [][]colony.PlanningSemanticChange{
		delta.Phases, delta.Tasks, delta.Dependencies, delta.RequirementLinks,
		delta.AcceptanceChecks, delta.NegativeExpectations, delta.RecoveryExpectations, delta.PublicPaths,
	}
	count := len(delta.AuthorityImpacts)
	seenChanges := make(map[string]struct{})
	for _, changes := range sections {
		count += len(changes)
		for i := range changes {
			if err := changes[i].Validate(); err != nil {
				return err
			}
			if _, duplicate := seenChanges[changes[i].SemanticID]; duplicate {
				return fmt.Errorf("semantic_id %q is duplicated", changes[i].SemanticID)
			}
			seenChanges[changes[i].SemanticID] = struct{}{}
		}
	}
	if count == 0 {
		return fmt.Errorf("semantic delta must contain at least one semantic change or authority impact")
	}
	seenImpacts := make(map[string]struct{}, len(delta.AuthorityImpacts))
	for i := range delta.AuthorityImpacts {
		impact := delta.AuthorityImpacts[i]
		if err := validateAddressedHash(fmt.Sprintf("authority_impacts[%d]", i), impact.ID, impact.ContentHash); err != nil {
			return err
		}
		if !impact.Kind.Valid() || strings.TrimSpace(impact.SourceID) == "" || strings.TrimSpace(impact.Rationale) == "" {
			return fmt.Errorf("authority_impacts[%d] has incomplete attribution", i)
		}
		if err := validateRetainedIDs(fmt.Sprintf("authority_impacts[%d].affected_semantic_ids", i), impact.AffectedSemanticIDs, true); err != nil {
			return err
		}
		if _, duplicate := seenImpacts[impact.ID]; duplicate {
			return fmt.Errorf("authority_impacts[%d].id %q is duplicated", i, impact.ID)
		}
		seenImpacts[impact.ID] = struct{}{}
	}
	return nil
}

func validateRetainedRecommendation(recommendation colony.QueenPlanRecommendation, candidateID string) error {
	if err := validatePlanningSchemaVersion("schema_version", recommendation.SchemaVersion, colony.PlanningSchemaVersion); err != nil {
		return err
	}
	if err := validateAddressedHash("queen recommendation", recommendation.ID, recommendation.ContentHash); err != nil {
		return err
	}
	if recommendation.CandidateID != candidateID || !recommendation.Disposition.Valid() || !recommendation.Producer.Valid() || recommendation.CreatedAt.IsZero() || strings.TrimSpace(recommendation.Rationale) == "" || strings.TrimSpace(recommendation.ProducerID) == "" {
		return fmt.Errorf("recommendation identity or attribution is incomplete")
	}
	return validateRetainedIDs("recommendation.evidence_ids", recommendation.EvidenceIDs, true)
}

func validateRetainedIDs(field string, values []string, required bool) error {
	if required && len(values) == 0 {
		return fmt.Errorf("%s are required", field)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != strings.TrimSpace(value) || value == "" {
			return fmt.Errorf("%s contains an empty or non-canonical ID", field)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("%s contains duplicate ID %q", field, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

type planCandidateBase struct {
	ID     string
	Hash   string
	Number int
}

// candidateAcceptanceBase translates a mutable planning-stage base snapshot
// into the immutable revision identity a stopped candidate must name. Legacy
// plans receive a deterministic retained baseline only when the candidate is
// accepted; fresh plans use the explicit plan-unbound genesis marker.
func candidateAcceptanceBase(plan colony.Plan, createdAt time.Time) (planCandidateBase, *colony.PlanRevision, error) {
	if len(plan.Phases) == 0 {
		hash, err := planStateHash(plan)
		if err != nil {
			return planCandidateBase{}, nil, err
		}
		return planCandidateBase{ID: "plan-unbound", Hash: hash}, nil, nil
	}
	if revision, ok := activePlanRevision(plan); ok {
		return planCandidateBase{ID: revision.ID, Hash: revision.PlanHash, Number: revision.Number}, nil, nil
	}
	if len(plan.Revisions) != 0 || strings.TrimSpace(plan.ActiveRevisionID) != "" {
		return planCandidateBase{}, nil, fmt.Errorf("base_plan_revision_id: active plan revision is not retained")
	}
	baselinePlan := plan
	baselinePlan.Phases = clonePhases(plan.Phases)
	baseline, err := newPlanRevision(baselinePlan, 1, "", colony.PlanRevisionLegacyImport, "Existing plan imported when explicit candidate acceptance was enabled", nil, "", "", "", nil, nil, phaseIDs(plan.Phases), createdAt)
	if err != nil {
		return planCandidateBase{}, nil, err
	}
	return planCandidateBase{ID: baseline.ID, Hash: baseline.PlanHash, Number: baseline.Number}, &baseline, nil
}

func validateStandalonePlanRevision(revision colony.PlanRevision) error {
	if revision.SchemaVersion > planRevisionSchemaVersion {
		return fmt.Errorf("schema_version uses unsupported future plan revision major %d", revision.SchemaVersion)
	}
	if revision.SchemaVersion != planRevisionSchemaVersion || revision.Number <= 0 || strings.TrimSpace(revision.ID) == "" || strings.TrimSpace(revision.CreatedAt) == "" || !revision.ReasonType.Valid() || strings.TrimSpace(revision.Reason) == "" {
		return fmt.Errorf("current proposal revision is partially populated")
	}
	if len(revision.Phases) == 0 {
		return fmt.Errorf("current proposal revision requires phases")
	}
	if err := validateSHA256("plan_hash", revision.PlanHash); err != nil {
		return err
	}
	wantID := fmt.Sprintf("plan-r%d-%s", revision.Number, revision.PlanHash[:12])
	if revision.ID != wantID {
		return fmt.Errorf("id %q is not content-addressed by plan_hash", revision.ID)
	}
	computed, err := canonicalPlanCandidateProposalHash(revision)
	if err != nil {
		return err
	}
	if computed != revision.PlanHash {
		return fmt.Errorf("plan_hash does not match proposal phases")
	}
	if err := validateSHA256("candidate_content_hash", revision.CandidateContentHash); err != nil {
		return err
	}
	if revision.CandidateID != "plan-candidate-"+revision.CandidateContentHash[:12] {
		return fmt.Errorf("candidate_id does not match candidate_content_hash")
	}
	for phaseIndex := range revision.Phases {
		phase := revision.Phases[phaseIndex]
		if phase.CandidateID != revision.CandidateID || phase.CandidateContentHash != revision.CandidateContentHash {
			return fmt.Errorf("phases[%d] candidate back-reference does not match proposal", phaseIndex)
		}
		for taskIndex := range phase.Tasks {
			if phase.Tasks[taskIndex].CandidateID != revision.CandidateID || phase.Tasks[taskIndex].CandidateContentHash != revision.CandidateContentHash {
				return fmt.Errorf("phases[%d].tasks[%d] candidate back-reference does not match proposal", phaseIndex, taskIndex)
			}
		}
	}
	return nil
}

func validatePlanningRecordHashes(candidate colony.PlanCandidate) error {
	if err := validatePlanningSchemaVersion("schema_version", candidate.SchemaVersion, colony.PlanCandidateSchemaVersion); err != nil {
		return err
	}
	if err := validateTimelineBindingShape(candidate.Timeline); err != nil {
		return fmt.Errorf("timeline: %w", err)
	}
	if err := validatePlanningStopShape("stop_decision", candidate.StopDecision); err != nil {
		return err
	}
	if err := validatePlanningDeltaShape("semantic_delta", candidate.SemanticDelta); err != nil {
		return err
	}
	for i := range candidate.DimensionAssessments {
		if err := validatePlanningAssessmentShape(fmt.Sprintf("dimension_assessments[%d]", i), candidate.DimensionAssessments[i]); err != nil {
			return err
		}
	}
	for i := range candidate.ResidualGaps {
		if err := validatePlanningGapShape(fmt.Sprintf("residual_gaps[%d]", i), candidate.ResidualGaps[i]); err != nil {
			return err
		}
	}
	if err := validatePlanningSchemaVersion("recommendation.schema_version", candidate.Recommendation.SchemaVersion, colony.PlanningSchemaVersion); err != nil {
		return err
	}
	if err := validateAddressedHash("recommendation", candidate.Recommendation.ID, candidate.Recommendation.ContentHash); err != nil {
		return err
	}
	if err := candidate.Recommendation.Validate(); err != nil {
		return fmt.Errorf("recommendation: %w", err)
	}
	if candidate.Acceptance != nil {
		if err := validatePlanningSchemaVersion("acceptance.schema_version", candidate.Acceptance.SchemaVersion, colony.PlanAcceptanceSchemaVersion); err != nil {
			return err
		}
		if err := validateAddressedHash("acceptance", candidate.Acceptance.ID, candidate.Acceptance.ContentHash); err != nil {
			return err
		}
		for _, value := range []struct{ name, hash string }{
			{name: "acceptance.candidate_content_hash", hash: candidate.Acceptance.CandidateContentHash},
			{name: "acceptance.specification_revision_hash", hash: candidate.Acceptance.SpecificationRevisionHash},
			{name: "acceptance.base_plan_revision_hash", hash: candidate.Acceptance.BasePlanRevisionHash},
			{name: "acceptance.timeline_digest", hash: candidate.Acceptance.TimelineDigest},
			{name: "acceptance.proposal_hash", hash: candidate.Acceptance.ProposalHash},
			{name: "acceptance.acceptance_token_hash", hash: candidate.Acceptance.AcceptanceTokenHash},
			{name: "acceptance.activated_plan_revision_hash", hash: candidate.Acceptance.ActivatedPlanRevisionHash},
		} {
			if err := validateSHA256(value.name, value.hash); err != nil {
				return err
			}
		}
	}
	for _, value := range []struct{ name, hash string }{
		{name: "content_hash", hash: candidate.ContentHash},
		{name: "proposal_hash", hash: candidate.ProposalHash},
		{name: "base_plan_revision_hash", hash: candidate.BasePlanRevisionHash},
		{name: "specification_revision_hash", hash: candidate.SpecificationRevisionHash},
	} {
		if err := validateSHA256(value.name, value.hash); err != nil {
			return err
		}
	}
	proposalHash, err := canonicalPlanCandidateProposalHash(candidate.Proposal)
	if err != nil {
		return fmt.Errorf("proposal_hash: %w", err)
	}
	if candidate.ProposalHash != proposalHash || candidate.Proposal.PlanHash != proposalHash {
		return fmt.Errorf("proposal_hash does not match the canonical stripped proposal")
	}
	if err := validateStandalonePlanRevision(candidate.Proposal); err != nil {
		return fmt.Errorf("proposal: %w", err)
	}
	wantCandidateHash, err := canonicalPlanCandidateContentHash(candidate)
	if err != nil {
		return err
	}
	if candidate.ContentHash != wantCandidateHash || candidate.ID != "plan-candidate-"+wantCandidateHash[:12] {
		return fmt.Errorf("candidate content address does not match its complete immutable review payload")
	}
	if err := validatePlanCandidateBackReferences(candidate); err != nil {
		return err
	}
	return nil
}

func validatePlanCandidateBackReferences(candidate colony.PlanCandidate) error {
	bindings := []struct {
		label string
		id    string
		hash  string
	}{
		{label: "proposal", id: candidate.Proposal.CandidateID, hash: candidate.Proposal.CandidateContentHash},
	}
	for phaseIndex := range candidate.Proposal.Phases {
		phase := candidate.Proposal.Phases[phaseIndex]
		bindings = append(bindings, struct {
			label string
			id    string
			hash  string
		}{label: fmt.Sprintf("proposal.phases[%d]", phaseIndex), id: phase.CandidateID, hash: phase.CandidateContentHash})
		for taskIndex := range phase.Tasks {
			bindings = append(bindings, struct {
				label string
				id    string
				hash  string
			}{label: fmt.Sprintf("proposal.phases[%d].tasks[%d]", phaseIndex, taskIndex), id: phase.Tasks[taskIndex].CandidateID, hash: phase.Tasks[taskIndex].CandidateContentHash})
		}
	}
	for _, binding := range bindings {
		if binding.id != candidate.ID || binding.hash != candidate.ContentHash {
			return fmt.Errorf("%s candidate back-reference does not match candidate identity", binding.label)
		}
	}
	if candidate.Recommendation.CandidateID != candidate.ID {
		return fmt.Errorf("recommendation candidate back-reference does not match candidate identity")
	}
	return nil
}

// validatePlanningTimelineBinding validates the immutable card chain supplied
// by the artifact loader against the small binding retained in state.
func validatePlanningTimelineBinding(binding colony.PlanningTimelineBinding, cards []colony.PlanningIterationCard) error {
	if err := validateTimelineBindingStructure(binding); err != nil {
		return err
	}
	if len(cards) == 0 || len(cards) != len(binding.CardIDs) {
		return fmt.Errorf("timeline card_ids do not resolve to the complete card chain")
	}
	previousIteration := 0
	seen := make(map[string]struct{}, len(cards))
	for i := range cards {
		card := cards[i]
		if err := validateRetainedPlanningIterationCardShape(card); err != nil {
			return fmt.Errorf("cards[%d]: %w", i, err)
		}
		if card.RunID != binding.RunID {
			return fmt.Errorf("cards[%d].run_id does not match timeline", i)
		}
		if card.ID != binding.CardIDs[i] {
			return fmt.Errorf("card_ids[%d] does not resolve to cards[%d]", i, i)
		}
		if _, duplicate := seen[card.ID]; duplicate {
			return fmt.Errorf("card_ids contains duplicate %q", card.ID)
		}
		seen[card.ID] = struct{}{}
		if card.Iteration <= previousIteration {
			return fmt.Errorf("card iteration order must be strictly increasing")
		}
		previousIteration = card.Iteration
	}
	if binding.FirstCardHash != cards[0].ContentHash {
		return fmt.Errorf("first_card_hash does not match the first card")
	}
	if binding.LastCardHash != cards[len(cards)-1].ContentHash {
		return fmt.Errorf("last_card_hash does not match the last card")
	}
	digest, err := planningTimelineDigest(cards)
	if err != nil {
		return fmt.Errorf("hash timeline: %w", err)
	}
	if binding.TimelineDigest != digest {
		return fmt.Errorf("timeline_digest does not match the ordered card chain")
	}
	return nil
}

func validateRetainedPlanningIterationCardShape(card colony.PlanningIterationCard) error {
	if err := validatePlanningSchemaVersion("schema_version", card.SchemaVersion, colony.PlanningIterationSchemaVersion); err != nil {
		return err
	}
	if err := validateAddressedHash("card", card.ID, card.ContentHash); err != nil {
		return err
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "run_id", value: card.RunID},
		{name: "scout_receipt_id", value: card.ScoutReceiptID},
		{name: "route_setter_receipt_id", value: card.RouteSetterReceiptID},
		{name: "evidence_that_would_change", value: card.EvidenceThatWouldChange},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("%s is required", required.name)
		}
	}
	if card.Iteration <= 0 || card.CreatedAt.IsZero() {
		return fmt.Errorf("iteration and created_at are required")
	}
	if err := validateSHA256("scout_receipt_hash", card.ScoutReceiptHash); err != nil {
		return err
	}
	if err := validateSHA256("route_setter_receipt_hash", card.RouteSetterReceiptHash); err != nil {
		return err
	}
	if err := validateRetainedIDs("evidence_ids", card.EvidenceIDs, true); err != nil {
		return err
	}
	if err := validateRetainedPlanningAssessments(card.DimensionAssessments); err != nil {
		return err
	}
	if err := card.WeakestGap.Validate(); err != nil {
		return fmt.Errorf("weakest_gap: %w", err)
	}
	if err := validateRetainedPlanningDelta(card.SemanticDelta); err != nil {
		return fmt.Errorf("semantic_delta: %w", err)
	}
	if err := card.Decision.Validate(); err != nil {
		return fmt.Errorf("decision: %w", err)
	}
	knownGaps := make(map[string]struct{}, len(card.DimensionAssessments))
	for _, assessment := range card.DimensionAssessments {
		knownGaps[assessment.RemainingGap.ID] = struct{}{}
	}
	if _, ok := knownGaps[card.WeakestGap.ID]; !ok {
		return fmt.Errorf("weakest_gap does not resolve to a dimension assessment remaining gap")
	}
	if card.Decision.SelectedGapID != "" && card.Decision.SelectedGapID != card.WeakestGap.ID {
		return fmt.Errorf("decision.selected_gap_id does not resolve to weakest_gap")
	}
	for _, id := range card.Decision.ResidualGapIDs {
		if _, ok := knownGaps[id]; !ok {
			return fmt.Errorf("decision.residual_gap_ids references absent gap %q", id)
		}
	}
	return nil
}

func planningTimelineDigest(cards []colony.PlanningIterationCard) (string, error) {
	type cardLink struct {
		ID          string `json:"id"`
		ContentHash string `json:"content_hash"`
		RunID       string `json:"run_id"`
		Iteration   int    `json:"iteration"`
	}
	links := make([]cardLink, len(cards))
	for i := range cards {
		links[i] = cardLink{ID: cards[i].ID, ContentHash: cards[i].ContentHash, RunID: cards[i].RunID, Iteration: cards[i].Iteration}
	}
	return jsonSHA256(links)
}

func validatePlanningIterationCardShape(card colony.PlanningIterationCard) error {
	if err := validatePlanningSchemaVersion("schema_version", card.SchemaVersion, colony.PlanningIterationSchemaVersion); err != nil {
		return err
	}
	if err := validateAddressedHash("card", card.ID, card.ContentHash); err != nil {
		return err
	}
	if err := validateSHA256("scout_receipt_hash", card.ScoutReceiptHash); err != nil {
		return err
	}
	if err := validateSHA256("route_setter_receipt_hash", card.RouteSetterReceiptHash); err != nil {
		return err
	}
	for i := range card.DimensionAssessments {
		if err := validatePlanningAssessmentShape(fmt.Sprintf("dimension_assessments[%d]", i), card.DimensionAssessments[i]); err != nil {
			return err
		}
	}
	if err := validatePlanningGapShape("weakest_gap", card.WeakestGap); err != nil {
		return err
	}
	if err := validatePlanningDeltaShape("semantic_delta", card.SemanticDelta); err != nil {
		return err
	}
	if err := validatePlanningStopShape("decision", card.Decision); err != nil {
		return err
	}
	knownGaps := make(map[string]struct{}, len(card.DimensionAssessments))
	for _, assessment := range card.DimensionAssessments {
		knownGaps[assessment.RemainingGap.ID] = struct{}{}
	}
	if _, ok := knownGaps[card.WeakestGap.ID]; !ok {
		return fmt.Errorf("weakest_gap does not resolve to a dimension assessment remaining gap")
	}
	if card.Decision.SelectedGapID != "" && card.Decision.SelectedGapID != card.WeakestGap.ID {
		return fmt.Errorf("decision.selected_gap_id does not resolve to weakest_gap")
	}
	for _, id := range card.Decision.ResidualGapIDs {
		if _, ok := knownGaps[id]; !ok {
			return fmt.Errorf("decision.residual_gap_ids references absent gap %q", id)
		}
	}
	return card.Validate()
}

func validateCandidateGapReachability(candidate colony.PlanCandidate) error {
	known := make(map[string]struct{}, len(candidate.ResidualGaps))
	for _, gap := range candidate.ResidualGaps {
		if _, duplicate := known[gap.ID]; duplicate {
			return fmt.Errorf("residual_gaps contains duplicate stable ID %q", gap.ID)
		}
		known[gap.ID] = struct{}{}
	}
	if candidate.StopDecision.SelectedGapID != "" {
		if _, ok := known[candidate.StopDecision.SelectedGapID]; !ok {
			return fmt.Errorf("stop_decision.selected_gap_id references absent residual gap %q", candidate.StopDecision.SelectedGapID)
		}
	}
	for _, id := range candidate.StopDecision.ResidualGapIDs {
		if _, ok := known[id]; !ok {
			return fmt.Errorf("stop_decision.residual_gap_ids references absent residual gap %q", id)
		}
	}
	return nil
}

func validateTimelineBindingShape(binding colony.PlanningTimelineBinding) error {
	if err := validatePlanningSchemaVersion("schema_version", binding.SchemaVersion, colony.PlanningTimelineSchemaVersion); err != nil {
		return err
	}
	if err := validateAddressedHash("timeline", binding.ID, binding.ContentHash); err != nil {
		return err
	}
	for _, value := range []struct{ name, hash string }{
		{name: "first_card_hash", hash: binding.FirstCardHash},
		{name: "last_card_hash", hash: binding.LastCardHash},
		{name: "timeline_digest", hash: binding.TimelineDigest},
	} {
		if err := validateSHA256(value.name, value.hash); err != nil {
			return err
		}
	}
	if err := validatePlanningArtifactPath(binding.Path); err != nil {
		return fmt.Errorf("path: %w", err)
	}
	if err := binding.Validate(); err != nil {
		return err
	}
	want, err := planningTimelineBindingContentHash(binding)
	if err != nil {
		return err
	}
	if binding.ContentHash != want || binding.ID != "planning-timeline-"+want[:12] {
		return fmt.Errorf("planning timeline binding content address does not match its canonical payload")
	}
	return nil
}

func validatePlanningAssessmentShape(label string, assessment colony.PlanningDimensionAssessment) error {
	if err := validatePlanningSchemaVersion(label+".schema_version", assessment.SchemaVersion, colony.PlanningSchemaVersion); err != nil {
		return err
	}
	if err := assessment.Validate(); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if err := validatePlanningGapShape(label+".remaining_gap", assessment.RemainingGap); err != nil {
		return err
	}
	return nil
}

func validatePlanningGapShape(label string, gap colony.PlanningGap) error {
	if err := validatePlanningSchemaVersion(label+".schema_version", gap.SchemaVersion, colony.PlanningSchemaVersion); err != nil {
		return err
	}
	if err := gap.Validate(); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	want, err := colony.CanonicalPlanningGapContentHash(gap)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if gap.ContentHash != want || gap.ID != "planning-gap-"+want[:12] {
		return fmt.Errorf("%s content address does not match its canonical body", label)
	}
	return nil
}

func validatePlanningDeltaShape(label string, delta colony.PlanningSemanticDelta) error {
	if err := validatePlanningSchemaVersion(label+".schema_version", delta.SchemaVersion, colony.PlanningSchemaVersion); err != nil {
		return err
	}
	if err := delta.Validate(); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return nil
}

func validatePlanningStopShape(label string, decision colony.PlanningStopDecision) error {
	if err := validatePlanningSchemaVersion(label+".schema_version", decision.SchemaVersion, colony.PlanningSchemaVersion); err != nil {
		return err
	}
	if err := validateAddressedHash(label, decision.ID, decision.ContentHash); err != nil {
		return err
	}
	want, err := canonicalPlanningStopDecisionHash(decision)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if decision.ContentHash != want || decision.ID != "planning-stop-"+want[:12] {
		return fmt.Errorf("%s content address does not match its canonical body", label)
	}
	return decision.Validate()
}

func canonicalPlanningStopDecisionHash(decision colony.PlanningStopDecision) (string, error) {
	payload := struct {
		Reason                  colony.PlanningStopReason `json:"reason"`
		SelectedGapID           string                    `json:"selected_gap_id"`
		ResidualGapIDs          []string                  `json:"residual_gap_ids"`
		EvidenceIDs             []string                  `json:"evidence_ids"`
		Rationale               string                    `json:"rationale"`
		EvidenceThatWouldChange string                    `json:"evidence_that_would_change"`
	}{
		Reason: decision.Reason, SelectedGapID: decision.SelectedGapID,
		ResidualGapIDs: append([]string(nil), decision.ResidualGapIDs...), EvidenceIDs: uniqueSortedStrings(decision.EvidenceIDs),
		Rationale: decision.Rationale, EvidenceThatWouldChange: decision.EvidenceThatWouldChange,
	}
	return jsonSHA256(payload)
}

func addressPlanningStopDecision(decision *colony.PlanningStopDecision) error {
	if decision == nil {
		return fmt.Errorf("planning stop decision is required")
	}
	decision.SchemaVersion = colony.PlanningSchemaVersion
	decision.EvidenceIDs = uniqueSortedStrings(decision.EvidenceIDs)
	hash, err := canonicalPlanningStopDecisionHash(*decision)
	if err != nil {
		return err
	}
	decision.ContentHash = hash
	decision.ID = "planning-stop-" + hash[:12]
	return validatePlanningStopShape("stop_decision", *decision)
}

func validatePlanningArtifactPath(path string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("planning artifact path must be repository-relative")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if clean != path || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("planning artifact path %q escapes or is not canonical", path)
	}
	if clean != ".aether/data/planning" && !strings.HasPrefix(clean, ".aether/data/planning/") {
		return fmt.Errorf("planning artifact path %q is outside .aether/data/planning", path)
	}
	return nil
}

func validatePlanningSchemaVersion(field, got, supported string) error {
	if got == supported {
		return nil
	}
	gotFamily, gotMajor, gotOK := splitPlanningSchemaVersion(got)
	wantFamily, wantMajor, wantOK := splitPlanningSchemaVersion(supported)
	if gotOK && wantOK && gotFamily == wantFamily && gotMajor > wantMajor {
		return fmt.Errorf("%s uses unsupported future schema major %d; this runtime supports %q", field, gotMajor, supported)
	}
	return fmt.Errorf("%s must be supported schema %q, got %q", field, supported, got)
}

func splitPlanningSchemaVersion(value string) (string, int, bool) {
	parts := strings.Split(value, "/v")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return "", 0, false
	}
	major, err := strconv.Atoi(parts[1])
	if err != nil || major < 1 {
		return "", 0, false
	}
	return parts[0], major, true
}

func validateSHA256(field, value string) error {
	if !planningSHA256Pattern.MatchString(value) {
		return fmt.Errorf("%s must be a lowercase 64-character SHA-256 digest", field)
	}
	return nil
}

func validateAddressedHash(label, id, hash string) error {
	if err := validateSHA256(label+".content_hash", hash); err != nil {
		return err
	}
	return validateContentAddressedID(label+".id", id, hash)
}

func validateContentAddressedID(field, id, digest string) error {
	if err := validateSHA256(strings.TrimSuffix(field, ".id")+".content_hash", digest); err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" || !strings.HasSuffix(id, "-"+digest[:12]) {
		return fmt.Errorf("%s %q is not content-addressed by its digest", field, id)
	}
	return nil
}

func planHasCurrentAuthority(plan colony.Plan) bool {
	return plan.PendingCandidateID != "" || len(plan.Candidates) != 0 || planHasCurrentBindings(plan)
}

func planHasCurrentBindings(plan colony.Plan) bool {
	for _, revision := range plan.Revisions {
		if revision.SemanticID != "" || revision.SpecificationRevisionID != "" || revision.CandidateID != "" || revision.PlanningTimelineID != "" {
			return true
		}
	}
	for _, phase := range plan.Phases {
		if phase.SemanticID != "" || phase.SpecificationRevisionID != "" || phase.CandidateID != "" || phase.PlanningTimelineID != "" {
			return true
		}
		for _, task := range phase.Tasks {
			if task.SemanticID != "" || task.SpecificationRevisionID != "" || task.CandidateID != "" || task.PlanningTimelineID != "" {
				return true
			}
		}
	}
	return false
}

// syncActivePlanRevisionExecutionFacts keeps mutable lifecycle status beside
// the immutable accepted definition. Definition hashes deliberately exclude
// these fields, but current-schema validation requires the active projection
// and its revision snapshot to describe the same execution state.
func syncActivePlanRevisionExecutionFacts(plan *colony.Plan) {
	if plan == nil || plan.AcceptancePolicy != colony.PlanAcceptanceExplicitOwner || strings.TrimSpace(plan.ActiveRevisionID) == "" {
		return
	}
	for revisionIndex := range plan.Revisions {
		revision := &plan.Revisions[revisionIndex]
		if revision.ID != plan.ActiveRevisionID || len(revision.Phases) != len(plan.Phases) {
			continue
		}
		for phaseIndex := range plan.Phases {
			if len(revision.Phases[phaseIndex].Tasks) != len(plan.Phases[phaseIndex].Tasks) {
				return
			}
			revision.Phases[phaseIndex].Status = plan.Phases[phaseIndex].Status
			revision.Phases[phaseIndex].WatcherFailureCount = plan.Phases[phaseIndex].WatcherFailureCount
			for taskIndex := range plan.Phases[phaseIndex].Tasks {
				revision.Phases[phaseIndex].Tasks[taskIndex].Status = plan.Phases[phaseIndex].Tasks[taskIndex].Status
			}
		}
		return
	}
}

func currentSpecificationRevision(specification colony.Specification) (colony.SpecRevision, bool) {
	return specificationRevisionByID(specification, specification.CurrentRevisionID)
}

func specificationRevisionByID(specification colony.Specification, id string) (colony.SpecRevision, bool) {
	for _, revision := range specification.Revisions {
		if revision.ID == id {
			return revision, true
		}
	}
	return colony.SpecRevision{}, false
}

func specOutcomeHashes(values []colony.SpecOutcome) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ContentHash
	}
	return result
}

func specIncludedBehaviorHashes(values []colony.SpecIncludedBehavior) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ContentHash
	}
	return result
}

func specExclusionHashes(values []colony.SpecExclusion) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ContentHash
	}
	return result
}

func specBindingDecisionHashes(values []colony.SpecBindingDecision) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ContentHash
	}
	return result
}

func specRequirementHashes(values []colony.SpecRequirement) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ContentHash
	}
	return result
}

func specAcceptanceCheckHashes(values []colony.SpecAcceptanceCheck) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ContentHash
	}
	return result
}

func specNegativeExpectationHashes(values []colony.SpecNegativeExpectation) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ContentHash
	}
	return result
}

func specRecoveryExpectationHashes(values []colony.SpecRecoveryExpectation) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ContentHash
	}
	return result
}

func specPublicPathHashes(values []colony.SpecPublicPath) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ContentHash
	}
	return result
}

func specRequirementIDs(values []colony.SpecRequirement) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value.ID] = struct{}{}
	}
	return result
}

func specAcceptanceCheckIDs(values []colony.SpecAcceptanceCheck) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value.ID] = struct{}{}
	}
	return result
}

func specNegativeExpectationIDs(values []colony.SpecNegativeExpectation) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value.ID] = struct{}{}
	}
	return result
}

func specRecoveryExpectationIDs(values []colony.SpecRecoveryExpectation) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value.ID] = struct{}{}
	}
	return result
}

func specPublicPathIDs(values []colony.SpecPublicPath) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value.ID] = struct{}{}
	}
	return result
}
