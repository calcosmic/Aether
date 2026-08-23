package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestCLAUDEMDVerificationDepthClaims(t *testing.T) {
	data, err := os.ReadFile("../CLAUDE.md")
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	content := string(data)

	// Check that CLAUDE.md documents the 5-level priority chain
	if !strings.Contains(content, "Explicit `--heavy` or `--light` flag") {
		t.Error("CLAUDE.md missing explicit flag priority")
	}
	if !strings.Contains(content, "Explicit `--verification-depth") {
		t.Error("CLAUDE.md missing explicit --verification-depth priority")
	}
	if !strings.Contains(content, "Keyword match in phase name") {
		t.Error("CLAUDE.md missing keyword match priority")
	}
	if !strings.Contains(content, "Smart default based on phase mode") {
		t.Error("CLAUDE.md missing smart default priority")
	}

	// Check that CLAUDE.md documents smart defaults correctly
	if !strings.Contains(content, "Discovery mode → light") {
		t.Error("CLAUDE.md missing discovery=light rule")
	}
	if !strings.Contains(content, "Production mode → at least standard") {
		t.Error("CLAUDE.md missing production=standard rule")
	}
	if !strings.Contains(content, "Final phase → heavy") {
		t.Error("CLAUDE.md missing final=heavy rule")
	}
	if !strings.Contains(content, "security") {
		t.Error("CLAUDE.md missing security keyword rule")
	}

	// Check that CLAUDE.md documents what each depth means
	if !strings.Contains(content, "Light") {
		t.Error("CLAUDE.md missing Light depth description")
	}
	if !strings.Contains(content, "Standard") {
		t.Error("CLAUDE.md missing Standard depth description")
	}
	if !strings.Contains(content, "Heavy") {
		t.Error("CLAUDE.md missing Heavy depth description")
	}

	// Verify claims against runtime behavior
	phase := colony.Phase{ID: 1, Name: "Security hardening", Mode: colony.PhaseModeProduction}
	depth := resolveVerificationDepth(phase, 5, false, false, "")
	if depth != colony.VerificationDepthHeavy {
		t.Errorf("security phase should get heavy, got %s", depth)
	}

	discoveryPhase := colony.Phase{ID: 2, Name: "Documentation cleanup", Mode: colony.PhaseModeDiscovery}
	discoveryDepth := resolveVerificationDepth(discoveryPhase, 5, false, false, "")
	if discoveryDepth != colony.VerificationDepthLight {
		t.Errorf("discovery phase should get light, got %s", discoveryDepth)
	}

	finalPhase := colony.Phase{ID: 5, Name: "Polish", Mode: colony.PhaseModePrototype}
	finalDepth := resolveVerificationDepth(finalPhase, 5, false, false, "")
	if finalDepth != colony.VerificationDepthHeavy {
		t.Errorf("final phase should get heavy, got %s", finalDepth)
	}
}

func TestCLAUDEMDNoOldModes(t *testing.T) {
	data, err := os.ReadFile("../CLAUDE.md")
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	content := string(data)

	// The old 3-mode description should be gone
	if strings.Contains(content, "`fast`: low-risk work; light continue verification") {
		t.Error("CLAUDE.md still contains old 'fast' mode description")
	}
	if strings.Contains(content, "watcher subprocess skipped") {
		t.Error("CLAUDE.md still contains 'watcher subprocess skipped'")
	}
}

// TestBuildWorkerCapHonoursVerificationDepth asserts the numbers CLAUDE.md
// promises, not the prose.
//
// The table in CLAUDE.md ("Build | Light max 5 | Standard max 6 | Heavy max 8")
// was false for months. Those numbers existed in queenMaxWorkersForBudget but
// were keyed to phase risk and mode, and the build branch never consulted depth
// at all — so the promise was inverted in the cases that mattered: a *light*
// build of a production phase returned 8, a *heavy* build of a discovery phase
// returned 5.
//
// TestCLAUDEMDVerificationDepthClaims above could not catch that: it asserts
// the words "Light", "Standard" and "Heavy" appear in the file and never
// evaluates a single worker count. This test evaluates the counts, so the
// documented contract cannot drift from the code again.
func TestBuildWorkerCapHonoursVerificationDepth(t *testing.T) {
	production := colony.Phase{ID: 1, Name: "Release hardening", Mode: colony.PhaseModeProduction}
	discovery := colony.Phase{ID: 2, Name: "Explore options", Mode: colony.PhaseModeDiscovery}

	cases := []struct {
		name    string
		phase   colony.Phase
		risk    string
		depth   colony.VerificationDepth
		want    int
		because string
	}{
		// The two inversions that proved the dial was disconnected.
		{"light caps a production build", production, "high", colony.VerificationDepthLight, 5,
			"light must cap even a high-risk build; safety castes bypass the budget separately"},
		{"heavy raises a discovery build", discovery, "low", colony.VerificationDepthHeavy, 8,
			"heavy must raise a cheap phase, or choosing heavy buys nothing"},

		// The documented caps.
		{"light caps at five", production, "high", colony.VerificationDepthLight, 5, "CLAUDE.md light = max 5"},
		{"heavy reaches eight", production, "high", colony.VerificationDepthHeavy, 8, "CLAUDE.md heavy = 8"},

		// Standard must NOT clamp. It means the Queen's ordinary judgement, which
		// mode and risk already express. A flat standard ceiling weakened exactly
		// the phases that need most help — it cost a DB-migration phase its
		// Architect before this case existed.
		{"standard leaves a high-risk build alone", production, "high", colony.VerificationDepthStandard, 8,
			"standard must not cap a phase whose own risk earned 8"},
		{"standard leaves a discovery build alone", discovery, "low", colony.VerificationDepthStandard, 5,
			"standard must not raise a cheap phase either"},

		// Depth must not inflate a budget that is already below its cap.
		{"light leaves a smaller budget alone", discovery, "low", colony.VerificationDepthLight, 5,
			"a discovery build already sits at 5; light must not raise it"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := colony.ColonyState{VerificationDepth: string(tc.depth)}
			got, reason := queenMaxWorkersForBudget(tc.phase, "build", state, tc.risk)
			if got != tc.want {
				t.Errorf("max workers = %d, want %d (%s)\n  reason given: %q", got, tc.want, tc.because, reason)
			}
		})
	}
}

// TestBuildDepthCapDoesNotStripRequiredSafetyCastes guards the reason the cap
// is safe to apply. Lowering the worker budget must never remove a caste a
// phase genuinely requires.
//
// Plan 194-02 (D-05, D-07) moved WHERE that requirement comes from: it used
// to be "a production phase requires a quality and security reviewer" at
// build (queenBuildSafetyRequiredCastes, now deleted); it is now "a phase
// whose wording names a risk signal keeps its forced reviewer", asserted at
// the continue step where reviewers are forced under the new rules. The
// claim this test guards is unchanged -- a light worker budget must never
// silently drop a reviewer the phase actually needs -- only its fixture and
// flow moved to match where the floor now lives.
func TestBuildDepthCapDoesNotStripRequiredSafetyCastes(t *testing.T) {
	phase := colony.Phase{ID: 1, Name: "Password reset", Description: "Let users reset their password by email", Mode: colony.PhaseModePrototype}

	dispatches := queenContinueDispatchesWithJudgement(phase, colony.VerificationDepthLight, nil, "", nil)
	found := false
	for _, dispatch := range dispatches {
		if dispatch.Caste == "gatekeeper" {
			found = true
		}
	}
	if !found {
		t.Fatalf("gatekeeper is missing from a light-depth continue dispatch for a phase whose wording names a security signal; light must not strip a forced reviewer: %+v", dispatches)
	}
}
