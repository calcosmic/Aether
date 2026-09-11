package cmd

import (
	"encoding/json"
	"reflect"
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
