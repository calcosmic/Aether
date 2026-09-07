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

type planningDecisionChoice struct {
	ID                  string                         `json:"id"`
	Label               string                         `json:"label"`
	Consequence         string                         `json:"consequence"`
	Impact              planningDecisionContractImpact `json:"impact"`
	AffectedSemanticIDs []string                       `json:"affected_semantic_ids,omitempty"`
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
	Choices               []planningDecisionChoice       `json:"choices,omitempty"`
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

// planningDecisionEquivalenceScope prevents an answer from crossing a goal,
// session, approved specification, or base-plan revision boundary.
type planningDecisionEquivalenceScope struct {
	GoalID                          string `json:"goal_id"`
	SessionID                       string `json:"session_id"`
	ApprovedSpecificationRevisionID string `json:"approved_specification_revision_id"`
	BasePlanRevisionID              string `json:"base_plan_revision_id"`
}

// planningDecisionEquivalenceKey captures every meaning-bearing dimension of
// an owner decision. ContentHash is derived from the other fields and checked
// again before an answer is reused.
type planningDecisionEquivalenceKey struct {
	GoalID                          string   `json:"goal_id"`
	SessionID                       string   `json:"session_id"`
	ApprovedSpecificationRevisionID string   `json:"approved_specification_revision_id"`
	BasePlanRevisionID              string   `json:"base_plan_revision_id"`
	NormalizedDecisionText          string   `json:"normalized_decision_text"`
	BehaviorImpact                  string   `json:"behavior_impact"`
	AuthorityImpact                 string   `json:"authority_impact,omitempty"`
	ScopeImpact                     string   `json:"scope_impact"`
	RiskImpact                      string   `json:"risk_impact"`
	AcceptanceImpact                string   `json:"acceptance_impact"`
	AffectedSemanticIDs             []string `json:"affected_semantic_ids"`
	ContentHash                     string   `json:"content_hash"`
}

func buildPlanningDecisionEquivalenceKey(scope planningDecisionEquivalenceScope, candidate planningDecisionCandidate) (planningDecisionEquivalenceKey, error) {
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "goal_id", value: scope.GoalID},
		{name: "session_id", value: scope.SessionID},
		{name: "approved_specification_revision_id", value: scope.ApprovedSpecificationRevisionID},
		{name: "base_plan_revision_id", value: scope.BasePlanRevisionID},
		{name: "decision", value: candidate.Decision},
	} {
		if strings.TrimSpace(required.value) == "" {
			return planningDecisionEquivalenceKey{}, fmt.Errorf("%s is required for answer equivalence", required.name)
		}
	}
	key := planningDecisionEquivalenceKey{
		GoalID:                          strings.TrimSpace(scope.GoalID),
		SessionID:                       strings.TrimSpace(scope.SessionID),
		ApprovedSpecificationRevisionID: strings.TrimSpace(scope.ApprovedSpecificationRevisionID),
		BasePlanRevisionID:              strings.TrimSpace(scope.BasePlanRevisionID),
		NormalizedDecisionText:          normalizeDecisionText(candidate.Decision),
		BehaviorImpact:                  normalizeDecisionText(candidate.Impact.Behavior),
		AuthorityImpact:                 normalizeDecisionText(candidate.Impact.Authority),
		ScopeImpact:                     normalizeDecisionText(candidate.Impact.Scope),
		RiskImpact:                      normalizeDecisionText(candidate.Impact.Risk),
		AcceptanceImpact:                normalizeDecisionText(candidate.Impact.Acceptance),
		AffectedSemanticIDs:             nonEmptyPlanningDecisionIDs(candidate.AffectedSemanticIDs),
	}
	contentHash, err := planningDecisionEquivalenceHash(key)
	if err != nil {
		return planningDecisionEquivalenceKey{}, fmt.Errorf("hash planning decision equivalence key: %w", err)
	}
	key.ContentHash = contentHash
	return key, nil
}

func planningDecisionEquivalenceHash(key planningDecisionEquivalenceKey) (string, error) {
	payload := struct {
		GoalID                          string   `json:"goal_id"`
		SessionID                       string   `json:"session_id"`
		ApprovedSpecificationRevisionID string   `json:"approved_specification_revision_id"`
		BasePlanRevisionID              string   `json:"base_plan_revision_id"`
		NormalizedDecisionText          string   `json:"normalized_decision_text"`
		BehaviorImpact                  string   `json:"behavior_impact"`
		AuthorityImpact                 string   `json:"authority_impact,omitempty"`
		ScopeImpact                     string   `json:"scope_impact"`
		RiskImpact                      string   `json:"risk_impact"`
		AcceptanceImpact                string   `json:"acceptance_impact"`
		AffectedSemanticIDs             []string `json:"affected_semantic_ids"`
	}{
		GoalID:                          strings.TrimSpace(key.GoalID),
		SessionID:                       strings.TrimSpace(key.SessionID),
		ApprovedSpecificationRevisionID: strings.TrimSpace(key.ApprovedSpecificationRevisionID),
		BasePlanRevisionID:              strings.TrimSpace(key.BasePlanRevisionID),
		NormalizedDecisionText:          normalizeDecisionText(key.NormalizedDecisionText),
		BehaviorImpact:                  normalizeDecisionText(key.BehaviorImpact),
		AuthorityImpact:                 normalizeDecisionText(key.AuthorityImpact),
		ScopeImpact:                     normalizeDecisionText(key.ScopeImpact),
		RiskImpact:                      normalizeDecisionText(key.RiskImpact),
		AcceptanceImpact:                normalizeDecisionText(key.AcceptanceImpact),
		AffectedSemanticIDs:             nonEmptyPlanningDecisionIDs(key.AffectedSemanticIDs),
	}
	return jsonSHA256(payload)
}

func validatePlanningDecisionEquivalenceKey(key planningDecisionEquivalenceKey) error {
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "goal_id", value: key.GoalID},
		{name: "session_id", value: key.SessionID},
		{name: "approved_specification_revision_id", value: key.ApprovedSpecificationRevisionID},
		{name: "base_plan_revision_id", value: key.BasePlanRevisionID},
		{name: "normalized_decision_text", value: key.NormalizedDecisionText},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("equivalence key %s is required", required.name)
		}
	}
	if !planningSHA256Pattern.MatchString(key.ContentHash) {
		return fmt.Errorf("equivalence key content_hash must be a lowercase SHA-256 digest")
	}
	expected, err := planningDecisionEquivalenceHash(key)
	if err != nil {
		return err
	}
	if key.ContentHash != expected {
		return fmt.Errorf("equivalence key content hash does not match its meaning")
	}
	return nil
}

type planningDecisionAnswerRecord struct {
	DecisionID string                         `json:"decision_id"`
	ChoiceID   string                         `json:"choice_id"`
	Answer     string                         `json:"answer"`
	Key        planningDecisionEquivalenceKey `json:"key"`
}

type planningDecisionAnswerReuse struct {
	Reused               bool   `json:"reused"`
	RequiresRevalidation bool   `json:"requires_revalidation"`
	ChoiceID             string `json:"choice_id,omitempty"`
	Answer               string `json:"answer,omitempty"`
	PriorAnswerEvidence  string `json:"prior_answer_evidence,omitempty"`
	RevalidationReason   string `json:"revalidation_reason,omitempty"`
}

func assessPlanningDecisionAnswerReuse(current planningDecisionEquivalenceKey, prior *planningDecisionAnswerRecord) planningDecisionAnswerReuse {
	if prior == nil {
		return planningDecisionAnswerReuse{
			RequiresRevalidation: true,
			RevalidationReason:   "No prior answer exists for this exact planning scope.",
		}
	}
	currentHash, currentErr := planningDecisionEquivalenceHash(current)
	priorHash, priorErr := planningDecisionEquivalenceHash(prior.Key)
	exact := currentErr == nil && priorErr == nil &&
		current.ContentHash == currentHash && prior.Key.ContentHash == priorHash &&
		current.ContentHash == prior.Key.ContentHash
	if exact {
		return planningDecisionAnswerReuse{
			Reused:   true,
			ChoiceID: strings.TrimSpace(prior.ChoiceID),
			Answer:   normalizePlanningDecisionText(prior.Answer),
		}
	}
	return planningDecisionAnswerReuse{
		RequiresRevalidation: true,
		PriorAnswerEvidence:  normalizePlanningDecisionText(prior.Answer),
		RevalidationReason:   "The prior answer is evidence only because the goal, revisions, meaning, impact, or affected scope changed.",
	}
}

type planningDecisionCardRequest struct {
	Candidate planningDecisionCandidate        `json:"candidate"`
	Scope     planningDecisionEquivalenceScope `json:"scope"`
	Prior     *planningDecisionAnswerRecord    `json:"prior,omitempty"`
}

type planningDecisionCard struct {
	ID                  string                         `json:"id"`
	ContentHash         string                         `json:"content_hash"`
	DecisionID          string                         `json:"decision_id"`
	Decision            string                         `json:"decision"`
	WhyNow              string                         `json:"why_now"`
	Evidence            []colony.PlanningEvidenceRef   `json:"evidence"`
	QueenRecommendation string                         `json:"queen_recommendation"`
	Choices             []planningDecisionChoice       `json:"choices"`
	AffectedSemanticIDs []string                       `json:"affected_semantic_ids"`
	PriorAnswer         string                         `json:"prior_answer"`
	Revalidation        string                         `json:"revalidation"`
	PlanningResumes     string                         `json:"planning_resumes"`
	EquivalenceKey      planningDecisionEquivalenceKey `json:"equivalence_key"`
}

func projectPlanningDecisionCard(request planningDecisionCardRequest) (planningDecisionCard, error) {
	classification, err := classifyPlanningDecision(request.Candidate)
	if err != nil {
		return planningDecisionCard{}, err
	}
	if !classification.RequiresOwner {
		return planningDecisionCard{}, fmt.Errorf("decision %q does not require an owner card", request.Candidate.StableID)
	}
	if len(request.Candidate.Choices) == 0 {
		return planningDecisionCard{}, fmt.Errorf("decision %q requires at least one viable choice", request.Candidate.StableID)
	}
	key, err := buildPlanningDecisionEquivalenceKey(request.Scope, request.Candidate)
	if err != nil {
		return planningDecisionCard{}, err
	}
	candidate := canonicalPlanningDecisionCandidate(request.Candidate)
	choices := append([]planningDecisionChoice(nil), candidate.Choices...)
	seenChoices := make(map[string]struct{}, len(choices))
	for i := range choices {
		choices[i] = canonicalPlanningDecisionChoice(choices[i])
		if err := validatePlanningDecisionChoice(choices[i]); err != nil {
			return planningDecisionCard{}, fmt.Errorf("choices[%d]: %w", i, err)
		}
		if _, duplicate := seenChoices[choices[i].ID]; duplicate {
			return planningDecisionCard{}, fmt.Errorf("duplicate choice ID %q", choices[i].ID)
		}
		seenChoices[choices[i].ID] = struct{}{}
	}
	sort.Slice(choices, func(left, right int) bool { return choices[left].ID < choices[right].ID })

	reuse := assessPlanningDecisionAnswerReuse(key, request.Prior)
	priorAnswer := reuse.PriorAnswerEvidence
	revalidation := reuse.RevalidationReason
	if reuse.Reused {
		priorAnswer = reuse.Answer
		revalidation = "The prior answer exactly matches the current goal, revisions, meaning, impacts, and affected scope."
	}
	if request.Prior == nil {
		revalidation = "No prior answer exists for this exact planning scope; owner authority is required."
	}
	payload := struct {
		DecisionID          string                         `json:"decision_id"`
		Decision            string                         `json:"decision"`
		WhyNow              string                         `json:"why_now"`
		Evidence            []colony.PlanningEvidenceRef   `json:"evidence"`
		QueenRecommendation string                         `json:"queen_recommendation"`
		Choices             []planningDecisionChoice       `json:"choices"`
		AffectedSemanticIDs []string                       `json:"affected_semantic_ids"`
		PriorAnswer         string                         `json:"prior_answer"`
		Revalidation        string                         `json:"revalidation"`
		PlanningResumes     string                         `json:"planning_resumes"`
		EquivalenceKey      planningDecisionEquivalenceKey `json:"equivalence_key"`
	}{
		DecisionID:          candidate.StableID,
		Decision:            candidate.Decision,
		WhyNow:              candidate.WhyNow,
		Evidence:            candidate.Evidence,
		QueenRecommendation: candidate.QueenRecommendation,
		Choices:             choices,
		AffectedSemanticIDs: candidate.AffectedSemanticIDs,
		PriorAnswer:         priorAnswer,
		Revalidation:        revalidation,
		PlanningResumes:     candidate.ResumeInstruction,
		EquivalenceKey:      key,
	}
	contentHash, err := jsonSHA256(payload)
	if err != nil {
		return planningDecisionCard{}, fmt.Errorf("hash planning decision card: %w", err)
	}
	return planningDecisionCard{
		ID:                  "planning-decision-card-" + contentHash[:16],
		ContentHash:         contentHash,
		DecisionID:          payload.DecisionID,
		Decision:            payload.Decision,
		WhyNow:              payload.WhyNow,
		Evidence:            payload.Evidence,
		QueenRecommendation: payload.QueenRecommendation,
		Choices:             payload.Choices,
		AffectedSemanticIDs: payload.AffectedSemanticIDs,
		PriorAnswer:         payload.PriorAnswer,
		Revalidation:        payload.Revalidation,
		PlanningResumes:     payload.PlanningResumes,
		EquivalenceKey:      payload.EquivalenceKey,
	}, nil
}

func canonicalPlanningDecisionChoice(choice planningDecisionChoice) planningDecisionChoice {
	choice.ID = strings.TrimSpace(choice.ID)
	choice.Label = normalizePlanningDecisionText(choice.Label)
	choice.Consequence = normalizePlanningDecisionText(choice.Consequence)
	choice.Impact = planningDecisionContractImpact{
		Behavior:   normalizePlanningDecisionText(choice.Impact.Behavior),
		Authority:  normalizePlanningDecisionText(choice.Impact.Authority),
		Scope:      normalizePlanningDecisionText(choice.Impact.Scope),
		Risk:       normalizePlanningDecisionText(choice.Impact.Risk),
		Acceptance: normalizePlanningDecisionText(choice.Impact.Acceptance),
	}
	choice.AffectedSemanticIDs = nonEmptyPlanningDecisionIDs(choice.AffectedSemanticIDs)
	return choice
}

func validatePlanningDecisionChoice(choice planningDecisionChoice) error {
	if strings.TrimSpace(choice.ID) == "" {
		return fmt.Errorf("choice ID is required")
	}
	if strings.TrimSpace(choice.Label) == "" {
		return fmt.Errorf("choice label is required")
	}
	if strings.TrimSpace(choice.Consequence) == "" {
		return fmt.Errorf("choice consequence is required")
	}
	return nil
}

type planningDecisionResolutionDisposition string

const (
	planningDecisionDispositionDirectResume          planningDecisionResolutionDisposition = "direct_resume"
	planningDecisionDispositionSuccessorSpecRequired planningDecisionResolutionDisposition = "successor_spec_required"
)

func (d planningDecisionResolutionDisposition) valid() bool {
	return d == planningDecisionDispositionDirectResume || d == planningDecisionDispositionSuccessorSpecRequired
}

type planningDecisionRevisionEvidence struct {
	Dimension           string   `json:"dimension"`
	ApprovedValue       string   `json:"approved_value,omitempty"`
	SelectedValue       string   `json:"selected_value"`
	AffectedSemanticIDs []string `json:"affected_semantic_ids,omitempty"`
}

type planningDecisionResolutionRequest struct {
	DecisionID     string                         `json:"decision_id"`
	ApprovedImpact planningDecisionContractImpact `json:"approved_impact"`
	SelectedChoice planningDecisionChoice         `json:"selected_choice"`
}

type planningDecisionResolution struct {
	Disposition         planningDecisionResolutionDisposition `json:"disposition"`
	AffectedSemanticIDs []string                              `json:"affected_semantic_ids"`
	RevisionEvidence    []planningDecisionRevisionEvidence    `json:"revision_evidence"`
}

func resolvePlanningDecisionAnswer(request planningDecisionResolutionRequest) (planningDecisionResolution, error) {
	if strings.TrimSpace(request.DecisionID) == "" {
		return planningDecisionResolution{}, fmt.Errorf("decision ID is required")
	}
	selected := canonicalPlanningDecisionChoice(request.SelectedChoice)
	if err := validatePlanningDecisionChoice(selected); err != nil {
		return planningDecisionResolution{}, err
	}
	approved := canonicalPlanningDecisionImpact(request.ApprovedImpact)
	revisionEvidence := planningDecisionImpactDifferences(approved, selected.Impact)
	affectedIDs := nonEmptyPlanningDecisionIDs(selected.AffectedSemanticIDs)
	if len(affectedIDs) > 0 {
		revisionEvidence = append(revisionEvidence, planningDecisionRevisionEvidence{
			Dimension:           "affected_specification_items",
			SelectedValue:       strings.Join(affectedIDs, ","),
			AffectedSemanticIDs: affectedIDs,
		})
	}
	if len(revisionEvidence) == 0 {
		return planningDecisionResolution{Disposition: planningDecisionDispositionDirectResume}, nil
	}
	sort.Slice(revisionEvidence, func(left, right int) bool {
		return revisionEvidence[left].Dimension < revisionEvidence[right].Dimension
	})
	return planningDecisionResolution{
		Disposition:         planningDecisionDispositionSuccessorSpecRequired,
		AffectedSemanticIDs: affectedIDs,
		RevisionEvidence:    revisionEvidence,
	}, nil
}

func canonicalPlanningDecisionImpact(impact planningDecisionContractImpact) planningDecisionContractImpact {
	return planningDecisionContractImpact{
		Behavior:   normalizeDecisionText(impact.Behavior),
		Authority:  normalizeDecisionText(impact.Authority),
		Scope:      normalizeDecisionText(impact.Scope),
		Risk:       normalizeDecisionText(impact.Risk),
		Acceptance: normalizeDecisionText(impact.Acceptance),
	}
}

func planningDecisionImpactDifferences(approved, selected planningDecisionContractImpact) []planningDecisionRevisionEvidence {
	selected = canonicalPlanningDecisionImpact(selected)
	fields := []struct {
		dimension string
		before    string
		after     string
	}{
		{dimension: "acceptance", before: approved.Acceptance, after: selected.Acceptance},
		{dimension: "authority", before: approved.Authority, after: selected.Authority},
		{dimension: "behavior", before: approved.Behavior, after: selected.Behavior},
		{dimension: "risk", before: approved.Risk, after: selected.Risk},
		{dimension: "scope", before: approved.Scope, after: selected.Scope},
	}
	result := make([]planningDecisionRevisionEvidence, 0, len(fields))
	for _, field := range fields {
		if field.before == field.after {
			continue
		}
		result = append(result, planningDecisionRevisionEvidence{
			Dimension:     field.dimension,
			ApprovedValue: field.before,
			SelectedValue: field.after,
		})
	}
	return result
}

type planningDecisionBoundAnswer struct {
	DecisionID     string                         `json:"decision_id"`
	ChoiceID       string                         `json:"choice_id"`
	Answer         string                         `json:"answer"`
	EquivalenceKey planningDecisionEquivalenceKey `json:"equivalence_key"`
}

type planningDecisionResumeBinding struct {
	GoalID              string                                `json:"goal_id"`
	SessionID           string                                `json:"session_id"`
	BatchID             string                                `json:"batch_id"`
	BatchHash           string                                `json:"batch_hash"`
	FrontierReceiptHash string                                `json:"frontier_receipt_hash,omitempty"`
	CompletedCardHash   string                                `json:"completed_card_hash,omitempty"`
	Answers             []planningDecisionBoundAnswer         `json:"answers"`
	Disposition         planningDecisionResolutionDisposition `json:"disposition"`
	AffectedSemanticIDs []string                              `json:"affected_semantic_ids,omitempty"`
	RevisionEvidence    []planningDecisionRevisionEvidence    `json:"revision_evidence,omitempty"`
	RecoveryCommand     string                                `json:"recovery_command"`
}

type planningDecisionResumeToken struct {
	ID                  string                                `json:"id"`
	ContentHash         string                                `json:"content_hash"`
	GoalID              string                                `json:"goal_id"`
	SessionID           string                                `json:"session_id"`
	BatchID             string                                `json:"batch_id"`
	BatchHash           string                                `json:"batch_hash"`
	FrontierReceiptHash string                                `json:"frontier_receipt_hash,omitempty"`
	CompletedCardHash   string                                `json:"completed_card_hash,omitempty"`
	Answers             []planningDecisionBoundAnswer         `json:"answers"`
	Disposition         planningDecisionResolutionDisposition `json:"disposition"`
	AffectedSemanticIDs []string                              `json:"affected_semantic_ids,omitempty"`
	RevisionEvidence    []planningDecisionRevisionEvidence    `json:"revision_evidence,omitempty"`
	RecoveryCommand     string                                `json:"recovery_command"`
}

func issuePlanningDecisionResumeToken(binding planningDecisionResumeBinding) (planningDecisionResumeToken, error) {
	binding = canonicalPlanningDecisionResumeBinding(binding)
	if err := validatePlanningDecisionResumeBinding(binding); err != nil {
		return planningDecisionResumeToken{}, err
	}
	contentHash, err := jsonSHA256(binding)
	if err != nil {
		return planningDecisionResumeToken{}, fmt.Errorf("hash planning decision resume token: %w", err)
	}
	return planningDecisionResumeToken{
		ID:                  "planning-decision-resume-" + contentHash[:16],
		ContentHash:         contentHash,
		GoalID:              binding.GoalID,
		SessionID:           binding.SessionID,
		BatchID:             binding.BatchID,
		BatchHash:           binding.BatchHash,
		FrontierReceiptHash: binding.FrontierReceiptHash,
		CompletedCardHash:   binding.CompletedCardHash,
		Answers:             binding.Answers,
		Disposition:         binding.Disposition,
		AffectedSemanticIDs: binding.AffectedSemanticIDs,
		RevisionEvidence:    binding.RevisionEvidence,
		RecoveryCommand:     binding.RecoveryCommand,
	}, nil
}

type planningDecisionResumeStatus string

const (
	planningDecisionResumeValid     planningDecisionResumeStatus = "valid"
	planningDecisionResumeStale     planningDecisionResumeStatus = "stale"
	planningDecisionResumeDivergent planningDecisionResumeStatus = "divergent"
)

type planningDecisionResumeValidation struct {
	Accepted        bool                         `json:"accepted"`
	Idempotent      bool                         `json:"idempotent"`
	Status          planningDecisionResumeStatus `json:"status"`
	RecoveryCommand string                       `json:"recovery_command"`
}

// validatePlanningDecisionResumeToken compares immutable values only. It does
// not read or mutate planning state, so retries are safe and deterministic.
func validatePlanningDecisionResumeToken(token planningDecisionResumeToken, expected planningDecisionResumeBinding) (planningDecisionResumeValidation, error) {
	recovery := normalizePlanningDecisionText(expected.RecoveryCommand)
	if recovery == "" {
		return planningDecisionResumeValidation{}, fmt.Errorf("expected recovery command is required")
	}
	presentedBinding := planningDecisionResumeBindingFromToken(token)
	rebuilt, rebuildErr := issuePlanningDecisionResumeToken(presentedBinding)
	if rebuildErr != nil || rebuilt.ID != token.ID || rebuilt.ContentHash != token.ContentHash {
		return planningDecisionResumeValidation{
			Status:          planningDecisionResumeDivergent,
			RecoveryCommand: recovery,
		}, nil
	}

	if planningDecisionResumeBoundaryChanged(presentedBinding, expected) {
		return planningDecisionResumeValidation{
			Status:          planningDecisionResumeStale,
			RecoveryCommand: recovery,
		}, nil
	}
	expectedToken, err := issuePlanningDecisionResumeToken(expected)
	if err != nil {
		return planningDecisionResumeValidation{}, fmt.Errorf("expected resume binding: %w", err)
	}
	if expectedToken.ContentHash != token.ContentHash || expectedToken.ID != token.ID {
		return planningDecisionResumeValidation{
			Status:          planningDecisionResumeDivergent,
			RecoveryCommand: recovery,
		}, nil
	}
	return planningDecisionResumeValidation{
		Accepted:        true,
		Idempotent:      true,
		Status:          planningDecisionResumeValid,
		RecoveryCommand: recovery,
	}, nil
}

func planningDecisionResumeBindingFromToken(token planningDecisionResumeToken) planningDecisionResumeBinding {
	return planningDecisionResumeBinding{
		GoalID:              token.GoalID,
		SessionID:           token.SessionID,
		BatchID:             token.BatchID,
		BatchHash:           token.BatchHash,
		FrontierReceiptHash: token.FrontierReceiptHash,
		CompletedCardHash:   token.CompletedCardHash,
		Answers:             append([]planningDecisionBoundAnswer(nil), token.Answers...),
		Disposition:         token.Disposition,
		AffectedSemanticIDs: append([]string(nil), token.AffectedSemanticIDs...),
		RevisionEvidence:    append([]planningDecisionRevisionEvidence(nil), token.RevisionEvidence...),
		RecoveryCommand:     token.RecoveryCommand,
	}
}

func canonicalPlanningDecisionResumeBinding(binding planningDecisionResumeBinding) planningDecisionResumeBinding {
	binding.GoalID = strings.TrimSpace(binding.GoalID)
	binding.SessionID = strings.TrimSpace(binding.SessionID)
	binding.BatchID = strings.TrimSpace(binding.BatchID)
	binding.BatchHash = strings.TrimSpace(binding.BatchHash)
	binding.FrontierReceiptHash = strings.TrimSpace(binding.FrontierReceiptHash)
	binding.CompletedCardHash = strings.TrimSpace(binding.CompletedCardHash)
	binding.RecoveryCommand = normalizePlanningDecisionText(binding.RecoveryCommand)
	binding.AffectedSemanticIDs = nonEmptyPlanningDecisionIDs(binding.AffectedSemanticIDs)
	binding.Answers = append([]planningDecisionBoundAnswer(nil), binding.Answers...)
	for i := range binding.Answers {
		binding.Answers[i].DecisionID = strings.TrimSpace(binding.Answers[i].DecisionID)
		binding.Answers[i].ChoiceID = strings.TrimSpace(binding.Answers[i].ChoiceID)
		binding.Answers[i].Answer = normalizePlanningDecisionText(binding.Answers[i].Answer)
	}
	sort.Slice(binding.Answers, func(left, right int) bool {
		return binding.Answers[left].DecisionID < binding.Answers[right].DecisionID
	})
	binding.RevisionEvidence = append([]planningDecisionRevisionEvidence(nil), binding.RevisionEvidence...)
	for i := range binding.RevisionEvidence {
		binding.RevisionEvidence[i].Dimension = strings.TrimSpace(binding.RevisionEvidence[i].Dimension)
		binding.RevisionEvidence[i].ApprovedValue = normalizeDecisionText(binding.RevisionEvidence[i].ApprovedValue)
		binding.RevisionEvidence[i].SelectedValue = normalizeDecisionText(binding.RevisionEvidence[i].SelectedValue)
		binding.RevisionEvidence[i].AffectedSemanticIDs = nonEmptyPlanningDecisionIDs(binding.RevisionEvidence[i].AffectedSemanticIDs)
	}
	sort.Slice(binding.RevisionEvidence, func(left, right int) bool {
		return binding.RevisionEvidence[left].Dimension < binding.RevisionEvidence[right].Dimension
	})
	return binding
}

func validatePlanningDecisionResumeBinding(binding planningDecisionResumeBinding) error {
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "goal_id", value: binding.GoalID},
		{name: "session_id", value: binding.SessionID},
		{name: "batch_id", value: binding.BatchID},
		{name: "recovery_command", value: binding.RecoveryCommand},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("resume binding %s is required", required.name)
		}
	}
	if !planningSHA256Pattern.MatchString(binding.BatchHash) {
		return fmt.Errorf("resume binding batch_hash must be a lowercase SHA-256 digest")
	}
	frontierSet := planningSHA256Pattern.MatchString(binding.FrontierReceiptHash)
	cardSet := planningSHA256Pattern.MatchString(binding.CompletedCardHash)
	if frontierSet == cardSet {
		return fmt.Errorf("resume binding requires exactly one frontier receipt or completed-card hash")
	}
	if len(binding.Answers) == 0 {
		return fmt.Errorf("resume binding requires owner answers")
	}
	seen := make(map[string]struct{}, len(binding.Answers))
	for i, answer := range binding.Answers {
		if strings.TrimSpace(answer.DecisionID) == "" || strings.TrimSpace(answer.ChoiceID) == "" || strings.TrimSpace(answer.Answer) == "" {
			return fmt.Errorf("answers[%d] requires decision, choice, and answer", i)
		}
		if _, duplicate := seen[answer.DecisionID]; duplicate {
			return fmt.Errorf("duplicate bound answer for decision %q", answer.DecisionID)
		}
		seen[answer.DecisionID] = struct{}{}
		if err := validatePlanningDecisionEquivalenceKey(answer.EquivalenceKey); err != nil {
			return fmt.Errorf("answers[%d].equivalence_key: %w", i, err)
		}
		if answer.EquivalenceKey.GoalID != binding.GoalID || answer.EquivalenceKey.SessionID != binding.SessionID {
			return fmt.Errorf("answers[%d] crosses goal or session scope", i)
		}
	}
	if !binding.Disposition.valid() {
		return fmt.Errorf("invalid resume disposition %q", binding.Disposition)
	}
	if binding.Disposition == planningDecisionDispositionDirectResume {
		if len(binding.AffectedSemanticIDs) != 0 || len(binding.RevisionEvidence) != 0 {
			return fmt.Errorf("direct_resume cannot carry successor specification evidence")
		}
	} else if len(binding.AffectedSemanticIDs) == 0 || len(binding.RevisionEvidence) == 0 {
		return fmt.Errorf("successor_spec_required requires affected IDs and revision evidence")
	}
	return nil
}

func planningDecisionResumeBoundaryChanged(presented, expected planningDecisionResumeBinding) bool {
	return presented.GoalID != strings.TrimSpace(expected.GoalID) ||
		presented.SessionID != strings.TrimSpace(expected.SessionID) ||
		presented.BatchID != strings.TrimSpace(expected.BatchID) ||
		presented.BatchHash != strings.TrimSpace(expected.BatchHash) ||
		presented.FrontierReceiptHash != strings.TrimSpace(expected.FrontierReceiptHash) ||
		presented.CompletedCardHash != strings.TrimSpace(expected.CompletedCardHash)
}
