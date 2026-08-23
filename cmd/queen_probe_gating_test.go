package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func probeGatingPhase(name, description string, mode colony.PhaseMode) colony.Phase {
	return colony.Phase{
		ID:          1,
		Name:        name,
		Description: description,
		Mode:        mode,
	}
}

// TestProbeIsRequiredOnlyWhereItCanFindSomething is the guard on the spawn
// change users actually feel.
//
// Plan 194-02 (D-07) deleted queenBuildSafetyRequiredCastes' unconditional
// Probe membership entirely -- Probe is no longer a REQUIRED build caste
// under any condition, so the positive half of this test (an implementation
// phase "requires" a Probe) no longer has a floor to assert. What survives
// is the negative rule: Probe must never be forced onto a phase with nothing
// for it to cover. Refusing an explicit Queen proposal naming Probe on such a
// phase is plan 194-05's job (the queenApplyJudgement zero-relevance refusal
// only fires today when casteRelevanceScore returns 0, and Probe is not yet
// keyword-gated the way ambassador/gatekeeper are) -- skipped here rather
// than landed inline, since gating Probe's score changes candidate selection
// for every flow that scores it (build, continue, swarm, seal), not just
// this floor.
func TestProbeIsRequiredOnlyWhereItCanFindSomething(t *testing.T) {
	for _, tc := range []struct {
		name         string
		phase        colony.Phase
		wantTestable bool
		wantReason   string
	}{
		{
			name:         "documentation-only phase produces no testable code",
			phase:        probeGatingPhase("Write the README", "Documentation for installation and usage", colony.PhaseModeMaintenance),
			wantTestable: false,
			wantReason:   "no code produced, nothing to cover",
		},
		{
			name:         "discovery phase produces no testable code",
			phase:        probeGatingPhase("Evaluate queue options", "Research candidate message brokers", colony.PhaseModeDiscovery),
			wantTestable: false,
			wantReason:   "research produces findings, not code",
		},
		{
			name:         "docs phase that also ships code still produces testable code",
			phase:        probeGatingPhase("Document and extend the API", "Update the guide and implement the /health endpoint", colony.PhaseModeProduction),
			wantTestable: true,
			wantReason:   "mixed phase still produces testable code",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := queenPhaseProducesTestableCode(tc.phase); got != tc.wantTestable {
				t.Errorf("queenPhaseProducesTestableCode = %v, want %v (%s)", got, tc.wantTestable, tc.wantReason)
			}
		})
	}

	t.Run("proposing probe on a documentation-only phase is not yet refused (plan 194-05)", func(t *testing.T) {
		t.Skip("Probe is not keyword-gated like ambassador/gatekeeper, so an explicit proposal naming it is not refused on zero relevance today; plan 194-05 wires the refusal gate and should remove this skip.")
	})
}

// TestHeavyContinueProbeRequiresTestableCode pins D-13: `--heavy` is the full
// review panel (gatekeeper + auditor + probe), but "full panel" does not
// override probe's own reason for existing. This test used to assert the
// opposite -- that heavy kept Probe even on a documentation phase -- which is
// exactly the unconditional-Probe cost D-07 removes; D-13 makes heavy no
// exception to it.
func TestHeavyContinueProbeRequiresTestableCode(t *testing.T) {
	state := colony.ColonyState{}
	state.VerificationDepth = string(colony.VerificationDepthHeavy)

	docsPhase := probeGatingPhase("Write the README", "Documentation only", colony.PhaseModeMaintenance)
	if isAlwaysRequired("probe", "continue", docsPhase, state) {
		t.Error("heavy continue should not force Probe onto a documentation phase with nothing to cover")
	}

	codePhase := probeGatingPhase("Add user authentication", "Implement login endpoints and session handling", colony.PhaseModeProduction)
	if !isAlwaysRequired("probe", "continue", codePhase, state) {
		t.Error("heavy continue should still keep Probe on a phase that produces testable code")
	}
}

// TestDocumentationOnlyDetectionIsAnchored guards against the substring bug
// that has bitten caste selection before ("api" inside "cAPItalize" dispatched
// an Ambassador). Dropping a caste on a loose match is the more expensive
// direction of that mistake.
func TestDocumentationOnlyDetectionIsAnchored(t *testing.T) {
	for _, tc := range []struct {
		description string
		wantDocsGap bool
	}{
		{"Documentation for the CLI", true},
		{"Update the readme", true},
		{"Rewrite the manual", true},
		{"Update the guides", true},
		// "docs" must not be found inside another word.
		{"Configure paddocks layout", false},
		// A verb that merely starts with a document's name is not a document.
		// "manually inspect any failures" classified a production safety
		// phase as documentation-only and dropped its Probe.
		{"Run the checks and manually inspect any failures", false},
		{"Document handling is guided by policy", false},
		// An implementation word anywhere disqualifies documentation-only.
		{"Documentation plus a bug fix", false},
		{"Write the guide and refactor the parser", false},
	} {
		phase := probeGatingPhase("Phase", tc.description, colony.PhaseModeMaintenance)
		if got := queenPhaseIsDocumentationOnly(phase); got != tc.wantDocsGap {
			t.Errorf("queenPhaseIsDocumentationOnly(%q) = %v, want %v",
				tc.description, got, tc.wantDocsGap)
		}
	}
}

// TestProbeGatingReducesLightBuildWorkers is the end-to-end statement of the
// user-visible win: a light build of a documentation phase should not be
// carrying a test-coverage specialist.
func TestProbeGatingReducesLightBuildWorkers(t *testing.T) {
	phase := probeGatingPhase("Write the README", "Documentation for installation", colony.PhaseModeMaintenance)
	state := colony.ColonyState{}
	state.VerificationDepth = string(colony.VerificationDepthLight)

	budget := queenSpawnBudgetForPhase(phase, "build", state)
	for _, caste := range budget.RequiredCastes {
		if caste == "probe" {
			t.Fatalf("light documentation build still forces a Probe: %v", budget.RequiredCastes)
		}
	}
	if len(budget.RequiredCastes) > 3 {
		t.Errorf("light documentation build requires %d castes (%s); expected a small team",
			len(budget.RequiredCastes), strings.Join(budget.RequiredCastes, ", "))
	}
}
