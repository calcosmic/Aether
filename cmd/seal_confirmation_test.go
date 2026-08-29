package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

// autoRecordSealConfirmationForTest pre-records the exact "yes" answer
// sealCmd's D-04 confirmation gate will ask for given s's CURRENT fixture
// state and pending-decisions.json -- the same recorded answer an owner
// running `aether seal` twice (ask, then confirm) would produce. Shared by
// every pre-existing seal test helper that predates the gate and is not
// itself testing confirmation behavior.
func autoRecordSealConfirmationForTest(t *testing.T, s *storage.Store) {
	t.Helper()
	var state colony.ColonyState
	_ = s.LoadJSON("COLONY_STATE.json", &state)
	blockers, issues := checkSealBlockers(s, state)
	card := buildSealStateOfPlay(state, blockers, issues)
	namedProblems := card.namedProblems()
	question := sealConfirmationQuestionText(namedProblems)
	source := sealConfirmationAnswerSource(namedProblems)
	if _, err := recordSealConfirmationAnswer(question, "yes", source); err != nil {
		t.Fatalf("auto-record seal confirmation answer: %v", err)
	}
}

// --- Task 1: the wisdom review runs exactly once, before anything changes ---

// sealTestState builds the minimal completed-colony fixture every Task 1
// test seals against: one completed phase, nothing blocking.
func sealTestState(goal string) colony.ColonyState {
	return colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Complete work",
			Status: colony.PhaseCompleted,
		}}},
	}
}

// countConsolidationSealEvents reads the real event bus a completed seal
// wrote to and counts how many times runSealConsolidation's own
// "consolidation.seal" topic was published -- instrumentation fed by the
// real call path (pkg/events' persisted JSONL), never a mock of
// runSealWisdomReview or runSealConsolidation themselves.
func countConsolidationSealEvents(t *testing.T, s interface {
	ReadJSONL(string) ([]json.RawMessage, error)
}) int {
	t.Helper()
	lines, err := s.ReadJSONL("event-bus.jsonl")
	if err != nil {
		t.Fatalf("read event bus: %v", err)
	}
	count := 0
	for _, line := range lines {
		var evt events.Event
		if err := json.Unmarshal(line, &evt); err != nil {
			t.Fatalf("unmarshal persisted event: %v", err)
		}
		if evt.Topic == "consolidation.seal" {
			count++
		}
	}
	return count
}

// TestSealWisdomReviewRunsExactlyOncePerSeal pins D-05's "runs once": a
// complete seal must publish exactly one consolidation.seal event, proving
// runSealConsolidation (the review's core) ran a single time -- not once
// for a stand-alone review pass and again inside completeSealRuntime.
func TestSealWisdomReviewRunsExactlyOncePerSeal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	stdout = &bytes.Buffer{}

	if err := s.SaveJSON("COLONY_STATE.json", sealTestState("Wisdom review runs once")); err != nil {
		t.Fatalf("save state: %v", err)
	}

	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal returned error: %v", err)
	}

	if got := countConsolidationSealEvents(t, s); got != 1 {
		t.Fatalf("consolidation.seal published %d time(s), want exactly 1", got)
	}
}

// TestSealWisdomReviewPrecedesStateChange asserts the learned-lessons store
// (local QUEEN.md, written by the seal-side promotion loop inside
// runSealWisdomReview) is written before COLONY_STATE.json records the
// finished state -- by comparing the two files' observed write order in a
// seeded fixture store, per D-05.
func TestSealWisdomReviewPrecedesStateChange(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	store = s
	stdout = &bytes.Buffer{}

	// An instinct eligible for the SEAL-SIDE local promotion loop
	// (Confidence >= 0.8, non-empty Action) but NOT QueenEligible via
	// consolidation (application history below 3), so promoteInstinctLocal
	// itself is the writer that touches QUEEN.md for this fixture.
	instincts := colony.InstinctsFile{Version: "1.0", Instincts: []colony.InstinctEntry{{
		ID:         "precedes-state-change",
		Trigger:    "seal ordering fixture",
		Action:     "Keep the wisdom review ahead of the state mutation",
		Domain:     "testing",
		Confidence: 0.95,
	}}}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("seed instincts: %v", err)
	}
	if err := s.SaveJSON("COLONY_STATE.json", sealTestState("Wisdom review precedes state change")); err != nil {
		t.Fatalf("save state: %v", err)
	}

	// D-04's confirmation gate (Task 2) now asks before completing; this
	// test is about ordering WITHIN a seal that actually completes, so
	// pre-record the answer exactly as an owner running seal twice would.
	autoRecordSealConfirmationForTest(t, s)

	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal returned error: %v", err)
	}

	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	queenStat, err := os.Stat(queenPath)
	if err != nil {
		t.Fatalf("expected local QUEEN.md to be written by the review: %v", err)
	}
	statePath := filepath.Join(tmpDir, ".aether", "data", "COLONY_STATE.json")
	stateStat, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("stat COLONY_STATE.json: %v", err)
	}

	if queenStat.ModTime().After(stateStat.ModTime()) {
		t.Fatalf("QUEEN.md (the review's own lessons store) was written AFTER COLONY_STATE.json recorded the finished state: queen=%s state=%s",
			queenStat.ModTime(), stateStat.ModTime())
	}

	var after colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &after); err != nil {
		t.Fatalf("load state after seal: %v", err)
	}
	if after.State != colony.StateCOMPLETED {
		t.Fatalf("state after seal = %s, want COMPLETED", after.State)
	}

	queenText, err := os.ReadFile(queenPath)
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	if !bytes.Contains(queenText, []byte("Keep the wisdom review ahead of the state mutation")) {
		t.Fatalf("QUEEN.md missing the promoted instinct's action text:\n%s", queenText)
	}
}

// hashTree returns a stable per-file content hash for every regular file
// under root, keyed by the path relative to root.
func hashTree(t *testing.T, root string) map[string]string {
	t.Helper()
	hashes := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// .aether/locks/ holds advisory lock files storage.Store creates
			// (and .cache_*.json read-through caches) as a side effect of
			// EVERY read, mutating or not -- infra bookkeeping, not colony
			// state or the learned-lessons store, so it is excluded from
			// this invariant.
			if info.Name() == "locks" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(info.Name(), ".cache_") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		hashes[rel] = fmt.Sprintf("%x", sha256.Sum256(data))
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return hashes
}

// TestSealPlanOnlyDoesNotMutate pins the CLAUDE.md dry-run corollary for the
// seal inspection path: --plan-only must never write to colony state or the
// learned-lessons store, byte-for-byte, across every file in the fixture
// store.
func TestSealPlanOnlyDoesNotMutate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	store = s
	stdout = &bytes.Buffer{}

	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         stringPtrForSealConfirmationTest("Seal plan-only must not mutate"),
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Complete work",
			Status: colony.PhaseCompleted,
			Mode:   colony.PhaseModeProduction,
		}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	before := hashTree(t, tmpDir)

	rootCmd.SetArgs([]string{"seal", "--plan-only"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal --plan-only returned error: %v", err)
	}

	after := hashTree(t, tmpDir)

	if len(before) != len(after) {
		beforeKeys := sortedKeys(before)
		afterKeys := sortedKeys(after)
		t.Fatalf("file set changed: before=%v after=%v", beforeKeys, afterKeys)
	}
	for path, wantHash := range before {
		gotHash, ok := after[path]
		if !ok {
			t.Fatalf("file disappeared during --plan-only: %s", path)
		}
		if gotHash != wantHash {
			t.Fatalf("file mutated by --plan-only: %s", path)
		}
	}
}

func stringPtrForSealConfirmationTest(s string) *string {
	return &s
}

// --- Task 2: the state-of-play card, the question, and the recorded answer ---

// newSealConfirmationTestStore builds a fresh fixture store with one
// completed phase and nothing blocking -- the common starting point for
// every Task 2 confirmation-gate test.
func newSealConfirmationTestStore(t *testing.T, goal string) (*storage.Store, string) {
	t.Helper()
	s, tmpDir := newTestStore(t)
	if err := s.SaveJSON("COLONY_STATE.json", sealTestState(goal)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	return s, tmpDir
}

// seedSealConfirmationIssue writes a single unresolved issue-severity flag
// (never a blocker, so checkSealBlockers still lets seal reach the
// confirmation gate) whose Description becomes one of D-06's named
// problems.
func seedSealConfirmationIssue(t *testing.T, s *storage.Store, description string) {
	t.Helper()
	flags := colony.FlagsFile{Version: "1", Decisions: []colony.FlagEntry{
		{ID: "issue-1", Type: "issue", Description: description, Resolved: false, CreatedAt: "2026-08-29", Source: "test"},
	}}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatalf("seed issue flag: %v", err)
	}
}

// beginSealConfirmationTest wires the package globals to s and resets
// rootCmd, BEFORE any pre-recording (recordSealConfirmationAnswer reads and
// writes through the package-level store, so it must already point at s).
func beginSealConfirmationTest(t *testing.T, s *storage.Store) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	store = s
	stdout = &bytes.Buffer{}
}

// executeSealForConfirmationTest runs the real `aether seal` command and
// returns everything written to stdout (prose and the final JSON envelope
// together) -- the real seal path's own effect, never an intermediate
// record. Call beginSealConfirmationTest first.
func executeSealForConfirmationTest(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal returned error: %v", err)
	}
	return buf.String()
}

// runSealForConfirmationTest is the common case: no pre-recorded answer, no
// seeded issues -- just wire globals and run seal once.
func runSealForConfirmationTest(t *testing.T, s *storage.Store) string {
	t.Helper()
	beginSealConfirmationTest(t, s)
	return executeSealForConfirmationTest(t)
}

func assertSealCompleted(t *testing.T, s *storage.Store) {
	t.Helper()
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Fatalf("expected seal to complete, state = %s", state.State)
	}
}

func assertSealNotCompleted(t *testing.T, s *storage.Store) {
	t.Helper()
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.State == colony.StateCOMPLETED {
		t.Fatal("expected seal NOT to complete without a covering recorded answer, but state = COMPLETED")
	}
}

// lastJSONLine extracts the final JSON object written to output -- the
// {"ok":true,"result":{...}} envelope outputOK writes as the very last
// thing, after any prose emitted earlier in the same run.
func lastJSONLine(t *testing.T, output string) map[string]interface{} {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err == nil {
			return m
		}
	}
	t.Fatalf("no JSON envelope found in output:\n%s", output)
	return nil
}

// TestSealAlwaysAsksBeforeFinishing tables the <behavior> rows from
// 198-03-PLAN.md's Task 2, asserting on the real seal path's effect on the
// fixture store (COLONY_STATE.json's State), never on an intermediate
// decision record.
func TestSealAlwaysAsksBeforeFinishing(t *testing.T) {
	t.Run("nothing failing, no recorded answer: asks, does not seal", func(t *testing.T) {
		s, _ := newSealConfirmationTestStore(t, "Nothing failing")
		out := runSealForConfirmationTest(t, s)
		assertSealNotCompleted(t, s)
		if !strings.Contains(out, "Finish this project?") {
			t.Fatalf("expected the plain confirmation question, got:\n%s", out)
		}
	})

	t.Run("something failing, no recorded answer: names it, asks the second question", func(t *testing.T) {
		s, _ := newSealConfirmationTestStore(t, "Something failing")
		seedSealConfirmationIssue(t, s, "the deploy script is untested")
		out := runSealForConfirmationTest(t, s)
		assertSealNotCompleted(t, s)
		if !strings.Contains(out, "Finish anyway with 1 check(s) failing") {
			t.Fatalf("expected the second, explicit question naming the count, got:\n%s", out)
		}
	})

	t.Run("recorded yes, no named problems: proceeds", func(t *testing.T) {
		s, _ := newSealConfirmationTestStore(t, "Clean finish")
		beginSealConfirmationTest(t, s)
		if _, err := recordSealConfirmationAnswer(sealConfirmationQuestionText(nil), "yes", "seal-confirmation"); err != nil {
			t.Fatalf("record answer: %v", err)
		}
		executeSealForConfirmationTest(t)
		assertSealCompleted(t, s)
	})

	t.Run("named problems, recorded plain yes that does not cover them: still does not proceed", func(t *testing.T) {
		s, _ := newSealConfirmationTestStore(t, "Stale plain yes")
		seedSealConfirmationIssue(t, s, "the deploy script is untested")
		beginSealConfirmationTest(t, s)
		if _, err := recordSealConfirmationAnswer(sealConfirmationQuestionText(nil), "yes", "seal-confirmation"); err != nil {
			t.Fatalf("record answer: %v", err)
		}
		executeSealForConfirmationTest(t)
		assertSealNotCompleted(t, s)
	})

	t.Run("named problems, recorded finish-anyway naming them: proceeds", func(t *testing.T) {
		s, _ := newSealConfirmationTestStore(t, "Finish anyway")
		seedSealConfirmationIssue(t, s, "the deploy script is untested")
		beginSealConfirmationTest(t, s)
		question := sealConfirmationQuestionText([]string{"the deploy script is untested"})
		if _, err := recordSealConfirmationAnswer(question, "yes", "seal-force-confirmation"); err != nil {
			t.Fatalf("record answer: %v", err)
		}
		executeSealForConfirmationTest(t)
		assertSealCompleted(t, s)
	})
}

// TestSealFinishAnywayIsRecordedNotInferred pins D-06: a card whose text
// happens to contain an affirmative phrase must never be mistaken for a
// recorded answer, and decideSealConfirmation's own signature proves it
// cannot even see such text -- it takes a plain struct of already-resolved
// facts, never a store handle or a rendered-card string.
func TestSealFinishAnywayIsRecordedNotInferred(t *testing.T) {
	t.Run("an affirmative-looking card never proceeds without a recorded answer", func(t *testing.T) {
		s, _ := newSealConfirmationTestStore(t, "Affirmative-looking card")
		seedSealConfirmationIssue(t, s, "yes, everything here looks fine and ready to ship")
		runSealForConfirmationTest(t, s)
		assertSealNotCompleted(t, s)
	})

	t.Run("decideSealConfirmation takes no store handle or rendered-card string", func(t *testing.T) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "seal_confirmation.go", nil, 0)
		if err != nil {
			t.Fatalf("parse seal_confirmation.go: %v", err)
		}
		var fn *ast.FuncDecl
		for _, decl := range file.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == "decideSealConfirmation" {
				fn = fd
				break
			}
		}
		if fn == nil {
			t.Fatal("decideSealConfirmation not found in seal_confirmation.go -- a rename would silently blind this guard")
		}
		if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
			got := 0
			if fn.Type.Params != nil {
				got = len(fn.Type.Params.List)
			}
			t.Fatalf("decideSealConfirmation should take exactly one parameter, got %d", got)
		}
		ident, ok := fn.Type.Params.List[0].Type.(*ast.Ident)
		if !ok || ident.Name != "sealConfirmationInput" {
			t.Fatalf("decideSealConfirmation's parameter type is not the plain sealConfirmationInput struct -- it must never take a *storage.Store, colony.ColonyState, or a rendered-card string")
		}
	})
}

// TestSealNoAnswerKeepsTheLessons pins D-05: a recorded "no" leaves the
// project unfinished, but the wisdom review's lessons -- already run before
// the question was ever asked -- are still present in the store afterward.
func TestSealNoAnswerKeepsTheLessons(t *testing.T) {
	s, tmpDir := newSealConfirmationTestStore(t, "Recorded no")

	instincts := colony.InstinctsFile{Version: "1.0", Instincts: []colony.InstinctEntry{{
		ID:         "no-answer-lesson",
		Trigger:    "seal confirmation no-answer fixture",
		Action:     "Keep this lesson even when the owner says no",
		Domain:     "testing",
		Confidence: 0.9,
	}}}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("seed instincts: %v", err)
	}

	beginSealConfirmationTest(t, s)
	if _, err := recordSealConfirmationAnswer(sealConfirmationQuestionText(nil), "no", "seal-confirmation"); err != nil {
		t.Fatalf("record answer: %v", err)
	}

	executeSealForConfirmationTest(t)
	assertSealNotCompleted(t, s)

	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	data, err := os.ReadFile(queenPath)
	if err != nil {
		t.Fatalf("expected the review's lessons on local QUEEN.md even after a recorded no: %v", err)
	}
	if !strings.Contains(string(data), "Keep this lesson even when the owner says no") {
		t.Fatalf("QUEEN.md missing the review's promoted lesson after a recorded no:\n%s", data)
	}
}

// TestSealConfirmationDispatchesNoWorkers pins that the whole confirmation
// path -- card, review, question -- spawns no additional helper: the JSON
// result it returns carries no dispatch list of any kind.
func TestSealConfirmationDispatchesNoWorkers(t *testing.T) {
	s, _ := newSealConfirmationTestStore(t, "No workers for confirmation")
	out := runSealForConfirmationTest(t, s)

	env := lastJSONLine(t, out)
	result, ok := env["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected a result envelope, got: %v", env)
	}
	if result["awaiting_owner_confirmation"] != true {
		t.Fatalf("expected awaiting_owner_confirmation:true, got: %+v", result)
	}
	for _, dispatchKey := range []string{"dispatches", "seal_manifest", "dispatch_count"} {
		if _, present := result[dispatchKey]; present {
			t.Fatalf("the confirmation path dispatched something (%q present) -- it must spawn no additional helper", dispatchKey)
		}
	}
}

// --- Task 3: autopilot provably cannot reach the seal path ---

// TestAutopilotNeverReachesTheSealPath is D-07's reachability ratchet: a
// call-graph walk over the cmd package, starting at runCompatibilityAutopilot,
// asserting completeSealRuntime is not reachable through any chain of
// package-level function calls. This is the same discipline as this repo's
// existing reachability ratchets (cmd/verify_out_of_band_reachability_test.go,
// cmd/worktree_destruction_reachability_test.go): it walks the real call
// graph rather than searching for a literal name, so it fails if a future
// change wires autopilot into finishing -- exactly D-07's requirement that
// this test can fail.
//
// The fact holds today because runCompatibilityAutopilot ends by returning
// the finishing command ("aether seal") as the owner's next step rather than
// running it (cmd/compatibility_cmds.go's buildRunExecutionResult) -- it
// never calls sealCmd's RunE or completeSealRuntime directly.
func TestAutopilotNeverReachesTheSealPath(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	g, err := buildCmdFuncGraph(filepath.Join(repoRoot, "cmd"))
	if err != nil {
		t.Fatalf("build call graph: %v", err)
	}

	if g.filesScanned == 0 {
		t.Fatalf("scanned zero .go files -- this guard cannot assert anything about an empty graph")
	}
	if g.funcsIndexed == 0 {
		t.Fatalf("found zero top-level functions while scanning %d files -- a guard that finds no functions to check would pass vacuously forever", g.filesScanned)
	}

	const startFunc = "runCompatibilityAutopilot"
	const sealFunc = "completeSealRuntime"

	if _, ok := g.calls[startFunc]; !ok {
		t.Fatalf("expected %s to be an indexed top-level function in cmd/, but it was missing -- this guard cannot assert an isolation property against a function it cannot locate (renamed? moved?)", startFunc)
	}
	if _, ok := g.calls[sealFunc]; !ok {
		t.Fatalf("expected %s to be an indexed top-level function in cmd/, but it was missing -- this guard cannot assert an isolation property against a function it cannot locate (renamed? moved?)", sealFunc)
	}

	reachable := reachableFrom(g, []string{startFunc})

	if reachable[sealFunc] {
		t.Errorf("%s is reachable from %s -- autopilot must never reach the code that finishes (seals) a project; it should stop at the last phase and hand the owner the finish command instead (D-07)", sealFunc, startFunc)
	}
}
