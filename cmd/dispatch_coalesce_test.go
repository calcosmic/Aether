package cmd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestCalVaultSixBatchesBecomeOneInRepoJob(t *testing.T) {
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

func TestGroupedJobBriefCarriesEveryTaskContract(t *testing.T) {
	saveGlobals(t)
	codeRepo(t)

	firstID := "3.1"
	secondID := "3.2"
	phase := colony.Phase{
		ID:   3,
		Name: "Grouped contract delivery",
		Tasks: []colony.Task{
			{
				ID:              &firstID,
				Goal:            "Implement the alpha behavior",
				Status:          colony.TaskPending,
				Constraints:     []string{"alpha-constraint"},
				Hints:           []string{"cmd/alpha.go", "follow-alpha-pattern"},
				SuccessCriteria: []string{"alpha-criterion"},
				EvidenceRequirements: []colony.CriterionEvidenceRequirement{{
					Criterion: "alpha-evidence", Artifacts: []string{"cmd/alpha.go"}, Checks: []string{"go test ./cmd -run Alpha"},
				}},
			},
			{
				ID:              &secondID,
				Goal:            "Add the beta regression coverage",
				Status:          colony.TaskPending,
				DependsOn:       []string{firstID},
				Constraints:     []string{"beta-constraint"},
				Hints:           []string{"cmd/beta_test.go", "follow-beta-pattern"},
				SuccessCriteria: []string{"beta-criterion"},
				EvidenceRequirements: []colony.CriterionEvidenceRequirement{{
					Criterion: "beta-evidence", Artifacts: []string{"cmd/beta_test.go"}, Checks: []string{"go test ./cmd -run Beta"},
				}},
			},
		},
	}
	dispatches := waveDispatchesOnly(plannedBuildDispatchesForSelectionWithState(
		phase, colony.ColonyState{}, nil, colony.VerificationDepthStandard,
	))
	if len(dispatches) != 1 {
		t.Fatalf("fixture should produce one grouped job, got %+v", dispatches)
	}

	brief := renderCodexBuildWorkerBrief(t.TempDir(), phase, dispatches[0], time.Unix(0, 0).UTC())
	for _, want := range []string{
		"## Covered Task Contracts",
		"Implement the alpha behavior", "alpha-constraint", "follow-alpha-pattern", "alpha-criterion", "alpha-evidence", "go test ./cmd -run Alpha",
		"Add the beta regression coverage", "beta-constraint", "follow-beta-pattern", "beta-criterion", "beta-evidence", "go test ./cmd -run Beta",
	} {
		if !strings.Contains(brief, want) {
			t.Errorf("grouped brief missing %q:\n%s", want, brief)
		}
	}
	for _, path := range []string{"cmd/alpha.go", "cmd/beta_test.go"} {
		if count := strings.Count(brief, path); count != 1 {
			t.Errorf("relevant path %q appears %d times, want exactly once:\n%s", path, count, brief)
		}
	}
}

func TestGroupedJobCreditsEveryCoveredTaskDuringContinue(t *testing.T) {
	firstID := "3.1"
	secondID := "3.2"
	phase := colony.Phase{
		ID: 3,
		Tasks: []colony.Task{
			{ID: &firstID, Goal: "Implement the behavior"},
			{ID: &secondID, Goal: "Add regression coverage"},
		},
	}
	manifest := codexContinueManifest{
		Present: true,
		Data: codexBuildManifest{
			DispatchMode: "real",
			Dispatches: []codexBuildDispatch{{
				Stage: "wave", Caste: "builder", Status: "completed", TaskID: firstID,
				CoveredTaskIDs: []string{firstID, secondID},
			}},
		},
	}
	verification := codexContinueVerificationReport{
		ChecksPassed: true,
		Passed:       true,
		Claims:       codexClaimVerification{Present: true, Passed: true},
	}

	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{}, time.Unix(0, 0).UTC())
	if !assessment.Passed || len(assessment.Tasks) != 2 {
		t.Fatalf("grouped task evidence did not support advancement: %+v", assessment)
	}
	for _, task := range assessment.Tasks {
		if task.Outcome != "verified" || !reflect.DeepEqual(task.DispatchStatuses, []string{"completed"}) {
			t.Fatalf("covered task %s lost grouped worker credit: %+v", task.TaskID, task)
		}
	}
}

func TestGroupedWorkerNameUsesOrderedCoveredIDs(t *testing.T) {
	firstID := "7.1"
	secondID := "7.2"
	phase := colony.Phase{
		ID: 7,
		Tasks: []colony.Task{
			{ID: &firstID, Goal: "Implement first step", Status: colony.TaskPending},
			{ID: &secondID, Goal: "Implement second step", Status: colony.TaskPending, DependsOn: []string{firstID}},
		},
	}
	plan := func(proposalName string) codexBuildDispatch {
		dispatches, _, err := plannedBuildDispatchesWithJobProposals(
			phase, colony.ColonyState{}, nil, colony.VerificationDepthStandard, nil, "", nil,
			[]coherentJobProposal{{
				Name: proposalName, TaskIDs: []string{firstID, secondID}, OwnerCaste: "builder",
				Relationship: "dependency_chain", Benefit: "one implementation context",
			}},
		)
		if err != nil {
			t.Fatalf("plan proposal %q: %v", proposalName, err)
		}
		waves := waveDispatchesOnly(dispatches)
		if len(waves) != 1 {
			t.Fatalf("proposal %q produced %+v", proposalName, waves)
		}
		return waves[0]
	}
	first := plan("queen-name-one")
	second := plan("queen-name-two")
	if first.Name != second.Name {
		t.Fatalf("proposal wording changed worker identity: %q != %q", first.Name, second.Name)
	}
	want := deterministicAntName(first.Caste, "phase:7:job:builder:7.1,7.2")
	if first.Name != want {
		t.Fatalf("grouped worker name = %q, want ordered covered-ID seed %q", first.Name, want)
	}
}

func TestSingleTaskWorkerNameIsStable(t *testing.T) {
	taskID := "8.1"
	task := colony.Task{ID: &taskID, Goal: "Implement the stable single task", Status: colony.TaskPending}
	phase := colony.Phase{ID: 8, Tasks: []colony.Task{task}}
	dispatches := waveDispatchesOnly(plannedBuildDispatchesForSelectionWithState(
		phase, colony.ColonyState{}, nil, colony.VerificationDepthStandard,
	))
	if len(dispatches) != 1 {
		t.Fatalf("single task produced %+v", dispatches)
	}
	want := deterministicAntName(dispatches[0].Caste, fmt.Sprintf("phase:%d:task:%d:%s", phase.ID, 0, task.Goal))
	if dispatches[0].Name != want {
		t.Fatalf("single-task worker name changed: got %q want legacy %q", dispatches[0].Name, want)
	}
}

func TestSelectedTaskGroupingStaysInScope(t *testing.T) {
	firstID := "9.1"
	secondID := "9.2"
	phase := colony.Phase{
		ID: 9,
		Tasks: []colony.Task{
			{ID: &firstID, Goal: "Leave this task outside redispatch", Status: colony.TaskPending},
			{ID: &secondID, Goal: "Redispatch only this task", Status: colony.TaskPending, DependsOn: []string{firstID}},
		},
	}
	dispatches := waveDispatchesOnly(plannedBuildDispatchesForSelectionWithState(
		phase, colony.ColonyState{}, []string{secondID}, colony.VerificationDepthStandard,
	))
	if len(dispatches) != 1 || !reflect.DeepEqual(dispatchCoveredTaskIDs(dispatches[0]), []string{secondID}) {
		t.Fatalf("selected-task grouping escaped scope: %+v", dispatches)
	}
}

func TestGroupedBuildSummaryUsesCoveredTasksLanguage(t *testing.T) {
	summary := renderCoveredTaskSummary([]codexBuildDispatch{{CoveredTaskIDs: []string{"1.1", "1.2"}}})
	if !strings.Contains(summary, "covered tasks") || strings.Contains(summary, "dependent tasks") {
		t.Fatalf("grouped summary uses stale dependency wording: %q", summary)
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
