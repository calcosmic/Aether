package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

const colonyStatePath = "COLONY_STATE.json"
const auditLogPath = "state-changelog.jsonl"

// AuditEntry represents a single entry in the state audit log.
// Each entry captures the full context of a state mutation.
type AuditEntry struct {
	Timestamp   string          `json:"timestamp"`
	Command     string          `json:"command"`
	Path        string          `json:"path,omitempty"`
	Before      json.RawMessage `json:"before,omitempty"`
	After       json.RawMessage `json:"after,omitempty"`
	Summary     string          `json:"summary,omitempty"`
	Checksum    string          `json:"checksum"`
	Destructive bool            `json:"destructive"`
}

// AuditLogger provides an append-only audit trail for colony state mutations.
// It wraps a Store and exposes WriteBoundary as the single entry point for
// all state mutations, ensuring every change is logged with full before/after
// context and integrity checksums.
type AuditLogger struct {
	store *Store
}

// NewAuditLogger creates an AuditLogger that writes audit entries to the
// state-changelog.jsonl file within the given Store.
func NewAuditLogger(s *Store) *AuditLogger {
	return &AuditLogger{store: s}
}

// WriteBoundary is the core read-mutate-validate-write-audit pipeline.
// It atomically:
//  1. Reads the current COLONY_STATE.json (captures before-state)
//  2. Calls the mutator callback to modify the state
//  3. Runs corruption detection on the mutated state
//  4. Creates an auto-checkpoint if destructive=true
//  5. Writes the mutated state back via AtomicWrite
//  6. Computes SHA-256 checksum of the after-state
//  7. Appends an audit entry to state-changelog.jsonl
//
// If the mutator returns an error, no write or audit entry is created.
// If corruption is detected, the state is NOT written.
// If the audit append fails, the state write is NOT rolled back (the write
// has already succeeded atomically).
func (al *AuditLogger) WriteBoundary(cmd string, destructive bool, mutator func(state *colony.ColonyState) (string, error)) error {
	// Step 1: Read current state (captures raw bytes for before-state)
	beforeBytes, err := al.store.ReadFile(colonyStatePath)
	if err != nil {
		return fmt.Errorf("audit: read state: %w", err)
	}
	// Trim trailing whitespace for clean JSON in audit entry
	beforeJSON := bytes.TrimRight(beforeBytes, "\n\r\t ")

	var state colony.ColonyState
	if err := json.Unmarshal(beforeJSON, &state); err != nil {
		return fmt.Errorf("audit: unmarshal state: %w", err)
	}

	// Step 2: Call mutator
	summary, err := mutator(&state)
	if err != nil {
		return fmt.Errorf("audit: mutator error: %w", err)
	}

	// Step 3: Run corruption detection
	if err := DetectCorruption(&state); err != nil {
		return fmt.Errorf("audit: %w", err)
	}

	// Step 4: Auto-checkpoint for destructive operations
	if destructive {
		if cpErr := AutoCheckpoint(al.store, beforeJSON); cpErr != nil {
			// Log checkpoint failure but don't block the mutation
			fmt.Fprintf(os.Stderr, "audit: checkpoint failed: %v\n", cpErr)
		}
	}

	// Step 5: Marshal and write the mutated state
	afterJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("audit: marshal state: %w", err)
	}
	afterBytes := append(afterJSON, '\n')

	if err := al.store.AtomicWrite(colonyStatePath, afterBytes); err != nil {
		return fmt.Errorf("audit: write state: %w", err)
	}

	// Step 6: Compute SHA-256 checksum of the after-state
	// Use compact JSON for the audit entry and checksum to ensure round-trip consistency
	// (json.RawMessage gets compacted when the AuditEntry is serialized to JSONL)
	afterCompact, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("audit: compact marshal state: %w", err)
	}
	hash := sha256.Sum256(afterCompact)
	checksum := hex.EncodeToString(hash[:])

	// Step 7: Build and append audit entry
	entry := AuditEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
		Command:     cmd,
		Path:        colonyStatePath,
		Before:      json.RawMessage(beforeJSON),
		After:       json.RawMessage(afterCompact),
		Summary:     summary,
		Checksum:    checksum,
		Destructive: destructive,
	}

	if appendErr := al.store.AppendJSONL(auditLogPath, entry); appendErr != nil {
		// Log but do NOT roll back the state write -- it has already succeeded atomically
		fmt.Fprintf(os.Stderr, "audit: failed to append changelog entry: %v\n", appendErr)
	}

	return nil
}

// ReadHistory reads the state changelog and returns the last `tail` entries.
// If tail <= 0, all entries are returned. Entries are returned in the order
// they were appended (oldest first).
func (al *AuditLogger) ReadHistory(tail int) ([]AuditEntry, error) {
	rawEntries, err := al.store.ReadJSONL(auditLogPath)
	if err != nil {
		// If the changelog file doesn't exist yet, return empty slice
		return nil, nil
	}

	var entries []AuditEntry
	for _, raw := range rawEntries {
		var entry AuditEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			// Skip malformed entries
			continue
		}
		entries = append(entries, entry)
	}

	if tail > 0 && len(entries) > tail {
		entries = entries[len(entries)-tail:]
	}

	return entries, nil
}

// GetLatestChecksum returns the checksum from the most recent audit entry.
// Returns an empty string if no entries exist.
func (al *AuditLogger) GetLatestChecksum() (string, error) {
	entries, err := al.ReadHistory(1)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", nil
	}
	return entries[0].Checksum, nil
}
