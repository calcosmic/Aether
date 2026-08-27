package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// Named refusal rules for the shared receipt trust boundary (D-08, D-09;
// 195-RESEARCH.md Pattern 4). Every rejected task_receipts entry is refused
// by exactly one of these, never a silent drop.
const (
	violationRuleTaskReceiptIDRequired          = "task_receipt.task_id_required"
	violationRuleTaskReceiptDuplicate           = "task_receipt.duplicate"
	violationRuleTaskReceiptUnknown             = "task_receipt.unknown_task"
	violationRuleTaskReceiptOutOfScope          = "task_receipt.out_of_scope"
	violationRuleTaskReceiptStatusInvalid       = "task_receipt.status_invalid"
	violationRuleTaskReceiptSummaryRequired     = "task_receipt.summary_required"
	violationRuleTaskReceiptHandoffInvalid      = "task_receipt.handoff_invalid"
	violationRuleTaskReceiptUnevidenced         = "task_receipt.unevidenced"
	violationRuleTaskReceiptPathLaundered       = "task_receipt.path_laundered"
	violationRuleTaskReceiptPathOutOfClaims     = "task_receipt.path_not_in_aggregate_claims"
	violationRuleTaskReceiptRequirementMismatch = "task_receipt.requirement_mismatch"
	violationRuleTaskReceiptRootEvidenceMissing = "task_receipt.root_evidence_missing"
)

// coherentJobReceiptCandidate is stage-1 output for exactly one task: the
// claim it would become if finalizeCoherentJobTaskReceiptEvidence later
// confirms its files are actually present in the root checkout. Admission
// alone never grants credit.
type coherentJobReceiptCandidate struct {
	TaskID string
	// Status is the receipt's own normalized terminal status (completed or
	// completed_no_change). Stage 2 needs it because the two are evidenced
	// differently: a `completed` receipt must name at least one file that is
	// really in root, while a `completed_no_change` receipt's evidence IS the
	// commands_run stage 1 already forced it to carry.
	Status string
	Claim  codexBuildTaskClaim
}

// coherentJobReceiptAdmission is the entirety of stage 1's output: candidate
// claims plus the normalized union of paths a later sync step (worktree
// mode, plan 195-08) needs to bring into root before finalization can run.
// It deliberately carries no CompletedTaskIDs field -- only
// finalizeCoherentJobTaskReceiptEvidence may ever produce one (D-08, D-09).
type coherentJobReceiptAdmission struct {
	Candidates []coherentJobReceiptCandidate
	SyncPaths  []string
}

// admitCoherentJobTaskReceipts is stage 1 of the shared two-stage receipt
// trust boundary (D-08, D-09; 195-RESEARCH.md Pattern 4). It proves a
// receipt is well-formed, in scope, evidenced, and bound to its own manifest
// task -- never that its files exist. This stage must not stat/read root
// artifacts, mutate state, or expose completion credit; see
// finalizeCoherentJobTaskReceiptEvidence for the second, root-backed stage.
// root is used only for LEXICAL path normalization (an absolute path that
// resolves inside root becomes repository-relative); it is never used to
// check whether a candidate file exists. aggregateClaims is the dispatch's
// own unioned files_created/files_modified/tests_written/outputs -- a
// receipt may only claim paths the dispatch's own result already reported
// touching, never a path invented solely inside the receipt.
func admitCoherentJobTaskReceipts(root string, phase colony.Phase, dispatch codexBuildDispatch, aggregateClaims []string, receipts []codex.TaskReceipt) (coherentJobReceiptAdmission, []contractViolation) {
	var admission coherentJobReceiptAdmission
	var violations []contractViolation
	worker := strings.TrimSpace(dispatch.Name)

	// WR-08 (195-REVIEW.md): normalize ONCE, here, for every lane. The direct
	// worker path normalized receipts on the way in; the wrapper/external path
	// decoded them straight off the submitted packet, so a receipt spelling its
	// passing check "passed" -- an alias the runtime's own handoff validator
	// accepts -- was credited on one lane and silently refused on the other.
	// This is lexical only: it never reads root, which is what keeps stage 1
	// safe to run before a worktree's files are copied back.
	receipts = codex.NormalizeTaskReceipts(root, receipts)

	covered := make(map[string]struct{}, len(dispatch.CoveredTaskIDs)+1)
	for _, id := range dispatchCoveredTaskIDs(dispatch) {
		covered[id] = struct{}{}
	}
	tasksByID := make(map[string]colony.Task, len(phase.Tasks))
	for idx := range phase.Tasks {
		tasksByID[buildTaskID(phase.Tasks[idx], idx)] = phase.Tasks[idx]
	}
	aggregateSet := make(map[string]struct{}, len(aggregateClaims))
	for _, raw := range aggregateClaims {
		if normalized, err := lexicallyNormalizeReceiptPath(root, raw); err == nil {
			aggregateSet[normalized] = struct{}{}
		}
	}

	seenTaskIDs := make(map[string]struct{}, len(receipts))
	var syncPaths []string
	for _, receipt := range receipts {
		taskID := strings.TrimSpace(receipt.TaskID)
		if taskID == "" {
			violations = append(violations, contractViolation{
				Worker:  worker,
				Field:   "task_receipts.task_id",
				Rule:    violationRuleTaskReceiptIDRequired,
				Message: fmt.Sprintf("%s submitted a task receipt with no task_id", worker),
			})
			continue
		}
		if _, duplicate := seenTaskIDs[taskID]; duplicate {
			violations = append(violations, contractViolation{
				Worker:  worker,
				Field:   "task_receipts.task_id",
				Value:   taskID,
				Rule:    violationRuleTaskReceiptDuplicate,
				Message: fmt.Sprintf("%s submitted more than one task receipt for task %s; neither is admitted", worker, taskID),
			})
			continue
		}
		seenTaskIDs[taskID] = struct{}{}

		task, known := tasksByID[taskID]
		if !known {
			violations = append(violations, contractViolation{
				Worker:  worker,
				Field:   "task_receipts.task_id",
				Value:   taskID,
				Rule:    violationRuleTaskReceiptUnknown,
				Message: fmt.Sprintf("%s submitted a task receipt for %s, but no task in this phase has that ID", worker, taskID),
			})
			continue
		}
		if _, inScope := covered[taskID]; !inScope {
			violations = append(violations, contractViolation{
				Worker:  worker,
				Field:   "task_receipts.task_id",
				Value:   taskID,
				Rule:    violationRuleTaskReceiptOutOfScope,
				Message: fmt.Sprintf("%s submitted a task receipt for %s, but that task was never assigned to this dispatch (covered_task_ids)", worker, taskID),
			})
			continue
		}

		status := strings.ToLower(strings.TrimSpace(receipt.Status))
		if status != codex.TaskReceiptStatusCompleted && status != codex.TaskReceiptStatusCompletedNoChange {
			violations = append(violations, contractViolation{
				Worker:  worker,
				Field:   "task_receipts.status",
				Value:   receipt.Status,
				Rule:    violationRuleTaskReceiptStatusInvalid,
				Message: fmt.Sprintf("%s's receipt for task %s has status %q, not a successful terminal status; refused", worker, taskID, receipt.Status),
			})
			continue
		}
		if strings.TrimSpace(receipt.Summary) == "" {
			violations = append(violations, contractViolation{
				Worker:  worker,
				Field:   "task_receipts.summary",
				Value:   taskID,
				Rule:    violationRuleTaskReceiptSummaryRequired,
				Message: fmt.Sprintf("%s's receipt for task %s has no summary", worker, taskID),
			})
			continue
		}
		if err := codex.ValidateWorkerHandoff(receipt.Handoff); err != nil {
			violations = append(violations, contractViolation{
				Worker:  worker,
				Field:   "task_receipts.handoff",
				Value:   taskID,
				Rule:    violationRuleTaskReceiptHandoffInvalid,
				Message: fmt.Sprintf("%s's receipt for task %s has an invalid handoff: %v", worker, taskID, err),
			})
			continue
		}
		verification := strings.ToLower(strings.TrimSpace(receipt.Handoff.VerificationStatus))
		if verification != "pass" || len(receipt.Handoff.CommandsRun) == 0 {
			violations = append(violations, contractViolation{
				Worker:  worker,
				Field:   "task_receipts.handoff.verification_status",
				Value:   taskID,
				Rule:    violationRuleTaskReceiptUnevidenced,
				Message: fmt.Sprintf("%s's receipt for task %s does not carry a passing, concrete verification -- verification_status: pass and at least one commands_run entry are both required", worker, taskID),
			})
			continue
		}

		claim := codexBuildTaskClaim{TaskID: taskID}
		pathsOK := true
		normalizeReceiptField := func(field string, values []string) []string {
			out := make([]string, 0, len(values))
			for _, raw := range values {
				normalized, err := lexicallyNormalizeReceiptPath(root, raw)
				if err != nil {
					violations = append(violations, contractViolation{
						Worker:  worker,
						Field:   field,
						Value:   raw,
						Rule:    violationRuleTaskReceiptPathLaundered,
						Message: fmt.Sprintf("%s's receipt for task %s claims %s %q, which is not a safe repository-relative path: %v", worker, taskID, field, raw, err),
					})
					pathsOK = false
					continue
				}
				if _, inAggregate := aggregateSet[normalized]; !inAggregate {
					violations = append(violations, contractViolation{
						Worker:  worker,
						Field:   field,
						Value:   normalized,
						Rule:    violationRuleTaskReceiptPathOutOfClaims,
						Message: fmt.Sprintf("%s's receipt for task %s claims %s %q, which %s's own result never reported touching", worker, taskID, field, normalized, worker),
					})
					pathsOK = false
					continue
				}
				out = append(out, normalized)
			}
			return uniqueSortedStrings(out)
		}
		claim.FilesCreated = normalizeReceiptField("task_receipts.files_created", receipt.FilesCreated)
		claim.FilesModified = normalizeReceiptField("task_receipts.files_modified", receipt.FilesModified)
		claim.TestsWritten = normalizeReceiptField("task_receipts.tests_written", receipt.TestsWritten)
		if !pathsOK {
			continue
		}

		// D-09: bind the claim to the task's OWN declared files whenever the
		// manifest declares any -- a receipt that touches only files with no
		// relationship to this task's own evidence artifacts/hints is not
		// evidence that THIS task's requirements were met, whatever else it
		// changed. Tasks with no declared paths at all skip this check: there
		// is nothing to bind against, and inferring ownership from prose
		// would violate the "no ownership inferred from prose tokens"
		// guidance (195-RESEARCH.md, What Might Have Been Missed).
		if declared := declaredPathsForTask(task); len(declared) > 0 {
			declaredSet := make(map[string]struct{}, len(declared))
			for _, d := range declared {
				declaredSet[d] = struct{}{}
			}
			matched := false
			for _, p := range claimedPaths(claim) {
				if _, ok := declaredSet[p]; ok {
					matched = true
					break
				}
			}
			if !matched {
				violations = append(violations, contractViolation{
					Worker:  worker,
					Field:   "task_receipts.files_modified",
					Value:   taskID,
					Rule:    violationRuleTaskReceiptRequirementMismatch,
					Message: fmt.Sprintf("%s's receipt for task %s touches none of that task's own declared files (%s)", worker, taskID, strings.Join(declared, ", ")),
				})
				continue
			}
		}

		admission.Candidates = append(admission.Candidates, coherentJobReceiptCandidate{TaskID: taskID, Status: status, Claim: claim})
		syncPaths = append(syncPaths, claimedPaths(claim)...)
	}
	sort.Slice(admission.Candidates, func(i, j int) bool { return admission.Candidates[i].TaskID < admission.Candidates[j].TaskID })
	admission.SyncPaths = uniqueSortedStrings(syncPaths)
	return admission, violations
}

// finalizeCoherentJobTaskReceiptEvidence is stage 2 of the shared receipt
// trust boundary. It consumes ONLY stage-1 candidates, confirms their files
// are present in the root checkout right now, and is the only function in
// this codebase allowed to populate a runtime-owned completed-task-ID set
// (D-08, D-09). phase and dispatch are accepted for lane-neutral symmetry
// with admitCoherentJobTaskReceipts -- plans 195-06 and 195-08 reuse both
// stages unchanged, the worktree lane inserting a sync step between them.
// A candidate whose claimed paths are not readable regular files in root
// right now is dropped without credit, and so is a `completed` candidate that
// names no path at all -- pointing at nothing is never root-backed evidence
// (CR-01, 195-REVIEW.md). The single exception is completed_no_change, whose
// evidence is the commands_run stage 1 already required (ruling D6).
func finalizeCoherentJobTaskReceiptEvidence(root string, phase colony.Phase, dispatch codexBuildDispatch, admission coherentJobReceiptAdmission) ([]codexBuildTaskClaim, []string, []contractViolation) {
	_ = phase
	var claims []codexBuildTaskClaim
	var completedTaskIDs []string
	var violations []contractViolation
	worker := strings.TrimSpace(dispatch.Name)

	for _, candidate := range admission.Candidates {
		claim := candidate.Claim
		paths := claimedPaths(claim)
		if len(paths) == 0 {
			// CR-01 (195-REVIEW.md): the root-evidence block below used to be
			// guarded by `if len(paths) > 0`, which made the ENTIRE
			// root-backed check optional -- a candidate claiming no path at
			// all fell straight through to credit, and did so even against a
			// root that does not exist. A receipt that points at nothing is
			// not evidence of anything.
			//
			// The one narrow exception is completed_no_change, whose evidence
			// is by definition not a file: stage 1 has already proven that
			// receipt carries a passing verification_status and at least one
			// concrete commands_run entry (ruling D6). Everything else is
			// refused by name here rather than credited.
			if candidate.Status != codex.TaskReceiptStatusCompletedNoChange {
				violations = append(violations, contractViolation{
					Worker:  worker,
					Field:   "task_receipts.files_modified",
					Value:   candidate.TaskID,
					Rule:    violationRuleTaskReceiptUnevidenced,
					Message: fmt.Sprintf("task %s's receipt claims completion but names no file in the project; not credited", candidate.TaskID),
				})
				continue
			}
		} else {
			evidenceInput := codexBuildClaims{
				FilesCreated:  claim.FilesCreated,
				FilesModified: claim.FilesModified,
				TestsWritten:  claim.TestsWritten,
			}
			attachBuildArtifactEvidence(root, &evidenceInput)
			evidenceByPath := make(map[string]codexBuildArtifactEvidence, len(evidenceInput.ArtifactEvidence))
			for _, e := range evidenceInput.ArtifactEvidence {
				evidenceByPath[e.Path] = e
			}
			var missing []string
			evidence := make([]codexBuildArtifactEvidence, 0, len(paths))
			for _, p := range paths {
				e, ok := evidenceByPath[p]
				if !ok {
					missing = append(missing, p)
					continue
				}
				evidence = append(evidence, e)
			}
			if len(missing) > 0 {
				violations = append(violations, contractViolation{
					Worker:  worker,
					Field:   "task_receipts.root_evidence",
					Value:   candidate.TaskID,
					Rule:    violationRuleTaskReceiptRootEvidenceMissing,
					Message: fmt.Sprintf("task %s's receipt claims %s, but the current root checkout does not have that file (or it is unreadable); not credited", candidate.TaskID, strings.Join(missing, ", ")),
				})
				continue
			}
			claim.ArtifactEvidence = evidence
		}
		claims = append(claims, claim)
		completedTaskIDs = append(completedTaskIDs, candidate.TaskID)
	}
	sort.Slice(claims, func(i, j int) bool { return claims[i].TaskID < claims[j].TaskID })
	return claims, uniqueSortedStrings(completedTaskIDs), violations
}

// resolveCoherentJobDispatchReceipts is the native/in-repo caller of both
// receipt stages: because in-repo files already live in root, it runs
// admission then immediately finalizes, storing ONLY the finalizer's own
// output onto each dispatch (D-08, D-09). Whole-success dispatches
// (completed/completed_no_change) are returned unchanged -- they already
// credit every covered task through completedBuildTaskIDs' existing branch,
// and a receipt is never needed to prove what a clean success already
// proved. Worktree mode (plan 195-08) reuses the same two stages with a
// sync step in between instead of calling this function.
func resolveCoherentJobDispatchReceipts(root string, phase colony.Phase, dispatches []codexBuildDispatch) []codexBuildDispatch {
	resolved := make([]codexBuildDispatch, len(dispatches))
	copy(resolved, dispatches)
	for i := range resolved {
		// Already resolved by a lane that inserted its own sync step between
		// the two stages (worktree mode). Re-running admission here would
		// read a root this build itself just wrote, so the earlier, honest
		// verdict stands.
		if resolved[i].ReceiptsResolved {
			continue
		}
		status := strings.TrimSpace(resolved[i].Status)
		if status == "completed" || isNoChangeExternalBuildStatus(status) {
			continue
		}
		if len(resolved[i].TaskReceipts) == 0 {
			continue
		}
		aggregateClaims := append([]string{}, resolved[i].Outputs...)
		admission, admissionViolations := admitCoherentJobTaskReceipts(root, phase, resolved[i], aggregateClaims, resolved[i].TaskReceipts)
		claims, completedTaskIDs, finalViolations := finalizeCoherentJobTaskReceiptEvidence(root, phase, resolved[i], admission)
		resolved[i].TaskClaims = claims
		resolved[i].CompletedTaskIDs = completedTaskIDs
		// WR-01 (195-REVIEW.md): both violation slices used to be discarded
		// here (`admission, _ :=` / `claims, completedTaskIDs, _ :=`), which
		// made a wrapper mis-shaping every receipt it submitted look exactly
		// like a worker that legitimately finished nothing. This file's own
		// header promises a refusal is "never a silent drop".
		reportCoherentJobReceiptRefusals(resolved[i].Name, append(admissionViolations, finalViolations...))
	}
	return resolved
}

// reportCoherentJobReceiptRefusals is the single owner-visible exit for every
// named receipt refusal, on every lane (WR-01, 195-REVIEW.md). It prints one
// line per refused receipt, naming the worker and the reason in the same plain
// words the violation already carries. It is deliberately unconditional and
// writes to stderr: a refusal is diagnostic output the owner must see even when
// the command's own result is being consumed as JSON.
func reportCoherentJobReceiptRefusals(worker string, violations []contractViolation) {
	if len(violations) == 0 {
		return
	}
	worker = strings.TrimSpace(worker)
	if worker == "" {
		worker = "a worker"
	}
	visualFprintf(stderr, "%s: %d task receipt(s) were not accepted as proof of finished work:\n", worker, len(violations))
	for _, violation := range violations {
		message := strings.TrimSpace(violation.Message)
		if message == "" {
			message = fmt.Sprintf("a task receipt was refused (%s)", violation.Rule)
		}
		visualFprintf(stderr, "  - %s\n", message)
	}
}

const violationRuleTaskReceiptSyncFailed = "task_receipt.sync_failed"

// coherentJobWorktreeReceipts is the result of running the shared two-stage
// boundary across a worktree boundary: what was brought back, what was
// credited, and what the worker touched but never proved.
type coherentJobWorktreeReceipts struct {
	// Claims and CompletedTaskIDs come from
	// finalizeCoherentJobTaskReceiptEvidence and nowhere else.
	Claims           []codexBuildTaskClaim
	CompletedTaskIDs []string
	// SyncedPaths are the candidate paths actually copied into root.
	SyncedPaths []string
	// UncreditedPaths are paths the worker touched that no admitted receipt
	// claimed. They are deliberately left in the worker's checkout.
	UncreditedPaths []string
	Violations      []contractViolation
}

// resolveCoherentJobWorktreeReceipts is the ONLY worktree-mode path from a
// worker's task receipts to task credit, and it is the same two-stage
// boundary both other lanes use -- never a second, worktree-only validator.
//
// The ordering is the whole point (D-08, D-09). Stage 1
// (admitCoherentJobTaskReceipts) is lexical: it proves each receipt is
// well-formed, in scope, evidenced and inside the worker's own reported
// claims WITHOUT reading root, which is exactly what makes it safe to run
// while the files still live only inside the worktree. Only the paths that
// admission returned are then copied into root. Stage 2
// (finalizeCoherentJobTaskReceiptEvidence) runs last, against root, and is
// the only function that may produce CompletedTaskIDs -- so a file that
// exists solely inside a worktree can never become task credit, and a
// candidate whose copy into root failed is dropped by name instead of being
// credited on the strength of a promise.
//
// touched is everything the worker changed in its checkout. Anything in it
// that no admitted receipt claimed is returned as UncreditedPaths and is
// deliberately NOT synced: the caller preserves that checkout instead, so
// unproven work is recoverable rather than either lost or falsely credited.
func resolveCoherentJobWorktreeReceipts(
	root string,
	workerRoot string,
	phase colony.Phase,
	dispatch codexBuildDispatch,
	aggregateClaims []string,
	touched []string,
	receipts []codex.TaskReceipt,
) coherentJobWorktreeReceipts {
	var out coherentJobWorktreeReceipts
	worker := strings.TrimSpace(dispatch.Name)

	admission, violations := admitCoherentJobTaskReceipts(root, phase, dispatch, aggregateClaims, receipts)
	out.Violations = append(out.Violations, violations...)

	candidatePaths := make(map[string]struct{}, len(admission.SyncPaths))
	for _, path := range admission.SyncPaths {
		candidatePaths[path] = struct{}{}
	}
	for _, path := range uniqueSortedStrings(touched) {
		if _, claimed := candidatePaths[filepath.ToSlash(strings.TrimSpace(path))]; claimed {
			continue
		}
		out.UncreditedPaths = append(out.UncreditedPaths, path)
	}

	// Sync ONLY the admitted candidate paths, one at a time, so a single
	// unwritable destination excludes exactly its own task rather than
	// silently poisoning or silently crediting the rest.
	syncFailures := map[string]error{}
	for _, path := range admission.SyncPaths {
		if err := syncRelativePath(workerRoot, root, path); err != nil {
			syncFailures[path] = err
			continue
		}
		out.SyncedPaths = append(out.SyncedPaths, path)
	}

	synced := coherentJobReceiptAdmission{SyncPaths: out.SyncedPaths}
	for _, candidate := range admission.Candidates {
		var failed []string
		for _, path := range claimedPaths(candidate.Claim) {
			if err, bad := syncFailures[path]; bad {
				failed = append(failed, fmt.Sprintf("%s (%v)", path, err))
			}
		}
		if len(failed) > 0 {
			out.Violations = append(out.Violations, contractViolation{
				Worker:  worker,
				Field:   "task_receipts.sync",
				Value:   candidate.TaskID,
				Rule:    violationRuleTaskReceiptSyncFailed,
				Message: fmt.Sprintf("task %s's proof could not be copied out of the worker's own workspace into the project: %s; not credited", candidate.TaskID, strings.Join(failed, ", ")),
			})
			continue
		}
		synced.Candidates = append(synced.Candidates, candidate)
	}

	claims, completedTaskIDs, finalViolations := finalizeCoherentJobTaskReceiptEvidence(root, phase, dispatch, synced)
	out.Claims = claims
	out.CompletedTaskIDs = completedTaskIDs
	out.Violations = append(out.Violations, finalViolations...)
	return out
}

// resolveWorktreeExternalDispatchReceipts is the external/wrapper lane's
// worktree entry point. It is deliberately the same function the native
// worktree lane calls (resolveCoherentJobWorktreeReceipts), applied to the
// checkout the colony state already tracks for that worker, so both lanes
// produce the same credit set from the same evidence. A dispatch whose
// checkout is gone resolves nothing rather than falling back to root, because
// crediting from a root nobody synced would credit work no one proved.
func resolveWorktreeExternalDispatchReceipts(root string, phase colony.Phase, state colony.ColonyState, phaseNum int, dispatches []codexBuildDispatch) []codexBuildDispatch {
	checkoutByWorker := map[string]string{}
	for _, entry := range state.Worktrees {
		if entry.Phase != phaseNum {
			continue
		}
		agent := strings.TrimSpace(entry.Agent)
		if agent == "" {
			continue
		}
		abs := entry.Path
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(root, filepath.FromSlash(entry.Path))
		}
		if info, err := os.Stat(abs); err != nil || !info.IsDir() {
			continue
		}
		checkoutByWorker[agent] = abs
	}

	resolved := make([]codexBuildDispatch, len(dispatches))
	copy(resolved, dispatches)
	for i := range resolved {
		status := strings.TrimSpace(resolved[i].Status)
		if status == "completed" || isNoChangeExternalBuildStatus(status) {
			continue
		}
		if len(resolved[i].TaskReceipts) == 0 {
			continue
		}
		checkout, ok := checkoutByWorker[strings.TrimSpace(resolved[i].Name)]
		if !ok {
			continue
		}
		touched := worktreeTouchedPathsForExternalResult(checkout, resolved[i])
		outcome := resolveCoherentJobWorktreeReceipts(
			root, checkout, phase, resolved[i], append([]string{}, resolved[i].Outputs...), touched, resolved[i].TaskReceipts)
		resolved[i].TaskClaims = outcome.Claims
		resolved[i].CompletedTaskIDs = outcome.CompletedTaskIDs
		resolved[i].ReceiptsResolved = true
		// WR-01: this lane never read outcome.Violations at all.
		reportCoherentJobReceiptRefusals(resolved[i].Name, outcome.Violations)
	}
	return resolved
}

// worktreeTouchedPathsForExternalResult reports what the worker actually
// changed inside its checkout, so paths it never proved can be identified and
// deliberately left there.
func worktreeTouchedPathsForExternalResult(checkout string, dispatch codexBuildDispatch) []string {
	touched := append([]string{}, dispatch.Outputs...)
	statuses, err := snapshotGitStatus(checkout)
	if err != nil {
		return uniqueSortedStrings(touched)
	}
	for rel := range statuses {
		if rel == "" || strings.HasPrefix(rel, ".aether/") {
			continue
		}
		touched = append(touched, rel)
	}
	return uniqueSortedStrings(touched)
}

// claimedPaths returns the union of a task claim's own files_created,
// files_modified, and tests_written, deduplicated and sorted.
func claimedPaths(claim codexBuildTaskClaim) []string {
	return uniqueSortedStrings(append(append(append([]string{}, claim.FilesCreated...), claim.FilesModified...), claim.TestsWritten...))
}

// lexicallyNormalizeReceiptPath applies ONLY lexical normalization: an
// absolute path that resolves inside root becomes repository-relative (no
// stat, no read), then normalizeCriterionArtifactPath enforces that the
// result is a safe, repository-relative, non-escaping path. It never checks
// whether the file exists -- that is finalizeCoherentJobTaskReceiptEvidence's
// job, using root-backed evidence instead of a lexical guess.
func lexicallyNormalizeReceiptPath(root, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("path is empty")
	}
	candidate := raw
	cleanedRoot := strings.TrimSpace(root)
	fromSlash := filepath.FromSlash(raw)
	if cleanedRoot != "" && filepath.IsAbs(fromSlash) {
		if rel, err := filepath.Rel(cleanedRoot, fromSlash); err == nil && !strings.HasPrefix(rel, "..") {
			candidate = rel
		}
	}
	return normalizeCriterionArtifactPath(candidate)
}
