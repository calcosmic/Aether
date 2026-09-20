package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestMergedDispatchCreditsEveryCoveredTask locks the receipt side of the
// one-worker-owns-a-chain design: when a single worker is given a run of
// dependent steps, finishing it must mark EVERY step complete, not just the
// first.
//
// Written to settle a downstream field report (2026-08-21, dashboard colony)
// which said the opposite — "sent one worker to do all six jobs but recorded
// it as having done only job #1" — and diagnosed it as build-finalize never
// reading CoveredTaskIDs, on the evidence that the identifier does not appear
// in cmd/codex_build_finalize.go.
//
// The identifier's absence is not the absence of the behaviour. Finalize
// starts each reconciled dispatch from the MANIFEST's dispatch (which carries
// CoveredTaskIDs verbatim), overlays only the reported status, and hands the
// result to reconcileCompletedBuildTasks -> completedBuildTaskIDs ->
// dispatchCoveredTaskIDs, which expands the covered chain. Run end to end
// below, all three steps of a three-step chain finish `completed`.
//
// So the reported symptom is real but its stated cause is not, and the fix it
// asked for would have been a no-op. The likeliest actual source is the
// stage/finalize inconsistency reported alongside it, where an attempt commits
// while its dispatches stay `planned` — in which case completedBuildTaskIDs
// skips them for failing the status check, long before covered IDs matter.
//
// This test exists because nothing asserted the invariant either way, which is
// why a plausible misreading of the code could stand unchallenged.
func TestMergedDispatchCreditsEveryCoveredTask(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Credit every covered task"
	ids := []string{"1.1", "1.2", "1.3"}
	tasks := make([]colony.Task, 0, len(ids))
	for i := range ids {
		id := ids[i]
		task := colony.Task{ID: &id, Goal: "Sequential step " + id, Status: colony.TaskPending}
		if i > 0 {
			// Dependencies are what put each step in its own single-dispatch
			// wave, which is the shape planCoherentJobs groups into one
			// job. Independent tasks share a wave and never group, so a
			// fixture without these silently proves nothing.
			task.DependsOn = []string{ids[i-1]}
		}
		tasks = append(tasks, task)
	}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "standard", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Name: "Merged chain", Description: "One worker, several dependent steps",
			Status: colony.PhaseReady, Tasks: tasks,
		}}},
	})

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("plan-only build: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)

	var chain codexBuildDispatch
	for _, dispatch := range manifest.Dispatches {
		if len(dispatch.CoveredTaskIDs) > 1 {
			chain = dispatch
			break
		}
	}
	if chain.Name == "" {
		t.Fatalf("fixture produced no merged dispatch, so it cannot exercise covered-task crediting at all; dispatches: %+v", manifest.Dispatches)
	}
	if len(chain.CoveredTaskIDs) != len(ids) {
		t.Fatalf("merged dispatch %s covers %v, want all of %v", chain.Name, chain.CoveredTaskIDs, ids)
	}

	results := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		worker := codexExternalBuildWorkerResult{
			Stage: dispatch.Stage, Wave: dispatch.Wave, ExecutionWave: normalizedDispatchWave(dispatch),
			Caste: dispatch.Caste, Name: dispatch.Name, TaskID: dispatch.TaskID,
			Status: "completed", Summary: dispatch.Name + " finished its chain",
			Handoff: codex.WorkerHandoff{
				CommandsRun:            []string{"go test ./..."},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{"chain complete"},
			},
		}
		if dispatch.Caste == "builder" {
			worker.FilesModified = []string{"external-evidence.txt"}
		}
		results = append(results, worker)
	}

	// The real finalize reconciliation, not a hand-built dispatch slice: this
	// is where a completion packet that carries no covered_task_ids field
	// meets a manifest dispatch that does.
	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("unexpected contract violations on a well-formed completion packet: %+v", violations)
	}

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("load colony state: %v", err)
	}
	reconcileCompletedBuildTasks(&state, 1, dispatches)

	for _, task := range state.Plan.Phases[0].Tasks {
		if task.Status != colony.TaskCompleted {
			t.Fatalf("task %s is %q after one worker completed the whole chain it was briefed on (%v), want %q — the worker did the work and the receipt lost it, so the phase can never advance",
				*task.ID, task.Status, chain.CoveredTaskIDs, colony.TaskCompleted)
		}
	}
}

// sixChainedTasks builds six dependent tasks (1.1 -> 1.2 -> ... -> 1.6),
// the exact shape the coherent-job planner merges into one dispatch, for
// D-08/D-09's "four of six" partial-credit fixtures below.
func sixChainedTasks() ([]colony.Task, []string) {
	ids := []string{"1.1", "1.2", "1.3", "1.4", "1.5", "1.6"}
	tasks := make([]colony.Task, 0, len(ids))
	for i := range ids {
		id := ids[i]
		task := colony.Task{ID: &id, Goal: "Sequential step " + id, Status: colony.TaskPending}
		if i > 0 {
			task.DependsOn = []string{ids[i-1]}
		}
		tasks = append(tasks, task)
	}
	return tasks, ids
}

func taskFileName(taskID string) string {
	return fmt.Sprintf("task-%s.go", taskID)
}

// receiptForTask builds a well-formed, root-backed task receipt: the file it
// claims must actually exist under root for finalizeCoherentJobTaskReceiptEvidence
// to credit it.
func receiptForTask(t *testing.T, root, taskID string) codex.TaskReceipt {
	t.Helper()
	file := taskFileName(taskID)
	if err := os.WriteFile(filepath.Join(root, file), []byte("package fixture\n"), 0644); err != nil {
		t.Fatalf("write fixture file for task %s: %v", taskID, err)
	}
	return codex.TaskReceipt{
		TaskID:  taskID,
		Status:  codex.TaskReceiptStatusCompleted,
		Summary: "finished " + taskID,
		// FilesCreated/TestsWritten are non-omitempty on codex.TaskReceipt --
		// the completion-packet schema requires them present as arrays, never
		// JSON null. A nil Go slice here is fine for the native-lane callers
		// (which never JSON-round-trip a receipt), but a full external
		// build-finalize call does round-trip through
		// validateCompletionPacketSemantics -> completionPacketAsRaw, so an
		// empty (not nil) slice keeps this fixture reusable for both lanes
		// (195-06).
		FilesCreated:  []string{},
		FilesModified: []string{file},
		TestsWritten:  []string{},
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "pass",
			CommandsRun:        []string{"go test ./..."},
		},
	}
}

// TestFailedGroupedDispatchCreditsExactlyReceiptedTasks is the shipped
// consequence of D-08/D-09: a single worker given six chained tasks that
// crashes after finishing four must credit exactly those four, honestly
// leaving the other two pending, never all six and never zero.
func TestFailedGroupedDispatchCreditsExactlyReceiptedTasks(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Credit exactly the receipted tasks"
	tasks, ids := sixChainedTasks()
	phase := colony.Phase{
		ID: 1, Name: "Merged chain", Description: "One worker, six dependent steps",
		Status: colony.PhaseReady, Tasks: tasks,
	}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "standard", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{phase}},
	})

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("plan-only build: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)

	var chain codexBuildDispatch
	for _, dispatch := range manifest.Dispatches {
		if len(dispatch.CoveredTaskIDs) > 1 {
			chain = dispatch
			break
		}
	}
	if chain.Name == "" || len(chain.CoveredTaskIDs) != len(ids) {
		t.Fatalf("fixture did not produce one merged dispatch covering all six tasks; dispatches: %+v", manifest.Dispatches)
	}

	receiptedIDs := ids[:4] // 1.1..1.4
	pendingIDs := ids[4:]   // 1.5, 1.6
	receipts := make([]codex.TaskReceipt, 0, len(receiptedIDs))
	touchedFiles := make([]string, 0, len(receiptedIDs))
	for _, id := range receiptedIDs {
		receipts = append(receipts, receiptForTask(t, root, id))
		touchedFiles = append(touchedFiles, taskFileName(id))
	}

	results := []codexExternalBuildWorkerResult{{
		Stage: chain.Stage, Wave: chain.Wave, ExecutionWave: normalizedDispatchWave(chain),
		Caste: chain.Caste, Name: chain.Name, TaskID: chain.TaskID,
		Status:        "failed",
		Summary:       "crashed after finishing four of six steps",
		FilesModified: touchedFiles,
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "fail",
			CommandsRun:        []string{"go test ./..."},
		},
		TaskReceipts: receipts,
	}}

	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("unexpected contract violations merging a failed dispatch with valid task receipts: %+v", violations)
	}
	dispatches = resolveCoherentJobDispatchReceipts(root, phase, dispatches)

	var resolved codexBuildDispatch
	for _, d := range dispatches {
		if d.Name == chain.Name {
			resolved = d
			break
		}
	}
	gotCompleted := append([]string{}, resolved.CompletedTaskIDs...)
	sort.Strings(gotCompleted)
	if fmt.Sprint(gotCompleted) != fmt.Sprint(receiptedIDs) {
		t.Fatalf("CompletedTaskIDs = %v, want exactly %v", gotCompleted, receiptedIDs)
	}
	gotClaimIDs := make([]string, 0, len(resolved.TaskClaims))
	for _, c := range resolved.TaskClaims {
		gotClaimIDs = append(gotClaimIDs, c.TaskID)
	}
	sort.Strings(gotClaimIDs)
	if fmt.Sprint(gotClaimIDs) != fmt.Sprint(receiptedIDs) {
		t.Fatalf("TaskClaims cover %v, want exactly %v (no claim for an uncredited task, D-08/D-09)", gotClaimIDs, receiptedIDs)
	}

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("load colony state: %v", err)
	}
	reconcileCompletedBuildTasks(&state, 1, dispatches)

	statusByID := map[string]string{}
	for _, task := range state.Plan.Phases[0].Tasks {
		statusByID[*task.ID] = string(task.Status)
	}
	for _, id := range receiptedIDs {
		if statusByID[id] != string(colony.TaskCompleted) {
			t.Fatalf("task %s is %q, want %q (a receipted task in a failed dispatch must still be credited)", id, statusByID[id], colony.TaskCompleted)
		}
	}
	for _, id := range pendingIDs {
		if statusByID[id] == string(colony.TaskCompleted) {
			t.Fatalf("task %s is %q, want it to remain pending -- it has no receipt, so D-08 forbids crediting it from a failed dispatch's touched files or covered_task_ids membership", id, statusByID[id])
		}
	}
}

// TestFailedGroupedDispatchWithoutReceiptsCreditsNone is D-08's negative
// case: touching every file and naming every task in the summary is not
// evidence. Without a single trustworthy task receipt, a failed dispatch
// credits nothing at all, however plausible the touched files look.
func TestFailedGroupedDispatchWithoutReceiptsCreditsNone(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Credit nothing without receipts"
	tasks, ids := sixChainedTasks()
	phase := colony.Phase{
		ID: 1, Name: "Merged chain", Description: "One worker, six dependent steps",
		Status: colony.PhaseReady, Tasks: tasks,
	}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "standard", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{phase}},
	})

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("plan-only build: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)

	var chain codexBuildDispatch
	for _, dispatch := range manifest.Dispatches {
		if len(dispatch.CoveredTaskIDs) > 1 {
			chain = dispatch
			break
		}
	}
	if chain.Name == "" || len(chain.CoveredTaskIDs) != len(ids) {
		t.Fatalf("fixture did not produce one merged dispatch covering all six tasks; dispatches: %+v", manifest.Dispatches)
	}

	touchedFiles := make([]string, 0, len(ids))
	for _, id := range ids {
		file := taskFileName(id)
		if err := os.WriteFile(filepath.Join(root, file), []byte("package fixture\n"), 0644); err != nil {
			t.Fatalf("write fixture file for task %s: %v", id, err)
		}
		touchedFiles = append(touchedFiles, file)
	}

	results := []codexExternalBuildWorkerResult{{
		Stage: chain.Stage, Wave: chain.Wave, ExecutionWave: normalizedDispatchWave(chain),
		Caste: chain.Caste, Name: chain.Name, TaskID: chain.TaskID,
		Status:  "failed",
		Summary: "touched every file for tasks " + fmt.Sprint(ids) + " but has no per-task receipt",
		// The counterexample: every file the six tasks would touch is
		// reported, and the summary names every task, but NO task_receipts
		// entry exists. D-08 forbids inferring completion from either signal.
		FilesModified: touchedFiles,
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "fail",
			CommandsRun:        []string{"go test ./..."},
		},
	}}

	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("unexpected contract violations: %+v", violations)
	}
	dispatches = resolveCoherentJobDispatchReceipts(root, phase, dispatches)

	var resolved codexBuildDispatch
	for _, d := range dispatches {
		if d.Name == chain.Name {
			resolved = d
			break
		}
	}
	if len(resolved.CompletedTaskIDs) != 0 || len(resolved.TaskClaims) != 0 {
		t.Fatalf("a failed dispatch with no task receipts must credit zero tasks; got CompletedTaskIDs=%v TaskClaims=%+v", resolved.CompletedTaskIDs, resolved.TaskClaims)
	}

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("load colony state: %v", err)
	}
	reconcileCompletedBuildTasks(&state, 1, dispatches)

	for _, task := range state.Plan.Phases[0].Tasks {
		if task.Status == colony.TaskCompleted {
			t.Fatalf("task %s is %q after a failed dispatch with no receipts -- touched files and a summary naming the task are not evidence (D-08)", *task.ID, task.Status)
		}
	}
}
