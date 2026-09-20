package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// TestWrapperPartialFinalizeDoesNotShowTheFinishedBuildScreen is the permanent
// regression lock for NEW-03 (195-REVIEW.iter2.md).
//
// The "partly done" screen was added to the direct build lane only. The
// wrapper's own lane -- `aether build --plan-only`, spawn the workers, then
// `aether build-finalize` -- still rendered the ordinary finished-build
// screen after a half-built phase: it told the owner to run the check-and-
// advance command "and advance honestly", named no unfinished work, and never
// showed the command that finishes the rest. That is the lane the project's
// own guide documents as the primary one.
func TestWrapperPartialFinalizeDoesNotShowTheFinishedBuildScreen(t *testing.T) {
	root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "Partial finalize does not look finished")
	t.Setenv("AETHER_OUTPUT_MODE", "json")

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
		FilesCreated:  []string{},
		FilesModified: touchedFiles,
		TestsWritten:  []string{},
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "fail",
			CommandsRun:        []string{"go test ./..."},
		},
		TaskReceipts: receipts,
	}}
	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}

	data, err := json.MarshalIndent(completion, "", "  ")
	if err != nil {
		t.Fatalf("marshal completion: %v", err)
	}
	completionPath := filepath.Join(root, "partial-completion.json")
	if err := os.WriteFile(completionPath, data, 0o644); err != nil {
		t.Fatalf("write completion: %v", err)
	}

	result, state, phase, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("build-finalize of a genuine partial returned an error: %v", err)
	}
	partial, ok := result["partial_recovery"].(partialBuildRetryOutcome)
	if !ok {
		t.Fatalf("wrapper partial result has no typed partial_recovery projection: %#v", result["partial_recovery"])
	}
	wantCommand := "aether build 1 --force --task 1.5 --task 1.6"
	if !reflect.DeepEqual(partial.UnfinishedTaskIDs, pending) || partial.RedispatchCommand != wantCommand {
		t.Fatalf("typed wrapper partial projection = %+v, want unfinished=%v command=%q", partial, pending, wantCommand)
	}

	stdout = &bytes.Buffer{}
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	outputWorkflow(result, renderBuildPartialCreditResultVisual(state, phase, result))

	out := stdout.(*bytes.Buffer).String()
	for _, claim := range []string{
		"advance honestly",
		"External Task worker results recorded.",
	} {
		if strings.Contains(out, claim) {
			t.Errorf("a half-built phase still shows the ordinary finished-build line %q on the wrapper lane:\n%s", claim, out)
		}
	}
	for _, want := range append([]string{"$ant-build 1 --force --task 1.5 --task 1.6"}, pending...) {
		if !strings.Contains(out, want) {
			t.Errorf("the wrapper lane's screen never mentions %q, so the owner is not told what is left or how to finish it:\n%s", want, out)
		}
	}

	stdout = &bytes.Buffer{}
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	outputWorkflow(result, "visual output must not leak into JSON")
	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			RecoveryJob       bool                     `json:"recovery_job"`
			UnfinishedTaskIDs []string                 `json:"unfinished_task_ids"`
			RecoveryCommand   string                   `json:"recovery_command"`
			PartialRecovery   partialBuildRetryOutcome `json:"partial_recovery"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("wrapper partial JSON is invalid: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	if !envelope.OK || !envelope.Result.RecoveryJob {
		t.Fatalf("wrapper partial JSON lost its machine outcome: %+v", envelope)
	}
	if !reflect.DeepEqual(envelope.Result.UnfinishedTaskIDs, pending) || envelope.Result.RecoveryCommand != wantCommand {
		t.Fatalf("wrapper partial JSON top-level facts = %+v, want unfinished=%v command=%q", envelope.Result, pending, wantCommand)
	}
	if !reflect.DeepEqual(envelope.Result.PartialRecovery.UnfinishedTaskIDs, pending) || envelope.Result.PartialRecovery.RedispatchCommand != wantCommand {
		t.Fatalf("wrapper partial JSON typed facts = %+v, want unfinished=%v command=%q", envelope.Result.PartialRecovery, pending, wantCommand)
	}
}
