package cmd

// Tests for plan 203-13: outcome-weighted pheromone strength tuning
// (BIO-08/CEC-07). This is the plan the owner rated costly -- every test
// here exists to prove one of its three load-bearing guarantees: strength
// moves ONLY on recorded credit, an owner-pinned note is NEVER
// automatically tuned, and every automatic movement is traceable back to
// the credit record that justified it.

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Shared fixtures
// ---------------------------------------------------------------------------

// seedOutcomeNote writes (or replaces) a real pheromone signal at the given
// strength, the fixture every test in this file starts from.
func seedOutcomeNote(t *testing.T, id string, strength float64) {
	t.Helper()
	sig := colony.PheromoneSignal{
		ID:        id,
		Type:      "FOCUS",
		Priority:  "normal",
		Source:    "cli",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Active:    true,
		Strength:  &strength,
		Content:   json.RawMessage(`{"text":"outcome tuning test note"}`),
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
		t.Fatalf("seed outcome note %q: %v", id, err)
	}
}

// corruptCreditStoreAtDataDir overwrites credit/records.json with invalid
// JSON directly on disk, simulating "the credit store cannot be read" --
// distinct from "the credit store does not exist yet." Writing through the
// store itself (SaveRawJSON) is not usable here: pkg/storage's own
// atomicWriteLocked validates JSON on every *.json write and refuses invalid
// content by design, so a genuinely corrupted file can only be produced by
// writing to the filesystem path directly, bypassing the store.
func corruptCreditStoreAtDataDir(t *testing.T, dataDir string) {
	t.Helper()
	path := filepath.Join(dataDir, recruitmentCreditPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir for corrupt credit store: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not valid json"), 0644); err != nil {
		t.Fatalf("corrupt credit store: %v", err)
	}
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// ---------------------------------------------------------------------------
// Task 1: strength moves only on recorded outcomes, and only from those.
// ---------------------------------------------------------------------------

// TestNoteStrengthTuningConstantsAreDeclared proves the six step/bound
// values this plan's own acceptance criteria name are declared as constants
// in cmd/pheromone_outcome.go -- fails by name if any is removed and
// replaced with an inline literal.
func TestNoteStrengthTuningConstantsAreDeclared(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "pheromone_outcome.go", nil, 0)
	if err != nil {
		t.Fatalf("parse cmd/pheromone_outcome.go: %v", err)
	}
	declared := map[string]bool{}
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				declared[name.Name] = true
			}
		}
	}
	for _, want := range []string{
		"noteStrengthHelpfulStep", "noteStrengthNeutralStep", "noteStrengthHarmfulStep",
		"noteStrengthFloor", "noteStrengthCeiling", "noteHarmfulQuarantineThreshold",
	} {
		if !declared[want] {
			t.Fatalf("expected %s to be declared as a named constant in cmd/pheromone_outcome.go", want)
		}
	}
}

// TestNoteStrengthTuningNoRecordsLeavesNoteUntouched: "A note with no credit
// records at all is untouched, even after a phase passes."
func TestNoteStrengthTuningNoRecordsLeavesNoteUntouched(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	seedOutcomeNote(t, "note-untouched", 0.5)
	before, err := store.LoadRawJSON("pheromones.json")
	if err != nil {
		t.Fatalf("load before: %v", err)
	}

	result := tuneNoteStrengthFromOutcomes()
	if !result.Ran {
		t.Fatalf("expected Ran=true with no records to consider, got %+v", result)
	}
	if result.NotesTuned != 0 {
		t.Fatalf("expected no notes tuned, got %+v", result)
	}

	after, err := store.LoadRawJSON("pheromones.json")
	if err != nil {
		t.Fatalf("load after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("pheromones.json changed with no credit records at all:\nbefore=%s\nafter=%s", before, after)
	}
}

// TestNoteStrengthTuningHelpfulNeutralHarmfulMovements: "Helpful, neutral and
// harmful each produce the declared movement, asserted against the
// constants rather than typed numbers."
func TestNoteStrengthTuningHelpfulNeutralHarmfulMovements(t *testing.T) {
	cases := []struct {
		name    string
		outcome recruitmentCreditOutcome
		step    float64
	}{
		{"helpful", recruitmentCreditOutcomeHelpful, noteStrengthHelpfulStep},
		{"neutral", recruitmentCreditOutcomeNeutral, noteStrengthNeutralStep},
		{"harmful", recruitmentCreditOutcomeHarmful, noteStrengthHarmfulStep},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			s, tmpDir := newTestStore(t)
			defer os.RemoveAll(tmpDir)
			store = s

			noteID := "note-" + tc.name
			seedOutcomeNote(t, noteID, 0.50)
			_, decisionID := newCreditTrophallaxisFixture(t, tc.name)
			rec, credited, err := recordRecruitmentCredit(noteID, recruitmentContributionNote, decisionID, "evidence-"+tc.name, tc.outcome, "")
			if err != nil || !credited {
				t.Fatalf("record credit: %v credited=%v", err, credited)
			}

			result := tuneNoteStrengthFromOutcomes()
			if !result.Ran {
				t.Fatalf("expected Ran=true, got %+v", result)
			}
			if result.NotesTuned != 1 {
				t.Fatalf("expected exactly one note tuned, got %+v", result)
			}

			pf, idx, found := findPheromoneSignalByID(noteID)
			if !found {
				t.Fatalf("note %q missing after tuning", noteID)
			}
			got := *pf.Signals[idx].Strength
			want, _ := clampNoteStrength(0.50 + tc.step)
			if !almostEqual(got, want) {
				t.Fatalf("strength = %.4f, want %.4f (0.50 %+.4f)", got, want, tc.step)
			}

			history, err := readInfluenceHistory(noteID)
			if err != nil {
				t.Fatalf("read history: %v", err)
			}
			if len(history) != 1 {
				t.Fatalf("expected exactly one history entry, got %d: %+v", len(history), history)
			}
			entry := history[0]
			if entry.ActorKind != pheromoneActorLearning {
				t.Fatalf("entry actor kind = %q, want %q", entry.ActorKind, pheromoneActorLearning)
			}
			if entry.Reason != rec.RecordID {
				t.Fatalf("entry reason = %q, want the credit record id %q", entry.Reason, rec.RecordID)
			}
		})
	}
}

// TestNoteStrengthTuningQuarantinesOnHarmfulThreshold: "Reaching the
// declared harmful threshold quarantines the note automatically and appends
// a history entry naming the threshold."
func TestNoteStrengthTuningQuarantinesOnHarmfulThreshold(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	noteID := "note-quarantine"
	seedOutcomeNote(t, noteID, 0.90)
	for i := 0; i < noteHarmfulQuarantineThreshold; i++ {
		_, decisionID := newCreditTrophallaxisFixture(t, fmt.Sprintf("q%d", i))
		if _, credited, err := recordRecruitmentCredit(noteID, recruitmentContributionNote, decisionID, fmt.Sprintf("evidence-q%d", i), recruitmentCreditOutcomeHarmful, ""); err != nil || !credited {
			t.Fatalf("record credit %d: %v credited=%v", i, err, credited)
		}
	}

	result := tuneNoteStrengthFromOutcomes()
	if !result.Ran {
		t.Fatalf("expected Ran=true, got %+v", result)
	}
	if len(result.Quarantined) != 1 || result.Quarantined[0] != noteID {
		t.Fatalf("expected %q to be quarantined, got %+v", noteID, result)
	}

	pf, idx, found := findPheromoneSignalByID(noteID)
	if !found {
		t.Fatalf("note missing")
	}
	if pf.Signals[idx].Quarantined == nil || !*pf.Signals[idx].Quarantined {
		t.Fatalf("expected note to be quarantined, got %+v", pf.Signals[idx])
	}

	history, err := readInfluenceHistory(noteID)
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	sawThreshold := false
	for _, e := range history {
		if strings.Contains(e.After, "quarantined=true") && strings.Contains(e.After, fmt.Sprintf("threshold=%d", noteHarmfulQuarantineThreshold)) {
			sawThreshold = true
		}
	}
	if !sawThreshold {
		t.Fatalf("expected a history entry naming the quarantine threshold, got %+v", history)
	}
}

// TestNoteStrengthTuningIsIdempotent: "Two runs over the same credit records
// produce the same result; tuning is idempotent against an already-consumed
// record" / "Running the tuning pass twice over the same credit records
// produces a byte-identical store after the second run."
func TestNoteStrengthTuningIsIdempotent(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	noteID := "note-idempotent"
	seedOutcomeNote(t, noteID, 0.50)
	_, decisionID := newCreditTrophallaxisFixture(t, "idempotent")
	if _, credited, err := recordRecruitmentCredit(noteID, recruitmentContributionNote, decisionID, "evidence-idempotent", recruitmentCreditOutcomeHelpful, ""); err != nil || !credited {
		t.Fatalf("record credit: %v credited=%v", err, credited)
	}

	first := tuneNoteStrengthFromOutcomes()
	if first.NotesTuned != 1 {
		t.Fatalf("first pass: expected 1 note tuned, got %+v", first)
	}

	pheromonesAfterFirst, err := store.LoadRawJSON("pheromones.json")
	if err != nil {
		t.Fatalf("load pheromones after first pass: %v", err)
	}
	historyAfterFirst, err := store.LoadRawJSON("pheromones-history.json")
	if err != nil {
		t.Fatalf("load history after first pass: %v", err)
	}

	second := tuneNoteStrengthFromOutcomes()
	if second.NotesTuned != 0 {
		t.Fatalf("second pass: expected 0 notes tuned (idempotent replay), got %+v", second)
	}

	pheromonesAfterSecond, err := store.LoadRawJSON("pheromones.json")
	if err != nil {
		t.Fatalf("load pheromones after second pass: %v", err)
	}
	historyAfterSecond, err := store.LoadRawJSON("pheromones-history.json")
	if err != nil {
		t.Fatalf("load history after second pass: %v", err)
	}

	if string(pheromonesAfterFirst) != string(pheromonesAfterSecond) {
		t.Fatalf("pheromones.json changed on a second pass over the same credit records")
	}
	if string(historyAfterFirst) != string(historyAfterSecond) {
		t.Fatalf("pheromones-history.json changed on a second pass over the same credit records")
	}
}

// TestNoteStrengthTuningClampRecordsClamp: "A clamped computation records
// the clamp in the history entry."
func TestNoteStrengthTuningClampRecordsClamp(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	noteID := "note-clamp"
	seedOutcomeNote(t, noteID, 0.95) // + helpful step would exceed the ceiling
	_, decisionID := newCreditTrophallaxisFixture(t, "clamp")
	if _, credited, err := recordRecruitmentCredit(noteID, recruitmentContributionNote, decisionID, "evidence-clamp", recruitmentCreditOutcomeHelpful, ""); err != nil || !credited {
		t.Fatalf("record credit: %v credited=%v", err, credited)
	}

	result := tuneNoteStrengthFromOutcomes()
	if result.NotesTuned != 1 {
		t.Fatalf("expected 1 note tuned, got %+v", result)
	}

	pf, idx, found := findPheromoneSignalByID(noteID)
	if !found {
		t.Fatalf("note missing")
	}
	if !almostEqual(*pf.Signals[idx].Strength, noteStrengthCeiling) {
		t.Fatalf("strength = %.4f, want the ceiling %.4f", *pf.Signals[idx].Strength, noteStrengthCeiling)
	}

	history, err := readInfluenceHistory(noteID)
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	if len(history) != 1 || !strings.Contains(history[0].After, "clamped") {
		t.Fatalf("expected the history entry to record the clamp, got %+v", history)
	}
}

// ---------------------------------------------------------------------------
// Task 2: owner decisions outrank learning, and the pass never blocks a
// phase.
// ---------------------------------------------------------------------------

// TestPinnedNoteIsNeverTuned covers both Task 2 behaviors naming pin: "A
// pinned note is skipped by every tuning pass in both directions and the
// skip is recorded" and "Pinning is an owner action ... the runtime cannot
// pin or unpin."
func TestPinnedNoteIsNeverTuned(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	noteID := "note-pinned"
	seedOutcomeNote(t, noteID, 0.50)
	if _, err := pinNote(noteID, pheromoneActorOwner, "owner-tester", "keep this steady", false); err != nil {
		t.Fatalf("pinNote (owner): %v", err)
	}

	if _, err := pinNote(noteID, pheromoneActorRuntime, "", "", false); err == nil {
		t.Fatal("expected a runtime actor's pin attempt to be refused")
	} else if !strings.Contains(err.Error(), pheromoneActorRuntime) {
		t.Fatalf("refusal %q does not name the actor kind %q", err.Error(), pheromoneActorRuntime)
	}

	// A harmful record too, not only helpful -- "in both directions".
	_, decisionIDHelpful := newCreditTrophallaxisFixture(t, "pinned-helpful")
	if _, credited, err := recordRecruitmentCredit(noteID, recruitmentContributionNote, decisionIDHelpful, "evidence-pinned-helpful", recruitmentCreditOutcomeHelpful, ""); err != nil || !credited {
		t.Fatalf("record helpful credit: %v credited=%v", err, credited)
	}
	_, decisionIDHarmful := newCreditTrophallaxisFixture(t, "pinned-harmful")
	if _, credited, err := recordRecruitmentCredit(noteID, recruitmentContributionNote, decisionIDHarmful, "evidence-pinned-harmful", recruitmentCreditOutcomeHarmful, ""); err != nil || !credited {
		t.Fatalf("record harmful credit: %v credited=%v", err, credited)
	}

	result := tuneNoteStrengthFromOutcomes()
	if !result.Ran {
		t.Fatalf("expected Ran=true, got %+v", result)
	}
	if result.NotesTuned != 0 {
		t.Fatalf("expected the pinned note to be skipped, not tuned: %+v", result)
	}
	if len(result.SkippedPinned) != 2 {
		t.Fatalf("expected both credit records against the pinned note to be recorded as skipped, got %+v", result)
	}

	pf, idx, found := findPheromoneSignalByID(noteID)
	if !found {
		t.Fatalf("note missing")
	}
	if !almostEqual(*pf.Signals[idx].Strength, 0.50) {
		t.Fatalf("pinned note's strength changed: got %.4f, want 0.50", *pf.Signals[idx].Strength)
	}

	history, err := readInfluenceHistory(noteID)
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	skips := 0
	for _, e := range history {
		if strings.Contains(e.Reason, "pinned") {
			skips++
		}
	}
	if skips != 2 {
		t.Fatalf("expected two history entries naming the pinned skip, got %d in %+v", skips, history)
	}
}

// TestTuningNeverBlocksAPhase: "A tuning pass that cannot read the credit
// store records that it could not run and leaves every note unchanged" and
// "a failure inside it is logged and never propagated ... asserted on both
// check lanes."
func TestTuningNeverBlocksAPhase(t *testing.T) {
	t.Run("a store-level failure records could-not-run and changes nothing", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		seedOutcomeNote(t, "note-unreadable", 0.50)
		corruptCreditStoreAtDataDir(t, filepath.Join(tmpDir, ".aether", "data"))

		before, err := store.LoadRawJSON("pheromones.json")
		if err != nil {
			t.Fatalf("load before: %v", err)
		}

		result := tuneNoteStrengthFromOutcomes()
		if result.Ran {
			t.Fatalf("expected Ran=false when the credit store cannot be read, got %+v", result)
		}
		if result.Error == "" {
			t.Fatal("expected a recorded error naming why the pass could not run")
		}

		after, err := store.LoadRawJSON("pheromones.json")
		if err != nil {
			t.Fatalf("load after: %v", err)
		}
		if string(before) != string(after) {
			t.Fatal("pheromones.json changed even though the pass could not run")
		}
	})

	cases := []struct {
		name string
		run  func(t *testing.T, root string) map[string]interface{}
	}{
		{name: "fast lane", run: driveFastLaneContinue},
		{name: "wrapper lane", run: driveWrapperLaneContinue},
	}
	for _, tc := range cases {
		t.Run(tc.name+" still advances when the credit store is unreadable", func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)

			root := setupPhaseEndHiveContinueFixture(t, "Tuning failure never blocks a phase, "+tc.name)
			corruptCreditStoreAtDataDir(t, filepath.Join(root, ".aether", "data"))

			result := tc.run(t, root)
			if advanced, _ := result["advanced"].(bool); !advanced {
				t.Fatalf("lane %q: expected the phase to still advance despite a tuning failure, got %v", tc.name, result)
			}

			tuning, ok := result["pheromone_outcome_tuning"].(map[string]interface{})
			if !ok {
				t.Fatalf("lane %q: expected pheromone_outcome_tuning in result, got %#v", tc.name, result["pheromone_outcome_tuning"])
			}
			if ran, _ := tuning["ran"].(bool); ran {
				t.Fatalf("lane %q: expected ran=false with an unreadable credit store, got %+v", tc.name, tuning)
			}
			if errMsg, _ := tuning["error"].(string); errMsg == "" {
				t.Fatalf("lane %q: expected a recorded error in the tuning summary, got %+v", tc.name, tuning)
			}
		})
	}
}

// TestStrengthOriginIsReadable: "The owner can see, for any note, whether
// its current strength was last set by them or learned."
func TestStrengthOriginIsReadable(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	ownerNote := "note-origin-owner"
	seedOutcomeNote(t, ownerNote, 0.50)
	if _, err := reinforceNote(ownerNote, pheromoneActorOwner, "owner-tester", "manual boost", false); err != nil {
		t.Fatalf("reinforceNote: %v", err)
	}
	origin, err := pheromoneStrengthOrigin(ownerNote)
	if err != nil {
		t.Fatalf("pheromoneStrengthOrigin: %v", err)
	}
	if origin != pheromoneActorOwner {
		t.Fatalf("origin = %q, want %q for an owner-reinforced note", origin, pheromoneActorOwner)
	}

	learnedNote := "note-origin-learning"
	seedOutcomeNote(t, learnedNote, 0.50)
	_, decisionID := newCreditTrophallaxisFixture(t, "origin")
	if _, credited, err := recordRecruitmentCredit(learnedNote, recruitmentContributionNote, decisionID, "evidence-origin", recruitmentCreditOutcomeHelpful, ""); err != nil || !credited {
		t.Fatalf("record credit: %v credited=%v", err, credited)
	}
	if result := tuneNoteStrengthFromOutcomes(); result.NotesTuned != 1 {
		t.Fatalf("expected the learning pass to tune exactly one note, got %+v", result)
	}
	origin2, err := pheromoneStrengthOrigin(learnedNote)
	if err != nil {
		t.Fatalf("pheromoneStrengthOrigin: %v", err)
	}
	if origin2 != pheromoneActorLearning {
		t.Fatalf("origin = %q, want %q for a tuned note", origin2, pheromoneActorLearning)
	}
}

// ---------------------------------------------------------------------------
// Task 3: prove tuning is caller-driven, evidence-driven and harmless on
// failure.
// ---------------------------------------------------------------------------

// TestBothCheckLanesTuneNotes: "Both check lanes reach the tuning pass,
// derived from the declared lane inventory rather than a hand-typed list" --
// reuses the exact two-lane inventory (driveFastLaneContinue,
// driveWrapperLaneContinue) cmd/phase_end_hive_test.go's own
// TestStrongInstinctReachesTheSharedStoreAtCheck already established as this
// codebase's declared check-lane inventory.
func TestBothCheckLanesTuneNotes(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T, root string) map[string]interface{}
	}{
		{name: "fast lane", run: driveFastLaneContinue},
		{name: "wrapper lane", run: driveWrapperLaneContinue},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)

			root := setupPhaseEndHiveContinueFixture(t, "Both check lanes tune notes, "+tc.name)
			noteID := "note-lane-" + strings.ReplaceAll(tc.name, " ", "-")
			seedOutcomeNote(t, noteID, 0.50)
			_, decisionID := newCreditTrophallaxisFixture(t, "lane-"+strings.ReplaceAll(tc.name, " ", "-"))
			if _, credited, err := recordRecruitmentCredit(noteID, recruitmentContributionNote, decisionID, "evidence-lane", recruitmentCreditOutcomeHelpful, ""); err != nil || !credited {
				t.Fatalf("record credit: %v credited=%v", err, credited)
			}

			result := tc.run(t, root)

			tuning, ok := result["pheromone_outcome_tuning"].(map[string]interface{})
			if !ok {
				t.Fatalf("lane %q: expected pheromone_outcome_tuning in result, got %#v", tc.name, result["pheromone_outcome_tuning"])
			}
			notesTuned, _ := tuning["notes_tuned"].(int)
			if notesTuned < 1 {
				t.Fatalf("lane %q did not reach the tuning pass -- pheromone_outcome_tuning = %+v", tc.name, tuning)
			}

			pf, idx, found := findPheromoneSignalByID(noteID)
			if !found {
				t.Fatalf("lane %q: note %q missing after continue", tc.name, noteID)
			}
			if *pf.Signals[idx].Strength <= 0.50 {
				t.Fatalf("lane %q: expected the note's strength to rise above 0.50, got %.4f", tc.name, *pf.Signals[idx].Strength)
			}
		})
	}
}

// appendInfluenceHistoryLearningActorCallSites reports whether fn contains a
// call to appendInfluenceHistory whose third argument (actorKind) is the
// pheromoneActorLearning identifier.
func appendInfluenceHistoryLearningActorCallSites(fn *ast.FuncDecl) bool {
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
		if len(call.Args) < 3 {
			return true
		}
		actorArg, ok := call.Args[2].(*ast.Ident)
		if ok && actorArg.Name == "pheromoneActorLearning" {
			found = true
		}
		return true
	})
	return found
}

// TestOnlyTheTuningPassLearnsStrength: "No function other than the tuning
// pass changes a note's strength with actor kind learning" -- an AST guard
// failing by name for any other function appending a strength change with
// actor kind learning.
func TestOnlyTheTuningPassLearnsStrength(t *testing.T) {
	exempt := map[string]bool{
		"tuneNoteStrengthFromOutcomes": true,
	}
	funcs := parseCmdPackageFuncs(t)
	var offenders []string
	for name, fn := range funcs {
		if exempt[name] {
			continue
		}
		if appendInfluenceHistoryLearningActorCallSites(fn) {
			offenders = append(offenders, name)
		}
	}
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Fatalf("found a function other than tuneNoteStrengthFromOutcomes appending a strength change with actor kind learning: %v", offenders)
	}
}

// TestDeliveryWithoutCreditChangesNothing: "A note delivered into a brief
// with a passing phase and no credit record is unchanged" -- drives a real
// note into a real resolved worker brief through a real passing verification
// step, mirroring cmd/recruitment_credit_test.go's
// TestDeliveredPlusPassingIsNotCredit for this plan's own tuning pass.
func TestDeliveryWithoutCreditChangesNothing(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	signal, _, err := writePheromoneSignal("FOCUS", "pay attention during outcome tuning", "normal", "test", "", "", 0, nil)
	if err != nil {
		t.Fatalf("write a real note: %v", err)
	}

	brief := resolvePheromoneSection()
	if !strings.Contains(brief, "pay attention during outcome tuning") {
		t.Fatalf("note was not actually delivered into the resolved worker brief:\n%s", brief)
	}

	step := runVerificationStep(context.Background(), tmpDir, "outcome-tuning-boundary-check", true, "true", 5*time.Second)
	if !step.Passed {
		t.Fatalf("expected the phase's own check to pass, got %+v", step)
	}

	before, err := store.LoadRawJSON("pheromones.json")
	if err != nil {
		t.Fatalf("load before: %v", err)
	}

	result := tuneNoteStrengthFromOutcomes()
	if result.NotesTuned != 0 {
		t.Fatalf("a delivered note plus a passing phase must never itself move strength, got %+v (signal %s)", result, signal.ID)
	}

	after, err := store.LoadRawJSON("pheromones.json")
	if err != nil {
		t.Fatalf("load after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("pheromones.json changed even though no credit record exists")
	}
}

// buildPathSourceFiles returns every non-test .go source file in the cmd
// package directory whose name marks it as part of the build path (the
// "codex_build*" and "build_*" families), the real files
// cmd/codex_build.go/cmd/build_flow_cmds.go and friends live in.
func buildPathSourceFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	for _, pattern := range []string{"codex_build*.go", "build_*.go"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob %s: %v", pattern, err)
		}
		files = append(files, matches...)
	}
	var nonTest []string
	for _, f := range files {
		if !strings.HasSuffix(f, "_test.go") {
			nonTest = append(nonTest, f)
		}
	}
	sort.Strings(nonTest)
	return nonTest
}

// TestTuningIsNotOnTheBuildPath: "The tuning pass is not reachable from a
// build path, only from a check path" -- an ordinary build pays nothing for
// this feature.
func TestTuningIsNotOnTheBuildPath(t *testing.T) {
	files := buildPathSourceFiles(t)
	if len(files) == 0 {
		t.Fatal("fixture is broken: no build-path source files found to scan")
	}

	fset := token.NewFileSet()
	for _, name := range files {
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			if ident.Name == "tuneNoteStrengthFromOutcomes" || ident.Name == "runPheromoneOutcomeTuning" {
				t.Fatalf("%s calls %s -- outcome-weighted tuning must only be reached from a check lane, never a build path", name, ident.Name)
			}
			return true
		})
	}
}
