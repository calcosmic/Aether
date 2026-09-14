package cmd

// LEARN-05 (204-07-PLAN.md): indexes every confirmed-failure fixture in the
// bank to the check that guards it. A fixture either names a guard test
// that exists in the real repository, or is counted on the unguarded list
// -- and that list may only shrink.

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestEveryFixtureNamesItsGuardOrIsCountedUnguarded loads the real bank,
// resolves each named guard against the real repository by parsing test
// function declarations, and fails naming any fixture whose guard does not
// resolve. It then compares the unguarded count against
// seedBankUnguardedFloor and fails if the count exceeds it.
func TestEveryFixtureNamesItsGuardOrIsCountedUnguarded(t *testing.T) {
	repoRoot := findTestModuleRoot(t)
	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load fixture bank: %v", err)
	}
	if len(bank.Fixtures) == 0 {
		t.Fatal("fixture bank is empty")
	}

	locations, err := evalGateTestFunctionLocations(repoRoot, []string{"cmd", "pkg"})
	if err != nil {
		t.Fatalf("index real test function locations: %v", err)
	}

	for _, f := range bank.Fixtures {
		if f.Guard == nil {
			continue
		}
		key := f.Guard.Package + "::" + f.Guard.Test
		if !locations[key] {
			t.Errorf("fixture %s names guard %s (package %s), which does not resolve to a real test function -- give it a real guard or remove the guard field", f.ID, f.Guard.Test, f.Guard.Package)
		}
	}

	if err := assertSeedBankUnguardedWithinFloor(bank, seedBankUnguardedFloor); err != nil {
		t.Error(err)
	}
}

// TestSeedBankIndexTotalsAgree asserts guarded plus unguarded equals the
// bank's own fixture count.
func TestSeedBankIndexTotalsAgree(t *testing.T) {
	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load fixture bank: %v", err)
	}
	index := buildSeedBankGuardIndex(bank, nil)
	if index.Guarded+index.Unguarded != index.Total {
		t.Fatalf("guarded (%d) + unguarded (%d) = %d, want total %d", index.Guarded, index.Unguarded, index.Guarded+index.Unguarded, index.Total)
	}
	if index.Total != len(bank.Fixtures) {
		t.Fatalf("index total %d does not equal bank fixture count %d", index.Total, len(bank.Fixtures))
	}
}

// TestHoldoutFixturesAreNotListedInTheVisibleIndex asserts that a fixture
// whose digest is in the holdout file does not appear in the index's
// readable listing, while still being counted in the total.
func TestHoldoutFixturesAreNotListedInTheVisibleIndex(t *testing.T) {
	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load fixture bank: %v", err)
	}
	holdoutFixtures := resolveEvalGateHoldouts(bank)
	if len(holdoutFixtures) == 0 {
		t.Fatal("no holdout fixtures resolved against the real bank -- cannot exercise the exclusion")
	}
	holdoutIDs := make(map[string]bool, len(holdoutFixtures))
	for _, f := range holdoutFixtures {
		holdoutIDs[f.ID] = true
	}

	index := buildSeedBankGuardIndex(bank, holdoutIDs)
	if index.Total != len(bank.Fixtures) {
		t.Fatalf("index total %d does not equal bank fixture count %d -- holdout exclusion must not affect the total", index.Total, len(bank.Fixtures))
	}
	for _, entry := range index.Visible {
		if holdoutIDs[entry.FixtureID] {
			t.Errorf("holdout fixture %s appears in the visible index listing", entry.FixtureID)
		}
	}
	if len(index.Visible) != index.Total-len(holdoutIDs) {
		t.Fatalf("visible listing has %d entries, want %d (total %d minus %d holdouts)", len(index.Visible), index.Total-len(holdoutIDs), index.Total, len(holdoutIDs))
	}
}

// TestSeedBankRatchetDetectsAnAddedUnguardedFixture proves the ratchet is
// not vacuous: add a synthetic unguarded fixture to a COPIED bank (never
// mutating the committed bank), run the check against the copy, and assert
// it fails naming the new fixture.
func TestSeedBankRatchetDetectsAnAddedUnguardedFixture(t *testing.T) {
	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load fixture bank: %v", err)
	}

	// Deep-copy via JSON round-trip so mutating the copy can never reach
	// the committed bank's own backing arrays/slices.
	raw, err := json.Marshal(bank)
	if err != nil {
		t.Fatalf("marshal bank for copy: %v", err)
	}
	var copyBank regressionFixtureBank
	if err := json.Unmarshal(raw, &copyBank); err != nil {
		t.Fatalf("unmarshal bank copy: %v", err)
	}

	const syntheticID = "fixture-synthetic-ratchet-test-only"
	copyBank.Fixtures = append(copyBank.Fixtures, regressionFixture{
		ID:    syntheticID,
		Title: "[broad-invariant] synthetic fixture for TestSeedBankRatchetDetectsAnAddedUnguardedFixture only",
		FailingBaseline: regressionFixtureFailingBaseline{
			Description: "synthetic, never a real confirmed incident",
			Reference:   "test-only",
		},
		Invariant:        "test-only synthetic invariant",
		AllowedMutations: []fixtureAllowedMutation{fixtureMutationNone},
		Provenance: []regressionFixtureProvenance{
			{Kind: fixtureProvenanceFailureLog, Identifier: "test-only"},
		},
		Privacy:       fixturePrivacyRepoLocal,
		Severity:      fixtureSeverityLow,
		ContentDigest: strings.Repeat("0", 64),
		// Deliberately no Guard -- this is the unguarded fixture the
		// ratchet must catch.
	})

	// Push the floor below whatever the copy's real unguarded count plus
	// the synthetic addition would be, so the ratchet is guaranteed to
	// trip regardless of the committed bank's current unguarded count --
	// this test's own guarantee must not depend on that number.
	unguardedBefore := 0
	for _, f := range bank.Fixtures {
		if f.Guard == nil {
			unguardedBefore++
		}
	}

	err = assertSeedBankUnguardedWithinFloor(copyBank, unguardedBefore)
	if err == nil {
		t.Fatal("expected the ratchet to fail against a bank with one more unguarded fixture than the floor allows")
	}
	if !strings.Contains(err.Error(), syntheticID) {
		t.Fatalf("ratchet error does not name the synthetic fixture: %v", err)
	}

	// Confirm the committed bank itself was never mutated by this test.
	reloaded, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("reload committed fixture bank: %v", err)
	}
	for _, f := range reloaded.Fixtures {
		if f.ID == syntheticID {
			t.Fatal("the committed fixture bank was mutated by this test -- the synthetic fixture leaked into it")
		}
	}
	if len(reloaded.Fixtures) != len(bank.Fixtures) {
		t.Fatalf("committed bank fixture count changed: before=%d after=%d", len(bank.Fixtures), len(reloaded.Fixtures))
	}
}
