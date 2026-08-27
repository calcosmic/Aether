package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// coherentJobSourceRetry marks a coherent job produced by planCoherentJobRetry
// rather than a Queen proposal, automatic grouping, or a single unmerged task
// (D-10, 195-CONTEXT.md). It is used only for display/diagnostics; the
// planner itself still routes a retry job through the same proposal-shaped
// validation every other job goes through.
const coherentJobSourceRetry = "retry"

// planCoherentJobRetry derives a new dependency-safe coherent job containing
// ONLY a parent job's unfinished (uncredited) tasks, per D-10:
// "Recovery creates a new dependency-safe job containing only the unfinished
// tasks. Revalidate their dependencies against the credited tasks, append a
// new attempt linked to the original grouped job, and never overwrite the
// first worker's receipt or ask a new worker to redo proven work."
//
// It is intentionally pure -- like planCoherentJobs, it never writes state,
// creates an attempt, or touches a worktree. coveredTaskIDs is the parent
// dispatch's own ordered task list (dispatchCoveredTaskIDs); creditedTaskIDs
// is the subset already proven complete (completedBuildTaskIDs' output for
// that one dispatch). Credited tasks are discounted from cycle/order
// validation entirely -- their own dependencies are satisfied by definition
// (they already finished), so only the REMAINING subgraph is checked, using
// the canonical coherentJobGraphPreflight/planCoherentJobs machinery rather
// than a second hand-rolled graph walk.
//
// Returns (nil, nil) when every covered task was already credited (nothing to
// retry). Returns a named error -- never a silent partial plan -- when the
// unfinished subgraph cannot be dependency-ordered, whether because a real
// cycle exists among the phase's tasks or because an unfinished task names a
// dependency that does not exist anywhere in the phase.
func planCoherentJobRetry(phase colony.Phase, coveredTaskIDs []string, creditedTaskIDs []string, ownerCaste, parentJobName string) (*coherentJob, error) {
	covered := make([]string, 0, len(coveredTaskIDs))
	for _, id := range coveredTaskIDs {
		if id = strings.TrimSpace(id); id != "" {
			covered = append(covered, id)
		}
	}
	if len(covered) == 0 {
		return nil, fmt.Errorf("coherent job retry requires at least one covered task")
	}

	credited := stringSet(creditedTaskIDs)
	unfinished := make([]string, 0, len(covered))
	for _, id := range covered {
		if !credited[id] {
			unfinished = append(unfinished, id)
		}
	}
	if len(unfinished) == 0 {
		// D-10: a fully credited job needs no recovery job at all.
		return nil, nil
	}

	tasksByID := make(map[string]colony.Task, len(phase.Tasks))
	for idx := range phase.Tasks {
		tasksByID[buildTaskID(phase.Tasks[idx], idx)] = phase.Tasks[idx]
	}
	seeds := make([]coherentJobTask, 0, len(unfinished))
	for _, id := range unfinished {
		task, ok := tasksByID[id]
		if !ok {
			return nil, fmt.Errorf("coherent job retry: unfinished task %s is not in phase %d; repair the completed job's covered task list before retrying", id, phase.ID)
		}
		seeds = append(seeds, coherentJobTask{ID: id, Task: task, Caste: ownerCaste})
	}

	trimmedParentName := strings.TrimSpace(parentJobName)
	retryName := trimmedParentName + "-retry"
	if trimmedParentName == "" {
		retryName = "retry-" + unfinished[0]
	}
	relationship := fmt.Sprintf(
		"tasks %s remain unfinished from job %s",
		strings.Join(unfinished, ", "),
		firstNonEmpty(trimmedParentName, "the original attempt"),
	)
	benefit := "the credited tasks' proof stays intact and only the unfinished work is redispatched to one worker"
	proposal := coherentJobProposal{
		Name:         retryName,
		TaskIDs:      unfinished,
		OwnerCaste:   ownerCaste,
		Relationship: relationship,
		Benefit:      benefit,
	}

	// coherentJobGraphPreflight (inside planCoherentJobs) validates the WHOLE
	// phase's dependency graph, which already proves any genuine cycle or
	// missing dependency by name -- this call never needs a second,
	// retry-specific graph walk. A credited task's own dependencies are, by
	// construction, already satisfied (it finished), so validateCoherentJobProposal
	// treating an external (non-member) dependency as satisfied whenever that
	// dependency does not itself reach back into the retry's member set is
	// exactly D-10's "credited tasks treated as satisfied" contract.
	plan, err := planCoherentJobs(phase, seeds, []coherentJobProposal{proposal})
	if err != nil {
		return nil, fmt.Errorf("coherent job retry: unfinished task path is not dependency-safe: %w", err)
	}
	for _, decision := range plan.Decisions {
		if decision.Status == coherentJobDecisionRefused {
			return nil, fmt.Errorf("coherent job retry: %s", decision.Reason)
		}
	}
	if len(plan.Jobs) != 1 {
		return nil, fmt.Errorf("coherent job retry: expected exactly one retry job for unfinished tasks %s, got %d", strings.Join(unfinished, ", "), len(plan.Jobs))
	}
	job := plan.Jobs[0]
	job.Source = coherentJobSourceRetry
	return &job, nil
}

// partialBuildRetryOutcome is the structured D-10 recovery info both the
// direct/native and external/wrapper build lanes surface after a validated
// partial completion: which attempt is the parent, which new attempt now
// owns the unfinished work, exactly what remains, and the exact command that
// redispatches it.
type partialBuildRetryOutcome struct {
	ParentAttemptID   string   `json:"parent_attempt_id"`
	RetryAttemptID    string   `json:"retry_attempt_id"`
	RetryAttemptPath  string   `json:"retry_attempt_path"`
	UnfinishedTaskIDs []string `json:"unfinished_task_ids"`
	RedispatchCommand string   `json:"redispatch_command"`
}

// reconcilePartialBuildRetry is the shared D-10 partial-terminal path both
// build lanes call once a dispatch's own receipts have been resolved
// (resolveCoherentJobDispatchReceipts). For every dispatch that covered more
// tasks than it credited, it derives an unfinished-only retry job
// (planCoherentJobRetry) and appends ONE new, parent-linked attempt covering
// every such job -- never touching the parent attempt's own file.
//
// Returns (nil, nil) when nothing in dispatches needs a retry: either every
// dispatch fully succeeded, or every non-succeeding dispatch credited nothing
// at all (in which case there is no validated partial proof to recover --
// the caller's existing whole-rollback/phantom-build-rejection path is the
// correct one, unchanged).
//
// Idempotent: a second call for the same parentAttemptID finds the retry
// attempt it already created (by scanning this phase's attempt journal for a
// record whose ParentAttemptID matches) and returns that, rather than
// creating a duplicate child or touching the existing one.
func reconcilePartialBuildRetry(state colony.ColonyState, phaseNum int, phase colony.Phase, parentAttemptID string, retryStartedAt time.Time, dispatches []codexBuildDispatch) (*partialBuildRetryOutcome, error) {
	parentAttemptID = strings.TrimSpace(parentAttemptID)
	if parentAttemptID == "" {
		return nil, fmt.Errorf("coherent job retry requires a parent attempt id")
	}

	var retryJobs []coherentJob
	var retryDispatches []codexBuildDispatch
	var allUnfinished []string
	for _, dispatch := range dispatches {
		covered := dispatchCoveredTaskIDs(dispatch)
		if len(covered) == 0 {
			continue
		}
		creditedSet := completedBuildTaskIDs([]codexBuildDispatch{dispatch})
		if len(creditedSet) == len(covered) {
			// Fully credited (whole success, or every covered task proven by
			// receipt) -- nothing to retry for this dispatch.
			continue
		}
		if len(creditedSet) == 0 {
			// No validated partial proof at all for this dispatch -- D-10
			// recovery only ever follows ACCEPTED credit (195-CONTEXT.md:
			// "retry follows accepted credit, never precedes it"). The
			// caller's existing whole-failure handling is correct here.
			continue
		}
		creditedIDs := make([]string, 0, len(creditedSet))
		for id := range creditedSet {
			creditedIDs = append(creditedIDs, id)
		}

		job, err := planCoherentJobRetry(phase, covered, creditedIDs, dispatch.Caste, dispatch.JobName)
		if err != nil {
			return nil, err
		}
		if job == nil {
			continue
		}
		retryJobs = append(retryJobs, *job)
		allUnfinished = append(allUnfinished, job.TaskIDs...)

		declaredPaths := make([]string, 0, len(job.Tasks))
		for _, seed := range job.Tasks {
			declaredPaths = append(declaredPaths, seed.DeclaredPaths...)
		}
		retryDispatches = append(retryDispatches, codexBuildDispatch{
			Stage:          "wave",
			Caste:          job.OwnerCaste,
			Name:           job.Name,
			Task:           job.JobReason,
			Status:         "planned",
			TaskID:         job.TaskIDs[0],
			CoveredTaskIDs: append([]string{}, job.TaskIDs...),
			JobName:        job.Name,
			JobReason:      job.JobReason,
			JobSource:      job.Source,
			DeclaredPaths:  uniqueSortedStrings(declaredPaths),
		})
	}

	if len(retryJobs) == 0 {
		return nil, nil
	}

	redispatchCommand := buildForceRedispatchCommand(phaseNum)

	// Idempotency: a retry attempt for this exact parent may already exist
	// (a second finalize/dispatch pass over the same partial outcome). Never
	// create a second child for the same parent.
	if existingRel, existing, ok := findExistingBuildAttemptRetry(phaseNum, parentAttemptID); ok {
		return &partialBuildRetryOutcome{
			ParentAttemptID:   parentAttemptID,
			RetryAttemptID:    existing.ID,
			RetryAttemptPath:  displayDataPath(existingRel),
			UnfinishedTaskIDs: uniqueSortedStrings(allUnfinished),
			RedispatchCommand: redispatchCommand,
		}, nil
	}

	childRel, err := beginChildBuildAttempt(
		state, phaseNum, phase, retryStartedAt,
		parentAttemptID, retryJobs[0].Name,
		uniqueSortedStrings(allUnfinished),
		"", "", "",
		buildExecutionOwner("real", false),
		retryDispatches,
	)
	if err != nil {
		return nil, fmt.Errorf("coherent job retry: create recovery attempt: %w", err)
	}
	childID := strings.TrimSuffix(filepath.Base(childRel), filepath.Ext(childRel))
	return &partialBuildRetryOutcome{
		ParentAttemptID:   parentAttemptID,
		RetryAttemptID:    childID,
		RetryAttemptPath:  displayDataPath(childRel),
		UnfinishedTaskIDs: uniqueSortedStrings(allUnfinished),
		RedispatchCommand: redispatchCommand,
	}, nil
}

// findExistingBuildAttemptRetry scans this phase's attempt journal
// (listBuildAttemptsForPhase) for a record already linked to parentAttemptID,
// so reconcilePartialBuildRetry never creates a duplicate child.
func findExistingBuildAttemptRetry(phaseNum int, parentAttemptID string) (string, buildAttemptRecord, bool) {
	parentAttemptID = strings.TrimSpace(parentAttemptID)
	if parentAttemptID == "" {
		return "", buildAttemptRecord{}, false
	}
	for _, record := range listBuildAttemptsForPhase(phaseNum) {
		if strings.TrimSpace(record.ParentAttemptID) == parentAttemptID {
			return buildAttemptPathForID(phaseNum, record.ID), record, true
		}
	}
	return "", buildAttemptRecord{}, false
}

// buildAttemptPathForID reconstructs an attempt's own relative journal path
// from its phase and ID -- the same naming scheme beginBuildAttempt uses
// (build/phase-<N>/attempts/<id>.json).
func buildAttemptPathForID(phaseNum int, attemptID string) string {
	return filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum), "attempts", attemptID+".json"))
}

// commitPartialBuildCredit is the direct/native lane's D-10 partial-terminal
// commit: it mutates ONLY the freshly-read on-disk colony state's task
// statuses (reconcileCompletedBuildTasks), the same reader every whole-build
// commit uses, and deliberately leaves state.State exactly as
// applyCodexBuildState already projected it (EXECUTING, phase in_progress)
// rather than advancing to StateBUILT -- a partial credit is, by definition,
// not a finished build. Mirrors the concurrency guard the existing
// EXECUTING->BUILT commit uses (validateRuntimeStateStillCurrent), so a
// concurrent pause or force-redispatch is never silently overwritten.
func commitPartialBuildCredit(phaseNum int, startedAt time.Time, dispatches []codexBuildDispatch) (colony.ColonyState, error) {
	var committedState colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &committedState, func() error {
		if err := validateRuntimeStateStillCurrent(committedState, phaseNum, &startedAt, colony.StateEXECUTING); err != nil {
			return err
		}
		reconcileCompletedBuildTasks(&committedState, phaseNum, dispatches)
		committedState.Events = append(trimmedEvents(committedState.Events),
			fmt.Sprintf("%s|build_partial_credit|build|Phase %d partial credit recorded; a D-10 recovery job covers the unfinished tasks", startedAt.Format(time.RFC3339), phaseNum),
		)
		return nil
	}); err != nil {
		return colony.ColonyState{}, fmt.Errorf("failed to save partial build credit: %w", err)
	}
	return committedState, nil
}
