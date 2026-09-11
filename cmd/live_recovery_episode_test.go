package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/events"
)

// TestRecoveryDecisionKeepsTheBuildEpisodeLive proves CR-02's fix: a
// recovery decision fired while a build or check episode is open is
// recorded on that same episode (never a synthetic identifier of its
// own), so `aether watch` keeps showing the real running episode and its
// workers, with the recovery state layered on as a detail. A recovery
// decision fired with nothing open is still recorded, under its own
// phase-derived fallback identifier, so it is never silently dropped.
func TestRecoveryDecisionKeepsTheBuildEpisodeLive(t *testing.T) {
	newFailingRecoveryContext := func(phase int) RecoveryContext {
		return RecoveryContext{
			Phase:        phase,
			Wave:         1,
			WorkerName:   "Mason-1",
			TaskID:       "task-1",
			Caste:        "builder",
			Status:       "failed",
			ErrorMessage: "generic worker failure",
			Dispatches: []codex.WorkerDispatch{
				{WorkerName: "Mason-1", Caste: "builder", TaskID: "task-1"},
			},
			Budget: newRecoveryBudget(1),
		}
	}

	t.Run("build episode open: recovery is a detail on it, not a hijack", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s

		buildEpisodeID := "build-ep-recovery-test"
		emitColonyLiveEpisodeStarted(buildEpisodeID, events.EpisodeKindBuild)
		emitColonyLiveWorkerStarted(buildEpisodeID, events.EpisodeKindBuild, codex.WorkerDispatch{
			WorkerName: "Mason-1",
			Caste:      "builder",
			Wave:       1,
		})

		restore := setActiveLiveBuildEpisode(buildEpisodeID)
		defer restore()

		outcome := orchestrateRecovery(newFailingRecoveryContext(202))

		mode, snapshot := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if mode != watchModeLive {
			t.Fatalf("resolveWatchMode mode = %q, want %q -- a recovery decision must not close the running build episode", mode, watchModeLive)
		}
		if snapshot.EpisodeID != buildEpisodeID {
			t.Fatalf("resolveWatchMode named episode %q, want the still-open build episode %q -- recovery hijacked the live view", snapshot.EpisodeID, buildEpisodeID)
		}
		if len(snapshot.Workers) != 1 || snapshot.Workers[0].WorkerID != "Mason-1" {
			t.Fatalf("recovery routing dropped the build episode's worker rows: %+v", snapshot.Workers)
		}
		if snapshot.RecoveryState != outcome.Action.Type {
			t.Fatalf("snapshot.RecoveryState = %q, want the recovery outcome's own action type %q", snapshot.RecoveryState, outcome.Action.Type)
		}
	})

	t.Run("check episode open: recovery routes onto the continue carrier", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s

		checkEpisodeID := "check-ep-recovery-test"
		emitColonyLiveEpisodeStarted(checkEpisodeID, events.EpisodeKindContinue)
		emitColonyLiveWorkerStarted(checkEpisodeID, events.EpisodeKindContinue, codex.WorkerDispatch{
			WorkerName: "Watcher-1",
			Caste:      "watcher",
			Wave:       1,
		})

		restore := setActiveLiveContinueEpisode(checkEpisodeID)
		defer restore()

		outcome := orchestrateRecovery(newFailingRecoveryContext(202))

		mode, snapshot := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if mode != watchModeLive {
			t.Fatalf("resolveWatchMode mode = %q, want %q -- a recovery decision must not close the running check episode", mode, watchModeLive)
		}
		if snapshot.EpisodeID != checkEpisodeID {
			t.Fatalf("resolveWatchMode named episode %q, want the still-open check episode %q -- recovery hijacked the live view", snapshot.EpisodeID, checkEpisodeID)
		}
		if len(snapshot.Workers) != 1 || snapshot.Workers[0].WorkerID != "Watcher-1" {
			t.Fatalf("recovery routing dropped the check episode's worker rows: %+v", snapshot.Workers)
		}
		if snapshot.RecoveryState != outcome.Action.Type {
			t.Fatalf("snapshot.RecoveryState = %q, want the recovery outcome's own action type %q", snapshot.RecoveryState, outcome.Action.Type)
		}
	})

	t.Run("no live episode open: recovery is still recorded under its own fallback episode", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s

		ctx := newFailingRecoveryContext(202)
		wantEpisodeID, wantEpisodeKind := currentLiveRecoveryEpisode(ctx.Phase)
		if wantEpisodeKind != events.EpisodeKindRecovery {
			t.Fatalf("fixture setup broken: currentLiveRecoveryEpisode with no carrier set returned kind %q, want %q", wantEpisodeKind, events.EpisodeKindRecovery)
		}

		outcome := orchestrateRecovery(ctx)

		raw := readColonyLiveEventsRaw(s, time.Time{})
		decoded := decodeColonyLiveEvents(raw)
		found := false
		for _, entry := range decoded {
			if entry.event.Topic != events.LiveTopicRecoveryChanged {
				continue
			}
			if entry.payload.EpisodeID != wantEpisodeID {
				t.Fatalf("recorded recovery event episode ID = %q, want the phase-derived fallback %q", entry.payload.EpisodeID, wantEpisodeID)
			}
			if entry.payload.EpisodeKind != events.EpisodeKindRecovery {
				t.Fatalf("recorded recovery event episode kind = %q, want %q", entry.payload.EpisodeKind, events.EpisodeKindRecovery)
			}
			if entry.payload.RecoveryState != outcome.Action.Type {
				t.Fatalf("recorded recovery event RecoveryState = %q, want the outcome's own action type %q", entry.payload.RecoveryState, outcome.Action.Type)
			}
			found = true
		}
		if !found {
			t.Fatalf("no live.recovery.changed event was recorded for a standalone recovery decision -- it must never be silently dropped")
		}
	})

	t.Run("while the build carrier is set, exactly one distinct episode ID is recorded", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s

		buildEpisodeID := "single-episode-recovery-test"
		emitColonyLiveEpisodeStarted(buildEpisodeID, events.EpisodeKindBuild)
		emitColonyLiveWorkerStarted(buildEpisodeID, events.EpisodeKindBuild, codex.WorkerDispatch{
			WorkerName: "Mason-1",
			Caste:      "builder",
			Wave:       1,
		})

		restore := setActiveLiveBuildEpisode(buildEpisodeID)
		defer restore()

		orchestrateRecovery(newFailingRecoveryContext(202))

		raw := readColonyLiveEventsRaw(s, time.Time{})
		decoded := decodeColonyLiveEvents(raw)
		seen := map[string]bool{}
		for _, entry := range decoded {
			if entry.payload.EpisodeID != "" {
				seen[entry.payload.EpisodeID] = true
			}
		}
		if len(seen) != 1 {
			t.Fatalf("expected exactly one distinct non-empty episode ID while the build carrier is set, got %d: %v", len(seen), seen)
		}
		if !seen[buildEpisodeID] {
			t.Fatalf("the one recorded episode ID is %v, want it to be the build episode %q", seen, buildEpisodeID)
		}
	})
}
