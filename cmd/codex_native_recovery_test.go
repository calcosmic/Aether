package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func nativeRecoveryFixture(t *testing.T, count int) (codexBuildManifest, []codexNativeWorkerRequest) {
	t.Helper()
	root := setupExternalBuildAttemptTest(t)
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < count; i++ {
		id := fmt.Sprintf("1.%d", i+1)
		state.Plan.Phases[0].Tasks = append(state.Plan.Phases[0].Tasks, colony.Task{ID: &id, Goal: "Write independent evidence " + id, Status: colony.TaskPending})
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	manifest := prepareBoundBuildManifestOnly(t, root)
	if len(manifest.Dispatches) != count {
		t.Fatalf("want %d runtime assignments: %+v", count, manifest.Dispatches)
	}
	root, _ = filepath.EvalSymlinks(root)
	requests := make([]codexNativeWorkerRequest, count)
	for i, d := range manifest.Dispatches {
		requests[i] = codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *manifest.ExecutionBinding, WorkerName: d.Name, TaskID: normalizedDispatchTaskID(d), HostSessionID: "recovery-host", Workspace: root, HostPermission: "workspace_write"}
	}
	return manifest, requests
}

func nativeRecoveryFinish(t *testing.T, manifest codexBuildManifest, request codexNativeWorkerRequest, index int) codexNativeWorkerRequest {
	t.Helper()
	request = nativeReserveForTest(t, request)
	request.ChildID = fmt.Sprintf("recovery-child-%d", index)
	if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err != nil {
		t.Fatal(err)
	}
	request = nativeTerminalRequestForTest(t, request, manifest.Dispatches[index].Caste, "completed")
	if _, err := runCodexNativeWorker("record", nativeRequestPath(t, request)); err != nil {
		t.Fatal(err)
	}
	return request
}

func nativeRecoveryDashboard(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	reopened, err := storage.NewStore(store.BasePath())
	if err != nil {
		t.Fatal(err)
	}
	store = reopened
	dashboard := buildResumeDashboardResult()
	raw, err := json.Marshal(dashboard["native_recovery"])
	if err != nil {
		t.Fatal(err)
	}
	var recovery map[string]any
	if err := json.Unmarshal(raw, &recovery); err != nil || recovery == nil {
		t.Fatalf("missing native recovery facts in fresh dashboard: %s (%v)", raw, err)
	}
	return dashboard, recovery
}

func nativeRecoveryItems(t *testing.T, recovery map[string]any, key string, count int) []any {
	t.Helper()
	items, ok := recovery[key].([]any)
	if !ok || len(items) != count {
		t.Fatalf("%s = %+v, want %d identities", key, recovery[key], count)
	}
	return items
}

func TestCodexNativeRecoveryPartialReady(t *testing.T) {
	manifest, requests := nativeRecoveryFixture(t, 2)
	nativeRecoveryFinish(t, manifest, requests[0], 0)
	before := nativeJournalBytes(t)
	stateBefore, _ := store.ReadFile("COLONY_STATE.json")
	dashboard, recovery := nativeRecoveryDashboard(t)
	finished := nativeRecoveryItems(t, recovery, "finished", 1)[0].(map[string]any)
	unfinished := nativeRecoveryItems(t, recovery, "unfinished", 1)[0].(map[string]any)
	nativeRecoveryItems(t, recovery, "unresolved", 0)
	if finished["worker_name"] != requests[0].WorkerName || finished["result_sha256"] == "" || unfinished["task_id"] != requests[1].TaskID {
		t.Fatalf("lost assignment identities: %+v", recovery)
	}
	if dashboard["resume_override_command"] != "aether codex-native-worker inspect --phase 1" {
		t.Fatalf("unsafe native recovery action: %+v", dashboard["resume_override_command"])
	}
	if err := resumeColonyCmd.RunE(resumeColonyCmd, nil); err != nil {
		t.Fatal(err)
	}
	stateAfter, _ := store.ReadFile("COLONY_STATE.json")
	if !bytes.Equal(before, nativeJournalBytes(t)) || !bytes.Equal(stateBefore, stateAfter) {
		t.Fatal("public partial resume changed the accepted attempt or granted lifecycle credit")
	}
	if replay, err := runCodexNativeWorker("reserve", nativeRequestPath(t, requests[0])); err != nil || replay.LaunchAllowed {
		t.Fatalf("finished helper relaunched: %+v, %v", replay, err)
	}
	if next, err := runCodexNativeWorker("reserve", nativeRequestPath(t, requests[1])); err != nil || !next.LaunchAllowed {
		t.Fatalf("never-started assignment cannot reserve: %+v, %v", next, err)
	}
}

func TestCodexNativeRecoveryAllTerminal(t *testing.T) {
	manifest, requests := nativeRecoveryFixture(t, 1)
	nativeRecoveryFinish(t, manifest, requests[0], 0)
	before := nativeJournalBytes(t)
	dashboard, recovery := nativeRecoveryDashboard(t)
	nativeRecoveryItems(t, recovery, "finished", 1)
	if dashboard["resume_override_command"] != "aether codex-native-worker stage --phase 1" {
		t.Fatalf("saved results need explicit staging: %+v", dashboard)
	}
	if !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatal("dashboard staged results implicitly")
	}
	_, attempt, _ := loadLatestBuildAttempt(1)
	if attempt.CompletionPath != "" {
		t.Fatal("inspect minted aggregate")
	}
}

func TestCodexNativeRecoveryAggregate(t *testing.T) {
	manifest, requests := nativeRecoveryFixture(t, 1)
	nativeRecoveryFinish(t, manifest, requests[0], 0)
	staged, err := runCodexNativeWorker("stage", nativeRequestPath(t, requests[0]))
	if err != nil {
		t.Fatal(err)
	}
	dashboard, recovery := nativeRecoveryDashboard(t)
	nativeRecoveryItems(t, recovery, "finished", 1)
	if dashboard["resume_override_command"] != buildFinalizeRecoveryCommand(1, staged.CompletionPath) {
		t.Fatalf("lost exact durable aggregate action: %+v", dashboard)
	}
}

func TestCodexNativeRecoveryUnresolved(t *testing.T) {
	for _, bound := range []bool{false, true} {
		t.Run(fmt.Sprint(bound), func(t *testing.T) {
			_, requests := nativeRecoveryFixture(t, 1)
			request := nativeReserveForTest(t, requests[0])
			if bound {
				if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err != nil {
					t.Fatal(err)
				}
			}
			before := nativeJournalBytes(t)
			dashboard, recovery := nativeRecoveryDashboard(t)
			unknown := nativeRecoveryItems(t, recovery, "unresolved", 1)[0].(map[string]any)
			if unknown["launch_id"] != request.LaunchID || strings.Contains(fmt.Sprint(dashboard["resume_override_command"]), "--force") {
				t.Fatalf("lost unresolved identity or inferred redispatch: %+v", recovery)
			}
			if _, err := pauseColonyAt(time.Now()); err == nil {
				t.Fatal("unresolved native launch accepted pause")
			}
			if err := resumeColonyCmd.RunE(resumeColonyCmd, nil); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, nativeJournalBytes(t)) {
				t.Fatal("resume rewrote unresolved evidence")
			}
		})
	}
}

func nativeRecoveryStoreSnapshot(t *testing.T) map[string]string {
	t.Helper()
	files := map[string]string{}
	if err := filepath.WalkDir(store.BasePath(), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = string(raw)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return files
}

func TestCodexNativeRecoveryReadOnly(t *testing.T) {
	_, requests := nativeRecoveryFixture(t, 1)
	nativeReserveForTest(t, requests[0])
	before := nativeRecoveryStoreSnapshot(t)
	for i := 0; i < 2; i++ {
		nativeRecoveryDashboard(t)
		if _, err := runCodexNativeWorker("inspect", nativeRequestPath(t, requests[0])); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(before, nativeRecoveryStoreSnapshot(t)) {
		t.Fatal("native recovery inspection mutated durable bytes")
	}
}

func TestCodexNativeRecoveryResumeTransaction(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(fmt.Sprint(corrupt), func(t *testing.T) {
			manifest, requests := nativeRecoveryFixture(t, 1)
			nativeRecoveryFinish(t, manifest, requests[0], 0)
			now := time.Now().UTC()
			if _, err := pauseColonyAt(now); err != nil {
				t.Fatal(err)
			}
			pauseResumeLifecycleFault = func(point string) error {
				if point == "after_target_commit:target-0001" {
					return errors.New("native resume prefix interrupted")
				}
				return nil
			}
			t.Cleanup(func() { pauseResumeLifecycleFault = nil })
			if _, err := resumeColonyAt(now.Add(time.Second)); err == nil {
				t.Fatal("missing interrupted resume control")
			}
			pauseResumeLifecycleFault = nil
			var state colony.ColonyState
			if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil || state.Paused {
				t.Fatalf("resume prefix did not unpause state: %v", err)
			}
			if corrupt {
				if err := store.AtomicWrite(pauseHandoffDataPath, []byte("{}")); err != nil {
					t.Fatal(err)
				}
			}
			before := nativeJournalBytes(t)
			outcome, err := resumeColonyAt(now.Add(2 * time.Second))
			if err != nil {
				t.Fatal(err)
			}
			if outcome.NativeRecovery != nil {
				t.Fatal("native projection bypassed authorized resume transaction replay")
			}
			if corrupt {
				if outcome.Provenance != colony.RecoveryProvenanceConflicting {
					t.Fatalf("corrupt handoff was not refused: %+v", outcome)
				}
			} else {
				if !outcome.Replay || outcome.Receipt.ReceiptID == "" {
					t.Fatalf("resume did not finish retained transaction: %+v", outcome)
				}
				var session colony.SessionFile
				if err := store.LoadJSON("session.json", &session); err != nil || session.ResumedAt == nil {
					t.Fatalf("resume left session target incomplete: %+v, %v", session, err)
				}
			}
			if !bytes.Equal(before, nativeJournalBytes(t)) {
				t.Fatal("resume transaction changed saved native result")
			}
		})
	}
}

func TestCodexNativeRecoveryCurrency(t *testing.T) {
	for _, field := range []string{"goal", "plan", "run"} {
		t.Run(field, func(t *testing.T) {
			manifest, requests := nativeRecoveryFixture(t, 1)
			nativeRecoveryFinish(t, manifest, requests[0], 0)
			var state colony.ColonyState
			if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
				t.Fatal(err)
			}
			switch field {
			case "goal":
				replacement := "Different same-phase colony"
				state.Goal = &replacement
			case "plan":
				state.Plan.Phases[0].Tasks[0].Goal = "Changed accepted assignment"
			case "run":
				replacement := "different-unproven-run"
				state.RunID = &replacement
			}
			if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
				t.Fatal(err)
			}
			before := nativeRecoveryStoreSnapshot(t)
			dashboard, recovery := nativeRecoveryDashboard(t)
			if recovery["valid"] != false || dashboard["resume_override_command"] != "aether status" {
				t.Fatalf("stale %s advertised actionable native recovery: %+v", field, recovery)
			}
			if !reflect.DeepEqual(before, nativeRecoveryStoreSnapshot(t)) {
				t.Fatal("currency conflict inspection wrote")
			}
		})
	}
}
