package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

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

	rootCmd.SetArgs([]string{"build-finalize", "1", "--completion-file", completionPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build-finalize of a genuine partial returned an error: %v", err)
	}
	rootCmd.SetArgs([]string{})

	out := stdout.(*bytes.Buffer).String()
	for _, claim := range []string{
		"advance honestly",
		"External Task worker results recorded.",
	} {
		if strings.Contains(out, claim) {
			t.Errorf("a half-built phase still shows the ordinary finished-build line %q on the wrapper lane:\n%s", claim, out)
		}
	}
	for _, want := range append([]string{"--task " + pending[0]}, pending...) {
		if !strings.Contains(out, want) {
			t.Errorf("the wrapper lane's screen never mentions %q, so the owner is not told what is left or how to finish it:\n%s", want, out)
		}
	}
}
