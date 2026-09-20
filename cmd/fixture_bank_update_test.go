package cmd

import (
	"flag"
	"sort"
	"strings"
	"testing"
)

// updateBank is the fixture bank's counterpart to -update-golden. The seeded
// bank (fixtureBankPath) is derived mechanically from this project's own
// confirmed-incident sources -- the failure log, the audit findings, the
// defect register's "fixed" rows and the phase verification reports -- and
// TestSeededBankIsReproducible fails the moment any of those sources moves
// (a WINDOWS.md entry closing is enough). Nothing regenerated the committed
// file until now; it was seeded by hand once, so every later source change
// left the bank stale and the reproducibility check red with no repair path.
var updateBank = flag.Bool("update-bank", false, "regenerate cmd/testdata/fixture-bank/v1/bank.json from its confirmed-incident sources, re-attaching every existing guard by fixture id")

// TestSeededBankUpdate rewrites the committed bank from its sources when
// -update-bank is passed, mirroring TestRegressionSnapshotUpdate. The
// mechanical conversion owns every field except guard (see the comment in
// TestSeededBankIsReproducible), so the manually-curated guard layer is
// carried over by fixture id -- ids are content-derived and stable -- and a
// guard whose fixture no longer exists in the regenerated bank is reported,
// never silently dropped. A fixture new to the bank arrives unguarded and is
// counted by seedBankUnguardedFloor's two-sided ratchet
// (TestSeedBankUnguardedFloorIsTheRealCount) until a real guard is named.
func TestSeededBankUpdate(t *testing.T) {
	if !*updateBank {
		t.Skip("skipping bank regeneration; run with -update-bank to refresh")
	}

	repoRoot := findTestModuleRoot(t)
	failureLog, audit, defects, phaseReports := realConfirmedIncidents(t, repoRoot)

	var all []confirmedIncident
	all = append(all, failureLog.Incidents...)
	all = append(all, audit.Incidents...)
	all = append(all, defects.Incidents...)
	all = append(all, phaseReports.Incidents...)

	regenerated, summary, err := convertConfirmedIncidentsToFixtures(all)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	committed, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load committed bank: %v", err)
	}
	guards := make(map[string]regressionFixtureGuard, len(committed.Fixtures))
	for _, fixture := range committed.Fixtures {
		if fixture.Guard != nil {
			guards[fixture.ID] = *fixture.Guard
		}
	}

	carried := 0
	present := make(map[string]bool, len(regenerated.Fixtures))
	var added []string
	for i := range regenerated.Fixtures {
		id := regenerated.Fixtures[i].ID
		present[id] = true
		if guard, ok := guards[id]; ok {
			guardCopy := guard
			regenerated.Fixtures[i].Guard = &guardCopy
			carried++
		}
		if !committedHasFixture(committed, id) {
			added = append(added, id)
		}
	}
	var orphanedGuards []string
	for id := range guards {
		if !present[id] {
			orphanedGuards = append(orphanedGuards, id)
		}
	}
	sort.Strings(orphanedGuards)
	sort.Strings(added)

	if err := writeFixtureBank(regenerated); err != nil {
		t.Fatalf("write regenerated bank: %v", err)
	}

	t.Logf("regenerated %s: %d fixtures (%d new: %s), %d guards carried over, conversion summary %+v",
		fixtureBankPath, len(regenerated.Fixtures), len(added), strings.Join(added, " "), carried, summary)
	if len(orphanedGuards) > 0 {
		t.Errorf("%d guard(s) named fixtures that no longer exist after regeneration and were dropped -- re-attach them to the successor fixture or record why they are gone: %s",
			len(orphanedGuards), strings.Join(orphanedGuards, " "))
	}
}

func committedHasFixture(bank regressionFixtureBank, id string) bool {
	for _, fixture := range bank.Fixtures {
		if fixture.ID == id {
			return true
		}
	}
	return false
}
