package cmd

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// WS1 — /ant-ask: the colony's full memory handed to a conversation with
// zero setup. The runtime side is colony-prime's first parameterization:
// --question boosts the sections the question is about and adds the
// activity tail; the wrapper answers FROM the briefing, no worker spawn.

func seedAskColony(t *testing.T) colony.ColonyState {
	t.Helper()
	goal := "Ship the payment reconciliation service"
	state := colony.ColonyState{Version: "3.0", State: colony.StateEXECUTING, Goal: &goal, CurrentPhase: 2}
	state.Plan.Phases = []colony.Phase{
		{ID: 1, Name: "Scaffold", Status: colony.PhaseCompleted},
		{ID: 2, Name: "Reconciliation engine", Status: colony.PhaseInProgress},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	// A blocker the question will be about.
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
		ID: "pd_1", Type: "blocker", Description: "ledger import fails on duplicate transaction ids", CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}}}); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	return state
}

func TestColonyPrimeQuestionBoostsRelevantSections(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	seedAskColony(t)

	// Ordering invariant, not section-exists: with a question about the
	// blocker, the blockers section must rank at least as high as it does
	// without the question (a keyword-matched boost can only raise it).
	rankOf := func(out colonyPrimeOutput, name string) int {
		for i, item := range out.Ledger.Included {
			if item.Name == name {
				return i
			}
		}
		return len(out.Ledger.Included) + 1
	}
	baseline := buildColonyPrimeOutputOpts(colonyPrimeOptions{Compact: true})
	asked := buildColonyPrimeOutputOpts(colonyPrimeOptions{Compact: true, Question: "why does the ledger import fail on duplicate transaction ids?"})

	if rankOf(asked, "blockers") > rankOf(baseline, "blockers") {
		t.Fatalf("a question ABOUT the blocker ranked the blockers section lower: baseline=%d asked=%d", rankOf(baseline, "blockers"), rankOf(asked, "blockers"))
	}
	if !strings.Contains(asked.Context, "duplicate transaction ids") {
		t.Fatalf("the asked briefing does not carry the blocker's content:\n%s", asked.Context)
	}

	// The boost is genuinely question-driven: a section with no keyword
	// overlap gets zero.
	if boost := questionRelevanceBoost("why does the ledger import fail?", colonyPrimeSection{name: "user_preferences", title: "User Preferences", content: "short answers please"}); boost != 0 {
		t.Fatalf("unrelated section got a question boost: %v", boost)
	}
	if boost := questionRelevanceBoost("why does the ledger import fail?", colonyPrimeSection{name: "blockers", title: "Blockers", content: "ledger import fails on duplicate ids"}); boost <= 0 {
		t.Fatalf("matching section got no question boost")
	}
}

func TestColonyPrimeActivityTailSection(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	seedAskColony(t)

	// activity.log holds the real feed while state.Events stays empty —
	// the exact gap that made "what changed?" unanswerable.
	for _, entry := range []map[string]interface{}{
		{"timestamp": "2026-08-17T10:00:00Z", "action": "COLONY_INITIALIZED", "detail": "goal set"},
		{"timestamp": "2026-08-17T11:00:00Z", "action": "BUILD_STARTED", "detail": "phase 2 reconciliation engine"},
	} {
		if err := store.AppendJSONL("activity.log", entry); err != nil {
			t.Fatalf("seed activity: %v", err)
		}
	}

	asked := buildColonyPrimeOutputOpts(colonyPrimeOptions{Question: "what happened recently?"})
	if !strings.Contains(asked.Context, "Recent Activity") || !strings.Contains(asked.Context, "BUILD_STARTED") {
		t.Fatalf("asked briefing missing the activity tail:\n%s", asked.Context)
	}

	// Without a question the roster is unchanged — worker briefings do not
	// grow an activity section.
	baseline := buildColonyPrimeOutput(false)
	if strings.Contains(baseline.Context, "Recent Activity") {
		t.Fatalf("activity tail leaked into the no-question briefing")
	}
}

// TestColonyPrimeQuestionIsReadOnly hashes every file under the colony's
// data dir before and after an ask — ask mode is a pure inspection.
func TestColonyPrimeQuestionIsReadOnly(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	seedAskColony(t)
	if err := store.AppendJSONL("activity.log", map[string]interface{}{"timestamp": "2026-08-17T10:00:00Z", "action": "COLONY_INITIALIZED"}); err != nil {
		t.Fatalf("seed activity: %v", err)
	}

	snapshot := func() map[string][32]byte {
		hashes := map[string][32]byte{}
		_ = filepath.WalkDir(s.BasePath(), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			hashes[path] = sha256.Sum256(data)
			return nil
		})
		return hashes
	}

	before := snapshot()
	var buf, errBuf strings.Builder
	stdout = &buf
	stderr = &errBuf
	rootCmd.SetArgs([]string{"colony-prime", "--question", "why did phase 2 block?"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --question: %v", err)
	}
	after := snapshot()

	if len(before) != len(after) {
		t.Fatalf("ask mode changed the file count in .aether/data: %d -> %d", len(before), len(after))
	}
	keys := make([]string, 0, len(before))
	for k := range before {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if before[k] != after[k] {
			t.Fatalf("ask mode modified %s — it must be a pure inspection", k)
		}
	}
}

// TestAskWrapperAnswersWithoutSpawning pins the wrapper contract in all
// three copies: colony-prime is the source, the answer comes from the
// briefing, no worker spawn, code questions route to /ant-quick.
func TestAskWrapperAnswersWithoutSpawning(t *testing.T) {
	for _, path := range []string{"../.claude/commands/ant/ask.md", "../.claude/commands/ant-ask.md", "../.opencode/commands/ant/ask.md"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		for _, anchor := range []string{
			"aether colony-prime --question",
			"Do NOT spawn workers",
			"Do NOT invent beyond the briefing",
			"/ant-quick",
		} {
			if !strings.Contains(text, anchor) {
				t.Fatalf("%s lost the ask contract anchor %q", path, anchor)
			}
		}
	}
}
