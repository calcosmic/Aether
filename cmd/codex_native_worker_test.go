package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func nativeAdmissionFixture(t *testing.T) (codexBuildManifest, codexNativeWorkerRequest) {
	t.Helper()
	root := setupExternalBuildAttemptTest(t)
	manifest := prepareBoundBuildManifestOnly(t, root)
	root, _ = filepath.EvalSymlinks(root)
	d := manifest.Dispatches[0]
	return manifest, codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *manifest.ExecutionBinding, WorkerName: d.Name, TaskID: normalizedDispatchTaskID(d), HostSessionID: "admission-host", Workspace: root, HostPermission: "workspace_write"}
}

func nativeRequestPath(t *testing.T, request codexNativeWorkerRequest) string {
	t.Helper()
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return writeCodexNativeRequestForTest(t, value)
}

func nativeJournalBytes(t *testing.T) []byte {
	t.Helper()
	path, _, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("missing attempt")
	}
	raw, err := os.ReadFile(filepath.Join(store.BasePath(), path))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func nativeReserveForTest(t *testing.T, request codexNativeWorkerRequest) codexNativeWorkerRequest {
	t.Helper()
	response, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request))
	if err != nil {
		t.Fatal(err)
	}
	request.LaunchID, request.ChildID = response.Worker.ProviderRunID, "admission-child"
	request.DispatchSHA256, request.PromptSHA256 = response.Worker.Native.DispatchSHA256, response.Worker.Native.PromptSHA256
	return request
}

func TestCodexNativeWorkerAdmission(t *testing.T) {
	t.Run("idle-pause-derivation-precedes-admission", func(t *testing.T) {
		_, request := nativeAdmissionFixture(t)
		path := nativeRequestPath(t, request)
		ready, release := make(chan struct{}), make(chan struct{})
		pauseResumeLifecycleFault = func(point string) error {
			if point == "after_validation" {
				close(ready)
				<-release
			}
			return nil
		}
		t.Cleanup(func() { pauseResumeLifecycleFault = nil })
		pauseResult := make(chan error, 1)
		go func() { _, err := pauseColonyAt(time.Now()); pauseResult <- err }()
		<-ready
		started, result := make(chan struct{}), make(chan error, 1)
		go func() {
			_, err := runCodexNativeWorkerWithHooks("reserve", path, codexNativeWorkerHooks{BeforeWrite: func() { close(started) }})
			result <- err
		}()
		<-started
		close(release)
		if err := <-pauseResult; err != nil {
			t.Fatal(err)
		}
		if err := <-result; err == nil {
			t.Fatal("native launch crossed an already-derived idle pause")
		}
		_, record, _ := loadLatestBuildAttempt(1)
		if len(record.WorkerRuns) != 0 {
			t.Fatal("paused admission wrote a reservation")
		}
	})
	t.Run("pause-cannot-cross-currency-read", func(t *testing.T) {
		_, request := nativeAdmissionFixture(t)
		ready, release := make(chan struct{}), make(chan struct{})
		path := nativeRequestPath(t, request)
		finished := make(chan error, 1)
		go func() {
			_, err := runCodexNativeWorkerWithHooks("reserve", path, codexNativeWorkerHooks{AfterCurrencyCheck: func() { close(ready); <-release }})
			finished <- err
		}()
		<-ready
		pauseStarted, paused := make(chan struct{}), make(chan error, 1)
		go func() { close(pauseStarted); _, err := pauseColonyAt(time.Now()); paused <- err }()
		<-pauseStarted
		close(release)
		if err := <-finished; err != nil {
			t.Fatal(err)
		}
		var pending pauseBoundaryPendingError
		if err := <-paused; !errors.As(err, &pending) {
			t.Fatalf("pause crossed native admission: %v", err)
		}
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			t.Fatal(err)
		}
		if state.Paused {
			t.Fatal("idle pause committed over a new reservation")
		}
	})
	t.Run("pause-before-reserve-or-release", func(t *testing.T) {
		for _, operation := range []string{"reserve", "bind"} {
			t.Run(operation, func(t *testing.T) {
				_, request := nativeAdmissionFixture(t)
				if operation == "bind" {
					request = nativeReserveForTest(t, request)
				}
				before := nativeJournalBytes(t)
				_, err := runCodexNativeWorkerWithHooks(operation, nativeRequestPath(t, request), codexNativeWorkerHooks{BeforeWrite: func() {
					// The already accepted pause state wins before this request takes
					// its repository session. Bind must refuse even with a valid child.
					var state colony.ColonyState
					if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
						t.Fatal(err)
					}
					state.Paused = true
					if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
						t.Fatal(err)
					}
				}})
				if err == nil {
					t.Fatal("paused request admitted")
				}
				if !bytes.Equal(before, nativeJournalBytes(t)) {
					t.Fatal("paused request wrote")
				}
			})
		}
	})
	t.Run("simultaneous-reservations", func(t *testing.T) {
		_, request := nativeAdmissionFixture(t)
		path := nativeRequestPath(t, request)
		ready, release := make(chan struct{}, 2), make(chan struct{})
		type outcome struct {
			response codexNativeWorkerResponse
			err      error
		}
		results := make(chan outcome, 2)
		for i := 0; i < 2; i++ {
			go func() {
				response, err := runCodexNativeWorkerWithHooks("reserve", path, codexNativeWorkerHooks{BeforeWrite: func() { ready <- struct{}{}; <-release }})
				results <- outcome{response, err}
			}()
		}
		<-ready
		<-ready
		close(release)
		launches, replays := 0, 0
		var launch string
		for i := 0; i < 2; i++ {
			got := <-results
			if got.err != nil {
				t.Fatal(got.err)
			}
			if got.response.LaunchAllowed {
				launches++
			}
			if got.response.Replay {
				replays++
			}
			if launch != "" && launch != got.response.Worker.ProviderRunID {
				t.Fatal("two launch identities")
			}
			launch = got.response.Worker.ProviderRunID
		}
		if launches != 1 || replays != 1 {
			t.Fatalf("launches=%d replays=%d", launches, replays)
		}
	})
	for _, change := range []string{"manifest", "attempt-pointer", "pause"} {
		t.Run("under-lock-"+change, func(t *testing.T) {
			_, request := nativeAdmissionFixture(t)
			path, record, _ := loadLatestBuildAttempt(1)
			var before []byte
			_, err := runCodexNativeWorkerWithHooks("reserve", nativeRequestPath(t, request), codexNativeWorkerHooks{BeforeWrite: func() {
				switch change {
				case "manifest":
					record.PlanManifest.Dispatches[0].Caste = "watcher"
					if err := store.SaveJSON(path, record); err != nil {
						t.Fatal(err)
					}
				case "attempt-pointer":
					var pointer latestBuildAttemptPointer
					if err := store.LoadJSON(latestBuildAttemptPointerPath(1), &pointer); err != nil {
						t.Fatal(err)
					}
					pointer.AttemptID = "attempt-superseded"
					if err := store.SaveJSON(latestBuildAttemptPointerPath(1), pointer); err != nil {
						t.Fatal(err)
					}
				case "pause":
					var state colony.ColonyState
					if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
						t.Fatal(err)
					}
					state.Paused = true
					if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
						t.Fatal(err)
					}
				}
				before, _ = os.ReadFile(filepath.Join(store.BasePath(), path))
			}})
			if err == nil {
				t.Fatal("prevalidated stale request was accepted")
			}
			after, _ := os.ReadFile(filepath.Join(store.BasePath(), path))
			if !bytes.Equal(before, after) {
				t.Fatal("stale request changed attempt")
			}
		})
	}
	for _, field := range []string{"binding", "run", "attempt", "manifest", "owner", "workspace", "worker", "task", "host", "prompt"} {
		t.Run("identity-"+field, func(t *testing.T) {
			_, request := nativeAdmissionFixture(t)
			request = nativeReserveForTest(t, request)
			switch field {
			case "binding":
				request.ExecutionBinding = codex.ExecutionBinding{}
			case "run":
				request.ExecutionBinding.RunID = "run-wrong"
			case "attempt":
				request.ExecutionBinding.AttemptID = "attempt-wrong"
			case "manifest":
				request.ExecutionBinding.ManifestSHA256 = strings.Repeat("a", 64)
			case "owner":
				request.ExecutionBinding.ExecutionOwner = "other-owner"
			case "workspace":
				request.ExecutionBinding.WorkspaceFingerprint = strings.Repeat("a", 64)
			case "worker":
				request.WorkerName = "unassigned"
			case "task":
				request.TaskID = "unassigned"
			case "host":
				request.HostSessionID = "other-host"
			case "prompt":
				request.PromptSHA256 = strings.Repeat("a", 64)
			}
			before := nativeJournalBytes(t)
			if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err == nil {
				t.Fatal("identity substitution accepted")
			}
			if !bytes.Equal(before, nativeJournalBytes(t)) {
				t.Fatal("identity refusal wrote")
			}
		})
	}
	for _, field := range []string{"bind-workspace", "bind-permission", "reserve-prompt", "manifest-caste", "manifest-wave", "manifest-coverage", "manifest-empty", "paused-bind"} {
		t.Run(field, func(t *testing.T) {
			_, request := nativeAdmissionFixture(t)
			operation := "bind"
			if field == "reserve-prompt" {
				operation, request.PromptSHA256 = "reserve", strings.Repeat("a", 64)
			} else {
				request = nativeReserveForTest(t, request)
			}
			switch field {
			case "bind-workspace":
				request.Workspace = filepath.Dir(request.Workspace)
			case "bind-permission":
				request.HostPermission = "repository_read_only"
			case "manifest-caste", "manifest-wave", "manifest-coverage", "manifest-empty", "paused-bind":
				path, record, _ := loadLatestBuildAttempt(1)
				switch field {
				case "manifest-caste":
					record.PlanManifest.Dispatches[0].Caste = "watcher"
				case "manifest-wave":
					record.PlanManifest.Dispatches[0].ExecutionWave++
				case "manifest-coverage":
					record.PlanManifest.Dispatches[0].CoveredTaskIDs = []string{"unassigned"}
				case "manifest-empty":
					record.PlanManifest.Dispatches = nil
				case "paused-bind":
					record.Status = buildAttemptInterrupted
				}
				if err := store.SaveJSON(path, record); err != nil {
					t.Fatal(err)
				}
			}
			before := nativeJournalBytes(t)
			if _, err := runCodexNativeWorker(operation, nativeRequestPath(t, request)); err == nil {
				t.Errorf("accepted %s", field)
			}
			if !bytes.Equal(before, nativeJournalBytes(t)) {
				t.Errorf("%s mutated attempt", field)
			}
		})
	}
}

func TestCodexNativeWorkerLaneSeparation(t *testing.T) {
	manifest, request := nativeAdmissionFixture(t)
	request = nativeReserveForTest(t, request)
	path, record, _ := loadLatestBuildAttempt(1)
	record.WorkerRuns[0].StartedAt = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	if err := store.SaveJSON(path, record); err != nil {
		t.Fatal(err)
	}
	before := nativeJournalBytes(t)
	if _, err := beginBuildAttemptWorkerRun(1, request.ExecutionBinding, internalWorkerDispatchRequest{WorkerName: request.WorkerName, TaskID: request.TaskID, Caste: manifest.Dispatches[0].Caste}, "other-provider", codex.PlatformFake); err == nil {
		t.Fatal("native reservation granted provider launch")
	}
	if !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatal("provider refusal changed native reservation")
	}
}

func nativeBoundForTest(t *testing.T) (codexBuildManifest, codexNativeWorkerRequest) {
	t.Helper()
	manifest, request := nativeAdmissionFixture(t)
	request = nativeReserveForTest(t, request)
	if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err != nil {
		t.Fatal(err)
	}
	return manifest, request
}

func nativeTerminalRequestForTest(t *testing.T, request codexNativeWorkerRequest, caste, status string) codexNativeWorkerRequest {
	t.Helper()
	if err := os.WriteFile(filepath.Join(request.Workspace, "evidence.txt"), []byte("saved native work\n"), 0600); err != nil {
		t.Fatal(err)
	}
	request.SourceEventID, request.SourceEventSHA256 = "terminal-event", strings.Repeat("a", 64)
	request.Result = &internalWorkerResult{Name: request.WorkerName, Caste: caste, TaskID: request.TaskID, Status: status, Summary: "Saved outcome", FilesCreated: []string{"evidence.txt"}, Handoff: codex.WorkerHandoff{ChangedFiles: []string{"evidence.txt"}, VerificationStatus: "pass"}, TaskReceipts: []codex.TaskReceipt{{TaskID: request.TaskID, Status: "completed", Summary: "saved task receipt", FilesCreated: []string{"evidence.txt"}, FilesModified: []string{}, TestsWritten: []string{}, Handoff: codex.WorkerHandoff{VerificationStatus: "pass", ChangedFiles: []string{"evidence.txt"}}}}}
	if status != "completed" {
		request.Result.Error = "original failure detail"
		request.Result.Blockers = []string{"original blocker"}
	}
	return request
}

func TestCodexNativeWorkerTerminal(t *testing.T) {
	t.Run("unassigned-task-receipt", func(t *testing.T) {
		manifest, request := nativeBoundForTest(t)
		request = nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "failed")
		request.Result.TaskReceipts[0].TaskID = "unassigned"
		before := nativeJournalBytes(t)
		if _, err := runCodexNativeWorker("record", nativeRequestPath(t, request)); err == nil {
			t.Fatal("unassigned receipt became durable")
		}
		if !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("unassigned receipt wrote")
		}
	})
	t.Run("concurrent-replay-and-late-pause", func(t *testing.T) {
		manifest, request := nativeBoundForTest(t)
		request = nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "failed")
		path := nativeRequestPath(t, request)
		ready, release := make(chan struct{}, 2), make(chan struct{})
		type outcome struct {
			response codexNativeWorkerResponse
			err      error
		}
		results := make(chan outcome, 2)
		for i := 0; i < 2; i++ {
			go func() {
				response, err := runCodexNativeWorkerWithHooks("record", path, codexNativeWorkerHooks{BeforeWrite: func() { ready <- struct{}{}; <-release }})
				results <- outcome{response, err}
			}()
		}
		<-ready
		<-ready
		// Pause does not erase an already returned child's evidence. The
		// current assignment remains eligible for terminal persistence only.
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			t.Fatal(err)
		}
		state.Paused = true
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatal(err)
		}
		close(release)
		accepted, replays := 0, 0
		var receipt *codexNativeWorkerReceipt
		for i := 0; i < 2; i++ {
			got := <-results
			if got.err != nil {
				t.Fatal(got.err)
			}
			if got.response.Replay {
				replays++
			} else {
				accepted++
			}
			if receipt != nil && !reflect.DeepEqual(receipt, got.response.Receipt) {
				t.Fatal("concurrent terminal minted new receipt")
			}
			receipt = got.response.Receipt
		}
		if accepted != 1 || replays != 1 {
			t.Fatalf("accepted=%d replays=%d", accepted, replays)
		}
	})
	t.Run("superseded-under-write-lock", func(t *testing.T) {
		manifest, request := nativeBoundForTest(t)
		request = nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "failed")
		path, _, _ := loadLatestBuildAttempt(1)
		before := nativeJournalBytes(t)
		_, err := runCodexNativeWorkerWithHooks("record", nativeRequestPath(t, request), codexNativeWorkerHooks{BeforeWrite: func() {
			var pointer latestBuildAttemptPointer
			if err := store.LoadJSON(latestBuildAttemptPointerPath(1), &pointer); err != nil {
				t.Fatal(err)
			}
			pointer.AttemptID = "attempt-superseded"
			if err := store.SaveJSON(latestBuildAttemptPointerPath(1), pointer); err != nil {
				t.Fatal(err)
			}
		}})
		if err == nil {
			t.Fatal("superseded terminal accepted")
		}
		after, _ := os.ReadFile(filepath.Join(store.BasePath(), path))
		if !bytes.Equal(before, after) {
			t.Fatal("superseded terminal wrote")
		}
	})
	for _, status := range []string{"completed", "failed", "blocked", "timeout", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			manifest, request := nativeBoundForTest(t)
			request = nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, status)
			first, err := runCodexNativeWorker("record", nativeRequestPath(t, request))
			if err != nil {
				t.Fatal(err)
			}
			before := nativeJournalBytes(t)
			if err := os.Remove(filepath.Join(request.Workspace, "evidence.txt")); err != nil {
				t.Fatal(err)
			}
			reopened, err := storage.NewStore(store.BasePath())
			if err != nil {
				t.Fatal(err)
			}
			store = reopened
			second, err := runCodexNativeWorker("record", nativeRequestPath(t, request))
			if err != nil || !second.Replay || first.Worker.ResultSHA256 != second.Worker.ResultSHA256 || !bytes.Equal(before, nativeJournalBytes(t)) {
				t.Fatalf("terminal replay changed: %v", err)
			}
			if !reflect.DeepEqual(first.Worker.Result, second.Worker.Result) || len(second.Worker.Result.TaskReceipts) != 1 {
				t.Fatal("result evidence lost")
			}
			request.Result.Summary = "conflict"
			if _, err := runCodexNativeWorker("record", nativeRequestPath(t, request)); err == nil {
				t.Fatal("conflicting terminal accepted")
			}
			if !bytes.Equal(before, nativeJournalBytes(t)) {
				t.Fatal("conflicting terminal changed bytes")
			}
		})
	}
}

func nativeObservationPath(t *testing.T, request codexNativeWorkerRequest, status string, at time.Time) string {
	t.Helper()
	raw, _ := json.Marshal(request)
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	value["observation_status"], value["observed_at"], value["observation_detail"] = status, at.UTC().Format(time.RFC3339Nano), "actual host observation fixture"
	value["source_event_id"], value["source_event_sha256"] = status+at.UTC().Format(time.RFC3339Nano), strings.Repeat("b", 64)
	return writeCodexNativeRequestForTest(t, value)
}

func TestCodexNativeWorkerAmbiguousLaunch(t *testing.T) {
	_, request := nativeAdmissionFixture(t)
	request = nativeReserveForTest(t, request)
	request.ChildID = ""
	path := nativeObservationPath(t, request, "launch_unresolved", time.Now().UTC())
	if _, err := runCodexNativeWorker("observe", path); err != nil {
		t.Fatal(err)
	}
	before := nativeJournalBytes(t)
	_, record, _ := loadLatestBuildAttempt(1)
	if !buildWorkerRunStillActive(record.WorkerRuns[0], time.Now().Add(24*time.Hour)) {
		t.Fatal("elapsed time ended unresolved launch")
	}
	if err := cancelBuildAttemptWorkerRuns(func() string { p, _, _ := loadLatestBuildAttempt(1); return p }(), "pause"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatal("bulk cancel fabricated native termination")
	}
	reopened, err := storage.NewStore(store.BasePath())
	if err != nil {
		t.Fatal(err)
	}
	store = reopened
	if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "unavailable", time.Now().Add(time.Minute))); err != nil {
		t.Fatal(err)
	}
	before = nativeJournalBytes(t)
	minimal := codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: request.ExecutionBinding}
	inspected, err := runCodexNativeWorker("inspect", nativeRequestPath(t, minimal))
	if err != nil || len(inspected.WorkerStates) != 1 || inspected.WorkerStates[0].Terminal || inspected.WorkerStates[0].LaunchState != "launch_unresolved" || !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatalf("inspection lost unresolved state: %+v %v", inspected.WorkerStates, err)
	}
	if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "no_launch", time.Now().Add(2*time.Minute))); err != nil {
		t.Fatal(err)
	}
	_, record, _ = loadLatestBuildAttempt(1)
	if !codexNativeWorkerIsTerminal(record.WorkerRuns[0]) || record.WorkerRuns[0].Native.LaunchState != "no_launch" || record.WorkerRuns[0].Status != "cancelled" {
		t.Fatal("confirmed no launch remained open")
	}
	request.ChildID = "late-new-child"
	if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err == nil {
		t.Fatal("closed no-launch reservation reused")
	}
}

func TestCodexNativeWorkerCancellation(t *testing.T) {
	t.Run("transition-receipts-and-running-order", func(t *testing.T) {
		manifest, request := nativeAdmissionFixture(t)
		originalReserve := request
		var transitions []string
		hooks := codexNativeWorkerHooks{AfterTransition: func(receipt codexNativeWorkerReceipt) { transitions = append(transitions, receipt.Operation) }}
		reserved, err := runCodexNativeWorkerWithHooks("reserve", nativeRequestPath(t, request), hooks)
		if err != nil {
			t.Fatal(err)
		}
		request.LaunchID, request.ChildID = reserved.Worker.ProviderRunID, "receipt-child"
		request.DispatchSHA256, request.PromptSHA256 = reserved.Worker.Native.DispatchSHA256, reserved.Worker.Native.PromptSHA256
		bound, err := runCodexNativeWorkerWithHooks("bind", nativeRequestPath(t, request), hooks)
		if err != nil {
			t.Fatal(err)
		}
		at := time.Now().UTC()
		observation := nativeObservationPath(t, request, "running", at)
		if _, err := runCodexNativeWorkerWithHooks("observe", observation, hooks); err != nil {
			t.Fatal(err)
		}
		before := nativeJournalBytes(t)
		if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "running", at.Add(-time.Second))); err == nil {
			t.Fatal("out-of-order running accepted")
		}
		if !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("old running observation wrote")
		}
		terminalRequest := nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "completed")
		if _, err := runCodexNativeWorkerWithHooks("record", nativeRequestPath(t, terminalRequest), hooks); err != nil {
			t.Fatal(err)
		}
		for operation, req := range map[string]codexNativeWorkerRequest{"reserve": originalReserve, "bind": request, "record": terminalRequest} {
			response, err := runCodexNativeWorkerWithHooks(operation, nativeRequestPath(t, req), hooks)
			if err != nil || !response.Replay {
				t.Fatalf("%s replay: %v", operation, err)
			}
			if operation == "reserve" && !reflect.DeepEqual(reserved.Receipt, response.Receipt) {
				t.Fatal("reservation receipt changed after terminal")
			}
			if operation == "bind" && !reflect.DeepEqual(bound.Receipt, response.Receipt) {
				t.Fatal("binding receipt changed after terminal")
			}
		}
		if !reflect.DeepEqual(transitions, []string{"reserve", "bind", "observe", "record"}) {
			t.Fatalf("transition hooks replayed: %v", transitions)
		}
		if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "running", at.Add(time.Minute))); err == nil {
			t.Fatal("terminal regressed to running")
		}
	})
	_, request := nativeBoundForTest(t)
	path, _, _ := loadLatestBuildAttempt(1)
	before := nativeJournalBytes(t)
	if err := cancelBuildAttemptWorkerRuns(path, "paused"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatal("pause cancelled a native child without host evidence")
	}
	observation := nativeObservationPath(t, request, "cancel_requested", time.Now().UTC())
	response, err := runCodexNativeWorker("observe", observation)
	if err != nil {
		t.Fatal(err)
	}
	if response.Worker.Result != nil || response.Worker.CompletedAt != "" {
		t.Fatal("request was treated as completed cancellation")
	}
	before = nativeJournalBytes(t)
	if again, err := runCodexNativeWorker("observe", observation); err != nil || !again.Replay || !reflect.DeepEqual(response.Receipt, again.Receipt) || !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatalf("observation replay changed receipt: %v", err)
	}
	if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "running", time.Now().Add(time.Minute))); err == nil {
		t.Fatal("cancel request regressed to running")
	}
	if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "cancelled", time.Now().Add(2*time.Minute))); err != nil {
		t.Fatal(err)
	}
	_, saved, _ := loadLatestBuildAttempt(1)
	if !codexNativeWorkerIsTerminal(saved.WorkerRuns[0]) || saved.WorkerRuns[0].Status != "cancelled" {
		t.Fatal("confirmed cancellation was not saved")
	}
}

func TestCodexNativeWorkerArrivalOrder(t *testing.T) {
	for _, order := range [][]int{{0, 1}, {1, 0}} {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			root := setupExternalBuildAttemptTest(t)
			var state colony.ColonyState
			if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
				t.Fatal(err)
			}
			id := "1.2"
			state.Plan.Phases[0].Tasks = append(state.Plan.Phases[0].Tasks, colony.Task{ID: &id, Goal: "Write independent second evidence", Status: colony.TaskPending})
			if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
				t.Fatal(err)
			}
			manifest := prepareBoundBuildManifestOnly(t, root)
			if len(manifest.Dispatches) != 2 {
				t.Fatalf("want two runtime assignments: %+v", manifest.Dispatches)
			}
			root, _ = filepath.EvalSymlinks(root)
			requests := make([]codexNativeWorkerRequest, 2)
			for i, dispatch := range manifest.Dispatches {
				request := codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *manifest.ExecutionBinding, WorkerName: dispatch.Name, TaskID: dispatch.TaskID, HostSessionID: "arrival-host", Workspace: root, HostPermission: "workspace_write"}
				requests[i] = nativeReserveForTest(t, request)
				requests[i].ChildID = fmt.Sprintf("child-%d", i)
				if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, requests[i])); err != nil {
					t.Fatal(err)
				}
			}
			for _, i := range order {
				request := nativeTerminalRequestForTest(t, requests[i], manifest.Dispatches[i].Caste, "completed")
				if _, err := runCodexNativeWorker("record", nativeRequestPath(t, request)); err != nil {
					t.Fatal(err)
				}
			}
			_, saved, _ := loadLatestBuildAttempt(1)
			completion, ok := buildCompletionFromWorkerRuns(saved)
			if !ok || len(completion.Dispatches) != 2 {
				t.Fatal("lost terminal worker")
			}
			for i, dispatch := range completion.Dispatches {
				if dispatch.Name != manifest.Dispatches[i].Name || dispatch.TaskID != manifest.Dispatches[i].TaskID {
					t.Fatal("arrival order changed task identity")
				}
			}
		})
	}
}

func TestCodexNativeBuildGuidePlatformIsolation(t *testing.T) {
	baseline := map[string]commandGuideResult{}
	for _, platform := range []string{"codex", "claude", "opencode", "opencode", "codex", "claude", "codex"} {
		guide, err := buildCommandGuide("build", platform)
		if err != nil {
			t.Fatal(err)
		}
		if before, ok := baseline[platform]; ok && !reflect.DeepEqual(before, guide) {
			t.Fatalf("%s guide changed after another platform read", platform)
		}
		baseline[platform] = guide
		// Include every instruction-bearing field, including raw bypass text.
		all := strings.Join(append(append(append([]string{guide.Intent, guide.RunCommand, guide.RawBypass}, guide.PreSteps...), guide.PostSteps...), guide.DriftGuards...), "\n")
		if platform == "codex" {
			for _, required := range []string{"codex-native-worker reserve", "codex-native-worker bind", "codex-native-worker record", "codex-native-worker stage", "spawn_agent", "sole launcher"} {
				if !strings.Contains(all, required) {
					t.Errorf("Codex guide missing %q", required)
				}
			}
			if strings.Contains(all, "build-wave playbook") {
				t.Error("Codex guide still delegates authority to retired playbook")
			}
		} else {
			for _, forbidden := range []string{"codex-native-worker", "spawn_agent"} {
				if strings.Contains(all, forbidden) {
					t.Errorf("%s includes %s", platform, forbidden)
				}
			}
			if guide.SkillReference != "" || !strings.Contains(guide.PreSteps[0], "generated "+platform+" slash-command wrapper") {
				t.Errorf("%s lost wrapper entrypoint", platform)
			}
			for _, required := range []string{"visible live Task/subagent", "spawn-log", "spawn-complete", "build-completion-stage"} {
				if !strings.Contains(all, required) {
					t.Errorf("%s lost %q", platform, required)
				}
			}
			if guide.RunCommand != "AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path returned by build-completion-stage>" {
				t.Errorf("%s finalizer changed: %s", platform, guide.RunCommand)
			}
		}
	}
	// The adapter must not mutate caller-owned instruction slices either.
	def := commandGuideCatalog()["build"]
	before, _ := json.Marshal(def)
	_ = adaptCommandGuideDefinitionForPlatform("build", "codex", def)
	after, _ := json.Marshal(def)
	if string(before) != string(after) {
		t.Fatal("Codex adapter mutated shared definition")
	}
}

// A deterministic host-boundary double proves journal/credit behavior only;
// the separately opted-in fresh-host test is the native execution proof.
func TestCodexNativeWorkerTracer(t *testing.T) {
	source := filepath.Join(antSkillSourceRoot(t), "cmd", "codex_native_worker.go")
	syntax, err := parser.ParseFile(token.NewFileSet(), source, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, imported := range syntax.Imports {
		if imported.Path.Value == "\"os/exec\"" {
			t.Fatal("native bridge imports a process launcher")
		}
	}
	forbidden := map[string]bool{"invokeInternalWorker": true, "runInternalWorkerAdapter": true, "runInternalWorkerAdapterForPhase": true, "executeCodexBuildDispatches": true}
	ast.Inspect(syntax, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if name, ok := call.Fun.(*ast.Ident); ok && forbidden[name.Name] {
				t.Errorf("native bridge invokes second launcher %s", name.Name)
			}
		}
		return true
	})
	root := setupExternalBuildAttemptTest(t)
	manifest := prepareBoundBuildManifestOnly(t, root)
	if len(manifest.Dispatches) != 1 {
		t.Fatalf("ordinary fixture selected %d workers", len(manifest.Dispatches))
	}
	dispatch := manifest.Dispatches[0]
	root, _ = filepath.EvalSymlinks(root)
	request := codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *manifest.ExecutionBinding, WorkerName: dispatch.Name, TaskID: normalizedDispatchTaskID(dispatch), HostSessionID: "unit-host", Workspace: root, HostPermission: "workspace_write"}
	call := func(operation string, request codexNativeWorkerRequest, wantOK bool) codexNativeWorkerResponse {
		t.Helper()
		raw, _ := json.Marshal(request)
		var value map[string]interface{}
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		if len(request.RawResult) > 0 {
			value["result"] = json.RawMessage(request.RawResult)
		}
		path := writeCodexNativeRequestForTest(t, value)
		var output bytes.Buffer
		stdout, stderr = &output, &output
		resetFlags(rootCmd)
		rootCmd.SetArgs([]string{"codex-native-worker", operation, "--request", path})
		rootCmd.SetOut(&output)
		rootCmd.SetErr(&output)
		err := rootCmd.Execute()
		if wantOK && err != nil {
			t.Fatalf("%s: %v %s", operation, err, output.String())
		}
		var envelope struct {
			OK     bool                      `json:"ok"`
			Result codexNativeWorkerResponse `json:"result"`
		}
		if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
			t.Fatalf("%s decode: %v %s", operation, err, output.String())
		}
		if envelope.OK != wantOK {
			t.Fatalf("%s success=%v want=%v: %s", operation, envelope.OK, wantOK, output.String())
		}
		renderedCommandExitCode.Store(0)
		return envelope.Result
	}
	call("reserve", codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1}, false)
	for _, field := range []string{"workspace", "permission", "membership"} {
		refused := request
		switch field {
		case "workspace":
			refused.Workspace = filepath.Dir(root)
		case "permission":
			refused.HostPermission = "repository_read_only"
		case "membership":
			refused.TaskID = "not-assigned"
		}
		call("reserve", refused, false)
	}
	first := call("reserve", request, true)
	if !first.LaunchAllowed || first.Worker.Native == nil || first.Worker.ProviderRunID == "" || first.Worker.ProcessID != 0 {
		t.Fatalf("bad reservation: %+v", first)
	}
	if lifecycleDigest([]byte(first.Worker.Native.Prompt)) != first.Worker.Native.PromptSHA256 {
		t.Fatal("prompt digest differs from actual launch bytes")
	}
	attemptPath, _, _ := loadLatestBuildAttempt(1)
	snapshot := func() []byte {
		raw, err := os.ReadFile(filepath.Join(store.BasePath(), attemptPath))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	before := snapshot()
	replay := call("reserve", request, true)
	if replay.LaunchAllowed || !replay.Replay || replay.Worker.ProviderRunID != first.Worker.ProviderRunID || !bytes.Equal(before, snapshot()) {
		t.Fatal("reservation replay changed bytes or licensed a duplicate launch")
	}
	// A legacy subprocess may not replace this reserved native worker.
	if _, err := beginBuildAttemptWorkerRun(1, request.ExecutionBinding, internalWorkerDispatchRequest{WorkerName: dispatch.Name, TaskID: request.TaskID, Caste: dispatch.Caste}, "forbidden-provider", codex.PlatformCodex); err == nil {
		t.Fatal("provider route reclaimed a native reservation")
	}
	request.LaunchID, request.ChildID = first.Worker.ProviderRunID, "unit-child"
	request.DispatchSHA256, request.PromptSHA256 = first.Worker.Native.DispatchSHA256, first.Worker.Native.PromptSHA256
	request.Result = &internalWorkerResult{Name: dispatch.Name, Caste: dispatch.Caste, TaskID: request.TaskID, Status: "completed", Summary: "deterministic boundary result", FilesCreated: []string{"evidence.txt"}, Handoff: codex.WorkerHandoff{ChangedFiles: []string{"evidence.txt"}, CommandsRun: []string{"deterministic fixture boundary"}, VerificationStatus: "pass", NextWorkerInstructions: []string{"inspect saved deterministic receipt"}}}
	call("record", request, false) // no binding, no source event
	request.Result = nil
	bound := call("bind", request, true)
	if bound.Worker.Native.Release == "" {
		t.Fatal("bind returned no release")
	}
	before = snapshot()
	call("bind", request, true)
	if !bytes.Equal(before, snapshot()) {
		t.Fatal("bind replay wrote")
	}
	wrong := request
	wrong.ChildID = "other-child"
	call("bind", wrong, false)
	call("record", request, false) // deliberately empty
	if err := os.WriteFile(filepath.Join(root, "evidence.txt"), []byte("deterministic worker-boundary fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	request.SourceEventID = "unit-event"
	request.SourceEventSHA256 = strings.Repeat("a", 64)
	// The installed Builder uses ant_name/tdd/code_written. Exercise its wire
	// shape through the registered request reader, not the internal Go struct.
	wire := map[string]any{"schema_version": 1, "phase": 1, "execution_binding": request.ExecutionBinding,
		"worker_name": request.WorkerName, "task_id": request.TaskID, "host_session_id": request.HostSessionID,
		"launch_id": request.LaunchID, "child_id": request.ChildID, "dispatch_sha256": request.DispatchSHA256,
		"prompt_sha256": request.PromptSHA256, "source_event_id": request.SourceEventID, "source_event_sha256": request.SourceEventSHA256,
		"result": map[string]any{"ant_name": dispatch.Name, "caste": dispatch.Caste, "task_id": request.TaskID,
			"status": "code_written", "summary": "Builder wire regression", "files_created": []string{"evidence.txt"},
			"tdd":     map[string]any{"cycles_completed": 1, "tests_added": 0, "coverage_percent": nil, "all_passing": true},
			"handoff": codex.WorkerHandoff{ChangedFiles: []string{"evidence.txt"}, VerificationStatus: "pass", CommandsRun: []string{"fixture check"}}}}
	loaded, err := loadCodexNativeWorkerRequest(writeCodexNativeRequestForTest(t, wire))
	if err != nil || loaded.Result == nil || loaded.Result.Name != dispatch.Name || loaded.Result.Status != "completed" {
		t.Fatalf("installed Builder wire refused: %+v %v", loaded.Result, err)
	}
	request.Result = &internalWorkerResult{Name: dispatch.Name, Caste: dispatch.Caste, TaskID: request.TaskID, Status: "completed", Summary: "deterministic boundary result", FilesCreated: []string{"evidence.txt"}, Handoff: codex.WorkerHandoff{ChangedFiles: []string{"evidence.txt"}, CommandsRun: []string{"deterministic fixture boundary"}, VerificationStatus: "pass", NextWorkerInstructions: []string{"inspect saved deterministic receipt"}}}
	before = snapshot()
	invalid := *request.Result
	invalid.Handoff.VerificationStatus = "Tests passed after the edit"
	wrong = request
	wrong.Result = &invalid
	call("record", wrong, false)
	invalid.Handoff = codex.WorkerHandoff{}
	call("record", wrong, false)
	if !bytes.Equal(before, snapshot()) {
		t.Fatal("invalid handoff poisoned the immutable terminal journal")
	}
	// Bind only fixture identities. Preserve the captured Builder's actual
	// result shape, including aliases/TDD and the invalid prose handoff.
	var captured map[string]any
	if err := json.Unmarshal([]byte(nativeCapturedBuilderTerminal), &captured); err != nil {
		t.Fatal(err)
	}
	captured["name"], captured["ant_name"], captured["task_id"] = dispatch.Name, dispatch.Name, request.TaskID
	for _, v := range captured["task_receipts"].([]any) {
		v.(map[string]any)["task_id"] = request.TaskID
	}
	if err := os.WriteFile(filepath.Join(root, "clamp.go"), []byte("package fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	request.RawResult, _ = json.Marshal(captured)
	request.SourceEventID = "captured-invalid-first"
	request.SourceEventSHA256 = strings.Repeat("b", 64)
	call("record", request, false)
	if !bytes.Equal(before, snapshot()) {
		t.Fatal("captured malformed first result changed journal")
	}
	// The same child corrects its enum values in a NEW event; this is its first
	// accepted terminal, not replacement of an immutable accepted response.
	captured["handoff"].(map[string]any)["verification_status"] = "pass"
	for _, v := range captured["task_receipts"].([]any) {
		v.(map[string]any)["handoff"].(map[string]any)["verification_status"] = "pass"
	}
	for _, variant := range []string{"conflicting_alias", "unknown_field", "unknown_status"} {
		bad := map[string]any{}
		for k, v := range captured {
			bad[k] = v
		}
		switch variant {
		case "conflicting_alias":
			bad["ant_name"] = "another-worker"
		case "unknown_field":
			bad["fabricated"] = true
		case "unknown_status":
			bad["status"] = "magic"
		}
		wrong = request
		wrong.RawResult, _ = json.Marshal(bad)
		call("record", wrong, false)
		if !bytes.Equal(before, snapshot()) {
			t.Fatalf("%s changed bound journal", variant)
		}
	}
	request.RawResult, _ = json.Marshal(captured)
	request.SourceEventID = "captured-corrected-event"
	request.SourceEventSHA256 = strings.Repeat("c", 64)
	terminal := call("record", request, true)
	if terminal.Worker.ResultSHA256 == "" || terminal.Worker.Native.LaunchState != "terminal" {
		t.Fatal("terminal was not durable")
	}
	var persisted, submitted any
	_ = json.Unmarshal(terminal.Worker.Native.RawResult, &persisted)
	_ = json.Unmarshal(request.RawResult, &submitted)
	if !reflect.DeepEqual(persisted, submitted) || terminal.Worker.Result.Status != "completed" {
		t.Fatal("raw Builder evidence lost during normalization")
	}
	before = snapshot()
	call("record", request, true)
	if !bytes.Equal(before, snapshot()) {
		t.Fatal("terminal replay wrote")
	}
	changed := *request.Result
	changed.Summary = "conflicting result"
	wrong = request
	wrong.RawResult = nil
	wrong.Result = &changed
	call("record", wrong, false)
	minimal := codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: request.ExecutionBinding}
	inspected := call("inspect", minimal, true)
	if inspected.Complete || len(inspected.Workers) != 1 || inspected.Workers[0].ResultSHA256 != terminal.Worker.ResultSHA256 || !bytes.Equal(before, snapshot()) {
		t.Fatal("inspect changed journal or lost terminal-before-aggregate state")
	}
	staged := call("stage", minimal, true)
	completion, err := loadExternalBuildCompletion(filepath.Join(root, filepath.FromSlash(staged.CompletionPath)))
	if err != nil {
		t.Fatal(err)
	}
	firstFinal, state, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatal(err)
	}
	if firstFinal["idempotent"] != false || state.Plan.Phases[0].Tasks[0].Status != colony.TaskCompleted {
		t.Fatalf("no finalizer credit: %+v", firstFinal)
	}
	_, saved, _ := loadLatestBuildAttempt(1)
	eventCount, historyCount := len(state.Events), len(saved.History)
	again, state, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatal(err)
	}
	_, saved, _ = loadLatestBuildAttempt(1)
	if again["idempotent"] != true || len(saved.WorkerRuns) != 1 || len(saved.History) != historyCount || len(state.Events) != eventCount {
		t.Fatal("finalizer replay repeated worker/credit")
	}
}

// Exact first Brick-46 terminal from the immutable 2026-09-17 ordinary capture.
// Its prose verification statuses are intentionally invalid regression inputs.
const nativeCapturedBuilderTerminal = `{
  "name": "Brick-46",
  "ant_name": "Brick-46",
  "caste": "builder",
  "task_id": "1.1",
  "status": "code_written",
  "summary": "Fixed Clamp to return low below the lower boundary, high above the upper boundary, and the original value within bounds. Existing tests reproduced both failures before the edit and passed afterward. Build and vet also passed.",
  "files_created": [],
  "files_modified": ["clamp.go"],
  "tests_written": [],
  "tdd": {
    "cycles_completed": 1,
    "tests_added": 0,
    "coverage_percent": null,
    "all_passing": true
  },
  "blockers": [],
  "spawns": [],
  "task_receipts": [
    {
      "task_id": "1.1",
      "status": "code_written",
      "summary": "Implemented integer Clamp boundaries and verified all existing boundary cases.",
      "files_created": [],
      "files_modified": ["clamp.go"],
      "tests_written": [],
      "handoff": {
        "changed_files": ["clamp.go"],
        "commands_run": [
          "go test ./...",
          "GOCACHE=/private/tmp/aether-brick46-go-cache-run-c671353f97f806a6edf2ddba216822f0 go test ./...",
          "GOCACHE=/private/tmp/aether-brick46-go-cache-run-c671353f97f806a6edf2ddba216822f0 go build ./...",
          "GOCACHE=/private/tmp/aether-brick46-go-cache-run-c671353f97f806a6edf2ddba216822f0 go vet ./...",
          "git diff --check",
          "git diff -- clamp.go clamp_test.go go.mod"
        ],
        "verification_status": "Passed after edit: tests, build, vet, and diff checks. The initial test command could not access the inherited home/Library/Caches/go-build cache: operation not permitted. A permitted /private/tmp GOCACHE resolved that setup failure. Using that cache, pre-edit tests reproduced both incorrect results (-3 instead of 0 and 15 instead of 10); post-edit tests passed.",
        "known_failures": [],
        "open_decisions": [],
        "assumptions": [
          "The assigned behavior uses ordered lower and upper bounds.",
          "Existing tests provide the required regression evidence; clamp_test.go and go.mod remain unchanged."
        ],
        "next_worker_instructions": [
          "Submit the native worker evidence to the existing build finalizer."
        ],
        "do_not_repeat": [
          "Do not use the inaccessible inherited Go build cache.",
          "Do not modify clamp_test.go or go.mod."
        ],
        "freshness": "2026-09-17T07:27:55Z"
      }
    }
  ],
  "handoff": {
    "changed_files": ["clamp.go"],
    "commands_run": [
      "cat AGENTS.md clamp.go clamp_test.go go.mod",
      "git status --short",
      "go test ./...",
      "GOCACHE=/private/tmp/aether-brick46-go-cache-run-c671353f97f806a6edf2ddba216822f0 go test ./...",
      "GOCACHE=/private/tmp/aether-brick46-go-cache-run-c671353f97f806a6edf2ddba216822f0 go build ./...",
      "GOCACHE=/private/tmp/aether-brick46-go-cache-run-c671353f97f806a6edf2ddba216822f0 go vet ./...",
      "git diff --check",
      "git diff -- clamp.go clamp_test.go go.mod"
    ],
    "verification_status": "Native Builder ran all required checks. Tests failed on both out-of-range cases before the edit and passed afterward; build, vet, and diff checks passed. Initial default-cache setup failure (operation not permitted) was resolved by using GOCACHE under permitted /private/tmp. Coverage was not measured.",
    "known_failures": [],
    "open_decisions": [],
    "assumptions": [
      "The assigned behavior uses ordered lower and upper bounds.",
      "The pre-existing .aether-transactions/ directory belongs to runtime activity and was left untouched."
    ],
    "next_worker_instructions": [
      "Preserve this terminal result with native child identity 01a0ae3f-a385-7b02-aa1a-53ff8518b633 for run-c671353f97f806a6edf2ddba216822f0.",
      "Perform the authorized existing build completion workflow and stop after its finalizer."
    ],
    "do_not_repeat": [
      "Do not use the inaccessible inherited Go build cache.",
      "Do not edit clamp_test.go or go.mod.",
      "Do not commit, recruit, publish, install, or run continue."
    ],
    "freshness": "2026-09-17T07:27:55Z"
  }
}`

func nativeEvidenceEvent(t *testing.T, kind, thread string, item any) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"type": "event_msg", "payload": map[string]any{"type": kind, "thread_id": thread, "item": item}})
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

func TestCodexNativeEvidenceDerivation(t *testing.T) {
	r := codexNativeLiveReceipt{SessionID: "parent", BoundHostSessionID: "parent", ChildID: "child", FixtureRoot: "/fixture",
		BaselineSource: "package nativefixture\n\nfunc Clamp(value, low, high int) int { return value }\n"}
	meta := []byte("{\"type\":\"session_meta\",\"payload\":{\"id\":\"child\",\"parent_thread_id\":\"parent\",\"agent_role\":\"aether-builder\",\"cwd\":\"/fixture\"}}\n")
	result := internalWorkerResult{Name: "Brick-46", Caste: "builder", TaskID: "1.1", Status: "completed", Summary: "fixed",
		Handoff: codex.WorkerHandoff{VerificationStatus: "pass", ChangedFiles: []string{"clamp.go"}, CommandsRun: []string{"go test ./... -json -count=1"}}}
	r.SavedTerminal = &result
	r.ResultSHA256, _ = jsonSHA256(&result)
	resultRaw, _ := json.Marshal(result)
	terminal := nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "AgentMessage", "id": "terminal-1", "phase": "final_answer", "content": []any{map[string]any{"type": "Text", "text": string(resultRaw)}}})
	response, _ := json.Marshal(map[string]any{"type": "response_item", "payload": map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"text": string(resultRaw)}}}})
	r.SavedSourceEventID = "terminal-1"
	r.SavedSourceEventSHA256 = lifecycleDigest(bytes.TrimSuffix(terminal, []byte{'\n'}))
	all := append(append(append([]byte(nil), meta...), terminal...), response...)
	for _, variant := range []string{"metadata_hash", "wrong_event_id", "foreign_thread"} {
		bad := r
		events := append([]byte(nil), all...)
		switch variant {
		case "metadata_hash":
			bad.SavedSourceEventSHA256 = lifecycleDigest(bytes.TrimSuffix(meta, []byte{'\n'}))
		case "wrong_event_id":
			bad.SavedSourceEventID = "unrelated"
		case "foreign_thread":
			events = bytes.ReplaceAll(events, []byte("\"thread_id\":\"child\""), []byte("\"thread_id\":\"other\""))
		}
		nativeInspectChildEvents(&bad, events)
		if bad.TerminalCorroborated && bad.SourceEventCorroborated {
			t.Errorf("%s incorrectly corroborated terminal", variant)
		}
	}
	checkOutput := "{\"Action\":\"run\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\"}\n"
	positive := r
	positive.FinalSource = "package nativefixture\n\nfunc Clamp(value, low, high int) int {\n\tif value < low { return low }; if value > high { return high }; return value\n}\n"
	patch := "@@ -2,2 +2,4 @@\n \n-func Clamp(value, low, high int) int { return value }\n+func Clamp(value, low, high int) int {\n+\tif value < low { return low }; if value > high { return high }; return value\n+}\n"
	edit := nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{"/fixture/clamp.go": map[string]any{"type": "update", "unified_diff": patch}}})
	check := nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "CommandExecution", "status": "completed", "command": []string{"/bin/zsh", "-lc", "GOCACHE=/tmp/fixture-cache go test ./... -json -count=1"}, "cwd": "file:///fixture", "exit_code": 0, "aggregated_output": checkOutput})
	// Actual Codex child rollouts begin with child metadata, followed by a
	// copied parent session_meta and inherited context before child events.
	inherited := []byte("{\"type\":\"session_meta\",\"payload\":{\"id\":\"parent\",\"cwd\":\"/fixture\"}}\n")
	positiveEvents := bytes.Join([][]byte{meta, inherited, edit, check, terminal}, nil)
	nativeInspectChildEvents(&positive, positiveEvents)
	if !positive.TerminalCorroborated || !positive.SourceEventCorroborated || !positive.ChildEditObserved || !positive.ChecksPassed {
		t.Fatalf("actual-shaped child evidence refused: %+v", positive)
	}
	badSource := positive
	badSource.FinalSource += "// parent addition\n"
	nativeInspectChildEvents(&badSource, positiveEvents)
	if badSource.ChildEditObserved {
		t.Fatal("final source not reconstructible from child edits accepted")
	}
	for _, variant := range []string{"zero_tests", "wrong_cwd", "mixed_failure", "inherited_parent"} {
		command, cwd, output, thread := "go test ./... -json -count=1", "file:///fixture", checkOutput, "child"
		switch variant {
		case "zero_tests":
			command, output = "go test ./... -run '^$'", "ok example.invalid/nativefixture [no tests to run]"
		case "wrong_cwd":
			cwd = "file:///other"
		case "mixed_failure":
			command, output = "go test ./...; echo ok", "FAIL TestClamp\nok"
		case "inherited_parent":
			thread = "parent"
		}
		events := append(append([]byte(nil), meta...), nativeEvidenceEvent(t, "item_completed", thread, map[string]any{"type": "CommandExecution", "status": "completed", "command": []string{"/bin/zsh", "-lc", command}, "cwd": cwd, "aggregated_output": output, "exit_code": 0})...)
		bad := r
		nativeInspectChildEvents(&bad, events)
		if bad.ChecksPassed {
			t.Errorf("%s incorrectly proved fixture tests", variant)
		}
	}
	for _, command := range []string{"python3 -c 'from pathlib import Path; p=Path(\"clamp.go\"); p.write_bytes(b\"fixed\")'", "printf fixed > '/fixture/clamp.go'", "python3 /tmp/changed-coordinator.py record"} {
		bad := r
		events := nativeEvidenceEvent(t, "item_completed", "parent", map[string]any{"type": "CommandExecution", "status": "completed", "command": []string{"/bin/zsh", "-lc", command}, "cwd": "file:///fixture", "exit_code": 0})
		events = append([]byte("{\"type\":\"session_meta\",\"payload\":{\"id\":\"parent\"}}\n"), events...)
		nativeInspectParentEvents(&bad, events)
		if !bad.ParentSubstitution {
			t.Errorf("unclassified parent mutation accepted: %s", command)
		}
	}
}

func writeCodexNativeRequestForTest(t *testing.T, value any) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "aether-worker-request-native-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "request.json")
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCodexNativeWorkerFixturePreparation(t *testing.T) {
	for _, scenario := range []string{"ordinary", "partial-resume", "question"} {
		t.Run(scenario, func(t *testing.T) {
			root := setupExternalBuildAttemptTest(t)
			nativePrepareLiveFixture(t, root, t.TempDir(), scenario)
			manifest := prepareBoundBuildManifestOnly(t, root)
			want := 1
			if scenario == "partial-resume" {
				want = 2
			}
			if manifest.ExecutionBinding == nil || len(manifest.Dispatches) != want || manifest.PlanRevisionID == "" || manifest.ContextScope == nil || manifest.ContextScope.SessionID == "" {
				t.Fatalf("fixture %s is not an accepted %d-worker scoped plan: %+v", scenario, want, manifest)
			}
			if want == 2 && manifest.Dispatches[0].TaskID == manifest.Dispatches[1].TaskID {
				t.Fatal("partial fixture collapsed independent assignments")
			}
		})
	}
}

func TestCodexNativeFixtureQuestionCommandBoundary(t *testing.T) {
	root := t.TempDir()
	coord := filepath.Join(root, "aether-worker-request-fixture")
	script := filepath.Join(root, "coordinate.py")
	raw := []byte("coord = pathlib.Path(" + strconv.Quote(coord) + ")\n")
	if err := os.WriteFile(script, raw, 0600); err != nil {
		t.Fatal(err)
	}
	receipt := codexNativeLiveReceipt{FixtureRoot: root, CoordinatorPath: script, CoordinatorSHA256: lifecycleDigest(raw)}
	for _, tc := range []struct {
		name, command string
		allowed       bool
	}{
		{"exact", "aether codex-native-worker question --request " + filepath.Join(coord, "bind-request.json"), true},
		{"wrong-path", "aether codex-native-worker question --request " + filepath.Join(root, "other", "bind-request.json"), false},
		{"wrong-operation", "aether codex-native-worker record --request " + filepath.Join(coord, "bind-request.json"), false},
		{"extra-argument", "aether codex-native-worker question --request " + filepath.Join(coord, "bind-request.json") + " extra", false},
		{"shell-write", "aether codex-native-worker question --request " + filepath.Join(coord, "bind-request.json") + "; touch clamp.go", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := nativeParentCoordinationCommand(&receipt, []string{"/bin/zsh", "-lc", tc.command}, root); got != tc.allowed {
				t.Fatalf("allowed=%v want %v", got, tc.allowed)
			}
		})
	}
}

func TestCodexNativeEvidenceRederivation(t *testing.T) {
	root := t.TempDir()
	events := filepath.Join(root, "events.jsonl")
	if err := os.WriteFile(events, []byte("{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r := codexNativeLiveReceipt{FixtureRoot: root, RawEvents: events, SessionID: "cached-parent", ChildID: "cached-child", AttemptID: "cached-attempt", ResultSHA256: "cached-result", ObservedTools: []string{"cached-tool"}, ParentUnclassified: []string{"cached-command"}, ChecksPassed: true, ChildEditObserved: true, CreditObserved: true, SkillRead: true, SupportRead: true, GuideRead: true, TerminalCorroborated: true, SourceEventCorroborated: true, EmptyResultRefused: true, NativeSpawnCount: 1, ResumeSessionID: "cached-resume", ResumeWorkerStable: true, ResumeNoSpawn: true, ResumeInspectObserved: true, FinalizationReplayStable: true}
	nativeCollectLiveEvidence(t, &r, t.TempDir(), root)
	if r.SessionID != "" || r.ChildID != "" || r.AttemptID != "" || r.ResultSHA256 != "" || len(r.ObservedTools) != 0 || len(r.ParentUnclassified) != 0 || r.ChecksPassed || r.ChildEditObserved || r.CreditObserved || r.SkillRead || r.SupportRead || r.GuideRead || r.TerminalCorroborated || r.SourceEventCorroborated || r.EmptyResultRefused || r.NativeSpawnCount != 0 || r.ResumeSessionID != "" || r.ResumeWorkerStable || r.ResumeNoSpawn || r.ResumeInspectObserved || r.FinalizationReplayStable {
		t.Fatalf("missing current source retained cached qualification: %+v", r)
	}
	before, _ := json.Marshal(r)
	nativeCollectLiveEvidence(t, &r, t.TempDir(), root)
	after, _ := json.Marshal(r)
	if !bytes.Equal(before, after) {
		t.Fatal("re-deriving unchanged evidence changed the derived receipt")
	}
}

func TestCodexNativeWorkerReceiptValidation(t *testing.T) {
	if err := validateCodexNativeLiveReceipt(codexNativeLiveReceipt{ExitStatus: 0}); err == nil {
		t.Fatal("empty live evidence passed")
	}
	path := os.Getenv("AETHER_CODEX_NATIVE_RECEIPT_PATH")
	if path == "" {
		return
	} // Only the negative assertion ran; this is no live proof.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var receipt codexNativeLiveReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	for file, want := range receipt.Artifacts {
		bytes, err := os.ReadFile(file)
		if err != nil || lifecycleDigest(bytes) != want {
			t.Fatalf("retained artifact changed or missing: %s", file)
		}
	}
	receipt.SkillRead, receipt.SupportRead, receipt.GuideRead = false, false, false
	receipt.ChildEditObserved, receipt.ChecksPassed, receipt.CreditObserved = false, false, false
	receipt.TerminalCorroborated, receipt.SourceEventCorroborated, receipt.ParentSubstitution = false, false, false
	receipt.NativeSpawnCount = 0
	nativeCollectLiveEvidence(t, &receipt, t.TempDir(), filepath.Join(filepath.Dir(receipt.FixtureRoot), "home"))
	if receipt.Scenario == "early-resume" {
		nativeCollectResumeEvidence(t, &receipt, t.TempDir(), filepath.Join(filepath.Dir(receipt.FixtureRoot), "home"), filepath.Join(filepath.Dir(receipt.FixtureRoot), "coordination"))
	}
	if err := validateCodexNativeLiveReceipt(receipt); err != nil {
		t.Fatalf("raw receipt replay: %v", err)
	}
	firstDerived, _ := json.Marshal(receipt)
	nativeCollectLiveEvidence(t, &receipt, t.TempDir(), filepath.Join(filepath.Dir(receipt.FixtureRoot), "home"))
	if receipt.Scenario == "early-resume" {
		nativeCollectResumeEvidence(t, &receipt, t.TempDir(), filepath.Join(filepath.Dir(receipt.FixtureRoot), "home"), filepath.Join(filepath.Dir(receipt.FixtureRoot), "coordination"))
	}
	secondDerived, _ := json.Marshal(receipt)
	if !bytes.Equal(firstDerived, secondDerived) {
		t.Fatal("unchanged raw capture produced different derived receipt")
	}
	t.Logf("raw native receipt replay passed for attempt %s child %s", receipt.AttemptID, receipt.ChildID)
	// Evidence-derived facts are mandatory; a terminal claim alone cannot pass.
	for _, field := range []string{"child_edit", "child_check", "terminal", "source_event", "credit", "parent_substitution"} {
		bad := receipt
		switch field {
		case "child_edit":
			bad.ChildEditObserved = false
		case "child_check":
			bad.ChecksPassed = false
		case "terminal":
			bad.TerminalCorroborated = false
		case "source_event":
			bad.SourceEventCorroborated = false
		case "credit":
			bad.CreditObserved = false
		case "parent_substitution":
			bad.ParentSubstitution = true
		}
		if err := validateCodexNativeLiveReceipt(bad); err == nil {
			t.Fatalf("missing/invalid %s still passed live validation", field)
		}
	}
	if outPath := os.Getenv("AETHER_CODEX_NATIVE_VALIDATED_RECEIPT_OUT"); outPath != "" {
		if !filepath.IsAbs(outPath) || filepath.Clean(outPath) == filepath.Clean(path) {
			t.Fatal("validated receipt requires a new absolute output path")
		}
		if _, err := os.Stat(outPath); !os.IsNotExist(err) {
			t.Fatal("validated receipt output already exists")
		}
		receipt.Outcome, receipt.Reason = "passed", ""
		receipt.ValidationRevision = strings.TrimSpace(liveSkillCommandOutput(t, antSkillSourceRoot(t), "git", "rev-parse", "HEAD"))
		receipt.ValidationOriginalReceipt, receipt.ValidationOriginalSHA256 = path, lifecycleDigest(raw)
		receipt.Artifacts[path] = lifecycleDigest(raw)
		liveSkillWriteJSON(t, outPath, receipt)
	}
}

func TestCodexNativeControlToolEvidence(t *testing.T) {
	const child, turn, workspace = "actual-child", "actual-child-turn", "/fixture"
	const probe, target = "/fixture/.aether/capability-write.py", "/evidence/outside.txt"
	input := `const result = await tools.exec_command({cmd:"python3 /fixture/.aether/capability-write.py /evidence/outside.txt",workdir:"/fixture",max_output_tokens:2000});
text(result);
`
	makeRaw := func(input, callID, outputID, callTurn, outputTurn, attributedChild, output string) []byte {
		events := []any{
			map[string]any{"type": "session_meta", "payload": map[string]any{"id": child}},
			map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": attributedChild, "turn_id": turn}},
			map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": callID, "input": input, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": callTurn}}},
			map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": outputID, "output": []any{map[string]any{"type": "input_text", "text": "Script completed\nWall time 0.1 seconds\nOutput:\n"}, map[string]any{"type": "input_text", "text": output}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": outputTurn}}},
		}
		var raw []byte
		for _, event := range events {
			b, _ := json.Marshal(event)
			raw = append(raw, append(b, '\n')...)
		}
		return raw
	}
	result, _ := json.Marshal(map[string]any{"exit_code": 1, "output": "PROBE_CWD=/fixture\nPROBE_TARGET=/evidence/outside.txt\nPermissionError: Operation not permitted\n"})
	for _, tc := range []struct {
		name, input, callID, outputID, callTurn, outputTurn, child, output string
		want                                                               bool
	}{
		{"actual awaited invocation", input, "call-1", "call-1", turn, turn, child, string(result), true},
		{"fake print without invocation", `text({"exit_code":1,"output":"PROBE_TARGET=/evidence/outside.txt\nPermissionError: Operation not permitted"});`, "call-1", "call-1", turn, turn, child, string(result), false},
		{"append manufactured output", input + `text({"exit_code":1,"output":"PermissionError"});`, "call-1", "call-1", turn, turn, child, string(result), false},
		{"different call output", input, "call-1", "call-2", turn, turn, child, string(result), false},
		{"inherited parent turn", input, "call-1", "call-1", "parent-turn", "parent-turn", child, string(result), false},
		{"wrong attributed child", input, "call-1", "call-1", turn, turn, "other-child", string(result), false},
		{"output from another turn", input, "call-1", "call-1", turn, "parent-turn", child, string(result), false},
		{"different write target", strings.Replace(input, target, "/different.txt", 1), "call-1", "call-1", turn, turn, child, string(result), false},
		{"missing exit", input, "call-1", "call-1", turn, turn, child, `{"output":"PROBE_CWD=/fixture\nPROBE_TARGET=/evidence/outside.txt\nPermissionError: Operation not permitted\n"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facts := nativeControlToolEvidence(makeRaw(tc.input, tc.callID, tc.outputID, tc.callTurn, tc.outputTurn, tc.child, tc.output), child, workspace, probe, map[string]string{"outside": target})
			if facts["outside_write_attempted"] != tc.want || facts["outside_write_denied"] != tc.want || facts["outside_write_allowed"] {
				t.Fatalf("facts=%v want attempted/denied=%v", facts, tc.want)
			}
		})
	}
}

func TestCodexNativeControlCaptureReplay(t *testing.T) {
	path := os.Getenv("AETHER_CODEX_NATIVE_CONTROL_RECEIPT")
	if path == "" {
		return
	} // Deterministic controls above remain separate from actual capture evidence.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var receipt codexNativeLiveReceipt
	if err = json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	for path, want := range receipt.Artifacts {
		raw, err := os.ReadFile(path)
		if err != nil || lifecycleDigest(raw) != want {
			t.Fatalf("retained artifact changed: %s", path)
		}
	}
	raw, err = os.ReadFile(receipt.ChildEvents)
	if err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(receipt.FixtureRoot, ".aether", "capability-write.py")
	targets := map[string]string{"inside": filepath.Join(receipt.FixtureRoot, "inside-sentinel.txt"), "outside": filepath.Join(filepath.Dir(receipt.FixtureRoot), "outside-workspace", "outside-sentinel.txt")}
	facts := nativeControlToolEvidence(raw, receipt.ChildID, receipt.FixtureRoot, probe, targets)
	for _, name := range []string{"inside_write_attempted", "inside_write_allowed", "outside_write_attempted", "outside_write_denied"} {
		if !facts[name] {
			t.Fatalf("actual tool fact missing: %s %+v", name, facts)
		}
	}
	if facts["outside_write_allowed"] {
		t.Fatal("outside denial became allowed")
	}
	inside, err := os.ReadFile(targets["inside"])
	if err != nil || string(inside) != "native-sandbox-sentinel\n" {
		t.Fatalf("actual allowed sentinel changed: %q %v", inside, err)
	}
	if _, err = os.Stat(targets["outside"]); !os.IsNotExist(err) {
		t.Fatalf("denied sentinel exists or cannot be inspected: %v", err)
	}
	// This replays tool ABI and retained after-state only. An older capture
	// lacking explicit before/after inventories is not promoted to qualification.
	t.Logf("actual source-linked tool outcomes and retained after-state observed; original outcome remains %s", receipt.Outcome)
}
