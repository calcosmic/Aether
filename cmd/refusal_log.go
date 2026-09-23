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

// refusalLogExcludedCommands are the NON-hook command paths already proven
// to write nothing -- TestStatusLineChangesNothingAndRepeatsItself,
// TestDirectRouteWritesNothing. appendRefusalToLog must honour that
// guarantee rather than quietly becoming a fifth writer. Held as one named
// var so TestRefusalLogIsSkippedForHookCommands iterates the exact set this
// function checks, rather than a re-typed copy that could drift from it.
//
// Every "hook-*" command is excluded structurally by
// refusalLogWriteExcludedForCommand below, never enumerated here one name
// at a time -- see that function's own doc comment for why (WR-03,
// 208-REVIEW.md).
//
// Named "Excluded" rather than "Skip" deliberately: TestSkipListDivergence
// (cmd/codex_colonize_test.go) guards against a *second* directory
// skip-list re-emerging alongside the one shared list in
// pkg/codegraph.ShouldSkipDir (fix(141-01)). This is a different concept
// entirely -- a fixed set of command paths excluded from local log writes,
// not a directory-walk skip list -- so it must not share that vocabulary.
var refusalLogExcludedCommands = map[string]bool{
	"status-line": true,
}

// refusalLogWriteExcludedForCommand reports whether command must never
// trigger a refusal-log write.
//
// Before this (WR-03, 208-REVIEW.md), the exclusion was a fixed
// enumeration of exactly the four commands proven never to mutate at the
// time it was written (hook-stop, hook-post-tool-use, hook-session-start,
// status-line). hook-pre-tool-use and hook-pre-compact
// (cmd/hook_cmds.go) call no refuse()/outputRefusal()/warnAndCarryOn()
// path today, so the gap was latent rather than active -- but nothing
// structurally stopped a future refusal being wired into either of those
// two hook commands, at which point appendRefusalToLog would have begun
// writing to refusals.jsonl during a hook invocation with no test catching
// the new mutation the way TestRefusalLogIsSkippedForHookCommands catches
// the four named ones.
//
// Every command whose own name starts with "hook-" now shares this
// exclusion structurally, so a future hook command never needs a new,
// easy-to-forget entry here. status-line is not a "hook-*" name, so it
// stays an explicit, named entry in refusalLogExcludedCommands above.
func refusalLogWriteExcludedForCommand(command string) bool {
	return strings.HasPrefix(command, "hook-") || refusalLogExcludedCommands[command]
}

// appendRefusalToLog appends one line to the local refusal log. It returns
// nothing and never propagates an error to the caller -- a refusal must
// still reach the owner's screen even when the log append itself fails, and
// the rendered screen must be byte-identical whether or not the append
// succeeded (TestRefusalLogAppendsAndNeverBlocks). It writes nothing when
// there is no project in this folder, and nothing for a command
// refusalLogWriteExcludedForCommand names.
func appendRefusalToLog(r refusal) {
	if store == nil {
		return
	}
	if refusalLogWriteExcludedForCommand(currentStreamingCommand) {
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
