package cmd

import (
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
	secondRecovery, _ := secondResult["recovery_command"].(string)
	if secondRecovery != firstRecovery {
		t.Errorf("replay handed a different recovery command: %q, first time %q", secondRecovery, firstRecovery)
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
