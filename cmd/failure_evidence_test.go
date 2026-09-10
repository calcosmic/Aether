package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// Shared fixtures (201-10)
// ---------------------------------------------------------------------------

// seedMinimalBuildAttempt persists just enough of a durable build attempt
// record (plus its latest-attempt pointer) for loadLatestBuildAttempt(phase)
// to resolve it -- the real mechanism recordDispatchWorkerOutcome now reads
// attempt/job identity from (cmd/memory_feed.go), without running the full
// commitBuildStart pipeline.
func seedMinimalBuildAttempt(t *testing.T, phase int, attemptID string, dispatches []codexBuildDispatch) {
	t.Helper()
	if store == nil {
		t.Fatal("seedMinimalBuildAttempt: store is nil")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	attemptRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase), "attempts", attemptID+".json"))
	record := buildAttemptRecord{
		SchemaVersion: buildAttemptSchemaVersion,
		ID:            attemptID,
		Phase:         phase,
		Status:        buildAttemptTerminal,
		StartedAt:     now,
		UpdatedAt:     now,
		Dispatches:    dispatches,
		History:       []buildAttemptTransition{},
	}
	if err := store.SaveJSON(attemptRel, record); err != nil {
		t.Fatalf("seed build attempt: %v", err)
	}
	pointer := latestBuildAttemptPointer{
		SchemaVersion: 1,
		AttemptID:     attemptID,
		Path:          attemptRel,
		UpdatedAt:     now,
	}
	if err := store.SaveJSON(latestBuildAttemptPointerPath(phase), pointer); err != nil {
		t.Fatalf("seed latest attempt pointer: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Task 1: failure evidence and blocker truth bound to the exact attempt.
// ---------------------------------------------------------------------------

func TestFailureEvidenceCarriesTheAttemptIdentity(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	const phase = 7
	const attemptID = "attempt-evidence-1"
	seedMinimalBuildAttempt(t, phase, attemptID, []codexBuildDispatch{
		{Name: "Mason-1", Caste: "builder", JobName: "job-alpha"},
	})

	dispatch := codex.WorkerDispatch{WorkerName: "Mason-1", Caste: "builder", Workflow: "build", Phase: phase}
	result := codex.DispatchResult{
		WorkerName: "Mason-1",
		Status:     "failed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Mason-1", Caste: "builder", Status: "failed",
			Blockers: []string{"go test ./cmd -run TestExpire failed: nil pointer dereference"},
		},
	}

	if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
		t.Fatalf("recordDispatchWorkerOutcome: %v", err)
	}

	mf, err := loadMiddenFile(s)
	if err != nil {
		t.Fatalf("load midden: %v", err)
	}
	if len(mf.Entries) != 1 {
		t.Fatalf("expected exactly 1 midden entry, got %d: %+v", len(mf.Entries), mf.Entries)
	}
	entry := mf.Entries[0]
	if got := middenEntryAttemptID(entry); got != attemptID {
		t.Fatalf("midden entry attempt ID = %q, want %q (tags: %v)", got, attemptID, entry.Tags)
	}
	if got := middenEntryJobName(entry); got != "job-alpha" {
		t.Fatalf("midden entry job name = %q, want %q (tags: %v)", got, "job-alpha", entry.Tags)
	}
	if !strings.HasPrefix(entry.Message, "go test ./cmd -run TestExpire failed") {
		t.Fatalf("midden entry message = %q, want it to start with the worker's own sentence", entry.Message)
	}
}

// TestEveryBuildLaneFeedsMemoryThroughOneBoundary (cmd/memory_feed_test.go)
// already proves both build lanes call recordDispatchWorkerOutcome and no
// other function persists a worker handoff -- this plan extends that one
// boundary rather than adding a second, so a single call through it (above)
// is sufficient evidence for "either lane": both lanes construct the exact
// same workerOutcomeFacts inside this one function.

func TestBlockerTruthIsOneStore(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	const phase = 9
	const attemptID = "attempt-blocker-1"
	seedMinimalBuildAttempt(t, phase, attemptID, []codexBuildDispatch{
		{Name: "Hammer-3", Caste: "builder"},
	})

	dispatch := codex.WorkerDispatch{WorkerName: "Hammer-3", Caste: "builder", Workflow: "build", Phase: phase}
	result := codex.DispatchResult{
		WorkerName: "Hammer-3",
		Status:     "blocked",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Hammer-3", Caste: "builder", Status: "blocked",
			Blockers: []string{"cannot deploy: missing PROD_DB_URL secret"},
		},
	}

	if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
		t.Fatalf("recordDispatchWorkerOutcome: %v", err)
	}

	// Surface 1: status/advancement read (readBlockerSnapshot,
	// cmd/blocker_snapshot.go).
	snapshot, err := readBlockerSnapshot(s)
	if err != nil {
		t.Fatalf("readBlockerSnapshot: %v", err)
	}
	if snapshot.Count != 1 || snapshot.EscalatedCount != 1 {
		t.Fatalf("blocker snapshot = %+v, want Count=1 EscalatedCount=1", snapshot)
	}

	// Surface 2: the advancement gate itself
	// (checkUnresolvedBlockerFlags, cmd/gate.go) -- the classic Iron Law.
	check := checkUnresolvedBlockerFlags()
	if check.Passed {
		t.Fatalf("checkUnresolvedBlockerFlags passed with an unresolved worker blocker outstanding: %+v", check)
	}

	// Surface 3: the closure read (LifecycleFacts.Blockers,
	// cmd/lifecycle_facts.go), which reads pending-decisions.json directly
	// via readLifecyclePendingDecisions -- the exact same file, proving all
	// three surfaces resolve from one store.
	_, flags, source := readLifecyclePendingDecisions(filepath.Join(s.BasePath(), "pending-decisions.json"))
	if source.Provenance != LifecycleFactConfirmed {
		t.Fatalf("readLifecyclePendingDecisions provenance = %v, want confirmed", source.Provenance)
	}
	found := false
	for _, flag := range flags.Decisions {
		if flag.AttemptID == attemptID && flag.Type == "blocker" {
			found = true
			if flag.Source != "escalation" {
				t.Errorf("blocker flag Source = %q, want %q", flag.Source, "escalation")
			}
		}
	}
	if !found {
		t.Fatalf("closure read (readLifecyclePendingDecisions) does not see the worker-reported blocker bound to attempt %q: %+v", attemptID, flags.Decisions)
	}
}

func TestEscalatedCountHasOneCountingPath(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	const phase = 11
	seedMinimalBuildAttempt(t, phase, "attempt-count-1", []codexBuildDispatch{
		{Name: "Worker-A", Caste: "builder"},
		{Name: "Worker-B", Caste: "builder"},
		{Name: "Worker-C", Caste: "builder"},
	})

	workers := []struct {
		name    string
		blocker string
	}{
		{"Worker-A", "blocked on missing API key A"},
		{"Worker-B", "blocked on missing API key B"},
		{"Worker-C", "blocked on missing API key C"},
	}
	for _, w := range workers {
		dispatch := codex.WorkerDispatch{WorkerName: w.name, Caste: "builder", Workflow: "build", Phase: phase}
		result := codex.DispatchResult{
			WorkerName: w.name, Status: "blocked",
			WorkerResult: &codex.WorkerResult{WorkerName: w.name, Caste: "builder", Status: "blocked", Blockers: []string{w.blocker}},
		}
		if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
			t.Fatalf("recordDispatchWorkerOutcome(%s): %v", w.name, err)
		}
	}

	// The fixture's own known count: 3 distinct worker blockers.
	const wantEscalated = 3

	snapshot, err := readBlockerSnapshot(s)
	if err != nil {
		t.Fatalf("readBlockerSnapshot: %v", err)
	}
	if snapshot.EscalatedCount != wantEscalated {
		t.Fatalf("readBlockerSnapshot.EscalatedCount = %d, want %d (fixture's own count)", snapshot.EscalatedCount, wantEscalated)
	}

	// addBlockerSnapshotFields (the status-surface renderer) must report the
	// exact same number -- computed from the same snapshot, never a second
	// count.
	result := map[string]interface{}{}
	addBlockerSnapshotFields(result, snapshot)
	if got := result["escalated_blockers"]; got != wantEscalated {
		t.Fatalf("addBlockerSnapshotFields[\"escalated_blockers\"] = %v, want %d", got, wantEscalated)
	}

	// The advancement gate's own detail names the same total blocker count.
	check := checkUnresolvedBlockerFlags()
	if check.Passed {
		t.Fatalf("checkUnresolvedBlockerFlags passed with %d unresolved blockers outstanding", wantEscalated)
	}
	if !strings.Contains(check.Detail, fmt.Sprintf("%d unresolved blocker", wantEscalated)) {
		t.Fatalf("checkUnresolvedBlockerFlags detail = %q, want it to name %d unresolved blockers", check.Detail, wantEscalated)
	}
}

func TestEvidenceStorageFailureNeverFailsTheRun(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s

	// Make the store's directory unwritable so any write inside it --
	// including the worker-handoffs file, midden.json, and
	// pending-decisions.json -- fails.
	if err := os.Chmod(dataDir, 0o500); err != nil {
		t.Fatalf("chmod data dir read-only: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dataDir, 0o755) })

	dispatch := codex.WorkerDispatch{WorkerName: "Mason-1", Caste: "builder", Workflow: "build", Phase: 1}
	result := codex.DispatchResult{
		WorkerName: "Mason-1",
		Status:     "failed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Mason-1", Caste: "builder", Status: "failed",
			Blockers: []string{"go build ./cmd/aether failed: unwritable store"},
		},
	}

	wantErr := persistDispatchWorkerHandoff(dispatch, result)
	if wantErr == nil {
		t.Skip("environment did not make the store directory unwritable (e.g. running as root); nothing to assert")
	}
	gotErr := recordDispatchWorkerOutcome(dispatch, result)
	if gotErr == nil || gotErr.Error() != wantErr.Error() {
		t.Fatalf("recordDispatchWorkerOutcome error = %v, want %v (persistDispatchWorkerHandoff's own error, unchanged -- the blocker/midden writes must never add their own failure)", gotErr, wantErr)
	}
}

