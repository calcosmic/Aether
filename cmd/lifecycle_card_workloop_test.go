package cmd

// Phase 197 plan 04 -- the two commands the owner runs most, and the closeout
// they share.
//
// Building a phase and checking it are where a project actually gets spent, and
// between them they carried seven separate hand-written closings: a full
// dispatch, a part-finished build, a plan-only run, a finalize, an ordinary
// advance, a plan-only check, and a blocked one. Each said what to type next in
// its own words, and several of them named a different command from the one the
// project's own state implied.
//
// These tests require every one of those to end with the shared card, WITHOUT
// losing what makes each situation specific: a part-finished build must still
// name which work was done and which was not, and a blocked check must still
// list the gates that failed and how to clear them.

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// workLoopSurface is one closing, however it is reached: some are whole command
// runs, some are a renderer driven over a result the runtime really produced.
type workLoopSurface struct {
	name string
	// visual returns what the owner sees, and the answer that screen should
	// have been built from.
	visual func(t *testing.T) lifecycleRun
	// envelope returns the machine-readable result the same situation produces.
	// Nil where the situation has no envelope of its own.
	envelope func(t *testing.T) lifecycleRun
	// keep are the situation-specific lines that must survive above the card.
	keep []string
}

// buildableWorkLoopState is a two-phase project with the first phase ready.
func buildableWorkLoopState(t *testing.T, dataDir string) {
	t.Helper()
	writeLifecycleState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         fixtureGoal("Ship the billing rewrite"),
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{Phases: []colony.Phase{
			fixturePhase(1, "Foundations", colony.PhaseReady),
			fixturePhase(2, "Billing engine", colony.PhasePending),
		}},
	})
}

// builtWorkLoopState is the same project with the first phase built and waiting
// to be checked, with the build record the check reads seeded beside it. Both
// are written the way the runtime writes them; a check run without the build
// record behind it takes a different path entirely and would prove nothing
// about the ordinary one.
func builtWorkLoopState(t *testing.T, dataDir string) {
	t.Helper()
	goal := "Ship the billing rewrite"
	now := time.Now().UTC()
	taskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Milestone:      "Open Chambers",
		Plan: colony.Plan{Phases: []colony.Phase{
			{
				ID:     1,
				Name:   "Foundations",
				Status: colony.PhaseInProgress,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Lay the foundations", Status: colony.TaskInProgress}},
			},
			{
				ID:     2,
				Name:   "Billing engine",
				Status: colony.PhasePending,
				Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Build the billing engine", Status: colony.TaskPending}},
			},
		}},
	})
	seedContinueBuildPacket(t, dataDir, 1, "Foundations", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-41", Task: "Lay the foundations", Status: "spawned", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-42", Task: "Independent verification", Status: "spawned"},
	})
	if err := store.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{}}); err != nil {
		t.Fatalf("seed instincts.json: %v", err)
	}
	if err := store.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed learning-observations.json: %v", err)
	}
}

func workLoopSurfaces() []workLoopSurface {
	commandSurface := func(name, command string, args []string, prepare func(*testing.T, string), keep ...string) workLoopSurface {
		c := lifecycleSurfaceCase{name: name, args: args, command: command, prepare: prepare, keep: keep}
		return workLoopSurface{
			name:     name,
			visual:   func(t *testing.T) lifecycleRun { return runLifecycleSurface(t, c, false) },
			envelope: func(t *testing.T) lifecycleRun { return runLifecycleSurface(t, c, true) },
			keep:     keep,
		}
	}

	return []workLoopSurface{
		commandSurface("building a phase", "build",
			[]string{"build", "1"}, buildableWorkLoopState,
			"── Tasks ──", "── Dispatch ──"),
		commandSurface("building a phase, plan only", "build",
			[]string{"build", "1", "--plan-only"}, buildableWorkLoopState,
			"No state was changed and no workers were spawned."),
		commandSurface("checking the work", "continue",
			[]string{"continue"}, builtWorkLoopState,
			"── Verification ──"),
		commandSurface("the shared closeout", "build",
			[]string{"closeout", "build"}, builtWorkLoopState,
			"State: "),
		{
			name:     "building a phase that only partly finished",
			visual:   partlyFinishedBuildRun,
			envelope: partlyFinishedBuildRun,
			keep:     []string{"Finished and kept", "Still to do", "NOT ready to be checked"},
		},
		{
			name:     "a check that is blocked",
			visual:   blockedCheckRun,
			envelope: blockedCheckRun,
			keep:     []string{"Blocking issues", "Way forward"},
		},
		{
			// Recording work that helpers did outside the program.
			name: "recording work done outside the program",
			visual: rendererSurface(func(state colony.ColonyState) string {
				return renderBuildFinalizeVisual(state, state.Plan.Phases[0], nil)
			}, "build"),
			keep: []string{"External Task worker results recorded."},
		},
		{
			// The check prepared but not run: the manifest-only variant.
			name: "checking the work, plan only",
			visual: rendererSurface(func(state colony.ColonyState) string {
				return renderContinuePlanOnlyVisual(state, state.Plan.Phases[0], nil, colony.VerificationDepthStandard)
			}, "continue"),
			keep: []string{"No state was changed and no review workers were spawned."},
		},
	}
}

// rendererSurface drives one renderer over a saved project, for the variants
// that are a screen rather than a whole command run. The project is written
// through the runtime's own store first, so the answer the renderer resolves is
// the one production would resolve.
func rendererSurface(render func(colony.ColonyState) string, command string) func(*testing.T) lifecycleRun {
	return func(t *testing.T) lifecycleRun {
		t.Helper()
		newNextActionFixtureStore(t)
		t.Setenv("AETHER_PLATFORM", "codex")
		state := normalizedFixtureState(t, colony.ColonyState{
			Version:      "3.0",
			Goal:         fixtureGoal("Ship the billing rewrite"),
			State:        colony.StateREADY,
			CurrentPhase: 1,
			Milestone:    "Open Chambers",
			Plan: colony.Plan{Phases: []colony.Phase{
				fixturePhase(1, "Foundations", colony.PhaseReady),
				fixturePhase(2, "Billing engine", colony.PhasePending),
			}},
		})
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatalf("write the fixture project: %v", err)
		}
		run := lifecycleRun{}
		run.answer = lifecycleNextActionForState(state, command, "", "")
		run.card = stripANSI(renderNextActionCardForPlatform(run.answer, "codex"))
		run.visual = stripANSI(render(state))
		return run
	}
}

// partlyFinishedBuildRun drives the real finalizer over a job that finished
// four of its six tasks, then renders the screen that situation shows. One run
// produces both halves, so the card and the envelope cannot be compared against
// two different situations.
func partlyFinishedBuildRun(t *testing.T) lifecycleRun {
	t.Helper()
	t.Setenv("AETHER_PLATFORM", "codex")
	root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "Partly finished build")

	proven := ids[:4]
	receipts := make([]codex.TaskReceipt, 0, len(proven))
	touched := make([]string, 0, len(proven))
	for _, id := range proven {
		receipts = append(receipts, receiptForTask(t, root, id))
		touched = append(touched, taskFileName(id))
	}
	results := []codexExternalBuildWorkerResult{{
		Stage: chain.Stage, Wave: chain.Wave, ExecutionWave: normalizedDispatchWave(chain),
		Caste: chain.Caste, Name: chain.Name, TaskID: chain.TaskID,
		Status:        "failed",
		Summary:       "stopped after finishing four of six steps",
		FilesModified: touched,
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "fail",
			CommandsRun:        []string{"go test ./..."},
		},
		TaskReceipts: receipts,
	}}

	result, state, phase, _, err := runCodexBuildFinalize(root, 1,
		codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}, false)
	if err != nil {
		t.Fatalf("finalizing a genuine part-finished build: %v", err)
	}
	recovery, ok := partialBuildRecoveryFromResult(result)
	if !ok {
		t.Fatal("a part-finished build handed the owner no typed durable recovery projection")
	}

	run := lifecycleRun{envelope: result}
	run.answer, ok = nextActionFromResult(result)
	if !ok || run.answer.Projection == nil {
		t.Fatal("a part-finished build did not carry the resolver-issued action used by its envelope")
	}
	if run.answer.Command != recovery.RedispatchCommand {
		t.Fatalf("resolver-issued partial action = %q, want durable recovery command %q", run.answer.Command, recovery.RedispatchCommand)
	}
	run.card = stripANSI(renderNextActionCardForPlatform(run.answer, "codex"))
	run.visual = stripANSI(renderBuildPartialCreditResultVisual(state, phase, result))
	return run
}

// blockedCheckRun drives a real blocked check: the saved build record and the
// project's own task list disagree, which is a situation the runtime refuses to
// advance past and hands back an exact command for. One run produces the screen
// and the machine-readable result together.
func blockedCheckRun(t *testing.T) lifecycleRun {
	t.Helper()
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Setenv("AETHER_PLATFORM", "codex")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Ship the billing rewrite"
	now := time.Now().UTC()
	taskOneID := "1.1"
	taskTwoID := "1.2"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{Phases: []colony.Phase{
			{
				ID:     1,
				Name:   "Foundations",
				Status: colony.PhaseInProgress,
				Tasks: []colony.Task{
					{ID: &taskOneID, Goal: "Ship the first task", Status: colony.TaskInProgress},
					{ID: &taskTwoID, Goal: "Ship the second task", Status: colony.TaskInProgress, DependsOn: []string{taskOneID}},
				},
			},
			{
				ID:     2,
				Name:   "Billing engine",
				Status: colony.PhasePending,
				Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Must wait", Status: colony.TaskPending}},
			},
		}},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Foundations", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-1", Task: "Ship the first task", Status: "completed", TaskID: taskOneID},
		{Stage: "wave", Wave: 2, Caste: "builder", Name: "Forge-2", Task: "Ship the second task", Status: "completed", TaskID: taskTwoID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-3", Task: "Independent verification before advancement", Status: "completed"},
	})
	var manifest codexBuildManifest
	manifestPath := filepath.ToSlash(filepath.Join("build", "phase-1", "manifest.json"))
	if err := store.LoadJSON(manifestPath, &manifest); err != nil {
		t.Fatalf("load the saved build record: %v", err)
	}
	manifest.Tasks = []codexBuildTaskPlan{{ID: taskOneID, Goal: "Ship the first task", Status: colony.TaskCompleted}}
	if err := store.SaveJSON(manifestPath, manifest); err != nil {
		t.Fatalf("write the drifted build record: %v", err)
	}

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	stdout = &bytes.Buffer{}
	result, state, phase, _, _, _, err := runCodexContinue(root, codexContinueOptions{})
	if err != nil {
		t.Fatalf("the blocked check returned an error: %v", err)
	}
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("this fixture was supposed to produce a blocked check; got %v", result["blocked"])
	}

	run := lifecycleRun{envelope: result}
	answer, ok := nextActionFromResult(result)
	if !ok {
		t.Fatal("a blocked check folds no resolved answer into its result; the card and the envelope cannot come from one decision")
	}
	run.answer = answer
	run.card = stripANSI(renderNextActionCardForPlatform(answer, "codex"))
	run.visual = stripANSI(renderContinueBlockedVisual(state, phase, result, reviewDepthFromResult(result)))
	return run
}

// TestWorkLoopCardsComeFromTheResolver is the screen half for building,
// checking and the shared closeout.
func TestWorkLoopCardsComeFromTheResolver(t *testing.T) {
	for _, surface := range workLoopSurfaces() {
		t.Run(surface.name, func(t *testing.T) {
			run := surface.visual(t)
			if !strings.Contains(run.visual, run.card) {
				t.Errorf("%s does not end with the shared card.\n--- the card the resolver produced ---\n%s\n--- what was printed ---\n%s",
					surface.name, run.card, run.visual)
			}
			for _, keep := range surface.keep {
				if !strings.Contains(run.visual, keep) {
					t.Errorf("%s lost %q from above the closing block -- only the what-next block was supposed to change",
						surface.name, keep)
				}
			}
		})
	}
}

// TestWorkLoopEnvelopesMatchTheirCards is the machine-readable half.
func TestWorkLoopEnvelopesMatchTheirCards(t *testing.T) {
	for _, surface := range workLoopSurfaces() {
		if surface.envelope == nil {
			continue
		}
		t.Run(surface.name, func(t *testing.T) {
			run := surface.envelope(t)
			assertEnvelopeMatchesCard(t, surface.name, run)
		})
	}
}

// TestRedispatchOverrideReachesBothTheCardAndTheEnvelope is the one that would
// catch the override being applied to only one of the two.
//
// A part-finished build knows something the saved project cannot work out for
// itself: the exact command that picks up only the work that was never done.
// That knowledge has to reach the decision, not be written over its answer
// afterwards -- otherwise the screen recommends one thing and the wrapper
// executes another.
func TestRedispatchOverrideReachesBothTheCardAndTheEnvelope(t *testing.T) {
	run := partlyFinishedBuildRun(t)

	recovery, ok := partialBuildRecoveryFromResult(run.envelope)
	if !ok {
		t.Fatal("this fixture was supposed to produce a typed command for the unfinished work")
	}
	if run.answer.Command != recovery.RedispatchCommand {
		t.Errorf("the card recommends %q; the run's own command for the unfinished work is %q",
			run.answer.Command, recovery.RedispatchCommand)
	}
	if envelopeCommand := stringValue(run.envelope[nextActionCommandKey]); envelopeCommand != recovery.RedispatchCommand {
		t.Errorf("the machine-readable answer says %q; the run's own command for the unfinished work is %q",
			envelopeCommand, recovery.RedispatchCommand)
	}
	if !strings.Contains(run.visual, recovery.RedispatchCommand) {
		t.Errorf("the screen never names %q at all:\n%s", recovery.RedispatchCommand, run.visual)
	}
	if legacy := strings.TrimSpace(stringValue(run.envelope["next"])); legacy != recovery.RedispatchCommand {
		t.Errorf("the older `next` key says %q while the card says %q -- the two answers separated again",
			legacy, recovery.RedispatchCommand)
	}
	if legacy := strings.TrimSpace(stringValue(run.envelope["recovery_command"])); legacy != recovery.RedispatchCommand {
		t.Errorf("legacy recovery_command %q is not a projection of typed partial_recovery %q",
			legacy, recovery.RedispatchCommand)
	}
	assertEnvelopeMatchesCard(t, "partial redispatch", run)
}

// TestBlockedCheckStillExplainsItself -- the card says what to type. What went
// wrong, and how to clear it, is the command reporting on its own run and must
// still be there, above the card.
func TestBlockedCheckStillExplainsItself(t *testing.T) {
	run := blockedCheckRun(t)
	cardIndex := strings.Index(run.visual, run.card)
	if cardIndex < 0 {
		t.Fatalf("the blocked check does not end with the shared card:\n%s", run.visual)
	}
	above := run.visual[:cardIndex]
	for _, want := range []string{"Blocking issues", "Way forward"} {
		if !strings.Contains(above, want) {
			t.Errorf("a blocked check no longer shows %q above the closing block:\n%s", want, above)
		}
	}
}

// TestMigratedWorkLoopClosingsSayItOnce -- the card carries the "is it safe to
// close this chat" verdict now, so no migrated screen may also carry the old
// sentence.
func TestMigratedWorkLoopClosingsSayItOnce(t *testing.T) {
	for _, surface := range workLoopSurfaces() {
		t.Run(surface.name, func(t *testing.T) {
			run := surface.visual(t)
			if strings.Contains(run.visual, "clear your context") {
				t.Errorf("%s still prints the old context sentence; the card carries that verdict now", surface.name)
			}
			if count := countCloseThisChatSentences(run.visual); count > 1 {
				t.Errorf("%s tells the owner whether to close the chat %d times", surface.name, count)
			}
		})
	}
}
