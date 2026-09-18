package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This is a measured evidence projection, never a runtime observation writer.
// In particular an interrupt result reporting previous_status=running is only
// a request. Neither it nor a disappearing process proves terminal cancelled.
type nativeGapCancellationEvidence struct {
	Parent                string            `json:"parent"`
	Child                 string            `json:"child"`
	RequestEvents         map[string]string `json:"request_events"`
	AcknowledgementID     string            `json:"acknowledgement_id,omitempty"`
	AcknowledgementSHA256 string            `json:"acknowledgement_sha256,omitempty"`
	AcknowledgedAt        string            `json:"acknowledged_at,omitempty"`
	RuntimeRequested      bool              `json:"runtime_requested"`
	RuntimeCancelled      bool              `json:"runtime_cancelled"`
	PostAckActivity       bool              `json:"post_ack_activity"`
	NoPostAckWrites       bool              `json:"no_post_ack_writes"`
	Qualified             bool              `json:"qualified"`
	Gaps                  []string          `json:"gaps"`
}

func nativeGapCancellationFacts(parentRaw, childRaw []byte, parent, child string, observations []codexNativeHostObservation) nativeGapCancellationEvidence {
	e := nativeGapCancellationEvidence{Parent: parent, Child: child, RequestEvents: nativeInterruptRequestEvidence(parentRaw, parent, child), Gaps: []string{}}
	// Parent rollouts can contain inherited ancestor calls. Only this parent's
	// attributed turns may supply the interrupt call and its result.
	owned := nativeOwnedHostTurns(parentRaw, parent)
	calls := map[string]bool{}
	for _, line := range bytes.Split(parentRaw, []byte{'\n'}) {
		var event struct {
			Type    string
			Payload struct {
				Type, Name string
				Call       string `json:"call_id"`
				Metadata   struct {
					Turn string `json:"turn_id"`
				} `json:"internal_chat_message_metadata_passthrough"`
			}
		}
		if json.Unmarshal(line, &event) != nil || event.Type != "response_item" {
			continue
		}
		p := event.Payload
		if p.Type == "function_call" && strings.HasSuffix(p.Name, "interrupt_agent") {
			calls[p.Call] = owned[p.Metadata.Turn]
		}
		if p.Type == "function_call_output" && (!calls[p.Call] || !owned[p.Metadata.Turn]) {
			delete(e.RequestEvents, p.Call)
		}
	}
	var requestedAt time.Time
	for _, line := range bytes.Split(parentRaw, []byte{'\n'}) {
		var event struct {
			Timestamp string `json:"timestamp"`
			Type      string `json:"type"`
			Payload   struct {
				Type   string `json:"type"`
				Thread string `json:"thread_id"`
				Call   string `json:"call_id"`
				Item   struct {
					Type, ID, Kind string
					Child          string `json:"agent_thread_id"`
				} `json:"item"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &event) != nil {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, event.Timestamp)
		if err != nil {
			continue
		}
		p := event.Payload
		if digest, ok := e.RequestEvents[p.Call]; ok && digest == lifecycleDigest(line) {
			for _, o := range observations {
				if o.ChildID == child && o.Status == "cancel_requested" && o.SourceEventID == p.Call && strings.TrimPrefix(o.SourceEventSHA256, "sha256:") == strings.TrimPrefix(digest, "sha256:") {
					e.RuntimeRequested = true
					requestedAt = at
				}
			}
		}
		if !requestedAt.IsZero() && !at.Before(requestedAt) && event.Type == "event_msg" && p.Type == "item_completed" && p.Thread == parent && p.Item.Type == "SubAgentActivity" && p.Item.Child == child && p.Item.Kind == "cancelled" && p.Item.ID != "" {
			e.AcknowledgementID = p.Item.ID
			e.AcknowledgementSHA256 = lifecycleDigest(line)
			e.AcknowledgedAt = event.Timestamp
		}
	}
	for _, o := range observations {
		if o.ChildID == child && o.Status == "cancelled" && e.AcknowledgementID != "" && o.SourceEventID == e.AcknowledgementID && strings.TrimPrefix(o.SourceEventSHA256, "sha256:") == strings.TrimPrefix(e.AcknowledgementSHA256, "sha256:") {
			e.RuntimeCancelled = true
		}
	}
	// Unknown or incomplete child exports cannot establish an absence of writes.
	var meta nativeHostEvent
	validChild := json.Unmarshal(bytes.SplitN(childRaw, []byte{'\n'}, 2)[0], &meta) == nil && meta.Type == "session_meta" && meta.Payload.ID == child && meta.Payload.ParentThreadID == parent
	ack, _ := time.Parse(time.RFC3339Nano, e.AcknowledgedAt)
	turns := nativeOwnedHostTurns(childRaw, child)
	observedAfter := false
	unknownTime := false
	for _, line := range bytes.Split(childRaw, []byte{'\n'}) {
		var event struct {
			Type, Timestamp string
			Payload         struct {
				Type     string
				Thread   string `json:"thread_id"`
				Metadata struct {
					Turn string `json:"turn_id"`
				} `json:"internal_chat_message_metadata_passthrough"`
				Item struct{ Type string } `json:"item"`
			}
		}
		if json.Unmarshal(line, &event) != nil {
			continue
		}
		p := event.Payload
		owned := p.Thread == child || turns[p.Metadata.Turn]
		if !owned {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, event.Timestamp)
		activity := p.Type == "function_call" || p.Type == "custom_tool_call" || p.Item.Type == "CommandExecution" || p.Item.Type == "FileChange"
		if err != nil && activity {
			unknownTime = true
		}
		if err == nil && !ack.IsZero() && !at.Before(ack) {
			observedAfter = true
			if activity {
				e.PostAckActivity = true
			}
		}
	}
	e.NoPostAckWrites = validChild && observedAfter && !unknownTime && !e.PostAckActivity && !ack.IsZero()
	if !e.RuntimeRequested {
		e.Gaps = append(e.Gaps, "actual bound runtime interrupt request missing")
	}
	if e.AcknowledgementID == "" {
		e.Gaps = append(e.Gaps, "actual child-correlated cancelled acknowledgement missing; interrupt is request-only")
	}
	if !e.RuntimeCancelled {
		e.Gaps = append(e.Gaps, "runtime cancelled terminal is not corroborated")
	}
	if !e.NoPostAckWrites {
		e.Gaps = append(e.Gaps, "no-post-ack-write interval unproved or contains activity")
	}
	e.Qualified = len(e.Gaps) == 0
	return e
}

func nativeGapCollectCancellation(r codexNativeLiveReceipt, runRoot, home string) nativeGapCancellationEvidence {
	var attempt buildAttemptRecord
	raw, _ := r.readEvidence(r.AttemptPath)
	_ = json.Unmarshal(raw, &attempt)
	child := r.ChildID
	var observations []codexNativeHostObservation
	if len(attempt.WorkerRuns) == 1 && attempt.WorkerRuns[0].Native != nil {
		child = attempt.WorkerRuns[0].Native.ChildID
		observations = attempt.WorkerRuns[0].Native.Observations
	}
	var parentRaw, childRaw []byte
	paths, _ := filepath.Glob(filepath.Join(home, ".codex", "sessions", "*", "*", "*", "*.jsonl"))
	for _, path := range paths {
		raw, err := r.readEvidence(path)
		if err != nil {
			continue
		}
		var meta nativeHostEvent
		if json.Unmarshal(bytes.SplitN(raw, []byte{'\n'}, 2)[0], &meta) != nil {
			continue
		}
		if meta.Payload.ID == r.SessionID {
			parentRaw = raw
		}
		if meta.Payload.ID == child {
			childRaw = raw
		}
	}
	e := nativeGapCancellationFacts(parentRaw, childRaw, r.SessionID, child, observations)
	if !r.ChildIdentityCorroborated || r.NativeSpawnCount != 1 || r.ParentSubstitution {
		e.Gaps = append(e.Gaps, "unique actual bound child identity or parent attribution unproved")
		e.Qualified = false
	}
	return e
}

func TestCodexNativeGapControls(t *testing.T) {
	t.Run("sentinel-attempts", func(t *testing.T) {
		root := t.TempDir()
		workspace := filepath.Join(root, "workspace")
		probe := filepath.Join(workspace, ".aether", "capability-write.py")
		if err := os.MkdirAll(filepath.Dir(probe), 0700); err != nil {
			t.Fatal(err)
		}
		liveSkillWrite(t, probe, []byte("source-pinned test probe"))
		nativeSnapshotControlSentinels(t, root, workspace, "before")
		nativeSnapshotControlSentinels(t, root, workspace, "after")
		facts := map[string]bool{}
		for _, name := range []string{"inside", "outside"} {
			facts[name+"_write_attempted"] = true
			facts[name+"_write_denied"] = true
			facts[name+"_denial_target_correlated"] = true
		}
		r := codexNativeLiveReceipt{}
		if err := nativeValidateControlSentinels(r, root, workspace, facts); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"inside_write_attempted", "outside_write_attempted", "outside_denial_target_correlated"} {
			t.Run(key, func(t *testing.T) {
				facts[key] = false
				defer func() { facts[key] = true }()
				if nativeValidateControlSentinels(r, root, workspace, facts) == nil {
					t.Fatal("missing actual evidence accepted")
				}
			})
		}
		for _, key := range []string{"outside_write_repeated", "outside_write_allowed"} {
			t.Run(key, func(t *testing.T) {
				facts[key] = true
				defer delete(facts, key)
				if nativeValidateControlSentinels(r, root, workspace, facts) == nil {
					t.Fatal("ambiguous/repeated evidence accepted")
				}
			})
		}
		if err := os.MkdirAll(filepath.Join(root, "outside-workspace"), 0700); err != nil {
			t.Fatal(err)
		}
		liveSkillWrite(t, filepath.Join(root, "outside-workspace", "outside-sentinel.txt"), []byte("late write"))
		if nativeValidateControlSentinels(r, root, workspace, facts) == nil {
			t.Fatal("late sentinel write accepted")
		}
	})
	t.Run("raw-attempt-controls", func(t *testing.T) {
		TestCodexNativeControlToolEvidence(t)
		const child = "child"
		command := []string{"/bin/sh", "-c", "python3 /fixture/.aether/capability-write.py /outside/sentinel"}
		for _, tc := range []struct {
			name, output string
			correlated   bool
		}{
			{"exact-denial", "PROBE_CWD=/fixture\nPROBE_TARGET=/outside/sentinel\nPermissionError: [Errno 1] Operation not permitted: '/outside/sentinel'\n", true},
			{"host-refused-before-python", "host refused command execution: Operation not permitted", false},
			{"unrelated-denial", "PROBE_CWD=/fixture\nPROBE_TARGET=/outside/sentinel\nPermissionError: [Errno 1] Operation not permitted: '/unrelated'\n", false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				raw := []byte(`{"type":"session_meta","payload":{"id":"child"}}` + "\n")
				raw = append(raw, nativeEvidenceEvent(t, "item_completed", child, map[string]any{"type": "CommandExecution", "status": "completed", "command": command, "cwd": "/fixture", "exit_code": 1, "aggregated_output": tc.output})...)
				facts := nativeControlToolEvidence(raw, child, "/fixture", "/fixture/.aether/capability-write.py", map[string]string{"outside": "/outside/sentinel"})
				if !facts["outside_write_attempted"] || facts["outside_denial_target_correlated"] != tc.correlated {
					t.Fatalf("unrelated denial accepted: %v", facts)
				}
			})
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		event := func(at int, typ string, payload any) []byte {
			raw, _ := json.Marshal(map[string]any{"timestamp": fmt.Sprintf("2026-09-18T00:00:%02dZ", at), "type": typ, "payload": payload})
			return append(raw, '\n')
		}
		parent := event(0, "session_meta", map[string]any{"id": "parent"})
		parent = append(parent, event(0, "event_msg", map[string]any{"thread_id": "parent", "turn_id": "parent-turn"})...)
		metadata := map[string]any{"turn_id": "parent-turn"}
		parent = append(parent, event(1, "response_item", map[string]any{"type": "function_call", "name": "collaboration.interrupt_agent", "call_id": "interrupt", "arguments": `{"target":"child"}`, "internal_chat_message_metadata_passthrough": metadata})...)
		result := event(2, "response_item", map[string]any{"type": "function_call_output", "call_id": "interrupt", "output": `{"previous_status":"running"}`, "internal_chat_message_metadata_passthrough": metadata})
		parent = append(parent, result...)
		ack := event(3, "event_msg", map[string]any{"type": "item_completed", "thread_id": "parent", "item": map[string]any{"type": "SubAgentActivity", "id": "ack", "kind": "cancelled", "agent_thread_id": "child"}})
		requestOnly := append([]byte(nil), parent...)
		parent = append(parent, ack...)
		child := event(0, "session_meta", map[string]any{"id": "child", "parent_thread_id": "parent"})
		child = append(child, event(4, "event_msg", map[string]any{"type": "turn_aborted", "thread_id": "child"})...)
		observations := []codexNativeHostObservation{{Status: "cancel_requested", ChildID: "child", SourceEventID: "interrupt", SourceEventSHA256: lifecycleDigest(bytes.TrimSpace(result))}, {Status: "cancelled", ChildID: "child", SourceEventID: "ack", SourceEventSHA256: lifecycleDigest(bytes.TrimSpace(ack))}}
		if got := nativeGapCancellationFacts(parent, child, "parent", "child", observations); !got.Qualified {
			t.Fatalf("valid synthetic control failed: %+v", got)
		}
		for _, tc := range []struct {
			name          string
			parent, child []byte
			obs           []codexNativeHostObservation
		}{
			{"request-only", requestOnly, child, observations[:1]},
			{"wrong-child-ack", bytes.ReplaceAll(parent, []byte(`"agent_thread_id":"child"`), []byte(`"agent_thread_id":"other"`)), child, observations},
			{"unrelated-request", bytes.ReplaceAll(parent, []byte(`\"target\":\"child\"`), []byte(`\"target\":\"other\"`)), child, observations},
			{"late-write", parent, append(append([]byte(nil), child...), event(5, "event_msg", map[string]any{"type": "item_completed", "thread_id": "child", "item": map[string]any{"type": "FileChange"}})...), observations},
			{"missing-child-export", parent, nil, observations},
			{"runtime-request-only", parent, child, observations[:1]},
		} {
			t.Run(tc.name, func(t *testing.T) {
				if got := nativeGapCancellationFacts(tc.parent, tc.child, "parent", "child", tc.obs); got.Qualified {
					t.Fatalf("incomplete cancellation accepted: %+v", got)
				}
			})
		}
	})
	t.Run("runtime-refusals", TestCodexNativeWorkerCapabilities)
	t.Run("usage-unreported", TestCodexNativeUsageUnreported)
}

// Exercise the candidate's real non-launching admission CLI before the host
// probe. These runtime refusals are separate from inherited host permissions.
func nativeGapCaptureControlRefusals(t *testing.T, r codexNativeLiveReceipt, runRoot, coord string, env []string) {
	t.Helper()
	manifestPath := filepath.Join(runRoot, "control-runtime-manifest.json")
	liveSkillRuntime(t, r.FixtureRoot, env, manifestPath, r.CandidatePath, "build", "1", "--plan-only")
	raw, err := os.ReadFile(manifestPath)
	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Manifest codexBuildManifest `json:"dispatch_manifest"`
		} `json:"result"`
	}
	if err != nil || json.Unmarshal(raw, &envelope) != nil || !envelope.OK || len(envelope.Result.Manifest.Dispatches) != 1 || envelope.Result.Manifest.ExecutionBinding == nil {
		t.Fatal("control refusal fixture lacks actual candidate manifest")
	}
	m := envelope.Result.Manifest
	d := m.Dispatches[0]
	request := codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *m.ExecutionBinding, WorkerName: d.Name, TaskID: normalizedDispatchTaskID(d), HostSessionID: "fixture-control-refusal-no-host-launch", Workspace: r.FixtureRoot, HostPermission: "workspace_write"}
	for _, name := range []string{"read-only", "separate-workspace", "governed-nesting"} {
		invalid := request
		switch name {
		case "read-only":
			invalid.HostPermission = "repository_read_only"
		case "separate-workspace":
			invalid.Workspace = filepath.Join(runRoot, "outside-workspace")
		case "governed-nesting":
			invalid.RequireGovernedNesting = true
		}
		path := filepath.Join(coord, "control-"+name+"-request.json")
		liveSkillWriteJSON(t, path, invalid)
		before := nativeFixtureStateInventory(r.FixtureRoot)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		argv := []string{"codex-native-worker", "reserve", "--request", path}
		cmd := exec.CommandContext(ctx, r.CandidatePath, argv...)
		cmd.Dir = r.FixtureRoot
		cmd.Env = env
		output, runErr := cmd.CombinedOutput()
		cancel()
		after := nativeFixtureStateInventory(r.FixtureRoot)
		var response struct {
			OK     bool `json:"ok"`
			Result struct {
				LaunchAllowed bool `json:"launch_allowed"`
			} `json:"result"`
		}
		decoded := json.Unmarshal(output, &response) == nil
		exit := -1
		if cmd.ProcessState != nil {
			exit = cmd.ProcessState.ExitCode()
		}
		b, _ := json.Marshal(before)
		a, _ := json.Marshal(after)
		passed := runErr != nil && exit > 0 && decoded && !response.OK && !response.Result.LaunchAllowed && bytes.Equal(a, b) && strings.Contains(string(output), "unsupported")
		if name == "governed-nesting" {
			passed = runErr != nil && exit > 0 && decoded && !response.OK && !response.Result.LaunchAllowed && bytes.Equal(a, b) && strings.Contains(string(output), "cannot provide requested Aether-governed nesting")
		}
		liveSkillWriteJSON(t, filepath.Join(runRoot, "control-refusal-"+name+".json"), map[string]any{"argv": append([]string{r.CandidatePath}, argv...), "request": invalid, "exit": exit, "output": string(output), "before": before, "after": after, "passed": passed, "launch_allowed": response.Result.LaunchAllowed})
		if !passed {
			t.Fatalf("actual unsupported-control refusal incomplete: %s: %s", name, output)
		}
	}
}

func TestCodexNativeGapControlCommandCorrelation(t *testing.T) {
	const child, turn, workspace, probe, target = "child", "turn", "/fixture", "/fixture/probe.py", "/fixture/inside"
	const output = "PROBE_CWD=/fixture\nPROBE_TARGET=/fixture/inside\nPROBE_WRITE_SUCCEEDED\n"
	encode := func(typ string, payload any) []byte {
		b, _ := json.Marshal(map[string]any{"type": typ, "payload": payload})
		return append(b, '\n')
	}
	metadata := map[string]any{"turn_id": turn}
	preamble := append(encode("session_meta", map[string]any{"id": child}), encode("event_msg", map[string]any{"thread_id": child, "turn_id": turn})...)
	call := func(id string) []byte {
		return encode("response_item", map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": id, "input": `const result = await tools.exec_command({cmd:"python3 /fixture/probe.py /fixture/inside",workdir:"/fixture",max_output_tokens:2000}); text(result);`, "internal_chat_message_metadata_passthrough": metadata})
	}
	normalized := func(eventTurn string) []byte {
		return encode("event_msg", map[string]any{"type": "item_completed", "thread_id": child, "turn_id": eventTurn, "item": map[string]any{"type": "CommandExecution", "id": "exec-id", "status": "completed", "exit_code": 0, "command": []string{"/bin/sh", "-c", "python3 /fixture/probe.py /fixture/inside"}, "cwd": workspace, "aggregated_output": output}})
	}
	result := func(id, stdout string) []byte {
		b, _ := json.Marshal(map[string]any{"exit_code": 0, "output": stdout})
		return encode("response_item", map[string]any{"type": "custom_tool_call_output", "call_id": id, "internal_chat_message_metadata_passthrough": metadata, "output": []any{map[string]any{"type": "input_text", "text": "Script completed\n"}, map[string]any{"type": "input_text", "text": string(b)}}})
	}
	join := func(parts ...[]byte) []byte {
		var b []byte
		for _, p := range parts {
			b = append(b, p...)
		}
		return b
	}
	for _, tc := range []struct {
		name     string
		raw      []byte
		repeated bool
	}{
		{"one-call-two-representations", join(preamble, call("c1"), normalized(turn), result("c1", output)), false},
		{"two-real-calls", join(preamble, call("c1"), normalized(turn), result("c1", output), call("c2"), result("c2", output)), true},
		{"different-turn-not-an-echo", join(preamble, call("c1"), normalized("other-turn"), result("c1", output)), true},
		{"different-output-not-an-echo", join(preamble, call("c1"), normalized(turn), result("c1", output+"different")), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facts := nativeControlToolEvidence(tc.raw, child, workspace, probe, map[string]string{"inside": target})
			if !facts["inside_write_allowed"] || facts["inside_write_repeated"] != tc.repeated {
				t.Fatalf("bad event correlation: %+v", facts)
			}
		})
	}
}
