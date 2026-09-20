package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestContinueLightDepthDoesNotRequireAnything pins D-13: light continue
// requires NOTHING unconditionally (194-05 removed watcher's unconditional
// membership, the last place the pre-D11 review floor survived at this
// depth). Watcher still appears on THIS fixture, but only because its own
// keyword ("test") matches "Test Phase" and clears the spawn threshold on
// relevance score alone -- a coincidence of this fixture's name, not a
// floor. TestContinueRequiredSetByDepth (cmd/owner_dials_test.go) is the
// test that actually pins isAlwaysRequired's return value at every depth.
func TestContinueLightDepthDoesNotRequireAnything(t *testing.T) {
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}
	phase := colony.Phase{ID: 1, Name: "Test Phase"}

	dispatches := queenCandidateDispatches(phase, "continue", state)
	if !HasCaste(dispatches, "watcher") {
		t.Error("watcher should be selected via its own keyword match on this fixture, not via an always-required floor")
	}

	// Builder/weaver/tracker should be suppressed for continue
	if HasCaste(dispatches, "builder") {
		t.Error("builder should not spawn for continue")
	}
	if HasCaste(dispatches, "weaver") {
		t.Error("weaver should not spawn for continue")
	}
	if HasCaste(dispatches, "tracker") {
		t.Error("tracker should not spawn for continue")
	}
}

// TestContinueStandardDepthDoesNotRequireProbeOrReview pins D-13: standard
// continue -- like light -- requires nothing unconditionally. Probe used to
// be forced onto every continue at standard depth regardless of whether the
// phase produced anything worth covering; this fixture ("Test Phase") has no
// coverage-specific keyword hit, so probe's own relevance score (15) sits
// below the spawn threshold (30) and it is not selected at all.
func TestContinueStandardDepthDoesNotRequireProbeOrReview(t *testing.T) {
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}
	phase := colony.Phase{ID: 1, Name: "Test Phase"}

	dispatches := queenCandidateDispatches(phase, "continue", state)
	if !HasCaste(dispatches, "watcher") {
		t.Error("watcher should be selected via its own keyword match on this fixture, not via an always-required floor")
	}
	if HasCaste(dispatches, "probe") {
		t.Error("standard continue should not force probe without a keyword hit or a named risk signal (D-13)")
	}

	// Gatekeeper/auditor should NOT be always-required at standard
	if HasCaste(dispatches, "gatekeeper") {
		t.Error("gatekeeper should not be always-required for standard continue")
	}
	if HasCaste(dispatches, "auditor") {
		t.Error("auditor should not be always-required for standard continue")
	}
}

// TestContinueHeavyDepthSpawnsFullReview pins D-13's other half: heavy is
// the owner's explicit ask for the full review panel -- gatekeeper, auditor,
// and probe where the phase produces testable code. Watcher is NOT part of
// that unconditional set any more (it survives on this fixture only via its
// own keyword match, same as the light/standard cases above).
func TestContinueHeavyDepthSpawnsFullReview(t *testing.T) {
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)}
	phase := colony.Phase{ID: 1, Name: "Test Phase"}

	dispatches := queenCandidateDispatches(phase, "continue", state)
	if !HasCaste(dispatches, "watcher") {
		t.Error("watcher should be selected via its own keyword match on this fixture, not via an always-required floor")
	}
	if !HasCaste(dispatches, "gatekeeper") {
		t.Error("heavy continue should always require gatekeeper")
	}
	if !HasCaste(dispatches, "auditor") {
		t.Error("heavy continue should always require auditor")
	}
	if !HasCaste(dispatches, "probe") {
		t.Error("heavy continue should always require probe")
	}
}

func TestContinueSpawnBudgetMatchesDepth(t *testing.T) {
	phase := colony.Phase{ID: 1, Name: "Test Phase"}

	lightBudget, _ := queenMaxWorkersForBudget(phase, "continue", colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}, "low")
	if lightBudget != 3 {
		t.Errorf("light continue budget should be 3, got %d", lightBudget)
	}

	standardBudget, _ := queenMaxWorkersForBudget(phase, "continue", colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}, "low")
	if standardBudget != 4 {
		t.Errorf("standard continue budget should be 4, got %d", standardBudget)
	}

	heavyBudget, _ := queenMaxWorkersForBudget(phase, "continue", colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)}, "low")
	if heavyBudget != 6 {
		t.Errorf("heavy continue budget should be 6, got %d", heavyBudget)
	}
}

func TestContinueSuppressedCastes(t *testing.T) {
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}
	phase := colony.Phase{ID: 1, Name: "Test Phase"}

	dispatches := queenCandidateDispatches(phase, "continue", state)
	for _, d := range dispatches {
		switch d.Caste {
		case "builder", "weaver", "tracker", "archaeologist", "ambassador":
			t.Errorf("%s should be suppressed for continue", d.Caste)
		}
	}
}
