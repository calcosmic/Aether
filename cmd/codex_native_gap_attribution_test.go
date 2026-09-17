package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the very same pure resolver embedded in the disposable coordinator.
// These captures model the final165 spawn call/activity/result and metadata,
// not worker prose. They never launch a native worker.
func TestCodexNativeGapAttribution(t *testing.T) {
	parent := map[string]any{"type": "session_meta", "payload": map[string]any{"id": "parent", "cwd": "/fixture"}}
	child := map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child-uuid", "parent_thread_id": "parent", "cwd": "/fixture", "agent_path": "/root/brick", "agent_role": "aether-builder"}}
	turn := map[string]any{"turn_id": "turn"}
	call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": "spawn_agent", "call_id": "spawn", "arguments": `{"task_name":"brick","agent_type":"aether-builder","message":"opaque"}`, "internal_chat_message_metadata_passthrough": turn}}
	activity := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": "parent", "turn_id": "turn", "item": map[string]any{"type": "SubAgentActivity", "id": "spawn", "kind": "started", "agent_thread_id": "child-uuid", "agent_path": "/root/brick"}}}
	output := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "spawn", "output": `{"task_name":"/root/brick"}`, "internal_chat_message_metadata_passthrough": turn}}
	encode := func(v any) []byte {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	for _, mode := range []string{"valid", "uuid_input", "missing", "ambiguous", "wrong_thread", "wrong_turn", "reused_activity", "duplicate_result", "wrong_metadata", "wrong_role", "result_before_call"} {
		t.Run(mode, func(t *testing.T) {
			events := []any{parent, call, activity, output}
			metas := []any{child}
			raw := encode(events)
			meta := encode(metas)
			target := "/root/brick"
			switch mode {
			case "uuid_input":
				target = "child-uuid"
			case "missing":
				raw = encode([]any{parent, call, output})
			case "ambiguous":
				meta = encode([]any{child, child})
			case "wrong_thread":
				raw = bytes.Replace(raw, []byte(`"thread_id":"parent"`), []byte(`"thread_id":"foreign"`), 1)
			case "wrong_turn":
				raw = bytes.Replace(raw, []byte(`"turn_id":"turn"`), []byte(`"turn_id":"foreign"`), 1)
			case "reused_activity":
				raw = encode([]any{parent, call, activity, activity, output})
			case "duplicate_result":
				raw = encode([]any{parent, call, activity, output, output})
			case "wrong_metadata":
				meta = bytes.Replace(meta, []byte(`"parent_thread_id":"parent"`), []byte(`"parent_thread_id":"foreign"`), 1)
			case "wrong_role":
				meta = bytes.Replace(meta, []byte(`aether-builder`), []byte(`aether-watcher`), 1)
			case "result_before_call":
				raw = encode([]any{parent, output, call, activity})
			}
			script := nativeGapResolverScript(t)
			input := []byte(`{"events":` + string(raw) + `,"metadata":` + string(meta) + `,"target":"` + target + `","parent":"parent","cwd":"/fixture"}`)
			cmd := exec.Command("python3", "-c", script+"\nimport sys,json\nx=json.load(sys.stdin)\nprint(json.dumps(resolve_native_child(**x)))\n")
			cmd.Stdin = bytes.NewReader(input)
			out, err := cmd.CombinedOutput()
			if mode == "valid" || mode == "uuid_input" {
				if err != nil || !bytes.Contains(out, []byte(`"child_id": "child-uuid"`)) || !bytes.Contains(out, []byte(`"task_path": "/root/brick"`)) {
					t.Fatalf("valid mapping: %v %s", err, out)
				}
			} else if err == nil {
				t.Fatalf("%s accepted: %s", mode, out)
			}
		})
	}
	t.Run("empty_result_refuses_without_mutation", func(t *testing.T) {
		_, request := nativeBoundForTest(t)
		before := nativeJournalBytes(t)
		for _, result := range []*internalWorkerResult{nil, {}} {
			request.Result = result
			if _, err := runCodexNativeWorker("record", nativeRequestPath(t, request)); err == nil {
				t.Fatal("empty result accepted")
			}
			if !bytes.Equal(before, nativeJournalBytes(t)) {
				t.Fatal("empty result mutated journal")
			}
		}
	})
	t.Run("equal_reservation_receipt_and_conflicting_key", func(t *testing.T) {
		_, request := nativeAdmissionFixture(t)
		first, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request))
		if err != nil {
			t.Fatal(err)
		}
		before := nativeJournalBytes(t)
		second, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request))
		if err != nil || !second.Replay {
			t.Fatalf("reservation replay: %v", err)
		}
		a, _ := json.Marshal(first.Receipt)
		b, _ := json.Marshal(second.Receipt)
		if !bytes.Equal(a, b) || !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("reservation replay minted receipt or wrote journal")
		}
		request.HostSessionID = "different-host"
		if _, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request)); err == nil {
			t.Fatal("conflicting equal key accepted")
		}
		if !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("conflicting equal key wrote journal")
		}
	})
	t.Run("runtime_identity_and_empty_controls", func(t *testing.T) {
		TestCodexNativeWorkerTerminal(t)
	})
	t.Run("parent_only_edits_and_unclassified_writes", TestCodexNativeEvidenceDerivation)
	t.Run("unique_command_events", TestCodexNativeGapParentCoordination)
}

func nativeGapResolverScript(t *testing.T) string {
	t.Helper()
	start := strings.Index(nativeFixtureCoordinator, "def resolve_native_child(")
	end := strings.Index(nativeFixtureCoordinator, "def latest_native_terminal(")
	if start < 0 || end <= start {
		t.Fatal("coordinator lacks causal task-path resolver")
	}
	return "import json\n" + nativeFixtureCoordinator[start:end]
}

// Optional read-only replay preserves original final165 files and identity.
func TestCodexNativeGapCapturedAttribution(t *testing.T) {
	root := os.Getenv("AETHER_CODEX_GAP_CAPTURE")
	if root == "" {
		t.Skip("explicit immutable capture path required")
	}
	paths, err := filepath.Glob(filepath.Join(root, "home/.codex/sessions/*/*/*/*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var events []any
	var metas []any
	var parent string
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lines := bytes.Split(bytes.TrimSpace(raw), []byte{'\n'})
		var first map[string]any
		if err := json.Unmarshal(lines[0], &first); err != nil {
			t.Fatal(err)
		}
		meta := first["payload"].(map[string]any)
		if meta["parent_thread_id"] == nil {
			parent = meta["id"].(string)
			for _, line := range lines {
				var e any
				if err := json.Unmarshal(line, &e); err != nil {
					t.Fatal(err)
				}
				events = append(events, e)
			}
		} else {
			metas = append(metas, first)
		}
	}
	input, _ := json.Marshal(map[string]any{"events": events, "metadata": metas, "target": "/root/brick_46", "parent": parent, "cwd": events[0].(map[string]any)["payload"].(map[string]any)["cwd"]})
	cmd := exec.Command("python3", "-c", nativeGapResolverScript(t)+"\nimport sys\nprint(json.dumps(resolve_native_child(**json.load(sys.stdin))))\n")
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("captured final165: %v %s", err, out)
	}
	if !bytes.Contains(out, []byte("01a0afbd-0935-7512-ad12-825dd8c1c3ff")) {
		t.Fatalf("wrong captured child: %s", out)
	}
	t.Log(string(out))
	r := codexNativeLiveReceipt{BoundHostSessionID: parent, ChildID: "01a0afbd-0935-7512-ad12-825dd8c1c3ff", FixtureRoot: filepath.Join(root, "repository")}
	nativeCollectChildIdentity(&r, filepath.Join(root, "home"))
	if !r.ChildIdentityCorroborated || r.ChildTaskPath != "/root/brick_46" {
		t.Fatalf("observer did not retain both identities: %+v", r)
	}
}

func TestCodexNativeGapParentCoordination(t *testing.T) {
	for _, mode := range []string{"valid", "file_uri", "missing", "wrong_thread", "wrong_turn", "wrong_cwd", "wrong_argv", "reused_event", "duplicate_call"} {
		t.Run(mode, func(t *testing.T) {
			var raw []byte
			add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
			add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "parent", "cwd": "/fixture"}})
			call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "call", "input": `text(await tools.exec_command({cmd:"aether status",workdir:"/fixture"}));`, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
			add(call)
			if mode == "duplicate_call" {
				add(call)
			}
			thread, turn, cwd, command := "parent", "turn", "/fixture", "aether status"
			switch mode {
			case "file_uri":
				cwd = "file:///fixture"
			case "wrong_thread":
				thread = "sibling"
			case "wrong_turn":
				turn = "other"
			case "wrong_cwd":
				cwd = "/other"
			case "wrong_argv":
				command = "aether pause"
			}
			event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": "shell", "status": "completed", "cwd": cwd, "command": []string{"/bin/sh", "-c", command}, "exit_code": 0}}}
			if mode != "missing" {
				add(event)
			}
			add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "call", "output": []any{map[string]any{"type": "input_text", "text": "Script completed\n"}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
			if mode == "reused_event" {
				add(event)
			}
			r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "parent", FixtureRoot: "/fixture"}
			nativeInspectParentEvents(&r, raw)
			if r.ParentSubstitution != (mode != "valid" && mode != "file_uri") {
				t.Fatalf("mode %s classification: %+v", mode, r.ParentUnclassified)
			}
		})
	}
}

func TestCodexNativeGapBoundTaskPathCommand(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "coordinator.py")
	raw := []byte(nativeFixtureCoordinator)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"mapped_path", "mapped_uuid", "unmapped", "wrong_path", "wrong_parent", "review_watcher"} {
		t.Run(mode, func(t *testing.T) {
			r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", FixtureRoot: root, CoordinatorPath: path, CoordinatorSHA256: lifecycleDigest(raw), SessionID: "parent", BoundHostSessionID: "parent", ChildID: "child-uuid", ChildTaskPath: "/root/brick", ChildIdentityCorroborated: true}
			target := "/root/brick"
			switch mode {
			case "mapped_uuid":
				target = "child-uuid"
			case "unmapped":
				r.ChildIdentityCorroborated = false
			case "wrong_path":
				target = "/root/other"
			case "wrong_parent":
				r.BoundHostSessionID = "other"
			case "review_watcher":
				r.Workers = []codexNativeLiveReceipt{{ChildID: "watcher-uuid", ChildTaskPath: "/root/keen", ChildIdentityCorroborated: true, BoundHostSessionID: "parent"}}
				target = "/root/keen"
			}
			got := nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", "python3 " + path + " bind " + target}, root)
			want := mode == "mapped_path" || mode == "mapped_uuid" || mode == "review_watcher"
			if got != want {
				t.Fatalf("%s classified=%v want=%v", mode, got, want)
			}
		})
	}
}

func TestCodexNativeGapObservedRefusal(t *testing.T) {
	for _, mode := range []string{"actual_failure", "failed_zero", "missing_exit", "wrong_thread"} {
		t.Run(mode, func(t *testing.T) {
			var raw []byte
			add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
			add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "call_id": "call", "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
			item := map[string]any{"type": "CommandExecution", "id": "event", "status": "failed", "command": []string{"/bin/sh", "-c", "python3 /fixture/coordinator.py empty-result"}, "cwd": "file:///fixture", "exit_code": 1}
			thread := "parent"
			switch mode {
			case "failed_zero":
				item["exit_code"] = 0
			case "missing_exit":
				delete(item, "exit_code")
			case "wrong_thread":
				thread = "foreign"
			}
			add(map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": "turn", "item": item}})
			add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "call", "output": []any{map[string]any{"type": "input_text", "text": "Script completed\n"}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
			got := nativeCorroboratedBatch(raw, "parent", "call", []nativeRecordedShellCommand{{Command: "python3 /fixture/coordinator.py empty-result", Cwd: "/fixture"}})
			if got != (mode == "actual_failure") {
				t.Fatalf("%s observed=%v", mode, got)
			}
		})
	}
}

func TestCodexNativeGapUncachedCheckSurvivesPlainRecheck(t *testing.T) {
	for _, mode := range []string{"valid", "no_prior", "wrong_thread", "wrong_turn", "duplicate_event", "failure", "missing_output", "later_edit"} {
		t.Run(mode, func(t *testing.T) {
			r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", Caste: "builder"}
			var raw []byte
			add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
			add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "cwd": "/fixture", "agent_role": "aether-builder"}})
			if mode != "no_prior" {
				raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "CommandExecution", "status": "completed", "id": "uncached", "command": []string{"/bin/sh", "-c", "go test ./... -json -count=1"}, "cwd": "/fixture", "exit_code": 0, "aggregated_output": "{\"Action\":\"run\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\"}\n"})...)
			}
			add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": "child", "turn_id": "turn"}})
			add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "plain", "input": `text(await tools.exec_command({cmd:"go test ./...",workdir:"/fixture"}));`, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
			thread, turn, exit, status := "child", "turn", 0, "completed"
			switch mode {
			case "wrong_thread":
				thread = "foreign"
			case "wrong_turn":
				turn = "foreign"
			case "failure":
				exit = 1
				status = "failed"
			}
			event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": "plain-event", "status": status, "command": []string{"/bin/sh", "-c", "go test ./..."}, "cwd": "/fixture", "exit_code": exit, "aggregated_output": "ok example.invalid/nativefixture (cached)"}}}
			add(event)
			if mode == "duplicate_event" {
				add(event)
			}
			if mode == "later_edit" {
				raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{}})...)
			}
			if mode != "missing_output" {
				result, _ := json.Marshal(map[string]any{"exit_code": exit, "output": "ok example.invalid/nativefixture (cached)"})
				add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "plain", "output": []any{map[string]any{"type": "input_text", "text": "Script completed\n"}, map[string]any{"type": "input_text", "text": string(result)}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
			}
			nativeInspectChildEvents(&r, raw)
			if r.ChecksPassed != (mode == "valid") {
				t.Fatalf("%s checks=%v", mode, r.ChecksPassed)
			}
		})
	}
}

func TestCodexNativeGapSelectedSkillDelivery(t *testing.T) {
	for _, mode := range []string{"valid", "manual_user_text", "wrong_turn", "wrong_thread", "wrong_path", "changed_bytes", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "SKILL.md")
			skill := []byte("# ant-build\nExact installed bytes.\n")
			if err := os.WriteFile(path, skill, 0600); err != nil {
				t.Fatal(err)
			}
			text := "<skill>\n<name>ant-build</name>\n<path>" + path + "</path>\n" + string(skill) + "\n</skill>"
			kinds := []string{"skills.selected_skill_instructions"}
			turn, thread := "turn", "parent"
			switch mode {
			case "manual_user_text":
				kinds = nil
			case "wrong_turn":
				turn = "foreign"
			case "wrong_thread":
				thread = "foreign"
			case "wrong_path":
				text = strings.Replace(text, path, "/other/SKILL.md", 1)
			case "changed_bytes":
				text = strings.Replace(text, "Exact", "Changed", 1)
			}
			var raw []byte
			add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
			add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "parent", "cwd": root}})
			add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": thread, "turn_id": "turn"}})
			message := map[string]any{"type": "response_item", "payload": map[string]any{"type": "message", "id": "skill-message", "role": "user", "content": []any{map[string]any{"type": "input_text", "text": text}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": turn, "content_item_kinds": kinds}}}
			add(message)
			if mode == "duplicate" {
				add(message)
			}
			r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "parent", FixtureRoot: root, SkillPath: path}
			nativeInspectParentEvents(&r, raw)
			if r.SkillRead != (mode == "valid") {
				t.Fatalf("%s selected skill=%v", mode, r.SkillRead)
			}
		})
	}
}

func TestCodexNativeGapRecordedFailedRefusal(t *testing.T) {
	for _, mode := range []string{"failed", "wrong_thread", "wrong_cwd", "duplicate", "zero_exit", "wrong_operation"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			fixture := filepath.Join(root, "repository")
			log := filepath.Join(root, "home/.codex/sessions/2026/09/17/rollout-parent.jsonl")
			if err := os.MkdirAll(filepath.Dir(log), 0700); err != nil {
				t.Fatal(err)
			}
			coordinator := filepath.Join(fixture, "coordinator.py")
			thread, cwd, operation, exit := "parent", fixture, "empty-result", 1
			switch mode {
			case "wrong_thread":
				thread = "foreign"
			case "wrong_cwd":
				cwd = "/other"
			case "zero_exit":
				exit = 0
			case "wrong_operation":
				operation = "record"
			}
			event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": "turn", "item": map[string]any{"type": "CommandExecution", "id": "refusal", "status": "failed", "cwd": cwd, "command": []string{"/bin/sh", "-c", "python3 " + coordinator + " " + operation + " 0"}, "exit_code": exit, "aggregated_output": "nonempty terminal result"}}}
			raw, _ := json.Marshal(event)
			raw = append(raw, '\n')
			if mode == "duplicate" {
				raw = append(raw, raw...)
			}
			if err := os.WriteFile(log, raw, 0600); err != nil {
				t.Fatal(err)
			}
			r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", FixtureRoot: fixture, CoordinatorPath: coordinator}
			got := nativeObservedFixtureOperation(r, codexNativeLiveReceipt{BoundHostSessionID: "parent"}, 0, "empty-result", true)
			if got != (mode == "failed") {
				t.Fatalf("%s observed=%v", mode, got)
			}
		})
	}
}
