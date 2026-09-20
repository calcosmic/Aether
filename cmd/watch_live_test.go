package cmd

import (
	"context"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

// TestSwarmInvestigationWaveReachesTheLiveWatchScreen drives the real Swarm
// investigation wave -- buildSwarmInvestigationPlans + executeSwarmWave, the
// exact production call shape cmd/swarm_cmd.go's runSwarmDestroy uses --
// against an isolated fixture root with a stub worker invoker, then proves
// two things end to end:
//
//  1. While the wave is still open (its wave-ended event has not yet been
//     emitted), resolveWatchMode resolves to the live branch and
//     renderLiveWatchVisual names the dispatched worker(s), their caste, and
//     the wave they were dispatched on.
//  2. The dashboard is a PURE projection of replayed events: a completely
//     fresh store handle -- reading nothing but the persisted event file --
//     reproduces the exact same snapshot field for field.
func TestSwarmInvestigationWaveReachesTheLiveWatchScreen(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	target := "Auth panic when session is missing"
	swarmID := "swarm-live-e2e-test"
	if err := initializeSwarmRun(swarmID); err != nil {
		t.Fatalf("initialize swarm run: %v", err)
	}

	investigation := buildSwarmInvestigationPlans(root, target)
	if len(investigation) == 0 {
		t.Fatalf("fixture is broken: buildSwarmInvestigationPlans returned no plans for %q -- this test proves nothing without a real dispatch to observe", target)
	}
	wave := swarmPlansWaveNumber(investigation)

	ctx := context.Background()
	invoker := &swarmTestInvoker{}

	emitColonyLive(events.LiveTopicWaveStarted, events.ColonyLivePayload{
		EpisodeID:   swarmID,
		EpisodeKind: "swarm",
		Wave:        wave,
		Status:      "starting",
	})

	runs, err := executeSwarmWave(ctx, root, swarmID, target, investigation, "", invoker, true)
	if err != nil {
		t.Fatalf("executeSwarmWave: %v", err)
	}
	if len(runs) != len(investigation) {
		t.Fatalf("executeSwarmWave returned %d runs, want %d (one per dispatched plan)", len(runs), len(investigation))
	}

	// The wave is still open at this point: wave-ended has not been
	// emitted yet.
	mode, liveSnapshot := resolveWatchMode(ctx, store, time.Now().UTC())
	if mode != watchModeLive {
		t.Fatalf("resolveWatchMode = %q while the investigation wave is still open, want %q", mode, watchModeLive)
	}
	if !liveSnapshot.Open {
		t.Fatalf("live snapshot Open = false while the investigation wave is still open")
	}
	if len(liveSnapshot.Workers) != len(investigation) {
		t.Fatalf("live snapshot has %d workers, want %d (one per dispatched investigation plan)", len(liveSnapshot.Workers), len(investigation))
	}

	rendered := renderLiveWatchVisual(liveSnapshot)
	for _, plan := range investigation {
		if !strings.Contains(rendered, plan.Name) {
			t.Errorf("rendered live watch output missing dispatched worker name %q:\n%s", plan.Name, rendered)
		}
		if !strings.Contains(rendered, casteLabel(plan.Caste)) {
			t.Errorf("rendered live watch output missing caste label for %q:\n%s", plan.Caste, rendered)
		}
	}
	if !strings.Contains(rendered, "Wave "+strconv.Itoa(wave)) {
		t.Errorf("rendered live watch output does not name wave %d:\n%s", wave, rendered)
	}

	// Close the wave, then prove the projection is a pure replay: a
	// completely fresh store handle, reading nothing but the persisted
	// event file, must reproduce the same snapshot field for field as the
	// live path itself now reports (wave closed).
	emitColonyLive(events.LiveTopicWaveEnded, events.ColonyLivePayload{
		EpisodeID:   swarmID,
		EpisodeKind: "swarm",
		Wave:        wave,
		Status:      "completed",
	})
	_, liveSnapshotClosed := resolveWatchMode(ctx, store, time.Now().UTC())
	if liveSnapshotClosed.Open {
		t.Fatalf("live snapshot Open = true after the wave-ended event was emitted")
	}

	freshStore, err := storage.NewStore(store.BasePath())
	if err != nil {
		t.Fatalf("open a fresh store handle on the same persisted directory: %v", err)
	}
	replaySnapshot, err := replayColonyLiveSnapshot(ctx, freshStore, swarmID, time.Time{})
	if err != nil {
		t.Fatalf("replayColonyLiveSnapshot: %v", err)
	}

	if !reflect.DeepEqual(replaySnapshot, liveSnapshotClosed) {
		t.Fatalf("replay-only snapshot does not equal the live snapshot field for field:\n replay: %+v\n live:   %+v", replaySnapshot, liveSnapshotClosed)
	}
}

// TestLiveProjectionIsPureReplay proves the reducer's own purity contract
// in isolation from the Swarm wiring: a nil store never writes, an absent
// persisted file replays as an empty snapshot with no error, and two events
// sharing an identical (second-precision) timestamp replay in a stable,
// sequence-number-ordered position across repeated replays of the same
// file.
func TestLiveProjectionIsPureReplay(t *testing.T) {
	t.Run("nil store never writes", func(t *testing.T) {
		saveGlobals(t)
		store = nil
		// The absence of a panic or any observable side effect IS the
		// proof here -- emitColonyLive has a void signature by design
		// (emission failure never changes the outcome of the work it
		// describes), so there is nothing else to assert against.
		emitColonyLive(events.LiveTopicWorkerStarted, events.ColonyLivePayload{EpisodeID: "e1"})
	})

	t.Run("absent file replays as empty snapshot, no error", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		snapshot, err := replayColonyLiveSnapshot(context.Background(), s, "does-not-exist", time.Time{})
		if err != nil {
			t.Fatalf("replayColonyLiveSnapshot over an absent file returned an error: %v", err)
		}
		if len(snapshot.Workers) != 0 {
			t.Fatalf("expected zero active workers from an absent persisted file, got %d", len(snapshot.Workers))
		}
		if snapshot.Open {
			t.Fatalf("expected an absent-file snapshot to report Open=false, got true")
		}
	})

	t.Run("equal timestamps order by sequence, repeated replay is deterministic", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s

		episodeID := "swarm-tiebreak-test"
		// Both events are emitted back to back -- events.FormatTimestamp is
		// second-precision, so these two very likely share an identical
		// persisted timestamp. Sequence number is what breaks the tie.
		emitColonyLive(events.LiveTopicWorkerStarted, events.ColonyLivePayload{
			EpisodeID: episodeID, Caste: "scout", WorkerID: "Scout-1", WorkerName: "Scout-1", Wave: 1,
		})
		emitColonyLive(events.LiveTopicWorkerStarted, events.ColonyLivePayload{
			EpisodeID: episodeID, Caste: "tracker", WorkerID: "Tracker-1", WorkerName: "Tracker-1", Wave: 1,
		})

		first, err := replayColonyLiveSnapshot(context.Background(), s, episodeID, time.Time{})
		if err != nil {
			t.Fatalf("replayColonyLiveSnapshot: %v", err)
		}
		second, err := replayColonyLiveSnapshot(context.Background(), s, episodeID, time.Time{})
		if err != nil {
			t.Fatalf("replayColonyLiveSnapshot: %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("replaying the same fixture file twice produced different snapshots:\n first:  %+v\n second: %+v", first, second)
		}
		if len(first.Workers) != 2 {
			t.Fatalf("expected 2 workers, got %d: %+v", len(first.Workers), first.Workers)
		}
		if first.Workers[0].WorkerID != "Scout-1" || first.Workers[1].WorkerID != "Tracker-1" {
			t.Fatalf("expected workers ordered [Scout-1, Tracker-1] by emission sequence, got [%s, %s]", first.Workers[0].WorkerID, first.Workers[1].WorkerID)
		}

		if renderLiveWatchVisual(first) != renderLiveWatchVisual(second) {
			t.Fatalf("rendered output differs between two replays of the same persisted fixture file")
		}
	})
}

// TestWatchFollowsTheMostRecentlyStartedOpenEpisode proves resolveWatchMode
// prefers the older, still-open episode over a newer episode that has
// already closed -- CR-02's core regression. Before this task,
// latestLiveEpisodeID picked whichever episode owned the chronologically
// newest single event, so once the newer episode closed (its own last
// event), the cockpit would follow it into replay instead of staying on
// the still-running older episode and its workers.
func TestWatchFollowsTheMostRecentlyStartedOpenEpisode(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	emitColonyLiveEpisodeStarted("older-open-ep", events.EpisodeKindBuild)
	emitColonyLiveWorkerStarted("older-open-ep", events.EpisodeKindBuild, codex.WorkerDispatch{
		WorkerName: "Mason-1",
		Caste:      "builder",
		Wave:       1,
	})

	// A genuine timestamp gap so the newer episode's own start event is
	// provably later than the older one's, not merely later in file order.
	time.Sleep(1100 * time.Millisecond)

	emitColonyLiveEpisodeStarted("newer-closed-ep", events.EpisodeKindContinue)
	emitColonyLiveEpisodeEnded("newer-closed-ep", events.EpisodeKindContinue, "completed")

	mode, snapshot := resolveWatchMode(context.Background(), s, time.Now().UTC())
	if mode != watchModeLive {
		t.Fatalf("resolveWatchMode mode = %q, want %q -- the older episode is still open", mode, watchModeLive)
	}
	if snapshot.EpisodeID != "older-open-ep" {
		t.Fatalf("resolveWatchMode named episode %q, want the still-open older episode %q -- a newer closed episode must never hijack the live view", snapshot.EpisodeID, "older-open-ep")
	}
	if len(snapshot.Workers) != 1 || snapshot.Workers[0].WorkerID != "Mason-1" {
		t.Fatalf("resolveWatchMode's snapshot dropped the still-open episode's worker rows: %+v", snapshot.Workers)
	}
}

// TestClosedEpisodesPickTheSameEpisodeTheReplaySummaryNames proves that
// with nothing open at all, resolveWatchMode's snapshot names the exact
// same episode mostRecentlyStartedLiveEpisode (the replay summary's own
// selection) does -- the live and replay branches can never disagree about
// "what just ran" (D-03).
func TestClosedEpisodesPickTheSameEpisodeTheReplaySummaryNames(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	emitColonyLiveEpisodeStarted("closed-ep-1", events.EpisodeKindBuild)
	emitColonyLiveEpisodeEnded("closed-ep-1", events.EpisodeKindBuild, "completed")

	time.Sleep(1100 * time.Millisecond)

	emitColonyLiveEpisodeStarted("closed-ep-2", events.EpisodeKindContinue)
	emitColonyLiveEpisodeEnded("closed-ep-2", events.EpisodeKindContinue, "completed")

	mode, snapshot := resolveWatchMode(context.Background(), s, time.Now().UTC())
	if mode != watchModeReplay {
		t.Fatalf("fixture setup broken: mode = %q, want %q (both episodes are closed)", mode, watchModeReplay)
	}

	want := mostRecentlyStartedLiveEpisode(s)
	if want == "" {
		t.Fatalf("fixture setup broken: mostRecentlyStartedLiveEpisode returned no episode")
	}
	if snapshot.EpisodeID != want {
		t.Fatalf("resolveWatchMode named episode %q, but the replay summary names %q -- with nothing open, live and replay must agree", snapshot.EpisodeID, want)
	}
}

// TestOneOpenBalanceRule drives one recorded event sequence through both
// foldColonyLiveEvents (via replayColonyLiveSnapshot) and
// openColonyLiveEpisodeIDs and fails if the two disagree about whether the
// episode is open -- proving the reducer and the selector share one open/
// close rule. It also asserts colonyLiveBoundaryDelta itself returns a
// non-zero delta for each of the four boundary topics and zero for a
// worker topic.
func TestOneOpenBalanceRule(t *testing.T) {
	for _, topic := range []string{events.LiveTopicEpisodeStarted, events.LiveTopicWaveStarted} {
		if d := colonyLiveBoundaryDelta(topic); d <= 0 {
			t.Errorf("colonyLiveBoundaryDelta(%q) = %d, want a positive delta", topic, d)
		}
	}
	for _, topic := range []string{events.LiveTopicEpisodeEnded, events.LiveTopicWaveEnded} {
		if d := colonyLiveBoundaryDelta(topic); d >= 0 {
			t.Errorf("colonyLiveBoundaryDelta(%q) = %d, want a negative delta", topic, d)
		}
	}
	if d := colonyLiveBoundaryDelta(events.LiveTopicWorkerStarted); d != 0 {
		t.Errorf("colonyLiveBoundaryDelta(%q) = %d, want 0 for a non-boundary topic", events.LiveTopicWorkerStarted, d)
	}

	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	episodeID := "balance-ep"
	emitColonyLiveEpisodeStarted(episodeID, events.EpisodeKindBuild)
	emitColonyLiveWorkerStarted(episodeID, events.EpisodeKindBuild, codex.WorkerDispatch{
		WorkerName: "Mason-1",
		Caste:      "builder",
		Wave:       1,
	})

	decodedOpen := func() []colonyLiveDecodedEvent {
		raw := readColonyLiveEventsRaw(s, time.Time{})
		decoded := decodeColonyLiveEvents(raw)
		sortColonyLiveEntries(decoded)
		return decoded
	}

	decoded := decodedOpen()
	snapshot, err := replayColonyLiveSnapshot(context.Background(), s, episodeID, time.Time{})
	if err != nil {
		t.Fatalf("replayColonyLiveSnapshot: %v", err)
	}
	openSet := openColonyLiveEpisodeIDs(decoded)
	if snapshot.Open != openSet[episodeID] {
		t.Fatalf("after starting: foldColonyLiveEvents reports Open=%v but openColonyLiveEpisodeIDs reports open=%v for the same recorded sequence", snapshot.Open, openSet[episodeID])
	}
	if !snapshot.Open {
		t.Fatalf("fixture setup broken: episode should be open (started, never ended)")
	}

	emitColonyLiveEpisodeEnded(episodeID, events.EpisodeKindBuild, "completed")

	decoded = decodedOpen()
	snapshot, err = replayColonyLiveSnapshot(context.Background(), s, episodeID, time.Time{})
	if err != nil {
		t.Fatalf("replayColonyLiveSnapshot: %v", err)
	}
	openSet = openColonyLiveEpisodeIDs(decoded)
	if snapshot.Open != openSet[episodeID] {
		t.Fatalf("after ending: foldColonyLiveEvents reports Open=%v but openColonyLiveEpisodeIDs reports open=%v for the same recorded sequence", snapshot.Open, openSet[episodeID])
	}
	if snapshot.Open {
		t.Fatalf("episode should be closed after episode.ended, both rules must agree it is closed")
	}
}

// oracleDeadControllerPID starts a short-lived process, waits for it to
// exit and be reaped, and returns its PID -- a controller PID the test
// itself proved is gone, never a typed-in "looks dead" literal (CLAUDE.md's
// fixture rule). Mirrors compatibility_cmds_test.go's
// TestOracleCompatibilityStopKillsControllerProcessTree, which derives a
// real, definitely-ALIVE PID the same way, from an actual process rather
// than a hand-picked number.
func oracleDeadControllerPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		if isPermissionDeniedForTest(err) {
			t.Skipf("process fixture unavailable in this sandbox: %v", err)
		}
		t.Fatalf("run throwaway process: %v", err)
	}
	pid := cmd.Process.Pid
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !oracleProcessExists(pid) {
			return pid
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("process %d did not become dead in time", pid)
	return pid
}

// TestAbandonedOracleRoundIsNotLive proves oracleLiveEpisodeAbandoned (via
// resolveWatchMode's colonyLiveEpisodeAbandoned dispatch) judges an Oracle
// episode by Oracle's own durable state and controller process, never by
// another lane's spawn-run record (202-17, CR-01's second half).
func TestAbandonedOracleRoundIsNotLive(t *testing.T) {
	t.Run("active oracle round stays live despite a terminated spawn run from an earlier build", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		writeSpawnRunFixture(t, s.BasePath(), "earlier-build-run", "completed")

		paths := oracleWorkspacePaths(root)
		if err := ensureOracleWorkspace(paths); err != nil {
			t.Fatalf("ensure oracle workspace: %v", err)
		}
		state := oracleStateFile{
			StartedAt:     "2026-01-01T00:00:00Z",
			Status:        "active",
			Phase:         "investigate",
			Iteration:     1,
			MaxIterations: 10,
		}
		if err := writeOracleStateFile(paths.StatePath, state); err != nil {
			t.Fatalf("write oracle state: %v", err)
		}
		openOracleLiveEpisode(state)
		emitOracleLiveRound(state)

		mode, _ := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if mode != watchModeLive {
			t.Fatalf("mode = %q, want %q -- an active Oracle round must not be demoted by a finished earlier build's spawn run", mode, watchModeLive)
		}
	})

	t.Run("oracle round whose controller process is dead resolves replay", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s

		paths := oracleWorkspacePaths(root)
		if err := ensureOracleWorkspace(paths); err != nil {
			t.Fatalf("ensure oracle workspace: %v", err)
		}
		deadPID := oracleDeadControllerPID(t)
		state := oracleStateFile{
			StartedAt:     "2026-01-01T00:00:00Z",
			Status:        "active",
			Phase:         "investigate",
			Iteration:     1,
			MaxIterations: 10,
			ControllerPID: deadPID,
		}
		if err := writeOracleStateFile(paths.StatePath, state); err != nil {
			t.Fatalf("write oracle state: %v", err)
		}
		openOracleLiveEpisode(state)
		emitOracleLiveRound(state)

		mode, _ := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if mode != watchModeReplay {
			t.Fatalf("mode = %q, want %q -- a round whose controller process is gone must not be shown as live", mode, watchModeReplay)
		}
	})

	t.Run("durable state naming a different run than the open episode resolves replay", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s

		paths := oracleWorkspacePaths(root)
		if err := ensureOracleWorkspace(paths); err != nil {
			t.Fatalf("ensure oracle workspace: %v", err)
		}
		firstRun := oracleStateFile{
			StartedAt:     "2026-01-01T00:00:00Z",
			Status:        "active",
			Phase:         "investigate",
			Iteration:     1,
			MaxIterations: 10,
		}
		episodeID, _ := openOracleLiveEpisode(firstRun)
		emitOracleLiveRound(firstRun)

		// The durable state has moved on to a second run -- e.g. a fresh
		// `aether oracle "new topic"` overwrote state.json -- without the
		// first episode's own close ever being emitted.
		secondRun := firstRun
		secondRun.StartedAt = "2026-02-01T00:00:00Z"
		if err := writeOracleStateFile(paths.StatePath, secondRun); err != nil {
			t.Fatalf("write second-run oracle state: %v", err)
		}
		if oracleLiveEpisodeID(secondRun) == episodeID {
			t.Fatalf("fixture is broken: second run's episode ID must differ from the first's")
		}

		mode, snapshot := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if snapshot.EpisodeID != episodeID {
			t.Fatalf("fixture setup broken: resolveWatchMode picked episode %q, want the first run's %q", snapshot.EpisodeID, episodeID)
		}
		if mode != watchModeReplay {
			t.Fatalf("mode = %q, want %q -- the first run's episode is abandoned once durable state names a different run", mode, watchModeReplay)
		}
	})

	t.Run("no readable oracle state resolves live -- absent evidence is not evidence of termination", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s

		state := oracleStateFile{StartedAt: "2026-01-01T00:00:00Z"}
		openOracleLiveEpisode(state)
		emitOracleLiveRound(state)

		mode, _ := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if mode != watchModeLive {
			t.Fatalf("mode = %q, want %q -- an unreadable/absent oracle state must not be treated as termination", mode, watchModeLive)
		}
	})

	t.Run("non-oracle episode kind keeps the existing spawn-run rule", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLiveEpisodeStarted("build-ep", events.EpisodeKindBuild)
		writeSpawnRunFixture(t, s.BasePath(), "run-1", "failed")

		mode, _ := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if mode != watchModeReplay {
			t.Fatalf("mode = %q, want %q -- a build episode's own spawn-run rule must be unchanged", mode, watchModeReplay)
		}
	})

	t.Run("resolving writes nothing to the colony data directory", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		writeSpawnRunFixture(t, s.BasePath(), "earlier-build-run", "completed")

		paths := oracleWorkspacePaths(root)
		if err := ensureOracleWorkspace(paths); err != nil {
			t.Fatalf("ensure oracle workspace: %v", err)
		}
		state := oracleStateFile{
			StartedAt:     "2026-01-01T00:00:00Z",
			Status:        "active",
			Phase:         "investigate",
			Iteration:     1,
			MaxIterations: 10,
		}
		if err := writeOracleStateFile(paths.StatePath, state); err != nil {
			t.Fatalf("write oracle state: %v", err)
		}
		openOracleLiveEpisode(state)
		emitOracleLiveRound(state)

		before := dirDigest(t, s.BasePath())
		resolveWatchMode(context.Background(), s, time.Now().UTC())
		after := dirDigest(t, s.BasePath())
		if before != after {
			t.Fatalf("resolveWatchMode wrote to the colony data directory: before=%s after=%s", before, after)
		}
	})
}
