package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// This file proves 203-14's own guarantees: one projection covers every
// governed descendant and every follow-on edge (Task 1), liveness is
// inline in the working session the owner is already in (Task 2), and the
// end-of-run family tree never weakens immutable attempt authorization
// while agreeing with the closing cost block (Task 3).

// --- test helpers ---

func seedRecruitmentManifestChild(t *testing.T, childName, parentName, workspace, adapterKind string) {
	t.Helper()
	var file recruitmentManifestFile
	_ = store.LoadJSON(recruitmentManifestPath, &file)
	file.Entries = append(file.Entries, recruitmentManifestRecord{
		SchemaVersion: recruitmentManifestSchemaVersion,
		IntentID:      "intent-" + childName,
		ChildName:     childName,
		ParentName:    parentName,
		Workspace:     workspace,
		AdapterKind:   adapterKind,
		State:         recruitmentManifestStateDispatched,
	})
	if err := store.SaveJSON(recruitmentManifestPath, file); err != nil {
		t.Fatalf("seed recruitment manifest: %v", err)
	}
}

// storeFileFingerprint is one file's identity for the read-only proof
// below: its size and modification time. Mirrors
// cmd/recruitment_result_test.go's TestReplayPerformsNoWrite technique
// (mtime/size, a stronger signal than byte-equality alone, since even a
// no-op rewrite of identical bytes bumps mtime).
type storeFileFingerprint struct {
	modTime time.Time
	size    int64
}

func snapshotStoreFiles(t *testing.T, root string) map[string]storeFileFingerprint {
	t.Helper()
	result := map[string]storeFileFingerprint{}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		result[rel] = storeFileFingerprint{modTime: info.ModTime(), size: info.Size()}
		return nil
	})
	return result
}

// --- Task 1: one projection covers every governed descendant ---

func TestGovernedSubtreeParsesLedgerOnce(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if err := st.RecordSpawn("Builder-01", "scout", "Scout-07", "research the bug", 2); err != nil {
		t.Fatalf("seed recruit: %v", err)
	}

	before := projectGovernedSubtreeParseCalls
	rows, err := projectGovernedSubtree("Builder-01", 0)
	if err != nil {
		t.Fatalf("projectGovernedSubtree: %v", err)
	}
	if projectGovernedSubtreeParseCalls != before+1 {
		t.Fatalf("expected exactly one parse call, got delta %d", projectGovernedSubtreeParseCalls-before)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (itself + one recruit), got %d: %+v", len(rows), rows)
	}
	if rows[0].Name != "Builder-01" || rows[1].Name != "Scout-07" {
		t.Fatalf("unexpected row order: %+v", rows)
	}
}

func TestGovernedSubtreeNoRecruitsReturnsItself(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rows, err := projectGovernedSubtree("Builder-01", 0)
	if err != nil {
		t.Fatalf("projectGovernedSubtree: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "Builder-01" {
		t.Fatalf("expected a one-row projection containing only itself, got %+v", rows)
	}
}

func TestGovernedSubtreeSentinelWithNoRecruitsIsEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	rows, err := projectGovernedSubtree("Queen", 0)
	if err != nil {
		t.Fatalf("projectGovernedSubtree: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected an empty projection for a coordinator with no recruits, got %+v", rows)
	}
}

func TestGovernedSubtreeUnknownRootIsRefused(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	if _, err := projectGovernedSubtree("Nobody-99", 0); err == nil {
		t.Fatalf("expected an error for a root that is neither a recorded entry nor a coordinator sentinel")
	}
}

func TestGovernedSubtreeFollowOnDedup(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if err := st.RecordSpawn("Builder-01", "scout", "Scout-07", "research the bug", 2); err != nil {
		t.Fatalf("seed recruit: %v", err)
	}

	packets := trophallaxisPacketsFile{Entries: []trophallaxisPacket{{
		SchemaVersion: TrophallaxisPacketSchemaVersion,
		PacketID:      "trophallaxis_r1_1",
		ResultID:      "r1",
		ChildName:     "Builder-01",
		Receiver:      "Scout-07",
		ReceiverKind:  trophallaxisReceiverFollowOn,
	}}}
	if err := store.SaveJSON(trophallaxisPacketsPath, packets); err != nil {
		t.Fatalf("seed packet: %v", err)
	}

	rows, err := projectGovernedSubtree("Builder-01", 0)
	if err != nil {
		t.Fatalf("projectGovernedSubtree: %v", err)
	}
	count := 0
	var followOn string
	for _, row := range rows {
		if row.Name == "Scout-07" {
			count++
			followOn = row.FollowOnFor
		}
	}
	if count != 1 {
		t.Fatalf("a descendant that is also a follow-on consumer must appear exactly once, appeared %d times: %+v", count, rows)
	}
	if followOn != "Builder-01" {
		t.Fatalf("expected FollowOnFor to name the origin child %q, got %q", "Builder-01", followOn)
	}
}

func TestGovernedSubtreeDeterministicOrder(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	fixture := strings.Join([]string{
		"2026-01-01T00:00:00Z|Queen|builder|Root-01|build|1|active",
		"2026-01-01T00:01:00Z|Root-01|scout|Charlie|research|2|active",
		"2026-01-01T00:01:00Z|Root-01|scout|Alpha|research|2|active",
		"2026-01-01T00:01:00Z|Root-01|scout|Bravo|research|2|active",
	}, "\n") + "\n"
	if err := store.AtomicWrite("spawn-tree.txt", []byte(fixture)); err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	var first []string
	for i := 0; i < 10; i++ {
		rows, err := projectGovernedSubtree("Root-01", 0)
		if err != nil {
			t.Fatalf("projectGovernedSubtree (read %d): %v", i, err)
		}
		names := make([]string, len(rows))
		for j, row := range rows {
			names[j] = row.Name
		}
		if i == 0 {
			first = names
			continue
		}
		if strings.Join(names, ",") != strings.Join(first, ",") {
			t.Fatalf("read %d returned a different order than read 0: %v vs %v", i, names, first)
		}
	}
	want := []string{"Root-01", "Alpha", "Bravo", "Charlie"}
	if strings.Join(first, ",") != strings.Join(want, ",") {
		t.Fatalf("expected same-depth same-time siblings in worker-identifier order, got %v, want %v", first, want)
	}
}

func TestGovernedSubtreeCostDashWhenNoLedgerRow(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rows, err := projectGovernedSubtree("Builder-01", 5)
	if err != nil {
		t.Fatalf("projectGovernedSubtree: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].CostFigure != spendNotReportedFigure {
		t.Fatalf("expected the dash sentinel with no spend ledger row, got %q", rows[0].CostFigure)
	}
}

func TestGovernedSubtreeCostReadFromLedger(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed: %v", err)
	}

	usage := codex.WorkerUsage{TotalTokens: 500000, Source: codex.UsageSourceProvider}
	ledger := spendLedger{Phase: 42, Workflow: spendWorkflowBuild, Rows: []spendRow{
		{AgentName: "Builder-01", Caste: "builder", Status: "completed", Usage: usage},
	}}
	if err := saveSpendLedger(ledger); err != nil {
		t.Fatalf("save ledger: %v", err)
	}

	rows, err := projectGovernedSubtree("Builder-01", 42)
	if err != nil {
		t.Fatalf("projectGovernedSubtree: %v", err)
	}
	want := spendCompactTokenFigure(usage.BilledTotalTokens())
	if rows[0].CostFigure != want {
		t.Fatalf("expected cost figure %q read from the ledger, got %q", want, rows[0].CostFigure)
	}
}

func TestGovernedSubtreeReadOnly(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if err := st.RecordSpawn("Builder-01", "scout", "Scout-07", "research the bug", 2); err != nil {
		t.Fatalf("seed recruit: %v", err)
	}
	seedRecruitmentManifestChild(t, "Scout-07", "Builder-01", "/tmp/workspace", "root-mediated")

	before := snapshotStoreFiles(t, store.BasePath())

	if _, err := projectGovernedSubtree("Builder-01", 0); err != nil {
		t.Fatalf("projectGovernedSubtree: %v", err)
	}
	if renderGovernedSubtree(nil) != "" {
		t.Fatalf("sanity: renderGovernedSubtree(nil) must be empty")
	}

	after := snapshotStoreFiles(t, store.BasePath())
	// No new file is created under the colony data directory by this
	// projection -- the acceptance criterion's own wording.
	if len(after) != len(before) {
		t.Fatalf("projectGovernedSubtree changed the file count under the colony data directory: before=%d after=%d", len(before), len(after))
	}
	for path, want := range before {
		got, ok := after[path]
		if !ok {
			t.Fatalf("file %q disappeared after a read-only projection call", path)
		}
		if got != want {
			t.Fatalf("file %q was modified by a read-only projection call (store write counter must stay at zero)", path)
		}
	}
}

// --- Task 2: one inline line per recruit, per refusal, and per note ---

func TestInlineRecruitLine(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	var buf bytes.Buffer
	stdout = &buf

	intent := recruitmentIntent{
		SchemaVersion: recruitmentSchemaVersion,
		ParentName:    "Builder-01",
		Caste:         "scout",
		Reason:        "need help checking a pagination bug",
	}
	emitColonyLiveRecruitAdmitted(intent, "Scout-07")

	output := buf.String()
	occurrences := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Scout-07") && strings.Contains(line, "joined") {
			occurrences++
		}
	}
	if occurrences != 1 {
		t.Fatalf("expected exactly one recruit line naming Scout-07, got %d in:\n%s", occurrences, output)
	}
	if !strings.Contains(output, casteLabel("scout")) {
		t.Fatalf("expected the caste label in the recruit line, got:\n%s", output)
	}
	if !strings.Contains(output, "need help checking a pagination bug") {
		t.Fatalf("expected the stated reason in the recruit line, got:\n%s", output)
	}
}

func TestInlineRecruitLineCostDashWithNoLedger(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	var buf bytes.Buffer
	stdout = &buf

	intent := recruitmentIntent{SchemaVersion: recruitmentSchemaVersion, ParentName: "Builder-01", Caste: "scout", Reason: "help"}
	emitColonyLiveRecruitAdmitted(intent, "Scout-07")

	output := buf.String()
	if !strings.Contains(output, spendNotReportedFigure) {
		t.Fatalf("removing the spend ledger must render the dash sentinel, not a zero, got:\n%s", output)
	}
	if strings.Contains(output, "cost so far: 0") {
		t.Fatalf("must never render a zero for an unreported cost, got:\n%s", output)
	}
}

func TestInlineRefusalLine(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	var buf bytes.Buffer
	stdout = &buf

	intent := recruitmentIntent{SchemaVersion: recruitmentSchemaVersion, ParentName: "Builder-01", Caste: "probe"}
	emitColonyLiveRecruitRefused(intent, "budget", "no slots remain")

	output := buf.String()
	occurrences := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "refused") && strings.Contains(line, "budget") {
			occurrences++
		}
	}
	if occurrences != 1 {
		t.Fatalf("expected exactly one refusal line, got %d in:\n%s", occurrences, output)
	}
	if !strings.Contains(output, "carrying on alone") {
		t.Fatalf("expected the refusal line to say the worker is carrying on alone, got:\n%s", output)
	}
}

func TestInlineDecisionChangedLine(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	var buf bytes.Buffer
	stdout = &buf

	if _, _, err := recordRecruitmentCredit("note-42", recruitmentContributionNote, "decision-9", "", recruitmentCreditOutcomePending, ""); err != nil {
		t.Fatalf("record credit: %v", err)
	}

	output := buf.String()
	occurrences := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "note-42") && strings.Contains(line, "decision-9") {
			occurrences++
		}
	}
	if occurrences != 1 {
		t.Fatalf("expected exactly one line naming the note, got %d in:\n%s", occurrences, output)
	}

	// A replay of the identical (contribution, decision) pair must not
	// print a second line -- it mutates nothing (recordRecruitmentCredit's
	// own replay discipline), and this inline line is gated on a genuinely
	// new write.
	buf.Reset()
	if _, _, err := recordRecruitmentCredit("note-42", recruitmentContributionNote, "decision-9", "", recruitmentCreditOutcomePending, ""); err != nil {
		t.Fatalf("replay record credit: %v", err)
	}
	if strings.Contains(buf.String(), "note-42") {
		t.Fatalf("a replayed credit write must not print a second inline line, got:\n%s", buf.String())
	}
}

func TestNoSecondTerminalInstruction(t *testing.T) {
	files := []string{
		"recruitment_subtree.go",
		"live_projection.go",
		"watch_live.go",
		"status.go",
		"codex_visuals.go",
		"live_events.go",
		"recruitment_credit.go",
		"spend_cost_line.go",
	}
	// These name the ACT of telling the owner to open a second surface --
	// distinct from this phase's own repeated PROHIBITION against doing so
	// ("never render colony liveness in a second terminal window"), which
	// legitimately mentions "second terminal" while instructing the exact
	// opposite. A line is only flagged when it reads as an instruction, not
	// a negated rule statement.
	forbidden := []string{
		"open a second terminal", "open another terminal",
		"in a separate terminal window", "start a second terminal",
		"run this in a new terminal", "open a new terminal window",
	}
	for _, name := range files {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		lower := strings.ToLower(string(data))
		for _, phrase := range forbidden {
			if strings.Contains(lower, phrase) {
				t.Fatalf("%s instructs the owner to open a second surface (%q) -- every live surface in this phase must be inline", name, phrase)
			}
		}
	}
}

// --- Task 3: the end-of-run family tree, and the proof nothing weakened ---

func TestRecruitmentFamilyTree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	run, err := st.BeginRun("build", time.Now().UTC())
	if err != nil {
		t.Fatalf("begin run: %v", err)
	}
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if err := st.RecordSpawn("Builder-01", "scout", "Scout-07", "research the bug", 2); err != nil {
		t.Fatalf("seed recruit: %v", err)
	}
	seedRecruitmentManifestChild(t, "Scout-07", "Builder-01", "/tmp/workspace", "root-mediated")

	intents := recruitmentIntentsFile{Entries: []recruitmentIntentRecord{{
		Intent: recruitmentIntent{
			SchemaVersion: recruitmentSchemaVersion,
			ParentName:    "Builder-01",
			Caste:         "probe",
			ParentRunID:   run.ID,
		},
		Decision: &recruitmentDecisionResult{Allowed: false, Reason: "budget", Detail: "no slots remain"},
	}}}
	if err := store.SaveJSON(recruitmentIntentsPath, intents); err != nil {
		t.Fatalf("seed intents: %v", err)
	}

	tree := renderRecruitmentFamilyTree(0)
	if tree == "" {
		t.Fatalf("expected a non-empty family tree")
	}
	if !strings.Contains(tree, "Builder-01") {
		t.Fatalf("expected the parent in the tree, got:\n%s", tree)
	}
	if !strings.Contains(tree, "Scout-07") {
		t.Fatalf("expected the recruit in the tree, got:\n%s", tree)
	}
	if !strings.Contains(tree, "budget") || !strings.Contains(tree, "the worker carried on alone") {
		t.Fatalf("expected the refusal in the tree, got:\n%s", tree)
	}
}

func TestRecruitmentFamilyTreeEmptyRunHasNoSection(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if _, err := st.BeginRun("build", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if got := renderRecruitmentFamilyTree(0); got != "" {
		t.Fatalf("a run with no recruits and no refusals must render no section at all, got:\n%s", got)
	}
}

func TestStatusShowsGovernedSubtree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if err := st.RecordSpawn("Builder-01", "scout", "Scout-07", "research the bug", 2); err != nil {
		t.Fatalf("seed recruit: %v", err)
	}

	section := renderGovernedSubtreeStatusSection(colony.ColonyState{CurrentPhase: 0})
	if section == "" {
		t.Fatalf("expected a non-empty governed subtree section")
	}
	if !strings.Contains(section, "Builder-01") || !strings.Contains(section, "Scout-07") {
		t.Fatalf("expected the subtree to name both workers, got:\n%s", section)
	}
	if !strings.Contains(section, "Governed Subtree") {
		t.Fatalf("expected the section heading, got:\n%s", section)
	}
}

func TestNotesThatChangedDecisionsList(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	if got := renderNotesThatChangedDecisions(); got != "" {
		t.Fatalf("expected no list when no note has changed a decision, got:\n%s", got)
	}

	if _, _, err := recordRecruitmentCredit("note-A", recruitmentContributionNote, "decision-1", "", recruitmentCreditOutcomePending, ""); err != nil {
		t.Fatalf("record credit A: %v", err)
	}
	if _, _, err := recordRecruitmentCredit("note-B", recruitmentContributionNote, "decision-2", "packet-99", recruitmentCreditOutcomeHelpful, ""); err != nil {
		t.Fatalf("record credit B: %v", err)
	}
	if _, _, err := recordRecruitmentCredit("result-C", recruitmentContributionRecruitmentResult, "decision-3", "", recruitmentCreditOutcomePending, ""); err != nil {
		t.Fatalf("record credit C: %v", err)
	}

	got := renderNotesThatChangedDecisions()
	if !strings.Contains(got, "note-A") || !strings.Contains(got, "decision-1") {
		t.Fatalf("expected note-A/decision-1 in the list, got:\n%s", got)
	}
	if !strings.Contains(got, "note-B") || !strings.Contains(got, "decision-2") {
		t.Fatalf("expected note-B/decision-2 in the list, got:\n%s", got)
	}
	if strings.Contains(got, "result-C") {
		t.Fatalf("a non-note contribution must never appear in this list, got:\n%s", got)
	}
}

func TestSubtreeReadersDoNotWeakenAttemptAuthorization(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if _, err := st.BeginRun("build", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if err := st.RecordSpawn("Builder-01", "scout", "Scout-07", "research the bug", 2); err != nil {
		t.Fatalf("seed recruit: %v", err)
	}
	seedRecruitmentManifestChild(t, "Scout-07", "Builder-01", "/tmp/workspace", "root-mediated")

	// The immutable attempt authorization: a recorded, allowed admission
	// decision, and a bound recruitment result carrying its own receipt.
	if _, err := recordRecruitmentIntent(recruitmentIntentRecord{
		Intent:    recruitmentIntent{SchemaVersion: recruitmentSchemaVersion, IntentID: "intent-Scout-07", ParentName: "Builder-01", Caste: "scout"},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("record intent: %v", err)
	}
	if _, err := recordRecruitmentDecision("intent-Scout-07", recruitmentDecisionResult{Allowed: true}, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("record decision: %v", err)
	}
	if _, err := bindRecruitmentResult(recruitmentResult{
		RecruitmentID:  "intent-Scout-07",
		IntentID:       "intent-Scout-07",
		ChildName:      "Scout-07",
		ParentName:     "Builder-01",
		TerminalStatus: RecruitmentTerminalStatusCompleted,
		Transaction:    colony.LifecycleTransactionReference{ID: "intent-Scout-07", Stage: colony.TransactionStageCommitted},
	}); err != nil {
		t.Fatalf("bind result: %v", err)
	}

	intentsPath := filepath.Join(store.BasePath(), recruitmentIntentsPath)
	resultsPath := filepath.Join(store.BasePath(), recruitmentResultsPath)
	beforeIntents, err := os.ReadFile(intentsPath)
	if err != nil {
		t.Fatalf("read intents before: %v", err)
	}
	beforeResults, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("read results before: %v", err)
	}

	// Exercise every subtree reader.
	if _, err := projectGovernedSubtree("Builder-01", 0); err != nil {
		t.Fatalf("projectGovernedSubtree: %v", err)
	}
	_ = renderGovernedSubtreeStatusSection(colony.ColonyState{CurrentPhase: 0})
	_ = renderRecruitmentFamilyTree(0)
	_ = renderNotesThatChangedDecisions()

	afterIntents, err := os.ReadFile(intentsPath)
	if err != nil {
		t.Fatalf("read intents after: %v", err)
	}
	afterResults, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("read results after: %v", err)
	}
	if !bytes.Equal(beforeIntents, afterIntents) {
		t.Fatalf("a subtree reader rewrote the recorded admission decision")
	}
	if !bytes.Equal(beforeResults, afterResults) {
		t.Fatalf("a subtree reader rewrote the bound recruitment result")
	}
}

func TestFamilyTreeAndCostBlockAgree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if _, err := st.BeginRun("build", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if err := st.RecordSpawn("Builder-01", "scout", "Scout-07", "research the bug", 2); err != nil {
		t.Fatalf("seed recruit: %v", err)
	}
	seedRecruitmentManifestChild(t, "Scout-07", "Builder-01", "/tmp/workspace", "root-mediated")

	const phase = 77
	builderUsage := codex.WorkerUsage{TotalTokens: 500000, Source: codex.UsageSourceProvider}
	scoutUsage := codex.WorkerUsage{TotalTokens: 300000, Source: codex.UsageSourceProvider}
	ledger := spendLedger{Phase: phase, Workflow: spendWorkflowBuild, Rows: []spendRow{
		{AgentName: "Builder-01", Caste: "builder", Status: "completed", Usage: builderUsage},
		{AgentName: "Scout-07", Caste: "scout", Status: "completed", Usage: scoutUsage},
	}}
	if err := saveSpendLedger(ledger); err != nil {
		t.Fatalf("save ledger: %v", err)
	}

	rows, err := projectGovernedSubtree("Builder-01", phase)
	if err != nil {
		t.Fatalf("projectGovernedSubtree: %v", err)
	}
	var builderRow, scoutRow governedSubtreeRow
	for _, row := range rows {
		switch row.Name {
		case "Builder-01":
			builderRow = row
		case "Scout-07":
			scoutRow = row
		}
	}
	wantBuilder := spendCompactTokenFigure(builderUsage.BilledTotalTokens())
	wantScout := spendCompactTokenFigure(scoutUsage.BilledTotalTokens())
	if builderRow.CostFigure != wantBuilder {
		t.Fatalf("Builder-01 branch figure = %q, want %q", builderRow.CostFigure, wantBuilder)
	}
	if scoutRow.CostFigure != wantScout {
		t.Fatalf("Scout-07 branch figure = %q, want %q", scoutRow.CostFigure, wantScout)
	}

	ledgers, ok := loadSpendLedgersForPhase(phase)
	if !ok {
		t.Fatalf("expected the ledger to load back")
	}
	totals := computeSpendTotals(ledgers)
	sumOfBranches := builderUsage.BilledTotalTokens() + scoutUsage.BilledTotalTokens()
	if sumOfBranches != totals.MeasuredTokens {
		t.Fatalf("the branch figures do not sum to the closing block's total: branches=%d total=%d", sumOfBranches, totals.MeasuredTokens)
	}

	closingBlock := renderSpendCostLineFromLedgers(ledgers)
	wantTotalFigure := spendCompactTokenFigure(totals.MeasuredTokens)
	if !strings.Contains(closingBlock, wantTotalFigure) {
		t.Fatalf("closing block does not show the expected total figure %q:\n%s", wantTotalFigure, closingBlock)
	}

	// And the family tree, reading the SAME ledger, agrees.
	tree := renderRecruitmentFamilyTree(phase)
	if !strings.Contains(tree, wantBuilder) || !strings.Contains(tree, wantScout) {
		t.Fatalf("family tree does not show the same branch figures as the closing block:\n%s", tree)
	}
}
