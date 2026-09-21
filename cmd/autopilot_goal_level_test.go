package cmd

import (
	"context"
	"errors"
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

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 11 (WORK-07, CAP-029) -- one accepted goal is enough: the
// controller picks the next transition itself, stops only at one of four
// declared boundaries naming which, and the quick one-question path runs on
// the same attempt/evidence model as the rest of the work cycle.

// ---------------------------------------------------------------------------
// Task 1: goal-level transition selection
// ---------------------------------------------------------------------------

// TestAutopilotSelectsTransitionsFromOneAcceptedGoal proves
// selectAutopilotGoalTransition (cmd/autopilot_policy.go) names the right
// transition, with a non-empty reason, for each of the seven declared
// states -- survey, planning, work, verification, bounded repair, replan,
// and ready-to-seal -- and that the real autopilot lane reaches the two
// shared decision bodies (runContinueAcceptVerifyAdvance,
// runBoundedRepairRound) it must consume rather than reimplementing its own
// interpretation.
func TestAutopilotSelectsTransitionsFromOneAcceptedGoal(t *testing.T) {
	cases := []struct {
		name      string
		facts     autopilotGoalLevelFacts
		wantTrans autopilotGoalTransition
		wantPhase int
	}{
		{
			name:      "no survey evidence selects survey",
			facts:     autopilotGoalLevelFacts{AcceptedGoal: "Build the thing"},
			wantTrans: autopilotTransitionSurvey,
		},
		{
			name: "survey evidence but no accepted plan selects planning",
			facts: autopilotGoalLevelFacts{
				AcceptedGoal:      "Build the thing",
				HasSurveyEvidence: true,
			},
			wantTrans: autopilotTransitionPlanning,
		},
		{
			name: "accepted plan and unbuilt work selects work",
			facts: autopilotGoalLevelFacts{
				AcceptedGoal:      "Build the thing",
				HasSurveyEvidence: true,
				HasAcceptedPlan:   true,
				RemainingPhaseIDs: []int{3, 4, 5},
			},
			wantTrans: autopilotTransitionWork,
			wantPhase: 3,
		},
		{
			name: "built unverified work selects verification",
			facts: autopilotGoalLevelFacts{
				AcceptedGoal:      "Build the thing",
				HasSurveyEvidence: true,
				HasAcceptedPlan:   true,
				RemainingPhaseIDs: []int{3, 4, 5},
				WorkBuilt:         true,
			},
			wantTrans: autopilotTransitionVerification,
			wantPhase: 3,
		},
		{
			name: "a failed verification selects the bounded repair round",
			facts: autopilotGoalLevelFacts{
				AcceptedGoal:       "Build the thing",
				HasSurveyEvidence:  true,
				HasAcceptedPlan:    true,
				RemainingPhaseIDs:  []int{3, 4, 5},
				WorkBuilt:          true,
				VerificationFailed: true,
			},
			wantTrans: autopilotTransitionBoundedRepair,
			wantPhase: 3,
		},
		{
			name: "a due replan cadence selects replan",
			facts: autopilotGoalLevelFacts{
				AcceptedGoal:      "Build the thing",
				HasSurveyEvidence: true,
				HasAcceptedPlan:   true,
				RemainingPhaseIDs: []int{3, 4, 5},
				ReplanDue:         true,
			},
			wantTrans: autopilotTransitionReplan,
			wantPhase: 3,
		},
		{
			name: "no phase remaining selects ready-to-seal",
			facts: autopilotGoalLevelFacts{
				AcceptedGoal:      "Build the thing",
				HasSurveyEvidence: true,
				HasAcceptedPlan:   true,
			},
			wantTrans: autopilotTransitionReadyToSeal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := selectAutopilotGoalTransition(tc.facts)
			if decision.Transition != tc.wantTrans {
				t.Fatalf("transition = %q, want %q (decision=%+v)", decision.Transition, tc.wantTrans, decision)
			}
			if strings.TrimSpace(decision.Reason) == "" {
				t.Fatalf("expected a non-empty reason, got %+v", decision)
			}
			if decision.PhaseID != tc.wantPhase {
				t.Fatalf("phase = %d, want %d (decision=%+v)", decision.PhaseID, tc.wantPhase, decision)
			}
		})
	}

	t.Run("the autopilot lane reaches the shared accept/verify/advance body", func(t *testing.T) {
		// runCompatibilityAutopilot's continue stage calls runAutopilotContinue,
		// a package-level alias for runCodexContinue (cmd/compatibility_cmds.go)
		// -- the exact direct-lane function 201-02 proved reaches
		// runContinueAcceptVerifyAdvance for both its pre-review and post-review
		// decisions. This is real production reachability, not a test-only
		// substitution: the alias is only ever swapped in tests, never in
		// runCompatibilityAutopilot's own production call.
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		aliases := autopilotDispatchAliasMap(t, fset)
		if !autopilotCallGraphReaches(funcs, aliases, "runCompatibilityAutopilot", "runContinueAcceptVerifyAdvance") {
			t.Fatal("runCompatibilityAutopilot does not reach runContinueAcceptVerifyAdvance -- the autopilot lane must consume the shared decision body, not reimplement it")
		}
	})

	t.Run("runBoundedRepairRound exists but is not yet reached by any production lane (documented gap)", func(t *testing.T) {
		// D-09's one bounded, checkpointed repair path (cmd/work_repair.go) is
		// fully implemented and unit-tested (cmd/work_repair_test.go), but --
		// discovered while proving this task's own acceptance criteria --
		// grep and this call-graph both confirm it has ZERO production
		// callers anywhere in the cmd package: the direct check-fix lane
		// (applyBoundedCheckFixRepair, cmd/work_repair.go) still reimplements
		// its own checkpoint/save/restore sequence inline rather than
		// delegating to it. Safely rewiring applyBoundedCheckFixRepair to
		// delegate through runBoundedRepairRound is out of this plan's scope
		// (cmd/work_repair.go is not in this plan's declared file list, and
		// runBoundedRepairRound's eligibility gate requires a non-empty
		// PermittedScope that repairScopePathsForCheckFix legitimately
		// returns empty for some real check-fix fixtures -- closing this gap
		// needs its own plan, not a same-file bolt-on). This test documents
		// the gap by name rather than silently asserting a false reachability
		// claim; see 201-11-SUMMARY.md's Deviations section.
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		if _, ok := funcs["runBoundedRepairRound"]; !ok {
			t.Fatal("expected runBoundedRepairRound to be declared in the cmd package")
		}
		aliases := autopilotDispatchAliasMap(t, fset)
		if autopilotCallGraphReaches(funcs, aliases, "runCompatibilityAutopilot", "runBoundedRepairRound") {
			t.Fatal("runBoundedRepairRound has become reachable from runCompatibilityAutopilot -- update this test (and 201-11-SUMMARY.md) to reflect the gap closing, it is no longer a documented gap")
		}
	})
}

// autopilotDispatchAliasMap scans every package-level "var X = Y" function-
// value assignment in the cmd package (e.g. runAutopilotContinue =
// runCodexContinue, cmd/compatibility_cmds.go) and returns it as an X -> Y
// edge. A plain AST call-graph walk sees "runAutopilotContinue(...)" as a
// call to the identifier "runAutopilotContinue", not to the function it
// aliases -- this map is what lets autopilotCallGraphReaches follow that
// indirection to the real declaration.
func autopilotDispatchAliasMap(t *testing.T, fset *token.FileSet) map[string]string {
	t.Helper()
	aliases := map[string]string{}
	pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse cmd package for the dispatch alias map: %v", err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.VAR {
					continue
				}
				for _, spec := range gen.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Names) != len(vs.Values) {
						continue
					}
					for i, name := range vs.Names {
						if ident, ok := vs.Values[i].(*ast.Ident); ok {
							aliases[name.Name] = ident.Name
						}
					}
				}
			}
		}
	}
	return aliases
}

// autopilotCallGraphReaches reports whether target is reachable from root by
// following direct callee names (continueDecisionDirectCalleeNames, the
// same call-graph-derivation precedent codex_verify_advance_test.go's own
// AST guard established), resolving a package-level function-value alias
// (autopilotDispatchAliasMap) as one further edge at each step.
func autopilotCallGraphReaches(funcs map[string]*ast.FuncDecl, aliases map[string]string, root, target string) bool {
	visited := map[string]bool{}
	var visit func(name string) bool
	visit = func(name string) bool {
		if name == target {
			return true
		}
		if visited[name] {
			return false
		}
		visited[name] = true
		if resolved, ok := aliases[name]; ok && visit(resolved) {
			return true
		}
		fn, ok := funcs[name]
		if !ok {
			return false
		}
		for callee := range continueDecisionDirectCalleeNames(fn) {
			if visit(callee) {
				return true
			}
		}
		return false
	}
	return visit(root)
}

// TestAutopilotLastPhaseAndBeyond proves the two boundary cases of goal-
// level transition selection: at the last remaining phase, the controller
// runs it and the very next selection (after that phase completes) is
// ready-to-seal; with no remaining phase at all, the controller selects
// ready-to-seal directly, naming that nothing remains; and no fixture, in
// any combination of built/verification-failed inputs, ever selects a phase
// beyond the last one recorded as remaining.
func TestAutopilotLastPhaseAndBeyond(t *testing.T) {
	t.Run("the last remaining phase runs and then yields ready-to-seal", func(t *testing.T) {
		facts := autopilotGoalLevelFacts{
			AcceptedGoal:      "Build the thing",
			HasSurveyEvidence: true,
			HasAcceptedPlan:   true,
			RemainingPhaseIDs: []int{9},
		}
		running := selectAutopilotGoalTransition(facts)
		if running.Transition != autopilotTransitionWork || running.PhaseID != 9 {
			t.Fatalf("expected work on phase 9, got %+v", running)
		}
		// Phase 9 completes: the recorded remaining set is now empty.
		facts.RemainingPhaseIDs = nil
		afterward := selectAutopilotGoalTransition(facts)
		if afterward.Transition != autopilotTransitionReadyToSeal {
			t.Fatalf("expected ready_to_seal after the last phase completed, got %+v", afterward)
		}
	})

	t.Run("no remaining phase dispatches nothing and states there is nothing remaining", func(t *testing.T) {
		facts := autopilotGoalLevelFacts{
			AcceptedGoal:      "Build the thing",
			HasSurveyEvidence: true,
			HasAcceptedPlan:   true,
		}
		decision := selectAutopilotGoalTransition(facts)
		if decision.Transition != autopilotTransitionReadyToSeal {
			t.Fatalf("expected ready_to_seal, got %+v", decision)
		}
		if decision.PhaseID != 0 {
			t.Fatalf("expected no phase to be selected, got %d", decision.PhaseID)
		}
		if !strings.Contains(strings.ToLower(decision.Reason), "nothing remaining") {
			t.Fatalf("expected the reason to state nothing remains, got %q", decision.Reason)
		}
	})

	t.Run("no fixture selects a phase past the last remaining one", func(t *testing.T) {
		remaining := []int{2, 5, 9}
		last := remaining[len(remaining)-1]
		for _, workBuilt := range []bool{false, true} {
			for _, verificationFailed := range []bool{false, true} {
				for _, replanDue := range []bool{false, true} {
					facts := autopilotGoalLevelFacts{
						AcceptedGoal:       "Build the thing",
						HasSurveyEvidence:  true,
						HasAcceptedPlan:    true,
						RemainingPhaseIDs:  remaining,
						WorkBuilt:          workBuilt,
						VerificationFailed: verificationFailed,
						ReplanDue:          replanDue,
					}
					decision := selectAutopilotGoalTransition(facts)
					if decision.PhaseID > last {
						t.Fatalf("selected phase %d is beyond the last remaining phase %d (facts=%+v)", decision.PhaseID, last, facts)
					}
					if decision.PhaseID != 0 && decision.PhaseID != remaining[0] {
						t.Fatalf("selected phase %d must be the first remaining phase %d (facts=%+v)", decision.PhaseID, remaining[0], facts)
					}
				}
			}
		}
	})
}

// TestAutopilotInvalidEntryIsZeroWrite proves that an invalid or absent
// accepted goal writes nothing at all: the pure selector returns the zero
// transition, and the real entry point (runCompatibilityAutopilot) against a
// colony with no accepted goal leaves the data directory's on-disk digest
// unchanged.
func TestAutopilotInvalidEntryIsZeroWrite(t *testing.T) {
	t.Run("the pure selector writes nothing for an absent accepted goal", func(t *testing.T) {
		decision := selectAutopilotGoalTransition(autopilotGoalLevelFacts{
			HasSurveyEvidence: true,
			HasAcceptedPlan:   true,
			RemainingPhaseIDs: []int{1},
		})
		if decision.Transition != "" {
			t.Fatalf("expected the zero transition for an absent accepted goal, got %+v", decision)
		}
		if strings.TrimSpace(decision.Reason) == "" {
			t.Fatalf("expected a non-empty reason even for the zero transition, got %+v", decision)
		}
	})

	t.Run("the real entry point leaves the data directory digest unchanged", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		dataDir := filepath.Join(root, ".aether", "data")

		before := hashDirForTest(t, dataDir)
		result, err := runCompatibilityAutopilot(root, runCompatibilityOptions{Headless: true})
		if err != nil {
			t.Fatalf("runCompatibilityAutopilot: %v", err)
		}
		preflight, ok := result["preflight"].(AutopilotPreflight)
		if !ok {
			t.Fatalf("expected an AutopilotPreflight in the result, got %T", result["preflight"])
		}
		if preflight.Valid {
			t.Fatalf("expected an invalid preflight (no accepted goal), got %+v", preflight)
		}
		if preflight.NextTransition.Transition != "" {
			t.Fatalf("expected the zero transition for an invalid entry, got %+v", preflight.NextTransition)
		}
		after := hashDirForTest(t, dataDir)
		if before != after {
			t.Fatalf("data directory digest changed on an invalid entry: before=%s after=%s", before, after)
		}
	})
}

// ---------------------------------------------------------------------------
// Task 2: the four declared stop boundaries
// ---------------------------------------------------------------------------

// autopilotStopBoundaryFixtures pairs each declared boundary with one
// already-recorded trigger code that reaches it -- the exact recorded facts
// autopilotRunDecisionForCode already classifies elsewhere in this package,
// never a synthetic code invented only for this test.
func autopilotStopBoundaryFixtures() map[autopilotStopBoundary]autopilotTriggerCode {
	return map[autopilotStopBoundary]autopilotTriggerCode{
		autopilotStopBoundaryOwner:         autopilotTriggerRuntimeVerificationNeeded,
		autopilotStopBoundaryAuthority:     autopilotTriggerMissingAuthority,
		autopilotStopBoundaryPhysical:      autopilotTriggerProviderUnavailable,
		autopilotStopBoundaryUnrecoverable: autopilotTriggerDeterministicVerificationFailed,
	}
}

// TestAutopilotStopsOnlyAtTheFourDeclaredBoundaries proves the closed,
// total four-member boundary set: each declared boundary has a fixture that
// reaches it and the stop card names it; no fixture outside the four
// produces a stop boundary; and a stop card carries the exact same full-
// ceremony closeout (canonical slot set, ending with the cost-and-time
// block) a success card carries.
func TestAutopilotStopsOnlyAtTheFourDeclaredBoundaries(t *testing.T) {
	fixtures := autopilotStopBoundaryFixtures()

	for _, boundary := range autopilotStopBoundaries() {
		code, ok := fixtures[boundary]
		if !ok {
			t.Fatalf("no fixture registered for declared boundary %q", boundary)
		}
		t.Run(string(boundary), func(t *testing.T) {
			got, ok := autopilotStopBoundaryForTriggerCode(code)
			if !ok {
				t.Fatalf("trigger code %q did not resolve to any stop boundary", code)
			}
			if got != boundary {
				t.Fatalf("trigger code %q resolved to boundary %q, want %q", code, got, boundary)
			}
		})
	}

	t.Run("no other condition produces a stop boundary", func(t *testing.T) {
		// autopilotStopBoundaryOnlyCodes is the full, explicit set of codes
		// autopilotStopBoundaryForTriggerCode's own switch maps to a
		// boundary -- deliberately typed out in full here (not just one
		// fixture per boundary above) so this negative assertion checks
		// every code the real mapper actually claims, not merely the one
		// representative fixture each boundary's own subtest exercises.
		boundaryCodes := map[autopilotTriggerCode]bool{
			autopilotTriggerRuntimeVerificationNeeded:       true,
			autopilotTriggerVisualCheckpointNeeded:          true,
			autopilotTriggerBlockerCountIncreased:           true,
			autopilotTriggerBlockerEscalated:                true,
			autopilotTriggerMissingAuthority:                true,
			autopilotTriggerProviderUnavailable:             true,
			autopilotTriggerDeterministicVerificationFailed: true,
			autopilotTriggerAuditorScoreBelowFloor:          true,
			autopilotTriggerCriticalReviewFinding:           true,
		}
		for _, spec := range autopilotTriggerSpecs() {
			boundary, mapped := autopilotStopBoundaryForTriggerCode(spec.Code)
			isBoundaryCode := boundaryCodes[spec.Code]
			if mapped && !isBoundaryCode {
				t.Fatalf("trigger code %q unexpectedly resolved to stop boundary %q", spec.Code, boundary)
			}
			if !mapped && isBoundaryCode {
				t.Fatalf("trigger code %q should resolve to a stop boundary but did not", spec.Code)
			}
			if mapped && !validAutopilotStopBoundary(boundary) {
				t.Fatalf("trigger code %q resolved to undeclared boundary %q", spec.Code, boundary)
			}
		}
	})

	t.Run("a stop card carries the same full ceremony a success card carries", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s

		projection := LifecycleProjection{ProjectionRevision: "test-revision"}
		successCloseout, err := buildLifecycleCloseout(projection, "run", colony.OutcomeKindInProgress, colony.LifecycleStateEffectNone, LifecycleCloseoutDetails{
			WorkOutcome: colony.WorkOutcomeSuccess,
		})
		if err != nil {
			t.Fatalf("build success closeout: %v", err)
		}
		if len(successCloseout.Slots) == 0 {
			t.Fatalf("expected the success closeout to carry the full canonical slot set, got none")
		}

		for _, boundary := range autopilotStopBoundaries() {
			verdict := autopilotStopBoundaryWorkOutcome(boundary)
			stopCloseout, err := buildLifecycleCloseout(projection, "run", colony.OutcomeKindInProgress, colony.LifecycleStateEffectNone, LifecycleCloseoutDetails{
				WorkOutcome: verdict,
				Summary:     fmt.Sprintf("autopilot stopped at the %s boundary", boundary),
			})
			if err != nil {
				t.Fatalf("build stop closeout for boundary %s: %v", boundary, err)
			}
			if !reflect.DeepEqual(successCloseout.Slots, stopCloseout.Slots) {
				t.Fatalf("boundary %s slot set = %v, want the success card's slot set %v", boundary, stopCloseout.Slots, successCloseout.Slots)
			}

			result := map[string]interface{}{"completion_phase": 7}
			before := "rendered closeout body"
			after := appendLifecycleCloseoutSpendLine(before, stopCloseout, result)
			if after == before {
				t.Fatalf("boundary %s: expected the cost-and-time block to be appended to the stop card", boundary)
			}
		}
	})
}

// TestAutopilotNeverWaivesOrAnswersForTheOwner proves autopilot never
// produces a reviewer waiver and never records an owner-decision answer:
// none of the stop-boundary evaluation functions directly call any function
// whose name suggests waiving a forced reviewer or answering an owner
// decision, derived from the actual call graph rather than a hand-typed
// list of "known safe" names.
func TestAutopilotNeverWaivesOrAnswersForTheOwner(t *testing.T) {
	fset := token.NewFileSet()
	funcs := continueDecisionPackageFuncs(t, fset)

	suspectSubstrings := []string{"waive", "declinereview", "answerdecision", "recordownerdecisionanswer"}
	targets := []string{
		"autopilotStopBoundaryForTriggerCode",
		"autopilotStopBoundaryWorkOutcome",
		"selectAutopilotGoalTransition",
		"autopilotGoalLevelFactsFromLifecycle",
	}
	for _, name := range targets {
		fn, ok := funcs[name]
		if !ok {
			t.Fatalf("expected %s to be declared in the cmd package", name)
		}
		for callee := range continueDecisionDirectCalleeNames(fn) {
			lower := strings.ToLower(callee)
			for _, suspect := range suspectSubstrings {
				if strings.Contains(lower, suspect) {
					t.Fatalf("%s must never call an owner-waiver/answer function, found call to %s", name, callee)
				}
			}
		}
	}

	// Structural check: the boundary decision type itself carries no field
	// that could record a waiver or an owner-decision answer.
	typ := reflect.TypeOf(autopilotGoalTransitionDecision{})
	for i := 0; i < typ.NumField(); i++ {
		lower := strings.ToLower(typ.Field(i).Name)
		if strings.Contains(lower, "waive") || strings.Contains(lower, "answer") {
			t.Fatalf("autopilotGoalTransitionDecision must never carry an owner-waiver/answer field, found %s", typ.Field(i).Name)
		}
	}
}

// ---------------------------------------------------------------------------
// Task 3: the quick one-question path on the shared attempt model (CAP-029)
// ---------------------------------------------------------------------------

// fileChangingWorkerInvoker is a minimal codex.WorkerInvoker that reports a
// completed run which modified one file, so runQuickScout's deterministic-
// check branch is exercised.
type fileChangingWorkerInvoker struct{}

func (i *fileChangingWorkerInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *fileChangingWorkerInvoker) ValidateAgent(path string) error      { return nil }
func (i *fileChangingWorkerInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{
		WorkerName:    config.WorkerName,
		Caste:         config.Caste,
		TaskID:        config.TaskID,
		Status:        "completed",
		Summary:       "fixed the typo",
		FilesModified: []string{"README.md"},
		Duration:      5 * time.Millisecond,
	}, nil
}

// countingWorkerInvoker is a minimal codex.WorkerInvoker that counts how
// many times Invoke was called -- proportionality proof for CAP-029.
type countingWorkerInvoker struct{ count *int }

func (i *countingWorkerInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *countingWorkerInvoker) ValidateAgent(path string) error      { return nil }
func (i *countingWorkerInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	*i.count++
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Duration:   5 * time.Millisecond,
	}, nil
}

// TestQuickRunsOnTheSharedAttemptModel proves CAP-029: a quick request opens
// exactly one attempt, records its one dispatch, runs the deterministic
// checks only when it changed files, reports the no-change verdict rather
// than success when nothing changed, and a failure writes exactly one
// failure record carrying the attempt identifier.
func TestQuickRunsOnTheSharedAttemptModel(t *testing.T) {
	t.Run("a no-change request reports the no-change verdict", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		origInvoker := newQuickWorkerInvoker
		newQuickWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
		t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

		result, err := runQuickScout("what does this repository do?", 2*time.Second)
		if err != nil {
			t.Fatalf("runQuickScout: %v", err)
		}
		verdict, ok := result["work_outcome"].(colony.WorkOutcome)
		if !ok {
			t.Fatalf("expected work_outcome in the result, got %T", result["work_outcome"])
		}
		if verdict != colony.WorkOutcomeNoChange {
			t.Fatalf("verdict = %q, want %q", verdict, colony.WorkOutcomeNoChange)
		}
		attemptID, _ := result["attempt_id"].(string)
		if strings.TrimSpace(attemptID) == "" {
			t.Fatalf("expected a non-empty attempt_id, got result=%v", result)
		}
	})

	t.Run("a file-changing request runs the deterministic checks and reports a verdict", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		origInvoker := newQuickWorkerInvoker
		newQuickWorkerInvoker = func() codex.WorkerInvoker { return &fileChangingWorkerInvoker{} }
		t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

		origChecks := runQuickDeterministicChecks
		checkCalls := 0
		var checkedFiles []string
		runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
			checkCalls++
			checkedFiles = files
			return quickChecksPassed, []string{"go build ./...: passed", "go vet ./...: passed"}, nil
		}
		t.Cleanup(func() { runQuickDeterministicChecks = origChecks })

		result, err := runQuickScout("fix the typo in the readme", 2*time.Second)
		if err != nil {
			t.Fatalf("runQuickScout: %v", err)
		}
		if checkCalls != 1 {
			t.Fatalf("expected the deterministic checks to run exactly once, got %d", checkCalls)
		}
		if len(checkedFiles) == 0 {
			t.Fatalf("expected the changed files to be passed to the deterministic checks")
		}
		verdict, _ := result["work_outcome"].(colony.WorkOutcome)
		if verdict != colony.WorkOutcomeSuccess {
			t.Fatalf("verdict = %q, want %q", verdict, colony.WorkOutcomeSuccess)
		}
	})

	t.Run("a failed request produces exactly one failure record carrying the attempt identifier", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		wantErr := errors.New("scout invocation failed")
		origInvoker := newQuickWorkerInvoker
		newQuickWorkerInvoker = func() codex.WorkerInvoker { return &failingWorkerInvoker{err: wantErr} }
		t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

		if _, err := runQuickScout("does this fail?", 2*time.Second); err == nil {
			t.Fatal("expected runQuickScout to return the invoker's error")
		}

		mf, err := loadMiddenFile(store)
		if err != nil {
			t.Fatalf("load midden file: %v", err)
		}
		if len(mf.Entries) != 1 {
			t.Fatalf("expected exactly 1 midden entry, got %d: %+v", len(mf.Entries), mf.Entries)
		}
		foundAttemptTag := false
		for _, tag := range mf.Entries[0].Tags {
			if strings.HasPrefix(tag, "attempt:") {
				foundAttemptTag = true
			}
		}
		if !foundAttemptTag {
			t.Fatalf("expected the midden entry to carry an attempt: tag, got tags=%v", mf.Entries[0].Tags)
		}
	})
}

// TestQuickIsProportionate proves CAP-029's proportionality rule: a quick,
// one-question request with no named risk signal dispatches exactly one
// worker.
func TestQuickIsProportionate(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	invokeCount := 0
	origInvoker := newQuickWorkerInvoker
	newQuickWorkerInvoker = func() codex.WorkerInvoker { return &countingWorkerInvoker{count: &invokeCount} }
	t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

	if _, err := runQuickScout("what is this project?", 2*time.Second); err != nil {
		t.Fatalf("runQuickScout: %v", err)
	}
	if invokeCount != 1 {
		t.Fatalf("expected exactly one worker dispatch, got %d", invokeCount)
	}
}
