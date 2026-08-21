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
// Probe sat in queenBuildSafetyRequiredCastes unconditionally, and required
// castes bypass the worker budget entirely — so a Probe spawned on every build
// including documentation phases, discovery phases, and `--light` builds of
// trivial changes. The standard continue path required Probe too, so a single
// phase paid for two test-coverage specialists whether or not any code existed
// for them to cover. That is the "why is it spawning all these workers"
// complaint in one line of code.
func TestProbeIsRequiredOnlyWhereItCanFindSomething(t *testing.T) {
	for _, tc := range []struct {
		name       string
		phase      colony.Phase
		wantProbe  bool
		wantReason string
	}{
		{
			name:       "implementation phase gets a probe",
			phase:      probeGatingPhase("Add user authentication", "Implement login endpoints and session handling", colony.PhaseModeProduction),
			wantProbe:  true,
			wantReason: "new code needs coverage",
		},
		{
			name:       "documentation-only phase does not",
			phase:      probeGatingPhase("Write the README", "Documentation for installation and usage", colony.PhaseModeMaintenance),
			wantProbe:  false,
			wantReason: "no code produced, nothing to cover",
		},
		{
			name:       "discovery phase does not",
			phase:      probeGatingPhase("Evaluate queue options", "Research candidate message brokers", colony.PhaseModeDiscovery),
			wantProbe:  false,
			wantReason: "research produces findings, not code",
		},
		{
			name:       "docs phase that also ships code still gets a probe",
			phase:      probeGatingPhase("Document and extend the API", "Update the guide and implement the /health endpoint", colony.PhaseModeProduction),
			wantProbe:  true,
			wantReason: "mixed phase still produces testable code",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			required := stringSet(queenBuildSafetyRequiredCastes(tc.phase))
			if got := required["probe"]; got != tc.wantProbe {
				t.Errorf("build probe required = %v, want %v (%s)\nrequired castes: %v",
					got, tc.wantProbe, tc.wantReason, queenBuildSafetyRequiredCastes(tc.phase))
			}

			// Standard continue must agree with build: gating one side only
			// would still bill the phase for a Probe it does not need.
			state := colony.ColonyState{}
			gotContinue := isAlwaysRequired("probe", "continue", tc.phase, state)
			if gotContinue != tc.wantProbe {
				t.Errorf("standard continue probe required = %v, want %v (%s)",
					gotContinue, tc.wantProbe, tc.wantReason)
			}
		})
	}
}

// TestWatcherIsAlwaysRequiredOnBuild pins the line that must not move. Gating
// Probe is a cost decision; gating the Watcher would mean a build could report
// success with nothing having checked it.
func TestWatcherIsAlwaysRequiredOnBuild(t *testing.T) {
	for _, phase := range []colony.Phase{
		probeGatingPhase("Write the README", "Documentation only", colony.PhaseModeMaintenance),
		probeGatingPhase("Evaluate options", "Research", colony.PhaseModeDiscovery),
		probeGatingPhase("Ship the release", "Production deploy", colony.PhaseModeProduction),
	} {
		required := stringSet(queenBuildSafetyRequiredCastes(phase))
		if !required["watcher"] {
			t.Errorf("watcher must be required on every build; phase %q got %v",
				phase.Name, queenBuildSafetyRequiredCastes(phase))
		}
	}
}

// TestSafetyCastesSurviveProbeGating confirms the change did not leak into the
// castes that exist for safety rather than cost. A high-risk or production
// phase keeps its Auditor and Gatekeeper regardless of what Probe does.
func TestSafetyCastesSurviveProbeGating(t *testing.T) {
	phase := probeGatingPhase("Security review of auth", "Documentation of the auth model", colony.PhaseModeProduction)
	required := stringSet(queenBuildSafetyRequiredCastes(phase))
	for _, caste := range []string{"auditor", "gatekeeper", "watcher"} {
		if !required[caste] {
			t.Errorf("%s must survive on a production/security phase, got %v",
				caste, queenBuildSafetyRequiredCastes(phase))
		}
	}
}

// TestHeavyContinueKeepsProbe pins the escape hatch: asking for heavy is an
// explicit request for the full gauntlet, so Probe stays even on a phase where
// the gate would otherwise drop it.
func TestHeavyContinueKeepsProbe(t *testing.T) {
	phase := probeGatingPhase("Write the README", "Documentation only", colony.PhaseModeMaintenance)
	state := colony.ColonyState{}
	state.VerificationDepth = string(colony.VerificationDepthHeavy)

	if !isAlwaysRequired("probe", "continue", phase, state) {
		t.Error("heavy continue must keep Probe even on a documentation phase")
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

// TestGatekeeperNeedsASecuritySignal pins the second half of the spawn
// trimming. Gatekeeper is a security specialist, and it used to be required
// beside Auditor on the same condition — which included "mode is production".
// Mode is inferred from wording, so most real phases land on production, and
// "Add a CSV export" was summoning a security auditor that could only report
// it had found no security surface.
func TestGatekeeperNeedsASecuritySignal(t *testing.T) {
	for _, tc := range []struct {
		name           string
		phase          colony.Phase
		wantGatekeeper bool
		wantAuditor    bool
	}{
		{
			name:           "ordinary production work gets a quality gate but no security review",
			phase:          probeGatingPhase("Add CSV export", "Let users download their table as a CSV file", colony.PhaseModeProduction),
			wantGatekeeper: false,
			wantAuditor:    true,
		},
		{
			name:           "credentials work keeps its security review",
			phase:          probeGatingPhase("Rotate API credentials", "Move the secret token out of the config file", colony.PhaseModeProduction),
			wantGatekeeper: true,
			wantAuditor:    true,
		},
		{
			name:           "an auth phase keeps its security review",
			phase:          probeGatingPhase("Add user authentication", "Implement login and session handling", colony.PhaseModeProduction),
			wantGatekeeper: true,
			wantAuditor:    true,
		},
		{
			// A release gate is the last point at which a security problem can
			// be caught before it ships. The first version of the security
			// signal list carried the security surfaces but dropped the gate
			// terms, and this phase silently lost its Gatekeeper.
			name:           "a final sign-off keeps its security review",
			phase:          probeGatingPhase("Final review", "Complete final signoff before handoff", colony.PhaseModeProduction),
			wantGatekeeper: true,
			wantAuditor:    true,
		},
		{
			name:           "a release phase keeps its security review",
			phase:          probeGatingPhase("Cut the release", "Prepare the release candidate", colony.PhaseModeProduction),
			wantGatekeeper: true,
			wantAuditor:    true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			required := stringSet(queenBuildSafetyRequiredCastes(tc.phase))
			if got := required["gatekeeper"]; got != tc.wantGatekeeper {
				t.Errorf("gatekeeper required = %v, want %v\nrequired: %v",
					got, tc.wantGatekeeper, queenBuildSafetyRequiredCastes(tc.phase))
			}
			if got := required["auditor"]; got != tc.wantAuditor {
				t.Errorf("auditor required = %v, want %v\nrequired: %v",
					got, tc.wantAuditor, queenBuildSafetyRequiredCastes(tc.phase))
			}
		})
	}
}

// TestHighRiskPhaseKeepsBothReviewers is the floor under the change above: a
// phase classified high risk keeps its security reviewer whatever its wording,
// so trimming Gatekeeper from ordinary production work cannot cost a genuinely
// risky phase its review.
func TestHighRiskPhaseKeepsBothReviewers(t *testing.T) {
	phase := probeGatingPhase("Rework the permissions model", "Change how access is granted", colony.PhaseModeProduction)
	if phaseRiskLevel(phase) != "high" {
		t.Skipf("fixture no longer classifies as high risk (got %q); the assertion below needs a high-risk phase", phaseRiskLevel(phase))
	}
	required := stringSet(queenBuildSafetyRequiredCastes(phase))
	for _, caste := range []string{"auditor", "gatekeeper", "watcher"} {
		if !required[caste] {
			t.Errorf("high-risk phase must keep %s, got %v", caste, queenBuildSafetyRequiredCastes(phase))
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
