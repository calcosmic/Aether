package memory

import "math"

// Source: trust-scoring.sh lines 100-111
// Shell: raw_score = 0.4 * source_score + 0.35 * evidence_score + 0.25 * activity_score
// Where activity_score = 0.5 ^ (days / 60)
// And score = max(0.2, raw_score)

var sourceWeights = map[string]float64{
	"user_feedback":    1.0,
	"error_resolution": 0.9,
	"success_pattern":  0.8,
	"observation":      0.6,
	"heuristic":        0.4,
}

var evidenceWeights = map[string]float64{
	"test_verified": 1.0,
	"multi_phase":   0.9,
	"single_phase":  0.7,
	"anecdotal":     0.4,
}

// TrustInput holds the inputs for trust score calculation.
type TrustInput struct {
	SourceType string
	Evidence   string
	DaysSince  int
}

// TrustResult holds the output of trust score calculation.
type TrustResult struct {
	Score         float64
	SourceScore   float64
	EvidenceScore float64
	ActivityScore float64
	Tier          string
	TierIndex     int
}

// Calculate computes a trust score from the given input.
// Source: Shell trust-scoring.sh lines 37-134
func Calculate(input TrustInput) TrustResult {
	sourceScore := sourceWeights[input.SourceType]
	evidenceScore := evidenceWeights[input.Evidence]
	activityScore := math.Pow(0.5, float64(input.DaysSince)/60.0)

	rawScore := 0.4*sourceScore + 0.35*evidenceScore + 0.25*activityScore
	score := math.Max(0.2, rawScore)

	// Round to 6 decimal places to match shell scale=6
	score = math.Round(score*1e6) / 1e6

	tier, tierIndex := scoreToTier(score)

	return TrustResult{
		Score:         score,
		SourceScore:   sourceScore,
		EvidenceScore: evidenceScore,
		ActivityScore: activityScore,
		Tier:          tier,
		TierIndex:     tierIndex,
	}
}

// Decay applies half-life decay to an existing trust score.
// Source: trust-scoring.sh decay logic
func Decay(score float64, days int) float64 {
	activity := math.Pow(0.5, float64(days)/60.0)
	decayed := score * activity
	// Round to 6 decimal places to match shell scale=6
	decayed = math.Round(decayed*1e6) / 1e6
	decayed = math.Max(0.2, decayed)
	return decayed
}

// Tier returns the trust tier name and index for a given score.
// Source: trust-scoring.sh lines 296-330
func Tier(score float64) (string, int) {
	return scoreToTier(score)
}

func scoreToTier(score float64) (string, int) {
	switch {
	case score >= 0.90:
		return "canonical", 0
	case score >= 0.80:
		return "trusted", 1
	case score >= 0.70:
		return "established", 2
	case score >= 0.60:
		return "emerging", 3
	case score >= 0.45:
		return "provisional", 4
	case score >= 0.30:
		return "suspect", 5
	default:
		return "dormant", 6
	}
}
