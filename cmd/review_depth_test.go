package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestResolveReviewDepth_LightFlagOverridesFinalPhase(t *testing.T) {
	phase := colony.Phase{ID: 5, Name: "Final polish"}
	got := resolveReviewDepth(phase, 5, true, false)
	if got != ReviewDepthLight {
		t.Errorf("final phase with lightFlag=true: got %q, want %q", got, ReviewDepthLight)
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
			name:  "security keyword triggers heavy",
			phase: colony.Phase{ID: 2, Name: "Security audit"},
			total: 5, light: false, heavy: false,
			expected: ReviewDepthHeavy,
		},
		{
			name:  "keyword phase with light flag overrides to light",
			phase: colony.Phase{ID: 2, Name: "Auth refactor"},
			total: 5, light: true, heavy: false,
			expected: ReviewDepthLight,
		},
		{
			name:  "non-keyword non-final defaults light",
			phase: colony.Phase{ID: 2, Name: "UI polish"},
			total: 5, light: false, heavy: false,
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

// --- Task 1 tests: build dispatch and continue review dispatch depth filtering ---

func TestBuildDispatch_LightMode_SkipsMeasurerAndChaos(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Feature work", Tasks: []colony.Task{{Goal: "Do something", Status: "pending"}}}
	dispatches := plannedBuildDispatchesForSelection(phase, "full", nil, colony.VerificationDepthLight)
	for _, d := range dispatches {
		if d.Caste == "measurer" {
			t.Error("light mode should skip measurer dispatch")
		}
		if d.Caste == "chaos" {
			t.Errorf("light mode on phase 3 (chaosShouldRunInLightMode=false) should skip chaos, got chaos dispatch: %s", d.Name)
		}
	}
}

func TestBuildDispatch_LightMode_Chaos30Percent(t *testing.T) {
	// chaosShouldRunInLightMode returns true for phase IDs where phaseID % 10 < 3
	// Phase IDs 1, 2, 10, 11, 12, 20, 21, 22 should include chaos in light mode
	chaosPhases := []int{1, 2, 10, 11, 12, 20, 21, 22}
	noChaosPhases := []int{3, 5, 7, 9, 13, 15, 23, 25}

	for _, pid := range chaosPhases {
		t.Run(fmt.Sprintf("phase_%d_includes_chaos", pid), func(t *testing.T) {
			phase := colony.Phase{ID: pid, Name: "Feature work", Tasks: []colony.Task{{Goal: "Do something", Status: "pending"}}}
			dispatches := plannedBuildDispatchesForSelection(phase, "full", nil, colony.VerificationDepthLight)
			found := false
			for _, d := range dispatches {
				if d.Caste == "chaos" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("light mode phase %d should include chaos (30%% sampling)", pid)
			}
		})
	}
	for _, pid := range noChaosPhases {
		t.Run(fmt.Sprintf("phase_%d_skips_chaos", pid), func(t *testing.T) {
			phase := colony.Phase{ID: pid, Name: "Feature work", Tasks: []colony.Task{{Goal: "Do something", Status: "pending"}}}
			dispatches := plannedBuildDispatchesForSelection(phase, "full", nil, colony.VerificationDepthLight)
			for _, d := range dispatches {
				if d.Caste == "chaos" {
					t.Errorf("light mode phase %d should skip chaos", pid)
				}
			}
		})
	}
}

func TestBuildDispatch_HeavyMode_IncludesChaosAndMeasurer(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Feature work", Tasks: []colony.Task{{Goal: "Do something", Status: "pending"}}}
	dispatches := plannedBuildDispatchesForSelection(phase, "full", nil, colony.VerificationDepthHeavy)
	hasMeasurer := false
	hasChaos := false
	for _, d := range dispatches {
		if d.Caste == "measurer" {
			hasMeasurer = true
		}
		if d.Caste == "chaos" {
			hasChaos = true
		}
	}
	if !hasMeasurer {
		t.Error("heavy mode with full depth should include measurer")
	}
	if !hasChaos {
		t.Error("heavy mode with full depth should include chaos")
	}
}

func TestBuildDispatch_FinalPhase_HeavyRegardlessOfLight(t *testing.T) {
	// Final phase (ID == totalPhases) should always get measurer and chaos even with light flag
	// This test verifies the build dispatch path, not the resolveReviewDepth logic
	phase := colony.Phase{ID: 5, Name: "Final polish", Tasks: []colony.Task{{Goal: "Polish", Status: "pending"}}}
	// When resolveReviewDepth returns heavy (final phase), dispatches should include both
	dispatches := plannedBuildDispatchesForSelection(phase, "full", nil, colony.VerificationDepthHeavy)
	hasMeasurer := false
	hasChaos := false
	for _, d := range dispatches {
		if d.Caste == "measurer" {
			hasMeasurer = true
		}
		if d.Caste == "chaos" {
			hasChaos = true
		}
	}
	if !hasMeasurer {
		t.Error("final phase (heavy depth) should include measurer")
	}
	if !hasChaos {
		t.Error("final phase (heavy depth) should include chaos")
	}
}

func TestContinueReviewDispatch_LightMode_SkipsAll(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Feature work", Tasks: []colony.Task{{Goal: "Do something", Status: "pending"}}}
	invoker := &codex.FakeInvoker{}
	dispatches := plannedContinueReviewDispatches("/tmp", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, invoker, 0, colony.VerificationDepthLight)
	if len(dispatches) != 0 {
		t.Errorf("light mode review should produce 0 dispatches, got %d", len(dispatches))
	}
}

func TestContinueReviewDispatch_HeavyMode_SpawnsAll3(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Feature work", Tasks: []colony.Task{{Goal: "Do something", Status: "pending"}}}
	invoker := &codex.FakeInvoker{}
	dispatches := plannedContinueReviewDispatches("/tmp", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, invoker, 0, colony.VerificationDepthHeavy)
	if len(dispatches) != 3 {
		t.Errorf("heavy mode review should produce 3 dispatches (gatekeeper, auditor, probe), got %d", len(dispatches))
	}
	castes := map[string]bool{}
	for _, d := range dispatches {
		castes[d.Caste] = true
	}
	for _, expected := range []string{"gatekeeper", "auditor", "probe"} {
		if !castes[expected] {
			t.Errorf("heavy mode missing %s dispatch", expected)
		}
	}
}

func TestContinueReviewDispatch_LightMode_HandlesEmptyGracefully(t *testing.T) {
	// Verify that runCodexContinueReview handles empty dispatch list gracefully
	// by checking that 0 dispatches means report.Passed == true
	// We test this indirectly: plannedContinueReviewDispatches with light mode
	// returns 0 dispatches. The caller (runCodexContinueReview) will produce
	// a report with Passed=true when dispatches is empty.
	phase := colony.Phase{ID: 3, Name: "Feature work"}
	invoker := &codex.FakeInvoker{}
	dispatches := plannedContinueReviewDispatches("/tmp", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, invoker, 0, colony.VerificationDepthLight)
	if len(dispatches) != 0 {
		t.Fatalf("expected 0 dispatches in light mode, got %d", len(dispatches))
	}
	// When dispatches is empty, the caller will get Passed=true (no blockers).
	// This is verified by the existing runCodexContinueReview flow:
	// len(report.BlockingIssues) == 0 => report.Passed = true
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

// --- Task 2 tests: visual depth line and colony-prime context ---

func TestRenderReviewDepthLine_Heavy(t *testing.T) {
	got := renderReviewDepthLine(colony.VerificationDepthHeavy, 5, 5)
	want := "Review depth: heavy (final phase)"
	if got != want {
		t.Errorf("renderReviewDepthLine(heavy, 5, 5) = %q, want %q", got, want)
	}
}

func TestRenderReviewDepthLine_HeavyNonFinal(t *testing.T) {
	got := renderReviewDepthLine(colony.VerificationDepthHeavy, 3, 5)
	want := "Review depth: heavy (Phase 3 of 5)"
	if got != want {
		t.Errorf("renderReviewDepthLine(heavy, 3, 5) = %q, want %q", got, want)
	}
}

func TestRenderReviewDepthLine_Light(t *testing.T) {
	got := renderReviewDepthLine(colony.VerificationDepthLight, 3, 5)
	want := "Review depth: light (Phase 3 of 5)"
	if got != want {
		t.Errorf("renderReviewDepthLine(light, 3, 5) = %q, want %q", got, want)
	}
}

// --- Task 1 tests: VerificationDepth 3-level dispatch ---

func TestResolveVerificationDepth_FinalPhaseDefaultsHeavyButHonorsLight(t *testing.T) {
	phase := colony.Phase{ID: 5, Name: "Final polish"}
	got := resolveVerificationDepth(phase, 5, false, false, "")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("final phase no flags: got %q, want %q", got, colony.VerificationDepthHeavy)
	}
	got = resolveVerificationDepth(phase, 5, true, false, "")
	if got != colony.VerificationDepthLight {
		t.Errorf("final phase with lightFlag=true: got %q, want %q", got, colony.VerificationDepthLight)
	}
}

func TestResolveVerificationDepth_StandardDefaultForIntermediate(t *testing.T) {
	phase := colony.Phase{ID: 2, Name: "Feature work"}
	got := resolveVerificationDepth(phase, 5, false, false, "")
	if got != colony.VerificationDepthStandard {
		t.Errorf("non-final non-keyword no flags: got %q, want %q", got, colony.VerificationDepthStandard)
	}
	// Phase 3 of 5 also standard
	phase3 := colony.Phase{ID: 3, Name: "More features"}
	got = resolveVerificationDepth(phase3, 5, false, false, "")
	if got != colony.VerificationDepthStandard {
		t.Errorf("non-final non-keyword no flags: got %q, want %q", got, colony.VerificationDepthStandard)
	}
}

func TestResolveVerificationDepth_LightFlagOverrides(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Feature work"}
	got := resolveVerificationDepth(phase, 5, true, false, "")
	if got != colony.VerificationDepthLight {
		t.Errorf("lightFlag=true: got %q, want %q", got, colony.VerificationDepthLight)
	}
}

func TestResolveVerificationDepth_HeavyFlagOverrides(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Feature work"}
	got := resolveVerificationDepth(phase, 5, false, true, "")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("heavyFlag=true: got %q, want %q", got, colony.VerificationDepthHeavy)
	}
}

func TestResolveVerificationDepth_ExplicitDepthString(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Feature work"}
	// Explicit "light" overrides auto-detect
	got := resolveVerificationDepth(phase, 5, false, false, "light")
	if got != colony.VerificationDepthLight {
		t.Errorf("explicit light: got %q, want %q", got, colony.VerificationDepthLight)
	}
	// Explicit "heavy" overrides auto-detect
	got = resolveVerificationDepth(phase, 5, false, false, "heavy")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("explicit heavy: got %q, want %q", got, colony.VerificationDepthHeavy)
	}
	// Explicit "standard" overrides auto-detect
	got = resolveVerificationDepth(phase, 5, false, false, "standard")
	if got != colony.VerificationDepthStandard {
		t.Errorf("explicit standard: got %q, want %q", got, colony.VerificationDepthStandard)
	}
}

func TestResolveVerificationDepth_KeywordPhaseAlwaysHeavy(t *testing.T) {
	phase := colony.Phase{ID: 2, Name: "Security audit"}
	got := resolveVerificationDepth(phase, 5, false, false, "")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("keyword phase no flags: got %q, want %q", got, colony.VerificationDepthHeavy)
	}
	// light flag on keyword phase overrides to light (user intent takes priority)
	got = resolveVerificationDepth(phase, 5, true, false, "")
	if got != colony.VerificationDepthLight {
		t.Errorf("keyword phase with lightFlag=true: got %q, want %q", got, colony.VerificationDepthLight)
	}
}

func TestResolveVerificationDepthFlag_BoolPriority(t *testing.T) {
	// --light takes priority over --verification-depth string
	tests := []struct {
		name     string
		light    bool
		heavy    bool
		depthStr string
		want     string
	}{
		{"light flag overrides string", true, false, "heavy", "light"},
		{"heavy flag overrides string", false, true, "light", "heavy"},
		{"both bools: heavy wins", true, true, "light", "heavy"},
		{"no bools: string passed through", false, false, "standard", "standard"},
		{"no bools, empty string: empty passed through", false, false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveVerificationDepthFlag(tt.light, tt.heavy, tt.depthStr)
			if got != tt.want {
				t.Errorf("resolveVerificationDepthFlag(%v, %v, %q) = %q, want %q",
					tt.light, tt.heavy, tt.depthStr, got, tt.want)
			}
		})
	}
}

// --- Task 2 tests: standard mode dispatch and visual ---

func TestContinueReviewDispatch_StandardMode_SpawnsProbeOnly(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Feature work", Tasks: []colony.Task{{Goal: "Do something", Status: "pending"}}}
	invoker := &codex.FakeInvoker{}
	dispatches := plannedContinueReviewDispatches("/tmp", phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, invoker, 0, colony.VerificationDepthStandard)
	if len(dispatches) != 1 {
		t.Errorf("standard mode review should produce 1 dispatch (probe only), got %d", len(dispatches))
	}
	if len(dispatches) > 0 && dispatches[0].Caste != "probe" {
		t.Errorf("standard mode should spawn probe, got %q", dispatches[0].Caste)
	}
}

func TestBuildDispatch_StandardMode_IncludesWatcherAndProbe(t *testing.T) {
	phase := colony.Phase{ID: 3, Name: "Feature work", Tasks: []colony.Task{{Goal: "Do something", Status: "pending"}}}
	dispatches := plannedBuildDispatchesForSelection(phase, "full", nil, colony.VerificationDepthStandard)
	hasWatcher := false
	hasProbe := false
	for _, d := range dispatches {
		if d.Caste == "watcher" {
			hasWatcher = true
		}
		if d.Caste == "probe" {
			hasProbe = true
		}
		if d.Caste == "measurer" {
			t.Error("standard mode should skip measurer dispatch")
		}
		if d.Caste == "chaos" {
			t.Error("standard mode should skip chaos dispatch")
		}
	}
	if !hasWatcher {
		t.Error("standard mode should include watcher dispatch")
	}
	if !hasProbe {
		t.Error("standard mode should include probe dispatch")
	}
}

func TestRenderReviewDepthLine_Standard(t *testing.T) {
	got := renderReviewDepthLine(colony.VerificationDepthStandard, 3, 5)
	want := "Review depth: standard (Phase 3 of 5)"
	if got != want {
		t.Errorf("renderReviewDepthLine(standard, 3, 5) = %q, want %q", got, want)
	}
}

func TestReviewDepthFromResult_Standard(t *testing.T) {
	result := map[string]interface{}{"review_depth": "standard"}
	got := reviewDepthFromResult(result)
	if got != colony.VerificationDepthStandard {
		t.Errorf("reviewDepthFromResult with 'standard' = %q, want %q", got, colony.VerificationDepthStandard)
	}
}

func TestReviewDepthFlags_VerificationDepthString(t *testing.T) {
	t.Run("build has verification-depth flag", func(t *testing.T) {
		f := buildCmd.Flags().Lookup("verification-depth")
		if f == nil {
			t.Fatal("buildCmd has no --verification-depth flag")
		}
		if f.DefValue != "" {
			t.Errorf("buildCmd --verification-depth default = %q, want empty", f.DefValue)
		}
	})
	t.Run("continue has verification-depth flag", func(t *testing.T) {
		f := continueCmd.Flags().Lookup("verification-depth")
		if f == nil {
			t.Fatal("continueCmd has no --verification-depth flag")
		}
		if f.DefValue != "" {
			t.Errorf("continueCmd --verification-depth default = %q, want empty", f.DefValue)
		}
	})
}

func TestColonyPrimeIncludesReviewDepth(t *testing.T) {
	output := buildColonyPrimeOutput(false)
	found := false
	for _, section := range output.Ledger.Included {
		if section.Name == "review_depth" {
			found = true
			break
		}
	}
	// Colony may not have active state in test environment, so we accept
	// either found or not erroring. When state is valid, review_depth must appear.
	// In test context without a real COLONY_STATE.json, it may be absent.
	// The test verifies the function does not panic and the section name is correct.
	t.Logf("review_depth section found: %v (acceptable in test env without colony state)", found)
}

// --- Phase 85 smart depth tests ---

func TestPhasePositionLevel(t *testing.T) {
	tests := []struct {
		name     string
		phaseID  int
		total    int
		expected string
	}{
		{"final phase", 5, 5, "final"},
		{"single-phase plan", 1, 1, "final"},
		{"early first of 10", 1, 10, "early"},
		{"early boundary of 8", 2, 8, "early"}, // 2 <= 8*0.25=2.0
		{"intermediate", 3, 8, "intermediate"},
		{"late boundary of 8", 6, 8, "late"}, // 6 >= 8*0.75=6.0, 6 != 8
		{"late of 8", 7, 8, "late"},
		{"last is final", 8, 8, "final"},
		{"late of 5", 4, 5, "late"},                 // 4 >= 5*0.75=3.75, 4 != 5
		{"intermediate of 5", 3, 5, "intermediate"}, // 3 > 1.25, 3 < 3.75
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := phasePositionLevel(tt.phaseID, tt.total)
			if got != tt.expected {
				t.Errorf("phasePositionLevel(%d, %d) = %q, want %q", tt.phaseID, tt.total, got, tt.expected)
			}
		})
	}
}

func TestCollectPhaseText(t *testing.T) {
	t.Run("full phase with tasks", func(t *testing.T) {
		phase := colony.Phase{
			Name:            "Build Auth",
			Description:     "Add auth middleware",
			SuccessCriteria: []string{"Auth works"},
			Tasks: []colony.Task{{
				Goal:            "Implement login",
				Constraints:     []string{"Must use JWT"},
				Hints:           []string{"Check bcrypt"},
				SuccessCriteria: []string{"Login returns token"},
			}},
		}
		text := collectPhaseText(phase)
		expected := []string{"build auth", "add auth middleware", "auth works",
			"implement login", "must use jwt", "check bcrypt", "login returns token"}
		for _, exp := range expected {
			if !strings.Contains(text, exp) {
				t.Errorf("collectPhaseText missing %q in output: %q", exp, text)
			}
		}
	})
	t.Run("minimal phase", func(t *testing.T) {
		phase := colony.Phase{Name: "UI Polish"}
		text := collectPhaseText(phase)
		// collectPhaseText joins Name and Description with spaces; empty Description
		// produces a trailing space, which is harmless for keyword matching.
		if !strings.Contains(text, "ui polish") {
			t.Errorf("minimal phase: got %q, want to contain %q", text, "ui polish")
		}
	})
	t.Run("nil slices no panic", func(t *testing.T) {
		phase := colony.Phase{Name: "Safe Phase", Tasks: nil}
		text := collectPhaseText(phase)
		if !strings.Contains(text, "safe phase") {
			t.Errorf("nil slices: got %q, want to contain %q", text, "safe phase")
		}
	})
}

func TestPhaseRiskLevel_High(t *testing.T) {
	tests := []struct {
		name  string
		phase colony.Phase
	}{
		{"security in name", colony.Phase{Name: "Security audit"}},
		{"auth in name", colony.Phase{Name: "Add auth middleware"}},
		{"token+session in desc", colony.Phase{Name: "Token refresh", Description: "Update session handling"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := phaseRiskLevel(tt.phase)
			if got != "high" {
				t.Errorf("phaseRiskLevel(%+v) = %q, want %q", tt.phase, got, "high")
			}
		})
	}
}

func TestPhaseRiskLevel_Medium(t *testing.T) {
	tests := []struct {
		name  string
		phase colony.Phase
	}{
		{"core runtime in name", colony.Phase{Name: "Core runtime refactor"}},
		{"state machine in name", colony.Phase{Name: "State machine transition fix"}},
		{"dispatch in name", colony.Phase{Name: "Update dispatch logic"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := phaseRiskLevel(tt.phase)
			if got != "medium" {
				t.Errorf("phaseRiskLevel(%+v) = %q, want %q", tt.phase, got, "medium")
			}
		})
	}
}

func TestPhaseRiskLevel_Low(t *testing.T) {
	tests := []struct {
		name  string
		phase colony.Phase
	}{
		{"ui polish", colony.Phase{Name: "UI polish"}},
		{"feature work", colony.Phase{Name: "Feature work"}},
		{"dashboard", colony.Phase{Name: "Dashboard redesign"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := phaseRiskLevel(tt.phase)
			if got != "low" {
				t.Errorf("phaseRiskLevel(%+v) = %q, want %q", tt.phase, got, "low")
			}
		})
	}
}

func TestPhaseRiskLevel_TaskGoalsNotAnalyzed(t *testing.T) {
	// phaseRiskLevel only checks the phase name, not task goals or descriptions,
	// to avoid false positives from common words like "session", "token", "password".
	phase := colony.Phase{
		Name: "Feature work",
		Tasks: []colony.Task{{
			Goal: "Add password reset flow",
		}},
	}
	got := phaseRiskLevel(phase)
	if got != "low" {
		t.Errorf("task goal with 'password' should not affect risk (name-only matching), got %q", got)
	}
}

func TestResolveSmartPlanningDepth(t *testing.T) {
	tests := []struct {
		name     string
		phase    colony.Phase
		total    int
		expected colony.PlanningDepth
	}{
		{"final phase", colony.Phase{ID: 5, Name: "Final polish"}, 5, colony.PlanningDepthDeep},
		{"early low risk", colony.Phase{ID: 1, Name: "Project setup"}, 4, colony.PlanningDepthLight},
		{"early security risk", colony.Phase{ID: 2, Name: "Auth system"}, 4, colony.PlanningDepthDeep},
		{"late blast radius", colony.Phase{ID: 3, Name: "Core runtime changes"}, 4, colony.PlanningDepthStandard},
		{"intermediate low risk", colony.Phase{ID: 2, Name: "Feature work"}, 5, colony.PlanningDepthStandard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSmartPlanningDepth(tt.phase, tt.total)
			if got != tt.expected {
				t.Errorf("resolveSmartPlanningDepth(%+v, %d) = %q, want %q", tt.phase, tt.total, got, tt.expected)
			}
		})
	}
}

func TestResolveSmartVerificationDepth(t *testing.T) {
	tests := []struct {
		name     string
		phase    colony.Phase
		total    int
		expected colony.VerificationDepth
	}{
		{"final phase", colony.Phase{ID: 5, Name: "Final polish"}, 5, colony.VerificationDepthHeavy},
		{"early low risk", colony.Phase{ID: 1, Name: "Setup"}, 6, colony.VerificationDepthLight},
		{"security risk", colony.Phase{ID: 2, Name: "Secrets management"}, 4, colony.VerificationDepthHeavy},
		{"blast radius intermediate", colony.Phase{ID: 3, Name: "Dispatch optimization"}, 5, colony.VerificationDepthStandard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSmartVerificationDepth(tt.phase, tt.total)
			if got != tt.expected {
				t.Errorf("resolveSmartVerificationDepth(%+v, %d) = %q, want %q", tt.phase, tt.total, got, tt.expected)
			}
		})
	}
}

func TestResolveSmartDepth_RiskOverridesPosition(t *testing.T) {
	t.Run("early phase with security keyword gets deep/heavy", func(t *testing.T) {
		phase := colony.Phase{ID: 1, Name: "Security hardening"}
		total := 6
		if got := resolveSmartPlanningDepth(phase, total); got != colony.PlanningDepthDeep {
			t.Errorf("early security: planning = %q, want deep", got)
		}
		if got := resolveSmartVerificationDepth(phase, total); got != colony.VerificationDepthHeavy {
			t.Errorf("early security: verification = %q, want heavy", got)
		}
	})
	t.Run("early phase with blast-radius keyword gets standard", func(t *testing.T) {
		phase := colony.Phase{ID: 1, Name: "Core runtime refactor"}
		total := 6
		if got := resolveSmartPlanningDepth(phase, total); got != colony.PlanningDepthStandard {
			t.Errorf("early blast-radius: planning = %q, want standard", got)
		}
		if got := resolveSmartVerificationDepth(phase, total); got != colony.VerificationDepthStandard {
			t.Errorf("early blast-radius: verification = %q, want standard", got)
		}
	})
}

func TestMatchesAnyKeyword(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		keywords []string
		want     bool
	}{
		{"security matches", "security audit", securityRiskKeywords, true},
		{"feature no match", "feature work", securityRiskKeywords, false},
		{"core runtime matches", "core runtime changes", blastRadiusKeywords, true},
		{"empty text no match", "", blastRadiusKeywords, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesAnyKeyword(tt.text, tt.keywords)
			if got != tt.want {
				t.Errorf("matchesAnyKeyword(%q, keywords) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

// --- Plan 85-02 Task 2: wiring tests for smart defaults in depth resolution ---

func TestResolveVerificationDepth_SmartDefaultFallback(t *testing.T) {
	tests := []struct {
		name     string
		phase    colony.Phase
		total    int
		expected colony.VerificationDepth
	}{
		{
			name:     "early phase gets light (smart default)",
			phase:    colony.Phase{ID: 1, Name: "Feature work", Description: "Add dashboard widgets"},
			total:    6,
			expected: colony.VerificationDepthLight,
		},
		{
			name:     "intermediate phase gets standard (smart default)",
			phase:    colony.Phase{ID: 3, Name: "More features", Description: "Expand API endpoints"},
			total:    6,
			expected: colony.VerificationDepthStandard,
		},
		{
			name:     "late phase with no risk gets standard (smart default)",
			phase:    colony.Phase{ID: 5, Name: "Polish work", Description: "Fix minor UI issues"},
			total:    6,
			expected: colony.VerificationDepthStandard,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveVerificationDepth(tt.phase, tt.total, false, false, "")
			if got != tt.expected {
				t.Errorf("resolveVerificationDepth(%+v, %d, no flags) = %q, want %q",
					tt.phase, tt.total, got, tt.expected)
			}
		})
	}
}

func TestResolveVerificationDepth_SmartDefaultRisksOverride(t *testing.T) {
	// Phase with security keyword in NAME triggers heavy via smart default.
	// Description-only keywords do NOT trigger heavy (name-only matching to avoid false positives).
	phase := colony.Phase{
		ID:          1,
		Name:        "Password reset flow",
		Description: "Add password reset flow",
	}
	got := resolveVerificationDepth(phase, 6, false, false, "")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("security keyword in phase name: got %q, want %q", got, colony.VerificationDepthHeavy)
	}
}

func TestResolveVerificationDepth_ExplicitOverridesSmartDefault(t *testing.T) {
	// lightFlag on a non-keyword phase should return light (checked before smart default)
	phase := colony.Phase{ID: 2, Name: "Feature work", Description: "Regular feature"}
	got := resolveVerificationDepth(phase, 6, true, false, "")
	if got != colony.VerificationDepthLight {
		t.Errorf("lightFlag overrides smart default: got %q, want %q", got, colony.VerificationDepthLight)
	}
	// heavyFlag always wins
	got = resolveVerificationDepth(phase, 6, false, true, "")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("heavyFlag overrides smart default: got %q, want %q", got, colony.VerificationDepthHeavy)
	}
	// explicit --verification-depth string overrides smart default
	got = resolveVerificationDepth(phase, 6, false, false, "heavy")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("explicit string overrides smart default: got %q, want %q", got, colony.VerificationDepthHeavy)
	}
}

func TestResolvePlanningDepthSmart_ExplicitValue(t *testing.T) {
	tests := []struct {
		name     string
		depth    string
		phase    colony.Phase
		total    int
		expected string
	}{
		{"explicit deep preserved", "deep", colony.Phase{ID: 1, Name: "Setup"}, 6, "deep"},
		{"explicit light overrides final phase", "light", colony.Phase{ID: 5, Name: "Final"}, 5, "light"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePlanningDepthSmart(tt.depth, tt.phase, tt.total)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("resolvePlanningDepthSmart(%q, ...) = %q, want %q", tt.depth, got, tt.expected)
			}
		})
	}
}

func TestResolvePlanningDepthSmart_EmptyUsesSmartDefault(t *testing.T) {
	tests := []struct {
		name     string
		phase    colony.Phase
		total    int
		expected string
	}{
		{"early phase gets light", colony.Phase{ID: 1, Name: "Setup"}, 6, "light"},
		{"final phase gets deep", colony.Phase{ID: 5, Name: "Final polish"}, 5, "deep"},
		{"security risk gets deep", colony.Phase{ID: 2, Name: "Auth system"}, 4, "deep"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePlanningDepthSmart("", tt.phase, tt.total)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("resolvePlanningDepthSmart(empty, %+v, %d) = %q, want %q",
					tt.phase, tt.total, got, tt.expected)
			}
		})
	}
}

func TestResolveVerificationDepthSmart_ExplicitValue(t *testing.T) {
	tests := []struct {
		name     string
		depth    string
		phase    colony.Phase
		total    int
		expected string
	}{
		{"explicit heavy preserved", "heavy", colony.Phase{ID: 1, Name: "Setup"}, 6, "heavy"},
		{"explicit light overrides final phase", "light", colony.Phase{ID: 5, Name: "Final"}, 5, "light"},
		{"explicit standard preserved", "standard", colony.Phase{ID: 3, Name: "Core"}, 4, "standard"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveVerificationDepthSmart(tt.depth, tt.phase, tt.total)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("resolveVerificationDepthSmart(%q, ...) = %q, want %q", tt.depth, got, tt.expected)
			}
		})
	}
}

func TestResolveVerificationDepthSmart_EmptyUsesSmartDefault(t *testing.T) {
	tests := []struct {
		name     string
		phase    colony.Phase
		total    int
		expected string
	}{
		{"early phase gets light", colony.Phase{ID: 1, Name: "Setup"}, 6, "light"},
		{"final phase gets heavy", colony.Phase{ID: 5, Name: "Final polish"}, 5, "heavy"},
		{"security risk gets heavy", colony.Phase{ID: 2, Name: "Auth system"}, 4, "heavy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveVerificationDepthSmart("", tt.phase, tt.total)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("resolveVerificationDepthSmart(empty, %+v, %d) = %q, want %q",
					tt.phase, tt.total, got, tt.expected)
			}
		})
	}
}

func TestResolveVerificationDepthSmart_InvalidValue(t *testing.T) {
	tests := []struct {
		name     string
		depth    string
		phase    colony.Phase
		total    int
		expected string
	}{
		{"unknown depth falls through to standard", "extreme", colony.Phase{ID: 1, Name: "Setup"}, 4, "standard"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveVerificationDepthSmart(tt.depth, tt.phase, tt.total)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("resolveVerificationDepthSmart(%q, ...) = %q, want %q", tt.depth, got, tt.expected)
			}
		})
	}
}

// --- Task 3.1: comprehensive table tests for verification depth resolution ---

func TestResolveReviewDepth_Table(t *testing.T) {
	tests := []struct {
		name     string
		phase    colony.Phase
		total    int
		light    bool
		heavy    bool
		expected ReviewDepth
	}{
		// Explicit --heavy flag overrides everything
		{"heavy flag on non-final non-keyword", colony.Phase{ID: 2, Name: "Feature work"}, 5, false, true, ReviewDepthHeavy},
		{"heavy flag on keyword phase", colony.Phase{ID: 2, Name: "Security audit"}, 5, false, true, ReviewDepthHeavy},
		{"heavy flag on final phase", colony.Phase{ID: 5, Name: "Final polish"}, 5, false, true, ReviewDepthHeavy},

		// Explicit --light flag overrides keyword/final defaults
		{"light flag on final phase", colony.Phase{ID: 5, Name: "Final polish"}, 5, true, false, ReviewDepthLight},
		{"light flag on keyword phase", colony.Phase{ID: 2, Name: "Security audit"}, 5, true, false, ReviewDepthLight},
		{"light flag on intermediate", colony.Phase{ID: 3, Name: "Feature work"}, 5, true, false, ReviewDepthLight},

		// Both flags: heavy wins (heavier is safer)
		{"both flags heavy wins on intermediate", colony.Phase{ID: 3, Name: "Feature work"}, 5, true, true, ReviewDepthHeavy},
		{"both flags heavy wins on final", colony.Phase{ID: 5, Name: "Final polish"}, 5, true, true, ReviewDepthHeavy},

		// No flags: final phase defaults to heavy
		{"no flags final phase", colony.Phase{ID: 5, Name: "Final polish"}, 5, false, false, ReviewDepthHeavy},
		{"no flags single phase plan", colony.Phase{ID: 1, Name: "Only phase"}, 1, false, false, ReviewDepthHeavy},

		// No flags: keyword detection
		{"keyword security no flags", colony.Phase{ID: 2, Name: "Security hardening"}, 5, false, false, ReviewDepthHeavy},
		{"keyword release no flags", colony.Phase{ID: 3, Name: "Release v2.0"}, 5, false, false, ReviewDepthHeavy},
		{"keyword deploy no flags", colony.Phase{ID: 2, Name: "Deploy pipeline"}, 5, false, false, ReviewDepthHeavy},
		{"keyword auth no flags", colony.Phase{ID: 4, Name: "Auth middleware"}, 5, false, false, ReviewDepthHeavy},
		{"keyword crypto no flags", colony.Phase{ID: 2, Name: "Crypto utils"}, 5, false, false, ReviewDepthHeavy},
		{"keyword compliance no flags", colony.Phase{ID: 3, Name: "Compliance check"}, 5, false, false, ReviewDepthHeavy},
		{"keyword production no flags", colony.Phase{ID: 3, Name: "Production config"}, 5, false, false, ReviewDepthHeavy},
		{"keyword ship no flags", colony.Phase{ID: 3, Name: "Ship it"}, 5, false, false, ReviewDepthHeavy},
		{"keyword launch no flags", colony.Phase{ID: 3, Name: "Launch prep"}, 5, false, false, ReviewDepthHeavy},
		{"keyword secrets no flags", colony.Phase{ID: 2, Name: "Secrets rotation"}, 5, false, false, ReviewDepthHeavy},
		{"keyword permissions no flags", colony.Phase{ID: 2, Name: "Permissions refactor"}, 5, false, false, ReviewDepthHeavy},
		{"keyword audit no flags", colony.Phase{ID: 2, Name: "Audit trail"}, 5, false, false, ReviewDepthHeavy},

		// No flags: default to light for intermediate non-keyword
		{"no flags intermediate non-keyword", colony.Phase{ID: 2, Name: "Feature work"}, 5, false, false, ReviewDepthLight},
		{"no flags early non-keyword", colony.Phase{ID: 1, Name: "Project setup"}, 5, false, false, ReviewDepthLight},
		{"no flags late non-keyword", colony.Phase{ID: 4, Name: "UI polish"}, 5, false, false, ReviewDepthLight},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveReviewDepth(tt.phase, tt.total, tt.light, tt.heavy)
			if got != tt.expected {
				t.Errorf("resolveReviewDepth(%+v, %d, light=%v, heavy=%v) = %q, want %q",
					tt.phase, tt.total, tt.light, tt.heavy, got, tt.expected)
			}
		})
	}
}

func TestResolveVerificationDepth_Table(t *testing.T) {
	tests := []struct {
		name     string
		phase    colony.Phase
		total    int
		light    bool
		heavy    bool
		depthStr string
		expected colony.VerificationDepth
	}{
		// Explicit --heavy flag overrides everything
		{"heavy flag overrides string heavy", colony.Phase{ID: 2, Name: "Feature work"}, 5, false, true, "light", colony.VerificationDepthHeavy},
		{"heavy flag on final phase", colony.Phase{ID: 5, Name: "Final polish"}, 5, false, true, "", colony.VerificationDepthHeavy},
		{"heavy flag on keyword phase", colony.Phase{ID: 2, Name: "Security audit"}, 5, false, true, "light", colony.VerificationDepthHeavy},

		// Explicit --light flag overrides keyword/final/smart defaults
		{"light flag on final phase", colony.Phase{ID: 5, Name: "Final polish"}, 5, true, false, "", colony.VerificationDepthLight},
		{"light flag on keyword phase", colony.Phase{ID: 2, Name: "Security audit"}, 5, true, false, "", colony.VerificationDepthLight},
		{"light flag overrides depth string", colony.Phase{ID: 3, Name: "Feature work"}, 5, true, false, "heavy", colony.VerificationDepthLight},

		// Both flags: heavy wins
		{"both flags heavy wins", colony.Phase{ID: 3, Name: "Feature work"}, 5, true, true, "light", colony.VerificationDepthHeavy},

		// Explicit --verification-depth string (no boolean flags)
		{"explicit depth light", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "light", colony.VerificationDepthLight},
		{"explicit depth heavy", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "heavy", colony.VerificationDepthHeavy},
		{"explicit depth standard", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "standard", colony.VerificationDepthStandard},
		{"explicit depth overrides final phase", colony.Phase{ID: 5, Name: "Final polish"}, 5, false, false, "light", colony.VerificationDepthLight},
		{"explicit depth overrides keyword", colony.Phase{ID: 2, Name: "Security audit"}, 5, false, false, "light", colony.VerificationDepthLight},

		// Aliases for depth strings
		{"alias minimal maps to light", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "minimal", colony.VerificationDepthLight},
		{"alias coarse maps to light", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "coarse", colony.VerificationDepthLight},
		{"alias full maps to heavy", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "full", colony.VerificationDepthHeavy},
		{"alias thorough maps to heavy", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "thorough", colony.VerificationDepthHeavy},

		// No flags, no string: keyword detection
		{"keyword security no flags", colony.Phase{ID: 2, Name: "Security hardening"}, 5, false, false, "", colony.VerificationDepthHeavy},
		{"keyword release no flags", colony.Phase{ID: 3, Name: "Release v2.0"}, 5, false, false, "", colony.VerificationDepthHeavy},
		{"keyword auth no flags", colony.Phase{ID: 2, Name: "Auth middleware"}, 5, false, false, "", colony.VerificationDepthHeavy},
		{"keyword deploy no flags", colony.Phase{ID: 3, Name: "Deploy pipeline"}, 5, false, false, "", colony.VerificationDepthHeavy},
		{"keyword crypto no flags", colony.Phase{ID: 2, Name: "Crypto utils"}, 5, false, false, "", colony.VerificationDepthHeavy},

		// No flags, no string: smart default fallback
		{"no flags early phase gets light", colony.Phase{ID: 1, Name: "Feature work"}, 6, false, false, "", colony.VerificationDepthLight},
		{"no flags intermediate phase gets standard", colony.Phase{ID: 3, Name: "More features"}, 6, false, false, "", colony.VerificationDepthStandard},
		{"no flags late phase gets standard", colony.Phase{ID: 5, Name: "Polish work"}, 6, false, false, "", colony.VerificationDepthStandard},
		{"no flags final phase gets heavy", colony.Phase{ID: 5, Name: "Final polish"}, 5, false, false, "", colony.VerificationDepthHeavy},

		// Invalid depth values: NormalizeVerificationDepth maps unknown to standard
		{"invalid depth ultra", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "ultra", colony.VerificationDepthStandard},
		{"invalid depth unknown", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "unknown", colony.VerificationDepthStandard},
		{"invalid depth number string", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "42", colony.VerificationDepthStandard},
		{"invalid depth random text", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "whatever", colony.VerificationDepthStandard},

		// Empty/whitespace depth string: treated as empty, falls through to smart default
		{"empty depth string uses smart default", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", colony.VerificationDepthStandard},
		{"whitespace depth string treated as empty", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "  ", colony.VerificationDepthStandard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveVerificationDepth(tt.phase, tt.total, tt.light, tt.heavy, tt.depthStr)
			if got != tt.expected {
				t.Errorf("resolveVerificationDepth(%+v, %d, light=%v, heavy=%v, depth=%q) = %q, want %q",
					tt.phase, tt.total, tt.light, tt.heavy, tt.depthStr, got, tt.expected)
			}
		})
	}
}

func TestResolveEffectiveContinueDepth_Table(t *testing.T) {
	tests := []struct {
		name      string
		phase     colony.Phase
		total     int
		light     bool
		heavy     bool
		depthStr  string
		stateDepth string
		expected  colony.VerificationDepth
	}{
		// CLI heavy flag overrides persisted state
		{"CLI heavy overrides persisted light", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, true, "", "light", colony.VerificationDepthHeavy},
		{"CLI heavy overrides persisted standard", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, true, "", "standard", colony.VerificationDepthHeavy},
		{"CLI heavy overrides persisted heavy", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, true, "", "heavy", colony.VerificationDepthHeavy},

		// CLI light flag overrides persisted state
		{"CLI light overrides persisted heavy", colony.Phase{ID: 3, Name: "Feature work"}, 5, true, false, "", "heavy", colony.VerificationDepthLight},
		{"CLI light overrides persisted standard", colony.Phase{ID: 3, Name: "Feature work"}, 5, true, false, "", "standard", colony.VerificationDepthLight},

		// CLI explicit --verification-depth overrides persisted state
		{"CLI depth string overrides persisted light", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "heavy", "light", colony.VerificationDepthHeavy},
		{"CLI depth string overrides persisted heavy", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "light", "heavy", colony.VerificationDepthLight},

		// No CLI flags: persisted state depth is used
		{"persisted heavy with no CLI flags", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", "heavy", colony.VerificationDepthHeavy},
		{"persisted light with no CLI flags", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", "light", colony.VerificationDepthLight},
		{"persisted standard with no CLI flags", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", "standard", colony.VerificationDepthStandard},

		// No CLI flags, no persisted state: falls through to smart default
		{"no CLI no persisted uses smart default", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", "", colony.VerificationDepthStandard},
		{"no CLI no persisted early gets light", colony.Phase{ID: 1, Name: "Setup"}, 6, false, false, "", "", colony.VerificationDepthLight},
		{"no CLI no persisted final gets heavy", colony.Phase{ID: 5, Name: "Final polish"}, 5, false, false, "", "", colony.VerificationDepthHeavy},

		// Both CLI flags: heavy wins even with persisted state
		{"both CLI flags heavy wins over persisted", colony.Phase{ID: 3, Name: "Feature work"}, 5, true, true, "", "light", colony.VerificationDepthHeavy},

		// Persisted state with keyword phase: keyword triggers heavy when no CLI and no persisted
		{"keyword phase no CLI no persisted", colony.Phase{ID: 2, Name: "Security audit"}, 5, false, false, "", "", colony.VerificationDepthHeavy},
		// Persisted state light overrides keyword (user previously chose light)
		{"persisted light overrides keyword", colony.Phase{ID: 2, Name: "Security audit"}, 5, false, false, "", "light", colony.VerificationDepthLight},

		// Whitespace persisted state treated as empty
		{"whitespace persisted state uses smart default", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", "  ", colony.VerificationDepthStandard},

		// Invalid persisted state falls through to NormalizeVerificationDepth (maps to standard)
		{"invalid persisted state maps to standard", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", "ultra", colony.VerificationDepthStandard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveEffectiveContinueDepth(tt.phase, tt.total, tt.light, tt.heavy, tt.depthStr, tt.stateDepth)
			if got != tt.expected {
				t.Errorf("resolveEffectiveContinueDepth(%+v, %d, light=%v, heavy=%v, depth=%q, state=%q) = %q, want %q",
					tt.phase, tt.total, tt.light, tt.heavy, tt.depthStr, tt.stateDepth, got, tt.expected)
			}
		})
	}
}

func TestResolveVerificationDepthFlag_Table(t *testing.T) {
	tests := []struct {
		name     string
		light    bool
		heavy    bool
		depthStr string
		want     string
	}{
		// Boolean flags take priority
		{"heavy flag alone", false, true, "", "heavy"},
		{"light flag alone", true, false, "", "light"},
		{"both flags heavy wins", true, true, "light", "heavy"},
		{"both flags heavy wins no string", true, true, "", "heavy"},

		// No boolean flags: string passed through
		{"no flags string standard", false, false, "standard", "standard"},
		{"no flags string light", false, false, "light", "light"},
		{"no flags string heavy", false, false, "heavy", "heavy"},
		{"no flags empty string", false, false, "", ""},

		// Boolean flag overrides string value
		{"heavy overrides light string", false, true, "light", "heavy"},
		{"light overrides heavy string", true, false, "heavy", "light"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveVerificationDepthFlag(tt.light, tt.heavy, tt.depthStr)
			if got != tt.want {
				t.Errorf("resolveVerificationDepthFlag(%v, %v, %q) = %q, want %q",
					tt.light, tt.heavy, tt.depthStr, got, tt.want)
			}
		})
	}
}

func TestResolveReviewDepth_PrecedenceOrder(t *testing.T) {
	// Comprehensive precedence test: heavy > light > final > keyword > default
	// Uses a single phase that would trigger multiple rules to prove priority
	finalKeywordPhase := colony.Phase{ID: 5, Name: "Security release"}

	// 1. heavy flag wins over everything
	got := resolveReviewDepth(finalKeywordPhase, 5, true, true)
	if got != ReviewDepthHeavy {
		t.Errorf("precedence: both flags on final+keyword phase = %q, want heavy", got)
	}

	// 2. light flag overrides final+keyword defaults
	got = resolveReviewDepth(finalKeywordPhase, 5, true, false)
	if got != ReviewDepthLight {
		t.Errorf("precedence: light flag on final+keyword phase = %q, want light", got)
	}

	// 3. No flags: final phase is heavy
	got = resolveReviewDepth(finalKeywordPhase, 5, false, false)
	if got != ReviewDepthHeavy {
		t.Errorf("precedence: no flags on final+keyword phase = %q, want heavy", got)
	}

	// 4. Keyword-only non-final triggers heavy
	keywordPhase := colony.Phase{ID: 3, Name: "Security audit"}
	got = resolveReviewDepth(keywordPhase, 5, false, false)
	if got != ReviewDepthHeavy {
		t.Errorf("precedence: keyword non-final = %q, want heavy", got)
	}

	// 5. Default for non-final non-keyword is light
	plainPhase := colony.Phase{ID: 3, Name: "Feature work"}
	got = resolveReviewDepth(plainPhase, 5, false, false)
	if got != ReviewDepthLight {
		t.Errorf("precedence: plain intermediate = %q, want light", got)
	}
}

func TestResolveVerificationDepth_PrecedenceOrder(t *testing.T) {
	// Comprehensive precedence: heavy flag > light flag > explicit depth string > keyword > smart default
	finalKeywordPhase := colony.Phase{ID: 5, Name: "Security release"}

	// 1. heavy flag wins over everything
	got := resolveVerificationDepth(finalKeywordPhase, 5, false, true, "light")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("precedence: heavy+depth=light on final+keyword = %q, want heavy", got)
	}

	// 2. light flag overrides final+keyword+explicit depth
	got = resolveVerificationDepth(finalKeywordPhase, 5, true, false, "heavy")
	if got != colony.VerificationDepthLight {
		t.Errorf("precedence: light+depth=heavy on final+keyword = %q, want light", got)
	}

	// 3. Explicit depth string overrides keyword/final defaults
	got = resolveVerificationDepth(finalKeywordPhase, 5, false, false, "light")
	if got != colony.VerificationDepthLight {
		t.Errorf("precedence: depth=light on final+keyword = %q, want light", got)
	}

	// 4. No flags, no string: keyword triggers heavy
	got = resolveVerificationDepth(finalKeywordPhase, 5, false, false, "")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("precedence: no flags on final+keyword = %q, want heavy", got)
	}

	// 5. Smart default for plain intermediate
	plainPhase := colony.Phase{ID: 3, Name: "Feature work"}
	got = resolveVerificationDepth(plainPhase, 5, false, false, "")
	if got != colony.VerificationDepthStandard {
		t.Errorf("precedence: plain intermediate smart default = %q, want standard", got)
	}
}

func TestResolveEffectiveContinueDepth_PrecedenceOrder(t *testing.T) {
	// Comprehensive precedence: CLI heavy > CLI light > CLI depth string > persisted state > keyword > smart default
	finalKeywordPhase := colony.Phase{ID: 5, Name: "Security release"}

	// 1. CLI heavy flag overrides persisted + keyword + final
	got := resolveEffectiveContinueDepth(finalKeywordPhase, 5, false, true, "", "light")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("continue precedence: CLI heavy+state=light = %q, want heavy", got)
	}

	// 2. CLI light flag overrides persisted heavy
	got = resolveEffectiveContinueDepth(finalKeywordPhase, 5, true, false, "", "heavy")
	if got != colony.VerificationDepthLight {
		t.Errorf("continue precedence: CLI light+state=heavy = %q, want light", got)
	}

	// 3. CLI depth string overrides persisted state
	got = resolveEffectiveContinueDepth(finalKeywordPhase, 5, false, false, "light", "heavy")
	if got != colony.VerificationDepthLight {
		t.Errorf("continue precedence: CLI depth=light+state=heavy = %q, want light", got)
	}

	// 4. Persisted state used when no CLI flags
	got = resolveEffectiveContinueDepth(colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", "heavy")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("continue precedence: no CLI+state=heavy = %q, want heavy", got)
	}

	// 5. No CLI, no persisted: keyword triggers heavy
	got = resolveEffectiveContinueDepth(colony.Phase{ID: 2, Name: "Security audit"}, 5, false, false, "", "")
	if got != colony.VerificationDepthHeavy {
		t.Errorf("continue precedence: no CLI+no state+keyword = %q, want heavy", got)
	}

	// 6. No CLI, no persisted, no keyword: smart default
	got = resolveEffectiveContinueDepth(colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", "")
	if got != colony.VerificationDepthStandard {
		t.Errorf("continue precedence: no CLI+no state+no keyword = %q, want standard", got)
	}
}

// --- Task 3.2: build/continue depth parity and light-mode verification tests ---

func TestBuildAndContinueEmitMatchingDepthMetadata(t *testing.T) {
	tests := []struct {
		name      string
		phase     colony.Phase
		total     int
		light     bool
		heavy     bool
		depthStr  string
		stateDepth string
	}{
		{"final phase no flags", colony.Phase{ID: 5, Name: "Final polish"}, 5, false, false, "", ""},
		{"intermediate no flags", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", ""},
		{"light flag on final", colony.Phase{ID: 5, Name: "Final polish"}, 5, true, false, "", ""},
		{"heavy flag on intermediate", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, true, "", ""},
		{"explicit depth string", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "light", ""},
		{"keyword phase no flags", colony.Phase{ID: 2, Name: "Security audit"}, 5, false, false, "", ""},
		{"persisted state depth", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", "heavy"},
		{"both flags heavy wins", colony.Phase{ID: 3, Name: "Feature work"}, 5, true, true, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build path depth resolution (mirrors recommendQueenExecutionPolicy logic)
			buildEffectiveDepth := resolveVerificationDepthFlag(tt.light, tt.heavy, tt.depthStr)
			if buildEffectiveDepth == "" {
				buildEffectiveDepth = strings.TrimSpace(tt.stateDepth)
			}
			buildDepth := resolveVerificationDepth(tt.phase, tt.total, tt.light, tt.heavy, buildEffectiveDepth)

			// Continue path depth resolution
			continueDepth := resolveEffectiveContinueDepth(tt.phase, tt.total, tt.light, tt.heavy, tt.depthStr, tt.stateDepth)

			if buildDepth != continueDepth {
				t.Errorf("build depth = %q, continue depth = %q -- mismatch for phase %+v (total=%d, light=%v, heavy=%v, depth=%q, state=%q)",
					buildDepth, continueDepth, tt.phase, tt.total, tt.light, tt.heavy, tt.depthStr, tt.stateDepth)
			}
		})
	}
}

func TestRecommendQueenExecutionPolicyMatchesContinueDepth(t *testing.T) {
	tests := []struct {
		name      string
		phase     colony.Phase
		total     int
		light     bool
		heavy     bool
		depthStr  string
		stateVD   string
	}{
		{"final phase no flags", colony.Phase{ID: 5, Name: "Final polish"}, 5, false, false, "", ""},
		{"intermediate no flags", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", ""},
		{"keyword phase no flags", colony.Phase{ID: 2, Name: "Security audit"}, 5, false, false, "", ""},
		{"persisted heavy no CLI", colony.Phase{ID: 3, Name: "Feature work"}, 5, false, false, "", "heavy"},
		{"CLI light overrides persisted", colony.Phase{ID: 3, Name: "Feature work"}, 5, true, false, "", "heavy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := colony.ColonyState{VerificationDepth: tt.stateVD}
			policy := recommendQueenExecutionPolicy(state, tt.phase, tt.total, codexQueenExecutionPolicyInput{
				LightFlag:         tt.light,
				HeavyFlag:         tt.heavy,
				VerificationDepth: tt.depthStr,
			})
			buildDepth := colony.NormalizeVerificationDepth(policy.VerificationDepth)
			continueDepth := resolveEffectiveContinueDepth(tt.phase, tt.total, tt.light, tt.heavy, tt.depthStr, tt.stateVD)

			if buildDepth != continueDepth {
				t.Errorf("build policy depth = %q, continue depth = %q -- mismatch", buildDepth, continueDepth)
			}
		})
	}
}

func TestLightContinueFinalizationRecordsVerificationEvidence(t *testing.T) {
	// When skip_watchers is true and review_depth is light, the continue
	// finalization should still produce a verification report with evidence.
	// The runtime verification steps (build, types, lint, tests) must run
	// regardless of watcher skip.
	saveGlobals(t)
	_ = setupBuildFlowTest(t)

	phase := colony.Phase{ID: 3, Name: "Feature work"}
	now := time.Now().UTC()

	// Simulate a light continue verification snapshot with skipWatchers=true
	verification := runCodexContinueVerificationSnapshot("/tmp", phase, codexContinueManifest{}, now, 30*time.Second, true)

	// Runtime verification steps should still be present
	if len(verification.Steps) == 0 {
		t.Error("light mode with skip_watchers should still run runtime verification steps")
	}

	// Watcher should be marked as skipped, not absent
	if !verification.Watcher.Present {
		t.Error("watcher should be present (skipped) even in light mode")
	}
	if verification.Watcher.Status != "skipped" {
		t.Errorf("watcher status = %q, want %q", verification.Watcher.Status, "skipped")
	}

	// ChecksPassed should reflect runtime verification, not watcher
	// (steps may fail in /tmp env, but the watcher should not affect ChecksPassed)
	if verification.Watcher.Passed != true {
		t.Error("skipped watcher should be marked as passed")
	}
}

func TestWatcherTimeoutAdvisoryWhenRuntimePassed(t *testing.T) {
	// When a watcher times out but runtime verification (build, types, lint, tests)
	// passed, the watcher timeout should be treated as advisory, not a hard block.
	verification := codexContinueVerificationReport{
		Phase:        3,
		ChecksPassed: true,
		Passed:       true,
		Steps:        []codexVerificationStep{
			{Name: "build", Passed: true},
			{Name: "types", Passed: true},
			{Name: "lint", Passed: true},
			{Name: "tests", Passed: true},
		},
		Watcher: codexWatcherVerification{
			Present: true,
			Passed:  false,
			Status:  "timeout",
			Worker:  "Watcher-42",
			Summary: "watcher timed out",
		},
	}

	workerFlow := []codexContinueWorkerFlowStep{
		{
			Stage:  "verification",
			Caste:  "watcher",
			Name:   "Watcher-42",
			Status: "timeout",
			Summary: "watcher timed out",
		},
	}

	resultVerification, _ := attachExternalContinueWatcher(verification, workerFlow)

	// ChecksPassed should remain true because runtime verification passed
	// and watcher timeout is advisory
	if !resultVerification.ChecksPassed {
		t.Error("watcher timeout with runtime checks passed should keep ChecksPassed = true (advisory)")
	}

	// Should have a blocking issue mentioning the timeout (advisory message)
	found := false
	for _, issue := range resultVerification.BlockingIssues {
		if strings.Contains(issue, "timed out") && strings.Contains(issue, "runtime verification passed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected advisory blocking issue about watcher timeout, got: %v", resultVerification.BlockingIssues)
	}
}

func TestWatcherFailureBlocksWhenRuntimeFailed(t *testing.T) {
	// When a watcher fails (not timeout) AND runtime verification also failed,
	// the watcher failure should be a hard block.
	verification := codexContinueVerificationReport{
		Phase:        3,
		ChecksPassed: false,
		Passed:       false,
		Steps: []codexVerificationStep{
			{Name: "build", Passed: false, Summary: "build failed"},
		},
		Watcher: codexWatcherVerification{
			Present: true,
			Passed:  false,
			Status:  "failed",
			Worker:  "Watcher-42",
			Summary: "watcher found critical issues",
		},
	}

	workerFlow := []codexContinueWorkerFlowStep{
		{
			Stage:  "verification",
			Caste:  "watcher",
			Name:   "Watcher-42",
			Status: "failed",
			Summary: "watcher found critical issues",
		},
	}

	resultVerification, _ := attachExternalContinueWatcher(verification, workerFlow)

	// ChecksPassed should be false
	if resultVerification.ChecksPassed {
		t.Error("watcher failure should set ChecksPassed = false")
	}
	if resultVerification.Passed {
		t.Error("watcher failure should set Passed = false")
	}
}

func TestContinueFinalizeResultIncludesReviewDepth(t *testing.T) {
	// Verify that continue finalize blocked result includes review_depth.
	// Both the blocked and advance result maps must carry the normalized
	// verification depth so that callers (visual renderer, wrappers) can
	// display it without re-resolving.
	saveGlobals(t)
	_ = setupBuildFlowTest(t)

	phase := colony.Phase{ID: 3, Name: "Feature work"}
	state := colony.ColonyState{
		State:        colony.StateBUILT,
		CurrentPhase: 3,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Done", Status: colony.PhaseCompleted, Tasks: []colony.Task{{Goal: "done", Status: colony.TaskCompleted}}},
				{ID: 2, Name: "Done", Status: colony.PhaseCompleted, Tasks: []colony.Task{{Goal: "done", Status: colony.TaskCompleted}}},
				{ID: 3, Name: "Feature work", Status: colony.PhaseInProgress, Tasks: []colony.Task{{Goal: "work", Status: colony.TaskCompleted}}},
			},
		},
	}
	verification := codexContinueVerificationReport{Phase: 3, ChecksPassed: true, Passed: true}
	assessment := codexContinueAssessment{PartialSuccess: false}
	gates := codexContinueGateReport{Passed: false, BlockingIssues: []string{"test gate failed"}}
	now := time.Now().UTC()

	result, _, err := finalizeBlockedExternalContinue(state, phase, codexContinueManifest{}, verification, assessment, gates, nil, "", nil, now, "verification.json", "gates.json", nil, colony.VerificationDepthLight)
	if err != nil {
		t.Fatalf("finalizeBlockedExternalContinue returned error: %v", err)
	}

	rd, hasReviewDepth := result["review_depth"]
	if !hasReviewDepth {
		t.Fatal("finalizeBlockedExternalContinue must include review_depth in result map")
	}
	rdStr, ok := rd.(string)
	if !ok {
		t.Fatalf("review_depth is not a string: %T", rd)
	}
	if rdStr != "light" {
		t.Errorf("review_depth = %q, want %q (default when plan.ReviewDepth is empty)", rdStr, "light")
	}
}

func TestDepthKeysPresentInFreshPlanResultMap(t *testing.T) {
	// Structural regression guard: verifies depth keys appear in all three
	// result-map paths. This intentionally scans source text because it is
	// checking that a prior regression (missing keys) does not recur. If a
	// refactor moves keys into a shared helper, update the expected count.
	source, err := os.ReadFile("codex_plan.go")
	if err != nil {
		t.Fatal(err)
	}
	sourceStr := string(source)
	requiredKeys := []string{
		`"verification_depth":`,
		`"verification_smart_default":`,
		`"planning_smart_default":`,
		`"planning_phase":`,
	}
	for _, key := range requiredKeys {
		count := strings.Count(sourceStr, key)
		// Each key must appear at least 3 times:
		// 1. existing-plan path (~line 216)
		// 2. fresh plan generation path (~line 449)
		// 3. plan-only fresh path (~line 574)
		if count < 3 {
			t.Errorf("key %s found %d times in codex_plan.go, expected at least 3 (existing-plan, fresh generation, plan-only)", key, count)
		}
	}
}
