package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// middenCanonicalPath is the single place the flat, canonical failure-record
// path string is declared. Both existing production writers (midden-write's
// RunE in cmd/midden_cmds.go, and spawn_budget.go's ceiling recorder) already
// write here; every reader in this package must converge on it via
// loadMiddenFile rather than hardcoding the string again.
const middenCanonicalPath = "midden.json"

// loadMiddenFile reads the canonical failure-record file via s. It is a thin
// wrapper around store.LoadJSON: every existing caller's err == nil /
// err != nil fallback-to-empty handling is preserved unchanged -- only the
// path this package resolves against is fixed.
func loadMiddenFile(s *storage.Store) (colony.MiddenFile, error) {
	var mf colony.MiddenFile
	err := s.LoadJSON(middenCanonicalPath, &mf)
	return mf, err
}

// appendMiddenEntry atomically appends one failure record, creating the file
// if it does not yet exist. Unlike the two existing production writers, which
// each do a separate Load-then-Save pair, this uses UpdateJSONAtomically so
// two appends issued in sequence never clobber each other.
func appendMiddenEntry(s *storage.Store, category, source, message string, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	// SYN-204-02 (204-03-PLAN.md Task 1, LEARN-01): the failure log's
	// provenance is runtime -- a midden entry records something the
	// running program observed, never a learning-pipeline decision.
	lineage := colony.NewMemoryRecordLineage(colony.MemoryProvenanceRuntime, source, now)
	entry := colony.MiddenEntry{
		ID:            fmt.Sprintf("midden_%d_%d", time.Now().UTC().Unix(), os.Getpid()),
		Timestamp:     now,
		Category:      category,
		Source:        source,
		Message:       message,
		Reviewed:      false,
		Tags:          tags,
		SchemaVersion: colony.CurrentMemorySchemaVersion,
		Lineage:       &lineage,
	}

	var mf colony.MiddenFile
	return s.UpdateJSONAtomically(middenCanonicalPath, &mf, func() error {
		if mf.Entries == nil {
			mf.Entries = []colony.MiddenEntry{}
		}
		if mf.Version == "" {
			mf.Version = "1.0.0"
		}
		mf.Entries = append(mf.Entries, entry)
		return nil
	})
}
