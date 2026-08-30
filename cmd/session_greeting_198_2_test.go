package cmd

// Phase 198.2 plan 03 -- the session-start greeting now carries what the
// colony remembers about the owner.
//
// Task 1 settles the ranking rule the greeting and the status dashboard will
// share: CONTEXT.md described the dashboard as calling a most-recent loader
// named loadRecentRuntimeInstincts; the current source has no such function
// -- cmd/status.go already calls loadStrongestRuntimeInstincts
// (cmd/instinct_runtime.go), which ranks by usefulness score, not recency.
// This file's first test settles the discrepancy by proving the real
// behaviour, and locks the one-ranking-rule invariant so a second selector
// cannot be added silently.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// Task 1 -- one ranking rule for "strongest instincts", shared by the
// greeting card and the status dashboard.
// ---------------------------------------------------------------------------

// promoteRealInstinctVaried promotes an instinct through the real
// memory.PromoteService, with the source/evidence/observation-count
// combination the caller chooses -- never a hand-typed colony.InstinctEntry
// literal (D-04). content must satisfy memory.IsAdmissibleInstinctContent.
func promoteRealInstinctVaried(t *testing.T, s *storage.Store, content, sourceType, evidenceType string, observationCount int) colony.InstinctEntry {
	t.Helper()
	bus := events.NewBus(s, events.DefaultConfig())
	now := time.Now().UTC()
	hash := sha256.Sum256([]byte(content + ":" + sourceType + ":" + evidenceType))
	obs := colony.Observation{
		ContentHash:      "sha256:" + hex.EncodeToString(hash[:]),
		Content:          content,
		WisdomType:       "pattern",
		ObservationCount: observationCount,
		FirstSeen:        events.FormatTimestamp(now),
		LastSeen:         events.FormatTimestamp(now),
		Colonies:         []string{"test-colony"},
		SourceType:       sourceType,
		EvidenceType:     evidenceType,
	}
	svc := memory.NewPromoteService(s, bus)
	result, err := svc.Promote(context.Background(), obs, "test-colony")
	if err != nil {
		t.Fatalf("promote real instinct for %q: %v", content, err)
	}
	return result.Instinct
}

// instinctRankingSite is one place in a scanned directory that calls
// memory.InstinctUsefulnessScore -- the one instinct-ranking rule this test
// locks (198.2 plan 03).
type instinctRankingSite struct {
	File     string
	Function string
}

// scanInstinctRankingSource parses every non-test .go file directly under
// dir and finds every call to memory.InstinctUsefulnessScore, attributed to
// its enclosing top-level function. Mirrors scanNextActionHardcodeSource's
// shape (cmd/next_action_hardcode_ratchet_test.go): read the directory,
// parse each file with go/parser (a syntax parse -- it needs no resolved
// imports, so a synthetic fixture file need not actually import anything),
// and walk each top-level function body for the call this test cares about.
func scanInstinctRankingSource(dir string) ([]instinctRankingSite, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var sites []instinctRankingSite
	fset := token.NewFileSet()
	filesScanned := 0

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", name, readErr)
		}
		file, parseErr := parser.ParseFile(fset, path, src, 0)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", name, parseErr)
		}
		filesScanned++

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if pkgIdent.Name == "memory" && sel.Sel.Name == "InstinctUsefulnessScore" {
					sites = append(sites, instinctRankingSite{File: name, Function: fn.Name.Name})
				}
				return true
			})
		}
	}

	if filesScanned == 0 {
		return nil, fmt.Errorf("scanned zero non-test .go files in %s", dir)
	}
	return sites, nil
}

// TestStrongestHabitsHaveOneRankingRule is 198.2 plan 03 task 1's proof.
//
// CONTEXT.md described the status dashboard as calling a most-recent loader
// named loadRecentRuntimeInstincts; the current source has no such function
// -- cmd/status.go already calls loadStrongestRuntimeInstincts
// (cmd/instinct_runtime.go), which ranks by usefulness score, not recency.
// This test settles the discrepancy by proving the real behaviour rather
// than trusting either document: the one loader ranks strongest-first, its
// ties are broken deterministically, and no second selector can be added
// without this test catching it by name.
func TestStrongestHabitsHaveOneRankingRule(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// (a) Highest confidence/trust first.
	strong := promoteRealInstinctVaried(t, s,
		"Run `go test ./cmd/...` before claiming a fix works.",
		"user_feedback", "test_verified", 5)
	weak := promoteRealInstinctVaried(t, s,
		"Check the error log at cmd/server.go for the panic before restarting.",
		"heuristic", "anecdotal", 1)

	ranked := loadStrongestRuntimeInstincts(s, &colony.ColonyState{}, 2)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 ranked instincts, got %d: %+v", len(ranked), ranked)
	}
	if ranked[0].ID != strong.ID || ranked[1].ID != weak.ID {
		t.Fatalf("expected the higher-confidence/trust instinct first; got order %s, %s want %s, %s",
			ranked[0].ID, ranked[1].ID, strong.ID, weak.ID)
	}

	// (b) An exact tie is broken deterministically, and stays broken the
	// same way across repeated calls.
	tieA := promoteRealInstinctVaried(t, s,
		"Read pkg/memory/promote.go before changing the trust score formula.",
		"success_pattern", "single_phase", 1)
	tieB := promoteRealInstinctVaried(t, s,
		"Verify `git status` is clean before starting a new task.",
		"success_pattern", "single_phase", 1)

	// tieA and tieB share source, evidence and observation count, so their
	// trust score and confidence are byte-identical. The one thing real
	// Promote() calls a few milliseconds apart cannot guarantee is an
	// identical CreatedAt, and a tiny freshness gap would (correctly) break
	// the tie on its own -- proving nothing about the ID tie-break this
	// sub-test exists to check. Aligning CreatedAt here forces the genuine
	// tie a fast enough machine would already have produced.
	if tieA.Provenance.CreatedAt != tieB.Provenance.CreatedAt {
		var file colony.InstinctsFile
		if err := s.LoadJSON("instincts.json", &file); err != nil {
			t.Fatalf("reload instincts.json: %v", err)
		}
		for i := range file.Instincts {
			if file.Instincts[i].ID == tieB.ID {
				file.Instincts[i].Provenance.CreatedAt = tieA.Provenance.CreatedAt
			}
		}
		if err := s.SaveJSON("instincts.json", file); err != nil {
			t.Fatalf("align tie timestamps: %v", err)
		}
	}

	wantFirst, wantSecond := tieA.ID, tieB.ID
	if tieB.ID < tieA.ID {
		wantFirst, wantSecond = tieB.ID, tieA.ID
	}

	for call := 0; call < 2; call++ {
		got := loadStrongestRuntimeInstincts(s, &colony.ColonyState{}, 4)
		idxA, idxB := -1, -1
		for i, inst := range got {
			switch inst.ID {
			case tieA.ID:
				idxA = i
			case tieB.ID:
				idxB = i
			}
		}
		if idxA == -1 || idxB == -1 {
			t.Fatalf("call %d: the tied instincts did not both come back: %+v", call, got)
		}
		gotFirst, gotSecond := tieA.ID, tieB.ID
		if idxB < idxA {
			gotFirst, gotSecond = tieB.ID, tieA.ID
		}
		if gotFirst != wantFirst || gotSecond != wantSecond {
			t.Fatalf("call %d: the tie was not broken by instinct ID ascending; got order %s, %s want %s, %s",
				call, gotFirst, gotSecond, wantFirst, wantSecond)
		}
	}

	// The CONTEXT.md-named function does not exist -- the discrepancy this
	// task's action resolved, checked directly rather than trusted.
	t.Run("loadRecentRuntimeInstincts does not exist", func(t *testing.T) {
		entries, err := os.ReadDir(".")
		if err != nil {
			t.Fatalf("read cmd/: %v", err)
		}
		needle := "func " + "loadRecentRuntimeInstincts"
		for _, entry := range entries {
			name := entry.Name()
			// Test files are excluded, the same way scanInstinctRankingSource
			// excludes them above -- otherwise this loop would find the
			// needle inside this very file's own source, which quotes it to
			// build the string being searched for.
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			data, readErr := os.ReadFile(name)
			if readErr != nil {
				continue
			}
			if strings.Contains(string(data), needle) {
				t.Fatalf("cmd/%s defines loadRecentRuntimeInstincts -- CONTEXT.md's description was not actually superseded", name)
			}
		}
	})

	// (c) No second function may rank instincts for display.
	t.Run("no second ranking function exists in cmd/", func(t *testing.T) {
		sites, err := scanInstinctRankingSource(".")
		if err != nil {
			t.Fatalf("scan cmd/ for instinct-ranking sites: %v", err)
		}
		for _, site := range sites {
			if site.Function != "rankedInstinctEntries" {
				t.Fatalf("a second instinct-ranking function was found: %s (%s) -- only rankedInstinctEntries "+
					"may call memory.InstinctUsefulnessScore to rank instincts for display", site.Function, site.File)
			}
		}
	})

	// The guard on the guard: a planted second ranking function, in a clean
	// synthetic file the real cmd/ tree never sees, must be caught by name.
	t.Run("the scan can fail", func(t *testing.T) {
		tmp := t.TempDir()
		planted := "package cmd\n\n" +
			"func secondSelector(file colony.InstinctsFile, now time.Time) []colony.InstinctEntry {\n" +
			"\t_ = memory.InstinctUsefulnessScore(file.Instincts[0], now)\n" +
			"\treturn nil\n" +
			"}\n"
		if err := os.WriteFile(filepath.Join(tmp, "synthetic.go"), []byte(planted), 0644); err != nil {
			t.Fatalf("write synthetic fixture: %v", err)
		}
		plantedSites, err := scanInstinctRankingSource(tmp)
		if err != nil {
			t.Fatalf("scan synthetic fixture: %v", err)
		}
		found := false
		for _, site := range plantedSites {
			if site.Function == "secondSelector" {
				found = true
			}
		}
		if !found {
			t.Fatalf("a planted second ranking function in a clean synthetic file was not detected; found: %+v", plantedSites)
		}
	})
}

// ---------------------------------------------------------------------------
// Task 2 -- preferences, habits and the last relay note reach the greeting.
// ---------------------------------------------------------------------------

// greetingFixtureState is a minimal in-progress project, the same shape the
// existing session-start tests use.
func greetingFixtureState(t *testing.T) colony.ColonyState {
	t.Helper()
	return normalizedFixtureState(t, colony.ColonyState{
		Version:        "1.0",
		Goal:           fixtureGoal("Ship the billing rewrite"),
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: runningBuildStartedAt(),
		Milestone:      "Open Chambers",
		Plan:           colony.Plan{Phases: []colony.Phase{fixturePhase(1, "Foundations", colony.PhaseInProgress)}},
	})
}

// writeHubPreferences writes preference lines into the isolated hub's
// QUEEN.md, in the exact shape readUserPreferences parses
// (cmd/context.go:1609) -- the same file /ant-preferences appends to.
func writeHubPreferences(t *testing.T, prefs ...string) {
	t.Helper()
	hubDir := strings.TrimSpace(os.Getenv("AETHER_HUB_DIR"))
	if hubDir == "" {
		t.Fatal("AETHER_HUB_DIR is not set -- call newSessionStartProject first")
	}
	if err := os.MkdirAll(hubDir, 0o755); err != nil {
		t.Fatalf("make hub dir: %v", err)
	}
	var b strings.Builder
	b.WriteString("# QUEEN.md\n\n## User Preferences\n")
	for _, p := range prefs {
		b.WriteString("- ")
		b.WriteString(p)
		b.WriteString("\n")
	}
	if err := os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write hub QUEEN.md: %v", err)
	}
}

// TestGreetingCarriesPreferencesHabitsAndTheLastNote is the first
// <behavior> row: given a colony with preferences, habits and relay notes,
// the greeting carries a preferences line naming one example, exactly 3
// habit sentences in the habits' own wording, and one relay line naming the
// last helper and what it left.
func TestGreetingCarriesPreferencesHabitsAndTheLastNote(t *testing.T) {
	newSessionStartProject(t)

	writeHubPreferences(t,
		"Plain English replies, no jargon.",
		"Keep answers short unless asked to go deeper.",
		"Prefer Go idioms over cleverness.",
		"Always show proof, not narration.",
	)

	state := greetingFixtureState(t)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write fixture state: %v", err)
	}

	// Five habits, deliberately spread in strength (source, evidence AND
	// observation count all vary) so the top 3 are unambiguous regardless
	// of the microscopic freshness gap between real Promote() calls.
	h1 := promoteRealInstinctVaried(t, store, "Run `go test ./cmd/...` before claiming a fix works.", "user_feedback", "test_verified", 5)
	h2 := promoteRealInstinctVaried(t, store, "Check the error log at cmd/server.go for the panic before restarting.", "error_resolution", "multi_phase", 4)
	h3 := promoteRealInstinctVaried(t, store, "Run `npm install` after editing package.json to avoid a broken build.", "success_pattern", "single_phase", 3)
	h4 := promoteRealInstinctVaried(t, store, "Read pkg/memory/promote.go before changing the trust score formula.", "observation", "anecdotal", 2)
	h5 := promoteRealInstinctVaried(t, store, "Verify `git status` is clean before starting a new task.", "heuristic", "anecdotal", 1)

	older := time.Now().UTC().Add(-2 * time.Hour)
	newer := time.Now().UTC().Add(-5 * time.Minute)
	writeWorkerHandoffRecords(t,
		workerHandoffRecord{ID: "h1", Workflow: "build", Phase: 1, WorkerName: "Mason-12",
			Summary: "wired the memory digest into the plan brief", Freshness: older.Format(time.RFC3339)},
		workerHandoffRecord{ID: "h2", Workflow: "build", Phase: 1, WorkerName: "Weaver-9",
			Summary: "fixed the flaky handoff freshness sort", Freshness: newer.Format(time.RFC3339)},
	)

	card := runSessionStartHook(t)

	if !strings.Contains(card, "4 preferences set -- e.g. Plain English replies, no jargon.") {
		t.Errorf("the card does not carry the preferences line naming a count and an example:\n%s", card)
	}

	for _, want := range []colony.InstinctEntry{h1, h2, h3} {
		if !strings.Contains(card, want.Action) {
			t.Errorf("the card does not carry the seeded habit %q:\n%s", want.Action, card)
		}
	}
	for _, notWant := range []colony.InstinctEntry{h4, h5} {
		if strings.Contains(card, notWant.Action) {
			t.Errorf("the card carries a 4th or 5th habit, weaker than the top 3, %q:\n%s", notWant.Action, card)
		}
	}
	if got := strings.Count(card, "Learned habit: "); got != 3 {
		t.Errorf("the card carries %d habit lines, want exactly 3:\n%s", got, card)
	}

	if !strings.Contains(card, "The last helper (Weaver-9) left a note for the next one: fixed the flaky handoff freshness sort") {
		t.Errorf("the card does not name the freshest helper and what it left:\n%s", card)
	}
	if strings.Contains(card, "Mason-12") {
		t.Errorf("the card names the OLDER handoff instead of the freshest one:\n%s", card)
	}
}

// TestGreetingOmitsWhatTheProjectDoesNotHaveYet is the second <behavior>
// row: with none of preferences, habits or a relay note, the greeting
// renders every pre-existing part unchanged and adds no new heading, no
// blank section and no filler sentence (D-14).
func TestGreetingOmitsWhatTheProjectDoesNotHaveYet(t *testing.T) {
	newSessionStartProject(t)

	state := greetingFixtureState(t)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write fixture state: %v", err)
	}

	card := runSessionStartHook(t)

	if strings.TrimSpace(card) == "" {
		t.Fatal("the card is empty for a real project -- this test would prove nothing")
	}
	if !strings.Contains(card, "Ship the billing rewrite") {
		t.Errorf("the pre-existing standing block is missing:\n%s", card)
	}
	for _, unwanted := range []string{
		"What I remember about you",
		"Learned habit:",
		"preferences set",
		"left a note for the next one",
	} {
		if strings.Contains(card, unwanted) {
			t.Errorf("the card shows %q with nothing behind it -- an empty heading or filler line:\n%s", unwanted, card)
		}
	}
}

// TestGreetingStaysSilentWithoutAProject is the third <behavior> row: a
// folder with no project set up in it still gets nothing at all, unchanged
// by anything this plan added. TestSessionStartCardIsSilentWithoutAColony
// already covers this at the whole-hook level; this is 198.2 plan 03's own
// named regression lock for the memory-carrying change specifically.
func TestGreetingStaysSilentWithoutAProject(t *testing.T) {
	newSessionStartProject(t)
	if out := runSessionStartHook(t); strings.TrimSpace(out) != "" {
		t.Errorf("a project with no colony was greeted anyway, after the memory block was added:\n%s", out)
	}
}

// TestClosingCardsAreUnchangedByTheGreetingBlock is the fourth <behavior>
// row: the closing card `aether continue` prints is byte-identical to
// before this plan, even when the project actually HAS preferences, habits
// and a relay note to show. lifecycleNextActionForState(state, "continue",
// "", "") is the exact function codex_visuals.go's aether-continue closing
// calls; it goes through loadNextActionInput's sibling
// nextActionInputForState, never loadNextActionInputForGreeting, so
// answer.Memory must come back zero and the card must carry none of the
// seeded memory content.
func TestClosingCardsAreUnchangedByTheGreetingBlock(t *testing.T) {
	newSessionStartProject(t)

	writeHubPreferences(t, "Plain English replies, no jargon.")

	state := greetingFixtureState(t)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write fixture state: %v", err)
	}

	habit := promoteRealInstinctVaried(t, store, "Run `go test ./cmd/...` before claiming a fix works.", "user_feedback", "test_verified", 5)
	writeWorkerHandoffRecords(t, workerHandoffRecord{
		ID: "h1", Workflow: "build", Phase: 1, WorkerName: "Weaver-9",
		Summary: "fixed the flaky handoff freshness sort", Freshness: time.Now().UTC().Format(time.RFC3339),
	})

	answer := lifecycleNextActionForState(state, "continue", "", "")
	if !reflect.DeepEqual(answer.Memory, nextActionMemory{}) {
		t.Fatalf("aether continue's own answer carries a non-zero memory block: %+v", answer.Memory)
	}

	card := renderNextActionCard(answer)
	for _, unwanted := range []string{
		"What I remember about you",
		"Learned habit:",
		"preferences set",
		"left a note for the next one",
		habit.Action,
		"Weaver-9",
	} {
		if strings.Contains(card, unwanted) {
			t.Errorf("the aether-continue closing card carries greeting-only content %q, even though it must be unchanged:\n%s", unwanted, card)
		}
	}

	// Same proof at the machine-readable envelope this same answer feeds
	// (NEXT-02): the memory block cannot leak into the envelope either.
	envelope := applyNextActionToResult(map[string]interface{}{}, answer)
	folded, ok := nextActionFromResult(envelope)
	if !ok {
		t.Fatal("the envelope did not carry the answer back out")
	}
	if !reflect.DeepEqual(folded.Memory, nextActionMemory{}) {
		t.Fatalf("the envelope's own answer carries a non-zero memory block: %+v", folded.Memory)
	}
}

// ---------------------------------------------------------------------------
// Task 3 -- plain English and the no-write lock, with the memory block wired
// in.
// ---------------------------------------------------------------------------

// TestGreetingWithMemoryStillDoesNotMutate extends
// TestSessionStartHookDoesNotMutate's proof to the two new hub-touching
// readers this plan adds. The pre-existing test only snapshots the project's
// own data directory; this one also snapshots the hub, because
// buildNextActionMemory (cmd/next_action_input.go) now reads the hub's
// QUEEN.md for the preferences line.
func TestGreetingWithMemoryStillDoesNotMutate(t *testing.T) {
	newSessionStartProject(t)
	hubDir := strings.TrimSpace(os.Getenv("AETHER_HUB_DIR"))
	if hubDir == "" {
		t.Fatal("AETHER_HUB_DIR is not set -- newSessionStartProject must isolate it")
	}
	writeHubPreferences(t, "Plain English replies, no jargon.")

	state := greetingFixtureState(t)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write fixture state: %v", err)
	}
	if err := store.SaveJSON("session.json", colony.SessionFile{SessionID: "s1", StartedAt: "2026-08-01T10:00:00Z"}); err != nil {
		t.Fatalf("write session fixture: %v", err)
	}

	promoteRealInstinctVaried(t, store, "Run `go test ./cmd/...` before claiming a fix works.", "user_feedback", "test_verified", 5)
	writeWorkerHandoffRecords(t, workerHandoffRecord{
		ID: "h1", Workflow: "build", Phase: 1, WorkerName: "Mason-12",
		Summary: "wired the memory digest into the plan brief", Freshness: time.Now().UTC().Format(time.RFC3339),
	})

	beforeData := snapshotProjectDataTree(t, store.BasePath())
	beforeHub := snapshotProjectDataTree(t, hubDir)
	first := runSessionStartHook(t)
	second := runSessionStartHook(t)
	afterData := snapshotProjectDataTree(t, store.BasePath())
	afterHub := snapshotProjectDataTree(t, hubDir)

	if !reflect.DeepEqual(beforeData, afterData) {
		t.Errorf("the greeting changed the project's saved data.\nbefore: %v\nafter:  %v", beforeData, afterData)
	}
	if !reflect.DeepEqual(beforeHub, afterHub) {
		t.Errorf("the greeting changed the hub's saved data.\nbefore: %v\nafter:  %v", beforeHub, afterHub)
	}
	if first != second {
		t.Errorf("two greetings over an unchanged project differed.\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if !strings.Contains(first, "Learned habit:") || !strings.Contains(first, "1 preference set") {
		t.Fatalf("the fixture produced a card with no memory content, so this test would not exercise the new hub-touching readers:\n%s", first)
	}
}
