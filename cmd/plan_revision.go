package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

const planRevisionSchemaVersion = 1

type codexPlanRevisionContext struct {
	BaseRevisionID    string                    `json:"base_revision_id,omitempty"`
	BasePlanStateHash string                    `json:"base_plan_state_hash"`
	ReasonType        colony.PlanRevisionReason `json:"reason_type"`
	Reason            string                    `json:"reason"`
	Evidence          []string                  `json:"evidence,omitempty"`
	EvidenceHash      string                    `json:"evidence_hash,omitempty"`
	CompletedPhases   []colony.Phase            `json:"completed_phases,omitempty"`
	SupersededPhases  []colony.Phase            `json:"superseded_phases,omitempty"`
}

type planCandidateAcceptanceOptions struct {
	AcceptedBy string
	AcceptedAt time.Time
	Fault      lifecycleTransactionFaultHook
	Rename     func(oldPath, newPath string) error
}

type planCandidateAcceptanceResult struct {
	Candidate colony.PlanCandidate         `json:"candidate"`
	Revision  colony.PlanRevision          `json:"revision"`
	Receipt   colony.PlanAcceptanceReceipt `json:"acceptance_receipt"`
	Replayed  bool                         `json:"replayed"`
}

func planningRouteAcceptanceRepositoryPath(runID string) string {
	return filepath.ToSlash(filepath.Join(".aether", "data", "planning", strings.TrimSpace(runID), "acceptance.json"))
}

// acceptPlanCandidate is the sole pending_review -> accepted authority. Every
// refusal happens before the lifecycle transaction is opened, so stale input
// cannot leave journals or partially mutate the plan frontier.
func acceptPlanCandidate(root string, request planCandidateAcceptanceRequest, opts planCandidateAcceptanceOptions) (planCandidateAcceptanceResult, error) {
	empty := planCandidateAcceptanceResult{}
	artifact, err := loadPlanCandidateArtifact(root, request.CandidateID)
	if err != nil {
		return empty, fmt.Errorf("candidate_id: %w", err)
	}
	candidate := artifact.Candidate
	if err := validatePlanCandidateAcceptanceRequest(candidate, request); err != nil {
		return empty, err
	}
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		return empty, err
	}
	if err := validatePlanningState(state); err != nil {
		return empty, fmt.Errorf("validate current planning state: %w", err)
	}

	if candidate.Status == colony.PlanCandidateAccepted {
		return replayAcceptedPlanCandidate(state, candidate)
	}
	if candidate.Status != colony.PlanCandidatePendingReview {
		return empty, fmt.Errorf("candidate status %q cannot be accepted; only pending_review is eligible", candidate.Status)
	}
	if artifact.Stage.Stage != planningStageCandidateReady {
		return empty, fmt.Errorf("candidate stage %q cannot be accepted; expected candidate_ready", artifact.Stage.Stage)
	}
	if pendingID := strings.TrimSpace(state.Plan.PendingCandidateID); pendingID != "" && pendingID != candidate.ID {
		return empty, fmt.Errorf("candidate_id: current pending candidate is %q, not %q", pendingID, candidate.ID)
	}
	if _, err := verifiedPlanCandidateTimeline(root, candidate); err != nil {
		return empty, err
	}
	if state.Specification == nil {
		return empty, fmt.Errorf("specification_revision_id: candidate acceptance requires the current approved specification")
	}
	currentSpec, ok := currentSpecificationRevision(*state.Specification)
	if !ok || currentSpec.Status != colony.SpecStatusApproved || currentSpec.Approval == nil {
		return empty, fmt.Errorf("specification_revision_id: candidate acceptance requires the current approved specification")
	}
	if currentSpec.ID != candidate.SpecificationRevisionID {
		return empty, fmt.Errorf("specification_revision_id: current revision is %q, candidate binds %q", currentSpec.ID, candidate.SpecificationRevisionID)
	}
	if currentSpec.ContentHash != candidate.SpecificationRevisionHash {
		return empty, fmt.Errorf("specification_revision_hash: current revision no longer matches candidate")
	}
	if err := validateCandidateRunBase(state.Plan, artifact.Header); err != nil {
		return empty, err
	}
	base, baseline, err := candidateAcceptanceBase(state.Plan, candidate.CreatedAt)
	if err != nil {
		return empty, err
	}
	if candidate.BasePlanRevisionID != base.ID {
		return empty, fmt.Errorf("base_plan_revision_id: current base is %q, candidate binds %q", base.ID, candidate.BasePlanRevisionID)
	}
	if candidate.BasePlanRevisionHash != base.Hash {
		return empty, fmt.Errorf("base_plan_revision_hash: current base no longer matches candidate")
	}
	computedProposalHash, err := planDefinitionHash(candidate.Proposal.Phases)
	if err != nil {
		return empty, fmt.Errorf("proposal_hash: %w", err)
	}
	if computedProposalHash != candidate.ProposalHash || candidate.Proposal.PlanHash != candidate.ProposalHash {
		return empty, fmt.Errorf("proposal_hash: candidate proposal no longer matches its exact binding")
	}

	activatedPhases, preserved, err := preserveCompletedCandidateWork(state.Plan.Phases, candidate.Proposal.Phases)
	if err != nil {
		return empty, err
	}
	revision := candidate.Proposal
	revision.Phases = activatedPhases
	revision.PreservedPhaseIDs = preserved
	if computed, hashErr := planDefinitionHash(revision.Phases); hashErr != nil || computed != revision.PlanHash {
		if hashErr != nil {
			return empty, fmt.Errorf("proposal_hash: %w", hashErr)
		}
		return empty, fmt.Errorf("proposal_hash: preserving completed work changed immutable proposal identity")
	}
	if err := validateCandidateProposalBase(revision, base); err != nil {
		return empty, err
	}

	acceptedBy := strings.Join(strings.Fields(opts.AcceptedBy), " ")
	if acceptedBy == "" {
		acceptedBy = "owner"
	}
	acceptedAt := opts.AcceptedAt.UTC()
	if acceptedAt.IsZero() {
		acceptedAt = time.Now().UTC()
	}
	receipt, err := newPlanCandidateAcceptanceReceipt(candidate, request.AcceptanceToken, revision, acceptedBy, acceptedAt)
	if err != nil {
		return empty, err
	}
	candidate.Status = colony.PlanCandidateAccepted
	candidate.Proposal = revision
	candidate.Acceptance = &receipt
	if err := validatePlanningRecordHashes(candidate); err != nil {
		return empty, fmt.Errorf("accepted candidate: %w", err)
	}
	if err := candidate.Validate(); err != nil {
		return empty, fmt.Errorf("accepted candidate: %w", err)
	}

	nextStage, _, err := reducePlanningStage(artifact.Stage, planningStageTransition{
		To: planningStageAccepted, AcceptanceReceiptID: receipt.ID, AcceptanceReceiptHash: receipt.ContentHash,
	})
	if err != nil {
		return empty, err
	}
	nextState := state
	nextState.Plan.AcceptancePolicy = colony.PlanAcceptanceExplicitOwner
	nextState.Plan.PendingCandidateID = ""
	nextState.Plan.ActiveRevisionID = revision.ID
	nextState.Plan.Phases = clonePhases(revision.Phases)
	generatedAt := candidate.CreatedAt.UTC()
	nextState.Plan.GeneratedAt = &generatedAt
	confidence := planningConfidenceScores{}
	for _, assessment := range candidate.DimensionAssessments {
		confidence.Set(assessment.Dimension, assessment.After)
	}
	overallConfidence := float64(confidence.Overall) / 100
	nextState.Plan.Confidence = &overallConfidence
	nextState.Plan.EvidencePolicy = colony.PlanEvidenceBoundV1
	nextState.Plan.Revisions = append([]colony.PlanRevision(nil), state.Plan.Revisions...)
	if baseline != nil {
		nextState.Plan.Revisions = append(nextState.Plan.Revisions, *baseline)
	}
	nextState.Plan.Revisions = append(nextState.Plan.Revisions, revision)
	nextState.Plan.Candidates, err = appendAcceptedPlanCandidate(state.Plan.Candidates, candidate)
	if err != nil {
		return empty, err
	}
	nextState.State = colony.StateREADY
	nextState.CurrentPhase = firstBuildablePhase(nextState.Plan.Phases)
	nextState.BuildStartedAt = nil
	nextState.Events = append(trimmedEvents(nextState.Events), fmt.Sprintf("%s|plan_candidate_accepted|plan-candidate-accept|Activated %s from exact candidate %s", acceptedAt.Format(time.RFC3339), revision.ID, candidate.ID))
	if err := validatePlanningState(nextState); err != nil {
		return empty, fmt.Errorf("validate accepted plan state: %w", err)
	}

	stateBytes, err := marshalSpecificationState(nextState)
	if err != nil {
		return empty, err
	}
	candidateBytes, err := marshalPlanningStageJSON(candidate)
	if err != nil {
		return empty, err
	}
	stageBytes, err := marshalPlanningStageJSON(nextStage)
	if err != nil {
		return empty, err
	}
	receiptBytes, err := marshalPlanningStageJSON(receipt)
	if err != nil {
		return empty, err
	}
	repositoryRoot, err := canonicalPlanningTimelineRoot(root)
	if err != nil {
		return empty, err
	}
	tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{
		TransactionID: "plan-candidate-accept-" + candidate.ContentHash[:24],
		Command:       "plan-candidate-accept",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot: repositoryRoot, LifecycleDataRoot: filepath.Join(repositoryRoot, ".aether", "data"),
		},
		Fault: opts.Fault, Rename: opts.Rename,
	})
	if err != nil {
		return empty, err
	}
	for _, target := range []struct {
		path    string
		content []byte
	}{
		{path: "COLONY_STATE.json", content: stateBytes},
		{path: planningStageDataRelativePath(planningRouteCandidateRepositoryPath(candidate.Timeline.RunID)), content: candidateBytes},
		{path: planningStageDataRelativePath(planningStageStateRepositoryPath(candidate.Timeline.RunID)), content: stageBytes},
		{path: planningStageDataRelativePath(planningRouteAcceptanceRepositoryPath(candidate.Timeline.RunID)), content: receiptBytes},
	} {
		if err := tx.DeclareWrite(lifecycleTransactionRootData, target.path, target.content); err != nil {
			return empty, err
		}
	}
	if err := tx.Validate(); err != nil {
		return empty, err
	}
	if _, err := tx.Commit(); err != nil {
		return empty, err
	}
	return planCandidateAcceptanceResult{Candidate: candidate, Revision: revision, Receipt: receipt}, nil
}

func validatePlanCandidateAcceptanceRequest(candidate colony.PlanCandidate, request planCandidateAcceptanceRequest) error {
	bindings := []struct {
		field string
		got   string
		want  string
	}{
		{field: "candidate_id", got: request.CandidateID, want: candidate.ID},
		{field: "specification_revision_id", got: request.SpecificationRevisionID, want: candidate.SpecificationRevisionID},
		{field: "specification_revision_hash", got: request.SpecificationRevisionHash, want: candidate.SpecificationRevisionHash},
		{field: "base_plan_revision_id", got: request.BasePlanRevisionID, want: candidate.BasePlanRevisionID},
		{field: "timeline_digest", got: request.TimelineDigest, want: candidate.Timeline.TimelineDigest},
		{field: "proposal_hash", got: request.ProposalHash, want: candidate.ProposalHash},
		{field: "acceptance_token", got: request.AcceptanceToken, want: planCandidateAcceptanceToken(candidate)},
	}
	for _, binding := range bindings {
		if strings.TrimSpace(binding.got) != binding.want {
			return fmt.Errorf("%s: supplied value does not match candidate %s", binding.field, candidate.ID)
		}
	}
	return nil
}

func validateCandidateRunBase(plan colony.Plan, header planningRunHeader) error {
	stateHash, err := planStateHash(plan)
	if err != nil {
		return fmt.Errorf("base_plan_revision_hash: hash current plan state: %w", err)
	}
	baseID, baseStateHash := planningBaseRevisionIdentity(plan, stateHash)
	if header.BasePlanRevisionID != baseID {
		return fmt.Errorf("base_plan_revision_id: planning run binds %q but current base is %q", header.BasePlanRevisionID, baseID)
	}
	if header.BasePlanRevisionHash != baseStateHash {
		return fmt.Errorf("base_plan_revision_hash: plan state changed after candidate generation")
	}
	return nil
}

func validateCandidateProposalBase(revision colony.PlanRevision, base planCandidateBase) error {
	wantNumber := base.Number + 1
	wantParent := base.ID
	if base.ID == "plan-unbound" {
		wantParent = ""
		wantNumber = 1
	}
	if revision.ParentID != wantParent || revision.Number != wantNumber {
		return fmt.Errorf("base_plan_revision_id: candidate proposal does not extend exact base %q", base.ID)
	}
	return nil
}

func preserveCompletedCandidateWork(previous, proposal []colony.Phase) ([]colony.Phase, []int, error) {
	result := clonePhases(proposal)
	prefix, err := completedPlanPrefix(previous)
	if err != nil {
		return nil, nil, err
	}
	var preserved []int
	bySemanticID := make(map[string]int, len(result))
	for index := range result {
		bySemanticID[strings.TrimSpace(result[index].SemanticID)] = index
	}
	for index := 0; index < prefix; index++ {
		prior := previous[index]
		candidateIndex, ok := bySemanticID[strings.TrimSpace(prior.SemanticID)]
		if !ok || strings.TrimSpace(prior.SemanticID) == "" {
			return nil, nil, fmt.Errorf("proposal_hash: candidate omits completed compatible phase %d", prior.ID)
		}
		priorHash, hashErr := completedPhaseCompatibilityHash(prior)
		if hashErr != nil {
			return nil, nil, hashErr
		}
		candidateHash, hashErr := completedPhaseCompatibilityHash(result[candidateIndex])
		if hashErr != nil {
			return nil, nil, hashErr
		}
		if priorHash != candidateHash {
			return nil, nil, fmt.Errorf("proposal_hash: candidate changes completed phase %d", prior.ID)
		}
		result[candidateIndex].Status = prior.Status
		result[candidateIndex].WatcherFailureCount = prior.WatcherFailureCount
		priorTasks := make(map[string]colony.Task, len(prior.Tasks))
		for _, task := range prior.Tasks {
			priorTasks[strings.TrimSpace(task.SemanticID)] = task
		}
		for taskIndex := range result[candidateIndex].Tasks {
			priorTask, found := priorTasks[strings.TrimSpace(result[candidateIndex].Tasks[taskIndex].SemanticID)]
			if !found {
				return nil, nil, fmt.Errorf("proposal_hash: candidate changes completed phase %d task membership", prior.ID)
			}
			result[candidateIndex].Tasks[taskIndex].Status = priorTask.Status
		}
		preserved = append(preserved, prior.ID)
	}
	return result, preserved, nil
}

func completedPhaseCompatibilityHash(phase colony.Phase) (string, error) {
	copyPhase := clonePhases([]colony.Phase{phase})[0]
	copyPhase.Status = ""
	copyPhase.WatcherFailureCount = 0
	copyPhase.SpecificationRevisionID = ""
	copyPhase.SpecificationRevisionHash = ""
	copyPhase.CandidateID = ""
	copyPhase.CandidateContentHash = ""
	copyPhase.PlanningTimelineID = ""
	copyPhase.PlanningTimelineDigest = ""
	copyPhase.AffectedSemanticIDs = nil
	copyPhase.PreservedSemanticIDs = nil
	for index := range copyPhase.Tasks {
		copyPhase.Tasks[index].Status = ""
		copyPhase.Tasks[index].SpecificationRevisionID = ""
		copyPhase.Tasks[index].SpecificationRevisionHash = ""
		copyPhase.Tasks[index].CandidateID = ""
		copyPhase.Tasks[index].CandidateContentHash = ""
		copyPhase.Tasks[index].PlanningTimelineID = ""
		copyPhase.Tasks[index].PlanningTimelineDigest = ""
		copyPhase.Tasks[index].AffectedSemanticIDs = nil
		copyPhase.Tasks[index].PreservedSemanticIDs = nil
	}
	return jsonSHA256(copyPhase)
}

func newPlanCandidateAcceptanceReceipt(candidate colony.PlanCandidate, token string, revision colony.PlanRevision, acceptedBy string, acceptedAt time.Time) (colony.PlanAcceptanceReceipt, error) {
	tokenHash := strings.TrimPrefix(lifecycleDigest([]byte(strings.TrimSpace(token))), "sha256:")
	receipt := colony.PlanAcceptanceReceipt{
		SchemaVersion: colony.PlanAcceptanceSchemaVersion,
		CandidateID:   candidate.ID, CandidateContentHash: candidate.ContentHash,
		SpecificationRevisionID: candidate.SpecificationRevisionID, SpecificationRevisionHash: candidate.SpecificationRevisionHash,
		BasePlanRevisionID: candidate.BasePlanRevisionID, BasePlanRevisionHash: candidate.BasePlanRevisionHash,
		TimelineID: candidate.Timeline.ID, TimelineDigest: candidate.Timeline.TimelineDigest,
		ProposalHash: candidate.ProposalHash, AcceptanceTokenHash: tokenHash,
		AcceptedBy: acceptedBy, AcceptedAt: acceptedAt.UTC(),
		ActivatedPlanRevisionID: revision.ID, ActivatedPlanRevisionHash: revision.PlanHash,
	}
	hash, err := jsonSHA256(receipt)
	if err != nil {
		return colony.PlanAcceptanceReceipt{}, err
	}
	receipt.ContentHash = hash
	receipt.ID = "plan-acceptance-" + hash[:12]
	if err := receipt.Validate(); err != nil {
		return colony.PlanAcceptanceReceipt{}, err
	}
	if err := validatePlanCandidateAcceptanceReceiptHash(receipt); err != nil {
		return colony.PlanAcceptanceReceipt{}, err
	}
	return receipt, nil
}

func appendAcceptedPlanCandidate(existing []colony.PlanCandidate, accepted colony.PlanCandidate) ([]colony.PlanCandidate, error) {
	result := append([]colony.PlanCandidate(nil), existing...)
	for index := range result {
		if result[index].ID != accepted.ID {
			continue
		}
		if result[index].ContentHash != accepted.ContentHash || result[index].Status != colony.PlanCandidatePendingReview {
			return nil, fmt.Errorf("candidate_id: retained candidate %q conflicts with exact acceptance", accepted.ID)
		}
		pending := accepted
		pending.Status = colony.PlanCandidatePendingReview
		pending.Acceptance = nil
		retainedHash, hashErr := jsonSHA256(result[index])
		if hashErr != nil {
			return nil, hashErr
		}
		pendingHash, hashErr := jsonSHA256(pending)
		if hashErr != nil {
			return nil, hashErr
		}
		if retainedHash != pendingHash {
			return nil, fmt.Errorf("candidate_id: retained candidate %q diverges from exact artifact", accepted.ID)
		}
		result[index] = accepted
		return result, nil
	}
	return append(result, accepted), nil
}

func replayAcceptedPlanCandidate(state colony.ColonyState, candidate colony.PlanCandidate) (planCandidateAcceptanceResult, error) {
	if candidate.Acceptance == nil {
		return planCandidateAcceptanceResult{}, fmt.Errorf("candidate status accepted is missing its original receipt")
	}
	if err := validatePlanCandidateAcceptanceReceiptHash(*candidate.Acceptance); err != nil {
		return planCandidateAcceptanceResult{}, err
	}
	expectedTokenHash := strings.TrimPrefix(lifecycleDigest([]byte(planCandidateAcceptanceToken(candidate))), "sha256:")
	if candidate.Acceptance.AcceptanceTokenHash != expectedTokenHash {
		return planCandidateAcceptanceResult{}, fmt.Errorf("acceptance_token: original receipt does not bind the exact candidate token")
	}
	if err := validatePlanningState(state); err != nil {
		return planCandidateAcceptanceResult{}, fmt.Errorf("candidate_id: retained accepted state is invalid: %w", err)
	}
	for _, retained := range state.Plan.Candidates {
		if retained.ID != candidate.ID {
			continue
		}
		if retained.Status != colony.PlanCandidateAccepted || retained.Acceptance == nil || retained.Acceptance.ID != candidate.Acceptance.ID || retained.Acceptance.ContentHash != candidate.Acceptance.ContentHash {
			return planCandidateAcceptanceResult{}, fmt.Errorf("candidate_id: accepted artifact diverges from retained state")
		}
		retainedHash, hashErr := jsonSHA256(retained)
		if hashErr != nil {
			return planCandidateAcceptanceResult{}, hashErr
		}
		artifactHash, hashErr := jsonSHA256(candidate)
		if hashErr != nil {
			return planCandidateAcceptanceResult{}, hashErr
		}
		if retainedHash != artifactHash {
			return planCandidateAcceptanceResult{}, fmt.Errorf("candidate_id: accepted artifact diverges from retained state")
		}
		for _, revision := range state.Plan.Revisions {
			if revision.ID == candidate.Acceptance.ActivatedPlanRevisionID && revision.PlanHash == candidate.Acceptance.ActivatedPlanRevisionHash {
				return planCandidateAcceptanceResult{Candidate: retained, Revision: revision, Receipt: *retained.Acceptance, Replayed: true}, nil
			}
		}
		return planCandidateAcceptanceResult{}, fmt.Errorf("candidate_id: accepted candidate revision is not retained")
	}
	return planCandidateAcceptanceResult{}, fmt.Errorf("candidate_id: accepted artifact is not retained in state")
}

func validatePlanCandidateAcceptanceReceiptHash(receipt colony.PlanAcceptanceReceipt) error {
	want := receipt.ContentHash
	payload := receipt
	payload.ID = ""
	payload.ContentHash = ""
	got, err := jsonSHA256(payload)
	if err != nil {
		return err
	}
	if got != want || receipt.ID != "plan-acceptance-"+got[:12] {
		return fmt.Errorf("acceptance receipt content hash does not match its immutable payload")
	}
	return nil
}

func planStateHash(plan colony.Plan) (string, error) {
	// Missing and explicit legacy_unbound acceptance policies describe the
	// same executable pre-Phase-200 plan. Canonicalize the additive migration
	// marker out of stale-packet hashes so an upgrade cannot invalidate an
	// otherwise unchanged planning packet.
	canonical := plan
	if canonical.AcceptancePolicy == colony.PlanAcceptanceLegacyUnbound {
		canonical.AcceptancePolicy = ""
	}
	return jsonSHA256(canonical)
}

func activePlanRevisionID(plan colony.Plan) string {
	if id := strings.TrimSpace(plan.ActiveRevisionID); id != "" {
		return id
	}
	if len(plan.Phases) == 0 {
		return ""
	}
	hash, err := planDefinitionHash(plan.Phases)
	if err != nil || len(hash) < 12 {
		return "legacy-plan"
	}
	return "legacy-" + hash[:12]
}

func planDefinitionHash(phases []colony.Phase) (string, error) {
	definitions := clonePhases(phases)
	for i := range definitions {
		definitions[i].Status = ""
		definitions[i].WatcherFailureCount = 0
		for j := range definitions[i].Tasks {
			definitions[i].Tasks[j].Status = ""
		}
	}
	return jsonSHA256(definitions)
}

func normalizePlanRevisionReason(raw string) (colony.PlanRevisionReason, error) {
	reason := colony.PlanRevisionReason(strings.ToLower(strings.TrimSpace(raw)))
	if reason == "" {
		reason = colony.PlanRevisionManual
	}
	if !reason.Valid() || reason == colony.PlanRevisionInitial || reason == colony.PlanRevisionLegacyImport {
		return "", fmt.Errorf("revision type %q is invalid; use manual, user_feedback, research, verification_failure, or scope_change", raw)
	}
	return reason, nil
}

func completedPlanPrefix(phases []colony.Phase) (int, error) {
	prefix := 0
	seenIncomplete := false
	for _, phase := range phases {
		completed := phase.Status == colony.PhaseCompleted
		if completed && seenIncomplete {
			return 0, fmt.Errorf("cannot revise an inconsistent plan: completed phase %d appears after unfinished work", phase.ID)
		}
		if completed {
			prefix++
			continue
		}
		seenIncomplete = true
	}
	return prefix, nil
}

func buildPlanRevisionContext(root string, state colony.ColonyState, opts codexPlanOptions) (*codexPlanRevisionContext, error) {
	if !opts.Refresh {
		if strings.TrimSpace(opts.RevisionReason) != "" || strings.TrimSpace(opts.RevisionType) != "" || len(opts.RevisionEvidence) > 0 {
			return nil, fmt.Errorf("plan revision metadata requires --refresh")
		}
		return nil, nil
	}
	if len(state.Plan.Phases) == 0 {
		// There is nothing to revise, but supplied evidence must not vanish
		// on the way. This branch used to return nil,nil, so on a fresh
		// colony's first plan -- exactly when someone hands over Oracle
		// findings -- the evidence was silently dropped.
		if strings.TrimSpace(opts.RevisionReason) != "" || strings.TrimSpace(opts.RevisionType) != "" || len(opts.RevisionEvidence) > 0 {
			return nil, fmt.Errorf("this colony has no phases yet, so there is nothing to revise; pass research to the first plan with `--research <path>` instead of `--revision-evidence`")
		}
		return nil, nil
	}

	prefix, err := completedPlanPrefix(state.Plan.Phases)
	if err != nil {
		return nil, err
	}
	if prefix == len(state.Plan.Phases) {
		return nil, fmt.Errorf("all plan phases are already completed; seal this colony or start a new goal instead of revising it")
	}
	if state.State == colony.StateEXECUTING {
		if _, attempt, ok := loadLatestBuildAttempt(state.CurrentPhase); ok && buildAttemptStatusActive(attempt.Status) {
			return nil, fmt.Errorf("cannot revise while build attempt %s is active for phase %d; finish or recover that attempt first", attempt.ID, state.CurrentPhase)
		}
	}

	reasonType, err := normalizePlanRevisionReason(opts.RevisionType)
	if err != nil {
		return nil, err
	}
	reason := strings.Join(strings.Fields(strings.TrimSpace(opts.RevisionReason)), " ")
	if prefix > 0 && reason == "" {
		return nil, fmt.Errorf("--revision-reason is required when revising a plan with completed phases")
	}
	if reason == "" {
		reason = "Plan refresh requested before any phase was completed"
	}
	evidence, err := validatePlanRevisionEvidence(root, opts.RevisionEvidence)
	if err != nil {
		return nil, err
	}
	if (reasonType == colony.PlanRevisionResearch || reasonType == colony.PlanRevisionVerificationFailure) && len(evidence) == 0 {
		return nil, fmt.Errorf("revision type %s requires at least one --revision-evidence path", reasonType)
	}
	evidenceHash, err := planRevisionInputEvidenceHash(root, evidence)
	if err != nil {
		return nil, err
	}
	baseHash, err := planStateHash(state.Plan)
	if err != nil {
		return nil, fmt.Errorf("hash current plan before revision: %w", err)
	}
	return &codexPlanRevisionContext{
		BaseRevisionID:    activePlanRevisionID(state.Plan),
		BasePlanStateHash: baseHash,
		ReasonType:        reasonType,
		Reason:            reason,
		Evidence:          evidence,
		EvidenceHash:      evidenceHash,
		CompletedPhases:   clonePhases(state.Plan.Phases[:prefix]),
		SupersededPhases:  clonePhases(state.Plan.Phases[prefix:]),
	}, nil
}

func planRevisionInputEvidenceHash(root string, evidence []string) (string, error) {
	if len(evidence) == 0 {
		return "", nil
	}
	type evidenceFingerprint struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
	}
	fingerprints := make([]evidenceFingerprint, 0, len(evidence))
	for _, rel := range evidence {
		path := filepath.Join(root, filepath.FromSlash(rel))
		hash, err := fileSHA256(path)
		if err != nil {
			return "", fmt.Errorf("hash revision evidence %s: %w", rel, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("stat revision evidence %s: %w", rel, err)
		}
		fingerprints = append(fingerprints, evidenceFingerprint{Path: rel, SHA256: hash, Size: info.Size()})
	}
	hash, err := jsonSHA256(fingerprints)
	if err != nil {
		return "", fmt.Errorf("hash revision evidence manifest: %w", err)
	}
	return hash, nil
}

func validatePlanRevisionEvidence(root string, values []string) ([]string, error) {
	result := uniqueSortedStrings(values)
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repository root for revision evidence: %w", err)
	}
	for _, rel := range result {
		if filepath.IsAbs(rel) {
			return nil, fmt.Errorf("revision evidence must be repository-relative: %s", rel)
		}
		clean := filepath.Clean(filepath.FromSlash(rel))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("revision evidence escapes the repository: %s", rel)
		}
		path := filepath.Join(root, clean)
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("revision evidence %s is unavailable: %w", rel, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("revision evidence must name a file: %s", rel)
		}
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, fmt.Errorf("resolve revision evidence %s: %w", rel, err)
		}
		relToRoot, err := filepath.Rel(realRoot, realPath)
		if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("revision evidence resolves outside the repository: %s", rel)
		}
	}
	return result, nil
}

func validatePlanManifestBase(root string, manifest codexPlanManifest, state colony.ColonyState) error {
	if strings.TrimSpace(manifest.BasePlanStateHash) == "" {
		return fmt.Errorf("plan_manifest base_plan_state_hash is required")
	}
	currentHash, err := planStateHash(state.Plan)
	if err != nil {
		return fmt.Errorf("hash current plan: %w", err)
	}
	if currentHash != strings.TrimSpace(manifest.BasePlanStateHash) {
		return fmt.Errorf("plan state changed after this planning packet was created; discard the stale packet and rerun `aether plan --plan-only`")
	}
	if strings.TrimSpace(manifest.BaseRevisionID) != activePlanRevisionID(state.Plan) {
		return fmt.Errorf("plan revision changed after this planning packet was created; discard the stale packet and rerun `aether plan --plan-only`")
	}
	if manifest.Refresh && len(state.Plan.Phases) > 0 {
		if manifest.Revision == nil {
			return fmt.Errorf("plan_manifest revision context is required when refreshing an existing plan")
		}
		if manifest.Revision.BasePlanStateHash != currentHash || strings.TrimSpace(manifest.Revision.BaseRevisionID) != activePlanRevisionID(state.Plan) {
			return fmt.Errorf("plan_manifest revision context no longer matches the active plan")
		}
		currentEvidenceHash, err := planRevisionInputEvidenceHash(root, manifest.Revision.Evidence)
		if err != nil {
			return err
		}
		if currentEvidenceHash != strings.TrimSpace(manifest.Revision.EvidenceHash) {
			return fmt.Errorf("plan revision evidence changed after this planning packet was created; discard the stale packet and rerun `aether plan --plan-only`")
		}
	}
	return nil
}

func activateGeneratedPlan(previous colony.Plan, candidate []colony.Phase, generatedAt time.Time, confidence *float64, evidencePolicy colony.PlanEvidencePolicy, manifest codexPlanManifest, evidenceHash string) (colony.Plan, colony.PlanRevision, error) {
	if len(candidate) == 0 {
		return colony.Plan{}, colony.PlanRevision{}, fmt.Errorf("generated plan contains no phases")
	}
	if !manifest.Refresh || len(previous.Phases) == 0 {
		phases := renumberRevisionPhases(candidate, 0)
		if err := validateUniquePlanTaskIDs(phases); err != nil {
			return colony.Plan{}, colony.PlanRevision{}, err
		}
		if err := colony.DetectCycles(phases); err != nil {
			return colony.Plan{}, colony.PlanRevision{}, fmt.Errorf("accepted plan dependency validation failed: %w", err)
		}
		plan := colony.Plan{
			GeneratedAt:    &generatedAt,
			Confidence:     confidence,
			EvidencePolicy: evidencePolicy,
			Phases:         phases,
		}
		revision, err := newPlanRevision(plan, 1, "", colony.PlanRevisionInitial, "Initial plan generated", []string{filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json"))}, "", evidenceHash, manifest.PlanningRunID, nil, nil, phaseIDs(phases), generatedAt)
		if err != nil {
			return colony.Plan{}, colony.PlanRevision{}, err
		}
		plan.ActiveRevisionID = revision.ID
		plan.Revisions = []colony.PlanRevision{revision}
		return plan, revision, nil
	}

	prefix, err := completedPlanPrefix(previous.Phases)
	if err != nil {
		return colony.Plan{}, colony.PlanRevision{}, err
	}
	if prefix == len(previous.Phases) {
		return colony.Plan{}, colony.PlanRevision{}, fmt.Errorf("cannot replace a fully completed plan")
	}
	preserved := clonePhases(previous.Phases[:prefix])
	replacements := renumberRevisionPhases(candidate, prefix)
	combined := append(preserved, replacements...)
	if err := validateUniquePlanTaskIDs(combined); err != nil {
		return colony.Plan{}, colony.PlanRevision{}, err
	}
	if err := colony.DetectCycles(combined); err != nil {
		return colony.Plan{}, colony.PlanRevision{}, fmt.Errorf("revised plan dependency validation failed: %w", err)
	}

	revisions := append([]colony.PlanRevision{}, previous.Revisions...)
	parentID := activePlanRevisionID(previous)
	if len(revisions) == 0 {
		baseline, err := newPlanRevision(previous, 1, "", colony.PlanRevisionLegacyImport, "Existing plan imported when revision tracking was enabled", nil, "", "", "", nil, nil, phaseIDs(previous.Phases), revisionCreatedAt(previous, generatedAt))
		if err != nil {
			return colony.Plan{}, colony.PlanRevision{}, err
		}
		revisions = append(revisions, baseline)
		parentID = baseline.ID
	}
	number := nextPlanRevisionNumber(revisions)
	plan := colony.Plan{
		GeneratedAt:      &generatedAt,
		Confidence:       confidence,
		EvidencePolicy:   evidencePolicy,
		ActiveRevisionID: parentID,
		Revisions:        revisions,
		Phases:           combined,
	}
	reasonType := colony.PlanRevisionManual
	reason := "Plan refreshed"
	var evidence []string
	if manifest.Revision != nil {
		reasonType = manifest.Revision.ReasonType
		reason = manifest.Revision.Reason
		evidence = append([]string{}, manifest.Revision.Evidence...)
	}
	inputEvidenceHash := ""
	if manifest.Revision != nil {
		inputEvidenceHash = manifest.Revision.EvidenceHash
	}
	revision, err := newPlanRevision(plan, number, parentID, reasonType, reason, evidence, inputEvidenceHash, evidenceHash, manifest.PlanningRunID, phaseIDs(preserved), phaseIDs(previous.Phases[prefix:]), phaseIDs(replacements), generatedAt)
	if err != nil {
		return colony.Plan{}, colony.PlanRevision{}, err
	}
	plan.ActiveRevisionID = revision.ID
	plan.Revisions = append(plan.Revisions, revision)
	return plan, revision, nil
}

func newPlanRevision(plan colony.Plan, number int, parentID string, reasonType colony.PlanRevisionReason, reason string, evidence []string, inputEvidenceHash, evidenceHash, planningRunID string, preserved, superseded, replacements []int, createdAt time.Time) (colony.PlanRevision, error) {
	hash, err := planDefinitionHash(plan.Phases)
	if err != nil {
		return colony.PlanRevision{}, fmt.Errorf("hash accepted plan revision: %w", err)
	}
	id := fmt.Sprintf("plan-r%d-%s", number, hash[:12])
	return colony.PlanRevision{
		SchemaVersion:       planRevisionSchemaVersion,
		Number:              number,
		ID:                  id,
		ParentID:            strings.TrimSpace(parentID),
		CreatedAt:           createdAt.UTC().Format(time.RFC3339Nano),
		ReasonType:          reasonType,
		Reason:              strings.TrimSpace(reason),
		Evidence:            append([]string{}, evidence...),
		InputEvidenceHash:   strings.TrimSpace(inputEvidenceHash),
		EvidenceHash:        strings.TrimSpace(evidenceHash),
		PlanningRunID:       strings.TrimSpace(planningRunID),
		PlanHash:            hash,
		PreservedPhaseIDs:   append([]int{}, preserved...),
		SupersededPhaseIDs:  append([]int{}, superseded...),
		ReplacementPhaseIDs: append([]int{}, replacements...),
		Phases:              clonePhases(plan.Phases),
	}, nil
}

func renumberRevisionPhases(phases []colony.Phase, offset int) []colony.Phase {
	result := clonePhases(phases)
	for i := range result {
		result[i].ID = offset + i + 1
		result[i].Status = colony.PhasePending
		result[i].WatcherFailureCount = 0
		if i == 0 {
			result[i].Status = colony.PhaseReady
		}
		for j := range result[i].Tasks {
			id := fmt.Sprintf("%d.%d", result[i].ID, j+1)
			result[i].Tasks[j].ID = &id
			result[i].Tasks[j].Status = colony.TaskPending
			for depIndex, dependency := range result[i].Tasks[j].DependsOn {
				parts := strings.Split(strings.TrimSpace(dependency), ".")
				if len(parts) != 2 {
					continue
				}
				phaseID, phaseErr := strconv.Atoi(parts[0])
				taskID, taskErr := strconv.Atoi(parts[1])
				if phaseErr == nil && taskErr == nil {
					result[i].Tasks[j].DependsOn[depIndex] = fmt.Sprintf("%d.%d", phaseID+offset, taskID)
				}
			}
		}
	}
	return result
}

func validateUniquePlanTaskIDs(phases []colony.Phase) error {
	seen := map[string]struct{}{}
	for _, phase := range phases {
		for _, task := range phase.Tasks {
			id := strings.TrimSpace(ptrStr(task.ID))
			if id == "" {
				return fmt.Errorf("phase %d contains a task without an id", phase.ID)
			}
			if _, exists := seen[id]; exists {
				return fmt.Errorf("revised plan duplicates task id %s", id)
			}
			seen[id] = struct{}{}
		}
	}
	return nil
}

func clonePhases(phases []colony.Phase) []colony.Phase {
	if phases == nil {
		return nil
	}
	copyPhases := make([]colony.Phase, len(phases))
	for i, phase := range phases {
		copyPhases[i] = phase
		copyPhases[i].Tasks = cloneTasks(phase.Tasks)
		copyPhases[i].SuccessCriteria = cloneStrings(phase.SuccessCriteria)
		copyPhases[i].EvidenceRequirements = cloneEvidenceRequirements(phase.EvidenceRequirements)
		for j := range copyPhases[i].Tasks {
			task := &copyPhases[i].Tasks[j]
			if task.ID != nil {
				id := *task.ID
				task.ID = &id
			}
			task.Constraints = cloneStrings(task.Constraints)
			task.Hints = cloneStrings(task.Hints)
			task.SuccessCriteria = cloneStrings(task.SuccessCriteria)
			task.EvidenceRequirements = cloneEvidenceRequirements(task.EvidenceRequirements)
			task.DependsOn = cloneStrings(task.DependsOn)
		}
	}
	return copyPhases
}

func cloneTasks(tasks []colony.Task) []colony.Task {
	if tasks == nil {
		return nil
	}
	return append([]colony.Task(nil), tasks...)
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func cloneEvidenceRequirements(requirements []colony.CriterionEvidenceRequirement) []colony.CriterionEvidenceRequirement {
	if requirements == nil {
		return nil
	}
	cloned := append([]colony.CriterionEvidenceRequirement(nil), requirements...)
	for i := range cloned {
		cloned[i].Artifacts = cloneStrings(requirements[i].Artifacts)
		cloned[i].Checks = cloneStrings(requirements[i].Checks)
	}
	return cloned
}

func phaseIDs(phases []colony.Phase) []int {
	ids := make([]int, 0, len(phases))
	for _, phase := range phases {
		ids = append(ids, phase.ID)
	}
	return ids
}

func nextPlanRevisionNumber(revisions []colony.PlanRevision) int {
	max := 0
	for _, revision := range revisions {
		if revision.Number > max {
			max = revision.Number
		}
	}
	return max + 1
}

func revisionCreatedAt(plan colony.Plan, fallback time.Time) time.Time {
	if plan.GeneratedAt != nil {
		return plan.GeneratedAt.UTC()
	}
	return fallback.UTC()
}

func activePlanRevision(plan colony.Plan) (colony.PlanRevision, bool) {
	id := strings.TrimSpace(plan.ActiveRevisionID)
	for _, revision := range plan.Revisions {
		if revision.ID == id {
			return revision, true
		}
	}
	return colony.PlanRevision{}, false
}

func planRevisionSummary(plan colony.Plan) map[string]interface{} {
	if revision, ok := activePlanRevision(plan); ok {
		return map[string]interface{}{
			"id":                    revision.ID,
			"number":                revision.Number,
			"reason_type":           revision.ReasonType,
			"reason":                revision.Reason,
			"evidence":              append([]string{}, revision.Evidence...),
			"created_at":            revision.CreatedAt,
			"parent_id":             revision.ParentID,
			"preserved_phase_ids":   append([]int{}, revision.PreservedPhaseIDs...),
			"superseded_phase_ids":  append([]int{}, revision.SupersededPhaseIDs...),
			"replacement_phase_ids": append([]int{}, revision.ReplacementPhaseIDs...),
		}
	}
	if len(plan.Phases) > 0 {
		return map[string]interface{}{
			"id":          activePlanRevisionID(plan),
			"number":      0,
			"reason_type": colony.PlanRevisionLegacyImport,
			"reason":      "Legacy plan has not yet been revised",
		}
	}
	return map[string]interface{}{}
}

func planRevisionCapsuleLine(plan colony.Plan) string {
	if revision, ok := activePlanRevision(plan); ok {
		return fmt.Sprintf("Plan revision: %s (r%d, %s) - %s\n", revision.ID, revision.Number, revision.ReasonType, revision.Reason)
	}
	if len(plan.Phases) > 0 {
		return fmt.Sprintf("Plan revision: %s (legacy plan)\n", activePlanRevisionID(plan))
	}
	return ""
}

func renderPlanRevisionWorkerAppendix(root string, revision *codexPlanRevisionContext) string {
	if revision == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n## Accepted Plan Revision Boundary\n")
	b.WriteString(fmt.Sprintf("- Revision reason: %s: %s\n", revision.ReasonType, revision.Reason))
	if len(revision.Evidence) > 0 {
		b.WriteString("- Read these revision evidence files before proposing replacement work: " + strings.Join(revision.Evidence, ", ") + "\n")
		// The evidence CONTENTS travel in the brief. A path string alone made
		// Oracle→plan human-mediated: findings written to a file never reached
		// the Route-Setter unless someone pasted them by hand.
		b.WriteString(renderRevisionEvidenceExcerpts(root, revision.Evidence))
	}
	if len(revision.CompletedPhases) > 0 {
		b.WriteString("- Completed phases are immutable and must not appear in phase-plan.json:\n")
		for _, phase := range revision.CompletedPhases {
			b.WriteString(fmt.Sprintf("  - Phase %d: %s\n", phase.ID, phase.Name))
		}
	}
	if len(revision.SupersededPhases) > 0 {
		b.WriteString("- Replace only this unfinished suffix:\n")
		for _, phase := range revision.SupersededPhases {
			b.WriteString(fmt.Sprintf("  - Phase %d: %s (%s)\n", phase.ID, phase.Name, phase.Status))
		}
	}
	b.WriteString("- phase-plan.json must contain only replacement unfinished phases. Aether will preserve completed phases and assign final phase/task IDs atomically.\n")
	b.WriteString("- Do not repeat completed task goals. Dependencies inside the replacement suffix use local artifact IDs starting at 1.1; Aether offsets them after the immutable prefix.\n")
	return b.String()
}

func planRevisionCLIArgs(revision *codexPlanRevisionContext) string {
	if revision == nil {
		return ""
	}
	parts := []string{
		"--refresh",
		"--revision-type", shellQuotePlanArg(string(revision.ReasonType)),
		"--revision-reason", shellQuotePlanArg(revision.Reason),
	}
	for _, evidence := range revision.Evidence {
		parts = append(parts, "--revision-evidence", shellQuotePlanArg(evidence))
	}
	return strings.Join(parts, " ")
}

func planRevisionRecommendation(reasonType colony.PlanRevisionReason, reason string, evidence ...string) map[string]interface{} {
	context := &codexPlanRevisionContext{
		ReasonType: reasonType,
		Reason:     strings.Join(strings.Fields(strings.TrimSpace(reason)), " "),
		Evidence:   uniqueSortedStrings(evidence),
	}
	return map[string]interface{}{
		"automatic":   false,
		"reason_type": context.ReasonType,
		"reason":      context.Reason,
		"evidence":    append([]string{}, context.Evidence...),
		"command":     "aether plan " + planRevisionCLIArgs(context),
	}
}

func planRevisionContextsEqual(left, right *codexPlanRevisionContext) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	if left.BaseRevisionID != right.BaseRevisionID ||
		left.BasePlanStateHash != right.BasePlanStateHash ||
		left.ReasonType != right.ReasonType ||
		left.Reason != right.Reason ||
		left.EvidenceHash != right.EvidenceHash {
		return false
	}
	leftEvidence := sortedPlanRevisionEvidence(left.Evidence)
	rightEvidence := sortedPlanRevisionEvidence(right.Evidence)
	if len(leftEvidence) != len(rightEvidence) {
		return false
	}
	for i := range leftEvidence {
		if leftEvidence[i] != rightEvidence[i] {
			return false
		}
	}
	return true
}

func shellQuotePlanArg(value string) string {
	return strconv.Quote(strings.TrimSpace(value))
}

func sortedPlanRevisionEvidence(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	return result
}

// revisionEvidenceExcerptBudget bounds total evidence content carried in a
// planning brief; revisionEvidencePerFileBudget bounds each file's share.
const (
	revisionEvidenceExcerptBudget = 6000
	revisionEvidencePerFileBudget = 2500
)

// renderRevisionEvidenceExcerpts reads each evidence file and inlines a
// bounded excerpt so the Route-Setter plans from the findings themselves, not
// a filename. Unreadable files degrade to their path (already listed above).
func renderRevisionEvidenceExcerpts(root string, evidence []string) string {
	var b strings.Builder
	remaining := revisionEvidenceExcerptBudget
	for _, rel := range evidence {
		if remaining <= 0 {
			b.WriteString(fmt.Sprintf("\n### Evidence: %s\n_(excerpt budget exhausted — read the file directly)_\n", rel))
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}
		limit := revisionEvidencePerFileBudget
		if limit > remaining {
			limit = remaining
		}
		truncated := false
		if len(content) > limit {
			cut := content[:limit]
			if idx := strings.LastIndex(cut, "\n\n"); idx > limit/2 {
				cut = cut[:idx]
			}
			content = cut
			truncated = true
		}
		remaining -= len(content)
		b.WriteString(fmt.Sprintf("\n### Evidence: %s\n\n%s\n", rel, content))
		if truncated {
			b.WriteString(fmt.Sprintf("_(truncated — full evidence: %s)_\n", rel))
		}
	}
	return b.String()
}
