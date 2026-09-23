package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRefusalLogAppendsAndNeverBlocks renders a refusal in a temporary
// store and asserts exactly one JSON line is appended carrying a non-empty
// id, next_command and recorded_at; it then makes the store path
// unwritable, renders again, and asserts the rendered screen is
// byte-identical to the writable case and the function returns normally.
func TestRefusalLogAppendsAndNeverBlocks(t *testing.T) {
	saveGlobals(t)
	repo := bindCommandTestRepository(t)
	r := refuse("colonize-finalize-missing-timestamp")

	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	var buf bytes.Buffer
	stderr = &buf
	outputRefusal(r)
	writableOutput := buf.String()

	entries := refusalLogEntries(10)
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 log entry, got %d: %+v", len(entries), entries)
	}
	if entries[0].ID == "" || entries[0].NextCommand == "" || entries[0].RecordedAt == "" {
		t.Fatalf("expected non-empty id/next_command/recorded_at, got: %+v", entries[0])
	}

	// Make the store's data directory unwritable so the append fails, and
	// prove the rendered screen and the function's return are unaffected.
	if err := os.Chmod(repo.DataDir, 0500); err != nil {
		t.Fatalf("chmod data dir read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(repo.DataDir, 0755) })

	var buf2 bytes.Buffer
	stderr = &buf2
	outputRefusal(r)
	unwritableOutput := buf2.String()

	if writableOutput != unwritableOutput {
		t.Errorf("rendered screen changed when the log append failed:\nwritable:   %q\nunwritable: %q", writableOutput, unwritableOutput)
	}
}

// TestRefusalLogIsSkippedForHookCommands iterates refusalLogExcludedCommands
// itself (not a re-typed copy), plus two known "hook-*" commands and one
// deliberately never-before-registered "hook-*" name, and asserts no file
// is created under the store for any of them.
//
// The never-before-registered case (WR-03, 208-REVIEW.md) is the point:
// before refusalLogWriteExcludedForCommand existed, the exclusion was a
// fixed enumeration of exactly the commands known at the time -- a future
// hook command would have started writing to refusals.jsonl with nothing
// catching it. This proves the exclusion is now structural (every
// "hook-*" name), not a list that has to remember to grow.
func TestRefusalLogIsSkippedForHookCommands(t *testing.T) {
	saveGlobals(t)
	repo := bindCommandTestRepository(t)

	commands := map[string]bool{}
	for command := range refusalLogExcludedCommands {
		commands[command] = true
	}
	commands["hook-stop"] = true
	commands["hook-post-tool-use"] = true
	commands["hook-session-start"] = true
	// hook-pre-tool-use and hook-pre-compact call no refuse(...) path
	// today (see refusalLogWriteExcludedForCommand's doc comment), and a
	// wholly synthetic name proves the exclusion is not a lookup table at
	// all -- any future "hook-*" command is covered the same way.
	commands["hook-pre-tool-use"] = true
	commands["hook-pre-compact"] = true
	commands["hook-a-command-nobody-has-registered-yet"] = true

	for command := range commands {
		t.Run(command, func(t *testing.T) {
			currentStreamingCommand = command
			t.Cleanup(func() { currentStreamingCommand = "" })

			appendRefusalToLog(refuse("colonize-finalize-missing-timestamp"))

			if _, err := os.Stat(filepath.Join(repo.DataDir, refusalLogPath)); err == nil {
				t.Errorf("expected no refusal log to be written for command %q", command)
			} else if !os.IsNotExist(err) {
				t.Fatalf("unexpected error checking for refusal log: %v", err)
			}
		})
	}
}

// TestRefusalLogIsNotSkippedForAnOrdinaryCommand is the guard on the guard:
// proves refusalLogWriteExcludedForCommand can return false, and that an
// ordinary (non-hook, non-status-line) command really does write a log
// entry -- without this, TestRefusalLogIsSkippedForHookCommands passing
// would prove nothing, since a function that always returns true would
// pass it too.
func TestRefusalLogIsNotSkippedForAnOrdinaryCommand(t *testing.T) {
	saveGlobals(t)
	repo := bindCommandTestRepository(t)

	if refusalLogWriteExcludedForCommand("continue") {
		t.Fatal("expected an ordinary command like \"continue\" to not be excluded")
	}

	currentStreamingCommand = "continue"
	t.Cleanup(func() { currentStreamingCommand = "" })
	appendRefusalToLog(refuse("colonize-finalize-missing-timestamp"))

	if _, err := os.Stat(filepath.Join(repo.DataDir, refusalLogPath)); err != nil {
		t.Fatalf("expected a refusal log to be written for an ordinary command, stat err = %v", err)
	}
}

// TestRefusalLogWritesNothingWithoutAProject asserts appendRefusalToLog is
// silent (never panics, never creates a file) when there is no project set
// up in this folder.
func TestRefusalLogWritesNothingWithoutAProject(t *testing.T) {
	saveGlobals(t)
	store = nil
	appendRefusalToLog(refuse("colonize-finalize-missing-timestamp"))
	if entries := refusalLogEntries(10); len(entries) != 0 {
		t.Errorf("expected no entries with no store, got %+v", entries)
	}
}

// TestFriendlyErrorSpecificityIsPatternLengthNotTableOrder proves the two
// folded-in rows that share a substring relationship (a general "json:"
// pattern and the more specific "invalid charter JSON" pattern) still
// resolve to the more specific row, exactly as the pre-fold errorPatternMap
// order guaranteed -- now via pattern length rather than table position,
// since the table itself must stay id-sorted.
func TestFriendlyErrorSpecificityIsPatternLengthNotTableOrder(t *testing.T) {
	entry, ok := friendlyErrorForPattern("invalid charter JSON: invalid character 'o' in literal null")
	if !ok {
		t.Fatal("expected a match")
	}
	if strings.Contains(strings.ToLower(entry.Explanation), "corrupted") {
		t.Errorf("expected the more specific invalid-charter-json row, got the generic corrupted-colony-data row: %s", entry.Explanation)
	}
}
