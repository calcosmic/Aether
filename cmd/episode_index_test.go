package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
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

func TestEpisodeIndexCoversThreeRecordSources(t *testing.T) {
	s, root := newEpisodeIndexTestStore(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	writeEpisodeIndexSwarmFixture(t, s, "swarm-episode-index-1", "Fix flaky test", now.Add(-time.Hour), now, false)
	writeEpisodeIndexResearchFixture(t, root, "index-research-1", "How should the index work?", "complete", now.Add(-2*time.Hour))
	writeEpisodeIndexAttemptFixture(t, s, 202, "attempt-index-1", "Episode index", buildAttemptBuilt, now.Add(-3*time.Hour), now.Add(-2*time.Hour), nil)

	idx, err := loadColonyEpisodeIndex(root, s)
	if err != nil {
		t.Fatalf("loadColonyEpisodeIndex: %v", err)
	}
	if len(idx.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d: %+v", len(idx.Entries), idx.Entries)
	}
	if len(idx.Unavailable) != 0 {
		t.Fatalf("expected no unavailable sources, got %v", idx.Unavailable)
	}

	kinds := map[string]bool{}
	for _, entry := range idx.Entries {
		kinds[entry.Kind] = true
	}
	for _, want := range []string{colonyEpisodeKindSwarm, colonyEpisodeKindOracleResearch, colonyEpisodeKindBuildAttempt} {
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

	// With all three sources absent, the index is empty and every source is
	// named unavailable.
	emptyStore, emptyRoot := newEpisodeIndexTestStore(t)
	idx, err := loadColonyEpisodeIndex(emptyRoot, emptyStore)
	if err != nil {
		t.Fatalf("loadColonyEpisodeIndex (empty): %v", err)
	}
	if len(idx.Entries) != 0 {
		t.Fatalf("expected no entries with all sources absent, got %+v", idx.Entries)
	}
	if len(idx.Unavailable) != 3 {
		t.Fatalf("expected all three sources named unavailable, got %v", idx.Unavailable)
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

	result := buildStatusResult(state, s)
	visual := renderDashboard(state, s, result)

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
