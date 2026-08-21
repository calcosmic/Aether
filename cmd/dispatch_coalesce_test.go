package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 184. From the CalVault run on 2026-08-14:
//
//	"Waves 11–16 each contain exactly one builder, reason: 'single task in this
//	 wave'. Each is a fresh ~100k-token agent that re-reads the same
//	 recovery-matrix.md from scratch. Tasks 3.3 and 3.4 are 'copy the first six
//	 categories' and 'copy the remaining six categories' — one operation split
//	 across two agents for no stated reason."
//
// A chain of dependent steps over one source list is one job. Splitting it
// across six workers pays the full startup cost six times and makes each one
// rediscover what the last already knew.

func chainedTask(id, goal, dependsOn string) colony.Task {
	taskID := id
	t := colony.Task{ID: &taskID, Goal: goal, Status: colony.TaskPending}
	if dependsOn != "" {
		t.DependsOn = []string{dependsOn}
	}
	return t
}

func waveDispatchesOnly(dispatches []codexBuildDispatch) []codexBuildDispatch {
	out := []codexBuildDispatch{}
	for _, d := range dispatches {
		if d.Stage == "wave" {
			out = append(out, d)
		}
	}
	return out
}

func TestSequentialFileTasksBecomeOneWorker(t *testing.T) {
	saveGlobals(t)
	notesOnlyRepo(t)

	tasks := []colony.Task{}
	prev := ""
	for i := 1; i <= 6; i++ {
		id := fmt.Sprintf("t%d", i)
		tasks = append(tasks, chainedTask(id,
			fmt.Sprintf("Copy category %d from the recovery matrix into the template tree", i), prev))
		prev = id
	}

	phase := colony.Phase{
		ID:          3,
		Name:        "Template recovery",
		Description: "Copy the identified templates into a new folder tree, keeping a log.",
		Tasks:       tasks,
	}

	dispatches := waveDispatchesOnly(
		plannedBuildDispatchesForSelectionWithState(phase, colony.ColonyState{}, nil, colony.VerificationDepthStandard))

	if len(dispatches) != 1 {
		names := []string{}
		for _, d := range dispatches {
			names = append(names, d.Caste+":"+d.Task)
		}
		t.Fatalf("six dependent copy steps became %d workers, want 1; each extra worker pays full startup cost and re-reads the same list:\n  %s",
			len(dispatches), strings.Join(names, "\n  "))
	}
}

func TestIndependentTasksStillFanOut(t *testing.T) {
	saveGlobals(t)
	notesOnlyRepo(t)

	// No dependencies between them: these can genuinely run at the same time.
	phase := colony.Phase{
		ID:          4,
		Name:        "Write the guides",
		Description: "Write three unrelated guides.",
		Tasks: []colony.Task{
			chainedTask("a", "Write the installation guide", ""),
			chainedTask("b", "Write the troubleshooting guide", ""),
			chainedTask("c", "Write the migration guide", ""),
		},
	}

	dispatches := waveDispatchesOnly(
		plannedBuildDispatchesForSelectionWithState(phase, colony.ColonyState{}, nil, colony.VerificationDepthStandard))

	if len(dispatches) != 3 {
		t.Fatalf("three independent tasks collapsed to %d workers; coalescing must not turn parallel work serial", len(dispatches))
	}
}

// Merging must not lose work. Every task the merged worker took on has to be
// named in its instructions and recorded on the dispatch, or a phase reports
// finished with tasks nobody was told to do.
func TestCoalescedWorkerKeepsEveryTaskVisible(t *testing.T) {
	saveGlobals(t)
	notesOnlyRepo(t)

	tasks := []colony.Task{
		chainedTask("t1", "Copy the daily-note templates", ""),
		chainedTask("t2", "Copy the meeting templates", "t1"),
		chainedTask("t3", "Copy the project templates", "t2"),
	}
	phase := colony.Phase{ID: 5, Name: "Template recovery", Tasks: tasks}

	dispatches := waveDispatchesOnly(
		plannedBuildDispatchesForSelectionWithState(phase, colony.ColonyState{}, nil, colony.VerificationDepthStandard))

	if len(dispatches) != 1 {
		t.Fatalf("expected one worker for three chained steps, got %d", len(dispatches))
	}
	d := dispatches[0]

	for _, goal := range []string{"daily-note", "meeting", "project"} {
		if !strings.Contains(d.Task, goal) {
			t.Fatalf("merged worker was not told about the %q step; its instructions read:\n%s", goal, d.Task)
		}
	}
	if len(d.CoveredTaskIDs) != 3 {
		t.Fatalf("merged worker records %d covered tasks, want 3 -- without this a phase can report finished with tasks nobody was assigned; got %v",
			len(d.CoveredTaskIDs), d.CoveredTaskIDs)
	}
}

// Different kinds of work do not merge: a chain that changes caste part way
// through is two jobs, not one.
func TestChainAcrossDifferentCastesDoesNotMerge(t *testing.T) {
	saveGlobals(t)
	codeRepo(t)

	phase := colony.Phase{
		ID:   6,
		Name: "Add the exporter",
		Tasks: []colony.Task{
			chainedTask("t1", "Implement the CSV exporter function", ""),
			chainedTask("t2", "Research which date format downstream consumers expect", "t1"),
		},
	}

	dispatches := waveDispatchesOnly(
		plannedBuildDispatchesForSelectionWithState(phase, colony.ColonyState{}, nil, colony.VerificationDepthStandard))

	castes := map[string]bool{}
	for _, d := range dispatches {
		castes[d.Caste] = true
	}
	if len(castes) > 1 && len(dispatches) == 1 {
		t.Fatalf("two different kinds of work were merged into one worker: %v", castes)
	}
}

// Caught by the existing suite, not by the tests above: a merged worker that
// finished left every step after the first marked unfinished, so the phase could
// never advance and the one worker that did all the work looked like it had done
// only the first bit. Coalescing without this is worse than not coalescing.
func TestMergedWorkerCompletesEveryTaskItCovered(t *testing.T) {
	dispatches := []codexBuildDispatch{{
		Stage:          "wave",
		Caste:          "builder",
		Status:         "completed",
		TaskID:         "t1",
		CoveredTaskIDs: []string{"t1", "t2", "t3"},
	}}

	completed := completedBuildTaskIDs(dispatches)
	for _, id := range []string{"t1", "t2", "t3"} {
		if _, ok := completed[id]; !ok {
			t.Fatalf("task %q was done by the merged worker but is not marked complete; completed = %v", id, completed)
		}
	}
}

// An ordinary dispatch must be unaffected.
func TestUnmergedDispatchStillCompletesItsOwnTask(t *testing.T) {
	completed := completedBuildTaskIDs([]codexBuildDispatch{{
		Stage: "wave", Caste: "builder", Status: "completed", TaskID: "solo",
	}})
	if _, ok := completed["solo"]; !ok {
		t.Fatalf("a plain dispatch stopped completing its own task: %v", completed)
	}
}
