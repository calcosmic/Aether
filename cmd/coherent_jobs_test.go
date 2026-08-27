package cmd

import (
	"fmt"
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

func coherentJobContainingTask(t *testing.T, plan *coherentJobPlan, taskID string) coherentJob {
	t.Helper()
	for _, job := range plan.Jobs {
		for _, candidateID := range job.TaskIDs {
			if candidateID == taskID {
				return job
			}
		}
	}
	t.Fatalf("task %q was not assigned to a coherent job: %#v", taskID, plan.Jobs)
	return coherentJob{}
}

func coherentJobTestTaskWithGoal(id, caste, goal string, dependsOn ...string) (colony.Task, coherentJobTask) {
	task, seed := coherentJobTestTask(id, caste, dependsOn...)
	task.Goal = goal
	seed.Task = task
	return task, seed
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

func TestCoherentJobsGroupMeaningfulPathsAndDependencies(t *testing.T) {
	t.Run("exact meaningful path groups same-caste tasks", func(t *testing.T) {
		taskA, seedA := coherentJobTestTask("runtime-a", "builder")
		taskB, seedB := coherentJobTestTask("runtime-b", "builder")
		taskA.Hints = []string{"cmd/runtime.go"}
		taskB.EvidenceRequirements = []colony.CriterionEvidenceRequirement{{
			Criterion: "runtime behavior is implemented",
			Artifacts: []string{"cmd/runtime.go"},
		}}

		plan, err := planCoherentJobs(
			coherentJobTestPhase(195, taskA, taskB),
			[]coherentJobTask{seedA, seedB},
			nil,
		)
		if err != nil {
			t.Fatalf("meaningful-path grouping returned error: %v", err)
		}
		if len(plan.Jobs) != 1 {
			t.Fatalf("same-caste tasks sharing an exact source path planned as %d jobs, want 1: %#v", len(plan.Jobs), plan.Jobs)
		}
		job := plan.Jobs[0]
		if job.Source != coherentJobSourceAutomatic || job.OwnerCaste != "builder" {
			t.Fatalf("meaningful-path component lost its automatic source or owner: %#v", job)
		}
		if got, want := strings.Join(job.TaskIDs, ","), "runtime-a,runtime-b"; got != want {
			t.Fatalf("meaningful-path component order = %q, want %q", got, want)
		}
		for _, required := range []string{"cmd/runtime.go", "preserves", "avoids"} {
			if !strings.Contains(job.JobReason, required) {
				t.Fatalf("automatic job reason %q does not explain %q", job.JobReason, required)
			}
		}
	})

	t.Run("non-consecutive dependencies form one topological component", func(t *testing.T) {
		// Deliberately put dependent tasks before their prerequisites in plan
		// order. The component must use dependency order, with plan order only
		// as the tie-breaker between simultaneously runnable tasks.
		taskC, seedC := coherentJobTestTask("dep-c", "builder", "dep-b")
		other, otherSeed := coherentJobTestTask("unrelated", "scout")
		taskB, seedB := coherentJobTestTask("dep-b", "builder", "dep-a")
		taskA, seedA := coherentJobTestTask("dep-a", "builder")

		plan, err := planCoherentJobs(
			coherentJobTestPhase(195, taskC, other, taskB, taskA),
			[]coherentJobTask{seedC, otherSeed, seedB, seedA},
			nil,
		)
		if err != nil {
			t.Fatalf("dependency grouping returned error: %v", err)
		}
		job := coherentJobContainingTask(t, plan, "dep-a")
		if got, want := strings.Join(job.TaskIDs, ","), "dep-a,dep-b,dep-c"; got != want {
			t.Fatalf("dependency component order = %q, want %q", got, want)
		}
		if !strings.Contains(job.JobReason, "declared dependenc") {
			t.Fatalf("dependency job reason does not name its relationship: %q", job.JobReason)
		}
		if otherJob := coherentJobContainingTask(t, plan, "unrelated"); len(otherJob.TaskIDs) != 1 {
			t.Fatalf("unrelated task was pulled into dependency component: %#v", otherJob)
		}
	})
}

func TestCoherentJobsIgnoreIncidentalPaths(t *testing.T) {
	incidentalPaths := []string{
		"README",
		"README.md",
		"docs/readme.rst",
		"docs/CHANGELOG.md",
		"docs/HISTORY.md",
		"docs/release-notes.md",
		"docs/RELEASE_NOTES.md",
		"go.mod",
		"go.sum",
		"package.json",
		"package-lock.json",
		"npm-shrinkwrap.json",
		"yarn.lock",
		"pnpm-lock.yaml",
		"pyproject.toml",
		"requirements-dev.txt",
		"poetry.lock",
		"Pipfile",
		"Pipfile.lock",
		"uv.lock",
		"Cargo.toml",
		"Cargo.lock",
		"Gemfile",
		"Gemfile.lock",
	}
	for _, declaredPath := range incidentalPaths {
		declaredPath := declaredPath
		t.Run(strings.ReplaceAll(declaredPath, "/", "_"), func(t *testing.T) {
			if !isIncidentalGroupingPath(declaredPath) {
				t.Fatalf("%q must be classified as incidental grouping input", declaredPath)
			}
			taskA, seedA := coherentJobTestTask("incidental-a", "builder")
			taskB, seedB := coherentJobTestTask("incidental-b", "builder")
			seedA.DeclaredPaths = []string{declaredPath}
			seedB.DeclaredPaths = []string{declaredPath}
			plan, err := planCoherentJobs(
				coherentJobTestPhase(195, taskA, taskB),
				[]coherentJobTask{seedA, seedB},
				nil,
			)
			if err != nil {
				t.Fatalf("incidental-path fixture returned error: %v", err)
			}
			if len(plan.Jobs) != 2 {
				t.Fatalf("shared incidental path %q collapsed unrelated tasks into %#v", declaredPath, plan.Jobs)
			}
		})
	}

	if isIncidentalGroupingPath("cmd/coherent_jobs.go") {
		t.Fatal("a source file was classified as incidental")
	}
}

func TestCoherentJobsRespectCasteBoundaries(t *testing.T) {
	t.Run("shared path and dependency do not cross castes", func(t *testing.T) {
		taskA, seedA := coherentJobTestTask("build-runtime", "builder")
		taskB, seedB := coherentJobTestTask("inspect-runtime", "scout", "build-runtime")
		seedA.DeclaredPaths = []string{"cmd/runtime.go"}
		seedB.DeclaredPaths = []string{"cmd/runtime.go"}

		plan, err := planCoherentJobs(
			coherentJobTestPhase(195, taskA, taskB),
			[]coherentJobTask{seedA, seedB},
			nil,
		)
		if err != nil {
			t.Fatalf("cross-caste automatic fixture returned error: %v", err)
		}
		if len(plan.Jobs) != 2 {
			t.Fatalf("automatic grouping crossed caste boundary: %#v", plan.Jobs)
		}
		buildJob := coherentJobContainingTask(t, plan, "build-runtime")
		inspectJob := coherentJobContainingTask(t, plan, "inspect-runtime")
		if buildJob.OwnerCaste != "builder" || inspectJob.OwnerCaste != "scout" {
			t.Fatalf("automatic jobs changed task owners: %#v", plan.Jobs)
		}
		if got, want := strings.Join(inspectJob.DependsOn, ","), buildJob.Name; got != want {
			t.Fatalf("cross-caste dependency edge = %q, want %q", got, want)
		}
		if buildJob.Wave != 1 || inspectJob.Wave != 2 {
			t.Fatalf("cross-caste waves = %d -> %d, want 1 -> 2", buildJob.Wave, inspectJob.Wave)
		}
	})

	t.Run("meaningful path cannot contract around an external dependency", func(t *testing.T) {
		taskA, seedA := coherentJobTestTask("boundary-a", "builder")
		taskB, seedB := coherentJobTestTask("boundary-b", "scout", "boundary-a")
		taskC, seedC := coherentJobTestTask("boundary-c", "builder", "boundary-b")
		seedA.DeclaredPaths = []string{"cmd/shared.go"}
		seedC.DeclaredPaths = []string{"cmd/shared.go"}

		plan, err := planCoherentJobs(
			coherentJobTestPhase(195, taskA, taskB, taskC),
			[]coherentJobTask{seedA, seedB, seedC},
			nil,
		)
		if err != nil {
			t.Fatalf("dependency-safe boundary fixture returned error: %v", err)
		}
		if len(plan.Jobs) != 3 {
			t.Fatalf("shared-path contraction created an unsafe component: %#v", plan.Jobs)
		}
		if got := len(plan.Waves); got != 3 {
			t.Fatalf("dependency-safe boundary produced %d waves, want 3: %#v", got, plan.Waves)
		}
	})
}

func TestCoherentJobsUseBriefAllowanceNotTaskCount(t *testing.T) {
	t.Run("projection includes every task-owned brief text class", func(t *testing.T) {
		task, seed := coherentJobTestTaskWithGoal("projected", "builder", "goal text", "dependency")
		task.Constraints = []string{"constraint text"}
		task.Hints = []string{"hint text"}
		task.SuccessCriteria = []string{"criterion text"}
		task.EvidenceRequirements = []colony.CriterionEvidenceRequirement{{
			Criterion: "evidence criterion",
			Artifacts: []string{"cmd/projected.go"},
			Checks:    []string{"go test ./cmd"},
		}}
		seed.Task = task
		want := len(task.Goal) + len(task.DependsOn[0]) +
			len(task.Constraints[0]) + len(task.Hints[0]) + len(task.SuccessCriteria[0]) +
			len(task.EvidenceRequirements[0].Criterion) + len(task.EvidenceRequirements[0].Artifacts[0]) + len(task.EvidenceRequirements[0].Checks[0])
		if got := projectCoherentJobBriefChars([]coherentJobTask{seed}); got != want {
			t.Fatalf("brief projection = %d, want exact task-owned text sum %d", got, want)
		}
	})

	t.Run("many small tasks stay together below the content allowance", func(t *testing.T) {
		const taskCount = 64
		tasks := make([]colony.Task, 0, taskCount)
		seeds := make([]coherentJobTask, 0, taskCount)
		for i := 0; i < taskCount; i++ {
			id := fmt.Sprintf("small-%02d", i+1)
			task, seed := coherentJobTestTaskWithGoal(id, "builder", "Apply one tiny field change")
			seed.DeclaredPaths = []string{"cmd/small_shared.go"}
			tasks = append(tasks, task)
			seeds = append(seeds, seed)
		}

		plan, err := planCoherentJobs(coherentJobTestPhase(195, tasks...), seeds, nil)
		if err != nil {
			t.Fatalf("many-small-task fixture returned error: %v", err)
		}
		if len(plan.Jobs) != 1 || len(plan.Jobs[0].TaskIDs) != taskCount {
			t.Fatalf("task-count heuristic split a brief-sized component: jobs=%d first=%#v", len(plan.Jobs), plan.Jobs[0])
		}
		if got := projectCoherentJobBriefChars(plan.Jobs[0].Tasks); got >= briefTaskContentAllowanceChars {
			t.Fatalf("fixture no longer proves a below-allowance component: %d >= %d", got, briefTaskContentAllowanceChars)
		}
	})

	t.Run("large automatic component splits at the content allowance", func(t *testing.T) {
		taskA, seedA := coherentJobTestTaskWithGoal("large-a", "builder", strings.Repeat("a", 3500))
		taskB, seedB := coherentJobTestTaskWithGoal("large-b", "builder", strings.Repeat("b", 3500), "large-a")
		seedA.DeclaredPaths = []string{"cmd/large_shared.go"}
		seedB.DeclaredPaths = []string{"cmd/large_shared.go"}

		plan, err := planCoherentJobs(
			coherentJobTestPhase(195, taskA, taskB),
			[]coherentJobTask{seedA, seedB},
			nil,
		)
		if err != nil {
			t.Fatalf("large automatic fixture returned error: %v", err)
		}
		if len(plan.Jobs) != 2 {
			t.Fatalf("over-allowance automatic component planned as %d jobs, want 2: %#v", len(plan.Jobs), plan.Jobs)
		}
		first := coherentJobContainingTask(t, plan, "large-a")
		second := coherentJobContainingTask(t, plan, "large-b")
		if got, want := strings.Join(second.DependsOn, ","), first.Name; got != want {
			t.Fatalf("brief split lost dependency order: %q, want %q", got, want)
		}
	})

	t.Run("one oversized task stays one job", func(t *testing.T) {
		task, seed := coherentJobTestTaskWithGoal("oversized-single", "builder", strings.Repeat("x", briefTaskContentAllowanceChars+1))
		plan, err := planCoherentJobs(coherentJobTestPhase(195, task), []coherentJobTask{seed}, nil)
		if err != nil {
			t.Fatalf("oversized single task returned error: %v", err)
		}
		if len(plan.Jobs) != 1 {
			t.Fatalf("oversized single task was dropped or fragmented: %#v", plan.Jobs)
		}
		if got := strings.Join(plan.Jobs[0].TaskIDs, ","); got != "oversized-single" {
			t.Fatalf("oversized single task was dropped or fragmented: %#v", plan.Jobs)
		}
	})

	t.Run("Queen proposal is not rewritten by the soft limit", func(t *testing.T) {
		taskA, seedA := coherentJobTestTaskWithGoal("queen-large-a", "builder", strings.Repeat("a", 3500))
		taskB, seedB := coherentJobTestTaskWithGoal("queen-large-b", "builder", strings.Repeat("b", 3500), "queen-large-a")
		plan, err := planCoherentJobs(
			coherentJobTestPhase(195, taskA, taskB),
			[]coherentJobTask{seedA, seedB},
			[]coherentJobProposal{{
				Name:         "queen-large-job",
				TaskIDs:      []string{"queen-large-a", "queen-large-b"},
				OwnerCaste:   "builder",
				Relationship: "the tasks implement one deliberately large contract",
				Benefit:      "the Queen prefers one owner despite the soft brief limit",
			}},
		)
		if err != nil {
			t.Fatalf("large Queen proposal returned error: %v", err)
		}
		job := coherentJobByName(t, plan, "queen-large-job")
		if job.Source != coherentJobSourceQueen || len(job.TaskIDs) != 2 {
			t.Fatalf("soft limit rewrote an accepted Queen proposal: %#v", job)
		}
	})
}

func TestCalVaultSixBatchesPlanAsOneCoherentJob(t *testing.T) {
	goals := []string{
		"Copy CalVault file batch 1 and record its recovery-matrix entries",
		"Copy CalVault file batch 2 and record its recovery-matrix entries",
		"Copy CalVault file batch 3 and record its recovery-matrix entries",
		"Copy CalVault file batch 4 and record its recovery-matrix entries",
		"Copy CalVault file batch 5 and record its recovery-matrix entries",
		"Copy CalVault file batch 6 and record its recovery-matrix entries",
	}
	tasks := make([]colony.Task, 0, len(goals))
	seeds := make([]coherentJobTask, 0, len(goals))
	wantIDs := make([]string, 0, len(goals))
	for i, goal := range goals {
		id := fmt.Sprintf("calvault-copy-batch-%d", i+1)
		var dependencies []string
		if i > 0 {
			dependencies = []string{wantIDs[i-1]}
		}
		task, seed := coherentJobTestTaskWithGoal(id, "builder", goal, dependencies...)
		seed.DeclaredPaths = []string{"docs/recovery-matrix.md"}
		tasks = append(tasks, task)
		seeds = append(seeds, seed)
		wantIDs = append(wantIDs, id)
	}

	plan, err := planCoherentJobs(coherentJobTestPhase(195, tasks...), seeds, nil)
	if err != nil {
		t.Fatalf("CalVault six-batch fixture returned error: %v", err)
	}
	if len(plan.Jobs) != 1 {
		t.Fatalf("six literal CalVault batches planned as %d jobs, want 1: %#v", len(plan.Jobs), plan.Jobs)
	}
	job := plan.Jobs[0]
	if job.Source != coherentJobSourceAutomatic || job.OwnerCaste != "builder" {
		t.Fatalf("CalVault component lost automatic Builder ownership: %#v", job)
	}
	if got, want := strings.Join(job.TaskIDs, ","), strings.Join(wantIDs, ","); got != want {
		t.Fatalf("CalVault batch order = %q, want %q", got, want)
	}
}
