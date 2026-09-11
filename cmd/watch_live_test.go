package cmd

import (
	"context"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

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
