package cmd

import (
	"fmt"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestResolveReviewDepth_FinalPhaseAlwaysHeavy(t *testing.T) {
	phase := colony.Phase{ID: 5, Name: "Final polish"}
	// Final phase (ID == totalPhases) must be heavy even with lightFlag=true
	got := resolveReviewDepth(phase, 5, true, false)
	if got != ReviewDepthHeavy {
		t.Errorf("final phase with lightFlag=true: got %q, want %q", got, ReviewDepthHeavy)
	}
}

func TestResolveReviewDepth_HeavyFlagOverrides(t *testing.T) {
	phase := colony.Phase{ID: 2, Name: "Feature work"}
	got := resolveReviewDepth(phase, 5, false, true)
	if got != ReviewDepthHeavy {
		t.Errorf("heavyFlag=true on non-final non-keyword phase: got %q, want %q", got, ReviewDepthHeavy)
	}
}

func TestResolveReviewDepth_LightFlagOnIntermediate(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Feature work"}
	got := resolveReviewDepth(phase, 5, true, false)
	if got != ReviewDepthLight {
		t.Errorf("lightFlag=true on non-final non-keyword phase: got %q, want %q", got, ReviewDepthLight)
	}
}

func TestResolveReviewDepth_AutoDetectDefaultLight(t *testing.T) {
	phase := colony.Phase{ID: 2, Name: "Feature work"}
	got := resolveReviewDepth(phase, 5, false, false)
	if got != ReviewDepthLight {
		t.Errorf("no flags, non-final non-keyword phase: got %q, want %q", got, ReviewDepthLight)
	}
}

func TestResolveReviewDepth_BothFlagsHeavyWins(t *testing.T) {
	phase := colony.Phase{ID: 2, Name: "Feature work"}
	got := resolveReviewDepth(phase, 5, true, true)
	if got != ReviewDepthHeavy {
		t.Errorf("both flags set: got %q, want %q (heavy is safer)", got, ReviewDepthHeavy)
	}
}

func TestResolveReviewDepth_FinalPhaseIgnoresHeavyFlag(t *testing.T) {
	phase := colony.Phase{ID: 4, Name: "Cleanup"}
	got := resolveReviewDepth(phase, 4, false, true)
	if got != ReviewDepthHeavy {
		t.Errorf("final phase with heavyFlag=true: got %q, want %q", got, ReviewDepthHeavy)
	}
}

func TestResolveReviewDepth_KeywordPhaseAutoHeavy(t *testing.T) {
	tests := []struct {
		name     string
		phase    colony.Phase
		total    int
		light    bool
		heavy    bool
		expected ReviewDepth
	}{
		{
			name:     "security keyword triggers heavy",
			phase:    colony.Phase{ID: 2, Name: "Security audit"},
			total:    5, light: false, heavy: false,
			expected: ReviewDepthHeavy,
		},
		{
			name:     "keyword phase with light flag still heavy",
			phase:    colony.Phase{ID: 2, Name: "Auth refactor"},
			total:    5, light: true, heavy: false,
			expected: ReviewDepthHeavy,
		},
		{
			name:     "non-keyword non-final defaults light",
			phase:    colony.Phase{ID: 2, Name: "UI polish"},
			total:    5, light: false, heavy: false,
			expected: ReviewDepthLight,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveReviewDepth(tt.phase, tt.total, tt.light, tt.heavy)
			if got != tt.expected {
				t.Errorf("resolveReviewDepth(%+v, %d, %v, %v) = %q, want %q",
					tt.phase, tt.total, tt.light, tt.heavy, got, tt.expected)
			}
		})
	}
}

func TestPhaseHasHeavyKeywords_All12Keywords(t *testing.T) {
	keywords := []string{
		"security", "auth", "crypto", "secrets",
		"permissions", "compliance", "audit",
		"release", "deploy", "production", "ship", "launch",
	}
	for _, kw := range keywords {
		t.Run(kw, func(t *testing.T) {
			if !phaseHasHeavyKeywords(kw) {
				t.Errorf("phaseHasHeavyKeywords(%q) = false, want true", kw)
			}
		})
	}
}

func TestPhaseHasHeavyKeywords_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"title case", "Security Audit", true},
		{"upper case", "SECURITY AUDIT", true},
		{"mixed case", "SeCuRiTy AuDiT", true},
		{"all lower", "security audit", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := phaseHasHeavyKeywords(tt.input)
			if got != tt.want {
				t.Errorf("phaseHasHeavyKeywords(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPhaseHasHeavyKeywords_SubstringMatch(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"authentication contains auth", "authentication", true},
		{"authorization contains auth", "authorization", true},
		{"cryptographic contains crypto", "cryptographic", true},
		{"deploying contains deploy", "deploying", true},
		{"shipping contains ship", "shipping", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := phaseHasHeavyKeywords(tt.input)
			if got != tt.want {
				t.Errorf("phaseHasHeavyKeywords(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPhaseHasHeavyKeywords_NoMatch(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"feature work", "feature work", false},
		{"ui polish", "ui polish", false},
		{"empty string", "", false},
		{"random text", "implement the new dashboard", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := phaseHasHeavyKeywords(tt.input)
			if got != tt.want {
				t.Errorf("phaseHasHeavyKeywords(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestReviewDepthFlags(t *testing.T) {
	t.Run("build has light flag", func(t *testing.T) {
		f := buildCmd.Flags().Lookup("light")
		if f == nil {
			t.Fatal("buildCmd has no --light flag")
		}
		if f.DefValue != "false" {
			t.Errorf("buildCmd --light default = %q, want false", f.DefValue)
		}
	})
	t.Run("build has heavy flag", func(t *testing.T) {
		f := buildCmd.Flags().Lookup("heavy")
		if f == nil {
			t.Fatal("buildCmd has no --heavy flag")
		}
		if f.DefValue != "false" {
			t.Errorf("buildCmd --heavy default = %q, want false", f.DefValue)
		}
	})
	t.Run("continue has light flag", func(t *testing.T) {
		f := continueCmd.Flags().Lookup("light")
		if f == nil {
			t.Fatal("continueCmd has no --light flag")
		}
		if f.DefValue != "false" {
			t.Errorf("continueCmd --light default = %q, want false", f.DefValue)
		}
	})
	t.Run("continue has heavy flag", func(t *testing.T) {
		f := continueCmd.Flags().Lookup("heavy")
		if f == nil {
			t.Fatal("continueCmd has no --heavy flag")
		}
		if f.DefValue != "false" {
			t.Errorf("continueCmd --heavy default = %q, want false", f.DefValue)
		}
	})
}

func TestChaosShouldRunInLightMode_Deterministic(t *testing.T) {
	tests := []struct {
		phaseID int
		want    bool
	}{
		// phaseID % 10 < 3 means IDs ending in 0, 1, 2 get true
		{1, true},
		{2, true},
		{3, false},
		{5, false},
		{7, false},
		{9, false},
		{10, true},
		{11, true},
		{12, true},
		{13, false},
		{20, true},
		{22, true},
		{23, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("phase_%d", tt.phaseID), func(t *testing.T) {
			got := chaosShouldRunInLightMode(tt.phaseID)
			if got != tt.want {
				t.Errorf("chaosShouldRunInLightMode(%d) = %v, want %v", tt.phaseID, got, tt.want)
			}
		})
	}
}
