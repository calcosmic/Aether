package cmd

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// refusalLogPath is where every refusal the owner sees is recorded locally,
// one JSON line per refusal -- the record `aether report` (Task 3) reads to
// build a sendable bundle.
const refusalLogPath = "refusals.jsonl"

// refusalLogSchemaVersion lets a future reader tell an old record's shape
// from a new one without guessing.
const refusalLogSchemaVersion = 1

// refusalLogEntry is one recorded refusal. Command is the cobra command
// path that refused, read from the resolved command at the moment of
// dispatch (currentStreamingCommand, cmd/codex_visuals.go) -- never
// re-typed at each refusal site.
type refusalLogEntry struct {
	SchemaVersion int    `json:"schema_version"`
	RecordedAt    string `json:"recorded_at"`
	ID            string `json:"id"`
	What          string `json:"what"`
	NextCommand   string `json:"next_command"`
	ProtectsWork  bool   `json:"protects_work"`
	Command       string `json:"command"`
}

// refusalLogSkippedCommands are the command paths already proven to write
// nothing -- TestStatusLineChangesNothingAndRepeatsItself, TestDirectRouteWritesNothing,
// TestStopHookScreenCheckDoesNotMutate. appendRefusalToLog must honour those
// guarantees rather than quietly becoming a fifth writer. Held as one named
// var so TestRefusalLogIsSkippedForHookCommands iterates the exact set this
// function checks, rather than a re-typed copy that could drift from it.
var refusalLogSkippedCommands = map[string]bool{
	"hook-stop":          true,
	"hook-post-tool-use": true,
	"hook-session-start": true,
	"status-line":        true,
}

// appendRefusalToLog appends one line to the local refusal log. It returns
// nothing and never propagates an error to the caller -- a refusal must
// still reach the owner's screen even when the log append itself fails, and
// the rendered screen must be byte-identical whether or not the append
// succeeded (TestRefusalLogAppendsAndNeverBlocks). It writes nothing when
// there is no project in this folder, and nothing for the command paths
// named above.
func appendRefusalToLog(r refusal) {
	if store == nil {
		return
	}
	if refusalLogSkippedCommands[currentStreamingCommand] {
		return
	}
	entry := refusalLogEntry{
		SchemaVersion: refusalLogSchemaVersion,
		RecordedAt:    time.Now().UTC().Format(time.RFC3339),
		ID:            r.ID,
		What:          strings.TrimSpace(r.What),
		NextCommand:   strings.TrimSpace(r.NextCommand),
		ProtectsWork:  r.ProtectsWork,
		Command:       currentStreamingCommand,
	}
	_ = store.AppendJSONL(refusalLogPath, entry)
}

// refusalLogEntries reads up to `limit` refusal log entries, newest first.
// Two entries carrying the same recorded_at keep the order they were
// written in (a stable sort, not a reverse) -- the same ordering contract
// the report bundle (Task 3) promises for every record it lists.
func refusalLogEntries(limit int) []refusalLogEntry {
	if store == nil || limit <= 0 {
		return nil
	}
	raw, err := store.ReadJSONL(refusalLogPath)
	if err != nil {
		return nil
	}
	entries := make([]refusalLogEntry, 0, len(raw))
	for _, line := range raw {
		var entry refusalLogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].RecordedAt > entries[j].RecordedAt
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries
}
