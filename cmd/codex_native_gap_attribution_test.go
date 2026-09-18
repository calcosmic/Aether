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
	t.Run("literal-command-attribution", TestCodexNativeGapLiteralCommandAttribution)
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

func TestCodexNativeGapLiteralCommandAttribution(t *testing.T) {
	t.Run("quoted-bytes-unchanged", func(t *testing.T) {
		for _, tc := range []struct{ input, want string }{
			{`text(await tools.exec_command({cmd:'printf "café 日本語 é"',workdir:'/fixture'}));`, `printf "café 日本語 é"`},
			{`text(await tools.exec_command({cmd:'printf \'quoted\' \\path\n',workdir:'/fixture'}));`, "printf 'quoted' \\path\n"},
			{`text(await tools.exec_command({cmd:'printf "{cmd: untouched}"',workdir:'/fixture'}));`, `printf "{cmd: untouched}"`},
			{`text(await tools.exec_command({cmd:'printf "\u65e5\u672c\u8a9e \ud83d\ude80"',workdir:'/fixture'}));`, `printf "日本語 🚀"`},
		} {
			got, ok := nativeCodeModeCommands(tc.input, "/fixture")
			if !ok || len(got) != 1 || got[0].Command != tc.want {
				t.Fatalf("literal bytes changed: got=%+v ok=%v want=%q", got, ok, tc.want)
			}
		}
	})
	for _, input := range []string{
		`text(await tools.exec_command({cmd:'aether spawn-log --help',max_output_tokens:2000}));`,
		`text(await tools.exec_command({'cmd':'aether spawn-log --name Brick-46 --caste builder --parent Queen --phase 1 --task "Fix Clamp"',workdir:'/fixture',max_output_tokens:1000}));`,
		`text(await tools.exec_command({cmd:"aether status"})); text(await tools.exec_command({cmd:'aether spawn-complete --name Brick-46 --status completed --summary "Fixed Clamp"'}));`,
		`const result = await tools.exec_command({cmd:'aether status',workdir:'/fixture'}); text(result);`,
	} {
		t.Run(input, func(t *testing.T) {
			if _, ok := nativeCodeModeCommands(input, "/fixture"); !ok {
				t.Fatal("observed literal grammar rejected")
			}
		})
	}
	for _, object := range []string{
		`{cmd:'aether status' + suffix}`, `{cmd:command}`, "{cmd:`aether status`}",
		`{...defaults,cmd:'aether status'}`, `{['cmd']:'aether status'}`, `{cmd:'aether status',cmd:'touch clamp.go'}`,
		`{"cmd":"aether status","cmd":"touch clamp.go"}`, `{cmd:'aether status',workdir:'/fixture',workdir:'/other'}`,
		`{cmd:'aether status',max_output_tokens:expression()}`, `{cmd:'aether status',max_output_tokens:{x:1}}`,
		`{cmd:'aether status',yield_time_ms:NaN}`, `{cmd:'aether status',unknown:1}`, `{cmd:'aether status',__proto__:{}}`,
		`{cmd:'aether status',"\u0063md":"touch clamp.go"}`, `{cmd:'printf "\ud800"'}`, `{cmd:'printf "\udc00"'}`,
		`{cmd:'printf "\ud800\u0041"'}`, `{cmd:'aether status',max_output_tokens:01}`, `{cmd:'aether status',max_output_tokens:1.0}`,
	} {
		t.Run("reject-"+object, func(t *testing.T) {
			if _, ok := nativeCodeModeCommands("text(await tools.exec_command("+object+"));", "/fixture"); ok {
				t.Fatal("nonliteral or ambiguous object accepted")
			}
		})
	}
	for _, input := range []string{
		`text(await tools.exec_command({cmd:'aether status'})); mutate();`,
		`text(await other.exec_command({cmd:'aether status'}));`,
		`const x=await tools.exec_command({cmd:'aether status'});text(y);`,
		`text(await tools.exec_command({cmd:'aether status'})); text(await tools.apply_patch('x'));`,
	} {
		t.Run("reject-extra-"+input, func(t *testing.T) {
			if _, ok := nativeCodeModeCommands(input, "/fixture"); ok {
				t.Fatal("unsupported execution accepted")
			}
		})
	}
	// Syntax decoding must not itself authorize a parent edit or required check.
	r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "parent", FixtureRoot: "/fixture"}
	for _, cmd := range []string{"go test ./... -json -count=1", "touch unrelated", "python3 -c 'print(1)'", "echo wrong > clamp.go"} {
		if nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", cmd}, "/fixture") {
			t.Fatalf("parent substitution accepted: %s", cmd)
		}
	}
}

func TestCodexNativeCodeModeOutputProjection(t *testing.T) {
	// Actual Plan 26 wrapper shape, with disposable paths shortened for this
	// deterministic decoder control. Printing output never supplies exit proof.
	input := `const r = await tools.exec_command({"cmd":"aether command-guide build --platform codex","workdir":"/fixture","yield_time_ms":30000,"max_output_tokens":30000}); text(r.output);`
	for _, tc := range []struct {
		name, input string
		want        bool
	}{
		{"observed", input, true},
		{"same-variable-renamed", strings.Replace(strings.Replace(input, "const r =", "const result =", 1), "text(r.output)", "text(result.output)", 1), true},
		{"single-quoted-command", `const result = await tools.exec_command({cmd:'aether status',workdir:'/fixture'}); text(result.output);`, true},
		{"different-variable", strings.Replace(input, "text(r.output)", "text(other.output)", 1), false},
		{"different-property", strings.Replace(input, ".output", ".exit_code", 1), false},
		{"computed-property", strings.Replace(input, ".output", `["output"]`, 1), false},
		{"rewritten-output", strings.Replace(input, "r.output", `r.output + "fabricated"`, 1), false},
		{"method-call", strings.Replace(input, "r.output", "r.output.trim()", 1), false},
		{"reassignment", strings.Replace(input, " text(", ` r.output = "forged"; text(`, 1), false},
		{"extra-command", input + `text(await tools.exec_command({cmd:'touch clamp.go',workdir:'/fixture'}));`, false},
		{"preceding-command", `text(await tools.exec_command({cmd:'aether status',workdir:'/fixture'}));` + input, false},
		{"extra-js", input + " mutate();", false},
		{"unawaited", strings.Replace(input, "await ", "", 1), false},
		{"other-tool", strings.Replace(input, "tools.exec_command", "other.exec_command", 1), false},
		{"computed-command", strings.Replace(input, `"aether command-guide build --platform codex"`, `command`, 1), false},
		{"duplicate-key", strings.Replace(input, `"cmd":`, `"cmd":"touch clamp.go","cmd":`, 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commands, ok := nativeCodeModeCommands(tc.input, "/fixture")
			if ok != tc.want {
				t.Fatalf("projection accepted=%v want=%v: %+v", ok, tc.want, commands)
			}
			command, cwd, single := nativeCodeModeCommand(tc.input)
			if single != tc.want || (single && (len(commands) != 1 || command != commands[0].Command || cwd != commands[0].Cwd)) {
				t.Fatalf("single-command decoder diverged: command=%q cwd=%q accepted=%v", command, cwd, single)
			}
		})
	}
	if _, ok := nativeCodeModeCommands(strings.Replace(input, `,"workdir":"/fixture"`, "", 1), ""); ok {
		t.Fatal("output projection manufactured missing workspace attribution")
	}
}

func TestCodexNativeGapDocsSearch(t *testing.T) {
	observed := `rg "spawn-log" .aether -g '*.md' -g '*.json' -g '*.yaml'`
	for _, tc := range []struct {
		name, command, cwd string
		want               bool
	}{
		{"observed", observed, "/fixture", true},
		{"help-already-supported", "aether spawn-log --help", "/fixture", true},
		{"foreign-cwd", observed, "/other", false},
		{"missing-cwd", observed, "", false},
		{"foreign-root", strings.Replace(observed, ".aether", "/other/.aether", 1), "/fixture", false},
		{"escaping-root", strings.Replace(observed, ".aether", "../.aether", 1), "/fixture", false},
		{"preprocessor", observed + " --pre mutate", "/fixture", false},
		{"follow-symlinks", observed + " --follow", "/fixture", false},
		{"hidden-files", observed + " --hidden", "/fixture", false},
		{"extra-command", observed + "; touch clamp.go", "/fixture", false},
		{"redirect", observed + " > clamp.go", "/fixture", false},
		{"expansion", strings.Replace(observed, "spawn-log", "$(touch clamp.go)", 1), "/fixture", false},
		{"glob-expansion", strings.Replace(observed, "'*.md'", "*.md", 1), "/fixture", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := codexNativeLiveReceipt{FixtureRoot: "/fixture"}
			if got := nativeParentCoordinationCommand(&r, []string{"/bin/zsh", "-lc", tc.command}, tc.cwd); got != tc.want {
				t.Fatalf("docs query accepted=%v want=%v", got, tc.want)
			}
		})
	}
}

// Explicit immutable replay of the observed single-quoted call shapes. This
// diagnoses parser behavior only and never upgrades the original receipt.
func TestCodexNativeGapObservedSingleQuoteCapture(t *testing.T) {
	root := os.Getenv("AETHER_CODEX_GAP_CAPTURE")
	if root == "" {
		t.Skip("explicit immutable capture required")
	}
	rawReceipt, err := os.ReadFile(filepath.Join(root, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt codexNativeLiveReceipt
	if json.Unmarshal(rawReceipt, &receipt) != nil {
		t.Fatal("invalid original receipt")
	}
	paths, err := filepath.Glob(filepath.Join(root, "home/.codex/sessions/*/*/*/*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{"call_ymn3RdT67KSPzOXx22HtWAWe": false, "call_WkaJC0j7lB5KyMQ74kVWOoPO": false, "call_xTUsSlBu4mtuSnP3Q7Cf8FEf": false}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var meta nativeHostEvent
		if json.Unmarshal(bytes.SplitN(raw, []byte{'\n'}, 2)[0], &meta) == nil && meta.Type == "session_meta" && meta.Payload.ID == receipt.SessionID {
			derived := receipt
			derived.ParentSubstitution = false
			derived.ParentUnclassified = nil
			nativeInspectParentEvents(&derived, raw)
			if derived.ParentSubstitution || len(derived.ParentUnclassified) != 0 {
				t.Errorf("parent remains unclassified: %v", derived.ParentUnclassified)
			}
		}
		for _, line := range bytes.Split(raw, []byte{'\n'}) {
			var w nativeCodeModeWire
			if json.Unmarshal(line, &w) != nil || w.Type != "response_item" || w.Payload.Type != "custom_tool_call" {
				continue
			}
			if _, ok := wanted[w.Payload.CallID]; !ok {
				continue
			}
			commands, ok := nativeCodeModeCommands(w.Payload.Input, receipt.FixtureRoot)
			if !ok || !nativeCorroboratedBatch(raw, receipt.SessionID, w.Payload.CallID, commands) {
				t.Errorf("actual call %s rejected", w.Payload.CallID)
				continue
			}
			wanted[w.Payload.CallID] = true
			t.Logf("actual raw=%s line=%s call=%s", lifecycleDigest(raw), lifecycleDigest(line), w.Payload.CallID)
		}
	}
	for call, seen := range wanted {
		if !seen {
			t.Errorf("missing accepted actual call %s", call)
		}
	}
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
	for _, wrapper := range []string{"direct", "output-projection"} {
		for _, mode := range []string{"valid", "file_uri", "actual_command_failure", "missing", "wrong_thread", "wrong_turn", "wrong_cwd", "wrong_argv", "reused_event", "duplicate_call", "missing_output", "wrong_output_call", "wrong_output_turn", "failed_script"} {
			t.Run(wrapper+"/"+mode, func(t *testing.T) {
				var raw []byte
				add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
				add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "parent", "cwd": "/fixture"}})
				input := `text(await tools.exec_command({cmd:"aether status",workdir:"/fixture"}));`
				if wrapper == "output-projection" {
					input = `const r = await tools.exec_command({cmd:"aether status",workdir:"/fixture"}); text(r.output);`
				}
				call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "call", "input": input, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
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
				status, exit := "completed", 0
				if mode == "actual_command_failure" {
					status, exit = "failed", 1
				}
				event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": "shell", "status": status, "cwd": cwd, "command": []string{"/bin/sh", "-c", command}, "exit_code": exit}}}
				if mode != "missing" {
					add(event)
				}
				outputCall, outputTurn, outputText := "call", "turn", "Script completed\nactual output text"
				switch mode {
				case "wrong_output_call":
					outputCall = "other-call"
				case "wrong_output_turn":
					outputTurn = "other-turn"
				case "failed_script":
					outputText = "Script failed\n"
				}
				if mode != "missing_output" {
					add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": outputCall, "output": []any{map[string]any{"type": "input_text", "text": outputText}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": outputTurn}}})
				}
				if mode == "reused_event" {
					add(event)
				}
				r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "parent", FixtureRoot: "/fixture"}
				nativeInspectParentEvents(&r, raw)
				if r.ParentSubstitution != (mode != "valid" && mode != "file_uri" && mode != "actual_command_failure") || r.ChecksPassed {
					t.Fatalf("mode %s classification: %+v", mode, r.ParentUnclassified)
				}
			})
		}
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
