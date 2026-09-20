package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planningConfidenceScores is the Go-owned five-dimensional frontier plus its
// derived weighted overall. Route-Setter proposes only the five dimension
// values; Overall is always recomputed by evaluatePlanningConfidence.
type planningConfidenceScores struct {
	Knowledge    int `json:"knowledge"`
	Requirements int `json:"requirements"`
	Risks        int `json:"risks"`
	Dependencies int `json:"dependencies"`
	Effort       int `json:"effort"`
	Overall      int `json:"overall"`
}

func (s planningConfidenceScores) Value(dimension colony.PlanningDimension) int {
	switch dimension {
	case colony.PlanningDimensionKnowledge:
		return s.Knowledge
	case colony.PlanningDimensionRequirements:
		return s.Requirements
	case colony.PlanningDimensionRisks:
		return s.Risks
	case colony.PlanningDimensionDependencies:
		return s.Dependencies
	case colony.PlanningDimensionEffort:
		return s.Effort
	default:
		return 0
	}
}

func (s *planningConfidenceScores) Set(dimension colony.PlanningDimension, value int) {
	if s == nil {
		return
	}
	switch dimension {
	case colony.PlanningDimensionKnowledge:
		s.Knowledge = value
	case colony.PlanningDimensionRequirements:
		s.Requirements = value
	case colony.PlanningDimensionRisks:
		s.Risks = value
	case colony.PlanningDimensionDependencies:
		s.Dependencies = value
	case colony.PlanningDimensionEffort:
		s.Effort = value
	default:
		return
	}
	s.Overall = planningConfidenceWeightedOverall(*s)
}

// planningConfidenceProposal is the complete pure-policy input at a
// Route-Setter boundary. SuppliedOverall exists only so the trust boundary can
// reject a worker-authored overall explicitly instead of silently trusting it.
type planningConfidenceProposal struct {
	PriorScores       planningConfidenceScores             `json:"prior_scores"`
	Assessments       []colony.PlanningDimensionAssessment `json:"assessments"`
	EvidenceCatalogue []colony.PlanningEvidenceRef         `json:"evidence_catalogue"`
	EvidenceFrontier  planningEvidenceFrontier             `json:"evidence_frontier"`
	SuppliedOverall   *int                                 `json:"supplied_overall,omitempty"`
}

// planningConfidenceEvaluation contains only validated semantic scoring
// output. Authority impacts remain in PlanningSemanticDelta.AuthorityImpacts;
// this evaluator neither accepts nor produces an authority transition.
type planningConfidenceEvaluation struct {
	Scores      planningConfidenceScores             `json:"scores"`
	Assessments []colony.PlanningDimensionAssessment `json:"assessments"`
	RankedGaps  []colony.PlanningGap                 `json:"ranked_gaps"`
	WeakestGap  colony.PlanningGap                   `json:"weakest_gap"`
}

const planningDiminishingPolicyGroundedTwoPassLT2 = "grounded_two_pass_lt2"

// planningConfidencePass is the policy-relevant result of one completed,
// validated Route-Setter pass. SemanticDelta carries executable changes;
// authority impacts remain in its separate AuthorityImpacts collection.
type planningConfidencePass struct {
	Iteration     int                          `json:"iteration"`
	Evaluation    planningConfidenceEvaluation `json:"evaluation"`
	SemanticDelta colony.PlanningSemanticDelta `json:"semantic_delta"`
}

type planningStopPolicyInput struct {
	Target  int                      `json:"target"`
	PassCap int                      `json:"pass_cap"`
	History []planningConfidencePass `json:"history"`
}

// planningStopPassDiagnostic persists every derived fact used by the stop
// classifier. Persisting these values makes a later replay explainable as well
// as deterministic, while InputDigest detects a changed diagnostic history.
type planningStopPassDiagnostic struct {
	Iteration                    int                      `json:"iteration"`
	Overall                      int                      `json:"overall"`
	Movement                     int                      `json:"movement"`
	Grounded                     bool                     `json:"grounded"`
	FreshEvidenceIDs             []string                 `json:"fresh_evidence_ids"`
	WeakestGapID                 string                   `json:"weakest_gap_id"`
	WeakestGapDimension          colony.PlanningDimension `json:"weakest_gap_dimension"`
	WeakestGapScore              int                      `json:"weakest_gap_score"`
	EvidenceBackedGapImprovement bool                     `json:"evidence_backed_gap_improvement"`
	MaterialSemanticChange       bool                     `json:"material_semantic_change"`
	SemanticDeltaID              string                   `json:"semantic_delta_id"`
	SemanticDeltaHash            string                   `json:"semantic_delta_hash"`
	ResidualGapIDs               []string                 `json:"residual_gap_ids"`
	MaterialGapIDs               []string                 `json:"material_gap_ids"`
	EvidenceThatWouldChange      string                   `json:"evidence_that_would_change"`
}

type planningStopPolicyEvaluation struct {
	Decision        colony.PlanningStopDecision  `json:"decision"`
	Trigger         colony.PlanningStopReason    `json:"trigger"`
	Policy          string                       `json:"policy"`
	Target          int                          `json:"target"`
	PassCap         int                          `json:"pass_cap"`
	CompletedPasses int                          `json:"completed_passes"`
	Diagnostics     []planningStopPassDiagnostic `json:"diagnostics"`
	InputDigest     string                       `json:"input_digest"`
}

// evaluatePlanningConfidence validates Route-Setter proposals without
// inventing a score from evidence. Evidence authorizes a proposed movement;
// the proposal remains the only source of each before/after dimension value.
func evaluatePlanningConfidence(proposal planningConfidenceProposal) (planningConfidenceEvaluation, error) {
	if proposal.SuppliedOverall != nil {
		return planningConfidenceEvaluation{}, fmt.Errorf("route-setter supplied overall is forbidden; Go computes overall confidence")
	}
	dimensions := colony.PlanningDimensions()
	if len(proposal.Assessments) != len(dimensions) {
		return planningConfidenceEvaluation{}, fmt.Errorf("dimension assessments must contain exactly five unique dimensions")
	}
	for _, dimension := range dimensions {
		prior := proposal.PriorScores.Value(dimension)
		if prior < 0 || prior > 100 {
			return planningConfidenceEvaluation{}, fmt.Errorf("persisted prior %s score must be between 0 and 100", dimension)
		}
	}

	evidenceByID := make(map[string]colony.PlanningEvidenceRef, len(proposal.EvidenceCatalogue))
	for i, reference := range proposal.EvidenceCatalogue {
		if err := reference.Validate(); err != nil {
			return planningConfidenceEvaluation{}, fmt.Errorf("evidence_catalogue[%d]: %w", i, err)
		}
		if _, duplicate := evidenceByID[reference.ID]; duplicate {
			return planningConfidenceEvaluation{}, fmt.Errorf("evidence catalogue contains duplicate stable ID %q", reference.ID)
		}
		evidenceByID[reference.ID] = reference
	}

	byDimension := make(map[colony.PlanningDimension]colony.PlanningDimensionAssessment, len(dimensions))
	gapIDs := make(map[string]struct{}, len(dimensions))
	for i := range proposal.Assessments {
		assessment := clonePlanningConfidenceAssessment(proposal.Assessments[i])
		if err := assessment.Validate(); err != nil {
			return planningConfidenceEvaluation{}, fmt.Errorf("dimension_assessments[%d]: %w", i, err)
		}
		if err := validatePlanningAssessmentShape(fmt.Sprintf("dimension_assessments[%d]", i), assessment); err != nil {
			return planningConfidenceEvaluation{}, err
		}
		if _, duplicate := byDimension[assessment.Dimension]; duplicate {
			return planningConfidenceEvaluation{}, fmt.Errorf("dimension assessments contain duplicate %q", assessment.Dimension)
		}
		prior := proposal.PriorScores.Value(assessment.Dimension)
		if assessment.Before != prior {
			return planningConfidenceEvaluation{}, fmt.Errorf("%s before score %d does not match persisted prior frontier %d", assessment.Dimension, assessment.Before, prior)
		}
		if _, duplicate := gapIDs[assessment.RemainingGap.ID]; duplicate {
			return planningConfidenceEvaluation{}, fmt.Errorf("remaining gaps contain duplicate stable ID %q", assessment.RemainingGap.ID)
		}
		gapIDs[assessment.RemainingGap.ID] = struct{}{}

		for _, evidenceID := range assessment.FreshEvidenceIDs {
			reference, ok := evidenceByID[evidenceID]
			if !ok {
				return planningConfidenceEvaluation{}, fmt.Errorf("%s fresh evidence %q is not present in the evidence catalogue", assessment.Dimension, evidenceID)
			}
			freshness := isFreshPlanningEvidence(reference, proposal.EvidenceFrontier)
			if !freshness.Allowed {
				return planningConfidenceEvaluation{}, fmt.Errorf("%s fresh evidence %q is not fresh: %s", assessment.Dimension, evidenceID, freshness.Reason)
			}
			applicability := evidenceAppliesToDimension(reference, assessment.Dimension)
			if !applicability.Allowed {
				return planningConfidenceEvaluation{}, fmt.Errorf("%s fresh evidence %q is not applicable: %s", assessment.Dimension, evidenceID, applicability.Reason)
			}
		}
		if assessment.After != assessment.Before && len(assessment.FreshEvidenceIDs) == 0 {
			return planningConfidenceEvaluation{}, fmt.Errorf("%s score changed from %d to %d without fresh applicable evidence", assessment.Dimension, assessment.Before, assessment.After)
		}
		byDimension[assessment.Dimension] = assessment
	}

	result := planningConfidenceEvaluation{
		Assessments: make([]colony.PlanningDimensionAssessment, 0, len(dimensions)),
	}
	for _, dimension := range dimensions {
		assessment, ok := byDimension[dimension]
		if !ok {
			return planningConfidenceEvaluation{}, fmt.Errorf("dimension assessments missing %q", dimension)
		}
		result.Assessments = append(result.Assessments, assessment)
		result.Scores.Set(dimension, assessment.After)
	}
	result.Scores.Overall = planningConfidenceWeightedOverall(result.Scores)
	result.RankedGaps = rankPlanningConfidenceGaps(result.Assessments, result.Scores)
	if len(result.RankedGaps) == 0 {
		return planningConfidenceEvaluation{}, fmt.Errorf("dimension assessments contain no remaining gaps")
	}
	result.WeakestGap = clonePlanningConfidenceGap(result.RankedGaps[0])
	return result, nil
}

// evaluatePlanningStopPolicy applies the complete automatic stop vocabulary
// to validated pass history. It never activates or accepts a plan: a terminal
// reason merely makes later candidate creation eligible.
func evaluatePlanningStopPolicy(input planningStopPolicyInput) (planningStopPolicyEvaluation, error) {
	if input.Target < 1 || input.Target > 100 {
		return planningStopPolicyEvaluation{}, fmt.Errorf("planning confidence target must be between 1 and 100")
	}
	if input.PassCap < 1 {
		return planningStopPolicyEvaluation{}, fmt.Errorf("planning pass cap must be positive")
	}
	if len(input.History) == 0 {
		return planningStopPolicyEvaluation{}, fmt.Errorf("planning confidence history is required")
	}

	diagnostics := make([]planningStopPassDiagnostic, 0, len(input.History))
	for index := range input.History {
		var previous *planningConfidencePass
		if index > 0 {
			previous = &input.History[index-1]
		}
		if err := validatePlanningConfidencePass(input.History[index], index+1, previous); err != nil {
			return planningStopPolicyEvaluation{}, fmt.Errorf("history[%d]: %w", index, err)
		}
		diagnostics = append(diagnostics, planningConfidenceDiagnostic(input.History[index], previous))
	}

	digestInput := struct {
		Target      int                          `json:"target"`
		PassCap     int                          `json:"pass_cap"`
		Policy      string                       `json:"policy"`
		Diagnostics []planningStopPassDiagnostic `json:"diagnostics"`
	}{
		Target: input.Target, PassCap: input.PassCap,
		Policy: planningDiminishingPolicyGroundedTwoPassLT2, Diagnostics: diagnostics,
	}
	inputDigest, err := jsonSHA256(digestInput)
	if err != nil {
		return planningStopPolicyEvaluation{}, fmt.Errorf("hash planning stop diagnostics: %w", err)
	}

	current := input.History[len(input.History)-1]
	trigger := colony.PlanningStopContinue
	switch {
	case current.Evaluation.Scores.Overall >= input.Target:
		trigger = colony.PlanningStopTargetMet
	case len(input.History) >= input.PassCap:
		trigger = colony.PlanningStopPassCap
	case planningConfidenceStallEligible(diagnostics):
		trigger = colony.PlanningStopStalledGap
	case planningConfidenceDiminishingEligible(diagnostics):
		trigger = colony.PlanningStopDiminishingReturns
	}

	reason := trigger
	if planningConfidenceIsBelowTargetStop(trigger) && len(diagnostics[len(diagnostics)-1].MaterialGapIDs) > 0 {
		reason = colony.PlanningStopOwnerDecision
	}
	rationale := planningConfidenceStopRationale(reason, trigger, input, current)
	decision, err := planningConfidenceStopDecision(reason, rationale, current)
	if err != nil {
		return planningStopPolicyEvaluation{}, err
	}

	return planningStopPolicyEvaluation{
		Decision: decision, Trigger: trigger,
		Policy: planningDiminishingPolicyGroundedTwoPassLT2,
		Target: input.Target, PassCap: input.PassCap, CompletedPasses: len(input.History),
		Diagnostics: diagnostics, InputDigest: inputDigest,
	}, nil
}

func validatePlanningConfidencePass(pass planningConfidencePass, expectedIteration int, previous *planningConfidencePass) error {
	if pass.Iteration != expectedIteration {
		return fmt.Errorf("iteration = %d, want contiguous ordinal %d", pass.Iteration, expectedIteration)
	}
	for _, dimension := range colony.PlanningDimensions() {
		score := pass.Evaluation.Scores.Value(dimension)
		if score < 0 || score > 100 {
			return fmt.Errorf("%s score must be between 0 and 100", dimension)
		}
	}
	derivedOverall := planningConfidenceWeightedOverall(pass.Evaluation.Scores)
	if pass.Evaluation.Scores.Overall != derivedOverall {
		return fmt.Errorf("Go-derived overall is %d, got persisted value %d", derivedOverall, pass.Evaluation.Scores.Overall)
	}
	if len(pass.Evaluation.Assessments) != len(colony.PlanningDimensions()) {
		return fmt.Errorf("evaluation must contain exactly five dimension assessments")
	}
	seen := make(map[colony.PlanningDimension]struct{}, len(pass.Evaluation.Assessments))
	for index, assessment := range pass.Evaluation.Assessments {
		if err := assessment.Validate(); err != nil {
			return fmt.Errorf("assessment[%d]: %w", index, err)
		}
		if err := validatePlanningAssessmentShape(fmt.Sprintf("assessment[%d]", index), assessment); err != nil {
			return err
		}
		if _, duplicate := seen[assessment.Dimension]; duplicate {
			return fmt.Errorf("evaluation contains duplicate %q assessment", assessment.Dimension)
		}
		seen[assessment.Dimension] = struct{}{}
		if assessment.After != pass.Evaluation.Scores.Value(assessment.Dimension) {
			return fmt.Errorf("%s assessment after score does not match persisted dimension score", assessment.Dimension)
		}
		if previous != nil {
			wantBefore := previous.Evaluation.Scores.Value(assessment.Dimension)
			if assessment.Before != wantBefore {
				return fmt.Errorf("%s assessment before score %d does not match prior pass %d", assessment.Dimension, assessment.Before, wantBefore)
			}
		}
		if assessment.After != assessment.Before && len(assessment.FreshEvidenceIDs) == 0 {
			return fmt.Errorf("%s changed without validated fresh evidence", assessment.Dimension)
		}
	}
	for _, dimension := range colony.PlanningDimensions() {
		if _, ok := seen[dimension]; !ok {
			return fmt.Errorf("evaluation is missing %q assessment", dimension)
		}
	}

	wantGaps := rankPlanningConfidenceGaps(pass.Evaluation.Assessments, pass.Evaluation.Scores)
	if len(pass.Evaluation.RankedGaps) != len(wantGaps) {
		return fmt.Errorf("ranked gaps do not match the five validated assessments")
	}
	for index := range wantGaps {
		got := pass.Evaluation.RankedGaps[index]
		if got.ID != wantGaps[index].ID || got.ContentHash != wantGaps[index].ContentHash {
			return fmt.Errorf("ranked gap %d does not match deterministic material-first ranking", index)
		}
	}
	if len(wantGaps) == 0 || pass.Evaluation.WeakestGap.ID != wantGaps[0].ID || pass.Evaluation.WeakestGap.ContentHash != wantGaps[0].ContentHash {
		return fmt.Errorf("weakest gap does not match deterministic ranking")
	}
	if err := pass.SemanticDelta.Validate(); err != nil {
		return fmt.Errorf("semantic_delta: %w", err)
	}
	if err := validatePlanningDeltaShape("semantic_delta", pass.SemanticDelta); err != nil {
		return err
	}
	return nil
}

func planningConfidenceDiagnostic(pass planningConfidencePass, previous *planningConfidencePass) planningStopPassDiagnostic {
	freshEvidenceIDs := make([]string, 0)
	residualGapIDs := make([]string, 0, len(pass.Evaluation.RankedGaps))
	materialGapIDs := make([]string, 0)
	for _, assessment := range pass.Evaluation.Assessments {
		freshEvidenceIDs = append(freshEvidenceIDs, assessment.FreshEvidenceIDs...)
	}
	for _, gap := range pass.Evaluation.RankedGaps {
		residualGapIDs = append(residualGapIDs, gap.ID)
		if gap.Materiality == colony.PlanningGapMaterial {
			materialGapIDs = append(materialGapIDs, gap.ID)
		}
	}
	freshEvidenceIDs = uniqueSortedStrings(freshEvidenceIDs)
	diagnostic := planningStopPassDiagnostic{
		Iteration: pass.Iteration, Overall: pass.Evaluation.Scores.Overall,
		Grounded: len(freshEvidenceIDs) > 0, FreshEvidenceIDs: freshEvidenceIDs,
		WeakestGapID:           pass.Evaluation.WeakestGap.ID,
		WeakestGapDimension:    pass.Evaluation.WeakestGap.Dimension,
		WeakestGapScore:        pass.Evaluation.Scores.Value(pass.Evaluation.WeakestGap.Dimension),
		MaterialSemanticChange: planningConfidenceHasMaterialSemanticChange(pass.SemanticDelta),
		SemanticDeltaID:        pass.SemanticDelta.ID, SemanticDeltaHash: pass.SemanticDelta.ContentHash,
		ResidualGapIDs: residualGapIDs, MaterialGapIDs: materialGapIDs,
		EvidenceThatWouldChange: strings.TrimSpace(pass.Evaluation.WeakestGap.EvidenceThatWouldChange),
	}
	if previous == nil {
		return diagnostic
	}
	diagnostic.Movement = pass.Evaluation.Scores.Overall - previous.Evaluation.Scores.Overall
	if previous.Evaluation.WeakestGap.ID == pass.Evaluation.WeakestGap.ID {
		previousScore := previous.Evaluation.Scores.Value(pass.Evaluation.WeakestGap.Dimension)
		diagnostic.EvidenceBackedGapImprovement = diagnostic.WeakestGapScore > previousScore && planningConfidenceAssessmentHasFreshEvidence(pass.Evaluation.Assessments, pass.Evaluation.WeakestGap.Dimension)
	}
	return diagnostic
}

func planningConfidenceAssessmentHasFreshEvidence(assessments []colony.PlanningDimensionAssessment, dimension colony.PlanningDimension) bool {
	for _, assessment := range assessments {
		if assessment.Dimension == dimension {
			return len(assessment.FreshEvidenceIDs) > 0
		}
	}
	return false
}

func planningConfidenceHasMaterialSemanticChange(delta colony.PlanningSemanticDelta) bool {
	sections := [][]colony.PlanningSemanticChange{
		delta.Phases, delta.Tasks, delta.Dependencies, delta.RequirementLinks,
		delta.AcceptanceChecks, delta.NegativeExpectations, delta.RecoveryExpectations, delta.PublicPaths,
	}
	for _, section := range sections {
		for _, change := range section {
			if change.Kind != colony.PlanningSemanticChangePreserved {
				return true
			}
		}
	}
	return false
}

func planningConfidenceDiminishingEligible(diagnostics []planningStopPassDiagnostic) bool {
	if len(diagnostics) < 3 {
		return false
	}
	for _, diagnostic := range diagnostics[len(diagnostics)-2:] {
		if !diagnostic.Grounded || planningConfidenceAbs(diagnostic.Movement) >= 2 || diagnostic.MaterialSemanticChange {
			return false
		}
	}
	return true
}

func planningConfidenceStallEligible(diagnostics []planningStopPassDiagnostic) bool {
	if len(diagnostics) < 3 {
		return false
	}
	last := diagnostics[len(diagnostics)-3:]
	if last[0].WeakestGapID == "" || last[0].WeakestGapID != last[1].WeakestGapID || last[1].WeakestGapID != last[2].WeakestGapID {
		return false
	}
	return !last[1].EvidenceBackedGapImprovement && !last[2].EvidenceBackedGapImprovement
}

func planningConfidenceIsBelowTargetStop(reason colony.PlanningStopReason) bool {
	switch reason {
	case colony.PlanningStopDiminishingReturns, colony.PlanningStopStalledGap, colony.PlanningStopPassCap:
		return true
	default:
		return false
	}
}

func planningConfidenceStopRationale(reason, trigger colony.PlanningStopReason, input planningStopPolicyInput, current planningConfidencePass) string {
	overall := current.Evaluation.Scores.Overall
	gapID := current.Evaluation.WeakestGap.ID
	if reason == colony.PlanningStopOwnerDecision {
		return fmt.Sprintf("Automatic %s below the %d%% target is blocked because material gap %s remains unresolved; owner guidance is required", trigger, input.Target, gapID)
	}
	switch reason {
	case colony.PlanningStopTargetMet:
		return fmt.Sprintf("Go-derived overall confidence %d%% met the %d%% target", overall, input.Target)
	case colony.PlanningStopPassCap:
		return fmt.Sprintf("Planning reached the configured pass cap of %d below the %d%% target", input.PassCap, input.Target)
	case colony.PlanningStopStalledGap:
		return fmt.Sprintf("Weakest gap %s remained unresolved through two consecutive passes with no evidence-backed improvement", gapID)
	case colony.PlanningStopDiminishingReturns:
		return fmt.Sprintf("Policy %s matched two grounded passes with absolute overall movement below two points and no material semantic change", planningDiminishingPolicyGroundedTwoPassLT2)
	default:
		return fmt.Sprintf("Overall confidence %d%% is below the %d%% target and no stop policy matched; continue with weakest gap %s", overall, input.Target, gapID)
	}
}

func planningConfidenceStopDecision(reason colony.PlanningStopReason, rationale string, current planningConfidencePass) (colony.PlanningStopDecision, error) {
	residualGapIDs := make([]string, 0, len(current.Evaluation.RankedGaps))
	evidenceIDs := make([]string, 0)
	for _, assessment := range current.Evaluation.Assessments {
		evidenceIDs = append(evidenceIDs, assessment.FreshEvidenceIDs...)
	}
	for _, gap := range current.Evaluation.RankedGaps {
		residualGapIDs = append(residualGapIDs, gap.ID)
		evidenceIDs = append(evidenceIDs, gap.EvidenceIDs...)
	}
	evidenceIDs = uniqueSortedStrings(evidenceIDs)
	decision := colony.PlanningStopDecision{
		SchemaVersion: colony.PlanningSchemaVersion,
		Reason:        reason, SelectedGapID: current.Evaluation.WeakestGap.ID,
		ResidualGapIDs: residualGapIDs, EvidenceIDs: evidenceIDs,
		Rationale:               strings.TrimSpace(rationale),
		EvidenceThatWouldChange: strings.TrimSpace(current.Evaluation.WeakestGap.EvidenceThatWouldChange),
	}
	payload := struct {
		Reason                  colony.PlanningStopReason `json:"reason"`
		SelectedGapID           string                    `json:"selected_gap_id"`
		ResidualGapIDs          []string                  `json:"residual_gap_ids"`
		EvidenceIDs             []string                  `json:"evidence_ids"`
		Rationale               string                    `json:"rationale"`
		EvidenceThatWouldChange string                    `json:"evidence_that_would_change"`
	}{
		Reason: decision.Reason, SelectedGapID: decision.SelectedGapID,
		ResidualGapIDs: decision.ResidualGapIDs, EvidenceIDs: decision.EvidenceIDs,
		Rationale: decision.Rationale, EvidenceThatWouldChange: decision.EvidenceThatWouldChange,
	}
	digest, err := jsonSHA256(payload)
	if err != nil {
		return colony.PlanningStopDecision{}, fmt.Errorf("hash planning stop decision: %w", err)
	}
	decision.ContentHash = digest
	decision.ID = "planning-stop-" + digest[:12]
	if err := decision.Validate(); err != nil {
		return colony.PlanningStopDecision{}, fmt.Errorf("planning stop decision: %w", err)
	}
	return decision, nil
}

func planningConfidenceAbs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func planningConfidenceWeightedOverall(scores planningConfidenceScores) int {
	weighted := scores.Knowledge*25 +
		scores.Requirements*25 +
		scores.Risks*20 +
		scores.Dependencies*15 +
		scores.Effort*15
	return (weighted + 50) / 100
}

func rankPlanningConfidenceGaps(assessments []colony.PlanningDimensionAssessment, scores planningConfidenceScores) []colony.PlanningGap {
	gaps := make([]colony.PlanningGap, 0, len(assessments))
	for i := range assessments {
		gaps = append(gaps, clonePlanningConfidenceGap(assessments[i].RemainingGap))
	}
	sort.Slice(gaps, func(i, j int) bool {
		left, right := gaps[i], gaps[j]
		if left.Materiality != right.Materiality {
			return left.Materiality == colony.PlanningGapMaterial
		}
		leftDeficit := 100 - scores.Value(left.Dimension)
		rightDeficit := 100 - scores.Value(right.Dimension)
		if leftDeficit != rightDeficit {
			return leftDeficit > rightDeficit
		}
		if left.Severity != right.Severity {
			return left.Severity > right.Severity
		}
		return strings.TrimSpace(left.ID) < strings.TrimSpace(right.ID)
	})
	return gaps
}

func clonePlanningConfidenceAssessment(value colony.PlanningDimensionAssessment) colony.PlanningDimensionAssessment {
	value.FreshEvidenceIDs = append([]string{}, value.FreshEvidenceIDs...)
	value.ResolvedGapIDs = append([]string{}, value.ResolvedGapIDs...)
	value.RemainingGap = clonePlanningConfidenceGap(value.RemainingGap)
	return value
}

func clonePlanningConfidenceGap(value colony.PlanningGap) colony.PlanningGap {
	value.EvidenceIDs = append([]string{}, value.EvidenceIDs...)
	return value
}
