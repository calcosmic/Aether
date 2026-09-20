package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestSealProbeRequiresTestableCode closes the gap the build and continue
// flows already closed: Probe was unconditionally required at standard seal
// depth, so a documentation-only or research-only final phase still paid for
// a test-coverage specialist with no code to cover — and required castes
// bypass the worker budget, so no depth short of light could remove it.
func TestSealProbeRequiresTestableCode(t *testing.T) {
	docPhase := probeGatingPhase("Write the README", "Documentation for installation and usage", colony.PhaseModeMaintenance)
	codePhase := probeGatingPhase("Add user authentication", "Implement login endpoints and session handling", colony.PhaseModeProduction)
	standard := colony.ColonyState{}
	heavy := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)}

	if HasCaste(queenOrchestrate(docPhase, "seal", standard), "probe") {
		t.Fatalf("standard seal of a documentation-only phase must not require a Probe: there is no code for it to cover")
	}
	if !HasCaste(queenOrchestrate(codePhase, "seal", standard), "probe") {
		t.Fatalf("standard seal of a code-producing phase must keep its Probe")
	}
	if !HasCaste(queenOrchestrate(docPhase, "seal", heavy), "probe") {
		t.Fatalf("heavy seal is an explicit request for the full gauntlet; Probe stays even on a doc phase")
	}
	if !HasCaste(queenOrchestrate(docPhase, "seal", standard), "auditor") {
		t.Fatalf("gating Probe must not disturb the seal Auditor")
	}
}
