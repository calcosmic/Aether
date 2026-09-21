package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// TestArchivedProjectKeepsOnlyCarriedFlags is the coordinator's HOLE fix
// (review round after C3): entomb must not leave the whole
// pending-decisions.json live after a project is archived. A blocker or a
// clarification row surviving live past entomb reproduces the exact
// 1.0.83 dead-end this repo already fixed once
// (cmd/entomb_archived_shell_test.go) -- an archived shell whose closing
// decision reads as something other than "start a new project" -- and lets
// a resolved/unresolved clarification keep reaching a worker's prompt as
// CLARIFIED INTENT for anything run between projects.
//
// The chamber archive still holds every row entomb ever saw (full, digest-
// verified copy); the live file afterwards holds only the carried subset --
// the exact same filter `aether init` applies (filterCarriedFlags).
func TestArchivedProjectKeepsOnlyCarriedFlags(t *testing.T) {
	downstream, _ := runRealLifecycleToSealForTest(t)
	dataDir := filepath.Join(downstream, ".aether", "data")

	// An open note and an open issue -- must carry forward.
	rootCmd.SetArgs([]string{"flag-add", "--type", "note", "--title", "come back to the caching layer", "--severity", "low"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("flag-add note: %v", err)
	}
	rootCmd.SetArgs([]string{"flag-add", "--type", "issue", "--title", "the export has duplicate rows", "--severity", "high"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("flag-add issue: %v", err)
	}
	// A blocker added after seal, before entomb -- the runtime accepts this
	// (proven by TestOpenNotesSurviveIntoTheNextProject's own fixture using
	// the same sequence); it must NOT carry forward live.
	rootCmd.SetArgs([]string{"flag-add", "--type", "blocker", "--title", "the release gate is red", "--severity", "critical"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("flag-add blocker: %v", err)
	}
	// A resolved clarification -- the shape a real /ant-discuss answer
	// leaves behind. Must not carry forward live, and must never render as
	// CLARIFIED INTENT once it belongs to an archived project.
	rootCmd.SetArgs([]string{"pending-decision-add", "--type", "clarification", "--description", "Should the export dedupe by ID? => yes, dedupe by ID"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pending-decision-add clarification: %v", err)
	}
	var pd PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &pd); err != nil {
		t.Fatalf("load pending decisions to resolve the clarification: %v", err)
	}
	resolvedAny := false
	for i := range pd.Decisions {
		if pd.Decisions[i].Type == "clarification" && strings.Contains(pd.Decisions[i].Description, "dedupe by ID") {
			pd.Decisions[i].Resolved = true
			pd.Decisions[i].Resolution = "yes, dedupe by ID"
			resolvedAny = true
		}
	}
	if !resolvedAny {
		t.Fatal("fixture guard: no clarification row found to resolve")
	}
	if err := store.SaveJSON(pendingDecisionsFile, pd); err != nil {
		t.Fatalf("save resolved clarification: %v", err)
	}

	// Snapshot everything that was live before entomb, for the chamber
	// assertion below (the chamber must hold the FULL pre-entomb set).
	var beforeEntomb colony.FlagsFile
	if err := store.LoadJSON(pendingDecisionsFile, &beforeEntomb); err != nil {
		t.Fatalf("load pre-entomb pending-decisions.json: %v", err)
	}
	preEntombCount := len(beforeEntomb.Decisions)
	if preEntombCount < 4 {
		t.Fatalf("fixture guard: pre-entomb pending-decisions.json has %d rows, want at least 4 (seal may add its own clarification row too): %+v", preEntombCount, beforeEntomb.Decisions)
	}

	rootCmd.SetArgs([]string{"entomb", "--confirm"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("entomb: %v", err)
	}

	// The live file holds exactly the note and the issue.
	pdPath := filepath.Join(dataDir, pendingDecisionsFile)
	raw, err := os.ReadFile(pdPath)
	if err != nil {
		t.Fatalf("pending-decisions.json did not survive entomb even though carried rows existed: %v", err)
	}
	var live colony.FlagsFile
	if err := json.Unmarshal(raw, &live); err != nil {
		t.Fatalf("decode live pending-decisions.json after entomb: %v", err)
	}
	if len(live.Decisions) != 2 {
		t.Fatalf("live pending-decisions.json after entomb has %d rows, want exactly 2 (the note and the issue): %+v", len(live.Decisions), live.Decisions)
	}
	liveByDescription := map[string]colony.FlagEntry{}
	for _, flag := range live.Decisions {
		liveByDescription[flag.Description] = flag
	}
	if _, ok := liveByDescription["come back to the caching layer"]; !ok {
		t.Errorf("the open note did not survive live past entomb: %+v", live.Decisions)
	}
	if _, ok := liveByDescription["the export has duplicate rows"]; !ok {
		t.Errorf("the open issue did not survive live past entomb: %+v", live.Decisions)
	}
	if _, ok := liveByDescription["the release gate is red"]; ok {
		t.Error("the blocker survived live past entomb")
	}
	for _, flag := range live.Decisions {
		if flag.Type == "clarification" {
			t.Errorf("a clarification row survived live past entomb: %+v", flag)
		}
	}

	// The chamber copy holds all four original rows.
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load post-entomb state: %v", err)
	}
	if state.ArchiveReference == nil {
		t.Fatal("post-entomb state has no ArchiveReference -- cannot locate the chamber")
	}
	chamberPath := filepath.Join(downstream, filepath.FromSlash(state.ArchiveReference.Path))
	chamberDir := filepath.Dir(chamberPath)
	archivedRaw, err := os.ReadFile(filepath.Join(chamberDir, pendingDecisionsFile))
	if err != nil {
		t.Fatalf("entomb did not archive pending-decisions.json into the chamber: %v", err)
	}
	var archived colony.FlagsFile
	if err := json.Unmarshal(archivedRaw, &archived); err != nil {
		t.Fatalf("decode archived pending-decisions.json: %v", err)
	}
	if len(archived.Decisions) != preEntombCount {
		t.Fatalf("archived pending-decisions.json has %d rows, want all %d original rows: %+v", len(archived.Decisions), preEntombCount, archived.Decisions)
	}

	// The closing decision is "initialize", not "resume" -- the exact
	// dead-end the 1.0.83 fix closed for a plain archived shell must not
	// reopen because a blocker or clarification is still live.
	answer := resolveNextAction(loadNextActionInputForCommand(""))
	if answer.Projection == nil || answer.Projection.NextAction.ID != "initialize" {
		gotID := ""
		if answer.Projection != nil {
			gotID = answer.Projection.NextAction.ID
		}
		t.Errorf("closing action id = %q, want %q\nreason: %s", gotID, "initialize", answer.Recommendation)
	}

	// Colony-prime output contains no CLARIFIED INTENT from the old
	// project's resolved clarification.
	rendered := clarifiedIntentPromptRenderResult()
	for _, line := range rendered.Lines {
		if contains(line, "dedupe by ID") {
			t.Errorf("the archived project's resolved clarification leaked into CLARIFIED INTENT: %q", line)
		}
	}
	if len(rendered.Lines) != 0 {
		t.Errorf("CLARIFIED INTENT is not empty after entomb: %v", rendered.Lines)
	}
}
