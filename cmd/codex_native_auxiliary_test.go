package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Use the real Queen-proposed Watcher dispatch. Reviewers have an auxiliary
// identity derived from their stage/caste/name, not a numbered plan task ID.
func nativeAuxiliaryFixture(t *testing.T) (string, codexBuildManifest, []codexNativeWorkerRequest) {
	t.Helper()
	root := setupExternalBuildAttemptTest(t)
	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{
		QueenCastes: []string{"builder", "watcher"}, QueenCasteWhy: []string{"watcher=independent verification before this lands"},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	if len(manifest.Dispatches) != 2 || manifest.Dispatches[0].Caste != "builder" || manifest.Dispatches[1].Caste != "watcher" || manifest.Dispatches[1].TaskID != "" {
		t.Fatalf("expected actual Builder and auxiliary Watcher: %+v", manifest.Dispatches)
	}
	workspace, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	requests := make([]codexNativeWorkerRequest, len(manifest.Dispatches))
	for i, d := range manifest.Dispatches {
		requests[i] = codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *manifest.ExecutionBinding,
			WorkerName: d.Name, TaskID: normalizedDispatchTaskID(d), HostSessionID: "auxiliary-host", Workspace: workspace, HostPermission: "workspace_write"}
	}
	return root, manifest, requests
}

func TestCodexNativeAuxiliaryWatcherLifecycle(t *testing.T) {
	root, manifest, requests := nativeAuxiliaryFixture(t)
	manifestBefore, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	before := nativeJournalBytes(t)
	if _, err := runCodexNativeWorker("reserve", nativeRequestPath(t, requests[1])); err == nil || !strings.Contains(err.Error(), "earlier execution wave is unfinished") {
		t.Fatalf("Watcher must wait for the Builder wave: %v", err)
	}
	if !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatal("premature Watcher reservation wrote to the attempt")
	}
	for i, request := range requests {
		request = nativeReserveForTest(t, request)
		request.ChildID = "auxiliary-child-" + request.WorkerName
		if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err != nil {
			t.Fatal(err)
		}
		request = nativeTerminalRequestForTest(t, request, manifest.Dispatches[i].Caste, "completed")
		if i == 1 {
			// A review result reports verification without claiming Builder edits
			// or inventing a numbered plan task for the auxiliary assignment.
			request.Result.FilesCreated = nil
			request.Result.Handoff.ChangedFiles = nil
			request.Result.TaskReceipts = nil
		}
		if _, err := runCodexNativeWorker("record", nativeRequestPath(t, request)); err != nil {
			t.Fatal(err)
		}
		before = nativeJournalBytes(t)
		if _, err := runCodexNativeWorker("record", nativeRequestPath(t, request)); err != nil {
			t.Fatalf("terminal replay refused: %v", err)
		}
		if !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("terminal replay mutated the attempt")
		}
	}
	stage, err := runCodexNativeWorker("stage", nativeRequestPath(t, codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *manifest.ExecutionBinding}))
	if err != nil || !stage.Complete {
		t.Fatalf("auxiliary terminal results did not stage: %+v %v", stage, err)
	}
	packet, err := loadExternalBuildCompletion(filepath.Join(root, filepath.FromSlash(stage.CompletionPath)))
	if err != nil {
		t.Fatal(err)
	}
	_, state, _, dispatches, err := runCodexBuildFinalize(root, 1, packet, true)
	if err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateBUILT || len(dispatches) != 2 || state.Plan.Phases[0].Tasks[0].Status != colony.TaskCompleted {
		t.Fatalf("Builder and auxiliary review were not finalized: state=%s dispatches=%+v", state.State, dispatches)
	}
	_, saved, ok := loadLatestBuildAttempt(1)
	if !ok || len(saved.WorkerRuns) != 2 || saved.WorkerRuns[1].TaskID != requests[1].TaskID {
		t.Fatalf("auxiliary identity was lost: %+v", saved.WorkerRuns)
	}
	manifestAfter, err := json.Marshal(saved.PlanManifest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Fatal("native lifecycle rewrote the accepted manifest")
	}
}

func TestCodexNativeAuxiliaryIdentityRefusals(t *testing.T) {
	for _, mutation := range []string{"empty-task", "wrong-task", "missing-worker", "wrong-worker"} {
		t.Run(mutation, func(t *testing.T) {
			_, _, requests := nativeAuxiliaryFixture(t)
			request := requests[1]
			wantError := "native assignment is not in the accepted manifest"
			switch mutation {
			case "empty-task":
				request.TaskID = ""
				wantError = "native worker requires worker_name, task_id and host_session_id"
			case "wrong-task":
				request.TaskID = requests[0].TaskID
			case "missing-worker":
				request.WorkerName = ""
				wantError = "native worker requires worker_name, task_id and host_session_id"
			case "wrong-worker":
				request.WorkerName = requests[0].WorkerName
			}
			before := nativeJournalBytes(t)
			if _, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request)); err == nil || err.Error() != wantError {
				t.Fatalf("invalid auxiliary identity must fail its own guard, want %q: %v", wantError, err)
			}
			if !bytes.Equal(before, nativeJournalBytes(t)) {
				t.Fatal("invalid identity mutated the attempt")
			}
		})
	}
}
