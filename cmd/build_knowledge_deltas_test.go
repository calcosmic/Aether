package cmd

import (
	"reflect"
	"testing"
)

// Phase 201 plan 18 (gap closure, 201-VERIFICATION.md CAP-066) -- these are
// deriveBuildKnowledgeDeltas' own unit tests: given a set of persisted
// worker handoff records and a set of dispatches, it derives exactly the
// decision/learning deltas this attempt's own workers left behind, nothing
// more.

// seedWorkerHandoffRecordsForTest writes records directly to the persisted
// handoff file (workerHandoffsPath) via the store, bypassing
// persistDispatchWorkerHandoff -- this is a fixture for exercising the pure
// read-side derivation in isolation, not a production write path.
func seedWorkerHandoffRecordsForTest(t *testing.T, records ...workerHandoffRecord) {
	t.Helper()
	if err := store.SaveJSON(workerHandoffsPath, workerHandoffFile{Entries: records}); err != nil {
		t.Fatalf("seed worker handoff records: %v", err)
	}
}

func TestDeriveBuildKnowledgeDeltas(t *testing.T) {
	t.Run("no dispatches returns nil", func(t *testing.T) {
		setupSpendTestStore(t)
		if got := deriveBuildKnowledgeDeltas(1, nil); got != nil {
			t.Fatalf("expected nil for no dispatches, got %+v", got)
		}
	})

	t.Run("workers with no decisions and no lessons return nil, not an empty slice", func(t *testing.T) {
		setupSpendTestStore(t)
		dispatches := []codexBuildDispatch{{Name: "Mason-1", Caste: "builder"}}
		seedWorkerHandoffRecordsForTest(t, workerHandoffRecord{
			Phase: 1, WorkerName: "Mason-1", Status: "completed",
		})
		if got := deriveBuildKnowledgeDeltas(1, dispatches); got != nil {
			t.Fatalf("expected nil for a content-free handoff, got %+v", got)
		}
	})

	t.Run("a different phase contributes nothing", func(t *testing.T) {
		setupSpendTestStore(t)
		dispatches := []codexBuildDispatch{{Name: "Mason-1", Caste: "builder"}}
		seedWorkerHandoffRecordsForTest(t, workerHandoffRecord{
			Phase: 2, WorkerName: "Mason-1",
			OpenDecisions: []string{"Chose Postgres for the widget store"},
		})
		if got := deriveBuildKnowledgeDeltas(1, dispatches); got != nil {
			t.Fatalf("expected nil for a different-phase handoff, got %+v", got)
		}
	})

	t.Run("a worker not in the supplied dispatch list contributes nothing", func(t *testing.T) {
		setupSpendTestStore(t)
		dispatches := []codexBuildDispatch{{Name: "Mason-1", Caste: "builder"}}
		seedWorkerHandoffRecordsForTest(t, workerHandoffRecord{
			Phase: 1, WorkerName: "Someone-Else",
			OpenDecisions: []string{"Chose Postgres for the widget store"},
		})
		if got := deriveBuildKnowledgeDeltas(1, dispatches); got != nil {
			t.Fatalf("expected nil for a worker outside this attempt's own dispatches, got %+v", got)
		}
	})

	t.Run("open decisions become decision deltas, do-not-repeat and next-worker instructions become learning deltas", func(t *testing.T) {
		setupSpendTestStore(t)
		dispatches := []codexBuildDispatch{{Name: "Mason-1", Caste: "builder"}}
		seedWorkerHandoffRecordsForTest(t, workerHandoffRecord{
			Phase: 1, WorkerName: "Mason-1",
			OpenDecisions:          []string{"Chose Postgres for the widget store"},
			DoNotRepeat:            []string{"Do not retry the flaky endpoint without backoff"},
			NextWorkerInstructions: []string{"Wire the new endpoint into the dashboard"},
		})
		got := deriveBuildKnowledgeDeltas(1, dispatches)
		want := []buildAttemptKnowledgeDelta{
			{Kind: "decision", Summary: "Chose Postgres for the widget store"},
			{Kind: "learning", Summary: "Do not retry the flaky endpoint without backoff"},
			{Kind: "learning", Summary: "Wire the new endpoint into the dashboard"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("deriveBuildKnowledgeDeltas = %+v, want %+v", got, want)
		}
	})

	t.Run("an exact duplicate across two workers is kept once", func(t *testing.T) {
		setupSpendTestStore(t)
		dispatches := []codexBuildDispatch{{Name: "Mason-1", Caste: "builder"}, {Name: "Keen-2", Caste: "watcher"}}
		seedWorkerHandoffRecordsForTest(t,
			workerHandoffRecord{Phase: 1, WorkerName: "Mason-1", OpenDecisions: []string{"Chose Postgres for the widget store"}},
			workerHandoffRecord{Phase: 1, WorkerName: "Keen-2", OpenDecisions: []string{"Chose Postgres for the widget store"}},
		)
		got := deriveBuildKnowledgeDeltas(1, dispatches)
		if len(got) != 1 {
			t.Fatalf("expected exactly 1 deduplicated delta, got %+v", got)
		}
	})

	t.Run("ordering is deterministic across repeated calls", func(t *testing.T) {
		setupSpendTestStore(t)
		dispatches := []codexBuildDispatch{{Name: "Mason-1", Caste: "builder"}, {Name: "Keen-2", Caste: "watcher"}}
		seedWorkerHandoffRecordsForTest(t,
			workerHandoffRecord{Phase: 1, WorkerName: "Keen-2", NextWorkerInstructions: []string{"Watch the new endpoint's latency"}},
			workerHandoffRecord{Phase: 1, WorkerName: "Mason-1", OpenDecisions: []string{"Chose Postgres for the widget store"}},
		)
		first := deriveBuildKnowledgeDeltas(1, dispatches)
		second := deriveBuildKnowledgeDeltas(1, dispatches)
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("two calls on identical input diverged: %+v vs %+v", first, second)
		}
		if len(first) != 2 || first[0].Summary != "Chose Postgres for the widget store" {
			t.Fatalf("expected dispatch-order-anchored output (Mason-1 before Keen-2), got %+v", first)
		}
	})

	t.Run("performs no write", func(t *testing.T) {
		setupSpendTestStore(t)
		dispatches := []codexBuildDispatch{{Name: "Mason-1", Caste: "builder"}}
		seedWorkerHandoffRecordsForTest(t, workerHandoffRecord{
			Phase: 1, WorkerName: "Mason-1",
			OpenDecisions: []string{"Chose Postgres for the widget store"},
		})
		before := hashDirForTest(t, store.BasePath())
		_ = deriveBuildKnowledgeDeltas(1, dispatches)
		after := hashDirForTest(t, store.BasePath())
		if before != after {
			t.Fatalf("deriveBuildKnowledgeDeltas wrote to the fixture directory")
		}
	})
}
