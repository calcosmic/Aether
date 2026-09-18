package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// Match colony.NewTaskReferenceIndex's whitespace normalization without
// rewriting authored task IDs or dependency strings in persisted state.
func buildDependencyReference(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// Dependency credit is a scheduling projection, never a new completion receipt.
// Failed jobs can supply only task-specific credit accepted by the existing
// admission/finalization boundary. Group membership alone conveys no credit.
func buildDependencyCredit(phase colony.Phase) map[string]bool {
	credit := map[string]bool{}
	phases := []colony.Phase{phase}
	if store != nil {
		var state colony.ColonyState
		if store.LoadJSON("COLONY_STATE.json", &state) == nil {
			for _, prior := range state.Plan.Phases {
				if prior.ID != phase.ID {
					phases = append(phases, prior)
				}
			}
		}
	}
	for _, p := range phases {
		for i, task := range p.Tasks {
			if task.Status == colony.TaskCompleted {
				// Legacy generated IDs are local scheduling identities. Other
				// phases can grant credit only under their authored aliases;
				// task-1 in a prior phase must not authorize this phase's task-1.
				if p.ID == phase.ID {
					credit[buildDependencyReference(buildTaskID(task, i))] = true
				} else if task.ID != nil && buildDependencyReference(*task.ID) != "" {
					credit[buildDependencyReference(*task.ID)] = true
				}
				if buildDependencyReference(task.SemanticID) != "" {
					credit[buildDependencyReference(task.SemanticID)] = true
				}
			}
		}
	}
	return credit
}

func partitionReadyBuildDispatches(phase colony.Phase, dispatches []codex.WorkerDispatch, credit map[string]bool) ([]codex.WorkerDispatch, []codex.DispatchResult) {
	var ready []codex.WorkerDispatch
	var blocked []codex.DispatchResult
	for _, dispatch := range dispatches {
		covered := map[string]bool{}
		if dispatch.TaskID != "" {
			covered[buildDependencyReference(dispatch.TaskID)] = true
		}
		for _, id := range dispatch.CoveredTaskIDs {
			if id != "" {
				covered[buildDependencyReference(id)] = true
			}
		}
		// Dependencies internal to a grouped job are handled within that job.
		for i, task := range phase.Tasks {
			if covered[buildDependencyReference(buildTaskID(task, i))] && task.SemanticID != "" {
				covered[buildDependencyReference(task.SemanticID)] = true
			}
		}
		var missing []string
		for i, task := range phase.Tasks {
			if !covered[buildDependencyReference(buildTaskID(task, i))] && !covered[buildDependencyReference(task.SemanticID)] {
				continue
			}
			for _, dependency := range task.DependsOn {
				if !covered[buildDependencyReference(dependency)] && !credit[buildDependencyReference(dependency)] {
					missing = append(missing, dependency)
				}
			}
		}
		if len(missing) == 0 {
			ready = append(ready, dispatch)
			continue
		}
		err := fmt.Errorf("not dispatched: prerequisite tasks lack accepted completion evidence: %s", strings.Join(uniqueSortedStrings(missing), ", "))
		blocked = append(blocked, codex.DispatchResult{WorkerName: dispatch.WorkerName, Status: "dependency_blocked", Error: err})
	}
	return ready, blocked
}

func acceptBuildWaveCredit(root string, phase colony.Phase, originals []codexBuildDispatch, results []codex.DispatchResult, ledger *worktreeReceiptLedger, credit map[string]bool, requireFiles bool) {
	byName := map[string]codexBuildDispatch{}
	for _, d := range originals {
		byName[d.Name] = d
	}
	for _, result := range results {
		d, ok := byName[result.WorkerName]
		if !ok {
			continue
		}
		d.Status = result.Status
		if result.WorkerResult != nil {
			d.TaskReceipts = result.WorkerResult.TaskReceipts
			d.Outputs = buildDispatchClaimOutputs(*result.WorkerResult)
			d.Summary = result.WorkerResult.Summary
		}
		if resolved, ok := ledger.lookup(d.Name); ok {
			d.TaskClaims = resolved.Claims
			d.CompletedTaskIDs = resolved.CompletedTaskIDs
			d.ReceiptsResolved = true
		}
		if isSuccessfulExternalBuildStatus(d.Status) {
			if result.Error != nil || result.WorkerResult == nil {
				continue
			}
			if validateRuntimeNoChangeEvidence([]codex.DispatchResult{result}) != nil {
				continue
			}
			if validateRuntimeBuildDispatchResults(phase, []codexBuildDispatch{d}, codex.ExtractClaims([]codex.DispatchResult{result}), requireFiles) != nil {
				continue
			}
		}
		resolved := resolveCoherentJobDispatchReceipts(root, phase, []codexBuildDispatch{d})
		for id := range completedBuildTaskIDs(resolved) {
			credit[buildDependencyReference(id)] = true
			for i, task := range phase.Tasks {
				if buildDependencyReference(buildTaskID(task, i)) == buildDependencyReference(id) && task.SemanticID != "" {
					credit[buildDependencyReference(task.SemanticID)] = true
				}
			}
		}
	}
}

// Report the surviving workspace, without claiming these paths were all created
// by this attempt: user edits may predate dispatch. Never remove drafts.
func retainedBuildDraftReport(root string) string {
	out, err := exec.Command("git", "-C", root, "status", "--porcelain=v1", "-z", "--untracked-files=all").Output()
	if err != nil {
		return "Working-tree edits were not rolled back. Draft inventory unavailable; inspect the workspace before retrying."
	}
	var paths []string
	entries := strings.Split(string(out), "\x00")
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if len(entry) < 4 {
			continue
		}
		path := entry[3:]
		if !strings.HasPrefix(path, ".aether/data/") {
			paths = append(paths, fmt.Sprintf("%q", path))
		}
		if entry[0] == 'R' || entry[0] == 'C' || entry[1] == 'R' || entry[1] == 'C' {
			i++
		}
	}
	if len(paths) == 0 {
		return "Working-tree edits were not rolled back. No tracked or untracked project changes were found; ignored files are not inventoried."
	}
	return "Surviving working-tree changes (uncredited drafts may remain; includes any pre-existing edits): " + strings.Join(uniqueSortedStrings(paths), ", ") + ". Inspect and reconcile these paths before retrying; ignored files are not inventoried."
}
