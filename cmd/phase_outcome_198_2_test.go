package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 198.2 plan 05 (WIRE-07, D-11): outcome.md must carry, word for word,
// the closing screen the owner saw when a check finished -- produced by the
// same D-12 renderer (closeoutContinueDirectVisual) fed the same finalizer
// result map, never a second hand-derivation.

// TestOutcomeFileMatchesTheClosingScreen builds a passing continue result the
// way a real finalize builds it (typed structs, not a JSON-round-tripped
// map -- writePhaseOutcomeDocument is called in-process on both lanes, never
// on a completion-file round trip), renders the closing screen independently
// via the same renderer writePhaseOutcomeDocument itself calls, and asserts
// the persisted file equals that stripped screen byte-for-byte -- equality,
// not containment, so the writer building even one line of its own prose
// would fail this test.
func TestOutcomeFileMatchesTheClosingScreen(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := colony.ColonyState{
		Version:      "3.0",
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Ship the thing", Status: colony.PhaseCompleted},
		}},
	}

	verification := codexContinueVerificationReport{
		Steps:        []codexVerificationStep{{Name: "build", Passed: true, Duration: 1.0}},
		ChecksPassed: true,
	}
	gates := codexContinueGateReport{Checks: []gateCheck{{Name: "verification_steps_passed", Passed: true}}}

	result := map[string]interface{}{
		"advanced":             true,
		"completed":            false,
		"current_phase":        1,
		"continued_phase":      1,
		"continued_phase_name": "Ship the thing",
		"state":                colony.StateEXECUTING,
		"next":                 "aether build 2",
		"review_depth":         "standard",
		"verification":         verification,
		"gates":                gates,
	}

	wantVisual, ok := closeoutContinueDirectVisual(result, state)
	if !ok {
		t.Fatalf("fixture broken: closeoutContinueDirectVisual could not resolve phase from the fixture")
	}
	want := stripPhaseOutcomeANSI(wantVisual)

	if err := writePhaseOutcomeDocument(1, result, state); err != nil {
		t.Fatalf("writePhaseOutcomeDocument: %v", err)
	}

	got, err := s.ReadFile(continuePlanArtifactsPath(1, "outcome.md"))
	if err != nil {
		t.Fatalf("read outcome.md: %v", err)
	}
	if string(got) != want {
		t.Fatalf("outcome.md does not equal the closing screen:\n%s", firstDiffLine(want, string(got)))
	}
}

// TestBlockedPhaseStillGetsAnOutcome proves a blocked check also gets its
// closing screen persisted -- a phase that failed is exactly the phase whose
// outcome the next one most needs (D-11).
func TestBlockedPhaseStillGetsAnOutcome(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := colony.ColonyState{
		Version:      "3.0",
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Ship the thing"}}},
	}

	gates := codexContinueGateReport{Checks: []gateCheck{
		{Name: "verification_steps_passed", Passed: false, FixHint: "fix the failing build/test check and run aether continue"},
	}}
	result := map[string]interface{}{
		"advanced":             false,
		"blocked":              true,
		"current_phase":        1,
		"continued_phase":      1,
		"continued_phase_name": "Ship the thing",
		"state":                colony.StateBUILT,
		"next":                 "aether unblock --dispatch",
		"review_depth":         "standard",
		"gates":                gates,
		"blocking_issues":      []string{"build failed"},
	}

	wantVisual, ok := closeoutContinueDirectVisual(result, state)
	if !ok {
		t.Fatalf("fixture broken: closeoutContinueDirectVisual could not resolve the blocked phase")
	}
	want := stripPhaseOutcomeANSI(wantVisual)

	if err := writePhaseOutcomeDocument(1, result, state); err != nil {
		t.Fatalf("writePhaseOutcomeDocument: %v", err)
	}

	got, err := s.ReadFile(continuePlanArtifactsPath(1, "outcome.md"))
	if err != nil {
		t.Fatalf("read outcome.md: %v", err)
	}
	if string(got) != want {
		t.Fatalf("blocked outcome.md does not equal the blocked closing screen:\n%s", firstDiffLine(want, string(got)))
	}
	if !strings.Contains(string(got), "build failed") {
		t.Errorf("blocked outcome.md missing the blocking issue text, got:\n%s", string(got))
	}
}

// TestUnresolvableResultWritesNoOutcomeFile proves that when the renderer
// cannot resolve a phase from the given inputs, writePhaseOutcomeDocument
// writes nothing and returns a non-nil error rather than an empty file.
func TestUnresolvableResultWritesNoOutcomeFile(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := colony.ColonyState{
		Version: "3.0",
		Plan:    colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Ship the thing"}}},
	}
	// Phase 99 does not exist in state -- colonyPhaseByID cannot resolve it,
	// so closeoutContinueDirectVisual must report handled=false.
	result := map[string]interface{}{"advanced": true, "completed": false}

	err := writePhaseOutcomeDocument(99, result, state)
	if err == nil {
		t.Fatalf("expected a non-nil error for an unresolvable phase, got nil")
	}

	path := filepath.Join(s.BasePath(), filepath.FromSlash(continuePlanArtifactsPath(99, "outcome.md")))
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("expected no outcome.md to be written for an unresolvable phase, but found one at %s", path)
	}
}

// TestOutcomeFileIsReplacedNotAppended proves calling
// writePhaseOutcomeDocument twice for the same phase replaces the first
// outcome rather than appending to it -- one outcome per phase, ever.
func TestOutcomeFileIsReplacedNotAppended(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := colony.ColonyState{
		Version:      "3.0",
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Ship the thing"}}},
	}

	firstResult := map[string]interface{}{
		"advanced":             false,
		"blocked":              true,
		"continued_phase":      1,
		"continued_phase_name": "Ship the thing",
		"state":                colony.StateBUILT,
		"next":                 "aether unblock --dispatch",
		"blocking_issues":      []string{"first failure"},
	}
	if err := writePhaseOutcomeDocument(1, firstResult, state); err != nil {
		t.Fatalf("first write: %v", err)
	}

	secondResult := map[string]interface{}{
		"advanced":             true,
		"completed":            false,
		"continued_phase":      1,
		"continued_phase_name": "Ship the thing",
		"state":                colony.StateEXECUTING,
		"next":                 "aether build 2",
	}
	if err := writePhaseOutcomeDocument(1, secondResult, state); err != nil {
		t.Fatalf("second write: %v", err)
	}

	got, err := s.ReadFile(continuePlanArtifactsPath(1, "outcome.md"))
	if err != nil {
		t.Fatalf("read outcome.md: %v", err)
	}
	if strings.Contains(string(got), "first failure") {
		t.Errorf("outcome.md still carries the first call's text -- it was appended, not replaced:\n%s", string(got))
	}

	wantVisual, ok := closeoutContinueDirectVisual(secondResult, state)
	if !ok {
		t.Fatalf("fixture broken: could not render the second result")
	}
	if string(got) != stripPhaseOutcomeANSI(wantVisual) {
		t.Errorf("outcome.md after replacement does not equal the second call's closing screen:\n%s", firstDiffLine(stripPhaseOutcomeANSI(wantVisual), string(got)))
	}
}

// ---------------------------------------------------------------------------
// Task 2 (WIRE-07): both check lanes write the outcome exactly once, never on
// the planning-only step, and never at the cost of the check completing.
// ---------------------------------------------------------------------------

// containsCall reports whether n contains a call expression whose function
// identifier is named funcName. Used both to find closeLifecycleRun
// statements and to verify the following statement calls
// writePhaseOutcomeDocument -- which may be wrapped in
// `if err := ...; err != nil { ... }`, so a direct type assertion on the
// statement is not enough.
func containsCall(n ast.Node, funcName string) bool {
	found := false
	ast.Inspect(n, func(inner ast.Node) bool {
		call, ok := inner.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == funcName {
			found = true
			return false
		}
		return true
	})
	return found
}

// isDirectCallStmt reports whether stmt is ITSELF an expression statement
// calling funcName -- unlike containsCall, this does not recurse into nested
// statements. It is the strict anchor for finding a closeLifecycleRun
// statement at a given block's own top level: an `if cond { ...
// closeLifecycleRun(...) ... }` wrapping statement would wrongly match
// containsCall (it contains a nested closeLifecycleRun call somewhere inside
// its body), but that wrapping `if` statement's own NEXT sibling is not
// where the corresponding write call lives -- the write lives inside the
// `if`'s own body, immediately after closeLifecycleRun there. Using
// isDirectCallStmt for anchor detection means that inner block is checked
// when checkCloseLifecycleRunAdjacency recurses into it, not the outer one.
func isDirectCallStmt(stmt ast.Stmt, funcName string) bool {
	exprStmt, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	ident, ok := call.Fun.(*ast.Ident)
	return ok && ident.Name == funcName
}

// checkCloseLifecycleRunAdjacency walks block's own statement list looking
// for a direct closeLifecycleRun call; when found, the immediately
// following statement in that SAME list must contain a
// writePhaseOutcomeDocument call (which may itself be wrapped in
// `if err := ...; err != nil { ... }`, hence the loose containsCall here).
// It then recurses into every nested block (if/else/for bodies) so a
// closeLifecycleRun call nested inside a branch is still found and checked
// against its own sibling statement, not the outer block's.
func checkCloseLifecycleRunAdjacency(t *testing.T, file string, fset *token.FileSet, block *ast.BlockStmt) {
	t.Helper()
	for i, stmt := range block.List {
		if isDirectCallStmt(stmt, "closeLifecycleRun") {
			if i+1 >= len(block.List) || !containsCall(block.List[i+1], "writePhaseOutcomeDocument") {
				t.Errorf("%s:%d: closeLifecycleRun is not immediately followed by a writePhaseOutcomeDocument call -- every terminal branch of the check-finish path must persist its closing screen (WIRE-07)",
					file, fset.Position(stmt.Pos()).Line)
			}
		}
		ast.Inspect(stmt, func(n ast.Node) bool {
			if nested, ok := n.(*ast.BlockStmt); ok && nested != block {
				checkCloseLifecycleRunAdjacency(t, file, fset, nested)
				return false
			}
			return true
		})
	}
}

// TestEveryCheckLaneWritesOneOutcome is the WIRE-07 AST guard: every
// closeLifecycleRun call site in cmd/codex_continue.go and
// cmd/codex_continue_finalize.go -- the codebase's own established marker
// for "a finished run's result map, about to become the closing screen" (see
// closeLifecycleRun's own doc comment in cmd/codex_visuals.go) -- must be
// immediately followed, in the same statement block, by a
// writePhaseOutcomeDocument call. A future terminal branch added to either
// file without that call fails this test by naming the file and line.
//
// Deliberately scoped to exactly these two files: cmd/codex_continue_plan.go
// also calls closeLifecycleRun (on the --plan-only path), which must NEVER
// write an outcome -- see TestPlanOnlyCheckWritesNoOutcome.
//
// runCodexContinueFinalize's own success terminal is a documented exception
// (Part 2 below): it is not adjacent to a closeLifecycleRun call, because
// that call already ran earlier inside advanceExternalContinue, before
// attachConsolidationSummary/attachHivePromotionSummary/
// attachPhaseCommitResult mutate the SAME result map further (see the
// comment at that call site in cmd/codex_continue_finalize.go).
func TestEveryCheckLaneWritesOneOutcome(t *testing.T) {
	files := []string{"codex_continue.go", "codex_continue_finalize.go"}
	fset := token.NewFileSet()
	for _, name := range files {
		path := filepath.Join(".", name)
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		// Part 1: closeLifecycleRun-adjacency, scoped per function so
		// advanceExternalContinue's own (documented) exception does not
		// require excluding its call site by line number.
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			// advanceExternalContinue's own closeLifecycleRun call (inside
			// cmd/codex_continue_finalize.go) is the one documented
			// exception: the write for its success terminal happens later,
			// in runCodexContinueFinalize (Part 2 below), after
			// attachConsolidationSummary/attachHivePromotionSummary/
			// attachPhaseCommitResult mutate the SAME result map further --
			// see the comment at that call site.
			if fn.Name.Name == "advanceExternalContinue" {
				continue
			}
			checkCloseLifecycleRunAdjacency(t, name, fset, fn.Body)
		}

		// Part 2: runCodexContinueFinalize's own final return -- its true
		// success terminal, after every result-map mutation -- must be
		// immediately preceded by a writePhaseOutcomeDocument call.
		if name == "codex_continue_finalize.go" {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name.Name != "runCodexContinueFinalize" || fn.Body == nil {
					continue
				}
				list := fn.Body.List
				if len(list) == 0 {
					t.Fatalf("runCodexContinueFinalize has an empty body -- test assumption broken, update this guard")
				}
				last := list[len(list)-1]
				if _, isReturn := last.(*ast.ReturnStmt); !isReturn {
					t.Fatalf("runCodexContinueFinalize's last statement is not a return -- test assumption broken, update this guard")
				}
				if len(list) < 2 || !containsCall(list[len(list)-2], "writePhaseOutcomeDocument") {
					t.Errorf("%s:%d: runCodexContinueFinalize's final return is not immediately preceded by a writePhaseOutcomeDocument call -- its success terminal must persist the fully-mutated closing screen (WIRE-07)",
						name, fset.Position(last.Pos()).Line)
				}
			}
		}
	}
}

// drivePhaseOutcomeFastLane drives cmd/codex_continue.go's default (fast)
// lane to a durable advance at light review depth, failing the test if it
// does not advance -- reaching past a call to this helper is itself the
// proof that a failure inside writePhaseOutcomeDocument never fails the
// check (T-198.2-15).
func drivePhaseOutcomeFastLane(t *testing.T, root string) map[string]interface{} {
	t.Helper()
	result, _, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{LightFlag: true, SkipWatchers: true})
	if err != nil {
		t.Fatalf("runCodexContinue: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true, got %v", result)
	}
	return result
}

// drivePhaseOutcomeWrapperLane drives cmd/codex_continue_finalize.go's
// external (wrapper-driven) lane to a durable advance at the same light
// review depth as drivePhaseOutcomeFastLane -- matched depth so the two
// lanes' closing screens are structurally comparable (a heavier depth would
// legitimately dispatch a different worker set, per CLAUDE.md's depth
// table).
func drivePhaseOutcomeWrapperLane(t *testing.T, root string) map[string]interface{} {
	t.Helper()
	planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{LightFlag: true, SkipWatchers: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly: %v", err)
	}
	plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
	if !ok {
		t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
	}
	results := make([]codexContinueExternalDispatch, 0, len(plan.Dispatches))
	for _, dispatch := range plan.Dispatches {
		results = append(results, codexContinueExternalDispatch{
			Stage: dispatch.Stage, Wave: dispatch.Wave, Caste: dispatch.Caste,
			Name: dispatch.Name, Task: dispatch.Task, TaskID: dispatch.TaskID,
			Status:  "completed",
			Summary: dispatch.Name + " cleared the check",
			Handoff: codex.WorkerHandoff{
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{dispatch.Name + " found no blocking issues"},
			},
		})
	}
	result, _, _, _, _, _, err := runCodexContinueFinalize(root, codexExternalContinueCompletion{
		ContinueManifest: &plan,
		Dispatches:       results,
	}, false, 0, false)
	if err != nil {
		t.Fatalf("runCodexContinueFinalize: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true, got %v", result)
	}
	return result
}

// TestPlanOnlyCheckWritesNoOutcome proves the --plan-only step -- which
// dispatches nothing and finishes nothing -- writes no outcome.md. Writing
// one there would describe a check that had not happened.
func TestPlanOnlyCheckWritesNoOutcome(t *testing.T) {
	saveGlobalsCmd(t)
	root := setupPhaseEndHiveContinueFixture(t, "Plan-only writes no outcome")
	s := store

	if _, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{LightFlag: true}); err != nil {
		t.Fatalf("runCodexContinuePlanOnly: %v", err)
	}

	path := filepath.Join(s.BasePath(), filepath.FromSlash(continuePlanArtifactsPath(1, "outcome.md")))
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("expected the --plan-only path to write no outcome.md, but found one at %s", path)
	}
}

// TestOutcomeWriteFailureDoesNotFailTheCheck forces writePhaseOutcomeDocument
// to fail (by pre-creating outcome.md's own path as a directory, so
// store.AtomicWrite's rename step cannot succeed) and proves the check still
// completes and still advances (T-198.2-15).
func TestOutcomeWriteFailureDoesNotFailTheCheck(t *testing.T) {
	saveGlobalsCmd(t)
	root := setupPhaseEndHiveContinueFixture(t, "Outcome write failure is non-fatal")
	s := store

	outcomePath := filepath.Join(s.BasePath(), filepath.FromSlash(continuePlanArtifactsPath(1, "outcome.md")))
	if err := os.MkdirAll(outcomePath, 0755); err != nil {
		t.Fatalf("seed outcome.md collision directory: %v", err)
	}

	// drivePhaseOutcomeFastLane itself fatals if the check does not advance
	// -- reaching past this call is the proof the write failure was
	// non-fatal.
	drivePhaseOutcomeFastLane(t, root)

	info, err := os.Stat(outcomePath)
	if err != nil {
		t.Fatalf("stat outcome.md path: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("test setup broken: outcome.md path is no longer the seeded directory, so the write failure this test relies on may not have occurred")
	}
}

// TestBothCheckLanesLeaveTheSameOutcomeText is WIRE-07's two-lane proof for
// the persisted record, in two parts.
//
// "live smoke": both lanes are driven through the real pipeline at matched
// (light, skip-watchers) depth and each independently produces a non-empty
// outcome.md naming the phase -- proving the wiring in both real call paths,
// not merely the writer's own contract in isolation. It deliberately does
// NOT assert exact text equality between the two live runs: attempting that
// first surfaced two genuine, pre-existing content divergences unrelated to
// WIRE-07 -- codex_continue.go's own --skip-watchers watcher summary wording
// (fixed in this same commit, since it directly broke the guarantee this
// test exists to prove) and a structural difference in how each lane's
// review report narrates "no reviewers ran" (runCodexContinueReview's
// explicit continueReviewSkippedFlowStep step vs
// externalContinueReviewReport's silent absence of one when no dispatches
// were planned). The second is 198.1's territory (review-report
// construction), not this plan's, and fixing it is out of scope here.
//
// "shape parity" (strict equality): given the SAME underlying
// verification/gates data, writePhaseOutcomeDocument persists byte-identical
// text whether fed a result map shaped the way the direct lane builds one
// (only "current_phase") or the way the finalize lane builds one
// ("continued_phase" already set) -- the specific claim D-11 makes about the
// writer itself: "the same finalizer result map ... not re-derived from a
// second, narrower source."
func TestBothCheckLanesLeaveTheSameOutcomeText(t *testing.T) {
	t.Run("live smoke", func(t *testing.T) {
		saveGlobalsCmd(t)

		directRoot := setupPhaseEndHiveContinueFixture(t, "Both lanes leave the same outcome text (direct)")
		directStore := store
		drivePhaseOutcomeFastLane(t, directRoot)
		directOutcome, err := directStore.ReadFile(continuePlanArtifactsPath(1, "outcome.md"))
		if err != nil {
			t.Fatalf("read direct-lane outcome.md: %v", err)
		}
		for _, want := range []string{"Both lanes leave the same outcome text (direct)", "Phase 1 verified and completed"} {
			if !strings.Contains(string(directOutcome), want) {
				t.Errorf("direct-lane outcome.md missing %q, got:\n%s", want, string(directOutcome))
			}
		}

		finalizeRoot := setupPhaseEndHiveContinueFixture(t, "Both lanes leave the same outcome text (finalize)")
		finalizeStore := store
		drivePhaseOutcomeWrapperLane(t, finalizeRoot)
		finalizeOutcome, err := finalizeStore.ReadFile(continuePlanArtifactsPath(1, "outcome.md"))
		if err != nil {
			t.Fatalf("read finalize-lane outcome.md: %v", err)
		}
		if !strings.Contains(string(finalizeOutcome), "Both lanes leave the same outcome text (finalize)") {
			t.Errorf("finalize-lane outcome.md missing the phase name, got:\n%s", string(finalizeOutcome))
		}
	})

	t.Run("shape parity", func(t *testing.T) {
		testWriterIsLaneShapeAgnostic(t)
	})
}

// testWriterIsLaneShapeAgnostic is the strict-equality half of
// TestBothCheckLanesLeaveTheSameOutcomeText -- see that test's doc comment
// for the full rationale.
func testWriterIsLaneShapeAgnostic(t *testing.T) {
	saveGlobalsCmd(t)

	verification := codexContinueVerificationReport{
		Steps:        []codexVerificationStep{{Name: "build", Passed: true}},
		ChecksPassed: true,
	}
	gates := codexContinueGateReport{Checks: []gateCheck{{Name: "verification_steps_passed", Passed: true}}}
	baseResult := map[string]interface{}{
		"advanced":             true,
		"completed":            false,
		"phase_name":           "Ship the thing",
		"continued_phase_name": "Ship the thing",
		"state":                colony.StateEXECUTING,
		"next":                 "aether build 2",
		"review_depth":         "light",
		"verification":         verification,
		"gates":                gates,
	}
	state := colony.ColonyState{
		Version:      "3.0",
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Ship the thing", Status: colony.PhaseCompleted}}},
	}

	// Direct-lane shape: only "current_phase", exactly as runCodexContinue's
	// own result maps carry it -- no "continued_phase" key present at all.
	directShaped := map[string]interface{}{"current_phase": 1}
	for k, v := range baseResult {
		directShaped[k] = v
	}
	directStore, directDir := newTestStoreCmd(t)
	defer os.RemoveAll(directDir)
	store = directStore
	if err := writePhaseOutcomeDocument(1, directShaped, state); err != nil {
		t.Fatalf("write direct-shaped outcome: %v", err)
	}
	directOutcome, err := directStore.ReadFile(continuePlanArtifactsPath(1, "outcome.md"))
	if err != nil {
		t.Fatalf("read direct-shaped outcome.md: %v", err)
	}

	// Finalize-lane shape: "continued_phase" already set, matching
	// advanceExternalContinue/finalizeBlockedExternalContinue's own
	// construction.
	finalizeShaped := map[string]interface{}{"current_phase": 1, "continued_phase": 1}
	for k, v := range baseResult {
		finalizeShaped[k] = v
	}
	finalizeStore, finalizeDir := newTestStoreCmd(t)
	defer os.RemoveAll(finalizeDir)
	store = finalizeStore
	if err := writePhaseOutcomeDocument(1, finalizeShaped, state); err != nil {
		t.Fatalf("write finalize-shaped outcome: %v", err)
	}
	finalizeOutcome, err := finalizeStore.ReadFile(continuePlanArtifactsPath(1, "outcome.md"))
	if err != nil {
		t.Fatalf("read finalize-shaped outcome.md: %v", err)
	}

	if string(directOutcome) != string(finalizeOutcome) {
		t.Fatalf("writePhaseOutcomeDocument produced different text for the two lanes' equivalent-data shapes:\n%s", firstDiffLine(string(directOutcome), string(finalizeOutcome)))
	}
}
