package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func coherentJobTestTask(id, caste string, dependsOn ...string) (colony.Task, coherentJobTask) {
	taskID := id
	task := colony.Task{
		ID:        &taskID,
		Goal:      "Complete " + id,
		Status:    colony.TaskPending,
		DependsOn: append([]string{}, dependsOn...),
	}
	return task, coherentJobTask{
		Task:      task,
		ID:        id,
		Caste:     caste,
		TaskIndex: 0,
	}
}

func coherentJobTestPhase(id int, tasks ...colony.Task) colony.Phase {
	return colony.Phase{ID: id, Name: "Coherent job fixture", Tasks: tasks}
}

func coherentJobDecisionByName(t *testing.T, plan *coherentJobPlan, name string) coherentJobDecision {
	t.Helper()
	for _, decision := range plan.Decisions {
		if decision.ProposalName == name {
			return decision
		}
	}
	t.Fatalf("decision for proposal %q was not recorded: %#v", name, plan.Decisions)
	return coherentJobDecision{}
}

func coherentJobByName(t *testing.T, plan *coherentJobPlan, name string) coherentJob {
	t.Helper()
	for _, job := range plan.Jobs {
		if job.Name == name {
			return job
		}
	}
	t.Fatalf("job %q was not planned: %#v", name, plan.Jobs)
	return coherentJob{}
}

func coherentJobTaskIDs(plan *coherentJobPlan) map[string]int {
	covered := map[string]int{}
	for _, job := range plan.Jobs {
		for _, taskID := range job.TaskIDs {
			covered[taskID]++
		}
	}
	return covered
}

func TestCoherentJobProposalAccepted(t *testing.T) {
	taskA, seedA := coherentJobTestTask("task-a", "builder")
	taskB, seedB := coherentJobTestTask("task-b", "builder", "task-a")
	seedB.TaskIndex = 1

	plan, err := planCoherentJobs(
		coherentJobTestPhase(195, taskA, taskB),
		[]coherentJobTask{seedA, seedB},
		[]coherentJobProposal{{
			Name:         "one-builder-job",
			TaskIDs:      []string{"task-a", "task-b"},
			OwnerCaste:   "Builder",
			Relationship: "the tasks implement the same runtime contract",
			Benefit:      "one Builder preserves context and avoids duplicate setup",
		}},
	)
	if err != nil {
		t.Fatalf("safe proposal returned an error: %v", err)
	}
	if plan == nil {
		t.Fatal("safe proposal returned no coherent-job plan")
	}

	decision := coherentJobDecisionByName(t, plan, "one-builder-job")
	if decision.Status != "accepted" {
		t.Fatalf("safe proposal status = %q, want accepted; decision = %#v", decision.Status, decision)
	}
	job := coherentJobByName(t, plan, "one-builder-job")
	if job.OwnerCaste != "builder" {
		t.Fatalf("normalized owner caste = %q, want builder", job.OwnerCaste)
	}
	if job.Source != "queen" {
		t.Fatalf("accepted proposal source = %q, want queen", job.Source)
	}
	if got, want := strings.Join(job.TaskIDs, ","), "task-a,task-b"; got != want {
		t.Fatalf("accepted proposal task order = %q, want %q", got, want)
	}
	if got, want := job.JobReason, "the tasks implement the same runtime contract, so one Builder preserves context and avoids duplicate setup"; got != want {
		t.Fatalf("job reason = %q, want structured relationship and benefit %q", got, want)
	}
}

func TestCoherentJobProposalOrderRefusedByName(t *testing.T) {
	taskA, seedA := coherentJobTestTask("task-a", "builder")
	taskB, seedB := coherentJobTestTask("task-b", "builder", "task-a")
	seedB.TaskIndex = 1

	plan, err := planCoherentJobs(
		coherentJobTestPhase(195, taskA, taskB),
		[]coherentJobTask{seedA, seedB},
		[]coherentJobProposal{{
			Name:         "unsafe-order",
			TaskIDs:      []string{"task-b", "task-a"},
			OwnerCaste:   "builder",
			Relationship: "the tasks implement the same runtime contract",
			Benefit:      "one Builder preserves context",
		}},
	)
	if err != nil {
		t.Fatalf("an unsafe proposal should be repaired locally, got hard error: %v", err)
	}
	if plan == nil {
		t.Fatal("an unsafe proposal should return a locally repaired plan")
	}

	decision := coherentJobDecisionByName(t, plan, "unsafe-order")
	if decision.Status != "refused" {
		t.Fatalf("unsafe proposal status = %q, want refused", decision.Status)
	}
	if decision.OffendingTaskID != "task-b" || decision.DependencyID != "task-a" {
		t.Fatalf("order refusal must name proposal member and unmet dependency: %#v", decision)
	}
	for _, required := range []string{"unsafe-order", "task-b", "task-a"} {
		if !strings.Contains(decision.Reason, required) {
			t.Fatalf("order refusal %q does not name %q", decision.Reason, required)
		}
	}
	if len(decision.ReplacementJobNames) == 0 {
		t.Fatalf("refused proposal has no visible runtime replacement jobs: %#v", decision)
	}
	if covered := coherentJobTaskIDs(plan); covered["task-a"] != 1 || covered["task-b"] != 1 {
		t.Fatalf("local repair did not preserve exactly one assignment per member: %v", covered)
	}
	for _, job := range plan.Jobs {
		if job.Name == "unsafe-order" || job.Source == "queen" {
			t.Fatalf("unsafe Queen job survived refusal: %#v", job)
		}
	}
}

func TestCoherentJobRepairKeepsSafeProposals(t *testing.T) {
	taskA, seedA := coherentJobTestTask("safe-a", "builder")
	taskB, seedB := coherentJobTestTask("safe-b", "builder", "safe-a")
	taskC, seedC := coherentJobTestTask("repair-a", "builder")
	taskD, seedD := coherentJobTestTask("repair-b", "builder", "repair-a")
	taskE, seedE := coherentJobTestTask("safe-c", "builder")
	taskF, seedF := coherentJobTestTask("safe-d", "builder", "safe-c")
	seeds := []coherentJobTask{seedA, seedB, seedC, seedD, seedE, seedF}
	for i := range seeds {
		seeds[i].TaskIndex = i
	}

	plan, err := planCoherentJobs(
		coherentJobTestPhase(195, taskA, taskB, taskC, taskD, taskE, taskF),
		seeds,
		[]coherentJobProposal{
			{Name: "safe-one", TaskIDs: []string{"safe-a", "safe-b"}, OwnerCaste: "builder", Relationship: "the first pair shares one contract", Benefit: "one worker keeps the contract consistent"},
			{Name: "unsafe-middle", TaskIDs: []string{"repair-b", "repair-a"}, OwnerCaste: "builder", Relationship: "the middle pair shares one contract", Benefit: "one worker keeps the contract consistent"},
			{Name: "safe-two", TaskIDs: []string{"safe-c", "safe-d"}, OwnerCaste: "builder", Relationship: "the final pair shares one contract", Benefit: "one worker keeps the contract consistent"},
		},
	)
	if err != nil {
		t.Fatalf("local repair returned hard error: %v", err)
	}

	for _, name := range []string{"safe-one", "safe-two"} {
		decision := coherentJobDecisionByName(t, plan, name)
		if decision.Status != "accepted" {
			t.Fatalf("unrelated safe proposal %q changed after another refusal: %#v", name, decision)
		}
		job := coherentJobByName(t, plan, name)
		if job.Source != "queen" {
			t.Fatalf("safe proposal %q no longer owns its Queen job: %#v", name, job)
		}
	}
	refused := coherentJobDecisionByName(t, plan, "unsafe-middle")
	if refused.Status != "refused" || len(refused.ReplacementJobNames) == 0 {
		t.Fatalf("unsafe proposal was not visibly repaired: %#v", refused)
	}
	if covered := coherentJobTaskIDs(plan); len(covered) != 6 {
		t.Fatalf("repair changed task scope: %v", covered)
	}
}

func TestCoherentJobCrossCasteNeedsOwnerReason(t *testing.T) {
	taskA, seedA := coherentJobTestTask("build-task", "builder")
	taskB, seedB := coherentJobTestTask("research-task", "scout", "build-task")
	seedB.TaskIndex = 1
	phase := coherentJobTestPhase(195, taskA, taskB)
	base := coherentJobProposal{
		Name:         "cross-caste",
		TaskIDs:      []string{"build-task", "research-task"},
		OwnerCaste:   "builder",
		Relationship: "the implementation depends on the research finding",
		Benefit:      "one owner can apply the finding without a handoff",
	}

	t.Run("missing owner reason is refused", func(t *testing.T) {
		plan, err := planCoherentJobs(phase, []coherentJobTask{seedA, seedB}, []coherentJobProposal{base})
		if err != nil {
			t.Fatalf("missing cross-caste owner reason should be locally refused: %v", err)
		}
		decision := coherentJobDecisionByName(t, plan, "cross-caste")
		if decision.Status != "refused" || !strings.Contains(decision.Reason, "owner_reason") {
			t.Fatalf("missing owner_reason was not named in refusal: %#v", decision)
		}
		for _, job := range plan.Jobs {
			if len(job.TaskIDs) > 1 && job.Source != "queen" {
				t.Fatalf("automatic repair crossed caste boundary: %#v", job)
			}
		}
	})

	t.Run("stated owner reason accepts one suitable owner", func(t *testing.T) {
		proposal := base
		proposal.OwnerReason = "the Builder can consume the research and implement the dependent change end to end"
		plan, err := planCoherentJobs(phase, []coherentJobTask{seedA, seedB}, []coherentJobProposal{proposal})
		if err != nil {
			t.Fatalf("reasoned cross-caste proposal returned error: %v", err)
		}
		decision := coherentJobDecisionByName(t, plan, "cross-caste")
		if decision.Status != "accepted" {
			t.Fatalf("reasoned cross-caste proposal was not accepted: %#v", decision)
		}
		job := coherentJobByName(t, plan, "cross-caste")
		if job.OwnerCaste != "builder" || len(job.TaskIDs) != 2 {
			t.Fatalf("reasoned cross-caste proposal lost its owner or members: %#v", job)
		}
	})
}

func TestCoherentJobGraphErrorsHaveNoPlan(t *testing.T) {
	t.Run("missing dependency", func(t *testing.T) {
		task, seed := coherentJobTestTask("orphan", "builder", "missing-task")
		plan, err := planCoherentJobs(coherentJobTestPhase(195, task), []coherentJobTask{seed}, nil)
		if err == nil || plan != nil {
			t.Fatalf("missing dependency must return an error and no plan; plan=%#v err=%v", plan, err)
		}
		for _, required := range []string{"orphan", "missing-task", "repair depends_on in the plan"} {
			if !strings.Contains(err.Error(), required) {
				t.Fatalf("missing-dependency error %q does not contain %q", err, required)
			}
		}
	})

	t.Run("named cycle", func(t *testing.T) {
		taskA, seedA := coherentJobTestTask("cycle-a", "builder", "cycle-c")
		taskB, seedB := coherentJobTestTask("cycle-b", "builder", "cycle-a")
		taskC, seedC := coherentJobTestTask("cycle-c", "builder", "cycle-b")
		seedB.TaskIndex = 1
		seedC.TaskIndex = 2
		plan, err := planCoherentJobs(
			coherentJobTestPhase(195, taskA, taskB, taskC),
			[]coherentJobTask{seedA, seedB, seedC},
			nil,
		)
		if err == nil || plan != nil {
			t.Fatalf("cycle must return an error and no plan; plan=%#v err=%v", plan, err)
		}
		for _, required := range []string{"cycle-a", "cycle-b", "cycle-c", "repair depends_on in the plan"} {
			if !strings.Contains(err.Error(), required) {
				t.Fatalf("cycle error %q does not contain %q", err, required)
			}
		}
	})
}
