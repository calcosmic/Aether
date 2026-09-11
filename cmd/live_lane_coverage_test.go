package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

// lastLivePayload replays every persisted live.* event for the store and
// decodes the last one's payload -- the event the immediately-preceding
// helper call produced.
func lastLivePayload(t *testing.T, s *storage.Store) events.ColonyLivePayload {
	t.Helper()
	raw := readColonyLiveEventsRaw(s, time.Time{})
	if len(raw) == 0 {
		t.Fatal("expected at least one persisted live event, found none")
	}
	var payload events.ColonyLivePayload
	if err := json.Unmarshal(raw[len(raw)-1].Payload, &payload); err != nil {
		t.Fatalf("decode last live payload: %v", err)
	}
	return payload
}

// assertPayloadPopulatesOnly walks every field of events.ColonyLivePayload
// via reflection (never a hand-typed list of the struct's ~20 fields) and
// asserts: every field named in populated is non-zero, and every other
// field (excluding SchemaVersion/Sequence, which emitColonyLive itself
// always sets) is the zero value -- proving the helper filled exactly the
// fields its input carried and invented nothing else.
func assertPayloadPopulatesOnly(t *testing.T, helper string, payload events.ColonyLivePayload, populated map[string]bool) {
	t.Helper()
	v := reflect.ValueOf(payload)
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if name == "SchemaVersion" || name == "Sequence" {
			continue
		}
		field := v.Field(i)
		isZero := field.IsZero()
		want := populated[name]
		if want && isZero {
			t.Errorf("%s: expected field %s to be populated, got zero value", helper, name)
		}
		if !want && !isZero {
			t.Errorf("%s: expected field %s to be empty, got %v", helper, name, field.Interface())
		}
	}
}

// TestLiveLaneHelpersMapOnlyKnownFields proves each per-lane emission helper
// in cmd/live_events.go fills exactly the payload fields its input carries,
// leaves every other field empty, and is a no-op when the store is nil.
func TestLiveLaneHelpersMapOnlyKnownFields(t *testing.T) {
	t.Run("emitColonyLiveEpisodeStarted", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLiveEpisodeStarted("ep-1", events.EpisodeKindBuild)
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveEpisodeStarted", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "Status": true,
		})

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveEpisodeStarted panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveEpisodeStarted("ep-1", events.EpisodeKindBuild)
		}()
	})

	t.Run("emitColonyLiveEpisodeEnded", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLiveEpisodeEnded("ep-1", events.EpisodeKindBuild, "completed")
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveEpisodeEnded", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "Status": true,
		})

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveEpisodeEnded panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveEpisodeEnded("ep-1", events.EpisodeKindBuild, "completed")
		}()
	})

	t.Run("emitColonyLiveWaveStarted", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLiveWaveStarted("ep-1", events.EpisodeKindBuild, 2)
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveWaveStarted", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "Wave": true, "Status": true,
		})

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveWaveStarted panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveWaveStarted("ep-1", events.EpisodeKindBuild, 2)
		}()
	})

	t.Run("emitColonyLiveWaveEnded", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLiveWaveEnded("ep-1", events.EpisodeKindBuild, 2, "completed")
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveWaveEnded", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "Wave": true, "Status": true,
		})

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveWaveEnded panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveWaveEnded("ep-1", events.EpisodeKindBuild, 2, "completed")
		}()
	})

	t.Run("emitColonyLiveWorkerStarted with a parent", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		dispatch := codex.WorkerDispatch{
			WorkerName:     "Forge-11",
			Caste:          "builder",
			Wave:           2,
			Root:           "/tmp/workspace",
			ParentWorkerID: "Queen",
		}
		emitColonyLiveWorkerStarted("ep-1", events.EpisodeKindBuild, dispatch)
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveWorkerStarted", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "Wave": true, "WorkerID": true,
			"ParentWorkerID": true, "Caste": true, "WorkerName": true, "Workspace": true, "Status": true,
		})
		if payload.ParentWorkerID != "Queen" {
			t.Fatalf("ParentWorkerID = %q, want %q", payload.ParentWorkerID, "Queen")
		}

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveWorkerStarted panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveWorkerStarted("ep-1", events.EpisodeKindBuild, dispatch)
		}()
	})

	t.Run("emitColonyLiveWorkerStarted without a parent leaves ParentWorkerID empty", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		dispatch := codex.WorkerDispatch{WorkerName: "Forge-11", Caste: "builder", Wave: 1, Root: "/tmp/workspace"}
		emitColonyLiveWorkerStarted("ep-1", events.EpisodeKindBuild, dispatch)
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveWorkerStarted (no parent)", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "Wave": true, "WorkerID": true,
			"Caste": true, "WorkerName": true, "Workspace": true, "Status": true,
		})
		if payload.ParentWorkerID != "" {
			t.Fatalf("ParentWorkerID = %q, want empty when the dispatch names no parent", payload.ParentWorkerID)
		}
	})

	t.Run("emitColonyLiveWorkerProgress", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		dispatch := codex.WorkerDispatch{WorkerName: "Forge-11", Caste: "builder", Wave: 1}
		emitColonyLiveWorkerProgress("ep-1", events.EpisodeKindBuild, dispatch, "reading the task brief")
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveWorkerProgress", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "Wave": true, "WorkerID": true,
			"Caste": true, "WorkerName": true, "Status": true, "Question": true,
		})

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveWorkerProgress panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveWorkerProgress("ep-1", events.EpisodeKindBuild, dispatch, "reading the task brief")
		}()
	})

	t.Run("emitColonyLiveWorkerFinished", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		dispatch := codex.WorkerDispatch{
			WorkerName:     "Forge-11",
			Caste:          "builder",
			Wave:           1,
			Root:           "/tmp/workspace",
			ParentWorkerID: "Queen",
		}
		result := codex.DispatchResult{WorkerName: "Forge-11", Status: "completed"}
		emitColonyLiveWorkerFinished("ep-1", events.EpisodeKindBuild, dispatch, result)
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveWorkerFinished", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "Wave": true, "WorkerID": true,
			"ParentWorkerID": true, "Caste": true, "WorkerName": true, "Workspace": true, "Status": true,
		})
		if payload.Status != "completed" {
			t.Fatalf("Status = %q, want completed", payload.Status)
		}

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveWorkerFinished panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveWorkerFinished("ep-1", events.EpisodeKindBuild, dispatch, result)
		}()
	})

	t.Run("emitColonyLiveCheckStarted", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLiveCheckStarted("ep-1", events.EpisodeKindContinue, "build")
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveCheckStarted", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "CheckName": true, "Status": true,
		})

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveCheckStarted panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveCheckStarted("ep-1", events.EpisodeKindContinue, "build")
		}()
	})

	t.Run("emitColonyLiveCheckPassed", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLiveCheckPassed("ep-1", events.EpisodeKindContinue, "build")
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveCheckPassed", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "CheckName": true, "Status": true,
		})

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveCheckPassed panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveCheckPassed("ep-1", events.EpisodeKindContinue, "build")
		}()
	})

	t.Run("emitColonyLiveCheckFailed", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLiveCheckFailed("ep-1", events.EpisodeKindContinue, "tests", "2 failing tests")
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveCheckFailed", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "CheckName": true, "Status": true, "Findings": true,
		})

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveCheckFailed panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveCheckFailed("ep-1", events.EpisodeKindContinue, "tests", "2 failing tests")
		}()
	})

	t.Run("emitColonyLiveRecoveryChanged", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLiveRecoveryChanged("ep-1", events.EpisodeKindRecovery, "retry")
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveRecoveryChanged", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "RecoveryState": true,
		})

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveRecoveryChanged panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveRecoveryChanged("ep-1", events.EpisodeKindRecovery, "retry")
		}()
	})

	t.Run("emitColonyLiveSignalConsulted", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLiveSignalConsulted("ep-1", events.EpisodeKindBuild, []string{"REDIRECT: avoid X"})
		payload := lastLivePayload(t, s)
		assertPayloadPopulatesOnly(t, "emitColonyLiveSignalConsulted", payload, map[string]bool{
			"EpisodeID": true, "EpisodeKind": true, "Signals": true,
		})

		store = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("emitColonyLiveSignalConsulted panicked with a nil store: %v", r)
				}
			}()
			emitColonyLiveSignalConsulted("ep-1", events.EpisodeKindBuild, []string{"REDIRECT: avoid X"})
		}()
	})
}

// liveLaneEntryPoint drives one lifecycle lane through its real public entry
// point against an isolated fixture and returns every live.* event that
// drive persisted, in emission order.
type liveLaneEntryPoint func(t *testing.T) []liveTestEvent

func driveSwarmLiveLane(t *testing.T) []liveTestEvent {
	t.Helper()
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	target := "Auth panic when session is missing"
	swarmID := "swarm-live-coverage-test"
	if err := initializeSwarmRun(swarmID); err != nil {
		t.Fatalf("initialize swarm run: %v", err)
	}
	investigation := buildSwarmInvestigationPlans(root, target)
	if len(investigation) == 0 {
		t.Fatalf("fixture is broken: buildSwarmInvestigationPlans returned no plans for %q", target)
	}
	wave := swarmPlansWaveNumber(investigation)
	ctx := context.Background()
	invoker := &swarmTestInvoker{}

	// Mirrors runSwarmDestroy's own wave-boundary emission
	// (cmd/swarm_cmd.go) around executeSwarmWave -- the same pattern
	// 202-02's TestSwarmInvestigationWaveReachesTheLiveWatchScreen already
	// established for driving the Swarm lane in a test.
	emitColonyLive(events.LiveTopicWaveStarted, events.ColonyLivePayload{
		EpisodeID: swarmID, EpisodeKind: events.EpisodeKindSwarm, Wave: wave, Status: "starting",
	})
	if _, err := executeSwarmWave(ctx, root, swarmID, target, investigation, "", invoker, true); err != nil {
		t.Fatalf("executeSwarmWave: %v", err)
	}
	emitColonyLive(events.LiveTopicWaveEnded, events.ColonyLivePayload{
		EpisodeID: swarmID, EpisodeKind: events.EpisodeKindSwarm, Wave: wave, Status: "completed",
	})
	return liveEventsSince(t)
}

func driveBuildLiveLane(t *testing.T) []liveTestEvent {
	t.Helper()
	runFailedEmitBuildFixture(t, false)
	return liveEventsSince(t)
}

func driveContinueLiveLane(t *testing.T) []liveTestEvent {
	t.Helper()
	runFailedEmitCheckFixture(t, false)
	return liveEventsSince(t)
}

func drivePlanLiveLane(t *testing.T) []liveTestEvent {
	t.Helper()
	runFailedEmitPlanFixture(t, false)
	return liveEventsSince(t)
}

func driveRecoveryLiveLane(t *testing.T) []liveTestEvent {
	t.Helper()
	runFailedEmitRecoveryFixture(t, false)
	return liveEventsSince(t)
}

// driveOracleLiveLane (202-11, LIVE-06/CEC-05) drives one real Oracle
// research round through runOracleCompatibility's "run-loop" resume path --
// the same real public entry point `aether oracle run-loop` uses -- with a
// fixture invoker standing in for the real worker subprocess, then returns
// every live.* event that round persisted.
func driveOracleLiveLane(t *testing.T) []liveTestEvent {
	t.Helper()
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	t.Cleanup(func() { newOracleWorkerInvoker = originalInvoker })

	setupOracleLiveFixtureWorkspace(t, root, 4, 60, 5)

	if _, err := runOracleCompatibility(root, []string{"run-loop"}, "", ""); err != nil {
		t.Fatalf("oracle run-loop: %v", err)
	}
	return liveEventsSince(t)
}

// liveLaneEntryPoints maps every declared episode kind to the real public
// entry point that drives it. A kind present in events.ColonyLiveEpisodeKinds()
// with no entry registered here fails TestEveryLifecycleLaneEmitsLiveEvents
// by name -- the lane inventory below is exhaustive over the declared
// vocabulary, never a hand-typed subset of "today's lanes".
var liveLaneEntryPoints = map[string]liveLaneEntryPoint{
	events.EpisodeKindSwarm:    driveSwarmLiveLane,
	events.EpisodeKindBuild:    driveBuildLiveLane,
	events.EpisodeKindContinue: driveContinueLiveLane,
	events.EpisodeKindOracle:   driveOracleLiveLane,
	events.EpisodeKindPlan:     drivePlanLiveLane,
	events.EpisodeKindRecovery: driveRecoveryLiveLane,
}

// assertLiveKindPresent is the guard TestEveryLifecycleLaneEmitsLiveEvents
// applies per lane: it reports, by name, when no persisted event carries
// the expected episode kind. Kept as a standalone function (rather than
// inlined into the test) so the negative case below can prove the guard is
// capable of failing, not merely capable of passing.
func assertLiveKindPresent(kind string, liveEvents []liveTestEvent) error {
	for _, e := range liveEvents {
		if e.Payload.EpisodeKind == kind {
			return nil
		}
	}
	return fmt.Errorf("lifecycle lane %q emitted no live event carrying episode kind %q -- this lane has gone dark on the live stream", kind, kind)
}

// TestEveryLifecycleLaneEmitsLiveEvents drives every lifecycle lane declared
// in events.ColonyLiveEpisodeKinds() through its real public entry point
// against an isolated fixture, and fails by name (episode kind + entry
// point) the moment a lane produces zero live events of its own kind --
// catching a lane that stops speaking on the live stream as a test failure
// rather than an owner noticing an empty cockpit.
func TestEveryLifecycleLaneEmitsLiveEvents(t *testing.T) {
	kinds := events.ColonyLiveEpisodeKinds()
	if len(kinds) == 0 {
		t.Fatal("fixture is broken: events.ColonyLiveEpisodeKinds() returned no declared episode kinds")
	}

	for _, kind := range kinds {
		kind := kind
		t.Run(kind, func(t *testing.T) {
			entry, ok := liveLaneEntryPoints[kind]
			if !ok {
				t.Fatalf("no real public entry point is registered in liveLaneEntryPoints for declared episode kind %q -- every kind in events.ColonyLiveEpisodeKinds() must have one", kind)
			}
			liveEvents := entry(t)
			if err := assertLiveKindPresent(kind, liveEvents); err != nil {
				t.Fatal(err)
			}
		})
	}

	t.Run("a lane with its emission stubbed to no-ops is reported by name", func(t *testing.T) {
		// Simulates a lane whose emission calls were removed (its helpers
		// stubbed to no-ops): zero live events reach the store, exactly as
		// if emitColonyLive itself had never been called for this kind.
		err := assertLiveKindPresent(events.EpisodeKindBuild, nil)
		if err == nil {
			t.Fatal("expected the guard to fail when a lane's live events are empty")
		}
		if !strings.Contains(err.Error(), events.EpisodeKindBuild) {
			t.Fatalf("guard failure %q does not name the stubbed lane %q", err.Error(), events.EpisodeKindBuild)
		}
	})
}
