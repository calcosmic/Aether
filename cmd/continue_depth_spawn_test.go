package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestContinueLightDepthSpawnsWatcherOnly(t *testing.T) {
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}
	phase := colony.Phase{ID: 1, Name: "Test Phase"}

	dispatches := queenCandidateDispatches(phase, "continue", state)
	if !HasCaste(dispatches, "watcher") {
		t.Error("light continue should always require watcher")
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

func TestContinueStandardDepthSpawnsWatcherAndProbe(t *testing.T) {
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}
	phase := colony.Phase{ID: 1, Name: "Test Phase"}

	dispatches := queenCandidateDispatches(phase, "continue", state)
	if !HasCaste(dispatches, "watcher") {
		t.Error("standard continue should always require watcher")
	}
	if !HasCaste(dispatches, "probe") {
		t.Error("standard continue should always require probe")
	}

	// Gatekeeper/auditor should NOT be always-required at standard
	if HasCaste(dispatches, "gatekeeper") {
		t.Error("gatekeeper should not be always-required for standard continue")
	}
	if HasCaste(dispatches, "auditor") {
		t.Error("auditor should not be always-required for standard continue")
	}
}

func TestContinueHeavyDepthSpawnsFullReview(t *testing.T) {
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)}
	phase := colony.Phase{ID: 1, Name: "Test Phase"}

	dispatches := queenCandidateDispatches(phase, "continue", state)
	if !HasCaste(dispatches, "watcher") {
		t.Error("heavy continue should always require watcher")
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
