package cmd

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

// --- fixtures -------------------------------------------------------------

// newColonyLiveDashboardFixtureState is a minimal ColonyState for dashboard
// header rendering -- the fields renderColonyLiveDashboardHeader reads and
// nothing else.
func newColonyLiveDashboardFixtureState(name string, phase int, state colony.State) colony.ColonyState {
	return colony.ColonyState{ColonyName: &name, CurrentPhase: phase, State: state}
}

// --- Task 1: renderColonyLiveDashboard -------------------------------------

// TestLiveDashboardShowsCurrentWaveInDepth drives a real two-wave episode
// through the actual emission boundary (emitColonyLive), replays it via
// resolveWatchMode, and proves the dashboard renders the current (later)
// wave's workers in depth while the earlier wave is compressed to one
// counted line -- plus the caste-glyph-is-shared-not-copied proof.
func TestLiveDashboardShowsCurrentWaveInDepth(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	episodeID := "dashboard-two-wave-test"
	emitColonyLive(events.LiveTopicWaveStarted, events.ColonyLivePayload{EpisodeID: episodeID, EpisodeKind: "swarm", Wave: 1})
	emitColonyLive(events.LiveTopicWorkerStarted, events.ColonyLivePayload{
		EpisodeID: episodeID, EpisodeKind: "swarm", Wave: 1,
		WorkerID: "Scout-1", WorkerName: "Scout-1", Caste: "scout", Workspace: "wave1-workspace",
	})
	emitColonyLive(events.LiveTopicWorkerFinished, events.ColonyLivePayload{
		EpisodeID: episodeID, EpisodeKind: "swarm", Wave: 1,
		WorkerID: "Scout-1", WorkerName: "Scout-1", Caste: "scout", Status: "completed",
	})
	emitColonyLive(events.LiveTopicWaveEnded, events.ColonyLivePayload{EpisodeID: episodeID, EpisodeKind: "swarm", Wave: 1, Status: "completed"})

	emitColonyLive(events.LiveTopicWaveStarted, events.ColonyLivePayload{EpisodeID: episodeID, EpisodeKind: "swarm", Wave: 2})
	emitColonyLive(events.LiveTopicWorkerStarted, events.ColonyLivePayload{
		EpisodeID: episodeID, EpisodeKind: "swarm", Wave: 2,
		WorkerID: "Tracker-1", WorkerName: "Tracker-1", Caste: "tracker",
		Workspace: "wave2-workspace", Question: "why does the panic happen",
	})
	emitColonyLive(events.LiveTopicFindingRecorded, events.ColonyLivePayload{
		EpisodeID: episodeID, EpisodeKind: "swarm", Wave: 2,
		WorkerID: "Tracker-1", WorkerName: "Tracker-1", Findings: []string{"the session pointer is nil before login"},
	})

	ctx := context.Background()
	mode, snapshot := resolveWatchMode(ctx, s, time.Now().UTC())
	if mode != watchModeLive {
		t.Fatalf("resolveWatchMode = %q, want %q (wave 2 is still open)", mode, watchModeLive)
	}
	if snapshot.Wave != 2 {
		t.Fatalf("snapshot.Wave = %d, want 2 (the current/latest wave)", snapshot.Wave)
	}

	state := newColonyLiveDashboardFixtureState("Aether", 202, colony.StateEXECUTING)
	rendered := stripANSI(renderColonyLiveDashboard(snapshot, state, nil, colonyLiveDrillSelector{}))

	for _, want := range []string{"Tracker-1", casteLabel("tracker"), "wave2-workspace", "why does the panic happen", "the session pointer is nil before login"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered dashboard missing current-wave detail %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "wave1-workspace") {
		t.Errorf("rendered dashboard leaked the earlier wave's workspace detail -- it should be compressed to a counted line:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Wave 1: 1 worker") {
		t.Errorf("rendered dashboard missing the earlier wave's counted line:\n%s", rendered)
	}

	// Changing one entry in the shared caste emoji map changes the
	// dashboard output -- proving the glyphs are read from the shared
	// map, not copied locally into this file.
	original := casteEmojiMap["tracker"]
	t.Cleanup(func() { casteEmojiMap["tracker"] = original })
	casteEmojiMap["tracker"] = "🧪"
	mutated := stripANSI(renderColonyLiveDashboard(snapshot, state, nil, colonyLiveDrillSelector{}))
	if mutated == rendered {
		t.Fatalf("mutating casteEmojiMap did not change the rendered dashboard -- the caste glyph must be read from the shared map, not a local copy")
	}
	if !strings.Contains(mutated, "🧪") {
		t.Fatalf("rendered dashboard does not reflect the mutated caste emoji map entry:\n%s", mutated)
	}
}

// TestLiveDashboardHeaderIsOneCompactLine proves the header is exactly one
// line and names project, phase, standing and episode.
func TestLiveDashboardHeaderIsOneCompactLine(t *testing.T) {
	state := newColonyLiveDashboardFixtureState("Aether", 42, colony.StateEXECUTING)
	snapshot := colonyLiveSnapshot{EpisodeID: "ep-99", EpisodeKind: "build"}

	header := renderColonyLiveDashboardHeader(snapshot, state)
	if got := strings.Count(strings.TrimRight(header, "\n"), "\n"); got != 0 {
		t.Fatalf("header carries %d embedded newlines beyond its own trailing one, want a single line:\n%q", got, header)
	}
	for _, want := range []string{"Aether", "42", string(colony.StateEXECUTING), "ep-99", "build"} {
		if !strings.Contains(header, want) {
			t.Errorf("header missing %q:\n%q", want, header)
		}
	}
}

// TestLiveDashboardOmitsFieldsTheSnapshotLacks proves an absent field is
// omitted entirely rather than rendered as a placeholder.
func TestLiveDashboardOmitsFieldsTheSnapshotLacks(t *testing.T) {
	state := newColonyLiveDashboardFixtureState("Aether", 1, colony.StateEXECUTING)
	snapshot := colonyLiveSnapshot{
		EpisodeID: "ep-1", EpisodeKind: "swarm", Wave: 1,
		Workers: []colonyLiveWorkerRow{{WorkerID: "Scout-1", WorkerName: "Scout-1", Caste: "scout", Wave: 1, Status: "active"}},
		// Confidence/TargetConfidence, Contradictions, Signals, RecoveryState
		// are all deliberately absent.
	}

	rendered := renderColonyLiveDashboard(snapshot, state, nil, colonyLiveDrillSelector{})
	for _, absent := range []string{"Confidence:", "Contradictions:", "Signals consulted:", "Recovery state:"} {
		if strings.Contains(rendered, absent) {
			t.Errorf("rendered dashboard carries a %q row for a field the snapshot does not have:\n%s", absent, rendered)
		}
	}
}

// TestLiveDashboardRendersDeterministically proves two renders of the exact
// same inputs are byte-identical.
func TestLiveDashboardRendersDeterministically(t *testing.T) {
	state := newColonyLiveDashboardFixtureState("Aether", 7, colony.StateEXECUTING)
	snapshot := colonyLiveSnapshot{
		EpisodeID: "ep-det", EpisodeKind: "build", Wave: 1, StartedAt: "2026-01-01T00:00:00Z", ElapsedSeconds: 12,
		Workers:       []colonyLiveWorkerRow{{WorkerID: "Mason-1", WorkerName: "Mason-1", Caste: "builder", Wave: 1, Status: "active", StartedAt: "2026-01-01T00:00:00Z"}},
		LastTimestamp: "2026-01-01T00:00:30Z",
	}
	ledger := spendLedger{Phase: 7, Workflow: spendWorkflowBuild, Rows: []spendRow{{AgentName: "Mason-1", Caste: "builder", Status: "completed"}}}

	first := renderColonyLiveDashboard(snapshot, state, []spendLedger{ledger}, colonyLiveDrillSelector{})
	second := renderColonyLiveDashboard(snapshot, state, []spendLedger{ledger}, colonyLiveDrillSelector{})
	if first != second {
		t.Fatalf("rendering the same snapshot twice produced different output:\n first:  %q\n second: %q", first, second)
	}
}

// TestLiveDashboardCostComesFromTheLedger proves the cost block is the
// last block in the rendered output and equals renderSpendCostLineFromLedgers
// for the exact same ledgers.
func TestLiveDashboardCostComesFromTheLedger(t *testing.T) {
	state := newColonyLiveDashboardFixtureState("Aether", 5, colony.StateEXECUTING)
	snapshot := colonyLiveSnapshot{EpisodeID: "ep-cost", EpisodeKind: "build", Wave: 1}
	ledgers := []spendLedger{{
		Phase: 5, Workflow: spendWorkflowBuild,
		Rows: []spendRow{{AgentName: "Mason-67", Caste: "builder", Status: "completed", Usage: codex.WorkerUsage{InputTokens: 1000, Source: codex.UsageSourceProvider}}},
	}}

	rendered := renderColonyLiveDashboard(snapshot, state, ledgers, colonyLiveDrillSelector{})
	costBlock := renderSpendCostLineFromLedgers(ledgers)
	if !strings.HasSuffix(rendered, costBlock) {
		t.Fatalf("rendered dashboard does not end with the exact cost block:\n rendered: %q\n costBlock: %q", rendered, costBlock)
	}
}

// TestLiveDashboardDrillDown proves drill-down by worker identity renders
// that worker's full detail and leaves the header line unchanged.
func TestLiveDashboardDrillDown(t *testing.T) {
	state := newColonyLiveDashboardFixtureState("Aether", 3, colony.StateEXECUTING)
	snapshot := colonyLiveSnapshot{
		EpisodeID: "ep-drill", EpisodeKind: "swarm", Wave: 2,
		Workers: []colonyLiveWorkerRow{
			{WorkerID: "Scout-1", WorkerName: "Scout-1", Caste: "scout", Wave: 1, Workspace: "scout-workspace", Status: "completed", Finished: true},
			{WorkerID: "Tracker-1", WorkerName: "Tracker-1", Caste: "tracker", Wave: 2, Workspace: "tracker-workspace", Status: "active"},
		},
	}

	headerOnly := renderColonyLiveDashboardHeader(snapshot, state)
	rendered := renderColonyLiveDashboard(snapshot, state, nil, colonyLiveDrillSelector{WorkerID: "Scout-1"})
	if !strings.HasPrefix(rendered[strings.Index(rendered, headerOnly):], headerOnly) {
		t.Fatalf("drill-down changed the header line:\n want prefix: %q\n got:  %q", headerOnly, rendered)
	}
	if !strings.Contains(rendered, "scout-workspace") {
		t.Errorf("drill-down by worker identity did not render that worker's own detail:\n%s", rendered)
	}
	if strings.Contains(rendered, "tracker-workspace") {
		t.Errorf("drill-down by worker identity leaked the non-selected worker's detail:\n%s", rendered)
	}
}

// --- Task 2: renderColonyLiveTicker -----------------------------------------

// TestLiveTickerIsBoundedAndNewestLast proves the ticker renders exactly
// colonyLiveTickerLimit lines, ending with the newest, when more events
// than the bound were emitted -- and exactly as many lines as it has when
// fewer were emitted.
func TestLiveTickerIsBoundedAndNewestLast(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	episodeID := "ticker-bound-test"
	total := colonyLiveTickerLimit + 5
	for i := 0; i < total; i++ {
		emitColonyLive(events.LiveTopicWorkerProgress, events.ColonyLivePayload{
			EpisodeID: episodeID, EpisodeKind: "swarm", WorkerID: fmt.Sprintf("Worker-%d", i), WorkerName: fmt.Sprintf("Worker-%d", i),
		})
	}

	_, snapshot := resolveWatchMode(context.Background(), s, time.Now().UTC())
	if len(snapshot.Ticker) != colonyLiveTickerLimit {
		t.Fatalf("snapshot.Ticker has %d entries, want exactly the bound %d", len(snapshot.Ticker), colonyLiveTickerLimit)
	}
	last := snapshot.Ticker[len(snapshot.Ticker)-1]
	if last.WorkerID != fmt.Sprintf("Worker-%d", total-1) {
		t.Fatalf("last ticker entry is %q, want the most recently emitted worker %q", last.WorkerID, fmt.Sprintf("Worker-%d", total-1))
	}

	rendered := renderColonyLiveTicker(snapshot.Ticker)
	lineCount := strings.Count(rendered, "\n") - 1 // subtract the stage-marker line
	if lineCount != colonyLiveTickerLimit {
		t.Fatalf("rendered ticker has %d lines, want exactly the bound %d:\n%s", lineCount, colonyLiveTickerLimit, rendered)
	}

	// Fewer events than the bound: exactly that many lines, no filler.
	saveGlobals(t)
	s2, _ := newTestStore(t)
	store = s2
	episodeID2 := "ticker-under-bound-test"
	for i := 0; i < 3; i++ {
		emitColonyLive(events.LiveTopicWorkerProgress, events.ColonyLivePayload{
			EpisodeID: episodeID2, EpisodeKind: "swarm", WorkerID: fmt.Sprintf("Worker-%d", i), WorkerName: fmt.Sprintf("Worker-%d", i),
		})
	}
	_, snapshot2 := resolveWatchMode(context.Background(), s2, time.Now().UTC())
	rendered2 := renderColonyLiveTicker(snapshot2.Ticker)
	lineCount2 := strings.Count(rendered2, "\n") - 1
	if lineCount2 != 3 {
		t.Fatalf("rendered ticker with 3 emitted events has %d lines, want exactly 3, no filler:\n%s", lineCount2, rendered2)
	}
}

// TestLiveTickerAndDashboardShareOneReplay proves rendering the dashboard
// (which includes the ticker) performs no additional read of the
// persisted live-event file beyond the single replay that already
// produced the snapshot passed in.
func TestLiveTickerAndDashboardShareOneReplay(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	episodeID := "ticker-one-replay-test"
	emitColonyLive(events.LiveTopicWorkerStarted, events.ColonyLivePayload{EpisodeID: episodeID, EpisodeKind: "swarm", WorkerID: "Scout-1", WorkerName: "Scout-1", Caste: "scout"})

	_, snapshot := resolveWatchMode(context.Background(), s, time.Now().UTC())
	state := newColonyLiveDashboardFixtureState("Aether", 1, colony.StateEXECUTING)

	before := colonyLiveRawReadCalls
	_ = renderColonyLiveDashboard(snapshot, state, nil, colonyLiveDrillSelector{})
	after := colonyLiveRawReadCalls
	if after != before {
		t.Fatalf("renderColonyLiveDashboard performed %d additional read(s) of the persisted live-event file; it must render from the already-replayed snapshot alone", after-before)
	}
}

// TestLiveTickerLinesArePlainEnglish proves every declared live.* topic's
// rendered ticker description is ordinary English, never the raw dotted
// topic string.
func TestLiveTickerLinesArePlainEnglish(t *testing.T) {
	for _, topic := range events.ColonyLiveTopics() {
		entry := colonyLiveTickerEntry{Topic: topic, Timestamp: "2026-01-01T00:00:00Z", WorkerID: "Scout-1", Caste: "scout", Status: "completed"}
		description := colonyLiveTickerDescription(entry)
		if description == "" {
			t.Errorf("topic %q has no rendered description", topic)
		}
		if strings.Contains(description, "live.") {
			t.Errorf("topic %q rendered %q -- the raw repository topic vocabulary leaked into owner-facing text", topic, description)
		}
		line := renderColonyLiveTickerLine(entry)
		if !strings.Contains(line, entry.Timestamp) {
			t.Errorf("rendered ticker line for %q missing its timestamp:\n%q", topic, line)
		}
	}
}
