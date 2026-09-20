package cmd

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 02 (SYN-201-04) -- the continue lanes' accept/verify/advance
// spine. Before this plan, "should this phase advance" was decided inline,
// separately, in the direct lane (cmd/codex_continue.go), the plan-only
// lane (cmd/codex_continue_plan.go) and the external finalize lane
// (cmd/codex_continue_finalize.go). This file proves the collapse onto one
// body (cmd/codex_verify_advance.go): unit coverage of the decision body's
// own rules, an end-to-end proof the direct lane reaches advancement only
// through it, a parity proof across all three lanes' own construction
// paths, a reconciliation-acceptance proof, an attempt-bound-evidence proof,
// and an AST guard refusing a fourth independent implementation.

// continueDecisionFixtureAssessment builds a real codexContinueAssessment
// for a zero-task phase via the production assessCodexContinue function --
// never a hand-typed literal -- so a decision-body test's inputs are
// derived exactly the way every continue lane derives them.
func continueDecisionFixtureAssessment(t *testing.T, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, options codexContinueOptions, now time.Time) codexContinueAssessment {
	t.Helper()
	return assessCodexContinue(phase, manifest, verification, options, now)
}

func TestRunContinueAcceptVerifyAdvanceAdvancesWithDeterministicFloorAsPassSource(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	// The charter_compliance_executed gate reads COLONY_STATE.json; a zero-
	// value state (no Charter declared) is enough for it to pass, exactly as
	// a fresh colony with no governance rules declared would.
	createTestColonyState(t, dataDir, colony.ColonyState{})

	phase := colony.Phase{ID: 21, Name: "Decision body advance"}
	manifest := codexContinueManifest{Present: true}
	now := time.Now().UTC()
	verification := codexContinueVerificationReport{ChecksPassed: true, Claims: codexClaimVerification{Skipped: true}}
	assessment := continueDecisionFixtureAssessment(t, phase, manifest, verification, codexContinueOptions{}, now)
	if !assessment.Passed {
		t.Fatalf("fixture assessment did not pass: %+v", assessment)
	}
	gates := runCodexContinueGates(phase, manifest, verification, assessment, now, nil)
	if !gates.Passed {
		t.Fatalf("fixture gates did not pass: %+v", gates)
	}
	review := codexContinueReviewReport{Passed: true}

	decision := runContinueAcceptVerifyAdvance(phase, assessment, gates, &review, colony.ColonyState{})
	if !decision.Advances() {
		t.Fatalf("expected the decision to advance, got %+v", decision)
	}
	if decision.Phase != phase.ID {
		t.Fatalf("decision.Phase = %d, want %d", decision.Phase, phase.ID)
	}
	if decision.PassSource != continuePassSourceDeterministicFloor {
		t.Fatalf("decision.PassSource = %q, want %q", decision.PassSource, continuePassSourceDeterministicFloor)
	}
	if len(decision.BlockingReasons) != 0 {
		t.Fatalf("expected no blocking reasons, got %v", decision.BlockingReasons)
	}
}

func TestRunContinueAcceptVerifyAdvanceBlocksOnFailingFloorRegardlessOfPassingReviewer(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	_ = setupBuildFlowTest(t)

	phase := colony.Phase{ID: 22, Name: "Decision body blocks on floor"}
	manifest := codexContinueManifest{Present: true}
	now := time.Now().UTC()
	const failingCheck = "tests failed: exit status 1"
	verification := codexContinueVerificationReport{
		ChecksPassed:   false,
		BlockingIssues: []string{failingCheck},
	}
	assessment := continueDecisionFixtureAssessment(t, phase, manifest, verification, codexContinueOptions{}, now)
	gates := runCodexContinueGates(phase, manifest, verification, assessment, now, nil)
	if gates.Passed {
		t.Fatalf("fixture gates unexpectedly passed: %+v", gates)
	}
	// A passing reviewer verdict must not matter -- the deterministic floor
	// remains the only source of a pass.
	review := codexContinueReviewReport{Passed: true}

	decision := runContinueAcceptVerifyAdvance(phase, assessment, gates, &review, colony.ColonyState{})
	if decision.Advances() {
		t.Fatalf("expected block on a failing deterministic floor despite a passing reviewer, got %+v", decision)
	}
	if decision.PassSource != "" {
		t.Fatalf("a blocked decision should carry no pass source, got %q", decision.PassSource)
	}
	if !containsString(decision.BlockingReasons, failingCheck) {
		t.Fatalf("blocking reasons = %v, want the failing check named (%q)", decision.BlockingReasons, failingCheck)
	}
}

func TestRunContinueAcceptVerifyAdvanceBlocksOnPassingFloorWithBlockingReviewerFinding(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	createTestColonyState(t, dataDir, colony.ColonyState{})

	phase := colony.Phase{ID: 23, Name: "Decision body blocks on reviewer"}
	manifest := codexContinueManifest{Present: true}
	now := time.Now().UTC()
	verification := codexContinueVerificationReport{ChecksPassed: true, Claims: codexClaimVerification{Skipped: true}}
	assessment := continueDecisionFixtureAssessment(t, phase, manifest, verification, codexContinueOptions{}, now)
	gates := runCodexContinueGates(phase, manifest, verification, assessment, now, nil)
	if !gates.Passed {
		t.Fatalf("fixture gates did not pass: %+v", gates)
	}
	const blockingFinding = "auditor found a blocking security issue"
	review := codexContinueReviewReport{Passed: false, BlockingIssues: []string{blockingFinding}}

	decision := runContinueAcceptVerifyAdvance(phase, assessment, gates, &review, colony.ColonyState{})
	if decision.Advances() {
		t.Fatalf("expected block on a blocking reviewer finding despite a passing floor, got %+v", decision)
	}
	if !containsString(decision.BlockingReasons, blockingFinding) {
		t.Fatalf("blocking reasons = %v, want the reviewer finding named (%q)", decision.BlockingReasons, blockingFinding)
	}
}

func TestRunContinueAcceptVerifyAdvanceAppliesOneReconciliationDisposition(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	createTestColonyState(t, dataDir, colony.ColonyState{})

	taskID := "1.1"
	phase := colony.Phase{ID: 24, Name: "Decision body reconciliation", Tasks: []colony.Task{{ID: &taskID, Goal: "Do the reconciled thing"}}}
	manifest := codexContinueManifest{Present: true}
	now := time.Now().UTC()
	verification := codexContinueVerificationReport{ChecksPassed: true, Claims: codexClaimVerification{Skipped: true}}
	options := codexContinueOptions{ReconcileTaskIDs: []string{taskID}}
	assessment := continueDecisionFixtureAssessment(t, phase, manifest, verification, options, now)
	if !assessment.Passed {
		t.Fatalf("fixture assessment with a reconciled task did not pass: %+v", assessment)
	}
	gates := runCodexContinueGates(phase, manifest, verification, assessment, now, nil)
	if !gates.Passed {
		t.Fatalf("fixture gates did not pass: %+v", gates)
	}
	review := codexContinueReviewReport{Passed: true}

	decision := runContinueAcceptVerifyAdvance(phase, assessment, gates, &review, colony.ColonyState{})
	if !decision.Advances() {
		t.Fatalf("expected a reconciled task with a passing floor to advance, got %+v", decision)
	}
	if !decision.ReconciliationAccepted {
		t.Fatalf("expected ReconciliationAccepted=true, got %+v", decision)
	}
	if !containsString(decision.ReconciledTasks, taskID) {
		t.Fatalf("ReconciledTasks = %v, want %q", decision.ReconciledTasks, taskID)
	}
}

// TestDirectContinueLaneReachesAdvancementThroughTheSharedBody drives the
// real `aether continue` command against an isolated fixture root and
// asserts both the rendered verdict and the resulting durable phase status,
// after the direct lane (runCodexContinue) was rewired to reach its
// advancement verdict only through runContinueAcceptVerifyAdvance.
func TestDirectContinueLaneReachesAdvancementThroughTheSharedBody(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Prove the direct lane reaches advancement through the shared decision body"
	now := time.Now().UTC()
	taskOneID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Direct lane decision",
					Description: "Prove the shared accept/verify/advance body decides the direct lane",
					Status:      colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Verify the direct lane", Status: colony.TaskInProgress},
					},
				},
				{
					ID:     2,
					Name:   "Next direct phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Continue forward", Status: colony.TaskPending}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-51", Task: "Verify the direct lane", Status: "spawned", TaskID: taskOneID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-52", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Direct lane decision", goal, dispatches)

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true through the shared decision body, got %v", result)
	}
	if blocked, _ := result["blocked"].(bool); blocked {
		t.Fatalf("expected unblocked continue result, got %v", result)
	}
	if nextPhase := int(result["next_phase"].(float64)); nextPhase != 2 {
		t.Fatalf("next_phase = %d, want 2", nextPhase)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Fatalf("phase 1 status = %s, want completed", state.Plan.Phases[0].Status)
	}
	if state.CurrentPhase != 2 {
		t.Fatalf("current_phase = %d, want 2", state.CurrentPhase)
	}
}

// continueDecisionParityFixture returns the (phase, manifest, verification)
// facts a passing zero-task phase carries, shared by every lane-construction
// path TestThreeContinueLanesProduceOneDecision drives.
func continueDecisionParityFixture() (colony.Phase, codexContinueManifest, codexContinueVerificationReport) {
	phase := colony.Phase{ID: 25, Name: "Three lanes agree"}
	manifest := codexContinueManifest{Present: true}
	verification := codexContinueVerificationReport{ChecksPassed: true, Claims: codexClaimVerification{Skipped: true}}
	return phase, manifest, verification
}

// TestThreeContinueLanesProduceOneDecision proves that the direct lane's own
// gate-construction path (runCodexContinueGatesWithAutopilotBaseline), the
// plan-only lane's preview path (runCodexContinueGates, review nil), and the
// external finalize lane's path (runCodexContinueGates, a real review
// report) all fold into the identical continueAcceptVerifyAdvanceDecision
// for the same normalized inputs -- the "same normalized inputs, same
// decision value on every lane" must_have this plan carries.
func TestThreeContinueLanesProduceOneDecision(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	// The charter_compliance_executed gate reads COLONY_STATE.json; a zero-
	// value state (no Charter declared) is enough for it to pass, exactly as
	// a fresh colony with no governance rules declared would.
	createTestColonyState(t, dataDir, colony.ColonyState{})

	phase, manifest, verification := continueDecisionParityFixture()
	now := time.Now().UTC()
	assessment := continueDecisionFixtureAssessment(t, phase, manifest, verification, codexContinueOptions{}, now)
	review := codexContinueReviewReport{Passed: true}
	state := colony.ColonyState{}

	// Direct lane's own construction: runCodexContinueGatesWithAutopilotBaseline
	// with no autopilot baseline (the common case), decided once review exists.
	directGates := runCodexContinueGatesWithAutopilotBaseline(phase, manifest, verification, assessment, now, nil, nil)
	directDecision := runContinueAcceptVerifyAdvance(phase, assessment, directGates, &review, state)

	// Plan-only lane's own construction: runCodexContinueGates (no baseline),
	// review nil -- the advisory preview taken before any reviewer exists.
	planGates := runCodexContinueGates(phase, manifest, verification, assessment, now, nil)
	planDecision := runContinueAcceptVerifyAdvance(phase, assessment, planGates, nil, state)

	// External finalize lane's own construction: runCodexContinueGates (no
	// baseline -- the finalize lane never receives an autopilot baseline),
	// decided against the wrapper-collected review report.
	finalizeGates := runCodexContinueGates(phase, manifest, verification, assessment, now, nil)
	finalizeDecision := runContinueAcceptVerifyAdvance(phase, assessment, finalizeGates, &review, state)

	if !directDecision.Advances() {
		t.Fatalf("direct lane construction did not advance: %+v", directDecision)
	}
	if !finalizeDecision.Advances() {
		t.Fatalf("finalize lane construction did not advance: %+v", finalizeDecision)
	}
	if !reflect.DeepEqual(directDecision, finalizeDecision) {
		t.Fatalf("direct and finalize lane decisions diverged for identical inputs:\n direct=%+v\n finalize=%+v", directDecision, finalizeDecision)
	}
	if planDecision.Gates.Passed != directDecision.Gates.Passed {
		t.Fatalf("plan-only preview gate verdict diverged from the direct/finalize gate verdict: plan=%v direct=%v", planDecision.Gates.Passed, directDecision.Gates.Passed)
	}
	if !reflect.DeepEqual(planDecision.ReconciledTasks, directDecision.ReconciledTasks) || planDecision.PartialSuccess != directDecision.PartialSuccess {
		t.Fatalf("plan-only preview decision diverged from the direct decision on assessment-derived fields: plan=%+v direct=%+v", planDecision, directDecision)
	}
}

// TestFinalizeLaneAcceptsReconciliationLikeTheDirectLane closes the
// CONCERNS.md-named divergence: a supplied reconciliation record is accepted
// on the finalize lane's own gate/decision construction exactly as the
// direct lane's construction accepts it, for identical underlying facts.
func TestFinalizeLaneAcceptsReconciliationLikeTheDirectLane(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	createTestColonyState(t, dataDir, colony.ColonyState{})

	taskID := "1.1"
	phase := colony.Phase{ID: 26, Name: "Finalize reconciliation parity", Tasks: []colony.Task{{ID: &taskID, Goal: "Reconciled outside the pipeline"}}}
	manifest := codexContinueManifest{Present: true}
	now := time.Now().UTC()
	// The deterministic floor itself is what a manually reconciled task
	// still has to satisfy (FLOOR-03) -- pass it here so the ONLY thing
	// under test is reconciliation acceptance, not floor evaluation.
	verification := codexContinueVerificationReport{ChecksPassed: true, Claims: codexClaimVerification{Skipped: true}}
	options := codexContinueOptions{ReconcileTaskIDs: []string{taskID}}
	assessment := continueDecisionFixtureAssessment(t, phase, manifest, verification, options, now)
	review := codexContinueReviewReport{Passed: true}
	state := colony.ColonyState{}

	// Direct lane's own gate construction.
	directGates := runCodexContinueGatesWithAutopilotBaseline(phase, manifest, verification, assessment, now, nil, nil)
	directDecision := runContinueAcceptVerifyAdvance(phase, assessment, directGates, &review, state)

	// Finalize lane's own gate construction.
	finalizeGates := runCodexContinueGates(phase, manifest, verification, assessment, now, nil)
	finalizeDecision := runContinueAcceptVerifyAdvance(phase, assessment, finalizeGates, &review, state)

	if !directDecision.Advances() {
		t.Fatalf("direct lane refused a reconciliation record the deterministic floor already satisfied: %+v", directDecision)
	}
	if !finalizeDecision.Advances() {
		t.Fatalf("finalize lane refused a reconciliation record the direct lane accepted: %+v", finalizeDecision)
	}
	if !reflect.DeepEqual(directDecision, finalizeDecision) {
		t.Fatalf("finalize lane's reconciliation disposition diverged from the direct lane's:\n direct=%+v\n finalize=%+v", directDecision, finalizeDecision)
	}
	if !finalizeDecision.ReconciliationAccepted {
		t.Fatalf("expected the finalize lane to accept the supplied reconciliation record, got %+v", finalizeDecision)
	}
}

// TestConcurrentLanesKeepEvidenceAttemptBound proves CEC-06's attempt-bound
// evidence invariant across two attempts tracked against the same phase --
// one standing in for an interrupted run, one for a lane concurrently
// picking the phase back up: each attempt's durable evidence file is named
// by, and carries, only its own attempt identity, never the other's.
//
// deriveBuildAttempt (cmd/build_attempt.go) is the pure, production
// constructor both the canonical build-start transaction and this test use
// to build a buildAttemptRecord -- this test writes the two records directly
// through the store rather than through the full build-start transaction,
// because that transaction's own "one live build per phase" rule (correct,
// separate production behavior) refuses a second literal build-start for a
// phase whose first attempt is still open. This test's concern is narrower
// and structural: does ATTEMPT EVIDENCE STORAGE itself keep two attempts
// against one phase isolated from each other, which is exactly what
// deriveBuildAttempt's per-attempt-ID file path
// (build/phase-N/attempts/<id>.json) already guarantees.
func TestConcurrentLanesKeepEvidenceAttemptBound(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	phase := colony.Phase{ID: 1, Name: "Concurrent attempt evidence"}
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-61", Task: "Do the work", Status: "spawned"},
	}
	workspaceSHA := strings.Repeat("a", 64)

	interruptedRel, interruptedRecord, _, err := deriveBuildAttempt(buildAttemptDerivation{
		State: colony.ColonyState{}, Phase: phase, PhaseNumber: phase.ID,
		StartedAt: time.Now().UTC(), AttemptID: "attempt-interrupted-lane",
		RunID: "run-interrupted-lane", ProcessID: 4100, WorkspaceSHA256: workspaceSHA,
		ExecutionOwner: "runtime-worker-dispatch", Dispatches: dispatches,
		InitialStatus: buildAttemptPrepared, InitialDispatchMode: "direct",
	})
	if err != nil {
		t.Fatalf("derive interrupted attempt: %v", err)
	}
	if err := store.SaveJSON(interruptedRel, interruptedRecord); err != nil {
		t.Fatalf("write interrupted attempt evidence: %v", err)
	}

	concurrentRel, concurrentRecord, _, err := deriveBuildAttempt(buildAttemptDerivation{
		State: colony.ColonyState{}, Phase: phase, PhaseNumber: phase.ID,
		StartedAt: time.Now().UTC().Add(time.Minute), AttemptID: "attempt-concurrent-lane",
		RunID: "run-concurrent-lane", ProcessID: 4200, WorkspaceSHA256: workspaceSHA,
		ExecutionOwner: "runtime-worker-dispatch", Dispatches: dispatches,
		InitialStatus: buildAttemptPrepared, InitialDispatchMode: "direct",
	})
	if err != nil {
		t.Fatalf("derive concurrent attempt: %v", err)
	}
	if err := store.SaveJSON(concurrentRel, concurrentRecord); err != nil {
		t.Fatalf("write concurrent attempt evidence: %v", err)
	}

	if interruptedRel == concurrentRel {
		t.Fatalf("expected distinct evidence file paths, both resolved to %q", interruptedRel)
	}

	interruptedData, err := os.ReadFile(filepath.Join(store.BasePath(), interruptedRel))
	if err != nil {
		t.Fatalf("read interrupted attempt evidence file: %v", err)
	}
	concurrentData, err := os.ReadFile(filepath.Join(store.BasePath(), concurrentRel))
	if err != nil {
		t.Fatalf("read concurrent attempt evidence file: %v", err)
	}

	if strings.Contains(string(interruptedData), concurrentRecord.ID) {
		t.Fatalf("the interrupted attempt's evidence file references the concurrent attempt's ID -- evidence is not attempt-bound:\n%s", interruptedData)
	}
	if strings.Contains(string(concurrentData), interruptedRecord.ID) {
		t.Fatalf("the concurrent attempt's evidence file references the interrupted attempt's ID -- evidence is not attempt-bound:\n%s", concurrentData)
	}

	records := listBuildAttemptsForPhase(phase.ID)
	seen := map[string]bool{}
	for _, record := range records {
		seen[record.ID] = true
	}
	if !seen[interruptedRecord.ID] || !seen[concurrentRecord.ID] {
		t.Fatalf("expected both attempts to persist as distinct records, got %v", records)
	}
}

// --- Task 3: an AST guard refusing a fourth independent implementation ---

// continueDecisionSharedBodyName is the one function name every continue
// lane's advancement verdict must route through.
const continueDecisionSharedBodyName = "runContinueAcceptVerifyAdvance"

// continueDecisionGateEvaluatorNames are the shared gate evaluators
// (cmd/codex_continue.go) a lane calls before reaching the decision body.
// These two names, and continueDecisionSharedBodyName itself, are the ONLY
// hardcoded names in this guard -- they identify the shared INFRASTRUCTURE,
// not the set of lanes the guard protects. Which functions count as "lane
// entry points" is derived from the call graph below, never typed here.
var continueDecisionGateEvaluatorNames = map[string]bool{
	"runCodexContinueGates":                      true,
	"runCodexContinueGatesWithAutopilotBaseline": true,
}

// continueDecisionPackageFuncs parses every non-test .go file in the cmd
// package (the working directory `go test` runs in) plus any extra
// synthetic files, and indexes every top-level function declaration by
// name, mirroring the renderedFieldASTFuncs precedent
// (cmd/rendered_fields_invariant_test.go).
func continueDecisionPackageFuncs(t *testing.T, fset *token.FileSet, extra ...*ast.File) map[string]*ast.FuncDecl {
	t.Helper()
	pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse cmd package for the continue decision call graph: %v", err)
	}
	funcs := map[string]*ast.FuncDecl{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
					funcs[fn.Name.Name] = fn
				}
			}
		}
	}
	for _, file := range extra {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
				funcs[fn.Name.Name] = fn
			}
		}
	}
	return funcs
}

// continueDecisionDirectCalleeNames returns the plain identifier names every
// direct call inside fn's body invokes. Package-level function calls only --
// this guard's whole authority is which OTHER top-level function a lane
// entry point calls, not a deep type-resolved call graph.
func continueDecisionDirectCalleeNames(fn *ast.FuncDecl) map[string]bool {
	callees := map[string]bool{}
	if fn == nil || fn.Body == nil {
		return callees
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok {
			callees[ident.Name] = true
		}
		return true
	})
	return callees
}

// continueDecisionDirectCallers returns the sorted names of every function
// in funcs whose body directly calls target -- the call-graph-derived set
// this guard's positive assertion checks, rather than a hand-maintained
// list of lane names.
func continueDecisionDirectCallers(funcs map[string]*ast.FuncDecl, target string) []string {
	var callers []string
	for name, fn := range funcs {
		if name == target {
			continue
		}
		if continueDecisionDirectCalleeNames(fn)[target] {
			callers = append(callers, name)
		}
	}
	sort.Strings(callers)
	return callers
}

// continueDecisionOffenders returns every top-level function (other than the
// shared decision body itself and the gate evaluators it wraps) that calls a
// shared gate evaluator directly without also calling the shared decision
// body -- the structural signature of "a fourth implementation growing its
// own advancement logic."
func continueDecisionOffenders(funcs map[string]*ast.FuncDecl) []string {
	callersOfSharedBody := map[string]bool{}
	for _, name := range continueDecisionDirectCallers(funcs, continueDecisionSharedBodyName) {
		callersOfSharedBody[name] = true
	}
	var offenders []string
	for name, fn := range funcs {
		if name == continueDecisionSharedBodyName || continueDecisionGateEvaluatorNames[name] {
			continue
		}
		if callersOfSharedBody[name] {
			continue
		}
		callees := continueDecisionDirectCalleeNames(fn)
		callsGateEvaluator := false
		for evaluator := range continueDecisionGateEvaluatorNames {
			if callees[evaluator] {
				callsGateEvaluator = true
				break
			}
		}
		if callsGateEvaluator {
			offenders = append(offenders, name)
		}
	}
	sort.Strings(offenders)
	return offenders
}

// TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody is the structural
// guard: exactly one runContinueAcceptVerifyAdvance is declared, the direct,
// plan-only, and finalize lane entry points each call it (derived from the
// call graph, not a hardcoded list), no other top-level function evaluates
// continue gates independently of it, and a synthetic fourth implementation
// is refused by name.
func TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody(t *testing.T) {
	t.Run("exactly one declaration", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		if _, ok := funcs[continueDecisionSharedBodyName]; !ok {
			t.Fatalf("%s is not declared in the cmd package", continueDecisionSharedBodyName)
		}

		count := 0
		pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
			return !strings.HasSuffix(info.Name(), "_test.go")
		}, 0)
		if err != nil {
			t.Fatalf("parse cmd package: %v", err)
		}
		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				for _, decl := range file.Decls {
					if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == continueDecisionSharedBodyName {
						count++
					}
				}
			}
		}
		if count != 1 {
			t.Fatalf("%s is declared %d time(s), want exactly 1", continueDecisionSharedBodyName, count)
		}
	})

	t.Run("known lane entry points call it", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		callers := continueDecisionDirectCallers(funcs, continueDecisionSharedBodyName)
		if len(callers) < 3 {
			t.Fatalf("only %d function(s) call %s directly: %v -- expected the direct, plan-only, and finalize lane entry points", len(callers), continueDecisionSharedBodyName, callers)
		}
		for _, want := range []string{"runCodexContinue", "runCodexContinuePlanOnly", "runCodexContinueFinalize"} {
			if !containsString(callers, want) {
				t.Errorf("%s does not call %s -- this guard cannot protect a lane it cannot see reaching the shared body", want, continueDecisionSharedBodyName)
			}
		}
	})

	t.Run("no other function independently evaluates continue gates", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		offenders := continueDecisionOffenders(funcs)
		for _, name := range offenders {
			fn := funcs[name]
			pos := fset.Position(fn.Pos())
			t.Errorf("%s (%s:%d) evaluates continue gates directly but never calls %s -- a fourth advancement implementation is growing its own decision logic", name, pos.Filename, pos.Line, continueDecisionSharedBodyName)
		}
	})

	t.Run("guard fails a synthetic fourth implementation by name", func(t *testing.T) {
		fset := token.NewFileSet()
		src := `package cmd

func fakeFourthContinueLane() bool {
	report := runCodexContinueGatesWithAutopilotBaseline(nil, nil, nil, nil, nil, nil, nil)
	return report.Passed
}
`
		extra, err := parser.ParseFile(fset, "fixture_fourth_continue_lane.go", src, 0)
		if err != nil {
			t.Fatalf("parse synthetic fourth-lane fixture: %v", err)
		}
		funcs := continueDecisionPackageFuncs(t, fset, extra)
		offenders := continueDecisionOffenders(funcs)
		if !containsString(offenders, "fakeFourthContinueLane") {
			t.Fatalf("expected the guard to name the synthetic fourth lane, got offenders=%v", offenders)
		}
		for _, name := range offenders {
			if name == "fakeFourthContinueLane" {
				fn := funcs[name]
				pos := fset.Position(fn.Pos())
				if pos.Filename == "" || pos.Line == 0 {
					t.Fatalf("violation for %s carries no file/line position: %+v", name, pos)
				}
			}
		}
	})
}
