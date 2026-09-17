package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
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
	var ready colony.ColonyState
	if err := json.Unmarshal(stateBefore, &ready); err != nil || ready.State != colony.StateREADY {
		t.Fatalf("partial recovery fixture is not READY: %s, %v", ready.State, err)
	}
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
	for _, finished := range []bool{false, true} {
		for _, field := range []string{"goal", "plan", "run"} {
			t.Run(fmt.Sprintf("%s/terminal=%t", field, finished), func(t *testing.T) {
				t.Setenv("AETHER_ACTIVE_PLATFORM", "codex")
				manifest, requests := nativeRecoveryFixture(t, 1)
				if finished {
					nativeRecoveryFinish(t, manifest, requests[0], 0)
				}
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
}

func TestCodexNativeRecoveryNeverStartedResume(t *testing.T) {
	t.Setenv("AETHER_ACTIVE_PLATFORM", "codex")
	nativeRecoveryFixture(t, 1)
	now := time.Now().UTC()
	if _, err := pauseColonyAt(now); err != nil {
		t.Fatal(err)
	}
	_, paused := nativeRecoveryDashboard(t)
	if paused["valid"] != true || paused["next"] != "aether resume" {
		t.Fatalf("valid never-started pause lost its recovery point: %+v", paused)
	}
	before := nativeJournalBytes(t)
	outcome, err := resumeColonyAt(now.Add(time.Second))
	if err != nil || outcome.Receipt.ReceiptID == "" {
		t.Fatalf("never-started native handoff did not resume: %+v %v", outcome, err)
	}
	_, recovery := nativeRecoveryDashboard(t)
	if recovery["valid"] != true || recovery["next"] != "aether codex-native-worker inspect --phase 1" {
		t.Fatalf("never-started authenticated resume lost saved assignment: %+v", recovery)
	}
	nativeRecoveryItems(t, recovery, "unfinished", 1)
	if !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatal("never-started pause/resume changed accepted attempt")
	}
}

func TestCodexNativeRecoveryActivity(t *testing.T) {
	manifest, requests := nativeRecoveryFixture(t, 1)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("NO_COLOR", "1")
	var out bytes.Buffer
	stdout = &out
	request := nativeReserveForTest(t, requests[0])
	if out.Len() != 0 {
		t.Fatal("reservation emitted a worker start")
	}
	if err := resumeColonyCmd.RunE(resumeColonyCmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "awaiting binding") || !strings.Contains(out.String(), "host session recovery-host") {
		t.Fatalf("reservation is not legible inline: %s", out.String())
	}
	out.Reset()
	bindPath := nativeRequestPath(t, request)
	if _, err := runCodexNativeWorker("bind", bindPath); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "starting wave") != 1 {
		t.Fatalf("bound worker start missing: %s", out.String())
	}
	before := out.String()
	if replay, err := runCodexNativeWorker("bind", bindPath); err != nil || !replay.Replay || out.String() != before {
		t.Fatalf("binding replay emitted another start: %v", err)
	}
	observation := nativeObservationPath(t, request, "running", time.Now().UTC())
	if _, err := runCodexNativeWorker("observe", observation); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "running wave") != 1 {
		t.Fatalf("host running event missing: %s", out.String())
	}
	before = out.String()
	if replay, err := runCodexNativeWorker("observe", observation); err != nil || !replay.Replay || out.String() != before {
		t.Fatalf("observation replay emitted activity: %v", err)
	}
	terminal := nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "completed")
	terminalPath := nativeRequestPath(t, terminal)
	if _, err := runCodexNativeWorker("record", terminalPath); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "  completed") != 1 {
		t.Fatalf("one recorded finish expected: %s", out.String())
	}
	before = out.String()
	journal := nativeJournalBytes(t)
	if replay, err := runCodexNativeWorker("record", terminalPath); err != nil || !replay.Replay || out.String() != before {
		t.Fatalf("terminal replay emitted a second finish: %v", err)
	}
	if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "running", time.Now().UTC().Add(time.Second))); err == nil {
		t.Fatal("late running resurrected a terminal worker")
	}
	if !bytes.Equal(journal, nativeJournalBytes(t)) {
		t.Fatal("replay/late observation changed terminal truth")
	}
	for i := 0; i < 2; i++ {
		nativeRecoveryDashboard(t)
	}
	if out.String() != before {
		t.Fatal("status rendering replayed transition events")
	}
}

func TestCodexNativeRecoveryCancelPending(t *testing.T) {
	_, request := nativeBoundForTest(t)
	now := time.Now().UTC()
	if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "cancel_requested", now)); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "unavailable", now.Add(time.Second))); err != nil {
		t.Fatal(err)
	}
	_, recovery := nativeRecoveryDashboard(t)
	active := nativeRecoveryItems(t, recovery, "active", 1)[0].(map[string]any)
	if active["status"] != "cancellation_pending" || active["last_host_status"] != "unavailable" {
		t.Fatalf("lost pending or unavailable evidence: %+v", active)
	}
	if !strings.Contains(fmt.Sprint(active["host_action"]), request.ChildID) || !strings.Contains(fmt.Sprint(active["host_action"]), "actual interruption/cancellation tool") {
		t.Fatalf("missing actionable same-child host instruction: %+v", active)
	}
	before := nativeRecoveryStoreSnapshot(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	var out bytes.Buffer
	stdout = &out
	if err := pauseColonyCmd.RunE(pauseColonyCmd, nil); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"cancellation pending", "capability unavailable", request.ChildID} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("pending pause omitted %q: %s", text, out.String())
		}
	}
	if !reflect.DeepEqual(before, nativeRecoveryStoreSnapshot(t)) {
		t.Fatal("pending native pause rewrote evidence or recorded a handoff")
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil || state.Paused {
		t.Fatalf("unknown host became paused: %v", err)
	}
}

func TestCodexNativeRecoveryCancelConfirmed(t *testing.T) {
	manifest, request := nativeBoundForTest(t)
	now := time.Now().UTC()
	if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "cancel_requested", now)); err != nil {
		t.Fatal(err)
	}
	terminal := nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "cancelled")
	ack := nativeObservationPath(t, terminal, "cancelled", now.Add(time.Second))
	wrong := terminal
	wrong.ChildID = "wrong-child"
	if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, wrong, "cancelled", now.Add(time.Second))); err == nil {
		t.Fatal("wrong child acknowledged cancellation")
	}
	if _, err := runCodexNativeWorker("observe", ack); err != nil {
		t.Fatal(err)
	}
	_, recovery := nativeRecoveryDashboard(t)
	finished := nativeRecoveryItems(t, recovery, "finished", 1)[0].(map[string]any)
	if finished["status"] != "cancelled" || finished["result_sha256"] == "" {
		t.Fatalf("host cancellation missing durable identity: %+v", finished)
	}
	before := nativeJournalBytes(t)
	if _, err := pauseColonyAt(now.Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	outcome, err := resumeColonyAt(now.Add(3 * time.Second))
	if err != nil || outcome.Receipt.ReceiptID == "" {
		t.Fatalf("validated paused native attempt did not resume: %+v %v", outcome, err)
	}
	if !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatal("pause/resume changed accepted cancellation")
	}
	raw, err := os.ReadFile(filepath.Join(request.Workspace, "evidence.txt"))
	if err != nil || string(raw) != "saved native work\n" {
		t.Fatal("cancel discarded changed files")
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.Plan.Phases[0].Tasks[0].Status == colony.TaskCompleted {
		t.Fatal("cancellation earned task credit")
	}
	_, recovery = nativeRecoveryDashboard(t)
	if recovery["valid"] != true || recovery["next"] != "aether codex-native-worker stage --phase 1" {
		t.Fatalf("authenticated resume lost saved-result recovery: %+v", recovery)
	}
}

func TestCodexNativeRecoveryPauseRace(t *testing.T) {
	manifest, request := nativeBoundForTest(t)
	request = nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "completed")
	path := nativeRequestPath(t, request)
	ready, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	result := make(chan error, 1)
	go func() {
		_, err := runCodexNativeWorkerWithHooks("record", path, codexNativeWorkerHooks{BeforeWrite: func() { close(ready); <-release }})
		result <- err
	}()
	<-ready
	_, err := pauseColonyAt(time.Now().UTC())
	var pending pauseBoundaryPendingError
	if !errors.As(err, &pending) {
		unblock()
		<-result
		t.Fatalf("pause claimed completion before terminal commit: %v", err)
	}
	unblock()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	before := nativeJournalBytes(t)
	if _, err := pauseColonyAt(time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatal("pause after terminal rewrote saved work")
	}
	if replay, err := runCodexNativeWorker("record", path); err != nil || !replay.Replay {
		t.Fatalf("paused terminal replay changed identity: %+v %v", replay, err)
	}
}

func TestCodexNativeRecoveryOrdinaryHostPlan(t *testing.T) {
	for _, terminalCount := range []int{0, 1, 2} {
		t.Run(fmt.Sprintf("terminal=%d", terminalCount), func(t *testing.T) {
			t.Setenv("AETHER_ACTIVE_PLATFORM", "codex")
			manifest, _ := nativeRecoveryFixture(t, 2)
			if manifest.HostPlatform != "codex" {
				t.Fatalf("fixture lost the Codex host identity: %q", manifest.HostPlatform)
			}
			if terminalCount == 0 {
				d := manifest.Dispatches[0]
				request := internalWorkerDispatchRequest{WorkerName: d.Name, TaskID: normalizedDispatchTaskID(d), Caste: d.Caste}
				if _, err := beginBuildAttemptWorkerRun(1, *manifest.ExecutionBinding, request, "ordinary-before-process", codex.PlatformCodex); err != nil {
					t.Fatal(err)
				}
			}
			for _, d := range manifest.Dispatches[:terminalCount] {
				requestPath := writeBoundBuildWorkerRequest(t, manifest, d)
				if _, err := runInternalWorkerAdapter(context.Background(), requestPath, false, true); err != nil {
					t.Fatal(err)
				}
			}
			var state colony.ColonyState
			if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
				t.Fatal(err)
			}
			before := nativeRecoveryStoreSnapshot(t)
			if recovery := buildCodexNativeRecovery(state); recovery != nil {
				t.Fatalf("ordinary adapter records were classified as native: %+v", recovery)
			}
			dashboard := buildResumeDashboardResult()
			if dashboard["native_recovery"] != nil {
				t.Fatalf("ordinary resume acquired a native projection: %+v", dashboard["native_recovery"])
			}
			if terminalCount == len(manifest.Dispatches) {
				_, record, ok := loadLatestBuildAttempt(1)
				if !ok || record.CompletionPath == "" || record.CompletionSHA256 == "" {
					t.Fatalf("ordinary terminal completion is missing: %+v", record)
				}
				recovery := dashboard["recovery"].(map[string]interface{})
				if recovery["next"] != buildFinalizeRecoveryCommand(1, record.CompletionPath) {
					t.Fatalf("ordinary completion lost its exact finalize route: %+v", recovery)
				}
			}
			if !reflect.DeepEqual(before, nativeRecoveryStoreSnapshot(t)) {
				t.Fatal("recovery inspection changed ordinary durable evidence")
			}
		})
	}
}

func TestCodexNativeRecoveryMixedAndDamagedRemainBlocked(t *testing.T) {
	for _, mode := range []string{"ordinary-first", "native-first", "damaged-native", "one-native-field-removed"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("AETHER_ACTIVE_PLATFORM", "codex")
			manifest, requests := nativeRecoveryFixture(t, 2)
			ordinaryIndex := 0
			if mode != "ordinary-first" {
				ordinaryIndex = 1
			}
			for i, request := range requests {
				if i == ordinaryIndex && (mode == "ordinary-first" || mode == "native-first") {
					d := manifest.Dispatches[i]
					ordinary := internalWorkerDispatchRequest{WorkerName: d.Name, TaskID: normalizedDispatchTaskID(d), Caste: d.Caste}
					if _, err := beginBuildAttemptWorkerRun(1, *manifest.ExecutionBinding, ordinary, "ordinary-mixed", codex.PlatformCodex); err != nil {
						t.Fatal(err)
					}
				} else {
					nativeReserveForTest(t, request)
				}
			}
			if mode == "damaged-native" || mode == "one-native-field-removed" {
				path, record, _ := loadLatestBuildAttempt(1)
				if mode == "damaged-native" {
					record.WorkerRuns[0].Native.PromptSHA256 = "tampered"
				} else {
					record.WorkerRuns[0].Native = nil
				}
				if err := store.SaveJSON(path, record); err != nil {
					t.Fatal(err)
				}
			}
			before := nativeRecoveryStoreSnapshot(t)
			dashboard, recovery := nativeRecoveryDashboard(t)
			if recovery["valid"] != false || recovery["error"] == "" || recovery["next"] != "aether status" || dashboard["resume_override_command"] != "aether status" {
				t.Fatalf("mixed or damaged native evidence became actionable: %+v", recovery)
			}
			if !reflect.DeepEqual(before, nativeRecoveryStoreSnapshot(t)) {
				t.Fatal("blocked recovery rewrote saved evidence")
			}
		})
	}
}

func TestCodexNativeRecoveryProcessLane(t *testing.T) {
	process := exec.Command("sleep", "30")
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	stop := func() { once.Do(func() { _ = process.Process.Kill(); _ = process.Wait() }) }
	t.Cleanup(stop)
	fixture := commitTestBuildStart(t, testBuildStartOptions{Variant: buildStartDirect, GeneratedAt: time.Now().UTC(), ProcessID: process.Process.Pid, ExecutionOwner: "go-runtime", DispatchMode: "direct", MakeLatest: testBuildStartBool(true)})
	if recovery := buildCodexNativeRecovery(fixture.State); recovery != nil {
		t.Fatal("ordinary subprocess was classified as native")
	}
	if _, err := pauseColonyAt(time.Now().UTC()); err == nil {
		t.Fatal("live subprocess pause no longer waits")
	}
	stop()
	if _, err := pauseColonyAt(time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	outcome, err := resumeColonyAt(time.Now().UTC())
	if err != nil || outcome.Receipt.ReceiptID == "" {
		t.Fatalf("ordinary subprocess recovery regressed: %+v %v", outcome, err)
	}
}

func TestCodexNativeRecoveryContextActivityTimestamp(t *testing.T) {
	for _, running := range []bool{false, true} {
		t.Run(fmt.Sprint(running), func(t *testing.T) {
			_, request := nativeBoundForTest(t)
			observed := time.Now().UTC()
			if running {
				if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "running", observed)); err != nil {
					t.Fatal(err)
				}
			}
			path, attempt, _ := loadLatestBuildAttempt(1)
			// Retained Plan 03 context-send event shape, pinned at de76511c.
			// This pure projection control exercises Plan 06's timestamp/status
			// fields; Plan 03 separately owns admission and HostStatus filtering.
			attempt.WorkerRuns[0].Native.Observations = append(attempt.WorkerRuns[0].Native.Observations, codexNativeHostObservation{
				SchemaVersion: 1, Status: "context_delivered", ChildID: request.ChildID,
				ObservedAt:    observed.Add(time.Minute).Format(time.RFC3339Nano),
				SourceEventID: "context-send-event", SourceEventSHA256: strings.Repeat("c", 64),
			})
			if err := store.SaveJSON(path, attempt); err != nil {
				t.Fatal(err)
			}
			before := nativeRecoveryStoreSnapshot(t)
			var state colony.ColonyState
			if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
				t.Fatal(err)
			}
			recovery := buildCodexNativeRecovery(state)
			if recovery == nil || !recovery.Valid {
				t.Fatalf("context event invalidated accepted native recovery: %+v", recovery)
			}
			workers := append(recovery.Active, recovery.Unresolved...)
			if len(workers) != 1 {
				t.Fatalf("lost native assignment: %+v", recovery)
			}
			wantTime, wantStatus := "", ""
			if running {
				wantTime, wantStatus = observed.Format(time.RFC3339Nano), "running"
			}
			if workers[0].ObservedAt != wantTime || workers[0].LastHostStatus != wantStatus {
				t.Fatalf("context delivery replaced host activity time/status: %+v; want %q %q", workers[0], wantTime, wantStatus)
			}
			if !reflect.DeepEqual(before, nativeRecoveryStoreSnapshot(t)) {
				t.Fatal("activity projection rewrote retained context evidence")
			}
		})
	}
}
