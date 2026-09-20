package cmd

// Phase 203 Plan 15, Task 1 (BIO-03's turnaround guarantee, folded from
// .planning/todos/pending/2026-08-27-worker-turnaround-is-too-slow.md): an
// ordinary build or check that never recruits must pay nothing for this
// phase's admission machinery. Every assertion in this file is a COUNTER --
// a call count, a field count, a file count -- never an elapsed-time
// measurement, so none of these tests can become flaky on a loaded machine.

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// TestNoRecruitmentPathCostsNothing
// ---------------------------------------------------------------------------

// creditAndTuningSymbolsForLatencyScan extends cmd/recruitment_test.go's own
// recruitmentSymbols map (the 203-02 tracer's established call-graph-proof
// pattern) to the credit/outcome-tuning symbols this phase's own sibling
// plans (203-12, 203-13) added. Reusing recruitmentSymbols and
// buildDispatchFilesForRecruitmentScan directly (both declared in
// cmd/recruitment_test.go, same package) means this scan never drifts from
// the one already proven to cover the real build-dispatch surface.
var creditAndTuningSymbolsForLatencyScan = map[string]bool{
	"recordRecruitmentCredit":          true,
	"recruitmentCreditAll":             true,
	"recruitmentCreditForDecision":     true,
	"recruitmentCreditForContribution": true,
	"recruitmentCreditFile":            true,
	"recruitmentCreditRecord":          true,
	"tuneNoteStrengthFromOutcomes":     true,
	"runPheromoneOutcomeTuning":        true,
	"probeNativeNestingOnce":           true,
}

// assertNoRecruitmentOrCreditSymbolsInBuildDispatch is the call-graph proof
// this task's own must_have names explicitly ("derived from the call graph
// rather than from a file-access counter alone"): it walks the parsed
// syntax tree of the same ordinary build-dispatch files
// TestPlainBuildPathDoesNotTouchRecruitment (cmd/recruitment_test.go)
// already scans, failing by name if any of them references a recruitment OR
// credit/tuning symbol. A file-access counter alone (nothing written to
// disk) cannot distinguish "the function was never entered" from "the
// function ran and legitimately wrote nothing" -- this static reachability
// check is the stronger claim the must_have asks for.
func assertNoRecruitmentOrCreditSymbolsInBuildDispatch(t *testing.T) {
	t.Helper()
	fset := token.NewFileSet()
	scanned := 0
	var violations []string
	for _, name := range buildDispatchFilesForRecruitmentScan {
		if _, statErr := os.Stat(name); statErr != nil {
			continue
		}
		scanned++
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if recruitmentSymbols[ident.Name] || creditAndTuningSymbolsForLatencyScan[ident.Name] {
				violations = append(violations, fmt.Sprintf(
					"%s: references recruitment/credit symbol %q",
					fset.Position(ident.Pos()).String(), ident.Name,
				))
			}
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("fixture is broken: none of the expected build-dispatch files were found in the cmd package directory")
	}
	if len(violations) != 0 {
		t.Fatalf(
			"plain build-dispatch files reach a recruitment or credit function (call-graph proof, not a file-access counter alone):\n%s",
			strings.Join(violations, "\n"),
		)
	}
}

// TestNoRecruitmentPathCostsNothing drives a real `aether build` through the
// same entry point cmd/codex_build_test.go's own core fixtures use
// (createApprovedAcceptedBuildTestColony + rootCmd.Execute against a single,
// non-recruiting task), with counting doubles wrapped around the platform
// probe launcher and the admission gate -- the two chokepoints a real
// recruitment must pass through. A build that never recruits must call
// neither, must touch no file under the recruitment/credit store prefixes,
// and must reach no recruitment/credit function at all.
func TestNoRecruitmentPathCostsNothing(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

	// recruitmentProbeRunner (cmd/recruitment_probe.go) and
	// spawnCanSpawnDecision (cmd/spawn.go) are both already declared as
	// package-level `var`s specifically so a test can substitute a
	// counting double without touching production code -- wrapping (not
	// replacing) the original means a genuine call, if one ever happened,
	// would still run through the real logic and be counted honestly.
	var probeCalls int32
	origProbeRunner := recruitmentProbeRunner
	recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
		atomic.AddInt32(&probeCalls, 1)
		return origProbeRunner(ctx, binary, dir)
	}
	defer func() { recruitmentProbeRunner = origProbeRunner }()

	var admissionCalls int32
	origDecision := spawnCanSpawnDecision
	spawnCanSpawnDecision = func(in spawnDecisionInput) spawnDecisionResult {
		atomic.AddInt32(&admissionCalls, 1)
		return origDecision(in)
	}
	defer func() { spawnCanSpawnDecision = origDecision }()

	goal := "Ordinary build, no recruitment, no spawn claims"
	taskID := "1.1"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:              1,
					Name:            "Plain phase",
					Description:     "One task, no recruitment, no spawn claims",
					Status:          colony.PhaseReady,
					Tasks:           []colony.Task{{ID: &taskID, Goal: "Do the one thing", Status: colony.TaskPending}},
					SuccessCriteria: []string{"The one thing is done"},
				},
			},
		},
	})
	dataDir, root := accepted.DataRoot, accepted.Root

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}

	rootCmd.SetArgs([]string{"build", "1"})
	buildErr := rootCmd.Execute()

	// Restore cwd to the cmd package directory BEFORE any assertion below:
	// assertNoRecruitmentOrCreditSymbolsInBuildDispatch parses the real
	// build-dispatch source files by relative path, which only resolve from
	// here, not from the temporary colony root the build itself ran in.
	if err := os.Chdir(oldDir); err != nil {
		t.Fatalf("failed to chdir back: %v", err)
	}
	if buildErr != nil {
		t.Fatalf("build returned error: %v", buildErr)
	}

	if got := atomic.LoadInt32(&probeCalls); got != 0 {
		t.Fatalf("a build with no recruitment must launch the platform-nesting probe zero times; it launched %d", got)
	}
	if got := atomic.LoadInt32(&admissionCalls); got != 0 {
		t.Fatalf("a build with no recruitment must reach the recruitment admission gate zero times; it reached it %d time(s)", got)
	}

	for _, prefix := range []string{"recruitment", "credit"} {
		matches, globErr := filepath.Glob(filepath.Join(dataDir, prefix, "*"))
		if globErr != nil {
			t.Fatalf("glob %s: %v", prefix, globErr)
		}
		if len(matches) != 0 {
			t.Fatalf("a build with no recruitment must touch zero files under the %q store prefix; found %v", prefix, matches)
		}
	}

	assertNoRecruitmentOrCreditSymbolsInBuildDispatch(t)
}

// ---------------------------------------------------------------------------
// TestNoNewMandatoryStep
// ---------------------------------------------------------------------------

// buildCheckinDecisionInputBaselineFields is the exact, currently-recorded
// field set of buildCheckinDecisionInput (cmd/ceremony_team_checkin.go) --
// none of them are recruitment-shaped. This is the "recorded baseline"
// TestNoNewMandatoryStep compares against: a NEW field on this struct would
// be the actual mechanism by which recruitment (or anything else this phase
// adds) could thread a forced pause into the build check-in decision, so a
// changed field set fails this test by name rather than the test silently
// passing on an unnoticed addition.
var buildCheckinDecisionInputBaselineFields = []string{
	"Autopilot",
	"NoCheckin",
	"Checkin",
	"PendingOwnerDecision",
	"PendingOwnerDecisionWhy",
	"ImplementationDispatches",
}

// TestNoNewMandatoryStep proves the folded-todo constraint ("nothing in this
// phase may add a new mandatory ceremony step") as a structural, breakable
// check rather than a comment:
//
//  1. decideBuildCheckin (cmd/ceremony_team_checkin.go) has exactly ONE
//     call site outside its own definition -- a second call site would be a
//     second, competing pause decision this phase (or anything else) could
//     have wired in without this test noticing a raw literal count change.
//  2. buildCheckinDecisionInput's field set is byte-identical to the
//     recorded baseline above -- a new field is the concrete mechanism a
//     future change would need to actually add a pause, so this is where a
//     real addition would show up.
//  3. The runtime's OWN unmodified policy function, called with the exact
//     "one worker, nothing pending" scenario the pre-existing
//     TestOneWorkerBuildSkipsCheckin already locks, still resolves to the
//     recorded fast-path outcome (buildCheckinReasonOneWorkerFastPath, an
//     existing named constant -- never a bare literal 0 or false typed here)
//     -- the owner-facing pause count for a no-recruitment build is
//     identical to the count before this phase.
func TestNoNewMandatoryStep(t *testing.T) {
	// 1. Exactly one call site.
	callSites := 0
	fset := token.NewFileSet()
	matches, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob cmd/*.go: %v", err)
	}
	for _, name := range matches {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
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
			if !ok || ident.Name != "decideBuildCheckin" {
				return true
			}
			callSites++
			return true
		})
	}
	if callSites != 1 {
		t.Fatalf(
			"decideBuildCheckin must have exactly one call site outside its own definition (a second call site is a second, competing pause decision); found %d",
			callSites,
		)
	}

	// 2. The decision input's field set is unchanged.
	gotFields := []string{}
	typ := reflect.TypeOf(buildCheckinDecisionInput{})
	for i := 0; i < typ.NumField(); i++ {
		gotFields = append(gotFields, typ.Field(i).Name)
	}
	if len(gotFields) != len(buildCheckinDecisionInputBaselineFields) {
		t.Fatalf(
			"buildCheckinDecisionInput's field count changed from the recorded baseline %v to %v -- a new field is how a forced pause could be threaded in without this test noticing",
			buildCheckinDecisionInputBaselineFields, gotFields,
		)
	}
	for i, want := range buildCheckinDecisionInputBaselineFields {
		if gotFields[i] != want {
			t.Fatalf("buildCheckinDecisionInput field %d changed from the recorded baseline %q to %q", i, want, gotFields[i])
		}
	}

	// 3. The real, unmodified policy still resolves the no-recruitment,
	// one-worker scenario to the recorded fast-path outcome (0 pauses).
	decision := decideBuildCheckin(buildCheckinDecisionInput{ImplementationDispatches: 1})
	if decision.Requested {
		t.Fatalf(
			"the owner-facing pause count for a no-recruitment, one-worker build changed from the recorded baseline (fast path, 0 pauses): decideBuildCheckin now requests one (%s: %s)",
			decision.Reason, decision.Why,
		)
	}
	if decision.Reason != buildCheckinReasonOneWorkerFastPath {
		t.Fatalf(
			"the recorded fast-path reason changed from %q to %q for the no-recruitment, one-worker build",
			buildCheckinReasonOneWorkerFastPath, decision.Reason,
		)
	}
}

// ---------------------------------------------------------------------------
// TestTuningPassIsFreeWithoutCredit
// ---------------------------------------------------------------------------

// TestTuningPassIsFreeWithoutCredit proves the second half of the folded-todo
// constraint for the check (continue) side: a check with no credit records
// reaches the outcome-weighted tuning pass (cmd/pheromone_outcome.go's
// tuneNoteStrengthFromOutcomes, wired to both continue lanes via
// cmd/phase_end_signals.go's runPheromoneOutcomeTuning) but performs ZERO
// store writes -- proven by file existence, not by trusting the function's
// own report of what it did.
func TestTuningPassIsFreeWithoutCredit(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	dataDir := s.BasePath()

	// No credit/records.json exists in this fresh store -- the "empty
	// credit store" case the must_have names.
	result := runPheromoneOutcomeTuning()
	if !result.Ran {
		t.Fatalf("the tuning pass must be able to run against an empty credit store (Ran=false, Error=%q)", result.Error)
	}
	if result.RecordsConsidered != 0 || result.NotesTuned != 0 {
		t.Fatalf("an empty credit store must yield zero considered/tuned records, got RecordsConsidered=%d NotesTuned=%d", result.RecordsConsidered, result.NotesTuned)
	}

	// Zero writes: with zero records, tuneNoteStrengthFromOutcomes returns
	// before ever calling loadPheromoneOutcomeState/savePheromoneOutcomeState
	// -- its own bookkeeping file must not exist.
	if _, statErr := os.Stat(filepath.Join(dataDir, pheromoneOutcomeStatePath)); statErr == nil {
		t.Fatalf("the tuning pass wrote %q against an empty credit store; a tuning pass with nothing to tune must write nothing", pheromoneOutcomeStatePath)
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("stat %q: %v", pheromoneOutcomeStatePath, statErr)
	}

	// The pass must also never touch the pheromone signal store or its
	// influence history when there is nothing to tune.
	for _, path := range []string{"pheromones.json", "pheromones-history.json"} {
		if _, statErr := os.Stat(filepath.Join(dataDir, path)); statErr == nil {
			t.Fatalf("the tuning pass wrote %q against an empty credit store; a tuning pass with nothing to tune must write nothing", path)
		} else if !os.IsNotExist(statErr) {
			t.Fatalf("stat %q: %v", path, statErr)
		}
	}
}
