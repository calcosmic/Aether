package colony

import "testing"

func TestPlanGranularityValid(t *testing.T) {
	validGranularities := []PlanGranularity{GranularitySprint, GranularityMilestone, GranularityQuarter, GranularityMajor}
	for _, g := range validGranularities {
		if !g.Valid() {
			t.Errorf("PlanGranularity.Valid() returned false for %q, expected true", g)
		}
	}

	invalidGranularities := []PlanGranularity{"invalid", "", "SPRINT", "Sprint", "banana"}
	for _, g := range invalidGranularities {
		if g.Valid() {
			t.Errorf("PlanGranularity.Valid() returned true for %q, expected false", g)
		}
	}
}

func TestGranularityRange(t *testing.T) {
	tests := []struct {
		granularity PlanGranularity
		wantMin     int
		wantMax     int
	}{
		{GranularitySprint, 1, 3},
		{GranularityMilestone, 4, 7},
		{GranularityQuarter, 8, 12},
		{GranularityMajor, 13, 20},
	}

	for _, tt := range tests {
		min, max := GranularityRange(tt.granularity)
		if min != tt.wantMin {
			t.Errorf("GranularityRange(%q) min = %d, want %d", tt.granularity, min, tt.wantMin)
		}
		if max != tt.wantMax {
			t.Errorf("GranularityRange(%q) max = %d, want %d", tt.granularity, max, tt.wantMax)
		}
	}
}

func TestGranularityRangeDefault(t *testing.T) {
	min, max := GranularityRange("unknown")
	if min != 1 {
		t.Errorf("GranularityRange(unknown) min = %d, want 1", min)
	}
	if max != 3 {
		t.Errorf("GranularityRange(unknown) max = %d, want 3", max)
	}
}
