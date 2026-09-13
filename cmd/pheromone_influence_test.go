package cmd

import (
	"encoding/json"
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// pheromoneJSONWriterAllowlist (cmd/pheromone_resolver_test.go, TestOnePheromoneWriterOnly)
// is a ratchet: a new writer of pheromones.json must be added here
// deliberately, with a stated reason, rather than by editing the file that
// declares it (the same extension contract cmd/pheromone_approval_test.go's
// own init() already follows for clearSignalQuarantine).
func init() {
	pheromoneJSONWriterAllowlist["pheromoneInfluenceSaveSignals"] = "the one save path reinforceNote, deferNote, expireNote and revokeNote route their pheromones.json mutation through, so the ratchet needs one new entry instead of four"
}

// --- shared test seeding helpers ---

// seedPheromoneInfluenceSignal upserts a minimal signal into pheromones.json
// by ID -- it never discards other already-seeded signals, so tests that
// seed several notes in sequence (e.g. TestInfluenceHistoryRetention's live
// note alongside many dead ones) can rely on every previously seeded note
// still existing until the test explicitly removes it.
func seedPheromoneInfluenceSignal(t *testing.T, id string) colony.PheromoneSignal {
	t.Helper()
	strength := 0.5
	sig := colony.PheromoneSignal{
		ID:        id,
		Type:      "FOCUS",
		Priority:  "normal",
		Source:    "cli",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Active:    true,
		Strength:  &strength,
		Content:   json.RawMessage(`{"text":"test note"}`),
	}
	var pf colony.PheromoneFile
	_ = store.LoadJSON("pheromones.json", &pf) // ignore error: the file may not exist yet
	replaced := false
	for i := range pf.Signals {
		if pf.Signals[i].ID == id {
			pf.Signals[i] = sig
			replaced = true
			break
		}
	}
	if !replaced {
		pf.Signals = append(pf.Signals, sig)
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("seed signal %q: %v", id, err)
	}
	return sig
}

func seedRejectedPendingNote(t *testing.T, id string) colony.PendingSuggestion {
	t.Helper()
	rejected := colony.PendingActionRejected
	now := time.Now().UTC().Format(time.RFC3339)
	origin := colony.PendingOriginSuggestion
	item := colony.PendingSuggestion{
		ID:          id,
		Type:        "FOCUS",
		Content:     "test",
		ContentHash: "sha256:test",
		CreatedAt:   now,
		Dismissed:   true,
		Origin:      &origin,
		Action:      &rejected,
		ActionAt:    &now,
	}
	items := []colony.PendingSuggestion{item}
	cs := colony.ColonyState{PendingSuggestions: &items}
	if err := store.SaveJSON("COLONY_STATE.json", cs); err != nil {
		t.Fatalf("seed rejected pending note %q: %v", id, err)
	}
	return item
}

// --- Task 1: an append-only history with an actor on every entry ---

// writesPheromonesHistoryJSON reports whether fn contains a direct call
// persisting "pheromones-history.json" -- via store.UpdateJSONAtomically,
// store.SaveJSON, or store.SaveRawJSON. Brand new file, no legacy writers:
// exactly one function in the package may do this.
func writesPheromonesHistoryJSON(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return true
		}
		switch sel.Sel.Name {
		case "UpdateJSONAtomically", "SaveJSON", "SaveRawJSON":
		default:
			return true
		}
		if len(call.Args) == 0 {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(lit.Value)
		if err == nil && value == "pheromones-history.json" {
			found = true
		}
		return true
	})
	return found
}

// assignsIntoInfluenceHistorySliceElement reports whether fn assigns into an
// element of the Notes field's slice-of-entries value -- a double-index LHS
// of the shape `X.Notes[id][i] = entry` -- the one shape a rewrite of an
// existing entry could take, since the history is stored as
// map[string][]pheromoneInfluenceEntry keyed by a "Notes" field. A plain
// `hf.Notes[id] = append(hf.Notes[id], entry)` (single index, RHS is append)
// is the sanctioned append and is never flagged. Scoped to the "Notes" field
// name, like TestNoUngovernedQuarantineClear's name-based heuristic on
// "Quarantined" in cmd/pheromone_resolver_test.go, so this detector does not
// false-positive against unrelated double-index assignments elsewhere in the
// package (pure AST inspection cannot type-check).
func assignsIntoInfluenceHistorySliceElement(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range assign.Lhs {
			outer, ok := lhs.(*ast.IndexExpr)
			if !ok {
				continue
			}
			inner, ok := outer.X.(*ast.IndexExpr)
			if !ok {
				continue
			}
			sel, ok := inner.X.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != "Notes" {
				continue
			}
			found = true
		}
		return true
	})
	return found
}

func TestInfluenceHistoryAppendOnly(t *testing.T) {
	t.Run("structural_single_writer", func(t *testing.T) {
		funcs := parseCmdPackageFuncs(t)
		var writers []string
		for name, fn := range funcs {
			if writesPheromonesHistoryJSON(fn) {
				writers = append(writers, name)
			}
		}
		sort.Strings(writers)
		if len(writers) != 1 || writers[0] != "appendInfluenceHistory" {
			t.Fatalf("expected exactly one writer of pheromones-history.json (appendInfluenceHistory), got %v", writers)
		}
	})

	t.Run("no_slice_element_assignment", func(t *testing.T) {
		funcs := parseCmdPackageFuncs(t)
		var offenders []string
		for name, fn := range funcs {
			if assignsIntoInfluenceHistorySliceElement(fn) {
				offenders = append(offenders, name)
			}
		}
		sort.Strings(offenders)
		if len(offenders) > 0 {
			t.Fatalf("found a function assigning into a pheromone influence history slice element (must only append): %v", offenders)
		}
	})

	t.Run("round_trip_byte_identical", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-round-trip")

		for i := 0; i < 10; i++ {
			if _, err := appendInfluenceHistory("note-round-trip", pheromoneActionReinforced, pheromoneActorOwner, "tester", "before", "after", "reason"); err != nil {
				t.Fatalf("append %d: %v", i, err)
			}
		}

		first, err := readInfluenceHistory("note-round-trip")
		if err != nil {
			t.Fatalf("read 1: %v", err)
		}
		second, err := readInfluenceHistory("note-round-trip")
		if err != nil {
			t.Fatalf("read 2: %v", err)
		}
		if len(first) != 10 {
			t.Fatalf("expected 10 entries, got %d", len(first))
		}
		firstJSON, _ := json.Marshal(first)
		secondJSON, _ := json.Marshal(second)
		if string(firstJSON) != string(secondJSON) {
			t.Fatalf("two reads returned different output:\n%s\nvs\n%s", firstJSON, secondJSON)
		}
	})

	t.Run("refuses_unknown_note", func(t *testing.T) {
		setupExchangeTest(t)
		_, err := appendInfluenceHistory("does-not-exist", pheromoneActionReinforced, pheromoneActorOwner, "tester", "", "", "")
		if err == nil {
			t.Fatalf("expected an error for an unknown note, got nil")
		}
		if !strings.Contains(err.Error(), "does-not-exist") {
			t.Fatalf("error does not name the missing note id: %v", err)
		}
		history, rerr := readInfluenceHistory("does-not-exist")
		if rerr != nil {
			t.Fatalf("read: %v", rerr)
		}
		if len(history) != 0 {
			t.Fatalf("expected nothing written for a refused note, got %d entries", len(history))
		}
	})
}

// TestPendingNoteActionsAppendRatherThanOverwriteHistory is CR-03's
// append-only proof for the three approval-surface actions specifically:
// repeatedly editing and then rejecting the SAME queued note must accumulate
// entries, never rewrite or drop an earlier one -- driven through the real
// editPendingNote/rejectPendingNote functions, not just the AST-structural
// checks above (which only prove nothing ELSE in the package writes the
// file, not that these three specific functions behave append-only in
// practice).
func TestPendingNoteActionsAppendRatherThanOverwriteHistory(t *testing.T) {
	setupExchangeTest(t)
	id := "append-only-behavioral"
	now := time.Now().UTC().Format(time.RFC3339)
	origin := colony.PendingOriginSuggestion
	item := colony.PendingSuggestion{
		ID:          id,
		Type:        "FOCUS",
		Content:     "first wording",
		ContentHash: "sha256:" + sha256Sum("first wording"),
		CreatedAt:   now,
		Origin:      &origin,
	}
	items := []colony.PendingSuggestion{item}
	if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{PendingSuggestions: &items}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := editPendingNote(id, pheromoneActorOwner, "first-editor", "second wording", false); err != nil {
		t.Fatalf("first edit: %v", err)
	}
	firstHistory, err := readInfluenceHistory(id)
	if err != nil {
		t.Fatalf("read after first edit: %v", err)
	}
	if len(firstHistory) != 1 {
		t.Fatalf("expected 1 entry after first edit, got %d: %+v", len(firstHistory), firstHistory)
	}
	firstEntrySnapshot := firstHistory[0]

	if _, err := editPendingNote(id, pheromoneActorOwner, "second-editor", "third wording", false); err != nil {
		t.Fatalf("second edit: %v", err)
	}
	if _, err := rejectPendingNote(id, pheromoneActorOwner, "rejector", false); err != nil {
		t.Fatalf("reject: %v", err)
	}

	finalHistory, err := readInfluenceHistory(id)
	if err != nil {
		t.Fatalf("read after further actions: %v", err)
	}
	if len(finalHistory) != 3 {
		t.Fatalf("expected 3 accumulated entries, got %d: %+v", len(finalHistory), finalHistory)
	}
	if finalHistory[0] != firstEntrySnapshot {
		t.Fatalf("the first entry was mutated by a later action:\noriginal: %+v\nnow:      %+v", firstEntrySnapshot, finalHistory[0])
	}
	if finalHistory[1].Action != pheromoneActionEdited || finalHistory[1].ActorName != "second-editor" {
		t.Fatalf("second entry wrong: %+v", finalHistory[1])
	}
	if finalHistory[2].Action != pheromoneActionRejected || finalHistory[2].ActorName != "rejector" {
		t.Fatalf("third entry wrong: %+v", finalHistory[2])
	}
}

func TestInfluenceHistoryActor(t *testing.T) {
	t.Run("declared_set", func(t *testing.T) {
		actors := pheromoneActors()
		want := map[string]bool{pheromoneActorOwner: true, pheromoneActorRuntime: true, pheromoneActorLearning: true}
		if len(actors) != len(want) {
			t.Fatalf("pheromoneActors() = %v, want exactly %v", actors, want)
		}
		for _, a := range actors {
			if !want[a] {
				t.Fatalf("pheromoneActors() contains undeclared kind %q", a)
			}
		}
	})

	t.Run("refuses_undeclared_actor_kind", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-actor")
		_, err := appendInfluenceHistory("note-actor", pheromoneActionReinforced, "saboteur", "tester", "", "", "")
		if err == nil {
			t.Fatalf("expected an error for an undeclared actor kind, got nil")
		}
		if !strings.Contains(err.Error(), "saboteur") {
			t.Fatalf("error does not name the undeclared actor kind: %v", err)
		}
		history, rerr := readInfluenceHistory("note-actor")
		if rerr != nil {
			t.Fatalf("read: %v", rerr)
		}
		if len(history) != 0 {
			t.Fatalf("expected nothing written for an undeclared actor kind, got %d entries", len(history))
		}
	})

	t.Run("records_declared_actor_kind", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-actor-2")
		entry, err := appendInfluenceHistory("note-actor-2", pheromoneActionReinforced, pheromoneActorRuntime, "phase-end", "", "", "")
		if err != nil {
			t.Fatalf("append: %v", err)
		}
		if entry.ActorKind != pheromoneActorRuntime {
			t.Fatalf("entry actor kind = %q, want %q", entry.ActorKind, pheromoneActorRuntime)
		}
		if entry.ActorName != "phase-end" {
			t.Fatalf("entry actor name = %q, want %q", entry.ActorName, "phase-end")
		}
	})
}

func TestInfluenceHistoryRetention(t *testing.T) {
	setupExchangeTest(t)

	// A live note whose history grows past the retention bound must never
	// be pruned.
	seedPheromoneInfluenceSignal(t, "note-live")
	for i := 0; i < pheromoneInfluenceHistoryRetentionDeadNotes+50; i++ {
		if _, err := appendInfluenceHistory("note-live", pheromoneActionReinforced, pheromoneActorOwner, "tester", "", "", ""); err != nil {
			t.Fatalf("append live note entry %d: %v", i, err)
		}
	}

	// Seed and then extinguish more dead notes than the retention bound
	// allows, each with its own single entry.
	deadCount := pheromoneInfluenceHistoryRetentionDeadNotes + 25
	for i := 0; i < deadCount; i++ {
		id := "note-dead-" + strconv.Itoa(i)
		seedPheromoneInfluenceSignal(t, id)
		if _, err := appendInfluenceHistory(id, pheromoneActionExpired, pheromoneActorOwner, "tester", "", "", ""); err != nil {
			t.Fatalf("append dead note entry %d: %v", i, err)
		}
	}
	// Remove every dead note from pheromones.json so pheromoneNoteExists
	// reports them as extinguished; keep the live note.
	live := seedPheromoneInfluenceSignal(t, "note-live")
	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{live}}); err != nil {
		t.Fatalf("collapse pheromones.json to the live note only: %v", err)
	}
	// One more append against the live note triggers pruning of the now
	// fully-extinguished dead notes.
	if _, err := appendInfluenceHistory("note-live", pheromoneActionReinforced, pheromoneActorOwner, "tester", "", "", ""); err != nil {
		t.Fatalf("trigger prune: %v", err)
	}

	liveHistory, err := readInfluenceHistory("note-live")
	if err != nil {
		t.Fatalf("read live history: %v", err)
	}
	if len(liveHistory) != pheromoneInfluenceHistoryRetentionDeadNotes+51 {
		t.Fatalf("live note history was pruned: got %d entries, want %d", len(liveHistory), pheromoneInfluenceHistoryRetentionDeadNotes+51)
	}

	var hf pheromoneInfluenceHistoryFile
	if err := store.LoadJSON("pheromones-history.json", &hf); err != nil {
		t.Fatalf("load history file: %v", err)
	}
	deadRemaining := 0
	for id := range hf.Notes {
		if id != "note-live" {
			deadRemaining++
		}
	}
	if deadRemaining > pheromoneInfluenceHistoryRetentionDeadNotes {
		t.Fatalf("dead note histories were not pruned to the retention bound: got %d, want <= %d", deadRemaining, pheromoneInfluenceHistoryRetentionDeadNotes)
	}
}

// --- Task 2: build the five actions that do not yet exist ---

func TestInfluenceActions(t *testing.T) {
	t.Run("reinforce", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-reinforce")

		before := pheromoneInfluenceWriteCount
		outcome, err := reinforceNote("note-reinforce", pheromoneActorOwner, "tester", "good note", false)
		if err != nil {
			t.Fatalf("reinforce: %v", err)
		}
		if !outcome.Found || outcome.Signal == nil {
			t.Fatalf("expected a found signal, got %+v", outcome)
		}
		if outcome.Signal.Strength == nil || *outcome.Signal.Strength != 1.0 {
			t.Fatalf("strength = %v, want 1.0", outcome.Signal.Strength)
		}
		if outcome.Signal.ReinforcementCount == nil || *outcome.Signal.ReinforcementCount != 1 {
			t.Fatalf("reinforcement count = %v, want 1", outcome.Signal.ReinforcementCount)
		}
		history, _ := readInfluenceHistory("note-reinforce")
		if len(history) != 1 || history[0].Action != pheromoneActionReinforced {
			t.Fatalf("expected exactly one reinforced history entry, got %+v", history)
		}
		if pheromoneInfluenceWriteCount != before+2 { // pheromones.json save + history append
			t.Fatalf("write count = %d, want %d", pheromoneInfluenceWriteCount, before+2)
		}
	})

	t.Run("defer", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-defer")
		until := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)

		outcome, err := deferNote("note-defer", pheromoneActorRuntime, "phase-end", until, "cooling off", false)
		if err != nil {
			t.Fatalf("defer: %v", err)
		}
		if outcome.Signal == nil || outcome.Signal.DeferredUntil == nil || *outcome.Signal.DeferredUntil != until {
			t.Fatalf("deferred_until = %v, want %v", outcome.Signal, until)
		}
		history, _ := readInfluenceHistory("note-defer")
		if len(history) != 1 || history[0].Action != pheromoneActionDeferred {
			t.Fatalf("expected exactly one deferred history entry, got %+v", history)
		}
	})

	t.Run("expire", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-expire")

		outcome, err := expireNote("note-expire", pheromoneActorLearning, "consolidation", "no longer useful", false)
		if err != nil {
			t.Fatalf("expire: %v", err)
		}
		if outcome.Signal == nil || outcome.Signal.ExpiresAt == nil {
			t.Fatalf("expected an expires_at to be set, got %+v", outcome.Signal)
		}
		expiresAt, err := time.Parse(time.RFC3339, *outcome.Signal.ExpiresAt)
		if err != nil {
			t.Fatalf("expires_at is not RFC3339: %v", err)
		}
		if expiresAt.After(time.Now().UTC().Add(time.Minute)) {
			t.Fatalf("expires_at = %v, want approximately now", expiresAt)
		}
		history, _ := readInfluenceHistory("note-expire")
		if len(history) != 1 || history[0].Action != pheromoneActionExpired {
			t.Fatalf("expected exactly one expired history entry, got %+v", history)
		}
	})

	t.Run("revoke", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-revoke")

		outcome, err := revokeNote("note-revoke", pheromoneActorOwner, "tester", "bad note", false)
		if err != nil {
			t.Fatalf("revoke: %v", err)
		}
		if outcome.Signal == nil || outcome.Signal.RevokedAt == nil || *outcome.Signal.RevokedAt == "" {
			t.Fatalf("expected revoked_at to be set, got %+v", outcome.Signal)
		}
		history, _ := readInfluenceHistory("note-revoke")
		if len(history) != 1 || history[0].Action != pheromoneActionRevoked {
			t.Fatalf("expected exactly one revoked history entry, got %+v", history)
		}
	})

	t.Run("appeal", func(t *testing.T) {
		setupExchangeTest(t)
		seedRejectedPendingNote(t, "note-appeal")

		outcome, err := appealNote("note-appeal", pheromoneActorOwner, "tester", "reconsider please", false)
		if err != nil {
			t.Fatalf("appeal: %v", err)
		}
		if !outcome.Found {
			t.Fatalf("expected the appeal to be found")
		}
		var cs colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &cs); err != nil {
			t.Fatalf("load colony state: %v", err)
		}
		if cs.PendingSuggestions == nil || len(*cs.PendingSuggestions) != 1 {
			t.Fatalf("expected the queued item to be untouched, got %+v", cs.PendingSuggestions)
		}
		item := (*cs.PendingSuggestions)[0]
		if item.Action == nil || *item.Action != colony.PendingActionRejected {
			t.Fatalf("appeal changed the rejected state: %+v", item)
		}
		history, _ := readInfluenceHistory("note-appeal")
		if len(history) != 1 || history[0].Action != pheromoneActionAppealed {
			t.Fatalf("expected exactly one appealed history entry, got %+v", history)
		}
	})

	t.Run("each_refuses_unknown_note", func(t *testing.T) {
		setupExchangeTest(t)
		cases := []struct {
			name string
			run  func() error
		}{
			{"reinforce", func() error { _, err := reinforceNote("missing", pheromoneActorOwner, "t", "", false); return err }},
			{"defer", func() error {
				_, err := deferNote("missing", pheromoneActorOwner, "t", time.Now().Add(time.Hour).UTC().Format(time.RFC3339), "", false)
				return err
			}},
			{"expire", func() error { _, err := expireNote("missing", pheromoneActorOwner, "t", "", false); return err }},
			{"revoke", func() error { _, err := revokeNote("missing", pheromoneActorOwner, "t", "", false); return err }},
			{"appeal", func() error { _, err := appealNote("missing", pheromoneActorOwner, "t", "", false); return err }},
		}
		for _, tc := range cases {
			err := tc.run()
			if err == nil {
				t.Fatalf("%s: expected an error for an unknown note, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), "missing") {
				t.Fatalf("%s: error does not name the missing note id: %v", tc.name, err)
			}
		}
	})

	t.Run("dry_run_leaves_store_untouched", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-dry-run")
		before := pheromoneInfluenceWriteCount

		if _, err := reinforceNote("note-dry-run", pheromoneActorOwner, "t", "", true); err != nil {
			t.Fatalf("dry-run reinforce: %v", err)
		}
		if _, err := deferNote("note-dry-run", pheromoneActorOwner, "t", time.Now().Add(time.Hour).UTC().Format(time.RFC3339), "", true); err != nil {
			t.Fatalf("dry-run defer: %v", err)
		}
		if _, err := expireNote("note-dry-run", pheromoneActorOwner, "t", "", true); err != nil {
			t.Fatalf("dry-run expire: %v", err)
		}
		if _, err := revokeNote("note-dry-run", pheromoneActorOwner, "t", "", true); err != nil {
			t.Fatalf("dry-run revoke: %v", err)
		}
		seedRejectedPendingNote(t, "note-dry-run-appeal")
		if _, err := appealNote("note-dry-run-appeal", pheromoneActorOwner, "t", "", true); err != nil {
			t.Fatalf("dry-run appeal: %v", err)
		}

		if pheromoneInfluenceWriteCount != before {
			t.Fatalf("dry-run mutated the store: write count went from %d to %d", before, pheromoneInfluenceWriteCount)
		}
		history, _ := readInfluenceHistory("note-dry-run")
		if len(history) != 0 {
			t.Fatalf("dry-run wrote history entries: %+v", history)
		}
	})
}

func TestDeferredNoteReturns(t *testing.T) {
	setupExchangeTest(t)
	seedPheromoneInfluenceSignal(t, "note-defer-returns")

	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)

	if _, err := deferNote("note-defer-returns", pheromoneActorOwner, "t", future, "", false); err != nil {
		t.Fatalf("defer: %v", err)
	}

	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("load pheromones: %v", err)
	}

	before := resolveEffectivePheromones(&pf, time.Now().UTC())
	if len(before) != 1 || before[0].InEffect || before[0].ExcludedReason != pheromoneExcludedDeferred {
		t.Fatalf("expected the note excluded with reason %q before its window ends, got %+v", pheromoneExcludedDeferred, before)
	}

	after := resolveEffectivePheromones(&pf, time.Now().Add(2*time.Hour).UTC())
	if len(after) != 1 || !after[0].InEffect {
		t.Fatalf("expected the note back in effect after its defer window passed, got %+v", after)
	}

	// A note deferred into the past is already back in effect -- no
	// intervening command needed.
	if _, err := deferNote("note-defer-returns", pheromoneActorOwner, "t", past, "", false); err != nil {
		t.Fatalf("defer into the past: %v", err)
	}
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("reload pheromones: %v", err)
	}
	stillIn := resolveEffectivePheromones(&pf, time.Now().UTC())
	if len(stillIn) != 1 || !stillIn[0].InEffect {
		t.Fatalf("expected a note deferred into the past to already be in effect, got %+v", stillIn)
	}
}

func TestRevokedNoteStaysOut(t *testing.T) {
	setupExchangeTest(t)
	seedPheromoneInfluenceSignal(t, "note-revoke-stays-out")

	if _, err := revokeNote("note-revoke-stays-out", pheromoneActorOwner, "t", "bad", false); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("load pheromones: %v", err)
	}
	resolved := resolveEffectivePheromones(&pf, time.Now().UTC())
	if len(resolved) != 1 || resolved[0].InEffect || resolved[0].ExcludedReason != pheromoneExcludedRevoked {
		t.Fatalf("expected the note excluded with reason %q, got %+v", pheromoneExcludedRevoked, resolved)
	}
	// Permanently: even far in the future, a revoked note stays out.
	future := resolveEffectivePheromones(&pf, time.Now().Add(365*24*time.Hour).UTC())
	if len(future) != 1 || future[0].InEffect {
		t.Fatalf("expected the revoked note to remain excluded far in the future, got %+v", future)
	}

	history, err := readInfluenceHistory("note-revoke-stays-out")
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	if len(history) != 1 || history[0].Action != pheromoneActionRevoked {
		t.Fatalf("expected the revoke to be fully readable in history, got %+v", history)
	}
}

func TestInfluenceActorPermissions(t *testing.T) {
	t.Run("runtime_cannot_revoke", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-perm-revoke")
		_, err := revokeNote("note-perm-revoke", pheromoneActorRuntime, "t", "", false)
		if err == nil {
			t.Fatalf("expected the runtime actor to be refused revoke")
		}
		if !strings.Contains(err.Error(), pheromoneActorRuntime) {
			t.Fatalf("error does not name the refused actor kind: %v", err)
		}
		var pf colony.PheromoneFile
		if err := store.LoadJSON("pheromones.json", &pf); err != nil {
			t.Fatalf("load pheromones: %v", err)
		}
		if pf.Signals[0].RevokedAt != nil {
			t.Fatalf("runtime actor's refused revoke still mutated the signal: %+v", pf.Signals[0])
		}
	})

	t.Run("learning_cannot_appeal", func(t *testing.T) {
		setupExchangeTest(t)
		seedRejectedPendingNote(t, "note-perm-appeal")
		_, err := appealNote("note-perm-appeal", pheromoneActorLearning, "t", "", false)
		if err == nil {
			t.Fatalf("expected the learning actor to be refused appeal")
		}
		if !strings.Contains(err.Error(), pheromoneActorLearning) {
			t.Fatalf("error does not name the refused actor kind: %v", err)
		}
	})

	t.Run("runtime_and_learning_may_reinforce_defer_expire", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-perm-ok")
		if _, err := reinforceNote("note-perm-ok", pheromoneActorRuntime, "t", "", false); err != nil {
			t.Fatalf("runtime reinforce should be allowed: %v", err)
		}
		if _, err := deferNote("note-perm-ok", pheromoneActorLearning, "t", time.Now().Add(time.Hour).UTC().Format(time.RFC3339), "", false); err != nil {
			t.Fatalf("learning defer should be allowed: %v", err)
		}
		if _, err := expireNote("note-perm-ok", pheromoneActorRuntime, "t", "", false); err != nil {
			t.Fatalf("runtime expire should be allowed: %v", err)
		}
	})

	t.Run("owner_may_revoke_and_appeal", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-perm-owner-revoke")
		if _, err := revokeNote("note-perm-owner-revoke", pheromoneActorOwner, "t", "", false); err != nil {
			t.Fatalf("owner revoke should be allowed: %v", err)
		}
		seedRejectedPendingNote(t, "note-perm-owner-appeal")
		if _, err := appealNote("note-perm-owner-appeal", pheromoneActorOwner, "t", "", false); err != nil {
			t.Fatalf("owner appeal should be allowed: %v", err)
		}
	})
}

// --- Task 3: prove every declared action is real and every real action is declared ---

func TestEveryDeclaredActionIsReachable(t *testing.T) {
	funcs := parseCmdPackageFuncs(t)
	commands := map[string]*cobra.Command{
		"suggestApproveCmd":   suggestApproveCmd,
		"pheromoneDisplayCmd": pheromoneDisplayCmd,
	}

	for _, action := range pheromoneInfluenceActionNames() {
		spec, ok := pheromoneInfluenceActionSurface[action]
		if !ok {
			t.Fatalf("declared action %q has no entry in pheromoneInfluenceActionSurface -- every declared action must name its implementation and surface", action)
		}
		if _, ok := funcs[spec.Implementation]; !ok {
			t.Fatalf("declared action %q names implementation %q, but no such function exists in the cmd package", action, spec.Implementation)
		}
		command, ok := commands[spec.Command]
		if !ok {
			t.Fatalf("declared action %q names command %q, which is not a known owner-facing surface", action, spec.Command)
		}
		if command.Flags().Lookup(spec.Flag) == nil {
			t.Fatalf("declared action %q names flag --%s on %s, but no such flag is registered", action, spec.Flag, spec.Command)
		}
	}
}

// declaredActionFixture seeds whatever precondition an action needs (a plain
// signal, an already-rejected queued note, or a freshly queued suggestion)
// under the given note id, then performs the declared action through its
// REAL implementation function, reporting whether the note was found.
type declaredActionFixture struct {
	setup  func(t *testing.T, id string)
	invoke func(id, actorKind, actorName string) (found bool, err error)
}

// seedFreshQueuedNote seeds a plain, not-yet-decided queued suggestion under
// id -- the precondition accept/edit/reject each need. Unlike
// seedRejectedPendingNote, the item carries no Action/ActionAt yet.
func seedFreshQueuedNote(t *testing.T, id string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	origin := colony.PendingOriginSuggestion
	item := colony.PendingSuggestion{
		ID:          id,
		Type:        "FOCUS",
		Content:     "declared action fixture content",
		ContentHash: "sha256:" + sha256Sum("declared action fixture content"),
		CreatedAt:   now,
		Origin:      &origin,
	}
	items := []colony.PendingSuggestion{item}
	// Goal must be set: suggestApproveCmd's RunE (loadActiveColonyState)
	// silently short-circuits every flag -- approve, edit, dismiss alike --
	// to an empty listing when Goal is unset, matching a genuinely
	// uninitialized colony. A direct approvePendingNote/editPendingNote/
	// rejectPendingNote call bypasses that gate entirely, but this fixture
	// is also driven through the real suggest-approve command
	// (TestSuggestApproveRecordsActorThroughTheRealCommand), which does not.
	goal := "declared action fixture goal"
	cs := colony.ColonyState{Goal: &goal, PendingSuggestions: &items}
	if err := store.SaveJSON("COLONY_STATE.json", cs); err != nil {
		t.Fatalf("seed fresh queued note %q: %v", id, err)
	}
}

// declaredActionFixtures wires every one of pheromoneInfluenceActionNames()
// to a real precondition and a real invocation of its declared
// implementation function (pheromoneInfluenceActionSurface). A declared
// action with no entry here fails
// TestEveryDeclaredActionAppendsHistoryWithAnActor by name, so a future
// addition to pheromoneInfluenceActions cannot pass silently uncovered.
func declaredActionFixtures() map[string]declaredActionFixture {
	untilSoon := func() string { return time.Now().Add(time.Hour).UTC().Format(time.RFC3339) }
	signalFixture := func(invoke func(id, actorKind, actorName string) (bool, error)) declaredActionFixture {
		return declaredActionFixture{
			setup:  func(t *testing.T, id string) { seedPheromoneInfluenceSignal(t, id) },
			invoke: invoke,
		}
	}
	return map[string]declaredActionFixture{
		pheromoneActionAccepted: {
			setup: seedFreshQueuedNote,
			invoke: func(id, actorKind, actorName string) (bool, error) {
				r, err := approvePendingNote(id, actorKind, actorName, false)
				return r.Found, err
			},
		},
		pheromoneActionEdited: {
			setup: seedFreshQueuedNote,
			invoke: func(id, actorKind, actorName string) (bool, error) {
				r, err := editPendingNote(id, actorKind, actorName, "edited wording", false)
				return r.Found, err
			},
		},
		pheromoneActionRejected: {
			setup: seedFreshQueuedNote,
			invoke: func(id, actorKind, actorName string) (bool, error) {
				r, err := rejectPendingNote(id, actorKind, actorName, false)
				return r.Found, err
			},
		},
		pheromoneActionReinforced: signalFixture(func(id, actorKind, actorName string) (bool, error) {
			o, err := reinforceNote(id, actorKind, actorName, "reason", false)
			return o.Found, err
		}),
		pheromoneActionDeferred: signalFixture(func(id, actorKind, actorName string) (bool, error) {
			o, err := deferNote(id, actorKind, actorName, untilSoon(), "reason", false)
			return o.Found, err
		}),
		pheromoneActionExpired: signalFixture(func(id, actorKind, actorName string) (bool, error) {
			o, err := expireNote(id, actorKind, actorName, "reason", false)
			return o.Found, err
		}),
		pheromoneActionRevoked: signalFixture(func(id, actorKind, actorName string) (bool, error) {
			o, err := revokeNote(id, actorKind, actorName, "reason", false)
			return o.Found, err
		}),
		pheromoneActionAppealed: {
			setup: func(t *testing.T, id string) { seedRejectedPendingNote(t, id) },
			invoke: func(id, actorKind, actorName string) (bool, error) {
				o, err := appealNote(id, actorKind, actorName, "reason", false)
				return o.Found, err
			},
		},
		pheromoneActionWeakened: signalFixture(func(id, actorKind, actorName string) (bool, error) {
			o, err := weakenNote(id, actorKind, actorName, "reason", false)
			return o.Found, err
		}),
		pheromoneActionPinned: signalFixture(func(id, actorKind, actorName string) (bool, error) {
			o, err := pinNote(id, actorKind, actorName, "reason", false)
			return o.Found, err
		}),
		pheromoneActionUnpinned: signalFixture(func(id, actorKind, actorName string) (bool, error) {
			o, err := unpinNote(id, actorKind, actorName, "reason", false)
			return o.Found, err
		}),
	}
}

// TestEveryDeclaredActionAppendsHistoryWithAnActor is CR-03's core proof.
// TestEveryDeclaredActionIsReachable above only proves a function and a flag
// exist for each declared action -- it never proves performing the action
// leaves a durable record. This test drives the REAL implementation function
// for every one of pheromoneInfluenceActionNames() (never a hand-typed
// subset, so a future action added to that list is picked up automatically
// and fails loudly if nobody has wired a fixture for it yet) and asserts the
// one property BIO-08 actually promises: exactly one append-only history
// entry, naming that action and a declared actor.
func TestEveryDeclaredActionAppendsHistoryWithAnActor(t *testing.T) {
	const actorName = "reachability-test"
	fixtures := declaredActionFixtures()

	for _, action := range pheromoneInfluenceActionNames() {
		action := action
		t.Run(action, func(t *testing.T) {
			setupExchangeTest(t)
			fixture, ok := fixtures[action]
			if !ok {
				t.Fatalf("declared action %q has no fixture wired in TestEveryDeclaredActionAppendsHistoryWithAnActor -- add one so the history+actor guarantee is proven for it too", action)
			}
			id := "declared-history-" + action

			fixture.setup(t, id)

			before, err := readInfluenceHistory(id)
			if err != nil {
				t.Fatalf("read history before %q: %v", action, err)
			}
			if len(before) != 0 {
				t.Fatalf("expected no history before performing %q, got %+v", action, before)
			}

			found, err := fixture.invoke(id, pheromoneActorOwner, actorName)
			if err != nil {
				t.Fatalf("performing %q: %v", action, err)
			}
			if !found {
				t.Fatalf("performing %q: note %q was not found", action, id)
			}

			history, err := readInfluenceHistory(id)
			if err != nil {
				t.Fatalf("read history after %q: %v", action, err)
			}
			if len(history) != 1 {
				t.Fatalf("expected exactly one history entry after %q, got %d: %+v", action, len(history), history)
			}
			entry := history[0]
			if entry.Action != action {
				t.Fatalf("history entry action = %q, want %q", entry.Action, action)
			}
			if entry.ActorKind == "" || !pheromoneActorDeclared(entry.ActorKind) {
				t.Fatalf("history entry for %q recorded actor kind %q, want one of the declared kinds", action, entry.ActorKind)
			}
			if entry.ActorName != actorName {
				t.Fatalf("history entry for %q recorded actor name %q, want %q", action, entry.ActorName, actorName)
			}
		})
	}
}

// appendInfluenceHistoryLiteralActionCallSites returns the name of fn if it
// calls appendInfluenceHistory with a raw string literal as the action
// argument (the second parameter) instead of a declared constant.
func appendInfluenceHistoryLiteralActionCallSites(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "appendInfluenceHistory" {
			return true
		}
		if len(call.Args) < 2 {
			return true
		}
		if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			found = true
		}
		return true
	})
	return found
}

func TestEveryRecordedActionIsDeclared(t *testing.T) {
	funcs := parseCmdPackageFuncs(t)
	var offenders []string
	for name, fn := range funcs {
		if appendInfluenceHistoryLiteralActionCallSites(fn) {
			offenders = append(offenders, name)
		}
	}
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Fatalf("found a call to appendInfluenceHistory passing a raw string literal as its action argument (must use a declared pheromoneAction* constant): %v", offenders)
	}
}

func TestInfluenceSurfaceIsOwnerFacing(t *testing.T) {
	t.Run("revoke_refuses_non_owner_through_the_real_command", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-surface-revoke")

		rootCmd.SetArgs([]string{"pheromones", "--revoke", "note-surface-revoke", "--actor", pheromoneActorRuntime})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("pheromones --revoke returned a Go error: %v", err)
		}

		var pf colony.PheromoneFile
		if err := store.LoadJSON("pheromones.json", &pf); err != nil {
			t.Fatalf("load pheromones: %v", err)
		}
		if pf.Signals[0].RevokedAt != nil {
			t.Fatalf("non-owner actor revoked a note through the real command: %+v", pf.Signals[0])
		}
	})

	t.Run("appeal_refuses_non_owner_through_the_real_command", func(t *testing.T) {
		setupExchangeTest(t)
		seedRejectedPendingNote(t, "note-surface-appeal")

		rootCmd.SetArgs([]string{"pheromones", "--appeal", "note-surface-appeal", "--actor", pheromoneActorLearning})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("pheromones --appeal returned a Go error: %v", err)
		}

		history, err := readInfluenceHistory("note-surface-appeal")
		if err != nil {
			t.Fatalf("read history: %v", err)
		}
		if len(history) != 0 {
			t.Fatalf("non-owner actor recorded an appeal through the real command: %+v", history)
		}
	})

	t.Run("owner_revoke_succeeds_through_the_real_command", func(t *testing.T) {
		setupExchangeTest(t)
		seedPheromoneInfluenceSignal(t, "note-surface-owner-revoke")

		rootCmd.SetArgs([]string{"pheromones", "--revoke", "note-surface-owner-revoke"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("pheromones --revoke returned a Go error: %v", err)
		}

		var pf colony.PheromoneFile
		if err := store.LoadJSON("pheromones.json", &pf); err != nil {
			t.Fatalf("load pheromones: %v", err)
		}
		if pf.Signals[0].RevokedAt == nil {
			t.Fatalf("expected the default actor (owner) to succeed at revoking through the real command, got %+v", pf.Signals[0])
		}
	})
}

// TestSuggestApproveRecordsActorThroughTheRealCommand closes the loop CR-03
// requires end to end: suggest-approve's --actor/--actor-name flags (added
// alongside this fix, mirroring pheromoneDisplayCmd's existing five-action
// mechanism) actually reach the history entry, not just the internal
// approvePendingNote/editPendingNote/rejectPendingNote functions this file's
// other tests call directly.
func TestSuggestApproveRecordsActorThroughTheRealCommand(t *testing.T) {
	t.Run("approve_records_the_named_actor", func(t *testing.T) {
		setupExchangeTest(t)
		id := "note-suggest-approve-actor"
		seedFreshQueuedNote(t, id)

		rootCmd.SetArgs([]string{"suggest-approve", "--approve", id, "--actor", pheromoneActorRuntime, "--actor-name", "phase-end"})
		defer rootCmd.SetArgs([]string{})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("suggest-approve --approve returned a Go error: %v", err)
		}

		history, err := readInfluenceHistory(id)
		if err != nil {
			t.Fatalf("read history: %v", err)
		}
		if len(history) != 1 {
			t.Fatalf("expected exactly one history entry, got %d: %+v", len(history), history)
		}
		if history[0].Action != pheromoneActionAccepted {
			t.Fatalf("action = %q, want %q", history[0].Action, pheromoneActionAccepted)
		}
		if history[0].ActorKind != pheromoneActorRuntime {
			t.Fatalf("actor kind = %q, want %q", history[0].ActorKind, pheromoneActorRuntime)
		}
		if history[0].ActorName != "phase-end" {
			t.Fatalf("actor name = %q, want %q", history[0].ActorName, "phase-end")
		}
	})

	t.Run("dismiss_defaults_to_the_owner_when_no_actor_flag_is_given", func(t *testing.T) {
		setupExchangeTest(t)
		id := "note-suggest-approve-default-actor"
		seedFreshQueuedNote(t, id)

		rootCmd.SetArgs([]string{"suggest-approve", "--dismiss", id})
		defer rootCmd.SetArgs([]string{})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("suggest-approve --dismiss returned a Go error: %v", err)
		}

		history, err := readInfluenceHistory(id)
		if err != nil {
			t.Fatalf("read history: %v", err)
		}
		if len(history) != 1 {
			t.Fatalf("expected exactly one history entry, got %d: %+v", len(history), history)
		}
		if history[0].Action != pheromoneActionRejected {
			t.Fatalf("action = %q, want %q", history[0].Action, pheromoneActionRejected)
		}
		if history[0].ActorKind != pheromoneActorOwner {
			t.Fatalf("expected the default actor to be owner, got %q", history[0].ActorKind)
		}
	})

	t.Run("approve_with_edit_records_both_actions_with_the_actor", func(t *testing.T) {
		setupExchangeTest(t)
		id := "note-suggest-approve-edit-then-accept"
		seedFreshQueuedNote(t, id)

		rootCmd.SetArgs([]string{"suggest-approve", "--approve", id, "--edit", "owner's rewording", "--actor", pheromoneActorOwner, "--actor-name", "the-owner"})
		defer rootCmd.SetArgs([]string{})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("suggest-approve --approve --edit returned a Go error: %v", err)
		}

		history, err := readInfluenceHistory(id)
		if err != nil {
			t.Fatalf("read history: %v", err)
		}
		if len(history) != 2 {
			t.Fatalf("expected exactly two history entries (edited then accepted), got %d: %+v", len(history), history)
		}
		if history[0].Action != pheromoneActionEdited || history[1].Action != pheromoneActionAccepted {
			t.Fatalf("expected [edited, accepted] in order, got %+v", history)
		}
		for _, e := range history {
			if e.ActorName != "the-owner" {
				t.Fatalf("entry %+v did not record the actor name", e)
			}
		}
	})
}
