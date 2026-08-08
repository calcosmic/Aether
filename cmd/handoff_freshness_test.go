package cmd

import (
	"strings"
	"testing"
	"time"
)

func freshnessRecord(name, freshness string, wave int) workerHandoffRecord {
	return workerHandoffRecord{
		WorkerName: name,
		Workflow:   "build",
		Phase:      1,
		Wave:       wave,
		Caste:      "builder",
		Summary:    "did " + name + "'s work",
		Freshness:  freshness,
	}
}

// TestHandoffSectionRanksRealTimestampsAboveNotRun is the regression test for
// the sort that quietly defeated the whole relay.
//
// Freshness is normally RFC3339, but pkg/codex/handoff.go deliberately keeps
// the literal "not-run" for a worker whose verification never executed. Three
// sorts compared the field as a raw string, and "not-run" collates above every
// "2026-…" timestamp. So the five-record window a worker actually reads filled
// up with the handoffs that had nothing in them, and pruning discarded real
// ones to keep them. The mechanism built to stop workers repeating each other
// was preferentially feeding them the empty entries.
func TestHandoffSectionRanksRealTimestampsAboveNotRun(t *testing.T) {
	saveGlobals(t)
	setupBuildFlowTest(t)
	now := time.Now().UTC()
	records := []workerHandoffRecord{
		freshnessRecord("NotRun-1", "not-run", 1),
		freshnessRecord("NotRun-2", "not-run", 1),
		freshnessRecord("Real-1", now.Add(-5*time.Minute).Format(time.RFC3339), 1),
		freshnessRecord("Real-2", now.Add(-10*time.Minute).Format(time.RFC3339), 1),
		freshnessRecord("Real-3", now.Add(-15*time.Minute).Format(time.RFC3339), 1),
		freshnessRecord("Real-4", now.Add(-20*time.Minute).Format(time.RFC3339), 1),
		freshnessRecord("Real-5", now.Add(-25*time.Minute).Format(time.RFC3339), 1),
	}

	rendered := renderWorkerHandoffRecordsForTest(t, records)

	for _, want := range []string{"Real-1", "Real-2", "Real-3", "Real-4", "Real-5"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("timestamped handoff %s was pushed out of the top-5 window by an un-run record:\n%s", want, rendered)
		}
	}
	for _, unwanted := range []string{"NotRun-1", "NotRun-2"} {
		if strings.Contains(rendered, unwanted) {
			t.Errorf("un-run handoff %s occupied a window slot ahead of real work", unwanted)
		}
	}
}

// TestPruneKeepsRealHandoffsOverNotRun covers the same defect where it does
// lasting damage: pruning to the retained 100 discarded real records to keep
// un-run ones, so the history degraded permanently rather than per-render.
func TestPruneKeepsRealHandoffsOverNotRun(t *testing.T) {
	now := time.Now().UTC()
	records := []workerHandoffRecord{
		freshnessRecord("NotRun-1", "not-run", 1),
		freshnessRecord("Real-1", now.Add(-time.Minute).Format(time.RFC3339), 1),
		freshnessRecord("Real-2", now.Add(-2*time.Minute).Format(time.RFC3339), 1),
	}

	pruned := pruneWorkerHandoffRecords(records, 2)

	names := map[string]bool{}
	for _, record := range pruned {
		names[record.WorkerName] = true
	}
	if !names["Real-1"] || !names["Real-2"] {
		t.Errorf("pruning discarded real handoffs in favour of an un-run one; kept %v", names)
	}
	if names["NotRun-1"] {
		t.Errorf("pruning retained an un-run handoff over real work; kept %v", names)
	}
}

// TestHandoffSectionRendersProvenanceForEveryRecord asserts the invariant
// rather than the presence of a section: every rendered handoff must say where
// it came from and how old it is. Caste, wave and freshness were stored on
// every record and printed on none, so a reader weighted a three-day-old note
// exactly like the one from the worker beside it.
func TestHandoffSectionRendersProvenanceForEveryRecord(t *testing.T) {
	saveGlobals(t)
	setupBuildFlowTest(t)
	now := time.Now().UTC()
	records := []workerHandoffRecord{
		freshnessRecord("Alpha", now.Add(-2*time.Hour).Format(time.RFC3339), 1),
		freshnessRecord("Beta", now.Add(-3*24*time.Hour).Format(time.RFC3339), 2),
		freshnessRecord("Gamma", "not-run", 2),
	}

	rendered := renderWorkerHandoffRecordsForTest(t, records)

	headers := strings.Count(rendered, "### ")
	provenance := strings.Count(rendered, "- From: ")
	if headers == 0 {
		t.Fatalf("no handoffs rendered:\n%s", rendered)
	}
	if provenance != headers {
		t.Errorf("%d handoff header(s) but %d provenance line(s); every record must state its origin:\n%s",
			headers, provenance, rendered)
	}
	// Age must be legible, and an un-run record must say so rather than being
	// presented with the same authority as a dated one.
	if !strings.Contains(rendered, "recorded") {
		t.Errorf("provenance carries no age:\n%s", rendered)
	}
	if !strings.Contains(rendered, "verification never ran") {
		t.Errorf("an un-run handoff must disclose that its verification never ran:\n%s", rendered)
	}
}

// TestHandoffAgePhraseIsHonestAboutUnknownDates pins the fallbacks directly.
func TestHandoffAgePhraseIsHonestAboutUnknownDates(t *testing.T) {
	for _, tc := range []struct{ freshness, want string }{
		{"not-run", "verification never ran"},
		{"Evidence collected after latest edit.", "undated"},
		{"", "undated"},
	} {
		if got := handoffAgePhrase(tc.freshness); got != tc.want {
			t.Errorf("handoffAgePhrase(%q) = %q, want %q", tc.freshness, got, tc.want)
		}
	}
	recent := time.Now().UTC().Add(-90 * time.Minute).Format(time.RFC3339)
	if got := handoffAgePhrase(recent); !strings.Contains(got, "h ago") {
		t.Errorf("handoffAgePhrase(recent) = %q, want an hours-ago phrase", got)
	}
}

// renderWorkerHandoffRecordsForTest seeds the real store and calls the real
// renderer, so the ordering property is tested through the path that ships
// rather than a shortcut around it.
func renderWorkerHandoffRecordsForTest(t *testing.T, records []workerHandoffRecord) string {
	t.Helper()
	if err := store.SaveJSON(workerHandoffsPath, workerHandoffFile{Entries: records}); err != nil {
		t.Fatalf("seed handoff store: %v", err)
	}
	return renderWorkerHandoffSection("build", 1, "Requester")
}
