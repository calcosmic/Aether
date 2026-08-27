package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

const (
	coherentJobSourceQueen     = "queen"
	coherentJobSourceAutomatic = "automatic"
	coherentJobSourceSingle    = "single"

	coherentJobDecisionAccepted = "accepted"
	coherentJobDecisionRefused  = "refused"
)

// coherentJobProposal is the Queen's structured suggestion for assigning an
// ordered set of tasks to one worker. The runtime treats this as input, never
// authority: every field and dependency is checked before a job is accepted.
type coherentJobProposal struct {
	Name         string   `json:"name"`
	TaskIDs      []string `json:"task_ids"`
	OwnerCaste   string   `json:"owner_caste"`
	Relationship string   `json:"relationship"`
	Benefit      string   `json:"benefit"`
	OwnerReason  string   `json:"owner_reason,omitempty"`
}

// coherentJobTask is the selected-task seed consumed by the pure planner. Task
// keeps the complete contract for later brief composition; ID, TaskIndex,
// Caste, and DeclaredPaths are the already-resolved runtime facts used for
// grouping and deterministic ordering.
type coherentJobTask struct {
	Task          colony.Task `json:"-"`
	ID            string      `json:"id"`
	TaskIndex     int         `json:"task_index"`
	Caste         string      `json:"caste"`
	DeclaredPaths []string    `json:"declared_paths,omitempty"`
}

// coherentJob is one dependency-safe worker assignment produced by the
// planner. TaskIDs preserve execution order. DependsOn names other jobs, while
// Wave is the deterministic job-DAG wave derived from those edges.
type coherentJob struct {
	Name        string            `json:"name"`
	Tasks       []coherentJobTask `json:"-"`
	TaskIDs     []string          `json:"task_ids"`
	OwnerCaste  string            `json:"owner_caste"`
	JobReason   string            `json:"job_reason"`
	Source      string            `json:"job_source"`
	DependsOn   []string          `json:"depends_on,omitempty"`
	Wave        int               `json:"wave"`
	OwnerReason string            `json:"owner_reason,omitempty"`
}

// coherentJobDecision records what happened to one Queen proposal. A refused
// decision names the exact invalid relationship and the runtime-generated
// replacement jobs without mutating unrelated accepted proposals.
type coherentJobDecision struct {
	ProposalName        string   `json:"name"`
	Status              string   `json:"status"`
	JobName             string   `json:"job_name,omitempty"`
	JobReason           string   `json:"job_reason,omitempty"`
	JobSource           string   `json:"job_source,omitempty"`
	Reason              string   `json:"reason,omitempty"`
	OffendingTaskID     string   `json:"offending_task_id,omitempty"`
	DependencyID        string   `json:"dependency_id,omitempty"`
	ReplacementJobNames []string `json:"replacement_job_names,omitempty"`
}

// coherentJobPlan is the complete pure planning result. Waves contains job
// names in runnable order so later dispatch code never needs the fail-open
// taskWaves fallback.
type coherentJobPlan struct {
	Jobs      []coherentJob         `json:"jobs"`
	Decisions []coherentJobDecision `json:"decisions,omitempty"`
	Waves     [][]string            `json:"waves,omitempty"`
}

// planCoherentJobs validates the whole phase before considering a proposal,
// then accepts safe Queen jobs and returns only refused members to automatic
// planning. It is intentionally pure: it never writes state, creates an
// attempt, constructs a manifest, or touches a worktree.
func planCoherentJobs(phase colony.Phase, seeds []coherentJobTask, proposals []coherentJobProposal) (*coherentJobPlan, error) {
	if err := coherentJobGraphPreflight(phase); err != nil {
		return nil, err
	}

	normalizedSeeds, phaseTasks, err := normalizeCoherentJobSeeds(phase, seeds)
	if err != nil {
		return nil, err
	}
	plan := &coherentJobPlan{}
	if len(normalizedSeeds) == 0 {
		return plan, nil
	}

	seedByID := make(map[string]coherentJobTask, len(normalizedSeeds))
	for _, seed := range normalizedSeeds {
		seedByID[seed.ID] = seed
	}

	proposalNames := map[string]bool{}
	claimed := map[string]bool{}
	refusedMembers := make(map[string][]string)
	for _, raw := range proposals {
		proposal := normalizeCoherentJobProposal(raw)
		if proposal.Name == "" {
			return nil, fmt.Errorf("coherent job proposal name is required")
		}
		if proposalNames[proposal.Name] {
			return nil, fmt.Errorf("coherent job proposal name %q is duplicated", proposal.Name)
		}
		proposalNames[proposal.Name] = true

		decision, members, owner := validateCoherentJobProposal(proposal, seedByID, phaseTasks, claimed)
		if decision.Status == coherentJobDecisionRefused {
			plan.Decisions = append(plan.Decisions, decision)
			refusedMembers[proposal.Name] = members
			continue
		}

		job := newCoherentJob(
			proposal.Name,
			orderedCoherentJobTasks(proposal.TaskIDs, seedByID),
			owner,
			structuredCoherentJobReason(proposal.Relationship, proposal.Benefit),
			coherentJobSourceQueen,
		)
		job.OwnerReason = proposal.OwnerReason
		plan.Jobs = append(plan.Jobs, job)
		for _, taskID := range job.TaskIDs {
			claimed[taskID] = true
		}
		decision.JobName = job.Name
		decision.JobReason = job.JobReason
		decision.JobSource = job.Source
		plan.Decisions = append(plan.Decisions, decision)
	}

	remaining := make([]coherentJobTask, 0, len(normalizedSeeds)-len(claimed))
	for _, seed := range normalizedSeeds {
		if !claimed[seed.ID] {
			remaining = append(remaining, seed)
		}
	}
	plan.Jobs = append(plan.Jobs, planAutomaticCoherentJobs(remaining)...)
	sortCoherentJobsByPlanOrder(plan.Jobs)
	if err := populateCoherentJobDAG(plan); err != nil {
		return nil, err
	}
	attachCoherentJobReplacementNames(plan, refusedMembers)
	return plan, nil
}

func coherentJobGraphPreflight(phase colony.Phase) error {
	if err := colony.DetectCycles([]colony.Phase{phase}); err != nil {
		var cycleErr *colony.CycleError
		var missingErr *colony.MissingDepError
		switch {
		case errors.As(err, &cycleErr):
			return fmt.Errorf("cannot plan coherent jobs for phase %d: dependency cycle %s; repair depends_on in the plan", phase.ID, strings.Join(cycleErr.Tasks, " -> "))
		case errors.As(err, &missingErr):
			return fmt.Errorf("cannot plan coherent jobs for phase %d: task %s depends on missing task %s; repair depends_on in the plan", phase.ID, missingErr.Task, missingErr.MissingDep)
		default:
			return fmt.Errorf("cannot plan coherent jobs for phase %d: %w; repair depends_on in the plan", phase.ID, err)
		}
	}
	return nil
}

func normalizeCoherentJobSeeds(phase colony.Phase, seeds []coherentJobTask) ([]coherentJobTask, map[string]colony.Task, error) {
	phaseTasks := make(map[string]colony.Task, len(phase.Tasks))
	phaseIndexes := make(map[string]int, len(phase.Tasks))
	for idx, task := range phase.Tasks {
		id := buildTaskID(task, idx)
		if _, exists := phaseTasks[id]; exists {
			return nil, nil, fmt.Errorf("phase %d has duplicate task ID %q", phase.ID, id)
		}
		phaseTasks[id] = task
		phaseIndexes[id] = idx
	}

	validCastes := coherentJobValidCastes()
	seen := map[string]bool{}
	normalized := make([]coherentJobTask, 0, len(seeds))
	for _, seed := range seeds {
		id := strings.TrimSpace(seed.ID)
		if id == "" && seed.Task.ID != nil {
			id = strings.TrimSpace(*seed.Task.ID)
		}
		task, exists := phaseTasks[id]
		if !exists {
			return nil, nil, fmt.Errorf("selected coherent-job task %q is not in phase %d", id, phase.ID)
		}
		if seen[id] {
			return nil, nil, fmt.Errorf("selected coherent-job task %q is duplicated", id)
		}
		seen[id] = true

		caste := resolveCasteName(seed.Caste, validCastes)
		if !validCastes[caste] {
			return nil, nil, fmt.Errorf("selected coherent-job task %q has unknown owner caste %q", id, seed.Caste)
		}
		seed.ID = id
		seed.Task = task
		seed.TaskIndex = phaseIndexes[id]
		seed.Caste = caste
		if len(seed.DeclaredPaths) == 0 {
			seed.DeclaredPaths = declaredPathsForTask(task)
		} else {
			seed.DeclaredPaths = uniqueSortedStrings(seed.DeclaredPaths)
		}
		normalized = append(normalized, seed)
	}
	sort.SliceStable(normalized, func(i, j int) bool { return normalized[i].TaskIndex < normalized[j].TaskIndex })
	return normalized, phaseTasks, nil
}

func normalizeCoherentJobProposal(proposal coherentJobProposal) coherentJobProposal {
	proposal.Name = strings.TrimSpace(proposal.Name)
	proposal.OwnerCaste = strings.TrimSpace(proposal.OwnerCaste)
	proposal.Relationship = strings.TrimSpace(proposal.Relationship)
	proposal.Benefit = strings.TrimSpace(proposal.Benefit)
	proposal.OwnerReason = strings.TrimSpace(proposal.OwnerReason)
	for idx := range proposal.TaskIDs {
		proposal.TaskIDs[idx] = strings.TrimSpace(proposal.TaskIDs[idx])
	}
	return proposal
}

func validateCoherentJobProposal(
	proposal coherentJobProposal,
	seedByID map[string]coherentJobTask,
	phaseTasks map[string]colony.Task,
	claimed map[string]bool,
) (coherentJobDecision, []string, string) {
	decision := coherentJobDecision{ProposalName: proposal.Name, Status: coherentJobDecisionAccepted}
	refuse := func(reason, offendingTaskID, dependencyID string, members []string) (coherentJobDecision, []string, string) {
		decision.Status = coherentJobDecisionRefused
		decision.Reason = fmt.Sprintf("proposal %q refused: %s", proposal.Name, reason)
		decision.OffendingTaskID = offendingTaskID
		decision.DependencyID = dependencyID
		return decision, members, ""
	}

	if len(proposal.TaskIDs) == 0 {
		return refuse("task_ids must name at least one selected task", "", "", nil)
	}
	if proposal.Relationship == "" {
		return refuse("relationship is required", "", "", nil)
	}
	if proposal.Benefit == "" {
		return refuse("benefit is required", "", "", nil)
	}

	validCastes := coherentJobValidCastes()
	owner := resolveCasteName(proposal.OwnerCaste, validCastes)
	if !validCastes[owner] {
		return refuse(fmt.Sprintf("owner_caste %q is not a dispatchable caste", proposal.OwnerCaste), "", "", nil)
	}

	members := make([]string, 0, len(proposal.TaskIDs))
	memberSet := make(map[string]bool, len(proposal.TaskIDs))
	memberCastes := map[string]bool{}
	for _, taskID := range proposal.TaskIDs {
		if taskID == "" {
			return refuse("task_ids contains an empty task ID", "", "", members)
		}
		if memberSet[taskID] {
			return refuse(fmt.Sprintf("task %s appears more than once", taskID), taskID, "", members)
		}
		memberSet[taskID] = true
		seed, selected := seedByID[taskID]
		if !selected {
			if _, exists := phaseTasks[taskID]; exists {
				return refuse(fmt.Sprintf("task %s is outside the selected task scope", taskID), taskID, "", members)
			}
			return refuse(fmt.Sprintf("task %s does not exist in the phase", taskID), taskID, "", members)
		}
		if claimed[taskID] {
			return refuse(fmt.Sprintf("task %s is already owned by an accepted proposal", taskID), taskID, "", members)
		}
		members = append(members, taskID)
		memberCastes[seed.Caste] = true
	}

	if !memberCastes[owner] {
		return refuse(fmt.Sprintf("owner_caste %s is not suitable because none of the proposed tasks resolves to it", owner), "", "", members)
	}
	if len(memberCastes) > 1 && proposal.OwnerReason == "" {
		return refuse("cross-caste grouping requires owner_reason for the one worker carrying the whole job", "", "", members)
	}

	seen := map[string]bool{}
	for _, taskID := range proposal.TaskIDs {
		task := seedByID[taskID].Task
		for _, dependencyID := range task.DependsOn {
			dependencyID = strings.TrimSpace(dependencyID)
			if memberSet[dependencyID] && !seen[dependencyID] {
				return refuse(
					fmt.Sprintf("task %s appears before its unmet dependency %s", taskID, dependencyID),
					taskID,
					dependencyID,
					members,
				)
			}
			if !memberSet[dependencyID] && dependencyReachesAnyCoherentJobMember(dependencyID, memberSet, phaseTasks, map[string]bool{}) {
				return refuse(
					fmt.Sprintf("task %s cannot run in this job before external dependency %s, which itself depends on a member of the proposal", taskID, dependencyID),
					taskID,
					dependencyID,
					members,
				)
			}
		}
		seen[taskID] = true
	}
	decision.Reason = fmt.Sprintf("proposal %q accepted: %s", proposal.Name, structuredCoherentJobReason(proposal.Relationship, proposal.Benefit))
	return decision, members, owner
}

func dependencyReachesAnyCoherentJobMember(id string, members map[string]bool, tasks map[string]colony.Task, visiting map[string]bool) bool {
	if members[id] {
		return true
	}
	if visiting[id] {
		return false
	}
	visiting[id] = true
	defer delete(visiting, id)
	task, ok := tasks[id]
	if !ok {
		return false
	}
	for _, dependencyID := range task.DependsOn {
		if dependencyReachesAnyCoherentJobMember(strings.TrimSpace(dependencyID), members, tasks, visiting) {
			return true
		}
	}
	return false
}

func coherentJobValidCastes() map[string]bool {
	valid := make(map[string]bool, len(casteRelevanceRegistry))
	for _, profile := range casteRelevanceRegistry {
		valid[profile.Caste] = true
	}
	return valid
}

func orderedCoherentJobTasks(ids []string, byID map[string]coherentJobTask) []coherentJobTask {
	tasks := make([]coherentJobTask, 0, len(ids))
	for _, id := range ids {
		tasks = append(tasks, byID[id])
	}
	return tasks
}

func structuredCoherentJobReason(relationship, benefit string) string {
	return strings.TrimSpace(relationship) + ", so " + strings.TrimSpace(benefit)
}

func newCoherentJob(name string, tasks []coherentJobTask, owner, reason, source string) coherentJob {
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	return coherentJob{
		Name:       name,
		Tasks:      tasks,
		TaskIDs:    ids,
		OwnerCaste: owner,
		JobReason:  reason,
		Source:     source,
	}
}

// planAutomaticCoherentJobs is deliberately conservative in Task 1: until the
// meaningful-path and component rules land, each unclaimed task remains one
// job. Task 2 replaces this body with dependency/path components while keeping
// the proposal and graph contracts unchanged.
func planAutomaticCoherentJobs(tasks []coherentJobTask) []coherentJob {
	jobs := make([]coherentJob, 0, len(tasks))
	for _, task := range tasks {
		relationship := fmt.Sprintf("task %s is not part of an accepted grouped proposal", task.ID)
		benefit := fmt.Sprintf("one %s owns its implementation without crossing a caste boundary", task.Caste)
		jobs = append(jobs, newCoherentJob(
			"single-"+task.ID,
			[]coherentJobTask{task},
			task.Caste,
			structuredCoherentJobReason(relationship, benefit),
			coherentJobSourceSingle,
		))
	}
	return jobs
}

func sortCoherentJobsByPlanOrder(jobs []coherentJob) {
	sort.SliceStable(jobs, func(i, j int) bool {
		left := coherentJobFirstTaskIndex(jobs[i])
		right := coherentJobFirstTaskIndex(jobs[j])
		if left != right {
			return left < right
		}
		return jobs[i].Name < jobs[j].Name
	})
}

func coherentJobFirstTaskIndex(job coherentJob) int {
	first := int(^uint(0) >> 1)
	for _, task := range job.Tasks {
		if task.TaskIndex < first {
			first = task.TaskIndex
		}
	}
	return first
}

func populateCoherentJobDAG(plan *coherentJobPlan) error {
	jobByTask := make(map[string]int)
	for jobIndex, job := range plan.Jobs {
		for _, taskID := range job.TaskIDs {
			if _, exists := jobByTask[taskID]; exists {
				return fmt.Errorf("coherent job task %s is assigned more than once", taskID)
			}
			jobByTask[taskID] = jobIndex
		}
	}

	for jobIndex := range plan.Jobs {
		seen := map[string]bool{}
		var dependencies []string
		for _, task := range plan.Jobs[jobIndex].Tasks {
			for _, dependencyID := range task.Task.DependsOn {
				dependencyJobIndex, selected := jobByTask[strings.TrimSpace(dependencyID)]
				if !selected || dependencyJobIndex == jobIndex {
					continue
				}
				dependencyName := plan.Jobs[dependencyJobIndex].Name
				if !seen[dependencyName] {
					seen[dependencyName] = true
					dependencies = append(dependencies, dependencyName)
				}
			}
		}
		plan.Jobs[jobIndex].DependsOn = dependencies
	}

	completed := map[string]bool{}
	for len(completed) < len(plan.Jobs) {
		var wave []string
		for idx := range plan.Jobs {
			job := &plan.Jobs[idx]
			if completed[job.Name] || !coherentJobDependenciesSatisfied(job.DependsOn, completed) {
				continue
			}
			wave = append(wave, job.Name)
		}
		if len(wave) == 0 {
			remaining := make([]string, 0, len(plan.Jobs)-len(completed))
			for _, job := range plan.Jobs {
				if !completed[job.Name] {
					remaining = append(remaining, job.Name)
				}
			}
			return fmt.Errorf("coherent job dependency cycle among %s; repair the proposed grouping or depends_on in the plan", strings.Join(remaining, ", "))
		}
		waveNumber := len(plan.Waves) + 1
		plan.Waves = append(plan.Waves, wave)
		for _, name := range wave {
			completed[name] = true
			for idx := range plan.Jobs {
				if plan.Jobs[idx].Name == name {
					plan.Jobs[idx].Wave = waveNumber
					break
				}
			}
		}
	}
	return nil
}

func coherentJobDependenciesSatisfied(dependencies []string, completed map[string]bool) bool {
	for _, dependency := range dependencies {
		if !completed[dependency] {
			return false
		}
	}
	return true
}

func attachCoherentJobReplacementNames(plan *coherentJobPlan, refusedMembers map[string][]string) {
	for decisionIndex := range plan.Decisions {
		decision := &plan.Decisions[decisionIndex]
		if decision.Status != coherentJobDecisionRefused {
			continue
		}
		members := stringSet(refusedMembers[decision.ProposalName])
		seenJobs := map[string]bool{}
		for _, job := range plan.Jobs {
			if job.Source == coherentJobSourceQueen {
				continue
			}
			for _, taskID := range job.TaskIDs {
				if members[taskID] && !seenJobs[job.Name] {
					seenJobs[job.Name] = true
					decision.ReplacementJobNames = append(decision.ReplacementJobNames, job.Name)
					break
				}
			}
		}
	}
}
