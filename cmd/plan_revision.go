package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
	Refusal   *planCandidateRefusalDetails `json:"refusal,omitempty"`
}

type phaseInsertCandidateRequest struct {
	After                             int
	Name                              string
	Description                       string
	Constraints                       string
	SpecificationItemID               string
	SpecificationRevisionPrerequisite string
	ExpectedBasePlanRevisionID        string
	CreatedAt                         time.Time
}

type phaseInsertCandidateResult struct {
	Candidate     colony.PlanCandidate
	InsertedPhase colony.Phase
	ReviewCommand string
	AcceptCommand string
}

type phaseInsertProofLinks struct {
	Requirements []string
	Acceptance   []string
	Negative     []string
	Recovery     []string
	PublicPaths  []string
}

func planningRouteAcceptanceRepositoryPath(runID string) string {
	return filepath.ToSlash(filepath.Join(".aether", "data", "planning", strings.TrimSpace(runID), "acceptance.json"))
}

// createPhaseInsertCandidate converts an owner-requested corrective phase into
// the same immutable, reviewable candidate consumed by acceptPlanCandidate.
// It deliberately leaves COLONY_STATE.json untouched: only exact later owner
// acceptance may replace the active revision.
func createPhaseInsertCandidate(root string, request phaseInsertCandidateRequest) (phaseInsertCandidateResult, error) {
	var result phaseInsertCandidateResult
	err := withPlanningMutationSession(root, "phase-insert-candidate", func(session *planningMutationSession) error {
		var insertErr error
		result, insertErr = createPhaseInsertCandidateInSession(session, request)
		return insertErr
	})
	return result, err
}

func createPhaseInsertCandidateInSession(session *planningMutationSession, request phaseInsertCandidateRequest) (phaseInsertCandidateResult, error) {
	empty := phaseInsertCandidateResult{}
	state, err := loadSpecificationColonyStateInSession(session)
	if err != nil {
		return empty, err
	}
	if err := validatePlanningState(state); err != nil {
		return empty, fmt.Errorf("validate current planning state: %w", err)
	}
	if state.Plan.AcceptancePolicy != colony.PlanAcceptanceExplicitOwner {
		return empty, fmt.Errorf("immutable phase insertion requires an explicitly accepted current plan")
	}
	active, ok := activePlanRevision(state.Plan)
	if !ok {
		return empty, fmt.Errorf("base plan revision is unavailable")
	}
	if expected := strings.TrimSpace(request.ExpectedBasePlanRevisionID); expected != "" && expected != active.ID {
		return empty, fmt.Errorf("base plan revision is stale: current is %q, request binds %q", active.ID, expected)
	}
	if state.State == colony.StateEXECUTING {
		return empty, fmt.Errorf("cannot insert a phase while an active execution attempt may be mutating the plan")
	}
	if _, attempt, found := loadRelevantBuildAttempt(state); found && buildAttemptStatusActive(attempt.Status) {
		return empty, fmt.Errorf("cannot insert a phase while build attempt %s is active for phase %d", attempt.ID, attempt.Phase)
	}
	if strings.TrimSpace(state.Plan.PendingCandidateID) != "" {
		return empty, fmt.Errorf("cannot insert a phase while candidate %s is already pending review", state.Plan.PendingCandidateID)
	}
	if artifact, loadErr := loadPlanCandidateArtifactInSession(session, ""); loadErr == nil {
		return empty, fmt.Errorf("cannot insert a phase while candidate %s is already pending review", artifact.Candidate.ID)
	} else if !strings.Contains(loadErr.Error(), "no reviewable plan candidate found") {
		return empty, fmt.Errorf("inspect pending plan candidates: %w", loadErr)
	}
	if request.After < 0 || request.After > len(state.Plan.Phases) {
		return empty, fmt.Errorf("invalid after index %d (plan has %d phases)", request.After, len(state.Plan.Phases))
	}

	approved, err := phaseInsertApprovedSpecification(state)
	if err != nil {
		return empty, err
	}
	coverageIDs, impact, err := phaseInsertSpecificationCoverage(state, approved.Revision, request)
	if err != nil {
		return empty, err
	}
	proofs, err := phaseInsertProofCoverage(approved.Revision, strings.TrimSpace(request.SpecificationItemID))
	if err != nil {
		return empty, err
	}

	createdAt := request.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	phaseSeed, err := jsonSHA256(struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Constraints string   `json:"constraints,omitempty"`
		Coverage    []string `json:"coverage"`
	}{canonicalPlanningText(request.Name), canonicalPlanningText(request.Description), canonicalPlanningText(request.Constraints), coverageIDs})
	if err != nil {
		return empty, fmt.Errorf("hash inserted phase identity: %w", err)
	}
	insertedSemanticID := "phase-insert-" + phaseSeed[:16]
	for _, phase := range state.Plan.Phases {
		if strings.TrimSpace(phase.SemanticID) == insertedSemanticID {
			return empty, fmt.Errorf("phase insertion %q already exists in the active plan", insertedSemanticID)
		}
	}
	newPhase := colony.Phase{
		SemanticID:            insertedSemanticID,
		Name:                  strings.TrimSpace(request.Name),
		Description:           strings.TrimSpace(request.Description),
		Status:                colony.PhasePending,
		Tasks:                 []colony.Task{},
		RequirementProofLinks: proofs.Requirements,
		AcceptanceProofLinks:  proofs.Acceptance,
		NegativeProofLinks:    proofs.Negative,
		RecoveryProofLinks:    proofs.Recovery,
		PublicPathProofLinks:  proofs.PublicPaths,
		AffectedSemanticIDs:   append([]string(nil), coverageIDs...),
	}
	proposalInput := clonePhases(state.Plan.Phases)
	proposalInput = append(proposalInput[:request.After], append([]colony.Phase{newPhase}, proposalInput[request.After:]...)...)
	proposalInput = renumberRevisionPhases(proposalInput, 0)

	proposalSkeleton := active
	proposalSkeleton.Phases = clonePhases(proposalInput)
	beforeSnapshot, err := buildPlanningSemanticSnapshot(planningSemanticSnapshotSource{
		CurrentSchema: true, Plan: state.Plan, Revision: &active, Specification: state.Specification,
	})
	if err != nil {
		return empty, fmt.Errorf("snapshot active plan before insertion: %w", err)
	}
	afterSnapshot, err := buildPlanningSemanticSnapshot(planningSemanticSnapshotSource{
		CurrentSchema: true, Plan: state.Plan, Revision: &proposalSkeleton, Specification: state.Specification,
	})
	if err != nil {
		return empty, fmt.Errorf("snapshot phase insertion proposal: %w", err)
	}
	delta, err := comparePlanningSemanticSnapshots(beforeSnapshot, afterSnapshot)
	if err != nil {
		return empty, err
	}
	evidenceHash, err := jsonSHA256(struct {
		BaseRevisionID  string   `json:"base_revision_id"`
		PhaseSemanticID string   `json:"phase_semantic_id"`
		Coverage        []string `json:"coverage"`
		RequestedAt     string   `json:"requested_at"`
	}{active.ID, insertedSemanticID, coverageIDs, createdAt.Format(time.RFC3339Nano)})
	if err != nil {
		return empty, fmt.Errorf("hash phase insertion evidence: %w", err)
	}
	evidenceID := "phase-insert-evidence-" + evidenceHash[:12]
	if err := phaseInsertApplyDeltaEvidence(&delta, evidenceID); err != nil {
		return empty, err
	}
	if len(impact.AffectedSemanticIDs) > 0 {
		if err := phaseInsertAppendImpactAuthority(&delta, approved.Revision.ID, impact.AffectedSemanticIDs); err != nil {
			return empty, err
		}
	}
	if err := addressPhaseInsertDelta(&delta); err != nil {
		return empty, err
	}

	score := phaseInsertConfidenceScore(state.Plan.Confidence)
	assessments, gaps, err := phaseInsertAssessments(score, evidenceID)
	if err != nil {
		return empty, err
	}
	decision, err := phaseInsertStopDecision(gaps, evidenceID)
	if err != nil {
		return empty, err
	}
	baseStateHash, err := planStateHash(state.Plan)
	if err != nil {
		return empty, fmt.Errorf("hash current plan state: %w", err)
	}
	base, _, err := candidateAcceptanceBase(state.Plan, createdAt)
	if err != nil {
		return empty, err
	}
	runSeed, err := jsonSHA256(struct {
		BaseID        string `json:"base_id"`
		BaseStateHash string `json:"base_state_hash"`
		PhaseID       string `json:"phase_id"`
		CreatedAt     string `json:"created_at"`
	}{active.ID, baseStateHash, insertedSemanticID, createdAt.Format(time.RFC3339Nano)})
	if err != nil {
		return empty, err
	}
	runID := "phase-insert-" + runSeed[:20]
	card := colony.PlanningIterationCard{
		SchemaVersion: colony.PlanningIterationSchemaVersion,
		RunID:         runID, Iteration: 1,
		ScoutReceiptID: "phase-insert-request-" + evidenceHash[:12], ScoutReceiptHash: evidenceHash,
		RouteSetterReceiptID: "phase-insert-proposal-" + delta.ContentHash[:12], RouteSetterReceiptHash: delta.ContentHash,
		EvidenceIDs: []string{evidenceID}, DimensionAssessments: assessments,
		WeakestGap: gaps[0], SemanticDelta: delta, Decision: decision,
		EvidenceThatWouldChange: decision.EvidenceThatWouldChange, CreatedAt: createdAt,
	}
	card, _, err = canonicalPlanningTimelineCard(card)
	if err != nil {
		return empty, err
	}
	timeline, err := phaseInsertTimelinePreview(card)
	if err != nil {
		return empty, err
	}

	affected, preserved := planningRouteDeltaSemanticIDs(delta)
	affected = canonicalPlanImpactIDs(append(affected, impact.AffectedSemanticIDs...))
	preserved = planImpactDifference(preserved, affected)
	boundPhases := phaseInsertCandidatePhases(proposalInput, "", "", approved.Binding, timeline, affected, preserved, coverageIDs)
	requirements, acceptance, negative, recovery, publicPaths := planningRouteProposalProofLinks(boundPhases)
	preservedPhaseIDs, supersededPhaseIDs := phaseInsertRevisionPhaseSets(state.Plan.Phases, impact)
	proposal := colony.PlanRevision{
		SchemaVersion: planRevisionSchemaVersion, Number: base.Number + 1,
		ParentID:  base.ID,
		CreatedAt: createdAt.Format(time.RFC3339Nano), ReasonType: colony.PlanRevisionScopeChange,
		Reason:       "Owner-requested corrective phase insertion: " + strings.TrimSpace(request.Name),
		EvidenceHash: evidenceHash, PlanningRunID: runID,
		PreservedPhaseIDs: preservedPhaseIDs, SupersededPhaseIDs: supersededPhaseIDs,
		ReplacementPhaseIDs: []int{boundPhases[request.After].ID}, SemanticID: active.SemanticID,
		RequirementProofLinks: requirements, AcceptanceProofLinks: acceptance,
		NegativeProofLinks: negative, RecoveryProofLinks: recovery, PublicPathProofLinks: publicPaths,
		SpecificationRevisionID: approved.Revision.ID, SpecificationRevisionHash: approved.Revision.ContentHash,
		PlanningTimelineID: timeline.ID, PlanningTimelineDigest: timeline.TimelineDigest,
		AffectedSemanticIDs: affected, PreservedSemanticIDs: preserved, Phases: boundPhases,
	}
	recommendation, err := phaseInsertRecommendation("", evidenceID, createdAt)
	if err != nil {
		return empty, err
	}
	candidate := colony.PlanCandidate{
		SchemaVersion: colony.PlanCandidateSchemaVersion,
		Status:        colony.PlanCandidatePendingReview, CreatedAt: createdAt, ExpiresAt: createdAt.Add(7 * 24 * time.Hour),
		Proposal: proposal, BasePlanRevisionID: base.ID, BasePlanRevisionHash: base.Hash,
		SpecificationRevisionID: approved.Revision.ID, SpecificationRevisionHash: approved.Revision.ContentHash,
		Timeline: timeline, StopDecision: decision, DimensionAssessments: assessments, SemanticDelta: delta,
		ResidualGaps: gaps, EvidenceThatWouldChange: decision.EvidenceThatWouldChange, Recommendation: recommendation,
	}
	if err := addressPlanCandidateReviewPayload(&candidate); err != nil {
		return empty, err
	}
	if err := validatePlanCandidateImpactCoverage(candidate, impact); err != nil {
		return empty, fmt.Errorf("affected_scope: %w", err)
	}
	if _, _, err := preserveCompletedCandidateWorkForImpact(state.Plan.Phases, candidate.Proposal.Phases, impact); err != nil {
		return empty, err
	}
	if err := validatePlanningRecordHashes(candidate); err != nil {
		return empty, err
	}
	if err := candidate.Validate(); err != nil {
		return empty, err
	}

	manifest, header, stage, err := phaseInsertPlanningBoundary(state, approved, active.ID, baseStateHash, runID, evidenceID, evidenceHash, gaps[0], card, score, createdAt)
	if err != nil {
		return empty, err
	}
	cardPath := planningTimelineCardRepositoryPath(runID, card.Iteration, card.ID)
	appendRequestDigest, err := planningTimelineAppendRequestDigest(card.RouteSetterReceiptID, card, "")
	if err != nil {
		return empty, err
	}
	timelineTransactionID, err := planningTimelineTransactionID(runID, card.RouteSetterReceiptID)
	if err != nil {
		return empty, err
	}
	timelineIndex := planningTimelineIndex{
		SchemaVersion: planningTimelineIndexSchemaVersion,
		RunID:         runID,
		Entries: []planningTimelineIndexEntry{{
			Iteration: card.Iteration, CardID: card.ID, CardHash: card.ContentHash,
			CardPath: cardPath, AppendReceiptID: card.RouteSetterReceiptID,
			AppendRequestDigest: appendRequestDigest, TransactionID: timelineTransactionID,
		}},
	}
	if err := addressPlanningTimelineIndex(&timelineIndex, []colony.PlanningIterationCard{card}); err != nil {
		return empty, err
	}
	if timelineIndex.TimelineDigest != timeline.TimelineDigest {
		return empty, fmt.Errorf("phase insertion timeline preview does not match its canonical index")
	}
	cardBytes, err := marshalPlanningTimelineJSON(card)
	if err != nil {
		return empty, err
	}
	indexBytes, err := marshalPlanningTimelineJSON(timelineIndex)
	if err != nil {
		return empty, err
	}
	if err := validatePlanningTimelineCardBytes(cardBytes, card); err != nil {
		return empty, err
	}
	if err := validatePlanningTimelineIndexBytes(indexBytes, timelineIndex, []colony.PlanningIterationCard{card}); err != nil {
		return empty, err
	}
	files := map[string][]byte{
		cardPath: cardBytes,
		planningTimelineIndexRepositoryPath(runID): indexBytes,
	}
	for path, value := range map[string]interface{}{
		planningRouteCandidateRepositoryPath(runID):                                              candidate,
		planningStageManifestRepositoryPath(runID, manifest.ID):                                  manifest,
		planningStageStateRepositoryPath(runID):                                                  stage,
		filepath.ToSlash(filepath.Join(".aether", "data", "planning", runID, "run-header.json")): header,
	} {
		content, marshalErr := marshalPlanningStageJSON(value)
		if marshalErr != nil {
			return empty, marshalErr
		}
		files[path] = content
	}
	if err := persistPlanningScoutFilesInSession(session, "phase-insert-candidate-"+candidate.ContentHash[:24], "phase-insert-candidate", candidate.ID, files, nil); err != nil {
		return empty, err
	}
	acceptRequest := planCandidateAcceptanceRequest{
		CandidateID: candidate.ID, SpecificationRevisionID: candidate.SpecificationRevisionID,
		SpecificationRevisionHash: candidate.SpecificationRevisionHash, BasePlanRevisionID: candidate.BasePlanRevisionID,
		TimelineDigest: candidate.Timeline.TimelineDigest, ProposalHash: candidate.ProposalHash,
		AcceptanceToken: planCandidateAcceptanceToken(candidate),
	}
	return phaseInsertCandidateResult{
		Candidate: candidate, InsertedPhase: boundPhases[request.After], ReviewCommand: "aether plan --candidate",
		AcceptCommand: planCandidateAcceptanceCommand(acceptRequest),
	}, nil
}

func phaseInsertApprovedSpecification(state colony.ColonyState) (approvedPlanningSpecification, error) {
	if state.Specification == nil {
		return approvedPlanningSpecification{}, fmt.Errorf("specification coverage requires a current approved specification")
	}
	if err := validateSpecificationState(*state.Specification); err != nil {
		return approvedPlanningSpecification{}, fmt.Errorf("specification coverage state is invalid: %w", err)
	}
	revision, ok := currentSpecificationRevision(*state.Specification)
	if !ok || revision.Status != colony.SpecStatusApproved || revision.Approval == nil {
		return approvedPlanningSpecification{}, fmt.Errorf("specification coverage requires a current approved specification revision")
	}
	if revision.Approval.RevisionID != revision.ID || revision.Approval.RevisionContentHash != revision.ContentHash {
		return approvedPlanningSpecification{}, fmt.Errorf("specification coverage approval does not bind the current revision")
	}
	if state.SessionID != nil && strings.TrimSpace(*state.SessionID) != "" && strings.TrimSpace(revision.Scope.SessionID) != strings.TrimSpace(*state.SessionID) {
		return approvedPlanningSpecification{}, fmt.Errorf("specification coverage belongs to a different colony session")
	}
	approvalHash, err := jsonSHA256(*revision.Approval)
	if err != nil {
		return approvedPlanningSpecification{}, err
	}
	return approvedPlanningSpecification{
		Specification: *state.Specification, Revision: revision,
		Binding: planningStageSpecificationBinding{
			RevisionID: revision.ID, ContentHash: revision.ContentHash,
			PredecessorRevisionID: revision.PredecessorID, Status: revision.Status,
			ApprovalReceiptID: revision.Approval.ID, ApprovalReceiptHash: approvalHash,
		},
		GoalID: state.Specification.GoalID, SessionID: revision.Scope.SessionID,
	}, nil
}

func phaseInsertSpecificationCoverage(state colony.ColonyState, revision colony.SpecRevision, request phaseInsertCandidateRequest) ([]string, planImpactClosure, error) {
	itemID := strings.TrimSpace(request.SpecificationItemID)
	prerequisite := strings.TrimSpace(request.SpecificationRevisionPrerequisite)
	if itemID == "" && prerequisite == "" {
		return nil, planImpactClosure{}, fmt.Errorf("specification coverage is required: pass --spec-item <approved-id> or --spec-revision <approved-successor-id>")
	}
	known := planImpactIDSet(planImpactSpecificationIDs(revision))
	if itemID != "" {
		if _, ok := known[itemID]; !ok {
			return nil, planImpactClosure{}, fmt.Errorf("specification coverage item %q is absent from approved revision %s", itemID, revision.ID)
		}
	}
	impact, unresolved, err := unresolvedPlanImpact(state)
	if err != nil {
		return nil, planImpactClosure{}, fmt.Errorf("specification coverage impact: %w", err)
	}
	if prerequisite != "" {
		if prerequisite != revision.ID {
			return nil, planImpactClosure{}, fmt.Errorf("specification revision prerequisite is stale: current approved revision is %q", revision.ID)
		}
		if !unresolved {
			return nil, planImpactClosure{}, fmt.Errorf("specification revision prerequisite %q has no unreconciled affected scope; pass --spec-item for existing approved scope", prerequisite)
		}
	}
	coverage := []string{}
	if itemID != "" {
		coverage = append(coverage, itemID)
	}
	if prerequisite != "" {
		coverage = append(coverage, impact.ChangedSpecItemIDs...)
	}
	return canonicalPlanImpactIDs(coverage), impact, nil
}

func phaseInsertProofCoverage(revision colony.SpecRevision, itemID string) (phaseInsertProofLinks, error) {
	links := phaseInsertProofLinks{}
	if len(revision.Requirements) == 0 || len(revision.AcceptanceChecks) == 0 || len(revision.NegativeExpectations) == 0 || len(revision.RecoveryExpectations) == 0 || len(revision.AffectedPublicPaths) == 0 {
		return links, fmt.Errorf("approved specification cannot cover a current phase because one or more proof categories are empty")
	}
	links.Requirements = []string{revision.Requirements[0].ID}
	links.Acceptance = []string{revision.AcceptanceChecks[0].ID}
	links.Negative = []string{revision.NegativeExpectations[0].ID}
	links.Recovery = []string{revision.RecoveryExpectations[0].ID}
	links.PublicPaths = []string{revision.AffectedPublicPaths[0].ID}
	appendIfCategory := func(destination *[]string, ids []string) {
		for _, id := range ids {
			if id == itemID {
				*destination = canonicalPlanImpactIDs(append(*destination, itemID))
				return
			}
		}
	}
	appendIfCategory(&links.Requirements, specRequirementIDList(revision.Requirements))
	appendIfCategory(&links.Acceptance, specAcceptanceCheckIDList(revision.AcceptanceChecks))
	appendIfCategory(&links.Negative, specNegativeExpectationIDList(revision.NegativeExpectations))
	appendIfCategory(&links.Recovery, specRecoveryExpectationIDList(revision.RecoveryExpectations))
	appendIfCategory(&links.PublicPaths, specPublicPathIDList(revision.AffectedPublicPaths))
	return links, nil
}

func specRequirementIDList(values []colony.SpecRequirement) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ID
	}
	return result
}

func specAcceptanceCheckIDList(values []colony.SpecAcceptanceCheck) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ID
	}
	return result
}

func specNegativeExpectationIDList(values []colony.SpecNegativeExpectation) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ID
	}
	return result
}

func specRecoveryExpectationIDList(values []colony.SpecRecoveryExpectation) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ID
	}
	return result
}

func specPublicPathIDList(values []colony.SpecPublicPath) []string {
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].ID
	}
	return result
}

func phaseInsertApplyDeltaEvidence(delta *colony.PlanningSemanticDelta, evidenceID string) error {
	if delta == nil {
		return fmt.Errorf("phase insertion semantic delta is required")
	}
	for _, section := range []struct {
		name    colony.PlanningSemanticSection
		changes *[]colony.PlanningSemanticChange
	}{
		{name: colony.PlanningSemanticSectionPhases, changes: &delta.Phases},
		{name: colony.PlanningSemanticSectionTasks, changes: &delta.Tasks},
		{name: colony.PlanningSemanticSectionDependencies, changes: &delta.Dependencies},
		{name: colony.PlanningSemanticSectionRequirementLinks, changes: &delta.RequirementLinks},
		{name: colony.PlanningSemanticSectionAcceptanceChecks, changes: &delta.AcceptanceChecks},
		{name: colony.PlanningSemanticSectionNegativeExpectations, changes: &delta.NegativeExpectations},
		{name: colony.PlanningSemanticSectionRecoveryExpectations, changes: &delta.RecoveryExpectations},
		{name: colony.PlanningSemanticSectionPublicPaths, changes: &delta.PublicPaths},
	} {
		for index := range *section.changes {
			change := &(*section.changes)[index]
			if change.Kind != colony.PlanningSemanticChangePreserved {
				change.EvidenceIDs = []string{evidenceID}
			}
			if err := colony.AddressPlanningSemanticChange(section.name, change); err != nil {
				return fmt.Errorf("phase insertion semantic change %q: %w", change.SemanticID, err)
			}
		}
	}
	return nil
}

func phaseInsertAppendImpactAuthority(delta *colony.PlanningSemanticDelta, specificationID string, affected []string) error {
	if delta == nil || len(affected) == 0 {
		return nil
	}
	rationale := "The approved specification successor requires this exact affected closure to be reconciled by the insertion candidate"
	impact := colony.PlanningAuthorityImpact{
		Kind: colony.PlanningAuthoritySpecSupersession, SourceID: specificationID,
		AffectedSemanticIDs: canonicalPlanImpactIDs(affected), Rationale: rationale,
	}
	if err := colony.AddressPlanningAuthorityImpact(&impact); err != nil {
		return err
	}
	delta.AuthorityImpacts = append(delta.AuthorityImpacts, impact)
	return nil
}

func addressPhaseInsertDelta(delta *colony.PlanningSemanticDelta) error {
	if delta == nil {
		return fmt.Errorf("phase insertion semantic delta is required")
	}
	return colony.AddressPlanningSemanticDelta(delta)
}

func phaseInsertConfidenceScore(confidence *float64) int {
	if confidence == nil {
		return 50
	}
	score := int(*confidence * 100)
	if *confidence > 1 {
		score = int(*confidence)
	}
	if score < 1 {
		return 1
	}
	if score > 100 {
		return 100
	}
	return score
}

func phaseInsertAssessments(score int, evidenceID string) ([]colony.PlanningDimensionAssessment, []colony.PlanningGap, error) {
	assessments := make([]colony.PlanningDimensionAssessment, 0, len(colony.PlanningDimensions()))
	gaps := make([]colony.PlanningGap, 0, len(colony.PlanningDimensions()))
	for _, dimension := range colony.PlanningDimensions() {
		gap := colony.PlanningGap{
			SchemaVersion: colony.PlanningSchemaVersion, Dimension: dimension,
			Materiality: colony.PlanningGapNonMaterial, Severity: 0,
			Description:             "Manual insertion makes no new " + string(dimension) + " readiness claim beyond the accepted base plan.",
			EvidenceIDs:             []string{evidenceID},
			EvidenceThatWouldChange: "Owner rejection, a newer approved specification, or a changed base plan requires a new insertion candidate.",
		}
		if err := colony.AddressPlanningGap(&gap); err != nil {
			return nil, nil, err
		}
		assessment := colony.PlanningDimensionAssessment{
			SchemaVersion: colony.PlanningSchemaVersion, Dimension: dimension,
			Before: score, After: score, FreshEvidenceIDs: []string{evidenceID}, RemainingGap: gap,
			Rationale:         "The owner-requested insertion is isolated as a proposal; readiness remains unchanged until exact acceptance.",
			ProducerReceiptID: "phase-insert-proposal-" + gap.ContentHash[:12],
		}
		if err := colony.AddressPlanningDimensionAssessment(&assessment); err != nil {
			return nil, nil, err
		}
		assessments = append(assessments, assessment)
		gaps = append(gaps, gap)
	}
	return assessments, gaps, nil
}

func phaseInsertStopDecision(gaps []colony.PlanningGap, evidenceID string) (colony.PlanningStopDecision, error) {
	if len(gaps) == 0 {
		return colony.PlanningStopDecision{}, fmt.Errorf("phase insertion requires residual review gaps")
	}
	gapIDs := make([]string, len(gaps))
	for index := range gaps {
		gapIDs[index] = gaps[index].ID
	}
	decision := colony.PlanningStopDecision{
		SchemaVersion: colony.PlanningSchemaVersion, Reason: colony.PlanningStopTargetMet,
		SelectedGapID: gaps[0].ID, ResidualGapIDs: gapIDs, EvidenceIDs: []string{evidenceID},
		Rationale:               "The bounded manual insertion proposal is complete and now requires exact owner acceptance rather than automatic activation.",
		EvidenceThatWouldChange: gaps[0].EvidenceThatWouldChange,
	}
	if err := addressPlanningStopDecision(&decision); err != nil {
		return colony.PlanningStopDecision{}, err
	}
	return decision, nil
}

func phaseInsertTimelinePreview(card colony.PlanningIterationCard) (colony.PlanningTimelineBinding, error) {
	receiptID := card.RouteSetterReceiptID
	requestDigest, err := planningTimelineAppendRequestDigest(receiptID, card, "")
	if err != nil {
		return colony.PlanningTimelineBinding{}, err
	}
	transactionID, err := planningTimelineTransactionID(card.RunID, receiptID)
	if err != nil {
		return colony.PlanningTimelineBinding{}, err
	}
	index := planningTimelineIndex{
		SchemaVersion: planningTimelineIndexSchemaVersion, RunID: card.RunID,
		Entries: []planningTimelineIndexEntry{{
			Iteration: card.Iteration, CardID: card.ID, CardHash: card.ContentHash,
			CardPath:        planningTimelineCardRepositoryPath(card.RunID, card.Iteration, card.ID),
			AppendReceiptID: receiptID, AppendRequestDigest: requestDigest, TransactionID: transactionID,
		}},
	}
	if err := addressPlanningTimelineIndex(&index, []colony.PlanningIterationCard{card}); err != nil {
		return colony.PlanningTimelineBinding{}, err
	}
	return planningTimelineBindingFor(index, []colony.PlanningIterationCard{card})
}

func phaseInsertCandidatePhases(phases []colony.Phase, candidateID, candidateHash string, specification planningStageSpecificationBinding, timeline colony.PlanningTimelineBinding, affected, preserved, coverage []string) []colony.Phase {
	result := clonePhases(phases)
	affectedSet := planImpactIDSet(affected)
	preservedSet := planImpactIDSet(preserved)
	for phaseIndex := range result {
		phase := &result[phaseIndex]
		phase.SpecificationRevisionID, phase.SpecificationRevisionHash = specification.RevisionID, specification.ContentHash
		phase.CandidateID, phase.CandidateContentHash = candidateID, candidateHash
		phase.PlanningTimelineID, phase.PlanningTimelineDigest = timeline.ID, timeline.TimelineDigest
		phase.AffectedSemanticIDs = nil
		phase.PreservedSemanticIDs = nil
		if _, ok := affectedSet[phase.SemanticID]; ok {
			phase.AffectedSemanticIDs = append([]string{phase.SemanticID}, coverage...)
		}
		if _, ok := preservedSet[phase.SemanticID]; ok {
			phase.PreservedSemanticIDs = []string{phase.SemanticID}
		}
		for taskIndex := range phase.Tasks {
			task := &phase.Tasks[taskIndex]
			task.SpecificationRevisionID, task.SpecificationRevisionHash = specification.RevisionID, specification.ContentHash
			task.CandidateID, task.CandidateContentHash = candidateID, candidateHash
			task.PlanningTimelineID, task.PlanningTimelineDigest = timeline.ID, timeline.TimelineDigest
			task.AffectedSemanticIDs = nil
			task.PreservedSemanticIDs = nil
			if _, ok := affectedSet[task.SemanticID]; ok {
				task.AffectedSemanticIDs = []string{task.SemanticID}
			}
			if _, ok := preservedSet[task.SemanticID]; ok {
				task.PreservedSemanticIDs = []string{task.SemanticID}
			}
		}
	}
	return result
}

func phaseInsertRevisionPhaseSets(phases []colony.Phase, impact planImpactClosure) ([]int, []int) {
	affected := planImpactIDSet(impact.AffectedSemanticIDs)
	var preserved, superseded []int
	for _, phase := range phases {
		if _, ok := affected[planImpactPhaseID(phase)]; ok {
			superseded = append(superseded, phase.ID)
		} else {
			preserved = append(preserved, phase.ID)
		}
	}
	return preserved, superseded
}

func phaseInsertRecommendation(candidateID, evidenceID string, createdAt time.Time) (colony.QueenPlanRecommendation, error) {
	recommendation := colony.QueenPlanRecommendation{
		SchemaVersion: colony.PlanningSchemaVersion, CandidateID: candidateID,
		Disposition: colony.PlanRecommendationAccept, EvidenceIDs: []string{evidenceID},
		Rationale: "The requested phase is specification-covered, isolated from the active revision, and safe to review for exact acceptance.",
		Producer:  colony.PlanRecommendationProducerQueen, ProducerID: "go-queen/phase-insert/v1", CreatedAt: createdAt,
	}
	if err := colony.AddressQueenPlanRecommendation(&recommendation); err != nil {
		return colony.QueenPlanRecommendation{}, err
	}
	if candidateID == "" {
		return recommendation, nil
	}
	return recommendation, recommendation.Validate()
}

func phaseInsertPlanningBoundary(state colony.ColonyState, approved approvedPlanningSpecification, baseRevisionID, baseStateHash, runID, evidenceID, evidenceHash string, gap colony.PlanningGap, card colony.PlanningIterationCard, score int, createdAt time.Time) (planningStageManifest, planningRunHeader, planningStageState, error) {
	var emptyManifest planningStageManifest
	var emptyHeader planningRunHeader
	var emptyStage planningStageState
	priorHash := planningEvidenceSHA256([]byte("phase-insert-origin\n" + baseRevisionID))
	manifest := planningStageManifest{
		AuthorizationID: "phase-insert-authorization-" + evidenceHash[:12], RunID: runID, Pass: 1,
		Preset: planningStagePresetFast, Specification: approved.Binding,
		BasePlanRevisionID: baseRevisionID, BasePlanRevisionHash: baseStateHash,
		PriorCardHash: priorHash, InputFrontierHash: evidenceHash,
		ExpectedCaste: planningStageCasteScout, ExpectedResultType: planningStageResultScout,
		EvidenceFrontier: []planningStageEvidenceBinding{{ID: evidenceID, ContentHash: evidenceHash}}, WeakestGap: &gap,
	}
	if err := addressPlanningStageManifest(&manifest); err != nil {
		return emptyManifest, emptyHeader, emptyStage, err
	}
	goal := "active colony"
	if state.Goal != nil && strings.TrimSpace(*state.Goal) != "" {
		goal = strings.TrimSpace(*state.Goal)
	}
	header := planningRunHeader{
		SchemaVersion: planningRunHeaderSchemaVersion, RunID: runID, Goal: goal,
		GoalID: approved.GoalID, SessionID: approved.SessionID, Specification: approved.Binding,
		BasePlanRevisionID: baseRevisionID, BasePlanRevisionHash: baseStateHash,
		Preset: planningStagePresetFast, TargetConfidence: score, PassCap: 1,
		EvidenceFrontier:  []planningStageEvidenceBinding{{ID: evidenceID, ContentHash: evidenceHash}},
		InputFrontierHash: evidenceHash, WeakestGap: gap,
		ResearchPolicy: phaseResearchAutomaticPolicy{
			SchemaVersion: phaseResearchAutomaticPolicySchemaVersion, Preset: planningStagePresetFast,
			WeakestGapID: gap.ID, RequiresOwnerPrompt: false, OwnerDecisionBoundary: "after_scout_pass",
			EvidenceContract: automaticPhaseResearchEvidenceContract(),
		},
		StageManifestID: manifest.ID, StageManifestHash: manifest.ContentHash, CreatedAt: createdAt,
	}
	headerPayload := header
	headerPayload.ID, headerPayload.ContentHash = "", ""
	headerHash, err := jsonSHA256(headerPayload)
	if err != nil {
		return emptyManifest, emptyHeader, emptyStage, err
	}
	header.ContentHash = headerHash
	header.ID = "planning-run-header-" + headerHash[:16]
	stage := planningStageState{
		Stage: planningStageCandidateReady, RunID: runID, Pass: 1, Preset: planningStagePresetFast,
		Specification: approved.Binding, BasePlanRevisionID: baseRevisionID, BasePlanRevisionHash: baseStateHash,
		PriorCardHash: card.ContentHash, InputFrontierHash: evidenceHash, WeakestGap: &gap,
		UsedAuthorizationIDs: []string{manifest.AuthorizationID},
	}
	if err := validatePlanningStageManifest(manifest); err != nil {
		return emptyManifest, emptyHeader, emptyStage, err
	}
	if err := validatePlanningStageAuthority(stage); err != nil {
		return emptyManifest, emptyHeader, emptyStage, err
	}
	return manifest, header, stage, nil
}

// acceptPlanCandidate is the sole pending_review -> accepted authority. The
// Plan 28 repository session is acquired before any authority read and remains
// held through derivation, atomic apply, rollback, or exact read-only replay.
func acceptPlanCandidate(root string, request planCandidateAcceptanceRequest, opts planCandidateAcceptanceOptions) (planCandidateAcceptanceResult, error) {
	// Direct internal callers predating the command clock seam may omit the
	// timestamp. Preserve that API while still sampling the same seam exactly
	// once; runPlanCandidateCommand always supplies its already-sampled value.
	if opts.AcceptedAt.IsZero() {
		opts.AcceptedAt = planCandidateNow().UTC()
	}
	var result planCandidateAcceptanceResult
	err := withPlanningMutationSession(root, "plan-candidate-accept", func(session *planningMutationSession) error {
		var err error
		result, err = acceptPlanCandidateInSession(session, request, opts)
		return err
	})
	return result, err
}

func acceptPlanCandidateInSession(session *planningMutationSession, request planCandidateAcceptanceRequest, opts planCandidateAcceptanceOptions) (planCandidateAcceptanceResult, error) {
	empty := planCandidateAcceptanceResult{}
	artifact, err := loadPlanCandidateArtifactInSession(session, request.CandidateID)
	if err != nil {
		return empty, fmt.Errorf("candidate_id: %w", err)
	}
	candidate := artifact.Candidate
	if err := validatePlanCandidateAcceptanceRequest(candidate, request); err != nil {
		assessment := unavailablePlanCandidateStanding("acceptance_request_changed", err.Error())
		return refusePlanCandidateAcceptance(candidate, assessment, err)
	}
	state, err := loadSpecificationColonyStateInSession(session)
	if err != nil {
		assessment := unavailablePlanCandidateStanding("planning_state_unavailable", err.Error())
		return refusePlanCandidateAcceptance(candidate, assessment, err)
	}
	if err := validatePlanningState(state); err != nil {
		cause := fmt.Errorf("validate current planning state: %w", err)
		assessment := unavailablePlanCandidateStanding("planning_state_invalid", cause.Error())
		return refusePlanCandidateAcceptance(candidate, assessment, cause)
	}
	timeline, err := loadPlanCandidateTimelineAuthorityInSession(session, candidate)
	if err != nil {
		assessment := unavailablePlanCandidateStanding("timeline_changed", err.Error())
		return refusePlanCandidateAcceptance(candidate, assessment, err)
	}
	persistedReceipt, receiptExists, err := loadPlanCandidateAcceptanceReceiptInSession(session, candidate)
	if err != nil {
		assessment := unavailablePlanCandidateStanding("acceptance_receipt_invalid", err.Error())
		return refusePlanCandidateAcceptance(candidate, assessment, err)
	}
	authority := planCandidateAuthorityFromState(state, artifact, *timeline.Binding)
	assessment := assessPlanCandidateStanding(candidate, authority, opts.AcceptedAt)

	if candidate.Status == colony.PlanCandidateAccepted {
		if !receiptExists || candidate.Acceptance == nil || !reflect.DeepEqual(*persistedReceipt, *candidate.Acceptance) {
			cause := fmt.Errorf("acceptance receipt: accepted replay does not bind the exact persisted receipt")
			assessment = unavailablePlanCandidateStanding("acceptance_receipt_invalid", cause.Error())
			return refusePlanCandidateAcceptance(candidate, assessment, cause)
		}
		if assessment.Standing != planCandidateStandingAccepted {
			return refusePlanCandidateAcceptance(candidate, assessment, fmt.Errorf("candidate %s cannot replay: %s", candidate.ID, assessment.WhyUnavailable))
		}
		return replayAcceptedPlanCandidate(state, candidate)
	}
	if receiptExists {
		cause := fmt.Errorf("acceptance receipt: pending candidate already has a receipt artifact")
		assessment = unavailablePlanCandidateStanding("acceptance_receipt_invalid", cause.Error())
		return refusePlanCandidateAcceptance(candidate, assessment, cause)
	}
	if assessment.Standing == planCandidateStandingExpired && candidate.Status == colony.PlanCandidatePendingReview {
		return expirePlanCandidateInSession(session, artifact, state, assessment, opts)
	}
	if assessment.Standing != planCandidateStandingCurrent {
		return refusePlanCandidateAcceptance(candidate, assessment, fmt.Errorf("candidate %s cannot be accepted: %s", candidate.ID, assessment.WhyUnavailable))
	}
	if pendingID := strings.TrimSpace(state.Plan.PendingCandidateID); pendingID != "" && pendingID != candidate.ID {
		return empty, fmt.Errorf("candidate_id: current pending candidate is %q, not %q", pendingID, candidate.ID)
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
	computedProposalHash, err := canonicalPlanCandidateProposalHash(candidate.Proposal)
	if err != nil {
		return empty, fmt.Errorf("proposal_hash: %w", err)
	}
	if computedProposalHash != candidate.ProposalHash || candidate.Proposal.PlanHash != candidate.ProposalHash {
		return empty, fmt.Errorf("proposal_hash: candidate proposal no longer matches its exact binding")
	}
	baseRevision, err := planCandidateDerivationBase(state.Plan, base, baseline)
	if err != nil {
		return empty, err
	}
	derived, err := derivePlanCandidateAuthority(baseRevision, *state.Specification, candidate.Proposal)
	if err != nil {
		return empty, fmt.Errorf("derived authority: %w", err)
	}
	if len(timeline.Cards) == 0 {
		return empty, fmt.Errorf("derived authority: candidate timeline has no final iteration")
	}
	finalCard := timeline.Cards[len(timeline.Cards)-1]
	if err := validateDerivedPlanCandidateAuthority(candidate, finalCard, derived); err != nil {
		return empty, fmt.Errorf("derived authority: %w", err)
	}
	derivedAffected, derivedPreserved := derivedPlanCandidateScope(finalCard.SemanticDelta, derived)
	// Completion preservation follows the independently verified candidate
	// scope, not the broader graph used to prove semantic reachability. The
	// latter may include a shared proof attached to an otherwise unchanged
	// completed phase (for example, an inserted corrective phase), which must
	// not erase already-earned execution credit.
	activationImpact := derived.Impact
	// A node the card marks affected purely through its declaration wrapper
	// (files/user-facing marking) while its repository-derived definition is
	// byte-preserved must not erase earned completion credit -- exclude it
	// from the activation set only; the immutable proposal markers above are
	// untouched.
	contentPreserved := derivedContentPreservedIDs(derived)
	activationAffected := make([]string, 0, len(derivedAffected))
	for _, id := range derivedAffected {
		if _, keepCredit := contentPreserved[id]; keepCredit {
			continue
		}
		activationAffected = append(activationAffected, id)
	}
	activationImpact.AffectedSemanticIDs = activationAffected
	activatedPhases, preserved, err := preserveCompletedCandidateWorkForImpact(state.Plan.Phases, candidate.Proposal.Phases, activationImpact)
	if err != nil {
		return empty, err
	}
	revision := candidate.Proposal
	revision.Phases = activatedPhases
	revision.PreservedPhaseIDs = preserved
	revision.AffectedSemanticIDs = append([]string(nil), derivedAffected...)
	revision.PreservedSemanticIDs = append([]string(nil), derivedPreserved...)
	if computed, hashErr := canonicalPlanCandidateProposalHash(revision); hashErr != nil || computed != revision.PlanHash {
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
	receipt, err := newPlanCandidateAcceptanceReceipt(candidate, request.AcceptanceToken, revision, acceptedBy, acceptedAt)
	if err != nil {
		return empty, err
	}
	candidate.Status = colony.PlanCandidateAccepted
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
	tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{
		TransactionID: "plan-candidate-accept-" + candidate.ContentHash[:24],
		Command:       "plan-candidate-accept",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot: session.RepositoryRoot(), LifecycleDataRoot: session.DataRoot(),
		},
		Session: session, Fault: opts.Fault, Rename: opts.Rename,
	})
	if err != nil {
		return empty, err
	}
	for _, target := range []struct {
		path    string
		content []byte
	}{
		{path: planningStageDataRelativePath(planningRouteCandidateRepositoryPath(candidate.Timeline.RunID)), content: candidateBytes},
		{path: planningStageDataRelativePath(planningRouteAcceptanceRepositoryPath(candidate.Timeline.RunID)), content: receiptBytes},
		{path: planningStageDataRelativePath(planningStageStateRepositoryPath(candidate.Timeline.RunID)), content: stageBytes},
		{path: "COLONY_STATE.json", content: stateBytes},
	} {
		if err := tx.DeclareWrite(lifecycleTransactionRootData, target.path, target.content); err != nil {
			return empty, err
		}
	}
	if err := tx.Validate(); err != nil {
		return empty, err
	}
	if _, err := tx.Commit(); err != nil {
		if rollbackErr := tx.rollbackPreparedTargets(); rollbackErr != nil {
			return empty, errors.Join(err, fmt.Errorf("rollback candidate acceptance: %w", rollbackErr))
		}
		return empty, err
	}
	return planCandidateAcceptanceResult{Candidate: candidate, Revision: revision, Receipt: receipt}, nil
}

func unavailablePlanCandidateStanding(reason string, evidence ...string) planCandidateStandingAssessment {
	return planCandidateStandingAssessment{
		Standing: planCandidateStandingStale, WhyUnavailable: strings.TrimSpace(reason),
		Evidence: append([]string(nil), evidence...), AcceptanceAvailable: false,
		StateEffect: planCandidateStateEffectUnchanged, ActivePlanEffect: planCandidateActivePlanEffectUnchanged,
		RecoveryCommand: planCandidateRefreshCommand,
	}
}

func refusePlanCandidateAcceptance(candidate colony.PlanCandidate, assessment planCandidateStandingAssessment, cause error) (planCandidateAcceptanceResult, error) {
	if assessment.StateEffect == "" {
		assessment.StateEffect = planCandidateStateEffectUnchanged
	}
	if assessment.ActivePlanEffect == "" {
		assessment.ActivePlanEffect = planCandidateActivePlanEffectUnchanged
	}
	if strings.TrimSpace(assessment.RecoveryCommand) == "" {
		assessment.RecoveryCommand = planCandidateRefreshCommand
	}
	details := planCandidateRefusal(candidate, assessment)
	return planCandidateAcceptanceResult{Candidate: candidate, Refusal: &details}, &planCandidateRefusalError{Details: details, Cause: cause}
}

func loadPlanCandidateTimelineAuthorityInSession(session *planningMutationSession, candidate colony.PlanCandidate) (planningTimeline, error) {
	if err := validatePlanningTimelineSegment("run_id", candidate.Timeline.RunID); err != nil {
		return planningTimeline{}, err
	}
	index, cards, exists, err := readPlanningTimelineChainInSession(session, candidate.Timeline.RunID)
	if err != nil {
		return planningTimeline{}, err
	}
	if !exists {
		return planningTimeline{}, fmt.Errorf("candidate %s requires a complete indexed timeline", candidate.ID)
	}
	binding, err := planningTimelineBindingFor(index, cards)
	if err != nil {
		return planningTimeline{}, err
	}
	indexCopy := index
	return planningTimeline{
		Classification: planningTimelineCandidateOnly, RunID: candidate.Timeline.RunID,
		Index: &indexCopy, Cards: cards, Binding: &binding,
	}, nil
}

func expirePlanCandidateInSession(session *planningMutationSession, artifact planCandidateArtifact, state colony.ColonyState, assessment planCandidateStandingAssessment, opts planCandidateAcceptanceOptions) (planCandidateAcceptanceResult, error) {
	candidate := artifact.Candidate
	if artifact.Stage.Stage != planningStageCandidateReady {
		assessment = unavailablePlanCandidateStanding("planning_stage_changed",
			fmt.Sprintf("candidate expiry expected stage %s, current stage is %s", planningStageCandidateReady, artifact.Stage.Stage))
		return refusePlanCandidateAcceptance(candidate, assessment, fmt.Errorf("candidate %s cannot expire from stage %s", candidate.ID, artifact.Stage.Stage))
	}

	candidate.Status = colony.PlanCandidateExpired
	candidate.Acceptance = nil
	if err := validatePlanningRecordHashes(candidate); err != nil {
		return planCandidateAcceptanceResult{}, fmt.Errorf("expired candidate: %w", err)
	}
	if err := candidate.Validate(); err != nil {
		return planCandidateAcceptanceResult{}, fmt.Errorf("expired candidate: %w", err)
	}
	nextStage, _, err := reducePlanningStage(artifact.Stage, planningStageTransition{
		To: planningStageFailed, FailureReason: planningStageFailureCandidateExpired,
	})
	if err != nil {
		return planCandidateAcceptanceResult{}, err
	}

	nextState := state
	nextState.Plan.Candidates = append([]colony.PlanCandidate(nil), state.Plan.Candidates...)
	stateChanged := false
	for index := range nextState.Plan.Candidates {
		retained := nextState.Plan.Candidates[index]
		if retained.ID != candidate.ID {
			continue
		}
		if retained.ContentHash != candidate.ContentHash || retained.Status != colony.PlanCandidatePendingReview {
			return planCandidateAcceptanceResult{}, fmt.Errorf("candidate_id: retained candidate %q conflicts with expiry", candidate.ID)
		}
		nextState.Plan.Candidates[index] = candidate
		stateChanged = true
	}
	if nextState.Plan.PendingCandidateID == candidate.ID {
		nextState.Plan.PendingCandidateID = ""
		stateChanged = true
	}
	if stateChanged {
		if err := validatePlanningState(nextState); err != nil {
			return planCandidateAcceptanceResult{}, fmt.Errorf("validate expired candidate state: %w", err)
		}
	}

	candidateBytes, err := marshalPlanningStageJSON(candidate)
	if err != nil {
		return planCandidateAcceptanceResult{}, err
	}
	stageBytes, err := marshalPlanningStageJSON(nextStage)
	if err != nil {
		return planCandidateAcceptanceResult{}, err
	}
	targets := []struct {
		path    string
		content []byte
	}{
		{path: planningStageDataRelativePath(planningRouteCandidateRepositoryPath(candidate.Timeline.RunID)), content: candidateBytes},
		{path: planningStageDataRelativePath(planningStageStateRepositoryPath(candidate.Timeline.RunID)), content: stageBytes},
	}
	if stateChanged {
		stateBytes, marshalErr := marshalSpecificationState(nextState)
		if marshalErr != nil {
			return planCandidateAcceptanceResult{}, marshalErr
		}
		targets = append(targets, struct {
			path    string
			content []byte
		}{path: "COLONY_STATE.json", content: stateBytes})
	}

	tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{
		TransactionID: "plan-candidate-expire-" + candidate.ContentHash[:24],
		Command:       "plan-candidate-expire",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot: session.RepositoryRoot(), LifecycleDataRoot: session.DataRoot(),
		},
		Session: session, Fault: opts.Fault, Rename: opts.Rename,
	})
	if err != nil {
		return planCandidateAcceptanceResult{}, err
	}
	for _, target := range targets {
		if err := tx.DeclareWrite(lifecycleTransactionRootData, target.path, target.content); err != nil {
			return planCandidateAcceptanceResult{}, err
		}
	}
	if err := tx.Validate(); err != nil {
		return planCandidateAcceptanceResult{}, err
	}
	if _, err := tx.Commit(); err != nil {
		if rollbackErr := tx.rollbackPreparedTargets(); rollbackErr != nil {
			return planCandidateAcceptanceResult{}, errors.Join(err, fmt.Errorf("rollback candidate expiry: %w", rollbackErr))
		}
		return planCandidateAcceptanceResult{}, err
	}

	assessment.StateEffect = planCandidateStateEffectMarkedExpired
	details := planCandidateRefusal(candidate, assessment)
	cause := fmt.Errorf("candidate %s expired at %s; run `%s`", candidate.ID, candidate.ExpiresAt.UTC().Format(time.RFC3339Nano), planCandidateRefreshCommand)
	return planCandidateAcceptanceResult{Candidate: candidate, Refusal: &details}, &planCandidateRefusalError{Details: details, Cause: cause}
}

func loadPlanCandidateArtifactInSession(session *planningMutationSession, requestedID string) (planCandidateArtifact, error) {
	if err := session.requireActive(); err != nil {
		return planCandidateArtifact{}, err
	}
	planningRoot := filepath.Join(session.DataRoot(), "planning")
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
		content, exists, err := readOptionalPlanningStageFileInSession(session, planningRouteCandidateRepositoryPath(runID))
		if err != nil {
			return planCandidateArtifact{}, err
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
		stage, err := loadPlanningStageStateInSession(session, runID)
		if err != nil {
			return planCandidateArtifact{}, err
		}
		header, err := loadPlanCandidateRunHeaderInSession(session, candidate)
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

func loadPlanCandidateRunHeaderInSession(session *planningMutationSession, candidate colony.PlanCandidate) (planningRunHeader, error) {
	headerPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", candidate.Timeline.RunID, "run-header.json"))
	content, exists, err := readOptionalPlanningStageFileInSession(session, headerPath)
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
	manifest, err := loadPlanningStageManifestInSession(session, header.RunID, header.StageManifestID)
	if err != nil {
		return planningRunHeader{}, err
	}
	payload := header
	payload.ID, payload.ContentHash = "", ""
	wantHash, err := jsonSHA256(payload)
	if err != nil {
		return planningRunHeader{}, fmt.Errorf("hash planning run header: %w", err)
	}
	if header.SchemaVersion != planningRunHeaderSchemaVersion || header.ContentHash != wantHash || header.ID != "planning-run-header-"+wantHash[:16] {
		return planningRunHeader{}, fmt.Errorf("planning run header is not a valid immutable content address")
	}
	if header.RunID != manifest.RunID || header.Preset != manifest.Preset ||
		!samePlanningScoutSpecification(header.Specification, manifest.Specification) ||
		header.BasePlanRevisionID != manifest.BasePlanRevisionID || header.BasePlanRevisionHash != manifest.BasePlanRevisionHash {
		return planningRunHeader{}, fmt.Errorf("planning run header does not match the first Scout manifest authority")
	}
	if strings.TrimSpace(header.GoalID) == "" || strings.TrimSpace(header.SessionID) == "" {
		return planningRunHeader{}, fmt.Errorf("planning run header requires goal and session scope")
	}
	if manifest.Pass == 1 && (header.StageManifestID != manifest.ID || header.StageManifestHash != manifest.ContentHash || header.InputFrontierHash != manifest.InputFrontierHash) {
		return planningRunHeader{}, fmt.Errorf("planning run header does not bind the first Scout manifest and frontier")
	}
	if header.Specification.RevisionID != candidate.SpecificationRevisionID || header.Specification.ContentHash != candidate.SpecificationRevisionHash {
		return planningRunHeader{}, fmt.Errorf("candidate %s does not match its immutable planning run header", candidate.ID)
	}
	return header, nil
}

func verifiedPlanCandidateTimelineInSession(session *planningMutationSession, candidate colony.PlanCandidate) (planningTimeline, error) {
	if err := validatePlanningTimelineSegment("run_id", candidate.Timeline.RunID); err != nil {
		return planningTimeline{}, err
	}
	index, cards, exists, err := readPlanningTimelineChainInSession(session, candidate.Timeline.RunID)
	if err != nil {
		return planningTimeline{}, err
	}
	if !exists {
		return planningTimeline{}, fmt.Errorf("candidate %s requires a complete indexed timeline", candidate.ID)
	}
	binding, err := planningTimelineBindingFor(index, cards)
	if err != nil {
		return planningTimeline{}, err
	}
	if !reflect.DeepEqual(binding, candidate.Timeline) {
		return planningTimeline{}, fmt.Errorf("candidate %s timeline binding is stale or divergent", candidate.ID)
	}
	indexCopy := index
	return planningTimeline{
		Classification: planningTimelineCandidateOnly, RunID: candidate.Timeline.RunID,
		Index: &indexCopy, Cards: cards, Binding: &binding,
	}, nil
}

func loadPlanCandidateAcceptanceReceiptInSession(session *planningMutationSession, candidate colony.PlanCandidate) (*colony.PlanAcceptanceReceipt, bool, error) {
	content, exists, err := readOptionalPlanningStageFileInSession(session, planningRouteAcceptanceRepositoryPath(candidate.Timeline.RunID))
	if err != nil || !exists {
		return nil, exists, err
	}
	var receipt colony.PlanAcceptanceReceipt
	if err := decodePlanningStageJSON(content, &receipt); err != nil {
		return nil, false, fmt.Errorf("decode candidate acceptance receipt: %w", err)
	}
	if err := receipt.Validate(); err != nil {
		return nil, false, fmt.Errorf("validate candidate acceptance receipt: %w", err)
	}
	if err := validatePlanCandidateAcceptanceReceiptHash(receipt); err != nil {
		return nil, false, err
	}
	return &receipt, true, nil
}

func planCandidateDerivationBase(plan colony.Plan, base planCandidateBase, baseline *colony.PlanRevision) (colony.PlanRevision, error) {
	if base.ID == "plan-unbound" {
		return colony.PlanRevision{ID: base.ID, PlanHash: base.Hash}, nil
	}
	if baseline != nil {
		return *baseline, nil
	}
	for _, revision := range plan.Revisions {
		if revision.ID == base.ID && revision.PlanHash == base.Hash {
			return revision, nil
		}
	}
	return colony.PlanRevision{}, fmt.Errorf("derived base: retained plan revision %q is unavailable", base.ID)
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
		copyPhases[i].RequirementProofLinks = cloneStrings(phase.RequirementProofLinks)
		copyPhases[i].AcceptanceProofLinks = cloneStrings(phase.AcceptanceProofLinks)
		copyPhases[i].NegativeProofLinks = cloneStrings(phase.NegativeProofLinks)
		copyPhases[i].RecoveryProofLinks = cloneStrings(phase.RecoveryProofLinks)
		copyPhases[i].PublicPathProofLinks = cloneStrings(phase.PublicPathProofLinks)
		copyPhases[i].AffectedSemanticIDs = cloneStrings(phase.AffectedSemanticIDs)
		copyPhases[i].PreservedSemanticIDs = cloneStrings(phase.PreservedSemanticIDs)
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
			task.RequirementProofLinks = cloneStrings(task.RequirementProofLinks)
			task.AcceptanceProofLinks = cloneStrings(task.AcceptanceProofLinks)
			task.NegativeProofLinks = cloneStrings(task.NegativeProofLinks)
			task.RecoveryProofLinks = cloneStrings(task.RecoveryProofLinks)
			task.PublicPathProofLinks = cloneStrings(task.PublicPathProofLinks)
			task.AffectedSemanticIDs = cloneStrings(task.AffectedSemanticIDs)
			task.PreservedSemanticIDs = cloneStrings(task.PreservedSemanticIDs)
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
