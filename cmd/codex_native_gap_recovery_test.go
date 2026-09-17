package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// A controller cut is evidence, not a simulated successful process exit.
// Snapshot paths refer to immutable pre-kill bytes, not the growing rollout.
type nativeGapRecoveryCheckpoint struct {
	Scenario      string            `json:"scenario"`
	ParentSession string            `json:"parent_session"`
	ParentPID     int               `json:"parent_pid"`
	ObservedAt    string            `json:"observed_at"`
	ChildID       string            `json:"child_id"`
	SpawnCallID   string            `json:"spawn_call_id"`
	AttemptID     string            `json:"attempt_id"`
	LaunchID      string            `json:"launch_id"`
	Inventory     map[string]string `json:"inventory"`
	KillSucceeded bool              `json:"kill_succeeded"`
	Waited        bool              `json:"waited"`
}

func nativeGapRecoveryScenario(s string) bool {
	return s == "early-resume" || s == "partial-resume" || s == "spawn-gap"
}

func nativeGapParentExitAccepted(r codexNativeLiveReceipt) bool {
	if r.RecoveryCheckpoint != nil {
		return nativeGapCheckpointRetained(r)
	}
	return r.ExitStatus == 0 // Historical captures retain their original exit semantics.
}

func nativeGapCheckpointRetained(r codexNativeLiveReceipt) bool {
	c := r.RecoveryCheckpoint
	if c == nil || !c.KillSucceeded || !c.Waited || c.ParentPID <= 0 || c.ParentSession == "" || c.ParentSession != r.SessionID || c.AttemptID != r.AttemptID || c.LaunchID != r.LaunchID || c.ChildID == "" || c.SpawnCallID == "" || c.Scenario != r.Scenario || len(c.Inventory) < 4 {
		return false
	}
	for path, want := range c.Inventory {
		raw, err := r.readEvidence(path)
		if err != nil || lifecycleDigest(raw) != want {
			return false
		}
	}
	// Re-derive the spawn identity from the frozen boundary captures. A boolean
	// in the controller receipt cannot substitute for actual event linkage.
	var parent, child []byte
	for path := range c.Inventory {
		switch filepath.Base(path) {
		case "parent-events.jsonl":
			parent, _ = r.readEvidence(path)
		case "child-events.jsonl":
			child, _ = r.readEvidence(path)
		}
	}
	var events []json.RawMessage
	for _, line := range bytes.Split(bytes.TrimSpace(parent), []byte{'\n'}) {
		if !json.Valid(line) {
			return false
		}
		events = append(events, json.RawMessage(line))
	}
	meta := bytes.SplitN(child, []byte{'\n'}, 2)[0]
	if !json.Valid(meta) {
		return false
	}
	input, err := json.Marshal(map[string]any{"events": events, "metadata": []json.RawMessage{meta}, "target": c.ChildID, "parent": c.ParentSession, "cwd": r.FixtureRoot})
	if err != nil {
		return false
	}
	start := strings.Index(nativeFixtureCoordinator, "def resolve_native_child(")
	end := strings.Index(nativeFixtureCoordinator, "def bound_native_child(")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "python3", "-c", "import json,sys\n"+nativeFixtureCoordinator[start:end]+"\nprint(json.dumps(resolve_native_child(**json.load(sys.stdin))))")
	command.Stdin = bytes.NewReader(input)
	output, err := command.Output()
	var identity struct {
		Child string `json:"child_id"`
		Call  string `json:"call_id"`
	}
	return err == nil && json.Unmarshal(output, &identity) == nil && identity.Child == c.ChildID && identity.Call == c.SpawnCallID
}

// Cut only after a completed, thread-attributed coordination command. A file
// appearing or a parent saying "recorded" is insufficient.
func nativeGapCoordinationObserved(r codexNativeLiveReceipt, raw []byte, op string) bool {
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var e nativeHostEvent
		if json.Unmarshal(line, &e) != nil || e.Type != "event_msg" || e.Payload.Type != "item_completed" || e.Payload.ThreadID != r.SessionID {
			continue
		}
		i := e.Payload.Item
		if i.Type != "CommandExecution" || i.Status != "completed" || i.ExitCode == nil || *i.ExitCode != 0 || len(i.Command) != 3 || !nativeSameCwd(i.Cwd, r.FixtureRoot) {
			continue
		}
		words, ok := nativeSimpleShellWords(i.Command[2])
		if !ok || len(words) < 3 || words[0] != "python3" || !nativeCoordinatorPathMatches(&r, i.Cwd, words[1]) || words[2] != op {
			continue
		}
		if len(words) != 3 && !(len(words) == 4 && words[3] == "0") {
			continue
		}
		return true
	}
	return false
}

// Only the same reserved host's uniquely corroborated child may establish the
// spawn gap. This observes metadata; it neither binds nor releases that child.
func nativeGapUnboundChild(r codexNativeLiveReceipt, home string) codexNativeLiveReceipt {
	var found []codexNativeLiveReceipt
	paths, _ := filepath.Glob(filepath.Join(home, ".codex", "sessions", "*", "*", "*", "*.jsonl"))
	for _, path := range paths {
		raw, err := r.readEvidence(path)
		if err != nil {
			continue
		}
		var meta nativeHostEvent
		if json.Unmarshal(bytes.SplitN(raw, []byte{'\n'}, 2)[0], &meta) != nil || meta.Type != "session_meta" || meta.Payload.ParentThreadID != r.BoundHostSessionID || meta.Payload.Cwd != r.FixtureRoot {
			continue
		}
		child := r
		child.ChildID, child.ChildEvents = meta.Payload.ID, path
		nativeCollectChildIdentity(&child, home)
		if child.ChildIdentityCorroborated {
			found = append(found, child)
		}
	}
	if len(found) == 1 {
		return found[0]
	}
	return codexNativeLiveReceipt{}
}

func nativeGapCheckpointReady(r codexNativeLiveReceipt, home string) (codexNativeLiveReceipt, []byte, bool) {
	raw, err := r.readEvidence(r.AttemptPath)
	var attempt buildAttemptRecord
	if err != nil || json.Unmarshal(raw, &attempt) != nil || attempt.ID != r.AttemptID || attempt.RunID != r.RunID || len(attempt.WorkerRuns) != 1 || attempt.CompletionPath != "" || r.CreditObserved || r.NativeSpawnCount != 1 || r.SessionID == "" || r.BoundHostSessionID != r.SessionID {
		return r, nil, false
	}
	paths, _ := filepath.Glob(filepath.Join(home, ".codex", "sessions", "*", "*", "*", "*"+r.SessionID+".jsonl"))
	if len(paths) != 1 {
		return r, nil, false
	}
	parent, err := r.readEvidence(paths[0])
	if err != nil {
		return r, nil, false
	}
	worker := attempt.WorkerRuns[0]
	if worker.Native == nil || worker.ProviderRunID != r.LaunchID || worker.Native.HostSessionID != r.SessionID {
		return r, nil, false
	}
	if r.Scenario == "spawn-gap" {
		child := nativeGapUnboundChild(r, home)
		return child, parent, worker.Native.ChildID == "" && worker.Result == nil && child.ChildIdentityCorroborated && nativeGapCoordinationObserved(r, parent, "reserve")
	}
	if !nativeGapCoordinationObserved(r, parent, "record") || !r.TerminalCorroborated || !r.SourceEventCorroborated || !r.ChildIdentityCorroborated || worker.Native.LaunchState != "terminal" || worker.Result == nil || worker.ResultSHA256 != r.ResultSHA256 {
		return r, nil, false
	}
	if r.Scenario == "partial-resume" {
		if attempt.PlanManifest == nil || len(attempt.PlanManifest.Dispatches) != 2 || !nativeGapCoordinationObserved(r, parent, "stale-result") || !nativeGapCoordinationObserved(r, parent, "child-mismatch") {
			return r, nil, false
		}
	}
	return r, parent, true
}

func nativeGapRunInterruptedParent(t *testing.T, host *exec.Cmd, r *codexNativeLiveReceipt, root, home string) error {
	t.Helper()
	if err := host.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- host.Wait() }()
	waited := false
	defer func() {
		if !waited {
			_ = host.Process.Kill()
			<-done
		}
	}()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			waited = true
			return err // Natural exit/timeout never claims a controller interruption.
		case <-ticker.C:
			nativeCollectLiveEvidence(t, r, root, home)
			child, parent, ready := nativeGapCheckpointReady(*r, home)
			if !ready {
				continue
			}
			c := &nativeGapRecoveryCheckpoint{Scenario: r.Scenario, ParentSession: r.SessionID, ParentPID: host.Process.Pid, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), ChildID: child.ChildID, SpawnCallID: child.ChildSpawnCallID, AttemptID: r.AttemptID, LaunchID: r.LaunchID, Inventory: map[string]string{}}
			if err := os.MkdirAll(filepath.Join(root, "checkpoint"), 0700); err != nil {
				t.Fatal(err)
			}
			snapshot := func(name string, raw []byte) {
				path := filepath.Join(root, "checkpoint", name)
				liveSkillWrite(t, path, raw)
				c.Inventory[path] = lifecycleDigest(raw)
			}
			snapshot("parent-events.jsonl", parent)
			for name, path := range map[string]string{"attempt.json": r.AttemptPath, "child-events.jsonl": child.ChildEvents, "clamp.go": filepath.Join(r.FixtureRoot, "clamp.go"), "state.json": filepath.Join(r.FixtureRoot, ".aether", "data", "COLONY_STATE.json")} {
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				snapshot(name, raw)
			}
			snapshot("inventory.json", mustNativeGapJSON(t, nativeGapRecoveryInventory(r.FixtureRoot)))
			c.KillSucceeded = host.Process.Kill() == nil
			err := <-done
			waited, c.Waited = true, true
			r.RecoveryCheckpoint = c
			liveSkillWriteJSON(t, filepath.Join(root, "checkpoint.json"), c)
			return err
		}
	}
}

func mustNativeGapJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func nativeGapFinishedWorkerStable(r codexNativeLiveReceipt) bool {
	before, err := r.readEvidence(r.BeforeResumeJournalPath)
	if err != nil || lifecycleDigest(before) != r.BeforeResumeJournalSHA256 {
		return false
	}
	after, err := r.readEvidence(r.AttemptPath)
	var a, b buildAttemptRecord
	if err != nil || json.Unmarshal(before, &a) != nil || json.Unmarshal(after, &b) != nil || a.ID != b.ID || a.RunID != b.RunID || len(a.WorkerRuns) != 1 || len(b.WorkerRuns) != 2 {
		return false
	}
	aw, _ := json.Marshal(a.WorkerRuns[0])
	bw, _ := json.Marshal(b.WorkerRuns[0])
	if !bytes.Equal(aw, bw) {
		return false
	}
	if r.RecoveryCheckpoint != nil {
		source, err := r.readEvidence(filepath.Join(filepath.Dir(r.BeforeResumeJournalPath), "checkpoint", "clamp.go"))
		if err != nil || string(source) != r.FinalSource {
			return false
		}
	}
	return true
}

func TestCodexNativeGapRecovery(t *testing.T) {
	for _, order := range [][]int{{0, 1}, {1, 0}} {
		t.Run(fmt.Sprint("equal_wave_credit_", order), func(t *testing.T) {
			root, manifest, requests := nativeFinalizeFixture(t, 2)
			for _, i := range order {
				nativeFinalizeRecord(t, manifest, requests[i], "completed")
			}
			path, packet := nativeFinalizeProjection(t)
			if _, _, err := stageBuildAttemptCompletion(path, packet); err != nil {
				t.Fatal(err)
			}
			_, state, _, dispatches, err := runCodexBuildFinalize(root, 1, packet, true)
			if err != nil || len(dispatches) != 2 {
				t.Fatalf("finalize: %v", err)
			}
			for i, task := range state.Plan.Phases[0].Tasks {
				if task.Status != colony.TaskCompleted || packet.Dispatches[i].TaskID != requests[i].TaskID || packet.Dispatches[i].Name != requests[i].WorkerName {
					t.Fatal("arrival order changed identity or credit")
				}
			}
			journal, stateBytes := nativeJournalBytes(t), nativeFinalizeStateBytes(t)
			if _, _, _, _, err := runCodexBuildFinalize(root, 1, packet, true); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(journal, nativeJournalBytes(t)) || !bytes.Equal(stateBytes, nativeFinalizeStateBytes(t)) {
				t.Fatal("finalizer replay changed journal or credit")
			}
		})
	}

	for _, mode := range []string{"unknown_launch", "stale_parent", "stale_session", "stale_attempt", "unreadable_child_events"} {
		t.Run(mode, func(t *testing.T) {
			manifest, requests := nativeRecoveryFixture(t, 1)
			request := nativeReserveForTest(t, requests[0])
			if mode != "unknown_launch" {
				if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err != nil {
					t.Fatal(err)
				}
			}
			before := nativeRecoveryStoreSnapshot(t)
			candidate := nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "completed")
			switch mode {
			case "unknown_launch":
				candidate.LaunchID = "unknown"
			case "stale_parent":
				candidate.HostSessionID = "different-parent"
			case "stale_session":
				candidate.ExecutionBinding.RunID = "stale-session-run"
			case "stale_attempt":
				candidate.ExecutionBinding.AttemptID = "stale-attempt"
			case "unreadable_child_events":
				candidate.Result = nil
				candidate.SourceEventID = ""
				candidate.SourceEventSHA256 = ""
			}
			// nativeTerminalRequestForTest writes an implementation fixture; snapshot
			// just the authoritative store after fixture preparation, before refusal.
			before = nativeRecoveryStoreSnapshot(t)
			if _, err := runCodexNativeWorker("record", nativeRequestPath(t, candidate)); err == nil {
				t.Fatal("unproved/stale result accepted")
			}
			if err := resumeColonyCmd.RunE(resumeColonyCmd, nil); err != nil {
				t.Fatal(err)
			}
			if replay, err := runCodexNativeWorker("reserve", nativeRequestPath(t, requests[0])); err != nil || replay.LaunchAllowed {
				t.Fatalf("ambiguous launch redispatched: %+v %v", replay, err)
			}
			if !reflect.DeepEqual(before, nativeRecoveryStoreSnapshot(t)) {
				t.Fatal("refusal/resume/replay changed journal or credit")
			}
		})
	}
	t.Run("public_resume_must_be_first_and_exact", func(t *testing.T) {
		for _, scenario := range []string{"early-resume", "partial-resume", "spawn-gap"} {
			r := codexNativeLiveReceipt{Scenario: scenario, SessionID: "old", ResumeSessionID: "new", AttemptID: "attempt", LaunchID: "launch", ChildID: "child", ResultSHA256: "result"}
			recovery := codexNativeRecovery{Valid: true, AttemptID: r.AttemptID}
			if scenario == "spawn-gap" {
				recovery.Unresolved = []codexNativeRecoveryWorker{{LaunchID: r.LaunchID}}
			} else {
				recovery.Finished = []codexNativeRecoveryWorker{{ChildID: r.ChildID, ResultSHA256: r.ResultSHA256}}
			}
			if scenario == "partial-resume" {
				recovery.Unfinished = []codexNativeRecoveryWorker{{TaskID: "remaining"}}
			}
			output := string(mustNativeGapJSON(t, map[string]any{"ok": true, "result": map[string]any{"native_recovery": recovery}}))
			event := func(command string) []byte {
				return append(mustNativeGapJSONCompact(t, map[string]any{"type": "item.completed", "item": map[string]any{"type": "command_execution", "status": "completed", "exit_code": 0, "command": command, "aggregated_output": output}}), '\n')
			}
			raw := event("aether resume")
			if !nativePublicResumeProof(r, raw) {
				t.Fatalf("valid %s rejected", scenario)
			}
			for _, bad := range [][]byte{append(event("aether codex-native-worker inspect --phase 1"), raw...), bytes.ReplaceAll(raw, []byte("attempt"), []byte("stale")), bytes.ReplaceAll(raw, []byte("\\\"valid\\\": true"), []byte("\\\"valid\\\": false"))} {
				if nativePublicResumeProof(r, bad) {
					t.Fatalf("bad %s recovery accepted: %s", scenario, bad)
				}
			}
			r.ResumeSessionID = r.SessionID
			if nativePublicResumeProof(r, raw) {
				t.Fatal("same parent accepted")
			}
		}
	})
	t.Run("checkpoint_requires_retained_bytes_and_controller", func(t *testing.T) {
		root := t.TempDir()
		r := codexNativeLiveReceipt{Scenario: "early-resume", SessionID: "parent", AttemptID: "attempt", LaunchID: "launch"}
		c := &nativeGapRecoveryCheckpoint{Scenario: r.Scenario, ParentSession: r.SessionID, AttemptID: r.AttemptID, LaunchID: r.LaunchID, ParentPID: 1, ChildID: "child", SpawnCallID: "spawn", KillSucceeded: true, Waited: true, Inventory: map[string]string{}}
		r.FixtureRoot = "/fixture"
		turn := map[string]any{"turn_id": "turn"}
		payloads := []any{
			map[string]any{"type": "session_meta", "payload": map[string]any{"id": "parent", "cwd": "/fixture"}},
			map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": "spawn_agent", "call_id": "spawn", "arguments": `{"task_name":"brick","agent_type":"aether-builder"}`, "internal_chat_message_metadata_passthrough": turn}},
			map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": "parent", "turn_id": "turn", "item": map[string]any{"type": "SubAgentActivity", "kind": "started", "id": "spawn", "agent_thread_id": "child", "agent_path": "/root/brick"}}},
			map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "spawn", "output": `{"task_name":"/root/brick"}`, "internal_chat_message_metadata_passthrough": turn}},
		}
		var parent []byte
		for _, e := range payloads {
			parent = append(parent, append(mustNativeGapJSONCompact(t, e), '\n')...)
		}
		child := mustNativeGapJSONCompact(t, map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "cwd": "/fixture", "agent_path": "/root/brick", "agent_role": "aether-builder"}})
		for name, raw := range map[string][]byte{"parent-events.jsonl": parent, "child-events.jsonl": child, "attempt.json": []byte("proof"), "clamp.go": []byte("proof")} {
			path := filepath.Join(root, name)
			liveSkillWrite(t, path, raw)
			c.Inventory[path] = lifecycleDigest(raw)
		}
		r.RecoveryCheckpoint = c
		if !nativeGapCheckpointRetained(r) {
			t.Fatal("valid retained controller receipt rejected")
		}
		c.KillSucceeded = false
		if nativeGapParentExitAccepted(r) {
			t.Fatal("natural exit converted into interruption")
		}
		c.KillSucceeded = true
		liveSkillWrite(t, filepath.Join(root, "clamp.go"), []byte("changed"))
		if nativeGapCheckpointRetained(r) {
			t.Fatal("changed cut-point bytes accepted")
		}
	})
	t.Run("unreadable_capture_does_not_corroborate_child", func(t *testing.T) {
		r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "parent", BoundHostSessionID: "parent", FixtureRoot: "/fixture", ChildID: "child", ChildIdentityCorroborated: true}
		nativeCollectChildIdentity(&r, t.TempDir())
		if r.ChildIdentityCorroborated || r.ChildSpawnCallID != "" {
			t.Fatal("unreadable capture inherited child proof")
		}
	})
	t.Run("unknown_coordination_and_wrong_thread", func(t *testing.T) {
		r := codexNativeLiveReceipt{SessionID: "parent", FixtureRoot: "/fixture", CoordinatorPath: "/fixture/helper.py"}
		raw := nativeEvidenceEvent(t, "item_completed", "parent", map[string]any{"type": "CommandExecution", "status": "completed", "command": []string{"/bin/sh", "-c", "python3 /fixture/helper.py record"}, "cwd": "/fixture", "exit_code": 0})
		if !nativeGapCoordinationObserved(r, raw, "record") {
			t.Fatal("valid actual record event rejected")
		}
		for _, bad := range []string{strings.ReplaceAll(string(raw), "parent", "stale"), strings.ReplaceAll(string(raw), " record", " record; echo forged"), strings.ReplaceAll(string(raw), "item_completed", "item_started")} {
			if nativeGapCoordinationObserved(r, []byte(bad), "record") {
				t.Fatal("unattributed coordination accepted")
			}
		}
	})
}

func mustNativeGapJSONCompact(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func nativeGapRecoveryInventory(root string) map[string]string {
	inventory := nativeFixtureStateInventory(root)
	for _, name := range []string{"double.go", "double_test.go"} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err == nil {
			inventory[name] = lifecycleDigest(raw)
		} else if !os.IsNotExist(err) {
			inventory["error:"+name] = err.Error()
		}
	}
	return inventory
}
