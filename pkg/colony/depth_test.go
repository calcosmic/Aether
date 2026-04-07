package colony

import "testing"

func TestColonyDepthValid(t *testing.T) {
	// Valid values must return true
	validDepths := []ColonyDepth{DepthLight, DepthStandard, DepthDeep, DepthFull}
	for _, d := range validDepths {
		if !d.Valid() {
			t.Errorf("ColonyDepth.Valid() returned false for %q, expected true", d)
		}
	}

	// Invalid values must return false
	invalidDepths := []ColonyDepth{"invalid", "", "banana", "LIGHT", "Standard", "FULL"}
	for _, d := range invalidDepths {
		if d.Valid() {
			t.Errorf("ColonyDepth.Valid() returned true for %q, expected false", d)
		}
	}
}

func TestDepthBudget(t *testing.T) {
	tests := []struct {
		depth          ColonyDepth
		wantContext    int
		wantSkills     int
	}{
		{DepthLight, 4000, 4000},
		{DepthStandard, 8000, 8000},
		{DepthDeep, 16000, 12000},
		{DepthFull, 24000, 16000},
	}

	for _, tt := range tests {
		ctx, skills := DepthBudget(tt.depth)
		if ctx != tt.wantContext {
			t.Errorf("DepthBudget(%q) context = %d, want %d", tt.depth, ctx, tt.wantContext)
		}
		if skills != tt.wantSkills {
			t.Errorf("DepthBudget(%q) skills = %d, want %d", tt.depth, skills, tt.wantSkills)
		}
	}
}

func TestDepthBudgetDefault(t *testing.T) {
	// Unknown depth should return standard budget
	ctx, skills := DepthBudget("unknown")
	if ctx != 8000 {
		t.Errorf("DepthBudget(unknown) context = %d, want 8000", ctx)
	}
	if skills != 8000 {
		t.Errorf("DepthBudget(unknown) skills = %d, want 8000", skills)
	}
}
