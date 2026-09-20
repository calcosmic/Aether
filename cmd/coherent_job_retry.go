package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
func reconcilePartialBuildRetry(state colony.ColonyState, phaseNum int, phase colony.Phase, parentAttemptID string, retryStartedAt time.Time, dispatches []codexBuildDispatch, startOptions ...buildStartOptions) (*partialBuildRetryOutcome, error) {
	plan, err := planPartialBuildRetry(phaseNum, phase, dispatches)
	if err != nil || plan == nil {
		return nil, err
	}
	return commitPartialBuildRetryPlan(state, phaseNum, phase, parentAttemptID, retryStartedAt, plan, startOptions...)
}

// partialBuildRetryPlan is the pure half of reconcilePartialBuildRetry: what
// the D-10 recovery would contain, worked out without writing anything.
//
// It exists so a caller can decide whether partial credit is worth committing
// BEFORE any attempt record is created (WR-10, 195-REVIEW.md). The direct lane
// used to create the recovery record first and commit the credit second, so a
// refused commit -- a concurrent pause, a store write error -- rolled the
// credit back and left an orphan record describing work nothing had recorded
// as partially done.
type partialBuildRetryPlan struct {
	Jobs              []coherentJob
	Dispatches        []codexBuildDispatch
	UnfinishedTaskIDs []string
	RedispatchCommand string
}

// planPartialBuildRetry derives the D-10 recovery plan for dispatches without
// writing state, creating an attempt, or touching a worktree. Returns
// (nil, nil) when nothing needs a retry.
func planPartialBuildRetry(phaseNum int, phase colony.Phase, dispatches []codexBuildDispatch) (*partialBuildRetryPlan, error) {
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

	return &partialBuildRetryPlan{
		Jobs:              retryJobs,
		Dispatches:        retryDispatches,
		UnfinishedTaskIDs: uniqueSortedStrings(allUnfinished),
		// CR-04 (195-REVIEW.md): the command handed to the owner must carry the
		// retry job's own task IDs. A bare `aether build <N> --force` re-plans the
		// phase from its full task list with no filter, so every task this build
		// already proved would be dispatched again -- exactly what D-10, the three
		// build wrapper copies, the command guide and CLAUDE.md all promise never
		// happens.
		RedispatchCommand: buildUnfinishedRetryRedispatchCommand(phaseNum, allUnfinished),
	}, nil
}

// commitPartialBuildRetryPlan is the writing half: it appends ONE new,
// parent-linked attempt covering every retry job in plan, never touching the
// parent attempt's own file.
//
// Idempotent: a second call for the same parentAttemptID finds the retry
// attempt it already created (by scanning this phase's attempt journal for a
// record whose ParentAttemptID matches) and returns that, rather than
// creating a duplicate child or touching the existing one.
func commitPartialBuildRetryPlan(state colony.ColonyState, phaseNum int, phase colony.Phase, parentAttemptID string, retryStartedAt time.Time, plan *partialBuildRetryPlan, startOptions ...buildStartOptions) (*partialBuildRetryOutcome, error) {
	parentAttemptID = strings.TrimSpace(parentAttemptID)
	if parentAttemptID == "" {
		return nil, fmt.Errorf("coherent job retry requires a parent attempt id")
	}
	if plan == nil || len(plan.Jobs) == 0 {
		return nil, nil
	}

	if existing, found, err := verifiedPartialBuildRetryOutcome(phaseNum, parentAttemptID, plan); found {
		return existing, err
	}

	if len(startOptions) > 1 {
		return nil, fmt.Errorf("coherent job retry accepts at most one build-start option set")
	}
	options := buildStartOptions{}
	if len(startOptions) == 1 {
		options = startOptions[0]
	}

	// Repository-session loading normalizes legacy-compatible execution facts
	// before it checks request.StateSHA256. Match that canonical shape here so
	// a legitimate partial-state projection does not manufacture a stale
	// baseline solely because EvidencePolicy was omitted on disk.
	state = normalizeLegacyColonyState(state)
	state, authority, err := resolveCodexBuildPlanAuthority(buildAttemptWorkspaceRoot(), state)
	if err != nil {
		return nil, fmt.Errorf("coherent job retry: resolve plan authority: %w", err)
	}
	request, err := newBuildStartRequest(
		buildAttemptWorkspaceRoot(), buildStartCoherentChildRetry, state, authority,
		phaseNum, append([]string{}, plan.UnfinishedTaskIDs...),
		buildExecutionOwner("real", false), "coherent-child-retry", retryStartedAt,
		plan.Dispatches, buildStartEffects{
			ParentAttemptID: parentAttemptID,
			ParentJobName:   plan.Jobs[0].Name,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("coherent job retry: prepare recovery attempt: %w", err)
	}
	receipt, err := commitBuildStart(buildAttemptWorkspaceRoot(), request, options)
	if err != nil {
		return nil, fmt.Errorf("coherent job retry: commit recovery attempt: %w", err)
	}
	return &partialBuildRetryOutcome{
		ParentAttemptID:   parentAttemptID,
		RetryAttemptID:    receipt.AttemptID,
		RetryAttemptPath:  displayDataPath(receipt.AttemptPath),
		UnfinishedTaskIDs: append([]string{}, plan.UnfinishedTaskIDs...),
		RedispatchCommand: plan.RedispatchCommand,
	}, nil
}

// verifiedPartialBuildRetryOutcome reads an existing child as untrusted
// durable evidence. It accepts exactly one canonical, still-prepared child
// whose receipt binds its current bytes, parent, unfinished task set,
// dispatch plan, and exact redispatch command. found distinguishes "there is
// no child yet" (the first commit may create one) from malformed or duplicate
// children (fail closed; never mint a replacement beside suspicious state).
func verifiedPartialBuildRetryOutcome(phaseNum int, parentAttemptID string, plan *partialBuildRetryPlan) (*partialBuildRetryOutcome, bool, error) {
	parentAttemptID = strings.TrimSpace(parentAttemptID)
	if parentAttemptID == "" || plan == nil {
		return nil, false, fmt.Errorf("coherent job retry evidence requires a parent and recovery plan")
	}
	type childEvidence struct {
		rel    string
		record buildAttemptRecord
	}
	var matches []childEvidence
	for _, record := range listBuildAttemptsForPhase(phaseNum) {
		if strings.TrimSpace(record.ParentAttemptID) == parentAttemptID {
			matches = append(matches, childEvidence{rel: buildAttemptPathForID(phaseNum, record.ID), record: record})
		}
	}
	if len(matches) == 0 {
		return nil, false, nil
	}
	if len(matches) != 1 {
		return nil, true, fmt.Errorf("partial recovery for parent %s has %d child attempts; expected exactly one", parentAttemptID, len(matches))
	}
	child := matches[0]
	record := child.record
	if record.SchemaVersion != buildAttemptSchemaVersion || record.Phase != phaseNum || !validBuildAttemptID(record.ID) ||
		strings.TrimSpace(record.ParentAttemptID) != parentAttemptID {
		return nil, true, fmt.Errorf("partial recovery child for parent %s has conflicting identity", parentAttemptID)
	}
	if record.Status != buildAttemptPrepared {
		return nil, true, fmt.Errorf("partial recovery attempt %s is exhausted with status %q", record.ID, record.Status)
	}
	if record.ExecutionOwner != buildExecutionOwner("real", false) || record.DispatchMode != "coherent-child-retry" {
		return nil, true, fmt.Errorf("partial recovery attempt %s has conflicting execution identity", record.ID)
	}
	wantTasks := uniqueSortedStrings(plan.UnfinishedTaskIDs)
	if !reflect.DeepEqual(record.SelectedTasks, wantTasks) || !reflect.DeepEqual(record.RecoveryTaskIDs, wantTasks) {
		return nil, true, fmt.Errorf("partial recovery attempt %s has conflicting unfinished task evidence", record.ID)
	}
	if record.RecoveryCommand != plan.RedispatchCommand || strings.TrimSpace(record.RecoveryCommand) == "" {
		return nil, true, fmt.Errorf("partial recovery attempt %s has conflicting redispatch command evidence", record.ID)
	}
	recordDispatchHash, recordDispatchErr := jsonSHA256(record.Dispatches)
	planDispatchHash, planDispatchErr := jsonSHA256(plan.Dispatches)
	if recordDispatchErr != nil || planDispatchErr != nil || recordDispatchHash != planDispatchHash {
		return nil, true, fmt.Errorf("partial recovery attempt %s has conflicting dispatch evidence", record.ID)
	}
	wantParentJob := ""
	if len(plan.Jobs) > 0 {
		wantParentJob = plan.Jobs[0].Name
	}
	if record.ParentJobName != wantParentJob {
		return nil, true, fmt.Errorf("partial recovery attempt %s has conflicting parent-job evidence", record.ID)
	}

	receiptRel := buildStartReceiptPath(phaseNum, record.ID)
	var receipt buildStartReceipt
	if receiptRel == "" || store.LoadJSON(receiptRel, &receipt) != nil {
		return nil, true, fmt.Errorf("partial recovery attempt %s is missing its canonical build-start receipt", record.ID)
	}
	if receipt.SchemaVersion != buildStartSchemaVersion || receipt.Path != receiptRel || receipt.Phase != phaseNum ||
		receipt.AttemptID != record.ID || receipt.AttemptPath != child.rel || receipt.GeneratedAt != record.StartedAt ||
		len(receipt.RequestSHA256) != 64 || receipt.TransactionID != "build-start-"+receipt.RequestSHA256[:24] ||
		receipt.ID != "build-start-receipt-"+receipt.RequestSHA256[:24] {
		return nil, true, fmt.Errorf("partial recovery attempt %s has conflicting canonical build-start receipt identity", record.ID)
	}
	payload := receipt
	payload.ContentHash = ""
	hash, err := jsonSHA256(payload)
	if err != nil || hash != receipt.ContentHash {
		return nil, true, fmt.Errorf("partial recovery attempt %s has invalid canonical build-start receipt content", record.ID)
	}
	requestShape := buildStartRequest{
		Phase: phaseNum, AttemptID: record.ID,
		Effects: buildStartEffects{ParentAttemptID: parentAttemptID, ParentJobName: record.ParentJobName},
	}
	if err := validateBuildStartReceiptTargets(receipt.Targets, requestShape); err != nil {
		return nil, true, fmt.Errorf("partial recovery attempt %s has invalid canonical build-start receipt targets: %w", record.ID, err)
	}
	childBytes, err := os.ReadFile(filepath.Join(store.BasePath(), filepath.FromSlash(child.rel)))
	if err != nil {
		return nil, true, fmt.Errorf("read partial recovery attempt %s: %w", record.ID, err)
	}
	wantDigest := lifecycleDigest(childBytes)
	bound := false
	for _, target := range receipt.Targets {
		if target.Path == child.rel {
			bound = target.Action == string(lifecycleTransactionWrite) && target.SHA256 == wantDigest
		}
	}
	if !bound {
		return nil, true, fmt.Errorf("partial recovery attempt %s bytes do not match its canonical build-start receipt", record.ID)
	}

	return &partialBuildRetryOutcome{
		ParentAttemptID:   parentAttemptID,
		RetryAttemptID:    record.ID,
		RetryAttemptPath:  displayDataPath(child.rel),
		UnfinishedTaskIDs: append([]string{}, record.RecoveryTaskIDs...),
		RedispatchCommand: record.RecoveryCommand,
	}, true, nil
}

// verifiedPartialBuildRetryPlanFromParent checks that the partial parent's
// durable terminal evidence names the same unfinished-only plan and exact
// command the pure planner derives. The returned command comes from the
// journal; derivation is used only as a tamper check.
func verifiedPartialBuildRetryPlanFromParent(phaseNum int, phase colony.Phase, parent buildAttemptRecord) (*partialBuildRetryPlan, error) {
	if parent.Phase != phaseNum || parent.Status != buildAttemptPartial {
		return nil, fmt.Errorf("build attempt %s is not a committed partial parent", parent.ID)
	}
	dispatches := restatePartialCreditFromCommittedState(phase, parent.Dispatches)
	plan, err := planPartialBuildRetry(phaseNum, phase, dispatches)
	if err != nil {
		return nil, fmt.Errorf("reconstruct partial recovery plan: %w", err)
	}
	if plan == nil || len(plan.UnfinishedTaskIDs) == 0 || strings.TrimSpace(plan.RedispatchCommand) == "" {
		return nil, fmt.Errorf("build attempt %s has no reconstructable partial recovery plan", parent.ID)
	}
	wantTasks := uniqueSortedStrings(plan.UnfinishedTaskIDs)
	if !reflect.DeepEqual(parent.RecoveryTaskIDs, wantTasks) || parent.RecoveryCommand != plan.RedispatchCommand {
		return nil, fmt.Errorf("build attempt %s has conflicting durable partial recovery evidence", parent.ID)
	}
	// Project the persisted values after validation. Replay callers must not
	// synthesize an owner command from a changed phase or free-form summary.
	plan.UnfinishedTaskIDs = append([]string{}, parent.RecoveryTaskIDs...)
	plan.RedispatchCommand = parent.RecoveryCommand
	return plan, nil
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
