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
	"regexp"
	"strconv"
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
	KilledAt      string            `json:"killed_at"`
	WaitedAt      string            `json:"waited_at"`
	WaitExitCode  int               `json:"wait_exit_code"`
}

func nativeGapRecoveryScenario(s string) bool {
	return s == "early-resume" || s == "partial-resume" || s == "spawn-gap"
}

func nativeGapParentExitAccepted(r codexNativeLiveReceipt) bool {
	if r.RecoveryCheckpoint != nil {
		return nativeGapCheckpointRetained(r)
	}
	// A successful natural exit is not a retained interruption boundary. Old
	// receipts remain unchanged but cannot waive the current replay contract.
	return !nativeGapRecoveryScenario(r.Scenario) && r.ExitStatus == 0
}

func nativeGapCheckpointRetained(r codexNativeLiveReceipt) bool {
	c := r.RecoveryCheckpoint
	if c == nil || !nativeGapRecoveryScenario(c.Scenario) || !c.KillSucceeded || !c.Waited || c.WaitExitCode != -1 || c.ParentPID <= 0 || c.ParentSession == "" || c.ParentSession != r.SessionID || c.AttemptID != r.AttemptID || c.LaunchID != r.LaunchID || c.ChildID == "" || c.SpawnCallID == "" || c.Scenario != r.Scenario || len(c.Inventory) != 6 {
		return false
	}
	observed, e1 := time.Parse(time.RFC3339Nano, c.ObservedAt)
	killed, e2 := time.Parse(time.RFC3339Nano, c.KilledAt)
	waited, e3 := time.Parse(time.RFC3339Nano, c.WaitedAt)
	if e1 != nil || e2 != nil || e3 != nil || killed.Before(observed) || waited.Before(killed) {
		return false
	}
	snapshots := map[string][]byte{}
	directory := ""
	for path, want := range c.Inventory {
		raw, err := r.readEvidence(path)
		if err != nil || lifecycleDigest(raw) != want || (directory != "" && filepath.Dir(path) != directory) {
			return false
		}
		directory = filepath.Dir(path)
		snapshots[filepath.Base(path)] = raw
	}
	for _, name := range []string{"parent-events.jsonl", "child-events.jsonl", "attempt.json", "state.json", "clamp.go", "inventory.json"} {
		if len(snapshots[name]) == 0 {
			return false
		}
	}
	if !nativeGapCheckpointSemantics(r, snapshots, observed) {
		return false
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

func nativeGapCheckpointSemantics(r codexNativeLiveReceipt, snapshots map[string][]byte, observed time.Time) bool {
	c := r.RecoveryCheckpoint
	var attempt buildAttemptRecord
	var state colony.ColonyState
	var inventory map[string]string
	if json.Unmarshal(snapshots["attempt.json"], &attempt) != nil || json.Unmarshal(snapshots["state.json"], &state) != nil || json.Unmarshal(snapshots["inventory.json"], &inventory) != nil {
		return false
	}
	if attempt.ID != c.AttemptID || attempt.RunID == "" || attempt.RunID != r.RunID || attempt.CompletionPath != "" || attempt.CompletionSHA256 != "" || len(attempt.CreditedFiles) != 0 || attempt.CompletedAt != "" || attempt.OutOfBandVerification != nil || len(attempt.WorkerRuns) != 1 || attempt.PlanManifest == nil {
		return false
	}
	m := attempt.PlanManifest
	b := m.ExecutionBinding
	if b == nil || b.Validate() != nil || b.RunID != attempt.RunID || b.AttemptID != attempt.ID || b.ManifestSHA256 != attempt.ManifestSHA256 || b.WorkspaceFingerprint != attempt.WorkspaceSHA256 || b.ExecutionOwner != attempt.ExecutionOwner || m.AttemptID != attempt.ID || m.Phase != attempt.Phase || !nativeSameCwd(m.Root, r.FixtureRoot) {
		return false
	}
	digest, err := buildManifestSHA256(*m)
	if err != nil || digest != attempt.ManifestSHA256 {
		return false
	}
	// Match the runtime's canonical state decoding before hashing. Legacy
	// plan-only state may validly have current_phase=0 and omitted defaults.
	migrated, err := migratePlanningState(r.FixtureRoot, normalizeLegacyColonyState(state))
	if err != nil {
		return false
	}
	state = migrated.State
	stateDigest, err := jsonSHA256(state)
	if err != nil || stateDigest != attempt.OriginalStateSHA || validateBuildFinalizeStateStillCurrent(state, attempt.Phase) != nil || state.State == colony.StateBUILT {
		return false
	}
	if inventory["clamp.go"] != lifecycleDigest(snapshots["clamp.go"]) || inventory[filepath.Join(r.FixtureRoot, ".aether", "data", "COLONY_STATE.json")] != lifecycleDigest(snapshots["state.json"]) || inventory[r.AttemptPath] != lifecycleDigest(snapshots["attempt.json"]) {
		return false
	}
	for path := range inventory {
		if strings.HasPrefix(path, "error:") {
			return false
		}
	}
	w := attempt.WorkerRuns[0]
	if w.Native == nil || w.ProviderRunID != c.LaunchID || w.Native.HostSessionID != c.ParentSession || !nativeSameCwd(w.Native.Workspace, r.FixtureRoot) {
		return false
	}
	assignments := 1
	if c.Scenario == "partial-resume" {
		assignments = 2
	}
	if len(m.Dispatches) != assignments {
		return false
	}
	matched := 0
	for _, d := range m.Dispatches {
		if d.Name == w.WorkerName && normalizedDispatchTaskID(d) == w.TaskID {
			matched++
			dh, err := jsonSHA256(d)
			if err != nil || dh != w.Native.DispatchSHA256 {
				return false
			}
		}
	}
	if matched != 1 {
		return false
	}
	var spawnAt, coordinationAt, terminalAt time.Time
	spawns := 0
	op := "record"
	if c.Scenario == "spawn-gap" {
		op = "reserve"
	}
	// Every saved record must be whole and no later than the controller cut.
	for _, name := range []string{"parent-events.jsonl", "child-events.jsonl"} {
		var last time.Time
		for _, line := range bytes.Split(snapshots[name], []byte{'\n'}) {
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			var event struct{ Timestamp string }
			if json.Unmarshal(line, &event) != nil {
				return false
			}
			at, err := time.Parse(time.RFC3339Nano, event.Timestamp)
			if err != nil || at.After(observed) || (!last.IsZero() && at.Before(last)) {
				return false
			}
			last = at
			var host nativeHostEvent
			if json.Unmarshal(line, &host) != nil {
				return false
			}
			if name == "parent-events.jsonl" {
				if host.Type == "event_msg" && host.Payload.Type == "item_completed" && host.Payload.ThreadID == c.ParentSession && host.Payload.Item.Type == "SubAgentActivity" && host.Payload.Item.Kind == "started" {
					spawns++
					if host.Payload.Item.ID == c.SpawnCallID {
						spawnAt = at
					}
				}
				if nativeGapCoordinationObserved(r, line, op) {
					coordinationAt = at
				}
			} else if host.Type == "event_msg" && host.Payload.Type == "item_completed" && host.Payload.ThreadID == c.ChildID && host.Payload.Item.ID == w.Native.SourceEventID && host.Payload.Item.Type == "AgentMessage" {
				terminalAt = at
			}
		}
	}
	if spawns != 1 || spawnAt.IsZero() || coordinationAt.IsZero() {
		return false
	}
	if c.Scenario == "spawn-gap" {
		return !coordinationAt.After(spawnAt) && string(snapshots["clamp.go"]) == r.BaselineSource && w.Native.LaunchState == "reserved" && w.Native.ChildID == "" && w.Native.SourceEventID == "" && w.Native.SourceEventSHA256 == "" && w.Result == nil && w.ResultSHA256 == "" && w.CompletedAt == "" && nativeGapCoordinationObserved(r, snapshots["parent-events.jsonl"], "reserve")
	}
	completed, err := time.Parse(time.RFC3339Nano, w.CompletedAt)
	if err != nil || terminalAt.IsZero() || terminalAt.Before(spawnAt) || coordinationAt.Before(terminalAt) || completed.Before(terminalAt) || completed.After(observed) {
		return false
	}
	if w.Native.LaunchState != "terminal" || w.Native.ChildID != c.ChildID || w.Status != "completed" || w.Result == nil || w.Result.Status != "completed" || w.Result.Name != w.WorkerName || w.Result.TaskID != w.TaskID || w.ResultSHA256 == "" {
		return false
	}
	digest, err = jsonSHA256(w.Result)
	if err != nil || digest != w.ResultSHA256 {
		return false
	}
	proof := r
	proof.ChildID, proof.SavedSourceEventID, proof.SavedSourceEventSHA256 = c.ChildID, w.Native.SourceEventID, w.Native.SourceEventSHA256
	proof.ResultSHA256, proof.SavedTerminal = w.ResultSHA256, w.Result
	proof.FinalSource = string(snapshots["clamp.go"])
	nativeInspectChildEvents(&proof, snapshots["child-events.jsonl"])
	return proof.TerminalCorroborated && proof.SourceEventCorroborated && proof.ChildEditObserved && nativeGapCoordinationObserved(r, snapshots["parent-events.jsonl"], "record")
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
			c.ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
			c.KillSucceeded = host.Process.Kill() == nil
			c.KilledAt = time.Now().UTC().Format(time.RFC3339Nano)
			err := <-done
			c.WaitedAt = time.Now().UTC().Format(time.RFC3339Nano)
			if host.ProcessState != nil {
				c.WaitExitCode = host.ProcessState.ExitCode()
			}
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
	t.Run("finalizer_comparison_exact_scope", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "coordinator.py")
		script := []byte("coord = pathlib.Path(" + strconv.Quote(root) + ")\n")
		liveSkillWrite(t, path, script)
		r := codexNativeLiveReceipt{CoordinatorPath: path, CoordinatorSHA256: lifecycleDigest(script), FixtureRoot: root}
		command := "diff -s " + filepath.Join(root, "post-finalize-1-state.json") + " " + filepath.Join(root, "post-finalize-2-state.json")
		if !nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", command}, root) {
			t.Fatal("exact read-only comparison rejected")
		}
		for _, bad := range []string{strings.Replace(command, "diff -s", "diff --output=clamp.go", 1), command + "; touch clamp.go", strings.Replace(command, "-2-state", "-2-attempt", 1), strings.Replace(command, "-1-state", "-3-state", 1), strings.Replace(command, root, "/foreign", 1)} {
			if nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", bad}, root) {
				t.Fatal("unowned or mutating comparison accepted")
			}
		}
		if nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", command}, "/foreign") {
			t.Fatal("foreign cwd accepted")
		}
		liveSkillWrite(t, path, append(script, []byte("changed")...))
		if nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", command}, root) {
			t.Fatal("modified coordinator accepted")
		}
	})

	t.Run("recovery_does_not_erase_qualification_gaps", func(t *testing.T) {
		r := codexNativeLiveReceipt{Scenario: "early-resume", ExitStatus: 0, TerminalCorroborated: true, SourceEventCorroborated: true, ChecksPassed: false, ParentSubstitution: true}
		if nativeGapEarlyResumeReady(r) {
			t.Fatal("terminal without retained cut point allowed recovery")
		}
		if validateCodexNativeLiveReceipt(r) == nil {
			t.Fatal("recovery eligibility erased missing qualification proof")
		}
		r.SourceEventCorroborated = false
		if nativeGapEarlyResumeReady(r) {
			t.Fatal("uncorroborated terminal allowed recovery")
		}
	})

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
	nativeGapCheckpointSemanticControls(t)
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

func nativeGapCheckpointSemanticControls(t *testing.T) {
	for _, scenario := range []string{"early-resume", "partial-resume", "spawn-gap"} {
		t.Run("retained-"+scenario, func(t *testing.T) {
			count := 1
			if scenario == "partial-resume" {
				count = 2
			}
			manifest, requests := nativeRecoveryFixture(t, count)
			request := nativeReserveForTest(t, requests[0])
			fixture := request.Workspace
			const baseline = "package fixture\nconst value = 0\n"
			const source = "package fixture\nconst value = 1\n"
			const childID = "checkpoint-child"
			at := time.Now().UTC()
			event := func(typ string, payload any) []byte {
				return append(mustNativeGapJSONCompact(t, map[string]any{"timestamp": at.Format(time.RFC3339Nano), "type": typ, "payload": payload}), '\n')
			}
			turn := map[string]any{"turn_id": "turn"}
			parent := event("session_meta", map[string]any{"id": request.HostSessionID, "cwd": fixture})
			parent = append(parent, event("response_item", map[string]any{"type": "function_call", "name": "spawn_agent", "call_id": "spawn", "arguments": `{"task_name":"brick","agent_type":"aether-builder"}`, "internal_chat_message_metadata_passthrough": turn})...)
			parent = append(parent, event("event_msg", map[string]any{"type": "item_completed", "thread_id": request.HostSessionID, "turn_id": "turn", "item": map[string]any{"type": "SubAgentActivity", "kind": "started", "id": "spawn", "agent_thread_id": childID, "agent_path": "/root/brick"}})...)
			parent = append(parent, event("response_item", map[string]any{"type": "function_call_output", "call_id": "spawn", "output": `{"task_name":"/root/brick"}`, "internal_chat_message_metadata_passthrough": turn})...)
			child := event("session_meta", map[string]any{"id": childID, "parent_thread_id": request.HostSessionID, "cwd": fixture, "agent_path": "/root/brick", "agent_role": "aether-builder"})
			op := "reserve"
			if scenario != "spawn-gap" {
				op = "record"
				request.ChildID = childID
				if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err != nil {
					t.Fatal(err)
				}
				request = nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "completed")
				child = append(child, event("event_msg", map[string]any{"type": "item_completed", "thread_id": childID, "item": map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{filepath.Join(fixture, "clamp.go"): map[string]any{"type": "update", "unified_diff": "@@ -1,2 +1,2 @@\n package fixture\n-const value = 0\n+const value = 1\n"}}}})...)
				terminal := event("event_msg", map[string]any{"type": "item_completed", "thread_id": childID, "item": map[string]any{"type": "AgentMessage", "id": "terminal", "phase": "final_answer", "content": string(mustNativeGapJSONCompact(t, request.Result))}})
				request.SourceEventID, request.SourceEventSHA256 = "terminal", lifecycleDigest(bytes.TrimSpace(terminal))
				child = append(child, terminal...)
				if _, err := runCodexNativeWorker("record", nativeRequestPath(t, request)); err != nil {
					t.Fatal(err)
				}
			}
			parent = append(parent, event("event_msg", map[string]any{"type": "item_completed", "thread_id": request.HostSessionID, "item": map[string]any{"type": "CommandExecution", "status": "completed", "exit_code": 0, "cwd": fixture, "command": []string{"/bin/sh", "-c", "python3 " + filepath.Join(fixture, "helper.py") + " " + op}}})...)
			attemptPath, attempt, ok := loadLatestBuildAttempt(1)
			if !ok {
				t.Fatal("missing runtime attempt")
			}
			absAttempt := filepath.Join(store.BasePath(), attemptPath)
			state := nativeFinalizeStateBytes(t)
			attemptBytes := mustNativeGapJSON(t, attempt)
			savedSource := source
			if scenario == "spawn-gap" {
				savedSource = baseline
			}
			inventory := map[string]string{"clamp.go": lifecycleDigest([]byte(savedSource)), filepath.Join(fixture, ".aether", "data", "COLONY_STATE.json"): lifecycleDigest(state), absAttempt: lifecycleDigest(attemptBytes)}
			r := codexNativeLiveReceipt{Scenario: scenario, FixtureRoot: fixture, SessionID: request.HostSessionID, BoundHostSessionID: request.HostSessionID, AttemptID: attempt.ID, RunID: attempt.RunID, LaunchID: request.LaunchID, AttemptPath: absAttempt, CoordinatorPath: filepath.Join(fixture, "helper.py"), BaselineSource: baseline}
			c := &nativeGapRecoveryCheckpoint{Scenario: scenario, ParentSession: r.SessionID, ParentPID: 1, ObservedAt: at.Add(time.Second).Format(time.RFC3339Nano), KilledAt: at.Add(2 * time.Second).Format(time.RFC3339Nano), WaitedAt: at.Add(3 * time.Second).Format(time.RFC3339Nano), WaitExitCode: -1, ChildID: childID, SpawnCallID: "spawn", AttemptID: attempt.ID, LaunchID: request.LaunchID, KillSucceeded: true, Waited: true, Inventory: map[string]string{}}
			r.RecoveryCheckpoint = c
			root := t.TempDir()
			original := map[string][]byte{"parent-events.jsonl": parent, "child-events.jsonl": child, "attempt.json": attemptBytes, "state.json": state, "clamp.go": []byte(savedSource), "inventory.json": mustNativeGapJSON(t, inventory)}
			write := func(name string, raw []byte) {
				path := filepath.Join(root, name)
				liveSkillWrite(t, path, raw)
				c.Inventory[path] = lifecycleDigest(raw)
			}
			for name, raw := range original {
				write(name, raw)
			}
			if !nativeGapCheckpointRetained(r) {
				t.Fatal("valid typed retained boundary rejected")
			}
			changeAttempt := func(change func(*buildAttemptRecord)) {
				var a buildAttemptRecord
				if err := json.Unmarshal(attemptBytes, &a); err != nil {
					t.Fatal(err)
				}
				change(&a)
				write("attempt.json", mustNativeGapJSON(t, a))
			}
			mutations := map[string]func(){
				"absent-checkpoint":   func() { r.RecoveryCheckpoint = nil; r.ExitStatus = 0 },
				"wrong-worker":        func() { changeAttempt(func(a *buildAttemptRecord) { a.WorkerRuns[0].WorkerName = "other" }) },
				"wrong-parent":        func() { changeAttempt(func(a *buildAttemptRecord) { a.WorkerRuns[0].Native.HostSessionID = "other" }) },
				"wrong-child-binding": func() { changeAttempt(func(a *buildAttemptRecord) { a.WorkerRuns[0].Native.ChildID = "other" }) },
				"wrong-manifest":      func() { changeAttempt(func(a *buildAttemptRecord) { a.PlanManifest.Dispatches[0].Name = "other" }) },
				"replacement-worker": func() {
					changeAttempt(func(a *buildAttemptRecord) { a.WorkerRuns = append(a.WorkerRuns, a.WorkerRuns[0]) })
				},
				"wrong-source-event": func() { changeAttempt(func(a *buildAttemptRecord) { a.WorkerRuns[0].Native.SourceEventID = "other" }) },
				"arbitrary-attempt":  func() { write("attempt.json", []byte("proof")) },
				"wrong-run":          func() { a := attempt; a.RunID = "wrong"; write("attempt.json", mustNativeGapJSON(t, a)) },
				"wrong-attempt":      func() { a := attempt; a.ID = "wrong"; write("attempt.json", mustNativeGapJSON(t, a)) },
				"aggregate": func() {
					a := attempt
					a.CompletionPath = "already-staged"
					write("attempt.json", mustNativeGapJSON(t, a))
				},
				"credited": func() {
					a := attempt
					a.CreditedFiles = []string{"clamp.go"}
					write("attempt.json", mustNativeGapJSON(t, a))
				},
				"finalized": func() {
					a := attempt
					a.CompletedAt = at.Format(time.RFC3339Nano)
					write("attempt.json", mustNativeGapJSON(t, a))
				},
				"wrong-launch": func() { c.LaunchID = "wrong"; r.LaunchID = "wrong" },
				"wrong-state": func() {
					var s colony.ColonyState
					_ = json.Unmarshal(state, &s)
					s.State = colony.StateBUILT
					write("state.json", mustNativeGapJSON(t, s))
				},
				"wrong-source":    func() { write("clamp.go", []byte("different")) },
				"missing-state":   func() { delete(c.Inventory, filepath.Join(root, "state.json")) },
				"wrong-scenario":  func() { c.Scenario = "ordinary"; r.Scenario = "ordinary" },
				"natural-exit":    func() { c.KillSucceeded = false },
				"successful-wait": func() { c.WaitExitCode = 0 },
				"wrong-order":     func() { c.WaitedAt = at.Format(time.RFC3339Nano) },
				"truncated-child": func() { write("child-events.jsonl", append(append([]byte(nil), child...), []byte("{")...)) },
			}
			for name, mutate := range mutations {
				t.Run(name, func(t *testing.T) {
					beforeC, beforeR := *c, r
					mutate()
					// Also rehash inventory references: reject semantic corruption,
					// not merely stale content digests at the snapshot layer.
					updated := map[string]string{}
					for key, value := range inventory {
						updated[key] = value
					}
					for name, key := range map[string]string{"clamp.go": "clamp.go", "state.json": filepath.Join(fixture, ".aether", "data", "COLONY_STATE.json"), "attempt.json": absAttempt} {
						raw, _ := os.ReadFile(filepath.Join(root, name))
						updated[key] = lifecycleDigest(raw)
					}
					write("inventory.json", mustNativeGapJSON(t, updated))
					if nativeGapParentExitAccepted(r) {
						t.Fatal("semantically invalid hash-valid boundary accepted")
					}
					*c, r = beforeC, beforeR
					for name, raw := range original {
						write(name, raw)
					}
				})
			}
		})
	}
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

// The controller has independently corroborated the cut before this gate.
// Missing check attribution must remain a final qualification failure, without
// suppressing the actual fresh-parent leg for an accepted durable terminal.
func nativeGapEarlyResumeReady(r codexNativeLiveReceipt) bool {
	return nativeGapParentExitAccepted(r) && r.TerminalCorroborated && r.SourceEventCorroborated && r.CompletionPath == "" && !r.CreditObserved
}

// Recognize only comparison of the two immutable outputs from this reviewed
// coordinator. Code-mode callers still require their actual unique command
// events via nativeCorroboratedBatch; this does not authorize script execution.
func nativeGapFinalizerComparison(r codexNativeLiveReceipt, words []string) bool {
	if len(words) != 4 || words[0] != "diff" || words[1] != "-s" {
		return false
	}
	raw, err := r.readEvidence(r.CoordinatorPath)
	if err != nil || lifecycleDigest(raw) != r.CoordinatorSHA256 {
		return false
	}
	match := regexp.MustCompile(`(?m)^coord = pathlib.Path\((.+)\)$`).FindStringSubmatch(string(raw))
	if len(match) != 2 {
		return false
	}
	root, err := strconv.Unquote(match[1])
	if err != nil || !filepath.IsAbs(root) {
		return false
	}
	for _, kind := range []string{"state", "attempt"} {
		if words[2] == filepath.Join(root, "post-finalize-1-"+kind+".json") && words[3] == filepath.Join(root, "post-finalize-2-"+kind+".json") {
			return true
		}
	}
	return false
}
