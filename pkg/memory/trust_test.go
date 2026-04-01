package memory

import (
	"fmt"
	"math"
	"testing"
)

const epsilon = 1e-6

func TestTrustCalculate_SourceWeights(t *testing.T) {
	tests := []struct {
		name       string
		sourceType string
		wantScore  float64
	}{
		{"user_feedback", "user_feedback", 1.0},
		{"error_resolution", "error_resolution", 0.9},
		{"success_pattern", "success_pattern", 0.8},
		{"observation", "observation", 0.6},
		{"heuristic", "heuristic", 0.4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TrustInput{
				SourceType: tt.sourceType,
				Evidence:   "test_verified",
				DaysSince:  0,
			}
			got := Calculate(input)

			// raw_score = 0.4*source + 0.35*evidence + 0.25*activity
			// With evidence=test_verified (1.0) and days_since=0 (activity=1.0):
			// raw_score = 0.4*source + 0.35*1.0 + 0.25*1.0 = 0.4*source + 0.6
			expected := math.Max(0.2, 0.4*tt.wantScore+0.35*1.0+0.25*1.0)
			expected = math.Round(expected*1e6) / 1e6

			if math.Abs(got.Score-expected) > epsilon {
				t.Errorf("Calculate(%+v).Score = %f, want %f", input, got.Score, expected)
			}
			if math.Abs(got.SourceScore-tt.wantScore) > epsilon {
				t.Errorf("SourceScore = %f, want %f", got.SourceScore, tt.wantScore)
			}
		})
	}
}

func TestTrustCalculate_EvidenceWeights(t *testing.T) {
	tests := []struct {
		name     string
		evidence string
		wantEv   float64
	}{
		{"test_verified", "test_verified", 1.0},
		{"multi_phase", "multi_phase", 0.9},
		{"single_phase", "single_phase", 0.7},
		{"anecdotal", "anecdotal", 0.4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TrustInput{
				SourceType: "user_feedback",
				Evidence:   tt.evidence,
				DaysSince:  0,
			}
			got := Calculate(input)

			// raw_score = 0.4*1.0 + 0.35*evidence + 0.25*1.0 = 0.4 + 0.35*evidence + 0.25
			expected := math.Max(0.2, 0.4*1.0+0.35*tt.wantEv+0.25*1.0)
			expected = math.Round(expected*1e6) / 1e6

			if math.Abs(got.Score-expected) > epsilon {
				t.Errorf("Calculate(%+v).Score = %f, want %f", input, got.Score, expected)
			}
			if math.Abs(got.EvidenceScore-tt.wantEv) > epsilon {
				t.Errorf("EvidenceScore = %f, want %f", got.EvidenceScore, tt.wantEv)
			}
		})
	}
}

func TestTrustCalculate_HalfLifeDecay(t *testing.T) {
	tests := []struct {
		name      string
		daysSince int
		wantAct   float64
	}{
		{"days_0", 0, 1.0},
		{"days_60", 60, 0.5},
		{"days_120", 120, 0.25},
		{"days_180", 180, 0.125},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TrustInput{
				SourceType: "user_feedback",
				Evidence:   "test_verified",
				DaysSince:  tt.daysSince,
			}
			got := Calculate(input)

			if math.Abs(got.ActivityScore-tt.wantAct) > epsilon {
				t.Errorf("ActivityScore = %f, want %f (days_since=%d)", got.ActivityScore, tt.wantAct, tt.daysSince)
			}

			// Verify score matches expected formula
			rawScore := 0.4*1.0 + 0.35*1.0 + 0.25*tt.wantAct
			expected := math.Max(0.2, rawScore)
			expected = math.Round(expected*1e6) / 1e6

			if math.Abs(got.Score-expected) > epsilon {
				t.Errorf("Score = %f, want %f", got.Score, expected)
			}
		})
	}
}

func TestTrustCalculate_Floor(t *testing.T) {
	input := TrustInput{
		SourceType: "heuristic",
		Evidence:   "anecdotal",
		DaysSince:  365,
	}
	got := Calculate(input)

	if got.Score < 0.2 {
		t.Errorf("Score = %f, should never be below 0.2 (floor)", got.Score)
	}

	// With heuristic (0.4) and anecdotal (0.4) and days=365:
	// activity = 0.5^(365/60) ≈ 0.0137
	// raw = 0.4*0.4 + 0.35*0.4 + 0.25*0.0137 = 0.16 + 0.14 + 0.0034 = 0.3034
	// But floor is 0.2, and raw is 0.3034 > 0.2, so no floor applied
	// Let's verify it's at least 0.2
	if got.Score < 0.2-epsilon {
		t.Errorf("Score = %f is below floor 0.2", got.Score)
	}
}

func TestTrustDecay(t *testing.T) {
	tests := []struct {
		name      string
		score     float64
		days      int
		wantApprox float64
	}{
		{"decay_30_days", 0.8, 30, 0.566},
		{"no_decay", 1.0, 0, 1.0},
		{"decay_60_days", 1.0, 60, 0.5},
		{"decay_with_floor", 0.3, 365, 0.2}, // 0.3 * 0.5^(365/60) ≈ 0.004 -> floor 0.2
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decay(tt.score, tt.days)
			if math.Abs(got-tt.wantApprox) > 0.01 {
				t.Errorf("Decay(%f, %d) = %f, want approximately %f", tt.score, tt.days, got, tt.wantApprox)
			}
			// Floor check
			if got < 0.2-epsilon {
				t.Errorf("Decay result %f is below floor 0.2", got)
			}
		})
	}
}

func TestTrustTier(t *testing.T) {
	tests := []struct {
		score     float64
		wantTier  string
		wantIndex int
	}{
		{0.95, "canonical", 0},
		{0.90, "canonical", 0},
		{0.85, "trusted", 1},
		{0.80, "trusted", 1},
		{0.75, "established", 2},
		{0.70, "established", 2},
		{0.65, "emerging", 3},
		{0.60, "emerging", 3},
		{0.50, "provisional", 4},
		{0.45, "provisional", 4},
		{0.35, "suspect", 5},
		{0.30, "suspect", 5},
		{0.20, "dormant", 6},
		{0.10, "dormant", 6},
		{0.0, "dormant", 6},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%f", tt.score), func(t *testing.T) {
			gotTier, gotIndex := Tier(tt.score)
			if gotTier != tt.wantTier {
				t.Errorf("Tier(%f) = %q, want %q", tt.score, gotTier, tt.wantTier)
			}
			if gotIndex != tt.wantIndex {
				t.Errorf("Tier(%f) index = %d, want %d", tt.score, gotIndex, tt.wantIndex)
			}
		})
	}
}

func TestTrustCalculate_InvalidSource(t *testing.T) {
	input := TrustInput{
		SourceType: "unknown_type",
		Evidence:   "test_verified",
		DaysSince:  0,
	}
	got := Calculate(input)

	// Unknown source defaults to 0.0, so:
	// raw = 0.4*0.0 + 0.35*1.0 + 0.25*1.0 = 0.6
	// score = max(0.2, 0.6) = 0.6
	expected := 0.6
	if math.Abs(got.Score-expected) > epsilon {
		t.Errorf("Calculate with unknown source: Score = %f, want %f", got.Score, expected)
	}
	if got.SourceScore != 0.0 {
		t.Errorf("SourceScore = %f, want 0.0 for unknown source", got.SourceScore)
	}
}

func TestTrustCalculate_InvalidEvidence(t *testing.T) {
	input := TrustInput{
		SourceType: "user_feedback",
		Evidence:   "unknown_evidence",
		DaysSince:  0,
	}
	got := Calculate(input)

	// Unknown evidence defaults to 0.0, so:
	// raw = 0.4*1.0 + 0.35*0.0 + 0.25*1.0 = 0.65
	// score = max(0.2, 0.65) = 0.65
	expected := 0.65
	if math.Abs(got.Score-expected) > epsilon {
		t.Errorf("Calculate with unknown evidence: Score = %f, want %f", got.Score, expected)
	}
	if got.EvidenceScore != 0.0 {
		t.Errorf("EvidenceScore = %f, want 0.0 for unknown evidence", got.EvidenceScore)
	}
}
