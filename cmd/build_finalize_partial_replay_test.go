package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestPartialFinalizeCanBeReplayedWithTheSamePacket is the permanent
// regression lock for the 195 review's second warning (WR-02).
//
// When a grouped job finishes some of its tasks and the journal write is lost,
// the runtime's own error message tells the caller to "rerun build-finalize
// with the same completion packet". For a partially credited build that
// instruction could not be followed: the partial terminal status was never
// added to the accepted-status list, so the identical packet was hard-refused
// with "build attempt <id> is partial and cannot be finalized", and there was
// no other route out.
//
// This test finalizes a four-of-six partial, then submits the exact same
// packet again and requires the second call to succeed, to change nothing, and
// to hand back the same recovery command.
func TestPartialFinalizeCanBeReplayedWithTheSamePacket(t *testing.T) {
	root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "Partial replay")

	proven := ids[:4]
	pending := ids[4:]

	receipts := make([]codex.TaskReceipt, 0, len(proven))
	touchedFiles := make([]string, 0, len(proven))
	for _, id := range proven {
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
	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}

	firstResult, firstState, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("first finalize of a genuine partial: %v", err)
	}
	firstRecovery, _ := firstResult["recovery_command"].(string)
	if strings.TrimSpace(firstRecovery) == "" {
		t.Fatal("a partial finalize handed the owner no recovery command")
	}
	childRel, child := partialRecoveryChild200(t, 1, manifest.AttemptID)
	if child.RecoveryCommand != firstRecovery {
		t.Fatalf("durable recovery command = %q, first finalize returned %q", child.RecoveryCommand, firstRecovery)
	}
	if !reflect.DeepEqual(child.SelectedTasks, pending) {
		t.Fatalf("durable unfinished task IDs = %v, want %v", child.SelectedTasks, pending)
	}
	firstProjection := partialRecoveryProjection200(firstResult)
	if firstProjection["retry_attempt_id"] != child.ID || firstProjection["retry_attempt_path"] != displayDataPath(childRel) {
		t.Fatalf("first finalize recovery projection = %+v, want child %s at %s", firstProjection, child.ID, displayDataPath(childRel))
	}

	// The same packet, submitted again -- exactly what the runtime's own
	// journal-failure message instructs.
	secondResult, secondState, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("resubmitting the identical completion packet for a partial build is refused, so the runtime's own \"rerun build-finalize with the same completion packet\" instruction cannot be followed: %v", err)
	}
	if secondState.State != firstState.State {
		t.Fatalf("replay changed colony state from %s to %s", firstState.State, secondState.State)
	}
	if idempotent, _ := secondResult["idempotent"].(bool); !idempotent {
		t.Errorf("replaying a partial packet must report itself as idempotent; got %#v", secondResult["idempotent"])
	}
	secondProjection := partialRecoveryProjection200(secondResult)
	if !reflect.DeepEqual(secondProjection, firstProjection) {
		t.Errorf("replay recovery projection = %+v, first finalize = %+v", secondProjection, firstProjection)
	}

	statusByID := map[string]string{}
	for _, task := range secondState.Plan.Phases[0].Tasks {
		statusByID[*task.ID] = string(task.Status)
	}
	for _, id := range proven {
		if statusByID[id] != string(colony.TaskCompleted) {
			t.Errorf("task %s is %q after replay, want it to stay %q", id, statusByID[id], colony.TaskCompleted)
		}
	}
	for _, id := range pending {
		if statusByID[id] == string(colony.TaskCompleted) {
			t.Errorf("task %s became %q on replay; a replay must credit nothing new", id, statusByID[id])
		}
	}
}

// TestPartialFinalizeReplayWritesNothing is the permanent regression lock for
// NEW-05 (195-REVIEW.iter2.md).
//
// Re-submitting an already-finalized packet is an inspection: it re-reports
// what was already decided. Its own documentation promised it "mutates
// nothing: no colony state write, no attempt transition, no new credit". That
// was only true while the recovery record from the first call still existed.
// If writing that record had failed the first time -- a warning to the screen,
// and the run continues -- the replay quietly created one, which is a write on
// a path this project's rules say must never write.
func TestPartialFinalizeReplayWritesNothing(t *testing.T) {
	root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "Replay writes nothing")

	proven := ids[:4]
	receipts := make([]codex.TaskReceipt, 0, len(proven))
	touchedFiles := make([]string, 0, len(proven))
	for _, id := range proven {
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
	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}

	firstResult, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("first finalize of a genuine partial: %v", err)
	}
	firstRecovery, _ := firstResult["recovery_command"].(string)
	if strings.TrimSpace(firstRecovery) == "" {
		t.Fatal("a partial finalize handed the owner no recovery command")
	}

	before := snapshotProjectDataTree(t, store.BasePath())

	secondResult, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("replaying the identical partial packet: %v", err)
	}

	after := snapshotProjectDataTree(t, store.BasePath())
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("identical partial replay mutated durable data\nbefore: %#v\nafter:  %#v", before, after)
	}
	secondRecovery, _ := secondResult["recovery_command"].(string)
	if secondRecovery != firstRecovery {
		t.Fatalf("the replay must still hand back the same recovery command; got %q, first time %q", secondRecovery, firstRecovery)
	}
}

func TestPartialFinalizeRejectsDifferentPacket200(t *testing.T) {
	root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "Changed partial packet fails closed")
	completion := partialCompletionPacket200(t, root, manifest, chain, ids[:4])

	firstResult, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("first finalize of a genuine partial: %v", err)
	}
	recoveryCommand, _ := firstResult["recovery_command"].(string)
	completion.Dispatches[0].Summary = "different packet with forged terminal prose"
	before := snapshotProjectDataTree(t, store.BasePath())

	result, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err == nil {
		t.Fatalf("changed partial packet was accepted: %+v", result)
	}
	if !strings.Contains(err.Error(), recoveryCommand) {
		t.Fatalf("changed-packet refusal omitted exact durable recovery guidance %q: %v", recoveryCommand, err)
	}
	after := snapshotProjectDataTree(t, store.BasePath())
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("changed-packet refusal mutated durable data\nbefore: %#v\nafter:  %#v", before, after)
	}
}

func TestPartialFinalizeRejectsMissingRecoveryChild200(t *testing.T) {
	root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "Missing partial child fails closed")
	completion := partialCompletionPacket200(t, root, manifest, chain, ids[:4])
	firstResult, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("first finalize of a genuine partial: %v", err)
	}
	recoveryCommand, _ := firstResult["recovery_command"].(string)
	childRel, child := partialRecoveryChild200(t, 1, manifest.AttemptID)
	if err := os.Remove(filepath.Join(store.BasePath(), filepath.FromSlash(childRel))); err != nil {
		t.Fatalf("remove recovery child to simulate missing durable evidence: %v", err)
	}
	if err := os.Remove(filepath.Join(store.BasePath(), filepath.FromSlash(buildStartReceiptPath(1, child.ID)))); err != nil {
		t.Fatalf("remove recovery child receipt: %v", err)
	}
	before := snapshotProjectDataTree(t, store.BasePath())

	result, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err == nil {
		t.Fatalf("partial replay accepted missing recovery evidence: %+v", result)
	}
	if !strings.Contains(err.Error(), "recovery") || !strings.Contains(err.Error(), recoveryCommand) {
		t.Fatalf("missing-child refusal omitted exact recovery guidance %q: %v", recoveryCommand, err)
	}
	after := snapshotProjectDataTree(t, store.BasePath())
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("missing-child refusal mutated durable data\nbefore: %#v\nafter:  %#v", before, after)
	}
}

func TestPartialFinalizeRejectsForgedRecoveryChild200(t *testing.T) {
	root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "Forged partial child fails closed")
	completion := partialCompletionPacket200(t, root, manifest, chain, ids[:4])
	firstResult, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("first finalize of a genuine partial: %v", err)
	}
	recoveryCommand, _ := firstResult["recovery_command"].(string)
	childRel, child := partialRecoveryChild200(t, 1, manifest.AttemptID)
	child.SelectedTasks = []string{ids[0]}
	child.RecoveryCommand = buildUnfinishedRetryRedispatchCommand(1, child.SelectedTasks)
	if err := store.SaveJSON(childRel, child); err != nil {
		t.Fatalf("forge recovery child: %v", err)
	}
	before := snapshotProjectDataTree(t, store.BasePath())

	result, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err == nil {
		t.Fatalf("partial replay trusted forged recovery child: %+v", result)
	}
	if !strings.Contains(err.Error(), "recovery") || !strings.Contains(err.Error(), recoveryCommand) {
		t.Fatalf("forged-child refusal omitted exact recovery guidance %q: %v", recoveryCommand, err)
	}
	after := snapshotProjectDataTree(t, store.BasePath())
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("forged-child refusal mutated durable data\nbefore: %#v\nafter:  %#v", before, after)
	}
}

func TestPartialFinalizeRejectsExhaustedRecoveryChild200(t *testing.T) {
	root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "Exhausted partial child fails closed")
	completion := partialCompletionPacket200(t, root, manifest, chain, ids[:4])
	firstResult, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("first finalize of a genuine partial: %v", err)
	}
	recoveryCommand, _ := firstResult["recovery_command"].(string)
	childRel, child := partialRecoveryChild200(t, 1, manifest.AttemptID)
	if err := transitionBuildAttempt(childRel, buildAttemptFailed, "recovery attempt exhausted", child.Dispatches, nil, child.ExecutionOwner, os.ErrDeadlineExceeded); err != nil {
		t.Fatalf("exhaust recovery child: %v", err)
	}
	before := snapshotProjectDataTree(t, store.BasePath())

	result, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err == nil {
		t.Fatalf("partial replay treated an exhausted recovery child as executable: %+v", result)
	}
	if !strings.Contains(err.Error(), "exhausted") || !strings.Contains(err.Error(), recoveryCommand) {
		t.Fatalf("exhausted-child refusal omitted exact recovery guidance %q: %v", recoveryCommand, err)
	}
	after := snapshotProjectDataTree(t, store.BasePath())
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("exhausted-child refusal mutated durable data\nbefore: %#v\nafter:  %#v", before, after)
	}
}

func partialCompletionPacket200(t *testing.T, root string, manifest codexBuildManifest, chain codexBuildDispatch, proven []string) codexExternalBuildCompletion {
	t.Helper()
	receipts := make([]codex.TaskReceipt, 0, len(proven))
	touchedFiles := make([]string, 0, len(proven))
	for _, id := range proven {
		receipts = append(receipts, receiptForTask(t, root, id))
		touchedFiles = append(touchedFiles, taskFileName(id))
	}
	return codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: []codexExternalBuildWorkerResult{{
		Stage: chain.Stage, Wave: chain.Wave, ExecutionWave: normalizedDispatchWave(chain),
		Caste: chain.Caste, Name: chain.Name, TaskID: chain.TaskID,
		Status: "failed", Summary: "crashed after finishing four of six steps", FilesModified: touchedFiles,
		Handoff:      codex.WorkerHandoff{VerificationStatus: "fail", CommandsRun: []string{"go test ./..."}},
		TaskReceipts: receipts,
	}}}
}

func partialRecoveryChild200(t *testing.T, phaseNum int, parentAttemptID string) (string, buildAttemptRecord) {
	t.Helper()
	rel, child, ok := findExistingBuildAttemptRetry(phaseNum, parentAttemptID)
	if !ok {
		t.Fatalf("no durable recovery child linked to parent %s", parentAttemptID)
	}
	return rel, child
}

func partialRecoveryProjection200(result map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"recovery_job":        result["recovery_job"],
		"parent_attempt_id":   result["parent_attempt_id"],
		"retry_attempt_id":    result["retry_attempt_id"],
		"retry_attempt_path":  result["retry_attempt_path"],
		"unfinished_task_ids": result["unfinished_task_ids"],
		"recovery_command":    result["recovery_command"],
		"next":                result["next"],
	}
}
