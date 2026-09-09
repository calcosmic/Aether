package cmd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// parseRedispatchTaskArgs turns a surfaced recovery command back into the task
// IDs it would actually redispatch, exactly the way the CLI's own flag parsing
// would. A command carrying no --task at all redispatches the whole phase, so
// it returns nil and the caller treats that as "everything".
func parseRedispatchTaskArgs(t *testing.T, command string) (phaseArg string, taskIDs []string, force bool) {
	t.Helper()
	fields := strings.Fields(command)
	if len(fields) < 3 || fields[0] != "aether" || fields[1] != "build" {
		t.Fatalf("recovery command %q is not an `aether build <phase>` invocation", command)
	}
	phaseArg = fields[2]
	for i := 3; i < len(fields); i++ {
		switch fields[i] {
		case "--force":
			force = true
		case "--task":
			if i+1 >= len(fields) {
				t.Fatalf("recovery command %q ends with a dangling --task", command)
			}
			taskIDs = append(taskIDs, fields[i+1])
			i++
		}
	}
	return phaseArg, taskIDs, force
}

// TestPartialRetryCommandNeverRedispatchesCreditedWork is the permanent
// regression lock for the 195 review's fourth critical finding (CR-04).
//
// When a grouped job finishes some of its tasks and then dies, the runtime
// plans a recovery job containing ONLY the unfinished tasks and writes it into
// a new attempt. Four separate shipped surfaces promise the owner that the
// command they are handed redispatches only that unfinished work and never
// asks a new worker to redo proven work. The command actually surfaced was
// `aether build <N> --force`, which re-plans the phase from its task list with
// no filter at all -- so every credited task was dispatched again.
//
// This test does not assert what the planner RETURNS. It takes the command the
// owner is literally handed, parses its arguments the way the CLI does, feeds
// those arguments through the real build planner, and fails if any credited
// task appears in the resulting spawn list.
func TestPartialRetryCommandNeverRedispatchesCreditedWork(t *testing.T) {
	tasks, ids := sixChainedTasks()
	credited := ids[:4]
	unfinished := ids[4:]
	for i := range tasks {
		if i < len(credited) {
			tasks[i].Status = colony.TaskCompleted
		}
	}
	phase := colony.Phase{ID: 1, Name: "Six-task grouped job", Tasks: tasks}
	goal := "Redispatch only unfinished grouped work"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			EvidencePolicy:   colony.PlanEvidenceNotRequired,
			Phases:           []colony.Phase{phase},
		},
	}

	startedAt := time.Now().UTC()
	dispatch := codexBuildDispatch{
		Name:             "Mason-1",
		Caste:            "builder",
		Stage:            "wave",
		TaskID:           ids[0],
		CoveredTaskIDs:   append([]string{}, ids...),
		JobName:          "automatic-six-step",
		Status:           "failed",
		CompletedTaskIDs: append([]string{}, credited...),
	}
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		Variant: buildStartDirect, GeneratedAt: startedAt,
		SelectedTasks: ids, Dispatches: []codexBuildDispatch{dispatch},
		ExecutionOwner: "go-runtime", DispatchMode: "direct", MakeLatest: testBuildStartBool(true),
		PrepareRoot: func(root string) {
			createTestColonyState(t, filepath.Join(root, ".aether", "data"), state)
		},
	})
	state, phase, parentRel := fixture.State, fixture.Phase, fixture.AttemptPath
	var parent buildAttemptRecord
	if err := store.LoadJSON(parentRel, &parent); err != nil {
		t.Fatalf("load parent attempt: %v", err)
	}

	outcome, err := reconcilePartialBuildRetry(state, 1, phase, parent.ID, startedAt.Add(time.Second), []codexBuildDispatch{dispatch})
	if err != nil {
		t.Fatalf("reconcilePartialBuildRetry: %v", err)
	}
	if outcome == nil {
		t.Fatal("expected a partial-retry outcome for a four-of-six grouped job")
	}

	phaseArg, taskArgs, _ := parseRedispatchTaskArgs(t, outcome.RedispatchCommand)
	if phaseArg != "1" {
		t.Fatalf("recovery command targets phase %q, want phase 1: %q", phaseArg, outcome.RedispatchCommand)
	}
	if len(taskArgs) == 0 {
		t.Fatalf("recovery command %q names no specific task, so running it re-plans the whole phase and redoes every credited task", outcome.RedispatchCommand)
	}

	// The command's own arguments, run through the real build planner.
	planned, _, err := plannedBuildDispatchesWithJobProposals(
		phase, state, taskArgs, colony.VerificationDepthStandard, nil, "", nil, nil,
	)
	if err != nil {
		t.Fatalf("planning the recovery command's own arguments failed: %v", err)
	}

	plannedTasks := map[string]struct{}{}
	for _, d := range planned {
		for _, id := range dispatchCoveredTaskIDs(d) {
			plannedTasks[id] = struct{}{}
		}
	}
	for _, id := range credited {
		if _, redone := plannedTasks[id]; redone {
			t.Fatalf("running the surfaced recovery command %q dispatches credited task %s again; the owner was promised proven work is never redone", outcome.RedispatchCommand, id)
		}
	}
	for _, id := range unfinished {
		if _, ok := plannedTasks[id]; !ok {
			t.Fatalf("running the surfaced recovery command %q never dispatches unfinished task %s; the recovery does not recover", outcome.RedispatchCommand, id)
		}
	}
}

// TestForceOnlyRedispatchWouldRedoCreditedWork is the counter-fixture that
// proves the test above has teeth: the command that USED to be surfaced --
// `aether build <N> --force`, with no task filter -- really does re-plan every
// credited task. If someone reintroduces it, the lock above fails for a real
// reason and not a cosmetic one.
func TestForceOnlyRedispatchWouldRedoCreditedWork(t *testing.T) {
	tasks, ids := sixChainedTasks()
	credited := ids[:4]
	for i := range tasks {
		if i < len(credited) {
			tasks[i].Status = colony.TaskCompleted
		}
	}
	phase := colony.Phase{ID: 1, Name: "Six-task grouped job", Tasks: tasks}
	state := colony.ColonyState{
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: []colony.Phase{phase}},
	}

	_, taskArgs, force := parseRedispatchTaskArgs(t, buildForceRedispatchCommand(1))
	if !force || len(taskArgs) != 0 {
		t.Fatalf("buildForceRedispatchCommand(1) = %q, expected a bare --force with no task filter", buildForceRedispatchCommand(1))
	}

	planned, _, err := plannedBuildDispatchesWithJobProposals(
		phase, state, taskArgs, colony.VerificationDepthStandard, nil, "", nil, nil,
	)
	if err != nil {
		t.Fatalf("plan unfiltered phase: %v", err)
	}
	plannedTasks := map[string]struct{}{}
	for _, d := range planned {
		for _, id := range dispatchCoveredTaskIDs(d) {
			plannedTasks[id] = struct{}{}
		}
	}
	redone := 0
	for _, id := range credited {
		if _, ok := plannedTasks[id]; ok {
			redone++
		}
	}
	if redone == 0 {
		t.Fatal("an unfiltered phase build no longer re-plans credited tasks; this counter-fixture no longer proves anything and the retry lock above may be vacuous")
	}
}
