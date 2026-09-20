package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 16 (gap closure, 201-VERIFICATION.md D-01) -- the
// verification-boundary decision (cmd/verification_boundary.go) and its
// persisted record (cmd/build_attempt.go) existed and passed their own unit
// tests, but nothing in production ever called either one: the Queen's
// build-end-versus-check-step choice never reached a real build. This file
// proves the write side is wired, and that a real build actually records and
// acts on the choice.

// --- Task 3a: an AST call-graph guard refusing a silent disconnection ---

// verificationBoundaryWriteSideTargets are the two write-side functions this
// guard requires at least one production (non-test) caller for: the
// reconciliation function (cmd/verification_boundary.go) and the
// stored-record writer (cmd/build_attempt.go). These two names identify the
// shared INFRASTRUCTURE this guard protects, mirroring
// continueDecisionGateEvaluatorNames' own doc comment
// (cmd/codex_verify_advance_test.go) -- not a maintained list of every
// caller.
var verificationBoundaryWriteSideTargets = []string{
	"queenApplyVerificationBoundary",
	"attachVerificationBoundary",
}

// verificationBoundaryWriteSideEntryPoints names the two build lane entry
// points that must each transitively reach BOTH write-side targets above.
// Which functions actually reach them is derived from the parsed call graph
// (verificationBoundaryReaches), never hand-maintained.
var verificationBoundaryWriteSideEntryPoints = []string{
	"runCodexBuildPlanOnlyWithOptions",
	"runCodexBuildWithOptions",
}

// verificationBoundaryReaches reports whether from transitively reaches to
// through funcs' direct-call graph (continueDecisionDirectCalleeNames,
// cmd/codex_verify_advance_test.go), breadth-first with a visited set so a
// cycle or mutual recursion in the real package cannot loop forever. This is
// the same "which OTHER top-level function does a call reach" authority
// level that guard already uses, extended one hop at a time rather than
// requiring a DIRECT call, because the reconciling call site
// (prepareDirectCodexBuild) and the direct-lane entry point
// (runCodexBuildWithOptions) are not the same function.
func verificationBoundaryReaches(funcs map[string]*ast.FuncDecl, from, to string) bool {
	if from == to {
		return true
	}
	visited := map[string]bool{from: true}
	queue := []string{from}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		fn, ok := funcs[name]
		if !ok {
			continue
		}
		for callee := range continueDecisionDirectCalleeNames(fn) {
			if callee == to {
				return true
			}
			if !visited[callee] {
				visited[callee] = true
				queue = append(queue, callee)
			}
		}
	}
	return false
}

// verificationBoundaryWriteSideOffenders returns, for every entry point in
// verificationBoundaryWriteSideEntryPoints, a named violation for each write-
// side target it fails to transitively reach -- the structural signature of
// a lane that has been (or never was) connected to the recording path.
func verificationBoundaryWriteSideOffenders(funcs map[string]*ast.FuncDecl) []string {
	var offenders []string
	for _, entry := range verificationBoundaryWriteSideEntryPoints {
		if _, ok := funcs[entry]; !ok {
			offenders = append(offenders, entry+" is not declared in the cmd package")
			continue
		}
		for _, target := range verificationBoundaryWriteSideTargets {
			if !verificationBoundaryReaches(funcs, entry, target) {
				offenders = append(offenders, entry+" does not reach "+target)
			}
		}
	}
	sort.Strings(offenders)
	return offenders
}

// gitWorkingTreeSnapshot captures `git status --porcelain` for the real
// source checkout this test runs inside (never the isolated fixture root a
// subtest builds via bindCommandTestRepository) so the caller can prove, by
// a before-and-after comparison, that this test file mutated nothing outside
// its own temp-dir fixtures. Skips (rather than fails) when git is
// unavailable -- this is a mutation guard, not a git-availability test.
func gitWorkingTreeSnapshot(t *testing.T) string {
	t.Helper()
	topLevel, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skipf("git not available to snapshot the source checkout: %v", err)
		return ""
	}
	out, err := exec.Command("git", "-C", strings.TrimSpace(string(topLevel)), "status", "--porcelain").Output()
	if err != nil {
		t.Skipf("git status failed while snapshotting the source checkout: %v", err)
		return ""
	}
	return string(out)
}

// assertWorkingTreeUnchanged registers a cleanup that fails the test if the
// real source checkout's working tree differs from the snapshot taken at
// call time -- the before-and-after proof both tests in this file carry.
func assertWorkingTreeUnchanged(t *testing.T) {
	t.Helper()
	before := gitWorkingTreeSnapshot(t)
	t.Cleanup(func() {
		after := gitWorkingTreeSnapshot(t)
		if before != after {
			t.Fatalf("test mutated the source checkout's working tree\nbefore:\n%s\nafter:\n%s", before, after)
		}
	})
}

// TestBoundaryWriteSideHasProductionCallers is the structural guard: both
// queenApplyVerificationBoundary and attachVerificationBoundary have at
// least one direct caller among the package's non-test functions, and both
// named build lane entry points transitively reach both of them. A synthetic
// disconnection is refused by name and file position, proving the guard can
// actually fail.
func TestBoundaryWriteSideHasProductionCallers(t *testing.T) {
	assertWorkingTreeUnchanged(t)

	t.Run("both write-side functions have at least one production caller", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		for _, target := range verificationBoundaryWriteSideTargets {
			if _, ok := funcs[target]; !ok {
				t.Fatalf("%s is not declared in the cmd package", target)
			}
			callers := continueDecisionDirectCallers(funcs, target)
			if len(callers) == 0 {
				t.Errorf("%s has zero callers outside _test.go files -- the write side is orphaned", target)
			}
		}
	})

	t.Run("the two build lane entry points reach the recording path", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		offenders := verificationBoundaryWriteSideOffenders(funcs)
		for _, offender := range offenders {
			t.Errorf("%s", offender)
		}
	})

	t.Run("guard fails a synthetic disconnection by name and position", func(t *testing.T) {
		fset := token.NewFileSet()
		src := `package cmd

func runCodexBuildPlanOnlyWithOptions() {
}
`
		extra, err := parser.ParseFile(fset, "fixture_disconnected_boundary_lane.go", src, 0)
		if err != nil {
			t.Fatalf("parse synthetic disconnection fixture: %v", err)
		}
		funcs := continueDecisionPackageFuncs(t, fset, extra)
		offenders := verificationBoundaryWriteSideOffenders(funcs)
		if len(offenders) == 0 {
			t.Fatalf("expected the guard to flag the synthetically disconnected lane, got no offenders")
		}
		found := false
		for _, offender := range offenders {
			if strings.Contains(offender, "runCodexBuildPlanOnlyWithOptions") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected an offender naming runCodexBuildPlanOnlyWithOptions, got %v", offenders)
		}
		fn := funcs["runCodexBuildPlanOnlyWithOptions"]
		pos := fset.Position(fn.Pos())
		if pos.Filename == "" || pos.Line == 0 {
			t.Fatalf("violation carries no file/line position: %+v", pos)
		}
	})
}

// --- Task 3b: end-to-end proof that a real build records and acts on it ---

// verificationBoundaryFixturePhase builds a minimal, real single-task phase
// for driving the plan-only build lane -- production mode so
// applySpecialRules' mode==production condition and auditor's non-zero base
// relevance score both cooperate with a proposed "auditor" caste without
// relying on keyword luck.
func verificationBoundaryFixturePhase() colony.Phase {
	taskID := "1.1"
	return colony.Phase{
		ID:          1,
		Name:        "Release readiness",
		Description: "Prepare the release artifact for sign-off",
		Mode:        colony.PhaseModeProduction,
		Status:      colony.PhaseReady,
		Tasks: []colony.Task{{
			ID:     &taskID,
			Goal:   "Implement the release readiness check",
			Status: colony.TaskPending,
		}},
	}
}

// verificationBoundaryFixtureState seeds an isolated repository (via
// setupBuildFlowTest/createTestColonyState, the same pattern
// TestBuildPlanOnlyExecutionPlanRunsWatcherAfterSpecialists uses) with one
// ready phase, and returns the build root the plan-only lane needs.
func verificationBoundaryFixtureState(t *testing.T) string {
	t.Helper()
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Ship the release"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan:         colony.Plan{Phases: []colony.Phase{verificationBoundaryFixturePhase()}},
	})
	return root
}

// verificationBoundaryPostWaveStages are the stages queenBuildPostWavePlans
// (cmd/codex_build.go) dispatches into -- the observable signature of "a
// build-end reviewer actually reached this build's own dispatch list."
var verificationBoundaryPostWaveStages = map[string]bool{"audit": true, "measurement": true, "resilience": true}

// hasPostWaveReviewerDispatch reports whether dispatches contains any
// dispatch queenBuildPostWaveDispatches could have produced.
func hasPostWaveReviewerDispatch(dispatches []codexBuildDispatch) bool {
	for _, d := range dispatches {
		if verificationBoundaryPostWaveStages[d.Stage] {
			return true
		}
	}
	return false
}

// TestQueenBoundaryChoiceReachesTheRecordedAttempt drives three real
// plan-only builds against isolated fixture roots and proves, for each, that
// the attempt the build actually wrote carries the decision
// queenApplyVerificationBoundary reconciled from the caller's proposal --
// read back exclusively through verificationBoundaryForAttempt, never by
// hand-decoding JSON -- and that a build-end decision (and only a build-end
// decision) produces a post-wave reviewer in that SAME build's own dispatch
// list.
func TestQueenBoundaryChoiceReachesTheRecordedAttempt(t *testing.T) {
	assertWorkingTreeUnchanged(t)

	t.Run("no proposal records the check-step default and dispatches no build-end reviewer", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		root := verificationBoundaryFixtureState(t)

		result, _, _, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{})
		if err != nil {
			t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
		}

		attemptRel, _, ok := loadLatestBuildAttempt(1)
		if !ok {
			t.Fatalf("expected a recorded build attempt for phase 1")
		}
		decision, ok := verificationBoundaryForAttempt(attemptRel)
		if !ok {
			t.Fatalf("expected a recorded verification-boundary decision on the attempt")
		}
		if decision.Choice != verificationBoundaryChoiceCheckStep {
			t.Fatalf("Choice = %q, want %q", decision.Choice, verificationBoundaryChoiceCheckStep)
		}
		if decision.Source != verificationBoundarySourceDeterministic {
			t.Fatalf("Source = %q, want %q", decision.Source, verificationBoundarySourceDeterministic)
		}
		if decision.Refused {
			t.Fatalf("expected no refusal for an empty proposal, got %+v", decision)
		}

		manifest, ok := result["dispatch_manifest"].(codexBuildManifest)
		if !ok {
			t.Fatalf("dispatch_manifest missing or wrong type: %T", result["dispatch_manifest"])
		}
		if hasPostWaveReviewerDispatch(manifest.Dispatches) {
			t.Fatalf("expected no post-wave reviewer dispatch with no proposal, got %+v", manifest.Dispatches)
		}
		if hasPostWaveReviewerDispatch(dispatches) {
			t.Fatalf("expected no post-wave reviewer dispatch in the returned dispatch list, got %+v", dispatches)
		}
	})

	t.Run("build end with a reason records it and dispatches build-end reviewers in this build", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		root := verificationBoundaryFixtureState(t)

		result, _, _, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{
			HeavyFlag:                    true,
			QueenCastes:                  []string{"auditor"},
			QueenCasteWhy:                []string{"auditor=quality gate for this release"},
			QueenVerificationBoundary:    "build_end",
			QueenVerificationBoundaryWhy: "release sign-off",
		})
		if err != nil {
			t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
		}

		attemptRel, _, ok := loadLatestBuildAttempt(1)
		if !ok {
			t.Fatalf("expected a recorded build attempt for phase 1")
		}
		decision, ok := verificationBoundaryForAttempt(attemptRel)
		if !ok {
			t.Fatalf("expected a recorded verification-boundary decision on the attempt")
		}
		if decision.Choice != verificationBoundaryChoiceBuildEnd {
			t.Fatalf("Choice = %q, want %q", decision.Choice, verificationBoundaryChoiceBuildEnd)
		}
		if decision.Source != verificationBoundarySourceQueen {
			t.Fatalf("Source = %q, want %q", decision.Source, verificationBoundarySourceQueen)
		}
		if decision.Reason != "release sign-off" {
			t.Fatalf("Reason = %q, want %q", decision.Reason, "release sign-off")
		}
		if decision.Refused {
			t.Fatalf("expected no refusal for a reasoned build-end proposal, got %+v", decision)
		}

		manifest, ok := result["dispatch_manifest"].(codexBuildManifest)
		if !ok {
			t.Fatalf("dispatch_manifest missing or wrong type: %T", result["dispatch_manifest"])
		}
		if !hasPostWaveReviewerDispatch(manifest.Dispatches) {
			t.Fatalf("expected at least one post-wave reviewer dispatch in this build's own manifest, got %+v", manifest.Dispatches)
		}
		if !hasPostWaveReviewerDispatch(dispatches) {
			t.Fatalf("expected at least one post-wave reviewer dispatch in this build's own returned dispatch list, got %+v", dispatches)
		}
	})

	t.Run("build end with no reason records the refused check-step fallback and dispatches nothing extra", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		root := verificationBoundaryFixtureState(t)

		result, _, _, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{
			QueenVerificationBoundary: "build_end",
		})
		if err != nil {
			t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
		}

		attemptRel, _, ok := loadLatestBuildAttempt(1)
		if !ok {
			t.Fatalf("expected a recorded build attempt for phase 1")
		}
		decision, ok := verificationBoundaryForAttempt(attemptRel)
		if !ok {
			t.Fatalf("expected a recorded verification-boundary decision on the attempt")
		}
		if !decision.Refused {
			t.Fatalf("expected the reasonless build-end proposal to be refused, got %+v", decision)
		}
		if strings.TrimSpace(decision.RefusedWhy) == "" {
			t.Fatalf("expected a non-empty RefusedWhy, got %+v", decision)
		}
		if decision.Choice != verificationBoundaryChoiceCheckStep {
			t.Fatalf("Choice = %q, want fallback to %q", decision.Choice, verificationBoundaryChoiceCheckStep)
		}

		manifest, ok := result["dispatch_manifest"].(codexBuildManifest)
		if !ok {
			t.Fatalf("dispatch_manifest missing or wrong type: %T", result["dispatch_manifest"])
		}
		if hasPostWaveReviewerDispatch(manifest.Dispatches) {
			t.Fatalf("expected no post-wave reviewer dispatch for a refused proposal, got %+v", manifest.Dispatches)
		}
		if hasPostWaveReviewerDispatch(dispatches) {
			t.Fatalf("expected no post-wave reviewer dispatch in the returned dispatch list, got %+v", dispatches)
		}
	})
}
