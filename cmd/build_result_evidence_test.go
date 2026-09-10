package cmd

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 18 (gap closure, 201-VERIFICATION.md D-08/CAP-066) -- this
// file proves both build lanes attach the credited/uncredited file split and
// the decision/learning knowledge deltas to the exact attempt they just
// sealed, that two attempts' evidence never crosses, and that both write
// functions have real production callers on both build lane entry points.

// ---------------------------------------------------------------------------
// TestBothBuildLanesAttachResultEvidenceToTheExactAttempt
// ---------------------------------------------------------------------------

// TestBothBuildLanesAttachResultEvidenceToTheExactAttempt drives one real
// native build and one real external build-finalize against isolated
// fixture roots, then loads each sealed attempt record back through the
// runtime's own accessor and asserts the recorded credited files, uncredited
// files and knowledge deltas. Every expected value is derived the way the
// runtime derives it -- deriveResultFilePrecision and
// deriveBuildKnowledgeDeltas over the exact same resolved dispatches the run
// produced -- never a hand-typed literal path or sentence.
func TestBothBuildLanesAttachResultEvidenceToTheExactAttempt(t *testing.T) {
	assertWorkingTreeUnchanged(t)

	t.Run("native lane", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		forceBuildJSONOutput(t)

		goal := "Prove native-lane result evidence reaches the sealed attempt"
		taskID := "1.1"
		accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			ColonyDepth:  "full",
			CurrentPhase: 0,
			Plan: colony.Plan{
				Phases: []colony.Phase{{
					ID:              1,
					Name:            "Evidence lane",
					Description:     "Prove the native build lane attaches its own result evidence",
					Status:          colony.PhaseReady,
					Tasks:           []colony.Task{{ID: &taskID, Goal: "Implement the evidence path", Status: colony.TaskPending}},
					SuccessCriteria: []string{"Evidence is attached to the exact attempt"},
				}},
			},
		})
		root := accepted.Root
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir to native fixture root: %v", err)
		}
		defer os.Chdir(oldDir)

		rootCmd.SetArgs([]string{"build", "1"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("native build returned error: %v", err)
		}
		var envelope map[string]interface{}
		if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
			t.Fatalf("parse native build output: %v\n%s", err, stdout.(*bytes.Buffer).String())
		}
		if envelope["ok"] != true {
			t.Fatalf("native build did not report ok:true: %v", envelope)
		}

		attemptRel, attempt, ok := loadLatestBuildAttempt(1)
		if !ok {
			t.Fatalf("expected a recorded native build attempt for phase 1")
		}
		if attempt.Status != buildAttemptBuilt {
			t.Fatalf("attempt status = %q, want %q", attempt.Status, buildAttemptBuilt)
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			t.Fatalf("load colony state: %v", err)
		}
		if len(state.Plan.Phases) == 0 {
			t.Fatal("committed state has no phases")
		}
		phase := state.Plan.Phases[0]

		planReality := buildPlanRealityForDispatches(root, phase, attempt.Dispatches)
		wantCredited, wantUncredited := deriveResultFilePrecision(attempt.Dispatches, state.Worktrees, blockedPlanRealityTasks(planReality))
		if !reflect.DeepEqual(attempt.CreditedFiles, wantCredited) {
			t.Errorf("CreditedFiles = %+v, want %+v (derived from the attempt's own dispatches)", attempt.CreditedFiles, wantCredited)
		}
		if !reflect.DeepEqual(attempt.UncreditedFiles, wantUncredited) {
			t.Errorf("UncreditedFiles = %+v, want %+v (derived from the attempt's own dispatches)", attempt.UncreditedFiles, wantUncredited)
		}

		wantDeltas := deriveBuildKnowledgeDeltas(1, attempt.Dispatches)
		if !reflect.DeepEqual(attempt.KnowledgeDeltas, wantDeltas) {
			t.Errorf("KnowledgeDeltas = %+v, want %+v (derived from the attempt's own dispatches)", attempt.KnowledgeDeltas, wantDeltas)
		}
		if len(wantDeltas) == 0 {
			t.Fatal("expected at least one knowledge delta from the synthetic worker's own next-worker-instructions sentence")
		}
		_ = attemptRel
	})

	t.Run("external lane", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		manifest, completion := prepareExternalBuildCompletion(t, root)
		_ = manifest

		// Add an open decision to the first worker's handoff so this lane's
		// deltas exercise both kinds (prepareExternalBuildCompletion already
		// gives every worker a NextWorkerInstructions "learning" sentence).
		completion.Dispatches[0].Handoff.OpenDecisions = []string{"Chose the external evidence file layout for this attempt"}

		result, state, phase, dispatches, err := runCodexBuildFinalize(root, 1, completion, false)
		if err != nil {
			t.Fatalf("external build-finalize returned error: %v", err)
		}
		if result["state"] != colony.StateBUILT {
			t.Fatalf("external build-finalize result state = %v, want %q", result["state"], colony.StateBUILT)
		}

		attemptRel, attempt, ok := loadLatestBuildAttempt(1)
		if !ok {
			t.Fatalf("expected a recorded external build attempt for phase 1")
		}
		if attempt.Status != buildAttemptBuilt {
			t.Fatalf("attempt status = %q, want %q", attempt.Status, buildAttemptBuilt)
		}

		planReality := buildPlanRealityForDispatches(root, phase, dispatches)
		wantCredited, wantUncredited := deriveResultFilePrecision(dispatches, state.Worktrees, blockedPlanRealityTasks(planReality))
		if !reflect.DeepEqual(attempt.CreditedFiles, wantCredited) {
			t.Errorf("CreditedFiles = %+v, want %+v (derived from the attempt's own dispatches)", attempt.CreditedFiles, wantCredited)
		}
		if !reflect.DeepEqual(attempt.UncreditedFiles, wantUncredited) {
			t.Errorf("UncreditedFiles = %+v, want %+v (derived from the attempt's own dispatches)", attempt.UncreditedFiles, wantUncredited)
		}

		wantDeltas := deriveBuildKnowledgeDeltas(1, dispatches)
		if !reflect.DeepEqual(attempt.KnowledgeDeltas, wantDeltas) {
			t.Errorf("KnowledgeDeltas = %+v, want %+v (derived from the attempt's own dispatches)", attempt.KnowledgeDeltas, wantDeltas)
		}
		foundDecision, foundLearning := false, false
		for _, delta := range wantDeltas {
			if delta.Kind == "decision" {
				foundDecision = true
			}
			if delta.Kind == "learning" {
				foundLearning = true
			}
		}
		if !foundDecision || !foundLearning {
			t.Fatalf("expected both a decision and a learning delta, got %+v", wantDeltas)
		}
		_ = attemptRel
	})

	// Warn-never-fail sub-case: mirrors attachBuildFreeCheckReport's own
	// non-fatal treatment. When the store cannot be written to, both
	// evidence writers return an error -- exactly what a caller warns on and
	// continues past -- and the attempt's own already-sealed `built` status
	// is left completely untouched by that failure.
	t.Run("an evidence write failure never fails an otherwise-complete build", func(t *testing.T) {
		setupSpendTestStore(t)
		dispatches := []codexBuildDispatch{{Name: "Mason-Warn", Caste: "builder"}}
		rel, seeded := seedBuildAttemptForTest(t, 1, "attempt-warn-never-fail", func(r *buildAttemptRecord) {
			r.Dispatches = dispatches
			r.Status = buildAttemptBuilt
		})
		if seeded.Status != buildAttemptBuilt {
			t.Fatalf("fixture attempt status = %q, want %q before the forced failure", seeded.Status, buildAttemptBuilt)
		}

		if err := os.Chmod(store.BasePath(), 0o500); err != nil {
			t.Fatalf("chmod store base path read-only: %v", err)
		}
		t.Cleanup(func() { os.Chmod(store.BasePath(), 0o755) })

		writeErr1 := attachResultFilePrecision(rel, []string{"a.go"}, nil)
		writeErr2 := attachBuildKnowledgeDeltas(rel, []buildAttemptKnowledgeDelta{{Kind: "decision", Summary: "should not persist"}})
		if writeErr1 == nil || writeErr2 == nil {
			t.Skip("environment did not make the store directory unwritable (e.g. running as root); nothing to assert")
		}

		os.Chmod(store.BasePath(), 0o755)
		var reloaded buildAttemptRecord
		if err := store.LoadJSON(rel, &reloaded); err != nil {
			t.Fatalf("reload attempt after forced evidence-write failure: %v", err)
		}
		if reloaded.Status != buildAttemptBuilt {
			t.Fatalf("attempt status = %q after a forced evidence-write failure, want it to remain %q (a build that otherwise completed must stay sealed)", reloaded.Status, buildAttemptBuilt)
		}
		if len(reloaded.CreditedFiles) != 0 || len(reloaded.KnowledgeDeltas) != 0 {
			t.Fatalf("evidence appears to have been partially written despite the forced failure: credited=%+v deltas=%+v", reloaded.CreditedFiles, reloaded.KnowledgeDeltas)
		}
	})
}

// ---------------------------------------------------------------------------
// TestAttemptEvidenceNeverCrossesAttempts
// ---------------------------------------------------------------------------

// buildResultEvidenceHandoffDispatch persists a real worker handoff record
// (via persistDispatchWorkerHandoff, the exact production function both
// build lanes call) for a single-worker dispatch, so the fixture below
// exercises the same handoff-persistence path a real build uses rather than
// hand-writing the handoffs file.
func buildResultEvidenceHandoffDispatch(t *testing.T, phase int, workerName string, decision string) {
	t.Helper()
	dispatch := codex.WorkerDispatch{WorkerName: workerName, Caste: "builder", Workflow: "build", Phase: phase}
	result := codex.DispatchResult{
		WorkerName: workerName,
		Status:     "completed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: workerName,
			Caste:      "builder",
			Status:     "completed",
			Handoff: codex.WorkerHandoff{
				VerificationStatus: "pass",
				OpenDecisions:      []string{decision},
			},
		},
	}
	if err := persistDispatchWorkerHandoff(dispatch, result); err != nil {
		t.Fatalf("persist handoff for %s: %v", workerName, err)
	}
}

// TestAttemptEvidenceNeverCrossesAttempts seeds two real attempt records for
// the SAME phase (1) in the same store, each with its own distinguishable
// worker and a real persisted handoff (persistDispatchWorkerHandoff), then
// attaches both pieces of evidence to each attempt through the exact
// production writer/derivation functions both build lanes call
// (deriveResultFilePrecision/attachResultFilePrecision,
// deriveBuildKnowledgeDeltas/attachBuildKnowledgeDeltas). It asserts each
// attempt -- read back by its own recorded path, never by picking the latest
// twice -- carries only its own evidence, even though both attempts' worker
// handoffs coexist under the same phase number in the same handoffs file.
func TestAttemptEvidenceNeverCrossesAttempts(t *testing.T) {
	assertWorkingTreeUnchanged(t)
	setupSpendTestStore(t)

	dispatchesA := []codexBuildDispatch{{Name: "Mason-Cross-A", Caste: "builder"}}
	dispatchesB := []codexBuildDispatch{{Name: "Mason-Cross-B", Caste: "builder"}}

	buildResultEvidenceHandoffDispatch(t, 1, "Mason-Cross-A", "Attempt A chose Postgres for the widget store")
	buildResultEvidenceHandoffDispatch(t, 1, "Mason-Cross-B", "Attempt B chose Redis for the widget cache")

	relA, _ := seedBuildAttemptForTest(t, 1, "attempt-cross-a", func(r *buildAttemptRecord) {
		r.Dispatches = dispatchesA
	})
	relB, _ := seedBuildAttemptForTest(t, 1, "attempt-cross-b", func(r *buildAttemptRecord) {
		r.Dispatches = dispatchesB
	})

	creditedA, uncreditedA := deriveResultFilePrecision(dispatchesA, nil, nil)
	creditedB, uncreditedB := deriveResultFilePrecision(dispatchesB, nil, nil)
	if err := attachResultFilePrecision(relA, creditedA, uncreditedA); err != nil {
		t.Fatalf("attach result file precision A: %v", err)
	}
	if err := attachResultFilePrecision(relB, creditedB, uncreditedB); err != nil {
		t.Fatalf("attach result file precision B: %v", err)
	}

	deltasA := deriveBuildKnowledgeDeltas(1, dispatchesA)
	deltasB := deriveBuildKnowledgeDeltas(1, dispatchesB)
	if err := attachBuildKnowledgeDeltas(relA, deltasA); err != nil {
		t.Fatalf("attach knowledge deltas A: %v", err)
	}
	if err := attachBuildKnowledgeDeltas(relB, deltasB); err != nil {
		t.Fatalf("attach knowledge deltas B: %v", err)
	}

	var loadedA, loadedB buildAttemptRecord
	if err := store.LoadJSON(relA, &loadedA); err != nil {
		t.Fatalf("load attempt A by its own recorded path: %v", err)
	}
	if err := store.LoadJSON(relB, &loadedB); err != nil {
		t.Fatalf("load attempt B by its own recorded path: %v", err)
	}

	if len(loadedA.KnowledgeDeltas) != 1 || !strings.Contains(loadedA.KnowledgeDeltas[0].Summary, "Postgres") {
		t.Fatalf("attempt A's knowledge deltas = %+v, want exactly its own Postgres decision", loadedA.KnowledgeDeltas)
	}
	if len(loadedB.KnowledgeDeltas) != 1 || !strings.Contains(loadedB.KnowledgeDeltas[0].Summary, "Redis") {
		t.Fatalf("attempt B's knowledge deltas = %+v, want exactly its own Redis decision", loadedB.KnowledgeDeltas)
	}
	for _, delta := range loadedA.KnowledgeDeltas {
		if strings.Contains(delta.Summary, "Redis") {
			t.Errorf("attempt A's evidence carries attempt B's delta: %+v", delta)
		}
	}
	for _, delta := range loadedB.KnowledgeDeltas {
		if strings.Contains(delta.Summary, "Postgres") {
			t.Errorf("attempt B's evidence carries attempt A's delta: %+v", delta)
		}
	}
}

// ---------------------------------------------------------------------------
// TestBuildResultEvidenceHasProductionCallers
// ---------------------------------------------------------------------------

// buildResultEvidenceTargets are the two write-side functions this guard
// requires at least one production (non-test) caller for.
var buildResultEvidenceTargets = []string{
	"attachResultFilePrecision",
	"attachBuildKnowledgeDeltas",
}

// buildResultEvidenceEntryPoints names the two build lane entry points that
// must each reach both write-side targets above -- derived from the parsed
// call graph (verificationBoundaryReaches, cmd/verification_boundary_wiring_test.go),
// never hand-maintained.
var buildResultEvidenceEntryPoints = []string{
	"runCodexBuildWithOptions",
	"runCodexBuildFinalize",
}

// buildResultEvidenceOffenders returns, for every entry point in
// buildResultEvidenceEntryPoints, a named violation for each write-side
// target it fails to reach -- the structural signature of a lane that has
// been (or never was) connected to the recording path.
func buildResultEvidenceOffenders(funcs map[string]*ast.FuncDecl) []string {
	var offenders []string
	for _, entry := range buildResultEvidenceEntryPoints {
		if _, ok := funcs[entry]; !ok {
			offenders = append(offenders, entry+" is not declared in the cmd package")
			continue
		}
		for _, target := range buildResultEvidenceTargets {
			if !verificationBoundaryReaches(funcs, entry, target) {
				offenders = append(offenders, entry+" does not reach "+target)
			}
		}
	}
	sort.Strings(offenders)
	return offenders
}

// TestBuildResultEvidenceHasProductionCallers is the structural guard: both
// attachResultFilePrecision and attachBuildKnowledgeDeltas have at least one
// direct caller among the package's non-test functions, and both named build
// lane entry points reach both of them. A synthetic disconnection is refused
// by name and file position, proving the guard can actually fail.
func TestBuildResultEvidenceHasProductionCallers(t *testing.T) {
	assertWorkingTreeUnchanged(t)

	t.Run("both write-side functions have at least one production caller", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		for _, target := range buildResultEvidenceTargets {
			if _, ok := funcs[target]; !ok {
				t.Fatalf("%s is not declared in the cmd package", target)
			}
			callers := continueDecisionDirectCallers(funcs, target)
			if len(callers) == 0 {
				t.Errorf("%s has zero callers outside _test.go files -- the write side is orphaned", target)
			}
		}
	})

	t.Run("both build lane entry points reach the recording path", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		offenders := buildResultEvidenceOffenders(funcs)
		for _, offender := range offenders {
			t.Errorf("%s", offender)
		}
	})

	t.Run("guard fails a synthetic disconnection by name and position", func(t *testing.T) {
		fset := token.NewFileSet()
		src := `package cmd

func runCodexBuildFinalize() {
}
`
		extra, err := parser.ParseFile(fset, "fixture_disconnected_result_evidence_lane.go", src, 0)
		if err != nil {
			t.Fatalf("parse synthetic disconnection fixture: %v", err)
		}
		funcs := continueDecisionPackageFuncs(t, fset, extra)
		offenders := buildResultEvidenceOffenders(funcs)
		if len(offenders) == 0 {
			t.Fatalf("expected the guard to flag the synthetically disconnected lane, got no offenders")
		}
		found := false
		for _, offender := range offenders {
			if strings.Contains(offender, "runCodexBuildFinalize") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected an offender naming runCodexBuildFinalize, got %v", offenders)
		}
		fn := funcs["runCodexBuildFinalize"]
		pos := fset.Position(fn.Pos())
		if pos.Filename == "" || pos.Line == 0 {
			t.Fatalf("violation carries no file/line position: %+v", pos)
		}
	})
}
