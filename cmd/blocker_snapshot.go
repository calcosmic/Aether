package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// blockerSnapshot is the live, deterministic projection of unresolved blocker
// flags. Existing blockers are evidence to report; comparison with a later
// snapshot decides whether the blocker situation became less safe.
type blockerSnapshot struct {
	Count          int      `json:"blockers"`
	IDs            []string `json:"blocker_ids"`
	EscalatedCount int      `json:"escalated_blockers"`
	EscalatedIDs   []string `json:"escalated_blocker_ids"`
}

// blockerSnapshotComparison records only the two changes that overnight policy
// treats as meaningful: more blocker rows, or newly observed escalation
// evidence. Ordinary ID churn at the same count is deliberately diagnostic
// evidence rather than a stop condition.
type blockerSnapshotComparison struct {
	CountIncreased    bool     `json:"count_increased"`
	EscalationAdded   bool     `json:"escalation_added"`
	AddedEscalatedIDs []string `json:"added_escalated_ids"`
}

// readBlockerSnapshot reads only the live flag store and projects unresolved
// Type=blocker records. A missing current file may use the legacy flags file;
// every other storage failure is unavailable truth, never an empty snapshot.
func readBlockerSnapshot(s *storage.Store) (blockerSnapshot, error) {
	snapshot := blockerSnapshot{
		IDs:          []string{},
		EscalatedIDs: []string{},
	}
	flags, _, err := readCanonicalBlockerFlags(s)
	if err != nil {
		return snapshot, err
	}

	ids := make(map[string]struct{})
	escalatedIDs := make(map[string]struct{})
	for _, flag := range flags.Decisions {
		if flag.Resolved || flag.Type != "blocker" {
			continue
		}
		snapshot.Count++
		id := strings.TrimSpace(flag.ID)
		if id != "" {
			ids[id] = struct{}{}
		}
		if flag.Source != "escalation" {
			continue
		}
		snapshot.EscalatedCount++
		if id != "" {
			escalatedIDs[id] = struct{}{}
		}
	}

	snapshot.IDs = sortedStringSet(ids)
	snapshot.EscalatedIDs = sortedStringSet(escalatedIDs)
	return snapshot, nil
}

func readCanonicalBlockerFlags(s *storage.Store) (colony.FlagsFile, bool, error) {
	if s == nil {
		return colony.FlagsFile{}, false, fmt.Errorf("blocker truth store is unavailable")
	}
	flags, err := loadBlockerFlagsPath(s, "pending-decisions.json")
	if err == nil {
		return flags, true, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return colony.FlagsFile{}, false, err
	}
	flags, err = loadBlockerFlagsPath(s, "flags.json")
	if err == nil {
		return flags, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return colony.FlagsFile{}, false, nil
	}
	return colony.FlagsFile{}, false, err
}

func loadBlockerFlagsPath(s *storage.Store, relativePath string) (colony.FlagsFile, error) {
	fullPath := filepath.Join(s.BasePath(), relativePath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return colony.FlagsFile{}, fmt.Errorf("blocker truth %s: %w", relativePath, err)
	}
	if !info.Mode().IsRegular() {
		return colony.FlagsFile{}, fmt.Errorf("blocker truth %s: expected a regular file", relativePath)
	}
	var flags colony.FlagsFile
	if err := s.LoadJSON(relativePath, &flags); err != nil {
		return colony.FlagsFile{}, fmt.Errorf("blocker truth %s: %w", relativePath, err)
	}
	return flags, nil
}

// blockerSnapshotErrorDetail produces the bounded, path-safe diagnostic used
// by command and report surfaces. The original error remains available to
// callers for control flow, while users never receive an absolute data path.
func blockerSnapshotErrorDetail(s *storage.Store, err error) string {
	if err == nil {
		return ""
	}
	detail := err.Error()
	if s != nil && strings.TrimSpace(s.BasePath()) != "" {
		detail = strings.ReplaceAll(detail, s.BasePath(), ".aether/data")
	}
	detail = codex.SanitizeWorkerDiagnosticOutput(detail)
	return compactActionText(detail, 240)
}

func compareBlockerSnapshots(before, after blockerSnapshot) blockerSnapshotComparison {
	beforeEscalated := make(map[string]struct{}, len(before.EscalatedIDs))
	for _, id := range before.EscalatedIDs {
		beforeEscalated[id] = struct{}{}
	}

	addedEscalated := make(map[string]struct{})
	for _, id := range after.EscalatedIDs {
		if _, existed := beforeEscalated[id]; !existed {
			addedEscalated[id] = struct{}{}
		}
	}
	addedIDs := sortedStringSet(addedEscalated)
	return blockerSnapshotComparison{
		CountIncreased:    after.Count > before.Count,
		EscalationAdded:   after.EscalatedCount > before.EscalatedCount || len(addedIDs) > 0,
		AddedEscalatedIDs: addedIDs,
	}
}

func sortedStringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

// addBlockerSnapshotFields keeps command and status JSON byte-for-field aligned
// without asking either adapter to understand blocker storage semantics.
func addBlockerSnapshotFields(result map[string]interface{}, snapshot blockerSnapshot) {
	result["blockers"] = snapshot.Count
	result["blocker_ids"] = snapshot.IDs
	result["escalated_blockers"] = snapshot.EscalatedCount
	result["escalated_blocker_ids"] = snapshot.EscalatedIDs
}
