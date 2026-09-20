package cmd

import (
	"os"
	"regexp"
	"strconv"
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

	// D-06 (194-CONTEXT.md, plan 194-05): position no longer raises
	// verification depth on its own -- a low-risk final phase gets the same
	// depth a low-risk middle phase gets. CLAUDE.md no longer claims
	// otherwise (194-09 removed the "Final phase → heavy" line entirely, so
	// there is nothing left for this test to check the PROSE against); this
	// assertion is now purely a runtime regression guard.
	finalPhase := colony.Phase{ID: 5, Name: "Polish", Mode: colony.PhaseModePrototype}
	finalDepth := resolveVerificationDepth(finalPhase, 5, false, false, "")
	if finalDepth != colony.VerificationDepthStandard {
		t.Errorf("final phase (low risk) should get standard now that position no longer raises depth (D-06), got %s", finalDepth)
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
	// D-06/D-11 (plan 194-09): no shipped instruction file may still claim a
	// phase's position in the plan raises its verification depth, or that a
	// build always gets a Watcher -- both floors were retired by ruling D11
	// (2026-08-22) and plan 194-05.
	if strings.Contains(content, "Final phase → heavy") {
		t.Error("CLAUDE.md still claims phase position raises verification depth (D-06 retired this)")
	}
	if strings.Contains(content, "always, on every build") {
		t.Error("CLAUDE.md still claims a caste is required on every build unconditionally")
	}
	if strings.Contains(content, "always — unchanged, and must stay so") {
		t.Error("CLAUDE.md still claims the Watcher is unconditionally required")
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

	dispatches := queenContinueDispatchesWithJudgement(phase, colony.VerificationDepthLight, nil, "", nil, nil)
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

// TestCLAUDEMDDepthTableEvaluates parses the "Continue" row of CLAUDE.md's
// "What each depth means" table and evaluates its light/standard/heavy cells
// against the real runtime -- queenMaxWorkersForBudget for the worker caps,
// isAlwaysRequired for which castes are unconditionally required at each
// depth -- the same discipline TestBuildWorkerCapHonoursVerificationDepth
// already applies to the build row, extended here to continue.
//
// TestCLAUDEMDVerificationDepthClaims only ever checked that words like
// "Light"/"Standard"/"Heavy" appeared somewhere in the file. That let the
// continue row describe a floor (Watcher-only at light, Watcher+Probe at
// standard) that plan 194-05 deleted from the code months before this test
// existed to catch it. Because this test reads the row's own text rather
// than a hardcoded expectation, editing the continue row of the table to
// name a different caste set than isAlwaysRequired actually returns makes
// this test fail -- the documented claim and the runtime cannot drift apart
// silently again.
func TestCLAUDEMDDepthTableEvaluates(t *testing.T) {
	data, err := os.ReadFile("../CLAUDE.md")
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	content := string(data)

	var continueLine string
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "| **Continue**") {
			continueLine = line
			break
		}
	}
	if continueLine == "" {
		t.Fatalf("could not find the Continue row of the depth table in CLAUDE.md")
	}

	var cells []string
	for _, part := range strings.Split(continueLine, "|") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" || trimmed == "**Continue**" {
			continue
		}
		cells = append(cells, trimmed)
	}
	if len(cells) != 3 {
		t.Fatalf("expected exactly 3 depth cells (light, standard, heavy) in CLAUDE.md's Continue row, found %d: %v", len(cells), cells)
	}
	light, standard, heavy := cells[0], cells[1], cells[2]

	maxPattern := regexp.MustCompile(`max (\d+)`)
	extractMax := func(cell string) int {
		m := maxPattern.FindStringSubmatch(cell)
		if m == nil {
			t.Fatalf("could not find a worker cap ('max N') in cell %q", cell)
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("could not parse worker cap %q: %v", m[1], err)
		}
		return n
	}

	phase := colony.Phase{ID: 1, Name: "Sample phase", Mode: colony.PhaseModePrototype}
	lightState := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}
	standardState := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}
	heavyState := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)}

	if got, _ := queenMaxWorkersForBudget(phase, "continue", lightState, "low"); got != extractMax(light) {
		t.Errorf("CLAUDE.md's continue/light cap says %d, queenMaxWorkersForBudget returns %d", extractMax(light), got)
	}
	if got, _ := queenMaxWorkersForBudget(phase, "continue", standardState, "low"); got != extractMax(standard) {
		t.Errorf("CLAUDE.md's continue/standard cap says %d, queenMaxWorkersForBudget returns %d", extractMax(standard), got)
	}
	if got, _ := queenMaxWorkersForBudget(phase, "continue", heavyState, "low"); got != extractMax(heavy) {
		t.Errorf("CLAUDE.md's continue/heavy cap says %d, queenMaxWorkersForBudget returns %d", extractMax(heavy), got)
	}

	// D-13: light and standard require NOTHING unconditionally at continue.
	// The doc's own light/standard cells must not name a reviewer caste, and
	// isAlwaysRequired must agree that none of them is unconditionally
	// required at these depths.
	for _, tc := range []struct {
		label string
		cell  string
		state colony.ColonyState
	}{
		{"light", light, lightState},
		{"standard", standard, standardState},
	} {
		for _, caste := range []string{"gatekeeper", "auditor", "probe", "watcher"} {
			namedInDoc := strings.Contains(strings.ToLower(tc.cell), caste)
			requiredByRuntime := isAlwaysRequired(caste, "continue", phase, tc.state)
			if namedInDoc || requiredByRuntime {
				t.Errorf("continue/%s: doc names %s=%v, isAlwaysRequired=%v -- D-13 requires neither to be true at this depth", tc.label, caste, namedInDoc, requiredByRuntime)
			}
		}
	}

	// D-13: heavy requires the security and quality reviewer unconditionally
	// (gatekeeper, auditor) -- the doc's heavy cell must name them and
	// isAlwaysRequired must agree. (Probe is deliberately excluded here: it
	// is conditional on testable code even at heavy, so "named in the doc"
	// and "unconditionally required" are not the same claim for it.)
	for _, caste := range []string{"gatekeeper", "auditor"} {
		namedInDoc := strings.Contains(strings.ToLower(heavy), caste)
		requiredByRuntime := isAlwaysRequired(caste, "continue", phase, heavyState)
		if !namedInDoc {
			t.Errorf("continue/heavy cell does not name %s, but heavy requires it unconditionally", caste)
		}
		if namedInDoc != requiredByRuntime {
			t.Errorf("continue/heavy: doc names %s=%v, isAlwaysRequired=%v -- these must agree", caste, namedInDoc, requiredByRuntime)
		}
	}
}
