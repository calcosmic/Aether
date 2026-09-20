package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// A colony sealed by a pre-SealOutcome runtime (state COMPLETED, milestone
// Crowned Anthill, CROWNED-ANTHILL.md written, but no seal outcome record in
// COLONY_STATE.json) must have a working recovery path on the new runtime:
// the entomb refusal names the re-seal repair, and a direct-owner forced
// re-seal writes a genuine verified SealOutcome that entomb then accepts.
// (2026-09-12: downstream CosmicDashboard colony hit exactly this wall.)

func legacySealedState() colony.ColonyState {
	facts := sealOutcome199Facts()
	state := facts.State.Value
	state.State = colony.StateCOMPLETED
	state.Milestone = "Crowned Anthill"
	state.SealOutcome = nil
	return state
}

func TestEntombRefusalNamesTheLegacyReSealRepair(t *testing.T) {
	state := legacySealedState()
	_, err := prepareEntombPreflight(entombTransactionInput{}, state)
	if err == nil {
		t.Fatal("entomb accepted a colony with no seal outcome record")
	}
	if !strings.Contains(err.Error(), "aether seal --force") {
		t.Fatalf("the refusal must name the re-seal repair for legacy-sealed colonies, got: %v", err)
	}
}

func TestLegacySealedColonyReSealsAndEntombs(t *testing.T) {
	fixture := newSealTransaction199Fixture(t, true)

	// Reshape the fixture into the legacy-sealed state: completed + crowned,
	// no SealOutcome (the old runtime never wrote one).
	fixture.Input.State.State = colony.StateCOMPLETED
	fixture.Input.State.Milestone = "Crowned Anthill"
	fixture.Input.State.SealOutcome = nil
	fixture.Input.Facts.State.Value = fixture.Input.State
	if err := fixture.Store.SaveJSON("COLONY_STATE.json", fixture.Input.State); err != nil {
		t.Fatal(err)
	}

	// The direct-owner forced re-seal must build an eligible preflight on the
	// already-completed state...
	preflight, err := BuildSealPreflight(fixture.Input.Facts, SealPreflightRequest{
		Caller: SealCallerDirectOwner, Force: true,
		Reason: "legacy colony sealed by an older runtime; re-sealing to write the verifiable record",
	})
	if err != nil {
		t.Fatalf("forced re-seal preflight refused a legacy-sealed colony: %v", err)
	}
	fixture.Input.Preflight = preflight

	// ...and the transaction must commit a genuine verified SealOutcome.
	result, err := CommitSealTransaction(fixture.Input)
	if err != nil {
		t.Fatalf("forced re-seal transaction failed on a legacy-sealed colony: %v", err)
	}
	if result.Outcome.Transaction.Stage != colony.TransactionStageVerified {
		t.Fatalf("re-seal outcome transaction stage = %q, want verified", result.Outcome.Transaction.Stage)
	}

	// The re-sealed state must now clear the entomb preflight wall.
	// (HANDOFF.md exists in any real colony; the fixture must supply it.)
	writeLifecycleFixtureFile(t, filepath.Join(fixture.Root, ".aether", "HANDOFF.md"), "# Handoff\n")
	resealed := fixture.Input.State
	resealed.SealOutcome = &result.Outcome
	if _, err := prepareEntombPreflight(entombTransactionInput{Root: fixture.Root, DataRoot: fixture.DataRoot}, resealed); err != nil {
		t.Fatalf("entomb preflight still refuses after a verified re-seal: %v", err)
	}
}
