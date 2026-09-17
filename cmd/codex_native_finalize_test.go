package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func nativeFinalizeFixture(t *testing.T, workers int) (string, codexBuildManifest, []codexNativeWorkerRequest) {
	t.Helper()
	root := setupExternalBuildAttemptTest(t)
	if workers == 2 {
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			t.Fatal(err)
		}
		id := "1.2"
		state.Plan.Phases[0].Tasks = append(state.Plan.Phases[0].Tasks, colony.Task{ID: &id, Goal: "Write independent second evidence", Status: colony.TaskPending})
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatal(err)
		}
	}
	manifest := prepareBoundBuildManifestOnly(t, root)
	if len(manifest.Dispatches) != workers {
		t.Fatalf("runtime planned %d workers, want %d", len(manifest.Dispatches), workers)
	}
	return root, manifest, nativeFinalizeReserve(t, manifest)
}

func nativeFinalizeReserve(t *testing.T, manifest codexBuildManifest) []codexNativeWorkerRequest {
	t.Helper()
	root, err := filepath.EvalSymlinks(manifest.Root)
	if err != nil {
		t.Fatal(err)
	}
	requests := make([]codexNativeWorkerRequest, len(manifest.Dispatches))
	for i, d := range manifest.Dispatches {
		r := codexNativeWorkerRequest{SchemaVersion: 1, Phase: manifest.Phase, ExecutionBinding: *manifest.ExecutionBinding, WorkerName: d.Name, TaskID: normalizedDispatchTaskID(d), HostSessionID: "finalize-host", Workspace: root, HostPermission: "workspace_write"}
		r = nativeReserveForTest(t, r)
		r.ChildID = fmt.Sprintf("finalize-child-%d", i)
		if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, r)); err != nil {
			t.Fatal(err)
		}
		requests[i] = r
	}
	return requests
}

func nativeFinalizeRecord(t *testing.T, manifest codexBuildManifest, request codexNativeWorkerRequest, status string) {
	t.Helper()
	request = nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, status)
	if _, err := runCodexNativeWorker("record", nativeRequestPath(t, request)); err != nil {
		t.Fatal(err)
	}
}

func nativeFinalizeProjection(t *testing.T) (string, codexExternalBuildCompletion) {
	t.Helper()
	path, record, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("missing native attempt")
	}
	packet, complete := buildCompletionFromWorkerRuns(record)
	if !complete {
		t.Fatal("saved terminal results did not project")
	}
	return path, packet
}

func nativeFinalizeStateBytes(t *testing.T) []byte {
	t.Helper()
	raw, err := store.ReadFile("COLONY_STATE.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCodexNativeFinalizeBoundJournal(t *testing.T) {
	root, manifest, requests := nativeFinalizeFixture(t, 2)
	nativeFinalizeRecord(t, manifest, requests[0], "completed")
	before := nativeJournalBytes(t)
	if _, complete, err := stageBuildAttemptCompletionFromWorkerRuns(1, *manifest.ExecutionBinding); err != nil || complete {
		t.Fatalf("unfinished worker produced aggregate: complete=%t err=%v", complete, err)
	}
	if !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatal("incomplete stage changed accepted result")
	}
	nativeFinalizeRecord(t, manifest, requests[1], "completed")
	path, packet := nativeFinalizeProjection(t)
	if _, _, err := stageBuildAttemptCompletion(path, packet); err != nil {
		t.Fatal(err)
	}
	_, state, _, dispatches, err := runCodexBuildFinalize(root, 1, packet, true)
	if err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateBUILT || len(dispatches) != 2 {
		t.Fatalf("bound journal was not credited: %s %+v", state.State, dispatches)
	}
	for _, task := range state.Plan.Phases[0].Tasks {
		if task.Status != colony.TaskCompleted {
			t.Fatalf("missing credit: %+v", task)
		}
	}
}

func TestCodexNativeFinalizeRejectsMutation(t *testing.T) {
	for _, boundary := range []string{"stage", "finalize"} {
		for _, mutation := range []string{"summary", "receipt", "missing", "duplicate", "extra", "binding", "claims", "saved-digest", "saved-child", "saved-prompt", "saved-dispatch", "saved-result", "unfinished"} {
			t.Run(boundary+"/"+mutation, func(t *testing.T) {
				root, manifest, requests := nativeFinalizeFixture(t, 1)
				nativeFinalizeRecord(t, manifest, requests[0], "completed")
				path, packet := nativeFinalizeProjection(t)
				switch mutation {
				case "summary":
					packet.Dispatches[0].Summary = "caller replacement"
				case "receipt":
					packet.Dispatches[0].TaskReceipts[0].Summary = "caller receipt replacement"
				case "missing":
					packet.Dispatches = nil
				case "duplicate":
					packet.Dispatches = append(packet.Dispatches, packet.Dispatches[0])
				case "extra":
					extra := packet.Dispatches[0]
					extra.Name = "unassigned"
					packet.Dispatches = append(packet.Dispatches, extra)
				case "binding":
					packet.DispatchManifest.ExecutionBinding = nil
					packet.DispatchManifest.AttemptID = ""
					packet.DispatchManifest.AttemptPath = ""
				case "claims":
					packet.Claims = &codexBuildClaims{}
				default:
					var saved buildAttemptRecord
					if err := store.LoadJSON(path, &saved); err != nil {
						t.Fatal(err)
					}
					switch mutation {
					case "saved-digest":
						saved.WorkerRuns[0].ResultSHA256 = strings.Repeat("b", 64)
					case "saved-child":
						saved.WorkerRuns[0].Native.ChildID = "replacement-child"
					case "saved-prompt":
						saved.WorkerRuns[0].Native.Prompt += " replacement"
					case "saved-dispatch":
						saved.WorkerRuns[0].Native.DispatchSHA256 = strings.Repeat("b", 64)
					case "saved-result":
						saved.WorkerRuns[0].Result.Summary = "unhashed replacement"
					case "unfinished":
						saved.WorkerRuns[0].Native.LaunchState = "bound"
						saved.WorkerRuns[0].Status = buildWorkerDispatching
					}
					if err := store.SaveJSON(path, saved); err != nil {
						t.Fatal(err)
					}
				}
				journal, state := nativeJournalBytes(t), nativeFinalizeStateBytes(t)
				var err error
				if boundary == "stage" {
					_, _, err = stageBuildAttemptCompletion(path, packet)
				} else {
					_, _, _, _, err = runCodexBuildFinalize(root, 1, packet, true)
				}
				if err == nil {
					t.Fatalf("%s admitted %s", boundary, mutation)
				}
				if !bytes.Equal(journal, nativeJournalBytes(t)) || !bytes.Equal(state, nativeFinalizeStateBytes(t)) {
					t.Fatalf("%s refusal changed journal or credit", boundary)
				}
			})
		}
	}
}

func TestCodexNativeFinalizeReplay(t *testing.T) {
	root, manifest, requests := nativeFinalizeFixture(t, 1)
	nativeFinalizeRecord(t, manifest, requests[0], "completed")
	path, packet := nativeFinalizeProjection(t)
	completionPath, digest, err := stageBuildAttemptCompletion(path, packet)
	if err != nil {
		t.Fatal(err)
	}
	before := nativeJournalBytes(t)
	// Time is deliberately not frozen: equal replay must retain the accepted timestamp.
	time.Sleep(time.Millisecond)
	again, againDigest, err := stageBuildAttemptCompletion(path, packet)
	if err != nil || again != completionPath || againDigest != digest || !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatalf("stage replay changed durable identity: %v", err)
	}
	if _, _, _, _, err := runCodexBuildFinalize(root, 1, packet, true); err != nil {
		t.Fatal(err)
	}
	before, state := nativeJournalBytes(t), nativeFinalizeStateBytes(t)
	result, _, _, _, err := runCodexBuildFinalize(root, 1, packet, true)
	if err != nil || result["idempotent"] != true {
		t.Fatalf("finalize replay: %v %+v", err, result)
	}
	if !bytes.Equal(before, nativeJournalBytes(t)) || !bytes.Equal(state, nativeFinalizeStateBytes(t)) {
		t.Fatal("replay changed journal or credit")
	}
}

func TestCodexNativeFinalizeArrivalOrder(t *testing.T) {
	for _, order := range [][]int{{0, 1}, {1, 0}} {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			root, manifest, requests := nativeFinalizeFixture(t, 2)
			for _, i := range order {
				nativeFinalizeRecord(t, manifest, requests[i], "completed")
			}
			path, packet := nativeFinalizeProjection(t)
			for i, result := range packet.Dispatches {
				if result.Name != manifest.Dispatches[i].Name {
					t.Fatal("arrival reordered manifest")
				}
			}
			if _, _, err := stageBuildAttemptCompletion(path, packet); err != nil {
				t.Fatal(err)
			}
			_, _, _, dispatches, err := runCodexBuildFinalize(root, 1, packet, true)
			if err != nil {
				t.Fatal(err)
			}
			if got := completedBuildTaskIDs(dispatches); !reflect.DeepEqual(got, map[string]struct{}{"1.1": {}, "1.2": {}}) {
				t.Fatalf("arrival changed credit: %v", got)
			}
		})
	}
}

func TestCodexNativeFinalizePartial(t *testing.T) {
	for _, status := range []string{"failed", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "Native partial results keep exact receipts")
			requests := nativeFinalizeReserve(t, manifest)
			r := requests[0]
			r.SourceEventID, r.SourceEventSHA256 = "partial-event", strings.Repeat("a", 64)
			r.Result = &internalWorkerResult{Name: chain.Name, Caste: chain.Caste, TaskID: chain.TaskID, Status: status, Summary: "Stopped after the first four tasks", Handoff: codex.WorkerHandoff{VerificationStatus: "fail"}}
			for _, id := range ids[:4] {
				r.Result.TaskReceipts = append(r.Result.TaskReceipts, receiptForTask(t, root, id))
				r.Result.FilesModified = append(r.Result.FilesModified, taskFileName(id))
			}
			if _, err := runCodexNativeWorker("record", nativeRequestPath(t, r)); err != nil {
				t.Fatal(err)
			}
			path, packet := nativeFinalizeProjection(t)
			if status == "cancelled" && packet.Dispatches[0].Status != "interrupted" {
				t.Fatal("cancellation was not projected to interrupted")
			}
			if _, _, err := stageBuildAttemptCompletion(path, packet); err != nil {
				t.Fatal(err)
			}
			result, state, _, dispatches, err := runCodexBuildFinalize(root, 1, packet, true)
			if err != nil {
				t.Fatal(err)
			}
			if state.State == colony.StateBUILT || len(completedBuildTaskIDs(dispatches)) != 4 || !allSelectedBuildTasksCredited(ids[:4], dispatches) {
				t.Fatalf("partial credit exaggerated result: %s %+v", state.State, dispatches)
			}
			if !reflect.DeepEqual(result["unfinished_task_ids"], ids[4:]) {
				t.Fatalf("wrong unfinished tasks: %+v", result)
			}
			var saved buildAttemptRecord
			if err := store.LoadJSON(path, &saved); err != nil {
				t.Fatal(err)
			}
			if saved.WorkerRuns[0].Result.Status != status {
				t.Fatal("projection rewrote native terminal")
			}
			journal, stateBytes := nativeJournalBytes(t), nativeFinalizeStateBytes(t)
			if _, _, _, _, err := runCodexBuildFinalize(root, 1, packet, true); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(journal, nativeJournalBytes(t)) || !bytes.Equal(stateBytes, nativeFinalizeStateBytes(t)) {
				t.Fatal("partial replay mutated accepted evidence")
			}
		})
	}
}

// JSON cloning deliberately uses the actual wire representation, as a fresh parent does.
func cloneNativeCompletion(t *testing.T, packet codexExternalBuildCompletion) codexExternalBuildCompletion {
	t.Helper()
	raw, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var result codexExternalBuildCompletion
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
