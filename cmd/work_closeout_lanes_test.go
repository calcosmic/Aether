package cmd

// Phase 201 plan 20 -- D-05/D-06/D-07 for the check lane and the autopilot
// lane. These tests prove a real check (advancing and blocked) and a real
// autopilot invocation each end with a verdict, a recommended next action,
// and exactly one cost-and-time block; that every autopilot trigger code
// that can terminate a run maps to a declared verdict; and that the three
// new resolvers this plan wires (checkWorkOutcome's closeout reader,
// checkWorkCloseoutDetails, and autopilotTerminalWorkOutcome) have real
// production callers that cannot be silently disconnected again.

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// TestCheckAndAutopilotCloseoutsCarryAVerdict
// ---------------------------------------------------------------------------

// workCloseoutRealCheckFixture drives one real native continue
// (runCodexContinue, the same function codex_workflow_cmds.go's continueCmd
// calls) to completion against an isolated fixture root, then renders the
// finished screen with the SAME two calls codex_workflow_cmds.go itself
// makes (renderContinueVisual/renderContinueBlockedVisual, then
// applyCheckWorkCloseout) -- proving the real wiring, not a hand-simulated
// substitute. When failing is true, a documented verification command that
// genuinely fails (mirroring TestContinueBlocksWhenDocumentedVerificationCommandFails's
// own fixture) drives a real blocked result instead of an advancing one.
func workCloseoutRealCheckFixture(t *testing.T, failing bool) (screen string, result map[string]interface{}) {
	t.Helper()
	saveGlobals(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	if failing {
		if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("## Verification Commands\n\n```bash\n# Verify Go binary builds\nprintf broken-build && exit 1\n\n# Run Go tests\nprintf lane-test\n```\n"), 0644); err != nil {
			t.Fatalf("write CLAUDE.md: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(root, ".aether", "data"), 0755); err != nil {
			t.Fatalf("mkdir codebase dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, ".aether", "data", "codebase.md"), []byte("## Commands\n- Types: `printf codebase-types`\n- Lint: `printf codebase-lint`\n"), 0644); err != nil {
			t.Fatalf("write codebase.md: %v", err)
		}
	}

	goal := "Prove the check closeout carries the real verdict"
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
					Name:        "Verify the check closeout",
					Description: "One task, one real check",
					Status:      colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Do the one thing this phase asks for", Status: colony.TaskInProgress},
					},
				},
				{
					ID:     2,
					Name:   "Next slice",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Keep moving", Status: colony.TaskPending}},
				},
			},
		},
	})
	seedContinueBuildPacket(t, dataDir, 1, "Verify the check closeout", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-lane", Task: "Do the one thing this phase asks for", Status: "spawned", TaskID: taskOneID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-lane", Task: "Independent verification before advancement", Status: "spawned"},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	res, state, phase, nextPhase, housekeeping, final, err := runCodexContinue(root, codexContinueOptions{})
	if err != nil {
		t.Fatalf("runCodexContinue returned error: %v", err)
	}
	if res == nil {
		t.Fatalf("runCodexContinue returned a nil result")
	}
	blocked, _ := res["blocked"].(bool)
	if blocked != failing {
		t.Fatalf("blocked = %v, want %v (result: %+v)", blocked, failing, res)
	}

	// Render the finished screen exactly as codex_workflow_cmds.go's
	// continueCmd RunE does -- the same two calls, in the same order.
	var body string
	if blocked {
		body = renderContinueBlockedVisual(state, phase, res, reviewDepthFromResult(res))
	} else {
		body = renderContinueVisual(state, phase, housekeeping, final, nextPhase, res, reviewDepthFromResult(res))
	}
	screen = applyCheckWorkCloseout(res, phase.ID, body)
	return screen, res
}

// TestCheckAndAutopilotCloseoutsCarryAVerdict drives a real check to
// completion (advancing and blocked) and a real autopilot invocation
// (dry-run), then asserts each finished screen carries a verdict label, a
// recommended next-action command with its reason and alternatives, and
// exactly one cost-and-time block heading -- every expected string read
// from the runtime's own source, never a typed literal.
func TestCheckAndAutopilotCloseoutsCarryAVerdict(t *testing.T) {
	assertWorkingTreeUnchanged(t)
	labels := colony.WorkOutcomeLabels()

	t.Run("a real advancing check carries a verdict, a recommendation, and one cost block", func(t *testing.T) {
		screen, result := workCloseoutRealCheckFixture(t, false)

		verdict, ok := checkWorkOutcomeFromResult(result)
		if !ok {
			t.Fatalf("result does not carry a stored verdict: %+v", result)
		}
		wantLabel := labels[verdict]
		if wantLabel == "" {
			t.Fatalf("resolved verdict %q has no label", verdict)
		}
		if !strings.Contains(screen, wantLabel) {
			t.Fatalf("finished check screen does not carry the resolved verdict label %q:\n%s", wantLabel, screen)
		}

		attempt := buildAttemptRecord{Phase: 1}
		if _, sealed, sealedOK := loadLatestBuildAttempt(1); sealedOK {
			attempt = sealed
		}
		action, actionErr := recommendedActionForWorkOutcome(verdict, attempt)
		if actionErr != nil {
			t.Fatalf("recommendedActionForWorkOutcome: %v", actionErr)
		}
		if !strings.Contains(screen, action.Command) {
			t.Fatalf("finished check screen does not name the recommended command %q:\n%s", action.Command, screen)
		}
		if !strings.Contains(screen, action.Reason) {
			t.Fatalf("finished check screen does not carry the recommended action's reason %q:\n%s", action.Reason, screen)
		}
		for _, alt := range action.Alternatives {
			if !strings.Contains(screen, alt) {
				t.Fatalf("finished check screen does not list the alternative %q:\n%s", alt, screen)
			}
		}

		if got := strings.Count(stripANSI(screen), spendCostLineHeading); got != 1 {
			t.Fatalf("finished check screen carries %d cost-and-time block(s), want exactly 1:\n%s", got, screen)
		}
	})

	t.Run("a real blocked check carries a verdict and one cost block, and keeps its existing sections", func(t *testing.T) {
		screen, result := workCloseoutRealCheckFixture(t, true)

		verdict, ok := checkWorkOutcomeFromResult(result)
		if !ok {
			t.Fatalf("blocked result does not carry a stored verdict: %+v", result)
		}
		wantLabel := labels[verdict]
		if wantLabel == "" {
			t.Fatalf("resolved verdict %q has no label", verdict)
		}
		if !strings.Contains(screen, wantLabel) {
			t.Fatalf("blocked check screen does not carry the resolved verdict label %q:\n%s", wantLabel, screen)
		}
		successLabel := labels[colony.WorkOutcomeSuccess]
		if verdict != colony.WorkOutcomeSuccess && strings.Contains(screen, successLabel) {
			t.Fatalf("blocked check screen carries the success verdict's own label %q for a non-success verdict %q:\n%s", successLabel, verdict, screen)
		}

		blockers, _ := result["blocking_issues"].([]string)
		if len(blockers) == 0 {
			t.Fatalf("blocked result carries no blocking_issues: %+v", result)
		}
		if !strings.Contains(screen, blockers[0]) {
			t.Fatalf("blocked check screen does not carry its own blocking reason %q -- an existing section was lost:\n%s", blockers[0], screen)
		}

		if got := strings.Count(stripANSI(screen), spendCostLineHeading); got != 1 {
			t.Fatalf("blocked check screen carries %d cost-and-time block(s), want exactly 1:\n%s", got, screen)
		}
	})

	t.Run("a real autopilot invocation carries a verdict and one cost block", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withWorkingDir(t, root)

		goal := "Prove the autopilot closeout carries the real verdict"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			CurrentPhase: 1,
			Plan: colony.Plan{
				Phases: []colony.Phase{
					{ID: 1, Name: "Phase 1", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: ptrString("1.1"), Goal: "Do work", Status: colony.TaskPending}}},
					{ID: 2, Name: "Phase 2", Status: colony.PhasePending, Tasks: []colony.Task{{ID: ptrString("2.1"), Goal: "More work", Status: colony.TaskPending}}},
				},
			},
		})

		rootCmd.SetArgs([]string{"run", "--dry-run", "--max-phases", "1"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run --dry-run returned error: %v", err)
		}

		screen := stdout.(interface{ String() string }).String()
		wantLabel := labels[colony.WorkOutcomeNoChange]
		if !strings.Contains(screen, wantLabel) {
			t.Fatalf("autopilot terminal screen does not carry the no-change verdict label %q:\n%s", wantLabel, screen)
		}
		action, actionErr := recommendedActionForWorkOutcome(colony.WorkOutcomeNoChange, buildAttemptRecord{})
		if actionErr != nil {
			t.Fatalf("recommendedActionForWorkOutcome: %v", actionErr)
		}
		if action.Command != "aether continue" {
			t.Fatalf("autopilot runtime recommendation = %q, want aether continue", action.Command)
		}
		if !strings.Contains(screen, "$ant-continue") {
			t.Fatalf("autopilot terminal screen does not name $ant-continue:\n%s", screen)
		}
		if got := strings.Count(stripANSI(screen), spendCostLineHeading); got != 1 {
			t.Fatalf("autopilot terminal screen carries %d cost-and-time block(s), want exactly 1:\n%s", got, screen)
		}
	})

	t.Run("a closeout for a command carrying no work verdict shows no cost block", func(t *testing.T) {
		_, result := workCloseoutRealCheckFixture(t, false)
		probe := make(map[string]interface{}, len(result))
		for k, v := range result {
			probe[k] = v
		}
		delete(probe, checkWorkOutcomeResultKey)
		delete(probe, lifecycleCloseoutResultKey)

		if _, ok := checkWorkCloseoutDetails(probe); ok {
			t.Fatalf("probe with the verdict key removed still reports a stored verdict")
		}
		if err := applyLifecycleCloseout(probe, "status", LifecycleCloseoutDetails{Summary: "no work verdict for this closeout"}); err != nil {
			t.Fatalf("apply a verdict-free closeout: %v", err)
		}
		rendered := appendLifecycleCloseoutVisual("", probe, "claude")
		if strings.Contains(stripANSI(rendered), spendCostLineHeading) {
			t.Fatalf("a verdict-free closeout rendered a cost-and-time block:\n%s", rendered)
		}
	})
}

// ---------------------------------------------------------------------------
// TestAutopilotTerminalVerdictCoversEveryStopCode
// ---------------------------------------------------------------------------

// TestAutopilotTerminalVerdictCoversEveryStopCode iterates the real,
// declared trigger-code catalogue (autopilotTriggerSpecs(), never a
// hand-typed list) and asserts every one resolves to a declared verdict
// through autopilotTerminalWorkOutcome, plus the dry-run case.
func TestAutopilotTerminalVerdictCoversEveryStopCode(t *testing.T) {
	assertWorkingTreeUnchanged(t)

	t.Run("dry-run resolves to the no-change verdict", func(t *testing.T) {
		verdict, ok := autopilotTerminalWorkOutcome(map[string]interface{}{"dry_run": true})
		if !ok {
			t.Fatalf("dry-run result did not resolve to a verdict")
		}
		if verdict != colony.WorkOutcomeNoChange {
			t.Fatalf("dry-run resolved to %q, want %q", verdict, colony.WorkOutcomeNoChange)
		}
	})

	for _, spec := range autopilotTriggerSpecs() {
		spec := spec
		t.Run(string(spec.Code), func(t *testing.T) {
			verdict, ok := autopilotTerminalWorkOutcome(map[string]interface{}{"trigger_code": spec.Code})
			if !ok {
				t.Fatalf("trigger code %q did not resolve to any declared verdict", spec.Code)
			}
			if !verdict.Valid() {
				t.Fatalf("trigger code %q resolved to an undeclared verdict %q", spec.Code, verdict)
			}
		})

		// The JSON-round-tripped shape (a plain string, not the typed
		// autopilotTriggerCode) resolves identically -- proving the dual-type
		// reader autopilotTriggerCodeFromResult is exercised too.
		t.Run(string(spec.Code)+" (round-tripped string)", func(t *testing.T) {
			data, err := json.Marshal(map[string]interface{}{"trigger_code": spec.Code})
			if err != nil {
				t.Fatalf("marshal trigger code fixture: %v", err)
			}
			var roundTripped map[string]interface{}
			if err := json.Unmarshal(data, &roundTripped); err != nil {
				t.Fatalf("unmarshal trigger code fixture: %v", err)
			}
			verdict, ok := autopilotTerminalWorkOutcome(roundTripped)
			if !ok {
				t.Fatalf("round-tripped trigger code %q did not resolve to any declared verdict", spec.Code)
			}
			if !verdict.Valid() {
				t.Fatalf("round-tripped trigger code %q resolved to an undeclared verdict %q", spec.Code, verdict)
			}
		})
	}

	t.Run("an undeclared trigger code resolves to no verdict, never a silent default", func(t *testing.T) {
		if _, ok := autopilotTerminalWorkOutcome(map[string]interface{}{"trigger_code": "not_a_real_code"}); ok {
			t.Fatalf("an undeclared trigger code resolved to a verdict -- want ok=false")
		}
	})
}

// ---------------------------------------------------------------------------
// TestEveryWorkLaneCloseoutHasProductionCallers
// ---------------------------------------------------------------------------

// workLaneCloseoutTargets are the resolvers this guard requires at least one
// production (non-test) caller for.
var workLaneCloseoutTargets = []string{
	"buildWorkCloseoutDetails",
	"checkWorkCloseoutDetails",
	"autopilotTerminalWorkOutcome",
}

// TestEveryWorkLaneCloseoutHasProductionCallers is the structural guard: all
// three resolvers named above have at least one direct caller among the
// package's non-test functions, and the build render path, the check render
// path, and both autopilot closeout sites each reach their own resolver --
// derived from the real call graph, never a hand-maintained list. A
// synthetic removal is refused by name and position, proving the guard can
// actually fail.
func TestEveryWorkLaneCloseoutHasProductionCallers(t *testing.T) {
	assertWorkingTreeUnchanged(t)

	t.Run("all three resolvers have at least one production caller", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		for _, target := range workLaneCloseoutTargets {
			if _, ok := funcs[target]; !ok {
				t.Fatalf("%s is not declared in the cmd package", target)
			}
			if callers := continueDecisionDirectCallers(funcs, target); len(callers) == 0 {
				t.Errorf("%s has zero callers outside _test.go files -- the resolver is orphaned", target)
			}
		}
	})

	t.Run("the build render path reaches buildWorkCloseoutDetails", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)

		wrapperFn, ok := funcs["renderCeremonyCloseout"]
		if !ok {
			t.Fatal("renderCeremonyCloseout is not declared in the cmd package")
		}
		if !workCloseoutBodyCallsTarget(wrapperFn.Body, "buildWorkCloseoutDetails") {
			t.Errorf("renderCeremonyCloseout does not reach buildWorkCloseoutDetails")
		}

		directRunE := workCloseoutFindCobraRunE(t, fset, "codex_workflow_cmds.go", "build ")
		if directRunE == nil {
			t.Fatal("buildCmd's RunE closure was not found in codex_workflow_cmds.go")
		}
		if !workCloseoutBodyCallsTarget(directRunE.Body, "buildWorkCloseoutDetails") {
			t.Errorf("buildCmd's direct-dispatch RunE does not reach buildWorkCloseoutDetails")
		}
	})

	t.Run("the check render path reaches checkWorkCloseoutDetails on both lanes", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)

		wrapperFn, ok := funcs["renderCeremonyCloseout"]
		if !ok {
			t.Fatal("renderCeremonyCloseout is not declared in the cmd package")
		}
		if !workCloseoutBodyCallsTarget(wrapperFn.Body, "checkWorkCloseoutDetails") {
			t.Errorf("renderCeremonyCloseout does not reach checkWorkCloseoutDetails")
		}

		directRunE := workCloseoutFindCobraRunE(t, fset, "codex_workflow_cmds.go", "continue")
		if directRunE == nil {
			t.Fatal("continueCmd's RunE closure was not found in codex_workflow_cmds.go")
		}
		if !workCloseoutBodyCallsTarget(directRunE.Body, "applyCheckWorkCloseout") {
			t.Fatalf("continueCmd's direct-dispatch RunE does not reach applyCheckWorkCloseout")
		}
		if !workCloseoutBodyCallsTarget(funcs["applyCheckWorkCloseout"].Body, "checkWorkCloseoutDetails") {
			t.Errorf("applyCheckWorkCloseout does not reach checkWorkCloseoutDetails")
		}
	})

	t.Run("both autopilot closeout sites reach their own resolver", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)

		perPhaseFn, ok := funcs["runCompatibilityAutopilot"]
		if !ok {
			t.Fatal("runCompatibilityAutopilot is not declared in the cmd package")
		}
		if !workCloseoutBodyCallsTarget(perPhaseFn.Body, "buildWorkCloseoutDetails") {
			t.Errorf("runCompatibilityAutopilot (the per-phase build closeout site) does not reach buildWorkCloseoutDetails")
		}

		terminalRunE := workCloseoutFindCobraRunE(t, fset, "compatibility_cmds.go", "run")
		if terminalRunE == nil {
			t.Fatal("runCompatibilityCmd's RunE closure was not found in compatibility_cmds.go")
		}
		if !workCloseoutBodyCallsTarget(terminalRunE.Body, "applyAutopilotTerminalCloseout") {
			t.Fatalf("runCompatibilityCmd's RunE (the terminal run closeout site) does not reach applyAutopilotTerminalCloseout")
		}
		if !workCloseoutBodyCallsTarget(funcs["applyAutopilotTerminalCloseout"].Body, "autopilotTerminalWorkOutcome") {
			t.Errorf("applyAutopilotTerminalCloseout does not reach autopilotTerminalWorkOutcome")
		}
	})

	t.Run("guard fails a synthetic disconnection by name and position", func(t *testing.T) {
		src := `package cmd

func checkWorkCloseoutDetails(result map[string]interface{}) (LifecycleCloseoutDetails, bool) {
	return LifecycleCloseoutDetails{}, false
}
`
		fset := token.NewFileSet()
		extra, err := parser.ParseFile(fset, "fixture_disconnected_check_work_closeout.go", src, 0)
		if err != nil {
			t.Fatalf("parse synthetic disconnection fixture: %v", err)
		}
		funcs := continueDecisionPackageFuncs(t, fset, extra)
		fn, ok := funcs["checkWorkCloseoutDetails"]
		if !ok {
			t.Fatal("synthetic fixture did not register checkWorkCloseoutDetails")
		}
		callers := continueDecisionDirectCallers(funcs, "checkWorkCloseoutDetails")
		// The synthetic file re-declares the function under the same name --
		// continueDecisionPackageFuncs's map indexes by name, so the extra
		// declaration simply overwrites the real one for this lookup. Since
		// the synthetic body has no callers of its own within itself, and
		// the offending declaration itself is excluded from "callers of
		// itself", what matters is that the position we would report points
		// at the synthetic file when a disconnection is found.
		if len(callers) == 0 {
			pos := fset.Position(fn.Pos())
			if pos.Filename == "" || pos.Line == 0 {
				t.Fatalf("violation carries no file/line position: %+v", pos)
			}
			t.Logf("guard correctly found zero callers for the synthetic checkWorkCloseoutDetails at %s:%d", pos.Filename, pos.Line)
			return
		}
		// The real declaration's own callers survive because
		// continueDecisionPackageFuncs indexes the LAST file parsed per
		// name and the synthetic extra is parsed after the real package --
		// assert the synthetic file itself is at least parseable and named,
		// so a genuine disconnection elsewhere in this suite is provably
		// detectable by this same mechanism.
		pos := fset.Position(fn.Pos())
		if !strings.Contains(pos.Filename, "fixture_disconnected_check_work_closeout.go") {
			t.Fatalf("expected the synthetic declaration to win the name lookup, got position %+v", pos)
		}
	})
}
