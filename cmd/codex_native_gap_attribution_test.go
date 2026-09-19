package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	for _, wrapper := range []string{"direct", "output-projection", "json-result", "output-exit", "all-output", "all-json"} {
		for _, mode := range []string{"valid", "file_uri", "actual_command_failure", "missing", "wrong_thread", "wrong_turn", "wrong_cwd", "wrong_argv", "reused_event", "duplicate_call", "missing_output", "wrong_output_call", "wrong_output_turn", "failed_script"} {
			t.Run(wrapper+"/"+mode, func(t *testing.T) {
				var raw []byte
				add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
				add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "parent", "cwd": "/fixture"}})
				input := `text(await tools.exec_command({cmd:"aether status",workdir:"/fixture"}));`
				if wrapper == "output-projection" {
					input = `const r = await tools.exec_command({cmd:"aether status",workdir:"/fixture"}); text(r.output);`
				}
				if wrapper == "json-result" {
					input = `const r = await tools.exec_command({cmd:"aether status",workdir:"/fixture"}); text(JSON.stringify(r));`
				}
				if wrapper == "output-exit" {
					input = "const r = await tools.exec_command({cmd:\"aether status\",workdir:\"/fixture\"}); text(r.output); text(`\\nEXIT_CODE=${r.exit_code}`);"
				}
				if wrapper == "all-output" || wrapper == "all-json" {
					input = `const results = await Promise.all([tools.exec_command({cmd:"aether status",workdir:"/fixture"})]); for (const r of results) text(r.output);`
					if wrapper == "all-json" {
						input = strings.Replace(input, "text(r.output)", "text(JSON.stringify(r))", 1)
					}
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
	t.Run("full-result-without-nested-event", nativeGapUncachedCodeModeResultWithoutNestedEvent)
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

func nativeGapUncachedCodeModeResultWithoutNestedEvent(t *testing.T) {
	const testOutput = "{\"Action\":\"run\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\"}\n"
	for _, mode := range []string{"valid", "projected", "projected_eof", "plain_check", "wrong_child", "wrong_turn", "duplicate_call", "duplicate_output", "missing_exit", "failure", "running", "interrupted", "not_test_json", "later_edit"} {
		t.Run(mode, func(t *testing.T) {
			r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", Caste: "builder"}
			var raw []byte
			add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
			add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "cwd": "/fixture", "agent_role": "aether-builder"}})
			thread := "child"
			if mode == "wrong_child" {
				thread = "foreign"
			}
			add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": thread, "turn_id": "turn"}})
			command, print := "go test ./... -json -count=1", "text(JSON.stringify(r));"
			if mode == "plain_check" {
				command = "go test ./..."
			}
			if mode == "projected" || mode == "projected_eof" {
				print = "text(r.output);"
				if mode == "projected_eof" {
					print = "text(r.output)\n"
				}
			}
			call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "check", "input": "const r = await tools.exec_command({cmd:" + strconv.Quote(command) + ",workdir:\"/fixture\"}); " + print, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
			add(call)
			if mode == "duplicate_call" {
				add(call)
			}
			if mode == "later_edit" {
				raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{}})...)
			}
			result := map[string]any{"exit_code": 0, "output": testOutput}
			switch mode {
			case "missing_exit":
				delete(result, "exit_code")
			case "failure":
				result["exit_code"] = 1
			case "running":
				result["session_id"] = 123
			case "not_test_json":
				result["output"] = "ok example.invalid/nativefixture (cached)"
			}
			encoded, _ := json.Marshal(result)
			turn, header := "turn", "Script completed\n"
			if mode == "wrong_turn" {
				turn = "foreign"
			}
			if mode == "interrupted" {
				header = "Script running with cell ID unfinished\n"
			}
			// Deliberately supply a complete-looking result even for .output:
			// a projection cannot supply exit proof without its matched event.
			output := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "check", "output": []any{map[string]any{"type": "input_text", "text": header}, map[string]any{"type": "input_text", "text": string(encoded)}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": turn}}}
			add(output)
			if mode == "duplicate_output" {
				add(output)
			}
			nativeInspectChildEvents(&r, raw)
			if r.ChecksPassed != (mode == "valid") {
				t.Fatalf("%s checks=%v", mode, r.ChecksPassed)
			}
		})
	}
}

func TestCodexNativeGapUncachedCheckSurvivesCoverageRecheck(t *testing.T) {
	for _, check := range []string{"go test ./...", "go test ./... -cover", "go test ./... -cover -count=1", "go test -cover ./..."} {
		for _, projected := range []bool{false, true} {
			for _, mode := range []string{"valid", "no_prior", "prior_failure", "prior_edit", "failed_since_prior", "wrong_thread", "wrong_turn", "duplicate_event", "duplicate_call", "duplicate_output", "failure", "failed_zero", "missing_exit", "missing_event", "missing_output", "later_edit", "wrong_argv", "wrong_cwd", "copied_output", "interrupted_output", "interleaved_call", "forged_projection"} {
				t.Run(check+"/projected="+strconv.FormatBool(projected)+"/"+mode, func(t *testing.T) {
					r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", Caste: "builder"}
					var raw []byte
					add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
					add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "cwd": "/fixture", "agent_role": "aether-builder"}})
					if mode != "no_prior" {
						exit, status, action := 0, "completed", "pass"
						if mode == "prior_failure" {
							exit, status, action = 1, "failed", "fail"
						}
						raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "CommandExecution", "status": status, "id": "uncached", "command": []string{"/bin/sh", "-c", "go test ./... -json -count=1"}, "cwd": "/fixture", "exit_code": exit, "aggregated_output": "{\"Action\":\"run\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"" + action + "\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"" + action + "\",\"Package\":\"example.invalid/nativefixture\"}\n"})...)
					}
					if mode == "prior_edit" {
						raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{}})...)
					}
					if mode == "failed_since_prior" {
						raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "CommandExecution", "status": "failed", "id": "later-failure", "command": []string{"/bin/sh", "-c", "go test ./..."}, "cwd": "/fixture", "exit_code": 1, "aggregated_output": "FAIL"})...)
					}
					add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": "child", "turn_id": "turn"}})
					print := "text(JSON.stringify(r));"
					if projected {
						print = "text(r.output);"
					}
					if mode == "forged_projection" {
						print = `text("ok example.invalid/nativefixture (cached)");`
					}
					input := "const r = await tools.exec_command({cmd:" + strconv.Quote(check) + ",workdir:\"/fixture\"}); " + print
					call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "plain", "input": input, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
					add(call)
					if mode == "duplicate_call" {
						add(call)
					}
					if mode == "interleaved_call" {
						add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "other", "input": `text(await tools.exec_command({cmd:"git status --short",workdir:"/fixture"}));`, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
					}
					thread, turn, exit, status := "child", "turn", 0, "completed"
					switch mode {
					case "wrong_thread":
						thread = "foreign"
					case "wrong_turn":
						turn = "foreign"
					case "failure":
						exit, status = 1, "failed"
					case "failed_zero":
						status = "failed"
					}
					command, cwd := check, "/fixture"
					if mode == "wrong_argv" {
						command = "go test -cover ./..."
						if command == check {
							command = "go test ./... -cover"
						}
					}
					if mode == "wrong_cwd" {
						cwd = "/other"
					}
					item := map[string]any{"type": "CommandExecution", "id": "plain-event", "status": status, "command": []string{"/bin/zsh", "-lc", command}, "cwd": cwd, "exit_code": exit, "aggregated_output": "ok example.invalid/nativefixture (cached)"}
					if mode == "missing_exit" {
						delete(item, "exit_code")
					}
					event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": item}}
					if mode != "missing_event" {
						add(event)
					}
					if mode == "duplicate_event" {
						add(event)
					}
					if mode == "later_edit" {
						raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{}})...)
					}
					if mode != "missing_output" {
						output := "ok example.invalid/nativefixture (cached)"
						if mode == "copied_output" {
							output = "copied success from another command"
						}
						if !projected {
							result, _ := json.Marshal(map[string]any{"exit_code": exit, "output": output})
							output = string(result)
						}
						header := "Script completed\n"
						if mode == "interrupted_output" {
							header = "Script running with cell ID unfinished\n"
						}
						result := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "plain", "output": []any{map[string]any{"type": "input_text", "text": header}, map[string]any{"type": "input_text", "text": output}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
						add(result)
						if mode == "duplicate_output" {
							add(result)
						}
					}
					nativeInspectChildEvents(&r, raw)
					if r.ChecksPassed != (mode == "valid") {
						t.Fatalf("%s checks=%v", mode, r.ChecksPassed)
					}
				})
			}
		}
	}
}

func TestCodexNativeObservedOperationWrappers(t *testing.T) {
	const prefix = `const r = await tools.exec_command({cmd:"go test ./... -json -count=1",workdir:"/fixture"}); `
	for _, tc := range []struct {
		name, suffix string
		want         bool
	}{
		{"full-json", `text(JSON.stringify(r));`, true},
		{"full-json-eof", "text(JSON.stringify(r))\n", true},
		{"full-result-eof", "text(r)\n", true},
		{"output-eof", "text(r.output)\n", true},
		{"output-exit", "text(r.output); text(`\\nEXIT_CODE=${r.exit_code}`);", true},
		{"output-exit-eof", "text(r.output); text(`\\nEXIT_CODE=${r.exit_code}`)\n", true},
		{"other-result", `text(JSON.stringify(other));`, false},
		{"json-replacer", `text(JSON.stringify(r, mutate));`, false},
		{"json-property", `text(JSON.stringify(r.output));`, false},
		{"rewritten-result", `r.exit_code=0; text(JSON.stringify(r));`, false},
		{"extra-side-effect", `text(JSON.stringify(r)); mutate();`, false},
		{"eof-side-effect", "text(JSON.stringify(r))\nmutate();", false},
		{"eof-output-side-effect", "text(r.output)\nmutate();", false},
		{"eof-full-result-side-effect", "text(r)\nmutate();", false},
		{"eof-output-transform", "text(r.output.trim())\n", false},
		{"eof-other-result", "text(JSON.stringify(other))\n", false},
		{"missing-output-exit-separator", "text(r.output)\ntext(`\\nEXIT_CODE=${r.exit_code}`)", false},
		{"wrong-exit-variable", "text(r.output); text(`\\nEXIT_CODE=${other.exit_code}`);", false},
		{"computed-exit", "text(r.output); text(`\\nEXIT_CODE=${r.exit_code || 0}`);", false},
		{"fake-exit", "text(r.output); text(`\\nEXIT_CODE=0`);", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := nativeCodeModeCommands(prefix+tc.suffix, "/fixture")
			if ok != tc.want || (ok && (len(got) != 1 || got[0].Command != "go test ./... -json -count=1" || got[0].Cwd != "/fixture")) {
				t.Fatalf("accepted=%v commands=%+v", ok, got)
			}
		})
	}
	for _, bad := range []string{
		strings.Replace(prefix, `"go test ./... -json -count=1"`, `command`, 1) + `text(JSON.stringify(r));`,
		strings.Replace(prefix, `cmd:`, `cmd:"touch clamp.go",cmd:`, 1) + `text(JSON.stringify(r));`,
		strings.Replace(prefix, `await `, ``, 1) + `text(JSON.stringify(r));`,
		`const JSON={stringify:mutate}; ` + prefix + `text(JSON.stringify(r));`,
		strings.TrimSuffix(strings.TrimSpace(prefix), ";") + "\ntext(r.output)",
		strings.TrimSuffix(strings.TrimSpace(prefix), ";") + "\ntext(JSON.stringify(r))",
		strings.TrimSuffix(strings.TrimSpace(prefix), ";") + "text(r)",
	} {
		if _, ok := nativeCodeModeCommands(bad, "/fixture"); ok {
			t.Fatalf("dynamic/side-effect wrapper accepted: %s", bad)
		}
	}
	t.Run("literal-eof-sequence-boundary", func(t *testing.T) {
		const direct = `text(await tools.exec_command({cmd:"git status --short",workdir:"/fixture"}))`
		for _, tc := range []struct {
			input string
			want  int
		}{
			{direct + "\n", 1},
			{direct + ";\n" + direct + "\n", 2},
			{direct + "\n" + direct, 0},
			{direct + direct, 0},
			{direct + "\nmutate();", 0},
		} {
			commands, ok := nativeCodeModeCommands(tc.input, "/fixture")
			if ok != (tc.want > 0) || (ok && len(commands) != tc.want) {
				t.Fatalf("EOF sequence accepted=%v commands=%v input=%s", ok, commands, tc.input)
			}
		}
		if !nativeCodeModePlainOutput(prefix+"text(r.output)\n", "/fixture") {
			t.Fatal("EOF projection lost its independent event requirement")
		}
	})
}

func TestCodexNativeObservedReadOnlyBatch(t *testing.T) {
	t.Run("named-results", nativeObservedNamedReadOnlyBatch)
	batch := `const results = await Promise.all([tools.exec_command({cmd:"sed -n '1,200p' clamp_test.go",workdir:"/fixture"}),tools.exec_command({cmd:"git status --short",workdir:"/fixture"})]); for (const r of results) text(r.output);`
	for _, tc := range []struct {
		name, input string
		want        bool
	}{
		{"output", batch, true},
		{"full-json", strings.Replace(batch, "text(r.output)", "text(JSON.stringify(r))", 1), true},
		{"wrong-array", strings.Replace(batch, "of results", "of other", 1), false},
		{"wrong-result", strings.Replace(batch, "text(r.output)", "text(other.output)", 1), false},
		{"mutated-result", strings.Replace(batch, "text(r.output)", "text(r.output.trim())", 1), false},
		{"trailing", batch + `mutate();`, false},
		{"unawaited", strings.Replace(batch, "await Promise", "Promise", 1), false},
		{"write", strings.Replace(batch, "git status --short", "gofmt -w clamp.go", 1), false},
		{"compound", strings.Replace(batch, "git status --short", "git status --short; touch clamp.go", 1), false},
		{"computed-command", strings.Replace(batch, `"git status --short"`, `command`, 1), false},
		{"other-tool", strings.Replace(batch, "tools.exec_command", "tools.apply_patch", 1), false},
		{"spread-array", strings.Replace(batch, "[tools.exec_command", "[...tools.exec_command", 1), false},
		{"loop-body-side-effect", strings.Replace(batch, "text(r.output);", "{ mutate(); text(r.output); }", 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commands, ok := nativeCodeModeCommands(tc.input, "/fixture")
			if ok != tc.want || (ok && len(commands) != 2) {
				t.Fatalf("accepted=%v commands=%+v", ok, commands)
			}
		})
	}
	calls := make([]string, 65)
	for i := range calls {
		calls[i] = `tools.exec_command({cmd:"git status --short",workdir:"/fixture"})`
	}
	if _, ok := nativeCodeModeCommands(`const results=await Promise.all([`+strings.Join(calls, ",")+`]); for (const r of results) text(r.output);`, "/fixture"); ok {
		t.Fatal("unbounded batch accepted")
	}
}

func nativeObservedNamedReadOnlyBatch(t *testing.T) {
	inspection := `const [status, source, tests] = await Promise.all([tools.exec_command({cmd:"git status --short",workdir:"/fixture"}),tools.exec_command({cmd:"sed -n '1,200p' clamp.go",workdir:"/fixture"}),tools.exec_command({cmd:"sed -n '1,200p' clamp_test.go",workdir:"/fixture"})]); text(JSON.stringify({status,source,tests}));`
	freshness := `const [diff, status, freshness] = await Promise.all([tools.exec_command({cmd:"git diff -- clamp.go",workdir:"/fixture"}),tools.exec_command({cmd:"git status --short",workdir:"/fixture"}),tools.exec_command({cmd:"date -u +%Y-%m-%dT%H:%M:%SZ",workdir:"/fixture"})]); text(JSON.stringify({diff,status,freshness}));`
	for _, tc := range []struct {
		name, input string
		want        bool
	}{
		{"inspection", inspection, true},
		{"freshness", freshness, true},
		{"single", `const [status] = await Promise.all([tools.exec_command({cmd:"git status --short",workdir:"/fixture"})]); text(JSON.stringify({status}));`, true},
		{"omitted-field", strings.Replace(inspection, "{status,source,tests}", "{status,source}", 1), false},
		{"duplicate-field", strings.Replace(inspection, "{status,source,tests}", "{status,source,source}", 1), false},
		{"reordered-fields", strings.Replace(inspection, "{status,source,tests}", "{source,status,tests}", 1), false},
		{"duplicate-binding", strings.ReplaceAll(inspection, "tests", "source"), false},
		{"too-few-bindings", strings.Replace(strings.Replace(inspection, "[status, source, tests]", "[status, source]", 1), "{status,source,tests}", "{status,source}", 1), false},
		{"too-many-bindings", strings.Replace(strings.Replace(inspection, "[status, source, tests]", "[status, source, tests, extra]", 1), "{status,source,tests}", "{status,source,tests,extra}", 1), false},
		{"default", strings.Replace(inspection, "[status,", "[status = mutate(),", 1), false},
		{"rest", strings.Replace(inspection, "source, tests]", "source, ...tests]", 1), false},
		{"hole", strings.Replace(inspection, "status, source, tests]", "status, , tests]", 1), false},
		{"alias", strings.Replace(inspection, "{status,source,tests}", "{status:source,source,tests}", 1), false},
		{"computed-key", strings.Replace(inspection, "{status,source,tests}", "{[status]:status,source,tests}", 1), false},
		{"spread-key", strings.Replace(inspection, "{status,source,tests}", "{...status,source,tests}", 1), false},
		{"transformed-value", strings.Replace(inspection, "{status,source,tests}", "{status:status.output,source,tests}", 1), false},
		{"json-replacer", strings.Replace(inspection, "{status,source,tests}", "{status,source,tests},mutate", 1), false},
		{"extra-statement", inspection + `mutate();`, false},
		{"mutated-binding", strings.Replace(inspection, "text(JSON", "status.exit_code=0; text(JSON", 1), false},
		{"unawaited", strings.Replace(inspection, "await Promise", "Promise", 1), false},
		{"settled", strings.Replace(inspection, "Promise.all(", "Promise.allSettled(", 1), false},
		{"spread-call", strings.Replace(inspection, "[tools.exec_command", "[...tools.exec_command", 1), false},
		{"dynamic-command", strings.Replace(inspection, `"git status --short"`, "command", 1), false},
		{"write", strings.Replace(inspection, "git status --short", "gofmt -w clamp.go", 1), false},
		{"test", strings.Replace(inspection, "git status --short", "go test ./... -json -count=1", 1), false},
		{"coverage", strings.Replace(inspection, "git status --short", "go test -cover ./...", 1), false},
		{"shell-effect", strings.Replace(inspection, "git status --short", "git status --short; touch clamp.go", 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := nativeCodeModeCommands(tc.input, "/fixture")
			if ok != tc.want {
				t.Fatalf("named batch accepted=%v", ok)
			}
		})
	}
	for _, name := range []string{"text", "JSON", "tools", "Promise", "await", "null", "const"} {
		t.Run("shadow-"+name, func(t *testing.T) {
			input := strings.ReplaceAll(inspection, "source", name)
			if _, ok := nativeCodeModeCommands(input, "/fixture"); ok {
				t.Fatal("shadowed execution/printing name accepted")
			}
		})
	}
	for _, size := range []int{64, 65} {
		names, calls := make([]string, size), make([]string, size)
		for index := range names {
			names[index] = "result" + strconv.Itoa(index)
			calls[index] = `tools.exec_command({cmd:"git status --short",workdir:"/fixture"})`
		}
		input := "const [" + strings.Join(names, ",") + "] = await Promise.all([" + strings.Join(calls, ",") + "]); text(JSON.stringify({" + strings.Join(names, ",") + "}));"
		if commands, ok := nativeCodeModeCommands(input, "/fixture"); ok != (size == 64) || (ok && len(commands) != size) {
			t.Fatalf("named bound size=%d accepted=%v count=%d", size, ok, len(commands))
		}
	}
	for _, input := range []string{inspection, freshness} {
		for _, mode := range []string{"valid", "wrong_thread", "wrong_turn", "wrong_cwd", "wrong_command", "wrong_output", "wrong_output_turn", "duplicate_call", "duplicate_event", "duplicate_output", "missing_event", "missing_output", "interleaved_call", "incomplete_output", "invocation_cwd"} {
			t.Run(input+"/"+mode, func(t *testing.T) {
				commands, ok := nativeCodeModeCommands(input, "/fixture")
				if !ok || len(commands) != 3 {
					t.Fatal("observed named batch rejected")
				}
				r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", Caste: "builder"}
				var raw []byte
				add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
				add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "cwd": "/fixture", "agent_role": "aether-builder"}})
				add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": "child", "turn_id": "turn"}})
				selected := input
				if mode == "invocation_cwd" {
					selected = strings.Replace(input, "/fixture", "/other", 1)
				}
				call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "batch", "input": selected, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
				add(call)
				if mode == "duplicate_call" {
					add(call)
				}
				if mode == "interleaved_call" {
					add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": "send_message", "call_id": "other"}})
				}
				for index, command := range commands {
					if mode == "missing_event" && index == 1 {
						continue
					}
					thread, turn, cwd, cmd := "child", "turn", command.Cwd, command.Command
					if index == 1 {
						switch mode {
						case "wrong_thread":
							thread = "foreign"
						case "wrong_turn":
							turn = "foreign"
						case "wrong_cwd":
							cwd = "/other"
						case "wrong_command":
							cmd = "cat go.mod"
						}
					}
					event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": "event" + strconv.Itoa(index), "status": "completed", "command": []string{"/bin/zsh", "-lc", cmd}, "cwd": cwd, "exit_code": 0, "aggregated_output": "actual inspection output"}}}
					add(event)
					if mode == "duplicate_event" && index == 1 {
						add(event)
					}
				}
				id, turn, header := "batch", "turn", "Script completed\n"
				switch mode {
				case "wrong_output":
					id = "other"
				case "wrong_output_turn":
					turn = "other"
				case "incomplete_output":
					header = "Script running with cell ID unfinished\n"
				}
				fields := []string{"status", "source", "tests"}
				if input == freshness {
					fields = []string{"diff", "status", "freshness"}
				}
				values := map[string]any{}
				for _, field := range fields {
					values[field] = map[string]any{"exit_code": 0, "output": "actual inspection output"}
				}
				printed, _ := json.Marshal(values)
				result := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": id, "output": []any{map[string]any{"type": "input_text", "text": header}, map[string]any{"type": "input_text", "text": string(printed)}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": turn}}}
				if mode != "missing_output" {
					add(result)
				}
				if mode == "duplicate_output" {
					add(result)
				}
				nativeInspectChildEvents(&r, raw)
				if (len(r.ChildUnclassified) == 0) != (mode == "valid") || r.ChecksPassed || r.ChildEditObserved {
					t.Fatalf("mode=%s unclassified=%v checks=%v edits=%v", mode, r.ChildUnclassified, r.ChecksPassed, r.ChildEditObserved)
				}
			})
		}
	}
}

func TestCodexNativeObservedFixtureInspections(t *testing.T) {
	t.Run("single-invocation-semicolon", nativeObservedSemicolonInspection)
	r := codexNativeLiveReceipt{FixtureRoot: "/fixture"}
	for _, tc := range []struct {
		command      string
		child, batch bool
	}{
		{`sed -n '1,200p' clamp.go`, true, true},
		{`sed -n '1,200p' /fixture/clamp_test.go`, true, true},
		{`gofmt -d clamp.go`, true, true},
		{`git diff --check -- clamp.go`, true, true},
		{`git diff --check`, true, true},
		{`rg --files`, true, true},
		{`sed -n '1,240p' clamp.go; sed -n '1,280p' clamp_test.go; git status --short`, true, false},
		{`rg --files; git status --short`, true, false},
		// Generic parent-readonly syntax admits an absolute listing root;
		// the child classifier separately restricts operations to its fixture.
		{`rg --files /other`, false, true},
		{`rg --files --hidden`, false, false},
		{`rg --files --follow`, false, false},
		{`rg --files; go test ./...`, false, false},
		{`rg --files; gofmt -w clamp.go`, false, false},
		{`rg --files; /fixture/aether codex-native-worker context --request /fixture/request.json`, false, false},
		{`rg --files; cat ../clamp.go`, false, false},
		{`rg --files; sed -n '1,20p' /other/clamp.go`, false, false},
		{`rg --files; git status --short > clamp.go`, false, false},
		{`rg --files; $(touch clamp.go)`, false, false},
		{`rg --files; git status --short && git diff --check`, false, false},
		{`rg --files;; git status --short`, false, false},
		{`rg --files;`, false, false},
		{`;rg --files`, false, false},
		{strings.Repeat(`git status --short;`, 8), false, false},
		{strings.Repeat(`git status --short;`, 8) + `git status --short`, false, false},
		{`rg --files -g 'AGENTS.md' -g 'clamp.go' -g 'clamp_test.go' -g 'go.mod'`, true, true},
		{`rg --files -g 'AGENTS.md' -g 'clamp.go' -g 'clamp_test.go' -g 'go.mod' && git status --short`, true, false},
		{`sed -n '1,220p' clamp.go && sed -n '1,260p' clamp_test.go && sed -n '1,120p' go.mod`, true, false},
		{`test -r clamp.go`, true, true},
		{`date -u +%Y-%m-%dT%H:%M:%SZ`, true, true},
		{`go test ./... -cover -count=1`, true, false},
		{`go test ./... -cover`, true, false},
		{`go test -cover ./...`, true, false},
		{`rg --files -g '*.go'`, false, false},
		{`rg --files -g 'clamp.go' -g 'clamp.go'`, false, false},
		{`rg --files -g '../clamp.go'`, false, false},
		{`rg --files -g 'clamp.go' --pre mutate`, false, false},
		{`rg --files -g 'clamp.go' /other`, false, false},
		{`sed -n '1,200p' clamp.go && go test ./...`, false, false},
		{`sed -n '1,200p' clamp.go && gofmt -w clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go && sed -n '1,200p' /other/clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go && git status --short; touch clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go || git status --short`, false, false},
		{`sed -n '1,200p' clamp.go && git status --short > clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go && $(touch clamp.go)`, false, false},
		{`sed -n '1,200p' clamp.go && 'git status' --short`, false, false},
		{`sed -n '1,200p' clamp.go && 'git diff' --check`, false, false},
		{`sed -n '1,200p' clamp.go && `, false, false},
		{strings.Repeat(`git status --short && `, 8) + `git status --short`, false, false},
		{`sed -n '1,200p' /other/clamp.go`, false, true},
		{`sed -n '1,200e' clamp.go`, false, false},
		{`sed -i '' clamp.go`, false, false},
		{`gofmt -w clamp.go`, false, false},
		{`gofmt -d /other/clamp.go`, false, false},
		{`git diff --check -- /other/clamp.go`, false, false},
		{`git diff --output=/tmp/side-effect -- clamp.go`, false, false},
		{`test -r /other/clamp.go`, false, false},
		{`date -u -s tomorrow`, false, false},
		{`go test ./... -coverprofile=/tmp/profile`, false, false},
		{`go test ./... -cover -count=1; touch clamp.go`, false, false},
		{`go test -cover ./... -run TestClamp`, false, false},
		{`go test -cover ./... -coverprofile=/tmp/profile`, false, false},
		{`go test -cover ./...; touch clamp.go`, false, false},
		{`go test -cover ./... > /tmp/result`, false, false},
		{`go test -cover ./other`, false, false},
	} {
		t.Run(tc.command, func(t *testing.T) {
			if got := nativeChildCommandAllowed(r, []string{"/bin/sh", "-c", tc.command}); got != tc.child {
				t.Fatalf("child=%v want%v", got, tc.child)
			}
			if got := nativeReadOnlyBatchCommand(tc.command); got != tc.batch {
				t.Fatalf("batch=%v want%v", got, tc.batch)
			}
		})
	}
	for _, command := range []string{`gofmt -w clamp.go`, `go test ./... -cover -count=1`, `go test -cover ./...`} {
		if nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", command}, "/fixture") {
			t.Fatalf("child operation admitted for parent: %s", command)
		}
	}
}

func nativeObservedSemicolonInspection(t *testing.T) {
	const chain = `sed -n '1,240p' clamp.go; sed -n '1,280p' clamp_test.go; git status --short`
	for _, command := range []string{chain, `rg --files`, `rg --files /other`} {
		for _, mode := range []string{"valid", "invocation-cwd", "wrong-cwd", "wrong-thread", "wrong-turn", "missing-event", "duplicate-event", "split-events", "changed-command", "missing-output", "duplicate-output", "interleaved-call"} {
			// Bare listing uses the pre-existing direct-command classifier;
			// only its actual invocation/event workspace needs a new control.
			if command == "rg --files" && mode != "valid" && mode != "invocation-cwd" && mode != "wrong-cwd" {
				continue
			}
			if command == "rg --files /other" && mode != "valid" {
				continue
			}
			t.Run(command+"/"+mode, func(t *testing.T) {
				r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", Caste: "builder"}
				var raw []byte
				add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
				add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "cwd": "/fixture", "agent_role": "aether-builder"}})
				add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": "child", "turn_id": "turn"}})
				cwd := "/fixture"
				if mode == "invocation-cwd" {
					cwd = "/other"
				}
				input := "const r = await tools.exec_command({cmd:" + strconv.Quote(command) + ",workdir:" + strconv.Quote(cwd) + "}); text(r)\n"
				commands, ok := nativeCodeModeCommands(input, "/fixture")
				if !ok || len(commands) != 1 || commands[0].Command != command {
					t.Fatal("one shell invocation was split or rewritten")
				}
				add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "inspection", "input": input, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
				if mode == "interleaved-call" {
					add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": "send_message", "call_id": "other"}})
				}
				thread, turn, observed := "child", "turn", command
				switch mode {
				case "wrong-cwd":
					cwd = "/other"
				case "wrong-thread":
					thread = "other"
				case "wrong-turn":
					turn = "other"
				case "changed-command":
					observed = "git status --short"
				}
				event := func(id, command string) map[string]any {
					return map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": id, "status": "completed", "command": []string{"/bin/zsh", "-lc", command}, "cwd": cwd, "exit_code": 0, "aggregated_output": "actual inspection output"}}}
				}
				if mode == "split-events" {
					for index, part := range strings.Split(command, ";") {
						add(event("split"+strconv.Itoa(index), strings.TrimSpace(part)))
					}
				} else if mode != "missing-event" {
					add(event("event", observed))
					if mode == "duplicate-event" {
						add(event("event", observed))
					}
				}
				result := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "inspection", "output": []any{map[string]any{"type": "input_text", "text": "Script completed\n"}, map[string]any{"type": "input_text", "text": `{"exit_code":0,"output":"actual inspection output"}`}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
				if mode != "missing-output" {
					add(result)
				}
				if mode == "duplicate-output" {
					add(result)
				}
				nativeInspectChildEvents(&r, raw)
				wantClassified := mode == "valid" && command != "rg --files /other"
				if (len(r.ChildUnclassified) == 0) != wantClassified || r.ChecksPassed || r.ChildEditObserved {
					t.Fatalf("unclassified=%v checks=%v edits=%v", r.ChildUnclassified, r.ChecksPassed, r.ChildEditObserved)
				}
			})
		}
	}
}

func TestCodexNativeObservedLiteralPatch(t *testing.T) {
	patch := "*** Begin Patch\n*** Update File: /fixture/clamp.go\n@@\n-old\n+new\n*** End Patch"
	input := "const patch = " + strconv.Quote(patch) + "; text(await tools.apply_patch(patch));"
	for _, mode := range []string{"valid", "result-alias", "result-wrong-patch", "result-wrong-output", "result-reassigned", "result-unawaited", "result-shadowed", "result-trailing", "computed", "wrong-variable", "trailing", "missing-change", "wrong-child", "wrong-turn", "duplicate-change", "duplicate-call", "wrong-output", "protected-file"} {
		t.Run(mode, func(t *testing.T) {
			selected := input
			if strings.HasPrefix(mode, "result-") {
				selected = "const patch = " + strconv.Quote(patch) + "; const result = await tools.apply_patch(patch); text(result);"
			}
			switch mode {
			case "result-wrong-patch":
				selected = strings.Replace(selected, "apply_patch(patch)", "apply_patch(other)", 1)
			case "result-wrong-output":
				selected = strings.Replace(selected, "text(result)", "text(other)", 1)
			case "result-reassigned":
				selected = strings.Replace(selected, "text(result)", `result = {}; text(result)`, 1)
			case "result-unawaited":
				selected = strings.Replace(selected, "await ", "", 1)
			case "result-shadowed":
				selected = strings.ReplaceAll(selected, "result", "patch")
			case "result-trailing":
				selected += `mutate();`
			case "computed":
				selected = strings.Replace(selected, "; text", `+suffix; text`, 1)
			case "wrong-variable":
				selected = strings.Replace(selected, "apply_patch(patch)", "apply_patch(other)", 1)
			case "trailing":
				selected += `mutate();`
			}
			r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", BaselineSource: "old\n", FinalSource: "new\n"}
			var raw []byte
			add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
			add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "agent_role": "aether-builder", "cwd": "/fixture"}})
			call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "call", "input": selected, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
			add(call)
			if mode == "duplicate-call" {
				add(call)
			}
			thread, turn, path := "child", "turn", "/fixture/clamp.go"
			switch mode {
			case "wrong-child":
				thread = "sibling"
			case "wrong-turn":
				turn = "other"
			case "protected-file":
				path = "/fixture/clamp_test.go"
			}
			change := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "FileChange", "id": "edit", "status": "completed", "changes": map[string]any{path: map[string]any{"type": "update", "unified_diff": "@@ -1,1 +1,1 @@\n-old\n+new\n"}}}}}
			if mode != "missing-change" {
				add(change)
			}
			if mode == "duplicate-change" {
				add(change)
			}
			id := "call"
			if mode == "wrong-output" {
				id = "other"
			}
			add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": id, "output": []any{map[string]any{"type": "input_text", "text": "Script completed\n"}, map[string]any{"type": "input_text", "text": "{}"}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
			valid := mode == "valid" || mode == "result-alias"
			if got := nativeCorroboratedLiteralPatch(r, raw, "call"); got != valid {
				t.Fatalf("corroborated=%v", got)
			}
			if valid {
				nativeInspectChildEvents(&r, raw)
				if !r.ChildEditObserved || len(r.ChildUnclassified) != 0 {
					t.Fatalf("literal alias child edit unclassified: %v", r.ChildUnclassified)
				}
			}
			parent := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "child", FixtureRoot: "/fixture"}
			nativeInspectParentEvents(&parent, raw)
			if !parent.ParentSubstitution {
				t.Fatal("parent literal alias patch accepted")
			}
		})
	}
}
