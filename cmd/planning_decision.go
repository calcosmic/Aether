package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planningDecisionDomain names the only five subjects for which iterative
// planning may require owner authority. The non-material values make the
// autonomous boundary explicit so research and presentation mechanics cannot
// accidentally become owner prompts.
type planningDecisionDomain string

const (
	planningDecisionDomainBehavior          planningDecisionDomain = "behavior"
	planningDecisionDomainAuthority         planningDecisionDomain = "authority"
	planningDecisionDomainRiskTolerance     planningDecisionDomain = "risk_tolerance"
	planningDecisionDomainScope             planningDecisionDomain = "scope"
	planningDecisionDomainAcceptanceMeaning planningDecisionDomain = "acceptance_meaning"

	planningDecisionDomainRoutineResearch planningDecisionDomain = "routine_research"
	planningDecisionDomainWorkerMechanics planningDecisionDomain = "worker_mechanics"
	planningDecisionDomainScoring         planningDecisionDomain = "scoring"
	planningDecisionDomainPlanFormatting  planningDecisionDomain = "plan_formatting"
)

func (d planningDecisionDomain) isMaterialDomain() bool {
	switch d {
	case planningDecisionDomainBehavior,
		planningDecisionDomainAuthority,
		planningDecisionDomainRiskTolerance,
		planningDecisionDomainScope,
		planningDecisionDomainAcceptanceMeaning:
		return true
	default:
		return false
	}
}

func (d planningDecisionDomain) isRoutineDomain() bool {
	switch d {
	case planningDecisionDomainRoutineResearch,
		planningDecisionDomainWorkerMechanics,
		planningDecisionDomainScoring,
		planningDecisionDomainPlanFormatting:
		return true
	default:
		return false
	}
}

type planningDecisionClassificationReason string

const (
	planningDecisionReasonMaterial           planningDecisionClassificationReason = "material"
	planningDecisionReasonEvidenceAnswerable planningDecisionClassificationReason = "evidence_answerable"
	planningDecisionReasonGenericPreference  planningDecisionClassificationReason = "generic_preference"
	planningDecisionReasonRoutineAutonomous  planningDecisionClassificationReason = "routine_autonomous"
)

type planningDecisionContractImpact struct {
	Behavior   string `json:"behavior,omitempty"`
	Authority  string `json:"authority,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Risk       string `json:"risk,omitempty"`
	Acceptance string `json:"acceptance,omitempty"`
}

func (i planningDecisionContractImpact) material() bool {
	return strings.TrimSpace(i.Behavior) != "" ||
		strings.TrimSpace(i.Authority) != "" ||
		strings.TrimSpace(i.Scope) != "" ||
		strings.TrimSpace(i.Risk) != "" ||
		strings.TrimSpace(i.Acceptance) != ""
}

type planningDecisionCandidate struct {
	StableID              string                         `json:"stable_id"`
	Domain                planningDecisionDomain         `json:"domain"`
	Decision              string                         `json:"decision"`
	WhyNow                string                         `json:"why_now"`
	Evidence              []colony.PlanningEvidenceRef   `json:"evidence"`
	AnswerableEvidenceIDs []string                       `json:"answerable_evidence_ids,omitempty"`
	QueenRecommendation   string                         `json:"queen_recommendation"`
	Impact                planningDecisionContractImpact `json:"impact"`
	AffectedSemanticIDs   []string                       `json:"affected_semantic_ids"`
	ResumeInstruction     string                         `json:"resume_instruction"`
}

type planningDecisionClassification struct {
	RequiresOwner bool                                 `json:"requires_owner"`
	Reason        planningDecisionClassificationReason `json:"reason"`
}

// classifyPlanningDecision is deliberately pure: evidence and impact determine
// authority, never the renderer or the order in which a worker found a gap.
func classifyPlanningDecision(candidate planningDecisionCandidate) (planningDecisionClassification, error) {
	if strings.TrimSpace(candidate.StableID) == "" {
		return planningDecisionClassification{}, fmt.Errorf("stable decision ID is required")
	}
	if strings.TrimSpace(candidate.Decision) == "" {
		return planningDecisionClassification{}, fmt.Errorf("exact decision text is required")
	}
	if candidate.Domain.isRoutineDomain() {
		return planningDecisionClassification{Reason: planningDecisionReasonRoutineAutonomous}, nil
	}
	if !candidate.Domain.isMaterialDomain() {
		return planningDecisionClassification{}, fmt.Errorf("unknown planning decision domain %q", candidate.Domain)
	}
	if !candidate.Impact.material() && len(nonEmptyPlanningDecisionIDs(candidate.AffectedSemanticIDs)) == 0 {
		return planningDecisionClassification{Reason: planningDecisionReasonGenericPreference}, nil
	}

	evidenceByID := make(map[string]colony.PlanningEvidenceRef, len(candidate.Evidence))
	for i, evidence := range candidate.Evidence {
		if err := evidence.Validate(); err != nil {
			return planningDecisionClassification{}, fmt.Errorf("evidence[%d]: %w", i, err)
		}
		if _, duplicate := evidenceByID[evidence.ID]; duplicate {
			return planningDecisionClassification{}, fmt.Errorf("duplicate evidence ID %q", evidence.ID)
		}
		evidenceByID[evidence.ID] = evidence
	}
	for _, evidenceID := range nonEmptyPlanningDecisionIDs(candidate.AnswerableEvidenceIDs) {
		evidence, ok := evidenceByID[evidenceID]
		if !ok {
			return planningDecisionClassification{}, fmt.Errorf("answerable evidence %q is not cited", evidenceID)
		}
		if evidence.Fresh && evidence.Admissible {
			return planningDecisionClassification{Reason: planningDecisionReasonEvidenceAnswerable}, nil
		}
	}
	if len(candidate.Evidence) == 0 {
		return planningDecisionClassification{}, fmt.Errorf("material decision %q requires cited evidence", candidate.StableID)
	}
	if strings.TrimSpace(candidate.WhyNow) == "" {
		return planningDecisionClassification{}, fmt.Errorf("material decision %q requires why-now context", candidate.StableID)
	}
	if strings.TrimSpace(candidate.QueenRecommendation) == "" {
		return planningDecisionClassification{}, fmt.Errorf("material decision %q requires a Queen recommendation", candidate.StableID)
	}
	if strings.TrimSpace(candidate.ResumeInstruction) == "" {
		return planningDecisionClassification{}, fmt.Errorf("material decision %q requires a resume instruction", candidate.StableID)
	}

	return planningDecisionClassification{RequiresOwner: true, Reason: planningDecisionReasonMaterial}, nil
}

type planningDecisionStageReceipt struct {
	ID          string `json:"id"`
	ContentHash string `json:"content_hash"`
	RunID       string `json:"run_id"`
	Pass        int    `json:"pass"`
}

type planningDecisionIterationCardReceipt struct {
	ID                     string `json:"id"`
	ContentHash            string `json:"content_hash"`
	RunID                  string `json:"run_id"`
	Pass                   int    `json:"pass"`
	ScoutReceiptID         string `json:"scout_receipt_id"`
	ScoutReceiptHash       string `json:"scout_receipt_hash"`
	RouteSetterReceiptID   string `json:"route_setter_receipt_id"`
	RouteSetterReceiptHash string `json:"route_setter_receipt_hash"`
}

type planningDecisionBoundary struct {
	RunID              string                                `json:"run_id"`
	Pass               int                                   `json:"pass"`
	ScoutReceipt       planningDecisionStageReceipt          `json:"scout_receipt"`
	RouteSetterReceipt *planningDecisionStageReceipt         `json:"route_setter_receipt,omitempty"`
	IterationCard      *planningDecisionIterationCardReceipt `json:"iteration_card,omitempty"`
	RecoveryCommand    string                                `json:"recovery_command"`
}

type planningDecisionBatch struct {
	ID               string                      `json:"id"`
	ContentHash      string                      `json:"content_hash"`
	RunID            string                      `json:"run_id"`
	Pass             int                         `json:"pass"`
	ScoutReceiptHash string                      `json:"scout_receipt_hash"`
	BoundaryCardHash string                      `json:"boundary_card_hash,omitempty"`
	Decisions        []planningDecisionCandidate `json:"decisions"`
	RecoveryCommand  string                      `json:"recovery_command"`
}

// buildPlanningDecisionBatch returns at most one batch. Pass one may stop at
// the post-Scout boundary; every later pass must first prove that Scout,
// Route-Setter, and the immutable iteration card all completed together.
func buildPlanningDecisionBatch(boundary planningDecisionBoundary, candidates []planningDecisionCandidate) (*planningDecisionBatch, error) {
	material := make([]planningDecisionCandidate, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		classification, err := classifyPlanningDecision(candidate)
		if err != nil {
			return nil, fmt.Errorf("classify %q: %w", candidate.StableID, err)
		}
		if !classification.RequiresOwner {
			continue
		}
		candidate = canonicalPlanningDecisionCandidate(candidate)
		if _, duplicate := seen[candidate.StableID]; duplicate {
			return nil, fmt.Errorf("duplicate material decision ID %q", candidate.StableID)
		}
		seen[candidate.StableID] = struct{}{}
		material = append(material, candidate)
	}
	if len(material) == 0 {
		return nil, nil
	}
	if err := validatePlanningDecisionBoundary(boundary); err != nil {
		return nil, err
	}

	sort.Slice(material, func(left, right int) bool {
		leftRank := planningDecisionDomainRank(material[left].Domain)
		rightRank := planningDecisionDomainRank(material[right].Domain)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return material[left].StableID < material[right].StableID
	})

	boundaryCardHash := ""
	if boundary.IterationCard != nil {
		boundaryCardHash = boundary.IterationCard.ContentHash
	}
	payload := struct {
		RunID            string                      `json:"run_id"`
		Pass             int                         `json:"pass"`
		ScoutReceiptHash string                      `json:"scout_receipt_hash"`
		BoundaryCardHash string                      `json:"boundary_card_hash,omitempty"`
		Decisions        []planningDecisionCandidate `json:"decisions"`
	}{
		RunID:            strings.TrimSpace(boundary.RunID),
		Pass:             boundary.Pass,
		ScoutReceiptHash: boundary.ScoutReceipt.ContentHash,
		BoundaryCardHash: boundaryCardHash,
		Decisions:        material,
	}
	contentHash, err := jsonSHA256(payload)
	if err != nil {
		return nil, fmt.Errorf("hash planning decision batch: %w", err)
	}
	return &planningDecisionBatch{
		ID:               "planning-decision-batch-" + contentHash[:16],
		ContentHash:      contentHash,
		RunID:            payload.RunID,
		Pass:             payload.Pass,
		ScoutReceiptHash: payload.ScoutReceiptHash,
		BoundaryCardHash: payload.BoundaryCardHash,
		Decisions:        material,
		RecoveryCommand:  normalizePlanningDecisionText(boundary.RecoveryCommand),
	}, nil
}

func validatePlanningDecisionBoundary(boundary planningDecisionBoundary) error {
	if strings.TrimSpace(boundary.RunID) == "" {
		return fmt.Errorf("planning decision boundary requires run ID")
	}
	if boundary.Pass <= 0 {
		return fmt.Errorf("planning decision boundary pass must be positive")
	}
	if strings.TrimSpace(boundary.RecoveryCommand) == "" {
		return fmt.Errorf("planning decision boundary requires recovery command")
	}
	if err := validatePlanningDecisionStageReceipt("Scout", boundary.ScoutReceipt, boundary.RunID, boundary.Pass); err != nil {
		return err
	}
	if boundary.Pass == 1 {
		return nil
	}
	if boundary.RouteSetterReceipt == nil {
		return fmt.Errorf("later planning decision boundary requires completed route-setter receipt")
	}
	if err := validatePlanningDecisionStageReceipt("route-setter", *boundary.RouteSetterReceipt, boundary.RunID, boundary.Pass); err != nil {
		return err
	}
	if boundary.IterationCard == nil {
		return fmt.Errorf("later planning decision boundary requires persisted iteration card")
	}
	card := boundary.IterationCard
	if strings.TrimSpace(card.ID) == "" || !planningSHA256Pattern.MatchString(card.ContentHash) {
		return fmt.Errorf("persisted iteration card requires ID and SHA-256 content hash")
	}
	if card.RunID != boundary.RunID || card.Pass != boundary.Pass {
		return fmt.Errorf("persisted iteration card does not match planning run and pass")
	}
	if card.ScoutReceiptID != boundary.ScoutReceipt.ID || card.ScoutReceiptHash != boundary.ScoutReceipt.ContentHash {
		return fmt.Errorf("persisted iteration card does not bind the Scout receipt")
	}
	if card.RouteSetterReceiptID != boundary.RouteSetterReceipt.ID || card.RouteSetterReceiptHash != boundary.RouteSetterReceipt.ContentHash {
		return fmt.Errorf("persisted iteration card does not bind the route-setter receipt")
	}
	return nil
}

func validatePlanningDecisionStageReceipt(stage string, receipt planningDecisionStageReceipt, runID string, pass int) error {
	if strings.TrimSpace(receipt.ID) == "" || !planningSHA256Pattern.MatchString(receipt.ContentHash) {
		return fmt.Errorf("%s receipt requires ID and SHA-256 content hash", stage)
	}
	if receipt.RunID != runID || receipt.Pass != pass {
		return fmt.Errorf("%s receipt does not match planning run and pass", stage)
	}
	return nil
}

func canonicalPlanningDecisionCandidate(candidate planningDecisionCandidate) planningDecisionCandidate {
	candidate.StableID = strings.TrimSpace(candidate.StableID)
	candidate.Decision = normalizePlanningDecisionText(candidate.Decision)
	candidate.WhyNow = normalizePlanningDecisionText(candidate.WhyNow)
	candidate.QueenRecommendation = normalizePlanningDecisionText(candidate.QueenRecommendation)
	candidate.ResumeInstruction = normalizePlanningDecisionText(candidate.ResumeInstruction)
	candidate.Impact = planningDecisionContractImpact{
		Behavior:   normalizePlanningDecisionText(candidate.Impact.Behavior),
		Authority:  normalizePlanningDecisionText(candidate.Impact.Authority),
		Scope:      normalizePlanningDecisionText(candidate.Impact.Scope),
		Risk:       normalizePlanningDecisionText(candidate.Impact.Risk),
		Acceptance: normalizePlanningDecisionText(candidate.Impact.Acceptance),
	}
	candidate.AnswerableEvidenceIDs = nonEmptyPlanningDecisionIDs(candidate.AnswerableEvidenceIDs)
	candidate.AffectedSemanticIDs = nonEmptyPlanningDecisionIDs(candidate.AffectedSemanticIDs)
	candidate.Evidence = append([]colony.PlanningEvidenceRef(nil), candidate.Evidence...)
	sort.Slice(candidate.Evidence, func(left, right int) bool {
		return candidate.Evidence[left].ID < candidate.Evidence[right].ID
	})
	return candidate
}

func nonEmptyPlanningDecisionIDs(values []string) []string {
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

func normalizePlanningDecisionText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func planningDecisionDomainRank(domain planningDecisionDomain) int {
	switch domain {
	case planningDecisionDomainBehavior:
		return 0
	case planningDecisionDomainAuthority:
		return 1
	case planningDecisionDomainRiskTolerance:
		return 2
	case planningDecisionDomainScope:
		return 3
	case planningDecisionDomainAcceptanceMeaning:
		return 4
	default:
		return 5
	}
}
