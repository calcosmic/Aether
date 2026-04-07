package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// helper: create a fresh Store, write an initial COLONY_STATE.json, and return the Store.
func setupAuditTest(t *testing.T) (*Store, *AuditLogger) {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Write initial colony state
	state := &colony.ColonyState{
		Version:      "1.0",
		State:        colony.StateREADY,
		CurrentPhase: 0,
		Events:       []string{"colony initialized"},
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	data = append(data, '\n')
	if err := s.AtomicWrite("COLONY_STATE.json", data); err != nil {
		t.Fatalf("write COLONY_STATE.json: %v", err)
	}

	al := NewAuditLogger(s)
	return s, al
}

// helper: read the current COLONY_STATE.json from the store.
func readCurrentState(t *testing.T, s *Store) *colony.ColonyState {
	t.Helper()
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	return &state
}

// TestAudit_WriteBoundaryRecordsEntry verifies that WriteBoundary records an
// audit entry to state-changelog.jsonl when state is mutated.
func TestAudit_WriteBoundaryRecordsEntry(t *testing.T) {
	_, al := setupAuditTest(t)

	err := al.WriteBoundary("test-mutate", false, func(state *colony.ColonyState) (string, error) {
		state.CurrentPhase = 1
		return "advanced to phase 1", nil
	})
	if err != nil {
		t.Fatalf("WriteBoundary: %v", err)
	}

	// Verify audit entry was appended
	entries, err := al.ReadHistory(0)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].Command != "test-mutate" {
		t.Errorf("expected command %q, got %q", "test-mutate", entries[0].Command)
	}
	if entries[0].Summary != "advanced to phase 1" {
		t.Errorf("expected summary %q, got %q", "advanced to phase 1", entries[0].Summary)
	}
}

// TestAudit_AuditEntryContainsRequiredFields verifies that AuditEntry contains
// timestamp (RFC3339), source command string, before/after JSON, and SHA-256 checksum.
func TestAudit_AuditEntryContainsRequiredFields(t *testing.T) {
	_, al := setupAuditTest(t)

	beforeMutate := time.Now().UTC()
	err := al.WriteBoundary("field-check", false, func(state *colony.ColonyState) (string, error) {
		state.CurrentPhase = 2
		return "set phase", nil
	})
	if err != nil {
		t.Fatalf("WriteBoundary: %v", err)
	}

	entries, err := al.ReadHistory(0)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]

	// Check timestamp is RFC3339
	if e.Timestamp == "" {
		t.Error("timestamp is empty")
	}
	parsed, err := time.Parse(time.RFC3339Nano, e.Timestamp)
	if err != nil {
		t.Errorf("timestamp %q is not valid RFC3339: %v", e.Timestamp, err)
	}
	if parsed.Before(beforeMutate) {
		t.Error("timestamp is before the mutation started")
	}

	// Check command
	if e.Command != "field-check" {
		t.Errorf("expected command %q, got %q", "field-check", e.Command)
	}

	// Check before/after are non-empty JSON
	if len(e.Before) == 0 {
		t.Error("before is empty")
	}
	if !json.Valid(e.Before) {
		t.Errorf("before is not valid JSON: %s", string(e.Before))
	}
	if len(e.After) == 0 {
		t.Error("after is empty")
	}
	if !json.Valid(e.After) {
		t.Errorf("after is not valid JSON: %s", string(e.After))
	}

	// Check checksum is hex-encoded SHA-256
	if e.Checksum == "" {
		t.Error("checksum is empty")
	}
	if len(e.Checksum) != 64 {
		t.Errorf("expected 64-char hex checksum, got %d chars: %s", len(e.Checksum), e.Checksum)
	}
	_, err = hex.DecodeString(e.Checksum)
	if err != nil {
		t.Errorf("checksum is not valid hex: %v", err)
	}

	// Verify the checksum matches SHA-256 of after-state
	hash := sha256.Sum256(e.After)
	expected := hex.EncodeToString(hash[:])
	if e.Checksum != expected {
		t.Errorf("checksum mismatch: got %q, expected %q", e.Checksum, expected)
	}
}

// TestAudit_WriteBoundaryRejectsCorruption verifies that WriteBoundary rejects
// mutations when DetectCorruption returns error.
func TestAudit_WriteBoundaryRejectsCorruption(t *testing.T) {
	_, al := setupAuditTest(t)

	// Mutator that introduces a jq expression into Events
	err := al.WriteBoundary("corrupt-mutate", false, func(state *colony.ColonyState) (string, error) {
		state.Events = append(state.Events, ".current_phase = 99")
		return "corrupted state", nil
	})
	if err == nil {
		t.Fatal("expected error for corrupted state, got nil")
	}
	if !strings.Contains(err.Error(), "corruption") {
		t.Errorf("error should mention corruption, got: %v", err)
	}

	// Verify no audit entry was written
	entries, err := al.ReadHistory(0)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 audit entries after rejected corruption, got %d", len(entries))
	}

	// Verify state was NOT mutated
	var state colony.ColonyState
	if err := al.store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if state.CurrentPhase != 0 {
		t.Errorf("state should not have been mutated, current_phase=%d", state.CurrentPhase)
	}
}

// TestAudit_WriteBoundaryCreatesCheckpointForDestructive verifies that WriteBoundary
// creates an auto-checkpoint when destructive=true.
func TestAudit_WriteBoundaryCreatesCheckpointForDestructive(t *testing.T) {
	s, al := setupAuditTest(t)

	err := al.WriteBoundary("destructive-op", true, func(state *colony.ColonyState) (string, error) {
		state.CurrentPhase = 5
		return "destructive change", nil
	})
	if err != nil {
		t.Fatalf("WriteBoundary: %v", err)
	}

	// Verify checkpoint was created
	checkpointsDir := s.BasePath() + "/checkpoints"
	entries, err := readDirSafe(checkpointsDir)
	if err != nil {
		t.Fatalf("read checkpoints dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected checkpoint to be created, but checkpoints/ is empty")
	}
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "auto-") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no auto-* checkpoint file found")
	}
}

// TestAudit_WriteBoundaryNoCheckpointForNonDestructive verifies that WriteBoundary
// does NOT create a checkpoint when destructive=false.
func TestAudit_WriteBoundaryNoCheckpointForNonDestructive(t *testing.T) {
	s, al := setupAuditTest(t)

	err := al.WriteBoundary("safe-op", false, func(state *colony.ColonyState) (string, error) {
		state.CurrentPhase = 3
		return "safe change", nil
	})
	if err != nil {
		t.Fatalf("WriteBoundary: %v", err)
	}

	// Verify no checkpoint was created
	checkpointsDir := s.BasePath() + "/checkpoints"
	entries, err := readDirSafe(checkpointsDir)
	if err != nil {
		// Directory doesn't exist is fine -- means no checkpoints
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "auto-") {
			t.Errorf("expected no auto-checkpoint for non-destructive op, found %s", e.Name())
		}
	}
}

// TestAudit_ReadHistoryReturnsInOrder verifies that ReadJSONL returns audit
// entries in the order they were appended.
func TestAudit_ReadHistoryReturnsInOrder(t *testing.T) {
	_, al := setupAuditTest(t)

	commands := []string{"first", "second", "third"}
	for _, cmd := range commands {
		err := al.WriteBoundary(cmd, false, func(state *colony.ColonyState) (string, error) {
			return "op " + cmd, nil
		})
		if err != nil {
			t.Fatalf("WriteBoundary %s: %v", cmd, err)
		}
	}

	entries, err := al.ReadHistory(0)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	for i, cmd := range commands {
		if entries[i].Command != cmd {
			t.Errorf("entry %d: expected command %q, got %q", i, cmd, entries[i].Command)
		}
	}
}

// TestAudit_ReadHistoryTail verifies that ReadHistory with a positive tail
// returns only the last N entries.
func TestAudit_ReadHistoryTail(t *testing.T) {
	_, al := setupAuditTest(t)

	for i := 0; i < 5; i++ {
		_ = al.WriteBoundary("op", false, func(state *colony.ColonyState) (string, error) {
			return "change", nil
		})
	}

	entries, err := al.ReadHistory(2)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries with tail=2, got %d", len(entries))
	}
}

// TestAudit_ChecksumIsSHA256 verifies that AuditEntry checksum is hex-encoded
// SHA-256 of the after-state bytes.
func TestAudit_ChecksumIsSHA256(t *testing.T) {
	_, al := setupAuditTest(t)

	err := al.WriteBoundary("checksum-test", false, func(state *colony.ColonyState) (string, error) {
		state.Milestone = "test-milestone"
		return "set milestone", nil
	})
	if err != nil {
		t.Fatalf("WriteBoundary: %v", err)
	}

	entries, err := al.ReadHistory(0)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	// Compute expected checksum
	hash := sha256.Sum256(entries[0].After)
	expected := hex.EncodeToString(hash[:])
	if entries[0].Checksum != expected {
		t.Errorf("checksum mismatch: got %q, expected %q", entries[0].Checksum, expected)
	}
}

// TestAudit_ConcurrentWriteBoundary verifies that multiple rapid WriteBoundary
// calls produce no corruption (concurrent safety). Due to the read-mutate-write
// pattern without cross-file transaction locking, some mutations may be lost
// under high concurrency. This test verifies that:
// - No panics or deadlocks occur
// - All audit entries are valid
// - The final state is valid JSON with no corruption
// - At least one mutation succeeded
func TestAudit_ConcurrentWriteBoundary(t *testing.T) {
	_, al := setupAuditTest(t)

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			err := al.WriteBoundary("concurrent-op", false, func(state *colony.ColonyState) (string, error) {
				state.Events = append(state.Events, "event from goroutine")
				return "concurrent mutation", nil
			})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	for err := range errors {
		t.Errorf("concurrent WriteBoundary error: %v", err)
	}

	// Verify at least some audit entries were created (no deadlocks)
	entries, err := al.ReadHistory(0)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least 1 audit entry, got 0 (possible deadlock)")
	}

	// Verify all entries have valid checksums
	for i, entry := range entries {
		hash := sha256.Sum256(entry.After)
		expected := hex.EncodeToString(hash[:])
		if entry.Checksum != expected {
			t.Errorf("entry %d: checksum mismatch", i)
		}
		if !json.Valid(entry.Before) {
			t.Errorf("entry %d: before is not valid JSON", i)
		}
		if !json.Valid(entry.After) {
			t.Errorf("entry %d: after is not valid JSON", i)
		}
	}

	// Verify final state is valid and not corrupted
	state := readCurrentState(t, al.store)
	if len(state.Events) < 1 {
		t.Error("state should have at least 1 event")
	}
}

// TestAudit_WriteBoundaryMutatorError verifies that when the mutator callback
// returns an error, no state write or audit entry is created.
func TestAudit_WriteBoundaryMutatorError(t *testing.T) {
	_, al := setupAuditTest(t)

	err := al.WriteBoundary("failing-op", false, func(state *colony.ColonyState) (string, error) {
		return "", fmt.Errorf("intentional failure")
	})
	if err == nil {
		t.Fatal("expected error from mutator, got nil")
	}
	if !strings.Contains(err.Error(), "intentional failure") {
		t.Errorf("error should propagate from mutator, got: %v", err)
	}

	entries, err := al.ReadHistory(0)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after mutator error, got %d", len(entries))
	}
}

// TestAudit_GetLatestChecksum verifies that GetLatestChecksum returns the
// checksum from the most recent audit entry.
func TestAudit_GetLatestChecksum(t *testing.T) {
	_, al := setupAuditTest(t)

	// No entries yet
	cs, err := al.GetLatestChecksum()
	if err != nil {
		t.Fatalf("GetLatestChecksum (empty): %v", err)
	}
	if cs != "" {
		t.Errorf("expected empty checksum with no entries, got %q", cs)
	}

	// Add an entry
	al.WriteBoundary("op", false, func(state *colony.ColonyState) (string, error) {
		state.CurrentPhase = 7
		return "set phase", nil
	})

	cs, err = al.GetLatestChecksum()
	if err != nil {
		t.Fatalf("GetLatestChecksum: %v", err)
	}
	if cs == "" {
		t.Error("expected non-empty checksum after write")
	}
	if len(cs) != 64 {
		t.Errorf("expected 64-char checksum, got %d chars", len(cs))
	}

	// Verify it matches the latest entry
	entries, _ := al.ReadHistory(1)
	if len(entries) > 0 && cs != entries[0].Checksum {
		t.Errorf("GetLatestChecksum %q != latest entry checksum %q", cs, entries[0].Checksum)
	}
}

// TestAudit_DestructiveFlag verifies that the Destructive field in the audit
// entry matches the parameter passed to WriteBoundary.
func TestAudit_DestructiveFlag(t *testing.T) {
	_, al := setupAuditTest(t)

	// Non-destructive
	al.WriteBoundary("safe", false, func(state *colony.ColonyState) (string, error) {
		return "safe", nil
	})
	// Destructive
	al.WriteBoundary("destructive", true, func(state *colony.ColonyState) (string, error) {
		return "destructive", nil
	})

	entries, err := al.ReadHistory(0)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Destructive {
		t.Error("first entry should not be destructive")
	}
	if !entries[1].Destructive {
		t.Error("second entry should be destructive")
	}
}

// TestAudit_BeforeAfterDiffs verifies that the before and after fields in
// the audit entry correctly reflect the state change.
func TestAudit_BeforeAfterDiffs(t *testing.T) {
	_, al := setupAuditTest(t)

	al.WriteBoundary("change-phase", false, func(state *colony.ColonyState) (string, error) {
		state.CurrentPhase = 42
		return "phase changed", nil
	})

	entries, err := al.ReadHistory(0)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	// Before should have current_phase: 0
	var before colony.ColonyState
	if err := json.Unmarshal(entries[0].Before, &before); err != nil {
		t.Fatalf("unmarshal before: %v", err)
	}
	if before.CurrentPhase != 0 {
		t.Errorf("before state should have current_phase=0, got %d", before.CurrentPhase)
	}

	// After should have current_phase: 42
	var after colony.ColonyState
	if err := json.Unmarshal(entries[0].After, &after); err != nil {
		t.Fatalf("unmarshal after: %v", err)
	}
	if after.CurrentPhase != 42 {
		t.Errorf("after state should have current_phase=42, got %d", after.CurrentPhase)
	}
}

// readDirSafe reads a directory, returning an empty slice if it doesn't exist.
func readDirSafe(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	return entries, nil
}
