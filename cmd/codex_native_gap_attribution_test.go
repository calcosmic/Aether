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
