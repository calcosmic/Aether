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
