package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// blockedRecoveryFixture seeds a colony with one in-progress phase carrying
// two tasks, both dispatched by a real build and both reported "failed" --
// and breaks the repository's own build so the deterministic floor blocks
// continue regardless of task-level evidence. Task IDs are never typed
// twice: they are derived the same way assessCodexContinue derives them,
// via buildTaskID against the phase's own Tasks slice (index-based fallback
// since neither task carries an explicit ID) -- the same pattern
// twoTaskVerdictFixture (cmd/continue_task_verdict_test.go) already uses.
// This is the shared setup for every test in this file that drives the
// real blocked continue path end to end.
func blockedRecoveryFixture(t *testing.T) (dataDir, taskOneID, taskTwoID string) {
	t.Helper()
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir = setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() { this_will_not_compile }\n"), 0644); err != nil {
		t.Fatalf("failed to break workspace build: %v", err)
	}

	phaseTasks := []colony.Task{
		{Goal: "First unfinished task", Status: colony.TaskInProgress},
		{Goal: "Second unfinished task", Status: colony.TaskInProgress},
	}
	taskOneID = buildTaskID(phaseTasks[0], 0)
	taskTwoID = buildTaskID(phaseTasks[1], 1)

	goal := "Never a dead end fixture"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			// Set explicitly: without it, the state loader's own migration
			// re-derives an evidence/acceptance policy on read but this
			// fixture's raw on-disk JSON never carries one, and a phase
			// gaining recovery tasks across two continue calls in the same
			// test made a real, latent "partially populated" migration
			// refusal reachable on the second load that no single-call
			// fixture had ever hit before -- setting the policy up front
			// avoids exercising that unrelated migration path at all.
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			EvidencePolicy:   colony.PlanEvidenceNotRequired,
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Blocked recovery phase",
					Status: colony.PhaseInProgress,
					Tasks:  phaseTasks,
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-1", Task: "First unfinished task", Status: "failed", TaskID: taskOneID},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-2", Task: "Second unfinished task", Status: "failed", TaskID: taskTwoID},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Blocked recovery phase", goal, dispatches)
	return dataDir, taskOneID, taskTwoID
}

// runBlockedContinue runs `aether continue` once against the current test
// store and returns the parsed "result" object from the JSON envelope.
func runBlockedContinue(t *testing.T) map[string]interface{} {
	t.Helper()
	resetRootCmd(t)
	// A test that calls this helper more than once (idempotency proofs) must
	// not let the second run's JSON envelope be parsed alongside the
	// first's stale output -- parseLifecycleEnvelope finds the FIRST '{' in
	// the buffer, so a stale first-run envelope left in place would be
	// mistaken for the second run's result.
	if buf, ok := stdout.(*bytes.Buffer); ok {
		buf.Reset()
	}
	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}
	env := parseLifecycleEnvelope(t, stdout.(*bytes.Buffer).String())
	result, ok := env["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("result missing or wrong shape in envelope: %v", env)
	}
	return result
}

// pendingTaskGoals returns the Goal of every task in phase whose Status is
// pending.
func pendingTaskGoals(phase colony.Phase) []string {
	goals := make([]string, 0, len(phase.Tasks))
	for _, task := range phase.Tasks {
		if task.Status == colony.TaskPending {
			goals = append(goals, task.Goal)
		}
	}
	return goals
}

// TestFailedCheckAddsTheUnfinishedWorkAsTasks is 208-04 Task 1's primary
// proof: a blocked check with two tasks needing recovery writes two pending
// tasks back onto the phase, each naming the original task it recovers.
func TestFailedCheckAddsTheUnfinishedWorkAsTasks(t *testing.T) {
	dataDir, taskOneID, taskTwoID := blockedRecoveryFixture(t)
	_ = dataDir

	result := runBlockedContinue(t)
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Fatalf("expected blocked:true, got %v", result)
	}
	if added := intValue(result["recovery_tasks_added"]); added != 2 {
		t.Fatalf("result[recovery_tasks_added] = %d, want 2: %v", added, result)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload COLONY_STATE.json: %v", err)
	}
	if len(state.Plan.Phases) != 1 {
		t.Fatalf("expected 1 phase, got %d", len(state.Plan.Phases))
	}
	phase := state.Plan.Phases[0]
	if len(phase.Tasks) != 4 {
		t.Fatalf("expected 2 original + 2 recovery tasks = 4, got %d: %+v", len(phase.Tasks), phase.Tasks)
	}

	pending := pendingTaskGoals(phase)
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending recovery tasks, got %d: %v", len(pending), pending)
	}
	for _, taskID := range []string{taskOneID, taskTwoID} {
		found := false
		for _, goal := range pending {
			if strings.Contains(goal, taskID) {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a pending recovery task naming task %s, got goals: %v", taskID, pending)
		}
	}
}

// TestFailedCheckAddsTheSameTasksOnlyOnce proves repeating the same blocked
// check is idempotent: running it a second time adds nothing further.
func TestFailedCheckAddsTheSameTasksOnlyOnce(t *testing.T) {
	blockedRecoveryFixture(t)

	first := runBlockedContinue(t)
	if blocked, _ := first["blocked"].(bool); !blocked {
		t.Fatalf("expected first run blocked:true, got %v", first)
	}
	firstAdded := intValue(first["recovery_tasks_added"])
	if firstAdded == 0 {
		t.Fatalf("expected the first run to add recovery tasks, got 0")
	}

	var afterFirst colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &afterFirst); err != nil {
		t.Fatalf("reload COLONY_STATE.json after first run: %v", err)
	}
	countAfterFirst := len(afterFirst.Plan.Phases[0].Tasks)

	second := runBlockedContinue(t)
	if blocked, _ := second["blocked"].(bool); !blocked {
		t.Fatalf("expected second run blocked:true, got %v", second)
	}
	secondAdded := intValue(second["recovery_tasks_added"])

	var afterSecond colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &afterSecond); err != nil {
		t.Fatalf("reload COLONY_STATE.json after second run: %v", err)
	}
	countAfterSecond := len(afterSecond.Plan.Phases[0].Tasks)
	if secondAdded != 0 {
		t.Fatalf("expected the second identical blocked run to add 0 recovery tasks, got %d", secondAdded)
	}

	if countAfterSecond != countAfterFirst {
		t.Fatalf("task count changed on repeated blocked check: after first=%d, after second=%d", countAfterFirst, countAfterSecond)
	}
}

// TestFailedCheckNeverAdvancesOrVerifies proves that writing recovery tasks
// back onto the phase never advances the phase or marks it verified or
// complete -- carrying on means the work list moves, never the phase.
func TestFailedCheckNeverAdvancesOrVerifies(t *testing.T) {
	dataDir, _, _ := blockedRecoveryFixture(t)
	_ = dataDir

	var before colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &before); err != nil {
		t.Fatalf("load COLONY_STATE.json before continue: %v", err)
	}

	result := runBlockedContinue(t)
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false on a blocked check, got %v", result)
	}
	if completed, ok := result["completed"].(bool); ok && completed {
		t.Fatalf("expected completed to be absent or false, got true: %v", result)
	}

	var after colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &after); err != nil {
		t.Fatalf("reload COLONY_STATE.json after continue: %v", err)
	}
	if after.CurrentPhase != before.CurrentPhase {
		t.Fatalf("CurrentPhase changed on a blocked check: before=%d after=%d", before.CurrentPhase, after.CurrentPhase)
	}
	if after.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase Status changed on a blocked check: got %q, want %q", after.Plan.Phases[0].Status, colony.PhaseInProgress)
	}
}

// TestBlockedCheckAlwaysNamesANextCommand table-drives every branch
// continueNextCommandForBlocked has and fails, naming the branch, if any of
// them ever returns a string that is empty after trimming. This is the
// direct proof for 208-04's must_have: "The result's next is never the
// empty string on the blocked path."
func TestBlockedCheckAlwaysNamesANextCommand(t *testing.T) {
	cases := []struct {
		name       string
		assessment codexContinueAssessment
		blockers   []string
		options    codexContinueOptions
	}{
		{
			name: "verification timeout",
			assessment: codexContinueAssessment{
				Recovery: codexContinueRecoveryPlan{ReverifyCommand: "aether continue"},
			},
			blockers: []string{"verification command timed out after 15m; increase with --verification-timeout or narrow the repo verification command"},
			options:  codexContinueOptions{VerificationTimeout: 15 * time.Minute},
		},
		{
			name: "worker timeout",
			assessment: codexContinueAssessment{
				Recovery: codexContinueRecoveryPlan{ReverifyCommand: "aether continue"},
			},
			blockers: []string{"worker timed out after 10m; increase with --worker-timeout"},
			options:  codexContinueOptions{WorkerTimeout: 10 * time.Minute},
		},
		{
			name: "watcher failure",
			assessment: codexContinueAssessment{
				Recovery: codexContinueRecoveryPlan{ReverifyCommand: "aether continue"},
			},
			blockers: []string{"watcher failed to complete its review"},
			options:  codexContinueOptions{},
		},
		{
			name: "options match the last invocation (D-10 fallback)",
			assessment: codexContinueAssessment{
				Recovery: codexContinueRecoveryPlan{RedispatchCommand: "aether continue"},
			},
			blockers: []string{"verification command timed out"},
			options:  codexContinueOptions{},
		},
		{
			name:       "previously-empty fall-through, nothing recorded at all",
			assessment: codexContinueAssessment{Recovery: codexContinueRecoveryPlan{ReverifyCommand: "aether continue"}},
			blockers:   []string{"a generic operational issue with no recognised shape"},
			options:    codexContinueOptions{},
		},
		{
			name: "previously-empty fall-through, reconcile tasks pending",
			assessment: codexContinueAssessment{
				Recovery: codexContinueRecoveryPlan{
					ReverifyCommand: "aether continue",
					ReconcileTasks:  []string{"1.1"},
				},
			},
			blockers: []string{"a generic operational issue with no recognised shape"},
			options:  codexContinueOptions{},
		},
		{
			name: "previously-empty fall-through, redispatch tasks pending",
			assessment: codexContinueAssessment{
				Recovery: codexContinueRecoveryPlan{
					ReverifyCommand: "aether continue",
					RedispatchTasks: []string{"1.2"},
				},
			},
			blockers: []string{"a generic operational issue with no recognised shape"},
			options:  codexContinueOptions{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			dataDir := setupBuildFlowTest(t)
			root := filepath.Dir(filepath.Dir(dataDir))
			withTestWorkspace(t, root)
			withWorkingDir(t, root)

			got := continueNextCommandForBlocked(tc.assessment, tc.blockers, tc.options, 1)
			if strings.TrimSpace(got) == "" {
				t.Fatalf("branch %q returned an empty next command (blockers=%v)", tc.name, tc.blockers)
			}
		})
	}
}

// TestRecoveryTasksCarryNoEvidenceRequirements proves every task
// recoveryTasksForBlockedContinue produces carries no SuccessCriteria and
// no EvidenceRequirements, and that the flattened criterion evidence
// requirement list is byte-identical before and after appending them --
// the guarantee that keeps criterionRequirementsEqual (cmd/criterion_evidence.go)
// from ever failing because of a recovery task rather than the build
// manifest.
func TestRecoveryTasksCarryNoEvidenceRequirements(t *testing.T) {
	phase := colony.Phase{
		ID:     7,
		Status: colony.PhaseInProgress,
		Tasks: []colony.Task{
			{
				Goal:                 "Original task with a bound criterion",
				SuccessCriteria:      []string{"the export button downloads a real file"},
				EvidenceRequirements: []colony.CriterionEvidenceRequirement{{Criterion: "the export button downloads a real file", Artifacts: []string{"export.csv"}}},
			},
		},
	}
	before := flattenPhaseCriterionEvidenceRequirements(phase)

	assessment := codexContinueAssessment{
		Tasks: []codexContinueTaskAssessment{
			{TaskID: "task-1", Summary: "worker evidence is incomplete"},
			{TaskID: "task-2", Summary: "worker evidence is incomplete"},
		},
		Recovery: codexContinueRecoveryPlan{
			RedispatchTasks: []string{"task-1", "task-2"},
		},
	}

	recovered := recoveryTasksForBlockedContinue(phase, assessment)
	if len(recovered) != 2 {
		t.Fatalf("expected 2 recovery tasks, got %d: %+v", len(recovered), recovered)
	}
	for _, task := range recovered {
		if len(task.SuccessCriteria) != 0 {
			t.Fatalf("recovery task %q carries SuccessCriteria, want none: %+v", task.Goal, task)
		}
		if len(task.EvidenceRequirements) != 0 {
			t.Fatalf("recovery task %q carries EvidenceRequirements, want none: %+v", task.Goal, task)
		}
		if task.Status != colony.TaskPending {
			t.Fatalf("recovery task %q Status = %q, want pending", task.Goal, task.Status)
		}
	}

	added := appendRecoveryTasks(&phase, recovered)
	if added != 2 {
		t.Fatalf("appendRecoveryTasks added = %d, want 2", added)
	}
	after := flattenPhaseCriterionEvidenceRequirements(phase)
	if !criterionRequirementsEqual(before, after) {
		t.Fatalf("flattened criterion evidence requirements changed after appending recovery tasks:\nbefore=%+v\nafter=%+v", before, after)
	}

	// Mutation proof (recorded verbatim in the SUMMARY): temporarily giving
	// a recovery task a non-empty EvidenceRequirements would make this
	// comparison fail, since the flattened list length would grow by one.
	tampered := append([]colony.Task{}, phase.Tasks...)
	tampered[len(tampered)-1].EvidenceRequirements = []colony.CriterionEvidenceRequirement{{Criterion: "invented criterion"}}
	tamperedPhase := phase
	tamperedPhase.Tasks = tampered
	tamperedAfter := flattenPhaseCriterionEvidenceRequirements(tamperedPhase)
	if criterionRequirementsEqual(before, tamperedAfter) {
		t.Fatalf("expected a tampered recovery task with evidence requirements to change the flattened list, but it did not -- the comparison itself cannot fail")
	}
}

// TestAppendRecoveryTasksIsIdempotentByGoalPrefix is a focused unit proof
// beneath the end-to-end TestFailedCheckAddsTheSameTasksOnlyOnce: appending
// the identical candidate task list twice adds it only once.
func TestAppendRecoveryTasksIsIdempotentByGoalPrefix(t *testing.T) {
	phase := colony.Phase{ID: 3, Tasks: []colony.Task{{Goal: "Original task"}}}
	assessment := codexContinueAssessment{
		Recovery: codexContinueRecoveryPlan{RedispatchTasks: []string{"task-1"}},
	}

	candidates := recoveryTasksForBlockedContinue(phase, assessment)
	firstAdded := appendRecoveryTasks(&phase, candidates)
	if firstAdded != 1 {
		t.Fatalf("first append added = %d, want 1", firstAdded)
	}

	candidatesAgain := recoveryTasksForBlockedContinue(phase, assessment)
	secondAdded := appendRecoveryTasks(&phase, candidatesAgain)
	if secondAdded != 0 {
		t.Fatalf("second append (identical candidates) added = %d, want 0", secondAdded)
	}
	if len(phase.Tasks) != 2 {
		t.Fatalf("expected 1 original + 1 recovery task = 2, got %d: %+v", len(phase.Tasks), phase.Tasks)
	}
}

// TestRecoveryTasksNeverSetSemanticID is the regression guard for a real
// bug this plan's own idempotency proof found while implementing it: a
// recovery task carrying a non-empty colony.Task.SemanticID trips
// validatePlanningState's "legacy_unbound plan cannot contain current
// candidate or acceptance bindings" refusal (cmd/planning_state.go) the
// moment acceptance_policy is "legacy_unbound" -- the common historical
// shape -- breaking `aether continue`/`aether status` outright on the very
// next load. Recovery tasks must identify themselves only via the
// deterministic Goal prefix (recoveryTaskGoalPrefix), never SemanticID.
func TestRecoveryTasksNeverSetSemanticID(t *testing.T) {
	phase := colony.Phase{ID: 9, Tasks: []colony.Task{{Goal: "Original task"}}}
	assessment := codexContinueAssessment{
		Recovery: codexContinueRecoveryPlan{
			ReconcileTasks:  []string{"task-1"},
			RedispatchTasks: []string{"task-2"},
		},
	}

	candidates := recoveryTasksForBlockedContinue(phase, assessment)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d: %+v", len(candidates), candidates)
	}
	for _, task := range candidates {
		if task.SemanticID != "" {
			t.Fatalf("recovery task %q carries a non-empty SemanticID %q -- this breaks legacy_unbound plans (see doc comment)", task.Goal, task.SemanticID)
		}
	}

	appendRecoveryTasks(&phase, candidates)

	// The exact scenario that surfaced the bug: a plan under the common
	// legacy_unbound acceptance policy must still validate cleanly after
	// recovery tasks are appended.
	state := colony.ColonyState{
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			Phases:           []colony.Phase{phase},
		},
	}
	if err := validatePlanningState(state); err != nil {
		t.Fatalf("a legacy_unbound plan carrying recovery tasks must still validate, got: %v", err)
	}
}

// TestHumanTaskWithRecoveryLikeGoalIsNeverExcluded is CR-02's own regression
// guard (208-REVIEW.md). Before this fix, a task was recognised as a
// system-generated recovery task purely by whether its free-text Goal
// began with the literal substring "Finish task " -- a perfectly plausible
// real task Goal ("Finish task queue implementation") collided with that
// shape and was silently dropped from tasksExcludingRecovery and
// phaseTaskIDSet, the exact comparisons validateCurrentPlanningState and
// the build-manifest task-set check run to catch a false record of what
// was actually built. Recognition is now structural (colony.Task.Origin,
// set only by appendRecoveryTasks), never derived from Goal text -- this
// test fails without that fix and passes with it.
func TestHumanTaskWithRecoveryLikeGoalIsNeverExcluded(t *testing.T) {
	humanTask := colony.Task{
		Goal:   "Finish task queue implementation",
		Status: colony.TaskPending,
	}
	genuineRecoveryTask := colony.Task{
		Goal:   recoveryTaskGoalPrefix("task-1", "redispatch"),
		Status: colony.TaskPending,
		Origin: colony.TaskOriginRecovery,
	}
	phase := colony.Phase{
		ID:    11,
		Tasks: []colony.Task{humanTask, genuineRecoveryTask},
	}

	// The human task's colliding Goal must NOT make it look like a
	// recovery task by structure.
	if taskIsRecoveryTask(humanTask) {
		t.Fatalf("a human-authored task with a Goal that merely looks like a recovery task's is being classified as one: %+v", humanTask)
	}
	if !taskIsRecoveryTask(genuineRecoveryTask) {
		t.Fatalf("a real, Origin-marked recovery task is not being classified as one: %+v", genuineRecoveryTask)
	}

	filtered := tasksExcludingRecovery(phase.Tasks)
	if len(filtered) != 1 || filtered[0].Goal != humanTask.Goal {
		t.Fatalf("tasksExcludingRecovery excluded the human-authored task with a colliding Goal; got %+v", filtered)
	}

	ids := phaseTaskIDSet(phase)
	humanTaskID := buildTaskID(humanTask, 0)
	found := false
	for _, id := range ids {
		if id == humanTaskID {
			found = true
		}
	}
	if !found {
		t.Fatalf("phaseTaskIDSet excluded the human-authored task with a colliding Goal (id %q); got %v", humanTaskID, ids)
	}
}

// Phase 210 blocker 17 (2026-10-02, Finish the Track deck): a phase check
// stopped and wrote its own "Finish task 2.3 (needs reconciling)" job onto the
// phase, then the owner rebuilt just task 2.3. The fresh build plan copied that
// recovery job in as an unnumbered "task-4", while the saved project record
// rightly leaves recovery jobs out, so saving the rebuild refused forever:
// "build manifest task set does not match COLONY_STATE (manifest: 2.1, 2.2,
// 2.3, task-4; state: 2.1, 2.2, 2.3)". A build plan must leave recovery jobs
// out too, as the saved record does.
func TestRebuildingOneTaskAfterAStoppedCheckStillMatchesTheRecord(t *testing.T) {
	_, taskOneID, _ := blockedRecoveryFixture(t)

	first := runBlockedContinue(t)
	if intValue(first["recovery_tasks_added"]) == 0 {
		t.Fatalf("fixture: the stopped check was expected to add recovery jobs, got %v", first)
	}

	resetRootCmd(t)
	stdout.(*bytes.Buffer).Reset()
	rootCmd.SetArgs([]string{"build", "1", "--task", taskOneID, "--plan-only"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build --task --plan-only returned error: %v", err)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	recoveryJobs := 0
	for _, task := range state.Plan.Phases[0].Tasks {
		if taskIsRecoveryTask(task) {
			recoveryJobs++
		}
	}
	if recoveryJobs == 0 {
		t.Fatalf("fixture: the phase should still carry the stopped check's recovery jobs: %+v", state.Plan.Phases[0].Tasks)
	}
	manifest := loadCodexContinueManifest(1)
	if !manifest.Present {
		t.Fatalf("the targeted rebuild wrote no build plan: %s", stdout.(*bytes.Buffer).String())
	}
	if err := validateBuildManifestTaskSetForPhase(manifest, state.Plan.Phases[0], true); err != nil {
		t.Fatalf("a rebuild after a stopped check must still match the saved record: %v", err)
	}
}

// The owner's own project already holds a build plan written before the fix
// above, with the recovery job copied in. Saving that rebuild must work too,
// or the project stays stuck until the whole phase is rebuilt. The comparison
// sets aside exactly the plan entries that are the phase's own recovery jobs,
// recognised by the structural recovery marker on the phase, never by wording.
func TestABuildPlanWrittenBeforeTheFixStillMatchesTheRecord(t *testing.T) {
	_, _, _ = blockedRecoveryFixture(t)
	if intValue(runBlockedContinue(t)["recovery_tasks_added"]) == 0 {
		t.Fatal("fixture: the stopped check was expected to add recovery jobs")
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	phase := state.Plan.Phases[0]

	// The plan exactly as the old code wrote it: every phase job, recovery
	// jobs included, each under the ID the runtime derives for it.
	var oldPlan []codexBuildTaskPlan
	for idx, task := range phase.Tasks {
		oldPlan = append(oldPlan, codexBuildTaskPlan{ID: buildTaskID(task, idx), Goal: task.Goal, Status: task.Status})
	}
	if len(oldPlan) == len(tasksExcludingRecovery(phase.Tasks)) {
		t.Fatalf("fixture: the old-style plan should include recovery jobs: %+v", oldPlan)
	}
	manifest := codexContinueManifest{Present: true, Data: codexBuildManifest{Tasks: oldPlan}}
	if err := validateBuildManifestTaskSetForPhase(manifest, phase, true); err != nil {
		t.Fatalf("an old build plan carrying the phase's own recovery jobs must still match: %v", err)
	}

	// A plan entry that is NOT one of the phase's recovery jobs still refuses.
	extra := codexContinueManifest{Present: true, Data: codexBuildManifest{Tasks: append(append([]codexBuildTaskPlan(nil), oldPlan...),
		codexBuildTaskPlan{ID: "task-99", Goal: "work nobody planned"})}}
	if err := validateBuildManifestTaskSetForPhase(extra, phase, true); err == nil {
		t.Fatal("a plan entry the phase does not hold must still be refused")
	}
}
