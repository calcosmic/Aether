package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

const planningStageManifestSchemaVersion = "planning-stage-manifest/v1"

// planningStage is the closed, renderer-neutral state vocabulary for one
// iterative planning run. A stage describes authority, not worker liveness:
// only the two *_running states have a one-worker dispatch manifest.
type planningStage string

const (
	planningStagePresetRequired         planningStage = "preset_required"
	planningStageScoutReady             planningStage = "scout_ready"
	planningStageScoutRunning           planningStage = "scout_running"
	planningStageOwnerDecision          planningStage = "owner_decision"
	planningStageSpecApprovalRequired   planningStage = "spec_approval_required"
	planningStageReconciliationRequired planningStage = "reconciliation_required"
	planningStageRouteReady             planningStage = "route_ready"
	planningStageRouteRunning           planningStage = "route_running"
	planningStageContinueReady          planningStage = "continue_ready"
	planningStageCandidateReady         planningStage = "candidate_ready"
	planningStageAccepted               planningStage = "accepted"
	planningStageFailed                 planningStage = "failed"
)

var planningStageOrder = []planningStage{
	planningStagePresetRequired,
	planningStageScoutReady,
	planningStageScoutRunning,
	planningStageOwnerDecision,
	planningStageSpecApprovalRequired,
	planningStageReconciliationRequired,
	planningStageRouteReady,
	planningStageRouteRunning,
	planningStageContinueReady,
	planningStageCandidateReady,
	planningStageAccepted,
	planningStageFailed,
}

// planningStageTransitions is deliberately closed. Adding a state is not
// enough to make it reachable: the exact edge must also be reviewed here and
// in the exhaustive matrix test.
var planningStageTransitions = map[planningStage]map[planningStage]struct{}{
	planningStagePresetRequired:         planningStageSet(planningStageScoutReady, planningStageFailed),
	planningStageScoutReady:             planningStageSet(planningStageScoutRunning, planningStageFailed),
	planningStageScoutRunning:           planningStageSet(planningStageOwnerDecision, planningStageRouteReady, planningStageFailed),
	planningStageOwnerDecision:          planningStageSet(planningStageScoutReady, planningStageSpecApprovalRequired, planningStageRouteReady, planningStageFailed),
	planningStageSpecApprovalRequired:   planningStageSet(planningStageReconciliationRequired, planningStageFailed),
	planningStageReconciliationRequired: planningStageSet(planningStageScoutReady, planningStageFailed),
	planningStageRouteReady:             planningStageSet(planningStageRouteRunning, planningStageFailed),
	planningStageRouteRunning:           planningStageSet(planningStageOwnerDecision, planningStageContinueReady, planningStageCandidateReady, planningStageFailed),
	planningStageContinueReady:          planningStageSet(planningStageScoutReady, planningStageFailed),
	planningStageCandidateReady:         planningStageSet(planningStageAccepted, planningStageFailed),
	planningStageAccepted:               planningStageSet(),
	planningStageFailed:                 planningStageSet(),
}

func planningStageSet(values ...planningStage) map[planningStage]struct{} {
	result := make(map[planningStage]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func planningStageValues() []planningStage {
	return append([]planningStage(nil), planningStageOrder...)
}

func (s planningStage) valid() bool {
	_, ok := planningStageTransitions[s]
	return ok
}

func planningStageTransitionAllowed(from, to planningStage) bool {
	allowed, ok := planningStageTransitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

type planningStagePreset string

const (
	planningStagePresetFast       planningStagePreset = "fast"
	planningStagePresetBalanced   planningStagePreset = "balanced"
	planningStagePresetDeep       planningStagePreset = "deep"
	planningStagePresetExhaustive planningStagePreset = "exhaustive"
)

func (p planningStagePreset) valid() bool {
	switch p {
	case planningStagePresetFast, planningStagePresetBalanced, planningStagePresetDeep, planningStagePresetExhaustive:
		return true
	default:
		return false
	}
}

type planningStageWorkerCaste string

const (
	planningStageCasteScout       planningStageWorkerCaste = "scout"
	planningStageCasteRouteSetter planningStageWorkerCaste = "route_setter"
)

func (c planningStageWorkerCaste) valid() bool {
	return c == planningStageCasteScout || c == planningStageCasteRouteSetter
}

type planningStageResultType string

const (
	planningStageResultScout       planningStageResultType = "planning-scout-result/v1"
	planningStageResultRouteSetter planningStageResultType = "planning-route-setter-result/v1"
)

func expectedPlanningStageResult(caste planningStageWorkerCaste) planningStageResultType {
	switch caste {
	case planningStageCasteScout:
		return planningStageResultScout
	case planningStageCasteRouteSetter:
		return planningStageResultRouteSetter
	default:
		return ""
	}
}

// planningStageEvidenceBinding is the bounded evidence catalogue projected
// into a Scout manifest. Content remains in the evidence store; the manifest
// authorizes only these exact content addresses.
type planningStageEvidenceBinding struct {
	ID          string `json:"id"`
	ContentHash string `json:"content_hash"`
}

func (b planningStageEvidenceBinding) validate() error {
	if strings.TrimSpace(b.ID) == "" {
		return fmt.Errorf("evidence ID is required")
	}
	if !planningSHA256Pattern.MatchString(strings.TrimSpace(b.ContentHash)) {
		return fmt.Errorf("evidence %q content hash must be a lowercase SHA-256 digest", b.ID)
	}
	return nil
}

// planningStageSpecificationBinding keeps content identity separate from the
// approval receipt. Approval changes authority without changing revision
// content, while an exact successor changes RevisionID and ContentHash.
type planningStageSpecificationBinding struct {
	RevisionID            string                    `json:"revision_id"`
	ContentHash           string                    `json:"content_hash"`
	PredecessorRevisionID string                    `json:"predecessor_revision_id,omitempty"`
	Status                colony.SpecRevisionStatus `json:"status"`
	ApprovalReceiptID     string                    `json:"approval_receipt_id,omitempty"`
	ApprovalReceiptHash   string                    `json:"approval_receipt_hash,omitempty"`
}

func (b planningStageSpecificationBinding) validate(requireApproved bool) error {
	if strings.TrimSpace(b.RevisionID) == "" {
		return fmt.Errorf("specification revision ID is required")
	}
	if !planningSHA256Pattern.MatchString(strings.TrimSpace(b.ContentHash)) {
		return fmt.Errorf("specification revision content hash must be a lowercase SHA-256 digest")
	}
	if !b.Status.Valid() {
		return fmt.Errorf("invalid specification status %q", b.Status)
	}
	if requireApproved && b.Status != colony.SpecStatusApproved {
		return fmt.Errorf("specification revision %q is not approved", b.RevisionID)
	}
	if b.Status == colony.SpecStatusApproved {
		if strings.TrimSpace(b.ApprovalReceiptID) == "" || !planningSHA256Pattern.MatchString(strings.TrimSpace(b.ApprovalReceiptHash)) {
			return fmt.Errorf("approved specification revision requires an exact approval receipt")
		}
	}
	return nil
}

// planningStageReceiptRef is the minimum predecessor binding required by the
// transition reducer. The durable StageReceipt in planning_stage_receipt.go
// contains this same identity plus artifact and state-transition details.
type planningStageReceiptRef struct {
	ID           string                   `json:"id"`
	ContentHash  string                   `json:"content_hash"`
	RunID        string                   `json:"run_id"`
	Pass         int                      `json:"pass"`
	Caste        planningStageWorkerCaste `json:"caste"`
	ManifestHash string                   `json:"manifest_hash"`
}

func (r planningStageReceiptRef) validate(runID string, pass int, caste planningStageWorkerCaste) error {
	if strings.TrimSpace(r.ID) == "" || !planningSHA256Pattern.MatchString(strings.TrimSpace(r.ContentHash)) {
		return fmt.Errorf("%s receipt requires ID and SHA-256 content hash", caste)
	}
	if r.RunID != runID || r.Pass != pass || r.Caste != caste {
		return fmt.Errorf("%s receipt does not match planning run, pass, and caste", caste)
	}
	if !planningSHA256Pattern.MatchString(strings.TrimSpace(r.ManifestHash)) {
		return fmt.Errorf("%s receipt manifest hash must be a lowercase SHA-256 digest", caste)
	}
	return nil
}

type planningStageState struct {
	Stage                      planningStage                      `json:"stage"`
	RunID                      string                             `json:"run_id"`
	Pass                       int                                `json:"pass"`
	ActiveManifestID           string                             `json:"active_manifest_id,omitempty"`
	ActiveManifestHash         string                             `json:"active_manifest_hash,omitempty"`
	Preset                     planningStagePreset                `json:"preset"`
	Specification              planningStageSpecificationBinding  `json:"specification"`
	BasePlanRevisionID         string                             `json:"base_plan_revision_id"`
	BasePlanRevisionHash       string                             `json:"base_plan_revision_hash"`
	PriorCardHash              string                             `json:"prior_card_hash"`
	InputFrontierHash          string                             `json:"input_frontier_hash"`
	WeakestGap                 *colony.PlanningGap                `json:"weakest_gap,omitempty"`
	ScoutReceipt               *planningStageReceiptRef           `json:"scout_receipt,omitempty"`
	CandidateSnapshotHash      string                             `json:"candidate_snapshot_hash,omitempty"`
	UsedAuthorizationIDs       []string                           `json:"used_authorization_ids,omitempty"`
	DecisionResumeStage        planningStage                      `json:"decision_resume_stage,omitempty"`
	PendingSpecification       *planningStageSpecificationBinding `json:"pending_specification,omitempty"`
	PendingAffectedSemanticIDs []string                           `json:"pending_affected_semantic_ids,omitempty"`
	AcceptanceReceiptID        string                             `json:"acceptance_receipt_id,omitempty"`
	AcceptanceReceiptHash      string                             `json:"acceptance_receipt_hash,omitempty"`
	FailureReason              string                             `json:"failure_reason,omitempty"`
}

type planningStageAuthorization struct {
	ID                    string                         `json:"id"`
	ExpectedCaste         planningStageWorkerCaste       `json:"expected_caste"`
	InputFrontierHash     string                         `json:"input_frontier_hash"`
	EvidenceFrontier      []planningStageEvidenceBinding `json:"evidence_frontier,omitempty"`
	WeakestGap            *colony.PlanningGap            `json:"weakest_gap,omitempty"`
	ScoutReceipt          *planningStageReceiptRef       `json:"scout_receipt,omitempty"`
	CandidateSnapshotHash string                         `json:"candidate_snapshot_hash,omitempty"`
}

type planningStageReconciliation struct {
	SpecificationRevisionID   string   `json:"specification_revision_id"`
	SpecificationRevisionHash string   `json:"specification_revision_hash"`
	AffectedSemanticIDs       []string `json:"affected_semantic_ids"`
	InputFrontierHash         string   `json:"input_frontier_hash"`
}

type planningStageTransition struct {
	To                     planningStage                      `json:"to"`
	Preset                 planningStagePreset                `json:"preset,omitempty"`
	Authorization          *planningStageAuthorization        `json:"authorization,omitempty"`
	ScoutReceipt           *planningStageReceiptRef           `json:"scout_receipt,omitempty"`
	CandidateSnapshotHash  string                             `json:"candidate_snapshot_hash,omitempty"`
	ResultingCardHash      string                             `json:"resulting_card_hash,omitempty"`
	NextInputFrontierHash  string                             `json:"next_input_frontier_hash,omitempty"`
	NextWeakestGap         *colony.PlanningGap                `json:"next_weakest_gap,omitempty"`
	DecisionResumeStage    planningStage                      `json:"decision_resume_stage,omitempty"`
	DecisionResolution     *planningDecisionResolution        `json:"decision_resolution,omitempty"`
	SuccessorSpecification *planningStageSpecificationBinding `json:"successor_specification,omitempty"`
	ApprovedSpecification  *planningStageSpecificationBinding `json:"approved_specification,omitempty"`
	Reconciliation         *planningStageReconciliation       `json:"reconciliation,omitempty"`
	AcceptanceReceiptID    string                             `json:"acceptance_receipt_id,omitempty"`
	AcceptanceReceiptHash  string                             `json:"acceptance_receipt_hash,omitempty"`
	FailureReason          string                             `json:"failure_reason,omitempty"`
}

// planningStageManifest contains one caste and one result contract by shape.
// It intentionally has no acceptance, activation, next-manifest, or state
// patch field. Strict decoding rejects attempts to smuggle those fields in.
type planningStageManifest struct {
	SchemaVersion         string                            `json:"schema_version"`
	ID                    string                            `json:"id"`
	ContentHash           string                            `json:"content_hash"`
	AuthorizationID       string                            `json:"authorization_id"`
	RunID                 string                            `json:"run_id"`
	Pass                  int                               `json:"pass"`
	Preset                planningStagePreset               `json:"preset"`
	Specification         planningStageSpecificationBinding `json:"specification"`
	BasePlanRevisionID    string                            `json:"base_plan_revision_id"`
	BasePlanRevisionHash  string                            `json:"base_plan_revision_hash"`
	PriorCardHash         string                            `json:"prior_card_hash"`
	InputFrontierHash     string                            `json:"input_frontier_hash"`
	ExpectedCaste         planningStageWorkerCaste          `json:"expected_caste"`
	ExpectedResultType    planningStageResultType           `json:"expected_result_type"`
	EvidenceFrontier      []planningStageEvidenceBinding    `json:"evidence_frontier,omitempty"`
	WeakestGap            *colony.PlanningGap               `json:"weakest_gap,omitempty"`
	ScoutReceipt          *planningStageReceiptRef          `json:"scout_receipt,omitempty"`
	CandidateSnapshotHash string                            `json:"candidate_snapshot_hash,omitempty"`
}

// reducePlanningStage is a pure reducer. It clones all mutable fields before
// applying an edge and never performs I/O or mutates its input on refusal.
func reducePlanningStage(state planningStageState, transition planningStageTransition) (planningStageState, *planningStageManifest, error) {
	if !state.Stage.valid() {
		return planningStageState{}, nil, fmt.Errorf("invalid current planning stage %q", state.Stage)
	}
	if !transition.To.valid() {
		return planningStageState{}, nil, fmt.Errorf("invalid target planning stage %q", transition.To)
	}
	if !planningStageTransitionAllowed(state.Stage, transition.To) {
		return planningStageState{}, nil, fmt.Errorf("illegal planning stage transition %s -> %s", state.Stage, transition.To)
	}

	next := clonePlanningStageState(state)
	next.Stage = transition.To
	if transition.To == planningStageFailed {
		next.ActiveManifestID = ""
		next.ActiveManifestHash = ""
		next.FailureReason = strings.TrimSpace(transition.FailureReason)
		if next.FailureReason == "" {
			next.FailureReason = "planning stage failed"
		}
		return next, nil, nil
	}

	switch state.Stage {
	case planningStagePresetRequired:
		if !transition.Preset.valid() {
			return planningStageState{}, nil, fmt.Errorf("preset_required transition requires fast, balanced, deep, or exhaustive")
		}
		next.Preset = transition.Preset
		if err := validatePlanningStageAuthority(next); err != nil {
			return planningStageState{}, nil, err
		}
		return next, nil, nil

	case planningStageScoutReady, planningStageRouteReady:
		if transition.Authorization == nil {
			return planningStageState{}, nil, fmt.Errorf("%s transition requires one-stage authorization", state.Stage)
		}
		manifest, err := authorizePlanningStage(state, *transition.Authorization)
		if err != nil {
			return planningStageState{}, nil, err
		}
		next.UsedAuthorizationIDs = append(next.UsedAuthorizationIDs, manifest.AuthorizationID)
		next.ActiveManifestID = manifest.ID
		next.ActiveManifestHash = manifest.ContentHash
		next.InputFrontierHash = manifest.InputFrontierHash
		next.CandidateSnapshotHash = manifest.CandidateSnapshotHash
		return next, &manifest, nil

	case planningStageScoutRunning:
		next.ActiveManifestID = ""
		next.ActiveManifestHash = ""
		if transition.To == planningStageRouteReady || transition.To == planningStageOwnerDecision {
			if transition.ScoutReceipt == nil {
				return planningStageState{}, nil, fmt.Errorf("completed Scout transition requires the exact Scout receipt")
			}
			if err := transition.ScoutReceipt.validate(state.RunID, state.Pass, planningStageCasteScout); err != nil {
				return planningStageState{}, nil, err
			}
			next.ScoutReceipt = clonePlanningStageReceiptRef(transition.ScoutReceipt)
			if !planningSHA256Pattern.MatchString(strings.TrimSpace(transition.CandidateSnapshotHash)) {
				return planningStageState{}, nil, fmt.Errorf("completed Scout transition requires the current candidate snapshot hash")
			}
			next.CandidateSnapshotHash = strings.TrimSpace(transition.CandidateSnapshotHash)
			next.DecisionResumeStage = transition.DecisionResumeStage
			if transition.To == planningStageOwnerDecision && next.DecisionResumeStage == "" {
				next.DecisionResumeStage = planningStageRouteReady
			}
		}

	case planningStageOwnerDecision:
		return reducePlanningOwnerDecision(state, next, transition)

	case planningStageSpecApprovalRequired:
		if transition.ApprovedSpecification == nil {
			return planningStageState{}, nil, fmt.Errorf("specification approval transition requires the exact approved successor")
		}
		approved := *transition.ApprovedSpecification
		if err := approved.validate(true); err != nil {
			return planningStageState{}, nil, err
		}
		if state.PendingSpecification == nil || approved.RevisionID != state.PendingSpecification.RevisionID || approved.ContentHash != state.PendingSpecification.ContentHash || approved.PredecessorRevisionID != state.PendingSpecification.PredecessorRevisionID {
			return planningStageState{}, nil, fmt.Errorf("approved specification does not match the exact pending successor")
		}
		next.Specification = approved

	case planningStageReconciliationRequired:
		if err := applyPlanningStageReconciliation(&next, transition.Reconciliation); err != nil {
			return planningStageState{}, nil, err
		}

	case planningStageRouteRunning:
		next.ActiveManifestID = ""
		next.ActiveManifestHash = ""
		if transition.To == planningStageContinueReady || transition.To == planningStageCandidateReady || transition.To == planningStageOwnerDecision {
			if !planningSHA256Pattern.MatchString(strings.TrimSpace(transition.ResultingCardHash)) {
				return planningStageState{}, nil, fmt.Errorf("completed Route-Setter transition requires the resulting iteration card hash")
			}
			next.PriorCardHash = transition.ResultingCardHash
			next.DecisionResumeStage = transition.DecisionResumeStage
			if transition.To == planningStageOwnerDecision && next.DecisionResumeStage == "" {
				next.DecisionResumeStage = planningStageScoutReady
			}
		}

	case planningStageContinueReady:
		if !planningSHA256Pattern.MatchString(strings.TrimSpace(transition.NextInputFrontierHash)) {
			return planningStageState{}, nil, fmt.Errorf("continue transition requires the next input frontier hash")
		}
		if transition.NextWeakestGap == nil {
			return planningStageState{}, nil, fmt.Errorf("continue transition requires the next weakest gap")
		}
		if err := transition.NextWeakestGap.Validate(); err != nil {
			return planningStageState{}, nil, fmt.Errorf("next weakest gap: %w", err)
		}
		next.Pass++
		next.InputFrontierHash = transition.NextInputFrontierHash
		next.WeakestGap = clonePlanningStageGap(transition.NextWeakestGap)
		next.ScoutReceipt = nil
		next.CandidateSnapshotHash = ""
		next.DecisionResumeStage = ""

	case planningStageCandidateReady:
		if strings.TrimSpace(transition.AcceptanceReceiptID) == "" || !planningSHA256Pattern.MatchString(strings.TrimSpace(transition.AcceptanceReceiptHash)) {
			return planningStageState{}, nil, fmt.Errorf("candidate acceptance requires an exact acceptance receipt")
		}
		next.AcceptanceReceiptID = strings.TrimSpace(transition.AcceptanceReceiptID)
		next.AcceptanceReceiptHash = strings.TrimSpace(transition.AcceptanceReceiptHash)
	}

	return next, nil, nil
}

func reducePlanningOwnerDecision(state, next planningStageState, transition planningStageTransition) (planningStageState, *planningStageManifest, error) {
	if transition.DecisionResolution == nil {
		return planningStageState{}, nil, fmt.Errorf("owner decision transition requires an exact decision resolution")
	}
	resolution := transition.DecisionResolution
	if !resolution.Disposition.valid() {
		return planningStageState{}, nil, fmt.Errorf("owner decision has invalid resolution disposition %q", resolution.Disposition)
	}
	switch resolution.Disposition {
	case planningDecisionDispositionDirectResume:
		if transition.To != state.DecisionResumeStage || (transition.To != planningStageScoutReady && transition.To != planningStageRouteReady) {
			return planningStageState{}, nil, fmt.Errorf("direct owner answer must resume the exact authorized stage %q", state.DecisionResumeStage)
		}
		if len(resolution.AffectedSemanticIDs) != 0 || len(resolution.RevisionEvidence) != 0 {
			return planningStageState{}, nil, fmt.Errorf("direct owner answer cannot carry successor specification evidence")
		}
		next.DecisionResumeStage = ""
		return next, nil, nil

	case planningDecisionDispositionSuccessorSpecRequired:
		if transition.To != planningStageSpecApprovalRequired {
			return planningStageState{}, nil, fmt.Errorf("contract-affecting owner answer must transition to spec_approval_required")
		}
		if len(resolution.AffectedSemanticIDs) == 0 || len(resolution.RevisionEvidence) == 0 {
			return planningStageState{}, nil, fmt.Errorf("successor specification resolution requires affected scope and revision evidence")
		}
		if transition.SuccessorSpecification == nil {
			return planningStageState{}, nil, fmt.Errorf("contract-affecting owner answer requires an exact successor specification draft")
		}
		draft := *transition.SuccessorSpecification
		if err := draft.validate(false); err != nil {
			return planningStageState{}, nil, err
		}
		if draft.Status != colony.SpecStatusDraft || draft.PredecessorRevisionID != state.Specification.RevisionID || draft.RevisionID == state.Specification.RevisionID || draft.ContentHash == state.Specification.ContentHash {
			return planningStageState{}, nil, fmt.Errorf("successor specification must be a distinct draft over the current approved revision")
		}
		next.PendingSpecification = clonePlanningStageSpecificationBinding(&draft)
		next.PendingAffectedSemanticIDs = canonicalPlanningStageIDs(resolution.AffectedSemanticIDs)
		return next, nil, nil
	}
	return planningStageState{}, nil, fmt.Errorf("unsupported owner decision disposition %q", resolution.Disposition)
}

func applyPlanningStageReconciliation(next *planningStageState, reconciliation *planningStageReconciliation) error {
	if next == nil || reconciliation == nil {
		return fmt.Errorf("reconciliation_required transition requires affected-scope reconciliation")
	}
	if reconciliation.SpecificationRevisionID != next.Specification.RevisionID || reconciliation.SpecificationRevisionHash != next.Specification.ContentHash {
		return fmt.Errorf("reconciliation does not bind the exact approved successor specification")
	}
	affected := canonicalPlanningStageIDs(reconciliation.AffectedSemanticIDs)
	if !samePlanningStageIDs(affected, next.PendingAffectedSemanticIDs) {
		return fmt.Errorf("reconciliation affected scope does not match the exact pending scope")
	}
	if !planningSHA256Pattern.MatchString(strings.TrimSpace(reconciliation.InputFrontierHash)) {
		return fmt.Errorf("reconciliation requires a rebound input frontier hash")
	}
	next.InputFrontierHash = reconciliation.InputFrontierHash
	next.PendingSpecification = nil
	next.PendingAffectedSemanticIDs = nil
	next.ScoutReceipt = nil
	next.CandidateSnapshotHash = ""
	next.DecisionResumeStage = ""
	return nil
}

func authorizePlanningStage(state planningStageState, authorization planningStageAuthorization) (planningStageManifest, error) {
	switch state.Stage {
	case planningStageScoutReady:
		if authorization.ExpectedCaste != planningStageCasteScout {
			return planningStageManifest{}, fmt.Errorf("scout_ready expected caste %q, got %q", planningStageCasteScout, authorization.ExpectedCaste)
		}
	case planningStageRouteReady:
		if authorization.ExpectedCaste != planningStageCasteRouteSetter {
			return planningStageManifest{}, fmt.Errorf("route_ready expected caste %q, got %q", planningStageCasteRouteSetter, authorization.ExpectedCaste)
		}
	default:
		return planningStageManifest{}, fmt.Errorf("planning stage %q does not dispatch a worker", state.Stage)
	}
	for _, used := range state.UsedAuthorizationIDs {
		if strings.TrimSpace(used) == strings.TrimSpace(authorization.ID) {
			return planningStageManifest{}, fmt.Errorf("planning authorization %q was already used", authorization.ID)
		}
	}
	if authorization.InputFrontierHash != state.InputFrontierHash {
		return planningStageManifest{}, fmt.Errorf("authorization input frontier does not match the current planning frontier")
	}
	if state.Stage == planningStageRouteReady {
		if state.ScoutReceipt == nil || authorization.ScoutReceipt == nil ||
			state.ScoutReceipt.ID != authorization.ScoutReceipt.ID ||
			state.ScoutReceipt.ContentHash != authorization.ScoutReceipt.ContentHash ||
			state.ScoutReceipt.ManifestHash != authorization.ScoutReceipt.ManifestHash {
			return planningStageManifest{}, fmt.Errorf("Route-Setter authorization does not bind the exact current Scout receipt")
		}
		if !planningSHA256Pattern.MatchString(state.CandidateSnapshotHash) || authorization.CandidateSnapshotHash != state.CandidateSnapshotHash {
			return planningStageManifest{}, fmt.Errorf("Route-Setter authorization does not bind the current candidate snapshot")
		}
	}
	return buildPlanningStageManifest(state, authorization)
}

func buildPlanningStageManifest(state planningStageState, authorization planningStageAuthorization) (planningStageManifest, error) {
	manifest := planningStageManifest{
		SchemaVersion:         planningStageManifestSchemaVersion,
		AuthorizationID:       strings.TrimSpace(authorization.ID),
		RunID:                 strings.TrimSpace(state.RunID),
		Pass:                  state.Pass,
		Preset:                state.Preset,
		Specification:         state.Specification,
		BasePlanRevisionID:    strings.TrimSpace(state.BasePlanRevisionID),
		BasePlanRevisionHash:  strings.TrimSpace(state.BasePlanRevisionHash),
		PriorCardHash:         strings.TrimSpace(state.PriorCardHash),
		InputFrontierHash:     strings.TrimSpace(authorization.InputFrontierHash),
		ExpectedCaste:         authorization.ExpectedCaste,
		ExpectedResultType:    expectedPlanningStageResult(authorization.ExpectedCaste),
		EvidenceFrontier:      clonePlanningStageEvidenceBindings(authorization.EvidenceFrontier),
		WeakestGap:            clonePlanningStageGap(authorization.WeakestGap),
		ScoutReceipt:          clonePlanningStageReceiptRef(authorization.ScoutReceipt),
		CandidateSnapshotHash: strings.TrimSpace(authorization.CandidateSnapshotHash),
	}
	if err := addressPlanningStageManifest(&manifest); err != nil {
		return planningStageManifest{}, err
	}
	return manifest, nil
}

func addressPlanningStageManifest(manifest *planningStageManifest) error {
	if manifest == nil {
		return fmt.Errorf("planning stage manifest is required")
	}
	manifest.SchemaVersion = planningStageManifestSchemaVersion
	manifest.AuthorizationID = strings.TrimSpace(manifest.AuthorizationID)
	manifest.RunID = strings.TrimSpace(manifest.RunID)
	manifest.BasePlanRevisionID = strings.TrimSpace(manifest.BasePlanRevisionID)
	manifest.BasePlanRevisionHash = strings.TrimSpace(manifest.BasePlanRevisionHash)
	manifest.PriorCardHash = strings.TrimSpace(manifest.PriorCardHash)
	manifest.InputFrontierHash = strings.TrimSpace(manifest.InputFrontierHash)
	manifest.CandidateSnapshotHash = strings.TrimSpace(manifest.CandidateSnapshotHash)
	manifest.EvidenceFrontier = canonicalPlanningStageEvidenceBindings(manifest.EvidenceFrontier)
	if err := validatePlanningStageManifestContract(*manifest); err != nil {
		return err
	}
	payload := *manifest
	payload.ID = ""
	payload.ContentHash = ""
	contentHash, err := jsonSHA256(payload)
	if err != nil {
		return fmt.Errorf("hash planning stage manifest: %w", err)
	}
	manifest.ContentHash = contentHash
	manifest.ID = "planning-stage-manifest-" + contentHash[:16]
	return nil
}

func validatePlanningStageManifest(manifest planningStageManifest) error {
	if manifest.SchemaVersion != planningStageManifestSchemaVersion {
		return fmt.Errorf("planning stage manifest schema_version must be %q", planningStageManifestSchemaVersion)
	}
	if !planningSHA256Pattern.MatchString(manifest.ContentHash) || manifest.ID != "planning-stage-manifest-"+manifest.ContentHash[:16] {
		return fmt.Errorf("planning stage manifest requires a matching content-addressed ID and hash")
	}
	if err := validatePlanningStageManifestContract(manifest); err != nil {
		return err
	}
	payload := manifest
	payload.ID = ""
	payload.ContentHash = ""
	wantHash, err := jsonSHA256(payload)
	if err != nil {
		return fmt.Errorf("hash planning stage manifest: %w", err)
	}
	if wantHash != manifest.ContentHash {
		return fmt.Errorf("planning stage manifest content hash does not match its canonical payload")
	}
	return nil
}

func validatePlanningStageManifestContract(manifest planningStageManifest) error {
	if strings.TrimSpace(manifest.AuthorizationID) == "" {
		return fmt.Errorf("planning stage manifest authorization ID is required")
	}
	if strings.TrimSpace(manifest.RunID) == "" {
		return fmt.Errorf("planning stage manifest run ID is required")
	}
	if manifest.Pass <= 0 {
		return fmt.Errorf("planning stage manifest pass must be positive")
	}
	if !manifest.Preset.valid() {
		return fmt.Errorf("planning stage manifest preset is invalid")
	}
	if err := manifest.Specification.validate(true); err != nil {
		return fmt.Errorf("planning stage manifest specification: %w", err)
	}
	if strings.TrimSpace(manifest.BasePlanRevisionID) == "" || !planningSHA256Pattern.MatchString(manifest.BasePlanRevisionHash) {
		return fmt.Errorf("planning stage manifest requires exact base plan revision ID and hash")
	}
	if !planningSHA256Pattern.MatchString(manifest.PriorCardHash) {
		return fmt.Errorf("planning stage manifest requires prior card hash")
	}
	if !planningSHA256Pattern.MatchString(manifest.InputFrontierHash) {
		return fmt.Errorf("planning stage manifest requires input frontier hash")
	}
	if !manifest.ExpectedCaste.valid() || manifest.ExpectedResultType != expectedPlanningStageResult(manifest.ExpectedCaste) {
		return fmt.Errorf("planning stage manifest expected caste and result type do not form one legal worker contract")
	}

	switch manifest.ExpectedCaste {
	case planningStageCasteScout:
		if len(manifest.EvidenceFrontier) == 0 || manifest.WeakestGap == nil {
			return fmt.Errorf("Scout manifest requires evidence frontier and weakest gap")
		}
		if manifest.ScoutReceipt != nil || manifest.CandidateSnapshotHash != "" {
			return fmt.Errorf("Scout manifest contains future-stage Route-Setter fields")
		}
		if err := manifest.WeakestGap.Validate(); err != nil {
			return fmt.Errorf("Scout manifest weakest gap: %w", err)
		}
		seen := make(map[string]struct{}, len(manifest.EvidenceFrontier))
		for index, evidence := range manifest.EvidenceFrontier {
			if err := evidence.validate(); err != nil {
				return fmt.Errorf("Scout manifest evidence_frontier[%d]: %w", index, err)
			}
			if _, duplicate := seen[evidence.ID]; duplicate {
				return fmt.Errorf("Scout manifest evidence frontier contains duplicate %q", evidence.ID)
			}
			seen[evidence.ID] = struct{}{}
		}

	case planningStageCasteRouteSetter:
		if len(manifest.EvidenceFrontier) != 0 || manifest.WeakestGap != nil {
			return fmt.Errorf("Route-Setter manifest contains future-stage Scout authorization fields")
		}
		if manifest.ScoutReceipt == nil {
			return fmt.Errorf("Route-Setter manifest requires the exact Scout receipt")
		}
		if err := manifest.ScoutReceipt.validate(manifest.RunID, manifest.Pass, planningStageCasteScout); err != nil {
			return err
		}
		if !planningSHA256Pattern.MatchString(manifest.CandidateSnapshotHash) {
			return fmt.Errorf("Route-Setter manifest requires current candidate snapshot hash")
		}
	}
	return nil
}

func validatePlanningStageAuthority(state planningStageState) error {
	if strings.TrimSpace(state.RunID) == "" {
		return fmt.Errorf("planning stage run ID is required")
	}
	if state.Pass <= 0 {
		return fmt.Errorf("planning stage pass must be positive")
	}
	if !state.Preset.valid() {
		return fmt.Errorf("planning stage preset is required")
	}
	if err := state.Specification.validate(true); err != nil {
		return fmt.Errorf("planning stage specification: %w", err)
	}
	if strings.TrimSpace(state.BasePlanRevisionID) == "" || !planningSHA256Pattern.MatchString(state.BasePlanRevisionHash) {
		return fmt.Errorf("planning stage base plan revision ID and hash are required")
	}
	if !planningSHA256Pattern.MatchString(state.PriorCardHash) {
		return fmt.Errorf("planning stage prior card hash is required")
	}
	if !planningSHA256Pattern.MatchString(state.InputFrontierHash) {
		return fmt.Errorf("planning stage input frontier hash is required")
	}
	return nil
}

func decodePlanningStageManifest(data []byte) (planningStageManifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest planningStageManifest
	if err := decoder.Decode(&manifest); err != nil {
		return planningStageManifest{}, fmt.Errorf("decode planning stage manifest: %w", err)
	}
	if err := ensurePlanningStageDecoderEOF(decoder); err != nil {
		return planningStageManifest{}, err
	}
	if err := validatePlanningStageManifest(manifest); err != nil {
		return planningStageManifest{}, err
	}
	return manifest, nil
}

func ensurePlanningStageDecoderEOF(decoder *json.Decoder) error {
	var extra interface{}
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing planning stage manifest data: %w", err)
	}
	return fmt.Errorf("planning stage manifest must contain exactly one JSON value")
}

func clonePlanningStageState(state planningStageState) planningStageState {
	state.WeakestGap = clonePlanningStageGap(state.WeakestGap)
	state.ScoutReceipt = clonePlanningStageReceiptRef(state.ScoutReceipt)
	state.UsedAuthorizationIDs = append([]string(nil), state.UsedAuthorizationIDs...)
	state.PendingSpecification = clonePlanningStageSpecificationBinding(state.PendingSpecification)
	state.PendingAffectedSemanticIDs = append([]string(nil), state.PendingAffectedSemanticIDs...)
	return state
}

func clonePlanningStageGap(gap *colony.PlanningGap) *colony.PlanningGap {
	if gap == nil {
		return nil
	}
	clone := *gap
	clone.EvidenceIDs = append([]string(nil), gap.EvidenceIDs...)
	return &clone
}

func clonePlanningStageReceiptRef(ref *planningStageReceiptRef) *planningStageReceiptRef {
	if ref == nil {
		return nil
	}
	clone := *ref
	return &clone
}

func clonePlanningStageSpecificationBinding(binding *planningStageSpecificationBinding) *planningStageSpecificationBinding {
	if binding == nil {
		return nil
	}
	clone := *binding
	return &clone
}

func clonePlanningStageEvidenceBindings(values []planningStageEvidenceBinding) []planningStageEvidenceBinding {
	return append([]planningStageEvidenceBinding(nil), values...)
}

func canonicalPlanningStageEvidenceBindings(values []planningStageEvidenceBinding) []planningStageEvidenceBinding {
	result := clonePlanningStageEvidenceBindings(values)
	for index := range result {
		result[index].ID = strings.TrimSpace(result[index].ID)
		result[index].ContentHash = strings.TrimSpace(result[index].ContentHash)
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].ID < result[right].ID
	})
	return result
}

func canonicalPlanningStageIDs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func samePlanningStageIDs(left, right []string) bool {
	left = canonicalPlanningStageIDs(left)
	right = canonicalPlanningStageIDs(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
