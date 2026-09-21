package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

// episode_index_test.go proves the shared lineage (LIVE-05, LIVE-07, D-11,
// D-12): loadColonyEpisodeIndex covers all three durable record sources,
// orders them deterministically, never invents a success outcome for a
// record that has none, and never writes anything while reading.

func newEpisodeIndexTestStore(t *testing.T) (*storage.Store, string) {
	t.Helper()
	root := t.TempDir()
	dataDir := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return s, root
}

// writeEpisodeIndexSwarmFixture persists a Swarm episode fixture through
// the real production writer (persistSwarmEpisode), never a hand-typed
// JSON literal -- so this test can never drift from what the runtime
// actually produces.
func writeEpisodeIndexSwarmFixture(t *testing.T, s *storage.Store, swarmID, target string, startedAt, endedAt time.Time, interrupted bool) swarmEpisodeRecord {
	t.Helper()
	status := swarmEpisodeStatusCompleted
	stage := ""
	if interrupted {
		status = swarmEpisodeStatusInterrupted
		stage = swarmEpisodeStageFix
	}
	record := buildSwarmEpisodeRecord(swarmEpisodeBuildParams{
		SwarmID:            swarmID,
		Target:             target,
		Status:             status,
		InterruptedStage:   stage,
		StartedAt:          startedAt,
		EndedAt:            endedAt,
		Comparison:         swarmComparison{},
		VerificationStatus: "completed",
	})
	if err := persistSwarmEpisode(s, record); err != nil {
		t.Fatalf("persist swarm episode fixture %s: %v", swarmID, err)
	}
	loaded, ok := loadSwarmEpisode(s, swarmID)
	if !ok {
		t.Fatalf("could not reload persisted swarm episode fixture %s", swarmID)
	}
	return loaded
}

// writeEpisodeIndexResearchFixture writes a saved Oracle research document
// directly, in the exact front-matter shape parseOracleResearchFrontMatter
// already reads (cmd/oracle_research_doc.go) -- the same shape every real
// saved document carries.
func writeEpisodeIndexResearchFixture(t *testing.T, root, name, coreQuestion, status string, generated time.Time) {
	t.Helper()
	dir := oracleResearchDir(root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir research dir: %v", err)
	}
	doc := fmt.Sprintf(`---
title: "%s"
core_question: "%s"
generated: "%s"
status: "%s"
confidence: 80
iterations: 3
---

Body of the research document.
`, name, coreQuestion, generated.UTC().Format(time.RFC3339), status)
	path := filepath.Join(dir, name+".md")
	if err := os.WriteFile(path, []byte(doc), 0644); err != nil {
		t.Fatalf("write research fixture: %v", err)
	}
}

// writeEpisodeIndexAttemptFixture writes a buildAttemptRecord fixture
// directly through the store, in the same relative path production build
// attempts use (build/phase-N/attempts/<id>.json).
func writeEpisodeIndexAttemptFixture(t *testing.T, s *storage.Store, phase int, id, phaseName, status string, startedAt, completedAt time.Time, checkFix *checkFixAttemptRecord) buildAttemptRecord {
	t.Helper()
	record := buildAttemptRecord{
		SchemaVersion: buildAttemptSchemaVersion,
		ID:            id,
		Phase:         phase,
		PhaseName:     phaseName,
		Status:        status,
		StartedAt:     startedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:     completedAt.UTC().Format(time.RFC3339Nano),
		CheckFix:      checkFix,
	}
	if status != "" && !completedAt.IsZero() {
		record.CompletedAt = completedAt.UTC().Format(time.RFC3339Nano)
	}
	rel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase), "attempts", id+".json"))
	if err := s.SaveJSON(rel, record); err != nil {
		t.Fatalf("write attempt fixture: %v", err)
	}
	return record
}

// writeEpisodeIndexQuickFixture writes a quickAttemptRecord fixture
// directly through the store, at the same relative path persistQuickAttempt
// (cmd/command_truth.go) uses in production (quick/attempts/<id>.json).
func writeEpisodeIndexQuickFixture(t *testing.T, s *storage.Store, id, question string, verdict colony.WorkOutcome, startedAt, completedAt time.Time) quickAttemptRecord {
	t.Helper()
	record := quickAttemptRecord{
		ID:          id,
		Mode:        "job",
		Question:    question,
		StartedAt:   startedAt.UTC().Format(time.RFC3339Nano),
		CompletedAt: completedAt.UTC().Format(time.RFC3339Nano),
		Verdict:     verdict,
	}
	rel := filepath.ToSlash(filepath.Join("quick", "attempts", id+".json"))
	if err := s.SaveJSON(rel, record); err != nil {
		t.Fatalf("write quick attempt fixture: %v", err)
	}
	return record
}

func TestEpisodeIndexCoversThreeRecordSources(t *testing.T) {
	s, root := newEpisodeIndexTestStore(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	writeEpisodeIndexSwarmFixture(t, s, "swarm-episode-index-1", "Fix flaky test", now.Add(-time.Hour), now, false)
	writeEpisodeIndexResearchFixture(t, root, "index-research-1", "How should the index work?", "complete", now.Add(-2*time.Hour))
	writeEpisodeIndexAttemptFixture(t, s, 202, "attempt-index-1", "Episode index", buildAttemptBuilt, now.Add(-3*time.Hour), now.Add(-2*time.Hour), nil)
	writeEpisodeIndexQuickFixture(t, s, "quick-index-1", "rename the label", colony.WorkOutcomeSuccess, now.Add(-4*time.Hour), now.Add(-3*time.Hour))

	idx, err := loadColonyEpisodeIndex(root, s)
	if err != nil {
		t.Fatalf("loadColonyEpisodeIndex: %v", err)
	}
	if len(idx.Entries) != 4 {
		t.Fatalf("expected 4 entries, got %d: %+v", len(idx.Entries), idx.Entries)
	}
	if len(idx.Unavailable) != 0 {
		t.Fatalf("expected no unavailable sources, got %v", idx.Unavailable)
	}

	kinds := map[string]bool{}
	for _, entry := range idx.Entries {
		kinds[entry.Kind] = true
	}
	for _, want := range []string{colonyEpisodeKindSwarm, colonyEpisodeKindOracleResearch, colonyEpisodeKindBuildAttempt, colonyEpisodeKindQuick} {
		if !kinds[want] {
			t.Errorf("expected an entry of kind %q, got kinds %v", want, kinds)
		}
	}
}

func TestEpisodeIndexOrdersNewestFirstWithStableTiebreak(t *testing.T) {
	s, root := newEpisodeIndexTestStore(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	writeEpisodeIndexAttemptFixture(t, s, 1, "attempt-old", "Older phase", buildAttemptBuilt, now.Add(-10*time.Hour), now.Add(-9*time.Hour), nil)
	writeEpisodeIndexAttemptFixture(t, s, 2, "attempt-new", "Newer phase", buildAttemptBuilt, now.Add(-1*time.Hour), now, nil)
	// Two entries with the exact same recorded end time.
	writeEpisodeIndexSwarmFixture(t, s, "swarm-tie-a", "Tie A", now.Add(-4*time.Hour), now.Add(-3*time.Hour), false)
	writeEpisodeIndexSwarmFixture(t, s, "swarm-tie-b", "Tie B", now.Add(-4*time.Hour), now.Add(-3*time.Hour), false)

	idx1, err := loadColonyEpisodeIndex(root, s)
	if err != nil {
		t.Fatalf("loadColonyEpisodeIndex: %v", err)
	}
	if len(idx1.Entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(idx1.Entries))
	}
	if idx1.Entries[0].Path != "build/phase-2/attempts/attempt-new.json" {
		t.Fatalf("expected the newest entry first, got %+v", idx1.Entries[0])
	}
	if idx1.Entries[len(idx1.Entries)-1].Path != "build/phase-1/attempts/attempt-old.json" {
		t.Fatalf("expected the oldest entry last, got %+v", idx1.Entries[len(idx1.Entries)-1])
	}

	idx2, err := loadColonyEpisodeIndex(root, s)
	if err != nil {
		t.Fatalf("loadColonyEpisodeIndex (second load): %v", err)
	}
	for i := range idx1.Entries {
		if idx1.Entries[i].Path != idx2.Entries[i].Path {
			t.Fatalf("tiebreak order changed across loads at index %d: %q vs %q", i, idx1.Entries[i].Path, idx2.Entries[i].Path)
		}
	}
}

func TestEpisodeIndexMarksMissingOutcomeUnknown(t *testing.T) {
	s, root := newEpisodeIndexTestStore(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	// An attempt record with no recorded status at all.
	writeEpisodeIndexAttemptFixture(t, s, 5, "attempt-no-status", "No status recorded", "", now.Add(-time.Hour), time.Time{}, nil)

	idx, err := loadColonyEpisodeIndex(root, s)
	if err != nil {
		t.Fatalf("loadColonyEpisodeIndex: %v", err)
	}
	if len(idx.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d: %+v", len(idx.Entries), idx.Entries)
	}
	entry := idx.Entries[0]
	if entry.OutcomeKnown {
		t.Fatalf("expected OutcomeKnown = false for a record with no status, got entry %+v", entry)
	}
	if entry.Standing == workStandingVerified {
		t.Fatalf("expected an attempt with no recorded status to never be verified, got standing %q", entry.Standing)
	}
	label := workStandingLabel(entry.Standing, entry.StandingReason)
	for _, forbidden := range []string{"verified", "built"} {
		if strings.Contains(label, forbidden) {
			t.Errorf("standing label %q for an unknown-outcome entry must not contain the success word %q", label, forbidden)
		}
	}
}

func TestEpisodeIndexIsReadOnly(t *testing.T) {
	s, root := newEpisodeIndexTestStore(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	writeEpisodeIndexSwarmFixture(t, s, "swarm-readonly-1", "Read only check", now.Add(-time.Hour), now, false)
	writeEpisodeIndexResearchFixture(t, root, "readonly-research", "Does this stay read-only?", "complete", now.Add(-2*time.Hour))
	writeEpisodeIndexAttemptFixture(t, s, 9, "attempt-readonly", "Read only phase", buildAttemptBuilt, now.Add(-3*time.Hour), now.Add(-2*time.Hour), nil)

	before := dirDigest(t, s.BasePath())
	if _, err := loadColonyEpisodeIndex(root, s); err != nil {
		t.Fatalf("loadColonyEpisodeIndex: %v", err)
	}
	after := dirDigest(t, s.BasePath())
	if before != after {
		t.Fatalf("loadColonyEpisodeIndex changed the colony data directory:\nbefore: %s\nafter:  %s", before, after)
	}

	// With every source absent, the index is empty and every source is
	// named unavailable.
	emptyStore, emptyRoot := newEpisodeIndexTestStore(t)
	idx, err := loadColonyEpisodeIndex(emptyRoot, emptyStore)
	if err != nil {
		t.Fatalf("loadColonyEpisodeIndex (empty): %v", err)
	}
	if len(idx.Entries) != 0 {
		t.Fatalf("expected no entries with all sources absent, got %+v", idx.Entries)
	}
	if len(idx.Unavailable) != 4 {
		t.Fatalf("expected all four sources named unavailable, got %v", idx.Unavailable)
	}
}

// --- Task 2: the most-recent-episode section on the status dashboard -----

func loadStatusTestState(t *testing.T, s *storage.Store) colony.ColonyState {
	t.Helper()
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	return state
}

func TestStatusShowsTheMostRecentEpisode(t *testing.T) {
	saveGlobals(t)
	s, root := setupTestStore(t)
	store = s
	t.Setenv("AETHER_ROOT", root)
	state := loadStatusTestState(t, s)

	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	writeEpisodeIndexSwarmFixture(t, s, "swarm-status-shows-1", "The status section target", now.Add(-30*time.Minute), now, false)

	// Warm every path the render pass will touch -- the storage layer's own
	// first-touch lock bookkeeping is a documented exception
	// (TestStatusRunningTotalWritesNothing), not a write this task made.
	_ = renderDashboard(state, s, buildStatusResult(state, s))

	before := dirDigest(t, s.BasePath())
	result := buildStatusResult(state, s)
	visual := renderDashboard(state, s, result)
	after := dirDigest(t, s.BasePath())
	if before != after {
		t.Fatalf("rendering status changed the colony data directory:\nbefore: %s\nafter:  %s", before, after)
	}

	if !strings.Contains(visual, "Read the full write-up:") {
		t.Fatalf("expected the most-recent-episode section, got:\n%s", visual)
	}
	if !strings.Contains(visual, "The status section target") {
		t.Fatalf("expected the episode's subject, got:\n%s", visual)
	}
	if !strings.Contains(visual, ".aether/data/swarms/swarm-status-shows-1/episode.json") {
		t.Fatalf("expected the episode's write-up path, got:\n%s", visual)
	}
}

func TestStatusOmitsTheEpisodeSectionWhenThereAreNone(t *testing.T) {
	saveGlobals(t)
	s, root := setupTestStore(t)
	store = s
	t.Setenv("AETHER_ROOT", root)
	state := loadStatusTestState(t, s)

	result := buildStatusResult(state, s)
	visual := renderDashboard(state, s, result)

	if strings.Contains(visual, "Read the full write-up:") {
		t.Fatalf("expected no episode section with nothing recorded, got:\n%s", visual)
	}
}

func TestStatusEpisodeSectionDoesNotDisturbOtherSections(t *testing.T) {
	saveGlobals(t)
	s, root := setupTestStore(t)
	store = s
	t.Setenv("AETHER_ROOT", root)
	state := loadStatusTestState(t, s)

	before := renderDashboard(state, s, buildStatusResult(state, s))

	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	writeEpisodeIndexSwarmFixture(t, s, "swarm-status-section-1", "Status section fixture", now.Add(-time.Hour), now, false)

	after := renderDashboard(state, s, buildStatusResult(state, s))

	section := renderMostRecentEpisodeStatusSection(s)
	if section == "" {
		t.Fatalf("expected a non-empty episode section once a swarm episode exists")
	}
	reconstructed := strings.Replace(after, "\n"+section, "", 1)
	if reconstructed != before {
		t.Fatalf("episode section disturbed other sections.\nbefore:\n%s\n\nafter (with section removed):\n%s", before, reconstructed)
	}
}

// --- Task 3: episodes and unverified work in the history listing ---------

func newHistoryTestProjection(root string) (LifecycleFacts, LifecycleProjection) {
	facts := unavailableLifecycleFacts(root, time.Now().UTC(), "episode index test fixture")
	projection := projectLifecycle(facts, LifecycleViewFocused, "codex")
	return facts, projection
}

func TestHistoryListsEveryEpisodeWithOutcomeAndCost(t *testing.T) {
	saveGlobals(t)
	s, root := newEpisodeIndexTestStore(t)
	store = s
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	writeEpisodeIndexSwarmFixture(t, s, "swarm-history-1", "History episode target A", now.Add(-3*time.Hour), now.Add(-2*time.Hour), false)
	writeEpisodeIndexResearchFixture(t, root, "history-research-1", "History episode target B", "complete", now.Add(-4*time.Hour))
	writeEpisodeIndexAttemptFixture(t, s, 44, "attempt-history-1", "History episode target C", buildAttemptBuilt, now.Add(-time.Hour), now, nil)
	seedSpendLedgerForTest(t, 44, spendWorkflowBuild, measuredSpendRowForTest("Mason-44", "builder", 100_000))

	facts, projection := newHistoryTestProjection(root)
	before := dirDigest(t, s.BasePath())
	result := buildLifecycleHistoryProjection(facts, projection, "", 0, "", root, s)
	after := dirDigest(t, s.BasePath())
	if before != after {
		t.Fatalf("rendering history changed the colony data directory:\nbefore: %s\nafter:  %s", before, after)
	}

	var episodeRows []LifecycleHistoryRow
	for _, row := range result.Events {
		if row.Category == lifecycleHistoryCategoryEpisode {
			episodeRows = append(episodeRows, row)
		}
	}
	if len(episodeRows) != 3 {
		t.Fatalf("expected 3 episode rows, got %d: %+v", len(episodeRows), episodeRows)
	}

	wantCostBlock := renderSpendCostLineFromLedgers(func() []spendLedger {
		ledgers, _ := loadSpendLedgersForPhase(44)
		return ledgers
	}())

	var sawBuildAttemptCost bool
	for _, row := range episodeRows {
		if !strings.HasPrefix(row.Result, "Outcome:") {
			t.Errorf("episode row %q missing an outcome: %+v", row.Event, row)
		}
		if row.Path == "" {
			t.Errorf("episode row %q missing a path to its full write-up: %+v", row.Event, row)
		}
		if row.Kind == colonyEpisodeKindBuildAttempt {
			if row.Cost != wantCostBlock {
				t.Errorf("build attempt row cost = %q, want the ledger authority's own block:\n%s", row.Cost, wantCostBlock)
			}
			sawBuildAttemptCost = true
		}
	}
	if !sawBuildAttemptCost {
		t.Fatalf("expected the build attempt row to carry a cost, got episode rows: %+v", episodeRows)
	}
}

func TestHistoryListsUnverifiedWorkWithItsStanding(t *testing.T) {
	saveGlobals(t)
	s, root := newEpisodeIndexTestStore(t)
	store = s

	planDir := filepath.Join(root, ".aether", "data", "planning")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("mkdir plan research dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(planDir, "SCOUT.md"), []byte("# Scout notes\n"), 0644); err != nil {
		t.Fatalf("write plan research fixture: %v", err)
	}

	dreamsDir := filepath.Join(root, ".aether", "dreams")
	if err := os.MkdirAll(dreamsDir, 0755); err != nil {
		t.Fatalf("mkdir dreams dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dreamsDir, "note.md"), []byte("# A reflection\n"), 0644); err != nil {
		t.Fatalf("write reflection fixture: %v", err)
	}

	facts, projection := newHistoryTestProjection(root)
	result := buildLifecycleHistoryProjection(facts, projection, "", 0, "", root, s)

	var unverifiedRows []LifecycleHistoryRow
	for _, row := range result.Events {
		if row.Category == lifecycleHistoryCategoryUnverified {
			unverifiedRows = append(unverifiedRows, row)
		}
	}
	if len(unverifiedRows) != 2 {
		t.Fatalf("expected 2 unverified-work rows, got %d: %+v", len(unverifiedRows), unverifiedRows)
	}
	for _, row := range unverifiedRows {
		if !strings.Contains(row.Standing, "useful notes, not verified") {
			t.Errorf("unverified row %q standing = %q, want the shared useful-notes label", row.Event, row.Standing)
		}
		if row.Path == "" {
			t.Errorf("unverified row %q missing a path", row.Event)
		}
	}
}

func TestHistoryVerifiedAndUnverifiedAreDistinguishable(t *testing.T) {
	saveGlobals(t)
	s, root := newEpisodeIndexTestStore(t)
	store = s
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	writeEpisodeIndexSwarmFixture(t, s, "swarm-verified-1", "Verified swarm run", now.Add(-2*time.Hour), now.Add(-time.Hour), false)
	writeEpisodeIndexSwarmFixture(t, s, "swarm-unverified-1", "Unverified swarm run", now.Add(-4*time.Hour), now.Add(-3*time.Hour), true)

	facts, projection := newHistoryTestProjection(root)
	result := buildLifecycleHistoryProjection(facts, projection, "", 0, "", root, s)

	var verifiedRow, unverifiedRow *LifecycleHistoryRow
	for i := range result.Events {
		row := result.Events[i]
		switch row.Event {
		case colonyEpisodeKindLabel(colonyEpisodeKindSwarm) + ": Verified swarm run":
			verifiedRow = &result.Events[i]
		case colonyEpisodeKindLabel(colonyEpisodeKindSwarm) + ": Unverified swarm run":
			unverifiedRow = &result.Events[i]
		}
	}
	if verifiedRow == nil || unverifiedRow == nil {
		t.Fatalf("expected both a verified and an unverified swarm row, got events: %+v", result.Events)
	}
	if verifiedRow.Standing != "verified" {
		t.Errorf("verified row standing = %q, want %q", verifiedRow.Standing, "verified")
	}
	if unverifiedRow.Standing == "verified" || !strings.Contains(unverifiedRow.Standing, "useful notes") {
		t.Errorf("unverified row standing = %q, want a useful-notes label", unverifiedRow.Standing)
	}
	if verifiedRow.Standing == unverifiedRow.Standing {
		t.Fatalf("verified and unverified rows carry the identical standing field: %q", verifiedRow.Standing)
	}

	var bVerified, bUnverified strings.Builder
	writeLifecycleHistoryRow(&bVerified, *verifiedRow)
	writeLifecycleHistoryRow(&bUnverified, *unverifiedRow)
	if bVerified.String() == bUnverified.String() {
		t.Fatalf("verified and unverified rows rendered identically")
	}
}

func TestHistoryFilterNarrowsWithoutChangingRows(t *testing.T) {
	saveGlobals(t)
	s, root := newEpisodeIndexTestStore(t)
	store = s
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	writeEpisodeIndexSwarmFixture(t, s, "swarm-filter-1", "Filter target A", now.Add(-2*time.Hour), now.Add(-time.Hour), false)
	writeEpisodeIndexResearchFixture(t, root, "filter-research-1", "Filter target B", "complete", now.Add(-3*time.Hour))
	writeEpisodeIndexAttemptFixture(t, s, 55, "attempt-filter-1", "Filter target C", buildAttemptBuilt, now.Add(-4*time.Hour), now.Add(-3*time.Hour), nil)

	facts, projection := newHistoryTestProjection(root)
	unfiltered := buildLifecycleHistoryProjection(facts, projection, "", 0, "", root, s)

	unfilteredSwarmRows := map[string]LifecycleHistoryRow{}
	for _, row := range unfiltered.Events {
		if row.Kind == colonyEpisodeKindSwarm {
			unfilteredSwarmRows[row.Event] = row
		}
	}
	if len(unfilteredSwarmRows) != 1 {
		t.Fatalf("expected exactly 1 unfiltered swarm row, got %d: %+v", len(unfilteredSwarmRows), unfiltered.Events)
	}

	filtered := buildLifecycleHistoryProjection(facts, projection, "", 0, colonyEpisodeKindSwarm, root, s)
	if len(filtered.Events) != 1 {
		t.Fatalf("expected exactly 1 row once filtered to kind %q, got %d: %+v", colonyEpisodeKindSwarm, len(filtered.Events), filtered.Events)
	}
	got := filtered.Events[0]
	want, ok := unfilteredSwarmRows[got.Event]
	if !ok {
		t.Fatalf("filtered row %q was not present in the unfiltered listing", got.Event)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filtering changed the row's own content:\nunfiltered: %+v\nfiltered:   %+v", want, got)
	}
}

// TestStatusHistoryAndWatchShareOneLineage drives one fixture through the
// status dashboard, the history listing, and the replay-backed watch
// summary, and proves all three name the same most-recent episode with the
// same outcome and the same cost block -- the point of 202-14: the three
// surfaces read one lineage and cannot disagree about what ran.
func TestStatusHistoryAndWatchShareOneLineage(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	t.Setenv("AETHER_ROOT", root)

	const phase = 77
	const episodeName = "lineage-episode-202-14"
	goal := "One shared lineage fixture"
	now := time.Now().UTC()
	createTestColonyState(t, s.BasePath(), colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: phase,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: phase, Name: "Lineage phase"},
		}},
	})

	seedSpendLedgerForTest(t, phase, spendWorkflowBuild, measuredSpendRowForTest("Mason-77", "builder", 250_000))

	writeEpisodeIndexAttemptFixture(t, s, phase, "attempt-lineage-1", episodeName, buildAttemptBuilt,
		now.Add(-30*time.Minute), now, nil)

	emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: episodeName, EpisodeKind: "build", Status: "starting"})
	emitColonyLive(events.LiveTopicEpisodeEnded, events.ColonyLivePayload{EpisodeID: episodeName, EpisodeKind: "build", Status: buildAttemptBuilt})

	wantCostBlock := renderSpendCostLineFromLedgers(func() []spendLedger {
		ledgers, _ := loadSpendLedgersForPhase(phase)
		return ledgers
	}())
	if !strings.Contains(wantCostBlock, "250") && !strings.Contains(wantCostBlock, "K") && !strings.Contains(wantCostBlock, "M") {
		t.Fatalf("test fixture cost block carries no figure, cannot prove agreement: %q", wantCostBlock)
	}

	// Watch: the replay-backed summary of the most recently started live
	// episode.
	ctx := context.Background()
	watchResult := buildReplayWatchResult(ctx, root, s, now)
	watchVisual := renderReplayWatchVisual(watchResult)
	if got := stringValue(watchResult["episode_id"]); got != episodeName {
		t.Fatalf("watch replayed episode %q, want %q", got, episodeName)
	}
	if got := stringValue(watchResult["outcome"]); got != buildAttemptBuilt {
		t.Fatalf("watch outcome = %q, want %q", got, buildAttemptBuilt)
	}
	if !strings.HasSuffix(watchVisual, wantCostBlock) {
		t.Fatalf("watch's rendered cost block does not match the ledger authority's own block:\nwatch:\n%s\nwant suffix:\n%s", watchVisual, wantCostBlock)
	}

	// Status: the most-recent-episode section.
	statusSection := renderMostRecentEpisodeStatusSection(s)
	if !strings.Contains(statusSection, episodeName) {
		t.Fatalf("status section does not name the shared episode %q:\n%s", episodeName, statusSection)
	}
	if !strings.Contains(statusSection, buildAttemptBuilt) {
		t.Fatalf("status section does not name the outcome %q:\n%s", buildAttemptBuilt, statusSection)
	}
	if !strings.Contains(statusSection, wantCostBlock) {
		t.Fatalf("status section's cost block does not match the ledger authority's own block:\nstatus:\n%s\nwant:\n%s", statusSection, wantCostBlock)
	}

	// History: the episode row for the same build attempt.
	facts, projection := newHistoryTestProjection(root)
	history := buildLifecycleHistoryProjection(facts, projection, "", 0, "", root, s)
	var historyRow *LifecycleHistoryRow
	for i := range history.Events {
		if strings.Contains(history.Events[i].Event, episodeName) {
			historyRow = &history.Events[i]
			break
		}
	}
	if historyRow == nil {
		t.Fatalf("history does not list the shared episode %q: %+v", episodeName, history.Events)
	}
	if !strings.Contains(historyRow.Result, buildAttemptBuilt) {
		t.Fatalf("history row outcome = %q, want to contain %q", historyRow.Result, buildAttemptBuilt)
	}
	if historyRow.Cost != wantCostBlock {
		t.Fatalf("history row cost block does not match the ledger authority's own block:\nhistory:\n%s\nwant:\n%s", historyRow.Cost, wantCostBlock)
	}
}

// TestQuickEpisodeEntryNamesItsActor is release 1.0.85's real-run fix: a
// quick attempt's history row named "Actor: Unknown" even though the
// attempt dispatched exactly one named helper. The episode entry (and the
// history row built from it) should name that helper instead.
func TestQuickEpisodeEntryNamesItsActor(t *testing.T) {
	record := quickAttemptRecord{
		ID:         "quick-actor-test",
		Question:   "fix the typo",
		StartedAt:  time.Now().UTC().Format(time.RFC3339Nano),
		WorkerName: "Forge-4",
		Caste:      "builder",
		Verdict:    colony.WorkOutcomeSuccess,
	}
	entry := quickAttemptIndexEntryFrom(record, "quick/attempts/quick-actor-test.json")
	if entry.Actor != "Forge-4 (builder)" {
		t.Fatalf("entry.Actor = %q, want %q", entry.Actor, "Forge-4 (builder)")
	}
	row := lifecycleHistoryRowFromEpisode(entry)
	if row.Actor != "Forge-4 (builder)" {
		t.Fatalf("row.Actor = %q, want %q", row.Actor, "Forge-4 (builder)")
	}
}
