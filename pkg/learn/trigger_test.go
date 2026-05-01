package learn

import "testing"

// TestIsLearningEligible tests all 16 boolean combinations of the 4-condition
// AND gate per D-01, D-02, D-04, D-16.
func TestIsLearningEligible(t *testing.T) {
	tests := []struct {
		name       string
		allSuccess bool
		provenance bool
		gates      bool
		enabled    bool
		want       bool
	}{
		{"all true", true, true, true, true, true},
		{"worker failed", false, true, true, true, false},
		{"provenance invalid", true, false, true, true, false},
		{"gates failed", true, true, false, true, false},
		{"learning disabled", true, true, true, false, false},
		{"all false", false, false, false, false, false},
		// All single-false combos (4 above already cover) + remaining 2-false combos
		{"workers+provenance false", false, false, true, true, false},
		{"workers+gates false", false, true, false, true, false},
		{"workers+enabled false", false, true, true, false, false},
		{"provenance+gates false", true, false, false, true, false},
		{"provenance+enabled false", true, false, true, false, false},
		{"gates+enabled false", true, true, false, false, false},
		// 3-false combos
		{"only workers true", true, false, false, false, false},
		{"only provenance true", false, true, false, false, false},
		{"only gates true", false, false, true, false, false},
		{"only enabled true", false, false, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLearningEligible(tt.allSuccess, tt.provenance, tt.gates, tt.enabled)
			if got != tt.want {
				t.Errorf("IsLearningEligible(%v, %v, %v, %v) = %v, want %v",
					tt.allSuccess, tt.provenance, tt.gates, tt.enabled, got, tt.want)
			}
		})
	}
}

// TestIsLearningEligible_D02_Strictest verifies D-02: any single worker
// failure causes learning to be blocked, regardless of other conditions.
func TestIsLearningEligible_D02_Strictest(t *testing.T) {
	// Even with provenance valid, gates passed, and learning enabled,
	// a single worker failure must block learning.
	if IsLearningEligible(false, true, true, true) {
		t.Error("D-02: worker failure should block learning even when all other conditions pass")
	}
}
