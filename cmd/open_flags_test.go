package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// openFlagsMixedFixture seeds one blocker, one issue, one note, one resolved
// note (must not count), and one clarification row (must not count as any of
// the three) -- the exact shape TestOpenFlagsHaveOneCountingRule needs to
// prove every reader agrees.
func openFlagsMixedFixture() []colony.FlagEntry {
	return []colony.FlagEntry{
		{ID: "b1", Type: "blocker", Description: "blocking work"},
		{ID: "i1", Type: "issue", Description: "an issue"},
		{ID: "n1", Type: "note", Description: "a note for later"},
		{ID: "n2", Type: "note", Description: "already handled", Resolved: true},
		{ID: "c1", Type: "clarification", Description: "an answered question"},
	}
}

// TestOpenFlagsHaveOneCountingRule proves status, flag-list, and reconcile
// all read the same pending-decisions.json file and report the same numbers
// -- a clarification row counts as none of blocker/issue/note anywhere.
func TestOpenFlagsHaveOneCountingRule(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	s, root := setupTestStore(t)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	t.Cleanup(func() { store = nil })
	t.Setenv("AETHER_ROOT", root)
	store = s
	writeTestFlags(t, openFlagsMixedFixture()...)

	c := classifyOpenFlags(readTestFlags(t))
	if len(c.Blockers) != 1 || len(c.Issues) != 1 || len(c.Notes) != 1 {
		t.Fatalf("classifyOpenFlags = %+v, want 1 blocker, 1 issue, 1 note", c)
	}

	// status.go's countStatusNonBlockerFlags
	issues, notes := countStatusNonBlockerFlags(s)
	if issues != 1 || notes != 1 {
		t.Errorf("status counting: issues=%d notes=%d, want 1 and 1", issues, notes)
	}

	// flags.go's renderFlagsTable summary line -- the renderer the real
	// `aether flags` command calls to build its visual output.
	flagsOut := renderFlagsTable(readTestFlags(t))
	if !bytesContainsAll(flagsOut, "1 blocker(s)", "1 issue(s)", "1 note(s)") {
		t.Errorf("flag-list summary does not report 1/1/1:\n%s", flagsOut)
	}

	// reconcile.go's inspectReconcileFlags, through the real report builder.
	report := buildReconcileReport(root, time.Now().UTC())
	if report.Flags.Blockers != 1 || report.Flags.Issues != 1 || report.Flags.Notes != 1 {
		t.Errorf("reconcile flag summary = %+v, want blockers=1 issues=1 notes=1", report.Flags)
	}
}

func bytesContainsAll(haystack string, needles ...string) bool {
	for _, n := range needles {
		if !contains(haystack, n) {
			return false
		}
	}
	return true
}

// TestFlagResolveReportsTheResolvedRowsTimestamp proves flag-resolve reports
// the timestamp of the row it actually resolved, not always row 0 -- the bug
// where resolving the second-or-later flag in the file reported the first
// flag's (possibly empty, possibly unrelated) resolved_at value.
func TestFlagResolveReportsTheResolvedRowsTimestamp(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	s, root := setupTestStore(t)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	t.Cleanup(func() { store = nil })
	t.Setenv("AETHER_ROOT", root)
	store = s
	writeTestFlags(t,
		colony.FlagEntry{ID: "first", Type: "note", Description: "first row, left alone"},
		colony.FlagEntry{ID: "second", Type: "issue", Description: "the one actually resolved"},
	)

	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"flag-resolve", "--id", "second", "--message", "fixed it"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("flag-resolve: %v", err)
	}
	result := parseEnvelope(t, buf.String())["result"].(map[string]interface{})
	timestamp, _ := result["timestamp"].(string)
	if timestamp == "" {
		t.Fatalf("flag-resolve reported an empty timestamp: %#v", result)
	}

	for _, flag := range readTestFlags(t) {
		if flag.ID == "second" {
			if flag.ResolvedAt != timestamp {
				t.Errorf("reported timestamp %q does not match the resolved row's own resolved_at %q", timestamp, flag.ResolvedAt)
			}
		}
		if flag.ID == "first" && flag.ResolvedAt != "" {
			t.Errorf("the untouched first row unexpectedly gained a resolved_at: %q", flag.ResolvedAt)
		}
	}
}

// TestStatusListsOpenFlagsByTitle proves `aether status` shows open flags by
// title, headed by kind (blocking work, issues, for later) -- not just the
// bare counts it showed before.
func TestStatusListsOpenFlagsByTitle(t *testing.T) {
	saveGlobals(t)
	s, root := setupTestStore(t)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	store = s
	writeTestFlags(t,
		colony.FlagEntry{ID: "b1", Type: "blocker", Description: "the release gate is red"},
		colony.FlagEntry{ID: "i1", Type: "issue", Description: "duplicate rows in the export"},
		colony.FlagEntry{ID: "n1", Type: "note", Description: "revisit the caching layer later", Acknowledged: true},
	)

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	result := buildStatusResult(state, s)
	output := renderDashboard(state, s, result)

	for _, want := range []string{
		"the release gate is red",
		"duplicate rows in the export",
		"revisit the caching layer later",
		"parked",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("status screen missing %q:\n%s", want, output)
		}
	}
}

// TestStatusFlagListIsCappedAndCounted proves each group shows at most five
// titles, then folds the rest into "and N more".
func TestStatusFlagListIsCappedAndCounted(t *testing.T) {
	saveGlobals(t)
	s, root := setupTestStore(t)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	store = s
	var entries []colony.FlagEntry
	for i := 0; i < 8; i++ {
		entries = append(entries, colony.FlagEntry{
			ID:          fmt.Sprintf("i%d", i),
			Type:        "issue",
			Description: fmt.Sprintf("issue number %d", i),
		})
	}
	writeTestFlags(t, entries...)

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	result := buildStatusResult(state, s)
	output := renderDashboard(state, s, result)

	if !strings.Contains(output, "and 3 more") {
		t.Errorf("status screen did not cap the issue list and fold the rest into a count:\n%s", output)
	}
	if strings.Contains(output, "issue number 5") {
		t.Errorf("status screen showed a sixth title past the cap:\n%s", output)
	}
}

// TestArchivedProjectStatusListsOpenFlags proves the archived-project screen
// shows the same open-items section as the live dashboard -- an owner's
// still-open note must remain visible even after the project is archived.
func TestArchivedProjectStatusListsOpenFlags(t *testing.T) {
	saveGlobals(t)
	s, root := setupTestStore(t)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	store = s
	writeTestFlags(t, colony.FlagEntry{ID: "n1", Type: "note", Description: "deal with this after the archive"})

	output := renderArchivedProjectStatusVisual("archive-1", nextAction{})
	if !strings.Contains(output, "deal with this after the archive") {
		t.Errorf("archived-project status screen dropped an open note:\n%s", output)
	}
}

// TestWriteCarriedFlagsFileIsAtomicAndRefusesNonRegularFiles proves
// writeCarriedFlagsFile writes through a temp-file-plus-rename (so a
// concurrent reader can never observe a half-written file) and refuses to
// write through anything that is not an ordinary file -- the same guard
// writeProjectChangelogEntry uses (cmd/project_changelog.go), since a
// directory or a symlink at that path is not ours to replace.
func TestWriteCarriedFlagsFileIsAtomicAndRefusesNonRegularFiles(t *testing.T) {
	dataDir := t.TempDir()

	kept := []colony.FlagEntry{{ID: "n1", Type: "note", Description: "a carried note"}}
	if err := writeCarriedFlagsFile(dataDir, colony.FlagsFile{Version: "1.0"}, kept); err != nil {
		t.Fatalf("writeCarriedFlagsFile: %v", err)
	}

	path := filepath.Join(dataDir, pendingDecisionsFile)
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatalf("read data dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".pending-decisions-") && strings.HasSuffix(entry.Name(), ".tmp") {
			t.Errorf("a temp file was left behind after a successful write: %s", entry.Name())
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	var ff colony.FlagsFile
	if err := json.Unmarshal(raw, &ff); err != nil {
		t.Fatalf("decode written file: %v", err)
	}
	if len(ff.Decisions) != 1 || ff.Decisions[0].Description != "a carried note" {
		t.Errorf("written file = %+v, want the one carried note", ff.Decisions)
	}

	t.Run("refuses a directory", func(t *testing.T) {
		dirDataDir := t.TempDir()
		dirPath := filepath.Join(dirDataDir, pendingDecisionsFile)
		if err := os.Mkdir(dirPath, 0755); err != nil {
			t.Fatalf("seed a directory at the target path: %v", err)
		}
		err := writeCarriedFlagsFile(dirDataDir, colony.FlagsFile{Version: "1.0"}, kept)
		if err == nil {
			t.Fatal("writeCarriedFlagsFile did not refuse a directory at the target path")
		}
	})
}

// TestStatusListsOpenFlagsWithNoProject covers the folder that has never had
// a project: a quick job or the owner can raise a flag there, so the status
// screen must list it (found in the 1.0.86 real-use check: the note was saved
// and showed under the flags command, but not on status).
func TestStatusListsOpenFlagsWithNoProject(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	root := t.TempDir()
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf
	rootCmd.SetArgs([]string{"flag", "come back and add a farewell message", "--type", "note"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("flag: %v", err)
	}

	resetRootCmd(t)
	buf.Reset()
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "come back and add a farewell message") {
		t.Fatalf("status with no project does not list the open note:\n%s", output)
	}
}
