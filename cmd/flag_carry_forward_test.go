package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// seedFlagCarryForwardFixture drives the real flag-add and pending-decision-
// add commands against the sealed downstream repo's own store: one open
// note, one open issue, one open blocker, and one resolved clarification
// (the shape a real /ant-discuss answer leaves behind). It returns nothing;
// callers read the file back through the real commands under test.
func seedFlagCarryForwardFixture(t *testing.T) {
	t.Helper()

	rootCmd.SetArgs([]string{"flag-add", "--type", "note", "--title", "come back to the caching layer", "--severity", "low"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("flag-add note: %v", err)
	}
	rootCmd.SetArgs([]string{"flag-add", "--type", "issue", "--title", "the export has duplicate rows", "--severity", "high"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("flag-add issue: %v", err)
	}
	rootCmd.SetArgs([]string{"flag-add", "--type", "blocker", "--title", "the release gate is red", "--severity", "critical"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("flag-add blocker: %v", err)
	}
	rootCmd.SetArgs([]string{"pending-decision-add", "--type", "clarification", "--description", "Should the export dedupe by ID? => yes, dedupe by ID"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pending-decision-add clarification: %v", err)
	}
}

// TestOpenNotesSurviveIntoTheNextProject drives the real init -> plan ->
// build -> continue -> seal -> entomb -> init flow (runRealLifecycleToSealForTest
// plus the real entomb and init commands) and proves an owner's still-open
// note and issue survive into the freshly-initialized next project, with
// their phase number cleared, while a blocker and a resolved clarification
// do not.
func TestOpenNotesSurviveIntoTheNextProject(t *testing.T) {
	downstream, _ := runRealLifecycleToSealForTest(t)
	dataDir := filepath.Join(downstream, ".aether", "data")

	seedFlagCarryForwardFixture(t)

	rootCmd.SetArgs([]string{"entomb", "--confirm"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("entomb: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "--confirm-reinit", "the next project"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	var ff colony.FlagsFile
	pdPath := filepath.Join(dataDir, "pending-decisions.json")
	raw, err := os.ReadFile(pdPath)
	if err != nil {
		t.Fatalf("pending-decisions.json did not survive re-init even though open items existed: %v", err)
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		t.Fatalf("decode pending-decisions.json after re-init: %v", err)
	}

	byDescription := map[string]colony.FlagEntry{}
	for _, flag := range ff.Decisions {
		byDescription[flag.Description] = flag
	}

	note, ok := byDescription["come back to the caching layer"]
	if !ok {
		t.Fatalf("the open note did not survive re-init: %+v", ff.Decisions)
	}
	if note.Resolved {
		t.Errorf("the carried note was marked resolved: %+v", note)
	}
	if note.Phase != nil {
		t.Errorf("the carried note kept its old phase number %v; it should be cleared", *note.Phase)
	}

	issue, ok := byDescription["the export has duplicate rows"]
	if !ok {
		t.Fatalf("the open issue did not survive re-init: %+v", ff.Decisions)
	}
	if issue.Phase != nil {
		t.Errorf("the carried issue kept its old phase number %v; it should be cleared", *issue.Phase)
	}

	if _, ok := byDescription["the release gate is red"]; ok {
		t.Error("the blocker survived re-init; a blocker would stop the new project's first check")
	}
	for _, flag := range ff.Decisions {
		if flag.Type == "clarification" {
			t.Errorf("a clarification row survived re-init: %+v", flag)
		}
	}
}

// TestArchiveHoldsTheFinishedProjectsFlags proves entomb archives a full,
// unfiltered copy of pending-decisions.json into the chamber -- the finished
// project's own record of every flag it ever raised, blockers and
// clarifications included, not just the two kinds that carry forward.
func TestArchiveHoldsTheFinishedProjectsFlags(t *testing.T) {
	downstream, _ := runRealLifecycleToSealForTest(t)

	seedFlagCarryForwardFixture(t)

	rootCmd.SetArgs([]string{"entomb", "--confirm"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("entomb: %v", err)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load post-entomb state: %v", err)
	}
	if state.ArchiveReference == nil {
		t.Fatal("post-entomb state has no ArchiveReference -- cannot locate the chamber")
	}
	chamberPath := filepath.Join(downstream, filepath.FromSlash(state.ArchiveReference.Path))
	chamberDir := filepath.Dir(chamberPath)
	archivedPD := filepath.Join(chamberDir, "pending-decisions.json")

	raw, err := os.ReadFile(archivedPD)
	if err != nil {
		t.Fatalf("entomb did not archive pending-decisions.json into the chamber: %v", err)
	}
	var archived colony.FlagsFile
	if err := json.Unmarshal(raw, &archived); err != nil {
		t.Fatalf("decode archived pending-decisions.json: %v", err)
	}

	found := map[string]bool{}
	for _, flag := range archived.Decisions {
		found[flag.Description] = true
	}
	for _, want := range []string{
		"come back to the caching layer",
		"the export has duplicate rows",
		"the release gate is red",
	} {
		if !found[want] {
			t.Errorf("the archive is missing %q -- entomb must hold the finished project's own full record: %+v", want, archived.Decisions)
		}
	}
}

// TestCarriedFlagsNeverReachWorkerPromptsAsIntent proves a carried-forward
// note or issue is never rendered into the CLARIFIED INTENT section a
// worker's prompt reads -- that section is reserved for resolved
// clarifications from the discussion the owner actually had, never for the
// owner's own open tracking items.
func TestCarriedFlagsNeverReachWorkerPromptsAsIntent(t *testing.T) {
	_, _ = runRealLifecycleToSealForTest(t)

	seedFlagCarryForwardFixture(t)

	rootCmd.SetArgs([]string{"entomb", "--confirm"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("entomb: %v", err)
	}
	rootCmd.SetArgs([]string{"init", "--confirm-reinit", "the next project"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	rendered := clarifiedIntentPromptRenderResult()
	for _, line := range rendered.Lines {
		if contains(line, "come back to the caching layer") || contains(line, "the export has duplicate rows") {
			t.Errorf("a carried-forward note/issue leaked into CLARIFIED INTENT: %q", line)
		}
	}
}
